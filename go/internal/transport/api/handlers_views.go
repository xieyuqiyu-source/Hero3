// 本文件归口玩家局部视图查询接口。
package api

import (
	"errors"
	"net/http"

	"hero3/internal/app/game"
)

// GameSummary 返回玩家摘要视图。
func (h *Handlers) GameSummary(w http.ResponseWriter, r *http.Request) {
	h.writePlayerView(w, r, func(playerID string) (any, error) {
		return h.gameService.GetPlayerSummaryView(playerID)
	})
}

// CityView 返回城池、建筑和资源田视图。
func (h *Handlers) CityView(w http.ResponseWriter, r *http.Request) {
	h.writePlayerView(w, r, func(playerID string) (any, error) {
		return h.gameService.GetCityView(playerID)
	})
}

// ResourceView 返回资源栏和产量视图。
func (h *Handlers) ResourceView(w http.ResponseWriter, r *http.Request) {
	h.writePlayerView(w, r, func(playerID string) (any, error) {
		return h.gameService.GetResourceView(playerID)
	})
}

// MilitaryView 返回军事视图。
func (h *Handlers) MilitaryView(w http.ResponseWriter, r *http.Request) {
	h.writePlayerView(w, r, func(playerID string) (any, error) {
		return h.gameService.GetMilitaryView(playerID)
	})
}

// InventoryView 返回背包视图。
func (h *Handlers) InventoryView(w http.ResponseWriter, r *http.Request) {
	h.writePlayerView(w, r, func(playerID string) (any, error) {
		return h.gameService.GetInventoryView(playerID)
	})
}

// GeneralsView 返回武将和派驻视图。
func (h *Handlers) GeneralsView(w http.ResponseWriter, r *http.Request) {
	h.writePlayerView(w, r, func(playerID string) (any, error) {
		return h.gameService.GetGeneralsView(playerID)
	})
}

// writePlayerView 校验 playerId 和归属后写出局部视图。
func (h *Handlers) writePlayerView(w http.ResponseWriter, r *http.Request, load func(playerID string) (any, error)) {
	playerID := r.URL.Query().Get("playerId")
	if playerID == "" {
		writeError(w, http.StatusBadRequest, "playerId is required")
		return
	}
	if !h.requireOwnership(w, r, playerID) {
		return
	}
	view, err := load(playerID)
	if err != nil {
		if errors.Is(err, game.ErrPlayerNotFound) {
			writeError(w, http.StatusNotFound, "player not found")
		} else {
			writeError(w, http.StatusInternalServerError, err.Error())
		}
		return
	}
	writeJSON(w, http.StatusOK, view)
}
