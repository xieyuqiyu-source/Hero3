// 本文件归口 MySQL 玩家兵力权威表和征兵队列权威表同步。
package storage

import (
	"database/sql"
	"errors"
	"sort"
	"strings"
	"time"

	"hero3/internal/app/game"
)

var errPlayerArmyMissing = errors.New("player_army_units rows missing; run backfill-army before using army as authoritative state")
var errPlayerRecruitQueuesMissing = errors.New("player_recruit_queues rows missing; run backfill-recruit-queues before using recruit queues as authoritative state")

// overlayAuthoritativeArmy 用 player_army_units 权威表覆盖兼容快照中的兵力。
func (r *MySQLRepository) overlayAuthoritativeArmy(state *game.GameState, playerID string) error {
	army, found, err := loadPlayerArmy(r.db, playerID)
	if err != nil {
		return err
	}
	return applyAuthoritativeArmy(state, army, found)
}

// overlayAuthoritativeArmyTx 在事务内锁定并加载玩家兵力权威表。
func overlayAuthoritativeArmyTx(tx *sql.Tx, state *game.GameState, playerID string) error {
	army, found, err := loadPlayerArmyTx(tx, playerID)
	if err != nil {
		return err
	}
	return applyAuthoritativeArmy(state, army, found)
}

// overlayAuthoritativeRecruitQueues 用 player_recruit_queues 权威表覆盖兼容快照中的征兵队列。
func (r *MySQLRepository) overlayAuthoritativeRecruitQueues(state *game.GameState, playerID string) error {
	queues, found, err := loadPlayerRecruitQueues(r.db, playerID)
	if err != nil {
		return err
	}
	return applyAuthoritativeRecruitQueues(state, queues, found)
}

// overlayAuthoritativeRecruitQueuesTx 在事务内锁定并加载玩家征兵队列权威表。
func overlayAuthoritativeRecruitQueuesTx(tx *sql.Tx, state *game.GameState, playerID string) error {
	queues, found, err := loadPlayerRecruitQueuesTx(tx, playerID)
	if err != nil {
		return err
	}
	return applyAuthoritativeRecruitQueues(state, queues, found)
}

// applyAuthoritativeArmy 将兵力权威表结果写回 GameState；旧快照有兵但表为空时显式报错。
func applyAuthoritativeArmy(state *game.GameState, army []game.ArmyUnit, found bool) error {
	if state == nil {
		return game.ErrPlayerNotFound
	}
	if !found {
		if len(armyUnitsFromState(state.Army)) == 0 {
			state.Army = []game.ArmyUnit{}
			return nil
		}
		return errPlayerArmyMissing
	}
	state.Army = army
	return nil
}

// applyAuthoritativeRecruitQueues 将征兵队列权威表结果写回 GameState；旧快照有队列但表为空时显式报错。
func applyAuthoritativeRecruitQueues(state *game.GameState, queues []game.RecruitQueue, found bool) error {
	if state == nil {
		return game.ErrPlayerNotFound
	}
	if !found {
		if len(recruitQueueIDsFromState(state.RecruitQueues)) == 0 {
			state.RecruitQueues = []game.RecruitQueue{}
			return nil
		}
		return errPlayerRecruitQueuesMissing
	}
	state.RecruitQueues = queues
	return nil
}

// loadPlayerArmy 从 player_army_units 读取玩家兵力权威状态。
func loadPlayerArmy(queryer resourceQueryer, playerID string) ([]game.ArmyUnit, bool, error) {
	return loadPlayerArmyWithQuery(queryer, playerID, "")
}

// loadPlayerArmyTx 在事务内读取并锁定玩家兵力权威状态。
func loadPlayerArmyTx(tx *sql.Tx, playerID string) ([]game.ArmyUnit, bool, error) {
	return loadPlayerArmyWithQuery(tx, playerID, " FOR UPDATE")
}

// loadPlayerArmyWithQuery 读取兵力表并还原兵力列表。
func loadPlayerArmyWithQuery(queryer resourceQueryer, playerID string, lockClause string) ([]game.ArmyUnit, bool, error) {
	rows, err := queryer.Query(
		`SELECT unit_type, amount
		 FROM player_army_units
		 WHERE player_id = ?
		 ORDER BY unit_type`+lockClause,
		playerID,
	)
	if err != nil {
		return nil, false, err
	}
	defer rows.Close()

	army := []game.ArmyUnit{}
	for rows.Next() {
		var unit game.ArmyUnit
		if err := rows.Scan(&unit.UnitType, &unit.Amount); err != nil {
			return nil, false, err
		}
		unit.UnitType = strings.TrimSpace(unit.UnitType)
		if unit.UnitType == "" || unit.Amount <= 0 {
			continue
		}
		army = append(army, unit)
	}
	if err := rows.Err(); err != nil {
		return nil, false, err
	}
	return army, len(army) > 0, nil
}

// loadPlayerArmyUnitRowsTx 在事务内只锁定指定兵种行。
func loadPlayerArmyUnitRowsTx(tx *sql.Tx, playerID string, unitTypes []string) ([]game.ArmyUnit, error) {
	unitTypes = normalizeArmyUnitTypes(unitTypes)
	if len(unitTypes) == 0 {
		return []game.ArmyUnit{}, nil
	}
	placeholders := strings.TrimRight(strings.Repeat("?,", len(unitTypes)), ",")
	args := make([]any, 0, len(unitTypes)+1)
	args = append(args, playerID)
	for _, unitType := range unitTypes {
		args = append(args, unitType)
	}
	rows, err := tx.Query(
		`SELECT unit_type, amount
		 FROM player_army_units
		 WHERE player_id = ? AND unit_type IN (`+placeholders+`)
		 ORDER BY unit_type
		 FOR UPDATE`,
		args...,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	army := []game.ArmyUnit{}
	for rows.Next() {
		var unit game.ArmyUnit
		if err := rows.Scan(&unit.UnitType, &unit.Amount); err != nil {
			return nil, err
		}
		unit.UnitType = strings.TrimSpace(unit.UnitType)
		if unit.UnitType == "" || unit.Amount <= 0 {
			continue
		}
		army = append(army, unit)
	}
	return army, rows.Err()
}

// loadPlayerRecruitQueues 从 player_recruit_queues 读取玩家征兵队列权威状态。
func loadPlayerRecruitQueues(queryer resourceQueryer, playerID string) ([]game.RecruitQueue, bool, error) {
	return loadPlayerRecruitQueuesWithQuery(queryer, playerID, "")
}

// loadPlayerRecruitQueuesTx 在事务内读取并锁定玩家征兵队列权威状态。
func loadPlayerRecruitQueuesTx(tx *sql.Tx, playerID string) ([]game.RecruitQueue, bool, error) {
	return loadPlayerRecruitQueuesWithQuery(tx, playerID, " FOR UPDATE")
}

// loadPlayerRecruitQueuesWithQuery 读取征兵队列表并还原队列列表。
func loadPlayerRecruitQueuesWithQuery(queryer resourceQueryer, playerID string, lockClause string) ([]game.RecruitQueue, bool, error) {
	rows, err := queryer.Query(
		`SELECT queue_id, unit_type, amount, ends_at
		 FROM player_recruit_queues
		 WHERE player_id = ?
		 ORDER BY queue_order, ends_at, queue_id`+lockClause,
		playerID,
	)
	if err != nil {
		return nil, false, err
	}
	defer rows.Close()

	queues := []game.RecruitQueue{}
	for rows.Next() {
		var queue game.RecruitQueue
		var endsAt time.Time
		if err := rows.Scan(&queue.ID, &queue.UnitType, &queue.Amount, &endsAt); err != nil {
			return nil, false, err
		}
		queue.ID = strings.TrimSpace(queue.ID)
		queue.UnitType = strings.TrimSpace(queue.UnitType)
		if queue.ID == "" || queue.UnitType == "" || queue.Amount <= 0 {
			continue
		}
		queue.EndsAt = endsAt.UTC().Format(time.RFC3339)
		queues = append(queues, queue)
	}
	if err := rows.Err(); err != nil {
		return nil, false, err
	}
	return queues, len(queues) > 0, nil
}

// syncPlayerArmyTx 把事务内兵力快照同步到 player_army_units 权威表。
func syncPlayerArmyTx(tx *sql.Tx, playerID string, army []game.ArmyUnit, updatedAt time.Time) error {
	unitTypes := armyUnitsFromState(army)
	if len(unitTypes) == 0 {
		_, err := tx.Exec(`DELETE FROM player_army_units WHERE player_id = ?`, playerID)
		return err
	}
	byType := armyByUnitType(army)
	for _, unitType := range unitTypes {
		if _, err := tx.Exec(
			`INSERT INTO player_army_units (player_id, unit_type, amount, updated_at)
			 VALUES (?, ?, ?, ?)
			 ON DUPLICATE KEY UPDATE
				amount = VALUES(amount),
				updated_at = VALUES(updated_at)`,
			playerID,
			unitType,
			byType[unitType].Amount,
			updatedAt.UTC(),
		); err != nil {
			return err
		}
	}
	return deleteStalePlayerArmyTx(tx, playerID, unitTypes)
}

// syncPlayerArmyDeltaTx 只写入变化后的兵种，并逐个删除消失兵种，避免高频战斗事务执行玩家级全量 DELETE。
func syncPlayerArmyDeltaTx(tx *sql.Tx, playerID string, before map[string]storageArmySnapshot, army []game.ArmyUnit, updatedAt time.Time) error {
	if before == nil {
		return syncPlayerArmyTx(tx, playerID, army, updatedAt)
	}
	next := armySnapshotsFromStorageState(army)
	nextTypes := make([]string, 0, len(next))
	for unitType := range next {
		nextTypes = append(nextTypes, unitType)
	}
	sort.Strings(nextTypes)
	byType := armyByUnitType(army)
	for _, unitType := range nextTypes {
		nextSnapshot := next[unitType]
		if beforeSnapshot, exists := before[unitType]; exists && beforeSnapshot == nextSnapshot {
			continue
		}
		unit := byType[unitType]
		if _, err := tx.Exec(
			`INSERT INTO player_army_units (player_id, unit_type, amount, updated_at)
			 VALUES (?, ?, ?, ?)
			 ON DUPLICATE KEY UPDATE
				amount = VALUES(amount),
				updated_at = VALUES(updated_at)`,
			playerID,
			unitType,
			unit.Amount,
			updatedAt.UTC(),
		); err != nil {
			return err
		}
	}
	for unitType := range before {
		if _, exists := next[unitType]; exists {
			continue
		}
		if _, err := tx.Exec(
			`DELETE FROM player_army_units WHERE player_id = ? AND unit_type = ?`,
			playerID,
			unitType,
		); err != nil {
			return err
		}
	}
	return nil
}

// syncPlayerRecruitQueuesTx 把事务内征兵队列快照同步到 player_recruit_queues 权威表。
func syncPlayerRecruitQueuesTx(tx *sql.Tx, playerID string, queues []game.RecruitQueue, updatedAt time.Time) error {
	queueIDs := recruitQueueIDsFromState(queues)
	if len(queueIDs) == 0 {
		_, err := tx.Exec(`DELETE FROM player_recruit_queues WHERE player_id = ?`, playerID)
		return err
	}
	byID := recruitQueuesByID(queues)
	orderByID := recruitQueueOrderByID(queues)
	for _, queueID := range queueIDs {
		queue := byID[queueID]
		endsAt := parseStorageTime(queue.EndsAt)
		if !endsAt.Valid {
			continue
		}
		if _, err := tx.Exec(
			`INSERT INTO player_recruit_queues (player_id, queue_id, unit_type, amount, ends_at, queue_order, updated_at)
			 VALUES (?, ?, ?, ?, ?, ?, ?)
			 ON DUPLICATE KEY UPDATE
				unit_type = VALUES(unit_type),
				amount = VALUES(amount),
				ends_at = VALUES(ends_at),
				queue_order = VALUES(queue_order),
				updated_at = VALUES(updated_at)`,
			playerID,
			queue.ID,
			queue.UnitType,
			queue.Amount,
			nullableTimeArg(endsAt),
			orderByID[queueID],
			updatedAt.UTC(),
		); err != nil {
			return err
		}
	}
	return deleteStalePlayerRecruitQueuesTx(tx, playerID, queueIDs)
}

// armySnapshotChanged 判断兵力是否发生变化。
func armySnapshotChanged(before map[string]storageArmySnapshot, after []game.ArmyUnit) bool {
	return !armySnapshotMapsEqual(before, armySnapshotsFromStorageState(after))
}

// recruitQueueSnapshotChanged 判断征兵队列是否发生变化。
func recruitQueueSnapshotChanged(before map[string]storageRecruitQueueSnapshot, after []game.RecruitQueue) bool {
	return !recruitQueueSnapshotMapsEqual(before, recruitQueueSnapshotsFromStorageState(after))
}

type storageArmySnapshot struct {
	Amount int
}

type storageRecruitQueueSnapshot struct {
	UnitType string
	Amount   int
	EndsAt   string
	Order    int
}

// armySnapshotsFromStorageState 从兵力列表生成同步比较快照。
func armySnapshotsFromStorageState(army []game.ArmyUnit) map[string]storageArmySnapshot {
	snapshots := map[string]storageArmySnapshot{}
	for _, unit := range army {
		unit.UnitType = strings.TrimSpace(unit.UnitType)
		if unit.UnitType == "" || unit.Amount <= 0 {
			continue
		}
		snapshots[unit.UnitType] = storageArmySnapshot{Amount: unit.Amount}
	}
	return snapshots
}

// recruitQueueSnapshotsFromStorageState 从征兵队列生成同步比较快照。
func recruitQueueSnapshotsFromStorageState(queues []game.RecruitQueue) map[string]storageRecruitQueueSnapshot {
	snapshots := map[string]storageRecruitQueueSnapshot{}
	for index, queue := range queues {
		queue.ID = strings.TrimSpace(queue.ID)
		queue.UnitType = strings.TrimSpace(queue.UnitType)
		if queue.ID == "" || queue.UnitType == "" || queue.Amount <= 0 {
			continue
		}
		snapshots[queue.ID] = storageRecruitQueueSnapshot{
			UnitType: queue.UnitType,
			Amount:   queue.Amount,
			EndsAt:   strings.TrimSpace(queue.EndsAt),
			Order:    index,
		}
	}
	return snapshots
}

// armySnapshotMapsEqual 比较两个兵力快照集合是否一致。
func armySnapshotMapsEqual(a map[string]storageArmySnapshot, b map[string]storageArmySnapshot) bool {
	if len(a) != len(b) {
		return false
	}
	for unitType, left := range a {
		if right, ok := b[unitType]; !ok || left != right {
			return false
		}
	}
	return true
}

// recruitQueueSnapshotMapsEqual 比较两个征兵队列快照集合是否一致。
func recruitQueueSnapshotMapsEqual(a map[string]storageRecruitQueueSnapshot, b map[string]storageRecruitQueueSnapshot) bool {
	if len(a) != len(b) {
		return false
	}
	for queueID, left := range a {
		if right, ok := b[queueID]; !ok || left != right {
			return false
		}
	}
	return true
}

// armyUnitsFromState 提取有效兵种 ID。
func armyUnitsFromState(army []game.ArmyUnit) []string {
	unitSet := map[string]struct{}{}
	for _, unit := range army {
		unitType := strings.TrimSpace(unit.UnitType)
		if unitType == "" || unit.Amount <= 0 {
			continue
		}
		unitSet[unitType] = struct{}{}
	}
	unitTypes := make([]string, 0, len(unitSet))
	for unitType := range unitSet {
		unitTypes = append(unitTypes, unitType)
	}
	sort.Strings(unitTypes)
	return unitTypes
}

// normalizeArmyUnitTypes 清理兵种 ID 列表。
func normalizeArmyUnitTypes(unitTypes []string) []string {
	unitSet := map[string]struct{}{}
	for _, unitType := range unitTypes {
		unitType = strings.TrimSpace(unitType)
		if unitType == "" {
			continue
		}
		unitSet[unitType] = struct{}{}
	}
	result := make([]string, 0, len(unitSet))
	for unitType := range unitSet {
		result = append(result, unitType)
	}
	sort.Strings(result)
	return result
}

// recruitQueueIDsFromState 提取有效征兵队列 ID。
func recruitQueueIDsFromState(queues []game.RecruitQueue) []string {
	idSet := map[string]struct{}{}
	for _, queue := range queues {
		queueID := strings.TrimSpace(queue.ID)
		if queueID == "" || strings.TrimSpace(queue.UnitType) == "" || queue.Amount <= 0 {
			continue
		}
		idSet[queueID] = struct{}{}
	}
	ids := make([]string, 0, len(idSet))
	for id := range idSet {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

// armyByUnitType 按兵种 ID 索引兵力。
func armyByUnitType(army []game.ArmyUnit) map[string]game.ArmyUnit {
	result := map[string]game.ArmyUnit{}
	for _, unit := range army {
		unit.UnitType = strings.TrimSpace(unit.UnitType)
		if unit.UnitType == "" || unit.Amount <= 0 {
			continue
		}
		result[unit.UnitType] = unit
	}
	return result
}

// recruitQueuesByID 按队列 ID 索引征兵队列。
func recruitQueuesByID(queues []game.RecruitQueue) map[string]game.RecruitQueue {
	result := map[string]game.RecruitQueue{}
	for _, queue := range queues {
		queue.ID = strings.TrimSpace(queue.ID)
		queue.UnitType = strings.TrimSpace(queue.UnitType)
		if queue.ID == "" || queue.UnitType == "" || queue.Amount <= 0 {
			continue
		}
		result[queue.ID] = queue
	}
	return result
}

// recruitQueueOrderByID 保留队列原始顺序。
func recruitQueueOrderByID(queues []game.RecruitQueue) map[string]int {
	result := map[string]int{}
	for index, queue := range queues {
		queue.ID = strings.TrimSpace(queue.ID)
		if queue.ID == "" {
			continue
		}
		result[queue.ID] = index
	}
	return result
}

// deleteStalePlayerArmyTx 删除兼容快照里已经不存在的兵种。
func deleteStalePlayerArmyTx(tx *sql.Tx, playerID string, unitTypes []string) error {
	placeholders := strings.TrimRight(strings.Repeat("?,", len(unitTypes)), ",")
	args := make([]any, 0, len(unitTypes)+1)
	args = append(args, playerID)
	for _, unitType := range unitTypes {
		args = append(args, unitType)
	}
	_, err := tx.Exec(
		`DELETE FROM player_army_units
		 WHERE player_id = ? AND unit_type NOT IN (`+placeholders+`)`,
		args...,
	)
	return err
}

// deleteStalePlayerRecruitQueuesTx 删除兼容快照里已经不存在的征兵队列。
func deleteStalePlayerRecruitQueuesTx(tx *sql.Tx, playerID string, queueIDs []string) error {
	placeholders := strings.TrimRight(strings.Repeat("?,", len(queueIDs)), ",")
	args := make([]any, 0, len(queueIDs)+1)
	args = append(args, playerID)
	for _, queueID := range queueIDs {
		args = append(args, queueID)
	}
	_, err := tx.Exec(
		`DELETE FROM player_recruit_queues
		 WHERE player_id = ? AND queue_id NOT IN (`+placeholders+`)`,
		args...,
	)
	return err
}
