// 本文件实现内存仓储中的世界地图权威坐标能力。
package game

import (
	"sort"
	"time"
)

// GetWorldPosition 读取玩家权威世界坐标。
func (r *MemoryRepository) GetWorldPosition(playerID string) (WorldPosition, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	position, ok := r.worldPositions[playerID]
	if !ok {
		return WorldPosition{}, ErrPlayerNotFound
	}
	return position, nil
}

// EnsureWorldPosition 确保玩家拥有唯一世界坐标。
func (r *MemoryRepository) EnsureWorldPosition(playerID string, assignedBy string, preferred *WorldCoordinate) (WorldPosition, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if position, ok := r.worldPositions[playerID]; ok {
		return position, nil
	}
	if _, ok := r.players[playerID]; !ok {
		return WorldPosition{}, ErrPlayerNotFound
	}
	start := worldMapPreferredCoordinate(playerID, preferred)
	x, y, ok := findAvailableWorldCoordinateLocked(r.worldPositions, defaultWorldID, start.X, start.Y)
	if !ok {
		return WorldPosition{}, ErrWorldMapFull
	}
	now := time.Now().UTC().Format(resourceDateLayout)
	position := WorldPosition{
		PlayerID:   playerID,
		WorldID:    defaultWorldID,
		X:          x,
		Y:          y,
		AssignedBy: assignedBy,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	r.worldPositions[playerID] = position
	return position, nil
}

// ListWorldPositions 查询指定范围内的玩家权威坐标。
func (r *MemoryRepository) ListWorldPositions(worldID string, minX int, maxX int, minY int, maxY int) ([]WorldPosition, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if worldID == "" {
		worldID = defaultWorldID
	}
	items := []WorldPosition{}
	for _, position := range r.worldPositions {
		if position.WorldID != worldID {
			continue
		}
		if position.X < minX || position.X > maxX || position.Y < minY || position.Y > maxY {
			continue
		}
		items = append(items, position)
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].Y == items[j].Y {
			return items[i].X < items[j].X
		}
		return items[i].Y < items[j].Y
	})
	return items, nil
}

// ListWorldMapPlayerCities 只投影当前地图范围内的玩家城池，避免读取全服账号列表。
func (r *MemoryRepository) ListWorldMapPlayerCities(worldID string, minX int, maxX int, minY int, maxY int) ([]WorldMapPlayerCity, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if worldID == "" {
		worldID = defaultWorldID
	}
	accountByPlayer := make(map[string]string, len(r.players))
	for accountID, playerIDs := range r.accountPlayers {
		for _, playerID := range playerIDs {
			accountByPlayer[playerID] = accountID
		}
	}
	items := []WorldMapPlayerCity{}
	for playerID, position := range r.worldPositions {
		if position.WorldID != worldID || position.X < minX || position.X > maxX || position.Y < minY || position.Y > maxY {
			continue
		}
		state, exists := r.players[playerID]
		if !exists {
			continue
		}
		summary := buildPlayerSummary(state, r.playerUpdatedAt[playerID])
		items = append(items, WorldMapPlayerCity{
			Position:      position,
			AccountID:     accountByPlayer[playerID],
			Name:          summary.Nickname,
			Faction:       summary.Faction,
			BuildingLevel: summary.BuildingLevel,
		})
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].Position.Y == items[j].Position.Y {
			return items[i].Position.X < items[j].Position.X
		}
		return items[i].Position.Y < items[j].Position.Y
	})
	return items, nil
}

// CountWorldPositions 统计指定世界已占用坐标数量。
func (r *MemoryRepository) CountWorldPositions(worldID string) (int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if worldID == "" {
		worldID = defaultWorldID
	}
	count := 0
	for _, position := range r.worldPositions {
		if position.WorldID == worldID {
			count++
		}
	}
	return count, nil
}

// AssignWorldPosition 手动设置玩家权威世界坐标。
func (r *MemoryRepository) AssignWorldPosition(playerID string, worldID string, x int, y int, assignedBy string) (WorldPosition, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.players[playerID]; !ok {
		return WorldPosition{}, ErrPlayerNotFound
	}
	if worldID == "" {
		worldID = defaultWorldID
	}
	if !worldCoordinateInBounds(x, y) {
		return WorldPosition{}, ErrInvalidWorldCoordinate
	}
	for existingPlayerID, position := range r.worldPositions {
		if existingPlayerID != playerID && position.WorldID == worldID && position.X == x && position.Y == y {
			return WorldPosition{}, ErrInvalidWorldCoordinate
		}
	}
	now := time.Now().UTC().Format(resourceDateLayout)
	position := WorldPosition{PlayerID: playerID, WorldID: worldID, X: x, Y: y, AssignedBy: assignedBy, CreatedAt: now, UpdatedAt: now}
	if existing, ok := r.worldPositions[playerID]; ok {
		position.CreatedAt = existing.CreatedAt
	}
	r.worldPositions[playerID] = position
	return position, nil
}
