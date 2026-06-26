// 本文件验证 MySQL 玩家状态仓储的资源影子表同步辅助逻辑。
package storage

import (
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
