// Package storage 提供 Hero3 的 MySQL 持久化实现。
package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/go-sql-driver/mysql"
)

type MySQLRepository struct {
	db *sql.DB
}

const (
	mysqlSchemaMigrationID          = "2026-06-26-core-schema"
	mysqlSchemaMigrationDescription = "core schema bootstrap and compatibility migrations"
)

// NewMySQLRepository 创建 MySQL 仓储实例。
func NewMySQLRepository(db *sql.DB) *MySQLRepository {
	return &MySQLRepository{db: db}
}

// OpenMySQL 打开 MySQL 连接池，并为连接、读写操作补齐默认超时。
func OpenMySQL(ctx context.Context, dsn string) (*sql.DB, error) {
	normalizedDSN, err := withMySQLTimeouts(dsn)
	if err != nil {
		return nil, err
	}

	db, err := sql.Open("mysql", normalizedDSN)
	if err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(20)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(30 * time.Minute)
	db.SetConnMaxIdleTime(5 * time.Minute)

	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}

	return db, nil
}

// withMySQLTimeouts 为旧 DSN 自动补上默认超时，避免数据库半断开时请求长时间挂起。
func withMySQLTimeouts(dsn string) (string, error) {
	cfg, err := mysql.ParseDSN(dsn)
	if err != nil {
		return "", err
	}

	if cfg.Timeout == 0 {
		cfg.Timeout = 5 * time.Second
	}
	if cfg.ReadTimeout == 0 {
		cfg.ReadTimeout = 5 * time.Second
	}
	if cfg.WriteTimeout == 0 {
		cfg.WriteTimeout = 5 * time.Second
	}

	return cfg.FormatDSN(), nil
}

// MySQLDatabaseName 从 DSN 中解析当前数据库名。
func MySQLDatabaseName(dsn string) (string, error) {
	cfg, err := mysql.ParseDSN(dsn)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(cfg.DBName) == "" {
		return "", errors.New("mysql database name is required")
	}
	return cfg.DBName, nil
}

// MySQLDSNWithDatabase 返回替换数据库名后的 DSN。
func MySQLDSNWithDatabase(dsn string, databaseName string) (string, error) {
	cfg, err := mysql.ParseDSN(dsn)
	if err != nil {
		return "", err
	}
	databaseName = strings.TrimSpace(databaseName)
	if databaseName == "" {
		return "", errors.New("mysql database name is required")
	}
	cfg.DBName = databaseName
	return cfg.FormatDSN(), nil
}

// MySQLTestDatabaseName 根据当前库名生成 test 前缀库名。
func MySQLTestDatabaseName(databaseName string) (string, error) {
	databaseName = strings.TrimSpace(databaseName)
	if databaseName == "" {
		return "", errors.New("mysql database name is required")
	}
	if strings.HasPrefix(databaseName, "test_") {
		return databaseName, nil
	}
	return "test_" + databaseName, nil
}

// MySQLTestDatabaseDSN 根据当前 DSN 生成 test 前缀库 DSN。
func MySQLTestDatabaseDSN(dsn string) (string, string, error) {
	databaseName, err := MySQLDatabaseName(dsn)
	if err != nil {
		return "", "", err
	}
	testDatabaseName, err := MySQLTestDatabaseName(databaseName)
	if err != nil {
		return "", "", err
	}
	testDSN, err := MySQLDSNWithDatabase(dsn, testDatabaseName)
	if err != nil {
		return "", "", err
	}
	return testDatabaseName, testDSN, nil
}

// RedactMySQLDSN 隐藏 DSN 密码，便于日志输出。
func RedactMySQLDSN(dsn string) string {
	cfg, err := mysql.ParseDSN(dsn)
	if err != nil {
		return "<invalid mysql dsn>"
	}
	if cfg.Passwd != "" {
		cfg.Passwd = "******"
	}
	return cfg.FormatDSN()
}

// CreateMySQLDatabaseFromDSN 使用 DSN 的账号创建指定数据库。
func CreateMySQLDatabaseFromDSN(ctx context.Context, dsn string, databaseName string) error {
	cfg, err := mysql.ParseDSN(dsn)
	if err != nil {
		return err
	}
	databaseName = strings.TrimSpace(databaseName)
	if databaseName == "" {
		return errors.New("mysql database name is required")
	}
	cfg.DBName = ""
	serverDSN, err := withMySQLTimeouts(cfg.FormatDSN())
	if err != nil {
		return err
	}
	db, err := sql.Open("mysql", serverDSN)
	if err != nil {
		return err
	}
	defer db.Close()
	if err := db.PingContext(ctx); err != nil {
		return err
	}
	query := fmt.Sprintf(
		"CREATE DATABASE IF NOT EXISTS `%s` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci",
		escapeMySQLIdentifier(databaseName),
	)
	_, err = db.ExecContext(ctx, query)
	return err
}

// escapeMySQLIdentifier 转义 MySQL 标识符中的反引号。
func escapeMySQLIdentifier(value string) string {
	return strings.ReplaceAll(value, "`", "``")
}

func isDuplicateEntry(err error) bool {
	var mysqlErr *mysql.MySQLError
	return errors.As(err, &mysqlErr) && mysqlErr.Number == 1062
}

func addColumnIfMissing(ctx context.Context, db *sql.DB, statement string) error {
	_, err := db.ExecContext(ctx, statement)
	if err == nil || isDuplicateColumn(err) {
		return nil
	}
	return err
}

func isDuplicateColumn(err error) bool {
	var mysqlErr *mysql.MySQLError
	return errors.As(err, &mysqlErr) && mysqlErr.Number == 1060
}

func isDuplicateKeyName(err error) bool {
	var mysqlErr *mysql.MySQLError
	return errors.As(err, &mysqlErr) && mysqlErr.Number == 1061
}

// recordMySQLMigration 记录当前轻量迁移已经应用，便于后续排查库结构来源。
func recordMySQLMigration(ctx context.Context, db *sql.DB, migrationID string, description string) error {
	_, err := db.ExecContext(ctx, `
		INSERT INTO schema_migrations (id, description, applied_at)
		VALUES (?, ?, UTC_TIMESTAMP(6))
		ON DUPLICATE KEY UPDATE description = VALUES(description)
	`, migrationID, description)
	return err
}

// MigrateMySQL 执行 MySQL 表结构初始化和轻量兼容迁移。
func MigrateMySQL(ctx context.Context, db *sql.DB) error {
	statements := []string{
		`CREATE TABLE IF NOT EXISTS schema_migrations (
			id VARCHAR(128) PRIMARY KEY,
			description VARCHAR(255) NOT NULL DEFAULT '',
			applied_at DATETIME(6) NOT NULL
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,
		`CREATE TABLE IF NOT EXISTS accounts (
			id VARCHAR(64) PRIMARY KEY,
			username VARCHAR(64) NOT NULL UNIQUE,
			password_hash CHAR(64) NOT NULL,
			gold INT NOT NULL DEFAULT 0,
			created_at DATETIME(6) NOT NULL
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,
		`CREATE TABLE IF NOT EXISTS players (
			id VARCHAR(64) PRIMARY KEY,
			account_id VARCHAR(64) NOT NULL,
			nickname VARCHAR(64) NOT NULL,
			faction VARCHAR(32) NOT NULL,
			mail_code VARCHAR(12) NOT NULL DEFAULT '',
			state_json JSON NOT NULL,
			created_at DATETIME(6) NOT NULL,
			updated_at DATETIME(6) NOT NULL,
			INDEX idx_players_account_updated (account_id, updated_at),
			INDEX idx_players_mail_address (nickname, mail_code),
			CONSTRAINT fk_players_account
				FOREIGN KEY (account_id) REFERENCES accounts(id)
				ON DELETE CASCADE
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,
		`CREATE TABLE IF NOT EXISTS player_resources (
			player_id VARCHAR(64) NOT NULL,
			resource_type VARCHAR(64) NOT NULL,
			amount INT NOT NULL DEFAULT 0,
			capacity INT NOT NULL DEFAULT 0,
			updated_at DATETIME(6) NOT NULL,
			PRIMARY KEY (player_id, resource_type),
			INDEX idx_player_resources_type (resource_type),
			CONSTRAINT fk_player_resources_player
				FOREIGN KEY (player_id) REFERENCES players(id)
				ON DELETE CASCADE
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,
		`CREATE TABLE IF NOT EXISTS player_inventory (
			player_id VARCHAR(64) NOT NULL,
			item_id VARCHAR(64) NOT NULL,
			amount INT NOT NULL DEFAULT 0,
			obtained_at DATETIME(6) NULL,
			updated_at DATETIME(6) NULL,
			PRIMARY KEY (player_id, item_id),
			INDEX idx_player_inventory_item (item_id),
			CONSTRAINT fk_player_inventory_player
				FOREIGN KEY (player_id) REFERENCES players(id)
				ON DELETE CASCADE
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,
		`CREATE TABLE IF NOT EXISTS player_buildings (
			player_id VARCHAR(64) NOT NULL,
			building_id VARCHAR(64) NOT NULL,
			building_type VARCHAR(64) NOT NULL,
			level INT NOT NULL DEFAULT 0,
			status VARCHAR(32) NOT NULL DEFAULT '',
			upgrade_ends_at DATETIME(6) NULL,
			status_ends_at DATETIME(6) NULL,
			updated_at DATETIME(6) NOT NULL,
			PRIMARY KEY (player_id, building_id),
			INDEX idx_player_buildings_type (building_type),
			CONSTRAINT fk_player_buildings_player
				FOREIGN KEY (player_id) REFERENCES players(id)
				ON DELETE CASCADE
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,
		`CREATE TABLE IF NOT EXISTS player_resource_slots (
			player_id VARCHAR(64) NOT NULL,
			slot_id VARCHAR(64) NOT NULL,
			resource_type VARCHAR(64) NOT NULL,
			building_id VARCHAR(64) NOT NULL DEFAULT '',
			unlocked_by VARCHAR(64) NOT NULL DEFAULT '',
			unlocked_at DATETIME(6) NULL,
			updated_at DATETIME(6) NOT NULL,
			PRIMARY KEY (player_id, slot_id),
			INDEX idx_player_resource_slots_resource (resource_type),
			INDEX idx_player_resource_slots_building (building_id),
			CONSTRAINT fk_player_resource_slots_player
				FOREIGN KEY (player_id) REFERENCES players(id)
				ON DELETE CASCADE
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,
		`CREATE TABLE IF NOT EXISTS player_army_units (
			player_id VARCHAR(64) NOT NULL,
			unit_type VARCHAR(64) NOT NULL,
			amount INT NOT NULL DEFAULT 0,
			updated_at DATETIME(6) NOT NULL,
			PRIMARY KEY (player_id, unit_type),
			INDEX idx_player_army_units_type (unit_type),
			CONSTRAINT fk_player_army_units_player
				FOREIGN KEY (player_id) REFERENCES players(id)
				ON DELETE CASCADE
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,
		`CREATE TABLE IF NOT EXISTS player_recruit_queues (
			player_id VARCHAR(64) NOT NULL,
			queue_id VARCHAR(64) NOT NULL,
			unit_type VARCHAR(64) NOT NULL,
			amount INT NOT NULL DEFAULT 0,
			ends_at DATETIME(6) NOT NULL,
			queue_order INT NOT NULL DEFAULT 0,
			updated_at DATETIME(6) NOT NULL,
			PRIMARY KEY (player_id, queue_id),
			INDEX idx_player_recruit_queues_unit (unit_type),
			INDEX idx_player_recruit_queues_ends (player_id, ends_at),
			CONSTRAINT fk_player_recruit_queues_player
				FOREIGN KEY (player_id) REFERENCES players(id)
				ON DELETE CASCADE
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,
		`CREATE TABLE IF NOT EXISTS player_generals (
			player_id VARCHAR(64) NOT NULL,
			general_id VARCHAR(64) NOT NULL,
			faction VARCHAR(32) NOT NULL DEFAULT '',
			level INT NOT NULL DEFAULT 1,
			exp INT NOT NULL DEFAULT 0,
			stats_json JSON NULL,
			acquired_at DATETIME(6) NULL,
			updated_at DATETIME(6) NOT NULL,
			PRIMARY KEY (player_id, general_id),
			INDEX idx_player_generals_general (general_id),
			CONSTRAINT fk_player_generals_player
				FOREIGN KEY (player_id) REFERENCES players(id)
				ON DELETE CASCADE
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,
		`CREATE TABLE IF NOT EXISTS player_general_assignments (
			player_id VARCHAR(64) NOT NULL,
			assignment_id VARCHAR(64) NOT NULL,
			general_id VARCHAR(64) NOT NULL,
			assignment_slot VARCHAR(64) NOT NULL DEFAULT '',
			module_id VARCHAR(64) NOT NULL DEFAULT '',
			status VARCHAR(32) NOT NULL DEFAULT '',
			assigned_at DATETIME(6) NULL,
			ends_at DATETIME(6) NULL,
			updated_at DATETIME(6) NOT NULL,
			PRIMARY KEY (player_id, assignment_id),
			INDEX idx_player_general_assignments_general (player_id, general_id),
			INDEX idx_player_general_assignments_slot (player_id, assignment_slot, module_id),
			CONSTRAINT fk_player_general_assignments_player
				FOREIGN KEY (player_id) REFERENCES players(id)
				ON DELETE CASCADE
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,
		`CREATE TABLE IF NOT EXISTS player_buffs (
			player_id VARCHAR(64) NOT NULL,
			buff_id VARCHAR(64) NOT NULL,
			source VARCHAR(64) NOT NULL DEFAULT '',
			modifier_key VARCHAR(64) NOT NULL,
			modifier_value DOUBLE NOT NULL DEFAULT 0,
			modifier_mode VARCHAR(32) NOT NULL,
			expires_at DATETIME(6) NULL,
			created_at DATETIME(6) NOT NULL,
			note VARCHAR(255) NOT NULL DEFAULT '',
			updated_at DATETIME(6) NOT NULL,
			PRIMARY KEY (player_id, buff_id),
			INDEX idx_player_buffs_key (modifier_key),
			INDEX idx_player_buffs_expires (player_id, expires_at),
			CONSTRAINT fk_player_buffs_player
				FOREIGN KEY (player_id) REFERENCES players(id)
				ON DELETE CASCADE
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,
		`CREATE TABLE IF NOT EXISTS event_processing_records (
			module_id VARCHAR(64) NOT NULL,
			handler_key VARCHAR(128) NOT NULL,
			event_key CHAR(64) NOT NULL,
			processed_at DATETIME(6) NOT NULL,
			PRIMARY KEY (module_id, handler_key, event_key),
			INDEX idx_event_processing_processed (processed_at)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,
		`CREATE TABLE IF NOT EXISTS gameplay_module_instances (
			id VARCHAR(64) PRIMARY KEY,
			module_id VARCHAR(64) NOT NULL,
			status VARCHAR(32) NOT NULL DEFAULT '',
			config_json JSON NULL,
			state_json JSON NULL,
			opens_at DATETIME(6) NULL,
			closes_at DATETIME(6) NULL,
			created_at DATETIME(6) NOT NULL,
			updated_at DATETIME(6) NOT NULL,
			INDEX idx_gameplay_instances_module_status (module_id, status, opens_at),
			INDEX idx_gameplay_instances_window (opens_at, closes_at)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,
		`CREATE TABLE IF NOT EXISTS gameplay_module_participants (
			instance_id VARCHAR(64) NOT NULL,
			player_id VARCHAR(64) NOT NULL,
			status VARCHAR(32) NOT NULL DEFAULT '',
			score BIGINT NOT NULL DEFAULT 0,
			joined_at DATETIME(6) NOT NULL,
			updated_at DATETIME(6) NOT NULL,
			PRIMARY KEY (instance_id, player_id),
			INDEX idx_gameplay_participants_player (player_id, updated_at),
			INDEX idx_gameplay_participants_score (instance_id, score),
			CONSTRAINT fk_gameplay_participants_instance
				FOREIGN KEY (instance_id) REFERENCES gameplay_module_instances(id)
				ON DELETE CASCADE,
			CONSTRAINT fk_gameplay_participants_player
				FOREIGN KEY (player_id) REFERENCES players(id)
				ON DELETE CASCADE
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,
		`CREATE TABLE IF NOT EXISTS gameplay_module_settlements (
			instance_id VARCHAR(64) NOT NULL,
			player_id VARCHAR(64) NOT NULL,
			settlement_key VARCHAR(128) NOT NULL,
			rewards_json JSON NULL,
			status VARCHAR(32) NOT NULL DEFAULT '',
			settled_at DATETIME(6) NULL,
			created_at DATETIME(6) NOT NULL,
			updated_at DATETIME(6) NOT NULL,
			PRIMARY KEY (instance_id, player_id, settlement_key),
			INDEX idx_gameplay_settlements_player (player_id, updated_at),
			CONSTRAINT fk_gameplay_settlements_instance
				FOREIGN KEY (instance_id) REFERENCES gameplay_module_instances(id)
				ON DELETE CASCADE,
			CONSTRAINT fk_gameplay_settlements_player
				FOREIGN KEY (player_id) REFERENCES players(id)
				ON DELETE CASCADE
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,
		`CREATE TABLE IF NOT EXISTS battle_reports (
			id VARCHAR(64) PRIMARY KEY,
			player_id VARCHAR(64) NOT NULL,
			report_json JSON NOT NULL,
			type VARCHAR(32) NOT NULL DEFAULT 'attack',
			is_read TINYINT(1) NOT NULL DEFAULT 0,
			deleted_by_player TINYINT(1) NOT NULL DEFAULT 0,
			created_at DATETIME(6) NOT NULL,
			INDEX idx_reports_player (player_id, deleted_by_player, created_at DESC)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,
		`CREATE TABLE IF NOT EXISTS mails (
			id VARCHAR(64) PRIMARY KEY,
			player_id VARCHAR(64) NOT NULL,
			mail_type VARCHAR(32) NOT NULL,
			sender_type VARCHAR(32) NOT NULL,
			sender_id VARCHAR(64) NOT NULL DEFAULT '',
			sender_name VARCHAR(64) NOT NULL DEFAULT '',
			title VARCHAR(120) NOT NULL,
			content TEXT NOT NULL,
			attachments_json JSON NULL,
			source_type VARCHAR(32) NOT NULL DEFAULT '',
			source_id VARCHAR(64) NOT NULL DEFAULT '',
			is_read TINYINT(1) NOT NULL DEFAULT 0,
			is_claimed TINYINT(1) NOT NULL DEFAULT 0,
			deleted_by_player TINYINT(1) NOT NULL DEFAULT 0,
			expires_at DATETIME(6) NULL,
			created_at DATETIME(6) NOT NULL,
			read_at DATETIME(6) NULL,
			claimed_at DATETIME(6) NULL,
			INDEX idx_mails_player_list (player_id, deleted_by_player, created_at DESC),
			INDEX idx_mails_player_unread (player_id, deleted_by_player, is_read),
			INDEX idx_mails_source (source_type, source_id)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,
		`CREATE TABLE IF NOT EXISTS minigame_records (
			id VARCHAR(64) PRIMARY KEY,
			player_id VARCHAR(64) NOT NULL,
			game_type VARCHAR(32) NOT NULL,
			result_name VARCHAR(64) NOT NULL,
			rarity VARCHAR(32) NOT NULL,
			reward_unit VARCHAR(64) NOT NULL DEFAULT '',
			reward_amount INT NOT NULL DEFAULT 0,
			remaining_amount INT NOT NULL DEFAULT 0,
			bet_unit VARCHAR(64) NOT NULL DEFAULT '',
			bet_amount INT NOT NULL DEFAULT 0,
			created_at DATETIME(6) NOT NULL,
			INDEX idx_minigame_player (player_id, created_at DESC),
			INDEX idx_minigame_type (player_id, game_type, created_at DESC)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,
		`CREATE TABLE IF NOT EXISTS gold_ledger (
			id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
			account_id VARCHAR(64) NOT NULL DEFAULT '',
			player_id VARCHAR(64) NOT NULL DEFAULT '',
			currency VARCHAR(16) NOT NULL,
			direction VARCHAR(8) NOT NULL,
			amount INT NOT NULL,
			balance_after INT NOT NULL,
			ref_type VARCHAR(64) NOT NULL DEFAULT '',
			ref_id VARCHAR(128) NOT NULL DEFAULT '',
			reason VARCHAR(255) NOT NULL DEFAULT '',
			created_at DATETIME(6) NOT NULL,
			INDEX idx_ledger_account (account_id, created_at),
			INDEX idx_ledger_player (player_id, created_at),
			INDEX idx_ledger_ref (ref_type, ref_id),
			INDEX idx_ledger_currency (currency, created_at)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,
	}

	for _, statement := range statements {
		if _, err := db.ExecContext(ctx, statement); err != nil {
			return err
		}
	}

	if err := addColumnIfMissing(ctx, db, `ALTER TABLE accounts ADD COLUMN gold INT NOT NULL DEFAULT 0`); err != nil {
		return err
	}
	if err := addColumnIfMissing(ctx, db, `ALTER TABLE players ADD COLUMN mail_code VARCHAR(12) NOT NULL DEFAULT ''`); err != nil {
		return err
	}
	if _, err := db.ExecContext(ctx, `CREATE INDEX idx_players_mail_address ON players (nickname, mail_code)`); err != nil && !isDuplicateKeyName(err) {
		return err
	}
	if err := addColumnIfMissing(ctx, db, `ALTER TABLE minigame_records ADD COLUMN bet_unit VARCHAR(64) NOT NULL DEFAULT ''`); err != nil {
		return err
	}
	if err := addColumnIfMissing(ctx, db, `ALTER TABLE minigame_records ADD COLUMN bet_amount INT NOT NULL DEFAULT 0`); err != nil {
		return err
	}
	if err := addColumnIfMissing(ctx, db, `ALTER TABLE minigame_records ADD COLUMN remaining_amount INT NOT NULL DEFAULT -1`); err != nil {
		return err
	}
	if _, err := db.ExecContext(ctx, `UPDATE minigame_records SET remaining_amount = reward_amount WHERE remaining_amount = -1 AND game_type = 'fishing'`); err != nil {
		return err
	}
	if _, err := db.ExecContext(ctx, `UPDATE minigame_records SET remaining_amount = 0 WHERE remaining_amount = -1`); err != nil {
		return err
	}

	return recordMySQLMigration(ctx, db, mysqlSchemaMigrationID, mysqlSchemaMigrationDescription)
}
