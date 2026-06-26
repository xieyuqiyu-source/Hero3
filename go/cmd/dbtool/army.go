// 本文件归口 player_army_units 与 player_recruit_queues 权威表回填和一致性校验。
package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"strings"
	"time"

	"hero3/internal/app/game"
	"hero3/internal/infrastructure/storage"
)

type armyBackfillResult struct {
	Players int
	Rows    int
}

type armyVerifyResult struct {
	Players      int
	ExpectedRows int
	ActualRows   int
	Mismatches   int
}

type recruitQueueBackfillResult struct {
	Players int
	Rows    int
}

type recruitQueueVerifyResult struct {
	Players      int
	ExpectedRows int
	ActualRows   int
	Mismatches   int
}

type armySnapshot struct {
	Amount int
}

type recruitQueueSnapshot struct {
	UnitType string
	Amount   int
	EndsAt   string
	Order    int
}

// runBackfillArmy 从旧 players.state_json.army 回填 player_army_units 权威表。
func runBackfillArmy(args []string) error {
	flags := flag.NewFlagSet("backfill-army", flag.ContinueOnError)
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
	result, err := backfillPlayerArmy(ctx, *dsn)
	if err != nil {
		return err
	}
	fmt.Printf("兵力权威表回填完成：数据库 %s，玩家 %d，兵力行 %d\n", databaseName, result.Players, result.Rows)
	return nil
}

// runVerifyArmy 校验 player_army_units 权威表是否和 state_json.army 兼容快照一致。
func runVerifyArmy(args []string) error {
	flags := flag.NewFlagSet("verify-army", flag.ContinueOnError)
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
	result, err := verifyPlayerArmy(ctx, *dsn)
	if err != nil {
		return err
	}
	if result.Mismatches > 0 {
		return fmt.Errorf("兵力权威表兼容快照校验失败：玩家 %d，期望兵力行 %d，实际兵力行 %d，不一致 %d", result.Players, result.ExpectedRows, result.ActualRows, result.Mismatches)
	}
	databaseName, err := storage.MySQLDatabaseName(*dsn)
	if err != nil {
		return err
	}
	fmt.Printf("兵力权威表兼容快照校验通过：数据库 %s，玩家 %d，兵力行 %d\n", databaseName, result.Players, result.ActualRows)
	return nil
}

// runBackfillRecruitQueues 从旧 players.state_json.recruitQueues 回填 player_recruit_queues 权威表。
func runBackfillRecruitQueues(args []string) error {
	flags := flag.NewFlagSet("backfill-recruit-queues", flag.ContinueOnError)
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
	result, err := backfillPlayerRecruitQueues(ctx, *dsn)
	if err != nil {
		return err
	}
	fmt.Printf("征兵队列权威表回填完成：数据库 %s，玩家 %d，队列行 %d\n", databaseName, result.Players, result.Rows)
	return nil
}

// runVerifyRecruitQueues 校验 player_recruit_queues 权威表是否和 state_json.recruitQueues 兼容快照一致。
func runVerifyRecruitQueues(args []string) error {
	flags := flag.NewFlagSet("verify-recruit-queues", flag.ContinueOnError)
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
	result, err := verifyPlayerRecruitQueues(ctx, *dsn)
	if err != nil {
		return err
	}
	if result.Mismatches > 0 {
		return fmt.Errorf("征兵队列权威表兼容快照校验失败：玩家 %d，期望队列行 %d，实际队列行 %d，不一致 %d", result.Players, result.ExpectedRows, result.ActualRows, result.Mismatches)
	}
	databaseName, err := storage.MySQLDatabaseName(*dsn)
	if err != nil {
		return err
	}
	fmt.Printf("征兵队列权威表兼容快照校验通过：数据库 %s，玩家 %d，队列行 %d\n", databaseName, result.Players, result.ActualRows)
	return nil
}

// backfillPlayerArmy 从玩家主状态兼容快照回填兵力权威表。
func backfillPlayerArmy(ctx context.Context, dsn string) (armyBackfillResult, error) {
	db, err := storage.OpenMySQL(ctx, dsn)
	if err != nil {
		return armyBackfillResult{}, err
	}
	defer db.Close()
	expected, err := loadPlayerArmySnapshots(ctx, db)
	if err != nil {
		return armyBackfillResult{}, err
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return armyBackfillResult{}, err
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.ExecContext(ctx, `DELETE FROM player_army_units`); err != nil {
		return armyBackfillResult{}, err
	}
	result := armyBackfillResult{Players: len(expected)}
	now := time.Now().UTC()
	for playerID, army := range expected {
		for unitType, snapshot := range army {
			if _, err := tx.ExecContext(ctx,
				`INSERT INTO player_army_units (player_id, unit_type, amount, updated_at)
				 VALUES (?, ?, ?, ?)`,
				playerID,
				unitType,
				snapshot.Amount,
				now,
			); err != nil {
				return armyBackfillResult{}, err
			}
			result.Rows++
		}
	}
	if err := tx.Commit(); err != nil {
		return armyBackfillResult{}, err
	}
	return result, nil
}

// verifyPlayerArmy 校验兵力权威表与玩家主状态兼容快照是否一致。
func verifyPlayerArmy(ctx context.Context, dsn string) (armyVerifyResult, error) {
	db, err := storage.OpenMySQL(ctx, dsn)
	if err != nil {
		return armyVerifyResult{}, err
	}
	defer db.Close()
	expected, err := loadPlayerArmySnapshots(ctx, db)
	if err != nil {
		return armyVerifyResult{}, err
	}
	actual, err := loadPlayerArmyRows(ctx, db)
	if err != nil {
		return armyVerifyResult{}, err
	}
	result := armyVerifyResult{Players: len(expected)}
	for playerID, army := range expected {
		result.ExpectedRows += len(army)
		for unitType, want := range army {
			got, ok := actual[playerID][unitType]
			if !ok || got != want {
				result.Mismatches++
			}
		}
	}
	for playerID, army := range actual {
		result.ActualRows += len(army)
		for unitType := range army {
			if _, ok := expected[playerID][unitType]; !ok {
				result.Mismatches++
			}
		}
	}
	return result, nil
}

// backfillPlayerRecruitQueues 从玩家主状态兼容快照回填征兵队列权威表。
func backfillPlayerRecruitQueues(ctx context.Context, dsn string) (recruitQueueBackfillResult, error) {
	db, err := storage.OpenMySQL(ctx, dsn)
	if err != nil {
		return recruitQueueBackfillResult{}, err
	}
	defer db.Close()
	expected, err := loadPlayerRecruitQueueSnapshots(ctx, db)
	if err != nil {
		return recruitQueueBackfillResult{}, err
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return recruitQueueBackfillResult{}, err
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.ExecContext(ctx, `DELETE FROM player_recruit_queues`); err != nil {
		return recruitQueueBackfillResult{}, err
	}
	result := recruitQueueBackfillResult{Players: len(expected)}
	now := time.Now().UTC()
	for playerID, queues := range expected {
		for queueID, snapshot := range queues {
			endsAt, err := time.Parse(time.RFC3339, snapshot.EndsAt)
			if err != nil {
				continue
			}
			if _, err := tx.ExecContext(ctx,
				`INSERT INTO player_recruit_queues (player_id, queue_id, unit_type, amount, ends_at, queue_order, updated_at)
				 VALUES (?, ?, ?, ?, ?, ?, ?)`,
				playerID,
				queueID,
				snapshot.UnitType,
				snapshot.Amount,
				endsAt.UTC(),
				snapshot.Order,
				now,
			); err != nil {
				return recruitQueueBackfillResult{}, err
			}
			result.Rows++
		}
	}
	if err := tx.Commit(); err != nil {
		return recruitQueueBackfillResult{}, err
	}
	return result, nil
}

// verifyPlayerRecruitQueues 校验征兵队列权威表与玩家主状态兼容快照是否一致。
func verifyPlayerRecruitQueues(ctx context.Context, dsn string) (recruitQueueVerifyResult, error) {
	db, err := storage.OpenMySQL(ctx, dsn)
	if err != nil {
		return recruitQueueVerifyResult{}, err
	}
	defer db.Close()
	expected, err := loadPlayerRecruitQueueSnapshots(ctx, db)
	if err != nil {
		return recruitQueueVerifyResult{}, err
	}
	actual, err := loadPlayerRecruitQueueRows(ctx, db)
	if err != nil {
		return recruitQueueVerifyResult{}, err
	}
	result := recruitQueueVerifyResult{Players: len(expected)}
	for playerID, queues := range expected {
		result.ExpectedRows += len(queues)
		for queueID, want := range queues {
			got, ok := actual[playerID][queueID]
			if !ok || got != want {
				result.Mismatches++
			}
		}
	}
	for playerID, queues := range actual {
		result.ActualRows += len(queues)
		for queueID := range queues {
			if _, ok := expected[playerID][queueID]; !ok {
				result.Mismatches++
			}
		}
	}
	return result, nil
}

// loadPlayerArmySnapshots 从 players.state_json 读取兵力快照。
func loadPlayerArmySnapshots(ctx context.Context, db *sql.DB) (map[string]map[string]armySnapshot, error) {
	states, err := loadPlayerStates(ctx, db)
	if err != nil {
		return nil, err
	}
	result := map[string]map[string]armySnapshot{}
	for playerID, state := range states {
		result[playerID] = armySnapshotsFromState(state.Army)
	}
	return result, nil
}

// loadPlayerArmyRows 读取 player_army_units 当前内容。
func loadPlayerArmyRows(ctx context.Context, db *sql.DB) (map[string]map[string]armySnapshot, error) {
	rows, err := db.QueryContext(ctx, `SELECT player_id, unit_type, amount FROM player_army_units ORDER BY player_id, unit_type`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := map[string]map[string]armySnapshot{}
	for rows.Next() {
		var playerID string
		var unitType string
		var snapshot armySnapshot
		if err := rows.Scan(&playerID, &unitType, &snapshot.Amount); err != nil {
			return nil, err
		}
		if result[playerID] == nil {
			result[playerID] = map[string]armySnapshot{}
		}
		result[playerID][unitType] = snapshot
	}
	return result, rows.Err()
}

// loadPlayerRecruitQueueSnapshots 从 players.state_json 读取征兵队列快照。
func loadPlayerRecruitQueueSnapshots(ctx context.Context, db *sql.DB) (map[string]map[string]recruitQueueSnapshot, error) {
	states, err := loadPlayerStates(ctx, db)
	if err != nil {
		return nil, err
	}
	result := map[string]map[string]recruitQueueSnapshot{}
	for playerID, state := range states {
		result[playerID] = recruitQueueSnapshotsFromState(state.RecruitQueues)
	}
	return result, nil
}

// loadPlayerRecruitQueueRows 读取 player_recruit_queues 当前内容。
func loadPlayerRecruitQueueRows(ctx context.Context, db *sql.DB) (map[string]map[string]recruitQueueSnapshot, error) {
	rows, err := db.QueryContext(ctx, `SELECT player_id, queue_id, unit_type, amount, ends_at, queue_order FROM player_recruit_queues ORDER BY player_id, queue_order, ends_at, queue_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := map[string]map[string]recruitQueueSnapshot{}
	for rows.Next() {
		var playerID string
		var queueID string
		var snapshot recruitQueueSnapshot
		var endsAt time.Time
		if err := rows.Scan(&playerID, &queueID, &snapshot.UnitType, &snapshot.Amount, &endsAt, &snapshot.Order); err != nil {
			return nil, err
		}
		snapshot.EndsAt = endsAt.UTC().Format(time.RFC3339)
		if result[playerID] == nil {
			result[playerID] = map[string]recruitQueueSnapshot{}
		}
		result[playerID][queueID] = snapshot
	}
	return result, rows.Err()
}

// armySnapshotsFromState 从兵力列表生成可比较快照。
func armySnapshotsFromState(army []game.ArmyUnit) map[string]armySnapshot {
	result := map[string]armySnapshot{}
	for _, unit := range army {
		unit.UnitType = strings.TrimSpace(unit.UnitType)
		if unit.UnitType == "" || unit.Amount <= 0 {
			continue
		}
		result[unit.UnitType] = armySnapshot{Amount: unit.Amount}
	}
	return result
}

// recruitQueueSnapshotsFromState 从征兵队列生成可比较快照。
func recruitQueueSnapshotsFromState(queues []game.RecruitQueue) map[string]recruitQueueSnapshot {
	result := map[string]recruitQueueSnapshot{}
	for index, queue := range queues {
		queue.ID = strings.TrimSpace(queue.ID)
		queue.UnitType = strings.TrimSpace(queue.UnitType)
		if queue.ID == "" || queue.UnitType == "" || queue.Amount <= 0 {
			continue
		}
		result[queue.ID] = recruitQueueSnapshot{
			UnitType: queue.UnitType,
			Amount:   queue.Amount,
			EndsAt:   strings.TrimSpace(queue.EndsAt),
			Order:    index,
		}
	}
	return result
}
