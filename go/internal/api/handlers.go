package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"hero3/internal/config"
	"hero3/internal/game"
)

type Handlers struct {
	cfg         config.Config
	gameService *game.Service
}

type accountRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type createPlayerRequest struct {
	AccountID string `json:"accountId"`
	Nickname  string `json:"nickname"`
	Faction   string `json:"faction"`
	GeneralID string `json:"generalId"`
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
	state, err := h.gameService.GetState(r.URL.Query().Get("playerId"))
	if err != nil {
		writeError(w, http.StatusNotFound, "player not found")
		return
	}
	writeJSON(w, http.StatusOK, state)
}

func (h *Handlers) RegisterAccount(w http.ResponseWriter, r *http.Request) {
	var payload accountRequest
	if !decodeJSON(w, r, &payload) {
		return
	}

	account, err := h.gameService.RegisterAccount(payload.Username, payload.Password)
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, game.ErrAccountExists) {
			status = http.StatusConflict
		}
		writeError(w, status, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, map[string]string{
		"accountId": account.ID,
		"username":  account.Username,
	})
}

func (h *Handlers) LoginAccount(w http.ResponseWriter, r *http.Request) {
	var payload accountRequest
	if !decodeJSON(w, r, &payload) {
		return
	}

	account, err := h.gameService.LoginAccount(payload.Username, payload.Password)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "invalid username or password")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"accountId": account.ID,
		"username":  account.Username,
	})
}

func (h *Handlers) AccountPlayers(w http.ResponseWriter, r *http.Request) {
	players, err := h.gameService.ListPlayers(r.PathValue("accountId"))
	if err != nil {
		writeError(w, http.StatusNotFound, "account not found")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"players": players})
}

func (h *Handlers) DeleteAccount(w http.ResponseWriter, r *http.Request) {
	if err := h.gameService.DeleteAccount(r.PathValue("accountId")); err != nil {
		writeError(w, http.StatusNotFound, "account not found")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
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

func (h *Handlers) CreatePlayer(w http.ResponseWriter, r *http.Request) {
	var payload createPlayerRequest
	if !decodeJSON(w, r, &payload) {
		return
	}

	playerID, state, err := h.gameService.CreatePlayer(payload.AccountID, payload.Nickname, payload.Faction)
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, game.ErrAccountNotFound) {
			status = http.StatusNotFound
		}
		writeError(w, status, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"playerId": playerID,
		"state":    state,
	})
}

func (h *Handlers) DeletePlayer(w http.ResponseWriter, r *http.Request) {
	if err := h.gameService.DeletePlayer(r.PathValue("playerId")); err != nil {
		writeError(w, http.StatusNotFound, "player not found")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(payload); err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}
}

func decodeJSON(w http.ResponseWriter, r *http.Request, target any) bool {
	if err := json.NewDecoder(r.Body).Decode(target); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json body")
		return false
	}
	return true
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
