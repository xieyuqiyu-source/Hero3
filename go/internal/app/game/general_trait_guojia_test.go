// 本文件验收郭嘉永久四维被动与战后真实阵亡复活，确保状态、战报和时间线一致。
package game

import (
	"math"
	"reflect"
	"testing"

	"hero3/internal/core/combat"
	"hero3/internal/core/general"
)

// TestNormalizeGuojiaLegacyTraits 验证旧征兵减耗和仅战败返兵配置被完整迁移。
func TestNormalizeGuojiaLegacyTraits(t *testing.T) {
	cfg := NormalizeGeneralsConfig(GeneralsConfig{Heroes: map[string]GeneralHeroConfig{
		"guojia": {
			SpecialTrait: GeneralTraitConfig{
				TraitID: "shengui_zhicai", Scope: "self_city",
				Params: map[string]float64{"resourceCostReduction": 0.5, "triggerChance": 1},
			},
			BonusTrait: GeneralTraitConfig{
				TraitID: "guicai_yice", Scope: "self_army", RequiredOutcome: "loss",
				Params: map[string]float64{
					"lossReductionRate": 0.1,
					"maxReturnCount":    10000,
					"maxAffectedRate":   0.5,
					"maxAffectedCount":  50000,
				},
			},
		},
	}})
	hero := cfg.Heroes["guojia"]
	if hero.SpecialTrait.Scope != "self_army" ||
		!reflect.DeepEqual(hero.SpecialTrait.Params, map[string]float64{"politicsBonus": 10, "intelligenceBonus": 10}) {
		t.Fatalf("expected legacy recruit reduction migrated to permanent stats, got %+v", hero.SpecialTrait)
	}
	if hero.BonusTrait.RequiredOutcome != "" ||
		!reflect.DeepEqual(hero.BonusTrait.AllowedSides, []string{"attacker", "defender", "reinforcement"}) ||
		!reflect.DeepEqual(hero.BonusTrait.Params, map[string]float64{"effectRate": 0.22, "triggerChance": 1}) {
		t.Fatalf("expected legacy loss-only return migrated to all-result revival, got %+v", hero.BonusTrait)
	}
}

// TestNormalizeGuojiaPreservesCurrentGMValues 验证新格式的 GM 四维、比例和概率不会被默认值覆盖。
func TestNormalizeGuojiaPreservesCurrentGMValues(t *testing.T) {
	cfg := NormalizeGeneralsConfig(GeneralsConfig{Heroes: map[string]GeneralHeroConfig{
		"guojia": {
			SpecialTrait: GeneralTraitConfig{
				TraitID: "shengui_zhicai",
				Params:  map[string]float64{"politicsBonus": 13, "intelligenceBonus": 17},
			},
			BonusTrait: GeneralTraitConfig{
				TraitID: "guicai_yice",
				Params:  map[string]float64{"effectRate": 0.31, "triggerChance": 0.63},
			},
		},
	}})
	hero := cfg.Heroes["guojia"]
	if !reflect.DeepEqual(hero.SpecialTrait.Params, map[string]float64{"politicsBonus": 13, "intelligenceBonus": 17}) ||
		!reflect.DeepEqual(hero.BonusTrait.Params, map[string]float64{"effectRate": 0.31, "triggerChance": 0.63}) {
		t.Fatalf("expected current GM values preserved, hero=%+v", hero)
	}
}

// TestGuojiaTraitSchemasExposeOnlyCurrentGMFields 验证 GM 仅看到当前永久属性和复活概率参数。
func TestGuojiaTraitSchemasExposeOnlyCurrentGMFields(t *testing.T) {
	special, ok := general.Get("shengui_zhicai")
	if !ok {
		t.Fatal("shengui_zhicai not registered")
	}
	specialFields := map[string]general.ParamField{}
	for _, field := range special.ParamSchema() {
		specialFields[field.Key] = field
	}
	if len(specialFields) != 2 || specialFields["politicsBonus"].Default != 10 || specialFields["intelligenceBonus"].Default != 10 {
		t.Fatalf("unexpected permanent stat GM fields: %+v", specialFields)
	}
	bonus, ok := general.Get("guicai_yice")
	if !ok {
		t.Fatal("guicai_yice not registered")
	}
	bonusFields := map[string]general.ParamField{}
	for _, field := range bonus.ParamSchema() {
		bonusFields[field.Key] = field
	}
	if len(bonusFields) != 2 || bonusFields["effectRate"].Default != 0.22 || bonusFields["triggerChance"].Default != 1 {
		t.Fatalf("unexpected revival GM fields: %+v", bonusFields)
	}
	for _, legacyOrIrrelevant := range []string{"lossReductionRate", "maxReturnCount", "maxAffectedRate", "maxAffectedCount"} {
		if _, exists := bonusFields[legacyOrIrrelevant]; exists {
			t.Fatalf("legacy or irrelevant field %s must not remain in GM schema: %+v", legacyOrIrrelevant, bonusFields)
		}
	}
}

// guojiaTestConfig 返回郭嘉当前正式语义的测试配置。
func guojiaTestConfig(reviveRate float64) GeneralHeroConfig {
	return GeneralHeroConfig{
		ID: "guojia", Name: "郭嘉", Faction: "wei", Enabled: true,
		SpecialTrait: GeneralTraitConfig{
			TraitID: "shengui_zhicai", TraitType: general.TraitTypeSpecial, Enabled: true, Scope: "self_army",
			Params: map[string]float64{"politicsBonus": 10, "intelligenceBonus": 10},
		},
		BonusTrait: GeneralTraitConfig{
			TraitID: "guicai_yice", TraitType: general.TraitTypeBonus, Enabled: true, Scope: "self_army",
			AllowedSides: []string{"attacker", "defender", "reinforcement"},
			Params:       map[string]float64{"effectRate": reviveRate, "triggerChance": 1},
		},
	}
}

// TestShenguiZhicaiAddsPoliticsAndIntelligenceWithoutBattleTrigger 验证永久被动进入最终属性和真实效果，但不进入战斗触发时间线。
func TestShenguiZhicaiAddsPoliticsAndIntelligenceWithoutBattleTrigger(t *testing.T) {
	setTestFactionsAndGenerals(t, FactionsConfig{
		"wei": {Name: "魏国", Generals: []GeneralInfo{{ID: "guojia", Name: "郭嘉"}}},
	}, GeneralsConfig{Enabled: true, Heroes: map[string]GeneralHeroConfig{"guojia": guojiaTestConfig(0.22)}})

	guojia := newGeneral("wei", "guojia")
	guojia.Stats = map[string]int{"intelligence": 3, "politics": 4}
	applyHeroConfigToGeneral(guojia)
	if guojia.EffectiveStats["intelligence"] != 13 || guojia.EffectiveStats["politics"] != 14 {
		t.Fatalf("expected intelligence/politics 3/4 -> 13/14, got %+v", guojia.EffectiveStats)
	}
	if math.Abs(guojia.Buffs[StatRecruitSpeedBonus]-0.26) > 1e-9 ||
		math.Abs(guojia.Buffs[StatMarchSpeedBonus]-0.26) > 1e-9 ||
		math.Abs(guojia.Buffs[StatProductionBonus]-0.28) > 1e-9 ||
		math.Abs(guojia.Buffs[StatCapacityBonus]-0.28) > 1e-9 {
		t.Fatalf("expected base stats plus passive stats to enter real attributes, buffs=%+v", guojia.Buffs)
	}

	attacker := combat.Army{Units: []combat.Unit{{ID: "weiInfantry", Count: 100, Attack: 10}}}
	defender := combat.Army{Units: []combat.Unit{{ID: "shuInfantry", Count: 100, Attack: 10}}}
	ctx := &general.BeforeBattleContext{Attacker: &attacker, Defender: &defender, AttackerOwnsTrait: true, Scene: "attack"}
	general.Dispatch(ctx, buildActiveTraits(guojia))
	if _, ok := ctx.Triggered["shengui_zhicai"]; ok {
		t.Fatalf("expected passive stat trait absent from battle timeline, outcomes=%+v", ctx.Triggered)
	}
}

// TestGuojiaDualTraitsReviveActualLossesAndMatchReport 验证战败时按逐兵种真实阵亡复活并进入返程兵力。
func TestGuojiaDualTraitsReviveActualLossesAndMatchReport(t *testing.T) {
	setTestFactionsAndGenerals(t, FactionsConfig{
		"wei": {Name: "魏国", Generals: []GeneralInfo{{ID: "guojia", Name: "郭嘉"}}},
		"shu": {Name: "蜀国", Generals: []GeneralInfo{{ID: "liubei", Name: "刘备"}}},
	}, GeneralsConfig{Enabled: true, Heroes: map[string]GeneralHeroConfig{
		"guojia": guojiaTestConfig(0.22),
		"liubei": {ID: "liubei", Name: "刘备", Faction: "shu", Enabled: true},
	}})

	svc, repo, attacker, defender := newPvpTestServiceForGenerals(t, "wei", "guojia", "shu", "liubei")
	attacker.Army = []ArmyUnit{{UnitType: "weiInfantry", Amount: 100}}
	defender.Army = []ArmyUnit{{UnitType: "shuInfantry", Amount: 1000}}
	defender.Buildings = nil
	repo.players[attacker.Player.ID] = attacker
	repo.players[defender.Player.ID] = defender

	started, err := svc.StartPvpAttack(PvpAttackRequest{
		PlayerID: attacker.Player.ID, TargetPlayerID: defender.Player.ID, MarchMode: PvpMarchTypeAttack,
		Troops: map[string]int{"weiInfantry": 100}, GeneralIDs: []string{"guojia"},
	})
	if err != nil {
		t.Fatalf("StartPvpAttack failed: %v", err)
	}
	forcePvpMarchDue(t, repo, started.March.ID)
	battle, err := svc.ResolvePvpMarch(started.March.ID)
	if err != nil {
		t.Fatalf("ResolvePvpMarch failed: %v", err)
	}
	if battle.Result["winner"] != "defender" {
		t.Fatalf("expected Guo Jia side to lose, result=%+v", battle.Result)
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
		assertGuojiaCurrentBattleReport(t, report)
	}

	storedMarch, err := repo.GetPvpMarch(started.March.ID)
	if err != nil || storedMarch.AttackTroops["weiInfantry"] != 22 {
		t.Fatalf("expected 22 revived infantry in return march, march=%+v err=%v", storedMarch, err)
	}
}

// assertGuojiaCurrentBattleReport 核对永久被动只留在快照，真实复活进入战报结果。
func assertGuojiaCurrentBattleReport(t *testing.T, report BattleReport) {
	t.Helper()
	if report.ViewType == ReportViewAttack {
		if report.LostUnits["weiInfantry"] != 100 || report.RevivedUnits["weiInfantry"] != 22 || report.SurvivedUnits["weiInfantry"] != 22 {
			t.Fatalf("expected gross loss 100, revived 22 and final survivors 22, report=%+v", report)
		}
	}
	if len(report.TraitTriggered) != 1 || report.TraitTriggered[0] != "guicai_yice" || len(report.TraitOutcomes) != 1 {
		t.Fatalf("expected only Guicai Yice in battle timeline, report=%+v", report)
	}
	if _, exists := report.TraitOutcomes["shengui_zhicai"]; exists {
		t.Fatalf("expected permanent passive absent from battle outcomes, outcomes=%+v", report.TraitOutcomes)
	}
	outcome := report.TraitOutcomes["guicai_yice"]
	actualLost, actualOK := outcome.Detail["actualLostUnits"].(map[string]int)
	revived, revivedOK := outcome.Detail["revivedUnits"].(map[string]int)
	if !actualOK || !revivedOK || outcome.OwnerGeneralID != "guojia" || outcome.OwnerSide != "attacker" ||
		outcome.Detail["effectRate"] != 0.22 || outcome.Detail["triggerChance"] != 1.0 ||
		actualLost["weiInfantry"] != 100 || revived["weiInfantry"] != 22 || outcome.Detail["totalRevived"] != 22 {
		t.Fatalf("expected exact Guicai revive details, outcome=%+v", outcome)
	}
	if report.Detail == nil || len(report.Detail.Traits) != 1 || report.Detail.Traits[0].TraitID != "guicai_yice" {
		t.Fatalf("expected only Guicai Yice in standard timeline, detail=%+v", report.Detail)
	}
	if len(report.PvpAttackerGenerals) != 1 || !pvpSnapshotHasTrait(report.PvpAttackerGenerals[0], "shengui_zhicai") ||
		!pvpSnapshotHasTrait(report.PvpAttackerGenerals[0], "guicai_yice") ||
		report.PvpAttackerGenerals[0].EffectiveStats["intelligence"]-report.PvpAttackerGenerals[0].Stats["intelligence"] != 10 ||
		report.PvpAttackerGenerals[0].EffectiveStats["politics"]-report.PvpAttackerGenerals[0].Stats["politics"] != 10 {
		t.Fatalf("expected snapshot to retain both traits and permanent stats, snapshots=%+v", report.PvpAttackerGenerals)
	}
}
