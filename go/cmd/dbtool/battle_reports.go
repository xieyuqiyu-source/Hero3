// 本文件归口战报系统重构后的事件、战报和玩家状态校验修复工具。
package main

import (
	"context"
	"crypto/sha1"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"strings"
	"time"

	"hero3/internal/app/game"
	"hero3/internal/infrastructure/storage"
)

type battleEventVerifyResult struct {
	ReportsMissingEvent         int
	ReportsReferMissingEvent    int
	PvpBattlesMissingReports    int
	MarchesMissingReports       int
	ReinforcementsMissingReport int
}

type battleReportVerifyResult struct {
	Reports               int
	MissingStandardFields int
	MissingParticipants   int
	InvalidReportJSON     int
	InvalidDetailJSON     int
	DuplicateShareTokens  int
}

type battleReportStateVerifyResult struct {
	ReportsMissingState int
	OrphanStates        int
	DuplicateStates     int
}

type battleReportBackfillResult struct {
	Reports int
	Events  int
	States  int
}

// runVerifyBattleEvents 校验战报事件关联、PVP 战报引用和增援战报引用。
func runVerifyBattleEvents(args []string) error {
	flags := flag.NewFlagSet("verify-battle-events", flag.ContinueOnError)
	dsn := flags.String("dsn", "", "目标数据库 DSN，默认使用 HERO3_DATABASE_DSN")
	if err := flags.Parse(args); err != nil {
		return err
	}
	resolvedDSN, err := resolveBattleReportDSN(*dsn)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), longCommandTimeout)
	defer cancel()
	result, err := verifyBattleEvents(ctx, resolvedDSN)
	if err != nil {
		return err
	}
	if result.ReportsMissingEvent+result.ReportsReferMissingEvent+result.PvpBattlesMissingReports+result.MarchesMissingReports+result.ReinforcementsMissingReport > 0 {
		return fmt.Errorf("战斗事件校验失败：缺事件字段 %d，引用缺失事件 %d，PVP 战斗缺战报 %d，行军缺战报 %d，增援缺战报 %d",
			result.ReportsMissingEvent,
			result.ReportsReferMissingEvent,
			result.PvpBattlesMissingReports,
			result.MarchesMissingReports,
			result.ReinforcementsMissingReport,
		)
	}
	fmt.Println("战斗事件校验通过")
	return nil
}

// runVerifyBattleReports 校验战报标准字段、旧 JSON 和详情 JSON 可解析性。
func runVerifyBattleReports(args []string) error {
	flags := flag.NewFlagSet("verify-battle-reports", flag.ContinueOnError)
	dsn := flags.String("dsn", "", "目标数据库 DSN，默认使用 HERO3_DATABASE_DSN")
	if err := flags.Parse(args); err != nil {
		return err
	}
	resolvedDSN, err := resolveBattleReportDSN(*dsn)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), longCommandTimeout)
	defer cancel()
	result, err := verifyBattleReports(ctx, resolvedDSN)
	if err != nil {
		return err
	}
	if result.MissingStandardFields+result.MissingParticipants+result.InvalidReportJSON+result.InvalidDetailJSON+result.DuplicateShareTokens > 0 {
		return fmt.Errorf("战报标准结构校验失败：总数 %d，标准字段缺失 %d，参与方快照缺失 %d，旧 JSON 异常 %d，详情 JSON 异常 %d，分享 token 重复 %d",
			result.Reports,
			result.MissingStandardFields,
			result.MissingParticipants,
			result.InvalidReportJSON,
			result.InvalidDetailJSON,
			result.DuplicateShareTokens,
		)
	}
	fmt.Printf("战报标准结构校验通过：战报 %d\n", result.Reports)
	return nil
}

// runVerifyBattleReportStates 校验每份玩家战报都有对应状态，且状态没有孤儿或重复。
func runVerifyBattleReportStates(args []string) error {
	flags := flag.NewFlagSet("verify-battle-report-states", flag.ContinueOnError)
	dsn := flags.String("dsn", "", "目标数据库 DSN，默认使用 HERO3_DATABASE_DSN")
	if err := flags.Parse(args); err != nil {
		return err
	}
	resolvedDSN, err := resolveBattleReportDSN(*dsn)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), longCommandTimeout)
	defer cancel()
	result, err := verifyBattleReportStates(ctx, resolvedDSN)
	if err != nil {
		return err
	}
	if result.ReportsMissingState+result.OrphanStates+result.DuplicateStates > 0 {
		return fmt.Errorf("战报状态校验失败：缺状态 %d，孤儿状态 %d，重复状态 %d",
			result.ReportsMissingState,
			result.OrphanStates,
			result.DuplicateStates,
		)
	}
	fmt.Println("战报状态校验通过")
	return nil
}

// runRepairBattleReportState 为缺失的 battle_report_states 记录补齐默认状态。
func runRepairBattleReportState(args []string) error {
	flags := flag.NewFlagSet("repair-battle-report-state", flag.ContinueOnError)
	dsn := flags.String("dsn", "", "目标数据库 DSN，默认使用 HERO3_DATABASE_DSN")
	allowNonTest := flags.Bool("allow-non-test", false, "允许修复非 test_ 前缀数据库")
	if err := flags.Parse(args); err != nil {
		return err
	}
	resolvedDSN, databaseName, err := resolveRepairDSN(*dsn, *allowNonTest)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), longCommandTimeout)
	defer cancel()
	rows, err := repairBattleReportStates(ctx, resolvedDSN)
	if err != nil {
		return err
	}
	fmt.Printf("战报状态修复完成：数据库 %s，补齐 %d 行\n", databaseName, rows)
	return nil
}

// runRepairBattleEventLink 为缺少 event_id 或 battle_events 的旧战报补齐事件。
func runRepairBattleEventLink(args []string) error {
	flags := flag.NewFlagSet("repair-battle-event-link", flag.ContinueOnError)
	dsn := flags.String("dsn", "", "目标数据库 DSN，默认使用 HERO3_DATABASE_DSN")
	allowNonTest := flags.Bool("allow-non-test", false, "允许修复非 test_ 前缀数据库")
	if err := flags.Parse(args); err != nil {
		return err
	}
	resolvedDSN, databaseName, err := resolveRepairDSN(*dsn, *allowNonTest)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), longCommandTimeout)
	defer cancel()
	rows, err := repairBattleEventLinks(ctx, resolvedDSN)
	if err != nil {
		return err
	}
	fmt.Printf("战报事件关联修复完成：数据库 %s，补齐事件 %d 个\n", databaseName, rows)
	return nil
}

// runBackfillBattleReportV2 从旧 report_json 回填标准列、detail_json、事件和状态。
func runBackfillBattleReportV2(args []string) error {
	flags := flag.NewFlagSet("backfill-battle-report-v2", flag.ContinueOnError)
	dsn := flags.String("dsn", "", "目标数据库 DSN，默认使用 HERO3_DATABASE_DSN")
	allowNonTest := flags.Bool("allow-non-test", false, "允许回填非 test_ 前缀数据库")
	if err := flags.Parse(args); err != nil {
		return err
	}
	resolvedDSN, databaseName, err := resolveRepairDSN(*dsn, *allowNonTest)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), longCommandTimeout)
	defer cancel()
	result, err := backfillBattleReportV2(ctx, resolvedDSN)
	if err != nil {
		return err
	}
	fmt.Printf("战报 V2 回填完成：数据库 %s，战报 %d，事件 %d，状态 %d\n", databaseName, result.Reports, result.Events, result.States)
	return nil
}

// verifyBattleEvents 执行事件关联一致性校验。
func verifyBattleEvents(ctx context.Context, dsn string) (battleEventVerifyResult, error) {
	db, err := storage.OpenMySQL(ctx, dsn)
	if err != nil {
		return battleEventVerifyResult{}, err
	}
	defer db.Close()
	databaseName, err := storage.MySQLDatabaseName(dsn)
	if err != nil {
		return battleEventVerifyResult{}, err
	}
	result := battleEventVerifyResult{}
	if err := scanCount(ctx, db, &result.ReportsMissingEvent, `SELECT COUNT(*) FROM battle_reports WHERE event_id = ''`); err != nil {
		return result, err
	}
	if err := scanCount(ctx, db, &result.ReportsReferMissingEvent, `SELECT COUNT(*) FROM battle_reports br LEFT JOIN battle_events be ON be.id = br.event_id WHERE br.event_id <> '' AND be.id IS NULL`); err != nil {
		return result, err
	}
	if err := scanCount(ctx, db, &result.PvpBattlesMissingReports, `SELECT COUNT(*) FROM pvp_battles b LEFT JOIN battle_reports ar ON ar.id = b.attacker_report_id LEFT JOIN battle_reports dr ON dr.id = b.defender_report_id WHERE b.status = 'resolved' AND (b.attacker_report_id = '' OR b.defender_report_id = '' OR ar.id IS NULL OR dr.id IS NULL)`); err != nil {
		return result, err
	}
	if err := scanCount(ctx, db, &result.MarchesMissingReports, `SELECT COUNT(*) FROM pvp_marches m LEFT JOIN battle_reports ar ON ar.id = m.attacker_report_id LEFT JOIN battle_reports dr ON dr.id = m.defender_report_id WHERE m.status IN ('returning','completed') AND (m.attacker_report_id = '' OR m.defender_report_id = '' OR ar.id IS NULL OR dr.id IS NULL)`); err != nil {
		return result, err
	}
	hasReinforcements, err := dbtoolTableExists(ctx, db, databaseName, "player_reinforcements")
	if err != nil {
		return result, err
	}
	if hasReinforcements {
		if err := scanCount(ctx, db, &result.ReinforcementsMissingReport, `SELECT COUNT(*) FROM player_reinforcements r LEFT JOIN battle_reports br ON br.id = r.last_battle_report_id WHERE COALESCE(r.last_battle_report_id, '') <> '' AND br.id IS NULL`); err != nil {
			return result, err
		}
	}
	return result, nil
}

// verifyBattleReports 校验标准字段和 JSON 内容。
func verifyBattleReports(ctx context.Context, dsn string) (battleReportVerifyResult, error) {
	db, err := storage.OpenMySQL(ctx, dsn)
	if err != nil {
		return battleReportVerifyResult{}, err
	}
	defer db.Close()
	result := battleReportVerifyResult{}
	if err := scanCount(ctx, db, &result.Reports, `SELECT COUNT(*) FROM battle_reports`); err != nil {
		return result, err
	}
	if err := scanCount(ctx, db, &result.MissingStandardFields, `SELECT COUNT(*) FROM battle_reports WHERE owner_player_id = '' OR view_type = '' OR source_type = '' OR battle_type = '' OR title = '' OR detail_json IS NULL`); err != nil {
		return result, err
	}
	if err := scanCount(ctx, db, &result.MissingParticipants, `SELECT COUNT(*) FROM (
		SELECT br.id
		FROM battle_reports br
		LEFT JOIN battle_report_participants p ON p.report_id = br.id
		WHERE br.event_id <> '' AND br.detail_json IS NOT NULL
		GROUP BY br.id
		HAVING COUNT(p.id) = 0
	) missing_participants`); err != nil {
		return result, err
	}
	if err := scanCount(ctx, db, &result.InvalidReportJSON, `SELECT COUNT(*) FROM battle_reports WHERE JSON_VALID(report_json) = 0`); err != nil {
		return result, err
	}
	if err := scanCount(ctx, db, &result.InvalidDetailJSON, `SELECT COUNT(*) FROM battle_reports WHERE detail_json IS NOT NULL AND JSON_VALID(detail_json) = 0`); err != nil {
		return result, err
	}
	if err := scanCount(ctx, db, &result.DuplicateShareTokens, `SELECT COUNT(*) FROM (SELECT token FROM battle_report_links GROUP BY token HAVING COUNT(*) > 1) duplicated_tokens`); err != nil {
		return result, err
	}
	return result, nil
}

// verifyBattleReportStates 校验战报状态完整性。
func verifyBattleReportStates(ctx context.Context, dsn string) (battleReportStateVerifyResult, error) {
	db, err := storage.OpenMySQL(ctx, dsn)
	if err != nil {
		return battleReportStateVerifyResult{}, err
	}
	defer db.Close()
	result := battleReportStateVerifyResult{}
	if err := scanCount(ctx, db, &result.ReportsMissingState, `SELECT COUNT(*) FROM battle_reports br LEFT JOIN battle_report_states s ON s.report_id = br.id AND s.player_id = br.owner_player_id WHERE br.owner_player_id <> '' AND s.id IS NULL`); err != nil {
		return result, err
	}
	if err := scanCount(ctx, db, &result.OrphanStates, `SELECT COUNT(*) FROM battle_report_states s LEFT JOIN battle_reports br ON br.id = s.report_id WHERE br.id IS NULL`); err != nil {
		return result, err
	}
	if err := scanCount(ctx, db, &result.DuplicateStates, `SELECT COUNT(*) FROM (SELECT report_id, player_id FROM battle_report_states GROUP BY report_id, player_id HAVING COUNT(*) > 1) duplicated_states`); err != nil {
		return result, err
	}
	return result, nil
}

// repairBattleReportStates 补齐缺失的玩家战报状态。
func repairBattleReportStates(ctx context.Context, dsn string) (int64, error) {
	db, err := storage.OpenMySQL(ctx, dsn)
	if err != nil {
		return 0, err
	}
	defer db.Close()
	now := time.Now().UTC()
	rows, err := db.QueryContext(ctx,
		`SELECT br.id, br.owner_player_id, br.is_read, br.deleted_by_player
		 FROM battle_reports br
		 LEFT JOIN battle_report_states s ON s.report_id = br.id AND s.player_id = br.owner_player_id
		 WHERE br.owner_player_id <> '' AND s.id IS NULL`,
	)
	if err != nil {
		return 0, err
	}
	defer rows.Close()
	var inserted int64
	for rows.Next() {
		var reportID, playerID string
		var isRead, isDeleted bool
		if err := rows.Scan(&reportID, &playerID, &isRead, &isDeleted); err != nil {
			return inserted, err
		}
		result, err := db.ExecContext(ctx,
			`INSERT INTO battle_report_states (id, report_id, player_id, is_read, is_deleted, created_at, updated_at)
			 VALUES (?, ?, ?, ?, ?, ?, ?)`,
			battleReportStateID(reportID, playerID),
			reportID,
			playerID,
			isRead,
			isDeleted,
			now,
			now,
		)
		if err != nil {
			return inserted, err
		}
		affected, _ := result.RowsAffected()
		inserted += affected
	}
	return inserted, rows.Err()
}

// repairBattleEventLinks 补齐 event_id 和 battle_events。
func repairBattleEventLinks(ctx context.Context, dsn string) (int, error) {
	db, err := storage.OpenMySQL(ctx, dsn)
	if err != nil {
		return 0, err
	}
	defer db.Close()
	if _, err := db.ExecContext(ctx, `UPDATE battle_reports SET event_id = CONCAT('event_', id) WHERE event_id = ''`); err != nil {
		return 0, err
	}
	rows, err := db.QueryContext(ctx,
		`SELECT br.id, br.report_json, br.created_at
		 FROM battle_reports br
		 LEFT JOIN battle_events be ON be.id = br.event_id
		 WHERE be.id IS NULL`,
	)
	if err != nil {
		return 0, err
	}
	defer rows.Close()
	inserted := 0
	for rows.Next() {
		report, createdAt, err := scanReportSnapshot(rows)
		if err != nil {
			return inserted, err
		}
		normalized := game.NormalizeBattleReport(report)
		if err := insertBattleEvent(ctx, db, normalized, createdAt); err != nil {
			return inserted, err
		}
		inserted++
	}
	return inserted, rows.Err()
}

// backfillBattleReportV2 将旧 JSON 标准化后写回新列。
func backfillBattleReportV2(ctx context.Context, dsn string) (battleReportBackfillResult, error) {
	db, err := storage.OpenMySQL(ctx, dsn)
	if err != nil {
		return battleReportBackfillResult{}, err
	}
	defer db.Close()
	rows, err := db.QueryContext(ctx, `SELECT id, report_json, created_at FROM battle_reports`)
	if err != nil {
		return battleReportBackfillResult{}, err
	}
	defer rows.Close()
	result := battleReportBackfillResult{}
	for rows.Next() {
		report, createdAt, err := scanReportSnapshot(rows)
		if err != nil {
			return result, err
		}
		normalized := game.NormalizeBattleReport(report)
		if normalized.CreatedAt == "" {
			normalized.CreatedAt = createdAt.UTC().Format(time.RFC3339)
		}
		reportJSON, err := json.Marshal(normalized)
		if err != nil {
			return result, err
		}
		detailJSON, err := json.Marshal(normalized.Detail)
		if err != nil {
			return result, err
		}
		update, err := db.ExecContext(ctx,
			`UPDATE battle_reports
			 SET event_id = ?, owner_player_id = ?, view_type = ?, source_type = ?, battle_type = ?, result = ?,
			     title = ?, summary = ?, target_type = ?, target_id = ?, target_name = ?, detail_json = ?, report_json = ?
			 WHERE id = ?`,
			normalized.EventID,
			normalized.OwnerPlayerID,
			normalized.ViewType,
			normalized.SourceType,
			normalized.BattleType,
			normalized.Result,
			normalized.Title,
			normalized.Summary,
			normalized.SourceType,
			normalized.TargetID,
			normalized.TargetName,
			detailJSON,
			reportJSON,
			normalized.ID,
		)
		if err != nil {
			return result, err
		}
		affected, _ := update.RowsAffected()
		result.Reports += int(affected)
		if err := insertBattleEvent(ctx, db, normalized, createdAt); err != nil {
			return result, err
		}
		if err := insertBattleReportParticipants(ctx, db, normalized, createdAt); err != nil {
			return result, err
		}
		result.Events++
	}
	if err := rows.Err(); err != nil {
		return result, err
	}
	stateRows, err := repairBattleReportStates(ctx, dsn)
	if err != nil {
		return result, err
	}
	result.States = int(stateRows)
	return result, nil
}

// scanReportSnapshot 读取旧战报 JSON 并兼容数据库创建时间。
func scanReportSnapshot(rows interface {
	Scan(dest ...interface{}) error
}) (game.BattleReport, time.Time, error) {
	var id string
	var reportJSON []byte
	var createdAt time.Time
	if err := rows.Scan(&id, &reportJSON, &createdAt); err != nil {
		return game.BattleReport{}, time.Time{}, err
	}
	var report game.BattleReport
	if err := json.Unmarshal(reportJSON, &report); err != nil {
		return game.BattleReport{}, time.Time{}, fmt.Errorf("战报 %s JSON 解析失败: %w", id, err)
	}
	if report.ID == "" {
		report.ID = id
	}
	if report.CreatedAt == "" {
		report.CreatedAt = createdAt.UTC().Format(time.RFC3339)
	}
	return report, createdAt, nil
}

// insertBattleEvent 根据标准战报补齐 battle_events。
func insertBattleEvent(ctx context.Context, db *sql.DB, report game.BattleReport, occurredAt time.Time) error {
	if report.EventID == "" {
		return nil
	}
	now := time.Now().UTC()
	_, err := db.ExecContext(ctx,
		`INSERT IGNORE INTO battle_events (
			id, source_type, source_id, scene, battle_type, result,
			attacker_player_id, defender_player_id, attacker_name, defender_name,
			attacker_faction, defender_faction, summary_json, occurred_at, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, JSON_OBJECT('reportId', ?), ?, ?)`,
		report.EventID,
		report.SourceType,
		report.TargetID,
		report.ViewType,
		report.BattleType,
		report.Result,
		report.PlayerID,
		report.TargetID,
		report.PlayerName,
		report.TargetName,
		report.PlayerFaction,
		report.DefenderFaction,
		report.ID,
		occurredAt,
		now,
	)
	return err
}

// insertBattleReportParticipants 根据标准详情补齐参与方快照。
func insertBattleReportParticipants(ctx context.Context, db *sql.DB, report game.BattleReport, createdAt time.Time) error {
	if report.Detail == nil || report.EventID == "" {
		return nil
	}
	participants := []game.BattleReportParticipant{
		buildBattleReportParticipantSnapshot(report, "primary", report.Detail.PrimarySide, createdAt),
	}
	if report.Detail.SecondarySide != nil {
		participants = append(participants, buildBattleReportParticipantSnapshot(report, "secondary", *report.Detail.SecondarySide, createdAt))
	}
	for _, participant := range participants {
		troopsBeforeJSON, err := json.Marshal(participant.TroopsBefore)
		if err != nil {
			return err
		}
		troopsLostJSON, err := json.Marshal(participant.TroopsLost)
		if err != nil {
			return err
		}
		troopsSurvivedJSON, err := json.Marshal(participant.TroopsSurvived)
		if err != nil {
			return err
		}
		generalsJSON, err := json.Marshal(participant.Generals)
		if err != nil {
			return err
		}
		rewardsJSON, err := json.Marshal(participant.Rewards)
		if err != nil {
			return err
		}
		pointsDeltaJSON, err := json.Marshal(participant.PointsDelta)
		if err != nil {
			return err
		}
		if _, err := db.ExecContext(ctx,
			`INSERT IGNORE INTO battle_report_participants (
				id, event_id, report_id, player_id, role, faction, nickname, city_name,
				troops_before_json, troops_lost_json, troops_survived_json, generals_json,
				rewards_json, points_delta_json, created_at
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			participant.ID,
			participant.EventID,
			participant.ReportID,
			participant.PlayerID,
			participant.Role,
			participant.Faction,
			participant.Nickname,
			participant.CityName,
			troopsBeforeJSON,
			troopsLostJSON,
			troopsSurvivedJSON,
			generalsJSON,
			rewardsJSON,
			pointsDeltaJSON,
			createdAt.UTC(),
		); err != nil {
			return err
		}
	}
	return nil
}

// buildBattleReportParticipantSnapshot 从战报一侧构造参与方快照。
func buildBattleReportParticipantSnapshot(report game.BattleReport, side string, snapshot game.BattleReportSide, createdAt time.Time) game.BattleReportParticipant {
	troopsBefore := map[string]int{}
	troopsLost := map[string]int{}
	troopsSurvived := map[string]int{}
	for _, unit := range snapshot.Units {
		troopsBefore[unit.UnitType] = unit.AmountBefore
		troopsLost[unit.UnitType] = unit.Lost
		troopsSurvived[unit.UnitType] = unit.Survived
	}
	return game.BattleReportParticipant{
		ID:             battleReportParticipantID(report.ID, side),
		EventID:        report.EventID,
		ReportID:       report.ID,
		PlayerID:       snapshot.PlayerID,
		Role:           snapshot.Role,
		Faction:        snapshot.Faction,
		Nickname:       snapshot.PlayerName,
		CityName:       snapshot.CityName,
		TroopsBefore:   troopsBefore,
		TroopsLost:     troopsLost,
		TroopsSurvived: troopsSurvived,
		Generals:       snapshot.Generals,
		Rewards:        report.Detail.Rewards,
		CreatedAt:      createdAt.UTC().Format(time.RFC3339),
	}
}

// battleReportParticipantID 生成长度安全的参与方快照 ID。
func battleReportParticipantID(reportID string, side string) string {
	raw := "participant_" + reportID + "_" + side
	if len(raw) <= 64 {
		return raw
	}
	sum := sha1.Sum([]byte(raw))
	return "participant_" + hex.EncodeToString(sum[:])
}

// battleReportStateID 生成长度安全的玩家战报状态 ID。
func battleReportStateID(reportID string, playerID string) string {
	raw := "state_" + reportID + "_" + playerID
	if len(raw) <= 64 {
		return raw
	}
	sum := sha1.Sum([]byte(raw))
	return "state_" + hex.EncodeToString(sum[:])
}

// scanCount 执行单值 COUNT 查询。
func scanCount(ctx context.Context, db *sql.DB, target *int, query string, args ...interface{}) error {
	return db.QueryRowContext(ctx, query, args...).Scan(target)
}

// resolveBattleReportDSN 解析战报工具使用的 DSN。
func resolveBattleReportDSN(input string) (string, error) {
	if strings.TrimSpace(input) != "" {
		return input, nil
	}
	return configuredDSN()
}

// resolveRepairDSN 解析修复类命令 DSN，并默认限制 test_ 前缀库。
func resolveRepairDSN(input string, allowNonTest bool) (string, string, error) {
	dsn, err := resolveBattleReportDSN(input)
	if err != nil {
		return "", "", err
	}
	databaseName, err := storage.MySQLDatabaseName(dsn)
	if err != nil {
		return "", "", err
	}
	if !strings.HasPrefix(databaseName, "test_") && !allowNonTest {
		return "", "", fmt.Errorf("target database must use test_ prefix or pass --allow-non-test")
	}
	return dsn, databaseName, nil
}

// dbtoolTableExists 判断当前数据库是否存在指定表。
func dbtoolTableExists(ctx context.Context, db *sql.DB, databaseName string, tableName string) (bool, error) {
	var count int
	if err := db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = ? AND table_name = ?`,
		databaseName,
		tableName,
	).Scan(&count); err != nil {
		return false, err
	}
	return count > 0, nil
}
