// 本文件验证曹操留城产出的虎卫进入真实战斗，并由魏武统御强化且准确写入双方战报。
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
				Params: map[string]float64{"guardPerMinute": 500, "maxGuardPerSettle": 3000},
			},
			BonusTrait: GeneralTraitConfig{
				TraitID: "weiwu_tongyu", TraitType: general.TraitTypeBonus, Enabled: true, Scope: "self_army", TargetUnitType: "huWei",
				AllowedSides: []string{"attacker", "defender", "reinforcement"},
				Params:       map[string]float64{"attackBonusRate": 0.1, "defenseBonusRate": 0.1, "triggerChance": 1},
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
	if err != nil || armySliceToMap(away.Army)["huWei"] != 2900 {
		t.Fatalf("expected departure to produce 3000 and dispatch 100 guards, state=%+v err=%v", away, err)
	}
	away.ResourceSettledAt = time.Now().UTC().Add(-24 * time.Hour).Format(resourceDateLayout)
	repo.players[attacker.Player.ID] = away
	awayView, err := svc.GetMilitaryView(attacker.Player.ID)
	if err != nil || armySliceToMap(awayView.Army)["huWei"] != 2900 {
		t.Fatalf("expected no guard production while Cao Cao is away, view=%+v err=%v", awayView.Army, err)
	}
	assertNoBattleReportsForTraitProcess(t, repo, attacker.Player.ID, "Cao Cao guard production")

	forcePvpMarchDue(t, repo, started.March.ID)
	battle, err := svc.ResolvePvpMarch(started.March.ID)
	if err != nil {
		t.Fatalf("ResolvePvpMarch failed: %v", err)
	}
	if battle.Result["attackerPower"] != float64(1100) || battle.Result["defensePower"] != float64(1000) {
		t.Fatalf("expected Weiwu power 1100/1000, result=%+v", battle.Result)
	}
	attackerLosses := pvpTestLossesFromBattle(t, battle, "attacker")
	defenderLosses := pvpTestLossesFromBattle(t, battle, "defender")
	if attackerLosses["huWei"] != 87 || defenderLosses["shuInfantry"] != 100 {
		t.Fatalf("expected real losses 87/100, attacker=%+v defender=%+v", attackerLosses, defenderLosses)
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
		assertCaoCaoDualTraitBattleReport(t, report)
	}

	storedMarch, err := repo.GetPvpMarch(started.March.ID)
	if err != nil || storedMarch.Status != PvpMarchStatusReturning || storedMarch.AttackTroops["huWei"] != 13 {
		t.Fatalf("expected 13 surviving guards in return march, march=%+v err=%v", storedMarch, err)
	}
	forcePvpReturnDue(t, repo, started.March.ID)
	if _, err := svc.CompletePvpRecall(started.March.ID); err != nil {
		t.Fatalf("CompletePvpRecall failed: %v", err)
	}
	returned, err := repo.GetState(attacker.Player.ID)
	if err != nil || !generalAvailableAtHome(returned.GeneralAssignments, "caocao") || armySliceToMap(returned.Army)["huWei"] != 2913 {
		t.Fatalf("expected 13 guards returned and Cao Cao released home, state=%+v err=%v", returned, err)
	}
	returned.ResourceSettledAt = time.Now().UTC().Add(-24 * time.Hour).Format(resourceDateLayout)
	repo.players[attacker.Player.ID] = returned
	homeAgain, err := svc.GetMilitaryView(attacker.Player.ID)
	if err != nil || armySliceToMap(homeAgain.Army)["huWei"] != 5913 {
		t.Fatalf("expected post-return interval to produce another 3000 guards, view=%+v err=%v", homeAgain.Army, err)
	}
}

// assertCaoCaoDualTraitBattleReport 核对产兵能力仅在快照中，战斗时间线只记录魏武统御及真实修正。
func assertCaoCaoDualTraitBattleReport(t *testing.T, report BattleReport) {
	t.Helper()
	if len(report.TraitTriggered) != 1 || report.TraitTriggered[0] != "weiwu_tongyu" || len(report.TraitOutcomes) != 1 {
		t.Fatalf("expected only Weiwu Tongyu in legacy timeline, report=%+v", report)
	}
	if _, exists := report.TraitOutcomes["weiwu_haoling"]; exists {
		t.Fatalf("expected guard production absent from battle outcomes, outcomes=%+v", report.TraitOutcomes)
	}
	outcome := report.TraitOutcomes["weiwu_tongyu"]
	attack, attackOK := outcome.Detail["attackModifiedUnits"].(map[string]int)
	infantry, infantryOK := outcome.Detail["infantryDefenseModifiedUnits"].(map[string]int)
	cavalry, cavalryOK := outcome.Detail["cavalryDefenseModifiedUnits"].(map[string]int)
	if !attackOK || !infantryOK || !cavalryOK || outcome.OwnerGeneralID != "caocao" || outcome.OwnerSide != "attacker" || attack["huWei"] != 1 || infantry["huWei"] != 1 || cavalry["huWei"] != 1 {
		t.Fatalf("expected exact Weiwu HuWei deltas +1/+1/+1, outcome=%+v", outcome)
	}
	if report.Detail == nil || len(report.Detail.Traits) != 1 || report.Detail.Traits[0].TraitID != "weiwu_tongyu" {
		t.Fatalf("expected only Weiwu Tongyu in standard timeline, detail=%+v", report.Detail)
	}
	if len(report.PvpAttackerGenerals) != 1 || !pvpSnapshotHasTrait(report.PvpAttackerGenerals[0], "weiwu_haoling") || !pvpSnapshotHasTrait(report.PvpAttackerGenerals[0], "weiwu_tongyu") {
		t.Fatalf("expected Cao Cao snapshot to retain both owned traits, snapshots=%+v", report.PvpAttackerGenerals)
	}
	foundUnit := false
	for _, unit := range report.Detail.PrimarySide.Units {
		if unit.UnitType == "huWei" {
			foundUnit = unit.Dispatched == 100 && unit.Lost == 87 && unit.Survived == 13
		}
	}
	if !foundUnit {
		t.Fatalf("expected standard HuWei row 100/87/13, units=%+v", report.Detail.PrimarySide.Units)
	}
}
