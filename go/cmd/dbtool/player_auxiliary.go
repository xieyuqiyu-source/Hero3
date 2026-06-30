// 本文件归口玩家货币与 NPC 状态新权威表的回填和覆盖校验。
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

type currencyBackfillResult struct {
	Players   int
	Rows      int
	Batches   int
	Remaining int
}

type currencyVerifyResult struct {
	Players    int
	ActualRows int
	Missing    int
}

type npcStateBackfillResult struct {
	Players   int
	Rows      int
	Batches   int
	Remaining int
}

type npcStateVerifyResult struct {
	Players      int
	ExpectedRows int
	ActualRows   int
	Missing      int
}

type currencyCompatSnapshot struct {
	CityGold       int
	LastExchangeAt string
}

type auxiliaryBackfillOptions struct {
	BatchSize  int
	MaxBatches int
}

// runBackfillCurrencies 从旧 players.state_json 补齐玩家货币权威表。
func runBackfillCurrencies(args []string) error {
	flags := flag.NewFlagSet("backfill-currencies", flag.ContinueOnError)
	dsn := flags.String("dsn", "", "目标数据库 DSN，默认使用 HERO3_DATABASE_DSN")
	allowNonTest := flags.Bool("allow-non-test", false, "允许回填非 test_ 前缀数据库")
	batchSize := flags.Int("batch-size", 500, "每批最多回填玩家数")
	maxBatches := flags.Int("max-batches", 10, "单次最多执行批次数，0 表示直到补齐")
	if err := flags.Parse(args); err != nil {
		return err
	}
	resolvedDSN, databaseName, err := resolveWritableDbtoolDSN(*dsn, *allowNonTest)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), longCommandTimeout)
	defer cancel()
	result, err := backfillPlayerCurrenciesWithOptions(ctx, resolvedDSN, auxiliaryBackfillOptions{
		BatchSize:  *batchSize,
		MaxBatches: *maxBatches,
	})
	if err != nil {
		return err
	}
	fmt.Printf("玩家货币权威表回填完成：数据库 %s，处理玩家 %d，新增行 %d，批次 %d，剩余缺失 %d\n", databaseName, result.Players, result.Rows, result.Batches, result.Remaining)
	return nil
}

// runVerifyCurrencies 校验 player_currencies 是否覆盖所有玩家。
func runVerifyCurrencies(args []string) error {
	flags := flag.NewFlagSet("verify-currencies", flag.ContinueOnError)
	dsn := flags.String("dsn", "", "目标数据库 DSN，默认使用 HERO3_DATABASE_DSN")
	if err := flags.Parse(args); err != nil {
		return err
	}
	resolvedDSN, databaseName, err := resolveReadonlyDbtoolDSN(*dsn)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), longCommandTimeout)
	defer cancel()
	result, err := verifyPlayerCurrencies(ctx, resolvedDSN)
	if err != nil {
		return err
	}
	if result.Missing > 0 {
		return fmt.Errorf("玩家货币权威表覆盖校验失败：玩家 %d，货币行 %d，缺失 %d", result.Players, result.ActualRows, result.Missing)
	}
	fmt.Printf("玩家货币权威表覆盖校验通过：数据库 %s，玩家 %d，货币行 %d\n", databaseName, result.Players, result.ActualRows)
	return nil
}

// runBackfillNpcStates 从旧 players.state_json.npcState 补齐 NPC 状态权威表。
func runBackfillNpcStates(args []string) error {
	flags := flag.NewFlagSet("backfill-npc-states", flag.ContinueOnError)
	dsn := flags.String("dsn", "", "目标数据库 DSN，默认使用 HERO3_DATABASE_DSN")
	allowNonTest := flags.Bool("allow-non-test", false, "允许回填非 test_ 前缀数据库")
	batchSize := flags.Int("batch-size", 500, "每批最多回填玩家数")
	maxBatches := flags.Int("max-batches", 10, "单次最多执行批次数，0 表示直到补齐")
	if err := flags.Parse(args); err != nil {
		return err
	}
	resolvedDSN, databaseName, err := resolveWritableDbtoolDSN(*dsn, *allowNonTest)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), longCommandTimeout)
	defer cancel()
	result, err := backfillPlayerNpcStatesWithOptions(ctx, resolvedDSN, auxiliaryBackfillOptions{
		BatchSize:  *batchSize,
		MaxBatches: *maxBatches,
	})
	if err != nil {
		return err
	}
	fmt.Printf("玩家 NPC 状态权威表回填完成：数据库 %s，处理旧 NPC 快照玩家 %d，新增行 %d，批次 %d，剩余缺失 %d\n", databaseName, result.Players, result.Rows, result.Batches, result.Remaining)
	return nil
}

// runVerifyNpcStates 校验旧 NPC 快照是否已经有独立权威行承接。
func runVerifyNpcStates(args []string) error {
	flags := flag.NewFlagSet("verify-npc-states", flag.ContinueOnError)
	dsn := flags.String("dsn", "", "目标数据库 DSN，默认使用 HERO3_DATABASE_DSN")
	if err := flags.Parse(args); err != nil {
		return err
	}
	resolvedDSN, databaseName, err := resolveReadonlyDbtoolDSN(*dsn)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), longCommandTimeout)
	defer cancel()
	result, err := verifyPlayerNpcStates(ctx, resolvedDSN)
	if err != nil {
		return err
	}
	if result.Missing > 0 {
		return fmt.Errorf("玩家 NPC 状态权威表覆盖校验失败：玩家 %d，旧 NPC 快照 %d，权威行 %d，缺失 %d", result.Players, result.ExpectedRows, result.ActualRows, result.Missing)
	}
	fmt.Printf("玩家 NPC 状态权威表覆盖校验通过：数据库 %s，旧 NPC 快照 %d，权威行 %d\n", databaseName, result.ExpectedRows, result.ActualRows)
	return nil
}

// backfillPlayerCurrencies 为缺失货币权威行的玩家补齐 player_currencies。
func backfillPlayerCurrencies(ctx context.Context, dsn string) (currencyBackfillResult, error) {
	return backfillPlayerCurrenciesWithOptions(ctx, dsn, auxiliaryBackfillOptions{BatchSize: 500})
}

// backfillPlayerCurrenciesWithOptions 按小批次补齐玩家货币权威行，避免线上长事务。
func backfillPlayerCurrenciesWithOptions(ctx context.Context, dsn string, options auxiliaryBackfillOptions) (currencyBackfillResult, error) {
	options = normalizeAuxiliaryBackfillOptions(options)
	db, err := storage.OpenMySQL(ctx, dsn)
	if err != nil {
		return currencyBackfillResult{}, err
	}
	defer db.Close()

	var result currencyBackfillResult
	for {
		if options.MaxBatches > 0 && result.Batches >= options.MaxBatches {
			break
		}
		snapshots, err := loadMissingPlayerCurrencyCompatSnapshots(ctx, db, options.BatchSize)
		if err != nil {
			return currencyBackfillResult{}, err
		}
		if len(snapshots) == 0 {
			break
		}
		written, err := insertPlayerCurrencyBackfillBatch(ctx, db, snapshots)
		if err != nil {
			return currencyBackfillResult{}, err
		}
		result.Players += len(snapshots)
		result.Rows += written
		result.Batches++
	}
	remaining, err := countMissingPlayerCurrencies(ctx, db)
	if err != nil {
		return currencyBackfillResult{}, err
	}
	result.Remaining = remaining
	return result, nil
}

// verifyPlayerCurrencies 校验每名玩家都有货币权威行。
func verifyPlayerCurrencies(ctx context.Context, dsn string) (currencyVerifyResult, error) {
	db, err := storage.OpenMySQL(ctx, dsn)
	if err != nil {
		return currencyVerifyResult{}, err
	}
	defer db.Close()

	var result currencyVerifyResult
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM players`).Scan(&result.Players); err != nil {
		return currencyVerifyResult{}, err
	}
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM player_currencies`).Scan(&result.ActualRows); err != nil {
		return currencyVerifyResult{}, err
	}
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*)
		FROM players p
		LEFT JOIN player_currencies c ON c.player_id = p.id
		WHERE c.player_id IS NULL`).Scan(&result.Missing); err != nil {
		return currencyVerifyResult{}, err
	}
	return result, nil
}

// backfillPlayerNpcStates 为旧 state_json.npcState 补齐独立 NPC 状态行。
func backfillPlayerNpcStates(ctx context.Context, dsn string) (npcStateBackfillResult, error) {
	return backfillPlayerNpcStatesWithOptions(ctx, dsn, auxiliaryBackfillOptions{BatchSize: 500})
}

// backfillPlayerNpcStatesWithOptions 按小批次补齐玩家 NPC 状态权威行。
func backfillPlayerNpcStatesWithOptions(ctx context.Context, dsn string, options auxiliaryBackfillOptions) (npcStateBackfillResult, error) {
	options = normalizeAuxiliaryBackfillOptions(options)
	db, err := storage.OpenMySQL(ctx, dsn)
	if err != nil {
		return npcStateBackfillResult{}, err
	}
	defer db.Close()

	var result npcStateBackfillResult
	for {
		if options.MaxBatches > 0 && result.Batches >= options.MaxBatches {
			break
		}
		snapshots, err := loadMissingPlayerNpcStateCompatSnapshots(ctx, db, options.BatchSize)
		if err != nil {
			return npcStateBackfillResult{}, err
		}
		if len(snapshots) == 0 {
			break
		}
		written, err := insertPlayerNpcStateBackfillBatch(ctx, db, snapshots)
		if err != nil {
			return npcStateBackfillResult{}, err
		}
		result.Players += len(snapshots)
		result.Rows += written
		result.Batches++
	}
	remaining, err := countMissingPlayerNpcStates(ctx, db)
	if err != nil {
		return npcStateBackfillResult{}, err
	}
	result.Remaining = remaining
	return result, nil
}

// verifyPlayerNpcStates 校验旧 NPC 兼容快照是否已有权威行承接。
func verifyPlayerNpcStates(ctx context.Context, dsn string) (npcStateVerifyResult, error) {
	db, err := storage.OpenMySQL(ctx, dsn)
	if err != nil {
		return npcStateVerifyResult{}, err
	}
	defer db.Close()

	expected, err := loadPlayerNpcStateCompatSnapshots(ctx, db)
	if err != nil {
		return npcStateVerifyResult{}, err
	}
	actual, err := loadPlayerNpcStateRows(ctx, db)
	if err != nil {
		return npcStateVerifyResult{}, err
	}
	result := npcStateVerifyResult{ExpectedRows: len(expected), ActualRows: len(actual)}
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM players`).Scan(&result.Players); err != nil {
		return npcStateVerifyResult{}, err
	}
	for playerID := range expected {
		if _, ok := actual[playerID]; !ok {
			result.Missing++
		}
	}
	return result, nil
}

// loadPlayerCurrencyCompatSnapshots 从 players.state_json 读取旧货币快照，缺省玩家按 0 城金处理。
func loadPlayerCurrencyCompatSnapshots(ctx context.Context, db *sql.DB) (map[string]currencyCompatSnapshot, error) {
	rows, err := db.QueryContext(ctx, `SELECT id, state_json FROM players ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := map[string]currencyCompatSnapshot{}
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
		result[playerID] = currencyCompatSnapshot{
			CityGold:       int(state.CityGold),
			LastExchangeAt: strings.TrimSpace(state.LastExchangeAt),
		}
	}
	return result, rows.Err()
}

// loadMissingPlayerCurrencyCompatSnapshots 读取一批尚未迁移到 player_currencies 的旧快照。
func loadMissingPlayerCurrencyCompatSnapshots(ctx context.Context, db *sql.DB, limit int) (map[string]currencyCompatSnapshot, error) {
	rows, err := db.QueryContext(ctx, `SELECT p.id, p.state_json
		FROM players p
		LEFT JOIN player_currencies c ON c.player_id = p.id
		WHERE c.player_id IS NULL
		ORDER BY p.id
		LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := map[string]currencyCompatSnapshot{}
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
		result[playerID] = currencyCompatSnapshot{
			CityGold:       int(state.CityGold),
			LastExchangeAt: strings.TrimSpace(state.LastExchangeAt),
		}
	}
	return result, rows.Err()
}

// loadPlayerNpcStateCompatSnapshots 从 players.state_json 读取旧 NPC 状态快照。
func loadPlayerNpcStateCompatSnapshots(ctx context.Context, db *sql.DB) (map[string][]byte, error) {
	rows, err := db.QueryContext(ctx, `SELECT id, state_json FROM players ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := map[string][]byte{}
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
		if state.NpcState == nil {
			continue
		}
		npcStateJSON, err := json.Marshal(state.NpcState)
		if err != nil {
			return nil, err
		}
		result[playerID] = npcStateJSON
	}
	return result, rows.Err()
}

// loadMissingPlayerNpcStateCompatSnapshots 读取一批旧 NPC 快照存在但权威表缺失的玩家。
func loadMissingPlayerNpcStateCompatSnapshots(ctx context.Context, db *sql.DB, limit int) (map[string][]byte, error) {
	rows, err := db.QueryContext(ctx, `SELECT p.id, p.state_json
		FROM players p
		LEFT JOIN player_npc_states n ON n.player_id = p.id
		WHERE n.player_id IS NULL
			AND JSON_CONTAINS_PATH(p.state_json, 'one', '$.npcState')
		ORDER BY p.id
		LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := map[string][]byte{}
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
		if state.NpcState == nil {
			continue
		}
		npcStateJSON, err := json.Marshal(state.NpcState)
		if err != nil {
			return nil, err
		}
		result[playerID] = npcStateJSON
	}
	return result, rows.Err()
}

// loadPlayerNpcStateRows 读取当前 NPC 状态权威表覆盖。
func loadPlayerNpcStateRows(ctx context.Context, db *sql.DB) (map[string]struct{}, error) {
	rows, err := db.QueryContext(ctx, `SELECT player_id FROM player_npc_states ORDER BY player_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := map[string]struct{}{}
	for rows.Next() {
		var playerID string
		if err := rows.Scan(&playerID); err != nil {
			return nil, err
		}
		result[playerID] = struct{}{}
	}
	return result, rows.Err()
}

// insertPlayerCurrencyBackfillBatch 用一个短事务写入一批缺失货币行。
func insertPlayerCurrencyBackfillBatch(ctx context.Context, db *sql.DB, snapshots map[string]currencyCompatSnapshot) (int, error) {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback() }()

	written := 0
	now := time.Now().UTC()
	for playerID, snapshot := range snapshots {
		exec, err := tx.ExecContext(ctx,
			`INSERT IGNORE INTO player_currencies (player_id, city_gold, last_exchange_at, updated_at)
			 VALUES (?, ?, ?, ?)`,
			playerID,
			snapshot.CityGold,
			nullableRFC3339(snapshot.LastExchangeAt),
			now,
		)
		if err != nil {
			return 0, err
		}
		affected, _ := exec.RowsAffected()
		written += int(affected)
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return written, nil
}

// insertPlayerNpcStateBackfillBatch 用一个短事务写入一批缺失 NPC 状态行。
func insertPlayerNpcStateBackfillBatch(ctx context.Context, db *sql.DB, snapshots map[string][]byte) (int, error) {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback() }()

	written := 0
	now := time.Now().UTC()
	for playerID, npcStateJSON := range snapshots {
		exec, err := tx.ExecContext(ctx,
			`INSERT IGNORE INTO player_npc_states (player_id, npc_state_json, updated_at)
			 VALUES (?, ?, ?)`,
			playerID,
			npcStateJSON,
			now,
		)
		if err != nil {
			return 0, err
		}
		affected, _ := exec.RowsAffected()
		written += int(affected)
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return written, nil
}

// countMissingPlayerCurrencies 统计尚未补齐货币权威行的玩家数。
func countMissingPlayerCurrencies(ctx context.Context, db *sql.DB) (int, error) {
	var missing int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*)
		FROM players p
		LEFT JOIN player_currencies c ON c.player_id = p.id
		WHERE c.player_id IS NULL`).Scan(&missing); err != nil {
		return 0, err
	}
	return missing, nil
}

// countMissingPlayerNpcStates 统计旧 NPC 快照仍未补齐权威行的玩家数。
func countMissingPlayerNpcStates(ctx context.Context, db *sql.DB) (int, error) {
	var missing int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*)
		FROM players p
		LEFT JOIN player_npc_states n ON n.player_id = p.id
		WHERE n.player_id IS NULL
			AND JSON_CONTAINS_PATH(p.state_json, 'one', '$.npcState')`).Scan(&missing); err != nil {
		return 0, err
	}
	return missing, nil
}

// normalizeAuxiliaryBackfillOptions 统一回填批次参数，避免 0 或负数导致异常扫描。
func normalizeAuxiliaryBackfillOptions(options auxiliaryBackfillOptions) auxiliaryBackfillOptions {
	if options.BatchSize <= 0 {
		options.BatchSize = 500
	}
	return options
}

// resolveWritableDbtoolDSN 解析写入型 dbtool DSN，并保护非测试库误操作。
func resolveWritableDbtoolDSN(rawDSN string, allowNonTest bool) (string, string, error) {
	resolvedDSN, databaseName, err := resolveReadonlyDbtoolDSN(rawDSN)
	if err != nil {
		return "", "", err
	}
	if !strings.HasPrefix(databaseName, "test_") && !allowNonTest {
		return "", "", fmt.Errorf("target database must use test_ prefix or pass --allow-non-test")
	}
	return resolvedDSN, databaseName, nil
}

// resolveReadonlyDbtoolDSN 解析只读型 dbtool DSN。
func resolveReadonlyDbtoolDSN(rawDSN string) (string, string, error) {
	resolvedDSN := strings.TrimSpace(rawDSN)
	if resolvedDSN == "" {
		configured, err := configuredDSN()
		if err != nil {
			return "", "", err
		}
		resolvedDSN = configured
	}
	databaseName, err := storage.MySQLDatabaseName(resolvedDSN)
	if err != nil {
		return "", "", err
	}
	return resolvedDSN, databaseName, nil
}
