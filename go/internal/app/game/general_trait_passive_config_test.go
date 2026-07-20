// 本文件锁定美人计及留城、内政、被动属性特性的正式配置与非战斗边界。
package game

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// TestFormalPassiveAndCityTraitConfigsMatchDesign 逐项核对六项独立或非战斗特性的正式配置。
func TestFormalPassiveAndCityTraitConfigsMatchDesign(t *testing.T) {
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
		generalID      string
		traitID        string
		traitType      string
		scope          string
		targetUnitType string
		allowedSides   []string
		params         map[string]float64
	}{
		{generalID: "caocao", traitID: "weiwu_haoling", traitType: "special", scope: "self_city", targetUnitType: "huWei", params: map[string]float64{"guardPerMinute": 500, "maxGuardPerSettle": 3000}},
		{generalID: "zhenmi", traitID: "meiren", traitType: "special", scope: "self_army", allowedSides: []string{"attacker"}, params: map[string]float64{"captureRate": 0.2, "captureMax": 10000, "triggerChance": 1}},
		{generalID: "guojia", traitID: "shengui_zhicai", traitType: "special", scope: "self_city", params: map[string]float64{"resourceCostReduction": 0.5}},
		{generalID: "xunyu", traitID: "wangzuo_zhicai", traitType: "special", scope: "self_city", params: map[string]float64{"resourceCostReduction": 0.05}},
		{generalID: "xunyu", traitID: "neizheng_jingying", traitType: "bonus", scope: "self_city", params: map[string]float64{"productionBonusRate": 0.05}},
		{generalID: "machao", traitID: "tianshen_xiafan", traitType: "bonus", scope: "self_army", params: map[string]float64{"forceBonus": 20}},
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
			if trait.TargetUnitType != tc.targetUnitType || !reflect.DeepEqual(trait.AllowedSides, tc.allowedSides) || len(trait.AllowedScenes) != 0 || trait.RequiredOutcome != "" {
				t.Fatalf("unexpected target or battle constraints: %+v", trait)
			}
			if !reflect.DeepEqual(trait.Params, tc.params) {
				t.Fatalf("unexpected formal parameters: %+v", trait.Params)
			}
		})
	}
}

// TestNormalizeWeiwuHaolingLegacyLimitParam 验证旧的每日上限参数会迁移为真实的单次结算上限。
func TestNormalizeWeiwuHaolingLegacyLimitParam(t *testing.T) {
	cfg := NormalizeGeneralsConfig(GeneralsConfig{Heroes: map[string]GeneralHeroConfig{
		"caocao": {
			SpecialTrait: GeneralTraitConfig{
				TraitID: "weiwu_haoling",
				Params:  map[string]float64{"guardPerMinute": 500, "maxGuardPerDay": 3000},
			},
		},
	}})
	params := cfg.Heroes["caocao"].SpecialTrait.Params
	if params["maxGuardPerSettle"] != 3000 {
		t.Fatalf("expected legacy limit to migrate to maxGuardPerSettle, got %+v", params)
	}
	if _, exists := params["maxGuardPerDay"]; exists {
		t.Fatalf("expected legacy maxGuardPerDay removed after normalization, got %+v", params)
	}
}
