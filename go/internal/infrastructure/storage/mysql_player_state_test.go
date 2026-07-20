// 本文件验证 MySQL 玩家状态仓储的资源影子表同步辅助逻辑。
package storage

import (
	"encoding/json"
	"reflect"
	"testing"

	"hero3/internal/app/game"
)

func TestResourceTypesFromStateMergesItemsAndCapacity(t *testing.T) {
	resources := game.ResourceState{
		Items: map[string]int{
			"wood":  100,
			"food":  200,
			"":      1,
			"stone": 0,
		},
		Capacity: map[string]int{
			"wood": 1000,
			"iron": 500,
		},
	}

	got := resourceTypesFromState(resources)
	want := []string{"food", "iron", "stone", "wood"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected resource types %v, got %v", want, got)
	}
}

func TestCompactPlayerStateSnapshotDropsAuthoritativeAssets(t *testing.T) {
	state := game.GameState{
		Player: game.Player{ID: "player_1", Nickname: "主公", Faction: "wei"},
		Resources: game.ResourceState{
			Items:    map[string]int{"wood": 100},
			Capacity: map[string]int{"wood": 1000},
		},
		Inventory:            map[string]game.ItemStack{"item_1": {ItemID: "item_1", Amount: 2}},
		Buildings:            []game.Building{{ID: "wood_camp-1", Type: "wood_camp", Level: 3}},
		ResourceSlots:        []game.ResourceSlot{{ID: "wood_slot-1", ResourceType: "wood"}},
		Generals:             []game.General{{ID: "caocao", Level: 1}},
		GeneralAssignments:   []game.GeneralAssignment{{ID: "main", GeneralID: "caocao", Slot: "main"}},
		Army:                 []game.ArmyUnit{{UnitType: "weiInfantry", Amount: 10}},
		RecruitQueues:        []game.RecruitQueue{{ID: "queue_1", UnitType: "weiInfantry", Amount: 5}},
		Buffs:                []game.Buff{{ID: "buff_1", Key: "woodProductionBonus", Mode: "add", Value: 10}},
		NpcState:             &game.NpcState{Cities: []game.NpcCity{{ID: "npc_1", Name: "NPC 1"}}},
		ResourceSettledAt:    "2026-06-26T00:00:00Z",
		GeneralTraitProgress: map[string]float64{"caocao:weiwu_haoling:huWei": 0.5},
		CityGold:             12,
		LastExchangeAt:       "2026-06-26T01:00:00Z",
		ProductionBoost:      2,
		ProductionBoostEnd:   "2026-06-27T00:00:00Z",
		ServerTime:           "2026-06-26T00:00:00Z",
		DeleteRequestedAt:    "2026-07-02T03:11:44Z",
		DeleteScheduledAt:    "2026-07-02T04:11:44Z",
	}

	snapshotJSON, err := marshalPlayerStateSnapshot(state)
	if err != nil {
		t.Fatalf("marshal compact snapshot: %v", err)
	}

	var snapshot map[string]any
	if err := json.Unmarshal(snapshotJSON, &snapshot); err != nil {
		t.Fatalf("unmarshal compact snapshot: %v", err)
	}

	for _, key := range []string{"resources", "inventory", "buildings", "resourceSlots", "generals", "generalAssignments", "army", "recruitQueues", "buffs", "npcState", "serverTime", "cityGold", "lastExchangeAt"} {
		if _, exists := snapshot[key]; exists {
			t.Fatalf("expected %s to be omitted from compact state_json", key)
		}
	}
	if snapshot["resourceSettledAt"] != state.ResourceSettledAt {
		t.Fatalf("expected resourceSettledAt to be preserved")
	}
	progress, ok := snapshot["generalTraitProgress"].(map[string]any)
	if !ok || progress["caocao:weiwu_haoling:huWei"] != 0.5 {
		t.Fatalf("expected generalTraitProgress to be preserved, got %+v", snapshot["generalTraitProgress"])
	}
	if snapshot["deleteRequestedAt"] != state.DeleteRequestedAt {
		t.Fatalf("expected deleteRequestedAt to be preserved")
	}
	if snapshot["deleteScheduledAt"] != state.DeleteScheduledAt {
		t.Fatalf("expected deleteScheduledAt to be preserved")
	}
}

func TestResourceSnapshotChangedDetectsInPlaceMapMutation(t *testing.T) {
	resources := game.ResourceState{
		Items: map[string]int{
			"wood": 100,
		},
		Capacity: map[string]int{
			"wood": 1000,
		},
	}
	before := resourceSnapshotsFromStorageState(resources)

	resources.Items["wood"] = 120
	if !resourceSnapshotChanged(before, resources) {
		t.Fatal("expected in-place resource map mutation to be detected")
	}
}

func TestResourceSnapshotChangedIgnoresEquivalentResources(t *testing.T) {
	resources := game.ResourceState{
		Items: map[string]int{
			"wood": 100,
		},
		Capacity: map[string]int{
			"wood": 1000,
		},
	}
	before := resourceSnapshotsFromStorageState(resources)

	after := game.ResourceState{
		Items: map[string]int{
			"wood": 100,
		},
		Capacity: map[string]int{
			"wood": 1000,
		},
	}
	if resourceSnapshotChanged(before, after) {
		t.Fatal("expected equivalent resource state to be unchanged")
	}
}
