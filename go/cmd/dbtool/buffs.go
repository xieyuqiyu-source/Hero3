// 本文件归口 player_buffs 权威表回填和一致性校验。
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

type buffsBackfillResult struct {
	Players int
	Rows    int
}

type buffsVerifyResult struct {
	Players      int
	ExpectedRows int
	ActualRows   int
	Mismatches   int
}

type buffSnapshot struct {
	Source    string
	Key       string
	Value     float64
	Mode      string
	ExpiresAt string
	CreatedAt string
	Note      string
}

func runBackfillBuffs(args []string) error {
	flags := flag.NewFlagSet("backfill-buffs", flag.ContinueOnError)
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
	result, err := backfillPlayerBuffs(ctx, *dsn)
	if err != nil {
		return err
	}
	fmt.Printf("Buff 权威表回填完成：数据库 %s，玩家 %d，Buff 行 %d\n", databaseName, result.Players, result.Rows)
	return nil
}

func runVerifyBuffs(args []string) error {
	flags := flag.NewFlagSet("verify-buffs", flag.ContinueOnError)
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
	result, err := verifyPlayerBuffs(ctx, *dsn)
	if err != nil {
		return err
	}
	if result.Mismatches > 0 {
		return fmt.Errorf("Buff 权威表兼容快照校验失败：玩家 %d，期望 Buff 行 %d，实际 Buff 行 %d，不一致 %d", result.Players, result.ExpectedRows, result.ActualRows, result.Mismatches)
	}
	databaseName, err := storage.MySQLDatabaseName(*dsn)
	if err != nil {
		return err
	}
	fmt.Printf("Buff 权威表兼容快照校验通过：数据库 %s，玩家 %d，Buff 行 %d\n", databaseName, result.Players, result.ActualRows)
	return nil
}

func backfillPlayerBuffs(ctx context.Context, dsn string) (buffsBackfillResult, error) {
	db, err := storage.OpenMySQL(ctx, dsn)
	if err != nil {
		return buffsBackfillResult{}, err
	}
	defer db.Close()
	expected, err := loadPlayerBuffSnapshots(ctx, db)
	if err != nil {
		return buffsBackfillResult{}, err
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return buffsBackfillResult{}, err
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.ExecContext(ctx, `DELETE FROM player_buffs`); err != nil {
		return buffsBackfillResult{}, err
	}
	result := buffsBackfillResult{Players: len(expected)}
	now := time.Now().UTC()
	for playerID, buffs := range expected {
		for buffID, snapshot := range buffs {
			createdAt := parseRFC3339Or(snapshot.CreatedAt, now)
			expiresAt := nullableRFC3339(snapshot.ExpiresAt)
			if _, err := tx.ExecContext(ctx,
				`INSERT INTO player_buffs (player_id, buff_id, source, modifier_key, modifier_value, modifier_mode, expires_at, created_at, note, updated_at)
				 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
				playerID,
				buffID,
				snapshot.Source,
				snapshot.Key,
				snapshot.Value,
				snapshot.Mode,
				expiresAt,
				createdAt,
				snapshot.Note,
				now,
			); err != nil {
				return buffsBackfillResult{}, err
			}
			result.Rows++
		}
	}
	if err := tx.Commit(); err != nil {
		return buffsBackfillResult{}, err
	}
	return result, nil
}

func verifyPlayerBuffs(ctx context.Context, dsn string) (buffsVerifyResult, error) {
	db, err := storage.OpenMySQL(ctx, dsn)
	if err != nil {
		return buffsVerifyResult{}, err
	}
	defer db.Close()
	expected, err := loadPlayerBuffSnapshots(ctx, db)
	if err != nil {
		return buffsVerifyResult{}, err
	}
	actual, err := loadPlayerBuffRows(ctx, db)
	if err != nil {
		return buffsVerifyResult{}, err
	}
	result := buffsVerifyResult{Players: len(expected)}
	for playerID, buffs := range expected {
		result.ExpectedRows += len(buffs)
		for buffID, want := range buffs {
			got, ok := actual[playerID][buffID]
			if !ok || got != want {
				result.Mismatches++
			}
		}
	}
	for playerID, buffs := range actual {
		result.ActualRows += len(buffs)
		for buffID := range buffs {
			if _, ok := expected[playerID][buffID]; !ok {
				result.Mismatches++
			}
		}
	}
	return result, nil
}

func loadPlayerBuffSnapshots(ctx context.Context, db *sql.DB) (map[string]map[string]buffSnapshot, error) {
	states, err := loadPlayerStates(ctx, db)
	if err != nil {
		return nil, err
	}
	result := map[string]map[string]buffSnapshot{}
	for playerID, state := range states {
		result[playerID] = buffSnapshotsFromState(state.Buffs)
	}
	return result, nil
}

func loadPlayerBuffRows(ctx context.Context, db *sql.DB) (map[string]map[string]buffSnapshot, error) {
	rows, err := db.QueryContext(ctx, `SELECT player_id, buff_id, source, modifier_key, modifier_value, modifier_mode, expires_at, created_at, note FROM player_buffs ORDER BY player_id, buff_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := map[string]map[string]buffSnapshot{}
	for rows.Next() {
		var playerID string
		var buffID string
		var snapshot buffSnapshot
		var expiresAt sql.NullTime
		var createdAt time.Time
		if err := rows.Scan(&playerID, &buffID, &snapshot.Source, &snapshot.Key, &snapshot.Value, &snapshot.Mode, &expiresAt, &createdAt, &snapshot.Note); err != nil {
			return nil, err
		}
		if expiresAt.Valid {
			snapshot.ExpiresAt = expiresAt.Time.UTC().Format(time.RFC3339)
		}
		snapshot.CreatedAt = createdAt.UTC().Format(time.RFC3339)
		if result[playerID] == nil {
			result[playerID] = map[string]buffSnapshot{}
		}
		result[playerID][buffID] = snapshot
	}
	return result, rows.Err()
}

func buffSnapshotsFromState(buffs []game.Buff) map[string]buffSnapshot {
	result := map[string]buffSnapshot{}
	for _, buff := range buffs {
		buff.ID = strings.TrimSpace(buff.ID)
		if buff.ID == "" {
			continue
		}
		result[buff.ID] = buffSnapshot{
			Source:    buff.Source,
			Key:       buff.Key,
			Value:     buff.Value,
			Mode:      buff.Mode,
			ExpiresAt: strings.TrimSpace(buff.ExpiresAt),
			CreatedAt: strings.TrimSpace(buff.CreatedAt),
			Note:      buff.Note,
		}
	}
	return result
}

func parseRFC3339Or(value string, fallback time.Time) time.Time {
	parsed, err := time.Parse(time.RFC3339, strings.TrimSpace(value))
	if err != nil {
		return fallback
	}
	return parsed.UTC()
}

func nullableRFC3339(value string) any {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return nil
	}
	return parsed.UTC()
}
