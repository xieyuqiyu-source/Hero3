// 本文件验证战报 dbtool 校验、修复和 V2 回填工具在本地 test_ MySQL 库上的行为。
package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"hero3/internal/app/game"
	"hero3/internal/infrastructure/storage"
)

// TestBattleReportToolsBackfillAndVerify 验证旧战报能被回填为标准战报结构。
func TestBattleReportToolsBackfillAndVerify(t *testing.T) {
	baseDSN := dbtoolTestDSN(t)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	baseName, err := storage.MySQLDatabaseName(baseDSN)
	if err != nil {
		t.Fatalf("parse base dsn: %v", err)
	}
	tempName := baseName + "_report_" + strings.NewReplacer(".", "_").Replace(time.Now().UTC().Format("20060102150405.000000000"))
	if err := storage.CreateMySQLDatabaseFromDSN(ctx, baseDSN, tempName); err != nil {
		t.Skipf("skip isolated report dbtool integration test, cannot create temp database: %v", err)
	}
	dsn, err := storage.MySQLDSNWithDatabase(baseDSN, tempName)
	if err != nil {
		t.Fatalf("build temp dsn: %v", err)
	}
	db, err := storage.OpenMySQL(ctx, dsn)
	if err != nil {
		t.Fatalf("open mysql: %v", err)
	}
	t.Cleanup(func() {
		_, _ = db.ExecContext(context.Background(), fmt.Sprintf("DROP DATABASE IF EXISTS `%s`", tempName))
		_ = db.Close()
	})
	if err := storage.MigrateMySQL(ctx, db); err != nil {
		t.Fatalf("migrate mysql: %v", err)
	}

	now := time.Now().UTC()
	suffix := strings.NewReplacer(".", "_").Replace(now.Format("150405.000000"))
	reportID := "tool_br_" + suffix
	playerID := "tool_p_" + suffix
	report := game.BattleReport{
		ID:              reportID,
		PlayerID:        playerID,
		PlayerName:      "工具测试城",
		PlayerFaction:   "wei",
		TargetID:        "npc_tool_" + suffix,
		TargetName:      "工具测试营地",
		Type:            "attack",
		Result:          "attacker_victory",
		DispatchedUnits: map[string]int{"weiInfantry": 5},
		LostUnits:       map[string]int{"weiInfantry": 1},
		DefenderFaction: "shu",
		DefenderUnits:   map[string]int{"shuInfantry": 3},
		CreatedAt:       now.Format(time.RFC3339),
	}
	reportJSON, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("marshal report: %v", err)
	}
	if _, err := db.ExecContext(ctx,
		`INSERT INTO battle_reports (
			id, player_id, event_id, owner_player_id, view_type, source_type, battle_type, result,
			title, summary, target_type, target_id, target_name, detail_json,
			report_json, type, is_read, deleted_by_player, created_at
		) VALUES (?, ?, '', '', '', '', '', ?, '', '', '', ?, ?, NULL, ?, ?, 0, 0, ?)`,
		reportID,
		playerID,
		report.Result,
		report.TargetID,
		report.TargetName,
		reportJSON,
		report.Type,
		now,
	); err != nil {
		t.Fatalf("insert legacy report: %v", err)
	}

	before, err := verifyBattleReports(ctx, dsn)
	if err != nil {
		t.Fatalf("verify battle reports before: %v", err)
	}
	if before.MissingStandardFields == 0 {
		t.Fatalf("expected legacy report to miss standard fields")
	}
	if _, err := backfillBattleReportV2(ctx, dsn); err != nil {
		t.Fatalf("backfill battle report v2: %v", err)
	}
	afterReports, err := verifyBattleReports(ctx, dsn)
	if err != nil {
		t.Fatalf("verify battle reports after: %v", err)
	}
	if afterReports.MissingStandardFields != 0 || afterReports.MissingParticipants != 0 || afterReports.InvalidReportJSON != 0 || afterReports.InvalidDetailJSON != 0 {
		t.Fatalf("expected backfilled report to pass standard verify, got %+v", afterReports)
	}
	var participantCount int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM battle_report_participants WHERE report_id = ?`, reportID).Scan(&participantCount); err != nil {
		t.Fatalf("count battle report participants: %v", err)
	}
	if participantCount == 0 {
		t.Fatal("expected backfill to create participant snapshots")
	}
	afterEvents, err := verifyBattleEvents(ctx, dsn)
	if err != nil {
		t.Fatalf("verify battle events after: %v", err)
	}
	if afterEvents.ReportsMissingEvent != 0 || afterEvents.ReportsReferMissingEvent != 0 {
		t.Fatalf("expected battle event verify to pass, got %+v", afterEvents)
	}
	afterStates, err := verifyBattleReportStates(ctx, dsn)
	if err != nil {
		t.Fatalf("verify battle report states after: %v", err)
	}
	if afterStates.ReportsMissingState != 0 || afterStates.OrphanStates != 0 || afterStates.DuplicateStates != 0 {
		t.Fatalf("expected battle report state verify to pass, got %+v", afterStates)
	}
}

// TestCleanupBattleReportsProtectsActiveSharesAndDeletesExpiredData 验证战报清理会保护有效分享并清理过期战报附属数据。
func TestCleanupBattleReportsProtectsActiveSharesAndDeletesExpiredData(t *testing.T) {
	baseDSN := dbtoolTestDSN(t)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	baseName, err := storage.MySQLDatabaseName(baseDSN)
	if err != nil {
		t.Fatalf("parse base dsn: %v", err)
	}
	tempName := baseName + "_cleanup_report_" + strings.NewReplacer(".", "_").Replace(time.Now().UTC().Format("20060102150405.000000000"))
	if err := storage.CreateMySQLDatabaseFromDSN(ctx, baseDSN, tempName); err != nil {
		t.Skipf("skip isolated cleanup integration test, cannot create temp database: %v", err)
	}
	dsn, err := storage.MySQLDSNWithDatabase(baseDSN, tempName)
	if err != nil {
		t.Fatalf("build temp dsn: %v", err)
	}
	db, err := storage.OpenMySQL(ctx, dsn)
	if err != nil {
		t.Fatalf("open mysql: %v", err)
	}
	t.Cleanup(func() {
		_, _ = db.ExecContext(context.Background(), fmt.Sprintf("DROP DATABASE IF EXISTS `%s`", tempName))
		_ = db.Close()
	})
	if err := storage.MigrateMySQL(ctx, db); err != nil {
		t.Fatalf("migrate mysql: %v", err)
	}
	repo := storage.NewMySQLRepository(db)

	now := time.Date(2026, 6, 30, 8, 0, 0, 0, time.UTC)
	oldTime := now.Add(-96 * time.Hour)
	suffix := strings.NewReplacer(".", "_").Replace(time.Now().UTC().Format("150405.000000"))
	expiredReportID := "cleanup_expired_" + suffix
	activeShareReportID := "cleanup_active_share_" + suffix
	expiredShareReportID := "cleanup_expired_share_" + suffix
	deletedReportID := "cleanup_deleted_" + suffix
	freshReportID := "cleanup_fresh_" + suffix
	playerID := "cleanup_player_" + suffix

	reports := []game.BattleReport{
		cleanupTestReport(expiredReportID, playerID, oldTime),
		cleanupTestReport(activeShareReportID, playerID, oldTime),
		cleanupTestReport(expiredShareReportID, playerID, oldTime),
		cleanupTestReport(deletedReportID, playerID, now),
		cleanupTestReport(freshReportID, playerID, now),
	}
	if err := repo.SaveReports(reports); err != nil {
		t.Fatalf("save cleanup reports: %v", err)
	}
	if _, err := repo.CreateBattleReportShareLink(playerID, activeShareReportID, "public", time.Time{}); err != nil {
		t.Fatalf("create active share link: %v", err)
	}
	if _, err := repo.CreateBattleReportShareLink(playerID, expiredShareReportID, "public", now.Add(-time.Hour)); err != nil {
		t.Fatalf("create expired share link: %v", err)
	}
	if _, err := db.ExecContext(ctx,
		`UPDATE battle_reports SET deleted_by_player = 1 WHERE id = ?`,
		deletedReportID,
	); err != nil {
		t.Fatalf("mark deleted report: %v", err)
	}
	if _, err := db.ExecContext(ctx,
		`UPDATE battle_report_states SET is_deleted = 1, deleted_at = ?, updated_at = ? WHERE report_id = ?`,
		now.Add(-25*time.Hour),
		now.Add(-25*time.Hour),
		deletedReportID,
	); err != nil {
		t.Fatalf("mark deleted report state: %v", err)
	}

	options := battleReportCleanupOptions{
		RetentionHours:        72,
		DeletedRetentionHours: 24,
		BatchSize:             10,
		MaxBatches:            10,
		DryRun:                true,
		Now:                   now,
	}
	dryRun, err := cleanupBattleReports(ctx, dsn, options)
	if err != nil {
		t.Fatalf("dry-run cleanup battle reports: %v", err)
	}
	if dryRun.CandidateReports != 3 || dryRun.DeletedReports != 0 {
		t.Fatalf("expected dry-run to find 3 candidates and delete nothing, got %+v", dryRun)
	}

	options.DryRun = false
	result, err := cleanupBattleReports(ctx, dsn, options)
	if err != nil {
		t.Fatalf("execute cleanup battle reports: %v", err)
	}
	if result.DeletedReports != 3 || result.DeletedEvents != 3 {
		t.Fatalf("expected cleanup to delete 3 reports and 3 orphan events, got %+v", result)
	}
	for _, reportID := range []string{expiredReportID, expiredShareReportID, deletedReportID} {
		assertCleanupReportMissing(t, db, reportID)
	}
	for _, reportID := range []string{activeShareReportID, freshReportID} {
		assertCleanupReportExists(t, db, reportID)
	}
	assertCleanupReportLinkCount(t, db, activeShareReportID, 1)
	assertCleanupReportLinkCount(t, db, expiredShareReportID, 0)
}

// TestCleanupBattleReportsUsesTypedRetention 验证 PVP、防守和侦查战报按更长保留期清理。
func TestCleanupBattleReportsUsesTypedRetention(t *testing.T) {
	baseDSN := dbtoolTestDSN(t)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	baseName, err := storage.MySQLDatabaseName(baseDSN)
	if err != nil {
		t.Fatalf("parse base dsn: %v", err)
	}
	tempName := baseName + "_cleanup_retention_" + strings.NewReplacer(".", "_").Replace(time.Now().UTC().Format("20060102150405.000000000"))
	if err := storage.CreateMySQLDatabaseFromDSN(ctx, baseDSN, tempName); err != nil {
		t.Skipf("skip isolated typed retention cleanup test, cannot create temp database: %v", err)
	}
	dsn, err := storage.MySQLDSNWithDatabase(baseDSN, tempName)
	if err != nil {
		t.Fatalf("build temp dsn: %v", err)
	}
	db, err := storage.OpenMySQL(ctx, dsn)
	if err != nil {
		t.Fatalf("open mysql: %v", err)
	}
	t.Cleanup(func() {
		_, _ = db.ExecContext(context.Background(), fmt.Sprintf("DROP DATABASE IF EXISTS `%s`", tempName))
		_ = db.Close()
	})
	if err := storage.MigrateMySQL(ctx, db); err != nil {
		t.Fatalf("migrate mysql: %v", err)
	}
	repo := storage.NewMySQLRepository(db)

	now := time.Date(2026, 6, 30, 8, 0, 0, 0, time.UTC)
	old96 := now.Add(-96 * time.Hour)
	old200 := now.Add(-200 * time.Hour)
	suffix := strings.NewReplacer(".", "_").Replace(time.Now().UTC().Format("150405.000000"))
	playerID := "cleanup_retention_player_" + suffix
	normalOldID := "cleanup_retention_npc_" + suffix
	pvpRecentID := "cleanup_retention_pvp_keep_" + suffix
	pvpVeryOldID := "cleanup_retention_pvp_delete_" + suffix
	defenseRecentID := "cleanup_retention_defense_" + suffix
	scoutRecentID := "cleanup_retention_scout_" + suffix
	deletedPvpID := "cleanup_retention_deleted_pvp_" + suffix

	normalOld := cleanupTestReport(normalOldID, playerID, old96)
	pvpRecent := cleanupTestReport(pvpRecentID, playerID, old96)
	pvpRecent.SourceType = game.ReportSourcePlayerCity
	pvpRecent.BattleType = "plunder"
	pvpVeryOld := cleanupTestReport(pvpVeryOldID, playerID, old200)
	pvpVeryOld.SourceType = game.ReportSourcePlayerCity
	pvpVeryOld.BattleType = "plunder"
	defenseRecent := cleanupTestReport(defenseRecentID, playerID, old96)
	defenseRecent.ViewType = game.ReportViewDefense
	defenseRecent.Type = "defense"
	defenseRecent.SourceType = game.ReportSourcePlayerCity
	scoutRecent := cleanupTestReport(scoutRecentID, playerID, old96)
	scoutRecent.Type = "scout"
	scoutRecent.BattleType = "scout"
	deletedPvp := cleanupTestReport(deletedPvpID, playerID, old96)
	deletedPvp.SourceType = game.ReportSourcePlayerCity
	deletedPvp.BattleType = "plunder"

	if err := repo.SaveReports([]game.BattleReport{normalOld, pvpRecent, pvpVeryOld, defenseRecent, scoutRecent, deletedPvp}); err != nil {
		t.Fatalf("save typed retention reports: %v", err)
	}
	if _, err := db.ExecContext(ctx, `UPDATE battle_reports SET deleted_by_player = 1 WHERE id = ?`, deletedPvpID); err != nil {
		t.Fatalf("mark typed retention deleted report: %v", err)
	}
	if _, err := db.ExecContext(ctx,
		`UPDATE battle_report_states SET is_deleted = 1, deleted_at = ?, updated_at = ? WHERE report_id = ?`,
		now.Add(-25*time.Hour),
		now.Add(-25*time.Hour),
		deletedPvpID,
	); err != nil {
		t.Fatalf("mark typed retention deleted state: %v", err)
	}

	options := battleReportCleanupOptions{
		RetentionHours:        72,
		PvpRetentionHours:     168,
		DefenseRetentionHours: 168,
		ScoutRetentionHours:   168,
		DeletedRetentionHours: 24,
		BatchSize:             10,
		MaxBatches:            10,
		DryRun:                true,
		Now:                   now,
	}
	dryRun, err := cleanupBattleReports(ctx, dsn, options)
	if err != nil {
		t.Fatalf("dry-run typed retention cleanup: %v", err)
	}
	if dryRun.CandidateReports != 3 {
		t.Fatalf("expected 3 typed retention candidates, got %+v", dryRun)
	}

	options.DryRun = false
	result, err := cleanupBattleReports(ctx, dsn, options)
	if err != nil {
		t.Fatalf("execute typed retention cleanup: %v", err)
	}
	if result.DeletedReports != 3 {
		t.Fatalf("expected cleanup to delete 3 typed retention reports, got %+v", result)
	}
	for _, reportID := range []string{normalOldID, pvpVeryOldID, deletedPvpID} {
		assertCleanupReportMissing(t, db, reportID)
	}
	for _, reportID := range []string{pvpRecentID, defenseRecentID, scoutRecentID} {
		assertCleanupReportExists(t, db, reportID)
	}
}

// TestCleanupBattleReportsRequiresCleanupIndexes 验证缺少清理索引时不会继续统计或删除战报。
func TestCleanupBattleReportsRequiresCleanupIndexes(t *testing.T) {
	baseDSN := dbtoolTestDSN(t)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	baseName, err := storage.MySQLDatabaseName(baseDSN)
	if err != nil {
		t.Fatalf("parse base dsn: %v", err)
	}
	tempName := baseName + "_cleanup_index_guard_" + strings.NewReplacer(".", "_").Replace(time.Now().UTC().Format("20060102150405.000000000"))
	if err := storage.CreateMySQLDatabaseFromDSN(ctx, baseDSN, tempName); err != nil {
		t.Skipf("skip isolated cleanup index guard test, cannot create temp database: %v", err)
	}
	dsn, err := storage.MySQLDSNWithDatabase(baseDSN, tempName)
	if err != nil {
		t.Fatalf("build temp dsn: %v", err)
	}
	db, err := storage.OpenMySQL(ctx, dsn)
	if err != nil {
		t.Fatalf("open mysql: %v", err)
	}
	t.Cleanup(func() {
		_, _ = db.ExecContext(context.Background(), fmt.Sprintf("DROP DATABASE IF EXISTS `%s`", tempName))
		_ = db.Close()
	})
	if err := storage.MigrateMySQL(ctx, db); err != nil {
		t.Fatalf("migrate mysql: %v", err)
	}
	if _, err := db.ExecContext(ctx, `DROP INDEX idx_battle_report_links_report ON battle_report_links`); err != nil {
		t.Fatalf("drop cleanup index: %v", err)
	}

	_, err = cleanupBattleReports(ctx, dsn, battleReportCleanupOptions{
		RetentionHours:        72,
		DeletedRetentionHours: 24,
		BatchSize:             10,
		MaxBatches:            1,
		DryRun:                true,
		Now:                   time.Now().UTC(),
	})
	if err == nil || !strings.Contains(err.Error(), "战报清理索引缺失") {
		t.Fatalf("expected cleanup to reject missing indexes, got %v", err)
	}
}

// TestBattleReportStatsCountsRecentReports 验证战报统计工具能输出总量、近期增长、类型和玩家 Top。
func TestBattleReportStatsCountsRecentReports(t *testing.T) {
	baseDSN := dbtoolTestDSN(t)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	baseName, err := storage.MySQLDatabaseName(baseDSN)
	if err != nil {
		t.Fatalf("parse base dsn: %v", err)
	}
	tempName := baseName + "_report_stats_" + strings.NewReplacer(".", "_").Replace(time.Now().UTC().Format("20060102150405.000000000"))
	if err := storage.CreateMySQLDatabaseFromDSN(ctx, baseDSN, tempName); err != nil {
		t.Skipf("skip isolated report stats integration test, cannot create temp database: %v", err)
	}
	dsn, err := storage.MySQLDSNWithDatabase(baseDSN, tempName)
	if err != nil {
		t.Fatalf("build temp dsn: %v", err)
	}
	db, err := storage.OpenMySQL(ctx, dsn)
	if err != nil {
		t.Fatalf("open mysql: %v", err)
	}
	t.Cleanup(func() {
		_, _ = db.ExecContext(context.Background(), fmt.Sprintf("DROP DATABASE IF EXISTS `%s`", tempName))
		_ = db.Close()
	})
	if err := storage.MigrateMySQL(ctx, db); err != nil {
		t.Fatalf("migrate mysql: %v", err)
	}
	repo := storage.NewMySQLRepository(db)

	now := time.Date(2026, 6, 30, 8, 0, 0, 0, time.UTC)
	suffix := strings.NewReplacer(".", "_").Replace(time.Now().UTC().Format("150405.000000"))
	p1 := "stats_player_1_" + suffix
	p2 := "stats_player_2_" + suffix
	attackReportID := "stats_attack_" + suffix
	deletedSweepID := "stats_deleted_sweep_" + suffix
	reports := []game.BattleReport{
		cleanupTestReport(attackReportID, p1, now),
		cleanupTestReport("stats_sweep_"+suffix, p1, now.Add(-24*time.Hour)),
		cleanupTestReport(deletedSweepID, p2, now.Add(-48*time.Hour)),
		cleanupTestReport("stats_old_"+suffix, p2, now.Add(-120*time.Hour)),
	}
	reports[1].BattleType = "sweep"
	reports[1].Type = "attack"
	reports[2].BattleType = "sweep"
	reports[2].Type = "attack"
	if err := repo.SaveReports(reports); err != nil {
		t.Fatalf("save stats reports: %v", err)
	}
	if _, err := repo.CreateBattleReportShareLink(p1, attackReportID, "public", time.Time{}); err != nil {
		t.Fatalf("create stats share link: %v", err)
	}
	if _, err := db.ExecContext(ctx, `UPDATE battle_reports SET deleted_by_player = 1 WHERE id = ?`, deletedSweepID); err != nil {
		t.Fatalf("mark stats report deleted: %v", err)
	}

	stats, err := collectBattleReportStats(ctx, dsn, battleReportStatsOptions{Days: 3, Top: 5, Now: now})
	if err != nil {
		t.Fatalf("collect battle report stats: %v", err)
	}
	if stats.Total != 4 || stats.Recent != 3 || stats.Deleted != 1 || stats.ActiveShares != 1 {
		t.Fatalf("unexpected stats summary: %+v", stats)
	}
	if len(stats.Daily) != 3 || stats.Daily[0].Day != "2026-06-30" || stats.Daily[0].Total != 1 {
		t.Fatalf("unexpected daily stats: %+v", stats.Daily)
	}
	if got := reportTypeTotal(stats.Types, "sweep"); got != 2 {
		t.Fatalf("expected sweep type total 2, got %d in %+v", got, stats.Types)
	}
	if len(stats.TopPlayers) == 0 || stats.TopPlayers[0].PlayerID != p1 || stats.TopPlayers[0].Total != 2 {
		t.Fatalf("unexpected top players: %+v", stats.TopPlayers)
	}
}

// TestLockSnapshotCollectsDatabaseSnapshot 验证锁快照工具能在测试库上只读采集。
func TestLockSnapshotCollectsDatabaseSnapshot(t *testing.T) {
	baseDSN := dbtoolTestDSN(t)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	snapshot, err := collectLockSnapshot(ctx, baseDSN, lockSnapshotOptions{Limit: 5})
	if err != nil {
		t.Fatalf("collect lock snapshot: %v", err)
	}
	if len(snapshot.Processes) == 0 && len(snapshot.Transactions) == 0 && len(snapshot.Waits) == 0 && len(snapshot.Warnings) == 0 {
		t.Fatal("expected lock snapshot to contain data or permission warnings")
	}
}

// TestCompactSQLTrimsWhitespaceAndLength 验证 SQL 输出会压缩空白并限制长度。
func TestCompactSQLTrimsWhitespaceAndLength(t *testing.T) {
	got := compactSQL(" SELECT  *\nFROM\tplayer_inventory   WHERE player_id = ? ")
	if got != "SELECT * FROM player_inventory WHERE player_id = ?" {
		t.Fatalf("unexpected compact sql: %q", got)
	}
	long := compactSQL(strings.Repeat("x", 260))
	if len(long) != 243 || !strings.HasSuffix(long, "...") {
		t.Fatalf("expected compact sql to truncate with suffix, got len=%d value=%q", len(long), long)
	}
}

// reportTypeTotal 返回指定战报类型的统计数量。
func reportTypeTotal(types []battleReportTypeStat, battleType string) int {
	for _, stat := range types {
		if stat.BattleType == battleType {
			return stat.Total
		}
	}
	return 0
}

// cleanupTestReport 构造清理测试所需的标准战报。
func cleanupTestReport(reportID string, playerID string, createdAt time.Time) game.BattleReport {
	return game.NormalizeBattleReport(game.BattleReport{
		ID:              reportID,
		EventID:         "event_" + reportID,
		PlayerID:        playerID,
		OwnerPlayerID:   playerID,
		PlayerName:      "清理测试城",
		PlayerFaction:   "wei",
		TargetID:        "npc_" + reportID,
		TargetName:      "清理测试营地",
		Type:            "attack",
		ViewType:        game.ReportViewAttack,
		SourceType:      game.ReportSourceNPCCity,
		BattleType:      "attack",
		Result:          "attacker_victory",
		DispatchedUnits: map[string]int{"weiInfantry": 10},
		LostUnits:       map[string]int{"weiInfantry": 1},
		DefenderFaction: "shu",
		DefenderUnits:   map[string]int{"shuInfantry": 3},
		CreatedAt:       createdAt.UTC().Format(time.RFC3339),
	})
}

// assertCleanupReportMissing 断言战报和附属状态已清理。
func assertCleanupReportMissing(t *testing.T, db *sql.DB, reportID string) {
	t.Helper()
	assertCleanupCount(t, db, `SELECT COUNT(*) FROM battle_reports WHERE id = ?`, reportID, 0)
	assertCleanupCount(t, db, `SELECT COUNT(*) FROM battle_report_states WHERE report_id = ?`, reportID, 0)
	assertCleanupCount(t, db, `SELECT COUNT(*) FROM battle_report_participants WHERE report_id = ?`, reportID, 0)
	assertCleanupCount(t, db, `SELECT COUNT(*) FROM battle_events WHERE id = ?`, "event_"+reportID, 0)
}

// assertCleanupReportExists 断言战报仍保留。
func assertCleanupReportExists(t *testing.T, db *sql.DB, reportID string) {
	t.Helper()
	assertCleanupCount(t, db, `SELECT COUNT(*) FROM battle_reports WHERE id = ?`, reportID, 1)
	assertCleanupCount(t, db, `SELECT COUNT(*) FROM battle_events WHERE id = ?`, "event_"+reportID, 1)
}

// assertCleanupReportLinkCount 断言分享链接数量。
func assertCleanupReportLinkCount(t *testing.T, db *sql.DB, reportID string, want int) {
	t.Helper()
	assertCleanupCount(t, db, `SELECT COUNT(*) FROM battle_report_links WHERE report_id = ?`, reportID, want)
}

// assertCleanupCount 断言单条 COUNT 查询结果。
func assertCleanupCount(t *testing.T, db *sql.DB, query string, arg string, want int) {
	t.Helper()
	var got int
	if err := db.QueryRow(query, arg).Scan(&got); err != nil {
		t.Fatalf("count cleanup rows: %v", err)
	}
	if got != want {
		t.Fatalf("expected count %d for %q arg %s, got %d", want, query, arg, got)
	}
}
