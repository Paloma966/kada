package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/chun/kada-backend/internal/domain"
)

// mockAuthService 用于 handler 测试
type mockAuthService struct {
	sendSMSCode     func(ctx context.Context, phone string) error
	loginByPhone    func(ctx context.Context, phone, code string) (*domain.AuthResponse, error)
	loginByEmail    func(ctx context.Context, email, password string) (*domain.AuthResponse, error)
	registerByEmail func(ctx context.Context, email, password, name string) (*domain.AuthResponse, error)
	getUserByID     func(ctx context.Context, userID int64) (*domain.UserInfo, error)
	updateUser      func(ctx context.Context, userID int64, name *string, email *string) (*domain.UserInfo, error)
}

func (m *mockAuthService) SendSMSCode(ctx context.Context, phone string) error {
	if m.sendSMSCode != nil {
		return m.sendSMSCode(ctx, phone)
	}
	return nil
}

func (m *mockAuthService) LoginByPhone(ctx context.Context, phone, code string) (*domain.AuthResponse, error) {
	if m.loginByPhone != nil {
		return m.loginByPhone(ctx, phone, code)
	}
	return &domain.AuthResponse{Token: "test-token", User: domain.UserInfo{ID: 1}}, nil
}

func (m *mockAuthService) LoginByEmail(ctx context.Context, email, password string) (*domain.AuthResponse, error) {
	if m.loginByEmail != nil {
		return m.loginByEmail(ctx, email, password)
	}
	return &domain.AuthResponse{Token: "test-token", User: domain.UserInfo{ID: 1}}, nil
}

func (m *mockAuthService) RegisterByEmail(ctx context.Context, email, password, name string) (*domain.AuthResponse, error) {
	if m.registerByEmail != nil {
		return m.registerByEmail(ctx, email, password, name)
	}
	return &domain.AuthResponse{Token: "test-token", User: domain.UserInfo{ID: 1}}, nil
}

func (m *mockAuthService) GetUserByID(ctx context.Context, userID int64) (*domain.UserInfo, error) {
	if m.getUserByID != nil {
		return m.getUserByID(ctx, userID)
	}
	return &domain.UserInfo{ID: userID, Phone: strPtr("13800138000")}, nil
}

func (m *mockAuthService) UpdateUser(ctx context.Context, userID int64, name *string, email *string) (*domain.UserInfo, error) {
	if m.updateUser != nil {
		return m.updateUser(ctx, userID, name, email)
	}
	return &domain.UserInfo{ID: userID, Name: name, Email: email}, nil
}

func strPtr(s string) *string { return &s }

func setupTestRouter(h *Handler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	return r
}

// ========== SendSMSCode ==========

func TestSendSMSCode_InvalidJSON(t *testing.T) {
	h := NewHandler(&mockAuthService{})
	r := setupTestRouter(h)
	r.POST("/auth/send-sms-code", h.SendSMSCode)

	req := httptest.NewRequest("POST", "/auth/send-sms-code", bytes.NewReader([]byte("not json")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid JSON, got %d", w.Code)
	}
}

func TestSendSMSCode_MissingPhone(t *testing.T) {
	h := NewHandler(&mockAuthService{})
	r := setupTestRouter(h)
	r.POST("/auth/send-sms-code", h.SendSMSCode)

	body, _ := json.Marshal(map[string]string{})
	req := httptest.NewRequest("POST", "/auth/send-sms-code", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for missing phone, got %d", w.Code)
	}
}

func TestSendSMSCode_Success(t *testing.T) {
	h := NewHandler(&mockAuthService{})
	r := setupTestRouter(h)
	r.POST("/auth/send-sms-code", h.SendSMSCode)

	body, _ := json.Marshal(map[string]string{"phone": "13800138000"})
	req := httptest.NewRequest("POST", "/auth/send-sms-code", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response failed: %v", err)
	}
	if resp["message"] != "验证码已发送" {
		t.Errorf("expected message '验证码已发送', got %q", resp["message"])
	}
}

// ========== LoginByPhone ==========

func TestLoginByPhone_MissingFields(t *testing.T) {
	h := NewHandler(&mockAuthService{})
	r := setupTestRouter(h)
	r.POST("/auth/login-by-phone", h.LoginByPhone)

	tests := []struct {
		name string
		body map[string]string
	}{
		{"missing phone", map[string]string{"code": "123456"}},
		{"missing code", map[string]string{"phone": "13800138000"}},
		{"empty body", map[string]string{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.body)
			req := httptest.NewRequest("POST", "/auth/login-by-phone", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if w.Code != http.StatusBadRequest {
				t.Errorf("expected 400, got %d: %s", w.Code, w.Body.String())
			}
		})
	}
}

func TestLoginByPhone_Success(t *testing.T) {
	h := NewHandler(&mockAuthService{})
	r := setupTestRouter(h)
	r.POST("/auth/login-by-phone", h.LoginByPhone)

	body, _ := json.Marshal(map[string]string{"phone": "13800138000", "code": "123456"})
	req := httptest.NewRequest("POST", "/auth/login-by-phone", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp domain.AuthResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response failed: %v", err)
	}
	if resp.Token == "" {
		t.Error("expected token in response")
	}
}

// ========== LoginByEmail ==========

func TestLoginByEmail_InvalidEmail(t *testing.T) {
	h := NewHandler(&mockAuthService{})
	r := setupTestRouter(h)
	r.POST("/auth/login-by-email", h.LoginByEmail)

	body, _ := json.Marshal(map[string]string{"email": "not-email", "password": "123456"})
	req := httptest.NewRequest("POST", "/auth/login-by-email", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid email, got %d", w.Code)
	}
}

// ========== RegisterByEmail ==========

func TestRegisterByEmail_Validation(t *testing.T) {
	h := NewHandler(&mockAuthService{})
	r := setupTestRouter(h)
	r.POST("/auth/register-by-email", h.RegisterByEmail)

	tests := []struct {
		name       string
		body       map[string]string
		wantStatus int
	}{
		{"missing email", map[string]string{"password": "123456", "name": "Test"}, http.StatusBadRequest},
		{"missing password", map[string]string{"email": "test@example.com", "name": "Test"}, http.StatusBadRequest},
		{"missing name", map[string]string{"email": "test@example.com", "password": "123456"}, http.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.body)
			req := httptest.NewRequest("POST", "/auth/register-by-email", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("expected %d, got %d: %s", tt.wantStatus, w.Code, w.Body.String())
			}
		})
	}
}

// ========== GetMe ==========

func TestGetMe_ReturnsUser(t *testing.T) {
	h := NewHandler(&mockAuthService{})
	r := setupTestRouter(h)
	// 模拟已认证用户
	r.GET("/me", func(c *gin.Context) {
		c.Set("user_id", int64(42))
		c.Next()
	}, h.GetMe)

	req := httptest.NewRequest("GET", "/me", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}
