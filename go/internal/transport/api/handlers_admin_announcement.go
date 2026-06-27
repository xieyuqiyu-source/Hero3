// 本文件归口 GM 公告管理 HTTP 接口处理。
package api

import (
	"net/http"

	"hero3/internal/app/game"
)

// AdminAnnouncements 查询 GM 公告列表。
func (h *Handlers) AdminAnnouncements(w http.ResponseWriter, r *http.Request) {
	page, pageSize, ok := parsePageQuery(w, r)
	if !ok {
		return
	}
	result, err := h.gameService.ListAdminAnnouncements(game.AdminAnnouncementFilter{
		Type:     r.URL.Query().Get("type"),
		Status:   r.URL.Query().Get("status"),
		Page:     page,
		PageSize: pageSize,
	})
	if err != nil {
		h.writeAnnouncementError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

// AdminCreateAnnouncement 新建公告。
func (h *Handlers) AdminCreateAnnouncement(w http.ResponseWriter, r *http.Request) {
	var payload game.SaveAnnouncementRequest
	if !decodeJSON(w, r, &payload) {
		return
	}
	result, err := h.gameService.CreateAnnouncement(payload)
	if err != nil {
		h.writeAnnouncementError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, result)
}

// AdminUpdateAnnouncement 编辑公告。
func (h *Handlers) AdminUpdateAnnouncement(w http.ResponseWriter, r *http.Request) {
	var payload game.SaveAnnouncementRequest
	if !decodeJSON(w, r, &payload) {
		return
	}
	result, err := h.gameService.UpdateAnnouncement(r.PathValue("announcementId"), payload)
	if err != nil {
		h.writeAnnouncementError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

// AdminPublishAnnouncement 立即发布公告。
func (h *Handlers) AdminPublishAnnouncement(w http.ResponseWriter, r *http.Request) {
	result, err := h.gameService.PublishAnnouncement(r.PathValue("announcementId"))
	if err != nil {
		h.writeAnnouncementError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

// AdminScheduleAnnouncement 定时发布公告。
func (h *Handlers) AdminScheduleAnnouncement(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		StartsAt string `json:"startsAt"`
	}
	if !decodeJSON(w, r, &payload) {
		return
	}
	result, err := h.gameService.ScheduleAnnouncement(r.PathValue("announcementId"), payload.StartsAt)
	if err != nil {
		h.writeAnnouncementError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

// AdminWithdrawAnnouncement 撤回公告。
func (h *Handlers) AdminWithdrawAnnouncement(w http.ResponseWriter, r *http.Request) {
	result, err := h.gameService.WithdrawAnnouncement(r.PathValue("announcementId"))
	if err != nil {
		h.writeAnnouncementError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

// AdminArchiveAnnouncement 归档公告。
func (h *Handlers) AdminArchiveAnnouncement(w http.ResponseWriter, r *http.Request) {
	result, err := h.gameService.ArchiveAnnouncement(r.PathValue("announcementId"))
	if err != nil {
		h.writeAnnouncementError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

// AdminDeleteAnnouncement 删除草稿公告。
func (h *Handlers) AdminDeleteAnnouncement(w http.ResponseWriter, r *http.Request) {
	if err := h.gameService.DeleteAnnouncementDraft(r.PathValue("announcementId")); err != nil {
		h.writeAnnouncementError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}
