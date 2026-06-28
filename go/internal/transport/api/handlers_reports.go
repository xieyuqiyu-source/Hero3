package api

import (
	"hero3/internal/app/game"
	"net/http"
	"strconv"
	"time"
)

func (h *Handlers) GetReport(w http.ResponseWriter, r *http.Request) {
	reportID := r.PathValue("reportId")
	if reportID == "" {
		writeError(w, http.StatusBadRequest, "reportId is required")
		return
	}

	playerID := r.URL.Query().Get("playerId")
	if playerID != "" {
		if !h.requireOwnership(w, r, playerID) {
			return
		}
		report, err := h.gameService.GetReportForPlayer(playerID, reportID)
		if err != nil {
			writeError(w, http.StatusNotFound, "report not found")
			return
		}
		writeJSON(w, http.StatusOK, report)
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

	query := game.BattleReportQuery{
		PlayerID:       playerID,
		ViewType:       r.URL.Query().Get("viewType"),
		SourceType:     r.URL.Query().Get("sourceType"),
		BattleType:     r.URL.Query().Get("battleType"),
		Result:         r.URL.Query().Get("result"),
		Page:           page,
		PageSize:       pageSize,
		IncludeDeleted: r.URL.Query().Get("includeDeleted") == "true",
	}
	if raw := r.URL.Query().Get("timeFrom"); raw != "" {
		parsed, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			writeError(w, http.StatusBadRequest, "timeFrom must be RFC3339")
			return
		}
		query.TimeFrom = parsed
	}
	if raw := r.URL.Query().Get("timeTo"); raw != "" {
		parsed, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			writeError(w, http.StatusBadRequest, "timeTo must be RFC3339")
			return
		}
		query.TimeTo = parsed
	}

	result, err := h.gameService.ListReportsByQuery(query)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, result)
}

// ShareReport 为战报创建公开分享 token。
func (h *Handlers) ShareReport(w http.ResponseWriter, r *http.Request) {
	reportID := r.PathValue("reportId")
	var payload struct {
		PlayerID string `json:"playerId"`
	}
	if !decodeJSON(w, r, &payload) {
		return
	}
	if !h.requireOwnership(w, r, payload.PlayerID) {
		return
	}
	link, err := h.gameService.ShareBattleReport(payload.PlayerID, reportID)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, link)
}

// GetSharedReport 通过分享 token 获取公开战报。
func (h *Handlers) GetSharedReport(w http.ResponseWriter, r *http.Request) {
	token := r.PathValue("token")
	report, err := h.gameService.GetSharedReportByToken(token)
	if err != nil {
		writeError(w, http.StatusNotFound, "report not found")
		return
	}
	writeJSON(w, http.StatusOK, report)
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

	var result any
	var err error

	if payload.ReportID != "" {
		// 标记单条
		result, err = h.gameService.MarkSingleReportRead(payload.PlayerID, payload.ReportID)
	} else {
		// 标记全部
		result, err = h.gameService.MarkReportsRead(payload.PlayerID)
	}

	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, result)
}

// MarkReportReadByPath 标记路径中的单条战报为已读。
func (h *Handlers) MarkReportReadByPath(w http.ResponseWriter, r *http.Request) {
	reportID := r.PathValue("reportId")
	var payload struct {
		PlayerID string `json:"playerId"`
	}
	if !decodeJSON(w, r, &payload) {
		return
	}
	if !h.requireOwnership(w, r, payload.PlayerID) {
		return
	}
	result, err := h.gameService.MarkSingleReportRead(payload.PlayerID, reportID)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, result)
}

// MarkAllReportsReadByPath 标记指定视角或全部战报为已读。
func (h *Handlers) MarkAllReportsReadByPath(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		PlayerID string `json:"playerId"`
		ViewType string `json:"viewType"`
	}
	if !decodeJSON(w, r, &payload) {
		return
	}
	if !h.requireOwnership(w, r, payload.PlayerID) {
		return
	}
	result, err := h.gameService.MarkReportsReadByView(payload.PlayerID, payload.ViewType)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, result)
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

	result, err := h.gameService.DeleteReport(payload.PlayerID, payload.ReportID)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, result)
}

// DeleteReportByPath 删除路径中的单条战报。
func (h *Handlers) DeleteReportByPath(w http.ResponseWriter, r *http.Request) {
	reportID := r.PathValue("reportId")
	var payload struct {
		PlayerID string `json:"playerId"`
	}
	if !decodeJSON(w, r, &payload) {
		return
	}
	if !h.requireOwnership(w, r, payload.PlayerID) {
		return
	}
	result, err := h.gameService.DeleteReport(payload.PlayerID, reportID)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, result)
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

	result, err := h.gameService.DeleteAllReports(payload.PlayerID)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, result)
}

// DeleteAllReportsByPath 删除指定视角或全部战报。
func (h *Handlers) DeleteAllReportsByPath(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		PlayerID string `json:"playerId"`
		ViewType string `json:"viewType"`
	}
	if !decodeJSON(w, r, &payload) {
		return
	}
	if !h.requireOwnership(w, r, payload.PlayerID) {
		return
	}
	result, err := h.gameService.DeleteReportsByView(payload.PlayerID, payload.ViewType)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, result)
}
