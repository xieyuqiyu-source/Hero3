// 本文件提供 HTTP 处理器的基础结构、通用解析和响应写入工具。
package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"hero3/internal/app/game"
	"hero3/internal/platform/auth"
	"hero3/internal/platform/config"
)

type Handlers struct {
	cfg         config.Config
	gameService *game.Service
}

type accountRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type createPlayerRequest struct {
	AccountID string `json:"accountId"`
	Nickname  string `json:"nickname"`
	Faction   string `json:"faction"`
	GeneralID string `json:"generalId"`
}

type upgradeBuildingRequest struct {
	PlayerID   string `json:"playerId"`
	BuildingID string `json:"buildingId"`
}

// NewHandlers 创建 API 处理器集合。
func NewHandlers(cfg config.Config, gameService *game.Service) *Handlers {
	return &Handlers{
		cfg:         cfg,
		gameService: gameService,
	}
}

func (h *Handlers) Health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"status":      "ok",
		"service":     h.cfg.ServiceName,
		"version":     h.cfg.Version,
		"environment": h.cfg.Environment,
		"time":        time.Now().UTC().Format(time.RFC3339),
	})
}

func (h *Handlers) Meta(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"name":       "Hero3",
		"apiVersion": "v1",
		"service":    h.cfg.ServiceName,
		"version":    h.cfg.Version,
	})
}

func (h *Handlers) GameBootstrap(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, h.gameService.Bootstrap())
}

func (h *Handlers) GameState(w http.ResponseWriter, r *http.Request) {
	playerID := r.URL.Query().Get("playerId")
	if playerID == "" {
		writeError(w, http.StatusBadRequest, "playerId is required")
		return
	}
	if !h.requireOwnership(w, r, playerID) {
		return
	}
	state, err := h.gameService.GetState(playerID)
	if err != nil {
		if errors.Is(err, game.ErrPlayerNotFound) {
			writeError(w, http.StatusNotFound, "player not found")
		} else {
			writeError(w, http.StatusInternalServerError, err.Error())
		}
		return
	}
	writeJSON(w, http.StatusOK, state)
}

func parsePageQuery(w http.ResponseWriter, r *http.Request) (int, int, bool) {
	page := 1
	if raw := r.URL.Query().Get("page"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 1 {
			writeError(w, http.StatusBadRequest, "page must be a positive integer")
			return 0, 0, false
		}
		page = parsed
	}

	pageSize := 10
	if raw := r.URL.Query().Get("pageSize"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 1 {
			writeError(w, http.StatusBadRequest, "pageSize must be a positive integer")
			return 0, 0, false
		}
		pageSize = parsed
	}
	if pageSize > 50 {
		pageSize = 50
	}
	return page, pageSize, true
}

func parseLimitOffset(r *http.Request, defaultLimit int, maxLimit int) (int, int) {
	limit := defaultLimit
	if raw := r.URL.Query().Get("limit"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 {
			limit = parsed
		}
	}
	if maxLimit > 0 && limit > maxLimit {
		limit = maxLimit
	}

	offset := 0
	if raw := r.URL.Query().Get("offset"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 {
			offset = parsed
		}
	}
	return limit, offset
}

// writeJSON 写入 JSON 响应；响应头已经发出后，写失败不能再追加错误响应。
func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(payload); err != nil {
		return
	}
}

// decodeJSON 解析请求体 JSON，并在格式错误时直接返回 400。
func decodeJSON(w http.ResponseWriter, r *http.Request, target any) bool {
	if err := json.NewDecoder(r.Body).Decode(target); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json body")
		return false
	}
	return true
}

// writeError 以统一 JSON 格式写入错误响应。
func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

// --- Gold Handlers ---

func (h *Handlers) requireOwnership(w http.ResponseWriter, r *http.Request, playerID string) bool {
	if auth.IsAdminFromContext(r.Context()) {
		return true
	}

	accountID, ok := auth.AccountIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return false
	}

	owns, err := h.gameService.OwnsPlayer(accountID, playerID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "ownership check failed")
		return false
	}
	if !owns {
		writeError(w, http.StatusForbidden, "you don't own this player")
		return false
	}
	return true
}

// requireAccount 校验当前请求的 accountID 是否匹配指定 accountID
// admin 请求直接通过

func (h *Handlers) requireAccount(w http.ResponseWriter, r *http.Request, accountID string) bool {
	if auth.IsAdminFromContext(r.Context()) {
		return true
	}

	ctxAccountID, ok := auth.AccountIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return false
	}

	if ctxAccountID != accountID {
		writeError(w, http.StatusForbidden, "account mismatch")
		return false
	}
	return true
}

// --- Generals 配置（GM） ---
