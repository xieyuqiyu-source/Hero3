// 本文件归口 MySQL 战报仓储方法。
package storage

import (
	"encoding/json"
	"errors"
	"time"

	"hero3/internal/app/game"
)

func (r *MySQLRepository) SaveReport(report game.BattleReport) error {
	reportJSON, err := json.Marshal(report)
	if err != nil {
		return err
	}

	createdAt, _ := time.Parse(time.RFC3339, report.CreatedAt)
	if createdAt.IsZero() {
		createdAt = time.Now()
	}

	_, err = r.db.Exec(
		`INSERT INTO battle_reports (id, player_id, report_json, type, is_read, deleted_by_player, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		report.ID,
		report.PlayerID,
		reportJSON,
		report.Type,
		report.Read,
		false,
		createdAt.UTC(),
	)
	return err
}

func (r *MySQLRepository) GetReportByID(reportID string) (game.BattleReport, error) {
	var reportJSON []byte
	err := r.db.QueryRow(`SELECT report_json FROM battle_reports WHERE id = ? LIMIT 1`, reportID).Scan(&reportJSON)
	if err != nil {
		return game.BattleReport{}, errors.New("report not found")
	}
	var report game.BattleReport
	if err := json.Unmarshal(reportJSON, &report); err != nil {
		return game.BattleReport{}, err
	}
	return report, nil
}

func (r *MySQLRepository) ListReports(playerID string, limit int, offset int) ([]game.BattleReport, int, error) {
	threeDaysAgo := time.Now().Add(-3 * 24 * time.Hour).UTC()
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
		`SELECT report_json, is_read FROM battle_reports
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
		var reportJSON []byte
		var isRead bool
		if err := rows.Scan(&reportJSON, &isRead); err != nil {
			return nil, 0, err
		}
		var report game.BattleReport
		if err := json.Unmarshal(reportJSON, &report); err != nil {
			continue
		}
		report.Read = isRead
		reports = append(reports, report)
	}
	return reports, total, rows.Err()
}

func (r *MySQLRepository) ListAllReports(playerID string) ([]game.BattleReport, error) {
	rows, err := r.db.Query(
		`SELECT report_json FROM battle_reports
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
		if err := rows.Scan(&reportJSON); err != nil {
			return nil, err
		}
		var report game.BattleReport
		if err := json.Unmarshal(reportJSON, &report); err != nil {
			continue
		}
		reports = append(reports, report)
	}
	return reports, rows.Err()
}

func (r *MySQLRepository) MarkReportsRead(playerID string) error {
	_, err := r.db.Exec(
		`UPDATE battle_reports SET is_read = 1 WHERE player_id = ? AND is_read = 0 AND deleted_by_player = 0`,
		playerID,
	)
	return err
}

func (r *MySQLRepository) MarkSingleReportRead(playerID string, reportID string) error {
	_, err := r.db.Exec(
		`UPDATE battle_reports
		 SET is_read = 1
		 WHERE id = ? AND player_id = ? AND is_read = 0 AND deleted_by_player = 0`,
		reportID, playerID,
	)
	return err
}

func (r *MySQLRepository) DeleteReport(playerID string, reportID string) error {
	_, err := r.db.Exec(
		`UPDATE battle_reports
		 SET deleted_by_player = 1
		 WHERE id = ? AND player_id = ? AND deleted_by_player = 0`,
		reportID, playerID,
	)
	return err
}

func (r *MySQLRepository) DeleteAllReports(playerID string) error {
	_, err := r.db.Exec(
		`UPDATE battle_reports SET deleted_by_player = 1 WHERE player_id = ? AND deleted_by_player = 0`,
		playerID,
	)
	return err
}

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
