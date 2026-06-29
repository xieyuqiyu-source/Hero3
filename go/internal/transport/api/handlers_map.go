// 本文件提供地图和 NPC 城池相关 HTTP 接口处理器。
package api

import (
	"errors"
	"hero3/internal/app/game"
	"net/http"
)

func (h *Handlers) NpcCities(w http.ResponseWriter, r *http.Request) {
	playerID := r.URL.Query().Get("playerId")
	if playerID == "" {
		writeError(w, http.StatusBadRequest, "playerId is required")
		return
	}
	if !h.requireOwnership(w, r, playerID) {
		return
	}
	npcState, err := h.gameService.GetNpcCities(playerID)
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, game.ErrPlayerNotFound) {
			status = http.StatusNotFound
		}
		writeError(w, status, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, npcState)
}

func (h *Handlers) RefreshNpcCities(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		PlayerID string `json:"playerId"`
	}
	if !decodeJSON(w, r, &payload) {
		return
	}
	if !h.requireOwnership(w, r, payload.PlayerID) {
		return
	}

	npcState, err := h.gameService.RefreshNpcCities(payload.PlayerID)
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, game.ErrPlayerNotFound) {
			status = http.StatusNotFound
		}
		writeError(w, status, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, npcState)
}

func (h *Handlers) AttackNpc(w http.ResponseWriter, r *http.Request) {
	var payload game.AttackNpcRequest
	if !decodeJSON(w, r, &payload) {
		return
	}
	if !h.requireOwnership(w, r, payload.PlayerID) {
		return
	}

	result, err := h.gameService.AttackNpc(payload)
	if err != nil {
		status := http.StatusBadRequest
		switch {
		case errors.Is(err, game.ErrPlayerNotFound):
			status = http.StatusNotFound
		case errors.Is(err, game.ErrNpcNotFound):
			status = http.StatusNotFound
		case errors.Is(err, game.ErrNoUnitsSelected):
			status = http.StatusBadRequest
		case errors.Is(err, game.ErrNonCombatUnit):
			status = http.StatusBadRequest
		case errors.Is(err, game.ErrInsufficientArmy):
			status = http.StatusUnprocessableEntity
		case errors.Is(err, game.ErrGeneralNotFound), errors.Is(err, game.ErrGeneralBusy):
			status = http.StatusBadRequest
		}
		writeError(w, status, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, result)
}

func (h *Handlers) SimulateBattle(w http.ResponseWriter, r *http.Request) {
	var payload game.BattleSimulationRequest
	if !decodeJSON(w, r, &payload) {
		return
	}
	if !h.requireOwnership(w, r, payload.PlayerID) {
		return
	}

	result, err := h.gameService.SimulateBattle(payload)
	if err != nil {
		status := http.StatusBadRequest
		switch {
		case errors.Is(err, game.ErrPlayerNotFound):
			status = http.StatusNotFound
		case errors.Is(err, game.ErrNoUnitsSelected):
			status = http.StatusBadRequest
		case errors.Is(err, game.ErrNonCombatUnit):
			status = http.StatusBadRequest
		case errors.Is(err, game.ErrUnitNotFound):
			status = http.StatusNotFound
		}
		writeError(w, status, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, result)
}

func (h *Handlers) ScoutNpc(w http.ResponseWriter, r *http.Request) {
	var payload game.ScoutNpcRequest
	if !decodeJSON(w, r, &payload) {
		return
	}
	if !h.requireOwnership(w, r, payload.PlayerID) {
		return
	}

	result, err := h.gameService.ScoutNpc(payload)
	if err != nil {
		status := http.StatusBadRequest
		switch {
		case errors.Is(err, game.ErrPlayerNotFound):
			status = http.StatusNotFound
		case errors.Is(err, game.ErrNpcNotFound):
			status = http.StatusNotFound
		case errors.Is(err, game.ErrInsufficientArmy):
			status = http.StatusUnprocessableEntity
		}
		writeError(w, status, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, result)
}
