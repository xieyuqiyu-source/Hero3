// 本文件验证战斗配置读写不会共享可变映射或污染后续场景规则。
package combat

import "testing"

// TestCombatConfigCopiesIsolateNestedMaps 验证读取副本和保存后的全局配置都与调用方隔离。
func TestCombatConfigCopiesIsolateNestedMaps(t *testing.T) {
	original := GetCombatConfig()
	t.Cleanup(func() {
		if err := SaveCombatConfig("", original); err != nil {
			t.Fatalf("restore combat config: %v", err)
		}
	})

	readCopy := GetCombatConfig()
	readCopy.ActiveRules[ScenePVEAttack] = RuleOfficialPlunder
	attackRule := readCopy.Rules[RuleOfficialAttack]
	attackRule.Name = "被调用方修改"
	readCopy.Rules[RuleOfficialAttack] = attackRule
	weiWall := readCopy.WallConfig["wei"]
	weiWall.Base = 9
	readCopy.WallConfig["wei"] = weiWall

	current := GetCombatConfig()
	if current.ActiveRules[ScenePVEAttack] == RuleOfficialPlunder || current.Rules[RuleOfficialAttack].Name == "被调用方修改" || current.WallConfig["wei"].Base == 9 {
		t.Fatalf("expected GetCombatConfig copy to isolate global state, current=%+v", current)
	}

	if err := SaveCombatConfig("", readCopy); err != nil {
		t.Fatalf("save combat config: %v", err)
	}
	readCopy.ActiveRules[ScenePVEAttack] = RuleOfficialAttack
	attackRule.Name = "保存后再次修改"
	readCopy.Rules[RuleOfficialAttack] = attackRule
	weiWall.Base = 7
	readCopy.WallConfig["wei"] = weiWall

	saved := GetCombatConfig()
	if saved.ActiveRules[ScenePVEAttack] != RuleOfficialPlunder || saved.Rules[RuleOfficialAttack].Name != "被调用方修改" || saved.WallConfig["wei"].Base != 9 {
		t.Fatalf("expected saved combat config to isolate caller mutations, saved=%+v", saved)
	}
}
