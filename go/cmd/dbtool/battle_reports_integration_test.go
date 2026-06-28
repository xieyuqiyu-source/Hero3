// 本文件验证战报 dbtool 校验、修复和 V2 回填工具在本地 test_ MySQL 库上的行为。
package main

import (
	"context"
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
