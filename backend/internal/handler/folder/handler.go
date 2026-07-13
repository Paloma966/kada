package folder

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/chun/kada-backend/internal/domain"
	"github.com/chun/kada-backend/internal/middleware"
	"github.com/chun/kada-backend/internal/service"
)

type Handler struct {
	svc *service.FolderService
}

func NewHandler(svc *service.FolderService) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) RegisterRoutes(r *gin.RouterGroup, authMW gin.HandlerFunc) {
	auth := r.Group("", authMW)
	auth.POST("/folders", h.Create)
	auth.GET("/folders", h.List)
	auth.PATCH("/folders/:id", h.Update)
	auth.DELETE("/folders/:id", h.Delete)
}

func (h *Handler) Create(c *gin.Context) {
	var req domain.CreateFolderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请提供文件夹名称"})
		return
	}
	f, err := h.svc.Create(c.Request.Context(), middleware.GetUserID(c), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"folder": f})
}

func (h *Handler) List(c *gin.Context) {
	folders, err := h.svc.List(c.Request.Context(), middleware.GetUserID(c))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"folders": folders})
}

func (h *Handler) Update(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的ID"})
		return
	}
	var req domain.CreateFolderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请提供文件夹名称"})
		return
	}
	f, err := h.svc.Update(c.Request.Context(), middleware.GetUserID(c), id, req.Name)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"folder": f})
}

func (h *Handler) Delete(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的ID"})
		return
	}
	if err := h.svc.Delete(c.Request.Context(), middleware.GetUserID(c), id); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "文件夹已删除"})
}
