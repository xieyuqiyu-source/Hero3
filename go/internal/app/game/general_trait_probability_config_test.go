// 本文件验证正式将领配置中的显式触发概率不会丢失、错配或退回模板默认值。
package game

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// TestFormalTraitTriggerChancesMatchDesign 逐项核对正式概率特性的将领归属和配置值。
func TestFormalTraitTriggerChancesMatchDesign(t *testing.T) {
	path := filepath.Join("..", "..", "..", "config", "generals.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read formal generals config failed: %v", err)
	}
	var cfg GeneralsConfig
	if err := json.Unmarshal(raw, &cfg); err != nil {
		t.Fatalf("decode formal generals config failed: %v", err)
	}
	want := map[string]map[string]float64{
		"simayi":     {"yibing_touxi": 0.35, "mouding_houfa": 0.35},
		"zhenmi":     {"meiren": 0.5, "meihuo_raozhen": 0.5},
		"xuchu":      {"huchi_chongzhen": 0.5},
		"xiahouyuan": {"dunzhen_fangyu": 0.6},
		"dianwei":    {"huzhu_xuezhan": 1, "sizhandaodi": 0.6},
		"zhangliao":  {"weizhen_zhenhe": 0.35, "weizhen_xiaoyao": 0.6},
		"guojia":     {"guicai_yice": 1},
		"liubei":     {"renzhu_shouhu": 0.6},
		"guanyu":     {"shuiyan_qijun": 0.35, "wusheng_pojun": 0.5},
		"zhangfei":   {"zhenhe_quanjun": 0.5, "wanren_nuhou": 0.4},
		"zhugeliang": {"qimen_dunjia": 1, "wolong_mouzhi": 0.6},
		"machao":     {"xiliang_tuji": 0.35},
		"huangzhong": {"baibu_chuanyang": 0.45},
		"sunquan":    {"jiangdong_gushou": 0.5},
		"sunce":      {"xiaobawang_zhuiji": 0.35},
		"zhouyu":     {"huogong": 1},
		"luxun":      {"huoshao_lianying": 0.35},
		"lvmeng":     {"baiyi_dujiang": 0.35},
		"taishici":   {"kuairu_shandian": 0.35},
		"huanggai":   {"kurouji": 0.35},
	}
	found := map[string]map[string]float64{}
	for generalID, hero := range cfg.Heroes {
		for _, trait := range []GeneralTraitConfig{hero.SpecialTrait, hero.BonusTrait} {
			chance, explicit := trait.Params["triggerChance"]
			if !explicit {
				continue
			}
			if found[generalID] == nil {
				found[generalID] = map[string]float64{}
			}
			found[generalID][trait.TraitID] = chance
		}
	}
	if len(found) != len(want) {
		t.Fatalf("expected %d generals with explicit trait chances, got %d: %+v", len(want), len(found), found)
	}
	for generalID, traits := range want {
		for traitID, chance := range traits {
			if found[generalID][traitID] != chance {
				t.Errorf("expected %s/%s triggerChance %.2f, got %.2f", generalID, traitID, chance, found[generalID][traitID])
			}
		}
	}
}
