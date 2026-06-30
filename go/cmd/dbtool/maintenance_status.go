// 本文件提供线上维护巡检汇总命令，集中检查战报、索引和权威表健康。
package main

import (
	"context"
	"flag"
	"fmt"
	"time"

	"hero3/internal/infrastructure/storage"
)

type maintenanceStatusOptions struct {
	Days                  int
	Top                   int
	RetentionHours        int
	DeletedRetentionHours int
	Now                   time.Time
}

type maintenanceStatusResult struct {
	DatabaseName             string
	Stats                    battleReportStats
	Indexes                  reportCleanupIndexResult
	Authority                authorityHealthcheckResult
	CleanupCandidates        int
	SkippedCleanupCandidates bool
	MissingAuthorityProblems int
	Healthy                  bool
}

// runMaintenanceStatus 输出战报清理和权威表健康汇总。
func runMaintenanceStatus(args []string) error {
	flags := flag.NewFlagSet("maintenance-status", flag.ContinueOnError)
	dsn := flags.String("dsn", "", "目标数据库 DSN，默认使用 HERO3_DATABASE_DSN")
	days := flags.Int("days", 3, "统计最近多少天的战报")
	top := flags.Int("top", 5, "输出战报最多的玩家数量")
	retentionHours := flags.Int("retention-hours", 72, "普通战报物理保留小时数")
	deletedRetentionHours := flags.Int("deleted-retention-hours", 24, "玩家已删除战报物理保留小时数")
	if err := flags.Parse(args); err != nil {
		return err
	}
	resolvedDSN, databaseName, err := resolveReadonlyDbtoolDSN(*dsn)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), longCommandTimeout)
	defer cancel()
	result, err := collectMaintenanceStatus(ctx, resolvedDSN, maintenanceStatusOptions{
		Days:                  *days,
		Top:                   *top,
		RetentionHours:        *retentionHours,
		DeletedRetentionHours: *deletedRetentionHours,
		Now:                   time.Now().UTC(),
	})
	if err != nil {
		return err
	}
	result.DatabaseName = databaseName
	printMaintenanceStatus(result)
	if !result.Healthy {
		return fmt.Errorf("维护巡检发现异常：缺失清理索引 %d，权威表问题 %d", len(result.Indexes.Missing), result.MissingAuthorityProblems)
	}
	return nil
}

// collectMaintenanceStatus 汇总只读巡检数据。
func collectMaintenanceStatus(ctx context.Context, dsn string, options maintenanceStatusOptions) (maintenanceStatusResult, error) {
	if options.Days <= 0 {
		return maintenanceStatusResult{}, fmt.Errorf("days must be positive")
	}
	if options.Top <= 0 {
		options.Top = 5
	}
	if options.RetentionHours <= 0 {
		return maintenanceStatusResult{}, fmt.Errorf("retention-hours must be positive")
	}
	if options.DeletedRetentionHours <= 0 {
		return maintenanceStatusResult{}, fmt.Errorf("deleted-retention-hours must be positive")
	}
	if options.Now.IsZero() {
		options.Now = time.Now().UTC()
	}
	options.Now = options.Now.UTC()

	stats, err := collectBattleReportStats(ctx, dsn, battleReportStatsOptions{Days: options.Days, Top: options.Top, Now: options.Now})
	if err != nil {
		return maintenanceStatusResult{}, err
	}
	indexes, err := ensureReportCleanupIndexes(ctx, dsn, false)
	if err != nil {
		return maintenanceStatusResult{}, err
	}
	authority, err := healthcheckAuthority(ctx, dsn)
	if err != nil {
		return maintenanceStatusResult{}, err
	}

	result := maintenanceStatusResult{
		Stats:                    stats,
		Indexes:                  indexes,
		Authority:                authority,
		MissingAuthorityProblems: countAuthorityHealthcheckProblems(authority),
	}
	if len(indexes.Missing) > 0 {
		result.SkippedCleanupCandidates = true
	} else {
		candidates, err := countMaintenanceCleanupCandidates(ctx, dsn, options)
		if err != nil {
			return maintenanceStatusResult{}, err
		}
		result.CleanupCandidates = candidates
	}
	result.Healthy = len(result.Indexes.Missing) == 0 && result.MissingAuthorityProblems == 0
	return result, nil
}

// countMaintenanceCleanupCandidates 在索引完整时统计可清理候选量。
func countMaintenanceCleanupCandidates(ctx context.Context, dsn string, options maintenanceStatusOptions) (int, error) {
	db, err := storage.OpenMySQL(ctx, dsn)
	if err != nil {
		return 0, err
	}
	defer db.Close()
	return countBattleReportCleanupCandidates(ctx, db, battleReportCleanupOptions{
		RetentionHours:        options.RetentionHours,
		DeletedRetentionHours: options.DeletedRetentionHours,
		BatchSize:             1,
		MaxBatches:            1,
		DryRun:                true,
		Now:                   options.Now,
	})
}

// countAuthorityHealthcheckProblems 汇总权威表健康问题数量。
func countAuthorityHealthcheckProblems(result authorityHealthcheckResult) int {
	return result.MissingResources +
		result.MissingBuildings +
		result.MissingResourceSlots +
		result.MissingGenerals +
		result.MissingCurrencies +
		result.MissingLegacyNpc +
		result.BigSnapshotPlayers
}

// printMaintenanceStatus 输出维护巡检摘要。
func printMaintenanceStatus(result maintenanceStatusResult) {
	fmt.Printf("数据库: %s\n", result.DatabaseName)
	fmt.Printf("健康状态: %s\n", maintenanceStatusLabel(result.Healthy))
	fmt.Println("")
	fmt.Println("战报:")
	fmt.Printf("  总数=%d 最近%d天=%d 软删除=%d 有效分享=%d\n",
		result.Stats.Total,
		result.Stats.Days,
		result.Stats.Recent,
		result.Stats.Deleted,
		result.Stats.ActiveShares,
	)
	if result.SkippedCleanupCandidates {
		fmt.Println("  可清理候选=跳过（清理索引缺失）")
	} else {
		fmt.Printf("  可清理候选=%d\n", result.CleanupCandidates)
	}
	fmt.Println("战报清理索引:")
	fmt.Printf("  检查=%d 缺失=%d\n", result.Indexes.Checked, len(result.Indexes.Missing))
	for _, definition := range result.Indexes.Missing {
		fmt.Printf("  缺失 %s.%s (%s)\n", definition.Table, definition.Name, joinDisplayColumns(definition.Columns))
	}
	fmt.Println("权威表:")
	fmt.Printf("  玩家=%d 缺资源=%d 缺建筑=%d 缺资源田=%d 缺武将=%d 缺货币=%d 旧NPC缺权威=%d state_json残留=%d\n",
		result.Authority.Players,
		result.Authority.MissingResources,
		result.Authority.MissingBuildings,
		result.Authority.MissingResourceSlots,
		result.Authority.MissingGenerals,
		result.Authority.MissingCurrencies,
		result.Authority.MissingLegacyNpc,
		result.Authority.BigSnapshotPlayers,
	)
}

// maintenanceStatusLabel 返回巡检状态文案。
func maintenanceStatusLabel(healthy bool) string {
	if healthy {
		return "OK"
	}
	return "FAILED"
}

// joinDisplayColumns 拼接用于展示的列名。
func joinDisplayColumns(columns []string) string {
	result := ""
	for i, column := range columns {
		if i > 0 {
			result += ", "
		}
		result += column
	}
	return result
}
