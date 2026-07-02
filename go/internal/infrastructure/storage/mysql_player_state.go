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

var errPlayerResourcesMissing = errors.New("player_resources rows missing; run backfill-resources before using resources as authoritative state")
var errPlayerInventoryMissing = errors.New("player_inventory rows missing; run backfill-inventory before using inventory as authoritative state")

func (r *MySQLRepository) CreatePlayer(accountID string, state game.GameState, updatedAt time.Time) error {
	now := updatedAt.UTC()
	if len(state.ResourceSlots) == 0 {
		state.ResourceSlots = game.BuildResourceSlotsFromBuildings(state.Buildings, now)
	}
	game.EnsureGeneralRoster(&state, now)
	stateJSON, err := marshalPlayerStateSnapshot(state)
	if err != nil {
		return err
	}

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
	if err := syncPlayerCurrencyTx(tx, state.Player.ID, &state, now); err != nil {
		return err
	}
	if err := syncPlayerNpcStateTx(tx, state.Player.ID, state.NpcState, now); err != nil {
		return err
	}
	if err := syncPlayerInventoryTx(tx, state.Player.ID, state.Inventory, state.InventorySlots, now); err != nil {
		return err
	}
	if err := syncPlayerBuildingsTx(tx, state.Player.ID, state.Buildings, now); err != nil {
		return err
	}
	if err := syncPlayerResourceSlotsTx(tx, state.Player.ID, state.ResourceSlots, now); err != nil {
		return err
	}
	if err := syncPlayerArmyTx(tx, state.Player.ID, state.Army, now); err != nil {
		return err
	}
	if err := syncPlayerRecruitQueuesTx(tx, state.Player.ID, state.RecruitQueues, now); err != nil {
		return err
	}
	if err := syncPlayerGeneralsTx(tx, state.Player.ID, state.Generals, now); err != nil {
		return err
	}
	if err := syncPlayerGeneralAssignmentsTx(tx, state.Player.ID, state.GeneralAssignments, now); err != nil {
		return err
	}
	if err := syncPlayerBuffsTx(tx, state.Player.ID, state.Buffs, now); err != nil {
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
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if err := deletePlayerRelatedRowsTx(tx, playerID); err != nil {
		return err
	}

	result, err := tx.Exec(`DELETE FROM players WHERE id = ?`, playerID)
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
	return tx.Commit()
}

// deletePlayerRelatedRowsTx 显式清理存档关联行，避免历史库缺少级联外键时删不掉存档。
func deletePlayerRelatedRowsTx(tx *sql.Tx, playerID string) error {
	deleteStatements := []struct {
		query string
		args  []any
	}{
		{`DELETE FROM battle_report_links WHERE report_id IN (SELECT id FROM battle_reports WHERE player_id = ?)`, []any{playerID}},
		{`DELETE FROM battle_report_states WHERE player_id = ? OR report_id IN (SELECT id FROM battle_reports WHERE player_id = ?)`, []any{playerID, playerID}},
		{`DELETE FROM battle_report_participants WHERE player_id = ? OR report_id IN (SELECT id FROM battle_reports WHERE player_id = ?)`, []any{playerID, playerID}},
		{`DELETE FROM battle_reports WHERE player_id = ?`, []any{playerID}},
		{`DELETE FROM battle_events WHERE attacker_player_id = ? OR defender_player_id = ?`, []any{playerID, playerID}},
		{`DELETE FROM pvp_marches WHERE attacker_player_id = ? OR defender_player_id = ?`, []any{playerID, playerID}},
		{`DELETE FROM pvp_battles WHERE attacker_player_id = ? OR defender_player_id = ?`, []any{playerID, playerID}},
		{`DELETE FROM pvp_player_states WHERE player_id = ?`, []any{playerID}},
		{`DELETE FROM pvp_season_players WHERE player_id = ?`, []any{playerID}},
		{`DELETE FROM player_world_positions WHERE player_id = ?`, []any{playerID}},
		{`DELETE FROM minigame_records WHERE player_id = ?`, []any{playerID}},
		{`DELETE FROM mails WHERE player_id = ?`, []any{playerID}},
		{`DELETE FROM gold_ledger WHERE player_id = ?`, []any{playerID}},
		{`DELETE FROM item_ledger WHERE player_id = ?`, []any{playerID}},
		{`DELETE FROM announcement_reads WHERE player_id = ?`, []any{playerID}},
		{`DELETE FROM gameplay_module_settlements WHERE player_id = ?`, []any{playerID}},
		{`DELETE FROM gameplay_module_participants WHERE player_id = ?`, []any{playerID}},
		{`DELETE FROM reincarnation_battles WHERE player_id = ? OR run_id IN (SELECT id FROM reincarnation_runs WHERE player_id = ?)`, []any{playerID, playerID}},
		{`DELETE FROM reincarnation_waves WHERE run_id IN (SELECT id FROM reincarnation_runs WHERE player_id = ?)`, []any{playerID}},
		{`DELETE FROM reincarnation_runs WHERE player_id = ?`, []any{playerID}},
		{`DELETE FROM player_reinforcements WHERE from_player_id = ? OR to_player_id = ? OR owner_player_id = ? OR host_player_id = ?`, []any{playerID, playerID, playerID, playerID}},
		{`DELETE FROM player_buffs WHERE player_id = ?`, []any{playerID}},
		{`DELETE FROM player_general_assignments WHERE player_id = ?`, []any{playerID}},
		{`DELETE FROM player_generals WHERE player_id = ?`, []any{playerID}},
		{`DELETE FROM player_recruit_queues WHERE player_id = ?`, []any{playerID}},
		{`DELETE FROM player_army_units WHERE player_id = ?`, []any{playerID}},
		{`DELETE FROM player_resource_slots WHERE player_id = ?`, []any{playerID}},
		{`DELETE FROM player_buildings WHERE player_id = ?`, []any{playerID}},
		{`DELETE FROM player_inventory WHERE player_id = ?`, []any{playerID}},
		{`DELETE FROM player_currencies WHERE player_id = ?`, []any{playerID}},
		{`DELETE FROM player_npc_states WHERE player_id = ?`, []any{playerID}},
		{`DELETE FROM player_resources WHERE player_id = ?`, []any{playerID}},
	}
	for _, statement := range deleteStatements {
		if _, err := tx.Exec(statement.query, statement.args...); err != nil {
			return err
		}
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
	if err := r.overlayAuthoritativeResources(&state, playerID); err != nil {
		return game.GameState{}, err
	}
	if err := r.overlayAuthoritativeCurrency(&state, playerID); err != nil {
		return game.GameState{}, err
	}
	if err := r.overlayAuthoritativeNpcState(&state, playerID); err != nil {
		return game.GameState{}, err
	}
	if err := r.overlayAuthoritativeInventory(&state, playerID); err != nil {
		return game.GameState{}, err
	}
	if err := r.overlayAuthoritativeBuildings(&state, playerID); err != nil {
		return game.GameState{}, err
	}
	if err := r.overlayAuthoritativeResourceSlots(&state, playerID); err != nil {
		return game.GameState{}, err
	}
	if err := r.overlayAuthoritativeArmy(&state, playerID); err != nil {
		return game.GameState{}, err
	}
	if err := r.overlayAuthoritativeRecruitQueues(&state, playerID); err != nil {
		return game.GameState{}, err
	}
	if err := r.overlayAuthoritativeGenerals(&state, playerID); err != nil {
		return game.GameState{}, err
	}
	if err := r.overlayAuthoritativeBuffs(&state, playerID); err != nil {
		return game.GameState{}, err
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
	if err := overlayAuthoritativeInventoryTx(tx, &state, playerID); err != nil {
		return game.GameState{}, err
	}
	buildings, buildingsFound, err := loadPlayerBuildingsTx(tx, playerID)
	if err != nil {
		return game.GameState{}, err
	}
	previousBuildingSnapshot := buildingSnapshotsFromStorageState(buildings)
	if !buildingsFound {
		previousBuildingSnapshot = buildingSnapshotsFromStorageState(state.Buildings)
	}
	if err := applyAuthoritativeBuildings(&state, buildings, buildingsFound); err != nil {
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
	previousResourceSlotSnapshot := resourceSlotSnapshotsFromStorageState(state.ResourceSlots)
	previousArmySnapshot := armySnapshotsFromStorageState(state.Army)
	previousRecruitQueueSnapshot := recruitQueueSnapshotsFromStorageState(state.RecruitQueues)
	previousGeneralSnapshot := generalSnapshotsFromStorageState(state.Generals)
	previousGeneralAssignmentSnapshot := generalAssignmentSnapshotsFromStorageState(state.GeneralAssignments)
	previousBuffSnapshot := buffSnapshotsFromStorageState(state.Buffs)
	previousNpcStateJSON, err := json.Marshal(state.NpcState)
	if err != nil {
		return game.GameState{}, err
	}
	if update != nil {
		if err = update(&state); err != nil {
			return game.GameState{}, err
		}
	}
	if len(state.ResourceSlots) == 0 {
		state.ResourceSlots = game.BuildResourceSlotsFromBuildings(state.Buildings, updatedAt)
	}
	game.EnsureGeneralRoster(&state, updatedAt)

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
	nextNpcStateJSON, err := json.Marshal(state.NpcState)
	if err != nil {
		return game.GameState{}, err
	}
	if !bytes.Equal(previousNpcStateJSON, nextNpcStateJSON) {
		if err := syncPlayerNpcStateTx(tx, playerID, state.NpcState, updatedAt.UTC()); err != nil {
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
	if state.Player.MailCode == "" {
		state.Player.MailCode = mailCode
	}
	if err := overlayAuthoritativeResourcesTx(tx, &state, playerID); err != nil {
		return game.Account{}, game.GameState{}, err
	}
	if err := overlayAuthoritativeCurrencyTx(tx, &state, playerID, updatedAt); err != nil {
		return game.Account{}, game.GameState{}, err
	}
	if err := overlayAuthoritativeNpcStateTx(tx, &state, playerID, updatedAt); err != nil {
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
	previousCurrencySnapshot := currencySnapshotFromState(state)
	previousInventorySnapshot := inventorySnapshotsFromStorageStateWithSlots(state.Inventory, state.InventorySlots)
	previousBuildingSnapshot := buildingSnapshotsFromStorageState(state.Buildings)
	previousResourceSlotSnapshot := resourceSlotSnapshotsFromStorageState(state.ResourceSlots)
	previousArmySnapshot := armySnapshotsFromStorageState(state.Army)
	previousRecruitQueueSnapshot := recruitQueueSnapshotsFromStorageState(state.RecruitQueues)
	previousGeneralSnapshot := generalSnapshotsFromStorageState(state.Generals)
	previousGeneralAssignmentSnapshot := generalAssignmentSnapshotsFromStorageState(state.GeneralAssignments)
	previousBuffSnapshot := buffSnapshotsFromStorageState(state.Buffs)
	previousNpcStateJSON, err := json.Marshal(state.NpcState)
	if err != nil {
		return game.Account{}, game.GameState{}, err
	}

	if update != nil {
		if err = update(&account, &state); err != nil {
			return game.Account{}, game.GameState{}, err
		}
	}
	if len(state.ResourceSlots) == 0 {
		state.ResourceSlots = game.BuildResourceSlotsFromBuildings(state.Buildings, updatedAt)
	}
	game.EnsureGeneralRoster(&state, updatedAt)

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
	nextNpcStateJSON, err := json.Marshal(state.NpcState)
	if err != nil {
		return game.Account{}, game.GameState{}, err
	}
	if !bytes.Equal(previousNpcStateJSON, nextNpcStateJSON) {
		if err := syncPlayerNpcStateTx(tx, playerID, state.NpcState, updatedAt.UTC()); err != nil {
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

// overlayAuthoritativeResources 用 player_resources 权威表覆盖兼容快照中的资源。
func (r *MySQLRepository) overlayAuthoritativeResources(state *game.GameState, playerID string) error {
	resources, found, err := loadPlayerResources(r.db, playerID)
	if err != nil {
		return err
	}
	return applyAuthoritativeResources(state, resources, found)
}

// overlayAuthoritativeResourcesTx 在事务内锁定并加载玩家资源权威表。
func overlayAuthoritativeResourcesTx(tx *sql.Tx, state *game.GameState, playerID string) error {
	resources, found, err := loadPlayerResourcesTx(tx, playerID)
	if err != nil {
		return err
	}
	return applyAuthoritativeResources(state, resources, found)
}

// overlayAuthoritativeInventory 用 player_inventory 权威表覆盖兼容快照中的背包。
func (r *MySQLRepository) overlayAuthoritativeInventory(state *game.GameState, playerID string) error {
	inventory, slots, found, err := loadPlayerInventory(r.db, playerID)
	if err != nil {
		return err
	}
	return applyAuthoritativeInventory(state, inventory, slots, found)
}

// overlayAuthoritativeInventoryTx 在事务内锁定并加载玩家背包权威表。
func overlayAuthoritativeInventoryTx(tx *sql.Tx, state *game.GameState, playerID string) error {
	inventory, slots, found, err := loadPlayerInventoryTx(tx, playerID)
	if err != nil {
		return err
	}
	return applyAuthoritativeInventory(state, inventory, slots, found)
}

// applyAuthoritativeInventory 将背包权威表结果写回 GameState；旧快照有道具但表为空时显式报错。
func applyAuthoritativeInventory(state *game.GameState, inventory map[string]game.ItemStack, slots []game.ItemStack, found bool) error {
	if state == nil {
		return game.ErrPlayerNotFound
	}
	if !found {
		if len(state.Inventory) == 0 {
			state.Inventory = map[string]game.ItemStack{}
			state.InventorySlots = []game.ItemStack{}
			return nil
		}
		return errPlayerInventoryMissing
	}
	state.Inventory = inventory
	state.InventorySlots = slots
	return nil
}

// applyAuthoritativeResources 将资源权威表结果写回 GameState，缺行时显式报错避免 JSON 回写覆盖权威表。
func applyAuthoritativeResources(state *game.GameState, resources game.ResourceState, found bool) error {
	if state == nil {
		return game.ErrPlayerNotFound
	}
	if !found {
		if len(resourceTypesFromState(state.Resources)) == 0 {
			state.Resources = game.ResourceState{Items: map[string]int{}, Capacity: map[string]int{}}
			return nil
		}
		return errPlayerResourcesMissing
	}
	state.Resources = resources
	return nil
}

type resourceQueryer interface {
	Query(query string, args ...any) (*sql.Rows, error)
}

// loadPlayerResources 从 player_resources 读取玩家资源权威状态。
func loadPlayerResources(queryer resourceQueryer, playerID string) (game.ResourceState, bool, error) {
	return loadPlayerResourcesWithQuery(queryer, playerID, "")
}

// loadPlayerResourcesTx 在事务内读取并锁定玩家资源权威状态。
func loadPlayerResourcesTx(tx *sql.Tx, playerID string) (game.ResourceState, bool, error) {
	return loadPlayerResourcesWithQuery(tx, playerID, " FOR UPDATE")
}

// loadPlayerResourcesWithQuery 读取资源表并还原 ResourceState。
func loadPlayerResourcesWithQuery(queryer resourceQueryer, playerID string, lockClause string) (game.ResourceState, bool, error) {
	rows, err := queryer.Query(
		`SELECT resource_type, amount, capacity
		 FROM player_resources
		 WHERE player_id = ?
		 ORDER BY resource_type`+lockClause,
		playerID,
	)
	if err != nil {
		return game.ResourceState{}, false, err
	}
	defer rows.Close()

	resources := game.ResourceState{
		Items:    map[string]int{},
		Capacity: map[string]int{},
	}
	found := false
	for rows.Next() {
		var resourceType string
		var amount int
		var capacity int
		if err := rows.Scan(&resourceType, &amount, &capacity); err != nil {
			return game.ResourceState{}, false, err
		}
		resourceType = strings.TrimSpace(resourceType)
		if resourceType == "" {
			continue
		}
		resources.Items[resourceType] = amount
		resources.Capacity[resourceType] = capacity
		found = true
	}
	if err := rows.Err(); err != nil {
		return game.ResourceState{}, false, err
	}
	return resources, found, nil
}

// loadPlayerInventory 从 player_inventory 读取玩家背包权威状态。
func loadPlayerInventory(queryer resourceQueryer, playerID string) (map[string]game.ItemStack, []game.ItemStack, bool, error) {
	return loadPlayerInventoryWithQuery(queryer, playerID, "")
}

// loadPlayerInventoryTx 在事务内读取并锁定玩家背包权威状态。
func loadPlayerInventoryTx(tx *sql.Tx, playerID string) (map[string]game.ItemStack, []game.ItemStack, bool, error) {
	return loadPlayerInventoryWithQuery(tx, playerID, " FOR UPDATE")
}

// loadPlayerInventoryWithQuery 读取背包表并还原 Inventory。
func loadPlayerInventoryWithQuery(queryer resourceQueryer, playerID string, lockClause string) (map[string]game.ItemStack, []game.ItemStack, bool, error) {
	rows, err := queryer.Query(
		`SELECT slot_id, item_id, amount, obtained_at, updated_at
		 FROM player_inventory
		 WHERE player_id = ?
		 ORDER BY slot_id`+lockClause,
		playerID,
	)
	if err != nil {
		return nil, nil, false, err
	}
	defer rows.Close()

	slots := []game.ItemStack{}
	found := false
	for rows.Next() {
		var slotID string
		var itemID string
		var amount int
		var obtainedAt sql.NullTime
		var updatedAt sql.NullTime
		if err := rows.Scan(&slotID, &itemID, &amount, &obtainedAt, &updatedAt); err != nil {
			return nil, nil, false, err
		}
		slotID = strings.TrimSpace(slotID)
		itemID = strings.TrimSpace(itemID)
		if itemID == "" || amount <= 0 {
			continue
		}
		stack := game.ItemStack{SlotID: slotID, ItemID: itemID, Amount: amount}
		if obtainedAt.Valid {
			stack.ObtainedAt = obtainedAt.Time.UTC().Format(time.RFC3339)
		}
		if updatedAt.Valid {
			stack.UpdatedAt = updatedAt.Time.UTC().Format(time.RFC3339)
		}
		slots = append(slots, stack)
		found = true
	}
	if err := rows.Err(); err != nil {
		return nil, nil, false, err
	}
	inventory := game.AggregateInventorySlotsForStorage(slots)
	return inventory, slots, found, nil
}

// syncPlayerResourcesTx 把事务内资源快照同步到 player_resources 权威表，并让 state_json.resources 只作为兼容快照。
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

// syncPlayerInventoryTx 把事务内背包快照同步到 player_inventory 权威表，并让 state_json.inventory 只作为兼容快照。
func syncPlayerInventoryTx(tx *sql.Tx, playerID string, inventory map[string]game.ItemStack, slots []game.ItemStack, updatedAt time.Time) error {
	slots = inventorySlotsFromStorageState(inventory, slots, updatedAt)
	if len(slots) == 0 {
		return deleteStalePlayerInventoryTx(tx, playerID, nil)
	}

	slotIDs := make([]string, 0, len(slots))
	for _, stack := range slots {
		slotIDs = append(slotIDs, stack.SlotID)
		obtainedAt := parseInventoryTime(stack.ObtainedAt)
		stackUpdatedAt := parseInventoryTime(stack.UpdatedAt)
		if !stackUpdatedAt.Valid {
			stackUpdatedAt = sql.NullTime{Time: updatedAt.UTC(), Valid: true}
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
	return deleteStalePlayerInventoryTx(tx, playerID, slotIDs)
}

// syncPlayerInventoryDeltaTx 只写入变化后的格子，并逐个删除消失格子，避免高频事务执行玩家级全量 DELETE。
func syncPlayerInventoryDeltaTx(tx *sql.Tx, playerID string, before map[string]storageInventorySnapshot, inventory map[string]game.ItemStack, slots []game.ItemStack, updatedAt time.Time) error {
	if before == nil {
		return syncPlayerInventoryTx(tx, playerID, inventory, slots, updatedAt)
	}
	slots = inventorySlotsFromStorageState(inventory, slots, updatedAt)
	nextSlotSet := make(map[string]struct{}, len(slots))
	for _, stack := range slots {
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
	for slotID := range before {
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

// resourceSnapshotChanged 判断资源数量或容量是否发生变化。
func resourceSnapshotChanged(before map[string]storageResourceSnapshot, after game.ResourceState) bool {
	return !resourceSnapshotMapsEqual(
		before,
		resourceSnapshotsFromStorageState(after),
	)
}

// inventorySnapshotChanged 判断背包数量或时间戳是否发生变化。
func inventorySnapshotChanged(before map[string]storageInventorySnapshot, after map[string]game.ItemStack) bool {
	return !inventorySnapshotMapsEqual(before, inventorySnapshotsFromStorageState(after))
}

// inventorySnapshotChangedWithSlots 判断带格子明细的背包快照是否发生变化。
func inventorySnapshotChangedWithSlots(before map[string]storageInventorySnapshot, after map[string]game.ItemStack, slots []game.ItemStack) bool {
	return !inventorySnapshotMapsEqual(before, inventorySnapshotsFromStorageStateWithSlots(after, slots))
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

// itemIDsFromInventory 提取背包中仍有效的道具 ID。
func itemIDsFromInventory(inventory map[string]game.ItemStack) []string {
	itemIDSet := map[string]struct{}{}
	for itemID, stack := range inventory {
		itemID = strings.TrimSpace(firstNonEmptyStorage(itemID, stack.ItemID))
		if itemID == "" || stack.Amount <= 0 {
			continue
		}
		itemIDSet[itemID] = struct{}{}
	}
	itemIDs := make([]string, 0, len(itemIDSet))
	for itemID := range itemIDSet {
		itemIDs = append(itemIDs, itemID)
	}
	sort.Strings(itemIDs)
	return itemIDs
}

type storageResourceSnapshot struct {
	Amount   int
	Capacity int
}

type storageInventorySnapshot struct {
	SlotID     string
	ItemID     string
	Amount     int
	ObtainedAt string
	UpdatedAt  string
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

// inventorySnapshotsFromStorageState 从 Inventory 生成同步比较快照。
func inventorySnapshotsFromStorageState(inventory map[string]game.ItemStack) map[string]storageInventorySnapshot {
	return inventorySnapshotsFromStorageStateWithSlots(inventory, nil)
}

// inventorySnapshotsFromStorageStateWithSlots 从 Inventory + InventorySlots 生成同步比较快照。
func inventorySnapshotsFromStorageStateWithSlots(inventory map[string]game.ItemStack, slots []game.ItemStack) map[string]storageInventorySnapshot {
	snapshots := map[string]storageInventorySnapshot{}
	for _, stack := range inventorySlotsFromStorageState(inventory, slots, time.Time{}) {
		snapshots[stack.SlotID] = storageInventorySnapshot{
			SlotID:     stack.SlotID,
			ItemID:     stack.ItemID,
			Amount:     stack.Amount,
			ObtainedAt: stack.ObtainedAt,
			UpdatedAt:  stack.UpdatedAt,
		}
	}
	return snapshots
}

// inventorySlotsFromStorageState 生成用于存储和比较的背包格子列表。
func inventorySlotsFromStorageState(inventory map[string]game.ItemStack, slots []game.ItemStack, now time.Time) []game.ItemStack {
	if len(slots) > 0 {
		normalized := make([]game.ItemStack, 0, len(slots))
		for _, stack := range slots {
			stack.ItemID = strings.TrimSpace(stack.ItemID)
			stack.SlotID = strings.TrimSpace(stack.SlotID)
			if stack.ItemID == "" || stack.Amount <= 0 {
				continue
			}
			if stack.SlotID == "" {
				stack.SlotID = "slot_" + stack.ItemID
			}
			normalized = append(normalized, stack)
		}
		sort.SliceStable(normalized, func(i, j int) bool {
			return normalized[i].SlotID < normalized[j].SlotID
		})
		if inventorySlotsMatchInventory(inventory, normalized) {
			return normalized
		}
	}
	result := make([]game.ItemStack, 0, len(inventory))
	for _, itemID := range itemIDsFromInventory(inventory) {
		stack := inventory[itemID]
		stack.ItemID = strings.TrimSpace(firstNonEmptyStorage(stack.ItemID, itemID))
		if stack.ItemID == "" || stack.Amount <= 0 {
			continue
		}
		if stack.SlotID == "" {
			stack.SlotID = "slot_" + stack.ItemID
		}
		result = append(result, stack)
	}
	sort.SliceStable(result, func(i, j int) bool {
		return result[i].SlotID < result[j].SlotID
	})
	return result
}

// inventorySlotsMatchInventory 判断格子明细是否和聚合背包一致；不一致时以聚合背包重建存储格子。
func inventorySlotsMatchInventory(inventory map[string]game.ItemStack, slots []game.ItemStack) bool {
	if len(inventory) == 0 {
		return len(slots) == 0
	}
	aggregated := map[string]int{}
	for _, stack := range slots {
		if stack.ItemID == "" || stack.Amount <= 0 {
			continue
		}
		aggregated[stack.ItemID] += stack.Amount
	}
	for _, itemID := range itemIDsFromInventory(inventory) {
		stack := inventory[itemID]
		itemID = strings.TrimSpace(firstNonEmptyStorage(itemID, stack.ItemID))
		if itemID == "" {
			continue
		}
		if aggregated[itemID] != stack.Amount {
			return false
		}
		delete(aggregated, itemID)
	}
	for _, amount := range aggregated {
		if amount > 0 {
			return false
		}
	}
	return true
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

// inventorySnapshotMapsEqual 比较两个背包快照集合是否一致。
func inventorySnapshotMapsEqual(a map[string]storageInventorySnapshot, b map[string]storageInventorySnapshot) bool {
	if len(a) != len(b) {
		return false
	}
	for itemID, left := range a {
		if right, ok := b[itemID]; !ok || left != right {
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

// deleteStalePlayerInventoryTx 删除兼容快照里已经不存在的背包格子。
func deleteStalePlayerInventoryTx(tx *sql.Tx, playerID string, slotIDs []string) error {
	keep := make(map[string]struct{}, len(slotIDs))
	for _, slotID := range slotIDs {
		slotID = strings.TrimSpace(slotID)
		if slotID != "" {
			keep[slotID] = struct{}{}
		}
	}
	rows, err := tx.Query(`SELECT slot_id FROM player_inventory WHERE player_id = ?`, playerID)
	if err != nil {
		return err
	}
	defer func() { _ = rows.Close() }()
	staleSlotIDs := []string{}
	for rows.Next() {
		var slotID string
		if err := rows.Scan(&slotID); err != nil {
			return err
		}
		if _, ok := keep[slotID]; ok {
			continue
		}
		staleSlotIDs = append(staleSlotIDs, slotID)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if err := rows.Close(); err != nil {
		return err
	}
	for _, slotID := range staleSlotIDs {
		if _, err := tx.Exec(`DELETE FROM player_inventory WHERE player_id = ? AND slot_id = ?`, playerID, slotID); err != nil {
			return err
		}
	}
	return nil
}

// parseInventoryTime 解析背包时间字符串为可空数据库时间。
func parseInventoryTime(value string) sql.NullTime {
	value = strings.TrimSpace(value)
	if value == "" {
		return sql.NullTime{}
	}
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return sql.NullTime{}
	}
	return sql.NullTime{Time: parsed.UTC(), Valid: true}
}

// nullableTimeArg 把 sql.NullTime 转成 Exec 可接受的 NULL 或 time.Time。
func nullableTimeArg(value sql.NullTime) any {
	if !value.Valid {
		return nil
	}
	return value.Time.UTC()
}

// firstNonEmptyStorage 返回第一个非空字符串。
func firstNonEmptyStorage(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
