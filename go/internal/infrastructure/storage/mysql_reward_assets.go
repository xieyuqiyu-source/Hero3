// 本文件归口奖励发放所需的 MySQL 资产级事务，避免走完整玩家状态事务。
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

// UpdateRewardState 只加载奖励可能影响的资源、背包、兵力、武将和 Buff 资产。
func (r *MySQLRepository) UpdateRewardState(playerID string, updatedAt time.Time, update func(state *game.GameState) error) (game.GameState, error) {
	return r.updateRewardStateWithScope(playerID, game.RewardAssetScope{
		Resources:    true,
		Currency:     true,
		AllInventory: true,
		AllArmy:      true,
		AllGenerals:  true,
		Buffs:        true,
	}, updatedAt, update)
}

// UpdateScopedRewardState 按奖励类型锁定和写回必要资产，避免资源奖励触碰货币、背包、兵力、武将和 Buff。
func (r *MySQLRepository) UpdateScopedRewardState(playerID string, scope game.RewardAssetScope, updatedAt time.Time, update func(state *game.GameState) error) (game.GameState, error) {
	return r.updateRewardStateWithScope(playerID, scope, updatedAt, update)
}

// updateRewardStateWithScope 执行玩家奖励事务；调用方通过 scope 明确资产范围。
func (r *MySQLRepository) updateRewardStateWithScope(playerID string, scope game.RewardAssetScope, updatedAt time.Time, update func(state *game.GameState) error) (game.GameState, error) {
	inventoryItemIDs := normalizeRewardScopeStrings(scope.InventoryItemIDs)
	unitTypes := normalizeRewardScopeStrings(scope.UnitTypes)
	generalIDs := normalizeRewardScopeStrings(scope.GeneralIDs)
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
	if scope.Currency {
		if err := overlayAuthoritativeCurrencyTx(tx, &state, playerID, updatedAt); err != nil {
			return game.GameState{}, err
		}
	}
	if scope.Resources {
		if err := overlayAuthoritativeResourcesTx(tx, &state, playerID); err != nil {
			return game.GameState{}, err
		}
	}
	if scope.AllInventory {
		if err := overlayAuthoritativeInventoryTx(tx, &state, playerID); err != nil {
			return game.GameState{}, err
		}
	} else if len(inventoryItemIDs) > 0 {
		if err := overlayCombatInventoryTx(tx, &state, playerID, inventoryItemIDs); err != nil {
			return game.GameState{}, err
		}
	}
	if scope.AllArmy {
		if err := overlayAuthoritativeArmyTx(tx, &state, playerID); err != nil {
			return game.GameState{}, err
		}
	} else if len(unitTypes) > 0 {
		army, err := loadPlayerArmyUnitRowsTx(tx, playerID, unitTypes)
		if err != nil {
			return game.GameState{}, err
		}
		state.Army = army
	}
	if scope.AllGenerals {
		if err := overlayAuthoritativeGeneralsTx(tx, &state, playerID); err != nil {
			return game.GameState{}, err
		}
	} else if scope.CurrentGeneral || len(generalIDs) > 0 {
		if err := overlayRewardGeneralsTx(tx, &state, playerID, generalIDs, scope.CurrentGeneral); err != nil {
			return game.GameState{}, err
		}
	}
	if scope.Buffs {
		if err := overlayAuthoritativeBuffsTx(tx, &state, playerID); err != nil {
			return game.GameState{}, err
		}
	}

	var previousResourceSnapshot map[string]storageResourceSnapshot
	var previousCurrencySnapshot playerCurrencySnapshot
	if scope.Currency {
		previousCurrencySnapshot = currencySnapshotFromState(state)
	}
	if scope.Resources {
		previousResourceSnapshot = resourceSnapshotsFromStorageState(state.Resources)
	}
	var previousInventorySnapshot map[string]storageInventorySnapshot
	if scope.AllInventory || len(inventoryItemIDs) > 0 {
		previousInventorySnapshot = inventorySnapshotsFromStorageStateWithSlots(state.Inventory, state.InventorySlots)
	}
	var previousArmySnapshot map[string]storageArmySnapshot
	if scope.AllArmy || len(unitTypes) > 0 {
		previousArmySnapshot = armySnapshotsFromStorageState(state.Army)
	}
	var previousGeneralSnapshot map[string]storageGeneralSnapshot
	var previousGeneralAssignmentSnapshot map[string]storageGeneralAssignmentSnapshot
	if scope.AllGenerals || scope.CurrentGeneral || len(generalIDs) > 0 {
		previousGeneralSnapshot = generalSnapshotsFromStorageState(state.Generals)
		previousGeneralAssignmentSnapshot = generalAssignmentSnapshotsFromStorageState(state.GeneralAssignments)
	}
	var previousBuffSnapshot map[string]storageBuffSnapshot
	if scope.Buffs {
		previousBuffSnapshot = buffSnapshotsFromStorageState(state.Buffs)
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
	if scope.Resources && resourceSnapshotChanged(previousResourceSnapshot, state.Resources) {
		if err := syncPlayerResourcesTx(tx, playerID, state.Resources, updatedAt.UTC()); err != nil {
			return game.GameState{}, err
		}
	}
	if scope.Currency && currencySnapshotChanged(previousCurrencySnapshot, state) {
		if err := syncPlayerCurrencyTx(tx, playerID, &state, updatedAt.UTC()); err != nil {
			return game.GameState{}, err
		}
	}
	if (scope.AllInventory || len(inventoryItemIDs) > 0) && inventorySnapshotChangedWithSlots(previousInventorySnapshot, state.Inventory, state.InventorySlots) {
		if len(inventoryItemIDs) > 0 && !scope.AllInventory {
			if err := syncPlayerInventoryScopedDeltaTx(tx, playerID, previousInventorySnapshot, state.Inventory, state.InventorySlots, inventoryItemIDs, updatedAt.UTC()); err != nil {
				return game.GameState{}, err
			}
		} else {
			if err := syncPlayerInventoryDeltaTx(tx, playerID, previousInventorySnapshot, state.Inventory, state.InventorySlots, updatedAt.UTC()); err != nil {
				return game.GameState{}, err
			}
		}
	}
	if (scope.AllArmy || len(unitTypes) > 0) && armySnapshotChanged(previousArmySnapshot, state.Army) {
		if err := syncPlayerArmyDeltaTx(tx, playerID, previousArmySnapshot, state.Army, updatedAt.UTC()); err != nil {
			return game.GameState{}, err
		}
	}
	if (scope.AllGenerals || scope.CurrentGeneral || len(generalIDs) > 0) && generalSnapshotChanged(previousGeneralSnapshot, state.Generals) {
		if scope.AllGenerals {
			if err := syncPlayerGeneralsTx(tx, playerID, state.Generals, updatedAt.UTC()); err != nil {
				return game.GameState{}, err
			}
		} else {
			if err := syncCombatGeneralsDeltaTx(tx, playerID, previousGeneralSnapshot, state.Generals, updatedAt.UTC()); err != nil {
				return game.GameState{}, err
			}
		}
	}
	if scope.AllGenerals && generalAssignmentSnapshotChanged(previousGeneralAssignmentSnapshot, state.GeneralAssignments) {
		if err := syncPlayerGeneralAssignmentsTx(tx, playerID, state.GeneralAssignments, updatedAt.UTC()); err != nil {
			return game.GameState{}, err
		}
	}
	if scope.Buffs && buffSnapshotChanged(previousBuffSnapshot, state.Buffs) {
		if err := syncPlayerBuffsTx(tx, playerID, state.Buffs, updatedAt.UTC()); err != nil {
			return game.GameState{}, err
		}
	}
	if !bytes.Equal(stateJSON, nextStateJSON) {
		result, err := tx.Exec(
			`UPDATE players
			 SET nickname = ?, faction = ?, mail_code = ?, state_json = ?, updated_at = ?
			 WHERE id = ?`,
			state.Player.Nickname,
			state.Player.Faction,
			state.Player.MailCode,
			nextStateJSON,
			updatedAt.UTC(),
			playerID,
		)
		if err != nil {
			return game.GameState{}, err
		}
		if affected, err := result.RowsAffected(); err != nil {
			return game.GameState{}, err
		} else if affected == 0 {
			return game.GameState{}, game.ErrPlayerNotFound
		}
	}
	if err = tx.Commit(); err != nil {
		return game.GameState{}, err
	}
	if len(inventoryItemIDs) > 0 && !scope.AllInventory {
		if err := r.overlayAuthoritativeInventory(&state, playerID); err != nil {
			return game.GameState{}, err
		}
	}
	if len(unitTypes) > 0 && !scope.AllArmy {
		if err := r.overlayAuthoritativeArmy(&state, playerID); err != nil {
			return game.GameState{}, err
		}
	}
	if (scope.CurrentGeneral || len(generalIDs) > 0) && !scope.AllGenerals {
		if err := r.overlayAuthoritativeGenerals(&state, playerID); err != nil {
			return game.GameState{}, err
		}
	}
	if scope.Buffs {
		if err := r.overlayAuthoritativeBuffs(&state, playerID); err != nil {
			return game.GameState{}, err
		}
	}
	if !scope.Currency {
		if err := r.overlayAuthoritativeCurrency(&state, playerID); err != nil {
			return game.GameState{}, err
		}
	}
	return state, nil
}

// normalizeRewardScopeStrings 清理奖励作用域里的 ID 列表。
func normalizeRewardScopeStrings(values []string) []string {
	seen := map[string]struct{}{}
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

// overlayRewardGeneralsTx 按奖励作用域锁定目标武将；current_general 会解析当前主将。
func overlayRewardGeneralsTx(tx *sql.Tx, state *game.GameState, playerID string, generalIDs []string, includeCurrent bool) error {
	ids := append([]string(nil), generalIDs...)
	var currentGeneralID string
	if includeCurrent {
		assignment, found, err := loadMainGeneralAssignmentRowTx(tx, playerID)
		if err != nil {
			return err
		}
		if found {
			currentGeneralID = strings.TrimSpace(assignment.GeneralID)
			if currentGeneralID != "" {
				ids = append(ids, currentGeneralID)
				state.GeneralAssignments = []game.GeneralAssignment{assignment}
			}
		}
	}
	ids = normalizeRewardScopeStrings(ids)
	if len(ids) == 0 {
		state.Generals = []game.General{}
		state.General = nil
		if state.GeneralAssignments == nil {
			state.GeneralAssignments = []game.GeneralAssignment{}
		}
		return nil
	}
	generals, _, err := loadPlayerGeneralRowsTx(tx, playerID, ids)
	if err != nil {
		return err
	}
	assignments, _, err := loadPlayerGeneralAssignmentsForGeneralsTx(tx, playerID, ids)
	if err != nil {
		return err
	}
	if len(assignments) > 0 {
		state.GeneralAssignments = assignments
	}
	state.Generals = generals
	state.General = nil
	if currentGeneralID != "" {
		for i := range generals {
			if generals[i].ID == currentGeneralID {
				general := generals[i]
				state.General = &general
				break
			}
		}
	}
	return nil
}

// UpdateAccountRewardState 在账号 + 奖励资产级事务中发放含账号金币的奖励。
func (r *MySQLRepository) UpdateAccountRewardState(accountID string, playerID string, updatedAt time.Time, update func(account *game.Account, state *game.GameState) error) (game.Account, game.GameState, error) {
	tx, err := r.db.Begin()
	if err != nil {
		return game.Account{}, game.GameState{}, err
	}
	defer func() { _ = tx.Rollback() }()

	var account game.Account
	err = tx.QueryRow(
		`SELECT id, username, password_hash, gold, created_at
		 FROM accounts
		 WHERE id = ?
		 LIMIT 1
		 FOR UPDATE`,
		accountID,
	).Scan(&account.ID, &account.Username, &account.PasswordHash, &account.Gold, &account.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return game.Account{}, game.GameState{}, game.ErrAccountNotFound
	}
	if err != nil {
		return game.Account{}, game.GameState{}, err
	}
	previousAccountGold := account.Gold

	var stateJSON []byte
	var mailCode string
	err = tx.QueryRow(
		`SELECT state_json, mail_code
		 FROM players
		 WHERE id = ? AND account_id = ?
		 LIMIT 1`,
		playerID,
		accountID,
	).Scan(&stateJSON, &mailCode)
	if errors.Is(err, sql.ErrNoRows) {
		return game.Account{}, game.GameState{}, game.ErrPlayerNotFound
	}
	if err != nil {
		return game.Account{}, game.GameState{}, err
	}

	var state game.GameState
	if err = json.Unmarshal(stateJSON, &state); err != nil {
		return game.Account{}, game.GameState{}, err
	}
	if state.Player.MailCode == "" {
		state.Player.MailCode = mailCode
	}
	if err := overlayAuthoritativeCurrencyTx(tx, &state, playerID, updatedAt); err != nil {
		return game.Account{}, game.GameState{}, err
	}
	if err := overlayAuthoritativeResourcesTx(tx, &state, playerID); err != nil {
		return game.Account{}, game.GameState{}, err
	}
	if err := overlayAuthoritativeInventoryTx(tx, &state, playerID); err != nil {
		return game.Account{}, game.GameState{}, err
	}
	if err := overlayAuthoritativeArmyTx(tx, &state, playerID); err != nil {
		return game.Account{}, game.GameState{}, err
	}
	if err := overlayAuthoritativeGeneralsTx(tx, &state, playerID); err != nil {
		return game.Account{}, game.GameState{}, err
	}
	if err := overlayAuthoritativeBuffsTx(tx, &state, playerID); err != nil {
		return game.Account{}, game.GameState{}, err
	}

	previousResourceSnapshot := resourceSnapshotsFromStorageState(state.Resources)
	previousCurrencySnapshot := currencySnapshotFromState(state)
	previousInventorySnapshot := inventorySnapshotsFromStorageStateWithSlots(state.Inventory, state.InventorySlots)
	previousArmySnapshot := armySnapshotsFromStorageState(state.Army)
	previousGeneralSnapshot := generalSnapshotsFromStorageState(state.Generals)
	previousGeneralAssignmentSnapshot := generalAssignmentSnapshotsFromStorageState(state.GeneralAssignments)
	previousBuffSnapshot := buffSnapshotsFromStorageState(state.Buffs)

	if update != nil {
		if err = update(&account, &state); err != nil {
			return game.Account{}, game.GameState{}, err
		}
	}

	nextStateJSON, err := marshalPlayerStateSnapshot(state)
	if err != nil {
		return game.Account{}, game.GameState{}, err
	}
	if resourceSnapshotChanged(previousResourceSnapshot, state.Resources) {
		if err := syncPlayerResourcesTx(tx, playerID, state.Resources, updatedAt.UTC()); err != nil {
			return game.Account{}, game.GameState{}, err
		}
	}
	if currencySnapshotChanged(previousCurrencySnapshot, state) {
		if err := syncPlayerCurrencyTx(tx, playerID, &state, updatedAt.UTC()); err != nil {
			return game.Account{}, game.GameState{}, err
		}
	}
	if inventorySnapshotChangedWithSlots(previousInventorySnapshot, state.Inventory, state.InventorySlots) {
		if err := syncPlayerInventoryDeltaTx(tx, playerID, previousInventorySnapshot, state.Inventory, state.InventorySlots, updatedAt.UTC()); err != nil {
			return game.Account{}, game.GameState{}, err
		}
	}
	if armySnapshotChanged(previousArmySnapshot, state.Army) {
		if err := syncPlayerArmyTx(tx, playerID, state.Army, updatedAt.UTC()); err != nil {
			return game.Account{}, game.GameState{}, err
		}
	}
	if generalSnapshotChanged(previousGeneralSnapshot, state.Generals) {
		if err := syncPlayerGeneralsTx(tx, playerID, state.Generals, updatedAt.UTC()); err != nil {
			return game.Account{}, game.GameState{}, err
		}
	}
	if generalAssignmentSnapshotChanged(previousGeneralAssignmentSnapshot, state.GeneralAssignments) {
		if err := syncPlayerGeneralAssignmentsTx(tx, playerID, state.GeneralAssignments, updatedAt.UTC()); err != nil {
			return game.Account{}, game.GameState{}, err
		}
	}
	if buffSnapshotChanged(previousBuffSnapshot, state.Buffs) {
		if err := syncPlayerBuffsTx(tx, playerID, state.Buffs, updatedAt.UTC()); err != nil {
			return game.Account{}, game.GameState{}, err
		}
	}
	if account.Gold != previousAccountGold {
		if _, err = tx.Exec(
			`UPDATE accounts SET gold = ? WHERE id = ?`,
			account.Gold,
			accountID,
		); err != nil {
			return game.Account{}, game.GameState{}, err
		}
	}
	if !bytes.Equal(stateJSON, nextStateJSON) {
		result, err := tx.Exec(
			`UPDATE players
			 SET nickname = ?, faction = ?, mail_code = ?, state_json = ?, updated_at = ?
			 WHERE id = ? AND account_id = ?`,
			state.Player.Nickname,
			state.Player.Faction,
			state.Player.MailCode,
			nextStateJSON,
			updatedAt.UTC(),
			playerID,
			accountID,
		)
		if err != nil {
			return game.Account{}, game.GameState{}, err
		}
		if affected, err := result.RowsAffected(); err != nil {
			return game.Account{}, game.GameState{}, err
		} else if affected == 0 {
			return game.Account{}, game.GameState{}, game.ErrPlayerNotFound
		}
	}
	if err = tx.Commit(); err != nil {
		return game.Account{}, game.GameState{}, err
	}
	return account, state, nil
}

// UpdateScopedAccountRewardState 在账号锁定基础上按奖励作用域加载玩家资产。
func (r *MySQLRepository) UpdateScopedAccountRewardState(accountID string, playerID string, scope game.RewardAssetScope, updatedAt time.Time, update func(account *game.Account, state *game.GameState) error) (game.Account, game.GameState, error) {
	inventoryItemIDs := normalizeRewardScopeStrings(scope.InventoryItemIDs)
	unitTypes := normalizeRewardScopeStrings(scope.UnitTypes)
	generalIDs := normalizeRewardScopeStrings(scope.GeneralIDs)
	tx, err := r.db.Begin()
	if err != nil {
		return game.Account{}, game.GameState{}, err
	}
	defer func() { _ = tx.Rollback() }()

	var account game.Account
	err = tx.QueryRow(
		`SELECT id, username, password_hash, gold, created_at
		 FROM accounts
		 WHERE id = ?
		 LIMIT 1
		 FOR UPDATE`,
		accountID,
	).Scan(&account.ID, &account.Username, &account.PasswordHash, &account.Gold, &account.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return game.Account{}, game.GameState{}, game.ErrAccountNotFound
	}
	if err != nil {
		return game.Account{}, game.GameState{}, err
	}
	previousAccountGold := account.Gold

	var stateJSON []byte
	var mailCode string
	err = tx.QueryRow(
		`SELECT state_json, mail_code
		 FROM players
		 WHERE id = ? AND account_id = ?
		 LIMIT 1`,
		playerID,
		accountID,
	).Scan(&stateJSON, &mailCode)
	if errors.Is(err, sql.ErrNoRows) {
		return game.Account{}, game.GameState{}, game.ErrPlayerNotFound
	}
	if err != nil {
		return game.Account{}, game.GameState{}, err
	}

	var state game.GameState
	if err = json.Unmarshal(stateJSON, &state); err != nil {
		return game.Account{}, game.GameState{}, err
	}
	if state.Player.MailCode == "" {
		state.Player.MailCode = mailCode
	}
	if scope.Currency {
		if err := overlayAuthoritativeCurrencyTx(tx, &state, playerID, updatedAt); err != nil {
			return game.Account{}, game.GameState{}, err
		}
	}
	if scope.Resources {
		if err := overlayAuthoritativeResourcesTx(tx, &state, playerID); err != nil {
			return game.Account{}, game.GameState{}, err
		}
	}
	if scope.AllInventory {
		if err := overlayAuthoritativeInventoryTx(tx, &state, playerID); err != nil {
			return game.Account{}, game.GameState{}, err
		}
	} else if len(inventoryItemIDs) > 0 {
		if err := overlayCombatInventoryTx(tx, &state, playerID, inventoryItemIDs); err != nil {
			return game.Account{}, game.GameState{}, err
		}
	}
	if scope.AllArmy {
		if err := overlayAuthoritativeArmyTx(tx, &state, playerID); err != nil {
			return game.Account{}, game.GameState{}, err
		}
	} else if len(unitTypes) > 0 {
		army, err := loadPlayerArmyUnitRowsTx(tx, playerID, unitTypes)
		if err != nil {
			return game.Account{}, game.GameState{}, err
		}
		state.Army = army
	}
	if scope.AllGenerals {
		if err := overlayAuthoritativeGeneralsTx(tx, &state, playerID); err != nil {
			return game.Account{}, game.GameState{}, err
		}
	} else if scope.CurrentGeneral || len(generalIDs) > 0 {
		if err := overlayRewardGeneralsTx(tx, &state, playerID, generalIDs, scope.CurrentGeneral); err != nil {
			return game.Account{}, game.GameState{}, err
		}
	}
	if scope.Buffs {
		if err := overlayAuthoritativeBuffsTx(tx, &state, playerID); err != nil {
			return game.Account{}, game.GameState{}, err
		}
	}

	var previousResourceSnapshot map[string]storageResourceSnapshot
	var previousCurrencySnapshot playerCurrencySnapshot
	if scope.Currency {
		previousCurrencySnapshot = currencySnapshotFromState(state)
	}
	if scope.Resources {
		previousResourceSnapshot = resourceSnapshotsFromStorageState(state.Resources)
	}
	var previousInventorySnapshot map[string]storageInventorySnapshot
	if scope.AllInventory || len(inventoryItemIDs) > 0 {
		previousInventorySnapshot = inventorySnapshotsFromStorageStateWithSlots(state.Inventory, state.InventorySlots)
	}
	var previousArmySnapshot map[string]storageArmySnapshot
	if scope.AllArmy || len(unitTypes) > 0 {
		previousArmySnapshot = armySnapshotsFromStorageState(state.Army)
	}
	var previousGeneralSnapshot map[string]storageGeneralSnapshot
	var previousGeneralAssignmentSnapshot map[string]storageGeneralAssignmentSnapshot
	if scope.AllGenerals || scope.CurrentGeneral || len(generalIDs) > 0 {
		previousGeneralSnapshot = generalSnapshotsFromStorageState(state.Generals)
		previousGeneralAssignmentSnapshot = generalAssignmentSnapshotsFromStorageState(state.GeneralAssignments)
	}
	var previousBuffSnapshot map[string]storageBuffSnapshot
	if scope.Buffs {
		previousBuffSnapshot = buffSnapshotsFromStorageState(state.Buffs)
	}

	if update != nil {
		if err = update(&account, &state); err != nil {
			return game.Account{}, game.GameState{}, err
		}
	}

	nextStateJSON, err := marshalPlayerStateSnapshot(state)
	if err != nil {
		return game.Account{}, game.GameState{}, err
	}
	if scope.Resources && resourceSnapshotChanged(previousResourceSnapshot, state.Resources) {
		if err := syncPlayerResourcesTx(tx, playerID, state.Resources, updatedAt.UTC()); err != nil {
			return game.Account{}, game.GameState{}, err
		}
	}
	if scope.Currency && currencySnapshotChanged(previousCurrencySnapshot, state) {
		if err := syncPlayerCurrencyTx(tx, playerID, &state, updatedAt.UTC()); err != nil {
			return game.Account{}, game.GameState{}, err
		}
	}
	if (scope.AllInventory || len(inventoryItemIDs) > 0) && inventorySnapshotChangedWithSlots(previousInventorySnapshot, state.Inventory, state.InventorySlots) {
		if len(inventoryItemIDs) > 0 && !scope.AllInventory {
			if err := syncPlayerInventoryScopedDeltaTx(tx, playerID, previousInventorySnapshot, state.Inventory, state.InventorySlots, inventoryItemIDs, updatedAt.UTC()); err != nil {
				return game.Account{}, game.GameState{}, err
			}
		} else {
			if err := syncPlayerInventoryDeltaTx(tx, playerID, previousInventorySnapshot, state.Inventory, state.InventorySlots, updatedAt.UTC()); err != nil {
				return game.Account{}, game.GameState{}, err
			}
		}
	}
	if (scope.AllArmy || len(unitTypes) > 0) && armySnapshotChanged(previousArmySnapshot, state.Army) {
		if err := syncPlayerArmyDeltaTx(tx, playerID, previousArmySnapshot, state.Army, updatedAt.UTC()); err != nil {
			return game.Account{}, game.GameState{}, err
		}
	}
	if (scope.AllGenerals || scope.CurrentGeneral || len(generalIDs) > 0) && generalSnapshotChanged(previousGeneralSnapshot, state.Generals) {
		if scope.AllGenerals {
			if err := syncPlayerGeneralsTx(tx, playerID, state.Generals, updatedAt.UTC()); err != nil {
				return game.Account{}, game.GameState{}, err
			}
		} else {
			if err := syncCombatGeneralsDeltaTx(tx, playerID, previousGeneralSnapshot, state.Generals, updatedAt.UTC()); err != nil {
				return game.Account{}, game.GameState{}, err
			}
		}
	}
	if scope.AllGenerals && generalAssignmentSnapshotChanged(previousGeneralAssignmentSnapshot, state.GeneralAssignments) {
		if err := syncPlayerGeneralAssignmentsTx(tx, playerID, state.GeneralAssignments, updatedAt.UTC()); err != nil {
			return game.Account{}, game.GameState{}, err
		}
	}
	if scope.Buffs && buffSnapshotChanged(previousBuffSnapshot, state.Buffs) {
		if err := syncPlayerBuffsTx(tx, playerID, state.Buffs, updatedAt.UTC()); err != nil {
			return game.Account{}, game.GameState{}, err
		}
	}
	if account.Gold != previousAccountGold {
		if _, err = tx.Exec(`UPDATE accounts SET gold = ? WHERE id = ?`, account.Gold, accountID); err != nil {
			return game.Account{}, game.GameState{}, err
		}
	}
	if !bytes.Equal(stateJSON, nextStateJSON) {
		result, err := tx.Exec(
			`UPDATE players
			 SET nickname = ?, faction = ?, mail_code = ?, state_json = ?, updated_at = ?
			 WHERE id = ? AND account_id = ?`,
			state.Player.Nickname,
			state.Player.Faction,
			state.Player.MailCode,
			nextStateJSON,
			updatedAt.UTC(),
			playerID,
			accountID,
		)
		if err != nil {
			return game.Account{}, game.GameState{}, err
		}
		if affected, err := result.RowsAffected(); err != nil {
			return game.Account{}, game.GameState{}, err
		} else if affected == 0 {
			return game.Account{}, game.GameState{}, game.ErrPlayerNotFound
		}
	}
	if err = tx.Commit(); err != nil {
		return game.Account{}, game.GameState{}, err
	}
	if len(inventoryItemIDs) > 0 && !scope.AllInventory {
		if err := r.overlayAuthoritativeInventory(&state, playerID); err != nil {
			return game.Account{}, game.GameState{}, err
		}
	}
	if len(unitTypes) > 0 && !scope.AllArmy {
		if err := r.overlayAuthoritativeArmy(&state, playerID); err != nil {
			return game.Account{}, game.GameState{}, err
		}
	}
	if (scope.CurrentGeneral || len(generalIDs) > 0) && !scope.AllGenerals {
		if err := r.overlayAuthoritativeGenerals(&state, playerID); err != nil {
			return game.Account{}, game.GameState{}, err
		}
	}
	if scope.Buffs {
		if err := r.overlayAuthoritativeBuffs(&state, playerID); err != nil {
			return game.Account{}, game.GameState{}, err
		}
	}
	if !scope.Currency {
		if err := r.overlayAuthoritativeCurrency(&state, playerID); err != nil {
			return game.Account{}, game.GameState{}, err
		}
	}
	return account, state, nil
}
