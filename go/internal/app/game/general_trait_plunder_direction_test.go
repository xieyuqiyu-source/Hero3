// 本文件验证掠夺特性处于错误攻防方向时，不会修改真实战力、资源和战报时间线。
package game

import (
	"testing"
	"time"

	"hero3/internal/core/general"
)

// TestPvpPlunderTraitsDoNotTriggerOnWrongOwnerSides 验证孙权主动进攻、甘宁守城时三项掠夺能力全部无效。
func TestPvpPlunderTraitsDoNotTriggerOnWrongOwnerSides(t *testing.T) {
	setTestFactionsAndGenerals(t, FactionsConfig{
		"wu": {Name: "吴国", Generals: []GeneralInfo{{ID: "sunquan", Name: "孙权"}, {ID: "ganning", Name: "甘宁"}}},
	}, GeneralsConfig{Enabled: true, Heroes: map[string]GeneralHeroConfig{
		"sunquan": {
			ID: "sunquan", Name: "孙权", Faction: "wu", Enabled: true,
			SpecialTrait: GeneralTraitConfig{
				TraitID: "jiangdong_haoling", TraitType: general.TraitTypeSpecial, Enabled: true,
				Scope: "enemy_plunder", AllowedSides: []string{"defender"}, AllowedScenes: []string{"plunder"},
				Params: map[string]float64{"plunderBonusRate": -0.2, "triggerChance": 1},
			},
		},
		"ganning": {
			ID: "ganning", Name: "甘宁", Faction: "wu", Enabled: true,
			SpecialTrait: GeneralTraitConfig{
				TraitID: "jinfan_jielue", TraitType: general.TraitTypeSpecial, Enabled: true,
				Scope: "self_plunder", AllowedSides: []string{"attacker"}, AllowedScenes: []string{"plunder"},
				Params: map[string]float64{"plunderBonusRate": 0.2, "triggerChance": 1},
			},
			BonusTrait: GeneralTraitConfig{
				TraitID: "jinfan_qixi", TraitType: general.TraitTypeBonus, Enabled: true,
				Scope: "self_army", AllowedSides: []string{"attacker"}, AllowedScenes: []string{"plunder"},
				Params: map[string]float64{"attackBonusRate": 0.1, "triggerChance": 1},
			},
		},
	}})
	svc, repo, attacker, defender := newPvpTestServiceForGenerals(t, "wu", "sunquan", "wu", "ganning")
	unitsMu.Lock()
	activeUnits["wu"]["wuInfantry"] = UnitConfig{
		Name: "吴步兵", Category: "infantry",
		Stats: map[string]int{"attack": 10, "infantryDefense": 10, "cavalryDefense": 8, "carryCapacity": 5, "upkeep": 1},
	}
	unitsMu.Unlock()
	attacker.Army = []ArmyUnit{{UnitType: "wuInfantry", Amount: 1000}}
	attacker.Buildings = []Building{{ID: "warehouse-1", Type: "warehouse", Level: 10}}
	attacker.Resources.Items = map[string]int{"wood": 0, "stone": 0, "iron": 0, "food": 0}
	attacker.Resources.Capacity = map[string]int{"wood": 100000, "stone": 100000, "iron": 100000, "food": 100000}
	defender.Army = []ArmyUnit{{UnitType: "wuInfantry", Amount: 10}}
	defender.Buildings = nil
	defender.Resources.Items = map[string]int{"wood": 10000, "stone": 0, "iron": 0, "food": 0}
	defender.Resources.Capacity = map[string]int{"wood": 0, "stone": 0, "iron": 0, "food": 0}
	nowText := time.Now().UTC().Format(resourceDateLayout)
	attacker.ResourceSettledAt = nowText
	defender.ResourceSettledAt = nowText
	repo.players[attacker.Player.ID] = attacker
	repo.players[defender.Player.ID] = defender

	started, err := svc.StartPvpAttack(PvpAttackRequest{
		PlayerID: attacker.Player.ID, TargetPlayerID: defender.Player.ID, MarchMode: PvpMarchTypePlunder,
		Troops: map[string]int{"wuInfantry": 1000}, GeneralIDs: []string{"sunquan"},
	})
	if err != nil {
		t.Fatalf("StartPvpAttack failed: %v", err)
	}
	forcePvpMarchDue(t, repo, started.March.ID)
	battle, err := svc.ResolvePvpMarch(started.March.ID)
	if err != nil {
		t.Fatalf("ResolvePvpMarch failed: %v", err)
	}
	if battle.Result["winner"] != "attacker" || battle.Result["attackerPower"] != float64(10000) || battle.Result["defensePower"] != float64(100) {
		t.Fatalf("expected successful base plunder with unmodified powers 10000/100, got %+v", battle.Result)
	}
	attackerLosses := pvpTestLossesFromBattle(t, battle, "attacker")
	defenderLosses := pvpTestLossesFromBattle(t, battle, "defender")
	if attackerLosses["wuInfantry"] != 1 || defenderLosses["wuInfantry"] != 9 {
		t.Fatalf("expected unmodified core losses 1/9, got attacker=%+v defender=%+v", attackerLosses, defenderLosses)
	}
	attackerSurvived := 1000 - attackerLosses["wuInfantry"]
	wantWood := attackerSurvived * 5
	if battle.Plunder["wood"] != wantWood {
		t.Fatalf("expected base plunder to equal surviving carry %d without trait modifiers, got %+v", wantWood, battle.Plunder)
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
		assertWrongSidePlunderTraitsAbsent(t, report, attackerLosses["wuInfantry"], defenderLosses["wuInfantry"], attackerSurvived, wantWood)
	}

	storedAttacker, err := repo.GetState(attacker.Player.ID)
	if err != nil || storedAttacker.Resources.Items["wood"] != wantWood || pvpTestGeneralExp(storedAttacker, "sunquan") != defenderLosses["wuInfantry"] {
		t.Fatalf("expected attacker resources and Sun Quan exp to match base settlement, state=%+v err=%v", storedAttacker, err)
	}
	storedDefender, err := repo.GetState(defender.Player.ID)
	if err != nil || storedDefender.Resources.Items["wood"] != 10000-wantWood || armySliceToMap(storedDefender.Army)["wuInfantry"] != 10-defenderLosses["wuInfantry"] || pvpTestGeneralExp(storedDefender, "ganning") != attackerLosses["wuInfantry"] {
		t.Fatalf("expected defender resources, army and Gan Ning exp to match base settlement, state=%+v err=%v", storedDefender, err)
	}
	storedMarch, err := repo.GetPvpMarch(started.March.ID)
	if err != nil || storedMarch.Status != PvpMarchStatusReturning || storedMarch.AttackTroops["wuInfantry"] != attackerSurvived {
		t.Fatalf("expected unmodified surviving attackers in return march, march=%+v err=%v", storedMarch, err)
	}
}

// assertWrongSidePlunderTraitsAbsent 核对拥有特性快照仍存在时，错误方向不会产生结果或伪造时间线。
func assertWrongSidePlunderTraitsAbsent(t *testing.T, report BattleReport, attackerLost int, defenderLost int, attackerSurvived int, wood int) {
	t.Helper()
	for _, traitID := range []string{"jiangdong_haoling", "jinfan_jielue", "jinfan_qixi"} {
		if _, triggered := report.TraitOutcomes[traitID]; triggered || standardReportHasTrait(report.Detail, traitID) {
			t.Fatalf("expected wrong-side trait %s to stay absent, report=%+v", traitID, report)
		}
	}
	if len(report.TraitTriggered) != 0 || report.Detail == nil || report.Detail.SecondarySide == nil || len(report.Detail.Traits) != 0 {
		t.Fatalf("expected empty trait timeline in both report formats, report=%+v", report)
	}
	wantExp := defenderLost
	if report.OwnerSide == "attacker" {
		if report.Rewards["wood"] != wood || report.Detail.Rewards.Resources["wood"] != wood {
			t.Fatalf("expected attacker report to record base wood reward %d, report=%+v", wood, report)
		}
	} else {
		wantExp = attackerLost
		if report.Rewards["wood"] != 0 || report.Detail.Rewards.Resources["wood"] != 0 || report.DefenderResources["wood"] != wood {
			t.Fatalf("expected defender report to record lost wood %d without treating it as a reward, report=%+v", wood, report)
		}
	}
	if report.GeneralExpGained != wantExp {
		t.Fatalf("expected owner general exp %d for %s report, report=%+v", wantExp, report.OwnerSide, report)
	}
	attackerSide, defenderSide := standardReportSidesByRole(report.Detail)
	if attackerSide == nil || defenderSide == nil || !reportSideGeneralOwnsTrait(*attackerSide, "sunquan", "jiangdong_haoling") || !reportSideGeneralOwnsTrait(*defenderSide, "ganning", "jinfan_jielue") || !reportSideGeneralOwnsTrait(*defenderSide, "ganning", "jinfan_qixi") {
		t.Fatalf("expected ownership snapshots without triggered outcomes, report=%+v", report)
	}
	assertStandardUnitRow(t, report.ID, *attackerSide, "wuInfantry", 1000, attackerLost, attackerSurvived)
	assertStandardUnitRow(t, report.ID, *defenderSide, "wuInfantry", 10, defenderLost, 10-defenderLost)
}

// standardReportSidesByRole 按实际攻防角色读取标准战报两侧，兼容进攻与防守查看视角。
func standardReportSidesByRole(detail *BattleReportDetail) (*BattleReportSide, *BattleReportSide) {
	if detail == nil || detail.SecondarySide == nil {
		return nil, nil
	}
	sides := []*BattleReportSide{&detail.PrimarySide, detail.SecondarySide}
	var attacker *BattleReportSide
	var defender *BattleReportSide
	for _, side := range sides {
		if side.Role == "attacker" {
			attacker = side
		} else if side.Role == "defender" {
			defender = side
		}
	}
	return attacker, defender
}

// reportSideGeneralOwnsTrait 判断指定标准战报一侧的将领快照是否保留某项拥有特性。
func reportSideGeneralOwnsTrait(side BattleReportSide, generalID string, traitID string) bool {
	for _, snapshot := range side.Generals {
		if snapshot.ID != generalID {
			continue
		}
		for _, trait := range snapshot.Traits {
			if trait.TraitID == traitID {
				return true
			}
		}
	}
	return false
}

// assertStandardUnitRow 核对标准战报中的指定兵种出动、阵亡和最终存活。
func assertStandardUnitRow(t *testing.T, reportID string, side BattleReportSide, unitType string, dispatched int, lost int, survived int) {
	t.Helper()
	for _, unit := range side.Units {
		if unit.UnitType == unitType {
			if unit.AmountBefore != dispatched || unit.Dispatched != dispatched || unit.Lost != lost || unit.Survived != survived {
				t.Fatalf("expected standard unit row %d/%d/%d, report=%s unit=%+v", dispatched, lost, survived, reportID, unit)
			}
			return
		}
	}
	t.Fatalf("expected unit %s in standard report %s, side=%+v", unitType, reportID, side)
}
