// 本文件验证马超西凉突击由防守主将触发时只追加敌方骑兵损失，并同步真实状态和双方战报。
package game

import (
	"testing"

	"hero3/internal/core/general"
)

type xiliangDefenseRunResult struct {
	attackerLosses map[string]int
	reports        []BattleReport
	march          PvpMarch
}

// runXiliangDefensePvp 执行一场马超守城的真实 PVP，并返回攻方最终战损、双方战报和返程队列。
func runXiliangDefensePvp(t *testing.T, enabled bool) xiliangDefenseRunResult {
	t.Helper()
	machao := GeneralHeroConfig{ID: "machao", Name: "马超", Faction: "shu", Enabled: true}
	if enabled {
		machao.SpecialTrait = GeneralTraitConfig{
			TraitID: "xiliang_tuji", TraitType: general.TraitTypeSpecial, Enabled: true, Scope: "enemy_army", TargetUnitType: "cavalry",
			Params: map[string]float64{"effectRate": 0.2, "triggerChance": 1},
		}
	}
	setTestFactionsAndGenerals(t, FactionsConfig{
		"wu":  {Name: "吴国", Generals: []GeneralInfo{{ID: "sunquan", Name: "孙权"}}},
		"shu": {Name: "蜀国", Generals: []GeneralInfo{{ID: "machao", Name: "马超"}}},
	}, GeneralsConfig{Enabled: true, Heroes: map[string]GeneralHeroConfig{
		"sunquan": {ID: "sunquan", Name: "孙权", Faction: "wu", Enabled: true},
		"machao":  machao,
	}})
	svc, repo, attacker, defender := newPvpTestServiceForGenerals(t, "wu", "sunquan", "shu", "machao")
	unitsMu.Lock()
	activeUnits["wu"]["wuCavalry"] = UnitConfig{
		Name: "吴测试骑兵", Category: "cavalry",
		Stats: map[string]int{"attack": 10, "infantryDefense": 10, "cavalryDefense": 10, "carryCapacity": 5, "upkeep": 1},
	}
	unitsMu.Unlock()
	attacker.Army = []ArmyUnit{{UnitType: "wuInfantry", Amount: 1000}, {UnitType: "wuCavalry", Amount: 1000}}
	defender.Army = []ArmyUnit{{UnitType: "shuInfantry", Amount: 100}}
	defender.Buildings = nil
	repo.players[attacker.Player.ID] = attacker
	repo.players[defender.Player.ID] = defender

	started, err := svc.StartPvpAttack(PvpAttackRequest{
		PlayerID: attacker.Player.ID, TargetPlayerID: defender.Player.ID, MarchMode: PvpMarchTypeAttack,
		Troops: map[string]int{"wuInfantry": 1000, "wuCavalry": 1000}, GeneralIDs: []string{"sunquan"},
	})
	if err != nil {
		t.Fatalf("StartPvpAttack failed: %v", err)
	}
	forcePvpMarchDue(t, repo, started.March.ID)
	battle, err := svc.ResolvePvpMarch(started.March.ID)
	if err != nil {
		t.Fatalf("ResolvePvpMarch failed: %v", err)
	}
	attackerLosses := pvpTestLossesFromBattle(t, battle, "attacker")
	attackerReports, _, err := repo.ListReports(attacker.Player.ID, 10, 0)
	if err != nil || len(attackerReports) == 0 || attackerReports[0].ID != battle.AttackerReportID {
		t.Fatalf("expected attacker report, reports=%+v err=%v", attackerReports, err)
	}
	defenderReports, _, err := repo.ListReports(defender.Player.ID, 10, 0)
	if err != nil || len(defenderReports) == 0 || defenderReports[0].ID != battle.DefenderReportID {
		t.Fatalf("expected defender report, reports=%+v err=%v", defenderReports, err)
	}
	if attackerReports[0].LostUnits["wuInfantry"] != attackerLosses["wuInfantry"] || attackerReports[0].LostUnits["wuCavalry"] != attackerLosses["wuCavalry"] ||
		defenderReports[0].DefenderLostUnits["wuInfantry"] != attackerLosses["wuInfantry"] || defenderReports[0].DefenderLostUnits["wuCavalry"] != attackerLosses["wuCavalry"] {
		t.Fatalf("expected both legacy reports to match attacker losses, battle=%+v reports=%+v/%+v", attackerLosses, attackerReports[0].LostUnits, defenderReports[0].DefenderLostUnits)
	}
	storedMarch, err := repo.GetPvpMarch(started.March.ID)
	if err != nil || storedMarch.AttackTroops["wuInfantry"] != 1000-attackerLosses["wuInfantry"] || storedMarch.AttackTroops["wuCavalry"] != 1000-attackerLosses["wuCavalry"] {
		t.Fatalf("expected return march to match attacker losses, march=%+v losses=%+v err=%v", storedMarch.AttackTroops, attackerLosses, err)
	}
	for _, report := range []BattleReport{attackerReports[0], defenderReports[0]} {
		assertXiliangAttackerStandardUnits(t, report, attackerLosses)
	}
	return xiliangDefenseRunResult{attackerLosses: attackerLosses, reports: []BattleReport{attackerReports[0], defenderReports[0]}, march: storedMarch}
}

// assertXiliangAttackerStandardUnits 核对双方标准详情中的攻方两类兵种最终阵亡和存活。
func assertXiliangAttackerStandardUnits(t *testing.T, report BattleReport, losses map[string]int) {
	t.Helper()
	if report.Detail == nil || report.Detail.SecondarySide == nil {
		t.Fatalf("expected complete standard report, got %+v", report.Detail)
	}
	for _, side := range []BattleReportSide{report.Detail.PrimarySide, *report.Detail.SecondarySide} {
		if side.Role != "attacker" {
			continue
		}
		matched := 0
		for _, unit := range side.Units {
			if unit.UnitType != "wuInfantry" && unit.UnitType != "wuCavalry" {
				continue
			}
			if unit.AmountBefore != 1000 || unit.Lost != losses[unit.UnitType] || unit.Survived != 1000-losses[unit.UnitType] {
				t.Fatalf("expected standard attacker %s to reconcile, unit=%+v losses=%+v", unit.UnitType, unit, losses)
			}
			matched++
		}
		if matched != 2 {
			t.Fatalf("expected both attacker unit rows, side=%+v", side)
		}
		return
	}
	t.Fatalf("expected attacker side in standard report, detail=%+v", report.Detail)
}

// TestPvpXiliangTujiDefenderOnlyAddsEnemyCavalryLosses 验证守城马超只增加攻方骑兵损失，并把实际增量写入双方战报。
func TestPvpXiliangTujiDefenderOnlyAddsEnemyCavalryLosses(t *testing.T) {
	control := runXiliangDefensePvp(t, false)
	active := runXiliangDefensePvp(t, true)
	wantExtra := 200
	if remaining := 1000 - control.attackerLosses["wuCavalry"]; wantExtra > remaining {
		wantExtra = remaining
	}
	if wantExtra <= 0 || active.attackerLosses["wuCavalry"] != control.attackerLosses["wuCavalry"]+wantExtra {
		t.Fatalf("expected cavalry losses %d + %d, got %d", control.attackerLosses["wuCavalry"], wantExtra, active.attackerLosses["wuCavalry"])
	}
	if active.attackerLosses["wuInfantry"] != control.attackerLosses["wuInfantry"] {
		t.Fatalf("expected infantry losses unchanged at %d, got %d", control.attackerLosses["wuInfantry"], active.attackerLosses["wuInfantry"])
	}
	for _, report := range active.reports {
		outcome, ok := report.TraitOutcomes["xiliang_tuji"]
		extra, detailOK := outcome.Detail["targetExtraLosses"].(map[string]int)
		if !ok || !detailOK || len(extra) != 1 || extra["wuCavalry"] != wantExtra || outcome.OwnerSide != "defender" || outcome.OwnerGeneralID != "machao" {
			t.Fatalf("expected defender Xiliang real cavalry extra %d, got %+v", wantExtra, outcome)
		}
		if outcome.Detail["effectRate"] != 0.2 {
			t.Fatalf("expected defender Xiliang design rate 0.2, got %+v", outcome.Detail)
		}
		standardMatched := false
		for _, trait := range report.Detail.Traits {
			if trait.TraitID == "xiliang_tuji" && trait.GeneralID == "machao" && trait.OwnerSide == "secondary" && trait.OwnerRole == "defender" {
				if trait.Detail["effectRate"] != 0.2 {
					t.Fatalf("expected standard defender Xiliang design rate 0.2, got %+v", trait.Detail)
				}
				standardMatched = true
			}
		}
		if !standardMatched {
			t.Fatalf("expected defender Xiliang in standard report, traits=%+v", report.Detail.Traits)
		}
	}
}
