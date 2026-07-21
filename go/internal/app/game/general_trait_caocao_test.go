// 本文件验证曹操留城产兵、离城停产，以及魏武统御在主动进攻时严格不触发。
package game

import (
	"testing"
	"time"

	"hero3/internal/core/general"
)

// TestCaoCaoDualTraitsConnectGuardProductionBattleAndReturn 验证产兵、出征、离城停产、战斗和归城复产完整闭环。
func TestCaoCaoDualTraitsConnectGuardProductionBattleAndReturn(t *testing.T) {
	setTestFactionsAndGenerals(t, FactionsConfig{
		"wei": {Name: "魏国", Generals: []GeneralInfo{{ID: "caocao", Name: "曹操"}}},
		"shu": {Name: "蜀国", Generals: []GeneralInfo{{ID: "liubei", Name: "刘备"}}},
	}, GeneralsConfig{Enabled: true, Heroes: map[string]GeneralHeroConfig{
		"caocao": {
			ID: "caocao", Name: "曹操", Faction: "wei", Enabled: true,
			SpecialTrait: GeneralTraitConfig{
				TraitID: "weiwu_haoling", TraitType: general.TraitTypeSpecial, Enabled: true, Scope: "self_city", TargetUnitType: "huWei",
				Params: map[string]float64{"guardPerMinute": 300},
			},
			BonusTrait: GeneralTraitConfig{
				TraitID: "weiwu_tongyu", TraitType: general.TraitTypeBonus, Enabled: true, Scope: "self_army",
				AllowedSides: []string{"defender", "reinforcement"},
				Params:       map[string]float64{"defenseBonusRate": 0.15},
			},
		},
		"liubei": {ID: "liubei", Name: "刘备", Faction: "shu", Enabled: true},
	}})

	svc, repo, attacker, defender := newPvpTestServiceForGenerals(t, "wei", "caocao", "shu", "liubei")
	unitsMu.Lock()
	activeUnits["wei"]["huWei"] = UnitConfig{
		Name: "虎卫", Category: "infantry",
		Stats: map[string]int{"attack": 10, "infantryDefense": 10, "cavalryDefense": 8, "carryCapacity": 5, "upkeep": 1},
	}
	unitsMu.Unlock()
	attacker.Army = []ArmyUnit{{UnitType: "weiInfantry", Amount: 100}}
	attacker.ResourceSettledAt = time.Now().UTC().Add(-24 * time.Hour).Format(resourceDateLayout)
	defender.Army = []ArmyUnit{{UnitType: "shuInfantry", Amount: 100}}
	defender.Buildings = nil
	repo.players[attacker.Player.ID] = attacker
	repo.players[defender.Player.ID] = defender

	started, err := svc.StartPvpAttack(PvpAttackRequest{
		PlayerID: attacker.Player.ID, TargetPlayerID: defender.Player.ID, MarchMode: PvpMarchTypeAttack,
		Troops: map[string]int{"huWei": 100}, GeneralIDs: []string{"caocao"},
	})
	if err != nil {
		t.Fatalf("StartPvpAttack with newly produced guards failed: %v", err)
	}
	away, err := repo.GetState(attacker.Player.ID)
	if err != nil || armySliceToMap(away.Army)["huWei"] != 431900 {
		t.Fatalf("expected departure to produce 432000 and dispatch 100 guards, state=%+v err=%v", away, err)
	}
	away.ResourceSettledAt = time.Now().UTC().Add(-24 * time.Hour).Format(resourceDateLayout)
	repo.players[attacker.Player.ID] = away
	awayView, err := svc.GetMilitaryView(attacker.Player.ID)
	if err != nil || armySliceToMap(awayView.Army)["huWei"] != 431900 {
		t.Fatalf("expected no guard production while Cao Cao is away, view=%+v err=%v", awayView.Army, err)
	}
	assertNoBattleReportsForTraitProcess(t, repo, attacker.Player.ID, "Cao Cao guard production")

	forcePvpMarchDue(t, repo, started.March.ID)
	battle, err := svc.ResolvePvpMarch(started.March.ID)
	if err != nil {
		t.Fatalf("ResolvePvpMarch failed: %v", err)
	}
	if battle.Result["attackerPower"] != float64(1000) || battle.Result["defensePower"] != float64(1000) {
		t.Fatalf("expected active attack to keep base power 1000/1000, result=%+v", battle.Result)
	}
	attackerLosses := pvpTestLossesFromBattle(t, battle, "attacker")
	defenderLosses := pvpTestLossesFromBattle(t, battle, "defender")
	if attackerLosses["huWei"] != 100 || defenderLosses["shuInfantry"] != 100 {
		t.Fatalf("expected equal-power real losses 100/100, attacker=%+v defender=%+v", attackerLosses, defenderLosses)
	}
	attackerReports, _, err := repo.ListReports(attacker.Player.ID, 10, 0)
	if err != nil || len(attackerReports) == 0 || attackerReports[0].ID != battle.AttackerReportID {
		t.Fatalf("expected attacker report, reports=%+v err=%v", attackerReports, err)
	}
	defenderReports, _, err := repo.ListReports(defender.Player.ID, 10, 0)
	if err != nil || len(defenderReports) == 0 || defenderReports[0].ID != battle.DefenderReportID {
		t.Fatalf("expected defender report, reports=%+v err=%v", defenderReports, err)
	}
	for _, report := range []BattleReport{attackerReports[0], defenderReports[0]} {
		assertCaoCaoAttackReportHasNoCombatTrait(t, report)
	}

	storedMarch, err := repo.GetPvpMarch(started.March.ID)
	if err != nil || storedMarch.Status != PvpMarchStatusResolved || len(storedMarch.AttackTroops) != 0 {
		t.Fatalf("expected fully lost march resolved without a return leg, march=%+v err=%v", storedMarch, err)
	}
	returned, err := repo.GetState(attacker.Player.ID)
	if err != nil || !generalAvailableAtHome(returned.GeneralAssignments, "caocao") || armySliceToMap(returned.Army)["huWei"] != 431900 {
		t.Fatalf("expected Cao Cao released home without away-period production, state=%+v err=%v", returned, err)
	}
	returned.ResourceSettledAt = time.Now().UTC().Add(-24 * time.Hour).Format(resourceDateLayout)
	repo.players[attacker.Player.ID] = returned
	homeAgain, err := svc.GetMilitaryView(attacker.Player.ID)
	if err != nil || armySliceToMap(homeAgain.Army)["huWei"] != 863900 {
		t.Fatalf("expected post-return interval to produce another 432000 guards, view=%+v err=%v", homeAgain.Army, err)
	}
}

// assertCaoCaoAttackReportHasNoCombatTrait 核对主动进攻战报仅保留持有快照，不伪造任何曹操特性触发。
func assertCaoCaoAttackReportHasNoCombatTrait(t *testing.T, report BattleReport) {
	t.Helper()
	if len(report.TraitTriggered) != 0 || len(report.TraitOutcomes) != 0 {
		t.Fatalf("expected no Cao Cao trait in active-attack timeline, report=%+v", report)
	}
	if _, exists := report.TraitOutcomes["weiwu_haoling"]; exists {
		t.Fatalf("expected guard production absent from battle outcomes, outcomes=%+v", report.TraitOutcomes)
	}
	if report.Detail == nil || len(report.Detail.Traits) != 0 {
		t.Fatalf("expected standard attack timeline to omit Weiwu Tongyu, detail=%+v", report.Detail)
	}
	if len(report.PvpAttackerGenerals) != 1 || !pvpSnapshotHasTrait(report.PvpAttackerGenerals[0], "weiwu_haoling") || !pvpSnapshotHasTrait(report.PvpAttackerGenerals[0], "weiwu_tongyu") {
		t.Fatalf("expected Cao Cao snapshot to retain both owned traits, snapshots=%+v", report.PvpAttackerGenerals)
	}
	foundUnit := false
	for _, unit := range report.Detail.PrimarySide.Units {
		if unit.UnitType == "huWei" {
			foundUnit = unit.Dispatched == 100 && unit.Lost == 100 && unit.Survived == 0
		}
	}
	if !foundUnit {
		t.Fatalf("expected standard HuWei row 100/100/0, units=%+v", report.Detail.PrimarySide.Units)
	}
}

// TestPvpCaoCaoDefenseStrengthensWholeArmyAndReportsActualDeltas 验证曹操守城时全军防御加成进入真实战力和双方战报。
func TestPvpCaoCaoDefenseStrengthensWholeArmyAndReportsActualDeltas(t *testing.T) {
	setTestFactionsAndGenerals(t, FactionsConfig{
		"wei": {Name: "魏国", Generals: []GeneralInfo{{ID: "caocao", Name: "曹操"}}},
		"wu":  {Name: "吴国", Generals: []GeneralInfo{{ID: "opponent", Name: "对手"}}},
	}, GeneralsConfig{Enabled: true, Heroes: map[string]GeneralHeroConfig{
		"caocao": {
			ID: "caocao", Name: "曹操", Faction: "wei", Enabled: true,
			BonusTrait: GeneralTraitConfig{
				TraitID: "weiwu_tongyu", TraitType: general.TraitTypeBonus, Enabled: true, Scope: "self_army",
				AllowedSides: []string{"defender", "reinforcement"}, Params: map[string]float64{"defenseBonusRate": 0.15},
			},
		},
		"opponent": {ID: "opponent", Name: "对手", Faction: "wu", Enabled: true},
	}})
	svc, repo, attacker, defender := newPvpTestServiceForGenerals(t, "wu", "opponent", "wei", "caocao")
	unitsMu.Lock()
	activeUnits["wei"]["huWei"] = UnitConfig{Name: "虎卫", Category: "infantry", Stats: map[string]int{"attack": 14, "infantryDefense": 8, "cavalryDefense": 5, "carryCapacity": 5, "upkeep": 1}}
	activeUnits["wei"]["qingZhouArmy"] = UnitConfig{Name: "青州兵", Category: "infantry", Stats: map[string]int{"attack": 8, "infantryDefense": 7, "cavalryDefense": 10, "carryCapacity": 5, "upkeep": 1}}
	unitsMu.Unlock()
	attacker.Army = []ArmyUnit{{UnitType: "wuInfantry", Amount: 200}}
	defender.Army = []ArmyUnit{{UnitType: "huWei", Amount: 100}, {UnitType: "qingZhouArmy", Amount: 100}}
	defender.Buildings = nil
	repo.players[attacker.Player.ID] = attacker
	repo.players[defender.Player.ID] = defender

	started, err := svc.StartPvpAttack(PvpAttackRequest{
		PlayerID: attacker.Player.ID, TargetPlayerID: defender.Player.ID, MarchMode: PvpMarchTypeAttack,
		Troops: map[string]int{"wuInfantry": 200}, GeneralIDs: []string{"opponent"},
	})
	if err != nil {
		t.Fatalf("StartPvpAttack failed: %v", err)
	}
	forcePvpMarchDue(t, repo, started.March.ID)
	battle, err := svc.ResolvePvpMarch(started.March.ID)
	if err != nil {
		t.Fatalf("ResolvePvpMarch failed: %v", err)
	}
	if got := battle.Result["defensePower"]; got != float64(1700) {
		t.Fatalf("expected whole-army defense power 1500 -> 1700, got %+v", battle.Result)
	}
	attackerReports, _, _ := repo.ListReports(attacker.Player.ID, 10, 0)
	defenderReports, _, _ := repo.ListReports(defender.Player.ID, 10, 0)
	if len(attackerReports) == 0 || len(defenderReports) == 0 {
		t.Fatalf("expected both battle reports, attacker=%+v defender=%+v", attackerReports, defenderReports)
	}
	for _, report := range []BattleReport{attackerReports[0], defenderReports[0]} {
		outcome, ok := report.TraitOutcomes["weiwu_tongyu"]
		infantry, infantryOK := outcome.Detail["infantryDefenseModifiedUnits"].(map[string]int)
		cavalry, cavalryOK := outcome.Detail["cavalryDefenseModifiedUnits"].(map[string]int)
		if !ok || !infantryOK || !cavalryOK || outcome.OwnerSide != "defender" || outcome.OwnerGeneralID != "caocao" || infantry["huWei"] != 1 || infantry["qingZhouArmy"] != 1 || cavalry["huWei"] != 1 || cavalry["qingZhouArmy"] != 2 {
			t.Fatalf("expected actual all-army defense deltas in report, got %+v", outcome)
		}
		if report.Detail == nil || !standardReportHasTrait(report.Detail, "weiwu_tongyu") {
			t.Fatalf("expected Weiwu Tongyu in standard report timeline, detail=%+v", report.Detail)
		}
	}
}
