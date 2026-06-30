// 本文件归口将领经验包使用所需的 MySQL 小事务。
package storage

import (
	"database/sql"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"hero3/internal/app/game"
)

// UpdateGeneralExpItemState 只锁定经验包使用需要的目标背包格子和当前主将行。
func (r *MySQLRepository) UpdateGeneralExpItemState(playerID string, itemID string, updatedAt time.Time, update func(state *game.GameState) error) (game.GameState, error) {
	itemID = strings.TrimSpace(itemID)
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
	snapshotItemAmount := 0
	if state.Inventory != nil {
		snapshotItemAmount = state.Inventory[itemID].Amount
	}
	inventory, slots, inventoryFound, err := loadPlayerInventoryItemRowsTx(tx, playerID, itemID)
	if err != nil {
		return game.GameState{}, err
	}
	if !inventoryFound && snapshotItemAmount > 0 {
		return game.GameState{}, errPlayerInventoryMissing
	}
	state.Inventory = inventory
	state.InventorySlots = slots

	previousInventorySnapshot := inventorySnapshotsFromStorageStateWithSlots(state.Inventory, state.InventorySlots)
	assignment, assignmentFound, err := loadMainGeneralAssignmentRowTx(tx, playerID)
	if err != nil {
		return game.GameState{}, err
	}
	var general game.General
	var generalFound bool
	if assignmentFound {
		general, generalFound, err = loadPlayerGeneralRowTx(tx, playerID, assignment.GeneralID)
		if err != nil {
			return game.GameState{}, err
		}
	}
	if !generalFound {
		general, generalFound, err = loadFirstPlayerGeneralRowTx(tx, playerID)
		if err != nil {
			return game.GameState{}, err
		}
	}
	if !generalFound {
		if state.General != nil || len(state.Generals) > 0 {
			return game.GameState{}, errPlayerGeneralsMissing
		}
		return game.GameState{}, game.ErrGeneralNotFound
	}
	state.General = &general
	state.Generals = []game.General{general}
	if assignmentFound {
		state.GeneralAssignments = []game.GeneralAssignment{assignment}
	} else {
		state.GeneralAssignments = nil
	}
	game.EnsureGeneralRoster(&state, updatedAt.UTC())
	previousGeneralSnapshot := generalSnapshotsFromStorageState(state.Generals)

	if update != nil {
		if err = update(&state); err != nil {
			return game.GameState{}, err
		}
	}

	if inventorySnapshotChangedWithSlots(previousInventorySnapshot, state.Inventory, state.InventorySlots) {
		if err := syncPlayerInventoryDeltaTx(tx, playerID, previousInventorySnapshot, state.Inventory, state.InventorySlots, updatedAt.UTC()); err != nil {
			return game.GameState{}, err
		}
	}
	if generalSnapshotChanged(previousGeneralSnapshot, state.Generals) {
		if state.General == nil {
			return game.GameState{}, game.ErrGeneralNotFound
		}
		if err := syncPlayerGeneralRowTx(tx, playerID, *state.General, updatedAt.UTC()); err != nil {
			return game.GameState{}, err
		}
	}
	if !assignmentFound && len(state.GeneralAssignments) > 0 {
		if err := syncPlayerGeneralAssignmentRowTx(tx, playerID, state.GeneralAssignments[0], updatedAt.UTC()); err != nil {
			return game.GameState{}, err
		}
	}
	if err = tx.Commit(); err != nil {
		return game.GameState{}, err
	}
	if err := r.overlayAuthoritativeInventory(&state, playerID); err != nil {
		return game.GameState{}, err
	}
	if err := r.overlayAuthoritativeGenerals(&state, playerID); err != nil {
		return game.GameState{}, err
	}
	return state, nil
}

// loadPlayerInventoryItemRowsTx 锁定玩家某个物品的实际背包格子。
func loadPlayerInventoryItemRowsTx(tx *sql.Tx, playerID string, itemID string) (map[string]game.ItemStack, []game.ItemStack, bool, error) {
	rows, err := tx.Query(
		`SELECT slot_id, item_id, amount, obtained_at, updated_at
		 FROM player_inventory FORCE INDEX (idx_player_inventory_player_item)
		 WHERE player_id = ? AND item_id = ?
		 ORDER BY slot_id
		 FOR UPDATE`,
		playerID,
		itemID,
	)
	if err != nil {
		return nil, nil, false, err
	}
	defer rows.Close()

	slots := []game.ItemStack{}
	for rows.Next() {
		var stack game.ItemStack
		var obtainedAt sql.NullTime
		var updatedAt sql.NullTime
		if err := rows.Scan(&stack.SlotID, &stack.ItemID, &stack.Amount, &obtainedAt, &updatedAt); err != nil {
			return nil, nil, false, err
		}
		stack.SlotID = strings.TrimSpace(stack.SlotID)
		stack.ItemID = strings.TrimSpace(stack.ItemID)
		if stack.ItemID == "" || stack.Amount <= 0 {
			continue
		}
		if obtainedAt.Valid {
			stack.ObtainedAt = obtainedAt.Time.UTC().Format(time.RFC3339)
		}
		if updatedAt.Valid {
			stack.UpdatedAt = updatedAt.Time.UTC().Format(time.RFC3339)
		}
		slots = append(slots, stack)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, false, err
	}
	return game.AggregateInventorySlotsForStorage(slots), slots, len(slots) > 0, nil
}

// loadMainGeneralAssignmentRowTx 锁定当前主将占用记录。
func loadMainGeneralAssignmentRowTx(tx *sql.Tx, playerID string) (game.GeneralAssignment, bool, error) {
	row := tx.QueryRow(
		`SELECT assignment_id, general_id, assignment_slot, module_id, status, assigned_at, ends_at
		 FROM player_general_assignments
		 WHERE player_id = ? AND (assignment_id = ? OR assignment_slot = ?)
		 ORDER BY CASE WHEN assignment_id = ? THEN 0 ELSE 1 END, assignment_id
		 LIMIT 1
		 FOR UPDATE`,
		playerID,
		game.GeneralAssignmentMain,
		game.GeneralAssignmentMain,
		game.GeneralAssignmentMain,
	)
	assignment, err := scanGeneralAssignmentRow(row)
	if errors.Is(err, sql.ErrNoRows) {
		return game.GeneralAssignment{}, false, nil
	}
	if err != nil {
		return game.GeneralAssignment{}, false, err
	}
	return assignment, true, nil
}

// loadPlayerGeneralRowTx 锁定指定武将行。
func loadPlayerGeneralRowTx(tx *sql.Tx, playerID string, generalID string) (game.General, bool, error) {
	row := tx.QueryRow(
		`SELECT general_id, level, exp, stats_json
		 FROM player_generals
		 WHERE player_id = ? AND general_id = ?
		 LIMIT 1
		 FOR UPDATE`,
		playerID,
		generalID,
	)
	general, err := scanGeneralRow(row)
	if errors.Is(err, sql.ErrNoRows) {
		return game.General{}, false, nil
	}
	if err != nil {
		return game.General{}, false, err
	}
	return general, true, nil
}

// loadFirstPlayerGeneralRowTx 兼容缺少主将占用记录的旧数据，锁定玩家第一个武将。
func loadFirstPlayerGeneralRowTx(tx *sql.Tx, playerID string) (game.General, bool, error) {
	row := tx.QueryRow(
		`SELECT general_id, level, exp, stats_json
		 FROM player_generals
		 WHERE player_id = ?
		 ORDER BY general_id
		 LIMIT 1
		 FOR UPDATE`,
		playerID,
	)
	general, err := scanGeneralRow(row)
	if errors.Is(err, sql.ErrNoRows) {
		return game.General{}, false, nil
	}
	if err != nil {
		return game.General{}, false, err
	}
	return general, true, nil
}

// scanGeneralRow 把单行武将记录还原为领域对象。
func scanGeneralRow(row *sql.Row) (game.General, error) {
	var general game.General
	var statsJSON []byte
	if err := row.Scan(&general.ID, &general.Level, &general.Exp, &statsJSON); err != nil {
		return game.General{}, err
	}
	general.ID = strings.TrimSpace(general.ID)
	if len(statsJSON) > 0 {
		_ = json.Unmarshal(statsJSON, &general.Stats)
	}
	if general.Stats == nil {
		general.Stats = map[string]int{}
	}
	if hero, ok := game.GetHeroConfig(general.ID); ok {
		general.Name = hero.Name
	} else {
		general.Name = general.ID
	}
	applyStorageGeneralConfig(&general)
	return general, nil
}

// scanGeneralAssignmentRow 把单行武将占用记录还原为领域对象。
func scanGeneralAssignmentRow(row *sql.Row) (game.GeneralAssignment, error) {
	var assignment game.GeneralAssignment
	var assignedAt sql.NullTime
	var endsAt sql.NullTime
	if err := row.Scan(&assignment.ID, &assignment.GeneralID, &assignment.Slot, &assignment.ModuleID, &assignment.Status, &assignedAt, &endsAt); err != nil {
		return game.GeneralAssignment{}, err
	}
	assignment.ID = strings.TrimSpace(assignment.ID)
	assignment.GeneralID = strings.TrimSpace(assignment.GeneralID)
	if assignedAt.Valid {
		assignment.AssignedAt = assignedAt.Time.UTC().Format(time.RFC3339)
	}
	if endsAt.Valid {
		assignment.EndsAt = endsAt.Time.UTC().Format(time.RFC3339)
	}
	return assignment, nil
}

// syncPlayerGeneralRowTx 只写回一个武将行，不删除其他武将。
func syncPlayerGeneralRowTx(tx *sql.Tx, playerID string, general game.General, updatedAt time.Time) error {
	generalID := strings.TrimSpace(general.ID)
	if generalID == "" {
		return game.ErrGeneralNotFound
	}
	statsJSON, err := json.Marshal(general.Stats)
	if err != nil {
		return err
	}
	hero, _ := game.GetHeroConfig(generalID)
	_, err = tx.Exec(
		`INSERT INTO player_generals (player_id, general_id, faction, level, exp, stats_json, acquired_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		 ON DUPLICATE KEY UPDATE
			faction = VALUES(faction),
			level = VALUES(level),
			exp = VALUES(exp),
			stats_json = VALUES(stats_json),
			updated_at = VALUES(updated_at)`,
		playerID,
		generalID,
		hero.Faction,
		general.Level,
		general.Exp,
		statsJSON,
		updatedAt.UTC(),
		updatedAt.UTC(),
	)
	return err
}

// syncPlayerGeneralAssignmentRowTx 只写回一个武将占用行，不删除其他占用。
func syncPlayerGeneralAssignmentRowTx(tx *sql.Tx, playerID string, assignment game.GeneralAssignment, updatedAt time.Time) error {
	assignment.ID = strings.TrimSpace(assignment.ID)
	assignment.GeneralID = strings.TrimSpace(assignment.GeneralID)
	if assignment.ID == "" || assignment.GeneralID == "" {
		return nil
	}
	_, err := tx.Exec(
		`INSERT INTO player_general_assignments (player_id, assignment_id, general_id, assignment_slot, module_id, status, assigned_at, ends_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		 ON DUPLICATE KEY UPDATE
			general_id = VALUES(general_id),
			assignment_slot = VALUES(assignment_slot),
			module_id = VALUES(module_id),
			status = VALUES(status),
			assigned_at = VALUES(assigned_at),
			ends_at = VALUES(ends_at),
			updated_at = VALUES(updated_at)`,
		playerID,
		assignment.ID,
		assignment.GeneralID,
		assignment.Slot,
		assignment.ModuleID,
		assignment.Status,
		nullableTimeArg(parseStorageTime(assignment.AssignedAt)),
		nullableTimeArg(parseStorageTime(assignment.EndsAt)),
		updatedAt.UTC(),
	)
	return err
}
