// 本文件提供 GM 后台轮回绝境配置和实例处理接口。
package api

import (
	"net/http"

	"hero3/internal/app/game"
)

// AdminReincarnationConfig 返回轮回绝境配置。
func (h *Handlers) AdminReincarnationConfig(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, h.gameService.GetReincarnationConfig())
}

// UpdateAdminReincarnationConfig 保存轮回绝境配置。
func (h *Handlers) UpdateAdminReincarnationConfig(w http.ResponseWriter, r *http.Request) {
	var payload game.ReincarnationConfig
	if !decodeJSON(w, r, &payload) {
		return
	}
	if err := h.gameService.UpdateReincarnationConfig(payload); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, h.gameService.GetReincarnationConfig())
}

// AdminReincarnationRuns 查询轮回绝境实例列表。
func (h *Handlers) AdminReincarnationRuns(w http.ResponseWriter, r *http.Request) {
	limit, offset := parseLimitOffset(r, 20, 100)
	items, total, err := h.gameService.ListReincarnationRunsForAdmin(r.URL.Query().Get("playerId"), limit, offset)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items, "total": total, "limit": limit, "offset": offset})
}

// AdminReincarnationRun 查询单个轮回绝境实例。
func (h *Handlers) AdminReincarnationRun(w http.ResponseWriter, r *http.Request) {
	run, err := h.gameService.GetReincarnationRunForAdmin(r.PathValue("runId"))
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, run)
}

// AdminForceSettleReincarnationRun 强制结算异常实例。
func (h *Handlers) AdminForceSettleReincarnationRun(w http.ResponseWriter, r *http.Request) {
	run, err := h.gameService.ForceSettleReincarnationRunForAdmin(r.PathValue("runId"))
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, run)
}

// AdminRepairReincarnationReward 修复累计奖励发放。
func (h *Handlers) AdminRepairReincarnationReward(w http.ResponseWriter, r *http.Request) {
	run, err := h.gameService.RepairReincarnationRewardForAdmin(r.PathValue("runId"))
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, run)
}
