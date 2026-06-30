// 本文件归口战斗所需的 MySQL 资产级事务，避免战斗服务直接走完整玩家状态事务。
package storage

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	"sort"
	"strings"
	"time"

	"hero3/internal/app/game"
)

// UpdateCombatState 只加载并锁定战斗必要资产，避免 NPC 战斗触碰建筑、资源田、征兵队列等无关写路径。
func (r *MySQLRepository) UpdateCombatState(playerID string, scope game.CombatAssetScope, updatedAt time.Time, update func(state *game.GameState) error) (game.GameState, error) {
	inventoryItemIDs := normalizeCombatScopeInventoryItemIDs(scope)
	tx, err := r.db.Begin()
	if err != nil {
		return game.GameState{}, err
	}
	defer func() { _ = tx.Rollback() }()

	var stateJSON []byte
	var mailCode string
	err = tx.QueryRow(
		`SELECT state_json, mail_code FROM players WHERE id = ? LIMIT 1`,
		playerID,
	).Scan(&stateJSON, &mailCode)
	if errors.Is(err, sql.ErrNoRows) {
		return game.GameState{}, game.ErrPlayerNotFound
	}
	if err != nil {
		return game.GameState{}, err
	}

	var state game.GameState
	if err = json.Unmarshal(stateJSON, &state); err != nil {
		return game.GameState{}, err
	}
	if state.Player.MailCode == "" {
		state.Player.MailCode = mailCode
	}
	if err := overlayAuthoritativeResourcesTx(tx, &state, playerID); err != nil {
		return game.GameState{}, err
	}
	if err := overlayAuthoritativeCurrencyTx(tx, &state, playerID, updatedAt); err != nil {
		return game.GameState{}, err
	}
	if err := overlayAuthoritativeNpcStateTx(tx, &state, playerID, updatedAt); err != nil {
		return game.GameState{}, err
	}
	if !scope.SkipInventory {
		if err := overlayCombatInventoryTx(tx, &state, playerID, inventoryItemIDs); err != nil {
			return game.GameState{}, err
		}
	}
	if err := overlayCombatArmyTx(tx, &state, playerID, scope); err != nil {
		return game.GameState{}, err
	}
	if err := overlayCombatGeneralsTx(tx, &state, playerID, scope); err != nil {
		return game.GameState{}, err
	}
	if err := overlayCombatReadOnlyState(tx, &state, playerID); err != nil {
		return game.GameState{}, err
	}

	previousResourceSnapshot := resourceSnapshotsFromStorageState(state.Resources)
	previousCurrencySnapshot := currencySnapshotFromState(state)
	previousInventorySnapshot := map[string]storageInventorySnapshot(nil)
	if !scope.SkipInventory {
		previousInventorySnapshot = inventorySnapshotsFromStorageStateWithSlots(state.Inventory, state.InventorySlots)
	}
	previousArmySnapshot := armySnapshotsFromStorageState(state.Army)
	previousGeneralSnapshot := generalSnapshotsFromStorageState(state.Generals)
	previousGeneralAssignmentSnapshot := generalAssignmentSnapshotsFromStorageState(state.GeneralAssignments)
	previousNpcStateJSON, err := json.Marshal(state.NpcState)
	if err != nil {
		return game.GameState{}, err
	}

	if update != nil {
		if err = update(&state); err != nil {
			return game.GameState{}, err
		}
	}

	nextStateJSON, err := marshalPlayerStateSnapshot(state)
	if err != nil {
		return game.GameState{}, err
	}
	if resourceSnapshotChanged(previousResourceSnapshot, state.Resources) {
		if err := syncPlayerResourcesTx(tx, playerID, state.Resources, updatedAt.UTC()); err != nil {
			return game.GameState{}, err
		}
	}
	if currencySnapshotChanged(previousCurrencySnapshot, state) {
		if err := syncPlayerCurrencyTx(tx, playerID, &state, updatedAt.UTC()); err != nil {
			return game.GameState{}, err
		}
	}
	if !scope.SkipInventory && inventorySnapshotChangedWithSlots(previousInventorySnapshot, state.Inventory, state.InventorySlots) {
		if len(inventoryItemIDs) > 0 {
			if err := syncPlayerInventoryScopedDeltaTx(tx, playerID, previousInventorySnapshot, state.Inventory, state.InventorySlots, inventoryItemIDs, updatedAt.UTC()); err != nil {
				return game.GameState{}, err
			}
		} else {
			if err := syncPlayerInventoryDeltaTx(tx, playerID, previousInventorySnapshot, state.Inventory, state.InventorySlots, updatedAt.UTC()); err != nil {
				return game.GameState{}, err
			}
		}
	}
	if armySnapshotChanged(previousArmySnapshot, state.Army) {
		if err := syncPlayerArmyDeltaTx(tx, playerID, previousArmySnapshot, state.Army, updatedAt.UTC()); err != nil {
			return game.GameState{}, err
		}
	}
	if generalSnapshotChanged(previousGeneralSnapshot, state.Generals) {
		if combatScopeHasAssets(scope) {
			if err := syncCombatGeneralsDeltaTx(tx, playerID, previousGeneralSnapshot, state.Generals, updatedAt.UTC()); err != nil {
				return game.GameState{}, err
			}
		} else {
			if err := syncPlayerGeneralsTx(tx, playerID, state.Generals, updatedAt.UTC()); err != nil {
				return game.GameState{}, err
			}
		}
	}
	if !combatScopeHasAssets(scope) && generalAssignmentSnapshotChanged(previousGeneralAssignmentSnapshot, state.GeneralAssignments) {
		if err := syncPlayerGeneralAssignmentsTx(tx, playerID, state.GeneralAssignments, updatedAt.UTC()); err != nil {
			return game.GameState{}, err
		}
	}
	nextNpcStateJSON, err := json.Marshal(state.NpcState)
	if err != nil {
		return game.GameState{}, err
	}
	if !bytes.Equal(previousNpcStateJSON, nextNpcStateJSON) {
		if err := syncPlayerNpcStateTx(tx, playerID, state.NpcState, updatedAt.UTC()); err != nil {
			return game.GameState{}, err
		}
	}
	if !bytes.Equal(stateJSON, nextStateJSON) {
		if _, err = tx.Exec(`UPDATE players SET state_json = ?, updated_at = ? WHERE id = ?`, nextStateJSON, updatedAt.UTC(), playerID); err != nil {
			return game.GameState{}, err
		}
	}
	if err = tx.Commit(); err != nil {
		return game.GameState{}, err
	}
	if scope.SkipInventory || len(inventoryItemIDs) > 0 {
		if err := r.overlayAuthoritativeInventory(&state, playerID); err != nil {
			return game.GameState{}, err
		}
	}
	if err := r.overlayAuthoritativeArmy(&state, playerID); err != nil {
		return game.GameState{}, err
	}
	if err := r.overlayAuthoritativeGenerals(&state, playerID); err != nil {
		return game.GameState{}, err
	}
	return state, nil
}

// overlayCombatInventoryTx 锁定候选掉落物品格子，并读取完整背包用于计算新格子 ID。
func overlayCombatInventoryTx(tx *sql.Tx, state *game.GameState, playerID string, itemIDs []string) error {
	if len(itemIDs) == 0 {
		return overlayAuthoritativeInventoryTx(tx, state, playerID)
	}
	for _, itemID := range itemIDs {
		if _, _, _, err := loadPlayerInventoryItemRowsTx(tx, playerID, itemID); err != nil {
			return err
		}
	}
	inventory, slots, found, err := loadPlayerInventory(tx, playerID)
	if err != nil {
		return err
	}
	return applyAuthoritativeInventory(state, inventory, slots, found)
}

// overlayCombatArmyTx 按战斗作用域锁定参战兵种；没有作用域时回退完整兵力锁定。
func overlayCombatArmyTx(tx *sql.Tx, state *game.GameState, playerID string, scope game.CombatAssetScope) error {
	unitTypes := normalizeCombatScopeUnitTypes(scope)
	if len(unitTypes) == 0 {
		return overlayAuthoritativeArmyTx(tx, state, playerID)
	}
	army, err := loadPlayerArmyUnitRowsTx(tx, playerID, unitTypes)
	if err != nil {
		return err
	}
	state.Army = army
	return nil
}

// normalizeCombatScopeUnitTypes 清理战斗事务兵种锁定范围。
func normalizeCombatScopeUnitTypes(scope game.CombatAssetScope) []string {
	seen := map[string]struct{}{}
	unitTypes := make([]string, 0, len(scope.UnitTypes))
	for _, unitType := range scope.UnitTypes {
		unitType = strings.TrimSpace(unitType)
		if unitType == "" {
			continue
		}
		if _, ok := seen[unitType]; ok {
			continue
		}
		seen[unitType] = struct{}{}
		unitTypes = append(unitTypes, unitType)
	}
	sort.Strings(unitTypes)
	return unitTypes
}

// normalizeCombatScopeInventoryItemIDs 清理战斗事务背包锁定范围。
func normalizeCombatScopeInventoryItemIDs(scope game.CombatAssetScope) []string {
	seen := map[string]struct{}{}
	itemIDs := make([]string, 0, len(scope.InventoryItemIDs))
	for _, itemID := range scope.InventoryItemIDs {
		itemID = strings.TrimSpace(itemID)
		if itemID == "" {
			continue
		}
		if _, ok := seen[itemID]; ok {
			continue
		}
		seen[itemID] = struct{}{}
		itemIDs = append(itemIDs, itemID)
	}
	sort.Strings(itemIDs)
	return itemIDs
}

// overlayCombatGeneralsTx 按战斗作用域锁定参战武将；空作用域回退完整武将锁定。
func overlayCombatGeneralsTx(tx *sql.Tx, state *game.GameState, playerID string, scope game.CombatAssetScope) error {
	generalIDs := normalizeCombatScopeGeneralIDs(scope)
	if len(generalIDs) == 0 {
		if combatScopeHasAssets(scope) {
			state.Generals = []game.General{}
			state.GeneralAssignments = []game.GeneralAssignment{}
			state.General = nil
			return nil
		}
		return overlayAuthoritativeGeneralsTx(tx, state, playerID)
	}
	generals, found, err := loadPlayerGeneralRowsTx(tx, playerID, generalIDs)
	if err != nil {
		return err
	}
	if !found {
		if state.General != nil || len(state.Generals) > 0 {
			return errPlayerGeneralsMissing
		}
		state.Generals = []game.General{}
		state.GeneralAssignments = []game.GeneralAssignment{}
		state.General = nil
		return nil
	}
	assignments, _, err := loadPlayerGeneralAssignmentsForGeneralsTx(tx, playerID, generalIDs)
	if err != nil {
		return err
	}
	state.Generals = generals
	state.GeneralAssignments = assignments
	state.General = nil
	return nil
}

// normalizeCombatScopeGeneralIDs 清理战斗事务武将锁定范围。
func normalizeCombatScopeGeneralIDs(scope game.CombatAssetScope) []string {
	seen := map[string]struct{}{}
	generalIDs := make([]string, 0, len(scope.GeneralIDs))
	for _, generalID := range scope.GeneralIDs {
		generalID = strings.TrimSpace(generalID)
		if generalID == "" {
			continue
		}
		if _, ok := seen[generalID]; ok {
			continue
		}
		seen[generalID] = struct{}{}
		generalIDs = append(generalIDs, generalID)
	}
	sort.Strings(generalIDs)
	return generalIDs
}

// combatScopeHasAssets 判断调用方是否已经传入明确战斗作用域。
func combatScopeHasAssets(scope game.CombatAssetScope) bool {
	return len(scope.UnitTypes) > 0 || len(scope.GeneralIDs) > 0 || len(scope.InventoryItemIDs) > 0
}

// syncPlayerInventoryScopedDeltaTx 只写回指定 item_id 的背包变化，避免候选掉落以外的格子被战斗事务刷新。
func syncPlayerInventoryScopedDeltaTx(tx *sql.Tx, playerID string, before map[string]storageInventorySnapshot, inventory map[string]game.ItemStack, slots []game.ItemStack, itemIDs []string, updatedAt time.Time) error {
	allowed := map[string]struct{}{}
	for _, itemID := range itemIDs {
		itemID = strings.TrimSpace(itemID)
		if itemID != "" {
			allowed[itemID] = struct{}{}
		}
	}
	if len(allowed) == 0 {
		return syncPlayerInventoryDeltaTx(tx, playerID, before, inventory, slots, updatedAt)
	}
	nextSlotSet := map[string]struct{}{}
	for _, stack := range inventorySlotsFromStorageState(inventory, slots, updatedAt) {
		if _, ok := allowed[stack.ItemID]; !ok {
			continue
		}
		nextSlotSet[stack.SlotID] = struct{}{}
		obtainedAt := parseInventoryTime(stack.ObtainedAt)
		stackUpdatedAt := parseInventoryTime(stack.UpdatedAt)
		if !stackUpdatedAt.Valid {
			stackUpdatedAt = sql.NullTime{Time: updatedAt.UTC(), Valid: true}
		}
		next := storageInventorySnapshot{
			SlotID:     stack.SlotID,
			ItemID:     stack.ItemID,
			Amount:     stack.Amount,
			ObtainedAt: stack.ObtainedAt,
			UpdatedAt:  stack.UpdatedAt,
		}
		if previous, ok := before[stack.SlotID]; ok && previous == next {
			continue
		}
		if _, err := tx.Exec(
			`INSERT INTO player_inventory (player_id, slot_id, item_id, amount, obtained_at, updated_at)
			 VALUES (?, ?, ?, ?, ?, ?)
			 ON DUPLICATE KEY UPDATE
				item_id = VALUES(item_id),
				amount = VALUES(amount),
				obtained_at = VALUES(obtained_at),
				updated_at = VALUES(updated_at)`,
			playerID,
			stack.SlotID,
			stack.ItemID,
			stack.Amount,
			nullableTimeArg(obtainedAt),
			nullableTimeArg(stackUpdatedAt),
		); err != nil {
			return err
		}
	}
	for slotID, previous := range before {
		if _, ok := allowed[previous.ItemID]; !ok {
			continue
		}
		if _, exists := nextSlotSet[slotID]; exists {
			continue
		}
		if _, err := tx.Exec(
			`DELETE FROM player_inventory WHERE player_id = ? AND slot_id = ?`,
			playerID,
			slotID,
		); err != nil {
			return err
		}
	}
	return nil
}

// syncCombatGeneralsDeltaTx 只写回作用域内发生变化的武将行，不删除未加载武将。
func syncCombatGeneralsDeltaTx(tx *sql.Tx, playerID string, before map[string]storageGeneralSnapshot, generals []game.General, updatedAt time.Time) error {
	after := generalSnapshotsFromStorageState(generals)
	byID := generalsByID(generals)
	generalIDs := make([]string, 0, len(after))
	for generalID := range after {
		generalIDs = append(generalIDs, generalID)
	}
	sort.Strings(generalIDs)
	for _, generalID := range generalIDs {
		next := after[generalID]
		if previous, ok := before[generalID]; ok && previous == next {
			continue
		}
		if err := syncPlayerGeneralRowTx(tx, playerID, byID[generalID], updatedAt.UTC()); err != nil {
			return err
		}
	}
	return nil
}

// overlayCombatReadOnlyState 非锁定读取战斗计算需要但不应在战斗事务中写回的资产快照。
func overlayCombatReadOnlyState(tx *sql.Tx, state *game.GameState, playerID string) error {
	buildings, found, err := loadPlayerBuildings(tx, playerID)
	if err != nil {
		return err
	}
	if err := applyAuthoritativeBuildings(state, buildings, found); err != nil {
		return err
	}
	resourceSlots, found, err := loadPlayerResourceSlots(tx, playerID)
	if err != nil {
		return err
	}
	if err := applyAuthoritativeResourceSlots(state, resourceSlots, found); err != nil {
		return err
	}
	recruitQueues, found, err := loadPlayerRecruitQueues(tx, playerID)
	if err != nil {
		return err
	}
	if err := applyAuthoritativeRecruitQueues(state, recruitQueues, found); err != nil {
		return err
	}
	buffs, found, err := loadPlayerBuffs(tx, playerID)
	if err != nil {
		return err
	}
	return applyAuthoritativeBuffs(state, buffs, found)
}
