// 本文件实现 MySQL 版 NPC 扫荡任务仓储。
package storage

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"time"

	"hero3/internal/app/game"
)

// CreateNpcSweepTask 原子创建 NPC 扫荡任务；通过玩家行锁避免同一玩家并发创建多个活跃任务。
func (r *MySQLRepository) CreateNpcSweepTask(task game.NpcSweepTask) (game.NpcSweepTask, error) {
	task.Normalize()
	if task.ID == "" {
		task.ID = "sweep_task_" + gameRandomID(12)
	}
	now := time.Now().UTC()
	if task.CreatedAt == "" {
		task.CreatedAt = now.Format(time.RFC3339)
	}
	if task.UpdatedAt == "" {
		task.UpdatedAt = task.CreatedAt
	}
	npcIDsJSON, err := json.Marshal(task.NpcIDs)
	if err != nil {
		return game.NpcSweepTask{}, err
	}
	generalIDsJSON, err := json.Marshal(task.GeneralIDs)
	if err != nil {
		return game.NpcSweepTask{}, err
	}
	resultJSON, err := marshalSweepTaskResult(task.Result)
	if err != nil {
		return game.NpcSweepTask{}, err
	}
	createdAt := parseSweepTaskStorageTime(task.CreatedAt)
	updatedAt := parseSweepTaskStorageTime(task.UpdatedAt)
	startedAt := nullableSweepTaskStorageTime(task.StartedAt)
	completedAt := nullableSweepTaskStorageTime(task.CompletedAt)
	tx, err := r.db.Begin()
	if err != nil {
		return game.NpcSweepTask{}, err
	}
	defer func() { _ = tx.Rollback() }()

	var lockedPlayerID string
	if err := tx.QueryRow(`SELECT id FROM players WHERE id = ? FOR UPDATE`, task.PlayerID).Scan(&lockedPlayerID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return game.NpcSweepTask{}, game.ErrPlayerNotFound
		}
		return game.NpcSweepTask{}, err
	}
	activeRow := tx.QueryRow(`
		SELECT id, player_id, status, mode, npc_ids_json, general_ids_json, requested, done, failed, stopped,
		       error_message, result_json, created_at, updated_at, started_at, completed_at
		FROM npc_sweep_tasks
		WHERE player_id = ? AND status IN (?, ?)
		ORDER BY created_at DESC
		LIMIT 1
	`, task.PlayerID, game.SweepTaskStatusQueued, game.SweepTaskStatusRunning)
	active, err := scanNpcSweepTask(activeRow)
	if err == nil {
		return active, game.ErrNpcSweepTaskRunning
	}
	if !errors.Is(err, game.ErrNpcSweepTaskNotFound) {
		return game.NpcSweepTask{}, err
	}

	if _, err := tx.Exec(`
			INSERT INTO npc_sweep_tasks (
				id, player_id, status, mode, npc_ids_json, general_ids_json, requested, done, failed, stopped,
				error_message, result_json, created_at, updated_at, started_at, completed_at
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, task.ID, task.PlayerID, task.Status, task.Mode, npcIDsJSON, generalIDsJSON, task.Requested, task.Done, task.Failed, task.Stopped, task.Error, resultJSON, createdAt, updatedAt, startedAt, completedAt); err != nil {
		return game.NpcSweepTask{}, err
	}
	if err := tx.Commit(); err != nil {
		return game.NpcSweepTask{}, err
	}
	return task, nil
}

// GetNpcSweepTask 读取指定玩家的一键扫荡任务。
func (r *MySQLRepository) GetNpcSweepTask(playerID string, taskID string) (game.NpcSweepTask, error) {
	row := r.db.QueryRow(`
		SELECT id, player_id, status, mode, npc_ids_json, general_ids_json, requested, done, failed, stopped,
		       error_message, result_json, created_at, updated_at, started_at, completed_at
		FROM npc_sweep_tasks
		WHERE id = ? AND player_id = ?
		LIMIT 1
	`, taskID, playerID)
	return scanNpcSweepTask(row)
}

// FindActiveNpcSweepTask 查找玩家当前未完成的扫荡任务。
func (r *MySQLRepository) FindActiveNpcSweepTask(playerID string) (game.NpcSweepTask, bool, error) {
	row := r.db.QueryRow(`
		SELECT id, player_id, status, mode, npc_ids_json, general_ids_json, requested, done, failed, stopped,
		       error_message, result_json, created_at, updated_at, started_at, completed_at
		FROM npc_sweep_tasks
		WHERE player_id = ? AND status IN (?, ?)
		ORDER BY created_at DESC
		LIMIT 1
	`, playerID, game.SweepTaskStatusQueued, game.SweepTaskStatusRunning)
	task, err := scanNpcSweepTask(row)
	if errors.Is(err, game.ErrNpcSweepTaskNotFound) {
		return game.NpcSweepTask{}, false, nil
	}
	if err != nil {
		return game.NpcSweepTask{}, false, err
	}
	return task, true, nil
}

// UpdateNpcSweepTask 在事务内更新扫荡任务进度和结果。
func (r *MySQLRepository) UpdateNpcSweepTask(taskID string, updatedAt time.Time, update func(task *game.NpcSweepTask) error) (game.NpcSweepTask, error) {
	tx, err := r.db.Begin()
	if err != nil {
		return game.NpcSweepTask{}, err
	}
	defer func() { _ = tx.Rollback() }()

	row := tx.QueryRow(`
		SELECT id, player_id, status, mode, npc_ids_json, general_ids_json, requested, done, failed, stopped,
		       error_message, result_json, created_at, updated_at, started_at, completed_at
		FROM npc_sweep_tasks
		WHERE id = ?
		FOR UPDATE
	`, taskID)
	task, err := scanNpcSweepTask(row)
	if err != nil {
		return game.NpcSweepTask{}, err
	}
	if update != nil {
		if err := update(&task); err != nil {
			return game.NpcSweepTask{}, err
		}
	}
	task.Normalize()
	if updatedAt.IsZero() {
		updatedAt = time.Now()
	}
	task.UpdatedAt = updatedAt.UTC().Format(time.RFC3339)
	npcIDsJSON, err := json.Marshal(task.NpcIDs)
	if err != nil {
		return game.NpcSweepTask{}, err
	}
	generalIDsJSON, err := json.Marshal(task.GeneralIDs)
	if err != nil {
		return game.NpcSweepTask{}, err
	}
	resultJSON, err := marshalSweepTaskResult(task.Result)
	if err != nil {
		return game.NpcSweepTask{}, err
	}
	if _, err := tx.Exec(`
		UPDATE npc_sweep_tasks
		SET status = ?, mode = ?, npc_ids_json = ?, general_ids_json = ?, requested = ?, done = ?, failed = ?,
		    stopped = ?, error_message = ?, result_json = ?, updated_at = ?, started_at = ?, completed_at = ?
		WHERE id = ?
	`, task.Status, task.Mode, npcIDsJSON, generalIDsJSON, task.Requested, task.Done, task.Failed, task.Stopped, task.Error, resultJSON, parseSweepTaskStorageTime(task.UpdatedAt), nullableSweepTaskStorageTime(task.StartedAt), nullableSweepTaskStorageTime(task.CompletedAt), task.ID); err != nil {
		return game.NpcSweepTask{}, err
	}
	if err := tx.Commit(); err != nil {
		return game.NpcSweepTask{}, err
	}
	return task, nil
}

// FailActiveNpcSweepTasks 把进程重启后无人执行的活跃扫荡任务统一置为失败。
func (r *MySQLRepository) FailActiveNpcSweepTasks(updatedAt time.Time, reason string) (int, error) {
	if updatedAt.IsZero() {
		updatedAt = time.Now()
	}
	if reason == "" {
		reason = "sweep task interrupted by service restart"
	}
	result, err := r.db.Exec(`
		UPDATE npc_sweep_tasks
		SET status = ?, error_message = ?, updated_at = ?, completed_at = ?
		WHERE status IN (?, ?)
	`, game.SweepTaskStatusFailed, reason, updatedAt.UTC(), updatedAt.UTC(), game.SweepTaskStatusQueued, game.SweepTaskStatusRunning)
	if err != nil {
		return 0, err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return 0, err
	}
	return int(affected), nil
}

// DeleteNpcSweepTasksCompletedBefore 删除过期的已完成或失败扫荡任务，控制任务表体积。
func (r *MySQLRepository) DeleteNpcSweepTasksCompletedBefore(cutoff time.Time) (int, error) {
	result, err := r.db.Exec(`
		DELETE FROM npc_sweep_tasks
		WHERE status IN (?, ?) AND completed_at IS NOT NULL AND completed_at < ?
	`, game.SweepTaskStatusCompleted, game.SweepTaskStatusFailed, cutoff.UTC())
	if err != nil {
		return 0, err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return 0, err
	}
	return int(affected), nil
}

type sweepTaskScanner interface {
	Scan(dest ...any) error
}

// scanNpcSweepTask 从 SQL 行读取扫荡任务。
func scanNpcSweepTask(row sweepTaskScanner) (game.NpcSweepTask, error) {
	var task game.NpcSweepTask
	var npcIDsJSON []byte
	var generalIDsJSON []byte
	var resultJSON sql.NullString
	var startedAt sql.NullTime
	var completedAt sql.NullTime
	var createdAt time.Time
	var updatedAt time.Time
	err := row.Scan(
		&task.ID, &task.PlayerID, &task.Status, &task.Mode, &npcIDsJSON, &generalIDsJSON,
		&task.Requested, &task.Done, &task.Failed, &task.Stopped, &task.Error, &resultJSON,
		&createdAt, &updatedAt, &startedAt, &completedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return game.NpcSweepTask{}, game.ErrNpcSweepTaskNotFound
	}
	if err != nil {
		return game.NpcSweepTask{}, err
	}
	if err := json.Unmarshal(npcIDsJSON, &task.NpcIDs); err != nil {
		return game.NpcSweepTask{}, err
	}
	if err := json.Unmarshal(generalIDsJSON, &task.GeneralIDs); err != nil {
		return game.NpcSweepTask{}, err
	}
	if resultJSON.Valid && resultJSON.String != "" {
		var result game.SweepNpcResponse
		if err := json.Unmarshal([]byte(resultJSON.String), &result); err != nil {
			return game.NpcSweepTask{}, err
		}
		task.Result = &result
	}
	task.CreatedAt = createdAt.UTC().Format(time.RFC3339)
	task.UpdatedAt = updatedAt.UTC().Format(time.RFC3339)
	if startedAt.Valid {
		task.StartedAt = startedAt.Time.UTC().Format(time.RFC3339)
	}
	if completedAt.Valid {
		task.CompletedAt = completedAt.Time.UTC().Format(time.RFC3339)
	}
	task.Normalize()
	return task, nil
}

// marshalSweepTaskResult 把任务结果编码成可空 JSON。
func marshalSweepTaskResult(result *game.SweepNpcResponse) (any, error) {
	if result == nil {
		return nil, nil
	}
	return json.Marshal(result)
}

// parseStorageTime 解析任务时间，失败时回退当前时间。
func parseSweepTaskStorageTime(value string) time.Time {
	if parsed, err := time.Parse(time.RFC3339, value); err == nil {
		return parsed.UTC()
	}
	return time.Now().UTC()
}

// nullableStorageTime 把可选时间字符串转为 SQL 可空时间。
func nullableSweepTaskStorageTime(value string) sql.NullTime {
	if value == "" {
		return sql.NullTime{}
	}
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return sql.NullTime{}
	}
	return sql.NullTime{Time: parsed.UTC(), Valid: true}
}

// gameRandomID 为 storage 包生成任务 ID，避免向外暴露 game.randomID。
func gameRandomID(bytesCount int) string {
	return time.Now().UTC().Format("20060102150405") + randomStorageSuffix(bytesCount)
}

// randomStorageSuffix 生成 storage 包内部使用的随机十六进制后缀。
func randomStorageSuffix(bytesCount int) string {
	if bytesCount <= 0 {
		bytesCount = 8
	}
	buf := make([]byte, bytesCount)
	if _, err := rand.Read(buf); err != nil {
		return "fallback"
	}
	return hex.EncodeToString(buf)
}
