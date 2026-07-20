// 本文件验证郭嘉的留城征兵减耗与战败返兵在同一真实存档中各自生效，且战报边界清晰。
package game

import (
	"testing"

	"hero3/internal/core/general"
)

// TestGuojiaDualTraitsKeepRecruitPassiveOutOfBattleTimeline 验证神鬼之才不混入战报，鬼才遗策与真实返程兵力一致。
func TestGuojiaDualTraitsKeepRecruitPassiveOutOfBattleTimeline(t *testing.T) {
	setTestFactionsAndGenerals(t, FactionsConfig{
		"wei": {Name: "魏国", Generals: []GeneralInfo{{ID: "guojia", Name: "郭嘉"}}},
		"shu": {Name: "蜀国", Generals: []GeneralInfo{{ID: "liubei", Name: "刘备"}}},
	}, GeneralsConfig{Enabled: true, Heroes: map[string]GeneralHeroConfig{
		"guojia": {
			ID: "guojia", Name: "郭嘉", Faction: "wei", Enabled: true,
			SpecialTrait: GeneralTraitConfig{
				TraitID: "shengui_zhicai", TraitType: general.TraitTypeSpecial, Enabled: true, Scope: "self_city",
				Params: map[string]float64{"resourceCostReduction": 0.5, "triggerChance": 1},
			},
			BonusTrait: GeneralTraitConfig{
				TraitID: "guicai_yice", TraitType: general.TraitTypeBonus, Enabled: true, Scope: "self_army", RequiredOutcome: "loss",
				Params: map[string]float64{"lossReductionRate": 0.1, "maxReturnCount": 10000, "triggerChance": 1},
			},
		},
		"liubei": {ID: "liubei", Name: "刘备", Faction: "shu", Enabled: true},
	}})

	svc, repo, attacker, defender := newPvpTestServiceForGenerals(t, "wei", "guojia", "shu", "liubei")
	unitsMu.Lock()
	weiInfantry := activeUnits["wei"]["weiInfantry"]
	weiInfantry.Cost = map[string]int{"wood": 100}
	weiInfantry.TrainSeconds = 60
	activeUnits["wei"]["weiInfantry"] = weiInfantry
	unitsMu.Unlock()
	attacker.Army = []ArmyUnit{{UnitType: "weiInfantry", Amount: 100}}
	attacker.Resources.Items["wood"] = 1200
	defender.Army = []ArmyUnit{{UnitType: "shuInfantry", Amount: 1000}}
	defender.Buildings = nil
	repo.players[attacker.Player.ID] = attacker
	repo.players[defender.Player.ID] = defender

	recruited, err := svc.Recruit(attacker.Player.ID, "weiInfantry", 2)
	if err != nil {
		t.Fatalf("Recruit failed: %v", err)
	}
	if recruited.Resources.Items["wood"] != 1100 || len(recruited.RecruitQueues) != 1 {
		t.Fatalf("expected home Guo Jia to reduce 200 wood cost to 100, state=%+v", recruited)
	}
	assertNoBattleReportsForTraitProcess(t, repo, attacker.Player.ID, "Guo Jia recruit cost reduction")

	started, err := svc.StartPvpAttack(PvpAttackRequest{
		PlayerID: attacker.Player.ID, TargetPlayerID: defender.Player.ID, MarchMode: PvpMarchTypeAttack,
		Troops: map[string]int{"weiInfantry": 100}, GeneralIDs: []string{"guojia"},
	})
	if err != nil {
		t.Fatalf("StartPvpAttack failed: %v", err)
	}
	forcePvpMarchDue(t, repo, started.March.ID)
	battle, err := svc.ResolvePvpMarch(started.March.ID)
	if err != nil {
		t.Fatalf("ResolvePvpMarch failed: %v", err)
	}
	if battle.Result["winner"] != "defender" {
		t.Fatalf("expected Guo Jia side to lose, result=%+v", battle.Result)
	}

	attackerReports, _, err := repo.ListReports(attacker.Player.ID, 10, 0)
	if err != nil || len(attackerReports) == 0 || attackerReports[0].ID != battle.AttackerReportID {
		t.Fatalf("expected attacker report, reports=%+v err=%v", attackerReports, err)
	}
	defenderReports, _, err := repo.ListReports(defender.Player.ID, 10, 0)
	if err != nil || len(defenderReports) == 0 || defenderReports[0].ID != battle.DefenderReportID {
		t.Fatalf("expected defender report, reports=%+v err=%v", defenderReports, err)
	}
	for _, report := range []BattleReport{attackerReports[0], defenderReports[0]} {
		assertGuojiaDualTraitBattleReport(t, report)
	}

	storedMarch, err := repo.GetPvpMarch(started.March.ID)
	if err != nil || storedMarch.AttackTroops["weiInfantry"] != 10 {
		t.Fatalf("expected 10 returned infantry in march, march=%+v err=%v", storedMarch, err)
	}
	storedAttacker, err := repo.GetState(attacker.Player.ID)
	if err != nil || storedAttacker.Resources.Items["wood"] != 1100 || len(storedAttacker.RecruitQueues) != 1 {
		t.Fatalf("expected recruit resources and queue unchanged by battle, state=%+v err=%v", storedAttacker, err)
	}
}

// assertGuojiaDualTraitBattleReport 核对郭嘉拥有双特性，但战斗时间线只记录本次真实返兵。
func assertGuojiaDualTraitBattleReport(t *testing.T, report BattleReport) {
	t.Helper()
	if report.ViewType == ReportViewAttack {
		if report.LostUnits["weiInfantry"] != 100 || report.RevivedUnits["weiInfantry"] != 10 || report.SurvivedUnits["weiInfantry"] != 10 {
			t.Fatalf("expected attacker legacy row to use gross loss 100, returned 10 and final survivors 10, report=%+v", report)
		}
	} else if report.DefenderUnits["weiInfantry"] != 100 || report.DefenderLostUnits["weiInfantry"] != 100 {
		t.Fatalf("expected defender-view legacy enemy row to preserve attacker dispatch and gross loss, report=%+v", report)
	}
	if len(report.TraitTriggered) != 1 || report.TraitTriggered[0] != "guicai_yice" || len(report.TraitOutcomes) != 1 {
		t.Fatalf("expected only Guicai Yice in legacy timeline, report=%+v", report)
	}
	if _, exists := report.TraitOutcomes["shengui_zhicai"]; exists {
		t.Fatalf("expected recruit passive absent from battle outcomes, outcomes=%+v", report.TraitOutcomes)
	}
	outcome := report.TraitOutcomes["guicai_yice"]
	returnedUnits, ok := outcome.Detail["returnedUnits"].(map[string]int)
	if !ok || outcome.OwnerGeneralID != "guojia" || outcome.OwnerSide != "attacker" || outcome.Detail["lossReductionRate"] != 0.1 || outcome.Detail["maxReturnCount"] != 10000 || returnedUnits["weiInfantry"] != 10 {
		t.Fatalf("expected exact Guicai return detail, outcome=%+v", outcome)
	}
	if report.Detail == nil || len(report.Detail.Traits) != 1 || report.Detail.Traits[0].TraitID != "guicai_yice" {
		t.Fatalf("expected only Guicai Yice in standard timeline, detail=%+v", report.Detail)
	}
	if len(report.PvpAttackerGenerals) != 1 || !pvpSnapshotHasTrait(report.PvpAttackerGenerals[0], "shengui_zhicai") || !pvpSnapshotHasTrait(report.PvpAttackerGenerals[0], "guicai_yice") {
		t.Fatalf("expected Guo Jia snapshot to retain both owned traits, snapshots=%+v", report.PvpAttackerGenerals)
	}
	foundUnit := false
	for _, unit := range report.Detail.PrimarySide.Units {
		if unit.UnitType == "weiInfantry" {
			foundUnit = unit.Dispatched == 100 && unit.Lost == 100 && unit.Survived == 10
		}
	}
	if !foundUnit {
		t.Fatalf("expected standard unit row 100/100/10, units=%+v", report.Detail.PrimarySide.Units)
	}
}
