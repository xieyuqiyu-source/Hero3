// 本文件归口当前权威表架构健康检查，区别于迁移期 legacy verify。
package main

import (
	"context"
	"flag"
	"fmt"
	"strings"
	"time"

	"hero3/internal/infrastructure/storage"
)

type authorityHealthcheckResult struct {
	Players              int
	MissingResources     int
	MissingBuildings     int
	MissingResourceSlots int
	MissingGenerals      int
	BigSnapshotPlayers   int
}

// runHealthcheckAuthority 检查当前权威表覆盖和轻量 state_json 是否干净。
func runHealthcheckAuthority(args []string) error {
	flags := flag.NewFlagSet("healthcheck-authority", flag.ContinueOnError)
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

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	result, err := healthcheckAuthority(ctx, *dsn)
	if err != nil {
		return err
	}
	if result.MissingResources > 0 || result.MissingBuildings > 0 || result.MissingResourceSlots > 0 || result.MissingGenerals > 0 || result.BigSnapshotPlayers > 0 {
		return fmt.Errorf("权威表健康检查失败：玩家 %d，缺资源 %d，缺建筑 %d，缺资源田 %d，缺武将 %d，state_json 大字段残留玩家 %d",
			result.Players,
			result.MissingResources,
			result.MissingBuildings,
			result.MissingResourceSlots,
			result.MissingGenerals,
			result.BigSnapshotPlayers,
		)
	}
	fmt.Printf("权威表健康检查通过：玩家 %d，基础权威表完整，state_json 无大字段残留\n", result.Players)
	return nil
}

// healthcheckAuthority 汇总当前库的权威表健康状态。
func healthcheckAuthority(ctx context.Context, dsn string) (authorityHealthcheckResult, error) {
	db, err := storage.OpenMySQL(ctx, dsn)
	if err != nil {
		return authorityHealthcheckResult{}, err
	}
	defer db.Close()

	var result authorityHealthcheckResult
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM players`).Scan(&result.Players); err != nil {
		return authorityHealthcheckResult{}, err
	}
	checks := []struct {
		target *int
		table  string
	}{
		{target: &result.MissingResources, table: "player_resources"},
		{target: &result.MissingBuildings, table: "player_buildings"},
		{target: &result.MissingResourceSlots, table: "player_resource_slots"},
		{target: &result.MissingGenerals, table: "player_generals"},
	}
	for _, check := range checks {
		query := fmt.Sprintf(`SELECT COUNT(*)
			FROM players p
			LEFT JOIN (SELECT DISTINCT player_id FROM %s) a ON a.player_id = p.id
			WHERE a.player_id IS NULL`, check.table)
		if err := db.QueryRowContext(ctx, query).Scan(check.target); err != nil {
			return authorityHealthcheckResult{}, err
		}
	}
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*)
		FROM players
		WHERE JSON_CONTAINS_PATH(state_json, 'one',
			'$.resources',
			'$.inventory',
			'$.buildings',
			'$.resourceSlots',
			'$.army',
			'$.recruitQueues',
			'$.generals',
			'$.generalAssignments',
			'$.buffs'
		)`).Scan(&result.BigSnapshotPlayers); err != nil {
		return authorityHealthcheckResult{}, err
	}
	return result, nil
}
