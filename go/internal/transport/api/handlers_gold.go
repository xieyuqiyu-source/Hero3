package api

import (
	"errors"
	"hero3/internal/app/game"
	"net/http"
	"strconv"
	"time"
)

func (h *Handlers) AddAccountGold(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		AccountID string `json:"accountId"`
		Amount    int    `json:"amount"`
	}
	if !decodeJSON(w, r, &payload) {
		return
	}
	if payload.Amount <= 0 {
		writeError(w, http.StatusBadRequest, "amount must be positive")
		return
	}

	if err := h.gameService.AddAccountGoldAdmin(payload.AccountID, payload.Amount); err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	account, _ := h.gameService.GetAccountByID(payload.AccountID)
	writeJSON(w, http.StatusOK, map[string]any{"gold": account.Gold})
}

func (h *Handlers) AdminGoldLedger(w http.ResponseWriter, r *http.Request) {
	limit := 200
	if raw := r.URL.Query().Get("limit"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil {
			limit = parsed
		}
	}
	filter := game.GoldLedgerFilter{
		AccountID: r.URL.Query().Get("accountId"),
		PlayerID:  r.URL.Query().Get("playerId"),
		Currency:  r.URL.Query().Get("currency"),
		RefType:   r.URL.Query().Get("refType"),
		Limit:     limit,
	}
	if raw := r.URL.Query().Get("from"); raw != "" {
		if parsed, err := time.Parse(time.RFC3339, raw); err == nil {
			filter.From = parsed
		}
	}
	if raw := r.URL.Query().Get("to"); raw != "" {
		if parsed, err := time.Parse(time.RFC3339, raw); err == nil {
			filter.To = parsed
		}
	}
	entries, err := h.gameService.ListGoldLedger(filter)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"entries": entries})
}

func (h *Handlers) AddGold(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		PlayerID string `json:"playerId"`
		Amount   int    `json:"amount"`
		Reason   string `json:"reason"`
	}
	if !decodeJSON(w, r, &payload) {
		return
	}

	result, err := h.gameService.AddGold(payload.PlayerID, payload.Amount, payload.Reason)
	if err != nil {
		status := http.StatusBadRequest
		switch {
		case errors.Is(err, game.ErrPlayerNotFound):
			status = http.StatusNotFound
		case errors.Is(err, game.ErrInvalidGoldAmount):
			status = http.StatusUnprocessableEntity
		}
		writeError(w, status, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, result)
}

func (h *Handlers) DeductGold(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		PlayerID string `json:"playerId"`
		Amount   int    `json:"amount"`
		Reason   string `json:"reason"`
	}
	if !decodeJSON(w, r, &payload) {
		return
	}

	result, err := h.gameService.DeductGold(payload.PlayerID, payload.Amount, payload.Reason)
	if err != nil {
		status := http.StatusBadRequest
		switch {
		case errors.Is(err, game.ErrPlayerNotFound):
			status = http.StatusNotFound
		case errors.Is(err, game.ErrInsufficientCityGold):
			status = http.StatusUnprocessableEntity
		case errors.Is(err, game.ErrInvalidGoldAmount):
			status = http.StatusUnprocessableEntity
		}
		writeError(w, status, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, result)
}

func (h *Handlers) ExchangeGold(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		AccountID string `json:"accountId"`
		PlayerID  string `json:"playerId"`
		Amount    int    `json:"amount"` // 金币数量
	}
	if !decodeJSON(w, r, &payload) {
		return
	}

	// 校验账户归属和玩家归属
	if !h.requireAccount(w, r, payload.AccountID) {
		return
	}
	if !h.requireOwnership(w, r, payload.PlayerID) {
		return
	}

	result, err := h.gameService.ExchangeGoldToCityGold(payload.AccountID, payload.PlayerID, payload.Amount)
	if err != nil {
		status := http.StatusBadRequest
		switch {
		case errors.Is(err, game.ErrPlayerNotFound):
			status = http.StatusNotFound
		case errors.Is(err, game.ErrAccountNotFound):
			status = http.StatusNotFound
		case errors.Is(err, game.ErrInvalidGoldAmount):
			status = http.StatusUnprocessableEntity
		case errors.Is(err, game.ErrInsufficientGold):
			status = http.StatusUnprocessableEntity
		case errors.Is(err, game.ErrExchangeCooldown):
			status = http.StatusTooManyRequests
		}
		writeError(w, status, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, result)
}

func (h *Handlers) ReverseExchangeGold(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		AccountID      string `json:"accountId"`
		PlayerID       string `json:"playerId"`
		CityGoldAmount int    `json:"cityGoldAmount"` // 要消耗的城金数量
	}
	if !decodeJSON(w, r, &payload) {
		return
	}

	// 校验账户归属和玩家归属
	if !h.requireAccount(w, r, payload.AccountID) {
		return
	}
	if !h.requireOwnership(w, r, payload.PlayerID) {
		return
	}

	result, err := h.gameService.ExchangeCityGoldToGold(payload.AccountID, payload.PlayerID, payload.CityGoldAmount)
	if err != nil {
		status := http.StatusBadRequest
		switch {
		case errors.Is(err, game.ErrPlayerNotFound):
			status = http.StatusNotFound
		case errors.Is(err, game.ErrAccountNotFound):
			status = http.StatusNotFound
		case errors.Is(err, game.ErrInvalidGoldAmount):
			status = http.StatusUnprocessableEntity
		case errors.Is(err, game.ErrInsufficientGold):
			status = http.StatusUnprocessableEntity
		case errors.Is(err, game.ErrInsufficientCityGold):
			status = http.StatusUnprocessableEntity
		case errors.Is(err, game.ErrExchangeCooldown):
			status = http.StatusTooManyRequests
		}
		writeError(w, status, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, result)
}

// --- Buff 管理 ---
