// 本文件验证夏侯渊、魏延、孙权和太史慈的防御特性进入真实援军战力、战损和三方战报。
package game

import (
	"math"
	"testing"
	"time"

	"hero3/internal/core/general"
)

type reinforcementDefenseCase struct {
	name                 string
	faction              string
	generalID            string
	generalName          string
	disabledSpecialID    string
	specialTrait         GeneralTraitConfig
	traitID              string
	scope                string
	allowedSides         []string
	params               map[string]float64
	wantDefensePower     float64
	wantInfantryIncrease int
	wantCavalryIncrease  int
}

type reinforcementDefenseResult struct {
	defensePower float64
	losses       int
	sent         Reinforcement
	record       Reinforcement
	reports      []BattleReport
}

type xiahouyuanReinforcementComboResult struct {
	sent           Reinforcement
	battle         PvpBattle
	record         Reinforcement
	attackerReport BattleReport
	defenderReport BattleReport
	helperReport   BattleReport
	helperState    GameState
}

// formalReinforcementDefenseCases 返回四项正式援军防御特性的确定触发配置。
func formalReinforcementDefenseCases() []reinforcementDefenseCase {
	return []reinforcementDefenseCase{
		{
			name: "夏侯渊盾阵防御", faction: "wei", generalID: "xiahouyuan", generalName: "夏侯渊", disabledSpecialID: "jixing_benxi",
			traitID: "dunzhen_fangyu", scope: "self_army", allowedSides: []string{"defender", "reinforcement"},
			params:           map[string]float64{"defenseBonusRate": 0.35, "triggerChance": 1},
			wantDefensePower: 1410, wantInfantryIncrease: 4, wantCavalryIncrease: 3,
		},
		{
			name: "魏延固守汉中", faction: "shu", generalID: "weiyan", generalName: "魏延", disabledSpecialID: "qibing_raohou",
			traitID: "gushou_hanzhong", scope: "self_army", allowedSides: []string{"defender", "reinforcement"},
			params:           map[string]float64{"generalDefenseFlat": 20, "triggerChance": 1},
			wantDefensePower: 3010, wantInfantryIncrease: 20, wantCavalryIncrease: 20,
		},
		{
			name: "孙权江东固守", faction: "wu", generalID: "sunquan", generalName: "孙权", disabledSpecialID: "jiangdong_haoling",
			traitID: "jiangdong_gushou", scope: "self_army", allowedSides: []string{"defender", "reinforcement"},
			params:           map[string]float64{"defenseBonusRate": 0.5, "triggerChance": 1},
			wantDefensePower: 1510, wantInfantryIncrease: 5, wantCavalryIncrease: 4,
		},
		{
			name: "太史慈信义勇烈", faction: "wu", generalID: "taishici", generalName: "太史慈", disabledSpecialID: "kuairu_shandian",
			traitID: "xinyi_yonglie", scope: "reinforcement_self", allowedSides: []string{"reinforcement"},
			params:           map[string]float64{"defenseBonusRate": 0.1, "triggerChance": 1},
			wantDefensePower: 1110, wantInfantryIncrease: 1, wantCavalryIncrease: 1,
		},
	}
}

// runReinforcementDefensePvp 执行一场正式将领援军参与的完整 PVP 防守事务。
func runReinforcementDefensePvp(t *testing.T, tc reinforcementDefenseCase, enabled bool) reinforcementDefenseResult {
	t.Helper()
	attackerFaction, attackerID := "wei", "caocao"
	defenderFaction, defenderID := "wu", "sunquan"
	if tc.faction == "wei" {
		attackerFaction, attackerID = "shu", "liubei"
	}
	if tc.faction == "wu" {
		defenderFaction, defenderID = "shu", "liubei"
	}
	heroes := map[string]GeneralHeroConfig{
		"caocao":  {ID: "caocao", Name: "曹操", Faction: "wei", Enabled: true},
		"liubei":  {ID: "liubei", Name: "刘备", Faction: "shu", Enabled: true},
		"sunquan": {ID: "sunquan", Name: "孙权", Faction: "wu", Enabled: true},
	}
	specialTrait := GeneralTraitConfig{TraitID: tc.disabledSpecialID, TraitType: general.TraitTypeSpecial, Enabled: false}
	if tc.specialTrait.TraitID != "" {
		specialTrait = tc.specialTrait
	}
	heroes[tc.generalID] = GeneralHeroConfig{
		ID: tc.generalID, Name: tc.generalName, Faction: tc.faction, Enabled: true,
		SpecialTrait: specialTrait,
		BonusTrait: GeneralTraitConfig{
			TraitID: tc.traitID, TraitType: general.TraitTypeBonus, Enabled: enabled, Scope: tc.scope,
			AllowedSides: tc.allowedSides, Params: tc.params,
		},
	}
	setTestFactionsAndGenerals(t, FactionsConfig{
		"wei": {Name: "魏国", Generals: []GeneralInfo{{ID: "caocao", Name: "曹操"}, {ID: "xiahouyuan", Name: "夏侯渊"}}},
		"shu": {Name: "蜀国", Generals: []GeneralInfo{{ID: "liubei", Name: "刘备"}, {ID: "weiyan", Name: "魏延"}}},
		"wu":  {Name: "吴国", Generals: []GeneralInfo{{ID: "sunquan", Name: "孙权"}, {ID: "taishici", Name: "太史慈"}}},
	}, GeneralsConfig{Enabled: true, Heroes: heroes})
	svc, repo, attacker, defender := newPvpTestServiceForGenerals(t, attackerFaction, attackerID, defenderFaction, defenderID)
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
	helperAccount := Account{ID: "account_defense_" + tc.generalID, Username: "defense_" + tc.generalID, PasswordHash: "hash", CreatedAt: now}
	if err := repo.CreateAccount(helperAccount); err != nil {
		t.Fatalf("CreateAccount helper failed: %v", err)
	}
	helper := newPlayerState("player_defense_"+tc.generalID, tc.generalName+"援军", tc.faction, tc.generalID, now)
	helper.Army = []ArmyUnit{{UnitType: helperUnit, Amount: 100}}
	if err := repo.CreatePlayer(helperAccount.ID, helper, now); err != nil {
		t.Fatalf("CreatePlayer helper failed: %v", err)
	}
	attackerUnit := attackerFaction + "Infantry"
	defenderUnit := defenderFaction + "Infantry"
	attacker.Army = []ArmyUnit{{UnitType: attackerUnit, Amount: 110}}
	defender.Army = []ArmyUnit{{UnitType: defenderUnit, Amount: 1}}
	defender.Buildings = nil
	repo.players[attacker.Player.ID] = attacker
	repo.players[defender.Player.ID] = defender
	for _, position := range []struct {
		playerID string
		x        int
		y        int
	}{
		{playerID: helper.Player.ID, x: 10, y: 10},
		{playerID: defender.Player.ID, x: 20, y: 10},
		{playerID: attacker.Player.ID, x: 30, y: 10},
	} {
		if _, err := repo.AssignWorldPosition(position.playerID, defaultWorldID, position.x, position.y, "test"); err != nil {
			t.Fatalf("AssignWorldPosition %s failed: %v", position.playerID, err)
		}
	}

	sent, err := svc.SendReinforcement(SendReinforcementRequest{
		FromPlayerID: helper.Player.ID, TargetPlayerID: defender.Player.ID,
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
		Troops: map[string]int{attackerUnit: 110}, GeneralIDs: []string{attackerID},
	})
	if err != nil {
		t.Fatalf("StartPvpAttack failed: %v", err)
	}
	forcePvpMarchDue(t, repo, started.March.ID)
	battle, err := svc.ResolvePvpMarch(started.March.ID)
	if err != nil {
		t.Fatalf("ResolvePvpMarch failed: %v", err)
	}
	defensePower, ok := battle.Result["defensePower"].(float64)
	if !ok {
		t.Fatalf("expected numeric defense power, got %+v", battle.Result)
	}
	attackerReports, _, err := repo.ListReports(attacker.Player.ID, 10, 0)
	if err != nil || len(attackerReports) == 0 || attackerReports[0].ID != battle.AttackerReportID {
		t.Fatalf("expected attacker report, reports=%+v err=%v", attackerReports, err)
	}
	defenderReports, _, err := repo.ListReports(defender.Player.ID, 10, 0)
	if err != nil || len(defenderReports) == 0 || defenderReports[0].ID != battle.DefenderReportID {
		t.Fatalf("expected defender report, reports=%+v err=%v", defenderReports, err)
	}
	helperReports, total, err := repo.ListReportsByQuery(BattleReportQuery{PlayerID: helper.Player.ID, ViewType: ReportViewReinforcement, Page: 1, PageSize: 10})
	if err != nil || total != 1 || len(helperReports) != 1 {
		t.Fatalf("expected helper report, reports=%+v total=%d err=%v", helperReports, total, err)
	}
	losses := defenderReports[0].PvpReinforcementLosses[sent.Reinforcement.ID][helperUnit]
	if attackerReports[0].PvpReinforcementLosses[sent.Reinforcement.ID][helperUnit] != losses || helperReports[0].LostUnits[helperUnit] != losses {
		t.Fatalf("expected three reports to agree on losses %d, reports=%+v/%+v/%+v", losses, attackerReports[0].PvpReinforcementLosses, defenderReports[0].PvpReinforcementLosses, helperReports[0].LostUnits)
	}
	stored, err := repo.GetReinforcement(sent.Reinforcement.ID)
	if err != nil || stored.RemainingTroops[helperUnit] != 100-losses {
		t.Fatalf("expected stored reinforcement %d, record=%+v err=%v", 100-losses, stored, err)
	}
	return reinforcementDefenseResult{
		defensePower: defensePower, losses: losses, sent: sent.Reinforcement, record: stored,
		reports: []BattleReport{attackerReports[0], defenderReports[0], helperReports[0]},
	}
}

// TestFormalDefenseTraitsStrengthenRealReinforcements 验证四项正式特性真实提高援军防御并进入三方战报。
func TestFormalDefenseTraitsStrengthenRealReinforcements(t *testing.T) {
	for _, tc := range formalReinforcementDefenseCases() {
		t.Run(tc.name, func(t *testing.T) {
			designKey := "defenseBonusRate"
			wantDesignValue := tc.params[designKey]
			if tc.params["generalDefenseFlat"] > 0 {
				designKey = "generalDefenseFlat"
				wantDesignValue = tc.params[designKey]
			}
			control := runReinforcementDefensePvp(t, tc, false)
			active := runReinforcementDefensePvp(t, tc, true)
			if control.defensePower != 1010 || active.defensePower != tc.wantDefensePower {
				t.Fatalf("expected defense power 1010 -> %.0f, control=%v active=%v", tc.wantDefensePower, control.defensePower, active.defensePower)
			}
			if active.losses >= control.losses {
				t.Fatalf("expected stronger reinforcement to lose fewer troops, control=%d active=%d", control.losses, active.losses)
			}
			for _, report := range active.reports {
				outcome, ok := report.TraitOutcomes[tc.traitID]
				infantry, infantryOK := outcome.Detail["infantryDefenseModifiedUnits"].(map[string]int)
				cavalry, cavalryOK := outcome.Detail["cavalryDefenseModifiedUnits"].(map[string]int)
				designValue, designOK := outcome.Detail[designKey].(float64)
				if !ok || !infantryOK || !cavalryOK || !designOK || designValue != wantDesignValue || infantry[tc.faction+"Infantry"] != tc.wantInfantryIncrease || cavalry[tc.faction+"Infantry"] != tc.wantCavalryIncrease ||
					outcome.OwnerSide != "reinforcement" || outcome.OwnerGeneralID != tc.generalID {
					t.Fatalf("expected reinforcement-owned defense deltas +%d/+%d, got %+v", tc.wantInfantryIncrease, tc.wantCavalryIncrease, outcome)
				}
				standardFound := false
				for _, trait := range report.Detail.Traits {
					if trait.TraitID == tc.traitID && trait.GeneralID == tc.generalID && trait.OwnerRole == "reinforcement" {
						standardFound = true
					}
				}
				if !standardFound {
					t.Fatalf("expected %s in standard reinforcement timeline, traits=%+v", tc.traitID, report.Detail.Traits)
				}
			}
		})
	}
}

// TestTaishiciLightningLegalMissKeepsXinyiReinforcementDefense 验证快如闪电合法未命中时行军不变，信义勇烈仍独立提高援军防御。
func TestTaishiciLightningLegalMissKeepsXinyiReinforcementDefense(t *testing.T) {
	var tc reinforcementDefenseCase
	for _, candidate := range formalReinforcementDefenseCases() {
		if candidate.generalID == "taishici" {
			tc = candidate
			break
		}
	}
	if tc.generalID == "" {
		t.Fatal("expected Taishi Ci reinforcement defense case")
	}
	control := runReinforcementDefensePvp(t, tc, true)
	tc.specialTrait = marchTraitConfig("kuairu_shandian", general.TraitTypeSpecial, 4, 30, 0)
	missed := runReinforcementDefensePvp(t, tc, true)
	if control.sent.MarchSeconds != 2970 || missed.sent.MarchSeconds != control.sent.MarchSeconds || missed.sent.ReturnSeconds != control.sent.ReturnSeconds || math.Abs(missed.sent.SpeedMultiplier-control.sent.SpeedMultiplier) > 1e-9 {
		t.Fatalf("expected missed lightning to keep authoritative 2970-second march and speed, control=%+v missed=%+v", control.sent, missed.sent)
	}
	if hitDuration := applyExpectedMarchRates(control.sent.MarchSeconds, []float64{4}, 30); hitDuration != 594 || missed.sent.MarchSeconds == hitDuration {
		t.Fatalf("expected missed duration 2970 to differ from forced-hit 594, control=%+v missed=%+v", control.sent, missed.sent)
	}
	assertMarchDurationTimestamps(t, missed.sent.SentAt, missed.sent.ExpectedArriveAt, missed.sent.MarchSeconds)
	if missed.defensePower != 1110 || missed.losses != 98 || missed.record.Losses["wuInfantry"] != 98 || missed.record.RemainingTroops["wuInfantry"] != 2 {
		t.Fatalf("expected Xinyi to keep defense 1110 and reinforcement losses/survivors 98/2, result=%+v", missed)
	}
	for _, report := range missed.reports {
		if len(report.TraitTriggered) != 1 || report.TraitTriggered[0] != "xinyi_yonglie" || len(report.TraitOutcomes) != 1 {
			t.Fatalf("expected only Xinyi in legacy battle timeline, report=%s triggered=%+v outcomes=%+v", report.ID, report.TraitTriggered, report.TraitOutcomes)
		}
		if _, exists := report.TraitOutcomes["kuairu_shandian"]; exists || standardReportHasTrait(report.Detail, "kuairu_shandian") {
			t.Fatalf("expected missed lightning absent from battle timeline, report=%s outcomes=%+v detail=%+v", report.ID, report.TraitOutcomes, report.Detail)
		}
		outcome := report.TraitOutcomes["xinyi_yonglie"]
		infantry, infantryOK := outcome.Detail["infantryDefenseModifiedUnits"].(map[string]int)
		cavalry, cavalryOK := outcome.Detail["cavalryDefenseModifiedUnits"].(map[string]int)
		if !infantryOK || !cavalryOK || infantry["wuInfantry"] != 1 || cavalry["wuInfantry"] != 1 || outcome.OwnerSide != "reinforcement" || outcome.OwnerGeneralID != "taishici" {
			t.Fatalf("expected Xinyi actual defense deltas +1/+1 owned by Taishi Ci reinforcement, report=%s outcome=%+v", report.ID, outcome)
		}
		if report.Detail == nil || len(report.Detail.Traits) != 1 || report.Detail.Traits[0].TraitID != "xinyi_yonglie" || report.Detail.Traits[0].OwnerRole != "reinforcement" {
			t.Fatalf("expected standard timeline to contain only reinforcement Xinyi, report=%s detail=%+v", report.ID, report.Detail)
		}
		if len(report.PvpReinforcements) != 1 || len(report.PvpReinforcements[0].Generals) != 1 {
			t.Fatalf("expected Taishi Ci snapshot to retain both owned traits, report=%s reinforcements=%+v", report.ID, report.PvpReinforcements)
		}
		ownedTraits := map[string]bool{}
		for _, trait := range report.PvpReinforcements[0].Generals[0].Traits {
			ownedTraits[trait.TraitID] = true
		}
		if !ownedTraits["kuairu_shandian"] || !ownedTraits["xinyi_yonglie"] {
			t.Fatalf("expected Taishi Ci snapshot to retain both owned traits, report=%s traits=%+v", report.ID, report.PvpReinforcements[0].Generals[0].Traits)
		}
		if report.PvpReinforcementLosses[missed.sent.ID]["wuInfantry"] != 98 {
			t.Fatalf("expected every report to use reinforcement loss 98, report=%s losses=%+v", report.ID, report.PvpReinforcementLosses)
		}
	}
	helperReport := missed.reports[2]
	if helperReport.DispatchedUnits["wuInfantry"] != 100 || helperReport.LostUnits["wuInfantry"] != 98 || helperReport.SurvivedUnits["wuInfantry"] != 2 || helperReport.GeneralExpGained != 110 || helperReport.Detail.Rewards.GeneralExp != 110 {
		t.Fatalf("expected helper report troops 100/98/2 and exp 110, report=%+v", helperReport)
	}
}

// TestSunquanJiangdongGushouLegalMissAsReinforcement 验证孙权作为援军时江东固守合法未命中，不增加防御且不生成虚假战报结果。
func TestSunquanJiangdongGushouLegalMissAsReinforcement(t *testing.T) {
	var tc reinforcementDefenseCase
	for _, candidate := range formalReinforcementDefenseCases() {
		if candidate.generalID == "sunquan" {
			tc = candidate
			break
		}
	}
	if tc.generalID == "" {
		t.Fatal("expected Sun Quan reinforcement defense case")
	}
	tc.params = map[string]float64{"defenseBonusRate": 0.5, "triggerChance": 0}
	missed := runReinforcementDefensePvp(t, tc, true)
	if missed.defensePower != 1010 || missed.losses != 100 || missed.record.Losses["wuInfantry"] != 100 || missed.record.RemainingTroops["wuInfantry"] != 0 {
		t.Fatalf("expected missed Jiangdong Gushou to keep defense 1010 and reinforcement losses/survivors 100/0, result=%+v", missed)
	}
	for _, report := range missed.reports {
		if len(report.TraitTriggered) != 0 || len(report.TraitOutcomes) != 0 || report.Detail == nil || len(report.Detail.Traits) != 0 {
			t.Fatalf("expected missed Jiangdong Gushou absent from every timeline, report=%+v", report)
		}
		if len(report.PvpReinforcements) != 1 || len(report.PvpReinforcements[0].Generals) != 1 || report.PvpReinforcements[0].Generals[0].ID != "sunquan" {
			t.Fatalf("expected Sun Quan reinforcement snapshot, report=%s reinforcements=%+v", report.ID, report.PvpReinforcements)
		}
		if !reinforcementSnapshotHasTrait(report.PvpReinforcements[0].Generals[0], "jiangdong_gushou") {
			t.Fatalf("expected snapshot to retain Jiangdong Gushou ownership after miss, report=%s generals=%+v", report.ID, report.PvpReinforcements[0].Generals)
		}
		if report.PvpReinforcementLosses[missed.sent.ID]["wuInfantry"] != 100 {
			t.Fatalf("expected every report to retain full reinforcement loss 100, report=%s losses=%+v", report.ID, report.PvpReinforcementLosses)
		}
	}
	attackerReport, defenderReport, helperReport := missed.reports[0], missed.reports[1], missed.reports[2]
	if attackerReport.GeneralExpGained != 101 || defenderReport.GeneralExpGained != 97 || helperReport.GeneralExpGained != 97 || helperReport.Detail.Rewards.GeneralExp != 97 {
		t.Fatalf("expected attacker/main/helper exp 101/97/97 from base losses, reports=%+v", missed.reports)
	}
	if helperReport.DispatchedUnits["wuInfantry"] != 100 || helperReport.LostUnits["wuInfantry"] != 100 || helperReport.SurvivedUnits["wuInfantry"] != 0 {
		t.Fatalf("expected helper report troops 100/100/0 without defense bonus, report=%+v", helperReport)
	}
}

// runXiahouyuanReinforcementCombo 执行夏侯渊从增援创建、到达驻防到真实参战的完整事务。
func runXiahouyuanReinforcementCombo(t *testing.T, enabled bool) xiahouyuanReinforcementComboResult {
	t.Helper()
	setTestFactionsAndGenerals(t, FactionsConfig{
		"wei": {Name: "魏国", Generals: []GeneralInfo{{ID: "xiahouyuan", Name: "夏侯渊"}}},
		"shu": {Name: "蜀国", Generals: []GeneralInfo{{ID: "liubei", Name: "刘备"}}},
		"wu":  {Name: "吴国", Generals: []GeneralInfo{{ID: "sunquan", Name: "孙权"}}},
	}, GeneralsConfig{Enabled: true, Heroes: map[string]GeneralHeroConfig{
		"xiahouyuan": {
			ID: "xiahouyuan", Name: "夏侯渊", Faction: "wei", Enabled: true,
			SpecialTrait: GeneralTraitConfig{
				TraitID: "jixing_benxi", TraitType: general.TraitTypeSpecial, Enabled: enabled, Scope: "self_army",
				Params: map[string]float64{"speedBonusRate": 0.2, "minMarchSeconds": 60, "triggerChance": 1},
			},
			BonusTrait: GeneralTraitConfig{
				TraitID: "dunzhen_fangyu", TraitType: general.TraitTypeBonus, Enabled: enabled, Scope: "self_army",
				AllowedSides: []string{"defender", "reinforcement"},
				Params:       map[string]float64{"defenseBonusRate": 0.35, "triggerChance": 1},
			},
		},
		"liubei":  {ID: "liubei", Name: "刘备", Faction: "shu", Enabled: true},
		"sunquan": {ID: "sunquan", Name: "孙权", Faction: "wu", Enabled: true},
	}})
	svc, repo, attacker, defender := newPvpTestServiceForGenerals(t, "shu", "liubei", "wu", "sunquan")
	unitsMu.Lock()
	activeUnits["wei"]["huWei"] = UnitConfig{
		Name: "虎卫", Category: "infantry",
		Stats: map[string]int{"attack": 10, "infantryDefense": 10, "cavalryDefense": 8, "speed": 1, "carryCapacity": 5, "upkeep": 1},
	}
	unitsMu.Unlock()
	now := time.Now().UTC()
	helperAccount := Account{ID: "account_xiahouyuan_combo", Username: "xiahouyuan_combo", PasswordHash: "hash", CreatedAt: now}
	if err := repo.CreateAccount(helperAccount); err != nil {
		t.Fatalf("CreateAccount helper failed: %v", err)
	}
	helper := newPlayerState("player_xiahouyuan_combo", "夏侯渊援军", "wei", "xiahouyuan", now)
	baselineExp := generalExpRequiredForLevelForTest(2) - 1
	setPvpTestGeneralProgress(&helper, "xiahouyuan", 1, baselineExp)
	helper.Army = []ArmyUnit{{UnitType: "huWei", Amount: 100}}
	if err := repo.CreatePlayer(helperAccount.ID, helper, now); err != nil {
		t.Fatalf("CreatePlayer helper failed: %v", err)
	}
	attacker.Army = []ArmyUnit{{UnitType: "shuInfantry", Amount: 110}}
	defender.Army = []ArmyUnit{{UnitType: "wuInfantry", Amount: 1}}
	defender.Buildings = nil
	repo.players[attacker.Player.ID] = attacker
	repo.players[defender.Player.ID] = defender
	for _, position := range []struct {
		playerID string
		x        int
		y        int
	}{
		{playerID: helper.Player.ID, x: 10, y: 10},
		{playerID: defender.Player.ID, x: 20, y: 10},
		{playerID: attacker.Player.ID, x: 30, y: 10},
	} {
		if _, err := repo.AssignWorldPosition(position.playerID, defaultWorldID, position.x, position.y, "test"); err != nil {
			t.Fatalf("AssignWorldPosition %s failed: %v", position.playerID, err)
		}
	}

	sent, err := svc.SendReinforcement(SendReinforcementRequest{
		FromPlayerID: helper.Player.ID, TargetPlayerID: defender.Player.ID,
		Troops: map[string]int{"huWei": 100}, GeneralIDs: []string{"xiahouyuan"},
	})
	if err != nil {
		t.Fatalf("SendReinforcement failed: %v", err)
	}
	assertNoPreCombatMarchReport(t, repo, helper.Player.ID)
	assertMarchDurationTimestamps(t, sent.Reinforcement.SentAt, sent.Reinforcement.ExpectedArriveAt, sent.Reinforcement.MarchSeconds)
	forceReinforcementDue(t, repo, sent.Reinforcement.ID, true)
	if _, err := svc.MarkReinforcementArrived(sent.Reinforcement.ID); err != nil {
		t.Fatalf("MarkReinforcementArrived failed: %v", err)
	}

	started, err := svc.StartPvpAttack(PvpAttackRequest{
		PlayerID: attacker.Player.ID, TargetPlayerID: defender.Player.ID, MarchMode: PvpMarchTypeAttack,
		Troops: map[string]int{"shuInfantry": 110}, GeneralIDs: []string{"liubei"},
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
	helperReports, total, err := repo.ListReportsByQuery(BattleReportQuery{PlayerID: helper.Player.ID, ViewType: ReportViewReinforcement, Page: 1, PageSize: 10})
	if err != nil || total != 1 || len(helperReports) != 1 {
		t.Fatalf("expected one helper report, reports=%+v total=%d err=%v", helperReports, total, err)
	}
	record, err := repo.GetReinforcement(sent.Reinforcement.ID)
	if err != nil {
		t.Fatalf("GetReinforcement failed: %v", err)
	}
	helperState, err := repo.GetState(helper.Player.ID)
	if err != nil {
		t.Fatalf("GetState helper failed: %v", err)
	}
	return xiahouyuanReinforcementComboResult{
		sent: sent.Reinforcement, battle: battle, record: record,
		attackerReport: attackerReports[0], defenderReport: defenderReports[0], helperReport: helperReports[0], helperState: helperState,
	}
}

// TestXiahouyuanMarchAndDefenseTraitsReconcileInRealReinforcement 验证行军加速只作用过程，盾阵防御才进入真实战斗和三方战报。
func TestXiahouyuanMarchAndDefenseTraitsReconcileInRealReinforcement(t *testing.T) {
	control := runXiahouyuanReinforcementCombo(t, false)
	active := runXiahouyuanReinforcementCombo(t, true)
	wantMarchSeconds := applyExpectedMarchRates(control.sent.MarchSeconds, []float64{0.2}, 60)
	wantSpeed := control.sent.SpeedMultiplier * float64(control.sent.MarchSeconds) / float64(wantMarchSeconds)
	if active.sent.MarchSeconds != wantMarchSeconds || active.sent.ReturnSeconds != wantMarchSeconds || math.Abs(active.sent.SpeedMultiplier-wantSpeed) > 1e-9 {
		t.Fatalf("expected Jixing duration %d and speed %.6f, control=%+v active=%+v", wantMarchSeconds, wantSpeed, control.sent, active.sent)
	}
	attackPower, attackOK := active.battle.Result["attackerPower"].(float64)
	defensePower, defenseOK := active.battle.Result["defensePower"].(float64)
	if !attackOK || !defenseOK || attackPower != 1100 || defensePower != 1410 || active.battle.Result["winner"] != "defender" {
		t.Fatalf("expected shield defense to produce 1100/1410 defender victory, result=%+v", active.battle.Result)
	}
	controlDefensePower, controlDefenseOK := control.battle.Result["defensePower"].(float64)
	if !controlDefenseOK || controlDefensePower != 1010 {
		t.Fatalf("expected no-trait defense baseline 1010, result=%+v", control.battle.Result)
	}
	if active.record.RemainingTroops["huWei"] != 30 || active.record.Losses["huWei"] != 70 || active.record.Status != ReinforcementStatusStationed {
		t.Fatalf("expected active reinforcement 100/70/30 and still stationed, record=%+v", active.record)
	}
	if control.record.RemainingTroops["huWei"] != 0 || control.record.Losses["huWei"] != 100 || !control.record.IsAnnihilated {
		t.Fatalf("expected no-trait reinforcement to be annihilated, record=%+v", control.record)
	}

	for _, report := range []BattleReport{active.attackerReport, active.defenderReport, active.helperReport} {
		if len(report.TraitTriggered) != 1 || report.TraitTriggered[0] != "dunzhen_fangyu" || len(report.TraitOutcomes) != 1 {
			t.Fatalf("expected only shield defense in battle timeline, report=%s timeline=%v outcomes=%+v", report.ID, report.TraitTriggered, report.TraitOutcomes)
		}
		if _, exists := report.TraitOutcomes["jixing_benxi"]; exists || standardReportHasTrait(report.Detail, "jixing_benxi") {
			t.Fatalf("expected march process trait absent from battle timeline, report=%s outcomes=%+v detail=%+v", report.ID, report.TraitOutcomes, report.Detail)
		}
		outcome := report.TraitOutcomes["dunzhen_fangyu"]
		infantry, infantryOK := outcome.Detail["infantryDefenseModifiedUnits"].(map[string]int)
		cavalry, cavalryOK := outcome.Detail["cavalryDefenseModifiedUnits"].(map[string]int)
		rate, rateOK := outcome.Detail["defenseBonusRate"].(float64)
		if !infantryOK || !cavalryOK || !rateOK || rate != 0.35 || infantry["huWei"] != 4 || cavalry["huWei"] != 3 || outcome.OwnerSide != "reinforcement" || outcome.OwnerGeneralID != "xiahouyuan" {
			t.Fatalf("expected reinforcement-owned shield deltas +4/+3, report=%s outcome=%+v", report.ID, outcome)
		}
		if report.Detail == nil || len(report.Detail.Traits) != 1 || report.Detail.Traits[0].TraitID != "dunzhen_fangyu" || report.Detail.Traits[0].OwnerRole != "reinforcement" {
			t.Fatalf("expected standard timeline to contain only reinforcement shield defense, report=%s detail=%+v", report.ID, report.Detail)
		}
		if len(report.PvpReinforcements) != 1 || len(report.PvpReinforcements[0].Generals) != 1 || report.PvpReinforcements[0].Generals[0].ID != "xiahouyuan" {
			t.Fatalf("expected Xiahou Yuan reinforcement snapshot, report=%s reinforcements=%+v", report.ID, report.PvpReinforcements)
		}
		ownedTraits := map[string]bool{}
		for _, trait := range report.PvpReinforcements[0].Generals[0].Traits {
			ownedTraits[trait.TraitID] = true
		}
		if !ownedTraits["jixing_benxi"] || !ownedTraits["dunzhen_fangyu"] {
			t.Fatalf("expected snapshot to retain both owned traits, report=%s traits=%+v", report.ID, report.PvpReinforcements[0].Generals[0].Traits)
		}
		if report.PvpReinforcementLosses[active.sent.ID]["huWei"] != 70 {
			t.Fatalf("expected three reports to use reinforcement loss 70, report=%s losses=%+v", report.ID, report.PvpReinforcementLosses)
		}
		if report.PvpReinforcements[0].GeneralExpGained != 110 || report.PvpReinforcements[0].GeneralLevelBefore != 1 || report.PvpReinforcements[0].GeneralLevelAfter != 2 {
			t.Fatalf("expected authoritative reinforcement progress 110 and level 1 -> 2, report=%s reinforcement=%+v", report.ID, report.PvpReinforcements[0])
		}
	}
	if active.helperReport.DispatchedUnits["huWei"] != 100 || active.helperReport.LostUnits["huWei"] != 70 || active.helperReport.SurvivedUnits["huWei"] != 30 {
		t.Fatalf("expected helper legacy row 100/70/30, report=%+v", active.helperReport)
	}
	if active.helperReport.Detail == nil || active.helperReport.Detail.SecondarySide == nil || active.helperReport.Detail.PrimarySide.Role != "attacker" || active.helperReport.Detail.SecondarySide.Role != "defender" {
		t.Fatalf("expected helper standard detail to preserve complete attacker and defender snapshots, detail=%+v", active.helperReport.Detail)
	}
	baselineExp := generalExpRequiredForLevelForTest(2) - 1
	if active.helperReport.GeneralExpGained != 110 || active.helperReport.GeneralLevelBefore != 1 || active.helperReport.GeneralLevelAfter != 2 || active.helperReport.Detail.Rewards.GeneralLevelBefore != 1 || active.helperReport.Detail.Rewards.GeneralLevelAfter != 2 || pvpTestGeneralExp(active.helperState, "xiahouyuan") != baselineExp+110 || pvpTestGeneralLevel(active.helperState, "xiahouyuan") != 2 {
		t.Fatalf("expected Xiahou Yuan exp 110 in report snapshot and owner state, report=%+v state=%+v", active.helperReport, active.helperState.Generals)
	}
	if active.record.Generals[0].Exp != baselineExp+110 || active.record.Generals[0].Level != 2 {
		t.Fatalf("expected reinforcement record to advance to level 2 and cumulative exp %d, got %+v", baselineExp+110, active.record.Generals[0])
	}
}
