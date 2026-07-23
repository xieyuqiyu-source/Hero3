// 本文件验证许褚当前双特性在真实 PVP、方向、战报和权威库存中的完整行为。
package game

import (
	"testing"

	"hero3/internal/core/general"
)

type xuchuPvpResult struct {
	battle         PvpBattle
	march          PvpMarch
	storedMarch    PvpMarch
	attackerReport BattleReport
	defenderReport BattleReport
	attackerState  GameState
	defenderState  GameState
}

// runXuChuPvp 执行许褚率虎豹骑主动进攻，允许固定虎痴命中或未命中。
func runXuChuPvp(t *testing.T, triggerChance float64, passiveEnabled bool) xuchuPvpResult {
	t.Helper()
	setTestCombatUnitsConfig(t)
	setTestFactionsAndGenerals(t, FactionsConfig{
		"wei": {Name: "魏国", Generals: []GeneralInfo{{ID: "xuchu", Name: "许褚"}}},
		"shu": {Name: "蜀国", Generals: []GeneralInfo{{ID: "liubei", Name: "刘备"}}},
	}, GeneralsConfig{Enabled: true, Heroes: map[string]GeneralHeroConfig{
		"xuchu": {
			ID: "xuchu", Name: "许褚", Faction: "wei", Enabled: true,
			SpecialTrait: GeneralTraitConfig{
				TraitID: "huchi_chongzhen", TraitType: general.TraitTypeSpecial, Enabled: true,
				Scope: "enemy_army", AllowedSides: []string{"attacker"},
				Params: map[string]float64{"triggerChance": triggerChance, "enemyDefenseReductionRate": 0.3},
			},
			BonusTrait: GeneralTraitConfig{
				TraitID: "huhu_shengwei", TraitType: general.TraitTypeBonus, Enabled: passiveEnabled,
				Scope: "self_army", TargetUnitType: "huBaoQi",
				Params: map[string]float64{"unitAttackFlat": 12, "unitSpeedFlat": 5},
			},
		},
		"liubei": {ID: "liubei", Name: "刘备", Faction: "shu", Enabled: true},
	}})
	svc, repo, attacker, defender := newPvpTestServiceForGenerals(t, "wei", "xuchu", "shu", "liubei")
	attacker.Army = []ArmyUnit{{UnitType: "huBaoQi", Amount: 100}}
	defender.Army = []ArmyUnit{{UnitType: "shuInfantry", Amount: 1000}}
	defender.Buildings = nil
	repo.players[attacker.Player.ID] = attacker
	repo.players[defender.Player.ID] = defender

	started, err := svc.StartPvpAttack(PvpAttackRequest{
		PlayerID: attacker.Player.ID, TargetPlayerID: defender.Player.ID, MarchMode: PvpMarchTypeAttack,
		Troops: map[string]int{"huBaoQi": 100}, GeneralIDs: []string{"xuchu"},
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
	attackerState, err := repo.GetState(attacker.Player.ID)
	if err != nil {
		t.Fatalf("GetState attacker failed: %v", err)
	}
	defenderState, err := repo.GetState(defender.Player.ID)
	if err != nil {
		t.Fatalf("GetState defender failed: %v", err)
	}
	storedMarch, err := repo.GetPvpMarch(started.March.ID)
	if err != nil {
		t.Fatalf("GetPvpMarch failed: %v", err)
	}
	return xuchuPvpResult{battle: battle, march: started.March, storedMarch: storedMarch, attackerReport: attackerReport, defenderReport: defenderReport, attackerState: attackerState, defenderState: defenderState}
}

// TestPvpXuChuHitMissAndPassiveCombination 验证虎虎生威持续生效，虎痴命中与未命中分别重算防御战力和兵损。
func TestPvpXuChuHitMissAndPassiveCombination(t *testing.T) {
	control := runXuChuPvp(t, 0, false)
	miss := runXuChuPvp(t, 0, true)
	hit := runXuChuPvp(t, 1, true)

	controlAttack := control.battle.Result["attackerPower"].(float64)
	missAttack := miss.battle.Result["attackerPower"].(float64)
	hitAttack := hit.battle.Result["attackerPower"].(float64)
	controlDefense := control.battle.Result["defensePower"].(float64)
	missDefense := miss.battle.Result["defensePower"].(float64)
	hitDefense := hit.battle.Result["defensePower"].(float64)
	if controlAttack != 3000 || missAttack != 4200 || hitAttack != 4200 {
		t.Fatalf("expected 100 huBaoQi passive attack power 3000 -> 4200, control=%v miss=%v hit=%v", controlAttack, missAttack, hitAttack)
	}
	if controlDefense != 8000 || missDefense != 8000 || hitDefense != 6000 {
		t.Fatalf("expected Huchi hit to reduce cavalry-facing defense power 8000 -> 6000 only on hit, control=%v miss=%v hit=%v", controlDefense, missDefense, hitDefense)
	}
	controlLosses := pvpTestLossesFromBattle(t, control.battle, "defender")["shuInfantry"]
	missLosses := pvpTestLossesFromBattle(t, miss.battle, "defender")["shuInfantry"]
	hitLosses := pvpTestLossesFromBattle(t, hit.battle, "defender")["shuInfantry"]
	if controlLosses != 247 || missLosses != 400 || hitLosses != 602 {
		t.Fatalf("expected exact defender losses 247/400/602 for no passive, passive miss and passive hit, got %d/%d/%d", controlLosses, missLosses, hitLosses)
	}
	if pvpTestLossesFromBattle(t, control.battle, "attacker")["huBaoQi"] != 100 || pvpTestLossesFromBattle(t, miss.battle, "attacker")["huBaoQi"] != 100 || pvpTestLossesFromBattle(t, hit.battle, "attacker")["huBaoQi"] != 100 {
		t.Fatalf("expected exact attacker losses 100 in all three cases, control=%+v miss=%+v hit=%+v", control.battle.Result, miss.battle.Result, hit.battle.Result)
	}
	if control.march.DurationSeconds != 2054 || miss.march.DurationSeconds != 1450 || hit.march.DurationSeconds != 1450 {
		t.Fatalf("expected passive movement to shorten march 2054 -> 1450 seconds for hit and miss, control=%+v miss=%+v hit=%+v", control.march, miss.march, hit.march)
	}

	for _, report := range []BattleReport{miss.attackerReport, miss.defenderReport} {
		if len(report.TraitTriggered) != 0 || len(report.TraitOutcomes) != 0 || standardReportHasTrait(report.Detail, "huhu_shengwei") || standardReportHasTrait(report.Detail, "huchi_chongzhen") {
			t.Fatalf("expected legal Huchi miss and passive absent from triggered timeline, report=%+v", report)
		}
		if report.Detail == nil || len(report.Detail.PrimarySide.Generals) != 1 {
			t.Fatalf("expected authoritative Xu Chu snapshot, detail=%+v", report.Detail)
		}
		generalSnapshot := report.Detail.PrimarySide.Generals[0]
		if !standardGeneralHasTrait(report.Detail.PrimarySide.Generals, "huhu_shengwei") || generalSnapshot.Buffs[unitAttackFlatModifierKey("huBaoQi")] != 12 || generalSnapshot.Buffs[unitSpeedFlatModifierKey("huBaoQi")] != 5 {
			t.Fatalf("expected passive +12/+5 in Xu Chu snapshot, general=%+v", generalSnapshot)
		}
	}
	for _, report := range []BattleReport{hit.attackerReport, hit.defenderReport} {
		if len(report.TraitTriggered) != 1 || report.TraitTriggered[0] != "huchi_chongzhen" || len(report.TraitOutcomes) != 1 || standardReportHasTrait(report.Detail, "huhu_shengwei") {
			t.Fatalf("expected only Huchi in triggered timeline, report=%+v", report)
		}
		outcome := report.TraitOutcomes["huchi_chongzhen"]
		infantry, infantryOK := outcome.Detail["infantryDefenseModifiedUnits"].(map[string]int)
		cavalry, cavalryOK := outcome.Detail["cavalryDefenseModifiedUnits"].(map[string]int)
		if !infantryOK || !cavalryOK || outcome.Detail["triggerChance"] != 1.0 || outcome.Detail["enemyDefenseReductionRate"] != 0.3 || infantry["shuInfantry"] != -3 || cavalry["shuInfantry"] != -2 || outcome.OwnerSide != "attacker" || outcome.OwnerGeneralID != "xuchu" {
			t.Fatalf("expected Huchi design values and actual -3/-2 defense deltas, outcome=%+v", outcome)
		}
	}

	for _, result := range []xuchuPvpResult{control, miss, hit} {
		attackerLosses := pvpTestLossesFromBattle(t, result.battle, "attacker")["huBaoQi"]
		defenderLosses := pvpTestLossesFromBattle(t, result.battle, "defender")["shuInfantry"]
		if result.storedMarch.AttackTroops["huBaoQi"] != 100-attackerLosses || armySliceToMap(result.defenderState.Army)["shuInfantry"] != 1000-defenderLosses {
			t.Fatalf("expected authoritative survivors to match core losses, march=%+v defender=%+v losses=%d/%d", result.storedMarch, result.defenderState.Army, attackerLosses, defenderLosses)
		}
		if pvpTestGeneralExp(result.attackerState, "xuchu") != defenderLosses || result.attackerReport.GeneralExpGained != defenderLosses {
			t.Fatalf("expected Xu Chu exp to equal real defender deaths, state=%+v report=%+v losses=%d", result.attackerState.Generals, result.attackerReport, defenderLosses)
		}
	}
}
