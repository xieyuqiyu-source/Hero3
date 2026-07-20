// 本文件验证甄宓双特性在真实 PVP 中依次俘虏、破防，并与兵力、经验和双方战报完整对账。
package game

import (
	"testing"

	"hero3/internal/core/general"
)

// TestZhenMiDualTraitsReconcilePvpCaptureDefenseAndRetry 验证俘虏、破防、核心战损及重复结算保持同一权威结果。
func TestZhenMiDualTraitsReconcilePvpCaptureDefenseAndRetry(t *testing.T) {
	setTestFactionsAndGenerals(t, FactionsConfig{
		"wei": {Name: "魏国", Generals: []GeneralInfo{{ID: "zhenmi", Name: "甄宓"}}},
		"shu": {Name: "蜀国", Generals: []GeneralInfo{{ID: "liubei", Name: "刘备"}}},
	}, GeneralsConfig{Enabled: true, Heroes: map[string]GeneralHeroConfig{
		"zhenmi": {
			ID: "zhenmi", Name: "甄宓", Faction: "wei", Enabled: true,
			SpecialTrait: GeneralTraitConfig{
				TraitID: "meiren", TraitType: general.TraitTypeSpecial, Enabled: true,
				Scope: "enemy_army", AllowedSides: []string{"attacker"},
				Params: map[string]float64{"captureRate": 0.2, "captureMax": 10000, "triggerChance": 1},
			},
			BonusTrait: GeneralTraitConfig{
				TraitID: "meihuo_raozhen", TraitType: general.TraitTypeBonus, Enabled: true,
				Scope: "enemy_army", AllowedSides: []string{"attacker"},
				Params: map[string]float64{"enemyDefenseReductionRate": 0.1},
			},
		},
		"liubei": {ID: "liubei", Name: "刘备", Faction: "shu", Enabled: true},
	}})

	svc, repo, attacker, defender := newPvpTestServiceForGenerals(t, "wei", "zhenmi", "shu", "liubei")
	attacker.Army = []ArmyUnit{{UnitType: "weiInfantry", Amount: 1000}}
	defender.Army = []ArmyUnit{{UnitType: "shuInfantry", Amount: 1000}}
	defender.Buildings = nil
	repo.players[attacker.Player.ID] = attacker
	repo.players[defender.Player.ID] = defender

	started, err := svc.StartPvpAttack(PvpAttackRequest{
		PlayerID: attacker.Player.ID, TargetPlayerID: defender.Player.ID, MarchMode: PvpMarchTypeAttack,
		Troops: map[string]int{"weiInfantry": 1000}, GeneralIDs: []string{"zhenmi"},
	})
	if err != nil {
		t.Fatalf("StartPvpAttack failed: %v", err)
	}
	forcePvpMarchDue(t, repo, started.March.ID)
	battle, err := svc.ResolvePvpMarch(started.March.ID)
	if err != nil {
		t.Fatalf("ResolvePvpMarch failed: %v", err)
	}
	if battle.Result["attackerPower"] != float64(10000) || battle.Result["defensePower"] != float64(7200) {
		t.Fatalf("expected 10000 attack and 800 remaining defenders at 9 defense = 7200, result=%+v", battle.Result)
	}
	attackerLosses := pvpTestLossesFromBattle(t, battle, "attacker")
	defenderLosses := pvpTestLossesFromBattle(t, battle, "defender")
	if attackerLosses["weiInfantry"] != 626 || defenderLosses["shuInfantry"] != 1000 {
		t.Fatalf("expected battle removals 626/1000 including 200 captures, attacker=%+v defender=%+v", attackerLosses, defenderLosses)
	}

	attackerReports, _, err := repo.ListReports(attacker.Player.ID, 10, 0)
	if err != nil || len(attackerReports) != 1 || attackerReports[0].ID != battle.AttackerReportID {
		t.Fatalf("expected one attacker report, reports=%+v err=%v", attackerReports, err)
	}
	defenderReports, _, err := repo.ListReports(defender.Player.ID, 10, 0)
	if err != nil || len(defenderReports) != 1 || defenderReports[0].ID != battle.DefenderReportID {
		t.Fatalf("expected one defender report, reports=%+v err=%v", defenderReports, err)
	}
	for _, report := range []BattleReport{attackerReports[0], defenderReports[0]} {
		assertZhenMiDualTraitPvpReport(t, report)
	}

	storedAttacker, err := repo.GetState(attacker.Player.ID)
	if err != nil || pvpTestGeneralExp(storedAttacker, "zhenmi") != 800 || attackerReports[0].GeneralExpGained != 800 {
		t.Fatalf("expected Zhen Mi exp to count 800 real deaths but not 200 captures, state=%+v report=%+v err=%v", storedAttacker.Generals, attackerReports[0], err)
	}
	storedDefender, err := repo.GetState(defender.Player.ID)
	if err != nil || armySliceToMap(storedDefender.Army)["shuInfantry"] != 0 {
		t.Fatalf("expected 200 captured plus 800 dead to empty defender army, state=%+v err=%v", storedDefender.Army, err)
	}
	garrison, err := repo.GetReinforcement(ObtainedGarrisonID(attacker.Player.ID))
	if err != nil || garrison.RemainingTroops["shuInfantry"] != 200 {
		t.Fatalf("expected 200 captured troops in obtained garrison, garrison=%+v err=%v", garrison, err)
	}
	returning, err := repo.GetPvpMarch(started.March.ID)
	if err != nil || returning.AttackTroops["weiInfantry"] != 374 {
		t.Fatalf("expected 374 surviving attackers in return march, march=%+v err=%v", returning, err)
	}

	retried, err := svc.ResolvePvpMarch(started.March.ID)
	if err != nil || retried.ID != battle.ID {
		t.Fatalf("expected repeated resolve to return original battle, battle=%+v err=%v", retried, err)
	}
	garrison, err = repo.GetReinforcement(ObtainedGarrisonID(attacker.Player.ID))
	if err != nil || garrison.RemainingTroops["shuInfantry"] != 200 {
		t.Fatalf("expected repeated resolve not to duplicate captures, garrison=%+v err=%v", garrison, err)
	}
	storedAttacker, err = repo.GetState(attacker.Player.ID)
	if err != nil || pvpTestGeneralExp(storedAttacker, "zhenmi") != 800 {
		t.Fatalf("expected repeated resolve not to duplicate exp, state=%+v err=%v", storedAttacker.Generals, err)
	}
	attackerReports, attackerTotal, err := repo.ListReports(attacker.Player.ID, 10, 0)
	if err != nil || attackerTotal != 1 || len(attackerReports) != 1 {
		t.Fatalf("expected repeated resolve not to duplicate reports, total=%d reports=%+v err=%v", attackerTotal, attackerReports, err)
	}
}

// assertZhenMiDualTraitPvpReport 核对双方战报中的俘虏去向、实际破防、时间线及标准兵力口径。
func assertZhenMiDualTraitPvpReport(t *testing.T, report BattleReport) {
	t.Helper()
	wantTimeline := []string{"meiren", "meihuo_raozhen"}
	if len(report.TraitTriggered) != len(wantTimeline) || len(report.TraitOutcomes) != len(wantTimeline) {
		t.Fatalf("expected two Zhen Mi outcomes, report=%+v", report)
	}
	for index, traitID := range wantTimeline {
		if report.TraitTriggered[index] != traitID || report.Detail == nil || len(report.Detail.Traits) != len(wantTimeline) || report.Detail.Traits[index].TraitID != traitID {
			t.Fatalf("expected stable Zhen Mi timeline %v, legacy=%+v standard=%+v", wantTimeline, report.TraitTriggered, report.Detail)
		}
	}
	captured, capturedOK := report.TraitOutcomes["meiren"].Detail["capturedToGarrison"].(map[string]int)
	infantryDefense, infantryOK := report.TraitOutcomes["meihuo_raozhen"].Detail["infantryDefenseModifiedUnits"].(map[string]int)
	cavalryDefense, cavalryOK := report.TraitOutcomes["meihuo_raozhen"].Detail["cavalryDefenseModifiedUnits"].(map[string]int)
	if !capturedOK || captured["shuInfantry"] != 200 || report.CapturedToGarrison["shuInfantry"] != 200 || !infantryOK || !cavalryOK || infantryDefense["shuInfantry"] != -1 || cavalryDefense["shuInfantry"] != -1 {
		t.Fatalf("expected actual capture 200 and defense changes -1/-1, report=%+v", report)
	}
	if len(report.PvpAttackerGenerals) != 1 || !pvpSnapshotHasTrait(report.PvpAttackerGenerals[0], "meiren") || !pvpSnapshotHasTrait(report.PvpAttackerGenerals[0], "meihuo_raozhen") {
		t.Fatalf("expected Zhen Mi snapshot to retain both owned traits, snapshots=%+v", report.PvpAttackerGenerals)
	}
	if report.ViewType == ReportViewAttack {
		if report.LostUnits["weiInfantry"] != 626 || report.DefenderLostUnits["shuInfantry"] != 800 || report.SurvivedUnits["weiInfantry"] != 374 {
			t.Fatalf("expected attack-view deaths and survivors 626/374 versus 800/0, report=%+v", report)
		}
	} else if report.LostUnits["shuInfantry"] != 800 || report.DefenderLostUnits["weiInfantry"] != 626 || report.SurvivedUnits["shuInfantry"] != 0 {
		t.Fatalf("expected defense-view deaths and survivors 800/0 versus 626/374, report=%+v", report)
	}
	assertStandardUnitRow(t, report.ID, report.Detail.PrimarySide, "weiInfantry", 1000, 626, 374)
	if report.Detail.SecondarySide == nil {
		t.Fatalf("expected defender standard side, detail=%+v", report.Detail)
	}
	assertStandardUnitRow(t, report.ID, *report.Detail.SecondarySide, "shuInfantry", 1000, 800, 0)
}
