package game

import "testing"

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
	if len(state.Buildings) != 4 {
		t.Fatalf("expected 4 core buildings, got %d", len(state.Buildings))
	}
	if ensureCoreBuildings(&state) {
		t.Fatal("expected second ensure not to append duplicate buildings")
	}
}
