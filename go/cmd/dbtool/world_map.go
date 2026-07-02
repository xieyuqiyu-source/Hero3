// 本文件提供世界地图权威坐标维护命令。
package main

import (
	"context"
	"flag"
	"fmt"

	"hero3/internal/app/game"
	"hero3/internal/infrastructure/storage"
)

// runBackfillWorldPositions 为所有已有玩家补齐世界地图权威坐标。
func runBackfillWorldPositions(args []string) error {
	flags := flag.NewFlagSet("backfill-world-positions", flag.ContinueOnError)
	dsn := flags.String("dsn", "", "目标数据库 DSN，默认使用 HERO3_DATABASE_DSN")
	allowNonTest := flags.Bool("allow-non-test", false, "允许回填非 test_ 前缀数据库")
	if err := flags.Parse(args); err != nil {
		return err
	}
	resolvedDSN, databaseName, err := resolveWritableDbtoolDSN(*dsn, *allowNonTest)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), longCommandTimeout)
	defer cancel()
	db, err := storage.OpenMySQL(ctx, resolvedDSN)
	if err != nil {
		return err
	}
	defer db.Close()
	if err := storage.MigrateMySQL(ctx, db); err != nil {
		return err
	}
	repo := storage.NewMySQLRepository(db)
	result, err := game.NewServiceWithRepository(repo).MigrateWorldPositions()
	fmt.Printf("世界地图坐标补齐：database=%s total=%d created=%d skipped=%d conflicts=%d failed=%d\n", databaseName, result.Total, result.Created, result.Skipped, result.Conflicts, result.Failed)
	for _, detail := range result.ConflictDetails {
		fmt.Printf("冲突：%s\n", detail)
	}
	for _, failure := range result.Failures {
		fmt.Printf("失败：%s\n", failure)
	}
	return err
}
