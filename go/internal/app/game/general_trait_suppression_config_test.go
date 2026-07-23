// 本文件锁定正式将领配置中的全体封禁与后续单项压制参数和作用范围。
package game

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// TestFormalTraitSuppressionConfigsMatchDesign 逐项核对卧龙奇谋与苦肉计的正式配置。
func TestFormalTraitSuppressionConfigsMatchDesign(t *testing.T) {
	path := filepath.Join("..", "..", "..", "config", "generals.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read formal generals config failed: %v", err)
	}
	var cfg GeneralsConfig
	if err := json.Unmarshal(raw, &cfg); err != nil {
		t.Fatalf("decode formal generals config failed: %v", err)
	}
	tests := []struct {
		generalID string
		traitID   string
		traitType string
		params    map[string]float64
		sides     []string
	}{
		{generalID: "zhugeliang", traitID: "wolong_mouzhi", traitType: "bonus", params: map[string]float64{"triggerChance": 0.6}, sides: []string{"attacker", "defender", "reinforcement"}},
		{generalID: "huanggai", traitID: "kurouji", traitType: "special", params: map[string]float64{"triggerChance": 0.35, "disableTraitCount": 1}},
	}
	for _, tc := range tests {
		t.Run(tc.traitID, func(t *testing.T) {
			hero, ok := cfg.Heroes[tc.generalID]
			if !ok {
				t.Fatalf("formal general %s not found", tc.generalID)
			}
			trait := hero.BonusTrait
			if tc.traitType == "special" {
				trait = hero.SpecialTrait
			}
			if !trait.Enabled || trait.TraitID != tc.traitID || trait.TraitType != tc.traitType || trait.Scope != "enemy_traits" {
				t.Fatalf("unexpected formal trait identity: %+v", trait)
			}
			if !reflect.DeepEqual(trait.AllowedSides, tc.sides) || len(trait.AllowedScenes) != 0 || trait.RequiredOutcome != "" || trait.TargetUnitType != "" {
				t.Fatalf("unexpected battle constraints: %+v", trait)
			}
			if !reflect.DeepEqual(trait.Params, tc.params) {
				t.Fatalf("unexpected formal parameters: %+v", trait.Params)
			}
		})
	}
}
