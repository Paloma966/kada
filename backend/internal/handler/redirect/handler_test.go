package redirect

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/chun/kada-backend/internal/domain"
)

// mockLinkService is used for redirect handler tests.
type mockLinkService struct {
	getByCode     func(ctx context.Context, code string) (*domain.LinkInfo, error)
	checkPassword func(ctx context.Context, code, password string) (bool, *domain.LinkInfo, error)
	logClick      func(ctx context.Context, event domain.ClickEvent)
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

func (m *mockLinkService) CheckPassword(ctx context.Context, code, password string) (bool, *domain.LinkInfo, error) {
	if m.checkPassword != nil {
		return m.checkPassword(ctx, code, password)
	}
	return true, &domain.LinkInfo{OriginalURL: "https://example.com/target"}, nil
}

func (m *mockLinkService) LogClick(ctx context.Context, event domain.ClickEvent) {
	if m.logClick != nil {
		m.logClick(ctx, event)
	}
}

func (m *mockLinkService) BuildShortURL(domain, code string) string {
	if m.buildShortURL != nil {
		return m.buildShortURL(domain, code)
	}
	return "https://" + domain + "/r/" + code
}

// The User-Agents these tests use. The WeChat one is the interesting case: it is what the in-app browser
// sends, and it is also what WeChat's preview fetcher sends, which is why nothing here can tell them apart
// and the guide page has to.
const (
	wechatUA    = "Mozilla/5.0 (iPhone; CPU iPhone OS 16_0 like Mac OS X) AppleWebKit/605.1.15 MicroMessenger/8.0.40"
	browserUA   = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 Chrome/120.0.0.0 Safari/537.36"
	googlebotUA = "Mozilla/5.0 (compatible; Googlebot/2.1; +http://www.google.com/bot.html)"
)

// clickRecorder collects the click events the handler hands to the service.
//
// They arrive on a channel rather than in a slice because the handler logs them from a goroutine - nothing
// about recording a click may sit between the visitor and the redirect - so the channel is what
// synchronizes the test with that goroutine instead of racing it.
type clickRecorder struct {
	events chan domain.ClickEvent
}

func newClickRecorder() *clickRecorder {
	return &clickRecorder{events: make(chan domain.ClickEvent, 16)}
}

func (r *clickRecorder) record(_ context.Context, event domain.ClickEvent) {
	r.events <- event
}

// collect reads exactly want events, failing instead of hanging when a request that should have been
// recorded was not.
func (r *clickRecorder) collect(t *testing.T, want int) []domain.ClickEvent {
	t.Helper()

	events := make([]domain.ClickEvent, 0, want)
	for len(events) < want {
		select {
		case e := <-r.events:
			events = append(events, e)
		case <-time.After(2 * time.Second):
			t.Fatalf("recorded %d of %d click events: %+v", len(events), want, events)
		}
	}
	return events
}

// counted mirrors what ClickStore does with the column: only a visit moves a counter. Asserting on this
// rather than on the labels is what makes these the tests for the numbers.
func counted(events []domain.ClickEvent) int {
	n := 0
	for _, e := range events {
		if e.Kind == domain.ClickVisit {
			n++
		}
	}
	return n
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

// ========== Unsafe scheme blocking ==========

func TestRedirect_UnsafeSchemeBlocked(t *testing.T) {
	gin.SetMode(gin.TestMode)

	svc := &mockLinkService{
		getByCode: func(ctx context.Context, code string) (*domain.LinkInfo, error) {
			return &domain.LinkInfo{
				ID:          1,
				ShortCode:   code,
				OriginalURL: "javascript:alert(document.cookie)",
				Domain:      "kada.click",
				IsActive:    true,
			}, nil
		},
	}
	h := NewHandler(svc)

	r := gin.New()
	r.GET("/r/:code", h.Redirect)

	// The WeChat UA goes through the guide page, which has the largest attack surface.
	req := httptest.NewRequest("GET", "/r/evil123", nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 MicroMessenger/8.0")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for javascript: target, got %d", w.Code)
	}
	if body := w.Body.String(); !strings.Contains(body, "not supported") && !strings.Contains(body, "Link unavailable") {
		t.Errorf("expected blocked page, got: %s", body)
	}
}

func TestVerifyPassword_UnsafeSchemeBlocked(t *testing.T) {
	gin.SetMode(gin.TestMode)

	svc := &mockLinkService{
		checkPassword: func(ctx context.Context, code, password string) (bool, *domain.LinkInfo, error) {
			return true, &domain.LinkInfo{OriginalURL: "data:text/html,<script>alert(1)</script>"}, nil
		},
	}
	h := NewHandler(svc)

	r := gin.New()
	r.POST("/r/:code/verify-password", h.VerifyPassword)

	req := httptest.NewRequest("POST", "/r/evil123/verify-password", strings.NewReader("password=x"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for data: target, got %d", w.Code)
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

// ========== What counts as a visit ==========

func newRedirectServer(t *testing.T, rec *clickRecorder) *gin.Engine {
	t.Helper()

	gin.SetMode(gin.TestMode)
	h := NewHandler(&mockLinkService{logClick: rec.record})
	r := gin.New()
	h.RegisterRoutes(r)
	return r
}

// do sends one request with a User-Agent and returns the response.
func do(r *gin.Engine, method, path, userAgent string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, nil)
	if userAgent != "" {
		req.Header.Set("User-Agent", userAgent)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// A prefetch and a visit are the same request with the same User-Agent: WeChat fetches the link to build a
// preview card with the in-app browser's own User-Agent, so the request alone cannot say which it is.
// What it can do is record the request - which is what keeps the amount of prefetch measurable - and
// refuse to count it.
func TestPrefetchIsRecordedButNotCounted(t *testing.T) {
	rec := newClickRecorder()
	r := newRedirectServer(t, rec)

	// One request, and nothing else: a crawler reads the HTML and stops there.
	if w := do(r, "GET", "/r/abc123", wechatUA); w.Code != http.StatusOK {
		t.Fatalf("expected the guide page, got %d", w.Code)
	}

	events := rec.collect(t, 1)
	if events[0].Kind != domain.ClickRequest {
		t.Errorf("the prefetch was recorded as %q, want %q", events[0].Kind, domain.ClickRequest)
	}
	if n := counted(events); n != 0 {
		t.Errorf("a prefetch moved %d counters", n)
	}
}

// The same request, followed by the page's own confirmation: a prefetcher does not run JavaScript, a person
// with a browser does.
func TestARequestFollowedByAConfirmationIsCountedOnce(t *testing.T) {
	rec := newClickRecorder()
	r := newRedirectServer(t, rec)

	do(r, "GET", "/r/abc123", wechatUA)
	if w := do(r, "POST", "/r/abc123/visit", wechatUA); w.Code != http.StatusNoContent {
		t.Fatalf("expected 204 from the confirmation, got %d", w.Code)
	}

	events := rec.collect(t, 2)
	if n := counted(events); n != 1 {
		t.Errorf("the confirmed visit moved %d counters, want exactly 1: %+v", n, events)
	}
}

// A client that is sent straight to the target never runs anything of ours, so its request is the only
// evidence that will ever exist. Dropping it would erase most of the traffic a normal browser produces.
func TestDirectBrowserRequestIsCounted(t *testing.T) {
	rec := newClickRecorder()
	r := newRedirectServer(t, rec)

	if w := do(r, "GET", "/r/abc123", browserUA); w.Code != http.StatusFound {
		t.Fatalf("expected 302 for a browser, got %d", w.Code)
	}

	events := rec.collect(t, 1)
	if events[0].Kind != domain.ClickVisit {
		t.Errorf("a browser request was recorded as %q, want %q", events[0].Kind, domain.ClickVisit)
	}
}

// The crawler list only holds names a browser never sends, so being on it is not a guess. Search engines
// are the traffic a link shortener cannot afford to count as readers: Googlebot follows every link it is
// given, whether or not anybody ever clicks it.
func TestCrawlerRequestIsRecordedButNotCounted(t *testing.T) {
	rec := newClickRecorder()
	r := newRedirectServer(t, rec)

	if w := do(r, "GET", "/r/abc123", googlebotUA); w.Code != http.StatusFound {
		t.Fatalf("expected the crawler to be redirected like anyone else, got %d", w.Code)
	}

	events := rec.collect(t, 1)
	if events[0].Kind != domain.ClickRequest {
		t.Errorf("a crawler request was recorded as %q, want %q", events[0].Kind, domain.ClickRequest)
	}
	if n := counted(events); n != 0 {
		t.Errorf("a crawler moved %d counters", n)
	}
}

// The guide page fires its confirmation when it loads, so every later tap is an interaction with a visit
// that is already counted. Counting them would make one person who copies the link and then opens it into
// two visitors.
func TestGuidePageActionsAreRecordedButNotCounted(t *testing.T) {
	rec := newClickRecorder()
	r := newRedirectServer(t, rec)

	req := httptest.NewRequest("POST", "/r/abc123/click-action", strings.NewReader(`{"action":"copy_link"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", wechatUA)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", w.Code)
	}

	events := rec.collect(t, 1)
	if events[0].Kind != domain.ClickAction {
		t.Errorf("a guide page action was recorded as %q, want %q", events[0].Kind, domain.ClickAction)
	}
	if n := counted(events); n != 0 {
		t.Errorf("an action moved %d counters", n)
	}
	if events[0].Referer == "" {
		t.Error("the action name is no longer recorded anywhere")
	}
}

// The password form is the other way into a redirect, and it follows the same rule: typing a password is
// evidence of a person, but not of which client the answer was sent to.
func TestVerifyPasswordCountsOnlyWhenTheTargetIsSent(t *testing.T) {
	tests := []struct {
		name      string
		userAgent string
		want      int
	}{
		{"browser gets the target", browserUA, 1},
		{"wechat gets the guide page", wechatUA, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := newClickRecorder()
			r := newRedirectServer(t, rec)

			req := httptest.NewRequest("POST", "/r/abc123/verify-password", strings.NewReader("password=x"))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			req.Header.Set("User-Agent", tt.userAgent)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if w.Code != http.StatusOK && w.Code != http.StatusFound {
				t.Fatalf("expected a redirect or the guide page, got %d", w.Code)
			}

			if n := counted(rec.collect(t, 1)); n != tt.want {
				t.Errorf("the visit was counted %d times, want %d", n, tt.want)
			}
		})
	}
}

// ========== Password Page ==========

func TestRedirect_PasswordPage(t *testing.T) {
	gin.SetMode(gin.TestMode)

	svc := &mockLinkService{
		getByCode: func(ctx context.Context, code string) (*domain.LinkInfo, error) {
			return &domain.LinkInfo{
				ID:          1,
				ShortCode:   code,
				OriginalURL: "https://example.com/target",
				Domain:      "kada.click",
				IsActive:    true,
				HasPassword: true,
			}, nil
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
