// 本文件验证援军将领的对敌特性进入真实 PVP 战力、扣兵、压制顺序和三方战报。
package game

import (
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"testing"
	"time"

	"hero3/internal/core/general"
)

type reinforcementEnemyPvpConfig struct {
	id              string
	attackerTroops  int
	attackerUnits   map[string]int
	defenderTroops  int
	marchMode       string
	helperFaction   string
	helperGeneralID string
	helperName      string
	helperTroops    int
	helperSpecial   GeneralTraitConfig
	helperBonus     GeneralTraitConfig
	attackerFaction string
	attackerGeneral string
	attackerName    string
	attackerSpecial GeneralTraitConfig
	attackerBonus   GeneralTraitConfig
	defenderFaction string
	defenderGeneral string
	defenderName    string
	defenderSpecial GeneralTraitConfig
	defenderBonus   GeneralTraitConfig
}

type reinforcementEnemyPvpResult struct {
	battle              PvpBattle
	storedMarch         PvpMarch
	attackerUnit        string
	attackerLosses      int
	defendingLosses     int
	reinforcementID     string
	storedReinforcement Reinforcement
	defenderState       GameState
	helperState         GameState
	attackerReport      BattleReport
	defenderReport      BattleReport
	reinforcementReport BattleReport
}

// totalPvpDefendingLosses 汇总主城守军和全部援军的真实损失。
func totalPvpDefendingLosses(t *testing.T, battle PvpBattle) int {
	t.Helper()
	total := 0
	for _, amount := range pvpTestLossesFromBattle(t, battle, "defender") {
		total += amount
	}
	reinforcements, ok := battle.Losses["reinforcements"].(map[string]map[string]int)
	if !ok {
		t.Fatalf("expected reinforcement losses map, got %+v", battle.Losses["reinforcements"])
	}
	for _, losses := range reinforcements {
		for _, amount := range losses {
			total += amount
		}
	}
	return total
}

// runReinforcementEnemyPvp 执行一场带援军对敌特性的完整 PVP 防守事务。
func runReinforcementEnemyPvp(t *testing.T, cfg reinforcementEnemyPvpConfig) reinforcementEnemyPvpResult {
	t.Helper()
	heroes := map[string]GeneralHeroConfig{
		cfg.attackerGeneral: {
			ID: cfg.attackerGeneral, Name: cfg.attackerName, Faction: cfg.attackerFaction, Enabled: true,
			SpecialTrait: cfg.attackerSpecial, BonusTrait: cfg.attackerBonus,
		},
		cfg.defenderGeneral: {
			ID: cfg.defenderGeneral, Name: cfg.defenderName, Faction: cfg.defenderFaction, Enabled: true,
			SpecialTrait: cfg.defenderSpecial, BonusTrait: cfg.defenderBonus,
		},
		cfg.helperGeneralID: {
			ID: cfg.helperGeneralID, Name: cfg.helperName, Faction: cfg.helperFaction, Enabled: true,
			SpecialTrait: cfg.helperSpecial, BonusTrait: cfg.helperBonus,
		},
	}
	factions := FactionsConfig{}
	for _, item := range []struct{ faction, id, name string }{
		{cfg.attackerFaction, cfg.attackerGeneral, cfg.attackerName},
		{cfg.defenderFaction, cfg.defenderGeneral, cfg.defenderName},
		{cfg.helperFaction, cfg.helperGeneralID, cfg.helperName},
	} {
		entry := factions[item.faction]
		entry.Name = item.faction
		entry.Generals = append(entry.Generals, GeneralInfo{ID: item.id, Name: item.name})
		factions[item.faction] = entry
	}
	setTestFactionsAndGenerals(t, factions, GeneralsConfig{Enabled: true, Heroes: heroes})
	svc, repo, attacker, defender := newPvpTestServiceForGenerals(
		t, cfg.attackerFaction, cfg.attackerGeneral, cfg.defenderFaction, cfg.defenderGeneral,
	)
	helperUnit := cfg.helperFaction + "Infantry"
	unitsMu.Lock()
	if activeUnits[cfg.helperFaction] == nil {
		activeUnits[cfg.helperFaction] = FactionUnits{}
	}
	if _, exists := activeUnits[cfg.helperFaction][helperUnit]; !exists {
		activeUnits[cfg.helperFaction][helperUnit] = UnitConfig{
			Name: cfg.helperName + "测试步兵", Category: "infantry",
			Stats: map[string]int{"attack": 10, "infantryDefense": 10, "cavalryDefense": 8, "carryCapacity": 5, "upkeep": 1},
		}
	}
	unitsMu.Unlock()
	now := time.Now().UTC()
	helperAccount := Account{ID: "account_enemy_" + cfg.id, Username: "enemy_" + cfg.id, PasswordHash: "hash", CreatedAt: now}
	if err := repo.CreateAccount(helperAccount); err != nil {
		t.Fatalf("CreateAccount helper failed: %v", err)
	}
	helper := newPlayerState("player_enemy_"+cfg.id, cfg.helperName+"援军", cfg.helperFaction, cfg.helperGeneralID, now)
	helperTroops := cfg.helperTroops
	if helperTroops <= 0 {
		helperTroops = 100
	}
	helper.Army = []ArmyUnit{{UnitType: helperUnit, Amount: helperTroops}}
	if err := repo.CreatePlayer(helperAccount.ID, helper, now); err != nil {
		t.Fatalf("CreatePlayer helper failed: %v", err)
	}
	attackerUnit := cfg.attackerFaction + "Infantry"
	defenderUnit := cfg.defenderFaction + "Infantry"
	attackerTroops := cfg.attackerTroops
	if attackerTroops <= 0 {
		attackerTroops = 110
	}
	attackTroops := cloneStringIntMap(cfg.attackerUnits)
	if len(attackTroops) == 0 {
		attackTroops = map[string]int{attackerUnit: attackerTroops}
	}
	defenderTroops := cfg.defenderTroops
	if defenderTroops <= 0 {
		defenderTroops = 1
	}
	marchMode := cfg.marchMode
	if marchMode == "" {
		marchMode = PvpMarchTypeAttack
	}
	attacker.Army = armyMapToSlice(attackTroops)
	defender.Army = []ArmyUnit{{UnitType: defenderUnit, Amount: defenderTroops}}
	defender.Buildings = nil
	attackerTraits := buildActiveTraitsForGeneralIDs(&attacker, []string{cfg.attackerGeneral})
	if cfg.attackerBonus.Enabled {
		found := false
		for _, trait := range attackerTraits {
			found = found || trait.TraitID == cfg.attackerBonus.TraitID
		}
		if !found {
			t.Fatalf("expected enabled attacker trait %s for %s, active=%+v roster=%+v config=%+v", cfg.attackerBonus.TraitID, cfg.attackerGeneral, attackerTraits, attacker.Generals, GetGeneralsConfig().Heroes[cfg.attackerGeneral])
		}
	}
	repo.players[attacker.Player.ID] = attacker
	repo.players[defender.Player.ID] = defender

	sent, err := svc.SendReinforcement(SendReinforcementRequest{
		FromPlayerID: helper.Player.ID, TargetPlayerID: defender.Player.ID,
		Troops: map[string]int{helperUnit: helperTroops}, GeneralIDs: []string{cfg.helperGeneralID},
	})
	if err != nil {
		t.Fatalf("SendReinforcement failed: %v", err)
	}
	forceReinforcementDue(t, repo, sent.Reinforcement.ID, true)
	if _, err := svc.MarkReinforcementArrived(sent.Reinforcement.ID); err != nil {
		t.Fatalf("MarkReinforcementArrived failed: %v", err)
	}
	started, err := svc.StartPvpAttack(PvpAttackRequest{
		PlayerID: attacker.Player.ID, TargetPlayerID: defender.Player.ID, MarchMode: marchMode,
		Troops: cloneStringIntMap(attackTroops), GeneralIDs: []string{cfg.attackerGeneral},
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
	helperReports, total, err := repo.ListReportsByQuery(BattleReportQuery{
		PlayerID: helper.Player.ID, ViewType: ReportViewReinforcement, Page: 1, PageSize: 10,
	})
	if err != nil || total != 1 || len(helperReports) != 1 {
		t.Fatalf("expected reinforcement report, reports=%+v total=%d err=%v", helperReports, total, err)
	}
	storedReinforcement, err := repo.GetReinforcement(sent.Reinforcement.ID)
	if err != nil {
		t.Fatalf("GetReinforcement failed: %v", err)
	}
	defenderState, err := repo.GetState(defender.Player.ID)
	if err != nil {
		t.Fatalf("GetState defender failed: %v", err)
	}
	helperState, err := repo.GetState(helper.Player.ID)
	if err != nil {
		t.Fatalf("GetState helper failed: %v", err)
	}
	storedMarch, err := repo.GetPvpMarch(started.March.ID)
	if err != nil {
		t.Fatalf("GetPvpMarch failed: %v", err)
	}
	return reinforcementEnemyPvpResult{
		battle: battle, storedMarch: storedMarch, attackerUnit: attackerUnit,
		attackerLosses:  pvpTestLossesFromBattle(t, battle, "attacker")[attackerUnit],
		defendingLosses: totalPvpDefendingLosses(t, battle),
		reinforcementID: sent.Reinforcement.ID, storedReinforcement: storedReinforcement,
		defenderState: defenderState, helperState: helperState,
		attackerReport: attackerReports[0], defenderReport: defenderReports[0], reinforcementReport: helperReports[0],
	}
}

// reinforcementOutcome 返回指定援军玩家的正式特性结果。
func reinforcementOutcome(report BattleReport, traitID string, playerID string) (TraitOutcomeReport, bool) {
	for _, outcome := range report.TraitOutcomes {
		if outcome.TraitID == traitID && outcome.OwnerSide == "reinforcement" && outcome.OwnerPlayerID == playerID {
			return outcome, true
		}
	}
	return TraitOutcomeReport{}, false
}

// TestReinforcementEnemyPreBattleTraitChangesRealAttackPower 验证张辽两项主动进攻特性作为援军时即使概率强制命中也不得生效。
func TestReinforcementEnemyPreBattleTraitChangesRealAttackPower(t *testing.T) {
	id := "zhangliao_reinforcement_wrong_direction"
	result := runReinforcementEnemyPvp(t, reinforcementEnemyPvpConfig{
		id: id, attackerTroops: 100, defenderTroops: 1, marchMode: PvpMarchTypePlunder,
		helperFaction: "wei", helperGeneralID: "zhangliao", helperName: "张辽", helperTroops: 99,
		attackerFaction: "shu", attackerGeneral: "liubei", attackerName: "刘备",
		defenderFaction: "wu", defenderGeneral: "sunquan", defenderName: "孙权",
		helperSpecial: GeneralTraitConfig{
			TraitID: "weizhen_zhenhe", TraitType: general.TraitTypeSpecial, Enabled: true, Scope: "enemy_army", AllowedSides: []string{"attacker"},
			Params: map[string]float64{"triggerChance": 1, "effectRate": 0.25},
		},
		helperBonus: GeneralTraitConfig{
			TraitID: "weizhen_xiaoyao", TraitType: general.TraitTypeBonus, Enabled: true, Scope: "self_army", TargetUnitType: "cavalry", AllowedSides: []string{"attacker"},
			Params: map[string]float64{"triggerChance": 1, "attackBonusRate": 0.35},
		},
	})
	if result.battle.Result["winner"] != "draw" || result.battle.Result["attackerPower"] != float64(1000) || result.battle.Result["defensePower"] != float64(1000) {
		t.Fatalf("expected reinforcement direction to keep baseline draw 1000/1000, result=%+v", result.battle.Result)
	}
	if result.attackerLosses != 50 || result.defendingLosses != 50 || result.defenderReport.LostUnits["wuInfantry"] != 1 || result.storedReinforcement.Losses["weiInfantry"] != 49 {
		t.Fatalf("expected baseline attacker/main/reinforcement losses 50/1/49, battle=%+v main=%+v reinforcement=%+v", result.battle.Losses, result.defenderReport.LostUnits, result.storedReinforcement)
	}
	for _, report := range []BattleReport{result.attackerReport, result.defenderReport, result.reinforcementReport} {
		if len(report.PvpReinforcements) != 1 || len(report.PvpReinforcements[0].Generals) != 1 || !reinforcementSnapshotHasTrait(report.PvpReinforcements[0].Generals[0], "weizhen_zhenhe") || !reinforcementSnapshotHasTrait(report.PvpReinforcements[0].Generals[0], "weizhen_xiaoyao") {
			t.Fatalf("expected Zhang Liao snapshot to retain both traits, report=%s snapshots=%+v", report.ID, report.PvpReinforcements)
		}
		if len(report.TraitTriggered) != 0 || len(report.TraitOutcomes) != 0 || report.Detail == nil || len(report.Detail.Traits) != 0 {
			t.Fatalf("expected both attacker-only traits excluded for reinforcement, report=%+v", report)
		}
	}
}

// TestPvpLuxunReinforcementHuoshaoHitAndMissKeepRealTraitOrder 验证陆逊援军火烧命中时不伪造零变化连营，未命中时连营仍独立追加真实损失。
func TestPvpLuxunReinforcementHuoshaoHitAndMissKeepRealTraitOrder(t *testing.T) {
	cases := []struct {
		name               string
		triggerChance      float64
		wantTraitID        string
		wantAttackerLosses int
		wantExtraLosses    int
	}{
		{name: "火烧命中烧完剩余步兵", triggerChance: 1, wantTraitID: "huoshao_lianying", wantAttackerLosses: 100, wantExtraLosses: 50},
		{name: "火烧未命中保留连营增伤", triggerChance: 0, wantTraitID: "lianying_zengshang", wantAttackerLosses: 60, wantExtraLosses: 10},
	}
	for index, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			id := "luxun_reinforcement_" + string(rune('a'+index))
			result := runReinforcementEnemyPvp(t, reinforcementEnemyPvpConfig{
				id: id, attackerTroops: 100, defenderTroops: 1, marchMode: PvpMarchTypePlunder,
				helperFaction: "wu", helperGeneralID: "luxun", helperName: "陆逊", helperTroops: 99,
				attackerFaction: "wei", attackerGeneral: "caocao", attackerName: "曹操",
				defenderFaction: "shu", defenderGeneral: "liubei", defenderName: "刘备",
				helperSpecial: GeneralTraitConfig{
					TraitID: "huoshao_lianying", TraitType: general.TraitTypeSpecial, Enabled: true, Scope: "enemy_army", TargetUnitType: "infantry",
					Params: map[string]float64{"triggerChance": tc.triggerChance, "effectRate": 1, "maxAffectedRate": 1},
				},
				helperBonus: GeneralTraitConfig{
					TraitID: "lianying_zengshang", TraitType: general.TraitTypeBonus, Enabled: true, Scope: "enemy_army", TargetUnitType: "infantry",
					Params: map[string]float64{"effectRate": 0.1},
				},
			})
			if result.battle.Result["winner"] != "draw" || result.battle.Result["attackerPower"] != float64(1000) || result.battle.Result["defensePower"] != float64(1000) {
				t.Fatalf("expected equal 1000/1000 core draw, result=%+v", result.battle.Result)
			}
			if result.attackerLosses != tc.wantAttackerLosses || result.defendingLosses != 50 || result.defenderReport.LostUnits["shuInfantry"] != 1 || result.storedReinforcement.Losses["wuInfantry"] != 49 {
				t.Fatalf("expected attacker/main/reinforcement losses %d/1/49, battle=%+v main=%+v reinforcement=%+v", tc.wantAttackerLosses, result.battle.Losses, result.defenderReport.LostUnits, result.storedReinforcement)
			}
			if result.attackerReport.SurvivedUnits["weiInfantry"] != 100-tc.wantAttackerLosses || armySliceToMap(result.defenderState.Army)["shuInfantry"] != 0 || result.storedReinforcement.RemainingTroops["wuInfantry"] != 50 {
				t.Fatalf("expected authoritative attacker/main/reinforcement survivors %d/0/50, attacker=%+v defender=%+v reinforcement=%+v", 100-tc.wantAttackerLosses, result.attackerReport.SurvivedUnits, result.defenderState.Army, result.storedReinforcement)
			}
			if result.attackerReport.GeneralExpGained != 50 || result.defenderReport.GeneralExpGained != tc.wantAttackerLosses || result.reinforcementReport.GeneralExpGained != tc.wantAttackerLosses || pvpTestGeneralExp(result.helperState, "luxun") != tc.wantAttackerLosses {
				t.Fatalf("expected attacker/main/helper exp 50/%d/%d, reports=%d/%d/%d helper=%+v", tc.wantAttackerLosses, tc.wantAttackerLosses, result.attackerReport.GeneralExpGained, result.defenderReport.GeneralExpGained, result.reinforcementReport.GeneralExpGained, result.helperState.Generals)
			}

			for _, report := range []BattleReport{result.attackerReport, result.defenderReport, result.reinforcementReport} {
				if len(report.TraitTriggered) != 1 || len(report.TraitOutcomes) != 1 || len(report.Detail.Traits) != 1 {
					t.Fatalf("expected exactly one real reinforcement outcome after zero-change filtering, report=%+v", report)
				}
				outcome, triggered := reinforcementOutcome(report, tc.wantTraitID, "player_enemy_"+id)
				extra, extraOK := outcome.Detail["targetExtraLosses"].(map[string]int)
				if !triggered || !extraOK || extra["weiInfantry"] != tc.wantExtraLosses || outcome.OwnerSide != "reinforcement" || outcome.OwnerGeneralID != "luxun" {
					t.Fatalf("expected reinforcement-owned %s actual extra loss %d, report=%s outcome=%+v", tc.wantTraitID, tc.wantExtraLosses, report.ID, outcome)
				}
				trait := report.Detail.Traits[0]
				standardExtra, standardExtraOK := trait.Detail["targetExtraLosses"].(map[string]int)
				if trait.TraitID != tc.wantTraitID || trait.OwnerRole != "reinforcement" || trait.OwnerPlayerID != "player_enemy_"+id || !standardExtraOK || standardExtra["weiInfantry"] != tc.wantExtraLosses {
					t.Fatalf("expected one standard reinforcement timeline with actual extra loss %d, report=%s trait=%+v", tc.wantExtraLosses, report.ID, trait)
				}
				if len(report.PvpReinforcements) != 1 || len(report.PvpReinforcements[0].Generals) != 1 || !reinforcementSnapshotHasTrait(report.PvpReinforcements[0].Generals[0], "huoshao_lianying") || !reinforcementSnapshotHasTrait(report.PvpReinforcements[0].Generals[0], "lianying_zengshang") {
					t.Fatalf("expected every report to retain both Luxun ownership traits, report=%s reinforcements=%+v", report.ID, report.PvpReinforcements)
				}
				if report.PvpReinforcementLosses[result.reinforcementID]["wuInfantry"] != 49 {
					t.Fatalf("expected every report to retain reinforcement loss 49, report=%s losses=%+v", report.ID, report.PvpReinforcementLosses)
				}
			}
			if tc.wantTraitID == "huoshao_lianying" {
				for _, report := range []BattleReport{result.attackerReport, result.defenderReport, result.reinforcementReport} {
					if _, exists := report.TraitOutcomes["lianying_zengshang"]; exists || standardReportHasTrait(report.Detail, "lianying_zengshang") {
						t.Fatalf("expected zero-change Lianying absent after Huoshao annihilates target, report=%+v", report)
					}
				}
			} else {
				for _, report := range []BattleReport{result.attackerReport, result.defenderReport, result.reinforcementReport} {
					if _, exists := report.TraitOutcomes["huoshao_lianying"]; exists || standardReportHasTrait(report.Detail, "huoshao_lianying") {
						t.Fatalf("expected missed Huoshao absent while Lianying remains, report=%+v", report)
					}
				}
			}
		})
	}
}

// TestPvpSimayiReinforcementTraitsIndependentCombinations 验证司马懿援军两项战前特性独立判定并进入真实结算。
func TestPvpSimayiReinforcementTraitsIndependentCombinations(t *testing.T) {
	cases := []struct {
		name              string
		yibingChance      float64
		moudingChance     float64
		wantAttackerPower float64
		wantDefensePower  float64
		wantWinner        string
		wantYibing        bool
		wantMouding       bool
	}{
		{name: "两项同时命中", yibingChance: 1, moudingChance: 1, wantAttackerPower: 650, wantDefensePower: 1396, wantWinner: "defender", wantYibing: true, wantMouding: true},
		{name: "仅疑兵命中", yibingChance: 1, moudingChance: 0, wantAttackerPower: 650, wantDefensePower: 1000, wantWinner: "defender", wantYibing: true},
		{name: "仅谋定命中", yibingChance: 0, moudingChance: 1, wantAttackerPower: 1000, wantDefensePower: 1396, wantWinner: "defender", wantMouding: true},
		{name: "两项均未命中", yibingChance: 0, moudingChance: 0, wantAttackerPower: 1000, wantDefensePower: 1000, wantWinner: "draw"},
	}
	for index, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			id := "simayi_reinforcement_" + string(rune('a'+index))
			result := runReinforcementEnemyPvp(t, reinforcementEnemyPvpConfig{
				id: id, attackerTroops: 100, defenderTroops: 1, marchMode: PvpMarchTypePlunder,
				helperFaction: "wei", helperGeneralID: "simayi", helperName: "司马懿", helperTroops: 99,
				attackerFaction: "shu", attackerGeneral: "liubei", attackerName: "刘备",
				defenderFaction: "wu", defenderGeneral: "sunquan", defenderName: "孙权",
				helperSpecial: GeneralTraitConfig{
					TraitID: "yibing_touxi", TraitType: general.TraitTypeSpecial, Enabled: true, Scope: "enemy_army",
					Params: map[string]float64{"triggerChance": tc.yibingChance, "effectRate": 0.35},
				},
				helperBonus: GeneralTraitConfig{
					TraitID: "mouding_houfa", TraitType: general.TraitTypeBonus, Enabled: true, Scope: "self_army", AllowedSides: []string{"defender", "reinforcement"},
					Params: map[string]float64{"defenseBonusRate": 0.35, "triggerChance": tc.moudingChance},
				},
			})
			if result.battle.Result["winner"] != tc.wantWinner || result.battle.Result["attackerPower"] != tc.wantAttackerPower || result.battle.Result["defensePower"] != tc.wantDefensePower {
				t.Fatalf("expected exact winner and power %s %.0f/%.0f, result=%+v", tc.wantWinner, tc.wantAttackerPower, tc.wantDefensePower, result.battle.Result)
			}
			mainLosses := result.defenderReport.LostUnits["wuInfantry"]
			reinforcementLosses := result.storedReinforcement.Losses["weiInfantry"]
			if result.attackerLosses <= 0 || result.defendingLosses <= 0 || result.defendingLosses != mainLosses+reinforcementLosses {
				t.Fatalf("expected positive reconciled coalition losses, battle=%+v main=%+v reinforcement=%+v", result.battle.Losses, result.defenderReport.LostUnits, result.storedReinforcement)
			}
			if result.attackerReport.SurvivedUnits["shuInfantry"] != 100-result.attackerLosses || armySliceToMap(result.defenderState.Army)["wuInfantry"] != 1-mainLosses || result.storedReinforcement.RemainingTroops["weiInfantry"] != 99-reinforcementLosses {
				t.Fatalf("expected authoritative state to match battle losses, attacker=%+v defender=%+v reinforcement=%+v", result.attackerReport.SurvivedUnits, result.defenderState.Army, result.storedReinforcement)
			}
			if result.attackerReport.GeneralExpGained != result.defendingLosses || result.defenderReport.GeneralExpGained != result.attackerLosses || result.reinforcementReport.GeneralExpGained != result.attackerLosses || pvpTestGeneralExp(result.helperState, "simayi") != result.attackerLosses {
				t.Fatalf("expected report and general exp to follow real losses, reports=%d/%d/%d helper=%+v", result.attackerReport.GeneralExpGained, result.defenderReport.GeneralExpGained, result.reinforcementReport.GeneralExpGained, result.helperState.Generals)
			}

			for _, report := range []BattleReport{result.attackerReport, result.defenderReport, result.reinforcementReport} {
				if len(report.PvpReinforcements) != 1 || len(report.PvpReinforcements[0].Generals) != 1 || !reinforcementSnapshotHasTrait(report.PvpReinforcements[0].Generals[0], "yibing_touxi") || !reinforcementSnapshotHasTrait(report.PvpReinforcements[0].Generals[0], "mouding_houfa") {
					t.Fatalf("expected reinforcement snapshot to retain both owned traits, report=%s snapshots=%+v", report.ID, report.PvpReinforcements)
				}
				wantTraitCount := 0
				if tc.wantYibing {
					wantTraitCount++
				}
				if tc.wantMouding {
					wantTraitCount++
				}
				if len(report.TraitTriggered) != wantTraitCount || len(report.TraitOutcomes) != wantTraitCount || len(report.Detail.Traits) != wantTraitCount {
					t.Fatalf("expected %d independently triggered traits, report=%+v", wantTraitCount, report)
				}
				yibing, yibingTriggered := reinforcementOutcome(report, "yibing_touxi", "player_enemy_"+id)
				if yibingTriggered != tc.wantYibing || standardReportHasTrait(report.Detail, "yibing_touxi") != tc.wantYibing {
					t.Fatalf("expected Yibing triggered=%v, report=%+v", tc.wantYibing, report)
				}
				if tc.wantYibing {
					preBattleAffected, detailOK := yibing.Detail["preBattleAffected"].(map[string]int)
					if !detailOK || preBattleAffected["shuInfantry"] != 35 || yibing.OwnerSide != "reinforcement" || yibing.OwnerGeneralID != "simayi" {
						t.Fatalf("expected reinforcement-owned Yibing to remove 35 original troops, report=%s outcome=%+v", report.ID, yibing)
					}
				}
				mouding, moudingTriggered := reinforcementOutcome(report, "mouding_houfa", "player_enemy_"+id)
				if moudingTriggered != tc.wantMouding || standardReportHasTrait(report.Detail, "mouding_houfa") != tc.wantMouding {
					t.Fatalf("expected Mouding triggered=%v, report=%+v", tc.wantMouding, report)
				}
				if tc.wantMouding {
					infantry, infantryOK := mouding.Detail["infantryDefenseModifiedUnits"].(map[string]int)
					cavalry, cavalryOK := mouding.Detail["cavalryDefenseModifiedUnits"].(map[string]int)
					if !infantryOK || !cavalryOK || infantry["weiInfantry"] != 4 || cavalry["weiInfantry"] != 3 || mouding.Detail["defenseBonusRate"] != 0.35 || mouding.OwnerSide != "reinforcement" {
						t.Fatalf("expected reinforcement-owned Mouding +4/+3 defense, report=%s outcome=%+v", report.ID, mouding)
					}
				}
				for _, trait := range report.Detail.Traits {
					if trait.OwnerRole != "reinforcement" || trait.OwnerPlayerID != "player_enemy_"+id || trait.GeneralID != "simayi" {
						t.Fatalf("expected standard timeline ownership to remain on Sima Yi reinforcement, report=%s trait=%+v", report.ID, trait)
					}
				}
			}
		})
	}
}

// TestPvpMachaoReinforcementXiliangHitAndMissKeepPassiveOutOfTimeline 验证马超援军西凉突击只追加骑兵损失，被动武力不改变防御或伪装成触发。
func TestPvpMachaoReinforcementXiliangHitAndMissKeepPassiveOutOfTimeline(t *testing.T) {
	originalUnits := GetUnitsConfig()
	unitsMu.Lock()
	if activeUnits["wei"] == nil {
		activeUnits["wei"] = FactionUnits{}
	}
	activeUnits["wei"]["weiCavalry"] = UnitConfig{
		Name: "魏测试骑兵", Category: "cavalry",
		Stats: map[string]int{"attack": 10, "infantryDefense": 10, "cavalryDefense": 10, "carryCapacity": 5, "upkeep": 1},
	}
	unitsMu.Unlock()
	t.Cleanup(func() {
		unitsMu.Lock()
		activeUnits = originalUnits
		unitsMu.Unlock()
	})
	cases := []struct {
		name               string
		triggerChance      float64
		wantCavalryLosses  int
		wantDefenderExp    int
		wantTraitTriggered bool
	}{
		{name: "西凉命中只追加来袭骑兵损失", triggerChance: 1, wantCavalryLosses: 26, wantDefenderExp: 71, wantTraitTriggered: true},
		{name: "西凉未命中保持两类核心损失", triggerChance: 0, wantCavalryLosses: 20, wantDefenderExp: 59},
	}
	for index, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			id := "machao_reinforcement_" + string(rune('a'+index))
			result := runReinforcementEnemyPvp(t, reinforcementEnemyPvpConfig{
				id: id, attackerUnits: map[string]int{"weiInfantry": 50, "weiCavalry": 50}, defenderTroops: 1, marchMode: PvpMarchTypePlunder,
				helperFaction: "shu", helperGeneralID: "machao", helperName: "马超", helperTroops: 99,
				attackerFaction: "wei", attackerGeneral: "caocao", attackerName: "曹操",
				defenderFaction: "wu", defenderGeneral: "sunquan", defenderName: "孙权",
				helperSpecial: GeneralTraitConfig{
					TraitID: "xiliang_tuji", TraitType: general.TraitTypeSpecial, Enabled: true, Scope: "enemy_army", TargetUnitType: "cavalry",
					Params: map[string]float64{"triggerChance": tc.triggerChance, "effectRate": 0.12},
				},
				helperBonus: GeneralTraitConfig{
					TraitID: "tianshen_xiafan", TraitType: general.TraitTypeBonus, Enabled: true, Scope: "self_army",
					Params: map[string]float64{"forceBonus": 20},
				},
			})
			if result.battle.Result["winner"] != "attacker" || result.battle.Result["attackerPower"] != float64(1200) || math.Abs(result.battle.Result["defensePower"].(float64)-883.3333333333334) > 1e-9 {
				t.Fatalf("expected formal mixed-unit baseline power 1200/883.33 and attacker victory, result=%+v", result.battle.Result)
			}
			attackerLosses := pvpTestLossesFromBattle(t, result.battle, "attacker")
			if attackerLosses["weiInfantry"] != 19 || attackerLosses["weiCavalry"] != tc.wantCavalryLosses || result.defendingLosses != 60 || result.defenderReport.LostUnits["wuInfantry"] != 0 || result.storedReinforcement.Losses["shuInfantry"] != 60 {
				t.Fatalf("expected infantry/cavalry/main/reinforcement losses 19/%d/0/60, battle=%+v main=%+v reinforcement=%+v", tc.wantCavalryLosses, result.battle.Losses, result.defenderReport.LostUnits, result.storedReinforcement)
			}
			if result.attackerReport.SurvivedUnits["weiInfantry"] != 31 || result.attackerReport.SurvivedUnits["weiCavalry"] != 50-tc.wantCavalryLosses || armySliceToMap(result.defenderState.Army)["wuInfantry"] != 1 || result.storedReinforcement.RemainingTroops["shuInfantry"] != 39 {
				t.Fatalf("expected authoritative infantry/cavalry/main/reinforcement survivors 31/%d/1/39, attacker=%+v defender=%+v reinforcement=%+v", 50-tc.wantCavalryLosses, result.attackerReport.SurvivedUnits, result.defenderState.Army, result.storedReinforcement)
			}
			if result.attackerReport.GeneralExpGained != 60 || result.defenderReport.GeneralExpGained != tc.wantDefenderExp || result.reinforcementReport.GeneralExpGained != tc.wantDefenderExp || pvpTestGeneralExp(result.helperState, "machao") != tc.wantDefenderExp {
				t.Fatalf("expected attacker/main/helper exp 60/%d/%d, reports=%d/%d/%d helper=%+v", tc.wantDefenderExp, tc.wantDefenderExp, result.attackerReport.GeneralExpGained, result.defenderReport.GeneralExpGained, result.reinforcementReport.GeneralExpGained, result.helperState.Generals)
			}

			for _, report := range []BattleReport{result.attackerReport, result.defenderReport, result.reinforcementReport} {
				if len(report.PvpReinforcements) != 1 || len(report.PvpReinforcements[0].Generals) != 1 {
					t.Fatalf("expected one Ma Chao reinforcement snapshot, report=%s snapshots=%+v", report.ID, report.PvpReinforcements)
				}
				snapshot := report.PvpReinforcements[0].Generals[0]
				if !reinforcementSnapshotHasTrait(snapshot, "xiliang_tuji") || !reinforcementSnapshotHasTrait(snapshot, "tianshen_xiafan") || snapshot.EffectiveStats["force"]-snapshot.Stats["force"] != 20 || math.Abs(snapshot.Buffs[StatAttackBonus]-0.4) > 1e-9 {
					t.Fatalf("expected Ma Chao snapshot to retain force +20, attack buff 40%% and both traits, report=%s snapshot=%+v", report.ID, snapshot)
				}
				if _, exists := reinforcementOutcome(report, "tianshen_xiafan", "player_enemy_"+id); exists || standardReportHasTrait(report.Detail, "tianshen_xiafan") {
					t.Fatalf("expected passive Tianshen absent from all trigger timelines, report=%+v", report)
				}
				if !tc.wantTraitTriggered {
					if len(report.TraitTriggered) != 0 || len(report.TraitOutcomes) != 0 || report.Detail == nil || len(report.Detail.Traits) != 0 {
						t.Fatalf("expected missed Xiliang and passive Tianshen to leave empty timelines, report=%+v", report)
					}
					continue
				}
				if len(report.TraitTriggered) != 1 || report.TraitTriggered[0] != "xiliang_tuji" || len(report.TraitOutcomes) != 1 || report.Detail == nil || len(report.Detail.Traits) != 1 {
					t.Fatalf("expected exactly one real Xiliang outcome, report=%+v", report)
				}
				outcome, exists := reinforcementOutcome(report, "xiliang_tuji", "player_enemy_"+id)
				extra, extraOK := outcome.Detail["targetExtraLosses"].(map[string]int)
				if !exists || !extraOK || len(extra) != 1 || extra["weiCavalry"] != 6 || outcome.OwnerSide != "reinforcement" || outcome.OwnerGeneralID != "machao" {
					t.Fatalf("expected reinforcement Xiliang to add exactly 6 cavalry losses, report=%s outcome=%+v", report.ID, outcome)
				}
				trait := report.Detail.Traits[0]
				standardExtra, standardOK := trait.Detail["targetExtraLosses"].(map[string]int)
				if trait.TraitID != "xiliang_tuji" || trait.OwnerRole != "reinforcement" || trait.OwnerPlayerID != "player_enemy_"+id || !standardOK || len(standardExtra) != 1 || standardExtra["weiCavalry"] != 6 {
					t.Fatalf("expected one standard reinforcement Xiliang timeline with cavalry loss 6, report=%s trait=%+v", report.ID, trait)
				}
			}
		})
	}
}

// TestReinforcementEnemyTraitsRespectExplicitAttackerOnlyRole 验证仅主动进攻特性不会因援军映射到防守联盟而误触发。
func TestReinforcementEnemyTraitsRespectExplicitAttackerOnlyRole(t *testing.T) {
	active := reinforcementEnemyCombatTraits([]general.ActiveTrait{{
		TraitID: "huogong", OwnerSide: "reinforcement", Scope: "enemy_army", AllowedSides: []string{"attacker"},
	}})
	if len(active) != 0 {
		t.Fatalf("expected attacker-only trait excluded from reinforcement dispatch, got %+v", active)
	}
}

// TestFormalReinforcementEnemyTraitRoleMatrix 锁定正式配置中可由援军触发和必须按方向排除的全部对敌特性。
func TestFormalReinforcementEnemyTraitRoleMatrix(t *testing.T) {
	path := filepath.Join("..", "..", "..", "config", "generals.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read formal generals config failed: %v", err)
	}
	var cfg GeneralsConfig
	if err := json.Unmarshal(raw, &cfg); err != nil {
		t.Fatalf("decode formal generals config failed: %v", err)
	}
	allowed := map[string]bool{
		"yibing_touxi": true, "qimen_dunjia": true, "wolong_mouzhi": true,
		"xiliang_tuji": true, "xiaobawang_zhuiji": true,
		"huoshao_lianying": true, "lianying_zengshang": true, "kurouji": true, "kurou_fanji": true,
	}
	excluded := map[string]bool{
		"meihuo_raozhen": true, "huchi_chongzhen": true, "weizhen_zhenhe": true,
		"baibu_chuanyang": true, "shuiyan_qijun": true, "zhenhe_quanjun": true, "huogong": true,
	}
	seen := map[string]bool{}
	for _, hero := range cfg.Heroes {
		for _, trait := range []GeneralTraitConfig{hero.SpecialTrait, hero.BonusTrait} {
			if !trait.Enabled || (trait.Scope != "enemy_army" && trait.Scope != "enemy_traits") {
				continue
			}
			seen[trait.TraitID] = true
			mapped := reinforcementEnemyCombatTraits([]general.ActiveTrait{{
				TraitID: trait.TraitID, TraitType: trait.TraitType, OwnerSide: "reinforcement",
				Scope: trait.Scope, AllowedSides: trait.AllowedSides,
			}})
			switch {
			case allowed[trait.TraitID]:
				if len(mapped) != 1 || mapped[0].OwnerSide != "defender" || len(mapped[0].AllowedSides) != 1 || mapped[0].AllowedSides[0] != "defender" {
					t.Fatalf("expected formal trait %s mapped into defender coalition, got %+v", trait.TraitID, mapped)
				}
			case excluded[trait.TraitID]:
				if len(mapped) != 0 {
					t.Fatalf("expected role-limited formal trait %s excluded for reinforcement, got %+v", trait.TraitID, mapped)
				}
			default:
				t.Fatalf("formal enemy trait %s is missing from reinforcement role matrix", trait.TraitID)
			}
		}
	}
	if len(seen) != len(allowed)+len(excluded) {
		t.Fatalf("expected %d formal enemy traits, got %d: %+v", len(allowed)+len(excluded), len(seen), seen)
	}
	for traitID := range allowed {
		if !seen[traitID] {
			t.Fatalf("expected reinforcement-eligible formal trait %s in config", traitID)
		}
	}
	for traitID := range excluded {
		if !seen[traitID] {
			t.Fatalf("expected role-limited formal trait %s in config", traitID)
		}
	}
}
