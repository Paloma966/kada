package domain

import (
	"errors"
	"time"
)

var ErrLinkNotFound = errors.New("link not found")

// RateLimitError is a refusal caused by a quota rather than by anything wrong with the request.
//
// It is a type rather than a sentinel because the caller needs two things a bare error cannot carry: the
// message to show the user, and how long to wait. The HTTP handler maps it to 429 + Retry-After; every
// other failure from the auth service is the caller's fault and stays a 400.
type RateLimitError struct {
	Message           string
	RetryAfterSeconds int
}

func (e *RateLimitError) Error() string { return e.Message }

// ==================== Request/Response models ====================

// ---- Auth ----

// SendSMSRequest asks for a login code.
//
// The captcha fields are required, not optional: the point of the challenge is that no SMS can be
// triggered without a human having solved one, and an optional challenge is one an attacker simply omits.
type SendSMSRequest struct {
	Phone       string `json:"phone" binding:"required"`
	CaptchaID   string `json:"captcha_id" binding:"required"`
	CaptchaCode string `json:"captcha_code" binding:"required"`
}

// CaptchaResponse is the graphical challenge: an opaque id to send back with the answer, and the image
// as a data URI so the client needs no second request and no separate image endpoint.
type CaptchaResponse struct {
	ID    string `json:"captcha_id"`
	Image string `json:"image"`
}

type LoginByPhoneRequest struct {
	Phone string `json:"phone" binding:"required"`
	Code  string `json:"code" binding:"required"`
}

type AuthResponse struct {
	Token string   `json:"token"`
	User  UserInfo `json:"user"`
}

// UserInfo is what a caller learns about an account: the phone number and nothing else.
//
// The phone is the whole of sign-in - the identifier, the second factor and the unique key - so there is
// no name to display, no email to edit and no avatar to render. Everything in this struct is published to
// every client and cached in localStorage, which is why its shape is pinned by a test.
type UserInfo struct {
	ID    int64   `json:"id"`
	Phone *string `json:"phone"`
}

// ---- Links ----

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

// ==================== Platform detection ====================

type Platform string

const (
	PlatformBrowser     Platform = "browser"
	PlatformWechat      Platform = "wechat"
	PlatformQQ          Platform = "qq"
	PlatformWeibo       Platform = "weibo"
	PlatformXiaohongshu Platform = "xiaohongshu"
	PlatformSMS         Platform = "sms"
	PlatformUnknown     Platform = "unknown"
)

// DeepLink represents one deeplink fallback option
type DeepLink struct {
	Name   string `json:"name"`   // option name, e.g. "Chrome Intent"
	Scheme string `json:"scheme"` // URL scheme, e.g. "intent://..."
}

// ==================== Folders ====================

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

// ==================== Tags ====================

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

// ==================== Domains ====================

type Domain struct {
	ID               int64      `json:"id"`
	UserID           int64      `json:"user_id"`
	Name             string     `json:"name"`
	Verified         bool       `json:"verified"`
	VerifiedAt       *time.Time `json:"verified_at"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
	VerificationCode string     `json:"verification_code,omitempty"`
}

type CreateDomainRequest struct {
	Name string `json:"name" binding:"required"`
}

// ==================== UTM templates ====================

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
	Token    string   `json:"token"`
	APIToken APIToken `json:"api_token"`
}

// ---- Workspaces ----

type Workspace struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Slug      string    `json:"slug"`
	UserID    int64     `json:"user_id"`
	LinkCount int64     `json:"link_count"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type CreateWorkspaceRequest struct {
	Name string `json:"name" binding:"required"`
	Slug string `json:"slug" binding:"required"`
}

type UpdateWorkspaceRequest struct {
	Name *string `json:"name"`
	Slug *string `json:"slug"`
}

type WorkspaceListResponse struct {
	Workspaces []Workspace `json:"workspaces"`
}

// ==================== Link preview ====================

type LinkPreviewRequest struct {
	URL string `json:"url" binding:"required,url"`
}

type LinkPreview struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	ImageURL    string `json:"image_url"`
	FaviconURL  string `json:"favicon_url"`
}
