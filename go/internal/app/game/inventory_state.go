package game

import "time"

// 本文件归口玩家背包道具数量变更 helper。

// addItemToInventory 给玩家背包增加道具。
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

// consumeItemFromInventory 从玩家背包消耗道具。
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
