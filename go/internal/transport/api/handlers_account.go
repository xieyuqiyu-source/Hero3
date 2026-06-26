package api

import (
	"errors"
	"hero3/internal/app/game"
	"hero3/internal/platform/auth"
	"net/http"
)

// RegisterAccount 处理玩家轻账号注册。
func (h *Handlers) RegisterAccount(w http.ResponseWriter, r *http.Request) {
	var payload accountRequest
	if !decodeJSON(w, r, &payload) {
		return
	}
	if h.cfg.JWTSecret == "" {
		writeError(w, http.StatusInternalServerError, "authentication is not configured")
		return
	}

	account, err := h.gameService.RegisterAccount(payload.Username, payload.Password)
	if err != nil {
		status := http.StatusInternalServerError
		message := "account service unavailable"
		switch {
		case errors.Is(err, game.ErrInvalidCredentials):
			status = http.StatusBadRequest
			message = err.Error()
		case errors.Is(err, game.ErrAccountExists):
			status = http.StatusConflict
			message = err.Error()
		}
		writeError(w, status, message)
		return
	}

	token, err := auth.IssueToken(auth.Config{
		JWTSecret: h.cfg.JWTSecret,
		TokenTTL:  h.cfg.TokenTTL,
	}, account.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "authentication is not configured")
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"accountId": account.ID,
		"username":  account.Username,
		"gold":      account.Gold,
		"token":     token,
	})
}

// LoginAccount 处理玩家轻账号登录并签发 JWT。
func (h *Handlers) LoginAccount(w http.ResponseWriter, r *http.Request) {
	var payload accountRequest
	if !decodeJSON(w, r, &payload) {
		return
	}
	if h.cfg.JWTSecret == "" {
		writeError(w, http.StatusInternalServerError, "authentication is not configured")
		return
	}

	account, err := h.gameService.LoginAccount(payload.Username, payload.Password)
	if err != nil {
		if errors.Is(err, game.ErrInvalidCredentials) {
			writeError(w, http.StatusUnauthorized, "invalid username or password")
			return
		}
		writeError(w, http.StatusInternalServerError, "account service unavailable")
		return
	}

	token, err := auth.IssueToken(auth.Config{
		JWTSecret: h.cfg.JWTSecret,
		TokenTTL:  h.cfg.TokenTTL,
	}, account.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "authentication is not configured")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"accountId": account.ID,
		"username":  account.Username,
		"gold":      account.Gold,
		"token":     token,
	})
}

func (h *Handlers) AccountPlayers(w http.ResponseWriter, r *http.Request) {
	accountID := r.PathValue("accountId")
	if !h.requireAccount(w, r, accountID) {
		return
	}
	players, err := h.gameService.ListPlayers(accountID)
	if err != nil {
		writeError(w, http.StatusNotFound, "account not found")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"players": players})
}

func (h *Handlers) AccountInfo(w http.ResponseWriter, r *http.Request) {
	accountID := r.PathValue("accountId")
	if !h.requireAccount(w, r, accountID) {
		return
	}
	account, err := h.gameService.GetAccountByID(accountID)
	if err != nil {
		writeError(w, http.StatusNotFound, "account not found")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"accountId": account.ID,
		"username":  account.Username,
		"gold":      account.Gold,
	})
}

func (h *Handlers) DeleteAccount(w http.ResponseWriter, r *http.Request) {
	accountID := r.PathValue("accountId")
	if !h.requireAccount(w, r, accountID) {
		return
	}
	if err := h.gameService.DeleteAccount(accountID); err != nil {
		writeError(w, http.StatusNotFound, "account not found")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (h *Handlers) CreatePlayer(w http.ResponseWriter, r *http.Request) {
	var payload createPlayerRequest
	if !decodeJSON(w, r, &payload) {
		return
	}
	// 创建玩家时校验 accountId 归属
	if !h.requireAccount(w, r, payload.AccountID) {
		return
	}

	playerID, state, err := h.gameService.CreatePlayer(payload.AccountID, payload.Nickname, payload.Faction, payload.GeneralID)
	if err != nil {
		status := http.StatusBadRequest
		switch {
		case errors.Is(err, game.ErrAccountNotFound):
			status = http.StatusNotFound
		case errors.Is(err, game.ErrInvalidGeneral):
			status = http.StatusUnprocessableEntity
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
	playerID := r.PathValue("playerId")
	if !h.requireOwnership(w, r, playerID) {
		return
	}
	if err := h.gameService.DeletePlayer(playerID); err != nil {
		writeError(w, http.StatusNotFound, "player not found")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}
