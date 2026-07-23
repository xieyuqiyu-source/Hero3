// 本文件验证独立特性、配置化特性及其触发边界。
package traits

import (
	"math/rand"
	"testing"

	"hero3/internal/core/combat"
	"hero3/internal/core/general"
)

// TestTriggerProbabilityBoundaries 验证所有特性共用的概率入口严格处理零、一和越界值。
func TestTriggerProbabilityBoundaries(t *testing.T) {
	for index := 0; index < 1000; index++ {
		if triggered(general.Params{"triggerChance": 0}) || triggered(general.Params{"triggerChance": -1}) || triggeredWithDefault(general.Params{}, 0) {
			t.Fatal("expected zero or negative trigger chance to never trigger")
		}
		if !triggered(general.Params{"triggerChance": 1}) || !triggered(general.Params{"triggerChance": 2}) || !triggeredWithDefault(general.Params{}, 1) {
			t.Fatal("expected one or greater trigger chance to always trigger")
		}
	}
}

// TestZhenMiTraitSchemasExposeCurrentGmParameters 验证 GM 只看到当前攻防比例和独立概率，不再出现旧俘虏参数。
func TestZhenMiTraitSchemasExposeCurrentGmParameters(t *testing.T) {
	tests := []struct {
		traitID string
		rateKey string
	}{
		{traitID: "meiren", rateKey: "attackBonusRate"},
		{traitID: "meihuo_raozhen", rateKey: "enemyDefenseReductionRate"},
	}
	for _, tc := range tests {
		t.Run(tc.traitID, func(t *testing.T) {
			trait, ok := general.Get(tc.traitID)
			if !ok {
				t.Fatalf("expected trait %s registered", tc.traitID)
			}
			fields := map[string]general.ParamField{}
			for _, field := range trait.ParamSchema() {
				fields[field.Key] = field
			}
			if fields[tc.rateKey].Default != 0.25 || fields["triggerChance"].Default != 0.5 {
				t.Fatalf("expected rate 25%% and chance 50%% defaults, fields=%+v", fields)
			}
			for _, legacyKey := range []string{"captureRate", "captureMax", "maxCapturePerUnit"} {
				if _, exists := fields[legacyKey]; exists {
					t.Fatalf("expected legacy field %s removed, fields=%+v", legacyKey, fields)
				}
			}
		})
	}
}

// TestSimaYiTraitSchemasExposeCurrentGmParameters 验证 GM 只看到司马懿当前概率和效果参数。
func TestSimaYiTraitSchemasExposeCurrentGmParameters(t *testing.T) {
	tests := []struct {
		traitID string
		rateKey string
	}{
		{traitID: "yibing_touxi", rateKey: "effectRate"},
		{traitID: "mouding_houfa", rateKey: "defenseBonusRate"},
	}
	for _, tc := range tests {
		t.Run(tc.traitID, func(t *testing.T) {
			trait, ok := general.Get(tc.traitID)
			if !ok {
				t.Fatalf("expected trait %s registered", tc.traitID)
			}
			fields := map[string]general.ParamField{}
			for _, field := range trait.ParamSchema() {
				fields[field.Key] = field
			}
			if fields[tc.rateKey].Default != 0.35 || fields["triggerChance"].Default != 0.35 {
				t.Fatalf("expected rate and chance 35%% defaults, fields=%+v", fields)
			}
			if tc.traitID == "yibing_touxi" {
				for _, legacyKey := range []string{"maxAffectedRate", "maxAffectedCount"} {
					if _, exists := fields[legacyKey]; exists {
						t.Fatalf("expected legacy cap %s removed from Yibing GM schema, fields=%+v", legacyKey, fields)
					}
				}
			}
		})
	}
}

// TestCurrentShuCombatSchemasHideUnusedCaps 验证本批蜀将概率攻防特性不向 GM 暴露未消费的通用上限。
func TestCurrentShuCombatSchemasHideUnusedCaps(t *testing.T) {
	for _, traitID := range []string{"shuiyan_qijun", "wusheng_pojun", "zhenhe_quanjun", "wanren_nuhou", "baibu_chuanyang"} {
		t.Run(traitID, func(t *testing.T) {
			trait, ok := general.Get(traitID)
			if !ok {
				t.Fatalf("expected trait %s registered", traitID)
			}
			fields := map[string]general.ParamField{}
			for _, field := range trait.ParamSchema() {
				fields[field.Key] = field
			}
			if _, exists := fields["triggerChance"]; !exists {
				t.Fatalf("expected %s to expose triggerChance, fields=%+v", traitID, fields)
			}
			for _, unusedKey := range []string{"maxAffectedRate", "maxAffectedCount"} {
				if _, exists := fields[unusedKey]; exists {
					t.Fatalf("expected %s to hide unused %s, fields=%+v", traitID, unusedKey, fields)
				}
			}
		})
	}
}

// TestIndependentTraitsRespectZeroTriggerChance 验证独立实现和配置化特性不会在零概率时改状态或写触发结果。
func TestIndependentTraitsRespectZeroTriggerChance(t *testing.T) {
	t.Run("美人心计", func(t *testing.T) {
		attacker := combat.Army{Units: []combat.Unit{{ID: "infantry", Count: 100, Attack: 10}}}
		defender := combat.Army{Units: []combat.Unit{{ID: "infantry", Count: 100, InfantryDefense: 10, CavalryDefense: 8}}}
		ctx := &general.BeforeBattleContext{Attacker: &attacker, Defender: &defender}
		general.Dispatch(ctx, []general.ActiveTrait{{TraitID: "meiren", OwnerSide: "attacker", Params: general.Params{"attackBonusRate": 0.25, "triggerChance": 0}}})
		if attacker.Units[0].Attack != 10 || len(ctx.Triggered) != 0 {
			t.Fatalf("expected zero chance Meiren Xinji to leave attack untouched, ctx=%+v attacker=%+v", ctx, attacker)
		}
	})
	t.Run("火攻", func(t *testing.T) {
		result := combat.CombatResult{DefenderLosses: []combat.UnitLoss{{ID: "infantry", Count: 100, Losses: 10}}}
		attacker := combat.Army{}
		defender := combat.Army{Units: []combat.Unit{{ID: "infantry", Count: 100}}}
		ctx := &general.AfterCombatResolveContext{Result: &result, Attacker: &attacker, Defender: &defender, AttackerOwnsTrait: true}
		general.Dispatch(ctx, []general.ActiveTrait{{TraitID: "huogong", OwnerSide: "attacker", Params: general.Params{"effectRate": 0.25, "triggerChance": 0}}})
		if result.DefenderLosses[0].Losses != 10 || len(ctx.Triggered) != 0 {
			t.Fatalf("expected zero chance Huogong to leave losses untouched, result=%+v outcomes=%+v", result, ctx.Triggered)
		}
	})
	t.Run("仁主守护", func(t *testing.T) {
		army := map[string]int{}
		ctx := &general.AfterBattleContext{PlayerArmy: army, PlayerLosses: map[string]int{"infantry": 100}}
		general.Dispatch(ctx, []general.ActiveTrait{{TraitID: "renzhu_shouhu", Params: general.Params{"effectRate": 0.35, "triggerChance": 0}}})
		if len(army) != 0 || len(ctx.Revived) != 0 || len(ctx.Triggered) != 0 {
			t.Fatalf("expected zero chance Renzhu Shouhu to leave army untouched, army=%+v ctx=%+v", army, ctx)
		}
	})
}

// TestAllConfigurableRandomCombatTraitsRespectZeroTriggerChance 验证全部配置化随机战斗特性未命中时不改上下文也不写结果。
func TestAllConfigurableRandomCombatTraitsRespectZeroTriggerChance(t *testing.T) {
	beforeBattleTraits := []struct {
		traitID   string
		ownerSide string
	}{
		{traitID: "yibing_touxi", ownerSide: "attacker"},
		{traitID: "huchi_chongzhen", ownerSide: "attacker"},
		{traitID: "weizhen_zhenhe", ownerSide: "attacker"},
		{traitID: "shuiyan_qijun", ownerSide: "attacker"},
		{traitID: "zhenhe_quanjun", ownerSide: "attacker"},
		{traitID: "baibu_chuanyang", ownerSide: "attacker"},
		{traitID: "qibing_raohou", ownerSide: "attacker"},
		{traitID: "jiangdong_gushou", ownerSide: "defender"},
	}
	for _, tc := range beforeBattleTraits {
		t.Run(tc.traitID, func(t *testing.T) {
			attacker := combat.Army{Units: []combat.Unit{{ID: "infantry", Category: "infantry", Count: 100, Attack: 10, InfantryDefense: 10, CavalryDefense: 8}}}
			defender := combat.Army{Units: []combat.Unit{{ID: "infantry", Category: "infantry", Count: 100, Attack: 10, InfantryDefense: 10, CavalryDefense: 8}}}
			ctx := &general.BeforeBattleContext{Attacker: &attacker, Defender: &defender, Scene: "plunder"}
			general.Dispatch(ctx, []general.ActiveTrait{{
				TraitID: tc.traitID, OwnerSide: tc.ownerSide, TargetUnitType: "infantry",
				Params: general.Params{
					"triggerChance": 0, "effectRate": 1, "enemyDefenseReductionRate": 0.5,
					"defenseBonusRate": 0.5, "maxAffectedRate": 1,
				},
			}})
			if attacker.Units[0].Count != 100 || attacker.Units[0].Attack != 10 || attacker.Units[0].InfantryDefense != 10 || attacker.Units[0].CavalryDefense != 8 ||
				defender.Units[0].Count != 100 || defender.Units[0].Attack != 10 || defender.Units[0].InfantryDefense != 10 || defender.Units[0].CavalryDefense != 8 ||
				len(ctx.AttackerPreBattleLosses) != 0 || len(ctx.DefenderPreBattleLosses) != 0 || len(ctx.AttackerSuppressedUnits) != 0 || len(ctx.DefenderSuppressedUnits) != 0 || len(ctx.Triggered) != 0 {
				t.Fatalf("expected zero chance %s to leave before-battle context untouched, ctx=%+v attacker=%+v defender=%+v", tc.traitID, ctx, attacker, defender)
			}
		})
	}

	for _, tc := range []struct {
		traitID   string
		ownerSide string
	}{
		{traitID: "longdan_jiuyuan", ownerSide: "defender"},
		{traitID: "xiliang_tuji", ownerSide: "attacker"},
		{traitID: "xiaobawang_zhuiji", ownerSide: "attacker"},
		{traitID: "huoshao_lianying", ownerSide: "attacker"},
		{traitID: "kurouji", ownerSide: "attacker"},
	} {
		t.Run(tc.traitID, func(t *testing.T) {
			attacker := combat.Army{Units: []combat.Unit{{ID: "infantry", Category: "infantry", Count: 100}}}
			defender := combat.Army{Units: []combat.Unit{{ID: "infantry", Category: "infantry", Count: 100}}}
			result := combat.CombatResult{
				Winner:         "attacker",
				AttackerLosses: []combat.UnitLoss{{ID: "infantry", Count: 100, Losses: 10}},
				DefenderLosses: []combat.UnitLoss{{ID: "infantry", Count: 100, Losses: 20}},
			}
			ctx := &general.AfterCombatResolveContext{Result: &result, Attacker: &attacker, Defender: &defender, Scene: "plunder"}
			general.Dispatch(ctx, []general.ActiveTrait{{
				TraitID: tc.traitID, OwnerSide: tc.ownerSide, TargetUnitType: "infantry",
				Params: general.Params{
					"triggerChance": 0, "effectRate": 1, "lossReductionRate": 1,
					"disableTraitCount": 1, "maxAffectedRate": 1,
				},
			}})
			if result.AttackerLosses[0].Losses != 10 || result.DefenderLosses[0].Losses != 20 || len(ctx.DisabledTraitSide) != 0 || len(ctx.DisabledTraitOutcomeKeys) != 0 || len(ctx.Triggered) != 0 {
				t.Fatalf("expected zero chance %s to leave combat result untouched, result=%+v ctx=%+v", tc.traitID, result, ctx)
			}
		})
	}

	t.Run("huzhu_xuezhan", func(t *testing.T) {
		defender := combat.Army{Units: []combat.Unit{{ID: "jinWeiSoldier", Category: "infantry", Count: 100, InfantryDefense: 13, CavalryDefense: 7}}}
		ctx := &general.BeforeBattleContext{Attacker: &combat.Army{}, Defender: &defender}
		general.Dispatch(ctx, []general.ActiveTrait{{
			TraitID: "huzhu_xuezhan", OwnerSide: "defender", AllowedSides: []string{"defender", "reinforcement"}, TargetUnitType: "jinWeiSoldier",
			Params: general.Params{"triggerChance": 0, "generalDefenseFlat": 20},
		}})
		if defender.Units[0].InfantryDefense != 13 || defender.Units[0].CavalryDefense != 7 || len(ctx.Triggered) != 0 {
			t.Fatalf("expected zero chance Huzhu Xuezhan to leave defense untouched, defender=%+v ctx=%+v", defender, ctx)
		}
	})
}

// TestZhenMiBeforeBattleTraitsCanTriggerTogether 验证甄宓两项独立概率特性可在同一战前阶段同时修改真实攻防。
func TestZhenMiBeforeBattleTraitsCanTriggerTogether(t *testing.T) {
	attacker := combat.Army{Units: []combat.Unit{
		{ID: "infantry", Category: "infantry", Count: 100, Attack: 10},
		{ID: "cavalry", Category: "cavalry", Count: 100, Attack: 12},
	}}
	defender := combat.Army{Units: []combat.Unit{
		{ID: "guard", Category: "infantry", Count: 100, InfantryDefense: 10, CavalryDefense: 8},
	}}
	ctx := &general.BeforeBattleContext{Attacker: &attacker, Defender: &defender}
	general.Dispatch(ctx, []general.ActiveTrait{
		{TraitID: "meiren", OwnerSide: "attacker", AllowedSides: []string{"attacker"}, Params: general.Params{"attackBonusRate": 0.25, "triggerChance": 1}},
		{TraitID: "meihuo_raozhen", OwnerSide: "attacker", AllowedSides: []string{"attacker"}, Params: general.Params{"enemyDefenseReductionRate": 0.25, "triggerChance": 1}},
	})
	if attacker.Units[0].Attack != 13 || attacker.Units[1].Attack != 15 {
		t.Fatalf("expected all attacks 10/12 -> 13/15, got %+v", attacker.Units)
	}
	if defender.Units[0].InfantryDefense != 8 || defender.Units[0].CavalryDefense != 6 {
		t.Fatalf("expected enemy defenses 10/8 -> 8/6, got %+v", defender.Units)
	}
	attackChanged := ctx.Triggered["meiren"].Detail["attackModifiedUnits"].(map[string]int)
	infantryDefenseChanged := ctx.Triggered["meihuo_raozhen"].Detail["infantryDefenseModifiedUnits"].(map[string]int)
	cavalryDefenseChanged := ctx.Triggered["meihuo_raozhen"].Detail["cavalryDefenseModifiedUnits"].(map[string]int)
	if attackChanged["infantry"] != 3 || attackChanged["cavalry"] != 3 || infantryDefenseChanged["guard"] != -2 || cavalryDefenseChanged["guard"] != -2 {
		t.Fatalf("expected exact Zhen Mi stat deltas, outcomes=%+v", ctx.Triggered)
	}
	if ctx.Triggered["meiren"].Name != "美人心计" || ctx.Triggered["meiren"].Detail["attackBonusRate"] != 0.25 || ctx.Triggered["meihuo_raozhen"].Detail["enemyDefenseReductionRate"] != 0.25 {
		t.Fatalf("expected current names and design rates, outcomes=%+v", ctx.Triggered)
	}
}

// TestZhenMiBeforeBattleTraitsRespectAttackDirection 验证两项特性在防守和增援方向都没有资格触发。
func TestZhenMiBeforeBattleTraitsRespectAttackDirection(t *testing.T) {
	for _, side := range []string{"defender", "reinforcement"} {
		t.Run(side, func(t *testing.T) {
			attacker := combat.Army{Units: []combat.Unit{{ID: "attacker", Count: 100, Attack: 10}}}
			defender := combat.Army{Units: []combat.Unit{{ID: "defender", Count: 100, Attack: 10, InfantryDefense: 10, CavalryDefense: 8}}}
			ctx := &general.BeforeBattleContext{Attacker: &attacker, Defender: &defender, Scene: "reinforcement_defense"}
			general.Dispatch(ctx, []general.ActiveTrait{
				{TraitID: "meiren", OwnerSide: side, AllowedSides: []string{"attacker"}, Params: general.Params{"attackBonusRate": 0.25, "triggerChance": 1}},
				{TraitID: "meihuo_raozhen", OwnerSide: side, AllowedSides: []string{"attacker"}, Params: general.Params{"enemyDefenseReductionRate": 0.25, "triggerChance": 1}},
			})
			if attacker.Units[0].Attack != 10 || defender.Units[0].Attack != 10 || defender.Units[0].InfantryDefense != 10 || len(ctx.Triggered) != 0 {
				t.Fatalf("expected side %s to keep base stats without outcomes, ctx=%+v", side, ctx)
			}
		})
	}
}

// 火攻：必触发时给敌方加伤害
func TestHuogong_AddsDamageToDefender(t *testing.T) {
	rand.Seed(1)
	result := &combat.CombatResult{
		DefenderLosses: []combat.UnitLoss{
			{ID: "infantry", Count: 100, Losses: 30},
			{ID: "cavalry", Count: 50, Losses: 10},
		},
	}
	ctx := &general.AfterCombatResolveContext{
		Result:            result,
		AttackerOwnsTrait: true,
	}
	general.Dispatch(ctx, []general.ActiveTrait{
		{TraitID: "huogong", Params: general.Params{"damagePercent": 0.2, "triggerChance": 1.0}},
	})
	// 100 × 0.2 = 20 额外损失，30 + 20 = 50
	if result.DefenderLosses[0].Losses != 50 {
		t.Errorf("expected 50 infantry losses, got %d", result.DefenderLosses[0].Losses)
	}
	// 50 × 0.2 = 10 额外，10 + 10 = 20
	if result.DefenderLosses[1].Losses != 20 {
		t.Errorf("expected 20 cavalry losses, got %d", result.DefenderLosses[1].Losses)
	}
	outcome := ctx.Triggered["huogong"]
	byUnit, ok := outcome.Detail["targetExtraLosses"].(map[string]int)
	if !ok || byUnit["infantry"] != 20 || byUnit["cavalry"] != 10 || outcome.Detail["extraDamage"] != 30 || outcome.Detail["triggerChance"] != 1.0 {
		t.Fatalf("expected fire report to record per-unit 20/10 and total 30, outcome=%+v", outcome)
	}
}

// 火攻：损失不超过总数
func TestHuogong_CapsAtTotal(t *testing.T) {
	result := &combat.CombatResult{
		DefenderLosses: []combat.UnitLoss{
			{ID: "infantry", Count: 100, Losses: 95},
		},
	}
	ctx := &general.AfterCombatResolveContext{Result: result, AttackerOwnsTrait: true}
	general.Dispatch(ctx, []general.ActiveTrait{
		{TraitID: "huogong", Params: general.Params{"damagePercent": 0.2, "triggerChance": 1.0}},
	})
	// 95 + 20 = 115 超过 100，应该被限制到 100
	if result.DefenderLosses[0].Losses != 100 {
		t.Errorf("expected losses capped at 100, got %d", result.DefenderLosses[0].Losses)
	}
}

// TestHuogongDefenderDoesNothing 验证周瑜火攻不会在防御方携带时误触发。
func TestHuogongDefenderDoesNothing(t *testing.T) {
	result := &combat.CombatResult{
		AttackerLosses: []combat.UnitLoss{{ID: "infantry", Count: 100, Losses: 10}},
		DefenderLosses: []combat.UnitLoss{{ID: "cavalry", Count: 100, Losses: 10}},
	}
	ctx := &general.AfterCombatResolveContext{Result: result, DefenderOwnsTrait: true, Scene: "attack"}
	general.Dispatch(ctx, []general.ActiveTrait{{
		TraitID: "huogong", OwnerSide: "defender", AllowedSides: []string{"attacker"},
		Params: general.Params{"damagePercent": 0.2, "triggerChance": 1},
	}})

	if result.AttackerLosses[0].Losses != 10 || result.DefenderLosses[0].Losses != 10 {
		t.Fatalf("expected defender fire attack blocked, got %+v", result)
	}
	if len(ctx.Triggered) != 0 {
		t.Fatalf("expected no fire attack outcome on defense, got %+v", ctx.Triggered)
	}
}

// 仁主守护：概率命中后按真实阵亡复活。
func TestRenzhuShouhuRevivesLosses(t *testing.T) {
	rand.Seed(1)
	playerArmy := map[string]int{"infantry": 80, "cavalry": 30}
	playerLosses := map[string]int{"infantry": 20, "cavalry": 10}

	ctx := &general.AfterBattleContext{
		PlayerArmy:   playerArmy,
		PlayerLosses: playerLosses,
		IsAttacker:   true,
		Won:          true,
	}
	general.Dispatch(ctx, []general.ActiveTrait{
		{TraitID: "renzhu_shouhu", Params: general.Params{"effectRate": 0.35, "triggerChance": 1.0}},
	})
	if playerArmy["infantry"] != 87 {
		t.Errorf("expected 87 infantry after revive, got %d", playerArmy["infantry"])
	}
	if playerArmy["cavalry"] != 33 {
		t.Errorf("expected 33 cavalry after revive, got %d", playerArmy["cavalry"])
	}
	if ctx.Revived["infantry"] != 7 || ctx.Revived["cavalry"] != 3 {
		t.Errorf("expected revived map {infantry:7, cavalry:3}, got %v", ctx.Revived)
	}
	if outcome := ctx.Triggered["renzhu_shouhu"]; outcome.Detail["effectRate"] != 0.35 || outcome.Detail["triggerChance"] != 1.0 {
		t.Fatalf("expected Renzhu outcome to include rate 0.35 and chance 1, got %+v", outcome)
	}
}

// 仁主守护：战败时同样按真实阵亡复活。
func TestRenzhuShouhuTriggersOnLoss(t *testing.T) {
	rand.Seed(1)
	playerArmy := map[string]int{"infantry": 0}
	playerLosses := map[string]int{"infantry": 100}

	ctx := &general.AfterBattleContext{
		PlayerArmy: playerArmy, PlayerLosses: playerLosses,
		IsAttacker: true, Won: false, // 失败
	}
	general.Dispatch(ctx, []general.ActiveTrait{
		{TraitID: "renzhu_shouhu", Params: general.Params{"effectRate": 0.35, "triggerChance": 1.0}},
	})
	if playerArmy["infantry"] != 35 {
		t.Errorf("expected 35 infantry revived on loss, got %d", playerArmy["infantry"])
	}
}

// 配置化追击：防守方触发时应追加进攻方损失。
func TestConfigurableExtraDamage_DefenderDamagesAttacker(t *testing.T) {
	result := &combat.CombatResult{
		AttackerLosses: []combat.UnitLoss{{ID: "infantry", Count: 100, Losses: 10}},
		DefenderLosses: []combat.UnitLoss{{ID: "cavalry", Count: 100, Losses: 10}},
	}
	ctx := &general.AfterCombatResolveContext{
		Result:            result,
		AttackerOwnsTrait: true,
		DefenderOwnsTrait: true,
	}

	general.Dispatch(ctx, []general.ActiveTrait{
		{
			TraitID:   "xiaobawang_zhuiji",
			OwnerSide: "defender",
			Params: general.Params{
				"effectRate":    0.2,
				"triggerChance": 1,
			},
		},
	})

	if result.AttackerLosses[0].Losses != 30 {
		t.Fatalf("expected attacker losses increased to 30, got %d", result.AttackerLosses[0].Losses)
	}
	if result.DefenderLosses[0].Losses != 10 {
		t.Fatalf("expected defender losses unchanged, got %d", result.DefenderLosses[0].Losses)
	}
	if ctx.Triggered["xiaobawang_zhuiji"].OwnerSide != "defender" {
		t.Fatalf("expected defender outcome, got %+v", ctx.Triggered["xiaobawang_zhuiji"])
	}
}

// 配置化压制：核心事件管线应跳过被压制方的后续特性。
func TestConfigurableDisableTraits_SkipsEnemyTraitInBus(t *testing.T) {
	result := &combat.CombatResult{
		AttackerLosses: []combat.UnitLoss{{ID: "infantry", Count: 100, Losses: 10}},
		DefenderLosses: []combat.UnitLoss{{ID: "cavalry", Count: 100, Losses: 10}},
	}
	ctx := &general.AfterCombatResolveContext{
		Result:            result,
		AttackerOwnsTrait: true,
		DefenderOwnsTrait: true,
	}

	general.Dispatch(ctx, []general.ActiveTrait{
		{
			TraitID:   "kurouji",
			OwnerSide: "attacker",
			Params: general.Params{
				"disableTraitCount": 1,
				"triggerChance":     1,
			},
		},
		{
			TraitID:   "xiaobawang_zhuiji",
			OwnerSide: "defender",
			Params: general.Params{
				"effectRate":    0.2,
				"triggerChance": 1,
			},
		},
	})

	if result.AttackerLosses[0].Losses != 10 {
		t.Fatalf("expected defender trait skipped, got attacker losses %d", result.AttackerLosses[0].Losses)
	}
	if outcome, ok := ctx.Triggered["kurouji"]; !ok || outcome.Detail["disableTraitCount"] != 1 || outcome.Detail["disabledTraitCount"] != 1 {
		t.Fatalf("expected kurouji outcome, got %v", ctx.Triggered)
	}
	if _, ok := ctx.Triggered["xiaobawang_zhuiji"]; ok {
		t.Fatalf("expected defender trait suppressed, got %v", ctx.Triggered)
	}
}

// TestDisableTraitsAtSamePriorityResolveSimultaneously 验证双方压制同时生效且只阻止后续特性。
func TestDisableTraitsAtSamePriorityResolveSimultaneously(t *testing.T) {
	result := &combat.CombatResult{
		AttackerLosses: []combat.UnitLoss{{ID: "infantry", Count: 100, Losses: 10}},
		DefenderLosses: []combat.UnitLoss{{ID: "cavalry", Count: 100, Losses: 10}},
	}
	ctx := &general.AfterCombatResolveContext{
		Result: result, AttackerOwnsTrait: true, DefenderOwnsTrait: true,
	}

	general.Dispatch(ctx, []general.ActiveTrait{
		{TraitID: "kurouji", OwnerSide: "attacker", Params: general.Params{"disableTraitCount": 1, "triggerChance": 1}},
		{TraitID: "kurouji", OwnerSide: "defender", Params: general.Params{"disableTraitCount": 1, "triggerChance": 1}},
		{TraitID: "kurou_fanji", OwnerSide: "attacker", Params: general.Params{"effectRate": 0.1, "triggerChance": 1}},
		{TraitID: "lianying_zengshang", OwnerSide: "defender", TargetUnitType: "infantry", Params: general.Params{"effectRate": 0.1, "triggerChance": 1}},
	})

	if result.AttackerLosses[0].Losses != 10 || result.DefenderLosses[0].Losses != 10 {
		t.Fatalf("expected both lower-priority damage traits suppressed, got %+v", result)
	}
	if _, ok := ctx.Triggered["kurouji"]; !ok {
		t.Fatalf("expected attacker suppression outcome, got %+v", ctx.Triggered)
	}
	if _, ok := ctx.Triggered["kurou_fanji"]; ok {
		t.Fatalf("expected attacker damage trait suppressed, got %+v", ctx.Triggered)
	}
	if _, ok := ctx.Triggered["lianying_zengshang"]; ok {
		t.Fatalf("expected defender damage trait suppressed, got %+v", ctx.Triggered)
	}
	suppressionOutcomes := 0
	for _, outcome := range ctx.Triggered {
		if outcome.TraitID == "kurouji" && outcome.Detail["disabledTraitCount"] == 1 {
			suppressionOutcomes++
		}
	}
	if suppressionOutcomes != 2 {
		t.Fatalf("expected both Huang Gai suppression outcomes to record one real interception, got %+v", ctx.Triggered)
	}
}

// TestDisableTraitsWithoutEnemyFollowupReportsZeroActual 验证没有敌方后续特性时只记录设计压制数，不伪报实际压制。
func TestDisableTraitsWithoutEnemyFollowupReportsZeroActual(t *testing.T) {
	ctx := &general.AfterCombatResolveContext{
		Result: &combat.CombatResult{}, AttackerOwnsTrait: true, DefenderOwnsTrait: true,
	}
	general.Dispatch(ctx, []general.ActiveTrait{
		{TraitID: "kurouji", OwnerSide: "attacker", Params: general.Params{"disableTraitCount": 1, "triggerChance": 1}},
	})
	outcome := ctx.Triggered["kurouji"]
	if outcome.Detail["disableTraitCount"] != 1 || outcome.Detail["disabledTraitCount"] != 0 {
		t.Fatalf("expected design count 1 and actual count 0 without enemy follow-up, got %+v", outcome)
	}
}

// TestSameTraitFromBothSidesKeepsBothOutcomes 验证双方同特性同时触发时核心结果不会互相覆盖。
func TestSameTraitFromBothSidesKeepsBothOutcomes(t *testing.T) {
	result := &combat.CombatResult{
		Winner:         "attacker",
		AttackerLosses: []combat.UnitLoss{{ID: "infantry", Count: 100, Losses: 10}},
		DefenderLosses: []combat.UnitLoss{{ID: "cavalry", Count: 100, Losses: 10}},
	}
	ctx := &general.AfterCombatResolveContext{Result: result, AttackerOwnsTrait: true, DefenderOwnsTrait: true, Scene: "attack"}
	params := general.Params{"effectRate": 0.2, "triggerChance": 1}
	general.Dispatch(ctx, []general.ActiveTrait{
		{TraitID: "xiaobawang_zhuiji", OwnerSide: "attacker", OwnerGeneralID: "sunce_a", Params: params},
		{TraitID: "xiaobawang_zhuiji", OwnerSide: "defender", OwnerGeneralID: "sunce_d", Params: params},
	})

	if result.AttackerLosses[0].Losses != 30 || result.DefenderLosses[0].Losses != 30 {
		t.Fatalf("expected both sides to receive pursuit losses, got %+v", result)
	}
	if len(ctx.Triggered) != 2 {
		t.Fatalf("expected two independent same-trait outcomes, got %+v", ctx.Triggered)
	}
}

// TestMoudingHoufaRaisesOwnedArmyDefense 验证谋定后发提升所属部队两类防御，并记录实际整数变化。
func TestMoudingHoufaRaisesOwnedArmyDefense(t *testing.T) {
	attacker := combat.Army{Units: []combat.Unit{{ID: "infantry", Attack: 100, Count: 100}}}
	defender := combat.Army{Units: []combat.Unit{{ID: "cavalry", InfantryDefense: 10, CavalryDefense: 8, Count: 100}}}
	ctx := &general.BeforeBattleContext{Attacker: &attacker, Defender: &defender, DefenderOwnsTrait: true, Scene: "attack"}

	general.Dispatch(ctx, []general.ActiveTrait{{
		TraitID: "mouding_houfa", OwnerSide: "defender",
		Params: general.Params{"defenseBonusRate": 0.35, "triggerChance": 1},
	}})

	if defender.Units[0].InfantryDefense != 14 || defender.Units[0].CavalryDefense != 11 {
		t.Fatalf("expected owned defense 10/8 -> 14/11, got %+v", defender.Units[0])
	}
	outcome, ok := ctx.Triggered["mouding_houfa"]
	infantry, infantryOK := outcome.Detail["infantryDefenseModifiedUnits"].(map[string]int)
	cavalry, cavalryOK := outcome.Detail["cavalryDefenseModifiedUnits"].(map[string]int)
	if !ok || !infantryOK || !cavalryOK || infantry["cavalry"] != 4 || cavalry["cavalry"] != 3 || outcome.Detail["defenseBonusRate"] != 0.35 {
		t.Fatalf("expected exact mouding_houfa defense outcome, got %+v", ctx.Triggered)
	}
}

// TestYibingTouxiIgnoresLegacyCaps 验证疑兵真实伤亡只由 GM 配置的伤亡比例决定。
func TestYibingTouxiIgnoresLegacyCaps(t *testing.T) {
	attacker := combat.Army{Units: []combat.Unit{{ID: "infantry", Count: 1000, Attack: 10}}}
	defender := combat.Army{Units: []combat.Unit{{ID: "infantry", Count: 1000, InfantryDefense: 10, CavalryDefense: 8}}}
	ctx := &general.BeforeBattleContext{Attacker: &attacker, Defender: &defender, AttackerOwnsTrait: true, Scene: "attack"}

	general.Dispatch(ctx, []general.ActiveTrait{{
		TraitID: "yibing_touxi", OwnerSide: "attacker", OwnerGeneralID: "simayi",
		Params: general.Params{"effectRate": 0.35, "triggerChance": 1, "maxAffectedRate": 0.1, "maxAffectedCount": 20},
	}})

	if defender.Units[0].Count != 650 {
		t.Fatalf("expected legacy caps ignored and exact 35%% real losses applied, got %+v", defender.Units[0])
	}
	outcome := ctx.Triggered["yibing_touxi"]
	affected, ok := outcome.Detail["preBattleAffected"].(map[string]int)
	if !ok || affected["infantry"] != 350 || outcome.Detail["effectRate"] != 0.35 {
		t.Fatalf("expected exact Yibing real-loss outcome, got %+v", outcome)
	}
	if _, exists := outcome.Detail["maxAffectedRate"]; exists {
		t.Fatalf("expected legacy maxAffectedRate absent from current report, got %+v", outcome.Detail)
	}
}

// TestTargetUnitDamageMatchesRealUnitCategory 验证分类目标可以命中真实兵种 ID，并且不会伤害其他分类。
func TestTargetUnitDamageMatchesRealUnitCategory(t *testing.T) {
	t.Run("进攻马超只追加骑兵损失", func(t *testing.T) {
		attacker := combat.Army{Units: []combat.Unit{{ID: "xiLiangCavalry", Category: "cavalry", Count: 100}}}
		defender := combat.Army{Units: []combat.Unit{
			{ID: "overlordRider", Category: "cavalry", Count: 100},
			{ID: "shadowGuard", Category: "infantry", Count: 100},
		}}
		result := &combat.CombatResult{DefenderLosses: []combat.UnitLoss{
			{ID: "overlordRider", Count: 100, Losses: 10},
			{ID: "shadowGuard", Count: 100, Losses: 10},
		}}
		ctx := &general.AfterCombatResolveContext{Result: result, Attacker: &attacker, Defender: &defender, AttackerOwnsTrait: true, Scene: "attack"}
		general.Dispatch(ctx, []general.ActiveTrait{{
			TraitID: "xiliang_tuji", OwnerSide: "attacker", TargetUnitType: "cavalry",
			Params: general.Params{"effectRate": 0.12, "triggerChance": 1},
		}})

		if result.DefenderLosses[0].Losses != 22 || result.DefenderLosses[1].Losses != 10 {
			t.Fatalf("expected only real cavalry ID to receive 12 extra losses, got %+v", result.DefenderLosses)
		}
		changed, ok := ctx.Triggered["xiliang_tuji"].Detail["targetExtraLosses"].(map[string]int)
		if !ok || changed["overlordRider"] != 12 || len(changed) != 1 {
			t.Fatalf("expected report to contain only overlordRider +12, got %+v", ctx.Triggered)
		}
	})

	t.Run("防守陆逊只追加步兵损失", func(t *testing.T) {
		attacker := combat.Army{Units: []combat.Unit{
			{ID: "qingZhouArmy", Category: "infantry", Count: 100},
			{ID: "huBaoQi", Category: "cavalry", Count: 100},
		}}
		defender := combat.Army{Units: []combat.Unit{{ID: "shadowGuard", Category: "infantry", Count: 100}}}
		result := &combat.CombatResult{AttackerLosses: []combat.UnitLoss{
			{ID: "qingZhouArmy", Count: 100, Losses: 10},
			{ID: "huBaoQi", Count: 100, Losses: 10},
		}}
		ctx := &general.AfterCombatResolveContext{Result: result, Attacker: &attacker, Defender: &defender, DefenderOwnsTrait: true, Scene: "attack"}
		general.Dispatch(ctx, []general.ActiveTrait{{
			TraitID: "huoshao_lianying", OwnerSide: "defender", TargetUnitType: "infantry",
			Params: general.Params{"effectRate": 1, "triggerChance": 1},
		}})

		if result.AttackerLosses[0].Losses != 100 || result.AttackerLosses[1].Losses != 10 {
			t.Fatalf("expected only real infantry ID to be fully damaged, got %+v", result.AttackerLosses)
		}
	})
}

// TestHuchiChongzhenReducesBothDefensesAndReportsActualDeltas 验证虎痴冲阵同时降低两类防御并记录实际整数差值。
func TestHuchiChongzhenReducesBothDefensesAndReportsActualDeltas(t *testing.T) {
	attacker := combat.Army{Units: []combat.Unit{{ID: "huWei", Category: "infantry", Count: 100, Attack: 14}}}
	defender := combat.Army{Units: []combat.Unit{{ID: "overlordRider", Category: "cavalry", Count: 100, InfantryDefense: 101, CavalryDefense: 101}}}
	ctx := &general.BeforeBattleContext{Attacker: &attacker, Defender: &defender, AttackerOwnsTrait: true, Scene: "attack"}
	general.Dispatch(ctx, []general.ActiveTrait{{TraitID: "huchi_chongzhen", OwnerSide: "attacker", Params: general.Params{"enemyDefenseReductionRate": 0.3, "triggerChance": 1}}})

	if defender.Units[0].InfantryDefense != 71 || defender.Units[0].CavalryDefense != 71 {
		t.Fatalf("expected rounded defense 101 -> 71, got %+v", defender.Units[0])
	}
	huchiInfantry := ctx.Triggered["huchi_chongzhen"].Detail["infantryDefenseModifiedUnits"].(map[string]int)
	huchiCavalry := ctx.Triggered["huchi_chongzhen"].Detail["cavalryDefenseModifiedUnits"].(map[string]int)
	if huchiInfantry["overlordRider"] != -30 || huchiCavalry["overlordRider"] != -30 || ctx.Triggered["huchi_chongzhen"].Detail["enemyDefenseReductionRate"] != 0.3 {
		t.Fatalf("expected Huchi to report design 30%% and actual -30/-30, got %+v", ctx.Triggered)
	}
}

// TestPlunderTraitsModifyRewardsAndRecordDelta 验证攻守双方掠夺特性修改同一份最终收益并写入明细。
func TestPlunderTraitsModifyRewardsAndRecordDelta(t *testing.T) {
	ctx := &general.PlunderResolveContext{Rewards: map[string]int{"wood": 100}, Scene: "plunder"}
	general.Dispatch(ctx, []general.ActiveTrait{
		{TraitID: "jinfan_jielue", TraitType: general.TraitTypeSpecial, OwnerSide: "attacker", AllowedSides: []string{"attacker"}, AllowedScenes: []string{"plunder"}, Params: general.Params{"plunderBonusRate": 0.2, "triggerChance": 1}},
		{TraitID: "jiangdong_haoling", TraitType: general.TraitTypeSpecial, OwnerSide: "defender", AllowedSides: []string{"defender"}, AllowedScenes: []string{"plunder"}, Params: general.Params{"plunderBonusRate": -0.2, "triggerChance": 1}},
	})
	if ctx.Rewards["wood"] != 96 || len(ctx.Triggered) != 2 {
		t.Fatalf("expected 100 -> 120 -> 96 and two outcomes, rewards=%+v outcomes=%+v", ctx.Rewards, ctx.Triggered)
	}
}

// TestAllMarchSpeedTraitsModifyFinalDuration 验证全部行军模板特性都订阅正确事件并产生确定时长。
func TestAllMarchSpeedTraitsModifyFinalDuration(t *testing.T) {
	for _, tc := range []struct {
		traitID string
		rate    float64
		minimum int
		want    int
	}{
		{traitID: "qijin_qichu", rate: 1, minimum: 60, want: 500},
		{traitID: "baiyi_dujiang", rate: 0.2, minimum: 60, want: 834},
		{traitID: "baiyi_jixing", rate: 0.2, minimum: 60, want: 834},
		{traitID: "kuairu_shandian", rate: 4, minimum: 30, want: 200},
	} {
		t.Run(tc.traitID, func(t *testing.T) {
			ctx := &general.MarchCreateContext{BaseSeconds: 1000, FinalSeconds: 1000, Scene: "attack"}
			general.Dispatch(ctx, []general.ActiveTrait{{
				TraitID: tc.traitID, OwnerSide: "attacker",
				Params: general.Params{"speedBonusRate": tc.rate, "minMarchSeconds": float64(tc.minimum), "triggerChance": 1},
			}})
			if ctx.FinalSeconds != tc.want || len(ctx.Triggered) != 1 {
				t.Fatalf("expected duration %d and one outcome, got duration=%d outcomes=%+v", tc.want, ctx.FinalSeconds, ctx.Triggered)
			}
		})
	}
}

// TestMarchSpeedTraitsRespectMinimumAndProbability 验证行军特性的最低时长和概率边界不会被模板默认值吞掉。
func TestMarchSpeedTraitsRespectMinimumAndProbability(t *testing.T) {
	for _, tc := range []struct {
		traitID string
		rate    float64
		base    int
		minimum int
	}{
		{traitID: "qijin_qichu", rate: 1, base: 100, minimum: 60},
		{traitID: "baiyi_dujiang", rate: 0.2, base: 70, minimum: 60},
		{traitID: "baiyi_jixing", rate: 0.2, base: 70, minimum: 60},
		{traitID: "kuairu_shandian", rate: 4, base: 100, minimum: 30},
	} {
		t.Run(tc.traitID+"最低时长", func(t *testing.T) {
			ctx := &general.MarchCreateContext{BaseSeconds: tc.base, FinalSeconds: tc.base, Scene: "reinforcement"}
			general.Dispatch(ctx, []general.ActiveTrait{{
				TraitID: tc.traitID, OwnerSide: "reinforcement",
				Params: general.Params{"speedBonusRate": tc.rate, "minMarchSeconds": float64(tc.minimum), "triggerChance": 1},
			}})
			if ctx.FinalSeconds != tc.minimum || len(ctx.Triggered) != 1 {
				t.Fatalf("expected minimum %d and one outcome, duration=%d outcomes=%+v", tc.minimum, ctx.FinalSeconds, ctx.Triggered)
			}
		})
	}

	for _, traitID := range []string{"baiyi_dujiang", "kuairu_shandian"} {
		t.Run(traitID+"零概率不触发", func(t *testing.T) {
			ctx := &general.MarchCreateContext{BaseSeconds: 1000, FinalSeconds: 1000, Scene: "attack"}
			general.Dispatch(ctx, []general.ActiveTrait{{
				TraitID: traitID, OwnerSide: "attacker",
				Params: general.Params{"speedBonusRate": 4, "minMarchSeconds": 30, "triggerChance": 0},
			}})
			if ctx.FinalSeconds != 1000 || len(ctx.Triggered) != 0 {
				t.Fatalf("expected zero chance not to trigger, duration=%d outcomes=%+v", ctx.FinalSeconds, ctx.Triggered)
			}
		})
	}
}

// TestAllRecruitCostTraitsModifyActualCost 验证当前征兵减耗特性修改真实资源消耗。
func TestAllRecruitCostTraitsModifyActualCost(t *testing.T) {
	for _, tc := range []struct {
		traitID string
		rate    float64
		want    int
	}{
		{traitID: "wangzuo_zhicai", rate: 0.05, want: 96},
	} {
		t.Run(tc.traitID, func(t *testing.T) {
			ctx := &general.RecruitCostContext{UnitType: "huWei", Category: "infantry", Amount: 1, Cost: map[string]int{"wood": 101}}
			general.Dispatch(ctx, []general.ActiveTrait{{
				TraitID: tc.traitID, OwnerSide: "defender",
				Params: general.Params{"resourceCostReduction": tc.rate, "triggerChance": 1},
			}})
			if ctx.Cost["wood"] != tc.want || len(ctx.Triggered) != 1 {
				t.Fatalf("expected cost %d and one outcome, got cost=%+v outcomes=%+v", tc.want, ctx.Cost, ctx.Triggered)
			}
		})
	}
}

// TestRemainingBeforeBattleTraitIDsModifyRealStats 验证正式配置中其余战前攻防特性 ID 都真实修改对应军团属性。
func TestRemainingBeforeBattleTraitIDsModifyRealStats(t *testing.T) {
	tests := []struct {
		traitID    string
		target     string
		params     general.Params
		check      func(t *testing.T, attacker combat.Army, defender combat.Army)
		outcomeKey string
	}{
		{traitID: "baibu_chuanyang", params: general.Params{"enemyDefenseReductionRate": 0.2, "triggerChance": 1}, outcomeKey: "infantryDefenseModifiedUnits", check: func(t *testing.T, _ combat.Army, defender combat.Army) {
			if defender.Units[0].InfantryDefense != 80 {
				t.Fatalf("expected enemy defense 80, got %+v", defender.Units)
			}
		}},
		{traitID: "dunzhen_fangyu", params: general.Params{"defenseBonusRate": 0.3, "triggerChance": 1}, outcomeKey: "infantryDefenseModifiedUnits", check: func(t *testing.T, attacker combat.Army, _ combat.Army) {
			if attacker.Units[0].InfantryDefense != 130 {
				t.Fatalf("expected own defense 130, got %+v", attacker.Units)
			}
		}},
		{traitID: "meihuo_raozhen", params: general.Params{"enemyDefenseReductionRate": 0.25, "triggerChance": 1}, outcomeKey: "infantryDefenseModifiedUnits", check: func(t *testing.T, _ combat.Army, defender combat.Army) {
			if defender.Units[0].InfantryDefense != 75 {
				t.Fatalf("expected enemy defense 75, got %+v", defender.Units)
			}
		}},
		{traitID: "meizhoulang_junlue", params: general.Params{"attackBonusRate": 0.05, "triggerChance": 1}, outcomeKey: "attackModifiedUnits", check: func(t *testing.T, attacker combat.Army, _ combat.Army) {
			if attacker.Units[0].Attack != 105 {
				t.Fatalf("expected own attack 105, got %+v", attacker.Units)
			}
		}},
		{traitID: "sizhandaodi", target: "infantry", params: general.Params{"attackBonusRate": 0.35, "triggerChance": 1}, outcomeKey: "attackModifiedUnits", check: func(t *testing.T, attacker combat.Army, _ combat.Army) {
			if attacker.Units[0].Attack != 135 || attacker.Units[1].Attack != 100 {
				t.Fatalf("expected only infantry attack 135, got %+v", attacker.Units)
			}
		}},
		{traitID: "wanren_nuhou", target: "southernElephant", params: general.Params{"attackBonusRate": 0.35, "triggerChance": 1}, outcomeKey: "attackModifiedUnits", check: func(t *testing.T, attacker combat.Army, _ combat.Army) {
			if attacker.Units[0].Attack != 100 || attacker.Units[1].Attack != 135 {
				t.Fatalf("expected only Southern Elephant attack 135, got %+v", attacker.Units)
			}
		}},
		{traitID: "weizhen_xiaoyao", target: "cavalry", params: general.Params{"attackBonusRate": 0.35, "triggerChance": 1}, outcomeKey: "attackModifiedUnits", check: func(t *testing.T, attacker combat.Army, _ combat.Army) {
			if attacker.Units[0].Attack != 100 || attacker.Units[1].Attack != 135 {
				t.Fatalf("expected only cavalry attack 135, got %+v", attacker.Units)
			}
		}},
		{traitID: "wusheng_pojun", target: "azureDragon", params: general.Params{"attackBonusRate": 0.38, "triggerChance": 1}, outcomeKey: "attackModifiedUnits", check: func(t *testing.T, attacker combat.Army, _ combat.Army) {
			if attacker.Units[0].Attack != 138 || attacker.Units[1].Attack != 100 {
				t.Fatalf("expected only Azure Dragon attack 138, got %+v", attacker.Units)
			}
		}},
	}
	for _, tc := range tests {
		t.Run(tc.traitID, func(t *testing.T) {
			attacker := combat.Army{Units: []combat.Unit{
				{ID: "azureDragon", Category: "infantry", Count: 100, Attack: 100, InfantryDefense: 100, CavalryDefense: 100},
				{ID: "southernElephant", Category: "cavalry", Count: 100, Attack: 100, InfantryDefense: 100, CavalryDefense: 100},
			}}
			defender := combat.Army{Units: []combat.Unit{
				{ID: "shadowGuard", Category: "infantry", Count: 100, Attack: 100, InfantryDefense: 100, CavalryDefense: 100},
				{ID: "overlordRider", Category: "cavalry", Count: 100, Attack: 100, InfantryDefense: 100, CavalryDefense: 100},
			}}
			ctx := &general.BeforeBattleContext{Attacker: &attacker, Defender: &defender, AttackerOwnsTrait: true, Scene: "attack"}
			general.Dispatch(ctx, []general.ActiveTrait{{TraitID: tc.traitID, OwnerSide: "attacker", TargetUnitType: tc.target, Params: tc.params}})
			tc.check(t, attacker, defender)
			outcome, ok := ctx.Triggered[tc.traitID]
			if !ok || outcome.Detail[tc.outcomeKey] == nil {
				t.Fatalf("expected structured %s outcome, got %+v", tc.outcomeKey, ctx.Triggered)
			}
		})
	}
}

// TestRemainingSuppressionTraitIDsModifyParticipants 验证张辽和张飞的震慑分别减少本场参战兵并记录保留兵力。
func TestRemainingSuppressionTraitIDsModifyParticipants(t *testing.T) {
	for _, tc := range []struct {
		traitID string
		rate    float64
		want    int
	}{
		{traitID: "weizhen_zhenhe", rate: 0.25, want: 75},
		{traitID: "zhenhe_quanjun", rate: 0.5, want: 50},
	} {
		t.Run(tc.traitID, func(t *testing.T) {
			attacker := combat.Army{Units: []combat.Unit{{ID: "huWei", Category: "infantry", Count: 100}}}
			defender := combat.Army{Units: []combat.Unit{{ID: "shadowGuard", Category: "infantry", Count: 100}}}
			ctx := &general.BeforeBattleContext{Attacker: &attacker, Defender: &defender, AttackerOwnsTrait: true, Scene: "attack"}
			general.Dispatch(ctx, []general.ActiveTrait{{TraitID: tc.traitID, OwnerSide: "attacker", Params: general.Params{"effectRate": tc.rate, "triggerChance": 1}}})
			if defender.Units[0].Count != tc.want || ctx.DefenderSuppressedUnits["shadowGuard"] != 100-tc.want {
				t.Fatalf("expected participants %d and preserved suppression %d, army=%+v suppressed=%+v", tc.want, 100-tc.want, defender.Units, ctx.DefenderSuppressedUnits)
			}
			if tc.traitID == "weizhen_zhenhe" {
				outcome := ctx.Triggered[tc.traitID]
				fled, fledOK := outcome.Detail["fledUnits"].(map[string]int)
				returned, returnedOK := outcome.Detail["returnedUnits"].(map[string]int)
				if outcome.Name != "震慑全军" || !fledOK || !returnedOK || fled["shadowGuard"] != 25 || returned["shadowGuard"] != 25 {
					t.Fatalf("expected Zhang Liao flee and return outcome, got %+v", outcome)
				}
			}
		})
	}
}

// TestZhangLiaoSuppressionUsesExactMixedArmyTotal 验证混编敌军按总人数精确溃逃 25%，不会因逐兵种向下取整少算。
func TestZhangLiaoSuppressionUsesExactMixedArmyTotal(t *testing.T) {
	attacker := combat.Army{Units: []combat.Unit{{ID: "huBaoQi", Category: "cavalry", Count: 20}}}
	defender := combat.Army{Units: []combat.Unit{
		{ID: "shadowGuard", Category: "infantry", Count: 1},
		{ID: "overlordRider", Category: "cavalry", Count: 3},
		{ID: "divineWind", Category: "special", Count: 3},
	}}
	ctx := &general.BeforeBattleContext{Attacker: &attacker, Defender: &defender, AttackerOwnsTrait: true, Scene: "attack"}
	general.Dispatch(ctx, []general.ActiveTrait{{
		TraitID: "weizhen_zhenhe", OwnerSide: "attacker",
		Params: general.Params{"effectRate": 0.25, "triggerChance": 1},
	}})
	totalRemaining := 0
	for _, unit := range defender.Units {
		totalRemaining += unit.Count
	}
	totalFled := 0
	for _, amount := range ctx.DefenderSuppressedUnits {
		totalFled += amount
	}
	if totalRemaining != 6 || totalFled != 1 {
		t.Fatalf("expected floor(7*25%%)=1 troop to flee and 6 to participate, remaining=%d fled=%+v", totalRemaining, ctx.DefenderSuppressedUnits)
	}
}

// TestRemainingAfterCombatTraitIDsModifyResult 验证黄盖反击、陆逊增伤和诸葛亮战前控制都进入正确阶段。
func TestRemainingAfterCombatTraitIDsModifyResult(t *testing.T) {
	t.Run("kurou_fanji", func(t *testing.T) {
		result := &combat.CombatResult{DefenderLosses: []combat.UnitLoss{{ID: "shadowGuard", Count: 100, Losses: 10}}}
		ctx := &general.AfterCombatResolveContext{Result: result, Defender: &combat.Army{Units: []combat.Unit{{ID: "shadowGuard", Category: "infantry", Count: 100}}}, AttackerOwnsTrait: true, Scene: "attack"}
		general.Dispatch(ctx, []general.ActiveTrait{{TraitID: "kurou_fanji", OwnerSide: "attacker", Params: general.Params{"effectRate": 0.1, "triggerChance": 1}}})
		if result.DefenderLosses[0].Losses != 20 || ctx.Triggered["kurou_fanji"].Detail["extraLosses"] == nil {
			t.Fatalf("expected Huang Gai counter losses 20, result=%+v outcomes=%+v", result, ctx.Triggered)
		}
	})
	t.Run("lianying_zengshang", func(t *testing.T) {
		result := &combat.CombatResult{DefenderLosses: []combat.UnitLoss{{ID: "shadowGuard", Count: 100, Losses: 10}, {ID: "overlordRider", Count: 100, Losses: 10}}}
		defender := &combat.Army{Units: []combat.Unit{{ID: "shadowGuard", Category: "infantry", Count: 100}, {ID: "overlordRider", Category: "cavalry", Count: 100}}}
		ctx := &general.AfterCombatResolveContext{Result: result, Defender: defender, AttackerOwnsTrait: true, Scene: "attack"}
		general.Dispatch(ctx, []general.ActiveTrait{{TraitID: "lianying_zengshang", OwnerSide: "attacker", TargetUnitType: "infantry", Params: general.Params{"effectRate": 0.1, "triggerChance": 1}}})
		if result.DefenderLosses[0].Losses != 20 || result.DefenderLosses[1].Losses != 10 {
			t.Fatalf("expected only infantry losses 20, got %+v", result.DefenderLosses)
		}
	})
	t.Run("wolong_mouzhi", func(t *testing.T) {
		ctx := &general.BattleTraitControlContext{
			Scene:               "attack",
			CombatGeneralCounts: map[string]int{"defender": 2},
			CombatTraitCounts:   map[string]int{"defender": 4},
		}
		general.Dispatch(ctx, []general.ActiveTrait{
			{TraitID: "wolong_mouzhi", OwnerSide: "attacker", Params: general.Params{"triggerChance": 1}},
		})
		outcome := ctx.Triggered["wolong_mouzhi"]
		if !ctx.DisabledCombatTraitSides["defender"] || outcome.Name != "卧龙奇谋" || outcome.Detail["disabledGeneralCount"] != 2 || outcome.Detail["disabledTraitCount"] != 4 {
			t.Fatalf("expected Wolong to disable all defender combat traits before battle, context=%+v outcomes=%+v", ctx, ctx.Triggered)
		}
	})
}

// TestPreDamageAndSuppressionUseDifferentOutputs 验证真实战前伤亡与临时压制不会混用结算字段。
func TestPreDamageAndSuppressionUseDifferentOutputs(t *testing.T) {
	attacker := combat.Army{Units: []combat.Unit{{ID: "infantry", Count: 100}}}
	defender := combat.Army{Units: []combat.Unit{{ID: "cavalry", Count: 100}}}
	ctx := &general.BeforeBattleContext{Attacker: &attacker, Defender: &defender, AttackerOwnsTrait: true, Scene: "attack"}
	general.Dispatch(ctx, []general.ActiveTrait{
		{TraitID: "yibing_touxi", OwnerSide: "attacker", Params: general.Params{"effectRate": 0.35, "triggerChance": 1}},
		{TraitID: "qimen_dunjia", OwnerSide: "attacker", Params: general.Params{"effectRate": 0.5, "triggerChance": 1}},
	})

	if defender.Units[0].Count != 33 {
		t.Fatalf("expected 100 - 35 pre-damage then floor(65*50%%) suppression to leave 33 participants, got %d", defender.Units[0].Count)
	}
	if ctx.DefenderPreBattleLosses["cavalry"] != 35 || len(ctx.AttackerPreBattleLosses) != 0 {
		t.Fatalf("expected only 35 real defender pre-battle losses, got attacker=%+v defender=%+v", ctx.AttackerPreBattleLosses, ctx.DefenderPreBattleLosses)
	}
	if ctx.DefenderSuppressedUnits["cavalry"] != 32 || len(ctx.AttackerSuppressedUnits) != 0 {
		t.Fatalf("expected 32 suppressed defenders outside real losses, got attacker=%+v defender=%+v", ctx.AttackerSuppressedUnits, ctx.DefenderSuppressedUnits)
	}
	if _, ok := ctx.Triggered["yibing_touxi"]; !ok {
		t.Fatalf("expected pre-damage outcome, got %+v", ctx.Triggered)
	}
	if _, ok := ctx.Triggered["qimen_dunjia"]; !ok {
		t.Fatalf("expected suppression outcome, got %+v", ctx.Triggered)
	}
}

// TestJiangdongGushouOnlyAppliesToDefense 验证江东固守在进攻时无效、防守时生效。
func TestJiangdongGushouOnlyAppliesToDefense(t *testing.T) {
	newContext := func() (*general.BeforeBattleContext, *combat.Army, *combat.Army) {
		attacker := &combat.Army{Units: []combat.Unit{{ID: "infantry", Count: 100, InfantryDefense: 100, CavalryDefense: 100}}}
		defender := &combat.Army{Units: []combat.Unit{{ID: "cavalry", Count: 100, InfantryDefense: 100, CavalryDefense: 100}}}
		return &general.BeforeBattleContext{Attacker: attacker, Defender: defender, Scene: "attack"}, attacker, defender
	}
	active := general.ActiveTrait{
		TraitID: "jiangdong_gushou", AllowedSides: []string{"defender", "reinforcement"},
		Params: general.Params{"defenseBonusRate": 0.5, "triggerChance": 1},
	}

	attackCtx, attacker, _ := newContext()
	attackCtx.AttackerOwnsTrait = true
	active.OwnerSide = "attacker"
	general.Dispatch(attackCtx, []general.ActiveTrait{active})
	if attacker.Units[0].InfantryDefense != 100 || len(attackCtx.Triggered) != 0 {
		t.Fatalf("expected attack-side Sun Quan trait blocked, army=%+v outcomes=%+v", attacker.Units, attackCtx.Triggered)
	}

	defenseCtx, _, defender := newContext()
	defenseCtx.DefenderOwnsTrait = true
	active.OwnerSide = "defender"
	general.Dispatch(defenseCtx, []general.ActiveTrait{active})
	if defender.Units[0].InfantryDefense != 150 || defender.Units[0].CavalryDefense != 150 {
		t.Fatalf("expected defense increased by 50%%, got %+v", defender.Units[0])
	}
	if _, ok := defenseCtx.Triggered["jiangdong_gushou"]; !ok {
		t.Fatalf("expected jiangdong_gushou outcome, got %+v", defenseCtx.Triggered)
	}
}

// TestTraitRequiredOutcome 验证胜负条件会阻止追击在失败时误触发。
func TestTraitRequiredOutcome(t *testing.T) {
	buildContext := func(winner string) *general.AfterCombatResolveContext {
		return &general.AfterCombatResolveContext{
			Result: &combat.CombatResult{
				Winner:         winner,
				AttackerLosses: []combat.UnitLoss{{ID: "infantry", Count: 100, Losses: 10}},
				DefenderLosses: []combat.UnitLoss{{ID: "cavalry", Count: 100, Losses: 10}},
			},
			AttackerOwnsTrait: true,
			Scene:             "attack",
		}
	}
	active := general.ActiveTrait{
		TraitID: "xiaobawang_zhuiji", OwnerSide: "attacker", RequiredOutcome: "win",
		Params: general.Params{"effectRate": 0.2, "triggerChance": 1},
	}

	lostCtx := buildContext("defender")
	general.Dispatch(lostCtx, []general.ActiveTrait{active})
	if lostCtx.Result.DefenderLosses[0].Losses != 10 || len(lostCtx.Triggered) != 0 {
		t.Fatalf("expected pursuit blocked after loss, result=%+v outcomes=%+v", lostCtx.Result, lostCtx.Triggered)
	}

	wonCtx := buildContext("attacker")
	general.Dispatch(wonCtx, []general.ActiveTrait{active})
	if wonCtx.Result.DefenderLosses[0].Losses != 30 {
		t.Fatalf("expected pursuit to increase losses to 30 after win, got %d", wonCtx.Result.DefenderLosses[0].Losses)
	}
}

// TestAfterBattleRequiredLossRejectsDraw 验证战后条件不会把平局解释成战败。
func TestAfterBattleRequiredLossRejectsDraw(t *testing.T) {
	ctx := &general.AfterBattleContext{
		PlayerArmy:   map[string]int{"infantry": 50},
		PlayerLosses: map[string]int{"infantry": 50},
		IsAttacker:   true,
		Won:          false,
		Winner:       "draw",
		Scene:        "plunder",
	}
	general.Dispatch(ctx, []general.ActiveTrait{{
		TraitID: "renzhu_shouhu", OwnerSide: "attacker", RequiredOutcome: "loss",
		Params: general.Params{"effectRate": 0.35, "triggerChance": 1},
	}})
	if ctx.PlayerArmy["infantry"] != 50 || len(ctx.Revived) != 0 || len(ctx.Triggered) != 0 {
		t.Fatalf("expected draw to reject loss-only after-battle trait, ctx=%+v", ctx)
	}
}

// TestGuicaiYiceRevivesActualLossesOnDraw 验证鬼才遗策不限制胜负并按真实阵亡复活。
func TestGuicaiYiceRevivesActualLossesOnDraw(t *testing.T) {
	ctx := &general.AfterBattleContext{
		PlayerArmy:   map[string]int{"infantry": 50, "cavalry": 10},
		PlayerLosses: map[string]int{"infantry": 50, "cavalry": 10},
		IsAttacker:   true,
		Winner:       "draw",
		Scene:        "plunder",
	}
	general.Dispatch(ctx, []general.ActiveTrait{{
		TraitID: "guicai_yice", OwnerSide: "attacker",
		Params: general.Params{"effectRate": 0.22, "maxReviveCount": 0, "triggerChance": 1},
	}})
	outcome, ok := ctx.Triggered["guicai_yice"]
	actualLost, actualOK := outcome.Detail["actualLostUnits"].(map[string]int)
	revived, revivedOK := outcome.Detail["revivedUnits"].(map[string]int)
	if !ok || !actualOK || !revivedOK || actualLost["infantry"] != 50 || actualLost["cavalry"] != 10 ||
		revived["infantry"] != 11 || revived["cavalry"] != 2 || outcome.Detail["totalRevived"] != 13 ||
		ctx.PlayerArmy["infantry"] != 61 || ctx.PlayerArmy["cavalry"] != 12 {
		t.Fatalf("expected Guicai to revive 22%% of each unit's actual losses on draw, ctx=%+v outcome=%+v", ctx, outcome)
	}
}

// TestExtraDamageDoesNotReportZeroChange 验证敌军已全灭时不会生成追加零损失的误导战报。
func TestExtraDamageDoesNotReportZeroChange(t *testing.T) {
	result := &combat.CombatResult{
		Winner:         "attacker",
		AttackerLosses: []combat.UnitLoss{{ID: "infantry", Count: 100, Losses: 10}},
		DefenderLosses: []combat.UnitLoss{{ID: "cavalry", Count: 100, Losses: 100}},
	}
	ctx := &general.AfterCombatResolveContext{Result: result, AttackerOwnsTrait: true, Scene: "plunder"}
	general.Dispatch(ctx, []general.ActiveTrait{{
		TraitID: "xiaobawang_zhuiji", OwnerSide: "attacker", AllowedScenes: []string{"plunder"}, RequiredOutcome: "win",
		Params: general.Params{"effectRate": 0.2, "triggerChance": 1},
	}})
	if _, ok := ctx.Triggered["xiaobawang_zhuiji"]; ok {
		t.Fatalf("expected no zero-effect pursuit outcome, got %+v", ctx.Triggered)
	}
}

// TestDuplicateUnitLossDetailsAggregateAcrossSources 验证同兵种来自多支部队时，追加伤害和减损明细汇总全部实际变化。
func TestDuplicateUnitLossDetailsAggregateAcrossSources(t *testing.T) {
	losses := []combat.UnitLoss{
		{ID: "wuInfantry", Count: 500, Losses: 250},
		{ID: "wuInfantry", Count: 500, Losses: 250},
	}
	added := addLosses(losses, general.Params{"effectRate": 0.1})
	if added.losses[0].Losses != 300 || added.losses[1].Losses != 300 || added.byUnit["wuInfantry"] != 100 {
		t.Fatalf("expected duplicate-unit extra losses 50+50 and detail 100, result=%+v detail=%+v", added.losses, added.byUnit)
	}
	army := &combat.Army{Units: []combat.Unit{
		{ID: "wuInfantry", Category: "infantry", Count: 500},
		{ID: "wuInfantry", Category: "infantry", Count: 500},
	}}
	targeted := addTargetLosses(losses, general.Params{"effectRate": 0.1}, "infantry", army)
	if targeted.losses[0].Losses != 300 || targeted.losses[1].Losses != 300 || targeted.byUnit["wuInfantry"] != 100 {
		t.Fatalf("expected duplicate-unit targeted losses 50+50 and detail 100, result=%+v detail=%+v", targeted.losses, targeted.byUnit)
	}
	reduced := reduceLosses(losses, general.Params{"lossReductionRate": 0.2})
	if reduced.losses[0].Losses != 200 || reduced.losses[1].Losses != 200 || reduced.byUnit["wuInfantry"] != 100 {
		t.Fatalf("expected duplicate-unit reductions 50+50 and detail 100, result=%+v detail=%+v", reduced.losses, reduced.byUnit)
	}
	unevenLosses := []combat.UnitLoss{
		{ID: "wuInfantry", Count: 1, Losses: 1},
		{ID: "wuInfantry", Count: 99, Losses: 49},
	}
	unevenAdded := addLosses(unevenLosses, general.Params{"effectRate": 0.1})
	if unevenAdded.losses[0].Losses != 1 || unevenAdded.losses[1].Losses != 59 || unevenAdded.byUnit["wuInfantry"] != 10 {
		t.Fatalf("expected uneven duplicate sources to share exact grouped extra loss 10, result=%+v detail=%+v", unevenAdded.losses, unevenAdded.byUnit)
	}
	unevenTargeted := addTargetLosses(unevenLosses, general.Params{"effectRate": 0.1}, "infantry", army)
	if unevenTargeted.losses[0].Losses != 1 || unevenTargeted.losses[1].Losses != 59 || unevenTargeted.byUnit["wuInfantry"] != 10 {
		t.Fatalf("expected uneven duplicate targeted sources to share exact grouped extra loss 10, result=%+v detail=%+v", unevenTargeted.losses, unevenTargeted.byUnit)
	}
}

// TestTraitAllowedScene 验证掠夺战攻击加成不会串到普通进攻。
func TestTraitAllowedScene(t *testing.T) {
	active := general.ActiveTrait{
		TraitID: "jinfan_qixi", OwnerSide: "attacker", AllowedSides: []string{"attacker"}, AllowedScenes: []string{"plunder"},
		Params: general.Params{"attackBonusRate": 0.1, "triggerChance": 1},
	}
	buildContext := func(scene string) (*general.BeforeBattleContext, *combat.Army) {
		attacker := &combat.Army{Units: []combat.Unit{{ID: "infantry", Count: 100, Attack: 100}}}
		defender := &combat.Army{Units: []combat.Unit{{ID: "cavalry", Count: 100, Attack: 100}}}
		return &general.BeforeBattleContext{Attacker: attacker, Defender: defender, AttackerOwnsTrait: true, Scene: scene}, attacker
	}

	attackCtx, attackArmy := buildContext("attack")
	general.Dispatch(attackCtx, []general.ActiveTrait{active})
	if attackArmy.Units[0].Attack != 100 || len(attackCtx.Triggered) != 0 {
		t.Fatalf("expected plunder trait blocked in attack, army=%+v outcomes=%+v", attackArmy.Units, attackCtx.Triggered)
	}

	plunderCtx, plunderArmy := buildContext("plunder")
	general.Dispatch(plunderCtx, []general.ActiveTrait{active})
	if plunderArmy.Units[0].Attack != 110 {
		t.Fatalf("expected plunder attack increased to 110, got %d", plunderArmy.Units[0].Attack)
	}
}
