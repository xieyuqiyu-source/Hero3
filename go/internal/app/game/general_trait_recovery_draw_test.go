// 本文件验证仅战败返兵的将领特性不会在真实 PVP 平局中误触发。
package game

import (
	"testing"

	"hero3/internal/core/general"
)

// TestPvpLossOnlyRecoveryTraitsDoNotTreatDrawAsDefeat 验证郭嘉在等势平局时不返兵。
func TestPvpLossOnlyRecoveryTraitsDoNotTreatDrawAsDefeat(t *testing.T) {
	cases := []struct {
		name        string
		generalID   string
		generalName string
		traitID     string
		traitType   string
		rate        float64
	}{
		{name: "郭嘉鬼才遗策", generalID: "guojia", generalName: "郭嘉", traitID: "guicai_yice", traitType: general.TraitTypeBonus, rate: 0.1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			trait := GeneralTraitConfig{
				TraitID: tc.traitID, TraitType: tc.traitType, Enabled: true, Scope: "self_army", RequiredOutcome: "loss",
				Params: map[string]float64{"lossReductionRate": tc.rate, "maxReturnCount": 10000, "triggerChance": 1},
			}
			hero := GeneralHeroConfig{ID: tc.generalID, Name: tc.generalName, Faction: "wei", Enabled: true}
			defenderGeneralID := "defender_" + tc.generalID
			defenderHero := GeneralHeroConfig{ID: defenderGeneralID, Name: tc.generalName + "守城", Faction: "shu", Enabled: true}
			if tc.traitType == general.TraitTypeSpecial {
				hero.SpecialTrait = trait
				defenderHero.SpecialTrait = trait
			} else {
				hero.BonusTrait = trait
				defenderHero.BonusTrait = trait
			}
			setTestFactionsAndGenerals(t, FactionsConfig{
				"wei": {Name: "魏国", Generals: []GeneralInfo{{ID: tc.generalID, Name: tc.generalName}}},
				"shu": {Name: "蜀国", Generals: []GeneralInfo{{ID: defenderGeneralID, Name: defenderHero.Name}}},
			}, GeneralsConfig{Enabled: true, Heroes: map[string]GeneralHeroConfig{
				tc.generalID:      hero,
				defenderGeneralID: defenderHero,
			}})

			svc, repo, attacker, defender := newPvpTestServiceForGenerals(t, "wei", tc.generalID, "shu", defenderGeneralID)
			attacker.Army = []ArmyUnit{{UnitType: "weiInfantry", Amount: 1000}}
			defender.Army = []ArmyUnit{{UnitType: "shuInfantry", Amount: 1000}}
			defender.Buildings = nil
			repo.players[attacker.Player.ID] = attacker
			repo.players[defender.Player.ID] = defender

			started, err := svc.StartPvpAttack(PvpAttackRequest{
				PlayerID: attacker.Player.ID, TargetPlayerID: defender.Player.ID, MarchMode: PvpMarchTypePlunder,
				Troops: map[string]int{"weiInfantry": 1000}, GeneralIDs: []string{tc.generalID},
			})
			if err != nil {
				t.Fatalf("StartPvpAttack failed: %v", err)
			}
			forcePvpMarchDue(t, repo, started.March.ID)
			battle, err := svc.ResolvePvpMarch(started.March.ID)
			if err != nil {
				t.Fatalf("ResolvePvpMarch failed: %v", err)
			}
			if battle.Result["winner"] != "draw" || pvpTestLossesFromBattle(t, battle, "attacker")["weiInfantry"] != 500 || pvpTestLossesFromBattle(t, battle, "defender")["shuInfantry"] != 500 {
				t.Fatalf("expected equal-power draw with gross losses 500/500, battle=%+v", battle)
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
				if len(report.TraitTriggered) != 0 || len(report.TraitOutcomes) != 0 || standardReportHasTrait(report.Detail, tc.traitID) {
					t.Fatalf("expected draw to omit loss-only trait %s, report=%+v", tc.traitID, report)
				}
			}
			storedMarch, err := repo.GetPvpMarch(started.March.ID)
			if err != nil || storedMarch.AttackTroops["weiInfantry"] != 500 || len(attackerReports[0].RevivedUnits) != 0 || attackerReports[0].SurvivedUnits["weiInfantry"] != 500 {
				t.Fatalf("expected no returned troops after draw, march=%+v report=%+v err=%v", storedMarch, attackerReports[0], err)
			}
			storedDefender, err := repo.GetState(defender.Player.ID)
			if err != nil || armySliceToMap(storedDefender.Army)["shuInfantry"] != 500 || len(defenderReports[0].RevivedUnits) != 0 || defenderReports[0].SurvivedUnits["shuInfantry"] != 500 {
				t.Fatalf("expected defender not to return troops after draw, state=%+v report=%+v err=%v", storedDefender, defenderReports[0], err)
			}
			if attackerReports[0].GeneralExpGained != 500 || defenderReports[0].GeneralExpGained != 500 {
				t.Fatalf("expected unchanged draw experience 500/500, reports=%d/%d", attackerReports[0].GeneralExpGained, defenderReports[0].GeneralExpGained)
			}
		})
	}
}

// TestPvpReinforcementLossOnlyRecoveryDoesNotTreatDrawAsDefeat 验证援军平局时也不触发战败返兵。
func TestPvpReinforcementLossOnlyRecoveryDoesNotTreatDrawAsDefeat(t *testing.T) {
	cases := []struct {
		name      string
		traitID   string
		traitType string
		rate      float64
	}{
		{name: "郭嘉鬼才遗策", traitID: "guicai_yice", traitType: general.TraitTypeBonus, rate: 0.1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			trait := GeneralTraitConfig{
				TraitID: tc.traitID, TraitType: tc.traitType, Enabled: true, Scope: "self_army", RequiredOutcome: "loss",
				Params: map[string]float64{"lossReductionRate": tc.rate, "maxReturnCount": 10000, "triggerChance": 1},
			}
			cfg := reinforcementEnemyPvpConfig{
				id: "draw_reinforcement_" + tc.traitID, attackerTroops: 1000, defenderTroops: 500, helperTroops: 500,
				marchMode:       PvpMarchTypePlunder,
				attackerFaction: "wei", attackerGeneral: "caocao", attackerName: "曹操",
				defenderFaction: "shu", defenderGeneral: "liubei", defenderName: "刘备",
				helperFaction: "shu", helperGeneralID: "helper_" + tc.traitID, helperName: tc.name,
			}
			if tc.traitType == general.TraitTypeSpecial {
				cfg.helperSpecial = trait
			} else {
				cfg.helperBonus = trait
			}
			result := runReinforcementEnemyPvp(t, cfg)
			if result.battle.Result["winner"] != "draw" || result.storedReinforcement.Losses["shuInfantry"] != 250 || result.storedReinforcement.RemainingTroops["shuInfantry"] != 250 {
				t.Fatalf("expected reinforcement draw without returned troops, battle=%+v reinforcement=%+v", result.battle, result.storedReinforcement)
			}
			for _, report := range []BattleReport{result.attackerReport, result.defenderReport, result.reinforcementReport} {
				if len(report.TraitTriggered) != 0 || len(report.TraitOutcomes) != 0 || standardReportHasTrait(report.Detail, tc.traitID) {
					t.Fatalf("expected draw to omit reinforcement loss-only trait %s, report=%+v", tc.traitID, report)
				}
			}
			if result.reinforcementReport.LostUnits["shuInfantry"] != 250 || result.reinforcementReport.SurvivedUnits["shuInfantry"] != 250 || result.reinforcementReport.GeneralExpGained != 500 {
				t.Fatalf("expected reinforcement report 500/250/250 and exp 500, report=%+v", result.reinforcementReport)
			}
		})
	}
}
