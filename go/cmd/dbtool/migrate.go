// 本文件归口数据库迁移和测试库 DSN 命令。
package main

import (
	"context"
	"flag"
	"fmt"
	"time"

	"github.com/go-sql-driver/mysql"

	"hero3/internal/infrastructure/storage"
)

const (
	commandTimeout       = 30 * time.Second
	migrationTimeout     = 5 * time.Minute
	migrationReadTimeout = 60 * time.Second
	migrationConnTimeout = 30 * time.Second
)

// runMigrate 迁移当前 HERO3_DATABASE_DSN 指向的数据库。
func runMigrate(args []string) error {
	flags := flag.NewFlagSet("migrate", flag.ContinueOnError)
	if err := flags.Parse(args); err != nil {
		return err
	}
	dsn, err := configuredDSN()
	if err != nil {
		return err
	}
	return migrateDSN(dsn)
}

// runMigrateYellowTurban 只迁移黄巾起义队列表。
func runMigrateYellowTurban(args []string) error {
	flags := flag.NewFlagSet("migrate-yellow-turban", flag.ContinueOnError)
	if err := flags.Parse(args); err != nil {
		return err
	}
	dsn, err := configuredDSN()
	if err != nil {
		return err
	}
	return migrateYellowTurbanDSN(dsn)
}

// runCreateTestDB 创建当前库对应的 test 前缀数据库。
func runCreateTestDB(args []string) error {
	flags := flag.NewFlagSet("create-test-db", flag.ContinueOnError)
	if err := flags.Parse(args); err != nil {
		return err
	}
	dsn, err := configuredDSN()
	if err != nil {
		return err
	}
	testDatabaseName, _, err := storage.MySQLTestDatabaseDSN(dsn)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), commandTimeout)
	defer cancel()
	if err := storage.CreateMySQLDatabaseFromDSN(ctx, dsn, testDatabaseName); err != nil {
		return err
	}
	fmt.Printf("已创建或确认测试库：%s\n", testDatabaseName)
	return nil
}

// runMigrateTest 创建并迁移当前库对应的 test 前缀数据库。
func runMigrateTest(args []string) error {
	flags := flag.NewFlagSet("migrate-test", flag.ContinueOnError)
	if err := flags.Parse(args); err != nil {
		return err
	}
	dsn, err := configuredDSN()
	if err != nil {
		return err
	}
	testDatabaseName, testDSN, err := storage.MySQLTestDatabaseDSN(dsn)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), commandTimeout)
	defer cancel()
	if err := storage.CreateMySQLDatabaseFromDSN(ctx, dsn, testDatabaseName); err != nil {
		return err
	}
	return migrateDSN(testDSN)
}

// runPrintTestDSN 输出当前 DSN 对应的 test 前缀库 DSN。
func runPrintTestDSN(args []string) error {
	flags := flag.NewFlagSet("print-test-dsn", flag.ContinueOnError)
	redact := flags.Bool("redact", true, "是否隐藏密码")
	if err := flags.Parse(args); err != nil {
		return err
	}
	dsn, err := configuredDSN()
	if err != nil {
		return err
	}
	_, testDSN, err := storage.MySQLTestDatabaseDSN(dsn)
	if err != nil {
		return err
	}
	if *redact {
		testDSN = storage.RedactMySQLDSN(testDSN)
	}
	fmt.Println(testDSN)
	return nil
}

// migrateDSN 对指定 DSN 执行迁移。
func migrateDSN(dsn string) error {
	ctx, cancel := context.WithTimeout(context.Background(), migrationTimeout)
	defer cancel()
	migratorDSN, err := normalizeMigrationDSN(dsn)
	if err != nil {
		return err
	}
	db, err := storage.OpenMySQL(ctx, migratorDSN)
	if err != nil {
		return err
	}
	defer db.Close()
	if err := storage.MigrateMySQL(ctx, db); err != nil {
		return err
	}
	databaseName, err := storage.MySQLDatabaseName(dsn)
	if err != nil {
		return err
	}
	fmt.Printf("数据库迁移完成：%s\n", databaseName)
	return nil
}

// migrateYellowTurbanDSN 对指定 DSN 执行黄巾起义小迁移。
func migrateYellowTurbanDSN(dsn string) error {
	ctx, cancel := context.WithTimeout(context.Background(), migrationTimeout)
	defer cancel()
	migratorDSN, err := normalizeMigrationDSN(dsn)
	if err != nil {
		return err
	}
	db, err := storage.OpenMySQL(ctx, migratorDSN)
	if err != nil {
		return err
	}
	defer db.Close()
	if err := storage.MigrateMySQLYellowTurban(ctx, db); err != nil {
		return err
	}
	databaseName, err := storage.MySQLDatabaseName(dsn)
	if err != nil {
		return err
	}
	fmt.Printf("黄巾起义数据库迁移完成：%s\n", databaseName)
	return nil
}

// normalizeMigrationDSN 为结构迁移提高 MySQL 连接读写超时，避免低峰 DDL 因服务默认短超时失败。
func normalizeMigrationDSN(dsn string) (string, error) {
	cfg, err := mysql.ParseDSN(dsn)
	if err != nil {
		return "", err
	}
	if cfg.Timeout < migrationConnTimeout {
		cfg.Timeout = migrationConnTimeout
	}
	if cfg.ReadTimeout < migrationReadTimeout {
		cfg.ReadTimeout = migrationReadTimeout
	}
	if cfg.WriteTimeout < migrationReadTimeout {
		cfg.WriteTimeout = migrationReadTimeout
	}
	return cfg.FormatDSN(), nil
}
