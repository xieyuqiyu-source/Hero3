// 本文件验证赵云龙胆救援在主将守城和增援真实 PVP 事务中的减损、状态与战报。
package game

import (
	"math"
	"reflect"
	"testing"
	"time"

	"hero3/internal/core/general"
)

type longdanMainRunResult struct {
	ownLosses int
	unitType  string
	reports   []BattleReport
}

// runLongdanMainPvp 执行一场赵云作为主将的真实 PVP，并完成战损、状态和双方基础战报对账。
func runLongdanMainPvp(t *testing.T, ownerSide string, enabled bool) longdanMainRunResult {
	t.Helper()
	zhaoyun := GeneralHeroConfig{ID: "zhaoyun", Name: "赵云", Faction: "shu", Enabled: true}
	if enabled {
		zhaoyun.SpecialTrait = GeneralTraitConfig{
			TraitID: "longdan_jiuyuan", TraitType: general.TraitTypeSpecial, Enabled: true, Scope: "reinforcement_self",
			AllowedSides: []string{"defender", "reinforcement"}, Params: map[string]float64{"lossReductionRate": 0.2, "triggerChance": 1},
		}
	}
	setTestFactionsAndGenerals(t, FactionsConfig{
		"wei": {Name: "魏国", Generals: []GeneralInfo{{ID: "caocao", Name: "曹操"}}},
		"shu": {Name: "蜀国", Generals: []GeneralInfo{{ID: "zhaoyun", Name: "赵云"}}},
	}, GeneralsConfig{Enabled: true, Heroes: map[string]GeneralHeroConfig{
		"caocao":  {ID: "caocao", Name: "曹操", Faction: "wei", Enabled: true},
		"zhaoyun": zhaoyun,
	}})

	attackerFaction, attackerGeneralID := "wei", "caocao"
	defenderFaction, defenderGeneralID := "shu", "zhaoyun"
	attackerAmount, defenderAmount := 1000, 100
	if ownerSide == "attacker" {
		attackerFaction, attackerGeneralID = "shu", "zhaoyun"
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
	if attackerReports[0].LostUnits[attackerUnit] != attackerLosses[attackerUnit] || attackerReports[0].DefenderLostUnits[defenderUnit] != defenderLosses[defenderUnit] ||
		defenderReports[0].LostUnits[defenderUnit] != defenderLosses[defenderUnit] || defenderReports[0].DefenderLostUnits[attackerUnit] != attackerLosses[attackerUnit] {
		t.Fatalf("expected both legacy reports to match battle losses, battle=%+v reports=%+v/%+v", battle.Losses, attackerReports[0], defenderReports[0])
	}
	storedMarch, err := repo.GetPvpMarch(started.March.ID)
	if err != nil || storedMarch.AttackTroops[attackerUnit] != attackerAmount-attackerLosses[attackerUnit] {
		t.Fatalf("expected attacker return troops %d, march=%+v err=%v", attackerAmount-attackerLosses[attackerUnit], storedMarch, err)
	}
	storedDefender, err := repo.GetState(defender.Player.ID)
	if err != nil || armySliceToMap(storedDefender.Army)[defenderUnit] != defenderAmount-defenderLosses[defenderUnit] {
		t.Fatalf("expected defender remaining troops %d, army=%+v err=%v", defenderAmount-defenderLosses[defenderUnit], storedDefender.Army, err)
	}
	for _, report := range []BattleReport{attackerReports[0], defenderReports[0]} {
		assertLongdanStandardSide(t, report, "attacker", attackerUnit, attackerAmount, attackerLosses[attackerUnit])
		assertLongdanStandardSide(t, report, "defender", defenderUnit, defenderAmount, defenderLosses[defenderUnit])
	}

	ownUnit, ownLosses := defenderUnit, defenderLosses[defenderUnit]
	if ownerSide == "attacker" {
		ownUnit, ownLosses = attackerUnit, attackerLosses[attackerUnit]
	}
	return longdanMainRunResult{ownLosses: ownLosses, unitType: ownUnit, reports: []BattleReport{attackerReports[0], defenderReports[0]}}
}

// assertLongdanStandardSide 核对标准战报指定一方的出动、阵亡和存活数值。
func assertLongdanStandardSide(t *testing.T, report BattleReport, role string, unitType string, before int, losses int) {
	t.Helper()
	if report.Detail == nil || report.Detail.SecondarySide == nil {
		t.Fatalf("expected complete standard report, got %+v", report.Detail)
	}
	for _, side := range []BattleReportSide{report.Detail.PrimarySide, *report.Detail.SecondarySide} {
		if side.Role != role {
			continue
		}
		for _, unit := range side.Units {
			if unit.UnitType == unitType && unit.AmountBefore == before && unit.Lost == losses && unit.Survived == before-losses {
				return
			}
		}
		t.Fatalf("expected standard %s side to reconcile %s %d/%d, side=%+v", role, unitType, before, losses, side)
	}
	t.Fatalf("expected standard report %s side, detail=%+v", role, report.Detail)
}

// TestPvpLongdanJiuyuanProtectsOnlyDefendingMainGeneral 验证龙胆救援真实减少守城主将损失，主动进攻时不会误触发。
func TestPvpLongdanJiuyuanProtectsOnlyDefendingMainGeneral(t *testing.T) {
	t.Run("防守主将真实减损", func(t *testing.T) {
		control := runLongdanMainPvp(t, "defender", false)
		active := runLongdanMainPvp(t, "defender", true)
		reduced := int(float64(control.ownLosses) * 0.2)
		if reduced <= 0 || active.ownLosses != control.ownLosses-reduced {
			t.Fatalf("expected defender loss %d - 20%% (%d) = %d, got %d", control.ownLosses, reduced, control.ownLosses-reduced, active.ownLosses)
		}
		for _, report := range active.reports {
			outcome, ok := report.TraitOutcomes["longdan_jiuyuan"]
			byUnit, detailOK := outcome.Detail["reducedLosses"].(map[string]int)
			if !ok || !detailOK || byUnit[active.unitType] != reduced || outcome.OwnerSide != "defender" || outcome.OwnerGeneralID != "zhaoyun" {
				t.Fatalf("expected real defender reduction outcome %d in both reports, got %+v", reduced, outcome)
			}
			if outcome.Detail["lossReductionRate"] != 0.2 || outcome.Detail["triggerChance"] != 1.0 {
				t.Fatalf("expected defender Longdan design rate 0.2 and chance 1, got %+v", outcome.Detail)
			}
			standardMatched := false
			for _, trait := range report.Detail.Traits {
				if trait.TraitID == "longdan_jiuyuan" && trait.OwnerSide == "secondary" && trait.OwnerRole == "defender" {
					if trait.Detail["lossReductionRate"] != 0.2 {
						t.Fatalf("expected standard defender Longdan design rate 0.2, got %+v", trait.Detail)
					}
					standardMatched = true
				}
			}
			if !standardMatched {
				t.Fatalf("expected defender Longdan in standard report, traits=%+v", report.Detail.Traits)
			}
		}
	})

	t.Run("主动进攻不触发", func(t *testing.T) {
		result := runLongdanMainPvp(t, "attacker", true)
		for _, report := range result.reports {
			if _, ok := report.TraitOutcomes["longdan_jiuyuan"]; ok {
				t.Fatalf("expected attacking Zhao Yun not to trigger Longdan, outcomes=%+v", report.TraitOutcomes)
			}
			for _, trait := range report.Detail.Traits {
				if trait.TraitID == "longdan_jiuyuan" {
					t.Fatalf("expected no fake Longdan in standard report, traits=%+v", report.Detail.Traits)
				}
			}
		}
	})
}

// TestPvpMainLongdanDoesNotReduceSameUnitReinforcementLosses 验证主城赵云只保护主守军，不替同兵种援军减损。
func TestPvpMainLongdanDoesNotReduceSameUnitReinforcementLosses(t *testing.T) {
	base := reinforcementEnemyPvpConfig{
		id: "main_longdan_source_control", attackerTroops: 1000, defenderTroops: 500, marchMode: PvpMarchTypePlunder,
		helperFaction: "shu", helperGeneralID: "liubei", helperName: "刘备", helperTroops: 500,
		attackerFaction: "wei", attackerGeneral: "caocao", attackerName: "曹操",
		defenderFaction: "shu", defenderGeneral: "zhaoyun", defenderName: "赵云",
		defenderSpecial: GeneralTraitConfig{
			TraitID: "longdan_jiuyuan", TraitType: general.TraitTypeSpecial, Enabled: false, Scope: "reinforcement_self",
			AllowedSides: []string{"defender", "reinforcement"}, Params: map[string]float64{"lossReductionRate": 0.2, "triggerChance": 1},
		},
	}
	control := runReinforcementEnemyPvp(t, base)
	base.id = "main_longdan_source_active"
	base.defenderSpecial.Enabled = true
	active := runReinforcementEnemyPvp(t, base)
	if control.battle.Result["attackerPower"] != float64(10000) || control.battle.Result["defensePower"] != float64(10000) ||
		active.battle.Result["attackerPower"] != float64(10000) || active.battle.Result["defensePower"] != float64(10000) {
		t.Fatalf("expected equal 10000/10000 powers, control=%+v active=%+v", control.battle.Result, active.battle.Result)
	}
	if control.defenderReport.LostUnits["shuInfantry"] != 250 || control.storedReinforcement.Losses["shuInfantry"] != 250 || control.defendingLosses != 500 {
		t.Fatalf("expected control main/reinforcement losses 250/250, main=%+v reinforcement=%+v total=%d", control.defenderReport.LostUnits, control.storedReinforcement.Losses, control.defendingLosses)
	}
	if active.defenderReport.LostUnits["shuInfantry"] != 200 || active.storedReinforcement.Losses["shuInfantry"] != 250 || active.defendingLosses != 450 {
		t.Fatalf("expected main Longdan to reduce only main loss 250 -> 200 while reinforcement remains 250, main=%+v reinforcement=%+v total=%d", active.defenderReport.LostUnits, active.storedReinforcement.Losses, active.defendingLosses)
	}
	for _, report := range []BattleReport{active.attackerReport, active.defenderReport} {
		outcome, ok := report.TraitOutcomes["longdan_jiuyuan"]
		reduced, reducedOK := outcome.Detail["reducedLosses"].(map[string]int)
		if !ok || !reducedOK || reduced["shuInfantry"] != 50 || outcome.OwnerSide != "defender" || outcome.OwnerGeneralID != "zhaoyun" {
			t.Fatalf("expected main Zhao Yun reduction 50 in both main reports, report=%s outcome=%+v", report.ID, outcome)
		}
		if report.PvpReinforcementLosses[active.reinforcementID]["shuInfantry"] != 250 {
			t.Fatalf("expected every main report to retain reinforcement loss 250, report=%s losses=%+v", report.ID, report.PvpReinforcementLosses)
		}
	}
	if len(active.reinforcementReport.TraitOutcomes) != 0 || active.reinforcementReport.LostUnits["shuInfantry"] != 250 || active.reinforcementReport.SurvivedUnits["shuInfantry"] != 250 {
		t.Fatalf("expected independent reinforcement report 500/250/250 without main Longdan, report=%+v", active.reinforcementReport)
	}
	if active.attackerReport.GeneralExpGained != 450 || active.defenderReport.GeneralExpGained != 500 || active.reinforcementReport.GeneralExpGained != 500 {
		t.Fatalf("expected attacker/main/helper exp 450/500/500 from real losses, attacker=%d defender=%d helper=%d", active.attackerReport.GeneralExpGained, active.defenderReport.GeneralExpGained, active.reinforcementReport.GeneralExpGained)
	}
	if armySliceToMap(active.defenderState.Army)["shuInfantry"] != 300 || active.storedReinforcement.RemainingTroops["shuInfantry"] != 250 {
		t.Fatalf("expected authoritative main/reinforcement survivors 300/250, defender=%+v reinforcement=%+v", active.defenderState.Army, active.storedReinforcement)
	}
}

// TestPvpMainLongdanLegalMissKeepsMainAndReinforcementFullLosses 验证主守将龙胆合法未命中时，主守军和同兵种援军都承担完整损失。
func TestPvpMainLongdanLegalMissKeepsMainAndReinforcementFullLosses(t *testing.T) {
	result := runReinforcementEnemyPvp(t, reinforcementEnemyPvpConfig{
		id: "main_longdan_probability_miss", attackerTroops: 1000, defenderTroops: 500, marchMode: PvpMarchTypePlunder,
		helperFaction: "shu", helperGeneralID: "liubei", helperName: "刘备", helperTroops: 500,
		attackerFaction: "wei", attackerGeneral: "caocao", attackerName: "曹操",
		defenderFaction: "shu", defenderGeneral: "zhaoyun", defenderName: "赵云",
		defenderSpecial: GeneralTraitConfig{
			TraitID: "longdan_jiuyuan", TraitType: general.TraitTypeSpecial, Enabled: true, Scope: "reinforcement_self",
			AllowedSides: []string{"defender", "reinforcement"}, Params: map[string]float64{"lossReductionRate": 0.2, "triggerChance": 0},
		},
	})
	if result.battle.Result["attackerPower"] != float64(10000) || result.battle.Result["defensePower"] != float64(10000) || result.battle.Result["winner"] != "draw" {
		t.Fatalf("expected equal 10000/10000 powers and draw, result=%+v", result.battle.Result)
	}
	if result.attackerLosses != 500 || result.defenderReport.LostUnits["shuInfantry"] != 250 || result.storedReinforcement.Losses["shuInfantry"] != 250 || result.defendingLosses != 500 {
		t.Fatalf("expected full attacker/main/reinforcement losses 500/250/250, attacker=%d main=%+v reinforcement=%+v total=%d", result.attackerLosses, result.defenderReport.LostUnits, result.storedReinforcement.Losses, result.defendingLosses)
	}
	if armySliceToMap(result.defenderState.Army)["shuInfantry"] != 250 || result.storedReinforcement.RemainingTroops["shuInfantry"] != 250 {
		t.Fatalf("expected authoritative main/reinforcement survivors 250/250, defender=%+v reinforcement=%+v", result.defenderState.Army, result.storedReinforcement)
	}
	for _, report := range []BattleReport{result.attackerReport, result.defenderReport, result.reinforcementReport} {
		if len(report.TraitTriggered) != 0 || len(report.TraitOutcomes) != 0 || report.Detail == nil || len(report.Detail.Traits) != 0 {
			t.Fatalf("expected missed main Longdan absent from every timeline, report=%+v", report)
		}
		if report.PvpReinforcementLosses[result.reinforcementID]["shuInfantry"] != 250 {
			t.Fatalf("expected every report to retain reinforcement loss 250, report=%s losses=%+v", report.ID, report.PvpReinforcementLosses)
		}
	}
	if len(result.attackerReport.PvpDefenderGenerals) != 1 || len(result.defenderReport.PvpDefenderGenerals) != 1 ||
		result.attackerReport.PvpDefenderGenerals[0].ID != "zhaoyun" || result.defenderReport.PvpDefenderGenerals[0].ID != "zhaoyun" ||
		!pvpSnapshotHasTrait(result.attackerReport.PvpDefenderGenerals[0], "longdan_jiuyuan") || !pvpSnapshotHasTrait(result.defenderReport.PvpDefenderGenerals[0], "longdan_jiuyuan") {
		t.Fatalf("expected both main reports to retain Zhao Yun Longdan ownership snapshot, attacker=%+v defender=%+v", result.attackerReport.PvpDefenderGenerals, result.defenderReport.PvpDefenderGenerals)
	}
	if result.attackerReport.GeneralExpGained != 500 || result.defenderReport.GeneralExpGained != 500 || result.reinforcementReport.GeneralExpGained != 500 {
		t.Fatalf("expected attacker/main/helper exp 500/500/500 from full losses, attacker=%d defender=%d helper=%d", result.attackerReport.GeneralExpGained, result.defenderReport.GeneralExpGained, result.reinforcementReport.GeneralExpGained)
	}
}

type longdanReinforcementRunResult struct {
	losses        int
	sent          Reinforcement
	battle        PvpBattle
	reinforcement Reinforcement
	reports       []BattleReport
	helperState   GameState
}

// runLongdanReinforcementPvp 执行赵云援军从创建、到达驻防到真实参战的完整 PVP 事务。
func runLongdanReinforcementPvp(t *testing.T, longdanEnabled bool, longdanChance float64, qijinEnabled bool) longdanReinforcementRunResult {
	t.Helper()
	setTestFactionsAndGenerals(t, FactionsConfig{
		"wei": {Name: "魏国", Generals: []GeneralInfo{{ID: "caocao", Name: "曹操"}}},
		"shu": {Name: "蜀国", Generals: []GeneralInfo{{ID: "liubei", Name: "刘备"}, {ID: "zhaoyun", Name: "赵云"}}},
	}, GeneralsConfig{Enabled: true, Heroes: map[string]GeneralHeroConfig{
		"caocao": {ID: "caocao", Name: "曹操", Faction: "wei", Enabled: true},
		"liubei": {ID: "liubei", Name: "刘备", Faction: "shu", Enabled: true},
		"zhaoyun": {
			ID: "zhaoyun", Name: "赵云", Faction: "shu", Enabled: true,
			SpecialTrait: GeneralTraitConfig{
				TraitID: "longdan_jiuyuan", TraitType: general.TraitTypeSpecial, Enabled: longdanEnabled, Scope: "reinforcement_self",
				AllowedSides: []string{"defender", "reinforcement"}, Params: map[string]float64{"lossReductionRate": 0.2, "triggerChance": longdanChance},
			},
			BonusTrait: GeneralTraitConfig{
				TraitID: "qijin_qichu", TraitType: general.TraitTypeBonus, Enabled: qijinEnabled, Scope: "self_army",
				Params: map[string]float64{"speedBonusRate": 1, "minMarchSeconds": 60, "triggerChance": 1},
			},
		},
	}})
	svc, repo, attacker, defender := newPvpTestServiceForGenerals(t, "wei", "caocao", "shu", "liubei")
	now := time.Now().UTC()
	helperAccount := Account{ID: "account_longdan_helper", Username: "longdan_helper", PasswordHash: "hash", CreatedAt: now}
	if err := repo.CreateAccount(helperAccount); err != nil {
		t.Fatalf("CreateAccount helper failed: %v", err)
	}
	helper := newPlayerState("player_longdan_helper", "常山援军", "shu", "zhaoyun", now)
	helper.Army = []ArmyUnit{{UnitType: "shuInfantry", Amount: 100}}
	if err := repo.CreatePlayer(helperAccount.ID, helper, now); err != nil {
		t.Fatalf("CreatePlayer helper failed: %v", err)
	}
	attacker.Army = []ArmyUnit{{UnitType: "weiInfantry", Amount: 110}}
	defender.Army = []ArmyUnit{{UnitType: "shuInfantry", Amount: 1}}
	defender.Buildings = nil
	repo.players[attacker.Player.ID] = attacker
	repo.players[defender.Player.ID] = defender
	for _, position := range []struct {
		playerID string
		x        int
		y        int
	}{
		{playerID: helper.Player.ID, x: 10, y: 10},
		{playerID: defender.Player.ID, x: 20, y: 10},
		{playerID: attacker.Player.ID, x: 30, y: 10},
	} {
		if _, err := repo.AssignWorldPosition(position.playerID, defaultWorldID, position.x, position.y, "test"); err != nil {
			t.Fatalf("AssignWorldPosition %s failed: %v", position.playerID, err)
		}
	}
	sent, err := svc.SendReinforcement(SendReinforcementRequest{
		FromPlayerID: helper.Player.ID, TargetPlayerID: defender.Player.ID,
		Troops: map[string]int{"shuInfantry": 100}, GeneralIDs: []string{"zhaoyun"},
	})
	if err != nil {
		t.Fatalf("SendReinforcement failed: %v", err)
	}
	assertNoPreCombatMarchReport(t, repo, helper.Player.ID)
	assertMarchDurationTimestamps(t, sent.Reinforcement.SentAt, sent.Reinforcement.ExpectedArriveAt, sent.Reinforcement.MarchSeconds)
	forceReinforcementDue(t, repo, sent.Reinforcement.ID, true)
	if _, err := svc.MarkReinforcementArrived(sent.Reinforcement.ID); err != nil {
		t.Fatalf("MarkReinforcementArrived failed: %v", err)
	}

	started, err := svc.StartPvpAttack(PvpAttackRequest{
		PlayerID: attacker.Player.ID, TargetPlayerID: defender.Player.ID, MarchMode: PvpMarchTypeAttack,
		Troops: map[string]int{"weiInfantry": 110}, GeneralIDs: []string{"caocao"},
	})
	if err != nil {
		t.Fatalf("StartPvpAttack failed: %v", err)
	}
	forcePvpMarchDue(t, repo, started.March.ID)
	battle, err := svc.ResolvePvpMarch(started.March.ID)
	if err != nil {
		t.Fatalf("ResolvePvpMarch failed: %v", err)
	}
	attackerReports, _, err := repo.ListReports(attacker.Player.ID, 10, 0)
	if err != nil || len(attackerReports) == 0 || attackerReports[0].ID != battle.AttackerReportID {
		t.Fatalf("expected attacker report, reports=%+v err=%v", attackerReports, err)
	}
	defenderReports, _, err := repo.ListReports(defender.Player.ID, 10, 0)
	if err != nil || len(defenderReports) == 0 || defenderReports[0].ID != battle.DefenderReportID {
		t.Fatalf("expected defender report, reports=%+v err=%v", defenderReports, err)
	}
	helperReports, total, err := repo.ListReportsByQuery(BattleReportQuery{PlayerID: helper.Player.ID, ViewType: ReportViewReinforcement, Page: 1, PageSize: 10})
	if err != nil || total != 1 || len(helperReports) != 1 {
		t.Fatalf("expected one reinforcement report, reports=%+v total=%d err=%v", helperReports, total, err)
	}
	losses := defenderReports[0].PvpReinforcementLosses[sent.Reinforcement.ID]["shuInfantry"]
	if losses <= 0 || attackerReports[0].PvpReinforcementLosses[sent.Reinforcement.ID]["shuInfantry"] != losses || helperReports[0].LostUnits["shuInfantry"] != losses {
		t.Fatalf("expected all three reports to use reinforcement loss %d, attacker=%+v defender=%+v helper=%+v", losses, attackerReports[0].PvpReinforcementLosses, defenderReports[0].PvpReinforcementLosses, helperReports[0].LostUnits)
	}
	stored, err := repo.GetReinforcement(sent.Reinforcement.ID)
	if err != nil || stored.RemainingTroops["shuInfantry"] != 100-losses {
		t.Fatalf("expected real reinforcement remaining %d, record=%+v err=%v", 100-losses, stored, err)
	}
	helperState, err := repo.GetState(helper.Player.ID)
	if err != nil {
		t.Fatalf("GetState helper failed: %v", err)
	}
	return longdanReinforcementRunResult{
		losses: losses, sent: sent.Reinforcement, battle: battle, reinforcement: stored,
		reports: []BattleReport{attackerReports[0], defenderReports[0], helperReports[0]}, helperState: helperState,
	}
}

// TestPvpLongdanJiuyuanReinforcementMatchesRealStateAndThreeReports 验证赵云援军减损进入真实驻防兵力、双方主战报和援军独立战报。
func TestPvpLongdanJiuyuanReinforcementMatchesRealStateAndThreeReports(t *testing.T) {
	control := runLongdanReinforcementPvp(t, false, 0, false)
	active := runLongdanReinforcementPvp(t, true, 1, true)
	wantMarchSeconds := applyExpectedMarchRates(control.sent.MarchSeconds, []float64{1}, 60)
	wantSpeed := control.sent.SpeedMultiplier * float64(control.sent.MarchSeconds) / float64(wantMarchSeconds)
	if active.sent.MarchSeconds != wantMarchSeconds || active.sent.ReturnSeconds != wantMarchSeconds || math.Abs(active.sent.SpeedMultiplier-wantSpeed) > 1e-9 {
		t.Fatalf("expected Qijin duration %d and speed %.6f, control=%+v active=%+v", wantMarchSeconds, wantSpeed, control.sent, active.sent)
	}
	attackPower, attackOK := active.battle.Result["attackerPower"].(float64)
	defensePower, defenseOK := active.battle.Result["defensePower"].(float64)
	if !attackOK || !defenseOK || attackPower != 1100 || defensePower != 1010 || active.battle.Result["winner"] != "attacker" {
		t.Fatalf("expected unchanged core powers 1100/1010 and attacker victory, result=%+v", active.battle.Result)
	}
	reduced := int(float64(control.losses) * 0.2)
	if control.losses != 100 || reduced != 20 || active.losses != 80 {
		t.Fatalf("expected reinforcement loss %d - 20%% (%d) = %d, got %d", control.losses, reduced, control.losses-reduced, active.losses)
	}
	if active.reinforcement.RemainingTroops["shuInfantry"] != 20 || active.reinforcement.Losses["shuInfantry"] != 80 || active.reinforcement.Status != ReinforcementStatusStationed {
		t.Fatalf("expected persisted reinforcement to match reduced losses, record=%+v", active.reinforcement)
	}
	if control.reports[0].GeneralExpGained != 101 || active.reports[0].GeneralExpGained != 81 {
		t.Fatalf("expected attacker exp to follow main plus reinforcement losses 101 -> 81, control=%d active=%d", control.reports[0].GeneralExpGained, active.reports[0].GeneralExpGained)
	}
	for _, report := range active.reports {
		if len(report.TraitTriggered) != 1 || report.TraitTriggered[0] != "longdan_jiuyuan" || len(report.TraitOutcomes) != 1 {
			t.Fatalf("expected only Longdan in battle timeline, report=%s timeline=%v outcomes=%+v", report.ID, report.TraitTriggered, report.TraitOutcomes)
		}
		if _, exists := report.TraitOutcomes["qijin_qichu"]; exists || standardReportHasTrait(report.Detail, "qijin_qichu") {
			t.Fatalf("expected march process trait absent from battle timeline, report=%s outcomes=%+v detail=%+v", report.ID, report.TraitOutcomes, report.Detail)
		}
		outcome, ok := report.TraitOutcomes["longdan_jiuyuan"]
		byUnit, detailOK := outcome.Detail["reducedLosses"].(map[string]int)
		if !ok || !detailOK || byUnit["shuInfantry"] != reduced || outcome.OwnerSide != "reinforcement" || outcome.OwnerGeneralID != "zhaoyun" {
			t.Fatalf("expected Zhao Yun reinforcement reduction %d in every report, got %+v", reduced, outcome)
		}
		if outcome.Detail["lossReductionRate"] != 0.2 || outcome.Detail["triggerChance"] != 1.0 {
			t.Fatalf("expected reinforcement Longdan design rate 0.2 and chance 1, got %+v", outcome.Detail)
		}
		standardMatched := false
		for _, trait := range report.Detail.Traits {
			if trait.TraitID == "longdan_jiuyuan" && trait.GeneralID == "zhaoyun" && trait.OwnerRole == "reinforcement" {
				standardReduced, standardReducedOK := trait.Detail["reducedLosses"].(map[string]int)
				if trait.Detail["lossReductionRate"] != 0.2 || !standardReducedOK || standardReduced["shuInfantry"] != 20 {
					t.Fatalf("expected standard reinforcement Longdan design rate 0.2, got %+v", trait.Detail)
				}
				standardMatched = true
			}
		}
		if !standardMatched {
			t.Fatalf("expected Longdan in standard report, traits=%+v", report.Detail.Traits)
		}
		if len(report.PvpReinforcements) != 1 || len(report.PvpReinforcements[0].Generals) != 1 || report.PvpReinforcements[0].Generals[0].ID != "zhaoyun" {
			t.Fatalf("expected Zhao Yun reinforcement snapshot, report=%s reinforcements=%+v", report.ID, report.PvpReinforcements)
		}
		ownedTraits := map[string]bool{}
		for _, trait := range report.PvpReinforcements[0].Generals[0].Traits {
			ownedTraits[trait.TraitID] = true
		}
		if !ownedTraits["longdan_jiuyuan"] || !ownedTraits["qijin_qichu"] {
			t.Fatalf("expected snapshot to retain both owned traits, report=%s traits=%+v", report.ID, report.PvpReinforcements[0].Generals[0].Traits)
		}
		if report.PvpReinforcementLosses[active.sent.ID]["shuInfantry"] != 80 {
			t.Fatalf("expected three reports to use final reinforcement loss 80, report=%s losses=%+v", report.ID, report.PvpReinforcementLosses)
		}
	}
	helperReport := active.reports[2]
	if helperReport.DispatchedUnits["shuInfantry"] != 100 || helperReport.LostUnits["shuInfantry"] != 80 || helperReport.SurvivedUnits["shuInfantry"] != 20 {
		t.Fatalf("expected helper legacy row 100/80/20 after reduction, report=%+v", helperReport)
	}
	if helperReport.Detail == nil || helperReport.Detail.SecondarySide == nil || helperReport.Detail.PrimarySide.Role != "attacker" || helperReport.Detail.SecondarySide.Role != "defender" {
		t.Fatalf("expected helper standard detail to preserve complete attacker and defender snapshots, detail=%+v", helperReport.Detail)
	}
	if helperReport.GeneralExpGained != 97 || helperReport.PvpReinforcements[0].GeneralExpGained != 97 || pvpTestGeneralExp(active.helperState, "zhaoyun") != 97 {
		t.Fatalf("expected Zhao Yun exp 97 in report snapshot and owner state, report=%+v state=%+v", helperReport, active.helperState.Generals)
	}
}

// TestPvpLongdanJiuyuanLegalMissKeepsQijinMarchAndFullLosses 验证龙胆在合法增援场景未命中时不减损，七进七出仍独立缩短行军。
func TestPvpLongdanJiuyuanLegalMissKeepsQijinMarchAndFullLosses(t *testing.T) {
	control := runLongdanReinforcementPvp(t, false, 0, false)
	missed := runLongdanReinforcementPvp(t, true, 0, true)
	wantMarchSeconds := applyExpectedMarchRates(control.sent.MarchSeconds, []float64{1}, 60)
	wantSpeed := control.sent.SpeedMultiplier * float64(control.sent.MarchSeconds) / float64(wantMarchSeconds)
	if missed.sent.MarchSeconds != wantMarchSeconds || missed.sent.ReturnSeconds != wantMarchSeconds || math.Abs(missed.sent.SpeedMultiplier-wantSpeed) > 1e-9 {
		t.Fatalf("expected Qijin to keep duration %d and speed %.6f when Longdan misses, control=%+v missed=%+v", wantMarchSeconds, wantSpeed, control.sent, missed.sent)
	}
	attackPower, attackOK := missed.battle.Result["attackerPower"].(float64)
	defensePower, defenseOK := missed.battle.Result["defensePower"].(float64)
	if !attackOK || !defenseOK || attackPower != 1100 || defensePower != 1010 || missed.battle.Result["winner"] != "attacker" {
		t.Fatalf("expected unchanged core powers 1100/1010 and attacker victory, result=%+v", missed.battle.Result)
	}
	if control.losses != 100 || missed.losses != 100 || missed.reinforcement.Losses["shuInfantry"] != 100 || missed.reinforcement.RemainingTroops["shuInfantry"] != 0 {
		t.Fatalf("expected missed Longdan to keep full 100 reinforcement losses, control=%d missed=%+v", control.losses, missed.reinforcement)
	}
	for _, report := range missed.reports {
		if len(report.TraitTriggered) != 0 || len(report.TraitOutcomes) != 0 || report.Detail == nil || len(report.Detail.Traits) != 0 {
			t.Fatalf("expected missed Longdan and process-only Qijin absent from battle timelines, report=%+v", report)
		}
		if len(report.PvpReinforcements) != 1 || len(report.PvpReinforcements[0].Generals) != 1 || report.PvpReinforcements[0].Generals[0].ID != "zhaoyun" {
			t.Fatalf("expected Zhao Yun reinforcement snapshot, report=%s reinforcements=%+v", report.ID, report.PvpReinforcements)
		}
		ownedTraits := map[string]bool{}
		for _, trait := range report.PvpReinforcements[0].Generals[0].Traits {
			ownedTraits[trait.TraitID] = true
		}
		if !ownedTraits["longdan_jiuyuan"] || !ownedTraits["qijin_qichu"] {
			t.Fatalf("expected snapshot to retain both owned traits after Longdan miss, report=%s traits=%+v", report.ID, report.PvpReinforcements[0].Generals[0].Traits)
		}
		if report.PvpReinforcementLosses[missed.sent.ID]["shuInfantry"] != 100 {
			t.Fatalf("expected every report to retain full reinforcement loss 100, report=%s losses=%+v", report.ID, report.PvpReinforcementLosses)
		}
	}
	helperReport := missed.reports[2]
	if helperReport.DispatchedUnits["shuInfantry"] != 100 || helperReport.LostUnits["shuInfantry"] != 100 || helperReport.SurvivedUnits["shuInfantry"] != 0 {
		t.Fatalf("expected helper legacy row 100/100/0 without reduction, report=%+v", helperReport)
	}
	if helperReport.Detail == nil || helperReport.Detail.SecondarySide == nil || helperReport.Detail.PrimarySide.Role != "attacker" || helperReport.Detail.SecondarySide.Role != "defender" {
		t.Fatalf("expected helper standard detail to preserve complete attacker and defender snapshots after miss, detail=%+v", helperReport.Detail)
	}
	if helperReport.GeneralExpGained != 97 || helperReport.PvpReinforcements[0].GeneralExpGained != 97 || pvpTestGeneralExp(missed.helperState, "zhaoyun") != 97 {
		t.Fatalf("expected Zhao Yun exp 97 to remain based on enemy losses, report=%+v state=%+v", helperReport, missed.helperState.Generals)
	}
}

// TestPvpTwoPlayersSameZhaoyunKeepIndependentReinforcementOutcomes 验证两名玩家的同将同特性援军不会在真实战斗和战报中互相覆盖。
func TestPvpTwoPlayersSameZhaoyunKeepIndependentReinforcementOutcomes(t *testing.T) {
	setTestFactionsAndGenerals(t, FactionsConfig{
		"wei": {Name: "魏国", Generals: []GeneralInfo{{ID: "caocao", Name: "曹操"}}},
		"shu": {Name: "蜀国", Generals: []GeneralInfo{{ID: "liubei", Name: "刘备"}, {ID: "zhaoyun", Name: "赵云"}}},
	}, GeneralsConfig{Enabled: true, Heroes: map[string]GeneralHeroConfig{
		"caocao": {ID: "caocao", Name: "曹操", Faction: "wei", Enabled: true},
		"liubei": {ID: "liubei", Name: "刘备", Faction: "shu", Enabled: true},
		"zhaoyun": {
			ID: "zhaoyun", Name: "赵云", Faction: "shu", Enabled: true,
			SpecialTrait: GeneralTraitConfig{
				TraitID: "longdan_jiuyuan", TraitType: general.TraitTypeSpecial, Enabled: true, Scope: "reinforcement_self",
				AllowedSides: []string{"defender", "reinforcement"}, Params: map[string]float64{"lossReductionRate": 0.2, "triggerChance": 1},
			},
		},
	}})

	svc, repo, attacker, defender := newPvpTestServiceForGenerals(t, "wei", "caocao", "shu", "liubei")
	attacker.Army = []ArmyUnit{{UnitType: "weiInfantry", Amount: 400}}
	defender.Army = []ArmyUnit{{UnitType: "shuInfantry", Amount: 1}}
	defender.Buildings = nil
	repo.players[attacker.Player.ID] = attacker
	repo.players[defender.Player.ID] = defender

	type helperFixture struct {
		playerID      string
		name          string
		amount        int
		x             int
		reinforcement Reinforcement
	}
	helpers := []helperFixture{
		{playerID: "player_longdan_same_a", name: "常山甲", amount: 100, x: 10},
		{playerID: "player_longdan_same_b", name: "常山乙", amount: 200, x: 11},
	}
	now := time.Now().UTC()
	for index := range helpers {
		helper := &helpers[index]
		account := Account{ID: "account_" + helper.playerID, Username: helper.playerID, PasswordHash: "hash", CreatedAt: now}
		if err := repo.CreateAccount(account); err != nil {
			t.Fatalf("CreateAccount %s failed: %v", helper.playerID, err)
		}
		state := newPlayerState(helper.playerID, helper.name, "shu", "zhaoyun", now)
		state.Army = []ArmyUnit{{UnitType: "shuInfantry", Amount: helper.amount}}
		if err := repo.CreatePlayer(account.ID, state, now); err != nil {
			t.Fatalf("CreatePlayer %s failed: %v", helper.playerID, err)
		}
	}
	for _, position := range []struct {
		playerID string
		x        int
	}{
		{playerID: helpers[0].playerID, x: helpers[0].x},
		{playerID: helpers[1].playerID, x: helpers[1].x},
		{playerID: defender.Player.ID, x: 20},
		{playerID: attacker.Player.ID, x: 30},
	} {
		if _, err := repo.AssignWorldPosition(position.playerID, defaultWorldID, position.x, 10, "test"); err != nil {
			t.Fatalf("AssignWorldPosition %s failed: %v", position.playerID, err)
		}
	}
	for index := range helpers {
		helper := &helpers[index]
		sent, err := svc.SendReinforcement(SendReinforcementRequest{
			FromPlayerID: helper.playerID, TargetPlayerID: defender.Player.ID,
			Troops: map[string]int{"shuInfantry": helper.amount}, GeneralIDs: []string{"zhaoyun"},
		})
		if err != nil {
			t.Fatalf("SendReinforcement %s failed: %v", helper.playerID, err)
		}
		helper.reinforcement = sent.Reinforcement
		forceReinforcementDue(t, repo, sent.Reinforcement.ID, true)
		if _, err := svc.MarkReinforcementArrived(sent.Reinforcement.ID); err != nil {
			t.Fatalf("MarkReinforcementArrived %s failed: %v", helper.playerID, err)
		}
	}

	started, err := svc.StartPvpAttack(PvpAttackRequest{
		PlayerID: attacker.Player.ID, TargetPlayerID: defender.Player.ID, MarchMode: PvpMarchTypeAttack,
		Troops: map[string]int{"weiInfantry": 400}, GeneralIDs: []string{"caocao"},
	})
	if err != nil {
		t.Fatalf("StartPvpAttack failed: %v", err)
	}
	forcePvpMarchDue(t, repo, started.March.ID)
	battle, err := svc.ResolvePvpMarch(started.March.ID)
	if err != nil {
		t.Fatalf("ResolvePvpMarch failed: %v", err)
	}
	attackerReport, err := repo.GetReportForPlayer(attacker.Player.ID, battle.AttackerReportID)
	if err != nil {
		t.Fatalf("GetReportForPlayer attacker failed: %v", err)
	}
	defenderReport, err := repo.GetReportForPlayer(defender.Player.ID, battle.DefenderReportID)
	if err != nil {
		t.Fatalf("GetReportForPlayer defender failed: %v", err)
	}
	if battle.Result["attackerPower"] != float64(4000) || battle.Result["defensePower"] != float64(3010) || battle.Result["winner"] != "attacker" {
		t.Fatalf("expected fixed 4000/3010 attacker victory, result=%+v", battle.Result)
	}
	wantReduced := map[string]int{
		helpers[0].playerID: helpers[0].amount / 5,
		helpers[1].playerID: helpers[1].amount / 5,
	}

	for _, report := range []BattleReport{attackerReport, defenderReport} {
		seenPlayers := map[string]int{}
		for _, outcome := range report.TraitOutcomes {
			if outcome.TraitID != "longdan_jiuyuan" {
				continue
			}
			reduced, ok := outcome.Detail["reducedLosses"].(map[string]int)
			if !ok || reduced["shuInfantry"] <= 0 || outcome.OwnerSide != "reinforcement" || outcome.OwnerGeneralID != "zhaoyun" {
				t.Fatalf("expected player-owned Longdan outcome in report %s, got %+v", report.ID, outcome)
			}
			seenPlayers[outcome.OwnerPlayerID] = reduced["shuInfantry"]
		}
		if !reflect.DeepEqual(seenPlayers, wantReduced) {
			t.Fatalf("expected two independent same-general outcomes in report %s, got %+v", report.ID, report.TraitOutcomes)
		}
		standardPlayers := map[string]int{}
		for _, trait := range report.Detail.Traits {
			if trait.TraitID != "longdan_jiuyuan" {
				continue
			}
			reduced, ok := trait.Detail["reducedLosses"].(map[string]int)
			if !ok || trait.OwnerRole != "reinforcement" || trait.GeneralID != "zhaoyun" {
				t.Fatalf("expected standard same-general reinforcement trait in report %s, got %+v", report.ID, trait)
			}
			standardPlayers[trait.OwnerPlayerID] = reduced["shuInfantry"]
		}
		if !reflect.DeepEqual(standardPlayers, seenPlayers) {
			t.Fatalf("expected standard ownership and values to match legacy outcomes in report %s, legacy=%+v standard=%+v", report.ID, seenPlayers, standardPlayers)
		}
	}

	for _, helper := range helpers {
		netLoss := defenderReport.PvpReinforcementLosses[helper.reinforcement.ID]["shuInfantry"]
		if netLoss != helper.amount*4/5 {
			t.Fatalf("expected %s net loss %d after Longdan, amount=%d losses=%+v", helper.playerID, helper.amount*4/5, helper.amount, defenderReport.PvpReinforcementLosses)
		}
		stored, err := repo.GetReinforcement(helper.reinforcement.ID)
		if err != nil || stored.RemainingTroops["shuInfantry"] != helper.amount-netLoss {
			t.Fatalf("expected %s stored troops %d, record=%+v err=%v", helper.playerID, helper.amount-netLoss, stored, err)
		}
		reports, total, err := repo.ListReportsByQuery(BattleReportQuery{PlayerID: helper.playerID, ViewType: ReportViewReinforcement, Page: 1, PageSize: 10})
		if err != nil || total != 1 || len(reports) != 1 {
			t.Fatalf("expected one helper report for %s, reports=%+v total=%d err=%v", helper.playerID, reports, total, err)
		}
		outcome := reports[0].TraitOutcomes["longdan_jiuyuan"]
		reduced, reducedOK := outcome.Detail["reducedLosses"].(map[string]int)
		if len(reports[0].TraitOutcomes) != 1 || outcome.OwnerPlayerID != helper.playerID || !reducedOK || reduced["shuInfantry"] != helper.amount/5 ||
			len(reports[0].Detail.Traits) != 1 || reports[0].Detail.Traits[0].OwnerPlayerID != helper.playerID {
			t.Fatalf("expected helper report to contain only its own Longdan result for %s, report=%+v", helper.playerID, reports[0])
		}
	}
}
