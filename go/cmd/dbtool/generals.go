// 本文件归口 player_generals 与 player_general_assignments 权威表回填和一致性校验。
package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"strings"
	"time"

	"hero3/internal/app/game"
	"hero3/internal/infrastructure/storage"
)

type generalsBackfillResult struct {
	Players     int
	Generals    int
	Assignments int
}

type generalsVerifyResult struct {
	Players             int
	ExpectedGenerals    int
	ActualGenerals      int
	ExpectedAssignments int
	ActualAssignments   int
	Mismatches          int
}

type generalSnapshot struct {
	Level int
	Exp   int
	Stats string
}

type generalAssignmentSnapshot struct {
	GeneralID string
	Slot      string
	ModuleID  string
	Status    string
}

func runBackfillGenerals(args []string) error {
	flags := flag.NewFlagSet("backfill-generals", flag.ContinueOnError)
	dsn := flags.String("dsn", "", "目标数据库 DSN，默认使用 HERO3_DATABASE_DSN")
	allowNonTest := flags.Bool("allow-non-test", false, "允许回填非 test_ 前缀数据库")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if strings.TrimSpace(*dsn) == "" {
		configured, err := configuredDSN()
		if err != nil {
			return err
		}
		*dsn = configured
	}
	databaseName, err := storage.MySQLDatabaseName(*dsn)
	if err != nil {
		return err
	}
	if !strings.HasPrefix(databaseName, "test_") && !*allowNonTest {
		return fmt.Errorf("target database must use test_ prefix or pass --allow-non-test")
	}
	ctx, cancel := context.WithTimeout(context.Background(), longCommandTimeout)
	defer cancel()
	result, err := backfillPlayerGenerals(ctx, *dsn)
	if err != nil {
		return err
	}
	fmt.Printf("武将权威表回填完成：数据库 %s，玩家 %d，武将行 %d，占用行 %d\n", databaseName, result.Players, result.Generals, result.Assignments)
	return nil
}

func runVerifyGenerals(args []string) error {
	flags := flag.NewFlagSet("verify-generals", flag.ContinueOnError)
	dsn := flags.String("dsn", "", "目标数据库 DSN，默认使用 HERO3_DATABASE_DSN")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if strings.TrimSpace(*dsn) == "" {
		configured, err := configuredDSN()
		if err != nil {
			return err
		}
		*dsn = configured
	}
	ctx, cancel := context.WithTimeout(context.Background(), longCommandTimeout)
	defer cancel()
	result, err := verifyPlayerGenerals(ctx, *dsn)
	if err != nil {
		return err
	}
	if result.Mismatches > 0 {
		return fmt.Errorf("武将权威表兼容快照校验失败：玩家 %d，期望武将行 %d，实际武将行 %d，期望占用行 %d，实际占用行 %d，不一致 %d", result.Players, result.ExpectedGenerals, result.ActualGenerals, result.ExpectedAssignments, result.ActualAssignments, result.Mismatches)
	}
	databaseName, err := storage.MySQLDatabaseName(*dsn)
	if err != nil {
		return err
	}
	fmt.Printf("武将权威表兼容快照校验通过：数据库 %s，玩家 %d，武将行 %d，占用行 %d\n", databaseName, result.Players, result.ActualGenerals, result.ActualAssignments)
	return nil
}

func backfillPlayerGenerals(ctx context.Context, dsn string) (generalsBackfillResult, error) {
	db, err := storage.OpenMySQL(ctx, dsn)
	if err != nil {
		return generalsBackfillResult{}, err
	}
	defer db.Close()

	expectedGenerals, expectedAssignments, err := loadPlayerGeneralSnapshots(ctx, db)
	if err != nil {
		return generalsBackfillResult{}, err
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return generalsBackfillResult{}, err
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.ExecContext(ctx, `DELETE FROM player_general_assignments`); err != nil {
		return generalsBackfillResult{}, err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM player_generals`); err != nil {
		return generalsBackfillResult{}, err
	}
	result := generalsBackfillResult{Players: len(expectedGenerals)}
	now := time.Now().UTC()
	for playerID, generals := range expectedGenerals {
		for generalID, snapshot := range generals {
			hero, _ := game.GetHeroConfig(generalID)
			if _, err := tx.ExecContext(ctx,
				`INSERT INTO player_generals (player_id, general_id, faction, level, exp, stats_json, acquired_at, updated_at)
				 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
				playerID,
				generalID,
				hero.Faction,
				snapshot.Level,
				snapshot.Exp,
				[]byte(snapshot.Stats),
				now,
				now,
			); err != nil {
				return generalsBackfillResult{}, err
			}
			result.Generals++
		}
	}
	for playerID, assignments := range expectedAssignments {
		for assignmentID, snapshot := range assignments {
			if _, err := tx.ExecContext(ctx,
				`INSERT INTO player_general_assignments (player_id, assignment_id, general_id, assignment_slot, module_id, status, assigned_at, ends_at, updated_at)
				 VALUES (?, ?, ?, ?, ?, ?, ?, NULL, ?)`,
				playerID,
				assignmentID,
				snapshot.GeneralID,
				snapshot.Slot,
				snapshot.ModuleID,
				snapshot.Status,
				now,
				now,
			); err != nil {
				return generalsBackfillResult{}, err
			}
			result.Assignments++
		}
	}
	if err := tx.Commit(); err != nil {
		return generalsBackfillResult{}, err
	}
	return result, nil
}

func verifyPlayerGenerals(ctx context.Context, dsn string) (generalsVerifyResult, error) {
	db, err := storage.OpenMySQL(ctx, dsn)
	if err != nil {
		return generalsVerifyResult{}, err
	}
	defer db.Close()

	expectedGenerals, expectedAssignments, err := loadPlayerGeneralSnapshots(ctx, db)
	if err != nil {
		return generalsVerifyResult{}, err
	}
	actualGenerals, err := loadPlayerGeneralRows(ctx, db)
	if err != nil {
		return generalsVerifyResult{}, err
	}
	actualAssignments, err := loadPlayerGeneralAssignmentRows(ctx, db)
	if err != nil {
		return generalsVerifyResult{}, err
	}
	result := generalsVerifyResult{Players: len(expectedGenerals)}
	for playerID, generals := range expectedGenerals {
		result.ExpectedGenerals += len(generals)
		for generalID, want := range generals {
			got, ok := actualGenerals[playerID][generalID]
			if !ok || got != want {
				result.Mismatches++
			}
		}
	}
	for playerID, generals := range actualGenerals {
		result.ActualGenerals += len(generals)
		for generalID := range generals {
			if _, ok := expectedGenerals[playerID][generalID]; !ok {
				result.Mismatches++
			}
		}
	}
	for playerID, assignments := range expectedAssignments {
		result.ExpectedAssignments += len(assignments)
		for assignmentID, want := range assignments {
			got, ok := actualAssignments[playerID][assignmentID]
			if !ok || got != want {
				result.Mismatches++
			}
		}
	}
	for playerID, assignments := range actualAssignments {
		result.ActualAssignments += len(assignments)
		for assignmentID := range assignments {
			if _, ok := expectedAssignments[playerID][assignmentID]; !ok {
				result.Mismatches++
			}
		}
	}
	return result, nil
}

func loadPlayerGeneralSnapshots(ctx context.Context, db *sql.DB) (map[string]map[string]generalSnapshot, map[string]map[string]generalAssignmentSnapshot, error) {
	states, err := loadPlayerStates(ctx, db)
	if err != nil {
		return nil, nil, err
	}
	generals := map[string]map[string]generalSnapshot{}
	assignments := map[string]map[string]generalAssignmentSnapshot{}
	for playerID, state := range states {
		game.EnsureGeneralRoster(&state, time.Now())
		generals[playerID] = generalSnapshotsFromState(state.Generals)
		assignments[playerID] = generalAssignmentSnapshotsFromState(state.GeneralAssignments)
	}
	return generals, assignments, nil
}

func loadPlayerGeneralRows(ctx context.Context, db *sql.DB) (map[string]map[string]generalSnapshot, error) {
	rows, err := db.QueryContext(ctx, `SELECT player_id, general_id, level, exp, stats_json FROM player_generals ORDER BY player_id, general_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := map[string]map[string]generalSnapshot{}
	for rows.Next() {
		var playerID string
		var generalID string
		var snapshot generalSnapshot
		var statsJSON []byte
		if err := rows.Scan(&playerID, &generalID, &snapshot.Level, &snapshot.Exp, &statsJSON); err != nil {
			return nil, err
		}
		snapshot.Stats = normalizeJSONMap(statsJSON)
		if result[playerID] == nil {
			result[playerID] = map[string]generalSnapshot{}
		}
		result[playerID][generalID] = snapshot
	}
	return result, rows.Err()
}

func loadPlayerGeneralAssignmentRows(ctx context.Context, db *sql.DB) (map[string]map[string]generalAssignmentSnapshot, error) {
	rows, err := db.QueryContext(ctx, `SELECT player_id, assignment_id, general_id, assignment_slot, module_id, status FROM player_general_assignments ORDER BY player_id, assignment_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := map[string]map[string]generalAssignmentSnapshot{}
	for rows.Next() {
		var playerID string
		var assignmentID string
		var snapshot generalAssignmentSnapshot
		if err := rows.Scan(&playerID, &assignmentID, &snapshot.GeneralID, &snapshot.Slot, &snapshot.ModuleID, &snapshot.Status); err != nil {
			return nil, err
		}
		if result[playerID] == nil {
			result[playerID] = map[string]generalAssignmentSnapshot{}
		}
		result[playerID][assignmentID] = snapshot
	}
	return result, rows.Err()
}

func generalSnapshotsFromState(generals []game.General) map[string]generalSnapshot {
	result := map[string]generalSnapshot{}
	for _, general := range generals {
		general.ID = strings.TrimSpace(general.ID)
		if general.ID == "" {
			continue
		}
		statsJSON, _ := json.Marshal(general.Stats)
		result[general.ID] = generalSnapshot{
			Level: general.Level,
			Exp:   general.Exp,
			Stats: normalizeJSONMap(statsJSON),
		}
	}
	return result
}

func generalAssignmentSnapshotsFromState(assignments []game.GeneralAssignment) map[string]generalAssignmentSnapshot {
	result := map[string]generalAssignmentSnapshot{}
	for _, assignment := range assignments {
		assignment.ID = strings.TrimSpace(assignment.ID)
		assignment.GeneralID = strings.TrimSpace(assignment.GeneralID)
		if assignment.ID == "" || assignment.GeneralID == "" {
			continue
		}
		result[assignment.ID] = generalAssignmentSnapshot{
			GeneralID: assignment.GeneralID,
			Slot:      assignment.Slot,
			ModuleID:  assignment.ModuleID,
			Status:    assignment.Status,
		}
	}
	return result
}

func normalizeJSONMap(data []byte) string {
	if len(data) == 0 {
		data = []byte(`{}`)
	}
	var value map[string]int
	if err := json.Unmarshal(data, &value); err != nil || value == nil {
		value = map[string]int{}
	}
	normalized, _ := json.Marshal(value)
	return string(normalized)
}
