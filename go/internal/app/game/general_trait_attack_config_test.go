// 本文件锁定正式将领配置中的战前攻击加成、目标兵种和适用场景。
package game

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// TestFormalAttackTraitConfigsMatchDesign 逐项核对七项攻击加成，曹操防御特性不得混入进攻矩阵。
func TestFormalAttackTraitConfigsMatchDesign(t *testing.T) {
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
		targetUnitType string
		allowedSides   []string
		allowedScenes  []string
		params         map[string]float64
	}{
		{generalID: "dianwei", traitID: "sizhandaodi", targetUnitType: "infantry", allowedSides: []string{"attacker"}, params: map[string]float64{"attackBonusRate": 0.35}},
		{generalID: "zhangliao", traitID: "weizhen_xiaoyao", targetUnitType: "cavalry", allowedSides: []string{"attacker"}, params: map[string]float64{"attackBonusRate": 0.35}},
		{generalID: "guanyu", traitID: "wusheng_pojun", allowedSides: []string{"attacker"}, params: map[string]float64{"attackBonusRate": 0.2}},
		{generalID: "zhangfei", traitID: "wanren_nuhou", targetUnitType: "infantry", allowedSides: []string{"attacker"}, params: map[string]float64{"attackBonusRate": 0.2}},
		{generalID: "sunce", traitID: "xiaobawang_tieqi", targetUnitType: "overlordRider", allowedSides: []string{"attacker"}, params: map[string]float64{"unitAttackFlat": 50}},
		{generalID: "zhouyu", traitID: "meizhoulang_junlue", allowedSides: []string{"attacker"}, params: map[string]float64{"attackBonusRate": 0.05}},
		{generalID: "ganning", traitID: "jinfan_qixi", allowedSides: []string{"attacker"}, allowedScenes: []string{"plunder"}, params: map[string]float64{"attackBonusRate": 0.1}},
	}
	for _, tc := range tests {
		t.Run(tc.traitID, func(t *testing.T) {
			hero, ok := cfg.Heroes[tc.generalID]
			if !ok {
				t.Fatalf("formal general %s not found", tc.generalID)
			}
			trait := hero.BonusTrait
			if !trait.Enabled || trait.TraitID != tc.traitID || trait.Scope != "self_army" {
				t.Fatalf("unexpected formal trait identity: %+v", trait)
			}
			if trait.TargetUnitType != tc.targetUnitType || !reflect.DeepEqual(trait.AllowedSides, tc.allowedSides) || !reflect.DeepEqual(trait.AllowedScenes, tc.allowedScenes) {
				t.Fatalf("unexpected target or scene constraints: %+v", trait)
			}
			if !reflect.DeepEqual(trait.Params, tc.params) {
				t.Fatalf("unexpected attack parameters: %+v", trait.Params)
			}
		})
	}
}
