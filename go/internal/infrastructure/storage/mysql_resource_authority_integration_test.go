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
		return nil
	})
	if err != nil {
		t.Fatalf("seed item: %v", err)
	}
	service := game.NewServiceWithRepository(repo)

	if _, err := service.UseItem(state.Player.ID, "resource_pack_small", 1); err != nil {
		t.Fatalf("use resource pack: %v", err)
	}
	amount, _ := readResourceRow(t, db, state.Player.ID, "wood")
	if amount != 2200 {
		t.Fatalf("expected wood=2200 after resource pack, got %d", amount)
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
