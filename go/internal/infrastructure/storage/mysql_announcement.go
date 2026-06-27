// 本文件归口公告系统的 MySQL 持久化、投放规则和玩家阅读状态。
package storage

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"hero3/internal/app/game"
)

// GetAnnouncementPlayerContext 读取公告投放判断所需的玩家上下文。
func (r *MySQLRepository) GetAnnouncementPlayerContext(playerID string) (game.AnnouncementPlayerContext, error) {
	var ctx game.AnnouncementPlayerContext
	var createdAt time.Time
	err := r.db.QueryRow(`SELECT id, account_id, faction, created_at FROM players WHERE id = ? LIMIT 1`, playerID).Scan(&ctx.PlayerID, &ctx.AccountID, &ctx.Faction, &createdAt)
	if errors.Is(err, sql.ErrNoRows) {
		return game.AnnouncementPlayerContext{}, game.ErrPlayerNotFound
	}
	if err != nil {
		return game.AnnouncementPlayerContext{}, err
	}
	// 等级段投放暂不开发：玩家等级来源尚未稳定，先固定为 0 并保留字段。
	ctx.Level = 0
	ctx.CreatedAt = createdAt.UTC().Format(time.RFC3339)
	return ctx, nil
}

// PromoteDueScheduledAnnouncements 将已到开始时间的定时公告转为已发布。
func (r *MySQLRepository) PromoteDueScheduledAnnouncements(now time.Time) error {
	_, err := r.db.Exec(`UPDATE announcements
		SET status = ?, published_at = COALESCE(published_at, ?), updated_at = ?
		WHERE status = ? AND starts_at IS NOT NULL AND starts_at <= ?`,
		game.AnnouncementStatusPublished, now.UTC(), now.UTC(), game.AnnouncementStatusScheduled, now.UTC())
	return err
}

// ListVisibleAnnouncements 返回当前玩家可见公告摘要。
func (r *MySQLRepository) ListVisibleAnnouncements(ctx game.AnnouncementPlayerContext, filter game.AnnouncementListFilter, now time.Time) ([]game.AnnouncementSummary, int, error) {
	candidates, err := r.listAnnouncementCandidates(filter)
	if err != nil {
		return nil, 0, err
	}
	readMap, err := r.loadAnnouncementReads(ctx.PlayerID)
	if err != nil {
		return nil, 0, err
	}
	items := []game.AnnouncementSummary{}
	for _, announcement := range candidates {
		if !game.AnnouncementVisibleToPlayer(announcement, ctx, filter.IncludeArchived, now) {
			continue
		}
		items = append(items, game.BuildAnnouncementSummary(announcement, readMap[announcement.ID]))
	}
	game.SortAnnouncementSummaries(items)
	total := len(items)
	return append([]game.AnnouncementSummary(nil), items...), total, nil
}

// GetVisibleAnnouncementDetail 返回当前玩家可见公告详情。
func (r *MySQLRepository) GetVisibleAnnouncementDetail(ctx game.AnnouncementPlayerContext, announcementID string, now time.Time) (game.AnnouncementDetail, error) {
	announcement, err := r.GetAdminAnnouncement(announcementID)
	if err != nil {
		return game.AnnouncementDetail{}, err
	}
	if !game.AnnouncementVisibleToPlayer(announcement, ctx, true, now) {
		return game.AnnouncementDetail{}, game.ErrAnnouncementNotVisible
	}
	readMap, err := r.loadAnnouncementReads(ctx.PlayerID)
	if err != nil {
		return game.AnnouncementDetail{}, err
	}
	return game.AnnouncementDetail{AnnouncementSummary: game.BuildAnnouncementSummary(announcement, readMap[announcement.ID]), Content: announcement.Content}, nil
}

// MarkAnnouncementRead 幂等标记公告已读。
func (r *MySQLRepository) MarkAnnouncementRead(ctx game.AnnouncementPlayerContext, announcementID string, now time.Time) (game.AnnouncementReadState, error) {
	return r.upsertAnnouncementRead(ctx, announcementID, now, "read")
}

// MarkAnnouncementPopupShown 幂等记录公告弹窗已展示。
func (r *MySQLRepository) MarkAnnouncementPopupShown(ctx game.AnnouncementPlayerContext, announcementID string, now time.Time) (game.AnnouncementReadState, error) {
	return r.upsertAnnouncementRead(ctx, announcementID, now, "popup")
}

// DismissAnnouncement 幂等记录公告弹窗关闭。
func (r *MySQLRepository) DismissAnnouncement(ctx game.AnnouncementPlayerContext, announcementID string, now time.Time) (game.AnnouncementReadState, error) {
	return r.upsertAnnouncementRead(ctx, announcementID, now, "dismiss")
}

// ListAdminAnnouncements 返回 GM 公告列表。
func (r *MySQLRepository) ListAdminAnnouncements(filter game.AdminAnnouncementFilter) ([]game.Announcement, int, error) {
	where := []string{"1=1"}
	args := []any{}
	if strings.TrimSpace(filter.Type) != "" {
		where = append(where, "type = ?")
		args = append(args, strings.TrimSpace(filter.Type))
	}
	if strings.TrimSpace(filter.Status) != "" {
		where = append(where, "status = ?")
		args = append(args, strings.TrimSpace(filter.Status))
	}
	whereSQL := strings.Join(where, " AND ")
	var total int
	if err := r.db.QueryRow(`SELECT COUNT(*) FROM announcements WHERE `+whereSQL, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	start, _ := announcementPageBounds(total, filter.Page, filter.PageSize)
	queryArgs := append(append([]any{}, args...), filter.PageSize, start)
	rows, err := r.db.Query(`SELECT `+announcementColumns+` FROM announcements WHERE `+whereSQL+` ORDER BY updated_at DESC LIMIT ? OFFSET ?`, queryArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	items, err := scanAnnouncements(rows)
	if err != nil {
		return nil, 0, err
	}
	if err := r.attachAnnouncementTargets(items); err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

// GetAdminAnnouncement 返回 GM 视角公告详情。
func (r *MySQLRepository) GetAdminAnnouncement(announcementID string) (game.Announcement, error) {
	announcement, err := scanAnnouncement(r.db.QueryRow(`SELECT `+announcementColumns+` FROM announcements WHERE id = ? LIMIT 1`, announcementID))
	if errors.Is(err, sql.ErrNoRows) {
		return game.Announcement{}, game.ErrAnnouncementNotFound
	}
	if err != nil {
		return game.Announcement{}, err
	}
	items := []game.Announcement{announcement}
	if err := r.attachAnnouncementTargets(items); err != nil {
		return game.Announcement{}, err
	}
	return items[0], nil
}

// SaveAnnouncement 保存公告主体和投放规则。
func (r *MySQLRepository) SaveAnnouncement(announcement game.Announcement) (game.Announcement, error) {
	tx, err := r.db.Begin()
	if err != nil {
		return game.Announcement{}, err
	}
	defer func() { _ = tx.Rollback() }()
	if err := saveAnnouncementTx(tx, announcement); err != nil {
		return game.Announcement{}, err
	}
	if _, err := tx.Exec(`DELETE FROM announcement_targets WHERE announcement_id = ?`, announcement.ID); err != nil {
		return game.Announcement{}, err
	}
	for _, target := range announcement.Targets {
		valueJSON, _ := json.Marshal(target.Value)
		if _, err := tx.Exec(`INSERT INTO announcement_targets (announcement_id, target_type, target_value_json, created_at, updated_at) VALUES (?, ?, ?, ?, ?)`,
			announcement.ID, target.Type, nullableJSON(valueJSON), parseStorageTime(announcement.CreatedAt), parseStorageTime(announcement.UpdatedAt)); err != nil {
			return game.Announcement{}, err
		}
	}
	if err := tx.Commit(); err != nil {
		return game.Announcement{}, err
	}
	return announcement, nil
}

// UpdateAnnouncementStatus 更新公告生命周期状态。
func (r *MySQLRepository) UpdateAnnouncementStatus(announcementID string, status string, now time.Time) (game.Announcement, error) {
	announcement, err := r.GetAdminAnnouncement(announcementID)
	if err != nil {
		return game.Announcement{}, err
	}
	nowText := now.UTC().Format(time.RFC3339)
	announcement.Status = status
	announcement.UpdatedAt = nowText
	switch status {
	case game.AnnouncementStatusPublished:
		if announcement.PublishedAt == "" {
			announcement.PublishedAt = nowText
		}
	case game.AnnouncementStatusWithdrawn:
		announcement.WithdrawnAt = nowText
	case game.AnnouncementStatusArchived:
		announcement.ArchivedAt = nowText
	}
	return r.SaveAnnouncement(announcement)
}

// DeleteAnnouncementDraft 删除草稿公告。
func (r *MySQLRepository) DeleteAnnouncementDraft(announcementID string) error {
	announcement, err := r.GetAdminAnnouncement(announcementID)
	if err != nil {
		return err
	}
	if announcement.Status != game.AnnouncementStatusDraft {
		return game.ErrAnnouncementDeleteDenied
	}
	_, err = r.db.Exec(`DELETE FROM announcements WHERE id = ? AND status = ?`, announcementID, game.AnnouncementStatusDraft)
	return err
}

const announcementColumns = `id, title, summary, content, type, status, display_mode, pinned, priority, force_popup, starts_at, ends_at, published_at, withdrawn_at, archived_at, created_at, updated_at`

type announcementScanner interface {
	Scan(dest ...any) error
}

func scanAnnouncement(scanner announcementScanner) (game.Announcement, error) {
	var item game.Announcement
	var startsAt, endsAt, publishedAt, withdrawnAt, archivedAt, createdAt, updatedAt sql.NullTime
	err := scanner.Scan(&item.ID, &item.Title, &item.Summary, &item.Content, &item.Type, &item.Status, &item.DisplayMode, &item.Pinned, &item.Priority, &item.ForcePopup, &startsAt, &endsAt, &publishedAt, &withdrawnAt, &archivedAt, &createdAt, &updatedAt)
	if err != nil {
		return game.Announcement{}, err
	}
	item.StartsAt = formatNullTime(startsAt)
	item.EndsAt = formatNullTime(endsAt)
	item.PublishedAt = formatNullTime(publishedAt)
	item.WithdrawnAt = formatNullTime(withdrawnAt)
	item.ArchivedAt = formatNullTime(archivedAt)
	item.CreatedAt = formatNullTime(createdAt)
	item.UpdatedAt = formatNullTime(updatedAt)
	return item, nil
}

func scanAnnouncements(rows *sql.Rows) ([]game.Announcement, error) {
	items := []game.Announcement{}
	for rows.Next() {
		item, err := scanAnnouncement(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func saveAnnouncementTx(tx *sql.Tx, item game.Announcement) error {
	_, err := tx.Exec(`INSERT INTO announcements (id, title, summary, content, type, status, display_mode, pinned, priority, force_popup, starts_at, ends_at, published_at, withdrawn_at, archived_at, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE title = VALUES(title), summary = VALUES(summary), content = VALUES(content), type = VALUES(type), status = VALUES(status), display_mode = VALUES(display_mode), pinned = VALUES(pinned), priority = VALUES(priority), force_popup = VALUES(force_popup), starts_at = VALUES(starts_at), ends_at = VALUES(ends_at), published_at = VALUES(published_at), withdrawn_at = VALUES(withdrawn_at), archived_at = VALUES(archived_at), updated_at = VALUES(updated_at)`,
		item.ID, item.Title, item.Summary, item.Content, item.Type, item.Status, item.DisplayMode, item.Pinned, item.Priority, item.ForcePopup, parseNullableTime(item.StartsAt), parseNullableTime(item.EndsAt), parseNullableTime(item.PublishedAt), parseNullableTime(item.WithdrawnAt), parseNullableTime(item.ArchivedAt), parseStorageTime(item.CreatedAt), parseStorageTime(item.UpdatedAt))
	return err
}

func (r *MySQLRepository) listAnnouncementCandidates(filter game.AnnouncementListFilter) ([]game.Announcement, error) {
	statuses := []string{game.AnnouncementStatusPublished}
	if filter.IncludeArchived {
		statuses = append(statuses, game.AnnouncementStatusArchived)
	}
	placeholders := strings.TrimRight(strings.Repeat("?,", len(statuses)), ",")
	args := []any{}
	for _, status := range statuses {
		args = append(args, status)
	}
	where := "status IN (" + placeholders + ")"
	if strings.TrimSpace(filter.Type) != "" {
		where += " AND type = ?"
		args = append(args, strings.TrimSpace(filter.Type))
	}
	rows, err := r.db.Query(`SELECT `+announcementColumns+` FROM announcements WHERE `+where+` ORDER BY pinned DESC, priority DESC, published_at DESC, updated_at DESC`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items, err := scanAnnouncements(rows)
	if err != nil {
		return nil, err
	}
	if err := r.attachAnnouncementTargets(items); err != nil {
		return nil, err
	}
	return items, nil
}

func (r *MySQLRepository) attachAnnouncementTargets(items []game.Announcement) error {
	if len(items) == 0 {
		return nil
	}
	ids := make([]string, 0, len(items))
	index := map[string]int{}
	for i := range items {
		ids = append(ids, items[i].ID)
		index[items[i].ID] = i
	}
	query := `SELECT announcement_id, target_type, target_value_json FROM announcement_targets WHERE announcement_id IN (` + sqlPlaceholders(len(ids)) + `)`
	args := make([]any, 0, len(ids))
	for _, id := range ids {
		args = append(args, id)
	}
	rows, err := r.db.Query(query, args...)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var announcementID, targetType string
		var valueJSON sql.NullString
		if err := rows.Scan(&announcementID, &targetType, &valueJSON); err != nil {
			return err
		}
		var value any
		if valueJSON.Valid && strings.TrimSpace(valueJSON.String) != "" {
			_ = json.Unmarshal([]byte(valueJSON.String), &value)
		}
		if i, ok := index[announcementID]; ok {
			items[i].Targets = append(items[i].Targets, game.AnnouncementTarget{Type: targetType, Value: value})
		}
	}
	return rows.Err()
}

func (r *MySQLRepository) loadAnnouncementReads(playerID string) (map[string]game.AnnouncementReadState, error) {
	rows, err := r.db.Query(`SELECT announcement_id, player_id, account_id, is_read, read_at, is_popup_shown, popup_shown_at, is_dismissed, dismissed_at, created_at, updated_at FROM announcement_reads WHERE player_id = ?`, playerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := map[string]game.AnnouncementReadState{}
	for rows.Next() {
		state, err := scanAnnouncementRead(rows)
		if err != nil {
			return nil, err
		}
		result[state.AnnouncementID] = state
	}
	return result, rows.Err()
}

func (r *MySQLRepository) upsertAnnouncementRead(ctx game.AnnouncementPlayerContext, announcementID string, now time.Time, action string) (game.AnnouncementReadState, error) {
	nowTime := now.UTC()
	nowArg := nowTime
	fields := map[string]string{
		"read":    "is_read = TRUE, read_at = COALESCE(read_at, VALUES(read_at))",
		"popup":   "is_popup_shown = TRUE, popup_shown_at = COALESCE(popup_shown_at, VALUES(popup_shown_at))",
		"dismiss": "is_dismissed = TRUE, dismissed_at = COALESCE(dismissed_at, VALUES(dismissed_at))",
	}
	updateSQL, ok := fields[action]
	if !ok {
		return game.AnnouncementReadState{}, fmt.Errorf("unknown announcement read action: %s", action)
	}
	readAt, popupAt, dismissedAt := any(nil), any(nil), any(nil)
	switch action {
	case "read":
		readAt = nowArg
	case "popup":
		popupAt = nowArg
	case "dismiss":
		dismissedAt = nowArg
	}
	_, err := r.db.Exec(`INSERT INTO announcement_reads (announcement_id, player_id, account_id, is_read, read_at, is_popup_shown, popup_shown_at, is_dismissed, dismissed_at, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE account_id = VALUES(account_id), `+updateSQL+`, updated_at = VALUES(updated_at)`,
		announcementID, ctx.PlayerID, ctx.AccountID, action == "read", readAt, action == "popup", popupAt, action == "dismiss", dismissedAt, nowArg, nowArg)
	if err != nil {
		return game.AnnouncementReadState{}, err
	}
	state, err := scanAnnouncementRead(r.db.QueryRow(`SELECT announcement_id, player_id, account_id, is_read, read_at, is_popup_shown, popup_shown_at, is_dismissed, dismissed_at, created_at, updated_at FROM announcement_reads WHERE announcement_id = ? AND player_id = ?`, announcementID, ctx.PlayerID))
	if errors.Is(err, sql.ErrNoRows) {
		return game.AnnouncementReadState{}, game.ErrAnnouncementNotFound
	}
	return state, err
}

func scanAnnouncementRead(scanner announcementScanner) (game.AnnouncementReadState, error) {
	var state game.AnnouncementReadState
	var readAt, popupShownAt, dismissedAt, createdAt, updatedAt sql.NullTime
	if err := scanner.Scan(&state.AnnouncementID, &state.PlayerID, &state.AccountID, &state.IsRead, &readAt, &state.IsPopupShown, &popupShownAt, &state.IsDismissed, &dismissedAt, &createdAt, &updatedAt); err != nil {
		return game.AnnouncementReadState{}, err
	}
	state.ReadAt = formatNullTime(readAt)
	state.PopupShownAt = formatNullTime(popupShownAt)
	state.DismissedAt = formatNullTime(dismissedAt)
	state.CreatedAt = formatNullTime(createdAt)
	state.UpdatedAt = formatNullTime(updatedAt)
	return state, nil
}

func sqlPlaceholders(count int) string {
	if count <= 0 {
		return ""
	}
	return strings.TrimRight(strings.Repeat("?,", count), ",")
}

func announcementPageBounds(total int, page int, pageSize int) (int, int) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	start := (page - 1) * pageSize
	if start > total {
		start = total
	}
	end := start + pageSize
	if end > total {
		end = total
	}
	return start, end
}
