// 本文件提供战报增长量和玩家分布的只读观测工具。
package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"time"

	"hero3/internal/infrastructure/storage"
)

type battleReportStatsOptions struct {
	Days int
	Top  int
	Now  time.Time
}

type battleReportStats struct {
	DatabaseName string
	Days         int
	Total        int
	Recent       int
	Deleted      int
	ActiveShares int
	Daily        []battleReportDailyStat
	Types        []battleReportTypeStat
	TopPlayers   []battleReportPlayerStat
}

type battleReportDailyStat struct {
	Day     string
	Total   int
	Deleted int
	Sweep   int
}

type battleReportTypeStat struct {
	BattleType string
	Total      int
	Deleted    int
}

type battleReportPlayerStat struct {
	PlayerID string
	Total    int
	Deleted  int
	Sweep    int
}

// runBattleReportStats 输出战报增长和玩家分布统计。
func runBattleReportStats(args []string) error {
	flags := flag.NewFlagSet("report-stats", flag.ContinueOnError)
	dsn := flags.String("dsn", "", "目标数据库 DSN，默认使用 HERO3_DATABASE_DSN")
	days := flags.Int("days", 3, "统计最近多少天的战报")
	top := flags.Int("top", 10, "输出战报最多的玩家数量")
	if err := flags.Parse(args); err != nil {
		return err
	}
	options := battleReportStatsOptions{Days: *days, Top: *top, Now: time.Now().UTC()}
	resolvedDSN, databaseName, err := resolveBattleReportStatsDSN(*dsn)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), longCommandTimeout)
	defer cancel()
	stats, err := collectBattleReportStats(ctx, resolvedDSN, options)
	if err != nil {
		return err
	}
	stats.DatabaseName = databaseName
	printBattleReportStats(stats)
	return nil
}

// collectBattleReportStats 从数据库读取战报增长统计。
func collectBattleReportStats(ctx context.Context, dsn string, options battleReportStatsOptions) (battleReportStats, error) {
	if options.Days <= 0 {
		return battleReportStats{}, fmt.Errorf("days must be positive")
	}
	if options.Top <= 0 {
		options.Top = 10
	}
	if options.Top > 100 {
		options.Top = 100
	}
	if options.Now.IsZero() {
		options.Now = time.Now().UTC()
	}
	options.Now = options.Now.UTC()
	db, err := storage.OpenMySQL(ctx, dsn)
	if err != nil {
		return battleReportStats{}, err
	}
	defer db.Close()

	cutoff := options.Now.Add(-time.Duration(options.Days) * 24 * time.Hour)
	stats := battleReportStats{Days: options.Days}
	if err := db.QueryRowContext(ctx,
		`SELECT
			COUNT(*),
			COALESCE(SUM(CASE WHEN created_at >= ? THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN deleted_by_player = 1 THEN 1 ELSE 0 END), 0)
		 FROM battle_reports`,
		cutoff,
	).Scan(&stats.Total, &stats.Recent, &stats.Deleted); err != nil {
		return battleReportStats{}, err
	}
	if err := db.QueryRowContext(ctx,
		`SELECT COUNT(DISTINCT br.id)
		 FROM battle_reports br
		 JOIN battle_report_links link ON link.report_id = br.id
		 WHERE link.expires_at IS NULL OR link.expires_at > ?`,
		options.Now,
	).Scan(&stats.ActiveShares); err != nil {
		return battleReportStats{}, err
	}
	daily, err := queryBattleReportDailyStats(ctx, db, cutoff)
	if err != nil {
		return battleReportStats{}, err
	}
	types, err := queryBattleReportTypeStats(ctx, db, cutoff)
	if err != nil {
		return battleReportStats{}, err
	}
	topPlayers, err := queryBattleReportTopPlayers(ctx, db, cutoff, options.Top)
	if err != nil {
		return battleReportStats{}, err
	}
	stats.Daily = daily
	stats.Types = types
	stats.TopPlayers = topPlayers
	return stats, nil
}

// queryBattleReportDailyStats 统计每日战报量。
func queryBattleReportDailyStats(ctx context.Context, db *sql.DB, cutoff time.Time) ([]battleReportDailyStat, error) {
	rows, err := db.QueryContext(ctx,
		`SELECT
			DATE_FORMAT(created_at, '%Y-%m-%d') AS day,
			COUNT(*) AS total,
			COALESCE(SUM(CASE WHEN deleted_by_player = 1 THEN 1 ELSE 0 END), 0) AS deleted_count,
			COALESCE(SUM(CASE WHEN battle_type = 'sweep' THEN 1 ELSE 0 END), 0) AS sweep_count
		 FROM battle_reports
		 WHERE created_at >= ?
		 GROUP BY DATE(created_at), DATE_FORMAT(created_at, '%Y-%m-%d')
		 ORDER BY day DESC`,
		cutoff,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []battleReportDailyStat{}
	for rows.Next() {
		var stat battleReportDailyStat
		if err := rows.Scan(&stat.Day, &stat.Total, &stat.Deleted, &stat.Sweep); err != nil {
			return nil, err
		}
		result = append(result, stat)
	}
	return result, rows.Err()
}

// queryBattleReportTypeStats 统计各战报类型数量。
func queryBattleReportTypeStats(ctx context.Context, db *sql.DB, cutoff time.Time) ([]battleReportTypeStat, error) {
	rows, err := db.QueryContext(ctx,
		`SELECT
			COALESCE(NULLIF(battle_type, ''), NULLIF(type, ''), 'unknown') AS report_type,
			COUNT(*) AS total,
			COALESCE(SUM(CASE WHEN deleted_by_player = 1 THEN 1 ELSE 0 END), 0) AS deleted_count
		 FROM battle_reports
		 WHERE created_at >= ?
		 GROUP BY report_type
		 ORDER BY total DESC, report_type ASC`,
		cutoff,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []battleReportTypeStat{}
	for rows.Next() {
		var stat battleReportTypeStat
		if err := rows.Scan(&stat.BattleType, &stat.Total, &stat.Deleted); err != nil {
			return nil, err
		}
		result = append(result, stat)
	}
	return result, rows.Err()
}

// queryBattleReportTopPlayers 统计近期战报最多的玩家。
func queryBattleReportTopPlayers(ctx context.Context, db *sql.DB, cutoff time.Time, limit int) ([]battleReportPlayerStat, error) {
	rows, err := db.QueryContext(ctx,
		`SELECT
			player_id,
			COUNT(*) AS total,
			COALESCE(SUM(CASE WHEN deleted_by_player = 1 THEN 1 ELSE 0 END), 0) AS deleted_count,
			COALESCE(SUM(CASE WHEN battle_type = 'sweep' THEN 1 ELSE 0 END), 0) AS sweep_count
		 FROM battle_reports
		 WHERE created_at >= ?
		 GROUP BY player_id
		 ORDER BY total DESC, player_id ASC
		 LIMIT ?`,
		cutoff,
		limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []battleReportPlayerStat{}
	for rows.Next() {
		var stat battleReportPlayerStat
		if err := rows.Scan(&stat.PlayerID, &stat.Total, &stat.Deleted, &stat.Sweep); err != nil {
			return nil, err
		}
		result = append(result, stat)
	}
	return result, rows.Err()
}

// resolveBattleReportStatsDSN 解析战报统计 DSN。
func resolveBattleReportStatsDSN(rawDSN string) (string, string, error) {
	dsn, err := resolveBattleReportDSN(rawDSN)
	if err != nil {
		return "", "", err
	}
	databaseName, err := storage.MySQLDatabaseName(dsn)
	if err != nil {
		return "", "", err
	}
	return dsn, databaseName, nil
}

// printBattleReportStats 输出战报统计。
func printBattleReportStats(stats battleReportStats) {
	fmt.Printf("数据库: %s\n", stats.DatabaseName)
	fmt.Printf("统计窗口: 最近 %d 天\n", stats.Days)
	fmt.Printf("战报总数: %d\n", stats.Total)
	fmt.Printf("窗口内战报: %d\n", stats.Recent)
	fmt.Printf("软删除战报: %d\n", stats.Deleted)
	fmt.Printf("有效分享战报: %d\n", stats.ActiveShares)
	fmt.Println("")
	fmt.Println("每日增长:")
	for _, stat := range stats.Daily {
		fmt.Printf("  %s total=%d sweep=%d deleted=%d\n", stat.Day, stat.Total, stat.Sweep, stat.Deleted)
	}
	fmt.Println("")
	fmt.Println("战报类型:")
	for _, stat := range stats.Types {
		fmt.Printf("  %s total=%d deleted=%d\n", stat.BattleType, stat.Total, stat.Deleted)
	}
	fmt.Println("")
	fmt.Println("玩家 Top:")
	for _, stat := range stats.TopPlayers {
		fmt.Printf("  %s total=%d sweep=%d deleted=%d\n", stat.PlayerID, stat.Total, stat.Sweep, stat.Deleted)
	}
}
