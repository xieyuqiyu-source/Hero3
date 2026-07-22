// 本文件锁定美人心计及留城、内政、被动属性特性的正式配置与非战斗边界。
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
		{generalID: "caocao", traitID: "weiwu_haoling", traitType: "special", scope: "self_city", targetUnitType: "huWei", params: map[string]float64{"guardPerMinute": 300}},
		{generalID: "zhenmi", traitID: "meiren", traitType: "special", scope: "self_army", allowedSides: []string{"attacker"}, params: map[string]float64{"attackBonusRate": 0.25, "triggerChance": 0.5}},
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

// TestNormalizeMeirenLegacyCaptureParams 验证旧俘虏参数会迁移为当前攻击加成，不再保留任何俘虏能力。
func TestNormalizeMeirenLegacyCaptureParams(t *testing.T) {
	cfg := NormalizeGeneralsConfig(GeneralsConfig{Heroes: map[string]GeneralHeroConfig{
		"zhenmi": {
			SpecialTrait: GeneralTraitConfig{
				TraitID: "meiren",
				Params:  map[string]float64{"captureRate": 0.2, "captureMax": 10000, "triggerChance": 1},
			},
		},
	}})
	params := cfg.Heroes["zhenmi"].SpecialTrait.Params
	if !reflect.DeepEqual(params, map[string]float64{"attackBonusRate": 0.25, "triggerChance": 0.5}) {
		t.Fatalf("expected legacy capture parameters migrated to current attack trait, got %+v", params)
	}
}

// TestNormalizeWeiwuHaolingLegacyLimitParam 验证历史上限参数会被清理，避免旧配置继续限制产兵。
func TestNormalizeWeiwuHaolingLegacyLimitParam(t *testing.T) {
	cfg := NormalizeGeneralsConfig(GeneralsConfig{Heroes: map[string]GeneralHeroConfig{
		"caocao": {
			SpecialTrait: GeneralTraitConfig{
				TraitID: "weiwu_haoling",
				Params:  map[string]float64{"guardPerMinute": 300, "maxGuardPerDay": 3000, "maxGuardPerSettle": 6000},
			},
		},
	}})
	params := cfg.Heroes["caocao"].SpecialTrait.Params
	if params["guardPerMinute"] != 300 {
		t.Fatalf("expected production rate preserved, got %+v", params)
	}
	if _, exists := params["maxGuardPerDay"]; exists {
		t.Fatalf("expected legacy maxGuardPerDay removed after normalization, got %+v", params)
	}
	if _, exists := params["maxGuardPerSettle"]; exists {
		t.Fatalf("expected legacy maxGuardPerSettle removed after normalization, got %+v", params)
	}
}

// TestNormalizeSimaYiLegacyTraitParams 验证司马懿旧版减攻与重复上限参数会迁移为当前规则。
func TestNormalizeSimaYiLegacyTraitParams(t *testing.T) {
	cfg := NormalizeGeneralsConfig(GeneralsConfig{Heroes: map[string]GeneralHeroConfig{
		"simayi": {
			SpecialTrait: GeneralTraitConfig{
				TraitID: "yibing_touxi", Scope: "enemy_army", AllowedSides: []string{"attacker"},
				Params: map[string]float64{"effectRate": 0.35, "maxAffectedRate": 0.35, "triggerChance": 0.6},
			},
			BonusTrait: GeneralTraitConfig{
				TraitID: "mouding_houfa", Scope: "enemy_army", AllowedSides: []string{"defender"},
				Params: map[string]float64{"effectRate": 0.1, "attackReductionRate": 0.1},
			},
		},
	}})
	simayi := cfg.Heroes["simayi"]
	if !reflect.DeepEqual(simayi.SpecialTrait.Params, map[string]float64{"effectRate": 0.35, "triggerChance": 0.6}) || simayi.SpecialTrait.Scope != "enemy_army" || len(simayi.SpecialTrait.AllowedSides) != 0 {
		t.Fatalf("expected Yibing to keep GM chance/rate and remove legacy cap or side restriction, got %+v", simayi.SpecialTrait)
	}
	if !reflect.DeepEqual(simayi.BonusTrait.Params, map[string]float64{"defenseBonusRate": 0.35, "triggerChance": 0.35}) || simayi.BonusTrait.Scope != "self_army" || !reflect.DeepEqual(simayi.BonusTrait.AllowedSides, []string{"defender", "reinforcement"}) {
		t.Fatalf("expected Mouding legacy attack reduction migrated to current probabilistic defense bonus, got %+v", simayi.BonusTrait)
	}
}
