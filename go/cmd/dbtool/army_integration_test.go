// 本文件验证兵力和征兵队列回填校验工具在隔离 test_ MySQL 库上的行为。
package main

import (
	"context"
	"testing"
	"time"

	"hero3/internal/app/game"
)

// TestBackfillArmyAndRecruitQueuesRepairsMissingRows 验证兵力和征兵队列回填能修复旧数据。
func TestBackfillArmyAndRecruitQueuesRepairsMissingRows(t *testing.T) {
	dsn, db := openIsolatedDBToolDatabase(t, "army")
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	now := time.Now().UTC()
	insertDBToolPlayerState(t, db, "dbtool_army_account", "dbtool_army_player", game.GameState{
		Player: game.Player{ID: "dbtool_army_player", Nickname: "兵力回填测试", Faction: "wei"},
		Resources: game.ResourceState{
			Items:    map[string]int{"wood": 100},
			Capacity: map[string]int{"wood": 1000},
		},
		Army: []game.ArmyUnit{
			{UnitType: "weiInfantry", Amount: 9},
		},
		RecruitQueues: []game.RecruitQueue{
			{ID: "rq_dbtool_army", UnitType: "qingZhouArmy", Amount: 3, EndsAt: now.Add(time.Hour).Format(time.RFC3339)},
		},
		ResourceSettledAt: now.Format(time.RFC3339),
		ServerTime:        now.Format(time.RFC3339),
	})

	armyVerifyBefore, err := verifyPlayerArmy(ctx, dsn)
	if err != nil {
		t.Fatalf("verify army before backfill: %v", err)
	}
	if armyVerifyBefore.Mismatches == 0 {
		t.Fatalf("expected verify-army to find missing player_army_units rows")
	}
	queueVerifyBefore, err := verifyPlayerRecruitQueues(ctx, dsn)
	if err != nil {
		t.Fatalf("verify recruit queues before backfill: %v", err)
	}
	if queueVerifyBefore.Mismatches == 0 {
		t.Fatalf("expected verify-recruit-queues to find missing player_recruit_queues rows")
	}

	if _, err := backfillPlayerArmy(ctx, dsn); err != nil {
		t.Fatalf("backfill army: %v", err)
	}
	if _, err := backfillPlayerRecruitQueues(ctx, dsn); err != nil {
		t.Fatalf("backfill recruit queues: %v", err)
	}
	armyVerifyAfter, err := verifyPlayerArmy(ctx, dsn)
	if err != nil {
		t.Fatalf("verify army after backfill: %v", err)
	}
	if armyVerifyAfter.Mismatches != 0 {
		t.Fatalf("expected backfill to repair army, got mismatches=%d", armyVerifyAfter.Mismatches)
	}
	queueVerifyAfter, err := verifyPlayerRecruitQueues(ctx, dsn)
	if err != nil {
		t.Fatalf("verify recruit queues after backfill: %v", err)
	}
	if queueVerifyAfter.Mismatches != 0 {
		t.Fatalf("expected backfill to repair recruit queues, got mismatches=%d", queueVerifyAfter.Mismatches)
	}
}
