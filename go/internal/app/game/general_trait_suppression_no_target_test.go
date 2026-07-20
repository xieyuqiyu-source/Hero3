// 本文件验证正式特性压制在没有可拦截目标时不修改真实战斗，只如实记录零次实际压制。
package game

import (
	"reflect"
	"testing"

	"hero3/internal/core/general"
)

// TestPvpFormalTraitSuppressorsReportZeroWithoutEnemyFollowup 验证卧龙谋制和苦肉计在攻守双方均不会伪报压制目标。
func TestPvpFormalTraitSuppressorsReportZeroWithoutEnemyFollowup(t *testing.T) {
	cases := []struct {
		name        string
		generalID   string
		generalName string
		faction     string
		traitID     string
		traitType   string
	}{
		{name: "诸葛亮卧龙谋制", generalID: "zhugeliang", generalName: "诸葛亮", faction: "shu", traitID: "wolong_mouzhi", traitType: general.TraitTypeBonus},
		{name: "黄盖苦肉计", generalID: "huanggai", generalName: "黄盖", faction: "wu", traitID: "kurouji", traitType: general.TraitTypeSpecial},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			for _, ownerSide := range []string{"attacker", "defender"} {
				t.Run(ownerSide, func(t *testing.T) {
					opponentFaction := "wei"
					if tc.faction == opponentFaction {
						opponentFaction = "wu"
					}
					trait := GeneralTraitConfig{
						TraitID: tc.traitID, TraitType: tc.traitType, Enabled: true, Scope: "enemy_traits",
						Params: map[string]float64{"disableTraitCount": 1, "triggerChance": 1},
					}
					suppressor := GeneralHeroConfig{ID: tc.generalID, Name: tc.generalName, Faction: tc.faction, Enabled: true}
					if tc.traitType == general.TraitTypeSpecial {
						suppressor.SpecialTrait = trait
					} else {
						suppressor.BonusTrait = trait
					}
					setTestFactionsAndGenerals(t, FactionsConfig{
						tc.faction:      {Name: tc.faction, Generals: []GeneralInfo{{ID: tc.generalID, Name: tc.generalName}}},
						opponentFaction: {Name: opponentFaction, Generals: []GeneralInfo{{ID: "plain_general", Name: "无特性将领"}}},
					}, GeneralsConfig{Enabled: true, Heroes: map[string]GeneralHeroConfig{
						tc.generalID: suppressor,
						"plain_general": {
							ID: "plain_general", Name: "无特性将领", Faction: opponentFaction, Enabled: true,
						},
					}})

					attackerFaction, attackerGeneralID := tc.faction, tc.generalID
					defenderFaction, defenderGeneralID := opponentFaction, "plain_general"
					if ownerSide == "defender" {
						attackerFaction, attackerGeneralID = opponentFaction, "plain_general"
						defenderFaction, defenderGeneralID = tc.faction, tc.generalID
					}
					svc, repo, attacker, defender := newPvpTestServiceForGenerals(t, attackerFaction, attackerGeneralID, defenderFaction, defenderGeneralID)
					attackerUnitType := attackerFaction + "Infantry"
					defenderUnitType := defenderFaction + "Infantry"
					attacker.Army = []ArmyUnit{{UnitType: attackerUnitType, Amount: 100}}
					defender.Army = []ArmyUnit{{UnitType: defenderUnitType, Amount: 100}}
					defender.Buildings = nil
					repo.players[attacker.Player.ID] = attacker
					repo.players[defender.Player.ID] = defender

					started, err := svc.StartPvpAttack(PvpAttackRequest{
						PlayerID: attacker.Player.ID, TargetPlayerID: defender.Player.ID, MarchMode: PvpMarchTypePlunder,
						Troops: map[string]int{attackerUnitType: 100}, GeneralIDs: []string{attackerGeneralID},
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
					if battle.Result["attackerPower"] != float64(1000) || battle.Result["defensePower"] != float64(1000) || battle.Result["winner"] != "draw" || attackerLosses[attackerUnitType] != 50 || defenderLosses[defenderUnitType] != 50 {
						t.Fatalf("expected unchanged 1000/1000 draw and 50/50 losses, result=%+v losses=%+v", battle.Result, battle.Losses)
					}

					attackerReport, err := repo.GetReportForPlayer(attacker.Player.ID, battle.AttackerReportID)
					if err != nil {
						t.Fatalf("GetReportForPlayer attacker failed: %v", err)
					}
					defenderReport, err := repo.GetReportForPlayer(defender.Player.ID, battle.DefenderReportID)
					if err != nil {
						t.Fatalf("GetReportForPlayer defender failed: %v", err)
					}
					for _, report := range []BattleReport{attackerReport, defenderReport} {
						if !reflect.DeepEqual(report.TraitTriggered, []string{tc.traitID}) || len(report.TraitOutcomes) != 1 {
							t.Fatalf("expected only %s in complete timeline, report=%s triggered=%+v outcomes=%+v", tc.traitID, report.ID, report.TraitTriggered, report.TraitOutcomes)
						}
						outcome, ok := report.TraitOutcomes[tc.traitID]
						if !ok || outcome.OwnerSide != ownerSide || outcome.OwnerGeneralID != tc.generalID || outcome.Detail["disableTraitCount"] != 1 || outcome.Detail["disabledTraitCount"] != 0 || outcome.Detail["triggerChance"] != float64(1) {
							t.Fatalf("expected design suppression 1 and actual 0, report=%s outcome=%+v", report.ID, outcome)
						}
						if report.Detail == nil || len(report.Detail.Traits) != 1 || report.Detail.Traits[0].TraitID != tc.traitID || report.Detail.Traits[0].OwnerRole != ownerSide || report.Detail.Traits[0].Detail["disableTraitCount"] != 1 || report.Detail.Traits[0].Detail["disabledTraitCount"] != 0 {
							t.Fatalf("expected standard report to preserve zero actual suppression, report=%s detail=%+v", report.ID, report.Detail)
						}
					}

					storedMarch, marchErr := repo.GetPvpMarch(started.March.ID)
					storedDefender, defenderErr := repo.GetState(defender.Player.ID)
					if marchErr != nil || defenderErr != nil || storedMarch.Status != PvpMarchStatusReturning || storedMarch.AttackTroops[attackerUnitType] != 50 || armySliceToMap(storedDefender.Army)[defenderUnitType] != 50 {
						t.Fatalf("expected authoritative troops to match 50/50 reports, march=%+v defender=%+v errors=%v/%v", storedMarch, storedDefender.Army, marchErr, defenderErr)
					}
					ownerState := storedDefender
					ownerReport := defenderReport
					if ownerSide == "attacker" {
						ownerState, err = repo.GetState(attacker.Player.ID)
						if err != nil {
							t.Fatalf("GetState attacker failed: %v", err)
						}
						ownerReport = attackerReport
					}
					if pvpTestGeneralExp(ownerState, tc.generalID) != 50 || ownerReport.GeneralExpGained != 50 {
						t.Fatalf("expected owner exp 50 from unchanged real losses, generals=%+v report=%+v", ownerState.Generals, ownerReport)
					}
				})
			}
		})
	}
}
