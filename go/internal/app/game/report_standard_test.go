// 本文件测试标准战报结构和兼容查询行为。
package game

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"
)

func TestNormalizeBattleReportBuildsAttackDetail(t *testing.T) {
	report := NormalizeBattleReport(BattleReport{
		ID:                "br_attack",
		PlayerID:          "player_a",
		PlayerName:        "许昌",
		PlayerFaction:     "wei",
		TargetID:          "npc_1",
		TargetName:        "黄巾营地（NPC）",
		Type:              "attack",
		Result:            "attacker_victory",
		PlayerPower:       100,
		EnemyPower:        80,
		DispatchedUnits:   map[string]int{"weiInfantry": 10},
		LostUnits:         map[string]int{"weiInfantry": 2},
		DefenderFaction:   "shu",
		DefenderUnits:     map[string]int{"shuInfantry": 8},
		DefenderLostUnits: map[string]int{"shuInfantry": 8},
		DefenderRevealed:  true,
		Rewards:           map[string]int{"wood": 100},
		CreatedAt:         time.Now().UTC().Format(time.RFC3339),
	})

	if report.ViewType != ReportViewAttack || report.SourceType != ReportSourceNPCCity {
		t.Fatalf("unexpected normalized types: view=%s source=%s", report.ViewType, report.SourceType)
	}
	if report.Detail == nil {
		t.Fatal("expected standard detail")
	}
	if report.Detail.PrimarySide.Role != "attacker" || report.Detail.SecondarySide == nil || report.Detail.SecondarySide.Role != "npc" {
		t.Fatalf("unexpected detail sides: %+v secondary=%+v", report.Detail.PrimarySide, report.Detail.SecondarySide)
	}
	if got := report.Detail.PrimarySide.Units[0].Survived; got != 8 {
		t.Fatalf("expected survived troops 8, got %d", got)
	}
}

// TestNormalizeBattleReportKeepsTraitOutcomeDetails 验证标准战报保留特性触发归属和详细数据。
func TestNormalizeBattleReportKeepsTraitOutcomeDetails(t *testing.T) {
	report := NormalizeBattleReport(BattleReport{
		ID:               "br_trait_detail",
		PlayerID:         "player_trait",
		PlayerFaction:    "shu",
		TargetID:         "npc_trait",
		TargetName:       "樊城（NPC）",
		Type:             "attack",
		Result:           "attacker_victory",
		DefenderRevealed: true,
		TraitTriggered:   []string{"laodang_yizhuang"},
		TraitOutcomes: map[string]TraitOutcomeReport{
			"laodang_yizhuang": {
				TraitID:        "laodang_yizhuang",
				Name:           "老当益壮",
				OwnerSide:      "attacker",
				OwnerGeneralID: "huangzhong",
				Detail: map[string]interface{}{
					"effectRate":    0.1,
					"extraLosses":   map[string]int{"greedyWolf": 193},
					"triggerChance": 1.0,
				},
			},
		},
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
	})
	if len(report.Detail.Traits) != 1 {
		t.Fatalf("expected one trait outcome, got %+v", report.Detail.Traits)
	}
	trait := report.Detail.Traits[0]
	if trait.OwnerSide != "primary" || trait.OwnerRole != "attacker" || trait.GeneralID != "huangzhong" || trait.Detail["effectRate"] != 0.1 {
		t.Fatalf("expected trait owner and detail snapshot, got %+v", trait)
	}
}

// TestNormalizeBattleReportKeepsReinforcementTraitPlayerOwners 验证同将领援军结果仍保留不同玩家归属。
func TestNormalizeBattleReportKeepsReinforcementTraitPlayerOwners(t *testing.T) {
	report := NormalizeBattleReport(BattleReport{
		ID: "br_reinforcement_trait_owners", PlayerID: "attacker", PlayerFaction: "wei", Type: "attack", Result: "defender_victory",
		TraitTriggered: []string{"longdan_jiuyuan", "longdan_jiuyuan::reinforcement::zhaoyun"},
		TraitOutcomes: map[string]TraitOutcomeReport{
			"longdan_jiuyuan": {
				TraitID: "longdan_jiuyuan", Name: "龙胆救援", OwnerSide: "reinforcement", OwnerPlayerID: "helper_a", OwnerGeneralID: "zhaoyun",
				Detail: map[string]interface{}{"reducedLosses": map[string]int{"greedyWolf": 10}},
			},
			"longdan_jiuyuan::reinforcement::zhaoyun": {
				TraitID: "longdan_jiuyuan", Name: "龙胆救援", OwnerSide: "reinforcement", OwnerPlayerID: "helper_b", OwnerGeneralID: "zhaoyun",
				Detail: map[string]interface{}{"reducedLosses": map[string]int{"greedyWolf": 20}},
			},
		},
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
	})
	if report.Detail == nil || len(report.Detail.Traits) != 2 {
		t.Fatalf("expected two reinforcement trait entries, detail=%+v", report.Detail)
	}
	for index, playerID := range []string{"helper_a", "helper_b"} {
		trait := report.Detail.Traits[index]
		if trait.TraitID != "longdan_jiuyuan" || trait.OwnerRole != "reinforcement" || trait.OwnerPlayerID != playerID || trait.GeneralID != "zhaoyun" {
			t.Fatalf("expected reinforcement trait owner %s at index %d, trait=%+v", playerID, index, trait)
		}
	}
}

// TestBattleReportJSONKeepsLegacyAndStandardTraitOwners 验证接口 JSON 同时保留旧结果和标准时间线的援军玩家归属。
func TestBattleReportJSONKeepsLegacyAndStandardTraitOwners(t *testing.T) {
	report := NormalizeBattleReport(BattleReport{
		ID: "br_trait_owner_json", PlayerID: "attacker", PlayerFaction: "wei", Type: "attack", Result: "defender_victory",
		TraitTriggered: []string{"longdan_jiuyuan"},
		TraitOutcomes: map[string]TraitOutcomeReport{
			"longdan_jiuyuan": {
				TraitID: "longdan_jiuyuan", Name: "龙胆救援", TraitType: "special", OwnerSide: "reinforcement",
				OwnerPlayerID: "helper_json", OwnerGeneralID: "zhaoyun", Scope: "reinforcement_self",
				Detail: map[string]interface{}{"reducedLosses": map[string]int{"shuInfantry": 20}},
			},
		},
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
	})
	payload, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("Marshal report failed: %v", err)
	}
	var decoded BattleReport
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatalf("Unmarshal report failed: %v", err)
	}
	legacy := decoded.TraitOutcomes["longdan_jiuyuan"]
	if legacy.OwnerPlayerID != "helper_json" || legacy.OwnerGeneralID != "zhaoyun" || legacy.Scope != "reinforcement_self" {
		t.Fatalf("expected legacy trait ownership after JSON round trip, got %+v", legacy)
	}
	if decoded.Detail == nil || len(decoded.Detail.Traits) != 1 || decoded.Detail.Traits[0].OwnerPlayerID != "helper_json" || decoded.Detail.Traits[0].GeneralID != "zhaoyun" {
		t.Fatalf("expected standard trait ownership after JSON round trip, got %+v", decoded.Detail)
	}
}

// TestSharedReinforcementReportKeepsOwnTraitPlayerOwner 验证公开分享援军战报时仍保留自身特性的玩家归属。
func TestSharedReinforcementReportKeepsOwnTraitPlayerOwner(t *testing.T) {
	repo := NewMemoryRepository()
	report := NormalizeBattleReport(BattleReport{
		ID: "br_shared_reinforcement_trait", PlayerID: "helper_share", OwnerPlayerID: "helper_share",
		ViewType: ReportViewReinforcement, OwnerSide: ReportOwnerSideReinforcement, PlayerFaction: "shu",
		Type: "reinforce", BattleType: "reinforcement_battle", Result: "defender_victory",
		DispatchedUnits: map[string]int{"shuInfantry": 100}, LostUnits: map[string]int{"shuInfantry": 80}, SurvivedUnits: map[string]int{"shuInfantry": 20},
		TraitTriggered: []string{"longdan_jiuyuan"},
		TraitOutcomes: map[string]TraitOutcomeReport{
			"longdan_jiuyuan": {
				TraitID: "longdan_jiuyuan", Name: "龙胆救援", TraitType: "special", OwnerSide: "reinforcement",
				OwnerPlayerID: "helper_share", OwnerGeneralID: "zhaoyun", Scope: "reinforcement_self",
				Detail: map[string]interface{}{"lossReductionRate": 0.2, "reducedLosses": map[string]int{"shuInfantry": 20}},
			},
		},
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
	})
	if err := repo.SaveReport(report); err != nil {
		t.Fatalf("SaveReport failed: %v", err)
	}
	svc := NewServiceWithRepository(repo)
	link, err := svc.ShareBattleReport("helper_share", report.ID)
	if err != nil {
		t.Fatalf("ShareBattleReport failed: %v", err)
	}
	shared, err := svc.GetSharedReportByToken(link.Token)
	if err != nil {
		t.Fatalf("GetSharedReportByToken failed: %v", err)
	}
	legacy := shared.TraitOutcomes["longdan_jiuyuan"]
	if legacy.OwnerPlayerID != "helper_share" || legacy.OwnerGeneralID != "zhaoyun" {
		t.Fatalf("expected shared legacy trait ownership, got %+v", legacy)
	}
	if shared.Detail == nil || len(shared.Detail.Traits) != 1 || shared.Detail.Traits[0].OwnerPlayerID != "helper_share" || shared.Detail.Traits[0].OwnerRole != "reinforcement" {
		t.Fatalf("expected shared standard trait ownership, got %+v", shared.Detail)
	}
}

func TestNormalizeBattleReportIncludesAllFactionUnits(t *testing.T) {
	previous := GetFactionUnits("test_report")
	if err := SaveFactionUnits("", "test_report", FactionUnits{
		"testInfantry": {Name: "测试步兵", Category: "infantry"},
		"testCavalry":  {Name: "测试骑兵", Category: "cavalry"},
		"testSiege":    {Name: "测试攻城", Category: "siege"},
	}); err != nil {
		t.Fatalf("SaveFactionUnits failed: %v", err)
	}
	t.Cleanup(func() {
		_ = SaveFactionUnits("", "test_report", previous)
	})

	report := NormalizeBattleReport(BattleReport{
		ID:              "br_full_units",
		PlayerID:        "player_full_units",
		PlayerFaction:   "test_report",
		TargetID:        "npc_full_units",
		Type:            "attack",
		Result:          "attacker_victory",
		DispatchedUnits: map[string]int{"testInfantry": 10},
		LostUnits:       map[string]int{"testInfantry": 2},
		CreatedAt:       time.Now().UTC().Format(time.RFC3339),
	})
	if report.Detail == nil {
		t.Fatal("expected standard detail")
	}
	units := report.Detail.PrimarySide.Units
	if len(units) != 3 {
		t.Fatalf("expected all configured units, got %+v", units)
	}
	if units[0].UnitType != "testInfantry" || units[0].UnitName != "测试步兵" || units[0].Survived != 8 {
		t.Fatalf("unexpected infantry snapshot: %+v", units[0])
	}
	if units[1].UnitType != "testCavalry" || units[1].Dispatched != 0 || units[1].Lost != 0 || units[1].Survived != 0 {
		t.Fatalf("expected cavalry zero snapshot, got %+v", units[1])
	}
	if units[2].UnitType != "testSiege" || units[2].Dispatched != 0 || units[2].Lost != 0 || units[2].Survived != 0 {
		t.Fatalf("expected siege zero snapshot, got %+v", units[2])
	}
}

func TestNormalizeBattleReportBuildsDefenseDetail(t *testing.T) {
	report := NormalizeBattleReport(BattleReport{
		ID:                "br_defense",
		PlayerID:          "player_d",
		PlayerName:        "成都",
		PlayerFaction:     "shu",
		TargetID:          "player_a",
		TargetName:        "许昌（玩家）",
		Type:              "defense",
		ViewType:          ReportViewDefense,
		SourceType:        ReportSourcePlayerCity,
		Result:            "defender_victory",
		PlayerPower:       120,
		EnemyPower:        90,
		DispatchedUnits:   map[string]int{"shuInfantry": 12},
		LostUnits:         map[string]int{"shuInfantry": 1},
		DefenderFaction:   "wei",
		DefenderUnits:     map[string]int{"weiInfantry": 9},
		DefenderLostUnits: map[string]int{"weiInfantry": 9},
		DefenderRevealed:  true,
		PvpAttackerGenerals: []PvpGeneralSnapshot{
			{ID: "caocao", Name: "曹操", Level: 3, Stats: map[string]int{"force": 70}, EffectiveStats: map[string]int{"force": 90}, Attributes: map[string]float64{"attack": 9}, Buffs: map[string]float64{StatAttackBonus: 0.2}},
		},
		PvpDefenderGenerals: []PvpGeneralSnapshot{
			{ID: "liubei", Name: "刘备", Level: 4},
		},
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
	})
	if report.Detail == nil || report.Detail.ViewType != ReportViewDefense {
		t.Fatalf("expected defense detail, got %+v", report.Detail)
	}
	if report.Detail.PrimarySide.Role != "attacker" || report.Detail.SecondarySide == nil || report.Detail.SecondarySide.Role != "defender" {
		t.Fatalf("defense report should show attacker first and defender second, got primary=%+v secondary=%+v", report.Detail.PrimarySide, report.Detail.SecondarySide)
	}
	if report.Title != "许昌（玩家） 攻击 成都" {
		t.Fatalf("defense report title should swap attacker and defender, got %q", report.Title)
	}
	if len(report.Detail.PrimarySide.Generals) != 1 || report.Detail.PrimarySide.Generals[0].ID != "caocao" || report.Detail.PrimarySide.Generals[0].Stats["force"] != 70 || report.Detail.PrimarySide.Generals[0].EffectiveStats["force"] != 90 || report.Detail.PrimarySide.Generals[0].Attributes["attack"] != 9 || report.Detail.PrimarySide.Generals[0].Buffs[StatAttackBonus] != 0.2 {
		t.Fatalf("defense report attacker side should show attacker general, got %+v", report.Detail.PrimarySide.Generals)
	}
	if len(report.Detail.SecondarySide.Generals) != 1 || report.Detail.SecondarySide.Generals[0].ID != "liubei" {
		t.Fatalf("defense report defender side should show defender general, got %+v", report.Detail.SecondarySide.Generals)
	}
}

func TestNormalizeBattleReportInfersOwnerOutcome(t *testing.T) {
	defenderLoss := NormalizeBattleReport(BattleReport{
		ID:          "br_defender_loss",
		PlayerID:    "player_defender_loss",
		Type:        "defense",
		ViewType:    ReportViewDefense,
		Result:      "defender_defeat",
		CreatedAt:   time.Now().UTC().Format(time.RFC3339),
		TargetID:    "player_attacker",
		TargetName:  "攻击方",
		PlayerName:  "防守方",
		SourceType:  ReportSourcePlayerCity,
		BattleType:  "attack",
		PlayerPower: 10,
		EnemyPower:  20,
	})
	if defenderLoss.WinnerSide != ReportWinnerAttacker || defenderLoss.OwnerSide != ReportOwnerSideDefender || defenderLoss.OwnerOutcome != ReportOwnerOutcomeDefeat {
		t.Fatalf("expected legacy defender_defeat to mean attacker victory and owner defeat, got winner=%s ownerSide=%s ownerOutcome=%s", defenderLoss.WinnerSide, defenderLoss.OwnerSide, defenderLoss.OwnerOutcome)
	}
	if defenderLoss.Detail == nil || defenderLoss.Detail.OwnerOutcome != ReportOwnerOutcomeDefeat {
		t.Fatalf("expected detail ownerOutcome synced, got %+v", defenderLoss.Detail)
	}
}

func TestNormalizeBattleReportClassifiesScoutView(t *testing.T) {
	report := NormalizeBattleReport(BattleReport{
		ID:              "br_scout_classify",
		PlayerID:        "player_scout",
		Type:            "scout",
		Result:          "attacker_victory",
		DispatchedUnits: map[string]int{"weiScout": 5},
		LostUnits:       map[string]int{"weiScout": 2},
		CreatedAt:       time.Now().UTC().Format(time.RFC3339),
	})
	if report.ViewType != ReportViewScout || report.OwnerSide != ReportOwnerSideScout || report.OwnerOutcome != ReportOwnerOutcomeIntelSuccess {
		t.Fatalf("expected scout semantic fields, got view=%s ownerSide=%s ownerOutcome=%s", report.ViewType, report.OwnerSide, report.OwnerOutcome)
	}
	scoutExtra, ok := report.Detail.Extra["scout"].(map[string]interface{})
	if !ok || scoutExtra["scoutSent"] != 5 || scoutExtra["scoutReturned"] != 3 {
		t.Fatalf("expected scout extra with sent/lost/returned, got %+v", report.Detail.Extra)
	}
}

func TestNormalizeBattleReportRepairsDefenseDetailGenerals(t *testing.T) {
	report := NormalizeBattleReport(BattleReport{
		ID:            "br_defense_repair",
		PlayerID:      "player_d",
		PlayerName:    "成都",
		PlayerFaction: "shu",
		TargetID:      "player_a",
		TargetName:    "许昌（玩家）",
		Type:          "defense",
		ViewType:      ReportViewDefense,
		SourceType:    ReportSourcePlayerCity,
		Result:        "defender_victory",
		PvpAttackerGenerals: []PvpGeneralSnapshot{
			{ID: "caocao", Name: "曹操", Level: 3},
		},
		PvpDefenderGenerals: []PvpGeneralSnapshot{
			{ID: "liubei", Name: "刘备", Level: 4},
		},
		Detail: &BattleReportDetail{
			PrimarySide: BattleReportSide{
				Role:     "attacker",
				Generals: []BattleReportGeneral{{ID: "liubei", Name: "刘备", Level: 4, Role: "attacker"}},
			},
			SecondarySide: &BattleReportSide{
				Role:     "defender",
				Generals: []BattleReportGeneral{{ID: "caocao", Name: "曹操", Level: 3, Role: "defender"}},
			},
		},
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
	})

	if len(report.Detail.PrimarySide.Generals) != 1 || report.Detail.PrimarySide.Generals[0].ID != "caocao" {
		t.Fatalf("expected repaired attacker general, got %+v", report.Detail.PrimarySide.Generals)
	}
	if report.Detail.SecondarySide == nil || len(report.Detail.SecondarySide.Generals) != 1 || report.Detail.SecondarySide.Generals[0].ID != "liubei" {
		t.Fatalf("expected repaired defender general, got %+v", report.Detail.SecondarySide)
	}
}

func TestNormalizeBattleReportHidesMerchantUnits(t *testing.T) {
	report := NormalizeBattleReport(BattleReport{
		ID:              "br_hide_merchant",
		PlayerID:        "player_hide_merchant",
		PlayerFaction:   "wei",
		TargetID:        "npc_hide_merchant",
		Type:            "attack",
		Result:          "attacker_victory",
		DispatchedUnits: map[string]int{"weiInfantry": 10, "weiMerchant": 5},
		LostUnits:       map[string]int{"weiInfantry": 1, "weiMerchant": 1},
		CreatedAt:       time.Now().UTC().Format(time.RFC3339),
	})
	for _, unit := range report.Detail.PrimarySide.Units {
		if unit.UnitType == "weiMerchant" {
			t.Fatalf("merchant unit should be hidden from battle report: %+v", report.Detail.PrimarySide.Units)
		}
	}
}

// TestNormalizeLegacyReinforcementReportKeepsCompactFallback 验证历史简报缺少完整快照时仍可兼容展示。
func TestNormalizeLegacyReinforcementReportKeepsCompactFallback(t *testing.T) {
	report := NormalizeBattleReport(BattleReport{
		ID:              "br_reinforcement",
		PlayerID:        "player_r",
		PlayerName:      "援军城",
		PlayerFaction:   "wu",
		TargetID:        "player_host",
		TargetName:      "驻扎城",
		Type:            "reinforce",
		Result:          "defender_victory",
		DispatchedUnits: map[string]int{"wuInfantry": 5},
		LostUnits:       map[string]int{"wuInfantry": 3},
		CreatedAt:       time.Now().UTC().Format(time.RFC3339),
	})

	if report.Detail == nil || report.Detail.ViewType != ReportViewReinforcement {
		t.Fatalf("expected reinforcement detail, got %+v", report.Detail)
	}
	if report.Detail.SecondarySide != nil {
		t.Fatalf("legacy reinforcement fallback should keep its compact single-side detail: %+v", report.Detail.SecondarySide)
	}
}

func TestMemoryReportsQueryByViewAndShareToken(t *testing.T) {
	repo := NewMemoryRepository()
	now := time.Now().UTC().Format(time.RFC3339)
	_ = repo.SaveReport(BattleReport{ID: "br_attack", PlayerID: "player_q", Type: "attack", Result: "attacker_victory", CreatedAt: now})
	_ = repo.SaveReport(BattleReport{ID: "br_defense", PlayerID: "player_q", Type: "defense", ViewType: ReportViewDefense, Result: "defender_victory", CreatedAt: now})

	reports, total, err := repo.ListReportsByQuery(BattleReportQuery{PlayerID: "player_q", ViewType: ReportViewDefense, Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("ListReportsByQuery failed: %v", err)
	}
	if total != 1 || len(reports) != 1 || reports[0].ID != "br_defense" {
		t.Fatalf("expected only defense report, total=%d reports=%+v", total, reports)
	}

	link, err := repo.CreateBattleReportShareLink("player_q", "br_defense", "public", time.Time{})
	if err != nil {
		t.Fatalf("CreateBattleReportShareLink failed: %v", err)
	}
	shared, err := repo.GetReportByShareToken(link.Token)
	if err != nil {
		t.Fatalf("GetReportByShareToken failed: %v", err)
	}
	if shared.ID != "br_defense" {
		t.Fatalf("unexpected shared report: %+v", shared)
	}
}

// TestGetHistoricalDefenseReportMergesGeneralExpFromEvent 验证旧防守战报可从同事件进攻战报补齐双方武将经验。
func TestGetHistoricalDefenseReportMergesGeneralExpFromEvent(t *testing.T) {
	repo := NewMemoryRepository()
	service := NewServiceWithRepository(repo)
	now := time.Now().UTC().Format(time.RFC3339)
	eventID := "event_historical_general_exp"
	attackerGenerals := []PvpGeneralSnapshot{{ID: "liubei", Name: "刘备", Level: 3}}
	defenderGenerals := []PvpGeneralSnapshot{{ID: "caocao", Name: "曹操", Level: 11}}
	attackerReport := BattleReport{
		ID: "br_historical_exp_attack", EventID: eventID, PlayerID: "player_attack_exp", OwnerPlayerID: "player_attack_exp",
		ViewType: ReportViewAttack, OwnerSide: ReportOwnerSideAttacker, SourceType: ReportSourcePlayerCity, BattleType: "attack", Type: "attack",
		Result: "defender_victory", GeneralExpGained: 380, GeneralLevelBefore: 1, GeneralLevelAfter: 3,
		PlayerFaction: "shu", PlayerName: "机器战测·蜀", TargetID: "player_defense_exp", TargetName: "神武", DefenderFaction: "wei", DefenderRevealed: true,
		PvpAttackerGenerals: attackerGenerals, PvpDefenderGenerals: defenderGenerals, CreatedAt: now,
	}
	defenderReport := BattleReport{
		ID: "br_historical_exp_defense", EventID: eventID, PlayerID: "player_defense_exp", OwnerPlayerID: "player_defense_exp",
		ViewType: ReportViewDefense, OwnerSide: ReportOwnerSideDefender, SourceType: ReportSourcePlayerCity, BattleType: "attack", Type: "defense",
		Result: "defender_victory", GeneralExpGained: 800, GeneralLevelBefore: 11, GeneralLevelAfter: 11,
		PlayerFaction: "wei", PlayerName: "神武", TargetID: "player_attack_exp", TargetName: "机器战测·蜀", DefenderFaction: "shu", DefenderRevealed: true,
		PvpAttackerGenerals: attackerGenerals, PvpDefenderGenerals: defenderGenerals, CreatedAt: now,
	}
	if err := repo.SaveReport(NormalizeBattleReport(attackerReport)); err != nil {
		t.Fatalf("SaveReport attacker failed: %v", err)
	}
	if err := repo.SaveReport(NormalizeBattleReport(defenderReport)); err != nil {
		t.Fatalf("SaveReport defender failed: %v", err)
	}

	visible, err := service.GetReportForPlayer(defenderReport.PlayerID, defenderReport.ID)
	if err != nil {
		t.Fatalf("GetReportForPlayer failed: %v", err)
	}
	if visible.Detail == nil || visible.Detail.SecondarySide == nil {
		t.Fatalf("expected complete defense detail, got %+v", visible.Detail)
	}
	attackerGeneral := visible.Detail.PrimarySide.Generals[0]
	defenderGeneral := visible.Detail.SecondarySide.Generals[0]
	if attackerGeneral.GeneralExpGained == nil || *attackerGeneral.GeneralExpGained != 380 || attackerGeneral.GeneralLevelBefore == nil || *attackerGeneral.GeneralLevelBefore != 1 || attackerGeneral.GeneralLevelAfter == nil || *attackerGeneral.GeneralLevelAfter != 3 {
		t.Fatalf("expected historical attacker result 380 and Lv1->3, got %+v", attackerGeneral)
	}
	if defenderGeneral.GeneralExpGained == nil || *defenderGeneral.GeneralExpGained != 800 {
		t.Fatalf("expected historical defender result 800, got %+v", defenderGeneral)
	}
}

// TestSynchronizeBattleReportGeneralResultsSeparatesMirrorGenerals 验证敌我同武将 ID 时经验按绝对角色隔离。
func TestSynchronizeBattleReportGeneralResultsSeparatesMirrorGenerals(t *testing.T) {
	now := time.Now().UTC().Format(time.RFC3339)
	sharedAttacker := []PvpGeneralSnapshot{{ID: "caocao", Name: "进攻曹操", Level: 2}}
	sharedDefender := []PvpGeneralSnapshot{{ID: "caocao", Name: "防守曹操", Level: 5}}
	reports := synchronizeBattleReportGeneralResults([]BattleReport{
		{ID: "br_mirror_attack", EventID: "event_mirror_exp", PlayerID: "mirror_a", OwnerPlayerID: "mirror_a", ViewType: ReportViewAttack, OwnerSide: ReportOwnerSideAttacker, SourceType: ReportSourcePlayerCity, Type: "attack", Result: "draw", GeneralExpGained: 100, GeneralLevelBefore: 1, GeneralLevelAfter: 2, PvpAttackerGenerals: sharedAttacker, PvpDefenderGenerals: sharedDefender, DefenderRevealed: true, CreatedAt: now},
		{ID: "br_mirror_defense", EventID: "event_mirror_exp", PlayerID: "mirror_d", OwnerPlayerID: "mirror_d", ViewType: ReportViewDefense, OwnerSide: ReportOwnerSideDefender, SourceType: ReportSourcePlayerCity, Type: "defense", Result: "draw", GeneralExpGained: 250, GeneralLevelBefore: 4, GeneralLevelAfter: 5, PvpAttackerGenerals: sharedAttacker, PvpDefenderGenerals: sharedDefender, DefenderRevealed: true, CreatedAt: now},
	})
	defense := reports[1]
	if defense.Detail == nil || defense.Detail.SecondarySide == nil {
		t.Fatalf("expected mirror defense detail, got %+v", defense.Detail)
	}
	attacker := defense.Detail.PrimarySide.Generals[0]
	defender := defense.Detail.SecondarySide.Generals[0]
	if attacker.GeneralExpGained == nil || *attacker.GeneralExpGained != 100 {
		t.Fatalf("expected mirror attacker exp 100, got %+v", attacker)
	}
	if defender.GeneralExpGained == nil || *defender.GeneralExpGained != 250 {
		t.Fatalf("expected mirror defender exp 250, got %+v", defender)
	}
}

func TestMemoryReportsVisibleCapAppliesPerView(t *testing.T) {
	repo := NewMemoryRepository()
	now := time.Now().UTC()
	playerID := "player_memory_cap"
	for i := 0; i < battleReportVisibleCapPerView+2; i++ {
		report := BattleReport{
			ID:        fmt.Sprintf("br_memory_cap_%05d", i),
			PlayerID:  playerID,
			Type:      "attack",
			ViewType:  ReportViewAttack,
			Result:    "attacker_victory",
			CreatedAt: now.Add(time.Duration(i) * time.Second).Format(time.RFC3339),
		}
		if err := repo.SaveReport(report); err != nil {
			t.Fatalf("SaveReport attack failed: %v", err)
		}
	}
	if err := repo.SaveReport(BattleReport{
		ID:        "br_memory_cap_defense",
		PlayerID:  playerID,
		Type:      "defense",
		ViewType:  ReportViewDefense,
		Result:    "defender_victory",
		CreatedAt: now.Add(time.Hour).Format(time.RFC3339),
	}); err != nil {
		t.Fatalf("SaveReport defense failed: %v", err)
	}

	allReports, err := repo.ListAllReports(playerID)
	if err != nil {
		t.Fatalf("ListAllReports failed: %v", err)
	}
	visibleAttack := 0
	for _, report := range allReports {
		report = NormalizeBattleReport(report)
		if report.ViewType == ReportViewAttack && !report.DeletedByPlayer {
			visibleAttack++
		}
	}
	if visibleAttack != battleReportVisibleCapPerView {
		t.Fatalf("expected attack reports capped at %d, got %d", battleReportVisibleCapPerView, visibleAttack)
	}
	if _, err := repo.GetReportForPlayer(playerID, "br_memory_cap_00000"); err == nil {
		t.Fatal("expected oldest attack report to be soft deleted by visible cap")
	}
	defenseReports, defenseTotal, err := repo.ListReportsByQuery(BattleReportQuery{PlayerID: playerID, ViewType: ReportViewDefense, Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("ListReportsByQuery defense failed: %v", err)
	}
	if defenseTotal != 1 || len(defenseReports) != 1 || defenseReports[0].ID != "br_memory_cap_defense" {
		t.Fatalf("expected defense report unaffected by attack cap, total=%d reports=%+v", defenseTotal, defenseReports)
	}
}

func TestCreateBattleReportsUsesStandardBatchEntry(t *testing.T) {
	repo := NewMemoryRepository()
	service := NewServiceWithRepository(repo)
	now := time.Now().UTC().Format(time.RFC3339)
	result, err := service.CreateBattleReports(BattleReportCreateInput{
		EventID:                "event_create_batch",
		SourceType:             ReportSourcePlayerCity,
		SourceID:               "source_create_batch",
		BattleType:             "attack",
		Result:                 "attacker_victory",
		RelatedMarchID:         "march_create_batch",
		RelatedReinforcementID: "reinforcement_create_batch",
		OccurredAt:             now,
		Extra:                  map[string]interface{}{"trace": "complete"},
		Reports: []BattleReport{
			{
				ID:               "br_create_attacker",
				PlayerID:         "player_create_attacker",
				PlayerName:       "攻击方",
				PlayerFaction:    "wei",
				TargetID:         "player_create_defender",
				TargetName:       "防守方",
				Type:             "attack",
				ViewType:         ReportViewAttack,
				DispatchedUnits:  map[string]int{"weiInfantry": 10},
				DefenderFaction:  "shu",
				DefenderUnits:    map[string]int{"shuInfantry": 5},
				DefenderRevealed: true,
			},
			{
				ID:               "br_create_defender",
				PlayerID:         "player_create_defender",
				PlayerName:       "防守方",
				PlayerFaction:    "shu",
				TargetID:         "player_create_attacker",
				TargetName:       "攻击方",
				Type:             "defense",
				ViewType:         ReportViewDefense,
				DispatchedUnits:  map[string]int{"shuInfantry": 5},
				DefenderFaction:  "wei",
				DefenderUnits:    map[string]int{"weiInfantry": 10},
				DefenderRevealed: true,
			},
		},
	})
	if err != nil {
		t.Fatalf("CreateBattleReports failed: %v", err)
	}
	if result.Event.ID != "event_create_batch" || len(result.Reports) != 2 {
		t.Fatalf("unexpected create result: %+v", result)
	}
	stored, err := repo.ListReportsByEventForAdmin("event_create_batch")
	if err != nil {
		t.Fatalf("ListReportsByEventForAdmin failed: %v", err)
	}
	if len(stored) != 2 {
		t.Fatalf("expected two reports stored under event, got %+v", stored)
	}
	if stored[0].Detail == nil || stored[1].Detail == nil {
		t.Fatalf("expected stored reports to have standard detail: %+v", stored)
	}
	adminEvent, err := repo.GetBattleEventForAdmin("event_create_batch")
	if err != nil || adminEvent.SourceID != "source_create_batch" || adminEvent.RelatedMarchID != "march_create_batch" || adminEvent.RelatedReinforcementID != "reinforcement_create_batch" || adminEvent.Summary["trace"] != "complete" {
		t.Fatalf("expected complete admin event input persisted, got event=%+v err=%v", adminEvent, err)
	}
	context, err := service.GetReportEventForPlayer("player_create_attacker", "br_create_attacker")
	if err != nil {
		t.Fatalf("GetReportEventForPlayer failed: %v", err)
	}
	if context.Event.ID != "event_create_batch" || len(context.Reports) != 1 || context.Reports[0].PlayerID != "player_create_attacker" {
		t.Fatalf("expected player-visible event context only, got %+v", context)
	}
	if context.Event.SourceID != "source_create_batch" || context.Event.RelatedMarchID != "march_create_batch" || context.Event.RelatedReinforcementID != "reinforcement_create_batch" || context.Event.Summary != nil || context.Event.Snapshot != nil || context.Event.ResultData != nil {
		t.Fatalf("expected player event index without GM diagnostic snapshots, got %+v", context.Event)
	}
}

func TestEnemyLossRevealThresholdUsesModifierPipeline(t *testing.T) {
	now := time.Now().UTC()
	state := newPlayerState("player_beacon_threshold", "烽火", "shu", "liubei", now)
	state.Buildings = []Building{{ID: "beacon_tower-20", Type: "beacon_tower", Level: 20}}
	threshold := enemyLossRevealThreshold(&state, now)
	if threshold < 0.44 || threshold > 0.46 {
		t.Fatalf("expected beacon tower to raise threshold to about 0.45, got %.2f", threshold)
	}
}

func TestReportVisibilityUsesLossThresholdSnapshot(t *testing.T) {
	report := NormalizeBattleReport(BattleReport{
		ID:                       "br_hidden",
		PlayerID:                 "player_hidden",
		TargetID:                 "player_target",
		TargetName:               "目标",
		Type:                     "attack",
		Result:                   "defender_victory",
		DefenderRevealed:         false,
		EnemyLossRevealThreshold: 0.4,
		EnemyLossRatio:           0.3,
		CreatedAt:                time.Now().UTC().Format(time.RFC3339),
	})
	if report.Detail == nil || report.Detail.Visibility.ShowEnemyRemainingUnits {
		t.Fatalf("expected enemy remaining units hidden, got %+v", report.Detail)
	}
	if report.Detail.Visibility.Threshold != 0.4 || report.Detail.Visibility.ActualLossRatio != 0.3 {
		t.Fatalf("expected threshold snapshot in visibility, got %+v", report.Detail.Visibility)
	}
}

// TestGetReportForPlayerRedactsHiddenEnemyTroops 验证玩家详情响应不会携带不可见的敌方剩余兵力。
func TestGetReportForPlayerRedactsHiddenEnemyTroops(t *testing.T) {
	enemyGeneralExp := 999
	repo := NewMemoryRepository()
	service := NewServiceWithRepository(repo)
	report := NormalizeBattleReport(BattleReport{
		ID:                  "br_hidden_response",
		PlayerID:            "player_hidden_response",
		OwnerPlayerID:       "player_hidden_response",
		ViewType:            ReportViewAttack,
		SourceType:          ReportSourcePlayerCity,
		BattleType:          "attack",
		Type:                "attack",
		Result:              "defender_victory",
		TargetID:            "player_hidden_target",
		TargetName:          "隐藏目标",
		DefenderUnits:       map[string]int{"shuInfantry": 100},
		DefenderLostUnits:   map[string]int{"shuInfantry": 10},
		DefenderResources:   map[string]int{"wood": 999},
		PvpDefenderGenerals: []PvpGeneralSnapshot{{ID: "liubei", Name: "刘备", GeneralExpGained: &enemyGeneralExp}},
		PvpReinforcements: []DefenseReinforcementUnit{{
			ReinforcementID:    "rein_hidden",
			Troops:             map[string]int{"shuInfantry": 50},
			Generals:           []ReinforcementGeneralSnapshot{{ID: "guanyu", Name: "关羽"}},
			GeneralExpGained:   88,
			GeneralLevelBefore: 10,
			GeneralLevelAfter:  11,
			Buffs:              []ModifierBreakdownItem{{Source: "关羽", Key: StatAttackBonus, Value: 0.2, Mode: "percentAdd"}},
		}},
		PvpAttackerGenerals: []PvpGeneralSnapshot{{ID: "caocao", Name: "曹操"}},
		TraitTriggered:      []string{"attacker_trait", "defender_trait"},
		TraitOutcomes: map[string]TraitOutcomeReport{
			"attacker_trait": {TraitID: "attacker_trait", Name: "攻击特性", OwnerSide: "attacker", OwnerGeneralID: "caocao"},
			"defender_trait": {TraitID: "defender_trait", Name: "防守特性", OwnerSide: "defender", OwnerGeneralID: "liubei"},
		},
		PvpWall:          &PvpWallSnapshot{Faction: "shu", Level: 10},
		DefenderRevealed: false,
		CreatedAt:        time.Now().UTC().Format(time.RFC3339),
	})
	report.Detail.Visibility.ShowEnemyResources = false
	report.Detail.Visibility.ShowEnemyGenerals = false
	report.Detail.Visibility.ShowEnemyCityDefense = false
	if err := repo.SaveReport(report); err != nil {
		t.Fatalf("SaveReport failed: %v", err)
	}

	visible, err := service.GetReportForPlayer(report.PlayerID, report.ID)
	if err != nil {
		t.Fatalf("GetReportForPlayer failed: %v", err)
	}
	if len(visible.DefenderUnits) != 0 {
		t.Fatalf("expected legacy enemy troops redacted, got %+v", visible.DefenderUnits)
	}
	if visible.Detail == nil || visible.Detail.SecondarySide == nil || len(visible.Detail.SecondarySide.Units) != 0 {
		t.Fatalf("expected standard enemy troop matrix redacted, got %+v", visible.Detail)
	}
	if len(visible.Detail.Traits) != 1 || visible.Detail.Traits[0].TraitID != "attacker_trait" {
		t.Fatalf("expected enemy traits redacted with enemy generals, got %+v", visible.Detail.Traits)
	}
	if len(visible.TraitTriggered) != 1 || visible.TraitTriggered[0] != "attacker_trait" || len(visible.TraitOutcomes) != 1 {
		t.Fatalf("expected legacy enemy trait payload redacted, got triggered=%+v outcomes=%+v", visible.TraitTriggered, visible.TraitOutcomes)
	}
	legacyOnly := report
	legacyDetail := *report.Detail
	legacyDetail.Traits = nil
	legacyOnly.Detail = &legacyDetail
	legacyVisible := projectBattleReportForViewer(legacyOnly)
	if len(legacyVisible.TraitTriggered) != 1 || legacyVisible.TraitTriggered[0] != "attacker_trait" || len(legacyVisible.TraitOutcomes) != 1 {
		t.Fatalf("expected legacy-only enemy traits redacted, got triggered=%+v outcomes=%+v", legacyVisible.TraitTriggered, legacyVisible.TraitOutcomes)
	}
	if len(visible.DefenderResources) != 0 || len(visible.PvpDefenderGenerals) != 0 || len(visible.PvpReinforcements[0].Troops) != 0 || len(visible.PvpReinforcements[0].Generals) != 0 || visible.PvpReinforcements[0].GeneralExpGained != 0 || visible.PvpReinforcements[0].GeneralLevelBefore != 0 || visible.PvpReinforcements[0].GeneralLevelAfter != 0 || len(visible.PvpReinforcements[0].Buffs) != 0 || visible.PvpWall != nil {
		t.Fatalf("expected enemy resources, generals and reinforcement troops redacted, got %+v", visible)
	}
	if len(visible.Detail.SecondarySide.Generals) != 0 {
		t.Fatalf("expected enemy general exp redacted together with enemy generals, got %+v", visible.Detail.SecondarySide.Generals)
	}
	listed, err := service.ListReports(report.PlayerID, 1, 10)
	if err != nil || len(listed.Reports) != 1 || len(listed.Reports[0].PvpDefenderGenerals) != 0 {
		t.Fatalf("expected unfiltered report list to redact enemy general exp, page=%+v err=%v", listed, err)
	}
	queried, err := service.ListReportsByQuery(BattleReportQuery{PlayerID: report.PlayerID, ViewType: ReportViewAttack, Page: 1, PageSize: 10})
	if err != nil || len(queried.Reports) != 1 || len(queried.Reports[0].PvpDefenderGenerals) != 0 {
		t.Fatalf("expected filtered report list to redact enemy general exp, page=%+v err=%v", queried, err)
	}
	if pvp, ok := visible.Detail.Extra["pvp"].(map[string]interface{}); ok {
		if _, exists := pvp["wall"]; exists {
			t.Fatalf("expected enemy wall snapshot redacted, got %+v", pvp)
		}
		if reinforcements, ok := pvp["reinforcements"].([]interface{}); ok && len(reinforcements) > 0 {
			if snapshot, ok := reinforcements[0].(map[string]interface{}); ok {
				if _, expExists := snapshot["generalExpGained"]; expExists {
					t.Fatalf("expected enemy reinforcement general exp redacted, got %+v", snapshot)
				}
				if _, beforeExists := snapshot["generalLevelBefore"]; beforeExists {
					t.Fatalf("expected enemy reinforcement pre-battle level redacted, got %+v", snapshot)
				}
				if _, afterExists := snapshot["generalLevelAfter"]; afterExists {
					t.Fatalf("expected enemy reinforcement post-battle level redacted, got %+v", snapshot)
				}
				if _, buffsExist := snapshot["buffs"]; buffsExist {
					t.Fatalf("expected enemy reinforcement buffs redacted, got %+v", snapshot)
				}
			}
		}
	}
	raw, err := repo.GetReportForPlayer(report.PlayerID, report.ID)
	if err != nil || len(raw.DefenderUnits) == 0 || len(raw.PvpReinforcements[0].Troops) == 0 || raw.PvpReinforcements[0].GeneralExpGained != 88 || raw.PvpReinforcements[0].GeneralLevelBefore != 10 || raw.PvpReinforcements[0].GeneralLevelAfter != 11 || len(raw.PvpReinforcements[0].Buffs) != 1 {
		t.Fatalf("expected repository raw snapshot unchanged, got report=%+v err=%v", raw, err)
	}
}

// TestCloneBattleReportDeepCopiesNestedSnapshots 验证玩家响应副本的嵌套修改不会污染仓储中的历史战报。
func TestCloneBattleReportDeepCopiesNestedSnapshots(t *testing.T) {
	trait := GeneralTraitInstance{
		TraitID:       "weiwu_tongyu",
		Params:        map[string]float64{"attackBonusRate": 0.1},
		AllowedSides:  []string{"attacker"},
		AllowedScenes: []string{"attack"},
	}
	original := BattleReport{
		CapturedUnits:          map[string]int{"weiInfantry": 1},
		PvpReinforcementLosses: map[string]map[string]int{"rein_1": {"shuInfantry": 2}},
		GrantedRewards:         []Reward{{Type: "item", ID: "token", Amount: 1, Metadata: map[string]any{"source": "battle"}}},
		TraitTriggered:         []string{"weiwu_tongyu"},
		TraitOutcomes: map[string]TraitOutcomeReport{
			"weiwu_tongyu": {TraitID: "weiwu_tongyu", Detail: map[string]interface{}{"attackModifiedUnits": map[string]int{"weiInfantry": 1}}},
		},
		PvpReinforcements: []DefenseReinforcementUnit{{
			ReinforcementID: "rein_1",
			Troops:          map[string]int{"shuInfantry": 10},
			Generals: []ReinforcementGeneralSnapshot{{
				ID: "guanyu", Buffs: map[string]float64{StatAttackBonus: 0.2}, Traits: []GeneralTraitInstance{trait},
			}},
			SourceTags: map[string]string{"scene": "pvp"},
		}},
		Detail: &BattleReportDetail{
			PrimarySide: BattleReportSide{Generals: []BattleReportGeneral{{
				ID: "caocao", Stats: map[string]int{"force": 70}, EffectiveStats: map[string]int{"force": 90}, Attributes: map[string]float64{"attack": 9}, Buffs: map[string]float64{StatAttackBonus: 0.1}, Traits: []GeneralTraitInstance{trait},
			}}},
			Rewards: BattleReportRewards{
				Resources: map[string]int{"wood": 100},
				Granted:   []Reward{{Type: "general_exp", ID: "caocao", Amount: 10, Metadata: map[string]any{"source": "battle"}}},
			},
			Traits: []BattleReportTrait{{
				TraitID: "weiwu_tongyu", Detail: map[string]interface{}{"attackModifiedUnits": map[string]int{"weiInfantry": 1}},
			}},
			Extra: map[string]interface{}{
				"pvp": map[string]interface{}{
					"reinforcements": []DefenseReinforcementUnit{{
						ReinforcementID: "rein_extra", Troops: map[string]int{"shuInfantry": 10},
					}},
					"reinforcementLosses": map[string]map[string]int{"rein_extra": {"shuInfantry": 2}},
					"wall":                &PvpWallSnapshot{Level: 10},
				},
				"sweep": map[string]interface{}{
					"defenders": []BattleReportSweepDefender{{
						TargetID: "npc_1", Units: []BattleReportUnit{{UnitType: "shuInfantry", Dispatched: 10}},
						Resources: map[string]int{"wood": 5},
					}},
				},
			},
		},
	}

	cloned := cloneBattleReport(original)
	cloned.CapturedUnits["weiInfantry"] = 9
	cloned.PvpReinforcementLosses["rein_1"]["shuInfantry"] = 9
	cloned.GrantedRewards[0].Metadata["source"] = "response"
	cloned.TraitTriggered[0] = "changed"
	cloned.TraitOutcomes["weiwu_tongyu"].Detail["attackModifiedUnits"].(map[string]int)["weiInfantry"] = 9
	cloned.PvpReinforcements[0].Generals[0].Buffs[StatAttackBonus] = 9
	cloned.PvpReinforcements[0].Generals[0].Traits[0].Params["attackBonusRate"] = 9
	cloned.PvpReinforcements[0].SourceTags["scene"] = "changed"
	cloned.Detail.PrimarySide.Generals[0].Stats["force"] = 7
	cloned.Detail.PrimarySide.Generals[0].EffectiveStats["force"] = 9
	cloned.Detail.PrimarySide.Generals[0].Attributes["attack"] = 1
	cloned.Detail.PrimarySide.Generals[0].Buffs[StatAttackBonus] = 9
	cloned.Detail.PrimarySide.Generals[0].Traits[0].AllowedSides[0] = "defender"
	cloned.Detail.Rewards.Resources["wood"] = 9
	cloned.Detail.Rewards.Granted[0].Metadata["source"] = "response"
	cloned.Detail.Traits[0].Detail["attackModifiedUnits"].(map[string]int)["weiInfantry"] = 9
	clonedExtraPvp := cloned.Detail.Extra["pvp"].(map[string]interface{})
	clonedExtraPvp["reinforcements"].([]DefenseReinforcementUnit)[0].Troops["shuInfantry"] = 9
	clonedExtraPvp["reinforcementLosses"].(map[string]map[string]int)["rein_extra"]["shuInfantry"] = 9
	clonedExtraPvp["wall"].(*PvpWallSnapshot).Level = 99
	clonedSweep := cloned.Detail.Extra["sweep"].(map[string]interface{})["defenders"].([]BattleReportSweepDefender)
	clonedSweep[0].Units[0].Dispatched = 99
	clonedSweep[0].Resources["wood"] = 99

	if original.CapturedUnits["weiInfantry"] != 1 || original.PvpReinforcementLosses["rein_1"]["shuInfantry"] != 2 || original.GrantedRewards[0].Metadata["source"] != "battle" || original.TraitTriggered[0] != "weiwu_tongyu" {
		t.Fatalf("expected top-level report snapshots to remain isolated, original=%+v", original)
	}
	if original.TraitOutcomes["weiwu_tongyu"].Detail["attackModifiedUnits"].(map[string]int)["weiInfantry"] != 1 || original.PvpReinforcements[0].Generals[0].Buffs[StatAttackBonus] != 0.2 || original.PvpReinforcements[0].Generals[0].Traits[0].Params["attackBonusRate"] != 0.1 || original.PvpReinforcements[0].SourceTags["scene"] != "pvp" {
		t.Fatalf("expected trait and reinforcement snapshots to remain isolated, original=%+v", original)
	}
	if original.Detail.PrimarySide.Generals[0].Stats["force"] != 70 || original.Detail.PrimarySide.Generals[0].EffectiveStats["force"] != 90 || original.Detail.PrimarySide.Generals[0].Attributes["attack"] != 9 || original.Detail.PrimarySide.Generals[0].Buffs[StatAttackBonus] != 0.1 || original.Detail.PrimarySide.Generals[0].Traits[0].AllowedSides[0] != "attacker" || original.Detail.Rewards.Resources["wood"] != 100 || original.Detail.Rewards.Granted[0].Metadata["source"] != "battle" || original.Detail.Traits[0].Detail["attackModifiedUnits"].(map[string]int)["weiInfantry"] != 1 {
		t.Fatalf("expected standard detail snapshots to remain isolated, detail=%+v", original.Detail)
	}
	originalExtraPvp := original.Detail.Extra["pvp"].(map[string]interface{})
	originalSweep := original.Detail.Extra["sweep"].(map[string]interface{})["defenders"].([]BattleReportSweepDefender)
	if originalExtraPvp["reinforcements"].([]DefenseReinforcementUnit)[0].Troops["shuInfantry"] != 10 || originalExtraPvp["reinforcementLosses"].(map[string]map[string]int)["rein_extra"]["shuInfantry"] != 2 || originalExtraPvp["wall"].(*PvpWallSnapshot).Level != 10 || originalSweep[0].Units[0].Dispatched != 10 || originalSweep[0].Resources["wood"] != 5 {
		t.Fatalf("expected standard extra snapshots to remain isolated, extra=%+v", original.Detail.Extra)
	}
}

// TestReportViewerAndShareResponsesDoNotMutateStoredExtra 验证本人查询和公开分享的扩展字段修改不会污染库存战报。
func TestReportViewerAndShareResponsesDoNotMutateStoredExtra(t *testing.T) {
	repo := NewMemoryRepository()
	report := NormalizeBattleReport(BattleReport{
		ID: "br_extra_response_isolation", PlayerID: "player_extra", OwnerPlayerID: "player_extra",
		ViewType: ReportViewAttack, OwnerSide: ReportOwnerSideAttacker, SourceType: ReportSourceDungeon,
		BattleType: "dungeon_reincarnation_attack", Type: "attack", Result: "attacker_victory",
		DefenderRevealed: true, CreatedAt: time.Now().UTC().Format(time.RFC3339),
	})
	report.Detail.Extra = map[string]interface{}{
		"dungeon": map[string]interface{}{"rewardMode": "preview"},
		"sweep": map[string]interface{}{
			"defenders": []BattleReportSweepDefender{{
				TargetID: "npc_extra", Resources: map[string]int{"wood": 5},
			}},
		},
	}
	if err := repo.SaveReport(report); err != nil {
		t.Fatalf("SaveReport failed: %v", err)
	}
	svc := NewServiceWithRepository(repo)

	visible, err := svc.GetReportForPlayer(report.PlayerID, report.ID)
	if err != nil {
		t.Fatalf("GetReportForPlayer failed: %v", err)
	}
	visible.Detail.Extra["dungeon"].(map[string]interface{})["rewardMode"] = "response"
	rawAfterViewer, err := repo.GetReportForPlayer(report.PlayerID, report.ID)
	if err != nil || rawAfterViewer.Detail.Extra["dungeon"].(map[string]interface{})["rewardMode"] != "preview" {
		t.Fatalf("expected viewer response isolated from stored extra, report=%+v err=%v", rawAfterViewer.Detail, err)
	}

	link, err := svc.ShareBattleReport(report.PlayerID, report.ID)
	if err != nil {
		t.Fatalf("ShareBattleReport failed: %v", err)
	}
	shared, err := svc.GetSharedReportByToken(link.Token)
	if err != nil {
		t.Fatalf("GetSharedReportByToken failed: %v", err)
	}
	sharedDefenders := shared.Detail.Extra["sweep"].(map[string]interface{})["defenders"].([]BattleReportSweepDefender)
	sharedDefenders[0].Resources["wood"] = 99
	rawAfterShare, err := repo.GetReportForPlayer(report.PlayerID, report.ID)
	if err != nil {
		t.Fatalf("GetReportForPlayer after share failed: %v", err)
	}
	rawDefenders := rawAfterShare.Detail.Extra["sweep"].(map[string]interface{})["defenders"].([]BattleReportSweepDefender)
	if rawDefenders[0].Resources["wood"] != 5 {
		t.Fatalf("expected shared response isolated from stored extra, defenders=%+v", rawDefenders)
	}
}

// TestDefenseReportTraitsBelongToDefender 验证防守视角不会把守方武将特性显示到攻击方。
func TestDefenseReportTraitsBelongToDefender(t *testing.T) {
	report := BattleReport{
		ID:                  "br_trait_defender",
		PlayerID:            "player_trait_defender",
		OwnerPlayerID:       "player_trait_defender",
		ViewType:            ReportViewDefense,
		OwnerSide:           ReportOwnerSideDefender,
		SourceType:          ReportSourceYellowTurban,
		BattleType:          BattleTypeYellowTurban,
		Type:                "defense",
		Result:              "defender_victory",
		PlayerName:          "守城玩家",
		PlayerFaction:       "shu",
		TargetID:            "yellow_turban",
		TargetName:          "黄巾军",
		DefenderFaction:     "wei",
		PvpDefenderGenerals: []PvpGeneralSnapshot{{ID: "zhugeliang", Name: "诸葛亮", Level: 61}},
		TraitTriggered:      []string{"qimen"},
		TraitOutcomes: map[string]TraitOutcomeReport{
			"qimen": {TraitID: "qimen", Name: "奇门遁甲", OwnerGeneralID: "zhugeliang"},
		},
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
	}
	detail := BuildBattleReportDetail(report)
	if len(detail.Traits) != 1 {
		t.Fatalf("expected one defender trait, got %+v", detail.Traits)
	}
	trait := detail.Traits[0]
	if trait.OwnerSide != "secondary" || trait.OwnerRole != "defender" || trait.GeneralID != "zhugeliang" || trait.GeneralName != "诸葛亮" || trait.Summary != "" {
		t.Fatalf("expected defender trait attribution without duplicated summary, got %+v", trait)
	}
}
