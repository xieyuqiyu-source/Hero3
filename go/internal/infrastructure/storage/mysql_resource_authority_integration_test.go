// 本文件验证 player_resources 作为资源权威表的 MySQL 集成场景。
package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/joho/godotenv"

	"hero3/internal/app/game"
)

// openResourceAuthorityTestRepository 打开本地 test_ MySQL 仓储，避免误连生产库。
func openResourceAuthorityTestRepository(t *testing.T) (*MySQLRepository, *sql.DB) {
	t.Helper()
	_ = godotenv.Load("../../../.env")
	dsn := strings.TrimSpace(os.Getenv("HERO3_DATABASE_DSN"))
	if dsn == "" {
		t.Skip("HERO3_DATABASE_DSN is not configured")
	}
	databaseName, err := MySQLDatabaseName(dsn)
	if err != nil {
		t.Fatalf("parse mysql dsn: %v", err)
	}
	if !strings.HasPrefix(databaseName, "test_") {
		t.Skipf("skip resource authority integration test on non-test database %s", databaseName)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	db, err := OpenMySQL(ctx, dsn)
	if err != nil {
		t.Fatalf("open mysql: %v", err)
	}
	if err := MigrateMySQL(ctx, db); err != nil {
		_ = db.Close()
		t.Fatalf("migrate mysql: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})
	return NewMySQLRepository(db), db
}

// createResourceAuthorityPlayer 创建一组可安全清理的测试账号和玩家。
func createResourceAuthorityPlayer(t *testing.T, repo *MySQLRepository, suffix string) (game.Account, game.GameState) {
	t.Helper()
	now := time.Now().UTC()
	suffix = suffix + "_" + time.Now().UTC().Format("20060102150405.000000000")
	suffix = strings.NewReplacer(".", "_").Replace(suffix)
	account := game.Account{
		ID:           "it_account_" + suffix,
		Username:     "it_user_" + suffix,
		PasswordHash: strings.Repeat("a", 64),
		CreatedAt:    now,
	}
	state := game.GameState{
		Player: game.Player{
			ID:       "it_player_" + suffix,
			Nickname: "集成测试" + suffix,
			Faction:  "wei",
		},
		Resources: game.ResourceState{
			Items: map[string]int{
				"wood":  1200,
				"stone": 900,
				"iron":  600,
				"food":  1500,
			},
			Capacity: map[string]int{
				"wood":  5000,
				"stone": 5000,
				"iron":  5000,
				"food":  5000,
			},
		},
		Buildings: []game.Building{
			{ID: "wood_camp-1", Type: "wood_camp", Level: 1},
			{ID: "wood_camp-2", Type: "wood_camp", Level: 1},
			{ID: "warehouse-1", Type: "warehouse", Level: 1},
		},
		Inventory:           map[string]game.ItemStack{},
		Army:                []game.ArmyUnit{},
		RecruitQueues:       []game.RecruitQueue{},
		RecentBattleReports: []game.BattleReport{},
		ResourceSettledAt:   now.Format(time.RFC3339),
		ServerTime:          now.Format(time.RFC3339),
	}
	_ = repo.DeleteAccount(account.ID)
	if err := repo.CreateAccount(account); err != nil {
		t.Fatalf("create account: %v", err)
	}
	t.Cleanup(func() {
		_ = repo.DeleteAccount(account.ID)
	})
	if err := repo.CreatePlayer(account.ID, state, now); err != nil {
		t.Fatalf("create player: %v", err)
	}
	return account, state
}

// TestMySQLDeletePlayerRemovesRelatedRows 验证删除存档时显式清理历史库可能缺少级联的关联表。
func TestMySQLDeletePlayerRemovesRelatedRows(t *testing.T) {
	repo, db := openResourceAuthorityTestRepository(t)
	account, state := createResourceAuthorityPlayer(t, repo, "delete_player_related")
	playerID := state.Player.ID
	now := time.Now().UTC()
	reportID := "it_report_delete_" + strings.NewReplacer(".", "_").Replace(now.Format("150405.000000000"))

	t.Cleanup(func() {
		_, _ = db.Exec(`DELETE FROM battle_report_links WHERE report_id = ?`, reportID)
		_, _ = db.Exec(`DELETE FROM battle_report_states WHERE report_id = ?`, reportID)
		_, _ = db.Exec(`DELETE FROM battle_reports WHERE id = ?`, reportID)
		_, _ = db.Exec(`DELETE FROM mails WHERE player_id = ?`, playerID)
		_, _ = db.Exec(`DELETE FROM minigame_records WHERE player_id = ?`, playerID)
		_, _ = db.Exec(`DELETE FROM pvp_player_states WHERE player_id = ?`, playerID)
		_, _ = db.Exec(`DELETE FROM gold_ledger WHERE player_id = ?`, playerID)
		_, _ = db.Exec(`DELETE FROM item_ledger WHERE player_id = ?`, playerID)
		_, _ = db.Exec(`DELETE FROM players WHERE id = ?`, playerID)
		_, _ = db.Exec(`DELETE FROM accounts WHERE id = ?`, account.ID)
	})

	if _, err := db.Exec(
		`INSERT INTO battle_reports (id, player_id, report_json, created_at) VALUES (?, ?, ?, ?)`,
		reportID, playerID, []byte(`{"id":"`+reportID+`"}`), now,
	); err != nil {
		t.Fatalf("insert battle report: %v", err)
	}
	if _, err := db.Exec(
		`INSERT INTO battle_report_states (id, report_id, player_id, created_at, updated_at) VALUES (?, ?, ?, ?, ?)`,
		"state_"+reportID, reportID, playerID, now, now,
	); err != nil {
		t.Fatalf("insert battle report state: %v", err)
	}
	if _, err := db.Exec(
		`INSERT INTO battle_report_links (id, report_id, token, visibility, created_at) VALUES (?, ?, ?, ?, ?)`,
		"link_"+reportID, reportID, "token_"+reportID, "private", now,
	); err != nil {
		t.Fatalf("insert battle report link: %v", err)
	}
	if _, err := db.Exec(
		`INSERT INTO mails (id, player_id, mail_type, sender_type, title, content, created_at) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		"mail_"+reportID, playerID, "system", "system", "测试信函", "删除存档清理验证", now,
	); err != nil {
		t.Fatalf("insert mail: %v", err)
	}
	if _, err := db.Exec(
		`INSERT INTO minigame_records (id, player_id, game_type, result_name, rarity, created_at) VALUES (?, ?, ?, ?, ?, ?)`,
		"mini_"+reportID, playerID, "fishing", "测试奖励", "common", now,
	); err != nil {
		t.Fatalf("insert minigame record: %v", err)
	}
	if _, err := db.Exec(
		`INSERT INTO pvp_player_states (player_id, created_at, updated_at) VALUES (?, ?, ?)`,
		playerID, now, now,
	); err != nil {
		t.Fatalf("insert pvp player state: %v", err)
	}
	if _, err := db.Exec(
		`INSERT INTO gold_ledger (account_id, player_id, currency, direction, amount, balance_after, created_at) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		account.ID, playerID, "gold", "credit", 1, 1, now,
	); err != nil {
		t.Fatalf("insert gold ledger: %v", err)
	}
	if _, err := db.Exec(
		`INSERT INTO item_ledger (id, player_id, item_id, change_amount, before_amount, after_amount, created_at) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		"item_ledger_"+reportID, playerID, "resource_pack_small", 1, 0, 1, now,
	); err != nil {
		t.Fatalf("insert item ledger: %v", err)
	}
	if _, err := repo.AssignWorldPosition(playerID, "world_1", 7, 8, "test"); err != nil {
		t.Fatalf("insert world position: %v", err)
	}

	if err := repo.DeletePlayer(playerID); err != nil {
		t.Fatalf("DeletePlayer failed: %v", err)
	}

	checks := map[string]string{
		"players":                `SELECT COUNT(*) FROM players WHERE id = ?`,
		"battle_reports":         `SELECT COUNT(*) FROM battle_reports WHERE player_id = ?`,
		"battle_report_states":   `SELECT COUNT(*) FROM battle_report_states WHERE player_id = ?`,
		"battle_report_links":    `SELECT COUNT(*) FROM battle_report_links WHERE report_id = ?`,
		"mails":                  `SELECT COUNT(*) FROM mails WHERE player_id = ?`,
		"minigame_records":       `SELECT COUNT(*) FROM minigame_records WHERE player_id = ?`,
		"pvp_player_states":      `SELECT COUNT(*) FROM pvp_player_states WHERE player_id = ?`,
		"player_world_positions": `SELECT COUNT(*) FROM player_world_positions WHERE player_id = ?`,
		"gold_ledger":            `SELECT COUNT(*) FROM gold_ledger WHERE player_id = ?`,
		"item_ledger":            `SELECT COUNT(*) FROM item_ledger WHERE player_id = ?`,
	}
	for table, query := range checks {
		arg := playerID
		if table == "battle_report_links" {
			arg = reportID
		}
		var count int
		if err := db.QueryRow(query, arg).Scan(&count); err != nil {
			t.Fatalf("count %s: %v", table, err)
		}
		if count != 0 {
			t.Fatalf("expected %s rows to be deleted, got %d", table, count)
		}
	}
}

// readResourceRow 读取 player_resources 中的单项资源。
func readResourceRow(t *testing.T, db *sql.DB, playerID string, resourceType string) (int, int) {
	t.Helper()
	var amount int
	var capacity int
	if err := db.QueryRow(
		`SELECT amount, capacity FROM player_resources WHERE player_id = ? AND resource_type = ?`,
		playerID,
		resourceType,
	).Scan(&amount, &capacity); err != nil {
		t.Fatalf("read player_resources: %v", err)
	}
	return amount, capacity
}

// readPlayerStorageSnapshot 读取 players 表中的轻量快照和更新时间，用于验证普通读取不写库。
func readPlayerStorageSnapshot(t *testing.T, db *sql.DB, playerID string) ([]byte, time.Time) {
	t.Helper()
	var stateJSON []byte
	var updatedAt time.Time
	if err := db.QueryRow(
		`SELECT state_json, updated_at FROM players WHERE id = ? LIMIT 1`,
		playerID,
	).Scan(&stateJSON, &updatedAt); err != nil {
		t.Fatalf("read player storage snapshot: %v", err)
	}
	return append([]byte(nil), stateJSON...), updatedAt
}

// readNpcStateRow 读取 player_npc_states 中的玩家 NPC 城池状态。
func readNpcStateRow(t *testing.T, db *sql.DB, playerID string) (game.NpcState, time.Time, bool) {
	t.Helper()
	var stateJSON []byte
	var updatedAt time.Time
	err := db.QueryRow(
		`SELECT npc_state_json, updated_at FROM player_npc_states WHERE player_id = ? LIMIT 1`,
		playerID,
	).Scan(&stateJSON, &updatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return game.NpcState{}, time.Time{}, false
	}
	if err != nil {
		t.Fatalf("read player_npc_states: %v", err)
	}
	var npcState game.NpcState
	if err := json.Unmarshal(stateJSON, &npcState); err != nil {
		t.Fatalf("unmarshal player_npc_states: %v", err)
	}
	return npcState, updatedAt, true
}

// readAccountGold 读取账号金币。
func readAccountGold(t *testing.T, db *sql.DB, accountID string) int {
	t.Helper()
	var gold int
	if err := db.QueryRow(`SELECT gold FROM accounts WHERE id = ?`, accountID).Scan(&gold); err != nil {
		t.Fatalf("read account gold: %v", err)
	}
	return gold
}

// readSnapshotResource 读取 players.state_json 兼容快照中的单项资源。
func readSnapshotResource(t *testing.T, db *sql.DB, playerID string, resourceType string) int {
	t.Helper()
	var stateJSON []byte
	if err := db.QueryRow(`SELECT state_json FROM players WHERE id = ?`, playerID).Scan(&stateJSON); err != nil {
		t.Fatalf("read state_json: %v", err)
	}
	var state game.GameState
	if err := json.Unmarshal(stateJSON, &state); err != nil {
		t.Fatalf("unmarshal state_json: %v", err)
	}
	return state.Resources.Items[resourceType]
}

// readInventoryRow 读取 player_inventory 中的单项道具。
func readInventoryRow(t *testing.T, db *sql.DB, playerID string, itemID string) (int, bool) {
	t.Helper()
	var amount int
	err := db.QueryRow(
		`SELECT amount FROM player_inventory WHERE player_id = ? AND item_id = ?`,
		playerID,
		itemID,
	).Scan(&amount)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, false
	}
	if err != nil {
		t.Fatalf("read player_inventory: %v", err)
	}
	return amount, true
}

// readInventoryUpdatedAt 读取 player_inventory 中单项道具更新时间。
func readInventoryUpdatedAt(t *testing.T, db *sql.DB, playerID string, itemID string) (time.Time, bool) {
	t.Helper()
	var updatedAt time.Time
	err := db.QueryRow(
		`SELECT updated_at FROM player_inventory WHERE player_id = ? AND item_id = ?`,
		playerID,
		itemID,
	).Scan(&updatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return time.Time{}, false
	}
	if err != nil {
		t.Fatalf("read player_inventory updated_at: %v", err)
	}
	return updatedAt.UTC(), true
}

// readSnapshotItem 读取 players.state_json 兼容快照中的单项道具。
func readSnapshotItem(t *testing.T, db *sql.DB, playerID string, itemID string) (int, bool) {
	t.Helper()
	var stateJSON []byte
	if err := db.QueryRow(`SELECT state_json FROM players WHERE id = ?`, playerID).Scan(&stateJSON); err != nil {
		t.Fatalf("read state_json: %v", err)
	}
	var state game.GameState
	if err := json.Unmarshal(stateJSON, &state); err != nil {
		t.Fatalf("unmarshal state_json: %v", err)
	}
	stack, ok := state.Inventory[itemID]
	return stack.Amount, ok
}

// indexExists 判断指定索引是否存在。
func indexExists(t *testing.T, db *sql.DB, tableName string, indexName string) bool {
	t.Helper()
	var found string
	err := db.QueryRow(
		`SELECT INDEX_NAME
		 FROM information_schema.STATISTICS
		 WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = ? AND INDEX_NAME = ?
		 LIMIT 1`,
		tableName,
		indexName,
	).Scan(&found)
	if errors.Is(err, sql.ErrNoRows) {
		return false
	}
	if err != nil {
		t.Fatalf("read index %s.%s: %v", tableName, indexName, err)
	}
	return found == indexName
}

// readBuildingRow 读取 player_buildings 中的单个建筑。
func readBuildingRow(t *testing.T, db *sql.DB, playerID string, buildingID string) (game.Building, bool) {
	t.Helper()
	var building game.Building
	var upgradeEndsAt sql.NullTime
	var statusEndsAt sql.NullTime
	err := db.QueryRow(
		`SELECT building_id, building_type, level, status, upgrade_ends_at, status_ends_at
		 FROM player_buildings WHERE player_id = ? AND building_id = ?`,
		playerID,
		buildingID,
	).Scan(&building.ID, &building.Type, &building.Level, &building.Status, &upgradeEndsAt, &statusEndsAt)
	if errors.Is(err, sql.ErrNoRows) {
		return game.Building{}, false
	}
	if err != nil {
		t.Fatalf("read player_buildings: %v", err)
	}
	if upgradeEndsAt.Valid {
		value := upgradeEndsAt.Time.UTC().Format(time.RFC3339)
		building.UpgradeEndsAt = &value
	}
	if statusEndsAt.Valid {
		value := statusEndsAt.Time.UTC().Format(time.RFC3339)
		building.StatusEndsAt = &value
	}
	return building, true
}

// readResourceSlotRow 读取 player_resource_slots 中的单个资源田格子。
func readResourceSlotRow(t *testing.T, db *sql.DB, playerID string, slotID string) (game.ResourceSlot, bool) {
	t.Helper()
	var slot game.ResourceSlot
	var unlockedAt sql.NullTime
	err := db.QueryRow(
		`SELECT slot_id, resource_type, building_id, unlocked_by, unlocked_at
		 FROM player_resource_slots WHERE player_id = ? AND slot_id = ?`,
		playerID,
		slotID,
	).Scan(&slot.ID, &slot.ResourceType, &slot.BuildingID, &slot.UnlockedBy, &unlockedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return game.ResourceSlot{}, false
	}
	if err != nil {
		t.Fatalf("read player_resource_slots: %v", err)
	}
	if unlockedAt.Valid {
		slot.UnlockedAt = unlockedAt.Time.UTC().Format(time.RFC3339)
	}
	return slot, true
}

// readArmyRow 读取 player_army_units 中的单项兵力。
func readArmyRow(t *testing.T, db *sql.DB, playerID string, unitType string) (int, bool) {
	t.Helper()
	var amount int
	err := db.QueryRow(
		`SELECT amount FROM player_army_units WHERE player_id = ? AND unit_type = ?`,
		playerID,
		unitType,
	).Scan(&amount)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, false
	}
	if err != nil {
		t.Fatalf("read player_army_units: %v", err)
	}
	return amount, true
}

// readArmyUpdatedAt 读取 player_army_units 中的单项兵力更新时间。
func readArmyUpdatedAt(t *testing.T, db *sql.DB, playerID string, unitType string) (time.Time, bool) {
	t.Helper()
	var updatedAt time.Time
	err := db.QueryRow(
		`SELECT updated_at FROM player_army_units WHERE player_id = ? AND unit_type = ?`,
		playerID,
		unitType,
	).Scan(&updatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return time.Time{}, false
	}
	if err != nil {
		t.Fatalf("read player_army_units updated_at: %v", err)
	}
	return updatedAt.UTC(), true
}

// readRecruitQueueRow 读取 player_recruit_queues 中的单项征兵队列。
func readRecruitQueueRow(t *testing.T, db *sql.DB, playerID string, queueID string) (game.RecruitQueue, bool) {
	t.Helper()
	var queue game.RecruitQueue
	var endsAt time.Time
	err := db.QueryRow(
		`SELECT queue_id, unit_type, amount, ends_at FROM player_recruit_queues WHERE player_id = ? AND queue_id = ?`,
		playerID,
		queueID,
	).Scan(&queue.ID, &queue.UnitType, &queue.Amount, &endsAt)
	if errors.Is(err, sql.ErrNoRows) {
		return game.RecruitQueue{}, false
	}
	if err != nil {
		t.Fatalf("read player_recruit_queues: %v", err)
	}
	queue.EndsAt = endsAt.UTC().Format(time.RFC3339)
	return queue, true
}

// readGeneralRow 读取 player_generals 中的单个武将成长。
func readGeneralRow(t *testing.T, db *sql.DB, playerID string, generalID string) (int, int, bool) {
	t.Helper()
	var level int
	var exp int
	err := db.QueryRow(
		`SELECT level, exp FROM player_generals WHERE player_id = ? AND general_id = ?`,
		playerID,
		generalID,
	).Scan(&level, &exp)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, 0, false
	}
	if err != nil {
		t.Fatalf("read player_generals: %v", err)
	}
	return level, exp, true
}

// readGeneralUpdatedAt 读取 player_generals 中的单个武将更新时间。
func readGeneralUpdatedAt(t *testing.T, db *sql.DB, playerID string, generalID string) (time.Time, bool) {
	t.Helper()
	var updatedAt time.Time
	err := db.QueryRow(
		`SELECT updated_at FROM player_generals WHERE player_id = ? AND general_id = ?`,
		playerID,
		generalID,
	).Scan(&updatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return time.Time{}, false
	}
	if err != nil {
		t.Fatalf("read player_generals updated_at: %v", err)
	}
	return updatedAt.UTC(), true
}

// readGeneralAssignmentCount 读取玩家武将占用记录数量。
func readGeneralAssignmentCount(t *testing.T, db *sql.DB, playerID string) int {
	t.Helper()
	var count int
	if err := db.QueryRow(
		`SELECT COUNT(*) FROM player_general_assignments WHERE player_id = ?`,
		playerID,
	).Scan(&count); err != nil {
		t.Fatalf("read player_general_assignments count: %v", err)
	}
	return count
}

// readBuffCount 读取玩家 Buff 权威表记录数量。
func readBuffCount(t *testing.T, db *sql.DB, playerID string) int {
	t.Helper()
	var count int
	if err := db.QueryRow(
		`SELECT COUNT(*) FROM player_buffs WHERE player_id = ?`,
		playerID,
	).Scan(&count); err != nil {
		t.Fatalf("read player_buffs count: %v", err)
	}
	return count
}

// readBuffValue 读取玩家 Buff 权威表中的单条加成值。
func readBuffValue(t *testing.T, db *sql.DB, playerID string, buffID string) (float64, bool) {
	t.Helper()
	var value float64
	err := db.QueryRow(
		`SELECT modifier_value FROM player_buffs WHERE player_id = ? AND buff_id = ?`,
		playerID,
		buffID,
	).Scan(&value)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, false
	}
	if err != nil {
		t.Fatalf("read player_buffs value: %v", err)
	}
	return value, true
}

// readCurrencyCityGold 读取 player_currencies 中的城金余额。
func readCurrencyCityGold(t *testing.T, db *sql.DB, playerID string) int {
	t.Helper()
	var cityGold int
	if err := db.QueryRow(`SELECT city_gold FROM player_currencies WHERE player_id = ?`, playerID).Scan(&cityGold); err != nil {
		t.Fatalf("read player_currencies: %v", err)
	}
	return cityGold
}

// readCurrencyUpdatedAt 读取 player_currencies 的更新时间。
func readCurrencyUpdatedAt(t *testing.T, db *sql.DB, playerID string) time.Time {
	t.Helper()
	var updatedAt time.Time
	if err := db.QueryRow(`SELECT updated_at FROM player_currencies WHERE player_id = ?`, playerID).Scan(&updatedAt); err != nil {
		t.Fatalf("read player_currencies updated_at: %v", err)
	}
	return updatedAt.UTC()
}

// overwriteSnapshotGeneral 手动篡改兼容快照，用于验证读取时不再相信 state_json.general。
func overwriteSnapshotGeneral(t *testing.T, db *sql.DB, playerID string, level int, exp int) {
	t.Helper()
	var stateJSON []byte
	if err := db.QueryRow(`SELECT state_json FROM players WHERE id = ?`, playerID).Scan(&stateJSON); err != nil {
		t.Fatalf("read state_json: %v", err)
	}
	var state game.GameState
	if err := json.Unmarshal(stateJSON, &state); err != nil {
		t.Fatalf("unmarshal state_json: %v", err)
	}
	if state.General == nil {
		state.General = &game.General{ID: "caocao", Name: "曹操", Buffs: map[string]float64{}}
	}
	state.General.Level = level
	state.General.Exp = exp
	for index := range state.Generals {
		if state.Generals[index].ID == state.General.ID {
			state.Generals[index].Level = level
			state.Generals[index].Exp = exp
		}
	}
	nextJSON, err := json.Marshal(state)
	if err != nil {
		t.Fatalf("marshal state_json: %v", err)
	}
	if _, err := db.Exec(`UPDATE players SET state_json = ? WHERE id = ?`, nextJSON, playerID); err != nil {
		t.Fatalf("overwrite state_json: %v", err)
	}
}

// overwriteSnapshotBuffValue 手动篡改兼容快照，用于验证读取时不再相信 state_json.buffs。
func overwriteSnapshotBuffValue(t *testing.T, db *sql.DB, playerID string, buffID string, value float64) {
	t.Helper()
	var stateJSON []byte
	if err := db.QueryRow(`SELECT state_json FROM players WHERE id = ?`, playerID).Scan(&stateJSON); err != nil {
		t.Fatalf("read state_json: %v", err)
	}
	var state game.GameState
	if err := json.Unmarshal(stateJSON, &state); err != nil {
		t.Fatalf("unmarshal state_json: %v", err)
	}
	for index := range state.Buffs {
		if state.Buffs[index].ID == buffID {
			state.Buffs[index].Value = value
		}
	}
	nextJSON, err := json.Marshal(state)
	if err != nil {
		t.Fatalf("marshal state_json: %v", err)
	}
	if _, err := db.Exec(`UPDATE players SET state_json = ? WHERE id = ?`, nextJSON, playerID); err != nil {
		t.Fatalf("overwrite state_json: %v", err)
	}
}

// readSnapshotBuilding 读取 players.state_json 兼容快照中的单个建筑。
func readSnapshotBuilding(t *testing.T, db *sql.DB, playerID string, buildingID string) (game.Building, bool) {
	t.Helper()
	var stateJSON []byte
	if err := db.QueryRow(`SELECT state_json FROM players WHERE id = ?`, playerID).Scan(&stateJSON); err != nil {
		t.Fatalf("read state_json: %v", err)
	}
	var state game.GameState
	if err := json.Unmarshal(stateJSON, &state); err != nil {
		t.Fatalf("unmarshal state_json: %v", err)
	}
	for _, building := range state.Buildings {
		if building.ID == buildingID {
			return building, true
		}
	}
	return game.Building{}, false
}

// TestMySQLGetStateUsesPlayerGeneralsAuthority 验证读取玩家状态时以 player_generals 为准。
func TestMySQLGetStateUsesPlayerGeneralsAuthority(t *testing.T) {
	repo, db := openResourceAuthorityTestRepository(t)
	_, state := createResourceAuthorityPlayer(t, repo, "generals_read_authority")
	now := time.Now().UTC()
	if _, err := repo.UpdatePlayerState(state.Player.ID, now, func(state *game.GameState) error {
		state.General = &game.General{ID: "caocao", Name: "曹操", Level: 3, Exp: 300, Stats: map[string]int{}, Buffs: map[string]float64{}}
		game.EnsureGeneralRoster(state, now)
		return nil
	}); err != nil {
		t.Fatalf("seed general: %v", err)
	}
	if _, err := db.Exec(
		`UPDATE player_generals SET level = 8, exp = 888 WHERE player_id = ? AND general_id = 'caocao'`,
		state.Player.ID,
	); err != nil {
		t.Fatalf("update player_generals: %v", err)
	}
	overwriteSnapshotGeneral(t, db, state.Player.ID, 1, 1)

	got, err := repo.GetState(state.Player.ID)
	if err != nil {
		t.Fatalf("get state: %v", err)
	}
	if got.General == nil || got.General.ID != "caocao" || got.General.Level != 8 || got.General.Exp != 888 {
		t.Fatalf("expected authoritative general level=8 exp=888, got %+v", got.General)
	}
}

// TestMySQLGetStateUsesPlayerBuffsAuthority 验证读取玩家状态时以 player_buffs 为准。
func TestMySQLGetStateUsesPlayerBuffsAuthority(t *testing.T) {
	repo, db := openResourceAuthorityTestRepository(t)
	_, state := createResourceAuthorityPlayer(t, repo, "buffs_read_authority")
	service := game.NewServiceWithRepository(repo)
	got, err := service.GrantBuff(state.Player.ID, game.StatAttackBonus, 0.1, "percentAdd", 1, "test buff")
	if err != nil {
		t.Fatalf("grant buff: %v", err)
	}
	if len(got.Buffs) != 1 {
		t.Fatalf("expected one buff, got %+v", got.Buffs)
	}
	buffID := got.Buffs[0].ID
	if _, err := db.Exec(
		`UPDATE player_buffs SET modifier_value = 0.35 WHERE player_id = ? AND buff_id = ?`,
		state.Player.ID,
		buffID,
	); err != nil {
		t.Fatalf("update player_buffs: %v", err)
	}
	overwriteSnapshotBuffValue(t, db, state.Player.ID, buffID, 0.01)

	next, err := repo.GetState(state.Player.ID)
	if err != nil {
		t.Fatalf("get state: %v", err)
	}
	if len(next.Buffs) != 1 || next.Buffs[0].Value != 0.35 {
		t.Fatalf("expected authoritative buff value=0.35, got %+v", next.Buffs)
	}
}

// TestMySQLClaimEventProcessingIsIdempotent 验证事件处理记录可以阻止模块重复处理同一事件。
func TestMySQLClaimEventProcessingIsIdempotent(t *testing.T) {
	repo, _ := openResourceAuthorityTestRepository(t)
	event := game.GameEvent{
		Type:      game.EventRewardGranted,
		PlayerID:  "player_event_processing",
		RefType:   "test",
		RefID:     "reward_" + time.Now().UTC().Format("20060102150405.000000000"),
		CreatedAt: time.Now().UTC().Format(time.RFC3339Nano),
	}
	key := game.BuildEventProcessingKey(event)

	first, err := repo.ClaimEventProcessing("mail", "integration_handler", key, time.Now())
	if err != nil {
		t.Fatalf("claim first: %v", err)
	}
	second, err := repo.ClaimEventProcessing("mail", "integration_handler", key, time.Now())
	if err != nil {
		t.Fatalf("claim second: %v", err)
	}
	if !first || second {
		t.Fatalf("expected first claim true and second false, got first=%v second=%v", first, second)
	}
}

// readSnapshotArmy 读取 players.state_json 兼容快照中的单项兵力。
func readSnapshotArmy(t *testing.T, db *sql.DB, playerID string, unitType string) (int, bool) {
	t.Helper()
	var stateJSON []byte
	if err := db.QueryRow(`SELECT state_json FROM players WHERE id = ?`, playerID).Scan(&stateJSON); err != nil {
		t.Fatalf("read state_json: %v", err)
	}
	var state game.GameState
	if err := json.Unmarshal(stateJSON, &state); err != nil {
		t.Fatalf("unmarshal state_json: %v", err)
	}
	for _, unit := range state.Army {
		if unit.UnitType == unitType {
			return unit.Amount, true
		}
	}
	return 0, false
}

// overwriteSnapshotResource 手动篡改兼容快照，用于验证读取时不再相信 state_json.resources。
func overwriteSnapshotResource(t *testing.T, db *sql.DB, playerID string, resourceType string, amount int) {
	t.Helper()
	var stateJSON []byte
	if err := db.QueryRow(`SELECT state_json FROM players WHERE id = ?`, playerID).Scan(&stateJSON); err != nil {
		t.Fatalf("read state_json: %v", err)
	}
	var state game.GameState
	if err := json.Unmarshal(stateJSON, &state); err != nil {
		t.Fatalf("unmarshal state_json: %v", err)
	}
	if state.Resources.Items == nil {
		state.Resources.Items = map[string]int{}
	}
	state.Resources.Items[resourceType] = amount
	nextJSON, err := json.Marshal(state)
	if err != nil {
		t.Fatalf("marshal state_json: %v", err)
	}
	if _, err := db.Exec(`UPDATE players SET state_json = ? WHERE id = ?`, nextJSON, playerID); err != nil {
		t.Fatalf("overwrite state_json: %v", err)
	}
}

// overwriteSnapshotBuildingLevel 手动篡改兼容快照，用于验证读取时不再相信 state_json.buildings。
func overwriteSnapshotBuildingLevel(t *testing.T, db *sql.DB, playerID string, buildingID string, level int) {
	t.Helper()
	var stateJSON []byte
	if err := db.QueryRow(`SELECT state_json FROM players WHERE id = ?`, playerID).Scan(&stateJSON); err != nil {
		t.Fatalf("read state_json: %v", err)
	}
	var state game.GameState
	if err := json.Unmarshal(stateJSON, &state); err != nil {
		t.Fatalf("unmarshal state_json: %v", err)
	}
	for i := range state.Buildings {
		if state.Buildings[i].ID == buildingID {
			state.Buildings[i].Level = level
		}
	}
	nextJSON, err := json.Marshal(state)
	if err != nil {
		t.Fatalf("marshal state_json: %v", err)
	}
	if _, err := db.Exec(`UPDATE players SET state_json = ? WHERE id = ?`, nextJSON, playerID); err != nil {
		t.Fatalf("overwrite state_json: %v", err)
	}
}

// overwriteSnapshotItem 手动篡改兼容快照，用于验证读取时不再相信 state_json.inventory。
func overwriteSnapshotItem(t *testing.T, db *sql.DB, playerID string, itemID string, amount int) {
	t.Helper()
	var stateJSON []byte
	if err := db.QueryRow(`SELECT state_json FROM players WHERE id = ?`, playerID).Scan(&stateJSON); err != nil {
		t.Fatalf("read state_json: %v", err)
	}
	var state game.GameState
	if err := json.Unmarshal(stateJSON, &state); err != nil {
		t.Fatalf("unmarshal state_json: %v", err)
	}
	if state.Inventory == nil {
		state.Inventory = map[string]game.ItemStack{}
	}
	state.Inventory[itemID] = game.ItemStack{ItemID: itemID, Amount: amount}
	nextJSON, err := json.Marshal(state)
	if err != nil {
		t.Fatalf("marshal state_json: %v", err)
	}
	if _, err := db.Exec(`UPDATE players SET state_json = ? WHERE id = ?`, nextJSON, playerID); err != nil {
		t.Fatalf("overwrite state_json: %v", err)
	}
}

// overwriteSnapshotArmy 手动篡改兼容快照，用于验证读取时不再相信 state_json.army。
func overwriteSnapshotArmy(t *testing.T, db *sql.DB, playerID string, unitType string, amount int) {
	t.Helper()
	var stateJSON []byte
	if err := db.QueryRow(`SELECT state_json FROM players WHERE id = ?`, playerID).Scan(&stateJSON); err != nil {
		t.Fatalf("read state_json: %v", err)
	}
	var state game.GameState
	if err := json.Unmarshal(stateJSON, &state); err != nil {
		t.Fatalf("unmarshal state_json: %v", err)
	}
	found := false
	for i := range state.Army {
		if state.Army[i].UnitType == unitType {
			state.Army[i].Amount = amount
			found = true
		}
	}
	if !found {
		state.Army = append(state.Army, game.ArmyUnit{UnitType: unitType, Amount: amount})
	}
	nextJSON, err := json.Marshal(state)
	if err != nil {
		t.Fatalf("marshal state_json: %v", err)
	}
	if _, err := db.Exec(`UPDATE players SET state_json = ? WHERE id = ?`, nextJSON, playerID); err != nil {
		t.Fatalf("overwrite state_json: %v", err)
	}
}

// TestMySQLCreatePlayerWritesPlayerResources 验证创建玩家时写入资源权威表。
func TestMySQLCreatePlayerWritesPlayerResources(t *testing.T) {
	repo, db := openResourceAuthorityTestRepository(t)
	_, state := createResourceAuthorityPlayer(t, repo, "create_resources")

	amount, capacity := readResourceRow(t, db, state.Player.ID, "wood")
	if amount != 1200 || capacity != 5000 {
		t.Fatalf("expected wood row amount=1200 capacity=5000, got amount=%d capacity=%d", amount, capacity)
	}
}

// TestMySQLGetStateUsesPlayerResourcesAuthority 验证读取玩家状态时以 player_resources 为准。
func TestMySQLGetStateUsesPlayerResourcesAuthority(t *testing.T) {
	repo, db := openResourceAuthorityTestRepository(t)
	_, state := createResourceAuthorityPlayer(t, repo, "read_authority")

	if _, err := db.Exec(
		`UPDATE player_resources SET amount = 777, capacity = 8888 WHERE player_id = ? AND resource_type = 'wood'`,
		state.Player.ID,
	); err != nil {
		t.Fatalf("update player_resources: %v", err)
	}
	overwriteSnapshotResource(t, db, state.Player.ID, "wood", 1)

	got, err := repo.GetState(state.Player.ID)
	if err != nil {
		t.Fatalf("get state: %v", err)
	}
	if got.Resources.Items["wood"] != 777 || got.Resources.Capacity["wood"] != 8888 {
		t.Fatalf("expected authoritative resource from table, got %+v", got.Resources)
	}
}

// TestMySQLGrantItemRefreshesInventoryAuthority 验证 GM 发道具会更新背包权威表，且轻量快照不再保存背包。
func TestMySQLGrantItemRefreshesInventoryAuthority(t *testing.T) {
	repo, db := openResourceAuthorityTestRepository(t)
	_, state := createResourceAuthorityPlayer(t, repo, "inventory_grant")
	if err := game.LoadItemsConfig("../../../config/items.json"); err != nil {
		t.Fatalf("load items config: %v", err)
	}
	service := game.NewServiceWithRepository(repo)

	got, err := service.GrantItem(state.Player.ID, "resource_pack_small", 2)
	if err != nil {
		t.Fatalf("grant item: %v", err)
	}
	tableAmount, ok := readInventoryRow(t, db, state.Player.ID, "resource_pack_small")
	snapshotAmount, snapshotOK := readSnapshotItem(t, db, state.Player.ID, "resource_pack_small")
	if got.Inventory["resource_pack_small"].Amount != 2 || !ok || tableAmount != 2 || snapshotOK || snapshotAmount != 0 {
		t.Fatalf("expected inventory state/table amount=2 and no state_json snapshot, state=%+v table=%d/%v snapshot=%d/%v", got.Inventory["resource_pack_small"], tableAmount, ok, snapshotAmount, snapshotOK)
	}
}

// TestMySQLGrantRewardsRefreshesRewardAssetAuthorities 验证标准奖励发放会更新奖励涉及的权威表，且轻量快照只保留兼容字段。
func TestMySQLGrantRewardsRefreshesRewardAssetAuthorities(t *testing.T) {
	repo, db := openResourceAuthorityTestRepository(t)
	_, state := createResourceAuthorityPlayer(t, repo, "reward_assets")
	if err := game.LoadItemsConfig("../../../config/items.json"); err != nil {
		t.Fatalf("load items config: %v", err)
	}
	if err := game.LoadUnitsConfig("../../../config/units"); err != nil {
		t.Fatalf("load units config: %v", err)
	}
	service := game.NewServiceWithRepository(repo)

	result, err := service.GrantRewards(state.Player.ID, []game.Reward{
		{Type: game.RewardTypeResource, ID: "wood", Amount: 25},
		{Type: game.RewardTypeItem, ID: "resource_pack_small", Amount: 2},
		{Type: game.RewardTypeUnit, ID: "qingZhouArmy", Amount: 3},
		{Type: game.RewardTypeCityGold, ID: game.RewardTypeCityGold, Amount: 7},
		{
			Type:   game.RewardTypeBuff,
			ID:     game.StatAttackBonus,
			Amount: 1,
			Metadata: map[string]any{
				"value": 0.15,
				"mode":  "percentAdd",
				"hours": 1,
			},
		},
	}, game.RewardGrantContext{RefType: "test_reward_assets", RefID: "grant_1"})
	if err != nil {
		t.Fatalf("grant rewards: %v", err)
	}

	wood, _ := readResourceRow(t, db, state.Player.ID, "wood")
	itemAmount, itemOK := readInventoryRow(t, db, state.Player.ID, "resource_pack_small")
	unitAmount, unitOK := readArmyRow(t, db, state.Player.ID, "qingZhouArmy")
	buffCount := readBuffCount(t, db, state.Player.ID)
	got := result.State
	if got.Resources.Items["wood"] != 1225 || wood != 1225 {
		t.Fatalf("expected wood authority/state=1225, state=%d table=%d", got.Resources.Items["wood"], wood)
	}
	if got.Inventory["resource_pack_small"].Amount != 2 || !itemOK || itemAmount != 2 {
		t.Fatalf("expected item authority/state=2, state=%+v table=%d/%v", got.Inventory["resource_pack_small"], itemAmount, itemOK)
	}
	if !unitOK || unitAmount != 3 {
		t.Fatalf("expected army authority qingZhouArmy=3, table=%d/%v", unitAmount, unitOK)
	}
	if int(got.CityGold) != 7 || readCurrencyCityGold(t, db, state.Player.ID) != 7 {
		t.Fatalf("expected city gold authority/state=7, state=%d", got.CityGold)
	}
	if len(got.Buffs) != 1 || buffCount != 1 {
		t.Fatalf("expected one buff in state/table, state=%+v table=%d", got.Buffs, buffCount)
	}
	if snapshot := readSnapshotResource(t, db, state.Player.ID, "wood"); snapshot != 0 {
		t.Fatalf("expected no state_json resource snapshot, got %d", snapshot)
	}
	if amount, ok := readSnapshotItem(t, db, state.Player.ID, "resource_pack_small"); ok || amount != 0 {
		t.Fatalf("expected no state_json inventory snapshot, got %d/%v", amount, ok)
	}
	if amount, ok := readSnapshotArmy(t, db, state.Player.ID, "qingZhouArmy"); ok || amount != 0 {
		t.Fatalf("expected no state_json army snapshot, got %d/%v", amount, ok)
	}
}

// TestMySQLGrantResourceRewardScopesAssets 验证纯资源奖励不会触碰背包、兵力和武将权威表。
func TestMySQLGrantResourceRewardScopesAssets(t *testing.T) {
	repo, db := openResourceAuthorityTestRepository(t)
	_, state := createResourceAuthorityPlayer(t, repo, "reward_resource_scope")
	now := time.Now().UTC()
	seedUpdatedAt := now.Add(-time.Hour)
	if _, err := db.Exec(
		`INSERT INTO player_inventory (player_id, slot_id, item_id, amount, obtained_at, updated_at)
		 VALUES (?, 'slot_0001', 'resource_pack_small', 2, ?, ?)
		 ON DUPLICATE KEY UPDATE item_id = VALUES(item_id), amount = VALUES(amount), updated_at = VALUES(updated_at)`,
		state.Player.ID,
		seedUpdatedAt,
		seedUpdatedAt,
	); err != nil {
		t.Fatalf("seed inventory: %v", err)
	}
	if _, err := db.Exec(
		`INSERT INTO player_army_units (player_id, unit_type, amount, updated_at)
		 VALUES (?, 'weiInfantry', 12, ?)
		 ON DUPLICATE KEY UPDATE amount = VALUES(amount), updated_at = VALUES(updated_at)`,
		state.Player.ID,
		seedUpdatedAt,
	); err != nil {
		t.Fatalf("seed army: %v", err)
	}
	if _, err := db.Exec(
		`INSERT INTO player_generals (player_id, general_id, faction, level, exp, stats_json, acquired_at, updated_at)
		 VALUES (?, 'caocao', 'wei', 1, 0, JSON_OBJECT(), ?, ?)
		 ON DUPLICATE KEY UPDATE exp = VALUES(exp), updated_at = VALUES(updated_at)`,
		state.Player.ID,
		seedUpdatedAt,
		seedUpdatedAt,
	); err != nil {
		t.Fatalf("seed general: %v", err)
	}
	beforeInventoryUpdatedAt, ok := readInventoryUpdatedAt(t, db, state.Player.ID, "resource_pack_small")
	if !ok {
		t.Fatalf("expected inventory row")
	}
	beforeArmyUpdatedAt, ok := readArmyUpdatedAt(t, db, state.Player.ID, "weiInfantry")
	if !ok {
		t.Fatalf("expected army row")
	}
	beforeGeneralUpdatedAt, ok := readGeneralUpdatedAt(t, db, state.Player.ID, "caocao")
	if !ok {
		t.Fatalf("expected general row")
	}
	beforeCurrencyUpdatedAt := readCurrencyUpdatedAt(t, db, state.Player.ID)
	service := game.NewServiceWithRepository(repo)

	result, err := service.GrantRewards(state.Player.ID, []game.Reward{
		{Type: game.RewardTypeResource, ID: "wood", Amount: 25},
	}, game.RewardGrantContext{RefType: "test_reward_scope", RefID: "resource_only"})
	if err != nil {
		t.Fatalf("grant resource reward: %v", err)
	}
	wood, _ := readResourceRow(t, db, state.Player.ID, "wood")
	if wood != 1225 || result.State.Resources.Items["wood"] != 1225 {
		t.Fatalf("expected wood updated to 1225, state=%d table=%d", result.State.Resources.Items["wood"], wood)
	}
	if amount, ok := readInventoryRow(t, db, state.Player.ID, "resource_pack_small"); !ok || amount != 2 {
		t.Fatalf("expected inventory row preserved, amount=%d ok=%v", amount, ok)
	}
	afterInventoryUpdatedAt, ok := readInventoryUpdatedAt(t, db, state.Player.ID, "resource_pack_small")
	if !ok || !afterInventoryUpdatedAt.Equal(beforeInventoryUpdatedAt) {
		t.Fatalf("expected inventory row not rewritten, before=%s after=%s ok=%v", beforeInventoryUpdatedAt, afterInventoryUpdatedAt, ok)
	}
	if amount, ok := readArmyRow(t, db, state.Player.ID, "weiInfantry"); !ok || amount != 12 {
		t.Fatalf("expected army row preserved, amount=%d ok=%v", amount, ok)
	}
	afterArmyUpdatedAt, ok := readArmyUpdatedAt(t, db, state.Player.ID, "weiInfantry")
	if !ok || !afterArmyUpdatedAt.Equal(beforeArmyUpdatedAt) {
		t.Fatalf("expected army row not rewritten, before=%s after=%s ok=%v", beforeArmyUpdatedAt, afterArmyUpdatedAt, ok)
	}
	_, exp, ok := readGeneralRow(t, db, state.Player.ID, "caocao")
	if !ok || exp != 0 {
		t.Fatalf("expected general row preserved, exp=%d ok=%v", exp, ok)
	}
	afterGeneralUpdatedAt, ok := readGeneralUpdatedAt(t, db, state.Player.ID, "caocao")
	if !ok || !afterGeneralUpdatedAt.Equal(beforeGeneralUpdatedAt) {
		t.Fatalf("expected general row not rewritten, before=%s after=%s ok=%v", beforeGeneralUpdatedAt, afterGeneralUpdatedAt, ok)
	}
	afterCurrencyUpdatedAt := readCurrencyUpdatedAt(t, db, state.Player.ID)
	if !afterCurrencyUpdatedAt.Equal(beforeCurrencyUpdatedAt) {
		t.Fatalf("expected currency row not rewritten, before=%s after=%s", beforeCurrencyUpdatedAt, afterCurrencyUpdatedAt)
	}
}

// TestMySQLGrantAccountRewardScopesPlayerAssets 验证账号奖励事务不会刷新无关玩家资产。
func TestMySQLGrantAccountRewardScopesPlayerAssets(t *testing.T) {
	repo, db := openResourceAuthorityTestRepository(t)
	account, state := createResourceAuthorityPlayer(t, repo, "reward_account_scope")
	now := time.Now().UTC()
	seedUpdatedAt := now.Add(-time.Hour)
	if _, err := db.Exec(
		`INSERT INTO player_inventory (player_id, slot_id, item_id, amount, obtained_at, updated_at)
		 VALUES (?, 'slot_0001', 'resource_pack_small', 2, ?, ?)
		 ON DUPLICATE KEY UPDATE item_id = VALUES(item_id), amount = VALUES(amount), updated_at = VALUES(updated_at)`,
		state.Player.ID,
		seedUpdatedAt,
		seedUpdatedAt,
	); err != nil {
		t.Fatalf("seed inventory: %v", err)
	}
	if _, err := db.Exec(
		`INSERT INTO player_army_units (player_id, unit_type, amount, updated_at)
		 VALUES (?, 'weiInfantry', 12, ?)
		 ON DUPLICATE KEY UPDATE amount = VALUES(amount), updated_at = VALUES(updated_at)`,
		state.Player.ID,
		seedUpdatedAt,
	); err != nil {
		t.Fatalf("seed army: %v", err)
	}
	if _, err := db.Exec(
		`INSERT INTO player_generals (player_id, general_id, faction, level, exp, stats_json, acquired_at, updated_at)
		 VALUES (?, 'caocao', 'wei', 1, 0, JSON_OBJECT(), ?, ?)
		 ON DUPLICATE KEY UPDATE exp = VALUES(exp), updated_at = VALUES(updated_at)`,
		state.Player.ID,
		seedUpdatedAt,
		seedUpdatedAt,
	); err != nil {
		t.Fatalf("seed general: %v", err)
	}
	beforeInventoryUpdatedAt, ok := readInventoryUpdatedAt(t, db, state.Player.ID, "resource_pack_small")
	if !ok {
		t.Fatalf("expected inventory row")
	}
	beforeArmyUpdatedAt, ok := readArmyUpdatedAt(t, db, state.Player.ID, "weiInfantry")
	if !ok {
		t.Fatalf("expected army row")
	}
	beforeGeneralUpdatedAt, ok := readGeneralUpdatedAt(t, db, state.Player.ID, "caocao")
	if !ok {
		t.Fatalf("expected general row")
	}
	beforeCurrencyUpdatedAt := readCurrencyUpdatedAt(t, db, state.Player.ID)
	service := game.NewServiceWithRepository(repo)

	result, err := service.GrantRewards(state.Player.ID, []game.Reward{
		{Type: game.RewardTypeGold, ID: game.RewardTypeGold, Amount: 5},
		{Type: game.RewardTypeResource, ID: "wood", Amount: 25},
	}, game.RewardGrantContext{AccountID: account.ID, RefType: "test_reward_scope", RefID: "account_resource"})
	if err != nil {
		t.Fatalf("grant account reward: %v", err)
	}
	if result.Account.Gold != 5 || readAccountGold(t, db, account.ID) != 5 {
		t.Fatalf("expected account gold=5, result=%d table=%d", result.Account.Gold, readAccountGold(t, db, account.ID))
	}
	wood, _ := readResourceRow(t, db, state.Player.ID, "wood")
	if wood != 1225 || result.State.Resources.Items["wood"] != 1225 {
		t.Fatalf("expected wood updated to 1225, state=%d table=%d", result.State.Resources.Items["wood"], wood)
	}
	if amount, ok := readInventoryRow(t, db, state.Player.ID, "resource_pack_small"); !ok || amount != 2 {
		t.Fatalf("expected inventory row preserved, amount=%d ok=%v", amount, ok)
	}
	afterInventoryUpdatedAt, ok := readInventoryUpdatedAt(t, db, state.Player.ID, "resource_pack_small")
	if !ok || !afterInventoryUpdatedAt.Equal(beforeInventoryUpdatedAt) {
		t.Fatalf("expected inventory row not rewritten, before=%s after=%s ok=%v", beforeInventoryUpdatedAt, afterInventoryUpdatedAt, ok)
	}
	if amount, ok := readArmyRow(t, db, state.Player.ID, "weiInfantry"); !ok || amount != 12 {
		t.Fatalf("expected army row preserved, amount=%d ok=%v", amount, ok)
	}
	afterArmyUpdatedAt, ok := readArmyUpdatedAt(t, db, state.Player.ID, "weiInfantry")
	if !ok || !afterArmyUpdatedAt.Equal(beforeArmyUpdatedAt) {
		t.Fatalf("expected army row not rewritten, before=%s after=%s ok=%v", beforeArmyUpdatedAt, afterArmyUpdatedAt, ok)
	}
	_, exp, ok := readGeneralRow(t, db, state.Player.ID, "caocao")
	if !ok || exp != 0 {
		t.Fatalf("expected general row preserved, exp=%d ok=%v", exp, ok)
	}
	afterGeneralUpdatedAt, ok := readGeneralUpdatedAt(t, db, state.Player.ID, "caocao")
	if !ok || !afterGeneralUpdatedAt.Equal(beforeGeneralUpdatedAt) {
		t.Fatalf("expected general row not rewritten, before=%s after=%s ok=%v", beforeGeneralUpdatedAt, afterGeneralUpdatedAt, ok)
	}
	afterCurrencyUpdatedAt := readCurrencyUpdatedAt(t, db, state.Player.ID)
	if !afterCurrencyUpdatedAt.Equal(beforeCurrencyUpdatedAt) {
		t.Fatalf("expected currency row not rewritten, before=%s after=%s", beforeCurrencyUpdatedAt, afterCurrencyUpdatedAt)
	}
}

// TestMySQLGetStateUsesPlayerInventoryAuthority 验证读取玩家状态时以 player_inventory 为准。
func TestMySQLGetStateUsesPlayerInventoryAuthority(t *testing.T) {
	repo, db := openResourceAuthorityTestRepository(t)
	_, state := createResourceAuthorityPlayer(t, repo, "inventory_read_authority")
	if err := game.LoadItemsConfig("../../../config/items.json"); err != nil {
		t.Fatalf("load items config: %v", err)
	}
	service := game.NewServiceWithRepository(repo)
	if _, err := service.GrantItem(state.Player.ID, "resource_pack_small", 1); err != nil {
		t.Fatalf("grant item: %v", err)
	}
	if _, err := db.Exec(
		`UPDATE player_inventory SET amount = 5 WHERE player_id = ? AND item_id = 'resource_pack_small'`,
		state.Player.ID,
	); err != nil {
		t.Fatalf("update player_inventory: %v", err)
	}
	overwriteSnapshotItem(t, db, state.Player.ID, "resource_pack_small", 1)

	got, err := repo.GetState(state.Player.ID)
	if err != nil {
		t.Fatalf("get state: %v", err)
	}
	if got.Inventory["resource_pack_small"].Amount != 5 {
		t.Fatalf("expected authoritative inventory amount=5, got %+v", got.Inventory["resource_pack_small"])
	}
}

// TestMySQLUseItemSpendsInventoryAuthority 验证使用道具会扣减背包权威表，且轻量快照不保存背包。
func TestMySQLUseItemSpendsInventoryAuthority(t *testing.T) {
	repo, db := openResourceAuthorityTestRepository(t)
	_, state := createResourceAuthorityPlayer(t, repo, "inventory_use")
	if err := game.LoadItemsConfig("../../../config/items.json"); err != nil {
		t.Fatalf("load items config: %v", err)
	}
	service := game.NewServiceWithRepository(repo)
	if _, err := service.GrantItem(state.Player.ID, "resource_pack_small", 1); err != nil {
		t.Fatalf("grant item: %v", err)
	}

	if _, err := service.UseItem(state.Player.ID, "resource_pack_small", 1); err != nil {
		t.Fatalf("use item: %v", err)
	}
	if amount, ok := readInventoryRow(t, db, state.Player.ID, "resource_pack_small"); ok || amount != 0 {
		t.Fatalf("expected resource_pack_small removed from player_inventory, amount=%d ok=%v", amount, ok)
	}
	if amount, ok := readSnapshotItem(t, db, state.Player.ID, "resource_pack_small"); ok || amount != 0 {
		t.Fatalf("expected resource_pack_small removed from state_json snapshot, amount=%d ok=%v", amount, ok)
	}
}

// TestMySQLGeneralExpItemStateOnlySyncsInventoryAndGenerals 验证经验包小事务只写背包和武将。
func TestMySQLGeneralExpItemStateOnlySyncsInventoryAndGenerals(t *testing.T) {
	repo, db := openResourceAuthorityTestRepository(t)
	_, state := createResourceAuthorityPlayer(t, repo, "general_exp_item_scope")
	if !indexExists(t, db, "player_inventory", "idx_player_inventory_player_item") {
		t.Fatalf("expected player_inventory(player_id, item_id) index for scoped item locks")
	}
	if err := game.LoadItemsConfig("../../../config/items.json"); err != nil {
		t.Fatalf("load items config: %v", err)
	}
	service := game.NewServiceWithRepository(repo)
	if _, err := service.GrantItem(state.Player.ID, "general_exp_small", 1); err != nil {
		t.Fatalf("grant item: %v", err)
	}
	if _, err := service.GrantItem(state.Player.ID, "resource_pack_small", 2); err != nil {
		t.Fatalf("grant unrelated item: %v", err)
	}
	if _, err := db.Exec(
		`INSERT INTO player_generals (player_id, general_id, faction, level, exp, stats_json, acquired_at, updated_at)
		 VALUES (?, 'caocao', 'wei', 1, 0, JSON_OBJECT(), UTC_TIMESTAMP(6), UTC_TIMESTAMP(6))
		 ON DUPLICATE KEY UPDATE exp = VALUES(exp), updated_at = VALUES(updated_at)`,
		state.Player.ID,
	); err != nil {
		t.Fatalf("seed general: %v", err)
	}
	if _, err := db.Exec(
		`INSERT INTO player_generals (player_id, general_id, faction, level, exp, stats_json, acquired_at, updated_at)
		 VALUES (?, 'xuchu', 'wei', 1, 77, JSON_OBJECT(), UTC_TIMESTAMP(6), UTC_TIMESTAMP(6))
		 ON DUPLICATE KEY UPDATE exp = VALUES(exp), updated_at = VALUES(updated_at)`,
		state.Player.ID,
	); err != nil {
		t.Fatalf("seed unrelated general: %v", err)
	}
	if _, err := db.Exec(
		`INSERT INTO player_general_assignments (player_id, assignment_id, general_id, assignment_slot, module_id, status, assigned_at, updated_at)
		 VALUES (?, 'main', 'caocao', 'main', '', 'active', UTC_TIMESTAMP(6), UTC_TIMESTAMP(6))
		 ON DUPLICATE KEY UPDATE general_id = VALUES(general_id), assignment_slot = VALUES(assignment_slot), status = VALUES(status), updated_at = VALUES(updated_at)`,
		state.Player.ID,
	); err != nil {
		t.Fatalf("seed general assignment: %v", err)
	}
	if _, err := db.Exec(
		`INSERT INTO player_army_units (player_id, unit_type, amount, updated_at)
		 VALUES (?, 'weiInfantry', 5, UTC_TIMESTAMP(6))
		 ON DUPLICATE KEY UPDATE amount = VALUES(amount), updated_at = VALUES(updated_at)`,
		state.Player.ID,
	); err != nil {
		t.Fatalf("seed army: %v", err)
	}
	beforeWood, _ := readResourceRow(t, db, state.Player.ID, "wood")

	result, err := repo.UpdateGeneralExpItemState(state.Player.ID, "general_exp_small", time.Now(), func(state *game.GameState) error {
		state.Inventory = map[string]game.ItemStack{}
		state.InventorySlots = nil
		if state.General == nil {
			t.Fatalf("expected current general in scoped transaction")
		}
		state.General.Exp += 123
		for index := range state.Generals {
			if state.Generals[index].ID == state.General.ID {
				state.Generals[index].Exp = state.General.Exp
			}
		}
		if state.Resources.Items == nil {
			state.Resources.Items = map[string]int{}
		}
		state.Resources.Items["wood"] += 999
		state.Army = append(state.Army, game.ArmyUnit{UnitType: "weiInfantry", Amount: 99})
		return nil
	})
	if err != nil {
		t.Fatalf("update general exp item state: %v", err)
	}

	if amount, ok := readInventoryRow(t, db, state.Player.ID, "general_exp_small"); ok || amount != 0 {
		t.Fatalf("expected scoped transaction to remove inventory item, amount=%d ok=%v", amount, ok)
	}
	if amount, ok := readInventoryRow(t, db, state.Player.ID, "resource_pack_small"); !ok || amount != 2 {
		t.Fatalf("expected scoped transaction not to touch unrelated item, amount=%d ok=%v", amount, ok)
	}
	if got := result.Inventory["resource_pack_small"].Amount; got != 2 {
		t.Fatalf("expected response to include full inventory after scoped commit, got resource_pack_small=%d", got)
	}
	_, exp, ok := readGeneralRow(t, db, state.Player.ID, "caocao")
	if !ok || exp != 123 {
		t.Fatalf("expected scoped transaction to update general exp=123, exp=%d ok=%v", exp, ok)
	}
	_, unrelatedExp, ok := readGeneralRow(t, db, state.Player.ID, "xuchu")
	if !ok || unrelatedExp != 77 {
		t.Fatalf("expected scoped transaction not to touch unrelated general, exp=%d ok=%v", unrelatedExp, ok)
	}
	if len(result.Generals) < 2 {
		t.Fatalf("expected response to include full general roster after scoped commit, got %+v", result.Generals)
	}
	afterWood, _ := readResourceRow(t, db, state.Player.ID, "wood")
	if afterWood != beforeWood {
		t.Fatalf("expected scoped transaction not to sync resources, before=%d after=%d", beforeWood, afterWood)
	}
	if amount, ok := readArmyRow(t, db, state.Player.ID, "weiInfantry"); !ok || amount != 5 {
		t.Fatalf("expected scoped transaction not to sync army, amount=%d ok=%v", amount, ok)
	}
}

// TestMySQLResourceGrantRefreshesAuthority 验证资源发放同步刷新权威表，且轻量快照不再保存资源。
func TestMySQLResourceGrantRefreshesAuthority(t *testing.T) {
	repo, db := openResourceAuthorityTestRepository(t)
	_, state := createResourceAuthorityPlayer(t, repo, "grant_refresh")
	service := game.NewServiceWithRepository(repo)

	got, err := service.AdjustResources(state.Player.ID, map[string]int{"wood": 25})
	if err != nil {
		t.Fatalf("adjust resources: %v", err)
	}
	amount, _ := readResourceRow(t, db, state.Player.ID, "wood")
	snapshot := readSnapshotResource(t, db, state.Player.ID, "wood")
	if got.Resources.Items["wood"] != 1225 || amount != 1225 || snapshot != 0 {
		t.Fatalf("expected table/state wood=1225 and no state_json resource snapshot, got state=%d table=%d snapshot=%d", got.Resources.Items["wood"], amount, snapshot)
	}
}

// TestMySQLResourceSpendRollbackOnInsufficient 验证资源不足时业务失败且资源表不被扣成负数。
func TestMySQLResourceSpendRollbackOnInsufficient(t *testing.T) {
	repo, db := openResourceAuthorityTestRepository(t)
	_, state := createResourceAuthorityPlayer(t, repo, "spend_rollback")
	service := game.NewServiceWithRepository(repo)

	if _, err := db.Exec(
		`UPDATE player_resources SET amount = 0 WHERE player_id = ? AND resource_type = 'wood'`,
		state.Player.ID,
	); err != nil {
		t.Fatalf("prepare resource shortage: %v", err)
	}
	_, err := service.UpgradeBuilding(state.Player.ID, "wood_camp-1")
	if !errors.Is(err, game.ErrInsufficientRes) {
		t.Fatalf("expected insufficient resources, got %v", err)
	}
	amount, _ := readResourceRow(t, db, state.Player.ID, "wood")
	if amount != 0 {
		t.Fatalf("expected wood unchanged after rollback, got %d", amount)
	}
}

// TestMySQLBuildingUpgradeSpendsPlayerResources 验证建筑升级扣减资源权威表，且轻量快照不再保存资源。
func TestMySQLBuildingUpgradeSpendsPlayerResources(t *testing.T) {
	repo, db := openResourceAuthorityTestRepository(t)
	_, state := createResourceAuthorityPlayer(t, repo, "building_spend")
	service := game.NewServiceWithRepository(repo)

	before, _ := readResourceRow(t, db, state.Player.ID, "wood")
	if _, err := service.UpgradeBuilding(state.Player.ID, "wood_camp-1"); err != nil {
		t.Fatalf("upgrade building: %v", err)
	}
	after, _ := readResourceRow(t, db, state.Player.ID, "wood")
	snapshot := readSnapshotResource(t, db, state.Player.ID, "wood")
	if after >= before || snapshot != 0 {
		t.Fatalf("expected wood spent in table and no state_json resource snapshot, before=%d table=%d snapshot=%d", before, after, snapshot)
	}
}

// TestMySQLRecruitSpendsPlayerResources 验证征兵扣减资源权威表，且轻量快照不再保存资源。
func TestMySQLRecruitSpendsPlayerResources(t *testing.T) {
	repo, db := openResourceAuthorityTestRepository(t)
	_, state := createResourceAuthorityPlayer(t, repo, "recruit_spend")
	if err := game.LoadUnitsConfig("../../../config/units"); err != nil {
		t.Fatalf("load units config: %v", err)
	}
	service := game.NewServiceWithRepository(repo)

	before, _ := readResourceRow(t, db, state.Player.ID, "wood")
	if _, err := service.Recruit(state.Player.ID, "qingZhouArmy", 1); err != nil {
		t.Fatalf("recruit: %v", err)
	}
	after, _ := readResourceRow(t, db, state.Player.ID, "wood")
	snapshot := readSnapshotResource(t, db, state.Player.ID, "wood")
	if before-after != 240 || snapshot != 0 {
		t.Fatalf("expected recruit to spend 240 wood in table and no state_json resource snapshot, before=%d table=%d snapshot=%d", before, after, snapshot)
	}
}

// TestMySQLConcurrentResourceSpendIsConsistent 验证并发扣减同一玩家资源时只会有一个事务成功。
func TestMySQLConcurrentResourceSpendIsConsistent(t *testing.T) {
	repo, db := openResourceAuthorityTestRepository(t)
	_, state := createResourceAuthorityPlayer(t, repo, "concurrent_spend")
	service := game.NewServiceWithRepository(repo)

	costs := map[string]int{"wood": 130, "stone": 330, "iron": 170, "food": 200}
	for resourceType, amount := range costs {
		if _, err := db.Exec(
			`UPDATE player_resources SET amount = ?, capacity = 5000 WHERE player_id = ? AND resource_type = ?`,
			amount,
			state.Player.ID,
			resourceType,
		); err != nil {
			t.Fatalf("prepare resource %s: %v", resourceType, err)
		}
	}

	errCh := make(chan error, 2)
	go func() {
		_, err := service.UpgradeBuilding(state.Player.ID, "wood_camp-1")
		errCh <- err
	}()
	go func() {
		_, err := service.UpgradeBuilding(state.Player.ID, "wood_camp-2")
		errCh <- err
	}()

	successes := 0
	failures := 0
	for i := 0; i < 2; i++ {
		err := <-errCh
		if err == nil {
			successes++
			continue
		}
		if errors.Is(err, game.ErrInsufficientRes) {
			failures++
			continue
		}
		t.Fatalf("unexpected concurrent upgrade error: %v", err)
	}
	if successes != 1 || failures != 1 {
		t.Fatalf("expected one success and one insufficient resource failure, got successes=%d failures=%d", successes, failures)
	}
	amount, _ := readResourceRow(t, db, state.Player.ID, "wood")
	snapshot := readSnapshotResource(t, db, state.Player.ID, "wood")
	if amount != 0 || snapshot != 0 {
		t.Fatalf("expected wood table=0 and no state_json resource snapshot after one spend, table=%d snapshot=%d", amount, snapshot)
	}
}

// TestMySQLMailClaimGrantsPlayerResources 验证信函附件领取会更新资源权威表。
func TestMySQLMailClaimGrantsPlayerResources(t *testing.T) {
	repo, db := openResourceAuthorityTestRepository(t)
	_, state := createResourceAuthorityPlayer(t, repo, "mail_grant")
	service := game.NewServiceWithRepository(repo)

	mail, err := service.SendMail(game.SendMailRequest{
		PlayerID:   state.Player.ID,
		MailType:   "reward",
		SenderType: "gm",
		Title:      "测试资源附件",
		Content:    "领取资源",
		Attachments: []game.MailAttachment{
			{Type: game.RewardTypeResource, ItemID: "wood", Amount: 30},
		},
	})
	if err != nil {
		t.Fatalf("send mail: %v", err)
	}
	if _, err := service.ClaimMailAttachments(state.Player.ID, mail.ID); err != nil {
		t.Fatalf("claim mail: %v", err)
	}
	amount, _ := readResourceRow(t, db, state.Player.ID, "wood")
	if amount != 1230 {
		t.Fatalf("expected wood=1230 after mail claim, got %d", amount)
	}
}

// TestMySQLResourcePackItemGrantsPlayerResources 验证道具资源包会更新资源权威表。
func TestMySQLResourcePackItemGrantsPlayerResources(t *testing.T) {
	repo, db := openResourceAuthorityTestRepository(t)
	_, state := createResourceAuthorityPlayer(t, repo, "item_grant")
	if err := game.LoadItemsConfig("../../../config/items.json"); err != nil {
		t.Fatalf("load items config: %v", err)
	}
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := repo.UpdatePlayerState(state.Player.ID, time.Now(), func(state *game.GameState) error {
		if state.Inventory == nil {
			state.Inventory = map[string]game.ItemStack{}
		}
		state.Inventory["resource_pack_small"] = game.ItemStack{
			ItemID:     "resource_pack_small",
			Amount:     1,
			ObtainedAt: now,
			UpdatedAt:  now,
		}
		state.Inventory["general_exp_small"] = game.ItemStack{
			ItemID:     "general_exp_small",
			Amount:     2,
			ObtainedAt: now,
			UpdatedAt:  now,
		}
		return nil
	})
	if err != nil {
		t.Fatalf("seed item: %v", err)
	}
	service := game.NewServiceWithRepository(repo)
	beforeWood, _ := readResourceRow(t, db, state.Player.ID, "wood")
	beforeUnrelatedUpdatedAt, ok := readInventoryUpdatedAt(t, db, state.Player.ID, "general_exp_small")
	if !ok {
		t.Fatalf("expected unrelated item row before use")
	}

	if _, err := service.UseItem(state.Player.ID, "resource_pack_small", 1); err != nil {
		t.Fatalf("use resource pack: %v", err)
	}
	amount, afterCapacity := readResourceRow(t, db, state.Player.ID, "wood")
	item, ok := game.GetItemDefinition("resource_pack_small")
	if !ok || len(item.Effects) == 0 {
		t.Fatalf("resource_pack_small item config missing")
	}
	expectedWood := beforeWood + item.Effects[0].Resources["wood"]
	if expectedWood > afterCapacity {
		expectedWood = afterCapacity
	}
	if amount != expectedWood {
		t.Fatalf("expected wood=%d after resource pack, got %d", expectedWood, amount)
	}
	if amount, ok := readInventoryRow(t, db, state.Player.ID, "general_exp_small"); !ok || amount != 2 {
		t.Fatalf("expected unrelated item preserved, amount=%d ok=%v", amount, ok)
	}
	afterUnrelatedUpdatedAt, ok := readInventoryUpdatedAt(t, db, state.Player.ID, "general_exp_small")
	if !ok || !afterUnrelatedUpdatedAt.Equal(beforeUnrelatedUpdatedAt) {
		t.Fatalf("expected unrelated item not to be refreshed, before=%s after=%s ok=%v", beforeUnrelatedUpdatedAt, afterUnrelatedUpdatedAt, ok)
	}
}

// TestMySQLGetStateUsesPlayerBuildingsAuthority 验证读取玩家状态时建筑以 player_buildings 为准。
func TestMySQLGetStateUsesPlayerBuildingsAuthority(t *testing.T) {
	repo, db := openResourceAuthorityTestRepository(t)
	_, state := createResourceAuthorityPlayer(t, repo, "building_authority")

	if _, err := db.Exec(
		`UPDATE player_buildings SET level = 7 WHERE player_id = ? AND building_id = ?`,
		state.Player.ID,
		"wood_camp-1",
	); err != nil {
		t.Fatalf("update player_buildings: %v", err)
	}
	overwriteSnapshotBuildingLevel(t, db, state.Player.ID, "wood_camp-1", 1)

	got, err := repo.GetState(state.Player.ID)
	if err != nil {
		t.Fatalf("get state: %v", err)
	}
	for _, building := range got.Buildings {
		if building.ID == "wood_camp-1" {
			if building.Level != 7 {
				t.Fatalf("expected authoritative building level=7, got %d", building.Level)
			}
			return
		}
	}
	t.Fatalf("expected wood_camp-1 in authoritative buildings")
}

// TestMySQLServiceGetStateDoesNotWriteStorage 验证服务层普通状态读取只读投影，不写 players 或资源权威表。
func TestMySQLServiceGetStateDoesNotWriteStorage(t *testing.T) {
	repo, db := openResourceAuthorityTestRepository(t)
	_, state := createResourceAuthorityPlayer(t, repo, "readonly_service_state")
	service := game.NewServiceWithRepository(repo)

	beforeJSON, beforeUpdatedAt := readPlayerStorageSnapshot(t, db, state.Player.ID)
	beforeWood, beforeCapacity := readResourceRow(t, db, state.Player.ID, "wood")

	got, err := service.GetState(state.Player.ID)
	if err != nil {
		t.Fatalf("service GetState: %v", err)
	}
	if got.Resources.Items["wood"] < beforeWood {
		t.Fatalf("expected readonly projected resources not to go backwards, before=%d got=%d", beforeWood, got.Resources.Items["wood"])
	}

	afterJSON, afterUpdatedAt := readPlayerStorageSnapshot(t, db, state.Player.ID)
	afterWood, afterCapacity := readResourceRow(t, db, state.Player.ID, "wood")
	if !beforeUpdatedAt.Equal(afterUpdatedAt) {
		t.Fatalf("expected GetState not to update players.updated_at, before=%s after=%s", beforeUpdatedAt, afterUpdatedAt)
	}
	if string(beforeJSON) != string(afterJSON) {
		t.Fatalf("expected GetState not to update players.state_json")
	}
	if beforeWood != afterWood || beforeCapacity != afterCapacity {
		t.Fatalf("expected GetState not to update player_resources, before=%d/%d after=%d/%d", beforeWood, beforeCapacity, afterWood, afterCapacity)
	}
}

// TestMySQLRepairPlayerCoreAssetsPersistsAuthorityRows 验证显式修复入口会把旧玩家核心资产补入权威表。
func TestMySQLRepairPlayerCoreAssetsPersistsAuthorityRows(t *testing.T) {
	repo, db := openResourceAuthorityTestRepository(t)
	now := time.Now().UTC()
	suffix := strings.NewReplacer(".", "_").Replace("repair_core_" + now.Format("20060102150405.000000000"))
	account := game.Account{
		ID:           "it_account_" + suffix,
		Username:     "it_user_" + suffix,
		PasswordHash: strings.Repeat("b", 64),
		CreatedAt:    now,
	}
	state := game.GameState{
		Player: game.Player{
			ID:       "it_player_" + suffix,
			Nickname: "修复测试" + suffix,
			Faction:  "wei",
		},
		Resources: game.ResourceState{
			Items:    map[string]int{"wood": 1000, "stone": 1000, "iron": 1000, "food": 1000},
			Capacity: map[string]int{"wood": 5000, "stone": 5000, "iron": 5000, "food": 5000},
		},
		Buildings:         []game.Building{{ID: "warehouse-1", Type: "warehouse", Level: 1}},
		Inventory:         nil,
		Army:              []game.ArmyUnit{},
		RecruitQueues:     []game.RecruitQueue{},
		ResourceSettledAt: now.Format(time.RFC3339),
		ServerTime:        now.Format(time.RFC3339),
	}
	_ = repo.DeleteAccount(account.ID)
	if err := repo.CreateAccount(account); err != nil {
		t.Fatalf("create account: %v", err)
	}
	t.Cleanup(func() {
		_ = repo.DeleteAccount(account.ID)
	})
	if err := repo.CreatePlayer(account.ID, state, now); err != nil {
		t.Fatalf("create player: %v", err)
	}
	service := game.NewServiceWithRepository(repo)

	if _, ok := readBuildingRow(t, db, state.Player.ID, "construction_bureau-1"); ok {
		t.Fatalf("expected legacy test player to start without construction bureau")
	}
	if _, _, ok := readGeneralRow(t, db, state.Player.ID, "caocao"); ok {
		t.Fatalf("expected legacy test player to start without general authority row")
	}

	result, err := service.RepairPlayerCoreAssets(state.Player.ID)
	if err != nil {
		t.Fatalf("RepairPlayerCoreAssets: %v", err)
	}
	if !result.Changed {
		t.Fatalf("expected repair to report changes")
	}
	if _, ok := readBuildingRow(t, db, state.Player.ID, "construction_bureau-1"); !ok {
		t.Fatalf("expected repair to persist construction bureau")
	}
	if _, ok := readResourceSlotRow(t, db, state.Player.ID, "construction_resource_slot-1"); ok {
		t.Fatalf("expected level 1 construction bureau not to persist construction resource slot")
	}
	if _, _, ok := readGeneralRow(t, db, state.Player.ID, "caocao"); !ok {
		t.Fatalf("expected repair to persist default general")
	}
	if count := readGeneralAssignmentCount(t, db, state.Player.ID); count == 0 {
		t.Fatalf("expected repair to persist general assignment")
	}
}

// TestMySQLBuildingUpgradeRefreshesBuildingAuthority 验证建筑升级会同步建筑权威表、资源田格子，且轻量快照不再保存建筑。
func TestMySQLBuildingUpgradeRefreshesBuildingAuthority(t *testing.T) {
	repo, db := openResourceAuthorityTestRepository(t)
	_, state := createResourceAuthorityPlayer(t, repo, "building_sync")
	service := game.NewServiceWithRepository(repo)

	if _, err := service.UpgradeBuilding(state.Player.ID, "wood_camp-1"); err != nil {
		t.Fatalf("upgrade building: %v", err)
	}
	row, ok := readBuildingRow(t, db, state.Player.ID, "wood_camp-1")
	if !ok {
		t.Fatalf("expected player_buildings row")
	}
	if row.Status != game.BuildingStatusUpgrading || row.UpgradeEndsAt == nil {
		t.Fatalf("expected authoritative building upgrading with ends_at, got %+v", row)
	}
	snapshot, ok := readSnapshotBuilding(t, db, state.Player.ID, "wood_camp-1")
	if ok || snapshot.ID != "" {
		t.Fatalf("expected no state_json building snapshot, got %+v", snapshot)
	}
	slot, ok := readResourceSlotRow(t, db, state.Player.ID, "wood_camp-1")
	if !ok {
		t.Fatalf("expected resource slot for wood_camp-1")
	}
	if slot.ResourceType != "wood" || slot.BuildingID != "wood_camp-1" {
		t.Fatalf("expected wood resource slot bound to wood_camp-1, got %+v", slot)
	}
}

// TestMySQLCombatStateDoesNotSyncNonCombatAssets 验证战斗事务不再写回建筑、资源田、征兵队列和 Buff。
func TestMySQLCombatStateDoesNotSyncNonCombatAssets(t *testing.T) {
	repo, db := openResourceAuthorityTestRepository(t)
	_, state := createResourceAuthorityPlayer(t, repo, "combat_scope")
	service := game.NewServiceWithRepository(repo)
	now := time.Now().UTC()
	buffResult, err := service.GrantBuff(state.Player.ID, game.StatAttackBonus, 0.25, "percentAdd", 1, "combat readonly buff")
	if err != nil {
		t.Fatalf("seed buff: %v", err)
	}
	if len(buffResult.Buffs) != 1 {
		t.Fatalf("expected seeded buff, got %+v", buffResult.Buffs)
	}
	buffID := buffResult.Buffs[0].ID
	if _, err := db.Exec(
		`INSERT INTO player_resource_slots (player_id, slot_id, resource_type, building_id, unlocked_by, unlocked_at, updated_at)
		 VALUES (?, 'combat_slot', 'wood', 'wood_camp-1', 'test', ?, ?)`,
		state.Player.ID,
		now,
		now,
	); err != nil {
		t.Fatalf("seed resource slot: %v", err)
	}
	queueEndsAt := now.Add(time.Hour)
	if _, err := db.Exec(
		`INSERT INTO player_recruit_queues (player_id, queue_id, unit_type, amount, ends_at, updated_at)
		 VALUES (?, 'combat_queue', 'qingZhouArmy', 1, ?, ?)`,
		state.Player.ID,
		queueEndsAt,
		now,
	); err != nil {
		t.Fatalf("seed recruit queue: %v", err)
	}
	stableArmyUpdatedAt := now.Add(-time.Hour)
	if _, err := db.Exec(
		`INSERT INTO player_army_units (player_id, unit_type, amount, updated_at)
		 VALUES (?, 'weiInfantry', 5, ?), (?, 'qingZhouArmy', 9, ?)
		 ON DUPLICATE KEY UPDATE amount = VALUES(amount), updated_at = VALUES(updated_at)`,
		state.Player.ID,
		stableArmyUpdatedAt,
		state.Player.ID,
		stableArmyUpdatedAt,
	); err != nil {
		t.Fatalf("seed army: %v", err)
	}
	beforeUnchangedArmyUpdatedAt, ok := readArmyUpdatedAt(t, db, state.Player.ID, "qingZhouArmy")
	if !ok {
		t.Fatalf("expected seeded unchanged army row")
	}

	beforeWood, _ := readResourceRow(t, db, state.Player.ID, "wood")
	_, err = repo.UpdateCombatState(state.Player.ID, game.CombatAssetScope{}, time.Now(), func(state *game.GameState) error {
		state.Resources.Items["wood"] += 10
		for index := range state.Army {
			if state.Army[index].UnitType == "weiInfantry" {
				state.Army[index].Amount = 3
			}
		}
		for index := range state.Buildings {
			if state.Buildings[index].ID == "wood_camp-1" {
				state.Buildings[index].Level = 9
			}
		}
		for index := range state.ResourceSlots {
			if state.ResourceSlots[index].ID == "combat_slot" {
				state.ResourceSlots[index].ResourceType = "iron"
			}
		}
		state.RecruitQueues = append(state.RecruitQueues, game.RecruitQueue{
			ID:       "combat_queue_new",
			UnitType: "qingZhouArmy",
			Amount:   99,
			EndsAt:   queueEndsAt.Add(time.Hour).Format(time.RFC3339),
		})
		state.Buffs = []game.Buff{{ID: "combat_buff_new", Key: game.StatAttackBonus, Value: 0.99, Mode: "percentAdd"}}
		return nil
	})
	if err != nil {
		t.Fatalf("update combat state: %v", err)
	}

	afterWood, _ := readResourceRow(t, db, state.Player.ID, "wood")
	if afterWood != beforeWood+10 {
		t.Fatalf("expected combat resource writeback, before=%d after=%d", beforeWood, afterWood)
	}
	if amount, ok := readArmyRow(t, db, state.Player.ID, "weiInfantry"); !ok || amount != 3 {
		t.Fatalf("expected changed army row amount=3, amount=%d ok=%v", amount, ok)
	}
	if amount, ok := readArmyRow(t, db, state.Player.ID, "qingZhouArmy"); !ok || amount != 9 {
		t.Fatalf("expected unchanged army row preserved, amount=%d ok=%v", amount, ok)
	}
	afterUnchangedArmyUpdatedAt, ok := readArmyUpdatedAt(t, db, state.Player.ID, "qingZhouArmy")
	if !ok || !afterUnchangedArmyUpdatedAt.Equal(beforeUnchangedArmyUpdatedAt) {
		t.Fatalf("expected unchanged army row not to be rewritten, before=%s after=%s ok=%v", beforeUnchangedArmyUpdatedAt, afterUnchangedArmyUpdatedAt, ok)
	}
	building, ok := readBuildingRow(t, db, state.Player.ID, "wood_camp-1")
	if !ok || building.Level != 1 {
		t.Fatalf("expected combat transaction not to sync building, got %+v ok=%v", building, ok)
	}
	slot, ok := readResourceSlotRow(t, db, state.Player.ID, "combat_slot")
	if !ok || slot.ResourceType != "wood" {
		t.Fatalf("expected combat transaction not to sync resource slot, got %+v ok=%v", slot, ok)
	}
	if _, ok := readRecruitQueueRow(t, db, state.Player.ID, "combat_queue_new"); ok {
		t.Fatalf("expected combat transaction not to insert recruit queue")
	}
	if queue, ok := readRecruitQueueRow(t, db, state.Player.ID, "combat_queue"); !ok || queue.Amount != 1 {
		t.Fatalf("expected original recruit queue unchanged, got %+v ok=%v", queue, ok)
	}
	if count := readBuffCount(t, db, state.Player.ID); count != 1 {
		t.Fatalf("expected combat transaction not to insert/delete buffs, count=%d", count)
	}
	if value, ok := readBuffValue(t, db, state.Player.ID, buffID); !ok || value != 0.25 {
		t.Fatalf("expected combat transaction not to sync buff value, value=%f ok=%v", value, ok)
	}
}

// TestMySQLCombatStateStoresNpcStateOutsidePlayerSnapshot 验证 NPC 战斗状态写入独立权威表，不再刷新 players 主行。
func TestMySQLCombatStateStoresNpcStateOutsidePlayerSnapshot(t *testing.T) {
	repo, db := openResourceAuthorityTestRepository(t)
	_, state := createResourceAuthorityPlayer(t, repo, "combat_npc_state")
	now := time.Now().UTC()
	seedNpcState := &game.NpcState{
		LastRefreshedAt: now.Format(time.RFC3339),
		Cities: []game.NpcCity{{
			ID:                "npc_state_1",
			Name:              "NPC 状态测试",
			Faction:           "wei",
			Resources:         map[string]int{"wood": 100},
			StorageCapacity:   map[string]int{"wood": 1000},
			ProductionPerHour: map[string]int{"wood": 0},
			Army:              []game.ArmyUnit{{UnitType: "weiInfantry", Amount: 10}},
			MaxArmy:           []game.ArmyUnit{{UnitType: "weiInfantry", Amount: 10}},
			ResourceSettledAt: now.Format(time.RFC3339),
			ArmySettledAt:     now.Format(time.RFC3339),
			GeneratedAt:       now.Format(time.RFC3339),
		}},
	}
	if _, err := repo.UpdatePlayerState(state.Player.ID, now, func(state *game.GameState) error {
		state.NpcState = seedNpcState
		state.ServerTime = now.Format(time.RFC3339)
		return nil
	}); err != nil {
		t.Fatalf("seed npc state: %v", err)
	}
	if npcState, _, ok := readNpcStateRow(t, db, state.Player.ID); !ok || len(npcState.Cities) != 1 {
		t.Fatalf("expected npc state authority row after seed, ok=%v state=%+v", ok, npcState)
	}

	beforeJSON, beforeUpdatedAt := readPlayerStorageSnapshot(t, db, state.Player.ID)
	if strings.Contains(string(beforeJSON), "npcState") || strings.Contains(string(beforeJSON), "serverTime") {
		t.Fatalf("expected player snapshot to omit npcState/serverTime, json=%s", string(beforeJSON))
	}
	if _, err := repo.UpdateCombatState(state.Player.ID, game.CombatAssetScope{SkipInventory: true}, now.Add(time.Second), func(state *game.GameState) error {
		if state.NpcState == nil || len(state.NpcState.Cities) != 1 {
			return errors.New("npc state missing in combat transaction")
		}
		state.NpcState.Cities[0].Resources["wood"] = 55
		state.ServerTime = now.Add(time.Second).Format(time.RFC3339)
		return nil
	}); err != nil {
		t.Fatalf("update combat npc state: %v", err)
	}

	afterJSON, afterUpdatedAt := readPlayerStorageSnapshot(t, db, state.Player.ID)
	if !beforeUpdatedAt.Equal(afterUpdatedAt) || string(beforeJSON) != string(afterJSON) {
		t.Fatalf("expected combat npc update not to refresh players row, before=%s/%s after=%s/%s", beforeUpdatedAt, beforeJSON, afterUpdatedAt, afterJSON)
	}
	npcState, _, ok := readNpcStateRow(t, db, state.Player.ID)
	if !ok || len(npcState.Cities) != 1 || npcState.Cities[0].Resources["wood"] != 55 {
		t.Fatalf("expected npc state authority row to update wood=55, ok=%v state=%+v", ok, npcState)
	}
	got, err := repo.GetState(state.Player.ID)
	if err != nil {
		t.Fatalf("get state: %v", err)
	}
	if got.NpcState == nil || got.NpcState.Cities[0].Resources["wood"] != 55 {
		t.Fatalf("expected GetState to overlay npc authority, got %+v", got.NpcState)
	}
}

// TestMySQLCombatStateStoresCityGoldOutsidePlayerSnapshot 验证 NPC 战斗溢出城金只写货币权威表，不刷新 players 主行。
func TestMySQLCombatStateStoresCityGoldOutsidePlayerSnapshot(t *testing.T) {
	repo, db := openResourceAuthorityTestRepository(t)
	_, state := createResourceAuthorityPlayer(t, repo, "combat_city_gold")
	beforeJSON, beforeUpdatedAt := readPlayerStorageSnapshot(t, db, state.Player.ID)

	if _, err := repo.UpdateCombatState(state.Player.ID, game.CombatAssetScope{SkipInventory: true}, time.Now(), func(state *game.GameState) error {
		state.CityGold += 9
		return nil
	}); err != nil {
		t.Fatalf("update combat city gold: %v", err)
	}

	afterJSON, afterUpdatedAt := readPlayerStorageSnapshot(t, db, state.Player.ID)
	if !beforeUpdatedAt.Equal(afterUpdatedAt) || string(beforeJSON) != string(afterJSON) {
		t.Fatalf("expected combat city gold update not to refresh players row, before=%s/%s after=%s/%s", beforeUpdatedAt, beforeJSON, afterUpdatedAt, afterJSON)
	}
	if got := readCurrencyCityGold(t, db, state.Player.ID); got != 9 {
		t.Fatalf("expected city gold authority=9, got %d", got)
	}
	got, err := repo.GetState(state.Player.ID)
	if err != nil {
		t.Fatalf("get state: %v", err)
	}
	if int(got.CityGold) != 9 {
		t.Fatalf("expected GetState city gold=9, got %d", got.CityGold)
	}
}

// TestMySQLGrantCityGoldRewardDoesNotRefreshPlayerSnapshot 验证纯城金奖励只写货币权威表，不刷新 players 主行。
func TestMySQLGrantCityGoldRewardDoesNotRefreshPlayerSnapshot(t *testing.T) {
	repo, db := openResourceAuthorityTestRepository(t)
	_, state := createResourceAuthorityPlayer(t, repo, "reward_city_gold")
	service := game.NewServiceWithRepository(repo)
	beforeJSON, beforeUpdatedAt := readPlayerStorageSnapshot(t, db, state.Player.ID)

	result, err := service.GrantRewards(state.Player.ID, []game.Reward{
		{Type: game.RewardTypeCityGold, ID: game.RewardTypeCityGold, Amount: 13},
	}, game.RewardGrantContext{RefType: "test_city_gold", RefID: "grant_1"})
	if err != nil {
		t.Fatalf("grant city gold: %v", err)
	}
	if int(result.State.CityGold) != 13 || readCurrencyCityGold(t, db, state.Player.ID) != 13 {
		t.Fatalf("expected city gold authority/state=13, state=%d", result.State.CityGold)
	}
	afterJSON, afterUpdatedAt := readPlayerStorageSnapshot(t, db, state.Player.ID)
	if !beforeUpdatedAt.Equal(afterUpdatedAt) || string(beforeJSON) != string(afterJSON) {
		t.Fatalf("expected city gold reward not to refresh players row, before=%s/%s after=%s/%s", beforeUpdatedAt, beforeJSON, afterUpdatedAt, afterJSON)
	}
}

// TestMySQLGetNpcCitiesDoesNotRefreshPlayerSnapshot 验证 NPC 列表刷新只写 NPC 状态权威表，不刷新 players 主行。
func TestMySQLGetNpcCitiesDoesNotRefreshPlayerSnapshot(t *testing.T) {
	repo, db := openResourceAuthorityTestRepository(t)
	_, state := createResourceAuthorityPlayer(t, repo, "npc_state_service")
	service := game.NewServiceWithRepository(repo)

	beforeJSON, beforeUpdatedAt := readPlayerStorageSnapshot(t, db, state.Player.ID)
	npcState, err := service.GetNpcCities(state.Player.ID)
	if err != nil {
		t.Fatalf("GetNpcCities: %v", err)
	}
	if len(npcState.Cities) == 0 {
		t.Fatalf("expected generated npc cities, got %+v", npcState)
	}
	afterJSON, afterUpdatedAt := readPlayerStorageSnapshot(t, db, state.Player.ID)
	if !beforeUpdatedAt.Equal(afterUpdatedAt) || string(beforeJSON) != string(afterJSON) {
		t.Fatalf("expected GetNpcCities not to refresh players row, before=%s/%s after=%s/%s", beforeUpdatedAt, beforeJSON, afterUpdatedAt, afterJSON)
	}
	storedNpcState, _, ok := readNpcStateRow(t, db, state.Player.ID)
	if !ok || len(storedNpcState.Cities) != len(npcState.Cities) {
		t.Fatalf("expected npc state authority row, ok=%v stored=%+v result=%+v", ok, storedNpcState, npcState)
	}
}

// TestMySQLCombatStateScopesArmyToRequestedUnits 验证战斗事务只加载并锁定作用域内兵种。
func TestMySQLCombatStateScopesArmyToRequestedUnits(t *testing.T) {
	repo, db := openResourceAuthorityTestRepository(t)
	_, state := createResourceAuthorityPlayer(t, repo, "combat_army_scope")
	now := time.Now().UTC()
	seedUpdatedAt := now.Add(-time.Hour)
	if _, err := db.Exec(
		`INSERT INTO player_army_units (player_id, unit_type, amount, updated_at)
		 VALUES (?, 'weiInfantry', 10, ?), (?, 'qingZhouArmy', 20, ?)
		 ON DUPLICATE KEY UPDATE amount = VALUES(amount), updated_at = VALUES(updated_at)`,
		state.Player.ID,
		seedUpdatedAt,
		state.Player.ID,
		seedUpdatedAt,
	); err != nil {
		t.Fatalf("seed army: %v", err)
	}
	beforeUnscopedUpdatedAt, ok := readArmyUpdatedAt(t, db, state.Player.ID, "qingZhouArmy")
	if !ok {
		t.Fatalf("expected unscoped army row")
	}
	sawUnscoped := false
	result, err := repo.UpdateCombatState(state.Player.ID, game.CombatAssetScope{UnitTypes: []string{"weiInfantry"}}, time.Now(), func(state *game.GameState) error {
		for index := range state.Army {
			switch state.Army[index].UnitType {
			case "weiInfantry":
				state.Army[index].Amount = 7
			case "qingZhouArmy":
				sawUnscoped = true
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("update scoped combat state: %v", err)
	}
	if sawUnscoped {
		t.Fatalf("expected scoped combat transaction not to load unrequested army unit")
	}
	if amount, ok := readArmyRow(t, db, state.Player.ID, "weiInfantry"); !ok || amount != 7 {
		t.Fatalf("expected scoped army row updated, amount=%d ok=%v", amount, ok)
	}
	if amount, ok := readArmyRow(t, db, state.Player.ID, "qingZhouArmy"); !ok || amount != 20 {
		t.Fatalf("expected unscoped army row preserved, amount=%d ok=%v", amount, ok)
	}
	afterUnscopedUpdatedAt, ok := readArmyUpdatedAt(t, db, state.Player.ID, "qingZhouArmy")
	if !ok || !afterUnscopedUpdatedAt.Equal(beforeUnscopedUpdatedAt) {
		t.Fatalf("expected unscoped army row not to be rewritten, before=%s after=%s ok=%v", beforeUnscopedUpdatedAt, afterUnscopedUpdatedAt, ok)
	}
	if len(result.Army) < 2 {
		t.Fatalf("expected response state to reload full army after scoped commit, got %+v", result.Army)
	}
}

// TestMySQLCombatStateCanSkipInventoryForScopedScout 验证侦查类战斗可跳过事务内背包锁，提交后仍返回权威背包。
func TestMySQLCombatStateCanSkipInventoryForScopedScout(t *testing.T) {
	repo, db := openResourceAuthorityTestRepository(t)
	_, state := createResourceAuthorityPlayer(t, repo, "combat_skip_inventory")
	now := time.Now().UTC()
	if _, err := db.Exec(
		`INSERT INTO player_inventory (player_id, slot_id, item_id, amount, obtained_at, updated_at)
		 VALUES (?, 'slot_0001', 'general_exp_small', 3, ?, ?)
		 ON DUPLICATE KEY UPDATE item_id = VALUES(item_id), amount = VALUES(amount), updated_at = VALUES(updated_at)`,
		state.Player.ID,
		now,
		now,
	); err != nil {
		t.Fatalf("seed inventory: %v", err)
	}
	if _, err := db.Exec(
		`INSERT INTO player_army_units (player_id, unit_type, amount, updated_at)
		 VALUES (?, 'weiScout', 5, ?)
		 ON DUPLICATE KEY UPDATE amount = VALUES(amount), updated_at = VALUES(updated_at)`,
		state.Player.ID,
		now,
	); err != nil {
		t.Fatalf("seed scout army: %v", err)
	}

	sawInventory := false
	result, err := repo.UpdateCombatState(state.Player.ID, game.CombatAssetScope{UnitTypes: []string{"weiScout"}, SkipInventory: true}, time.Now(), func(state *game.GameState) error {
		if len(state.Inventory) > 0 || len(state.InventorySlots) > 0 {
			sawInventory = true
		}
		state.Inventory = map[string]game.ItemStack{
			"resource_pack_small": {ItemID: "resource_pack_small", Amount: 99},
		}
		state.InventorySlots = []game.ItemStack{{SlotID: "slot_0002", ItemID: "resource_pack_small", Amount: 99}}
		return nil
	})
	if err != nil {
		t.Fatalf("update combat state skip inventory: %v", err)
	}
	if sawInventory {
		t.Fatalf("expected skip inventory scope not to load inventory in transaction")
	}
	if amount, ok := readInventoryRow(t, db, state.Player.ID, "general_exp_small"); !ok || amount != 3 {
		t.Fatalf("expected original inventory row preserved, amount=%d ok=%v", amount, ok)
	}
	if amount, ok := readInventoryRow(t, db, state.Player.ID, "resource_pack_small"); ok || amount != 0 {
		t.Fatalf("expected skipped inventory mutation not to be written, amount=%d ok=%v", amount, ok)
	}
	if got := result.Inventory["general_exp_small"].Amount; got != 3 {
		t.Fatalf("expected response to reload full inventory after skipped transaction, got %d", got)
	}
	if got := result.Inventory["resource_pack_small"].Amount; got != 0 {
		t.Fatalf("expected response not to expose skipped inventory mutation, got %d", got)
	}
}

// TestMySQLCombatStateScopesInventoryToCandidateItems 验证 NPC 战斗背包写入只触碰候选掉落物品。
func TestMySQLCombatStateScopesInventoryToCandidateItems(t *testing.T) {
	repo, db := openResourceAuthorityTestRepository(t)
	_, state := createResourceAuthorityPlayer(t, repo, "combat_inventory_scope")
	now := time.Now().UTC()
	seedUpdatedAt := now.Add(-time.Hour)
	if _, err := db.Exec(
		`INSERT INTO player_inventory (player_id, slot_id, item_id, amount, obtained_at, updated_at)
		 VALUES (?, 'slot_0001', 'general_exp_small', 1, ?, ?),
		        (?, 'slot_0002', 'resource_pack_small', 2, ?, ?)
		 ON DUPLICATE KEY UPDATE item_id = VALUES(item_id), amount = VALUES(amount), updated_at = VALUES(updated_at)`,
		state.Player.ID,
		seedUpdatedAt,
		seedUpdatedAt,
		state.Player.ID,
		seedUpdatedAt,
		seedUpdatedAt,
	); err != nil {
		t.Fatalf("seed inventory: %v", err)
	}
	beforeUnscopedUpdatedAt, ok := readInventoryUpdatedAt(t, db, state.Player.ID, "resource_pack_small")
	if !ok {
		t.Fatalf("expected unscoped inventory row")
	}

	result, err := repo.UpdateCombatState(state.Player.ID, game.CombatAssetScope{InventoryItemIDs: []string{"general_exp_small"}}, time.Now(), func(state *game.GameState) error {
		for index := range state.InventorySlots {
			switch state.InventorySlots[index].ItemID {
			case "general_exp_small":
				state.InventorySlots[index].Amount = 3
			case "resource_pack_small":
				state.InventorySlots[index].Amount = 99
			}
		}
		state.Inventory = map[string]game.ItemStack{
			"general_exp_small":   {ItemID: "general_exp_small", Amount: 3},
			"resource_pack_small": {ItemID: "resource_pack_small", Amount: 99},
		}
		return nil
	})
	if err != nil {
		t.Fatalf("update combat state scoped inventory: %v", err)
	}
	if amount, ok := readInventoryRow(t, db, state.Player.ID, "general_exp_small"); !ok || amount != 3 {
		t.Fatalf("expected scoped inventory item updated, amount=%d ok=%v", amount, ok)
	}
	if amount, ok := readInventoryRow(t, db, state.Player.ID, "resource_pack_small"); !ok || amount != 2 {
		t.Fatalf("expected unscoped inventory item preserved, amount=%d ok=%v", amount, ok)
	}
	afterUnscopedUpdatedAt, ok := readInventoryUpdatedAt(t, db, state.Player.ID, "resource_pack_small")
	if !ok || !afterUnscopedUpdatedAt.Equal(beforeUnscopedUpdatedAt) {
		t.Fatalf("expected unscoped inventory row not to be rewritten, before=%s after=%s ok=%v", beforeUnscopedUpdatedAt, afterUnscopedUpdatedAt, ok)
	}
	if got := result.Inventory["general_exp_small"].Amount; got != 3 {
		t.Fatalf("expected response to reload scoped inventory item amount=3, got %d", got)
	}
	if got := result.Inventory["resource_pack_small"].Amount; got != 2 {
		t.Fatalf("expected response to reload unscoped inventory item amount=2, got %d", got)
	}
}

// TestMySQLCombatStateScopesGeneralsToRequestedIDs 验证战斗事务只加载并锁定参战武将。
func TestMySQLCombatStateScopesGeneralsToRequestedIDs(t *testing.T) {
	repo, db := openResourceAuthorityTestRepository(t)
	_, state := createResourceAuthorityPlayer(t, repo, "combat_general_scope")
	now := time.Now().UTC()
	seedUpdatedAt := now.Add(-time.Hour)
	if _, err := db.Exec(
		`INSERT INTO player_generals (player_id, general_id, faction, level, exp, stats_json, acquired_at, updated_at)
		 VALUES (?, 'caocao', 'wei', 1, 0, JSON_OBJECT(), ?, ?),
		        (?, 'xuchu', 'wei', 1, 77, JSON_OBJECT(), ?, ?)
		 ON DUPLICATE KEY UPDATE exp = VALUES(exp), updated_at = VALUES(updated_at)`,
		state.Player.ID,
		seedUpdatedAt,
		seedUpdatedAt,
		state.Player.ID,
		seedUpdatedAt,
		seedUpdatedAt,
	); err != nil {
		t.Fatalf("seed generals: %v", err)
	}
	if _, err := db.Exec(
		`INSERT INTO player_general_assignments (player_id, assignment_id, general_id, assignment_slot, module_id, status, assigned_at, updated_at)
		 VALUES (?, 'main', 'caocao', 'main', '', 'active', ?, ?)
		 ON DUPLICATE KEY UPDATE general_id = VALUES(general_id), assignment_slot = VALUES(assignment_slot), status = VALUES(status), updated_at = VALUES(updated_at)`,
		state.Player.ID,
		seedUpdatedAt,
		seedUpdatedAt,
	); err != nil {
		t.Fatalf("seed general assignment: %v", err)
	}
	beforeUnscopedUpdatedAt, ok := readGeneralUpdatedAt(t, db, state.Player.ID, "xuchu")
	if !ok {
		t.Fatalf("expected unscoped general row")
	}
	sawUnscoped := false
	result, err := repo.UpdateCombatState(state.Player.ID, game.CombatAssetScope{GeneralIDs: []string{"caocao"}}, time.Now(), func(state *game.GameState) error {
		for index := range state.Generals {
			switch state.Generals[index].ID {
			case "caocao":
				state.Generals[index].Exp = 123
			case "xuchu":
				sawUnscoped = true
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("update scoped combat generals: %v", err)
	}
	if sawUnscoped {
		t.Fatalf("expected scoped combat transaction not to load unrequested general")
	}
	_, exp, ok := readGeneralRow(t, db, state.Player.ID, "caocao")
	if !ok || exp != 123 {
		t.Fatalf("expected scoped general row updated, exp=%d ok=%v", exp, ok)
	}
	_, unscopedExp, ok := readGeneralRow(t, db, state.Player.ID, "xuchu")
	if !ok || unscopedExp != 77 {
		t.Fatalf("expected unscoped general row preserved, exp=%d ok=%v", unscopedExp, ok)
	}
	afterUnscopedUpdatedAt, ok := readGeneralUpdatedAt(t, db, state.Player.ID, "xuchu")
	if !ok || !afterUnscopedUpdatedAt.Equal(beforeUnscopedUpdatedAt) {
		t.Fatalf("expected unscoped general row not to be rewritten, before=%s after=%s ok=%v", beforeUnscopedUpdatedAt, afterUnscopedUpdatedAt, ok)
	}
	if len(result.Generals) < 2 {
		t.Fatalf("expected response state to reload full general roster after scoped commit, got %+v", result.Generals)
	}
}

// TestMySQLGetStateUsesPlayerArmyAuthority 验证读取玩家状态时兵力以 player_army_units 为准。
func TestMySQLGetStateUsesPlayerArmyAuthority(t *testing.T) {
	repo, db := openResourceAuthorityTestRepository(t)
	_, state := createResourceAuthorityPlayer(t, repo, "army_authority")

	if _, err := db.Exec(
		`INSERT INTO player_army_units (player_id, unit_type, amount, updated_at)
		 VALUES (?, 'weiInfantry', 77, UTC_TIMESTAMP(6))
		 ON DUPLICATE KEY UPDATE amount = VALUES(amount), updated_at = VALUES(updated_at)`,
		state.Player.ID,
	); err != nil {
		t.Fatalf("update player_army_units: %v", err)
	}
	overwriteSnapshotArmy(t, db, state.Player.ID, "weiInfantry", 1)

	got, err := repo.GetState(state.Player.ID)
	if err != nil {
		t.Fatalf("get state: %v", err)
	}
	for _, unit := range got.Army {
		if unit.UnitType == "weiInfantry" {
			if unit.Amount != 77 {
				t.Fatalf("expected authoritative weiInfantry=77, got %d", unit.Amount)
			}
			return
		}
	}
	t.Fatalf("expected weiInfantry in authoritative army")
}

// TestMySQLRecruitRefreshesRecruitQueueAuthority 验证征兵开始会写入征兵队列权威表。
func TestMySQLRecruitRefreshesRecruitQueueAuthority(t *testing.T) {
	repo, db := openResourceAuthorityTestRepository(t)
	_, state := createResourceAuthorityPlayer(t, repo, "recruit_queue")
	if err := game.LoadUnitsConfig("../../../config/units"); err != nil {
		t.Fatalf("load units config: %v", err)
	}
	service := game.NewServiceWithRepository(repo)

	next, err := service.Recruit(state.Player.ID, "qingZhouArmy", 1)
	if err != nil {
		t.Fatalf("recruit: %v", err)
	}
	if len(next.RecruitQueues) != 1 {
		t.Fatalf("expected one recruit queue, got %+v", next.RecruitQueues)
	}
	queue := next.RecruitQueues[0]
	row, ok := readRecruitQueueRow(t, db, state.Player.ID, queue.ID)
	if !ok {
		t.Fatalf("expected recruit queue row")
	}
	if row.UnitType != queue.UnitType || row.Amount != queue.Amount || row.EndsAt != queue.EndsAt {
		t.Fatalf("expected recruit queue table to match state, table=%+v state=%+v", row, queue)
	}
}

// TestMySQLInstantCompleteRecruitRefreshesArmyAuthority 验证极速完成征兵会写入兵力权威表并删除队列，且轻量快照不再保存兵力。
func TestMySQLInstantCompleteRecruitRefreshesArmyAuthority(t *testing.T) {
	repo, db := openResourceAuthorityTestRepository(t)
	_, state := createResourceAuthorityPlayer(t, repo, "recruit_complete")
	if err := game.LoadUnitsConfig("../../../config/units"); err != nil {
		t.Fatalf("load units config: %v", err)
	}
	service := game.NewServiceWithRepository(repo)

	next, err := service.Recruit(state.Player.ID, "qingZhouArmy", 1)
	if err != nil {
		t.Fatalf("recruit: %v", err)
	}
	queueID := next.RecruitQueues[0].ID
	if _, err := repo.UpdatePlayerState(state.Player.ID, time.Now(), func(state *game.GameState) error {
		state.CityGold = 1000
		return nil
	}); err != nil {
		t.Fatalf("seed city gold: %v", err)
	}
	completed, err := service.InstantCompleteRecruit(state.Player.ID, queueID)
	if err != nil {
		t.Fatalf("instant complete recruit: %v", err)
	}
	if len(completed.RecruitQueues) != 0 {
		t.Fatalf("expected recruit queues empty after instant complete, got %+v", completed.RecruitQueues)
	}
	amount, ok := readArmyRow(t, db, state.Player.ID, "qingZhouArmy")
	if !ok || amount != 1 {
		t.Fatalf("expected army table qingZhouArmy=1, got amount=%d ok=%v", amount, ok)
	}
	if _, ok := readRecruitQueueRow(t, db, state.Player.ID, queueID); ok {
		t.Fatalf("expected recruit queue row deleted")
	}
	snapshot, ok := readSnapshotArmy(t, db, state.Player.ID, "qingZhouArmy")
	if ok || snapshot != 0 {
		t.Fatalf("expected no state_json army snapshot, got amount=%d ok=%v", snapshot, ok)
	}
}
