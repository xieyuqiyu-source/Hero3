package game

import (
	"strings"
	"time"
)

type UseItemResult struct {
	Patch   ItemActionResult `json:"patch"`
	ItemID  string           `json:"itemId"`
	Used    int              `json:"used"`
	Effects map[string]int   `json:"effects"`
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
	state, err := s.repo.UpdateItemState(playerID, now, func(state *GameState) error {
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
	if err := s.applyPvpProtectionItemEffects(playerID, item, amount, now); err != nil {
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
	return UseItemResult{Patch: BuildItemActionResult(state), ItemID: itemID, Used: amount, Effects: effects}, nil
}

// applyPvpProtectionItemEffects 应用道具中的 PVP 保护模块效果。
func (s *Service) applyPvpProtectionItemEffects(playerID string, item ItemDefinition, amount int, now time.Time) error {
	for _, effect := range item.Effects {
		if strings.TrimSpace(effect.Type) != "pvp_protection" {
			continue
		}
		duration := time.Duration(effect.DurationSeconds*amount) * time.Second
		if _, err := s.SetPvpProtection(playerID, effect.ProtectionType, duration, "item:"+item.ID, now); err != nil {
			return err
		}
	}
	return nil
}
