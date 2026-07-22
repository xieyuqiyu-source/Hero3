// 本文件验证甄宓两项概率特性在真实 PVP 战前独立判定，并以修改后的攻防重新计算兵损和战报。
package game

import (
	"reflect"
	"testing"

	"hero3/internal/core/general"
)

// TestZhenMiDualTraitsIndependentlyRecalculatePvpCombat 验证四种命中组合的真实战力、兵损、战报和权威状态。
func TestZhenMiDualTraitsIndependentlyRecalculatePvpCombat(t *testing.T) {
	tests := []struct {
		name             string
		attackChance     float64
		defenseChance    float64
		wantAttackPower  int
		wantDefensePower int
		wantTimeline     []string
	}{
		{name: "两项同时触发", attackChance: 1, defenseChance: 1, wantAttackPower: 13000, wantDefensePower: 8000, wantTimeline: []string{"meiren", "meihuo_raozhen"}},
		{name: "仅美人心计触发", attackChance: 1, defenseChance: 0, wantAttackPower: 13000, wantDefensePower: 10000, wantTimeline: []string{"meiren"}},
		{name: "仅魅惑扰阵触发", attackChance: 0, defenseChance: 1, wantAttackPower: 10000, wantDefensePower: 8000, wantTimeline: []string{"meihuo_raozhen"}},
		{name: "两项均未触发", attackChance: 0, defenseChance: 0, wantAttackPower: 10000, wantDefensePower: 10000},
	}
	type losses struct {
		attacker int
		defender int
	}
	results := map[string]losses{}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			report := resolveZhenMiPvpCombination(t, tc.attackChance, tc.defenseChance)
			if report.PlayerPower != tc.wantAttackPower || report.EnemyPower != tc.wantDefensePower {
				t.Fatalf("expected recalculated power %d/%d, got %d/%d", tc.wantAttackPower, tc.wantDefensePower, report.PlayerPower, report.EnemyPower)
			}
			if !reflect.DeepEqual(report.TraitTriggered, tc.wantTimeline) {
				t.Fatalf("expected timeline %v, got %v", tc.wantTimeline, report.TraitTriggered)
			}
			if report.Detail == nil || len(report.Detail.Traits) != len(tc.wantTimeline) {
				t.Fatalf("expected standard report to contain only hit traits, detail=%+v", report.Detail)
			}
			if len(report.CapturedUnits) != 0 || len(report.CapturedToGarrison) != 0 {
				t.Fatalf("expected removed capture trait to produce no captured troops, report=%+v", report)
			}
			assertZhenMiTraitDetails(t, report, tc.attackChance > 0, tc.defenseChance > 0)
			results[tc.name] = losses{attacker: report.LostUnits["weiInfantry"], defender: report.DefenderLostUnits["shuInfantry"]}
		})
	}

	baseline := results["两项均未触发"]
	for _, name := range []string{"两项同时触发", "仅美人心计触发", "仅魅惑扰阵触发"} {
		changed := results[name]
		if changed.attacker >= baseline.attacker || changed.defender < baseline.defender || changed == baseline {
			t.Fatalf("expected %s to recalculate into fewer attacker losses without reducing defender losses, baseline=%+v changed=%+v", name, baseline, changed)
		}
	}
}

// TestZhenMiTraitsDoNotTriggerWhileDefending 验证甄宓两项主动进攻特性在真实防守战中均不触发。
func TestZhenMiTraitsDoNotTriggerWhileDefending(t *testing.T) {
	setTestFactionsAndGenerals(t, FactionsConfig{
		"shu": {Name: "蜀国", Generals: []GeneralInfo{{ID: "liubei", Name: "刘备"}}},
		"wei": {Name: "魏国", Generals: []GeneralInfo{{ID: "zhenmi", Name: "甄宓"}}},
	}, GeneralsConfig{Enabled: true, Heroes: map[string]GeneralHeroConfig{
		"liubei": {ID: "liubei", Name: "刘备", Faction: "shu", Enabled: true},
		"zhenmi": {
			ID: "zhenmi", Name: "甄宓", Faction: "wei", Enabled: true,
			SpecialTrait: GeneralTraitConfig{
				TraitID: "meiren", TraitType: general.TraitTypeSpecial, Enabled: true,
				Scope: "self_army", AllowedSides: []string{"attacker"},
				Params: map[string]float64{"attackBonusRate": 0.25, "triggerChance": 1},
			},
			BonusTrait: GeneralTraitConfig{
				TraitID: "meihuo_raozhen", TraitType: general.TraitTypeBonus, Enabled: true,
				Scope: "enemy_army", AllowedSides: []string{"attacker"},
				Params: map[string]float64{"enemyDefenseReductionRate": 0.25, "triggerChance": 1},
			},
		},
	}})

	svc, repo, attacker, defender := newPvpTestServiceForGenerals(t, "shu", "liubei", "wei", "zhenmi")
	attacker.Army = []ArmyUnit{{UnitType: "shuInfantry", Amount: 1000}}
	defender.Army = []ArmyUnit{{UnitType: "weiInfantry", Amount: 1000}}
	defender.Buildings = nil
	repo.players[attacker.Player.ID] = attacker
	repo.players[defender.Player.ID] = defender

	started, err := svc.StartPvpAttack(PvpAttackRequest{
		PlayerID: attacker.Player.ID, TargetPlayerID: defender.Player.ID, MarchMode: PvpMarchTypeAttack,
		Troops: map[string]int{"shuInfantry": 1000},
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
		t.Fatalf("expected base power 10000/10000 while Zhen Mi defends, result=%+v", battle.Result)
	}
	for _, playerID := range []string{attacker.Player.ID, defender.Player.ID} {
		reports, _, listErr := repo.ListReports(playerID, 10, 0)
		if listErr != nil || len(reports) != 1 {
			t.Fatalf("expected one report for %s, reports=%+v err=%v", playerID, reports, listErr)
		}
		report := reports[0]
		if len(report.TraitTriggered) != 0 || len(report.TraitOutcomes) != 0 || report.Detail == nil || len(report.Detail.Traits) != 0 || len(report.CapturedUnits) != 0 || len(report.CapturedToGarrison) != 0 {
			t.Fatalf("expected no Zhen Mi outcome while defending, report=%+v", report)
		}
		if len(report.PvpDefenderGenerals) != 1 || !pvpSnapshotHasTrait(report.PvpDefenderGenerals[0], "meiren") || !pvpSnapshotHasTrait(report.PvpDefenderGenerals[0], "meihuo_raozhen") {
			t.Fatalf("expected ownership snapshot without trigger timeline, report=%+v", report)
		}
	}
}

// resolveZhenMiPvpCombination 以可控概率完成一场真实甄宓 PVP，并返回进攻方战报。
func resolveZhenMiPvpCombination(t *testing.T, attackChance float64, defenseChance float64) BattleReport {
	t.Helper()
	setTestFactionsAndGenerals(t, FactionsConfig{
		"wei": {Name: "魏国", Generals: []GeneralInfo{{ID: "zhenmi", Name: "甄宓"}}},
		"shu": {Name: "蜀国", Generals: []GeneralInfo{{ID: "liubei", Name: "刘备"}}},
	}, GeneralsConfig{Enabled: true, Heroes: map[string]GeneralHeroConfig{
		"zhenmi": {
			ID: "zhenmi", Name: "甄宓", Faction: "wei", Enabled: true,
			SpecialTrait: GeneralTraitConfig{
				TraitID: "meiren", TraitType: general.TraitTypeSpecial, Enabled: true,
				Scope: "self_army", AllowedSides: []string{"attacker"},
				Params: map[string]float64{"attackBonusRate": 0.25, "triggerChance": attackChance},
			},
			BonusTrait: GeneralTraitConfig{
				TraitID: "meihuo_raozhen", TraitType: general.TraitTypeBonus, Enabled: true,
				Scope: "enemy_army", AllowedSides: []string{"attacker"},
				Params: map[string]float64{"enemyDefenseReductionRate": 0.25, "triggerChance": defenseChance},
			},
		},
		"liubei": {ID: "liubei", Name: "刘备", Faction: "shu", Enabled: true},
	}})

	svc, repo, attacker, defender := newPvpTestServiceForGenerals(t, "wei", "zhenmi", "shu", "liubei")
	attacker.Army = []ArmyUnit{{UnitType: "weiInfantry", Amount: 1000}}
	defender.Army = []ArmyUnit{{UnitType: "shuInfantry", Amount: 1000}}
	defender.Buildings = nil
	repo.players[attacker.Player.ID] = attacker
	repo.players[defender.Player.ID] = defender

	started, err := svc.StartPvpAttack(PvpAttackRequest{
		PlayerID: attacker.Player.ID, TargetPlayerID: defender.Player.ID, MarchMode: PvpMarchTypeAttack,
		Troops: map[string]int{"weiInfantry": 1000}, GeneralIDs: []string{"zhenmi"},
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
	if err != nil || len(attackerReports) != 1 || attackerReports[0].ID != battle.AttackerReportID {
		t.Fatalf("expected one attacker report, reports=%+v err=%v", attackerReports, err)
	}
	defenderReports, _, err := repo.ListReports(defender.Player.ID, 10, 0)
	if err != nil || len(defenderReports) != 1 || defenderReports[0].ID != battle.DefenderReportID {
		t.Fatalf("expected one defender report, reports=%+v err=%v", defenderReports, err)
	}
	if !reflect.DeepEqual(attackerReports[0].TraitTriggered, defenderReports[0].TraitTriggered) || !reflect.DeepEqual(attackerReports[0].TraitOutcomes, defenderReports[0].TraitOutcomes) {
		t.Fatalf("expected both reports to persist the same Zhen Mi results, attacker=%+v defender=%+v", attackerReports[0], defenderReports[0])
	}
	if attackerReports[0].LostUnits["weiInfantry"]+attackerReports[0].SurvivedUnits["weiInfantry"] != 1000 || defenderReports[0].LostUnits["shuInfantry"]+defenderReports[0].SurvivedUnits["shuInfantry"] != 1000 {
		t.Fatalf("expected both report views to reconcile dispatched troops, attacker=%+v defender=%+v", attackerReports[0], defenderReports[0])
	}
	storedDefender, err := repo.GetState(defender.Player.ID)
	if err != nil || armySliceToMap(storedDefender.Army)["shuInfantry"] != attackerReports[0].DefenderUnits["shuInfantry"]-attackerReports[0].DefenderLostUnits["shuInfantry"] {
		t.Fatalf("expected defender state to match recalculated losses, state=%+v report=%+v err=%v", storedDefender.Army, attackerReports[0], err)
	}
	return attackerReports[0]
}

// assertZhenMiTraitDetails 核对命中项的设计比例、实际逐兵种变化和正式中文名。
func assertZhenMiTraitDetails(t *testing.T, report BattleReport, attackHit bool, defenseHit bool) {
	t.Helper()
	attackOutcome, hasAttack := report.TraitOutcomes["meiren"]
	if hasAttack != attackHit {
		t.Fatalf("expected 美人心计 hit=%t, outcomes=%+v", attackHit, report.TraitOutcomes)
	}
	if attackHit {
		changed, ok := attackOutcome.Detail["attackModifiedUnits"].(map[string]int)
		if !ok || attackOutcome.Name != "美人心计" || attackOutcome.Detail["attackBonusRate"] != 0.25 || changed["weiInfantry"] != 3 || attackOutcome.Detail["triggerChance"] != 1.0 {
			t.Fatalf("expected exact 美人心计 report detail, outcome=%+v", attackOutcome)
		}
	}
	defenseOutcome, hasDefense := report.TraitOutcomes["meihuo_raozhen"]
	if hasDefense != defenseHit {
		t.Fatalf("expected 魅惑扰阵 hit=%t, outcomes=%+v", defenseHit, report.TraitOutcomes)
	}
	if defenseHit {
		infantry, infantryOK := defenseOutcome.Detail["infantryDefenseModifiedUnits"].(map[string]int)
		cavalry, cavalryOK := defenseOutcome.Detail["cavalryDefenseModifiedUnits"].(map[string]int)
		if !infantryOK || !cavalryOK || defenseOutcome.Detail["enemyDefenseReductionRate"] != 0.25 || infantry["shuInfantry"] != -2 || cavalry["shuInfantry"] != -2 || defenseOutcome.Detail["triggerChance"] != 1.0 {
			t.Fatalf("expected exact 魅惑扰阵 report detail, outcome=%+v", defenseOutcome)
		}
	}
}
