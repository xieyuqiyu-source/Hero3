// 本文件验证战报系统重构后的 MySQL 事件、战报、状态和分享链路。
package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/joho/godotenv"

	"hero3/internal/app/game"
)

// openReportTestRepository 打开本地 test_ MySQL 仓储，避免误连生产库。
func openReportTestRepository(t *testing.T) (*MySQLRepository, *sql.DB) {
	t.Helper()
	_ = godotenv.Load("../../../.env")
	dsn := strings.TrimSpace(os.Getenv("HERO3_DATABASE_DSN"))
	if dsn == "" {
		t.Skip("HERO3_DATABASE_DSN is not configured")
	}
	databaseName, err := MySQLDatabaseName(dsn)
	if err != nil {
		t.Fatalf("parse mysql dsn: %v", err)
	}
	if !strings.HasPrefix(databaseName, "test_") {
		t.Skipf("skip report integration test on non-test database %s", databaseName)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	db, err := OpenMySQL(ctx, dsn)
	if err != nil {
		t.Fatalf("open mysql: %v", err)
	}
	if err := MigrateMySQL(ctx, db); err != nil {
		_ = db.Close()
		t.Fatalf("migrate mysql: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})
	return NewMySQLRepository(db), db
}

// TestBattleReportBodyJSONOmitsDetail 验证 report_json 不再重复保存 detail_json 中的重型详情。
func TestBattleReportBodyJSONOmitsDetail(t *testing.T) {
	report := game.BattleReport{
		ID:               "br_compact_detail",
		PlayerID:         "player_compact_detail",
		ViewType:         game.ReportViewAttack,
		SourceType:       game.ReportSourceNPCCity,
		BattleType:       "sweep",
		Type:             "attack",
		Result:           "attacker_victory",
		TargetID:         "npc_sweep",
		TargetName:       "NPC 扫荡",
		DefenderRevealed: true,
		CreatedAt:        time.Now().UTC().Format(time.RFC3339),
		Detail: &game.BattleReportDetail{
			Title:   "NPC 扫荡",
			Summary: "扫荡完成",
			PrimarySide: game.BattleReportSide{
				PlayerID: "player_compact_detail",
				Role:     "attacker",
				Units:    []game.BattleReportUnit{{UnitType: "weiInfantry", AmountBefore: 10}},
			},
			Rewards: game.BattleReportRewards{},
			Extra: map[string]interface{}{
				"sweep": map[string]interface{}{
					"defenders": []game.BattleReportSweepDefender{{TargetID: "npc_1", TargetName: "NPC 1"}},
				},
			},
		},
	}

	bodyJSON, err := marshalBattleReportBodyJSON(report)
	if err != nil {
		t.Fatalf("marshal body: %v", err)
	}
	var body map[string]interface{}
	if err := json.Unmarshal(bodyJSON, &body); err != nil {
		t.Fatalf("unmarshal body: %v", err)
	}
	if _, ok := body["detail"]; ok {
		t.Fatalf("expected report_json body to omit detail, got %s", string(bodyJSON))
	}
	detailJSON, err := json.Marshal(report.Detail)
	if err != nil {
		t.Fatalf("marshal detail: %v", err)
	}
	restored, err := scanBattleReportJSON(bodyJSON, detailJSON, false)
	if err != nil {
		t.Fatalf("scan report: %v", err)
	}
	if restored.Detail == nil || restored.Detail.Extra["sweep"] == nil {
		t.Fatalf("expected detail_json to be restored, got %+v", restored.Detail)
	}
}

// TestListReportsRepairsMissingSweepReport 验证军情列表会从扫荡任务快照补写缺失战报。
func TestListReportsRepairsMissingSweepReport(t *testing.T) {
	repo, db := openReportTestRepository(t)
	now := time.Now().UTC()
	suffix := strings.NewReplacer(".", "_").Replace(now.Format("150405.000000"))
	accountID := "it_repair_acc_" + suffix
	playerID := "it_repair_player_" + suffix
	taskID := "it_repair_task_" + suffix
	reportID := "it_repair_report_" + suffix
	eventID := "it_repair_event_" + suffix
	t.Cleanup(func() {
		_, _ = db.Exec(`DELETE FROM npc_sweep_tasks WHERE id = ?`, taskID)
		_, _ = db.Exec(`DELETE FROM battle_report_states WHERE report_id = ?`, reportID)
		_, _ = db.Exec(`DELETE FROM battle_report_participants WHERE report_id = ?`, reportID)
		_, _ = db.Exec(`DELETE FROM battle_reports WHERE id = ?`, reportID)
		_, _ = db.Exec(`DELETE FROM battle_events WHERE id = ?`, eventID)
		_, _ = db.Exec(`DELETE FROM players WHERE id = ?`, playerID)
		_, _ = db.Exec(`DELETE FROM accounts WHERE id = ?`, accountID)
	})

	if _, err := db.Exec(
		`INSERT INTO accounts (id, username, password_hash, gold, created_at) VALUES (?, ?, ?, 0, ?)`,
		accountID,
		"repair_"+suffix,
		strings.Repeat("a", 64),
		now,
	); err != nil {
		t.Fatalf("insert account: %v", err)
	}
	if _, err := db.Exec(
		`INSERT INTO players (id, account_id, nickname, faction, mail_code, state_json, created_at, updated_at)
		 VALUES (?, ?, ?, ?, '', ?, ?, ?)`,
		playerID,
		accountID,
		"Repair",
		"wei",
		`{"player":{"id":"`+playerID+`","nickname":"Repair","faction":"wei"}}`,
		now,
		now,
	); err != nil {
		t.Fatalf("insert player: %v", err)
	}

	report := game.NormalizeBattleReport(game.BattleReport{
		ID:               reportID,
		EventID:          eventID,
		PlayerID:         playerID,
		OwnerPlayerID:    playerID,
		PlayerName:       "Repair",
		PlayerFaction:    "wei",
		ViewType:         game.ReportViewAttack,
		SourceType:       game.ReportSourceNPCCity,
		BattleType:       "sweep",
		Title:            "NPC 扫荡",
		Summary:          "扫荡 2 城，成功 2 场，失败 0 场。",
		Type:             "attack",
		Result:           "attacker_victory",
		TargetID:         "npc_sweep",
		TargetName:       "NPC 扫荡",
		DefenderFaction:  "wei",
		DefenderRevealed: true,
		Rewards:          map[string]int{"wood": 100},
		CreatedAt:        now.Format(time.RFC3339),
		Detail: &game.BattleReportDetail{
			Title:   "NPC 扫荡",
			Summary: "扫荡 2 城，成功 2 场，失败 0 场。",
			PrimarySide: game.BattleReportSide{
				PlayerID:   playerID,
				PlayerName: "Repair",
				Role:       "attacker",
				Faction:    "wei",
			},
			Extra: map[string]interface{}{
				"sweep": map[string]interface{}{"requested": 2, "success": 2},
			},
		},
	})
	resultJSON, err := json.Marshal(game.SweepNpcResponse{BattleReport: report, Done: 2, Failed: 0})
	if err != nil {
		t.Fatalf("marshal sweep result: %v", err)
	}
	if _, err := db.Exec(
		`INSERT INTO npc_sweep_tasks (
			id, player_id, status, mode, npc_ids_json, general_ids_json, requested, done, failed, stopped,
			error_message, result_json, created_at, updated_at, started_at, completed_at
		) VALUES (?, ?, 'completed', 'attack', ?, ?, 2, 2, 0, 0, '', ?, ?, ?, ?, ?)`,
		taskID,
		playerID,
		`["npc_1","npc_2"]`,
		`[]`,
		string(resultJSON),
		now,
		now,
		now,
		now,
	); err != nil {
		t.Fatalf("insert sweep task: %v", err)
	}

	reports, total, err := repo.ListReportsByQuery(game.BattleReportQuery{PlayerID: playerID, Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("list reports: %v", err)
	}
	if total != 1 || len(reports) != 1 || reports[0].ID != reportID {
		t.Fatalf("expected repaired report in list, total=%d reports=%+v", total, reports)
	}
	restored, err := repo.GetReportForPlayer(playerID, reportID)
	if err != nil {
		t.Fatalf("get repaired report: %v", err)
	}
	if restored.Detail == nil || restored.Detail.Extra["sweep"] == nil {
		t.Fatalf("expected repaired report detail, got %+v", restored.Detail)
	}

	if err := repo.DeleteReport(playerID, reportID); err != nil {
		t.Fatalf("delete repaired report: %v", err)
	}
	if _, err := repo.GetReportForPlayer(playerID, reportID); err == nil {
		t.Fatal("expected physically deleted repaired report to be missing")
	}
	reports, total, err = repo.ListReportsByQuery(game.BattleReportQuery{PlayerID: playerID, Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("list reports after physical delete: %v", err)
	}
	if total != 0 || len(reports) != 0 {
		t.Fatalf("expected deleted sweep report not to be repaired again, total=%d reports=%+v", total, reports)
	}
	var taskReportID sql.NullString
	if err := db.QueryRow(
		`SELECT JSON_UNQUOTE(JSON_EXTRACT(result_json, '$.battleReport.id')) FROM npc_sweep_tasks WHERE id = ?`,
		taskID,
	).Scan(&taskReportID); err != nil {
		t.Fatalf("read sweep task report id after delete: %v", err)
	}
	if taskReportID.Valid && taskReportID.String != "" {
		t.Fatalf("expected sweep task battleReport snapshot to be removed, got %q", taskReportID.String)
	}
}

// TestMySQLBattleReportEventStateAndShare 验证标准战报主链路的 MySQL 持久化能力。
func TestMySQLBattleReportEventStateAndShare(t *testing.T) {
	repo, db := openReportTestRepository(t)
	now := time.Now().UTC()
	suffix := strings.NewReplacer(".", "_").Replace(now.Format("150405.000000"))
	eventID := "it_evt_" + suffix
	attackerReportID := "it_atk_" + suffix
	defenderReportID := "it_def_" + suffix
	attackerPlayerID := "it_pa_" + suffix
	defenderPlayerID := "it_pd_" + suffix
	t.Cleanup(func() {
		_, _ = db.Exec(`DELETE FROM battle_report_links WHERE report_id IN (?, ?)`, attackerReportID, defenderReportID)
		_, _ = db.Exec(`DELETE FROM battle_report_participants WHERE event_id = ?`, eventID)
		_, _ = db.Exec(`DELETE FROM battle_report_states WHERE report_id IN (?, ?)`, attackerReportID, defenderReportID)
		_, _ = db.Exec(`DELETE FROM battle_reports WHERE id IN (?, ?)`, attackerReportID, defenderReportID)
		_, _ = db.Exec(`DELETE FROM battle_events WHERE id = ?`, eventID)
	})

	attackerReport := game.NormalizeBattleReport(game.BattleReport{
		ID:              attackerReportID,
		EventID:         eventID,
		PlayerID:        attackerPlayerID,
		OwnerPlayerID:   attackerPlayerID,
		PlayerName:      "攻击城",
		PlayerFaction:   "wei",
		TargetID:        defenderPlayerID,
		TargetName:      "防守城",
		Type:            "attack",
		ViewType:        game.ReportViewAttack,
		SourceType:      game.ReportSourcePlayerCity,
		BattleType:      "attack",
		Result:          "attacker_victory",
		DispatchedUnits: map[string]int{"weiInfantry": 30},
		LostUnits:       map[string]int{"weiInfantry": 5},
		DefenderFaction: "shu",
		DefenderUnits: map[string]int{
			"shuInfantry": 20,
		},
		DefenderLostUnits: map[string]int{"shuInfantry": 20},
		DefenderRevealed:  true,
		Rewards:           map[string]int{"wood": 100},
		CreatedAt:         now.Format(time.RFC3339),
	})
	defenderReport := game.NormalizeBattleReport(game.BattleReport{
		ID:                defenderReportID,
		EventID:           eventID,
		PlayerID:          defenderPlayerID,
		OwnerPlayerID:     defenderPlayerID,
		PlayerName:        "防守城",
		PlayerFaction:     "shu",
		TargetID:          attackerPlayerID,
		TargetName:        "攻击城",
		Type:              "defense",
		ViewType:          game.ReportViewDefense,
		SourceType:        game.ReportSourcePlayerCity,
		BattleType:        "attack",
		Result:            "defender_victory",
		DispatchedUnits:   map[string]int{"shuInfantry": 20},
		LostUnits:         map[string]int{"shuInfantry": 20},
		DefenderFaction:   "wei",
		DefenderUnits:     map[string]int{"weiInfantry": 30},
		DefenderLostUnits: map[string]int{"weiInfantry": 5},
		DefenderRevealed:  true,
		CreatedAt:         now.Format(time.RFC3339),
	})

	if err := repo.SaveReports([]game.BattleReport{attackerReport, defenderReport}); err != nil {
		t.Fatalf("insert attacker report: %v", err)
	}

	var eventCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM battle_events WHERE id = ?`, eventID).Scan(&eventCount); err != nil {
		t.Fatalf("count battle_events: %v", err)
	}
	if eventCount != 1 {
		t.Fatalf("expected one battle event, got %d", eventCount)
	}
	var stateCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM battle_report_states WHERE report_id IN (?, ?)`, attackerReportID, defenderReportID).Scan(&stateCount); err != nil {
		t.Fatalf("count battle_report_states: %v", err)
	}
	if stateCount != 2 {
		t.Fatalf("expected two report states, got %d", stateCount)
	}
	var participantCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM battle_report_participants WHERE event_id = ?`, eventID).Scan(&participantCount); err != nil {
		t.Fatalf("count battle_report_participants: %v", err)
	}
	if participantCount != 4 {
		t.Fatalf("expected four participant snapshots, got %d", participantCount)
	}

	attackReports, attackTotal, err := repo.ListReportsByQuery(game.BattleReportQuery{
		PlayerID:   attackerPlayerID,
		ViewType:   game.ReportViewAttack,
		SourceType: game.ReportSourcePlayerCity,
		Page:       1,
		PageSize:   10,
	})
	if err != nil {
		t.Fatalf("list attack reports: %v", err)
	}
	if attackTotal != 1 || len(attackReports) != 1 || attackReports[0].ID != attackerReportID {
		t.Fatalf("unexpected attack reports total=%d reports=%+v", attackTotal, attackReports)
	}
	if attackReports[0].Detail != nil {
		t.Fatalf("expected report list to return summary without heavy detail, got %+v", attackReports[0].Detail)
	}
	attackDetail, err := repo.GetReportForPlayer(attackerPlayerID, attackerReportID)
	if err != nil {
		t.Fatalf("get attack report detail: %v", err)
	}
	if attackDetail.Detail == nil {
		t.Fatalf("expected detail endpoint to restore detail_json, got %+v", attackDetail)
	}

	eventReports, err := repo.ListReportsByEventForAdmin(eventID)
	if err != nil {
		t.Fatalf("list reports by event: %v", err)
	}
	if len(eventReports) != 2 {
		t.Fatalf("expected two event reports, got %+v", eventReports)
	}
	participants, err := repo.ListParticipantsByEventForAdmin(eventID)
	if err != nil {
		t.Fatalf("list participants by event: %v", err)
	}
	if len(participants) != 4 {
		t.Fatalf("expected four event participants, got %+v", participants)
	}
	if participants[0].ReportID == "" || len(participants[0].TroopsBefore) == 0 {
		t.Fatalf("expected participant troop snapshot, got %+v", participants[0])
	}

	link, err := repo.CreateBattleReportShareLink(attackerPlayerID, attackerReportID, "public", time.Time{})
	if err != nil {
		t.Fatalf("create share link: %v", err)
	}
	shared, err := repo.GetReportByShareToken(link.Token)
	if err != nil {
		t.Fatalf("get shared report: %v", err)
	}
	if shared.ID != attackerReportID || shared.Share == nil || shared.Share.Token != link.Token {
		t.Fatalf("unexpected shared report: %+v", shared)
	}

	if err := repo.MarkReportsReadByView(attackerPlayerID, game.ReportViewAttack); err != nil {
		t.Fatalf("mark reports read by view: %v", err)
	}
	var isRead bool
	if err := db.QueryRow(`SELECT is_read FROM battle_report_states WHERE report_id = ? AND player_id = ?`, attackerReportID, attackerPlayerID).Scan(&isRead); err != nil {
		t.Fatalf("read report state: %v", err)
	}
	if !isRead {
		t.Fatal("expected report state to be marked read")
	}
	if err := repo.DeleteReportsByView(attackerPlayerID, game.ReportViewAttack); err != nil {
		t.Fatalf("delete reports by view: %v", err)
	}
	var deletedReportRows int
	if err := db.QueryRow(`SELECT COUNT(*) FROM battle_reports WHERE id = ?`, attackerReportID).Scan(&deletedReportRows); err != nil {
		t.Fatalf("count deleted report: %v", err)
	}
	if deletedReportRows != 0 {
		t.Fatalf("expected report to be physically deleted, got %d rows", deletedReportRows)
	}
	var deletedStateRows int
	if err := db.QueryRow(`SELECT COUNT(*) FROM battle_report_states WHERE report_id = ? AND player_id = ?`, attackerReportID, attackerPlayerID).Scan(&deletedStateRows); err != nil {
		t.Fatalf("count deleted report state: %v", err)
	}
	if deletedStateRows != 0 {
		t.Fatalf("expected report state to be physically deleted, got %d rows", deletedStateRows)
	}
	var deletedLinkRows int
	if err := db.QueryRow(`SELECT COUNT(*) FROM battle_report_links WHERE report_id = ?`, attackerReportID).Scan(&deletedLinkRows); err != nil {
		t.Fatalf("count deleted report links: %v", err)
	}
	if deletedLinkRows != 0 {
		t.Fatalf("expected report share links to be physically deleted, got %d rows", deletedLinkRows)
	}
	remainingEventReports, err := repo.ListReportsByEventForAdmin(eventID)
	if err != nil {
		t.Fatalf("list event reports after delete: %v", err)
	}
	if len(remainingEventReports) != 1 || remainingEventReports[0].ID != defenderReportID {
		t.Fatalf("expected only defender report to remain, got %+v", remainingEventReports)
	}
}

// TestMySQLBattleReportStateIDUsesSafeLength 验证长 report/player ID 不会撑爆状态表主键。
func TestMySQLBattleReportStateIDUsesSafeLength(t *testing.T) {
	repo, db := openReportTestRepository(t)
	now := time.Now().UTC()
	suffix := strings.NewReplacer(".", "_").Replace(now.Format("150405.000000"))
	eventID := "it_evt_long_state_" + suffix
	reportID := "br_pvp_scout_" + strings.Repeat("abcdef", 7)
	playerID := "player_" + strings.Repeat("123456", 5)
	t.Cleanup(func() {
		_, _ = db.Exec(`DELETE FROM battle_report_participants WHERE event_id = ?`, eventID)
		_, _ = db.Exec(`DELETE FROM battle_report_states WHERE report_id = ?`, reportID)
		_, _ = db.Exec(`DELETE FROM battle_reports WHERE id = ?`, reportID)
		_, _ = db.Exec(`DELETE FROM battle_events WHERE id = ?`, eventID)
	})

	report := game.NormalizeBattleReport(game.BattleReport{
		ID:               reportID,
		EventID:          eventID,
		PlayerID:         playerID,
		OwnerPlayerID:    playerID,
		PlayerName:       "长 ID 侦查方",
		PlayerFaction:    "wei",
		TargetID:         "player_target_" + suffix,
		TargetName:       "目标",
		Type:             "scout",
		ViewType:         game.ReportViewAttack,
		SourceType:       game.ReportSourcePlayerCity,
		BattleType:       "scout",
		Result:           "attacker_victory",
		DispatchedUnits:  map[string]int{"weiScout": 3},
		DefenderFaction:  "shu",
		DefenderRevealed: true,
		CreatedAt:        now.Format(time.RFC3339),
	})
	if err := repo.SaveReports([]game.BattleReport{report}); err != nil {
		t.Fatalf("save long id report: %v", err)
	}
	var stateID string
	if err := db.QueryRow(`SELECT id FROM battle_report_states WHERE report_id = ? AND player_id = ?`, reportID, playerID).Scan(&stateID); err != nil {
		t.Fatalf("read battle report state: %v", err)
	}
	if len(stateID) > 64 {
		t.Fatalf("expected state id length <= 64, got %d: %s", len(stateID), stateID)
	}
}

// TestMySQLBattleReportVisibleCapSoftDeletesOldReports 验证同玩家同视角战报超过上限后会软删除旧普通战报并保护有效分享。
func TestMySQLBattleReportVisibleCapSoftDeletesOldReports(t *testing.T) {
	repo, db := openReportTestRepository(t)
	now := time.Now().UTC()
	suffix := strings.NewReplacer(".", "_").Replace(now.Format("150405.000000"))
	playerID := "it_cap_player_" + suffix
	sharedReportID := "it_cap_shared_" + suffix
	prefix := "it_cap_" + suffix + "_"
	t.Cleanup(func() {
		_, _ = db.Exec(`DELETE FROM battle_report_links WHERE report_id = ? OR report_id LIKE ?`, sharedReportID, prefix+"%")
		_, _ = db.Exec(`DELETE FROM battle_report_participants WHERE report_id = ? OR report_id LIKE ?`, sharedReportID, prefix+"%")
		_, _ = db.Exec(`DELETE FROM battle_report_states WHERE report_id = ? OR report_id LIKE ?`, sharedReportID, prefix+"%")
		_, _ = db.Exec(`DELETE FROM battle_reports WHERE id = ? OR id LIKE ?`, sharedReportID, prefix+"%")
		_, _ = db.Exec(`DELETE FROM battle_events WHERE id = ? OR id LIKE ?`, "event_"+sharedReportID, "event_"+prefix+"%")
	})

	shared := reportCapTestReport(sharedReportID, playerID, now.Add(-2*time.Hour))
	if err := repo.SaveReport(shared); err != nil {
		t.Fatalf("save shared report: %v", err)
	}
	if _, err := repo.CreateBattleReportShareLink(playerID, sharedReportID, "public", time.Time{}); err != nil {
		t.Fatalf("create share link: %v", err)
	}

	reports := make([]game.BattleReport, 0, battleReportVisibleCapPerView+2)
	for i := 0; i < battleReportVisibleCapPerView+2; i++ {
		reportID := fmt.Sprintf("%s%05d", prefix, i)
		reports = append(reports, reportCapTestReport(reportID, playerID, now.Add(time.Duration(i)*time.Second)))
	}
	if err := repo.SaveReports(reports); err != nil {
		t.Fatalf("save capped reports: %v", err)
	}

	var nonSharedVisible int
	if err := db.QueryRow(
		`SELECT COUNT(*)
		 FROM battle_reports
		 WHERE player_id = ? AND view_type = ? AND deleted_by_player = 0 AND id <> ?`,
		playerID,
		game.ReportViewAttack,
		sharedReportID,
	).Scan(&nonSharedVisible); err != nil {
		t.Fatalf("count visible non-shared reports: %v", err)
	}
	if nonSharedVisible != battleReportVisibleCapPerView {
		t.Fatalf("expected non-shared visible reports capped at %d, got %d", battleReportVisibleCapPerView, nonSharedVisible)
	}

	var sharedDeleted bool
	if err := db.QueryRow(`SELECT deleted_by_player FROM battle_reports WHERE id = ?`, sharedReportID).Scan(&sharedDeleted); err != nil {
		t.Fatalf("read shared report deleted flag: %v", err)
	}
	if sharedDeleted {
		t.Fatal("expected active shared report to be protected from visible cap")
	}

	oldestDeletedID := reports[0].ID
	var reportDeleted bool
	if err := db.QueryRow(`SELECT deleted_by_player FROM battle_reports WHERE id = ?`, oldestDeletedID).Scan(&reportDeleted); err != nil {
		t.Fatalf("read old capped report: %v", err)
	}
	if !reportDeleted {
		t.Fatalf("expected oldest non-shared report %s to be soft deleted", oldestDeletedID)
	}
	var stateDeleted bool
	if err := db.QueryRow(`SELECT is_deleted FROM battle_report_states WHERE report_id = ? AND player_id = ?`, oldestDeletedID, playerID).Scan(&stateDeleted); err != nil {
		t.Fatalf("read old capped report state: %v", err)
	}
	if !stateDeleted {
		t.Fatalf("expected oldest non-shared report state %s to be soft deleted", oldestDeletedID)
	}
}

// reportCapTestReport 构造战报上限测试用标准战报。
func reportCapTestReport(reportID string, playerID string, createdAt time.Time) game.BattleReport {
	return game.NormalizeBattleReport(game.BattleReport{
		ID:              reportID,
		EventID:         "event_" + reportID,
		PlayerID:        playerID,
		OwnerPlayerID:   playerID,
		PlayerName:      "上限测试城",
		PlayerFaction:   "wei",
		TargetID:        "npc_cap",
		TargetName:      "上限测试营地",
		Type:            "attack",
		ViewType:        game.ReportViewAttack,
		SourceType:      game.ReportSourceNPCCity,
		BattleType:      "attack",
		Result:          "attacker_victory",
		DispatchedUnits: map[string]int{"weiInfantry": 1},
		DefenderFaction: "shu",
		DefenderUnits:   map[string]int{"shuInfantry": 1},
		CreatedAt:       createdAt.UTC().Format(time.RFC3339),
	})
}
