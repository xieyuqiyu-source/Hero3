// 本文件验证战报系统重构后的 MySQL 事件、战报、状态和分享链路。
package storage

import (
	"context"
	"database/sql"
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
	var isDeleted bool
	if err := db.QueryRow(`SELECT is_deleted FROM battle_report_states WHERE report_id = ? AND player_id = ?`, attackerReportID, attackerPlayerID).Scan(&isDeleted); err != nil {
		t.Fatalf("read deleted report state: %v", err)
	}
	if !isDeleted {
		t.Fatal("expected report state to be marked deleted")
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
