// 本文件验证同一场带掠夺特性的 PVP 战斗被并发解析时只结算一次资源、兵力、经验和战报。
package game

import (
	"sync"
	"testing"
	"time"

	"hero3/internal/core/general"
)

// TestPvpPlunderTraitsConcurrentResolveIsIdempotent 验证甘宁与孙权的掠夺修正不会因并发解析重复发放。
func TestPvpPlunderTraitsConcurrentResolveIsIdempotent(t *testing.T) {
	setTestFactionsAndGenerals(t, FactionsConfig{
		"wu":  {Name: "吴国", Generals: []GeneralInfo{{ID: "ganning", Name: "甘宁"}}},
		"wei": {Name: "魏国", Generals: []GeneralInfo{{ID: "sunquan", Name: "孙权"}}},
	}, GeneralsConfig{Enabled: true, Heroes: map[string]GeneralHeroConfig{
		"ganning": {
			ID: "ganning", Name: "甘宁", Faction: "wu", Enabled: true,
			SpecialTrait: GeneralTraitConfig{
				TraitID: "jinfan_jielue", TraitType: general.TraitTypeSpecial, Enabled: true,
				Scope: "self_plunder", AllowedSides: []string{"attacker"}, AllowedScenes: []string{"plunder"},
				Params: map[string]float64{"plunderBonusRate": 0.2, "triggerChance": 1},
			},
		},
		"sunquan": {
			ID: "sunquan", Name: "孙权", Faction: "wei", Enabled: true,
			SpecialTrait: GeneralTraitConfig{
				TraitID: "jiangdong_haoling", TraitType: general.TraitTypeSpecial, Enabled: true,
				Scope: "enemy_plunder", AllowedSides: []string{"defender"}, AllowedScenes: []string{"plunder"},
				Params: map[string]float64{"plunderBonusRate": -0.2, "triggerChance": 1},
			},
		},
	}})

	svc, repo, attacker, defender := newPvpTestServiceForGenerals(t, "wu", "ganning", "wei", "sunquan")
	attacker.Army = []ArmyUnit{{UnitType: "wuInfantry", Amount: 1000}}
	attacker.Resources.Items = map[string]int{"wood": 0, "stone": 0, "iron": 0, "food": 0}
	attacker.Resources.Capacity = map[string]int{"wood": 100000, "stone": 100000, "iron": 100000, "food": 100000}
	defender.Army = []ArmyUnit{{UnitType: "weiInfantry", Amount: 10}}
	defender.Buildings = nil
	defender.Resources.Items = map[string]int{"wood": 10000, "stone": 0, "iron": 0, "food": 0}
	defender.Resources.Capacity = map[string]int{"wood": 0, "stone": 0, "iron": 0, "food": 0}
	now := time.Now().UTC()
	attacker.ResourceSettledAt = now.Format(resourceDateLayout)
	defender.ResourceSettledAt = now.Format(resourceDateLayout)
	repo.players[attacker.Player.ID] = attacker
	repo.players[defender.Player.ID] = defender

	started, err := svc.StartPvpAttack(PvpAttackRequest{
		PlayerID: attacker.Player.ID, TargetPlayerID: defender.Player.ID, MarchMode: PvpMarchTypePlunder,
		Troops: map[string]int{"wuInfantry": 1000}, GeneralIDs: []string{"ganning"},
	})
	if err != nil {
		t.Fatalf("StartPvpAttack failed: %v", err)
	}
	forcePvpMarchDue(t, repo, started.March.ID)
	attackerPvpBefore, err := svc.GetPvpState(attacker.Player.ID)
	if err != nil {
		t.Fatalf("GetPvpPlayerState attacker before failed: %v", err)
	}
	defenderPvpBefore, err := svc.GetPvpState(defender.Player.ID)
	if err != nil {
		t.Fatalf("GetPvpPlayerState defender before failed: %v", err)
	}

	start := make(chan struct{})
	type resolveResult struct {
		battle PvpBattle
		err    error
	}
	results := make(chan resolveResult, 2)
	var workers sync.WaitGroup
	for range 2 {
		workers.Add(1)
		go func() {
			defer workers.Done()
			<-start
			battle, resolveErr := svc.ResolvePvpMarch(started.March.ID)
			results <- resolveResult{battle: battle, err: resolveErr}
		}()
	}
	close(start)
	workers.Wait()
	close(results)

	resolved := make([]PvpBattle, 0, 2)
	for result := range results {
		if result.err != nil {
			t.Fatalf("concurrent ResolvePvpMarch failed: %v", result.err)
		}
		resolved = append(resolved, result.battle)
	}
	if len(resolved) != 2 || resolved[0].ID == "" || resolved[0].ID != resolved[1].ID || resolved[0].AttackerReportID != resolved[1].AttackerReportID || resolved[0].DefenderReportID != resolved[1].DefenderReportID || resolved[0].Plunder["wood"] <= 0 || resolved[0].Plunder["wood"] != resolved[1].Plunder["wood"] {
		t.Fatalf("expected both callers to receive one persisted battle, results=%+v", resolved)
	}
	battle := resolved[0]
	attackerLosses := pvpTestLossesFromBattle(t, battle, "attacker")
	defenderLosses := pvpTestLossesFromBattle(t, battle, "defender")

	storedAttacker, err := repo.GetState(attacker.Player.ID)
	if err != nil {
		t.Fatalf("GetState attacker failed: %v", err)
	}
	storedDefender, err := repo.GetState(defender.Player.ID)
	if err != nil {
		t.Fatalf("GetState defender failed: %v", err)
	}
	if storedAttacker.Resources.Items["wood"] != battle.Plunder["wood"] || storedDefender.Resources.Items["wood"] != 10000-battle.Plunder["wood"] {
		t.Fatalf("expected plunder transferred once, attacker=%d defender=%d battle=%+v", storedAttacker.Resources.Items["wood"], storedDefender.Resources.Items["wood"], battle.Plunder)
	}
	if pvpTestGeneralExp(storedAttacker, "ganning") != defenderLosses["weiInfantry"] || pvpTestGeneralExp(storedDefender, "sunquan") != attackerLosses["wuInfantry"] {
		t.Fatalf("expected general exp awarded once from real losses, attacker=%+v defender=%+v losses=%+v/%+v", storedAttacker.Generals, storedDefender.Generals, attackerLosses, defenderLosses)
	}

	attackerReports, attackerTotal, attackerReportErr := repo.ListReports(attacker.Player.ID, 10, 0)
	defenderReports, defenderTotal, defenderReportErr := repo.ListReports(defender.Player.ID, 10, 0)
	if attackerReportErr != nil || defenderReportErr != nil || attackerTotal != 1 || defenderTotal != 1 || len(attackerReports) != 1 || len(defenderReports) != 1 {
		t.Fatalf("expected exactly one report per player, totals=%d/%d reports=%+v/%+v errors=%v/%v", attackerTotal, defenderTotal, attackerReports, defenderReports, attackerReportErr, defenderReportErr)
	}
	for _, report := range []BattleReport{attackerReports[0], defenderReports[0]} {
		isAttackerReport := report.PlayerID == attacker.Player.ID
		expectedReportID := battle.DefenderReportID
		if isAttackerReport {
			expectedReportID = battle.AttackerReportID
		}
		if report.ID != expectedReportID || len(report.TraitTriggered) != 2 || len(report.TraitOutcomes) != 2 {
			t.Fatalf("expected one complete authoritative plunder report, report=%+v battle=%+v", report, battle)
		}
		if isAttackerReport {
			if report.Rewards["wood"] != battle.Plunder["wood"] || report.Detail.Rewards.Resources["wood"] != battle.Plunder["wood"] {
				t.Fatalf("expected attacker reward to equal final plunder, report=%+v battle=%+v", report, battle)
			}
		} else if report.Rewards["wood"] != 0 || report.Detail.Rewards.Resources["wood"] != 0 || report.DefenderResources["wood"] != battle.Plunder["wood"] {
			t.Fatalf("expected defender view to expose enemy plunder without granting a reward, report=%+v battle=%+v", report, battle)
		}
		ganningDelta, ganningOK := report.TraitOutcomes["jinfan_jielue"].Detail["plunderDelta"].(map[string]int)
		sunquanDelta, sunquanOK := report.TraitOutcomes["jiangdong_haoling"].Detail["plunderDelta"].(map[string]int)
		if !ganningOK || !sunquanOK || ganningDelta["wood"] <= 0 || sunquanDelta["wood"] >= 0 {
			t.Fatalf("expected both actual plunder deltas once, report=%+v", report)
		}
	}

	battles, err := repo.ListPvpBattlesForPlayer(attacker.Player.ID)
	if err != nil || len(battles) != 1 || battles[0].ID != battle.ID {
		t.Fatalf("expected exactly one persisted battle, battles=%+v err=%v", battles, err)
	}
	attackerPvpAfter, err := svc.GetPvpState(attacker.Player.ID)
	if err != nil {
		t.Fatalf("GetPvpPlayerState attacker after failed: %v", err)
	}
	defenderPvpAfter, err := svc.GetPvpState(defender.Player.ID)
	if err != nil {
		t.Fatalf("GetPvpPlayerState defender after failed: %v", err)
	}
	attackerDelta, defenderDelta := pvpPointDeltas(battle.Result["winner"].(string))
	wantAttackerPoints := max(0, attackerPvpBefore.SeasonPoints+attackerDelta)
	wantDefenderPoints := max(0, defenderPvpBefore.SeasonPoints+defenderDelta)
	if attackerPvpAfter.SeasonPoints != wantAttackerPoints || defenderPvpAfter.SeasonPoints != wantDefenderPoints {
		t.Fatalf("expected PVP points applied once, before=%d/%d after=%d/%d deltas=%d/%d", attackerPvpBefore.SeasonPoints, defenderPvpBefore.SeasonPoints, attackerPvpAfter.SeasonPoints, defenderPvpAfter.SeasonPoints, attackerDelta, defenderDelta)
	}
}
