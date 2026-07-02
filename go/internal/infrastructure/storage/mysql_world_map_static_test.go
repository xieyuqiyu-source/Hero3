// 本文件测试 MySQL 世界地图仓储的关键一致性约束。
package storage

import (
	"os"
	"strings"
	"testing"
)

// TestMySQLWorldMapWritesCheckPlayerExists 防止坐标写入绕过玩家存在校验。
func TestMySQLWorldMapWritesCheckPlayerExists(t *testing.T) {
	content, err := os.ReadFile("mysql_world_map.go")
	if err != nil {
		t.Fatalf("read mysql_world_map.go: %v", err)
	}
	source := string(content)
	for _, fn := range []string{"EnsureWorldPosition", "AssignWorldPosition"} {
		marker := "func (r *MySQLRepository) " + fn
		start := strings.Index(source, marker)
		if start < 0 {
			t.Fatalf("%s not found", fn)
		}
		section := source[start:]
		if next := strings.Index(section[len(marker):], "\nfunc "); next >= 0 {
			section = section[:len(marker)+next]
		}
		if !strings.Contains(section, "r.GetAccountIDByPlayerID(playerID)") {
			t.Fatalf("%s must check player existence before writing world coordinates", fn)
		}
	}
}

// TestMySQLWorldMapUsesSharedCoordinateCandidates 防止线上仓储和内存仓储坐标冲突顺序分叉。
func TestMySQLWorldMapUsesSharedCoordinateCandidates(t *testing.T) {
	content, err := os.ReadFile("mysql_world_map.go")
	if err != nil {
		t.Fatalf("read mysql_world_map.go: %v", err)
	}
	source := string(content)
	ensureSection := worldMapStorageFunctionSection(t, source, "EnsureWorldPosition")
	if !strings.Contains(ensureSection, "game.WorldMapCoordinateCandidates(start.X, start.Y)") {
		t.Fatalf("mysql world map repository must use shared world coordinate candidate order in EnsureWorldPosition")
	}
	if !strings.Contains(ensureSection, "return game.WorldPosition{}, game.ErrWorldMapFull") {
		t.Fatalf("mysql world map repository must return ErrWorldMapFull after all coordinate candidates are exhausted")
	}
	if strings.Contains(source, "manhattanForStorage") {
		t.Fatalf("mysql world map repository must not keep a separate manhattan search order")
	}
	preferredSection := worldMapStorageFunctionSection(t, source, "worldMapPreferredCoordinateForStorage")
	if !strings.Contains(preferredSection, "game.LegacyWorldCoordinateForPlayer(playerID)") {
		t.Fatalf("mysql world map repository must reuse app legacy coordinate mapping")
	}
}

// TestMySQLWorldMapViewUsesBoundedProjection 防止地图视野退化为全服账号和资产聚合查询。
func TestMySQLWorldMapViewUsesBoundedProjection(t *testing.T) {
	content, err := os.ReadFile("mysql_world_map.go")
	if err != nil {
		t.Fatalf("read mysql_world_map.go: %v", err)
	}
	section := worldMapStorageFunctionSection(t, string(content), "ListWorldMapPlayerCities")
	for _, required := range []string{
		"FROM player_world_positions w",
		"INNER JOIN players p ON p.id = w.player_id",
		"w.world_id = ?",
		"w.x BETWEEN ? AND ?",
		"w.y BETWEEN ? AND ?",
	} {
		if !strings.Contains(section, required) {
			t.Fatalf("bounded world map projection missing %q", required)
		}
	}
	for _, forbidden := range []string{"FROM accounts", "player_army_units", "state_json"} {
		if strings.Contains(section, forbidden) {
			t.Fatalf("bounded world map projection must not read %q", forbidden)
		}
	}
}

// TestMySQLWorldMapSchemaKeepsCoordinateConstraints 防止权威坐标表丢失唯一键和查询索引。
func TestMySQLWorldMapSchemaKeepsCoordinateConstraints(t *testing.T) {
	content, err := os.ReadFile("mysql.go")
	if err != nil {
		t.Fatalf("read mysql.go: %v", err)
	}
	source := string(content)
	start := strings.Index(source, "CREATE TABLE IF NOT EXISTS player_world_positions")
	if start < 0 {
		t.Fatalf("player_world_positions schema not found")
	}
	section := source[start:]
	if next := strings.Index(section, ") ENGINE=InnoDB"); next >= 0 {
		section = section[:next]
	}
	for _, required := range []string{
		"PRIMARY KEY (player_id)",
		"UNIQUE KEY uk_player_world_positions_world_xy (world_id, x, y)",
		"INDEX idx_player_world_positions_world_xy (world_id, x, y)",
		"INDEX idx_player_world_positions_world_player (world_id, player_id)",
		"FOREIGN KEY (player_id) REFERENCES players(id)",
		"ON DELETE CASCADE",
	} {
		if !strings.Contains(section, required) {
			t.Fatalf("player_world_positions schema missing %q", required)
		}
	}
}

// worldMapStorageFunctionSection 截取 MySQL 世界地图仓储中的单个函数。
func worldMapStorageFunctionSection(t *testing.T, source string, name string) string {
	t.Helper()
	marker := "func (r *MySQLRepository) "
	start := strings.Index(source, marker+name)
	if start < 0 {
		marker = "func "
		start = strings.Index(source, marker+name)
	}
	if start < 0 {
		t.Fatalf("%s not found", name)
	}
	section := source[start:]
	if next := strings.Index(section[len(marker+name):], "\nfunc "); next >= 0 {
		section = section[:len(marker+name)+next]
	}
	return section
}
