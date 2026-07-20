// 本文件验证攻守双方连续修改同一攻击属性时的真实结算顺序与战报口径。
package game

import (
	"reflect"
	"testing"
	"time"

	"hero3/internal/core/general"
)

// TestPvpSunCeAttackBonusThenSimaYiAttackReductionReconcile 验证固定加攻先结算、比例降攻后结算，并分别记录本次实际变化。
func TestPvpSunCeAttackBonusThenSimaYiAttackReductionReconcile(t *testing.T) {
	setTestFactionsAndGenerals(t, FactionsConfig{
		"wu":  {Name: "吴国", Generals: []GeneralInfo{{ID: "sunce", Name: "孙策"}}},
		"wei": {Name: "魏国", Generals: []GeneralInfo{{ID: "simayi", Name: "司马懿"}}},
	}, GeneralsConfig{Enabled: true, Heroes: map[string]GeneralHeroConfig{
		"sunce": {
			ID: "sunce", Name: "孙策", Faction: "wu", Enabled: true,
			BonusTrait: GeneralTraitConfig{
				TraitID: "xiaobawang_tieqi", TraitType: general.TraitTypeBonus, Enabled: true,
				Scope: "self_army", TargetUnitType: "overlordRider", AllowedSides: []string{"attacker"},
				Params: map[string]float64{"unitAttackFlat": 50},
			},
		},
		"simayi": {
			ID: "simayi", Name: "司马懿", Faction: "wei", Enabled: true,
			BonusTrait: GeneralTraitConfig{
				TraitID: "mouding_houfa", TraitType: general.TraitTypeBonus, Enabled: true,
				Scope: "enemy_army", AllowedSides: []string{"defender"},
				Params: map[string]float64{"effectRate": 0.1},
			},
		},
	}})

	svc, repo, attacker, defender := newPvpTestServiceForGenerals(t, "wu", "sunce", "wei", "simayi")
	unitsMu.Lock()
	activeUnits["wu"]["overlordRider"] = UnitConfig{
		Name: "霸王骑", Category: "cavalry",
		Stats: map[string]int{"attack": 28, "infantryDefense": 10, "cavalryDefense": 33, "carryCapacity": 130, "upkeep": 4},
	}
	unitsMu.Unlock()
	attacker.Army = []ArmyUnit{{UnitType: "overlordRider", Amount: 200}}
	defender.Army = []ArmyUnit{{UnitType: "weiInfantry", Amount: 1000}}
	defender.Buildings = nil
	nowText := time.Now().UTC().Format(resourceDateLayout)
	attacker.ResourceSettledAt = nowText
	defender.ResourceSettledAt = nowText
	repo.players[attacker.Player.ID] = attacker
	repo.players[defender.Player.ID] = defender

	started, err := svc.StartPvpAttack(PvpAttackRequest{
		PlayerID: attacker.Player.ID, TargetPlayerID: defender.Player.ID, MarchMode: PvpMarchTypeAttack,
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
	if battle.Result["attackerPower"] != float64(14000) || battle.Result["defensePower"] != float64(8000) {
		t.Fatalf("expected 28+50 attack then 10%% reduction to produce 70*200=14000 against 8000, result=%+v", battle.Result)
	}

	attackerReports, _, err := repo.ListReports(attacker.Player.ID, 10, 0)
	if err != nil || len(attackerReports) != 1 || attackerReports[0].ID != battle.AttackerReportID {
		t.Fatalf("expected exactly one attacker report, reports=%+v err=%v", attackerReports, err)
	}
	defenderReports, _, err := repo.ListReports(defender.Player.ID, 10, 0)
	if err != nil || len(defenderReports) != 1 || defenderReports[0].ID != battle.DefenderReportID {
		t.Fatalf("expected exactly one defender report, reports=%+v err=%v", defenderReports, err)
	}
	wantTimeline := []string{"xiaobawang_tieqi", "mouding_houfa"}
	for _, report := range []BattleReport{attackerReports[0], defenderReports[0]} {
		if !reflect.DeepEqual(report.TraitTriggered, wantTimeline) {
			t.Fatalf("expected attacker bonus then defender reduction timeline %v, report=%s timeline=%v", wantTimeline, report.ID, report.TraitTriggered)
		}
		bonus, bonusOK := report.TraitOutcomes["xiaobawang_tieqi"].Detail["attackModifiedUnits"].(map[string]int)
		reduction, reductionOK := report.TraitOutcomes["mouding_houfa"].Detail["attackModifiedUnits"].(map[string]int)
		if !bonusOK || bonus["overlordRider"] != 50 || report.TraitOutcomes["xiaobawang_tieqi"].Detail["unitAttackFlat"] != float64(50) {
			t.Fatalf("expected Sun Ce outcome to keep design +50 and actual +50, report=%s outcome=%+v", report.ID, report.TraitOutcomes["xiaobawang_tieqi"])
		}
		if !reductionOK || reduction["overlordRider"] != -8 || report.TraitOutcomes["mouding_houfa"].Detail["attackReductionRate"] != 0.1 {
			t.Fatalf("expected Sima Yi outcome to reduce current 78 attack by actual 8, report=%s outcome=%+v", report.ID, report.TraitOutcomes["mouding_houfa"])
		}
		if report.Detail == nil || len(report.Detail.Traits) != 2 || report.Detail.Traits[0].TraitID != wantTimeline[0] || report.Detail.Traits[1].TraitID != wantTimeline[1] {
			t.Fatalf("expected standard report to preserve cross-owner execution order, report=%s detail=%+v", report.ID, report.Detail)
		}
	}

	attackerLosses := pvpTestLossesFromBattle(t, battle, "attacker")
	defenderLosses := pvpTestLossesFromBattle(t, battle, "defender")
	if attackerLosses["overlordRider"] != 90 || defenderLosses["weiInfantry"] != 1000 {
		t.Fatalf("expected exact attack-mode losses 90 overlord riders and 1000 infantry, attacker=%+v defender=%+v", attackerLosses, defenderLosses)
	}
	attackerReport := attackerReports[0]
	if attackerReport.LostUnits["overlordRider"] != attackerLosses["overlordRider"] || attackerReport.DefenderLostUnits["weiInfantry"] != defenderLosses["weiInfantry"] {
		t.Fatalf("expected legacy report losses to match persisted battle, report=%+v battle=%+v", attackerReport, battle.Losses)
	}
	storedDefender, err := repo.GetState(defender.Player.ID)
	if err != nil || armySliceToMap(storedDefender.Army)["weiInfantry"] != 1000-defenderLosses["weiInfantry"] {
		t.Fatalf("expected defender authoritative army to match report losses, state=%+v losses=%+v err=%v", storedDefender.Army, defenderLosses, err)
	}
	storedMarch, err := repo.GetPvpMarch(started.March.ID)
	if err != nil || storedMarch.Status != PvpMarchStatusReturning || storedMarch.AttackTroops["overlordRider"] != 200-attackerLosses["overlordRider"] {
		t.Fatalf("expected returning troops to match attacker losses, march=%+v losses=%+v err=%v", storedMarch, attackerLosses, err)
	}
	storedAttacker, err := repo.GetState(attacker.Player.ID)
	if err != nil || pvpTestGeneralExp(storedAttacker, "sunce") != defenderLosses["weiInfantry"] || attackerReport.GeneralExpGained != defenderLosses["weiInfantry"] {
		t.Fatalf("expected Sun Ce exp to equal real defender deaths, stored=%d report=%d losses=%+v err=%v", pvpTestGeneralExp(storedAttacker, "sunce"), attackerReport.GeneralExpGained, defenderLosses, err)
	}
	defenderExp := calculateGeneralBattleExpFromLosses(attacker.Player.Faction, pvpTestUnitLosses(attackerLosses))
	if defenderExp != 360 || pvpTestGeneralExp(storedDefender, "simayi") != defenderExp || defenderReports[0].GeneralExpGained != defenderExp {
		t.Fatalf("expected Sima Yi weighted exp 360 from 90 overlord rider deaths, stored=%d report=%d calculated=%d", pvpTestGeneralExp(storedDefender, "simayi"), defenderReports[0].GeneralExpGained, defenderExp)
	}
	for _, report := range []BattleReport{attackerReports[0], defenderReports[0]} {
		if report.Detail == nil || report.Detail.SecondarySide == nil {
			t.Fatalf("expected complete two-sided standard report, report=%s detail=%+v", report.ID, report.Detail)
		}
		primary := attackDefenseCrossReportUnit(t, report.Detail.PrimarySide, "overlordRider")
		secondary := attackDefenseCrossReportUnit(t, *report.Detail.SecondarySide, "weiInfantry")
		if primary.UnitType != "overlordRider" || primary.AmountBefore != 200 || primary.Lost != attackerLosses["overlordRider"] || primary.Survived != 200-attackerLosses["overlordRider"] {
			t.Fatalf("expected standard attacker row to reconcile, report=%s unit=%+v losses=%+v", report.ID, primary, attackerLosses)
		}
		if secondary.UnitType != "weiInfantry" || secondary.AmountBefore != 1000 || secondary.Lost != defenderLosses["weiInfantry"] || secondary.Survived != 1000-defenderLosses["weiInfantry"] {
			t.Fatalf("expected standard defender row to reconcile, report=%s unit=%+v losses=%+v", report.ID, secondary, defenderLosses)
		}
	}
}
