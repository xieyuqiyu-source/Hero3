// 本文件归口 MySQL 玩家 Buff/Modifier 权威表同步。
package storage

import (
	"database/sql"
	"errors"
	"sort"
	"strings"
	"time"

	"hero3/internal/app/game"
)

var errPlayerBuffsMissing = errors.New("player_buffs rows missing; run backfill-buffs before using buffs as authoritative state")

// overlayAuthoritativeBuffs 用 player_buffs 权威表覆盖兼容快照中的 Buff。
func (r *MySQLRepository) overlayAuthoritativeBuffs(state *game.GameState, playerID string) error {
	buffs, found, err := loadPlayerBuffs(r.db, playerID)
	if err != nil {
		return err
	}
	return applyAuthoritativeBuffs(state, buffs, found)
}

// overlayAuthoritativeBuffsTx 在事务内锁定并加载玩家 Buff 权威状态。
func overlayAuthoritativeBuffsTx(tx *sql.Tx, state *game.GameState, playerID string) error {
	buffs, found, err := loadPlayerBuffsTx(tx, playerID)
	if err != nil {
		return err
	}
	return applyAuthoritativeBuffs(state, buffs, found)
}

func applyAuthoritativeBuffs(state *game.GameState, buffs []game.Buff, found bool) error {
	if state == nil {
		return game.ErrPlayerNotFound
	}
	if !found {
		if len(state.Buffs) == 0 {
			state.Buffs = []game.Buff{}
			return nil
		}
		return errPlayerBuffsMissing
	}
	state.Buffs = buffs
	return nil
}

func loadPlayerBuffs(queryer resourceQueryer, playerID string) ([]game.Buff, bool, error) {
	return loadPlayerBuffsWithQuery(queryer, playerID, "")
}

func loadPlayerBuffsTx(tx *sql.Tx, playerID string) ([]game.Buff, bool, error) {
	return loadPlayerBuffsWithQuery(tx, playerID, " FOR UPDATE")
}

func loadPlayerBuffsWithQuery(queryer resourceQueryer, playerID string, lockClause string) ([]game.Buff, bool, error) {
	rows, err := queryer.Query(
		`SELECT buff_id, source, modifier_key, modifier_value, modifier_mode, expires_at, created_at, note
		 FROM player_buffs
		 WHERE player_id = ?
		 ORDER BY created_at, buff_id`+lockClause,
		playerID,
	)
	if err != nil {
		return nil, false, err
	}
	defer rows.Close()
	buffs := []game.Buff{}
	for rows.Next() {
		var buff game.Buff
		var expiresAt sql.NullTime
		var createdAt time.Time
		if err := rows.Scan(&buff.ID, &buff.Source, &buff.Key, &buff.Value, &buff.Mode, &expiresAt, &createdAt, &buff.Note); err != nil {
			return nil, false, err
		}
		buff.ID = strings.TrimSpace(buff.ID)
		buff.Key = strings.TrimSpace(buff.Key)
		buff.Mode = strings.TrimSpace(buff.Mode)
		if buff.ID == "" || buff.Key == "" || buff.Mode == "" {
			continue
		}
		if expiresAt.Valid {
			buff.ExpiresAt = expiresAt.Time.UTC().Format(time.RFC3339)
		}
		buff.CreatedAt = createdAt.UTC().Format(time.RFC3339)
		buffs = append(buffs, buff)
	}
	if err := rows.Err(); err != nil {
		return nil, false, err
	}
	return buffs, len(buffs) > 0, nil
}

func syncPlayerBuffsTx(tx *sql.Tx, playerID string, buffs []game.Buff, updatedAt time.Time) error {
	buffIDs := buffIDsFromState(buffs)
	if len(buffIDs) == 0 {
		_, err := tx.Exec(`DELETE FROM player_buffs WHERE player_id = ?`, playerID)
		return err
	}
	byID := buffsByID(buffs)
	for _, buffID := range buffIDs {
		buff := byID[buffID]
		createdAt := parseStorageTime(buff.CreatedAt)
		if !createdAt.Valid {
			createdAt = sql.NullTime{Time: updatedAt.UTC(), Valid: true}
		}
		if _, err := tx.Exec(
			`INSERT INTO player_buffs (player_id, buff_id, source, modifier_key, modifier_value, modifier_mode, expires_at, created_at, note, updated_at)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
			 ON DUPLICATE KEY UPDATE
				source = VALUES(source),
				modifier_key = VALUES(modifier_key),
				modifier_value = VALUES(modifier_value),
				modifier_mode = VALUES(modifier_mode),
				expires_at = VALUES(expires_at),
				created_at = VALUES(created_at),
				note = VALUES(note),
				updated_at = VALUES(updated_at)`,
			playerID,
			buff.ID,
			buff.Source,
			buff.Key,
			buff.Value,
			buff.Mode,
			nullableTimeArg(parseStorageTime(buff.ExpiresAt)),
			nullableTimeArg(createdAt),
			buff.Note,
			updatedAt.UTC(),
		); err != nil {
			return err
		}
	}
	return deleteStalePlayerBuffsTx(tx, playerID, buffIDs)
}

type storageBuffSnapshot struct {
	Source    string
	Key       string
	Value     float64
	Mode      string
	ExpiresAt string
	CreatedAt string
	Note      string
}

func buffSnapshotChanged(before map[string]storageBuffSnapshot, after []game.Buff) bool {
	return !buffSnapshotMapsEqual(before, buffSnapshotsFromStorageState(after))
}

func buffSnapshotsFromStorageState(buffs []game.Buff) map[string]storageBuffSnapshot {
	result := map[string]storageBuffSnapshot{}
	for _, buff := range buffs {
		buff.ID = strings.TrimSpace(buff.ID)
		if buff.ID == "" {
			continue
		}
		result[buff.ID] = storageBuffSnapshot{
			Source:    buff.Source,
			Key:       buff.Key,
			Value:     buff.Value,
			Mode:      buff.Mode,
			ExpiresAt: strings.TrimSpace(buff.ExpiresAt),
			CreatedAt: strings.TrimSpace(buff.CreatedAt),
			Note:      buff.Note,
		}
	}
	return result
}

func buffSnapshotMapsEqual(a map[string]storageBuffSnapshot, b map[string]storageBuffSnapshot) bool {
	if len(a) != len(b) {
		return false
	}
	for key, left := range a {
		if right, ok := b[key]; !ok || left != right {
			return false
		}
	}
	return true
}

func buffIDsFromState(buffs []game.Buff) []string {
	idSet := map[string]struct{}{}
	for _, buff := range buffs {
		buffID := strings.TrimSpace(buff.ID)
		if buffID == "" || strings.TrimSpace(buff.Key) == "" || strings.TrimSpace(buff.Mode) == "" {
			continue
		}
		idSet[buffID] = struct{}{}
	}
	ids := make([]string, 0, len(idSet))
	for id := range idSet {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

func buffsByID(buffs []game.Buff) map[string]game.Buff {
	result := map[string]game.Buff{}
	for _, buff := range buffs {
		buff.ID = strings.TrimSpace(buff.ID)
		if buff.ID == "" || strings.TrimSpace(buff.Key) == "" || strings.TrimSpace(buff.Mode) == "" {
			continue
		}
		result[buff.ID] = buff
	}
	return result
}

func deleteStalePlayerBuffsTx(tx *sql.Tx, playerID string, buffIDs []string) error {
	placeholders := strings.TrimRight(strings.Repeat("?,", len(buffIDs)), ",")
	args := make([]any, 0, len(buffIDs)+1)
	args = append(args, playerID)
	for _, buffID := range buffIDs {
		args = append(args, buffID)
	}
	_, err := tx.Exec(`DELETE FROM player_buffs WHERE player_id = ? AND buff_id NOT IN (`+placeholders+`)`, args...)
	return err
}
