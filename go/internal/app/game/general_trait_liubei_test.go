// 本文件验收刘备永久内政/统率被动与战后概率复活，确保属性、状态和战报一致。
package game

import (
	"math"
	"reflect"
	"testing"

	"hero3/internal/core/combat"
	"hero3/internal/core/general"
)

// TestNormalizeLiubeiLegacyTraits 验证旧双返兵配置被完整迁移为永久四维和概率复活。
func TestNormalizeLiubeiLegacyTraits(t *testing.T) {
	cfg := NormalizeGeneralsConfig(GeneralsConfig{Heroes: map[string]GeneralHeroConfig{
		"liubei": {
			SpecialTrait: GeneralTraitConfig{
				TraitID: "rende", Scope: "self_army",
				Params: map[string]float64{"effectRate": 0.5, "reviveRate": 0.5, "maxReviveCount": 10000, "triggerChance": 1},
			},
			BonusTrait: GeneralTraitConfig{
				TraitID: "renzhu_shouhu", Scope: "self_army",
				Params: map[string]float64{
					"lossReductionRate": 0.1,
					"maxReturnCount":    10000,
					"maxAffectedRate":   0.5,
					"maxAffectedCount":  50000,
				},
			},
		},
	}})
	hero := cfg.Heroes["liubei"]
	if hero.SpecialTrait.Scope != "self_army" ||
		!reflect.DeepEqual(hero.SpecialTrait.Params, map[string]float64{"politicsBonus": 10, "commandBonus": 12}) {
		t.Fatalf("expected legacy Rende migrated to permanent stats, got %+v", hero.SpecialTrait)
	}
	if !reflect.DeepEqual(hero.BonusTrait.AllowedSides, []string{"attacker", "defender", "reinforcement"}) ||
		!reflect.DeepEqual(hero.BonusTrait.Params, map[string]float64{"effectRate": 0.35, "triggerChance": 0.6}) {
		t.Fatalf("expected legacy Renzhu return migrated to probability revival, got %+v", hero.BonusTrait)
	}
}

// TestNormalizeLiubeiPreservesCurrentGMValues 验证当前 GM 四维、复活比例和概率不会被默认值覆盖。
func TestNormalizeLiubeiPreservesCurrentGMValues(t *testing.T) {
	cfg := NormalizeGeneralsConfig(GeneralsConfig{Heroes: map[string]GeneralHeroConfig{
		"liubei": {
			SpecialTrait: GeneralTraitConfig{
				TraitID: "rende",
				Params:  map[string]float64{"politicsBonus": 13, "commandBonus": 17},
			},
			BonusTrait: GeneralTraitConfig{
				TraitID: "renzhu_shouhu",
				Params:  map[string]float64{"effectRate": 0.41, "triggerChance": 0.72},
			},
		},
	}})
	hero := cfg.Heroes["liubei"]
	if !reflect.DeepEqual(hero.SpecialTrait.Params, map[string]float64{"politicsBonus": 13, "commandBonus": 17}) ||
		!reflect.DeepEqual(hero.BonusTrait.Params, map[string]float64{"effectRate": 0.41, "triggerChance": 0.72}) {
		t.Fatalf("expected current GM values preserved, hero=%+v", hero)
	}
}

// TestLiubeiTraitSchemasExposeOnlyCurrentGMFields 验证 GM 只暴露当前永久属性、复活比例和概率。
func TestLiubeiTraitSchemasExposeOnlyCurrentGMFields(t *testing.T) {
	special, ok := general.Get("rende")
	if !ok {
		t.Fatal("rende not registered")
	}
	specialFields := map[string]general.ParamField{}
	for _, field := range special.ParamSchema() {
		specialFields[field.Key] = field
	}
	if len(specialFields) != 2 || specialFields["politicsBonus"].Default != 10 || specialFields["commandBonus"].Default != 12 {
		t.Fatalf("unexpected Rende permanent stat fields: %+v", specialFields)
	}

	bonus, ok := general.Get("renzhu_shouhu")
	if !ok {
		t.Fatal("renzhu_shouhu not registered")
	}
	bonusFields := map[string]general.ParamField{}
	for _, field := range bonus.ParamSchema() {
		bonusFields[field.Key] = field
	}
	if len(bonusFields) != 2 || bonusFields["effectRate"].Default != 0.35 || bonusFields["triggerChance"].Default != 0.6 {
		t.Fatalf("unexpected Renzhu revival fields: %+v", bonusFields)
	}
	for _, legacyOrIrrelevant := range []string{"lossReductionRate", "maxReturnCount", "maxReviveCount", "maxAffectedRate", "maxAffectedCount"} {
		if _, exists := bonusFields[legacyOrIrrelevant]; exists {
			t.Fatalf("legacy or irrelevant field %s must not remain in GM schema: %+v", legacyOrIrrelevant, bonusFields)
		}
	}
}

// liubeiCurrentTestConfig 返回刘备当前正式语义的测试配置。
func liubeiCurrentTestConfig(chance float64) GeneralHeroConfig {
	return GeneralHeroConfig{
		ID: "liubei", Name: "刘备", Faction: "shu", Enabled: true,
		SpecialTrait: GeneralTraitConfig{
			TraitID: "rende", TraitType: general.TraitTypeSpecial, Enabled: true, Scope: "self_army",
			Params: map[string]float64{"politicsBonus": 10, "commandBonus": 12},
		},
		BonusTrait: GeneralTraitConfig{
			TraitID: "renzhu_shouhu", TraitType: general.TraitTypeBonus, Enabled: true, Scope: "self_army",
			AllowedSides: []string{"attacker", "defender", "reinforcement"},
			Params:       map[string]float64{"effectRate": 0.35, "triggerChance": chance},
		},
	}
}

// TestRendeAddsPoliticsAndCommandWithoutBattleTrigger 验证永久被动进入最终属性和真实效果但不进入时间线。
func TestRendeAddsPoliticsAndCommandWithoutBattleTrigger(t *testing.T) {
	setTestFactionsAndGenerals(t, FactionsConfig{
		"shu": {Name: "蜀国", Generals: []GeneralInfo{{ID: "liubei", Name: "刘备"}}},
	}, GeneralsConfig{Enabled: true, Heroes: map[string]GeneralHeroConfig{"liubei": liubeiCurrentTestConfig(0.6)}})

	liubei := newGeneral("shu", "liubei")
	liubei.Stats = map[string]int{"politics": 4, "command": 3}
	applyHeroConfigToGeneral(liubei)
	if liubei.EffectiveStats["politics"] != 14 || liubei.EffectiveStats["command"] != 15 {
		t.Fatalf("expected politics/command 4/3 -> 14/15, got %+v", liubei.EffectiveStats)
	}
	if math.Abs(liubei.Buffs[StatProductionBonus]-0.28) > 1e-9 ||
		math.Abs(liubei.Buffs[StatCapacityBonus]-0.28) > 1e-9 ||
		math.Abs(liubei.Buffs[StatDefenseBonus]-0.30) > 1e-9 {
		t.Fatalf("expected permanent stats to enter real modifiers, buffs=%+v", liubei.Buffs)
	}

	attacker := combat.Army{Units: []combat.Unit{{ID: "shuInfantry", Count: 100, Attack: 10}}}
	defender := combat.Army{Units: []combat.Unit{{ID: "weiInfantry", Count: 100, Attack: 10}}}
	ctx := &general.BeforeBattleContext{Attacker: &attacker, Defender: &defender, AttackerOwnsTrait: true, Scene: "attack"}
	general.Dispatch(ctx, buildActiveTraits(liubei))
	if _, ok := ctx.Triggered["rende"]; ok {
		t.Fatalf("expected permanent Rende absent from battle timeline, outcomes=%+v", ctx.Triggered)
	}
}

// TestPvpLiubeiRenzhuHitRevivesActualLossesAndMatchesReports 验证概率命中后按真实阵亡复活并进入返程兵力。
func TestPvpLiubeiRenzhuHitRevivesActualLossesAndMatchesReports(t *testing.T) {
	result := runLiubeiCurrentPvp(t, 1)
	if result.losses != 100 || result.revived != 35 || result.returned != 35 {
		t.Fatalf("expected 100 real losses, 35 revived and 35 returned, result=%+v", result)
	}
	for _, report := range result.reports {
		assertLiubeiCurrentReport(t, report, true, 100, 35)
	}
}

// TestPvpLiubeiRenzhuMissLeavesLossesAndTimelineUntouched 验证合法未命中时不复活、不改返程且不生成触发项。
func TestPvpLiubeiRenzhuMissLeavesLossesAndTimelineUntouched(t *testing.T) {
	result := runLiubeiCurrentPvp(t, 0)
	if result.losses != 100 || result.revived != 0 || result.returned != 0 {
		t.Fatalf("expected probability miss to keep full losses, result=%+v", result)
	}
	for _, report := range result.reports {
		assertLiubeiCurrentReport(t, report, false, 100, 0)
	}
}

type liubeiCurrentPvpResult struct {
	losses   int
	revived  int
	returned int
	reports  []BattleReport
}

// runLiubeiCurrentPvp 执行刘备主动进攻真实 PVP，并返回原始阵亡、复活和返程结果。
func runLiubeiCurrentPvp(t *testing.T, chance float64) liubeiCurrentPvpResult {
	t.Helper()
	setTestFactionsAndGenerals(t, FactionsConfig{
		"shu": {Name: "蜀国", Generals: []GeneralInfo{{ID: "liubei", Name: "刘备"}}},
		"wei": {Name: "魏国", Generals: []GeneralInfo{{ID: "caocao", Name: "曹操"}}},
	}, GeneralsConfig{Enabled: true, Heroes: map[string]GeneralHeroConfig{
		"liubei": liubeiCurrentTestConfig(chance),
		"caocao": {ID: "caocao", Name: "曹操", Faction: "wei", Enabled: true},
	}})

	svc, repo, attacker, defender := newPvpTestServiceForGenerals(t, "shu", "liubei", "wei", "caocao")
	attacker.Army = []ArmyUnit{{UnitType: "shuInfantry", Amount: 100}}
	defender.Army = []ArmyUnit{{UnitType: "weiInfantry", Amount: 1000}}
	defender.Buildings = nil
	repo.players[attacker.Player.ID] = attacker
	repo.players[defender.Player.ID] = defender

	started, err := svc.StartPvpAttack(PvpAttackRequest{
		PlayerID: attacker.Player.ID, TargetPlayerID: defender.Player.ID, MarchMode: PvpMarchTypeAttack,
		Troops: map[string]int{"shuInfantry": 100}, GeneralIDs: []string{"liubei"},
	})
	if err != nil {
		t.Fatalf("StartPvpAttack failed: %v", err)
	}
	forcePvpMarchDue(t, repo, started.March.ID)
	battle, err := svc.ResolvePvpMarch(started.March.ID)
	if err != nil {
		t.Fatalf("ResolvePvpMarch failed: %v", err)
	}
	losses := pvpTestLossesFromBattle(t, battle, "attacker")["shuInfantry"]

	attackerReports, _, err := repo.ListReports(attacker.Player.ID, 10, 0)
	if err != nil || len(attackerReports) == 0 {
		t.Fatalf("expected attacker report, reports=%+v err=%v", attackerReports, err)
	}
	defenderReports, _, err := repo.ListReports(defender.Player.ID, 10, 0)
	if err != nil || len(defenderReports) == 0 {
		t.Fatalf("expected defender report, reports=%+v err=%v", defenderReports, err)
	}
	storedMarch, err := repo.GetPvpMarch(started.March.ID)
	if err != nil {
		t.Fatalf("GetPvpMarch failed: %v", err)
	}
	return liubeiCurrentPvpResult{
		losses: losses, revived: attackerReports[0].RevivedUnits["shuInfantry"],
		returned: storedMarch.AttackTroops["shuInfantry"],
		reports:  []BattleReport{attackerReports[0], defenderReports[0]},
	}
}

// assertLiubeiCurrentReport 核对永久被动只留在快照，实际复活按命中状态进入战报。
func assertLiubeiCurrentReport(t *testing.T, report BattleReport, triggered bool, actualLost int, revived int) {
	t.Helper()
	if _, exists := report.TraitOutcomes["rende"]; exists {
		t.Fatalf("expected permanent Rende absent from battle outcomes, report=%+v", report)
	}
	if triggered {
		if len(report.TraitTriggered) != 1 || report.TraitTriggered[0] != "renzhu_shouhu" || len(report.TraitOutcomes) != 1 {
			t.Fatalf("expected only Renzhu in timeline, report=%+v", report)
		}
		outcome := report.TraitOutcomes["renzhu_shouhu"]
		actual, actualOK := outcome.Detail["actualLostUnits"].(map[string]int)
		revivedUnits, revivedOK := outcome.Detail["revivedUnits"].(map[string]int)
		if !actualOK || !revivedOK || outcome.OwnerGeneralID != "liubei" || outcome.OwnerSide != "attacker" ||
			outcome.Detail["effectRate"] != 0.35 || outcome.Detail["triggerChance"] != 1.0 ||
			actual["shuInfantry"] != actualLost || revivedUnits["shuInfantry"] != revived ||
			outcome.Detail["totalRevived"] != revived {
			t.Fatalf("expected exact Renzhu revival detail, outcome=%+v", outcome)
		}
		if report.Detail == nil || len(report.Detail.Traits) != 1 || report.Detail.Traits[0].TraitID != "renzhu_shouhu" {
			t.Fatalf("expected only Renzhu in standard timeline, detail=%+v", report.Detail)
		}
	} else if len(report.TraitTriggered) != 0 || len(report.TraitOutcomes) != 0 ||
		(report.Detail != nil && len(report.Detail.Traits) != 0) {
		t.Fatalf("expected probability miss to leave timeline empty, report=%+v", report)
	}

	if len(report.PvpAttackerGenerals) != 1 || !pvpSnapshotHasTrait(report.PvpAttackerGenerals[0], "rende") ||
		!pvpSnapshotHasTrait(report.PvpAttackerGenerals[0], "renzhu_shouhu") ||
		report.PvpAttackerGenerals[0].EffectiveStats["politics"]-report.PvpAttackerGenerals[0].Stats["politics"] != 10 ||
		report.PvpAttackerGenerals[0].EffectiveStats["command"]-report.PvpAttackerGenerals[0].Stats["command"] != 12 {
		t.Fatalf("expected snapshot to retain both traits and permanent stats, snapshots=%+v", report.PvpAttackerGenerals)
	}
}
