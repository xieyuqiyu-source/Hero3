package game

import (
	"testing"
	"time"
)

// 本文件验证应用层 Effect Pipeline 复用奖励、建筑和 Modifier 入口。

func TestEffectPipelineAppliesRewardEffect(t *testing.T) {
	now := time.Now()
	state := newPlayerState("player_effect_reward", "EffectReward", "wei", "caocao", now)
	before := state.Resources.Items["wood"]

	result, err := ExecuteEffectsOnState(&state, rewardsToEffects("test", []Reward{
		{Type: RewardTypeResource, ID: "wood", Amount: 20},
	}), EffectContext{PlayerID: state.Player.ID, RefType: "test_effect", RefID: "reward"}, now)
	if err != nil {
		t.Fatalf("ExecuteEffectsOnState reward failed: %v", err)
	}
	if result.Core.Applied != 1 {
		t.Fatalf("expected 1 applied effect, got %+v", result.Core)
	}
	if state.Resources.Items["wood"] != before+20 {
		t.Fatalf("expected wood +20, got before=%d after=%d", before, state.Resources.Items["wood"])
	}
}

func TestEffectPipelineAppliesBuildingMutationEffect(t *testing.T) {
	now := time.Now()
	state := newPlayerState("player_effect_building", "EffectBuilding", "wei", "caocao", now)
	state.Buildings = []Building{{ID: "warehouse-1", Type: "warehouse", Level: 1}}

	_, err := ExecuteEffectsOnState(&state, []Effect{
		buildingMutationToEffect("test_destroy", BuildingMutation{Type: BuildingMutationDestroy, BuildingID: "warehouse-1"}),
	}, EffectContext{PlayerID: state.Player.ID, RefType: "test_effect", RefID: "warehouse-1"}, now)
	if err != nil {
		t.Fatalf("ExecuteEffectsOnState building failed: %v", err)
	}
	if state.Buildings[0].Status != BuildingStatusDestroyed {
		t.Fatalf("expected warehouse destroyed, got %q", state.Buildings[0].Status)
	}
}

func TestEffectPipelineAppliesModifierEffect(t *testing.T) {
	now := time.Now()
	state := newPlayerState("player_effect_modifier", "EffectModifier", "wei", "caocao", now)

	result, err := ExecuteEffectsOnState(&state, []Effect{{
		Type:   EffectTypeModifier,
		Source: "event",
		Modifier: &ModifierEffect{
			Key:    StatAttackBonus,
			Value:  0.2,
			Mode:   "percentAdd",
			Hours:  1,
			Source: "event",
		},
	}}, EffectContext{PlayerID: state.Player.ID, RefType: "test_effect", RefID: "modifier"}, now)
	if err != nil {
		t.Fatalf("ExecuteEffectsOnState modifier failed: %v", err)
	}
	if result.Reward.Granted[StatAttackBonus] != 1 {
		t.Fatalf("expected modifier reward granted once, got %+v", result.Reward.Granted)
	}
	if len(state.Buffs) != 1 || state.Buffs[0].Key != StatAttackBonus {
		t.Fatalf("expected attack buff, got %+v", state.Buffs)
	}
}

func TestServiceExecuteEffectsPublishesAssetDiff(t *testing.T) {
	repo := NewMemoryRepository()
	service := NewServiceWithRepository(repo)
	now := time.Now()
	account := Account{ID: "account_effect", Username: "effect_user", PasswordHash: "hash", CreatedAt: now}
	if err := repo.CreateAccount(account); err != nil {
		t.Fatalf("create account: %v", err)
	}
	state := newPlayerState("player_effect_service", "EffectService", "wei", "caocao", now)
	state.Buildings = []Building{{ID: "warehouse-1", Type: "warehouse", Level: 1}}
	if err := repo.CreatePlayer(account.ID, state, now); err != nil {
		t.Fatalf("create player: %v", err)
	}

	events := []GameEvent{}
	service.SubscribeEvent(EventBuildingUpgraded, func(event GameEvent) {
		events = append(events, event)
	})
	result, err := service.ExecuteEffects(state.Player.ID, []Effect{
		buildingMutationToEffect("test_destroy", BuildingMutation{Type: BuildingMutationDestroy, BuildingID: "warehouse-1"}),
	}, EffectContext{PlayerID: state.Player.ID, RefType: "test_effect", RefID: "warehouse-1"})
	if err != nil {
		t.Fatalf("ExecuteEffects failed: %v", err)
	}
	if result.State.Buildings[0].Status != BuildingStatusDestroyed {
		t.Fatalf("expected destroyed building, got %+v", result.State.Buildings[0])
	}
	if len(events) != 1 {
		t.Fatalf("expected one building event, got %+v", events)
	}
}

func TestServiceExecuteEffectsWithAccountAppliesAccountGold(t *testing.T) {
	repo := NewMemoryRepository()
	service := NewServiceWithRepository(repo)
	now := time.Now()
	account := Account{ID: "account_effect_gold", Username: "effect_gold_user", PasswordHash: "hash", Gold: 3, CreatedAt: now}
	if err := repo.CreateAccount(account); err != nil {
		t.Fatalf("create account: %v", err)
	}
	state := newPlayerState("player_effect_gold", "EffectGold", "wei", "caocao", now)
	if err := repo.CreatePlayer(account.ID, state, now); err != nil {
		t.Fatalf("create player: %v", err)
	}

	result, err := service.ExecuteEffectsWithAccount(account.ID, state.Player.ID, rewardsToEffects("test_gold_effect", []Reward{
		{Type: RewardTypeGold, ID: RewardTypeGold, Amount: 12},
	}), EffectContext{RefType: "test_gold_effect", RefID: "gold_reward", Reason: "effect_account_gold"})
	if err != nil {
		t.Fatalf("ExecuteEffectsWithAccount failed: %v", err)
	}
	if result.Account.Gold != 15 {
		t.Fatalf("expected account gold 15, got %d", result.Account.Gold)
	}
	if result.Apply.Reward.AccountGold != 12 {
		t.Fatalf("expected account gold apply 12, got %+v", result.Apply.Reward)
	}

	accountAfter, err := service.GetAccountByID(account.ID)
	if err != nil {
		t.Fatalf("GetAccountByID failed: %v", err)
	}
	if accountAfter.Gold != 15 {
		t.Fatalf("expected persisted account gold 15, got %d", accountAfter.Gold)
	}
	entries, err := service.ListGoldLedger(GoldLedgerFilter{AccountID: account.ID, RefType: "test_gold_effect"})
	if err != nil {
		t.Fatalf("ListGoldLedger failed: %v", err)
	}
	if len(entries) != 1 || entries[0].Amount != 12 || entries[0].BalanceAfter != 15 {
		t.Fatalf("expected one gold ledger entry, got %+v", entries)
	}
}
