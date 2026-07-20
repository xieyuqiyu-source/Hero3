// 本文件验证正式进攻加成与防守加成由攻守双方同时触发时的战力、兵损和战报归属。
package game

import (
	"reflect"
	"testing"

	"hero3/internal/core/general"
)

// setAttackDefenseCrossUnits 固定本测试的虎卫与吴步兵基础属性，并在结束后恢复全局兵种配置。
func setAttackDefenseCrossUnits(t *testing.T) {
	t.Helper()
	unitsMu.Lock()
	if activeUnits["wei"] == nil {
		activeUnits["wei"] = FactionUnits{}
	}
	if activeUnits["wu"] == nil {
		activeUnits["wu"] = FactionUnits{}
	}
	previousWei, hadWei := activeUnits["wei"]["huWei"]
	previousWu, hadWu := activeUnits["wu"]["wuInfantry"]
	activeUnits["wei"]["huWei"] = UnitConfig{Name: "虎卫", Category: "infantry", Stats: map[string]int{"attack": 10, "infantryDefense": 10, "cavalryDefense": 8, "carryCapacity": 5, "upkeep": 1}}
	activeUnits["wu"]["wuInfantry"] = UnitConfig{Name: "吴步兵", Category: "infantry", Stats: map[string]int{"attack": 10, "infantryDefense": 10, "cavalryDefense": 8, "carryCapacity": 5, "upkeep": 1}}
	unitsMu.Unlock()
	t.Cleanup(func() {
		unitsMu.Lock()
		defer unitsMu.Unlock()
		if hadWei {
			activeUnits["wei"]["huWei"] = previousWei
		} else {
			delete(activeUnits["wei"], "huWei")
		}
		if hadWu {
			activeUnits["wu"]["wuInfantry"] = previousWu
		} else {
			delete(activeUnits["wu"], "wuInfantry")
		}
	})
}

// attackDefenseCrossReportUnit 返回标准战报指定一方的兵种行。
func attackDefenseCrossReportUnit(t *testing.T, side BattleReportSide, unitType string) BattleReportUnit {
	t.Helper()
	for _, unit := range side.Units {
		if unit.UnitType == unitType {
			return unit
		}
	}
	t.Fatalf("expected unit %s in side %+v", unitType, side)
	return BattleReportUnit{}
}

// TestPvpCaoCaoAttackBonusAgainstSunQuanDefenseBonus 验证魏武统御与江东固守在真实攻守交叉场景分别修改正确战斗输入。
func TestPvpCaoCaoAttackBonusAgainstSunQuanDefenseBonus(t *testing.T) {
	setTestFactionsAndGenerals(t, FactionsConfig{
		"wei": {Name: "魏国", Generals: []GeneralInfo{{ID: "caocao", Name: "曹操"}}},
		"wu":  {Name: "吴国", Generals: []GeneralInfo{{ID: "sunquan", Name: "孙权"}}},
	}, GeneralsConfig{Enabled: true, Heroes: map[string]GeneralHeroConfig{
		"caocao": {
			ID: "caocao", Name: "曹操", Faction: "wei", Enabled: true,
			SpecialTrait: GeneralTraitConfig{TraitID: "weiwu_haoling", TraitType: general.TraitTypeSpecial, Enabled: false, Scope: "self_city"},
			BonusTrait: GeneralTraitConfig{
				TraitID: "weiwu_tongyu", TraitType: general.TraitTypeBonus, Enabled: true, Scope: "self_army", TargetUnitType: "huWei",
				AllowedSides: []string{"attacker", "defender", "reinforcement"}, Params: map[string]float64{"attackBonusRate": 0.1, "defenseBonusRate": 0.1, "triggerChance": 1},
			},
		},
		"sunquan": {
			ID: "sunquan", Name: "孙权", Faction: "wu", Enabled: true,
			SpecialTrait: GeneralTraitConfig{TraitID: "jiangdong_haoling", TraitType: general.TraitTypeSpecial, Enabled: false, Scope: "enemy_plunder"},
			BonusTrait: GeneralTraitConfig{
				TraitID: "jiangdong_gushou", TraitType: general.TraitTypeBonus, Enabled: true, Scope: "self_army", AllowedSides: []string{"defender", "reinforcement"},
				Params: map[string]float64{"defenseBonusRate": 0.5, "triggerChance": 1},
			},
		},
	}})
	svc, repo, attacker, defender := newPvpTestServiceForGenerals(t, "wei", "caocao", "wu", "sunquan")
	setAttackDefenseCrossUnits(t)
	attacker.Army = []ArmyUnit{{UnitType: "huWei", Amount: 1000}}
	defender.Army = []ArmyUnit{{UnitType: "wuInfantry", Amount: 1000}}
	defender.Buildings = nil
	repo.players[attacker.Player.ID] = attacker
	repo.players[defender.Player.ID] = defender

	started, err := svc.StartPvpAttack(PvpAttackRequest{
		PlayerID: attacker.Player.ID, TargetPlayerID: defender.Player.ID, MarchMode: PvpMarchTypePlunder,
		Troops: map[string]int{"huWei": 1000}, GeneralIDs: []string{"caocao"},
	})
	if err != nil {
		t.Fatalf("StartPvpAttack failed: %v", err)
	}
	forcePvpMarchDue(t, repo, started.March.ID)
	battle, err := svc.ResolvePvpMarch(started.March.ID)
	if err != nil {
		t.Fatalf("ResolvePvpMarch failed: %v", err)
	}
	attackPower, attackOK := battle.Result["attackerPower"].(float64)
	defensePower, defenseOK := battle.Result["defensePower"].(float64)
	if !attackOK || !defenseOK || attackPower != 11000 || defensePower != 15000 {
		t.Fatalf("expected attack 10000 -> 11000 and defense 10000 -> 15000, result=%+v", battle.Result)
	}
	attackerReports, _, err := repo.ListReports(attacker.Player.ID, 10, 0)
	if err != nil || len(attackerReports) != 1 || attackerReports[0].ID != battle.AttackerReportID {
		t.Fatalf("expected attacker report, reports=%+v err=%v", attackerReports, err)
	}
	defenderReports, _, err := repo.ListReports(defender.Player.ID, 10, 0)
	if err != nil || len(defenderReports) != 1 || defenderReports[0].ID != battle.DefenderReportID {
		t.Fatalf("expected defender report, reports=%+v err=%v", defenderReports, err)
	}
	wantTimeline := []string{"weiwu_tongyu", "jiangdong_gushou"}
	for _, report := range []BattleReport{attackerReports[0], defenderReports[0]} {
		weiwu, weiwuOK := report.TraitOutcomes["weiwu_tongyu"]
		gushou, gushouOK := report.TraitOutcomes["jiangdong_gushou"]
		weiwuAttack, weiwuAttackOK := weiwu.Detail["attackModifiedUnits"].(map[string]int)
		gushouInfantry, gushouInfantryOK := gushou.Detail["infantryDefenseModifiedUnits"].(map[string]int)
		gushouCavalry, gushouCavalryOK := gushou.Detail["cavalryDefenseModifiedUnits"].(map[string]int)
		if !weiwuOK || !gushouOK || !weiwuAttackOK || !gushouInfantryOK || !gushouCavalryOK ||
			weiwu.OwnerSide != "attacker" || weiwu.OwnerGeneralID != "caocao" || weiwuAttack["huWei"] != 1 ||
			gushou.OwnerSide != "defender" || gushou.OwnerGeneralID != "sunquan" || gushouInfantry["wuInfantry"] != 5 || gushouCavalry["wuInfantry"] != 4 {
			t.Fatalf("expected independent attacker and defender actual deltas, report=%s outcomes=%+v", report.ID, report.TraitOutcomes)
		}
		if !reflect.DeepEqual(report.TraitTriggered, wantTimeline) || report.Detail == nil || len(report.Detail.Traits) != 2 ||
			report.Detail.Traits[0].TraitID != wantTimeline[0] || report.Detail.Traits[0].OwnerRole != "attacker" ||
			report.Detail.Traits[1].TraitID != wantTimeline[1] || report.Detail.Traits[1].OwnerRole != "defender" {
			t.Fatalf("expected attacker then defender prebattle timeline, report=%s legacy=%+v standard=%+v", report.ID, report.TraitTriggered, report.Detail)
		}
		attackerUnit := attackDefenseCrossReportUnit(t, report.Detail.PrimarySide, "huWei")
		defenderUnit := attackDefenseCrossReportUnit(t, *report.Detail.SecondarySide, "wuInfantry")
		if attackerUnit.AmountBefore != 1000 || attackerUnit.Lost != 608 || attackerUnit.Survived != 392 || defenderUnit.AmountBefore != 1000 || defenderUnit.Lost != 391 || defenderUnit.Survived != 609 {
			t.Fatalf("expected standard unit rows to reconcile 608/391 losses, report=%s attacker=%+v defender=%+v", report.ID, attackerUnit, defenderUnit)
		}
	}

	storedMarch, err := repo.GetPvpMarch(started.March.ID)
	if err != nil || storedMarch.Status != PvpMarchStatusReturning || storedMarch.AttackTroops["huWei"] != 392 {
		t.Fatalf("expected 392 HuWei returning, march=%+v err=%v", storedMarch, err)
	}
	storedDefender, err := repo.GetState(defender.Player.ID)
	if err != nil || armySliceToMap(storedDefender.Army)["wuInfantry"] != 609 {
		t.Fatalf("expected 609 defenders remaining, state=%+v err=%v", storedDefender, err)
	}
	storedAttacker, err := repo.GetState(attacker.Player.ID)
	if err != nil || pvpTestGeneralExp(storedAttacker, "caocao") != 391 || pvpTestGeneralExp(storedDefender, "sunquan") != 608 {
		t.Fatalf("expected general exp to follow real enemy losses 391/608, attacker=%+v defender=%+v err=%v", storedAttacker.Generals, storedDefender.Generals, err)
	}
}

// TestPvpZhenMiDefenseReductionThenSunQuanDefenseBonusReconcile 验证攻方破防后，守方加防读取当前整数属性并分别记录实际变化。
func TestPvpZhenMiDefenseReductionThenSunQuanDefenseBonusReconcile(t *testing.T) {
	setTestFactionsAndGenerals(t, FactionsConfig{
		"wei": {Name: "魏国", Generals: []GeneralInfo{{ID: "zhenmi", Name: "甄宓"}}},
		"wu":  {Name: "吴国", Generals: []GeneralInfo{{ID: "sunquan", Name: "孙权"}}},
	}, GeneralsConfig{Enabled: true, Heroes: map[string]GeneralHeroConfig{
		"zhenmi": {
			ID: "zhenmi", Name: "甄宓", Faction: "wei", Enabled: true,
			SpecialTrait: GeneralTraitConfig{TraitID: "meiren", TraitType: general.TraitTypeSpecial, Enabled: false, Scope: "self_army"},
			BonusTrait: GeneralTraitConfig{
				TraitID: "meihuo_raozhen", TraitType: general.TraitTypeBonus, Enabled: true, Scope: "enemy_army", AllowedSides: []string{"attacker"},
				Params: map[string]float64{"enemyDefenseReductionRate": 0.1},
			},
		},
		"sunquan": {
			ID: "sunquan", Name: "孙权", Faction: "wu", Enabled: true,
			SpecialTrait: GeneralTraitConfig{TraitID: "jiangdong_haoling", TraitType: general.TraitTypeSpecial, Enabled: false, Scope: "enemy_plunder"},
			BonusTrait: GeneralTraitConfig{
				TraitID: "jiangdong_gushou", TraitType: general.TraitTypeBonus, Enabled: true, Scope: "self_army", AllowedSides: []string{"defender", "reinforcement"},
				Params: map[string]float64{"defenseBonusRate": 0.5, "triggerChance": 1},
			},
		},
	}})
	svc, repo, attacker, defender := newPvpTestServiceForGenerals(t, "wei", "zhenmi", "wu", "sunquan")
	setAttackDefenseCrossUnits(t)
	attacker.Army = []ArmyUnit{{UnitType: "huWei", Amount: 1000}}
	defender.Army = []ArmyUnit{{UnitType: "wuInfantry", Amount: 1000}}
	defender.Buildings = nil
	repo.players[attacker.Player.ID] = attacker
	repo.players[defender.Player.ID] = defender

	started, err := svc.StartPvpAttack(PvpAttackRequest{
		PlayerID: attacker.Player.ID, TargetPlayerID: defender.Player.ID, MarchMode: PvpMarchTypePlunder,
		Troops: map[string]int{"huWei": 1000}, GeneralIDs: []string{"zhenmi"},
	})
	if err != nil {
		t.Fatalf("StartPvpAttack failed: %v", err)
	}
	forcePvpMarchDue(t, repo, started.March.ID)
	battle, err := svc.ResolvePvpMarch(started.March.ID)
	if err != nil {
		t.Fatalf("ResolvePvpMarch failed: %v", err)
	}
	if battle.Result["attackerPower"] != float64(10000) || battle.Result["defensePower"] != float64(14000) {
		t.Fatalf("expected defense 10/8 -> 9/7 -> 14/11 and power 10000/14000, result=%+v", battle.Result)
	}

	attackerReports, _, err := repo.ListReports(attacker.Player.ID, 10, 0)
	if err != nil || len(attackerReports) != 1 || attackerReports[0].ID != battle.AttackerReportID {
		t.Fatalf("expected attacker report, reports=%+v err=%v", attackerReports, err)
	}
	defenderReports, _, err := repo.ListReports(defender.Player.ID, 10, 0)
	if err != nil || len(defenderReports) != 1 || defenderReports[0].ID != battle.DefenderReportID {
		t.Fatalf("expected defender report, reports=%+v err=%v", defenderReports, err)
	}
	wantTimeline := []string{"meihuo_raozhen", "jiangdong_gushou"}
	for _, report := range []BattleReport{attackerReports[0], defenderReports[0]} {
		reduction, reductionOK := report.TraitOutcomes["meihuo_raozhen"]
		bonus, bonusOK := report.TraitOutcomes["jiangdong_gushou"]
		reductionInfantry, reductionInfantryOK := reduction.Detail["infantryDefenseModifiedUnits"].(map[string]int)
		reductionCavalry, reductionCavalryOK := reduction.Detail["cavalryDefenseModifiedUnits"].(map[string]int)
		bonusInfantry, bonusInfantryOK := bonus.Detail["infantryDefenseModifiedUnits"].(map[string]int)
		bonusCavalry, bonusCavalryOK := bonus.Detail["cavalryDefenseModifiedUnits"].(map[string]int)
		if !reductionOK || !bonusOK || !reductionInfantryOK || !reductionCavalryOK || !bonusInfantryOK || !bonusCavalryOK ||
			reduction.OwnerSide != "attacker" || reduction.OwnerGeneralID != "zhenmi" || reduction.Detail["enemyDefenseReductionRate"] != 0.1 ||
			reductionInfantry["wuInfantry"] != -1 || reductionCavalry["wuInfantry"] != -1 ||
			bonus.OwnerSide != "defender" || bonus.OwnerGeneralID != "sunquan" || bonus.Detail["defenseBonusRate"] != 0.5 ||
			bonusInfantry["wuInfantry"] != 5 || bonusCavalry["wuInfantry"] != 4 {
			t.Fatalf("expected sequential -1/-1 then +5/+4 defense outcomes, report=%s outcomes=%+v", report.ID, report.TraitOutcomes)
		}
		if !reflect.DeepEqual(report.TraitTriggered, wantTimeline) || report.Detail == nil || len(report.Detail.Traits) != 2 ||
			report.Detail.Traits[0].TraitID != wantTimeline[0] || report.Detail.Traits[0].OwnerRole != "attacker" ||
			report.Detail.Traits[1].TraitID != wantTimeline[1] || report.Detail.Traits[1].OwnerRole != "defender" {
			t.Fatalf("expected reduction then bonus timeline, report=%s legacy=%+v standard=%+v", report.ID, report.TraitTriggered, report.Detail)
		}
		attackerUnit := attackDefenseCrossReportUnit(t, report.Detail.PrimarySide, "huWei")
		defenderUnit := attackDefenseCrossReportUnit(t, *report.Detail.SecondarySide, "wuInfantry")
		if attackerUnit.AmountBefore != 1000 || attackerUnit.Lost != 617 || attackerUnit.Survived != 383 ||
			defenderUnit.AmountBefore != 1000 || defenderUnit.Lost != 382 || defenderUnit.Survived != 618 {
			t.Fatalf("expected standard unit rows to reconcile 617/382 losses, report=%s attacker=%+v defender=%+v", report.ID, attackerUnit, defenderUnit)
		}
	}

	storedMarch, err := repo.GetPvpMarch(started.March.ID)
	if err != nil || storedMarch.Status != PvpMarchStatusReturning || storedMarch.AttackTroops["huWei"] != 383 {
		t.Fatalf("expected 383 HuWei returning, march=%+v err=%v", storedMarch, err)
	}
	storedDefender, err := repo.GetState(defender.Player.ID)
	if err != nil || armySliceToMap(storedDefender.Army)["wuInfantry"] != 618 {
		t.Fatalf("expected 618 defenders remaining, state=%+v err=%v", storedDefender, err)
	}
	storedAttacker, err := repo.GetState(attacker.Player.ID)
	if err != nil || pvpTestGeneralExp(storedAttacker, "zhenmi") != 382 || pvpTestGeneralExp(storedDefender, "sunquan") != 617 {
		t.Fatalf("expected general exp to follow real enemy losses 382/617, attacker=%+v defender=%+v err=%v", storedAttacker.Generals, storedDefender.Generals, err)
	}
}
