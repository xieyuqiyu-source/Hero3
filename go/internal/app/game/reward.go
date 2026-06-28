package game

import (
	"encoding/json"
	"strings"
	"time"

	corereward "hero3/internal/core/reward"
)

const (
	RewardTypeResource   = corereward.TypeResource
	RewardTypeCityGold   = corereward.TypeCityGold
	RewardTypeGold       = corereward.TypeGold
	RewardTypeItem       = corereward.TypeItem
	RewardTypeUnit       = corereward.TypeUnit
	RewardTypeGeneral    = corereward.TypeGeneral
	RewardTypeGeneralExp = corereward.TypeGeneralExp
	RewardTypeBuff       = corereward.TypeBuff
)

type Reward = corereward.Reward

type RewardGrantContext = corereward.GrantContext

type RewardApplyResult struct {
	Granted           map[string]int    `json:"granted"`
	AccountGold       int               `json:"accountGold,omitempty"`
	LedgerEntries     []GoldLedgerEntry `json:"ledgerEntries,omitempty"`
	ItemLedgerEntries []ItemLedgerEntry `json:"itemLedgerEntries,omitempty"`
	Events            []GameEvent       `json:"events,omitempty"`
}

type RewardGrantResult struct {
	State      GameState         `json:"state"`
	Account    Account           `json:"account"`
	Apply      RewardApplyResult `json:"apply"`
	AccountSet bool              `json:"accountSet"`
}

func ApplyRewardsToState(state *GameState, rewards []Reward, now time.Time) (RewardApplyResult, error) {
	return ApplyRewardsToStateWithContext(state, rewards, RewardGrantContext{}, now)
}

func ApplyRewardsToStateWithContext(state *GameState, rewards []Reward, ctx RewardGrantContext, now time.Time) (RewardApplyResult, error) {
	if state == nil {
		return RewardApplyResult{}, ErrPlayerNotFound
	}
	if err := ensureResourceState(state); err != nil {
		return RewardApplyResult{}, err
	}
	if state.Inventory == nil {
		state.Inventory = map[string]ItemStack{}
	}
	normalizeInventoryState(state, now)
	if now.IsZero() {
		now = time.Now()
	}

	result := RewardApplyResult{Granted: map[string]int{}}
	for _, reward := range rewards {
		rewardType := strings.TrimSpace(reward.Type)
		rewardID := strings.TrimSpace(reward.ID)
		if _, ok := GetRewardTypeDefinition(rewardType); !ok {
			return RewardApplyResult{}, ErrMailInvalidAttachment
		}
		if reward.Amount <= 0 {
			return RewardApplyResult{}, ErrMailInvalidAttachment
		}

		switch rewardType {
		case RewardTypeResource:
			if !isCoreResourceType(rewardID) {
				return RewardApplyResult{}, ErrMailInvalidAttachment
			}
			granted, _, err := addResourceCapped(state, rewardID, reward.Amount)
			if err != nil {
				return RewardApplyResult{}, err
			}
			result.Granted[rewardID] += granted
			result.Events = append(result.Events, buildRewardEvent(ctx, state.Player.ID, reward, granted, now))
		case RewardTypeCityGold:
			if rewardID != RewardTypeCityGold {
				return RewardApplyResult{}, ErrMailInvalidAttachment
			}
			state.CityGold += FlexInt(reward.Amount)
			result.Granted[RewardTypeCityGold] += reward.Amount
			result.LedgerEntries = append(result.LedgerEntries, GoldLedgerEntry{
				AccountID:    ctx.AccountID,
				PlayerID:     firstNonEmpty(ctx.PlayerID, state.Player.ID),
				Currency:     LedgerCurrencyCityGold,
				Direction:    LedgerDirectionCredit,
				Amount:       reward.Amount,
				BalanceAfter: int(state.CityGold),
				RefType:      ctx.RefType,
				RefID:        ctx.RefID,
				Reason:       ctx.Reason,
				CreatedAt:    now.UTC().Format(resourceDateLayout),
			})
			result.Events = append(result.Events, buildRewardEvent(ctx, state.Player.ID, reward, reward.Amount, now))
		case RewardTypeGold:
			if rewardID != RewardTypeGold {
				return RewardApplyResult{}, ErrMailInvalidAttachment
			}
			result.AccountGold += reward.Amount
			result.Granted[RewardTypeGold] += reward.Amount
			result.Events = append(result.Events, buildRewardEvent(ctx, state.Player.ID, reward, reward.Amount, now))
		case RewardTypeItem:
			if _, ok := GetItemDefinition(rewardID); !ok {
				return RewardApplyResult{}, ErrItemNotFound
			}
			beforeAmount := inventoryItemAmount(state, rewardID)
			if err := addItemToInventory(state, rewardID, reward.Amount, now); err != nil {
				return RewardApplyResult{}, err
			}
			afterAmount := inventoryItemAmount(state, rewardID)
			result.ItemLedgerEntries = append(result.ItemLedgerEntries, ItemLedgerEntry{
				ID:           "item_ledger_" + randomID(12),
				PlayerID:     state.Player.ID,
				ItemID:       rewardID,
				ChangeAmount: reward.Amount,
				BeforeAmount: beforeAmount,
				AfterAmount:  afterAmount,
				Reason:       firstNonEmpty(ctx.Reason, "reward"),
				RefType:      ctx.RefType,
				RefID:        ctx.RefID,
				Metadata:     reward.Metadata,
				CreatedAt:    now.UTC().Format(resourceDateLayout),
			})
			result.Granted[rewardID] += reward.Amount
			result.Events = append(result.Events, buildRewardEvent(ctx, state.Player.ID, reward, reward.Amount, now))
		case RewardTypeUnit:
			if _, ok := GetUnitConfig(state.Player.Faction, rewardID); !ok {
				return RewardApplyResult{}, ErrCrossFactionReward
			}
			AddArmyUnit(state, rewardID, reward.Amount)
			result.Granted[rewardID] += reward.Amount
			result.Events = append(result.Events, buildRewardEvent(ctx, state.Player.ID, reward, reward.Amount, now))
		case RewardTypeGeneral:
			if reward.Amount != 1 {
				return RewardApplyResult{}, ErrMailInvalidAttachment
			}
			if err := AddOwnedGeneral(state, rewardID, now); err != nil {
				return RewardApplyResult{}, err
			}
			result.Granted[rewardID] += 1
			result.Events = append(result.Events, buildRewardEvent(ctx, state.Player.ID, reward, 1, now))
		case RewardTypeGeneralExp:
			if rewardID != "" && rewardID != "current_general" && (state.General == nil || rewardID != state.General.ID) {
				return RewardApplyResult{}, ErrGeneralNotFound
			}
			if state.General == nil {
				return RewardApplyResult{}, ErrGeneralNotFound
			}
			expResult := applyGeneralBattleExp(state.General, reward.Amount)
			syncActiveGeneralToRoster(state)
			result.Granted[RewardTypeGeneralExp] += expResult.Gained
			refreshGeneralDerivedState(state, now)
			result.Events = append(result.Events, buildRewardEvent(ctx, state.Player.ID, reward, expResult.Gained, now))
		case RewardTypeBuff:
			buff, err := buildRewardBuff(reward, ctx, now)
			if err != nil {
				return RewardApplyResult{}, err
			}
			state.Buffs = append(state.Buffs, buff)
			result.Granted[rewardID] += reward.Amount
			refreshGeneralDerivedState(state, now)
			result.Events = append(result.Events, buildRewardEvent(ctx, state.Player.ID, reward, reward.Amount, now))
		default:
			return RewardApplyResult{}, ErrMailInvalidAttachment
		}
	}
	return result, nil
}

func rewardsFromMailAttachments(attachments []MailAttachment) []Reward {
	rewards := make([]Reward, 0, len(attachments))
	for _, attachment := range attachments {
		rewards = append(rewards, Reward{
			Type:     strings.TrimSpace(attachment.Type),
			ID:       strings.TrimSpace(attachment.ItemID),
			Amount:   attachment.Amount,
			Metadata: attachment.Metadata,
		})
	}
	return rewards
}

func rewardsRequireAccount(rewards []Reward) bool {
	for _, reward := range rewards {
		def, ok := GetRewardTypeDefinition(reward.Type)
		if ok && def.RequiresAccount {
			return true
		}
	}
	return false
}

func (s *Service) GrantRewards(playerID string, rewards []Reward, ctx RewardGrantContext) (RewardGrantResult, error) {
	playerID = strings.TrimSpace(playerID)
	if playerID == "" {
		return RewardGrantResult{}, ErrPlayerNotFound
	}
	ctx.PlayerID = firstNonEmpty(ctx.PlayerID, playerID)
	if strings.TrimSpace(ctx.AccountID) == "" && rewardsRequireAccount(rewards) {
		return RewardGrantResult{}, ErrAccountNotFound
	}
	now := time.Now()

	if ctx.AccountID != "" {
		account, state, applyResult, err := s.grantRewardsWithAccount(ctx.AccountID, playerID, rewards, ctx, now)
		if err != nil {
			return RewardGrantResult{}, err
		}
		s.flushRewardSideEffects(applyResult)
		return RewardGrantResult{State: state, Account: account, Apply: applyResult, AccountSet: true}, nil
	}

	state, applyResult, err := s.grantRewardsToPlayerState(playerID, rewards, ctx, now)
	if err != nil {
		return RewardGrantResult{}, err
	}
	if applyResult.AccountGold > 0 {
		return RewardGrantResult{}, ErrAccountNotFound
	}
	s.flushRewardSideEffects(applyResult)
	return RewardGrantResult{State: state, Apply: applyResult}, nil
}

func (s *Service) grantRewardsWithAccount(accountID string, playerID string, rewards []Reward, ctx RewardGrantContext, now time.Time) (Account, GameState, RewardApplyResult, error) {
	var applyResult RewardApplyResult
	account, state, err := s.repo.UpdateAccountRewardState(accountID, playerID, now, func(account *Account, state *GameState) error {
		result, err := ApplyRewardsToStateWithContext(state, rewards, ctx, now)
		if err != nil {
			return err
		}
		if result.AccountGold > 0 {
			account.Gold += result.AccountGold
			result.LedgerEntries = append(result.LedgerEntries, GoldLedgerEntry{
				AccountID:    accountID,
				PlayerID:     playerID,
				Currency:     LedgerCurrencyGold,
				Direction:    LedgerDirectionCredit,
				Amount:       result.AccountGold,
				BalanceAfter: account.Gold,
				RefType:      ctx.RefType,
				RefID:        ctx.RefID,
				Reason:       ctx.Reason,
				CreatedAt:    now.UTC().Format(resourceDateLayout),
			})
		}
		applyResult = result
		state.ServerTime = now.UTC().Format(resourceDateLayout)
		return nil
	})
	if err != nil {
		return Account{}, GameState{}, RewardApplyResult{}, err
	}
	return account, state, applyResult, nil
}

func (s *Service) grantRewardsToPlayerState(playerID string, rewards []Reward, ctx RewardGrantContext, now time.Time) (GameState, RewardApplyResult, error) {
	var applyResult RewardApplyResult
	state, err := s.repo.UpdateRewardState(playerID, now, func(state *GameState) error {
		result, err := ApplyRewardsToStateWithContext(state, rewards, ctx, now)
		if err != nil {
			return err
		}
		applyResult = result
		state.ServerTime = now.UTC().Format(resourceDateLayout)
		return nil
	})
	if err != nil {
		return GameState{}, RewardApplyResult{}, err
	}
	return state, applyResult, nil
}

func (s *Service) flushRewardSideEffects(result RewardApplyResult) {
	for _, entry := range result.LedgerEntries {
		s.recordLedger(entry)
	}
	for _, entry := range result.ItemLedgerEntries {
		_ = s.repo.WriteItemLedger(entry)
	}
	for _, event := range result.Events {
		s.publishEvent(event)
	}
}

func mergeRewardApplyResult(dst *RewardApplyResult, src RewardApplyResult) {
	if dst.Granted == nil {
		dst.Granted = map[string]int{}
	}
	for key, amount := range src.Granted {
		dst.Granted[key] += amount
	}
	dst.AccountGold += src.AccountGold
	dst.LedgerEntries = append(dst.LedgerEntries, src.LedgerEntries...)
	dst.ItemLedgerEntries = append(dst.ItemLedgerEntries, src.ItemLedgerEntries...)
	dst.Events = append(dst.Events, src.Events...)
}

func buildRewardEvent(ctx RewardGrantContext, fallbackPlayerID string, reward Reward, appliedAmount int, now time.Time) GameEvent {
	return GameEvent{
		Type:      EventRewardGranted,
		AccountID: ctx.AccountID,
		PlayerID:  firstNonEmpty(ctx.PlayerID, fallbackPlayerID),
		RefType:   ctx.RefType,
		RefID:     ctx.RefID,
		Payload: map[string]any{
			"rewardType":    strings.TrimSpace(reward.Type),
			"rewardId":      strings.TrimSpace(reward.ID),
			"amount":        reward.Amount,
			"appliedAmount": appliedAmount,
		},
		CreatedAt: now.UTC().Format(resourceDateLayout),
	}
}

func buildRewardBuff(reward Reward, ctx RewardGrantContext, now time.Time) (Buff, error) {
	key := strings.TrimSpace(reward.ID)
	if !IsValidStatKey(key) {
		return Buff{}, ErrInvalidStatKey
	}
	mode := rewardMetadataString(reward.Metadata, "mode", "percentAdd")
	if mode != "flat" && mode != "percentAdd" && mode != "percentMultiply" {
		return Buff{}, ErrInvalidStatKey
	}
	value, ok := rewardMetadataFloat(reward.Metadata, "value")
	if !ok {
		value = float64(reward.Amount)
	}
	source := rewardMetadataString(reward.Metadata, "source", "reward")
	note := rewardMetadataString(reward.Metadata, "note", ctx.Reason)
	buff := Buff{
		ID:        "buff_" + randomID(8),
		Source:    source,
		Key:       key,
		Value:     value,
		Mode:      mode,
		CreatedAt: now.UTC().Format(resourceDateLayout),
		Note:      note,
	}
	if hours := rewardMetadataInt(reward.Metadata, "hours", 0); hours > 0 {
		buff.ExpiresAt = now.Add(time.Duration(hours) * time.Hour).UTC().Format(resourceDateLayout)
	}
	return buff, nil
}

func rewardMetadataString(metadata map[string]any, key string, fallback string) string {
	if metadata == nil {
		return fallback
	}
	value, ok := metadata[key]
	if !ok {
		return fallback
	}
	text, ok := value.(string)
	if !ok || strings.TrimSpace(text) == "" {
		return fallback
	}
	return strings.TrimSpace(text)
}

func rewardMetadataFloat(metadata map[string]any, key string) (float64, bool) {
	if metadata == nil {
		return 0, false
	}
	switch value := metadata[key].(type) {
	case float64:
		return value, true
	case float32:
		return float64(value), true
	case int:
		return float64(value), true
	case int64:
		return float64(value), true
	case json.Number:
		parsed, err := value.Float64()
		return parsed, err == nil
	default:
		return 0, false
	}
}

func rewardMetadataInt(metadata map[string]any, key string, fallback int) int {
	value, ok := rewardMetadataFloat(metadata, key)
	if !ok {
		return fallback
	}
	return int(value)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
