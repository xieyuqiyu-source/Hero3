// 本文件覆盖游戏应用服务的核心行为测试。
package game

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"hero3/internal/core/combat"
	coregeneral "hero3/internal/core/general"
	_ "hero3/internal/core/general/traits"
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

func TestGetMilitaryViewPersistsCaoCaoGuardProduction(t *testing.T) {
	setRealCaoCaoGuardConfig(t)
	now := time.Now().UTC()
	settledAt := now.Add(-24 * time.Hour)
	repo := NewMemoryRepository()
	svc := NewServiceWithRepository(repo)
	state := newPlayerState("player_guard_view", "主公", "wei", "caocao", settledAt)
	state.Army = []ArmyUnit{}
	state.ResourceSettledAt = settledAt.Format(resourceDateLayout)
	state.ServerTime = settledAt.Format(resourceDateLayout)
	repo.players[state.Player.ID] = state

	view, err := svc.GetMilitaryView(state.Player.ID)
	if err != nil {
		t.Fatalf("GetMilitaryView failed: %v", err)
	}
	if got := armySliceToMap(view.Army)["huWei"]; got != 3000 {
		t.Fatalf("expected military view huWei 3000, got %d army=%+v", got, view.Army)
	}
	stored, err := repo.GetState(state.Player.ID)
	if err != nil {
		t.Fatalf("GetState failed: %v", err)
	}
	if got := armySliceToMap(stored.Army)["huWei"]; got != 3000 {
		t.Fatalf("expected persisted huWei 3000, got %d army=%+v", got, stored.Army)
	}
	repeated, err := svc.GetMilitaryView(state.Player.ID)
	if err != nil {
		t.Fatalf("repeated GetMilitaryView failed: %v", err)
	}
	repeatedStored, err := repo.GetState(state.Player.ID)
	if err != nil {
		t.Fatalf("GetState after repeated view failed: %v", err)
	}
	firstSettledAt, firstErr := time.Parse(resourceDateLayout, stored.ResourceSettledAt)
	repeatedSettledAt, repeatedErr := time.Parse(resourceDateLayout, repeatedStored.ResourceSettledAt)
	if firstErr != nil || repeatedErr != nil {
		t.Fatalf("parse settlement timestamps failed: first=%v repeated=%v", firstErr, repeatedErr)
	}
	legitimateElapsedSeconds := int(repeatedSettledAt.Sub(firstSettledAt).Seconds())
	wantRepeated := 3000 + int(500*float64(legitimateElapsedSeconds)/60)
	if got := armySliceToMap(repeated.Army)["huWei"]; got != wantRepeated {
		t.Fatalf("expected repeated view to add only %d whole-second guards, got %d army=%+v", wantRepeated-3000, got, repeated.Army)
	}
	reports, total, err := repo.ListReports(state.Player.ID, 10, 0)
	if err != nil || total != 0 || len(reports) != 0 {
		t.Fatalf("expected guard production not to create battle report, reports=%+v total=%d err=%v", reports, total, err)
	}
}

// TestSettleResourcesDoesNotReplayFractionalSecondGuardProduction 验证秒级时间戳不会在同一秒重复结算小数秒产兵。
func TestSettleResourcesDoesNotReplayFractionalSecondGuardProduction(t *testing.T) {
	setRealCaoCaoGuardConfig(t)
	base := time.Date(2026, 7, 19, 8, 0, 0, 0, time.UTC)
	state := newPlayerState("player_guard_fraction", "主公", "wei", "caocao", base.Add(-24*time.Hour))
	state.Army = nil
	state.ResourceSettledAt = base.Add(-24 * time.Hour).Format(resourceDateLayout)

	first, _ := settleResources(state, base.Add(900*time.Millisecond))
	if got := armySliceToMap(first.Army)["huWei"]; got != 3000 {
		t.Fatalf("expected first settlement capped at 3000, got %d", got)
	}
	second, _ := settleResources(first, base.Add(950*time.Millisecond))
	if got := armySliceToMap(second.Army)["huWei"]; got != 3000 {
		t.Fatalf("expected same-second settlement not to replay fractional production, got %d", got)
	}
}

// TestWeiwuHaolingProductionIsIndependentFromSettlementFrequency 验证高频资源结算不会因逐次取整少发虎卫。
func TestWeiwuHaolingProductionIsIndependentFromSettlementFrequency(t *testing.T) {
	setRealCaoCaoGuardConfig(t)
	base := time.Date(2026, 7, 19, 8, 0, 0, 0, time.UTC)
	newState := func(id string) GameState {
		state := newPlayerState(id, "主公", "wei", "caocao", base)
		state.Army = nil
		state.ResourceSettledAt = base.Format(resourceDateLayout)
		return state
	}

	single, _ := settleResources(newState("player_guard_single"), base.Add(3*time.Second))
	frequent := newState("player_guard_frequent")
	for second := 1; second <= 3; second++ {
		frequent, _ = settleResources(frequent, base.Add(time.Duration(second)*time.Second))
	}

	singleGuards := armySliceToMap(single.Army)["huWei"]
	frequentGuards := armySliceToMap(frequent.Army)["huWei"]
	if singleGuards != 25 || frequentGuards != singleGuards {
		t.Fatalf("expected one 3-second settlement and three 1-second settlements both produce 25 guards, single=%d frequent=%d progress=%+v", singleGuards, frequentGuards, frequent.GeneralTraitProgress)
	}
	if len(frequent.GeneralTraitProgress) != 0 {
		t.Fatalf("expected exact three-second production to consume fractional progress, got %+v", frequent.GeneralTraitProgress)
	}
}

// TestWeiwuHaolingSettlementCapDiscardsOverflowProgress 验证单次达到上限后不会把被截断的小数延后补发。
func TestWeiwuHaolingSettlementCapDiscardsOverflowProgress(t *testing.T) {
	amount, remainder := calculateGuardProduction(0.5, 500, 360*time.Second.Seconds(), 3000)
	if amount != 3000 || remainder != 0 {
		t.Fatalf("expected capped settlement to produce 3000 and discard overflow progress, got amount=%d remainder=%f", amount, remainder)
	}
}

func setRealCaoCaoGuardConfig(t *testing.T) {
	t.Helper()
	originalUnits := GetUnitsConfig()
	originalFactions := GetFactionsConfig()
	originalGenerals := GetGeneralsConfig()
	if err := LoadFactionsConfig(filepath.Join("..", "..", "..", "config", "factions.json")); err != nil {
		t.Fatalf("LoadFactionsConfig failed: %v", err)
	}
	if err := LoadUnitsConfig(filepath.Join("..", "..", "..", "config", "units")); err != nil {
		t.Fatalf("LoadUnitsConfig failed: %v", err)
	}
	if err := LoadGeneralsConfig(filepath.Join("..", "..", "..", "config", "generals.json")); err != nil {
		t.Fatalf("LoadGeneralsConfig failed: %v", err)
	}
	t.Cleanup(func() {
		unitsMu.Lock()
		activeUnits = originalUnits
		unitsMu.Unlock()
		factionsMu.Lock()
		activeFactions = originalFactions
		factionsMu.Unlock()
		generalsMu.Lock()
		activeGenerals = originalGenerals
		generalsMu.Unlock()
	})
}

func TestSettleResourcesCapsAtCapacity(t *testing.T) {
	settledAt := time.Date(2026, 5, 22, 10, 0, 0, 0, time.UTC)
	state := newPlayerState("player_test", "主公", "wei", "caocao", settledAt)
	state.Resources = ResourceState{
		Items: map[string]int{
			"wood":  4700,
			"stone": 4700,
			"iron":  4700,
			"food":  4700,
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

func TestSettleResourcesPreservesOverflowResources(t *testing.T) {
	settledAt := time.Date(2026, 5, 22, 10, 0, 0, 0, time.UTC)
	state := newPlayerState("player_test", "主公", "wei", "caocao", settledAt)
	state.Resources = ResourceState{
		Items: map[string]int{
			"wood":  320000,
			"stone": 280000,
			"iron":  240000,
			"food":  160000,
		},
		Capacity: map[string]int{
			"wood":  3200,
			"stone": 3200,
			"iron":  3200,
			"food":  3200,
		},
	}
	state.ResourceSettledAt = settledAt.Format(time.RFC3339)

	next, _ := settleResources(state, settledAt.Add(time.Hour))
	if next.Resources.Items["wood"] != 320000 || next.Resources.Items["stone"] != 280000 {
		t.Fatalf("expected overflow resources to be preserved, got %+v", next.Resources.Items)
	}
	if next.Resources.Items["iron"] != 240000 || next.Resources.Items["food"] != 160000 {
		t.Fatalf("expected overflow resources to be preserved, got %+v", next.Resources.Items)
	}
}

func TestSettleResourcesAdvancesTimestampWhenCapacityIsFull(t *testing.T) {
	settledAt := time.Date(2026, 5, 22, 10, 0, 0, 0, time.UTC)
	state := newPlayerState("player_test", "主公", "wei", "caocao", settledAt)
	state.Resources = ResourceState{
		Items: map[string]int{
			"wood":  4800,
			"stone": 4800,
			"iron":  4800,
			"food":  4800,
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

	cfg = fillTestDualTraits(cfg)
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

func fillTestDualTraits(cfg GeneralsConfig) GeneralsConfig {
	cfg = NormalizeGeneralsConfig(cfg)
	for id, hero := range cfg.Heroes {
		if !hero.Enabled {
			cfg.Heroes[id] = hero
			continue
		}
		if strings.TrimSpace(hero.SpecialTrait.TraitID) == "" {
			hero.SpecialTrait = GeneralTraitConfig{
				TraitID:   "rende",
				TraitType: "special",
				Enabled:   true,
				Scope:     "self_army",
				Params:    map[string]float64{"effectRate": 0.1, "reviveRate": 0.1, "maxReviveCount": 1000, "triggerChance": 0},
			}
		}
		if strings.TrimSpace(hero.BonusTrait.TraitID) == "" {
			hero.BonusTrait = GeneralTraitConfig{
				TraitID:   "renzhu_shouhu",
				TraitType: "bonus",
				Enabled:   true,
				Scope:     "self_army",
				Params:    map[string]float64{"lossReductionRate": 0.1, "maxReturnCount": 1000, "triggerChance": 0},
			}
		}
		cfg.Heroes[id] = hero
	}
	return cfg
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
				SpecialTrait: GeneralTraitConfig{
					TraitID:   "meiren",
					TraitType: "special",
					Enabled:   true,
					Params:    map[string]float64{"captureRate": 2, "captureMax": 1000, "triggerChance": 1},
				},
				BonusTrait: GeneralTraitConfig{
					TraitID:   "renzhu_shouhu",
					TraitType: "bonus",
					Enabled:   true,
					Params:    map[string]float64{"lossReductionRate": 0.1, "maxReturnCount": 1000},
				},
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

func TestValidateGeneralsConfigRejectsMissingDualTraits(t *testing.T) {
	originalFactions := GetFactionsConfig()
	factionsMu.Lock()
	activeFactions = FactionsConfig{"wei": {Generals: []GeneralInfo{{ID: "caocao", Name: "曹操"}}}}
	factionsMu.Unlock()
	t.Cleanup(func() {
		factionsMu.Lock()
		activeFactions = originalFactions
		factionsMu.Unlock()
	})

	err := ValidateGeneralsConfig(GeneralsConfig{
		Enabled: true,
		Heroes: map[string]GeneralHeroConfig{
			"caocao": {ID: "caocao", Name: "曹操", Faction: "wei", Enabled: true},
		},
	})
	if err == nil {
		t.Fatalf("expected enabled general without specialTrait and bonusTrait to fail")
	}
}

// TestGeneralsConfigFileHasAllDualTraits 验证正式将领双特性、注册表和关键场景约束完整一致。
func TestGeneralsConfigFileHasAllDualTraits(t *testing.T) {
	original := GetGeneralsConfig()
	originalFactions := GetFactionsConfig()
	originalUnits := GetUnitsConfig()
	if err := LoadFactionsConfig("../../../config/factions.json"); err != nil {
		t.Fatalf("load factions config: %v", err)
	}
	if err := LoadUnitsConfig("../../../config/units"); err != nil {
		t.Fatalf("load units config: %v", err)
	}
	t.Cleanup(func() {
		generalsMu.Lock()
		activeGenerals = original
		generalsMu.Unlock()
		factionsMu.Lock()
		activeFactions = originalFactions
		factionsMu.Unlock()
		unitsMu.Lock()
		activeUnits = originalUnits
		unitsMu.Unlock()
	})
	if err := LoadGeneralsConfig("../../../config/generals.json"); err != nil {
		t.Fatalf("load generals config: %v", err)
	}
	cfg := GetGeneralsConfig()
	if len(cfg.Heroes) < 24 {
		t.Fatalf("expected at least 24 generals, got %d", len(cfg.Heroes))
	}
	officialTraitIDs := map[string]bool{}
	for id, hero := range cfg.Heroes {
		if strings.TrimSpace(hero.SpecialTrait.TraitID) == "" || strings.TrimSpace(hero.BonusTrait.TraitID) == "" {
			t.Fatalf("general %s missing dual traits: %+v", id, hero)
		}
		if hero.SpecialTrait.TraitType != "special" || hero.BonusTrait.TraitType != "bonus" {
			t.Fatalf("general %s has wrong trait types: special=%s bonus=%s", id, hero.SpecialTrait.TraitType, hero.BonusTrait.TraitType)
		}
		officialTraitIDs[hero.SpecialTrait.TraitID] = true
		officialTraitIDs[hero.BonusTrait.TraitID] = true
	}
	if len(officialTraitIDs) != 50 {
		t.Fatalf("expected exactly 50 official trait IDs, got %d: %+v", len(officialTraitIDs), officialTraitIDs)
	}
	for generalID, trait := range map[string]GeneralTraitConfig{
		"xiahouyuan": cfg.Heroes["xiahouyuan"].BonusTrait,
		"weiyan":     cfg.Heroes["weiyan"].BonusTrait,
	} {
		if strings.Join(trait.AllowedSides, ",") != "defender,reinforcement" {
			t.Fatalf("expected %s defense-only trait to allow defender and reinforcement, got %+v", generalID, trait.AllowedSides)
		}
	}
	for generalID, trait := range map[string]GeneralTraitConfig{
		"zhenmi":       cfg.Heroes["zhenmi"].BonusTrait,
		"xuchuSpecial": cfg.Heroes["xuchu"].SpecialTrait,
		"xuchuBonus":   cfg.Heroes["xuchu"].BonusTrait,
		"huangzhong":   cfg.Heroes["huangzhong"].SpecialTrait,
	} {
		if strings.Join(trait.AllowedSides, ",") != "attacker" {
			t.Fatalf("expected %s defense-reduction trait to allow attacker only, got %+v", generalID, trait.AllowedSides)
		}
	}
	if trait := cfg.Heroes["simayi"].BonusTrait; strings.Join(trait.AllowedSides, ",") != "defender" {
		t.Fatalf("expected Simayi attack-reduction trait to allow defender only, got %+v", trait.AllowedSides)
	}
	for generalID, trait := range map[string]GeneralTraitConfig{
		"dianwei":   cfg.Heroes["dianwei"].BonusTrait,
		"zhangliao": cfg.Heroes["zhangliao"].BonusTrait,
		"guanyu":    cfg.Heroes["guanyu"].BonusTrait,
		"zhangfei":  cfg.Heroes["zhangfei"].BonusTrait,
		"sunce":     cfg.Heroes["sunce"].BonusTrait,
		"zhouyu":    cfg.Heroes["zhouyu"].BonusTrait,
	} {
		if strings.Join(trait.AllowedSides, ",") != "attacker" {
			t.Fatalf("expected %s attack-only trait to allow attacker only, got %+v", generalID, trait.AllowedSides)
		}
	}
	registered := coregeneral.All()
	if len(registered) != len(officialTraitIDs) {
		t.Fatalf("expected registry to exactly match %d official traits, got %d", len(officialTraitIDs), len(registered))
	}
	for _, trait := range registered {
		if !officialTraitIDs[trait.ID()] {
			t.Fatalf("registered trait %s is not used by official general config", trait.ID())
		}
	}
}

// TestValidateGeneralsConfigRejectsUnknownTargetUnitType 验证错误兵种目标会在配置加载时直接失败。
func TestValidateGeneralsConfigRejectsUnknownTargetUnitType(t *testing.T) {
	setTestCombatUnitsConfig(t)
	originalFactions := GetFactionsConfig()
	factionsMu.Lock()
	activeFactions = FactionsConfig{"wei": {Generals: []GeneralInfo{{ID: "caocao", Name: "曹操"}}}}
	factionsMu.Unlock()
	t.Cleanup(func() {
		factionsMu.Lock()
		activeFactions = originalFactions
		factionsMu.Unlock()
	})

	cfg := fillTestDualTraits(GeneralsConfig{
		Enabled: true,
		Heroes: map[string]GeneralHeroConfig{
			"caocao": {
				ID: "caocao", Name: "曹操", Faction: "wei", Enabled: true,
				BonusTrait: GeneralTraitConfig{
					TraitID: "weiwu_tongyu", TraitType: "bonus", Enabled: true,
					Scope: "self_army", TargetUnitType: "missing_unit",
					Params: map[string]float64{"attackBonusRate": 0.1, "defenseBonusRate": 0.1},
				},
			},
		},
	})
	if err := ValidateGeneralsConfig(cfg); err == nil || !strings.Contains(err.Error(), "unknown unit or category") {
		t.Fatalf("expected unknown target unit validation error, got %v", err)
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

func TestCreatePlayerUsesDefaultGeneralWhenMissing(t *testing.T) {
	setTestFactionsAndGenerals(t, FactionsConfig{
		"wei": {Generals: []GeneralInfo{
			{ID: "zhenmi", Name: "甄宓"},
			{ID: "caocao", Name: "曹操"},
		}},
	}, GeneralsConfig{
		Enabled: true,
		Heroes: map[string]GeneralHeroConfig{
			"zhenmi": {ID: "zhenmi", Name: "甄宓", Faction: "wei", Enabled: false},
			"caocao": {ID: "caocao", Name: "曹操", Faction: "wei", Enabled: true},
		},
	})

	repo := NewMemoryRepository()
	service := NewServiceWithRepository(repo)
	now := time.Date(2026, 6, 26, 11, 30, 0, 0, time.UTC)
	account := Account{ID: "account_default_general", Username: "default_general", PasswordHash: "hash", CreatedAt: now}
	if err := repo.CreateAccount(account); err != nil {
		t.Fatalf("create account: %v", err)
	}

	_, state, err := service.CreatePlayer(account.ID, "默认将领", "wei", "")
	if err != nil {
		t.Fatalf("CreatePlayer without general failed: %v", err)
	}
	if state.General == nil || state.General.ID != "caocao" {
		t.Fatalf("expected caocao default general, got %+v", state.General)
	}
}

func TestCreatePlayerPublishesCreatedEvent(t *testing.T) {
	setTestFactionsAndGenerals(t, FactionsConfig{
		"wei": {Generals: []GeneralInfo{{ID: "caocao", Name: "曹操"}}},
	}, GeneralsConfig{
		Enabled: true,
		Heroes: map[string]GeneralHeroConfig{
			"caocao": {ID: "caocao", Name: "曹操", Faction: "wei", Enabled: true},
		},
	})
	service := NewService()
	repo := service.repo.(*MemoryRepository)
	now := time.Now()
	account := Account{ID: "account_created_event", Username: "created_event", PasswordHash: "x", CreatedAt: now}
	if err := repo.CreateAccount(account); err != nil {
		t.Fatalf("create account: %v", err)
	}

	events := []GameEvent{}
	service.SubscribeEvent(EventPlayerCreated, func(event GameEvent) {
		events = append(events, event)
	})

	playerID, _, err := service.CreatePlayer(account.ID, "创建事件", "wei", "caocao")
	if err != nil {
		t.Fatalf("CreatePlayer failed: %v", err)
	}
	if len(events) != 1 || events[0].Type != EventPlayerCreated || events[0].PlayerID != playerID {
		t.Fatalf("expected player created event, got %+v", events)
	}
}

func TestDeletePlayerSchedulesAndRestoresDeletion(t *testing.T) {
	service := NewService()
	repo := service.repo.(*MemoryRepository)
	now := time.Now()
	account := Account{ID: "acc_delete_delay", Username: "delete_delay", PasswordHash: "x", CreatedAt: now}
	if err := repo.CreateAccount(account); err != nil {
		t.Fatalf("create account: %v", err)
	}
	state := newPlayerState("player_delete_delay", "待删除", "wei", "caocao", now)
	if err := repo.CreatePlayer(account.ID, state, now); err != nil {
		t.Fatalf("create player: %v", err)
	}

	result, err := service.DeletePlayer(state.Player.ID)
	if err != nil {
		t.Fatalf("DeletePlayer schedule failed: %v", err)
	}
	if result.Status != "scheduled" || result.DeleteScheduledAt == "" {
		t.Fatalf("expected scheduled deletion result, got %+v", result)
	}

	players, err := service.ListPlayers(account.ID)
	if err != nil {
		t.Fatalf("ListPlayers failed: %v", err)
	}
	if len(players) != 1 || players[0].DeleteScheduledAt == "" {
		t.Fatalf("expected pending deletion summary, got %+v", players)
	}

	restored, err := service.RestorePlayerDeletion(state.Player.ID)
	if err != nil {
		t.Fatalf("RestorePlayerDeletion failed: %v", err)
	}
	if restored.Status != "restored" {
		t.Fatalf("expected restored result, got %+v", restored)
	}
	players, err = service.ListPlayers(account.ID)
	if err != nil {
		t.Fatalf("ListPlayers after restore failed: %v", err)
	}
	if len(players) != 1 || players[0].DeleteScheduledAt != "" {
		t.Fatalf("expected restored player without deletion schedule, got %+v", players)
	}
}

func TestListPlayersPurgesDueDeletedPlayers(t *testing.T) {
	service := NewService()
	repo := service.repo.(*MemoryRepository)
	now := time.Now()
	account := Account{ID: "acc_delete_due", Username: "delete_due", PasswordHash: "x", CreatedAt: now}
	if err := repo.CreateAccount(account); err != nil {
		t.Fatalf("create account: %v", err)
	}
	state := newPlayerState("player_delete_due", "到期删除", "wei", "caocao", now)
	state.DeleteRequestedAt = now.Add(-2 * time.Hour).UTC().Format(resourceDateLayout)
	state.DeleteScheduledAt = now.Add(-time.Hour).UTC().Format(resourceDateLayout)
	if err := repo.CreatePlayer(account.ID, state, now); err != nil {
		t.Fatalf("create player: %v", err)
	}

	players, err := service.ListPlayers(account.ID)
	if err != nil {
		t.Fatalf("ListPlayers failed: %v", err)
	}
	if len(players) != 0 {
		t.Fatalf("expected due deletion to be purged, got %+v", players)
	}
	if _, err := repo.GetState(state.Player.ID); !errors.Is(err, ErrPlayerNotFound) {
		t.Fatalf("expected player hard deleted, got %v", err)
	}
}

func TestDeletePlayerHardDeletesAfterDelay(t *testing.T) {
	service := NewService()
	repo := service.repo.(*MemoryRepository)
	now := time.Now()
	account := Account{ID: "acc_delete_due_click", Username: "delete_due_click", PasswordHash: "x", CreatedAt: now}
	if err := repo.CreateAccount(account); err != nil {
		t.Fatalf("create account: %v", err)
	}
	state := newPlayerState("player_delete_due_click", "到期点击删除", "wei", "caocao", now)
	state.DeleteRequestedAt = now.Add(-2 * time.Hour).UTC().Format(resourceDateLayout)
	state.DeleteScheduledAt = now.Add(-time.Hour).UTC().Format(resourceDateLayout)
	if err := repo.CreatePlayer(account.ID, state, now); err != nil {
		t.Fatalf("create player: %v", err)
	}

	result, err := service.DeletePlayer(state.Player.ID)
	if err != nil {
		t.Fatalf("DeletePlayer hard delete failed: %v", err)
	}
	if result.Status != "deleted" {
		t.Fatalf("expected deleted result, got %+v", result)
	}
	if _, err := repo.GetState(state.Player.ID); !errors.Is(err, ErrPlayerNotFound) {
		t.Fatalf("expected player hard deleted after second delete, got %v", err)
	}
}

func TestAdjustResourcesPublishesResourceChangedEvent(t *testing.T) {
	service := NewService()
	repo := service.repo.(*MemoryRepository)
	now := time.Now()
	account := Account{ID: "account_resource_event", Username: "resource_event", PasswordHash: "x", CreatedAt: now}
	if err := repo.CreateAccount(account); err != nil {
		t.Fatalf("create account: %v", err)
	}
	state := newPlayerState("player_resource_event", "ResourceEvent", "wei", "caocao", now)
	if err := repo.CreatePlayer(account.ID, state, now); err != nil {
		t.Fatalf("create player: %v", err)
	}

	events := []GameEvent{}
	service.SubscribeEvent(EventResourceChanged, func(event GameEvent) {
		events = append(events, event)
	})

	if _, err := service.AdjustResources(state.Player.ID, map[string]int{"wood": 10}); err != nil {
		t.Fatalf("AdjustResources failed: %v", err)
	}
	if len(events) != 1 || events[0].Type != EventResourceChanged || events[0].RefType != "resource_adjust" {
		t.Fatalf("expected resource changed event, got %+v", events)
	}
	changes, ok := events[0].Payload["changes"].(map[string]int)
	if !ok || changes["wood"] != 10 {
		t.Fatalf("expected wood delta 10, got %+v", events[0].Payload)
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
	cfg := loadOfficialGeneralsConfigForTest(t)
	zhangliao := cfg.Heroes["zhangliao"]
	zhangliao.SpecialTrait.Params["effectRate"] = 1.5
	cfg.Heroes["zhangliao"] = zhangliao
	err := ValidateGeneralsConfig(cfg)
	if err == nil || !strings.Contains(err.Error(), "effectRate=1.5 out of range") {
		t.Fatalf("expected invalid effectRate to be rejected, got %v", err)
	}
}

func TestValidateGeneralsConfigRejectsUnknownTraitParam(t *testing.T) {
	cfg := loadOfficialGeneralsConfigForTest(t)
	zhangliao := cfg.Heroes["zhangliao"]
	zhangliao.SpecialTrait.Params["badParam"] = 1
	cfg.Heroes["zhangliao"] = zhangliao
	err := ValidateGeneralsConfig(cfg)
	if err == nil || !strings.Contains(err.Error(), "contains unknown param badParam") {
		t.Fatalf("expected unknown trait param to be rejected, got %v", err)
	}
}

// loadOfficialGeneralsConfigForTest 直接读取正式配置副本，避免参数校验测试依赖可变全局状态。
func loadOfficialGeneralsConfigForTest(t *testing.T) GeneralsConfig {
	t.Helper()
	originalFactions := GetFactionsConfig()
	if err := LoadFactionsConfig("../../../config/factions.json"); err != nil {
		t.Fatalf("load factions config: %v", err)
	}
	t.Cleanup(func() {
		factionsMu.Lock()
		activeFactions = originalFactions
		factionsMu.Unlock()
	})
	content, err := os.ReadFile("../../../config/generals.json")
	if err != nil {
		t.Fatalf("read generals config: %v", err)
	}
	var cfg GeneralsConfig
	if err := json.Unmarshal(content, &cfg); err != nil {
		t.Fatalf("decode generals config: %v", err)
	}
	return NormalizeGeneralsConfig(cfg)
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

func TestModifierPipelineIncludesFactionTraitSource(t *testing.T) {
	original := GetFactionsConfig()
	t.Cleanup(func() {
		if err := SaveFactionsConfig("", original); err != nil {
			t.Fatalf("restore factions config: %v", err)
		}
	})

	if err := SaveFactionsConfig("", FactionsConfig{
		"wei": {
			Name:   "魏",
			Traits: map[string]float64{StatProductionBonus: 1.1},
		},
	}); err != nil {
		t.Fatalf("save factions config: %v", err)
	}

	state := newPlayerState("player_faction_trait", "FactionTrait", "wei", "caocao", time.Now())
	items := GetModifierBreakdown(&state, time.Now())
	found := false
	for _, item := range items {
		if item.Source == "阵营特性" && item.Key == StatProductionBonus {
			found = true
			if item.Mode != "percentMultiply" {
				t.Fatalf("expected percentMultiply, got %+v", item)
			}
		}
	}
	if !found {
		t.Fatalf("expected faction trait modifier in breakdown: %+v", items)
	}
}

func TestRegisterModifierSourceProviderAddsCustomSource(t *testing.T) {
	name := "test_custom_modifier_source"
	_ = RegisterModifierSourceProvider(name, func(state *GameState) []ModifierSource {
		return []ModifierSource{
			&StaticModifierSource{
				Name: "测试来源",
				Mods: []Modifier{
					{Key: "testRegistryBonus", Value: 0.25, Mode: "percentAdd"},
				},
			},
		}
	})

	state := newPlayerState("player_custom_modifier", "CustomModifier", "wei", "caocao", time.Now())
	items := GetModifierBreakdown(&state, time.Now())
	found := false
	for _, item := range items {
		if item.Source == "测试来源" && item.Key == "testRegistryBonus" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected custom modifier source in breakdown: %+v", items)
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

	next, err := service.AllocateGeneralStat("player_stat", "force", 1)
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

	if _, err := service.AllocateGeneralStat("player_stat_max", "force", 1); !errors.Is(err, ErrStatMaxLevel) {
		t.Fatalf("expected ErrStatMaxLevel, got %v", err)
	}
}

func TestResetGeneralStatsDeductsGoldAndClearsStats(t *testing.T) {
	setTestFactionsAndGenerals(t, FactionsConfig{
		"wei": {Name: "魏国", Generals: []GeneralInfo{{ID: "caocao", Name: "曹操"}}},
	}, GeneralsConfig{
		Enabled: true,
		Heroes: map[string]GeneralHeroConfig{
			"caocao": {ID: "caocao", Name: "曹操", Faction: "wei", Enabled: true},
		},
	})

	repo := NewMemoryRepository()
	service := NewServiceWithRepository(repo)
	now := time.Date(2026, 6, 24, 10, 0, 0, 0, time.UTC)
	account := Account{ID: "account_reset_general", Username: "reset_general", PasswordHash: "hash", Gold: 20, CreatedAt: now}
	if err := repo.CreateAccount(account); err != nil {
		t.Fatalf("create account: %v", err)
	}
	state := newPlayerState("player_reset_general", "测试", "wei", "caocao", now)
	state.General.Level = 5
	state.General.Stats = map[string]int{"force": 3, "politics": 2}
	applyHeroConfigToGeneral(state.General)
	if err := repo.CreatePlayer(account.ID, state, now); err != nil {
		t.Fatalf("create player: %v", err)
	}

	events := []GameEvent{}
	service.SubscribeEvent(EventGeneralChanged, func(event GameEvent) {
		events = append(events, event)
	})

	result, err := service.ResetGeneralStats(state.Player.ID)
	if err != nil {
		t.Fatalf("reset general stats: %v", err)
	}
	if result.AccountGold != 10 {
		t.Fatalf("expected account gold 10, got %d", result.AccountGold)
	}
	if result.State.General.Stats["force"] != 0 || result.State.General.Stats["politics"] != 0 {
		t.Fatalf("expected stats reset, got %+v", result.State.General.Stats)
	}
	if result.State.General.Level != 5 {
		t.Fatalf("expected level preserved, got %d", result.State.General.Level)
	}
	if result.State.General.AvailableStatPoints != 5 {
		t.Fatalf("expected 5 available points, got %d", result.State.General.AvailableStatPoints)
	}
	entries, err := service.ListGoldLedger(GoldLedgerFilter{AccountID: account.ID, RefType: LedgerRefGeneralReset})
	if err != nil {
		t.Fatalf("list ledger: %v", err)
	}
	if len(entries) != 1 || entries[0].Amount != GeneralResetStatsGoldCost || entries[0].BalanceAfter != 10 {
		t.Fatalf("unexpected ledger entries: %+v", entries)
	}
	if len(events) != 1 || events[0].Type != EventGeneralChanged || events[0].RefType != LedgerRefGeneralReset {
		t.Fatalf("expected general changed event, got %+v", events)
	}
}

func TestResetGeneralStatsRollbackWhenGoldInsufficient(t *testing.T) {
	setTestFactionsAndGenerals(t, FactionsConfig{
		"wei": {Name: "魏国", Generals: []GeneralInfo{{ID: "caocao", Name: "曹操"}}},
	}, GeneralsConfig{
		Enabled: true,
		Heroes: map[string]GeneralHeroConfig{
			"caocao": {ID: "caocao", Name: "曹操", Faction: "wei", Enabled: true},
		},
	})

	repo := NewMemoryRepository()
	service := NewServiceWithRepository(repo)
	now := time.Date(2026, 6, 24, 10, 0, 0, 0, time.UTC)
	account := Account{ID: "account_reset_general_no_gold", Username: "reset_general_no_gold", PasswordHash: "hash", Gold: 9, CreatedAt: now}
	if err := repo.CreateAccount(account); err != nil {
		t.Fatalf("create account: %v", err)
	}
	state := newPlayerState("player_reset_general_no_gold", "测试", "wei", "caocao", now)
	state.General.Level = 5
	state.General.Stats = map[string]int{"force": 3, "politics": 2}
	applyHeroConfigToGeneral(state.General)
	if err := repo.CreatePlayer(account.ID, state, now); err != nil {
		t.Fatalf("create player: %v", err)
	}

	if _, err := service.ResetGeneralStats(state.Player.ID); !errors.Is(err, ErrInsufficientGold) {
		t.Fatalf("expected ErrInsufficientGold, got %v", err)
	}
	currentAccount, err := repo.GetAccountByID(account.ID)
	if err != nil {
		t.Fatalf("get account: %v", err)
	}
	currentState, err := repo.GetState(state.Player.ID)
	if err != nil {
		t.Fatalf("get state: %v", err)
	}
	if currentAccount.Gold != 9 {
		t.Fatalf("expected account gold unchanged, got %d", currentAccount.Gold)
	}
	if currentState.General.Stats["force"] != 3 || currentState.General.Stats["politics"] != 2 {
		t.Fatalf("expected stats unchanged, got %+v", currentState.General.Stats)
	}
	entries, err := service.ListGoldLedger(GoldLedgerFilter{AccountID: account.ID, RefType: LedgerRefGeneralReset})
	if err != nil {
		t.Fatalf("list ledger: %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("expected no ledger entry on rollback, got %+v", entries)
	}
}

func TestChangeGeneralPreservesGrowthAndResetsStats(t *testing.T) {
	setTestFactionsAndGenerals(t, FactionsConfig{
		"wei": {Name: "魏国", Generals: []GeneralInfo{{ID: "caocao", Name: "曹操"}, {ID: "zhenmi", Name: "甄宓"}}},
		"wu":  {Name: "吴国", Generals: []GeneralInfo{{ID: "zhouyu", Name: "周瑜"}}},
	}, GeneralsConfig{
		Enabled: true,
		Heroes: map[string]GeneralHeroConfig{
			"caocao": {ID: "caocao", Name: "曹操", Faction: "wei", Enabled: true},
			"zhenmi": {ID: "zhenmi", Name: "甄宓", Faction: "wei", Enabled: true, Buffs: map[string]float64{"productionBonus": 0.1}},
			"zhouyu": {ID: "zhouyu", Name: "周瑜", Faction: "wu", Enabled: true},
		},
	})

	repo := NewMemoryRepository()
	service := NewServiceWithRepository(repo)
	now := time.Date(2026, 6, 24, 10, 0, 0, 0, time.UTC)
	account := Account{ID: "account_change_general", Username: "change_general", PasswordHash: "hash", Gold: 20, CreatedAt: now}
	if err := repo.CreateAccount(account); err != nil {
		t.Fatalf("create account: %v", err)
	}
	state := newPlayerState("player_change_general", "测试", "wei", "caocao", now)
	state.General.Level = 9
	state.General.Exp = 12345
	state.General.Stats = map[string]int{"force": 4}
	applyHeroConfigToGeneral(state.General)
	if err := repo.CreatePlayer(account.ID, state, now); err != nil {
		t.Fatalf("create player: %v", err)
	}

	result, err := service.ChangeGeneral(state.Player.ID, "zhenmi", "")
	if err != nil {
		t.Fatalf("change general: %v", err)
	}
	if result.State.General.ID != "zhenmi" || result.State.General.Name != "甄宓" {
		t.Fatalf("unexpected general after change: %+v", result.State.General)
	}
	if result.State.General.Level != 9 || result.State.General.Exp != 12345 {
		t.Fatalf("expected growth preserved, got level=%d exp=%d", result.State.General.Level, result.State.General.Exp)
	}
	if result.State.General.Stats["force"] != 0 || result.State.General.AvailableStatPoints != 9 {
		t.Fatalf("expected stats reset and points returned, got stats=%+v points=%d", result.State.General.Stats, result.State.General.AvailableStatPoints)
	}
	if result.State.General.Attributes[StatProductionBonus] <= 0 {
		t.Fatalf("expected new general fixed buff to apply, got %+v", result.State.General.Attributes)
	}

	if _, err := service.ChangeGeneral(state.Player.ID, "zhouyu", ""); !errors.Is(err, ErrInvalidGeneral) {
		t.Fatalf("expected cross-faction change rejected, got %v", err)
	}
	if _, err := service.ChangeGeneral(state.Player.ID, "zhenmi", ""); !errors.Is(err, ErrInvalidGeneral) {
		t.Fatalf("expected same-general change rejected, got %v", err)
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

func TestRecruitAllowsTwentyQueuesAndRejectsTwentyFirst(t *testing.T) {
	setTestCombatUnitsConfig(t)
	svc := NewService()
	repo := svc.repo.(*MemoryRepository)
	now := time.Now()
	account := Account{ID: "acc_recruit_limit", Username: "recruit-limit", PasswordHash: "x", CreatedAt: now}
	if err := repo.CreateAccount(account); err != nil {
		t.Fatalf("create account: %v", err)
	}
	state := newPlayerState("player_recruit_limit", "QueueLimit", "wei", "caocao", now)
	for key := range state.Resources.Items {
		state.Resources.Items[key] = 100000000
		state.Resources.Capacity[key] = 100000000
	}
	for index := 0; index < MaxRecruitQueues-1; index++ {
		state.RecruitQueues = append(state.RecruitQueues, RecruitQueue{ID: fmt.Sprintf("rq_%d", index), UnitType: "weiInfantry", Amount: 1, EndsAt: now.Add(time.Duration(index+1) * time.Hour).UTC().Format(resourceDateLayout)})
	}
	if err := repo.CreatePlayer(account.ID, state, now); err != nil {
		t.Fatalf("create player: %v", err)
	}
	next, err := svc.Recruit(state.Player.ID, "weiInfantry", 1)
	if err != nil {
		t.Fatalf("expected twentieth queue to succeed: %v", err)
	}
	if len(next.RecruitQueues) != MaxRecruitQueues {
		t.Fatalf("expected %d queues, got %d", MaxRecruitQueues, len(next.RecruitQueues))
	}
	if _, err := svc.Recruit(state.Player.ID, "weiInfantry", 1); !errors.Is(err, ErrQueueFull) {
		t.Fatalf("expected twenty-first queue to fail with ErrQueueFull, got %v", err)
	}
}

func TestInstantCompleteRecruitPublishesUnitChangedEvent(t *testing.T) {
	setTestCombatUnitsConfig(t)
	svc := NewService()
	repo := svc.repo.(*MemoryRepository)

	now := time.Now()
	account := Account{ID: "acc_recruit_event", Username: "recruit_event", PasswordHash: "x", CreatedAt: now}
	if err := repo.CreateAccount(account); err != nil {
		t.Fatalf("create account: %v", err)
	}

	state := newPlayerState("player_recruit_event", "RecruitEvent", "wei", "caocao", now)
	state.CityGold = 100
	state.RecruitQueues = []RecruitQueue{
		{
			ID:       "rq_event",
			UnitType: "weiInfantry",
			Amount:   7,
			EndsAt:   now.Add(60 * time.Second).UTC().Format(resourceDateLayout),
		},
	}
	if err := repo.CreatePlayer(account.ID, state, now); err != nil {
		t.Fatalf("create player: %v", err)
	}

	events := []GameEvent{}
	svc.SubscribeEvent(EventUnitChanged, func(event GameEvent) {
		events = append(events, event)
	})

	if _, err := svc.InstantCompleteRecruit(state.Player.ID, "rq_event"); err != nil {
		t.Fatalf("InstantCompleteRecruit failed: %v", err)
	}
	if len(events) != 1 || events[0].Type != EventUnitChanged || events[0].RefType != LedgerRefInstantRecruit {
		t.Fatalf("expected unit changed event, got %+v", events)
	}
	changes, ok := events[0].Payload["changes"].(map[string]int)
	if !ok || changes["weiInfantry"] != 7 {
		t.Fatalf("expected weiInfantry delta 7, got %+v", events[0].Payload)
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

	events := []GameEvent{}
	svc.SubscribeEvent(EventBuildingUpgraded, func(event GameEvent) {
		events = append(events, event)
	})

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
	if len(events) != 1 || events[0].Type != EventBuildingUpgraded || events[0].RefType != LedgerRefInstantBuilding {
		t.Fatalf("expected building upgraded event, got %+v", events)
	}
	changes, ok := events[0].Payload["changes"].(map[string]int)
	if !ok || changes["weapon_bureau-1"] != 1 {
		t.Fatalf("expected weapon bureau level delta 1, got %+v", events[0].Payload)
	}
}

func TestMutateBuildingCanDestroyBuildingAndPublishEvent(t *testing.T) {
	svc := NewService()
	repo := svc.repo.(*MemoryRepository)

	now := time.Now()
	account := Account{ID: "acc_building_destroy", Username: "building_destroy", PasswordHash: "x", CreatedAt: now}
	if err := repo.CreateAccount(account); err != nil {
		t.Fatalf("create account: %v", err)
	}
	state := newPlayerState("player_building_destroy", "Destroy", "wei", "caocao", now)
	if err := repo.CreatePlayer(account.ID, state, now); err != nil {
		t.Fatalf("create player: %v", err)
	}

	events := []GameEvent{}
	svc.SubscribeEvent(EventBuildingUpgraded, func(event GameEvent) {
		events = append(events, event)
	})

	next, err := svc.MutateBuilding(state.Player.ID, BuildingMutation{
		Type:       BuildingMutationDestroy,
		BuildingID: "wood_camp-1",
		Reason:     "test_destroy",
	})
	if err != nil {
		t.Fatalf("MutateBuilding destroy failed: %v", err)
	}
	var found Building
	for _, building := range next.Buildings {
		if building.ID == "wood_camp-1" {
			found = building
			break
		}
	}
	if found.Status != BuildingStatusDestroyed {
		t.Fatalf("expected building destroyed, got %+v", found)
	}
	if len(events) != 1 || events[0].RefType != "test_destroy" {
		t.Fatalf("expected building status event, got %+v", events)
	}
	statusChanges, ok := events[0].Payload["statusChanges"].(map[string]string)
	if !ok || statusChanges["wood_camp-1"] != BuildingStatusDestroyed {
		t.Fatalf("expected destroyed status change, got %+v", events[0].Payload)
	}
}

func TestDestroyedBuildingCannotStartUpgrade(t *testing.T) {
	svc := NewService()
	repo := svc.repo.(*MemoryRepository)

	now := time.Now()
	account := Account{ID: "acc_building_blocked", Username: "building_blocked", PasswordHash: "x", CreatedAt: now}
	if err := repo.CreateAccount(account); err != nil {
		t.Fatalf("create account: %v", err)
	}
	state := newPlayerState("player_building_blocked", "Blocked", "wei", "caocao", now)
	for i := range state.Buildings {
		if state.Buildings[i].ID == "wood_camp-1" {
			state.Buildings[i].Status = BuildingStatusDestroyed
			break
		}
	}
	if err := repo.CreatePlayer(account.ID, state, now); err != nil {
		t.Fatalf("create player: %v", err)
	}

	_, err := svc.UpgradeBuilding(state.Player.ID, "wood_camp-1")
	if !errors.Is(err, ErrBuildingStatusBlocked) {
		t.Fatalf("expected blocked upgrade error, got %v", err)
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

func TestSimulateBattleUsesRegisteredSceneRule(t *testing.T) {
	setTestCombatUnitsConfig(t)
	originalCombat := combat.GetCombatConfig()
	t.Cleanup(func() {
		if err := combat.SaveCombatConfig("", originalCombat); err != nil {
			t.Fatalf("restore combat config: %v", err)
		}
	})
	if err := combat.RegisterRule(combat.RuleConfig{
		ID:               "test_scene_plunder_rule",
		Name:             "测试场景掠夺规则",
		Mode:             "plunder",
		Exponent:         1.422,
		EqualResult:      "half_each",
		LossDistribution: "proportional",
		DefenseFormula:   "weighted",
	}); err != nil {
		t.Fatalf("register combat rule: %v", err)
	}
	if err := combat.SetActiveRule(combat.ScenePVEAttack, "test_scene_plunder_rule"); err != nil {
		t.Fatalf("set active combat rule: %v", err)
	}

	svc := NewService()
	repo := svc.repo.(*MemoryRepository)
	now := time.Now()
	account := Account{ID: "acc_sim_scene_rule", Username: "sim_scene_rule", PasswordHash: "x", CreatedAt: now}
	if err := repo.CreateAccount(account); err != nil {
		t.Fatalf("create account: %v", err)
	}
	state := newPlayerState("player_sim_scene_rule", "SceneRule", "wei", "caocao", now)
	if err := repo.CreatePlayer(account.ID, state, now); err != nil {
		t.Fatalf("create player: %v", err)
	}

	result, err := svc.SimulateBattle(BattleSimulationRequest{
		PlayerID:        state.Player.ID,
		Mode:            "attack",
		AttackerFaction: "wei",
		DefenderFaction: "wei",
		AttackerUnits:   map[string]int{"weiInfantry": 100},
		DefenderUnits:   map[string]int{"weiInfantry": 100},
	})
	if err != nil {
		t.Fatalf("SimulateBattle failed: %v", err)
	}
	if result.Result.Mode != "plunder" {
		t.Fatalf("expected scene rule mode plunder, got %+v", result.Result)
	}
}

func TestAttackNpcUsesStateTransactionAndPublishesBattleEvent(t *testing.T) {
	setTestCombatUnitsConfig(t)

	svc := NewService()
	repo := svc.repo.(*MemoryRepository)
	now := time.Now()
	account := Account{ID: "acc_attack_event", Username: "attack_event", PasswordHash: "x", CreatedAt: now}
	if err := repo.CreateAccount(account); err != nil {
		t.Fatalf("create account: %v", err)
	}
	state := newPlayerState("player_attack_event", "Attacker", "wei", "caocao", now)
	state.Army = []ArmyUnit{{UnitType: "weiInfantry", Amount: 10}}
	state.NpcState = &NpcState{
		Cities: []NpcCity{{
			ID:                "npc_event_1",
			Name:              "事件城",
			Faction:           "wei",
			Resources:         map[string]int{"wood": 20},
			StorageCapacity:   map[string]int{},
			ProductionPerHour: map[string]int{},
			Army:              []ArmyUnit{},
			ResourceSettledAt: now.UTC().Format(resourceDateLayout),
			ArmySettledAt:     now.UTC().Format(resourceDateLayout),
			GeneratedAt:       now.UTC().Format(resourceDateLayout),
		}},
		LastRefreshedAt: now.UTC().Format(resourceDateLayout),
	}
	if err := repo.CreatePlayer(account.ID, state, now); err != nil {
		t.Fatalf("create player: %v", err)
	}

	events := []GameEvent{}
	rewardEvents := []GameEvent{}
	svc.SubscribeEvent(EventBattleFinished, func(event GameEvent) {
		events = append(events, event)
	})
	svc.SubscribeEvent(EventRewardGranted, func(event GameEvent) {
		rewardEvents = append(rewardEvents, event)
	})

	result, err := svc.AttackNpc(AttackNpcRequest{
		PlayerID: state.Player.ID,
		NpcID:    "npc_event_1",
		Mode:     "attack",
		Units:    map[string]int{"weiInfantry": 5},
	})
	if err != nil {
		t.Fatalf("AttackNpc failed: %v", err)
	}
	if result.BattleReport.ID == "" || result.BattleReport.Type != "attack" {
		t.Fatalf("unexpected battle report: %+v", result.BattleReport)
	}
	if len(result.BattleReport.GrantedRewards) != 1 ||
		result.BattleReport.GrantedRewards[0].Type != RewardTypeResource ||
		result.BattleReport.GrantedRewards[0].ID != "wood" ||
		result.BattleReport.GrantedRewards[0].Amount != 20 {
		t.Fatalf("expected battle granted wood reward, got %+v", result.BattleReport.GrantedRewards)
	}
	if len(events) != 1 || events[0].Type != EventBattleFinished || events[0].RefID != result.BattleReport.ID {
		t.Fatalf("expected battle finished event, got %+v", events)
	}
	if len(rewardEvents) != 1 || rewardEvents[0].Type != EventRewardGranted || rewardEvents[0].RefID != result.BattleReport.ID {
		t.Fatalf("expected battle reward event, got %+v", rewardEvents)
	}

	stored, err := repo.GetState(state.Player.ID)
	if err != nil {
		t.Fatalf("get state: %v", err)
	}
	if len(stored.Army) != 1 || stored.Army[0].UnitType != "weiInfantry" || stored.Army[0].Amount != 10 {
		t.Fatalf("expected surviving army returned through state transaction, got %+v", stored.Army)
	}
}

// combatScopeRecorderRepository 记录服务传入战斗仓储的作用域，便于验证业务路径没有退回全量锁。
type combatScopeRecorderRepository struct {
	*MemoryRepository
	combatScopes []CombatAssetScope
}

// UpdateCombatState 记录战斗作用域后复用内存仓储行为。
func (r *combatScopeRecorderRepository) UpdateCombatState(playerID string, scope CombatAssetScope, updatedAt time.Time, update func(state *GameState) error) (GameState, error) {
	r.combatScopes = append(r.combatScopes, scope)
	return r.MemoryRepository.UpdateCombatState(playerID, scope, updatedAt, update)
}

func TestScoutNpcPassesScoutUnitCombatScope(t *testing.T) {
	setTestCombatUnitsConfig(t)
	addPvpScoutTestUnits(t)

	baseRepo := NewMemoryRepository()
	repo := &combatScopeRecorderRepository{MemoryRepository: baseRepo}
	svc := NewServiceWithRepository(repo)
	now := time.Now()
	account := Account{ID: "acc_scout_scope", Username: "scout_scope", PasswordHash: "x", CreatedAt: now}
	if err := repo.CreateAccount(account); err != nil {
		t.Fatalf("create account: %v", err)
	}
	state := newPlayerState("player_scout_scope", "ScoutScope", "wei", "caocao", now)
	state.Army = []ArmyUnit{{UnitType: "weiScout", Amount: 5}, {UnitType: "weiInfantry", Amount: 99}}
	state.NpcState = &NpcState{
		Cities: []NpcCity{{
			ID:                "npc_scout_scope",
			Name:              "侦查作用域城",
			Faction:           "wei",
			Resources:         map[string]int{"wood": 10},
			StorageCapacity:   map[string]int{},
			ProductionPerHour: map[string]int{},
			Army:              []ArmyUnit{{UnitType: "weiScout", Amount: 2}, {UnitType: "weiInfantry", Amount: 10}},
			ResourceSettledAt: now.UTC().Format(resourceDateLayout),
			ArmySettledAt:     now.UTC().Format(resourceDateLayout),
			GeneratedAt:       now.UTC().Format(resourceDateLayout),
		}},
		LastRefreshedAt: now.UTC().Format(resourceDateLayout),
	}
	if err := repo.CreatePlayer(account.ID, state, now); err != nil {
		t.Fatalf("create player: %v", err)
	}

	if _, err := svc.ScoutNpc(ScoutNpcRequest{PlayerID: state.Player.ID, NpcID: "npc_scout_scope"}); err != nil {
		t.Fatalf("ScoutNpc failed: %v", err)
	}
	if len(repo.combatScopes) != 1 {
		t.Fatalf("expected one combat scope, got %+v", repo.combatScopes)
	}
	scope := repo.combatScopes[0]
	if len(scope.UnitTypes) != 1 || scope.UnitTypes[0] != "weiScout" {
		t.Fatalf("expected scout scope to lock only weiScout, got %+v", scope)
	}
	if len(scope.GeneralIDs) != 0 {
		t.Fatalf("expected scout scope not to lock generals, got %+v", scope)
	}
	if !scope.SkipInventory {
		t.Fatalf("expected scout scope to skip inventory, got %+v", scope)
	}
}

func TestSweepNpcCreatesOneAggregateReport(t *testing.T) {
	setTestCombatUnitsConfig(t)

	baseRepo := NewMemoryRepository()
	repo := &combatScopeRecorderRepository{MemoryRepository: baseRepo}
	svc := NewServiceWithRepository(repo)
	now := time.Now()
	account := Account{ID: "acc_sweep_report", Username: "sweep_report", PasswordHash: "x", CreatedAt: now}
	if err := repo.CreateAccount(account); err != nil {
		t.Fatalf("create account: %v", err)
	}
	state := newPlayerState("player_sweep_report", "Sweep", "wei", "caocao", now)
	state.Army = []ArmyUnit{{UnitType: "weiInfantry", Amount: 100}}
	state.Resources.Capacity = map[string]int{"wood": 10000, "stone": 10000, "iron": 10000, "food": 10000}
	state.NpcState = &NpcState{
		Cities: []NpcCity{
			testNpcCity("npc_sweep_1", now),
			testNpcCity("npc_sweep_2", now),
		},
		LastRefreshedAt: now.UTC().Format(resourceDateLayout),
	}
	if err := repo.CreatePlayer(account.ID, state, now); err != nil {
		t.Fatalf("create player: %v", err)
	}

	events := []GameEvent{}
	svc.SubscribeEvent(EventBattleFinished, func(event GameEvent) {
		events = append(events, event)
	})

	result, err := svc.SweepNpc(SweepNpcRequest{
		PlayerID: state.Player.ID,
		NpcIDs:   []string{"npc_sweep_1", "npc_sweep_2"},
		Mode:     "attack",
	})
	if err != nil {
		t.Fatalf("SweepNpc failed: %v", err)
	}
	if result.Done != 2 || result.Failed != 0 || result.Stopped {
		t.Fatalf("unexpected sweep result: %+v", result)
	}
	if result.BattleReport.BattleType != "sweep" || result.BattleReport.Title != "NPC 扫荡" {
		t.Fatalf("expected aggregate sweep report, got %+v", result.BattleReport)
	}
	if result.BattleReport.DispatchedUnits["weiInfantry"] != 100 {
		t.Fatalf("expected sweep attacker dispatched units to use initial army only, got %+v", result.BattleReport.DispatchedUnits)
	}
	if result.BattleReport.Detail == nil || result.BattleReport.Detail.PrimarySide.Units[0].Dispatched != 100 {
		t.Fatalf("expected sweep detail primary side to use initial dispatched units, got %+v", result.BattleReport.Detail)
	}
	sweepExtra, ok := result.BattleReport.Detail.Extra["sweep"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected sweep extra in aggregate report detail, got %+v", result.BattleReport.Detail.Extra)
	}
	if sweepExtra["requested"] != 2 || sweepExtra["success"] != 2 || sweepExtra["failed"] != 0 || sweepExtra["stopped"] != false {
		t.Fatalf("unexpected sweep extra: %+v", sweepExtra)
	}
	defenders, ok := sweepExtra["defenders"].([]BattleReportSweepDefender)
	if !ok || len(defenders) != 2 {
		t.Fatalf("expected two sweep defender snapshots, got %#v", sweepExtra["defenders"])
	}
	for _, defender := range defenders {
		if defender.TargetID == "" || defender.Power <= 0 || len(defender.Units) != 0 || len(defender.Resources) != 0 {
			t.Fatalf("expected lightweight defender snapshot without units/resources, got %+v", defender)
		}
	}

	reports, total, err := repo.ListReportsByQuery(BattleReportQuery{PlayerID: state.Player.ID, BattleType: "sweep", Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("list sweep reports: %v", err)
	}
	if total != 1 || len(reports) != 1 || reports[0].ID != result.BattleReport.ID {
		t.Fatalf("expected one stored sweep report, total=%d reports=%+v", total, reports)
	}
	attackReports, attackTotal, err := repo.ListReportsByQuery(BattleReportQuery{PlayerID: state.Player.ID, BattleType: "attack", Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("list attack reports: %v", err)
	}
	if attackTotal != 0 || len(attackReports) != 0 {
		t.Fatalf("expected no per-attack reports from sweep, total=%d reports=%+v", attackTotal, attackReports)
	}
	if len(events) != 1 || events[0].RefID != result.BattleReport.ID {
		t.Fatalf("expected one aggregate battle event, got %+v", events)
	}
	if len(repo.combatScopes) != 1 {
		t.Fatalf("expected sweep to use one batch combat transaction, got %d scopes", len(repo.combatScopes))
	}
}

func TestSweepNpcCapsTargetsToLatencyLimit(t *testing.T) {
	setTestCombatUnitsConfig(t)

	svc := NewService()
	repo := svc.repo.(*MemoryRepository)
	now := time.Now()
	account := Account{ID: "acc_sweep_cap", Username: "sweep_cap", PasswordHash: "x", CreatedAt: now}
	if err := repo.CreateAccount(account); err != nil {
		t.Fatalf("create account: %v", err)
	}
	state := newPlayerState("player_sweep_cap", "SweepCap", "wei", "caocao", now)
	state.Army = []ArmyUnit{{UnitType: "weiInfantry", Amount: 5000}}
	state.Resources.Capacity = map[string]int{"wood": 10000, "stone": 10000, "iron": 10000, "food": 10000}
	npcIDs := make([]string, 0, maxNpcSweepTargets+5)
	state.NpcState = &NpcState{LastRefreshedAt: now.UTC().Format(resourceDateLayout)}
	for i := 0; i < maxNpcSweepTargets+5; i++ {
		npcID := fmt.Sprintf("npc_sweep_cap_%02d", i)
		npcIDs = append(npcIDs, npcID)
		state.NpcState.Cities = append(state.NpcState.Cities, testNpcCity(npcID, now))
	}
	if err := repo.CreatePlayer(account.ID, state, now); err != nil {
		t.Fatalf("create player: %v", err)
	}

	result, err := svc.SweepNpc(SweepNpcRequest{
		PlayerID: state.Player.ID,
		NpcIDs:   npcIDs,
		Mode:     "attack",
	})
	if err != nil {
		t.Fatalf("SweepNpc failed: %v", err)
	}
	if result.Done != maxNpcSweepTargets || result.Failed != 0 || result.Stopped {
		t.Fatalf("expected sweep to process capped target count, got %+v", result)
	}
	sweepExtra, ok := result.BattleReport.Detail.Extra["sweep"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected sweep extra in aggregate report detail, got %+v", result.BattleReport.Detail.Extra)
	}
	if sweepExtra["requested"] != maxNpcSweepTargets || sweepExtra["success"] != maxNpcSweepTargets {
		t.Fatalf("expected sweep report to use capped requested count, got %+v", sweepExtra)
	}
	defenders, ok := sweepExtra["defenders"].([]BattleReportSweepDefender)
	if !ok || len(defenders) != maxNpcSweepTargets {
		t.Fatalf("expected capped defender snapshots, got %#v", sweepExtra["defenders"])
	}
}

func TestNpcSweepTaskCompletesWithAggregateResult(t *testing.T) {
	setTestCombatUnitsConfig(t)

	svc := NewService()
	repo := svc.repo.(*MemoryRepository)
	now := time.Now()
	account := Account{ID: "acc_sweep_task", Username: "sweep_task", PasswordHash: "x", CreatedAt: now}
	if err := repo.CreateAccount(account); err != nil {
		t.Fatalf("create account: %v", err)
	}
	state := newPlayerState("player_sweep_task", "SweepTask", "wei", "caocao", now)
	state.Army = []ArmyUnit{{UnitType: "weiInfantry", Amount: 100}}
	state.Resources.Capacity = map[string]int{"wood": 10000, "stone": 10000, "iron": 10000, "food": 10000}
	state.NpcState = &NpcState{
		Cities: []NpcCity{
			testNpcCity("npc_sweep_task_1", now),
			testNpcCity("npc_sweep_task_2", now),
		},
		LastRefreshedAt: now.UTC().Format(resourceDateLayout),
	}
	if err := repo.CreatePlayer(account.ID, state, now); err != nil {
		t.Fatalf("create player: %v", err)
	}

	task, err := svc.StartNpcSweepTask(SweepNpcRequest{
		PlayerID: state.Player.ID,
		NpcIDs:   []string{"npc_sweep_task_1", "npc_sweep_task_2"},
		Mode:     "attack",
	})
	if err != nil {
		t.Fatalf("StartNpcSweepTask failed: %v", err)
	}
	if task.ID == "" || task.Status != SweepTaskStatusQueued || task.Requested != 2 {
		t.Fatalf("unexpected initial task: %+v", task)
	}

	completed := waitForNpcSweepTask(t, svc, state.Player.ID, task.ID)
	if completed.Status != SweepTaskStatusCompleted || completed.Result == nil {
		t.Fatalf("expected completed task with result, got %+v", completed)
	}
	if completed.Result.Done != 2 || completed.Result.BattleReport.BattleType != "sweep" {
		t.Fatalf("unexpected sweep task result: %+v", completed.Result)
	}
}

func TestCreateNpcSweepTaskRejectsConcurrentActiveTasks(t *testing.T) {
	repo := NewMemoryRepository()
	now := time.Now().UTC().Format(resourceDateLayout)
	var successCount int32
	var runningCount int32
	unexpected := make(chan error, 8)
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, err := repo.CreateNpcSweepTask(NpcSweepTask{
				ID:        fmt.Sprintf("sweep_task_atomic_%d", i),
				PlayerID:  "player_sweep_atomic",
				NpcIDs:    []string{"npc_sweep_atomic"},
				Mode:      "attack",
				Status:    SweepTaskStatusQueued,
				CreatedAt: now,
				UpdatedAt: now,
			})
			if err == nil {
				atomic.AddInt32(&successCount, 1)
				return
			}
			if errors.Is(err, ErrNpcSweepTaskRunning) {
				atomic.AddInt32(&runningCount, 1)
				return
			}
			unexpected <- err
		}(i)
	}
	wg.Wait()
	close(unexpected)
	for err := range unexpected {
		t.Fatalf("unexpected create error: %v", err)
	}
	if successCount != 1 || runningCount != 7 {
		t.Fatalf("expected one created task and seven running errors, success=%d running=%d", successCount, runningCount)
	}
}

func TestNewServiceMarksInterruptedNpcSweepTasksFailed(t *testing.T) {
	repo := NewMemoryRepository()
	now := time.Now().UTC().Format(resourceDateLayout)
	created, err := repo.CreateNpcSweepTask(NpcSweepTask{
		ID:        "sweep_task_interrupted",
		PlayerID:  "player_sweep_interrupted",
		NpcIDs:    []string{"npc_sweep_interrupted"},
		Mode:      "attack",
		Status:    SweepTaskStatusRunning,
		CreatedAt: now,
		UpdatedAt: now,
	})
	if err != nil {
		t.Fatalf("create active task: %v", err)
	}

	svc := NewServiceWithRepository(repo)
	recovered, err := svc.GetNpcSweepTask("player_sweep_interrupted", created.ID)
	if err != nil {
		t.Fatalf("get recovered task: %v", err)
	}
	if recovered.Status != SweepTaskStatusFailed || recovered.CompletedAt == "" || recovered.Error == "" {
		t.Fatalf("expected interrupted task to be marked failed, got %+v", recovered)
	}
}

func TestStartNpcSweepTaskRejectsTooManyTargets(t *testing.T) {
	svc := NewService()
	npcIDs := make([]string, 0, maxNpcSweepTaskTargets+1)
	for i := 0; i < maxNpcSweepTaskTargets+1; i++ {
		npcIDs = append(npcIDs, fmt.Sprintf("npc_sweep_limit_%02d", i))
	}

	_, err := svc.StartNpcSweepTask(SweepNpcRequest{
		PlayerID: "player_sweep_limit",
		NpcIDs:   npcIDs,
		Mode:     "attack",
	})
	if !errors.Is(err, ErrSweepTargetsTooMany) {
		t.Fatalf("expected too many targets error, got %v", err)
	}
}

func TestSweepNpcKeepsInitialGeneralSnapshotAndAggregatesCrossBattleLevelUps(t *testing.T) {
	setTestCombatUnitsConfig(t)
	setTestGeneralsConfig(t, GeneralsConfig{
		Enabled: true,
		Common: GeneralsCommonConfig{
			ExpCurve:   []int{0, 1, 2, 1000},
			LevelBuffs: map[int]map[string]float64{1: {}, 2: {}, 3: {}},
		},
		Heroes: map[string]GeneralHeroConfig{
			"test_general": {ID: "test_general", Name: "测试将领", Faction: "wei", Enabled: true},
		},
	})

	svc := NewService()
	repo := svc.repo.(*MemoryRepository)
	now := time.Now()
	account := Account{ID: "acc_sweep_level_up", Username: "sweep_level_up", PasswordHash: "x", CreatedAt: now}
	if err := repo.CreateAccount(account); err != nil {
		t.Fatalf("create account: %v", err)
	}
	state := newPlayerState("player_sweep_level_up", "SweepLevel", "wei", "test_general", now)
	state.Army = []ArmyUnit{{UnitType: "weiInfantry", Amount: 100}}
	state.Resources.Capacity = map[string]int{"wood": 10000, "stone": 10000, "iron": 10000, "food": 10000}
	firstCity := testNpcCity("npc_sweep_level_up_1", now)
	firstCity.Army = []ArmyUnit{{UnitType: "weiInfantry", Amount: 1}}
	firstCity.MaxArmy = []ArmyUnit{{UnitType: "weiInfantry", Amount: 1}}
	secondCity := testNpcCity("npc_sweep_level_up_2", now)
	secondCity.Army = []ArmyUnit{{UnitType: "weiInfantry", Amount: 1}}
	secondCity.MaxArmy = []ArmyUnit{{UnitType: "weiInfantry", Amount: 1}}
	state.NpcState = &NpcState{
		Cities:          []NpcCity{firstCity, secondCity},
		LastRefreshedAt: now.UTC().Format(resourceDateLayout),
	}
	if err := repo.CreatePlayer(account.ID, state, now); err != nil {
		t.Fatalf("create player: %v", err)
	}

	result, err := svc.SweepNpc(SweepNpcRequest{
		PlayerID:   state.Player.ID,
		NpcIDs:     []string{"npc_sweep_level_up_1", "npc_sweep_level_up_2"},
		Mode:       "attack",
		GeneralIDs: []string{"test_general"},
	})
	if err != nil {
		t.Fatalf("SweepNpc failed: %v", err)
	}
	if result.Done != 2 || result.BattleReport.BattleType != "sweep" || result.BattleReport.GeneralExpGained != 2 || result.BattleReport.GeneralLevelBefore != 1 || result.BattleReport.GeneralLevelAfter != 3 {
		t.Fatalf("expected two-battle sweep to aggregate 2 exp and Lv.1 -> Lv.3, got %+v", result.BattleReport)
	}
	if len(result.BattleReport.PvpAttackerGenerals) != 1 || result.BattleReport.PvpAttackerGenerals[0].Level != 1 {
		t.Fatalf("expected sweep aggregate to keep the first battle pre-battle general snapshot, got %+v", result.BattleReport.PvpAttackerGenerals)
	}
	if result.BattleReport.Detail == nil || len(result.BattleReport.Detail.PrimarySide.Generals) != 1 || result.BattleReport.Detail.PrimarySide.Generals[0].Level != 1 ||
		result.BattleReport.Detail.Rewards.GeneralLevelBefore != result.BattleReport.GeneralLevelBefore || result.BattleReport.Detail.Rewards.GeneralLevelAfter != result.BattleReport.GeneralLevelAfter {
		t.Fatalf("expected sweep detail to separate pre-battle snapshot and final upgrade, got %+v", result.BattleReport.Detail)
	}
	sweepExtra, ok := result.BattleReport.Detail.Extra["sweep"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected sweep extra in aggregate report detail, got %+v", result.BattleReport.Detail.Extra)
	}
	defenders, ok := sweepExtra["defenders"].([]BattleReportSweepDefender)
	if !ok || len(defenders) != 2 {
		t.Fatalf("expected two sweep defender snapshots, got %#v", sweepExtra["defenders"])
	}
	stored, err := repo.GetState(state.Player.ID)
	if err != nil {
		t.Fatalf("get state after sweep: %v", err)
	}
	if got := pvpTestGeneralLevel(stored, "test_general"); got != 3 || pvpTestGeneralExp(stored, "test_general") != 2 {
		t.Fatalf("expected stored general Lv.3 with 2 exp, got level=%d exp=%d", got, pvpTestGeneralExp(stored, "test_general"))
	}

	sweepReports, sweepTotal, err := repo.ListReportsByQuery(BattleReportQuery{PlayerID: state.Player.ID, BattleType: "sweep", Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("list sweep reports: %v", err)
	}
	if sweepTotal != 1 || len(sweepReports) != 1 {
		t.Fatalf("expected one sweep report, total=%d reports=%+v", sweepTotal, sweepReports)
	}
	attackReports, attackTotal, err := repo.ListReportsByQuery(BattleReportQuery{PlayerID: state.Player.ID, BattleType: "attack", Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("list attack reports: %v", err)
	}
	if attackTotal != 0 || len(attackReports) != 0 {
		t.Fatalf("expected no per-attack reports from sweep level up, total=%d reports=%+v", attackTotal, attackReports)
	}
}

func TestPlayerAssetActionsAreSerializedPerPlayer(t *testing.T) {
	setTestCombatUnitsConfig(t)
	loadTestItemsConfig(t)

	svc := NewService()
	baseRepo := svc.repo.(*MemoryRepository)
	blockingRepo := &blockingAssetRepository{
		MemoryRepository: baseRepo,
		entered:          make(chan string, 2),
		release:          make(chan struct{}),
	}
	svc.repo = blockingRepo

	now := time.Now()
	account := Account{ID: "acc_serial_assets", Username: "serial_assets", PasswordHash: "x", CreatedAt: now}
	if err := blockingRepo.CreateAccount(account); err != nil {
		t.Fatalf("create account: %v", err)
	}
	state := newPlayerState("player_serial_assets", "Serial", "wei", "caocao", now)
	state.Army = []ArmyUnit{{UnitType: "weiInfantry", Amount: 100}}
	state.Inventory = map[string]ItemStack{
		"test_general_exp_small": {ItemID: "test_general_exp_small", Amount: 1, UpdatedAt: now.UTC().Format(resourceDateLayout)},
	}
	state.InventorySlots = []ItemStack{{ItemID: "test_general_exp_small", Amount: 1, UpdatedAt: now.UTC().Format(resourceDateLayout)}}
	state.NpcState = &NpcState{
		Cities:          []NpcCity{testNpcCity("npc_serial_assets", now)},
		LastRefreshedAt: now.UTC().Format(resourceDateLayout),
	}
	if err := blockingRepo.CreatePlayer(account.ID, state, now); err != nil {
		t.Fatalf("create player: %v", err)
	}

	errs := make(chan error, 2)
	go func() {
		_, err := svc.AttackNpc(AttackNpcRequest{
			PlayerID: state.Player.ID,
			NpcID:    "npc_serial_assets",
			Mode:     "attack",
			Units:    map[string]int{"weiInfantry": 10},
		})
		errs <- err
	}()
	if got := <-blockingRepo.entered; got != "combat" {
		t.Fatalf("expected combat transaction to enter first, got %s", got)
	}

	go func() {
		_, err := svc.UseItem(state.Player.ID, "test_general_exp_small", 1)
		errs <- err
	}()
	select {
	case got := <-blockingRepo.entered:
		t.Fatalf("expected item transaction to wait for player lock, but %s entered concurrently", got)
	case <-time.After(50 * time.Millisecond):
	}

	blockingRepo.release <- struct{}{}
	if got := <-blockingRepo.entered; got != "general_exp_item" {
		t.Fatalf("expected general exp item transaction after combat release, got %s", got)
	}
	blockingRepo.release <- struct{}{}

	for i := 0; i < 2; i++ {
		if err := <-errs; err != nil {
			t.Fatalf("concurrent asset action failed: %v", err)
		}
	}
	if got := atomic.LoadInt32(&blockingRepo.maxActive); got != 1 {
		t.Fatalf("expected at most one active player asset transaction, got %d", got)
	}
}

func TestNpcAttackReturnsTooFastWhileSweepIsRunning(t *testing.T) {
	setTestCombatUnitsConfig(t)

	svc := NewService()
	baseRepo := svc.repo.(*MemoryRepository)
	blockingRepo := &blockingAssetRepository{
		MemoryRepository: baseRepo,
		entered:          make(chan string, 2),
		release:          make(chan struct{}),
	}
	svc.repo = blockingRepo

	now := time.Now()
	account := Account{ID: "acc_sweep_busy", Username: "sweep_busy", PasswordHash: "x", CreatedAt: now}
	if err := blockingRepo.CreateAccount(account); err != nil {
		t.Fatalf("create account: %v", err)
	}
	state := newPlayerState("player_sweep_busy", "SweepBusy", "wei", "caocao", now)
	state.Army = []ArmyUnit{{UnitType: "weiInfantry", Amount: 100}}
	state.Resources.Capacity = map[string]int{"wood": 10000, "stone": 10000, "iron": 10000, "food": 10000}
	state.NpcState = &NpcState{
		Cities: []NpcCity{
			testNpcCity("npc_sweep_busy_1", now),
			testNpcCity("npc_sweep_busy_2", now),
		},
		LastRefreshedAt: now.UTC().Format(resourceDateLayout),
	}
	if err := blockingRepo.CreatePlayer(account.ID, state, now); err != nil {
		t.Fatalf("create player: %v", err)
	}

	sweepDone := make(chan error, 1)
	go func() {
		_, err := svc.SweepNpc(SweepNpcRequest{
			PlayerID: state.Player.ID,
			NpcIDs:   []string{"npc_sweep_busy_1"},
			Mode:     "attack",
		})
		sweepDone <- err
	}()
	if got := <-blockingRepo.entered; got != "combat" {
		t.Fatalf("expected sweep combat transaction to enter first, got %s", got)
	}

	_, err := svc.AttackNpc(AttackNpcRequest{
		PlayerID: state.Player.ID,
		NpcID:    "npc_sweep_busy_2",
		Mode:     "attack",
		Units:    map[string]int{"weiInfantry": 10},
	})
	if !errors.Is(err, ErrOperationTooFast) {
		t.Fatalf("expected ErrOperationTooFast while sweep is running, got %v", err)
	}
	select {
	case got := <-blockingRepo.entered:
		t.Fatalf("expected busy attack not to enter storage transaction, but %s entered", got)
	default:
	}

	blockingRepo.release <- struct{}{}
	if err := <-sweepDone; err != nil {
		t.Fatalf("sweep failed: %v", err)
	}
}

func TestAttackNpcGrantsConfiguredItemDrops(t *testing.T) {
	setTestCombatUnitsConfig(t)
	root := filepath.Join("..", "..", "..", "config")
	if err := LoadItemsConfig(filepath.Join(root, "items.json")); err != nil {
		t.Fatalf("LoadItemsConfig failed: %v", err)
	}
	defer func() { _ = LoadItemsConfig(filepath.Join(root, "items.json")) }()
	if err := LoadDropPoolsConfig(filepath.Join(root, "drop_pools.json")); err != nil {
		t.Fatalf("LoadDropPoolsConfig root failed: %v", err)
	}
	defer func() { _ = LoadDropPoolsConfig(filepath.Join(root, "drop_pools.json")) }()
	if err := LoadNpcConfig(filepath.Join(root, "npc.json")); err != nil {
		t.Fatalf("LoadNpcConfig root failed: %v", err)
	}
	defer func() { _ = LoadNpcConfig(filepath.Join(root, "npc.json")) }()
	npcCfg := GetNpcConfig()

	poolPath := filepath.Join(t.TempDir(), "drop_pools.json")
	pools := DropPoolsConfig{
		"test_npc_exp_pack": {
			ID: "test_npc_exp_pack",
			Slots: []DropPoolSlot{{
				Items: []DropPoolReward{{Type: RewardTypeItem, ID: "general_exp_small", Amount: 1, Weight: 10000}},
			}},
		},
	}
	poolData, err := json.Marshal(pools)
	if err != nil {
		t.Fatalf("marshal pools: %v", err)
	}
	if err := os.WriteFile(poolPath, poolData, 0o644); err != nil {
		t.Fatalf("write pools: %v", err)
	}
	if err := LoadDropPoolsConfig(poolPath); err != nil {
		t.Fatalf("LoadDropPoolsConfig failed: %v", err)
	}

	for tier, tierCfg := range npcCfg.Tiers {
		tierCfg.DropPoolID = "test_npc_exp_pack"
		npcCfg.Tiers[tier] = tierCfg
	}
	npcPath := filepath.Join(t.TempDir(), "npc.json")
	npcData, err := json.Marshal(npcCfg)
	if err != nil {
		t.Fatalf("marshal npc config: %v", err)
	}
	if err := os.WriteFile(npcPath, npcData, 0o644); err != nil {
		t.Fatalf("write npc config: %v", err)
	}
	if err := LoadNpcConfig(npcPath); err != nil {
		t.Fatalf("LoadNpcConfig custom failed: %v", err)
	}

	svc := NewService()
	repo := svc.repo.(*MemoryRepository)
	now := time.Now()
	account := Account{ID: "acc_attack_npc_drop", Username: "attack_npc_drop", PasswordHash: "x", CreatedAt: now}
	if err := repo.CreateAccount(account); err != nil {
		t.Fatalf("create account: %v", err)
	}
	state := newPlayerState("player_attack_npc_drop", "NpcDrop", "wei", "caocao", now)
	state.Army = []ArmyUnit{{UnitType: "weiInfantry", Amount: 10}}
	state.NpcState = &NpcState{
		Cities: []NpcCity{{
			ID:                "npc_drop_1",
			Name:              "掉落城",
			Faction:           "wei",
			Tier:              "small",
			Resources:         map[string]int{"wood": 0},
			StorageCapacity:   map[string]int{},
			ProductionPerHour: map[string]int{},
			Army:              []ArmyUnit{},
			ResourceSettledAt: now.UTC().Format(resourceDateLayout),
			ArmySettledAt:     now.UTC().Format(resourceDateLayout),
			GeneratedAt:       now.UTC().Format(resourceDateLayout),
		}},
		LastRefreshedAt: now.UTC().Format(resourceDateLayout),
	}
	if err := repo.CreatePlayer(account.ID, state, now); err != nil {
		t.Fatalf("create player: %v", err)
	}

	result, err := svc.AttackNpc(AttackNpcRequest{
		PlayerID: state.Player.ID,
		NpcID:    "npc_drop_1",
		Mode:     "attack",
		Units:    map[string]int{"weiInfantry": 5},
	})
	if err != nil {
		t.Fatalf("AttackNpc failed: %v", err)
	}
	if len(result.BattleReport.Drops) != 1 || result.BattleReport.Drops[0].ItemID != "general_exp_small" {
		t.Fatalf("expected general exp pack drop in report, got %+v", result.BattleReport.Drops)
	}
	if len(result.BattleReport.GrantedRewards) == 0 {
		t.Fatalf("expected granted rewards to include item drop, got %+v", result.BattleReport.GrantedRewards)
	}
	stored, err := repo.GetState(state.Player.ID)
	if err != nil {
		t.Fatalf("get state: %v", err)
	}
	if got := inventoryItemAmount(&stored, "general_exp_small"); got != 1 {
		t.Fatalf("expected dropped item in inventory, got %d", got)
	}
}

func TestAttackNpcOnlyAppliesSelectedGeneral(t *testing.T) {
	setTestCombatUnitsConfig(t)
	setTestGeneralsConfig(t, GeneralsConfig{
		Enabled: true,
		Heroes: map[string]GeneralHeroConfig{
			"test_general": {
				ID:      "test_general",
				Name:    "测试将领",
				Faction: "wei",
				Enabled: true,
				Buffs:   map[string]float64{StatAttackBonus: 1},
				SpecialTrait: GeneralTraitConfig{
					TraitID: "huogong", TraitType: coregeneral.TraitTypeSpecial, Enabled: true,
					Scope: "enemy_army", AllowedSides: []string{"attacker"},
					Params: map[string]float64{"damagePercent": 0.5, "triggerChance": 1},
				},
			},
		},
	})

	svc := NewService()
	repo := svc.repo.(*MemoryRepository)
	now := time.Now()
	account := Account{ID: "acc_attack_general_rule", Username: "attack_general_rule", PasswordHash: "x", CreatedAt: now}
	if err := repo.CreateAccount(account); err != nil {
		t.Fatalf("create account: %v", err)
	}
	state := newPlayerState("player_attack_general_rule", "GeneralRule", "wei", "test_general", now)
	state.Army = []ArmyUnit{{UnitType: "weiInfantry", Amount: 500}}
	withoutGeneralCity := testNpcCity("npc_without_general", now)
	withoutGeneralCity.Army = []ArmyUnit{{UnitType: "weiInfantry", Amount: 1000}}
	withoutGeneralCity.MaxArmy = []ArmyUnit{{UnitType: "weiInfantry", Amount: 1000}}
	withGeneralCity := testNpcCity("npc_with_general", now)
	withGeneralCity.Army = []ArmyUnit{{UnitType: "weiInfantry", Amount: 1000}}
	withGeneralCity.MaxArmy = []ArmyUnit{{UnitType: "weiInfantry", Amount: 1000}}
	state.NpcState = &NpcState{
		Cities:          []NpcCity{withoutGeneralCity, withGeneralCity},
		LastRefreshedAt: now.UTC().Format(resourceDateLayout),
	}
	if err := repo.CreatePlayer(account.ID, state, now); err != nil {
		t.Fatalf("create player: %v", err)
	}

	without, err := svc.AttackNpc(AttackNpcRequest{
		PlayerID: state.Player.ID,
		NpcID:    "npc_without_general",
		Mode:     "attack",
		Units:    map[string]int{"weiInfantry": 100},
	})
	if err != nil {
		t.Fatalf("AttackNpc without general failed: %v", err)
	}
	storedWithout, err := repo.GetState(state.Player.ID)
	if err != nil {
		t.Fatalf("get state without general: %v", err)
	}
	if got := pvpTestGeneralExp(storedWithout, "test_general"); got != 0 {
		t.Fatalf("expected no general exp without selected general, got %d", got)
	}
	if without.BattleReport.GeneralExpGained != 0 || len(without.BattleReport.PvpAttackerGenerals) != 0 {
		t.Fatalf("expected report without general participation, got %+v", without.BattleReport)
	}
	if _, triggered := without.BattleReport.TraitOutcomes["huogong"]; triggered || len(without.BattleReport.TraitTriggered) != 0 {
		t.Fatalf("expected no formal trait without selected general, got triggered=%+v outcomes=%+v", without.BattleReport.TraitTriggered, without.BattleReport.TraitOutcomes)
	}
	if without.BattleReport.Detail == nil || len(without.BattleReport.Detail.Traits) != 0 {
		t.Fatalf("expected no standard trait timeline without selected general, detail=%+v", without.BattleReport.Detail)
	}

	withGeneral, err := svc.AttackNpc(AttackNpcRequest{
		PlayerID:   state.Player.ID,
		NpcID:      "npc_with_general",
		Mode:       "attack",
		Units:      map[string]int{"weiInfantry": 100},
		GeneralIDs: []string{"test_general"},
	})
	if err != nil {
		t.Fatalf("AttackNpc with general failed: %v", err)
	}
	storedWith, err := repo.GetState(state.Player.ID)
	if err != nil {
		t.Fatalf("get state with general: %v", err)
	}
	if got := pvpTestGeneralExp(storedWith, "test_general"); got <= 0 {
		t.Fatalf("expected selected general to gain exp, got %d", got)
	}
	if withGeneral.BattleReport.GeneralExpGained <= 0 || len(withGeneral.BattleReport.PvpAttackerGenerals) != 1 {
		t.Fatalf("expected report to include selected general and exp, got %+v", withGeneral.BattleReport)
	}
	if outcome, triggered := withGeneral.BattleReport.TraitOutcomes["huogong"]; !triggered || outcome.TraitID != "huogong" {
		t.Fatalf("expected selected general formal trait to trigger, outcomes=%+v", withGeneral.BattleReport.TraitOutcomes)
	}
	if withGeneral.BattleReport.PlayerPower <= without.BattleReport.PlayerPower {
		t.Fatalf("expected selected general to increase attack power, without=%d with=%d", without.BattleReport.PlayerPower, withGeneral.BattleReport.PlayerPower)
	}
}

func testNpcCity(id string, now time.Time) NpcCity {
	return NpcCity{
		ID:                id,
		Name:              id,
		Faction:           "wei",
		Resources:         map[string]int{"wood": 0},
		StorageCapacity:   map[string]int{},
		ProductionPerHour: map[string]int{},
		Army:              []ArmyUnit{{UnitType: "weiInfantry", Amount: 10}},
		MaxArmy:           []ArmyUnit{{UnitType: "weiInfantry", Amount: 10}},
		ResourceSettledAt: now.UTC().Format(resourceDateLayout),
		ArmySettledAt:     now.UTC().Format(resourceDateLayout),
		GeneratedAt:       now.UTC().Format(resourceDateLayout),
	}
}

func waitForNpcSweepTask(t *testing.T, svc *Service, playerID string, taskID string) NpcSweepTask {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		task, err := svc.GetNpcSweepTask(playerID, taskID)
		if err != nil {
			t.Fatalf("GetNpcSweepTask failed: %v", err)
		}
		if task.Status == SweepTaskStatusCompleted || task.Status == SweepTaskStatusFailed {
			return task
		}
		time.Sleep(10 * time.Millisecond)
	}
	task, err := svc.GetNpcSweepTask(playerID, taskID)
	if err != nil {
		t.Fatalf("GetNpcSweepTask after timeout failed: %v", err)
	}
	t.Fatalf("sweep task did not finish before timeout: %+v", task)
	return task
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

func TestAddGoldUsesGrantRewardsLedgerAndEvent(t *testing.T) {
	svc := NewService()
	repo := svc.repo.(*MemoryRepository)
	now := time.Now()
	account := Account{ID: "acc_city_gold_grant", Username: "city_gold_grant", PasswordHash: "x", CreatedAt: now}
	if err := repo.CreateAccount(account); err != nil {
		t.Fatalf("create account: %v", err)
	}
	state := newPlayerState("player_city_gold_grant", "CityGoldGrant", "wei", "caocao", now)
	if err := repo.CreatePlayer(account.ID, state, now); err != nil {
		t.Fatalf("create player: %v", err)
	}

	events := []GameEvent{}
	svc.SubscribeEvent(EventRewardGranted, func(event GameEvent) {
		events = append(events, event)
	})

	result, err := svc.AddGold(state.Player.ID, 50, "gm_add_city_gold")
	if err != nil {
		t.Fatalf("AddGold failed: %v", err)
	}
	if int(result.CityGold) != 50 {
		t.Fatalf("expected city gold 50, got %d", result.CityGold)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 reward event, got %+v", events)
	}
	entries, err := svc.ListGoldLedger(GoldLedgerFilter{PlayerID: state.Player.ID, RefType: LedgerRefAdminAdjust})
	if err != nil {
		t.Fatalf("ListGoldLedger failed: %v", err)
	}
	if len(entries) != 1 || entries[0].Amount != 50 || entries[0].BalanceAfter != 50 {
		t.Fatalf("unexpected city gold ledger: %+v", entries)
	}
}

func TestGrantRewardsRejectsAccountGoldWithoutAccountBeforeStateChange(t *testing.T) {
	svc := NewService()
	repo := svc.repo.(*MemoryRepository)
	now := time.Now()
	account := Account{ID: "acc_gold_reward_guard", Username: "gold_reward_guard", PasswordHash: "x", CreatedAt: now}
	if err := repo.CreateAccount(account); err != nil {
		t.Fatalf("create account: %v", err)
	}
	state := newPlayerState("player_gold_reward_guard", "GoldRewardGuard", "wei", "caocao", now)
	if err := repo.CreatePlayer(account.ID, state, now); err != nil {
		t.Fatalf("create player: %v", err)
	}

	_, err := svc.GrantRewards(state.Player.ID, []Reward{
		{Type: RewardTypeCityGold, ID: RewardTypeCityGold, Amount: 50},
		{Type: RewardTypeGold, ID: RewardTypeGold, Amount: 5},
	}, RewardGrantContext{RefType: "test_guard"})
	if !errors.Is(err, ErrAccountNotFound) {
		t.Fatalf("expected ErrAccountNotFound, got %v", err)
	}
	next, err := repo.GetState(state.Player.ID)
	if err != nil {
		t.Fatalf("GetState failed: %v", err)
	}
	if int(next.CityGold) != 0 {
		t.Fatalf("expected no state change, got city gold %d", next.CityGold)
	}
}

func TestCoreRewardTypesAreRegistered(t *testing.T) {
	required := []string{
		RewardTypeResource,
		RewardTypeCityGold,
		RewardTypeGold,
		RewardTypeItem,
		RewardTypeUnit,
		RewardTypeGeneralExp,
		RewardTypeBuff,
	}
	for _, rewardType := range required {
		if _, ok := GetRewardTypeDefinition(rewardType); !ok {
			t.Fatalf("expected reward type %s to be registered", rewardType)
		}
	}
}

func TestRegisterResourceTypeAllowsNewResourceRewards(t *testing.T) {
	resourceType := "crystal_test_resource"
	_ = RegisterResourceType(ResourceTypeDefinition{Type: resourceType, Description: "测试晶石"})

	state := newPlayerState("player_resource_registry", "ResourceRegistry", "wei", "caocao", time.Now())
	state.Resources.Capacity[resourceType] = 100
	result, err := ApplyRewardsToState(&state, []Reward{
		{Type: RewardTypeResource, ID: resourceType, Amount: 25},
	}, time.Now())
	if err != nil {
		t.Fatalf("ApplyRewardsToState failed: %v", err)
	}
	if state.Resources.Items[resourceType] != 25 || result.Granted[resourceType] != 25 {
		t.Fatalf("expected registered resource reward to apply, state=%+v result=%+v", state.Resources.Items, result.Granted)
	}
}

func TestGrantRewardsRejectsUnregisteredRewardType(t *testing.T) {
	svc := NewService()
	repo := svc.repo.(*MemoryRepository)
	now := time.Now()
	account := Account{ID: "acc_unknown_reward", Username: "unknown_reward", PasswordHash: "x", CreatedAt: now}
	if err := repo.CreateAccount(account); err != nil {
		t.Fatalf("create account: %v", err)
	}
	state := newPlayerState("player_unknown_reward", "UnknownReward", "wei", "caocao", now)
	if err := repo.CreatePlayer(account.ID, state, now); err != nil {
		t.Fatalf("create player: %v", err)
	}

	_, err := svc.GrantRewards(state.Player.ID, []Reward{{Type: "unknown_asset", ID: "x", Amount: 1}}, RewardGrantContext{})
	if !errors.Is(err, ErrMailInvalidAttachment) {
		t.Fatalf("expected ErrMailInvalidAttachment, got %v", err)
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

	events := []GameEvent{}
	svc.SubscribeEvent(EventCurrencyChanged, func(event GameEvent) {
		events = append(events, event)
	})

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
	if len(events) != 1 || events[0].Type != EventCurrencyChanged || events[0].RefType != LedgerRefExchange {
		t.Fatalf("expected currency changed event, got %+v", events)
	}
	if events[0].RefID == "" || events[0].RefID != entries[0].RefID {
		t.Fatalf("expected currency event to share exchange refId, event=%+v entries=%+v", events[0], entries)
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

	events := []GameEvent{}
	svc.SubscribeEvent(EventCurrencyChanged, func(event GameEvent) {
		events = append(events, event)
	})

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
	if len(events) != 1 || events[0].Type != EventCurrencyChanged || events[0].RefType != LedgerRefExchange {
		t.Fatalf("expected currency changed event, got %+v", events)
	}
	if events[0].RefID == "" || events[0].RefID != entries[0].RefID {
		t.Fatalf("expected currency event to share exchange refId, event=%+v entries=%+v", events[0], entries)
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

func TestRedeemMiniGameRewardPartialAddsFactionArmy(t *testing.T) {
	svc := NewService()
	if err := svc.SetUnitsDir(filepath.Join("..", "..", "..", "config", "units")); err != nil {
		t.Fatalf("load units: %v", err)
	}
	repo := svc.repo.(*MemoryRepository)
	now := time.Now()
	account := Account{ID: "acc_fishing_redeem", Username: "fishing_redeem", PasswordHash: "x", CreatedAt: now}
	if err := repo.CreateAccount(account); err != nil {
		t.Fatalf("create account: %v", err)
	}
	state := newPlayerState("player_fishing_redeem", "Fisher", "wei", "caocao", now)
	if err := repo.CreatePlayer(account.ID, state, now); err != nil {
		t.Fatalf("create player: %v", err)
	}
	record, err := svc.SaveMiniGameRecord(state.Player.ID, "fishing", "金龙鱼", "rare", "骁骑营", 5000, "", 0)
	if err != nil {
		t.Fatalf("save minigame record: %v", err)
	}

	redeemEvents := []GameEvent{}
	rewardEvents := []GameEvent{}
	svc.SubscribeEvent(EventMiniGameRedeemed, func(event GameEvent) {
		redeemEvents = append(redeemEvents, event)
	})
	svc.SubscribeEvent(EventRewardGranted, func(event GameEvent) {
		rewardEvents = append(rewardEvents, event)
	})

	result, err := svc.RedeemMiniGameReward(state.Player.ID, record.ID, 1200)
	if err != nil {
		t.Fatalf("RedeemMiniGameReward failed: %v", err)
	}
	if result.Record.RemainingAmount != 3800 {
		t.Fatalf("expected remaining 3800, got %d", result.Record.RemainingAmount)
	}
	if result.RedeemedUnitID != "qiQiYing" {
		t.Fatalf("expected wei unit qiQiYing, got %s", result.RedeemedUnitID)
	}
	if len(result.GrantedRewards) != 1 || result.GrantedRewards[0].Type != RewardTypeUnit || result.GrantedRewards[0].Amount != 1200 {
		t.Fatalf("unexpected granted rewards: %+v", result.GrantedRewards)
	}
	if len(redeemEvents) != 1 {
		t.Fatalf("expected one minigame redeemed event, got %+v", redeemEvents)
	}
	if len(rewardEvents) != 1 {
		t.Fatalf("expected one reward granted event, got %+v", rewardEvents)
	}
	found := false
	for _, unit := range result.Army {
		if unit.UnitType == "qiQiYing" && unit.Amount == 1200 {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected redeemed unit in army, got %+v", result.Army)
	}
}

func TestGamblingMiniGameRecordCreatesRedeemableStock(t *testing.T) {
	svc := NewService()
	repo := svc.repo.(*MemoryRepository)
	now := time.Now()
	account := Account{ID: "acc_gambling_stock", Username: "gambling_stock", PasswordHash: "x", CreatedAt: now}
	if err := repo.CreateAccount(account); err != nil {
		t.Fatalf("create account: %v", err)
	}
	state := newPlayerState("player_gambling_stock", "Gambler", "wei", "caocao", now)
	if err := repo.CreatePlayer(account.ID, state, now); err != nil {
		t.Fatalf("create player: %v", err)
	}

	record, err := svc.SaveMiniGameRecord(state.Player.ID, "gambling", "猜18点 赢 ×150", "legendary", "骁骑营", 150000, "骁骑营", 1000)
	if err != nil {
		t.Fatalf("SaveMiniGameRecord failed: %v", err)
	}
	if record.RemainingAmount != 150000 {
		t.Fatalf("expected gambling stock 150000, got %d", record.RemainingAmount)
	}
}

func TestRedeemGamblingMiniGameRewardAddsFactionArmy(t *testing.T) {
	svc := NewService()
	if err := svc.SetUnitsDir(filepath.Join("..", "..", "..", "config", "units")); err != nil {
		t.Fatalf("load units: %v", err)
	}
	repo := svc.repo.(*MemoryRepository)
	now := time.Now()
	account := Account{ID: "acc_gambling_redeem", Username: "gambling_redeem", PasswordHash: "x", CreatedAt: now}
	if err := repo.CreateAccount(account); err != nil {
		t.Fatalf("create account: %v", err)
	}
	state := newPlayerState("player_gambling_redeem", "Gambler", "wei", "caocao", now)
	if err := repo.CreatePlayer(account.ID, state, now); err != nil {
		t.Fatalf("create player: %v", err)
	}
	record, err := svc.SaveMiniGameRecord(state.Player.ID, "gambling", "猜18点 赢 ×150", "legendary", "骁骑营", 150000, "骁骑营", 1000)
	if err != nil {
		t.Fatalf("save gambling record: %v", err)
	}

	result, err := svc.RedeemMiniGameReward(state.Player.ID, record.ID, 50000)
	if err != nil {
		t.Fatalf("RedeemMiniGameReward failed: %v", err)
	}
	if result.Record.GameType != "gambling" || result.Record.RemainingAmount != 100000 {
		t.Fatalf("unexpected redeemed gambling record: %+v", result.Record)
	}
	if result.RedeemedTarget != "army" || result.RedeemedUnitID != "qiQiYing" || result.RedeemedAmount != 50000 {
		t.Fatalf("unexpected gambling redeem result: %+v", result)
	}
	found := false
	for _, unit := range result.Army {
		if unit.UnitType == "qiQiYing" && unit.Amount == 50000 {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected redeemed gambling unit in army, got %+v", result.Army)
	}
}

func TestResolveGamblingRoundDeductsBetArmyAndWritesRecord(t *testing.T) {
	svc := NewService()
	if err := svc.SetUnitsDir(filepath.Join("..", "..", "..", "config", "units")); err != nil {
		t.Fatalf("load units: %v", err)
	}
	repo := svc.repo.(*MemoryRepository)
	now := time.Now()
	account := Account{ID: "acc_gambling_resolve", Username: "gambling_resolve", PasswordHash: "x", CreatedAt: now}
	if err := repo.CreateAccount(account); err != nil {
		t.Fatalf("create account: %v", err)
	}
	state := newPlayerState("player_gambling_resolve", "Gambler", "wei", "caocao", now)
	state.Army = []ArmyUnit{{UnitType: "qiQiYing", Amount: 100000}}
	if err := repo.CreatePlayer(account.ID, state, now); err != nil {
		t.Fatalf("create player: %v", err)
	}

	result, err := svc.ResolveGamblingRound(state.Player.ID, "qiQiYing", 10000, "big", 10)
	if err != nil {
		t.Fatalf("ResolveGamblingRound failed: %v", err)
	}
	if result.BetUnitID != "qiQiYing" || result.BetUnit != "骁骑营" || result.BetAmount != 10000 {
		t.Fatalf("unexpected gambling round bet: %+v", result)
	}
	if result.DiceTotal < 3 || result.DiceTotal > 18 || len(result.DiceValues) != 3 {
		t.Fatalf("unexpected dice result: %+v", result)
	}
	if armySliceToMap(result.Army)["qiQiYing"] != 90000 {
		t.Fatalf("expected bet army deducted to 90000, got %+v", result.Army)
	}
	records, _, err := repo.ListMiniGameRecords(state.Player.ID, "gambling", 10, 0)
	if err != nil {
		t.Fatalf("list gambling records: %v", err)
	}
	if len(records) != 1 || records[0].ID != result.Record.ID || records[0].BetAmount != 10000 {
		t.Fatalf("expected gambling record persisted, got %+v", records)
	}
	if result.Won && (records[0].RemainingAmount <= 0 || records[0].RewardUnit != "骁骑营") {
		t.Fatalf("winning gambling record should be redeemable, got %+v", records[0])
	}
	if !result.Won && records[0].RemainingAmount != 0 {
		t.Fatalf("losing gambling record should not be redeemable, got %+v", records[0])
	}
}

func TestResolveGamblingRoundRejectsInsufficientArmyWithoutRecord(t *testing.T) {
	svc := NewService()
	if err := svc.SetUnitsDir(filepath.Join("..", "..", "..", "config", "units")); err != nil {
		t.Fatalf("load units: %v", err)
	}
	repo := svc.repo.(*MemoryRepository)
	now := time.Now()
	account := Account{ID: "acc_gambling_short", Username: "gambling_short", PasswordHash: "x", CreatedAt: now}
	if err := repo.CreateAccount(account); err != nil {
		t.Fatalf("create account: %v", err)
	}
	state := newPlayerState("player_gambling_short", "Gambler", "wei", "caocao", now)
	state.Army = []ArmyUnit{{UnitType: "qiQiYing", Amount: 999}}
	if err := repo.CreatePlayer(account.ID, state, now); err != nil {
		t.Fatalf("create player: %v", err)
	}

	if _, err := svc.ResolveGamblingRound(state.Player.ID, "qiQiYing", 1000, "big", 10); !errors.Is(err, ErrInsufficientArmy) {
		t.Fatalf("expected ErrInsufficientArmy, got %v", err)
	}
	records, _, err := repo.ListMiniGameRecords(state.Player.ID, "gambling", 10, 0)
	if err != nil {
		t.Fatalf("list gambling records: %v", err)
	}
	if len(records) != 0 {
		t.Fatalf("insufficient bet should not write record, got %+v", records)
	}
	stored, err := repo.GetState(state.Player.ID)
	if err != nil {
		t.Fatalf("GetState failed: %v", err)
	}
	if armySliceToMap(stored.Army)["qiQiYing"] != 999 {
		t.Fatalf("insufficient bet should not deduct army, got %+v", stored.Army)
	}
}

func testSlotConfigForRules() SlotConfig {
	return SlotConfig{
		MinLineBet:           1000,
		LineCount:            5,
		MaxFreeSpinsPerRound: 20,
		Symbols: []SlotSymbol{
			{ID: "bronze_charm", Name: "玄铜符", Rarity: "common", Type: "normal", Weight: 30, Multiplier: 3},
			{ID: "silver_charm", Name: "白银符", Rarity: "rare", Type: "normal", Weight: 22, Multiplier: 6},
			{ID: "gold_charm", Name: "赤金符", Rarity: "rare", Type: "normal", Weight: 16, Multiplier: 12},
			{ID: "jade_seal", Name: "玉玺", Rarity: "epic", Type: "normal", Weight: 10, Multiplier: 30},
			{ID: "tiger_tally", Name: "虎符", Rarity: "epic", Type: "normal", Weight: 6, Multiplier: 80},
			{ID: "heaven_order", Name: "天命令", Rarity: "legendary", Type: "normal", Weight: 2, Multiplier: 250},
			{ID: "wild", Name: "天机令", Rarity: "epic", Type: "wild", Weight: 5, Multiplier: 250},
			{ID: "scatter", Name: "星陨", Rarity: "epic", Type: "scatter", Weight: 4, FreeSpins: 5, RetriggerFreeSpins: 3},
			{ID: "bonus", Name: "宝匣", Rarity: "rare", Type: "bonus", Weight: 5, BonusMultipliers: []SlotBonusMultiplier{
				{Multiplier: 5, Weight: 50},
				{Multiplier: 10, Weight: 30},
				{Multiplier: 20, Weight: 15},
				{Multiplier: 50, Weight: 5},
			}},
		},
	}
}

func testSlotGrid(cfg SlotConfig, rows [3][3]string) [3][3]SlotSymbol {
	symbols := slotSymbolsByID(cfg)
	var grid [3][3]SlotSymbol
	for row := range rows {
		for col := range rows[row] {
			grid[row][col] = symbols[rows[row][col]]
		}
	}
	return grid
}

func setTestSlotGridSequence(t *testing.T, cfg SlotConfig, grids ...[3][3]string) {
	t.Helper()
	if err := SetSlotConfig(cfg); err != nil {
		t.Fatalf("set slot config: %v", err)
	}
	previousGridRoller := slotGridRoller
	previousBonusRoller := slotBonusRoller
	index := 0
	slotGridRoller = func(cfg SlotConfig) ([3][3]SlotSymbol, error) {
		if len(grids) == 0 {
			return testSlotGrid(cfg, [3][3]string{
				{"bronze_charm", "silver_charm", "gold_charm"},
				{"silver_charm", "gold_charm", "bronze_charm"},
				{"jade_seal", "bronze_charm", "silver_charm"},
			}), nil
		}
		if index >= len(grids) {
			return testSlotGrid(cfg, grids[len(grids)-1]), nil
		}
		grid := testSlotGrid(cfg, grids[index])
		index++
		return grid, nil
	}
	slotBonusRoller = func(symbol SlotSymbol) (int, error) {
		return 20, nil
	}
	t.Cleanup(func() {
		slotGridRoller = previousGridRoller
		slotBonusRoller = previousBonusRoller
		_ = SetSlotConfig(defaultSlotConfig)
	})
}

func TestResolveSlotRoundDeductsBetArmyAndWritesWinningStock(t *testing.T) {
	svc := NewService()
	if err := svc.SetUnitsDir(filepath.Join("..", "..", "..", "config", "units")); err != nil {
		t.Fatalf("load units: %v", err)
	}
	cfg := testSlotConfigForRules()
	setTestSlotGridSequence(t, cfg, [3][3]string{
		{"bronze_charm", "silver_charm", "tiger_tally"},
		{"gold_charm", "gold_charm", "gold_charm"},
		{"jade_seal", "bronze_charm", "silver_charm"},
	})
	repo := svc.repo.(*MemoryRepository)
	now := time.Now()
	account := Account{ID: "acc_slot_resolve", Username: "slot_resolve", PasswordHash: "x", CreatedAt: now}
	if err := repo.CreateAccount(account); err != nil {
		t.Fatalf("create account: %v", err)
	}
	state := newPlayerState("player_slot_resolve", "Slotter", "wei", "caocao", now)
	state.Army = []ArmyUnit{{UnitType: "qiQiYing", Amount: 1000000}}
	if err := repo.CreatePlayer(account.ID, state, now); err != nil {
		t.Fatalf("create player: %v", err)
	}

	result, err := svc.ResolveSlotRound(state.Player.ID, "qiQiYing", 10000)
	if err != nil {
		t.Fatalf("ResolveSlotRound failed: %v", err)
	}
	if !result.Won || result.LineBet != 10000 || result.TotalBet != 10000 || result.WinAmount != 120000 || len(result.WinningLines) != 1 {
		t.Fatalf("expected winning slot result, got %+v", result)
	}
	if result.WinningLines[0].LineID != "middle" || result.WinningLines[0].Multiplier != 12 {
		t.Fatalf("expected middle gold line, got %+v", result.WinningLines)
	}
	if armySliceToMap(result.Army)["qiQiYing"] != 990000 {
		t.Fatalf("expected single spin bet army deducted to 990000, got %+v", result.Army)
	}
	records, _, err := repo.ListMiniGameRecords(state.Player.ID, "slot", 10, 0)
	if err != nil {
		t.Fatalf("list slot records: %v", err)
	}
	if len(records) != 1 || records[0].GameType != "slot" || records[0].BetAmount != 10000 || records[0].RemainingAmount != 120000 || records[0].RewardUnit != "骁骑营" {
		t.Fatalf("expected winning slot record persisted, got %+v", records)
	}
}

func TestResolveSlotRoundWritesLosingRecordWithoutStock(t *testing.T) {
	svc := NewService()
	if err := svc.SetUnitsDir(filepath.Join("..", "..", "..", "config", "units")); err != nil {
		t.Fatalf("load units: %v", err)
	}
	cfg := testSlotConfigForRules()
	setTestSlotGridSequence(t, cfg, [3][3]string{
		{"bronze_charm", "silver_charm", "gold_charm"},
		{"silver_charm", "gold_charm", "bronze_charm"},
		{"jade_seal", "bronze_charm", "silver_charm"},
	})
	repo := svc.repo.(*MemoryRepository)
	now := time.Now()
	account := Account{ID: "acc_slot_loss", Username: "slot_loss", PasswordHash: "x", CreatedAt: now}
	if err := repo.CreateAccount(account); err != nil {
		t.Fatalf("create account: %v", err)
	}
	state := newPlayerState("player_slot_loss", "Slotter", "wei", "caocao", now)
	state.Army = []ArmyUnit{{UnitType: "qiQiYing", Amount: 200000}}
	if err := repo.CreatePlayer(account.ID, state, now); err != nil {
		t.Fatalf("create player: %v", err)
	}

	result, err := svc.ResolveSlotRound(state.Player.ID, "qiQiYing", 1000)
	if err != nil {
		t.Fatalf("ResolveSlotRound failed: %v", err)
	}
	if result.Won || result.WinAmount != 0 || result.Record.RemainingAmount != 0 {
		t.Fatalf("expected losing slot result, got %+v", result)
	}
	records, _, err := repo.ListMiniGameRecords(state.Player.ID, "slot", 10, 0)
	if err != nil {
		t.Fatalf("list slot records: %v", err)
	}
	if len(records) != 1 || records[0].ResultName != "未中奖" || records[0].RemainingAmount != 0 || records[0].RewardUnit != "" {
		t.Fatalf("expected losing slot record without stock, got %+v", records)
	}
}

func TestResolveSlotRoundWildSubstitutesNormalSymbols(t *testing.T) {
	svc := NewService()
	if err := svc.SetUnitsDir(filepath.Join("..", "..", "..", "config", "units")); err != nil {
		t.Fatalf("load units: %v", err)
	}
	cfg := testSlotConfigForRules()
	setTestSlotGridSequence(t, cfg, [3][3]string{
		{"bronze_charm", "silver_charm", "tiger_tally"},
		{"wild", "wild", "gold_charm"},
		{"jade_seal", "bronze_charm", "silver_charm"},
	})
	repo := svc.repo.(*MemoryRepository)
	now := time.Now()
	account := Account{ID: "acc_slot_wild", Username: "slot_wild", PasswordHash: "x", CreatedAt: now}
	if err := repo.CreateAccount(account); err != nil {
		t.Fatalf("create account: %v", err)
	}
	state := newPlayerState("player_slot_wild", "Slotter", "wei", "caocao", now)
	state.Army = []ArmyUnit{{UnitType: "qiQiYing", Amount: 200000}}
	if err := repo.CreatePlayer(account.ID, state, now); err != nil {
		t.Fatalf("create player: %v", err)
	}

	result, err := svc.ResolveSlotRound(state.Player.ID, "qiQiYing", 1000)
	if err != nil {
		t.Fatalf("ResolveSlotRound failed: %v", err)
	}
	if len(result.WinningLines) != 1 || result.WinningLines[0].Symbol != "gold_charm" || result.WinAmount != 12000 {
		t.Fatalf("expected wild to pay gold line, got %+v", result)
	}
}

func TestResolveSlotRoundWildTriplePaysHeavenOrder(t *testing.T) {
	svc := NewService()
	if err := svc.SetUnitsDir(filepath.Join("..", "..", "..", "config", "units")); err != nil {
		t.Fatalf("load units: %v", err)
	}
	cfg := testSlotConfigForRules()
	setTestSlotGridSequence(t, cfg, [3][3]string{
		{"bronze_charm", "silver_charm", "gold_charm"},
		{"wild", "wild", "wild"},
		{"jade_seal", "tiger_tally", "heaven_order"},
	})
	repo := svc.repo.(*MemoryRepository)
	now := time.Now()
	account := Account{ID: "acc_slot_wild_triple", Username: "slot_wild_triple", PasswordHash: "x", CreatedAt: now}
	if err := repo.CreateAccount(account); err != nil {
		t.Fatalf("create account: %v", err)
	}
	state := newPlayerState("player_slot_wild_triple", "Slotter", "wei", "caocao", now)
	state.Army = []ArmyUnit{{UnitType: "qiQiYing", Amount: 200000}}
	if err := repo.CreatePlayer(account.ID, state, now); err != nil {
		t.Fatalf("create player: %v", err)
	}

	result, err := svc.ResolveSlotRound(state.Player.ID, "qiQiYing", 1000)
	if err != nil {
		t.Fatalf("ResolveSlotRound failed: %v", err)
	}
	if len(result.WinningLines) != 1 || result.WinningLines[0].Symbol != "heaven_order" || result.WinAmount != 250000 {
		t.Fatalf("expected wild triple to pay heaven order, got %+v", result)
	}
}

func TestResolveSlotRoundWildCompletesScatterTrigger(t *testing.T) {
	svc := NewService()
	if err := svc.SetUnitsDir(filepath.Join("..", "..", "..", "config", "units")); err != nil {
		t.Fatalf("load units: %v", err)
	}
	cfg := testSlotConfigForRules()
	losing := [3][3]string{
		{"bronze_charm", "silver_charm", "gold_charm"},
		{"silver_charm", "gold_charm", "bronze_charm"},
		{"jade_seal", "bronze_charm", "silver_charm"},
	}
	setTestSlotGridSequence(t, cfg,
		[3][3]string{
			{"wild", "scatter", "wild"},
			{"bronze_charm", "jade_seal", "silver_charm"},
			{"gold_charm", "jade_seal", "tiger_tally"},
		},
		losing, losing, losing, losing, losing,
	)
	repo := svc.repo.(*MemoryRepository)
	now := time.Now()
	account := Account{ID: "acc_slot_wild_special", Username: "slot_wild_special", PasswordHash: "x", CreatedAt: now}
	if err := repo.CreateAccount(account); err != nil {
		t.Fatalf("create account: %v", err)
	}
	state := newPlayerState("player_slot_wild_special", "Slotter", "wei", "caocao", now)
	state.Army = []ArmyUnit{{UnitType: "qiQiYing", Amount: 200000}}
	if err := repo.CreatePlayer(account.ID, state, now); err != nil {
		t.Fatalf("create player: %v", err)
	}

	result, err := svc.ResolveSlotRound(state.Player.ID, "qiQiYing", 1000)
	if err != nil {
		t.Fatalf("ResolveSlotRound failed: %v", err)
	}
	if len(result.FreeSpins) != 5 || result.FreeSpins[0].ScatterCount != 0 {
		t.Fatalf("expected wild to complete scatter trigger and play five losing free spins, got %+v", result)
	}
	if result.Won || result.WinAmount != 0 {
		t.Fatalf("expected scatter trigger without free spin wins to keep zero stock reward, got %+v", result)
	}
}

func TestResolveSlotRoundScatterTriggersFreeSpinsWithoutExtraDeduction(t *testing.T) {
	svc := NewService()
	if err := svc.SetUnitsDir(filepath.Join("..", "..", "..", "config", "units")); err != nil {
		t.Fatalf("load units: %v", err)
	}
	cfg := testSlotConfigForRules()
	losing := [3][3]string{
		{"bronze_charm", "silver_charm", "gold_charm"},
		{"silver_charm", "gold_charm", "bronze_charm"},
		{"jade_seal", "bronze_charm", "silver_charm"},
	}
	setTestSlotGridSequence(t, cfg,
		[3][3]string{
			{"scatter", "silver_charm", "gold_charm"},
			{"silver_charm", "gold_charm", "bronze_charm"},
			{"jade_seal", "scatter", "scatter"},
		},
		[3][3]string{
			{"bronze_charm", "silver_charm", "gold_charm"},
			{"tiger_tally", "tiger_tally", "tiger_tally"},
			{"jade_seal", "bronze_charm", "silver_charm"},
		},
		losing, losing, losing, losing,
	)
	repo := svc.repo.(*MemoryRepository)
	now := time.Now()
	account := Account{ID: "acc_slot_scatter", Username: "slot_scatter", PasswordHash: "x", CreatedAt: now}
	if err := repo.CreateAccount(account); err != nil {
		t.Fatalf("create account: %v", err)
	}
	state := newPlayerState("player_slot_scatter", "Slotter", "wei", "caocao", now)
	state.Army = []ArmyUnit{{UnitType: "qiQiYing", Amount: 200000}}
	if err := repo.CreatePlayer(account.ID, state, now); err != nil {
		t.Fatalf("create player: %v", err)
	}

	result, err := svc.ResolveSlotRound(state.Player.ID, "qiQiYing", 1000)
	if err != nil {
		t.Fatalf("ResolveSlotRound failed: %v", err)
	}
	if len(result.FreeSpins) != 5 || result.WinAmount != 80000 {
		t.Fatalf("expected five free spins with one tiger win, got %+v", result)
	}
	if armySliceToMap(result.Army)["qiQiYing"] != 199000 {
		t.Fatalf("expected only main spin bet deducted, got %+v", result.Army)
	}
}

func TestResolveSlotRoundFreeSpinsCapAtTwenty(t *testing.T) {
	svc := NewService()
	if err := svc.SetUnitsDir(filepath.Join("..", "..", "..", "config", "units")); err != nil {
		t.Fatalf("load units: %v", err)
	}
	cfg := testSlotConfigForRules()
	scatterGrid := [3][3]string{
		{"scatter", "silver_charm", "gold_charm"},
		{"silver_charm", "gold_charm", "bronze_charm"},
		{"jade_seal", "scatter", "scatter"},
	}
	setTestSlotGridSequence(t, cfg, scatterGrid)
	repo := svc.repo.(*MemoryRepository)
	now := time.Now()
	account := Account{ID: "acc_slot_free_cap", Username: "slot_free_cap", PasswordHash: "x", CreatedAt: now}
	if err := repo.CreateAccount(account); err != nil {
		t.Fatalf("create account: %v", err)
	}
	state := newPlayerState("player_slot_free_cap", "Slotter", "wei", "caocao", now)
	state.Army = []ArmyUnit{{UnitType: "qiQiYing", Amount: 200000}}
	if err := repo.CreatePlayer(account.ID, state, now); err != nil {
		t.Fatalf("create player: %v", err)
	}

	result, err := svc.ResolveSlotRound(state.Player.ID, "qiQiYing", 1000)
	if err != nil {
		t.Fatalf("ResolveSlotRound failed: %v", err)
	}
	if len(result.FreeSpins) != 20 {
		t.Fatalf("expected free spins capped at 20, got %d", len(result.FreeSpins))
	}
}

func TestResolveSlotRoundBonusTriggersReward(t *testing.T) {
	svc := NewService()
	if err := svc.SetUnitsDir(filepath.Join("..", "..", "..", "config", "units")); err != nil {
		t.Fatalf("load units: %v", err)
	}
	cfg := testSlotConfigForRules()
	setTestSlotGridSequence(t, cfg, [3][3]string{
		{"bonus", "silver_charm", "gold_charm"},
		{"silver_charm", "gold_charm", "bonus"},
		{"jade_seal", "bonus", "silver_charm"},
	})
	repo := svc.repo.(*MemoryRepository)
	now := time.Now()
	account := Account{ID: "acc_slot_bonus", Username: "slot_bonus", PasswordHash: "x", CreatedAt: now}
	if err := repo.CreateAccount(account); err != nil {
		t.Fatalf("create account: %v", err)
	}
	state := newPlayerState("player_slot_bonus", "Slotter", "wei", "caocao", now)
	state.Army = []ArmyUnit{{UnitType: "qiQiYing", Amount: 200000}}
	if err := repo.CreatePlayer(account.ID, state, now); err != nil {
		t.Fatalf("create player: %v", err)
	}

	result, err := svc.ResolveSlotRound(state.Player.ID, "qiQiYing", 1000)
	if err != nil {
		t.Fatalf("ResolveSlotRound failed: %v", err)
	}
	if len(result.BonusRewards) != 1 || result.BonusRewards[0].Multiplier != 20 || result.WinAmount != 20000 {
		t.Fatalf("expected bonus reward, got %+v", result)
	}
}

func TestResolveSlotRoundWildCompletesBonusTrigger(t *testing.T) {
	svc := NewService()
	if err := svc.SetUnitsDir(filepath.Join("..", "..", "..", "config", "units")); err != nil {
		t.Fatalf("load units: %v", err)
	}
	cfg := testSlotConfigForRules()
	setTestSlotGridSequence(t, cfg, [3][3]string{
		{"bonus", "silver_charm", "gold_charm"},
		{"silver_charm", "wild", "bronze_charm"},
		{"jade_seal", "tiger_tally", "wild"},
	})
	repo := svc.repo.(*MemoryRepository)
	now := time.Now()
	account := Account{ID: "acc_slot_wild_bonus", Username: "slot_wild_bonus", PasswordHash: "x", CreatedAt: now}
	if err := repo.CreateAccount(account); err != nil {
		t.Fatalf("create account: %v", err)
	}
	state := newPlayerState("player_slot_wild_bonus", "Slotter", "wei", "caocao", now)
	state.Army = []ArmyUnit{{UnitType: "qiQiYing", Amount: 200000}}
	if err := repo.CreatePlayer(account.ID, state, now); err != nil {
		t.Fatalf("create player: %v", err)
	}

	result, err := svc.ResolveSlotRound(state.Player.ID, "qiQiYing", 1000)
	if err != nil {
		t.Fatalf("ResolveSlotRound failed: %v", err)
	}
	if len(result.BonusRewards) != 1 || result.BonusRewards[0].Multiplier != 20 || result.WinAmount != 20000 {
		t.Fatalf("expected wild to complete bonus reward, got %+v", result)
	}
	expectedPositions := [][]int{{0, 0}, {1, 1}, {2, 2}}
	if !reflect.DeepEqual(result.BonusRewards[0].Positions, expectedPositions) {
		t.Fatalf("expected diagonal bonus trigger positions %+v, got %+v", expectedPositions, result.BonusRewards[0].Positions)
	}
}

func TestResolveSlotRoundWildCompletesAllPayReward(t *testing.T) {
	svc := NewService()
	if err := svc.SetUnitsDir(filepath.Join("..", "..", "..", "config", "units")); err != nil {
		t.Fatalf("load units: %v", err)
	}
	cfg := testSlotConfigForRules()
	setTestSlotGridSequence(t, cfg, [3][3]string{
		{"gold_charm", "silver_charm", "bronze_charm"},
		{"wild", "tiger_tally", "gold_charm"},
		{"gold_charm", "jade_seal", "gold_charm"},
	})
	repo := svc.repo.(*MemoryRepository)
	now := time.Now()
	account := Account{ID: "acc_slot_all_pay", Username: "slot_all_pay", PasswordHash: "x", CreatedAt: now}
	if err := repo.CreateAccount(account); err != nil {
		t.Fatalf("create account: %v", err)
	}
	state := newPlayerState("player_slot_all_pay", "Slotter", "wei", "caocao", now)
	state.Army = []ArmyUnit{{UnitType: "qiQiYing", Amount: 200000}}
	if err := repo.CreatePlayer(account.ID, state, now); err != nil {
		t.Fatalf("create player: %v", err)
	}

	result, err := svc.ResolveSlotRound(state.Player.ID, "qiQiYing", 1000)
	if err != nil {
		t.Fatalf("ResolveSlotRound failed: %v", err)
	}
	if len(result.WinningLines) != 0 || len(result.AllPayRewards) != 1 {
		t.Fatalf("expected only one all-pay reward, got %+v", result)
	}
	reward := result.AllPayRewards[0]
	if reward.Symbol != "gold_charm" || reward.Count != 5 || reward.Multiplier != 24 || reward.Amount != 24000 || result.WinAmount != 24000 {
		t.Fatalf("expected wild-completed gold all-pay reward, got %+v", result)
	}
	expectedPositions := [][]int{{0, 0}, {1, 2}, {2, 0}, {2, 2}, {1, 0}}
	if !reflect.DeepEqual(reward.Positions, expectedPositions) {
		t.Fatalf("expected all-pay positions %+v, got %+v", expectedPositions, reward.Positions)
	}
}

func TestResolveSlotRoundAccumulatesMultiplePaylines(t *testing.T) {
	svc := NewService()
	if err := svc.SetUnitsDir(filepath.Join("..", "..", "..", "config", "units")); err != nil {
		t.Fatalf("load units: %v", err)
	}
	cfg := testSlotConfigForRules()
	setTestSlotGridSequence(t, cfg, [3][3]string{
		{"gold_charm", "gold_charm", "gold_charm"},
		{"gold_charm", "gold_charm", "gold_charm"},
		{"gold_charm", "gold_charm", "gold_charm"},
	})
	repo := svc.repo.(*MemoryRepository)
	now := time.Now()
	account := Account{ID: "acc_slot_multiline", Username: "slot_multiline", PasswordHash: "x", CreatedAt: now}
	if err := repo.CreateAccount(account); err != nil {
		t.Fatalf("create account: %v", err)
	}
	state := newPlayerState("player_slot_multiline", "Slotter", "wei", "caocao", now)
	state.Army = []ArmyUnit{{UnitType: "qiQiYing", Amount: 200000}}
	if err := repo.CreatePlayer(account.ID, state, now); err != nil {
		t.Fatalf("create player: %v", err)
	}

	result, err := svc.ResolveSlotRound(state.Player.ID, "qiQiYing", 1000)
	if err != nil {
		t.Fatalf("ResolveSlotRound failed: %v", err)
	}
	if len(result.WinningLines) != 5 || len(result.AllPayRewards) != 1 || result.WinAmount != 420000 {
		t.Fatalf("expected five accumulated lines plus gold all-pay, got %+v", result)
	}
}

func TestResolveSlotRoundRejectsInvalidOrUnaffordableBetWithoutRecord(t *testing.T) {
	svc := NewService()
	if err := svc.SetUnitsDir(filepath.Join("..", "..", "..", "config", "units")); err != nil {
		t.Fatalf("load units: %v", err)
	}
	cfg := testSlotConfigForRules()
	setTestSlotGridSequence(t, cfg, [3][3]string{
		{"bronze_charm", "silver_charm", "gold_charm"},
		{"silver_charm", "gold_charm", "bronze_charm"},
		{"jade_seal", "bronze_charm", "silver_charm"},
	})
	repo := svc.repo.(*MemoryRepository)
	now := time.Now()
	account := Account{ID: "acc_slot_bounds", Username: "slot_bounds", PasswordHash: "x", CreatedAt: now}
	if err := repo.CreateAccount(account); err != nil {
		t.Fatalf("create account: %v", err)
	}
	state := newPlayerState("player_slot_bounds", "Slotter", "wei", "caocao", now)
	state.Army = []ArmyUnit{{UnitType: "qiQiYing", Amount: 100000}}
	if err := repo.CreatePlayer(account.ID, state, now); err != nil {
		t.Fatalf("create player: %v", err)
	}

	if _, err := svc.ResolveSlotRound(state.Player.ID, "qiQiYing", 999); !errors.Is(err, ErrMiniGameBetTooLow) {
		t.Fatalf("expected ErrMiniGameBetTooLow, got %v", err)
	}
	if _, err := svc.ResolveSlotRound(state.Player.ID, "qiQiYing", 10000); err != nil {
		t.Fatalf("expected arbitrary upper bet limit removed, got %v", err)
	}
	if _, err := svc.ResolveSlotRound(state.Player.ID, "qiQiYing", 100000); !errors.Is(err, ErrInsufficientArmy) {
		t.Fatalf("expected ErrInsufficientArmy for unaffordable spin bet, got %v", err)
	}
	records, _, err := repo.ListMiniGameRecords(state.Player.ID, "slot", 10, 0)
	if err != nil {
		t.Fatalf("list slot records: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("only affordable round should write record, got %+v", records)
	}
	stored, err := repo.GetState(state.Player.ID)
	if err != nil {
		t.Fatalf("GetState failed: %v", err)
	}
	if armySliceToMap(stored.Army)["qiQiYing"] != 90000 {
		t.Fatalf("unaffordable bet should not deduct extra army, got %+v", stored.Army)
	}
}

func TestResolveSlotRoundRejectsInvalidUnits(t *testing.T) {
	svc := NewService()
	if err := svc.SetUnitsDir(filepath.Join("..", "..", "..", "config", "units")); err != nil {
		t.Fatalf("load units: %v", err)
	}
	repo := svc.repo.(*MemoryRepository)
	now := time.Now()
	account := Account{ID: "acc_slot_units", Username: "slot_units", PasswordHash: "x", CreatedAt: now}
	if err := repo.CreateAccount(account); err != nil {
		t.Fatalf("create account: %v", err)
	}
	state := newPlayerState("player_slot_units", "Slotter", "wei", "caocao", now)
	state.Army = []ArmyUnit{{UnitType: "zhanYingTanMa", Amount: 100000}, {UnitType: "southernElephant", Amount: 100000}}
	if err := repo.CreatePlayer(account.ID, state, now); err != nil {
		t.Fatalf("create player: %v", err)
	}

	if _, err := svc.ResolveSlotRound(state.Player.ID, "southernElephant", 1000); !errors.Is(err, ErrUnitNotFound) {
		t.Fatalf("expected ErrUnitNotFound for cross faction unit, got %v", err)
	}
	if _, err := svc.ResolveSlotRound(state.Player.ID, "zhanYingTanMa", 1000); !errors.Is(err, ErrNonCombatUnit) {
		t.Fatalf("expected ErrNonCombatUnit for scout unit, got %v", err)
	}
}

func TestRedeemSlotMiniGameRewardAddsFactionArmy(t *testing.T) {
	svc := NewService()
	if err := svc.SetUnitsDir(filepath.Join("..", "..", "..", "config", "units")); err != nil {
		t.Fatalf("load units: %v", err)
	}
	repo := svc.repo.(*MemoryRepository)
	now := time.Now()
	account := Account{ID: "acc_slot_redeem", Username: "slot_redeem", PasswordHash: "x", CreatedAt: now}
	if err := repo.CreateAccount(account); err != nil {
		t.Fatalf("create account: %v", err)
	}
	state := newPlayerState("player_slot_redeem", "Slotter", "wei", "caocao", now)
	if err := repo.CreatePlayer(account.ID, state, now); err != nil {
		t.Fatalf("create player: %v", err)
	}
	record := MiniGameRecord{
		ID:              "mg_slot_redeem",
		PlayerID:        state.Player.ID,
		GameType:        "slot",
		ResultName:      "玄铜符三连 ×5",
		Rarity:          "common",
		RewardUnit:      "骁骑营",
		RewardAmount:    50000,
		RemainingAmount: 50000,
		BetUnit:         "骁骑营",
		BetAmount:       10000,
		CreatedAt:       now.UTC().Format(resourceDateLayout),
	}
	if err := repo.SaveMiniGameRecord(record); err != nil {
		t.Fatalf("save slot record: %v", err)
	}

	result, err := svc.RedeemMiniGameReward(state.Player.ID, record.ID, 12000)
	if err != nil {
		t.Fatalf("RedeemMiniGameReward failed: %v", err)
	}
	if result.Record.GameType != "slot" || result.Record.RemainingAmount != 38000 {
		t.Fatalf("unexpected redeemed slot record: %+v", result.Record)
	}
	if result.RedeemedTarget != "army" || result.RedeemedUnitID != "qiQiYing" || result.RedeemedAmount != 12000 {
		t.Fatalf("unexpected slot redeem result: %+v", result)
	}
	if armySliceToMap(result.Army)["qiQiYing"] != 12000 {
		t.Fatalf("expected redeemed slot unit in army, got %+v", result.Army)
	}
}

func TestRedeemAllSlotMiniGameRewardsOnlyRedeemsSlot(t *testing.T) {
	svc := NewService()
	if err := svc.SetUnitsDir(filepath.Join("..", "..", "..", "config", "units")); err != nil {
		t.Fatalf("load units: %v", err)
	}
	repo := svc.repo.(*MemoryRepository)
	now := time.Now()
	account := Account{ID: "acc_slot_redeem_all", Username: "slot_redeem_all", PasswordHash: "x", CreatedAt: now}
	if err := repo.CreateAccount(account); err != nil {
		t.Fatalf("create account: %v", err)
	}
	state := newPlayerState("player_slot_redeem_all", "Slotter", "wei", "caocao", now)
	if err := repo.CreatePlayer(account.ID, state, now); err != nil {
		t.Fatalf("create player: %v", err)
	}
	if err := repo.SaveMiniGameRecord(MiniGameRecord{ID: "mg_slot_all", PlayerID: state.Player.ID, GameType: "slot", ResultName: "玄铜符三连 ×5", Rarity: "common", RewardUnit: "骁骑营", RewardAmount: 50000, RemainingAmount: 50000, BetUnit: "骁骑营", BetAmount: 10000, CreatedAt: now.UTC().Format(resourceDateLayout)}); err != nil {
		t.Fatalf("save slot record: %v", err)
	}
	if _, err := svc.SaveMiniGameRecord(state.Player.ID, "fishing", "金龙鱼", "rare", "骁骑营", 5000, "", 0); err != nil {
		t.Fatalf("save fishing record: %v", err)
	}

	result, err := svc.RedeemAllFactionMiniGameRewards(state.Player.ID, "slot")
	if err != nil {
		t.Fatalf("RedeemAllFactionMiniGameRewards failed: %v", err)
	}
	if result.RedeemedAmount != 50000 || result.RedeemedRecords != 1 || result.RedeemedUnits["骁骑营"] != 50000 {
		t.Fatalf("unexpected slot redeem all result: %+v", result)
	}
	slotRecords, _, err := repo.ListMiniGameRecords(state.Player.ID, "slot", 10, 0)
	if err != nil {
		t.Fatalf("list slot records: %v", err)
	}
	if len(slotRecords) != 1 || slotRecords[0].RemainingAmount != 0 {
		t.Fatalf("expected slot stock redeemed, got %+v", slotRecords)
	}
	fishingRecords, _, err := repo.ListMiniGameRecords(state.Player.ID, "fishing", 10, 0)
	if err != nil {
		t.Fatalf("list fishing records: %v", err)
	}
	if len(fishingRecords) != 1 || fishingRecords[0].RemainingAmount != 5000 {
		t.Fatalf("expected fishing stock untouched, got %+v", fishingRecords)
	}
}

func TestSaveMiniGameRecordRejectsSlotGameType(t *testing.T) {
	svc := NewService()
	repo := svc.repo.(*MemoryRepository)
	now := time.Now()
	account := Account{ID: "acc_slot_record_reject", Username: "slot_record_reject", PasswordHash: "x", CreatedAt: now}
	if err := repo.CreateAccount(account); err != nil {
		t.Fatalf("create account: %v", err)
	}
	state := newPlayerState("player_slot_record_reject", "Slotter", "wei", "caocao", now)
	if err := repo.CreatePlayer(account.ID, state, now); err != nil {
		t.Fatalf("create player: %v", err)
	}
	if _, err := svc.SaveMiniGameRecord(state.Player.ID, "slot", "玄铜符三连 ×5", "common", "骁骑营", 50000, "骁骑营", 10000); !errors.Is(err, ErrInvalidMiniGame) {
		t.Fatalf("expected ErrInvalidMiniGame for slot record upload, got %v", err)
	}
}

func TestRedeemAllGamblingMiniGameRewardsOnlyRedeemsGambling(t *testing.T) {
	svc := NewService()
	if err := svc.SetUnitsDir(filepath.Join("..", "..", "..", "config", "units")); err != nil {
		t.Fatalf("load units: %v", err)
	}
	repo := svc.repo.(*MemoryRepository)
	now := time.Now()
	account := Account{ID: "acc_gambling_redeem_all", Username: "gambling_redeem_all", PasswordHash: "x", CreatedAt: now}
	if err := repo.CreateAccount(account); err != nil {
		t.Fatalf("create account: %v", err)
	}
	state := newPlayerState("player_gambling_redeem_all", "Gambler", "wei", "caocao", now)
	if err := repo.CreatePlayer(account.ID, state, now); err != nil {
		t.Fatalf("create player: %v", err)
	}
	if _, err := svc.SaveMiniGameRecord(state.Player.ID, "gambling", "大 赢 ×2", "rare", "骁骑营", 20000, "骁骑营", 10000); err != nil {
		t.Fatalf("save gambling record: %v", err)
	}
	if _, err := svc.SaveMiniGameRecord(state.Player.ID, "fishing", "金龙鱼", "rare", "骁骑营", 5000, "", 0); err != nil {
		t.Fatalf("save fishing record: %v", err)
	}

	result, err := svc.RedeemAllFactionMiniGameRewards(state.Player.ID, "gambling")
	if err != nil {
		t.Fatalf("RedeemAllFactionMiniGameRewards failed: %v", err)
	}
	if result.RedeemedAmount != 20000 || result.RedeemedRecords != 1 || result.RedeemedUnits["骁骑营"] != 20000 {
		t.Fatalf("unexpected gambling redeem all result: %+v", result)
	}
	gamblingRecords, _, err := repo.ListMiniGameRecords(state.Player.ID, "gambling", 10, 0)
	if err != nil {
		t.Fatalf("list gambling records: %v", err)
	}
	if len(gamblingRecords) != 1 || gamblingRecords[0].RemainingAmount != 0 {
		t.Fatalf("expected gambling stock redeemed, got %+v", gamblingRecords)
	}
	fishingRecords, _, err := repo.ListMiniGameRecords(state.Player.ID, "fishing", 10, 0)
	if err != nil {
		t.Fatalf("list fishing records: %v", err)
	}
	if len(fishingRecords) != 1 || fishingRecords[0].RemainingAmount != 5000 {
		t.Fatalf("expected fishing stock untouched, got %+v", fishingRecords)
	}
}

func TestUseFishingBaitDeductsCityGoldAndWritesLedger(t *testing.T) {
	svc := NewService()
	repo := svc.repo.(*MemoryRepository)
	now := time.Now()
	account := Account{ID: "acc_fishing_bait", Username: "fishing_bait", PasswordHash: "x", CreatedAt: now}
	if err := repo.CreateAccount(account); err != nil {
		t.Fatalf("create account: %v", err)
	}
	state := newPlayerState("player_fishing_bait", "Fisher", "wei", "caocao", now)
	state.CityGold = 150
	if err := repo.CreatePlayer(account.ID, state, now); err != nil {
		t.Fatalf("create player: %v", err)
	}

	result, err := svc.UseFishingBait(state.Player.ID, "golden")
	if err != nil {
		t.Fatalf("UseFishingBait failed: %v", err)
	}
	if result.CityGold == nil || result.CityGoldRemain == nil || result.CityGoldCost != 120 || *result.CityGoldRemain != 30 || *result.CityGold != 30 {
		t.Fatalf("unexpected bait result: %+v", result)
	}

	entries, err := svc.ListGoldLedger(GoldLedgerFilter{PlayerID: state.Player.ID, RefType: LedgerRefMiniGameBait})
	if err != nil {
		t.Fatalf("ListGoldLedger failed: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 ledger entry, got %d: %+v", len(entries), entries)
	}
	entry := entries[0]
	if entry.Currency != LedgerCurrencyCityGold || entry.Direction != LedgerDirectionDebit || entry.Amount != 120 || entry.BalanceAfter != 30 {
		t.Fatalf("unexpected ledger entry: %+v", entry)
	}
}

func TestRedeemMiniGameRewardCrossFactionCreatesGarrison(t *testing.T) {
	svc := NewService()
	if err := svc.SetUnitsDir(filepath.Join("..", "..", "..", "config", "units")); err != nil {
		t.Fatalf("load units: %v", err)
	}
	repo := svc.repo.(*MemoryRepository)
	now := time.Now()
	account := Account{ID: "acc_fishing_cross", Username: "fishing_cross", PasswordHash: "x", CreatedAt: now}
	if err := repo.CreateAccount(account); err != nil {
		t.Fatalf("create account: %v", err)
	}
	state := newPlayerState("player_fishing_cross", "Fisher", "wei", "caocao", now)
	if err := repo.CreatePlayer(account.ID, state, now); err != nil {
		t.Fatalf("create player: %v", err)
	}
	record, err := svc.SaveMiniGameRecord(state.Player.ID, "fishing", "蛟龙", "epic", "南蛮象", 50000, "", 0)
	if err != nil {
		t.Fatalf("save minigame record: %v", err)
	}

	result, err := svc.RedeemMiniGameReward(state.Player.ID, record.ID, 1000)
	if err != nil {
		t.Fatalf("RedeemMiniGameReward failed: %v", err)
	}
	if result.RedeemedTarget != "garrison" || result.RedeemedUnit != "南蛮象" || result.RedeemedAmount != 1000 {
		t.Fatalf("unexpected redeem result: %+v", result)
	}
	if result.Garrison == nil || result.Garrison.SourceType != GarrisonSourceObtained || result.Garrison.RemainingTroops[result.RedeemedUnitID] != 1000 {
		t.Fatalf("expected obtained garrison, got %+v", result.Garrison)
	}
	records, _, err := repo.ListMiniGameRecords(state.Player.ID, "fishing", 10, 0)
	if err != nil {
		t.Fatalf("list records: %v", err)
	}
	if len(records) != 1 || records[0].RemainingAmount != 49000 {
		t.Fatalf("expected stock reduced, got %+v", records)
	}
	next, err := repo.GetState(state.Player.ID)
	if err != nil {
		t.Fatalf("GetState failed: %v", err)
	}
	for _, unit := range next.Army {
		if unit.UnitType == result.RedeemedUnitID {
			t.Fatalf("cross faction unit should not enter regular army: %+v", next.Army)
		}
	}
	defenders, err := svc.BuildDefenseReinforcementUnits(state.Player.ID)
	if err != nil {
		t.Fatalf("BuildDefenseReinforcementUnits failed: %v", err)
	}
	if len(defenders) != 1 || defenders[0].SourceTags["source_type"] != GarrisonSourceObtained {
		t.Fatalf("expected garrison defender, got %+v", defenders)
	}
}

func TestRedeemAllFactionMiniGameRewardsSkipsCrossFactionStock(t *testing.T) {
	svc := NewService()
	if err := svc.SetUnitsDir(filepath.Join("..", "..", "..", "config", "units")); err != nil {
		t.Fatalf("load units: %v", err)
	}
	repo := svc.repo.(*MemoryRepository)
	now := time.Now()
	account := Account{ID: "acc_fishing_redeem_all", Username: "fishing_redeem_all", PasswordHash: "x", CreatedAt: now}
	if err := repo.CreateAccount(account); err != nil {
		t.Fatalf("create account: %v", err)
	}
	state := newPlayerState("player_fishing_redeem_all", "Fisher", "wei", "caocao", now)
	if err := repo.CreatePlayer(account.ID, state, now); err != nil {
		t.Fatalf("create player: %v", err)
	}
	if _, err := svc.SaveMiniGameRecord(state.Player.ID, "fishing", "金龙鱼", "rare", "骁骑营", 5000, "", 0); err != nil {
		t.Fatalf("save first record: %v", err)
	}
	if _, err := svc.SaveMiniGameRecord(state.Player.ID, "fishing", "银鲤", "common", "青州军", 2000, "", 0); err != nil {
		t.Fatalf("save second record: %v", err)
	}
	if _, err := svc.SaveMiniGameRecord(state.Player.ID, "fishing", "蛟龙", "epic", "南蛮象", 50000, "", 0); err != nil {
		t.Fatalf("save cross faction record: %v", err)
	}

	redeemEvents := []GameEvent{}
	rewardEvents := []GameEvent{}
	svc.SubscribeEvent(EventMiniGameRedeemed, func(event GameEvent) {
		redeemEvents = append(redeemEvents, event)
	})
	svc.SubscribeEvent(EventRewardGranted, func(event GameEvent) {
		rewardEvents = append(rewardEvents, event)
	})

	result, err := svc.RedeemAllFactionMiniGameRewards(state.Player.ID, "fishing")
	if err != nil {
		t.Fatalf("RedeemAllFactionMiniGameRewards failed: %v", err)
	}
	if result.RedeemedAmount != 57000 {
		t.Fatalf("expected redeemed amount 57000, got %d", result.RedeemedAmount)
	}
	if result.RedeemedRecords != 3 {
		t.Fatalf("expected 3 redeemed records, got %d", result.RedeemedRecords)
	}
	if result.SkippedRecords != 0 || len(result.SkippedUnits) != 0 {
		t.Fatalf("expected no skipped cross faction stock, got records=%d units=%+v", result.SkippedRecords, result.SkippedUnits)
	}
	if result.RedeemedUnits["骁骑营"] != 5000 || result.RedeemedUnits["青州军"] != 2000 {
		t.Fatalf("unexpected redeemed unit totals: %+v", result.RedeemedUnits)
	}
	if result.GarrisonedUnits["南蛮象"] != 50000 || result.GarrisonRecords != 1 {
		t.Fatalf("unexpected garrison totals: records=%d units=%+v", result.GarrisonRecords, result.GarrisonedUnits)
	}

	records, _, err := repo.ListMiniGameRecords(state.Player.ID, "fishing", 10, 0)
	if err != nil {
		t.Fatalf("list records: %v", err)
	}
	remainingByUnit := map[string]int{}
	for _, record := range records {
		remainingByUnit[record.RewardUnit] += record.RemainingAmount
	}
	if remainingByUnit["骁骑营"] != 0 || remainingByUnit["青州军"] != 0 || remainingByUnit["南蛮象"] != 0 {
		t.Fatalf("unexpected remaining stock: %+v", remainingByUnit)
	}
	if len(redeemEvents) != 1 || redeemEvents[0].Type != EventMiniGameRedeemed {
		t.Fatalf("expected one minigame redeemed event, got %+v", redeemEvents)
	}
	if len(rewardEvents) != 2 {
		t.Fatalf("expected two reward events from core reward pipeline, got %+v", rewardEvents)
	}
	defenders, err := svc.BuildDefenseReinforcementUnits(state.Player.ID)
	if err != nil {
		t.Fatalf("BuildDefenseReinforcementUnits failed: %v", err)
	}
	if len(defenders) != 1 || defenders[0].Troops["southernElephant"] != 50000 {
		t.Fatalf("expected cross faction garrison, got %+v", defenders)
	}
}

func TestRedeemMiniGameRewardAcceptsLegacyTuZuName(t *testing.T) {
	svc := NewService()
	if err := svc.SetUnitsDir(filepath.Join("..", "..", "..", "config", "units")); err != nil {
		t.Fatalf("load units: %v", err)
	}
	repo := svc.repo.(*MemoryRepository)
	now := time.Now()
	account := Account{ID: "acc_fishing_legacy_tuzu", Username: "fishing_legacy_tuzu", PasswordHash: "x", CreatedAt: now}
	if err := repo.CreateAccount(account); err != nil {
		t.Fatalf("create account: %v", err)
	}
	state := newPlayerState("player_fishing_legacy_tuzu", "Fisher", "wei", "caocao", now)
	if err := repo.CreatePlayer(account.ID, state, now); err != nil {
		t.Fatalf("create player: %v", err)
	}
	record, err := svc.SaveMiniGameRecord(state.Player.ID, "fishing", "神龙", "legendary", "土族", 2000, "", 0)
	if err != nil {
		t.Fatalf("save legacy record: %v", err)
	}

	result, err := svc.RedeemMiniGameReward(state.Player.ID, record.ID, 2000)
	if err != nil {
		t.Fatalf("RedeemMiniGameReward failed: %v", err)
	}
	if result.RedeemedUnitID != "tuZu" || result.RedeemedUnit != "士族" || result.RedeemedAmount != 2000 {
		t.Fatalf("unexpected redeem result: %+v", result)
	}
}

func TestUseItemConsumesInventoryAndAppliesGeneralExp(t *testing.T) {
	svc := NewService()
	loadTestItemsConfig(t)
	repo := svc.repo.(*MemoryRepository)
	now := time.Now()
	account := Account{ID: "account_item_use", Username: "item_user", PasswordHash: "hash", CreatedAt: now}
	if err := repo.CreateAccount(account); err != nil {
		t.Fatalf("create account: %v", err)
	}
	state := newPlayerState("player_item_use", "ItemUser", "wei", "caocao", now)
	if err := repo.CreatePlayer(account.ID, state, now); err != nil {
		t.Fatalf("create player: %v", err)
	}

	state, err := svc.GrantItem("player_item_use", "test_general_exp_small", 2)
	if err != nil {
		t.Fatalf("GrantItem failed: %v", err)
	}
	if state.Inventory["test_general_exp_small"].Amount != 2 {
		t.Fatalf("expected 2 items after grant, got %d", state.Inventory["test_general_exp_small"].Amount)
	}

	result, err := svc.UseItem("player_item_use", "test_general_exp_small", 1)
	if err != nil {
		t.Fatalf("UseItem failed: %v", err)
	}
	if result.Patch.Inventory["test_general_exp_small"].Amount != 1 {
		t.Fatalf("expected 1 item left, got %d", result.Patch.Inventory["test_general_exp_small"].Amount)
	}
	if result.Patch.General.Exp != 100 {
		t.Fatalf("expected general exp 100, got %d", result.Patch.General.Exp)
	}
	payload, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("marshal use item result: %v", err)
	}
	if strings.Contains(string(payload), `"cityGold"`) {
		t.Fatalf("general exp item patch must not include cityGold, got %s", string(payload))
	}
}

func TestUseRecruitTicketGrantsBaseFactionUnits(t *testing.T) {
	svc := NewService()
	loadRecruitItemProductionConfig(t)
	repo := svc.repo.(*MemoryRepository)
	now := time.Now()
	account := Account{ID: "account_recruit_ticket", Username: "recruit_ticket", PasswordHash: "hash", CreatedAt: now}
	if err := repo.CreateAccount(account); err != nil {
		t.Fatalf("create account: %v", err)
	}
	state := newPlayerState("player_recruit_ticket", "RecruitTicket", "wei", "caocao", now)
	if err := repo.CreatePlayer(account.ID, state, now); err != nil {
		t.Fatalf("create player: %v", err)
	}
	if _, err := svc.GrantItem(state.Player.ID, "recruit_ticket_medium", 1); err != nil {
		t.Fatalf("GrantItem failed: %v", err)
	}

	result, err := svc.UseItem(state.Player.ID, "recruit_ticket_medium", 1)
	if err != nil {
		t.Fatalf("UseItem failed: %v", err)
	}
	army := armySliceToMap(result.Patch.Army)
	if army["qingZhouArmy"] != 20000 || army["qiQiYing"] != 5000 {
		t.Fatalf("expected medium recruit ticket to grant base wei units, got %+v", army)
	}
	if army["zhanYingTanMa"] != 0 {
		t.Fatalf("recruit ticket must not grant scout units, got %+v", army)
	}
}

func TestUseConscriptionOrderGrantsAllBaseFactionUnits(t *testing.T) {
	svc := NewService()
	loadRecruitItemProductionConfig(t)
	repo := svc.repo.(*MemoryRepository)
	now := time.Now()
	account := Account{ID: "account_conscription_order", Username: "conscription_order", PasswordHash: "hash", CreatedAt: now}
	if err := repo.CreateAccount(account); err != nil {
		t.Fatalf("create account: %v", err)
	}
	state := newPlayerState("player_conscription_order", "ConscriptionOrder", "shu", "liubei", now)
	if err := repo.CreatePlayer(account.ID, state, now); err != nil {
		t.Fatalf("create player: %v", err)
	}
	if _, err := svc.GrantItem(state.Player.ID, "conscription_order", 1); err != nil {
		t.Fatalf("GrantItem failed: %v", err)
	}

	result, err := svc.UseItem(state.Player.ID, "conscription_order", 1)
	if err != nil {
		t.Fatalf("UseItem failed: %v", err)
	}
	army := armySliceToMap(result.Patch.Army)
	if army["greedyWolf"] != 50000 || army["xiLiangCavalry"] != 15000 || army["siegeTower"] != 4000 {
		t.Fatalf("expected conscription order to grant all base shu units, got %+v", army)
	}
	if army["flyingKite"] != 0 || army["shuMerchant"] != 0 {
		t.Fatalf("conscription order must not grant function units, got %+v", army)
	}
}

func TestSpecialConscriptionOrderUsesTwoNonMerchantSpecialUnits(t *testing.T) {
	loadRecruitItemProductionConfig(t)
	expected := []string{"jianzhuShi", "tuZu"}
	if got := factionUnitIDsByCategory("wei", "special", "all"); !reflect.DeepEqual(got, expected) {
		t.Fatalf("expected wei special combat pool %+v, got %+v", expected, got)
	}
}

func TestUseSpecialConscriptionOrderGrantsOneSpecialUnit(t *testing.T) {
	svc := NewService()
	loadRecruitItemProductionConfig(t)
	repo := svc.repo.(*MemoryRepository)
	now := time.Now()
	account := Account{ID: "account_special_conscription", Username: "special_conscription", PasswordHash: "hash", CreatedAt: now}
	if err := repo.CreateAccount(account); err != nil {
		t.Fatalf("create account: %v", err)
	}
	state := newPlayerState("player_special_conscription", "SpecialConscription", "wu", "sunquan", now)
	if err := repo.CreatePlayer(account.ID, state, now); err != nil {
		t.Fatalf("create player: %v", err)
	}
	if _, err := svc.GrantItem(state.Player.ID, "special_conscription_order", 1); err != nil {
		t.Fatalf("GrantItem failed: %v", err)
	}

	result, err := svc.UseItem(state.Player.ID, "special_conscription_order", 1)
	if err != nil {
		t.Fatalf("UseItem failed: %v", err)
	}
	army := armySliceToMap(result.Patch.Army)
	if army["fengShuiMaster"]+army["taiPingShi"] != 10000 {
		t.Fatalf("expected exactly one wu special combat unit with 10000 troops, got %+v", army)
	}
	if army["wuMerchant"] != 0 || army["shadowGuard"] != 0 || army["divineWind"] != 0 || army["chongChe"] != 0 {
		t.Fatalf("special conscription order must not grant merchant or non-special units, got %+v", army)
	}
}

func TestGrantItemSplitsInventoryStacksByMaxStack(t *testing.T) {
	svc := NewService()
	loadStackSplitItemsConfig(t)
	repo := svc.repo.(*MemoryRepository)
	now := time.Now()
	account := Account{ID: "account_item_stack_split", Username: "item_stack_split", PasswordHash: "hash", CreatedAt: now}
	if err := repo.CreateAccount(account); err != nil {
		t.Fatalf("create account: %v", err)
	}
	state := newPlayerState("player_item_stack_split", "ItemStackSplit", "wei", "caocao", now)
	if err := repo.CreatePlayer(account.ID, state, now); err != nil {
		t.Fatalf("create player: %v", err)
	}

	state, err := svc.GrantItem(state.Player.ID, "test_stack_item", 2500)
	if err != nil {
		t.Fatalf("GrantItem failed: %v", err)
	}
	if state.Inventory["test_stack_item"].Amount != 2500 {
		t.Fatalf("expected aggregate amount 2500, got %d", state.Inventory["test_stack_item"].Amount)
	}
	if len(state.InventorySlots) != 3 {
		t.Fatalf("expected 3 inventory slots, got %+v", state.InventorySlots)
	}
	amounts := []int{state.InventorySlots[0].Amount, state.InventorySlots[1].Amount, state.InventorySlots[2].Amount}
	if amounts[0] != 999 || amounts[1] != 999 || amounts[2] != 502 {
		t.Fatalf("expected split 999/999/502, got %+v", amounts)
	}
}

func TestUseItemPublishesItemUsedEvent(t *testing.T) {
	svc := NewService()
	loadTestItemsConfig(t)
	repo := svc.repo.(*MemoryRepository)
	now := time.Now()
	account := Account{ID: "account_item_event", Username: "item_event", PasswordHash: "hash", CreatedAt: now}
	if err := repo.CreateAccount(account); err != nil {
		t.Fatalf("create account: %v", err)
	}
	state := newPlayerState("player_item_event", "ItemEvent", "wei", "caocao", now)
	if err := repo.CreatePlayer(account.ID, state, now); err != nil {
		t.Fatalf("create player: %v", err)
	}
	if _, err := svc.GrantItem(state.Player.ID, "test_general_exp_small", 1); err != nil {
		t.Fatalf("GrantItem failed: %v", err)
	}

	events := []GameEvent{}
	svc.SubscribeEvent(EventItemUsed, func(event GameEvent) {
		events = append(events, event)
	})

	if _, err := svc.UseItem(state.Player.ID, "test_general_exp_small", 1); err != nil {
		t.Fatalf("UseItem failed: %v", err)
	}
	if len(events) != 1 || events[0].Type != EventItemUsed || events[0].RefID != "test_general_exp_small" {
		t.Fatalf("expected item used event, got %+v", events)
	}
}

func TestItemUseRewardAssetScopeIncludesConsumedAndEffectAssets(t *testing.T) {
	scope := itemUseRewardAssetScope(ItemDefinition{
		ID: "resource_pack_small",
		Effects: []ItemEffect{
			{Type: "resources", Resources: map[string]int{"wood": 100}},
			{Type: "item", ID: "general_exp_small", Amount: 1},
			{Type: "unit_by_faction", UnitByFaction: map[string]string{"wei": "weiInfantry", "shu": "shuInfantry"}},
			{Type: "general_exp", Amount: 50},
			{Type: "buff", BuffKey: "attackBonus", Amount: 1},
		},
	})
	itemIDs := map[string]bool{}
	for _, itemID := range scope.InventoryItemIDs {
		itemIDs[itemID] = true
	}
	unitTypes := map[string]bool{}
	for _, unitType := range scope.UnitTypes {
		unitTypes[unitType] = true
	}
	if !scope.Resources || !scope.CurrentGeneral || !scope.Buffs {
		t.Fatalf("expected resources/current general/buffs scope, got %+v", scope)
	}
	if !itemIDs["resource_pack_small"] || !itemIDs["general_exp_small"] {
		t.Fatalf("expected consumed and rewarded item ids in scope, got %+v", scope.InventoryItemIDs)
	}
	if !unitTypes["weiInfantry"] || !unitTypes["shuInfantry"] {
		t.Fatalf("expected candidate faction units in scope, got %+v", scope.UnitTypes)
	}
}

func loadStackSplitItemsConfig(t *testing.T) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "items.json")
	data := []byte(`{
		"test_stack_item": {
			"id": "test_stack_item",
			"name": "测试堆叠物品",
			"description": "测试用",
			"category": "material",
			"quality": "common",
			"usable": false,
			"stackable": true,
			"maxStack": 999,
			"useTarget": "self",
			"confirmOnUse": "auto",
			"effects": []
		}
	}`)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("write stack item config: %v", err)
	}
	if err := LoadItemsConfig(path); err != nil {
		t.Fatalf("load stack item config: %v", err)
	}
	t.Cleanup(func() {
		_ = LoadItemsConfig(filepath.Join("..", "..", "..", "config", "items.json"))
	})
}

func TestMailItemAttachmentAddsInventory(t *testing.T) {
	svc := NewService()
	loadTestItemsConfig(t)
	repo := svc.repo.(*MemoryRepository)
	now := time.Now()
	account := Account{ID: "account_mail_item", Username: "mail_item", PasswordHash: "hash", CreatedAt: now}
	if err := repo.CreateAccount(account); err != nil {
		t.Fatalf("create account: %v", err)
	}
	state := newPlayerState("player_mail_item", "MailItem", "wei", "caocao", now)
	if err := repo.CreatePlayer(account.ID, state, now); err != nil {
		t.Fatalf("create player: %v", err)
	}

	mail, err := svc.SendMail(SendMailRequest{
		PlayerID:   "player_mail_item",
		MailType:   "reward",
		SenderType: "gm",
		Title:      "物品测试",
		Content:    "测试物品附件",
		Attachments: []MailAttachment{{
			Type:   "item",
			ItemID: "test_general_exp_small",
			Amount: 3,
		}},
	})
	if err != nil {
		t.Fatalf("SendMail failed: %v", err)
	}
	result, err := svc.ClaimMailAttachments("player_mail_item", mail.ID)
	if err != nil {
		t.Fatalf("ClaimMailAttachments failed: %v", err)
	}
	next, err := repo.GetState("player_mail_item")
	if err != nil {
		t.Fatalf("GetState failed: %v", err)
	}
	if next.Inventory["test_general_exp_small"].Amount != 3 {
		t.Fatalf("expected 3 mail items, got %d", next.Inventory["test_general_exp_small"].Amount)
	}
	if result.GrantedItems["test_general_exp_small"] != 3 {
		t.Fatalf("expected granted item count 3, got %d", result.GrantedItems["test_general_exp_small"])
	}
}

func TestClaimMailCityGoldWritesLedgerAndEvents(t *testing.T) {
	svc := NewService()
	repo := svc.repo.(*MemoryRepository)
	now := time.Now()
	account := Account{ID: "account_mail_city_gold", Username: "mail_city_gold", PasswordHash: "hash", CreatedAt: now}
	if err := repo.CreateAccount(account); err != nil {
		t.Fatalf("create account: %v", err)
	}
	state := newPlayerState("player_mail_city_gold", "MailCityGold", "wei", "caocao", now)
	if err := repo.CreatePlayer(account.ID, state, now); err != nil {
		t.Fatalf("create player: %v", err)
	}

	mailEvents := []GameEvent{}
	rewardEvents := []GameEvent{}
	svc.SubscribeEvent(EventMailClaimed, func(event GameEvent) {
		mailEvents = append(mailEvents, event)
	})
	svc.SubscribeEvent(EventRewardGranted, func(event GameEvent) {
		rewardEvents = append(rewardEvents, event)
	})

	mail, err := svc.SendMail(SendMailRequest{
		PlayerID:   "player_mail_city_gold",
		MailType:   "reward",
		SenderType: "gm",
		Title:      "城金测试",
		Content:    "测试城金附件",
		Attachments: []MailAttachment{{
			Type:   RewardTypeCityGold,
			ItemID: RewardTypeCityGold,
			Amount: 30,
		}},
	})
	if err != nil {
		t.Fatalf("SendMail failed: %v", err)
	}
	result, err := svc.ClaimMailAttachments("player_mail_city_gold", mail.ID)
	if err != nil {
		t.Fatalf("ClaimMailAttachments failed: %v", err)
	}
	if result.CityGold != 30 || result.GrantedItems[RewardTypeCityGold] != 30 {
		t.Fatalf("unexpected claim result: %+v", result)
	}
	entries, err := svc.ListGoldLedger(GoldLedgerFilter{PlayerID: "player_mail_city_gold", RefType: LedgerRefMailClaim})
	if err != nil {
		t.Fatalf("ListGoldLedger failed: %v", err)
	}
	if len(entries) != 1 || entries[0].Amount != 30 || entries[0].BalanceAfter != 30 {
		t.Fatalf("unexpected mail claim ledger: %+v", entries)
	}
	if len(mailEvents) != 1 || mailEvents[0].Type != EventMailClaimed {
		t.Fatalf("expected mail claimed event, got %+v", mailEvents)
	}
	if len(rewardEvents) != 1 || rewardEvents[0].Type != EventRewardGranted {
		t.Fatalf("expected reward granted event, got %+v", rewardEvents)
	}
}

func TestClaimMailAccountGoldUsesCombinedTransactionLedgerAndEvents(t *testing.T) {
	svc := NewService()
	repo := svc.repo.(*MemoryRepository)
	now := time.Now()
	account := Account{ID: "account_mail_gold", Username: "mail_gold", PasswordHash: "hash", Gold: 5, CreatedAt: now}
	if err := repo.CreateAccount(account); err != nil {
		t.Fatalf("create account: %v", err)
	}
	state := newPlayerState("player_mail_gold", "MailGold", "wei", "caocao", now)
	if err := repo.CreatePlayer(account.ID, state, now); err != nil {
		t.Fatalf("create player: %v", err)
	}

	mailEvents := []GameEvent{}
	rewardEvents := []GameEvent{}
	svc.SubscribeEvent(EventMailClaimed, func(event GameEvent) {
		mailEvents = append(mailEvents, event)
	})
	svc.SubscribeEvent(EventRewardGranted, func(event GameEvent) {
		rewardEvents = append(rewardEvents, event)
	})

	mail, err := svc.SendMail(SendMailRequest{
		PlayerID:   state.Player.ID,
		MailType:   "reward",
		SenderType: "gm",
		Title:      "金币测试",
		Content:    "测试账号金币附件",
		Attachments: []MailAttachment{{
			Type:   RewardTypeGold,
			ItemID: RewardTypeGold,
			Amount: 7,
		}},
	})
	if err != nil {
		t.Fatalf("SendMail failed: %v", err)
	}
	result, err := svc.ClaimMailAttachments(state.Player.ID, mail.ID)
	if err != nil {
		t.Fatalf("ClaimMailAttachments failed: %v", err)
	}
	if result.AccountGold != 12 || result.GrantedItems[RewardTypeGold] != 7 {
		t.Fatalf("unexpected mail gold claim result: %+v", result)
	}
	currentAccount, err := repo.GetAccountByID(account.ID)
	if err != nil {
		t.Fatalf("GetAccountByID failed: %v", err)
	}
	if currentAccount.Gold != 12 {
		t.Fatalf("expected account gold 12, got %d", currentAccount.Gold)
	}
	entries, err := svc.ListGoldLedger(GoldLedgerFilter{AccountID: account.ID, RefType: LedgerRefMailClaim})
	if err != nil {
		t.Fatalf("ListGoldLedger failed: %v", err)
	}
	if len(entries) != 1 || entries[0].Currency != LedgerCurrencyGold || entries[0].Amount != 7 || entries[0].BalanceAfter != 12 {
		t.Fatalf("unexpected account gold ledger: %+v", entries)
	}
	if len(mailEvents) != 1 || mailEvents[0].Type != EventMailClaimed {
		t.Fatalf("expected mail claimed event, got %+v", mailEvents)
	}
	if len(rewardEvents) != 1 || rewardEvents[0].Type != EventRewardGranted {
		t.Fatalf("expected one reward event from core reward pipeline, got %+v", rewardEvents)
	}
}

func TestGMSendMailAppearsInPlayerMailbox(t *testing.T) {
	svc := NewService()
	repo := svc.repo.(*MemoryRepository)
	now := time.Now()
	account := Account{ID: "account_gm_mail", Username: "gm_mail", PasswordHash: "hash", CreatedAt: now}
	if err := repo.CreateAccount(account); err != nil {
		t.Fatalf("create account: %v", err)
	}
	state := newPlayerState("player_gm_mail", "MailTarget", "wei", "caocao", now)
	if err := repo.CreatePlayer(account.ID, state, now); err != nil {
		t.Fatalf("create player: %v", err)
	}

	mail, err := svc.SendMail(SendMailRequest{
		PlayerID:   state.Player.ID,
		MailType:   "gm_notice",
		SenderType: "gm",
		Title:      "GM 测试",
		Content:    "这是一封 GM 测试信函",
	})
	if err != nil {
		t.Fatalf("SendMail failed: %v", err)
	}
	page, err := svc.ListMails(state.Player.ID, 1, 10, "")
	if err != nil {
		t.Fatalf("ListMails failed: %v", err)
	}
	if page.Total != 1 || page.Unread != 1 || len(page.Mails) != 1 {
		t.Fatalf("unexpected mailbox page: %+v", page)
	}
	if page.Mails[0].ID != mail.ID || page.Mails[0].Title != "GM 测试" {
		t.Fatalf("unexpected listed mail: %+v", page.Mails[0])
	}
}

func TestCreatePlayerSendsNewPlayerRewardMail(t *testing.T) {
	setTestFactionsAndGenerals(t, FactionsConfig{
		"wei": {Name: "魏国", Generals: []GeneralInfo{{ID: "caocao", Name: "曹操"}}},
	}, GeneralsConfig{
		Enabled: true,
		Heroes: map[string]GeneralHeroConfig{
			"caocao": {ID: "caocao", Name: "曹操", Faction: "wei", Enabled: true},
		},
	})
	svc := NewService()
	repo := svc.repo.(*MemoryRepository)
	now := time.Now()
	account := Account{ID: "account_new_player_reward", Username: "new_player_reward", PasswordHash: "hash", CreatedAt: now}
	if err := repo.CreateAccount(account); err != nil {
		t.Fatalf("create account: %v", err)
	}

	playerID, _, err := svc.CreatePlayer(account.ID, "新手奖励测试", "wei", "caocao")
	if err != nil {
		t.Fatalf("CreatePlayer failed: %v", err)
	}
	page, err := svc.ListMails(playerID, 1, 10, newPlayerRewardMailType)
	if err != nil {
		t.Fatalf("ListMails failed: %v", err)
	}
	if page.Total != 1 || len(page.Mails) != 1 {
		t.Fatalf("expected one new player reward mail, got %+v", page)
	}
	mail := page.Mails[0]
	if mail.Title != "新手奖励" || mail.SenderType != "system" || mail.IsClaimed || len(mail.Attachments) != 1 {
		t.Fatalf("unexpected new player reward mail: %+v", mail)
	}
	attachment := mail.Attachments[0]
	if attachment.Type != RewardTypeGold || attachment.ItemID != RewardTypeGold || attachment.Amount != newPlayerRewardGold {
		t.Fatalf("unexpected new player reward attachment: %+v", attachment)
	}

	claimed, err := svc.ClaimMailAttachments(playerID, mail.ID)
	if err != nil {
		t.Fatalf("ClaimMailAttachments failed: %v", err)
	}
	if claimed.AccountGold != newPlayerRewardGold || claimed.GrantedItems[RewardTypeGold] != newPlayerRewardGold {
		t.Fatalf("unexpected new player reward claim: %+v", claimed)
	}
	currentAccount, err := repo.GetAccountByID(account.ID)
	if err != nil {
		t.Fatalf("GetAccountByID failed: %v", err)
	}
	if currentAccount.Gold != newPlayerRewardGold {
		t.Fatalf("expected account gold %d, got %d", newPlayerRewardGold, currentAccount.Gold)
	}
}

func TestSendServerBroadcastMailDeductsCityGoldAndDeliversToAllPlayers(t *testing.T) {
	svc := NewService()
	repo := svc.repo.(*MemoryRepository)
	now := time.Now()
	senderAccount := Account{ID: "account_broadcast_sender", Username: "broadcast_sender", PasswordHash: "hash", CreatedAt: now}
	receiverAccount := Account{ID: "account_broadcast_receiver", Username: "broadcast_receiver", PasswordHash: "hash", CreatedAt: now}
	if err := repo.CreateAccount(senderAccount); err != nil {
		t.Fatalf("create sender account: %v", err)
	}
	if err := repo.CreateAccount(receiverAccount); err != nil {
		t.Fatalf("create receiver account: %v", err)
	}
	sender := newPlayerState("player_broadcast_sender", "喊话者", "wei", "caocao", now)
	sender.CityGold = FlexInt(ServerBroadcastCost)
	sender.Player.MailCode = "111111"
	receiver := newPlayerState("player_broadcast_receiver", "听众", "wei", "caocao", now)
	receiver.Player.MailCode = "222222"
	if err := repo.CreatePlayer(senderAccount.ID, sender, now); err != nil {
		t.Fatalf("create sender: %v", err)
	}
	if err := repo.CreatePlayer(receiverAccount.ID, receiver, now); err != nil {
		t.Fatalf("create receiver: %v", err)
	}

	result, err := svc.SendServerBroadcastMail(SendServerBroadcastMailRequest{
		SenderPlayerID: sender.Player.ID,
		Title:          "全服集合",
		Content:        "此乃全服喊话",
	})
	if err != nil {
		t.Fatalf("SendServerBroadcastMail failed: %v", err)
	}
	if result.Cost != ServerBroadcastCost || int(result.CityGold) != 0 || result.RecipientCount != 2 {
		t.Fatalf("unexpected broadcast result: %+v", result)
	}
	currentSender, err := repo.GetState(sender.Player.ID)
	if err != nil {
		t.Fatalf("GetState sender failed: %v", err)
	}
	if int(currentSender.CityGold) != 0 {
		t.Fatalf("expected sender city gold 0, got %d", currentSender.CityGold)
	}
	for _, playerID := range []string{sender.Player.ID, receiver.Player.ID} {
		page, err := svc.ListMails(playerID, 1, 10, ServerBroadcastMailType)
		if err != nil {
			t.Fatalf("ListMails failed: %v", err)
		}
		if page.Total != 1 || len(page.Mails) != 1 {
			t.Fatalf("expected one broadcast mail for %s, got %+v", playerID, page)
		}
		if page.Mails[0].Title != "全服集合" || page.Mails[0].MailType != ServerBroadcastMailType {
			t.Fatalf("unexpected broadcast mail: %+v", page.Mails[0])
		}
	}
	entries, err := svc.ListGoldLedger(GoldLedgerFilter{PlayerID: sender.Player.ID, RefType: LedgerRefServerBroadcast})
	if err != nil {
		t.Fatalf("ListGoldLedger failed: %v", err)
	}
	if len(entries) != 1 || entries[0].Amount != ServerBroadcastCost || entries[0].BalanceAfter != 0 {
		t.Fatalf("unexpected broadcast ledger: %+v", entries)
	}
}

func TestSendServerBroadcastMailRejectsInsufficientCityGold(t *testing.T) {
	svc := NewService()
	repo := svc.repo.(*MemoryRepository)
	now := time.Now()
	account := Account{ID: "account_broadcast_poor", Username: "broadcast_poor", PasswordHash: "hash", CreatedAt: now}
	if err := repo.CreateAccount(account); err != nil {
		t.Fatalf("create account: %v", err)
	}
	state := newPlayerState("player_broadcast_poor", "贫穷喊话者", "wei", "caocao", now)
	state.CityGold = FlexInt(ServerBroadcastCost - 1)
	if err := repo.CreatePlayer(account.ID, state, now); err != nil {
		t.Fatalf("create player: %v", err)
	}

	if _, err := svc.SendServerBroadcastMail(SendServerBroadcastMailRequest{
		SenderPlayerID: state.Player.ID,
		Title:          "发不起",
		Content:        "城金不足",
	}); !errors.Is(err, ErrInsufficientCityGold) {
		t.Fatalf("expected ErrInsufficientCityGold, got %v", err)
	}
	page, err := svc.ListMails(state.Player.ID, 1, 10, ServerBroadcastMailType)
	if err != nil {
		t.Fatalf("ListMails failed: %v", err)
	}
	if page.Total != 0 {
		t.Fatalf("expected no broadcast mail, got %+v", page)
	}
}

func TestApplyRewardsBuildsEventsAndLedgerIntent(t *testing.T) {
	now := time.Now()
	state := newPlayerState("player_reward_core", "RewardCore", "wei", "caocao", now)

	result, err := ApplyRewardsToStateWithContext(&state, []Reward{
		{Type: RewardTypeCityGold, ID: RewardTypeCityGold, Amount: 25},
	}, RewardGrantContext{
		AccountID: "account_reward_core",
		PlayerID:  "player_reward_core",
		RefType:   "test_reward",
		RefID:     "reward_1",
		Reason:    "核心奖励测试",
	}, now)
	if err != nil {
		t.Fatalf("ApplyRewardsToStateWithContext failed: %v", err)
	}
	if int(state.CityGold) != 25 {
		t.Fatalf("expected city gold 25, got %d", state.CityGold)
	}
	if result.Granted[RewardTypeCityGold] != 25 {
		t.Fatalf("expected granted city gold 25, got %d", result.Granted[RewardTypeCityGold])
	}
	if len(result.Events) != 1 || result.Events[0].Type != EventRewardGranted {
		t.Fatalf("expected reward event, got %+v", result.Events)
	}
	if len(result.LedgerEntries) != 1 {
		t.Fatalf("expected one ledger intent, got %+v", result.LedgerEntries)
	}
	entry := result.LedgerEntries[0]
	if entry.Currency != LedgerCurrencyCityGold || entry.Direction != LedgerDirectionCredit || entry.BalanceAfter != 25 {
		t.Fatalf("unexpected ledger intent: %+v", entry)
	}
}

func TestUpdateAccountPlayerStateCommitsAccountAndPlayerTogether(t *testing.T) {
	repo := NewMemoryRepository()
	now := time.Now()
	account := Account{ID: "account_asset_tx", Username: "asset_tx", PasswordHash: "hash", Gold: 100, CreatedAt: now}
	if err := repo.CreateAccount(account); err != nil {
		t.Fatalf("create account: %v", err)
	}
	state := newPlayerState("player_asset_tx", "AssetTx", "wei", "caocao", now)
	if err := repo.CreatePlayer(account.ID, state, now); err != nil {
		t.Fatalf("create player: %v", err)
	}

	nextAccount, nextState, err := repo.UpdateAccountPlayerState(account.ID, state.Player.ID, now, func(account *Account, state *GameState) error {
		account.Gold -= 30
		state.CityGold += 300
		return nil
	})
	if err != nil {
		t.Fatalf("UpdateAccountPlayerState failed: %v", err)
	}
	if nextAccount.Gold != 70 || int(nextState.CityGold) != 300 {
		t.Fatalf("unexpected transaction result: account=%+v state.cityGold=%d", nextAccount, nextState.CityGold)
	}
}

func TestUpdateAccountPlayerStateRollsBackOnError(t *testing.T) {
	repo := NewMemoryRepository()
	now := time.Now()
	account := Account{ID: "account_asset_tx_rollback", Username: "asset_tx_rollback", PasswordHash: "hash", Gold: 100, CreatedAt: now}
	if err := repo.CreateAccount(account); err != nil {
		t.Fatalf("create account: %v", err)
	}
	state := newPlayerState("player_asset_tx_rollback", "AssetTxRollback", "wei", "caocao", now)
	if err := repo.CreatePlayer(account.ID, state, now); err != nil {
		t.Fatalf("create player: %v", err)
	}

	_, _, err := repo.UpdateAccountPlayerState(account.ID, state.Player.ID, now, func(account *Account, state *GameState) error {
		account.Gold -= 30
		state.CityGold += 300
		return ErrInsufficientGold
	})
	if !errors.Is(err, ErrInsufficientGold) {
		t.Fatalf("expected ErrInsufficientGold, got %v", err)
	}
	currentAccount, err := repo.GetAccountByID(account.ID)
	if err != nil {
		t.Fatalf("GetAccountByID failed: %v", err)
	}
	currentState, err := repo.GetState(state.Player.ID)
	if err != nil {
		t.Fatalf("GetState failed: %v", err)
	}
	if currentAccount.Gold != 100 || int(currentState.CityGold) != 0 {
		t.Fatalf("expected rollback, got account.gold=%d cityGold=%d", currentAccount.Gold, currentState.CityGold)
	}
}

func TestGrantRewardsUsesAccountTransactionLedgerAndEvents(t *testing.T) {
	svc := NewService()
	repo := svc.repo.(*MemoryRepository)
	now := time.Now()
	account := Account{ID: "account_grant_rewards", Username: "grant_rewards", PasswordHash: "hash", Gold: 100, CreatedAt: now}
	if err := repo.CreateAccount(account); err != nil {
		t.Fatalf("create account: %v", err)
	}
	state := newPlayerState("player_grant_rewards", "GrantRewards", "wei", "caocao", now)
	if err := repo.CreatePlayer(account.ID, state, now); err != nil {
		t.Fatalf("create player: %v", err)
	}

	events := []GameEvent{}
	svc.SubscribeEvent(EventRewardGranted, func(event GameEvent) {
		events = append(events, event)
	})

	result, err := svc.GrantRewards(state.Player.ID, []Reward{
		{Type: RewardTypeCityGold, ID: RewardTypeCityGold, Amount: 20},
		{Type: RewardTypeGold, ID: RewardTypeGold, Amount: 5},
	}, RewardGrantContext{
		AccountID: account.ID,
		RefType:   "test_grant",
		RefID:     "grant_1",
		Reason:    "奖励编排测试",
	})
	if err != nil {
		t.Fatalf("GrantRewards failed: %v", err)
	}
	if int(result.State.CityGold) != 20 || result.Account.Gold != 105 {
		t.Fatalf("unexpected grant result: state.cityGold=%d account.gold=%d", result.State.CityGold, result.Account.Gold)
	}
	if len(events) != 2 {
		t.Fatalf("expected 2 reward events, got %+v", events)
	}
	entries, err := svc.ListGoldLedger(GoldLedgerFilter{AccountID: account.ID, RefType: "test_grant"})
	if err != nil {
		t.Fatalf("ListGoldLedger failed: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 ledger entries, got %+v", entries)
	}
}

func TestGrantRewardsAppliesGeneralExp(t *testing.T) {
	svc := NewService()
	repo := svc.repo.(*MemoryRepository)
	now := time.Now()
	account := Account{ID: "account_grant_general_exp", Username: "grant_general_exp", PasswordHash: "hash", CreatedAt: now}
	if err := repo.CreateAccount(account); err != nil {
		t.Fatalf("create account: %v", err)
	}
	state := newPlayerState("player_grant_general_exp", "GrantGeneralExp", "wei", "caocao", now)
	if err := repo.CreatePlayer(account.ID, state, now); err != nil {
		t.Fatalf("create player: %v", err)
	}

	result, err := svc.GrantRewards(state.Player.ID, []Reward{
		{Type: RewardTypeGeneralExp, ID: "current_general", Amount: 100},
	}, RewardGrantContext{RefType: "test_general_exp", RefID: "grant_exp_1"})
	if err != nil {
		t.Fatalf("GrantRewards general_exp failed: %v", err)
	}
	if result.State.General == nil || result.State.General.Exp != 100 {
		t.Fatalf("expected general exp 100, got %+v", result.State.General)
	}
	if result.Apply.Granted[RewardTypeGeneralExp] != 100 {
		t.Fatalf("expected granted general_exp 100, got %+v", result.Apply.Granted)
	}
}

func TestGrantRewardsAppliesBuff(t *testing.T) {
	svc := NewService()
	repo := svc.repo.(*MemoryRepository)
	now := time.Now()
	account := Account{ID: "account_grant_buff", Username: "grant_buff", PasswordHash: "hash", CreatedAt: now}
	if err := repo.CreateAccount(account); err != nil {
		t.Fatalf("create account: %v", err)
	}
	state := newPlayerState("player_grant_buff", "GrantBuff", "wei", "caocao", now)
	if err := repo.CreatePlayer(account.ID, state, now); err != nil {
		t.Fatalf("create player: %v", err)
	}

	result, err := svc.GrantRewards(state.Player.ID, []Reward{
		{
			Type:   RewardTypeBuff,
			ID:     StatAttackBonus,
			Amount: 1,
			Metadata: map[string]any{
				"value": 0.2,
				"mode":  "percentAdd",
				"hours": 2,
			},
		},
	}, RewardGrantContext{RefType: "test_buff", RefID: "grant_buff_1"})
	if err != nil {
		t.Fatalf("GrantRewards buff failed: %v", err)
	}
	if len(result.State.Buffs) != 1 {
		t.Fatalf("expected one buff, got %+v", result.State.Buffs)
	}
	buff := result.State.Buffs[0]
	if buff.Key != StatAttackBonus || math.Abs(buff.Value-0.2) > 1e-9 || buff.Mode != "percentAdd" || buff.ExpiresAt == "" {
		t.Fatalf("unexpected buff: %+v", buff)
	}
}

func TestGetStateUsesReadonlyRepositoryWithoutRepairingCoreAssets(t *testing.T) {
	svc := NewService()
	repo := svc.repo.(*MemoryRepository)
	now := time.Now()
	account := Account{ID: "account_construction_slots", Username: "construction_slots", PasswordHash: "hash", CreatedAt: now}
	if err := repo.CreateAccount(account); err != nil {
		t.Fatalf("create account: %v", err)
	}
	state := newPlayerState("player_construction_slots", "ConstructionSlots", "wei", "caocao", now)
	state.Player.MailCode = "CS0001"
	state.Buildings = []Building{{ID: "warehouse-1", Type: "warehouse", Level: 1}}
	state.ResourceSlots = nil
	if err := repo.CreatePlayer(account.ID, state, now); err != nil {
		t.Fatalf("create player: %v", err)
	}
	storedBefore := repo.players[state.Player.ID]
	updatedAtBefore := repo.playerUpdatedAt[state.Player.ID]

	next, err := svc.GetState(state.Player.ID)
	if err != nil {
		t.Fatalf("GetState failed: %v", err)
	}
	if findBuildingByID(&next, "construction_bureau-1") != nil {
		t.Fatalf("expected readonly GetState not to repair missing construction bureau")
	}
	if findBuildingByID(&next, "construction_resource_slot-1") != nil {
		t.Fatalf("expected readonly GetState not to repair construction resource slot")
	}
	storedAfter := repo.players[state.Player.ID]
	if !updatedAtBefore.Equal(repo.playerUpdatedAt[state.Player.ID]) {
		t.Fatalf("expected readonly GetState not to update stored timestamp")
	}
	if countConstructionResourceSlots(storedAfter.Buildings) != countConstructionResourceSlots(storedBefore.Buildings) {
		t.Fatalf("expected readonly GetState not to mutate stored buildings")
	}
}

func TestConstructionBureauUpgradeUsesAccountGold(t *testing.T) {
	svc := NewService()
	repo := svc.repo.(*MemoryRepository)
	now := time.Now()
	account := Account{ID: "account_construction_gold", Username: "construction_gold", PasswordHash: "hash", Gold: 100, CreatedAt: now}
	if err := repo.CreateAccount(account); err != nil {
		t.Fatalf("create account: %v", err)
	}
	state := newPlayerState("player_construction_gold", "ConstructionGold", "wei", "caocao", now)
	state.Player.MailCode = "CG0001"
	if err := repo.CreatePlayer(account.ID, state, now); err != nil {
		t.Fatalf("create player: %v", err)
	}

	next, err := svc.UpgradeBuilding(state.Player.ID, "construction_bureau-1")
	if err != nil {
		t.Fatalf("UpgradeBuilding construction bureau failed: %v", err)
	}
	accountAfter, err := repo.GetAccountByID(account.ID)
	if err != nil {
		t.Fatalf("GetAccountByID failed: %v", err)
	}
	if accountAfter.Gold != 88 {
		t.Fatalf("expected account gold 88 after level 1 upgrade cost 12, got %d", accountAfter.Gold)
	}
	building := findBuildingByID(&next, "construction_bureau-1")
	if building == nil || building.UpgradeEndsAt == nil || building.Status != BuildingStatusUpgrading {
		t.Fatalf("expected construction bureau upgrading, got %+v", building)
	}
	entries, err := svc.ListGoldLedger(GoldLedgerFilter{AccountID: account.ID, RefType: LedgerRefBuildingGoldUpgrade})
	if err != nil {
		t.Fatalf("ListGoldLedger failed: %v", err)
	}
	if len(entries) != 1 || entries[0].Amount != 12 || entries[0].BalanceAfter != 88 {
		t.Fatalf("unexpected construction gold ledger: %+v", entries)
	}
}

func TestGetStateProjectsCompletedConstructionBureauUpgradeWithoutPersisting(t *testing.T) {
	svc := NewService()
	repo := svc.repo.(*MemoryRepository)
	now := time.Now()
	account := Account{ID: "account_construction_unlock", Username: "construction_unlock", PasswordHash: "hash", CreatedAt: now}
	if err := repo.CreateAccount(account); err != nil {
		t.Fatalf("create account: %v", err)
	}
	state := newPlayerState("player_construction_unlock", "ConstructionUnlock", "wei", "caocao", now)
	state.Player.MailCode = "CU0001"
	for i := range state.Buildings {
		if state.Buildings[i].Type == "construction_bureau" {
			state.Buildings[i].Level = 4
			endsAt := now.Add(-time.Minute).UTC().Format(resourceDateLayout)
			state.Buildings[i].UpgradeEndsAt = &endsAt
			state.Buildings[i].Status = BuildingStatusUpgrading
		}
	}
	if err := repo.CreatePlayer(account.ID, state, now); err != nil {
		t.Fatalf("create player: %v", err)
	}
	storedBefore := repo.players[state.Player.ID]

	next, err := svc.GetState(state.Player.ID)
	if err != nil {
		t.Fatalf("GetState failed: %v", err)
	}
	building := findBuildingByID(&next, "construction_bureau-1")
	if building == nil || building.Level != 5 || building.UpgradeEndsAt != nil {
		t.Fatalf("expected construction bureau completed at level 5, got %+v", building)
	}
	if countConstructionResourceSlots(next.Buildings) != 4 {
		t.Fatalf("expected four construction resource slots after level 5, got %+v", next.Buildings)
	}
	storedAfter := repo.players[state.Player.ID]
	buildingAfter := findBuildingByID(&storedAfter, "construction_bureau-1")
	if buildingAfter == nil || buildingAfter.Level != 4 || buildingAfter.UpgradeEndsAt == nil || buildingAfter.Status != BuildingStatusUpgrading {
		t.Fatalf("expected readonly GetState not to persist completed upgrade, before=%+v after=%+v", storedBefore.Buildings, storedAfter.Buildings)
	}
}

func TestRepairPlayerCoreAssetsRepairsLegacyStateExplicitly(t *testing.T) {
	svc := NewService()
	repo := svc.repo.(*MemoryRepository)
	now := time.Now()
	account := Account{ID: "account_repair_core", Username: "repair_core", PasswordHash: "hash", CreatedAt: now}
	if err := repo.CreateAccount(account); err != nil {
		t.Fatalf("create account: %v", err)
	}
	state := newPlayerState("player_repair_core", "RepairCore", "wei", "caocao", now)
	state.Player.MailCode = ""
	state.Buildings = []Building{{ID: "warehouse-1", Type: "warehouse", Level: 1}}
	state.ResourceSlots = nil
	state.Inventory = nil
	state.General = nil
	state.Generals = nil
	state.GeneralAssignments = nil
	if err := repo.CreatePlayer(account.ID, state, now); err != nil {
		t.Fatalf("create player: %v", err)
	}

	readonly, err := svc.GetState(state.Player.ID)
	if err != nil {
		t.Fatalf("GetState failed: %v", err)
	}
	if findBuildingByID(&readonly, "construction_bureau-1") != nil || readonly.General != nil {
		t.Fatalf("expected readonly GetState not to repair legacy state, got building=%+v general=%+v", readonly.Buildings, readonly.General)
	}

	repaired, err := svc.RepairPlayerCoreAssets(state.Player.ID)
	if err != nil {
		t.Fatalf("RepairPlayerCoreAssets failed: %v", err)
	}
	if !repaired.Changed {
		t.Fatalf("expected repair to report changes")
	}
	if findBuildingByID(&repaired.State, "construction_bureau-1") == nil {
		t.Fatalf("expected repair to add construction bureau")
	}
	if findBuildingByID(&repaired.State, "construction_resource_slot-1") != nil {
		t.Fatalf("expected level 1 construction bureau not to add construction resource slot")
	}
	if repaired.State.General == nil || repaired.State.General.ID == "" {
		t.Fatalf("expected repair to add default general")
	}
	if len(repaired.State.Generals) == 0 || len(repaired.State.GeneralAssignments) == 0 {
		t.Fatalf("expected repair to add general roster and assignment, got generals=%+v assignments=%+v", repaired.State.Generals, repaired.State.GeneralAssignments)
	}
	if repaired.State.Inventory == nil {
		t.Fatalf("expected repair to initialize inventory")
	}
	stored := repo.players[state.Player.ID]
	if findBuildingByID(&stored, "construction_bureau-1") == nil || stored.General == nil {
		t.Fatalf("expected explicit repair to persist core assets, got %+v", stored)
	}
}

func countConstructionResourceSlots(buildings []Building) int {
	count := 0
	for _, building := range buildings {
		if strings.HasPrefix(building.ID, "construction_resource_slot-") {
			count++
		}
	}
	return count
}

func loadRecruitItemProductionConfig(t *testing.T) {
	t.Helper()
	originalUnits := GetUnitsConfig()
	originalItems := GetItemsConfig()
	root := filepath.Join("..", "..", "..", "config")
	if err := LoadUnitsConfig(filepath.Join(root, "units")); err != nil {
		t.Fatalf("LoadUnitsConfig failed: %v", err)
	}
	if err := LoadItemsConfig(filepath.Join(root, "items.json")); err != nil {
		t.Fatalf("LoadItemsConfig failed: %v", err)
	}
	t.Cleanup(func() {
		unitsMu.Lock()
		activeUnits = originalUnits
		unitsMu.Unlock()
		itemsMu.Lock()
		itemsConfig = originalItems
		itemsMu.Unlock()
	})
}

func loadTestItemsConfig(t *testing.T) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "items.json")
	data := []byte(`{
		"test_general_exp_small": {
			"id": "test_general_exp_small",
			"name": "测试经验包",
			"description": "测试用",
			"type": "consumable",
			"rarity": "common",
			"usable": true,
			"stackable": true,
			"maxStack": 999999,
			"useTarget": "current_general",
			"effects": [
				{ "type": "general_exp", "amount": 100 }
			]
		}
	}`)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("write item config: %v", err)
	}
	if err := LoadItemsConfig(path); err != nil {
		t.Fatalf("load item config: %v", err)
	}
	t.Cleanup(func() {
		_ = LoadItemsConfig(filepath.Join("..", "..", "..", "config", "items.json"))
	})
}

// blockingAssetRepository 用于验证同玩家资产事务在服务层被串行化。
type blockingAssetRepository struct {
	*MemoryRepository
	active    int32
	maxActive int32
	entered   chan string
	release   chan struct{}
}

func (r *blockingAssetRepository) UpdateCombatState(playerID string, scope CombatAssetScope, updatedAt time.Time, update func(state *GameState) error) (GameState, error) {
	r.enterAssetTransaction("combat")
	defer r.leaveAssetTransaction()
	return r.MemoryRepository.UpdateCombatState(playerID, scope, updatedAt, update)
}

func (r *blockingAssetRepository) UpdateGeneralExpItemState(playerID string, itemID string, updatedAt time.Time, update func(state *GameState) error) (GameState, error) {
	r.enterAssetTransaction("general_exp_item")
	defer r.leaveAssetTransaction()
	return r.MemoryRepository.UpdateGeneralExpItemState(playerID, itemID, updatedAt, update)
}

func (r *blockingAssetRepository) enterAssetTransaction(name string) {
	active := atomic.AddInt32(&r.active, 1)
	for {
		maxActive := atomic.LoadInt32(&r.maxActive)
		if active <= maxActive || atomic.CompareAndSwapInt32(&r.maxActive, maxActive, active) {
			break
		}
	}
	r.entered <- name
	<-r.release
}

func (r *blockingAssetRepository) leaveAssetTransaction() {
	atomic.AddInt32(&r.active, -1)
}
