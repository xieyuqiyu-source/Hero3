// 本文件验证曹操魏武统御进入真实援军防御、三方战报和最终驻防兵力。
package game

import (
	"testing"
	"time"

	"hero3/internal/core/general"
)

type weiwuReinforcementResult struct {
	defensePower float64
	losses       int
	record       Reinforcement
	reports      []BattleReport
	helperState  GameState
}

type weiwuYellowTurbanReinforcementResult struct {
	defensePower int
	record       Reinforcement
	reports      []BattleReport
}

// runWeiwuReinforcementPvp 执行曹操援军驻防后遭受攻击的完整 PVP 事务。
func runWeiwuReinforcementPvp(t *testing.T, enabled bool) weiwuReinforcementResult {
	t.Helper()
	return runWeiwuReinforcementPvpWithGenerals(t, enabled, []string{"caocao"})
}

// runWeiwuReinforcementPvpWithGenerals 按指定携将列表执行曹操援军完整 PVP 事务。
func runWeiwuReinforcementPvpWithGenerals(t *testing.T, enabled bool, generalIDs []string) weiwuReinforcementResult {
	return runWeiwuReinforcementPvpWithConfigChange(t, enabled, nil, generalIDs)
}

// runWeiwuReinforcementPvpWithConfigChange 在援军派出后切换全局配置，验证驻防使用派出快照。
func runWeiwuReinforcementPvpWithConfigChange(t *testing.T, dispatchEnabled bool, battleEnabled *bool, generalIDs []string) weiwuReinforcementResult {
	t.Helper()
	weiwuTrait := GeneralTraitConfig{
		TraitID: "weiwu_tongyu", TraitType: general.TraitTypeBonus, Enabled: dispatchEnabled, Scope: "self_army",
		AllowedSides: []string{"defender", "reinforcement"},
		Params:       map[string]float64{"defenseBonusRate": 0.15},
	}
	setTestFactionsAndGenerals(t, FactionsConfig{
		"shu": {Name: "蜀国", Generals: []GeneralInfo{{ID: "liubei", Name: "刘备"}}},
		"wu":  {Name: "吴国", Generals: []GeneralInfo{{ID: "sunquan", Name: "孙权"}}},
		"wei": {Name: "魏国", Generals: []GeneralInfo{{ID: "caocao", Name: "曹操"}}},
	}, GeneralsConfig{Enabled: true, Heroes: map[string]GeneralHeroConfig{
		"liubei":  {ID: "liubei", Name: "刘备", Faction: "shu", Enabled: true},
		"sunquan": {ID: "sunquan", Name: "孙权", Faction: "wu", Enabled: true},
		"caocao": {
			ID: "caocao", Name: "曹操", Faction: "wei", Enabled: true,
			SpecialTrait: GeneralTraitConfig{TraitID: "weiwu_haoling", TraitType: general.TraitTypeSpecial, Enabled: false, Scope: "self_city"},
			BonusTrait:   weiwuTrait,
		},
	}})
	svc, repo, attacker, defender := newPvpTestServiceForGenerals(t, "shu", "liubei", "wu", "sunquan")
	unitsMu.Lock()
	if activeUnits["wei"] == nil {
		activeUnits["wei"] = FactionUnits{}
	}
	activeUnits["wei"]["huWei"] = UnitConfig{
		Name: "虎卫", Category: "infantry",
		Stats: map[string]int{"attack": 10, "infantryDefense": 10, "cavalryDefense": 8, "carryCapacity": 5, "upkeep": 1},
	}
	unitsMu.Unlock()
	now := time.Now().UTC()
	helperAccount := Account{ID: "account_weiwu_helper", Username: "weiwu_helper", PasswordHash: "hash", CreatedAt: now}
	if err := repo.CreateAccount(helperAccount); err != nil {
		t.Fatalf("CreateAccount helper failed: %v", err)
	}
	helper := newPlayerState("player_weiwu_helper", "魏国援军", "wei", "caocao", now)
	helper.Army = []ArmyUnit{{UnitType: "huWei", Amount: 100}}
	if err := repo.CreatePlayer(helperAccount.ID, helper, now); err != nil {
		t.Fatalf("CreatePlayer helper failed: %v", err)
	}
	attacker.Army = []ArmyUnit{{UnitType: "shuInfantry", Amount: 110}}
	defender.Army = []ArmyUnit{{UnitType: "wuInfantry", Amount: 1}}
	defender.Buildings = nil
	repo.players[attacker.Player.ID] = attacker
	repo.players[defender.Player.ID] = defender

	sent, err := svc.SendReinforcement(SendReinforcementRequest{
		FromPlayerID: helper.Player.ID, TargetPlayerID: defender.Player.ID,
		Troops: map[string]int{"huWei": 100}, GeneralIDs: generalIDs,
	})
	if err != nil {
		t.Fatalf("SendReinforcement failed: %v", err)
	}
	if battleEnabled != nil {
		battleConfig := GetGeneralsConfig()
		battleCaoCao := battleConfig.Heroes["caocao"]
		battleCaoCao.BonusTrait.Enabled = *battleEnabled
		battleConfig.Heroes["caocao"] = battleCaoCao
		if err := SetGeneralsConfig(battleConfig); err != nil {
			t.Fatalf("change Cao Cao trait config after reinforcement dispatch failed: %v", err)
		}
	}
	forceReinforcementDue(t, repo, sent.Reinforcement.ID, true)
	if _, err := svc.MarkReinforcementArrived(sent.Reinforcement.ID); err != nil {
		t.Fatalf("MarkReinforcementArrived failed: %v", err)
	}
	started, err := svc.StartPvpAttack(PvpAttackRequest{
		PlayerID: attacker.Player.ID, TargetPlayerID: defender.Player.ID, MarchMode: PvpMarchTypeAttack,
		Troops: map[string]int{"shuInfantry": 110}, GeneralIDs: []string{"liubei"},
	})
	if err != nil {
		t.Fatalf("StartPvpAttack failed: %v", err)
	}
	forcePvpMarchDue(t, repo, started.March.ID)
	battle, err := svc.ResolvePvpMarch(started.March.ID)
	if err != nil {
		t.Fatalf("ResolvePvpMarch failed: %v", err)
	}
	defensePower, ok := battle.Result["defensePower"].(float64)
	if !ok {
		t.Fatalf("expected numeric defense power, got %+v", battle.Result)
	}
	attackerReports, _, err := repo.ListReports(attacker.Player.ID, 10, 0)
	if err != nil || len(attackerReports) == 0 || attackerReports[0].ID != battle.AttackerReportID {
		t.Fatalf("expected attacker report, reports=%+v err=%v", attackerReports, err)
	}
	defenderReports, _, err := repo.ListReports(defender.Player.ID, 10, 0)
	if err != nil || len(defenderReports) == 0 || defenderReports[0].ID != battle.DefenderReportID {
		t.Fatalf("expected defender report, reports=%+v err=%v", defenderReports, err)
	}
	helperReports, total, err := repo.ListReportsByQuery(BattleReportQuery{PlayerID: helper.Player.ID, ViewType: ReportViewReinforcement, Page: 1, PageSize: 10})
	if err != nil || total != 1 || len(helperReports) != 1 {
		t.Fatalf("expected one helper report, reports=%+v total=%d err=%v", helperReports, total, err)
	}
	losses := defenderReports[0].PvpReinforcementLosses[sent.Reinforcement.ID]["huWei"]
	if attackerReports[0].PvpReinforcementLosses[sent.Reinforcement.ID]["huWei"] != losses || helperReports[0].LostUnits["huWei"] != losses {
		t.Fatalf("expected three reports to agree on reinforcement losses %d, reports=%+v/%+v/%+v", losses, attackerReports[0].PvpReinforcementLosses, defenderReports[0].PvpReinforcementLosses, helperReports[0].LostUnits)
	}
	stored, err := repo.GetReinforcement(sent.Reinforcement.ID)
	if err != nil || stored.RemainingTroops["huWei"] != 100-losses {
		t.Fatalf("expected stored reinforcement %d, record=%+v err=%v", 100-losses, stored, err)
	}
	helperState, err := repo.GetState(helper.Player.ID)
	if err != nil {
		t.Fatalf("GetState helper failed: %v", err)
	}
	return weiwuReinforcementResult{
		defensePower: defensePower, losses: losses, record: stored,
		reports: []BattleReport{attackerReports[0], defenderReports[0], helperReports[0]}, helperState: helperState,
	}
}

// boolPointer 返回布尔指针，便于测试显式区分“不改配置”和“改为指定值”。
func boolPointer(value bool) *bool {
	return &value
}

// reinforcementSnapshotHasTrait 判断援军将领快照是否包含指定特性。
func reinforcementSnapshotHasTrait(snapshot ReinforcementGeneralSnapshot, traitID string) bool {
	for _, trait := range snapshot.Traits {
		if trait.TraitID == traitID {
			return true
		}
	}
	return false
}

// TestPvpWeiwuTongyuStrengthensOwnReinforcement 验证魏武统御只强化曹操援军并进入三方战报。
func TestPvpWeiwuTongyuStrengthensOwnReinforcement(t *testing.T) {
	control := runWeiwuReinforcementPvp(t, false)
	active := runWeiwuReinforcementPvp(t, true)
	if control.defensePower != 1010 || active.defensePower != 1210 {
		t.Fatalf("expected real defense power 1010 -> 1210, control=%v active=%v", control.defensePower, active.defensePower)
	}
	if active.losses >= control.losses {
		t.Fatalf("expected stronger reinforcement to lose fewer troops, control=%d active=%d", control.losses, active.losses)
	}
	for _, report := range active.reports {
		outcome, ok := report.TraitOutcomes["weiwu_tongyu"]
		infantry, infantryOK := outcome.Detail["infantryDefenseModifiedUnits"].(map[string]int)
		cavalry, cavalryOK := outcome.Detail["cavalryDefenseModifiedUnits"].(map[string]int)
		_, hasAttackDelta := outcome.Detail["attackModifiedUnits"]
		if !ok || !infantryOK || !cavalryOK || hasAttackDelta || infantry["huWei"] != 2 || cavalry["huWei"] != 1 || outcome.OwnerSide != "reinforcement" || outcome.OwnerGeneralID != "caocao" {
			t.Fatalf("expected reinforcement-owned HuWei defense deltas +2/+1 without attack bonus, got %+v", outcome)
		}
		standardFound := false
		for _, trait := range report.Detail.Traits {
			if trait.TraitID == "weiwu_tongyu" && trait.GeneralID == "caocao" && trait.OwnerRole == "reinforcement" {
				standardFound = true
			}
		}
		if !standardFound {
			t.Fatalf("expected Weiwu Tongyu in standard report, traits=%+v", report.Detail.Traits)
		}
	}
}

// TestPvpReinforcementTraitUsesDispatchConfigEverywhere 验证驻防特性的效果、快照和战报统一使用派出时配置。
func TestPvpReinforcementTraitUsesDispatchConfigEverywhere(t *testing.T) {
	t.Run("派出时开启而战前关闭", func(t *testing.T) {
		result := runWeiwuReinforcementPvpWithConfigChange(t, true, boolPointer(false), []string{"caocao"})
		assertWeiwuReinforcementDispatchSnapshot(t, result, true, 1210)
	})
	t.Run("派出时关闭而战前开启", func(t *testing.T) {
		result := runWeiwuReinforcementPvpWithConfigChange(t, false, boolPointer(true), []string{"caocao"})
		assertWeiwuReinforcementDispatchSnapshot(t, result, false, 1010)
	})
}

// assertWeiwuReinforcementDispatchSnapshot 核对援军持久快照、真实战力和三方战报时间线保持一致。
func assertWeiwuReinforcementDispatchSnapshot(t *testing.T, result weiwuReinforcementResult, expected bool, expectedDefensePower float64) {
	t.Helper()
	if result.defensePower != expectedDefensePower {
		t.Fatalf("expected dispatch-config defense power %v, got %v", expectedDefensePower, result.defensePower)
	}
	if len(result.record.Generals) != 1 || reinforcementSnapshotHasTrait(result.record.Generals[0], "weiwu_tongyu") != expected {
		t.Fatalf("expected stored deployment snapshot trait=%v, record=%+v", expected, result.record)
	}
	for _, report := range result.reports {
		_, hasOutcome := report.TraitOutcomes["weiwu_tongyu"]
		hasTimeline := report.Detail != nil && standardReportHasTrait(report.Detail, "weiwu_tongyu")
		if hasOutcome != expected || hasTimeline != expected {
			t.Fatalf("expected report outcome and timeline trait=%v, report=%s outcomes=%+v detail=%+v", expected, report.ID, report.TraitOutcomes, report.Detail)
		}
		if len(report.PvpReinforcements) != 1 || len(report.PvpReinforcements[0].Generals) != 1 || reinforcementSnapshotHasTrait(report.PvpReinforcements[0].Generals[0], "weiwu_tongyu") != expected {
			t.Fatalf("expected report deployment snapshot trait=%v, report=%s reinforcements=%+v", expected, report.ID, report.PvpReinforcements)
		}
	}
}

// TestYellowTurbanReinforcementTraitUsesDispatchConfigEverywhere 验证黄巾协防同样固定援军派出配置。
func TestYellowTurbanReinforcementTraitUsesDispatchConfigEverywhere(t *testing.T) {
	t.Run("派出时开启而战前关闭", func(t *testing.T) {
		result := runWeiwuYellowTurbanReinforcementWithConfigChange(t, true, false)
		assertWeiwuYellowTurbanDispatchSnapshot(t, result, true, 1210)
	})
	t.Run("派出时关闭而战前开启", func(t *testing.T) {
		result := runWeiwuYellowTurbanReinforcementWithConfigChange(t, false, true)
		assertWeiwuYellowTurbanDispatchSnapshot(t, result, false, 1010)
	})
}

// runWeiwuYellowTurbanReinforcementWithConfigChange 执行真实派援后切换配置的黄巾协防事务。
func runWeiwuYellowTurbanReinforcementWithConfigChange(t *testing.T, dispatchEnabled bool, battleEnabled bool) weiwuYellowTurbanReinforcementResult {
	t.Helper()
	setTestFactionsAndGenerals(t, FactionsConfig{
		"wei": {Name: "魏国", Generals: []GeneralInfo{{ID: "caocao", Name: "曹操"}}},
		"shu": {Name: "蜀国", Generals: []GeneralInfo{{ID: "liubei", Name: "刘备"}}},
	}, GeneralsConfig{Enabled: true, Heroes: map[string]GeneralHeroConfig{
		"caocao": {
			ID: "caocao", Name: "曹操", Faction: "wei", Enabled: true,
			BonusTrait: GeneralTraitConfig{
				TraitID: "weiwu_tongyu", TraitType: general.TraitTypeBonus, Enabled: dispatchEnabled,
				Scope: "self_army", AllowedSides: []string{"defender", "reinforcement"},
				Params: map[string]float64{"defenseBonusRate": 0.15},
			},
		},
		"liubei": {ID: "liubei", Name: "刘备", Faction: "shu", Enabled: true},
	}})
	svc, repo, helper, defender := newReinforcementTestService(t)
	unitsMu.Lock()
	activeUnits["wei"]["huWei"] = UnitConfig{
		Name: "虎卫", Category: "infantry",
		Stats: map[string]int{"attack": 10, "infantryDefense": 10, "cavalryDefense": 8, "carryCapacity": 5, "upkeep": 1},
	}
	activeUnits["shu"] = FactionUnits{"shuInfantry": {
		Name: "蜀步兵", Category: "infantry",
		Stats: map[string]int{"attack": 10, "infantryDefense": 10, "cavalryDefense": 8, "carryCapacity": 5, "upkeep": 1},
	}}
	unitsMu.Unlock()
	helper.Army = []ArmyUnit{{UnitType: "huWei", Amount: 100}}
	defender.Army = []ArmyUnit{{UnitType: "shuInfantry", Amount: 1}}
	defender.Buildings = nil
	repo.players[helper.Player.ID] = helper
	repo.players[defender.Player.ID] = defender

	sent, err := svc.SendReinforcement(SendReinforcementRequest{
		FromPlayerID: helper.Player.ID, TargetPlayerID: defender.Player.ID,
		Troops: map[string]int{"huWei": 100}, GeneralIDs: []string{"caocao"},
	})
	if err != nil {
		t.Fatalf("SendReinforcement failed: %v", err)
	}
	battleConfig := GetGeneralsConfig()
	battleCaoCao := battleConfig.Heroes["caocao"]
	battleCaoCao.BonusTrait.Enabled = battleEnabled
	battleConfig.Heroes["caocao"] = battleCaoCao
	if err := SetGeneralsConfig(battleConfig); err != nil {
		t.Fatalf("change Cao Cao config before yellow turban battle failed: %v", err)
	}
	forceReinforcementDue(t, repo, sent.Reinforcement.ID, true)
	if _, err := svc.MarkReinforcementArrived(sent.Reinforcement.ID); err != nil {
		t.Fatalf("MarkReinforcementArrived failed: %v", err)
	}

	SetYellowTurbanConfig(defaultYellowTurbanConfig())
	now := time.Now().UTC()
	march, err := repo.CreateYellowTurbanMarch(YellowTurbanMarch{
		ID:             "yt_weiwu_" + sent.Reinforcement.ID,
		TargetPlayerID: defender.Player.ID, SourceCityID: "yt_weiwu_source", SourceName: "黄巾军",
		SourceFaction: "wei", SourceRegionID: "wei", RiskLevelID: 1, RiskLevelName: "黄巾·流寇",
		PlayerFood: 1000, FoodCapacity: 100, Pressure: 10, Troops: map[string]int{"weiInfantry": 100},
		Status: YellowTurbanMarchStatusMarching, DurationSeconds: 1,
		StartedAt: now.Add(-2 * time.Minute).Format(resourceDateLayout), ArrivesAt: now.Add(-time.Minute).Format(resourceDateLayout),
		CreatedAt: now.Add(-2 * time.Minute).Format(resourceDateLayout), UpdatedAt: now.Add(-2 * time.Minute).Format(resourceDateLayout),
	})
	if err != nil {
		t.Fatalf("CreateYellowTurbanMarch failed: %v", err)
	}
	report, err := svc.ResolveYellowTurbanMarch(march.ID)
	if err != nil {
		t.Fatalf("ResolveYellowTurbanMarch failed: %v", err)
	}
	helperReports, total, err := repo.ListReportsByQuery(BattleReportQuery{
		PlayerID: helper.Player.ID, ViewType: ReportViewReinforcement, Page: 1, PageSize: 10,
	})
	if err != nil || total != 1 || len(helperReports) != 1 {
		t.Fatalf("expected one yellow turban helper report, reports=%+v total=%d err=%v", helperReports, total, err)
	}
	stored, err := repo.GetReinforcement(sent.Reinforcement.ID)
	if err != nil {
		t.Fatalf("GetReinforcement failed: %v", err)
	}
	return weiwuYellowTurbanReinforcementResult{
		defensePower: report.PlayerPower,
		record:       stored,
		reports:      []BattleReport{report, helperReports[0]},
	}
}

// assertWeiwuYellowTurbanDispatchSnapshot 核对黄巾主战报和援军战报统一采用派出快照。
func assertWeiwuYellowTurbanDispatchSnapshot(t *testing.T, result weiwuYellowTurbanReinforcementResult, expected bool, expectedDefensePower int) {
	t.Helper()
	if result.defensePower != expectedDefensePower {
		t.Fatalf("expected yellow turban defense power %d, got %d", expectedDefensePower, result.defensePower)
	}
	if len(result.record.Generals) != 1 || reinforcementSnapshotHasTrait(result.record.Generals[0], "weiwu_tongyu") != expected {
		t.Fatalf("expected stored yellow turban deployment trait=%v, record=%+v", expected, result.record)
	}
	for _, report := range result.reports {
		_, hasOutcome := report.TraitOutcomes["weiwu_tongyu"]
		hasTimeline := report.Detail != nil && standardReportHasTrait(report.Detail, "weiwu_tongyu")
		if hasOutcome != expected || hasTimeline != expected {
			t.Fatalf("expected yellow turban report outcome and timeline trait=%v, report=%s outcomes=%+v detail=%+v", expected, report.ID, report.TraitOutcomes, report.Detail)
		}
		if len(report.PvpReinforcements) != 1 || len(report.PvpReinforcements[0].Generals) != 1 || reinforcementSnapshotHasTrait(report.PvpReinforcements[0].Generals[0], "weiwu_tongyu") != expected {
			t.Fatalf("expected yellow turban report deployment snapshot trait=%v, report=%s reinforcements=%+v", expected, report.ID, report.PvpReinforcements)
		}
	}
}

// TestPvpReinforcementWithoutGeneralDoesNotBorrowHomeGeneral 验证无将援军不借用城内曹操的属性、特性、经验或战报快照。
func TestPvpReinforcementWithoutGeneralDoesNotBorrowHomeGeneral(t *testing.T) {
	result := runWeiwuReinforcementPvpWithGenerals(t, true, nil)
	if result.defensePower != 1010 {
		t.Fatalf("expected general-free reinforcement to keep base defense power 1010, got %v", result.defensePower)
	}
	if len(result.record.Generals) != 0 || len(result.record.BuffSnapshot) != 0 {
		t.Fatalf("expected no borrowed home-general snapshot or buffs, record=%+v", result.record)
	}
	if got := pvpTestGeneralExp(result.helperState, "caocao"); got != 0 {
		t.Fatalf("expected home Cao Cao exp unchanged, got %d", got)
	}
	for _, report := range result.reports {
		if len(report.TraitTriggered) != 0 || len(report.TraitOutcomes) != 0 || standardReportHasTrait(report.Detail, "weiwu_tongyu") {
			t.Fatalf("expected no borrowed Weiwu Tongyu outcome, report=%s triggered=%+v outcomes=%+v detail=%+v", report.ID, report.TraitTriggered, report.TraitOutcomes, report.Detail)
		}
		if len(report.PvpReinforcements) != 1 || len(report.PvpReinforcements[0].Generals) != 0 || report.PvpReinforcements[0].GeneralExpGained != 0 || report.PvpReinforcements[0].GeneralLevelBefore != 0 || report.PvpReinforcements[0].GeneralLevelAfter != 0 {
			t.Fatalf("expected general-free reinforcement snapshot without progress, report=%s reinforcements=%+v", report.ID, report.PvpReinforcements)
		}
	}
	helperReport := result.reports[2]
	if helperReport.GeneralExpGained != 0 || helperReport.GeneralLevelBefore != 0 || helperReport.GeneralLevelAfter != 0 {
		t.Fatalf("expected reinforcement owner report without general progress, report=%+v", helperReport)
	}
}
