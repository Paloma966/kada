package workspace

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/chun/kada-backend/internal/domain"
	"github.com/chun/kada-backend/internal/middleware"
	"github.com/chun/kada-backend/internal/service"
)

type Handler struct {
	svc *service.WorkspaceService
}

func NewHandler(svc *service.WorkspaceService) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) RegisterRoutes(r *gin.RouterGroup, authMW gin.HandlerFunc) {
	auth := r.Group("", authMW)
	auth.GET("/workspaces", h.List)
	auth.POST("/workspaces", h.Create)
	auth.GET("/workspaces/:id", h.Get)
	auth.PATCH("/workspaces/:id", h.Update)
	auth.DELETE("/workspaces/:id", h.Delete)
}

// List 获取工作区列表
func (h *Handler) List(c *gin.Context) {
	userID := middleware.GetUserID(c)

	workspaces, err := h.svc.List(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"workspaces": workspaces})
}

// Create 创建工作区
func (h *Handler) Create(c *gin.Context) {
	userID := middleware.GetUserID(c)

	var req domain.CreateWorkspaceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请提供名称和 slug"})
		return
	}

	w, err := h.svc.Create(c.Request.Context(), userID, req)
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"workspace": w})
}

// Get 获取单个工作区
func (h *Handler) Get(c *gin.Context) {
	userID := middleware.GetUserID(c)
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的工作区 ID"})
		return
	}

	w, err := h.svc.GetByID(c.Request.Context(), id, userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"workspace": w})
}

// Update 更新工作区
func (h *Handler) Update(c *gin.Context) {
	userID := middleware.GetUserID(c)
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的工作区 ID"})
		return
	}

	var req domain.UpdateWorkspaceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求格式无效"})
		return
	}

	w, err := h.svc.Update(c.Request.Context(), id, userID, req)
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"workspace": w})
}

// Delete 删除工作区
func (h *Handler) Delete(c *gin.Context) {
	userID := middleware.GetUserID(c)
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的工作区 ID"})
		return
	}

	if err := h.svc.Delete(c.Request.Context(), id, userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "工作区已删除"})
}
