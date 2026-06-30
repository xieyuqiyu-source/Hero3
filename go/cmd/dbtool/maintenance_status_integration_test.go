// 本文件验证维护巡检命令能汇总战报清理和权威表健康状态。
package main

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"hero3/internal/infrastructure/storage"
)

// TestMaintenanceStatusReportsHealthyDatabase 验证新迁移库能通过维护巡检。
func TestMaintenanceStatusReportsHealthyDatabase(t *testing.T) {
	dsn, cleanup := createMaintenanceStatusTestDatabase(t, "maintenance_ok")
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	result, err := collectMaintenanceStatus(ctx, dsn, maintenanceStatusOptions{
		Days:                  3,
		Top:                   5,
		RetentionHours:        72,
		DeletedRetentionHours: 24,
		Now:                   time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("collect maintenance status: %v", err)
	}
	if !result.Healthy || len(result.Indexes.Missing) != 0 || result.MissingAuthorityProblems != 0 || result.SkippedCleanupCandidates {
		t.Fatalf("expected healthy maintenance status, got %+v", result)
	}
}

// TestMaintenanceStatusSkipsCleanupCandidatesWhenIndexesMissing 验证缺索引时不会统计清理候选量。
func TestMaintenanceStatusSkipsCleanupCandidatesWhenIndexesMissing(t *testing.T) {
	dsn, cleanup := createMaintenanceStatusTestDatabase(t, "maintenance_missing_index")
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	db, err := storage.OpenMySQL(ctx, dsn)
	if err != nil {
		t.Fatalf("open mysql: %v", err)
	}
	defer db.Close()
	if _, err := db.ExecContext(ctx, `DROP INDEX idx_battle_report_links_report ON battle_report_links`); err != nil {
		t.Fatalf("drop cleanup index: %v", err)
	}

	result, err := collectMaintenanceStatus(ctx, dsn, maintenanceStatusOptions{
		Days:                  3,
		Top:                   5,
		RetentionHours:        72,
		DeletedRetentionHours: 24,
		Now:                   time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("collect maintenance status with missing index: %v", err)
	}
	if result.Healthy || len(result.Indexes.Missing) != 1 || !result.SkippedCleanupCandidates {
		t.Fatalf("expected unhealthy status with skipped cleanup candidates, got %+v", result)
	}
}

// createMaintenanceStatusTestDatabase 创建维护巡检测试用隔离库。
func createMaintenanceStatusTestDatabase(t *testing.T, prefix string) (string, func()) {
	t.Helper()
	baseDSN := dbtoolTestDSN(t)
	ctx := context.Background()
	baseName, err := storage.MySQLDatabaseName(baseDSN)
	if err != nil {
		t.Fatalf("parse base dsn: %v", err)
	}
	tempName := baseName + "_" + prefix + "_" + strings.NewReplacer(".", "_").Replace(time.Now().UTC().Format("20060102150405.000000000"))
	if err := storage.CreateMySQLDatabaseFromDSN(ctx, baseDSN, tempName); err != nil {
		t.Skipf("skip isolated maintenance status test, cannot create temp database: %v", err)
	}
	dsn, err := storage.MySQLDSNWithDatabase(baseDSN, tempName)
	if err != nil {
		t.Fatalf("build temp dsn: %v", err)
	}
	db, err := storage.OpenMySQL(ctx, dsn)
	if err != nil {
		t.Fatalf("open mysql: %v", err)
	}
	if err := storage.MigrateMySQL(ctx, db); err != nil {
		_ = db.Close()
		t.Fatalf("migrate mysql: %v", err)
	}
	return dsn, func() {
		_, _ = db.ExecContext(context.Background(), fmt.Sprintf("DROP DATABASE IF EXISTS `%s`", tempName))
		_ = db.Close()
	}
}
