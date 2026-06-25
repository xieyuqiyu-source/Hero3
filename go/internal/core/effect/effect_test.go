package effect

import (
	"testing"

	corebuilding "hero3/internal/core/building"
	corereward "hero3/internal/core/reward"
)

// 本文件验证核心效果契约的构造 helper。

func TestRewardEffectBuildsStandardEffect(t *testing.T) {
	eff := RewardEffect("item", corereward.Reward{Type: corereward.TypeGeneralExp, ID: "current_general", Amount: 10})
	if eff.Type != TypeReward || eff.Source != "item" {
		t.Fatalf("unexpected reward effect: %+v", eff)
	}
	if len(eff.Rewards) != 1 || eff.Rewards[0].Amount != 10 {
		t.Fatalf("unexpected rewards: %+v", eff.Rewards)
	}
}

func TestBuildingMutationEffectBuildsStandardEffect(t *testing.T) {
	mutation := corebuilding.Mutation{Type: corebuilding.MutationDestroy, BuildingID: "warehouse-1"}
	eff := BuildingMutationEffect("building", mutation)
	if eff.Type != TypeBuildingMutation || eff.ID != "warehouse-1" {
		t.Fatalf("unexpected building effect: %+v", eff)
	}
	if eff.BuildingMutation == nil || eff.BuildingMutation.Type != corebuilding.MutationDestroy {
		t.Fatalf("unexpected mutation: %+v", eff.BuildingMutation)
	}
}

func TestModifierEffectRequestBuildsStandardEffect(t *testing.T) {
	eff := ModifierEffectRequest("event", ModifierEffect{Key: "attackBonus", Value: 0.2, Mode: "percentAdd"})
	if eff.Type != TypeModifier || eff.ID != "attackBonus" {
		t.Fatalf("unexpected modifier effect: %+v", eff)
	}
	if eff.Modifier == nil || eff.Modifier.Value != 0.2 {
		t.Fatalf("unexpected modifier: %+v", eff.Modifier)
	}
}
