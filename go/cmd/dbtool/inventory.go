// 本文件归口 player_inventory 权威表回填和兼容快照一致性校验。
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

type inventoryBackfillResult struct {
	Players int
	Rows    int
}

type inventoryVerifyResult struct {
	Players      int
	ExpectedRows int
	ActualRows   int
	Mismatches   int
}

type inventorySnapshot struct {
	Amount     int
	ObtainedAt string
	UpdatedAt  string
}

// runBackfillInventory 从旧 players.state_json 兼容快照回填 player_inventory 权威表。
func runBackfillInventory(args []string) error {
	flags := flag.NewFlagSet("backfill-inventory", flag.ContinueOnError)
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
	result, err := backfillPlayerInventory(ctx, *dsn)
	if err != nil {
		return err
	}
	fmt.Printf("背包权威表回填完成：数据库 %s，玩家 %d，道具行 %d\n", databaseName, result.Players, result.Rows)
	return nil
}

// runVerifyInventory 校验 player_inventory 权威表是否和 state_json.inventory 兼容快照一致。
func runVerifyInventory(args []string) error {
	flags := flag.NewFlagSet("verify-inventory", flag.ContinueOnError)
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
	result, err := verifyPlayerInventory(ctx, *dsn)
	if err != nil {
		return err
	}
	if result.Mismatches > 0 {
		return fmt.Errorf("背包权威表兼容快照校验失败：玩家 %d，期望道具行 %d，实际道具行 %d，不一致 %d", result.Players, result.ExpectedRows, result.ActualRows, result.Mismatches)
	}
	databaseName, err := storage.MySQLDatabaseName(*dsn)
	if err != nil {
		return err
	}
	fmt.Printf("背包权威表兼容快照校验通过：数据库 %s，玩家 %d，道具行 %d\n", databaseName, result.Players, result.ActualRows)
	return nil
}

// backfillPlayerInventory 从玩家主状态兼容快照回填背包权威表。
func backfillPlayerInventory(ctx context.Context, dsn string) (inventoryBackfillResult, error) {
	db, err := storage.OpenMySQL(ctx, dsn)
	if err != nil {
		return inventoryBackfillResult{}, err
	}
	defer db.Close()

	players, err := loadPlayerInventorySnapshots(ctx, db)
	if err != nil {
		return inventoryBackfillResult{}, err
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return inventoryBackfillResult{}, err
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx, `DELETE FROM player_inventory`); err != nil {
		return inventoryBackfillResult{}, err
	}
	result := inventoryBackfillResult{Players: len(players)}
	for playerID, inventory := range players {
		for itemID, snapshot := range inventory {
			if _, err := tx.ExecContext(ctx,
				`INSERT INTO player_inventory (player_id, item_id, amount, obtained_at, updated_at)
				 VALUES (?, ?, ?, ?, ?)`,
				playerID,
				itemID,
				snapshot.Amount,
				inventoryTimeArg(snapshot.ObtainedAt),
				inventoryTimeArg(snapshot.UpdatedAt),
			); err != nil {
				return inventoryBackfillResult{}, err
			}
			result.Rows++
		}
	}
	if err := tx.Commit(); err != nil {
		return inventoryBackfillResult{}, err
	}
	return result, nil
}

// verifyPlayerInventory 校验背包权威表与玩家主状态兼容快照是否一致。
func verifyPlayerInventory(ctx context.Context, dsn string) (inventoryVerifyResult, error) {
	db, err := storage.OpenMySQL(ctx, dsn)
	if err != nil {
		return inventoryVerifyResult{}, err
	}
	defer db.Close()

	expected, err := loadPlayerInventorySnapshots(ctx, db)
	if err != nil {
		return inventoryVerifyResult{}, err
	}
	actual, err := loadPlayerInventoryRows(ctx, db)
	if err != nil {
		return inventoryVerifyResult{}, err
	}

	result := inventoryVerifyResult{Players: len(expected)}
	for playerID, inventory := range expected {
		result.ExpectedRows += len(inventory)
		for itemID, want := range inventory {
			got, ok := actual[playerID][itemID]
			if !ok || got != want {
				result.Mismatches++
			}
		}
	}
	for playerID, inventory := range actual {
		result.ActualRows += len(inventory)
		for itemID := range inventory {
			if _, ok := expected[playerID][itemID]; !ok {
				result.Mismatches++
			}
		}
	}
	return result, nil
}

// loadPlayerInventorySnapshots 从 players.state_json 读取背包快照。
func loadPlayerInventorySnapshots(ctx context.Context, db *sql.DB) (map[string]map[string]inventorySnapshot, error) {
	rows, err := db.QueryContext(ctx, `SELECT id, state_json FROM players ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := map[string]map[string]inventorySnapshot{}
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
		result[playerID] = inventorySnapshotsFromState(state.Inventory)
	}
	return result, rows.Err()
}

// loadPlayerInventoryRows 读取 player_inventory 当前内容。
func loadPlayerInventoryRows(ctx context.Context, db *sql.DB) (map[string]map[string]inventorySnapshot, error) {
	rows, err := db.QueryContext(ctx, `SELECT player_id, item_id, amount, obtained_at, updated_at FROM player_inventory ORDER BY player_id, item_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := map[string]map[string]inventorySnapshot{}
	for rows.Next() {
		var playerID string
		var itemID string
		var snapshot inventorySnapshot
		var obtainedAt sql.NullTime
		var updatedAt sql.NullTime
		if err := rows.Scan(&playerID, &itemID, &snapshot.Amount, &obtainedAt, &updatedAt); err != nil {
			return nil, err
		}
		if obtainedAt.Valid {
			snapshot.ObtainedAt = obtainedAt.Time.UTC().Format(time.RFC3339)
		}
		if updatedAt.Valid {
			snapshot.UpdatedAt = updatedAt.Time.UTC().Format(time.RFC3339)
		}
		if result[playerID] == nil {
			result[playerID] = map[string]inventorySnapshot{}
		}
		result[playerID][itemID] = snapshot
	}
	return result, rows.Err()
}

// inventorySnapshotsFromState 从 Inventory 生成可比较背包快照。
func inventorySnapshotsFromState(inventory map[string]game.ItemStack) map[string]inventorySnapshot {
	snapshots := map[string]inventorySnapshot{}
	for itemID, stack := range inventory {
		itemID = strings.TrimSpace(firstNonEmpty(itemID, stack.ItemID))
		if itemID == "" || stack.Amount <= 0 {
			continue
		}
		snapshots[itemID] = inventorySnapshot{
			Amount:     stack.Amount,
			ObtainedAt: stack.ObtainedAt,
			UpdatedAt:  stack.UpdatedAt,
		}
	}
	return snapshots
}

// inventoryTimeArg 把 RFC3339 时间字符串转为数据库时间或 NULL。
func inventoryTimeArg(value string) any {
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

// firstNonEmpty 返回第一个非空字符串。
func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
