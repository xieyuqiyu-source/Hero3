// 本文件归口战斗所需的 MySQL 资产级事务，避免战斗服务直接走完整玩家状态事务。
package storage

import (
	"time"

	"hero3/internal/app/game"
)

// UpdateCombatState 复用战斗所需的资产事务范围，覆盖资源结算、兵力、武将、Buff 和 NPC 轻量快照。
func (r *MySQLRepository) UpdateCombatState(playerID string, updatedAt time.Time, update func(state *game.GameState) error) (game.GameState, error) {
	return r.UpdateItemState(playerID, updatedAt, update)
}
