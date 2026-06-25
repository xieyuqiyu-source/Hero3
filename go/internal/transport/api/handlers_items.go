package api

import (
	"errors"
	"hero3/internal/app/game"
	"net/http"
)

func (h *Handlers) ItemsConfig(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, h.gameService.ListItemsConfig())
}

func (h *Handlers) UseItem(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		PlayerID string `json:"playerId"`
		ItemID   string `json:"itemId"`
		Amount   int    `json:"amount"`
	}
	if !decodeJSON(w, r, &payload) {
		return
	}
	if !h.requireOwnership(w, r, payload.PlayerID) {
		return
	}

	result, err := h.gameService.UseItem(payload.PlayerID, payload.ItemID, payload.Amount)
	if err != nil {
		status := http.StatusBadRequest
		switch {
		case errors.Is(err, game.ErrPlayerNotFound), errors.Is(err, game.ErrItemNotFound), errors.Is(err, game.ErrGeneralNotFound), errors.Is(err, game.ErrUnitNotFound):
			status = http.StatusNotFound
		case errors.Is(err, game.ErrInsufficientItem), errors.Is(err, game.ErrItemNotUsable):
			status = http.StatusUnprocessableEntity
		}
		writeError(w, status, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, result)
}

func (h *Handlers) AdminGrantItem(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		PlayerID string `json:"playerId"`
		ItemID   string `json:"itemId"`
		Amount   int    `json:"amount"`
	}
	if !decodeJSON(w, r, &payload) {
		return
	}

	state, err := h.gameService.GrantItem(payload.PlayerID, payload.ItemID, payload.Amount)
	if err != nil {
		status := http.StatusBadRequest
		switch {
		case errors.Is(err, game.ErrPlayerNotFound), errors.Is(err, game.ErrItemNotFound):
			status = http.StatusNotFound
		case errors.Is(err, game.ErrInvalidAmount):
			status = http.StatusUnprocessableEntity
		}
		writeError(w, status, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"state": state})
}
