// 本文件验证援军将领的对敌特性进入真实 PVP 战力、扣兵、压制顺序和三方战报。
package game

import (
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"testing"
	"time"

	"hero3/internal/core/combat"
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

// TestReinforcementEnemyPreBattleTraitChangesRealAttackPower 验证张辽援军震慑真实降低来袭战力且不把压制兵算作阵亡。
func TestReinforcementEnemyPreBattleTraitChangesRealAttackPower(t *testing.T) {
	cases := []struct {
		name                    string
		triggerChance           float64
		wantAttackerPower       float64
		wantAttackerLosses      int
		wantDefendingLosses     int
		wantMainDefenderLosses  int
		wantReinforcementLosses int
		wantSuppressed          int
	}{
		{name: "威震命中仅让两成来袭兵暂不参战", triggerChance: 1, wantAttackerPower: 800, wantAttackerLosses: 46, wantDefendingLosses: 42, wantMainDefenderLosses: 0, wantReinforcementLosses: 42, wantSuppressed: 20},
		{name: "威震未命中保持基础平局", triggerChance: 0, wantAttackerPower: 1000, wantAttackerLosses: 50, wantDefendingLosses: 50, wantMainDefenderLosses: 1, wantReinforcementLosses: 49},
	}
	for index, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			id := "zhangliao_reinforcement_" + string(rune('a'+index))
			result := runReinforcementEnemyPvp(t, reinforcementEnemyPvpConfig{
				id: id, attackerTroops: 100, defenderTroops: 1, marchMode: PvpMarchTypePlunder,
				helperFaction: "wei", helperGeneralID: "zhangliao", helperName: "张辽", helperTroops: 99,
				attackerFaction: "shu", attackerGeneral: "liubei", attackerName: "刘备",
				defenderFaction: "wu", defenderGeneral: "sunquan", defenderName: "孙权",
				helperSpecial: GeneralTraitConfig{
					TraitID: "weizhen_zhenhe", TraitType: general.TraitTypeSpecial, Enabled: true, Scope: "enemy_army",
					Params: map[string]float64{"triggerChance": tc.triggerChance, "effectRate": 0.2, "maxAffectedRate": 0.2},
				},
				helperBonus: GeneralTraitConfig{
					TraitID: "weizhen_xiaoyao", TraitType: general.TraitTypeBonus, Enabled: true, Scope: "self_army", TargetUnitType: "cavalry", AllowedSides: []string{"attacker"},
					Params: map[string]float64{"attackBonusRate": 0.35},
				},
			})
			wantWinner := "draw"
			if tc.wantSuppressed > 0 {
				wantWinner = "defender"
			}
			if result.battle.Result["winner"] != wantWinner || result.battle.Result["attackerPower"] != tc.wantAttackerPower || result.battle.Result["defensePower"] != float64(1000) {
				t.Fatalf("expected exact winner and power %s %.0f/1000, result=%+v", wantWinner, tc.wantAttackerPower, result.battle.Result)
			}
			if result.attackerLosses != tc.wantAttackerLosses || result.defendingLosses != tc.wantDefendingLosses || result.defenderReport.LostUnits["wuInfantry"] != tc.wantMainDefenderLosses || result.storedReinforcement.Losses["weiInfantry"] != tc.wantReinforcementLosses {
				t.Fatalf("expected attacker/main/reinforcement losses %d/%d/%d, battle=%+v main=%+v reinforcement=%+v", tc.wantAttackerLosses, tc.wantMainDefenderLosses, tc.wantReinforcementLosses, result.battle.Losses, result.defenderReport.LostUnits, result.storedReinforcement)
			}
			if result.attackerReport.DispatchedUnits["shuInfantry"] != 100 || result.attackerReport.SurvivedUnits["shuInfantry"] != 100-tc.wantAttackerLosses || result.storedMarch.AttackTroops["shuInfantry"] != 100-tc.wantAttackerLosses || armySliceToMap(result.defenderState.Army)["wuInfantry"] != 1-tc.wantMainDefenderLosses || result.storedReinforcement.RemainingTroops["weiInfantry"] != 99-tc.wantReinforcementLosses {
				t.Fatalf("expected suppressed troops retained and authoritative survivors %d/%d/%d, report=%+v march=%+v defender=%+v reinforcement=%+v", 100-tc.wantAttackerLosses, 1-tc.wantMainDefenderLosses, 99-tc.wantReinforcementLosses, result.attackerReport, result.storedMarch, result.defenderState.Army, result.storedReinforcement)
			}
			if result.attackerReport.GeneralExpGained != tc.wantDefendingLosses || result.defenderReport.GeneralExpGained != tc.wantAttackerLosses || result.reinforcementReport.GeneralExpGained != tc.wantAttackerLosses || pvpTestGeneralExp(result.helperState, "zhangliao") != tc.wantAttackerLosses {
				t.Fatalf("expected attacker/main/helper exp %d/%d/%d, reports=%d/%d/%d helper=%+v", tc.wantDefendingLosses, tc.wantAttackerLosses, tc.wantAttackerLosses, result.attackerReport.GeneralExpGained, result.defenderReport.GeneralExpGained, result.reinforcementReport.GeneralExpGained, result.helperState.Generals)
			}
			for _, report := range []BattleReport{result.attackerReport, result.defenderReport, result.reinforcementReport} {
				if len(report.PvpReinforcements) != 1 || len(report.PvpReinforcements[0].Generals) != 1 || !reinforcementSnapshotHasTrait(report.PvpReinforcements[0].Generals[0], "weizhen_zhenhe") || !reinforcementSnapshotHasTrait(report.PvpReinforcements[0].Generals[0], "weizhen_xiaoyao") {
					t.Fatalf("expected Zhang Liao snapshot to retain both traits, report=%s snapshots=%+v", report.ID, report.PvpReinforcements)
				}
				if _, exists := reinforcementOutcome(report, "weizhen_xiaoyao", "player_enemy_"+id); exists || standardReportHasTrait(report.Detail, "weizhen_xiaoyao") {
					t.Fatalf("expected attacker-only Xiaoyao excluded for reinforcement, report=%+v", report)
				}
				if tc.wantSuppressed == 0 {
					if len(report.TraitTriggered) != 0 || len(report.TraitOutcomes) != 0 || report.Detail == nil || len(report.Detail.Traits) != 0 {
						t.Fatalf("expected missed Weizhen and excluded Xiaoyao to leave empty timelines, report=%+v", report)
					}
					continue
				}
				outcome, triggered := reinforcementOutcome(report, "weizhen_zhenhe", "player_enemy_"+id)
				suppressed, detailOK := outcome.Detail["suppressedUnits"].(map[string]int)
				if !triggered || !detailOK || suppressed["shuInfantry"] != tc.wantSuppressed || len(report.TraitTriggered) != 1 || report.Detail == nil || len(report.Detail.Traits) != 1 {
					t.Fatalf("expected one reinforcement Weizhen suppression %d, report=%s outcome=%+v", tc.wantSuppressed, report.ID, outcome)
				}
				trait := report.Detail.Traits[0]
				standardSuppressed, standardOK := trait.Detail["suppressedUnits"].(map[string]int)
				if trait.TraitID != "weizhen_zhenhe" || trait.OwnerRole != "reinforcement" || trait.OwnerPlayerID != "player_enemy_"+id || !standardOK || standardSuppressed["shuInfantry"] != tc.wantSuppressed {
					t.Fatalf("expected standard reinforcement Weizhen timeline with actual amount %d, report=%s trait=%+v", tc.wantSuppressed, report.ID, trait)
				}
			}
		})
	}
}

// TestReinforcementEnemyPreBattleDamageAddsRealLosses 验证关羽援军的战前扣兵进入来袭方最终兵损和三方战报。
func TestReinforcementEnemyPreBattleDamageAddsRealLosses(t *testing.T) {
	cases := []struct {
		name                    string
		triggerChance           float64
		wantAttackerPower       float64
		wantAttackerLosses      int
		wantDefendingLosses     int
		wantMainDefenderLosses  int
		wantReinforcementLosses int
		wantPreDamage           int
	}{
		{name: "水淹命中先扣三成半兵力再进入核心", triggerChance: 1, wantAttackerPower: 650, wantAttackerLosses: 77, wantDefendingLosses: 35, wantMainDefenderLosses: 0, wantReinforcementLosses: 35, wantPreDamage: 35},
		{name: "水淹未命中保持基础平局", triggerChance: 0, wantAttackerPower: 1000, wantAttackerLosses: 50, wantDefendingLosses: 50, wantMainDefenderLosses: 1, wantReinforcementLosses: 49},
	}
	for index, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			id := "guanyu_reinforcement_" + string(rune('a'+index))
			result := runReinforcementEnemyPvp(t, reinforcementEnemyPvpConfig{
				id: id, attackerTroops: 100, defenderTroops: 1, marchMode: PvpMarchTypePlunder,
				helperFaction: "shu", helperGeneralID: "guanyu", helperName: "关羽", helperTroops: 99,
				attackerFaction: "wei", attackerGeneral: "caocao", attackerName: "曹操",
				defenderFaction: "wu", defenderGeneral: "sunquan", defenderName: "孙权",
				helperSpecial: GeneralTraitConfig{
					TraitID: "shuiyan_qijun", TraitType: general.TraitTypeSpecial, Enabled: true, Scope: "enemy_army",
					Params: map[string]float64{"triggerChance": tc.triggerChance, "effectRate": 0.35, "maxAffectedRate": 0.35},
				},
				helperBonus: GeneralTraitConfig{
					TraitID: "wusheng_pojun", TraitType: general.TraitTypeBonus, Enabled: true, Scope: "self_army", AllowedSides: []string{"attacker"},
					Params: map[string]float64{"attackBonusRate": 0.2},
				},
			})
			wantWinner := "draw"
			if tc.wantPreDamage > 0 {
				wantWinner = "defender"
			}
			if result.battle.Result["winner"] != wantWinner || result.battle.Result["attackerPower"] != tc.wantAttackerPower || result.battle.Result["defensePower"] != float64(1000) {
				t.Fatalf("expected exact winner and power %s %.0f/1000, result=%+v", wantWinner, tc.wantAttackerPower, result.battle.Result)
			}
			if result.attackerLosses != tc.wantAttackerLosses || result.defendingLosses != tc.wantDefendingLosses || result.defenderReport.LostUnits["wuInfantry"] != tc.wantMainDefenderLosses || result.storedReinforcement.Losses["shuInfantry"] != tc.wantReinforcementLosses {
				t.Fatalf("expected attacker/main/reinforcement losses %d/%d/%d, battle=%+v main=%+v reinforcement=%+v", tc.wantAttackerLosses, tc.wantMainDefenderLosses, tc.wantReinforcementLosses, result.battle.Losses, result.defenderReport.LostUnits, result.storedReinforcement)
			}
			if result.attackerReport.SurvivedUnits["weiInfantry"] != 100-tc.wantAttackerLosses || result.storedMarch.AttackTroops["weiInfantry"] != 100-tc.wantAttackerLosses || armySliceToMap(result.defenderState.Army)["wuInfantry"] != 1-tc.wantMainDefenderLosses || result.storedReinforcement.RemainingTroops["shuInfantry"] != 99-tc.wantReinforcementLosses {
				t.Fatalf("expected authoritative attacker/main/reinforcement survivors %d/%d/%d, report=%+v march=%+v defender=%+v reinforcement=%+v", 100-tc.wantAttackerLosses, 1-tc.wantMainDefenderLosses, 99-tc.wantReinforcementLosses, result.attackerReport, result.storedMarch, result.defenderState.Army, result.storedReinforcement)
			}
			if result.attackerReport.GeneralExpGained != tc.wantDefendingLosses || result.defenderReport.GeneralExpGained != tc.wantAttackerLosses || result.reinforcementReport.GeneralExpGained != tc.wantAttackerLosses || pvpTestGeneralExp(result.helperState, "guanyu") != tc.wantAttackerLosses {
				t.Fatalf("expected attacker/main/helper exp %d/%d/%d, reports=%d/%d/%d helper=%+v", tc.wantDefendingLosses, tc.wantAttackerLosses, tc.wantAttackerLosses, result.attackerReport.GeneralExpGained, result.defenderReport.GeneralExpGained, result.reinforcementReport.GeneralExpGained, result.helperState.Generals)
			}
			for _, report := range []BattleReport{result.attackerReport, result.defenderReport, result.reinforcementReport} {
				if len(report.PvpReinforcements) != 1 || len(report.PvpReinforcements[0].Generals) != 1 || !reinforcementSnapshotHasTrait(report.PvpReinforcements[0].Generals[0], "shuiyan_qijun") || !reinforcementSnapshotHasTrait(report.PvpReinforcements[0].Generals[0], "wusheng_pojun") {
					t.Fatalf("expected Guan Yu snapshot to retain both traits, report=%s snapshots=%+v", report.ID, report.PvpReinforcements)
				}
				if _, exists := reinforcementOutcome(report, "wusheng_pojun", "player_enemy_"+id); exists || standardReportHasTrait(report.Detail, "wusheng_pojun") {
					t.Fatalf("expected attacker-only Wusheng excluded for reinforcement, report=%+v", report)
				}
				if tc.wantPreDamage == 0 {
					if len(report.TraitTriggered) != 0 || len(report.TraitOutcomes) != 0 || report.Detail == nil || len(report.Detail.Traits) != 0 {
						t.Fatalf("expected missed Shuiyan and excluded Wusheng to leave empty timelines, report=%+v", report)
					}
					continue
				}
				outcome, triggered := reinforcementOutcome(report, "shuiyan_qijun", "player_enemy_"+id)
				preDamage, detailOK := outcome.Detail["preBattleAffected"].(map[string]int)
				if !triggered || !detailOK || preDamage["weiInfantry"] != tc.wantPreDamage || len(report.TraitTriggered) != 1 || report.Detail == nil || len(report.Detail.Traits) != 1 {
					t.Fatalf("expected one reinforcement Shuiyan pre-battle loss %d, report=%s outcome=%+v", tc.wantPreDamage, report.ID, outcome)
				}
				trait := report.Detail.Traits[0]
				standardAffected, standardOK := trait.Detail["preBattleAffected"].(map[string]int)
				if trait.TraitID != "shuiyan_qijun" || trait.OwnerRole != "reinforcement" || trait.OwnerPlayerID != "player_enemy_"+id || !standardOK || standardAffected["weiInfantry"] != tc.wantPreDamage {
					t.Fatalf("expected standard reinforcement Shuiyan timeline with actual loss %d, report=%s trait=%+v", tc.wantPreDamage, report.ID, trait)
				}
			}
		})
	}
}

// TestReinforcementEnemyAfterCombatTraitAddsRealLosses 验证黄忠援军在战斗结算后真实追加来袭方损失。
func TestReinforcementEnemyAfterCombatTraitAddsRealLosses(t *testing.T) {
	base := reinforcementEnemyPvpConfig{
		id: "huangzhong_control", helperFaction: "shu", helperGeneralID: "huangzhong", helperName: "黄忠",
		attackerFaction: "wei", attackerGeneral: "caocao", attackerName: "曹操",
		defenderFaction: "wu", defenderGeneral: "sunquan", defenderName: "孙权",
		helperBonus: GeneralTraitConfig{TraitID: "laodang_yizhuang", TraitType: general.TraitTypeBonus, Enabled: false, Scope: "enemy_army", Params: map[string]float64{"effectRate": 0.1, "triggerChance": 1}},
	}
	control := runReinforcementEnemyPvp(t, base)
	base.id = "huangzhong_active"
	base.helperBonus.Enabled = true
	active := runReinforcementEnemyPvp(t, base)
	if active.attackerLosses != control.attackerLosses+11 {
		t.Fatalf("expected reinforcement extra damage +11, control=%d active=%d", control.attackerLosses, active.attackerLosses)
	}
	for _, report := range []BattleReport{active.attackerReport, active.defenderReport, active.reinforcementReport} {
		outcome, ok := reinforcementOutcome(report, "laodang_yizhuang", "player_enemy_huangzhong_active")
		extra, extraOK := outcome.Detail["extraLosses"].(map[string]int)
		if !ok || !extraOK || extra[active.attackerUnit] != 11 {
			t.Fatalf("expected reinforcement-owned real extra losses 11, report=%s outcome=%+v", report.ID, outcome)
		}
	}
	if active.attackerReport.LostUnits[active.attackerUnit] != active.attackerLosses || active.attackerReport.SurvivedUnits[active.attackerUnit] != 110-active.attackerLosses {
		t.Fatalf("expected report losses and survivors to reconcile, losses=%+v survived=%+v battle=%d", active.attackerReport.LostUnits, active.attackerReport.SurvivedUnits, active.attackerLosses)
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

// TestPvpZhangfeiReinforcementSuppressionHitAndMissExcludeWanren 验证张飞援军临时压兵不计阵亡且不会越权触发主动进攻加攻。
func TestPvpZhangfeiReinforcementSuppressionHitAndMissExcludeWanren(t *testing.T) {
	cases := []struct {
		name                    string
		triggerChance           float64
		wantAttackerPower       float64
		wantAttackerLosses      int
		wantDefendingLosses     int
		wantMainDefenderLosses  int
		wantReinforcementLosses int
		wantSuppressed          int
	}{
		{name: "震慑命中只让半数来袭兵暂不参战", triggerChance: 1, wantAttackerPower: 500, wantAttackerLosses: 36, wantDefendingLosses: 27, wantMainDefenderLosses: 0, wantReinforcementLosses: 27, wantSuppressed: 50},
		{name: "震慑未命中保持基础平局", triggerChance: 0, wantAttackerPower: 1000, wantAttackerLosses: 50, wantDefendingLosses: 50, wantMainDefenderLosses: 1, wantReinforcementLosses: 49},
	}
	for index, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			id := "zhangfei_reinforcement_" + string(rune('a'+index))
			result := runReinforcementEnemyPvp(t, reinforcementEnemyPvpConfig{
				id: id, attackerTroops: 100, defenderTroops: 1, marchMode: PvpMarchTypePlunder,
				helperFaction: "shu", helperGeneralID: "zhangfei", helperName: "张飞", helperTroops: 99,
				attackerFaction: "wei", attackerGeneral: "caocao", attackerName: "曹操",
				defenderFaction: "wu", defenderGeneral: "sunquan", defenderName: "孙权",
				helperSpecial: GeneralTraitConfig{
					TraitID: "zhenhe_quanjun", TraitType: general.TraitTypeSpecial, Enabled: true, Scope: "enemy_army",
					Params: map[string]float64{"triggerChance": tc.triggerChance, "effectRate": 0.5, "maxAffectedRate": 0.5},
				},
				helperBonus: GeneralTraitConfig{
					TraitID: "wanren_nuhou", TraitType: general.TraitTypeBonus, Enabled: true, Scope: "self_army", TargetUnitType: "infantry", AllowedSides: []string{"attacker"},
					Params: map[string]float64{"attackBonusRate": 0.2},
				},
			})
			wantWinner := "draw"
			if tc.wantSuppressed > 0 {
				wantWinner = "defender"
			}
			if result.battle.Result["winner"] != wantWinner || result.battle.Result["attackerPower"] != tc.wantAttackerPower || result.battle.Result["defensePower"] != float64(1000) {
				t.Fatalf("expected exact winner and power %s %.0f/1000, result=%+v", wantWinner, tc.wantAttackerPower, result.battle.Result)
			}
			if result.attackerLosses != tc.wantAttackerLosses || result.defendingLosses != tc.wantDefendingLosses || result.defenderReport.LostUnits["wuInfantry"] != tc.wantMainDefenderLosses || result.storedReinforcement.Losses["shuInfantry"] != tc.wantReinforcementLosses {
				t.Fatalf("expected attacker/main/reinforcement losses %d/%d/%d, battle=%+v main=%+v reinforcement=%+v", tc.wantAttackerLosses, tc.wantMainDefenderLosses, tc.wantReinforcementLosses, result.battle.Losses, result.defenderReport.LostUnits, result.storedReinforcement)
			}
			if result.attackerReport.DispatchedUnits["weiInfantry"] != 100 || result.attackerReport.SurvivedUnits["weiInfantry"] != 100-tc.wantAttackerLosses || armySliceToMap(result.defenderState.Army)["wuInfantry"] != 1-tc.wantMainDefenderLosses || result.storedReinforcement.RemainingTroops["shuInfantry"] != 99-tc.wantReinforcementLosses {
				t.Fatalf("expected suppressed troops preserved in authoritative survivors %d/%d/%d, attacker=%+v defender=%+v reinforcement=%+v", 100-tc.wantAttackerLosses, 1-tc.wantMainDefenderLosses, 99-tc.wantReinforcementLosses, result.attackerReport, result.defenderState.Army, result.storedReinforcement)
			}
			if result.attackerReport.GeneralExpGained != tc.wantDefendingLosses || result.defenderReport.GeneralExpGained != tc.wantAttackerLosses || result.reinforcementReport.GeneralExpGained != tc.wantAttackerLosses || pvpTestGeneralExp(result.helperState, "zhangfei") != tc.wantAttackerLosses {
				t.Fatalf("expected attacker/main/helper exp %d/%d/%d, reports=%d/%d/%d helper=%+v", tc.wantDefendingLosses, tc.wantAttackerLosses, tc.wantAttackerLosses, result.attackerReport.GeneralExpGained, result.defenderReport.GeneralExpGained, result.reinforcementReport.GeneralExpGained, result.helperState.Generals)
			}

			for _, report := range []BattleReport{result.attackerReport, result.defenderReport, result.reinforcementReport} {
				if len(report.PvpReinforcements) != 1 || len(report.PvpReinforcements[0].Generals) != 1 || !reinforcementSnapshotHasTrait(report.PvpReinforcements[0].Generals[0], "zhenhe_quanjun") || !reinforcementSnapshotHasTrait(report.PvpReinforcements[0].Generals[0], "wanren_nuhou") {
					t.Fatalf("expected reinforcement snapshot to retain both owned traits, report=%s snapshots=%+v", report.ID, report.PvpReinforcements)
				}
				if _, exists := reinforcementOutcome(report, "wanren_nuhou", "player_enemy_"+id); exists || standardReportHasTrait(report.Detail, "wanren_nuhou") {
					t.Fatalf("expected attacker-only Wanren excluded for reinforcement, report=%+v", report)
				}
				if tc.wantSuppressed == 0 {
					if len(report.TraitTriggered) != 0 || len(report.TraitOutcomes) != 0 || len(report.Detail.Traits) != 0 {
						t.Fatalf("expected missed suppression and excluded Wanren to leave empty timelines, report=%+v", report)
					}
					continue
				}
				if len(report.TraitTriggered) != 1 || len(report.TraitOutcomes) != 1 || len(report.Detail.Traits) != 1 {
					t.Fatalf("expected exactly one real suppression outcome, report=%+v", report)
				}
				outcome, triggered := reinforcementOutcome(report, "zhenhe_quanjun", "player_enemy_"+id)
				suppressed, detailOK := outcome.Detail["suppressedUnits"].(map[string]int)
				if !triggered || !detailOK || suppressed["weiInfantry"] != tc.wantSuppressed || outcome.OwnerSide != "reinforcement" || outcome.OwnerGeneralID != "zhangfei" {
					t.Fatalf("expected reinforcement-owned temporary suppression %d, report=%s outcome=%+v", tc.wantSuppressed, report.ID, outcome)
				}
				trait := report.Detail.Traits[0]
				standardSuppressed, standardOK := trait.Detail["suppressedUnits"].(map[string]int)
				if trait.TraitID != "zhenhe_quanjun" || trait.OwnerRole != "reinforcement" || trait.OwnerPlayerID != "player_enemy_"+id || !standardOK || standardSuppressed["weiInfantry"] != tc.wantSuppressed {
					t.Fatalf("expected one standard reinforcement suppression timeline with actual amount %d, report=%s trait=%+v", tc.wantSuppressed, report.ID, trait)
				}
			}
		})
	}
}

// TestPvpHuangGaiReinforcementKurouHitAndMissKeepCrossSideOrder 验证黄盖援军苦肉计只压制敌方后续特性，并保留自身反击和三方结算顺序。
func TestPvpHuangGaiReinforcementKurouHitAndMissKeepCrossSideOrder(t *testing.T) {
	cases := []struct {
		name                    string
		triggerChance           float64
		wantDefendingLosses     int
		wantReinforcementLosses int
		wantAttackerExp         int
		wantTimeline            []string
	}{
		{name: "苦肉命中压制老当但保留自身反击", triggerChance: 1, wantDefendingLosses: 50, wantReinforcementLosses: 49, wantAttackerExp: 50, wantTimeline: []string{"kurouji", "kurou_fanji"}},
		{name: "苦肉未命中双方后续伤害依次生效", triggerChance: 0, wantDefendingLosses: 60, wantReinforcementLosses: 59, wantAttackerExp: 60, wantTimeline: []string{"laodang_yizhuang", "kurou_fanji"}},
	}
	for index, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			id := "huanggai_reinforcement_" + string(rune('a'+index))
			result := runReinforcementEnemyPvp(t, reinforcementEnemyPvpConfig{
				id: id, attackerTroops: 100, defenderTroops: 1, marchMode: PvpMarchTypePlunder,
				helperFaction: "wu", helperGeneralID: "huanggai", helperName: "黄盖", helperTroops: 99,
				attackerFaction: "shu", attackerGeneral: "huangzhong", attackerName: "黄忠",
				defenderFaction: "wu", defenderGeneral: "sunquan", defenderName: "孙权",
				attackerBonus: GeneralTraitConfig{
					TraitID: "laodang_yizhuang", TraitType: general.TraitTypeBonus, Enabled: true, Scope: "enemy_army",
					Params: map[string]float64{"effectRate": 0.1, "triggerChance": 1},
				},
				helperSpecial: GeneralTraitConfig{
					TraitID: "kurouji", TraitType: general.TraitTypeSpecial, Enabled: true, Scope: "enemy_traits",
					Params: map[string]float64{"disableTraitCount": 1, "triggerChance": tc.triggerChance},
				},
				helperBonus: GeneralTraitConfig{
					TraitID: "kurou_fanji", TraitType: general.TraitTypeBonus, Enabled: true, Scope: "enemy_army",
					Params: map[string]float64{"effectRate": 0.1, "triggerChance": 1},
				},
			})
			if result.battle.Result["winner"] != "draw" || result.battle.Result["attackerPower"] != float64(1000) || result.battle.Result["defensePower"] != float64(1000) {
				t.Fatalf("expected exact draw power 1000/1000, result=%+v", result.battle.Result)
			}
			if result.attackerLosses != 60 || result.defendingLosses != tc.wantDefendingLosses || result.defenderReport.LostUnits["wuInfantry"] != 1 || result.storedReinforcement.Losses["wuInfantry"] != tc.wantReinforcementLosses {
				t.Fatalf("expected attacker/main/reinforcement losses 60/1/%d, battle=%+v main=%+v reinforcement=%+v", tc.wantReinforcementLosses, result.battle.Losses, result.defenderReport.LostUnits, result.storedReinforcement)
			}
			if result.attackerReport.SurvivedUnits["shuInfantry"] != 40 || armySliceToMap(result.defenderState.Army)["wuInfantry"] != 0 || result.storedReinforcement.RemainingTroops["wuInfantry"] != 99-tc.wantReinforcementLosses {
				t.Fatalf("expected authoritative attacker/main/reinforcement survivors 40/0/%d, attacker=%+v defender=%+v reinforcement=%+v", 99-tc.wantReinforcementLosses, result.attackerReport.SurvivedUnits, result.defenderState.Army, result.storedReinforcement)
			}
			if result.attackerReport.GeneralExpGained != tc.wantAttackerExp || result.defenderReport.GeneralExpGained != 60 || result.reinforcementReport.GeneralExpGained != 60 || pvpTestGeneralExp(result.helperState, "huanggai") != 60 {
				t.Fatalf("expected attacker/main/helper exp %d/60/60, reports=%d/%d/%d helper=%+v", tc.wantAttackerExp, result.attackerReport.GeneralExpGained, result.defenderReport.GeneralExpGained, result.reinforcementReport.GeneralExpGained, result.helperState.Generals)
			}

			for reportIndex, report := range []BattleReport{result.attackerReport, result.defenderReport, result.reinforcementReport} {
				if len(report.PvpReinforcements) != 1 || len(report.PvpReinforcements[0].Generals) != 1 || !reinforcementSnapshotHasTrait(report.PvpReinforcements[0].Generals[0], "kurouji") || !reinforcementSnapshotHasTrait(report.PvpReinforcements[0].Generals[0], "kurou_fanji") {
					t.Fatalf("expected reinforcement snapshot to retain both owned traits, report=%s reinforcement=%+v", report.ID, report.PvpReinforcements)
				}
				if reportIndex < 2 && (len(report.PvpAttackerGenerals) != 1 || !pvpSnapshotHasTrait(report.PvpAttackerGenerals[0], "laodang_yizhuang")) {
					t.Fatalf("expected both main reports to retain attacker Laodang snapshot, report=%s attacker=%+v", report.ID, report.PvpAttackerGenerals)
				}
				wantTimeline := tc.wantTimeline
				if reportIndex == 2 && tc.triggerChance == 0 {
					wantTimeline = []string{"kurou_fanji"}
				}
				if len(report.TraitTriggered) != len(wantTimeline) || len(report.TraitOutcomes) != len(wantTimeline) || report.Detail == nil || len(report.Detail.Traits) != len(wantTimeline) {
					t.Fatalf("expected exactly %d ordered real outcomes, report=%+v", len(wantTimeline), report)
				}
				for traitIndex, traitID := range wantTimeline {
					if report.TraitTriggered[traitIndex] != traitID || report.Detail.Traits[traitIndex].TraitID != traitID {
						t.Fatalf("expected timeline %v, report=%s legacy=%+v standard=%+v", wantTimeline, report.ID, report.TraitTriggered, report.Detail.Traits)
					}
				}

				counter, counterOK := reinforcementOutcome(report, "kurou_fanji", "player_enemy_"+id)
				counterLosses, counterDetailOK := counter.Detail["extraLosses"].(map[string]int)
				if !counterOK || !counterDetailOK || counterLosses["shuInfantry"] != 10 || counter.OwnerSide != "reinforcement" || counter.OwnerGeneralID != "huanggai" {
					t.Fatalf("expected reinforcement-owned Kurou counter loss 10, report=%s outcome=%+v", report.ID, counter)
				}
				counterTrait := report.Detail.Traits[len(report.Detail.Traits)-1]
				counterStandardLosses, counterStandardOK := counterTrait.Detail["extraLosses"].(map[string]int)
				if counterTrait.TraitID != "kurou_fanji" || counterTrait.OwnerRole != "reinforcement" || counterTrait.OwnerPlayerID != "player_enemy_"+id || !counterStandardOK || counterStandardLosses["shuInfantry"] != 10 {
					t.Fatalf("expected final standard reinforcement counter loss 10, report=%s trait=%+v", report.ID, counterTrait)
				}

				if tc.triggerChance == 1 {
					suppression, suppressionOK := reinforcementOutcome(report, "kurouji", "player_enemy_"+id)
					disabled, disabledOK := suppression.Detail["disabledTraitCount"].(int)
					if !suppressionOK || !disabledOK || disabled != 1 || suppression.OwnerSide != "reinforcement" || suppression.OwnerGeneralID != "huanggai" || standardReportHasTrait(report.Detail, "laodang_yizhuang") {
						t.Fatalf("expected reinforcement Kurou to suppress only attacker Laodang, report=%+v suppression=%+v", report, suppression)
					}
					continue
				}
				if _, exists := reinforcementOutcome(report, "kurouji", "player_enemy_"+id); exists || standardReportHasTrait(report.Detail, "kurouji") {
					t.Fatalf("expected missed Kurou absent from real timelines, report=%+v", report)
				}
				if reportIndex == 2 {
					if _, exists := report.TraitOutcomes["laodang_yizhuang"]; exists || standardReportHasTrait(report.Detail, "laodang_yizhuang") {
						t.Fatalf("expected independent reinforcement report to omit enemy Laodang, report=%+v", report)
					}
					continue
				}
				laodang, exists := report.TraitOutcomes["laodang_yizhuang"]
				laodangLosses, lossesOK := laodang.Detail["extraLosses"].(map[string]int)
				if !exists || !lossesOK || laodangLosses["wuInfantry"] != 10 || laodang.OwnerSide != "attacker" || laodang.OwnerGeneralID != "huangzhong" {
					t.Fatalf("expected attacker Laodang extra coalition loss 10 after missed Kurou, report=%s outcome=%+v", report.ID, laodang)
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

// TestAttackerAfterCombatDamageAggregatesMainAndReinforcementLosses 验证同兵种主守军和援军的追加损失完整汇总到三方战报与权威状态。
func TestAttackerAfterCombatDamageAggregatesMainAndReinforcementLosses(t *testing.T) {
	base := reinforcementEnemyPvpConfig{
		id: "coalition_damage_control", attackerTroops: 1000, defenderTroops: 500, marchMode: PvpMarchTypePlunder,
		helperFaction: "wu", helperGeneralID: "taishici", helperName: "太史慈", helperTroops: 500,
		attackerFaction: "shu", attackerGeneral: "huangzhong", attackerName: "黄忠",
		defenderFaction: "wu", defenderGeneral: "sunquan", defenderName: "孙权",
		attackerBonus: GeneralTraitConfig{
			TraitID: "laodang_yizhuang", TraitType: general.TraitTypeBonus, Enabled: false, Scope: "enemy_army",
			Params: map[string]float64{"effectRate": 0.1, "triggerChance": 1},
		},
	}
	control := runReinforcementEnemyPvp(t, base)
	base.id = "coalition_damage_active"
	base.attackerBonus.Enabled = true
	active := runReinforcementEnemyPvp(t, base)
	if control.battle.Result["attackerPower"] != float64(10000) || control.battle.Result["defensePower"] != float64(10000) ||
		active.battle.Result["attackerPower"] != float64(10000) || active.battle.Result["defensePower"] != float64(10000) {
		t.Fatalf("expected equal 10000/10000 coalition power, control=%+v active=%+v", control.battle.Result, active.battle.Result)
	}
	if control.attackerLosses != 500 || active.attackerLosses != 500 || control.defendingLosses != 500 || active.defendingLosses != 600 {
		t.Fatalf("expected core 500/500 and active defender extra 100, control=%d/%d active=%d/%d", control.attackerLosses, control.defendingLosses, active.attackerLosses, active.defendingLosses)
	}
	if _, triggered := control.attackerReport.TraitOutcomes["laodang_yizhuang"]; triggered {
		t.Fatalf("expected disabled control without outcome, outcomes=%+v", control.attackerReport.TraitOutcomes)
	}
	for _, report := range []BattleReport{active.attackerReport, active.defenderReport} {
		outcome, ok := report.TraitOutcomes["laodang_yizhuang"]
		extra, extraOK := outcome.Detail["extraLosses"].(map[string]int)
		if !ok || !extraOK || extra["wuInfantry"] != 100 || outcome.OwnerSide != "attacker" || outcome.OwnerGeneralID != "huangzhong" || outcome.Detail["effectRate"] != 0.1 {
			t.Fatalf("expected aggregated attacker-owned extra losses 100, report=%s outcome=%+v", report.ID, outcome)
		}
		if len(report.TraitTriggered) != 1 || report.TraitTriggered[0] != "laodang_yizhuang" || !standardReportHasTrait(report.Detail, "laodang_yizhuang") {
			t.Fatalf("expected one real after-combat timeline entry, report=%s legacy=%+v detail=%+v", report.ID, report.TraitTriggered, report.Detail)
		}
	}
	if len(active.reinforcementReport.TraitTriggered) != 0 || len(active.reinforcementReport.TraitOutcomes) != 0 || standardReportHasTrait(active.reinforcementReport.Detail, "laodang_yizhuang") {
		t.Fatalf("expected independent reinforcement report to omit main-general traits, report=%+v", active.reinforcementReport)
	}
	for _, report := range []BattleReport{active.attackerReport, active.defenderReport, active.reinforcementReport} {
		if report.PvpReinforcementLosses[active.reinforcementID]["wuInfantry"] != 300 {
			t.Fatalf("expected every report to use final reinforcement loss 300, report=%s reinforcement=%+v", report.ID, report.PvpReinforcementLosses)
		}
	}
	if active.attackerReport.DefenderLostUnits["wuInfantry"] != 300 || active.defenderReport.LostUnits["wuInfantry"] != 300 || active.reinforcementReport.LostUnits["wuInfantry"] != 300 {
		t.Fatalf("expected attacker and both defender perspectives to use final main/reinforcement loss 300, attacker=%+v defender=%+v reinforcement=%+v", active.attackerReport.DefenderLostUnits, active.defenderReport.LostUnits, active.reinforcementReport.LostUnits)
	}
	if active.defenderReport.DefenderLostUnits["shuInfantry"] != 500 || len(active.reinforcementReport.DefenderLostUnits) != 0 {
		t.Fatalf("expected main defender report to use attacker loss 500 and independent reinforcement report to omit enemy detail, defender=%+v reinforcement=%+v", active.defenderReport.DefenderLostUnits, active.reinforcementReport.DefenderLostUnits)
	}
	if active.reinforcementReport.LostUnits["wuInfantry"] != 300 || active.reinforcementReport.SurvivedUnits["wuInfantry"] != 200 {
		t.Fatalf("expected independent reinforcement report 500/300/200, report=%+v", active.reinforcementReport)
	}
	if armySliceToMap(active.defenderState.Army)["wuInfantry"] != 200 || active.storedReinforcement.RemainingTroops["wuInfantry"] != 200 || active.storedReinforcement.Losses["wuInfantry"] != 300 {
		t.Fatalf("expected authoritative main and reinforcement remaining 200, defender=%+v reinforcement=%+v", active.defenderState.Army, active.storedReinforcement)
	}
	if pvpTestGeneralExp(active.helperState, "taishici") != 500 || active.reinforcementReport.GeneralExpGained != 500 || active.attackerReport.GeneralExpGained != 600 {
		t.Fatalf("expected helper/attacker experience 500/600, helper=%+v helperReport=%d attackerReport=%d", active.helperState.Generals, active.reinforcementReport.GeneralExpGained, active.attackerReport.GeneralExpGained)
	}
}

// TestReinforcementTraitSuppressionSharesDefenderCoalitionBudget 验证诸葛亮援军优先压制进攻主将的一项后续特性。
func TestReinforcementTraitSuppressionSharesDefenderCoalitionBudget(t *testing.T) {
	base := reinforcementEnemyPvpConfig{
		id: "wolong_control", helperFaction: "shu", helperGeneralID: "zhugeliang", helperName: "诸葛亮", helperTroops: 200,
		attackerFaction: "shu", attackerGeneral: "huangzhong", attackerName: "黄忠",
		defenderFaction: "wu", defenderGeneral: "sunquan", defenderName: "孙权",
		attackerBonus: GeneralTraitConfig{TraitID: "laodang_yizhuang", TraitType: general.TraitTypeBonus, Enabled: true, Scope: "enemy_army", Params: map[string]float64{"effectRate": 0.1, "triggerChance": 1}},
		helperBonus:   GeneralTraitConfig{TraitID: "wolong_mouzhi", TraitType: general.TraitTypeBonus, Enabled: false, Scope: "enemy_traits", Params: map[string]float64{"disableTraitCount": 1, "triggerChance": 1}},
	}
	withoutDamage := base
	withoutDamage.id = "wolong_baseline"
	withoutDamage.attackerBonus.Enabled = false
	baseline := runReinforcementEnemyPvp(t, withoutDamage)
	control := runReinforcementEnemyPvp(t, base)
	base.id = "wolong_active"
	base.helperBonus.Enabled = true
	active := runReinforcementEnemyPvp(t, base)
	if control.defendingLosses != baseline.defendingLosses+20 || active.defendingLosses != baseline.defendingLosses {
		t.Fatalf("expected one attacker extra-damage trait worth 20 defending losses to be suppressed, baseline=%d control=%d active=%d controlOutcomes=%+v activeOutcomes=%+v controlGenerals=%+v", baseline.defendingLosses, control.defendingLosses, active.defendingLosses, control.attackerReport.TraitOutcomes, active.attackerReport.TraitOutcomes, control.attackerReport.PvpAttackerGenerals)
	}
	for _, report := range []BattleReport{active.attackerReport, active.defenderReport, active.reinforcementReport} {
		outcome, ok := reinforcementOutcome(report, "wolong_mouzhi", "player_enemy_wolong_active")
		actual, actualOK := outcome.Detail["disabledTraitCount"].(int)
		if !ok || !actualOK || actual != 1 {
			t.Fatalf("expected reinforcement-owned actual suppression 1, report=%s outcome=%+v", report.ID, outcome)
		}
		if _, exists := report.TraitOutcomes["laodang_yizhuang"]; exists || standardReportHasTrait(report.Detail, "laodang_yizhuang") {
			t.Fatalf("expected suppressed attacker trait absent, report=%s outcomes=%+v", report.ID, report.TraitOutcomes)
		}
	}
}

// TestPvpZhugeLiangReinforcementQimenAndWolongKeepCrossPhaseOrder 验证诸葛亮援军临时困兵与后续特性压制独立生效并保持跨阶段顺序。
func TestPvpZhugeLiangReinforcementQimenAndWolongKeepCrossPhaseOrder(t *testing.T) {
	run := func(id string, wolongEnabled bool) reinforcementEnemyPvpResult {
		return runReinforcementEnemyPvp(t, reinforcementEnemyPvpConfig{
			id: id, attackerTroops: 100, defenderTroops: 1, marchMode: PvpMarchTypePlunder,
			helperFaction: "shu", helperGeneralID: "zhugeliang", helperName: "诸葛亮", helperTroops: 99,
			attackerFaction: "shu", attackerGeneral: "huangzhong", attackerName: "黄忠",
			defenderFaction: "wu", defenderGeneral: "sunquan", defenderName: "孙权",
			attackerBonus: GeneralTraitConfig{
				TraitID: "laodang_yizhuang", TraitType: general.TraitTypeBonus, Enabled: true, Scope: "enemy_army",
				Params: map[string]float64{"effectRate": 0.1, "triggerChance": 1},
			},
			helperSpecial: GeneralTraitConfig{
				TraitID: "qimen_dunjia", TraitType: general.TraitTypeSpecial, Enabled: true, Scope: "enemy_army",
				Params: map[string]float64{"effectRate": 0.25, "maxAffectedRate": 0.25, "triggerChance": 1},
			},
			helperBonus: GeneralTraitConfig{
				TraitID: "wolong_mouzhi", TraitType: general.TraitTypeBonus, Enabled: wolongEnabled, Scope: "enemy_traits",
				Params: map[string]float64{"disableTraitCount": 1, "triggerChance": 1},
			},
		})
	}

	control := run("zhugeliang_reinforcement_control", false)
	active := run("zhugeliang_reinforcement_active", true)
	for _, current := range []reinforcementEnemyPvpResult{control, active} {
		if current.battle.Result["winner"] != "defender" || current.battle.Result["attackerPower"] != float64(750) || current.battle.Result["defensePower"] != float64(1000) {
			t.Fatalf("expected Qimen to form formal 750/1000 defender victory, result=%+v", current.battle.Result)
		}
		if current.attackerLosses != 45 || current.defenderReport.LostUnits["wuInfantry"] != 0 || current.storedReinforcement.Losses["shuInfantry"] != current.defendingLosses {
			t.Fatalf("expected attacker/main losses 45/0 and all defending losses on reinforcement, battle=%+v main=%+v reinforcement=%+v", current.battle.Losses, current.defenderReport.LostUnits, current.storedReinforcement)
		}
		if current.attackerReport.SurvivedUnits["shuInfantry"] != 55 || current.storedMarch.AttackTroops["shuInfantry"] != 55 || armySliceToMap(current.defenderState.Army)["wuInfantry"] != 1 || current.storedReinforcement.RemainingTroops["shuInfantry"] != 99-current.defendingLosses {
			t.Fatalf("expected suppressed troops retained and authoritative attacker/main/reinforcement survivors, report=%+v march=%+v defender=%+v reinforcement=%+v", current.attackerReport.SurvivedUnits, current.storedMarch, current.defenderState.Army, current.storedReinforcement)
		}
		for _, report := range []BattleReport{current.attackerReport, current.defenderReport, current.reinforcementReport} {
			qimen, exists := reinforcementOutcome(report, "qimen_dunjia", current.helperState.Player.ID)
			suppressed, suppressedOK := qimen.Detail["suppressedUnits"].(map[string]int)
			if !exists || !suppressedOK || suppressed["shuInfantry"] != 25 {
				t.Fatalf("expected Qimen to suppress 25 temporary attacker troops in every report, report=%s outcome=%+v", report.ID, qimen)
			}
		}
	}

	if control.defendingLosses != 48 || control.attackerReport.GeneralExpGained != 48 || control.defenderReport.GeneralExpGained != 45 || control.reinforcementReport.GeneralExpGained != 45 {
		t.Fatalf("expected control Laodang to add 9 Shu infantry losses for total 48 and keep defender exp 45, losses=%d exp=%d/%d/%d", control.defendingLosses, control.attackerReport.GeneralExpGained, control.defenderReport.GeneralExpGained, control.reinforcementReport.GeneralExpGained)
	}
	if active.defendingLosses != 39 || active.attackerReport.GeneralExpGained != 39 || active.defenderReport.GeneralExpGained != 45 || active.reinforcementReport.GeneralExpGained != 45 || pvpTestGeneralExp(active.helperState, "zhugeliang") != 45 {
		t.Fatalf("expected Wolong to remove exactly 9 extra Shu infantry losses while preserving defender exp 45, losses=%d exp=%d/%d/%d helper=%+v", active.defendingLosses, active.attackerReport.GeneralExpGained, active.defenderReport.GeneralExpGained, active.reinforcementReport.GeneralExpGained, active.helperState.Generals)
	}
	for reportIndex, report := range []BattleReport{control.attackerReport, control.defenderReport, control.reinforcementReport} {
		if reportIndex < 2 {
			if len(report.TraitTriggered) != 2 || report.TraitTriggered[0] != "qimen_dunjia" || report.TraitTriggered[1] != "laodang_yizhuang" || report.Detail == nil || len(report.Detail.Traits) != 2 || report.Detail.Traits[0].TraitID != "qimen_dunjia" || report.Detail.Traits[1].TraitID != "laodang_yizhuang" {
				t.Fatalf("expected control main timeline Qimen then Laodang, report=%+v", report)
			}
			continue
		}
		if len(report.TraitTriggered) != 1 || report.TraitTriggered[0] != "qimen_dunjia" || report.Detail == nil || len(report.Detail.Traits) != 1 || report.Detail.Traits[0].TraitID != "qimen_dunjia" {
			t.Fatalf("expected independent reinforcement report to keep only own Qimen, report=%+v", report)
		}
	}

	for _, report := range []BattleReport{active.attackerReport, active.defenderReport, active.reinforcementReport} {
		qimen, qimenOK := reinforcementOutcome(report, "qimen_dunjia", "player_enemy_zhugeliang_reinforcement_active")
		suppressed, suppressedOK := qimen.Detail["suppressedUnits"].(map[string]int)
		wolong, wolongOK := reinforcementOutcome(report, "wolong_mouzhi", "player_enemy_zhugeliang_reinforcement_active")
		disabled, disabledOK := wolong.Detail["disabledTraitCount"].(int)
		if !qimenOK || !suppressedOK || suppressed["shuInfantry"] != 25 || !wolongOK || !disabledOK || disabled != 1 {
			t.Fatalf("expected reinforcement Qimen 25 and Wolong actual suppression 1, report=%s qimen=%+v wolong=%+v", report.ID, qimen, wolong)
		}
		if len(report.TraitTriggered) != 2 || report.TraitTriggered[0] != "qimen_dunjia" || report.TraitTriggered[1] != "wolong_mouzhi" || report.Detail == nil || len(report.Detail.Traits) != 2 || report.Detail.Traits[0].TraitID != "qimen_dunjia" || report.Detail.Traits[1].TraitID != "wolong_mouzhi" {
			t.Fatalf("expected real cross-phase timeline Qimen then Wolong, report=%+v", report)
		}
		if _, exists := report.TraitOutcomes["laodang_yizhuang"]; exists || standardReportHasTrait(report.Detail, "laodang_yizhuang") {
			t.Fatalf("expected Wolong-suppressed Laodang absent from timelines, report=%+v", report)
		}
		if len(report.PvpReinforcements) != 1 || len(report.PvpReinforcements[0].Generals) != 1 || !reinforcementSnapshotHasTrait(report.PvpReinforcements[0].Generals[0], "qimen_dunjia") || !reinforcementSnapshotHasTrait(report.PvpReinforcements[0].Generals[0], "wolong_mouzhi") {
			t.Fatalf("expected Zhuge Liang reinforcement snapshot to retain both formal traits, report=%s snapshots=%+v", report.ID, report.PvpReinforcements)
		}
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
		"yibing_touxi": true, "weizhen_zhenhe": true, "shuiyan_qijun": true,
		"zhenhe_quanjun": true, "qimen_dunjia": true, "wolong_mouzhi": true,
		"xiliang_tuji": true, "laodang_yizhuang": true, "xiaobawang_zhuiji": true,
		"huoshao_lianying": true, "lianying_zengshang": true, "kurouji": true, "kurou_fanji": true,
	}
	excluded := map[string]bool{
		"meihuo_raozhen": true, "huchi_chongzhen": true,
		"pojun_pofang": true, "baibu_chuanyang": true, "qibing_raohou": true, "huogong": true,
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

// TestAttackerSuppressionConsumesReinforcementEnemyTrait 验证进攻方一次压制会拦截防守联盟中的援军对敌特性。
func TestAttackerSuppressionConsumesReinforcementEnemyTrait(t *testing.T) {
	record := Reinforcement{
		ID: "rein_huangzhong", FromPlayerID: "player_huangzhong", Status: ReinforcementStatusStationed,
		Rules: GarrisonRules{CanFight: true}, Generals: []ReinforcementGeneralSnapshot{{ID: "huangzhong", Traits: []GeneralTraitInstance{{
			TraitID: "laodang_yizhuang", TraitType: general.TraitTypeBonus, Scope: "enemy_army",
			Params: map[string]float64{"effectRate": 0.1, "triggerChance": 1},
		}}}},
	}
	result := combat.CombatResult{
		Winner:         "defender",
		AttackerLosses: []combat.UnitLoss{{ID: "weiInfantry", Count: 100, Losses: 50}},
		DefenderLosses: []combat.UnitLoss{{ID: "shuInfantry", Count: 100, Losses: 40}},
	}
	ctx := &general.AfterCombatResolveContext{
		Result: &result, Attacker: &combat.Army{}, Defender: &combat.Army{},
		AttackerOwnsTrait: true, DefenderOwnsTrait: true, Scene: "attack",
	}
	active := []general.ActiveTrait{{
		TraitID: "wolong_mouzhi", TraitType: general.TraitTypeBonus, OwnerSide: "attacker", OwnerGeneralID: "zhugeliang",
		Scope: "enemy_traits", Params: general.Params{"disableTraitCount": 1, "triggerChance": 1},
	}}
	active = append(active, activeReinforcementEnemyTraits([]Reinforcement{record})...)
	general.Dispatch(ctx, active)
	mainOutcomes, reinforcementOutcomes := splitReinforcementTraitOutcomes(ctx.Triggered, []Reinforcement{record})
	if result.AttackerLosses[0].Losses != 50 || len(reinforcementOutcomes[record.ID]) != 0 {
		t.Fatalf("expected reinforcement extra damage suppressed without changing attacker losses, result=%+v outcomes=%+v", result, reinforcementOutcomes)
	}
	outcome := mainOutcomes["wolong_mouzhi"]
	if actual, ok := outcome.Detail["disabledTraitCount"].(int); !ok || actual != 1 {
		t.Fatalf("expected attacker suppression to record one actual interception, outcome=%+v", outcome)
	}
}
