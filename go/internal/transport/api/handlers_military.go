package api

import (
	"errors"
	"hero3/internal/app/game"
	"net/http"
)

func (h *Handlers) Recruit(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		PlayerID string `json:"playerId"`
		UnitID   string `json:"unitId"`
		Amount   int    `json:"amount"`
	}
	if !decodeJSON(w, r, &payload) {
		return
	}
	if !h.requireOwnership(w, r, payload.PlayerID) {
		return
	}

	state, err := h.gameService.Recruit(payload.PlayerID, payload.UnitID, payload.Amount)
	if err != nil {
		status := http.StatusBadRequest
		switch {
		case errors.Is(err, game.ErrPlayerNotFound):
			status = http.StatusNotFound
		case errors.Is(err, game.ErrUnitNotFound):
			status = http.StatusNotFound
		case errors.Is(err, game.ErrInsufficientRes):
			status = http.StatusUnprocessableEntity
		case errors.Is(err, game.ErrInvalidAmount):
			status = http.StatusBadRequest
		case errors.Is(err, game.ErrQueueFull):
			status = http.StatusConflict
		}
		writeError(w, status, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, game.BuildMilitaryActionResult(state))
}

func (h *Handlers) InstantCompleteRecruit(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		PlayerID string `json:"playerId"`
		QueueID  string `json:"queueId"`
	}
	if !decodeJSON(w, r, &payload) {
		return
	}
	if !h.requireOwnership(w, r, payload.PlayerID) {
		return
	}

	state, err := h.gameService.InstantCompleteRecruit(payload.PlayerID, payload.QueueID)
	if err != nil {
		status := http.StatusBadRequest
		switch {
		case errors.Is(err, game.ErrPlayerNotFound):
			status = http.StatusNotFound
		case errors.Is(err, game.ErrInsufficientCityGold):
			status = http.StatusUnprocessableEntity
		}
		writeError(w, status, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, game.BuildMilitaryActionResult(state))
}

func (h *Handlers) AllocateGeneralStat(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		PlayerID string `json:"playerId"`
		StatKey  string `json:"statKey"`
	}
	if !decodeJSON(w, r, &payload) {
		return
	}
	if !h.requireOwnership(w, r, payload.PlayerID) {
		return
	}

	state, err := h.gameService.AllocateGeneralStat(payload.PlayerID, payload.StatKey)
	if err != nil {
		status := http.StatusBadRequest
		switch {
		case errors.Is(err, game.ErrPlayerNotFound), errors.Is(err, game.ErrGeneralNotFound):
			status = http.StatusNotFound
		case errors.Is(err, game.ErrNoStatPoints), errors.Is(err, game.ErrStatMaxLevel):
			status = http.StatusConflict
		case errors.Is(err, game.ErrInvalidStatKey):
			status = http.StatusBadRequest
		}
		writeError(w, status, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, game.BuildGeneralViewActionResult(state, 0))
}

func (h *Handlers) ResetGeneralStats(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		PlayerID string `json:"playerId"`
	}
	if !decodeJSON(w, r, &payload) {
		return
	}
	if !h.requireOwnership(w, r, payload.PlayerID) {
		return
	}

	result, err := h.gameService.ResetGeneralStats(payload.PlayerID)
	if err != nil {
		status := http.StatusBadRequest
		switch {
		case errors.Is(err, game.ErrPlayerNotFound), errors.Is(err, game.ErrGeneralNotFound), errors.Is(err, game.ErrAccountNotFound):
			status = http.StatusNotFound
		case errors.Is(err, game.ErrInsufficientGold):
			status = http.StatusUnprocessableEntity
		case errors.Is(err, game.ErrInvalidGoldAmount):
			status = http.StatusBadRequest
		}
		writeError(w, status, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, game.BuildGeneralViewActionResult(result.State, result.AccountGold))
}

func (h *Handlers) ChangeGeneral(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		PlayerID  string `json:"playerId"`
		GeneralID string `json:"generalId"`
		ItemID    string `json:"itemId,omitempty"`
	}
	if !decodeJSON(w, r, &payload) {
		return
	}
	if !h.requireOwnership(w, r, payload.PlayerID) {
		return
	}

	result, err := h.gameService.ChangeGeneral(payload.PlayerID, payload.GeneralID, payload.ItemID)
	if err != nil {
		status := http.StatusBadRequest
		switch {
		case errors.Is(err, game.ErrPlayerNotFound), errors.Is(err, game.ErrGeneralNotFound), errors.Is(err, game.ErrItemNotFound):
			status = http.StatusNotFound
		case errors.Is(err, game.ErrInvalidGeneral), errors.Is(err, game.ErrInsufficientItem):
			status = http.StatusUnprocessableEntity
		}
		writeError(w, status, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, game.BuildGeneralViewActionResult(result.State, result.AccountGold))
}
