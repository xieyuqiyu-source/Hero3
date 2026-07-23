// 本文件验证刘备仁主守护按真实阵亡逐兵种复活且不沿用旧人数上限。
package traits

import (
	"testing"

	"hero3/internal/core/general"
)

// TestRenzhuShouhuRevivesWithoutLegacyCountCap 验证大额多兵种阵亡不会被旧 10000 人上限截断。
func TestRenzhuShouhuRevivesWithoutLegacyCountCap(t *testing.T) {
	ctx := &general.AfterBattleContext{
		PlayerArmy:   map[string]int{"shuInfantry": 0, "shuCavalry": 0},
		PlayerLosses: map[string]int{"shuInfantry": 30000, "shuCavalry": 30000},
		IsAttacker:   true,
	}
	general.Dispatch(ctx, []general.ActiveTrait{{
		TraitID: "renzhu_shouhu", TraitType: general.TraitTypeBonus, OwnerSide: "attacker",
		Params: general.Params{"effectRate": 0.35, "triggerChance": 1},
	}})
	if ctx.PlayerArmy["shuCavalry"] != 10500 || ctx.PlayerArmy["shuInfantry"] != 10500 ||
		ctx.Revived["shuCavalry"] != 10500 || ctx.Revived["shuInfantry"] != 10500 {
		t.Fatalf("expected each unit to revive 35%% without count cap, army=%+v revived=%+v", ctx.PlayerArmy, ctx.Revived)
	}
	outcome := ctx.Triggered["renzhu_shouhu"]
	actual, actualOK := outcome.Detail["actualLostUnits"].(map[string]int)
	revived, revivedOK := outcome.Detail["revivedUnits"].(map[string]int)
	if !actualOK || !revivedOK || actual["shuCavalry"] != 30000 || actual["shuInfantry"] != 30000 ||
		revived["shuCavalry"] != 10500 || revived["shuInfantry"] != 10500 || outcome.Detail["totalRevived"] != 21000 {
		t.Fatalf("expected exact uncapped revival detail, outcome=%+v", outcome)
	}
}

// TestRenzhuShouhuZeroChanceLeavesContextUntouched 验证概率未命中时不修改兵力和触发结果。
func TestRenzhuShouhuZeroChanceLeavesContextUntouched(t *testing.T) {
	ctx := &general.AfterBattleContext{
		PlayerArmy:   map[string]int{"infantry": 0},
		PlayerLosses: map[string]int{"infantry": 100},
		IsAttacker:   true,
	}
	general.Dispatch(ctx, []general.ActiveTrait{{
		TraitID: "renzhu_shouhu", TraitType: general.TraitTypeBonus, OwnerSide: "attacker",
		Params: general.Params{"effectRate": 0.35, "triggerChance": 0},
	}})
	if ctx.PlayerArmy["infantry"] != 0 || len(ctx.Revived) != 0 || len(ctx.Triggered) != 0 {
		t.Fatalf("expected probability miss to keep context untouched, ctx=%+v", ctx)
	}
}
