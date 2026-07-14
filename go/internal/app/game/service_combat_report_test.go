// 本文件验证 NPC 战斗的情报阈值与战前兵力、实际阵亡快照彼此独立。
package game

import (
	"testing"
	"time"

	"hero3/internal/core/combat"
)

// npcReportFixture 构造只用于战报快照验证的 NPC 战斗状态。
func npcReportFixture() (GameState, NpcCity) {
	return GameState{
			Player:    Player{ID: "player_npc_report", Nickname: "测试玩家", Faction: "shu"},
			Resources: ResourceState{Items: map[string]int{}, Capacity: map[string]int{}},
		}, NpcCity{
			ID:        "npc_report",
			Name:      "落凤坡",
			Faction:   "shu",
			Resources: map[string]int{},
			Army:      []ArmyUnit{{UnitType: "greedyWolf", Amount: 100}},
		}
}

// TestNpcReportKeepsBeforeAndLostCounts 验证达到 25% 后分别保存战前数量和实际阵亡。
func TestNpcReportKeepsBeforeAndLostCounts(t *testing.T) {
	state, npc := npcReportFixture()
	result := combat.CombatResult{
		Winner:         "defender",
		DefenderLosses: []combat.UnitLoss{{ID: "greedyWolf", Count: 100, Losses: 25}},
	}
	report := applyNpcBattleResult(&state, &npc, result, nil, "plunder", time.Date(2026, 7, 13, 12, 0, 0, 0, time.UTC))
	if !report.DefenderRevealed {
		t.Fatal("expected defender information to be revealed at the 25% threshold")
	}
	if report.DefenderUnits["greedyWolf"] != 100 || report.DefenderLostUnits["greedyWolf"] != 25 {
		t.Fatalf("expected independent before/lost snapshots 100/25, got before=%v lost=%v", report.DefenderUnits, report.DefenderLostUnits)
	}
	detail := BuildBattleReportDetail(report)
	unit := detail.SecondarySide.Units[0]
	if unit.AmountBefore != 100 || unit.Lost != 25 || unit.Survived != 75 {
		t.Fatalf("expected standardized NPC unit snapshot 100/25/75, got %+v", unit)
	}
}

// TestNpcReportHidesDefenderBelowThreshold 验证不足 25% 时不向玩家战报泄露守军数量。
func TestNpcReportHidesDefenderBelowThreshold(t *testing.T) {
	state, npc := npcReportFixture()
	result := combat.CombatResult{
		Winner:         "defender",
		DefenderLosses: []combat.UnitLoss{{ID: "greedyWolf", Count: 100, Losses: 24}},
	}
	report := applyNpcBattleResult(&state, &npc, result, nil, "plunder", time.Date(2026, 7, 13, 12, 0, 0, 0, time.UTC))
	if report.DefenderRevealed || len(report.DefenderUnits) != 0 || len(report.DefenderLostUnits) != 0 {
		t.Fatalf("expected defender information hidden below threshold, got %+v", report)
	}
}
