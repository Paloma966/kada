package redirect

import (
	"bytes"
	"context"
	"html/template"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	qrcode "github.com/skip2/go-qrcode"

	"github.com/chun/kada-backend/internal/domain"
	"github.com/chun/kada-backend/internal/infra/ua"
	"github.com/chun/kada-backend/internal/infra/urlcheck"
	"github.com/chun/kada-backend/internal/middleware"
)

// LinkService 短链服务接口（方便测试 mock）
type LinkService interface {
	GetByCode(ctx context.Context, shortCode string) (*domain.LinkInfo, error)
	HasPassword(ctx context.Context, shortCode string) bool
	CheckPassword(ctx context.Context, shortCode, password string) (bool, *domain.LinkInfo, error)
	LogClick(ctx context.Context, linkID int64, ip, userAgent, platform, referer string)
	BuildShortURL(domain, code string) string
}

type Handler struct {
	svc LinkService
}

func NewHandler(svc LinkService) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) RegisterRoutes(r *gin.Engine, mw ...gin.HandlerFunc) {
	rg := r.Group("/r")
	if len(mw) > 0 && mw[0] != nil {
		rg.Use(mw[0])
	}
	rg.GET("/:code", h.Redirect)
	rg.GET("/:code/qrcode", h.QRCode)
	rg.POST("/:code/verify-password", h.VerifyPassword)
	rg.POST("/:code/click-action", h.LogClickAction)
}

// Redirect 短链重定向（含平台检测和密码检查）
func (h *Handler) Redirect(c *gin.Context) {
	code := c.Param("code")

	link, err := h.svc.GetByCode(c.Request.Context(), code)
	if err != nil {
		c.String(http.StatusNotFound, "链接不存在或已过期")
		return
	}

	// 防御存量脏数据：目标 URL 必须为 http/https，防止 javascript: 等协议在引导页执行（XSS）
	if !urlcheck.IsSafeTarget(link.OriginalURL) {
		h.renderUnsafeTargetPage(c)
		return
	}

	// 检查是否需要密码
	if h.svc.HasPassword(c.Request.Context(), code) {
		c.Header("Content-Type", "text/html; charset=utf-8")
		c.String(http.StatusOK, passwordPageHTML(code))
		return
	}

	userAgent := c.GetHeader("User-Agent")
	platform := ua.Detect(userAgent)
	ip := middleware.RealIP(c)
	referer := c.GetHeader("Referer")

	// 记录点击
	clickPlatform := string(platform)
	go h.svc.LogClick(context.Background(), link.ID, ip, userAgent, clickPlatform, referer)

	// 判断是否需要中间引导页
	if ua.NeedsIntermediatePage(platform) {
		h.renderIntermediatePage(c, link.OriginalURL, code, platform)
		return
	}

	// 普通浏览器直接 302 跳转
	c.Redirect(http.StatusFound, link.OriginalURL)
}

// QRCode 为短链生成 QR 码
func (h *Handler) QRCode(c *gin.Context) {
	code := c.Param("code")

	link, err := h.svc.GetByCode(c.Request.Context(), code)
	if err != nil {
		c.String(http.StatusNotFound, "链接不存在或已过期")
		return
	}

	// 使用链接实际域名（此前传空串会生成 "https:///r/CODE" 的无效 URL）
	shortURL := h.svc.BuildShortURL(link.Domain, code)

	png, err := qrcode.Encode(shortURL, qrcode.Medium, 256)
	if err != nil {
		c.String(http.StatusInternalServerError, "QR 码生成失败")
		return
	}

	c.Header("Content-Type", "image/png")
	c.Header("Cache-Control", "public, max-age=86400")
	if _, err := c.Writer.Write(png); err != nil {
		log.Printf("write qrcode png failed: %v", err)
	}
}

// LogClickAction 记录用户在引导页上的行为（复制、扫码、deeplink 尝试等）
func (h *Handler) LogClickAction(c *gin.Context) {
	code := c.Param("code")
	var req struct {
		Action string `json:"action"` // copy_link, qr_scan, deeplink_try, open_browser
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Status(http.StatusBadRequest)
		return
	}

	link, err := h.svc.GetByCode(c.Request.Context(), code)
	if err != nil {
		c.Status(http.StatusNotFound)
		return
	}

	userAgent := c.GetHeader("User-Agent")
	platform := ua.Detect(userAgent)
	ip := middleware.RealIP(c)

	// 记录 action 事件（platform 保持不变，action 存入 referer 字段）
	go h.svc.LogClick(context.Background(), link.ID, ip, userAgent, string(platform), "action:"+req.Action)

	c.Status(http.StatusNoContent)
}

// VerifyPassword 验证密码后跳转
func (h *Handler) VerifyPassword(c *gin.Context) {
	code := c.Param("code")
	password := c.PostForm("password")

	ok, info, err := h.svc.CheckPassword(c.Request.Context(), code, password)
	if err != nil || !ok {
		c.Header("Content-Type", "text/html; charset=utf-8")
		c.String(http.StatusOK, passwordPageHTMLWithError(code, "密码错误"))
		return
	}

	// 防御存量脏数据：目标 URL 必须为 http/https
	if !urlcheck.IsSafeTarget(info.OriginalURL) {
		h.renderUnsafeTargetPage(c)
		return
	}

	// 密码正确，记录点击并跳转
	userAgent := c.GetHeader("User-Agent")
	platform := ua.Detect(userAgent)
	ip := middleware.RealIP(c)
	referer := c.GetHeader("Referer")
	go h.svc.LogClick(context.Background(), info.ID, ip, userAgent, string(platform), referer)

	if ua.NeedsIntermediatePage(platform) {
		h.renderIntermediatePage(c, info.OriginalURL, code, platform)
		return
	}
	c.Redirect(http.StatusFound, info.OriginalURL)
}

// renderIntermediatePage 渲染中间引导页（微信/QQ等）
func (h *Handler) renderIntermediatePage(c *gin.Context, targetURL, code string, platform domain.Platform) {
	platformName := ua.PlatformName(platform)
	platformTips := ua.PlatformTips(platform)
	qrURL := "/r/" + code + "/qrcode"

	data := struct {
		TargetURL    string
		Code         string
		PlatformName string
		Platform     string
		PlatformTips string
		QRURL        string
	}{
		TargetURL:    targetURL,
		Code:         code,
		PlatformName: platformName,
		Platform:     string(platform),
		PlatformTips: platformTips,
		QRURL:        qrURL,
	}

	// 微信/QQ 内置浏览器的 deeplink URL scheme
	deeplinks := ua.GetDeeplinks(targetURL)
	_ = deeplinks // 注入到模板中使用

	c.Header("Content-Type", "text/html; charset=utf-8")
	if _, err := c.Writer.Write([]byte(renderGuidePage(data, deeplinks))); err != nil {
		log.Printf("write guide page failed: %v", err)
	}
}

// renderGuidePage 渲染引导页（含 deeplinks）
func renderGuidePage(data struct {
	TargetURL    string
	Code         string
	PlatformName string
	Platform     string
	PlatformTips string
	QRURL        string
}, deeplinks []domain.DeepLink) string {
	var buf bytes.Buffer
	page := template.Must(template.New("guide").Parse(guidePageHTML))
	if err := page.Execute(&buf, data); err != nil {
		log.Printf("execute guide template failed: %v", err)
	}
	return buf.String()
}

const guidePageHTML = `<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0, maximum-scale=1.0, user-scalable=no">
    <title>即将打开链接 - Kada</title>
    <style>
        :root {
            --primary: #6366f1;
            --primary-dark: #4f46e5;
            --bg: #f8fafc;
            --card: #ffffff;
            --text: #1e293b;
            --text-secondary: #64748b;
            --border: #e2e8f0;
            --success: #22c55e;
            --warning: #f59e0b;
        }
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body {
            font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", "PingFang SC", "Hiragino Sans GB", "Microsoft YaHei", sans-serif;
            background: var(--bg);
            display: flex; justify-content: center; align-items: center;
            min-height: 100vh; padding: 16px;
            -webkit-tap-highlight-color: transparent;
        }
        .container { width: 100%; max-width: 420px; }

        /* 顶部品牌 */
        .brand {
            text-align: center; margin-bottom: 20px;
            font-size: 14px; color: var(--text-secondary);
            display: flex; align-items: center; justify-content: center; gap: 6px;
        }
        .brand-logo {
            width: 28px; height: 28px; background: var(--primary);
            border-radius: 8px; display: flex; align-items: center; justify-content: center;
            color: white; font-weight: 700; font-size: 14px;
        }

        /* 主卡片 */
        .card {
            background: var(--card); border-radius: 20px;
            padding: 32px 24px 24px;
            box-shadow: 0 1px 3px rgba(0,0,0,0.04), 0 8px 24px rgba(0,0,0,0.06);
            text-align: center;
        }

        /* 平台标签 */
        .platform-badge {
            display: inline-flex; align-items: center; gap: 4px;
            padding: 4px 12px; border-radius: 20px;
            font-size: 12px; font-weight: 500;
            background: #eef2ff; color: var(--primary);
            margin-bottom: 20px;
        }

        .icon-area { margin-bottom: 20px; }
        .link-icon {
            width: 64px; height: 64px; border-radius: 16px;
            background: linear-gradient(135deg, #6366f1, #8b5cf6);
            display: inline-flex; align-items: center; justify-content: center;
            font-size: 28px; color: white;
            box-shadow: 0 4px 12px rgba(99,102,241,0.3);
        }

        .title { font-size: 20px; font-weight: 700; color: var(--text); margin-bottom: 6px; }
        .subtitle { font-size: 13px; color: var(--text-secondary); margin-bottom: 8px; line-height: 1.5; }

        /* 目标链接预览 */
        .url-preview {
            display: flex; align-items: center; gap: 8px;
            padding: 10px 14px; background: #f1f5f9;
            border-radius: 10px; margin-bottom: 24px;
            font-size: 12px; color: var(--text-secondary);
            word-break: break-all; text-align: left;
        }
        .url-preview .favicon {
            width: 20px; height: 20px; border-radius: 4px; background: #cbd5e1;
            flex-shrink: 0; display: flex; align-items: center; justify-content: center;
            font-size: 10px;
        }

        /* 操作按钮区 */
        .actions { display: flex; flex-direction: column; gap: 10px; margin-bottom: 20px; }
        .btn {
            display: flex; align-items: center; justify-content: center; gap: 8px;
            width: 100%; padding: 14px 20px; border-radius: 12px;
            font-size: 15px; font-weight: 600; cursor: pointer; border: none;
            transition: all 0.15s ease; text-decoration: none;
            position: relative; overflow: hidden;
        }
        .btn:active { transform: scale(0.98); }
        .btn-primary {
            background: linear-gradient(135deg, #6366f1, #4f46e5);
            color: white; box-shadow: 0 2px 8px rgba(99,102,241,0.25);
        }
        .btn-primary:hover { box-shadow: 0 4px 16px rgba(99,102,241,0.35); }
        .btn-outline {
            background: var(--card); color: var(--text);
            border: 1.5px solid var(--border);
        }
        .btn-outline:hover { background: #f8fafc; border-color: #cbd5e1; }
        .btn-ghost {
            background: transparent; color: var(--text-secondary);
            font-size: 14px; font-weight: 500; padding: 8px;
        }
        .btn-icon { font-size: 18px; }

        /* QR 码区域 */
        .qr-section {
            border-top: 1px solid var(--border);
            padding-top: 20px; margin-top: 4px;
        }
        .qr-toggle {
            font-size: 13px; color: var(--text-secondary);
            background: none; border: none; cursor: pointer;
            display: flex; align-items: center; justify-content: center; gap: 4px;
            width: 100%; padding: 8px;
        }
        .qr-container {
            display: none; margin-top: 12px;
        }
        .qr-container.show { display: block; }
        .qr-img {
            width: 180px; height: 180px; border-radius: 12px;
            border: 1px solid var(--border); padding: 8px; background: white;
        }
        .qr-hint {
            font-size: 12px; color: var(--text-secondary); margin-top: 8px;
        }

        /* Toast */
        .toast {
            position: fixed; top: 24px; left: 50%; transform: translateX(-50%);
            background: #1e293b; color: white; padding: 10px 24px;
            border-radius: 10px; font-size: 14px; font-weight: 500;
            opacity: 0; transition: opacity 0.3s; z-index: 999;
            pointer-events: none; white-space: nowrap;
            box-shadow: 0 4px 16px rgba(0,0,0,0.15);
        }
        .toast.show { opacity: 1; }

        /* 安全提示 */
        .safety-note {
            font-size: 11px; color: #94a3b8; margin-top: 16px;
            display: flex; align-items: center; justify-content: center; gap: 4px;
        }

        /* 平台提示卡片 */
        .platform-tips {
            background: #fffbeb; border: 1px solid #fde68a;
            border-radius: 10px; padding: 12px 14px; margin-bottom: 16px;
            font-size: 12px; color: #92400e; text-align: left;
            display: flex; gap: 8px; align-items: flex-start;
        }
        .platform-tips .tip-icon { font-size: 16px; flex-shrink: 0; }
    </style>
</head>
<body>
    <div class="container">
        <div class="brand">
            <div class="brand-logo">K</div>
            Kada 短链
        </div>

        <div class="card">
            <div class="platform-badge">
                📱 {{.PlatformName}} 内访问
            </div>
            <div class="icon-area">
                <div class="link-icon">🔗</div>
            </div>
            <div class="title">即将打开链接</div>
            <div class="subtitle">{{.PlatformTips}}</div>

            <div class="url-preview">
                <div class="favicon">🌐</div>
                <span>{{.TargetURL}}</span>
            </div>

            {{if or (eq .Platform "wechat") (eq .Platform "qq")}}
            <div class="platform-tips">
                <span class="tip-icon">💡</span>
                <span>{{.PlatformName}}内置浏览器可能限制直接跳转。如无法打开，请点击右上角「<strong>在浏览器中打开</strong>」或扫描下方二维码。</span>
            </div>
            {{end}}

            <div class="actions">
                <button class="btn btn-primary" onclick="tryOpenLink()">
                    <span class="btn-icon">🚀</span> 打开链接
                </button>
                <button class="btn btn-outline" onclick="tryDeeplink()">
                    <span class="btn-icon">📲</span> 在浏览器中打开
                </button>
                <button class="btn btn-outline" onclick="copyLink()">
                    <span class="btn-icon">📋</span> 复制链接
                </button>
            </div>

            <!-- QR 码折叠区 -->
            <div class="qr-section">
                <button class="qr-toggle" onclick="toggleQR()" id="qrToggleBtn">
                    <span>📱</span> 扫码打开 <span style="font-size:10px">▼</span>
                </button>
                <div class="qr-container" id="qrContainer">
                    <img class="qr-img" src="{{.QRURL}}" alt="扫码打开链接" />
                    <p class="qr-hint">使用手机相机或微信扫一扫打开</p>
                </div>
            </div>
        </div>

        <p class="safety-note">🔒 由 Kada 短链服务提供</p>
    </div>

    <!-- Toast -->
    <div class="toast" id="toast"></div>

    <script>
        const targetURL = "{{.TargetURL}}";
        const code = "{{.Code}}";

        // === DeepLink 尝试 ===
        function tryDeeplink() {
            logAction('open_browser');

            // 尝试通过 intent / URL scheme 唤起外部浏览器
            var schemes = [
                'intent://' + encodeURIComponent(targetURL.replace(/^https?:\/\//, '')) + '#Intent;scheme=https;package=com.android.chrome;end',
                'googlechrome://navigate?url=' + encodeURIComponent(targetURL),
            ];

            var opened = false;
            var startTime = Date.now();

            // 尝试第一个 scheme
            try {
                var iframe = document.createElement('iframe');
                iframe.style.display = 'none';
                iframe.src = schemes[0];
                document.body.appendChild(iframe);
                setTimeout(function() { document.body.removeChild(iframe); }, 2000);
            } catch(e) {}

            // 超时后如果还在本页，说明未唤起成功，直接 302 跳转
            setTimeout(function() {
                if (Date.now() - startTime > 2500) return;
                window.location.href = targetURL;
            }, 800);

            showToast('正在尝试打开浏览器...');
        }

        // === 直接打开链接 ===
        function tryOpenLink() {
            logAction('open_link');
            // 在微信/QQ中尝试用系统浏览器打开
            var ua = navigator.userAgent.toLowerCase();
            if (ua.indexOf('micromessenger') > -1 || ua.indexOf('qq/') > -1) {
                // 先尝试 deeplink 方式
                tryDeeplink();
                return;
            }
            window.location.href = targetURL;
        }

        // === 复制链接 ===
        function copyLink() {
            logAction('copy_link');
            var url = targetURL;
            if (navigator.clipboard && navigator.clipboard.writeText) {
                navigator.clipboard.writeText(url).then(function() {
                    showToast('✅ 链接已复制，请在浏览器中打开');
                }).catch(function() {
                    fallbackCopy(url);
                });
            } else {
                fallbackCopy(url);
            }
        }

        function fallbackCopy(text) {
            var ta = document.createElement('textarea');
            ta.value = text;
            ta.style.position = 'fixed'; ta.style.left = '-9999px';
            document.body.appendChild(ta);
            ta.select();
            try { document.execCommand('copy'); showToast('✅ 链接已复制'); } catch(e) { showToast('复制失败，请手动复制'); }
            document.body.removeChild(ta);
        }

        // === QR 码折叠 ===
        function toggleQR() {
            var container = document.getElementById('qrContainer');
            var btn = document.getElementById('qrToggleBtn');
            var isOpen = container.classList.contains('show');
            if (isOpen) {
                container.classList.remove('show');
                btn.querySelector('span:last-child').textContent = '▼';
            } else {
                container.classList.add('show');
                btn.querySelector('span:last-child').textContent = '▲';
                logAction('qr_view');
            }
        }

        // === Toast ===
        function showToast(msg) {
            var toast = document.getElementById('toast');
            toast.textContent = msg;
            toast.classList.add('show');
            clearTimeout(toast._timeout);
            toast._timeout = setTimeout(function() {
                toast.classList.remove('show');
            }, 2000);
        }

        // === 记录用户行为 ===
        function logAction(action) {
            fetch('/r/' + code + '/click-action', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ action: action }),
                keepalive: true
            }).catch(function(){});
        }

        // 页面加载后自动尝试跳转
        setTimeout(function() {
            tryOpenLink();
        }, 2000);
    </script>
</body>
</html>`

// renderUnsafeTargetPage 目标 URL 协议不受支持时渲染提示页
func (h *Handler) renderUnsafeTargetPage(c *gin.Context) {
	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(http.StatusBadRequest, `<!DOCTYPE html>
<html lang="zh-CN">
<head><meta charset="UTF-8"><meta name="viewport" content="width=device-width, initial-scale=1.0"><title>链接不可用 - Kada</title>
<style>body{font-family:-apple-system,"PingFang SC","Microsoft YaHei",sans-serif;background:#f8fafc;display:flex;justify-content:center;align-items:center;min-height:100vh;margin:0}.card{background:#fff;border-radius:20px;padding:40px 32px;max-width:400px;width:100%;text-align:center;box-shadow:0 1px 3px rgba(0,0,0,.04),0 8px 24px rgba(0,0,0,.06)}.icon{font-size:48px;margin-bottom:16px}.title{font-size:20px;font-weight:700;color:#1e293b;margin-bottom:8px}.subtitle{font-size:14px;color:#64748b}</style>
</head>
<body><div class="card"><div class="icon">⚠️</div><div class="title">链接不可用</div><div class="subtitle">该短链的目标地址协议不受支持，已阻止跳转</div></div></body>
</html>`)
}

func passwordPageHTML(code string) string {
	return passwordPageHTMLWithError(code, "")
}

func passwordPageHTMLWithError(code, errorMsg string) string {
	errHTML := ""
	if errorMsg != "" {
		errHTML = `<p style="color: #ef4444; font-size: 13px; margin-bottom: 12px;">` + errorMsg + `</p>`
	}
	return `<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0, maximum-scale=1.0, user-scalable=no">
    <title>链接已加密 - Kada</title>
    <style>
        :root {
            --primary: #6366f1;
            --bg: #f8fafc;
            --card: #ffffff;
            --text: #1e293b;
            --text-secondary: #64748b;
            --border: #e2e8eb;
        }
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body {
            font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", "PingFang SC", "Hiragino Sans GB", "Microsoft YaHei", sans-serif;
            background: var(--bg);
            display: flex; justify-content: center; align-items: center;
            min-height: 100vh; padding: 16px;
        }
        .card {
            background: var(--card); border-radius: 20px; padding: 40px 32px;
            max-width: 400px; width: 100%; text-align: center;
            box-shadow: 0 1px 3px rgba(0,0,0,0.04), 0 8px 24px rgba(0,0,0,0.06);
        }
        .icon { font-size: 48px; margin-bottom: 16px; }
        .title { font-size: 20px; font-weight: 700; color: var(--text); margin-bottom: 8px; }
        .subtitle { font-size: 14px; color: var(--text-secondary); margin-bottom: 20px; }
        .input {
            width: 100%; padding: 14px 16px; border: 1.5px solid var(--border);
            border-radius: 12px; font-size: 16px; text-align: center;
            margin-bottom: 12px; outline: none; transition: border-color 0.15s;
        }
        .input:focus { border-color: var(--primary); box-shadow: 0 0 0 3px rgba(99,102,241,0.1); }
        .btn {
            display: block; width: 100%; padding: 14px; border-radius: 12px;
            font-size: 16px; font-weight: 600; cursor: pointer; border: none;
            background: linear-gradient(135deg, #6366f1, #4f46e5); color: white;
            transition: all 0.15s; box-shadow: 0 2px 8px rgba(99,102,241,0.25);
        }
        .btn:active { transform: scale(0.98); }
        .btn:hover { box-shadow: 0 4px 16px rgba(99,102,241,0.35); }
    </style>
</head>
<body>
    <div class="card">
        <div class="icon">🔒</div>
        <div class="title">此链接已加密</div>
        <div class="subtitle">请输入密码以访问此链接</div>
        ` + errHTML + `
        <form method="POST" action="/r/` + code + `/verify-password">
            <input type="password" name="password" class="input" placeholder="输入密码" autofocus required />
            <button type="submit" class="btn">访问链接</button>
        </form>
    </div>
</body>
</html>`
}
