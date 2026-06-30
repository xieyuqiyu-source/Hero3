// 本文件归口战报清理专用索引检查与创建命令，避免清理任务扫大表。
package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"strings"

	"hero3/internal/infrastructure/storage"
)

type reportCleanupIndexDefinition struct {
	Table   string
	Name    string
	Columns []string
}

type reportCleanupIndexResult struct {
	DatabaseName string
	DryRun       bool
	Checked      int
	Missing      []reportCleanupIndexDefinition
	Created      []reportCleanupIndexDefinition
}

var reportCleanupIndexes = []reportCleanupIndexDefinition{
	{Table: "battle_reports", Name: "idx_battle_reports_cleanup_created", Columns: []string{"created_at", "id"}},
	{Table: "battle_reports", Name: "idx_battle_reports_cleanup_deleted", Columns: []string{"deleted_by_player", "created_at", "id"}},
	{Table: "battle_reports", Name: "idx_battle_reports_player_deleted_created", Columns: []string{"player_id", "deleted_by_player", "created_at"}},
	{Table: "battle_reports", Name: "idx_battle_reports_event", Columns: []string{"event_id"}},
	{Table: "battle_reports", Name: "idx_battle_reports_source", Columns: []string{"source_type", "target_id", "created_at"}},
	{Table: "battle_reports", Name: "idx_battle_reports_type", Columns: []string{"source_type", "view_type", "battle_type", "created_at"}},
	{Table: "battle_reports", Name: "idx_battle_reports_player_view_visible", Columns: []string{"player_id", "view_type", "deleted_by_player", "created_at", "id"}},
	{Table: "battle_report_links", Name: "idx_battle_report_links_report", Columns: []string{"report_id"}},
	{Table: "battle_report_states", Name: "idx_battle_report_states_report_player", Columns: []string{"report_id", "player_id"}},
	{Table: "battle_report_states", Name: "idx_battle_report_states_cleanup_deleted", Columns: []string{"report_id", "is_deleted", "deleted_at"}},
	{Table: "battle_report_participants", Name: "idx_battle_report_participants_report", Columns: []string{"report_id"}},
	{Table: "battle_report_participants", Name: "idx_battle_report_participants_event", Columns: []string{"event_id"}},
	{Table: "battle_events", Name: "idx_battle_events_occurred", Columns: []string{"occurred_at"}},
}

// runEnsureReportCleanupIndexes 检查或创建战报清理专用索引。
func runEnsureReportCleanupIndexes(args []string) error {
	flags := flag.NewFlagSet("ensure-report-cleanup-indexes", flag.ContinueOnError)
	dsn := flags.String("dsn", "", "目标数据库 DSN，默认使用 HERO3_DATABASE_DSN")
	execute := flags.Bool("execute", false, "执行创建索引；未设置时只检查")
	allowNonTest := flags.Bool("allow-non-test", false, "允许在非 test_ 前缀数据库创建索引")
	if err := flags.Parse(args); err != nil {
		return err
	}
	resolvedDSN, databaseName, err := resolveReadonlyDbtoolDSN(*dsn)
	if err != nil {
		return err
	}
	if *execute && !strings.HasPrefix(databaseName, "test_") && !*allowNonTest {
		return fmt.Errorf("target database must use test_ prefix or pass --allow-non-test")
	}
	ctx, cancel := context.WithTimeout(context.Background(), longCommandTimeout)
	defer cancel()
	result, err := ensureReportCleanupIndexes(ctx, resolvedDSN, *execute)
	if err != nil {
		return err
	}
	result.DatabaseName = databaseName
	printReportCleanupIndexResult(result)
	if !*execute && len(result.Missing) > 0 {
		return fmt.Errorf("战报清理索引缺失 %d 个；确认低峰窗口后使用 --execute 创建", len(result.Missing))
	}
	return nil
}

// ensureReportCleanupIndexes 检查并按需创建战报清理索引。
func ensureReportCleanupIndexes(ctx context.Context, dsn string, execute bool) (reportCleanupIndexResult, error) {
	db, err := storage.OpenMySQL(ctx, dsn)
	if err != nil {
		return reportCleanupIndexResult{}, err
	}
	defer db.Close()

	databaseName, err := storage.MySQLDatabaseName(dsn)
	if err != nil {
		return reportCleanupIndexResult{}, err
	}
	result := reportCleanupIndexResult{DatabaseName: databaseName, DryRun: !execute}
	for _, definition := range reportCleanupIndexes {
		result.Checked++
		exists, err := hasIndexWithColumns(ctx, db, databaseName, definition)
		if err != nil {
			return result, err
		}
		if exists {
			continue
		}
		result.Missing = append(result.Missing, definition)
		if !execute {
			continue
		}
		if err := createReportCleanupIndex(ctx, db, definition); err != nil {
			return result, err
		}
		result.Created = append(result.Created, definition)
	}
	return result, nil
}

// hasIndexWithColumns 判断表中是否已有同名或同列顺序索引。
func hasIndexWithColumns(ctx context.Context, db *sql.DB, databaseName string, definition reportCleanupIndexDefinition) (bool, error) {
	rows, err := db.QueryContext(ctx, `SELECT INDEX_NAME, COLUMN_NAME
		FROM information_schema.STATISTICS
		WHERE TABLE_SCHEMA = ?
		  AND TABLE_NAME = ?
		ORDER BY INDEX_NAME, SEQ_IN_INDEX`, databaseName, definition.Table)
	if err != nil {
		return false, err
	}
	defer rows.Close()

	indexColumns := map[string][]string{}
	for rows.Next() {
		var indexName string
		var columnName string
		if err := rows.Scan(&indexName, &columnName); err != nil {
			return false, err
		}
		indexColumns[indexName] = append(indexColumns[indexName], columnName)
	}
	if err := rows.Err(); err != nil {
		return false, err
	}
	for indexName, columns := range indexColumns {
		if indexName == definition.Name || stringSliceHasPrefix(columns, definition.Columns) {
			return true, nil
		}
	}
	return false, nil
}

// createReportCleanupIndex 创建单个战报清理索引。
func createReportCleanupIndex(ctx context.Context, db *sql.DB, definition reportCleanupIndexDefinition) error {
	query := fmt.Sprintf(
		"CREATE INDEX %s ON %s (%s)",
		quoteIdentifier(definition.Name),
		quoteIdentifier(definition.Table),
		joinQuotedIdentifiers(definition.Columns),
	)
	_, err := db.ExecContext(ctx, query)
	return err
}

// printReportCleanupIndexResult 输出索引检查结果。
func printReportCleanupIndexResult(result reportCleanupIndexResult) {
	mode := "dry-run"
	if !result.DryRun {
		mode = "execute"
	}
	fmt.Printf("战报清理索引检查完成：数据库 %s，模式 %s，检查 %d，缺失 %d，已创建 %d\n",
		result.DatabaseName,
		mode,
		result.Checked,
		len(result.Missing),
		len(result.Created),
	)
	if len(result.Missing) > 0 {
		fmt.Println("缺失索引：")
		for _, definition := range result.Missing {
			fmt.Printf("  %s.%s (%s)\n", definition.Table, definition.Name, strings.Join(definition.Columns, ", "))
		}
	}
	if len(result.Created) > 0 {
		fmt.Println("已创建索引：")
		for _, definition := range result.Created {
			fmt.Printf("  %s.%s (%s)\n", definition.Table, definition.Name, strings.Join(definition.Columns, ", "))
		}
	}
}

// stringSliceHasPrefix 判断已有索引列是否覆盖所需列前缀。
func stringSliceHasPrefix(values []string, prefix []string) bool {
	if len(values) < len(prefix) {
		return false
	}
	for i := range prefix {
		if values[i] != prefix[i] {
			return false
		}
	}
	return true
}
