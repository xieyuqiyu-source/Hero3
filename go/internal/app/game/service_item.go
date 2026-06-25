package game

import (
	"strings"
	"time"
)

type UseItemResult struct {
	State   GameState      `json:"state"`
	ItemID  string         `json:"itemId"`
	Used    int            `json:"used"`
	Effects map[string]int `json:"effects"`
}

func (s *Service) ListItemsConfig() ItemsConfig {
	return GetItemsConfig()
}

func (s *Service) GrantItem(playerID string, itemID string, amount int) (GameState, error) {
	playerID = strings.TrimSpace(playerID)
	itemID = strings.TrimSpace(itemID)
	if playerID == "" {
		return GameState{}, ErrPlayerNotFound
	}
	if amount <= 0 {
		return GameState{}, ErrInvalidAmount
	}
	if _, ok := GetItemDefinition(itemID); !ok {
		return GameState{}, ErrItemNotFound
	}

	result, err := s.GrantRewards(playerID, []Reward{{
		Type:   RewardTypeItem,
		ID:     itemID,
		Amount: amount,
	}}, RewardGrantContext{
		PlayerID: playerID,
		RefType:  "admin_grant_item",
		RefID:    itemID,
		Reason:   "admin_grant_item",
	})
	if err != nil {
		return GameState{}, err
	}
	state := result.State
	hydrateStateForResponse(&state, time.Now())
	return state, nil
}

func (s *Service) UseItem(playerID string, itemID string, amount int) (UseItemResult, error) {
	playerID = strings.TrimSpace(playerID)
	itemID = strings.TrimSpace(itemID)
	if playerID == "" {
		return UseItemResult{}, ErrPlayerNotFound
	}
	if amount <= 0 {
		return UseItemResult{}, ErrInvalidAmount
	}
	item, ok := GetItemDefinition(itemID)
	if !ok {
		return UseItemResult{}, ErrItemNotFound
	}
	if !item.Usable {
		return UseItemResult{}, ErrItemNotUsable
	}

	lock := s.getPlayerLock(playerID)
	lock.Lock()
	defer lock.Unlock()

	now := time.Now()
	effects := map[string]int{}
	var before, after coreAssetSnapshot
	state, err := s.repo.UpdatePlayerState(playerID, now, func(state *GameState) error {
		nextState, _ := settleResources(*state, now)
		*state = nextState
		before = snapshotCoreAssets(state)
		if !consumeItemFromInventory(state, itemID, amount, now) {
			return ErrInsufficientItem
		}
		applied, err := applyItemEffects(state, item, amount)
		if err != nil {
			return err
		}
		effects = applied
		after = snapshotCoreAssets(state)
		return nil
	})
	if err != nil {
		return UseItemResult{}, err
	}
	s.publishEvent(GameEvent{
		Type:     EventItemUsed,
		PlayerID: playerID,
		RefType:  "item_use",
		RefID:    itemID,
		Payload: map[string]any{
			"itemId":  itemID,
			"amount":  amount,
			"effects": effects,
		},
		CreatedAt: now.UTC().Format(resourceDateLayout),
	})
	s.publishCoreAssetDiff(playerID, "item_use", itemID, before, after, now)
	hydrateStateForResponse(&state, now)
	return UseItemResult{State: state, ItemID: itemID, Used: amount, Effects: effects}, nil
}

func addItemToInventory(state *GameState, itemID string, amount int, now time.Time) {
	if state == nil || itemID == "" || amount <= 0 {
		return
	}
	if state.Inventory == nil {
		state.Inventory = map[string]ItemStack{}
	}
	stack := state.Inventory[itemID]
	stack.ItemID = itemID
	if stack.Amount <= 0 {
		stack.ObtainedAt = now.UTC().Format(resourceDateLayout)
	}
	stack.Amount += amount
	stack.UpdatedAt = now.UTC().Format(resourceDateLayout)
	state.Inventory[itemID] = stack
}

func consumeItemFromInventory(state *GameState, itemID string, amount int, now time.Time) bool {
	if state == nil || state.Inventory == nil || itemID == "" || amount <= 0 {
		return false
	}
	stack, ok := state.Inventory[itemID]
	if !ok || stack.Amount < amount {
		return false
	}
	stack.Amount -= amount
	if stack.Amount <= 0 {
		delete(state.Inventory, itemID)
		return true
	}
	stack.UpdatedAt = now.UTC().Format(resourceDateLayout)
	state.Inventory[itemID] = stack
	return true
}

func applyItemEffects(state *GameState, item ItemDefinition, count int) (map[string]int, error) {
	result := map[string]int{}
	if state.Resources.Items == nil {
		state.Resources.Items = map[string]int{}
	}
	for _, effect := range item.Effects {
		switch effect.Type {
		case "general_exp":
			if state.General == nil {
				return nil, ErrGeneralNotFound
			}
			gained := effect.Amount * count
			applyGeneralBattleExp(state.General, gained)
			result["general_exp"] += gained
		case "resources":
			for key, value := range effect.Resources {
				add := value * count
				capacity := state.Resources.Capacity[key]
				current := state.Resources.Items[key]
				next := current + add
				if capacity > 0 && next > capacity {
					next = capacity
				}
				state.Resources.Items[key] = next
				result[key] += next - current
			}
		case "unit_by_faction":
			unitID := strings.TrimSpace(effect.UnitByFaction[state.Player.Faction])
			if unitID == "" {
				return nil, ErrUnitNotFound
			}
			if _, ok := GetFactionUnits(state.Player.Faction)[unitID]; !ok {
				return nil, ErrUnitNotFound
			}
			amount := effect.Amount * count
			AddArmyUnit(state, unitID, amount)
			result[unitID] += amount
		default:
			return nil, ErrItemNotUsable
		}
	}
	return result, nil
}
