package api

import (
	"hero3/internal/app/game"
	"net/http"
	"strconv"
)

func (h *Handlers) GetReport(w http.ResponseWriter, r *http.Request) {
	reportID := r.PathValue("reportId")
	if reportID == "" {
		writeError(w, http.StatusBadRequest, "reportId is required")
		return
	}

	report, err := h.gameService.GetReportByID(reportID)
	if err != nil {
		writeError(w, http.StatusNotFound, "report not found")
		return
	}

	writeJSON(w, http.StatusOK, report)
}

func (h *Handlers) ListReports(w http.ResponseWriter, r *http.Request) {
	playerID := r.URL.Query().Get("playerId")
	if playerID == "" {
		writeError(w, http.StatusBadRequest, "playerId is required")
		return
	}
	if !h.requireOwnership(w, r, playerID) {
		return
	}

	page := 1
	if raw := r.URL.Query().Get("page"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 1 {
			writeError(w, http.StatusBadRequest, "page must be a positive integer")
			return
		}
		page = parsed
	}

	pageSize := 10
	if raw := r.URL.Query().Get("pageSize"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 1 {
			writeError(w, http.StatusBadRequest, "pageSize must be a positive integer")
			return
		}
		pageSize = parsed
	}
	if pageSize > 50 {
		pageSize = 50
	}

	result, err := h.gameService.ListReports(playerID, page, pageSize)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, result)
}

func (h *Handlers) MarkReportsRead(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		PlayerID string `json:"playerId"`
		ReportID string `json:"reportId"`
	}
	if !decodeJSON(w, r, &payload) {
		return
	}
	if !h.requireOwnership(w, r, payload.PlayerID) {
		return
	}

	var state game.GameState
	var err error

	if payload.ReportID != "" {
		// 标记单条
		state, err = h.gameService.MarkSingleReportRead(payload.PlayerID, payload.ReportID)
	} else {
		// 标记全部
		state, err = h.gameService.MarkReportsRead(payload.PlayerID)
	}

	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"state": state})
}

func (h *Handlers) DeleteReport(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		PlayerID string `json:"playerId"`
		ReportID string `json:"reportId"`
	}
	if !decodeJSON(w, r, &payload) {
		return
	}
	if !h.requireOwnership(w, r, payload.PlayerID) {
		return
	}

	state, err := h.gameService.DeleteReport(payload.PlayerID, payload.ReportID)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"state": state})
}

func (h *Handlers) DeleteAllReports(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		PlayerID string `json:"playerId"`
	}
	if !decodeJSON(w, r, &payload) {
		return
	}
	if !h.requireOwnership(w, r, payload.PlayerID) {
		return
	}

	state, err := h.gameService.DeleteAllReports(payload.PlayerID)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"state": state})
}
