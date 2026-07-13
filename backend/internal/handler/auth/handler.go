package auth

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/chun/kada-backend/internal/domain"
	"github.com/chun/kada-backend/internal/middleware"
	"github.com/chun/kada-backend/internal/service"
)

type Handler struct {
	svc *service.AuthService
}

func NewHandler(svc *service.AuthService) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) RegisterRoutes(r *gin.RouterGroup, authMW gin.HandlerFunc) {
	r.POST("/auth/send-sms-code", h.SendSMSCode)
	r.POST("/auth/login-by-phone", h.LoginByPhone)
	r.POST("/auth/login-by-email", h.LoginByEmail)
	r.POST("/auth/register-by-email", h.RegisterByEmail)

	// 需要认证的路由
	auth := r.Group("").Use(authMW)
	auth.GET("/me", h.GetMe)
}

// SendSMSCode 发送短信验证码
func (h *Handler) SendSMSCode(c *gin.Context) {
	var req domain.SendSMSRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请提供有效的手机号"})
		return
	}

	if err := h.svc.SendSMSCode(c.Request.Context(), req.Phone); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "验证码已发送",
		"phone":   req.Phone,
	})
}

// LoginByPhone 手机号+验证码登录
func (h *Handler) LoginByPhone(c *gin.Context) {
	var req domain.LoginByPhoneRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请提供手机号和验证码"})
		return
	}

	resp, err := h.svc.LoginByPhone(c.Request.Context(), req.Phone, req.Code)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"token": resp.Token, "user": resp.User})
}

// LoginByEmail 邮箱+密码登录
func (h *Handler) LoginByEmail(c *gin.Context) {
	var req domain.LoginByEmailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请提供邮箱和密码"})
		return
	}

	resp, err := h.svc.LoginByEmail(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"token": resp.Token, "user": resp.User})
}

// RegisterByEmail 邮箱注册
func (h *Handler) RegisterByEmail(c *gin.Context) {
	var req struct {
		Email    string `json:"email" binding:"required,email"`
		Password string `json:"password" binding:"required,min=6"`
		Name     string `json:"name" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请提供有效的注册信息"})
		return
	}

	resp, err := h.svc.RegisterByEmail(c.Request.Context(), req.Email, req.Password, req.Name)
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"token": resp.Token, "user": resp.User})
}

// GetMe 获取当前用户信息
func (h *Handler) GetMe(c *gin.Context) {
	userID := middleware.GetUserID(c)

	user, err := h.svc.GetUserByID(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "用户不存在"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"user": user})
}
