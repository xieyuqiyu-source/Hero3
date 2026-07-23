// 本文件以真实 PVP 事务锁定张辽双概率特性的组合、兵损、库存和战报。
package game

import (
	"reflect"
	"testing"

	"hero3/internal/core/general"
)

// TestPvpZhangLiaoTraitProbabilityMatrix 验证两项特性独立判定并覆盖四种命中组合。
func TestPvpZhangLiaoTraitProbabilityMatrix(t *testing.T) {
	tests := []struct {
		name              string
		fleeChance        float64
		attackChance      float64
		wantFlee          bool
		wantAttack        bool
		wantAttackPower   float64
		wantDefensePower  float64
		wantAttackLosses  int
		wantDefenseLosses int
	}{
		{name: "两项同时命中", fleeChance: 1, attackChance: 1, wantFlee: true, wantAttack: true, wantAttackPower: 19000, wantDefensePower: 6120, wantAttackLosses: 199, wantDefenseLosses: 750},
		{name: "仅震慑全军命中", fleeChance: 1, attackChance: 0, wantFlee: true, wantAttackPower: 14000, wantDefensePower: 6120, wantAttackLosses: 308, wantDefenseLosses: 750},
		{name: "仅威震逍遥命中", fleeChance: 0, attackChance: 1, wantAttack: true, wantAttackPower: 19000, wantDefensePower: 8160, wantAttackLosses: 300, wantDefenseLosses: 1000},
		{name: "两项同时未命中", fleeChance: 0, attackChance: 0, wantAttackPower: 14000, wantDefensePower: 8160, wantAttackLosses: 464, wantDefenseLosses: 1000},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			setTestFactionsAndGenerals(t, FactionsConfig{
				"wei": {Name: "魏国", Generals: []GeneralInfo{{ID: "zhangliao", Name: "张辽"}}},
				"shu": {Name: "蜀国", Generals: []GeneralInfo{{ID: "liubei", Name: "刘备"}}},
			}, GeneralsConfig{Enabled: true, Heroes: map[string]GeneralHeroConfig{
				"zhangliao": {
					ID: "zhangliao", Name: "张辽", Faction: "wei", Enabled: true,
					SpecialTrait: GeneralTraitConfig{
						TraitID: "weizhen_zhenhe", TraitType: general.TraitTypeSpecial, Enabled: true,
						Scope: "enemy_army", AllowedSides: []string{"attacker"},
						Params: map[string]float64{"triggerChance": tc.fleeChance, "effectRate": 0.25},
					},
					BonusTrait: GeneralTraitConfig{
						TraitID: "weizhen_xiaoyao", TraitType: general.TraitTypeBonus, Enabled: true,
						Scope: "self_army", TargetUnitType: "cavalry", AllowedSides: []string{"attacker"},
						Params: map[string]float64{"triggerChance": tc.attackChance, "attackBonusRate": 0.35},
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
			if battle.Result["attackerPower"] != tc.wantAttackPower || battle.Result["defensePower"] != tc.wantDefensePower {
				t.Fatalf("expected power %.0f/%.0f, got %+v", tc.wantAttackPower, tc.wantDefensePower, battle.Result)
			}

			attackerReports, _, _ := repo.ListReports(attacker.Player.ID, 10, 0)
			defenderReports, _, _ := repo.ListReports(defender.Player.ID, 10, 0)
			if len(attackerReports) != 1 || len(defenderReports) != 1 {
				t.Fatalf("expected one report for each side, attacker=%+v defender=%+v", attackerReports, defenderReports)
			}
			var wantTimeline []string
			if tc.wantFlee {
				wantTimeline = append(wantTimeline, "weizhen_zhenhe")
			}
			if tc.wantAttack {
				wantTimeline = append(wantTimeline, "weizhen_xiaoyao")
			}
			for _, report := range []BattleReport{attackerReports[0], defenderReports[0]} {
				if len(report.TraitTriggered) != len(wantTimeline) || (len(wantTimeline) > 0 && !reflect.DeepEqual(report.TraitTriggered, wantTimeline)) || report.Detail == nil || len(report.Detail.Traits) != len(wantTimeline) {
					t.Fatalf("expected timeline %v, report=%+v", wantTimeline, report)
				}
				_, fleeExists := report.TraitOutcomes["weizhen_zhenhe"]
				_, attackExists := report.TraitOutcomes["weizhen_xiaoyao"]
				if fleeExists != tc.wantFlee || attackExists != tc.wantAttack {
					t.Fatalf("expected independent outcomes flee=%t attack=%t, got %+v", tc.wantFlee, tc.wantAttack, report.TraitOutcomes)
				}
				if tc.wantFlee {
					outcome := report.TraitOutcomes["weizhen_zhenhe"]
					fled, fledOK := outcome.Detail["fledUnits"].(map[string]int)
					returned, returnedOK := outcome.Detail["returnedUnits"].(map[string]int)
					if !fledOK || !returnedOK || fled["shuInfantry"] != 250 || returned["shuInfantry"] != 250 || outcome.Detail["triggerChance"] != tc.fleeChance {
						t.Fatalf("expected authoritative 250 flee/return result, outcome=%+v", outcome)
					}
				}
				if tc.wantAttack {
					outcome := report.TraitOutcomes["weizhen_xiaoyao"]
					modified, ok := outcome.Detail["attackModifiedUnits"].(map[string]int)
					if !ok || modified["weiCavalry"] != 5 || outcome.Detail["attackBonusRate"] != 0.35 || outcome.Detail["triggerChance"] != tc.attackChance {
						t.Fatalf("expected authoritative cavalry +5 result, outcome=%+v", outcome)
					}
				}
			}

			attackerLosses := pvpTestLossesFromBattle(t, battle, "attacker")["weiCavalry"]
			defenderLosses := pvpTestLossesFromBattle(t, battle, "defender")["shuInfantry"]
			if attackerLosses != tc.wantAttackLosses || defenderLosses != tc.wantDefenseLosses {
				t.Fatalf("expected exact losses %d/%d, got %d/%d", tc.wantAttackLosses, tc.wantDefenseLosses, attackerLosses, defenderLosses)
			}
			storedDefender, err := repo.GetState(defender.Player.ID)
			if err != nil {
				t.Fatalf("GetState defender failed: %v", err)
			}
			storedMarch, err := repo.GetPvpMarch(started.March.ID)
			if err != nil {
				t.Fatalf("GetPvpMarch failed: %v", err)
			}
			if armySliceToMap(storedDefender.Army)["shuInfantry"] != 1000-defenderLosses || storedMarch.AttackTroops["weiCavalry"] != 1000-attackerLosses {
				t.Fatalf("expected authoritative inventory to match actual deaths, battle=%+v defender=%+v march=%+v", battle.Losses, storedDefender.Army, storedMarch)
			}
			if tc.wantFlee && (defenderLosses > 750 || armySliceToMap(storedDefender.Army)["shuInfantry"] < 250) {
				t.Fatalf("fled troops must not die, losses=%d remaining=%+v", defenderLosses, storedDefender.Army)
			}
		})
	}
}
