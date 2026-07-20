// 本文件归口 MySQL 轮回绝境副本状态仓储。
package storage

import (
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"hero3/internal/app/game"
)

// GetActiveReincarnationRun 获取玩家进行中的轮回实例。
func (r *MySQLRepository) GetActiveReincarnationRun(playerID string, now time.Time) (game.ReincarnationRun, bool, error) {
	row := r.db.QueryRow(
		`SELECT metadata_json FROM reincarnation_runs
		 WHERE player_id = ? AND status = ?
		 ORDER BY started_at DESC LIMIT 1`,
		playerID, game.ReincarnationRunRunning,
	)
	var data []byte
	if err := row.Scan(&data); errors.Is(err, sql.ErrNoRows) {
		return game.ReincarnationRun{}, false, nil
	} else if err != nil {
		return game.ReincarnationRun{}, false, err
	}
	var run game.ReincarnationRun
	if err := json.Unmarshal(data, &run); err != nil {
		return game.ReincarnationRun{}, false, err
	}
	return run, true, nil
}

// GetReincarnationRun 获取指定轮回实例。
func (r *MySQLRepository) GetReincarnationRun(runID string) (game.ReincarnationRun, error) {
	var data []byte
	err := r.db.QueryRow(`SELECT metadata_json FROM reincarnation_runs WHERE id = ? LIMIT 1`, runID).Scan(&data)
	if errors.Is(err, sql.ErrNoRows) {
		return game.ReincarnationRun{}, game.ErrReincarnationRunNotFound
	}
	if err != nil {
		return game.ReincarnationRun{}, err
	}
	var run game.ReincarnationRun
	if err := json.Unmarshal(data, &run); err != nil {
		return game.ReincarnationRun{}, err
	}
	return run, nil
}

// SaveReincarnationRun 保存轮回实例及其波次、战斗快照。
func (r *MySQLRepository) SaveReincarnationRun(run game.ReincarnationRun) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if err := upsertReincarnationRunTx(tx, run); err != nil {
		return err
	}
	if err := syncReincarnationWavesTx(tx, run); err != nil {
		return err
	}
	if err := syncReincarnationBattlesTx(tx, run); err != nil {
		return err
	}
	return tx.Commit()
}

// UpdateReincarnationRunWithState 更新玩家战斗资产和轮回实例。
func (r *MySQLRepository) UpdateReincarnationRunWithState(playerID string, runID string, updatedAt time.Time, update func(state *game.GameState, run *game.ReincarnationRun) ([]game.BattleReport, error)) (game.GameState, game.ReincarnationRun, []game.BattleReport, error) {
	tx, err := r.db.Begin()
	if err != nil {
		return game.GameState{}, game.ReincarnationRun{}, nil, err
	}
	defer func() { _ = tx.Rollback() }()

	var runJSON []byte
	err = tx.QueryRow(`SELECT metadata_json FROM reincarnation_runs WHERE id = ? LIMIT 1 FOR UPDATE`, runID).Scan(&runJSON)
	if errors.Is(err, sql.ErrNoRows) {
		return game.GameState{}, game.ReincarnationRun{}, nil, game.ErrReincarnationRunNotFound
	}
	if err != nil {
		return game.GameState{}, game.ReincarnationRun{}, nil, err
	}
	var run game.ReincarnationRun
	if err := json.Unmarshal(runJSON, &run); err != nil {
		return game.GameState{}, game.ReincarnationRun{}, nil, err
	}
	if run.PlayerID != playerID {
		return game.GameState{}, game.ReincarnationRun{}, nil, game.ErrReincarnationRunNotFound
	}
	var reports []game.BattleReport
	state, err := updateItemStateTx(tx, playerID, updatedAt, func(state *game.GameState) error {
		nextReports, err := update(state, &run)
		if err != nil {
			return err
		}
		reports = nextReports
		return nil
	})
	if err != nil {
		return game.GameState{}, game.ReincarnationRun{}, nil, err
	}
	for _, report := range reports {
		sourceID := report.EventID
		if sourceID == "" {
			sourceID = report.ID
		}
		if err := upsertCapturedGarrisonTx(tx, state, report.DefenderFaction, report.CapturedToGarrison, sourceID, updatedAt); err != nil {
			return game.GameState{}, game.ReincarnationRun{}, nil, err
		}
	}
	if err := upsertReincarnationRunTx(tx, run); err != nil {
		return game.GameState{}, game.ReincarnationRun{}, nil, err
	}
	if err := syncReincarnationWavesTx(tx, run); err != nil {
		return game.GameState{}, game.ReincarnationRun{}, nil, err
	}
	if err := syncReincarnationBattlesTx(tx, run); err != nil {
		return game.GameState{}, game.ReincarnationRun{}, nil, err
	}
	if len(reports) > 0 {
		for _, report := range reports {
			if err := insertBattleReportTx(tx, report); err != nil {
				return game.GameState{}, game.ReincarnationRun{}, nil, err
			}
		}
	}
	if err := tx.Commit(); err != nil {
		return game.GameState{}, game.ReincarnationRun{}, nil, err
	}
	return state, run, reports, nil
}

// ListReincarnationRuns 查询轮回实例列表。
func (r *MySQLRepository) ListReincarnationRuns(playerID string, limit int, offset int) ([]game.ReincarnationRun, int, error) {
	if limit <= 0 {
		limit = 20
	}
	var total int
	args := []any{}
	where := ""
	if playerID != "" {
		where = " WHERE player_id = ?"
		args = append(args, playerID)
	}
	if err := r.db.QueryRow(`SELECT COUNT(*) FROM reincarnation_runs`+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	args = append(args, limit, offset)
	rows, err := r.db.Query(`SELECT metadata_json FROM reincarnation_runs`+where+` ORDER BY started_at DESC LIMIT ? OFFSET ?`, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	runs := []game.ReincarnationRun{}
	for rows.Next() {
		var data []byte
		if err := rows.Scan(&data); err != nil {
			return nil, 0, err
		}
		var run game.ReincarnationRun
		if err := json.Unmarshal(data, &run); err != nil {
			return nil, 0, err
		}
		runs = append(runs, run)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return runs, total, nil
}

func upsertReincarnationRunTx(tx *sql.Tx, run game.ReincarnationRun) error {
	runJSON, err := json.Marshal(run)
	if err != nil {
		return err
	}
	rewardsJSON, err := json.Marshal(run.PendingRewards)
	if err != nil {
		return err
	}
	_, err = tx.Exec(
		`INSERT INTO reincarnation_runs (
			id, player_id, level, status, current_wave, started_at, expires_at, completed_at, failed_at,
			ended_reason, pending_rewards_json, reward_granted_at, metadata_json, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE
			status = VALUES(status), current_wave = VALUES(current_wave), completed_at = VALUES(completed_at),
			failed_at = VALUES(failed_at), ended_reason = VALUES(ended_reason),
			pending_rewards_json = VALUES(pending_rewards_json), reward_granted_at = VALUES(reward_granted_at),
			metadata_json = VALUES(metadata_json), updated_at = VALUES(updated_at)`,
		run.ID, run.PlayerID, run.Level, run.Status, run.CurrentWave, run.StartedAt.UTC(), run.ExpiresAt.UTC(),
		nullableTime(run.CompletedAt), nullableTime(run.FailedAt), run.EndedReason, rewardsJSON,
		nullableTime(run.RewardGrantedAt), runJSON, run.CreatedAt.UTC(), run.UpdatedAt.UTC(),
	)
	return err
}

func syncReincarnationWavesTx(tx *sql.Tx, run game.ReincarnationRun) error {
	for _, wave := range run.Waves {
		enemyTroops, _ := json.Marshal(wave.EnemyTroops)
		enemyRemaining, _ := json.Marshal(wave.EnemyRemaining)
		allyBonus, _ := json.Marshal(wave.AllyBonus)
		enemyBonus, _ := json.Marshal(wave.EnemyBonus)
		rewardPreview, _ := json.Marshal(wave.RewardPreview)
		rewardResult, _ := json.Marshal(wave.RewardResult)
		if _, err := tx.Exec(
			`INSERT INTO reincarnation_waves (
				id, run_id, wave_index, wave_type, enemy_faction, enemy_troops_json, enemy_remaining_json,
				ally_bonus_json, enemy_bonus_json, reward_preview_json, reward_result_json, status,
				started_at, cleared_at, created_at, updated_at
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
			ON DUPLICATE KEY UPDATE
				enemy_remaining_json = VALUES(enemy_remaining_json), reward_result_json = VALUES(reward_result_json),
				status = VALUES(status), cleared_at = VALUES(cleared_at), updated_at = VALUES(updated_at)`,
			wave.ID, run.ID, wave.WaveIndex, wave.WaveType, wave.EnemyFaction, enemyTroops, enemyRemaining,
			allyBonus, enemyBonus, rewardPreview, rewardResult, wave.Status, wave.StartedAt.UTC(),
			nullableTime(wave.ClearedAt), run.CreatedAt.UTC(), run.UpdatedAt.UTC(),
		); err != nil {
			return err
		}
	}
	return nil
}

func syncReincarnationBattlesTx(tx *sql.Tx, run game.ReincarnationRun) error {
	for _, battle := range run.Battles {
		troops, _ := json.Marshal(battle.AttackTroops)
		losses, _ := json.Marshal(battle.Losses)
		enemyLosses, _ := json.Marshal(battle.EnemyLosses)
		resultJSON, _ := json.Marshal(battle)
		if _, err := tx.Exec(
			`INSERT IGNORE INTO reincarnation_battles (
				id, run_id, wave_id, player_id, client_action_id, wave_index, wave_type, attack_troops_json,
				losses_json, enemy_losses_json, result_json, report_id, created_at
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			battle.ID, battle.RunID, battle.WaveID, battle.PlayerID, battle.ClientActionID, battle.WaveIndex, battle.WaveType,
			troops, losses, enemyLosses, resultJSON, battle.ReportID, battle.CreatedAt.UTC(),
		); err != nil {
			return err
		}
	}
	return nil
}

func nullableTime(value *time.Time) any {
	if value == nil || value.IsZero() {
		return nil
	}
	return value.UTC()
}
