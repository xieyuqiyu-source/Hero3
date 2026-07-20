// 本文件锁定正式将领配置中的掠夺收益增减方向和适用场景。
package game

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// TestFormalPlunderTraitConfigsMatchDesign 逐项核对锦帆劫掠与江东号令的正式配置。
func TestFormalPlunderTraitConfigsMatchDesign(t *testing.T) {
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
		generalID   string
		traitID     string
		scope       string
		allowedSide string
		rate        float64
	}{
		{generalID: "ganning", traitID: "jinfan_jielue", scope: "self_plunder", allowedSide: "attacker", rate: 0.2},
		{generalID: "sunquan", traitID: "jiangdong_haoling", scope: "enemy_plunder", allowedSide: "defender", rate: -0.2},
	}
	for _, tc := range tests {
		t.Run(tc.traitID, func(t *testing.T) {
			hero, ok := cfg.Heroes[tc.generalID]
			if !ok {
				t.Fatalf("formal general %s not found", tc.generalID)
			}
			trait := hero.SpecialTrait
			if !trait.Enabled || trait.TraitID != tc.traitID || trait.TraitType != "special" || trait.Scope != tc.scope {
				t.Fatalf("unexpected formal plunder trait identity: %+v", trait)
			}
			if !reflect.DeepEqual(trait.AllowedSides, []string{tc.allowedSide}) || !reflect.DeepEqual(trait.AllowedScenes, []string{"plunder"}) || trait.RequiredOutcome != "" || trait.TargetUnitType != "" {
				t.Fatalf("unexpected formal plunder constraints: %+v", trait)
			}
			if !reflect.DeepEqual(trait.Params, map[string]float64{"plunderBonusRate": tc.rate}) {
				t.Fatalf("unexpected formal plunder parameters: %+v", trait.Params)
			}
		})
	}
}
