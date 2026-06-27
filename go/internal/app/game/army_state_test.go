package game

import "testing"

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
