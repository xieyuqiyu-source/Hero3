// 本文件锁定张辽双特性的旧配置迁移和 GM 参数默认值。
package game

import (
	"reflect"
	"testing"

	"hero3/internal/core/general"
)

// TestNormalizeZhangLiaoLegacyTraitParams 验证旧压制和必定加攻参数迁移为当前主动进攻概率规则。
func TestNormalizeZhangLiaoLegacyTraitParams(t *testing.T) {
	cfg := NormalizeGeneralsConfig(GeneralsConfig{Heroes: map[string]GeneralHeroConfig{
		"zhangliao": {
			SpecialTrait: GeneralTraitConfig{
				TraitID: "weizhen_zhenhe", Scope: "enemy_army",
				Params: map[string]float64{"triggerChance": 0.35, "effectRate": 0.2, "maxAffectedRate": 0.2, "maxAffectedCount": 10000},
			},
			BonusTrait: GeneralTraitConfig{
				TraitID: "weizhen_xiaoyao", Scope: "self_army", TargetUnitType: "cavalry", AllowedSides: []string{"attacker"},
				Params: map[string]float64{"attackBonusRate": 0.35},
			},
		},
	}})
	hero := cfg.Heroes["zhangliao"]
	if !reflect.DeepEqual(hero.SpecialTrait.Params, map[string]float64{"triggerChance": 0.35, "effectRate": 0.25}) || !reflect.DeepEqual(hero.SpecialTrait.AllowedSides, []string{"attacker"}) {
		t.Fatalf("expected legacy suppression migrated to attacker-only 35%%/25%% flee, got %+v", hero.SpecialTrait)
	}
	if !reflect.DeepEqual(hero.BonusTrait.Params, map[string]float64{"triggerChance": 0.6, "attackBonusRate": 0.35}) || !reflect.DeepEqual(hero.BonusTrait.AllowedSides, []string{"attacker"}) {
		t.Fatalf("expected legacy cavalry bonus migrated to attacker-only 60%%/+35%%, got %+v", hero.BonusTrait)
	}
}

// TestZhangLiaoTraitSchemasExposeGMProbabilityAndRates 验证 GM 能配置两项概率和对应效果比例。
func TestZhangLiaoTraitSchemasExposeGMProbabilityAndRates(t *testing.T) {
	tests := []struct {
		traitID       string
		chance        float64
		effectKey     string
		effectDefault float64
	}{
		{traitID: "weizhen_zhenhe", chance: 0.35, effectKey: "effectRate", effectDefault: 0.25},
		{traitID: "weizhen_xiaoyao", chance: 0.6, effectKey: "attackBonusRate", effectDefault: 0.35},
	}
	for _, tc := range tests {
		t.Run(tc.traitID, func(t *testing.T) {
			trait, ok := general.Get(tc.traitID)
			if !ok {
				t.Fatalf("trait %s not registered", tc.traitID)
			}
			fields := map[string]general.ParamField{}
			for _, field := range trait.ParamSchema() {
				fields[field.Key] = field
			}
			if fields["triggerChance"].Default != tc.chance || fields[tc.effectKey].Default != tc.effectDefault {
				t.Fatalf("unexpected GM defaults for %s: %+v", tc.traitID, fields)
			}
		})
	}
}
