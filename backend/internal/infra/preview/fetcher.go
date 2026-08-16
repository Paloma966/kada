package preview

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"golang.org/x/net/html"

	"github.com/chun/kada-backend/internal/domain"
)

// blockedCIDRs 内网/保留地址段：阻止 SSRF 访问本机、云元数据与内网服务
var blockedCIDRs = []string{
	"0.0.0.0/8",       // 本网络
	"10.0.0.0/8",      // RFC1918 私有
	"100.64.0.0/10",   // CGNAT
	"127.0.0.0/8",     // loopback
	"169.254.0.0/16",  // link-local（含云元数据 169.254.169.254）
	"172.16.0.0/12",   // RFC1918 私有
	"192.0.0.0/24",    // IETF 保留
	"192.0.2.0/24",    // TEST-NET-1
	"192.168.0.0/16",  // RFC1918 私有
	"198.18.0.0/15",   // 基准测试
	"198.51.100.0/24", // TEST-NET-2
	"203.0.113.0/24",  // TEST-NET-3
	"224.0.0.0/4",     // 组播
	"240.0.0.0/4",     // 保留
	"::/128",          // 未指定
	"::1/128",         // loopback v6
	"fc00::/7",        // ULA
	"fe80::/10",       // link-local v6
	"ff00::/8",        // 组播 v6
}

var blockedNets = func() []*net.IPNet {
	nets := make([]*net.IPNet, 0, len(blockedCIDRs))
	for _, cidr := range blockedCIDRs {
		_, n, err := net.ParseCIDR(cidr)
		if err != nil {
			continue
		}
		nets = append(nets, n)
	}
	return nets
}()

// isBlockedIP 判断 IP 是否属于内网/保留地址段
func isBlockedIP(ip net.IP) bool {
	for _, n := range blockedNets {
		if n.Contains(ip) {
			return true
		}
	}
	return false
}

// validateHost 校验主机：IP 直连检查地址段；域名解析后逐一检查（防 DNS rebinding 前置防线）
func validateHost(ctx context.Context, host string) error {
	host = strings.TrimSuffix(host, ".")
	if host == "" {
		return fmt.Errorf("empty host")
	}
	if ip := net.ParseIP(host); ip != nil {
		if isBlockedIP(ip) {
			return fmt.Errorf("host %s 是内网/保留地址，已阻止访问", host)
		}
		return nil
	}
	ips, err := net.DefaultResolver.LookupIPAddr(ctx, host)
	if err != nil {
		return fmt.Errorf("域名解析失败: %w", err)
	}
	if len(ips) == 0 {
		return fmt.Errorf("域名 %s 未解析到任何地址", host)
	}
	for _, a := range ips {
		if isBlockedIP(a.IP) {
			return fmt.Errorf("域名 %s 解析到内网地址 %s，已阻止访问", host, a.IP)
		}
	}
	return nil
}

// Fetcher 抓取目标 URL 的 Open Graph / Twitter Card 元数据
type Fetcher struct {
	client  *http.Client
	timeout time.Duration
}

// NewFetcher 创建元数据抓取器（含 SSRF 防护：仅 http/https、仅 80/443、内网地址黑名单、拨号时复核）
func NewFetcher() *Fetcher {
	dialer := &net.Dialer{Timeout: 5 * time.Second}
	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			host, _, err := net.SplitHostPort(addr)
			if err != nil {
				return nil, err
			}
			// 拨号前再次校验（DNS rebinding 最终防线）
			if err := validateHost(ctx, host); err != nil {
				return nil, err
			}
			return dialer.DialContext(ctx, network, addr)
		},
	}
	return &Fetcher{
		client: &http.Client{
			Timeout:   5 * time.Second,
			Transport: transport,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) >= 3 {
					return fmt.Errorf("too many redirects")
				}
				// 每个重定向目标同样校验
				return validateTarget(req.Context(), req.URL.String())
			},
		},
		timeout: 5 * time.Second,
	}
}

// validateTarget 校验抓取目标：协议、端口、主机地址段
func validateTarget(ctx context.Context, rawURL string) error {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}
	switch strings.ToLower(parsedURL.Scheme) {
	case "http", "https":
	default:
		return fmt.Errorf("不支持的协议 %q，仅允许 http/https", parsedURL.Scheme)
	}

	host := parsedURL.Hostname()
	if host == "" {
		return fmt.Errorf("URL 缺少主机名")
	}

	port := parsedURL.Port()
	if port == "" {
		if strings.EqualFold(parsedURL.Scheme, "https") {
			port = "443"
		} else {
			port = "80"
		}
	}
	if port != "80" && port != "443" {
		return fmt.Errorf("仅允许访问 80/443 端口，收到 %s", port)
	}

	return validateHost(ctx, host)
}

// Fetch 抓取 URL 的元数据
func (f *Fetcher) Fetch(ctx context.Context, rawURL string) (*domain.LinkPreview, error) {
	rawURL = strings.TrimSpace(rawURL)

	// 无协议时补全为 https（如 "example.com/path"）
	if parsed, err := url.Parse(rawURL); err == nil && parsed.Scheme == "" {
		rawURL = "https://" + rawURL
	}

	if err := validateTarget(ctx, rawURL); err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "GET", rawURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("User-Agent", "Kada-LinkPreview/1.0 (compatible; +https://kada.click)")
	req.Header.Set("Accept", "text/html,application/xhtml+xml")

	resp, err := f.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch URL: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("URL returned status %d", resp.StatusCode)
	}

	// 只解析前 1MB
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}

	meta := parseHTML(strings.NewReader(string(body)))

	// Fallback: 用域名作为标题
	parsedURL, _ := url.Parse(rawURL)
	if meta.Title == "" {
		meta.Title = parsedURL.Host
	}

	// Favicon
	meta.FaviconURL = fmt.Sprintf("https://www.google.com/s2/favicons?domain=%s&sz=32", parsedURL.Host)

	return &meta, nil
}

// parseHTML 解析 HTML 中的 OG / Twitter Card 标签
func parseHTML(r io.Reader) domain.LinkPreview {
	var meta domain.LinkPreview
	z := html.NewTokenizer(r)

	for {
		tt := z.Next()
		if tt == html.ErrorToken {
			break
		}
		if tt != html.StartTagToken && tt != html.SelfClosingTagToken {
			continue
		}

		tagName, hasAttr := z.TagName()
		if string(tagName) != "meta" || !hasAttr {
			continue
		}

		var prop, content string
		for {
			key, val, more := z.TagAttr()
			k := string(key)
			v := string(val)
			if k == "property" || k == "name" {
				prop = strings.ToLower(v)
			}
			if k == "content" {
				content = v
			}
			if !more {
				break
			}
		}

		if prop == "" || content == "" {
			continue
		}

		switch prop {
		case "og:title", "twitter:title":
			if meta.Title == "" {
				meta.Title = content
			}
		case "og:description", "twitter:description":
			if meta.Description == "" {
				meta.Description = content
			}
		case "og:image", "twitter:image":
			if meta.ImageURL == "" {
				meta.ImageURL = content
			}
		}
	}

	// 如果没有 OG 标签，上层会用域名作为 fallback 标题
	return meta
}
