// Hero3 公告服务，负责公告创建、发布、展示窗口和玩家已读状态。
package game

import (
	"sort"
	"strings"
	"time"
	"unicode/utf8"
)

// ListAnnouncements 返回玩家当前可见公告和未读数量。
func (s *Service) ListAnnouncements(playerID string) (AnnouncementPage, error) {
	playerID = strings.TrimSpace(playerID)
	if playerID == "" {
		return AnnouncementPage{}, ErrPlayerNotFound
	}
	if _, err := s.repo.GetState(playerID); err != nil {
		return AnnouncementPage{}, err
	}

	announcements, err := s.repo.ListVisibleAnnouncements(playerID, time.Now())
	if err != nil {
		return AnnouncementPage{}, err
	}
	return AnnouncementPage{
		Announcements: announcements,
		Unread:        announcementUnreadCount(announcements),
	}, nil
}

// GetAnnouncement 返回玩家可见的单条公告详情。
func (s *Service) GetAnnouncement(playerID string, announcementID string) (Announcement, error) {
	playerID = strings.TrimSpace(playerID)
	announcementID = strings.TrimSpace(announcementID)
	if playerID == "" {
		return Announcement{}, ErrPlayerNotFound
	}
	if announcementID == "" {
		return Announcement{}, ErrAnnouncementNotFound
	}
	if _, err := s.repo.GetState(playerID); err != nil {
		return Announcement{}, err
	}

	announcements, err := s.repo.ListVisibleAnnouncements(playerID, time.Now())
	if err != nil {
		return Announcement{}, err
	}
	for _, announcement := range announcements {
		if announcement.ID == announcementID {
			return announcement, nil
		}
	}
	return Announcement{}, ErrAnnouncementNotFound
}

// MarkAnnouncementRead 标记玩家已读公告，并返回已读后的公告详情。
func (s *Service) MarkAnnouncementRead(playerID string, announcementID string) (Announcement, error) {
	announcement, err := s.GetAnnouncement(playerID, announcementID)
	if err != nil {
		return Announcement{}, err
	}
	if err := s.repo.MarkAnnouncementRead(strings.TrimSpace(playerID), announcement.ID, time.Now()); err != nil {
		return Announcement{}, err
	}
	announcement.Read = true
	return announcement, nil
}

// AdminListAnnouncements 返回 GM 后台公告列表。
func (s *Service) AdminListAnnouncements() ([]Announcement, error) {
	return s.repo.ListAdminAnnouncements()
}

// AdminCreateAnnouncement 创建草稿公告。
func (s *Service) AdminCreateAnnouncement(input AnnouncementInput) (Announcement, error) {
	now := time.Now()
	announcement, err := buildAnnouncement(input, now, AnnouncementStatusDraft)
	if err != nil {
		return Announcement{}, err
	}
	if err := s.repo.CreateAnnouncement(announcement); err != nil {
		return Announcement{}, err
	}
	return announcement, nil
}

// AdminUpdateAnnouncement 更新公告基础内容，保留原有状态和创建时间。
func (s *Service) AdminUpdateAnnouncement(announcementID string, input AnnouncementInput) (Announcement, error) {
	announcementID = strings.TrimSpace(announcementID)
	if announcementID == "" {
		return Announcement{}, ErrAnnouncementNotFound
	}
	current, err := s.repo.GetAnnouncementByID(announcementID)
	if err != nil {
		return Announcement{}, ErrAnnouncementNotFound
	}
	updated, err := normalizeAnnouncementInput(input, time.Now())
	if err != nil {
		return Announcement{}, err
	}
	current.Title = updated.Title
	current.Content = updated.Content
	current.Type = updated.Type
	current.Pinned = updated.Pinned
	current.Priority = updated.Priority
	current.StartsAt = updated.StartsAt
	current.EndsAt = updated.EndsAt
	current.UpdatedAt = updated.UpdatedAt
	if err := s.repo.UpdateAnnouncement(current); err != nil {
		return Announcement{}, err
	}
	return current, nil
}

// AdminPublishAnnouncement 发布公告。
func (s *Service) AdminPublishAnnouncement(announcementID string) (Announcement, error) {
	return s.updateAnnouncementStatus(announcementID, AnnouncementStatusPublished)
}

// AdminArchiveAnnouncement 归档或下架公告。
func (s *Service) AdminArchiveAnnouncement(announcementID string) (Announcement, error) {
	return s.updateAnnouncementStatus(announcementID, AnnouncementStatusArchived)
}

// AdminDeleteAnnouncement 软删除公告，保持玩家已读记录不受物理删除影响。
func (s *Service) AdminDeleteAnnouncement(announcementID string) (Announcement, error) {
	return s.AdminArchiveAnnouncement(announcementID)
}

// updateAnnouncementStatus 修改公告状态并刷新更新时间。
func (s *Service) updateAnnouncementStatus(announcementID string, status AnnouncementStatus) (Announcement, error) {
	announcementID = strings.TrimSpace(announcementID)
	if announcementID == "" {
		return Announcement{}, ErrAnnouncementNotFound
	}
	announcement, err := s.repo.GetAnnouncementByID(announcementID)
	if err != nil {
		return Announcement{}, ErrAnnouncementNotFound
	}
	announcement.Status = status
	announcement.UpdatedAt = time.Now().UTC().Format(resourceDateLayout)
	if err := s.repo.UpdateAnnouncement(announcement); err != nil {
		return Announcement{}, err
	}
	return announcement, nil
}

// buildAnnouncement 根据输入构造新公告。
func buildAnnouncement(input AnnouncementInput, now time.Time, status AnnouncementStatus) (Announcement, error) {
	announcement, err := normalizeAnnouncementInput(input, now)
	if err != nil {
		return Announcement{}, err
	}
	announcement.ID = "ann_" + randomID(12)
	announcement.Status = status
	announcement.CreatedAt = now.UTC().Format(resourceDateLayout)
	return announcement, nil
}

// normalizeAnnouncementInput 清理并校验公告输入。
func normalizeAnnouncementInput(input AnnouncementInput, now time.Time) (Announcement, error) {
	title := strings.TrimSpace(input.Title)
	content := strings.TrimSpace(input.Content)
	if title == "" || content == "" {
		return Announcement{}, ErrInvalidAnnouncement
	}
	if utf8.RuneCountInString(title) > 120 || utf8.RuneCountInString(content) > 10000 {
		return Announcement{}, ErrInvalidAnnouncement
	}

	announcementType := normalizeAnnouncementType(input.Type)
	startsAt, err := normalizeAnnouncementTime(input.StartsAt)
	if err != nil {
		return Announcement{}, ErrInvalidAnnouncement
	}
	endsAt, err := normalizeAnnouncementTime(input.EndsAt)
	if err != nil {
		return Announcement{}, ErrInvalidAnnouncement
	}
	if startsAt != "" && endsAt != "" {
		start, _ := time.Parse(time.RFC3339, startsAt)
		end, _ := time.Parse(time.RFC3339, endsAt)
		if start.After(end) {
			return Announcement{}, ErrInvalidAnnouncement
		}
	}

	return Announcement{
		Title:     title,
		Content:   content,
		Type:      announcementType,
		Status:    AnnouncementStatusDraft,
		Pinned:    input.Pinned,
		Priority:  input.Priority,
		StartsAt:  startsAt,
		EndsAt:    endsAt,
		UpdatedAt: now.UTC().Format(resourceDateLayout),
	}, nil
}

// normalizeAnnouncementType 规范公告类型，未知值回落为系统公告。
func normalizeAnnouncementType(announcementType AnnouncementType) AnnouncementType {
	switch announcementType {
	case AnnouncementTypeMaintenance, AnnouncementTypeEvent, AnnouncementTypeUpdate:
		return announcementType
	default:
		return AnnouncementTypeSystem
	}
}

// normalizeAnnouncementTime 解析公告时间，支持 RFC3339 和 datetime-local 格式。
func normalizeAnnouncementTime(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", nil
	}
	layouts := []string{
		time.RFC3339,
		"2006-01-02T15:04",
		"2006-01-02 15:04:05",
		"2006-01-02 15:04",
	}
	for _, layout := range layouts {
		if parsed, err := time.Parse(layout, value); err == nil {
			return parsed.UTC().Format(resourceDateLayout), nil
		}
	}
	return "", ErrInvalidAnnouncement
}

// isAnnouncementVisible 判断公告当前是否对玩家可见。
func isAnnouncementVisible(announcement Announcement, now time.Time) bool {
	if announcement.Status != AnnouncementStatusPublished {
		return false
	}
	if startsAt, ok := parseAnnouncementTime(announcement.StartsAt); ok && now.UTC().Before(startsAt) {
		return false
	}
	if endsAt, ok := parseAnnouncementTime(announcement.EndsAt); ok && !now.UTC().Before(endsAt) {
		return false
	}
	return true
}

// sortAnnouncements 按置顶、优先级和时间对公告排序。
func sortAnnouncements(items []Announcement) {
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].Pinned != items[j].Pinned {
			return items[i].Pinned
		}
		if items[i].Priority != items[j].Priority {
			return items[i].Priority > items[j].Priority
		}
		left := announcementSortTime(items[i])
		right := announcementSortTime(items[j])
		if !left.Equal(right) {
			return left.After(right)
		}
		return items[i].ID < items[j].ID
	})
}

// announcementSortTime 返回公告排序使用的展示时间。
func announcementSortTime(announcement Announcement) time.Time {
	if startsAt, ok := parseAnnouncementTime(announcement.StartsAt); ok {
		return startsAt
	}
	if createdAt, ok := parseAnnouncementTime(announcement.CreatedAt); ok {
		return createdAt
	}
	return time.Time{}
}

// parseAnnouncementTime 解析公告时间字符串。
func parseAnnouncementTime(value string) (time.Time, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}, false
	}
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return time.Time{}, false
	}
	return parsed.UTC(), true
}

// announcementUnreadCount 统计公告列表未读数量。
func announcementUnreadCount(items []Announcement) int {
	unread := 0
	for _, item := range items {
		if !item.Read {
			unread++
		}
	}
	return unread
}
