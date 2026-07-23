// 本文件锁定正式将领配置中的战后追加伤害数值、目标兵种和适用条件。
package game

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// TestFormalAfterCombatDamageTraitConfigsMatchDesign 逐项核对六项战后追加伤害特性的正式配置。
func TestFormalAfterCombatDamageTraitConfigsMatchDesign(t *testing.T) {
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
		targetUnitType  string
		allowedSides    []string
		allowedScenes   []string
		requiredOutcome string
		params          map[string]float64
	}{
		{generalID: "machao", traitID: "xiliang_tuji", traitType: "special", targetUnitType: "cavalry", params: map[string]float64{"triggerChance": 0.35, "effectRate": 0.12}},
		{generalID: "sunce", traitID: "xiaobawang_zhuiji", traitType: "special", allowedScenes: []string{"plunder"}, requiredOutcome: "win", params: map[string]float64{"triggerChance": 0.35, "effectRate": 0.1}},
		{generalID: "zhouyu", traitID: "huogong", traitType: "special", allowedSides: []string{"attacker"}, params: map[string]float64{"effectRate": 0.25, "damagePercent": 0.25, "triggerChance": 1}},
		{generalID: "luxun", traitID: "huoshao_lianying", traitType: "special", targetUnitType: "infantry", params: map[string]float64{"triggerChance": 0.35, "effectRate": 1, "maxAffectedRate": 1}},
		{generalID: "luxun", traitID: "lianying_zengshang", traitType: "bonus", targetUnitType: "infantry", params: map[string]float64{"effectRate": 0.1}},
		{generalID: "huanggai", traitID: "kurou_fanji", traitType: "bonus", params: map[string]float64{"effectRate": 0.1}},
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
			if !trait.Enabled || trait.TraitID != tc.traitID || trait.TraitType != tc.traitType || trait.Scope != "enemy_army" {
				t.Fatalf("unexpected formal trait identity: %+v", trait)
			}
			if trait.TargetUnitType != tc.targetUnitType || !reflect.DeepEqual(trait.AllowedSides, tc.allowedSides) || !reflect.DeepEqual(trait.AllowedScenes, tc.allowedScenes) || trait.RequiredOutcome != tc.requiredOutcome {
				t.Fatalf("unexpected target or battle constraints: %+v", trait)
			}
			if !reflect.DeepEqual(trait.Params, tc.params) {
				t.Fatalf("unexpected formal parameters: %+v", trait.Params)
			}
		})
	}
}
