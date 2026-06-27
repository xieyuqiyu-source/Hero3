// 本文件归口玩家侧公告 HTTP 接口处理。
package api

import (
	"errors"
	"net/http"
	"strconv"

	"hero3/internal/app/game"
)

// ListAnnouncements 查询当前玩家可见公告摘要。
func (h *Handlers) ListAnnouncements(w http.ResponseWriter, r *http.Request) {
	playerID := r.URL.Query().Get("playerId")
	if !h.requireOwnership(w, r, playerID) {
		return
	}
	page, pageSize, ok := parsePageQuery(w, r)
	if !ok {
		return
	}
	includeArchived := false
	if raw := r.URL.Query().Get("includeArchived"); raw != "" {
		includeArchived, _ = strconv.ParseBool(raw)
	}
	result, err := h.gameService.ListAnnouncements(game.AnnouncementListFilter{
		PlayerID:        playerID,
		Type:            r.URL.Query().Get("type"),
		IncludeArchived: includeArchived,
		Page:            page,
		PageSize:        pageSize,
	})
	if err != nil {
		h.writeAnnouncementError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

// GetAnnouncement 查询公告详情。
func (h *Handlers) GetAnnouncement(w http.ResponseWriter, r *http.Request) {
	playerID := r.URL.Query().Get("playerId")
	if !h.requireOwnership(w, r, playerID) {
		return
	}
	result, err := h.gameService.GetAnnouncementDetail(playerID, "", r.PathValue("announcementId"))
	if err != nil {
		h.writeAnnouncementError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

// MarkAnnouncementRead 标记公告已读。
func (h *Handlers) MarkAnnouncementRead(w http.ResponseWriter, r *http.Request) {
	playerID, ok := h.decodeAnnouncementPlayerPayload(w, r)
	if !ok {
		return
	}
	result, err := h.gameService.MarkAnnouncementRead(playerID, "", r.PathValue("announcementId"))
	if err != nil {
		h.writeAnnouncementError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

// MarkAnnouncementPopupShown 记录公告弹窗已展示。
func (h *Handlers) MarkAnnouncementPopupShown(w http.ResponseWriter, r *http.Request) {
	playerID, ok := h.decodeAnnouncementPlayerPayload(w, r)
	if !ok {
		return
	}
	result, err := h.gameService.MarkAnnouncementPopupShown(playerID, "", r.PathValue("announcementId"))
	if err != nil {
		h.writeAnnouncementError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

// DismissAnnouncement 记录公告弹窗已关闭。
func (h *Handlers) DismissAnnouncement(w http.ResponseWriter, r *http.Request) {
	playerID, ok := h.decodeAnnouncementPlayerPayload(w, r)
	if !ok {
		return
	}
	result, err := h.gameService.DismissAnnouncement(playerID, "", r.PathValue("announcementId"))
	if err != nil {
		h.writeAnnouncementError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

// ListAnnouncementPopups 查询当前玩家待展示公告弹窗队列。
func (h *Handlers) ListAnnouncementPopups(w http.ResponseWriter, r *http.Request) {
	playerID := r.URL.Query().Get("playerId")
	if !h.requireOwnership(w, r, playerID) {
		return
	}
	items, err := h.gameService.ListAnnouncementPopups(playerID, "")
	if err != nil {
		h.writeAnnouncementError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

// decodeAnnouncementPlayerPayload 读取公告状态写接口中的玩家 ID 并校验归属。
func (h *Handlers) decodeAnnouncementPlayerPayload(w http.ResponseWriter, r *http.Request) (string, bool) {
	var payload struct {
		PlayerID string `json:"playerId"`
	}
	if !decodeJSON(w, r, &payload) {
		return "", false
	}
	if !h.requireOwnership(w, r, payload.PlayerID) {
		return "", false
	}
	return payload.PlayerID, true
}

// writeAnnouncementError 写出公告系统统一错误。
func (h *Handlers) writeAnnouncementError(w http.ResponseWriter, err error) {
	status := http.StatusBadRequest
	switch {
	case errors.Is(err, game.ErrPlayerNotFound):
		status = http.StatusNotFound
	case errors.Is(err, game.ErrAnnouncementNotFound), errors.Is(err, game.ErrAnnouncementNotVisible):
		status = http.StatusNotFound
	case errors.Is(err, game.ErrInvalidAnnouncement), errors.Is(err, game.ErrAnnouncementDeleteDenied):
		status = http.StatusUnprocessableEntity
	}
	writeError(w, status, err.Error())
}
