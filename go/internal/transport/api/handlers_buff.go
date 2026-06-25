package api

import (
	"net/http"
)

func (h *Handlers) GrantBuff(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		PlayerID string  `json:"playerId"`
		Key      string  `json:"key"`
		Value    float64 `json:"value"`
		Mode     string  `json:"mode"`
		Hours    int     `json:"hours"` // 0 = 永久
		Note     string  `json:"note"`
	}
	if !decodeJSON(w, r, &payload) {
		return
	}

	if payload.PlayerID == "" || payload.Key == "" {
		writeError(w, http.StatusBadRequest, "playerId and key are required")
		return
	}
	if payload.Mode == "" {
		payload.Mode = "percentAdd"
	}

	state, err := h.gameService.GrantBuff(payload.PlayerID, payload.Key, payload.Value, payload.Mode, payload.Hours, payload.Note)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"state": state})
}

func (h *Handlers) RevokeBuff(w http.ResponseWriter, r *http.Request) {
	buffId := r.PathValue("buffId")
	playerID := r.URL.Query().Get("playerId")

	if buffId == "" || playerID == "" {
		writeError(w, http.StatusBadRequest, "buffId and playerId are required")
		return
	}

	state, err := h.gameService.RevokeBuff(playerID, buffId)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"state": state})
}

// --- MiniGame Records ---
