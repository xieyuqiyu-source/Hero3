// 本文件实现内存仓储中的公告系统能力，供测试和无数据库开发模式使用。
package game

import (
	"encoding/json"
	"sort"
	"strings"
	"time"
)

// PromoteDueScheduledAnnouncements 将已到开始时间的定时公告转为已发布。
func (r *MemoryRepository) PromoteDueScheduledAnnouncements(now time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	nowText := now.UTC().Format(resourceDateLayout)
	for id, announcement := range r.announcements {
		if announcement.Status != AnnouncementStatusScheduled || strings.TrimSpace(announcement.StartsAt) == "" {
			continue
		}
		startsAt, err := time.Parse(resourceDateLayout, strings.TrimSpace(announcement.StartsAt))
		if err != nil || now.Before(startsAt) {
			continue
		}
		announcement.Status = AnnouncementStatusPublished
		announcement.PublishedAt = nowText
		announcement.UpdatedAt = nowText
		r.announcements[id] = announcement
	}
	return nil
}

// GetAnnouncementPlayerContext 返回公告投放判断所需的玩家上下文。
func (r *MemoryRepository) GetAnnouncementPlayerContext(playerID string) (AnnouncementPlayerContext, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	state, ok := r.players[playerID]
	if !ok {
		return AnnouncementPlayerContext{}, ErrPlayerNotFound
	}
	accountID := ""
	for id, playerIDs := range r.accountPlayers {
		for _, candidate := range playerIDs {
			if candidate == playerID {
				accountID = id
				break
			}
		}
		if accountID != "" {
			break
		}
	}
	createdAt := r.playerUpdatedAt[playerID].UTC().Format(resourceDateLayout)
	// 等级段投放暂不开发：玩家等级来源尚未稳定，先固定为 0 并保留字段。
	return AnnouncementPlayerContext{PlayerID: playerID, AccountID: accountID, Faction: state.Player.Faction, Level: 0, CreatedAt: createdAt}, nil
}

// ListVisibleAnnouncements 返回内存中当前玩家可见公告摘要。
func (r *MemoryRepository) ListVisibleAnnouncements(ctx AnnouncementPlayerContext, filter AnnouncementListFilter, now time.Time) ([]AnnouncementSummary, int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	items := []AnnouncementSummary{}
	for _, announcement := range r.announcements {
		if !announcementVisibleToPlayer(announcement, ctx, filter.IncludeArchived, now) {
			continue
		}
		if filter.Type != "" && announcement.Type != filter.Type {
			continue
		}
		read := r.announcementReads[announcementReadKey(announcement.ID, ctx.PlayerID)]
		items = append(items, buildAnnouncementSummary(announcement, read))
	}
	sortAnnouncementSummaries(items)
	total := len(items)
	return append([]AnnouncementSummary(nil), items...), total, nil
}

// GetVisibleAnnouncementDetail 返回玩家可见公告详情。
func (r *MemoryRepository) GetVisibleAnnouncementDetail(ctx AnnouncementPlayerContext, announcementID string, now time.Time) (AnnouncementDetail, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	announcement, ok := r.announcements[announcementID]
	if !ok {
		return AnnouncementDetail{}, ErrAnnouncementNotFound
	}
	if !announcementVisibleToPlayer(announcement, ctx, true, now) {
		return AnnouncementDetail{}, ErrAnnouncementNotVisible
	}
	read := r.announcementReads[announcementReadKey(announcement.ID, ctx.PlayerID)]
	return AnnouncementDetail{AnnouncementSummary: buildAnnouncementSummary(announcement, read), Content: announcement.Content}, nil
}

// MarkAnnouncementRead 幂等标记公告已读。
func (r *MemoryRepository) MarkAnnouncementRead(ctx AnnouncementPlayerContext, announcementID string, now time.Time) (AnnouncementReadState, error) {
	return r.upsertAnnouncementRead(ctx, announcementID, now, func(state *AnnouncementReadState, nowText string) {
		state.IsRead = true
		if state.ReadAt == "" {
			state.ReadAt = nowText
		}
	})
}

// MarkAnnouncementPopupShown 幂等记录公告弹窗已展示。
func (r *MemoryRepository) MarkAnnouncementPopupShown(ctx AnnouncementPlayerContext, announcementID string, now time.Time) (AnnouncementReadState, error) {
	return r.upsertAnnouncementRead(ctx, announcementID, now, func(state *AnnouncementReadState, nowText string) {
		state.IsPopupShown = true
		if state.PopupShownAt == "" {
			state.PopupShownAt = nowText
		}
	})
}

// DismissAnnouncement 幂等记录公告弹窗关闭。
func (r *MemoryRepository) DismissAnnouncement(ctx AnnouncementPlayerContext, announcementID string, now time.Time) (AnnouncementReadState, error) {
	return r.upsertAnnouncementRead(ctx, announcementID, now, func(state *AnnouncementReadState, nowText string) {
		state.IsDismissed = true
		if state.DismissedAt == "" {
			state.DismissedAt = nowText
		}
	})
}

// ListAdminAnnouncements 返回 GM 公告列表。
func (r *MemoryRepository) ListAdminAnnouncements(filter AdminAnnouncementFilter) ([]Announcement, int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	items := []Announcement{}
	for _, announcement := range r.announcements {
		if filter.Type != "" && announcement.Type != filter.Type {
			continue
		}
		if filter.Status != "" && announcement.Status != filter.Status {
			continue
		}
		items = append(items, announcement)
	}
	sort.SliceStable(items, func(i, j int) bool { return items[i].UpdatedAt > items[j].UpdatedAt })
	total := len(items)
	start, end := pageBounds(total, filter.Page, filter.PageSize)
	return append([]Announcement(nil), items[start:end]...), total, nil
}

// GetAdminAnnouncement 返回 GM 视角公告详情。
func (r *MemoryRepository) GetAdminAnnouncement(announcementID string) (Announcement, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	announcement, ok := r.announcements[announcementID]
	if !ok {
		return Announcement{}, ErrAnnouncementNotFound
	}
	return announcement, nil
}

// SaveAnnouncement 保存公告和投放规则。
func (r *MemoryRepository) SaveAnnouncement(announcement Announcement) (Announcement, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.announcements[announcement.ID] = announcement
	return announcement, nil
}

// UpdateAnnouncementStatus 更新公告生命周期状态。
func (r *MemoryRepository) UpdateAnnouncementStatus(announcementID string, status string, now time.Time) (Announcement, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	announcement, ok := r.announcements[announcementID]
	if !ok {
		return Announcement{}, ErrAnnouncementNotFound
	}
	nowText := now.UTC().Format(resourceDateLayout)
	announcement.Status = status
	announcement.UpdatedAt = nowText
	switch status {
	case AnnouncementStatusPublished:
		if announcement.PublishedAt == "" {
			announcement.PublishedAt = nowText
		}
	case AnnouncementStatusWithdrawn:
		announcement.WithdrawnAt = nowText
	case AnnouncementStatusArchived:
		announcement.ArchivedAt = nowText
	}
	r.announcements[announcement.ID] = announcement
	return announcement, nil
}

// DeleteAnnouncementDraft 删除草稿公告。
func (r *MemoryRepository) DeleteAnnouncementDraft(announcementID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	announcement, ok := r.announcements[announcementID]
	if !ok {
		return ErrAnnouncementNotFound
	}
	if announcement.Status != AnnouncementStatusDraft {
		return ErrAnnouncementDeleteDenied
	}
	delete(r.announcements, announcementID)
	return nil
}

// upsertAnnouncementRead 统一写入玩家公告阅读状态。
func (r *MemoryRepository) upsertAnnouncementRead(ctx AnnouncementPlayerContext, announcementID string, now time.Time, update func(state *AnnouncementReadState, nowText string)) (AnnouncementReadState, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.announcements[announcementID]; !ok {
		return AnnouncementReadState{}, ErrAnnouncementNotFound
	}
	key := announcementReadKey(announcementID, ctx.PlayerID)
	nowText := now.UTC().Format(resourceDateLayout)
	state := r.announcementReads[key]
	if state.AnnouncementID == "" {
		state = AnnouncementReadState{AnnouncementID: announcementID, PlayerID: ctx.PlayerID, AccountID: ctx.AccountID, CreatedAt: nowText}
	}
	update(&state, nowText)
	state.UpdatedAt = nowText
	r.announcementReads[key] = state
	return state, nil
}

func announcementReadKey(announcementID string, playerID string) string {
	return announcementID + ":" + playerID
}

func buildAnnouncementSummary(announcement Announcement, read AnnouncementReadState) AnnouncementSummary {
	publishedAt := announcement.PublishedAt
	if publishedAt == "" && announcement.Status == AnnouncementStatusScheduled {
		publishedAt = announcement.StartsAt
	}
	return AnnouncementSummary{
		ID:           announcement.ID,
		Title:        announcement.Title,
		Summary:      announcement.Summary,
		Type:         announcement.Type,
		Status:       announcement.Status,
		DisplayMode:  announcement.DisplayMode,
		Pinned:       announcement.Pinned,
		Priority:     announcement.Priority,
		ForcePopup:   announcement.ForcePopup,
		PublishedAt:  publishedAt,
		StartsAt:     announcement.StartsAt,
		EndsAt:       announcement.EndsAt,
		IsRead:       read.IsRead,
		IsPopupShown: read.IsPopupShown,
		IsDismissed:  read.IsDismissed,
	}
}

// BuildAnnouncementSummary 从公告和玩家阅读状态构建公告摘要。
func BuildAnnouncementSummary(announcement Announcement, read AnnouncementReadState) AnnouncementSummary {
	return buildAnnouncementSummary(announcement, read)
}

func announcementVisibleToPlayer(announcement Announcement, ctx AnnouncementPlayerContext, includeArchived bool, now time.Time) bool {
	if !announcementStatusVisible(announcement, includeArchived, now) {
		return false
	}
	if !timeWindowContains(announcement.StartsAt, announcement.EndsAt, now) {
		return false
	}
	return announcementTargetsMatch(announcement.Targets, ctx)
}

func announcementStatusVisible(announcement Announcement, includeArchived bool, now time.Time) bool {
	switch announcement.Status {
	case AnnouncementStatusPublished:
		return true
	case AnnouncementStatusArchived:
		return includeArchived
	default:
		return false
	}
}

// AnnouncementVisibleToPlayer 判断公告对指定玩家上下文是否可见。
func AnnouncementVisibleToPlayer(announcement Announcement, ctx AnnouncementPlayerContext, includeArchived bool, now time.Time) bool {
	return announcementVisibleToPlayer(announcement, ctx, includeArchived, now)
}

func timeWindowContains(startsAt string, endsAt string, now time.Time) bool {
	if t, err := time.Parse(resourceDateLayout, strings.TrimSpace(startsAt)); err == nil && now.Before(t) {
		return false
	}
	if t, err := time.Parse(resourceDateLayout, strings.TrimSpace(endsAt)); err == nil && now.After(t) {
		return false
	}
	return true
}

func announcementTargetsMatch(targets []AnnouncementTarget, ctx AnnouncementPlayerContext) bool {
	for _, target := range targets {
		switch target.Type {
		case AnnouncementTargetAll:
			return true
		case AnnouncementTargetPlayerIDs:
			if stringListContains(target.Value, ctx.PlayerID) {
				return true
			}
		case AnnouncementTargetAccountIDs:
			if stringListContains(target.Value, ctx.AccountID) {
				return true
			}
		case AnnouncementTargetFactions:
			if stringListContains(target.Value, ctx.Faction) {
				return true
			}
		case AnnouncementTargetLevelRange:
			if numberRangeContains(target.Value, ctx.Level) {
				return true
			}
		case AnnouncementTargetCreatedAtRange:
			if timeRangeContains(target.Value, ctx.CreatedAt) {
				return true
			}
		}
	}
	return false
}

func stringListContains(value any, needle string) bool {
	needle = strings.TrimSpace(needle)
	if needle == "" {
		return false
	}
	for _, item := range anyStringList(value) {
		if strings.TrimSpace(item) == needle {
			return true
		}
	}
	return false
}

func anyStringList(value any) []string {
	switch typed := value.(type) {
	case []string:
		return typed
	case []any:
		result := []string{}
		for _, item := range typed {
			if text, ok := item.(string); ok {
				result = append(result, text)
			}
		}
		return result
	case string:
		return []string{typed}
	default:
		raw, _ := json.Marshal(value)
		result := []string{}
		_ = json.Unmarshal(raw, &result)
		return result
	}
}

func numberRangeContains(value any, current int) bool {
	raw, _ := json.Marshal(value)
	var parsed struct {
		Min *int `json:"min"`
		Max *int `json:"max"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return false
	}
	if parsed.Min != nil && current < *parsed.Min {
		return false
	}
	if parsed.Max != nil && current > *parsed.Max {
		return false
	}
	return true
}

func timeRangeContains(value any, currentText string) bool {
	current, err := time.Parse(resourceDateLayout, currentText)
	if err != nil {
		return false
	}
	raw, _ := json.Marshal(value)
	var parsed struct {
		From string `json:"from"`
		To   string `json:"to"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return false
	}
	if t, err := time.Parse(resourceDateLayout, strings.TrimSpace(parsed.From)); err == nil && current.Before(t) {
		return false
	}
	if t, err := time.Parse(resourceDateLayout, strings.TrimSpace(parsed.To)); err == nil && current.After(t) {
		return false
	}
	return true
}

func paginateAnnouncementSummaries(items []AnnouncementSummary, page int, pageSize int) []AnnouncementSummary {
	start, end := pageBounds(len(items), page, pageSize)
	if start == end {
		return []AnnouncementSummary{}
	}
	return append([]AnnouncementSummary(nil), items[start:end]...)
}

func pageBounds(total int, page int, pageSize int) (int, int) {
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
