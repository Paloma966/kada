package redirect

import (
	"html/template"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/chun/kada-backend/internal/domain"
	"github.com/chun/kada-backend/internal/infra/ua"
	"github.com/chun/kada-backend/internal/service"
)

type Handler struct {
	svc *service.LinkService
}

func NewHandler(svc *service.LinkService) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) RegisterRoutes(r *gin.Engine) {
	r.GET("/r/:code", h.Redirect)
}

// Redirect 短链重定向（含平台检测）
func (h *Handler) Redirect(c *gin.Context) {
	code := c.Param("code")

	link, err := h.svc.GetByCode(c.Request.Context(), code)
	if err != nil {
		c.String(http.StatusNotFound, "链接不存在或已过期")
		return
	}

	userAgent := c.GetHeader("User-Agent")
	platform := ua.Detect(userAgent)
	ip := c.ClientIP()
	referer := c.GetHeader("Referer")

	// 记录点击
	clickPlatform := string(platform)
	go h.svc.LogClick(c.Request.Context(), link.ID, ip, userAgent, clickPlatform, referer)

	// 判断是否需要中间引导页
	if ua.NeedsIntermediatePage(platform) {
		h.renderIntermediatePage(c, link.OriginalURL, platform)
		return
	}

	// 普通浏览器直接 302 跳转
	c.Redirect(http.StatusFound, link.OriginalURL)
}

// renderIntermediatePage 渲染中间引导页（微信/QQ等）
func (h *Handler) renderIntermediatePage(c *gin.Context, targetURL string, platform domain.Platform) {
	page := template.Must(template.New("guide").Parse(guidePageHTML))
	platformName := ua.PlatformName(platform)

	data := struct {
		TargetURL    string
		PlatformName string
		Platform     string
	}{
		TargetURL:    targetURL,
		PlatformName: platformName,
		Platform:     string(platform),
	}

	c.Header("Content-Type", "text/html; charset=utf-8")
	page.Execute(c.Writer, data)
}

const guidePageHTML = `<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>即将打开链接</title>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body {
            font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
            background: #f5f5f5;
            display: flex; justify-content: center; align-items: center;
            min-height: 100vh; padding: 20px;
        }
        .card {
            background: white; border-radius: 16px; padding: 40px 32px;
            max-width: 400px; width: 100%; text-align: center;
            box-shadow: 0 2px 20px rgba(0,0,0,0.08);
        }
        .icon { font-size: 48px; margin-bottom: 16px; }
        .title { font-size: 20px; font-weight: 600; color: #1a1a1a; margin-bottom: 8px; }
        .subtitle { font-size: 14px; color: #666; margin-bottom: 24px; line-height: 1.6; }
        .btn {
            display: block; width: 100%; padding: 14px; border-radius: 10px;
            font-size: 16px; font-weight: 500; cursor: pointer; border: none;
            margin-bottom: 12px; transition: opacity 0.2s;
        }
        .btn:hover { opacity: 0.9; }
        .btn-primary { background: #6366f1; color: white; }
        .btn-secondary { background: #f0f0f0; color: #333; }
        .target-url {
            font-size: 12px; color: #999; margin-top: 16px;
            word-break: break-all;
        }
    </style>
</head>
<body>
    <div class="card">
        <div class="icon">🔗</div>
        <div class="title">即将打开链接</div>
        <div class="subtitle">
            您正在 <strong>{{.PlatformName}}</strong> 中访问此链接<br>
            如页面无法正常打开，请尝试：<br>
            1. 点击右上角「在浏览器中打开」<br>
            2. 复制链接到浏览器访问
        </div>
        <button class="btn btn-primary" onclick="openInBrowser()">在浏览器中打开</button>
        <button class="btn btn-secondary" onclick="copyLink()">复制链接</button>
        <p id="copy-success" style="color: #22c55e; margin-top: 8px; display: none;">✅ 链接已复制</p>
        <p class="target-url" id="url">{{.TargetURL}}</p>
    </div>
    <script>
        const url = "{{.TargetURL}}";
        function openInBrowser() {
            window.location.href = url;
        }
        function copyLink() {
            if (navigator.clipboard) {
                navigator.clipboard.writeText(url).then(function() {
                    document.getElementById('copy-success').style.display = 'block';
                });
            }
        }
        // 自动尝试跳转
        setTimeout(function() { window.location.href = url; }, 1500);
    </script>
</body>
</html>`
