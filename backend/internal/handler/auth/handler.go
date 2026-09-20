package auth

import (
	"context"
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/chun/kada-backend/internal/domain"
	"github.com/chun/kada-backend/internal/middleware"
)

// AuthService is the authentication service interface (mockable for tests).
type AuthService interface {
	GenerateCaptcha(ctx context.Context, ip string) (*domain.CaptchaResponse, error)
	SendSMSCode(ctx context.Context, phone, ip, captchaID, captchaAnswer string) error
	LoginByPhone(ctx context.Context, phone, code string) (*domain.AuthResponse, error)
	GetUserByID(ctx context.Context, userID int64) (*domain.UserInfo, error)
	UpdateUser(ctx context.Context, userID int64, name *string, email *string) (*domain.UserInfo, error)
}

type Handler struct {
	svc AuthService
}

func NewHandler(svc AuthService) *Handler {
	return &Handler{svc: svc}
}

// RegisterRoutes wires the auth endpoints.
//
// Signing in is phone-only. The email/password and WeChat routes are gone, not merely hidden in the UI:
// an endpoint nobody uses is still an endpoint that can be attacked, and leaving `/auth/login-by-email`
// reachable would keep a password path alive that no page links to and no test covers.
func (h *Handler) RegisterRoutes(r *gin.RouterGroup, authMW gin.HandlerFunc, strictMW ...gin.HandlerFunc) {
	// Public routes (strict rate limiting can be applied).
	public := r.Group("")
	if len(strictMW) > 0 && strictMW[0] != nil {
		public.Use(strictMW[0])
	}
	public.GET("/auth/captcha", h.Captcha)
	public.POST("/auth/send-sms-code", h.SendSMSCode)
	public.POST("/auth/login-by-phone", h.LoginByPhone)

	// Routes that require authentication.
	auth := r.Group("").Use(authMW)
	auth.GET("/me", h.GetMe)
	auth.PATCH("/me", h.UpdateMe)
}

// Captcha issues the graphical challenge that SendSMSCode requires.
//
// It is a GET because it has no side effect on the caller's session and is safe to retry; the row it
// creates is one-time and expires in five minutes either way.
func (h *Handler) Captcha(c *gin.Context) {
	challenge, err := h.svc.GenerateCaptcha(c.Request.Context(), middleware.RealIP(c))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// The image is a data URI, and the challenge is per-request: a cached copy would hand the same
	// unsolved puzzle to two people behind one proxy.
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, gin.H{"captcha_id": challenge.ID, "image": challenge.Image})
}

// SendSMSCode sends an SMS verification code.
func (h *Handler) SendSMSCode(c *gin.Context) {
	var req domain.SendSMSRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "a valid phone number and the graphical verification code are required"})
		return
	}

	// The IP is taken from the same place the rate limiter takes it: X-Real-IP, which nginx rewrites from
	// $remote_addr. Reading X-Forwarded-For here instead would let a caller choose which quota bucket to
	// be counted in by sending a header of their own.
	if err := h.svc.SendSMSCode(c.Request.Context(), req.Phone, middleware.RealIP(c), req.CaptchaID, req.CaptchaCode); err != nil {
		// A quota refusal is 429 with a wait time; a wrong captcha or a malformed number is 400. Reporting
		// "too many requests" as 500 - which this handler used to do - tells a client to retry a request
		// that will keep failing, and tells the operator to look for a server fault that does not exist.
		var limited *domain.RateLimitError
		if errors.As(err, &limited) {
			if limited.RetryAfterSeconds > 0 {
				c.Header("Retry-After", strconv.Itoa(limited.RetryAfterSeconds))
			}
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":               limited.Message,
				"retry_after_seconds": limited.RetryAfterSeconds,
			})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "verification code sent",
		"phone":   req.Phone,
	})
}

// LoginByPhone logs in with phone number + verification code, creating the account on first use.
func (h *Handler) LoginByPhone(c *gin.Context) {
	var req domain.LoginByPhoneRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "phone number and verification code are required"})
		return
	}

	resp, err := h.svc.LoginByPhone(c.Request.Context(), req.Phone, req.Code)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"token": resp.Token, "user": resp.User})
}

// UpdateMe updates the current user's profile.
func (h *Handler) UpdateMe(c *gin.Context) {
	var req struct {
		Name  *string `json:"name"`
		Email *string `json:"email" binding:"omitempty,email"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "valid update details are required"})
		return
	}

	user, err := h.svc.UpdateUser(c.Request.Context(), middleware.GetUserID(c), req.Name, req.Email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"user": user})
}

// GetMe returns the current user's profile.
func (h *Handler) GetMe(c *gin.Context) {
	userID := middleware.GetUserID(c)

	user, err := h.svc.GetUserByID(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"user": user})
}
