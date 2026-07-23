// 本文件验证正式行军特性 ID 在真实 PVP 出征与增援事务中修改最终时长和速度记录。
package game

import (
	"math"
	"testing"
	"time"

	"hero3/internal/core/general"
)

type realMarchTraitCase struct {
	name         string
	faction      string
	generalID    string
	generalName  string
	specialTrait GeneralTraitConfig
	bonusTrait   GeneralTraitConfig
	rates        []float64
	minimum      int
}

// marchTraitConfig 构造可确定触发的正式行军特性测试配置。
func marchTraitConfig(traitID string, traitType string, rate float64, minimum int, chance float64) GeneralTraitConfig {
	return GeneralTraitConfig{
		TraitID: traitID, TraitType: traitType, Enabled: true, Scope: "self_army",
		Params: map[string]float64{"speedBonusRate": rate, "minMarchSeconds": float64(minimum), "triggerChance": chance},
	}
}

// setRealMarchTraitGenerals 配置被测将领及一个无行军特性的敌方将领。
func setRealMarchTraitGenerals(t *testing.T, tc realMarchTraitCase, enabled bool) (string, string, string) {
	t.Helper()
	hero := GeneralHeroConfig{ID: tc.generalID, Name: tc.generalName, Faction: tc.faction, Enabled: true}
	if enabled {
		hero.SpecialTrait = tc.specialTrait
		hero.BonusTrait = tc.bonusTrait
	}
	opponentFaction, opponentID, opponentName := "wei", "caocao", "曹操"
	if tc.faction == "wei" {
		opponentFaction, opponentID, opponentName = "shu", "liubei", "刘备"
	}
	setTestFactionsAndGenerals(t, FactionsConfig{
		tc.faction:      {Name: tc.faction, Generals: []GeneralInfo{{ID: tc.generalID, Name: tc.generalName}}},
		opponentFaction: {Name: opponentFaction, Generals: []GeneralInfo{{ID: opponentID, Name: opponentName}}},
	}, GeneralsConfig{Enabled: true, Heroes: map[string]GeneralHeroConfig{
		tc.generalID: hero,
		opponentID:   {ID: opponentID, Name: opponentName, Faction: opponentFaction, Enabled: true},
	}})
	return opponentFaction, opponentID, tc.faction + "Infantry"
}

// runRealMarchTraitPvp 创建一条被测将领领军的真实 PVP 出征记录。
func runRealMarchTraitPvp(t *testing.T, tc realMarchTraitCase, enabled bool) PvpMarch {
	t.Helper()
	opponentFaction, opponentID, unitType := setRealMarchTraitGenerals(t, tc, enabled)
	svc, repo, attacker, defender := newPvpTestServiceForGenerals(t, tc.faction, tc.generalID, opponentFaction, opponentID)
	attacker.Army = []ArmyUnit{{UnitType: unitType, Amount: 100}}
	defender.Army = []ArmyUnit{{UnitType: opponentFaction + "Infantry", Amount: 100}}
	repo.players[attacker.Player.ID] = attacker
	repo.players[defender.Player.ID] = defender
	if _, err := repo.AssignWorldPosition(attacker.Player.ID, defaultWorldID, 10, 10, "test"); err != nil {
		t.Fatalf("AssignWorldPosition attacker failed: %v", err)
	}
	if _, err := repo.AssignWorldPosition(defender.Player.ID, defaultWorldID, 20, 10, "test"); err != nil {
		t.Fatalf("AssignWorldPosition defender failed: %v", err)
	}
	result, err := svc.StartPvpAttack(PvpAttackRequest{
		PlayerID: attacker.Player.ID, TargetPlayerID: defender.Player.ID, MarchMode: PvpMarchTypeAttack,
		Troops: map[string]int{unitType: 100}, GeneralIDs: []string{tc.generalID},
	})
	if err != nil {
		t.Fatalf("StartPvpAttack failed: %v", err)
	}
	assertNoPreCombatMarchReport(t, repo, attacker.Player.ID)
	return result.March
}

// runRealMarchTraitReinforcement 创建一条被测将领领军的真实增援记录。
func runRealMarchTraitReinforcement(t *testing.T, tc realMarchTraitCase, enabled bool) Reinforcement {
	t.Helper()
	opponentFaction, opponentID, unitType := setRealMarchTraitGenerals(t, tc, enabled)
	svc, repo, from, to := newPvpTestServiceForGenerals(t, tc.faction, tc.generalID, opponentFaction, opponentID)
	from.Army = []ArmyUnit{{UnitType: unitType, Amount: 100}}
	to.Army = []ArmyUnit{{UnitType: opponentFaction + "Infantry", Amount: 100}}
	repo.players[from.Player.ID] = from
	repo.players[to.Player.ID] = to
	if _, err := repo.AssignWorldPosition(from.Player.ID, defaultWorldID, 10, 10, "test"); err != nil {
		t.Fatalf("AssignWorldPosition from failed: %v", err)
	}
	if _, err := repo.AssignWorldPosition(to.Player.ID, defaultWorldID, 20, 10, "test"); err != nil {
		t.Fatalf("AssignWorldPosition to failed: %v", err)
	}
	result, err := svc.SendReinforcement(SendReinforcementRequest{
		FromPlayerID: from.Player.ID, TargetPlayerID: to.Player.ID,
		Troops: map[string]int{unitType: 100}, GeneralIDs: []string{tc.generalID},
	})
	if err != nil {
		t.Fatalf("SendReinforcement failed: %v", err)
	}
	assertNoPreCombatMarchReport(t, repo, from.Player.ID)
	return result.Reinforcement
}

// assertNoPreCombatMarchReport 验证过程类行军特性不会伪造战斗战报。
func assertNoPreCombatMarchReport(t *testing.T, repo *MemoryRepository, playerID string) {
	t.Helper()
	reports, total, err := repo.ListReports(playerID, 10, 0)
	if err != nil || total != 0 || len(reports) != 0 {
		t.Fatalf("expected no battle report before combat, reports=%+v total=%d err=%v", reports, total, err)
	}
}

// applyExpectedMarchRates 按核心逐项向上取整规则计算最终行军秒数。
func applyExpectedMarchRates(base int, rates []float64, minimum int) int {
	result := base
	for _, rate := range rates {
		result = int(math.Ceil(float64(result) / (1 + rate)))
		if result < minimum {
			result = minimum
		}
	}
	return result
}

// TestFormalMarchTraitIDsChangeRealPvpAndReinforcement 验证现行行军特性都进入真实出征和增援记录。
func TestFormalMarchTraitIDsChangeRealPvpAndReinforcement(t *testing.T) {
	tests := []realMarchTraitCase{
		{
			name: "吕蒙白衣渡江", faction: "wu", generalID: "lvmeng", generalName: "吕蒙",
			specialTrait: marchTraitConfig("baiyi_dujiang", general.TraitTypeSpecial, 0.2, 60, 1),
			rates:        []float64{0.2}, minimum: 60,
		},
		{
			name: "吕蒙白衣急行", faction: "wu", generalID: "lvmeng", generalName: "吕蒙",
			bonusTrait: marchTraitConfig("baiyi_jixing", general.TraitTypeBonus, 0.2, 60, 1),
			rates:      []float64{0.2}, minimum: 60,
		},
		{
			name: "太史慈快如闪电", faction: "wu", generalID: "taishici", generalName: "太史慈",
			specialTrait: marchTraitConfig("kuairu_shandian", general.TraitTypeSpecial, 4, 30, 1),
			rates:        []float64{4}, minimum: 30,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			controlPvp := runRealMarchTraitPvp(t, tc, false)
			activePvp := runRealMarchTraitPvp(t, tc, true)
			wantPvp := applyExpectedMarchRates(controlPvp.DurationSeconds, tc.rates, tc.minimum)
			wantPvpSpeed := controlPvp.SpeedMultiplier * float64(controlPvp.DurationSeconds) / float64(wantPvp)
			if activePvp.DurationSeconds != wantPvp || math.Abs(activePvp.SpeedMultiplier-wantPvpSpeed) > 1e-9 {
				t.Fatalf("expected PVP duration %d and speed %.6f, control=%+v active=%+v", wantPvp, wantPvpSpeed, controlPvp, activePvp)
			}
			assertMarchDurationTimestamps(t, activePvp.StartedAt, activePvp.ArrivesAt, activePvp.DurationSeconds)

			controlReinforcement := runRealMarchTraitReinforcement(t, tc, false)
			activeReinforcement := runRealMarchTraitReinforcement(t, tc, true)
			wantReinforcement := applyExpectedMarchRates(controlReinforcement.MarchSeconds, tc.rates, tc.minimum)
			wantReinforcementSpeed := controlReinforcement.SpeedMultiplier * float64(controlReinforcement.MarchSeconds) / float64(wantReinforcement)
			if activeReinforcement.MarchSeconds != wantReinforcement || activeReinforcement.ReturnSeconds != wantReinforcement || math.Abs(activeReinforcement.SpeedMultiplier-wantReinforcementSpeed) > 1e-9 {
				t.Fatalf("expected reinforcement duration %d and speed %.6f, control=%+v active=%+v", wantReinforcement, wantReinforcementSpeed, controlReinforcement, activeReinforcement)
			}
			assertMarchDurationTimestamps(t, activeReinforcement.SentAt, activeReinforcement.ExpectedArriveAt, activeReinforcement.MarchSeconds)
		})
	}
}

// TestLvmengDualMarchTraitsStackInRealTransactions 验证吕蒙双特性在真实事务中逐项生效并按次序取整。
func TestLvmengDualMarchTraitsStackInRealTransactions(t *testing.T) {
	tc := realMarchTraitCase{
		name: "吕蒙双行军特性", faction: "wu", generalID: "lvmeng", generalName: "吕蒙",
		specialTrait: marchTraitConfig("baiyi_dujiang", general.TraitTypeSpecial, 0.2, 60, 1),
		bonusTrait:   marchTraitConfig("baiyi_jixing", general.TraitTypeBonus, 0.2, 60, 1),
		rates:        []float64{0.2, 0.2}, minimum: 60,
	}
	controlPvp := runRealMarchTraitPvp(t, tc, false)
	activePvp := runRealMarchTraitPvp(t, tc, true)
	wantPvp := applyExpectedMarchRates(controlPvp.DurationSeconds, tc.rates, tc.minimum)
	if activePvp.DurationSeconds != wantPvp {
		t.Fatalf("expected dual-trait PVP duration %d, control=%+v active=%+v", wantPvp, controlPvp, activePvp)
	}
	assertMarchDurationTimestamps(t, activePvp.StartedAt, activePvp.ArrivesAt, activePvp.DurationSeconds)

	controlReinforcement := runRealMarchTraitReinforcement(t, tc, false)
	activeReinforcement := runRealMarchTraitReinforcement(t, tc, true)
	wantReinforcement := applyExpectedMarchRates(controlReinforcement.MarchSeconds, tc.rates, tc.minimum)
	if activeReinforcement.MarchSeconds != wantReinforcement || activeReinforcement.ReturnSeconds != wantReinforcement {
		t.Fatalf("expected dual-trait reinforcement duration %d, control=%+v active=%+v", wantReinforcement, controlReinforcement, activeReinforcement)
	}
	assertMarchDurationTimestamps(t, activeReinforcement.SentAt, activeReinforcement.ExpectedArriveAt, activeReinforcement.MarchSeconds)
}

// TestLvmengDualMarchTraitsStayOutOfBattleTimeline 验证吕蒙双加速生效后不会伪装成战斗触发特性。
func TestLvmengDualMarchTraitsStayOutOfBattleTimeline(t *testing.T) {
	tc := realMarchTraitCase{
		name: "吕蒙双行军特性完整流程", faction: "wu", generalID: "lvmeng", generalName: "吕蒙",
		specialTrait: marchTraitConfig("baiyi_dujiang", general.TraitTypeSpecial, 0.2, 60, 1),
		bonusTrait:   marchTraitConfig("baiyi_jixing", general.TraitTypeBonus, 0.2, 60, 1),
		rates:        []float64{0.2, 0.2}, minimum: 60,
	}
	controlMarch := runRealMarchTraitPvp(t, tc, false)
	opponentFaction, opponentID, unitType := setRealMarchTraitGenerals(t, tc, true)
	svc, repo, attacker, defender := newPvpTestServiceForGenerals(t, tc.faction, tc.generalID, opponentFaction, opponentID)
	attacker.Army = []ArmyUnit{{UnitType: unitType, Amount: 100}}
	defender.Army = []ArmyUnit{{UnitType: opponentFaction + "Infantry", Amount: 1}}
	repo.players[attacker.Player.ID] = attacker
	repo.players[defender.Player.ID] = defender
	if _, err := repo.AssignWorldPosition(attacker.Player.ID, defaultWorldID, 10, 10, "test"); err != nil {
		t.Fatalf("AssignWorldPosition attacker failed: %v", err)
	}
	if _, err := repo.AssignWorldPosition(defender.Player.ID, defaultWorldID, 20, 10, "test"); err != nil {
		t.Fatalf("AssignWorldPosition defender failed: %v", err)
	}
	started, err := svc.StartPvpAttack(PvpAttackRequest{
		PlayerID: attacker.Player.ID, TargetPlayerID: defender.Player.ID, MarchMode: PvpMarchTypeAttack,
		Troops: map[string]int{unitType: 100}, GeneralIDs: []string{tc.generalID},
	})
	if err != nil {
		t.Fatalf("StartPvpAttack failed: %v", err)
	}
	wantDuration := applyExpectedMarchRates(controlMarch.DurationSeconds, tc.rates, tc.minimum)
	if started.March.DurationSeconds != wantDuration {
		t.Fatalf("expected dual march traits to reduce duration %d -> %d, march=%+v", controlMarch.DurationSeconds, wantDuration, started.March)
	}
	assertNoPreCombatMarchReport(t, repo, attacker.Player.ID)
	forcePvpMarchDueWithOutboundDuration(t, repo, started.March.ID, time.Duration(wantDuration)*time.Second)
	battle, err := svc.ResolvePvpMarch(started.March.ID)
	if err != nil {
		t.Fatalf("ResolvePvpMarch failed: %v", err)
	}
	resolvedMarch, err := repo.GetPvpMarch(started.March.ID)
	if err != nil || resolvedMarch.Status != PvpMarchStatusReturning {
		t.Fatalf("expected resolved march returning, march=%+v err=%v", resolvedMarch, err)
	}
	assertMarchDurationTimestamps(t, resolvedMarch.ReturnStartedAt, resolvedMarch.ReturnsAt, wantDuration)
	attackerReport, err := repo.GetReportForPlayer(attacker.Player.ID, battle.AttackerReportID)
	if err != nil {
		t.Fatalf("GetReportForPlayer attacker failed: %v", err)
	}
	defenderReport, err := repo.GetReportForPlayer(defender.Player.ID, battle.DefenderReportID)
	if err != nil {
		t.Fatalf("GetReportForPlayer defender failed: %v", err)
	}
	for _, report := range []BattleReport{attackerReport, defenderReport} {
		if len(report.PvpAttackerGenerals) != 1 || report.PvpAttackerGenerals[0].ID != "lvmeng" || len(report.PvpAttackerGenerals[0].Traits) != 2 {
			t.Fatalf("expected report to retain Lvmeng owned traits snapshot, report=%s generals=%+v", report.ID, report.PvpAttackerGenerals)
		}
		if len(report.TraitTriggered) != 0 || len(report.TraitOutcomes) != 0 || report.Detail == nil || len(report.Detail.Traits) != 0 {
			t.Fatalf("expected march traits absent from battle timeline, report=%s triggered=%+v outcomes=%+v detail=%+v", report.ID, report.TraitTriggered, report.TraitOutcomes, report.Detail)
		}
	}
	stored, err := repo.GetState(attacker.Player.ID)
	if err != nil || pvpTestGeneralExp(stored, "lvmeng") != attackerReport.GeneralExpGained {
		t.Fatalf("expected normal battle exp without march trait outcome, generals=%+v report=%+v err=%v", stored.Generals, attackerReport, err)
	}
}

// TestLvmengBaiyiDujiangLegalMissKeepsBaiyiJixing 验证白衣渡江合法未命中时不加速，白衣急行仍独立进入真实出征和返程。
func TestLvmengBaiyiDujiangLegalMissKeepsBaiyiJixing(t *testing.T) {
	tc := realMarchTraitCase{
		name: "吕蒙白衣渡江合法未命中", faction: "wu", generalID: "lvmeng", generalName: "吕蒙",
		specialTrait: marchTraitConfig("baiyi_dujiang", general.TraitTypeSpecial, 0.2, 60, 0),
		bonusTrait:   marchTraitConfig("baiyi_jixing", general.TraitTypeBonus, 0.2, 60, 1),
		rates:        []float64{0.2}, minimum: 60,
	}
	controlMarch := runRealMarchTraitPvp(t, tc, false)
	if controlMarch.DurationSeconds != 2970 {
		t.Fatalf("expected authoritative no-trait baseline 2970 seconds, control=%+v", controlMarch)
	}
	opponentFaction, opponentID, unitType := setRealMarchTraitGenerals(t, tc, true)
	svc, repo, attacker, defender := newPvpTestServiceForGenerals(t, tc.faction, tc.generalID, opponentFaction, opponentID)
	attacker.Army = []ArmyUnit{{UnitType: unitType, Amount: 100}}
	defender.Army = []ArmyUnit{{UnitType: opponentFaction + "Infantry", Amount: 1}}
	defender.Buildings = nil
	repo.players[attacker.Player.ID] = attacker
	repo.players[defender.Player.ID] = defender
	if _, err := repo.AssignWorldPosition(attacker.Player.ID, defaultWorldID, 10, 10, "test"); err != nil {
		t.Fatalf("AssignWorldPosition attacker failed: %v", err)
	}
	if _, err := repo.AssignWorldPosition(defender.Player.ID, defaultWorldID, 20, 10, "test"); err != nil {
		t.Fatalf("AssignWorldPosition defender failed: %v", err)
	}
	started, err := svc.StartPvpAttack(PvpAttackRequest{
		PlayerID: attacker.Player.ID, TargetPlayerID: defender.Player.ID, MarchMode: PvpMarchTypeAttack,
		Troops: map[string]int{unitType: 100}, GeneralIDs: []string{tc.generalID},
	})
	if err != nil {
		t.Fatalf("StartPvpAttack failed: %v", err)
	}
	wantDuration := applyExpectedMarchRates(controlMarch.DurationSeconds, tc.rates, tc.minimum)
	wantSpeed := controlMarch.SpeedMultiplier * float64(controlMarch.DurationSeconds) / float64(wantDuration)
	if wantDuration != 2475 || started.March.DurationSeconds != wantDuration || math.Abs(started.March.SpeedMultiplier-wantSpeed) > 1e-9 {
		t.Fatalf("expected only Baiyi Jixing to reduce duration 2970 -> 2475 with speed %.6f, march=%+v", wantSpeed, started.March)
	}
	assertMarchDurationTimestamps(t, started.March.StartedAt, started.March.ArrivesAt, wantDuration)
	assertNoPreCombatMarchReport(t, repo, attacker.Player.ID)
	forcePvpMarchDueWithOutboundDuration(t, repo, started.March.ID, time.Duration(wantDuration)*time.Second)
	battle, err := svc.ResolvePvpMarch(started.March.ID)
	if err != nil {
		t.Fatalf("ResolvePvpMarch failed: %v", err)
	}
	attackPower, attackOK := battle.Result["attackerPower"].(float64)
	defensePower, defenseOK := battle.Result["defensePower"].(float64)
	if !attackOK || !defenseOK || attackPower != 1000 || defensePower != 10 || battle.Result["winner"] != "attacker" {
		t.Fatalf("expected march traits not to alter 1000/10 attacker victory, result=%+v", battle.Result)
	}
	attackerLosses := pvpTestLossesFromBattle(t, battle, "attacker")
	defenderLosses := pvpTestLossesFromBattle(t, battle, "defender")
	if attackerLosses[unitType] != 0 || defenderLosses[opponentFaction+"Infantry"] != 1 {
		t.Fatalf("expected exact core losses 0/1 after arrival, attacker=%+v defender=%+v", attackerLosses, defenderLosses)
	}
	resolvedMarch, err := repo.GetPvpMarch(started.March.ID)
	if err != nil || resolvedMarch.Status != PvpMarchStatusReturning || resolvedMarch.AttackTroops[unitType] != 100 {
		t.Fatalf("expected all 100 attackers returning, march=%+v err=%v", resolvedMarch, err)
	}
	assertMarchDurationTimestamps(t, resolvedMarch.ReturnStartedAt, resolvedMarch.ReturnsAt, wantDuration)
	attackerReport, err := repo.GetReportForPlayer(attacker.Player.ID, battle.AttackerReportID)
	if err != nil {
		t.Fatalf("GetReportForPlayer attacker failed: %v", err)
	}
	defenderReport, err := repo.GetReportForPlayer(defender.Player.ID, battle.DefenderReportID)
	if err != nil {
		t.Fatalf("GetReportForPlayer defender failed: %v", err)
	}
	for _, report := range []BattleReport{attackerReport, defenderReport} {
		if len(report.PvpAttackerGenerals) != 1 || report.PvpAttackerGenerals[0].ID != "lvmeng" || !pvpSnapshotHasTrait(report.PvpAttackerGenerals[0], "baiyi_dujiang") || !pvpSnapshotHasTrait(report.PvpAttackerGenerals[0], "baiyi_jixing") {
			t.Fatalf("expected report to retain both Lvmeng owned traits, report=%s generals=%+v", report.ID, report.PvpAttackerGenerals)
		}
		if len(report.TraitTriggered) != 0 || len(report.TraitOutcomes) != 0 || report.Detail == nil || report.Detail.SecondarySide == nil || len(report.Detail.Traits) != 0 {
			t.Fatalf("expected missed and process-only march traits absent from battle timelines, report=%+v", report)
		}
		if !standardDetailGeneralHasTrait(report.Detail, "baiyi_dujiang") || !standardDetailGeneralHasTrait(report.Detail, "baiyi_jixing") {
			t.Fatalf("expected standard Lvmeng snapshot to retain both owned traits, report=%s detail=%+v", report.ID, report.Detail)
		}
		attackerUnit := attackDefenseCrossReportUnit(t, report.Detail.PrimarySide, unitType)
		defenderUnit := attackDefenseCrossReportUnit(t, *report.Detail.SecondarySide, opponentFaction+"Infantry")
		if attackerUnit.AmountBefore != 100 || attackerUnit.Lost != 0 || attackerUnit.Survived != 100 || defenderUnit.AmountBefore != 1 || defenderUnit.Lost != 1 || defenderUnit.Survived != 0 {
			t.Fatalf("expected standard rows 100/0/100 and 1/1/0, report=%s attacker=%+v defender=%+v", report.ID, attackerUnit, defenderUnit)
		}
	}
	stored, err := repo.GetState(attacker.Player.ID)
	if err != nil || pvpTestGeneralExp(stored, "lvmeng") != 1 || attackerReport.GeneralExpGained != 1 || attackerReport.Detail.Rewards.GeneralExp != 1 {
		t.Fatalf("expected Lvmeng to gain exact 1 exp from real defender loss, state=%+v report=%+v err=%v", stored.Generals, attackerReport, err)
	}
}

// TestLvmengDualMarchTraitsStayOutOfReinforcementBattleTimeline 验证吕蒙双加速援军参战后只结算真实战损和经验。
func TestLvmengDualMarchTraitsStayOutOfReinforcementBattleTimeline(t *testing.T) {
	tc := realMarchTraitCase{
		name: "吕蒙双行军特性援军完整流程", faction: "wu", generalID: "lvmeng", generalName: "吕蒙",
		specialTrait: marchTraitConfig("baiyi_dujiang", general.TraitTypeSpecial, 0.2, 60, 1),
		bonusTrait:   marchTraitConfig("baiyi_jixing", general.TraitTypeBonus, 0.2, 60, 1),
		rates:        []float64{0.2, 0.2}, minimum: 60,
	}
	controlReinforcement := runRealMarchTraitReinforcement(t, tc, false)
	setTestFactionsAndGenerals(t, FactionsConfig{
		"wei": {Name: "魏国", Generals: []GeneralInfo{{ID: "caocao", Name: "曹操"}}},
		"shu": {Name: "蜀国", Generals: []GeneralInfo{{ID: "liubei", Name: "刘备"}}},
		"wu":  {Name: "吴国", Generals: []GeneralInfo{{ID: "lvmeng", Name: "吕蒙"}}},
	}, GeneralsConfig{Enabled: true, Heroes: map[string]GeneralHeroConfig{
		"caocao": {ID: "caocao", Name: "曹操", Faction: "wei", Enabled: true},
		"liubei": {ID: "liubei", Name: "刘备", Faction: "shu", Enabled: true},
		"lvmeng": {
			ID: "lvmeng", Name: "吕蒙", Faction: "wu", Enabled: true,
			SpecialTrait: tc.specialTrait, BonusTrait: tc.bonusTrait,
		},
	}})
	svc, repo, attacker, defender := newPvpTestServiceForGenerals(t, "wei", "caocao", "shu", "liubei")
	unitsMu.Lock()
	if activeUnits["wu"] == nil {
		activeUnits["wu"] = FactionUnits{}
	}
	activeUnits["wu"]["wuInfantry"] = UnitConfig{
		Name: "吴步兵", Category: "infantry",
		Stats: map[string]int{"attack": 10, "infantryDefense": 10, "cavalryDefense": 8, "speed": 1, "carryCapacity": 5, "upkeep": 1},
	}
	unitsMu.Unlock()
	now := time.Now().UTC()
	helperAccount := Account{ID: "account_lvmeng_reinforcement_flow", Username: "lvmeng_reinforcement_flow", CreatedAt: now}
	if err := repo.CreateAccount(helperAccount); err != nil {
		t.Fatalf("CreateAccount helper failed: %v", err)
	}
	helper := newPlayerState("player_lvmeng_reinforcement_flow", "吕蒙援军", "wu", "lvmeng", now)
	baselineExp := generalExpRequiredForLevelForTest(2) - 1
	setPvpTestGeneralProgress(&helper, "lvmeng", 1, baselineExp)
	helper.Army = []ArmyUnit{{UnitType: "wuInfantry", Amount: 100}}
	if err := repo.CreatePlayer(helperAccount.ID, helper, now); err != nil {
		t.Fatalf("CreatePlayer helper failed: %v", err)
	}
	attacker.Army = []ArmyUnit{{UnitType: "weiInfantry", Amount: 100}}
	defender.Army = []ArmyUnit{{UnitType: "shuInfantry", Amount: 1}}
	defender.Buildings = nil
	repo.players[attacker.Player.ID] = attacker
	repo.players[defender.Player.ID] = defender
	for _, position := range []struct {
		playerID string
		x        int
		y        int
	}{
		{playerID: helper.Player.ID, x: 10, y: 10},
		{playerID: defender.Player.ID, x: 20, y: 10},
		{playerID: attacker.Player.ID, x: 30, y: 10},
	} {
		if _, err := repo.AssignWorldPosition(position.playerID, defaultWorldID, position.x, position.y, "test"); err != nil {
			t.Fatalf("AssignWorldPosition %s failed: %v", position.playerID, err)
		}
	}
	sent, err := svc.SendReinforcement(SendReinforcementRequest{
		FromPlayerID: helper.Player.ID, TargetPlayerID: defender.Player.ID,
		Troops: map[string]int{"wuInfantry": 100}, GeneralIDs: []string{"lvmeng"},
	})
	if err != nil {
		t.Fatalf("SendReinforcement failed: %v", err)
	}
	wantMarchSeconds := applyExpectedMarchRates(controlReinforcement.MarchSeconds, tc.rates, tc.minimum)
	if sent.Reinforcement.MarchSeconds != wantMarchSeconds || sent.Reinforcement.ReturnSeconds != wantMarchSeconds {
		t.Fatalf("expected Lvmeng reinforcement duration %d, control=%+v active=%+v", wantMarchSeconds, controlReinforcement, sent.Reinforcement)
	}
	assertNoPreCombatMarchReport(t, repo, helper.Player.ID)
	forceReinforcementDue(t, repo, sent.Reinforcement.ID, true)
	if _, err := svc.MarkReinforcementArrived(sent.Reinforcement.ID); err != nil {
		t.Fatalf("MarkReinforcementArrived failed: %v", err)
	}
	started, err := svc.StartPvpAttack(PvpAttackRequest{
		PlayerID: attacker.Player.ID, TargetPlayerID: defender.Player.ID, MarchMode: PvpMarchTypeAttack,
		Troops: map[string]int{"weiInfantry": 100}, GeneralIDs: []string{"caocao"},
	})
	if err != nil {
		t.Fatalf("StartPvpAttack failed: %v", err)
	}
	forcePvpMarchDue(t, repo, started.March.ID)
	battle, err := svc.ResolvePvpMarch(started.March.ID)
	if err != nil {
		t.Fatalf("ResolvePvpMarch failed: %v", err)
	}
	attackPower, attackOK := battle.Result["attackerPower"].(float64)
	defensePower, defenseOK := battle.Result["defensePower"].(float64)
	if !attackOK || !defenseOK || attackPower != 1000 || defensePower != 1010 || battle.Result["winner"] != "defender" {
		t.Fatalf("expected exact 1000/1010 defender victory, result=%+v", battle.Result)
	}
	attackerReport, err := repo.GetReportForPlayer(attacker.Player.ID, battle.AttackerReportID)
	if err != nil {
		t.Fatalf("GetReportForPlayer attacker failed: %v", err)
	}
	defenderReport, err := repo.GetReportForPlayer(defender.Player.ID, battle.DefenderReportID)
	if err != nil {
		t.Fatalf("GetReportForPlayer defender failed: %v", err)
	}
	helperReports, total, err := repo.ListReportsByQuery(BattleReportQuery{PlayerID: helper.Player.ID, ViewType: ReportViewReinforcement, Page: 1, PageSize: 10})
	if err != nil || total != 1 || len(helperReports) != 1 {
		t.Fatalf("expected one helper report, reports=%+v total=%d err=%v", helperReports, total, err)
	}
	for _, report := range []BattleReport{attackerReport, defenderReport, helperReports[0]} {
		if len(report.PvpReinforcements) != 1 || len(report.PvpReinforcements[0].Generals) != 1 || report.PvpReinforcements[0].Generals[0].ID != "lvmeng" || len(report.PvpReinforcements[0].Generals[0].Traits) != 2 {
			t.Fatalf("expected Lvmeng reinforcement owned traits snapshot, report=%s reinforcements=%+v", report.ID, report.PvpReinforcements)
		}
		if len(report.TraitTriggered) != 0 || len(report.TraitOutcomes) != 0 || report.Detail == nil || len(report.Detail.Traits) != 0 {
			t.Fatalf("expected march traits absent from reinforcement battle timeline, report=%s triggered=%+v outcomes=%+v detail=%+v", report.ID, report.TraitTriggered, report.TraitOutcomes, report.Detail)
		}
		if report.PvpReinforcementLosses[sent.Reinforcement.ID]["wuInfantry"] != 98 {
			t.Fatalf("expected three reports to agree on 98 Lvmeng reinforcement losses, report=%s losses=%+v", report.ID, report.PvpReinforcementLosses)
		}
	}
	record, err := repo.GetReinforcement(sent.Reinforcement.ID)
	if err != nil {
		t.Fatalf("GetReinforcement failed: %v", err)
	}
	helperState, err := repo.GetState(helper.Player.ID)
	if err != nil {
		t.Fatalf("GetState helper failed: %v", err)
	}
	gained := helperReports[0].GeneralExpGained
	if attackerReport.LostUnits["weiInfantry"] != 100 || defenderReport.LostUnits["shuInfantry"] != 1 || helperReports[0].LostUnits["wuInfantry"] != 98 || helperReports[0].SurvivedUnits["wuInfantry"] != 2 {
		t.Fatalf("expected exact attacker/defender/helper losses 100/1/98 and helper survivors 2, reports=%+v/%+v/%+v", attackerReport, defenderReport, helperReports[0])
	}
	if gained != 100 || helperReports[0].Detail.Rewards.GeneralExp != 100 || helperReports[0].GeneralLevelBefore != 1 || helperReports[0].GeneralLevelAfter != 2 {
		t.Fatalf("expected authoritative helper exp and level up, report=%+v", helperReports[0])
	}
	if len(record.Generals) != 1 || record.Generals[0].Exp != baselineExp+100 || record.Generals[0].Level != 2 || pvpTestGeneralExp(helperState, "lvmeng") != baselineExp+100 || pvpTestGeneralLevel(helperState, "lvmeng") != 2 {
		t.Fatalf("expected reinforcement record and owner state to agree, record=%+v state=%+v report=%+v", record.Generals, helperState.Generals, helperReports[0])
	}
	if record.LastBattleReportID != helperReports[0].ID || record.RemainingTroops["wuInfantry"] != helperReports[0].SurvivedUnits["wuInfantry"] || record.Losses["wuInfantry"] != helperReports[0].LostUnits["wuInfantry"] {
		t.Fatalf("expected reinforcement state and helper report troops to agree, record=%+v report=%+v", record, helperReports[0])
	}
}

// TestTaishiciLightningDoesNotActivateReinforcementOnlyTraitOnAttack 验证太史慈主动出征只触发行军加速，信义勇烈方向无效。
func TestTaishiciLightningDoesNotActivateReinforcementOnlyTraitOnAttack(t *testing.T) {
	tc := realMarchTraitCase{
		name: "太史慈主动出征方向约束", faction: "wu", generalID: "taishici", generalName: "太史慈",
		specialTrait: marchTraitConfig("kuairu_shandian", general.TraitTypeSpecial, 4, 30, 1),
		bonusTrait: GeneralTraitConfig{
			TraitID: "xinyi_yonglie", TraitType: general.TraitTypeBonus, Enabled: true,
			Scope: "reinforcement_self", AllowedSides: []string{"reinforcement"},
			Params: map[string]float64{"defenseBonusRate": 0.1},
		},
		rates: []float64{4}, minimum: 30,
	}
	controlMarch := runRealMarchTraitPvp(t, tc, false)
	opponentFaction, opponentID, unitType := setRealMarchTraitGenerals(t, tc, true)
	svc, repo, attacker, defender := newPvpTestServiceForGenerals(t, tc.faction, tc.generalID, opponentFaction, opponentID)
	attacker.Army = []ArmyUnit{{UnitType: unitType, Amount: 1000}}
	defender.Army = []ArmyUnit{{UnitType: opponentFaction + "Infantry", Amount: 1}}
	defender.Buildings = nil
	repo.players[attacker.Player.ID] = attacker
	repo.players[defender.Player.ID] = defender
	if _, err := repo.AssignWorldPosition(attacker.Player.ID, defaultWorldID, 10, 10, "test"); err != nil {
		t.Fatalf("AssignWorldPosition attacker failed: %v", err)
	}
	if _, err := repo.AssignWorldPosition(defender.Player.ID, defaultWorldID, 20, 10, "test"); err != nil {
		t.Fatalf("AssignWorldPosition defender failed: %v", err)
	}
	started, err := svc.StartPvpAttack(PvpAttackRequest{
		PlayerID: attacker.Player.ID, TargetPlayerID: defender.Player.ID, MarchMode: PvpMarchTypeAttack,
		Troops: map[string]int{unitType: 1000}, GeneralIDs: []string{"taishici"},
	})
	if err != nil {
		t.Fatalf("StartPvpAttack failed: %v", err)
	}
	wantDuration := applyExpectedMarchRates(controlMarch.DurationSeconds, tc.rates, tc.minimum)
	if started.March.DurationSeconds != wantDuration {
		t.Fatalf("expected lightning duration %d -> %d, march=%+v", controlMarch.DurationSeconds, wantDuration, started.March)
	}
	assertNoPreCombatMarchReport(t, repo, attacker.Player.ID)
	forcePvpMarchDueWithOutboundDuration(t, repo, started.March.ID, time.Duration(wantDuration)*time.Second)
	battle, err := svc.ResolvePvpMarch(started.March.ID)
	if err != nil {
		t.Fatalf("ResolvePvpMarch failed: %v", err)
	}
	attackPower, attackOK := battle.Result["attackerPower"].(float64)
	defensePower, defenseOK := battle.Result["defensePower"].(float64)
	if !attackOK || !defenseOK || attackPower != 10000 || defensePower != 10 || battle.Result["winner"] != "attacker" {
		t.Fatalf("expected reinforcement-only trait not to alter 10000/10 attacker victory, result=%+v", battle.Result)
	}
	resolvedMarch, err := repo.GetPvpMarch(started.March.ID)
	if err != nil || resolvedMarch.Status != PvpMarchStatusReturning {
		t.Fatalf("expected surviving Taishi Ci army to return, march=%+v err=%v", resolvedMarch, err)
	}
	assertMarchDurationTimestamps(t, resolvedMarch.ReturnStartedAt, resolvedMarch.ReturnsAt, wantDuration)
	attackerReport, err := repo.GetReportForPlayer(attacker.Player.ID, battle.AttackerReportID)
	if err != nil {
		t.Fatalf("GetReportForPlayer attacker failed: %v", err)
	}
	defenderReport, err := repo.GetReportForPlayer(defender.Player.ID, battle.DefenderReportID)
	if err != nil {
		t.Fatalf("GetReportForPlayer defender failed: %v", err)
	}
	for _, report := range []BattleReport{attackerReport, defenderReport} {
		if len(report.PvpAttackerGenerals) != 1 || report.PvpAttackerGenerals[0].ID != "taishici" || len(report.PvpAttackerGenerals[0].Traits) != 2 {
			t.Fatalf("expected Taishi Ci owned traits snapshot, report=%s generals=%+v", report.ID, report.PvpAttackerGenerals)
		}
		if len(report.TraitTriggered) != 0 || len(report.TraitOutcomes) != 0 || report.Detail == nil || len(report.Detail.Traits) != 0 {
			t.Fatalf("expected process and reinforcement-only traits absent from attack timeline, report=%s triggered=%+v outcomes=%+v detail=%+v", report.ID, report.TraitTriggered, report.TraitOutcomes, report.Detail)
		}
	}
	if attackerReport.LostUnits[unitType] != 0 || attackerReport.DefenderLostUnits[opponentFaction+"Infantry"] != 1 {
		t.Fatalf("expected attacker report to preserve own/enemy losses 0/1, report=%+v", attackerReport)
	}
	if defenderReport.LostUnits[opponentFaction+"Infantry"] != 1 || defenderReport.DefenderLostUnits[unitType] != 0 {
		t.Fatalf("expected defender report to preserve own/enemy losses 1/0, report=%+v", defenderReport)
	}
	stored, err := repo.GetState(attacker.Player.ID)
	if err != nil || pvpTestGeneralExp(stored, "taishici") != 1 || attackerReport.GeneralExpGained != 1 {
		t.Fatalf("expected Taishi Ci exp 1 from real losses, generals=%+v report=%+v err=%v", stored.Generals, attackerReport, err)
	}
}
