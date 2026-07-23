// 本文件锁定正式将领配置中的战前减攻、破防、加防数值和适用方向。
package game

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// TestFormalPreBattleDefenseTraitConfigsMatchDesign 逐项核对十项战前防御相关特性的正式配置。
func TestFormalPreBattleDefenseTraitConfigsMatchDesign(t *testing.T) {
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
		allowedSides   []string
		targetUnitType string
		params         map[string]float64
	}{
		{generalID: "caocao", traitID: "weiwu_tongyu", traitType: "bonus", scope: "self_army", allowedSides: []string{"defender", "reinforcement"}, params: map[string]float64{"defenseBonusRate": 0.15}},
		{generalID: "simayi", traitID: "mouding_houfa", traitType: "bonus", scope: "self_army", allowedSides: []string{"defender", "reinforcement"}, params: map[string]float64{"defenseBonusRate": 0.35, "triggerChance": 0.35}},
		{generalID: "zhenmi", traitID: "meihuo_raozhen", traitType: "bonus", scope: "enemy_army", allowedSides: []string{"attacker"}, params: map[string]float64{"enemyDefenseReductionRate": 0.25, "triggerChance": 0.5}},
		{generalID: "xuchu", traitID: "huchi_chongzhen", traitType: "special", scope: "enemy_army", allowedSides: []string{"attacker"}, params: map[string]float64{"triggerChance": 0.5, "enemyDefenseReductionRate": 0.3}},
		{generalID: "dianwei", traitID: "huzhu_xuezhan", traitType: "special", scope: "self_army", allowedSides: []string{"defender", "reinforcement"}, targetUnitType: "jinWeiSoldier", params: map[string]float64{"triggerChance": 1, "generalDefenseFlat": 20}},
		{generalID: "xiahouyuan", traitID: "dunzhen_fangyu", traitType: "bonus", scope: "self_army", allowedSides: []string{"defender", "reinforcement"}, params: map[string]float64{"defenseBonusRate": 0.3, "triggerChance": 0.6}},
		{generalID: "huangzhong", traitID: "baibu_chuanyang", traitType: "special", scope: "enemy_army", allowedSides: []string{"attacker"}, params: map[string]float64{"triggerChance": 0.35, "enemyDefenseReductionRate": 0.2}},
		{generalID: "weiyan", traitID: "qibing_raohou", traitType: "special", scope: "enemy_army", allowedSides: []string{"attacker"}, params: map[string]float64{"triggerChance": 0.35, "enemyDefenseReductionRate": 0.2}},
		{generalID: "weiyan", traitID: "gushou_hanzhong", traitType: "bonus", scope: "self_army", allowedSides: []string{"defender", "reinforcement"}, params: map[string]float64{"generalDefenseFlat": 20}},
		{generalID: "sunquan", traitID: "jiangdong_gushou", traitType: "bonus", scope: "self_army", allowedSides: []string{"defender", "reinforcement"}, params: map[string]float64{"triggerChance": 0.5, "defenseBonusRate": 0.5}},
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
			if !reflect.DeepEqual(trait.AllowedSides, tc.allowedSides) || len(trait.AllowedScenes) != 0 || trait.TargetUnitType != tc.targetUnitType {
				t.Fatalf("unexpected target or scene constraints: %+v", trait)
			}
			if !reflect.DeepEqual(trait.Params, tc.params) {
				t.Fatalf("unexpected formal parameters: %+v", trait.Params)
			}
		})
	}
}
