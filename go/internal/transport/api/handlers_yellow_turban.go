// 本文件提供黄巾起义玩家侧和 GM 侧 HTTP 接口。
package api

import (
	"errors"
	"net/http"

	"hero3/internal/app/game"
)

func (h *Handlers) YellowTurbanStatus(w http.ResponseWriter, r *http.Request) {
	playerID := r.URL.Query().Get("playerId")
	if playerID == "" {
		writeError(w, http.StatusBadRequest, "playerId is required")
		return
	}
	if !h.requireOwnership(w, r, playerID) {
		return
	}
	result, err := h.gameService.GetYellowTurbanStatus(playerID)
	if err != nil {
		writeYellowTurbanError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *Handlers) CheckYellowTurban(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		PlayerID string `json:"playerId"`
	}
	if !decodeJSON(w, r, &payload) {
		return
	}
	if !h.requireOwnership(w, r, payload.PlayerID) {
		return
	}
	result, err := h.gameService.CheckYellowTurbanForPlayer(payload.PlayerID)
	if err != nil {
		writeYellowTurbanError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *Handlers) ResolveYellowTurbanMarch(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		PlayerID string `json:"playerId"`
	}
	if !decodeJSON(w, r, &payload) {
		return
	}
	if !h.requireOwnership(w, r, payload.PlayerID) {
		return
	}
	marchID := r.PathValue("marchId")
	report, err := h.gameService.ResolveYellowTurbanMarchForPlayer(payload.PlayerID, marchID)
	if err != nil {
		writeYellowTurbanError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, report)
}

func (h *Handlers) AdminYellowTurbanConfig(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, h.gameService.GetYellowTurbanConfig())
}

func (h *Handlers) UpdateAdminYellowTurbanConfig(w http.ResponseWriter, r *http.Request) {
	var payload game.YellowTurbanConfig
	if !decodeJSON(w, r, &payload) {
		return
	}
	if err := h.gameService.UpdateYellowTurbanConfig(payload); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, h.gameService.GetYellowTurbanConfig())
}

func (h *Handlers) AdminCheckYellowTurbanAll(w http.ResponseWriter, r *http.Request) {
	results, err := h.gameService.CheckYellowTurbanForAllPlayers()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"results": results})
}

func writeYellowTurbanError(w http.ResponseWriter, err error) {
	status := http.StatusBadRequest
	switch {
	case errors.Is(err, game.ErrPlayerNotFound), errors.Is(err, game.ErrYellowTurbanMarchNotFound):
		status = http.StatusNotFound
	case errors.Is(err, game.ErrPvpMarchNotReady):
		status = http.StatusConflict
	case errors.Is(err, game.ErrInsufficientArmy), errors.Is(err, game.ErrNoUnitsSelected):
		status = http.StatusUnprocessableEntity
	}
	writeError(w, status, err.Error())
}
