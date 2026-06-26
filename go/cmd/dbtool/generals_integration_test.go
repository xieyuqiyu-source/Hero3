// 本文件验证武将权威表回填校验工具在隔离 test_ MySQL 库上的行为。
package main

import (
	"context"
	"testing"
	"time"

	"hero3/internal/app/game"
)

// TestBackfillGeneralsRepairsMissingRows 验证武将回填能修复旧单武将存档。
func TestBackfillGeneralsRepairsMissingRows(t *testing.T) {
	dsn, db := openIsolatedDBToolDatabase(t, "generals")
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	now := time.Now().UTC()
	state := game.GameState{
		Player: game.Player{ID: "dbtool_generals_player", Nickname: "武将回填测试", Faction: "wei"},
		Resources: game.ResourceState{
			Items:    map[string]int{"wood": 100},
			Capacity: map[string]int{"wood": 1000},
		},
		General: &game.General{
			ID:    "caocao",
			Name:  "曹操",
			Level: 7,
			Exp:   1234,
			Stats: map[string]int{"force": 2},
			Buffs: map[string]float64{},
		},
		ResourceSettledAt: now.Format(time.RFC3339),
		ServerTime:        now.Format(time.RFC3339),
	}
	game.EnsureGeneralRoster(&state, now)
	insertDBToolPlayerState(t, db, "dbtool_generals_account", "dbtool_generals_player", state)

	verifyBefore, err := verifyPlayerGenerals(ctx, dsn)
	if err != nil {
		t.Fatalf("verify generals before backfill: %v", err)
	}
	if verifyBefore.Mismatches == 0 {
		t.Fatalf("expected verify-generals to find missing player_generals rows")
	}

	if _, err := backfillPlayerGenerals(ctx, dsn); err != nil {
		t.Fatalf("backfill generals: %v", err)
	}
	verifyAfter, err := verifyPlayerGenerals(ctx, dsn)
	if err != nil {
		t.Fatalf("verify generals after backfill: %v", err)
	}
	if verifyAfter.Mismatches != 0 {
		t.Fatalf("expected backfill to repair generals, got mismatches=%d", verifyAfter.Mismatches)
	}
	if verifyAfter.ActualGenerals != 1 || verifyAfter.ActualAssignments != 1 {
		t.Fatalf("expected 1 general and 1 assignment, got generals=%d assignments=%d", verifyAfter.ActualGenerals, verifyAfter.ActualAssignments)
	}
}
