// 本文件验证随机将领特性未命中时，真实 PVP 状态、经验和双方战报保持基础结算结果。
package game

import (
	"math"
	"testing"
	"time"

	"hero3/internal/core/general"
)

// TestPvpRandomTraitMissesLeaveNoBattleOrReportEffects 验证战前伤亡和战后返兵未命中时都不产生隐藏副作用。
func TestPvpRandomTraitMissesLeaveNoBattleOrReportEffects(t *testing.T) {
	t.Run("疑兵偷袭未命中", func(t *testing.T) {
		setTestFactionsAndGenerals(t, FactionsConfig{
			"wei": {Name: "魏国", Generals: []GeneralInfo{{ID: "simayi", Name: "司马懿"}}},
			"shu": {Name: "蜀国", Generals: []GeneralInfo{{ID: "liubei", Name: "刘备"}}},
		}, GeneralsConfig{Enabled: true, Heroes: map[string]GeneralHeroConfig{
			"simayi": {
				ID: "simayi", Name: "司马懿", Faction: "wei", Enabled: true,
				SpecialTrait: GeneralTraitConfig{
					TraitID: "yibing_touxi", TraitType: general.TraitTypeSpecial, Enabled: true, Scope: "enemy_army",
					Params: map[string]float64{"triggerChance": 0, "effectRate": 0.35, "maxAffectedRate": 0.35},
				},
			},
			"liubei": {ID: "liubei", Name: "刘备", Faction: "shu", Enabled: true},
		}})
		svc, repo, attacker, defender := newPvpTestServiceForGenerals(t, "wei", "simayi", "shu", "liubei")
		attacker.Army = []ArmyUnit{{UnitType: "weiInfantry", Amount: 1000}}
		defender.Army = []ArmyUnit{{UnitType: "shuInfantry", Amount: 1000}}
		defender.Buildings = nil
		repo.players[attacker.Player.ID] = attacker
		repo.players[defender.Player.ID] = defender

		started, err := svc.StartPvpAttack(PvpAttackRequest{
			PlayerID: attacker.Player.ID, TargetPlayerID: defender.Player.ID, MarchMode: PvpMarchTypePlunder,
			Troops: map[string]int{"weiInfantry": 1000}, GeneralIDs: []string{"simayi"},
		})
		if err != nil {
			t.Fatalf("StartPvpAttack failed: %v", err)
		}
		forcePvpMarchDue(t, repo, started.March.ID)
		battle, err := svc.ResolvePvpMarch(started.March.ID)
		if err != nil {
			t.Fatalf("ResolvePvpMarch failed: %v", err)
		}
		if battle.Result["attackerPower"] != float64(10000) || battle.Result["defensePower"] != float64(10000) {
			t.Fatalf("expected missed pre-battle trait to keep 10000/10000 power, result=%+v", battle.Result)
		}
		attackerLosses := pvpTestLossesFromBattle(t, battle, "attacker")
		defenderLosses := pvpTestLossesFromBattle(t, battle, "defender")
		if attackerLosses["weiInfantry"] != 500 || defenderLosses["shuInfantry"] != 500 {
			t.Fatalf("expected baseline 500/500 losses, attacker=%+v defender=%+v", attackerLosses, defenderLosses)
		}
		assertRandomMissPvpReports(t, repo, battle, attacker.Player.ID, defender.Player.ID, "simayi", "yibing_touxi")

		storedAttacker, err := repo.GetState(attacker.Player.ID)
		if err != nil || pvpTestGeneralExp(storedAttacker, "simayi") != 500 {
			t.Fatalf("expected Sima Yi to gain only baseline 500 exp, state=%+v err=%v", storedAttacker.Generals, err)
		}
		storedDefender, err := repo.GetState(defender.Player.ID)
		if err != nil || armySliceToMap(storedDefender.Army)["shuInfantry"] != 500 {
			t.Fatalf("expected defender baseline 500 survivors, state=%+v err=%v", storedDefender.Army, err)
		}
		returning, err := repo.GetPvpMarch(started.March.ID)
		if err != nil || returning.AttackTroops["weiInfantry"] != 500 {
			t.Fatalf("expected baseline 500 returning attackers, march=%+v err=%v", returning, err)
		}
	})

	t.Run("护主死战未命中", func(t *testing.T) {
		setTestFactionsAndGenerals(t, FactionsConfig{
			"wei": {Name: "魏国", Generals: []GeneralInfo{{ID: "dianwei", Name: "典韦"}}},
			"shu": {Name: "蜀国", Generals: []GeneralInfo{{ID: "liubei", Name: "刘备"}}},
		}, GeneralsConfig{Enabled: true, Heroes: map[string]GeneralHeroConfig{
			"dianwei": {
				ID: "dianwei", Name: "典韦", Faction: "wei", Enabled: true,
				SpecialTrait: GeneralTraitConfig{
					TraitID: "huzhu_sizhan", TraitType: general.TraitTypeSpecial, Enabled: true, Scope: "self_army", RequiredOutcome: "loss",
					Params: map[string]float64{"triggerChance": 0, "lossReductionRate": 0.15, "maxReturnCount": 10000},
				},
			},
			"liubei": {ID: "liubei", Name: "刘备", Faction: "shu", Enabled: true},
		}})
		svc, repo, attacker, defender := newPvpTestServiceForGenerals(t, "wei", "dianwei", "shu", "liubei")
		attacker.Army = []ArmyUnit{{UnitType: "weiInfantry", Amount: 100}}
		defender.Army = []ArmyUnit{{UnitType: "shuInfantry", Amount: 1000}}
		defender.Buildings = nil
		repo.players[attacker.Player.ID] = attacker
		repo.players[defender.Player.ID] = defender

		started, err := svc.StartPvpAttack(PvpAttackRequest{
			PlayerID: attacker.Player.ID, TargetPlayerID: defender.Player.ID, MarchMode: PvpMarchTypePlunder,
			Troops: map[string]int{"weiInfantry": 100}, GeneralIDs: []string{"dianwei"},
		})
		if err != nil {
			t.Fatalf("StartPvpAttack failed: %v", err)
		}
		forcePvpMarchDue(t, repo, started.March.ID)
		battle, err := svc.ResolvePvpMarch(started.March.ID)
		if err != nil {
			t.Fatalf("ResolvePvpMarch failed: %v", err)
		}
		if battle.Result["attackerPower"] != float64(1000) || battle.Result["defensePower"] != float64(10000) {
			t.Fatalf("expected missed recovery trait to keep 1000/10000 power, result=%+v", battle.Result)
		}
		attackerLosses := pvpTestLossesFromBattle(t, battle, "attacker")
		defenderLosses := pvpTestLossesFromBattle(t, battle, "defender")
		if attackerLosses["weiInfantry"] != 96 || defenderLosses["shuInfantry"] != 36 {
			t.Fatalf("expected baseline 96/36 losses, attacker=%+v defender=%+v", attackerLosses, defenderLosses)
		}
		assertRandomMissPvpReports(t, repo, battle, attacker.Player.ID, defender.Player.ID, "dianwei", "huzhu_sizhan")

		storedAttacker, err := repo.GetState(attacker.Player.ID)
		if err != nil || pvpTestGeneralExp(storedAttacker, "dianwei") != 36 {
			t.Fatalf("expected Dian Wei to gain only baseline 36 exp, state=%+v err=%v", storedAttacker.Generals, err)
		}
		storedDefender, err := repo.GetState(defender.Player.ID)
		if err != nil || armySliceToMap(storedDefender.Army)["shuInfantry"] != 964 {
			t.Fatalf("expected defender baseline 964 survivors, state=%+v err=%v", storedDefender.Army, err)
		}
		returning, err := repo.GetPvpMarch(started.March.ID)
		if err != nil || returning.AttackTroops["weiInfantry"] != 4 {
			t.Fatalf("expected only 4 baseline survivors without trait return, march=%+v err=%v", returning, err)
		}
	})

	t.Run("江东固守防守未命中", func(t *testing.T) {
		setTestFactionsAndGenerals(t, FactionsConfig{
			"wei": {Name: "魏国", Generals: []GeneralInfo{{ID: "caocao", Name: "曹操"}}},
			"wu":  {Name: "吴国", Generals: []GeneralInfo{{ID: "sunquan", Name: "孙权"}}},
		}, GeneralsConfig{Enabled: true, Heroes: map[string]GeneralHeroConfig{
			"caocao": {ID: "caocao", Name: "曹操", Faction: "wei", Enabled: true},
			"sunquan": {
				ID: "sunquan", Name: "孙权", Faction: "wu", Enabled: true,
				SpecialTrait: GeneralTraitConfig{
					TraitID: "jiangdong_haoling", TraitType: general.TraitTypeSpecial, Enabled: false,
					Scope: "enemy_plunder", AllowedSides: []string{"defender"}, AllowedScenes: []string{"plunder"}, RequiredOutcome: "loss",
					Params: map[string]float64{"plunderBonusRate": -0.2},
				},
				BonusTrait: GeneralTraitConfig{
					TraitID: "jiangdong_gushou", TraitType: general.TraitTypeBonus, Enabled: true,
					Scope: "self_army", AllowedSides: []string{"defender", "reinforcement"},
					Params: map[string]float64{"triggerChance": 0, "defenseBonusRate": 0.5},
				},
			},
		}})
		svc, repo, attacker, defender := newPvpTestServiceForGenerals(t, "wei", "caocao", "wu", "sunquan")
		attacker.Army = []ArmyUnit{{UnitType: "weiInfantry", Amount: 1000}}
		defender.Army = []ArmyUnit{{UnitType: "wuInfantry", Amount: 1000}}
		defender.Buildings = nil
		repo.players[attacker.Player.ID] = attacker
		repo.players[defender.Player.ID] = defender

		started, err := svc.StartPvpAttack(PvpAttackRequest{
			PlayerID: attacker.Player.ID, TargetPlayerID: defender.Player.ID, MarchMode: PvpMarchTypePlunder,
			Troops: map[string]int{"weiInfantry": 1000}, GeneralIDs: []string{"caocao"},
		})
		if err != nil {
			t.Fatalf("StartPvpAttack failed: %v", err)
		}
		forcePvpMarchDue(t, repo, started.March.ID)
		battle, err := svc.ResolvePvpMarch(started.March.ID)
		if err != nil {
			t.Fatalf("ResolvePvpMarch failed: %v", err)
		}
		if battle.Result["attackerPower"] != float64(10000) || battle.Result["defensePower"] != float64(10000) {
			t.Fatalf("expected missed defense trait to keep 10000/10000 power, result=%+v", battle.Result)
		}
		attackerLosses := pvpTestLossesFromBattle(t, battle, "attacker")
		defenderLosses := pvpTestLossesFromBattle(t, battle, "defender")
		if attackerLosses["weiInfantry"] != 500 || defenderLosses["wuInfantry"] != 500 {
			t.Fatalf("expected baseline 500/500 losses, attacker=%+v defender=%+v", attackerLosses, defenderLosses)
		}
		assertRandomMissDefenderPvpReports(t, repo, battle, attacker.Player.ID, defender.Player.ID, "sunquan", "jiangdong_gushou")

		storedAttacker, err := repo.GetState(attacker.Player.ID)
		if err != nil || pvpTestGeneralExp(storedAttacker, "caocao") != 500 {
			t.Fatalf("expected Cao Cao to gain baseline 500 exp, state=%+v err=%v", storedAttacker.Generals, err)
		}
		storedDefender, err := repo.GetState(defender.Player.ID)
		if err != nil || armySliceToMap(storedDefender.Army)["wuInfantry"] != 500 || pvpTestGeneralExp(storedDefender, "sunquan") != 500 {
			t.Fatalf("expected Sun Quan baseline 500 survivors and 500 exp, state=%+v err=%v", storedDefender, err)
		}
		returning, err := repo.GetPvpMarch(started.March.ID)
		if err != nil || returning.AttackTroops["weiInfantry"] != 500 {
			t.Fatalf("expected baseline 500 returning attackers, march=%+v err=%v", returning, err)
		}
	})

	t.Run("奇兵绕后主动进攻未命中", func(t *testing.T) {
		setTestFactionsAndGenerals(t, FactionsConfig{
			"shu": {Name: "蜀国", Generals: []GeneralInfo{{ID: "weiyan", Name: "魏延"}}},
			"wei": {Name: "魏国", Generals: []GeneralInfo{{ID: "caocao", Name: "曹操"}}},
		}, GeneralsConfig{Enabled: true, Heroes: map[string]GeneralHeroConfig{
			"weiyan": {
				ID: "weiyan", Name: "魏延", Faction: "shu", Enabled: true,
				SpecialTrait: GeneralTraitConfig{
					TraitID: "qibing_raohou", TraitType: general.TraitTypeSpecial, Enabled: true,
					Scope: "enemy_army", AllowedSides: []string{"attacker"},
					Params: map[string]float64{"triggerChance": 0, "enemyDefenseReductionRate": 0.2},
				},
				BonusTrait: GeneralTraitConfig{
					TraitID: "gushou_hanzhong", TraitType: general.TraitTypeBonus, Enabled: true,
					Scope: "self_army", AllowedSides: []string{"defender", "reinforcement"},
					Params: map[string]float64{"triggerChance": 1, "generalDefenseFlat": 20},
				},
			},
			"caocao": {ID: "caocao", Name: "曹操", Faction: "wei", Enabled: true},
		}})
		svc, repo, attacker, defender := newPvpTestServiceForGenerals(t, "shu", "weiyan", "wei", "caocao")
		attacker.Army = []ArmyUnit{{UnitType: "shuInfantry", Amount: 1000}}
		defender.Army = []ArmyUnit{{UnitType: "weiInfantry", Amount: 1000}}
		defender.Buildings = nil
		repo.players[attacker.Player.ID] = attacker
		repo.players[defender.Player.ID] = defender

		started, err := svc.StartPvpAttack(PvpAttackRequest{
			PlayerID: attacker.Player.ID, TargetPlayerID: defender.Player.ID, MarchMode: PvpMarchTypePlunder,
			Troops: map[string]int{"shuInfantry": 1000}, GeneralIDs: []string{"weiyan"},
		})
		if err != nil {
			t.Fatalf("StartPvpAttack failed: %v", err)
		}
		forcePvpMarchDue(t, repo, started.March.ID)
		battle, err := svc.ResolvePvpMarch(started.March.ID)
		if err != nil {
			t.Fatalf("ResolvePvpMarch failed: %v", err)
		}
		if battle.Result["attackerPower"] != float64(10000) || battle.Result["defensePower"] != float64(10000) {
			t.Fatalf("expected missed Qibing and direction-inactive Gushou to keep 10000/10000 power, result=%+v", battle.Result)
		}
		attackerLosses := pvpTestLossesFromBattle(t, battle, "attacker")
		defenderLosses := pvpTestLossesFromBattle(t, battle, "defender")
		if attackerLosses["shuInfantry"] != 500 || defenderLosses["weiInfantry"] != 500 {
			t.Fatalf("expected baseline 500/500 losses without either Wei Yan trait, attacker=%+v defender=%+v", attackerLosses, defenderLosses)
		}

		attackerReports, _, err := repo.ListReports(attacker.Player.ID, 10, 0)
		if err != nil || len(attackerReports) != 1 || attackerReports[0].ID != battle.AttackerReportID {
			t.Fatalf("expected one attacker report, reports=%+v err=%v", attackerReports, err)
		}
		defenderReports, _, err := repo.ListReports(defender.Player.ID, 10, 0)
		if err != nil || len(defenderReports) != 1 || defenderReports[0].ID != battle.DefenderReportID {
			t.Fatalf("expected one defender report, reports=%+v err=%v", defenderReports, err)
		}
		for _, report := range []BattleReport{attackerReports[0], defenderReports[0]} {
			if len(report.TraitTriggered) != 0 || len(report.TraitOutcomes) != 0 || report.Detail == nil || report.Detail.SecondarySide == nil || len(report.Detail.Traits) != 0 {
				t.Fatalf("expected no Wei Yan trait in either timeline, report=%+v", report)
			}
			if len(report.PvpAttackerGenerals) != 1 || !pvpSnapshotHasTrait(report.PvpAttackerGenerals[0], "qibing_raohou") || !pvpSnapshotHasTrait(report.PvpAttackerGenerals[0], "gushou_hanzhong") {
				t.Fatalf("expected Wei Yan snapshot to retain both owned traits, report=%s generals=%+v", report.ID, report.PvpAttackerGenerals)
			}
			if !standardDetailGeneralHasTrait(report.Detail, "qibing_raohou") || !standardDetailGeneralHasTrait(report.Detail, "gushou_hanzhong") {
				t.Fatalf("expected standard Wei Yan snapshot to retain both owned traits, report=%s detail=%+v", report.ID, report.Detail)
			}
			attackerUnit := attackDefenseCrossReportUnit(t, report.Detail.PrimarySide, "shuInfantry")
			defenderUnit := attackDefenseCrossReportUnit(t, *report.Detail.SecondarySide, "weiInfantry")
			if attackerUnit.AmountBefore != 1000 || attackerUnit.Lost != 500 || attackerUnit.Survived != 500 || defenderUnit.AmountBefore != 1000 || defenderUnit.Lost != 500 || defenderUnit.Survived != 500 {
				t.Fatalf("expected standard rows 1000/500/500 on both sides, report=%s attacker=%+v defender=%+v", report.ID, attackerUnit, defenderUnit)
			}
		}

		storedMarch, marchErr := repo.GetPvpMarch(started.March.ID)
		storedAttacker, attackerErr := repo.GetState(attacker.Player.ID)
		storedDefender, defenderErr := repo.GetState(defender.Player.ID)
		if marchErr != nil || attackerErr != nil || defenderErr != nil {
			t.Fatalf("expected stored battle state, march=%v attacker=%v defender=%v", marchErr, attackerErr, defenderErr)
		}
		if storedMarch.AttackTroops["shuInfantry"] != 500 || armySliceToMap(storedDefender.Army)["weiInfantry"] != 500 {
			t.Fatalf("expected authoritative survivors 500/500, march=%+v defender=%+v", storedMarch, storedDefender.Army)
		}
		if pvpTestGeneralExp(storedAttacker, "weiyan") != 500 || attackerReports[0].GeneralExpGained != 500 || attackerReports[0].Detail.Rewards.GeneralExp != 500 || pvpTestGeneralExp(storedDefender, "caocao") != 500 || defenderReports[0].GeneralExpGained != 500 || defenderReports[0].Detail.Rewards.GeneralExp != 500 {
			t.Fatalf("expected baseline exp 500/500, attacker=%d/%d/%d defender=%d/%d/%d", pvpTestGeneralExp(storedAttacker, "weiyan"), attackerReports[0].GeneralExpGained, attackerReports[0].Detail.Rewards.GeneralExp, pvpTestGeneralExp(storedDefender, "caocao"), defenderReports[0].GeneralExpGained, defenderReports[0].Detail.Rewards.GeneralExp)
		}
	})

	t.Run("天神下凡生效但西凉突击未命中", func(t *testing.T) {
		setTestFactionsAndGenerals(t, FactionsConfig{
			"shu": {Name: "蜀国", Generals: []GeneralInfo{{ID: "machao", Name: "马超"}}},
			"wei": {Name: "魏国", Generals: []GeneralInfo{{ID: "caocao", Name: "曹操"}}},
		}, GeneralsConfig{Enabled: true, Heroes: map[string]GeneralHeroConfig{
			"machao": {
				ID: "machao", Name: "马超", Faction: "shu", Enabled: true,
				SpecialTrait: GeneralTraitConfig{
					TraitID: "xiliang_tuji", TraitType: general.TraitTypeSpecial, Enabled: true,
					Scope: "enemy_army", TargetUnitType: "cavalry",
					Params: map[string]float64{"triggerChance": 0, "effectRate": 0.12},
				},
				BonusTrait: GeneralTraitConfig{
					TraitID: "tianshen_xiafan", TraitType: general.TraitTypeBonus, Enabled: true,
					Scope: "self_army", Params: map[string]float64{"forceBonus": 20},
				},
			},
			"caocao": {ID: "caocao", Name: "曹操", Faction: "wei", Enabled: true},
		}})
		svc, repo, attacker, defender := newPvpTestServiceForGenerals(t, "shu", "machao", "wei", "caocao")
		unitsMu.Lock()
		activeUnits["wei"]["weiCavalry"] = UnitConfig{
			Name: "魏骑兵", Category: "cavalry",
			Stats: map[string]int{"attack": 10, "infantryDefense": 10, "cavalryDefense": 8, "carryCapacity": 5, "upkeep": 1},
		}
		unitsMu.Unlock()
		attacker.Army = []ArmyUnit{{UnitType: "shuInfantry", Amount: 1000}}
		defender.Army = []ArmyUnit{{UnitType: "weiCavalry", Amount: 1000}}
		defender.Buildings = nil
		repo.players[attacker.Player.ID] = attacker
		repo.players[defender.Player.ID] = defender

		started, err := svc.StartPvpAttack(PvpAttackRequest{
			PlayerID: attacker.Player.ID, TargetPlayerID: defender.Player.ID, MarchMode: PvpMarchTypePlunder,
			Troops: map[string]int{"shuInfantry": 1000}, GeneralIDs: []string{"machao"},
		})
		if err != nil {
			t.Fatalf("StartPvpAttack failed: %v", err)
		}
		forcePvpMarchDue(t, repo, started.March.ID)
		battle, err := svc.ResolvePvpMarch(started.March.ID)
		if err != nil {
			t.Fatalf("ResolvePvpMarch failed: %v", err)
		}
		if battle.Result["attackerPower"] != float64(14000) || battle.Result["defensePower"] != float64(10000) {
			t.Fatalf("expected passive force to remain effective at 14000/10000 when Xiliang misses, result=%+v", battle.Result)
		}
		attackerLosses := pvpTestLossesFromBattle(t, battle, "attacker")
		defenderLosses := pvpTestLossesFromBattle(t, battle, "defender")
		if attackerLosses["shuInfantry"] != 382 || defenderLosses["weiCavalry"] != 617 {
			t.Fatalf("expected passive-adjusted core losses 382/617 without extra cavalry damage, attacker=%+v defender=%+v", attackerLosses, defenderLosses)
		}
		assertRandomMissPvpReports(t, repo, battle, attacker.Player.ID, defender.Player.ID, "machao", "xiliang_tuji")

		attackerReports, _, err := repo.ListReports(attacker.Player.ID, 10, 0)
		if err != nil || len(attackerReports) != 1 {
			t.Fatalf("expected one attacker report, reports=%+v err=%v", attackerReports, err)
		}
		defenderReports, _, err := repo.ListReports(defender.Player.ID, 10, 0)
		if err != nil || len(defenderReports) != 1 {
			t.Fatalf("expected one defender report, reports=%+v err=%v", defenderReports, err)
		}
		for _, report := range []BattleReport{attackerReports[0], defenderReports[0]} {
			snapshot := report.PvpAttackerGenerals[0]
			if snapshot.EffectiveStats["force"]-snapshot.Stats["force"] != 20 || math.Abs(snapshot.Buffs[StatAttackBonus]-0.4) > 1e-9 || !pvpSnapshotHasTrait(snapshot, "tianshen_xiafan") || !pvpSnapshotHasTrait(snapshot, "xiliang_tuji") {
				t.Fatalf("expected Ma Chao snapshot to keep passive 40%% modifier and both owned traits, report=%s snapshot=%+v", report.ID, snapshot)
			}
			if !standardDetailGeneralHasTrait(report.Detail, "tianshen_xiafan") || !standardDetailGeneralHasTrait(report.Detail, "xiliang_tuji") {
				t.Fatalf("expected standard general snapshot to retain both owned traits, report=%s detail=%+v", report.ID, report.Detail)
			}
			if len(report.Detail.PrimarySide.Generals) != 1 || report.Detail.PrimarySide.Generals[0].EffectiveStats["force"]-report.Detail.PrimarySide.Generals[0].Stats["force"] != 20 || math.Abs(report.Detail.PrimarySide.Generals[0].Buffs[StatAttackBonus]-0.4) > 1e-9 {
				t.Fatalf("expected standard report snapshot to retain passive 40%% modifier, report=%s generals=%+v", report.ID, report.Detail.PrimarySide.Generals)
			}
			attackerUnit := attackDefenseCrossReportUnit(t, report.Detail.PrimarySide, "shuInfantry")
			defenderUnit := attackDefenseCrossReportUnit(t, *report.Detail.SecondarySide, "weiCavalry")
			if attackerUnit.AmountBefore != 1000 || attackerUnit.Lost != 382 || attackerUnit.Survived != 618 || defenderUnit.AmountBefore != 1000 || defenderUnit.Lost != 617 || defenderUnit.Survived != 383 {
				t.Fatalf("expected standard rows 1000/382/618 and 1000/617/383, report=%s attacker=%+v defender=%+v", report.ID, attackerUnit, defenderUnit)
			}
		}

		storedMarch, marchErr := repo.GetPvpMarch(started.March.ID)
		storedAttacker, attackerErr := repo.GetState(attacker.Player.ID)
		storedDefender, defenderErr := repo.GetState(defender.Player.ID)
		if marchErr != nil || attackerErr != nil || defenderErr != nil {
			t.Fatalf("expected stored battle state, march=%v attacker=%v defender=%v", marchErr, attackerErr, defenderErr)
		}
		if storedMarch.AttackTroops["shuInfantry"] != 618 || armySliceToMap(storedDefender.Army)["weiCavalry"] != 383 {
			t.Fatalf("expected authoritative survivors 618/383, march=%+v defender=%+v", storedMarch, storedDefender.Army)
		}
		if pvpTestGeneralExp(storedAttacker, "machao") != 617 || attackerReports[0].GeneralExpGained != 617 || pvpTestGeneralExp(storedDefender, "caocao") != 382 || defenderReports[0].GeneralExpGained != 382 {
			t.Fatalf("expected general exp 617/382 from core losses only, stored=%d/%d reports=%d/%d", pvpTestGeneralExp(storedAttacker, "machao"), pvpTestGeneralExp(storedDefender, "caocao"), attackerReports[0].GeneralExpGained, defenderReports[0].GeneralExpGained)
		}
	})

	t.Run("小霸王铁骑生效但小霸王追击未命中", func(t *testing.T) {
		setTestFactionsAndGenerals(t, FactionsConfig{
			"wu":  {Name: "吴国", Generals: []GeneralInfo{{ID: "sunce", Name: "孙策"}}},
			"wei": {Name: "魏国", Generals: []GeneralInfo{{ID: "caocao", Name: "曹操"}}},
		}, GeneralsConfig{Enabled: true, Heroes: map[string]GeneralHeroConfig{
			"sunce": {
				ID: "sunce", Name: "孙策", Faction: "wu", Enabled: true,
				SpecialTrait: GeneralTraitConfig{
					TraitID: "xiaobawang_zhuiji", TraitType: general.TraitTypeSpecial, Enabled: true,
					Scope: "enemy_army", AllowedScenes: []string{"plunder"}, RequiredOutcome: "win",
					Params: map[string]float64{"triggerChance": 0, "effectRate": 0.1},
				},
				BonusTrait: GeneralTraitConfig{
					TraitID: "xiaobawang_tieqi", TraitType: general.TraitTypeBonus, Enabled: true,
					Scope: "self_army", TargetUnitType: "overlordRider", AllowedSides: []string{"attacker"},
					Params: map[string]float64{"unitAttackFlat": 50},
				},
			},
			"caocao": {ID: "caocao", Name: "曹操", Faction: "wei", Enabled: true},
		}})
		svc, repo, attacker, defender := newPvpTestServiceForGenerals(t, "wu", "sunce", "wei", "caocao")
		unitsMu.Lock()
		activeUnits["wu"]["overlordRider"] = UnitConfig{
			Name: "霸王骑", Category: "cavalry",
			Stats: map[string]int{"attack": 28, "infantryDefense": 10, "cavalryDefense": 33, "carryCapacity": 130, "upkeep": 4},
		}
		unitsMu.Unlock()
		attacker.Army = []ArmyUnit{{UnitType: "overlordRider", Amount: 200}}
		attacker.Resources.Items = map[string]int{"wood": 0, "stone": 0, "iron": 0, "food": 0}
		attacker.Resources.Capacity = map[string]int{"wood": 100000, "stone": 100000, "iron": 100000, "food": 100000}
		defender.Army = []ArmyUnit{{UnitType: "weiInfantry", Amount: 1000}}
		defender.Buildings = nil
		defender.Resources.Items = map[string]int{"wood": 10000, "stone": 0, "iron": 0, "food": 0}
		defender.Resources.Capacity = map[string]int{"wood": 0, "stone": 0, "iron": 0, "food": 0}
		nowText := time.Now().UTC().Format(resourceDateLayout)
		attacker.ResourceSettledAt = nowText
		defender.ResourceSettledAt = nowText
		repo.players[attacker.Player.ID] = attacker
		repo.players[defender.Player.ID] = defender

		started, err := svc.StartPvpAttack(PvpAttackRequest{
			PlayerID: attacker.Player.ID, TargetPlayerID: defender.Player.ID, MarchMode: PvpMarchTypePlunder,
			Troops: map[string]int{"overlordRider": 200}, GeneralIDs: []string{"sunce"},
		})
		if err != nil {
			t.Fatalf("StartPvpAttack failed: %v", err)
		}
		forcePvpMarchDue(t, repo, started.March.ID)
		battle, err := svc.ResolvePvpMarch(started.March.ID)
		if err != nil {
			t.Fatalf("ResolvePvpMarch failed: %v", err)
		}
		if battle.Result["attackerPower"] != float64(15600) || battle.Result["defensePower"] != float64(8000) {
			t.Fatalf("expected cavalry bonus to remain effective at 15600/8000 when pursuit misses, result=%+v", battle.Result)
		}
		attackerLosses := pvpTestLossesFromBattle(t, battle, "attacker")
		defenderLosses := pvpTestLossesFromBattle(t, battle, "defender")
		if attackerLosses["overlordRider"] != 55 || defenderLosses["weiInfantry"] != 721 {
			t.Fatalf("expected core losses 55/721 without 100 pursuit losses, attacker=%+v defender=%+v", attackerLosses, defenderLosses)
		}

		attackerReports, _, err := repo.ListReports(attacker.Player.ID, 10, 0)
		if err != nil || len(attackerReports) != 1 {
			t.Fatalf("expected one attacker report, reports=%+v err=%v", attackerReports, err)
		}
		defenderReports, _, err := repo.ListReports(defender.Player.ID, 10, 0)
		if err != nil || len(defenderReports) != 1 {
			t.Fatalf("expected one defender report, reports=%+v err=%v", defenderReports, err)
		}
		for _, report := range []BattleReport{attackerReports[0], defenderReports[0]} {
			if len(report.TraitTriggered) != 1 || report.TraitTriggered[0] != "xiaobawang_tieqi" || len(report.TraitOutcomes) != 1 {
				t.Fatalf("expected only deterministic cavalry bonus in legacy timeline, report=%s triggered=%+v outcomes=%+v", report.ID, report.TraitTriggered, report.TraitOutcomes)
			}
			if _, exists := report.TraitOutcomes["xiaobawang_zhuiji"]; exists {
				t.Fatalf("expected missed pursuit absent from legacy outcomes, report=%s outcomes=%+v", report.ID, report.TraitOutcomes)
			}
			if len(report.PvpAttackerGenerals) != 1 || !pvpSnapshotHasTrait(report.PvpAttackerGenerals[0], "xiaobawang_tieqi") || !pvpSnapshotHasTrait(report.PvpAttackerGenerals[0], "xiaobawang_zhuiji") {
				t.Fatalf("expected Sun Ce snapshot to retain both owned traits, report=%s generals=%+v", report.ID, report.PvpAttackerGenerals)
			}
			if report.Detail == nil || report.Detail.SecondarySide == nil || len(report.Detail.Traits) != 1 || report.Detail.Traits[0].TraitID != "xiaobawang_tieqi" {
				t.Fatalf("expected standard timeline to contain only cavalry bonus, report=%s detail=%+v", report.ID, report.Detail)
			}
			if !standardDetailGeneralHasTrait(report.Detail, "xiaobawang_tieqi") || !standardDetailGeneralHasTrait(report.Detail, "xiaobawang_zhuiji") {
				t.Fatalf("expected standard general snapshot to retain both owned traits, report=%s detail=%+v", report.ID, report.Detail)
			}
			attackerUnit := attackDefenseCrossReportUnit(t, report.Detail.PrimarySide, "overlordRider")
			defenderUnit := attackDefenseCrossReportUnit(t, *report.Detail.SecondarySide, "weiInfantry")
			if attackerUnit.AmountBefore != 200 || attackerUnit.Lost != 55 || attackerUnit.Survived != 145 || defenderUnit.AmountBefore != 1000 || defenderUnit.Lost != 721 || defenderUnit.Survived != 279 {
				t.Fatalf("expected standard rows 200/55/145 and 1000/721/279, report=%s attacker=%+v defender=%+v", report.ID, attackerUnit, defenderUnit)
			}
		}

		storedMarch, marchErr := repo.GetPvpMarch(started.March.ID)
		storedAttacker, attackerErr := repo.GetState(attacker.Player.ID)
		storedDefender, defenderErr := repo.GetState(defender.Player.ID)
		if marchErr != nil || attackerErr != nil || defenderErr != nil {
			t.Fatalf("expected stored battle state, march=%v attacker=%v defender=%v", marchErr, attackerErr, defenderErr)
		}
		if storedMarch.AttackTroops["overlordRider"] != 145 || armySliceToMap(storedDefender.Army)["weiInfantry"] != 279 {
			t.Fatalf("expected authoritative survivors 145/279, march=%+v defender=%+v", storedMarch, storedDefender.Army)
		}
		defenderExp := calculateGeneralBattleExpFromLosses(attacker.Player.Faction, pvpTestUnitLosses(attackerLosses))
		if pvpTestGeneralExp(storedAttacker, "sunce") != 721 || attackerReports[0].GeneralExpGained != 721 || defenderExp != 220 || pvpTestGeneralExp(storedDefender, "caocao") != defenderExp || defenderReports[0].GeneralExpGained != defenderExp {
			t.Fatalf("expected exp 721/220 without missed pursuit deaths, attacker=%d/%d defender=%d/%d calculated=%d", pvpTestGeneralExp(storedAttacker, "sunce"), attackerReports[0].GeneralExpGained, pvpTestGeneralExp(storedDefender, "caocao"), defenderReports[0].GeneralExpGained, defenderExp)
		}
		if battle.Plunder["wood"] <= 0 || storedAttacker.Resources.Items["wood"] != battle.Plunder["wood"] || storedDefender.Resources.Items["wood"] != 10000-battle.Plunder["wood"] {
			t.Fatalf("expected plunder resources to reconcile independently of missed pursuit, battle=%+v attacker=%+v defender=%+v", battle.Plunder, storedAttacker.Resources.Items, storedDefender.Resources.Items)
		}
	})

	t.Run("虎痴冲阵未命中但破敌防御生效", func(t *testing.T) {
		setTestFactionsAndGenerals(t, FactionsConfig{
			"wei": {Name: "魏国", Generals: []GeneralInfo{{ID: "xuchu", Name: "许褚"}}},
			"shu": {Name: "蜀国", Generals: []GeneralInfo{{ID: "liubei", Name: "刘备"}}},
		}, GeneralsConfig{Enabled: true, Heroes: map[string]GeneralHeroConfig{
			"xuchu": {
				ID: "xuchu", Name: "许褚", Faction: "wei", Enabled: true,
				SpecialTrait: GeneralTraitConfig{
					TraitID: "huchi_chongzhen", TraitType: general.TraitTypeSpecial, Enabled: true,
					Scope: "enemy_army", AllowedSides: []string{"attacker"},
					Params: map[string]float64{"triggerChance": 0, "enemyDefenseReductionRate": 0.2},
				},
				BonusTrait: GeneralTraitConfig{
					TraitID: "pojun_pofang", TraitType: general.TraitTypeBonus, Enabled: true,
					Scope: "enemy_army", AllowedSides: []string{"attacker"},
					Params: map[string]float64{"enemyDefenseReductionRate": 0.35},
				},
			},
			"liubei": {ID: "liubei", Name: "刘备", Faction: "shu", Enabled: true},
		}})
		svc, repo, attacker, defender := newPvpTestServiceForGenerals(t, "wei", "xuchu", "shu", "liubei")
		attacker.Army = []ArmyUnit{{UnitType: "weiInfantry", Amount: 1000}}
		defender.Army = []ArmyUnit{{UnitType: "shuInfantry", Amount: 1000}}
		defender.Buildings = nil
		repo.players[attacker.Player.ID] = attacker
		repo.players[defender.Player.ID] = defender

		started, err := svc.StartPvpAttack(PvpAttackRequest{
			PlayerID: attacker.Player.ID, TargetPlayerID: defender.Player.ID, MarchMode: PvpMarchTypeAttack,
			Troops: map[string]int{"weiInfantry": 1000}, GeneralIDs: []string{"xuchu"},
		})
		if err != nil {
			t.Fatalf("StartPvpAttack failed: %v", err)
		}
		forcePvpMarchDue(t, repo, started.March.ID)
		battle, err := svc.ResolvePvpMarch(started.March.ID)
		if err != nil {
			t.Fatalf("ResolvePvpMarch failed: %v", err)
		}
		if battle.Result["attackerPower"] != float64(10000) || battle.Result["defensePower"] != float64(7000) {
			t.Fatalf("expected only deterministic reduction to produce 10000/7000 when Huchi misses, result=%+v", battle.Result)
		}
		attackerLosses := pvpTestLossesFromBattle(t, battle, "attacker")
		defenderLosses := pvpTestLossesFromBattle(t, battle, "defender")
		if attackerLosses["weiInfantry"] != 602 || defenderLosses["shuInfantry"] != 1000 {
			t.Fatalf("expected exact attack-mode losses 602/1000 with defense 7, attacker=%+v defender=%+v", attackerLosses, defenderLosses)
		}

		attackerReports, _, err := repo.ListReports(attacker.Player.ID, 10, 0)
		if err != nil || len(attackerReports) != 1 {
			t.Fatalf("expected one attacker report, reports=%+v err=%v", attackerReports, err)
		}
		defenderReports, _, err := repo.ListReports(defender.Player.ID, 10, 0)
		if err != nil || len(defenderReports) != 1 {
			t.Fatalf("expected one defender report, reports=%+v err=%v", defenderReports, err)
		}
		for _, report := range []BattleReport{attackerReports[0], defenderReports[0]} {
			if len(report.TraitTriggered) != 1 || report.TraitTriggered[0] != "pojun_pofang" || len(report.TraitOutcomes) != 1 {
				t.Fatalf("expected only deterministic defense reduction in legacy timeline, report=%s triggered=%+v outcomes=%+v", report.ID, report.TraitTriggered, report.TraitOutcomes)
			}
			if _, exists := report.TraitOutcomes["huchi_chongzhen"]; exists {
				t.Fatalf("expected missed Huchi absent from legacy outcomes, report=%s outcomes=%+v", report.ID, report.TraitOutcomes)
			}
			pojun := report.TraitOutcomes["pojun_pofang"]
			infantry, infantryOK := pojun.Detail["infantryDefenseModifiedUnits"].(map[string]int)
			cavalry, cavalryOK := pojun.Detail["cavalryDefenseModifiedUnits"].(map[string]int)
			if !infantryOK || !cavalryOK || pojun.Detail["enemyDefenseReductionRate"] != 0.35 || infantry["shuInfantry"] != -3 || cavalry["shuInfantry"] != -3 {
				t.Fatalf("expected Pojun to reduce original defenses by actual -3/-3, report=%s outcome=%+v", report.ID, pojun)
			}
			if len(report.PvpAttackerGenerals) != 1 || !pvpSnapshotHasTrait(report.PvpAttackerGenerals[0], "huchi_chongzhen") || !pvpSnapshotHasTrait(report.PvpAttackerGenerals[0], "pojun_pofang") {
				t.Fatalf("expected Xu Chu snapshot to retain both owned traits, report=%s generals=%+v", report.ID, report.PvpAttackerGenerals)
			}
			if report.Detail == nil || report.Detail.SecondarySide == nil || len(report.Detail.Traits) != 1 || report.Detail.Traits[0].TraitID != "pojun_pofang" {
				t.Fatalf("expected standard timeline to contain only Pojun, report=%s detail=%+v", report.ID, report.Detail)
			}
			if !standardDetailGeneralHasTrait(report.Detail, "huchi_chongzhen") || !standardDetailGeneralHasTrait(report.Detail, "pojun_pofang") {
				t.Fatalf("expected standard general snapshot to retain both owned traits, report=%s detail=%+v", report.ID, report.Detail)
			}
			attackerUnit := attackDefenseCrossReportUnit(t, report.Detail.PrimarySide, "weiInfantry")
			defenderUnit := attackDefenseCrossReportUnit(t, *report.Detail.SecondarySide, "shuInfantry")
			if attackerUnit.AmountBefore != 1000 || attackerUnit.Lost != 602 || attackerUnit.Survived != 398 || defenderUnit.AmountBefore != 1000 || defenderUnit.Lost != 1000 || defenderUnit.Survived != 0 {
				t.Fatalf("expected standard rows 1000/602/398 and 1000/1000/0, report=%s attacker=%+v defender=%+v", report.ID, attackerUnit, defenderUnit)
			}
		}

		storedMarch, marchErr := repo.GetPvpMarch(started.March.ID)
		storedAttacker, attackerErr := repo.GetState(attacker.Player.ID)
		storedDefender, defenderErr := repo.GetState(defender.Player.ID)
		if marchErr != nil || attackerErr != nil || defenderErr != nil {
			t.Fatalf("expected stored battle state, march=%v attacker=%v defender=%v", marchErr, attackerErr, defenderErr)
		}
		if storedMarch.AttackTroops["weiInfantry"] != 398 || armySliceToMap(storedDefender.Army)["shuInfantry"] != 0 {
			t.Fatalf("expected authoritative survivors 398/0, march=%+v defender=%+v", storedMarch, storedDefender.Army)
		}
		if pvpTestGeneralExp(storedAttacker, "xuchu") != 1000 || attackerReports[0].GeneralExpGained != 1000 || pvpTestGeneralExp(storedDefender, "liubei") != 602 || defenderReports[0].GeneralExpGained != 602 {
			t.Fatalf("expected exp 1000/602 from core losses only, attacker=%d/%d defender=%d/%d", pvpTestGeneralExp(storedAttacker, "xuchu"), attackerReports[0].GeneralExpGained, pvpTestGeneralExp(storedDefender, "liubei"), defenderReports[0].GeneralExpGained)
		}
	})

	t.Run("百步穿杨未命中但老当益壮生效", func(t *testing.T) {
		setTestFactionsAndGenerals(t, FactionsConfig{
			"shu": {Name: "蜀国", Generals: []GeneralInfo{{ID: "huangzhong", Name: "黄忠"}}},
			"wei": {Name: "魏国", Generals: []GeneralInfo{{ID: "caocao", Name: "曹操"}}},
		}, GeneralsConfig{Enabled: true, Heroes: map[string]GeneralHeroConfig{
			"huangzhong": {
				ID: "huangzhong", Name: "黄忠", Faction: "shu", Enabled: true,
				SpecialTrait: GeneralTraitConfig{
					TraitID: "baibu_chuanyang", TraitType: general.TraitTypeSpecial, Enabled: true,
					Scope: "enemy_army", AllowedSides: []string{"attacker"},
					Params: map[string]float64{"triggerChance": 0, "enemyDefenseReductionRate": 0.2},
				},
				BonusTrait: GeneralTraitConfig{
					TraitID: "laodang_yizhuang", TraitType: general.TraitTypeBonus, Enabled: true,
					Scope: "enemy_army", Params: map[string]float64{"effectRate": 0.1},
				},
			},
			"caocao": {ID: "caocao", Name: "曹操", Faction: "wei", Enabled: true},
		}})
		svc, repo, attacker, defender := newPvpTestServiceForGenerals(t, "shu", "huangzhong", "wei", "caocao")
		attacker.Army = []ArmyUnit{{UnitType: "shuInfantry", Amount: 1000}}
		defender.Army = []ArmyUnit{{UnitType: "weiInfantry", Amount: 1000}}
		defender.Buildings = nil
		repo.players[attacker.Player.ID] = attacker
		repo.players[defender.Player.ID] = defender

		started, err := svc.StartPvpAttack(PvpAttackRequest{
			PlayerID: attacker.Player.ID, TargetPlayerID: defender.Player.ID, MarchMode: PvpMarchTypePlunder,
			Troops: map[string]int{"shuInfantry": 1000}, GeneralIDs: []string{"huangzhong"},
		})
		if err != nil {
			t.Fatalf("StartPvpAttack failed: %v", err)
		}
		forcePvpMarchDue(t, repo, started.March.ID)
		battle, err := svc.ResolvePvpMarch(started.March.ID)
		if err != nil {
			t.Fatalf("ResolvePvpMarch failed: %v", err)
		}
		if battle.Result["attackerPower"] != float64(10000) || battle.Result["defensePower"] != float64(10000) {
			t.Fatalf("expected missed defense break to preserve 10000/10000 core powers, result=%+v", battle.Result)
		}
		attackerLosses := pvpTestLossesFromBattle(t, battle, "attacker")
		defenderLosses := pvpTestLossesFromBattle(t, battle, "defender")
		if attackerLosses["shuInfantry"] != 500 || defenderLosses["weiInfantry"] != 600 {
			t.Fatalf("expected core 500/500 plus 100 Laodang defender losses, attacker=%+v defender=%+v", attackerLosses, defenderLosses)
		}

		attackerReports, _, err := repo.ListReports(attacker.Player.ID, 10, 0)
		if err != nil || len(attackerReports) != 1 {
			t.Fatalf("expected one attacker report, reports=%+v err=%v", attackerReports, err)
		}
		defenderReports, _, err := repo.ListReports(defender.Player.ID, 10, 0)
		if err != nil || len(defenderReports) != 1 {
			t.Fatalf("expected one defender report, reports=%+v err=%v", defenderReports, err)
		}
		for _, report := range []BattleReport{attackerReports[0], defenderReports[0]} {
			if len(report.TraitTriggered) != 1 || report.TraitTriggered[0] != "laodang_yizhuang" || len(report.TraitOutcomes) != 1 {
				t.Fatalf("expected only deterministic after-combat damage in legacy timeline, report=%s triggered=%+v outcomes=%+v", report.ID, report.TraitTriggered, report.TraitOutcomes)
			}
			if _, exists := report.TraitOutcomes["baibu_chuanyang"]; exists {
				t.Fatalf("expected missed Baibu absent from legacy outcomes, report=%s outcomes=%+v", report.ID, report.TraitOutcomes)
			}
			laodang := report.TraitOutcomes["laodang_yizhuang"]
			extra, extraOK := laodang.Detail["extraLosses"].(map[string]int)
			if !extraOK || laodang.Detail["effectRate"] != 0.1 || extra["weiInfantry"] != 100 {
				t.Fatalf("expected Laodang to add exact 100 losses after core combat, report=%s outcome=%+v", report.ID, laodang)
			}
			if len(report.PvpAttackerGenerals) != 1 || !pvpSnapshotHasTrait(report.PvpAttackerGenerals[0], "baibu_chuanyang") || !pvpSnapshotHasTrait(report.PvpAttackerGenerals[0], "laodang_yizhuang") {
				t.Fatalf("expected Huang Zhong snapshot to retain both owned traits, report=%s generals=%+v", report.ID, report.PvpAttackerGenerals)
			}
			if report.Detail == nil || report.Detail.SecondarySide == nil || len(report.Detail.Traits) != 1 || report.Detail.Traits[0].TraitID != "laodang_yizhuang" {
				t.Fatalf("expected standard timeline to contain only Laodang, report=%s detail=%+v", report.ID, report.Detail)
			}
			if !standardDetailGeneralHasTrait(report.Detail, "baibu_chuanyang") || !standardDetailGeneralHasTrait(report.Detail, "laodang_yizhuang") {
				t.Fatalf("expected standard general snapshot to retain both owned traits, report=%s detail=%+v", report.ID, report.Detail)
			}
			attackerUnit := attackDefenseCrossReportUnit(t, report.Detail.PrimarySide, "shuInfantry")
			defenderUnit := attackDefenseCrossReportUnit(t, *report.Detail.SecondarySide, "weiInfantry")
			if attackerUnit.AmountBefore != 1000 || attackerUnit.Lost != 500 || attackerUnit.Survived != 500 || defenderUnit.AmountBefore != 1000 || defenderUnit.Lost != 600 || defenderUnit.Survived != 400 {
				t.Fatalf("expected standard rows 1000/500/500 and 1000/600/400, report=%s attacker=%+v defender=%+v", report.ID, attackerUnit, defenderUnit)
			}
		}

		storedMarch, marchErr := repo.GetPvpMarch(started.March.ID)
		storedAttacker, attackerErr := repo.GetState(attacker.Player.ID)
		storedDefender, defenderErr := repo.GetState(defender.Player.ID)
		if marchErr != nil || attackerErr != nil || defenderErr != nil {
			t.Fatalf("expected stored battle state, march=%v attacker=%v defender=%v", marchErr, attackerErr, defenderErr)
		}
		if storedMarch.AttackTroops["shuInfantry"] != 500 || armySliceToMap(storedDefender.Army)["weiInfantry"] != 400 {
			t.Fatalf("expected authoritative survivors 500/400, march=%+v defender=%+v", storedMarch, storedDefender.Army)
		}
		if pvpTestGeneralExp(storedAttacker, "huangzhong") != 600 || attackerReports[0].GeneralExpGained != 600 || pvpTestGeneralExp(storedDefender, "caocao") != 500 || defenderReports[0].GeneralExpGained != 500 {
			t.Fatalf("expected exp 600/500 including only real after-combat deaths, attacker=%d/%d defender=%d/%d", pvpTestGeneralExp(storedAttacker, "huangzhong"), attackerReports[0].GeneralExpGained, pvpTestGeneralExp(storedDefender, "caocao"), defenderReports[0].GeneralExpGained)
		}
	})

	t.Run("水淹七军未命中但武圣破军生效", func(t *testing.T) {
		setTestFactionsAndGenerals(t, FactionsConfig{
			"shu": {Name: "蜀国", Generals: []GeneralInfo{{ID: "guanyu", Name: "关羽"}}},
			"wei": {Name: "魏国", Generals: []GeneralInfo{{ID: "caocao", Name: "曹操"}}},
		}, GeneralsConfig{Enabled: true, Heroes: map[string]GeneralHeroConfig{
			"guanyu": {
				ID: "guanyu", Name: "关羽", Faction: "shu", Enabled: true,
				SpecialTrait: GeneralTraitConfig{
					TraitID: "shuiyan_qijun", TraitType: general.TraitTypeSpecial, Enabled: true,
					Scope:  "enemy_army",
					Params: map[string]float64{"triggerChance": 0, "effectRate": 0.35, "maxAffectedRate": 0.35},
				},
				BonusTrait: GeneralTraitConfig{
					TraitID: "wusheng_pojun", TraitType: general.TraitTypeBonus, Enabled: true,
					Scope: "self_army", AllowedSides: []string{"attacker"},
					Params: map[string]float64{"attackBonusRate": 0.2},
				},
			},
			"caocao": {ID: "caocao", Name: "曹操", Faction: "wei", Enabled: true},
		}})
		svc, repo, attacker, defender := newPvpTestServiceForGenerals(t, "shu", "guanyu", "wei", "caocao")
		attacker.Army = []ArmyUnit{{UnitType: "shuInfantry", Amount: 1000}}
		defender.Army = []ArmyUnit{{UnitType: "weiInfantry", Amount: 1000}}
		defender.Buildings = nil
		repo.players[attacker.Player.ID] = attacker
		repo.players[defender.Player.ID] = defender

		started, err := svc.StartPvpAttack(PvpAttackRequest{
			PlayerID: attacker.Player.ID, TargetPlayerID: defender.Player.ID, MarchMode: PvpMarchTypePlunder,
			Troops: map[string]int{"shuInfantry": 1000}, GeneralIDs: []string{"guanyu"},
		})
		if err != nil {
			t.Fatalf("StartPvpAttack failed: %v", err)
		}
		forcePvpMarchDue(t, repo, started.March.ID)
		battle, err := svc.ResolvePvpMarch(started.March.ID)
		if err != nil {
			t.Fatalf("ResolvePvpMarch failed: %v", err)
		}
		if battle.Result["attackerPower"] != float64(12000) || battle.Result["defensePower"] != float64(10000) {
			t.Fatalf("expected attack bonus but no pre-battle casualties to produce 12000/10000, result=%+v", battle.Result)
		}
		attackerLosses := pvpTestLossesFromBattle(t, battle, "attacker")
		defenderLosses := pvpTestLossesFromBattle(t, battle, "defender")
		if attackerLosses["shuInfantry"] != 435 || defenderLosses["weiInfantry"] != 564 {
			t.Fatalf("expected full 1000 defenders to enter core and produce losses 435/564, attacker=%+v defender=%+v", attackerLosses, defenderLosses)
		}

		attackerReports, _, err := repo.ListReports(attacker.Player.ID, 10, 0)
		if err != nil || len(attackerReports) != 1 {
			t.Fatalf("expected one attacker report, reports=%+v err=%v", attackerReports, err)
		}
		defenderReports, _, err := repo.ListReports(defender.Player.ID, 10, 0)
		if err != nil || len(defenderReports) != 1 {
			t.Fatalf("expected one defender report, reports=%+v err=%v", defenderReports, err)
		}
		for _, report := range []BattleReport{attackerReports[0], defenderReports[0]} {
			if len(report.TraitTriggered) != 1 || report.TraitTriggered[0] != "wusheng_pojun" || len(report.TraitOutcomes) != 1 {
				t.Fatalf("expected only deterministic attack bonus in legacy timeline, report=%s triggered=%+v outcomes=%+v", report.ID, report.TraitTriggered, report.TraitOutcomes)
			}
			if _, exists := report.TraitOutcomes["shuiyan_qijun"]; exists {
				t.Fatalf("expected missed Shuiyan absent from legacy outcomes, report=%s outcomes=%+v", report.ID, report.TraitOutcomes)
			}
			wusheng := report.TraitOutcomes["wusheng_pojun"]
			modified, modifiedOK := wusheng.Detail["attackModifiedUnits"].(map[string]int)
			if !modifiedOK || wusheng.Detail["attackBonusRate"] != 0.2 || modified["shuInfantry"] != 2 {
				t.Fatalf("expected Wusheng to keep actual +2 attack, report=%s outcome=%+v", report.ID, wusheng)
			}
			if len(report.PvpAttackerGenerals) != 1 || !pvpSnapshotHasTrait(report.PvpAttackerGenerals[0], "shuiyan_qijun") || !pvpSnapshotHasTrait(report.PvpAttackerGenerals[0], "wusheng_pojun") {
				t.Fatalf("expected Guan Yu snapshot to retain both owned traits, report=%s generals=%+v", report.ID, report.PvpAttackerGenerals)
			}
			if report.Detail == nil || report.Detail.SecondarySide == nil || len(report.Detail.Traits) != 1 || report.Detail.Traits[0].TraitID != "wusheng_pojun" {
				t.Fatalf("expected standard timeline to contain only Wusheng, report=%s detail=%+v", report.ID, report.Detail)
			}
			if !standardDetailGeneralHasTrait(report.Detail, "shuiyan_qijun") || !standardDetailGeneralHasTrait(report.Detail, "wusheng_pojun") {
				t.Fatalf("expected standard general snapshot to retain both owned traits, report=%s detail=%+v", report.ID, report.Detail)
			}
			attackerUnit := attackDefenseCrossReportUnit(t, report.Detail.PrimarySide, "shuInfantry")
			defenderUnit := attackDefenseCrossReportUnit(t, *report.Detail.SecondarySide, "weiInfantry")
			if attackerUnit.AmountBefore != 1000 || attackerUnit.Lost != 435 || attackerUnit.Survived != 565 || defenderUnit.AmountBefore != 1000 || defenderUnit.Lost != 564 || defenderUnit.Survived != 436 {
				t.Fatalf("expected standard rows 1000/435/565 and 1000/564/436, report=%s attacker=%+v defender=%+v", report.ID, attackerUnit, defenderUnit)
			}
		}

		storedMarch, marchErr := repo.GetPvpMarch(started.March.ID)
		storedAttacker, attackerErr := repo.GetState(attacker.Player.ID)
		storedDefender, defenderErr := repo.GetState(defender.Player.ID)
		if marchErr != nil || attackerErr != nil || defenderErr != nil {
			t.Fatalf("expected stored battle state, march=%v attacker=%v defender=%v", marchErr, attackerErr, defenderErr)
		}
		if storedMarch.AttackTroops["shuInfantry"] != 565 || armySliceToMap(storedDefender.Army)["weiInfantry"] != 436 {
			t.Fatalf("expected authoritative survivors 565/436, march=%+v defender=%+v", storedMarch, storedDefender.Army)
		}
		if pvpTestGeneralExp(storedAttacker, "guanyu") != 564 || attackerReports[0].GeneralExpGained != 564 || pvpTestGeneralExp(storedDefender, "caocao") != 435 || defenderReports[0].GeneralExpGained != 435 {
			t.Fatalf("expected exp 564/435 without pre-battle deaths, attacker=%d/%d defender=%d/%d", pvpTestGeneralExp(storedAttacker, "guanyu"), attackerReports[0].GeneralExpGained, pvpTestGeneralExp(storedDefender, "caocao"), defenderReports[0].GeneralExpGained)
		}
	})

	t.Run("威震震慑未命中但威震逍遥生效", func(t *testing.T) {
		setTestFactionsAndGenerals(t, FactionsConfig{
			"wei": {Name: "魏国", Generals: []GeneralInfo{{ID: "zhangliao", Name: "张辽"}}},
			"shu": {Name: "蜀国", Generals: []GeneralInfo{{ID: "liubei", Name: "刘备"}}},
		}, GeneralsConfig{Enabled: true, Heroes: map[string]GeneralHeroConfig{
			"zhangliao": {
				ID: "zhangliao", Name: "张辽", Faction: "wei", Enabled: true,
				SpecialTrait: GeneralTraitConfig{
					TraitID: "weizhen_zhenhe", TraitType: general.TraitTypeSpecial, Enabled: true,
					Scope:  "enemy_army",
					Params: map[string]float64{"triggerChance": 0, "effectRate": 0.2, "maxAffectedRate": 0.2},
				},
				BonusTrait: GeneralTraitConfig{
					TraitID: "weizhen_xiaoyao", TraitType: general.TraitTypeBonus, Enabled: true,
					Scope: "self_army", TargetUnitType: "cavalry", AllowedSides: []string{"attacker"},
					Params: map[string]float64{"attackBonusRate": 0.35},
				},
			},
			"liubei": {ID: "liubei", Name: "刘备", Faction: "shu", Enabled: true},
		}})
		svc, repo, attacker, defender := newPvpTestServiceForGenerals(t, "wei", "zhangliao", "shu", "liubei")
		attacker.Army = []ArmyUnit{{UnitType: "weiCavalry", Amount: 1000}}
		defender.Army = []ArmyUnit{{UnitType: "shuInfantry", Amount: 1000}}
		repo.players[attacker.Player.ID] = attacker
		repo.players[defender.Player.ID] = defender

		started, err := svc.StartPvpAttack(PvpAttackRequest{
			PlayerID: attacker.Player.ID, TargetPlayerID: defender.Player.ID, MarchMode: PvpMarchTypeAttack,
			Troops: map[string]int{"weiCavalry": 1000}, GeneralIDs: []string{"zhangliao"},
		})
		if err != nil {
			t.Fatalf("StartPvpAttack failed: %v", err)
		}
		forcePvpMarchDue(t, repo, started.March.ID)
		battle, err := svc.ResolvePvpMarch(started.March.ID)
		if err != nil {
			t.Fatalf("ResolvePvpMarch failed: %v", err)
		}
		if battle.Result["attackerPower"] != float64(19000) || battle.Result["defensePower"] != float64(8160) {
			t.Fatalf("expected cavalry attack bonus but no suppression to produce 19000/8160, result=%+v", battle.Result)
		}
		attackerLosses := pvpTestLossesFromBattle(t, battle, "attacker")
		defenderLosses := pvpTestLossesFromBattle(t, battle, "defender")
		if attackerLosses["weiCavalry"] != 300 || defenderLosses["shuInfantry"] != 1000 {
			t.Fatalf("expected full 1000 defenders to enter attack-mode combat and produce losses 300/1000, attacker=%+v defender=%+v", attackerLosses, defenderLosses)
		}

		attackerReports, _, err := repo.ListReports(attacker.Player.ID, 10, 0)
		if err != nil || len(attackerReports) != 1 {
			t.Fatalf("expected one attacker report, reports=%+v err=%v", attackerReports, err)
		}
		defenderReports, _, err := repo.ListReports(defender.Player.ID, 10, 0)
		if err != nil || len(defenderReports) != 1 {
			t.Fatalf("expected one defender report, reports=%+v err=%v", defenderReports, err)
		}
		for _, report := range []BattleReport{attackerReports[0], defenderReports[0]} {
			if len(report.TraitTriggered) != 1 || report.TraitTriggered[0] != "weizhen_xiaoyao" || len(report.TraitOutcomes) != 1 {
				t.Fatalf("expected only deterministic cavalry attack bonus in legacy timeline, report=%s triggered=%+v outcomes=%+v", report.ID, report.TraitTriggered, report.TraitOutcomes)
			}
			if _, exists := report.TraitOutcomes["weizhen_zhenhe"]; exists {
				t.Fatalf("expected missed Weizhen suppression absent from legacy outcomes, report=%s outcomes=%+v", report.ID, report.TraitOutcomes)
			}
			xiaoyao := report.TraitOutcomes["weizhen_xiaoyao"]
			modified, modifiedOK := xiaoyao.Detail["attackModifiedUnits"].(map[string]int)
			if !modifiedOK || xiaoyao.Detail["attackBonusRate"] != 0.35 || modified["weiCavalry"] != 5 {
				t.Fatalf("expected Weizhen Xiaoyao to keep actual +5 cavalry attack, report=%s outcome=%+v", report.ID, xiaoyao)
			}
			if len(report.PvpAttackerGenerals) != 1 || !pvpSnapshotHasTrait(report.PvpAttackerGenerals[0], "weizhen_zhenhe") || !pvpSnapshotHasTrait(report.PvpAttackerGenerals[0], "weizhen_xiaoyao") {
				t.Fatalf("expected Zhang Liao snapshot to retain both owned traits, report=%s generals=%+v", report.ID, report.PvpAttackerGenerals)
			}
			if report.Detail == nil || report.Detail.SecondarySide == nil || len(report.Detail.Traits) != 1 || report.Detail.Traits[0].TraitID != "weizhen_xiaoyao" {
				t.Fatalf("expected standard timeline to contain only Weizhen Xiaoyao, report=%s detail=%+v", report.ID, report.Detail)
			}
			if !standardDetailGeneralHasTrait(report.Detail, "weizhen_zhenhe") || !standardDetailGeneralHasTrait(report.Detail, "weizhen_xiaoyao") {
				t.Fatalf("expected standard general snapshot to retain both owned traits, report=%s detail=%+v", report.ID, report.Detail)
			}
			attackerUnit := attackDefenseCrossReportUnit(t, report.Detail.PrimarySide, "weiCavalry")
			defenderUnit := attackDefenseCrossReportUnit(t, *report.Detail.SecondarySide, "shuInfantry")
			if attackerUnit.AmountBefore != 1000 || attackerUnit.Lost != 300 || attackerUnit.Survived != 700 || defenderUnit.AmountBefore != 1000 || defenderUnit.Lost != 1000 || defenderUnit.Survived != 0 {
				t.Fatalf("expected standard rows 1000/300/700 and 1000/1000/0, report=%s attacker=%+v defender=%+v", report.ID, attackerUnit, defenderUnit)
			}
		}

		storedMarch, marchErr := repo.GetPvpMarch(started.March.ID)
		storedAttacker, attackerErr := repo.GetState(attacker.Player.ID)
		storedDefender, defenderErr := repo.GetState(defender.Player.ID)
		if marchErr != nil || attackerErr != nil || defenderErr != nil {
			t.Fatalf("expected stored battle state, march=%v attacker=%v defender=%v", marchErr, attackerErr, defenderErr)
		}
		if storedMarch.AttackTroops["weiCavalry"] != 700 || armySliceToMap(storedDefender.Army)["shuInfantry"] != 0 {
			t.Fatalf("expected authoritative survivors 700/0, march=%+v defender=%+v", storedMarch, storedDefender.Army)
		}
		if pvpTestGeneralExp(storedAttacker, "zhangliao") != 1000 || attackerReports[0].GeneralExpGained != 1000 || pvpTestGeneralExp(storedDefender, "liubei") != 600 || defenderReports[0].GeneralExpGained != 600 {
			t.Fatalf("expected exp 1000/600 with cavalry maintenance weighting, attacker=%d/%d defender=%d/%d", pvpTestGeneralExp(storedAttacker, "zhangliao"), attackerReports[0].GeneralExpGained, pvpTestGeneralExp(storedDefender, "liubei"), defenderReports[0].GeneralExpGained)
		}
	})

	t.Run("震慑全军未命中但万人怒吼生效", func(t *testing.T) {
		setTestFactionsAndGenerals(t, FactionsConfig{
			"shu": {Name: "蜀国", Generals: []GeneralInfo{{ID: "zhangfei", Name: "张飞"}}},
			"wei": {Name: "魏国", Generals: []GeneralInfo{{ID: "caocao", Name: "曹操"}}},
		}, GeneralsConfig{Enabled: true, Heroes: map[string]GeneralHeroConfig{
			"zhangfei": {
				ID: "zhangfei", Name: "张飞", Faction: "shu", Enabled: true,
				SpecialTrait: GeneralTraitConfig{
					TraitID: "zhenhe_quanjun", TraitType: general.TraitTypeSpecial, Enabled: true,
					Scope:  "enemy_army",
					Params: map[string]float64{"triggerChance": 0, "effectRate": 0.5, "maxAffectedRate": 0.5},
				},
				BonusTrait: GeneralTraitConfig{
					TraitID: "wanren_nuhou", TraitType: general.TraitTypeBonus, Enabled: true,
					Scope: "self_army", TargetUnitType: "infantry", AllowedSides: []string{"attacker"},
					Params: map[string]float64{"attackBonusRate": 0.2},
				},
			},
			"caocao": {ID: "caocao", Name: "曹操", Faction: "wei", Enabled: true},
		}})
		svc, repo, attacker, defender := newPvpTestServiceForGenerals(t, "shu", "zhangfei", "wei", "caocao")
		attacker.Army = []ArmyUnit{{UnitType: "shuInfantry", Amount: 1000}}
		defender.Army = []ArmyUnit{{UnitType: "weiInfantry", Amount: 1000}}
		repo.players[attacker.Player.ID] = attacker
		repo.players[defender.Player.ID] = defender

		started, err := svc.StartPvpAttack(PvpAttackRequest{
			PlayerID: attacker.Player.ID, TargetPlayerID: defender.Player.ID, MarchMode: PvpMarchTypeAttack,
			Troops: map[string]int{"shuInfantry": 1000}, GeneralIDs: []string{"zhangfei"},
		})
		if err != nil {
			t.Fatalf("StartPvpAttack failed: %v", err)
		}
		forcePvpMarchDue(t, repo, started.March.ID)
		battle, err := svc.ResolvePvpMarch(started.March.ID)
		if err != nil {
			t.Fatalf("ResolvePvpMarch failed: %v", err)
		}
		if battle.Result["attackerPower"] != float64(12000) || battle.Result["defensePower"] != float64(10300) {
			t.Fatalf("expected infantry attack bonus but no suppression to produce 12000/10300, result=%+v", battle.Result)
		}
		attackerLosses := pvpTestLossesFromBattle(t, battle, "attacker")
		defenderLosses := pvpTestLossesFromBattle(t, battle, "defender")
		if attackerLosses["shuInfantry"] != 804 || defenderLosses["weiInfantry"] != 1000 {
			t.Fatalf("expected full 1000 defenders to enter attack-mode combat and produce losses 804/1000, attacker=%+v defender=%+v", attackerLosses, defenderLosses)
		}

		attackerReports, _, err := repo.ListReports(attacker.Player.ID, 10, 0)
		if err != nil || len(attackerReports) != 1 {
			t.Fatalf("expected one attacker report, reports=%+v err=%v", attackerReports, err)
		}
		defenderReports, _, err := repo.ListReports(defender.Player.ID, 10, 0)
		if err != nil || len(defenderReports) != 1 {
			t.Fatalf("expected one defender report, reports=%+v err=%v", defenderReports, err)
		}
		for _, report := range []BattleReport{attackerReports[0], defenderReports[0]} {
			if len(report.TraitTriggered) != 1 || report.TraitTriggered[0] != "wanren_nuhou" || len(report.TraitOutcomes) != 1 {
				t.Fatalf("expected only deterministic infantry attack bonus in legacy timeline, report=%s triggered=%+v outcomes=%+v", report.ID, report.TraitTriggered, report.TraitOutcomes)
			}
			if _, exists := report.TraitOutcomes["zhenhe_quanjun"]; exists {
				t.Fatalf("expected missed full-army suppression absent from legacy outcomes, report=%s outcomes=%+v", report.ID, report.TraitOutcomes)
			}
			wanren := report.TraitOutcomes["wanren_nuhou"]
			modified, modifiedOK := wanren.Detail["attackModifiedUnits"].(map[string]int)
			if !modifiedOK || wanren.Detail["attackBonusRate"] != 0.2 || modified["shuInfantry"] != 2 {
				t.Fatalf("expected Wanren Nuhou to keep actual +2 infantry attack, report=%s outcome=%+v", report.ID, wanren)
			}
			if len(report.PvpAttackerGenerals) != 1 || !pvpSnapshotHasTrait(report.PvpAttackerGenerals[0], "zhenhe_quanjun") || !pvpSnapshotHasTrait(report.PvpAttackerGenerals[0], "wanren_nuhou") {
				t.Fatalf("expected Zhang Fei snapshot to retain both owned traits, report=%s generals=%+v", report.ID, report.PvpAttackerGenerals)
			}
			if report.Detail == nil || report.Detail.SecondarySide == nil || len(report.Detail.Traits) != 1 || report.Detail.Traits[0].TraitID != "wanren_nuhou" {
				t.Fatalf("expected standard timeline to contain only Wanren Nuhou, report=%s detail=%+v", report.ID, report.Detail)
			}
			if !standardDetailGeneralHasTrait(report.Detail, "zhenhe_quanjun") || !standardDetailGeneralHasTrait(report.Detail, "wanren_nuhou") {
				t.Fatalf("expected standard general snapshot to retain both owned traits, report=%s detail=%+v", report.ID, report.Detail)
			}
			attackerUnit := attackDefenseCrossReportUnit(t, report.Detail.PrimarySide, "shuInfantry")
			defenderUnit := attackDefenseCrossReportUnit(t, *report.Detail.SecondarySide, "weiInfantry")
			if attackerUnit.AmountBefore != 1000 || attackerUnit.Lost != 804 || attackerUnit.Survived != 196 || defenderUnit.AmountBefore != 1000 || defenderUnit.Lost != 1000 || defenderUnit.Survived != 0 {
				t.Fatalf("expected standard rows 1000/804/196 and 1000/1000/0, report=%s attacker=%+v defender=%+v", report.ID, attackerUnit, defenderUnit)
			}
		}

		storedMarch, marchErr := repo.GetPvpMarch(started.March.ID)
		storedAttacker, attackerErr := repo.GetState(attacker.Player.ID)
		storedDefender, defenderErr := repo.GetState(defender.Player.ID)
		if marchErr != nil || attackerErr != nil || defenderErr != nil {
			t.Fatalf("expected stored battle state, march=%v attacker=%v defender=%v", marchErr, attackerErr, defenderErr)
		}
		if storedMarch.AttackTroops["shuInfantry"] != 196 || armySliceToMap(storedDefender.Army)["weiInfantry"] != 0 {
			t.Fatalf("expected authoritative survivors 196/0, march=%+v defender=%+v", storedMarch, storedDefender.Army)
		}
		if pvpTestGeneralExp(storedAttacker, "zhangfei") != 1000 || attackerReports[0].GeneralExpGained != 1000 || pvpTestGeneralExp(storedDefender, "caocao") != 804 || defenderReports[0].GeneralExpGained != 804 {
			t.Fatalf("expected exp 1000/804 from all real infantry deaths, attacker=%d/%d defender=%d/%d", pvpTestGeneralExp(storedAttacker, "zhangfei"), attackerReports[0].GeneralExpGained, pvpTestGeneralExp(storedDefender, "caocao"), defenderReports[0].GeneralExpGained)
		}
	})

	t.Run("火烧联营未命中但连营增伤生效", func(t *testing.T) {
		setTestFactionsAndGenerals(t, FactionsConfig{
			"wu":  {Name: "吴国", Generals: []GeneralInfo{{ID: "luxun", Name: "陆逊"}}},
			"wei": {Name: "魏国", Generals: []GeneralInfo{{ID: "caocao", Name: "曹操"}}},
		}, GeneralsConfig{Enabled: true, Heroes: map[string]GeneralHeroConfig{
			"luxun": {
				ID: "luxun", Name: "陆逊", Faction: "wu", Enabled: true,
				SpecialTrait: GeneralTraitConfig{
					TraitID: "huoshao_lianying", TraitType: general.TraitTypeSpecial, Enabled: true,
					Scope: "enemy_army", TargetUnitType: "infantry",
					Params: map[string]float64{"triggerChance": 0, "effectRate": 1, "maxAffectedRate": 1},
				},
				BonusTrait: GeneralTraitConfig{
					TraitID: "lianying_zengshang", TraitType: general.TraitTypeBonus, Enabled: true,
					Scope: "enemy_army", TargetUnitType: "infantry",
					Params: map[string]float64{"effectRate": 0.1},
				},
			},
			"caocao": {ID: "caocao", Name: "曹操", Faction: "wei", Enabled: true},
		}})
		svc, repo, attacker, defender := newPvpTestServiceForGenerals(t, "wu", "luxun", "wei", "caocao")
		attacker.Army = []ArmyUnit{{UnitType: "wuInfantry", Amount: 1000}}
		defender.Army = []ArmyUnit{{UnitType: "weiInfantry", Amount: 1000}}
		defender.Buildings = nil
		repo.players[attacker.Player.ID] = attacker
		repo.players[defender.Player.ID] = defender

		started, err := svc.StartPvpAttack(PvpAttackRequest{
			PlayerID: attacker.Player.ID, TargetPlayerID: defender.Player.ID, MarchMode: PvpMarchTypePlunder,
			Troops: map[string]int{"wuInfantry": 1000}, GeneralIDs: []string{"luxun"},
		})
		if err != nil {
			t.Fatalf("StartPvpAttack failed: %v", err)
		}
		forcePvpMarchDue(t, repo, started.March.ID)
		battle, err := svc.ResolvePvpMarch(started.March.ID)
		if err != nil {
			t.Fatalf("ResolvePvpMarch failed: %v", err)
		}
		if battle.Result["attackerPower"] != float64(10000) || battle.Result["defensePower"] != float64(10000) {
			t.Fatalf("expected after-combat traits to preserve 10000/10000 core powers, result=%+v", battle.Result)
		}
		attackerLosses := pvpTestLossesFromBattle(t, battle, "attacker")
		defenderLosses := pvpTestLossesFromBattle(t, battle, "defender")
		if attackerLosses["wuInfantry"] != 500 || defenderLosses["weiInfantry"] != 600 {
			t.Fatalf("expected core 500/500 plus 100 Lianying defender losses, attacker=%+v defender=%+v", attackerLosses, defenderLosses)
		}

		attackerReports, _, err := repo.ListReports(attacker.Player.ID, 10, 0)
		if err != nil || len(attackerReports) != 1 {
			t.Fatalf("expected one attacker report, reports=%+v err=%v", attackerReports, err)
		}
		defenderReports, _, err := repo.ListReports(defender.Player.ID, 10, 0)
		if err != nil || len(defenderReports) != 1 {
			t.Fatalf("expected one defender report, reports=%+v err=%v", defenderReports, err)
		}
		for _, report := range []BattleReport{attackerReports[0], defenderReports[0]} {
			if len(report.TraitTriggered) != 1 || report.TraitTriggered[0] != "lianying_zengshang" || len(report.TraitOutcomes) != 1 {
				t.Fatalf("expected only deterministic Lianying damage in legacy timeline, report=%s triggered=%+v outcomes=%+v", report.ID, report.TraitTriggered, report.TraitOutcomes)
			}
			if _, exists := report.TraitOutcomes["huoshao_lianying"]; exists {
				t.Fatalf("expected missed fire absent from legacy outcomes, report=%s outcomes=%+v", report.ID, report.TraitOutcomes)
			}
			lianying := report.TraitOutcomes["lianying_zengshang"]
			extra, extraOK := lianying.Detail["targetExtraLosses"].(map[string]int)
			if !extraOK || lianying.Detail["effectRate"] != 0.1 || extra["weiInfantry"] != 100 {
				t.Fatalf("expected Lianying to add exact 100 infantry losses after core combat, report=%s outcome=%+v", report.ID, lianying)
			}
			if len(report.PvpAttackerGenerals) != 1 || !pvpSnapshotHasTrait(report.PvpAttackerGenerals[0], "huoshao_lianying") || !pvpSnapshotHasTrait(report.PvpAttackerGenerals[0], "lianying_zengshang") {
				t.Fatalf("expected Lu Xun snapshot to retain both owned traits, report=%s generals=%+v", report.ID, report.PvpAttackerGenerals)
			}
			if report.Detail == nil || report.Detail.SecondarySide == nil || len(report.Detail.Traits) != 1 || report.Detail.Traits[0].TraitID != "lianying_zengshang" {
				t.Fatalf("expected standard timeline to contain only Lianying damage, report=%s detail=%+v", report.ID, report.Detail)
			}
			if !standardDetailGeneralHasTrait(report.Detail, "huoshao_lianying") || !standardDetailGeneralHasTrait(report.Detail, "lianying_zengshang") {
				t.Fatalf("expected standard general snapshot to retain both owned traits, report=%s detail=%+v", report.ID, report.Detail)
			}
			attackerUnit := attackDefenseCrossReportUnit(t, report.Detail.PrimarySide, "wuInfantry")
			defenderUnit := attackDefenseCrossReportUnit(t, *report.Detail.SecondarySide, "weiInfantry")
			if attackerUnit.AmountBefore != 1000 || attackerUnit.Lost != 500 || attackerUnit.Survived != 500 || defenderUnit.AmountBefore != 1000 || defenderUnit.Lost != 600 || defenderUnit.Survived != 400 {
				t.Fatalf("expected standard rows 1000/500/500 and 1000/600/400, report=%s attacker=%+v defender=%+v", report.ID, attackerUnit, defenderUnit)
			}
		}

		storedMarch, marchErr := repo.GetPvpMarch(started.March.ID)
		storedAttacker, attackerErr := repo.GetState(attacker.Player.ID)
		storedDefender, defenderErr := repo.GetState(defender.Player.ID)
		if marchErr != nil || attackerErr != nil || defenderErr != nil {
			t.Fatalf("expected stored battle state, march=%v attacker=%v defender=%v", marchErr, attackerErr, defenderErr)
		}
		if storedMarch.AttackTroops["wuInfantry"] != 500 || armySliceToMap(storedDefender.Army)["weiInfantry"] != 400 {
			t.Fatalf("expected authoritative survivors 500/400, march=%+v defender=%+v", storedMarch, storedDefender.Army)
		}
		if pvpTestGeneralExp(storedAttacker, "luxun") != 600 || attackerReports[0].GeneralExpGained != 600 || pvpTestGeneralExp(storedDefender, "caocao") != 500 || defenderReports[0].GeneralExpGained != 500 {
			t.Fatalf("expected exp 600/500 including only real infantry deaths, attacker=%d/%d defender=%d/%d", pvpTestGeneralExp(storedAttacker, "luxun"), attackerReports[0].GeneralExpGained, pvpTestGeneralExp(storedDefender, "caocao"), defenderReports[0].GeneralExpGained)
		}
	})

	t.Run("苦肉计未命中且双方后续伤害均生效", func(t *testing.T) {
		setTestFactionsAndGenerals(t, FactionsConfig{
			"wu":  {Name: "吴国", Generals: []GeneralInfo{{ID: "huanggai", Name: "黄盖"}}},
			"shu": {Name: "蜀国", Generals: []GeneralInfo{{ID: "huangzhong", Name: "黄忠"}}},
		}, GeneralsConfig{Enabled: true, Heroes: map[string]GeneralHeroConfig{
			"huanggai": {
				ID: "huanggai", Name: "黄盖", Faction: "wu", Enabled: true,
				SpecialTrait: GeneralTraitConfig{
					TraitID: "kurouji", TraitType: general.TraitTypeSpecial, Enabled: true,
					Scope: "enemy_traits", Params: map[string]float64{"triggerChance": 0, "disableTraitCount": 1},
				},
				BonusTrait: GeneralTraitConfig{
					TraitID: "kurou_fanji", TraitType: general.TraitTypeBonus, Enabled: true,
					Scope: "enemy_army", Params: map[string]float64{"effectRate": 0.1},
				},
			},
			"huangzhong": {
				ID: "huangzhong", Name: "黄忠", Faction: "shu", Enabled: true,
				BonusTrait: GeneralTraitConfig{
					TraitID: "laodang_yizhuang", TraitType: general.TraitTypeBonus, Enabled: true,
					Scope: "enemy_army", Params: map[string]float64{"effectRate": 0.1},
				},
			},
		}})
		svc, repo, attacker, defender := newPvpTestServiceForGenerals(t, "wu", "huanggai", "shu", "huangzhong")
		attacker.Army = []ArmyUnit{{UnitType: "wuInfantry", Amount: 1000}}
		defender.Army = []ArmyUnit{{UnitType: "shuInfantry", Amount: 1000}}
		defender.Buildings = nil
		repo.players[attacker.Player.ID] = attacker
		repo.players[defender.Player.ID] = defender

		started, err := svc.StartPvpAttack(PvpAttackRequest{
			PlayerID: attacker.Player.ID, TargetPlayerID: defender.Player.ID, MarchMode: PvpMarchTypePlunder,
			Troops: map[string]int{"wuInfantry": 1000}, GeneralIDs: []string{"huanggai"},
		})
		if err != nil {
			t.Fatalf("StartPvpAttack failed: %v", err)
		}
		forcePvpMarchDue(t, repo, started.March.ID)
		battle, err := svc.ResolvePvpMarch(started.March.ID)
		if err != nil {
			t.Fatalf("ResolvePvpMarch failed: %v", err)
		}
		if battle.Result["attackerPower"] != float64(10000) || battle.Result["defensePower"] != float64(10000) {
			t.Fatalf("expected after-combat traits to preserve 10000/10000 core powers, result=%+v", battle.Result)
		}
		attackerLosses := pvpTestLossesFromBattle(t, battle, "attacker")
		defenderLosses := pvpTestLossesFromBattle(t, battle, "defender")
		if attackerLosses["wuInfantry"] != 600 || defenderLosses["shuInfantry"] != 600 {
			t.Fatalf("expected core 500/500 plus 100 counter losses on each side, attacker=%+v defender=%+v", attackerLosses, defenderLosses)
		}

		attackerReports, _, err := repo.ListReports(attacker.Player.ID, 10, 0)
		if err != nil || len(attackerReports) != 1 {
			t.Fatalf("expected one attacker report, reports=%+v err=%v", attackerReports, err)
		}
		defenderReports, _, err := repo.ListReports(defender.Player.ID, 10, 0)
		if err != nil || len(defenderReports) != 1 {
			t.Fatalf("expected one defender report, reports=%+v err=%v", defenderReports, err)
		}
		for _, report := range []BattleReport{attackerReports[0], defenderReports[0]} {
			if len(report.TraitTriggered) != 2 || report.TraitTriggered[0] != "kurou_fanji" || report.TraitTriggered[1] != "laodang_yizhuang" || len(report.TraitOutcomes) != 2 {
				t.Fatalf("expected both unsuppressed follow-up damages in legacy timeline, report=%s triggered=%+v outcomes=%+v", report.ID, report.TraitTriggered, report.TraitOutcomes)
			}
			if _, exists := report.TraitOutcomes["kurouji"]; exists {
				t.Fatalf("expected missed Kurouji absent from legacy outcomes, report=%s outcomes=%+v", report.ID, report.TraitOutcomes)
			}
			counterExtra, counterOK := report.TraitOutcomes["kurou_fanji"].Detail["extraLosses"].(map[string]int)
			laodangExtra, laodangOK := report.TraitOutcomes["laodang_yizhuang"].Detail["extraLosses"].(map[string]int)
			if !counterOK || !laodangOK || counterExtra["shuInfantry"] != 100 || laodangExtra["wuInfantry"] != 100 {
				t.Fatalf("expected both follow-up traits to add exact 100 losses, report=%s outcomes=%+v", report.ID, report.TraitOutcomes)
			}
			if len(report.PvpAttackerGenerals) != 1 || !pvpSnapshotHasTrait(report.PvpAttackerGenerals[0], "kurouji") || !pvpSnapshotHasTrait(report.PvpAttackerGenerals[0], "kurou_fanji") {
				t.Fatalf("expected Huang Gai snapshot to retain both owned traits, report=%s generals=%+v", report.ID, report.PvpAttackerGenerals)
			}
			if len(report.PvpDefenderGenerals) != 1 || !pvpSnapshotHasTrait(report.PvpDefenderGenerals[0], "laodang_yizhuang") {
				t.Fatalf("expected Huang Zhong snapshot to retain Laodang, report=%s generals=%+v", report.ID, report.PvpDefenderGenerals)
			}
			if report.Detail == nil || report.Detail.SecondarySide == nil || len(report.Detail.Traits) != 2 || report.Detail.Traits[0].TraitID != "kurou_fanji" || report.Detail.Traits[0].OwnerRole != "attacker" || report.Detail.Traits[1].TraitID != "laodang_yizhuang" || report.Detail.Traits[1].OwnerRole != "defender" {
				t.Fatalf("expected standard timeline to preserve attacker counter then defender follow-up, report=%s detail=%+v", report.ID, report.Detail)
			}
			if !standardDetailGeneralHasTrait(report.Detail, "kurouji") || !standardDetailGeneralHasTrait(report.Detail, "kurou_fanji") || !standardDetailGeneralHasTrait(report.Detail, "laodang_yizhuang") {
				t.Fatalf("expected standard general snapshots to retain all owned traits, report=%s detail=%+v", report.ID, report.Detail)
			}
			attackerUnit := attackDefenseCrossReportUnit(t, report.Detail.PrimarySide, "wuInfantry")
			defenderUnit := attackDefenseCrossReportUnit(t, *report.Detail.SecondarySide, "shuInfantry")
			if attackerUnit.AmountBefore != 1000 || attackerUnit.Lost != 600 || attackerUnit.Survived != 400 || defenderUnit.AmountBefore != 1000 || defenderUnit.Lost != 600 || defenderUnit.Survived != 400 {
				t.Fatalf("expected standard rows 1000/600/400 on both sides, report=%s attacker=%+v defender=%+v", report.ID, attackerUnit, defenderUnit)
			}
		}

		storedMarch, marchErr := repo.GetPvpMarch(started.March.ID)
		storedAttacker, attackerErr := repo.GetState(attacker.Player.ID)
		storedDefender, defenderErr := repo.GetState(defender.Player.ID)
		if marchErr != nil || attackerErr != nil || defenderErr != nil {
			t.Fatalf("expected stored battle state, march=%v attacker=%v defender=%v", marchErr, attackerErr, defenderErr)
		}
		if storedMarch.AttackTroops["wuInfantry"] != 400 || armySliceToMap(storedDefender.Army)["shuInfantry"] != 400 {
			t.Fatalf("expected authoritative survivors 400/400, march=%+v defender=%+v", storedMarch, storedDefender.Army)
		}
		if pvpTestGeneralExp(storedAttacker, "huanggai") != 600 || attackerReports[0].GeneralExpGained != 600 || pvpTestGeneralExp(storedDefender, "huangzhong") != 600 || defenderReports[0].GeneralExpGained != 600 {
			t.Fatalf("expected exp 600/600 from final real infantry deaths, attacker=%d/%d defender=%d/%d", pvpTestGeneralExp(storedAttacker, "huanggai"), attackerReports[0].GeneralExpGained, pvpTestGeneralExp(storedDefender, "huangzhong"), defenderReports[0].GeneralExpGained)
		}
	})
}

// assertRandomMissPvpReports 核对随机特性未命中时双方只保留拥有快照，不生成任何触发结果。
func assertRandomMissPvpReports(t *testing.T, repo *MemoryRepository, battle PvpBattle, attackerPlayerID string, defenderPlayerID string, generalID string, traitID string) {
	t.Helper()
	attackerReports, _, err := repo.ListReports(attackerPlayerID, 10, 0)
	if err != nil || len(attackerReports) != 1 || attackerReports[0].ID != battle.AttackerReportID {
		t.Fatalf("expected one attacker report, reports=%+v err=%v", attackerReports, err)
	}
	defenderReports, _, err := repo.ListReports(defenderPlayerID, 10, 0)
	if err != nil || len(defenderReports) != 1 || defenderReports[0].ID != battle.DefenderReportID {
		t.Fatalf("expected one defender report, reports=%+v err=%v", defenderReports, err)
	}
	for _, report := range []BattleReport{attackerReports[0], defenderReports[0]} {
		if len(report.TraitTriggered) != 0 || len(report.TraitOutcomes) != 0 || report.Detail == nil || len(report.Detail.Traits) != 0 {
			t.Fatalf("expected missed %s absent from all timelines, report=%+v", traitID, report)
		}
		if len(report.PvpAttackerGenerals) != 1 || report.PvpAttackerGenerals[0].ID != generalID || !pvpSnapshotHasTrait(report.PvpAttackerGenerals[0], traitID) {
			t.Fatalf("expected snapshot to retain owned %s without triggering, snapshots=%+v", traitID, report.PvpAttackerGenerals)
		}
	}
}

// assertRandomMissDefenderPvpReports 核对防守随机特性未命中时只保留防守将领拥有快照，不生成触发结果。
func assertRandomMissDefenderPvpReports(t *testing.T, repo *MemoryRepository, battle PvpBattle, attackerPlayerID string, defenderPlayerID string, generalID string, traitID string) {
	t.Helper()
	attackerReports, _, err := repo.ListReports(attackerPlayerID, 10, 0)
	if err != nil || len(attackerReports) != 1 || attackerReports[0].ID != battle.AttackerReportID {
		t.Fatalf("expected one attacker report, reports=%+v err=%v", attackerReports, err)
	}
	defenderReports, _, err := repo.ListReports(defenderPlayerID, 10, 0)
	if err != nil || len(defenderReports) != 1 || defenderReports[0].ID != battle.DefenderReportID {
		t.Fatalf("expected one defender report, reports=%+v err=%v", defenderReports, err)
	}
	for _, report := range []BattleReport{attackerReports[0], defenderReports[0]} {
		if len(report.TraitTriggered) != 0 || len(report.TraitOutcomes) != 0 || report.Detail == nil || len(report.Detail.Traits) != 0 {
			t.Fatalf("expected missed %s absent from all timelines, report=%+v", traitID, report)
		}
		if len(report.PvpDefenderGenerals) != 1 || report.PvpDefenderGenerals[0].ID != generalID || !pvpSnapshotHasTrait(report.PvpDefenderGenerals[0], traitID) {
			t.Fatalf("expected defender snapshot to retain owned %s without triggering, snapshots=%+v", traitID, report.PvpDefenderGenerals)
		}
		if report.Detail.SecondarySide == nil || len(report.Detail.SecondarySide.Generals) != 1 || report.Detail.SecondarySide.Generals[0].ID != generalID {
			t.Fatalf("expected standard defender snapshot without trait outcome, report=%+v", report.Detail)
		}
	}
}
