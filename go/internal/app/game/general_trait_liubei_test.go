// 本文件验证刘备仁德天下与仁主守护在真实 PVP 中的叠加返兵、状态和双方战报。
package game

import (
	"testing"

	"hero3/internal/core/general"
)

type liubeiPvpRunResult struct {
	ownerSide string
	unitType  string
	losses    int
	revived   int
	reports   []BattleReport
}

// runLiubeiDualTraitPvp 执行刘备作为进攻或防守主将的真实 PVP，并核对基础战损与持久化状态。
func runLiubeiDualTraitPvp(t *testing.T, ownerSide string, enabled bool) liubeiPvpRunResult {
	t.Helper()
	liubei := GeneralHeroConfig{ID: "liubei", Name: "刘备", Faction: "shu", Enabled: true}
	if enabled {
		liubei.SpecialTrait = GeneralTraitConfig{
			TraitID: "rende", TraitType: general.TraitTypeSpecial, Enabled: true, Scope: "self_army",
			Params: map[string]float64{"effectRate": 0.5, "reviveRate": 0.5, "maxReviveCount": 10000, "triggerChance": 1},
		}
		liubei.BonusTrait = GeneralTraitConfig{
			TraitID: "renzhu_shouhu", TraitType: general.TraitTypeBonus, Enabled: true, Scope: "self_army",
			Params: map[string]float64{"lossReductionRate": 0.1, "maxReturnCount": 10000, "triggerChance": 1},
		}
	}
	setTestFactionsAndGenerals(t, FactionsConfig{
		"wei": {Name: "魏国", Generals: []GeneralInfo{{ID: "caocao", Name: "曹操"}}},
		"shu": {Name: "蜀国", Generals: []GeneralInfo{{ID: "liubei", Name: "刘备"}}},
	}, GeneralsConfig{Enabled: true, Heroes: map[string]GeneralHeroConfig{
		"caocao": {ID: "caocao", Name: "曹操", Faction: "wei", Enabled: true},
		"liubei": liubei,
	}})

	attackerFaction, attackerGeneralID := "wei", "caocao"
	defenderFaction, defenderGeneralID := "shu", "liubei"
	attackerAmount, defenderAmount := 1000, 100
	if ownerSide == "attacker" {
		attackerFaction, attackerGeneralID = "shu", "liubei"
		defenderFaction, defenderGeneralID = "wei", "caocao"
		attackerAmount, defenderAmount = 100, 1000
	}
	svc, repo, attacker, defender := newPvpTestServiceForGenerals(t, attackerFaction, attackerGeneralID, defenderFaction, defenderGeneralID)
	attackerUnit := attackerFaction + "Infantry"
	defenderUnit := defenderFaction + "Infantry"
	attacker.Army = []ArmyUnit{{UnitType: attackerUnit, Amount: attackerAmount}}
	defender.Army = []ArmyUnit{{UnitType: defenderUnit, Amount: defenderAmount}}
	defender.Buildings = nil
	repo.players[attacker.Player.ID] = attacker
	repo.players[defender.Player.ID] = defender

	started, err := svc.StartPvpAttack(PvpAttackRequest{
		PlayerID: attacker.Player.ID, TargetPlayerID: defender.Player.ID, MarchMode: PvpMarchTypeAttack,
		Troops: map[string]int{attackerUnit: attackerAmount}, GeneralIDs: []string{attackerGeneralID},
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
	attackerReports, _, err := repo.ListReports(attacker.Player.ID, 10, 0)
	if err != nil || len(attackerReports) == 0 || attackerReports[0].ID != battle.AttackerReportID {
		t.Fatalf("expected attacker report, reports=%+v err=%v", attackerReports, err)
	}
	defenderReports, _, err := repo.ListReports(defender.Player.ID, 10, 0)
	if err != nil || len(defenderReports) == 0 || defenderReports[0].ID != battle.DefenderReportID {
		t.Fatalf("expected defender report, reports=%+v err=%v", defenderReports, err)
	}

	ownerUnit, ownerAmount, ownerLosses := defenderUnit, defenderAmount, defenderLosses[defenderUnit]
	if ownerSide == "attacker" {
		ownerUnit, ownerAmount, ownerLosses = attackerUnit, attackerAmount, attackerLosses[attackerUnit]
	}
	ownerReport, opponentReport := attackerReports[0], defenderReports[0]
	if ownerSide == "defender" {
		ownerReport, opponentReport = defenderReports[0], attackerReports[0]
	}
	revived := ownerReport.RevivedUnits[ownerUnit]
	if opponentReport.RevivedUnits[ownerUnit] != 0 {
		t.Fatalf("expected legacy returned troops only on owner report, owner=%+v opponent=%+v", ownerReport.RevivedUnits, opponentReport.RevivedUnits)
	}
	expectedRemaining := ownerAmount - ownerLosses + revived
	if ownerSide == "attacker" {
		storedMarch, getErr := repo.GetPvpMarch(started.March.ID)
		if getErr != nil || storedMarch.AttackTroops[ownerUnit] != expectedRemaining {
			t.Fatalf("expected attacker return troops %d, march=%+v err=%v", expectedRemaining, storedMarch, getErr)
		}
	} else {
		storedDefender, getErr := repo.GetState(defender.Player.ID)
		if getErr != nil || armySliceToMap(storedDefender.Army)[ownerUnit] != expectedRemaining {
			t.Fatalf("expected defender remaining troops %d, army=%+v err=%v", expectedRemaining, storedDefender.Army, getErr)
		}
	}
	for _, report := range []BattleReport{attackerReports[0], defenderReports[0]} {
		assertLiubeiStandardSide(t, report, ownerSide, ownerUnit, ownerAmount, ownerLosses, expectedRemaining)
	}
	return liubeiPvpRunResult{ownerSide: ownerSide, unitType: ownerUnit, losses: ownerLosses, revived: revived, reports: []BattleReport{attackerReports[0], defenderReports[0]}}
}

// assertLiubeiStandardSide 核对标准战报中的原始出动、阵亡和包含归队后的最终存活。
func assertLiubeiStandardSide(t *testing.T, report BattleReport, role string, unitType string, before int, losses int, survived int) {
	t.Helper()
	if report.Detail == nil || report.Detail.SecondarySide == nil {
		t.Fatalf("expected complete standard report, got %+v", report.Detail)
	}
	for _, side := range []BattleReportSide{report.Detail.PrimarySide, *report.Detail.SecondarySide} {
		if side.Role != role {
			continue
		}
		for _, unit := range side.Units {
			if unit.UnitType == unitType && unit.AmountBefore == before && unit.Lost == losses && unit.Survived == survived {
				return
			}
		}
		t.Fatalf("expected standard %s side to reconcile %s %d/%d/%d, side=%+v", role, unitType, before, losses, survived, side)
	}
	t.Fatalf("expected standard report %s side, detail=%+v", role, report.Detail)
}

// TestPvpLiubeiDualTraitsReconcileAttackAndDefense 验证刘备双特性在进攻和守城时分别结算并与真实兵力对账。
func TestPvpLiubeiDualTraitsReconcileAttackAndDefense(t *testing.T) {
	for _, ownerSide := range []string{"attacker", "defender"} {
		t.Run(ownerSide, func(t *testing.T) {
			control := runLiubeiDualTraitPvp(t, ownerSide, false)
			active := runLiubeiDualTraitPvp(t, ownerSide, true)
			rende := int(float64(control.losses) * 0.5)
			guard := int(float64(control.losses) * 0.1)
			if control.losses <= 0 || active.losses != control.losses || active.revived != rende+guard {
				t.Fatalf("expected unchanged losses %d and returned %d + %d, got losses=%d returned=%d", control.losses, rende, guard, active.losses, active.revived)
			}
			for _, report := range active.reports {
				assertLiubeiTraitOutcome(t, report, "rende", "revivedUnits", active.unitType, rende, ownerSide)
				assertLiubeiTraitOutcome(t, report, "renzhu_shouhu", "returnedUnits", active.unitType, guard, ownerSide)
			}
		})
	}
}

// TestPvpMainLiubeiReturnsOnlyMainDefenderLosses 验证主守将刘备不会替同兵种援军复活或返兵。
func TestPvpMainLiubeiReturnsOnlyMainDefenderLosses(t *testing.T) {
	base := reinforcementEnemyPvpConfig{
		id: "main_liubei_source_control", attackerTroops: 1000, defenderTroops: 500, helperTroops: 500,
		marchMode:       PvpMarchTypePlunder,
		attackerFaction: "wei", attackerGeneral: "caocao", attackerName: "曹操",
		defenderFaction: "shu", defenderGeneral: "liubei", defenderName: "刘备",
		helperFaction: "shu", helperGeneralID: "guanyu", helperName: "关羽",
	}
	control := runReinforcementEnemyPvp(t, base)
	base.id = "main_liubei_source_active"
	base.defenderSpecial = GeneralTraitConfig{
		TraitID: "rende", TraitType: general.TraitTypeSpecial, Enabled: true, Scope: "self_army",
		Params: map[string]float64{"effectRate": 0.5, "reviveRate": 0.5, "maxReviveCount": 10000, "triggerChance": 1},
	}
	base.defenderBonus = GeneralTraitConfig{
		TraitID: "renzhu_shouhu", TraitType: general.TraitTypeBonus, Enabled: true, Scope: "self_army",
		Params: map[string]float64{"lossReductionRate": 0.1, "maxReturnCount": 10000, "triggerChance": 1},
	}
	active := runReinforcementEnemyPvp(t, base)

	controlMainLosses := pvpTestLossesFromBattle(t, control.battle, "defender")["shuInfantry"]
	activeMainLosses := pvpTestLossesFromBattle(t, active.battle, "defender")["shuInfantry"]
	controlReinforcementLosses := control.battle.Losses["reinforcements"].(map[string]map[string]int)[control.reinforcementID]["shuInfantry"]
	activeReinforcementLosses := active.battle.Losses["reinforcements"].(map[string]map[string]int)[active.reinforcementID]["shuInfantry"]
	if controlMainLosses != 250 || activeMainLosses != 250 || controlReinforcementLosses != 250 || activeReinforcementLosses != 250 {
		t.Fatalf("expected stable raw source losses main/reinforcement 250/250, control=%d/%d active=%d/%d", controlMainLosses, controlReinforcementLosses, activeMainLosses, activeReinforcementLosses)
	}
	if armySliceToMap(active.defenderState.Army)["shuInfantry"] != 400 || active.storedReinforcement.RemainingTroops["shuInfantry"] != 250 {
		t.Fatalf("expected Liu Bei main army to return 150 while reinforcement remains 250, defender=%+v reinforcement=%+v", active.defenderState.Army, active.storedReinforcement)
	}
	if active.defenderReport.RevivedUnits["shuInfantry"] != 150 || len(active.reinforcementReport.RevivedUnits) != 0 {
		t.Fatalf("expected main report returned troops 150 and no reinforcement return, defender=%+v reinforcement=%+v", active.defenderReport.RevivedUnits, active.reinforcementReport.RevivedUnits)
	}
	for _, report := range []BattleReport{active.attackerReport, active.defenderReport} {
		assertLiubeiTraitOutcome(t, report, "rende", "revivedUnits", "shuInfantry", 125, "defender")
		assertLiubeiTraitOutcome(t, report, "renzhu_shouhu", "returnedUnits", "shuInfantry", 25, "defender")
	}
	if len(active.reinforcementReport.TraitTriggered) != 0 || len(active.reinforcementReport.TraitOutcomes) != 0 || standardReportHasTrait(active.reinforcementReport.Detail, "rende") || standardReportHasTrait(active.reinforcementReport.Detail, "renzhu_shouhu") {
		t.Fatalf("expected independent reinforcement report to omit main Liu Bei traits, report=%+v", active.reinforcementReport)
	}
	if active.attackerReport.GeneralExpGained != 500 || active.defenderReport.GeneralExpGained != 500 || active.reinforcementReport.GeneralExpGained != 500 {
		t.Fatalf("expected all generals to use original combat deaths for exp 500, reports=%d/%d/%d", active.attackerReport.GeneralExpGained, active.defenderReport.GeneralExpGained, active.reinforcementReport.GeneralExpGained)
	}
	assertLiubeiStandardSide(t, active.attackerReport, "defender", "shuInfantry", 500, 250, 400)
	assertLiubeiStandardSide(t, active.defenderReport, "defender", "shuInfantry", 500, 250, 400)
	if active.reinforcementReport.LostUnits["shuInfantry"] != 250 || active.reinforcementReport.SurvivedUnits["shuInfantry"] != 250 {
		t.Fatalf("expected independent reinforcement report 500/250/250, report=%+v", active.reinforcementReport)
	}
}

// assertLiubeiTraitOutcome 核对双方新旧战报中的特性归属和逐兵种实际返还值。
func assertLiubeiTraitOutcome(t *testing.T, report BattleReport, traitID string, detailKey string, unitType string, amount int, ownerSide string) {
	t.Helper()
	outcome, ok := report.TraitOutcomes[traitID]
	byUnit, detailOK := outcome.Detail[detailKey].(map[string]int)
	if !ok || !detailOK || byUnit[unitType] != amount || outcome.OwnerSide != ownerSide || outcome.OwnerGeneralID != "liubei" {
		t.Fatalf("expected %s %s=%d for %s, got %+v", traitID, detailKey, amount, unitType, outcome)
	}
	designRateKey := "lossReductionRate"
	designCapKey := "maxReturnCount"
	if traitID == "rende" {
		designRateKey = "effectRate"
		designCapKey = "maxReviveCount"
	}
	expectedRate := 0.1
	if traitID == "rende" {
		expectedRate = 0.5
	}
	if outcome.Detail[designRateKey] != expectedRate || outcome.Detail[designCapKey] != 10000 || outcome.Detail["triggerChance"] != 1.0 {
		t.Fatalf("expected persisted %s design rate %.1f, cap 10000 and chance 1, got %+v", traitID, expectedRate, outcome)
	}
	standardSide := "primary"
	if ownerSide == "defender" {
		standardSide = "secondary"
	}
	for _, trait := range report.Detail.Traits {
		if trait.TraitID == traitID && trait.GeneralID == "liubei" && trait.OwnerSide == standardSide {
			if trait.Detail[designRateKey] != expectedRate || trait.Detail[designCapKey] != 10000 {
				t.Fatalf("expected standard %s design rate %.1f and cap 10000, got %+v", traitID, expectedRate, trait.Detail)
			}
			return
		}
	}
	t.Fatalf("expected %s in standard report for %s, traits=%+v", traitID, standardSide, report.Detail.Traits)
}

// TestPvpBothLiubeiDualTraitsKeepIndependentReturnsAndReports 验证攻守双方刘备的两项返兵能力分别回写状态并保留四条结果。
func TestPvpBothLiubeiDualTraitsKeepIndependentReturnsAndReports(t *testing.T) {
	liubei := GeneralHeroConfig{
		ID: "liubei", Name: "刘备", Faction: "shu", Enabled: true,
		SpecialTrait: GeneralTraitConfig{
			TraitID: "rende", TraitType: general.TraitTypeSpecial, Enabled: true, Scope: "self_army",
			Params: map[string]float64{"effectRate": 0.5, "reviveRate": 0.5, "maxReviveCount": 10000, "triggerChance": 1},
		},
		BonusTrait: GeneralTraitConfig{
			TraitID: "renzhu_shouhu", TraitType: general.TraitTypeBonus, Enabled: true, Scope: "self_army",
			Params: map[string]float64{"lossReductionRate": 0.1, "maxReturnCount": 10000, "triggerChance": 1},
		},
	}
	setTestFactionsAndGenerals(t, FactionsConfig{
		"shu": {Name: "蜀国", Generals: []GeneralInfo{{ID: "liubei", Name: "刘备"}}},
	}, GeneralsConfig{Enabled: true, Heroes: map[string]GeneralHeroConfig{"liubei": liubei}})
	svc, repo, attacker, defender := newPvpTestServiceForGenerals(t, "shu", "liubei", "shu", "liubei")
	attacker.Army = []ArmyUnit{{UnitType: "shuInfantry", Amount: 1000}}
	defender.Army = []ArmyUnit{{UnitType: "shuInfantry", Amount: 1000}}
	defender.Buildings = nil
	repo.players[attacker.Player.ID] = attacker
	repo.players[defender.Player.ID] = defender

	started, err := svc.StartPvpAttack(PvpAttackRequest{
		PlayerID: attacker.Player.ID, TargetPlayerID: defender.Player.ID, MarchMode: PvpMarchTypePlunder,
		Troops: map[string]int{"shuInfantry": 1000}, GeneralIDs: []string{"liubei"},
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
		t.Fatalf("expected equal powers 10000/10000, got %v/%v", battle.Result["attackerPower"], battle.Result["defensePower"])
	}
	attackerLosses := pvpTestLossesFromBattle(t, battle, "attacker")
	defenderLosses := pvpTestLossesFromBattle(t, battle, "defender")
	if attackerLosses["shuInfantry"] != 500 || defenderLosses["shuInfantry"] != 500 {
		t.Fatalf("expected raw plunder losses 500/500 before returns, got attacker=%+v defender=%+v", attackerLosses, defenderLosses)
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
		assertBothLiubeiReport(t, report)
	}

	storedMarch, err := repo.GetPvpMarch(started.March.ID)
	if err != nil || storedMarch.Status != PvpMarchStatusReturning || storedMarch.AttackTroops["shuInfantry"] != 800 {
		t.Fatalf("expected attacker final return troops 800, march=%+v err=%v", storedMarch, err)
	}
	storedAttacker, err := repo.GetState(attacker.Player.ID)
	if err != nil || pvpTestGeneralExp(storedAttacker, "liubei") != 500 {
		t.Fatalf("expected attacker Liu Bei exp 500, state=%+v err=%v", storedAttacker.Generals, err)
	}
	storedDefender, err := repo.GetState(defender.Player.ID)
	if err != nil || armySliceToMap(storedDefender.Army)["shuInfantry"] != 800 || pvpTestGeneralExp(storedDefender, "liubei") != 500 {
		t.Fatalf("expected defender final troops 800 and Liu Bei exp 500, state=%+v err=%v", storedDefender, err)
	}

	retriedBattle, err := svc.ResolvePvpMarch(started.March.ID)
	if err != nil || retriedBattle.ID != battle.ID || retriedBattle.AttackerReportID != battle.AttackerReportID || retriedBattle.DefenderReportID != battle.DefenderReportID {
		t.Fatalf("expected idempotent read to return original battle and reports, original=%+v retried=%+v err=%v", battle, retriedBattle, err)
	}
	retriedMarch, marchErr := repo.GetPvpMarch(started.March.ID)
	retriedAttacker, attackerErr := repo.GetState(attacker.Player.ID)
	retriedDefender, defenderErr := repo.GetState(defender.Player.ID)
	if marchErr != nil || attackerErr != nil || defenderErr != nil || retriedMarch.AttackTroops["shuInfantry"] != 800 || armySliceToMap(retriedDefender.Army)["shuInfantry"] != 800 || pvpTestGeneralExp(retriedAttacker, "liubei") != 500 || pvpTestGeneralExp(retriedDefender, "liubei") != 500 {
		t.Fatalf("expected retry not to duplicate returns or exp, march=%+v attacker=%+v defender=%+v errors=%v/%v/%v", retriedMarch, retriedAttacker.Generals, retriedDefender, marchErr, attackerErr, defenderErr)
	}
	retriedAttackerReports, attackerTotal, attackerReportErr := repo.ListReports(attacker.Player.ID, 10, 0)
	retriedDefenderReports, defenderTotal, defenderReportErr := repo.ListReports(defender.Player.ID, 10, 0)
	if attackerReportErr != nil || defenderReportErr != nil || attackerTotal != 1 || defenderTotal != 1 || len(retriedAttackerReports) != 1 || len(retriedDefenderReports) != 1 || retriedAttackerReports[0].ID != battle.AttackerReportID || retriedDefenderReports[0].ID != battle.DefenderReportID {
		t.Fatalf("expected retry not to duplicate reports, totals=%d/%d reports=%+v/%+v errors=%v/%v", attackerTotal, defenderTotal, retriedAttackerReports, retriedDefenderReports, attackerReportErr, defenderReportErr)
	}
}

// assertBothLiubeiReport 核对双方战报中的四条同名结果、原始阵亡和返兵后最终存活。
func assertBothLiubeiReport(t *testing.T, report BattleReport) {
	t.Helper()
	wantKeys := []string{"rende", "renzhu_shouhu", "rende::defender::liubei", "renzhu_shouhu::defender::liubei"}
	if len(report.TraitTriggered) != len(wantKeys) {
		t.Fatalf("expected four ordered Liu Bei outcomes, report=%s timeline=%+v", report.ID, report.TraitTriggered)
	}
	for index, want := range wantKeys {
		if report.TraitTriggered[index] != want {
			t.Fatalf("expected timeline key %d to be %s, report=%s timeline=%+v", index, want, report.ID, report.TraitTriggered)
		}
	}
	if len(report.TraitOutcomes) != 4 || report.GeneralExpGained != 500 {
		t.Fatalf("expected four legacy outcomes and owner exp 500, report=%+v", report)
	}
	legacyCounts := map[string]int{}
	for _, outcome := range report.TraitOutcomes {
		if outcome.OwnerGeneralID != "liubei" || (outcome.OwnerSide != "attacker" && outcome.OwnerSide != "defender") {
			t.Fatalf("expected explicit Liu Bei owner, report=%s outcome=%+v", report.ID, outcome)
		}
		amount := 0
		switch outcome.TraitID {
		case "rende":
			values, ok := outcome.Detail["revivedUnits"].(map[string]int)
			if !ok || outcome.Detail["effectRate"] != 0.5 || outcome.Detail["maxReviveCount"] != 10000 || outcome.Detail["triggerChance"] != 1.0 {
				t.Fatalf("expected formal Rende detail, report=%s outcome=%+v", report.ID, outcome)
			}
			amount = values["shuInfantry"]
		case "renzhu_shouhu":
			values, ok := outcome.Detail["returnedUnits"].(map[string]int)
			if !ok || outcome.Detail["lossReductionRate"] != 0.1 || outcome.Detail["maxReturnCount"] != 10000 || outcome.Detail["triggerChance"] != 1.0 {
				t.Fatalf("expected formal Ren Zhu detail, report=%s outcome=%+v", report.ID, outcome)
			}
			amount = values["shuInfantry"]
		default:
			t.Fatalf("unexpected bilateral Liu Bei trait, report=%s outcome=%+v", report.ID, outcome)
		}
		legacyCounts[outcome.OwnerSide+":"+outcome.TraitID] = amount
	}
	for _, side := range []string{"attacker", "defender"} {
		if legacyCounts[side+":rende"] != 250 || legacyCounts[side+":renzhu_shouhu"] != 50 {
			t.Fatalf("expected %s Rende 250 and Ren Zhu 50, report=%s counts=%+v", side, report.ID, legacyCounts)
		}
	}
	if report.Detail == nil || report.Detail.SecondarySide == nil || len(report.Detail.Traits) != 4 {
		t.Fatalf("expected standard report with four Liu Bei outcomes, report=%s detail=%+v", report.ID, report.Detail)
	}
	standardCounts := map[string]int{}
	for _, trait := range report.Detail.Traits {
		if trait.GeneralID != "liubei" || (trait.OwnerRole != "attacker" && trait.OwnerRole != "defender") {
			t.Fatalf("expected standard Liu Bei role, report=%s trait=%+v", report.ID, trait)
		}
		key := "revivedUnits"
		if trait.TraitID == "renzhu_shouhu" {
			key = "returnedUnits"
		}
		values, ok := trait.Detail[key].(map[string]int)
		if !ok {
			t.Fatalf("expected standard %s values, report=%s trait=%+v", key, report.ID, trait)
		}
		standardCounts[trait.OwnerRole+":"+trait.TraitID] = values["shuInfantry"]
	}
	for _, side := range []string{"attacker", "defender"} {
		if standardCounts[side+":rende"] != 250 || standardCounts[side+":renzhu_shouhu"] != 50 {
			t.Fatalf("expected standard %s Rende 250 and Ren Zhu 50, report=%s counts=%+v", side, report.ID, standardCounts)
		}
	}
	for _, side := range []BattleReportSide{report.Detail.PrimarySide, *report.Detail.SecondarySide} {
		if len(side.Units) != 1 || side.Units[0].AmountBefore != 1000 || side.Units[0].Lost != 500 || side.Units[0].Survived != 800 {
			t.Fatalf("expected each standard side row 1000/500/800, report=%s side=%+v", report.ID, side)
		}
	}
}
