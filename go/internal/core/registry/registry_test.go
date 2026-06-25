package registry

import "testing"

func TestRegisterResourceType(t *testing.T) {
	name := "test_resource_registry_core"
	if err := RegisterResourceType(ResourceTypeDefinition{Type: name, Description: "测试资源"}); err != nil {
		t.Fatalf("RegisterResourceType failed: %v", err)
	}
	if !IsResourceTypeRegistered(name) {
		t.Fatalf("expected resource type %q to be registered", name)
	}
	if err := RegisterResourceType(ResourceTypeDefinition{Type: name}); err == nil {
		t.Fatal("expected duplicate resource registration to fail")
	}
}

func TestRegisterRewardType(t *testing.T) {
	name := "test_reward_registry_core"
	if err := RegisterRewardType(RewardTypeDefinition{Type: name, Description: "测试奖励", RequiresAccount: true}); err != nil {
		t.Fatalf("RegisterRewardType failed: %v", err)
	}
	def, ok := GetRewardTypeDefinition(name)
	if !ok || !def.RequiresAccount {
		t.Fatalf("expected reward definition to be registered, got %+v ok=%v", def, ok)
	}
	if err := RegisterRewardType(RewardTypeDefinition{Type: name}); err == nil {
		t.Fatal("expected duplicate reward registration to fail")
	}
}

func TestRegisterEventType(t *testing.T) {
	name := "test.event.registry.core"
	if err := RegisterEventType(EventTypeDefinition{Type: name, Description: "测试事件"}); err != nil {
		t.Fatalf("RegisterEventType failed: %v", err)
	}
	if !IsEventTypeRegistered(name) {
		t.Fatalf("expected event type %q to be registered", name)
	}
	if err := RegisterEventType(EventTypeDefinition{Type: name}); err == nil {
		t.Fatal("expected duplicate event registration to fail")
	}
}
