package game

import (
	"testing"
	"time"
)

// 本文件验证黄巾起义口粮压力和派兵规则。

func TestYellowTurbanDefaultConfigIsValid(t *testing.T) {
	if err := ValidateYellowTurbanConfig(defaultYellowTurbanConfig()); err != nil {
		t.Fatalf("default yellow turban config should be valid: %v", err)
	}
}

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

func TestYellowTurbanDefenseUsesHomeGeneralAndGrantsExp(t *testing.T) {
	if err := LoadFactionsConfig("../../../config/factions.json"); err != nil {
		t.Fatalf("LoadFactionsConfig failed: %v", err)
	}
	if err := LoadGeneralsConfig("../../../config/generals.json"); err != nil {
		t.Fatalf("LoadGeneralsConfig failed: %v", err)
	}
	if err := LoadUnitsConfig("../../../config/units"); err != nil {
		t.Fatalf("LoadUnitsConfig failed: %v", err)
	}
	SetYellowTurbanConfig(defaultYellowTurbanConfig())
	repo := NewMemoryRepository()
	svc := NewServiceWithRepository(repo)
	now := time.Now().UTC()
	account := Account{ID: "account_yt_general", Username: "yt_general", CreatedAt: now}
	if err := repo.CreateAccount(account); err != nil {
		t.Fatalf("CreateAccount failed: %v", err)
	}
	state := newPlayerState("player_yt_general", "防守玩家", "shu", "liubei", now)
	EnsureGeneralRoster(&state, now)
	state.Army = []ArmyUnit{{UnitType: "greedyWolf", Amount: 5000}}
	if err := repo.CreatePlayer(account.ID, state, now); err != nil {
		t.Fatalf("CreatePlayer failed: %v", err)
	}

	march, err := repo.CreateYellowTurbanMarch(YellowTurbanMarch{
		ID:              "yt_general_march",
		TargetPlayerID:  state.Player.ID,
		SourceCityID:    "yt_wei_1",
		SourceName:      "黄巾军·魏地",
		SourceFaction:   "wei",
		SourceRegionID:  "wei",
		RiskLevelID:     1,
		RiskLevelName:   "黄巾·流寇",
		PlayerFood:      10000,
		FoodCapacity:    1000,
		Pressure:        10,
		Troops:          map[string]int{"qingZhouArmy": 1200},
		Status:          YellowTurbanMarchStatusMarching,
		DurationSeconds: 1,
		StartedAt:       now.Add(-2 * time.Minute).Format(resourceDateLayout),
		ArrivesAt:       now.Add(-time.Minute).Format(resourceDateLayout),
		CreatedAt:       now.Add(-2 * time.Minute).Format(resourceDateLayout),
		UpdatedAt:       now.Add(-2 * time.Minute).Format(resourceDateLayout),
	})
	if err != nil {
		t.Fatalf("CreateYellowTurbanMarch failed: %v", err)
	}

	report, err := svc.ResolveYellowTurbanMarch(march.ID)
	if err != nil {
		t.Fatalf("ResolveYellowTurbanMarch failed: %v", err)
	}
	if len(report.PvpDefenderGenerals) != 1 || report.PvpDefenderGenerals[0].ID != "liubei" {
		t.Fatalf("expected yellow turban defense report to show defender general, got %+v", report.PvpDefenderGenerals)
	}
	if report.GeneralExpGained <= 0 {
		t.Fatalf("expected defender general exp in report, got %+v", report)
	}
	if report.Detail == nil || report.Detail.Rewards.GeneralExp != report.GeneralExpGained {
		t.Fatalf("expected standard report detail to show general exp, detail=%+v reportExp=%d", report.Detail, report.GeneralExpGained)
	}
	updated, err := repo.GetState(state.Player.ID)
	if err != nil {
		t.Fatalf("GetState failed: %v", err)
	}
	if got := pvpTestGeneralExp(updated, "liubei"); got != report.GeneralExpGained {
		t.Fatalf("expected liubei exp %d, got %d", report.GeneralExpGained, got)
	}
}

func TestYellowTurbanDefenseUsesStationedReinforcements(t *testing.T) {
	if err := LoadFactionsConfig("../../../config/factions.json"); err != nil {
		t.Fatalf("LoadFactionsConfig failed: %v", err)
	}
	if err := LoadGeneralsConfig("../../../config/generals.json"); err != nil {
		t.Fatalf("LoadGeneralsConfig failed: %v", err)
	}
	if err := LoadUnitsConfig("../../../config/units"); err != nil {
		t.Fatalf("LoadUnitsConfig failed: %v", err)
	}
	SetYellowTurbanConfig(defaultYellowTurbanConfig())
	repo := NewMemoryRepository()
	svc := NewServiceWithRepository(repo)
	now := time.Now().UTC()
	defenderAccount := Account{ID: "account_yt_rein_def", Username: "yt_rein_def", CreatedAt: now}
	helperAccount := Account{ID: "account_yt_rein_helper", Username: "yt_rein_helper", CreatedAt: now}
	if err := repo.CreateAccount(defenderAccount); err != nil {
		t.Fatalf("CreateAccount defender failed: %v", err)
	}
	if err := repo.CreateAccount(helperAccount); err != nil {
		t.Fatalf("CreateAccount helper failed: %v", err)
	}
	defender := newPlayerState("player_yt_rein_def", "被援玩家", "shu", "liubei", now)
	helper := newPlayerState("player_yt_rein_helper", "协防玩家", "wu", "sunquan", now)
	defender.Army = []ArmyUnit{{UnitType: "greedyWolf", Amount: 1}}
	if err := repo.CreatePlayer(defenderAccount.ID, defender, now); err != nil {
		t.Fatalf("CreatePlayer defender failed: %v", err)
	}
	if err := repo.CreatePlayer(helperAccount.ID, helper, now); err != nil {
		t.Fatalf("CreatePlayer helper failed: %v", err)
	}
	reinforcement := Reinforcement{
		ID:                "reinforcement_yt_report",
		FromPlayerID:      helper.Player.ID,
		FromPlayerName:    helper.Player.Nickname,
		FromPlayerFaction: helper.Player.Faction,
		ToPlayerID:        defender.Player.ID,
		ToPlayerName:      defender.Player.Nickname,
		ToPlayerFaction:   defender.Player.Faction,
		OwnerPlayerID:     helper.Player.ID,
		HostPlayerID:      defender.Player.ID,
		SourceType:        GarrisonSourceReinforcement,
		SourceID:          "reinforcement_yt_report",
		TargetType:        ReinforcementTargetPlayerCity,
		TargetID:          defender.Player.ID,
		Status:            ReinforcementStatusStationed,
		Troops:            map[string]int{"shadowGuard": 40},
		RemainingTroops:   map[string]int{"shadowGuard": 40},
		Losses:            map[string]int{},
		Rules:             defaultGarrisonRules(GarrisonSourceReinforcement),
		SentAt:            now.Add(-4 * time.Hour).UTC().Format(resourceDateLayout),
		ArrivedAt:         now.Add(-3 * time.Hour).UTC().Format(resourceDateLayout),
		CreatedAt:         now.Add(-4 * time.Hour).UTC().Format(resourceDateLayout),
		UpdatedAt:         now.Add(-3 * time.Hour).UTC().Format(resourceDateLayout),
	}
	repo.reinforcements[reinforcement.ID] = reinforcement
	march, err := repo.CreateYellowTurbanMarch(YellowTurbanMarch{
		ID:              "yt_rein_march",
		TargetPlayerID:  defender.Player.ID,
		SourceCityID:    "yt_wei_1",
		SourceName:      "黄巾军·魏地",
		SourceFaction:   "wei",
		SourceRegionID:  "wei",
		RiskLevelID:     2,
		RiskLevelName:   "黄巾·聚众",
		PlayerFood:      10000,
		FoodCapacity:    1000,
		Pressure:        10,
		Troops:          map[string]int{"qingZhouArmy": 500},
		Status:          YellowTurbanMarchStatusMarching,
		DurationSeconds: 1,
		StartedAt:       now.Add(-2 * time.Minute).Format(resourceDateLayout),
		ArrivesAt:       now.Add(-time.Minute).Format(resourceDateLayout),
		CreatedAt:       now.Add(-2 * time.Minute).Format(resourceDateLayout),
		UpdatedAt:       now.Add(-2 * time.Minute).Format(resourceDateLayout),
	})
	if err != nil {
		t.Fatalf("CreateYellowTurbanMarch failed: %v", err)
	}

	report, err := svc.ResolveYellowTurbanMarch(march.ID)
	if err != nil {
		t.Fatalf("ResolveYellowTurbanMarch failed: %v", err)
	}
	if len(report.PvpReinforcements) != 1 || report.PvpReinforcements[0].ReinforcementID != reinforcement.ID {
		t.Fatalf("expected yellow turban defense report to include reinforcement, got %+v", report.PvpReinforcements)
	}
	if len(report.PvpReinforcementLosses[reinforcement.ID]) == 0 {
		t.Fatalf("expected reinforcement losses in yellow turban report, got %+v", report.PvpReinforcementLosses)
	}
	helperReports, total, err := repo.ListReportsByQuery(BattleReportQuery{PlayerID: helper.Player.ID, ViewType: ReportViewReinforcement, Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("ListReportsByQuery helper failed: %v", err)
	}
	if total != 1 || len(helperReports) != 1 {
		t.Fatalf("expected one helper yellow turban reinforcement report, total=%d reports=%+v", total, helperReports)
	}
	if helperReports[0].SourceType != ReportSourceYellowTurban || helperReports[0].BattleType != BattleTypeYellowTurban {
		t.Fatalf("expected yellow turban reinforcement report type, got %+v", helperReports[0])
	}
	updated := repo.reinforcements[reinforcement.ID]
	if updated.LastBattleReportID != helperReports[0].ID {
		t.Fatalf("expected reinforcement last report %s, got %+v", helperReports[0].ID, updated)
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
