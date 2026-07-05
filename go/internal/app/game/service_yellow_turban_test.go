package game

import (
	"testing"
	"time"
)

// 本文件验证黄巾起义口粮压力和派兵规则。

func TestYellowTurbanFoodPressureUsesThousandTentCamp(t *testing.T) {
	if err := LoadUnitsConfig("../../../config/units"); err != nil {
		t.Fatalf("LoadUnitsConfig failed: %v", err)
	}
	cfg := defaultYellowTurbanConfig()
	state := newPlayerState("yt_food_player", "黄巾测试", "wei", "", time.Now())
	state.Buildings = append(state.Buildings, Building{ID: "thousand_tent_camp-test", Type: ThousandTentCampType, Level: 2})
	state.Army = []ArmyUnit{{UnitType: "qingZhouArmy", Amount: 200000}}

	pressure := CalculateFoodPressure(state, cfg)
	if pressure.CurrentFood != 400000 {
		t.Fatalf("expected current food 400000, got %d", pressure.CurrentFood)
	}
	if pressure.FoodCapacity != 300000 {
		t.Fatalf("expected capacity 300000, got %d", pressure.FoodCapacity)
	}
	if !pressure.OverCapacity {
		t.Fatal("expected over capacity")
	}
}

func TestYellowTurbanCheckDoesNotSpawnWhenSafe(t *testing.T) {
	if err := LoadUnitsConfig("../../../config/units"); err != nil {
		t.Fatalf("LoadUnitsConfig failed: %v", err)
	}
	SetYellowTurbanConfig(defaultYellowTurbanConfig())
	repo := NewMemoryRepository()
	svc := NewServiceWithRepository(repo)
	now := time.Now()
	account := Account{ID: "account_yt_safe", Username: "yt_safe", CreatedAt: now}
	if err := repo.CreateAccount(account); err != nil {
		t.Fatalf("CreateAccount failed: %v", err)
	}
	state := newPlayerState("player_yt_safe", "安全玩家", "wei", "", now)
	state.Army = []ArmyUnit{{UnitType: "qingZhouArmy", Amount: 1000}}
	if err := repo.CreatePlayer(account.ID, state, now); err != nil {
		t.Fatalf("CreatePlayer failed: %v", err)
	}

	result, err := svc.CheckYellowTurbanForPlayer(state.Player.ID)
	if err != nil {
		t.Fatalf("CheckYellowTurbanForPlayer failed: %v", err)
	}
	if result.Spawned {
		t.Fatal("expected no march for safe player")
	}
}

func TestYellowTurbanCheckSpawnsFromCurrentFood(t *testing.T) {
	if err := LoadUnitsConfig("../../../config/units"); err != nil {
		t.Fatalf("LoadUnitsConfig failed: %v", err)
	}
	SetYellowTurbanConfig(defaultYellowTurbanConfig())
	repo := NewMemoryRepository()
	svc := NewServiceWithRepository(repo)
	now := time.Now()
	account := Account{ID: "account_yt_spawn", Username: "yt_spawn", CreatedAt: now}
	if err := repo.CreateAccount(account); err != nil {
		t.Fatalf("CreateAccount failed: %v", err)
	}
	state := newPlayerState("player_yt_spawn", "超限玩家", "wei", "", now)
	state.Army = []ArmyUnit{{UnitType: "qingZhouArmy", Amount: 80000}}
	if err := repo.CreatePlayer(account.ID, state, now); err != nil {
		t.Fatalf("CreatePlayer failed: %v", err)
	}

	result, err := svc.CheckYellowTurbanForPlayer(state.Player.ID)
	if err != nil {
		t.Fatalf("CheckYellowTurbanForPlayer failed: %v", err)
	}
	if !result.Spawned || result.March == nil {
		t.Fatalf("expected spawned march, got %+v", result)
	}
	if totalFoodForFaction(result.March.SourceFaction, result.March.Troops) <= 0 {
		t.Fatalf("expected generated troops, got %+v", result.March.Troops)
	}
}

func TestYellowTurbanCheckRespectsMaxIncoming(t *testing.T) {
	if err := LoadUnitsConfig("../../../config/units"); err != nil {
		t.Fatalf("LoadUnitsConfig failed: %v", err)
	}
	cfg := defaultYellowTurbanConfig()
	cfg.MaxIncomingMarchesPerPlayer = 1
	for i := range cfg.RiskLevels {
		cfg.RiskLevels[i].MaxIncoming = 1
	}
	SetYellowTurbanConfig(cfg)
	repo := NewMemoryRepository()
	svc := NewServiceWithRepository(repo)
	now := time.Now()
	account := Account{ID: "account_yt_cap", Username: "yt_cap", CreatedAt: now}
	if err := repo.CreateAccount(account); err != nil {
		t.Fatalf("CreateAccount failed: %v", err)
	}
	state := newPlayerState("player_yt_cap", "上限玩家", "wei", "", now)
	state.Army = []ArmyUnit{{UnitType: "qingZhouArmy", Amount: 80000}}
	if err := repo.CreatePlayer(account.ID, state, now); err != nil {
		t.Fatalf("CreatePlayer failed: %v", err)
	}
	first, err := svc.CheckYellowTurbanForPlayer(state.Player.ID)
	if err != nil || !first.Spawned {
		t.Fatalf("expected first spawn, result=%+v err=%v", first, err)
	}
	second, err := svc.CheckYellowTurbanForPlayer(state.Player.ID)
	if err != nil {
		t.Fatalf("second check failed: %v", err)
	}
	if second.Spawned {
		t.Fatal("expected max incoming to block second spawn")
	}
}

func totalFoodForFaction(faction string, troops map[string]int) int {
	total := 0
	for unitID, amount := range troops {
		cfg, ok := GetUnitConfig(faction, unitID)
		if ok {
			total += amount * cfg.Stats["upkeep"]
		}
	}
	return total
}
