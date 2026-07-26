package redirect

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/chun/kada-backend/internal/domain"
)

// mockLinkService 用于 redirect handler 测试
type mockLinkService struct {
	getByCode     func(ctx context.Context, code string) (*domain.LinkInfo, error)
	hasPassword   func(ctx context.Context, code string) bool
	checkPassword func(ctx context.Context, code, password string) (bool, *domain.LinkInfo, error)
	logClick      func(ctx context.Context, linkID int64, ip, userAgent, platform, referer string)
	buildShortURL func(domain, code string) string
}

func (m *mockLinkService) GetByCode(ctx context.Context, code string) (*domain.LinkInfo, error) {
	if m.getByCode != nil {
		return m.getByCode(ctx, code)
	}
	return &domain.LinkInfo{
		ID:          1,
		ShortCode:   code,
		ShortURL:    "https://kada.click/r/" + code,
		OriginalURL: "https://example.com/target",
		Domain:      "kada.click",
		ClickCount:  0,
		IsActive:    true,
	}, nil
}

func (m *mockLinkService) HasPassword(ctx context.Context, code string) bool {
	if m.hasPassword != nil {
		return m.hasPassword(ctx, code)
	}
	return false
}

func (m *mockLinkService) CheckPassword(ctx context.Context, code, password string) (bool, *domain.LinkInfo, error) {
	if m.checkPassword != nil {
		return m.checkPassword(ctx, code, password)
	}
	return true, &domain.LinkInfo{OriginalURL: "https://example.com/target"}, nil
}

func (m *mockLinkService) LogClick(ctx context.Context, linkID int64, ip, userAgent, platform, referer string) {
	if m.logClick != nil {
		m.logClick(ctx, linkID, ip, userAgent, platform, referer)
	}
}

func (m *mockLinkService) BuildShortURL(domain, code string) string {
	if m.buildShortURL != nil {
		return m.buildShortURL(domain, code)
	}
	return "https://" + domain + "/r/" + code
}

// ========== Redirect ==========

func TestRedirect_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)

	svc := &mockLinkService{
		getByCode: func(ctx context.Context, code string) (*domain.LinkInfo, error) {
			return nil, domain.ErrLinkNotFound
		},
	}
	h := NewHandler(svc)

	r := gin.New()
	r.GET("/r/:code", h.Redirect)

	req := httptest.NewRequest("GET", "/r/nonexistent", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestRedirect_Browser(t *testing.T) {
	gin.SetMode(gin.TestMode)

	h := NewHandler(&mockLinkService{})

	r := gin.New()
	r.GET("/r/:code", h.Redirect)

	req := httptest.NewRequest("GET", "/r/abc123", nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 Chrome/120")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusFound {
		t.Errorf("expected 302 for browser, got %d: %s", w.Code, w.Body.String())
	}
}

func TestRedirect_WechatReturnsGuidePage(t *testing.T) {
	gin.SetMode(gin.TestMode)

	h := NewHandler(&mockLinkService{})

	r := gin.New()
	r.GET("/r/:code", h.Redirect)

	req := httptest.NewRequest("GET", "/r/abc123", nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 MicroMessenger/8.0")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for wechat guide page, got %d", w.Code)
	}

	body := w.Body.String()
	if len(body) < 100 {
		t.Error("guide page HTML seems too short")
	}
}

func TestRedirect_QQReturnsGuidePage(t *testing.T) {
	gin.SetMode(gin.TestMode)

	h := NewHandler(&mockLinkService{})

	r := gin.New()
	r.GET("/r/:code", h.Redirect)

	req := httptest.NewRequest("GET", "/r/abc123", nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 QQ/9.0")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for QQ guide page, got %d", w.Code)
	}
}

// ========== QR Code ==========

func TestQRCode_Valid(t *testing.T) {
	gin.SetMode(gin.TestMode)

	h := NewHandler(&mockLinkService{})

	r := gin.New()
	r.GET("/r/:code/qrcode", h.QRCode)

	req := httptest.NewRequest("GET", "/r/abc123/qrcode", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if w.Header().Get("Content-Type") != "image/png" {
		t.Errorf("expected image/png, got %s", w.Header().Get("Content-Type"))
	}
	if len(w.Body.Bytes()) < 100 {
		t.Error("QR code PNG too small")
	}
}

func TestQRCode_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)

	svc := &mockLinkService{
		getByCode: func(ctx context.Context, code string) (*domain.LinkInfo, error) {
			return nil, domain.ErrLinkNotFound
		},
	}
	h := NewHandler(svc)

	r := gin.New()
	r.GET("/r/:code/qrcode", h.QRCode)

	req := httptest.NewRequest("GET", "/r/nonexistent/qrcode", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

// ========== Password Page ==========

func TestRedirect_PasswordPage(t *testing.T) {
	gin.SetMode(gin.TestMode)

	svc := &mockLinkService{
		hasPassword: func(ctx context.Context, code string) bool {
			return true
		},
	}
	h := NewHandler(svc)

	r := gin.New()
	r.GET("/r/:code", h.Redirect)

	req := httptest.NewRequest("GET", "/r/secret123", nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 Chrome/120")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for password page, got %d", w.Code)
	}
}
