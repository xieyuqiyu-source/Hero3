package game

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

// 本文件归口玩家背包道具数量变更 helper。

// normalizeInventoryState 同步背包格子和兼容聚合视图。
func normalizeInventoryState(state *GameState, now time.Time) {
	if state == nil {
		return
	}
	if len(state.InventorySlots) == 0 && len(state.Inventory) > 0 {
		for itemID, stack := range state.Inventory {
			stack.ItemID = firstNonEmpty(stack.ItemID, itemID)
			if stack.Amount <= 0 || strings.TrimSpace(stack.ItemID) == "" {
				continue
			}
			if stack.SlotID == "" {
				stack.SlotID = buildInventorySlotID(len(state.InventorySlots) + 1)
			}
			state.InventorySlots = append(state.InventorySlots, stack)
		}
	}
	state.InventorySlots = normalizeInventorySlots(state.InventorySlots, now)
	state.Inventory = aggregateInventorySlots(state.InventorySlots)
}

// addItemToInventory 给玩家背包增加道具。
func addItemToInventory(state *GameState, itemID string, amount int, now time.Time) error {
	if state == nil || itemID == "" || amount <= 0 {
		return nil
	}
	item, ok := GetItemDefinition(itemID)
	if !ok {
		return ErrItemNotFound
	}
	normalizeInventoryState(state, now)
	maxStack := item.MaxStack
	if maxStack <= 0 || !item.Stackable {
		maxStack = 1
	}
	if now.IsZero() {
		now = time.Now()
	}
	timestamp := now.UTC().Format(resourceDateLayout)
	for i := range state.InventorySlots {
		if amount <= 0 {
			break
		}
		if state.InventorySlots[i].ItemID != itemID || state.InventorySlots[i].Amount >= maxStack {
			continue
		}
		space := maxStack - state.InventorySlots[i].Amount
		added := minInt(space, amount)
		state.InventorySlots[i].Amount += added
		state.InventorySlots[i].UpdatedAt = timestamp
		amount -= added
	}
	for amount > 0 {
		if len(state.InventorySlots) >= InventoryCapacity {
			normalizeInventoryState(state, now)
			return ErrInventoryFull
		}
		added := minInt(maxStack, amount)
		state.InventorySlots = append(state.InventorySlots, ItemStack{
			SlotID:     nextInventorySlotID(state.InventorySlots),
			ItemID:     itemID,
			Amount:     added,
			ObtainedAt: timestamp,
			UpdatedAt:  timestamp,
		})
		amount -= added
	}
	normalizeInventoryState(state, now)
	return nil
}

// consumeItemFromInventory 从玩家背包消耗道具。
func consumeItemFromInventory(state *GameState, itemID string, amount int, now time.Time) bool {
	if state == nil || itemID == "" || amount <= 0 {
		return false
	}
	normalizeInventoryState(state, now)
	if state.Inventory[itemID].Amount < amount {
		return false
	}
	if now.IsZero() {
		now = time.Now()
	}
	timestamp := now.UTC().Format(resourceDateLayout)
	for i := range state.InventorySlots {
		if amount <= 0 {
			break
		}
		if state.InventorySlots[i].ItemID != itemID || state.InventorySlots[i].Amount <= 0 {
			continue
		}
		spent := minInt(state.InventorySlots[i].Amount, amount)
		state.InventorySlots[i].Amount -= spent
		state.InventorySlots[i].UpdatedAt = timestamp
		amount -= spent
	}
	normalizeInventoryState(state, now)
	return true
}

// aggregateInventorySlots 汇总格子背包，提供旧接口兼容视图。
func aggregateInventorySlots(slots []ItemStack) map[string]ItemStack {
	result := map[string]ItemStack{}
	for _, slot := range slots {
		itemID := strings.TrimSpace(slot.ItemID)
		if itemID == "" || slot.Amount <= 0 {
			continue
		}
		stack := result[itemID]
		stack.ItemID = itemID
		stack.Amount += slot.Amount
		if stack.ObtainedAt == "" || (slot.ObtainedAt != "" && slot.ObtainedAt < stack.ObtainedAt) {
			stack.ObtainedAt = slot.ObtainedAt
		}
		if slot.UpdatedAt > stack.UpdatedAt {
			stack.UpdatedAt = slot.UpdatedAt
		}
		result[itemID] = stack
	}
	return result
}

// AggregateInventorySlotsForStorage 为存储层提供背包格子聚合视图。
func AggregateInventorySlotsForStorage(slots []ItemStack) map[string]ItemStack {
	return aggregateInventorySlots(slots)
}

// normalizeInventorySlots 清理无效格子并补齐格子 ID。
func normalizeInventorySlots(slots []ItemStack, now time.Time) []ItemStack {
	if now.IsZero() {
		now = time.Now()
	}
	result := make([]ItemStack, 0, len(slots))
	used := map[string]struct{}{}
	for _, slot := range slots {
		slot.ItemID = strings.TrimSpace(slot.ItemID)
		if slot.ItemID == "" || slot.Amount <= 0 {
			continue
		}
		slot.SlotID = strings.TrimSpace(slot.SlotID)
		if slot.SlotID == "" {
			slot.SlotID = nextInventorySlotID(result)
		}
		if _, exists := used[slot.SlotID]; exists {
			slot.SlotID = nextInventorySlotID(result)
		}
		if slot.ObtainedAt == "" {
			slot.ObtainedAt = now.UTC().Format(resourceDateLayout)
		}
		if slot.UpdatedAt == "" {
			slot.UpdatedAt = slot.ObtainedAt
		}
		used[slot.SlotID] = struct{}{}
		result = append(result, slot)
	}
	sort.SliceStable(result, func(i, j int) bool {
		return result[i].SlotID < result[j].SlotID
	})
	return result
}

// nextInventorySlotID 生成当前背包中未使用的格子 ID。
func nextInventorySlotID(slots []ItemStack) string {
	used := map[string]struct{}{}
	for _, slot := range slots {
		if slot.SlotID != "" {
			used[slot.SlotID] = struct{}{}
		}
	}
	for index := 1; index <= InventoryCapacity; index++ {
		id := buildInventorySlotID(index)
		if _, exists := used[id]; !exists {
			return id
		}
	}
	return buildInventorySlotID(InventoryCapacity + 1)
}

// buildInventorySlotID 生成稳定格子 ID。
func buildInventorySlotID(index int) string {
	return fmt.Sprintf("slot_%04d", index)
}

func minInt(a int, b int) int {
	if a < b {
		return a
	}
	return b
}

// inventoryItemAmount 读取某个物品的聚合数量。
func inventoryItemAmount(state *GameState, itemID string) int {
	if state == nil {
		return 0
	}
	normalizeInventoryState(state, time.Now())
	return state.Inventory[itemID].Amount
}
