// 本文件归口武将操作所需的 MySQL 资产级事务，避免走完整玩家状态事务。
package storage

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"hero3/internal/app/game"
)

// UpdateGeneralState 复用武将操作所需资产范围，支持加点和切换武将。
func (r *MySQLRepository) UpdateGeneralState(playerID string, updatedAt time.Time, update func(state *game.GameState) error) (game.GameState, error) {
	return r.UpdateItemState(playerID, updatedAt, update)
}

// UpdateAccountGeneralState 在账号 + 武将资产级事务中执行洗点等账号金币操作。
func (r *MySQLRepository) UpdateAccountGeneralState(accountID string, playerID string, updatedAt time.Time, update func(account *game.Account, state *game.GameState) error) (game.Account, game.GameState, error) {
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
		 LIMIT 1
		 FOR UPDATE`,
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
	if err := overlayAuthoritativeResourcesTx(tx, &state, playerID); err != nil {
		return game.Account{}, game.GameState{}, err
	}
	if err := overlayAuthoritativeInventoryTx(tx, &state, playerID); err != nil {
		return game.Account{}, game.GameState{}, err
	}
	if err := overlayAuthoritativeBuildingsTx(tx, &state, playerID); err != nil {
		return game.Account{}, game.GameState{}, err
	}
	if err := overlayAuthoritativeResourceSlotsTx(tx, &state, playerID); err != nil {
		return game.Account{}, game.GameState{}, err
	}
	if err := overlayAuthoritativeArmyTx(tx, &state, playerID); err != nil {
		return game.Account{}, game.GameState{}, err
	}
	if err := overlayAuthoritativeRecruitQueuesTx(tx, &state, playerID); err != nil {
		return game.Account{}, game.GameState{}, err
	}
	if err := overlayAuthoritativeGeneralsTx(tx, &state, playerID); err != nil {
		return game.Account{}, game.GameState{}, err
	}
	if err := overlayAuthoritativeBuffsTx(tx, &state, playerID); err != nil {
		return game.Account{}, game.GameState{}, err
	}

	previousResourceSnapshot := resourceSnapshotsFromStorageState(state.Resources)
	previousInventorySnapshot := inventorySnapshotsFromStorageState(state.Inventory)
	previousBuildingSnapshot := buildingSnapshotsFromStorageState(state.Buildings)
	previousResourceSlotSnapshot := resourceSlotSnapshotsFromStorageState(state.ResourceSlots)
	previousArmySnapshot := armySnapshotsFromStorageState(state.Army)
	previousRecruitQueueSnapshot := recruitQueueSnapshotsFromStorageState(state.RecruitQueues)
	previousGeneralSnapshot := generalSnapshotsFromStorageState(state.Generals)
	previousGeneralAssignmentSnapshot := generalAssignmentSnapshotsFromStorageState(state.GeneralAssignments)
	previousBuffSnapshot := buffSnapshotsFromStorageState(state.Buffs)

	if update != nil {
		if err = update(&account, &state); err != nil {
			return game.Account{}, game.GameState{}, err
		}
	}
	if len(state.ResourceSlots) == 0 {
		state.ResourceSlots = game.BuildResourceSlotsFromBuildings(state.Buildings, updatedAt)
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
	if inventorySnapshotChanged(previousInventorySnapshot, state.Inventory) {
		if err := syncPlayerInventoryTx(tx, playerID, state.Inventory, state.InventorySlots, updatedAt.UTC()); err != nil {
			return game.Account{}, game.GameState{}, err
		}
	}
	if buildingSnapshotChanged(previousBuildingSnapshot, state.Buildings) {
		if err := syncPlayerBuildingsTx(tx, playerID, state.Buildings, updatedAt.UTC()); err != nil {
			return game.Account{}, game.GameState{}, err
		}
	}
	if resourceSlotSnapshotChanged(previousResourceSlotSnapshot, state.ResourceSlots) || buildingSnapshotChanged(previousBuildingSnapshot, state.Buildings) {
		if err := syncPlayerResourceSlotsTx(tx, playerID, state.ResourceSlots, updatedAt.UTC()); err != nil {
			return game.Account{}, game.GameState{}, err
		}
	}
	if armySnapshotChanged(previousArmySnapshot, state.Army) {
		if err := syncPlayerArmyTx(tx, playerID, state.Army, updatedAt.UTC()); err != nil {
			return game.Account{}, game.GameState{}, err
		}
	}
	if recruitQueueSnapshotChanged(previousRecruitQueueSnapshot, state.RecruitQueues) {
		if err := syncPlayerRecruitQueuesTx(tx, playerID, state.RecruitQueues, updatedAt.UTC()); err != nil {
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
