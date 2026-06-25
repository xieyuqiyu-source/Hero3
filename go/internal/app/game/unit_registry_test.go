package game

import "testing"

// 本文件验证兵种配置注册表入口。

func TestUnitRegistryExposesUnitConfigs(t *testing.T) {
	setTestCombatUnitsConfig(t)

	unit, exists := GetUnitConfig("wei", "weiInfantry")
	if !exists {
		t.Fatal("expected weiInfantry config to exist")
	}
	if unit.Category != "infantry" {
		t.Fatalf("expected infantry category, got %q", unit.Category)
	}
	if !UnitRegistered("wei", "weiCavalry") {
		t.Fatal("expected weiCavalry to be registered")
	}
	if UnitRegistered("wei", "unknown_unit") {
		t.Fatal("expected unknown_unit to be unregistered")
	}
}

func TestListUnitConfigsReturnsCopy(t *testing.T) {
	setTestCombatUnitsConfig(t)

	units := GetUnitsConfig()
	delete(units["wei"], "weiInfantry")
	if !UnitRegistered("wei", "weiInfantry") {
		t.Fatal("expected mutating listed units not to change active registry")
	}

	units = GetUnitsConfig()
	unit := units["wei"]["weiInfantry"]
	unit.Stats["attack"] = 999
	units["wei"]["weiInfantry"] = unit

	current, exists := GetUnitConfig("wei", "weiInfantry")
	if !exists {
		t.Fatal("expected weiInfantry config to exist")
	}
	if current.Stats["attack"] == 999 {
		t.Fatal("expected mutating listed unit stats not to change active registry")
	}
}

func TestFindFactionUnitByNameSupportsAliases(t *testing.T) {
	original := GetUnitsConfig()
	defer func() {
		unitsMu.Lock()
		activeUnits = original
		unitsMu.Unlock()
	}()

	unitsMu.Lock()
	activeUnits = UnitsConfig{
		"wei": FactionUnits{
			"weiScholar": UnitConfig{Name: "士族", Stats: map[string]int{"upkeep": 1}},
		},
	}
	unitsMu.Unlock()

	unitID, _, exists := FindFactionUnitByName("wei", "土族")
	if !exists || unitID != "weiScholar" {
		t.Fatalf("expected alias 土族 to find weiScholar, got id=%q exists=%v", unitID, exists)
	}
}
