// 本文件验证援军武将经验快照、升级结果和连续参战基线。
package game

import (
	"testing"
	"time"
)

// TestReserveReinforcementGeneralCopiesExperience 验证援军出发时冻结累计经验。
func TestReserveReinforcementGeneralCopiesExperience(t *testing.T) {
	now := time.Now().UTC()
	state := &GameState{Generals: []General{{ID: "sunquan", Name: "孙权", Level: 2, Exp: 777}}}

	snapshots, err := reserveReinforcementGenerals(state, []string{"sunquan"}, "rein_exp_snapshot", now)
	if err != nil {
		t.Fatalf("reserveReinforcementGenerals failed: %v", err)
	}
	if len(snapshots) != 1 || snapshots[0].Level != 2 || snapshots[0].Exp != 777 {
		t.Fatalf("expected level 2 and cumulative exp 777, got %+v", snapshots)
	}
}

// TestReinforcementGeneralProgressCarriesAcrossBattles 验证连续两场战斗使用上一场推进后的经验和等级。
func TestReinforcementGeneralProgressCarriesAcrossBattles(t *testing.T) {
	firstBaseline := generalExpRequiredForLevel(2) - 50
	firstGain := 110
	records := []Reinforcement{{
		ID: "rein_progress",
		Generals: []ReinforcementGeneralSnapshot{{
			ID: "xiahouyuan", Level: 1, Exp: firstBaseline,
		}},
	}}

	first := reinforcementGeneralProgress(records[0].Generals, firstGain)
	if first.LevelBefore != 1 || first.LevelAfter != 2 {
		t.Fatalf("expected first battle level 1 -> 2, got %+v", first)
	}
	advanceReinforcementGeneralSnapshots(records, map[string]int{records[0].ID: firstGain})
	if records[0].Generals[0].Level != 2 || records[0].Generals[0].Exp != firstBaseline+firstGain {
		t.Fatalf("expected persisted reinforcement baseline to advance, got %+v", records[0].Generals[0])
	}

	secondGain := generalExpRequiredForLevel(3) - records[0].Generals[0].Exp
	second := reinforcementGeneralProgress(records[0].Generals, secondGain)
	if second.LevelBefore != 2 || second.LevelAfter != 3 {
		t.Fatalf("expected second battle level 2 -> 3, got %+v", second)
	}
}

// TestReinforcementReportSnapshotUsesAuthoritativeLevelProgress 验证战报快照直接携带后端计算的等级变化。
func TestReinforcementReportSnapshotUsesAuthoritativeLevelProgress(t *testing.T) {
	baseline := generalExpRequiredForLevel(2) - 1
	record := Reinforcement{
		ID: "rein_report_progress", OwnerPlayerID: "player_helper", FromPlayerFaction: "wu",
		Generals: []ReinforcementGeneralSnapshot{{ID: "sunquan", Name: "孙权", Level: 1, Exp: baseline}},
	}

	snapshot := reinforcementReportSnapshot(record, map[string]int{"shadowGuard": 10}, 1)
	if snapshot.GeneralExpGained != 1 || snapshot.GeneralLevelBefore != 1 || snapshot.GeneralLevelAfter != 2 {
		t.Fatalf("expected authoritative reinforcement progress 1 exp and level 1 -> 2, got %+v", snapshot)
	}
}
