// 本文件验证 Buff 权威表回填校验工具在隔离 test_ MySQL 库上的行为。
package main

import (
	"context"
	"testing"
	"time"

	"hero3/internal/app/game"
)

// TestBackfillBuffsRepairsMissingRows 验证 Buff 回填能修复旧 JSON 快照。
func TestBackfillBuffsRepairsMissingRows(t *testing.T) {
	dsn, db := openIsolatedDBToolDatabase(t, "buffs")
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	now := time.Now().UTC()
	insertDBToolPlayerState(t, db, "dbtool_buffs_account", "dbtool_buffs_player", game.GameState{
		Player: game.Player{ID: "dbtool_buffs_player", Nickname: "Buff 回填测试", Faction: "wei"},
		Resources: game.ResourceState{
			Items:    map[string]int{"wood": 100},
			Capacity: map[string]int{"wood": 1000},
		},
		Buffs: []game.Buff{
			{ID: "buff_dbtool", Source: "gm", Key: game.StatAttackBonus, Value: 0.1, Mode: "percentAdd", CreatedAt: now.Format(time.RFC3339), Note: "test"},
		},
		ResourceSettledAt: now.Format(time.RFC3339),
		ServerTime:        now.Format(time.RFC3339),
	})

	verifyBefore, err := verifyPlayerBuffs(ctx, dsn)
	if err != nil {
		t.Fatalf("verify buffs before backfill: %v", err)
	}
	if verifyBefore.Mismatches == 0 {
		t.Fatalf("expected verify-buffs to find missing player_buffs rows")
	}
	if _, err := backfillPlayerBuffs(ctx, dsn); err != nil {
		t.Fatalf("backfill buffs: %v", err)
	}
	verifyAfter, err := verifyPlayerBuffs(ctx, dsn)
	if err != nil {
		t.Fatalf("verify buffs after backfill: %v", err)
	}
	if verifyAfter.Mismatches != 0 || verifyAfter.ActualRows != 1 {
		t.Fatalf("expected backfill to repair buffs, got mismatches=%d rows=%d", verifyAfter.Mismatches, verifyAfter.ActualRows)
	}
}
