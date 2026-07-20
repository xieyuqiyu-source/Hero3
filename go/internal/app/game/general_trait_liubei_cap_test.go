// 本文件验证刘备双返兵在大额多兵种阵亡时按稳定顺序执行单场总上限，并与真实返程和战报一致。
package game

import (
	"testing"

	"hero3/internal/core/general"
)

// TestPvpLiubeiReturnCapsUseStableUnitOrderAndRealMarch 验证仁德封顶 10000 后，仁主守护只按剩余真实损失稳定返兵。
func TestPvpLiubeiReturnCapsUseStableUnitOrderAndRealMarch(t *testing.T) {
	setTestFactionsAndGenerals(t, FactionsConfig{
		"shu": {Name: "蜀国", Generals: []GeneralInfo{{ID: "liubei", Name: "刘备"}}},
		"wei": {Name: "魏国", Generals: []GeneralInfo{{ID: "caocao", Name: "曹操"}}},
	}, GeneralsConfig{Enabled: true, Heroes: map[string]GeneralHeroConfig{
		"liubei": {
			ID: "liubei", Name: "刘备", Faction: "shu", Enabled: true,
			SpecialTrait: GeneralTraitConfig{
				TraitID: "rende", TraitType: general.TraitTypeSpecial, Enabled: true, Scope: "self_army",
				Params: map[string]float64{"effectRate": 0.5, "reviveRate": 0.5, "maxReviveCount": 10000, "triggerChance": 1},
			},
			BonusTrait: GeneralTraitConfig{
				TraitID: "renzhu_shouhu", TraitType: general.TraitTypeBonus, Enabled: true, Scope: "self_army",
				Params: map[string]float64{"lossReductionRate": 0.1, "maxReturnCount": 10000, "triggerChance": 1},
			},
		},
		"caocao": {ID: "caocao", Name: "曹操", Faction: "wei", Enabled: true},
	}})
	svc, repo, attacker, defender := newPvpTestServiceForGenerals(t, "shu", "liubei", "wei", "caocao")
	unitsMu.Lock()
	activeUnits["shu"]["shuInfantry"] = UnitConfig{
		Name: "蜀步兵", Category: "infantry",
		Stats: map[string]int{"attack": 10, "infantryDefense": 10, "cavalryDefense": 8, "carryCapacity": 5, "upkeep": 1},
	}
	activeUnits["shu"]["shuCavalry"] = UnitConfig{
		Name: "蜀骑兵", Category: "cavalry",
		Stats: map[string]int{"attack": 14, "infantryDefense": 8, "cavalryDefense": 10, "carryCapacity": 6, "upkeep": 2},
	}
	activeUnits["wei"]["weiInfantry"] = UnitConfig{
		Name: "魏步兵", Category: "infantry",
		Stats: map[string]int{"attack": 10, "infantryDefense": 10, "cavalryDefense": 8, "carryCapacity": 5, "upkeep": 1},
	}
	unitsMu.Unlock()
	attacker.Army = []ArmyUnit{{UnitType: "shuInfantry", Amount: 30000}, {UnitType: "shuCavalry", Amount: 30000}}
	defender.Army = []ArmyUnit{{UnitType: "weiInfantry", Amount: 1000000}}
	defender.Buildings = nil
	repo.players[attacker.Player.ID] = attacker
	repo.players[defender.Player.ID] = defender

	started, err := svc.StartPvpAttack(PvpAttackRequest{
		PlayerID: attacker.Player.ID, TargetPlayerID: defender.Player.ID, MarchMode: PvpMarchTypeAttack,
		Troops: map[string]int{"shuInfantry": 30000, "shuCavalry": 30000}, GeneralIDs: []string{"liubei"},
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
	defenderLosses := pvpTestLossesFromBattle(t, battle, "defender")
	if battle.Result["winner"] != "defender" || attackerLosses["shuCavalry"] != 30000 || attackerLosses["shuInfantry"] != 30000 {
		t.Fatalf("expected defender victory and complete original attacker losses, result=%+v losses=%+v", battle.Result, attackerLosses)
	}
	attackerExp := calculateGeneralBattleExpFromLosses("wei", pvpTestUnitLosses(defenderLosses))
	defenderExp := calculateGeneralBattleExpFromLosses("shu", pvpTestUnitLosses(attackerLosses))

	attackerReports, _, err := repo.ListReports(attacker.Player.ID, 10, 0)
	if err != nil || len(attackerReports) == 0 || attackerReports[0].ID != battle.AttackerReportID {
		t.Fatalf("expected attacker report, reports=%+v err=%v", attackerReports, err)
	}
	defenderReports, _, err := repo.ListReports(defender.Player.ID, 10, 0)
	if err != nil || len(defenderReports) == 0 || defenderReports[0].ID != battle.DefenderReportID {
		t.Fatalf("expected defender report, reports=%+v err=%v", defenderReports, err)
	}
	for _, report := range []BattleReport{attackerReports[0], defenderReports[0]} {
		assertLiubeiStableReturnCapReport(t, report, attackerExp, defenderExp)
	}

	storedMarch, err := repo.GetPvpMarch(started.March.ID)
	if err != nil || storedMarch.Status != PvpMarchStatusReturning || storedMarch.AttackTroops["shuCavalry"] != 13000 || storedMarch.AttackTroops["shuInfantry"] != 3000 || totalTroops(storedMarch.AttackTroops) != 16000 {
		t.Fatalf("expected stable capped return march cavalry/infantry 13000/3000, march=%+v err=%v", storedMarch, err)
	}
	storedAttacker, err := repo.GetState(attacker.Player.ID)
	if err != nil || pvpTestGeneralExp(storedAttacker, "liubei") != attackerExp {
		t.Fatalf("expected Liu Bei exp to equal real defender deaths, state=%+v losses=%+v err=%v", storedAttacker.Generals, defenderLosses, err)
	}
}

// assertLiubeiStableReturnCapReport 核对刘备两项能力各自的设计上限、实际逐兵种返还和最终兵力行。
func assertLiubeiStableReturnCapReport(t *testing.T, report BattleReport, attackerExp int, defenderExp int) {
	t.Helper()
	if len(report.TraitTriggered) != 2 || report.TraitTriggered[0] != "rende" || report.TraitTriggered[1] != "renzhu_shouhu" || len(report.TraitOutcomes) != 2 {
		t.Fatalf("expected Rende then Renzhu timeline, report=%+v", report)
	}
	rende := report.TraitOutcomes["rende"]
	rendeUnits, rendeOK := rende.Detail["revivedUnits"].(map[string]int)
	if !rendeOK || rende.Detail["effectRate"] != 0.5 || rende.Detail["maxReviveCount"] != 10000 || rende.Detail["totalRevived"] != 10000 || rende.Detail["triggerChance"] != 1.0 || rendeUnits["shuCavalry"] != 10000 || rendeUnits["shuInfantry"] != 0 {
		t.Fatalf("expected Rende actual stable cap 10000 on cavalry, report=%s outcome=%+v", report.ID, rende)
	}
	guard := report.TraitOutcomes["renzhu_shouhu"]
	guardUnits, guardOK := guard.Detail["returnedUnits"].(map[string]int)
	if !guardOK || guard.Detail["lossReductionRate"] != 0.1 || guard.Detail["maxReturnCount"] != 10000 || guard.Detail["triggerChance"] != 1.0 || guardUnits["shuCavalry"] != 3000 || guardUnits["shuInfantry"] != 3000 {
		t.Fatalf("expected Renzhu actual returns 3000 per unit after Rende, report=%s outcome=%+v", report.ID, guard)
	}
	if report.Detail == nil || len(report.Detail.Traits) != 2 || report.Detail.Traits[0].TraitID != "rende" || report.Detail.Traits[1].TraitID != "renzhu_shouhu" {
		t.Fatalf("expected complete standard return timeline, report=%+v", report)
	}
	attackerSide, defenderSide := standardReportSidesByRole(report.Detail)
	if attackerSide == nil || defenderSide == nil {
		t.Fatalf("expected complete standard sides, report=%+v", report)
	}
	assertStandardUnitRow(t, report.ID, *attackerSide, "shuCavalry", 30000, 30000, 13000)
	assertStandardUnitRow(t, report.ID, *attackerSide, "shuInfantry", 30000, 30000, 3000)
	if report.OwnerSide == "attacker" {
		if report.LostUnits["shuCavalry"] != 30000 || report.LostUnits["shuInfantry"] != 30000 || report.SurvivedUnits["shuCavalry"] != 13000 || report.SurvivedUnits["shuInfantry"] != 3000 {
			t.Fatalf("expected attacker legacy original losses and final returned survivors, report=%+v", report)
		}
	} else if report.DefenderLostUnits["shuCavalry"] != 30000 || report.DefenderLostUnits["shuInfantry"] != 30000 {
		t.Fatalf("expected defender view to preserve enemy original losses, report=%+v", report)
	}
	wantExp := attackerExp
	if report.OwnerSide == "defender" {
		wantExp = defenderExp
	}
	if report.GeneralExpGained != wantExp {
		t.Fatalf("expected owner exp %d from real enemy deaths, report=%+v", wantExp, report)
	}
}
