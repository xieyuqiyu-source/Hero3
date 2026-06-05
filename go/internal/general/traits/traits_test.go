package traits

import (
	"math/rand"
	"testing"

	"hero3/internal/combat"
	"hero3/internal/general"
)

// 美人计：俘虏 10% 敌方，归我方军队
func TestMeiren_BasicCapture(t *testing.T) {
	rand.Seed(1)

	defender := combat.Army{
		Faction: "shu",
		Units: []combat.Unit{
			{ID: "infantry", Count: 100},
			{ID: "cavalry", Count: 50},
		},
	}
	ctx := &general.BeforeBattleContext{
		Attacker:          &combat.Army{Faction: "wei"},
		Defender:          &defender,
		AttackerOwnsTrait: true,
		IsPvP:             false,
		SameFaction:       true, // PvE
	}

	activeTraits := []general.ActiveTrait{
		{TraitID: "meiren", Params: general.Params{
			"captureRate":   0.1,
			"captureMax":    1000,
			"triggerChance": 1.0,
		}},
	}
	general.Dispatch(ctx, activeTraits)

	if ctx.CapturedToArmy["infantry"] != 10 {
		t.Errorf("expected 10 infantry captured, got %d", ctx.CapturedToArmy["infantry"])
	}
	if ctx.CapturedToArmy["cavalry"] != 5 {
		t.Errorf("expected 5 cavalry captured, got %d", ctx.CapturedToArmy["cavalry"])
	}
	// 敌方剩余兵
	if defender.Units[0].Count != 90 {
		t.Errorf("expected 90 infantry remaining, got %d", defender.Units[0].Count)
	}
	if defender.Units[1].Count != 45 {
		t.Errorf("expected 45 cavalry remaining, got %d", defender.Units[1].Count)
	}
}

// 美人计：跨阵营进驻防
func TestMeiren_CrossFactionGoesToGarrison(t *testing.T) {
	rand.Seed(1)

	defender := combat.Army{Units: []combat.Unit{{ID: "infantry", Count: 100}}}
	ctx := &general.BeforeBattleContext{
		Attacker:          &combat.Army{},
		Defender:          &defender,
		AttackerOwnsTrait: true,
		IsPvP:             true,
		SameFaction:       false, // 跨阵营
	}
	general.Dispatch(ctx, []general.ActiveTrait{
		{TraitID: "meiren", Params: general.Params{"captureRate": 0.1, "captureMax": 1000, "triggerChance": 1.0}},
	})

	if ctx.CapturedToGarrison["infantry"] != 10 {
		t.Errorf("expected 10 captured to garrison, got %d", ctx.CapturedToGarrison["infantry"])
	}
	if len(ctx.CapturedToArmy) != 0 {
		t.Errorf("expected no army captures, got %v", ctx.CapturedToArmy)
	}
}

// 美人计：单兵种上限
func TestMeiren_CaptureMax(t *testing.T) {
	defender := combat.Army{Units: []combat.Unit{{ID: "infantry", Count: 10000}}}
	ctx := &general.BeforeBattleContext{
		Attacker:          &combat.Army{},
		Defender:          &defender,
		AttackerOwnsTrait: true,
		SameFaction:       true,
	}
	general.Dispatch(ctx, []general.ActiveTrait{
		{TraitID: "meiren", Params: general.Params{"captureRate": 0.5, "captureMax": 100, "triggerChance": 1.0}},
	})
	// 50% × 10000 = 5000，但 max=100，应该被限制到 100
	if ctx.CapturedToArmy["infantry"] != 100 {
		t.Errorf("expected 100 captured (max), got %d", ctx.CapturedToArmy["infantry"])
	}
}

// 美人计：防守方拥有特性时不触发
func TestMeiren_DefenderOwnsTraitDoesNothing(t *testing.T) {
	defender := combat.Army{Units: []combat.Unit{{ID: "infantry", Count: 100}}}
	ctx := &general.BeforeBattleContext{
		Attacker:          &combat.Army{},
		Defender:          &defender,
		AttackerOwnsTrait: false,
		DefenderOwnsTrait: true,
		SameFaction:       true,
	}
	general.Dispatch(ctx, []general.ActiveTrait{
		{TraitID: "meiren", Params: general.Params{"captureRate": 0.1, "captureMax": 1000, "triggerChance": 1.0}},
	})
	if len(ctx.CapturedToArmy) != 0 {
		t.Errorf("expected no captures when only defender has trait")
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

// 仁德：复活损失的兵
func TestRende_RevivesLosses(t *testing.T) {
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
		{TraitID: "rende", Params: general.Params{"reviveRate": 0.5, "triggerChance": 1.0}},
	})
	// 20 × 0.5 = 10 复活，80 + 10 = 90
	if playerArmy["infantry"] != 90 {
		t.Errorf("expected 90 infantry after revive, got %d", playerArmy["infantry"])
	}
	// 10 × 0.5 = 5 复活，30 + 5 = 35
	if playerArmy["cavalry"] != 35 {
		t.Errorf("expected 35 cavalry after revive, got %d", playerArmy["cavalry"])
	}
	if ctx.Revived["infantry"] != 10 || ctx.Revived["cavalry"] != 5 {
		t.Errorf("expected revived map {infantry:10, cavalry:5}, got %v", ctx.Revived)
	}
}

// 仁德：失败也能触发
func TestRende_TriggersOnLoss(t *testing.T) {
	rand.Seed(1)
	playerArmy := map[string]int{"infantry": 0}
	playerLosses := map[string]int{"infantry": 100}

	ctx := &general.AfterBattleContext{
		PlayerArmy: playerArmy, PlayerLosses: playerLosses,
		IsAttacker: true, Won: false, // 失败
	}
	general.Dispatch(ctx, []general.ActiveTrait{
		{TraitID: "rende", Params: general.Params{"reviveRate": 0.3, "triggerChance": 1.0}},
	})
	if playerArmy["infantry"] != 30 {
		t.Errorf("expected 30 infantry revived on loss, got %d", playerArmy["infantry"])
	}
}

// 威震逍遥：以少打多时震慑防守方部分兵力，不参与防御计算
func TestWeizhenXiaoyao_SuppressesDefenderWhenOutnumbered(t *testing.T) {
	rand.Seed(1)

	attacker := combat.Army{
		Faction: "wei",
		Units: []combat.Unit{
			{ID: "huBaoQi", Count: 10, Upkeep: 4},
		},
	}
	defender := combat.Army{
		Faction: "wu",
		Units: []combat.Unit{
			{ID: "infantry", Count: 100, Upkeep: 2},
			{ID: "cavalry", Count: 50, Upkeep: 3},
		},
	}
	ctx := &general.BeforeBattleContext{
		Attacker:          &attacker,
		Defender:          &defender,
		AttackerOwnsTrait: true,
	}

	general.Dispatch(ctx, []general.ActiveTrait{
		{TraitID: "weizhenxiaoyao", Params: general.Params{
			"baseChance":       1.0,
			"chancePerRatio":   0,
			"maxChance":        1.0,
			"baseSuppressRate": 0.10,
			"suppressPerRatio": 0,
			"maxSuppressRate":  0.10,
		}},
	})

	if defender.Units[0].Count != 90 {
		t.Errorf("expected 90 infantry participating after suppression, got %d", defender.Units[0].Count)
	}
	if defender.Units[1].Count != 45 {
		t.Errorf("expected 45 cavalry participating after suppression, got %d", defender.Units[1].Count)
	}
	outcome, ok := ctx.Triggered["weizhenxiaoyao"]
	if !ok {
		t.Fatalf("expected weizhenxiaoyao outcome")
	}
	if outcome.Detail["totalSuppressed"] != 15 {
		t.Errorf("expected totalSuppressed 15, got %v", outcome.Detail["totalSuppressed"])
	}
}

// 威震逍遥：不是以少打多时不进入判定
func TestWeizhenXiaoyao_DoesNothingWhenNotOutnumbered(t *testing.T) {
	attacker := combat.Army{
		Faction: "wei",
		Units: []combat.Unit{
			{ID: "huBaoQi", Count: 100, Upkeep: 4},
		},
	}
	defender := combat.Army{
		Faction: "wu",
		Units: []combat.Unit{
			{ID: "infantry", Count: 10, Upkeep: 2},
		},
	}
	ctx := &general.BeforeBattleContext{
		Attacker:          &attacker,
		Defender:          &defender,
		AttackerOwnsTrait: true,
	}

	general.Dispatch(ctx, []general.ActiveTrait{
		{TraitID: "weizhenxiaoyao", Params: general.Params{
			"baseChance":       1.0,
			"baseSuppressRate": 0.50,
			"maxSuppressRate":  0.50,
		}},
	})

	if defender.Units[0].Count != 10 {
		t.Errorf("expected defender unchanged, got %d", defender.Units[0].Count)
	}
	if len(ctx.Triggered) != 0 {
		t.Errorf("expected no triggered traits, got %v", ctx.Triggered)
	}
}

// 威震逍遥：极低震慑率按总兵力向下取整，不应按每个兵种保底 1 个放大
func TestWeizhenXiaoyao_LowSuppressRateDoesNotSuppress(t *testing.T) {
	attacker := combat.Army{Units: []combat.Unit{{ID: "huBaoQi", Count: 1, Upkeep: 1}}}
	defender := combat.Army{Units: []combat.Unit{
		{ID: "infantry", Count: 10, Upkeep: 1},
		{ID: "cavalry", Count: 10, Upkeep: 1},
	}}
	ctx := &general.BeforeBattleContext{
		Attacker:          &attacker,
		Defender:          &defender,
		AttackerOwnsTrait: true,
	}

	general.Dispatch(ctx, []general.ActiveTrait{
		{TraitID: "weizhenxiaoyao", Params: general.Params{
			"baseChance":       1.0,
			"maxChance":        1.0,
			"baseSuppressRate": 0.001,
			"maxSuppressRate":  0.001,
		}},
	})

	if defender.Units[0].Count != 10 || defender.Units[1].Count != 10 {
		t.Fatalf("expected defender unchanged, got %+v", defender.Units)
	}
	if len(ctx.Triggered) != 0 {
		t.Fatalf("expected no triggered trait when total suppressed is 0, got %v", ctx.Triggered)
	}
}

// 威震逍遥：0 口粮单位不参与口粮比计算，己方只有 0 口粮单位时不触发
func TestWeizhenXiaoyao_ZeroUpkeepAttackerDoesNotTrigger(t *testing.T) {
	attacker := combat.Army{Units: []combat.Unit{{ID: "merchant", Count: 100, Upkeep: 0}}}
	defender := combat.Army{Units: []combat.Unit{{ID: "infantry", Count: 100, Upkeep: 1}}}
	ctx := &general.BeforeBattleContext{
		Attacker:          &attacker,
		Defender:          &defender,
		AttackerOwnsTrait: true,
	}

	general.Dispatch(ctx, []general.ActiveTrait{
		{TraitID: "weizhenxiaoyao", Params: general.Params{
			"baseChance":       1.0,
			"maxChance":        1.0,
			"baseSuppressRate": 0.5,
			"maxSuppressRate":  0.5,
		}},
	})

	if defender.Units[0].Count != 100 {
		t.Fatalf("expected defender unchanged, got %d", defender.Units[0].Count)
	}
	if len(ctx.Triggered) != 0 {
		t.Fatalf("expected no triggered trait, got %v", ctx.Triggered)
	}
}
