// 本文件验证美人计的单兵种俘虏上限会同时约束真实驻防、目标库存和双方战报。
package game

import (
	"testing"

	"hero3/internal/core/general"
)

// TestPvpMeirenCapsEachUnitTypeAndReconcilesRealState 验证两个兵种分别触顶时各俘虏 1000，总数和权威状态准确对账。
func TestPvpMeirenCapsEachUnitTypeAndReconcilesRealState(t *testing.T) {
	setTestFactionsAndGenerals(t, FactionsConfig{
		"wei": {Name: "魏国", Generals: []GeneralInfo{{ID: "zhenmi", Name: "甄宓"}}},
		"shu": {Name: "蜀国", Generals: []GeneralInfo{{ID: "liubei", Name: "刘备"}}},
	}, GeneralsConfig{Enabled: true, Heroes: map[string]GeneralHeroConfig{
		"zhenmi": {
			ID: "zhenmi", Name: "甄宓", Faction: "wei", Enabled: true,
			SpecialTrait: GeneralTraitConfig{
				TraitID: "meiren", TraitType: general.TraitTypeSpecial, Enabled: true, Scope: "self_army",
				AllowedSides: []string{"attacker"},
				Params:       map[string]float64{"captureRate": 0.2, "captureMax": 1000, "triggerChance": 1},
			},
		},
		"liubei": {ID: "liubei", Name: "刘备", Faction: "shu", Enabled: true},
	}})
	svc, repo, attacker, defender := newPvpTestServiceForGenerals(t, "wei", "zhenmi", "shu", "liubei")
	unitsMu.Lock()
	activeUnits["shu"]["shuInfantry"] = UnitConfig{
		Name: "蜀步兵", Category: "infantry",
		Stats: map[string]int{"attack": 10, "infantryDefense": 10, "cavalryDefense": 8, "carryCapacity": 5, "upkeep": 1},
	}
	activeUnits["shu"]["shuCavalry"] = UnitConfig{
		Name: "蜀骑兵", Category: "cavalry",
		Stats: map[string]int{"attack": 14, "infantryDefense": 8, "cavalryDefense": 10, "carryCapacity": 6, "upkeep": 2},
	}
	unitsMu.Unlock()
	attacker.Army = []ArmyUnit{{UnitType: "weiInfantry", Amount: 1}}
	defender.Army = []ArmyUnit{{UnitType: "shuInfantry", Amount: 20000}, {UnitType: "shuCavalry", Amount: 20000}}
	defender.Buildings = nil
	repo.players[attacker.Player.ID] = attacker
	repo.players[defender.Player.ID] = defender

	started, err := svc.StartPvpAttack(PvpAttackRequest{
		PlayerID: attacker.Player.ID, TargetPlayerID: defender.Player.ID, MarchMode: PvpMarchTypeAttack,
		Troops: map[string]int{"weiInfantry": 1}, GeneralIDs: []string{"zhenmi"},
	})
	if err != nil {
		t.Fatalf("StartPvpAttack failed: %v", err)
	}
	forcePvpMarchDue(t, repo, started.March.ID)
	battle, err := svc.ResolvePvpMarch(started.March.ID)
	if err != nil {
		t.Fatalf("ResolvePvpMarch failed: %v", err)
	}
	if battle.Result["winner"] != "defender" || battle.Result["attackerPower"] != float64(10) || battle.Result["defensePower"] != float64(342000) {
		t.Fatalf("expected captures to leave real battle powers 10/342000 and defender victory, got %+v", battle.Result)
	}
	attackerLosses := pvpTestLossesFromBattle(t, battle, "attacker")
	defenderTotalLosses := pvpTestLossesFromBattle(t, battle, "defender")
	defenderDeaths := map[string]int{
		"shuInfantry": max(0, defenderTotalLosses["shuInfantry"]-1000),
		"shuCavalry":  max(0, defenderTotalLosses["shuCavalry"]-1000),
	}
	if attackerLosses["weiInfantry"] != 1 || defenderTotalLosses["shuInfantry"] != 1000 || defenderTotalLosses["shuCavalry"] != 1000 || totalTroops(defenderDeaths) != 0 {
		t.Fatalf("expected one attacker death, 1000 captures per defender unit and no defender deaths, attacker=%+v defenderTotal=%+v defenderDeaths=%+v", attackerLosses, defenderTotalLosses, defenderDeaths)
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
		assertMeirenPerUnitCapReport(t, report, attackerLosses, defenderTotalLosses, defenderDeaths)
	}

	garrison, err := repo.GetReinforcement(ObtainedGarrisonID(attacker.Player.ID))
	if err != nil || garrison.RemainingTroops["shuInfantry"] != 1000 || garrison.RemainingTroops["shuCavalry"] != 1000 || totalTroops(garrison.RemainingTroops) != 2000 {
		t.Fatalf("expected real obtained garrison 1000 per unit and 2000 total, garrison=%+v err=%v", garrison, err)
	}
	storedDefender, err := repo.GetState(defender.Player.ID)
	if err != nil {
		t.Fatalf("GetState defender failed: %v", err)
	}
	wantInfantry := 20000 - defenderTotalLosses["shuInfantry"]
	wantCavalry := 20000 - defenderTotalLosses["shuCavalry"]
	defenderArmy := armySliceToMap(storedDefender.Army)
	if defenderArmy["shuInfantry"] != wantInfantry || defenderArmy["shuCavalry"] != wantCavalry {
		t.Fatalf("expected defender inventory to subtract captures and deaths, want=%d/%d state=%+v totalLosses=%+v", wantInfantry, wantCavalry, defenderArmy, defenderTotalLosses)
	}
	storedAttacker, err := repo.GetState(attacker.Player.ID)
	if err != nil || pvpTestGeneralExp(storedAttacker, "zhenmi") != totalTroops(defenderDeaths) {
		t.Fatalf("expected Zhen Mi exp to equal real deaths only, state=%+v deaths=%+v err=%v", storedAttacker.Generals, defenderDeaths, err)
	}
	if pvpTestGeneralExp(storedDefender, "liubei") != totalTroops(attackerLosses) {
		t.Fatalf("expected Liu Bei exp to equal attacker deaths only, state=%+v losses=%+v", storedDefender.Generals, attackerLosses)
	}

	if _, err := svc.ResolvePvpMarch(started.March.ID); err != nil {
		t.Fatalf("repeated ResolvePvpMarch failed: %v", err)
	}
	garrison, err = repo.GetReinforcement(ObtainedGarrisonID(attacker.Player.ID))
	if err != nil || garrison.RemainingTroops["shuInfantry"] != 1000 || garrison.RemainingTroops["shuCavalry"] != 1000 {
		t.Fatalf("expected repeated settlement not to duplicate capped captures, garrison=%+v err=%v", garrison, err)
	}
}

// assertMeirenPerUnitCapReport 核对美人计每兵种上限、总俘虏数、兵力行和查看方经验。
func assertMeirenPerUnitCapReport(t *testing.T, report BattleReport, attackerLosses map[string]int, defenderTotalLosses map[string]int, defenderDeaths map[string]int) {
	t.Helper()
	if report.CapturedToGarrison["shuInfantry"] != 1000 || report.CapturedToGarrison["shuCavalry"] != 1000 || totalTroops(report.CapturedToGarrison) != 2000 || len(report.CapturedUnits) != 0 {
		t.Fatalf("expected report capture 1000 per unit and 2000 total to garrison, report=%+v", report)
	}
	outcome, ok := report.TraitOutcomes["meiren"]
	if !ok || outcome.Detail["captureRate"] != 0.2 || outcome.Detail["captureMax"] != 1000 || outcome.Detail["totalCaptured"] != 2000 || outcome.Detail["triggerChance"] != 1.0 {
		t.Fatalf("expected complete Meiren design and total fields, report=%s outcome=%+v", report.ID, outcome)
	}
	captured, ok := outcome.Detail["capturedToGarrison"].(map[string]int)
	if !ok || captured["shuInfantry"] != 1000 || captured["shuCavalry"] != 1000 {
		t.Fatalf("expected actual capped per-unit capture detail, report=%s outcome=%+v", report.ID, outcome)
	}
	if report.Detail == nil || len(report.Detail.Traits) != 1 || report.Detail.Traits[0].TraitID != "meiren" {
		t.Fatalf("expected one Meiren result in standard timeline, report=%+v", report)
	}
	standard := report.Detail.Traits[0]
	standardCaptured, ok := standard.Detail["capturedToGarrison"].(map[string]int)
	if !ok || standard.Detail["captureRate"] != 0.2 || standard.Detail["captureMax"] != 1000 || standard.Detail["totalCaptured"] != 2000 || standardCaptured["shuInfantry"] != 1000 || standardCaptured["shuCavalry"] != 1000 {
		t.Fatalf("expected standard Meiren fields to match legacy result, report=%s trait=%+v", report.ID, standard)
	}
	attackerSide, defenderSide := standardReportSidesByRole(report.Detail)
	if attackerSide == nil || defenderSide == nil {
		t.Fatalf("expected complete attacker and defender sides, report=%+v", report)
	}
	assertStandardUnitRow(t, report.ID, *attackerSide, "weiInfantry", 1, attackerLosses["weiInfantry"], 1-attackerLosses["weiInfantry"])
	assertStandardUnitRow(t, report.ID, *defenderSide, "shuInfantry", 20000, defenderDeaths["shuInfantry"], 20000-defenderTotalLosses["shuInfantry"])
	assertStandardUnitRow(t, report.ID, *defenderSide, "shuCavalry", 20000, defenderDeaths["shuCavalry"], 20000-defenderTotalLosses["shuCavalry"])
	wantExp := totalTroops(defenderDeaths)
	if report.OwnerSide == "defender" {
		wantExp = totalTroops(attackerLosses)
	}
	if report.GeneralExpGained != wantExp {
		t.Fatalf("expected owner exp %d to exclude captured soldiers, report=%+v", wantExp, report)
	}
}
