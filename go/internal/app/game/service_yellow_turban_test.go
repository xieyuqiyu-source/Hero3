// 本文件验证黄巾起义口粮压力、派兵规则、增援和将领特性结算。
package game

import (
	"errors"
	"math"
	"testing"
	"time"

	"hero3/internal/core/general"
)

type rejectStandaloneReportBundleRepository struct {
	*MemoryRepository
}

// SaveReportBundle 模拟事务外战报保存失败，黄巾结算不应再依赖该路径。
func (r *rejectStandaloneReportBundleRepository) SaveReportBundle(_ BattleEvent, _ []BattleReport) error {
	return errors.New("standalone report save rejected")
}

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
	generalsCfg := GetGeneralsConfig()
	liubeiCfg := generalsCfg.Heroes["liubei"]
	liubeiCfg.BonusTrait.Params["triggerChance"] = 1
	generalsCfg.Heroes["liubei"] = liubeiCfg
	if err := SetGeneralsConfig(generalsCfg); err != nil {
		t.Fatalf("SetGeneralsConfig failed: %v", err)
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
	baselineExp := generalExpRequiredForLevelForTest(2) - 1
	setPvpTestGeneralProgress(&state, "liubei", 1, baselineExp)
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
	if report.GeneralLevelBefore != 1 || report.GeneralLevelAfter <= report.GeneralLevelBefore {
		t.Fatalf("expected yellow turban defense to level up from level 1, got before=%d after=%d", report.GeneralLevelBefore, report.GeneralLevelAfter)
	}
	if report.Detail == nil || report.Detail.Rewards.GeneralExp != report.GeneralExpGained ||
		report.Detail.Rewards.GeneralLevelBefore != report.GeneralLevelBefore || report.Detail.Rewards.GeneralLevelAfter != report.GeneralLevelAfter {
		t.Fatalf("expected standard report detail to show general exp, detail=%+v reportExp=%d", report.Detail, report.GeneralExpGained)
	}
	if report.Detail.OwnerSide != ReportOwnerSideDefender || report.Detail.PrimarySide.Role != "attacker" ||
		report.Detail.SecondarySide == nil || report.Detail.SecondarySide.Role != "defender" ||
		len(report.Detail.SecondarySide.Generals) != 1 || report.Detail.SecondarySide.Generals[0].ID != "liubei" {
		t.Fatalf("expected defense rewards to belong to secondary defender general, detail=%+v", report.Detail)
	}
	updated, err := repo.GetState(state.Player.ID)
	if err != nil {
		t.Fatalf("GetState failed: %v", err)
	}
	if got := pvpTestGeneralExp(updated, "liubei"); got != baselineExp+report.GeneralExpGained {
		t.Fatalf("expected liubei cumulative exp %d, got %d", baselineExp+report.GeneralExpGained, got)
	}
	if got := pvpTestGeneralLevel(updated, "liubei"); got != report.GeneralLevelAfter {
		t.Fatalf("expected liubei stored level %d, got %d", report.GeneralLevelAfter, got)
	}
	lost := report.LostUnits["greedyWolf"]
	revived := report.RevivedUnits["greedyWolf"]
	expectedSurvived := 5000 - lost + revived
	if lost <= 0 || revived <= 0 || report.SurvivedUnits["greedyWolf"] != expectedSurvived {
		t.Fatalf("expected yellow turban Renzhu Shouhu survivor formula, lost=%d revived=%d survived=%+v", lost, revived, report.SurvivedUnits)
	}
	if got := armySliceToMap(updated.Army)["greedyWolf"]; got != expectedSurvived {
		t.Fatalf("expected real city army %d to match report, got %d", expectedSurvived, got)
	}
	if _, exists := report.TraitOutcomes["rende"]; exists {
		t.Fatalf("expected passive Rende to stay out of battle timeline, outcomes=%+v", report.TraitOutcomes)
	}
	guardOutcome := report.TraitOutcomes["renzhu_shouhu"]
	revivedUnits, ok := guardOutcome.Detail["revivedUnits"].(map[string]int)
	if !ok || revivedUnits["greedyWolf"] != int(math.Floor(float64(lost)*0.35)) {
		t.Fatalf("expected Renzhu Shouhu to revive 35%% of actual losses, got %+v", guardOutcome)
	}
	standardSurvived := -1
	if report.Detail.SecondarySide == nil {
		t.Fatal("expected yellow turban defender side in standard report")
	}
	for _, unit := range report.Detail.SecondarySide.Units {
		if unit.UnitType == "greedyWolf" {
			standardSurvived = unit.Survived
			break
		}
	}
	if standardSurvived != expectedSurvived {
		t.Fatalf("expected standard yellow turban survivor %d, got %d units=%+v", expectedSurvived, standardSurvived, report.Detail.SecondarySide.Units)
	}
}

// TestYellowTurbanLossReductionTraitsRespectOutcome 验证郭嘉在黄巾守城不同胜负下都按真实阵亡复活。
func TestYellowTurbanLossReductionTraitsRespectOutcome(t *testing.T) {
	cases := []struct {
		name      string
		traitID   string
		traitType string
	}{
		{name: "郭嘉鬼才遗策", traitID: "guicai_yice", traitType: general.TraitTypeBonus},
	}
	for traitIndex, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			traitCfg := GeneralTraitConfig{
				TraitID: tc.traitID, TraitType: tc.traitType, Enabled: true, Scope: "self_army",
				AllowedSides: []string{"attacker", "defender", "reinforcement"},
				Params:       map[string]float64{"effectRate": 0.5, "triggerChance": 1},
			}
			t.Run("战败触发", func(t *testing.T) {
				report, stored := resolveYellowTurbanTraitTest(t, traitCfg, 100, 200, "loss_"+string(rune('a'+traitIndex)))
				revived, ok := report.TraitOutcomes[tc.traitID].Detail["revivedUnits"].(map[string]int)
				if !ok || revived["weiInfantry"] != 50 {
					t.Fatalf("expected %s to revive 50 troops after loss, got %+v", tc.traitID, report.TraitOutcomes[tc.traitID])
				}
				if report.LostUnits["weiInfantry"] != 100 || report.RevivedUnits["weiInfantry"] != 50 || report.SurvivedUnits["weiInfantry"] != 50 {
					t.Fatalf("expected loss/return/survivor 100/50/50, got lost=%+v returned=%+v survived=%+v", report.LostUnits, report.RevivedUnits, report.SurvivedUnits)
				}
				if armySliceToMap(stored.Army)["weiInfantry"] != 50 || yellowTurbanStandardDefenderSurvived(report, "weiInfantry") != 50 {
					t.Fatalf("expected real army and standard report to keep 50 troops, army=%+v detail=%+v", stored.Army, report.Detail)
				}
			})
			t.Run("获胜触发", func(t *testing.T) {
				report, stored := resolveYellowTurbanTraitTest(t, traitCfg, 200, 100, "win_"+string(rune('a'+traitIndex)))
				lost := report.LostUnits["weiInfantry"]
				wantRevived := lost / 2
				outcome, triggered := report.TraitOutcomes[tc.traitID]
				revived, revivedOK := outcome.Detail["revivedUnits"].(map[string]int)
				if lost <= 0 || !triggered || !revivedOK || revived["weiInfantry"] != wantRevived || report.RevivedUnits["weiInfantry"] != wantRevived {
					t.Fatalf("expected %s to revive %d after defense victory, report=%+v", tc.traitID, wantRevived, report)
				}
				realSurvived := armySliceToMap(stored.Army)["weiInfantry"]
				if realSurvived != 200-lost+wantRevived || realSurvived != report.SurvivedUnits["weiInfantry"] || yellowTurbanStandardDefenderSurvived(report, "weiInfantry") != realSurvived {
					t.Fatalf("expected winning army and reports to agree, army=%d report=%+v detail=%+v", realSurvived, report.SurvivedUnits, report.Detail)
				}
			})
			t.Run("平局触发", func(t *testing.T) {
				report, stored := resolveYellowTurbanTraitTest(t, traitCfg, 100, 103, "draw_"+string(rune('a'+traitIndex)))
				if report.Result != "draw" || report.PlayerPower != 1030 || report.EnemyPower != 1030 || report.LostUnits["weiInfantry"] != 100 || report.DefenderLostUnits["weiInfantry"] != 103 {
					t.Fatalf("expected exact yellow turban draw 1030/1030 with attack-rule defender/attacker losses 100/103, report=%+v", report)
				}
				outcome, triggered := report.TraitOutcomes[tc.traitID]
				revived, revivedOK := outcome.Detail["revivedUnits"].(map[string]int)
				if !triggered || !revivedOK || revived["weiInfantry"] != 50 || report.RevivedUnits["weiInfantry"] != 50 ||
					len(report.TraitTriggered) != 1 || !standardReportHasTrait(report.Detail, tc.traitID) {
					t.Fatalf("expected %s to revive after yellow turban draw, report=%+v", tc.traitID, report)
				}
				if report.SurvivedUnits["weiInfantry"] != 50 || report.GeneralExpGained != 103 || armySliceToMap(stored.Army)["weiInfantry"] != 50 || yellowTurbanStandardDefenderSurvived(report, "weiInfantry") != 50 {
					t.Fatalf("expected yellow turban draw to keep 50 revived troops and exp 103, report=%+v stored=%+v", report, stored.Army)
				}
			})
		})
	}
}

// TestYellowTurbanDefensivePreBattleTraitsMatchPowerStateAndReports 验证四项守城战前特性真实改变黄巾战力并与最终状态、双方格式战报一致。
func TestYellowTurbanDefensivePreBattleTraitsMatchPowerStateAndReports(t *testing.T) {
	cases := []struct {
		name           string
		traitID        string
		traitType      string
		params         map[string]float64
		designKey      string
		designValue    float64
		attackChange   int
		infantryChange int
		cavalryChange  int
	}{
		{
			name: "谋定后发", traitID: "mouding_houfa", traitType: general.TraitTypeBonus,
			params:    map[string]float64{"defenseBonusRate": 0.35, "triggerChance": 1},
			designKey: "defenseBonusRate", designValue: 0.35, infantryChange: 4, cavalryChange: 3,
		},
		{
			name: "盾阵防御", traitID: "dunzhen_fangyu", traitType: general.TraitTypeBonus,
			params:    map[string]float64{"defenseBonusRate": 0.3, "triggerChance": 1},
			designKey: "defenseBonusRate", designValue: 0.3, infantryChange: 3, cavalryChange: 2,
		},
		{
			name: "固守汉中", traitID: "gushou_hanzhong", traitType: general.TraitTypeBonus,
			params:    map[string]float64{"generalDefenseFlat": 20, "triggerChance": 1},
			designKey: "generalDefenseFlat", designValue: 20, infantryChange: 20, cavalryChange: 20,
		},
		{
			name: "江东固守", traitID: "jiangdong_gushou", traitType: general.TraitTypeBonus,
			params:    map[string]float64{"defenseBonusRate": 0.5, "triggerChance": 1},
			designKey: "defenseBonusRate", designValue: 0.5, infantryChange: 5, cavalryChange: 4,
		},
	}

	for index, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			baseline, _ := resolveYellowTurbanTraitTest(t, GeneralTraitConfig{}, 100, 100, "pre_baseline_"+string(rune('a'+index)))
			traitCfg := GeneralTraitConfig{
				TraitID: tc.traitID, TraitType: tc.traitType, Enabled: true, Scope: "self_army",
				AllowedSides: []string{"defender", "reinforcement"}, Params: tc.params,
			}
			report, stored := resolveYellowTurbanTraitTest(t, traitCfg, 100, 100, "pre_trait_"+string(rune('a'+index)))

			if report.PlayerPower <= baseline.PlayerPower || report.EnemyPower != baseline.EnemyPower {
				t.Fatalf("expected %s to increase only city defense power, baseline=%d/%d actual=%d/%d", tc.traitID, baseline.EnemyPower, baseline.PlayerPower, report.EnemyPower, report.PlayerPower)
			}

			outcome, ok := report.TraitOutcomes[tc.traitID]
			if !ok || outcome.OwnerSide != "defender" || outcome.OwnerGeneralID != "test_general" || outcome.Detail[tc.designKey] != tc.designValue {
				t.Fatalf("expected defender-owned %s design result, outcome=%+v", tc.traitID, outcome)
			}
			if tc.attackChange != 0 {
				modified, detailOK := outcome.Detail["attackModifiedUnits"].(map[string]int)
				if !detailOK || modified["weiInfantry"] != tc.attackChange {
					t.Fatalf("expected %s actual attack change %d, outcome=%+v", tc.traitID, tc.attackChange, outcome)
				}
			} else {
				infantry, infantryOK := outcome.Detail["infantryDefenseModifiedUnits"].(map[string]int)
				cavalry, cavalryOK := outcome.Detail["cavalryDefenseModifiedUnits"].(map[string]int)
				if !infantryOK || !cavalryOK || infantry["weiInfantry"] != tc.infantryChange || cavalry["weiInfantry"] != tc.cavalryChange {
					t.Fatalf("expected %s actual defense changes %d/%d, outcome=%+v", tc.traitID, tc.infantryChange, tc.cavalryChange, outcome)
				}
			}

			standardFound := false
			if report.Detail != nil {
				for _, trait := range report.Detail.Traits {
					if trait.TraitID == tc.traitID {
						standardFound = trait.OwnerSide == "secondary" && trait.OwnerRole == "defender" && trait.GeneralID == "test_general" && trait.Detail[tc.designKey] == tc.designValue
					}
				}
			}
			if !standardFound {
				t.Fatalf("expected standard yellow turban report to preserve defender %s result, detail=%+v", tc.traitID, report.Detail)
			}
			remaining := armySliceToMap(stored.Army)["weiInfantry"]
			if remaining != report.SurvivedUnits["weiInfantry"] || yellowTurbanStandardDefenderSurvived(report, "weiInfantry") != remaining {
				t.Fatalf("expected %s real army, legacy and standard survivors to agree, army=%d legacy=%+v detail=%+v", tc.traitID, remaining, report.SurvivedUnits, report.Detail)
			}
		})
	}
}

// TestYellowTurbanSunQuanUsesDefenseTraitWithoutPlunderTrait 验证孙权守黄巾只触发加防，不误触发掠夺减益。
func TestYellowTurbanSunQuanUsesDefenseTraitWithoutPlunderTrait(t *testing.T) {
	baseline, _ := resolveYellowTurbanSunQuanTest(t, "baseline", false)
	report, stored := resolveYellowTurbanSunQuanTest(t, "trait", true)
	if baseline.PlayerPower != 1025 || report.PlayerPower != 1537 || baseline.EnemyPower != 1000 || report.EnemyPower != baseline.EnemyPower {
		t.Fatalf("expected Gushou to change the same wall baseline from 1025 to 1537 without changing enemy power, baseline=%d/%d actual=%d/%d", baseline.PlayerPower, baseline.EnemyPower, report.PlayerPower, report.EnemyPower)
	}
	gushou, triggered := report.TraitOutcomes["jiangdong_gushou"]
	infantry, infantryOK := gushou.Detail["infantryDefenseModifiedUnits"].(map[string]int)
	cavalry, cavalryOK := gushou.Detail["cavalryDefenseModifiedUnits"].(map[string]int)
	if !triggered || gushou.OwnerSide != "defender" || gushou.OwnerGeneralID != "sunquan" || !infantryOK || !cavalryOK || infantry["wuInfantry"] != 5 || cavalry["wuInfantry"] != 4 {
		t.Fatalf("expected Sun Quan defense outcome with +5/+4, got %+v", gushou)
	}
	if _, triggered := report.TraitOutcomes["jiangdong_haoling"]; triggered || standardReportHasTrait(report.Detail, "jiangdong_haoling") {
		t.Fatalf("yellow turban defense must not trigger plunder-only trait, outcomes=%+v detail=%+v", report.TraitOutcomes, report.Detail)
	}
	if report.Detail == nil || report.Detail.SecondarySide == nil || !standardReportHasTrait(report.Detail, "jiangdong_gushou") || len(report.Detail.Traits) != 1 ||
		!reportSideGeneralOwnsTrait(*report.Detail.SecondarySide, "sunquan", "jiangdong_gushou") || !reportSideGeneralOwnsTrait(*report.Detail.SecondarySide, "sunquan", "jiangdong_haoling") {
		t.Fatalf("expected dual owned traits but only Gushou in real timeline, detail=%+v", report.Detail)
	}
	remaining := armySliceToMap(stored.Army)["wuInfantry"]
	if remaining != report.SurvivedUnits["wuInfantry"] || yellowTurbanStandardDefenderSurvived(report, "wuInfantry") != remaining {
		t.Fatalf("expected real army and both report formats to agree, army=%d report=%+v detail=%+v", remaining, report.SurvivedUnits, report.Detail)
	}
	if stored.Resources.Items["wood"] != 1000 || report.GeneralExpGained <= 0 || pvpTestGeneralExp(stored, "sunquan") != report.GeneralExpGained {
		t.Fatalf("expected no plunder and exact defender exp, resources=%+v reportExp=%d storedExp=%d", stored.Resources.Items, report.GeneralExpGained, pvpTestGeneralExp(stored, "sunquan"))
	}
}

// TestYellowTurbanHomeGeneralUsesSettlementConfigEverywhere 验证黄巾到达时按最新主城将领配置结算。
func TestYellowTurbanHomeGeneralUsesSettlementConfigEverywhere(t *testing.T) {
	t.Run("行军创建时开启而结算前关闭", func(t *testing.T) {
		report, stored := resolveYellowTurbanSunQuanTest(t, "settlement_disabled", true, false)
		assertYellowTurbanSunQuanSettlementConfig(t, report, stored, false, 1025)
	})
	t.Run("行军创建时关闭而结算前开启", func(t *testing.T) {
		report, stored := resolveYellowTurbanSunQuanTest(t, "settlement_enabled", false, true)
		assertYellowTurbanSunQuanSettlementConfig(t, report, stored, true, 1537)
	})
}

// TestYellowTurbanHomeGeneralAssignmentUsesSettlementStateEverywhere 验证黄巾到达时按主将真实在城状态结算。
func TestYellowTurbanHomeGeneralAssignmentUsesSettlementStateEverywhere(t *testing.T) {
	t.Run("来袭途中离城后不参与守城", func(t *testing.T) {
		report, stored, reinforcement := resolveYellowTurbanSunQuanAssignmentTest(t, "away", false)
		assertYellowTurbanSunQuanAssignmentState(t, report, stored, false, 100, 1020)
		if reinforcement.Status != ReinforcementStatusMarching || len(reinforcement.Generals) != 1 || reinforcement.Generals[0].ID != "sunquan" {
			t.Fatalf("expected outbound Sun Quan reinforcement to remain separate, record=%+v", reinforcement)
		}
	})
	t.Run("来袭到达前归城后恢复守城", func(t *testing.T) {
		report, stored, reinforcement := resolveYellowTurbanSunQuanAssignmentTest(t, "returned", true)
		assertYellowTurbanSunQuanAssignmentState(t, report, stored, true, 101, 3090)
		if reinforcement.Status != ReinforcementStatusCompleted {
			t.Fatalf("expected reinforcement return completed before yellow turban settlement, record=%+v", reinforcement)
		}
	})
}

// TestYellowTurbanGeneralChangeUsesSettlementGeneralAndPreBattleSnapshot 验证黄巾途中正式换将后只使用新主将且快照保持战前等级。
func TestYellowTurbanGeneralChangeUsesSettlementGeneralAndPreBattleSnapshot(t *testing.T) {
	setTestCombatUnitsConfig(t)
	setTestFactionsAndGenerals(t, FactionsConfig{
		"wei": {Name: "魏国", Generals: []GeneralInfo{{ID: "simayi", Name: "司马懿"}, {ID: "xiahouyuan", Name: "夏侯渊"}}},
	}, GeneralsConfig{Enabled: true, Heroes: map[string]GeneralHeroConfig{
		"simayi": {
			ID: "simayi", Name: "司马懿", Faction: "wei", Enabled: true,
			SpecialTrait: GeneralTraitConfig{
				TraitID: "yibing_touxi", TraitType: general.TraitTypeSpecial, Enabled: true, Scope: "enemy_army",
				Params: map[string]float64{"triggerChance": 1, "effectRate": 0.35, "maxAffectedRate": 0.35},
			},
			BonusTrait: GeneralTraitConfig{
				TraitID: "mouding_houfa", TraitType: general.TraitTypeBonus, Enabled: true, Scope: "enemy_army", AllowedSides: []string{"defender"},
				Params: map[string]float64{"effectRate": 0.1},
			},
		},
		"xiahouyuan": {
			ID: "xiahouyuan", Name: "夏侯渊", Faction: "wei", Enabled: true,
			SpecialTrait: GeneralTraitConfig{
				TraitID: "jixing_benxi", TraitType: general.TraitTypeSpecial, Enabled: true, Scope: "self_army", TargetUnitType: "qiQiYing",
				Params: map[string]float64{"unitAttackFlat": 18, "unitSpeedFlat": 5},
			},
			BonusTrait: GeneralTraitConfig{
				TraitID: "dunzhen_fangyu", TraitType: general.TraitTypeBonus, Enabled: true, Scope: "self_army", AllowedSides: []string{"defender", "reinforcement"},
				Params: map[string]float64{"defenseBonusRate": 0.3, "triggerChance": 1},
			},
		},
	}})
	SetYellowTurbanConfig(defaultYellowTurbanConfig())
	repo := NewMemoryRepository()
	svc := NewServiceWithRepository(repo)
	now := time.Now().UTC()
	account := Account{ID: "account_yt_general_change", Username: "yt_general_change", PasswordHash: "hash", Gold: 20, CreatedAt: now}
	if err := repo.CreateAccount(account); err != nil {
		t.Fatalf("create yellow turban general-change account: %v", err)
	}
	state := newPlayerState("player_yt_general_change", "黄巾换将测试", "wei", "simayi", now)
	EnsureGeneralRoster(&state, now)
	baselineExp := generalExpRequiredForLevelForTest(2) - 1
	setPvpTestGeneralProgress(&state, "simayi", 1, baselineExp)
	state.Army = []ArmyUnit{{UnitType: "weiInfantry", Amount: 100}}
	if err := repo.CreatePlayer(account.ID, state, now); err != nil {
		t.Fatalf("create yellow turban general-change player: %v", err)
	}
	march, err := repo.CreateYellowTurbanMarch(YellowTurbanMarch{
		ID: "yt_general_change_march", TargetPlayerID: state.Player.ID, SourceCityID: "yt_general_change_source",
		SourceName: "黄巾军", SourceFaction: "wei", SourceRegionID: "wei", RiskLevelID: 1, RiskLevelName: "黄巾测试",
		PlayerFood: 1000, FoodCapacity: 100, Pressure: 10, Troops: map[string]int{"weiInfantry": 100},
		Status: YellowTurbanMarchStatusMarching, DurationSeconds: 600,
		StartedAt: now.Format(resourceDateLayout), ArrivesAt: now.Add(10 * time.Minute).Format(resourceDateLayout),
		CreatedAt: now.Format(resourceDateLayout), UpdatedAt: now.Format(resourceDateLayout),
	})
	if err != nil {
		t.Fatalf("create yellow turban general-change march: %v", err)
	}
	changed, err := svc.ChangeGeneral(state.Player.ID, "xiahouyuan", "")
	if err != nil {
		t.Fatalf("change yellow turban defender to Xiahou Yuan: %v", err)
	}
	if changed.State.General == nil || changed.State.General.ID != "xiahouyuan" || mainAssignedGeneralID(changed.State.GeneralAssignments) != "xiahouyuan" || changed.AccountGold != 20-GeneralChangeGoldCost {
		t.Fatalf("expected formal general change to Xiahou Yuan, result=%+v", changed)
	}
	if _, err := repo.UpdateYellowTurbanMarch(march.ID, time.Now().UTC(), func(current *YellowTurbanMarch) error {
		current.ArrivesAt = time.Now().UTC().Add(-time.Minute).Format(resourceDateLayout)
		return nil
	}); err != nil {
		t.Fatalf("force yellow turban general-change march due: %v", err)
	}
	report, err := svc.ResolveYellowTurbanMarch(march.ID)
	if err != nil {
		t.Fatalf("resolve yellow turban general-change march: %v", err)
	}
	if report.EnemyPower != 1000 || report.PlayerPower != 1339 {
		t.Fatalf("expected only Xiahou Yuan +30%% defense on 1000/1339 power, got %d/%d", report.EnemyPower, report.PlayerPower)
	}
	if _, ok := report.TraitOutcomes["dunzhen_fangyu"]; !ok || standardReportHasTrait(report.Detail, "jixing_benxi") {
		t.Fatalf("expected only Xiahou Yuan battle trait, outcomes=%+v detail=%+v", report.TraitOutcomes, report.Detail)
	}
	for _, staleTraitID := range []string{"yibing_touxi", "mouding_houfa"} {
		if _, ok := report.TraitOutcomes[staleTraitID]; ok || standardReportHasTrait(report.Detail, staleTraitID) {
			t.Fatalf("expected old Sima Yi trait %s absent after formal change, report=%+v", staleTraitID, report)
		}
	}
	if len(report.PvpDefenderGenerals) != 1 || report.PvpDefenderGenerals[0].ID != "xiahouyuan" || report.PvpDefenderGenerals[0].Level != 1 ||
		report.Detail == nil || report.Detail.SecondarySide == nil || len(report.Detail.SecondarySide.Generals) != 1 || report.Detail.SecondarySide.Generals[0].ID != "xiahouyuan" || report.Detail.SecondarySide.Generals[0].Level != 1 {
		t.Fatalf("expected only pre-battle Lv.1 Xiahou Yuan in both snapshots, report=%+v", report)
	}
	if report.GeneralExpGained <= 0 || report.GeneralLevelBefore != 1 || report.GeneralLevelAfter <= 1 || report.Detail.Rewards.GeneralLevelBefore != 1 || report.Detail.Rewards.GeneralLevelAfter != report.GeneralLevelAfter {
		t.Fatalf("expected Xiahou Yuan upgrade only in reward fields, report=%+v", report)
	}
	stored, err := repo.GetState(state.Player.ID)
	if err != nil {
		t.Fatalf("get yellow turban general-change state: %v", err)
	}
	if got := pvpTestGeneralExp(stored, "xiahouyuan"); got != baselineExp+report.GeneralExpGained {
		t.Fatalf("expected Xiahou Yuan exp %d, got %d", baselineExp+report.GeneralExpGained, got)
	}
	if got := pvpTestGeneralExp(stored, "simayi"); got != baselineExp {
		t.Fatalf("expected old Sima Yi exp unchanged at %d, got %d", baselineExp, got)
	}
	remaining := armySliceToMap(stored.Army)["weiInfantry"]
	if remaining != report.SurvivedUnits["weiInfantry"] || yellowTurbanStandardDefenderSurvived(report, "weiInfantry") != remaining {
		t.Fatalf("expected changed-general battle state and reports to reconcile, army=%d report=%+v detail=%+v", remaining, report.SurvivedUnits, report.Detail)
	}
}

// assertYellowTurbanSunQuanAssignmentState 核对主将在城状态对战力、经验、快照、时间线和兵力的一致影响。
func assertYellowTurbanSunQuanAssignmentState(t *testing.T, report BattleReport, stored GameState, expectedHome bool, expectedBefore int, expectedPower int) {
	t.Helper()
	if report.PlayerPower != expectedPower || report.EnemyPower != 1000 || report.DispatchedUnits["shuInfantry"] != expectedBefore {
		t.Fatalf("expected home=%v power/before %d/1000/%d, got %d/%d/%+v", expectedHome, expectedPower, expectedBefore, report.PlayerPower, report.EnemyPower, report.DispatchedUnits)
	}
	_, hasOutcome := report.TraitOutcomes["jiangdong_gushou"]
	hasLegacySnapshot := len(report.PvpDefenderGenerals) == 1 && report.PvpDefenderGenerals[0].ID == "sunquan"
	hasStandardSnapshot := report.Detail != nil && standardDetailHasGeneral(report.Detail, "sunquan")
	hasTimeline := report.Detail != nil && standardReportHasTrait(report.Detail, "jiangdong_gushou")
	if hasOutcome != expectedHome || hasLegacySnapshot != expectedHome || hasStandardSnapshot != expectedHome || hasTimeline != expectedHome {
		t.Fatalf("expected home=%v across outcome and report snapshots, report=%+v", expectedHome, report)
	}
	if generalAvailableAtHome(stored.GeneralAssignments, "sunquan") != expectedHome {
		t.Fatalf("expected stored Sun Quan home=%v, assignments=%+v", expectedHome, stored.GeneralAssignments)
	}
	if expectedHome {
		if report.GeneralExpGained <= 0 || pvpTestGeneralExp(stored, "sunquan") != report.GeneralExpGained || report.Detail == nil || report.Detail.Rewards.GeneralExp != report.GeneralExpGained {
			t.Fatalf("expected returned Sun Quan exp to match state and standard report, report=%+v stored=%+v", report, stored.Generals)
		}
	} else if report.GeneralExpGained != 0 || pvpTestGeneralExp(stored, "sunquan") != 0 || (report.Detail != nil && report.Detail.Rewards.GeneralExp != 0) {
		t.Fatalf("expected away Sun Quan to gain no defense exp, report=%+v stored=%+v", report, stored.Generals)
	}
	remaining := armySliceToMap(stored.Army)["shuInfantry"]
	wantRemaining := expectedBefore - report.LostUnits["shuInfantry"]
	if remaining != wantRemaining || report.SurvivedUnits["shuInfantry"] != remaining || yellowTurbanStandardDefenderSurvived(report, "shuInfantry") != remaining {
		t.Fatalf("expected home=%v army and both report formats to reconcile, want=%d army=%d legacy=%+v detail=%+v", expectedHome, wantRemaining, remaining, report.SurvivedUnits, report.Detail)
	}
}

// resolveYellowTurbanSunQuanAssignmentTest 用真实增援往返流程构造孙权离城或归城后的黄巾结算。
func resolveYellowTurbanSunQuanAssignmentTest(t *testing.T, suffix string, returnHome bool) (BattleReport, GameState, Reinforcement) {
	t.Helper()
	defenseTrait := GeneralTraitConfig{
		TraitID: "jiangdong_gushou", TraitType: general.TraitTypeBonus, Enabled: true,
		Scope: "self_army", AllowedSides: []string{"defender", "reinforcement"},
		Params: map[string]float64{"defenseBonusRate": 0.5, "triggerChance": 1},
	}
	setTestFactionsAndGenerals(t, FactionsConfig{
		"wei": {Name: "魏国", Generals: []GeneralInfo{{ID: "caocao", Name: "曹操"}}},
		"shu": {Name: "蜀国", Generals: []GeneralInfo{{ID: "sunquan", Name: "孙权"}}},
	}, GeneralsConfig{Enabled: true, Heroes: map[string]GeneralHeroConfig{
		"caocao": {ID: "caocao", Name: "曹操", Faction: "wei", Enabled: true},
		"sunquan": {
			ID: "sunquan", Name: "孙权", Faction: "shu", Enabled: true,
			Buffs: map[string]float64{StatDefenseBonus: 1}, BonusTrait: defenseTrait,
		},
	}})
	SetYellowTurbanConfig(defaultYellowTurbanConfig())
	svc, repo, host, defender := newPvpTestServiceForGenerals(t, "wei", "caocao", "shu", "sunquan")
	defender.Army = []ArmyUnit{{UnitType: "shuInfantry", Amount: 101}}
	repo.players[host.Player.ID] = host
	repo.players[defender.Player.ID] = defender
	now := time.Now().UTC()
	march, err := repo.CreateYellowTurbanMarch(YellowTurbanMarch{
		ID: "yt_assignment_march_" + suffix, TargetPlayerID: defender.Player.ID, SourceCityID: "yt_assignment_source_" + suffix,
		SourceName: "黄巾军", SourceFaction: "wei", SourceRegionID: "wei", RiskLevelID: 1, RiskLevelName: "黄巾测试",
		PlayerFood: 1000, FoodCapacity: 100, Pressure: 10, Troops: map[string]int{"weiInfantry": 100},
		Status: YellowTurbanMarchStatusMarching, DurationSeconds: 600,
		StartedAt: now.Format(resourceDateLayout), ArrivesAt: now.Add(10 * time.Minute).Format(resourceDateLayout),
		CreatedAt: now.Format(resourceDateLayout), UpdatedAt: now.Format(resourceDateLayout),
	})
	if err != nil {
		t.Fatalf("create yellow turban assignment march: %v", err)
	}
	sent, err := svc.SendReinforcement(SendReinforcementRequest{
		FromPlayerID: defender.Player.ID, TargetPlayerID: host.Player.ID,
		Troops: map[string]int{"shuInfantry": 1}, GeneralIDs: []string{"sunquan"},
	})
	if err != nil {
		t.Fatalf("send Sun Quan away during yellow turban march: %v", err)
	}
	if returnHome {
		forceReinforcementDue(t, repo, sent.Reinforcement.ID, true)
		if _, err := svc.MarkReinforcementArrived(sent.Reinforcement.ID); err != nil {
			t.Fatalf("mark Sun Quan reinforcement arrived: %v", err)
		}
		if _, err := svc.RecallReinforcement(defender.Player.ID, sent.Reinforcement.ID); err != nil {
			t.Fatalf("recall Sun Quan reinforcement: %v", err)
		}
		forceReinforcementDue(t, repo, sent.Reinforcement.ID, false)
		if _, err := svc.CompleteReinforcementReturn(sent.Reinforcement.ID); err != nil {
			t.Fatalf("complete Sun Quan reinforcement return: %v", err)
		}
	}
	if _, err := repo.UpdateYellowTurbanMarch(march.ID, time.Now().UTC(), func(current *YellowTurbanMarch) error {
		current.ArrivesAt = time.Now().UTC().Add(-time.Minute).Format(resourceDateLayout)
		return nil
	}); err != nil {
		t.Fatalf("force yellow turban assignment march due: %v", err)
	}
	report, err := svc.ResolveYellowTurbanMarch(march.ID)
	if err != nil {
		t.Fatalf("resolve yellow turban assignment march: %v", err)
	}
	stored, err := repo.GetState(defender.Player.ID)
	if err != nil {
		t.Fatalf("get yellow turban assignment defender: %v", err)
	}
	reinforcement, err := repo.GetReinforcement(sent.Reinforcement.ID)
	if err != nil {
		t.Fatalf("get yellow turban assignment reinforcement: %v", err)
	}
	return report, stored, reinforcement
}

// assertYellowTurbanSunQuanSettlementConfig 核对最新配置对应的战力、快照、时间线和真实兵力。
func assertYellowTurbanSunQuanSettlementConfig(t *testing.T, report BattleReport, stored GameState, expected bool, expectedPower int) {
	t.Helper()
	if report.PlayerPower != expectedPower || report.EnemyPower != 1000 {
		t.Fatalf("expected settlement power %d/1000, got %d/%d", expectedPower, report.PlayerPower, report.EnemyPower)
	}
	_, hasOutcome := report.TraitOutcomes["jiangdong_gushou"]
	hasSnapshot := len(report.PvpDefenderGenerals) == 1 && pvpSnapshotHasTrait(report.PvpDefenderGenerals[0], "jiangdong_gushou")
	hasTimeline := report.Detail != nil && standardReportHasTrait(report.Detail, "jiangdong_gushou")
	hasStandardSnapshot := report.Detail != nil && report.Detail.SecondarySide != nil && reportSideGeneralOwnsTrait(*report.Detail.SecondarySide, "sunquan", "jiangdong_gushou")
	if hasOutcome != expected || hasSnapshot != expected || hasTimeline != expected || hasStandardSnapshot != expected {
		t.Fatalf("expected settlement trait=%v across outcome and snapshots, report=%+v", expected, report)
	}
	remaining := armySliceToMap(stored.Army)["wuInfantry"]
	if remaining != report.SurvivedUnits["wuInfantry"] || yellowTurbanStandardDefenderSurvived(report, "wuInfantry") != remaining {
		t.Fatalf("expected settlement state and reports to reconcile, army=%d report=%+v detail=%+v", remaining, report.SurvivedUnits, report.Detail)
	}
}

// resolveYellowTurbanSunQuanTest 使用相同公共加成构造开启或关闭江东固守的真实黄巾守城对照。
func resolveYellowTurbanSunQuanTest(t *testing.T, suffix string, defenseEnabled bool, settlementEnabled ...bool) (BattleReport, GameState) {
	t.Helper()
	setTestCombatUnitsConfig(t)
	unitsMu.Lock()
	activeUnits["wu"] = FactionUnits{
		"wuInfantry": UnitConfig{
			Name: "吴步兵", Category: "infantry",
			Stats: map[string]int{"attack": 10, "infantryDefense": 10, "cavalryDefense": 8, "carryCapacity": 5, "upkeep": 1},
		},
	}
	unitsMu.Unlock()
	setTestFactionsAndGenerals(t, FactionsConfig{
		"wu": {Name: "吴国", Generals: []GeneralInfo{{ID: "sunquan", Name: "孙权"}}},
	}, GeneralsConfig{Enabled: true, Heroes: map[string]GeneralHeroConfig{
		"sunquan": {
			ID: "sunquan", Name: "孙权", Faction: "wu", Enabled: true,
			SpecialTrait: GeneralTraitConfig{
				TraitID: "jiangdong_haoling", TraitType: general.TraitTypeSpecial, Enabled: true,
				Scope: "enemy_plunder", AllowedSides: []string{"defender"}, AllowedScenes: []string{"plunder"},
				Params: map[string]float64{"plunderBonusRate": -0.2},
			},
			BonusTrait: GeneralTraitConfig{
				TraitID: "jiangdong_gushou", TraitType: general.TraitTypeBonus, Enabled: defenseEnabled,
				Scope: "self_army", AllowedSides: []string{"defender", "reinforcement"},
				Params: map[string]float64{"triggerChance": 1, "defenseBonusRate": 0.5},
			},
		},
	}})
	SetYellowTurbanConfig(defaultYellowTurbanConfig())
	repo := NewMemoryRepository()
	svc := NewServiceWithRepository(repo)
	now := time.Now().UTC()
	account := Account{ID: "account_yt_sunquan_" + suffix, Username: "yt_sunquan_" + suffix, CreatedAt: now}
	if err := repo.CreateAccount(account); err != nil {
		t.Fatalf("create account: %v", err)
	}
	state := newPlayerState("player_yt_sunquan_"+suffix, "孙权守城", "wu", "sunquan", now)
	EnsureGeneralRoster(&state, now)
	state.Army = []ArmyUnit{{UnitType: "wuInfantry", Amount: 100}}
	state.Resources.Items["wood"] = 1000
	if err := repo.CreatePlayer(account.ID, state, now); err != nil {
		t.Fatalf("create player: %v", err)
	}
	march, err := repo.CreateYellowTurbanMarch(YellowTurbanMarch{
		ID: "yt_sunquan_march_" + suffix, TargetPlayerID: state.Player.ID, SourceCityID: "yt_sunquan_source_" + suffix,
		SourceName: "黄巾军", SourceFaction: "wei", SourceRegionID: "wei", RiskLevelID: 1, RiskLevelName: "黄巾测试",
		PlayerFood: 1000, FoodCapacity: 100, Pressure: 10, Troops: map[string]int{"weiInfantry": 100},
		Status: YellowTurbanMarchStatusMarching, DurationSeconds: 1,
		StartedAt: now.Add(-2 * time.Minute).Format(resourceDateLayout), ArrivesAt: now.Add(-time.Minute).Format(resourceDateLayout),
		CreatedAt: now.Add(-2 * time.Minute).Format(resourceDateLayout), UpdatedAt: now.Add(-2 * time.Minute).Format(resourceDateLayout),
	})
	if err != nil {
		t.Fatalf("create yellow turban march: %v", err)
	}
	if len(settlementEnabled) > 0 {
		settlementConfig := GetGeneralsConfig()
		settlementSunQuan := settlementConfig.Heroes["sunquan"]
		settlementSunQuan.BonusTrait.Enabled = settlementEnabled[0]
		settlementConfig.Heroes["sunquan"] = settlementSunQuan
		if err := SetGeneralsConfig(settlementConfig); err != nil {
			t.Fatalf("switch Sun Quan trait before yellow turban settlement: %v", err)
		}
	}

	report, err := svc.ResolveYellowTurbanMarch(march.ID)
	if err != nil {
		t.Fatalf("resolve yellow turban march: %v", err)
	}
	stored, err := repo.GetState(state.Player.ID)
	if err != nil {
		t.Fatalf("get state: %v", err)
	}
	return report, stored
}

// TestYellowTurbanSimaYiStacksPreDamageBeforeDefenseBonus 验证黄巾战力使用疑兵伤亡后的兵力和谋定加防后的属性。
func TestYellowTurbanSimaYiStacksPreDamageBeforeDefenseBonus(t *testing.T) {
	buildHero := func(enabled bool) GeneralHeroConfig {
		return GeneralHeroConfig{
			ID: "simayi", Name: "司马懿", Faction: "wei", Enabled: true,
			SpecialTrait: GeneralTraitConfig{
				TraitID: "yibing_touxi", TraitType: general.TraitTypeSpecial, Enabled: enabled, Scope: "enemy_army",
				Params: map[string]float64{"triggerChance": 1, "effectRate": 0.35},
			},
			BonusTrait: GeneralTraitConfig{
				TraitID: "mouding_houfa", TraitType: general.TraitTypeBonus, Enabled: enabled, Scope: "self_army",
				AllowedSides: []string{"defender", "reinforcement"}, Params: map[string]float64{"defenseBonusRate": 0.35, "triggerChance": 1},
			},
		}
	}
	baseline, _ := resolveYellowTurbanHeroTest(t, buildHero(false), 1000, 1000, "simayi_baseline")
	report, stored := resolveYellowTurbanHeroTest(t, buildHero(true), 1000, 1000, "simayi_traits")
	if baseline.EnemyPower != 10000 || report.EnemyPower != 6500 || baseline.PlayerPower != 10300 || report.PlayerPower != 14420 {
		t.Fatalf("expected 35%% direct enemy losses and 35%% owned defense bonus with wall, baseline=%d/%d actual=%d/%d", baseline.EnemyPower, baseline.PlayerPower, report.EnemyPower, report.PlayerPower)
	}
	preDamage, preDamageOK := report.TraitOutcomes["yibing_touxi"].Detail["preBattleAffected"].(map[string]int)
	infantryModified, infantryOK := report.TraitOutcomes["mouding_houfa"].Detail["infantryDefenseModifiedUnits"].(map[string]int)
	cavalryModified, cavalryOK := report.TraitOutcomes["mouding_houfa"].Detail["cavalryDefenseModifiedUnits"].(map[string]int)
	if !preDamageOK || preDamage["weiInfantry"] != 350 || !infantryOK || !cavalryOK || infantryModified["weiInfantry"] != 4 || cavalryModified["weiInfantry"] != 3 || report.TraitOutcomes["mouding_houfa"].Detail["defenseBonusRate"] != 0.35 {
		t.Fatalf("expected 350 real pre-damage then +4/+3 owned defense, outcomes=%+v", report.TraitOutcomes)
	}
	wantTimeline := []string{"yibing_touxi", "mouding_houfa"}
	if len(report.TraitTriggered) != len(wantTimeline) || report.TraitTriggered[0] != wantTimeline[0] || report.TraitTriggered[1] != wantTimeline[1] || report.Detail == nil || len(report.Detail.Traits) != len(wantTimeline) {
		t.Fatalf("expected special then bonus timeline, legacy=%+v detail=%+v", report.TraitTriggered, report.Detail)
	}
	for index, traitID := range wantTimeline {
		trait := report.Detail.Traits[index]
		if trait.TraitID != traitID || trait.OwnerSide != "secondary" || trait.OwnerRole != "defender" || trait.GeneralID != "simayi" {
			t.Fatalf("expected defender-owned timeline item %d to be %s, got %+v", index, traitID, trait)
		}
	}
	attackerLost := report.DefenderLostUnits["weiInfantry"]
	if attackerLost != 1000 || report.LostUnits["weiInfantry"] <= 0 || report.LostUnits["weiInfantry"] >= 1000 || report.SurvivedUnits["weiInfantry"] != 1000-report.LostUnits["weiInfantry"] || report.GeneralExpGained != 1000 || pvpTestGeneralExp(stored, "simayi") != 1000 {
		t.Fatalf("expected exact enemy elimination and reconciled boosted defenders, report=%+v storedExp=%d", report, pvpTestGeneralExp(stored, "simayi"))
	}
	attackerUnit := attackDefenseCrossReportUnit(t, report.Detail.PrimarySide, "weiInfantry")
	if attackerUnit.AmountBefore != 1000 || attackerUnit.Lost != attackerLost || attackerUnit.Survived != 1000-attackerLost {
		t.Fatalf("expected standard yellow turban losses to match actual enemy losses, unit=%+v", attackerUnit)
	}
	remaining := armySliceToMap(stored.Army)["weiInfantry"]
	if remaining != report.SurvivedUnits["weiInfantry"] || yellowTurbanStandardDefenderSurvived(report, "weiInfantry") != remaining {
		t.Fatalf("expected defender army and both report formats to agree, army=%d report=%+v detail=%+v", remaining, report.SurvivedUnits, report.Detail)
	}
	if report.Detail.SecondarySide == nil || !reportSideGeneralOwnsTrait(*report.Detail.SecondarySide, "simayi", "yibing_touxi") || !reportSideGeneralOwnsTrait(*report.Detail.SecondarySide, "simayi", "mouding_houfa") {
		t.Fatalf("expected Sima Yi hit snapshot to preserve both owned traits, detail=%+v", report.Detail)
	}
}

// TestYellowTurbanSimaYiYibingLegalMissKeepsMouding 验证疑兵合法未命中时谋定后发仍独立生效。
func TestYellowTurbanSimaYiYibingLegalMissKeepsMouding(t *testing.T) {
	hero := GeneralHeroConfig{
		ID: "simayi", Name: "司马懿", Faction: "wei", Enabled: true,
		SpecialTrait: GeneralTraitConfig{
			TraitID: "yibing_touxi", TraitType: general.TraitTypeSpecial, Enabled: true, Scope: "enemy_army",
			Params: map[string]float64{"triggerChance": 0, "effectRate": 0.35},
		},
		BonusTrait: GeneralTraitConfig{
			TraitID: "mouding_houfa", TraitType: general.TraitTypeBonus, Enabled: true, Scope: "self_army",
			AllowedSides: []string{"defender", "reinforcement"}, Params: map[string]float64{"defenseBonusRate": 0.35, "triggerChance": 1},
		},
	}
	report, stored := resolveYellowTurbanHeroTest(t, hero, 1000, 1000, "simayi_yibing_miss")
	if report.Result != "defender_victory" || report.EnemyPower != 10000 || report.PlayerPower != 14420 {
		t.Fatalf("expected Sima Yi miss to keep only 10000/14420 Mouding defense power, report=%+v", report)
	}
	if report.DefenderLostUnits["weiInfantry"] != 1000 || report.LostUnits["weiInfantry"] != 594 || report.SurvivedUnits["weiInfantry"] != 406 || report.GeneralExpGained != 1000 {
		t.Fatalf("expected exact Sima Yi miss losses/survivors/exp, report=%+v", report)
	}
	if armySliceToMap(stored.Army)["weiInfantry"] != 406 || yellowTurbanStandardDefenderSurvived(report, "weiInfantry") != 406 {
		t.Fatalf("expected Sima Yi miss state and standard report to keep 406 defenders, stored=%+v detail=%+v", stored.Army, report.Detail)
	}
	if report.Detail == nil || report.Detail.SecondarySide == nil || !reportSideGeneralOwnsTrait(*report.Detail.SecondarySide, "simayi", "yibing_touxi") || !reportSideGeneralOwnsTrait(*report.Detail.SecondarySide, "simayi", "mouding_houfa") {
		t.Fatalf("expected Sima Yi miss snapshot to preserve both traits, detail=%+v", report.Detail)
	}
	if len(report.TraitTriggered) != 1 || report.TraitTriggered[0] != "mouding_houfa" || len(report.TraitOutcomes) != 1 || len(report.Detail.Traits) != 1 || standardReportHasTrait(report.Detail, "yibing_touxi") {
		t.Fatalf("expected only Mouding after legal Yibing miss, report=%+v", report)
	}
	modified, ok := report.TraitOutcomes["mouding_houfa"].Detail["infantryDefenseModifiedUnits"].(map[string]int)
	if !ok || modified["weiInfantry"] != 4 {
		t.Fatalf("expected Mouding to increase owned infantry defense by 4, outcome=%+v", report.TraitOutcomes["mouding_houfa"])
	}
}

// TestYellowTurbanDianWeiHuzhuStrengthensJinwei 验证黄巾防守时护主血战加防，死战到底方向无效。
func TestYellowTurbanDianWeiHuzhuStrengthensJinwei(t *testing.T) {
	buildHero := func(enabled bool) GeneralHeroConfig {
		return GeneralHeroConfig{
			ID: "dianwei", Name: "典韦", Faction: "wei", Enabled: true,
			SpecialTrait: GeneralTraitConfig{
				TraitID: "huzhu_xuezhan", TraitType: general.TraitTypeSpecial, Enabled: enabled, Scope: "self_army", TargetUnitType: "jinWeiSoldier",
				AllowedSides: []string{"defender", "reinforcement"}, Params: map[string]float64{"triggerChance": 1, "generalDefenseFlat": 20},
			},
			BonusTrait: GeneralTraitConfig{
				TraitID: "sizhandaodi", TraitType: general.TraitTypeBonus, Enabled: true, Scope: "self_army", TargetUnitType: "infantry",
				AllowedSides: []string{"attacker"}, Params: map[string]float64{"triggerChance": 1, "attackBonusRate": 0.35},
			},
		}
	}
	control, _ := resolveYellowTurbanHeroFactionTest(t, buildHero(false), "wei", "jinWeiSoldier", 100, "wei", "weiInfantry", 200, "dianwei_huzhu_control")
	report, stored := resolveYellowTurbanHeroFactionTest(t, buildHero(true), "wei", "jinWeiSoldier", 100, "wei", "weiInfantry", 200, "dianwei_huzhu_active")
	if control.EnemyPower != 2000 || report.EnemyPower != 2000 || control.PlayerPower != 1339 || report.PlayerPower != 3399 {
		t.Fatalf("expected Huzhu plus Wei faction defense power 1339 -> 3399, control=%+v active=%+v", control, report)
	}
	controlLoss := control.LostUnits["jinWeiSoldier"]
	activeLoss := report.LostUnits["jinWeiSoldier"]
	if controlLoss != 100 || activeLoss != 47 || control.DefenderLostUnits["weiInfantry"] != 113 || report.DefenderLostUnits["weiInfantry"] != 200 {
		t.Fatalf("expected exact Huzhu losses 100/113 -> 47/200, control=%+v active=%+v", control, report)
	}
	if report.RevivedUnits["jinWeiSoldier"] != 0 || report.SurvivedUnits["jinWeiSoldier"] != 100-activeLoss || armySliceToMap(stored.Army)["jinWeiSoldier"] != 100-activeLoss {
		t.Fatalf("expected no obsolete return and authoritative survivors, report=%+v stored=%+v", report, stored.Army)
	}
	if report.Detail == nil || report.Detail.SecondarySide == nil || !reportSideGeneralOwnsTrait(*report.Detail.SecondarySide, "dianwei", "huzhu_xuezhan") || !reportSideGeneralOwnsTrait(*report.Detail.SecondarySide, "dianwei", "sizhandaodi") || standardReportHasTrait(report.Detail, "sizhandaodi") {
		t.Fatalf("expected current traits in snapshot and only Huzhu direction valid, report=%+v", report)
	}
	outcome := report.TraitOutcomes["huzhu_xuezhan"]
	infantry, infantryOK := outcome.Detail["infantryDefenseModifiedUnits"].(map[string]int)
	cavalry, cavalryOK := outcome.Detail["cavalryDefenseModifiedUnits"].(map[string]int)
	if !infantryOK || !cavalryOK || infantry["jinWeiSoldier"] != 20 || cavalry["jinWeiSoldier"] != 20 || len(report.TraitTriggered) != 1 || report.TraitTriggered[0] != "huzhu_xuezhan" || len(report.Detail.Traits) != 1 {
		t.Fatalf("expected only Huzhu +20/+20 in yellow turban timeline, report=%+v", report)
	}
}

// TestYellowTurbanZhaoYunAppliesLongdanWithoutMarchTraitTimeline 验证赵云守黄巾只触发龙胆减损，七进七出不伪装成战斗效果。
func TestYellowTurbanZhaoYunAppliesLongdanWithoutMarchTraitTimeline(t *testing.T) {
	buildHero := func(longdanEnabled bool) GeneralHeroConfig {
		return GeneralHeroConfig{
			ID: "zhaoyun", Name: "赵云", Faction: "shu", Enabled: true,
			SpecialTrait: GeneralTraitConfig{
				TraitID: "longdan_jiuyuan", TraitType: general.TraitTypeSpecial, Enabled: longdanEnabled,
				Scope: "reinforcement_self", AllowedSides: []string{"defender", "reinforcement"},
				Params: map[string]float64{"triggerChance": 1, "lossReductionRate": 0.2},
			},
			BonusTrait: GeneralTraitConfig{
				TraitID: "qijin_qichu", TraitType: general.TraitTypeBonus, Enabled: true, Scope: "self_army",
				Params: map[string]float64{"speedBonusRate": 1, "minMarchSeconds": 60},
			},
		}
	}
	control, _ := resolveYellowTurbanHeroFactionTest(t, buildHero(false), "shu", "greedyWolf", 100, "wei", "weiInfantry", 200, "zhaoyun_control")
	report, stored := resolveYellowTurbanHeroFactionTest(t, buildHero(true), "shu", "greedyWolf", 100, "wei", "weiInfantry", 200, "zhaoyun_longdan")
	if control.EnemyPower != 2000 || report.EnemyPower != 2000 || control.PlayerPower != 1020 || report.PlayerPower != 1020 || control.DefenderLostUnits["weiInfantry"] != 76 || report.DefenderLostUnits["weiInfantry"] != 76 {
		t.Fatalf("expected Longdan to preserve 2000/1020 power and 76 enemy losses, control=%d/%d %+v active=%d/%d %+v", control.EnemyPower, control.PlayerPower, control.DefenderLostUnits, report.EnemyPower, report.PlayerPower, report.DefenderLostUnits)
	}
	if control.LostUnits["greedyWolf"] != 100 || control.SurvivedUnits["greedyWolf"] != 0 || report.LostUnits["greedyWolf"] != 80 || report.SurvivedUnits["greedyWolf"] != 20 {
		t.Fatalf("expected Longdan to change real defender loss/survivor 100/0 -> 80/20, control=%+v/%+v active=%+v/%+v", control.LostUnits, control.SurvivedUnits, report.LostUnits, report.SurvivedUnits)
	}
	outcome, triggered := report.TraitOutcomes["longdan_jiuyuan"]
	reduced, reducedOK := outcome.Detail["reducedLosses"].(map[string]int)
	if !triggered || !reducedOK || reduced["greedyWolf"] != 20 || outcome.OwnerSide != "defender" || outcome.OwnerGeneralID != "zhaoyun" || outcome.Detail["lossReductionRate"] != 0.2 || outcome.Detail["triggerChance"] != 1.0 {
		t.Fatalf("expected defender Longdan to record real reduction 20 and formal design values, outcome=%+v", outcome)
	}
	if report.Detail == nil || report.Detail.SecondarySide == nil || len(report.TraitTriggered) != 1 || report.TraitTriggered[0] != "longdan_jiuyuan" || len(report.TraitOutcomes) != 1 || len(report.Detail.Traits) != 1 ||
		standardReportHasTrait(report.Detail, "qijin_qichu") || !reportSideGeneralOwnsTrait(*report.Detail.SecondarySide, "zhaoyun", "longdan_jiuyuan") || !reportSideGeneralOwnsTrait(*report.Detail.SecondarySide, "zhaoyun", "qijin_qichu") {
		t.Fatalf("expected dual owned traits but only Longdan battle timeline, legacy=%+v outcomes=%+v detail=%+v", report.TraitTriggered, report.TraitOutcomes, report.Detail)
	}
	trait := report.Detail.Traits[0]
	if trait.TraitID != "longdan_jiuyuan" || trait.OwnerSide != "secondary" || trait.OwnerRole != "defender" || trait.GeneralID != "zhaoyun" {
		t.Fatalf("expected standard Longdan to belong to defending Zhao Yun, trait=%+v", trait)
	}
	if armySliceToMap(stored.Army)["greedyWolf"] != 20 || yellowTurbanStandardDefenderSurvived(report, "greedyWolf") != 20 {
		t.Fatalf("expected player army and standard report to keep 20 troops, army=%+v detail=%+v", stored.Army, report.Detail)
	}
	if report.DefenderLostUnits["weiInfantry"] != control.DefenderLostUnits["weiInfantry"] || report.GeneralExpGained != report.DefenderLostUnits["weiInfantry"] || pvpTestGeneralExp(stored, "zhaoyun") != report.GeneralExpGained {
		t.Fatalf("expected Longdan not to alter enemy losses and Zhao Yun exp to match them, control=%+v active=%+v reportExp=%d storedExp=%d", control.DefenderLostUnits, report.DefenderLostUnits, report.GeneralExpGained, pvpTestGeneralExp(stored, "zhaoyun"))
	}
}

// TestYellowTurbanDefenderExtraDamageMatchesReport 验证防守方战后追加伤害进入黄巾损失和标准战报。
func TestYellowTurbanDefenderExtraDamageMatchesReport(t *testing.T) {
	traitCfg := GeneralTraitConfig{
		TraitID: "kurou_fanji", TraitType: general.TraitTypeBonus, Enabled: true, Scope: "enemy_army",
		Params: map[string]float64{"effectRate": 0.1, "triggerChance": 1},
	}
	report, _ := resolveYellowTurbanTraitTest(t, traitCfg, 100, 200, "counter_damage")
	outcome := report.TraitOutcomes["kurou_fanji"]
	extraLosses, ok := outcome.Detail["extraLosses"].(map[string]int)
	if !ok || extraLosses["weiInfantry"] != 20 {
		t.Fatalf("expected defender trait to add 20 yellow turban losses, got %+v", outcome)
	}
	if report.DefenderLostUnits["weiInfantry"] < 20 || report.Detail == nil {
		t.Fatalf("expected legacy and standard yellow turban attacker losses to agree, report=%+v detail=%+v", report.DefenderLostUnits, report.Detail)
	}
	assertStandardUnitRow(t, report.ID, report.Detail.PrimarySide, "weiInfantry", 200, report.DefenderLostUnits["weiInfantry"], 200-report.DefenderLostUnits["weiInfantry"])
}

// TestYellowTurbanFormalAfterCombatDamageTraitsMatchStateAndReports 验证四项正式守城战后伤害逐项进入黄巾损失与双方格式战报。
func TestYellowTurbanFormalAfterCombatDamageTraitsMatchStateAndReports(t *testing.T) {
	cases := []struct {
		name           string
		traitID        string
		traitType      string
		generalID      string
		generalName    string
		targetUnitType string
		rate           float64
		detailKey      string
	}{
		{name: "老当益壮", traitID: "laodang_yizhuang", traitType: general.TraitTypeBonus, generalID: "huangzhong", generalName: "黄忠", rate: 0.1, detailKey: "extraLosses"},
		{name: "火烧联营", traitID: "huoshao_lianying", traitType: general.TraitTypeSpecial, generalID: "luxun", generalName: "陆逊", targetUnitType: "infantry", rate: 1, detailKey: "targetExtraLosses"},
		{name: "连营增伤", traitID: "lianying_zengshang", traitType: general.TraitTypeBonus, generalID: "luxun", generalName: "陆逊", targetUnitType: "infantry", rate: 0.1, detailKey: "targetExtraLosses"},
		{name: "苦肉反击", traitID: "kurou_fanji", traitType: general.TraitTypeBonus, generalID: "huanggai", generalName: "黄盖", rate: 0.1, detailKey: "extraLosses"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			baseline, _ := resolveYellowTurbanTraitTest(t, GeneralTraitConfig{}, 100, 200, "formal_after_baseline_"+tc.traitID)
			traitCfg := GeneralTraitConfig{
				TraitID: tc.traitID, TraitType: tc.traitType, Enabled: true, Scope: "enemy_army", TargetUnitType: tc.targetUnitType,
				Params: map[string]float64{"effectRate": tc.rate, "maxAffectedRate": tc.rate, "triggerChance": 1},
			}
			report, stored := resolveYellowTurbanTraitForGeneralTest(t, traitCfg, 100, 200, "formal_after_"+tc.traitID, tc.generalID, tc.generalName)
			outcome, ok := report.TraitOutcomes[tc.traitID]
			extra, detailOK := outcome.Detail[tc.detailKey].(map[string]int)
			wantExtra := int(200 * tc.rate)
			if remaining := 200 - baseline.DefenderLostUnits["weiInfantry"]; wantExtra > remaining {
				wantExtra = remaining
			}
			if !ok || !detailOK || extra["weiInfantry"] != wantExtra || outcome.Detail["effectRate"] != tc.rate || outcome.Detail["triggerChance"] != 1.0 || outcome.OwnerSide != "defender" || outcome.OwnerGeneralID != tc.generalID {
				t.Fatalf("expected defender %s extra loss %d, outcome=%+v", tc.traitID, wantExtra, outcome)
			}
			if report.DefenderLostUnits["weiInfantry"] != baseline.DefenderLostUnits["weiInfantry"]+wantExtra {
				t.Fatalf("expected %s yellow turban loss %d + %d, baseline=%+v actual=%+v", tc.traitID, baseline.DefenderLostUnits["weiInfantry"], wantExtra, baseline.DefenderLostUnits, report.DefenderLostUnits)
			}
			if armySliceToMap(stored.Army)["weiInfantry"] != report.SurvivedUnits["weiInfantry"] || yellowTurbanStandardDefenderSurvived(report, "weiInfantry") != report.SurvivedUnits["weiInfantry"] {
				t.Fatalf("expected %s player army and reports to reconcile, stored=%+v report=%+v", tc.traitID, stored.Army, report)
			}
			standardFound := false
			for _, trait := range report.Detail.Traits {
				if trait.TraitID == tc.traitID && trait.OwnerSide == "secondary" && trait.OwnerRole == "defender" && trait.GeneralID == tc.generalID {
					standardExtra, standardOK := trait.Detail[tc.detailKey].(map[string]int)
					standardFound = standardOK && standardExtra["weiInfantry"] == wantExtra
				}
			}
			if !standardFound {
				t.Fatalf("expected standard yellow turban %s result, detail=%+v", tc.traitID, report.Detail)
			}
		})
	}
}

// TestYellowTurbanAttackOnlyAfterCombatTraitsDoNotTriggerOnDefense 验证火攻和小霸王追击不会在黄巾守城方向误触发。
func TestYellowTurbanAttackOnlyAfterCombatTraitsDoNotTriggerOnDefense(t *testing.T) {
	cases := []struct {
		name        string
		traitID     string
		generalID   string
		generalName string
		traitCfg    GeneralTraitConfig
	}{
		{name: "火攻", traitID: "huogong", generalID: "zhouyu", generalName: "周瑜", traitCfg: GeneralTraitConfig{TraitID: "huogong", TraitType: general.TraitTypeSpecial, Enabled: true, Scope: "enemy_army", AllowedSides: []string{"attacker"}, Params: map[string]float64{"effectRate": 0.25, "damagePercent": 0.25, "triggerChance": 1}}},
		{name: "小霸王追击", traitID: "xiaobawang_zhuiji", generalID: "sunce", generalName: "孙策", traitCfg: GeneralTraitConfig{TraitID: "xiaobawang_zhuiji", TraitType: general.TraitTypeSpecial, Enabled: true, Scope: "enemy_army", AllowedScenes: []string{"plunder"}, RequiredOutcome: "win", Params: map[string]float64{"effectRate": 0.1, "triggerChance": 1}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			report, stored := resolveYellowTurbanTraitForGeneralTest(t, tc.traitCfg, 100, 200, "formal_after_negative_"+tc.traitID, tc.generalID, tc.generalName)
			if _, triggered := report.TraitOutcomes[tc.traitID]; triggered {
				t.Fatalf("expected defender %s not to trigger in yellow turban battle, outcomes=%+v", tc.traitID, report.TraitOutcomes)
			}
			for _, trait := range report.Detail.Traits {
				if trait.TraitID == tc.traitID {
					t.Fatalf("expected no fake standard %s outcome, detail=%+v", tc.traitID, report.Detail)
				}
			}
			if armySliceToMap(stored.Army)["weiInfantry"] != report.SurvivedUnits["weiInfantry"] {
				t.Fatalf("expected negative %s player army to match report, stored=%+v report=%+v", tc.traitID, stored.Army, report)
			}
		})
	}
}

// TestYellowTurbanRandomPreBattleTraitsHitAndMiss 验证关羽、张飞守黄巾时随机战前能力独立判断概率与方向。
func TestYellowTurbanRandomPreBattleTraitsHitAndMiss(t *testing.T) {
	cases := []struct {
		name            string
		generalID       string
		generalName     string
		defenderFaction string
		defenderUnit    string
		attackerFaction string
		attackerUnit    string
		specialTraitID  string
		bonusTraitID    string
		bonusTarget     string
		bonusAttackRate float64
		effectRate      float64
		actualDetailKey string
		defensePower    int
		hitAttackPower  int
		hitDefenseLost  int
		hitAttackLost   int
		missDefenseLost int
	}{
		{
			name: "关羽水淹七军", generalID: "guanyu", generalName: "关羽",
			defenderFaction: "shu", defenderUnit: "shuInfantry", attackerFaction: "wei", attackerUnit: "weiInfantry",
			specialTraitID: "shuiyan_qijun", bonusTraitID: "wusheng_pojun", bonusAttackRate: 0.2, effectRate: 0.35, actualDetailKey: "preBattleAffected",
			defensePower: 1020, hitAttackPower: 650, hitDefenseLost: 52, hitAttackLost: 100, missDefenseLost: 97,
		},
		{
			name: "张飞震慑全军", generalID: "zhangfei", generalName: "张飞",
			defenderFaction: "shu", defenderUnit: "shuInfantry", attackerFaction: "wei", attackerUnit: "weiInfantry",
			specialTraitID: "zhenhe_quanjun", bonusTraitID: "wanren_nuhou", bonusTarget: "infantry", bonusAttackRate: 0.2, effectRate: 0.5, actualDetailKey: "suppressedUnits",
			defensePower: 1020, hitAttackPower: 500, hitDefenseLost: 36, hitAttackLost: 50, missDefenseLost: 97,
		},
	}

	for index, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			for _, triggerChance := range []float64{1, 0} {
				label := "命中"
				if triggerChance == 0 {
					label = "合法未命中"
				}
				t.Run(label, func(t *testing.T) {
					hero := GeneralHeroConfig{
						ID: tc.generalID, Name: tc.generalName, Faction: tc.defenderFaction, Enabled: true,
						SpecialTrait: GeneralTraitConfig{
							TraitID: tc.specialTraitID, TraitType: general.TraitTypeSpecial, Enabled: true, Scope: "enemy_army",
							Params: map[string]float64{"effectRate": tc.effectRate, "maxAffectedRate": tc.effectRate, "triggerChance": triggerChance},
						},
						BonusTrait: GeneralTraitConfig{
							TraitID: tc.bonusTraitID, TraitType: general.TraitTypeBonus, Enabled: true, Scope: "self_army", TargetUnitType: tc.bonusTarget,
							AllowedSides: []string{"attacker"}, Params: map[string]float64{"attackBonusRate": tc.bonusAttackRate, "triggerChance": 1},
						},
					}
					suffix := "random_prebattle_" + string(rune('a'+index)) + "_hit"
					if triggerChance == 0 {
						suffix = "random_prebattle_" + string(rune('a'+index)) + "_miss"
					}
					report, stored := resolveYellowTurbanHeroFactionTest(t, hero, tc.defenderFaction, tc.defenderUnit, 100, tc.attackerFaction, tc.attackerUnit, 100, suffix)

					wantAttackPower := 1000
					wantDefenseLost := tc.missDefenseLost
					wantAttackLost := 100
					if triggerChance == 1 {
						wantAttackPower = tc.hitAttackPower
						wantDefenseLost = tc.hitDefenseLost
						wantAttackLost = tc.hitAttackLost
					}
					wantDefenseSurvived := 100 - wantDefenseLost
					wantAttackSurvived := 100 - wantAttackLost
					if report.Result != "defender_victory" || report.EnemyPower != wantAttackPower || report.PlayerPower != tc.defensePower {
						t.Fatalf("expected exact yellow turban powers %d/%d and defender victory, report=%+v", wantAttackPower, tc.defensePower, report)
					}
					if report.LostUnits[tc.defenderUnit] != wantDefenseLost || report.DefenderLostUnits[tc.attackerUnit] != wantAttackLost || report.SurvivedUnits[tc.defenderUnit] != wantDefenseSurvived {
						t.Fatalf("expected exact defender/attacker losses %d/%d and defender survived %d, report=%+v", wantDefenseLost, wantAttackLost, wantDefenseSurvived, report)
					}
					if armySliceToMap(stored.Army)[tc.defenderUnit] != wantDefenseSurvived || report.GeneralExpGained != wantAttackLost || pvpTestGeneralExp(stored, tc.generalID) != wantAttackLost {
						t.Fatalf("expected authoritative army %d and general exp %d, state=%+v report=%+v", wantDefenseSurvived, wantAttackLost, stored, report)
					}
					if report.Detail == nil || report.Detail.SecondarySide == nil {
						t.Fatalf("expected complete standard yellow turban detail, report=%+v", report)
					}
					assertStandardUnitRow(t, report.ID, report.Detail.PrimarySide, tc.attackerUnit, 100, wantAttackLost, wantAttackSurvived)
					assertStandardUnitRow(t, report.ID, *report.Detail.SecondarySide, tc.defenderUnit, 100, wantDefenseLost, wantDefenseSurvived)
					if !reportSideGeneralOwnsTrait(*report.Detail.SecondarySide, tc.generalID, tc.specialTraitID) || !reportSideGeneralOwnsTrait(*report.Detail.SecondarySide, tc.generalID, tc.bonusTraitID) {
						t.Fatalf("expected defender snapshot to preserve both owned traits, side=%+v", report.Detail.SecondarySide)
					}
					if _, triggered := report.TraitOutcomes[tc.bonusTraitID]; triggered || standardReportHasTrait(report.Detail, tc.bonusTraitID) {
						t.Fatalf("attack-only trait %s must not trigger in yellow turban defense, report=%+v", tc.bonusTraitID, report)
					}

					outcome, triggered := report.TraitOutcomes[tc.specialTraitID]
					if triggerChance == 0 {
						if triggered || len(report.TraitTriggered) != 0 || len(report.TraitOutcomes) != 0 || len(report.Detail.Traits) != 0 {
							t.Fatalf("expected legal random miss to keep an empty timeline, report=%+v", report)
						}
						return
					}
					actual, actualOK := outcome.Detail[tc.actualDetailKey].(map[string]int)
					wantActual := int(100 * tc.effectRate)
					if !triggered || outcome.OwnerSide != "defender" || outcome.OwnerGeneralID != tc.generalID || outcome.Detail["effectRate"] != tc.effectRate || outcome.Detail["maxAffectedRate"] != tc.effectRate || outcome.Detail["triggerChance"] != 1.0 || !actualOK || actual[tc.attackerUnit] != wantActual {
						t.Fatalf("expected exact defender-owned %s result with actual %d, outcome=%+v", tc.specialTraitID, wantActual, outcome)
					}
					if len(report.Detail.Traits) != 1 || report.Detail.Traits[0].TraitID != tc.specialTraitID || report.Detail.Traits[0].OwnerSide != "secondary" || report.Detail.Traits[0].OwnerRole != "defender" || report.Detail.Traits[0].GeneralID != tc.generalID {
						t.Fatalf("expected one correctly owned standard trait result, traits=%+v", report.Detail.Traits)
					}
				})
			}
		})
	}
}

// TestYellowTurbanZhangLiaoAttackerOnlyTraitsDoNotDefend 验证张辽两项概率即使强制命中也不能在黄巾守城中触发。
func TestYellowTurbanZhangLiaoAttackerOnlyTraitsDoNotDefend(t *testing.T) {
	hero := GeneralHeroConfig{
		ID: "zhangliao", Name: "张辽", Faction: "wei", Enabled: true,
		SpecialTrait: GeneralTraitConfig{
			TraitID: "weizhen_zhenhe", TraitType: general.TraitTypeSpecial, Enabled: true, Scope: "enemy_army", AllowedSides: []string{"attacker"},
			Params: map[string]float64{"effectRate": 0.25, "triggerChance": 1},
		},
		BonusTrait: GeneralTraitConfig{
			TraitID: "weizhen_xiaoyao", TraitType: general.TraitTypeBonus, Enabled: true, Scope: "self_army", TargetUnitType: "cavalry", AllowedSides: []string{"attacker"},
			Params: map[string]float64{"attackBonusRate": 0.35, "triggerChance": 1},
		},
	}
	report, stored := resolveYellowTurbanHeroFactionTest(t, hero, "wei", "weiInfantry", 100, "shu", "shuInfantry", 100, "zhangliao_wrong_direction")
	if report.Result != "defender_victory" || report.EnemyPower != 1000 || report.PlayerPower != 1030 || report.LostUnits["weiInfantry"] != 95 || report.DefenderLostUnits["shuInfantry"] != 100 || report.SurvivedUnits["weiInfantry"] != 5 {
		t.Fatalf("expected baseline yellow turban defense without Zhang Liao traits, report=%+v", report)
	}
	if armySliceToMap(stored.Army)["weiInfantry"] != 5 || report.GeneralExpGained != 100 || pvpTestGeneralExp(stored, "zhangliao") != 100 {
		t.Fatalf("expected authoritative baseline army and exp, state=%+v report=%+v", stored, report)
	}
	if report.Detail == nil || report.Detail.SecondarySide == nil || !reportSideGeneralOwnsTrait(*report.Detail.SecondarySide, "zhangliao", "weizhen_zhenhe") || !reportSideGeneralOwnsTrait(*report.Detail.SecondarySide, "zhangliao", "weizhen_xiaoyao") {
		t.Fatalf("expected owned traits to remain in defender snapshot, detail=%+v", report.Detail)
	}
	if len(report.TraitTriggered) != 0 || len(report.TraitOutcomes) != 0 || len(report.Detail.Traits) != 0 {
		t.Fatalf("expected attacker-only traits absent from yellow turban timeline, report=%+v", report)
	}
}

// TestYellowTurbanLuXunAfterCombatTraitsHitAndMiss 验证陆逊守黄巾时火烧命中封顶且未命中不影响后序连营增伤。
func TestYellowTurbanLuXunAfterCombatTraitsHitAndMiss(t *testing.T) {
	for _, triggerChance := range []float64{1, 0} {
		label := "火烧命中"
		if triggerChance == 0 {
			label = "火烧合法未命中"
		}
		t.Run(label, func(t *testing.T) {
			hero := GeneralHeroConfig{
				ID: "luxun", Name: "陆逊", Faction: "wu", Enabled: true,
				SpecialTrait: GeneralTraitConfig{
					TraitID: "huoshao_lianying", TraitType: general.TraitTypeSpecial, Enabled: true, Scope: "enemy_army", TargetUnitType: "infantry",
					Params: map[string]float64{"effectRate": 1, "maxAffectedRate": 1, "triggerChance": triggerChance},
				},
				BonusTrait: GeneralTraitConfig{
					TraitID: "lianying_zengshang", TraitType: general.TraitTypeBonus, Enabled: true, Scope: "enemy_army", TargetUnitType: "infantry",
					Params: map[string]float64{"effectRate": 0.1, "maxAffectedRate": 0.1, "triggerChance": 1},
				},
			}
			suffix := "luxun_after_hit"
			wantTraitID := "huoshao_lianying"
			wantDetailKey := "targetExtraLosses"
			wantExtra := 123
			wantAttackerLost := 200
			wantEffectRate := 1.0
			if triggerChance == 0 {
				suffix = "luxun_after_miss"
				wantTraitID = "lianying_zengshang"
				wantExtra = 20
				wantAttackerLost = 97
				wantEffectRate = 0.1
			}
			report, stored := resolveYellowTurbanHeroFactionTest(t, hero, "wu", "wuInfantry", 100, "wei", "weiInfantry", 200, suffix)
			if report.Result != "attacker_victory" || report.EnemyPower != 2000 || report.PlayerPower != 1025 {
				t.Fatalf("expected exact yellow turban powers 2000/1025 and attacker victory, report=%+v", report)
			}
			if report.LostUnits["wuInfantry"] != 100 || report.SurvivedUnits["wuInfantry"] != 0 || report.DefenderLostUnits["weiInfantry"] != wantAttackerLost {
				t.Fatalf("expected defender full loss and attacker loss %d, report=%+v", wantAttackerLost, report)
			}
			if armySliceToMap(stored.Army)["wuInfantry"] != 0 || report.GeneralExpGained != wantAttackerLost || pvpTestGeneralExp(stored, "luxun") != wantAttackerLost {
				t.Fatalf("expected empty defending army and Lu Xun exp %d, state=%+v report=%+v", wantAttackerLost, stored, report)
			}
			if report.Detail == nil || report.Detail.SecondarySide == nil || len(report.Detail.PrimarySide.Units) == 0 || len(report.Detail.SecondarySide.Units) == 0 {
				t.Fatalf("expected complete standard yellow turban detail, report=%+v", report)
			}
			assertStandardUnitRow(t, report.ID, report.Detail.PrimarySide, "weiInfantry", 200, wantAttackerLost, 200-wantAttackerLost)
			assertStandardUnitRow(t, report.ID, *report.Detail.SecondarySide, "wuInfantry", 100, 100, 0)
			if !reportSideGeneralOwnsTrait(*report.Detail.SecondarySide, "luxun", "huoshao_lianying") || !reportSideGeneralOwnsTrait(*report.Detail.SecondarySide, "luxun", "lianying_zengshang") {
				t.Fatalf("expected Lu Xun snapshot to preserve both owned traits, side=%+v", report.Detail.SecondarySide)
			}
			if len(report.TraitTriggered) != 1 || report.TraitTriggered[0] != wantTraitID || len(report.TraitOutcomes) != 1 || len(report.Detail.Traits) != 1 {
				t.Fatalf("expected only %s in both timelines, report=%+v", wantTraitID, report)
			}
			outcome := report.TraitOutcomes[wantTraitID]
			extra, detailOK := outcome.Detail[wantDetailKey].(map[string]int)
			if !detailOK || extra["weiInfantry"] != wantExtra || outcome.OwnerSide != "defender" || outcome.OwnerGeneralID != "luxun" || outcome.Detail["effectRate"] != wantEffectRate || outcome.Detail["maxAffectedRate"] != wantEffectRate || outcome.Detail["triggerChance"] != 1.0 {
				t.Fatalf("expected defender-owned %s actual extra loss %d, outcome=%+v", wantTraitID, wantExtra, outcome)
			}
			trait := report.Detail.Traits[0]
			if trait.TraitID != wantTraitID || trait.OwnerSide != "secondary" || trait.OwnerRole != "defender" || trait.GeneralID != "luxun" {
				t.Fatalf("expected standard result to belong to defending Lu Xun, trait=%+v", trait)
			}
			otherTraitID := "lianying_zengshang"
			if triggerChance == 0 {
				otherTraitID = "huoshao_lianying"
			}
			if _, triggered := report.TraitOutcomes[otherTraitID]; triggered || standardReportHasTrait(report.Detail, otherTraitID) {
				t.Fatalf("expected non-effective %s to stay out of timeline, report=%+v", otherTraitID, report)
			}
		})
	}
}

// TestYellowTurbanHuangGaiSuppressionHitAndMissKeepsCounter 验证黄盖苦肉计无目标时如实记录零压制且不影响自身反击。
func TestYellowTurbanHuangGaiSuppressionHitAndMissKeepsCounter(t *testing.T) {
	for _, triggerChance := range []float64{1, 0} {
		label := "苦肉命中"
		if triggerChance == 0 {
			label = "苦肉合法未命中"
		}
		t.Run(label, func(t *testing.T) {
			hero := GeneralHeroConfig{
				ID: "huanggai", Name: "黄盖", Faction: "wu", Enabled: true,
				SpecialTrait: GeneralTraitConfig{
					TraitID: "kurouji", TraitType: general.TraitTypeSpecial, Enabled: true, Scope: "enemy_traits",
					Params: map[string]float64{"disableTraitCount": 1, "triggerChance": triggerChance},
				},
				BonusTrait: GeneralTraitConfig{
					TraitID: "kurou_fanji", TraitType: general.TraitTypeBonus, Enabled: true, Scope: "enemy_army",
					Params: map[string]float64{"effectRate": 0.1, "triggerChance": 1},
				},
			}
			suffix := "huanggai_suppression_hit"
			if triggerChance == 0 {
				suffix = "huanggai_suppression_miss"
			}
			report, stored := resolveYellowTurbanHeroFactionTest(t, hero, "wu", "wuInfantry", 100, "wei", "weiInfantry", 200, suffix)
			if report.Result != "attacker_victory" || report.EnemyPower != 2000 || report.PlayerPower != 1025 {
				t.Fatalf("expected exact yellow turban powers 2000/1025 and attacker victory, report=%+v", report)
			}
			if report.LostUnits["wuInfantry"] != 100 || report.SurvivedUnits["wuInfantry"] != 0 || report.DefenderLostUnits["weiInfantry"] != 97 {
				t.Fatalf("expected defender full loss and attacker core 77 plus counter 20, report=%+v", report)
			}
			if armySliceToMap(stored.Army)["wuInfantry"] != 0 || report.GeneralExpGained != 97 || pvpTestGeneralExp(stored, "huanggai") != 97 {
				t.Fatalf("expected empty defending army and Huang Gai exp 97, state=%+v report=%+v", stored, report)
			}
			if report.Detail == nil || report.Detail.SecondarySide == nil || len(report.Detail.PrimarySide.Units) == 0 || len(report.Detail.SecondarySide.Units) == 0 {
				t.Fatalf("expected complete standard yellow turban detail, report=%+v", report)
			}
			assertStandardUnitRow(t, report.ID, report.Detail.PrimarySide, "weiInfantry", 200, 97, 103)
			assertStandardUnitRow(t, report.ID, *report.Detail.SecondarySide, "wuInfantry", 100, 100, 0)
			if !reportSideGeneralOwnsTrait(*report.Detail.SecondarySide, "huanggai", "kurouji") || !reportSideGeneralOwnsTrait(*report.Detail.SecondarySide, "huanggai", "kurou_fanji") {
				t.Fatalf("expected Huang Gai snapshot to preserve both owned traits, side=%+v", report.Detail.SecondarySide)
			}
			counter, counterOK := report.TraitOutcomes["kurou_fanji"]
			counterExtra, counterDetailOK := counter.Detail["extraLosses"].(map[string]int)
			if !counterOK || !counterDetailOK || counterExtra["weiInfantry"] != 20 || counter.OwnerSide != "defender" || counter.OwnerGeneralID != "huanggai" || counter.Detail["effectRate"] != 0.1 || counter.Detail["triggerChance"] != 1.0 {
				t.Fatalf("expected defending Huang Gai counter to add 20 losses, outcome=%+v", counter)
			}

			if triggerChance == 0 {
				if len(report.TraitTriggered) != 1 || report.TraitTriggered[0] != "kurou_fanji" || len(report.TraitOutcomes) != 1 || len(report.Detail.Traits) != 1 || standardReportHasTrait(report.Detail, "kurouji") {
					t.Fatalf("expected missed Kurouji absent while counter remains, report=%+v", report)
				}
				return
			}
			if len(report.TraitTriggered) != 2 || report.TraitTriggered[0] != "kurouji" || report.TraitTriggered[1] != "kurou_fanji" || len(report.TraitOutcomes) != 2 || len(report.Detail.Traits) != 2 {
				t.Fatalf("expected Kurouji then counter in both timelines, report=%+v", report)
			}
			suppression, suppressionOK := report.TraitOutcomes["kurouji"]
			if !suppressionOK || suppression.OwnerSide != "defender" || suppression.OwnerGeneralID != "huanggai" || suppression.Detail["disableTraitCount"] != 1 || suppression.Detail["disabledTraitCount"] != 0 || suppression.Detail["triggerChance"] != 1.0 {
				t.Fatalf("expected design suppression 1 and actual 0, outcome=%+v", suppression)
			}
			if report.Detail.Traits[0].TraitID != "kurouji" || report.Detail.Traits[0].OwnerSide != "secondary" || report.Detail.Traits[0].OwnerRole != "defender" || report.Detail.Traits[0].Detail["disabledTraitCount"] != 0 ||
				report.Detail.Traits[1].TraitID != "kurou_fanji" || report.Detail.Traits[1].OwnerSide != "secondary" || report.Detail.Traits[1].OwnerRole != "defender" {
				t.Fatalf("expected ordered defender-owned standard outcomes, traits=%+v", report.Detail.Traits)
			}
		})
	}
}

// TestYellowTurbanSunQuanLegalMissKeepsOwnedSnapshot 验证孙权江东固守合法未命中时只保留双特性拥有快照。
func TestYellowTurbanSunQuanLegalMissKeepsOwnedSnapshot(t *testing.T) {
	hero := GeneralHeroConfig{
		ID: "sunquan", Name: "孙权", Faction: "wu", Enabled: true,
		SpecialTrait: GeneralTraitConfig{
			TraitID: "jiangdong_haoling", TraitType: general.TraitTypeSpecial, Enabled: true, Scope: "enemy_plunder",
			AllowedSides: []string{"defender"}, AllowedScenes: []string{"plunder"}, Params: map[string]float64{"plunderBonusRate": -0.2},
		},
		BonusTrait: GeneralTraitConfig{
			TraitID: "jiangdong_gushou", TraitType: general.TraitTypeBonus, Enabled: true, Scope: "self_army",
			AllowedSides: []string{"defender", "reinforcement"}, Params: map[string]float64{"triggerChance": 0, "defenseBonusRate": 0.5},
		},
	}
	report, stored := resolveYellowTurbanHeroFactionTest(t, hero, "wu", "wuInfantry", 100, "wei", "weiInfantry", 100, "sunquan_legal_miss")
	if report.Result != "defender_victory" || report.EnemyPower != 1000 || report.PlayerPower != 1025 {
		t.Fatalf("expected exact yellow turban powers 1000/1025 and defender victory, report=%+v", report)
	}
	if report.LostUnits["wuInfantry"] != 96 || report.SurvivedUnits["wuInfantry"] != 4 || report.DefenderLostUnits["weiInfantry"] != 100 {
		t.Fatalf("expected exact defender/attacker losses 96/100, report=%+v", report)
	}
	if armySliceToMap(stored.Army)["wuInfantry"] != 4 || report.GeneralExpGained != 100 || pvpTestGeneralExp(stored, "sunquan") != 100 {
		t.Fatalf("expected authoritative defender survivor 4 and Sun Quan exp 100, state=%+v report=%+v", stored, report)
	}
	if report.Detail == nil || report.Detail.SecondarySide == nil || len(report.Detail.PrimarySide.Units) == 0 || len(report.Detail.SecondarySide.Units) == 0 {
		t.Fatalf("expected complete standard yellow turban detail, report=%+v", report)
	}
	assertStandardUnitRow(t, report.ID, report.Detail.PrimarySide, "weiInfantry", 100, 100, 0)
	assertStandardUnitRow(t, report.ID, *report.Detail.SecondarySide, "wuInfantry", 100, 96, 4)
	if !reportSideGeneralOwnsTrait(*report.Detail.SecondarySide, "sunquan", "jiangdong_haoling") || !reportSideGeneralOwnsTrait(*report.Detail.SecondarySide, "sunquan", "jiangdong_gushou") {
		t.Fatalf("expected Sun Quan snapshot to preserve both owned traits, side=%+v", report.Detail.SecondarySide)
	}
	if len(report.TraitTriggered) != 0 || len(report.TraitOutcomes) != 0 || len(report.Detail.Traits) != 0 {
		t.Fatalf("expected legal Gushou miss and non-plunder Haoling to keep an empty timeline, report=%+v", report)
	}
}

// TestYellowTurbanMaChaoXiliangHitAndMissKeepsPassiveSnapshot 验证马超西凉突击只追加来袭骑兵损失且天神被动不改变守城防御。
func TestYellowTurbanMaChaoXiliangHitAndMissKeepsPassiveSnapshot(t *testing.T) {
	for _, triggerChance := range []float64{1, 0} {
		label := "西凉命中"
		if triggerChance == 0 {
			label = "西凉合法未命中"
		}
		t.Run(label, func(t *testing.T) {
			hero := GeneralHeroConfig{
				ID: "machao", Name: "马超", Faction: "shu", Enabled: true,
				SpecialTrait: GeneralTraitConfig{
					TraitID: "xiliang_tuji", TraitType: general.TraitTypeSpecial, Enabled: true, Scope: "enemy_army", TargetUnitType: "cavalry",
					Params: map[string]float64{"triggerChance": triggerChance, "effectRate": 0.12},
				},
				BonusTrait: GeneralTraitConfig{
					TraitID: "tianshen_xiafan", TraitType: general.TraitTypeBonus, Enabled: true, Scope: "self_army",
					Params: map[string]float64{"forceBonus": 20},
				},
			}
			suffix := "machao_xiliang_hit"
			wantAttackerLost := 58
			wantGeneralExp := 116
			if triggerChance == 0 {
				suffix = "machao_xiliang_miss"
				wantAttackerLost = 34
				wantGeneralExp = 68
			}
			report, stored := resolveYellowTurbanHeroFactionTest(t, hero, "shu", "shuInfantry", 100, "wei", "weiCavalry", 200, suffix)
			if report.Result != "attacker_victory" || report.EnemyPower != 2800 || report.PlayerPower != 816 {
				t.Fatalf("expected cavalry attack powers 2800/816 and attacker victory, report=%+v", report)
			}
			if report.LostUnits["shuInfantry"] != 100 || report.SurvivedUnits["shuInfantry"] != 0 || report.DefenderLostUnits["weiCavalry"] != wantAttackerLost {
				t.Fatalf("expected defender full loss and cavalry attacker loss %d, report=%+v", wantAttackerLost, report)
			}
			if armySliceToMap(stored.Army)["shuInfantry"] != 0 || report.GeneralExpGained != wantGeneralExp || pvpTestGeneralExp(stored, "machao") != wantGeneralExp {
				t.Fatalf("expected empty defending army and weighted Ma Chao exp %d, state=%+v report=%+v", wantGeneralExp, stored, report)
			}
			if report.Detail == nil || report.Detail.SecondarySide == nil || len(report.Detail.PrimarySide.Units) == 0 || len(report.Detail.SecondarySide.Units) == 0 || len(report.Detail.SecondarySide.Generals) != 1 {
				t.Fatalf("expected complete standard yellow turban detail, report=%+v", report)
			}
			primaryUnit := attackDefenseCrossReportUnit(t, report.Detail.PrimarySide, "weiCavalry")
			secondaryUnit := attackDefenseCrossReportUnit(t, *report.Detail.SecondarySide, "shuInfantry")
			if primaryUnit.UnitType != "weiCavalry" || primaryUnit.Dispatched != 200 || primaryUnit.Lost != wantAttackerLost || primaryUnit.Survived != 200-wantAttackerLost ||
				secondaryUnit.UnitType != "shuInfantry" || secondaryUnit.Dispatched != 100 || secondaryUnit.Lost != 100 || secondaryUnit.Survived != 0 {
				t.Fatalf("expected exact standard unit rows, primary=%+v secondary=%+v", primaryUnit, secondaryUnit)
			}
			snapshot := report.Detail.SecondarySide.Generals[0]
			if snapshot.ID != "machao" || snapshot.EffectiveStats["force"]-snapshot.Stats["force"] != 20 || snapshot.Buffs[StatAttackBonus] != 0.4 || !reportSideGeneralOwnsTrait(*report.Detail.SecondarySide, "machao", "xiliang_tuji") || !reportSideGeneralOwnsTrait(*report.Detail.SecondarySide, "machao", "tianshen_xiafan") {
				t.Fatalf("expected Ma Chao passive snapshot with force +20 and attack bonus 0.4, snapshot=%+v", snapshot)
			}
			if standardReportHasTrait(report.Detail, "tianshen_xiafan") {
				t.Fatalf("passive Tianshen must not enter yellow turban trigger timeline, report=%+v", report)
			}
			if triggerChance == 0 {
				if len(report.TraitTriggered) != 0 || len(report.TraitOutcomes) != 0 || len(report.Detail.Traits) != 0 {
					t.Fatalf("expected legal Xiliang miss to keep an empty timeline, report=%+v", report)
				}
				return
			}
			outcome, triggered := report.TraitOutcomes["xiliang_tuji"]
			extra, detailOK := outcome.Detail["targetExtraLosses"].(map[string]int)
			if !triggered || !detailOK || len(extra) != 1 || extra["weiCavalry"] != 24 || outcome.OwnerSide != "defender" || outcome.OwnerGeneralID != "machao" || outcome.Detail["effectRate"] != 0.12 || outcome.Detail["triggerChance"] != 1.0 {
				t.Fatalf("expected defender Xiliang to add 24 cavalry losses, outcome=%+v", outcome)
			}
			if len(report.TraitTriggered) != 1 || report.TraitTriggered[0] != "xiliang_tuji" || len(report.TraitOutcomes) != 1 || len(report.Detail.Traits) != 1 || report.Detail.Traits[0].TraitID != "xiliang_tuji" || report.Detail.Traits[0].OwnerSide != "secondary" || report.Detail.Traits[0].OwnerRole != "defender" {
				t.Fatalf("expected one defender-owned Xiliang result, report=%+v", report)
			}
		})
	}
}

// TestYellowTurbanZhaoYunLongdanLegalMissKeepsMarchTraitSnapshot 验证赵云龙胆合法未命中时不减损且七进七出只保留为拥有快照。
func TestYellowTurbanZhaoYunLongdanLegalMissKeepsMarchTraitSnapshot(t *testing.T) {
	hero := GeneralHeroConfig{
		ID: "zhaoyun", Name: "赵云", Faction: "shu", Enabled: true,
		SpecialTrait: GeneralTraitConfig{
			TraitID: "longdan_jiuyuan", TraitType: general.TraitTypeSpecial, Enabled: true, Scope: "reinforcement_self",
			AllowedSides: []string{"defender", "reinforcement"}, Params: map[string]float64{"triggerChance": 0, "lossReductionRate": 0.2},
		},
		BonusTrait: GeneralTraitConfig{
			TraitID: "qijin_qichu", TraitType: general.TraitTypeBonus, Enabled: true, Scope: "self_army",
			Params: map[string]float64{"speedBonusRate": 1, "minMarchSeconds": 60},
		},
	}
	report, stored := resolveYellowTurbanHeroFactionTest(t, hero, "shu", "greedyWolf", 100, "wei", "weiInfantry", 200, "zhaoyun_longdan_miss")
	if report.Result != "attacker_victory" || report.EnemyPower != 2000 || report.PlayerPower != 1020 || report.DefenderLostUnits["weiInfantry"] != 76 {
		t.Fatalf("expected unchanged 2000/1020 attack victory and enemy loss 76, report=%+v", report)
	}
	if report.LostUnits["greedyWolf"] != 100 || report.SurvivedUnits["greedyWolf"] != 0 || armySliceToMap(stored.Army)["greedyWolf"] != 0 {
		t.Fatalf("expected missed Longdan to keep full defender loss, state=%+v report=%+v", stored.Army, report)
	}
	if report.GeneralExpGained != 76 || pvpTestGeneralExp(stored, "zhaoyun") != 76 {
		t.Fatalf("expected Zhao Yun exp 76 from real enemy losses, state=%+v report=%+v", stored.Generals, report)
	}
	if report.Detail == nil || report.Detail.SecondarySide == nil || len(report.Detail.PrimarySide.Units) == 0 || len(report.Detail.SecondarySide.Units) == 0 {
		t.Fatalf("expected complete standard yellow turban detail, report=%+v", report)
	}
	assertStandardUnitRow(t, report.ID, report.Detail.PrimarySide, "weiInfantry", 200, 76, 124)
	assertStandardUnitRow(t, report.ID, *report.Detail.SecondarySide, "greedyWolf", 100, 100, 0)
	if !reportSideGeneralOwnsTrait(*report.Detail.SecondarySide, "zhaoyun", "longdan_jiuyuan") || !reportSideGeneralOwnsTrait(*report.Detail.SecondarySide, "zhaoyun", "qijin_qichu") {
		t.Fatalf("expected Zhao Yun snapshot to preserve both owned traits, side=%+v", report.Detail.SecondarySide)
	}
	if len(report.TraitTriggered) != 0 || len(report.TraitOutcomes) != 0 || len(report.Detail.Traits) != 0 {
		t.Fatalf("expected legal Longdan miss and march-only Qijin to keep an empty timeline, report=%+v", report)
	}
}

// resolveYellowTurbanTraitTest 构造一场确定兵力的黄巾战斗并返回战报与最终玩家状态。
func resolveYellowTurbanTraitTest(t *testing.T, traitCfg GeneralTraitConfig, defenderAmount int, attackerAmount int, suffix string) (BattleReport, GameState) {
	return resolveYellowTurbanTraitForGeneralTest(t, traitCfg, defenderAmount, attackerAmount, suffix, "test_general", "测试将领")
}

// resolveYellowTurbanTraitForGeneralTest 构造指定正式将领参与的确定兵力黄巾战斗。
func resolveYellowTurbanTraitForGeneralTest(t *testing.T, traitCfg GeneralTraitConfig, defenderAmount int, attackerAmount int, suffix string, generalID string, generalName string) (BattleReport, GameState) {
	t.Helper()
	hero := GeneralHeroConfig{ID: generalID, Name: generalName, Faction: "wei", Enabled: true}
	if traitCfg.TraitType == general.TraitTypeSpecial {
		hero.SpecialTrait = traitCfg
	} else {
		hero.BonusTrait = traitCfg
	}
	return resolveYellowTurbanHeroTest(t, hero, defenderAmount, attackerAmount, suffix)
}

// resolveYellowTurbanHeroTest 构造携带完整正式特性组合的黄巾守城真实事务。
func resolveYellowTurbanHeroTest(t *testing.T, hero GeneralHeroConfig, defenderAmount int, attackerAmount int, suffix string) (BattleReport, GameState) {
	return resolveYellowTurbanHeroFactionTest(t, hero, "wei", "weiInfantry", defenderAmount, "wei", "weiInfantry", attackerAmount, suffix)
}

// resolveYellowTurbanHeroFactionTest 构造指定阵营、兵种和完整将领配置的黄巾守城真实事务。
func resolveYellowTurbanHeroFactionTest(t *testing.T, hero GeneralHeroConfig, defenderFaction string, defenderUnit string, defenderAmount int, attackerFaction string, attackerUnit string, attackerAmount int, suffix string) (BattleReport, GameState) {
	t.Helper()
	setTestCombatUnitsConfig(t)
	unitsMu.Lock()
	for _, entry := range []struct{ faction, unitType string }{{defenderFaction, defenderUnit}, {attackerFaction, attackerUnit}} {
		faction, unitType := entry.faction, entry.unitType
		if activeUnits[faction] == nil {
			activeUnits[faction] = FactionUnits{}
		}
		if _, exists := activeUnits[faction][unitType]; !exists {
			activeUnits[faction][unitType] = UnitConfig{
				Name: unitType, Category: "infantry",
				Stats: map[string]int{"attack": 10, "infantryDefense": 10, "cavalryDefense": 8, "carryCapacity": 5, "upkeep": 1},
			}
		}
	}
	unitsMu.Unlock()
	generalID := hero.ID
	generalName := hero.Name
	setTestFactionsAndGenerals(t, FactionsConfig{
		defenderFaction: {Name: defenderFaction, Generals: []GeneralInfo{{ID: generalID, Name: generalName}}},
	}, GeneralsConfig{Enabled: true, Heroes: map[string]GeneralHeroConfig{generalID: hero}})
	SetYellowTurbanConfig(defaultYellowTurbanConfig())
	repo := NewMemoryRepository()
	svc := NewServiceWithRepository(repo)
	now := time.Now().UTC()
	account := Account{ID: "account_yt_trait_" + suffix, Username: "yt_trait_" + suffix, CreatedAt: now}
	if err := repo.CreateAccount(account); err != nil {
		t.Fatalf("create account: %v", err)
	}
	state := newPlayerState("player_yt_trait_"+suffix, "黄巾特性测试", defenderFaction, generalID, now)
	EnsureGeneralRoster(&state, now)
	state.Army = []ArmyUnit{{UnitType: defenderUnit, Amount: defenderAmount}}
	if err := repo.CreatePlayer(account.ID, state, now); err != nil {
		t.Fatalf("create player: %v", err)
	}
	march, err := repo.CreateYellowTurbanMarch(YellowTurbanMarch{
		ID: "yt_trait_march_" + suffix, TargetPlayerID: state.Player.ID, SourceCityID: "yt_trait_city_" + suffix,
		SourceName: "黄巾军", SourceFaction: attackerFaction, SourceRegionID: attackerFaction, RiskLevelID: 1, RiskLevelName: "黄巾测试",
		PlayerFood: 1000, FoodCapacity: 100, Pressure: 10, Troops: map[string]int{attackerUnit: attackerAmount},
		Status: YellowTurbanMarchStatusMarching, DurationSeconds: 1,
		StartedAt: now.Add(-2 * time.Minute).Format(resourceDateLayout), ArrivesAt: now.Add(-time.Minute).Format(resourceDateLayout),
		CreatedAt: now.Add(-2 * time.Minute).Format(resourceDateLayout), UpdatedAt: now.Add(-2 * time.Minute).Format(resourceDateLayout),
	})
	if err != nil {
		t.Fatalf("create yellow turban march: %v", err)
	}
	report, err := svc.ResolveYellowTurbanMarch(march.ID)
	if err != nil {
		t.Fatalf("resolve yellow turban march: %v", err)
	}
	stored, err := repo.GetState(state.Player.ID)
	if err != nil {
		t.Fatalf("get state: %v", err)
	}
	return report, stored
}

// yellowTurbanStandardDefenderSurvived 返回标准黄巾战报中守城方指定兵种的存活数。
func yellowTurbanStandardDefenderSurvived(report BattleReport, unitType string) int {
	if report.Detail == nil || report.Detail.SecondarySide == nil {
		return -1
	}
	for _, unit := range report.Detail.SecondarySide.Units {
		if unit.UnitType == unitType {
			return unit.Survived
		}
	}
	return -1
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
	svc := NewServiceWithRepository(&rejectStandaloneReportBundleRepository{MemoryRepository: repo})
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
	baselineExp := generalExpRequiredForLevelForTest(2) - 1
	setPvpTestGeneralProgress(&helper, "sunquan", 1, baselineExp)
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
		Troops:            map[string]int{"shadowGuard": 400},
		RemainingTroops:   map[string]int{"shadowGuard": 400},
		Generals: []ReinforcementGeneralSnapshot{{
			ID:    "sunquan",
			Name:  "孙权",
			Level: 1,
			Exp:   baselineExp,
		}},
		Losses:    map[string]int{},
		Rules:     defaultGarrisonRules(GarrisonSourceReinforcement),
		SentAt:    now.Add(-4 * time.Hour).UTC().Format(resourceDateLayout),
		ArrivedAt: now.Add(-3 * time.Hour).UTC().Format(resourceDateLayout),
		CreatedAt: now.Add(-4 * time.Hour).UTC().Format(resourceDateLayout),
		UpdatedAt: now.Add(-3 * time.Hour).UTC().Format(resourceDateLayout),
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
		Troops:          map[string]int{"qingZhouArmy": 300},
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
	if report.PvpReinforcements[0].Troops["shadowGuard"] != 400 {
		t.Fatalf("expected yellow turban report to preserve dispatched reinforcement troops, got %+v", report.PvpReinforcements[0])
	}
	if len(report.PvpReinforcementLosses[reinforcement.ID]) == 0 {
		t.Fatalf("expected reinforcement losses in yellow turban report, got %+v", report.PvpReinforcementLosses)
	}
	if report.Detail == nil || report.Detail.Extra["yellowTurban"] == nil {
		t.Fatalf("expected yellow turban context in defense report detail, got %+v", report.Detail)
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
	if helperReports[0].Detail == nil || helperReports[0].Detail.Extra["yellowTurban"] == nil || helperReports[0].Detail.Extra["reinforcement"] == nil {
		t.Fatalf("expected yellow turban reinforcement detail contexts, got %+v", helperReports[0].Detail)
	}
	if helperReports[0].GeneralExpGained <= 0 || helperReports[0].Detail == nil || helperReports[0].Detail.Rewards.GeneralExp != helperReports[0].GeneralExpGained {
		t.Fatalf("expected reinforcement general exp in report detail, got %+v", helperReports[0])
	}
	firstLevelAfter := helperReports[0].GeneralLevelAfter
	if helperReports[0].GeneralLevelBefore != 1 || firstLevelAfter <= 1 || helperReports[0].Detail.Rewards.GeneralLevelBefore != 1 || helperReports[0].Detail.Rewards.GeneralLevelAfter != firstLevelAfter {
		t.Fatalf("expected first yellow turban reinforcement battle to level up from 1, got %+v", helperReports[0])
	}
	if report.PvpReinforcements[0].GeneralExpGained != helperReports[0].GeneralExpGained {
		t.Fatalf("expected defense report reinforcement exp %d, got %+v", helperReports[0].GeneralExpGained, report.PvpReinforcements[0])
	}
	if helperReports[0].Detail.SecondarySide == nil || helperReports[0].Detail.PrimarySide.Role != "attacker" || helperReports[0].Detail.SecondarySide.Role != "defender" {
		t.Fatalf("expected reinforcement report to preserve complete yellow turban attack and defense snapshots, got %+v", helperReports[0].Detail)
	}
	if len(helperReports[0].PvpReinforcements) != 1 || len(helperReports[0].PvpReinforcements[0].Generals) != 1 || helperReports[0].PvpReinforcements[0].Generals[0].ID != "sunquan" {
		t.Fatalf("expected reinforcement report to show helper general in reinforcement snapshot, got %+v", helperReports[0].PvpReinforcements)
	}
	if report.PvpReinforcements[0].GeneralLevelBefore != 1 || report.PvpReinforcements[0].GeneralLevelAfter != firstLevelAfter {
		t.Fatalf("expected defense report reinforcement level 1 -> %d, got %+v", firstLevelAfter, report.PvpReinforcements[0])
	}
	if helperReports[0].Detail.Visibility.Threshold != 0 {
		t.Fatalf("reinforcement report should not show enemy reveal threshold, got %+v", helperReports[0].Detail.Visibility)
	}
	updated := repo.reinforcements[reinforcement.ID]
	if updated.LastBattleReportID != helperReports[0].ID {
		t.Fatalf("expected reinforcement last report %s, got %+v", helperReports[0].ID, updated)
	}
	updatedHelper, err := repo.GetState(helper.Player.ID)
	if err != nil {
		t.Fatalf("GetState helper failed: %v", err)
	}
	if got := pvpTestGeneralExp(updatedHelper, "sunquan"); got != baselineExp+helperReports[0].GeneralExpGained || pvpTestGeneralLevel(updatedHelper, "sunquan") != firstLevelAfter {
		t.Fatalf("expected sunquan cumulative exp %d at level %d, got exp=%d level=%d", baselineExp+helperReports[0].GeneralExpGained, firstLevelAfter, got, pvpTestGeneralLevel(updatedHelper, "sunquan"))
	}
	if len(updated.Generals) != 1 || updated.Generals[0].Exp != baselineExp+helperReports[0].GeneralExpGained || updated.Generals[0].Level != firstLevelAfter {
		t.Fatalf("expected reinforcement record to persist first settlement progress, got %+v", updated.Generals)
	}
	retriedReport, err := svc.ResolveYellowTurbanMarch(march.ID)
	if err != nil || retriedReport.ID != report.ID {
		t.Fatalf("expected resolved yellow turban march to return persisted report, report=%+v err=%v", retriedReport, err)
	}
	retriedHelperReports, retriedTotal, err := repo.ListReportsByQuery(BattleReportQuery{PlayerID: helper.Player.ID, ViewType: ReportViewReinforcement, Page: 1, PageSize: 10})
	if err != nil || retriedTotal != 1 || len(retriedHelperReports) != 1 {
		t.Fatalf("expected retry not to duplicate helper report, total=%d reports=%+v err=%v", retriedTotal, retriedHelperReports, err)
	}
	retriedHelper, err := repo.GetState(helper.Player.ID)
	if err != nil || pvpTestGeneralExp(retriedHelper, "sunquan") != baselineExp+helperReports[0].GeneralExpGained {
		t.Fatalf("expected retry not to duplicate helper exp, state=%+v err=%v", retriedHelper.Generals, err)
	}

	secondMarch, err := repo.CreateYellowTurbanMarch(YellowTurbanMarch{
		ID:              "yt_rein_march_second",
		TargetPlayerID:  defender.Player.ID,
		SourceCityID:    "yt_wei_2",
		SourceName:      "黄巾军·魏地二队",
		SourceFaction:   "wei",
		SourceRegionID:  "wei",
		RiskLevelID:     1,
		RiskLevelName:   "黄巾·流寇",
		PlayerFood:      10000,
		FoodCapacity:    1000,
		Pressure:        10,
		Troops:          map[string]int{"qingZhouArmy": 50},
		Status:          YellowTurbanMarchStatusMarching,
		DurationSeconds: 1,
		StartedAt:       now.Add(-2 * time.Minute).Format(resourceDateLayout),
		ArrivesAt:       now.Add(-time.Minute).Format(resourceDateLayout),
		CreatedAt:       now.Add(-2 * time.Minute).Format(resourceDateLayout),
		UpdatedAt:       now.Add(-2 * time.Minute).Format(resourceDateLayout),
	})
	if err != nil {
		t.Fatalf("CreateYellowTurbanMarch second failed: %v", err)
	}
	secondDefenseReport, err := svc.ResolveYellowTurbanMarch(secondMarch.ID)
	if err != nil {
		t.Fatalf("ResolveYellowTurbanMarch second failed: %v", err)
	}
	secondHelperReports, secondTotal, err := repo.ListReportsByQuery(BattleReportQuery{PlayerID: helper.Player.ID, ViewType: ReportViewReinforcement, Page: 1, PageSize: 10})
	if err != nil || secondTotal != 2 || len(secondHelperReports) != 2 {
		t.Fatalf("expected two helper reports after consecutive battles, total=%d reports=%+v err=%v", secondTotal, secondHelperReports, err)
	}
	secondHelperReport := secondHelperReports[0]
	if secondHelperReport.GeneralExpGained <= 0 || secondHelperReport.GeneralLevelBefore != firstLevelAfter || secondHelperReport.GeneralLevelAfter < firstLevelAfter || secondHelperReport.Detail.Rewards.GeneralLevelBefore != firstLevelAfter || secondHelperReport.Detail.Rewards.GeneralLevelAfter != secondHelperReport.GeneralLevelAfter {
		t.Fatalf("expected second yellow turban reinforcement battle to continue at level %d, got %+v", firstLevelAfter, secondHelperReport)
	}
	if len(secondDefenseReport.PvpReinforcements) != 1 || secondDefenseReport.PvpReinforcements[0].GeneralLevelBefore != firstLevelAfter || secondDefenseReport.PvpReinforcements[0].GeneralLevelAfter != secondHelperReport.GeneralLevelAfter {
		t.Fatalf("expected second defense report to use carried level %d baseline, got %+v", firstLevelAfter, secondDefenseReport.PvpReinforcements)
	}
	secondUpdated := repo.reinforcements[reinforcement.ID]
	secondUpdatedHelper, err := repo.GetState(helper.Player.ID)
	if err != nil {
		t.Fatalf("GetState helper after second battle failed: %v", err)
	}
	wantFinalExp := baselineExp + helperReports[0].GeneralExpGained + secondHelperReport.GeneralExpGained
	if len(secondUpdated.Generals) != 1 || secondUpdated.Generals[0].Exp != wantFinalExp || secondUpdated.Generals[0].Level != secondHelperReport.GeneralLevelAfter || pvpTestGeneralExp(secondUpdatedHelper, "sunquan") != wantFinalExp || pvpTestGeneralLevel(secondUpdatedHelper, "sunquan") != secondHelperReport.GeneralLevelAfter {
		t.Fatalf("expected consecutive record and owner progress exp=%d level=%d, record=%+v state=%+v", wantFinalExp, secondHelperReport.GeneralLevelAfter, secondUpdated.Generals, secondUpdatedHelper.Generals)
	}
	repeatedReport, err := svc.ResolveYellowTurbanMarch(march.ID)
	if err != nil {
		t.Fatalf("repeat ResolveYellowTurbanMarch failed: %v", err)
	}
	if repeatedReport.ID != report.ID {
		t.Fatalf("expected repeat resolve to return existing report %s, got %s", report.ID, repeatedReport.ID)
	}
	repeatedHelper, err := repo.GetState(helper.Player.ID)
	if err != nil {
		t.Fatalf("GetState repeated helper failed: %v", err)
	}
	if got := pvpTestGeneralExp(repeatedHelper, "sunquan"); got != wantFinalExp {
		t.Fatalf("expected repeated yellow turban resolve to keep cumulative exp %d, got %d", wantFinalExp, got)
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
