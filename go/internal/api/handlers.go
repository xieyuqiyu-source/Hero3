package api

import (
	"encoding/json"
	"net/http"
	"time"

	"hero3/internal/config"
	"hero3/internal/game"
)

type Handlers struct {
	cfg         config.Config
	gameService *game.Service
}

func NewHandlers(cfg config.Config, gameService *game.Service) *Handlers {
	return &Handlers{
		cfg:         cfg,
		gameService: gameService,
	}
}

func (h *Handlers) Health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"status":      "ok",
		"service":     h.cfg.ServiceName,
		"version":     h.cfg.Version,
		"environment": h.cfg.Environment,
		"time":        time.Now().UTC().Format(time.RFC3339),
	})
}

func (h *Handlers) Meta(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"name":       "Hero3",
		"apiVersion": "v1",
		"service":    h.cfg.ServiceName,
		"version":    h.cfg.Version,
	})
}

func (h *Handlers) GameBootstrap(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, h.gameService.Bootstrap())
}

func (h *Handlers) GameState(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, h.gameService.GetState())
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(payload); err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}
}
