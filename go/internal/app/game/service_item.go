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

// UpdateItemsConfig 保存并热更新物品配置。
func (s *Service) UpdateItemsConfig(config ItemsConfig) error {
	current := GetItemsConfig()
	for itemID := range current {
		if _, ok := config[itemID]; !ok {
			return ErrItemIDLocked
		}
	}
	return SaveItemsConfig(s.itemsPath, config)
}

// ValidateItemsConfigForAdmin 校验 GM 提交的物品配置。
func (s *Service) ValidateItemsConfigForAdmin(config ItemsConfig) error {
	return ValidateItemsConfig(config)
}

// ListItemLedger 查询物品流水。
func (s *Service) ListItemLedger(filter ItemLedgerFilter) (ItemLedgerPage, error) {
	entries, total, err := s.repo.ListItemLedger(filter)
	if err != nil {
		return ItemLedgerPage{}, err
	}
	limit := filter.Limit
	if limit <= 0 || limit > 1000 {
		limit = 200
	}
	offset := filter.Offset
	if offset < 0 {
		offset = 0
	}
	return ItemLedgerPage{Entries: entries, Total: total, Limit: limit, Offset: offset}, nil
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
	beforeItemAmount := 0
	afterItemAmount := 0
	state, err := s.repo.UpdateItemState(playerID, now, func(state *GameState) error {
		nextState, _ := settleResources(*state, now)
		*state = nextState
		before = snapshotCoreAssets(state)
		beforeItemAmount = inventoryItemAmount(state, itemID)
		if !consumeItemFromInventory(state, itemID, amount, now) {
			return ErrInsufficientItem
		}
		applied, err := applyItemEffects(state, item, amount)
		if err != nil {
			return err
		}
		effects = applied
		afterItemAmount = inventoryItemAmount(state, itemID)
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
	_ = s.repo.WriteItemLedger(ItemLedgerEntry{
		ID:           "item_ledger_" + randomID(12),
		PlayerID:     playerID,
		ItemID:       itemID,
		ChangeAmount: -amount,
		BeforeAmount: beforeItemAmount,
		AfterAmount:  afterItemAmount,
		Reason:       "item_use",
		RefType:      "item_use",
		RefID:        itemID,
		CreatedAt:    now.UTC().Format(resourceDateLayout),
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
