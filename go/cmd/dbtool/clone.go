// 本文件归口 MySQL 数据复制命令，服务于 test_ 测试库迁移。
package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"os"
	"strings"

	"hero3/internal/infrastructure/storage"
)

const defaultCloneBatchSize = 200

type cloneDataOptions struct {
	SourceDSN      string
	TargetDSN      string
	TruncateTarget bool
	BatchSize      int
}

type cloneDataResult struct {
	Tables int
	Rows   int
}

type cloneTablePlan struct {
	CopyTables     []string
	TruncateTables []string
}

// runCloneData 把源库数据复制到目标库，适合把稳定库迁移到 test_ 测试库。
func runCloneData(args []string) error {
	flags := flag.NewFlagSet("clone-data", flag.ContinueOnError)
	sourceDSN := flags.String("source-dsn", os.Getenv("HERO3_SOURCE_DATABASE_DSN"), "源数据库 DSN")
	targetDSN := flags.String("target-dsn", "", "目标数据库 DSN，默认使用 HERO3_DATABASE_DSN")
	truncateTarget := flags.Bool("truncate", false, "复制前清空目标库表数据")
	batchSize := flags.Int("batch-size", defaultCloneBatchSize, "批量插入行数")
	skipResourceRefresh := flags.Bool("skip-resource-refresh", false, "跳过资源影子表回填和校验")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if strings.TrimSpace(*sourceDSN) == "" {
		return fmt.Errorf("source dsn is required, use --source-dsn or HERO3_SOURCE_DATABASE_DSN")
	}
	if strings.TrimSpace(*targetDSN) == "" {
		dsn, err := configuredDSN()
		if err != nil {
			return err
		}
		*targetDSN = dsn
	}
	if *batchSize <= 0 {
		return fmt.Errorf("batch-size must be positive")
	}
	sourceName, err := storage.MySQLDatabaseName(*sourceDSN)
	if err != nil {
		return err
	}
	targetName, err := storage.MySQLDatabaseName(*targetDSN)
	if err != nil {
		return err
	}
	if sourceName == targetName && storage.RedactMySQLDSN(*sourceDSN) == storage.RedactMySQLDSN(*targetDSN) {
		return fmt.Errorf("source and target database must be different")
	}
	if !strings.HasPrefix(targetName, "test_") {
		return fmt.Errorf("target database must use test_ prefix")
	}

	ctx, cancel := context.WithTimeout(context.Background(), longCommandTimeout)
	defer cancel()
	result, err := cloneMySQLData(ctx, cloneDataOptions{
		SourceDSN:      *sourceDSN,
		TargetDSN:      *targetDSN,
		TruncateTarget: *truncateTarget,
		BatchSize:      *batchSize,
	})
	if err != nil {
		return err
	}
	fmt.Printf("数据复制完成：%s -> %s，表 %d，行 %d\n", sourceName, targetName, result.Tables, result.Rows)
	if !*skipResourceRefresh {
		backfillResult, err := backfillPlayerResources(ctx, *targetDSN)
		if err != nil {
			return err
		}
		verifyResult, err := verifyPlayerResources(ctx, *targetDSN)
		if err != nil {
			return err
		}
		if verifyResult.Mismatches > 0 {
			return fmt.Errorf("资源影子表校验失败：玩家 %d，期望资源行 %d，实际资源行 %d，不一致 %d", verifyResult.Players, verifyResult.ExpectedRows, verifyResult.ActualRows, verifyResult.Mismatches)
		}
		fmt.Printf("资源影子表已刷新：玩家 %d，资源行 %d\n", backfillResult.Players, backfillResult.Rows)
	}
	return nil
}

// cloneMySQLData 按表复制 MySQL 数据，避免依赖 mysqldump 权限。
func cloneMySQLData(ctx context.Context, options cloneDataOptions) (cloneDataResult, error) {
	sourceDB, err := storage.OpenMySQL(ctx, options.SourceDSN)
	if err != nil {
		return cloneDataResult{}, err
	}
	defer sourceDB.Close()
	targetDB, err := storage.OpenMySQL(ctx, options.TargetDSN)
	if err != nil {
		return cloneDataResult{}, err
	}
	defer targetDB.Close()

	sourceName, err := storage.MySQLDatabaseName(options.SourceDSN)
	if err != nil {
		return cloneDataResult{}, err
	}
	targetName, err := storage.MySQLDatabaseName(options.TargetDSN)
	if err != nil {
		return cloneDataResult{}, err
	}

	sourceTables, err := listBaseTables(ctx, sourceDB, sourceName)
	if err != nil {
		return cloneDataResult{}, err
	}
	targetTables, err := listBaseTables(ctx, targetDB, targetName)
	if err != nil {
		return cloneDataResult{}, err
	}
	tablePlan := buildCloneTablePlan(sourceTables, targetTables)
	if options.TruncateTarget {
		if err := truncateTables(ctx, targetDB, tablePlan.TruncateTables); err != nil {
			return cloneDataResult{}, err
		}
	}

	result := cloneDataResult{}
	for _, tableName := range tablePlan.CopyTables {
		sourceColumns, err := listTableColumns(ctx, sourceDB, sourceName, tableName)
		if err != nil {
			return cloneDataResult{}, err
		}
		targetColumns, err := listTableColumns(ctx, targetDB, targetName, tableName)
		if err != nil {
			return cloneDataResult{}, err
		}
		columns, skippedTargetColumns, err := cloneableColumns(tableName, sourceColumns, targetColumns)
		if err != nil {
			return cloneDataResult{}, err
		}
		if len(skippedTargetColumns) > 0 {
			fmt.Printf("表 %s 跳过目标新增列：%s\n", tableName, strings.Join(skippedTargetColumns, ","))
		}
		rows, err := cloneTableData(ctx, sourceDB, targetDB, tableName, columns, options.BatchSize)
		if err != nil {
			return cloneDataResult{}, err
		}
		result.Tables++
		result.Rows += rows
		fmt.Printf("已复制表：%s，行 %d\n", tableName, rows)
	}
	return result, nil
}

// buildCloneTablePlan 明确复制源表、清理目标表，避免目标新表残留旧数据。
func buildCloneTablePlan(sourceTables []string, targetTables []string) cloneTablePlan {
	return cloneTablePlan{
		CopyTables:     append([]string(nil), sourceTables...),
		TruncateTables: append([]string(nil), targetTables...),
	}
}

// cloneableColumns 返回源库和目标库都存在的可复制列，并报告目标库新增列。
func cloneableColumns(tableName string, sourceColumns []string, targetColumns []string) ([]string, []string, error) {
	targetColumnSet := map[string]struct{}{}
	for _, column := range targetColumns {
		targetColumnSet[column] = struct{}{}
	}
	sourceColumnSet := map[string]struct{}{}
	for _, column := range sourceColumns {
		sourceColumnSet[column] = struct{}{}
	}

	cloneColumns := make([]string, 0, len(sourceColumns))
	for _, column := range sourceColumns {
		if _, ok := targetColumnSet[column]; !ok {
			return nil, nil, fmt.Errorf("table %s source column %s is missing in target", tableName, column)
		}
		cloneColumns = append(cloneColumns, column)
	}
	if len(cloneColumns) == 0 {
		return nil, nil, fmt.Errorf("table %s has no cloneable columns", tableName)
	}

	skippedTargetColumns := []string{}
	for _, column := range targetColumns {
		if _, ok := sourceColumnSet[column]; !ok {
			skippedTargetColumns = append(skippedTargetColumns, column)
		}
	}
	return cloneColumns, skippedTargetColumns, nil
}

// listBaseTables 列出源库普通表。
func listBaseTables(ctx context.Context, db *sql.DB, databaseName string) ([]string, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT table_name
		FROM information_schema.tables
		WHERE table_schema = ? AND table_type = 'BASE TABLE'
		ORDER BY table_name
	`, databaseName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tables := []string{}
	for rows.Next() {
		var tableName string
		if err := rows.Scan(&tableName); err != nil {
			return nil, err
		}
		tables = append(tables, tableName)
	}
	return tables, rows.Err()
}

// listTableColumns 按顺序列出表字段。
func listTableColumns(ctx context.Context, db *sql.DB, databaseName string, tableName string) ([]string, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT column_name
		FROM information_schema.columns
		WHERE table_schema = ? AND table_name = ?
		ORDER BY ordinal_position
	`, databaseName, tableName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	columns := []string{}
	for rows.Next() {
		var columnName string
		if err := rows.Scan(&columnName); err != nil {
			return nil, err
		}
		columns = append(columns, columnName)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(columns) == 0 {
		return nil, fmt.Errorf("table %s has no columns", tableName)
	}
	return columns, nil
}

// truncateTables 清空目标表并临时关闭外键检查。
func truncateTables(ctx context.Context, db *sql.DB, tables []string) error {
	if _, err := db.ExecContext(ctx, "SET FOREIGN_KEY_CHECKS = 0"); err != nil {
		return err
	}
	defer func() { _, _ = db.ExecContext(context.Background(), "SET FOREIGN_KEY_CHECKS = 1") }()
	for _, tableName := range tables {
		if _, err := db.ExecContext(ctx, "TRUNCATE TABLE "+quoteIdentifier(tableName)); err != nil {
			return err
		}
	}
	return nil
}

// cloneTableData 复制单表数据。
func cloneTableData(ctx context.Context, sourceDB *sql.DB, targetDB *sql.DB, tableName string, columns []string, batchSize int) (int, error) {
	selectSQL := "SELECT " + joinQuotedIdentifiers(columns) + " FROM " + quoteIdentifier(tableName)
	rows, err := sourceDB.QueryContext(ctx, selectSQL)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	tx, err := targetDB.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback() }()

	total := 0
	batch := make([][]any, 0, batchSize)
	for rows.Next() {
		values := make([]any, len(columns))
		scanTargets := make([]any, len(columns))
		for i := range values {
			scanTargets[i] = &values[i]
		}
		if err := rows.Scan(scanTargets...); err != nil {
			return 0, err
		}
		batch = append(batch, values)
		if len(batch) >= batchSize {
			inserted, err := insertBatch(ctx, tx, tableName, columns, batch)
			if err != nil {
				return 0, err
			}
			total += inserted
			batch = batch[:0]
		}
	}
	if err := rows.Err(); err != nil {
		return 0, err
	}
	if len(batch) > 0 {
		inserted, err := insertBatch(ctx, tx, tableName, columns, batch)
		if err != nil {
			return 0, err
		}
		total += inserted
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return total, nil
}

// insertBatch 批量插入数据。
func insertBatch(ctx context.Context, tx *sql.Tx, tableName string, columns []string, rows [][]any) (int, error) {
	if len(rows) == 0 {
		return 0, nil
	}
	valuePlaceholders := make([]string, 0, len(rows))
	args := make([]any, 0, len(rows)*len(columns))
	rowPlaceholder := "(" + strings.TrimRight(strings.Repeat("?,", len(columns)), ",") + ")"
	for _, row := range rows {
		valuePlaceholders = append(valuePlaceholders, rowPlaceholder)
		args = append(args, row...)
	}
	query := "INSERT INTO " + quoteIdentifier(tableName) + " (" + joinQuotedIdentifiers(columns) + ") VALUES " + strings.Join(valuePlaceholders, ",")
	if _, err := tx.ExecContext(ctx, query, args...); err != nil {
		return 0, err
	}
	return len(rows), nil
}
