// 本文件提供轮回绝境副本的玩家接口处理器。
package api

import (
	"encoding/json"
	"errors"
	"net/http"

	"hero3/internal/app/game"
)

// ReincarnationConfig 返回轮回绝境配置。
func (h *Handlers) ReincarnationConfig(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, h.gameService.GetReincarnationConfigForPlayer())
}

// ReincarnationRun 返回玩家当前轮回绝境实例。
func (h *Handlers) ReincarnationRun(w http.ResponseWriter, r *http.Request) {
	playerID := r.URL.Query().Get("playerId")
	if playerID == "" {
		writeError(w, http.StatusBadRequest, "playerId is required")
		return
	}
	if !h.requireOwnership(w, r, playerID) {
		return
	}
	result, err := h.gameService.GetActiveReincarnationRun(playerID)
	if err != nil {
		writeReincarnationError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

// StartReincarnation 开启轮回绝境实例。
func (h *Handlers) StartReincarnation(w http.ResponseWriter, r *http.Request) {
	var req game.StartReincarnationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if !h.requireOwnership(w, r, req.PlayerID) {
		return
	}
	result, err := h.gameService.StartReincarnationRun(req.PlayerID, req.Level)
	if err != nil {
		writeReincarnationError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, result)
}

// AttackReincarnationWave 结算进攻波。
func (h *Handlers) AttackReincarnationWave(w http.ResponseWriter, r *http.Request) {
	var req game.ReincarnationTroopRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if !h.requireOwnership(w, r, req.PlayerID) {
		return
	}
	result, err := h.gameService.AttackReincarnationWave(req.PlayerID, r.PathValue("waveId"), req.Troops, req.ClientActionID)
	if err != nil {
		writeReincarnationError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

// ReadyReincarnationDefense 结算防守波。
func (h *Handlers) ReadyReincarnationDefense(w http.ResponseWriter, r *http.Request) {
	var req game.ReincarnationTroopRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if !h.requireOwnership(w, r, req.PlayerID) {
		return
	}
	result, err := h.gameService.ReadyReincarnationDefense(req.PlayerID, r.PathValue("waveId"), req.Troops, req.ClientActionID)
	if err != nil {
		writeReincarnationError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

// SettleReincarnation 结算当前轮回绝境实例。
func (h *Handlers) SettleReincarnation(w http.ResponseWriter, r *http.Request) {
	var req struct {
		PlayerID string `json:"playerId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if !h.requireOwnership(w, r, req.PlayerID) {
		return
	}
	result, err := h.gameService.SettleReincarnationRun(req.PlayerID)
	if err != nil {
		writeReincarnationError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

// ReincarnationReports 返回轮回绝境副本战报。
func (h *Handlers) ReincarnationReports(w http.ResponseWriter, r *http.Request) {
	playerID := r.URL.Query().Get("playerId")
	if playerID == "" {
		writeError(w, http.StatusBadRequest, "playerId is required")
		return
	}
	if !h.requireOwnership(w, r, playerID) {
		return
	}
	page, pageSize, ok := parsePageQuery(w, r)
	if !ok {
		return
	}
	result, err := h.gameService.ListReincarnationReports(playerID, page, pageSize)
	if err != nil {
		writeReincarnationError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func writeReincarnationError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, game.ErrPlayerNotFound), errors.Is(err, game.ErrReincarnationRunNotFound):
		writeError(w, http.StatusNotFound, err.Error())
	case errors.Is(err, game.ErrReincarnationActive), errors.Is(err, game.ErrInvalidReincarnation), errors.Is(err, game.ErrInvalidAmount), errors.Is(err, game.ErrNoUnitsSelected), errors.Is(err, game.ErrInsufficientArmy):
		writeError(w, http.StatusBadRequest, err.Error())
	default:
		writeError(w, http.StatusInternalServerError, err.Error())
	}
}
