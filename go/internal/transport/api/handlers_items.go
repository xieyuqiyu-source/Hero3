package api

import (
	"errors"
	"hero3/internal/app/game"
	"net/http"
	"strconv"
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

// AdminItemsConfig 返回 GM 后台物品配置。
func (h *Handlers) AdminItemsConfig(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, h.gameService.ListItemsConfig())
}

// UpdateAdminItemsConfig 保存 GM 后台物品配置。
func (h *Handlers) UpdateAdminItemsConfig(w http.ResponseWriter, r *http.Request) {
	var payload game.ItemsConfig
	if !decodeJSON(w, r, &payload) {
		return
	}
	if err := h.gameService.UpdateItemsConfig(payload); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, h.gameService.ListItemsConfig())
}

// ValidateAdminItemsConfig 校验 GM 后台物品配置但不保存。
func (h *Handlers) ValidateAdminItemsConfig(w http.ResponseWriter, r *http.Request) {
	var payload game.ItemsConfig
	if !decodeJSON(w, r, &payload) {
		return
	}
	if err := h.gameService.ValidateItemsConfigForAdmin(payload); err != nil {
		writeJSON(w, http.StatusOK, map[string]any{"ok": false, "error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// AdminInventoryView 查看指定玩家背包格子。
func (h *Handlers) AdminInventoryView(w http.ResponseWriter, r *http.Request) {
	playerID := r.URL.Query().Get("playerId")
	view, err := h.gameService.GetInventoryView(playerID)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, view)
}

// AdminItemLedger 查询物品流水。
func (h *Handlers) AdminItemLedger(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	limit, _ := strconv.Atoi(query.Get("limit"))
	offset, _ := strconv.Atoi(query.Get("offset"))
	page, err := h.gameService.ListItemLedger(game.ItemLedgerFilter{
		PlayerID: query.Get("playerId"),
		ItemID:   query.Get("itemId"),
		RefType:  query.Get("refType"),
		From:     query.Get("from"),
		To:       query.Get("to"),
		Limit:    limit,
		Offset:   offset,
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, page)
}
