// 本文件归口 player_resources 影子表回填和一致性校验。
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

const longCommandTimeout = 10 * time.Minute

type resourceBackfillResult struct {
	Players int
	Rows    int
}

type resourceVerifyResult struct {
	Players      int
	ExpectedRows int
	ActualRows   int
	Mismatches   int
}

type resourceSnapshot struct {
	Amount   int
	Capacity int
}

// runBackfillResources 从 players.state_json 回填 player_resources 影子表。
func runBackfillResources(args []string) error {
	flags := flag.NewFlagSet("backfill-resources", flag.ContinueOnError)
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
	result, err := backfillPlayerResources(ctx, *dsn)
	if err != nil {
		return err
	}
	fmt.Printf("资源影子表回填完成：数据库 %s，玩家 %d，资源行 %d\n", databaseName, result.Players, result.Rows)
	return nil
}

// runVerifyResources 校验 player_resources 是否和 state_json.resources 一致。
func runVerifyResources(args []string) error {
	flags := flag.NewFlagSet("verify-resources", flag.ContinueOnError)
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
	result, err := verifyPlayerResources(ctx, *dsn)
	if err != nil {
		return err
	}
	if result.Mismatches > 0 {
		return fmt.Errorf("资源影子表校验失败：玩家 %d，期望资源行 %d，实际资源行 %d，不一致 %d", result.Players, result.ExpectedRows, result.ActualRows, result.Mismatches)
	}
	databaseName, err := storage.MySQLDatabaseName(*dsn)
	if err != nil {
		return err
	}
	fmt.Printf("资源影子表校验通过：数据库 %s，玩家 %d，资源行 %d\n", databaseName, result.Players, result.ActualRows)
	return nil
}

// backfillPlayerResources 从玩家主状态回填资源影子表。
func backfillPlayerResources(ctx context.Context, dsn string) (resourceBackfillResult, error) {
	db, err := storage.OpenMySQL(ctx, dsn)
	if err != nil {
		return resourceBackfillResult{}, err
	}
	defer db.Close()

	players, err := loadPlayerResourceSnapshots(ctx, db)
	if err != nil {
		return resourceBackfillResult{}, err
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return resourceBackfillResult{}, err
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx, `DELETE FROM player_resources`); err != nil {
		return resourceBackfillResult{}, err
	}
	result := resourceBackfillResult{Players: len(players)}
	for playerID, resources := range players {
		for resourceType, snapshot := range resources {
			if _, err := tx.ExecContext(ctx,
				`INSERT INTO player_resources (player_id, resource_type, amount, capacity, updated_at)
				 VALUES (?, ?, ?, ?, ?)`,
				playerID,
				resourceType,
				snapshot.Amount,
				snapshot.Capacity,
				time.Now().UTC(),
			); err != nil {
				return resourceBackfillResult{}, err
			}
			result.Rows++
		}
	}
	if err := tx.Commit(); err != nil {
		return resourceBackfillResult{}, err
	}
	return result, nil
}

// verifyPlayerResources 校验资源影子表与玩家主状态是否一致。
func verifyPlayerResources(ctx context.Context, dsn string) (resourceVerifyResult, error) {
	db, err := storage.OpenMySQL(ctx, dsn)
	if err != nil {
		return resourceVerifyResult{}, err
	}
	defer db.Close()

	expected, err := loadPlayerResourceSnapshots(ctx, db)
	if err != nil {
		return resourceVerifyResult{}, err
	}
	actual, err := loadPlayerResourceRows(ctx, db)
	if err != nil {
		return resourceVerifyResult{}, err
	}

	result := resourceVerifyResult{Players: len(expected)}
	for playerID, resources := range expected {
		result.ExpectedRows += len(resources)
		for resourceType, want := range resources {
			got, ok := actual[playerID][resourceType]
			if !ok || got != want {
				result.Mismatches++
			}
		}
	}
	for playerID, resources := range actual {
		result.ActualRows += len(resources)
		for resourceType := range resources {
			if _, ok := expected[playerID][resourceType]; !ok {
				result.Mismatches++
			}
		}
	}
	return result, nil
}

// loadPlayerResourceSnapshots 从 players.state_json 读取资源快照。
func loadPlayerResourceSnapshots(ctx context.Context, db *sql.DB) (map[string]map[string]resourceSnapshot, error) {
	rows, err := db.QueryContext(ctx, `SELECT id, state_json FROM players ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := map[string]map[string]resourceSnapshot{}
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
		result[playerID] = resourceSnapshotsFromState(state.Resources)
	}
	return result, rows.Err()
}

// loadPlayerResourceRows 读取 player_resources 当前内容。
func loadPlayerResourceRows(ctx context.Context, db *sql.DB) (map[string]map[string]resourceSnapshot, error) {
	rows, err := db.QueryContext(ctx, `SELECT player_id, resource_type, amount, capacity FROM player_resources ORDER BY player_id, resource_type`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := map[string]map[string]resourceSnapshot{}
	for rows.Next() {
		var playerID string
		var resourceType string
		var snapshot resourceSnapshot
		if err := rows.Scan(&playerID, &resourceType, &snapshot.Amount, &snapshot.Capacity); err != nil {
			return nil, err
		}
		if result[playerID] == nil {
			result[playerID] = map[string]resourceSnapshot{}
		}
		result[playerID][resourceType] = snapshot
	}
	return result, rows.Err()
}

// resourceSnapshotsFromState 从 ResourceState 生成可比较资源快照。
func resourceSnapshotsFromState(resources game.ResourceState) map[string]resourceSnapshot {
	resourceTypes := map[string]struct{}{}
	for resourceType := range resources.Items {
		resourceTypes[resourceType] = struct{}{}
	}
	for resourceType := range resources.Capacity {
		resourceTypes[resourceType] = struct{}{}
	}
	snapshots := map[string]resourceSnapshot{}
	for resourceType := range resourceTypes {
		snapshots[resourceType] = resourceSnapshot{
			Amount:   resources.Items[resourceType],
			Capacity: resources.Capacity[resourceType],
		}
	}
	return snapshots
}
