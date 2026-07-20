// 本文件验证兵力结构转换的阵营归属与稳定排序。
package game

import (
	"reflect"
	"testing"
	"time"
)

// TestSplitCapturedUnitsByOwnerFaction 验证俘虏兵按阵营进入军队或驻防。
func TestSplitCapturedUnitsByOwnerFaction(t *testing.T) {
	previousUnits := GetFactionUnits("test_owner")
	if err := SaveFactionUnits("", "test_owner", FactionUnits{"ownerUnit": UnitConfig{Name: "本阵营兵"}}); err != nil {
		t.Fatalf("SaveFactionUnits failed: %v", err)
	}
	t.Cleanup(func() {
		_ = SaveFactionUnits("", "test_owner", previousUnits)
	})

	toArmy, toGarrison := splitCapturedUnitsByOwnerFaction("test_owner", map[string]int{
		"ownerUnit":   10,
		"foreignUnit": 25,
		"":            9,
		"zeroUnit":    0,
	})

	if toArmy["ownerUnit"] != 10 {
		t.Fatalf("expected own faction unit to enter army, got %+v", toArmy)
	}
	if _, ok := toArmy["foreignUnit"]; ok {
		t.Fatalf("cross faction unit should not enter army, got %+v", toArmy)
	}
	if toGarrison["foreignUnit"] != 25 {
		t.Fatalf("expected cross faction unit to enter garrison, got %+v", toGarrison)
	}
	if _, ok := toGarrison["ownerUnit"]; ok {
		t.Fatalf("own faction unit should not enter garrison, got %+v", toGarrison)
	}
}

// TestArmyMapToSliceUsesStableUnitOrder 验证兵力 map 每次都按兵种 ID 转为稳定切片，避免战损余数随遍历顺序漂移。
func TestArmyMapToSliceUsesStableUnitOrder(t *testing.T) {
	want := []ArmyUnit{{UnitType: "aInfantry", Amount: 10}, {UnitType: "mSpecial", Amount: 30}, {UnitType: "zCavalry", Amount: 20}}
	for run := 0; run < 100; run++ {
		got := armyMapToSlice(map[string]int{"zCavalry": 20, "ignored": 0, "mSpecial": 30, "aInfantry": 10})
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("expected stable unit order %+v, got %+v on run %d", want, got, run)
		}
	}
}

// TestBuildSimulatedCombatUnitsUsesStableUnitOrder 验证真实战斗输入不受请求兵力 map 的随机遍历顺序影响。
func TestBuildSimulatedCombatUnitsUsesStableUnitOrder(t *testing.T) {
	previousUnits := GetFactionUnits("stable_order")
	if err := SaveFactionUnits("", "stable_order", FactionUnits{
		"zCavalry":  {Name: "测试骑兵", Category: "cavalry", Stats: map[string]int{"attack": 10, "infantryDefense": 10, "cavalryDefense": 10, "upkeep": 1}},
		"aInfantry": {Name: "测试步兵", Category: "infantry", Stats: map[string]int{"attack": 10, "infantryDefense": 10, "cavalryDefense": 10, "upkeep": 1}},
	}); err != nil {
		t.Fatalf("SaveFactionUnits failed: %v", err)
	}
	t.Cleanup(func() {
		_ = SaveFactionUnits("", "stable_order", previousUnits)
	})

	for run := 0; run < 100; run++ {
		units, err := buildSimulatedCombatUnits("stable_order", map[string]int{"zCavalry": 50, "aInfantry": 50}, time.Now())
		if err != nil || len(units) != 2 || units[0].ID != "aInfantry" || units[1].ID != "zCavalry" {
			t.Fatalf("expected stable combat unit order on run %d, units=%+v err=%v", run, units, err)
		}
	}
}
