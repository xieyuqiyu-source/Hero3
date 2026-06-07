package game

import (
	"math"
	"testing"
	"time"
)

func TestBalanceConfigLoadsMilitaryBuildingModifiers(t *testing.T) {
	original := GetBalanceConfig()
	defer func() {
		if err := SetBalanceConfig(original); err != nil {
			t.Fatalf("restore balance config failed: %v", err)
		}
	}()

	if err := LoadBalanceConfig("../../config/balance.json"); err != nil {
		t.Fatalf("LoadBalanceConfig failed: %v", err)
	}

	cfg := GetBalanceConfig()
	infantryCamp, exists := cfg.Buildings["infantry_camp"]
	if !exists {
		t.Fatal("expected infantry_camp config to exist")
	}
	levelThree := infantryCamp.ModifiersByLevel[3]
	if len(levelThree) != 1 {
		t.Fatalf("expected 1 level 3 modifier, got %d", len(levelThree))
	}
	expectedLevelThree := recruitSpeedBonusForLevel(3, 60, 5, 20)
	if levelThree[0].Key != StatInfantryRecruitSpeedBonus || math.Abs(levelThree[0].Value-expectedLevelThree) > 1e-6 {
		t.Fatalf("unexpected infantry_camp level 3 modifier: %+v", levelThree[0])
	}
	levelTwenty := infantryCamp.ModifiersByLevel[20]
	if len(levelTwenty) != 1 {
		t.Fatalf("expected 1 level 20 modifier, got %d", len(levelTwenty))
	}
	if levelTwenty[0].Key != StatInfantryRecruitSpeedBonus || math.Abs(levelTwenty[0].Value-11) > 1e-6 {
		t.Fatalf("expected level 20 recruit speed bonus 11, got %+v", levelTwenty[0])
	}
}

func TestMilitaryBuildingsApplyCombatModifiers(t *testing.T) {
	originalUnits := GetUnitsConfig()
	defer func() {
		unitsMu.Lock()
		activeUnits = originalUnits
		unitsMu.Unlock()
	}()

	unitsMu.Lock()
	activeUnits = UnitsConfig{
		"wei": FactionUnits{
			"weiInfantry": UnitConfig{
				Category: "infantry",
				Stats: map[string]int{
					"attack":          100,
					"infantryDefense": 80,
					"cavalryDefense":  60,
					"carryCapacity":   10,
					"upkeep":          1,
				},
			},
		},
	}
	unitsMu.Unlock()

	state := GameState{
		Player: Player{ID: "p1", Faction: "wei"},
		Army:   []ArmyUnit{{UnitType: "weiInfantry", Amount: 10}},
		Buildings: []Building{
			{ID: "weapon_bureau-1", Type: "weapon_bureau", Level: 2},
			{ID: "armor_bureau-1", Type: "armor_bureau", Level: 4},
		},
	}

	units, err := validateAndConsumeArmy(&state, map[string]int{"weiInfantry": 10})
	if err != nil {
		t.Fatalf("validateAndConsumeArmy failed: %v", err)
	}
	if len(units) != 1 {
		t.Fatalf("expected 1 combat unit, got %d", len(units))
	}

	unit := units[0]
	if unit.Attack != 102 {
		t.Fatalf("expected infantry attack 102, got %d", unit.Attack)
	}
	if unit.InfantryDefense != 83 {
		t.Fatalf("expected infantry defense 83, got %d", unit.InfantryDefense)
	}
	if unit.CavalryDefense != 62 {
		t.Fatalf("expected cavalry defense 62, got %d", unit.CavalryDefense)
	}
}

func TestCampBuildingsApplyCategoryRecruitSpeed(t *testing.T) {
	sources := []ModifierSource{&BuildingBonusSource{
		Buildings: []Building{
			{ID: "infantry_camp-1", Type: "infantry_camp", Level: 20},
			{ID: "cavalry_camp-1", Type: "cavalry_camp", Level: 1},
		},
	}}
	now := time.Now()

	infantrySeconds := applyCategoryRecruitSpeedBonus(60, "infantry", now, sources)
	if infantrySeconds != 5 {
		t.Fatalf("expected infantry recruit seconds 5, got %d", infantrySeconds)
	}

	longInfantrySeconds := applyCategoryRecruitSpeedBonus(240, "infantry", now, sources)
	if longInfantrySeconds != 20 {
		t.Fatalf("expected long infantry recruit seconds 20, got %d", longInfantrySeconds)
	}

	cavalrySeconds := applyCategoryRecruitSpeedBonus(100, "cavalry", now, sources)
	if cavalrySeconds != 100 {
		t.Fatalf("expected cavalry recruit seconds 100, got %d", cavalrySeconds)
	}

	otherSeconds := applyCategoryRecruitSpeedBonus(100, "siege", now, sources)
	if otherSeconds != 100 {
		t.Fatalf("expected siege recruit seconds 100, got %d", otherSeconds)
	}
}

func TestSpeedBonusSupportsPercentModes(t *testing.T) {
	now := time.Now()
	sources := []ModifierSource{&StaticModifierSource{
		Name: "测试速度加成",
		Mods: []Modifier{
			{Key: StatRecruitSpeedBonus, Value: 0.5, Mode: "percentAdd"},
			{Key: StatRecruitSpeedBonus, Value: 1, Mode: "percentMultiply"},
		},
	}}

	seconds := applySpeedBonus(120, StatRecruitSpeedBonus, now, sources)
	if seconds != 40 {
		t.Fatalf("expected 120 seconds with 1.5x then 2x speed to become 40, got %d", seconds)
	}
}

func TestValidateBalanceRejectsInvalidModifierKey(t *testing.T) {
	config := GetBalanceConfig()
	building := config.Buildings["weapon_bureau"]
	building.ModifiersByLevel = map[int][]Modifier{
		1: {{Key: "atackBonus", Value: 0.1, Mode: "percentAdd"}},
	}
	config.Buildings["weapon_bureau"] = building

	if err := SetBalanceConfig(config); err == nil {
		t.Fatal("expected invalid modifier key to be rejected")
	}
}

func TestValidateBalanceRejectsInvalidModifierMode(t *testing.T) {
	config := GetBalanceConfig()
	building := config.Buildings["weapon_bureau"]
	building.ModifiersByLevel = map[int][]Modifier{
		1: {{Key: StatAttackBonus, Value: 0.1, Mode: "precentAdd"}},
	}
	config.Buildings["weapon_bureau"] = building

	if err := SetBalanceConfig(config); err == nil {
		t.Fatal("expected invalid modifier mode to be rejected")
	}
}

func TestBuildingBonusSourceReportsActiveModifiers(t *testing.T) {
	state := GameState{
		Buildings: []Building{
			{ID: "cavalry_camp-1", Type: "cavalry_camp", Level: 2},
		},
	}

	items := GetModifierBreakdown(&state, time.Now())
	if len(items) != 1 {
		t.Fatalf("expected 1 active building modifier, got %d", len(items))
	}
	for _, item := range items {
		if item.Source != "军事建筑" {
			t.Fatalf("expected source 军事建筑, got %q", item.Source)
		}
		expected := recruitSpeedBonusForLevel(2, 60, 5, 20)
		if math.Abs(item.Value-expected) > 1e-9 {
			t.Fatalf("expected value %.4f, got %.4f", expected, item.Value)
		}
	}
}
