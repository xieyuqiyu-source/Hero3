// 本文件归口 MySQL 小游戏记录仓储方法。
package storage

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"hero3/internal/app/game"
)

func (r *MySQLRepository) SaveMiniGameRecord(record game.MiniGameRecord) error {
	createdAt, _ := time.Parse(time.RFC3339, record.CreatedAt)
	if createdAt.IsZero() {
		createdAt = time.Now()
	}
	if (record.GameType == "fishing" || record.GameType == "gambling") && record.RemainingAmount == 0 && record.RewardAmount > 0 {
		record.RemainingAmount = record.RewardAmount
	}

	_, err := r.db.Exec(
		`INSERT INTO minigame_records (id, player_id, game_type, result_name, rarity, reward_unit, reward_amount, remaining_amount, bet_unit, bet_amount, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		record.ID,
		record.PlayerID,
		record.GameType,
		record.ResultName,
		record.Rarity,
		record.RewardUnit,
		record.RewardAmount,
		record.RemainingAmount,
		record.BetUnit,
		record.BetAmount,
		createdAt.UTC(),
	)
	return err
}

func (r *MySQLRepository) ListMiniGameRecords(playerID string, gameType string, limit int, offset int) ([]game.MiniGameRecord, int, error) {
	if limit <= 0 {
		limit = 100
	}
	if limit > 500 {
		limit = 500
	}
	if offset < 0 {
		offset = 0
	}

	where := `WHERE player_id = ?`
	args := []any{playerID}
	if gameType != "" {
		where += ` AND game_type = ?`
		args = append(args, gameType)
	}

	var total int
	countArgs := append([]any{}, args...)
	if err := r.db.QueryRow(
		`SELECT COUNT(*)
		 FROM minigame_records
		 `+where,
		countArgs...,
	).Scan(&total); err != nil {
		return nil, 0, err
	}

	queryArgs := append(append([]any{}, args...), limit, offset)
	rows, err := r.db.Query(
		`SELECT id, player_id, game_type, result_name, rarity, reward_unit, reward_amount, remaining_amount, bet_unit, bet_amount, created_at
		 FROM minigame_records
		 `+where+`
		 ORDER BY created_at DESC
		 LIMIT ? OFFSET ?`,
		queryArgs...,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	records := []game.MiniGameRecord{}
	for rows.Next() {
		var r game.MiniGameRecord
		var createdAt time.Time
		if err := rows.Scan(&r.ID, &r.PlayerID, &r.GameType, &r.ResultName, &r.Rarity, &r.RewardUnit, &r.RewardAmount, &r.RemainingAmount, &r.BetUnit, &r.BetAmount, &createdAt); err != nil {
			return nil, 0, err
		}
		r.CreatedAt = createdAt.UTC().Format(time.RFC3339)
		records = append(records, r)
	}
	return records, total, rows.Err()
}

func (r *MySQLRepository) UpdateMiniGamePlayerState(playerID string, updatedAt time.Time, update func(state *game.GameState, records []game.MiniGameRecord) ([]game.MiniGameRecord, error)) (game.GameState, []game.MiniGameRecord, error) {
	tx, err := r.db.Begin()
	if err != nil {
		return game.GameState{}, nil, err
	}
	defer func() { _ = tx.Rollback() }()

	var stateJSON []byte
	var mailCode string
	err = tx.QueryRow(`SELECT state_json, mail_code FROM players WHERE id = ? LIMIT 1 FOR UPDATE`, playerID).Scan(&stateJSON, &mailCode)
	if errors.Is(err, sql.ErrNoRows) {
		return game.GameState{}, nil, game.ErrPlayerNotFound
	}
	if err != nil {
		return game.GameState{}, nil, err
	}
	var state game.GameState
	if err = json.Unmarshal(stateJSON, &state); err != nil {
		return game.GameState{}, nil, err
	}
	if state.Player.MailCode == "" {
		state.Player.MailCode = mailCode
	}
	if err := overlayAuthoritativeCurrencyTx(tx, &state, playerID, updatedAt); err != nil {
		return game.GameState{}, nil, err
	}
	if err := overlayAuthoritativeArmyTx(tx, &state, playerID); err != nil {
		return game.GameState{}, nil, err
	}
	previousCurrencySnapshot := currencySnapshotFromState(state)
	previousArmySnapshot := armySnapshotsFromStorageState(state.Army)

	rows, err := tx.Query(
		`SELECT id, player_id, game_type, result_name, rarity, reward_unit, reward_amount, remaining_amount, bet_unit, bet_amount, created_at
		 FROM minigame_records
		 WHERE player_id = ?
		 ORDER BY created_at DESC
		 FOR UPDATE`,
		playerID,
	)
	if err != nil {
		return game.GameState{}, nil, err
	}

	records := []game.MiniGameRecord{}
	previousRemainingByRecordID := map[string]int{}
	for rows.Next() {
		var record game.MiniGameRecord
		var createdAt time.Time
		if err := rows.Scan(
			&record.ID,
			&record.PlayerID,
			&record.GameType,
			&record.ResultName,
			&record.Rarity,
			&record.RewardUnit,
			&record.RewardAmount,
			&record.RemainingAmount,
			&record.BetUnit,
			&record.BetAmount,
			&createdAt,
		); err != nil {
			return game.GameState{}, nil, err
		}
		record.CreatedAt = createdAt.UTC().Format(time.RFC3339)
		records = append(records, record)
		previousRemainingByRecordID[record.ID] = record.RemainingAmount
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return game.GameState{}, nil, err
	}
	if err := rows.Close(); err != nil {
		return game.GameState{}, nil, err
	}

	if update != nil {
		records, err = update(&state, records)
		if err != nil {
			return game.GameState{}, nil, err
		}
	}
	for _, record := range records {
		if _, exists := previousRemainingByRecordID[record.ID]; !exists {
			createdAt, _ := time.Parse(time.RFC3339, record.CreatedAt)
			if createdAt.IsZero() {
				createdAt = updatedAt
			}
			if _, err := tx.Exec(
				`INSERT INTO minigame_records (id, player_id, game_type, result_name, rarity, reward_unit, reward_amount, remaining_amount, bet_unit, bet_amount, created_at)
				 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
				record.ID,
				record.PlayerID,
				record.GameType,
				record.ResultName,
				record.Rarity,
				record.RewardUnit,
				record.RewardAmount,
				record.RemainingAmount,
				record.BetUnit,
				record.BetAmount,
				createdAt.UTC(),
			); err != nil {
				return game.GameState{}, nil, err
			}
			continue
		}
		if !miniGameRemainingAmountChanged(previousRemainingByRecordID, record) {
			continue
		}
		if _, err := tx.Exec(
			`UPDATE minigame_records SET remaining_amount = ? WHERE id = ? AND player_id = ?`,
			record.RemainingAmount, record.ID, playerID,
		); err != nil {
			return game.GameState{}, nil, err
		}
	}

	nextStateJSON, err := marshalPlayerStateSnapshot(state)
	if err != nil {
		return game.GameState{}, nil, err
	}
	if armySnapshotChanged(previousArmySnapshot, state.Army) {
		if err := syncPlayerArmyTx(tx, playerID, state.Army, updatedAt.UTC()); err != nil {
			return game.GameState{}, nil, err
		}
	}
	if currencySnapshotChanged(previousCurrencySnapshot, state) {
		if err := syncPlayerCurrencyTx(tx, playerID, &state, updatedAt.UTC()); err != nil {
			return game.GameState{}, nil, err
		}
	}
	if !bytes.Equal(stateJSON, nextStateJSON) {
		if _, err = tx.Exec(
			`UPDATE players
			 SET nickname = ?, faction = ?, mail_code = ?, state_json = ?, updated_at = ?
			 WHERE id = ?`,
			state.Player.Nickname,
			state.Player.Faction,
			state.Player.MailCode,
			nextStateJSON,
			updatedAt.UTC(),
			playerID,
		); err != nil {
			return game.GameState{}, nil, err
		}
	}

	if err = tx.Commit(); err != nil {
		return game.GameState{}, nil, err
	}
	return state, records, nil
}

// miniGameRemainingAmountChanged 判断小游戏库存数量是否真正变化。
func miniGameRemainingAmountChanged(previous map[string]int, record game.MiniGameRecord) bool {
	before, ok := previous[record.ID]
	return !ok || before != record.RemainingAmount
}

// --- Gold Ledger Methods ---
