package game

import "testing"

// 本文件验证道具配置注册表入口。

func TestItemRegistryExposesItemDefinitions(t *testing.T) {
	loadTestItemsConfig(t)

	item, exists := GetItemDefinition("test_general_exp_small")
	if !exists {
		t.Fatal("expected test item config to exist")
	}
	if item.ID != "test_general_exp_small" {
		t.Fatalf("expected item id test_general_exp_small, got %q", item.ID)
	}
	if !ItemRegistered("test_general_exp_small") {
		t.Fatal("expected test_general_exp_small to be registered")
	}
	if ItemRegistered("unknown_item") {
		t.Fatal("expected unknown_item to be unregistered")
	}
}

func TestListItemConfigsReturnsCopy(t *testing.T) {
	loadTestItemsConfig(t)

	items := GetItemsConfig()
	delete(items, "test_general_exp_small")
	if !ItemRegistered("test_general_exp_small") {
		t.Fatal("expected mutating listed items not to change active registry")
	}

	items = GetItemsConfig()
	item := items["test_general_exp_small"]
	item.Effects[0].Amount = 999
	items["test_general_exp_small"] = item

	current, exists := GetItemDefinition("test_general_exp_small")
	if !exists {
		t.Fatal("expected test item config to exist")
	}
	if current.Effects[0].Amount == 999 {
		t.Fatal("expected mutating listed item effects not to change active registry")
	}
}
