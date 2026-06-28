package game

import (
	"testing"
	"time"
)

// 本文件验证建筑注册表与核心建筑补齐入口。

func TestBuildingRegistryExposesConfigs(t *testing.T) {
	config, exists := GetBuildingConfig("warehouse")
	if !exists {
		t.Fatal("expected warehouse config to exist")
	}
	if config.Type != "warehouse" {
		t.Fatalf("expected warehouse type, got %q", config.Type)
	}
	if !BuildingTypeRegistered("weapon_bureau") {
		t.Fatal("expected weapon_bureau to be registered")
	}
	if BuildingTypeRegistered("unknown_building") {
		t.Fatal("expected unknown_building to be unregistered")
	}
}

func TestListBuildingConfigsReturnsCopy(t *testing.T) {
	configs := ListBuildingConfigs()
	delete(configs, "warehouse")
	if !BuildingTypeRegistered("warehouse") {
		t.Fatal("expected mutating listed configs not to change active registry")
	}
}

func TestEnsureCoreBuildingsAppendsMissingBuildings(t *testing.T) {
	state := GameState{}
	if !ensureCoreBuildings(&state) {
		t.Fatal("expected missing core buildings to be appended")
	}
	if len(state.Buildings) != 8 {
		t.Fatalf("expected 8 core buildings, got %d", len(state.Buildings))
	}
	if findBuildingByID(&state, "administration-1") == nil {
		t.Fatalf("expected administration core building, got %+v", state.Buildings)
	}
	if findBuildingByID(&state, "relay_station-1") == nil {
		t.Fatalf("expected relay station core building, got %+v", state.Buildings)
	}
	if findBuildingByID(&state, "city_wall-1") == nil {
		t.Fatalf("expected city wall core building, got %+v", state.Buildings)
	}
	if ensureCoreBuildings(&state) {
		t.Fatal("expected second ensure not to append duplicate buildings")
	}
}

func TestConstructionBureauLevelOneDoesNotUnlockResourceSlots(t *testing.T) {
	state := GameState{
		Buildings: []Building{
			{ID: "construction_bureau-1", Type: "construction_bureau", Level: 1},
			{ID: "construction_resource_slot-1", Type: "wood_camp", Level: 1},
		},
		ResourceSlots: []ResourceSlot{
			{ID: "construction_resource_slot-1", BuildingID: "construction_resource_slot-1", ResourceType: "wood"},
		},
	}

	if !ApplyConstructionBureauResourceSlots(&state, time.Now()) {
		t.Fatal("expected stale construction resource slot to be pruned")
	}
	if countConstructionResourceSlots(state.Buildings) != 0 {
		t.Fatalf("expected level 1 construction bureau not to keep resource slots, got %+v", state.Buildings)
	}
	if len(state.ResourceSlots) != 0 {
		t.Fatalf("expected stale construction resource slot rows to be pruned, got %+v", state.ResourceSlots)
	}
}

func TestConstructionBureauUnlocksFullResourceSlotRounds(t *testing.T) {
	state := GameState{
		Buildings: []Building{
			{ID: "construction_bureau-1", Type: "construction_bureau", Level: 25},
		},
	}

	if !ApplyConstructionBureauResourceSlots(&state, time.Now()) {
		t.Fatal("expected construction bureau to add resource slots")
	}

	expectedTypes := []string{
		"wood_camp", "stone_quarry", "iron_mine", "farm",
		"wood_camp", "stone_quarry", "iron_mine", "farm",
		"wood_camp", "stone_quarry", "iron_mine", "farm",
		"wood_camp", "stone_quarry", "iron_mine", "farm",
		"wood_camp", "stone_quarry", "iron_mine", "farm",
	}
	for i, expectedType := range expectedTypes {
		building := findBuildingByID(&state, constructionBureauResourceSlotID(i+1))
		if building == nil {
			t.Fatalf("expected construction resource slot %d to exist", i+1)
		}
		if building.Type != expectedType {
			t.Fatalf("expected construction resource slot %d type %s, got %s", i+1, expectedType, building.Type)
		}
	}
	if countConstructionResourceSlots(state.Buildings) != len(expectedTypes) {
		t.Fatalf("expected %d construction resource slots, got %+v", len(expectedTypes), state.Buildings)
	}
}
