// 本文件实现 PVP 系统 HTTP 接口处理器。
package api

import (
	"errors"
	"net/http"
	"strconv"

	"hero3/internal/app/game"
)

// PvpTargets 返回 PVP 玩家目标列表。
func (h *Handlers) PvpTargets(w http.ResponseWriter, r *http.Request) {
	playerID := r.URL.Query().Get("playerId")
	if !h.requireOwnership(w, r, playerID) {
		return
	}
	filter := game.PvpTargetFilter{
		CenterX: queryInt(r, "centerX", 0),
		CenterY: queryInt(r, "centerY", 0),
		Radius:  queryInt(r, "radius", 0),
		Limit:   queryInt(r, "limit", 0),
	}
	result, err := h.gameService.ListPvpTargetsInArea(playerID, filter)
	if err != nil {
		writePvpError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

// queryInt 读取整数查询参数，失败时返回默认值。
func queryInt(r *http.Request, key string, fallback int) int {
	raw := r.URL.Query().Get(key)
	if raw == "" {
		return fallback
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	return value
}

// PvpTarget 返回单个 PVP 目标摘要。
func (h *Handlers) PvpTarget(w http.ResponseWriter, r *http.Request) {
	playerID := r.URL.Query().Get("playerId")
	if !h.requireOwnership(w, r, playerID) {
		return
	}
	result, err := h.gameService.GetPvpTarget(playerID, r.PathValue("targetPlayerId"))
	if err != nil {
		writePvpError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

// ScoutPvpTarget 侦查一个玩家目标。
func (h *Handlers) ScoutPvpTarget(w http.ResponseWriter, r *http.Request) {
	var payload game.PvpScoutRequest
	if !decodeJSON(w, r, &payload) {
		return
	}
	if !h.requireOwnership(w, r, payload.PlayerID) {
		return
	}
	result, err := h.gameService.ScoutPvpTarget(payload)
	if err != nil {
		writePvpError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

// StartPvpAttack 发起 PVP 攻击或掠夺行军。
func (h *Handlers) StartPvpAttack(w http.ResponseWriter, r *http.Request) {
	var payload game.PvpAttackRequest
	if !decodeJSON(w, r, &payload) {
		return
	}
	if !h.requireOwnership(w, r, payload.PlayerID) {
		return
	}
	result, err := h.gameService.StartPvpAttack(payload)
	if err != nil {
		writePvpError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

// PvpMarches 返回玩家相关 PVP 行军。
func (h *Handlers) PvpMarches(w http.ResponseWriter, r *http.Request) {
	playerID := r.URL.Query().Get("playerId")
	if !h.requireOwnership(w, r, playerID) {
		return
	}
	result, err := h.gameService.ListPvpMarches(playerID)
	if err != nil {
		writePvpError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

// RecallPvpMarch 召回玩家自己的 PVP 行军。
func (h *Handlers) RecallPvpMarch(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		PlayerID string `json:"playerId"`
	}
	if !decodeJSON(w, r, &payload) {
		return
	}
	if !h.requireOwnership(w, r, payload.PlayerID) {
		return
	}
	result, err := h.gameService.RecallPvpMarch(payload.PlayerID, r.PathValue("marchId"))
	if err != nil {
		writePvpError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

// AcceleratePvpMarch 使用城金加速玩家自己的 PVP 行军。
func (h *Handlers) AcceleratePvpMarch(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		PlayerID string `json:"playerId"`
	}
	if !decodeJSON(w, r, &payload) {
		return
	}
	if !h.requireOwnership(w, r, payload.PlayerID) {
		return
	}
	result, err := h.gameService.AcceleratePvpMarch(payload.PlayerID, r.PathValue("marchId"))
	if err != nil {
		writePvpError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

// PvpBattles 返回玩家相关 PVP 战斗。
func (h *Handlers) PvpBattles(w http.ResponseWriter, r *http.Request) {
	playerID := r.URL.Query().Get("playerId")
	if !h.requireOwnership(w, r, playerID) {
		return
	}
	result, err := h.gameService.ListPvpBattles(playerID)
	if err != nil {
		writePvpError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

// PvpBattle 返回单场 PVP 战斗详情。
func (h *Handlers) PvpBattle(w http.ResponseWriter, r *http.Request) {
	playerID := r.URL.Query().Get("playerId")
	if !h.requireOwnership(w, r, playerID) {
		return
	}
	result, err := h.gameService.GetPvpBattle(playerID, r.PathValue("battleId"))
	if err != nil {
		writePvpError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

// PvpState 返回玩家 PVP 状态、积分和复仇概览。
func (h *Handlers) PvpState(w http.ResponseWriter, r *http.Request) {
	playerID := r.URL.Query().Get("playerId")
	if !h.requireOwnership(w, r, playerID) {
		return
	}
	result, err := h.gameService.GetPvpState(playerID)
	if err != nil {
		writePvpError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

// PvpRevenge 返回玩家当前可用复仇记录。
func (h *Handlers) PvpRevenge(w http.ResponseWriter, r *http.Request) {
	playerID := r.URL.Query().Get("playerId")
	if !h.requireOwnership(w, r, playerID) {
		return
	}
	result, err := h.gameService.ListPvpRevengeRecords(playerID)
	if err != nil {
		writePvpError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

// PvpSeason 返回当前 PVP 赛季摘要和玩家自身排名。
func (h *Handlers) PvpSeason(w http.ResponseWriter, r *http.Request) {
	playerID := r.URL.Query().Get("playerId")
	if !h.requireOwnership(w, r, playerID) {
		return
	}
	result, err := h.gameService.GetPvpSeason(playerID)
	if err != nil {
		writePvpError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

// PvpRankings 返回当前 PVP 排行榜。
func (h *Handlers) PvpRankings(w http.ResponseWriter, r *http.Request) {
	playerID := r.URL.Query().Get("playerId")
	if !h.requireOwnership(w, r, playerID) {
		return
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	result, err := h.gameService.ListPvpRankings(playerID, limit)
	if err != nil {
		writePvpError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

// writePvpError 按 PVP 常见错误转换 HTTP 状态码。
func writePvpError(w http.ResponseWriter, err error) {
	status := http.StatusBadRequest
	switch {
	case errors.Is(err, game.ErrPlayerNotFound):
		status = http.StatusNotFound
	case errors.Is(err, game.ErrPvpTargetSelf), errors.Is(err, game.ErrPvpSameAccountTarget):
		status = http.StatusForbidden
	case errors.Is(err, game.ErrPvpTargetProtected), errors.Is(err, game.ErrPvpDailyLimitReached):
		status = http.StatusForbidden
	case errors.Is(err, game.ErrPvpMarchNotRecallable):
		status = http.StatusConflict
	case errors.Is(err, game.ErrPvpMarchNotAccelerable):
		status = http.StatusConflict
	case errors.Is(err, game.ErrInsufficientArmy):
		status = http.StatusUnprocessableEntity
	case errors.Is(err, game.ErrInsufficientCityGold):
		status = http.StatusUnprocessableEntity
	case errors.Is(err, game.ErrNoUnitsSelected):
		status = http.StatusBadRequest
	}
	writeError(w, status, err.Error())
}
