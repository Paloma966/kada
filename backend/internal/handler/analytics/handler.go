package analytics

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/chun/kada-backend/internal/middleware"
)

type Handler struct {
	db *pgxpool.Pool
}

func NewHandler(db *pgxpool.Pool) *Handler {
	return &Handler{db: db}
}

func (h *Handler) RegisterRoutes(r *gin.RouterGroup, authMW gin.HandlerFunc) {
	auth := r.Group("", authMW)
	auth.GET("/analytics/overview", h.Overview)
	auth.GET("/analytics/platforms", h.Platforms)
	auth.GET("/analytics/daily", h.DailyClicks)
}

// Overview 统计概览
func (h *Handler) Overview(c *gin.Context) {
	userID := middleware.GetUserID(c)

	var totalLinks, totalClicks int64
	h.db.QueryRow(c.Request.Context(),
		`SELECT COUNT(*), COALESCE(SUM(click_count), 0) FROM links WHERE user_id = $1`, userID,
	).Scan(&totalLinks, &totalClicks)

	c.JSON(http.StatusOK, gin.H{
		"total_links":  totalLinks,
		"total_clicks": totalClicks,
	})
}

// Platforms 平台来源分布
func (h *Handler) Platforms(c *gin.Context) {
	userID := middleware.GetUserID(c)

	rows, err := h.db.Query(c.Request.Context(), `
		SELECT COALESCE(cl.platform, 'browser'), COUNT(*)
		FROM click_logs cl
		JOIN links l ON cl.link_id = l.id
		WHERE l.user_id = $1
		GROUP BY cl.platform
		ORDER BY COUNT(*) DESC
	`, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
		return
	}
	defer rows.Close()

	type PlatformStat struct {
		Platform string `json:"platform"`
		Count    int64  `json:"count"`
	}

	var stats []PlatformStat
	for rows.Next() {
		var s PlatformStat
		rows.Scan(&s.Platform, &s.Count)
		stats = append(stats, s)
	}
	if stats == nil {
		stats = []PlatformStat{}
	}

	c.JSON(http.StatusOK, gin.H{"platforms": stats})
}

// DailyClicks 每日点击量（最近30天）
func (h *Handler) DailyClicks(c *gin.Context) {
	userID := middleware.GetUserID(c)

	rows, err := h.db.Query(c.Request.Context(), `
		SELECT DATE(cl.created_at) as date, COUNT(*)
		FROM click_logs cl
		JOIN links l ON cl.link_id = l.id
		WHERE l.user_id = $1 AND cl.created_at > NOW() - INTERVAL '30 days'
		GROUP BY DATE(cl.created_at)
		ORDER BY date
	`, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
		return
	}
	defer rows.Close()

	type DailyStat struct {
		Date  string `json:"date"`
		Count int64  `json:"count"`
	}

	var stats []DailyStat
	for rows.Next() {
		var s DailyStat
		var date interface{}
		rows.Scan(&date, &s.Count)
		if t, ok := date.(interface{ Format(string) string }); ok {
			s.Date = t.Format("2006-01-02")
		}
		stats = append(stats, s)
	}
	if stats == nil {
		stats = []DailyStat{}
	}

	c.JSON(http.StatusOK, gin.H{"daily": stats})
}
