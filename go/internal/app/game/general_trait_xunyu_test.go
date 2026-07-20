// 本文件验证荀彧双留城能力在同一真实存档中随出征状态同步失效和恢复，且不会伪造战斗触发。
package game

import (
	"testing"
	"time"

	"hero3/internal/core/general"
)

// TestXunyuDualHomeTraitsFollowRealPvpAssignmentAndStayOutOfReport 验证留城、出征和归城三段资源与战报边界。
func TestXunyuDualHomeTraitsFollowRealPvpAssignmentAndStayOutOfReport(t *testing.T) {
	setTestFactionsAndGenerals(t, FactionsConfig{
		"wei": {Name: "魏国", Generals: []GeneralInfo{{ID: "xunyu", Name: "荀彧"}}},
		"shu": {Name: "蜀国", Generals: []GeneralInfo{{ID: "liubei", Name: "刘备"}}},
	}, GeneralsConfig{Enabled: true, Heroes: map[string]GeneralHeroConfig{
		"xunyu": {
			ID: "xunyu", Name: "荀彧", Faction: "wei", Enabled: true,
			SpecialTrait: GeneralTraitConfig{
				TraitID: "wangzuo_zhicai", TraitType: general.TraitTypeSpecial, Enabled: true, Scope: "self_city",
				Params: map[string]float64{"resourceCostReduction": 0.05, "triggerChance": 1},
			},
			BonusTrait: GeneralTraitConfig{
				TraitID: "neizheng_jingying", TraitType: general.TraitTypeBonus, Enabled: true, Scope: "self_city",
				Params: map[string]float64{"productionBonusRate": 0.05},
			},
		},
		"liubei": {ID: "liubei", Name: "刘备", Faction: "shu", Enabled: true},
	}})

	svc, repo, attacker, defender := newPvpTestServiceForGenerals(t, "wei", "xunyu", "shu", "liubei")
	unitsMu.Lock()
	weiInfantry := activeUnits["wei"]["weiInfantry"]
	weiInfantry.Cost = map[string]int{"wood": 100}
	weiInfantry.TrainSeconds = 60
	activeUnits["wei"]["weiInfantry"] = weiInfantry
	unitsMu.Unlock()
	attacker.Army = []ArmyUnit{{UnitType: "weiInfantry", Amount: 100}}
	attacker.Resources.Items["wood"] = 1200
	attacker.Resources.Capacity["wood"] = 100000
	attacker.ResourceSettledAt = time.Now().UTC().Add(-time.Hour).Format(resourceDateLayout)
	defender.Army = []ArmyUnit{{UnitType: "shuInfantry", Amount: 50}}
	defender.Buildings = nil
	repo.players[attacker.Player.ID] = attacker
	repo.players[defender.Player.ID] = defender

	home, err := svc.Recruit(attacker.Player.ID, "weiInfantry", 2)
	if err != nil {
		t.Fatalf("Recruit at home failed: %v", err)
	}
	if home.ResourceProduction["wood"] != 53 || home.Resources.Items["wood"] != 1063 || len(home.RecruitQueues) != 1 {
		t.Fatalf("expected home interval +53 and discounted cost 190, state=%+v", home)
	}
	assertNoBattleReportsForTraitProcess(t, repo, attacker.Player.ID, "Xun Yu home traits")

	started, err := svc.StartPvpAttack(PvpAttackRequest{
		PlayerID: attacker.Player.ID, TargetPlayerID: defender.Player.ID, MarchMode: PvpMarchTypeAttack,
		Troops: map[string]int{"weiInfantry": 100}, GeneralIDs: []string{"xunyu"},
	})
	if err != nil {
		t.Fatalf("StartPvpAttack failed: %v", err)
	}
	away, err := repo.GetState(attacker.Player.ID)
	if err != nil {
		t.Fatalf("GetState away failed: %v", err)
	}
	away.Resources.Items["wood"] = 1000
	away.ResourceSettledAt = time.Now().UTC().Add(-time.Hour).Format(resourceDateLayout)
	repo.players[attacker.Player.ID] = away
	awayAfterRecruit, err := svc.Recruit(attacker.Player.ID, "weiInfantry", 2)
	if err != nil {
		t.Fatalf("Recruit while away failed: %v", err)
	}
	if awayAfterRecruit.ResourceProduction["wood"] != 50 || awayAfterRecruit.Resources.Items["wood"] != 850 || len(awayAfterRecruit.RecruitQueues) != 2 {
		t.Fatalf("expected away interval +50 and full cost 200, state=%+v", awayAfterRecruit)
	}
	assertNoBattleReportsForTraitProcess(t, repo, attacker.Player.ID, "Xun Yu away traits")

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
	for _, report := range []BattleReport{attackerReports[0], defenderReports[0]} {
		assertXunyuPassiveBattleReport(t, report)
	}

	storedMarch, err := repo.GetPvpMarch(started.March.ID)
	if err != nil || storedMarch.Status != PvpMarchStatusReturning || storedMarch.AttackTroops["weiInfantry"] != 63 {
		t.Fatalf("expected 63 surviving infantry in return march, march=%+v err=%v", storedMarch, err)
	}
	forcePvpReturnDue(t, repo, started.March.ID)
	if _, err := svc.CompletePvpRecall(started.March.ID); err != nil {
		t.Fatalf("CompletePvpRecall failed: %v", err)
	}
	returned, err := repo.GetState(attacker.Player.ID)
	if err != nil {
		t.Fatalf("GetState returned failed: %v", err)
	}
	if !generalAvailableAtHome(returned.GeneralAssignments, "xunyu") {
		t.Fatalf("expected Xun Yu available at home after return, assignments=%+v", returned.GeneralAssignments)
	}
	returned.Resources.Items["wood"] = 1000
	returned.ResourceSettledAt = time.Now().UTC().Add(-time.Hour).Format(resourceDateLayout)
	repo.players[attacker.Player.ID] = returned
	homeAgain, err := svc.Recruit(attacker.Player.ID, "weiInfantry", 2)
	if err != nil {
		t.Fatalf("Recruit after return failed: %v", err)
	}
	if homeAgain.ResourceProduction["wood"] != 53 || homeAgain.Resources.Items["wood"] != 863 || len(homeAgain.RecruitQueues) != 3 {
		t.Fatalf("expected returned interval +53 and discounted cost 190, state=%+v", homeAgain)
	}
}

// assertXunyuPassiveBattleReport 核对荀彧拥有双特性，但双方战报没有任何虚假触发结果。
func assertXunyuPassiveBattleReport(t *testing.T, report BattleReport) {
	t.Helper()
	if len(report.TraitTriggered) != 0 || len(report.TraitOutcomes) != 0 || report.Detail == nil || len(report.Detail.Traits) != 0 {
		t.Fatalf("expected no battle outcomes from Xun Yu home traits, report=%+v", report)
	}
	if len(report.PvpAttackerGenerals) != 1 || !pvpSnapshotHasTrait(report.PvpAttackerGenerals[0], "wangzuo_zhicai") || !pvpSnapshotHasTrait(report.PvpAttackerGenerals[0], "neizheng_jingying") {
		t.Fatalf("expected Xun Yu snapshot to retain both owned traits, snapshots=%+v", report.PvpAttackerGenerals)
	}
	foundUnit := false
	for _, unit := range report.Detail.PrimarySide.Units {
		if unit.UnitType == "weiInfantry" {
			foundUnit = unit.Dispatched == 100 && unit.Lost == 37 && unit.Survived == 63
		}
	}
	if !foundUnit {
		t.Fatalf("expected standard attacker row 100/37/63, units=%+v", report.Detail.PrimarySide.Units)
	}
}
