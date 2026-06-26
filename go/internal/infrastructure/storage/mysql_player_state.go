// 本文件归口 MySQL 玩家存档主状态仓储方法。
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

func (r *MySQLRepository) CreatePlayer(accountID string, state game.GameState, updatedAt time.Time) error {
	stateJSON, err := json.Marshal(state)
	if err != nil {
		return err
	}

	now := updatedAt.UTC()
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if _, err = tx.Exec(
		`INSERT INTO players (id, account_id, nickname, faction, mail_code, state_json, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		state.Player.ID,
		accountID,
		state.Player.Nickname,
		state.Player.Faction,
		state.Player.MailCode,
		stateJSON,
		now,
		now,
	); err != nil {
		return err
	}
	if err := syncPlayerResourcesTx(tx, state.Player.ID, state.Resources, now); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *MySQLRepository) DeleteAccount(accountID string) error {
	rows, err := r.db.Query(`SELECT id FROM players WHERE account_id = ?`, accountID)
	if err != nil {
		return err
	}
	playerIDs := []string{}
	for rows.Next() {
		var playerID string
		if err := rows.Scan(&playerID); err != nil {
			_ = rows.Close()
			return err
		}
		playerIDs = append(playerIDs, playerID)
	}
	if err := rows.Close(); err != nil {
		return err
	}
	for _, playerID := range playerIDs {
		_, _ = r.db.Exec(`DELETE FROM battle_reports WHERE player_id = ?`, playerID)
		_, _ = r.db.Exec(`DELETE FROM mails WHERE player_id = ?`, playerID)
	}

	result, err := r.db.Exec(`DELETE FROM accounts WHERE id = ?`, accountID)
	if err != nil {
		return err
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return game.ErrAccountNotFound
	}
	return nil
}

func (r *MySQLRepository) DeletePlayer(playerID string) error {
	// 先删独立存储
	_, _ = r.db.Exec(`DELETE FROM battle_reports WHERE player_id = ?`, playerID)
	_, _ = r.db.Exec(`DELETE FROM mails WHERE player_id = ?`, playerID)

	result, err := r.db.Exec(`DELETE FROM players WHERE id = ?`, playerID)
	if err != nil {
		return err
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return game.ErrPlayerNotFound
	}
	return nil
}

func (r *MySQLRepository) GetState(playerID string) (game.GameState, error) {
	var stateJSON []byte
	var mailCode string
	err := r.db.QueryRow(`SELECT state_json, mail_code FROM players WHERE id = ? LIMIT 1`, playerID).Scan(&stateJSON, &mailCode)
	if errors.Is(err, sql.ErrNoRows) {
		return game.GameState{}, game.ErrPlayerNotFound
	}
	if err != nil {
		return game.GameState{}, err
	}

	var state game.GameState
	if err := json.Unmarshal(stateJSON, &state); err != nil {
		return game.GameState{}, err
	}
	if state.Player.MailCode == "" {
		state.Player.MailCode = mailCode
	}
	return state, nil
}

func (r *MySQLRepository) UpdatePlayerState(playerID string, updatedAt time.Time, update func(state *game.GameState) error) (game.GameState, error) {
	tx, err := r.db.Begin()
	if err != nil {
		return game.GameState{}, err
	}
	defer func() { _ = tx.Rollback() }()

	var stateJSON []byte
	var mailCode string
	err = tx.QueryRow(
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
	previousResourceSnapshot := resourceSnapshotsFromStorageState(state.Resources)
	if state.Player.MailCode == "" {
		state.Player.MailCode = mailCode
	}
	if update != nil {
		if err = update(&state); err != nil {
			return game.GameState{}, err
		}
	}

	nextStateJSON, err := json.Marshal(state)
	if err != nil {
		return game.GameState{}, err
	}
	if resourceSnapshotChanged(previousResourceSnapshot, state.Resources) {
		if err := syncPlayerResourcesTx(tx, playerID, state.Resources, updatedAt.UTC()); err != nil {
			return game.GameState{}, err
		}
	}
	if bytes.Equal(stateJSON, nextStateJSON) {
		if err = tx.Commit(); err != nil {
			return game.GameState{}, err
		}
		return state, nil
	}
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
	affected, err := result.RowsAffected()
	if err != nil {
		return game.GameState{}, err
	}
	if affected == 0 {
		return game.GameState{}, game.ErrPlayerNotFound
	}
	if err = tx.Commit(); err != nil {
		return game.GameState{}, err
	}
	return state, nil
}

func (r *MySQLRepository) UpdateAccountPlayerState(accountID string, playerID string, updatedAt time.Time, update func(account *game.Account, state *game.GameState) error) (game.Account, game.GameState, error) {
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
	previousResourceSnapshot := resourceSnapshotsFromStorageState(state.Resources)
	if state.Player.MailCode == "" {
		state.Player.MailCode = mailCode
	}

	if update != nil {
		if err = update(&account, &state); err != nil {
			return game.Account{}, game.GameState{}, err
		}
	}

	nextStateJSON, err := json.Marshal(state)
	if err != nil {
		return game.Account{}, game.GameState{}, err
	}
	if resourceSnapshotChanged(previousResourceSnapshot, state.Resources) {
		if err := syncPlayerResourcesTx(tx, playerID, state.Resources, updatedAt.UTC()); err != nil {
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

// syncPlayerResourcesTx 把主状态中的资源快照同步到规范化资源表。
func syncPlayerResourcesTx(tx *sql.Tx, playerID string, resources game.ResourceState, updatedAt time.Time) error {
	resourceTypes := resourceTypesFromState(resources)
	if len(resourceTypes) == 0 {
		_, err := tx.Exec(`DELETE FROM player_resources WHERE player_id = ?`, playerID)
		return err
	}

	for _, resourceType := range resourceTypes {
		if _, err := tx.Exec(
			`INSERT INTO player_resources (player_id, resource_type, amount, capacity, updated_at)
			 VALUES (?, ?, ?, ?, ?)
			 ON DUPLICATE KEY UPDATE
				amount = VALUES(amount),
				capacity = VALUES(capacity),
				updated_at = VALUES(updated_at)`,
			playerID,
			resourceType,
			resources.Items[resourceType],
			resources.Capacity[resourceType],
			updatedAt.UTC(),
		); err != nil {
			return err
		}
	}
	if err := deleteStalePlayerResourcesTx(tx, playerID, resourceTypes); err != nil {
		return err
	}
	return nil
}

// resourceSnapshotChanged 判断资源数量或容量是否发生变化。
func resourceSnapshotChanged(before map[string]storageResourceSnapshot, after game.ResourceState) bool {
	return !resourceSnapshotMapsEqual(
		before,
		resourceSnapshotsFromStorageState(after),
	)
}

// resourceTypesFromState 合并资源数量和容量里的资源类型，保证资源注册扩展后也能同步。
func resourceTypesFromState(resources game.ResourceState) []string {
	resourceTypeSet := map[string]struct{}{}
	for resourceType := range resources.Items {
		if strings.TrimSpace(resourceType) != "" {
			resourceTypeSet[resourceType] = struct{}{}
		}
	}
	for resourceType := range resources.Capacity {
		if strings.TrimSpace(resourceType) != "" {
			resourceTypeSet[resourceType] = struct{}{}
		}
	}
	resourceTypes := make([]string, 0, len(resourceTypeSet))
	for resourceType := range resourceTypeSet {
		resourceTypes = append(resourceTypes, resourceType)
	}
	sort.Strings(resourceTypes)
	return resourceTypes
}

type storageResourceSnapshot struct {
	Amount   int
	Capacity int
}

// resourceSnapshotsFromStorageState 从 ResourceState 生成同步比较快照。
func resourceSnapshotsFromStorageState(resources game.ResourceState) map[string]storageResourceSnapshot {
	snapshots := map[string]storageResourceSnapshot{}
	for _, resourceType := range resourceTypesFromState(resources) {
		snapshots[resourceType] = storageResourceSnapshot{
			Amount:   resources.Items[resourceType],
			Capacity: resources.Capacity[resourceType],
		}
	}
	return snapshots
}

// resourceSnapshotMapsEqual 比较两个资源快照集合是否一致。
func resourceSnapshotMapsEqual(a map[string]storageResourceSnapshot, b map[string]storageResourceSnapshot) bool {
	if len(a) != len(b) {
		return false
	}
	for resourceType, left := range a {
		if right, ok := b[resourceType]; !ok || left != right {
			return false
		}
	}
	return true
}

// deleteStalePlayerResourcesTx 删除主状态里已经不存在的资源类型。
func deleteStalePlayerResourcesTx(tx *sql.Tx, playerID string, resourceTypes []string) error {
	placeholders := strings.TrimRight(strings.Repeat("?,", len(resourceTypes)), ",")
	args := make([]any, 0, len(resourceTypes)+1)
	args = append(args, playerID)
	for _, resourceType := range resourceTypes {
		args = append(args, resourceType)
	}
	_, err := tx.Exec(
		`DELETE FROM player_resources
		 WHERE player_id = ? AND resource_type NOT IN (`+placeholders+`)`,
		args...,
	)
	return err
}
