// 本文件归口 MySQL 玩家建筑权威表和资源田格子权威表同步。
package storage

import (
	"database/sql"
	"errors"
	"sort"
	"strings"
	"time"

	"hero3/internal/app/game"
)

var errPlayerBuildingsMissing = errors.New("player_buildings rows missing; run backfill-buildings before using buildings as authoritative state")
var errPlayerResourceSlotsMissing = errors.New("player_resource_slots rows missing; run backfill-resource-slots before using resource slots as authoritative state")

// overlayAuthoritativeBuildings 用 player_buildings 权威表覆盖兼容快照中的建筑。
func (r *MySQLRepository) overlayAuthoritativeBuildings(state *game.GameState, playerID string) error {
	buildings, found, err := loadPlayerBuildings(r.db, playerID)
	if err != nil {
		return err
	}
	return applyAuthoritativeBuildings(state, buildings, found)
}

// overlayAuthoritativeBuildingsTx 在事务内锁定并加载玩家建筑权威表。
func overlayAuthoritativeBuildingsTx(tx *sql.Tx, state *game.GameState, playerID string) error {
	buildings, found, err := loadPlayerBuildingsTx(tx, playerID)
	if err != nil {
		return err
	}
	return applyAuthoritativeBuildings(state, buildings, found)
}

// overlayAuthoritativeResourceSlots 用 player_resource_slots 权威表覆盖兼容快照中的资源田格子。
func (r *MySQLRepository) overlayAuthoritativeResourceSlots(state *game.GameState, playerID string) error {
	slots, found, err := loadPlayerResourceSlots(r.db, playerID)
	if err != nil {
		return err
	}
	return applyAuthoritativeResourceSlots(state, slots, found)
}

// overlayAuthoritativeResourceSlotsTx 在事务内锁定并加载玩家资源田格子权威表。
func overlayAuthoritativeResourceSlotsTx(tx *sql.Tx, state *game.GameState, playerID string) error {
	slots, found, err := loadPlayerResourceSlotsTx(tx, playerID)
	if err != nil {
		return err
	}
	return applyAuthoritativeResourceSlots(state, slots, found)
}

// applyAuthoritativeBuildings 将建筑权威表结果写回 GameState；旧快照有建筑但表为空时显式报错。
func applyAuthoritativeBuildings(state *game.GameState, buildings []game.Building, found bool) error {
	if state == nil {
		return game.ErrPlayerNotFound
	}
	if !found {
		if len(state.Buildings) == 0 {
			state.Buildings = []game.Building{}
			return nil
		}
		return errPlayerBuildingsMissing
	}
	state.Buildings = buildings
	return nil
}

// applyAuthoritativeResourceSlots 将资源田格子权威表结果写回 GameState。
func applyAuthoritativeResourceSlots(state *game.GameState, slots []game.ResourceSlot, found bool) error {
	if state == nil {
		return game.ErrPlayerNotFound
	}
	if !found {
		if len(state.ResourceSlots) == 0 {
			derived := game.BuildResourceSlotsFromBuildings(state.Buildings, time.Now())
			if len(derived) == 0 {
				state.ResourceSlots = []game.ResourceSlot{}
				return nil
			}
		}
		return errPlayerResourceSlotsMissing
	}
	state.ResourceSlots = slots
	return nil
}

// loadPlayerBuildings 从 player_buildings 读取玩家建筑权威状态。
func loadPlayerBuildings(queryer resourceQueryer, playerID string) ([]game.Building, bool, error) {
	return loadPlayerBuildingsWithQuery(queryer, playerID, "")
}

// loadPlayerBuildingsTx 在事务内读取并锁定玩家建筑权威状态。
func loadPlayerBuildingsTx(tx *sql.Tx, playerID string) ([]game.Building, bool, error) {
	return loadPlayerBuildingsWithQuery(tx, playerID, " FOR UPDATE")
}

// loadPlayerBuildingsWithQuery 读取建筑表并还原建筑列表。
func loadPlayerBuildingsWithQuery(queryer resourceQueryer, playerID string, lockClause string) ([]game.Building, bool, error) {
	rows, err := queryer.Query(
		`SELECT building_id, building_type, level, status, upgrade_ends_at, status_ends_at
		 FROM player_buildings
		 WHERE player_id = ?
		 ORDER BY building_id`+lockClause,
		playerID,
	)
	if err != nil {
		return nil, false, err
	}
	defer rows.Close()

	buildings := []game.Building{}
	for rows.Next() {
		var building game.Building
		var upgradeEndsAt sql.NullTime
		var statusEndsAt sql.NullTime
		if err := rows.Scan(&building.ID, &building.Type, &building.Level, &building.Status, &upgradeEndsAt, &statusEndsAt); err != nil {
			return nil, false, err
		}
		building.ID = strings.TrimSpace(building.ID)
		building.Type = strings.TrimSpace(building.Type)
		if building.ID == "" || building.Type == "" {
			continue
		}
		if upgradeEndsAt.Valid {
			formatted := upgradeEndsAt.Time.UTC().Format(time.RFC3339)
			building.UpgradeEndsAt = &formatted
		}
		if statusEndsAt.Valid {
			formatted := statusEndsAt.Time.UTC().Format(time.RFC3339)
			building.StatusEndsAt = &formatted
		}
		buildings = append(buildings, building)
	}
	if err := rows.Err(); err != nil {
		return nil, false, err
	}
	return buildings, len(buildings) > 0, nil
}

// loadPlayerResourceSlots 从 player_resource_slots 读取玩家资源田格子权威状态。
func loadPlayerResourceSlots(queryer resourceQueryer, playerID string) ([]game.ResourceSlot, bool, error) {
	return loadPlayerResourceSlotsWithQuery(queryer, playerID, "")
}

// loadPlayerResourceSlotsTx 在事务内读取并锁定玩家资源田格子权威状态。
func loadPlayerResourceSlotsTx(tx *sql.Tx, playerID string) ([]game.ResourceSlot, bool, error) {
	return loadPlayerResourceSlotsWithQuery(tx, playerID, " FOR UPDATE")
}

// loadPlayerResourceSlotsWithQuery 读取资源田格子表并还原资源田格子列表。
func loadPlayerResourceSlotsWithQuery(queryer resourceQueryer, playerID string, lockClause string) ([]game.ResourceSlot, bool, error) {
	rows, err := queryer.Query(
		`SELECT slot_id, resource_type, building_id, unlocked_by, unlocked_at
		 FROM player_resource_slots
		 WHERE player_id = ?
		 ORDER BY slot_id`+lockClause,
		playerID,
	)
	if err != nil {
		return nil, false, err
	}
	defer rows.Close()

	slots := []game.ResourceSlot{}
	for rows.Next() {
		var slot game.ResourceSlot
		var unlockedAt sql.NullTime
		if err := rows.Scan(&slot.ID, &slot.ResourceType, &slot.BuildingID, &slot.UnlockedBy, &unlockedAt); err != nil {
			return nil, false, err
		}
		slot.ID = strings.TrimSpace(slot.ID)
		slot.ResourceType = strings.TrimSpace(slot.ResourceType)
		if slot.ID == "" || slot.ResourceType == "" {
			continue
		}
		if unlockedAt.Valid {
			slot.UnlockedAt = unlockedAt.Time.UTC().Format(time.RFC3339)
		}
		slots = append(slots, slot)
	}
	if err := rows.Err(); err != nil {
		return nil, false, err
	}
	return slots, len(slots) > 0, nil
}

// syncPlayerBuildingsTx 把事务内建筑快照同步到 player_buildings 权威表。
func syncPlayerBuildingsTx(tx *sql.Tx, playerID string, buildings []game.Building, updatedAt time.Time) error {
	buildingIDs := buildingIDsFromBuildings(buildings)
	if len(buildingIDs) == 0 {
		_, err := tx.Exec(`DELETE FROM player_buildings WHERE player_id = ?`, playerID)
		return err
	}
	byID := buildingsByID(buildings)
	for _, buildingID := range buildingIDs {
		building := byID[buildingID]
		if _, err := tx.Exec(
			`INSERT INTO player_buildings (player_id, building_id, building_type, level, status, upgrade_ends_at, status_ends_at, updated_at)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?)
			 ON DUPLICATE KEY UPDATE
				building_type = VALUES(building_type),
				level = VALUES(level),
				status = VALUES(status),
				upgrade_ends_at = VALUES(upgrade_ends_at),
				status_ends_at = VALUES(status_ends_at),
				updated_at = VALUES(updated_at)`,
			playerID,
			building.ID,
			building.Type,
			building.Level,
			building.Status,
			nullableTimeArg(parseStorageTimePtr(building.UpgradeEndsAt)),
			nullableTimeArg(parseStorageTimePtr(building.StatusEndsAt)),
			updatedAt.UTC(),
		); err != nil {
			return err
		}
	}
	return deleteStalePlayerBuildingsTx(tx, playerID, buildingIDs)
}

// syncPlayerResourceSlotsTx 把事务内资源田格子快照同步到 player_resource_slots 权威表。
func syncPlayerResourceSlotsTx(tx *sql.Tx, playerID string, slots []game.ResourceSlot, updatedAt time.Time) error {
	slotIDs := slotIDsFromResourceSlots(slots)
	if len(slotIDs) == 0 {
		_, err := tx.Exec(`DELETE FROM player_resource_slots WHERE player_id = ?`, playerID)
		return err
	}
	byID := resourceSlotsByID(slots)
	for _, slotID := range slotIDs {
		slot := byID[slotID]
		if _, err := tx.Exec(
			`INSERT INTO player_resource_slots (player_id, slot_id, resource_type, building_id, unlocked_by, unlocked_at, updated_at)
			 VALUES (?, ?, ?, ?, ?, ?, ?)
			 ON DUPLICATE KEY UPDATE
				resource_type = VALUES(resource_type),
				building_id = VALUES(building_id),
				unlocked_by = VALUES(unlocked_by),
				unlocked_at = VALUES(unlocked_at),
				updated_at = VALUES(updated_at)`,
			playerID,
			slot.ID,
			slot.ResourceType,
			slot.BuildingID,
			slot.UnlockedBy,
			nullableTimeArg(parseStorageTime(slot.UnlockedAt)),
			updatedAt.UTC(),
		); err != nil {
			return err
		}
	}
	return deleteStalePlayerResourceSlotsTx(tx, playerID, slotIDs)
}

// buildingSnapshotChanged 判断建筑状态是否发生变化。
func buildingSnapshotChanged(before map[string]storageBuildingSnapshot, after []game.Building) bool {
	return !buildingSnapshotMapsEqual(before, buildingSnapshotsFromStorageState(after))
}

// resourceSlotSnapshotChanged 判断资源田格子是否发生变化。
func resourceSlotSnapshotChanged(before map[string]storageResourceSlotSnapshot, after []game.ResourceSlot) bool {
	return !resourceSlotSnapshotMapsEqual(before, resourceSlotSnapshotsFromStorageState(after))
}

type storageBuildingSnapshot struct {
	Type          string
	Level         int
	Status        string
	UpgradeEndsAt string
	StatusEndsAt  string
}

type storageResourceSlotSnapshot struct {
	ResourceType string
	BuildingID   string
	UnlockedBy   string
	UnlockedAt   string
}

// buildingSnapshotsFromStorageState 从建筑列表生成同步比较快照。
func buildingSnapshotsFromStorageState(buildings []game.Building) map[string]storageBuildingSnapshot {
	snapshots := map[string]storageBuildingSnapshot{}
	for _, building := range buildings {
		building.ID = strings.TrimSpace(building.ID)
		building.Type = strings.TrimSpace(building.Type)
		if building.ID == "" || building.Type == "" {
			continue
		}
		snapshot := storageBuildingSnapshot{
			Type:   building.Type,
			Level:  building.Level,
			Status: building.Status,
		}
		if building.UpgradeEndsAt != nil {
			snapshot.UpgradeEndsAt = strings.TrimSpace(*building.UpgradeEndsAt)
		}
		if building.StatusEndsAt != nil {
			snapshot.StatusEndsAt = strings.TrimSpace(*building.StatusEndsAt)
		}
		snapshots[building.ID] = snapshot
	}
	return snapshots
}

// resourceSlotSnapshotsFromStorageState 从资源田格子列表生成同步比较快照。
func resourceSlotSnapshotsFromStorageState(slots []game.ResourceSlot) map[string]storageResourceSlotSnapshot {
	snapshots := map[string]storageResourceSlotSnapshot{}
	for _, slot := range slots {
		slot.ID = strings.TrimSpace(slot.ID)
		slot.ResourceType = strings.TrimSpace(slot.ResourceType)
		if slot.ID == "" || slot.ResourceType == "" {
			continue
		}
		snapshots[slot.ID] = storageResourceSlotSnapshot{
			ResourceType: slot.ResourceType,
			BuildingID:   strings.TrimSpace(slot.BuildingID),
			UnlockedBy:   strings.TrimSpace(slot.UnlockedBy),
			UnlockedAt:   strings.TrimSpace(slot.UnlockedAt),
		}
	}
	return snapshots
}

// buildingSnapshotMapsEqual 比较两个建筑快照集合是否一致。
func buildingSnapshotMapsEqual(a map[string]storageBuildingSnapshot, b map[string]storageBuildingSnapshot) bool {
	if len(a) != len(b) {
		return false
	}
	for buildingID, left := range a {
		if right, ok := b[buildingID]; !ok || left != right {
			return false
		}
	}
	return true
}

// resourceSlotSnapshotMapsEqual 比较两个资源田格子快照集合是否一致。
func resourceSlotSnapshotMapsEqual(a map[string]storageResourceSlotSnapshot, b map[string]storageResourceSlotSnapshot) bool {
	if len(a) != len(b) {
		return false
	}
	for slotID, left := range a {
		if right, ok := b[slotID]; !ok || left != right {
			return false
		}
	}
	return true
}

// buildingIDsFromBuildings 提取建筑 ID。
func buildingIDsFromBuildings(buildings []game.Building) []string {
	idSet := map[string]struct{}{}
	for _, building := range buildings {
		buildingID := strings.TrimSpace(building.ID)
		if buildingID == "" || strings.TrimSpace(building.Type) == "" {
			continue
		}
		idSet[buildingID] = struct{}{}
	}
	ids := make([]string, 0, len(idSet))
	for id := range idSet {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

// slotIDsFromResourceSlots 提取资源田格子 ID。
func slotIDsFromResourceSlots(slots []game.ResourceSlot) []string {
	idSet := map[string]struct{}{}
	for _, slot := range slots {
		slotID := strings.TrimSpace(slot.ID)
		if slotID == "" || strings.TrimSpace(slot.ResourceType) == "" {
			continue
		}
		idSet[slotID] = struct{}{}
	}
	ids := make([]string, 0, len(idSet))
	for id := range idSet {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

// buildingsByID 按 ID 索引建筑。
func buildingsByID(buildings []game.Building) map[string]game.Building {
	result := map[string]game.Building{}
	for _, building := range buildings {
		building.ID = strings.TrimSpace(building.ID)
		building.Type = strings.TrimSpace(building.Type)
		if building.ID == "" || building.Type == "" {
			continue
		}
		result[building.ID] = building
	}
	return result
}

// resourceSlotsByID 按 ID 索引资源田格子。
func resourceSlotsByID(slots []game.ResourceSlot) map[string]game.ResourceSlot {
	result := map[string]game.ResourceSlot{}
	for _, slot := range slots {
		slot.ID = strings.TrimSpace(slot.ID)
		slot.ResourceType = strings.TrimSpace(slot.ResourceType)
		if slot.ID == "" || slot.ResourceType == "" {
			continue
		}
		result[slot.ID] = slot
	}
	return result
}

// deleteStalePlayerBuildingsTx 删除兼容快照里已经不存在的建筑。
func deleteStalePlayerBuildingsTx(tx *sql.Tx, playerID string, buildingIDs []string) error {
	placeholders := strings.TrimRight(strings.Repeat("?,", len(buildingIDs)), ",")
	args := make([]any, 0, len(buildingIDs)+1)
	args = append(args, playerID)
	for _, buildingID := range buildingIDs {
		args = append(args, buildingID)
	}
	_, err := tx.Exec(
		`DELETE FROM player_buildings
		 WHERE player_id = ? AND building_id NOT IN (`+placeholders+`)`,
		args...,
	)
	return err
}

// deleteStalePlayerResourceSlotsTx 删除兼容快照里已经不存在的资源田格子。
func deleteStalePlayerResourceSlotsTx(tx *sql.Tx, playerID string, slotIDs []string) error {
	placeholders := strings.TrimRight(strings.Repeat("?,", len(slotIDs)), ",")
	args := make([]any, 0, len(slotIDs)+1)
	args = append(args, playerID)
	for _, slotID := range slotIDs {
		args = append(args, slotID)
	}
	_, err := tx.Exec(
		`DELETE FROM player_resource_slots
		 WHERE player_id = ? AND slot_id NOT IN (`+placeholders+`)`,
		args...,
	)
	return err
}

// parseStorageTimePtr 解析可空时间字符串。
func parseStorageTimePtr(value *string) sql.NullTime {
	if value == nil {
		return sql.NullTime{}
	}
	return parseStorageTime(*value)
}

// parseStorageTime 解析时间字符串为可空数据库时间。
func parseStorageTime(value string) sql.NullTime {
	value = strings.TrimSpace(value)
	if value == "" {
		return sql.NullTime{}
	}
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return sql.NullTime{}
	}
	return sql.NullTime{Time: parsed.UTC(), Valid: true}
}
