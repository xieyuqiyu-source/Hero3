package api

import (
	"errors"
	"hero3/internal/app/game"
	"net/http"
)

func (h *Handlers) ListMails(w http.ResponseWriter, r *http.Request) {
	playerID := r.URL.Query().Get("playerId")
	if playerID == "" {
		writeError(w, http.StatusBadRequest, "playerId is required")
		return
	}
	if !h.requireOwnership(w, r, playerID) {
		return
	}

	page, pageSize, ok := parsePageQuery(w, r)
	if !ok {
		return
	}
	result, err := h.gameService.ListMails(playerID, page, pageSize, r.URL.Query().Get("mailType"))
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *Handlers) GetMail(w http.ResponseWriter, r *http.Request) {
	playerID := r.URL.Query().Get("playerId")
	if playerID == "" {
		writeError(w, http.StatusBadRequest, "playerId is required")
		return
	}
	if !h.requireOwnership(w, r, playerID) {
		return
	}

	mailID := r.PathValue("mailId")
	mail, err := h.gameService.GetMail(playerID, mailID)
	if err != nil {
		writeError(w, http.StatusNotFound, "mail not found")
		return
	}
	writeJSON(w, http.StatusOK, mail)
}

func (h *Handlers) DeleteMail(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		PlayerID string `json:"playerId"`
	}
	if !decodeJSON(w, r, &payload) {
		return
	}
	if !h.requireOwnership(w, r, payload.PlayerID) {
		return
	}

	if err := h.gameService.DeleteMail(payload.PlayerID, r.PathValue("mailId")); err != nil {
		writeError(w, http.StatusNotFound, "mail not found")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (h *Handlers) ClaimMailAttachments(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		PlayerID string `json:"playerId"`
	}
	if !decodeJSON(w, r, &payload) {
		return
	}
	if !h.requireOwnership(w, r, payload.PlayerID) {
		return
	}
	result, err := h.gameService.ClaimMailAttachments(payload.PlayerID, r.PathValue("mailId"))
	if err != nil {
		status := http.StatusBadRequest
		switch {
		case errors.Is(err, game.ErrMailNotFound):
			status = http.StatusNotFound
		case errors.Is(err, game.ErrMailAlreadyClaimed), errors.Is(err, game.ErrMailExpired), errors.Is(err, game.ErrMailClaimForbidden), errors.Is(err, game.ErrMailNoAttachments):
			status = http.StatusConflict
		case errors.Is(err, game.ErrMailInvalidAttachment), errors.Is(err, game.ErrItemNotFound):
			status = http.StatusUnprocessableEntity
		}
		writeError(w, status, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *Handlers) SendPlayerMail(w http.ResponseWriter, r *http.Request) {
	var payload game.SendPlayerMailRequest
	if !decodeJSON(w, r, &payload) {
		return
	}
	if !h.requireOwnership(w, r, payload.SenderPlayerID) {
		return
	}
	mail, err := h.gameService.SendPlayerMail(payload)
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, game.ErrPlayerNotFound) {
			status = http.StatusNotFound
		}
		writeError(w, status, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, mail)
}

func (h *Handlers) SendServerBroadcastMail(w http.ResponseWriter, r *http.Request) {
	var payload game.SendServerBroadcastMailRequest
	if !decodeJSON(w, r, &payload) {
		return
	}
	if !h.requireOwnership(w, r, payload.SenderPlayerID) {
		return
	}
	result, err := h.gameService.SendServerBroadcastMail(payload)
	if err != nil {
		status := http.StatusBadRequest
		switch {
		case errors.Is(err, game.ErrPlayerNotFound):
			status = http.StatusNotFound
		case errors.Is(err, game.ErrInsufficientCityGold):
			status = http.StatusConflict
		case errors.Is(err, game.ErrInvalidMail):
			status = http.StatusUnprocessableEntity
		}
		writeError(w, status, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, result)
}

func (h *Handlers) AdminSendMail(w http.ResponseWriter, r *http.Request) {
	var payload game.SendMailRequest
	if !decodeJSON(w, r, &payload) {
		return
	}
	payload.SenderType = "gm"
	if payload.SenderName == "" {
		payload.SenderName = "Hero3 GM"
	}
	if payload.SourceType == "" {
		payload.SourceType = "manual"
	}

	mail, err := h.gameService.SendMail(payload)
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, game.ErrPlayerNotFound) {
			status = http.StatusNotFound
		}
		writeError(w, status, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, mail)
}

func (h *Handlers) AdminPlayerMails(w http.ResponseWriter, r *http.Request) {
	playerID := r.PathValue("playerId")
	if playerID == "" {
		writeError(w, http.StatusBadRequest, "playerId is required")
		return
	}
	page, pageSize, ok := parsePageQuery(w, r)
	if !ok {
		return
	}
	result, err := h.gameService.ListMails(playerID, page, pageSize, r.URL.Query().Get("mailType"))
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, result)
}
