// 本文件验证被动将领特性进入最终属性，但不会伪装成战斗触发结果。
package game

import (
	"math"
	"testing"
	"time"

	"hero3/internal/core/combat"
	"hero3/internal/core/general"
)

// TestTianshenXiafanAddsForceWithoutBattleTrigger 验证马超天神下凡增加 20 武力并转为攻击加成。
func TestTianshenXiafanAddsForceWithoutBattleTrigger(t *testing.T) {
	setTestFactionsAndGenerals(t, FactionsConfig{
		"shu": {Name: "蜀国", Generals: []GeneralInfo{{ID: "machao", Name: "马超"}}},
	}, GeneralsConfig{
		Enabled: true,
		Heroes: map[string]GeneralHeroConfig{
			"machao": {
				ID: "machao", Name: "马超", Faction: "shu", Enabled: true,
				BonusTrait: GeneralTraitConfig{
					TraitID: "tianshen_xiafan", TraitType: general.TraitTypeBonus, Enabled: true,
					Scope: "self_army", Params: map[string]float64{"forceBonus": 20},
				},
			},
		},
	})

	machao := newGeneral("shu", "machao")
	machao.Stats = map[string]int{"force": 5}
	applyHeroConfigToGeneral(machao)
	if machao.EffectiveStats["force"] != 25 {
		t.Fatalf("expected effective force 25, got %+v", machao.EffectiveStats)
	}
	if math.Abs(machao.Buffs[StatAttackBonus]-0.5) > 1e-9 {
		t.Fatalf("expected 25 force to produce 50%% attack bonus, got %+v", machao.Buffs)
	}

	attacker := combat.Army{Units: []combat.Unit{{ID: "cavalry", Count: 100, Attack: 100}}}
	defender := combat.Army{Units: []combat.Unit{{ID: "infantry", Count: 100, Attack: 100}}}
	ctx := &general.BeforeBattleContext{Attacker: &attacker, Defender: &defender, AttackerOwnsTrait: true, Scene: "attack"}
	general.Dispatch(ctx, buildActiveTraits(machao))
	if _, ok := ctx.Triggered["tianshen_xiafan"]; ok {
		t.Fatalf("expected passive force trait absent from battle outcomes, got %+v", ctx.Triggered)
	}
}

// runTianshenXiafanPvp 执行一场马超主动进攻的真实 PVP，返回战斗和双方战报。
func runTianshenXiafanPvp(t *testing.T, enabled bool) (PvpBattle, []BattleReport) {
	t.Helper()
	machao := GeneralHeroConfig{ID: "machao", Name: "马超", Faction: "shu", Enabled: true}
	if enabled {
		machao.BonusTrait = GeneralTraitConfig{
			TraitID: "tianshen_xiafan", TraitType: general.TraitTypeBonus, Enabled: true, Scope: "self_army",
			Params: map[string]float64{"forceBonus": 20},
		}
	}
	setTestFactionsAndGenerals(t, FactionsConfig{
		"shu": {Name: "蜀国", Generals: []GeneralInfo{{ID: "machao", Name: "马超"}}},
		"wu":  {Name: "吴国", Generals: []GeneralInfo{{ID: "sunquan", Name: "孙权"}}},
	}, GeneralsConfig{Enabled: true, Heroes: map[string]GeneralHeroConfig{
		"machao":  machao,
		"sunquan": {ID: "sunquan", Name: "孙权", Faction: "wu", Enabled: true},
	}})

	svc, repo, attacker, defender := newPvpTestServiceForGenerals(t, "shu", "machao", "wu", "sunquan")
	attacker.Army = []ArmyUnit{{UnitType: "shuInfantry", Amount: 100}}
	defender.Army = []ArmyUnit{{UnitType: "wuInfantry", Amount: 100}}
	defender.Buildings = nil
	repo.players[attacker.Player.ID] = attacker
	repo.players[defender.Player.ID] = defender
	started, err := svc.StartPvpAttack(PvpAttackRequest{
		PlayerID: attacker.Player.ID, TargetPlayerID: defender.Player.ID, MarchMode: PvpMarchTypeAttack,
		Troops: map[string]int{"shuInfantry": 100}, GeneralIDs: []string{"machao"},
	})
	if err != nil {
		t.Fatalf("StartPvpAttack failed: %v", err)
	}
	forcePvpMarchDue(t, repo, started.March.ID)
	battle, err := svc.ResolvePvpMarch(started.March.ID)
	if err != nil {
		t.Fatalf("ResolvePvpMarch failed: %v", err)
	}
	attackerReports, _, err := repo.ListReports(attacker.Player.ID, 10, 0)
	if err != nil || len(attackerReports) == 0 || attackerReports[0].ID != battle.AttackerReportID {
		t.Fatalf("expected attacker report, reports=%+v err=%v", attackerReports, err)
	}
	defenderReports, _, err := repo.ListReports(defender.Player.ID, 10, 0)
	if err != nil || len(defenderReports) == 0 || defenderReports[0].ID != battle.DefenderReportID {
		t.Fatalf("expected defender report, reports=%+v err=%v", defenderReports, err)
	}
	attackerLosses := pvpTestLossesFromBattle(t, battle, "attacker")
	defenderLosses := pvpTestLossesFromBattle(t, battle, "defender")
	storedMarch, err := repo.GetPvpMarch(started.March.ID)
	if err != nil || storedMarch.AttackTroops["shuInfantry"] != 100-attackerLosses["shuInfantry"] {
		t.Fatalf("expected return troops to match attacker losses, march=%+v losses=%+v err=%v", storedMarch, attackerLosses, err)
	}
	storedDefender, err := repo.GetState(defender.Player.ID)
	if err != nil || armySliceToMap(storedDefender.Army)["wuInfantry"] != 100-defenderLosses["wuInfantry"] {
		t.Fatalf("expected defender state to match losses, army=%+v losses=%+v err=%v", storedDefender.Army, defenderLosses, err)
	}
	for _, report := range []BattleReport{attackerReports[0], defenderReports[0]} {
		assertLongdanStandardSide(t, report, "attacker", "shuInfantry", 100, attackerLosses["shuInfantry"])
		assertLongdanStandardSide(t, report, "defender", "wuInfantry", 100, defenderLosses["wuInfantry"])
	}
	return battle, []BattleReport{attackerReports[0], defenderReports[0]}
}

// TestPvpTianshenXiafanChangesPowerWithoutTriggerOutcome 验证被动武力真实提高进攻战力，但不伪造成战斗触发。
func TestPvpTianshenXiafanChangesPowerWithoutTriggerOutcome(t *testing.T) {
	controlBattle, _ := runTianshenXiafanPvp(t, false)
	activeBattle, activeReports := runTianshenXiafanPvp(t, true)
	controlPower, controlOK := controlBattle.Result["attackerPower"].(float64)
	activePower, activeOK := activeBattle.Result["attackerPower"].(float64)
	if !controlOK || !activeOK || controlPower != 1000 || activePower != 1400 {
		t.Fatalf("expected passive force to change real attack power 1000 -> 1400, control=%+v active=%+v", controlBattle.Result, activeBattle.Result)
	}
	for _, report := range activeReports {
		if report.ViewType == ReportViewAttack {
			if report.PlayerPower != 1400 {
				t.Fatalf("expected attacker report player power 1400, got %d", report.PlayerPower)
			}
		} else if report.EnemyPower != 1400 {
			t.Fatalf("expected defender report enemy power 1400, got %d", report.EnemyPower)
		}
		if _, ok := report.TraitOutcomes["tianshen_xiafan"]; ok {
			t.Fatalf("expected passive trait absent from legacy trigger outcomes, got %+v", report.TraitOutcomes)
		}
		for _, trait := range report.Detail.Traits {
			if trait.TraitID == "tianshen_xiafan" {
				t.Fatalf("expected passive trait absent from standard trigger timeline, got %+v", report.Detail.Traits)
			}
		}
	}
	attackerSnapshot := activeReports[0].PvpAttackerGenerals
	if len(attackerSnapshot) != 1 || math.Abs(attackerSnapshot[0].Buffs[StatAttackBonus]-0.4) > 1e-9 {
		t.Fatalf("expected Ma Chao snapshot to carry passive 40%% attack modifier, got %+v", attackerSnapshot)
	}
}

// TestWeiwuHaolingAddsRealTroopsOnResourceSettlement 验证魏武号令按结算时间真实增加军队兵力。
func TestWeiwuHaolingAddsRealTroopsOnResourceSettlement(t *testing.T) {
	setTestCombatUnitsConfig(t)
	setTestFactionsAndGenerals(t, FactionsConfig{
		"wei": {Name: "魏国", Generals: []GeneralInfo{{ID: "caocao", Name: "曹操"}}},
	}, GeneralsConfig{
		Enabled: true,
		Heroes: map[string]GeneralHeroConfig{
			"caocao": {
				ID: "caocao", Name: "曹操", Faction: "wei", Enabled: true,
				SpecialTrait: GeneralTraitConfig{
					TraitID: "weiwu_haoling", TraitType: general.TraitTypeSpecial, Enabled: true,
					Scope: "self_city", TargetUnitType: "weiInfantry",
					Params: map[string]float64{"guardPerMinute": 300},
				},
			},
		},
	})
	state := GameState{Player: Player{Faction: "wei"}, General: newGeneral("wei", "caocao")}
	if changed := applyGuardPerMinuteTraits(&state, 120); !changed {
		t.Fatal("expected guard production to change army")
	}
	if got := armySliceToMap(state.Army)["weiInfantry"]; got != 600 {
		t.Fatalf("expected 600 produced guards after two minutes, got %d", got)
	}
	away := GameState{
		Player: Player{Faction: "wei"}, General: newGeneral("wei", "caocao"),
		GeneralAssignments: []GeneralAssignment{{ID: "march_caocao", GeneralID: "caocao", Slot: "march", Status: "marching"}},
	}
	if changed := applyGuardPerMinuteTraits(&away, 120); changed || len(away.Army) != 0 {
		t.Fatalf("expected Cao Cao away from city to produce no guards, changed=%t army=%+v", changed, away.Army)
	}
}

// TestNeizhengJingyingEntersModifierBuffs 验证内政精营的被动产量会进入真实 Modifier 来源。
func TestNeizhengJingyingEntersModifierBuffs(t *testing.T) {
	setTestFactionsAndGenerals(t, FactionsConfig{
		"wei": {Name: "魏国", Generals: []GeneralInfo{{ID: "xunyu", Name: "荀彧"}}},
	}, GeneralsConfig{
		Enabled: true,
		Heroes: map[string]GeneralHeroConfig{
			"xunyu": {
				ID: "xunyu", Name: "荀彧", Faction: "wei", Enabled: true,
				BonusTrait: GeneralTraitConfig{
					TraitID: "neizheng_jingying", TraitType: general.TraitTypeBonus, Enabled: true,
					Scope: "self_city", Params: map[string]float64{"productionBonusRate": 0.05},
				},
			},
		},
	})
	xunyu := newGeneral("wei", "xunyu")
	applyHeroConfigToGeneral(xunyu)
	if math.Abs(xunyu.Buffs[StatProductionBonus]-0.05) > 1e-9 {
		t.Fatalf("expected production modifier 0.05, got %+v", xunyu.Buffs)
	}
	attacker := combat.Army{Units: []combat.Unit{{ID: "weiInfantry", Count: 100, Attack: 10}}}
	defender := combat.Army{Units: []combat.Unit{{ID: "shuInfantry", Count: 100, Attack: 10}}}
	ctx := &general.BeforeBattleContext{Attacker: &attacker, Defender: &defender, AttackerOwnsTrait: true, Scene: "attack"}
	general.Dispatch(ctx, buildActiveTraits(xunyu))
	if _, ok := ctx.Triggered["neizheng_jingying"]; ok {
		t.Fatalf("expected passive production trait absent from battle outcomes, got %+v", ctx.Triggered)
	}
}

// TestNeizhengJingyingSettlesOnlyWhileGeneralIsHome 验证内政精营真实增加资源产出，武将离城后不再给城内结算提供加成。
func TestNeizhengJingyingSettlesOnlyWhileGeneralIsHome(t *testing.T) {
	setTestFactionsAndGenerals(t, FactionsConfig{
		"wei": {Name: "魏国", Generals: []GeneralInfo{{ID: "xunyu", Name: "荀彧"}}},
	}, GeneralsConfig{
		Enabled: true,
		Heroes: map[string]GeneralHeroConfig{
			"xunyu": {
				ID: "xunyu", Name: "荀彧", Faction: "wei", Enabled: true,
				BonusTrait: GeneralTraitConfig{
					TraitID: "neizheng_jingying", TraitType: general.TraitTypeBonus, Enabled: true,
					Scope: "self_city", Params: map[string]float64{"productionBonusRate": 0.05},
				},
			},
		},
	})
	settledAt := time.Date(2026, 7, 19, 8, 0, 0, 0, time.UTC)

	for _, tc := range []struct {
		name           string
		away           bool
		wantProduction int
		wantWood       int
	}{
		{name: "home", away: false, wantProduction: 53, wantWood: 53},
		{name: "away", away: true, wantProduction: 50, wantWood: 50},
	} {
		t.Run(tc.name, func(t *testing.T) {
			state := newPlayerState("player_neizheng_"+tc.name, "内政测试", "wei", "xunyu", settledAt)
			for resourceType := range state.Resources.Items {
				state.Resources.Items[resourceType] = 0
				state.Resources.Capacity[resourceType] = 100000
			}
			if tc.away {
				state.GeneralAssignments = append(state.GeneralAssignments, GeneralAssignment{
					ID: "march_neizheng", GeneralID: "xunyu", Slot: "march", Status: "marching",
				})
			}

			next, changed := settleResources(state, settledAt.Add(time.Hour))
			if !changed {
				t.Fatal("expected one-hour resource settlement to change state")
			}
			if next.ResourceProduction["wood"] != tc.wantProduction || next.Resources.Items["wood"] != tc.wantWood {
				t.Fatalf("expected %s production/items %d/%d, got %d/%d", tc.name, tc.wantProduction, tc.wantWood, next.ResourceProduction["wood"], next.Resources.Items["wood"])
			}
		})
	}
}

// TestPvpGeneralReturnSettlesNeizhengAsAwayBeforeRelease 验证 PVP 归还武将前先按离城状态结算，避免归城后追补内政加成。
func TestPvpGeneralReturnSettlesNeizhengAsAwayBeforeRelease(t *testing.T) {
	setTestFactionsAndGenerals(t, FactionsConfig{
		"wei": {Name: "魏国", Generals: []GeneralInfo{{ID: "xunyu", Name: "荀彧"}}},
		"shu": {Name: "蜀国", Generals: []GeneralInfo{{ID: "liubei", Name: "刘备"}}},
	}, GeneralsConfig{
		Enabled: true,
		Heroes: map[string]GeneralHeroConfig{
			"xunyu": {
				ID: "xunyu", Name: "荀彧", Faction: "wei", Enabled: true,
				BonusTrait: GeneralTraitConfig{
					TraitID: "neizheng_jingying", TraitType: general.TraitTypeBonus, Enabled: true,
					Scope: "self_city", Params: map[string]float64{"productionBonusRate": 0.05},
				},
			},
			"liubei": {ID: "liubei", Name: "刘备", Faction: "shu", Enabled: true},
		},
	})
	svc, repo, attacker, defender := newPvpTestServiceForGenerals(t, "wei", "xunyu", "shu", "liubei")
	attacker.Army = []ArmyUnit{{UnitType: "weiInfantry", Amount: 100}}
	defender.Army = []ArmyUnit{{UnitType: "shuInfantry", Amount: 100}}
	repo.players[attacker.Player.ID] = attacker
	repo.players[defender.Player.ID] = defender
	started, err := svc.StartPvpAttack(PvpAttackRequest{
		PlayerID: attacker.Player.ID, TargetPlayerID: defender.Player.ID,
		Troops: map[string]int{"weiInfantry": 40}, GeneralIDs: []string{"xunyu"},
	})
	if err != nil {
		t.Fatalf("StartPvpAttack failed: %v", err)
	}

	awayState, err := repo.GetState(attacker.Player.ID)
	if err != nil {
		t.Fatalf("GetState away failed: %v", err)
	}
	for resourceType := range awayState.Resources.Items {
		awayState.Resources.Items[resourceType] = 0
	}
	awayState.ResourceSettledAt = time.Now().UTC().Add(-time.Hour).Format(resourceDateLayout)
	repo.players[attacker.Player.ID] = awayState
	if _, err := svc.AdminCancelPvpMarch(started.March.ID); err != nil {
		t.Fatalf("AdminCancelPvpMarch failed: %v", err)
	}

	returned, err := repo.GetState(attacker.Player.ID)
	if err != nil {
		t.Fatalf("GetState returned failed: %v", err)
	}
	if returned.Resources.Items["wood"] != 50 || returned.ResourceProduction["wood"] != 53 {
		t.Fatalf("expected away interval to add base 50 then current home production to recover to 53, items=%+v production=%+v", returned.Resources.Items, returned.ResourceProduction)
	}
	if !generalAvailableAtHome(returned.GeneralAssignments, "xunyu") {
		t.Fatalf("expected Xun Yu released back home after cancellation, assignments=%+v", returned.GeneralAssignments)
	}
}

// TestReinforcementGeneralReturnSettlesNeizhengAsAwayBeforeRelease 验证增援召回也会先结清离城产量，再恢复归城后的当前加成。
func TestReinforcementGeneralReturnSettlesNeizhengAsAwayBeforeRelease(t *testing.T) {
	setTestFactionsAndGenerals(t, FactionsConfig{
		"wei": {Name: "魏国", Generals: []GeneralInfo{{ID: "xunyu", Name: "荀彧"}}},
		"shu": {Name: "蜀国", Generals: []GeneralInfo{{ID: "liubei", Name: "刘备"}}},
	}, GeneralsConfig{
		Enabled: true,
		Heroes: map[string]GeneralHeroConfig{
			"xunyu": {
				ID: "xunyu", Name: "荀彧", Faction: "wei", Enabled: true,
				BonusTrait: GeneralTraitConfig{
					TraitID: "neizheng_jingying", TraitType: general.TraitTypeBonus, Enabled: true,
					Scope: "self_city", Params: map[string]float64{"productionBonusRate": 0.05},
				},
			},
			"liubei": {ID: "liubei", Name: "刘备", Faction: "shu", Enabled: true},
		},
	})
	svc, repo, from, to := newReinforcementTestService(t)
	from = newPlayerState(from.Player.ID, from.Player.Nickname, "wei", "xunyu", time.Now().UTC())
	from.Army = []ArmyUnit{{UnitType: "weiInfantry", Amount: 100}}
	repo.players[from.Player.ID] = from
	sent, err := svc.SendReinforcement(SendReinforcementRequest{
		FromPlayerID: from.Player.ID, TargetPlayerID: to.Player.ID,
		Troops: map[string]int{"weiInfantry": 40}, GeneralIDs: []string{"xunyu"},
	})
	if err != nil {
		t.Fatalf("SendReinforcement failed: %v", err)
	}
	if _, err := svc.RecallReinforcement(from.Player.ID, sent.Reinforcement.ID); err != nil {
		t.Fatalf("RecallReinforcement failed: %v", err)
	}

	awayState, err := repo.GetState(from.Player.ID)
	if err != nil {
		t.Fatalf("GetState away failed: %v", err)
	}
	for resourceType := range awayState.Resources.Items {
		awayState.Resources.Items[resourceType] = 0
	}
	awayState.ResourceSettledAt = time.Now().UTC().Add(-time.Hour).Format(resourceDateLayout)
	repo.players[from.Player.ID] = awayState
	forceReinforcementDue(t, repo, sent.Reinforcement.ID, false)
	if _, err := svc.CompleteReinforcementReturn(sent.Reinforcement.ID); err != nil {
		t.Fatalf("CompleteReinforcementReturn failed: %v", err)
	}

	returned, err := repo.GetState(from.Player.ID)
	if err != nil {
		t.Fatalf("GetState returned failed: %v", err)
	}
	if returned.Resources.Items["wood"] != 50 || returned.ResourceProduction["wood"] != 53 {
		t.Fatalf("expected reinforcement away interval to add 50 then current home production to recover to 53, items=%+v production=%+v", returned.Resources.Items, returned.ResourceProduction)
	}
	if !generalAvailableAtHome(returned.GeneralAssignments, "xunyu") {
		t.Fatalf("expected Xun Yu released home after reinforcement return, assignments=%+v", returned.GeneralAssignments)
	}
}

// TestPvpCaoCaoGuardProductionStopsWhileAwayAndResumesAfterReturn 验证曹操出征取消前后只结算真实留城时段。
func TestPvpCaoCaoGuardProductionStopsWhileAwayAndResumesAfterReturn(t *testing.T) {
	setRealCaoCaoGuardConfig(t)
	svc, repo, attacker, defender := newPvpTestServiceForGenerals(t, "wei", "caocao", "shu", "liubei")
	attacker.Army = []ArmyUnit{{UnitType: "weiInfantry", Amount: 100}}
	attacker.ResourceSettledAt = time.Now().UTC().Add(-24 * time.Hour).Format(resourceDateLayout)
	defender.Army = []ArmyUnit{{UnitType: "shuInfantry", Amount: 100}}
	repo.players[attacker.Player.ID] = attacker
	repo.players[defender.Player.ID] = defender

	started, err := svc.StartPvpAttack(PvpAttackRequest{
		PlayerID: attacker.Player.ID, TargetPlayerID: defender.Player.ID,
		Troops: map[string]int{"weiInfantry": 40}, GeneralIDs: []string{"caocao"},
	})
	if err != nil {
		t.Fatalf("StartPvpAttack failed: %v", err)
	}
	away, err := repo.GetState(attacker.Player.ID)
	if err != nil {
		t.Fatalf("GetState away failed: %v", err)
	}
	if guards := armySliceToMap(away.Army)["huWei"]; guards != 3000 {
		t.Fatalf("expected departure to settle 3000 guards before Cao Cao leaves, got %d", guards)
	}
	away.ResourceSettledAt = time.Now().UTC().Add(-24 * time.Hour).Format(resourceDateLayout)
	repo.players[attacker.Player.ID] = away

	if _, err := svc.AdminCancelPvpMarch(started.March.ID); err != nil {
		t.Fatalf("AdminCancelPvpMarch failed: %v", err)
	}
	returned, err := repo.GetState(attacker.Player.ID)
	if err != nil {
		t.Fatalf("GetState returned failed: %v", err)
	}
	if guards := armySliceToMap(returned.Army)["huWei"]; guards != 3000 {
		t.Fatalf("expected no guard production during PVP absence, got %d", guards)
	}
	if !generalAvailableAtHome(returned.GeneralAssignments, "caocao") {
		t.Fatalf("expected Cao Cao available after cancellation, assignments=%+v", returned.GeneralAssignments)
	}
	returned.ResourceSettledAt = time.Now().UTC().Add(-24 * time.Hour).Format(resourceDateLayout)
	repo.players[attacker.Player.ID] = returned
	view, err := svc.GetMilitaryView(attacker.Player.ID)
	if err != nil {
		t.Fatalf("GetMilitaryView after return failed: %v", err)
	}
	if guards := armySliceToMap(view.Army)["huWei"]; guards != 6000 {
		t.Fatalf("expected post-return interval to resume guard production at 6000 total, got %d", guards)
	}
	assertNoBattleReportsForTraitProcess(t, repo, attacker.Player.ID, "PVP guard production")
}

// TestReinforcementCaoCaoGuardProductionStopsWhileAwayAndResumesAfterReturn 验证曹操增援召回前后只结算真实留城时段。
func TestReinforcementCaoCaoGuardProductionStopsWhileAwayAndResumesAfterReturn(t *testing.T) {
	setRealCaoCaoGuardConfig(t)
	svc, repo, from, to := newReinforcementTestService(t)
	from.Army = []ArmyUnit{{UnitType: "weiInfantry", Amount: 100}}
	from.ResourceSettledAt = time.Now().UTC().Add(-24 * time.Hour).Format(resourceDateLayout)
	repo.players[from.Player.ID] = from

	sent, err := svc.SendReinforcement(SendReinforcementRequest{
		FromPlayerID: from.Player.ID, TargetPlayerID: to.Player.ID,
		Troops: map[string]int{"weiInfantry": 40}, GeneralIDs: []string{"caocao"},
	})
	if err != nil {
		t.Fatalf("SendReinforcement failed: %v", err)
	}
	away, err := repo.GetState(from.Player.ID)
	if err != nil {
		t.Fatalf("GetState away failed: %v", err)
	}
	if guards := armySliceToMap(away.Army)["huWei"]; guards != 3000 {
		t.Fatalf("expected reinforcement departure to settle 3000 guards, got %d", guards)
	}
	if _, err := svc.RecallReinforcement(from.Player.ID, sent.Reinforcement.ID); err != nil {
		t.Fatalf("RecallReinforcement failed: %v", err)
	}
	away.ResourceSettledAt = time.Now().UTC().Add(-24 * time.Hour).Format(resourceDateLayout)
	repo.players[from.Player.ID] = away
	forceReinforcementDue(t, repo, sent.Reinforcement.ID, false)
	if _, err := svc.CompleteReinforcementReturn(sent.Reinforcement.ID); err != nil {
		t.Fatalf("CompleteReinforcementReturn failed: %v", err)
	}

	returned, err := repo.GetState(from.Player.ID)
	if err != nil {
		t.Fatalf("GetState returned failed: %v", err)
	}
	if guards := armySliceToMap(returned.Army)["huWei"]; guards != 3000 {
		t.Fatalf("expected no guard production during reinforcement absence, got %d", guards)
	}
	if !generalAvailableAtHome(returned.GeneralAssignments, "caocao") {
		t.Fatalf("expected Cao Cao available after reinforcement return, assignments=%+v", returned.GeneralAssignments)
	}
	returned.ResourceSettledAt = time.Now().UTC().Add(-24 * time.Hour).Format(resourceDateLayout)
	repo.players[from.Player.ID] = returned
	view, err := svc.GetMilitaryView(from.Player.ID)
	if err != nil {
		t.Fatalf("GetMilitaryView after reinforcement return failed: %v", err)
	}
	if guards := armySliceToMap(view.Army)["huWei"]; guards != 6000 {
		t.Fatalf("expected post-return interval to resume guard production at 6000 total, got %d", guards)
	}
	assertNoBattleReportsForTraitProcess(t, repo, from.Player.ID, "reinforcement guard production")
}

// assertNoBattleReportsForTraitProcess 验证非战斗特性流程不会伪造战斗报告。
func assertNoBattleReportsForTraitProcess(t *testing.T, repo *MemoryRepository, playerID string, process string) {
	t.Helper()
	reports, total, err := repo.ListReports(playerID, 10, 0)
	if err != nil || total != 0 || len(reports) != 0 {
		t.Fatalf("expected %s to create no battle reports, reports=%+v total=%d err=%v", process, reports, total, err)
	}
}

// TestMarchTraitsStackIntoFinalDuration 验证吕蒙两项行军特性逐次进入应用层最终时长。
func TestMarchTraitsStackIntoFinalDuration(t *testing.T) {
	setTestFactionsAndGenerals(t, FactionsConfig{
		"wu": {Name: "吴国", Generals: []GeneralInfo{{ID: "lvmeng", Name: "吕蒙"}}},
	}, GeneralsConfig{
		Enabled: true,
		Heroes: map[string]GeneralHeroConfig{
			"lvmeng": {
				ID: "lvmeng", Name: "吕蒙", Faction: "wu", Enabled: true,
				SpecialTrait: GeneralTraitConfig{TraitID: "baiyi_dujiang", TraitType: general.TraitTypeSpecial, Enabled: true, Scope: "self_army", Params: map[string]float64{"speedBonusRate": 0.2, "minMarchSeconds": 60, "triggerChance": 1}},
				BonusTrait:   GeneralTraitConfig{TraitID: "baiyi_jixing", TraitType: general.TraitTypeBonus, Enabled: true, Scope: "self_army", Params: map[string]float64{"speedBonusRate": 0.2, "minMarchSeconds": 60, "triggerChance": 1}},
			},
		},
	})
	state := newPlayerState("player_march_traits", "March", "wu", "lvmeng", time.Now())
	if got := dispatchMarchCreateTraits(1000, "attack", &state, []string{"lvmeng"}); got != 695 {
		t.Fatalf("expected sequential march duration ceil(ceil(1000/1.2)/1.2)=695, got %d", got)
	}
}

// TestRecruitCostTraitsUseOnlyHomeGeneral 验证两项正式征兵减耗特性留城生效、离城失效。
func TestRecruitCostTraitsUseOnlyHomeGeneral(t *testing.T) {
	for _, traitCase := range []struct {
		generalID   string
		generalName string
		traitID     string
		rate        float64
		wantHome    int
	}{
		{generalID: "guojia", generalName: "郭嘉", traitID: "shengui_zhicai", rate: 0.5, wantHome: 1100},
		{generalID: "xunyu", generalName: "荀彧", traitID: "wangzuo_zhicai", rate: 0.05, wantHome: 1010},
	} {
		t.Run(traitCase.generalName+traitCase.traitID, func(t *testing.T) {
			setTestCombatUnitsConfig(t)
			unitsMu.Lock()
			unit := activeUnits["wei"]["weiInfantry"]
			unit.Cost = map[string]int{"wood": 100}
			unit.TrainSeconds = 60
			activeUnits["wei"]["weiInfantry"] = unit
			unitsMu.Unlock()
			setTestFactionsAndGenerals(t, FactionsConfig{
				"wei": {Name: "魏国", Generals: []GeneralInfo{{ID: traitCase.generalID, Name: traitCase.generalName}}},
			}, GeneralsConfig{Enabled: true, Heroes: map[string]GeneralHeroConfig{
				traitCase.generalID: {
					ID: traitCase.generalID, Name: traitCase.generalName, Faction: "wei", Enabled: true,
					SpecialTrait: GeneralTraitConfig{TraitID: traitCase.traitID, TraitType: general.TraitTypeSpecial, Enabled: true, Scope: "self_city", Params: map[string]float64{"resourceCostReduction": traitCase.rate, "triggerChance": 1}},
				},
			}})
			svc := NewService()
			repo := svc.repo.(*MemoryRepository)
			now := time.Now().UTC()
			for _, stateCase := range []struct {
				id       string
				away     bool
				wantWood int
			}{
				{id: "home", away: false, wantWood: traitCase.wantHome},
				{id: "away", away: true, wantWood: 1000},
			} {
				id := traitCase.generalID + "_" + stateCase.id
				account := Account{ID: "account_recruit_" + id, Username: "recruit_" + id, PasswordHash: "hash", CreatedAt: now}
				if err := repo.CreateAccount(account); err != nil {
					t.Fatalf("CreateAccount %s failed: %v", id, err)
				}
				state := newPlayerState("player_recruit_"+id, "Recruit", "wei", traitCase.generalID, now)
				state.Resources.Items["wood"] = 1200
				if stateCase.away {
					state.GeneralAssignments = append(state.GeneralAssignments, GeneralAssignment{ID: "march_busy", GeneralID: traitCase.generalID, Slot: "march", Status: "marching"})
				}
				if err := repo.CreatePlayer(account.ID, state, now); err != nil {
					t.Fatalf("CreatePlayer %s failed: %v", id, err)
				}
				next, err := svc.Recruit(state.Player.ID, "weiInfantry", 2)
				if err != nil {
					t.Fatalf("Recruit %s failed: %v", id, err)
				}
				if next.Resources.Items["wood"] != stateCase.wantWood {
					t.Fatalf("expected %s wood %d after recruit, got %d", id, stateCase.wantWood, next.Resources.Items["wood"])
				}
				reports, total, reportErr := repo.ListReports(state.Player.ID, 10, 0)
				if reportErr != nil || total != 0 || len(reports) != 0 {
					t.Fatalf("expected recruit trait not to create battle report, reports=%+v total=%d err=%v", reports, total, reportErr)
				}
			}
		})
	}
}
