// 本文件测试 MySQL 建筑权威表加载后的核心建筑兼容补齐。
package storage

import (
	"testing"

	"hero3/internal/app/game"
)

// TestApplyAuthoritativeBuildingsEnsuresCoreBuildings 验证旧权威表缺少新核心建筑时视图仍能补齐。
func TestApplyAuthoritativeBuildingsEnsuresCoreBuildings(t *testing.T) {
	state := game.GameState{}
	err := applyAuthoritativeBuildings(&state, []game.Building{
		{ID: "infantry_camp-1", Type: "infantry_camp", Level: 1},
		{ID: "cavalry_camp-1", Type: "cavalry_camp", Level: 1},
		{ID: "weapon_bureau-1", Type: "weapon_bureau", Level: 1},
		{ID: "armor_bureau-1", Type: "armor_bureau", Level: 1},
		{ID: "construction_bureau-1", Type: "construction_bureau", Level: 1},
	}, true)
	if err != nil {
		t.Fatalf("applyAuthoritativeBuildings failed: %v", err)
	}
	expected := map[string]string{
		"administration-1": "administration",
		"relay_station-1":  "relay_station",
		"city_wall-1":      "city_wall",
	}
	for id, buildingType := range expected {
		found := false
		for _, building := range state.Buildings {
			if building.Type == buildingType && building.ID == id {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("expected %s to be projected, got %+v", buildingType, state.Buildings)
		}
	}
}
