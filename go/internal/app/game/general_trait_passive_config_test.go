// 本文件锁定美人心计及留城、内政、被动属性特性的正式配置与非战斗边界。
package game

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"hero3/internal/core/general"
)

// TestFormalPassiveAndCityTraitConfigsMatchDesign 逐项核对独立或非战斗特性的正式配置。
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
		{generalID: "guojia", traitID: "shengui_zhicai", traitType: "special", scope: "self_army", params: map[string]float64{"politicsBonus": 10, "intelligenceBonus": 10}},
		{generalID: "xunyu", traitID: "wangzuo_zhicai", traitType: "special", scope: "self_city", params: map[string]float64{"resourceCostReduction": 0.05}},
		{generalID: "xunyu", traitID: "neizheng_jingying", traitType: "bonus", scope: "self_city", params: map[string]float64{"productionBonusRate": 0.05}},
		{generalID: "liubei", traitID: "rende", traitType: "special", scope: "self_army", params: map[string]float64{"politicsBonus": 10, "commandBonus": 12}},
		{generalID: "machao", traitID: "tianshen_xiafan", traitType: "bonus", scope: "self_army", params: map[string]float64{"forceBonus": 20}},
		{generalID: "xiahouyuan", traitID: "jixing_benxi", traitType: "special", scope: "self_army", targetUnitType: "qiQiYing", params: map[string]float64{"unitAttackFlat": 18, "unitSpeedFlat": 5}},
		{generalID: "xuchu", traitID: "huhu_shengwei", traitType: "bonus", scope: "self_army", targetUnitType: "huBaoQi", params: map[string]float64{"unitAttackFlat": 12, "unitSpeedFlat": 5}},
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

// TestNormalizeXuChuLegacyTraitParams 验证许褚旧双破防配置迁移为当前概率破防与虎豹骑永久被动。
func TestNormalizeXuChuLegacyTraitParams(t *testing.T) {
	cfg := NormalizeGeneralsConfig(GeneralsConfig{Heroes: map[string]GeneralHeroConfig{
		"xuchu": {
			SpecialTrait: GeneralTraitConfig{
				TraitID: "huchi_chongzhen", Scope: "enemy_army", AllowedSides: []string{"attacker"},
				Params: map[string]float64{"triggerChance": 0.35, "enemyDefenseReductionRate": 0.2},
			},
			BonusTrait: GeneralTraitConfig{
				TraitID: "pojun_pofang", Scope: "enemy_army", AllowedSides: []string{"attacker"},
				Params: map[string]float64{"enemyDefenseReductionRate": 0.35},
			},
		},
	}})
	hero := cfg.Heroes["xuchu"]
	if !reflect.DeepEqual(hero.SpecialTrait.Params, map[string]float64{"triggerChance": 0.5, "enemyDefenseReductionRate": 0.3}) || hero.SpecialTrait.Scope != "enemy_army" || !reflect.DeepEqual(hero.SpecialTrait.AllowedSides, []string{"attacker"}) {
		t.Fatalf("expected legacy Huchi migrated to attacker-only 50%%/-30%%, got %+v", hero.SpecialTrait)
	}
	if hero.BonusTrait.TraitID != "huhu_shengwei" || hero.BonusTrait.TraitType != general.TraitTypeBonus || hero.BonusTrait.Scope != "self_army" || hero.BonusTrait.TargetUnitType != "huBaoQi" || len(hero.BonusTrait.AllowedSides) != 0 || !reflect.DeepEqual(hero.BonusTrait.Params, map[string]float64{"unitAttackFlat": 12, "unitSpeedFlat": 5}) {
		t.Fatalf("expected legacy Pojun migrated to Huhu passive +12/+5, got %+v", hero.BonusTrait)
	}
}

// TestNormalizeXuChuCurrentGMParamsPreservesExplicitValues 验证现行 GM 自定义概率、比例和固定值不会被迁移覆盖。
func TestNormalizeXuChuCurrentGMParamsPreservesExplicitValues(t *testing.T) {
	cfg := NormalizeGeneralsConfig(GeneralsConfig{Heroes: map[string]GeneralHeroConfig{
		"xuchu": {
			SpecialTrait: GeneralTraitConfig{TraitID: "huchi_chongzhen", Params: map[string]float64{"triggerChance": 0.8, "enemyDefenseReductionRate": 0.4}},
			BonusTrait:   GeneralTraitConfig{TraitID: "huhu_shengwei", Params: map[string]float64{"unitAttackFlat": 20, "unitSpeedFlat": 7, "triggerChance": 1, "enemyDefenseReductionRate": 0.35}},
		},
	}})
	hero := cfg.Heroes["xuchu"]
	if !reflect.DeepEqual(hero.SpecialTrait.Params, map[string]float64{"triggerChance": 0.8, "enemyDefenseReductionRate": 0.4}) {
		t.Fatalf("expected current Huchi GM values preserved, got %+v", hero.SpecialTrait)
	}
	if !reflect.DeepEqual(hero.BonusTrait.Params, map[string]float64{"unitAttackFlat": 20, "unitSpeedFlat": 7}) || hero.BonusTrait.TargetUnitType != "huBaoQi" {
		t.Fatalf("expected current Huhu GM values preserved and obsolete fields removed, got %+v", hero.BonusTrait)
	}
}

// TestXuChuTraitSchemasExposeOnlyApplicableGMFields 验证虎痴暴露概率与破防比例，虎虎生威只暴露固定属性。
func TestXuChuTraitSchemasExposeOnlyApplicableGMFields(t *testing.T) {
	huchi, ok := general.Get("huchi_chongzhen")
	if !ok {
		t.Fatal("huchi_chongzhen trait not registered")
	}
	huchiFields := map[string]general.ParamField{}
	for _, field := range huchi.ParamSchema() {
		huchiFields[field.Key] = field
	}
	if huchiFields["triggerChance"].Default != 0.5 || huchiFields["enemyDefenseReductionRate"].Default != 0.3 {
		t.Fatalf("expected Huchi GM defaults 50%%/-30%%, fields=%+v", huchiFields)
	}

	huhu, ok := general.Get("huhu_shengwei")
	if !ok {
		t.Fatal("huhu_shengwei trait not registered")
	}
	huhuFields := map[string]general.ParamField{}
	for _, field := range huhu.ParamSchema() {
		huhuFields[field.Key] = field
	}
	if huhuFields["unitAttackFlat"].Default != 12 || huhuFields["unitSpeedFlat"].Default != 5 {
		t.Fatalf("expected Huhu GM defaults +12/+5, fields=%+v", huhuFields)
	}
	if _, exists := huhuFields["triggerChance"]; exists {
		t.Fatalf("expected permanent Huhu passive to omit triggerChance, fields=%+v", huhuFields)
	}
}

// TestNormalizeXiahouyuanLegacyTraitParams 验证夏侯渊旧行军与旧盾阵参数迁移为当前规则。
func TestNormalizeXiahouyuanLegacyTraitParams(t *testing.T) {
	cfg := NormalizeGeneralsConfig(GeneralsConfig{Heroes: map[string]GeneralHeroConfig{
		"xiahouyuan": {
			SpecialTrait: GeneralTraitConfig{
				TraitID: "jixing_benxi", Scope: "self_army",
				Params: map[string]float64{"speedBonusRate": 0.2, "minMarchSeconds": 60, "triggerChance": 1},
			},
			BonusTrait: GeneralTraitConfig{
				TraitID: "dunzhen_fangyu", Scope: "self_army", AllowedSides: []string{"defender", "reinforcement"},
				Params: map[string]float64{"defenseBonusRate": 0.35},
			},
		},
	}})
	hero := cfg.Heroes["xiahouyuan"]
	if hero.SpecialTrait.TargetUnitType != "qiQiYing" || !reflect.DeepEqual(hero.SpecialTrait.Params, map[string]float64{"unitAttackFlat": 18, "unitSpeedFlat": 5}) {
		t.Fatalf("expected legacy march trait migrated to passive unit stats, got %+v", hero.SpecialTrait)
	}
	if !reflect.DeepEqual(hero.BonusTrait.Params, map[string]float64{"defenseBonusRate": 0.3, "triggerChance": 0.6}) || !reflect.DeepEqual(hero.BonusTrait.AllowedSides, []string{"defender", "reinforcement"}) {
		t.Fatalf("expected legacy defense trait migrated to 60%%/+30%%, got %+v", hero.BonusTrait)
	}
}

// TestXiahouyuanTraitSchemasExposeOnlyApplicableGMFields 验证 GM 可配置固定值与盾阵概率，不给永久被动伪造概率字段。
func TestXiahouyuanTraitSchemasExposeOnlyApplicableGMFields(t *testing.T) {
	passive, ok := general.Get("jixing_benxi")
	if !ok {
		t.Fatal("jixing_benxi trait not registered")
	}
	passiveFields := map[string]general.ParamField{}
	for _, field := range passive.ParamSchema() {
		passiveFields[field.Key] = field
	}
	if passiveFields["unitAttackFlat"].Default != 18 || passiveFields["unitSpeedFlat"].Default != 5 {
		t.Fatalf("expected passive GM defaults +18/+5, fields=%+v", passiveFields)
	}
	if _, exists := passiveFields["triggerChance"]; exists {
		t.Fatalf("expected permanent passive to omit triggerChance, fields=%+v", passiveFields)
	}

	shield, ok := general.Get("dunzhen_fangyu")
	if !ok {
		t.Fatal("dunzhen_fangyu trait not registered")
	}
	shieldFields := map[string]general.ParamField{}
	for _, field := range shield.ParamSchema() {
		shieldFields[field.Key] = field
	}
	if shieldFields["defenseBonusRate"].Default != 0.3 || shieldFields["triggerChance"].Default != 0.6 {
		t.Fatalf("expected shield GM defaults 30%%/60%%, fields=%+v", shieldFields)
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
