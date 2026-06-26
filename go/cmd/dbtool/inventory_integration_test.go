// 本文件验证背包回填和校验工具在隔离 test_ MySQL 库上的行为。
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

// openIsolatedDBToolDatabase 创建隔离测试库，避免回填工具影响当前 test_hero3。
func openIsolatedDBToolDatabase(t *testing.T, prefix string) (string, *sql.DB) {
	t.Helper()
	baseDSN := dbtoolTestDSN(t)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	baseName, err := storage.MySQLDatabaseName(baseDSN)
	if err != nil {
		t.Fatalf("parse base dsn: %v", err)
	}
	tempName := baseName + "_" + prefix + "_" + strings.NewReplacer(".", "_").Replace(time.Now().UTC().Format("20060102150405.000000000"))
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
	return dsn, db
}

// insertDBToolPlayerState 插入一条旧格式玩家状态。
func insertDBToolPlayerState(t *testing.T, db *sql.DB, accountID string, playerID string, state game.GameState) {
	t.Helper()
	now := time.Now().UTC()
	if _, err := db.Exec(
		`INSERT INTO accounts (id, username, password_hash, gold, created_at) VALUES (?, ?, ?, 0, ?)`,
		accountID,
		"dbtool_user_"+playerID,
		strings.Repeat("c", 64),
		now,
	); err != nil {
		t.Fatalf("insert account: %v", err)
	}
	stateJSON, err := json.Marshal(state)
	if err != nil {
		t.Fatalf("marshal state: %v", err)
	}
	if _, err := db.Exec(
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
}

// TestBackfillInventoryRepairsMissingRows 验证校验能发现缺失背包行，回填能修复旧数据。
func TestBackfillInventoryRepairsMissingRows(t *testing.T) {
	dsn, db := openIsolatedDBToolDatabase(t, "inventory")
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	now := time.Now().UTC().Format(time.RFC3339)
	insertDBToolPlayerState(t, db, "dbtool_inventory_account", "dbtool_inventory_player", game.GameState{
		Player: game.Player{ID: "dbtool_inventory_player", Nickname: "背包回填测试", Faction: "wei"},
		Resources: game.ResourceState{
			Items:    map[string]int{"wood": 100},
			Capacity: map[string]int{"wood": 1000},
		},
		Inventory: map[string]game.ItemStack{
			"resource_pack_small": {ItemID: "resource_pack_small", Amount: 2, ObtainedAt: now, UpdatedAt: now},
		},
		ResourceSettledAt: now,
		ServerTime:        now,
	})

	verifyBefore, err := verifyPlayerInventory(ctx, dsn)
	if err != nil {
		t.Fatalf("verify before backfill: %v", err)
	}
	if verifyBefore.Mismatches == 0 {
		t.Fatalf("expected verify-inventory to find missing player_inventory rows")
	}
	if _, err := backfillPlayerInventory(ctx, dsn); err != nil {
		t.Fatalf("backfill inventory: %v", err)
	}
	verifyAfter, err := verifyPlayerInventory(ctx, dsn)
	if err != nil {
		t.Fatalf("verify after backfill: %v", err)
	}
	if verifyAfter.Mismatches != 0 {
		t.Fatalf("expected backfill to repair inventory, got mismatches=%d", verifyAfter.Mismatches)
	}
}
