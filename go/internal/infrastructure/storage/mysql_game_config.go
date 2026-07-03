// 本文件实现线上 GM 配置在 MySQL 中的持久化读写。
package storage

import (
	"database/sql"
	"strings"
	"time"

	"hero3/internal/app/game"
)

// GetGameConfig 读取数据库中的 GM 配置。
func (r *MySQLRepository) GetGameConfig(key string) (game.GameConfigRecord, bool, error) {
	key = strings.TrimSpace(key)
	var record game.GameConfigRecord
	var createdAt time.Time
	var updatedAt time.Time
	err := r.db.QueryRow(`
		SELECT config_key, value_json, version, updated_by, created_at, updated_at
		FROM game_configs
		WHERE config_key = ?
	`, key).Scan(&record.Key, &record.ValueJSON, &record.Version, &record.UpdatedBy, &createdAt, &updatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return game.GameConfigRecord{}, false, nil
		}
		return game.GameConfigRecord{}, false, err
	}
	record.CreatedAt = createdAt.UTC()
	record.UpdatedAt = updatedAt.UTC()
	record.ValueJSON = append([]byte(nil), record.ValueJSON...)
	return record, true, nil
}

// SaveGameConfig 写入数据库 GM 配置，并在覆盖时递增版本。
func (r *MySQLRepository) SaveGameConfig(key string, valueJSON []byte, updatedBy string, updatedAt time.Time) (game.GameConfigRecord, error) {
	key = strings.TrimSpace(key)
	updatedBy = strings.TrimSpace(updatedBy)
	if updatedAt.IsZero() {
		updatedAt = time.Now().UTC()
	}
	_, err := r.db.Exec(`
		INSERT INTO game_configs (config_key, value_json, version, updated_by, created_at, updated_at)
		VALUES (?, ?, 1, ?, ?, ?)
		ON DUPLICATE KEY UPDATE
			value_json = VALUES(value_json),
			version = version + 1,
			updated_by = VALUES(updated_by),
			updated_at = VALUES(updated_at)
	`, key, string(valueJSON), updatedBy, updatedAt.UTC(), updatedAt.UTC())
	if err != nil {
		return game.GameConfigRecord{}, err
	}
	record, _, err := r.GetGameConfig(key)
	return record, err
}
