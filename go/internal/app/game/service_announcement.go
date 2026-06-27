// 本文件实现公告系统应用服务，负责公告管理、可见性判断和玩家阅读状态。
package game

import (
	"errors"
	"sort"
	"strings"
	"time"
)

var (
	ErrAnnouncementNotFound     = errors.New("announcement not found")
	ErrInvalidAnnouncement      = errors.New("invalid announcement")
	ErrAnnouncementDeleteDenied = errors.New("only draft announcement can be deleted")
	ErrAnnouncementNotVisible   = errors.New("announcement not visible")
)

// ListAnnouncements 返回当前玩家可见公告摘要。
func (s *Service) ListAnnouncements(filter AnnouncementListFilter) (AnnouncementPage, error) {
	filter.PlayerID = strings.TrimSpace(filter.PlayerID)
	if filter.PlayerID == "" {
		return AnnouncementPage{}, ErrPlayerNotFound
	}
	filter.Page, filter.PageSize = normalizePage(filter.Page, filter.PageSize, 20)
	now := time.Now()
	if err := s.repo.PromoteDueScheduledAnnouncements(now); err != nil {
		return AnnouncementPage{}, err
	}
	ctx, err := s.repo.GetAnnouncementPlayerContext(filter.PlayerID)
	if err != nil {
		return AnnouncementPage{}, err
	}
	if filter.AccountID != "" && ctx.AccountID != filter.AccountID {
		return AnnouncementPage{}, ErrPlayerNotFound
	}
	repoFilter := filter
	repoFilter.Page = 1
	repoFilter.PageSize = 0
	items, total, err := s.repo.ListVisibleAnnouncements(ctx, repoFilter, now)
	if err != nil {
		return AnnouncementPage{}, err
	}
	unread := false
	for _, item := range items {
		if !item.IsRead {
			unread = true
			break
		}
	}
	return AnnouncementPage{Items: paginateAnnouncementSummaries(items, filter.Page, filter.PageSize), Page: filter.Page, PageSize: filter.PageSize, Total: total, Unread: unread}, nil
}

// GetAnnouncementDetail 返回公告详情并验证玩家可见性。
func (s *Service) GetAnnouncementDetail(playerID string, accountID string, announcementID string) (AnnouncementDetail, error) {
	playerID = strings.TrimSpace(playerID)
	announcementID = strings.TrimSpace(announcementID)
	if playerID == "" {
		return AnnouncementDetail{}, ErrPlayerNotFound
	}
	now := time.Now()
	if err := s.repo.PromoteDueScheduledAnnouncements(now); err != nil {
		return AnnouncementDetail{}, err
	}
	ctx, err := s.repo.GetAnnouncementPlayerContext(playerID)
	if err != nil {
		return AnnouncementDetail{}, err
	}
	if accountID != "" && ctx.AccountID != accountID {
		return AnnouncementDetail{}, ErrPlayerNotFound
	}
	return s.repo.GetVisibleAnnouncementDetail(ctx, announcementID, now)
}

// MarkAnnouncementRead 幂等标记公告已读。
func (s *Service) MarkAnnouncementRead(playerID string, accountID string, announcementID string) (AnnouncementReadState, error) {
	ctx, err := s.validAnnouncementContext(playerID, accountID, announcementID)
	if err != nil {
		return AnnouncementReadState{}, err
	}
	return s.repo.MarkAnnouncementRead(ctx, announcementID, time.Now())
}

// MarkAnnouncementPopupShown 幂等记录公告弹窗已展示。
func (s *Service) MarkAnnouncementPopupShown(playerID string, accountID string, announcementID string) (AnnouncementReadState, error) {
	ctx, err := s.validAnnouncementContext(playerID, accountID, announcementID)
	if err != nil {
		return AnnouncementReadState{}, err
	}
	return s.repo.MarkAnnouncementPopupShown(ctx, announcementID, time.Now())
}

// DismissAnnouncement 幂等记录公告弹窗已关闭。
func (s *Service) DismissAnnouncement(playerID string, accountID string, announcementID string) (AnnouncementReadState, error) {
	ctx, err := s.validAnnouncementContext(playerID, accountID, announcementID)
	if err != nil {
		return AnnouncementReadState{}, err
	}
	return s.repo.DismissAnnouncement(ctx, announcementID, time.Now())
}

// ListAnnouncementPopups 返回当前玩家待弹窗公告队列。
func (s *Service) ListAnnouncementPopups(playerID string, accountID string) ([]AnnouncementSummary, error) {
	// 弹窗队列独立分页暂不开发：当前复用公告第一页 20 条作为轻量队列来源。
	page, err := s.ListAnnouncements(AnnouncementListFilter{PlayerID: playerID, AccountID: accountID, Page: 1, PageSize: 20})
	if err != nil {
		return nil, err
	}
	items := []AnnouncementSummary{}
	for _, item := range page.Items {
		if item.ForcePopup && item.DisplayMode == AnnouncementDisplayPopup && !item.IsDismissed {
			items = append(items, item)
		}
	}
	sortAnnouncementSummaries(items)
	return items, nil
}

// ListAdminAnnouncements 返回 GM 公告列表。
func (s *Service) ListAdminAnnouncements(filter AdminAnnouncementFilter) (AdminAnnouncementPage, error) {
	filter.Page, filter.PageSize = normalizePage(filter.Page, filter.PageSize, 20)
	if err := s.repo.PromoteDueScheduledAnnouncements(time.Now()); err != nil {
		return AdminAnnouncementPage{}, err
	}
	items, total, err := s.repo.ListAdminAnnouncements(filter)
	if err != nil {
		return AdminAnnouncementPage{}, err
	}
	return AdminAnnouncementPage{Items: items, Page: filter.Page, PageSize: filter.PageSize, Total: total}, nil
}

// CreateAnnouncement 创建草稿或直接创建指定状态公告。
func (s *Service) CreateAnnouncement(req SaveAnnouncementRequest) (Announcement, error) {
	now := time.Now()
	announcement, err := buildAnnouncementFromRequest("ann_"+randomID(12), req, now)
	if err != nil {
		return Announcement{}, err
	}
	if announcement.Status == "" {
		announcement.Status = AnnouncementStatusDraft
	}
	if announcement.Status == AnnouncementStatusPublished && announcement.PublishedAt == "" {
		announcement.PublishedAt = now.UTC().Format(resourceDateLayout)
	}
	return s.repo.SaveAnnouncement(announcement)
}

// UpdateAnnouncement 编辑公告正文和展示配置。
func (s *Service) UpdateAnnouncement(announcementID string, req SaveAnnouncementRequest) (Announcement, error) {
	current, err := s.repo.GetAdminAnnouncement(strings.TrimSpace(announcementID))
	if err != nil {
		return Announcement{}, err
	}
	next, err := buildAnnouncementFromRequest(current.ID, req, time.Now())
	if err != nil {
		return Announcement{}, err
	}
	next.Status = firstNonEmpty(strings.TrimSpace(req.Status), current.Status)
	next.PublishedAt = current.PublishedAt
	next.WithdrawnAt = current.WithdrawnAt
	next.ArchivedAt = current.ArchivedAt
	next.CreatedAt = current.CreatedAt
	return s.repo.SaveAnnouncement(next)
}

// PublishAnnouncement 立即发布公告。
func (s *Service) PublishAnnouncement(announcementID string) (Announcement, error) {
	return s.repo.UpdateAnnouncementStatus(strings.TrimSpace(announcementID), AnnouncementStatusPublished, time.Now())
}

// ScheduleAnnouncement 把公告设置为定时发布。
func (s *Service) ScheduleAnnouncement(announcementID string, startsAt string) (Announcement, error) {
	announcement, err := s.repo.GetAdminAnnouncement(strings.TrimSpace(announcementID))
	if err != nil {
		return Announcement{}, err
	}
	if strings.TrimSpace(startsAt) != "" {
		announcement.StartsAt = startsAt
	}
	announcement.Status = AnnouncementStatusScheduled
	announcement.UpdatedAt = time.Now().UTC().Format(resourceDateLayout)
	announcement.PublishedAt = ""
	return s.repo.SaveAnnouncement(announcement)
}

// WithdrawAnnouncement 撤回公告。
func (s *Service) WithdrawAnnouncement(announcementID string) (Announcement, error) {
	return s.repo.UpdateAnnouncementStatus(strings.TrimSpace(announcementID), AnnouncementStatusWithdrawn, time.Now())
}

// ArchiveAnnouncement 归档公告。
func (s *Service) ArchiveAnnouncement(announcementID string) (Announcement, error) {
	return s.repo.UpdateAnnouncementStatus(strings.TrimSpace(announcementID), AnnouncementStatusArchived, time.Now())
}

// DeleteAnnouncementDraft 删除草稿公告。
func (s *Service) DeleteAnnouncementDraft(announcementID string) error {
	return s.repo.DeleteAnnouncementDraft(strings.TrimSpace(announcementID))
}

// validAnnouncementContext 校验玩家上下文和公告可见性。
func (s *Service) validAnnouncementContext(playerID string, accountID string, announcementID string) (AnnouncementPlayerContext, error) {
	playerID = strings.TrimSpace(playerID)
	announcementID = strings.TrimSpace(announcementID)
	if playerID == "" {
		return AnnouncementPlayerContext{}, ErrPlayerNotFound
	}
	ctx, err := s.repo.GetAnnouncementPlayerContext(playerID)
	if err != nil {
		return AnnouncementPlayerContext{}, err
	}
	if accountID != "" && ctx.AccountID != accountID {
		return AnnouncementPlayerContext{}, ErrPlayerNotFound
	}
	now := time.Now()
	if err := s.repo.PromoteDueScheduledAnnouncements(now); err != nil {
		return AnnouncementPlayerContext{}, err
	}
	if _, err := s.repo.GetVisibleAnnouncementDetail(ctx, announcementID, now); err != nil {
		return AnnouncementPlayerContext{}, err
	}
	return ctx, nil
}

// buildAnnouncementFromRequest 标准化 GM 保存公告请求。
func buildAnnouncementFromRequest(id string, req SaveAnnouncementRequest, now time.Time) (Announcement, error) {
	title := strings.TrimSpace(req.Title)
	content := strings.TrimSpace(req.Content)
	if title == "" || content == "" {
		return Announcement{}, ErrInvalidAnnouncement
	}
	announcementType := normalizeAnnouncementType(req.Type)
	status := normalizeAnnouncementStatus(req.Status)
	displayMode := normalizeAnnouncementDisplayMode(req.DisplayMode)
	nowText := now.UTC().Format(resourceDateLayout)
	targets := normalizeAnnouncementTargets(req.Targets)
	return Announcement{
		ID:          id,
		Title:       title,
		Summary:     strings.TrimSpace(req.Summary),
		Content:     content,
		Type:        announcementType,
		Status:      status,
		DisplayMode: displayMode,
		Pinned:      req.Pinned,
		Priority:    req.Priority,
		ForcePopup:  req.ForcePopup,
		StartsAt:    strings.TrimSpace(req.StartsAt),
		EndsAt:      strings.TrimSpace(req.EndsAt),
		CreatedAt:   nowText,
		UpdatedAt:   nowText,
		Targets:     targets,
	}, nil
}

// normalizeAnnouncementType 规范公告类型。
func normalizeAnnouncementType(value string) string {
	switch strings.TrimSpace(value) {
	case AnnouncementTypeMaintenance, AnnouncementTypeUpdate, AnnouncementTypeActivity, AnnouncementTypeCompensation, AnnouncementTypeEmergency:
		return strings.TrimSpace(value)
	default:
		return AnnouncementTypeSystem
	}
}

// normalizeAnnouncementStatus 规范公告状态。
func normalizeAnnouncementStatus(value string) string {
	switch strings.TrimSpace(value) {
	case AnnouncementStatusScheduled, AnnouncementStatusPublished, AnnouncementStatusWithdrawn, AnnouncementStatusArchived:
		return strings.TrimSpace(value)
	default:
		return AnnouncementStatusDraft
	}
}

// normalizeAnnouncementDisplayMode 规范公告展示模式。
func normalizeAnnouncementDisplayMode(value string) string {
	switch strings.TrimSpace(value) {
	case AnnouncementDisplayPopup, AnnouncementDisplayBanner:
		return strings.TrimSpace(value)
	default:
		return AnnouncementDisplayCenterOnly
	}
}

// normalizeAnnouncementTargets 规范公告投放规则；未配置时不默认全体。
func normalizeAnnouncementTargets(targets []AnnouncementTarget) []AnnouncementTarget {
	result := []AnnouncementTarget{}
	for _, target := range targets {
		target.Type = strings.TrimSpace(target.Type)
		if target.Type == "" {
			continue
		}
		result = append(result, target)
	}
	return result
}

// normalizePage 规范分页参数。
func normalizePage(page int, pageSize int, defaultPageSize int) (int, int) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = defaultPageSize
	}
	if pageSize > 50 {
		pageSize = 50
	}
	return page, pageSize
}

// sortAnnouncementSummaries 按置顶、优先级和发布时间排序。
func sortAnnouncementSummaries(items []AnnouncementSummary) {
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].Pinned != items[j].Pinned {
			return items[i].Pinned
		}
		if items[i].Priority != items[j].Priority {
			return items[i].Priority > items[j].Priority
		}
		return items[i].PublishedAt > items[j].PublishedAt
	})
}

// SortAnnouncementSummaries 按置顶、优先级和发布时间排序公告摘要。
func SortAnnouncementSummaries(items []AnnouncementSummary) {
	sortAnnouncementSummaries(items)
}
