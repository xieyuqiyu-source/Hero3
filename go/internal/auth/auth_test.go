package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const testSecret = "test-secret-not-for-prod"
const testAdmin = "test-admin-token"

func testCfg() Config {
	return Config{
		JWTSecret:  testSecret,
		AdminToken: testAdmin,
		TokenTTL:   1 * time.Hour,
	}
}

// --- IssueToken / ParseToken ---

func TestIssueAndParseToken_Roundtrip(t *testing.T) {
	cfg := testCfg()
	token, err := IssueToken(cfg, "acc_alice")
	if err != nil {
		t.Fatalf("issue token: %v", err)
	}
	if token == "" {
		t.Fatalf("expected non-empty token")
	}

	accountID, err := ParseToken(cfg, token)
	if err != nil {
		t.Fatalf("parse token: %v", err)
	}
	if accountID != "acc_alice" {
		t.Errorf("expected acc_alice, got %q", accountID)
	}
}

func TestIssueToken_RequiresSecret(t *testing.T) {
	cfg := Config{}
	_, err := IssueToken(cfg, "acc_x")
	if err == nil {
		t.Errorf("expected error when secret is empty")
	}
}

func TestParseToken_Expired(t *testing.T) {
	cfg := testCfg()
	cfg.TokenTTL = -1 * time.Hour // 已经过期
	token, err := IssueToken(cfg, "acc_x")
	if err != nil {
		t.Fatalf("issue: %v", err)
	}

	cfg.TokenTTL = 1 * time.Hour
	_, err = ParseToken(cfg, token)
	if err != ErrExpiredToken {
		t.Errorf("expected ErrExpiredToken, got %v", err)
	}
}

func TestParseToken_InvalidSignature(t *testing.T) {
	cfg := testCfg()
	token, _ := IssueToken(cfg, "acc_x")

	otherCfg := Config{JWTSecret: "different-secret"}
	_, err := ParseToken(otherCfg, token)
	if err != ErrInvalidToken {
		t.Errorf("expected ErrInvalidToken, got %v", err)
	}
}

func TestParseToken_MalformedToken(t *testing.T) {
	cfg := testCfg()
	_, err := ParseToken(cfg, "not.a.valid.jwt")
	if err != ErrInvalidToken {
		t.Errorf("expected ErrInvalidToken for malformed, got %v", err)
	}
}

func TestParseToken_WrongSigningMethod(t *testing.T) {
	// 用 none 签名方法构造一个 token
	tok := jwt.New(jwt.SigningMethodNone)
	tokenStr, _ := tok.SignedString(jwt.UnsafeAllowNoneSignatureType)

	cfg := testCfg()
	_, err := ParseToken(cfg, tokenStr)
	if err != ErrInvalidToken {
		t.Errorf("expected ErrInvalidToken for wrong signing method, got %v", err)
	}
}

// --- AuthMiddleware ---

func newTestHandler(t *testing.T, expectAccountID string, expectAdmin bool) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		accountID, _ := AccountIDFromContext(r.Context())
		isAdmin := IsAdminFromContext(r.Context())

		if accountID != expectAccountID {
			t.Errorf("expected accountID=%q, got %q", expectAccountID, accountID)
		}
		if isAdmin != expectAdmin {
			t.Errorf("expected isAdmin=%v, got %v", expectAdmin, isAdmin)
		}

		w.WriteHeader(http.StatusOK)
	})
}

func TestMiddleware_PublicPath_NoToken(t *testing.T) {
	cfg := testCfg()
	mw := AuthMiddleware(cfg, []string{"/healthz", "/api/v1/meta"})

	handler := mw(newTestHandler(t, "", false))

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for public path, got %d", w.Code)
	}
}

func TestMiddleware_ProtectedPath_NoToken_Returns401(t *testing.T) {
	cfg := testCfg()
	mw := AuthMiddleware(cfg, []string{})

	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("handler should not be reached without token")
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/game/state", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestMiddleware_ProtectedPath_ValidToken(t *testing.T) {
	cfg := testCfg()
	token, _ := IssueToken(cfg, "acc_alice")

	mw := AuthMiddleware(cfg, []string{})
	handler := mw(newTestHandler(t, "acc_alice", false))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/game/state", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 with valid token, got %d", w.Code)
	}
}

func TestMiddleware_ProtectedPath_InvalidToken_Returns401(t *testing.T) {
	cfg := testCfg()
	mw := AuthMiddleware(cfg, []string{})
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("handler should not be reached with invalid token")
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/game/state", nil)
	req.Header.Set("Authorization", "Bearer invalid.token.here")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for invalid token, got %d", w.Code)
	}
}

func TestMiddleware_AdminPath_NoToken_Returns403(t *testing.T) {
	cfg := testCfg()
	mw := AuthMiddleware(cfg, []string{})
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("handler should not be reached without admin token")
	}))

	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/gold/add", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403 for admin path without admin token, got %d", w.Code)
	}
}

func TestMiddleware_AdminPath_WithJWTOnly_Returns403(t *testing.T) {
	// 即使有有效 JWT，没有 admin token 也不能访问 /admin/*
	cfg := testCfg()
	token, _ := IssueToken(cfg, "acc_alice")
	mw := AuthMiddleware(cfg, []string{})
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("handler should not be reached")
	}))

	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/gold/add", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403 for admin path with only JWT, got %d", w.Code)
	}
}

func TestMiddleware_AdminPath_ValidAdminToken(t *testing.T) {
	cfg := testCfg()
	mw := AuthMiddleware(cfg, []string{})
	handler := mw(newTestHandler(t, "", true)) // admin: accountID 为空，isAdmin=true

	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/gold/add", nil)
	req.Header.Set("X-Admin-Token", testAdmin)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 with admin token, got %d", w.Code)
	}
}

func TestMiddleware_InvalidAdminToken_Returns401(t *testing.T) {
	cfg := testCfg()
	mw := AuthMiddleware(cfg, []string{})
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("handler should not be reached")
	}))

	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/gold/add", nil)
	req.Header.Set("X-Admin-Token", "wrong-token")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for invalid admin token, got %d", w.Code)
	}
}

func TestMiddleware_PublicReportPath(t *testing.T) {
	// 战报详情 /api/v1/reports/{reportId} 公开（用于分享）
	cfg := testCfg()
	mw := AuthMiddleware(cfg, []string{})
	handler := mw(newTestHandler(t, "", false))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/reports/abc123", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for public report path, got %d", w.Code)
	}
}

func TestMiddleware_MalformedAuthHeader(t *testing.T) {
	cfg := testCfg()
	mw := AuthMiddleware(cfg, []string{})
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("handler should not be reached")
	}))

	tests := []struct {
		name   string
		header string
	}{
		{"missing Bearer prefix", "abc.def.ghi"},
		{"Basic auth", "Basic dXNlcjpwYXNz"},
		{"empty", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/game/state", nil)
			if tt.header != "" {
				req.Header.Set("Authorization", tt.header)
			}
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)

			if w.Code != http.StatusUnauthorized {
				t.Errorf("expected 401 for %q, got %d", tt.name, w.Code)
			}
		})
	}
}

// --- Context helpers ---

func TestAccountIDFromContext_Empty(t *testing.T) {
	ctx := httptest.NewRequest(http.MethodGet, "/", nil).Context()
	id, ok := AccountIDFromContext(ctx)
	if ok || id != "" {
		t.Errorf("expected empty + false, got %q + %v", id, ok)
	}
}

func TestIsAdminFromContext_Default(t *testing.T) {
	ctx := httptest.NewRequest(http.MethodGet, "/", nil).Context()
	if IsAdminFromContext(ctx) {
		t.Errorf("expected false by default")
	}
}
