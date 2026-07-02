package api

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// 本文件验证世界地图接口实现和中文 OpenAPI 文档保持一致。

// TestOpenAPIWorldMapStaticConsistency 锁定世界地图玩家端、GM 端和迁移结果文档。
func TestOpenAPIWorldMapStaticConsistency(t *testing.T) {
	root := filepath.Join("..", "..", "..", "..")
	mapPaths := readRepoFile(t, root, "docs", "接口文档", "openapi", "paths", "map.yaml")
	adminPaths := readRepoFile(t, root, "docs", "接口文档", "openapi", "paths", "admin.yaml")
	mapSchemas := readRepoFile(t, root, "docs", "接口文档", "openapi", "schemas", "map.yaml")
	handlers := readRepoFile(t, root, "go", "internal", "transport", "api", "handlers_world_map.go")
	bundled := readRepoFile(t, root, "docs", "接口文档", "openapi打包.yaml")

	assertContainsAll(t, "玩家地图路径", mapPaths,
		"/api/v1/world-map/view:",
		"operationId: getWorldMapView",
		"security:",
		"- bearerAuth: []",
		"'401':",
		"'403':",
		"当前账号不拥有该玩家存档",
		"当前账号不拥有查看者玩家存档",
		"maximum: 100",
		"接口不返回兵力、资源、武将或驻防详情",
		"/api/v1/world-map/targets/player_city/{playerId}:",
		"operationId: getWorldMapPlayerCityTarget",
		"第一版不返回兵力、资源、武将或驻防详情",
	)
	assertContainsAll(t, "GM 地图路径", adminPaths,
		"/api/v1/admin/world-map/occupancy:",
		"operationId: adminGetWorldMapOccupancy",
		"security:",
		"- adminToken: []",
		"/api/v1/admin/world-map/positions/check:",
		"operationId: adminCheckWorldCoordinate",
		"保存玩家世界坐标前",
		"/api/v1/admin/world-map/positions/{playerId}:",
		"operationId: adminGetWorldPosition",
		"operationId: adminUpdateWorldPosition",
		"坐标范围为 0-99",
	)
	assertContainsAll(t, "地图 schema", mapSchemas,
		"WorldPosition:",
		"WorldMapOccupancyStats:",
		"WorldMapCoordinateCheck:",
		"WorldMapTarget:",
		"WorldMapViewResponse:",
		"WorldMapMigrationResult:",
		"- conflicts",
		"conflictDetails:",
		"description: 旧坐标被占用后改分配到附近空格的玩家数量。",
		"description: '旧坐标冲突玩家的迁移明细，格式为 playerId: (oldX,oldY) -> (newX,newY)。'",
	)
	assertContainsAll(t, "地图目标 schema", worldMapTargetSchema(mapSchemas),
		"enum: [self, ally, normal, protected, truce, attackable, unavailable]",
		"canScout:",
		"canAttack:",
		"canPlunder:",
		"canReinforce:",
		"scoutReason:",
		"attackReason:",
		"plunderReason:",
		"reinforceReason:",
	)
	assertNotContainsAny(t, "地图目标 schema", worldMapTargetSchema(mapSchemas),
		"totalArmy",
		"resources",
		"generals",
		"garrison",
	)
	assertContainsAll(t, "世界地图处理器鉴权", handlers,
		"func (h *Handlers) WorldMapView",
		"if !h.requireOwnership(w, r, playerID)",
		"func (h *Handlers) WorldMapPlayerCityTarget",
		"if !h.requireOwnership(w, r, viewerID)",
	)
	assertContainsAll(t, "OpenAPI 打包文档", bundled,
		"/api/v1/world-map/view:",
		"/api/v1/world-map/targets/player_city/{playerId}:",
		"/api/v1/admin/world-map/occupancy:",
		"/api/v1/admin/world-map/positions/check:",
		"/api/v1/admin/world-map/positions/{playerId}:",
		"securitySchemes:",
		"bearerAuth:",
		"adminToken:",
		"name: X-Admin-Token",
		"WorldMapCoordinateCheck:",
		"WorldMapMigrationResult:",
		"- conflicts",
		"conflictDetails:",
	)
}

// readRepoFile 读取仓库内文档文件并在失败时终止测试。
func readRepoFile(t *testing.T, root string, parts ...string) string {
	t.Helper()
	path := filepath.Join(append([]string{root}, parts...)...)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("读取文件失败 %s: %v", path, err)
	}
	return string(data)
}

// assertContainsAll 验证文本包含所有指定片段。
func assertContainsAll(t *testing.T, label string, source string, fragments ...string) {
	t.Helper()
	for _, fragment := range fragments {
		if !strings.Contains(source, fragment) {
			t.Fatalf("%s 缺少片段 %q", label, fragment)
		}
	}
}

// assertNotContainsAny 验证文本不包含任何指定片段。
func assertNotContainsAny(t *testing.T, label string, source string, fragments ...string) {
	t.Helper()
	for _, fragment := range fragments {
		if strings.Contains(source, fragment) {
			t.Fatalf("%s 不应包含片段 %q", label, fragment)
		}
	}
}

// worldMapTargetSchema 截取 WorldMapTarget schema，避免其它 schema 字段干扰。
func worldMapTargetSchema(source string) string {
	start := strings.Index(source, "WorldMapTarget:")
	end := strings.Index(source, "WorldMapViewResponse:")
	if start < 0 || end < 0 || end <= start {
		return source
	}
	return source[start:end]
}
