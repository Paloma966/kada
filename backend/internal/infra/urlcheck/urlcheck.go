// Package urlcheck 提供目标 URL 安全性校验。
// 短链目标 URL 会被写入数据库并在跳转/引导页中执行（window.location.href），
// 必须限制为 http/https 协议，防止 javascript:/data: 等协议造成存储型 XSS。
package urlcheck

import (
	"net/url"
	"strings"
)

// IsSafeTarget 校验目标 URL 是否为可安全跳转的 http/https 绝对 URL。
func IsSafeTarget(raw string) bool {
	if raw == "" || len(raw) > 2048 {
		return false
	}
	u, err := url.Parse(raw)
	if err != nil {
		return false
	}
	switch strings.ToLower(u.Scheme) {
	case "http", "https":
		return u.Host != ""
	default:
		return false
	}
}
