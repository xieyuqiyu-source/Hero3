// 本文件验证防守方随机特性合法未命中时不会吞掉后续独立效果。
package game

import (
	"reflect"
	"testing"

	"hero3/internal/core/general"
)

// TestPvpDefenderYibingMissKeepsMoudingDefenseBonus 验证防守司马懿疑兵未命中后谋定仍独立加防。
func TestPvpDefenderYibingMissKeepsMoudingDefenseBonus(t *testing.T) {
	setTestFactionsAndGenerals(t, FactionsConfig{
		"shu": {Name: "蜀国", Generals: []GeneralInfo{{ID: "liubei", Name: "刘备"}}},
		"wei": {Name: "魏国", Generals: []GeneralInfo{{ID: "simayi", Name: "司马懿"}}},
	}, GeneralsConfig{Enabled: true, Heroes: map[string]GeneralHeroConfig{
		"liubei": {ID: "liubei", Name: "刘备", Faction: "shu", Enabled: true},
		"simayi": {
			ID: "simayi", Name: "司马懿", Faction: "wei", Enabled: true,
			SpecialTrait: GeneralTraitConfig{
				TraitID: "yibing_touxi", TraitType: general.TraitTypeSpecial, Enabled: true,
				Scope: "enemy_army", Params: map[string]float64{"triggerChance": 0, "effectRate": 0.35},
			},
			BonusTrait: GeneralTraitConfig{
				TraitID: "mouding_houfa", TraitType: general.TraitTypeBonus, Enabled: true,
				Scope: "self_army", AllowedSides: []string{"defender", "reinforcement"}, Params: map[string]float64{"defenseBonusRate": 0.35, "triggerChance": 1},
			},
		},
	}})
	svc, repo, attacker, defender := newPvpTestServiceForGenerals(t, "shu", "liubei", "wei", "simayi")
	attacker.Army = []ArmyUnit{{UnitType: "shuInfantry", Amount: 1000}}
	defender.Army = []ArmyUnit{{UnitType: "weiInfantry", Amount: 1000}}
	defender.Buildings = nil
	repo.players[attacker.Player.ID] = attacker
	repo.players[defender.Player.ID] = defender

	started, err := svc.StartPvpAttack(PvpAttackRequest{
		PlayerID: attacker.Player.ID, TargetPlayerID: defender.Player.ID, MarchMode: PvpMarchTypePlunder,
		Troops: map[string]int{"shuInfantry": 1000}, GeneralIDs: []string{"liubei"},
	})
	if err != nil {
		t.Fatalf("StartPvpAttack failed: %v", err)
	}
	forcePvpMarchDue(t, repo, started.March.ID)
	battle, err := svc.ResolvePvpMarch(started.March.ID)
	if err != nil {
		t.Fatalf("ResolvePvpMarch failed: %v", err)
	}
	attackerLosses := pvpTestLossesFromBattle(t, battle, "attacker")
	defenderLosses := pvpTestLossesFromBattle(t, battle, "defender")
	if battle.Result["winner"] != "defender" || battle.Result["attackerPower"] != float64(10000) || battle.Result["defensePower"] != float64(14000) || attackerLosses["shuInfantry"] != 617 || defenderLosses["weiInfantry"] != 382 {
		t.Fatalf("expected missed ambush plus active 35%% defender defense bonus to produce exact result, battle=%+v", battle)
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
		if !reflect.DeepEqual(report.TraitTriggered, []string{"mouding_houfa"}) || len(report.TraitOutcomes) != 1 {
			t.Fatalf("expected only defender Mouding timeline, report=%s triggered=%+v outcomes=%+v", report.ID, report.TraitTriggered, report.TraitOutcomes)
		}
		if _, exists := report.TraitOutcomes["yibing_touxi"]; exists || standardReportHasTrait(report.Detail, "yibing_touxi") {
			t.Fatalf("expected missed defender Yibing absent from trigger timeline, report=%+v", report)
		}
		outcome := report.TraitOutcomes["mouding_houfa"]
		infantry, infantryOK := outcome.Detail["infantryDefenseModifiedUnits"].(map[string]int)
		cavalry, cavalryOK := outcome.Detail["cavalryDefenseModifiedUnits"].(map[string]int)
		if !infantryOK || !cavalryOK || outcome.OwnerSide != "defender" || outcome.Detail["defenseBonusRate"] != 0.35 || infantry["weiInfantry"] != 4 || cavalry["weiInfantry"] != 3 {
			t.Fatalf("expected exact defender defense bonus, report=%s outcome=%+v", report.ID, outcome)
		}
		if report.Detail == nil || report.Detail.SecondarySide == nil || len(report.Detail.Traits) != 1 || report.Detail.Traits[0].TraitID != "mouding_houfa" || report.Detail.Traits[0].OwnerRole != "defender" {
			t.Fatalf("expected one defender standard timeline entry, report=%s detail=%+v", report.ID, report.Detail)
		}
		if !standardDetailGeneralHasTrait(report.Detail, "yibing_touxi") || !standardDetailGeneralHasTrait(report.Detail, "mouding_houfa") {
			t.Fatalf("expected defender snapshot to retain both owned traits, report=%s detail=%+v", report.ID, report.Detail)
		}
		attackerUnit := attackDefenseCrossReportUnit(t, report.Detail.PrimarySide, "shuInfantry")
		defenderUnit := attackDefenseCrossReportUnit(t, *report.Detail.SecondarySide, "weiInfantry")
		if attackerUnit.AmountBefore != 1000 || attackerUnit.Lost != 617 || attackerUnit.Survived != 383 || defenderUnit.AmountBefore != 1000 || defenderUnit.Lost != 382 || defenderUnit.Survived != 618 {
			t.Fatalf("expected exact standard rows after defender bonus, report=%s rows=%+v/%+v", report.ID, attackerUnit, defenderUnit)
		}
	}

	storedMarch, marchErr := repo.GetPvpMarch(started.March.ID)
	storedAttacker, attackerErr := repo.GetState(attacker.Player.ID)
	storedDefender, defenderErr := repo.GetState(defender.Player.ID)
	if marchErr != nil || attackerErr != nil || defenderErr != nil {
		t.Fatalf("expected stored battle state, march=%v attacker=%v defender=%v", marchErr, attackerErr, defenderErr)
	}
	if storedMarch.AttackTroops["shuInfantry"] != 383 || armySliceToMap(storedDefender.Army)["weiInfantry"] != 618 {
		t.Fatalf("expected authoritative survivors 383/618, march=%+v defender=%+v", storedMarch, storedDefender.Army)
	}
	if pvpTestGeneralExp(storedAttacker, "liubei") != 382 || attackerReports[0].GeneralExpGained != 382 || pvpTestGeneralExp(storedDefender, "simayi") != 617 || defenderReports[0].GeneralExpGained != 617 {
		t.Fatalf("expected generals to gain exact real-death experience, states=%+v/%+v reports=%+v/%+v", storedAttacker.Generals, storedDefender.Generals, attackerReports[0], defenderReports[0])
	}
}

// TestPvpDefenderHuoshaoMissKeepsLianyingDamage 验证防守陆逊火烧未命中后连营增伤仍独立扣兵。
func TestPvpDefenderHuoshaoMissKeepsLianyingDamage(t *testing.T) {
	setTestFactionsAndGenerals(t, FactionsConfig{
		"shu": {Name: "蜀国", Generals: []GeneralInfo{{ID: "liubei", Name: "刘备"}}},
		"wu":  {Name: "吴国", Generals: []GeneralInfo{{ID: "luxun", Name: "陆逊"}}},
	}, GeneralsConfig{Enabled: true, Heroes: map[string]GeneralHeroConfig{
		"liubei": {ID: "liubei", Name: "刘备", Faction: "shu", Enabled: true},
		"luxun": {
			ID: "luxun", Name: "陆逊", Faction: "wu", Enabled: true,
			SpecialTrait: GeneralTraitConfig{
				TraitID: "huoshao_lianying", TraitType: general.TraitTypeSpecial, Enabled: true,
				Scope: "enemy_army", TargetUnitType: "infantry",
				Params: map[string]float64{"triggerChance": 0, "effectRate": 1, "maxAffectedRate": 1},
			},
			BonusTrait: GeneralTraitConfig{
				TraitID: "lianying_zengshang", TraitType: general.TraitTypeBonus, Enabled: true,
				Scope: "enemy_army", TargetUnitType: "infantry", Params: map[string]float64{"effectRate": 0.1},
			},
		},
	}})
	svc, repo, attacker, defender := newPvpTestServiceForGenerals(t, "shu", "liubei", "wu", "luxun")
	attacker.Army = []ArmyUnit{{UnitType: "shuInfantry", Amount: 1000}}
	defender.Army = []ArmyUnit{{UnitType: "wuInfantry", Amount: 1000}}
	defender.Buildings = nil
	repo.players[attacker.Player.ID] = attacker
	repo.players[defender.Player.ID] = defender

	started, err := svc.StartPvpAttack(PvpAttackRequest{
		PlayerID: attacker.Player.ID, TargetPlayerID: defender.Player.ID, MarchMode: PvpMarchTypePlunder,
		Troops: map[string]int{"shuInfantry": 1000}, GeneralIDs: []string{"liubei"},
	})
	if err != nil {
		t.Fatalf("StartPvpAttack failed: %v", err)
	}
	forcePvpMarchDue(t, repo, started.March.ID)
	battle, err := svc.ResolvePvpMarch(started.March.ID)
	if err != nil {
		t.Fatalf("ResolvePvpMarch failed: %v", err)
	}
	attackerLosses := pvpTestLossesFromBattle(t, battle, "attacker")
	defenderLosses := pvpTestLossesFromBattle(t, battle, "defender")
	if battle.Result["winner"] != "draw" || battle.Result["attackerPower"] != float64(10000) || battle.Result["defensePower"] != float64(10000) || attackerLosses["shuInfantry"] != 600 || defenderLosses["wuInfantry"] != 500 {
		t.Fatalf("expected equal-power draw plus only defender Lianying damage, battle=%+v", battle)
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
		if !reflect.DeepEqual(report.TraitTriggered, []string{"lianying_zengshang"}) || len(report.TraitOutcomes) != 1 {
			t.Fatalf("expected only defender Lianying timeline, report=%s triggered=%+v outcomes=%+v", report.ID, report.TraitTriggered, report.TraitOutcomes)
		}
		if _, exists := report.TraitOutcomes["huoshao_lianying"]; exists || standardReportHasTrait(report.Detail, "huoshao_lianying") {
			t.Fatalf("expected missed defender Huoshao absent from trigger timeline, report=%+v", report)
		}
		outcome := report.TraitOutcomes["lianying_zengshang"]
		extra, ok := outcome.Detail["targetExtraLosses"].(map[string]int)
		if !ok || outcome.OwnerSide != "defender" || outcome.Detail["effectRate"] != 0.1 || extra["shuInfantry"] != 100 {
			t.Fatalf("expected exact defender Lianying damage, report=%s outcome=%+v", report.ID, outcome)
		}
		if report.Detail == nil || report.Detail.SecondarySide == nil || len(report.Detail.Traits) != 1 || report.Detail.Traits[0].TraitID != "lianying_zengshang" || report.Detail.Traits[0].OwnerRole != "defender" {
			t.Fatalf("expected one defender standard timeline entry, report=%s detail=%+v", report.ID, report.Detail)
		}
		if !standardDetailGeneralHasTrait(report.Detail, "huoshao_lianying") || !standardDetailGeneralHasTrait(report.Detail, "lianying_zengshang") {
			t.Fatalf("expected defender snapshot to retain both owned traits, report=%s detail=%+v", report.ID, report.Detail)
		}
		attackerUnit := attackDefenseCrossReportUnit(t, report.Detail.PrimarySide, "shuInfantry")
		defenderUnit := attackDefenseCrossReportUnit(t, *report.Detail.SecondarySide, "wuInfantry")
		if attackerUnit.AmountBefore != 1000 || attackerUnit.Lost != 600 || attackerUnit.Survived != 400 || defenderUnit.AmountBefore != 1000 || defenderUnit.Lost != 500 || defenderUnit.Survived != 500 {
			t.Fatalf("expected exact standard rows after defender Lianying, report=%s rows=%+v/%+v", report.ID, attackerUnit, defenderUnit)
		}
	}

	storedMarch, marchErr := repo.GetPvpMarch(started.March.ID)
	storedAttacker, attackerErr := repo.GetState(attacker.Player.ID)
	storedDefender, defenderErr := repo.GetState(defender.Player.ID)
	if marchErr != nil || attackerErr != nil || defenderErr != nil {
		t.Fatalf("expected stored battle state, march=%v attacker=%v defender=%v", marchErr, attackerErr, defenderErr)
	}
	if storedMarch.AttackTroops["shuInfantry"] != 400 || armySliceToMap(storedDefender.Army)["wuInfantry"] != 500 {
		t.Fatalf("expected authoritative survivors 400/500, march=%+v defender=%+v", storedMarch, storedDefender.Army)
	}
	if pvpTestGeneralExp(storedAttacker, "liubei") != 500 || attackerReports[0].GeneralExpGained != 500 || pvpTestGeneralExp(storedDefender, "luxun") != 600 || defenderReports[0].GeneralExpGained != 600 {
		t.Fatalf("expected generals to gain exact real-death experience, states=%+v/%+v reports=%+v/%+v", storedAttacker.Generals, storedDefender.Generals, attackerReports[0], defenderReports[0])
	}
}

// TestPvpDefenderRandomPreBattleMissesDoNotEnableAttackerOnlyBonuses 验证随机未命中或张辽方向不符时不会越界启用主动进攻特性。
func TestPvpDefenderRandomPreBattleMissesDoNotEnableAttackerOnlyBonuses(t *testing.T) {
	tests := []struct {
		name              string
		defenderFaction   string
		defenderUnit      string
		attackerFaction   string
		attackerUnit      string
		generalID         string
		generalName       string
		specialTraitID    string
		specialEffectRate float64
		specialChance     float64
		specialSides      []string
		bonusTraitID      string
		bonusTarget       string
		bonusAttackRate   float64
	}{
		{name: "关羽水淹七军", defenderFaction: "shu", defenderUnit: "shuInfantry", attackerFaction: "wei", attackerUnit: "weiInfantry", generalID: "guanyu", generalName: "关羽", specialTraitID: "shuiyan_qijun", specialEffectRate: 0.35, bonusTraitID: "wusheng_pojun", bonusAttackRate: 0.2},
		{name: "张辽主动进攻双特性方向隔离", defenderFaction: "wei", defenderUnit: "weiInfantry", attackerFaction: "shu", attackerUnit: "shuInfantry", generalID: "zhangliao", generalName: "张辽", specialTraitID: "weizhen_zhenhe", specialEffectRate: 0.25, specialChance: 1, specialSides: []string{"attacker"}, bonusTraitID: "weizhen_xiaoyao", bonusTarget: "cavalry", bonusAttackRate: 0.35},
		{name: "张飞震慑全军", defenderFaction: "shu", defenderUnit: "shuInfantry", attackerFaction: "wei", attackerUnit: "weiInfantry", generalID: "zhangfei", generalName: "张飞", specialTraitID: "zhenhe_quanjun", specialEffectRate: 0.5, bonusTraitID: "wanren_nuhou", bonusTarget: "infantry", bonusAttackRate: 0.2},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			setTestFactionsAndGenerals(t, FactionsConfig{
				tc.attackerFaction: {Name: "进攻阵营", Generals: []GeneralInfo{{ID: "opponent", Name: "对手"}}},
				tc.defenderFaction: {Name: "防守阵营", Generals: []GeneralInfo{{ID: tc.generalID, Name: tc.generalName}}},
			}, GeneralsConfig{Enabled: true, Heroes: map[string]GeneralHeroConfig{
				"opponent": {ID: "opponent", Name: "对手", Faction: tc.attackerFaction, Enabled: true},
				tc.generalID: {
					ID: tc.generalID, Name: tc.generalName, Faction: tc.defenderFaction, Enabled: true,
					SpecialTrait: GeneralTraitConfig{
						TraitID: tc.specialTraitID, TraitType: general.TraitTypeSpecial, Enabled: true,
						Scope: "enemy_army", AllowedSides: tc.specialSides, Params: map[string]float64{"triggerChance": tc.specialChance, "effectRate": tc.specialEffectRate},
					},
					BonusTrait: GeneralTraitConfig{
						TraitID: tc.bonusTraitID, TraitType: general.TraitTypeBonus, Enabled: true,
						Scope: "self_army", TargetUnitType: tc.bonusTarget, AllowedSides: []string{"attacker"},
						Params: map[string]float64{"triggerChance": 1, "attackBonusRate": tc.bonusAttackRate},
					},
				},
			}})
			svc, repo, attacker, defender := newPvpTestServiceForGenerals(t, tc.attackerFaction, "opponent", tc.defenderFaction, tc.generalID)
			attacker.Army = []ArmyUnit{{UnitType: tc.attackerUnit, Amount: 1000}}
			defender.Army = []ArmyUnit{{UnitType: tc.defenderUnit, Amount: 1000}}
			defender.Buildings = nil
			repo.players[attacker.Player.ID] = attacker
			repo.players[defender.Player.ID] = defender

			started, err := svc.StartPvpAttack(PvpAttackRequest{
				PlayerID: attacker.Player.ID, TargetPlayerID: defender.Player.ID, MarchMode: PvpMarchTypePlunder,
				Troops: map[string]int{tc.attackerUnit: 1000}, GeneralIDs: []string{"opponent"},
			})
			if err != nil {
				t.Fatalf("StartPvpAttack failed: %v", err)
			}
			forcePvpMarchDue(t, repo, started.March.ID)
			battle, err := svc.ResolvePvpMarch(started.March.ID)
			if err != nil {
				t.Fatalf("ResolvePvpMarch failed: %v", err)
			}
			attackerLosses := pvpTestLossesFromBattle(t, battle, "attacker")
			defenderLosses := pvpTestLossesFromBattle(t, battle, "defender")
			if battle.Result["winner"] != "draw" || battle.Result["attackerPower"] != float64(10000) || battle.Result["defensePower"] != float64(10000) || attackerLosses[tc.attackerUnit] != 500 || defenderLosses[tc.defenderUnit] != 500 {
				t.Fatalf("expected untouched equal-power baseline when defender random trait misses, battle=%+v", battle)
			}
			assertRandomMissDefenderPvpReports(t, repo, battle, attacker.Player.ID, defender.Player.ID, tc.generalID, tc.specialTraitID)

			attackerReports, _, attackerReportErr := repo.ListReports(attacker.Player.ID, 10, 0)
			defenderReports, _, defenderReportErr := repo.ListReports(defender.Player.ID, 10, 0)
			if attackerReportErr != nil || defenderReportErr != nil || len(attackerReports) != 1 || len(defenderReports) != 1 {
				t.Fatalf("expected one report per side, attacker=%+v/%v defender=%+v/%v", attackerReports, attackerReportErr, defenderReports, defenderReportErr)
			}
			for _, report := range []BattleReport{attackerReports[0], defenderReports[0]} {
				if len(report.PvpDefenderGenerals) != 1 || !pvpSnapshotHasTrait(report.PvpDefenderGenerals[0], tc.bonusTraitID) || !standardDetailGeneralHasTrait(report.Detail, tc.specialTraitID) || !standardDetailGeneralHasTrait(report.Detail, tc.bonusTraitID) {
					t.Fatalf("expected defender snapshots to retain both owned traits, report=%+v", report)
				}
				attackerUnit := attackDefenseCrossReportUnit(t, report.Detail.PrimarySide, tc.attackerUnit)
				defenderUnit := attackDefenseCrossReportUnit(t, *report.Detail.SecondarySide, tc.defenderUnit)
				if attackerUnit.AmountBefore != 1000 || attackerUnit.Lost != 500 || attackerUnit.Survived != 500 || defenderUnit.AmountBefore != 1000 || defenderUnit.Lost != 500 || defenderUnit.Survived != 500 {
					t.Fatalf("expected exact baseline standard rows, report=%s rows=%+v/%+v", report.ID, attackerUnit, defenderUnit)
				}
			}

			storedMarch, marchErr := repo.GetPvpMarch(started.March.ID)
			storedAttacker, attackerErr := repo.GetState(attacker.Player.ID)
			storedDefender, defenderErr := repo.GetState(defender.Player.ID)
			if marchErr != nil || attackerErr != nil || defenderErr != nil {
				t.Fatalf("expected stored battle state, march=%v attacker=%v defender=%v", marchErr, attackerErr, defenderErr)
			}
			if storedMarch.AttackTroops[tc.attackerUnit] != 500 || armySliceToMap(storedDefender.Army)[tc.defenderUnit] != 500 {
				t.Fatalf("expected authoritative survivors 500/500, march=%+v defender=%+v", storedMarch, storedDefender.Army)
			}
			if pvpTestGeneralExp(storedAttacker, "opponent") != 500 || attackerReports[0].GeneralExpGained != 500 || pvpTestGeneralExp(storedDefender, tc.generalID) != 500 || defenderReports[0].GeneralExpGained != 500 {
				t.Fatalf("expected both generals to gain baseline 500 experience, states=%+v/%+v reports=%+v/%+v", storedAttacker.Generals, storedDefender.Generals, attackerReports[0], defenderReports[0])
			}
		})
	}
}
