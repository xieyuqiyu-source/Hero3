// 本文件归口 player_buildings 与 player_resource_slots 权威表回填和一致性校验。
package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"strings"
	"time"

	"hero3/internal/app/game"
	"hero3/internal/infrastructure/storage"
)

type buildingBackfillResult struct {
	Players int
	Rows    int
}

type buildingVerifyResult struct {
	Players      int
	ExpectedRows int
	ActualRows   int
	Mismatches   int
}

type resourceSlotBackfillResult struct {
	Players int
	Rows    int
}

type resourceSlotVerifyResult struct {
	Players      int
	ExpectedRows int
	ActualRows   int
	Mismatches   int
}

type buildingSnapshot struct {
	Type          string
	Level         int
	Status        string
	UpgradeEndsAt string
	StatusEndsAt  string
}

type resourceSlotSnapshot struct {
	ResourceType string
	BuildingID   string
	UnlockedBy   string
	UnlockedAt   string
}

// runBackfillBuildings 从旧 players.state_json.buildings 回填 player_buildings 权威表。
func runBackfillBuildings(args []string) error {
	flags := flag.NewFlagSet("backfill-buildings", flag.ContinueOnError)
	dsn := flags.String("dsn", "", "目标数据库 DSN，默认使用 HERO3_DATABASE_DSN")
	allowNonTest := flags.Bool("allow-non-test", false, "允许回填非 test_ 前缀数据库")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if strings.TrimSpace(*dsn) == "" {
		configured, err := configuredDSN()
		if err != nil {
			return err
		}
		*dsn = configured
	}
	databaseName, err := storage.MySQLDatabaseName(*dsn)
	if err != nil {
		return err
	}
	if !strings.HasPrefix(databaseName, "test_") && !*allowNonTest {
		return fmt.Errorf("target database must use test_ prefix or pass --allow-non-test")
	}
	ctx, cancel := context.WithTimeout(context.Background(), longCommandTimeout)
	defer cancel()
	result, err := backfillPlayerBuildings(ctx, *dsn)
	if err != nil {
		return err
	}
	fmt.Printf("建筑权威表回填完成：数据库 %s，玩家 %d，建筑行 %d\n", databaseName, result.Players, result.Rows)
	return nil
}

// runVerifyBuildings 校验 player_buildings 权威表是否和 state_json.buildings 兼容快照一致。
func runVerifyBuildings(args []string) error {
	flags := flag.NewFlagSet("verify-buildings", flag.ContinueOnError)
	dsn := flags.String("dsn", "", "目标数据库 DSN，默认使用 HERO3_DATABASE_DSN")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if strings.TrimSpace(*dsn) == "" {
		configured, err := configuredDSN()
		if err != nil {
			return err
		}
		*dsn = configured
	}
	ctx, cancel := context.WithTimeout(context.Background(), longCommandTimeout)
	defer cancel()
	result, err := verifyPlayerBuildings(ctx, *dsn)
	if err != nil {
		return err
	}
	if result.Mismatches > 0 {
		return fmt.Errorf("建筑权威表兼容快照校验失败：玩家 %d，期望建筑行 %d，实际建筑行 %d，不一致 %d", result.Players, result.ExpectedRows, result.ActualRows, result.Mismatches)
	}
	databaseName, err := storage.MySQLDatabaseName(*dsn)
	if err != nil {
		return err
	}
	fmt.Printf("建筑权威表兼容快照校验通过：数据库 %s，玩家 %d，建筑行 %d\n", databaseName, result.Players, result.ActualRows)
	return nil
}

// runBackfillResourceSlots 从建筑快照回填 player_resource_slots 权威表。
func runBackfillResourceSlots(args []string) error {
	flags := flag.NewFlagSet("backfill-resource-slots", flag.ContinueOnError)
	dsn := flags.String("dsn", "", "目标数据库 DSN，默认使用 HERO3_DATABASE_DSN")
	allowNonTest := flags.Bool("allow-non-test", false, "允许回填非 test_ 前缀数据库")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if strings.TrimSpace(*dsn) == "" {
		configured, err := configuredDSN()
		if err != nil {
			return err
		}
		*dsn = configured
	}
	databaseName, err := storage.MySQLDatabaseName(*dsn)
	if err != nil {
		return err
	}
	if !strings.HasPrefix(databaseName, "test_") && !*allowNonTest {
		return fmt.Errorf("target database must use test_ prefix or pass --allow-non-test")
	}
	ctx, cancel := context.WithTimeout(context.Background(), longCommandTimeout)
	defer cancel()
	result, err := backfillPlayerResourceSlots(ctx, *dsn)
	if err != nil {
		return err
	}
	fmt.Printf("资源田格子权威表回填完成：数据库 %s，玩家 %d，格子行 %d\n", databaseName, result.Players, result.Rows)
	return nil
}

// runVerifyResourceSlots 校验 player_resource_slots 权威表是否和兼容快照一致。
func runVerifyResourceSlots(args []string) error {
	flags := flag.NewFlagSet("verify-resource-slots", flag.ContinueOnError)
	dsn := flags.String("dsn", "", "目标数据库 DSN，默认使用 HERO3_DATABASE_DSN")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if strings.TrimSpace(*dsn) == "" {
		configured, err := configuredDSN()
		if err != nil {
			return err
		}
		*dsn = configured
	}
	ctx, cancel := context.WithTimeout(context.Background(), longCommandTimeout)
	defer cancel()
	result, err := verifyPlayerResourceSlots(ctx, *dsn)
	if err != nil {
		return err
	}
	if result.Mismatches > 0 {
		return fmt.Errorf("资源田格子权威表兼容快照校验失败：玩家 %d，期望格子行 %d，实际格子行 %d，不一致 %d", result.Players, result.ExpectedRows, result.ActualRows, result.Mismatches)
	}
	databaseName, err := storage.MySQLDatabaseName(*dsn)
	if err != nil {
		return err
	}
	fmt.Printf("资源田格子权威表兼容快照校验通过：数据库 %s，玩家 %d，格子行 %d\n", databaseName, result.Players, result.ActualRows)
	return nil
}

// backfillPlayerBuildings 从玩家主状态兼容快照回填建筑权威表。
func backfillPlayerBuildings(ctx context.Context, dsn string) (buildingBackfillResult, error) {
	db, err := storage.OpenMySQL(ctx, dsn)
	if err != nil {
		return buildingBackfillResult{}, err
	}
	defer db.Close()
	expected, err := loadPlayerBuildingSnapshots(ctx, db)
	if err != nil {
		return buildingBackfillResult{}, err
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return buildingBackfillResult{}, err
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.ExecContext(ctx, `DELETE FROM player_buildings`); err != nil {
		return buildingBackfillResult{}, err
	}
	result := buildingBackfillResult{Players: len(expected)}
	now := time.Now().UTC()
	for playerID, buildings := range expected {
		for buildingID, snapshot := range buildings {
			if _, err := tx.ExecContext(ctx,
				`INSERT INTO player_buildings (player_id, building_id, building_type, level, status, upgrade_ends_at, status_ends_at, updated_at)
				 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
				playerID,
				buildingID,
				snapshot.Type,
				snapshot.Level,
				snapshot.Status,
				timeArg(snapshot.UpgradeEndsAt),
				timeArg(snapshot.StatusEndsAt),
				now,
			); err != nil {
				return buildingBackfillResult{}, err
			}
			result.Rows++
		}
	}
	if err := tx.Commit(); err != nil {
		return buildingBackfillResult{}, err
	}
	return result, nil
}

// verifyPlayerBuildings 校验建筑权威表与玩家主状态兼容快照是否一致。
func verifyPlayerBuildings(ctx context.Context, dsn string) (buildingVerifyResult, error) {
	db, err := storage.OpenMySQL(ctx, dsn)
	if err != nil {
		return buildingVerifyResult{}, err
	}
	defer db.Close()
	expected, err := loadPlayerBuildingSnapshots(ctx, db)
	if err != nil {
		return buildingVerifyResult{}, err
	}
	actual, err := loadPlayerBuildingRows(ctx, db)
	if err != nil {
		return buildingVerifyResult{}, err
	}
	result := buildingVerifyResult{Players: len(expected)}
	for playerID, buildings := range expected {
		result.ExpectedRows += len(buildings)
		for buildingID, want := range buildings {
			got, ok := actual[playerID][buildingID]
			if !ok || got != want {
				result.Mismatches++
			}
		}
	}
	for playerID, buildings := range actual {
		result.ActualRows += len(buildings)
		for buildingID := range buildings {
			if _, ok := expected[playerID][buildingID]; !ok {
				result.Mismatches++
			}
		}
	}
	return result, nil
}

// backfillPlayerResourceSlots 从玩家建筑快照回填资源田格子权威表。
func backfillPlayerResourceSlots(ctx context.Context, dsn string) (resourceSlotBackfillResult, error) {
	db, err := storage.OpenMySQL(ctx, dsn)
	if err != nil {
		return resourceSlotBackfillResult{}, err
	}
	defer db.Close()
	expected, err := loadPlayerResourceSlotSnapshots(ctx, db)
	if err != nil {
		return resourceSlotBackfillResult{}, err
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return resourceSlotBackfillResult{}, err
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.ExecContext(ctx, `DELETE FROM player_resource_slots`); err != nil {
		return resourceSlotBackfillResult{}, err
	}
	result := resourceSlotBackfillResult{Players: len(expected)}
	now := time.Now().UTC()
	for playerID, slots := range expected {
		for slotID, snapshot := range slots {
			if _, err := tx.ExecContext(ctx,
				`INSERT INTO player_resource_slots (player_id, slot_id, resource_type, building_id, unlocked_by, unlocked_at, updated_at)
				 VALUES (?, ?, ?, ?, ?, ?, ?)`,
				playerID,
				slotID,
				snapshot.ResourceType,
				snapshot.BuildingID,
				snapshot.UnlockedBy,
				timeArg(snapshot.UnlockedAt),
				now,
			); err != nil {
				return resourceSlotBackfillResult{}, err
			}
			result.Rows++
		}
	}
	if err := tx.Commit(); err != nil {
		return resourceSlotBackfillResult{}, err
	}
	return result, nil
}

// verifyPlayerResourceSlots 校验资源田格子权威表与玩家主状态兼容快照是否一致。
func verifyPlayerResourceSlots(ctx context.Context, dsn string) (resourceSlotVerifyResult, error) {
	db, err := storage.OpenMySQL(ctx, dsn)
	if err != nil {
		return resourceSlotVerifyResult{}, err
	}
	defer db.Close()
	expected, err := loadPlayerResourceSlotSnapshots(ctx, db)
	if err != nil {
		return resourceSlotVerifyResult{}, err
	}
	actual, err := loadPlayerResourceSlotRows(ctx, db)
	if err != nil {
		return resourceSlotVerifyResult{}, err
	}
	result := resourceSlotVerifyResult{Players: len(expected)}
	for playerID, slots := range expected {
		result.ExpectedRows += len(slots)
		for slotID, want := range slots {
			got, ok := actual[playerID][slotID]
			if !ok || got != want {
				result.Mismatches++
			}
		}
	}
	for playerID, slots := range actual {
		result.ActualRows += len(slots)
		for slotID := range slots {
			if _, ok := expected[playerID][slotID]; !ok {
				result.Mismatches++
			}
		}
	}
	return result, nil
}

// loadPlayerBuildingSnapshots 从 players.state_json 读取建筑快照。
func loadPlayerBuildingSnapshots(ctx context.Context, db *sql.DB) (map[string]map[string]buildingSnapshot, error) {
	states, err := loadPlayerStates(ctx, db)
	if err != nil {
		return nil, err
	}
	result := map[string]map[string]buildingSnapshot{}
	for playerID, state := range states {
		result[playerID] = buildingSnapshotsFromState(state.Buildings)
	}
	return result, nil
}

// loadPlayerBuildingRows 读取 player_buildings 当前内容。
func loadPlayerBuildingRows(ctx context.Context, db *sql.DB) (map[string]map[string]buildingSnapshot, error) {
	rows, err := db.QueryContext(ctx, `SELECT player_id, building_id, building_type, level, status, upgrade_ends_at, status_ends_at FROM player_buildings ORDER BY player_id, building_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := map[string]map[string]buildingSnapshot{}
	for rows.Next() {
		var playerID string
		var buildingID string
		var snapshot buildingSnapshot
		var upgradeEndsAt sql.NullTime
		var statusEndsAt sql.NullTime
		if err := rows.Scan(&playerID, &buildingID, &snapshot.Type, &snapshot.Level, &snapshot.Status, &upgradeEndsAt, &statusEndsAt); err != nil {
			return nil, err
		}
		if upgradeEndsAt.Valid {
			snapshot.UpgradeEndsAt = upgradeEndsAt.Time.UTC().Format(time.RFC3339)
		}
		if statusEndsAt.Valid {
			snapshot.StatusEndsAt = statusEndsAt.Time.UTC().Format(time.RFC3339)
		}
		if result[playerID] == nil {
			result[playerID] = map[string]buildingSnapshot{}
		}
		result[playerID][buildingID] = snapshot
	}
	return result, rows.Err()
}

// loadPlayerResourceSlotSnapshots 从 state_json.resourceSlots 或建筑快照推导资源田格子快照。
func loadPlayerResourceSlotSnapshots(ctx context.Context, db *sql.DB) (map[string]map[string]resourceSlotSnapshot, error) {
	states, err := loadPlayerStates(ctx, db)
	if err != nil {
		return nil, err
	}
	result := map[string]map[string]resourceSlotSnapshot{}
	for playerID, state := range states {
		slots := state.ResourceSlots
		if len(slots) == 0 {
			slots = deriveResourceSlotsForBackfill(state.Buildings)
		}
		result[playerID] = resourceSlotSnapshotsFromState(slots)
	}
	return result, nil
}

// loadPlayerResourceSlotRows 读取 player_resource_slots 当前内容。
func loadPlayerResourceSlotRows(ctx context.Context, db *sql.DB) (map[string]map[string]resourceSlotSnapshot, error) {
	rows, err := db.QueryContext(ctx, `SELECT player_id, slot_id, resource_type, building_id, unlocked_by, unlocked_at FROM player_resource_slots ORDER BY player_id, slot_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := map[string]map[string]resourceSlotSnapshot{}
	for rows.Next() {
		var playerID string
		var slotID string
		var snapshot resourceSlotSnapshot
		var unlockedAt sql.NullTime
		if err := rows.Scan(&playerID, &slotID, &snapshot.ResourceType, &snapshot.BuildingID, &snapshot.UnlockedBy, &unlockedAt); err != nil {
			return nil, err
		}
		if unlockedAt.Valid {
			_ = unlockedAt.Time
		}
		if result[playerID] == nil {
			result[playerID] = map[string]resourceSlotSnapshot{}
		}
		result[playerID][slotID] = snapshot
	}
	return result, rows.Err()
}

// loadPlayerStates 从 players.state_json 读取玩家状态。
func loadPlayerStates(ctx context.Context, db *sql.DB) (map[string]game.GameState, error) {
	rows, err := db.QueryContext(ctx, `SELECT id, state_json FROM players ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := map[string]game.GameState{}
	for rows.Next() {
		var playerID string
		var stateJSON []byte
		if err := rows.Scan(&playerID, &stateJSON); err != nil {
			return nil, err
		}
		var state game.GameState
		if err := json.Unmarshal(stateJSON, &state); err != nil {
			return nil, err
		}
		game.EnsureCoreBuildings(&state)
		game.ApplyConstructionBureauResourceSlots(&state, time.Now())
		result[playerID] = state
	}
	return result, rows.Err()
}

// buildingSnapshotsFromState 从建筑列表生成可比较建筑快照。
func buildingSnapshotsFromState(buildings []game.Building) map[string]buildingSnapshot {
	result := map[string]buildingSnapshot{}
	for _, building := range buildings {
		building.ID = strings.TrimSpace(building.ID)
		building.Type = strings.TrimSpace(building.Type)
		if building.ID == "" || building.Type == "" {
			continue
		}
		snapshot := buildingSnapshot{Type: building.Type, Level: building.Level, Status: building.Status}
		if building.UpgradeEndsAt != nil {
			snapshot.UpgradeEndsAt = strings.TrimSpace(*building.UpgradeEndsAt)
		}
		if building.StatusEndsAt != nil {
			snapshot.StatusEndsAt = strings.TrimSpace(*building.StatusEndsAt)
		}
		result[building.ID] = snapshot
	}
	return result
}

// resourceSlotSnapshotsFromState 从资源田格子列表生成可比较快照。
func resourceSlotSnapshotsFromState(slots []game.ResourceSlot) map[string]resourceSlotSnapshot {
	result := map[string]resourceSlotSnapshot{}
	for _, slot := range slots {
		slot.ID = strings.TrimSpace(slot.ID)
		slot.ResourceType = strings.TrimSpace(slot.ResourceType)
		if slot.ID == "" || slot.ResourceType == "" {
			continue
		}
		result[slot.ID] = resourceSlotSnapshot{
			ResourceType: slot.ResourceType,
			BuildingID:   strings.TrimSpace(slot.BuildingID),
			UnlockedBy:   strings.TrimSpace(slot.UnlockedBy),
		}
	}
	return result
}

// deriveResourceSlotsForBackfill 为旧存档按资源建筑推导资源田格子，避免生成每次变化的时间。
func deriveResourceSlotsForBackfill(buildings []game.Building) []game.ResourceSlot {
	slots := []game.ResourceSlot{}
	for _, building := range buildings {
		building.ID = strings.TrimSpace(building.ID)
		building.Type = strings.TrimSpace(building.Type)
		if building.ID == "" || building.Type == "" {
			continue
		}
		config, exists := game.GetBuildingConfig(building.Type)
		if !exists || strings.TrimSpace(config.ResourceType) == "" {
			continue
		}
		slots = append(slots, game.ResourceSlot{
			ID:           building.ID,
			ResourceType: strings.TrimSpace(config.ResourceType),
			BuildingID:   building.ID,
			UnlockedBy:   "initial_building",
		})
	}
	return slots
}

// timeArg 把 RFC3339 时间字符串转换成 SQL 参数。
func timeArg(value string) any {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return nil
	}
	return parsed.UTC()
}
