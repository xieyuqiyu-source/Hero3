// 本文件验证郭嘉鬼才遗策在 PVP 平局及援军平局中仍按真实阵亡复活。
package game

import (
	"testing"

	"hero3/internal/core/general"
)

// guicaiYiceDrawTrait 返回平局场景使用的当前鬼才遗策配置。
func guicaiYiceDrawTrait() GeneralTraitConfig {
	return GeneralTraitConfig{
		TraitID: "guicai_yice", TraitType: general.TraitTypeBonus, Enabled: true, Scope: "self_army",
		AllowedSides: []string{"attacker", "defender", "reinforcement"},
		Params:       map[string]float64{"effectRate": 0.22, "triggerChance": 1},
	}
}

// TestPvpGuicaiYiceRevivesActualLossesOnDraw 验证主动进攻平局不再被旧战败条件拦截。
func TestPvpGuicaiYiceRevivesActualLossesOnDraw(t *testing.T) {
	hero := GeneralHeroConfig{
		ID: "guojia", Name: "郭嘉", Faction: "wei", Enabled: true,
		BonusTrait: guicaiYiceDrawTrait(),
	}
	setTestFactionsAndGenerals(t, FactionsConfig{
		"wei": {Name: "魏国", Generals: []GeneralInfo{{ID: "guojia", Name: "郭嘉"}}},
		"shu": {Name: "蜀国", Generals: []GeneralInfo{{ID: "defender", Name: "守将"}}},
	}, GeneralsConfig{Enabled: true, Heroes: map[string]GeneralHeroConfig{
		"guojia": hero,
		"defender": {
			ID: "defender", Name: "守将", Faction: "shu", Enabled: true,
		},
	}})

	svc, repo, attacker, defender := newPvpTestServiceForGenerals(t, "wei", "guojia", "shu", "defender")
	attacker.Army = []ArmyUnit{{UnitType: "weiInfantry", Amount: 1000}}
	defender.Army = []ArmyUnit{{UnitType: "shuInfantry", Amount: 1000}}
	defender.Buildings = nil
	repo.players[attacker.Player.ID] = attacker
	repo.players[defender.Player.ID] = defender

	started, err := svc.StartPvpAttack(PvpAttackRequest{
		PlayerID: attacker.Player.ID, TargetPlayerID: defender.Player.ID, MarchMode: PvpMarchTypePlunder,
		Troops: map[string]int{"weiInfantry": 1000}, GeneralIDs: []string{"guojia"},
	})
	if err != nil {
		t.Fatalf("StartPvpAttack failed: %v", err)
	}
	forcePvpMarchDue(t, repo, started.March.ID)
	battle, err := svc.ResolvePvpMarch(started.March.ID)
	if err != nil {
		t.Fatalf("ResolvePvpMarch failed: %v", err)
	}
	if battle.Result["winner"] != "draw" || pvpTestLossesFromBattle(t, battle, "attacker")["weiInfantry"] != 500 {
		t.Fatalf("expected equal-power draw with 500 actual losses, battle=%+v", battle)
	}

	attackerReports, _, err := repo.ListReports(attacker.Player.ID, 10, 0)
	if err != nil || len(attackerReports) != 1 {
		t.Fatalf("expected one attacker report, reports=%+v err=%v", attackerReports, err)
	}
	report := attackerReports[0]
	outcome, ok := report.TraitOutcomes["guicai_yice"]
	actualLost, lostOK := outcome.Detail["actualLostUnits"].(map[string]int)
	revived, revivedOK := outcome.Detail["revivedUnits"].(map[string]int)
	if !ok || !lostOK || !revivedOK || actualLost["weiInfantry"] != 500 || revived["weiInfantry"] != 110 ||
		report.RevivedUnits["weiInfantry"] != 110 || report.SurvivedUnits["weiInfantry"] != 610 ||
		!standardReportHasTrait(report.Detail, "guicai_yice") {
		t.Fatalf("expected draw report to show 500 actual losses and 110 revivals, report=%+v", report)
	}
	storedMarch, err := repo.GetPvpMarch(started.March.ID)
	if err != nil || storedMarch.AttackTroops["weiInfantry"] != 610 {
		t.Fatalf("expected draw return march to contain 610 troops, march=%+v err=%v", storedMarch, err)
	}
}

// TestPvpReinforcementGuicaiYiceRevivesActualLossesOnDraw 验证郭嘉作为援军时平局也按自身真实阵亡复活。
func TestPvpReinforcementGuicaiYiceRevivesActualLossesOnDraw(t *testing.T) {
	cfg := reinforcementEnemyPvpConfig{
		id: "draw_reinforcement_guicai", attackerTroops: 1000, defenderTroops: 500, helperTroops: 500,
		marchMode:       PvpMarchTypePlunder,
		attackerFaction: "wei", attackerGeneral: "caocao", attackerName: "曹操",
		defenderFaction: "shu", defenderGeneral: "liubei", defenderName: "刘备",
		helperFaction: "shu", helperGeneralID: "guojia", helperName: "郭嘉",
		helperBonus: guicaiYiceDrawTrait(),
	}
	result := runReinforcementEnemyPvp(t, cfg)
	if result.battle.Result["winner"] != "draw" || result.storedReinforcement.Losses["shuInfantry"] != 195 ||
		result.storedReinforcement.RemainingTroops["shuInfantry"] != 305 {
		t.Fatalf("expected reinforcement gross loss 250, revival 55 and final loss 195, battle=%+v reinforcement=%+v", result.battle, result.storedReinforcement)
	}
	for _, report := range []BattleReport{result.attackerReport, result.defenderReport, result.reinforcementReport} {
		outcome, ok := report.TraitOutcomes["guicai_yice"]
		actualLost, lostOK := outcome.Detail["actualLostUnits"].(map[string]int)
		revived, revivedOK := outcome.Detail["revivedUnits"].(map[string]int)
		if !ok || !lostOK || !revivedOK || actualLost["shuInfantry"] != 250 || revived["shuInfantry"] != 55 ||
			outcome.OwnerSide != "reinforcement" || !standardReportHasTrait(report.Detail, "guicai_yice") {
			t.Fatalf("expected all reports to show reinforcement revival, report=%+v", report)
		}
	}
	if result.reinforcementReport.RevivedUnits["shuInfantry"] != 55 ||
		result.reinforcementReport.SurvivedUnits["shuInfantry"] != 305 {
		t.Fatalf("expected reinforcement owner report 500/250/55/305, report=%+v", result.reinforcementReport)
	}
}
