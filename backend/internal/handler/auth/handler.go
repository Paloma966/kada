package auth

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/chun/kada-backend/internal/domain"
	"github.com/chun/kada-backend/internal/middleware"
)

// AuthService is the authentication service interface (mockable for tests).
type AuthService interface {
	SendSMSCode(ctx context.Context, phone string) error
	LoginByPhone(ctx context.Context, phone, code string) (*domain.AuthResponse, error)
	LoginByEmail(ctx context.Context, email, password string) (*domain.AuthResponse, error)
	RegisterByEmail(ctx context.Context, email, password, name string) (*domain.AuthResponse, error)
	GetUserByID(ctx context.Context, userID int64) (*domain.UserInfo, error)
	UpdateUser(ctx context.Context, userID int64, name *string, email *string) (*domain.UserInfo, error)
}

type Handler struct {
	svc AuthService
}

func NewHandler(svc AuthService) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) RegisterRoutes(r *gin.RouterGroup, authMW gin.HandlerFunc, strictMW ...gin.HandlerFunc) {
	// Public routes (strict rate limiting can be applied).
	public := r.Group("")
	if len(strictMW) > 0 && strictMW[0] != nil {
		public.Use(strictMW[0])
	}
	public.POST("/auth/send-sms-code", h.SendSMSCode)
	public.POST("/auth/login-by-phone", h.LoginByPhone)
	public.POST("/auth/login-by-email", h.LoginByEmail)
	public.POST("/auth/register-by-email", h.RegisterByEmail)

	// Routes that require authentication.
	auth := r.Group("").Use(authMW)
	auth.GET("/me", h.GetMe)
	auth.PATCH("/me", h.UpdateMe)
}

// SendSMSCode sends an SMS verification code.
func (h *Handler) SendSMSCode(c *gin.Context) {
	var req domain.SendSMSRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "a valid phone number is required"})
		return
	}

	if err := h.svc.SendSMSCode(c.Request.Context(), req.Phone); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "verification code sent",
		"phone":   req.Phone,
	})
}

// LoginByPhone logs in with phone number + verification code.
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

// LoginByEmail logs in with email + password.
func (h *Handler) LoginByEmail(c *gin.Context) {
	var req domain.LoginByEmailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "email and password are required"})
		return
	}

	resp, err := h.svc.LoginByEmail(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"token": resp.Token, "user": resp.User})
}

// RegisterByEmail registers an account by email.
func (h *Handler) RegisterByEmail(c *gin.Context) {
	var req struct {
		Email    string `json:"email" binding:"required,email"`
		Password string `json:"password" binding:"required,min=6"`
		Name     string `json:"name" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "valid registration details are required"})
		return
	}

	resp, err := h.svc.RegisterByEmail(c.Request.Context(), req.Email, req.Password, req.Name)
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"token": resp.Token, "user": resp.User})
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
