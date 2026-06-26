// 本文件验证资源回填和校验工具在本地 test_ MySQL 库上的行为。
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/joho/godotenv"

	"hero3/internal/app/game"
	"hero3/internal/infrastructure/storage"
)

// dbtoolTestDSN 读取本地测试 DSN，非 test_ 库直接跳过。
func dbtoolTestDSN(t *testing.T) string {
	t.Helper()
	_ = godotenv.Load("../../.env")
	dsn := strings.TrimSpace(os.Getenv("HERO3_DATABASE_DSN"))
	if dsn == "" {
		t.Skip("HERO3_DATABASE_DSN is not configured")
	}
	databaseName, err := storage.MySQLDatabaseName(dsn)
	if err != nil {
		t.Fatalf("parse mysql dsn: %v", err)
	}
	if !strings.HasPrefix(databaseName, "test_") {
		t.Skipf("skip dbtool integration test on non-test database %s", databaseName)
	}
	return dsn
}

// TestBackfillResourcesRepairsMissingRows 验证校验能发现缺失资源行，回填能修复旧数据。
func TestBackfillResourcesRepairsMissingRows(t *testing.T) {
	baseDSN := dbtoolTestDSN(t)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	baseName, err := storage.MySQLDatabaseName(baseDSN)
	if err != nil {
		t.Fatalf("parse base dsn: %v", err)
	}
	tempName := baseName + "_dbtool_" + strings.NewReplacer(".", "_").Replace(time.Now().UTC().Format("20060102150405.000000000"))
	if err := storage.CreateMySQLDatabaseFromDSN(ctx, baseDSN, tempName); err != nil {
		t.Skipf("skip isolated dbtool integration test, cannot create temp database: %v", err)
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
	suffix := strings.NewReplacer(".", "_").Replace(now.Format("20060102150405.000000000"))
	accountID := "dbtool_account_" + suffix
	playerID := "dbtool_player_" + suffix
	t.Cleanup(func() {
		_, _ = db.ExecContext(context.Background(), `DELETE FROM accounts WHERE id = ?`, accountID)
	})

	if _, err := db.ExecContext(ctx,
		`INSERT INTO accounts (id, username, password_hash, gold, created_at) VALUES (?, ?, ?, 0, ?)`,
		accountID,
		"dbtool_user_"+suffix,
		strings.Repeat("b", 64),
		now,
	); err != nil {
		t.Fatalf("insert account: %v", err)
	}
	state := game.GameState{
		Player: game.Player{ID: playerID, Nickname: "回填测试", Faction: "wei"},
		Resources: game.ResourceState{
			Items:    map[string]int{"wood": 321, "stone": 222},
			Capacity: map[string]int{"wood": 1000, "stone": 1000},
		},
		ResourceSettledAt: now.Format(time.RFC3339),
		ServerTime:        now.Format(time.RFC3339),
	}
	stateJSON, err := json.Marshal(state)
	if err != nil {
		t.Fatalf("marshal state: %v", err)
	}
	if _, err := db.ExecContext(ctx,
		`INSERT INTO players (id, account_id, nickname, faction, mail_code, state_json, created_at, updated_at)
		 VALUES (?, ?, ?, ?, '', ?, ?, ?)`,
		playerID,
		accountID,
		state.Player.Nickname,
		state.Player.Faction,
		stateJSON,
		now,
		now,
	); err != nil {
		t.Fatalf("insert player: %v", err)
	}

	verifyBefore, err := verifyPlayerResources(ctx, dsn)
	if err != nil {
		t.Fatalf("verify before backfill: %v", err)
	}
	if verifyBefore.Mismatches == 0 {
		t.Fatalf("expected verify-resources to find missing player_resources rows")
	}

	if _, err := backfillPlayerResources(ctx, dsn); err != nil {
		t.Fatalf("backfill resources: %v", err)
	}
	verifyAfter, err := verifyPlayerResources(ctx, dsn)
	if err != nil {
		t.Fatalf("verify after backfill: %v", err)
	}
	if verifyAfter.Mismatches != 0 {
		t.Fatalf("expected backfill to repair resources, got mismatches=%d", verifyAfter.Mismatches)
	}
}
