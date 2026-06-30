// 本文件测试当前权威表健康检查命令的基础语义。
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

// TestHealthcheckAuthorityAcceptsEmptyOptionalTables 验证空背包、空兵力、空队列和空 Buff 不会被判为异常。
func TestHealthcheckAuthorityAcceptsEmptyOptionalTables(t *testing.T) {
	ctx := context.Background()
	baseDSN := dbtoolTestDSN(t)
	baseName, err := storage.MySQLDatabaseName(baseDSN)
	if err != nil {
		t.Fatalf("parse base dsn: %v", err)
	}
	tempName := baseName + "_authority_" + strings.NewReplacer(".", "_").Replace(time.Now().UTC().Format("20060102150405.000000000"))
	if err := storage.CreateMySQLDatabaseFromDSN(ctx, baseDSN, tempName); err != nil {
		t.Skipf("skip isolated authority healthcheck test, cannot create temp database: %v", err)
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

	player := game.GameState{
		Player: game.Player{ID: "authority_player", Nickname: "权威检查", Faction: "wei", MailCode: "A10001"},
		Resources: game.ResourceState{
			Items:    map[string]int{"wood": 1200, "stone": 900, "iron": 600, "food": 1500},
			Capacity: map[string]int{"wood": 5000, "stone": 5000, "iron": 5000, "food": 5000},
		},
		Buildings:     []game.Building{{ID: "warehouse-1", Type: "warehouse", Level: 1}},
		ResourceSlots: []game.ResourceSlot{{ID: "slot-1", ResourceType: "wood", BuildingID: "warehouse-1"}},
		Generals:      []game.General{{ID: "caocao", Name: "曹操", Level: 1, Stats: map[string]int{}}},
		ServerTime:    time.Now().UTC().Format(time.RFC3339),
	}
	insertAuthorityHealthcheckPlayer(t, db, player)

	result, err := healthcheckAuthority(ctx, dsn)
	if err != nil {
		t.Fatalf("healthcheck authority: %v", err)
	}
	if result.Players != 1 || result.MissingResources != 0 || result.MissingBuildings != 0 || result.MissingResourceSlots != 0 || result.MissingGenerals != 0 || result.MissingCurrencies != 0 || result.MissingLegacyNpc != 0 || result.BigSnapshotPlayers != 0 {
		t.Fatalf("unexpected healthcheck result: %+v", result)
	}
}

// insertAuthorityHealthcheckPlayer 插入一名只有基础权威表完整、可空权威表为空的玩家。
func insertAuthorityHealthcheckPlayer(t *testing.T, db *sql.DB, state game.GameState) {
	t.Helper()
	state.Inventory = nil
	state.Army = nil
	state.RecruitQueues = nil
	state.Buffs = nil
	snapshot := map[string]any{
		"player":            state.Player,
		"resourceSettledAt": state.ResourceSettledAt,
	}
	stateJSON, err := json.Marshal(snapshot)
	if err != nil {
		t.Fatalf("marshal state: %v", err)
	}
	now := time.Now().UTC()
	if _, err := db.Exec(`INSERT INTO accounts (id, username, password_hash, gold, created_at) VALUES (?, ?, ?, ?, ?)`,
		"authority_account", "authority_account", "hash", 0, now); err != nil {
		t.Fatalf("insert account: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO players (id, account_id, nickname, faction, mail_code, state_json, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		state.Player.ID, "authority_account", state.Player.Nickname, state.Player.Faction, state.Player.MailCode, stateJSON, now, now); err != nil {
		t.Fatalf("insert player: %v", err)
	}
	for resourceType, amount := range state.Resources.Items {
		capacity := state.Resources.Capacity[resourceType]
		if _, err := db.Exec(`INSERT INTO player_resources (player_id, resource_type, amount, capacity, updated_at) VALUES (?, ?, ?, ?, ?)`,
			state.Player.ID, resourceType, amount, capacity, now); err != nil {
			t.Fatalf("insert resource: %v", err)
		}
	}
	for _, building := range state.Buildings {
		if _, err := db.Exec(`INSERT INTO player_buildings (player_id, building_id, building_type, level, status, status_ends_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)`,
			state.Player.ID, building.ID, building.Type, building.Level, building.Status, nil, now); err != nil {
			t.Fatalf("insert building: %v", err)
		}
	}
	for _, slot := range state.ResourceSlots {
		if _, err := db.Exec(`INSERT INTO player_resource_slots (player_id, slot_id, resource_type, building_id, unlocked_by, unlocked_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)`,
			state.Player.ID, slot.ID, slot.ResourceType, slot.BuildingID, slot.UnlockedBy, nil, now); err != nil {
			t.Fatalf("insert slot: %v", err)
		}
	}
	for _, general := range state.Generals {
		if _, err := db.Exec(`INSERT INTO player_generals (player_id, general_id, faction, level, exp, stats_json, acquired_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
			state.Player.ID, general.ID, state.Player.Faction, general.Level, general.Exp, `{}`, now, now); err != nil {
			t.Fatalf("insert general: %v", err)
		}
	}
	if _, err := db.Exec(`INSERT INTO player_currencies (player_id, city_gold, last_exchange_at, updated_at) VALUES (?, ?, ?, ?)`,
		state.Player.ID, int(state.CityGold), nil, now); err != nil {
		t.Fatalf("insert currency: %v", err)
	}
}
