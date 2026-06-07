package game

import (
	"encoding/json"
	"errors"
	"math"
	"os"
	"path/filepath"
	"testing"
	"time"

	"hero3/internal/combat"
	_ "hero3/internal/general/traits"
)

func TestSettleResourcesAddsProducedResources(t *testing.T) {
	settledAt := time.Date(2026, 5, 22, 10, 0, 0, 0, time.UTC)
	state := newPlayerState("player_test", "主公", "wei", "caocao", settledAt)
	state.Resources = ResourceState{
		Items: map[string]int{
			"wood":  100,
			"stone": 100,
			"iron":  100,
			"food":  100,
		},
		Capacity: map[string]int{
			"wood":  7500,
			"stone": 7500,
			"iron":  7500,
			"food":  7500,
		},
	}
	state.ResourceSettledAt = settledAt.Format(time.RFC3339)

	next, changed := settleResources(state, settledAt.Add(time.Hour))
	if !changed {
		t.Fatal("expected resource settlement to change state")
	}

	if next.Resources.Items["wood"] <= state.Resources.Items["wood"] {
		t.Fatalf("expected wood to grow, got %d", next.Resources.Items["wood"])
	}
	if next.Resources.Items["food"] <= state.Resources.Items["food"] {
		t.Fatalf("expected food to grow, got %d", next.Resources.Items["food"])
	}
	if next.ResourceSettledAt != settledAt.Add(time.Hour).Format(time.RFC3339) {
		t.Fatalf("expected settlement timestamp to advance, got %s", next.ResourceSettledAt)
	}
}

func TestSettleResourcesCapsAtCapacity(t *testing.T) {
	settledAt := time.Date(2026, 5, 22, 10, 0, 0, 0, time.UTC)
	state := newPlayerState("player_test", "主公", "wei", "caocao", settledAt)
	state.Resources = ResourceState{
		Items: map[string]int{
			"wood":  7499,
			"stone": 7499,
			"iron":  7499,
			"food":  7499,
		},
		Capacity: map[string]int{
			"wood":  7500,
			"stone": 7500,
			"iron":  7500,
			"food":  7500,
		},
	}
	state.ResourceSettledAt = settledAt.Format(time.RFC3339)

	next, _ := settleResources(state, settledAt.Add(24*time.Hour))
	if next.Resources.Items["wood"] != next.Resources.Capacity["wood"] {
		t.Fatalf("expected wood to cap at %d, got %d", next.Resources.Capacity["wood"], next.Resources.Items["wood"])
	}
	if next.Resources.Items["food"] != next.Resources.Capacity["food"] {
		t.Fatalf("expected food to cap at %d, got %d", next.Resources.Capacity["food"], next.Resources.Items["food"])
	}
}

func TestSettleResourcesAdvancesTimestampWhenCapacityIsFull(t *testing.T) {
	settledAt := time.Date(2026, 5, 22, 10, 0, 0, 0, time.UTC)
	state := newPlayerState("player_test", "主公", "wei", "caocao", settledAt)
	state.Resources = ResourceState{
		Items: map[string]int{
			"wood":  7500,
			"stone": 7500,
			"iron":  7500,
			"food":  7500,
		},
		Capacity: map[string]int{
			"wood":  7500,
			"stone": 7500,
			"iron":  7500,
			"food":  7500,
		},
	}
	state.ResourceSettledAt = settledAt.Format(time.RFC3339)

	next, changed := settleResources(state, settledAt.Add(time.Hour))
	if !changed {
		t.Fatal("expected full-capacity settlement to advance timestamp")
	}
	if next.ResourceSettledAt != settledAt.Add(time.Hour).Format(time.RFC3339) {
		t.Fatalf("expected settlement timestamp to advance, got %s", next.ResourceSettledAt)
	}
}

func TestResourceStateUnmarshalMigratesLegacyShape(t *testing.T) {
	var resources ResourceState
	err := json.Unmarshal([]byte(`{"wood":1200,"stone":900,"iron":600,"food":1500,"capacity":5000}`), &resources)
	if err != nil {
		t.Fatalf("unexpected unmarshal error: %v", err)
	}

	if resources.Items["wood"] != 1200 {
		t.Fatalf("expected legacy wood to migrate, got %d", resources.Items["wood"])
	}
	if resources.Capacity["iron"] != 5000 {
		t.Fatalf("expected legacy capacity to apply per resource, got %d", resources.Capacity["iron"])
	}
}

func TestResourceProductionUnmarshalMigratesLegacyShape(t *testing.T) {
	var production ResourceProduction
	err := json.Unmarshal([]byte(`{"woodPerHour":84,"stonePerHour":62,"ironPerHour":48,"foodPerHour":100}`), &production)
	if err != nil {
		t.Fatalf("unexpected unmarshal error: %v", err)
	}

	if production["wood"] != 84 {
		t.Fatalf("expected legacy wood production to migrate, got %d", production["wood"])
	}
	if production["food"] != 100 {
		t.Fatalf("expected legacy food production to migrate, got %d", production["food"])
	}
}

func TestCalculateResourceProductionUsesBalanceConfig(t *testing.T) {
	production := calculateResourceProduction([]Building{
		{Type: "wood_camp", Level: 3},
		{Type: "stone_quarry", Level: 2},
		{Type: "iron_mine", Level: 2},
		{Type: "farm", Level: 3},
	}, nil)

	if production["wood"] != 30 {
		t.Fatalf("expected wood production from config to be 30, got %d", production["wood"])
	}
	if production["food"] != 30 {
		t.Fatalf("expected food production from config to be 30, got %d", production["food"])
	}
}

func TestCalculateResourceCapacityUsesWarehouseConfig(t *testing.T) {
	capacity := calculateResourceCapacity([]Building{{Type: "warehouse", Level: 3}})

	if capacity["wood"] != 9200 {
		t.Fatalf("expected level 3 warehouse capacity to be 9200, got %d", capacity["wood"])
	}
	if capacity["food"] != 9200 {
		t.Fatalf("expected level 3 warehouse food capacity to be 9200, got %d", capacity["food"])
	}
}

func setTestGeneralsConfig(t *testing.T, cfg GeneralsConfig) {
	t.Helper()
	setTestFactionsAndGenerals(t, FactionsConfig{
		"wei": {
			Name:     "魏国",
			Generals: []GeneralInfo{{ID: "test_general", Name: "测试将领", Title: "测试"}},
		},
	}, cfg)
}

func setTestFactionsAndGenerals(t *testing.T, factions FactionsConfig, cfg GeneralsConfig) {
	t.Helper()
	original := GetGeneralsConfig()
	originalFactions := GetFactionsConfig()

	factionsMu.Lock()
	activeFactions = factions
	factionsMu.Unlock()

	if err := SetGeneralsConfig(cfg); err != nil {
		t.Fatalf("set generals config: %v", err)
	}

	t.Cleanup(func() {
		generalsMu.Lock()
		activeGenerals = original
		generalsMu.Unlock()
		factionsMu.Lock()
		activeFactions = originalFactions
		factionsMu.Unlock()
	})
}

func setTestCombatUnitsConfig(t *testing.T) {
	t.Helper()
	originalUnits := GetUnitsConfig()

	unitsMu.Lock()
	activeUnits = UnitsConfig{
		"wei": FactionUnits{
			"weiInfantry": UnitConfig{
				Name:     "魏步兵",
				Category: "infantry",
				Stats: map[string]int{
					"attack":          10,
					"infantryDefense": 10,
					"cavalryDefense":  8,
					"carryCapacity":   5,
					"upkeep":          1,
				},
			},
			"weiCavalry": UnitConfig{
				Name:     "魏骑兵",
				Category: "cavalry",
				Stats: map[string]int{
					"attack":          14,
					"infantryDefense": 8,
					"cavalryDefense":  10,
					"carryCapacity":   6,
					"upkeep":          2,
				},
			},
		},
	}
	unitsMu.Unlock()

	t.Cleanup(func() {
		unitsMu.Lock()
		activeUnits = originalUnits
		unitsMu.Unlock()
	})
}

func TestValidateGeneralsConfigRejectsUnsafeTraitParams(t *testing.T) {
	factions := FactionsConfig{
		"wei": {Generals: []GeneralInfo{{ID: "zhenmi", Name: "甄宓"}}},
	}
	cfg := GeneralsConfig{
		Enabled: true,
		Heroes: map[string]GeneralHeroConfig{
			"zhenmi": {
				ID:      "zhenmi",
				Name:    "甄宓",
				Faction: "wei",
				Enabled: true,
				Traits: []GeneralTraitConfig{{
					TraitID: "meiren",
					Enabled: true,
					Params:  map[string]float64{"captureRate": 2, "captureMax": 1000, "triggerChance": 1},
				}},
			},
		},
	}

	originalFactions := GetFactionsConfig()
	factionsMu.Lock()
	activeFactions = factions
	factionsMu.Unlock()
	t.Cleanup(func() {
		factionsMu.Lock()
		activeFactions = originalFactions
		factionsMu.Unlock()
	})

	if err := ValidateGeneralsConfig(cfg); err == nil {
		t.Fatalf("expected out-of-range trait param to be rejected")
	}
}

func TestValidateGeneralsConfigRejectsUnknownBuffKey(t *testing.T) {
	factions := FactionsConfig{
		"wei": {Generals: []GeneralInfo{{ID: "zhenmi", Name: "甄宓"}}},
	}
	cfg := GeneralsConfig{
		Enabled: true,
		Heroes: map[string]GeneralHeroConfig{
			"zhenmi": {
				ID:      "zhenmi",
				Name:    "甄宓",
				Faction: "wei",
				Enabled: true,
				Buffs:   map[string]float64{"cheatBonus": 999},
			},
		},
	}

	originalFactions := GetFactionsConfig()
	factionsMu.Lock()
	activeFactions = factions
	factionsMu.Unlock()
	t.Cleanup(func() {
		factionsMu.Lock()
		activeFactions = originalFactions
		factionsMu.Unlock()
	})

	if err := ValidateGeneralsConfig(cfg); err == nil {
		t.Fatalf("expected unknown buff key to be rejected")
	}
}

func TestCreatePlayerRejectsDisabledGeneral(t *testing.T) {
	setTestFactionsAndGenerals(t, FactionsConfig{
		"wei": {Generals: []GeneralInfo{{ID: "zhenmi", Name: "甄宓"}}},
	}, GeneralsConfig{
		Enabled: true,
		Heroes: map[string]GeneralHeroConfig{
			"zhenmi": {ID: "zhenmi", Name: "甄宓", Faction: "wei", Enabled: false},
		},
	})

	repo := NewMemoryRepository()
	service := NewServiceWithRepository(repo)
	now := time.Date(2026, 6, 4, 10, 0, 0, 0, time.UTC)
	if err := repo.CreateAccount(Account{ID: "account_disabled_general", Username: "disabled_general", PasswordHash: "hash", CreatedAt: now}); err != nil {
		t.Fatalf("create account: %v", err)
	}

	if _, _, err := service.CreatePlayer("account_disabled_general", "测试", "wei", "zhenmi"); !errors.Is(err, ErrInvalidGeneral) {
		t.Fatalf("expected ErrInvalidGeneral for disabled general, got %v", err)
	}
}

func TestCreatePlayerRejectsGeneralConfigFactionMismatch(t *testing.T) {
	original := GetGeneralsConfig()
	originalFactions := GetFactionsConfig()
	factionsMu.Lock()
	activeFactions = FactionsConfig{
		"wei": {Generals: []GeneralInfo{{ID: "zhouyu", Name: "周瑜"}}},
	}
	factionsMu.Unlock()
	generalsMu.Lock()
	activeGenerals = GeneralsConfig{
		Enabled: true,
		Heroes: map[string]GeneralHeroConfig{
			"zhouyu": {ID: "zhouyu", Name: "周瑜", Faction: "wu", Enabled: true},
		},
	}
	generalsMu.Unlock()
	t.Cleanup(func() {
		generalsMu.Lock()
		activeGenerals = original
		generalsMu.Unlock()
		factionsMu.Lock()
		activeFactions = originalFactions
		factionsMu.Unlock()
	})

	repo := NewMemoryRepository()
	service := NewServiceWithRepository(repo)
	now := time.Date(2026, 6, 4, 10, 0, 0, 0, time.UTC)
	if err := repo.CreateAccount(Account{ID: "account_mismatch_general", Username: "mismatch_general", PasswordHash: "hash", CreatedAt: now}); err != nil {
		t.Fatalf("create account: %v", err)
	}

	if _, _, err := service.CreatePlayer("account_mismatch_general", "测试", "wei", "zhouyu"); !errors.Is(err, ErrInvalidGeneral) {
		t.Fatalf("expected ErrInvalidGeneral for mismatched general faction, got %v", err)
	}
}

func TestApplyHeroConfigCombinesLevelAndHeroAttributes(t *testing.T) {
	setTestGeneralsConfig(t, GeneralsConfig{
		Enabled: true,
		Common: GeneralsCommonConfig{
			ExpCurve: []int{0, 100, 300},
			LevelBuffs: map[int]map[string]float64{
				1: {},
				2: {"productionBonus": 0.02},
			},
		},
		Heroes: map[string]GeneralHeroConfig{
			"test_general": {
				ID:      "test_general",
				Name:    "测试将领",
				Faction: "wei",
				Enabled: true,
				Buffs:   map[string]float64{"productionBonus": 0.1, "attackBonus": 0.05},
			},
		},
	})

	general := &General{ID: "test_general", Name: "测试将领", Level: 2, Exp: 120}
	applyHeroConfigToGeneral(general)

	expectedLevelAttack := 2.0 / 99.0
	if math.Abs(general.Attributes["productionBonus"]-0.1) > 1e-9 {
		t.Fatalf("expected production bonus to combine level and hero attributes, got %.2f", general.Attributes["productionBonus"])
	}
	if math.Abs(general.Buffs["attackBonus"]-(0.05+expectedLevelAttack)) > 1e-9 {
		t.Fatalf("expected attack bonus to sync into buffs, got %.2f", general.Buffs["attackBonus"])
	}
	if general.NextLevelExp != generalExpRequiredForLevelForTest(3) {
		t.Fatalf("expected next level exp %d, got %d", generalExpRequiredForLevelForTest(3), general.NextLevelExp)
	}
}

func TestValidateGeneralsConfigRejectsInvalidTraitParams(t *testing.T) {
	err := ValidateGeneralsConfig(GeneralsConfig{
		Enabled: true,
		Heroes: map[string]GeneralHeroConfig{
			"zhangliao": {
				ID:      "zhangliao",
				Name:    "张辽",
				Enabled: true,
				Traits: []GeneralTraitConfig{
					{
						TraitID: "weizhenxiaoyao",
						Enabled: true,
						Params: map[string]float64{
							"baseChance":       0.08,
							"maxChance":        0.35,
							"baseSuppressRate": 0.08,
							"maxSuppressRate":  1.5,
						},
					},
				},
			},
		},
	})
	if err == nil {
		t.Fatal("expected invalid maxSuppressRate to be rejected")
	}
}

func TestValidateGeneralsConfigRejectsUnknownTraitParam(t *testing.T) {
	err := ValidateGeneralsConfig(GeneralsConfig{
		Enabled: true,
		Heroes: map[string]GeneralHeroConfig{
			"zhangliao": {
				ID:      "zhangliao",
				Name:    "张辽",
				Enabled: true,
				Traits: []GeneralTraitConfig{
					{
						TraitID: "weizhenxiaoyao",
						Enabled: true,
						Params: map[string]float64{
							"baseChance": 0.08,
							"badParam":   1,
						},
					},
				},
			},
		},
	})
	if err == nil {
		t.Fatal("expected unknown trait param to be rejected")
	}
}

func TestApplyGeneralBattleExpPromotesLevel(t *testing.T) {
	setTestGeneralsConfig(t, GeneralsConfig{
		Enabled: true,
		Common: GeneralsCommonConfig{
			ExpCurve: []int{0, 10, 30},
			LevelBuffs: map[int]map[string]float64{
				1: {},
				2: {"attackBonus": 0.02},
			},
		},
		Heroes: map[string]GeneralHeroConfig{
			"test_general": {ID: "test_general", Name: "测试将领", Faction: "wei", Enabled: true},
		},
	})

	general := &General{ID: "test_general", Name: "测试将领", Level: 1, Exp: 9}
	result := applyGeneralBattleExp(general, generalExpRequiredForLevelForTest(2))

	if result.Gained != generalExpRequiredForLevelForTest(2) {
		t.Fatalf("expected battle exp %d, got %d", generalExpRequiredForLevelForTest(2), result.Gained)
	}
	if result.LevelBefore != 1 || result.LevelAfter != 2 || general.Level != 2 {
		t.Fatalf("expected general to level from 1 to 2, result=%+v general=%+v", result, general)
	}
	if math.Abs(general.Attributes["attackBonus"]-(2.0/99.0)) > 1e-9 {
		t.Fatalf("expected level 2 attack bonus to apply, got %.2f", general.Attributes["attackBonus"])
	}
}

func TestGeneralExpRequiredUsesConfiguredCurve(t *testing.T) {
	setTestGeneralsConfig(t, GeneralsConfig{
		Enabled: true,
		Common: GeneralsCommonConfig{
			ExpCurve:   []int{0, 10, 30, 80},
			LevelBuffs: map[int]map[string]float64{1: {}},
		},
		Heroes: map[string]GeneralHeroConfig{
			"test_general": {ID: "test_general", Name: "测试将领", Faction: "wei", Enabled: true},
		},
	})

	if generalExpRequiredForLevelForTest(4) != 80 {
		t.Fatalf("expected configured level 4 exp 80, got %d", generalExpRequiredForLevelForTest(4))
	}

	general := &General{ID: "test_general", Name: "测试将领", Level: 1, Exp: 29}
	result := applyGeneralBattleExp(general, 1)
	if result.LevelAfter != 3 || general.Level != 3 {
		t.Fatalf("expected configured curve to promote general to level 3, result=%+v general=%+v", result, general)
	}
}

func TestValidateGeneralsConfigRejectsInvalidExpCurve(t *testing.T) {
	err := ValidateGeneralsConfig(GeneralsConfig{
		Enabled: true,
		Common: GeneralsCommonConfig{
			ExpCurve: []int{0, 100, 90},
		},
		Heroes: map[string]GeneralHeroConfig{},
	})
	if err == nil {
		t.Fatal("expected non-increasing exp curve to be rejected")
	}
}

func TestGeneralExpFormulaPreventsLevel90SingleHugeBattleLevelUp(t *testing.T) {
	level90Exp := generalExpRequiredForLevelForTest(90)
	level91Exp := generalExpRequiredForLevelForTest(91)
	if level91Exp-level90Exp <= 1_200_000_000 {
		t.Fatalf("expected Lv90->Lv91 to require more than 1.2B exp, got %d", level91Exp-level90Exp)
	}

	general := &General{ID: "test_general", Name: "测试将领", Level: 90, Exp: level90Exp}
	result := applyGeneralBattleExp(general, 1_200_000_000)
	if result.LevelAfter != 90 || general.Level != 90 {
		t.Fatalf("expected 1.2B exp not to level Lv90 general, result=%+v general=%+v", result, general)
	}
}

func TestGeneralBattleExpUsesKilledUnitUpkeep(t *testing.T) {
	original := GetGeneralsConfig()
	originalUnits := GetUnitsConfig()
	t.Cleanup(func() {
		generalsMu.Lock()
		activeGenerals = original
		generalsMu.Unlock()
		unitsMu.Lock()
		activeUnits = originalUnits
		unitsMu.Unlock()
	})

	unitsMu.Lock()
	activeUnits = UnitsConfig{
		"shu": FactionUnits{
			"southernElephant": UnitConfig{Stats: map[string]int{"upkeep": 4}},
			"hanRoyalty":       UnitConfig{Stats: map[string]int{"upkeep": 6}},
		},
	}
	unitsMu.Unlock()

	exp := calculateGeneralBattleExpFromLosses("shu", []combat.UnitLoss{
		{ID: "southernElephant", Losses: 10},
		{ID: "hanRoyalty", Losses: 3},
	})
	if exp != 58 {
		t.Fatalf("expected exp by killed upkeep to be 58, got %d", exp)
	}
}

func TestValidateAndConsumeArmyRejectsTransportUnits(t *testing.T) {
	originalUnits := GetUnitsConfig()
	t.Cleanup(func() {
		unitsMu.Lock()
		activeUnits = originalUnits
		unitsMu.Unlock()
	})

	unitsMu.Lock()
	activeUnits = UnitsConfig{
		"wei": FactionUnits{
			"weiMerchant": UnitConfig{
				Role:  "transport",
				Stats: map[string]int{"attack": 0, "infantryDefense": 0, "cavalryDefense": 0, "carryCapacity": 1000, "upkeep": 0},
			},
		},
	}
	unitsMu.Unlock()

	state := newPlayerState("player_transport", "测试", "wei", "caocao", time.Now())
	state.Army = []ArmyUnit{{UnitType: "weiMerchant", Amount: 10}}
	_, err := validateAndConsumeArmy(&state, map[string]int{"weiMerchant": 1})
	if !errors.Is(err, ErrNonCombatUnit) {
		t.Fatalf("expected ErrNonCombatUnit, got %v", err)
	}
	if state.Army[0].Amount != 10 {
		t.Fatalf("expected transport unit amount unchanged, got %d", state.Army[0].Amount)
	}
}

func TestAllocateGeneralStatUpdatesAttributes(t *testing.T) {
	setTestGeneralsConfig(t, GeneralsConfig{
		Enabled: true,
		Heroes: map[string]GeneralHeroConfig{
			"test_general": {ID: "test_general", Name: "测试将领", Faction: "wei", Enabled: true},
		},
	})

	repo := NewMemoryRepository()
	service := NewServiceWithRepository(repo)
	now := time.Date(2026, 6, 2, 10, 0, 0, 0, time.UTC)
	if err := repo.CreateAccount(Account{ID: "account_stat", Username: "stat_user", PasswordHash: "hash", CreatedAt: now}); err != nil {
		t.Fatalf("create account: %v", err)
	}
	state := newPlayerState("player_stat", "测试", "wei", "test_general", now)
	state.General.Level = 2
	state.General.Exp = generalExpRequiredForLevelForTest(2)
	applyHeroConfigToGeneral(state.General)
	if err := repo.CreatePlayer("account_stat", state, now); err != nil {
		t.Fatalf("create player: %v", err)
	}
	attackBefore := state.General.Attributes[StatAttackBonus]

	next, err := service.AllocateGeneralStat("player_stat", "force")
	if err != nil {
		t.Fatalf("allocate general stat: %v", err)
	}
	if next.General.Stats["force"] != 1 {
		t.Fatalf("expected force to become 1, got %d", next.General.Stats["force"])
	}
	if next.General.AvailableStatPoints != 1 {
		t.Fatalf("expected 1 stat point remaining, got %d", next.General.AvailableStatPoints)
	}
	if next.General.Attributes[StatAttackBonus] <= attackBefore {
		t.Fatalf("expected attack bonus to increase, before %.4f after %.4f", attackBefore, next.General.Attributes[StatAttackBonus])
	}
	attackBreakdown := next.General.AttributeBreakdown[StatAttackBonus]
	if len(attackBreakdown) != 2 {
		t.Fatalf("expected attack bonus breakdown from level and force, got %+v", attackBreakdown)
	}
	if attackBreakdown[0].Source != "等级成长" || attackBreakdown[1].Source != "武力" {
		t.Fatalf("unexpected attack bonus breakdown sources: %+v", attackBreakdown)
	}
}

func TestAllocateGeneralStatRejectsMaxedStat(t *testing.T) {
	repo := NewMemoryRepository()
	service := NewServiceWithRepository(repo)
	now := time.Date(2026, 6, 2, 10, 0, 0, 0, time.UTC)
	if err := repo.CreateAccount(Account{ID: "account_stat_max", Username: "stat_max_user", PasswordHash: "hash", CreatedAt: now}); err != nil {
		t.Fatalf("create account: %v", err)
	}
	state := newPlayerState("player_stat_max", "测试", "wei", "caocao", now)
	state.General.Level = 100
	state.General.Stats = map[string]int{"force": 100}
	applyHeroConfigToGeneral(state.General)
	if err := repo.CreatePlayer("account_stat_max", state, now); err != nil {
		t.Fatalf("create player: %v", err)
	}

	if _, err := service.AllocateGeneralStat("player_stat_max", "force"); !errors.Is(err, ErrStatMaxLevel) {
		t.Fatalf("expected ErrStatMaxLevel, got %v", err)
	}
}

func TestServiceUpdateBalancePersistsConfig(t *testing.T) {
	original := GetBalanceConfig()
	t.Cleanup(func() {
		if err := SetBalanceConfig(original); err != nil {
			t.Fatalf("restore balance config: %v", err)
		}
	})

	service := NewService()
	path := filepath.Join(t.TempDir(), "balance.json")
	if err := service.SetBalancePath(path); err != nil {
		t.Fatalf("set balance path: %v", err)
	}

	next := service.GetBalance()
	next.BaseProduction["wood"] = 99
	if err := service.UpdateBalance(next); err != nil {
		t.Fatalf("update balance: %v", err)
	}

	var saved BalanceConfig
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read balance file: %v", err)
	}
	if err := json.Unmarshal(content, &saved); err != nil {
		t.Fatalf("unmarshal balance: %v", err)
	}
	if saved.BaseProduction["wood"] != 99 {
		t.Fatalf("expected updated wood base production, got %d", saved.BaseProduction["wood"])
	}
}

func TestSettleResourcesTimeSlicingOnUpgradeCompletion(t *testing.T) {
	// 场景：10:00 结算，wood_camp-1 Lv.1 正在升级（10:01 完成），12:00 上线
	// 期望：wood = base + Lv.1产量×1分钟 + Lv.2产量×119分钟
	// 而不是全段按 Lv.2 产量算 120 分钟

	settledAt := time.Date(2026, 5, 22, 10, 0, 0, 0, time.UTC)
	upgradeEndsAt := settledAt.Add(1 * time.Minute).Format(time.RFC3339) // 10:01 完成
	onlineAt := settledAt.Add(2 * time.Hour)                             // 12:00 上线

	state := GameState{
		Player: Player{ID: "test", Nickname: "测试", Faction: "wei"},
		Resources: ResourceState{
			Items: map[string]int{
				"wood":  0,
				"stone": 0,
				"iron":  0,
				"food":  0,
			},
			Capacity: map[string]int{
				"wood":  999999,
				"stone": 999999,
				"iron":  999999,
				"food":  999999,
			},
		},
		Buildings: []Building{
			{ID: "wood_camp-1", Type: "wood_camp", Level: 1, UpgradeEndsAt: &upgradeEndsAt},
		},
		ResourceProduction: ResourceProduction{"wood": 10}, // 旧产量（Lv.1）
		ResourceSettledAt:  settledAt.Format(time.RFC3339),
	}

	result, changed := settleResources(state, onlineAt)
	if !changed {
		t.Fatal("expected state to change")
	}

	// 升级应该完成
	if result.Buildings[0].Level != 2 {
		t.Fatalf("expected building to be Lv.2, got Lv.%d", result.Buildings[0].Level)
	}
	if result.Buildings[0].UpgradeEndsAt != nil {
		t.Fatal("expected upgradeEndsAt to be cleared")
	}

	// 计算期望产量：
	// Lv.1 产量 = productionByLevel[1] = 10（wood_camp Lv.1）
	// Lv.2 产量 = productionByLevel[2] = 18（wood_camp Lv.2）
	// 10:00-10:01 (60秒): wood += 10 * 60 / 3600 = 0（不足1单位）
	// 10:01-12:00 (7140秒): wood += 18 * 7140 / 3600 = 35
	// 如果错误地全段按 Lv.2 算：18 * 7200 / 3600 = 36
	lv1Production := getProductionAtLevel("wood_camp", 1)
	lv2Production := getProductionAtLevel("wood_camp", 2)

	slice1Seconds := 60.0   // 10:00 - 10:01
	slice2Seconds := 7140.0 // 10:01 - 12:00

	expectedWood := int(float64(lv1Production)*slice1Seconds/3600) + int(float64(lv2Production)*slice2Seconds/3600)

	// 错误计算（全段按新产量）
	wrongWood := int(float64(lv2Production) * 7200.0 / 3600)

	if expectedWood == wrongWood {
		t.Fatalf("test setup error: expected and wrong values should differ (expected=%d, wrong=%d)", expectedWood, wrongWood)
	}

	if result.Resources.Items["wood"] != expectedWood {
		t.Fatalf("time slicing error: expected wood=%d, got %d (wrong full-period calc would give %d)",
			expectedWood, result.Resources.Items["wood"], wrongWood)
	}
}

func TestSettleResourcesMultipleUpgradesTimeSlicing(t *testing.T) {
	// 场景：两个建筑在离线期间先后完成升级
	// wood_camp-1: 10:01 完成, wood_camp-2: 10:30 完成
	// 验证三段切片都正确

	settledAt := time.Date(2026, 5, 22, 10, 0, 0, 0, time.UTC)
	upgrade1EndsAt := settledAt.Add(1 * time.Minute).Format(time.RFC3339)
	upgrade2EndsAt := settledAt.Add(30 * time.Minute).Format(time.RFC3339)
	onlineAt := settledAt.Add(1 * time.Hour)

	state := GameState{
		Player: Player{ID: "test", Nickname: "测试", Faction: "wei"},
		Resources: ResourceState{
			Items: map[string]int{
				"wood":  0,
				"stone": 0,
				"iron":  0,
				"food":  0,
			},
			Capacity: map[string]int{
				"wood":  999999,
				"stone": 999999,
				"iron":  999999,
				"food":  999999,
			},
		},
		Buildings: []Building{
			{ID: "wood_camp-1", Type: "wood_camp", Level: 1, UpgradeEndsAt: &upgrade1EndsAt},
			{ID: "wood_camp-2", Type: "wood_camp", Level: 1, UpgradeEndsAt: &upgrade2EndsAt},
		},
		ResourceProduction: ResourceProduction{"wood": 20}, // 2 × Lv.1
		ResourceSettledAt:  settledAt.Format(time.RFC3339),
	}

	result, changed := settleResources(state, onlineAt)
	if !changed {
		t.Fatal("expected state to change")
	}

	// 两个建筑都应该升到 Lv.2
	if result.Buildings[0].Level != 2 || result.Buildings[1].Level != 2 {
		t.Fatalf("expected both buildings at Lv.2, got Lv.%d and Lv.%d",
			result.Buildings[0].Level, result.Buildings[1].Level)
	}

	// 三段切片：
	// 10:00-10:01 (60s): 2×Lv.1 产量
	// 10:01-10:30 (1740s): 1×Lv.2 + 1×Lv.1 产量
	// 10:30-11:00 (1800s): 2×Lv.2 产量
	lv1 := getProductionAtLevel("wood_camp", 1)
	lv2 := getProductionAtLevel("wood_camp", 2)

	slice1 := int(float64(2*lv1) * 60.0 / 3600)
	slice2 := int(float64(lv2+lv1) * 1740.0 / 3600)
	slice3 := int(float64(2*lv2) * 1800.0 / 3600)
	expectedWood := slice1 + slice2 + slice3

	if result.Resources.Items["wood"] != expectedWood {
		t.Fatalf("multi-upgrade time slicing error: expected wood=%d, got %d (slices: %d+%d+%d)",
			expectedWood, result.Resources.Items["wood"], slice1, slice2, slice3)
	}
}

func getProductionAtLevel(buildingType string, level int) int {
	config, exists := getBuildingConfig(buildingType)
	if !exists || len(config.ProductionByLevel) == 0 {
		return 0
	}
	if level < 0 {
		return 0
	}
	if level >= len(config.ProductionByLevel) {
		return config.ProductionByLevel[len(config.ProductionByLevel)-1]
	}
	return config.ProductionByLevel[level]
}

func TestInstantCompleteRecruitChargesRemainingTime(t *testing.T) {
	svc := NewService()
	repo := svc.repo.(*MemoryRepository)

	now := time.Now()
	account := Account{ID: "acc_recruit_speedup", Username: "speedup", PasswordHash: "x", CreatedAt: now}
	if err := repo.CreateAccount(account); err != nil {
		t.Fatalf("create account: %v", err)
	}

	state := newPlayerState("player_recruit_speedup", "Speedup", "wei", "caocao", now)
	state.CityGold = 100
	state.RecruitQueues = []RecruitQueue{
		{
			ID:       "rq_remaining",
			UnitType: "weiInfantry",
			Amount:   10,
			EndsAt:   now.Add(240 * time.Second).UTC().Format(resourceDateLayout),
		},
	}
	if err := repo.CreatePlayer(account.ID, state, now); err != nil {
		t.Fatalf("create player: %v", err)
	}

	next, err := svc.InstantCompleteRecruit(state.Player.ID, "rq_remaining")
	if err != nil {
		t.Fatalf("InstantCompleteRecruit failed: %v", err)
	}

	if next.CityGold != 98 {
		t.Fatalf("expected remaining-time cost 2 city gold, got balance %d", next.CityGold)
	}
	if len(next.RecruitQueues) != 0 {
		t.Fatalf("expected queue to complete, got %d queues", len(next.RecruitQueues))
	}
}

func TestInstantCompleteBuildingReturnsFreshModifiers(t *testing.T) {
	svc := NewService()
	repo := svc.repo.(*MemoryRepository)

	now := time.Now()
	account := Account{ID: "acc_building_modifiers", Username: "building_modifiers", PasswordHash: "x", CreatedAt: now}
	if err := repo.CreateAccount(account); err != nil {
		t.Fatalf("create account: %v", err)
	}

	state := newPlayerState("player_building_modifiers", "Builder", "wei", "caocao", now)
	state.CityGold = 100
	for i := range state.Buildings {
		if state.Buildings[i].Type == "weapon_bureau" {
			endsAt := now.Add(60 * time.Second).UTC().Format(resourceDateLayout)
			state.Buildings[i].UpgradeEndsAt = &endsAt
			break
		}
	}
	if err := repo.CreatePlayer(account.ID, state, now); err != nil {
		t.Fatalf("create player: %v", err)
	}

	next, err := svc.InstantCompleteBuilding(state.Player.ID, "weapon_bureau-1")
	if err != nil {
		t.Fatalf("InstantCompleteBuilding failed: %v", err)
	}

	var attackBonus float64
	for _, item := range next.ActiveModifiers {
		if item.Key == StatAttackBonus && item.Source == "军事建筑" {
			attackBonus = item.Value
			break
		}
	}
	if math.Abs(attackBonus-0.02) > 1e-9 {
		t.Fatalf("expected fresh weapon bureau attack bonus 0.02, got %.4f", attackBonus)
	}
}

func TestSimulateBattleDoesNotConsumeArmy(t *testing.T) {
	setTestCombatUnitsConfig(t)

	svc := NewService()
	repo := svc.repo.(*MemoryRepository)
	now := time.Now()
	account := Account{ID: "acc_sim_no_consume", Username: "sim_no_consume", PasswordHash: "x", CreatedAt: now}
	if err := repo.CreateAccount(account); err != nil {
		t.Fatalf("create account: %v", err)
	}

	state := newPlayerState("player_sim_no_consume", "Simulator", "wei", "caocao", now)
	state.Army = []ArmyUnit{{UnitType: "weiInfantry", Amount: 100}}
	if err := repo.CreatePlayer(account.ID, state, now); err != nil {
		t.Fatalf("create player: %v", err)
	}

	_, err := svc.SimulateBattle(BattleSimulationRequest{
		PlayerID:        state.Player.ID,
		Mode:            "attack",
		AttackerFaction: "wei",
		DefenderFaction: "wei",
		AttackerUnits:   map[string]int{"weiInfantry": 80},
		DefenderUnits:   map[string]int{"weiInfantry": 80},
	})
	if err != nil {
		t.Fatalf("SimulateBattle failed: %v", err)
	}

	stored, err := repo.GetState(state.Player.ID)
	if err != nil {
		t.Fatalf("get state: %v", err)
	}
	if len(stored.Army) != 1 || stored.Army[0].Amount != 100 {
		t.Fatalf("expected simulated battle not to consume army, got %+v", stored.Army)
	}
}

func TestSimulateBattleAppliesCurrentPlayerBonuses(t *testing.T) {
	setTestCombatUnitsConfig(t)
	setTestGeneralsConfig(t, GeneralsConfig{
		Enabled: true,
		Heroes: map[string]GeneralHeroConfig{
			"test_general": {
				ID:      "test_general",
				Name:    "测试将领",
				Faction: "wei",
				Enabled: true,
				Buffs:   map[string]float64{StatAttackBonus: 0.5},
			},
		},
	})

	svc := NewService()
	repo := svc.repo.(*MemoryRepository)
	now := time.Now()
	account := Account{ID: "acc_sim_bonus", Username: "sim_bonus", PasswordHash: "x", CreatedAt: now}
	if err := repo.CreateAccount(account); err != nil {
		t.Fatalf("create account: %v", err)
	}

	state := newPlayerState("player_sim_bonus", "Simulator", "wei", "test_general", now)
	if err := repo.CreatePlayer(account.ID, state, now); err != nil {
		t.Fatalf("create player: %v", err)
	}

	base, err := svc.SimulateBattle(BattleSimulationRequest{
		PlayerID:        state.Player.ID,
		Mode:            "attack",
		AttackerFaction: "wei",
		DefenderFaction: "wei",
		AttackerUnits:   map[string]int{"weiInfantry": 100},
		DefenderUnits:   map[string]int{"weiInfantry": 100},
	})
	if err != nil {
		t.Fatalf("base SimulateBattle failed: %v", err)
	}
	boosted, err := svc.SimulateBattle(BattleSimulationRequest{
		PlayerID:             state.Player.ID,
		Mode:                 "attack",
		AttackerFaction:      "wei",
		DefenderFaction:      "wei",
		AttackerUnits:        map[string]int{"weiInfantry": 100},
		DefenderUnits:        map[string]int{"weiInfantry": 100},
		ApplyAttackerBonuses: true,
	})
	if err != nil {
		t.Fatalf("boosted SimulateBattle failed: %v", err)
	}

	if boosted.Result.AttackPower <= base.Result.AttackPower {
		t.Fatalf("expected attacker bonus to increase attack power, base %.2f boosted %.2f", base.Result.AttackPower, boosted.Result.AttackPower)
	}
	if boosted.Attacker.Units[0].Attack <= base.Attacker.Units[0].Attack {
		t.Fatalf("expected attacker unit attack to increase, base %+v boosted %+v", base.Attacker.Units[0], boosted.Attacker.Units[0])
	}
}

func TestAddAccountGoldAdminWritesLedger(t *testing.T) {
	svc := NewService()
	repo := svc.repo.(*MemoryRepository)
	now := time.Now()
	account := Account{ID: "acc_gold_ledger", Username: "gold_ledger", PasswordHash: "x", CreatedAt: now}
	if err := repo.CreateAccount(account); err != nil {
		t.Fatalf("create account: %v", err)
	}

	if err := svc.AddAccountGoldAdmin(account.ID, 25); err != nil {
		t.Fatalf("AddAccountGoldAdmin failed: %v", err)
	}

	entries, err := svc.ListGoldLedger(GoldLedgerFilter{AccountID: account.ID})
	if err != nil {
		t.Fatalf("ListGoldLedger failed: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 ledger entry, got %d", len(entries))
	}
	entry := entries[0]
	if entry.Currency != LedgerCurrencyGold || entry.Direction != LedgerDirectionCredit || entry.Amount != 25 || entry.BalanceAfter != 25 {
		t.Fatalf("unexpected ledger entry: %+v", entry)
	}
	if entry.RefType != LedgerRefAdminAdjust {
		t.Fatalf("expected admin adjust ref type, got %q", entry.RefType)
	}
}

func TestExchangeGoldToCityGoldWritesLinkedLedgerEntries(t *testing.T) {
	original := GetBalanceConfig()
	t.Cleanup(func() {
		if err := SetBalanceConfig(original); err != nil {
			t.Fatalf("restore balance config: %v", err)
		}
	})
	balance := GetBalanceConfig()
	balance.ExchangeRate = 10
	balance.ExchangeCooldownSecs = 0
	if err := SetBalanceConfig(balance); err != nil {
		t.Fatalf("set balance config: %v", err)
	}

	svc := NewService()
	repo := svc.repo.(*MemoryRepository)
	now := time.Now()
	account := Account{ID: "acc_exchange_ledger", Username: "exchange_ledger", PasswordHash: "x", Gold: 50, CreatedAt: now}
	if err := repo.CreateAccount(account); err != nil {
		t.Fatalf("create account: %v", err)
	}
	state := newPlayerState("player_exchange_ledger", "Exchange", "wei", "caocao", now)
	state.CityGold = 5
	if err := repo.CreatePlayer(account.ID, state, now); err != nil {
		t.Fatalf("create player: %v", err)
	}

	if _, err := svc.ExchangeGoldToCityGold(account.ID, state.Player.ID, 3); err != nil {
		t.Fatalf("ExchangeGoldToCityGold failed: %v", err)
	}

	entries, err := svc.ListGoldLedger(GoldLedgerFilter{AccountID: account.ID, RefType: LedgerRefExchange})
	if err != nil {
		t.Fatalf("ListGoldLedger failed: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 ledger entries, got %d: %+v", len(entries), entries)
	}
	if entries[0].RefID == "" || entries[0].RefID != entries[1].RefID {
		t.Fatalf("expected exchange ledger entries to share refId, got %+v", entries)
	}

	var goldDebit, cityGoldCredit *GoldLedgerEntry
	for i := range entries {
		entry := &entries[i]
		if entry.Currency == LedgerCurrencyGold && entry.Direction == LedgerDirectionDebit {
			goldDebit = entry
		}
		if entry.Currency == LedgerCurrencyCityGold && entry.Direction == LedgerDirectionCredit {
			cityGoldCredit = entry
		}
	}
	if goldDebit == nil || goldDebit.Amount != 3 || goldDebit.BalanceAfter != 47 {
		t.Fatalf("unexpected gold debit entry: %+v", goldDebit)
	}
	if cityGoldCredit == nil || cityGoldCredit.Amount != 30 || cityGoldCredit.BalanceAfter != 35 {
		t.Fatalf("unexpected city gold credit entry: %+v", cityGoldCredit)
	}
}

func TestExchangeCityGoldToGoldWritesLinkedLedgerEntries(t *testing.T) {
	original := GetBalanceConfig()
	t.Cleanup(func() {
		if err := SetBalanceConfig(original); err != nil {
			t.Fatalf("restore balance config: %v", err)
		}
	})
	balance := GetBalanceConfig()
	balance.ReverseExchangeRate = 15
	balance.ExchangeCooldownSecs = 0
	if err := SetBalanceConfig(balance); err != nil {
		t.Fatalf("set balance config: %v", err)
	}

	svc := NewService()
	repo := svc.repo.(*MemoryRepository)
	now := time.Now()
	account := Account{ID: "acc_reverse_ledger", Username: "reverse_ledger", PasswordHash: "x", Gold: 2, CreatedAt: now}
	if err := repo.CreateAccount(account); err != nil {
		t.Fatalf("create account: %v", err)
	}
	state := newPlayerState("player_reverse_ledger", "Reverse", "wei", "caocao", now)
	state.CityGold = 45
	if err := repo.CreatePlayer(account.ID, state, now); err != nil {
		t.Fatalf("create player: %v", err)
	}

	if _, err := svc.ExchangeCityGoldToGold(account.ID, state.Player.ID, 30); err != nil {
		t.Fatalf("ExchangeCityGoldToGold failed: %v", err)
	}

	entries, err := svc.ListGoldLedger(GoldLedgerFilter{AccountID: account.ID, RefType: LedgerRefExchange})
	if err != nil {
		t.Fatalf("ListGoldLedger failed: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 ledger entries, got %d: %+v", len(entries), entries)
	}
	if entries[0].RefID == "" || entries[0].RefID != entries[1].RefID {
		t.Fatalf("expected reverse exchange ledger entries to share refId, got %+v", entries)
	}

	var cityGoldDebit, goldCredit *GoldLedgerEntry
	for i := range entries {
		entry := &entries[i]
		if entry.Currency == LedgerCurrencyCityGold && entry.Direction == LedgerDirectionDebit {
			cityGoldDebit = entry
		}
		if entry.Currency == LedgerCurrencyGold && entry.Direction == LedgerDirectionCredit {
			goldCredit = entry
		}
	}
	if cityGoldDebit == nil || cityGoldDebit.Amount != 30 || cityGoldDebit.BalanceAfter != 15 {
		t.Fatalf("unexpected city gold debit entry: %+v", cityGoldDebit)
	}
	if goldCredit == nil || goldCredit.Amount != 2 || goldCredit.BalanceAfter != 4 {
		t.Fatalf("unexpected gold credit entry: %+v", goldCredit)
	}
}

func TestOwnsPlayer(t *testing.T) {
	svc := NewService()
	repo := svc.repo.(*MemoryRepository)

	// 直接创建账号和玩家（绕过 faction 校验）
	now := time.Now()
	aliceAccount := Account{ID: "acc_alice", Username: "alice", PasswordHash: "x", CreatedAt: now}
	bobAccount := Account{ID: "acc_bob", Username: "bob", PasswordHash: "x", CreatedAt: now}
	if err := repo.CreateAccount(aliceAccount); err != nil {
		t.Fatalf("create alice: %v", err)
	}
	if err := repo.CreateAccount(bobAccount); err != nil {
		t.Fatalf("create bob: %v", err)
	}

	aliceState := newPlayerState("player_alice_1", "Alice", "wei", "caocao", now)
	if err := repo.CreatePlayer(aliceAccount.ID, aliceState, now); err != nil {
		t.Fatalf("create alice player: %v", err)
	}

	tests := []struct {
		name      string
		accountID string
		playerID  string
		expected  bool
	}{
		{"owner matches", aliceAccount.ID, "player_alice_1", true},
		{"different account", bobAccount.ID, "player_alice_1", false},
		{"empty account", "", "player_alice_1", false},
		{"empty player", aliceAccount.ID, "", false},
		{"non-existent account", "acc_fake", "player_alice_1", false},
		{"non-existent player", aliceAccount.ID, "player_fake", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			owns, err := svc.OwnsPlayer(tt.accountID, tt.playerID)
			if err != nil {
				t.Fatalf("OwnsPlayer error: %v", err)
			}
			if owns != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, owns)
			}
		})
	}
}
