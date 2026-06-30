// 本文件验证战报清理专用索引检查和创建工具。
package main

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"hero3/internal/infrastructure/storage"
)

// TestEnsureReportCleanupIndexesRepairsMissingIndex 验证战报清理索引工具能发现并补回缺失索引。
func TestEnsureReportCleanupIndexesRepairsMissingIndex(t *testing.T) {
	baseDSN := dbtoolTestDSN(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	baseName, err := storage.MySQLDatabaseName(baseDSN)
	if err != nil {
		t.Fatalf("parse base dsn: %v", err)
	}
	tempName := baseName + "_report_indexes_" + strings.NewReplacer(".", "_").Replace(time.Now().UTC().Format("20060102150405.000000000"))
	if err := storage.CreateMySQLDatabaseFromDSN(ctx, baseDSN, tempName); err != nil {
		t.Skipf("skip isolated report index test, cannot create temp database: %v", err)
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

	initial, err := ensureReportCleanupIndexes(ctx, dsn, false)
	if err != nil {
		t.Fatalf("check report cleanup indexes: %v", err)
	}
	if len(initial.Missing) != 0 {
		t.Fatalf("expected migrated database to have cleanup indexes, missing=%+v", initial.Missing)
	}

	if _, err := db.ExecContext(ctx, `DROP INDEX idx_battle_report_links_report ON battle_report_links`); err != nil {
		t.Fatalf("drop test index: %v", err)
	}
	if _, err := db.ExecContext(ctx, `DROP INDEX idx_battle_report_participants_event ON battle_report_participants`); err != nil {
		t.Fatalf("drop participant event index: %v", err)
	}
	if _, err := db.ExecContext(ctx, `DROP INDEX idx_battle_reports_type ON battle_reports`); err != nil {
		t.Fatalf("drop battle reports type index: %v", err)
	}
	if _, err := db.ExecContext(ctx, `DROP INDEX idx_battle_reports_player_view_visible ON battle_reports`); err != nil {
		t.Fatalf("drop battle reports player view visible index: %v", err)
	}
	if _, err := db.ExecContext(ctx, `DROP INDEX idx_battle_events_occurred ON battle_events`); err != nil {
		t.Fatalf("drop battle events occurred index: %v", err)
	}
	missing, err := ensureReportCleanupIndexes(ctx, dsn, false)
	if err != nil {
		t.Fatalf("recheck missing report cleanup indexes: %v", err)
	}
	if !cleanupIndexMissing(missing.Missing, "idx_battle_report_links_report") || !cleanupIndexMissing(missing.Missing, "idx_battle_report_participants_event") || !cleanupIndexMissing(missing.Missing, "idx_battle_reports_type") || !cleanupIndexMissing(missing.Missing, "idx_battle_reports_player_view_visible") || !cleanupIndexMissing(missing.Missing, "idx_battle_events_occurred") {
		t.Fatalf("expected missing report link, participant event, battle report type, visible cap and battle event occurred indexes, got %+v", missing.Missing)
	}
	repaired, err := ensureReportCleanupIndexes(ctx, dsn, true)
	if err != nil {
		t.Fatalf("repair report cleanup indexes: %v", err)
	}
	if !cleanupIndexMissing(repaired.Created, "idx_battle_report_links_report") || !cleanupIndexMissing(repaired.Created, "idx_battle_report_participants_event") || !cleanupIndexMissing(repaired.Created, "idx_battle_reports_type") || !cleanupIndexMissing(repaired.Created, "idx_battle_reports_player_view_visible") || !cleanupIndexMissing(repaired.Created, "idx_battle_events_occurred") {
		t.Fatalf("expected created report link, participant event, battle report type, visible cap and battle event occurred indexes, got %+v", repaired.Created)
	}
	final, err := ensureReportCleanupIndexes(ctx, dsn, false)
	if err != nil {
		t.Fatalf("final report cleanup index check: %v", err)
	}
	if len(final.Missing) != 0 {
		t.Fatalf("expected no missing indexes after repair, got %+v", final.Missing)
	}
}

// cleanupIndexMissing 判断索引检查结果里是否包含指定索引名。
func cleanupIndexMissing(definitions []reportCleanupIndexDefinition, name string) bool {
	for _, definition := range definitions {
		if definition.Name == name {
			return true
		}
	}
	return false
}
