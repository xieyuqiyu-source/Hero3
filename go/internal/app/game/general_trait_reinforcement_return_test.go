// 本文件验证刘备、典韦和郭嘉的战后返兵特性进入真实援军状态与三方战报。
package game

import (
	"math"
	"testing"
	"time"

	"hero3/internal/core/general"
)

type reinforcementReturnCase struct {
	name         string
	faction      string
	generalID    string
	generalName  string
	specialTrait GeneralTraitConfig
	bonusTrait   GeneralTraitConfig
	expected     map[string]struct {
		detailKey string
		rate      float64
	}
	requiresLoss bool
}

type reinforcementReturnResult struct {
	winner  string
	losses  int
	record  Reinforcement
	reports []BattleReport
}

// enabledTraitCopy 切换正式特性开关，保留 ID 防止测试配置自动补入其他特性。
func enabledTraitCopy(trait GeneralTraitConfig, enabled bool) GeneralTraitConfig {
	trait.Enabled = enabled
	return trait
}

// runReinforcementReturnPvp 执行一场正式将领援军参与的真实 PVP。
func runReinforcementReturnPvp(t *testing.T, tc reinforcementReturnCase, enabled bool, attackerAmount int) reinforcementReturnResult {
	t.Helper()
	helper := GeneralHeroConfig{
		ID: tc.generalID, Name: tc.generalName, Faction: tc.faction, Enabled: true,
		SpecialTrait: enabledTraitCopy(tc.specialTrait, enabled),
		BonusTrait:   enabledTraitCopy(tc.bonusTrait, enabled),
	}
	attackerFaction, attackerID, attackerName := "shu", "liubei", "刘备"
	if tc.faction == "shu" {
		attackerFaction, attackerID, attackerName = "wei", "caocao", "曹操"
	}
	setTestFactionsAndGenerals(t, FactionsConfig{
		attackerFaction: {Name: attackerFaction, Generals: []GeneralInfo{{ID: attackerID, Name: attackerName}}},
		"wu":            {Name: "吴国", Generals: []GeneralInfo{{ID: "sunquan", Name: "孙权"}}},
		tc.faction:      {Name: tc.faction, Generals: []GeneralInfo{{ID: tc.generalID, Name: tc.generalName}}},
	}, GeneralsConfig{Enabled: true, Heroes: map[string]GeneralHeroConfig{
		attackerID:   {ID: attackerID, Name: attackerName, Faction: attackerFaction, Enabled: true},
		"sunquan":    {ID: "sunquan", Name: "孙权", Faction: "wu", Enabled: true},
		tc.generalID: helper,
	}})
	svc, repo, attacker, defender := newPvpTestServiceForGenerals(t, attackerFaction, attackerID, "wu", "sunquan")
	helperUnit := tc.faction + "Infantry"
	unitsMu.Lock()
	if activeUnits[tc.faction] == nil {
		activeUnits[tc.faction] = FactionUnits{}
	}
	activeUnits[tc.faction][helperUnit] = UnitConfig{
		Name: tc.generalName + "测试步兵", Category: "infantry",
		Stats: map[string]int{"attack": 10, "infantryDefense": 10, "cavalryDefense": 8, "carryCapacity": 5, "upkeep": 1},
	}
	unitsMu.Unlock()
	now := time.Now().UTC()
	helperAccount := Account{ID: "account_return_" + tc.generalID, Username: "return_" + tc.generalID, PasswordHash: "hash", CreatedAt: now}
	if err := repo.CreateAccount(helperAccount); err != nil {
		t.Fatalf("CreateAccount helper failed: %v", err)
	}
	helperState := newPlayerState("player_return_"+tc.generalID, tc.generalName+"援军", tc.faction, tc.generalID, now)
	helperState.Army = []ArmyUnit{{UnitType: helperUnit, Amount: 100}}
	if err := repo.CreatePlayer(helperAccount.ID, helperState, now); err != nil {
		t.Fatalf("CreatePlayer helper failed: %v", err)
	}
	attackerUnit := attackerFaction + "Infantry"
	attacker.Army = []ArmyUnit{{UnitType: attackerUnit, Amount: attackerAmount}}
	defender.Army = []ArmyUnit{{UnitType: "wuInfantry", Amount: 1}}
	defender.Buildings = nil
	repo.players[attacker.Player.ID] = attacker
	repo.players[defender.Player.ID] = defender

	sent, err := svc.SendReinforcement(SendReinforcementRequest{
		FromPlayerID: helperState.Player.ID, TargetPlayerID: defender.Player.ID,
		Troops: map[string]int{helperUnit: 100}, GeneralIDs: []string{tc.generalID},
	})
	if err != nil {
		t.Fatalf("SendReinforcement failed: %v", err)
	}
	forceReinforcementDue(t, repo, sent.Reinforcement.ID, true)
	if _, err := svc.MarkReinforcementArrived(sent.Reinforcement.ID); err != nil {
		t.Fatalf("MarkReinforcementArrived failed: %v", err)
	}
	started, err := svc.StartPvpAttack(PvpAttackRequest{
		PlayerID: attacker.Player.ID, TargetPlayerID: defender.Player.ID, MarchMode: PvpMarchTypeAttack,
		Troops: map[string]int{attackerUnit: attackerAmount}, GeneralIDs: []string{attackerID},
	})
	if err != nil {
		t.Fatalf("StartPvpAttack failed: %v", err)
	}
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
	helperReports, total, err := repo.ListReportsByQuery(BattleReportQuery{PlayerID: helperState.Player.ID, ViewType: ReportViewReinforcement, Page: 1, PageSize: 10})
	if err != nil || total != 1 || len(helperReports) != 1 {
		t.Fatalf("expected helper report, reports=%+v total=%d err=%v", helperReports, total, err)
	}
	finalLosses := attackerReports[0].PvpReinforcementLosses[sent.Reinforcement.ID][helperUnit]
	if defenderReports[0].PvpReinforcementLosses[sent.Reinforcement.ID][helperUnit] != finalLosses || helperReports[0].PvpReinforcementLosses[sent.Reinforcement.ID][helperUnit] != finalLosses {
		t.Fatalf("expected three reports to agree on final losses %d, reports=%+v/%+v/%+v", finalLosses, attackerReports[0].PvpReinforcementLosses, defenderReports[0].PvpReinforcementLosses, helperReports[0].PvpReinforcementLosses)
	}
	stored, err := repo.GetReinforcement(sent.Reinforcement.ID)
	if err != nil || stored.RemainingTroops[helperUnit] != 100-finalLosses {
		t.Fatalf("expected stored reinforcement %d, record=%+v err=%v", 100-finalLosses, stored, err)
	}
	winner, _ := battle.Result["winner"].(string)
	return reinforcementReturnResult{
		winner: winner, losses: finalLosses, record: stored,
		reports: []BattleReport{attackerReports[0], defenderReports[0], helperReports[0]},
	}
}

// formalReinforcementReturnCases 返回需要进入援军战后结算的正式特性配置。
func formalReinforcementReturnCases() []reinforcementReturnCase {
	return []reinforcementReturnCase{
		{
			name: "刘备仁德与仁主守护", faction: "shu", generalID: "liubei", generalName: "刘备",
			specialTrait: GeneralTraitConfig{TraitID: "rende", TraitType: general.TraitTypeSpecial, Scope: "self_army", Params: map[string]float64{"effectRate": 0.5, "reviveRate": 0.5, "maxReviveCount": 10000, "triggerChance": 1}},
			bonusTrait:   GeneralTraitConfig{TraitID: "renzhu_shouhu", TraitType: general.TraitTypeBonus, Scope: "self_army", Params: map[string]float64{"lossReductionRate": 0.1, "maxReturnCount": 10000, "triggerChance": 1}},
			expected: map[string]struct {
				detailKey string
				rate      float64
			}{"rende": {detailKey: "revivedUnits", rate: 0.5}, "renzhu_shouhu": {detailKey: "returnedUnits", rate: 0.1}},
		},
		{
			name: "郭嘉鬼才遗策", faction: "wei", generalID: "guojia", generalName: "郭嘉", requiresLoss: true,
			specialTrait: GeneralTraitConfig{TraitID: "shengui_zhicai", TraitType: general.TraitTypeSpecial, Scope: "self_city", Enabled: false},
			bonusTrait:   GeneralTraitConfig{TraitID: "guicai_yice", TraitType: general.TraitTypeBonus, Scope: "self_army", RequiredOutcome: "loss", Params: map[string]float64{"lossReductionRate": 0.1, "maxReturnCount": 10000, "triggerChance": 1}},
			expected: map[string]struct {
				detailKey string
				rate      float64
			}{"guicai_yice": {detailKey: "returnedUnits", rate: 0.1}},
		},
	}
}

// TestFormalReinforcementReturnTraitsMatchStateAndReports 验证援军返兵实际状态、三方战报和兼容归队字段一致。
func TestFormalReinforcementReturnTraitsMatchStateAndReports(t *testing.T) {
	for _, tc := range formalReinforcementReturnCases() {
		t.Run(tc.name, func(t *testing.T) {
			control := runReinforcementReturnPvp(t, tc, false, 110)
			active := runReinforcementReturnPvp(t, tc, true, 110)
			if control.winner != "attacker" || active.winner != "attacker" || control.losses <= 0 {
				t.Fatalf("expected reinforcement side to lose with real losses, control=%+v active=%+v", control, active)
			}
			totalReturned := 0
			for _, expectation := range tc.expected {
				totalReturned += int(math.Floor(float64(control.losses) * expectation.rate))
			}
			if active.losses != control.losses-totalReturned || active.record.RemainingTroops[tc.faction+"Infantry"] != 100-active.losses {
				t.Fatalf("expected losses %d - returned %d = %d, active=%+v", control.losses, totalReturned, control.losses-totalReturned, active)
			}
			for _, report := range active.reports {
				for traitID, expectation := range tc.expected {
					outcome, ok := report.TraitOutcomes[traitID]
					byUnit, detailOK := outcome.Detail[expectation.detailKey].(map[string]int)
					want := int(math.Floor(float64(control.losses) * expectation.rate))
					if !ok || !detailOK || byUnit[tc.faction+"Infantry"] != want || outcome.OwnerSide != "reinforcement" || outcome.OwnerPlayerID != active.record.OwnerPlayerID || outcome.OwnerGeneralID != tc.generalID {
						t.Fatalf("expected %s reinforcement return %d, got %+v", traitID, want, outcome)
					}
					rateKey, capKey := "lossReductionRate", "maxReturnCount"
					if traitID == "rende" {
						rateKey, capKey = "effectRate", "maxReviveCount"
					}
					if outcome.Detail[rateKey] != expectation.rate || outcome.Detail[capKey] != 10000 {
						t.Fatalf("expected %s design rate %.2f and cap 10000, got %+v", traitID, expectation.rate, outcome.Detail)
					}
					standardMatched := false
					for _, trait := range report.Detail.Traits {
						if trait.TraitID != traitID || trait.OwnerPlayerID != active.record.OwnerPlayerID || trait.GeneralID != tc.generalID {
							continue
						}
						if trait.Detail[rateKey] != expectation.rate || trait.Detail[capKey] != 10000 {
							t.Fatalf("expected standard %s design rate %.2f and cap 10000, got %+v", traitID, expectation.rate, trait.Detail)
						}
						standardMatched = true
					}
					if !standardMatched {
						t.Fatalf("expected %s in standard reinforcement report, got %+v", traitID, report.Detail.Traits)
					}
				}
			}
			helperReport := active.reports[2]
			if helperReport.RevivedUnits[tc.faction+"Infantry"] != totalReturned {
				t.Fatalf("expected reinforcement report revivedUnits %d, got %+v", totalReturned, helperReport.RevivedUnits)
			}
			if helperReport.LostUnits[tc.faction+"Infantry"] != control.losses || helperReport.SurvivedUnits[tc.faction+"Infantry"] != 100-active.losses {
				t.Fatalf("expected gross loss %d and final survivors %d, report=%+v", control.losses, 100-active.losses, helperReport)
			}
		})
	}
}

// TestConditionalReinforcementReturnTraitsDoNotTriggerAfterDefenseWin 验证典韦和郭嘉在援军一方获胜时不返兵。
func TestConditionalReinforcementReturnTraitsDoNotTriggerAfterDefenseWin(t *testing.T) {
	for _, tc := range formalReinforcementReturnCases() {
		if !tc.requiresLoss {
			continue
		}
		t.Run(tc.name, func(t *testing.T) {
			control := runReinforcementReturnPvp(t, tc, false, 100)
			active := runReinforcementReturnPvp(t, tc, true, 100)
			if control.winner != "defender" || active.winner != "defender" || control.losses != active.losses {
				t.Fatalf("expected defense win without returned losses, control=%+v active=%+v", control, active)
			}
			for _, report := range active.reports {
				for traitID := range tc.expected {
					if _, ok := report.TraitOutcomes[traitID]; ok {
						t.Fatalf("expected %s not to trigger after defense win, outcomes=%+v", traitID, report.TraitOutcomes)
					}
				}
			}
		})
	}
}
