// 本文件实现 MySQL 版黄巾起义来袭队列仓储。
package storage

import (
	"database/sql"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"hero3/internal/app/game"
)

const yellowTurbanMarchColumns = `id, target_player_id, source_city_id, source_name, source_faction, source_region_id,
	risk_level_id, risk_level_name, player_food, food_capacity, pressure,
	troops_json, status, duration_seconds, started_at, arrives_at, resolved_at,
	defender_report_id, error_message, created_at, updated_at`

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
	return getYellowTurbanMarchTx(r.db, marchID, "")
}

// UpdateYellowTurbanMarch 在事务内更新黄巾来袭。
func (r *MySQLRepository) UpdateYellowTurbanMarch(marchID string, updatedAt time.Time, update func(march *game.YellowTurbanMarch) error) (game.YellowTurbanMarch, error) {
	tx, err := r.db.Begin()
	if err != nil {
		return game.YellowTurbanMarch{}, err
	}
	defer func() { _ = tx.Rollback() }()
	march, err := getYellowTurbanMarchTx(tx, marchID, " FOR UPDATE")
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
	if err := updateYellowTurbanMarchTx(tx, march); err != nil {
		return game.YellowTurbanMarch{}, err
	}
	if err := tx.Commit(); err != nil {
		return game.YellowTurbanMarch{}, err
	}
	return march, nil
}

// ResolveYellowTurbanBattleTransaction 在事务内结算黄巾防守和驻防协防。
func (r *MySQLRepository) ResolveYellowTurbanBattleTransaction(marchID string, updatedAt time.Time, update func(defender *game.GameState, reinforcements []game.Reinforcement, march *game.YellowTurbanMarch) (game.BattleReport, []game.BattleReport, []game.Reinforcement, error)) (game.GameState, game.YellowTurbanMarch, game.BattleReport, []game.BattleReport, error) {
	tx, err := r.db.Begin()
	if err != nil {
		return game.GameState{}, game.YellowTurbanMarch{}, game.BattleReport{}, nil, err
	}
	defer func() { _ = tx.Rollback() }()
	march, err := getYellowTurbanMarchTx(tx, marchID, " FOR UPDATE")
	if err != nil {
		return game.GameState{}, game.YellowTurbanMarch{}, game.BattleReport{}, nil, err
	}
	defender, defenderJSON, err := loadPvpPlayerStateTx(tx, march.TargetPlayerID)
	if err != nil {
		return game.GameState{}, game.YellowTurbanMarch{}, game.BattleReport{}, nil, err
	}
	defenderArmy := armySnapshotsFromStorageState(defender.Army)
	defenderAssignments := generalAssignmentSnapshotsFromStorageState(defender.GeneralAssignments)
	reinforcements, err := listReceivedReinforcementsTx(tx, defender.Player.ID, " FOR UPDATE")
	if err != nil {
		return game.GameState{}, game.YellowTurbanMarch{}, game.BattleReport{}, nil, err
	}
	report, reinforcementReports, changedReinforcements, err := update(&defender, reinforcements, &march)
	if err != nil {
		return game.GameState{}, game.YellowTurbanMarch{}, game.BattleReport{}, nil, err
	}
	if updatedAt.IsZero() {
		updatedAt = time.Now().UTC()
	}
	defender.ServerTime = updatedAt.UTC().Format(time.RFC3339)
	if err := savePvpPlayerStateTx(tx, defender.Player.ID, defender, defenderJSON, updatedAt, defenderArmy, defenderAssignments); err != nil {
		return game.GameState{}, game.YellowTurbanMarch{}, game.BattleReport{}, nil, err
	}
	for _, record := range changedReinforcements {
		if err := updateReinforcementTx(tx, record.ID, record); err != nil {
			return game.GameState{}, game.YellowTurbanMarch{}, game.BattleReport{}, nil, err
		}
	}
	march.UpdatedAt = updatedAt.UTC().Format(time.RFC3339)
	if err := updateYellowTurbanMarchTx(tx, march); err != nil {
		return game.GameState{}, game.YellowTurbanMarch{}, game.BattleReport{}, nil, err
	}
	if err := tx.Commit(); err != nil {
		return game.GameState{}, game.YellowTurbanMarch{}, game.BattleReport{}, nil, err
	}
	return defender, march, report, reinforcementReports, nil
}

// ListYellowTurbanMarchesForPlayer 返回玩家黄巾来袭列表。
func (r *MySQLRepository) ListYellowTurbanMarchesForPlayer(playerID string) ([]game.YellowTurbanMarch, error) {
	rows, err := r.db.Query(`
		SELECT `+yellowTurbanMarchColumns+`
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
		SELECT `+yellowTurbanMarchColumns+`
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

type yellowTurbanMarchQueryer interface {
	QueryRow(query string, args ...any) *sql.Row
}

// getYellowTurbanMarchTx 按需加锁读取单条黄巾来袭。
func getYellowTurbanMarchTx(queryer yellowTurbanMarchQueryer, marchID string, lockClause string) (game.YellowTurbanMarch, error) {
	lockClause = strings.TrimSpace(lockClause)
	if lockClause != "" {
		lockClause = " " + lockClause
	}
	return scanYellowTurbanMarch(queryer.QueryRow(
		`SELECT `+yellowTurbanMarchColumns+`
		 FROM yellow_turban_marches
		 WHERE id = ?
		 LIMIT 1`+lockClause,
		marchID,
	))
}

// updateYellowTurbanMarchTx 保存黄巾来袭状态。
func updateYellowTurbanMarchTx(tx *sql.Tx, march game.YellowTurbanMarch) error {
	troopsJSON, err := json.Marshal(march.Troops)
	if err != nil {
		return err
	}
	_, err = tx.Exec(`
		UPDATE yellow_turban_marches
		SET source_name = ?, source_faction = ?, risk_level_id = ?, risk_level_name = ?,
		    player_food = ?, food_capacity = ?, pressure = ?, troops_json = ?, status = ?,
		    duration_seconds = ?, started_at = ?, arrives_at = ?, resolved_at = ?,
		    defender_report_id = ?, error_message = ?, updated_at = ?
		WHERE id = ?
	`, march.SourceName, march.SourceFaction, march.RiskLevelID, march.RiskLevelName,
		march.PlayerFood, march.FoodCapacity, march.Pressure, troopsJSON, march.Status,
		march.DurationSeconds, parseNullableTime(march.StartedAt), parseNullableTime(march.ArrivesAt), parseNullableTime(march.ResolvedAt),
		march.DefenderReportID, march.Error, parseNullableTime(march.UpdatedAt), march.ID)
	return err
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
