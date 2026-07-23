// 本文件锁定正式将领配置中的战后减损、复活和返还规则。
package game

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// TestFormalRecoveryTraitConfigsMatchDesign 逐项核对当前三项战后恢复特性的正式配置和适用条件。
func TestFormalRecoveryTraitConfigsMatchDesign(t *testing.T) {
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
		generalID       string
		traitID         string
		traitType       string
		scope           string
		allowedSides    []string
		requiredOutcome string
		params          map[string]float64
	}{
		{generalID: "guojia", traitID: "guicai_yice", traitType: "bonus", scope: "self_army", allowedSides: []string{"attacker", "defender", "reinforcement"}, params: map[string]float64{"effectRate": 0.22, "triggerChance": 1}},
		{generalID: "liubei", traitID: "renzhu_shouhu", traitType: "bonus", scope: "self_army", allowedSides: []string{"attacker", "defender", "reinforcement"}, params: map[string]float64{"effectRate": 0.35, "triggerChance": 0.6}},
		{generalID: "zhaoyun", traitID: "longdan_jiuyuan", traitType: "special", scope: "reinforcement_self", allowedSides: []string{"defender", "reinforcement"}, params: map[string]float64{"triggerChance": 0.35, "lossReductionRate": 0.2}},
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
			if !trait.Enabled || trait.TraitID != tc.traitID || trait.TraitType != tc.traitType || trait.Scope != tc.scope {
				t.Fatalf("unexpected formal trait identity: %+v", trait)
			}
			if !reflect.DeepEqual(trait.AllowedSides, tc.allowedSides) || len(trait.AllowedScenes) != 0 || trait.RequiredOutcome != tc.requiredOutcome || trait.TargetUnitType != "" {
				t.Fatalf("unexpected formal recovery constraints: %+v", trait)
			}
			if !reflect.DeepEqual(trait.Params, tc.params) {
				t.Fatalf("unexpected formal recovery parameters: %+v", trait.Params)
			}
		})
	}
}
