// 本文件验证公告系统 MySQL 持久化的投放规则兼容行为。
package storage

import (
	"testing"
	"time"

	"hero3/internal/app/game"
)

// TestMySQLAnnouncementAllTargetWithNullValue 验证全体投放的 NULL JSON 不会导致玩家公告列表读取失败。
func TestMySQLAnnouncementAllTargetWithNullValue(t *testing.T) {
	repo, db := openResourceAuthorityTestRepository(t)
	account, state := createResourceAuthorityPlayer(t, repo, "announcement_all_target")
	now := time.Now().UTC()
	announcement := game.Announcement{
		ID:          "it_ann_all_" + now.Format("20060102150405"),
		Title:       "全体公告",
		Summary:     "全体摘要",
		Content:     "全体正文",
		Type:        game.AnnouncementTypeSystem,
		Status:      game.AnnouncementStatusPublished,
		DisplayMode: game.AnnouncementDisplayCenterOnly,
		PublishedAt: now.Format(time.RFC3339),
		CreatedAt:   now.Format(time.RFC3339),
		UpdatedAt:   now.Format(time.RFC3339),
		Targets:     []game.AnnouncementTarget{{Type: game.AnnouncementTargetAll}},
	}
	if _, err := repo.SaveAnnouncement(announcement); err != nil {
		t.Fatalf("SaveAnnouncement failed: %v", err)
	}
	t.Cleanup(func() {
		_, _ = db.Exec(`DELETE FROM announcement_reads WHERE announcement_id = ?`, announcement.ID)
		_, _ = db.Exec(`DELETE FROM announcement_targets WHERE announcement_id = ?`, announcement.ID)
		_, _ = db.Exec(`DELETE FROM announcements WHERE id = ?`, announcement.ID)
	})

	ctx := game.AnnouncementPlayerContext{
		PlayerID:  state.Player.ID,
		AccountID: account.ID,
		Faction:   state.Player.Faction,
		CreatedAt: now.Format(time.RFC3339),
	}
	items, total, err := repo.ListVisibleAnnouncements(ctx, game.AnnouncementListFilter{PlayerID: state.Player.ID}, now)
	if err != nil {
		t.Fatalf("ListVisibleAnnouncements failed: %v", err)
	}
	if total == 0 || len(items) == 0 {
		t.Fatalf("expected visible announcement, got total=%d items=%+v", total, items)
	}
	found := false
	for _, item := range items {
		if item.ID == announcement.ID {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected announcement %s in visible items: %+v", announcement.ID, items)
	}
}
