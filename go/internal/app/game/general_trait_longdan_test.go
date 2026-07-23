// 本文件保留赵云行军与既有 PVP 测试辅助，龙胆新规则由蜀将批次测试统一验收。
package game

import (
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
