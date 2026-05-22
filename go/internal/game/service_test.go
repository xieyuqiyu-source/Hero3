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
