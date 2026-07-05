// 本文件提供黄巾起义线上配置同步命令。
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	"hero3/internal/app/game"
	"hero3/internal/infrastructure/storage"
)

// runSyncYellowTurbanConfig 将文件中的黄巾配置写入 game_configs。
func runSyncYellowTurbanConfig(args []string) error {
	flags := flag.NewFlagSet("sync-yellow-turban-config", flag.ContinueOnError)
	dsn := flags.String("dsn", "", "目标数据库 DSN，默认使用 HERO3_DATABASE_DSN")
	path := flags.String("path", "config/yellow_turban.json", "黄巾配置文件路径")
	allowNonTest := flags.Bool("allow-non-test", false, "允许同步非 test_ 前缀数据库")
	updatedBy := flags.String("updated-by", "dbtool", "配置更新人标记")
	if err := flags.Parse(args); err != nil {
		return err
	}
	content, err := os.ReadFile(*path)
	if err != nil {
		return err
	}
	var cfg game.YellowTurbanConfig
	if err := json.Unmarshal(content, &cfg); err != nil {
		return err
	}
	if err := game.ValidateYellowTurbanConfig(cfg); err != nil {
		return err
	}
	resolvedDSN, databaseName, err := resolveWritableDbtoolDSN(*dsn, *allowNonTest)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), commandTimeout)
	defer cancel()
	if err := syncYellowTurbanConfig(ctx, resolvedDSN, content, *updatedBy); err != nil {
		return err
	}
	fmt.Printf("黄巾起义配置已同步：数据库 %s，千帐营等级 %d，满级口粮 %d\n", databaseName, len(cfg.ThousandTentCamp.CapacityByLevel), cfg.ThousandTentCamp.CapacityByLevel[len(cfg.ThousandTentCamp.CapacityByLevel)-1])
	return nil
}

// syncYellowTurbanConfig 写入黄巾配置快照。
func syncYellowTurbanConfig(ctx context.Context, dsn string, content []byte, updatedBy string) error {
	db, err := storage.OpenMySQL(ctx, dsn)
	if err != nil {
		return err
	}
	defer db.Close()
	now := time.Now().UTC()
	_, err = db.ExecContext(ctx, `
		INSERT INTO game_configs (config_key, value_json, version, updated_by, created_at, updated_at)
		VALUES (?, ?, 1, ?, ?, ?)
		ON DUPLICATE KEY UPDATE
			value_json = VALUES(value_json),
			version = version + 1,
			updated_by = VALUES(updated_by),
			updated_at = VALUES(updated_at)
	`, game.YellowTurbanModuleID, string(content), updatedBy, now, now)
	return err
}
