package game

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestSettleResourcesAddsProducedResources(t *testing.T) {
	settledAt := time.Date(2026, 5, 22, 10, 0, 0, 0, time.UTC)
	state := newPlayerState("player_test", "主公", "wei", settledAt)
	state.Resources = ResourceState{
		Items: map[string]int{
			"wood":  100,
			"stone": 100,
			"iron":  100,
			"food":  100,
		},
		Capacity: map[string]int{
			"wood":  7500,
			"stone": 7500,
			"iron":  7500,
			"food":  7500,
		},
	}
	state.ResourceSettledAt = settledAt.Format(time.RFC3339)

	next, changed := settleResources(state, settledAt.Add(time.Hour))
	if !changed {
		t.Fatal("expected resource settlement to change state")
	}

	if next.Resources.Items["wood"] <= state.Resources.Items["wood"] {
		t.Fatalf("expected wood to grow, got %d", next.Resources.Items["wood"])
	}
	if next.Resources.Items["food"] <= state.Resources.Items["food"] {
		t.Fatalf("expected food to grow, got %d", next.Resources.Items["food"])
	}
	if next.ResourceSettledAt != settledAt.Add(time.Hour).Format(time.RFC3339) {
		t.Fatalf("expected settlement timestamp to advance, got %s", next.ResourceSettledAt)
	}
}

func TestSettleResourcesCapsAtCapacity(t *testing.T) {
	settledAt := time.Date(2026, 5, 22, 10, 0, 0, 0, time.UTC)
	state := newPlayerState("player_test", "主公", "wei", settledAt)
	state.Resources = ResourceState{
		Items: map[string]int{
			"wood":  7499,
			"stone": 7499,
			"iron":  7499,
			"food":  7499,
		},
		Capacity: map[string]int{
			"wood":  7500,
			"stone": 7500,
			"iron":  7500,
			"food":  7500,
		},
	}
	state.ResourceSettledAt = settledAt.Format(time.RFC3339)

	next, _ := settleResources(state, settledAt.Add(24*time.Hour))
	if next.Resources.Items["wood"] != next.Resources.Capacity["wood"] {
		t.Fatalf("expected wood to cap at %d, got %d", next.Resources.Capacity["wood"], next.Resources.Items["wood"])
	}
	if next.Resources.Items["food"] != next.Resources.Capacity["food"] {
		t.Fatalf("expected food to cap at %d, got %d", next.Resources.Capacity["food"], next.Resources.Items["food"])
	}
}

func TestSettleResourcesAdvancesTimestampWhenCapacityIsFull(t *testing.T) {
	settledAt := time.Date(2026, 5, 22, 10, 0, 0, 0, time.UTC)
	state := newPlayerState("player_test", "主公", "wei", settledAt)
	state.Resources = ResourceState{
		Items: map[string]int{
			"wood":  7500,
			"stone": 7500,
			"iron":  7500,
			"food":  7500,
		},
		Capacity: map[string]int{
			"wood":  7500,
			"stone": 7500,
			"iron":  7500,
			"food":  7500,
		},
	}
	state.ResourceSettledAt = settledAt.Format(time.RFC3339)

	next, changed := settleResources(state, settledAt.Add(time.Hour))
	if !changed {
		t.Fatal("expected full-capacity settlement to advance timestamp")
	}
	if next.ResourceSettledAt != settledAt.Add(time.Hour).Format(time.RFC3339) {
		t.Fatalf("expected settlement timestamp to advance, got %s", next.ResourceSettledAt)
	}
}

func TestResourceStateUnmarshalMigratesLegacyShape(t *testing.T) {
	var resources ResourceState
	err := json.Unmarshal([]byte(`{"wood":1200,"stone":900,"iron":600,"food":1500,"capacity":5000}`), &resources)
	if err != nil {
		t.Fatalf("unexpected unmarshal error: %v", err)
	}

	if resources.Items["wood"] != 1200 {
		t.Fatalf("expected legacy wood to migrate, got %d", resources.Items["wood"])
	}
	if resources.Capacity["iron"] != 5000 {
		t.Fatalf("expected legacy capacity to apply per resource, got %d", resources.Capacity["iron"])
	}
}

func TestResourceProductionUnmarshalMigratesLegacyShape(t *testing.T) {
	var production ResourceProduction
	err := json.Unmarshal([]byte(`{"woodPerHour":84,"stonePerHour":62,"ironPerHour":48,"foodPerHour":100}`), &production)
	if err != nil {
		t.Fatalf("unexpected unmarshal error: %v", err)
	}

	if production["wood"] != 84 {
		t.Fatalf("expected legacy wood production to migrate, got %d", production["wood"])
	}
	if production["food"] != 100 {
		t.Fatalf("expected legacy food production to migrate, got %d", production["food"])
	}
}

func TestCalculateResourceProductionUsesBalanceConfig(t *testing.T) {
	production := calculateResourceProduction([]Building{
		{Type: "wood_camp", Level: 3},
		{Type: "stone_quarry", Level: 2},
		{Type: "iron_mine", Level: 2},
		{Type: "farm", Level: 3},
	})

	if production["wood"] != 30 {
		t.Fatalf("expected wood production from config to be 30, got %d", production["wood"])
	}
	if production["food"] != 30 {
		t.Fatalf("expected food production from config to be 30, got %d", production["food"])
	}
}

func TestCalculateResourceCapacityUsesWarehouseConfig(t *testing.T) {
	capacity := calculateResourceCapacity([]Building{{Type: "warehouse", Level: 3}})

	if capacity["wood"] != 9200 {
		t.Fatalf("expected level 3 warehouse capacity to be 9200, got %d", capacity["wood"])
	}
	if capacity["food"] != 9200 {
		t.Fatalf("expected level 3 warehouse food capacity to be 9200, got %d", capacity["food"])
	}
}

func TestServiceUpdateBalancePersistsConfig(t *testing.T) {
	original := GetBalanceConfig()
	t.Cleanup(func() {
		if err := SetBalanceConfig(original); err != nil {
			t.Fatalf("restore balance config: %v", err)
		}
	})

	service := NewService()
	path := filepath.Join(t.TempDir(), "balance.json")
	if err := service.SetBalancePath(path); err != nil {
		t.Fatalf("set balance path: %v", err)
	}

	next := service.GetBalance()
	next.BaseProduction["wood"] = 99
	if err := service.UpdateBalance(next); err != nil {
		t.Fatalf("update balance: %v", err)
	}

	var saved BalanceConfig
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read balance file: %v", err)
	}
	if err := json.Unmarshal(content, &saved); err != nil {
		t.Fatalf("unmarshal balance: %v", err)
	}
	if saved.BaseProduction["wood"] != 99 {
		t.Fatalf("expected updated wood base production, got %d", saved.BaseProduction["wood"])
	}
}

func TestSettleResourcesTimeSlicingOnUpgradeCompletion(t *testing.T) {
	// 场景：10:00 结算，wood_camp-1 Lv.1 正在升级（10:01 完成），12:00 上线
	// 期望：wood = base + Lv.1产量×1分钟 + Lv.2产量×119分钟
	// 而不是全段按 Lv.2 产量算 120 分钟

	settledAt := time.Date(2026, 5, 22, 10, 0, 0, 0, time.UTC)
	upgradeEndsAt := settledAt.Add(1 * time.Minute).Format(time.RFC3339) // 10:01 完成
	onlineAt := settledAt.Add(2 * time.Hour)                             // 12:00 上线

	state := GameState{
		Player: Player{ID: "test", Nickname: "测试", Faction: "wei"},
		Resources: ResourceState{
			Items: map[string]int{
				"wood":  0,
				"stone": 0,
				"iron":  0,
				"food":  0,
			},
			Capacity: map[string]int{
				"wood":  999999,
				"stone": 999999,
				"iron":  999999,
				"food":  999999,
			},
		},
		Buildings: []Building{
			{ID: "wood_camp-1", Type: "wood_camp", Level: 1, UpgradeEndsAt: &upgradeEndsAt},
		},
		ResourceProduction: ResourceProduction{"wood": 10}, // 旧产量（Lv.1）
		ResourceSettledAt:  settledAt.Format(time.RFC3339),
	}

	result, changed := settleResources(state, onlineAt)
	if !changed {
		t.Fatal("expected state to change")
	}

	// 升级应该完成
	if result.Buildings[0].Level != 2 {
		t.Fatalf("expected building to be Lv.2, got Lv.%d", result.Buildings[0].Level)
	}
	if result.Buildings[0].UpgradeEndsAt != nil {
		t.Fatal("expected upgradeEndsAt to be cleared")
	}

	// 计算期望产量：
	// Lv.1 产量 = productionByLevel[1] = 10（wood_camp Lv.1）
	// Lv.2 产量 = productionByLevel[2] = 18（wood_camp Lv.2）
	// 10:00-10:01 (60秒): wood += 10 * 60 / 3600 = 0（不足1单位）
	// 10:01-12:00 (7140秒): wood += 18 * 7140 / 3600 = 35
	// 如果错误地全段按 Lv.2 算：18 * 7200 / 3600 = 36
	lv1Production := getProductionAtLevel("wood_camp", 1)
	lv2Production := getProductionAtLevel("wood_camp", 2)

	slice1Seconds := 60.0   // 10:00 - 10:01
	slice2Seconds := 7140.0 // 10:01 - 12:00

	expectedWood := int(float64(lv1Production)*slice1Seconds/3600) + int(float64(lv2Production)*slice2Seconds/3600)

	// 错误计算（全段按新产量）
	wrongWood := int(float64(lv2Production) * 7200.0 / 3600)

	if expectedWood == wrongWood {
		t.Fatalf("test setup error: expected and wrong values should differ (expected=%d, wrong=%d)", expectedWood, wrongWood)
	}

	if result.Resources.Items["wood"] != expectedWood {
		t.Fatalf("time slicing error: expected wood=%d, got %d (wrong full-period calc would give %d)",
			expectedWood, result.Resources.Items["wood"], wrongWood)
	}
}

func TestSettleResourcesMultipleUpgradesTimeSlicing(t *testing.T) {
	// 场景：两个建筑在离线期间先后完成升级
	// wood_camp-1: 10:01 完成, wood_camp-2: 10:30 完成
	// 验证三段切片都正确

	settledAt := time.Date(2026, 5, 22, 10, 0, 0, 0, time.UTC)
	upgrade1EndsAt := settledAt.Add(1 * time.Minute).Format(time.RFC3339)
	upgrade2EndsAt := settledAt.Add(30 * time.Minute).Format(time.RFC3339)
	onlineAt := settledAt.Add(1 * time.Hour)

	state := GameState{
		Player: Player{ID: "test", Nickname: "测试", Faction: "wei"},
		Resources: ResourceState{
			Items: map[string]int{
				"wood":  0,
				"stone": 0,
				"iron":  0,
				"food":  0,
			},
			Capacity: map[string]int{
				"wood":  999999,
				"stone": 999999,
				"iron":  999999,
				"food":  999999,
			},
		},
		Buildings: []Building{
			{ID: "wood_camp-1", Type: "wood_camp", Level: 1, UpgradeEndsAt: &upgrade1EndsAt},
			{ID: "wood_camp-2", Type: "wood_camp", Level: 1, UpgradeEndsAt: &upgrade2EndsAt},
		},
		ResourceProduction: ResourceProduction{"wood": 20}, // 2 × Lv.1
		ResourceSettledAt:  settledAt.Format(time.RFC3339),
	}

	result, changed := settleResources(state, onlineAt)
	if !changed {
		t.Fatal("expected state to change")
	}

	// 两个建筑都应该升到 Lv.2
	if result.Buildings[0].Level != 2 || result.Buildings[1].Level != 2 {
		t.Fatalf("expected both buildings at Lv.2, got Lv.%d and Lv.%d",
			result.Buildings[0].Level, result.Buildings[1].Level)
	}

	// 三段切片：
	// 10:00-10:01 (60s): 2×Lv.1 产量
	// 10:01-10:30 (1740s): 1×Lv.2 + 1×Lv.1 产量
	// 10:30-11:00 (1800s): 2×Lv.2 产量
	lv1 := getProductionAtLevel("wood_camp", 1)
	lv2 := getProductionAtLevel("wood_camp", 2)

	slice1 := int(float64(2*lv1) * 60.0 / 3600)
	slice2 := int(float64(lv2+lv1) * 1740.0 / 3600)
	slice3 := int(float64(2*lv2) * 1800.0 / 3600)
	expectedWood := slice1 + slice2 + slice3

	if result.Resources.Items["wood"] != expectedWood {
		t.Fatalf("multi-upgrade time slicing error: expected wood=%d, got %d (slices: %d+%d+%d)",
			expectedWood, result.Resources.Items["wood"], slice1, slice2, slice3)
	}
}

func getProductionAtLevel(buildingType string, level int) int {
	config, exists := getBuildingConfig(buildingType)
	if !exists || len(config.ProductionByLevel) == 0 {
		return 0
	}
	if level < 0 {
		return 0
	}
	if level >= len(config.ProductionByLevel) {
		return config.ProductionByLevel[len(config.ProductionByLevel)-1]
	}
	return config.ProductionByLevel[level]
}
