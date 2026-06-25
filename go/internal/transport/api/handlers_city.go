package api

import (
	"errors"
	"fmt"
	"hero3/internal/app/game"
	"net/http"
)

func (h *Handlers) FillResources(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		PlayerID string `json:"playerId"`
	}
	if !decodeJSON(w, r, &payload) {
		return
	}
	if !h.requireOwnership(w, r, payload.PlayerID) {
		return
	}

	state, err := h.gameService.FillResources(payload.PlayerID)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"state": state})
}

func (h *Handlers) FillResourcesPaid(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		PlayerID string `json:"playerId"`
	}
	if !decodeJSON(w, r, &payload) {
		return
	}
	if !h.requireOwnership(w, r, payload.PlayerID) {
		return
	}

	state, cost, err := h.gameService.FillResourcesPaid(payload.PlayerID)
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

	writeJSON(w, http.StatusOK, map[string]any{"state": state, "cost": cost})
}

func (h *Handlers) UpgradeBuilding(w http.ResponseWriter, r *http.Request) {
	var payload upgradeBuildingRequest
	if !decodeJSON(w, r, &payload) {
		return
	}
	if !h.requireOwnership(w, r, payload.PlayerID) {
		return
	}

	state, err := h.gameService.UpgradeBuilding(payload.PlayerID, payload.BuildingID)
	if err != nil {
		status := http.StatusBadRequest
		switch {
		case errors.Is(err, game.ErrPlayerNotFound):
			status = http.StatusNotFound
		case errors.Is(err, game.ErrBuildingNotFound):
			status = http.StatusNotFound
		case errors.Is(err, game.ErrInsufficientRes):
			status = http.StatusUnprocessableEntity
		case errors.Is(err, game.ErrAlreadyUpgrading):
			status = http.StatusConflict
		case errors.Is(err, game.ErrMaxLevel):
			status = http.StatusUnprocessableEntity
		}
		writeError(w, status, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"state": state})
}

// --- Boost Handlers ---

func (h *Handlers) InstantCompleteBuilding(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		PlayerID   string `json:"playerId"`
		BuildingID string `json:"buildingId"`
	}
	if !decodeJSON(w, r, &payload) {
		return
	}
	if !h.requireOwnership(w, r, payload.PlayerID) {
		return
	}

	state, err := h.gameService.InstantCompleteBuilding(payload.PlayerID, payload.BuildingID)
	if err != nil {
		status := http.StatusBadRequest
		switch {
		case errors.Is(err, game.ErrPlayerNotFound):
			status = http.StatusNotFound
		case errors.Is(err, game.ErrBuildingNotFound):
			status = http.StatusNotFound
		case errors.Is(err, game.ErrNotUpgrading):
			status = http.StatusConflict
		case errors.Is(err, game.ErrInsufficientCityGold):
			status = http.StatusUnprocessableEntity
		}
		writeError(w, status, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"state": state})
}

func (h *Handlers) PurchaseBoost(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		PlayerID   string `json:"playerId"`
		Multiplier int    `json:"multiplier"` // 2, 4, 8, 16
		Hours      int    `json:"hours"`      // 1, 6, 12, 24
	}
	if !decodeJSON(w, r, &payload) {
		return
	}
	if !h.requireOwnership(w, r, payload.PlayerID) {
		return
	}

	state, err := h.gameService.PurchaseBoost(payload.PlayerID, payload.Multiplier, payload.Hours)
	if err != nil {
		status := http.StatusBadRequest
		switch {
		case errors.Is(err, game.ErrPlayerNotFound):
			status = http.StatusNotFound
		case errors.Is(err, game.ErrInsufficientCityGold):
			status = http.StatusUnprocessableEntity
		case errors.Is(err, game.ErrBoostActive):
			status = http.StatusConflict
		case errors.Is(err, game.ErrInvalidBoost):
			status = http.StatusBadRequest
		case errors.Is(err, game.ErrInvalidDuration):
			status = http.StatusBadRequest
		}
		writeError(w, status, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"state": state})
}

func (h *Handlers) BoostPrices(w http.ResponseWriter, r *http.Request) {
	multipliers := []int{2, 4, 8, 16}
	hours := []int{1, 6, 12, 24}

	prices := map[string]int{}
	for _, m := range multipliers {
		for _, h := range hours {
			key := fmt.Sprintf("%dx_%dh", m, h)
			prices[key] = game.GetBoostCost(m, h)
		}
	}

	writeJSON(w, http.StatusOK, prices)
}

func (h *Handlers) PurchaseCapacityBoost(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		PlayerID   string `json:"playerId"`
		Multiplier int    `json:"multiplier"`
		Hours      int    `json:"hours"`
	}
	if !decodeJSON(w, r, &payload) {
		return
	}
	if !h.requireOwnership(w, r, payload.PlayerID) {
		return
	}

	state, err := h.gameService.PurchaseCapacityBoost(payload.PlayerID, payload.Multiplier, payload.Hours)
	if err != nil {
		status := http.StatusBadRequest
		switch {
		case errors.Is(err, game.ErrPlayerNotFound):
			status = http.StatusNotFound
		case errors.Is(err, game.ErrInsufficientCityGold):
			status = http.StatusUnprocessableEntity
		case errors.Is(err, game.ErrBoostActive):
			status = http.StatusConflict
		case errors.Is(err, game.ErrInvalidBoost):
			status = http.StatusBadRequest
		case errors.Is(err, game.ErrInvalidDuration):
			status = http.StatusBadRequest
		}
		writeError(w, status, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"state": state})
}

func (h *Handlers) UpgradeBuildingBatch(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		PlayerID string `json:"playerId"`
	}
	if !decodeJSON(w, r, &payload) {
		return
	}
	if !h.requireOwnership(w, r, payload.PlayerID) {
		return
	}

	state, upgraded, err := h.gameService.UpgradeBuildingBatch(payload.PlayerID)
	if err != nil {
		status := http.StatusBadRequest
		switch {
		case errors.Is(err, game.ErrPlayerNotFound):
			status = http.StatusNotFound
		case errors.Is(err, game.ErrInsufficientRes):
			status = http.StatusUnprocessableEntity
		}
		writeError(w, status, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"state": state, "upgraded": upgraded})
}
