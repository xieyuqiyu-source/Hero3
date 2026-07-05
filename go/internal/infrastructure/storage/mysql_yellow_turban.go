// 本文件实现 MySQL 版黄巾起义来袭队列仓储。
package storage

import (
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"hero3/internal/app/game"
)

// CreateYellowTurbanMarch 创建一条黄巾来袭记录。
func (r *MySQLRepository) CreateYellowTurbanMarch(march game.YellowTurbanMarch) (game.YellowTurbanMarch, error) {
	if march.ID == "" {
		march.ID = "yt_march_" + gameRandomID(12)
	}
	troopsJSON, err := json.Marshal(march.Troops)
	if err != nil {
		return game.YellowTurbanMarch{}, err
	}
	if _, err := r.db.Exec(`
		INSERT INTO yellow_turban_marches (
			id, target_player_id, source_city_id, source_name, source_faction, source_region_id,
			risk_level_id, risk_level_name, player_food, food_capacity, pressure,
			troops_json, status, duration_seconds, started_at, arrives_at, resolved_at,
			defender_report_id, error_message, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, march.ID, march.TargetPlayerID, march.SourceCityID, march.SourceName, march.SourceFaction, march.SourceRegionID,
		march.RiskLevelID, march.RiskLevelName, march.PlayerFood, march.FoodCapacity, march.Pressure,
		troopsJSON, march.Status, march.DurationSeconds, parseNullableTime(march.StartedAt), parseNullableTime(march.ArrivesAt), parseNullableTime(march.ResolvedAt),
		march.DefenderReportID, march.Error, parseNullableTime(march.CreatedAt), parseNullableTime(march.UpdatedAt)); err != nil {
		return game.YellowTurbanMarch{}, err
	}
	return march, nil
}

// GetYellowTurbanMarch 读取单条黄巾来袭。
func (r *MySQLRepository) GetYellowTurbanMarch(marchID string) (game.YellowTurbanMarch, error) {
	row := r.db.QueryRow(`
		SELECT id, target_player_id, source_city_id, source_name, source_faction, source_region_id,
		       risk_level_id, risk_level_name, player_food, food_capacity, pressure,
		       troops_json, status, duration_seconds, started_at, arrives_at, resolved_at,
		       defender_report_id, error_message, created_at, updated_at
		FROM yellow_turban_marches
		WHERE id = ?
		LIMIT 1
	`, marchID)
	return scanYellowTurbanMarch(row)
}

// UpdateYellowTurbanMarch 在事务内更新黄巾来袭。
func (r *MySQLRepository) UpdateYellowTurbanMarch(marchID string, updatedAt time.Time, update func(march *game.YellowTurbanMarch) error) (game.YellowTurbanMarch, error) {
	tx, err := r.db.Begin()
	if err != nil {
		return game.YellowTurbanMarch{}, err
	}
	defer func() { _ = tx.Rollback() }()
	row := tx.QueryRow(`
		SELECT id, target_player_id, source_city_id, source_name, source_faction, source_region_id,
		       risk_level_id, risk_level_name, player_food, food_capacity, pressure,
		       troops_json, status, duration_seconds, started_at, arrives_at, resolved_at,
		       defender_report_id, error_message, created_at, updated_at
		FROM yellow_turban_marches
		WHERE id = ?
		FOR UPDATE
	`, marchID)
	march, err := scanYellowTurbanMarch(row)
	if err != nil {
		return game.YellowTurbanMarch{}, err
	}
	if update != nil {
		if err := update(&march); err != nil {
			return game.YellowTurbanMarch{}, err
		}
	}
	if updatedAt.IsZero() {
		updatedAt = time.Now().UTC()
	}
	march.UpdatedAt = updatedAt.UTC().Format(time.RFC3339)
	troopsJSON, err := json.Marshal(march.Troops)
	if err != nil {
		return game.YellowTurbanMarch{}, err
	}
	if _, err := tx.Exec(`
		UPDATE yellow_turban_marches
		SET source_name = ?, source_faction = ?, risk_level_id = ?, risk_level_name = ?,
		    player_food = ?, food_capacity = ?, pressure = ?, troops_json = ?, status = ?,
		    duration_seconds = ?, started_at = ?, arrives_at = ?, resolved_at = ?,
		    defender_report_id = ?, error_message = ?, updated_at = ?
		WHERE id = ?
	`, march.SourceName, march.SourceFaction, march.RiskLevelID, march.RiskLevelName,
		march.PlayerFood, march.FoodCapacity, march.Pressure, troopsJSON, march.Status,
		march.DurationSeconds, parseNullableTime(march.StartedAt), parseNullableTime(march.ArrivesAt), parseNullableTime(march.ResolvedAt),
		march.DefenderReportID, march.Error, parseNullableTime(march.UpdatedAt), march.ID); err != nil {
		return game.YellowTurbanMarch{}, err
	}
	if err := tx.Commit(); err != nil {
		return game.YellowTurbanMarch{}, err
	}
	return march, nil
}

// ListYellowTurbanMarchesForPlayer 返回玩家黄巾来袭列表。
func (r *MySQLRepository) ListYellowTurbanMarchesForPlayer(playerID string) ([]game.YellowTurbanMarch, error) {
	rows, err := r.db.Query(`
		SELECT id, target_player_id, source_city_id, source_name, source_faction, source_region_id,
		       risk_level_id, risk_level_name, player_food, food_capacity, pressure,
		       troops_json, status, duration_seconds, started_at, arrives_at, resolved_at,
		       defender_report_id, error_message, created_at, updated_at
		FROM yellow_turban_marches
		WHERE target_player_id = ?
		ORDER BY arrives_at ASC, id ASC
	`, playerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanYellowTurbanMarches(rows)
}

// CountActiveYellowTurbanMarches 统计玩家活跃黄巾来袭。
func (r *MySQLRepository) CountActiveYellowTurbanMarches(playerID string) (int, error) {
	var count int
	if err := r.db.QueryRow(`
		SELECT COUNT(*)
		FROM yellow_turban_marches
		WHERE target_player_id = ? AND status IN (?, ?)
	`, playerID, game.YellowTurbanMarchStatusMarching, game.YellowTurbanMarchStatusResolving).Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}

// ListDueYellowTurbanMarches 返回已到达待结算黄巾来袭。
func (r *MySQLRepository) ListDueYellowTurbanMarches(playerID string, now time.Time) ([]game.YellowTurbanMarch, error) {
	args := []any{game.YellowTurbanMarchStatusMarching, now.UTC()}
	wherePlayer := ""
	if playerID != "" {
		wherePlayer = " AND target_player_id = ?"
		args = append(args, playerID)
	}
	rows, err := r.db.Query(`
		SELECT id, target_player_id, source_city_id, source_name, source_faction, source_region_id,
		       risk_level_id, risk_level_name, player_food, food_capacity, pressure,
		       troops_json, status, duration_seconds, started_at, arrives_at, resolved_at,
		       defender_report_id, error_message, created_at, updated_at
		FROM yellow_turban_marches
		WHERE status = ? AND arrives_at <= ?`+wherePlayer+`
		ORDER BY arrives_at ASC, id ASC
	`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanYellowTurbanMarches(rows)
}

type yellowTurbanMarchScanner interface {
	Scan(dest ...any) error
}

// scanYellowTurbanMarch 从 SQL 行读取黄巾来袭。
func scanYellowTurbanMarch(row yellowTurbanMarchScanner) (game.YellowTurbanMarch, error) {
	var march game.YellowTurbanMarch
	var troopsJSON []byte
	var startedAt sql.NullTime
	var arrivesAt sql.NullTime
	var resolvedAt sql.NullTime
	var createdAt sql.NullTime
	var updatedAt sql.NullTime
	err := row.Scan(&march.ID, &march.TargetPlayerID, &march.SourceCityID, &march.SourceName, &march.SourceFaction, &march.SourceRegionID,
		&march.RiskLevelID, &march.RiskLevelName, &march.PlayerFood, &march.FoodCapacity, &march.Pressure,
		&troopsJSON, &march.Status, &march.DurationSeconds, &startedAt, &arrivesAt, &resolvedAt,
		&march.DefenderReportID, &march.Error, &createdAt, &updatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return game.YellowTurbanMarch{}, game.ErrYellowTurbanMarchNotFound
	}
	if err != nil {
		return game.YellowTurbanMarch{}, err
	}
	if err := json.Unmarshal(troopsJSON, &march.Troops); err != nil {
		return game.YellowTurbanMarch{}, err
	}
	march.StartedAt = formatNullTime(startedAt)
	march.ArrivesAt = formatNullTime(arrivesAt)
	march.ResolvedAt = formatNullTime(resolvedAt)
	march.CreatedAt = formatNullTime(createdAt)
	march.UpdatedAt = formatNullTime(updatedAt)
	return march, nil
}

// scanYellowTurbanMarches 读取黄巾来袭列表。
func scanYellowTurbanMarches(rows *sql.Rows) ([]game.YellowTurbanMarch, error) {
	items := []game.YellowTurbanMarch{}
	for rows.Next() {
		item, err := scanYellowTurbanMarch(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}
