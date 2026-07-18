package link

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/chun/kada-backend/internal/domain"
	"github.com/chun/kada-backend/internal/middleware"
	"github.com/chun/kada-backend/internal/service"
)

type Handler struct {
	svc *service.LinkService
}

func NewHandler(svc *service.LinkService) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) RegisterRoutes(r *gin.RouterGroup, authMW gin.HandlerFunc) {
	r.Use(authMW)
	r.POST("/links", h.Create)
	r.GET("/links", h.List)
	r.POST("/links/batch-delete", h.BatchDelete)
	r.POST("/links/batch-tag", h.BatchTag)
	r.GET("/links/export", h.ExportCSV)
	r.GET("/links/:id", h.Get)
	r.PATCH("/links/:id", h.Update)
	r.DELETE("/links/:id", h.Delete)
}

// Create 创建短链接
func (h *Handler) Create(c *gin.Context) {
	var req domain.CreateLinkRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请提供有效的链接信息: " + err.Error()})
		return
	}

	link, err := h.svc.Create(c.Request.Context(), middleware.GetUserID(c), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"link": link})
}

// Get 获取单个链接
func (h *Handler) Get(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的链接ID"})
		return
	}

	link, err := h.svc.GetByID(c.Request.Context(), id, middleware.GetUserID(c))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"link": link})
}

// List 获取链接列表
func (h *Handler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	search := c.Query("search")
	folderID, _ := strconv.ParseInt(c.Query("folder_id"), 10, 64)
	tagID, _ := strconv.ParseInt(c.Query("tag_id"), 10, 64)
	sort := c.DefaultQuery("sort", "created_desc")

	result, err := h.svc.List(c.Request.Context(), middleware.GetUserID(c), page, pageSize, search, folderID, tagID, sort)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"links":       result.Links,
		"total_count": result.TotalCount,
		"page":        result.Page,
		"page_size":   result.PageSize,
	})
}

// Update 更新链接
func (h *Handler) Update(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的链接ID"})
		return
	}

	var req domain.UpdateLinkRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数错误"})
		return
	}

	link, err := h.svc.Update(c.Request.Context(), id, middleware.GetUserID(c), req)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"link": link})
}

// Delete 删除链接
func (h *Handler) Delete(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的链接ID"})
		return
	}

	if err := h.svc.Delete(c.Request.Context(), id, middleware.GetUserID(c)); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "链接已删除"})
}

// BatchDelete 批量删除链接
func (h *Handler) BatchDelete(c *gin.Context) {
	var req struct {
		IDs []int64 `json:"ids" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请提供链接ID列表"})
		return
	}

	count, err := h.svc.BatchDelete(c.Request.Context(), req.IDs, middleware.GetUserID(c))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"deleted": count})
}

// BatchTag 批量打标签
func (h *Handler) BatchTag(c *gin.Context) {
	var req struct {
		IDs  []int64 `json:"ids" binding:"required"`
		TagID int64  `json:"tag_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请提供链接ID和标签ID"})
		return
	}

	if err := h.svc.BatchTag(c.Request.Context(), req.IDs, req.TagID, middleware.GetUserID(c)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "批量打标签成功"})
}

// ExportCSV 导出链接数据为 CSV
func (h *Handler) ExportCSV(c *gin.Context) {
	csvStr, err := h.svc.ExportCSV(c.Request.Context(), middleware.GetUserID(c))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition", `attachment; filename="kada-links.csv"`)
	c.String(http.StatusOK, csvStr)
}
