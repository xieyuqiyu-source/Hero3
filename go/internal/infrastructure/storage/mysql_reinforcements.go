// 本文件归口增援系统的 MySQL 持久化和双方资产事务。
package storage

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"hero3/internal/app/game"
)

// CreateReinforcementWithState 创建增援批次并在同一事务中扣出派出方资产。
func (r *MySQLRepository) CreateReinforcementWithState(fromPlayerID string, toPlayerID string, updatedAt time.Time, update func(from *game.GameState, to *game.GameState, targetRecords []game.Reinforcement) (game.Reinforcement, error)) (game.GameState, game.GameState, game.Reinforcement, error) {
	tx, err := r.db.Begin()
	if err != nil {
		return game.GameState{}, game.GameState{}, game.Reinforcement{}, err
	}
	defer func() { _ = tx.Rollback() }()

	from, fromJSON, to, toJSON, err := loadReinforcementPlayerPairOrderedTx(tx, fromPlayerID, toPlayerID)
	if err != nil {
		return game.GameState{}, game.GameState{}, game.Reinforcement{}, err
	}
	targetRecords, err := listReceivedReinforcementsTx(tx, toPlayerID, " FOR UPDATE")
	if err != nil {
		return game.GameState{}, game.GameState{}, game.Reinforcement{}, err
	}

	previousFromArmy := armySnapshotsFromStorageState(from.Army)
	previousFromAssignments := generalAssignmentSnapshotsFromStorageState(from.GeneralAssignments)
	record, err := update(&from, &to, targetRecords)
	if err != nil {
		return game.GameState{}, game.GameState{}, game.Reinforcement{}, err
	}
	if err := insertReinforcementTx(tx, record); err != nil {
		return game.GameState{}, game.GameState{}, game.Reinforcement{}, err
	}
	if err := saveReinforcementPlayerStateTx(tx, fromPlayerID, from, fromJSON, updatedAt, previousFromArmy, previousFromAssignments); err != nil {
		return game.GameState{}, game.GameState{}, game.Reinforcement{}, err
	}
	if err := saveReinforcementPlayerStateTx(tx, toPlayerID, to, toJSON, updatedAt, armySnapshotsFromStorageState(to.Army), generalAssignmentSnapshotsFromStorageState(to.GeneralAssignments)); err != nil {
		return game.GameState{}, game.GameState{}, game.Reinforcement{}, err
	}
	if err := tx.Commit(); err != nil {
		return game.GameState{}, game.GameState{}, game.Reinforcement{}, err
	}
	return from, to, record, nil
}

// UpdateReinforcement 更新单个增援批次并同步双方相关资产。
func (r *MySQLRepository) UpdateReinforcement(reinforcementID string, updatedAt time.Time, update func(from *game.GameState, to *game.GameState, reinforcement *game.Reinforcement) error) (game.GameState, game.GameState, game.Reinforcement, error) {
	tx, err := r.db.Begin()
	if err != nil {
		return game.GameState{}, game.GameState{}, game.Reinforcement{}, err
	}
	defer func() { _ = tx.Rollback() }()

	record, err := getReinforcementTx(tx, reinforcementID, " FOR UPDATE")
	if err != nil {
		return game.GameState{}, game.GameState{}, game.Reinforcement{}, err
	}
	from, fromJSON, to, toJSON, err := loadReinforcementPlayerPairOrderedTx(tx, record.FromPlayerID, record.ToPlayerID)
	if err != nil {
		return game.GameState{}, game.GameState{}, game.Reinforcement{}, err
	}
	previousFromArmy := armySnapshotsFromStorageState(from.Army)
	previousFromAssignments := generalAssignmentSnapshotsFromStorageState(from.GeneralAssignments)
	previousToArmy := armySnapshotsFromStorageState(to.Army)
	previousToAssignments := generalAssignmentSnapshotsFromStorageState(to.GeneralAssignments)
	previousReinforcementID := record.ID
	if update != nil {
		if err := update(&from, &to, &record); err != nil {
			return game.GameState{}, game.GameState{}, game.Reinforcement{}, err
		}
	}
	if err := updateReinforcementTx(tx, previousReinforcementID, record); err != nil {
		return game.GameState{}, game.GameState{}, game.Reinforcement{}, err
	}
	if err := saveReinforcementPlayerStateTx(tx, record.FromPlayerID, from, fromJSON, updatedAt, previousFromArmy, previousFromAssignments); err != nil {
		return game.GameState{}, game.GameState{}, game.Reinforcement{}, err
	}
	if err := saveReinforcementPlayerStateTx(tx, record.ToPlayerID, to, toJSON, updatedAt, previousToArmy, previousToAssignments); err != nil {
		return game.GameState{}, game.GameState{}, game.Reinforcement{}, err
	}
	if err := tx.Commit(); err != nil {
		return game.GameState{}, game.GameState{}, game.Reinforcement{}, err
	}
	return from, to, record, nil
}

// GetReinforcement 读取单个增援批次。
func (r *MySQLRepository) GetReinforcement(reinforcementID string) (game.Reinforcement, error) {
	return getReinforcementTx(r.db, reinforcementID, "")
}

// ListSentReinforcements 读取玩家派出的增援。
func (r *MySQLRepository) ListSentReinforcements(playerID string) ([]game.Reinforcement, error) {
	rows, err := r.db.Query(
		`SELECT `+reinforcementColumns+`
		 FROM player_reinforcements
		 WHERE from_player_id = ?
		 ORDER BY created_at DESC`,
		playerID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanReinforcementRows(rows)
}

// ListReceivedReinforcements 读取玩家收到的增援。
func (r *MySQLRepository) ListReceivedReinforcements(playerID string) ([]game.Reinforcement, error) {
	return listReceivedReinforcementsTx(r.db, playerID, "")
}

func loadReinforcementPlayerStateTx(tx *sql.Tx, playerID string) (game.GameState, []byte, error) {
	var stateJSON []byte
	var mailCode string
	err := tx.QueryRow(`SELECT state_json, mail_code FROM players WHERE id = ? LIMIT 1 FOR UPDATE`, playerID).Scan(&stateJSON, &mailCode)
	if errors.Is(err, sql.ErrNoRows) {
		return game.GameState{}, nil, game.ErrPlayerNotFound
	}
	if err != nil {
		return game.GameState{}, nil, err
	}
	var state game.GameState
	if err := json.Unmarshal(stateJSON, &state); err != nil {
		return game.GameState{}, nil, err
	}
	if state.Player.MailCode == "" {
		state.Player.MailCode = mailCode
	}
	if err := overlayAuthoritativeArmyTx(tx, &state, playerID); err != nil {
		return game.GameState{}, nil, err
	}
	if err := overlayAuthoritativeGeneralsTx(tx, &state, playerID); err != nil {
		return game.GameState{}, nil, err
	}
	return state, stateJSON, nil
}

func saveReinforcementPlayerStateTx(tx *sql.Tx, playerID string, state game.GameState, previousJSON []byte, updatedAt time.Time, previousArmy map[string]storageArmySnapshot, previousAssignments map[string]storageGeneralAssignmentSnapshot) error {
	if armySnapshotChanged(previousArmy, state.Army) {
		if err := syncPlayerArmyTx(tx, playerID, state.Army, updatedAt.UTC()); err != nil {
			return err
		}
	}
	if generalAssignmentSnapshotChanged(previousAssignments, state.GeneralAssignments) {
		if err := syncPlayerGeneralAssignmentsTx(tx, playerID, state.GeneralAssignments, updatedAt.UTC()); err != nil {
			return err
		}
	}
	nextJSON, err := marshalPlayerStateSnapshot(state)
	if err != nil {
		return err
	}
	if bytes.Equal(previousJSON, nextJSON) {
		return nil
	}
	_, err = tx.Exec(
		`UPDATE players
		 SET nickname = ?, faction = ?, mail_code = ?, state_json = ?, updated_at = ?
		 WHERE id = ?`,
		state.Player.Nickname,
		state.Player.Faction,
		state.Player.MailCode,
		nextJSON,
		updatedAt.UTC(),
		playerID,
	)
	return err
}

const reinforcementColumns = `reinforcement_id, from_player_id, to_player_id, owner_player_id, host_player_id,
	source_type, source_id, source_faction, target_type, target_id, status,
	troops_json, remaining_troops_json, generals_json, losses_json, buff_snapshot_json, rules_json,
	speed_multiplier, march_seconds, return_seconds, sent_at, arrived_at, recalled_at, expelled_at,
	return_started_at, returned_at, last_battle_report_id, last_battle_at, is_annihilated,
	reward_state_json, mail_state_json, metadata_json, created_at, updated_at`

type reinforcementQueryer interface {
	Query(query string, args ...any) (*sql.Rows, error)
	QueryRow(query string, args ...any) *sql.Row
}

func getReinforcementTx(queryer reinforcementQueryer, reinforcementID string, lockClause string) (game.Reinforcement, error) {
	return scanReinforcement(queryer.QueryRow(
		`SELECT `+reinforcementColumns+`
		 FROM player_reinforcements
		 WHERE reinforcement_id = ?
		 LIMIT 1`+lockClause,
		reinforcementID,
	))
}

func listReceivedReinforcementsTx(queryer resourceQueryer, playerID string, lockClause string) ([]game.Reinforcement, error) {
	rows, err := queryer.Query(
		`SELECT `+reinforcementColumns+`
		 FROM player_reinforcements
		 WHERE to_player_id = ?
		 ORDER BY created_at DESC`+lockClause,
		playerID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanReinforcementRows(rows)
}

func insertReinforcementTx(tx *sql.Tx, record game.Reinforcement) error {
	return execSaveReinforcementTx(tx, record, false)
}

func updateReinforcementTx(tx *sql.Tx, previousReinforcementID string, record game.Reinforcement) error {
	return execSaveReinforcementByIDTx(tx, previousReinforcementID, record, true)
}

func execSaveReinforcementTx(tx *sql.Tx, record game.Reinforcement, updateOnly bool) error {
	return execSaveReinforcementByIDTx(tx, record.ID, record, updateOnly)
}

// execSaveReinforcementByIDTx 保存增援记录，并允许把历史驻防批次改写成固定 ID。
func execSaveReinforcementByIDTx(tx *sql.Tx, previousReinforcementID string, record game.Reinforcement, updateOnly bool) error {
	troopsJSON, err := json.Marshal(record.Troops)
	if err != nil {
		return err
	}
	remainingJSON, err := json.Marshal(record.RemainingTroops)
	if err != nil {
		return err
	}
	generalsJSON, _ := json.Marshal(record.Generals)
	lossesJSON, _ := json.Marshal(record.Losses)
	buffJSON, _ := json.Marshal(record.BuffSnapshot)
	rulesJSON, _ := json.Marshal(record.Rules)
	rewardJSON, _ := json.Marshal(record.RewardState)
	mailJSON, _ := json.Marshal(record.MailState)
	metadataJSON, _ := json.Marshal(record.Metadata)
	args := []any{
		record.ID, record.FromPlayerID, record.ToPlayerID, record.OwnerPlayerID, record.HostPlayerID,
		record.SourceType, record.SourceID, record.FromPlayerFaction, record.TargetType, record.TargetID, record.Status,
		troopsJSON, remainingJSON, nullableJSON(generalsJSON), nullableJSON(lossesJSON), nullableJSON(buffJSON), nullableJSON(rulesJSON),
		record.SpeedMultiplier, record.MarchSeconds, record.ReturnSeconds,
		parseNullableTime(record.SentAt), parseNullableTime(record.ArrivedAt), parseNullableTime(record.RecalledAt), parseNullableTime(record.ExpelledAt),
		parseNullableTime(record.ReturnStartedAt), parseNullableTime(record.ReturnedAt), nullableString(record.LastBattleReportID), parseNullableTime(record.LastBattleAt),
		record.IsAnnihilated, nullableJSON(rewardJSON), nullableJSON(mailJSON), nullableJSON(metadataJSON),
		parseNullableTime(record.CreatedAt), parseNullableTime(record.UpdatedAt),
	}
	if updateOnly {
		args = append(args, previousReinforcementID)
		_, err = tx.Exec(
			`UPDATE player_reinforcements
			 SET reinforcement_id = ?, from_player_id = ?, to_player_id = ?, owner_player_id = ?, host_player_id = ?,
				source_type = ?, source_id = ?, source_faction = ?, target_type = ?, target_id = ?, status = ?,
				troops_json = ?, remaining_troops_json = ?, generals_json = ?, losses_json = ?, buff_snapshot_json = ?, rules_json = ?,
				speed_multiplier = ?, march_seconds = ?, return_seconds = ?, sent_at = ?, arrived_at = ?, recalled_at = ?, expelled_at = ?,
				return_started_at = ?, returned_at = ?, last_battle_report_id = ?, last_battle_at = ?, is_annihilated = ?,
				reward_state_json = ?, mail_state_json = ?, metadata_json = ?, created_at = ?, updated_at = ?
			 WHERE reinforcement_id = ?`,
			args...,
		)
		return err
	}
	_, err = tx.Exec(
		`INSERT INTO player_reinforcements (
			reinforcement_id, from_player_id, to_player_id, owner_player_id, host_player_id,
			source_type, source_id, source_faction, target_type, target_id, status,
			troops_json, remaining_troops_json, generals_json, losses_json, buff_snapshot_json, rules_json,
			speed_multiplier, march_seconds, return_seconds, sent_at, arrived_at, recalled_at, expelled_at,
			return_started_at, returned_at, last_battle_report_id, last_battle_at, is_annihilated,
			reward_state_json, mail_state_json, metadata_json, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		args...,
	)
	return err
}

type reinforcementScanner interface {
	Scan(dest ...any) error
}

func scanReinforcementRows(rows *sql.Rows) ([]game.Reinforcement, error) {
	records := []game.Reinforcement{}
	for rows.Next() {
		record, err := scanReinforcement(rows)
		if err != nil {
			return nil, err
		}
		records = append(records, record)
	}
	return records, rows.Err()
}

func scanReinforcement(scanner reinforcementScanner) (game.Reinforcement, error) {
	var record game.Reinforcement
	var troopsJSON, remainingJSON []byte
	var generalsJSON, lossesJSON, buffJSON, rulesJSON, rewardJSON, mailJSON, metadataJSON sql.NullString
	var sentAt, arrivedAt, recalledAt, expelledAt, returnStartedAt, returnedAt, lastBattleAt, createdAt, updatedAt sql.NullTime
	var lastBattleReportID sql.NullString
	err := scanner.Scan(
		&record.ID, &record.FromPlayerID, &record.ToPlayerID, &record.OwnerPlayerID, &record.HostPlayerID,
		&record.SourceType, &record.SourceID, &record.FromPlayerFaction, &record.TargetType, &record.TargetID, &record.Status,
		&troopsJSON, &remainingJSON, &generalsJSON, &lossesJSON, &buffJSON, &rulesJSON,
		&record.SpeedMultiplier, &record.MarchSeconds, &record.ReturnSeconds,
		&sentAt, &arrivedAt, &recalledAt, &expelledAt, &returnStartedAt, &returnedAt,
		&lastBattleReportID, &lastBattleAt, &record.IsAnnihilated,
		&rewardJSON, &mailJSON, &metadataJSON, &createdAt, &updatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return game.Reinforcement{}, game.ErrReinforcementNotFound
	}
	if err != nil {
		return game.Reinforcement{}, err
	}
	_ = json.Unmarshal(troopsJSON, &record.Troops)
	_ = json.Unmarshal(remainingJSON, &record.RemainingTroops)
	unmarshalNullJSON(generalsJSON, &record.Generals)
	unmarshalNullJSON(lossesJSON, &record.Losses)
	unmarshalNullJSON(buffJSON, &record.BuffSnapshot)
	unmarshalNullJSON(rulesJSON, &record.Rules)
	unmarshalNullJSON(rewardJSON, &record.RewardState)
	unmarshalNullJSON(mailJSON, &record.MailState)
	unmarshalNullJSON(metadataJSON, &record.Metadata)
	record.SentAt = formatNullTime(sentAt)
	record.ArrivedAt = formatNullTime(arrivedAt)
	record.RecalledAt = formatNullTime(recalledAt)
	record.ExpelledAt = formatNullTime(expelledAt)
	record.ReturnStartedAt = formatNullTime(returnStartedAt)
	record.ReturnedAt = formatNullTime(returnedAt)
	record.LastBattleReportID = lastBattleReportID.String
	record.LastBattleAt = formatNullTime(lastBattleAt)
	record.CreatedAt = formatNullTime(createdAt)
	record.UpdatedAt = formatNullTime(updatedAt)
	if record.SentAt != "" && record.MarchSeconds > 0 {
		if sent, err := time.Parse(time.RFC3339, record.SentAt); err == nil {
			record.ExpectedArriveAt = sent.Add(time.Duration(record.MarchSeconds) * time.Second).UTC().Format(time.RFC3339)
		}
	}
	if record.ReturnStartedAt != "" && record.ReturnSeconds > 0 {
		if started, err := time.Parse(time.RFC3339, record.ReturnStartedAt); err == nil {
			record.ExpectedReturnedAt = started.Add(time.Duration(record.ReturnSeconds) * time.Second).UTC().Format(time.RFC3339)
		}
	}
	return record, nil
}

func nullableJSON(data []byte) any {
	if len(data) == 0 || bytes.Equal(data, []byte("null")) {
		return nil
	}
	return data
}

func nullableString(value string) any {
	if value == "" {
		return nil
	}
	return value
}

func parseNullableTime(value string) any {
	if value == "" {
		return nil
	}
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return nil
	}
	return parsed.UTC()
}

func formatNullTime(value sql.NullTime) string {
	if !value.Valid {
		return ""
	}
	return value.Time.UTC().Format(time.RFC3339)
}

func unmarshalNullJSON(value sql.NullString, target any) {
	if !value.Valid || value.String == "" {
		return
	}
	_ = json.Unmarshal([]byte(value.String), target)
}
