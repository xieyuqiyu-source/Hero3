// 本文件归口 MySQL 战报仓储方法。
package storage

import (
	"crypto/rand"
	"crypto/sha1"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"hero3/internal/app/game"
)

const (
	battleReportVisibleCapPerView   = 1000
	battleReportCapDeleteBatchLimit = 500
	battleReportSweepRepairLimit    = 50
	battleReportHardDeleteBatchSize = 500
)

// SaveReport 保存战报，同时写入标准索引字段和玩家阅读状态。
func (r *MySQLRepository) SaveReport(report game.BattleReport) error {
	return r.SaveReports([]game.BattleReport{report})
}

// SaveReports 在同一事务内批量保存一组标准战报。
func (r *MySQLRepository) SaveReports(reports []game.BattleReport) error {
	return r.SaveReportBundle(game.BattleEvent{}, reports)
}

// SaveReportBundle 在同一事务内保存事件和它的全部玩家视角，防止管理端事件索引与战报脱节。
func (r *MySQLRepository) SaveReportBundle(event game.BattleEvent, reports []game.BattleReport) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if event.ID != "" {
		if err := upsertBattleEventTx(tx, event); err != nil {
			return err
		}
	}
	for _, report := range reports {
		if err := insertBattleReportTx(tx, report); err != nil && !isDuplicateEntry(err) {
			return err
		}
	}
	return tx.Commit()
}

// saveReportLegacy 保存战报，同时写入标准索引字段和玩家阅读状态。
func (r *MySQLRepository) saveReportLegacy(report game.BattleReport) error {
	report = game.NormalizeBattleReport(report)
	reportJSON, err := marshalBattleReportBodyJSON(report)
	if err != nil {
		return err
	}
	detailJSON, err := json.Marshal(report.Detail)
	if err != nil {
		return err
	}

	createdAt, _ := time.Parse(time.RFC3339, report.CreatedAt)
	if createdAt.IsZero() {
		createdAt = time.Now()
	}

	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err = tx.Exec(
		`INSERT INTO battle_reports (
			id, player_id, event_id, owner_player_id, view_type, source_type, battle_type, result,
			title, summary, target_type, target_id, target_name, detail_json,
			report_json, type, is_read, deleted_by_player, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		report.ID,
		report.PlayerID,
		report.EventID,
		report.OwnerPlayerID,
		report.ViewType,
		report.SourceType,
		report.BattleType,
		report.Result,
		report.Title,
		report.Summary,
		report.SourceType,
		report.TargetID,
		report.TargetName,
		detailJSON,
		reportJSON,
		report.Type,
		report.Read,
		false,
		createdAt.UTC(),
	); err != nil {
		return err
	}
	if err := insertBattleEventForReportTx(tx, report, createdAt.UTC()); err != nil {
		return err
	}
	if err := insertBattleReportParticipantsForReportTx(tx, report, createdAt.UTC()); err != nil {
		return err
	}
	if _, err = tx.Exec(
		`INSERT INTO battle_report_states (id, report_id, player_id, is_read, is_deleted, created_at, updated_at)
		 VALUES (?, ?, ?, ?, 0, ?, ?)
		 ON DUPLICATE KEY UPDATE is_read = VALUES(is_read), updated_at = VALUES(updated_at)`,
		battleReportStateID(report.ID, report.PlayerID),
		report.ID,
		report.PlayerID,
		report.Read,
		createdAt.UTC(),
		createdAt.UTC(),
	); err != nil {
		return err
	}
	return tx.Commit()
}

// battleReportStateID 生成长度安全的玩家战报状态主键。
func battleReportStateID(reportID string, playerID string) string {
	raw := "state_" + reportID + "_" + playerID
	if len(raw) <= 64 {
		return raw
	}
	sum := sha1.Sum([]byte(raw))
	return "state_" + hex.EncodeToString(sum[:])
}

// insertBattleEventForReportTx 按战报快照幂等写入战斗事件。
func insertBattleEventForReportTx(tx *sql.Tx, report game.BattleReport, occurredAt time.Time) error {
	if report.EventID == "" {
		return nil
	}
	event := game.BuildBattleEventFromReport(report)
	event.OccurredAt = occurredAt.UTC().Format(time.RFC3339)
	event.CreatedAt = event.OccurredAt
	summaryJSON, err := json.Marshal(event.Summary)
	if err != nil {
		return err
	}
	_, err = tx.Exec(
		`INSERT IGNORE INTO battle_events (
			id, source_type, source_id, scene, battle_type, result,
			attacker_player_id, defender_player_id, attacker_name, defender_name,
			attacker_faction, defender_faction, related_march_id, related_reinforcement_id,
			summary_json, occurred_at, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, '', '', ?, ?, ?)`,
		event.ID, event.SourceType, event.SourceID, event.Scene, event.BattleType, event.Result,
		event.AttackerPlayerID, event.DefenderPlayerID, event.AttackerName, event.DefenderName,
		event.AttackerFaction, event.DefenderFaction, summaryJSON,
		occurredAt,
		occurredAt,
	)
	return err
}

// upsertBattleEventTx 完整写入显式战斗事件，包括关联行军和管理端诊断快照。
func upsertBattleEventTx(tx *sql.Tx, event game.BattleEvent) error {
	summaryJSON, err := json.Marshal(event.Summary)
	if err != nil {
		return err
	}
	snapshotJSON, err := json.Marshal(event.Snapshot)
	if err != nil {
		return err
	}
	resultJSON, err := json.Marshal(event.ResultData)
	if err != nil {
		return err
	}
	occurredAt, _ := time.Parse(time.RFC3339, event.OccurredAt)
	if occurredAt.IsZero() {
		occurredAt = time.Now().UTC()
	}
	createdAt, _ := time.Parse(time.RFC3339, event.CreatedAt)
	if createdAt.IsZero() {
		createdAt = occurredAt
	}
	_, err = tx.Exec(
		`INSERT INTO battle_events (
			id, source_type, source_id, scene, battle_type, result,
			attacker_player_id, defender_player_id, attacker_name, defender_name,
			attacker_faction, defender_faction, related_march_id, related_reinforcement_id,
			summary_json, snapshot_json, result_json, occurred_at, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE source_type=VALUES(source_type), source_id=VALUES(source_id), scene=VALUES(scene),
			battle_type=VALUES(battle_type), result=VALUES(result), attacker_player_id=VALUES(attacker_player_id),
			defender_player_id=VALUES(defender_player_id), attacker_name=VALUES(attacker_name), defender_name=VALUES(defender_name),
			attacker_faction=VALUES(attacker_faction), defender_faction=VALUES(defender_faction),
			related_march_id=VALUES(related_march_id), related_reinforcement_id=VALUES(related_reinforcement_id),
			summary_json=VALUES(summary_json), snapshot_json=VALUES(snapshot_json), result_json=VALUES(result_json),
			occurred_at=VALUES(occurred_at)`,
		event.ID, event.SourceType, event.SourceID, event.Scene, event.BattleType, event.Result,
		event.AttackerPlayerID, event.DefenderPlayerID, event.AttackerName, event.DefenderName,
		event.AttackerFaction, event.DefenderFaction, event.RelatedMarchID, event.RelatedReinforcementID,
		summaryJSON, snapshotJSON, resultJSON, occurredAt.UTC(), createdAt.UTC(),
	)
	return err
}

// insertBattleReportParticipantsForReportTx 按标准详情写入事件参与方快照。
func insertBattleReportParticipantsForReportTx(tx *sql.Tx, report game.BattleReport, createdAt time.Time) error {
	if report.Detail == nil || report.EventID == "" {
		return nil
	}
	participants := buildBattleReportParticipants(report, createdAt)
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
		if _, err := tx.Exec(
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
			createdAt,
		); err != nil {
			return err
		}
	}
	return nil
}

// buildBattleReportParticipants 从标准详情生成参与方快照。
func buildBattleReportParticipants(report game.BattleReport, createdAt time.Time) []game.BattleReportParticipant {
	if report.Detail == nil {
		return nil
	}
	participants := make([]game.BattleReportParticipant, 0, 2)
	participants = append(participants, buildBattleReportParticipant(report, "primary", report.Detail.PrimarySide, createdAt))
	if report.Detail.SecondarySide != nil {
		participants = append(participants, buildBattleReportParticipant(report, "secondary", *report.Detail.SecondarySide, createdAt))
	}
	return participants
}

// buildBattleReportParticipant 从单侧详情生成参与方快照。
func buildBattleReportParticipant(report game.BattleReport, side string, snapshot game.BattleReportSide, createdAt time.Time) game.BattleReportParticipant {
	pointsDelta := map[string]int{}
	if snapshot.PlayerID == report.PlayerID {
		pointsDelta = report.PvpPointsDelta
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
		TroopsBefore:   reportParticipantUnitMap(snapshot.Units, "before"),
		TroopsLost:     reportParticipantUnitMap(snapshot.Units, "lost"),
		TroopsSurvived: reportParticipantUnitMap(snapshot.Units, "survived"),
		Generals:       snapshot.Generals,
		Rewards:        report.Detail.Rewards,
		PointsDelta:    pointsDelta,
		CreatedAt:      createdAt.UTC().Format(time.RFC3339),
	}
}

// battleReportParticipantID 生成长度稳定的参与方主键。
func battleReportParticipantID(reportID string, side string) string {
	raw := "participant_" + reportID + "_" + side
	if len(raw) <= 64 {
		return raw
	}
	sum := sha1.Sum([]byte(raw))
	return "participant_" + hex.EncodeToString(sum[:])
}

// reportParticipantUnitMap 将标准兵种数组转换为参与方快照 map。
func reportParticipantUnitMap(units []game.BattleReportUnit, field string) map[string]int {
	result := make(map[string]int, len(units))
	for _, unit := range units {
		switch field {
		case "lost":
			result[unit.UnitType] = unit.Lost
		case "survived":
			result[unit.UnitType] = unit.Survived
		default:
			result[unit.UnitType] = unit.AmountBefore
		}
	}
	return result
}

// marshalBattleReportBodyJSON 保存战报主体，详情只写 detail_json，避免大型扫荡详情重复写入。
func marshalBattleReportBodyJSON(report game.BattleReport) ([]byte, error) {
	report.Detail = nil
	return json.Marshal(report)
}

// scanBattleReportJSON 合并主体 JSON 和详情 JSON，兼容旧 report_json 已包含 detail 的历史数据。
func scanBattleReportJSON(reportJSON []byte, detailJSON []byte, isRead bool) (game.BattleReport, error) {
	var report game.BattleReport
	if err := json.Unmarshal(reportJSON, &report); err != nil {
		return game.BattleReport{}, err
	}
	if report.Detail == nil && len(detailJSON) > 0 && string(detailJSON) != "null" {
		var detail game.BattleReportDetail
		if err := json.Unmarshal(detailJSON, &detail); err != nil {
			return game.BattleReport{}, err
		}
		report.Detail = &detail
	}
	report.Read = isRead
	return game.NormalizeBattleReport(report), nil
}

const battleReportSummaryColumns = `id, player_id, event_id, owner_player_id, view_type, source_type, battle_type, result,
	title, summary, target_id, target_name, type, is_read, deleted_by_player, created_at,
	COALESCE(JSON_UNQUOTE(JSON_EXTRACT(report_json, '$.winnerSide')), ''),
	COALESCE(JSON_UNQUOTE(JSON_EXTRACT(report_json, '$.ownerSide')), ''),
	COALESCE(JSON_UNQUOTE(JSON_EXTRACT(report_json, '$.ownerOutcome')), ''),
	COALESCE(CAST(JSON_EXTRACT(report_json, '$.rewards') AS CHAR), '{}'),
	COALESCE(CAST(JSON_EXTRACT(report_json, '$.lostUnits') AS CHAR), '{}')`

// scanBattleReportSummary 从索引列构造军情列表摘要，避免列表读取完整 report_json。
func scanBattleReportSummary(scanner interface{ Scan(dest ...any) error }) (game.BattleReport, error) {
	var report game.BattleReport
	var createdAt time.Time
	var rewardsJSON, lostUnitsJSON []byte
	if err := scanner.Scan(
		&report.ID,
		&report.PlayerID,
		&report.EventID,
		&report.OwnerPlayerID,
		&report.ViewType,
		&report.SourceType,
		&report.BattleType,
		&report.Result,
		&report.Title,
		&report.Summary,
		&report.TargetID,
		&report.TargetName,
		&report.Type,
		&report.Read,
		&report.DeletedByPlayer,
		&createdAt,
		&report.WinnerSide,
		&report.OwnerSide,
		&report.OwnerOutcome,
		&rewardsJSON,
		&lostUnitsJSON,
	); err != nil {
		return game.BattleReport{}, err
	}
	report.CreatedAt = createdAt.UTC().Format(time.RFC3339)
	if err := json.Unmarshal(rewardsJSON, &report.Rewards); err != nil {
		return game.BattleReport{}, err
	}
	if err := json.Unmarshal(lostUnitsJSON, &report.LostUnits); err != nil {
		return game.BattleReport{}, err
	}
	return report, nil
}

// GetReportByID 供管理端读取原始战报；玩家端必须走归属校验或分享令牌入口。
func (r *MySQLRepository) GetReportByID(reportID string) (game.BattleReport, error) {
	var reportJSON []byte
	var detailJSON []byte
	err := r.db.QueryRow(`SELECT report_json, COALESCE(CAST(detail_json AS CHAR), 'null') FROM battle_reports WHERE id = ? LIMIT 1`, reportID).Scan(&reportJSON, &detailJSON)
	if err != nil {
		return game.BattleReport{}, errors.New("report not found")
	}
	return scanBattleReportJSON(reportJSON, detailJSON, false)
}

// GetReportForPlayer 获取玩家自己的未删除战报。
func (r *MySQLRepository) GetReportForPlayer(playerID string, reportID string) (game.BattleReport, error) {
	var reportJSON []byte
	var detailJSON []byte
	var isRead bool
	err := r.db.QueryRow(
		`SELECT report_json, COALESCE(CAST(detail_json AS CHAR), 'null'), is_read FROM battle_reports
		 WHERE id = ? AND player_id = ? AND deleted_by_player = 0 LIMIT 1`,
		reportID, playerID,
	).Scan(&reportJSON, &detailJSON, &isRead)
	if err != nil {
		return game.BattleReport{}, errors.New("report not found")
	}
	return scanBattleReportJSON(reportJSON, detailJSON, isRead)
}

// GetReportByShareToken 通过分享 token 获取战报。
func (r *MySQLRepository) GetReportByShareToken(token string) (game.BattleReport, error) {
	var reportJSON []byte
	var detailJSON []byte
	var tokenValue, visibility string
	var expiresAt sql.NullTime
	err := r.db.QueryRow(
		`SELECT br.report_json, COALESCE(CAST(br.detail_json AS CHAR), 'null'), l.token, l.visibility, l.expires_at
		 FROM battle_report_links l
		 JOIN battle_reports br ON br.id = l.report_id
		 WHERE l.token = ? AND (l.expires_at IS NULL OR l.expires_at > ?)
		 LIMIT 1`,
		token, time.Now().UTC(),
	).Scan(&reportJSON, &detailJSON, &tokenValue, &visibility, &expiresAt)
	if err != nil {
		return game.BattleReport{}, errors.New("report not found")
	}
	report, err := scanBattleReportJSON(reportJSON, detailJSON, false)
	if err != nil {
		return game.BattleReport{}, err
	}
	report.Share = &game.BattleReportShare{Token: tokenValue, Visibility: visibility, ExpiresAt: formatNullTime(expiresAt)}
	return game.NormalizeBattleReport(report), nil
}

// ListReports 按旧接口分页获取玩家军情战报。
func (r *MySQLRepository) ListReports(playerID string, limit int, offset int) ([]game.BattleReport, int, error) {
	threeDaysAgo := time.Now().Add(-3 * 24 * time.Hour).UTC()
	if repaired, err := r.repairMissingSweepReportsForPlayer(playerID, threeDaysAgo); err != nil {
		slog.Warn("missing sweep report repair failed", "playerId", playerID, "error", err)
	} else if repaired > 0 {
		slog.Info("missing sweep reports repaired", "playerId", playerID, "count", repaired)
	}
	if limit <= 0 {
		limit = 10
	}
	if offset < 0 {
		offset = 0
	}

	var total int
	if err := r.db.QueryRow(
		`SELECT COUNT(*) FROM battle_reports
		 WHERE player_id = ? AND deleted_by_player = 0 AND created_at > ?`,
		playerID, threeDaysAgo,
	).Scan(&total); err != nil {
		return nil, 0, err
	}

	rows, err := r.db.Query(
		`SELECT `+battleReportSummaryColumns+` FROM battle_reports
		 WHERE player_id = ? AND deleted_by_player = 0 AND created_at > ?
		 ORDER BY created_at DESC LIMIT ? OFFSET ?`,
		playerID, threeDaysAgo, limit, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var reports []game.BattleReport
	for rows.Next() {
		report, err := scanBattleReportSummary(rows)
		if err != nil {
			continue
		}
		reports = append(reports, report)
	}
	return reports, total, rows.Err()
}

// ListReportsByQuery 按标准筛选条件分页查询玩家战报。
func (r *MySQLRepository) ListReportsByQuery(query game.BattleReportQuery) ([]game.BattleReport, int, error) {
	if query.PageSize <= 0 {
		query.PageSize = 10
	}
	if query.PageSize > 50 {
		query.PageSize = 50
	}
	if query.Page <= 0 {
		query.Page = 1
	}
	offset := (query.Page - 1) * query.PageSize
	timeFrom := query.TimeFrom
	if timeFrom.IsZero() {
		timeFrom = time.Now().Add(-3 * 24 * time.Hour).UTC()
	}
	if repaired, err := r.repairMissingSweepReportsForPlayer(query.PlayerID, timeFrom); err != nil {
		slog.Warn("missing sweep report repair failed", "playerId", query.PlayerID, "error", err)
	} else if repaired > 0 {
		slog.Info("missing sweep reports repaired", "playerId", query.PlayerID, "count", repaired)
	}

	where, args := buildReportWhere(query, timeFrom)
	var total int
	if err := r.db.QueryRow(`SELECT COUNT(*) FROM battle_reports `+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := r.db.Query(
		`SELECT `+battleReportSummaryColumns+` FROM battle_reports `+where+` ORDER BY created_at DESC LIMIT ? OFFSET ?`,
		append(args, query.PageSize, offset)...,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var reports []game.BattleReport
	for rows.Next() {
		report, err := scanBattleReportSummary(rows)
		if err != nil {
			continue
		}
		reports = append(reports, report)
	}
	return reports, total, rows.Err()
}

type sweepTaskReportRepairPayload struct {
	BattleReport game.BattleReport `json:"battleReport"`
}

// repairMissingSweepReportsForPlayer 从扫荡任务结果快照补写缺失战报，修复历史保存失败造成的军情空洞。
func (r *MySQLRepository) repairMissingSweepReportsForPlayer(playerID string, cutoff time.Time) (int, error) {
	playerID = strings.TrimSpace(playerID)
	if playerID == "" {
		return 0, nil
	}
	if cutoff.IsZero() {
		cutoff = time.Now().Add(-3 * 24 * time.Hour).UTC()
	}
	rows, err := r.db.Query(
		`SELECT t.id, t.result_json
		 FROM npc_sweep_tasks t
		 LEFT JOIN battle_reports br
		   ON br.id = JSON_UNQUOTE(JSON_EXTRACT(t.result_json, '$.battleReport.id'))
		 WHERE t.player_id = ?
		   AND t.created_at > ?
		   AND t.status = 'completed'
		   AND COALESCE(JSON_UNQUOTE(JSON_EXTRACT(t.result_json, '$.battleReport.id')), '') <> ''
		   AND br.id IS NULL
		 ORDER BY t.created_at DESC
		 LIMIT ?`,
		playerID,
		cutoff.UTC(),
		battleReportSweepRepairLimit,
	)
	if err != nil {
		return 0, err
	}

	type repairCandidate struct {
		taskID     string
		resultJSON []byte
	}
	candidates := []repairCandidate{}
	for rows.Next() {
		var candidate repairCandidate
		if err := rows.Scan(&candidate.taskID, &candidate.resultJSON); err != nil {
			_ = rows.Close()
			return 0, err
		}
		candidates = append(candidates, candidate)
	}
	if err := rows.Close(); err != nil {
		return 0, err
	}
	if err := rows.Err(); err != nil {
		return 0, err
	}

	repaired := 0
	for _, candidate := range candidates {
		var payload sweepTaskReportRepairPayload
		if err := json.Unmarshal(candidate.resultJSON, &payload); err != nil {
			return repaired, fmt.Errorf("decode sweep task %s result: %w", candidate.taskID, err)
		}
		report := game.NormalizeBattleReport(payload.BattleReport)
		if strings.TrimSpace(report.ID) == "" || strings.TrimSpace(report.PlayerID) == "" {
			continue
		}
		if report.PlayerID != playerID {
			slog.Warn("skip sweep report repair for mismatched player", "taskId", candidate.taskID, "playerId", playerID, "reportPlayerId", report.PlayerID, "reportId", report.ID)
			continue
		}
		if err := r.SaveReport(report); err != nil {
			if _, lookupErr := r.GetReportForPlayer(playerID, report.ID); lookupErr == nil {
				continue
			}
			return repaired, fmt.Errorf("save repaired sweep report %s from task %s: %w", report.ID, candidate.taskID, err)
		}
		repaired++
	}
	return repaired, nil
}

// buildReportWhere 构建战报查询条件。
func buildReportWhere(query game.BattleReportQuery, timeFrom time.Time) (string, []any) {
	conditions := []string{"player_id = ?", "created_at > ?"}
	args := []any{query.PlayerID, timeFrom.UTC()}
	if !query.IncludeDeleted {
		conditions = append(conditions, "deleted_by_player = 0")
	}
	if query.ViewType != "" {
		conditions = append(conditions, "view_type = ?")
		args = append(args, query.ViewType)
	}
	if query.SourceType != "" {
		conditions = append(conditions, "source_type = ?")
		args = append(args, query.SourceType)
	}
	if query.BattleType != "" {
		conditions = append(conditions, "battle_type = ?")
		args = append(args, query.BattleType)
	}
	if query.Result != "" {
		conditions = append(conditions, "result = ?")
		args = append(args, query.Result)
	}
	if query.OwnerOutcome != "" {
		conditions = append(conditions, "(JSON_UNQUOTE(JSON_EXTRACT(detail_json, '$.ownerOutcome')) = ? OR JSON_UNQUOTE(JSON_EXTRACT(report_json, '$.ownerOutcome')) = ?)")
		args = append(args, query.OwnerOutcome, query.OwnerOutcome)
	}
	if !query.TimeTo.IsZero() {
		conditions = append(conditions, "created_at <= ?")
		args = append(args, query.TimeTo.UTC())
	}
	return "WHERE " + strings.Join(conditions, " AND "), args
}

// ListAllReports 列出玩家所有战报，供维护和兼容逻辑使用。
func (r *MySQLRepository) ListAllReports(playerID string) ([]game.BattleReport, error) {
	rows, err := r.db.Query(
		`SELECT report_json, COALESCE(CAST(detail_json AS CHAR), 'null') FROM battle_reports
		 WHERE player_id = ?
		 ORDER BY created_at DESC`,
		playerID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var reports []game.BattleReport
	for rows.Next() {
		var reportJSON []byte
		var detailJSON []byte
		if err := rows.Scan(&reportJSON, &detailJSON); err != nil {
			return nil, err
		}
		report, err := scanBattleReportJSON(reportJSON, detailJSON, false)
		if err != nil {
			continue
		}
		reports = append(reports, report)
	}
	return reports, rows.Err()
}

// MarkReportsRead 标记玩家所有战报已读。
func (r *MySQLRepository) MarkReportsRead(playerID string) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err = tx.Exec(
		`UPDATE battle_reports SET is_read = 1 WHERE player_id = ? AND is_read = 0 AND deleted_by_player = 0`,
		playerID,
	); err != nil {
		return err
	}
	if _, err = tx.Exec(
		`UPDATE battle_report_states SET is_read = 1, read_at = ?, updated_at = ?
		 WHERE player_id = ? AND is_read = 0 AND is_deleted = 0`,
		time.Now().UTC(), time.Now().UTC(), playerID,
	); err != nil {
		return err
	}
	return tx.Commit()
}

// MarkReportsReadByView 标记指定视角 Tab 的战报已读。
func (r *MySQLRepository) MarkReportsReadByView(playerID string, viewType string) error {
	if strings.TrimSpace(viewType) == "" {
		return r.MarkReportsRead(playerID)
	}
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	now := time.Now().UTC()
	if _, err := tx.Exec(
		`UPDATE battle_reports SET is_read = 1
		 WHERE player_id = ? AND view_type = ? AND is_read = 0 AND deleted_by_player = 0`,
		playerID, viewType,
	); err != nil {
		return err
	}
	if _, err := tx.Exec(
		`UPDATE battle_report_states s
		 JOIN battle_reports br ON br.id = s.report_id
		 SET s.is_read = 1, s.read_at = ?, s.updated_at = ?
		 WHERE s.player_id = ? AND br.view_type = ? AND s.is_read = 0 AND s.is_deleted = 0`,
		now, now, playerID, viewType,
	); err != nil {
		return err
	}
	return tx.Commit()
}

// MarkSingleReportRead 标记单条战报已读。
func (r *MySQLRepository) MarkSingleReportRead(playerID string, reportID string) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	now := time.Now().UTC()
	if _, err := tx.Exec(
		`UPDATE battle_reports
		 SET is_read = 1
		 WHERE id = ? AND player_id = ? AND is_read = 0 AND deleted_by_player = 0`,
		reportID, playerID,
	); err != nil {
		return err
	}
	if _, err := tx.Exec(
		`UPDATE battle_report_states
		 SET is_read = 1, read_at = ?, updated_at = ?
		 WHERE report_id = ? AND player_id = ? AND is_read = 0 AND is_deleted = 0`,
		now, now, reportID, playerID,
	); err != nil {
		return err
	}
	return tx.Commit()
}

// DeleteReport 物理删除单条玩家战报及其关联数据。
func (r *MySQLRepository) DeleteReport(playerID string, reportID string) error {
	rows, err := r.selectReportDeleteTargets(`WHERE player_id = ? AND id = ?`, playerID, reportID)
	if err != nil {
		return err
	}
	return r.deleteBattleReportsHard(rows)
}

// DeleteReportsByView 物理删除指定视角 Tab 下的战报。
func (r *MySQLRepository) DeleteReportsByView(playerID string, viewType string) error {
	return r.DeleteReportsByFilter(game.BattleReportDeleteFilter{PlayerID: playerID, ViewType: viewType})
}

// DeleteReportsByFilter 按列表相同的视角和战斗类型条件物理删除战报。
func (r *MySQLRepository) DeleteReportsByFilter(filter game.BattleReportDeleteFilter) error {
	where := `WHERE player_id = ?`
	args := []any{strings.TrimSpace(filter.PlayerID)}
	if viewType := strings.TrimSpace(filter.ViewType); viewType != "" {
		where += ` AND view_type = ?`
		args = append(args, viewType)
	}
	if battleType := strings.TrimSpace(filter.BattleType); battleType != "" {
		where += ` AND battle_type = ?`
		args = append(args, battleType)
	}
	rows, err := r.selectReportDeleteTargets(where, args...)
	if err != nil {
		return err
	}
	return r.deleteBattleReportsHard(rows)
}

// DeleteAllReports 物理删除玩家全部战报。
func (r *MySQLRepository) DeleteAllReports(playerID string) error {
	rows, err := r.selectReportDeleteTargets(`WHERE player_id = ?`, playerID)
	if err != nil {
		return err
	}
	return r.deleteBattleReportsHard(rows)
}

type battleReportDeleteTarget struct {
	ReportID string
	EventID  string
}

// selectReportDeleteTargets 读取待物理删除的战报主键和事件 ID。
func (r *MySQLRepository) selectReportDeleteTargets(where string, args ...any) ([]battleReportDeleteTarget, error) {
	rows, err := r.db.Query(`SELECT id, event_id FROM battle_reports `+where, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	targets := []battleReportDeleteTarget{}
	for rows.Next() {
		var target battleReportDeleteTarget
		if err := rows.Scan(&target.ReportID, &target.EventID); err != nil {
			return nil, err
		}
		if strings.TrimSpace(target.ReportID) != "" {
			targets = append(targets, target)
		}
	}
	return targets, rows.Err()
}

// deleteBattleReportsHard 分批物理删除战报及附属数据，并清掉扫荡任务里的战报快照。
func (r *MySQLRepository) deleteBattleReportsHard(targets []battleReportDeleteTarget) error {
	for start := 0; start < len(targets); start += battleReportHardDeleteBatchSize {
		end := start + battleReportHardDeleteBatchSize
		if end > len(targets) {
			end = len(targets)
		}
		if err := r.deleteBattleReportsHardBatch(targets[start:end]); err != nil {
			return err
		}
	}
	return nil
}

// deleteBattleReportsHardBatch 在短事务内删除一批战报，避免大事务拖慢线上库。
func (r *MySQLRepository) deleteBattleReportsHardBatch(targets []battleReportDeleteTarget) error {
	if len(targets) == 0 {
		return nil
	}
	reportIDs := make([]string, 0, len(targets))
	eventIDs := make([]string, 0, len(targets))
	seenEvents := map[string]bool{}
	for _, target := range targets {
		reportIDs = append(reportIDs, target.ReportID)
		if strings.TrimSpace(target.EventID) != "" && !seenEvents[target.EventID] {
			eventIDs = append(eventIDs, target.EventID)
			seenEvents[target.EventID] = true
		}
	}

	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if err := removeSweepTaskBattleReportsTx(tx, reportIDs); err != nil {
		return err
	}
	if err := execReportIDDeleteTx(tx, `DELETE FROM battle_report_links WHERE report_id IN (%s)`, reportIDs); err != nil {
		return err
	}
	if err := execReportIDDeleteTx(tx, `DELETE FROM battle_report_states WHERE report_id IN (%s)`, reportIDs); err != nil {
		return err
	}
	if err := execReportIDDeleteTx(tx, `DELETE FROM battle_report_participants WHERE report_id IN (%s)`, reportIDs); err != nil {
		return err
	}
	if err := execReportIDDeleteTx(tx, `DELETE FROM battle_reports WHERE id IN (%s)`, reportIDs); err != nil {
		return err
	}
	if len(eventIDs) > 0 {
		if err := execReportIDDeleteTx(tx,
			`DELETE FROM battle_events
			 WHERE id IN (%s)
			   AND NOT EXISTS (SELECT 1 FROM battle_reports br WHERE br.event_id = battle_events.id)
			   AND NOT EXISTS (SELECT 1 FROM battle_report_participants p WHERE p.event_id = battle_events.id)`,
			eventIDs,
		); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// removeSweepTaskBattleReportsTx 移除扫荡任务结果中的战报快照，防止物理删除后被自动补写恢复。
func removeSweepTaskBattleReportsTx(tx *sql.Tx, reportIDs []string) error {
	return execReportIDDeleteTx(
		tx,
		`UPDATE npc_sweep_tasks
		 SET result_json = JSON_REMOVE(result_json, '$.battleReport')
		 WHERE JSON_UNQUOTE(JSON_EXTRACT(result_json, '$.battleReport.id')) IN (%s)`,
		reportIDs,
	)
}

// execReportIDDeleteTx 执行基于字符串主键 IN 条件的删除或更新语句。
func execReportIDDeleteTx(tx *sql.Tx, statement string, values []string) error {
	if len(values) == 0 {
		return nil
	}
	placeholders := strings.TrimRight(strings.Repeat("?,", len(values)), ",")
	args := make([]any, 0, len(values))
	for _, value := range values {
		args = append(args, value)
	}
	_, err := tx.Exec(fmt.Sprintf(statement, placeholders), args...)
	return err
}

// CreateBattleReportShareLink 创建或更新战报分享 token。
func (r *MySQLRepository) CreateBattleReportShareLink(playerID string, reportID string, visibility string, expiresAt time.Time) (game.BattleReportShareLink, error) {
	if visibility == "" {
		visibility = "public"
	}
	if _, err := r.GetReportForPlayer(playerID, reportID); err != nil {
		return game.BattleReportShareLink{}, err
	}
	now := time.Now().UTC()
	link := game.BattleReportShareLink{
		ID:         "share_" + reportRandomID(12),
		ReportID:   reportID,
		Token:      "br_" + reportRandomID(24),
		Visibility: visibility,
		CreatedAt:  now.Format(time.RFC3339),
	}
	var expiresArg any
	if !expiresAt.IsZero() {
		expiresAt = expiresAt.UTC()
		expiresArg = expiresAt
		link.ExpiresAt = expiresAt.Format(time.RFC3339)
	}
	_, err := r.db.Exec(
		`INSERT INTO battle_report_links (id, report_id, token, visibility, expires_at, created_at)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		link.ID, link.ReportID, link.Token, link.Visibility, expiresArg, now,
	)
	return link, err
}

// ListBattleEventsForAdmin 返回 GM 战斗事件列表。
func (r *MySQLRepository) ListBattleEventsForAdmin(query game.BattleEventQuery) ([]game.BattleEvent, int, error) {
	if query.Page <= 0 {
		query.Page = 1
	}
	if query.PageSize <= 0 {
		query.PageSize = 20
	}
	if query.PageSize > 100 {
		query.PageSize = 100
	}
	where, args := buildBattleEventWhere(query)
	var total int
	if err := r.db.QueryRow(`SELECT COUNT(*) FROM battle_events `+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	offset := (query.Page - 1) * query.PageSize
	rows, err := r.db.Query(
		`SELECT id, source_type, source_id, scene, battle_type, result,
			attacker_player_id, defender_player_id, attacker_name, defender_name,
			attacker_faction, defender_faction, related_march_id, related_reinforcement_id,
			summary_json, snapshot_json, result_json, occurred_at, created_at
		 FROM battle_events `+where+` ORDER BY occurred_at DESC LIMIT ? OFFSET ?`,
		append(args, query.PageSize, offset)...,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	items := []game.BattleEvent{}
	for rows.Next() {
		event, err := scanBattleEvent(rows)
		if err != nil {
			return nil, 0, err
		}
		items = append(items, event)
	}
	return items, total, rows.Err()
}

// GetBattleEventForAdmin 获取单个战斗事件。
func (r *MySQLRepository) GetBattleEventForAdmin(eventID string) (game.BattleEvent, error) {
	event, err := scanBattleEvent(r.db.QueryRow(
		`SELECT id, source_type, source_id, scene, battle_type, result,
			attacker_player_id, defender_player_id, attacker_name, defender_name,
			attacker_faction, defender_faction, related_march_id, related_reinforcement_id,
			summary_json, snapshot_json, result_json, occurred_at, created_at
		 FROM battle_events WHERE id = ? LIMIT 1`,
		eventID,
	))
	if errors.Is(err, sql.ErrNoRows) {
		return game.BattleEvent{}, errors.New("battle event not found")
	}
	return event, err
}

// ListReportsByEventForAdmin 返回同一事件下所有玩家视角战报。
func (r *MySQLRepository) ListReportsByEventForAdmin(eventID string) ([]game.BattleReport, error) {
	rows, err := r.db.Query(
		`SELECT report_json, COALESCE(CAST(detail_json AS CHAR), 'null'), is_read FROM battle_reports WHERE event_id = ? ORDER BY created_at DESC`,
		eventID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	reports := []game.BattleReport{}
	for rows.Next() {
		var reportJSON []byte
		var detailJSON []byte
		var isRead bool
		if err := rows.Scan(&reportJSON, &detailJSON, &isRead); err != nil {
			return nil, err
		}
		report, err := scanBattleReportJSON(reportJSON, detailJSON, isRead)
		if err != nil {
			continue
		}
		reports = append(reports, report)
	}
	return reports, rows.Err()
}

// ListParticipantsByEventForAdmin 返回同一事件下所有参与方快照。
func (r *MySQLRepository) ListParticipantsByEventForAdmin(eventID string) ([]game.BattleReportParticipant, error) {
	rows, err := r.db.Query(
		`SELECT id, event_id, report_id, player_id, role, faction, nickname, city_name,
			troops_before_json, troops_lost_json, troops_survived_json, generals_json,
			rewards_json, points_delta_json, created_at
		 FROM battle_report_participants
		 WHERE event_id = ?
		 ORDER BY created_at ASC, report_id ASC, role ASC`,
		eventID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	participants := []game.BattleReportParticipant{}
	for rows.Next() {
		participant, err := scanBattleReportParticipant(rows)
		if err != nil {
			return nil, err
		}
		participants = append(participants, participant)
	}
	return participants, rows.Err()
}

// buildBattleEventWhere 构建 GM 战斗事件查询条件。
func buildBattleEventWhere(query game.BattleEventQuery) (string, []any) {
	conditions := []string{"1=1"}
	args := []any{}
	if query.EventID != "" {
		conditions = append(conditions, "id = ?")
		args = append(args, query.EventID)
	}
	if query.PlayerID != "" {
		conditions = append(conditions, "(attacker_player_id = ? OR defender_player_id = ?)")
		args = append(args, query.PlayerID, query.PlayerID)
	}
	if query.SourceType != "" {
		conditions = append(conditions, "source_type = ?")
		args = append(args, query.SourceType)
	}
	if query.SourceID != "" {
		conditions = append(conditions, "source_id = ?")
		args = append(args, query.SourceID)
	}
	if query.BattleType != "" {
		conditions = append(conditions, "battle_type = ?")
		args = append(args, query.BattleType)
	}
	if query.Result != "" {
		conditions = append(conditions, "result = ?")
		args = append(args, query.Result)
	}
	if query.RelatedMarchID != "" {
		conditions = append(conditions, "related_march_id = ?")
		args = append(args, query.RelatedMarchID)
	}
	if query.RelatedReinforcementID != "" {
		conditions = append(conditions, "related_reinforcement_id = ?")
		args = append(args, query.RelatedReinforcementID)
	}
	if !query.TimeFrom.IsZero() {
		conditions = append(conditions, "occurred_at >= ?")
		args = append(args, query.TimeFrom.UTC())
	}
	if !query.TimeTo.IsZero() {
		conditions = append(conditions, "occurred_at <= ?")
		args = append(args, query.TimeTo.UTC())
	}
	return "WHERE " + strings.Join(conditions, " AND "), args
}

// scanBattleEvent 扫描 GM 战斗事件记录。
func scanBattleEvent(scanner interface{ Scan(dest ...any) error }) (game.BattleEvent, error) {
	var event game.BattleEvent
	var summaryJSON, snapshotJSON, resultJSON sql.NullString
	var occurredAt, createdAt time.Time
	err := scanner.Scan(
		&event.ID, &event.SourceType, &event.SourceID, &event.Scene, &event.BattleType, &event.Result,
		&event.AttackerPlayerID, &event.DefenderPlayerID, &event.AttackerName, &event.DefenderName,
		&event.AttackerFaction, &event.DefenderFaction, &event.RelatedMarchID, &event.RelatedReinforcementID,
		&summaryJSON, &snapshotJSON, &resultJSON, &occurredAt, &createdAt,
	)
	if err != nil {
		return game.BattleEvent{}, err
	}
	unmarshalNullJSON(summaryJSON, &event.Summary)
	unmarshalNullJSON(snapshotJSON, &event.Snapshot)
	unmarshalNullJSON(resultJSON, &event.ResultData)
	event.OccurredAt = occurredAt.UTC().Format(time.RFC3339)
	event.CreatedAt = createdAt.UTC().Format(time.RFC3339)
	return event, nil
}

// scanBattleReportParticipant 扫描 GM 参与方快照记录。
func scanBattleReportParticipant(scanner interface{ Scan(dest ...any) error }) (game.BattleReportParticipant, error) {
	var participant game.BattleReportParticipant
	var troopsBeforeJSON, troopsLostJSON, troopsSurvivedJSON sql.NullString
	var generalsJSON, rewardsJSON, pointsDeltaJSON sql.NullString
	var createdAt time.Time
	err := scanner.Scan(
		&participant.ID,
		&participant.EventID,
		&participant.ReportID,
		&participant.PlayerID,
		&participant.Role,
		&participant.Faction,
		&participant.Nickname,
		&participant.CityName,
		&troopsBeforeJSON,
		&troopsLostJSON,
		&troopsSurvivedJSON,
		&generalsJSON,
		&rewardsJSON,
		&pointsDeltaJSON,
		&createdAt,
	)
	if err != nil {
		return game.BattleReportParticipant{}, err
	}
	unmarshalNullJSON(troopsBeforeJSON, &participant.TroopsBefore)
	unmarshalNullJSON(troopsLostJSON, &participant.TroopsLost)
	unmarshalNullJSON(troopsSurvivedJSON, &participant.TroopsSurvived)
	unmarshalNullJSON(generalsJSON, &participant.Generals)
	unmarshalNullJSON(rewardsJSON, &participant.Rewards)
	unmarshalNullJSON(pointsDeltaJSON, &participant.PointsDelta)
	participant.CreatedAt = createdAt.UTC().Format(time.RFC3339)
	return participant, nil
}

// reportRandomID 生成战报分享使用的随机十六进制 ID。
func reportRandomID(bytesCount int) string {
	if bytesCount <= 0 {
		bytesCount = 8
	}
	buf := make([]byte, bytesCount)
	if _, err := rand.Read(buf); err != nil {
		return hex.EncodeToString([]byte(time.Now().Format("20060102150405.000000000")))
	}
	return hex.EncodeToString(buf)
}

// CountUnreadReports 统计玩家未读战报。
func (r *MySQLRepository) CountUnreadReports(playerID string) (int, error) {
	threeDaysAgo := time.Now().Add(-3 * 24 * time.Hour).UTC()
	var count int
	err := r.db.QueryRow(
		`SELECT COUNT(*) FROM battle_reports WHERE player_id = ? AND is_read = 0 AND deleted_by_player = 0 AND created_at > ?`,
		playerID, threeDaysAgo,
	).Scan(&count)
	return count, err
}

// --- Mail Methods ---
