// 本文件锁定正式将领配置中的行军加速倍率、概率和最低时长。
package game

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// TestFormalMarchTraitConfigsMatchDesign 逐项核对五项行军过程特性的正式配置。
func TestFormalMarchTraitConfigsMatchDesign(t *testing.T) {
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
	}{
		{generalID: "zhaoyun", traitID: "qijin_qichu", traitType: "bonus", params: map[string]float64{"speedBonusRate": 1, "minMarchSeconds": 60}},
		{generalID: "lvmeng", traitID: "baiyi_dujiang", traitType: "special", params: map[string]float64{"triggerChance": 0.35, "speedBonusRate": 0.2, "minMarchSeconds": 60}},
		{generalID: "lvmeng", traitID: "baiyi_jixing", traitType: "bonus", params: map[string]float64{"speedBonusRate": 0.2, "minMarchSeconds": 60}},
		{generalID: "taishici", traitID: "kuairu_shandian", traitType: "special", params: map[string]float64{"triggerChance": 0.35, "speedBonusRate": 4, "minMarchSeconds": 30}},
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
			if !trait.Enabled || trait.TraitID != tc.traitID || trait.TraitType != tc.traitType || trait.Scope != "self_army" {
				t.Fatalf("unexpected formal march trait identity: %+v", trait)
			}
			if len(trait.AllowedSides) != 0 || len(trait.AllowedScenes) != 0 || trait.RequiredOutcome != "" || trait.TargetUnitType != "" {
				t.Fatalf("unexpected formal march constraints: %+v", trait)
			}
			if !reflect.DeepEqual(trait.Params, tc.params) {
				t.Fatalf("unexpected formal march parameters: %+v", trait.Params)
			}
		})
	}
}
