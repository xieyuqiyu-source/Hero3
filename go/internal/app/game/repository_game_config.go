// 本文件实现内存仓储中的线上 GM 配置读写能力。
package game

import (
	"strings"
	"time"
)

// GetGameConfig 读取一份内存 GM 配置。
func (r *MemoryRepository) GetGameConfig(key string) (GameConfigRecord, bool, error) {
	key = strings.TrimSpace(key)
	r.mu.RLock()
	defer r.mu.RUnlock()
	record, exists := r.gameConfigs[key]
	if !exists {
		return GameConfigRecord{}, false, nil
	}
	record.ValueJSON = append([]byte(nil), record.ValueJSON...)
	return record, true, nil
}

// SaveGameConfig 写入一份内存 GM 配置，并递增版本号。
func (r *MemoryRepository) SaveGameConfig(key string, valueJSON []byte, updatedBy string, updatedAt time.Time) (GameConfigRecord, error) {
	key = strings.TrimSpace(key)
	updatedBy = strings.TrimSpace(updatedBy)
	if updatedAt.IsZero() {
		updatedAt = time.Now().UTC()
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	record := r.gameConfigs[key]
	if record.Key == "" {
		record.Key = key
		record.Version = 1
		record.CreatedAt = updatedAt
	} else {
		record.Version++
	}
	record.ValueJSON = append([]byte(nil), valueJSON...)
	record.UpdatedBy = updatedBy
	record.UpdatedAt = updatedAt
	r.gameConfigs[key] = record

	record.ValueJSON = append([]byte(nil), record.ValueJSON...)
	return record, nil
}
