// 本文件归口道具使用所需的 MySQL 资产级事务，避免走完整玩家状态事务。
package storage

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"hero3/internal/app/game"
)

// UpdateItemState 只加载道具使用、资源结算和道具效果可能影响的资产。
func (r *MySQLRepository) UpdateItemState(playerID string, updatedAt time.Time, update func(state *game.GameState) error) (game.GameState, error) {
	tx, err := r.db.Begin()
	if err != nil {
		return game.GameState{}, err
	}
	defer func() { _ = tx.Rollback() }()

	state, err := updateItemStateTx(tx, playerID, updatedAt, update)
	if err != nil {
		return game.GameState{}, err
	}
	if err = tx.Commit(); err != nil {
		return game.GameState{}, err
	}
	return state, nil
}

// ApplyGeneralExpEvent 在同一事务内写入援军武将经验和幂等标记。
func (r *MySQLRepository) ApplyGeneralExpEvent(playerID string, eventKey string, updatedAt time.Time, update func(state *game.GameState) error) (bool, error) {
	playerID = strings.TrimSpace(playerID)
	eventKey = strings.TrimSpace(eventKey)
	if playerID == "" || eventKey == "" {
		return false, nil
	}
	tx, err := r.db.Begin()
	if err != nil {
		return false, err
	}
	defer func() { _ = tx.Rollback() }()
	result, err := tx.Exec(
		`INSERT IGNORE INTO event_processing_records (module_id, handler_key, event_key, processed_at)
		 VALUES (?, ?, ?, ?)`,
		"general",
		"reinforcement_exp",
		eventKey,
		updatedAt.UTC(),
	)
	if err != nil {
		return false, err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return false, err
	}
	if affected == 0 {
		if err := tx.Commit(); err != nil {
			return false, err
		}
		return false, nil
	}
	if _, err := updateItemStateTx(tx, playerID, updatedAt, update); err != nil {
		return false, err
	}
	if err := tx.Commit(); err != nil {
		return false, err
	}
	return true, nil
}

// updateItemStateTx 在外部事务内更新玩家权威资产快照。
func updateItemStateTx(tx *sql.Tx, playerID string, updatedAt time.Time, update func(state *game.GameState) error) (game.GameState, error) {
	var stateJSON []byte
	var mailCode string
	err := tx.QueryRow(
		`SELECT state_json, mail_code FROM players WHERE id = ? LIMIT 1 FOR UPDATE`,
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
	if err := overlayAuthoritativeCurrencyTx(tx, &state, playerID, updatedAt); err != nil {
		return game.GameState{}, err
	}
	if err := overlayAuthoritativeResourcesTx(tx, &state, playerID); err != nil {
		return game.GameState{}, err
	}
	if err := overlayAuthoritativeInventoryTx(tx, &state, playerID); err != nil {
		return game.GameState{}, err
	}
	if err := overlayAuthoritativeBuildingsTx(tx, &state, playerID); err != nil {
		return game.GameState{}, err
	}
	if err := overlayAuthoritativeResourceSlotsTx(tx, &state, playerID); err != nil {
		return game.GameState{}, err
	}
	if err := overlayAuthoritativeArmyTx(tx, &state, playerID); err != nil {
		return game.GameState{}, err
	}
	if err := overlayAuthoritativeRecruitQueuesTx(tx, &state, playerID); err != nil {
		return game.GameState{}, err
	}
	if err := overlayAuthoritativeGeneralsTx(tx, &state, playerID); err != nil {
		return game.GameState{}, err
	}
	if err := overlayAuthoritativeBuffsTx(tx, &state, playerID); err != nil {
		return game.GameState{}, err
	}

	previousResourceSnapshot := resourceSnapshotsFromStorageState(state.Resources)
	previousCurrencySnapshot := currencySnapshotFromState(state)
	previousInventorySnapshot := inventorySnapshotsFromStorageStateWithSlots(state.Inventory, state.InventorySlots)
	previousBuildingSnapshot := buildingSnapshotsFromStorageState(state.Buildings)
	previousResourceSlotSnapshot := resourceSlotSnapshotsFromStorageState(state.ResourceSlots)
	previousArmySnapshot := armySnapshotsFromStorageState(state.Army)
	previousRecruitQueueSnapshot := recruitQueueSnapshotsFromStorageState(state.RecruitQueues)
	previousGeneralSnapshot := generalSnapshotsFromStorageState(state.Generals)
	previousGeneralAssignmentSnapshot := generalAssignmentSnapshotsFromStorageState(state.GeneralAssignments)
	previousBuffSnapshot := buffSnapshotsFromStorageState(state.Buffs)

	if update != nil {
		if err = update(&state); err != nil {
			return game.GameState{}, err
		}
	}
	if len(state.ResourceSlots) == 0 {
		state.ResourceSlots = game.BuildResourceSlotsFromBuildings(state.Buildings, updatedAt)
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
	if inventorySnapshotChangedWithSlots(previousInventorySnapshot, state.Inventory, state.InventorySlots) {
		if err := syncPlayerInventoryDeltaTx(tx, playerID, previousInventorySnapshot, state.Inventory, state.InventorySlots, updatedAt.UTC()); err != nil {
			return game.GameState{}, err
		}
	}
	if buildingSnapshotChanged(previousBuildingSnapshot, state.Buildings) {
		if err := syncPlayerBuildingsTx(tx, playerID, state.Buildings, updatedAt.UTC()); err != nil {
			return game.GameState{}, err
		}
	}
	if resourceSlotSnapshotChanged(previousResourceSlotSnapshot, state.ResourceSlots) || buildingSnapshotChanged(previousBuildingSnapshot, state.Buildings) {
		if err := syncPlayerResourceSlotsTx(tx, playerID, state.ResourceSlots, updatedAt.UTC()); err != nil {
			return game.GameState{}, err
		}
	}
	if armySnapshotChanged(previousArmySnapshot, state.Army) {
		if err := syncPlayerArmyTx(tx, playerID, state.Army, updatedAt.UTC()); err != nil {
			return game.GameState{}, err
		}
	}
	if recruitQueueSnapshotChanged(previousRecruitQueueSnapshot, state.RecruitQueues) {
		if err := syncPlayerRecruitQueuesTx(tx, playerID, state.RecruitQueues, updatedAt.UTC()); err != nil {
			return game.GameState{}, err
		}
	}
	if generalSnapshotChanged(previousGeneralSnapshot, state.Generals) {
		if err := syncPlayerGeneralsTx(tx, playerID, state.Generals, updatedAt.UTC()); err != nil {
			return game.GameState{}, err
		}
	}
	if generalAssignmentSnapshotChanged(previousGeneralAssignmentSnapshot, state.GeneralAssignments) {
		if err := syncPlayerGeneralAssignmentsTx(tx, playerID, state.GeneralAssignments, updatedAt.UTC()); err != nil {
			return game.GameState{}, err
		}
	}
	if buffSnapshotChanged(previousBuffSnapshot, state.Buffs) {
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
	return state, nil
}
