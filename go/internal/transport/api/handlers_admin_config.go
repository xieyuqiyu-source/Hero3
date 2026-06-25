package api

import (
	"errors"
	"hero3/internal/app/game"
	"hero3/internal/core/combat"
	"hero3/internal/core/general"
	"net/http"
)

func (h *Handlers) AdminPlayerState(w http.ResponseWriter, r *http.Request) {
	playerID := r.PathValue("playerId")
	state, err := h.gameService.GetState(playerID)
	if err != nil {
		writeError(w, http.StatusNotFound, "player not found")
		return
	}
	writeJSON(w, http.StatusOK, state)
}

func (h *Handlers) AdminAdjustResources(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		PlayerID    string         `json:"playerId"`
		Adjustments map[string]int `json:"adjustments"`
	}
	if !decodeJSON(w, r, &payload) {
		return
	}

	if payload.PlayerID == "" || len(payload.Adjustments) == 0 {
		writeError(w, http.StatusBadRequest, "playerId and adjustments are required")
		return
	}

	state, err := h.gameService.AdjustResources(payload.PlayerID, payload.Adjustments)
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, game.ErrPlayerNotFound) {
			status = http.StatusNotFound
		}
		writeError(w, status, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"state": state})
}

func (h *Handlers) AdminAccounts(w http.ResponseWriter, r *http.Request) {
	accounts, err := h.gameService.ListAccounts()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "accounts load failed")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"accounts": accounts})
}

func (h *Handlers) AdminBalance(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, h.gameService.GetBalance())
}

func (h *Handlers) UpdateAdminBalance(w http.ResponseWriter, r *http.Request) {
	var payload game.BalanceConfig
	if !decodeJSON(w, r, &payload) {
		return
	}

	if err := h.gameService.UpdateBalance(payload); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, h.gameService.GetBalance())
}

func (h *Handlers) AdminNpcConfig(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, h.gameService.GetNpcConfig())
}

func (h *Handlers) UpdateAdminNpcConfig(w http.ResponseWriter, r *http.Request) {
	var payload game.NpcConfig
	if !decodeJSON(w, r, &payload) {
		return
	}

	if err := h.gameService.UpdateNpcConfig(payload); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, h.gameService.GetNpcConfig())
}

func (h *Handlers) AdminCombatConfig(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, combat.GetCombatConfig())
}

func (h *Handlers) UpdateAdminCombatConfig(w http.ResponseWriter, r *http.Request) {
	var payload combat.CombatConfig
	if !decodeJSON(w, r, &payload) {
		return
	}

	if err := h.gameService.UpdateCombatConfig(payload); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, combat.GetCombatConfig())
}

func (h *Handlers) AdminFishingConfig(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, h.gameService.GetFishingConfig())
}

func (h *Handlers) UpdateAdminFishingConfig(w http.ResponseWriter, r *http.Request) {
	var payload game.FishingConfig
	if !decodeJSON(w, r, &payload) {
		return
	}

	if err := h.gameService.UpdateFishingConfig(payload); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, h.gameService.GetFishingConfig())
}

func (h *Handlers) AdminFactionsConfig(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, game.GetFactionsConfig())
}

func (h *Handlers) UpdateAdminFactionsConfig(w http.ResponseWriter, r *http.Request) {
	var payload game.FactionsConfig
	if !decodeJSON(w, r, &payload) {
		return
	}

	if err := h.gameService.UpdateFactionsConfig(payload); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, game.GetFactionsConfig())
}

func (h *Handlers) AdminUnitsConfig(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, game.GetUnitsConfig())
}

func (h *Handlers) AdminFactionUnitsConfig(w http.ResponseWriter, r *http.Request) {
	faction := r.PathValue("faction")
	units := game.GetFactionUnits(faction)
	if units == nil {
		writeError(w, http.StatusNotFound, "faction not found")
		return
	}
	writeJSON(w, http.StatusOK, units)
}

func (h *Handlers) UpdateAdminFactionUnitsConfig(w http.ResponseWriter, r *http.Request) {
	faction := r.PathValue("faction")
	var payload game.FactionUnits
	if !decodeJSON(w, r, &payload) {
		return
	}

	if err := h.gameService.UpdateFactionUnits(faction, payload); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, game.GetFactionUnits(faction))
}

func (h *Handlers) AdminGeneralsConfig(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, h.gameService.GetGeneralsConfig())
}

func (h *Handlers) UpdateAdminGeneralsConfig(w http.ResponseWriter, r *http.Request) {
	var payload game.GeneralsConfig
	if !decodeJSON(w, r, &payload) {
		return
	}
	if err := h.gameService.UpdateGeneralsConfig(payload); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, h.gameService.GetGeneralsConfig())
}

// AdminGeneralTraitRegistry 返回所有已注册的特性元信息（GM 后台用来选择特性）

func (h *Handlers) AdminGeneralTraitRegistry(w http.ResponseWriter, r *http.Request) {
	type traitMeta struct {
		ID          string               `json:"id"`
		Name        string               `json:"name"`
		Description string               `json:"description"`
		ParamSchema []general.ParamField `json:"paramSchema"`
	}
	traits := general.All()
	out := make([]traitMeta, 0, len(traits))
	for _, t := range traits {
		out = append(out, traitMeta{
			ID:          t.ID(),
			Name:        t.Name(),
			Description: t.Description(general.Params{}),
			ParamSchema: t.ParamSchema(),
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"traits": out})
}
