package game

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

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

func TestLoadUnitsConfigRequiresSpeedForMarchUnits(t *testing.T) {
	dir := t.TempDir()
	content := `{
  "weiInfantry": {
    "name": "魏步兵",
    "category": "infantry",
    "stats": {
      "attack": 10,
      "infantryDefense": 10,
      "cavalryDefense": 8,
      "upkeep": 1
    }
  }
}`
	if err := os.WriteFile(filepath.Join(dir, "wei.json"), []byte(content), 0o644); err != nil {
		t.Fatalf("write units config: %v", err)
	}
	err := LoadUnitsConfig(dir)
	if err == nil || !strings.Contains(err.Error(), "wei/weiInfantry") || !strings.Contains(err.Error(), "stats.speed") {
		t.Fatalf("expected missing speed validation error, got %v", err)
	}
}

func TestValidateUnitsConfigAllowsTransportWithoutSpeed(t *testing.T) {
	config := UnitsConfig{
		"wei": FactionUnits{
			"weiCart": UnitConfig{
				Name:  "辎重车",
				Role:  "transport",
				Stats: map[string]int{"carryCapacity": 1000, "upkeep": 0},
			},
		},
	}
	if err := ValidateUnitsConfig(config); err != nil {
		t.Fatalf("expected transport without speed to pass validation, got %v", err)
	}
}
