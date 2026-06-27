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

// ExecuteEffects 按效果类型选择资产级事务执行，并发布核心资产变化。
func (s *Service) ExecuteEffects(playerID string, effects []Effect, ctx EffectContext) (EffectExecutionResult, error) {
	playerID = strings.TrimSpace(playerID)
	if playerID == "" {
		return EffectExecutionResult{}, ErrPlayerNotFound
	}
	ctx.PlayerID = firstNonEmpty(ctx.PlayerID, playerID)
	now := time.Now()
	var before, after coreAssetSnapshot
	var applyResult EffectApplyResult
	updateFn := func(state *GameState) error {
		if effectListContainsBuildingMutation(effects) {
			nextState, _ := settleResources(*state, now)
			*state = nextState
		}
		before = snapshotCoreAssets(state)
		result, err := ExecuteEffectsOnState(state, effects, ctx, now)
		if err != nil {
			return err
		}
		if result.Reward.AccountGold > 0 {
			return ErrAccountNotFound
		}
		if effectListContainsBuildingMutation(effects) {
			nextState, _ := settleResources(*state, now)
			*state = nextState
		}
		after = snapshotCoreAssets(state)
		applyResult = result
		return nil
	}
	state, err := s.executeEffectsInPlayerAssetTransaction(playerID, now, effects, updateFn)
	if err != nil {
		return EffectExecutionResult{}, err
	}
	s.flushRewardSideEffects(applyResult.Reward)
	s.publishCoreAssetDiff(playerID, firstNonEmpty(ctx.RefType, "effect"), ctx.RefID, before, after, now)
	hydrateStateForResponse(&state, now)
	return EffectExecutionResult{State: state, Apply: applyResult}, nil
}

// ExecuteEffectsWithAccount 按效果类型选择账号 + 资产级事务执行，支持账号级奖励。
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
	updateFn := func(account *Account, state *GameState) error {
		if effectListContainsBuildingMutation(effects) {
			nextState, _ := settleResources(*state, now)
			*state = nextState
		}
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
		if effectListContainsBuildingMutation(effects) {
			nextState, _ := settleResources(*state, now)
			*state = nextState
		}
		after = snapshotCoreAssets(state)
		applyResult = result
		return nil
	}
	account, state, err := s.executeEffectsInAccountAssetTransaction(accountID, playerID, now, effects, updateFn)
	if err != nil {
		return AccountEffectExecutionResult{}, err
	}
	s.flushRewardSideEffects(applyResult.Reward)
	s.publishCoreAssetDiff(playerID, firstNonEmpty(ctx.RefType, "effect"), ctx.RefID, before, after, now)
	hydrateStateForResponse(&state, now)
	return AccountEffectExecutionResult{Account: account, State: state, Apply: applyResult}, nil
}

// executeEffectsInPlayerAssetTransaction 根据 Effect 类型选择最小可用的玩家资产事务。
func (s *Service) executeEffectsInPlayerAssetTransaction(playerID string, now time.Time, effects []Effect, update func(state *GameState) error) (GameState, error) {
	if effectListOnlyTouchesRewardAssets(effects) {
		return s.repo.UpdateRewardState(playerID, now, update)
	}
	if effectListOnlyTouchesBuildingAssets(effects) {
		return s.repo.UpdateBuildingResourceState(playerID, now, update)
	}
	return GameState{}, ErrMixedEffectAssets
}

// executeEffectsInAccountAssetTransaction 根据 Effect 类型选择最小可用的账号资产事务。
func (s *Service) executeEffectsInAccountAssetTransaction(accountID string, playerID string, now time.Time, effects []Effect, update func(account *Account, state *GameState) error) (Account, GameState, error) {
	if effectListOnlyTouchesRewardAssets(effects) {
		return s.repo.UpdateAccountRewardState(accountID, playerID, now, update)
	}
	return Account{}, GameState{}, ErrMixedEffectAssets
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

// effectListOnlyTouchesRewardAssets 判断效果列表是否只影响奖励资产。
func effectListOnlyTouchesRewardAssets(effects []Effect) bool {
	if len(effects) == 0 {
		return true
	}
	for _, effect := range effects {
		effectType := coreeffect.NormalizeType(effect.Type)
		if effectType != EffectTypeReward && effectType != EffectTypeModifier {
			return false
		}
	}
	return true
}

// effectListOnlyTouchesBuildingAssets 判断效果列表是否只影响建筑资产。
func effectListOnlyTouchesBuildingAssets(effects []Effect) bool {
	if len(effects) == 0 {
		return false
	}
	for _, effect := range effects {
		if coreeffect.NormalizeType(effect.Type) != EffectTypeBuildingMutation {
			return false
		}
	}
	return true
}

// effectListContainsBuildingMutation 判断效果列表是否包含建筑变更。
func effectListContainsBuildingMutation(effects []Effect) bool {
	for _, effect := range effects {
		if coreeffect.NormalizeType(effect.Type) == EffectTypeBuildingMutation {
			return true
		}
	}
	return false
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
