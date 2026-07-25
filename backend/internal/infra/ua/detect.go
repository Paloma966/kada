package ua

import (
	"fmt"
	"net/url"
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

// PlatformTips 返回平台专属引导文案
func PlatformTips(platform domain.Platform) string {
	switch platform {
	case domain.PlatformWechat:
		return "请在微信内打开链接，或点击右上角菜单选择「在浏览器中打开」"
	case domain.PlatformQQ:
		return "QQ 内置浏览器可能限制页面跳转，建议使用外部浏览器打开"
	case domain.PlatformXiaohongshu:
		return "小红书暂不支持直接跳转外链，请复制链接后在浏览器中打开"
	case domain.PlatformWeibo:
		return "微博内打开链接可能受限，建议在浏览器中打开"
	default:
		return "正在为您打开链接..."
	}
}

// GetDeeplinks 为目标 URL 生成各平台的 deeplink 尝试方案
func GetDeeplinks(targetURL string) []domain.DeepLink {
	// 去掉 protocol 前缀，用于 intent scheme
	stripped := strings.TrimPrefix(targetURL, "https://")
	stripped = strings.TrimPrefix(stripped, "http://")
	encoded := url.QueryEscape(targetURL)

	links := []domain.DeepLink{
		{
			Name:   "直接打开",
			Scheme: targetURL,
		},
		{
			Name:   "Chrome",
			Scheme: fmt.Sprintf("intent://%s#Intent;scheme=https;package=com.android.chrome;end", stripped),
		},
		{
			Name:   "系统浏览器",
			Scheme: fmt.Sprintf("intent://%s#Intent;scheme=https;end", stripped),
		},
	}

	// 微信内部跳转：尝试通过微信 URL scheme 中转
	_ = encoded // reserved for WeChat-specific schemes

	return links
}
