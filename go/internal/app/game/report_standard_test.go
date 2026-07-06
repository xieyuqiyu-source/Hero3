// 本文件测试标准战报结构和兼容查询行为。
package game

import (
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
			{ID: "caocao", Name: "曹操", Level: 3},
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
	if len(report.Detail.PrimarySide.Generals) != 1 || report.Detail.PrimarySide.Generals[0].ID != "caocao" {
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

func TestNormalizeBattleReportBuildsReinforcementDetailWithoutSecondarySide(t *testing.T) {
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
		t.Fatalf("reinforcement report should not expose secondary side: %+v", report.Detail.SecondarySide)
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
		EventID:    "event_create_batch",
		SourceType: ReportSourcePlayerCity,
		BattleType: "attack",
		Result:     "attacker_victory",
		OccurredAt: now,
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
	context, err := service.GetReportEventForPlayer("player_create_attacker", "br_create_attacker")
	if err != nil {
		t.Fatalf("GetReportEventForPlayer failed: %v", err)
	}
	if context.Event.ID != "event_create_batch" || len(context.Reports) != 1 || context.Reports[0].PlayerID != "player_create_attacker" {
		t.Fatalf("expected player-visible event context only, got %+v", context)
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
