// 本文件测试购买加成的叠加、续时和配置化选项。
package game

import (
	"testing"
	"time"
)

// newBoostTestService 构造可购买加成的测试服务。
func newBoostTestService(t *testing.T) (*Service, *MemoryRepository, string) {
	t.Helper()
	now := time.Now().UTC()
	repo := NewMemoryRepository()
	account := Account{ID: "account_boost", Username: "boost", PasswordHash: "hash", CreatedAt: now}
	if err := repo.CreateAccount(account); err != nil {
		t.Fatalf("CreateAccount failed: %v", err)
	}
	state := newPlayerState("player_boost", "加成测试", "wei", "caocao", now)
	state.CityGold = 10000
	if err := repo.CreatePlayer(account.ID, state, now); err != nil {
		t.Fatalf("CreatePlayer failed: %v", err)
	}
	return NewServiceWithRepository(repo), repo, state.Player.ID
}

// TestPurchaseBoostStacksMultiplierAndExtendsTime 验证产量加成可以叠倍率并续时。
func TestPurchaseBoostStacksMultiplierAndExtendsTime(t *testing.T) {
	svc, _, playerID := newBoostTestService(t)
	first, err := svc.PurchaseBoost(playerID, 2, 1)
	if err != nil {
		t.Fatalf("first PurchaseBoost failed: %v", err)
	}
	firstEnd, err := time.Parse(resourceDateLayout, first.ProductionBoostEnd)
	if err != nil {
		t.Fatalf("parse first end: %v", err)
	}
	second, err := svc.PurchaseBoost(playerID, 4, 6)
	if err != nil {
		t.Fatalf("second PurchaseBoost failed: %v", err)
	}
	secondEnd, err := time.Parse(resourceDateLayout, second.ProductionBoostEnd)
	if err != nil {
		t.Fatalf("parse second end: %v", err)
	}
	if second.ProductionBoost != 8 {
		t.Fatalf("expected stacked production boost x8, got x%d", second.ProductionBoost)
	}
	if secondEnd.Before(firstEnd.Add(6*time.Hour-5*time.Second)) || secondEnd.After(firstEnd.Add(6*time.Hour+5*time.Second)) {
		t.Fatalf("expected second end to extend from first end by 6h, first=%s second=%s", firstEnd, secondEnd)
	}
}

// TestPurchaseCapacityBoostStacksMultiplierAndExtendsTime 验证容量加成可以叠倍率并续时。
func TestPurchaseCapacityBoostStacksMultiplierAndExtendsTime(t *testing.T) {
	svc, _, playerID := newBoostTestService(t)
	first, err := svc.PurchaseCapacityBoost(playerID, 2, 1)
	if err != nil {
		t.Fatalf("first PurchaseCapacityBoost failed: %v", err)
	}
	firstEnd, err := time.Parse(resourceDateLayout, first.CapacityBoostEnd)
	if err != nil {
		t.Fatalf("parse first end: %v", err)
	}
	second, err := svc.PurchaseCapacityBoost(playerID, 4, 6)
	if err != nil {
		t.Fatalf("second PurchaseCapacityBoost failed: %v", err)
	}
	secondEnd, err := time.Parse(resourceDateLayout, second.CapacityBoostEnd)
	if err != nil {
		t.Fatalf("parse second end: %v", err)
	}
	if second.CapacityBoost != 8 {
		t.Fatalf("expected stacked capacity boost x8, got x%d", second.CapacityBoost)
	}
	if secondEnd.Before(firstEnd.Add(6*time.Hour-5*time.Second)) || secondEnd.After(firstEnd.Add(6*time.Hour+5*time.Second)) {
		t.Fatalf("expected second end to extend from first end by 6h, first=%s second=%s", firstEnd, secondEnd)
	}
}

// TestBoostOptionsFollowBalanceConfig 验证倍率和时长开放项来自 balance 配置。
func TestBoostOptionsFollowBalanceConfig(t *testing.T) {
	original := GetBalanceConfig()
	t.Cleanup(func() {
		if err := SetBalanceConfig(original); err != nil {
			t.Fatalf("restore balance config: %v", err)
		}
	})
	custom := GetBalanceConfig()
	custom.BoostMultiplierFactor = map[int]int{3: 2}
	custom.BoostDurationFactor = map[int]int{2: 2}
	if err := SetBalanceConfig(custom); err != nil {
		t.Fatalf("SetBalanceConfig failed: %v", err)
	}

	svc, _, playerID := newBoostTestService(t)
	if _, err := svc.PurchaseBoost(playerID, 2, 2); err != ErrInvalidBoost {
		t.Fatalf("expected ErrInvalidBoost for disabled multiplier, got %v", err)
	}
	if _, err := svc.PurchaseBoost(playerID, 3, 1); err != ErrInvalidDuration {
		t.Fatalf("expected ErrInvalidDuration for disabled hours, got %v", err)
	}
	state, err := svc.PurchaseBoost(playerID, 3, 2)
	if err != nil {
		t.Fatalf("expected custom boost option to work: %v", err)
	}
	if state.ProductionBoost != 3 {
		t.Fatalf("expected custom production boost x3, got x%d", state.ProductionBoost)
	}
}
