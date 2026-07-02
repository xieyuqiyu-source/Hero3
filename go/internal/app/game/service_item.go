package game

import (
	"errors"
	"log/slog"
	"sort"
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

// ListDropPoolsConfig 返回当前掉落池配置。
func (s *Service) ListDropPoolsConfig() DropPoolsConfig {
	return GetDropPoolsConfig()
}

// UpdateDropPoolsConfig 保存并热更新掉落池配置。
func (s *Service) UpdateDropPoolsConfig(config DropPoolsConfig) error {
	if err := s.ValidateDropPoolsConfigForAdmin(config); err != nil {
		return err
	}
	return SaveDropPoolsConfig(s.dropPoolsPath, config)
}

// ValidateDropPoolsConfigForAdmin 校验 GM 提交的掉落池配置。
func (s *Service) ValidateDropPoolsConfigForAdmin(config DropPoolsConfig) error {
	if err := ValidateDropPoolsConfig(config); err != nil {
		return err
	}
	npcConfig := GetNpcConfig()
	for tier, tierConfig := range npcConfig.Tiers {
		dropPoolID := strings.TrimSpace(tierConfig.DropPoolID)
		if dropPoolID == "" {
			continue
		}
		if _, ok := config[dropPoolID]; !ok {
			return errors.New("NPC 层级 " + tier + " 引用的掉落池不存在: " + dropPoolID)
		}
	}
	return nil
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

	if expAmount, ok := generalExpItemAmount(item, amount); ok {
		return s.useGeneralExpItem(playerID, item, amount, expAmount)
	}

	now := time.Now()
	var before, after coreAssetSnapshot
	var effects map[string]int
	var state GameState
	var err error
	beforeItemAmount := 0
	afterItemAmount := 0
	for attempt := 0; attempt < 3; attempt++ {
		effects = map[string]int{}
		before = coreAssetSnapshot{}
		after = coreAssetSnapshot{}
		beforeItemAmount = 0
		afterItemAmount = 0
		scope := itemUseRewardAssetScope(item)
		state, err = s.repo.UpdateScopedRewardState(playerID, scope, now, func(state *GameState) error {
			if scope.Resources {
				nextState, _ := settleResources(*state, now)
				*state = nextState
			}
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
		if err == nil {
			break
		}
		if !isRetryableStorageConflict(err) || attempt == 2 {
			return UseItemResult{}, err
		}
		slog.Warn("item use transaction retry after storage conflict", "playerId", playerID, "itemId", itemID, "amount", amount, "attempt", attempt+1, "error", err)
		time.Sleep(time.Duration(attempt+1) * 80 * time.Millisecond)
	}
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

// itemUseRewardAssetScope 根据道具效果推导本次使用需要锁定和写回的资产范围。
func itemUseRewardAssetScope(item ItemDefinition) RewardAssetScope {
	scope := RewardAssetScope{}
	inventoryItems := map[string]struct{}{}
	itemID := strings.TrimSpace(item.ID)
	if itemID != "" {
		inventoryItems[itemID] = struct{}{}
	}
	for _, effect := range item.Effects {
		switch strings.TrimSpace(effect.Type) {
		case "item":
			if id := strings.TrimSpace(effect.ID); id != "" {
				inventoryItems[id] = struct{}{}
			}
		case "general":
			scope.AllGenerals = true
		case "general_exp":
			scope.CurrentGeneral = true
		case "resources":
			if len(effect.Resources) > 0 {
				scope.Resources = true
			}
		case "currency":
			if strings.TrimSpace(effect.CurrencyType) == RewardTypeCityGold {
				scope.Currency = true
			}
		case "unit_by_faction":
			for _, unitID := range effect.UnitByFaction {
				if unitID = strings.TrimSpace(unitID); unitID != "" {
					scope.UnitTypes = append(scope.UnitTypes, unitID)
				}
			}
		case "random_unit_by_faction_category", "all_units_by_faction_category":
			scope.AllArmy = true
		case "buff":
			scope.Buffs = true
		case "random_reward":
			collectDropPoolItemIDs(effect.DropPoolID, map[string]struct{}{}, inventoryItems)
		}
	}
	scope.InventoryItemIDs = itemIDsFromSet(inventoryItems)
	return scope
}

// itemIDsFromSet 把物品 ID 集合转回列表。
func itemIDsFromSet(items map[string]struct{}) []string {
	result := make([]string, 0, len(items))
	for itemID := range items {
		result = append(result, itemID)
	}
	sort.Strings(result)
	return result
}

// useGeneralExpItem 使用将领经验包，只进入背包 + 武将小事务。
func (s *Service) useGeneralExpItem(playerID string, item ItemDefinition, amount int, expAmount int) (UseItemResult, error) {
	now := time.Now()
	effects := map[string]int{}
	var state GameState
	var err error
	beforeItemAmount := 0
	afterItemAmount := 0
	for attempt := 0; attempt < 3; attempt++ {
		effects = map[string]int{}
		beforeItemAmount = 0
		afterItemAmount = 0
		state, err = s.repo.UpdateGeneralExpItemState(playerID, item.ID, now, func(state *GameState) error {
			beforeItemAmount = inventoryItemAmount(state, item.ID)
			if !consumeItemFromInventory(state, item.ID, amount, now) {
				return ErrInsufficientItem
			}
			if state.General == nil {
				return ErrGeneralNotFound
			}
			expResult := applyGeneralBattleExp(state.General, expAmount)
			syncActiveGeneralToRoster(state)
			effects[RewardTypeGeneralExp] = expResult.Gained
			afterItemAmount = inventoryItemAmount(state, item.ID)
			state.ServerTime = now.UTC().Format(resourceDateLayout)
			return nil
		})
		if err == nil {
			break
		}
		if !isRetryableStorageConflict(err) || attempt == 2 {
			return UseItemResult{}, err
		}
		slog.Warn("general exp item use transaction retry after storage conflict", "playerId", playerID, "itemId", item.ID, "amount", amount, "attempt", attempt+1, "error", err)
		time.Sleep(time.Duration(attempt+1) * 80 * time.Millisecond)
	}
	if err != nil {
		return UseItemResult{}, err
	}
	s.publishEvent(GameEvent{
		Type:     EventItemUsed,
		PlayerID: playerID,
		RefType:  "item_use",
		RefID:    item.ID,
		Payload: map[string]any{
			"itemId":  item.ID,
			"amount":  amount,
			"effects": effects,
		},
		CreatedAt: now.UTC().Format(resourceDateLayout),
	})
	_ = s.repo.WriteItemLedger(ItemLedgerEntry{
		ID:           "item_ledger_" + randomID(12),
		PlayerID:     playerID,
		ItemID:       item.ID,
		ChangeAmount: -amount,
		BeforeAmount: beforeItemAmount,
		AfterAmount:  afterItemAmount,
		Reason:       "item_use",
		RefType:      "item_use",
		RefID:        item.ID,
		CreatedAt:    now.UTC().Format(resourceDateLayout),
	})
	state.ServerTime = now.UTC().Format(resourceDateLayout)
	normalizeInventoryState(&state, now)
	return UseItemResult{Patch: BuildGeneralExpItemActionResult(state), ItemID: item.ID, Used: amount, Effects: effects}, nil
}

// generalExpItemAmount 判断道具是否为纯将领经验包，并返回本次总经验。
func generalExpItemAmount(item ItemDefinition, count int) (int, bool) {
	total := 0
	for _, effect := range item.Effects {
		if strings.TrimSpace(effect.Type) != "general_exp" {
			return 0, false
		}
		total += effect.Amount * count
	}
	return total, total > 0
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
