package ua

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/chun/kada-backend/internal/domain"
)

// Detect determines the source platform from the User-Agent
func Detect(userAgent string) domain.Platform {
	ua := strings.ToLower(userAgent)

	// WeChat in-app browser
	if strings.Contains(ua, "micromessenger") {
		return domain.PlatformWechat
	}

	// QQ in-app browser (note the order: check QQ before WeChat, since QQ UAs may also contain MQQBrowser)
	if strings.Contains(ua, "qq/") || strings.Contains(ua, "mqqbrowser") {
		return domain.PlatformQQ
	}

	// Weibo
	if strings.Contains(ua, "weibo") || strings.Contains(ua, "weibo__") {
		return domain.PlatformWeibo
	}

	// Xiaohongshu
	if strings.Contains(ua, "xhs") || strings.Contains(ua, "redapp") {
		return domain.PlatformXiaohongshu
	}

	return domain.PlatformBrowser
}

// NeedsIntermediatePage reports whether an intermediate guidance page is required (instead of a direct 302)
func NeedsIntermediatePage(platform domain.Platform) bool {
	switch platform {
	case domain.PlatformWechat, domain.PlatformQQ, domain.PlatformXiaohongshu:
		return true
	default:
		return false
	}
}

// PlatformName returns the platform display name
func PlatformName(platform domain.Platform) string {
	switch platform {
	case domain.PlatformWechat:
		return "WeChat"
	case domain.PlatformQQ:
		return "QQ"
	case domain.PlatformWeibo:
		return "Weibo"
	case domain.PlatformXiaohongshu:
		return "Xiaohongshu"
	case domain.PlatformSMS:
		return "SMS"
	default:
		return "Browser"
	}
}

// PlatformTips returns the platform-specific guidance copy
func PlatformTips(platform domain.Platform) string {
	switch platform {
	case domain.PlatformWechat:
		return "Please open the link in WeChat, or tap the menu in the top-right corner and choose \"Open in Browser\""
	case domain.PlatformQQ:
		return "The built-in QQ browser may block page redirects; please open the link in an external browser"
	case domain.PlatformXiaohongshu:
		return "Xiaohongshu does not support direct external links; please copy the link and open it in a browser"
	case domain.PlatformWeibo:
		return "Opening links inside Weibo may be restricted; please open it in a browser"
	default:
		return "Opening the link for you..."
	}
}

// GetDeeplinks builds per-platform deeplink fallback options for the target URL
func GetDeeplinks(targetURL string) []domain.DeepLink {
	// Strip the protocol prefix for use in the intent scheme
	stripped := strings.TrimPrefix(targetURL, "https://")
	stripped = strings.TrimPrefix(stripped, "http://")
	encoded := url.QueryEscape(targetURL)

	links := []domain.DeepLink{
		{
			Name:   "Open directly",
			Scheme: targetURL,
		},
		{
			Name:   "Chrome",
			Scheme: fmt.Sprintf("intent://%s#Intent;scheme=https;package=com.android.chrome;end", stripped),
		},
		{
			Name:   "System browser",
			Scheme: fmt.Sprintf("intent://%s#Intent;scheme=https;end", stripped),
		},
	}

	// In-WeChat navigation: try relaying through a WeChat URL scheme
	_ = encoded // reserved for WeChat-specific schemes

	return links
}
