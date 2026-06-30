// 本文件提供战报生命周期清理工具，按小批次物理删除过期战报和孤儿事件。
package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"strings"
	"time"

	"hero3/internal/infrastructure/storage"
)

type battleReportCleanupOptions struct {
	RetentionHours        int
	PvpRetentionHours     int
	DefenseRetentionHours int
	ScoutRetentionHours   int
	DeletedRetentionHours int
	BatchSize             int
	MaxBatches            int
	DryRun                bool
	AllowNonTest          bool
	Now                   time.Time
}

const (
	defaultBattleReportRetentionHours        = 72
	defaultPvpBattleReportRetentionHours     = 168
	defaultDefenseBattleReportRetentionHours = 168
	defaultScoutBattleReportRetentionHours   = 168
	defaultDeletedBattleReportRetentionHours = 24
)

type battleReportCleanupResult struct {
	DatabaseName        string
	DryRun              bool
	CandidateReports    int
	DeletedLinks        int64
	DeletedStates       int64
	DeletedParticipants int64
	DeletedReports      int64
	DeletedEvents       int64
	Batches             int
}

type cleanupBattleReportTarget struct {
	ReportID string
	EventID  string
}

// runCleanupBattleReports 分批清理过期战报，默认只 dry-run。
func runCleanupBattleReports(args []string) error {
	flags := flag.NewFlagSet("cleanup-battle-reports", flag.ContinueOnError)
	dsn := flags.String("dsn", "", "目标数据库 DSN，默认使用 HERO3_DATABASE_DSN")
	retentionHours := flags.Int("retention-hours", defaultBattleReportRetentionHours, "普通战报物理保留小时数")
	pvpRetentionHours := flags.Int("pvp-retention-hours", defaultPvpBattleReportRetentionHours, "PVP 战报物理保留小时数")
	defenseRetentionHours := flags.Int("defense-retention-hours", defaultDefenseBattleReportRetentionHours, "防守/被打战报物理保留小时数")
	scoutRetentionHours := flags.Int("scout-retention-hours", defaultScoutBattleReportRetentionHours, "侦查战报物理保留小时数")
	deletedRetentionHours := flags.Int("deleted-retention-hours", defaultDeletedBattleReportRetentionHours, "玩家已删除战报物理保留小时数")
	batchSize := flags.Int("batch-size", 500, "每批最多清理战报数量")
	maxBatches := flags.Int("max-batches", 1, "本次最多执行批次数")
	dryRun := flags.Bool("dry-run", true, "只统计不删除；正式执行必须使用 --execute")
	execute := flags.Bool("execute", false, "执行物理删除；未设置时保持 dry-run")
	allowNonTest := flags.Bool("allow-non-test", false, "允许清理非 test_ 前缀数据库")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if !*dryRun && !*execute {
		return fmt.Errorf("正式清理必须显式传入 --execute")
	}
	options := battleReportCleanupOptions{
		RetentionHours:        *retentionHours,
		PvpRetentionHours:     *pvpRetentionHours,
		DefenseRetentionHours: *defenseRetentionHours,
		ScoutRetentionHours:   *scoutRetentionHours,
		DeletedRetentionHours: *deletedRetentionHours,
		BatchSize:             *batchSize,
		MaxBatches:            *maxBatches,
		DryRun:                !*execute,
		AllowNonTest:          *allowNonTest,
		Now:                   time.Now().UTC(),
	}
	resolvedDSN, databaseName, err := resolveBattleReportCleanupDSN(*dsn, options)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), longCommandTimeout)
	defer cancel()
	result, err := cleanupBattleReports(ctx, resolvedDSN, options)
	if err != nil {
		return err
	}
	result.DatabaseName = databaseName
	printBattleReportCleanupResult(result)
	return nil
}

// cleanupBattleReports 按配置统计或物理清理战报。
func cleanupBattleReports(ctx context.Context, dsn string, options battleReportCleanupOptions) (battleReportCleanupResult, error) {
	if err := normalizeBattleReportCleanupOptions(&options); err != nil {
		return battleReportCleanupResult{}, err
	}
	db, err := storage.OpenMySQL(ctx, dsn)
	if err != nil {
		return battleReportCleanupResult{}, err
	}
	defer db.Close()

	result := battleReportCleanupResult{DryRun: options.DryRun}
	indexResult, err := ensureReportCleanupIndexes(ctx, dsn, false)
	if err != nil {
		return result, err
	}
	if len(indexResult.Missing) > 0 {
		return result, fmt.Errorf("战报清理索引缺失 %d 个，请先低峰执行 ensure-report-cleanup-indexes --execute", len(indexResult.Missing))
	}
	candidates, err := countBattleReportCleanupCandidates(ctx, db, options)
	if err != nil {
		return result, err
	}
	result.CandidateReports = candidates
	if options.DryRun || candidates == 0 {
		return result, nil
	}

	for result.Batches < options.MaxBatches {
		targets, err := selectBattleReportCleanupTargets(ctx, db, options)
		if err != nil {
			return result, err
		}
		if len(targets) == 0 {
			break
		}
		batch, err := deleteBattleReportCleanupBatch(ctx, db, targets)
		if err != nil {
			return result, err
		}
		result.DeletedLinks += batch.DeletedLinks
		result.DeletedStates += batch.DeletedStates
		result.DeletedParticipants += batch.DeletedParticipants
		result.DeletedReports += batch.DeletedReports
		result.DeletedEvents += batch.DeletedEvents
		result.Batches++
		if len(targets) < options.BatchSize {
			break
		}
	}
	return result, nil
}

// normalizeBattleReportCleanupOptions 归一化清理参数，避免误执行大范围删除。
func normalizeBattleReportCleanupOptions(options *battleReportCleanupOptions) error {
	if options.RetentionHours == 0 {
		options.RetentionHours = defaultBattleReportRetentionHours
	}
	if options.PvpRetentionHours == 0 {
		options.PvpRetentionHours = defaultPvpBattleReportRetentionHours
	}
	if options.DefenseRetentionHours == 0 {
		options.DefenseRetentionHours = defaultDefenseBattleReportRetentionHours
	}
	if options.ScoutRetentionHours == 0 {
		options.ScoutRetentionHours = defaultScoutBattleReportRetentionHours
	}
	if options.DeletedRetentionHours == 0 {
		options.DeletedRetentionHours = defaultDeletedBattleReportRetentionHours
	}
	if options.RetentionHours <= 0 {
		return fmt.Errorf("retention-hours must be positive")
	}
	if options.PvpRetentionHours <= 0 {
		return fmt.Errorf("pvp-retention-hours must be positive")
	}
	if options.DefenseRetentionHours <= 0 {
		return fmt.Errorf("defense-retention-hours must be positive")
	}
	if options.ScoutRetentionHours <= 0 {
		return fmt.Errorf("scout-retention-hours must be positive")
	}
	if options.DeletedRetentionHours <= 0 {
		return fmt.Errorf("deleted-retention-hours must be positive")
	}
	if options.BatchSize <= 0 || options.BatchSize > 2000 {
		return fmt.Errorf("batch-size must be between 1 and 2000")
	}
	if options.MaxBatches <= 0 || options.MaxBatches > 1000 {
		return fmt.Errorf("max-batches must be between 1 and 1000")
	}
	if options.Now.IsZero() {
		options.Now = time.Now().UTC()
	}
	options.Now = options.Now.UTC()
	return nil
}

// countBattleReportCleanupCandidates 统计当前规则会清理的战报数量。
func countBattleReportCleanupCandidates(ctx context.Context, db *sql.DB, options battleReportCleanupOptions) (int, error) {
	var count int
	query, args := buildBattleReportCleanupWhere(options)
	err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM battle_reports br WHERE `+query, args...).Scan(&count)
	return count, err
}

// selectBattleReportCleanupTargets 选择一批可清理战报。
func selectBattleReportCleanupTargets(ctx context.Context, db *sql.DB, options battleReportCleanupOptions) ([]cleanupBattleReportTarget, error) {
	query, args := buildBattleReportCleanupWhere(options)
	args = append(args, options.BatchSize)
	rows, err := db.QueryContext(ctx,
		`SELECT br.id, br.event_id
		 FROM battle_reports br
		 WHERE `+query+`
		 ORDER BY br.created_at ASC, br.id ASC
		 LIMIT ?`,
		args...,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	targets := []cleanupBattleReportTarget{}
	for rows.Next() {
		var target cleanupBattleReportTarget
		if err := rows.Scan(&target.ReportID, &target.EventID); err != nil {
			return nil, err
		}
		targets = append(targets, target)
	}
	return targets, rows.Err()
}

// buildBattleReportCleanupWhere 构造过期和软删战报的清理条件。
func buildBattleReportCleanupWhere(options battleReportCleanupOptions) (string, []any) {
	normalCutoff := options.Now.Add(-time.Duration(options.RetentionHours) * time.Hour)
	pvpCutoff := options.Now.Add(-time.Duration(options.PvpRetentionHours) * time.Hour)
	defenseCutoff := options.Now.Add(-time.Duration(options.DefenseRetentionHours) * time.Hour)
	scoutCutoff := options.Now.Add(-time.Duration(options.ScoutRetentionHours) * time.Hour)
	deletedCutoff := options.Now.Add(-time.Duration(options.DeletedRetentionHours) * time.Hour)
	return `
		NOT EXISTS (
			SELECT 1
			FROM battle_report_links active_link
			WHERE active_link.report_id = br.id
			  AND (active_link.expires_at IS NULL OR active_link.expires_at > ?)
		)
		AND (
			(
				br.deleted_by_player = 1
				AND (
					br.created_at <= ?
					OR EXISTS (
						SELECT 1
						FROM battle_report_states deleted_state
						WHERE deleted_state.report_id = br.id
						  AND deleted_state.is_deleted = 1
						  AND deleted_state.deleted_at IS NOT NULL
						  AND deleted_state.deleted_at <= ?
					)
				)
			)
			OR (
				COALESCE(br.deleted_by_player, 0) = 0
				AND (
					(
						(COALESCE(br.source_type, '') = 'player_city' OR COALESCE(br.source_type, '') = 'pvp' OR COALESCE(br.battle_type, '') = 'pvp' OR COALESCE(br.type, '') = 'pvp')
						AND br.created_at <= ?
					)
					OR (
						(COALESCE(br.view_type, '') = 'defense' OR COALESCE(br.type, '') = 'defense')
						AND br.created_at <= ?
					)
					OR (
						(COALESCE(br.battle_type, '') = 'scout' OR COALESCE(br.type, '') = 'scout')
						AND br.created_at <= ?
					)
					OR (
						NOT (COALESCE(br.source_type, '') = 'player_city' OR COALESCE(br.source_type, '') = 'pvp' OR COALESCE(br.battle_type, '') = 'pvp' OR COALESCE(br.type, '') = 'pvp' OR COALESCE(br.view_type, '') = 'defense' OR COALESCE(br.type, '') = 'defense' OR COALESCE(br.battle_type, '') = 'scout' OR COALESCE(br.type, '') = 'scout')
						AND br.created_at <= ?
					)
				)
			)
		)`, []any{options.Now, deletedCutoff, deletedCutoff, pvpCutoff, defenseCutoff, scoutCutoff, normalCutoff}
}

// deleteBattleReportCleanupBatch 在一个短事务内删除一批战报及其附属数据。
func deleteBattleReportCleanupBatch(ctx context.Context, db *sql.DB, targets []cleanupBattleReportTarget) (battleReportCleanupResult, error) {
	result := battleReportCleanupResult{}
	if len(targets) == 0 {
		return result, nil
	}
	reportIDs := make([]string, 0, len(targets))
	eventIDs := make([]string, 0, len(targets))
	seenEvents := map[string]bool{}
	for _, target := range targets {
		reportIDs = append(reportIDs, target.ReportID)
		if strings.TrimSpace(target.EventID) != "" && !seenEvents[target.EventID] {
			eventIDs = append(eventIDs, target.EventID)
			seenEvents[target.EventID] = true
		}
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return result, err
	}
	defer tx.Rollback()

	affected, err := execDeleteInTx(ctx, tx, `DELETE FROM battle_report_links WHERE report_id IN (%s)`, reportIDs)
	if err != nil {
		return result, err
	}
	result.DeletedLinks += affected
	affected, err = execDeleteInTx(ctx, tx, `DELETE FROM battle_report_states WHERE report_id IN (%s)`, reportIDs)
	if err != nil {
		return result, err
	}
	result.DeletedStates += affected
	affected, err = execDeleteInTx(ctx, tx, `DELETE FROM battle_report_participants WHERE report_id IN (%s)`, reportIDs)
	if err != nil {
		return result, err
	}
	result.DeletedParticipants += affected
	affected, err = execDeleteInTx(ctx, tx, `DELETE FROM battle_reports WHERE id IN (%s)`, reportIDs)
	if err != nil {
		return result, err
	}
	result.DeletedReports += affected

	if len(eventIDs) > 0 {
		affected, err = execDeleteInTx(ctx, tx,
			`DELETE p
			 FROM battle_report_participants p
			 WHERE p.event_id IN (%s)
			   AND NOT EXISTS (
			   	SELECT 1 FROM battle_reports br WHERE br.event_id = p.event_id
			   )`,
			eventIDs,
		)
		if err != nil {
			return result, err
		}
		result.DeletedParticipants += affected
		affected, err = execDeleteInTx(ctx, tx,
			`DELETE be
			 FROM battle_events be
			 WHERE be.id IN (%s)
			   AND NOT EXISTS (
			   	SELECT 1 FROM battle_reports br WHERE br.event_id = be.id
			   )`,
			eventIDs,
		)
		if err != nil {
			return result, err
		}
		result.DeletedEvents += affected
	}
	return result, tx.Commit()
}

// execDeleteInTx 执行 IN 条件删除并返回影响行数。
func execDeleteInTx(ctx context.Context, tx *sql.Tx, statement string, values []string) (int64, error) {
	if len(values) == 0 {
		return 0, nil
	}
	placeholders := strings.TrimRight(strings.Repeat("?,", len(values)), ",")
	args := make([]any, 0, len(values))
	for _, value := range values {
		args = append(args, value)
	}
	result, err := tx.ExecContext(ctx, fmt.Sprintf(statement, placeholders), args...)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

// resolveBattleReportCleanupDSN 解析清理命令 DSN，并默认限制 test_ 库或 dry-run。
func resolveBattleReportCleanupDSN(input string, options battleReportCleanupOptions) (string, string, error) {
	dsn, err := resolveBattleReportDSN(input)
	if err != nil {
		return "", "", err
	}
	databaseName, err := storage.MySQLDatabaseName(dsn)
	if err != nil {
		return "", "", err
	}
	if !options.DryRun && !strings.HasPrefix(databaseName, "test_") && !options.AllowNonTest {
		return "", "", fmt.Errorf("target database must use test_ prefix or pass --allow-non-test")
	}
	return dsn, databaseName, nil
}

// printBattleReportCleanupResult 输出清理统计。
func printBattleReportCleanupResult(result battleReportCleanupResult) {
	mode := "dry-run"
	if !result.DryRun {
		mode = "execute"
	}
	fmt.Printf("战报清理完成：数据库 %s，模式 %s，候选战报 %d，批次 %d\n",
		result.DatabaseName,
		mode,
		result.CandidateReports,
		result.Batches,
	)
	if result.DryRun {
		return
	}
	fmt.Printf("删除：links=%d states=%d participants=%d reports=%d events=%d\n",
		result.DeletedLinks,
		result.DeletedStates,
		result.DeletedParticipants,
		result.DeletedReports,
		result.DeletedEvents,
	)
}
