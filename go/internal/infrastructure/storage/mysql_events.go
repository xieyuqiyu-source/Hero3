// 本文件归口 MySQL 事件处理幂等记录。
package storage

import (
	"strings"
	"time"
)

// ClaimEventProcessing 尝试声明模块事件处理权，重复处理返回 false。
func (r *MySQLRepository) ClaimEventProcessing(moduleID string, handlerKey string, eventKey string, processedAt time.Time) (bool, error) {
	moduleID = strings.TrimSpace(moduleID)
	handlerKey = strings.TrimSpace(handlerKey)
	eventKey = strings.TrimSpace(eventKey)
	if moduleID == "" || handlerKey == "" || eventKey == "" {
		return false, nil
	}
	result, err := r.db.Exec(
		`INSERT IGNORE INTO event_processing_records (module_id, handler_key, event_key, processed_at)
		 VALUES (?, ?, ?, ?)`,
		moduleID,
		handlerKey,
		eventKey,
		processedAt.UTC(),
	)
	if err != nil {
		return false, err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return false, err
	}
	return affected > 0, nil
}
