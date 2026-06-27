// 本文件测试公告系统应用服务的生命周期、投放、已读和弹窗行为。
package game

import (
	"testing"
	"time"
)

// newAnnouncementTestService 构造带玩家的公告测试服务。
func newAnnouncementTestService(t *testing.T) (*Service, *MemoryRepository, string) {
	t.Helper()
	now := time.Date(2026, 6, 27, 10, 0, 0, 0, time.UTC)
	repo := NewMemoryRepository()
	account := Account{ID: "account_announcement", Username: "announcement", PasswordHash: "hash", CreatedAt: now}
	if err := repo.CreateAccount(account); err != nil {
		t.Fatalf("CreateAccount failed: %v", err)
	}
	playerID := "player_announcement"
	state := newPlayerState(playerID, "公告测试", "wei", "caocao", now)
	if err := repo.CreatePlayer(account.ID, state, now); err != nil {
		t.Fatalf("CreatePlayer failed: %v", err)
	}
	return NewServiceWithRepository(repo), repo, playerID
}

// TestAnnouncementTargetsRequireExplicitAll 验证未配置投放规则时公告默认不可见。
func TestAnnouncementTargetsRequireExplicitAll(t *testing.T) {
	svc, _, playerID := newAnnouncementTestService(t)
	announcement, err := svc.CreateAnnouncement(SaveAnnouncementRequest{
		Title:   "未投放公告",
		Content: "没有投放范围时，玩家不应该看到。",
		Status:  AnnouncementStatusPublished,
	})
	if err != nil {
		t.Fatalf("CreateAnnouncement failed: %v", err)
	}

	page, err := svc.ListAnnouncements(AnnouncementListFilter{PlayerID: playerID, Page: 1, PageSize: 20})
	if err != nil {
		t.Fatalf("ListAnnouncements failed: %v", err)
	}
	if page.Total != 0 {
		t.Fatalf("expected no visible announcements, got %d", page.Total)
	}
	if _, err := svc.GetAnnouncementDetail(playerID, "", announcement.ID); err != ErrAnnouncementNotVisible {
		t.Fatalf("expected ErrAnnouncementNotVisible, got %v", err)
	}
}

// TestAnnouncementUnreadUsesAllVisibleItems 验证未读红点不会只看当前分页。
func TestAnnouncementUnreadUsesAllVisibleItems(t *testing.T) {
	svc, _, playerID := newAnnouncementTestService(t)
	first, err := svc.CreateAnnouncement(SaveAnnouncementRequest{
		Title:    "第一条",
		Content:  "第一条内容",
		Status:   AnnouncementStatusPublished,
		Priority: 20,
		Targets:  []AnnouncementTarget{{Type: AnnouncementTargetAll}},
	})
	if err != nil {
		t.Fatalf("CreateAnnouncement first failed: %v", err)
	}
	if _, err := svc.CreateAnnouncement(SaveAnnouncementRequest{
		Title:    "第二条",
		Content:  "第二条内容",
		Status:   AnnouncementStatusPublished,
		Priority: 10,
		Targets:  []AnnouncementTarget{{Type: AnnouncementTargetAll}},
	}); err != nil {
		t.Fatalf("CreateAnnouncement second failed: %v", err)
	}
	if _, err := svc.MarkAnnouncementRead(playerID, "", first.ID); err != nil {
		t.Fatalf("MarkAnnouncementRead failed: %v", err)
	}

	page, err := svc.ListAnnouncements(AnnouncementListFilter{PlayerID: playerID, Page: 1, PageSize: 1})
	if err != nil {
		t.Fatalf("ListAnnouncements failed: %v", err)
	}
	if len(page.Items) != 1 || !page.Items[0].IsRead {
		t.Fatalf("expected first page to contain the read announcement, got %+v", page.Items)
	}
	if !page.Unread {
		t.Fatal("expected unread flag to include announcements outside current page")
	}
}

// TestAnnouncementPopupReadStateIsIdempotent 验证弹窗展示、关闭和已读状态可以重复写入。
func TestAnnouncementPopupReadStateIsIdempotent(t *testing.T) {
	svc, _, playerID := newAnnouncementTestService(t)
	announcement, err := svc.CreateAnnouncement(SaveAnnouncementRequest{
		Title:       "强弹公告",
		Summary:     "摘要",
		Content:     "强弹内容",
		Status:      AnnouncementStatusPublished,
		DisplayMode: AnnouncementDisplayPopup,
		ForcePopup:  true,
		Targets:     []AnnouncementTarget{{Type: AnnouncementTargetAll}},
	})
	if err != nil {
		t.Fatalf("CreateAnnouncement failed: %v", err)
	}
	popups, err := svc.ListAnnouncementPopups(playerID, "")
	if err != nil {
		t.Fatalf("ListAnnouncementPopups failed: %v", err)
	}
	if len(popups) != 1 || popups[0].ID != announcement.ID {
		t.Fatalf("expected popup announcement, got %+v", popups)
	}
	if _, err := svc.MarkAnnouncementPopupShown(playerID, "", announcement.ID); err != nil {
		t.Fatalf("MarkAnnouncementPopupShown failed: %v", err)
	}
	if _, err := svc.DismissAnnouncement(playerID, "", announcement.ID); err != nil {
		t.Fatalf("DismissAnnouncement failed: %v", err)
	}
	if _, err := svc.DismissAnnouncement(playerID, "", announcement.ID); err != nil {
		t.Fatalf("second DismissAnnouncement failed: %v", err)
	}
	popups, err = svc.ListAnnouncementPopups(playerID, "")
	if err != nil {
		t.Fatalf("ListAnnouncementPopups after dismiss failed: %v", err)
	}
	if len(popups) != 0 {
		t.Fatalf("expected dismissed popup to disappear, got %+v", popups)
	}
}

// TestAnnouncementAdminLifecycle 验证 GM 公告创建、编辑、发布、撤回、归档和草稿删除。
func TestAnnouncementAdminLifecycle(t *testing.T) {
	svc, _, _ := newAnnouncementTestService(t)
	announcement, err := svc.CreateAnnouncement(SaveAnnouncementRequest{
		Title:   "草稿公告",
		Content: "草稿内容",
		Targets: []AnnouncementTarget{{Type: AnnouncementTargetAll}},
	})
	if err != nil {
		t.Fatalf("CreateAnnouncement failed: %v", err)
	}
	if announcement.Status != AnnouncementStatusDraft {
		t.Fatalf("expected draft, got %s", announcement.Status)
	}
	updated, err := svc.UpdateAnnouncement(announcement.ID, SaveAnnouncementRequest{
		Title:   "编辑后的公告",
		Summary: "编辑摘要",
		Content: "编辑后的内容",
		Targets: []AnnouncementTarget{{Type: AnnouncementTargetAll}},
	})
	if err != nil {
		t.Fatalf("UpdateAnnouncement failed: %v", err)
	}
	if updated.Title != "编辑后的公告" || updated.Status != AnnouncementStatusDraft {
		t.Fatalf("unexpected updated announcement: %+v", updated)
	}
	published, err := svc.PublishAnnouncement(announcement.ID)
	if err != nil {
		t.Fatalf("PublishAnnouncement failed: %v", err)
	}
	if published.Status != AnnouncementStatusPublished || published.PublishedAt == "" {
		t.Fatalf("expected published status with publishedAt, got %+v", published)
	}
	if err := svc.DeleteAnnouncementDraft(announcement.ID); err != ErrAnnouncementDeleteDenied {
		t.Fatalf("expected ErrAnnouncementDeleteDenied, got %v", err)
	}
	withdrawn, err := svc.WithdrawAnnouncement(announcement.ID)
	if err != nil {
		t.Fatalf("WithdrawAnnouncement failed: %v", err)
	}
	if withdrawn.Status != AnnouncementStatusWithdrawn || withdrawn.WithdrawnAt == "" {
		t.Fatalf("expected withdrawn status, got %+v", withdrawn)
	}
	archived, err := svc.ArchiveAnnouncement(announcement.ID)
	if err != nil {
		t.Fatalf("ArchiveAnnouncement failed: %v", err)
	}
	if archived.Status != AnnouncementStatusArchived || archived.ArchivedAt == "" {
		t.Fatalf("expected archived status, got %+v", archived)
	}

	draft, err := svc.CreateAnnouncement(SaveAnnouncementRequest{Title: "可删除草稿", Content: "草稿内容"})
	if err != nil {
		t.Fatalf("CreateAnnouncement draft failed: %v", err)
	}
	if err := svc.DeleteAnnouncementDraft(draft.ID); err != nil {
		t.Fatalf("DeleteAnnouncementDraft failed: %v", err)
	}
}

// TestArchivedAnnouncementVisibleOnlyWhenIncluded 验证历史公告只在请求历史时可见。
func TestArchivedAnnouncementVisibleOnlyWhenIncluded(t *testing.T) {
	svc, _, playerID := newAnnouncementTestService(t)
	announcement, err := svc.CreateAnnouncement(SaveAnnouncementRequest{
		Title:   "历史公告",
		Content: "历史内容",
		Status:  AnnouncementStatusPublished,
		Targets: []AnnouncementTarget{{Type: AnnouncementTargetAll}},
	})
	if err != nil {
		t.Fatalf("CreateAnnouncement failed: %v", err)
	}
	if _, err := svc.ArchiveAnnouncement(announcement.ID); err != nil {
		t.Fatalf("ArchiveAnnouncement failed: %v", err)
	}
	page, err := svc.ListAnnouncements(AnnouncementListFilter{PlayerID: playerID, Page: 1, PageSize: 20})
	if err != nil {
		t.Fatalf("ListAnnouncements failed: %v", err)
	}
	if page.Total != 0 {
		t.Fatalf("expected archived announcement hidden by default, got %d", page.Total)
	}
	page, err = svc.ListAnnouncements(AnnouncementListFilter{PlayerID: playerID, IncludeArchived: true, Page: 1, PageSize: 20})
	if err != nil {
		t.Fatalf("ListAnnouncements include archived failed: %v", err)
	}
	if page.Total != 1 || page.Items[0].ID != announcement.ID {
		t.Fatalf("expected archived announcement visible, got %+v", page.Items)
	}
}

// TestAnnouncementTargetsMatchSupportedScopes 验证玩家、账号、阵营、等级和创建时间投放命中规则。
func TestAnnouncementTargetsMatchSupportedScopes(t *testing.T) {
	now := time.Date(2026, 6, 27, 10, 0, 0, 0, time.UTC)
	ctx := AnnouncementPlayerContext{
		PlayerID:  "player_target",
		AccountID: "account_target",
		Faction:   "wei",
		Level:     12,
		CreatedAt: now.Add(-24 * time.Hour).UTC().Format(resourceDateLayout),
	}
	cases := []AnnouncementTarget{
		{Type: AnnouncementTargetPlayerIDs, Value: []string{"player_target"}},
		{Type: AnnouncementTargetAccountIDs, Value: []string{"account_target"}},
		{Type: AnnouncementTargetFactions, Value: []string{"wei"}},
		{Type: AnnouncementTargetLevelRange, Value: map[string]any{"min": 10, "max": 20}},
		{Type: AnnouncementTargetCreatedAtRange, Value: map[string]any{"from": now.Add(-48 * time.Hour).UTC().Format(resourceDateLayout), "to": now.UTC().Format(resourceDateLayout)}},
	}
	for _, target := range cases {
		announcement := Announcement{ID: "ann_target", Status: AnnouncementStatusPublished, Targets: []AnnouncementTarget{target}}
		if !AnnouncementVisibleToPlayer(announcement, ctx, false, now) {
			t.Fatalf("expected target %+v to match", target)
		}
	}
}

// TestScheduledAnnouncementPromotesBeforeVisible 验证定时公告到点后先转为已发布，再对玩家可见。
func TestScheduledAnnouncementPromotesBeforeVisible(t *testing.T) {
	svc, repo, playerID := newAnnouncementTestService(t)
	now := time.Date(2026, 6, 27, 10, 0, 0, 0, time.UTC)
	ctx := AnnouncementPlayerContext{PlayerID: playerID}
	announcement := Announcement{
		ID:       "ann_scheduled",
		Status:   AnnouncementStatusScheduled,
		StartsAt: now.Add(-time.Minute).Format(resourceDateLayout),
		Targets:  []AnnouncementTarget{{Type: AnnouncementTargetAll}},
	}
	if AnnouncementVisibleToPlayer(announcement, ctx, false, now) {
		t.Fatal("expected raw scheduled announcement to stay hidden before promotion")
	}
	startsAt := time.Now().Add(-time.Minute).UTC().Format(resourceDateLayout)
	scheduled, err := svc.CreateAnnouncement(SaveAnnouncementRequest{
		Title:    "定时公告",
		Content:  "到点发布内容",
		StartsAt: startsAt,
		Targets:  []AnnouncementTarget{{Type: AnnouncementTargetAll}},
	})
	if err != nil {
		t.Fatalf("CreateAnnouncement failed: %v", err)
	}
	if _, err := svc.ScheduleAnnouncement(scheduled.ID, startsAt); err != nil {
		t.Fatalf("ScheduleAnnouncement failed: %v", err)
	}
	page, err := svc.ListAnnouncements(AnnouncementListFilter{PlayerID: playerID, Page: 1, PageSize: 20})
	if err != nil {
		t.Fatalf("ListAnnouncements failed: %v", err)
	}
	if page.Total != 1 || page.Items[0].Status != AnnouncementStatusPublished {
		t.Fatalf("expected scheduled announcement promoted and visible, got %+v", page.Items)
	}
	stored, err := repo.GetAdminAnnouncement(scheduled.ID)
	if err != nil {
		t.Fatalf("GetAdminAnnouncement failed: %v", err)
	}
	if stored.Status != AnnouncementStatusPublished || stored.PublishedAt == "" {
		t.Fatalf("expected stored announcement promoted, got %+v", stored)
	}
}
