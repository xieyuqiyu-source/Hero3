// 本文件实现 MySQL 世界地图玩家权威坐标仓储。
package storage

import (
	"database/sql"
	"errors"
	"time"

	"hero3/internal/app/game"
)

// GetWorldPosition 读取玩家权威世界坐标。
func (r *MySQLRepository) GetWorldPosition(playerID string) (game.WorldPosition, error) {
	var position game.WorldPosition
	var createdAt time.Time
	var updatedAt time.Time
	err := r.db.QueryRow(
		`SELECT player_id, world_id, x, y, assigned_by, created_at, updated_at
		 FROM player_world_positions
		 WHERE player_id = ?`,
		playerID,
	).Scan(&position.PlayerID, &position.WorldID, &position.X, &position.Y, &position.AssignedBy, &createdAt, &updatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return game.WorldPosition{}, game.ErrPlayerNotFound
	}
	if err != nil {
		return game.WorldPosition{}, err
	}
	position.CreatedAt = createdAt.UTC().Format(time.RFC3339)
	position.UpdatedAt = updatedAt.UTC().Format(time.RFC3339)
	return position, nil
}

// EnsureWorldPosition 确保玩家拥有唯一世界坐标。
func (r *MySQLRepository) EnsureWorldPosition(playerID string, assignedBy string, preferred *game.WorldCoordinate) (game.WorldPosition, error) {
	if _, err := r.GetAccountIDByPlayerID(playerID); err != nil {
		return game.WorldPosition{}, err
	}
	if position, err := r.GetWorldPosition(playerID); err == nil {
		return position, nil
	} else if !errors.Is(err, game.ErrPlayerNotFound) {
		return game.WorldPosition{}, err
	}
	start := worldMapPreferredCoordinateForStorage(playerID, preferred)
	for _, coordinate := range game.WorldMapCoordinateCandidates(start.X, start.Y) {
		position, err := r.AssignWorldPosition(playerID, "world_1", coordinate.X, coordinate.Y, assignedBy)
		if err == nil {
			return position, nil
		}
		if !errors.Is(err, game.ErrInvalidWorldCoordinate) {
			return game.WorldPosition{}, err
		}
	}
	return game.WorldPosition{}, game.ErrWorldMapFull
}

// ListWorldPositions 查询指定范围内的玩家权威坐标。
func (r *MySQLRepository) ListWorldPositions(worldID string, minX int, maxX int, minY int, maxY int) ([]game.WorldPosition, error) {
	if worldID == "" {
		worldID = "world_1"
	}
	rows, err := r.db.Query(
		`SELECT player_id, world_id, x, y, assigned_by, created_at, updated_at
		 FROM player_world_positions
		 WHERE world_id = ? AND x BETWEEN ? AND ? AND y BETWEEN ? AND ?
		 ORDER BY y ASC, x ASC`,
		worldID, minX, maxX, minY, maxY,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	positions := []game.WorldPosition{}
	for rows.Next() {
		var position game.WorldPosition
		var createdAt time.Time
		var updatedAt time.Time
		if err := rows.Scan(&position.PlayerID, &position.WorldID, &position.X, &position.Y, &position.AssignedBy, &createdAt, &updatedAt); err != nil {
			return nil, err
		}
		position.CreatedAt = createdAt.UTC().Format(time.RFC3339)
		position.UpdatedAt = updatedAt.UTC().Format(time.RFC3339)
		positions = append(positions, position)
	}
	return positions, rows.Err()
}

// CountWorldPositions 统计指定世界已占用坐标数量。
func (r *MySQLRepository) CountWorldPositions(worldID string) (int, error) {
	if worldID == "" {
		worldID = "world_1"
	}
	var count int
	if err := r.db.QueryRow(`SELECT COUNT(*) FROM player_world_positions WHERE world_id = ?`, worldID).Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}

// AssignWorldPosition 手动设置玩家权威世界坐标。
func (r *MySQLRepository) AssignWorldPosition(playerID string, worldID string, x int, y int, assignedBy string) (game.WorldPosition, error) {
	if _, err := r.GetAccountIDByPlayerID(playerID); err != nil {
		return game.WorldPosition{}, err
	}
	if worldID == "" {
		worldID = "world_1"
	}
	if !worldCoordinateInBoundsForStorage(x, y) {
		return game.WorldPosition{}, game.ErrInvalidWorldCoordinate
	}
	now := time.Now().UTC()
	if _, err := r.GetWorldPosition(playerID); err == nil {
		_, err = r.db.Exec(
			`UPDATE player_world_positions
			 SET world_id = ?, x = ?, y = ?, assigned_by = ?, updated_at = ?
			 WHERE player_id = ?`,
			worldID, x, y, assignedBy, now, playerID,
		)
		if err != nil {
			return game.WorldPosition{}, game.ErrInvalidWorldCoordinate
		}
		return r.GetWorldPosition(playerID)
	} else if !errors.Is(err, game.ErrPlayerNotFound) {
		return game.WorldPosition{}, err
	}
	if _, err := r.db.Exec(
		`INSERT INTO player_world_positions (player_id, world_id, x, y, assigned_by, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		playerID, worldID, x, y, assignedBy, now, now,
	); err != nil {
		return game.WorldPosition{}, game.ErrInvalidWorldCoordinate
	}
	return r.GetWorldPosition(playerID)
}

// gameWorldWidth 返回第一版世界地图宽度。
func gameWorldWidth() int {
	return 100
}

// gameWorldHeight 返回第一版世界地图高度。
func gameWorldHeight() int {
	return 100
}

// worldMapPreferredCoordinateForStorage 选择 MySQL 仓储写入时的优先坐标。
func worldMapPreferredCoordinateForStorage(playerID string, preferred *game.WorldCoordinate) game.WorldCoordinate {
	if preferred != nil && worldCoordinateInBoundsForStorage(preferred.X, preferred.Y) {
		return *preferred
	}
	return game.LegacyWorldCoordinateForPlayer(playerID)
}

// worldCoordinateInBoundsForStorage 判断 MySQL 仓储坐标是否在第一版世界范围内。
func worldCoordinateInBoundsForStorage(x int, y int) bool {
	return x >= 0 && x < gameWorldWidth() && y >= 0 && y < gameWorldHeight()
}
