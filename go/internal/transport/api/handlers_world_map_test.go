package api

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"hero3/internal/app/game"
	"hero3/internal/platform/auth"
	"hero3/internal/platform/config"
)

// 本文件验证世界地图玩家接口的真实 HTTP 鉴权行为。

// TestWorldMapHandlersRequireOwnedViewer 验证世界地图玩家接口必须登录且只能使用自己的玩家存档查看。
func TestWorldMapHandlersRequireOwnedViewer(t *testing.T) {
	router, playerA, playerB, tokenA, tokenB := newWorldMapAuthTestRouter(t)

	assertWorldMapStatus(t, router, http.MethodGet, "/api/v1/world-map/view?playerId="+playerA, "", http.StatusUnauthorized)
	assertWorldMapStatus(t, router, http.MethodGet, "/api/v1/world-map/view?playerId="+playerA, tokenB, http.StatusForbidden)
	assertWorldMapStatus(t, router, http.MethodGet, "/api/v1/world-map/view?playerId="+playerA, tokenA, http.StatusOK)

	targetURL := "/api/v1/world-map/targets/player_city/" + playerB + "?viewerId=" + playerA
	assertWorldMapStatus(t, router, http.MethodGet, targetURL, tokenB, http.StatusForbidden)
	assertWorldMapStatus(t, router, http.MethodGet, targetURL, tokenA, http.StatusOK)
}

// TestWorldMapAdminHandlersRequireAdminToken 验证 GM 世界地图坐标接口只能使用 Admin Token。
func TestWorldMapAdminHandlersRequireAdminToken(t *testing.T) {
	router, playerA, playerB, tokenA, _ := newWorldMapAuthTestRouter(t)

	assertWorldMapStatus(t, router, http.MethodGet, "/api/v1/admin/world-map/occupancy", "", http.StatusForbidden)
	assertWorldMapStatus(t, router, http.MethodGet, "/api/v1/admin/world-map/occupancy", tokenA, http.StatusForbidden)
	assertWorldMapAdminStatus(t, router, http.MethodGet, "/api/v1/admin/world-map/occupancy", "", http.StatusOK)
	assertWorldMapStatus(t, router, http.MethodGet, "/api/v1/admin/world-map/positions/check?x=0&y=0", tokenA, http.StatusForbidden)
	checkBody := assertWorldMapAdminBody(t, router, http.MethodGet, "/api/v1/admin/world-map/positions/check?x=21&y=22", "", http.StatusOK)
	var check struct {
		X        int    `json:"x"`
		Y        int    `json:"y"`
		Occupied bool   `json:"occupied"`
		PlayerID string `json:"playerId"`
	}
	if err := json.Unmarshal([]byte(checkBody), &check); err != nil {
		t.Fatalf("解析 GM 坐标检查响应失败: %v body=%s", err, checkBody)
	}
	if check.X != 21 || check.Y != 22 || check.Occupied || check.PlayerID != "" {
		t.Fatalf("expected empty coordinate check, got %+v", check)
	}

	positionURL := "/api/v1/admin/world-map/positions/" + playerA
	assertWorldMapStatus(t, router, http.MethodGet, positionURL, tokenA, http.StatusForbidden)
	assertWorldMapAdminStatus(t, router, http.MethodGet, positionURL, "", http.StatusOK)
	assertWorldMapAdminStatus(t, router, http.MethodPut, positionURL, `{"x":21,"y":22}`, http.StatusOK)
	checkBody = assertWorldMapAdminBody(t, router, http.MethodGet, "/api/v1/admin/world-map/positions/check?x=21&y=22", "", http.StatusOK)
	if err := json.Unmarshal([]byte(checkBody), &check); err != nil {
		t.Fatalf("解析 GM 已占坐标检查响应失败: %v body=%s", err, checkBody)
	}
	if !check.Occupied || check.PlayerID != playerA {
		t.Fatalf("expected occupied coordinate check for %s, got %+v", playerA, check)
	}
	assertWorldMapAdminStatus(t, router, http.MethodPut, "/api/v1/admin/world-map/positions/"+playerB, `{"x":21,"y":22}`, http.StatusBadRequest)
}

// TestWorldMapHandlersUseRuntimeQueryParametersAndHidePrivateFields 验证真实 HTTP 路由保留 0 坐标参数且不泄露隐藏字段。
func TestWorldMapHandlersUseRuntimeQueryParametersAndHidePrivateFields(t *testing.T) {
	router, playerA, playerB, tokenA, _ := newWorldMapAuthTestRouter(t)

	viewBody := assertWorldMapBody(t, router, http.MethodGet, "/api/v1/world-map/view?playerId="+playerA+"&centerX=0&centerY=0&radius=100", tokenA, http.StatusOK)
	var view struct {
		CenterX int `json:"centerX"`
		CenterY int `json:"centerY"`
		Radius  int `json:"radius"`
		Targets []struct {
			PlayerID string `json:"playerId"`
		} `json:"targets"`
	}
	if err := json.Unmarshal([]byte(viewBody), &view); err != nil {
		t.Fatalf("解析世界地图视图响应失败: %v body=%s", err, viewBody)
	}
	if view.CenterX != 0 || view.CenterY != 0 || view.Radius != 100 {
		t.Fatalf("世界地图视图未按请求参数返回: center=(%d,%d) radius=%d", view.CenterX, view.CenterY, view.Radius)
	}
	if len(view.Targets) == 0 {
		t.Fatalf("世界地图全量半径视图应返回玩家城池")
	}

	targetBody := assertWorldMapBody(t, router, http.MethodGet, "/api/v1/world-map/targets/player_city/"+playerB+"?viewerId="+playerA, tokenA, http.StatusOK)
	assertWorldMapBodyDoesNotContain(t, targetBody, "totalArmy", "resources", "generals", "garrison")
}

// newWorldMapAuthTestRouter 创建带真实 JWT 中间件和内存仓储的测试路由。
func newWorldMapAuthTestRouter(t *testing.T) (http.Handler, string, string, string, string) {
	t.Helper()
	if err := game.LoadFactionsConfig("../../../config/factions.json"); err != nil {
		t.Fatalf("LoadFactionsConfig failed: %v", err)
	}
	if err := game.LoadGeneralsConfig("../../../config/generals.json"); err != nil {
		t.Fatalf("LoadGeneralsConfig failed: %v", err)
	}

	repo := game.NewMemoryRepository()
	svc := game.NewServiceWithRepository(repo)
	now := time.Now()
	for _, account := range []game.Account{
		{ID: "account_world_api_a", Username: "world_api_a", CreatedAt: now},
		{ID: "account_world_api_b", Username: "world_api_b", CreatedAt: now},
	} {
		if err := repo.CreateAccount(account); err != nil {
			t.Fatalf("CreateAccount %s failed: %v", account.ID, err)
		}
	}
	playerA, _, err := svc.CreatePlayer("account_world_api_a", "地图甲", "wei", "caocao")
	if err != nil {
		t.Fatalf("CreatePlayer A failed: %v", err)
	}
	playerB, _, err := svc.CreatePlayer("account_world_api_b", "地图乙", "shu", "liubei")
	if err != nil {
		t.Fatalf("CreatePlayer B failed: %v", err)
	}

	cfg := config.Config{
		JWTSecret:      "world-map-auth-test-secret",
		AdminToken:     "world-map-admin-test-token",
		TokenTTL:       time.Hour,
		AllowedOrigins: []string{"http://localhost:5173"},
	}
	router := NewRouter(RouterOptions{
		Config:      cfg,
		Logger:      slog.New(slog.NewTextHandler(io.Discard, nil)),
		GameService: svc,
	})
	tokenA := issueWorldMapAuthToken(t, cfg, "account_world_api_a")
	tokenB := issueWorldMapAuthToken(t, cfg, "account_world_api_b")
	return router, playerA, playerB, tokenA, tokenB
}

// issueWorldMapAuthToken 签发测试用玩家 JWT。
func issueWorldMapAuthToken(t *testing.T, cfg config.Config, accountID string) string {
	t.Helper()
	token, err := auth.IssueToken(auth.Config{JWTSecret: cfg.JWTSecret, TokenTTL: cfg.TokenTTL}, accountID)
	if err != nil {
		t.Fatalf("IssueToken failed: %v", err)
	}
	return token
}

// assertWorldMapStatus 发起请求并断言 HTTP 状态码。
func assertWorldMapStatus(t *testing.T, router http.Handler, method string, path string, token string, want int) {
	t.Helper()
	req := httptest.NewRequest(method, path, nil)
	if strings.TrimSpace(token) != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)
	if recorder.Code != want {
		t.Fatalf("%s %s expected status %d, got %d body=%s", method, path, want, recorder.Code, recorder.Body.String())
	}
}

// assertWorldMapBody 发起玩家世界地图请求并返回响应体。
func assertWorldMapBody(t *testing.T, router http.Handler, method string, path string, token string, want int) string {
	t.Helper()
	req := httptest.NewRequest(method, path, nil)
	if strings.TrimSpace(token) != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)
	if recorder.Code != want {
		t.Fatalf("%s %s expected status %d, got %d body=%s", method, path, want, recorder.Code, recorder.Body.String())
	}
	return recorder.Body.String()
}

// assertWorldMapBodyDoesNotContain 验证响应体没有泄露指定字段。
func assertWorldMapBodyDoesNotContain(t *testing.T, body string, fragments ...string) {
	t.Helper()
	for _, fragment := range fragments {
		if strings.Contains(body, fragment) {
			t.Fatalf("世界地图响应不应包含 %q body=%s", fragment, body)
		}
	}
}

// assertWorldMapAdminStatus 使用 GM Token 发起世界地图后台请求并断言状态码。
func assertWorldMapAdminStatus(t *testing.T, router http.Handler, method string, path string, body string, want int) {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("X-Admin-Token", "world-map-admin-test-token")
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)
	if recorder.Code != want {
		t.Fatalf("%s %s expected admin status %d, got %d body=%s", method, path, want, recorder.Code, recorder.Body.String())
	}
}

// assertWorldMapAdminBody 使用 GM Token 发起世界地图后台请求并返回响应体。
func assertWorldMapAdminBody(t *testing.T, router http.Handler, method string, path string, body string, want int) string {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("X-Admin-Token", "world-map-admin-test-token")
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)
	if recorder.Code != want {
		t.Fatalf("%s %s expected admin status %d, got %d body=%s", method, path, want, recorder.Code, recorder.Body.String())
	}
	return recorder.Body.String()
}
