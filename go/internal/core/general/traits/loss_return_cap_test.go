// 本文件验证同场多个战后复活或返还特性不能累计生成超过真实损失的兵力。
package traits

import (
	"testing"

	"hero3/internal/core/general"
)

// TestAfterBattleReturnsAreCappedByRemainingLosses 验证后执行的返兵特性只能处理尚未归队的真实损失。
func TestAfterBattleReturnsAreCappedByRemainingLosses(t *testing.T) {
	ctx := &general.AfterBattleContext{
		PlayerArmy:   map[string]int{"infantry": 0},
		PlayerLosses: map[string]int{"infantry": 100},
		IsAttacker:   true,
		Won:          false,
	}
	general.Dispatch(ctx, []general.ActiveTrait{
		{TraitID: "rende", TraitType: general.TraitTypeSpecial, OwnerSide: "attacker", Params: general.Params{"effectRate": 0.8, "maxReviveCount": 10000, "triggerChance": 1}},
		{TraitID: "renzhu_shouhu", TraitType: general.TraitTypeBonus, OwnerSide: "attacker", Params: general.Params{"lossReductionRate": 0.8, "maxReturnCount": 10000, "triggerChance": 1}},
	})
	if ctx.PlayerArmy["infantry"] != 100 || ctx.Revived["infantry"] != 100 {
		t.Fatalf("expected aggregate return capped at 100 real losses, army=%+v returned=%+v", ctx.PlayerArmy, ctx.Revived)
	}
	rendeUnits, rendeOK := ctx.Triggered["rende"].Detail["revivedUnits"].(map[string]int)
	guardUnits, guardOK := ctx.Triggered["renzhu_shouhu"].Detail["returnedUnits"].(map[string]int)
	if !rendeOK || !guardOK || rendeUnits["infantry"] != 80 || guardUnits["infantry"] != 20 {
		t.Fatalf("expected reports to show actual capped returns 80 + 20, outcomes=%+v", ctx.Triggered)
	}
}

// TestAfterBattleReturnCapsUseStableUnitOrder 验证多兵种触发总上限时按兵种 ID 稳定分配，不受 map 遍历顺序影响。
func TestAfterBattleReturnCapsUseStableUnitOrder(t *testing.T) {
	for attempt := 0; attempt < 50; attempt++ {
		ctx := &general.AfterBattleContext{
			PlayerArmy:   map[string]int{"shuInfantry": 0, "shuCavalry": 0},
			PlayerLosses: map[string]int{"shuInfantry": 30000, "shuCavalry": 30000},
			IsAttacker:   true,
			Won:          false,
		}
		general.Dispatch(ctx, []general.ActiveTrait{
			{TraitID: "rende", TraitType: general.TraitTypeSpecial, OwnerSide: "attacker", Params: general.Params{"effectRate": 0.5, "maxReviveCount": 10000, "triggerChance": 1}},
			{TraitID: "renzhu_shouhu", TraitType: general.TraitTypeBonus, OwnerSide: "attacker", Params: general.Params{"lossReductionRate": 0.1, "maxReturnCount": 10000, "triggerChance": 1}},
		})
		if ctx.PlayerArmy["shuCavalry"] != 13000 || ctx.PlayerArmy["shuInfantry"] != 3000 || ctx.Revived["shuCavalry"] != 13000 || ctx.Revived["shuInfantry"] != 3000 {
			t.Fatalf("expected stable cavalry-first capped returns 13000/3000, attempt=%d army=%+v returned=%+v", attempt, ctx.PlayerArmy, ctx.Revived)
		}
		rendeUnits, rendeOK := ctx.Triggered["rende"].Detail["revivedUnits"].(map[string]int)
		guardUnits, guardOK := ctx.Triggered["renzhu_shouhu"].Detail["returnedUnits"].(map[string]int)
		if !rendeOK || !guardOK || rendeUnits["shuCavalry"] != 10000 || rendeUnits["shuInfantry"] != 0 || guardUnits["shuCavalry"] != 3000 || guardUnits["shuInfantry"] != 3000 {
			t.Fatalf("expected stable per-trait actual returns, attempt=%d outcomes=%+v", attempt, ctx.Triggered)
		}
	}
}
