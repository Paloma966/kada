package domain

import "time"

// ==================== 请求/响应模型 ====================

// ---- 认证 ----

type SendSMSRequest struct {
	Phone string `json:"phone" binding:"required"`
}

type LoginByPhoneRequest struct {
	Phone string `json:"phone" binding:"required"`
	Code  string `json:"code" binding:"required"`
}

type LoginByEmailRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type AuthResponse struct {
	Token string    `json:"token"`
	User  UserInfo  `json:"user"`
}

type UserInfo struct {
	ID           int64   `json:"id"`
	Phone        *string `json:"phone"`
	Email        *string `json:"email"`
	Name         *string `json:"name"`
	Avatar       *string `json:"avatar"`
	WechatOpenID *string `json:"wechat_openid,omitempty"`
}

// ---- 短链 ----

type CreateLinkRequest struct {
	OriginalURL string  `json:"original_url" binding:"required,url"`
	Title       *string `json:"title"`
	Description *string `json:"description"`
	ImageURL    *string `json:"image_url"`
	Domain      *string `json:"domain"`
	Password    *string `json:"password"`
	ExpiresAt   *string `json:"expires_at"`
	WorkspaceID *int64  `json:"workspace_id"`
	UTMSource   *string `json:"utm_source"`
	UTMMedium   *string `json:"utm_medium"`
	UTMCampaign *string `json:"utm_campaign"`
	UTMTerm     *string `json:"utm_term"`
	UTMContent  *string `json:"utm_content"`
	IosURL      *string `json:"ios_url"`
	AndroidURL  *string `json:"android_url"`
}

type UpdateLinkRequest struct {
	OriginalURL *string `json:"original_url"`
	Title       *string `json:"title"`
	Description *string `json:"description"`
	ImageURL    *string `json:"image_url"`
	Password    *string `json:"password"`
	ExpiresAt   *string `json:"expires_at"`
	IsActive    *bool   `json:"is_active"`
}

type LinkInfo struct {
	ID          int64      `json:"id"`
	ShortCode   string     `json:"short_code"`
	ShortURL    string     `json:"short_url"`
	OriginalURL string     `json:"original_url"`
	Title       *string    `json:"title"`
	Description *string    `json:"description"`
	ImageURL    *string    `json:"image_url"`
	Domain      string     `json:"domain"`
	ClickCount  int64      `json:"click_count"`
	IsActive    bool       `json:"is_active"`
	ExpiresAt   *time.Time `json:"expires_at"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

type PaginatedLinks struct {
	Links      []LinkInfo `json:"links"`
	TotalCount int64      `json:"total_count"`
	Page       int        `json:"page"`
	PageSize   int        `json:"page_size"`
}

// ==================== 平台检测 ====================

type Platform string

const (
	PlatformBrowser      Platform = "browser"
	PlatformWechat       Platform = "wechat"
	PlatformQQ           Platform = "qq"
	PlatformWeibo        Platform = "weibo"
	PlatformXiaohongshu  Platform = "xiaohongshu"
	PlatformSMS          Platform = "sms"
	PlatformUnknown      Platform = "unknown"
)
