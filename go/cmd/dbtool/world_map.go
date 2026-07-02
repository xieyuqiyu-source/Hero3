// 本文件提供世界地图权威坐标维护命令。
package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"strings"

	"hero3/internal/app/game"
	"hero3/internal/infrastructure/storage"
)

type worldPositionBackfillOptions struct {
	BatchSize  int
	MaxBatches int
}

type worldPositionBackfillResult struct {
	TotalPlayers     int
	MissingBefore    int
	Created          int
	Skipped          int
	Conflicts        int
	Failed           int
	Remaining        int
	ConflictDetails  []string
	Failures         []string
	ProcessedBatches int
}

// runBackfillWorldPositions 为所有已有玩家补齐世界地图权威坐标。
func runBackfillWorldPositions(args []string) error {
	flags := flag.NewFlagSet("backfill-world-positions", flag.ContinueOnError)
	dsn := flags.String("dsn", "", "目标数据库 DSN，默认使用 HERO3_DATABASE_DSN")
	allowNonTest := flags.Bool("allow-non-test", false, "允许回填非 test_ 前缀数据库")
	batchSize := flags.Int("batch-size", 500, "每批扫描缺失世界坐标的玩家数量")
	maxBatches := flags.Int("max-batches", 0, "最多执行批次数，0 表示直到补完")
	if err := flags.Parse(args); err != nil {
		return err
	}
	resolvedDSN, databaseName, err := resolveWritableDbtoolDSN(*dsn, *allowNonTest)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), longCommandTimeout)
	defer cancel()
	db, err := storage.OpenMySQL(ctx, resolvedDSN)
	if err != nil {
		return err
	}
	defer db.Close()
	if err := ensureWorldPositionsTable(ctx, db); err != nil {
		return err
	}
	result, err := backfillWorldPositions(ctx, db, worldPositionBackfillOptions{BatchSize: *batchSize, MaxBatches: *maxBatches})
	fmt.Printf("世界地图坐标补齐：database=%s totalPlayers=%d missingBefore=%d created=%d skipped=%d conflicts=%d failed=%d remaining=%d batches=%d\n", databaseName, result.TotalPlayers, result.MissingBefore, result.Created, result.Skipped, result.Conflicts, result.Failed, result.Remaining, result.ProcessedBatches)
	for _, detail := range result.ConflictDetails {
		fmt.Printf("冲突：%s\n", detail)
	}
	for _, failure := range result.Failures {
		fmt.Printf("失败：%s\n", failure)
	}
	return err
}

// ensureWorldPositionsTable 只确保世界坐标表存在，避免回填命令触发全库兼容迁移。
func ensureWorldPositionsTable(ctx context.Context, db *sql.DB) error {
	_, err := db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS player_world_positions (
		player_id VARCHAR(64) NOT NULL,
		world_id VARCHAR(64) NOT NULL DEFAULT 'world_1',
		x INT NOT NULL,
		y INT NOT NULL,
		assigned_by VARCHAR(32) NOT NULL DEFAULT 'create',
		created_at DATETIME(6) NOT NULL,
		updated_at DATETIME(6) NOT NULL,
		PRIMARY KEY (player_id),
		UNIQUE KEY uk_player_world_positions_world_xy (world_id, x, y),
		INDEX idx_player_world_positions_world_xy (world_id, x, y),
		INDEX idx_player_world_positions_world_player (world_id, player_id),
		CONSTRAINT fk_player_world_positions_player
			FOREIGN KEY (player_id) REFERENCES players(id)
			ON DELETE CASCADE
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`)
	return err
}

// backfillWorldPositions 分批扫描缺少世界坐标的玩家并补齐权威坐标。
func backfillWorldPositions(ctx context.Context, db *sql.DB, options worldPositionBackfillOptions) (worldPositionBackfillResult, error) {
	options = normalizeWorldPositionBackfillOptions(options)
	var result worldPositionBackfillResult
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM players`).Scan(&result.TotalPlayers); err != nil {
		return result, err
	}
	missingBefore, err := countMissingWorldPositions(ctx, db)
	if err != nil {
		return result, err
	}
	result.MissingBefore = missingBefore
	result.Skipped = result.TotalPlayers - result.MissingBefore
	repo := storage.NewMySQLRepository(db)
	var firstFailureErr error
	lastPlayerID := ""
	for {
		if options.MaxBatches > 0 && result.ProcessedBatches >= options.MaxBatches {
			break
		}
		playerIDs, err := listMissingWorldPositionPlayerIDs(ctx, db, lastPlayerID, options.BatchSize)
		if err != nil {
			return result, err
		}
		if len(playerIDs) == 0 {
			break
		}
		result.ProcessedBatches++
		for _, playerID := range playerIDs {
			lastPlayerID = playerID
			expected := game.LegacyWorldCoordinateForPlayer(playerID)
			position, err := repo.EnsureWorldPosition(playerID, "migration", &expected)
			if err != nil {
				if firstFailureErr == nil {
					firstFailureErr = err
				}
				result.Failed++
				result.Failures = append(result.Failures, playerID+": "+err.Error())
				continue
			}
			if position.X != expected.X || position.Y != expected.Y {
				result.Conflicts++
				result.ConflictDetails = append(result.ConflictDetails, fmt.Sprintf("%s: (%d,%d) -> (%d,%d)", playerID, expected.X, expected.Y, position.X, position.Y))
			}
			result.Created++
		}
	}
	remaining, err := countMissingWorldPositions(ctx, db)
	if err != nil {
		return result, err
	}
	result.Remaining = remaining
	if result.Failed > 0 {
		return result, firstFailureErr
	}
	return result, nil
}

// normalizeWorldPositionBackfillOptions 规范化世界坐标批量回填参数。
func normalizeWorldPositionBackfillOptions(options worldPositionBackfillOptions) worldPositionBackfillOptions {
	if options.BatchSize <= 0 {
		options.BatchSize = 500
	}
	if options.BatchSize > 5000 {
		options.BatchSize = 5000
	}
	if options.MaxBatches < 0 {
		options.MaxBatches = 0
	}
	return options
}

// countMissingWorldPositions 统计还没有世界坐标的玩家数。
func countMissingWorldPositions(ctx context.Context, db *sql.DB) (int, error) {
	var count int
	err := db.QueryRowContext(ctx, `SELECT COUNT(*)
		FROM players p
		LEFT JOIN player_world_positions w ON w.player_id = p.id
		WHERE w.player_id IS NULL`).Scan(&count)
	return count, err
}

// listMissingWorldPositionPlayerIDs 按玩家 ID 分页读取缺少世界坐标的玩家。
func listMissingWorldPositionPlayerIDs(ctx context.Context, db *sql.DB, afterPlayerID string, limit int) ([]string, error) {
	afterPlayerID = strings.TrimSpace(afterPlayerID)
	if limit <= 0 {
		limit = 500
	}
	rows, err := db.QueryContext(ctx, `SELECT p.id
		FROM players p
		LEFT JOIN player_world_positions w ON w.player_id = p.id
		WHERE w.player_id IS NULL AND p.id > ?
		ORDER BY p.id ASC
		LIMIT ?`, afterPlayerID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	playerIDs := []string{}
	for rows.Next() {
		var playerID string
		if err := rows.Scan(&playerID); err != nil {
			return nil, err
		}
		playerIDs = append(playerIDs, playerID)
	}
	return playerIDs, rows.Err()
}
