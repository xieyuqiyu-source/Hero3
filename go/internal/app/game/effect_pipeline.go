package game

import (
	"strings"
	"time"

	coreeffect "hero3/internal/core/effect"
)

// 本文件归口应用层 Effect Pipeline，把玩法提交的标准效果落到玩家状态。

const (
	EffectTypeReward           = coreeffect.TypeReward
	EffectTypeBuildingMutation = coreeffect.TypeBuildingMutation
	EffectTypeModifier         = coreeffect.TypeModifier
)

type Effect = coreeffect.Effect
type EffectContext = coreeffect.Context
type ModifierEffect = coreeffect.ModifierEffect

type EffectApplyResult struct {
	Core          coreeffect.ApplyResult `json:"core"`
	Reward        RewardApplyResult      `json:"reward"`
	MutatedAssets []string               `json:"mutatedAssets,omitempty"`
}

type EffectExecutionResult struct {
	State GameState         `json:"state"`
	Apply EffectApplyResult `json:"apply"`
}

type AccountEffectExecutionResult struct {
	Account Account           `json:"account"`
	State   GameState         `json:"state"`
	Apply   EffectApplyResult `json:"apply"`
}

// ExecuteEffects 在玩家状态事务中执行标准效果，并发布核心资产变化。
func (s *Service) ExecuteEffects(playerID string, effects []Effect, ctx EffectContext) (EffectExecutionResult, error) {
	playerID = strings.TrimSpace(playerID)
	if playerID == "" {
		return EffectExecutionResult{}, ErrPlayerNotFound
	}
	ctx.PlayerID = firstNonEmpty(ctx.PlayerID, playerID)
	now := time.Now()
	var before, after coreAssetSnapshot
	var applyResult EffectApplyResult
	state, err := s.repo.UpdatePlayerState(playerID, now, func(state *GameState) error {
		nextState, _ := settleResources(*state, now)
		*state = nextState
		before = snapshotCoreAssets(state)
		result, err := ExecuteEffectsOnState(state, effects, ctx, now)
		if err != nil {
			return err
		}
		if result.Reward.AccountGold > 0 {
			return ErrAccountNotFound
		}
		nextState, _ = settleResources(*state, now)
		*state = nextState
		after = snapshotCoreAssets(state)
		applyResult = result
		return nil
	})
	if err != nil {
		return EffectExecutionResult{}, err
	}
	s.flushRewardSideEffects(applyResult.Reward)
	s.publishCoreAssetDiff(playerID, firstNonEmpty(ctx.RefType, "effect"), ctx.RefID, before, after, now)
	hydrateStateForResponse(&state, now)
	return EffectExecutionResult{State: state, Apply: applyResult}, nil
}

// ExecuteEffectsWithAccount 在账号 + 玩家组合事务中执行标准效果，支持账号级奖励。
func (s *Service) ExecuteEffectsWithAccount(accountID string, playerID string, effects []Effect, ctx EffectContext) (AccountEffectExecutionResult, error) {
	accountID = strings.TrimSpace(accountID)
	playerID = strings.TrimSpace(playerID)
	if accountID == "" {
		return AccountEffectExecutionResult{}, ErrAccountNotFound
	}
	if playerID == "" {
		return AccountEffectExecutionResult{}, ErrPlayerNotFound
	}
	ctx.AccountID = firstNonEmpty(ctx.AccountID, accountID)
	ctx.PlayerID = firstNonEmpty(ctx.PlayerID, playerID)
	now := time.Now()

	var before, after coreAssetSnapshot
	var applyResult EffectApplyResult
	account, state, err := s.repo.UpdateAccountPlayerState(accountID, playerID, now, func(account *Account, state *GameState) error {
		nextState, _ := settleResources(*state, now)
		*state = nextState
		before = snapshotCoreAssets(state)
		result, err := ExecuteEffectsOnState(state, effects, ctx, now)
		if err != nil {
			return err
		}
		if result.Reward.AccountGold > 0 {
			account.Gold += result.Reward.AccountGold
			result.Reward.LedgerEntries = append(result.Reward.LedgerEntries, GoldLedgerEntry{
				AccountID:    accountID,
				PlayerID:     playerID,
				Currency:     LedgerCurrencyGold,
				Direction:    LedgerDirectionCredit,
				Amount:       result.Reward.AccountGold,
				BalanceAfter: account.Gold,
				RefType:      ctx.RefType,
				RefID:        ctx.RefID,
				Reason:       ctx.Reason,
				CreatedAt:    now.UTC().Format(resourceDateLayout),
			})
		}
		nextState, _ = settleResources(*state, now)
		*state = nextState
		after = snapshotCoreAssets(state)
		applyResult = result
		return nil
	})
	if err != nil {
		return AccountEffectExecutionResult{}, err
	}
	s.flushRewardSideEffects(applyResult.Reward)
	s.publishCoreAssetDiff(playerID, firstNonEmpty(ctx.RefType, "effect"), ctx.RefID, before, after, now)
	hydrateStateForResponse(&state, now)
	return AccountEffectExecutionResult{Account: account, State: state, Apply: applyResult}, nil
}

// ExecuteEffectsOnState 执行标准效果列表，所有长期资产变更必须复用既有核心入口。
func ExecuteEffectsOnState(state *GameState, effects []Effect, ctx EffectContext, now time.Time) (EffectApplyResult, error) {
	if state == nil {
		return EffectApplyResult{}, ErrPlayerNotFound
	}
	if now.IsZero() {
		now = time.Now()
	}

	result := EffectApplyResult{
		Core:   coreeffect.ApplyResult{},
		Reward: RewardApplyResult{Granted: map[string]int{}},
	}
	rewardCtx := RewardGrantContext{
		AccountID: ctx.AccountID,
		PlayerID:  firstNonEmpty(ctx.PlayerID, state.Player.ID),
		RefType:   ctx.RefType,
		RefID:     ctx.RefID,
		Reason:    ctx.Reason,
	}

	for _, effect := range effects {
		effectType := coreeffect.NormalizeType(effect.Type)
		switch effectType {
		case EffectTypeReward:
			apply, err := ApplyRewardsToStateWithContext(state, effect.Rewards, rewardCtx, now)
			if err != nil {
				return EffectApplyResult{}, err
			}
			mergeRewardApplyResult(&result.Reward, apply)
			result.Core.Applied++
			result.Core.Types = append(result.Core.Types, effectType)
		case EffectTypeBuildingMutation:
			if effect.BuildingMutation == nil {
				return EffectApplyResult{}, ErrInvalidBuildingMutation
			}
			mutation := *effect.BuildingMutation
			mutation.BuildingID = strings.TrimSpace(mutation.BuildingID)
			building := findBuildingByID(state, mutation.BuildingID)
			if building == nil {
				return EffectApplyResult{}, ErrBuildingNotFound
			}
			if err := applyBuildingMutation(building, mutation, now); err != nil {
				return EffectApplyResult{}, err
			}
			result.MutatedAssets = append(result.MutatedAssets, "building:"+mutation.BuildingID)
			result.Core.Applied++
			result.Core.Types = append(result.Core.Types, effectType)
		case EffectTypeModifier:
			reward, err := rewardFromModifierEffect(effect, ctx)
			if err != nil {
				return EffectApplyResult{}, err
			}
			apply, err := ApplyRewardsToStateWithContext(state, []Reward{reward}, rewardCtx, now)
			if err != nil {
				return EffectApplyResult{}, err
			}
			mergeRewardApplyResult(&result.Reward, apply)
			result.Core.Applied++
			result.Core.Types = append(result.Core.Types, effectType)
		default:
			return EffectApplyResult{}, ErrInvalidEffectType
		}
	}

	state.ServerTime = now.UTC().Format(resourceDateLayout)
	return result, nil
}

// rewardFromModifierEffect 把 Modifier 效果转换为标准 Buff 奖励。
func rewardFromModifierEffect(effect Effect, ctx EffectContext) (Reward, error) {
	if effect.Modifier == nil {
		return Reward{}, ErrInvalidBuffKey
	}
	mod := *effect.Modifier
	if err := validateBuffModifierSpec(mod.Key, mod.Mode); err != nil {
		return Reward{}, err
	}
	source := firstNonEmpty(mod.Source, ctx.Source, effect.Source, "effect")
	note := firstNonEmpty(mod.Note, ctx.Reason)
	amount := mod.Stack
	if amount <= 0 {
		amount = 1
	}
	return Reward{
		Type:   RewardTypeBuff,
		ID:     strings.TrimSpace(mod.Key),
		Amount: amount,
		Metadata: map[string]any{
			"value":  mod.Value,
			"mode":   strings.TrimSpace(mod.Mode),
			"hours":  mod.Hours,
			"source": source,
			"note":   note,
		},
	}, nil
}

// rewardsToEffects 把奖励列表转换为标准效果列表。
func rewardsToEffects(source string, rewards []Reward) []Effect {
	if len(rewards) == 0 {
		return nil
	}
	return []Effect{coreeffect.RewardEffect(source, rewards...)}
}

// buildingMutationToEffect 把建筑变更转换为标准效果。
func buildingMutationToEffect(source string, mutation BuildingMutation) Effect {
	return coreeffect.BuildingMutationEffect(source, mutation)
}
