// 本文件验证攻守双方同时触发同一特性 ID 时，真实战损、经验和双方战报不会互相覆盖。
package game

import (
	"testing"

	"hero3/internal/core/general"
)

// TestPvpSameTraitFromBothSidesKeepsIndependentRealOutcomes 验证双方黄忠的老当益壮分别追加敌方损失并保留两条结果。
func TestPvpSameTraitFromBothSidesKeepsIndependentRealOutcomes(t *testing.T) {
	trait := GeneralTraitConfig{
		TraitID: "laodang_yizhuang", TraitType: general.TraitTypeBonus, Enabled: true, Scope: "enemy_army",
		Params: map[string]float64{"effectRate": 0.1, "triggerChance": 1},
	}
	setTestFactionsAndGenerals(t, FactionsConfig{
		"shu": {Name: "蜀国", Generals: []GeneralInfo{{ID: "huangzhong", Name: "黄忠"}}},
	}, GeneralsConfig{Enabled: true, Heroes: map[string]GeneralHeroConfig{
		"huangzhong": {
			ID: "huangzhong", Name: "黄忠", Faction: "shu", Enabled: true,
			BonusTrait: trait,
		},
	}})
	svc, repo, attacker, defender := newPvpTestServiceForGenerals(t, "shu", "huangzhong", "shu", "huangzhong")
	attacker.Army = []ArmyUnit{{UnitType: "shuInfantry", Amount: 1000}}
	defender.Army = []ArmyUnit{{UnitType: "shuInfantry", Amount: 1000}}
	defender.Buildings = nil
	repo.players[attacker.Player.ID] = attacker
	repo.players[defender.Player.ID] = defender

	started, err := svc.StartPvpAttack(PvpAttackRequest{
		PlayerID: attacker.Player.ID, TargetPlayerID: defender.Player.ID, MarchMode: PvpMarchTypePlunder,
		Troops: map[string]int{"shuInfantry": 1000}, GeneralIDs: []string{"huangzhong"},
	})
	if err != nil {
		t.Fatalf("StartPvpAttack failed: %v", err)
	}
	forcePvpMarchDue(t, repo, started.March.ID)
	battle, err := svc.ResolvePvpMarch(started.March.ID)
	if err != nil {
		t.Fatalf("ResolvePvpMarch failed: %v", err)
	}
	if battle.Result["attackerPower"] != float64(10000) || battle.Result["defensePower"] != float64(10000) {
		t.Fatalf("expected equal base powers 10000/10000, got %v/%v", battle.Result["attackerPower"], battle.Result["defensePower"])
	}
	attackerLosses := pvpTestLossesFromBattle(t, battle, "attacker")
	defenderLosses := pvpTestLossesFromBattle(t, battle, "defender")

	attackerReports, _, err := repo.ListReports(attacker.Player.ID, 10, 0)
	if err != nil || len(attackerReports) == 0 || attackerReports[0].ID != battle.AttackerReportID {
		t.Fatalf("expected attacker report, reports=%+v err=%v", attackerReports, err)
	}
	defenderReports, _, err := repo.ListReports(defender.Player.ID, 10, 0)
	if err != nil || len(defenderReports) == 0 || defenderReports[0].ID != battle.DefenderReportID {
		t.Fatalf("expected defender report, reports=%+v err=%v", defenderReports, err)
	}
	if attackerLosses["shuInfantry"] != 600 || defenderLosses["shuInfantry"] != 600 {
		t.Fatalf("expected core loss 500 plus each opposing trait extra 100, got attacker=%+v defender=%+v attackerOutcomes=%+v defenderOutcomes=%+v", attackerLosses, defenderLosses, attackerReports[0].TraitOutcomes, defenderReports[0].TraitOutcomes)
	}
	for _, report := range []BattleReport{attackerReports[0], defenderReports[0]} {
		assertSameTraitBothSidesReport(t, report)
	}

	storedMarch, err := repo.GetPvpMarch(started.March.ID)
	if err != nil || storedMarch.Status != PvpMarchStatusReturning || storedMarch.AttackTroops["shuInfantry"] != 400 {
		t.Fatalf("expected 400 surviving attackers in return march, march=%+v err=%v", storedMarch, err)
	}
	storedAttacker, err := repo.GetState(attacker.Player.ID)
	if err != nil || pvpTestGeneralExp(storedAttacker, "huangzhong") != 600 {
		t.Fatalf("expected attacker Huang Zhong exp 600, state=%+v err=%v", storedAttacker.Generals, err)
	}
	storedDefender, err := repo.GetState(defender.Player.ID)
	if err != nil || armySliceToMap(storedDefender.Army)["shuInfantry"] != 400 || pvpTestGeneralExp(storedDefender, "huangzhong") != 600 {
		t.Fatalf("expected defender army 400 and Huang Zhong exp 600, state=%+v err=%v", storedDefender, err)
	}
}

// TestPvpSamePreBattleTraitFromBothSidesKeepsPowerAndReportOwnership 验证双方携带同 ID 战前特性时分别强化己方虎卫并保留独立结果。
func TestPvpSamePreBattleTraitFromBothSidesKeepsPowerAndReportOwnership(t *testing.T) {
	trait := GeneralTraitConfig{
		TraitID: "weiwu_tongyu", TraitType: general.TraitTypeBonus, Enabled: true, Scope: "self_army", TargetUnitType: "huWei",
		AllowedSides: []string{"attacker", "defender", "reinforcement"},
		Params:       map[string]float64{"defenseBonusRate": 0.15, "triggerChance": 1},
	}
	setTestFactionsAndGenerals(t, FactionsConfig{
		"wei": {Name: "魏国", Generals: []GeneralInfo{{ID: "caocao", Name: "曹操"}}},
	}, GeneralsConfig{Enabled: true, Heroes: map[string]GeneralHeroConfig{
		"caocao": {
			ID: "caocao", Name: "曹操", Faction: "wei", Enabled: true,
			BonusTrait: trait,
		},
	}})
	svc, repo, attacker, defender := newPvpTestServiceForGenerals(t, "wei", "caocao", "wei", "caocao")
	unitsMu.Lock()
	activeUnits["wei"]["huWei"] = UnitConfig{
		Name: "虎卫", Category: "infantry",
		Stats: map[string]int{"attack": 10, "infantryDefense": 10, "cavalryDefense": 8, "carryCapacity": 5, "upkeep": 1},
	}
	unitsMu.Unlock()
	attacker.Army = []ArmyUnit{{UnitType: "huWei", Amount: 100}}
	defender.Army = []ArmyUnit{{UnitType: "huWei", Amount: 100}}
	defender.Buildings = nil
	repo.players[attacker.Player.ID] = attacker
	repo.players[defender.Player.ID] = defender

	started, err := svc.StartPvpAttack(PvpAttackRequest{
		PlayerID: attacker.Player.ID, TargetPlayerID: defender.Player.ID, MarchMode: PvpMarchTypePlunder,
		Troops: map[string]int{"huWei": 100}, GeneralIDs: []string{"caocao"},
	})
	if err != nil {
		t.Fatalf("StartPvpAttack failed: %v", err)
	}
	forcePvpMarchDue(t, repo, started.March.ID)
	battle, err := svc.ResolvePvpMarch(started.March.ID)
	if err != nil {
		t.Fatalf("ResolvePvpMarch failed: %v", err)
	}
	if battle.Result["attackerPower"] != float64(1000) || battle.Result["defensePower"] != float64(1200) {
		t.Fatalf("expected attacker defense buff not to alter attack power and defender defense to reach 1200, got %v/%v", battle.Result["attackerPower"], battle.Result["defensePower"])
	}
	attackerLosses := pvpTestLossesFromBattle(t, battle, "attacker")
	defenderLosses := pvpTestLossesFromBattle(t, battle, "defender")
	if attackerLosses["huWei"] != 56 || defenderLosses["huWei"] != 43 {
		t.Fatalf("expected recalculated plunder losses 56/43, got attacker=%+v defender=%+v", attackerLosses, defenderLosses)
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
		assertSameWeiwuBothSidesReport(t, report)
	}

	storedMarch, err := repo.GetPvpMarch(started.March.ID)
	if err != nil || storedMarch.Status != PvpMarchStatusReturning || storedMarch.AttackTroops["huWei"] != 44 {
		t.Fatalf("expected 44 surviving attackers in return march, march=%+v err=%v", storedMarch, err)
	}
	storedAttacker, err := repo.GetState(attacker.Player.ID)
	if err != nil || pvpTestGeneralExp(storedAttacker, "caocao") != 43 {
		t.Fatalf("expected attacker Cao Cao exp 43, state=%+v err=%v", storedAttacker.Generals, err)
	}
	storedDefender, err := repo.GetState(defender.Player.ID)
	if err != nil || armySliceToMap(storedDefender.Army)["huWei"] != 57 || pvpTestGeneralExp(storedDefender, "caocao") != 56 {
		t.Fatalf("expected defender army 57 and Cao Cao exp 56, state=%+v err=%v", storedDefender, err)
	}
}

// assertSameWeiwuBothSidesReport 核对双方同名战前加成在旧战报和标准战报中都保持独立归属与真实整数变化。
func assertSameWeiwuBothSidesReport(t *testing.T, report BattleReport) {
	t.Helper()
	if len(report.TraitTriggered) != 2 || report.TraitTriggered[0] != "weiwu_tongyu" || report.TraitTriggered[1] != "weiwu_tongyu::defender::caocao" {
		t.Fatalf("expected ordered unique Weiwu storage keys, report=%s timeline=%+v", report.ID, report.TraitTriggered)
	}
	if len(report.TraitOutcomes) != 2 || report.Detail == nil || report.Detail.SecondarySide == nil || len(report.Detail.Traits) != 2 {
		t.Fatalf("expected two complete Weiwu outcomes in both report formats, report=%+v", report)
	}
	legacySides := map[string]bool{}
	for _, outcome := range report.TraitOutcomes {
		if outcome.TraitID != "weiwu_tongyu" || outcome.OwnerGeneralID != "caocao" {
			t.Fatalf("expected Cao Cao Weiwu legacy outcome, report=%s outcome=%+v", report.ID, outcome)
		}
		assertWeiwuActualDeltas(t, report.ID, outcome.Detail)
		legacySides[outcome.OwnerSide] = true
	}
	standardSides := map[string]bool{}
	for _, outcome := range report.Detail.Traits {
		if outcome.TraitID != "weiwu_tongyu" || outcome.GeneralID != "caocao" {
			t.Fatalf("expected Cao Cao Weiwu standard outcome, report=%s outcome=%+v", report.ID, outcome)
		}
		assertWeiwuActualDeltas(t, report.ID, outcome.Detail)
		standardSides[outcome.OwnerRole] = true
	}
	if !legacySides["attacker"] || !legacySides["defender"] || !standardSides["attacker"] || !standardSides["defender"] {
		t.Fatalf("expected independent attacker and defender Weiwu ownership, report=%s legacy=%+v standard=%+v", report.ID, report.TraitOutcomes, report.Detail.Traits)
	}
	seenPowers := map[int]bool{}
	seenLosses := map[int]bool{}
	for _, side := range []BattleReportSide{report.Detail.PrimarySide, *report.Detail.SecondarySide} {
		var huWei *BattleReportUnit
		for index := range side.Units {
			if side.Units[index].UnitType == "huWei" {
				huWei = &side.Units[index]
				break
			}
		}
		if huWei == nil || huWei.AmountBefore != 100 || huWei.Lost+huWei.Survived != 100 {
			t.Fatalf("expected each standard side row to reconcile 100 dispatched troops, report=%s side=%+v", report.ID, side)
		}
		seenPowers[side.Power] = true
		seenLosses[huWei.Lost] = true
	}
	if !seenPowers[1000] || !seenPowers[1200] || !seenLosses[56] || !seenLosses[43] {
		t.Fatalf("expected both power and loss views in standard report, report=%+v", report.Detail)
	}
}

// assertWeiwuActualDeltas 核对魏武统御记录的是应用后的真实防御整数变化而非仅保存配置比例。
func assertWeiwuActualDeltas(t *testing.T, reportID string, detail map[string]any) {
	t.Helper()
	infantry, infantryOK := detail["infantryDefenseModifiedUnits"].(map[string]int)
	cavalry, cavalryOK := detail["cavalryDefenseModifiedUnits"].(map[string]int)
	if !infantryOK || !cavalryOK || detail["defenseBonusRate"] != 0.15 || infantry["huWei"] != 2 || cavalry["huWei"] != 1 {
		t.Fatalf("expected actual HuWei defense deltas +2/+1, report=%s detail=%+v", reportID, detail)
	}
}

// assertSameTraitBothSidesReport 核对一份战报同时保留攻守双方同名特性的独立归属和实际增量。
func assertSameTraitBothSidesReport(t *testing.T, report BattleReport) {
	t.Helper()
	if len(report.TraitTriggered) != 2 || report.TraitTriggered[0] != "laodang_yizhuang" || report.TraitTriggered[1] != "laodang_yizhuang::defender::huangzhong" {
		t.Fatalf("expected ordered unique storage keys for two same-id outcomes, report=%s timeline=%+v", report.ID, report.TraitTriggered)
	}
	if len(report.TraitOutcomes) != 2 {
		t.Fatalf("expected two independently stored legacy outcomes, report=%s outcomes=%+v", report.ID, report.TraitOutcomes)
	}
	legacySides := map[string]bool{}
	for _, outcome := range report.TraitOutcomes {
		if outcome.TraitID != "laodang_yizhuang" || outcome.OwnerGeneralID != "huangzhong" {
			t.Fatalf("expected Huang Zhong same-id outcome, report=%s outcome=%+v", report.ID, outcome)
		}
		extra, ok := outcome.Detail["extraLosses"].(map[string]int)
		if !ok || extra["shuInfantry"] != 100 || outcome.Detail["effectRate"] != 0.1 {
			t.Fatalf("expected actual extra loss 100 and design rate 10%%, report=%s outcome=%+v", report.ID, outcome)
		}
		legacySides[outcome.OwnerSide] = true
	}
	if !legacySides["attacker"] || !legacySides["defender"] {
		t.Fatalf("expected attacker and defender legacy ownership, report=%s outcomes=%+v", report.ID, report.TraitOutcomes)
	}
	if report.LostUnits["shuInfantry"] != 600 || report.DefenderLostUnits["shuInfantry"] != 600 || report.GeneralExpGained != 600 {
		t.Fatalf("expected report losses and owner exp all equal 600, report=%+v", report)
	}
	if report.Detail == nil || report.Detail.SecondarySide == nil || len(report.Detail.Traits) != 2 {
		t.Fatalf("expected complete two-trait standard report, report=%s detail=%+v", report.ID, report.Detail)
	}
	standardSides := map[string]bool{}
	for _, outcome := range report.Detail.Traits {
		if outcome.TraitID != "laodang_yizhuang" || outcome.GeneralID != "huangzhong" || outcome.Detail["effectRate"] != 0.1 {
			t.Fatalf("expected standard Huang Zhong same-id outcome, report=%s outcome=%+v", report.ID, outcome)
		}
		extra, ok := outcome.Detail["extraLosses"].(map[string]int)
		if !ok || extra["shuInfantry"] != 100 {
			t.Fatalf("expected standard actual extra loss 100, report=%s outcome=%+v", report.ID, outcome)
		}
		standardSides[outcome.OwnerRole] = true
	}
	if !standardSides["attacker"] || !standardSides["defender"] {
		t.Fatalf("expected attacker and defender standard ownership, report=%s traits=%+v", report.ID, report.Detail.Traits)
	}
	for _, side := range []BattleReportSide{report.Detail.PrimarySide, *report.Detail.SecondarySide} {
		if len(side.Units) != 1 || side.Units[0].AmountBefore != 1000 || side.Units[0].Lost != 600 || side.Units[0].Survived != 400 {
			t.Fatalf("expected each standard side row 1000/600/400, report=%s side=%+v", report.ID, side)
		}
	}
}
