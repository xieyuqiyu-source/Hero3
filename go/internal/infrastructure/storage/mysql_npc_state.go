// 本文件归口玩家 NPC 城池状态的 MySQL 权威存储。
package storage

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"hero3/internal/app/game"
)

type rowQueryer interface {
	QueryRow(query string, args ...any) *sql.Row
}

// overlayAuthoritativeNpcState 使用 NPC 状态权威表覆盖兼容快照；旧玩家缺表时保留 state_json 兜底。
func (r *MySQLRepository) overlayAuthoritativeNpcState(state *game.GameState, playerID string) error {
	npcState, found, err := loadPlayerNpcState(r.db, playerID)
	if err != nil {
		return err
	}
	applyAuthoritativeNpcState(state, npcState, found)
	return nil
}

// UpdateNpcState 只锁定并写回玩家 NPC 城池状态，不刷新 players 主行。
func (r *MySQLRepository) UpdateNpcState(playerID string, updatedAt time.Time, update func(state *game.GameState) error) (game.GameState, error) {
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
	if err := json.Unmarshal(stateJSON, &state); err != nil {
		return game.GameState{}, err
	}
	if state.Player.MailCode == "" {
		state.Player.MailCode = mailCode
	}
	if err := overlayAuthoritativeNpcStateTx(tx, &state, playerID, updatedAt); err != nil {
		return game.GameState{}, err
	}
	previousNpcStateJSON, err := json.Marshal(state.NpcState)
	if err != nil {
		return game.GameState{}, err
	}

	if update != nil {
		if err := update(&state); err != nil {
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
	if err := tx.Commit(); err != nil {
		return game.GameState{}, err
	}
	return state, nil
}

// overlayAuthoritativeNpcStateTx 在事务内锁定 NPC 状态；旧玩家缺表时把旧快照回填到权威表。
func overlayAuthoritativeNpcStateTx(tx *sql.Tx, state *game.GameState, playerID string, updatedAt time.Time) error {
	npcState, found, err := loadPlayerNpcStateTx(tx, playerID)
	if err != nil {
		return err
	}
	if found {
		applyAuthoritativeNpcState(state, npcState, true)
		return nil
	}
	if state.NpcState == nil {
		return nil
	}
	if err := syncPlayerNpcStateTx(tx, playerID, state.NpcState, updatedAt.UTC()); err != nil {
		return err
	}
	return nil
}

// applyAuthoritativeNpcState 将 NPC 权威表结果写回 GameState。
func applyAuthoritativeNpcState(state *game.GameState, npcState *game.NpcState, found bool) {
	if !found {
		return
	}
	state.NpcState = npcState
}

// loadPlayerNpcState 从 player_npc_states 读取玩家 NPC 城池状态。
func loadPlayerNpcState(queryer rowQueryer, playerID string) (*game.NpcState, bool, error) {
	return loadPlayerNpcStateWithQuery(queryer, playerID, "")
}

// loadPlayerNpcStateTx 在事务内读取并锁定玩家 NPC 城池状态。
func loadPlayerNpcStateTx(tx *sql.Tx, playerID string) (*game.NpcState, bool, error) {
	return loadPlayerNpcStateWithQuery(tx, playerID, " FOR UPDATE")
}

// loadPlayerNpcStateWithQuery 读取 NPC 状态 JSON 并还原结构。
func loadPlayerNpcStateWithQuery(queryer rowQueryer, playerID string, lockClause string) (*game.NpcState, bool, error) {
	var stateJSON []byte
	err := queryer.QueryRow(
		`SELECT npc_state_json
		 FROM player_npc_states
		 WHERE player_id = ?
		 LIMIT 1`+lockClause,
		playerID,
	).Scan(&stateJSON)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	var npcState game.NpcState
	if err := json.Unmarshal(stateJSON, &npcState); err != nil {
		return nil, false, err
	}
	return &npcState, true, nil
}

// syncPlayerNpcStateTx 把玩家 NPC 城池状态写入独立权威表。
func syncPlayerNpcStateTx(tx *sql.Tx, playerID string, npcState *game.NpcState, updatedAt time.Time) error {
	if npcState == nil {
		_, err := tx.Exec(`DELETE FROM player_npc_states WHERE player_id = ?`, playerID)
		return err
	}
	stateJSON, err := json.Marshal(npcState)
	if err != nil {
		return err
	}
	_, err = tx.Exec(
		`INSERT INTO player_npc_states (player_id, npc_state_json, updated_at)
		 VALUES (?, ?, ?)
		 ON DUPLICATE KEY UPDATE
			npc_state_json = VALUES(npc_state_json),
			updated_at = VALUES(updated_at)`,
		playerID,
		stateJSON,
		updatedAt.UTC(),
	)
	return err
}
