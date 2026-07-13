package ua

import (
	"strings"

	"github.com/chun/kada-backend/internal/domain"
)

// Detect 根据 User-Agent 判断来源平台
func Detect(userAgent string) domain.Platform {
	ua := strings.ToLower(userAgent)

	// 微信内置浏览器
	if strings.Contains(ua, "micromessenger") {
		return domain.PlatformWechat
	}

	// QQ 内置浏览器（注意顺序：要先检测 QQ 再检测微信，因为 QQ 可能也含 MQQBrowser）
	if strings.Contains(ua, "qq/") || strings.Contains(ua, "mqqbrowser") {
		return domain.PlatformQQ
	}

	// 微博
	if strings.Contains(ua, "weibo") || strings.Contains(ua, "weibo__") {
		return domain.PlatformWeibo
	}

	// 小红书
	if strings.Contains(ua, "xhs") || strings.Contains(ua, "redapp") {
		return domain.PlatformXiaohongshu
	}

	return domain.PlatformBrowser
}

// NeedsIntermediatePage 判断是否需要中间引导页（而非直接 302）
func NeedsIntermediatePage(platform domain.Platform) bool {
	switch platform {
	case domain.PlatformWechat, domain.PlatformQQ, domain.PlatformXiaohongshu:
		return true
	default:
		return false
	}
}

// PlatformName 返回中文平台名
func PlatformName(platform domain.Platform) string {
	switch platform {
	case domain.PlatformWechat:
		return "微信"
	case domain.PlatformQQ:
		return "QQ"
	case domain.PlatformWeibo:
		return "微博"
	case domain.PlatformXiaohongshu:
		return "小红书"
	case domain.PlatformSMS:
		return "短信"
	default:
		return "浏览器"
	}
}
