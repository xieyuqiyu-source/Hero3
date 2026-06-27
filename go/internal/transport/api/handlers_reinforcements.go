// 本文件归口增援系统的 HTTP 接口处理。
package api

import (
	"errors"
	"net/http"

	"hero3/internal/app/game"
)

// SendReinforcement 发起增援。
func (h *Handlers) SendReinforcement(w http.ResponseWriter, r *http.Request) {
	var payload game.SendReinforcementRequest
	if !decodeJSON(w, r, &payload) {
		return
	}
	if !h.requireOwnership(w, r, payload.FromPlayerID) {
		return
	}
	result, err := h.gameService.SendReinforcement(payload)
	if err != nil {
		h.writeReinforcementError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

// ListSentReinforcements 查询我派出的增援。
func (h *Handlers) ListSentReinforcements(w http.ResponseWriter, r *http.Request) {
	playerID := r.URL.Query().Get("playerId")
	if !h.requireOwnership(w, r, playerID) {
		return
	}
	result, err := h.gameService.ListSentReinforcements(playerID)
	if err != nil {
		h.writeReinforcementError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

// ListReceivedReinforcements 查询我收到的增援。
func (h *Handlers) ListReceivedReinforcements(w http.ResponseWriter, r *http.Request) {
	playerID := r.URL.Query().Get("playerId")
	if !h.requireOwnership(w, r, playerID) {
		return
	}
	result, err := h.gameService.ListReceivedReinforcements(playerID)
	if err != nil {
		h.writeReinforcementError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

// GetReinforcement 查询单个增援批次。
func (h *Handlers) GetReinforcement(w http.ResponseWriter, r *http.Request) {
	playerID := r.URL.Query().Get("playerId")
	if !h.requireOwnership(w, r, playerID) {
		return
	}
	record, err := h.gameService.GetReinforcement(playerID, r.PathValue("reinforcementId"))
	if err != nil {
		h.writeReinforcementError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"reinforcement": record})
}

// RecallReinforcement 召回我派出的增援。
func (h *Handlers) RecallReinforcement(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		PlayerID string `json:"playerId"`
	}
	if !decodeJSON(w, r, &payload) {
		return
	}
	if !h.requireOwnership(w, r, payload.PlayerID) {
		return
	}
	result, err := h.gameService.RecallReinforcement(payload.PlayerID, r.PathValue("reinforcementId"))
	if err != nil {
		h.writeReinforcementError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

// ExpelReinforcement 遣返我收到的增援。
func (h *Handlers) ExpelReinforcement(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		PlayerID string `json:"playerId"`
	}
	if !decodeJSON(w, r, &payload) {
		return
	}
	if !h.requireOwnership(w, r, payload.PlayerID) {
		return
	}
	result, err := h.gameService.ExpelReinforcement(payload.PlayerID, r.PathValue("reinforcementId"))
	if err != nil {
		h.writeReinforcementError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

// writeReinforcementError 写出增援系统统一错误。
func (h *Handlers) writeReinforcementError(w http.ResponseWriter, err error) {
	status := http.StatusBadRequest
	switch {
	case errors.Is(err, game.ErrPlayerNotFound), errors.Is(err, game.ErrReinforcementNotFound), errors.Is(err, game.ErrGeneralNotFound), errors.Is(err, game.ErrUnitNotFound):
		status = http.StatusNotFound
	case errors.Is(err, game.ErrInsufficientArmy), errors.Is(err, game.ErrReinforcementSlotFull), errors.Is(err, game.ErrGeneralBusy), errors.Is(err, game.ErrReinforcementBusy):
		status = http.StatusConflict
	case errors.Is(err, game.ErrNoUnitsSelected), errors.Is(err, game.ErrReinforcementTargetSelf), errors.Is(err, game.ErrReinforcementTargetNPC), errors.Is(err, game.ErrInvalidReinforcement), errors.Is(err, game.ErrNonCombatUnit):
		status = http.StatusBadRequest
	}
	writeError(w, status, err.Error())
}
