package analytics

import (
	"net/http"
	"strconv"
	"time"

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
	auth.GET("/analytics/events", h.Events)
	auth.GET("/analytics/customers", h.Customers)
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

// Events 点击事件列表
func (h *Handler) Events(c *gin.Context) {
	userID := middleware.GetUserID(c)
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	var total int64
	h.db.QueryRow(c.Request.Context(), `
		SELECT COUNT(*) FROM click_logs cl
		JOIN links l ON cl.link_id = l.id
		WHERE l.user_id = $1
	`, userID).Scan(&total)

	rows, err := h.db.Query(c.Request.Context(), `
		SELECT cl.id, cl.link_id, l.short_code, l.original_url, cl.platform, cl.ip, cl.referer, cl.created_at
		FROM click_logs cl
		JOIN links l ON cl.link_id = l.id
		WHERE l.user_id = $1
		ORDER BY cl.created_at DESC
		LIMIT $2 OFFSET $3
	`, userID, pageSize, (page-1)*pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
		return
	}
	defer rows.Close()

	type Event struct {
		ID          int64     `json:"id"`
		LinkID      int64     `json:"link_id"`
		ShortCode   string    `json:"short_code"`
		OriginalURL string    `json:"original_url"`
		Platform    string    `json:"platform"`
		IP          string    `json:"ip"`
		Referer     string    `json:"referer"`
		CreatedAt   time.Time `json:"created_at"`
	}

	var events []Event
	for rows.Next() {
		var e Event
		var platform, ip, referer *string
		rows.Scan(&e.ID, &e.LinkID, &e.ShortCode, &e.OriginalURL, &platform, &ip, &referer, &e.CreatedAt)
		if platform != nil {
			e.Platform = *platform
		}
		if ip != nil {
			e.IP = *ip
		}
		if referer != nil {
			e.Referer = *referer
		}
		events = append(events, e)
	}
	if events == nil {
		events = []Event{}
	}

	c.JSON(http.StatusOK, gin.H{
		"events":     events,
		"total":      total,
		"page":       page,
		"page_size":  pageSize,
	})
}

// Customers 独立访客列表
func (h *Handler) Customers(c *gin.Context) {
	userID := middleware.GetUserID(c)

	rows, err := h.db.Query(c.Request.Context(), `
		SELECT cl.ip,
		       COUNT(*) as click_count,
		       MAX(cl.created_at) as last_seen,
		       COUNT(DISTINCT cl.link_id) as unique_links
		FROM click_logs cl
		JOIN links l ON cl.link_id = l.id
		WHERE l.user_id = $1 AND cl.ip IS NOT NULL AND cl.ip != ''
		GROUP BY cl.ip
		ORDER BY click_count DESC
		LIMIT 50
	`, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
		return
	}
	defer rows.Close()

	type Customer struct {
		IP          string    `json:"ip"`
		ClickCount  int64     `json:"click_count"`
		UniqueLinks int64     `json:"unique_links"`
		LastSeen    time.Time `json:"last_seen"`
	}

	var customers []Customer
	for rows.Next() {
		var c Customer
		rows.Scan(&c.IP, &c.ClickCount, &c.LastSeen, &c.UniqueLinks)
		customers = append(customers, c)
	}
	if customers == nil {
		customers = []Customer{}
	}

	c.JSON(http.StatusOK, gin.H{"customers": customers})
}
