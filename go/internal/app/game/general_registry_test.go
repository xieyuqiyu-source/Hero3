package game

import "testing"

// 本文件验证武将配置注册表入口。

func TestGeneralRegistryExposesHeroConfigs(t *testing.T) {
	hero, exists := GetGeneralHeroConfig("zhenmi")
	if !exists {
		t.Fatal("expected zhenmi config to exist")
	}
	if hero.ID != "zhenmi" {
		t.Fatalf("expected zhenmi id, got %q", hero.ID)
	}
	if !GeneralRegistered("zhouyu") {
		t.Fatal("expected zhouyu to be registered")
	}
	if GeneralRegistered("unknown_general") {
		t.Fatal("expected unknown_general to be unregistered")
	}
}

func TestListGeneralHeroConfigsReturnsCopy(t *testing.T) {
	configs := ListGeneralHeroConfigs()
	delete(configs, "zhenmi")
	if !GeneralRegistered("zhenmi") {
		t.Fatal("expected mutating listed configs not to change active registry")
	}

	configs = ListGeneralHeroConfigs()
	hero := configs["zhenmi"]
	hero.Buffs[StatProductionBonus] = 9
	configs["zhenmi"] = hero

	current, exists := GetGeneralHeroConfig("zhenmi")
	if !exists {
		t.Fatal("expected zhenmi config to exist")
	}
	if current.Buffs[StatProductionBonus] == 9 {
		t.Fatal("expected mutating listed hero buffs not to change active registry")
	}
}
