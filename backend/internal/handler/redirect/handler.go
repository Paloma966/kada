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

// LinkService is the short link service interface (mockable for tests).
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

// Redirect performs the short-link redirect (with platform detection and password check).
func (h *Handler) Redirect(c *gin.Context) {
	code := c.Param("code")

	link, err := h.svc.GetByCode(c.Request.Context(), code)
	if err != nil {
		c.String(http.StatusNotFound, "link not found or expired")
		return
	}

	// Guard against legacy dirty data: the target URL must be http/https so that schemes such as javascript: cannot execute on the guide page (XSS).
	if !urlcheck.IsSafeTarget(link.OriginalURL) {
		h.renderUnsafeTargetPage(c)
		return
	}

	// Check whether a password is required.
	if h.svc.HasPassword(c.Request.Context(), code) {
		c.Header("Content-Type", "text/html; charset=utf-8")
		c.String(http.StatusOK, passwordPageHTML(code))
		return
	}

	userAgent := c.GetHeader("User-Agent")
	platform := ua.Detect(userAgent)
	ip := middleware.RealIP(c)
	referer := c.GetHeader("Referer")

	// Log the click.
	clickPlatform := string(platform)
	go h.svc.LogClick(context.Background(), link.ID, ip, userAgent, clickPlatform, referer)

	// Check whether an intermediate guide page is needed.
	if ua.NeedsIntermediatePage(platform) {
		h.renderIntermediatePage(c, link.OriginalURL, code, platform)
		return
	}

	// Regular browsers get a direct 302 redirect.
	c.Redirect(http.StatusFound, link.OriginalURL)
}

// QRCode generates a QR code for the short link.
func (h *Handler) QRCode(c *gin.Context) {
	code := c.Param("code")

	link, err := h.svc.GetByCode(c.Request.Context(), code)
	if err != nil {
		c.String(http.StatusNotFound, "link not found or expired")
		return
	}

	// Use the link's actual domain (passing an empty string previously produced the invalid URL "https:///r/CODE").
	shortURL := h.svc.BuildShortURL(link.Domain, code)

	png, err := qrcode.Encode(shortURL, qrcode.Medium, 256)
	if err != nil {
		c.String(http.StatusInternalServerError, "failed to generate QR code")
		return
	}

	c.Header("Content-Type", "image/png")
	c.Header("Cache-Control", "public, max-age=86400")
	if _, err := c.Writer.Write(png); err != nil {
		log.Printf("write qrcode png failed: %v", err)
	}
}

// LogClickAction records user actions on the guide page (copy, QR scan, deeplink attempt, etc.).
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

	// Log the action event (platform is unchanged; the action is stored in the referer field).
	go h.svc.LogClick(context.Background(), link.ID, ip, userAgent, string(platform), "action:"+req.Action)

	c.Status(http.StatusNoContent)
}

// VerifyPassword verifies the password and then redirects.
func (h *Handler) VerifyPassword(c *gin.Context) {
	code := c.Param("code")
	password := c.PostForm("password")

	ok, info, err := h.svc.CheckPassword(c.Request.Context(), code, password)
	if err != nil || !ok {
		c.Header("Content-Type", "text/html; charset=utf-8")
		c.String(http.StatusOK, passwordPageHTMLWithError(code, "incorrect password"))
		return
	}

	// Guard against legacy dirty data: the target URL must be http/https.
	if !urlcheck.IsSafeTarget(info.OriginalURL) {
		h.renderUnsafeTargetPage(c)
		return
	}

	// Password is correct: log the click and redirect.
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

// renderIntermediatePage renders the intermediate guide page (WeChat/QQ, etc.).
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

	// Deeplink URL schemes for the WeChat/QQ in-app browsers.
	deeplinks := ua.GetDeeplinks(targetURL)
	_ = deeplinks // injected into the template

	c.Header("Content-Type", "text/html; charset=utf-8")
	if _, err := c.Writer.Write([]byte(renderGuidePage(data, deeplinks))); err != nil {
		log.Printf("write guide page failed: %v", err)
	}
}

// renderGuidePage renders the guide page (with deeplinks).
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
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0, maximum-scale=1.0, user-scalable=no">
    <title>Opening link - Kada</title>
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

        /* Brand header */
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

        /* Main card */
        .card {
            background: var(--card); border-radius: 20px;
            padding: 32px 24px 24px;
            box-shadow: 0 1px 3px rgba(0,0,0,0.04), 0 8px 24px rgba(0,0,0,0.06);
            text-align: center;
        }

        /* Platform badge */
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

        /* Target link preview */
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

        /* Action buttons */
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

        /* QR code section */
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

        /* Safety note */
        .safety-note {
            font-size: 11px; color: #94a3b8; margin-top: 16px;
            display: flex; align-items: center; justify-content: center; gap: 4px;
        }

        /* Platform tips card */
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
            Kada Short Link
        </div>

        <div class="card">
            <div class="platform-badge">
                📱 Opening in {{.PlatformName}}
            </div>
            <div class="icon-area">
                <div class="link-icon">🔗</div>
            </div>
            <div class="title">Opening link</div>
            <div class="subtitle">{{.PlatformTips}}</div>

            <div class="url-preview">
                <div class="favicon">🌐</div>
                <span>{{.TargetURL}}</span>
            </div>

            {{if or (eq .Platform "wechat") (eq .Platform "qq")}}
            <div class="platform-tips">
                <span class="tip-icon">💡</span>
                <span>The {{.PlatformName}} in-app browser may block direct navigation. If the link does not open, tap "<strong>Open in browser</strong>" in the top-right corner or scan the QR code below.</span>
            </div>
            {{end}}

            <div class="actions">
                <button class="btn btn-primary" onclick="tryOpenLink()">
                    <span class="btn-icon">🚀</span> Open link
                </button>
                <button class="btn btn-outline" onclick="tryDeeplink()">
                    <span class="btn-icon">📲</span> Open in browser
                </button>
                <button class="btn btn-outline" onclick="copyLink()">
                    <span class="btn-icon">📋</span> Copy link
                </button>
            </div>

            <!-- QR code collapsible section -->
            <div class="qr-section">
                <button class="qr-toggle" onclick="toggleQR()" id="qrToggleBtn">
                    <span>📱</span> Scan to open <span style="font-size:10px">▼</span>
                </button>
                <div class="qr-container" id="qrContainer">
                    <img class="qr-img" src="{{.QRURL}}" alt="Scan to open link" />
                    <p class="qr-hint">Scan with your phone camera or WeChat</p>
                </div>
            </div>
        </div>

        <p class="safety-note">🔒 Provided by Kada Short Link</p>
    </div>

    <!-- Toast -->
    <div class="toast" id="toast"></div>

    <script>
        const targetURL = "{{.TargetURL}}";
        const code = "{{.Code}}";

        // === DeepLink attempt ===
        function tryDeeplink() {
            logAction('open_browser');

            // Try to launch an external browser via intent / URL scheme
            var schemes = [
                'intent://' + encodeURIComponent(targetURL.replace(/^https?:\/\//, '')) + '#Intent;scheme=https;package=com.android.chrome;end',
                'googlechrome://navigate?url=' + encodeURIComponent(targetURL),
            ];

            var opened = false;
            var startTime = Date.now();

            // Try the first scheme
            try {
                var iframe = document.createElement('iframe');
                iframe.style.display = 'none';
                iframe.src = schemes[0];
                document.body.appendChild(iframe);
                setTimeout(function() { document.body.removeChild(iframe); }, 2000);
            } catch(e) {}

            // If we are still on this page after the timeout, the launch failed; fall back to a direct redirect
            setTimeout(function() {
                if (Date.now() - startTime > 2500) return;
                window.location.href = targetURL;
            }, 800);

            showToast('Trying to open browser...');
        }

        // === Open the link directly ===
        function tryOpenLink() {
            logAction('open_link');
            // In WeChat/QQ, try to open with the system browser
            var ua = navigator.userAgent.toLowerCase();
            if (ua.indexOf('micromessenger') > -1 || ua.indexOf('qq/') > -1) {
                // Try the deeplink approach first
                tryDeeplink();
                return;
            }
            window.location.href = targetURL;
        }

        // === Copy link ===
        function copyLink() {
            logAction('copy_link');
            var url = targetURL;
            if (navigator.clipboard && navigator.clipboard.writeText) {
                navigator.clipboard.writeText(url).then(function() {
                    showToast('✅ Link copied, please open it in your browser');
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
            try { document.execCommand('copy'); showToast('✅ Link copied'); } catch(e) { showToast('Copy failed, please copy manually'); }
            document.body.removeChild(ta);
        }

        // === QR code toggle ===
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

        // === Log user actions ===
        function logAction(action) {
            fetch('/r/' + code + '/click-action', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ action: action }),
                keepalive: true
            }).catch(function(){});
        }

        // Automatically try to open the link after the page loads
        setTimeout(function() {
            tryOpenLink();
        }, 2000);
    </script>
</body>
</html>`

// renderUnsafeTargetPage renders a notice page when the target URL scheme is not supported.
func (h *Handler) renderUnsafeTargetPage(c *gin.Context) {
	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(http.StatusBadRequest, `<!DOCTYPE html>
<html lang="en">
<head><meta charset="UTF-8"><meta name="viewport" content="width=device-width, initial-scale=1.0"><title>Link unavailable - Kada</title>
<style>body{font-family:-apple-system,"PingFang SC","Microsoft YaHei",sans-serif;background:#f8fafc;display:flex;justify-content:center;align-items:center;min-height:100vh;margin:0}.card{background:#fff;border-radius:20px;padding:40px 32px;max-width:400px;width:100%;text-align:center;box-shadow:0 1px 3px rgba(0,0,0,.04),0 8px 24px rgba(0,0,0,.06)}.icon{font-size:48px;margin-bottom:16px}.title{font-size:20px;font-weight:700;color:#1e293b;margin-bottom:8px}.subtitle{font-size:14px;color:#64748b}</style>
</head>
<body><div class="card"><div class="icon">⚠️</div><div class="title">Link unavailable</div><div class="subtitle">The target URL scheme of this short link is not supported, so the redirect was blocked</div></div></body>
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
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0, maximum-scale=1.0, user-scalable=no">
    <title>Link protected - Kada</title>
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
        <div class="title">This link is protected</div>
        <div class="subtitle">Enter the password to access this link</div>
        ` + errHTML + `
        <form method="POST" action="/r/` + code + `/verify-password">
            <input type="password" name="password" class="input" placeholder="Enter password" autofocus required />
            <button type="submit" class="btn">Access link</button>
        </form>
    </div>
</body>
</html>`
}
