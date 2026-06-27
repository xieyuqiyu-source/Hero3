// 本文件提供 GM 后台 PVP 只读查询接口，避免管理端绕过服务边界直接读取仓储。
package api

import (
	"errors"
	"hero3/internal/app/game"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// AdminPvpOverview 返回 GM PVP 工作台总览数据。
func (h *Handlers) AdminPvpOverview(w http.ResponseWriter, r *http.Request) {
	playerID := strings.TrimSpace(r.URL.Query().Get("playerId"))
	limit := parseAdminPvpLimit(r)
	overview, err := h.gameService.AdminPvpOverview(playerID, limit)
	if err != nil {
		writeAdminPvpError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, overview)
}

// AdminPvpMarches 返回 GM 视角 PVP 行军列表。
func (h *Handlers) AdminPvpMarches(w http.ResponseWriter, r *http.Request) {
	playerID := strings.TrimSpace(r.URL.Query().Get("playerId"))
	limit := parseAdminPvpLimit(r)
	marches, err := h.gameService.AdminPvpMarches(playerID, limit)
	if err != nil {
		writeAdminPvpError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, marches)
}

// AdminPvpBattles 返回 GM 视角 PVP 战斗列表。
func (h *Handlers) AdminPvpBattles(w http.ResponseWriter, r *http.Request) {
	playerID := strings.TrimSpace(r.URL.Query().Get("playerId"))
	limit := parseAdminPvpLimit(r)
	battles, err := h.gameService.AdminPvpBattles(playerID, limit)
	if err != nil {
		writeAdminPvpError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, battles)
}

// AdminPvpSeasons 返回 GM 视角 PVP 赛季列表。
func (h *Handlers) AdminPvpSeasons(w http.ResponseWriter, r *http.Request) {
	seasons, err := h.gameService.AdminPvpSeasons()
	if err != nil {
		writeAdminPvpError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, seasons)
}

// AdminCreatePvpSeason 创建 PVP 赛季。
func (h *Handlers) AdminCreatePvpSeason(w http.ResponseWriter, r *http.Request) {
	var payload game.AdminSavePvpSeasonRequest
	if !decodeJSON(w, r, &payload) {
		return
	}
	season, err := h.gameService.AdminCreatePvpSeason(payload)
	if err != nil {
		writeAdminPvpError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, season)
}

// AdminUpdatePvpSeason 更新 PVP 赛季。
func (h *Handlers) AdminUpdatePvpSeason(w http.ResponseWriter, r *http.Request) {
	seasonID := strings.TrimSpace(r.PathValue("seasonId"))
	var payload game.AdminSavePvpSeasonRequest
	if !decodeJSON(w, r, &payload) {
		return
	}
	season, err := h.gameService.AdminUpdatePvpSeason(seasonID, payload)
	if err != nil {
		writeAdminPvpError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, season)
}

// AdminSettlePvpSeason 结算指定 PVP 赛季。
func (h *Handlers) AdminSettlePvpSeason(w http.ResponseWriter, r *http.Request) {
	seasonID := strings.TrimSpace(r.PathValue("seasonId"))
	result, err := h.gameService.AdminSettlePvpSeason(seasonID)
	if err != nil {
		writeAdminPvpError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

// AdminPvpPlayer 返回 GM 视角单个玩家 PVP 状态。
func (h *Handlers) AdminPvpPlayer(w http.ResponseWriter, r *http.Request) {
	playerID := strings.TrimSpace(r.PathValue("playerId"))
	state, err := h.gameService.GetPvpState(playerID)
	if err != nil {
		writeAdminPvpError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, state)
}

// AdminSetPvpProtection 设置玩家 PVP 保护，用于系统保护和维护保护排障。
func (h *Handlers) AdminSetPvpProtection(w http.ResponseWriter, r *http.Request) {
	playerID := strings.TrimSpace(r.PathValue("playerId"))
	var payload game.AdminSetPvpProtectionRequest
	if !decodeJSON(w, r, &payload) {
		return
	}
	state, err := h.gameService.SetPvpProtection(playerID, payload.ProtectionType, time.Duration(payload.Hours)*time.Hour, payload.Reason, time.Now().UTC())
	if err != nil {
		writeAdminPvpError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, state)
}

// AdminForceResolvePvpMarch 强制结算一条异常 PVP 行军。
func (h *Handlers) AdminForceResolvePvpMarch(w http.ResponseWriter, r *http.Request) {
	marchID := strings.TrimSpace(r.PathValue("marchId"))
	battle, err := h.gameService.AdminForceResolvePvpMarch(marchID)
	if err != nil {
		writeAdminPvpError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, battle)
}

// AdminCancelPvpMarch 取消一条未结算 PVP 行军并返还出征兵力。
func (h *Handlers) AdminCancelPvpMarch(w http.ResponseWriter, r *http.Request) {
	marchID := strings.TrimSpace(r.PathValue("marchId"))
	result, err := h.gameService.AdminCancelPvpMarch(marchID)
	if err != nil {
		writeAdminPvpError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

// parseAdminPvpLimit 解析 GM PVP 列表查询上限。
func parseAdminPvpLimit(r *http.Request) int {
	limit, _ := strconv.Atoi(strings.TrimSpace(r.URL.Query().Get("limit")))
	return limit
}

// writeAdminPvpError 统一返回 GM PVP 查询错误。
func writeAdminPvpError(w http.ResponseWriter, err error) {
	status := http.StatusBadRequest
	if errors.Is(err, game.ErrPlayerNotFound) {
		status = http.StatusNotFound
	}
	writeError(w, status, err.Error())
}
