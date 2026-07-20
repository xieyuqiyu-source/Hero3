// 本文件验证孙策小霸王追击作为防守主将时的场景与胜负边界。
package game

import (
	"testing"

	"hero3/internal/core/general"
)

// TestPvpDefenderSunCePursuitRequiresPlunderVictory 验证孙策守城只在掠夺胜利后追击来袭方。
func TestPvpDefenderSunCePursuitRequiresPlunderVictory(t *testing.T) {
	cases := []struct {
		name          string
		marchMode     string
		attackerCount int
		defenderCount int
		wantWinner    string
		wantTriggered bool
	}{
		{name: "掠夺守城获胜触发", marchMode: PvpMarchTypePlunder, attackerCount: 100, defenderCount: 200, wantWinner: "defender", wantTriggered: true},
		{name: "普通守城获胜不触发", marchMode: PvpMarchTypeAttack, attackerCount: 100, defenderCount: 200, wantWinner: "defender", wantTriggered: false},
		{name: "掠夺守城战败不触发", marchMode: PvpMarchTypePlunder, attackerCount: 200, defenderCount: 100, wantWinner: "attacker", wantTriggered: false},
		{name: "掠夺守城平局不触发", marchMode: PvpMarchTypePlunder, attackerCount: 100, defenderCount: 100, wantWinner: "draw", wantTriggered: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			setTestFactionsAndGenerals(t, FactionsConfig{
				"wei": {Name: "魏国", Generals: []GeneralInfo{{ID: "caocao", Name: "曹操"}}},
				"wu":  {Name: "吴国", Generals: []GeneralInfo{{ID: "sunce", Name: "孙策"}}},
			}, GeneralsConfig{Enabled: true, Heroes: map[string]GeneralHeroConfig{
				"caocao": {ID: "caocao", Name: "曹操", Faction: "wei", Enabled: true},
				"sunce": {
					ID: "sunce", Name: "孙策", Faction: "wu", Enabled: true,
					SpecialTrait: GeneralTraitConfig{
						TraitID: "xiaobawang_zhuiji", TraitType: general.TraitTypeSpecial, Enabled: true,
						Scope: "enemy_army", AllowedScenes: []string{"plunder"}, RequiredOutcome: "win",
						Params: map[string]float64{"effectRate": 0.5, "triggerChance": 1},
					},
				},
			}})
			svc, repo, attacker, defender := newPvpTestServiceForGenerals(t, "wei", "caocao", "wu", "sunce")
			attacker.Army = []ArmyUnit{{UnitType: "weiInfantry", Amount: tc.attackerCount}}
			defender.Army = []ArmyUnit{{UnitType: "wuInfantry", Amount: tc.defenderCount}}
			defender.Buildings = nil
			repo.players[attacker.Player.ID] = attacker
			repo.players[defender.Player.ID] = defender

			started, err := svc.StartPvpAttack(PvpAttackRequest{
				PlayerID: attacker.Player.ID, TargetPlayerID: defender.Player.ID, MarchMode: tc.marchMode,
				Troops: map[string]int{"weiInfantry": tc.attackerCount}, GeneralIDs: []string{"caocao"},
			})
			if err != nil {
				t.Fatalf("StartPvpAttack failed: %v", err)
			}
			forcePvpMarchDue(t, repo, started.March.ID)
			battle, err := svc.ResolvePvpMarch(started.March.ID)
			if err != nil {
				t.Fatalf("ResolvePvpMarch failed: %v", err)
			}
			if battle.Result["winner"] != tc.wantWinner {
				t.Fatalf("expected winner %s, battle=%+v", tc.wantWinner, battle)
			}

			attackerReports, _, err := repo.ListReports(attacker.Player.ID, 10, 0)
			if err != nil || len(attackerReports) != 1 {
				t.Fatalf("expected one attacker report, reports=%+v err=%v", attackerReports, err)
			}
			defenderReports, _, err := repo.ListReports(defender.Player.ID, 10, 0)
			if err != nil || len(defenderReports) != 1 {
				t.Fatalf("expected one defender report, reports=%+v err=%v", defenderReports, err)
			}
			attackerReport := attackerReports[0]
			defenderReport := defenderReports[0]
			outcome, triggered := defenderReport.TraitOutcomes["xiaobawang_zhuiji"]
			if triggered != tc.wantTriggered {
				t.Fatalf("expected triggered=%t, defender report=%+v", tc.wantTriggered, defenderReport)
			}
			for _, report := range []BattleReport{attackerReport, defenderReport} {
				if _, exists := report.TraitOutcomes["xiaobawang_zhuiji"]; exists != tc.wantTriggered || standardReportHasTrait(report.Detail, "xiaobawang_zhuiji") != tc.wantTriggered {
					t.Fatalf("expected both reports triggered=%t, report=%+v", tc.wantTriggered, report)
				}
				if report.Detail == nil || report.Detail.SecondarySide == nil || !standardDetailGeneralHasTrait(report.Detail, "xiaobawang_zhuiji") {
					t.Fatalf("expected carried defender trait snapshot in both reports, report=%+v", report)
				}
			}

			attackerLosses := pvpTestLossesFromBattle(t, battle, "attacker")
			defenderLosses := pvpTestLossesFromBattle(t, battle, "defender")
			storedMarch, err := repo.GetPvpMarch(started.March.ID)
			if err != nil || storedMarch.AttackTroops["weiInfantry"] != tc.attackerCount-attackerLosses["weiInfantry"] {
				t.Fatalf("expected returning attacker troops to match losses, march=%+v losses=%+v err=%v", storedMarch, attackerLosses, err)
			}
			storedDefender, err := repo.GetState(defender.Player.ID)
			if err != nil || armySliceToMap(storedDefender.Army)["wuInfantry"] != tc.defenderCount-defenderLosses["wuInfantry"] {
				t.Fatalf("expected defender army to match losses, state=%+v losses=%+v err=%v", storedDefender.Army, defenderLosses, err)
			}
			if pvpTestGeneralExp(storedDefender, "sunce") != attackerLosses["weiInfantry"] || defenderReport.GeneralExpGained != attackerLosses["weiInfantry"] {
				t.Fatalf("expected Sun Ce experience to use final attacker deaths, state=%+v report=%+v losses=%+v", storedDefender.Generals, defenderReport, attackerLosses)
			}

			if tc.wantTriggered {
				extra, ok := outcome.Detail["extraLosses"].(map[string]int)
				if !ok || extra["weiInfantry"] != 28 || outcome.OwnerSide != "defender" || outcome.OwnerGeneralID != "sunce" || outcome.Detail["effectRate"] != 0.5 {
					t.Fatalf("expected defender-owned positive pursuit result, outcome=%+v", outcome)
				}
				coreLosses := attackerLosses["weiInfantry"] - extra["weiInfantry"]
				if battle.Result["attackerPower"] != float64(1000) || battle.Result["defensePower"] != float64(2000) || coreLosses != 72 || attackerLosses["weiInfantry"] != 100 || defenderLosses["wuInfantry"] != 54 || attackerReport.SurvivedUnits["weiInfantry"] != 0 || defenderReport.SurvivedUnits["wuInfantry"] != 146 {
					t.Fatalf("expected 1000/2000 power, core losses 72/54 and pursuit +28, battle=%+v reports=%+v/%+v", battle, attackerReport, defenderReport)
				}
			}
			if tc.wantWinner == "draw" {
				if battle.Result["attackerPower"] != float64(1000) || battle.Result["defensePower"] != float64(1000) || attackerLosses["weiInfantry"] != 50 || defenderLosses["wuInfantry"] != 50 || defenderReport.GeneralExpGained != 50 {
					t.Fatalf("expected exact defender-side draw 1000/1000 and losses 50/50, battle=%+v report=%+v", battle, defenderReport)
				}
			}
		})
	}
}
