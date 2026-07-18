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
	ShortCode   *string `json:"short_code"`
	Title       *string `json:"title"`
	Description *string `json:"description"`
	ImageURL    *string `json:"image_url"`
	Domain      *string `json:"domain"`
	Password    *string `json:"password"`
	ExpiresAt   *string `json:"expires_at"`
	FolderID    *int64  `json:"folder_id"`
	TagIDs      []int64 `json:"tag_ids"`
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
	ShortCode   *string `json:"short_code"`
	Title       *string `json:"title"`
	Description *string `json:"description"`
	ImageURL    *string `json:"image_url"`
	Domain      *string `json:"domain"`
	Password    *string `json:"password"`
	ExpiresAt   *string `json:"expires_at"`
	IsActive    *bool   `json:"is_active"`
	FolderID    *int64  `json:"folder_id"`
	TagIDs      []int64 `json:"tag_ids"`
	UTMSource   *string `json:"utm_source"`
	UTMMedium   *string `json:"utm_medium"`
	UTMCampaign *string `json:"utm_campaign"`
	UTMTerm     *string `json:"utm_term"`
	UTMContent  *string `json:"utm_content"`
	IosURL      *string `json:"ios_url"`
	AndroidURL  *string `json:"android_url"`
}

type LinkTagInfo struct {
	ID    int64  `json:"id"`
	Name  string `json:"name"`
	Color string `json:"color"`
}

type LinkInfo struct {
	ID           int64         `json:"id"`
	ShortCode    string        `json:"short_code"`
	ShortURL     string        `json:"short_url"`
	OriginalURL  string        `json:"original_url"`
	Title        *string       `json:"title"`
	Description  *string       `json:"description"`
	ImageURL     *string       `json:"image_url"`
	Domain       string        `json:"domain"`
	ClickCount   int64         `json:"click_count"`
	IsActive     bool          `json:"is_active"`
	ExpiresAt    *time.Time    `json:"expires_at"`
	CreatedAt    time.Time     `json:"created_at"`
	UpdatedAt    time.Time     `json:"updated_at"`
	FolderID     *int64        `json:"folder_id"`
	FolderName   *string       `json:"folder_name"`
	Tags         []LinkTagInfo `json:"tags"`
	PasswordHash *string       `json:"password_hash,omitempty"`
	UTMSource    *string       `json:"utm_source,omitempty"`
	UTMMedium    *string       `json:"utm_medium,omitempty"`
	UTMCampaign  *string       `json:"utm_campaign,omitempty"`
	UTMTerm      *string       `json:"utm_term,omitempty"`
	UTMContent   *string       `json:"utm_content,omitempty"`
	IosURL       *string       `json:"ios_url,omitempty"`
	AndroidURL   *string       `json:"android_url,omitempty"`
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

// ==================== 文件夹 ====================

type Folder struct {
	ID        int64     `json:"id"`
	UserID    int64     `json:"user_id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	LinkCount int64     `json:"link_count,omitempty"`
}

type CreateFolderRequest struct {
	Name string `json:"name" binding:"required"`
}

// ==================== 标签 ====================

type Tag struct {
	ID        int64     `json:"id"`
	UserID    int64     `json:"user_id"`
	Name      string    `json:"name"`
	Color     string    `json:"color"`
	CreatedAt time.Time `json:"created_at"`
}

type CreateTagRequest struct {
	Name  string  `json:"name" binding:"required"`
	Color *string `json:"color"`
}

// ==================== 域名 ====================

type Domain struct {
	ID         int64      `json:"id"`
	UserID     int64      `json:"user_id"`
	Name       string     `json:"name"`
	Verified   bool       `json:"verified"`
	VerifiedAt *time.Time `json:"verified_at"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
}

type CreateDomainRequest struct {
	Name string `json:"name" binding:"required"`
}

// ==================== UTM 模板 ====================

type UTMTemplate struct {
	ID          int64     `json:"id"`
	UserID      int64     `json:"user_id"`
	Name        string    `json:"name"`
	UTMSource   *string   `json:"utm_source"`
	UTMMedium   *string   `json:"utm_medium"`
	UTMCampaign *string   `json:"utm_campaign"`
	UTMTerm     *string   `json:"utm_term"`
	UTMContent  *string   `json:"utm_content"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type CreateUTMTemplateRequest struct {
	Name        string  `json:"name" binding:"required"`
	UTMSource   *string `json:"utm_source"`
	UTMMedium   *string `json:"utm_medium"`
	UTMCampaign *string `json:"utm_campaign"`
	UTMTerm     *string `json:"utm_term"`
	UTMContent  *string `json:"utm_content"`
}

// ==================== API Token ====================

type APIToken struct {
	ID        int64      `json:"id"`
	UserID    int64      `json:"user_id"`
	Name      string     `json:"name"`
	LastUsed  *time.Time `json:"last_used"`
	CreatedAt time.Time  `json:"created_at"`
}

type CreateAPITokenRequest struct {
	Name string `json:"name" binding:"required"`
}

type CreateAPITokenResponse struct {
	Token     string   `json:"token"`
	APIToken  APIToken `json:"api_token"`
}
