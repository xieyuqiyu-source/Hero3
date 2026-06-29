package game

import (
	"strings"
	"time"
)

type GeneralActionResult struct {
	State       GameState `json:"state"`
	AccountGold int       `json:"accountGold"`
}

func (s *Service) AllocateGeneralStat(playerID string, statKey string, amount int) (GameState, error) {
	playerID = strings.TrimSpace(playerID)
	statKey = strings.TrimSpace(statKey)
	if playerID == "" {
		return GameState{}, ErrPlayerNotFound
	}
	if !isValidGeneralStatKey(statKey) {
		return GameState{}, ErrInvalidStatKey
	}
	if amount <= 0 {
		amount = 1
	}

	now := time.Now()
	state, err := s.repo.UpdateGeneralState(playerID, now, func(state *GameState) error {
		nextState, _ := settleResources(*state, now)
		*state = nextState
		if state.General == nil {
			return ErrGeneralNotFound
		}

		applyHeroConfigToGeneral(state.General)
		if state.General.Stats[statKey]+amount > GeneralMaxStatPointsPerKey {
			return ErrStatMaxLevel
		}
		if state.General.AvailableStatPoints < amount {
			return ErrNoStatPoints
		}

		state.General.Stats[statKey] += amount
		applyHeroConfigToGeneral(state.General)
		syncActiveGeneralToRoster(state)
		refreshGeneralDerivedState(state, now)
		return nil
	})
	if err != nil {
		return GameState{}, err
	}
	return state, nil
}

func (s *Service) ResetGeneralStats(playerID string) (GeneralActionResult, error) {
	playerID = strings.TrimSpace(playerID)
	if playerID == "" {
		return GeneralActionResult{}, ErrPlayerNotFound
	}

	now := time.Now()
	accountID, err := s.repo.GetAccountIDByPlayerID(playerID)
	if err != nil {
		return GeneralActionResult{}, err
	}
	account, state, err := s.repo.UpdateAccountGeneralState(accountID, playerID, now, func(account *Account, state *GameState) error {
		nextState, _ := settleResources(*state, now)
		*state = nextState
		if state.General == nil {
			return ErrGeneralNotFound
		}
		if account.Gold < GeneralResetStatsGoldCost {
			return ErrInsufficientGold
		}
		account.Gold -= GeneralResetStatsGoldCost
		state.General.Stats = map[string]int{}
		applyHeroConfigToGeneral(state.General)
		syncActiveGeneralToRoster(state)
		refreshGeneralDerivedState(state, now)
		return nil
	})
	if err != nil {
		return GeneralActionResult{}, err
	}

	s.recordLedger(GoldLedgerEntry{
		AccountID:    accountID,
		PlayerID:     playerID,
		Currency:     LedgerCurrencyGold,
		Direction:    LedgerDirectionDebit,
		Amount:       GeneralResetStatsGoldCost,
		BalanceAfter: account.Gold,
		RefType:      LedgerRefGeneralReset,
		RefID:        state.General.ID,
		Reason:       "将领洗点",
	})
	s.publishCurrencyChanged(playerID, accountID, state.General.ID, LedgerRefGeneralReset)
	s.publishEvent(GameEvent{
		Type:      EventGeneralChanged,
		PlayerID:  playerID,
		AccountID: accountID,
		RefType:   LedgerRefGeneralReset,
		RefID:     state.General.ID,
		Payload: map[string]any{
			"action": "reset_stats",
		},
		CreatedAt: now.UTC().Format(resourceDateLayout),
	})

	return GeneralActionResult{State: state, AccountGold: account.Gold}, nil
}

func (s *Service) ChangeGeneral(playerID string, generalID string, itemID string) (GeneralActionResult, error) {
	playerID = strings.TrimSpace(playerID)
	generalID = strings.TrimSpace(generalID)
	itemID = strings.TrimSpace(itemID)
	if playerID == "" {
		return GeneralActionResult{}, ErrPlayerNotFound
	}
	if generalID == "" {
		return GeneralActionResult{}, ErrInvalidGeneral
	}

	now := time.Now()
	accountID, err := s.repo.GetAccountIDByPlayerID(playerID)
	if err != nil {
		return GeneralActionResult{}, err
	}
	beforeItemAmount := 0
	afterItemAmount := 0
	account, state, err := s.repo.UpdateAccountGeneralState(accountID, playerID, now, func(account *Account, state *GameState) error {
		nextState, _ := settleResources(*state, now)
		*state = nextState
		if state.General == nil {
			return ErrGeneralNotFound
		}
		if account.Gold < GeneralChangeGoldCost {
			return ErrInsufficientGold
		}
		if state.General.ID == generalID {
			return ErrInvalidGeneral
		}
		if !isGeneralAllowedForFaction(state.Player.Faction, generalID) {
			return ErrInvalidGeneral
		}
		hero, ok := GetHeroConfig(generalID)
		if !ok || !hero.Enabled || hero.Faction != state.Player.Faction {
			return ErrInvalidGeneral
		}
		if changeUntil, err := time.Parse(resourceDateLayout, strings.TrimSpace(state.GeneralChangeUntil)); err == nil && now.Before(changeUntil) {
			return ErrGeneralChangeCooldown
		}

		if itemID != "" {
			if _, ok := GetItemDefinition(itemID); !ok {
				return ErrItemNotFound
			}
			beforeItemAmount = inventoryItemAmount(state, itemID)
			if !consumeItemFromInventory(state, itemID, 1, now) {
				return ErrInsufficientItem
			}
			afterItemAmount = inventoryItemAmount(state, itemID)
		}

		account.Gold -= GeneralChangeGoldCost
		state.GeneralChangeUntil = now.Add(GeneralChangeCooldownHours * time.Hour).UTC().Format(resourceDateLayout)
		EnsureGeneralRoster(state, now)
		if _, owned := findOwnedGeneral(state.Generals, generalID); owned {
			if err := SetActiveGeneral(state, generalID, now); err != nil {
				return err
			}
		} else {
			state.General.ID = generalID
			state.General.Name = hero.Name
			state.General.Stats = map[string]int{}
			applyHeroConfigToGeneral(state.General)
			state.GeneralAssignments = upsertMainGeneralAssignment(state.GeneralAssignments, generalID, now)
			syncActiveGeneralToRoster(state)
		}
		refreshGeneralDerivedState(state, now)
		return nil
	})
	if err != nil {
		return GeneralActionResult{}, err
	}
	s.recordLedger(GoldLedgerEntry{
		AccountID:    accountID,
		PlayerID:     playerID,
		Currency:     LedgerCurrencyGold,
		Direction:    LedgerDirectionDebit,
		Amount:       GeneralChangeGoldCost,
		BalanceAfter: account.Gold,
		RefType:      "general_change",
		RefID:        generalID,
		Reason:       "更换将领",
	})
	s.publishCurrencyChanged(playerID, accountID, generalID, "general_change")
	s.publishEvent(GameEvent{
		Type:      EventGeneralChanged,
		PlayerID:  playerID,
		AccountID: accountID,
		RefType:   "general_change",
		RefID:     generalID,
		Payload: map[string]any{
			"action": "change_general",
			"cost":   GeneralChangeGoldCost,
		},
		CreatedAt: now.UTC().Format(resourceDateLayout),
	})
	if itemID != "" {
		_ = s.repo.WriteItemLedger(ItemLedgerEntry{
			ID:           "item_ledger_" + randomID(12),
			PlayerID:     playerID,
			ItemID:       itemID,
			ChangeAmount: -1,
			BeforeAmount: beforeItemAmount,
			AfterAmount:  afterItemAmount,
			Reason:       "item_use",
			RefType:      "general_change",
			RefID:        generalID,
			CreatedAt:    now.UTC().Format(resourceDateLayout),
		})
	}

	return GeneralActionResult{State: state, AccountGold: account.Gold}, nil
}

func isGeneralAllowedForFaction(faction string, generalID string) bool {
	factions := GetFactionsConfig()
	fc, ok := factions[faction]
	if !ok {
		return false
	}
	for _, g := range fc.Generals {
		if g.ID == generalID {
			return true
		}
	}
	return false
}

func refreshGeneralDerivedState(state *GameState, now time.Time) {
	if state == nil {
		return
	}
	modSources := CollectModifierSources(state)
	production := calculateResourceProduction(state.Buildings, state.General)
	state.ResourceProduction = applyProductionModifiers(production, now, modSources)
	capacity := calculateResourceCapacity(state.Buildings)
	_ = replaceResourceCapacity(state, applyCapacityModifiers(capacity, now, modSources))
	state.ActiveModifiers = GetModifierBreakdown(state, now)
	state.ServerTime = now.UTC().Format(resourceDateLayout)
}

func isValidGeneralStatKey(key string) bool {
	for _, statKey := range generalStatKeys {
		if key == statKey {
			return true
		}
	}
	return false
}
