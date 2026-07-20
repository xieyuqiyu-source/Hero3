// 本文件验证孙策小霸王追击作为援军时的掠夺场景、胜负、概率和三方战报边界。
package game

import (
	"testing"

	"hero3/internal/core/general"
)

// TestPvpSunCeReinforcementPursuitRequiresPlunderVictory 验证孙策援军只在守方联盟赢得掠夺战且概率命中后追击来袭方。
func TestPvpSunCeReinforcementPursuitRequiresPlunderVictory(t *testing.T) {
	cases := []struct {
		name                    string
		marchMode               string
		attackerTroops          int
		helperTroops            int
		triggerChance           float64
		wantWinner              string
		wantTriggered           bool
		wantAttackerLosses      int
		wantDefendingLosses     int
		wantMainLosses          int
		wantReinforcementLosses int
		wantAttackerExp         int
		wantDefenderExp         int
	}{
		{name: "掠夺守方获胜命中", marchMode: PvpMarchTypePlunder, attackerTroops: 100, helperTroops: 199, triggerChance: 1, wantWinner: "defender", wantTriggered: true, wantAttackerLosses: 82, wantDefendingLosses: 54, wantMainLosses: 0, wantReinforcementLosses: 54, wantAttackerExp: 54, wantDefenderExp: 82},
		{name: "掠夺守方获胜合法未命中", marchMode: PvpMarchTypePlunder, attackerTroops: 100, helperTroops: 199, triggerChance: 0, wantWinner: "defender", wantTriggered: false, wantAttackerLosses: 72, wantDefendingLosses: 54, wantMainLosses: 0, wantReinforcementLosses: 54, wantAttackerExp: 54, wantDefenderExp: 72},
		{name: "普通进攻守方获胜不触发", marchMode: PvpMarchTypeAttack, attackerTroops: 100, helperTroops: 199, triggerChance: 1, wantWinner: "defender", wantTriggered: false, wantAttackerLosses: 100, wantDefendingLosses: 74, wantMainLosses: 0, wantReinforcementLosses: 74, wantAttackerExp: 74, wantDefenderExp: 100},
		{name: "掠夺守方战败不触发", marchMode: PvpMarchTypePlunder, attackerTroops: 200, helperTroops: 99, triggerChance: 1, wantWinner: "attacker", wantTriggered: false, wantAttackerLosses: 54, wantDefendingLosses: 72, wantMainLosses: 0, wantReinforcementLosses: 72, wantAttackerExp: 72, wantDefenderExp: 54},
		{name: "掠夺平局不触发", marchMode: PvpMarchTypePlunder, attackerTroops: 100, helperTroops: 99, triggerChance: 1, wantWinner: "draw", wantTriggered: false, wantAttackerLosses: 50, wantDefendingLosses: 50, wantMainLosses: 1, wantReinforcementLosses: 49, wantAttackerExp: 50, wantDefenderExp: 50},
	}
	for index, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			id := "sunce_reinforcement_" + string(rune('a'+index))
			result := runReinforcementEnemyPvp(t, reinforcementEnemyPvpConfig{
				id: id, attackerTroops: tc.attackerTroops, defenderTroops: 1, marchMode: tc.marchMode,
				helperFaction: "wu", helperGeneralID: "sunce", helperName: "孙策", helperTroops: tc.helperTroops,
				attackerFaction: "wei", attackerGeneral: "caocao", attackerName: "曹操",
				defenderFaction: "shu", defenderGeneral: "liubei", defenderName: "刘备",
				helperSpecial: GeneralTraitConfig{
					TraitID: "xiaobawang_zhuiji", TraitType: general.TraitTypeSpecial, Enabled: true, Scope: "enemy_army",
					AllowedScenes: []string{"plunder"}, RequiredOutcome: "win",
					Params: map[string]float64{"effectRate": 0.1, "triggerChance": tc.triggerChance},
				},
			})
			if result.battle.Result["winner"] != tc.wantWinner || result.battle.Result["attackerPower"] != float64(tc.attackerTroops*10) || result.battle.Result["defensePower"] != float64((tc.helperTroops+1)*10) {
				t.Fatalf("expected winner/powers %s %d/%d, result=%+v", tc.wantWinner, tc.attackerTroops*10, (tc.helperTroops+1)*10, result.battle.Result)
			}
			if result.attackerLosses != tc.wantAttackerLosses || result.defendingLosses != tc.wantDefendingLosses || result.defenderReport.LostUnits["shuInfantry"] != tc.wantMainLosses || result.storedReinforcement.Losses["wuInfantry"] != tc.wantReinforcementLosses {
				t.Fatalf("expected attacker/main/reinforcement losses %d/%d/%d and defending total %d, battle=%+v main=%+v reinforcement=%+v", tc.wantAttackerLosses, tc.wantMainLosses, tc.wantReinforcementLosses, tc.wantDefendingLosses, result.battle.Losses, result.defenderReport.LostUnits, result.storedReinforcement)
			}
			if result.attackerReport.SurvivedUnits["weiInfantry"] != tc.attackerTroops-tc.wantAttackerLosses || result.reinforcementReport.SurvivedUnits["wuInfantry"] != tc.helperTroops-tc.wantReinforcementLosses || result.storedReinforcement.RemainingTroops["wuInfantry"] != tc.helperTroops-tc.wantReinforcementLosses {
				t.Fatalf("expected authoritative attacker/reinforcement survivors %d/%d, reports=%+v/%+v record=%+v", tc.attackerTroops-tc.wantAttackerLosses, tc.helperTroops-tc.wantReinforcementLosses, result.attackerReport.SurvivedUnits, result.reinforcementReport.SurvivedUnits, result.storedReinforcement)
			}
			if armySliceToMap(result.defenderState.Army)["shuInfantry"] != 1-tc.wantMainLosses || result.reinforcementReport.DispatchedUnits["wuInfantry"] != tc.helperTroops || result.reinforcementReport.LostUnits["wuInfantry"] != tc.wantReinforcementLosses {
				t.Fatalf("expected authoritative main survivor and helper report to reconcile, defender=%+v helper=%+v", result.defenderState.Army, result.reinforcementReport)
			}
			if result.attackerReport.GeneralExpGained != tc.wantAttackerExp || result.defenderReport.GeneralExpGained != tc.wantDefenderExp || result.reinforcementReport.GeneralExpGained != tc.wantDefenderExp || pvpTestGeneralExp(result.helperState, "sunce") != tc.wantDefenderExp {
				t.Fatalf("expected attacker/main/helper exp %d/%d/%d, reports=%d/%d/%d helper=%+v", tc.wantAttackerExp, tc.wantDefenderExp, tc.wantDefenderExp, result.attackerReport.GeneralExpGained, result.defenderReport.GeneralExpGained, result.reinforcementReport.GeneralExpGained, result.helperState.Generals)
			}

			for _, report := range []BattleReport{result.attackerReport, result.defenderReport, result.reinforcementReport} {
				outcome, triggered := reinforcementOutcome(report, "xiaobawang_zhuiji", "player_enemy_"+id)
				if triggered != tc.wantTriggered || standardReportHasTrait(report.Detail, "xiaobawang_zhuiji") != tc.wantTriggered {
					t.Fatalf("expected reinforcement pursuit triggered=%t in all timelines, report=%+v", tc.wantTriggered, report)
				}
				if len(report.PvpReinforcements) != 1 || len(report.PvpReinforcements[0].Generals) != 1 || report.PvpReinforcements[0].Generals[0].ID != "sunce" || !reinforcementSnapshotHasTrait(report.PvpReinforcements[0].Generals[0], "xiaobawang_zhuiji") {
					t.Fatalf("expected every report to retain Sun Ce reinforcement ownership snapshot, report=%s reinforcements=%+v", report.ID, report.PvpReinforcements)
				}
				if report.PvpReinforcementLosses[result.reinforcementID]["wuInfantry"] != tc.wantReinforcementLosses {
					t.Fatalf("expected every report to use reinforcement loss %d, report=%s losses=%+v", tc.wantReinforcementLosses, report.ID, report.PvpReinforcementLosses)
				}
				if !tc.wantTriggered {
					if len(report.TraitOutcomes) != 0 || len(report.TraitTriggered) != 0 || report.Detail == nil || len(report.Detail.Traits) != 0 {
						t.Fatalf("expected rejected or missed pursuit to leave empty timelines, report=%+v", report)
					}
					continue
				}
				extra, extraOK := outcome.Detail["extraLosses"].(map[string]int)
				if !extraOK || extra["weiInfantry"] != 10 || outcome.OwnerSide != "reinforcement" || outcome.OwnerGeneralID != "sunce" || outcome.Detail["effectRate"] != 0.1 || outcome.Detail["triggerChance"] != 1.0 {
					t.Fatalf("expected Sun Ce reinforcement pursuit to add real loss 10 with formal design values, report=%s outcome=%+v", report.ID, outcome)
				}
				standardMatched := false
				for _, trait := range report.Detail.Traits {
					standardExtra, standardExtraOK := trait.Detail["extraLosses"].(map[string]int)
					if trait.TraitID == "xiaobawang_zhuiji" && trait.OwnerRole == "reinforcement" && trait.OwnerPlayerID == "player_enemy_"+id && standardExtraOK && standardExtra["weiInfantry"] == 10 {
						standardMatched = true
					}
				}
				if !standardMatched {
					t.Fatalf("expected standard timeline to retain reinforcement player ownership and extra loss 10, report=%s traits=%+v", report.ID, report.Detail.Traits)
				}
			}
		})
	}
}
