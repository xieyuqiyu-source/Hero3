// 本文件提供 GM 战报查询接口。
package api

import (
	"hero3/internal/app/game"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// AdminReports 返回 GM 视角战报列表或单条战报。
func (h *Handlers) AdminReports(w http.ResponseWriter, r *http.Request) {
	reportID := strings.TrimSpace(r.URL.Query().Get("reportId"))
	if reportID != "" {
		report, err := h.gameService.GetReportByID(reportID)
		if err != nil {
			writeError(w, http.StatusNotFound, "report not found")
			return
		}
		writeJSON(w, http.StatusOK, report)
		return
	}

	playerID := strings.TrimSpace(r.URL.Query().Get("playerId"))
	if playerID == "" {
		writeError(w, http.StatusBadRequest, "playerId or reportId is required")
		return
	}
	page, pageSize := parseAdminReportPage(r)
	result, err := h.gameService.ListReportsByQuery(game.BattleReportQuery{
		PlayerID:       playerID,
		ViewType:       strings.TrimSpace(r.URL.Query().Get("viewType")),
		SourceType:     strings.TrimSpace(r.URL.Query().Get("sourceType")),
		BattleType:     strings.TrimSpace(r.URL.Query().Get("battleType")),
		Result:         strings.TrimSpace(r.URL.Query().Get("result")),
		OwnerOutcome:   strings.TrimSpace(r.URL.Query().Get("ownerOutcome")),
		Page:           page,
		PageSize:       pageSize,
		IncludeDeleted: true,
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, result)
}

// AdminReport 返回 GM 单条战报详情。
func (h *Handlers) AdminReport(w http.ResponseWriter, r *http.Request) {
	reportID := strings.TrimSpace(r.PathValue("reportId"))
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

// parseAdminReportPage 解析 GM 战报分页参数。
func parseAdminReportPage(r *http.Request) (int, int) {
	page, _ := strconv.Atoi(strings.TrimSpace(r.URL.Query().Get("page")))
	if page < 1 {
		page = 1
	}
	pageSize, _ := strconv.Atoi(strings.TrimSpace(r.URL.Query().Get("pageSize")))
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	return page, pageSize
}

// AdminBattleEvents 返回 GM 战斗事件列表。
func (h *Handlers) AdminBattleEvents(w http.ResponseWriter, r *http.Request) {
	page, pageSize := parseAdminReportPage(r)
	query := game.BattleEventQuery{
		PlayerID:               strings.TrimSpace(r.URL.Query().Get("playerId")),
		EventID:                strings.TrimSpace(r.URL.Query().Get("eventId")),
		SourceType:             strings.TrimSpace(r.URL.Query().Get("sourceType")),
		SourceID:               strings.TrimSpace(r.URL.Query().Get("sourceId")),
		BattleType:             strings.TrimSpace(r.URL.Query().Get("battleType")),
		Result:                 strings.TrimSpace(r.URL.Query().Get("result")),
		RelatedMarchID:         strings.TrimSpace(r.URL.Query().Get("relatedMarchId")),
		RelatedReinforcementID: strings.TrimSpace(r.URL.Query().Get("relatedReinforcementId")),
		Page:                   page,
		PageSize:               pageSize,
	}
	if raw := strings.TrimSpace(r.URL.Query().Get("timeFrom")); raw != "" {
		parsed, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			writeError(w, http.StatusBadRequest, "timeFrom must be RFC3339")
			return
		}
		query.TimeFrom = parsed
	}
	if raw := strings.TrimSpace(r.URL.Query().Get("timeTo")); raw != "" {
		parsed, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			writeError(w, http.StatusBadRequest, "timeTo must be RFC3339")
			return
		}
		query.TimeTo = parsed
	}
	result, err := h.gameService.ListBattleEventsForAdmin(query)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, result)
}

// AdminBattleEvent 返回 GM 单个战斗事件。
func (h *Handlers) AdminBattleEvent(w http.ResponseWriter, r *http.Request) {
	eventID := strings.TrimSpace(r.PathValue("eventId"))
	event, err := h.gameService.GetBattleEventForAdmin(eventID)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, event)
}

// AdminBattleEventReports 返回同一战斗事件下的所有玩家视角战报。
func (h *Handlers) AdminBattleEventReports(w http.ResponseWriter, r *http.Request) {
	eventID := strings.TrimSpace(r.PathValue("eventId"))
	reports, err := h.gameService.ListReportsByEventForAdmin(eventID)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"reports": reports})
}

// AdminBattleEventParticipants 返回同一战斗事件下的参与方快照。
func (h *Handlers) AdminBattleEventParticipants(w http.ResponseWriter, r *http.Request) {
	eventID := strings.TrimSpace(r.PathValue("eventId"))
	participants, err := h.gameService.ListParticipantsByEventForAdmin(eventID)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"participants": participants})
}
