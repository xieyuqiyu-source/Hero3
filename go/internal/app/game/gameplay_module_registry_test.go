package game

import "testing"

// 本文件验证玩法模块边界注册表。

func TestGameplayModuleRegistryIncludesMailAndMiniGame(t *testing.T) {
	mail, exists := GetGameplayModuleDefinition("mail")
	if !exists {
		t.Fatal("expected mail gameplay module to be registered")
	}
	if mail.RepositoryPort != "MailRepository" {
		t.Fatalf("expected mail repository port, got %q", mail.RepositoryPort)
	}

	minigame, exists := GetGameplayModuleDefinition("minigame")
	if !exists {
		t.Fatal("expected minigame gameplay module to be registered")
	}
	if minigame.RewardEntrypoint != "ApplyRewardsToStateWithContext" {
		t.Fatalf("expected minigame reward entrypoint, got %q", minigame.RewardEntrypoint)
	}
}

func TestGameplayModuleDefinitionsReturnCopy(t *testing.T) {
	defs := ListGameplayModuleDefinitions()
	if len(defs) == 0 {
		t.Fatal("expected gameplay modules to exist")
	}
	defs[0].BoundaryRules = append(defs[0].BoundaryRules, "mutated")

	current, exists := GetGameplayModuleDefinition(defs[0].ID)
	if !exists {
		t.Fatalf("expected gameplay module %q to exist", defs[0].ID)
	}
	for _, rule := range current.BoundaryRules {
		if rule == "mutated" {
			t.Fatal("expected mutating listed module definitions not to change active registry")
		}
	}
}

func TestBootstrapIncludesGameplayModules(t *testing.T) {
	service := NewServiceWithRepository(NewMemoryRepository())
	bootstrap := service.Bootstrap()
	if !stringSliceContains(bootstrap.Modules, "mail") {
		t.Fatalf("expected bootstrap modules to include mail, got %+v", bootstrap.Modules)
	}
	if !stringSliceContains(bootstrap.Modules, "minigame") {
		t.Fatalf("expected bootstrap modules to include minigame, got %+v", bootstrap.Modules)
	}
}

func stringSliceContains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
