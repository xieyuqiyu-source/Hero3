// 本文件验证建筑和资源田格子回填校验工具在隔离 test_ MySQL 库上的行为。
package main

import (
	"context"
	"testing"
	"time"

	"hero3/internal/app/game"
)

// TestBackfillBuildingsAndResourceSlotsRepairsMissingRows 验证建筑和资源田格子回填能修复旧数据。
func TestBackfillBuildingsAndResourceSlotsRepairsMissingRows(t *testing.T) {
	dsn, db := openIsolatedDBToolDatabase(t, "buildings")
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	now := time.Now().UTC().Format(time.RFC3339)
	insertDBToolPlayerState(t, db, "dbtool_building_account", "dbtool_building_player", game.GameState{
		Player: game.Player{ID: "dbtool_building_player", Nickname: "建筑回填测试", Faction: "wei"},
		Resources: game.ResourceState{
			Items:    map[string]int{"wood": 100},
			Capacity: map[string]int{"wood": 1000},
		},
		Buildings: []game.Building{
			{ID: "wood_camp-1", Type: "wood_camp", Level: 3},
			{ID: "warehouse-1", Type: "warehouse", Level: 2},
		},
		ResourceSettledAt: now,
		ServerTime:        now,
	})

	buildingVerifyBefore, err := verifyPlayerBuildings(ctx, dsn)
	if err != nil {
		t.Fatalf("verify buildings before backfill: %v", err)
	}
	if buildingVerifyBefore.Mismatches == 0 {
		t.Fatalf("expected verify-buildings to find missing player_buildings rows")
	}
	slotVerifyBefore, err := verifyPlayerResourceSlots(ctx, dsn)
	if err != nil {
		t.Fatalf("verify resource slots before backfill: %v", err)
	}
	if slotVerifyBefore.Mismatches == 0 {
		t.Fatalf("expected verify-resource-slots to find missing player_resource_slots rows")
	}

	if _, err := backfillPlayerBuildings(ctx, dsn); err != nil {
		t.Fatalf("backfill buildings: %v", err)
	}
	if _, err := backfillPlayerResourceSlots(ctx, dsn); err != nil {
		t.Fatalf("backfill resource slots: %v", err)
	}
	buildingVerifyAfter, err := verifyPlayerBuildings(ctx, dsn)
	if err != nil {
		t.Fatalf("verify buildings after backfill: %v", err)
	}
	if buildingVerifyAfter.Mismatches != 0 {
		t.Fatalf("expected backfill to repair buildings, got mismatches=%d", buildingVerifyAfter.Mismatches)
	}
	slotVerifyAfter, err := verifyPlayerResourceSlots(ctx, dsn)
	if err != nil {
		t.Fatalf("verify resource slots after backfill: %v", err)
	}
	if slotVerifyAfter.Mismatches != 0 {
		t.Fatalf("expected backfill to repair resource slots, got mismatches=%d", slotVerifyAfter.Mismatches)
	}
}
