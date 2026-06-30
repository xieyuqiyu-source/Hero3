// 本文件验证玩家货币和 NPC 状态新权威表回填工具的行为。
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

// TestBackfillPlayerAuxiliaryTablesRepairsMissingRows 验证旧快照可以补齐新拆出的玩家辅助权威表。
func TestBackfillPlayerAuxiliaryTablesRepairsMissingRows(t *testing.T) {
	baseDSN := dbtoolTestDSN(t)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	baseName, err := storage.MySQLDatabaseName(baseDSN)
	if err != nil {
		t.Fatalf("parse base dsn: %v", err)
	}
	tempName := baseName + "_auxiliary_" + strings.NewReplacer(".", "_").Replace(time.Now().UTC().Format("20060102150405.000000000"))
	if err := storage.CreateMySQLDatabaseFromDSN(ctx, baseDSN, tempName); err != nil {
		t.Skipf("skip isolated auxiliary backfill test, cannot create temp database: %v", err)
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
	accountID := "auxiliary_account_" + suffix
	playerID := "auxiliary_player_" + suffix

	if _, err := db.ExecContext(ctx,
		`INSERT INTO accounts (id, username, password_hash, gold, created_at) VALUES (?, ?, ?, 0, ?)`,
		accountID,
		"auxiliary_user_"+suffix,
		strings.Repeat("c", 64),
		now,
	); err != nil {
		t.Fatalf("insert account: %v", err)
	}
	state := game.GameState{
		Player:         game.Player{ID: playerID, Nickname: "辅助回填", Faction: "wei"},
		CityGold:       88,
		LastExchangeAt: now.Add(-time.Hour).UTC().Format(time.RFC3339),
		NpcState: &game.NpcState{
			LastRefreshedAt: now.Add(-2 * time.Hour).UTC().Format(time.RFC3339),
			Cities: []game.NpcCity{
				{
					ID:                "npc-aux-1",
					Name:              "辅助测试城",
					Faction:           "yellow_turban",
					Tier:              "t1",
					Resources:         map[string]int{"wood": 100},
					StorageCapacity:   map[string]int{"wood": 500},
					ProductionPerHour: map[string]int{"wood": 10},
					Army:              []game.ArmyUnit{{UnitType: "infantry_wei", Amount: 10}},
					MaxArmy:           []game.ArmyUnit{{UnitType: "infantry_wei", Amount: 10}},
					GeneratedAt:       now.UTC().Format(time.RFC3339),
				},
			},
		},
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

	currencyBefore, err := verifyPlayerCurrencies(ctx, dsn)
	if err != nil {
		t.Fatalf("verify currencies before: %v", err)
	}
	if currencyBefore.Missing == 0 {
		t.Fatalf("expected verify-currencies to find missing player_currencies rows")
	}
	npcBefore, err := verifyPlayerNpcStates(ctx, dsn)
	if err != nil {
		t.Fatalf("verify npc states before: %v", err)
	}
	if npcBefore.Missing == 0 {
		t.Fatalf("expected verify-npc-states to find missing player_npc_states rows")
	}

	if _, err := backfillPlayerCurrencies(ctx, dsn); err != nil {
		t.Fatalf("backfill currencies: %v", err)
	}
	if _, err := backfillPlayerNpcStates(ctx, dsn); err != nil {
		t.Fatalf("backfill npc states: %v", err)
	}

	currencyAfter, err := verifyPlayerCurrencies(ctx, dsn)
	if err != nil {
		t.Fatalf("verify currencies after: %v", err)
	}
	if currencyAfter.Missing != 0 || currencyAfter.ActualRows != 1 {
		t.Fatalf("expected currency backfill to repair rows, got %+v", currencyAfter)
	}
	npcAfter, err := verifyPlayerNpcStates(ctx, dsn)
	if err != nil {
		t.Fatalf("verify npc states after: %v", err)
	}
	if npcAfter.Missing != 0 || npcAfter.ActualRows != 1 {
		t.Fatalf("expected npc backfill to repair rows, got %+v", npcAfter)
	}
}

// TestBackfillPlayerAuxiliaryTablesHonorsBatchLimit 验证辅助权威表回填会按批次停住，方便线上低峰多次执行。
func TestBackfillPlayerAuxiliaryTablesHonorsBatchLimit(t *testing.T) {
	baseDSN := dbtoolTestDSN(t)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	baseName, err := storage.MySQLDatabaseName(baseDSN)
	if err != nil {
		t.Fatalf("parse base dsn: %v", err)
	}
	tempName := baseName + "_auxiliary_batch_" + strings.NewReplacer(".", "_").Replace(time.Now().UTC().Format("20060102150405.000000000"))
	if err := storage.CreateMySQLDatabaseFromDSN(ctx, baseDSN, tempName); err != nil {
		t.Skipf("skip isolated auxiliary batch test, cannot create temp database: %v", err)
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
	insertAuxiliaryBackfillPlayer(t, ctx, db, "batch_account_1", "batch_player_1", now)
	insertAuxiliaryBackfillPlayer(t, ctx, db, "batch_account_2", "batch_player_2", now)

	currencyResult, err := backfillPlayerCurrenciesWithOptions(ctx, dsn, auxiliaryBackfillOptions{BatchSize: 1, MaxBatches: 1})
	if err != nil {
		t.Fatalf("backfill currencies batch: %v", err)
	}
	if currencyResult.Rows != 1 || currencyResult.Batches != 1 || currencyResult.Remaining != 1 {
		t.Fatalf("expected one currency batch and one remaining row, got %+v", currencyResult)
	}
	npcResult, err := backfillPlayerNpcStatesWithOptions(ctx, dsn, auxiliaryBackfillOptions{BatchSize: 1, MaxBatches: 1})
	if err != nil {
		t.Fatalf("backfill npc states batch: %v", err)
	}
	if npcResult.Rows != 1 || npcResult.Batches != 1 || npcResult.Remaining != 1 {
		t.Fatalf("expected one npc batch and one remaining row, got %+v", npcResult)
	}
}

// insertAuxiliaryBackfillPlayer 插入带旧货币和 NPC 快照的测试玩家。
func insertAuxiliaryBackfillPlayer(t *testing.T, ctx context.Context, db storageSQLExecutor, accountID string, playerID string, now time.Time) {
	t.Helper()
	if _, err := db.ExecContext(ctx,
		`INSERT INTO accounts (id, username, password_hash, gold, created_at) VALUES (?, ?, ?, 0, ?)`,
		accountID,
		accountID,
		strings.Repeat("d", 64),
		now,
	); err != nil {
		t.Fatalf("insert account: %v", err)
	}
	state := game.GameState{
		Player:         game.Player{ID: playerID, Nickname: "批次回填", Faction: "wei"},
		CityGold:       66,
		LastExchangeAt: now.Add(-time.Hour).UTC().Format(time.RFC3339),
		NpcState: &game.NpcState{
			LastRefreshedAt: now.Add(-2 * time.Hour).UTC().Format(time.RFC3339),
			Cities: []game.NpcCity{{
				ID:                "npc-" + playerID,
				Name:              "批次测试城",
				Faction:           "yellow_turban",
				Tier:              "t1",
				Resources:         map[string]int{"wood": 100},
				StorageCapacity:   map[string]int{"wood": 500},
				ProductionPerHour: map[string]int{"wood": 10},
				Army:              []game.ArmyUnit{{UnitType: "infantry_wei", Amount: 10}},
				MaxArmy:           []game.ArmyUnit{{UnitType: "infantry_wei", Amount: 10}},
				GeneratedAt:       now.UTC().Format(time.RFC3339),
			}},
		},
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
}

type storageSQLExecutor interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
}
