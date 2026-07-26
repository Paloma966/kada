package preview

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"golang.org/x/net/html"

	"github.com/chun/kada-backend/internal/domain"
)

// Fetcher 抓取目标 URL 的 Open Graph / Twitter Card 元数据
type Fetcher struct {
	client  *http.Client
	timeout time.Duration
}

// NewFetcher 创建元数据抓取器
func NewFetcher() *Fetcher {
	return &Fetcher{
		client: &http.Client{
			Timeout: 5 * time.Second,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) >= 3 {
					return fmt.Errorf("too many redirects")
				}
				return nil
			},
		},
		timeout: 5 * time.Second,
	}
}

// Fetch 抓取 URL 的元数据
func (f *Fetcher) Fetch(ctx context.Context, rawURL string) (*domain.LinkPreview, error) {
	// 补全协议
	if !strings.HasPrefix(rawURL, "http://") && !strings.HasPrefix(rawURL, "https://") {
		rawURL = "https://" + rawURL
	}

	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
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
