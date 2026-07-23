// 本文件锁定正式将领配置中的战前真实伤亡与临时压制数值和适用范围。
package game

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// TestFormalPreBattleForceTraitConfigsMatchDesign 逐项核对五项战前扣兵与压制特性的正式配置。
func TestFormalPreBattleForceTraitConfigsMatchDesign(t *testing.T) {
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
		generalID    string
		traitID      string
		allowedSides []string
		params       map[string]float64
	}{
		{generalID: "simayi", traitID: "yibing_touxi", params: map[string]float64{"triggerChance": 0.35, "effectRate": 0.35}},
		{generalID: "guanyu", traitID: "shuiyan_qijun", params: map[string]float64{"triggerChance": 0.35, "effectRate": 0.35, "maxAffectedRate": 0.35}},
		{generalID: "zhangliao", traitID: "weizhen_zhenhe", allowedSides: []string{"attacker"}, params: map[string]float64{"triggerChance": 0.35, "effectRate": 0.25}},
		{generalID: "zhangfei", traitID: "zhenhe_quanjun", params: map[string]float64{"triggerChance": 0.5, "effectRate": 0.5, "maxAffectedRate": 0.5}},
		{generalID: "zhugeliang", traitID: "qimen_dunjia", params: map[string]float64{"effectRate": 0.25, "maxAffectedRate": 0.25}},
	}
	for _, tc := range tests {
		t.Run(tc.traitID, func(t *testing.T) {
			hero, ok := cfg.Heroes[tc.generalID]
			if !ok {
				t.Fatalf("formal general %s not found", tc.generalID)
			}
			trait := hero.SpecialTrait
			if !trait.Enabled || trait.TraitID != tc.traitID || trait.TraitType != "special" || trait.Scope != "enemy_army" {
				t.Fatalf("unexpected formal trait identity: %+v", trait)
			}
			if !reflect.DeepEqual(trait.AllowedSides, tc.allowedSides) || len(trait.AllowedScenes) != 0 || trait.RequiredOutcome != "" || trait.TargetUnitType != "" {
				t.Fatalf("unexpected battle constraints: %+v", trait)
			}
			if !reflect.DeepEqual(trait.Params, tc.params) {
				t.Fatalf("unexpected formal parameters: %+v", trait.Params)
			}
		})
	}
}
