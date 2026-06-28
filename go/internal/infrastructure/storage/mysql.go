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
		`CREATE TABLE IF NOT EXISTS player_reinforcements (
			id BIGINT NOT NULL AUTO_INCREMENT,
			reinforcement_id VARCHAR(64) NOT NULL,
			from_player_id VARCHAR(64) NOT NULL,
			to_player_id VARCHAR(64) NOT NULL,
			owner_player_id VARCHAR(64) NOT NULL DEFAULT '',
			host_player_id VARCHAR(64) NOT NULL DEFAULT '',
			source_type VARCHAR(64) NOT NULL DEFAULT 'reinforcement',
			source_id VARCHAR(128) NOT NULL DEFAULT '',
			source_faction VARCHAR(32) NOT NULL DEFAULT '',
			target_type VARCHAR(64) NOT NULL,
			target_id VARCHAR(64) NOT NULL,
			status VARCHAR(32) NOT NULL,
			troops_json JSON NOT NULL,
			remaining_troops_json JSON NOT NULL,
			generals_json JSON NULL,
			losses_json JSON NULL,
			buff_snapshot_json JSON NULL,
			rules_json JSON NULL,
			speed_multiplier DECIMAL(10,4) NOT NULL DEFAULT 1,
			march_seconds INT NOT NULL DEFAULT 10800,
			return_seconds INT NOT NULL DEFAULT 10800,
			sent_at DATETIME(6) NOT NULL,
			arrived_at DATETIME(6) NULL,
			recalled_at DATETIME(6) NULL,
			expelled_at DATETIME(6) NULL,
			return_started_at DATETIME(6) NULL,
			returned_at DATETIME(6) NULL,
			last_battle_report_id VARCHAR(64) NULL,
			last_battle_at DATETIME(6) NULL,
			is_annihilated BOOLEAN NOT NULL DEFAULT FALSE,
			reward_state_json JSON NULL,
			mail_state_json JSON NULL,
			metadata_json JSON NULL,
			created_at DATETIME(6) NOT NULL,
			updated_at DATETIME(6) NOT NULL,
			PRIMARY KEY (id),
			UNIQUE KEY uk_reinforcement_id (reinforcement_id),
			KEY idx_reinforcements_from_status (from_player_id, status),
			KEY idx_reinforcements_to_status (to_player_id, status),
			KEY idx_reinforcements_owner_status (owner_player_id, status),
			KEY idx_reinforcements_host_status (host_player_id, status),
			KEY idx_reinforcements_source (source_type, source_id),
			KEY idx_reinforcements_target_status (target_type, target_id, status),
			KEY idx_reinforcements_arrive (status, sent_at),
			KEY idx_reinforcements_return (status, return_started_at),
			KEY idx_reinforcements_last_battle_report (last_battle_report_id),
			CONSTRAINT fk_reinforcements_from_player
				FOREIGN KEY (from_player_id) REFERENCES players(id)
				ON DELETE CASCADE,
			CONSTRAINT fk_reinforcements_to_player
				FOREIGN KEY (to_player_id) REFERENCES players(id)
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
		`CREATE TABLE IF NOT EXISTS announcements (
			id VARCHAR(64) PRIMARY KEY,
			title VARCHAR(160) NOT NULL,
			summary VARCHAR(255) NOT NULL DEFAULT '',
			content TEXT NOT NULL,
			type VARCHAR(32) NOT NULL,
			status VARCHAR(32) NOT NULL,
			display_mode VARCHAR(32) NOT NULL DEFAULT 'center_only',
			pinned BOOLEAN NOT NULL DEFAULT FALSE,
			priority INT NOT NULL DEFAULT 0,
			force_popup BOOLEAN NOT NULL DEFAULT FALSE,
			starts_at DATETIME(6) NULL,
			ends_at DATETIME(6) NULL,
			published_at DATETIME(6) NULL,
			withdrawn_at DATETIME(6) NULL,
			archived_at DATETIME(6) NULL,
			created_at DATETIME(6) NOT NULL,
			updated_at DATETIME(6) NOT NULL,
			INDEX idx_announcements_visible (status, starts_at, ends_at, pinned, priority, published_at),
			INDEX idx_announcements_admin (updated_at),
			INDEX idx_announcements_type (type, status, published_at)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,
		`CREATE TABLE IF NOT EXISTS announcement_targets (
			announcement_id VARCHAR(64) NOT NULL,
			target_type VARCHAR(32) NOT NULL,
			target_value_json JSON NULL,
			created_at DATETIME(6) NOT NULL,
			updated_at DATETIME(6) NOT NULL,
			PRIMARY KEY (announcement_id, target_type),
			INDEX idx_announcement_targets_type (target_type),
			CONSTRAINT fk_announcement_targets_announcement
				FOREIGN KEY (announcement_id) REFERENCES announcements(id)
				ON DELETE CASCADE
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,
		`CREATE TABLE IF NOT EXISTS announcement_reads (
			announcement_id VARCHAR(64) NOT NULL,
			player_id VARCHAR(64) NOT NULL,
			account_id VARCHAR(64) NOT NULL DEFAULT '',
			is_read BOOLEAN NOT NULL DEFAULT FALSE,
			read_at DATETIME(6) NULL,
			is_popup_shown BOOLEAN NOT NULL DEFAULT FALSE,
			popup_shown_at DATETIME(6) NULL,
			is_dismissed BOOLEAN NOT NULL DEFAULT FALSE,
			dismissed_at DATETIME(6) NULL,
			created_at DATETIME(6) NOT NULL,
			updated_at DATETIME(6) NOT NULL,
			PRIMARY KEY (announcement_id, player_id),
			INDEX idx_announcement_reads_player (player_id, updated_at),
			INDEX idx_announcement_reads_account (account_id, updated_at),
			CONSTRAINT fk_announcement_reads_announcement
				FOREIGN KEY (announcement_id) REFERENCES announcements(id)
				ON DELETE CASCADE,
			CONSTRAINT fk_announcement_reads_player
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
			event_id VARCHAR(64) NOT NULL DEFAULT '',
			owner_player_id VARCHAR(64) NOT NULL DEFAULT '',
			view_type VARCHAR(32) NOT NULL DEFAULT 'attack',
			source_type VARCHAR(32) NOT NULL DEFAULT 'npc_city',
			battle_type VARCHAR(32) NOT NULL DEFAULT 'attack',
			result VARCHAR(32) NOT NULL DEFAULT '',
			title VARCHAR(128) NOT NULL DEFAULT '',
			summary VARCHAR(512) NOT NULL DEFAULT '',
			target_type VARCHAR(32) NOT NULL DEFAULT '',
			target_id VARCHAR(64) NOT NULL DEFAULT '',
			target_name VARCHAR(128) NOT NULL DEFAULT '',
			detail_json JSON NULL,
			report_json JSON NOT NULL,
			type VARCHAR(32) NOT NULL DEFAULT 'attack',
			is_read TINYINT(1) NOT NULL DEFAULT 0,
			deleted_by_player TINYINT(1) NOT NULL DEFAULT 0,
			created_at DATETIME(6) NOT NULL,
			INDEX idx_reports_player (player_id, deleted_by_player, created_at DESC),
			INDEX idx_battle_reports_owner (owner_player_id, view_type, created_at),
			INDEX idx_battle_reports_event (event_id),
			INDEX idx_battle_reports_source (source_type, target_id, created_at),
			INDEX idx_battle_reports_type (source_type, view_type, battle_type, created_at)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,
		`CREATE TABLE IF NOT EXISTS battle_events (
			id VARCHAR(64) PRIMARY KEY,
			source_type VARCHAR(32) NOT NULL,
			source_id VARCHAR(64) NOT NULL DEFAULT '',
			scene VARCHAR(32) NOT NULL DEFAULT '',
			battle_type VARCHAR(32) NOT NULL,
			result VARCHAR(32) NOT NULL,
			attacker_player_id VARCHAR(64) NOT NULL DEFAULT '',
			defender_player_id VARCHAR(64) NOT NULL DEFAULT '',
			attacker_name VARCHAR(64) NOT NULL DEFAULT '',
			defender_name VARCHAR(64) NOT NULL DEFAULT '',
			attacker_faction VARCHAR(32) NOT NULL DEFAULT '',
			defender_faction VARCHAR(32) NOT NULL DEFAULT '',
			related_march_id VARCHAR(64) NOT NULL DEFAULT '',
			related_reinforcement_id VARCHAR(64) NOT NULL DEFAULT '',
			summary_json JSON NULL,
			snapshot_json JSON NULL,
			result_json JSON NULL,
			occurred_at DATETIME(6) NOT NULL,
			created_at DATETIME(6) NOT NULL,
			INDEX idx_battle_events_source (source_type, source_id),
			INDEX idx_battle_events_players (attacker_player_id, defender_player_id, occurred_at),
			INDEX idx_battle_events_march (related_march_id),
			INDEX idx_battle_events_reinforcement (related_reinforcement_id),
			INDEX idx_battle_events_occurred (occurred_at)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,
		`CREATE TABLE IF NOT EXISTS battle_report_states (
			id VARCHAR(64) PRIMARY KEY,
			report_id VARCHAR(64) NOT NULL,
			player_id VARCHAR(64) NOT NULL,
			is_read TINYINT(1) NOT NULL DEFAULT 0,
			read_at DATETIME(6) NULL,
			is_deleted TINYINT(1) NOT NULL DEFAULT 0,
			deleted_at DATETIME(6) NULL,
			is_pinned TINYINT(1) NOT NULL DEFAULT 0,
			pinned_at DATETIME(6) NULL,
			created_at DATETIME(6) NOT NULL,
			updated_at DATETIME(6) NOT NULL,
			UNIQUE KEY uniq_report_state_player (report_id, player_id),
			INDEX idx_report_states_player (player_id, is_deleted, is_read, updated_at)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,
		`CREATE TABLE IF NOT EXISTS battle_report_participants (
			id VARCHAR(64) PRIMARY KEY,
			event_id VARCHAR(64) NOT NULL,
			report_id VARCHAR(64) NOT NULL DEFAULT '',
			player_id VARCHAR(64) NOT NULL DEFAULT '',
			role VARCHAR(32) NOT NULL,
			faction VARCHAR(32) NOT NULL DEFAULT '',
			nickname VARCHAR(64) NOT NULL DEFAULT '',
			city_name VARCHAR(64) NOT NULL DEFAULT '',
			troops_before_json JSON NULL,
			troops_lost_json JSON NULL,
			troops_survived_json JSON NULL,
			generals_json JSON NULL,
			rewards_json JSON NULL,
			points_delta_json JSON NULL,
			created_at DATETIME(6) NOT NULL,
			INDEX idx_battle_report_participants_event (event_id),
			INDEX idx_battle_report_participants_report (report_id),
			INDEX idx_battle_report_participants_player (player_id, created_at)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,
		`CREATE TABLE IF NOT EXISTS battle_report_links (
			id VARCHAR(64) PRIMARY KEY,
			report_id VARCHAR(64) NOT NULL,
			token VARCHAR(96) NOT NULL,
			visibility VARCHAR(32) NOT NULL,
			expires_at DATETIME(6) NULL,
			created_at DATETIME(6) NOT NULL,
			UNIQUE KEY uniq_battle_report_token (token)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,
		`CREATE TABLE IF NOT EXISTS pvp_marches (
			id VARCHAR(64) PRIMARY KEY,
			attacker_player_id VARCHAR(64) NOT NULL,
			attacker_name VARCHAR(64) NOT NULL DEFAULT '',
			attacker_faction VARCHAR(32) NOT NULL DEFAULT '',
			defender_player_id VARCHAR(64) NOT NULL,
			defender_name VARCHAR(64) NOT NULL DEFAULT '',
			defender_faction VARCHAR(32) NOT NULL DEFAULT '',
			march_type VARCHAR(32) NOT NULL,
			status VARCHAR(32) NOT NULL,
			attack_troops_json JSON NOT NULL,
			attack_generals_json JSON NULL,
			speed_multiplier DECIMAL(10,4) NOT NULL DEFAULT 1,
			duration_seconds INT NOT NULL,
			started_at DATETIME(6) NOT NULL,
			arrives_at DATETIME(6) NOT NULL,
			return_started_at DATETIME(6) NULL,
			returns_at DATETIME(6) NULL,
			resolved_at DATETIME(6) NULL,
			attacker_report_id VARCHAR(64) NOT NULL DEFAULT '',
			defender_report_id VARCHAR(64) NOT NULL DEFAULT '',
			battle_id VARCHAR(64) NOT NULL DEFAULT '',
			accelerated_times INT NOT NULL DEFAULT 0,
			created_at DATETIME(6) NOT NULL,
			updated_at DATETIME(6) NOT NULL,
			INDEX idx_pvp_marches_attacker (attacker_player_id, status, arrives_at),
			INDEX idx_pvp_marches_defender (defender_player_id, status, arrives_at),
			INDEX idx_pvp_marches_due (status, arrives_at)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,
		`CREATE TABLE IF NOT EXISTS pvp_battles (
			id VARCHAR(64) PRIMARY KEY,
			march_id VARCHAR(64) NOT NULL,
			attacker_player_id VARCHAR(64) NOT NULL,
			defender_player_id VARCHAR(64) NOT NULL,
			status VARCHAR(32) NOT NULL,
			attacker_snapshot_json JSON NULL,
			defender_snapshot_json JSON NULL,
			reinforcement_snapshot_json JSON NULL,
			result_json JSON NULL,
			losses_json JSON NULL,
			plunder_json JSON NULL,
			attacker_report_id VARCHAR(64) NOT NULL DEFAULT '',
			defender_report_id VARCHAR(64) NOT NULL DEFAULT '',
			resolved_at DATETIME(6) NULL,
			created_at DATETIME(6) NOT NULL,
			updated_at DATETIME(6) NOT NULL,
			UNIQUE KEY uk_pvp_battles_march (march_id),
			INDEX idx_pvp_battles_attacker (attacker_player_id, created_at),
			INDEX idx_pvp_battles_defender (defender_player_id, created_at),
			INDEX idx_pvp_battles_status (status, created_at)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,
		`CREATE TABLE IF NOT EXISTS pvp_player_states (
			player_id VARCHAR(64) PRIMARY KEY,
			status VARCHAR(32) NOT NULL DEFAULT 'normal',
			protection_type VARCHAR(32) NOT NULL DEFAULT '',
			protected_until DATETIME(6) NULL,
			cooldown_until DATETIME(6) NULL,
			daily_attack_count INT NOT NULL DEFAULT 0,
			daily_attack_limit INT NOT NULL DEFAULT 0,
			daily_reset_at DATETIME(6) NULL,
			target_cooldown_json JSON NULL,
			metadata_json JSON NULL,
			created_at DATETIME(6) NOT NULL,
			updated_at DATETIME(6) NOT NULL
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,
		`CREATE TABLE IF NOT EXISTS pvp_seasons (
			id VARCHAR(64) PRIMARY KEY,
			name VARCHAR(120) NOT NULL,
			status VARCHAR(32) NOT NULL,
			starts_at DATETIME(6) NOT NULL,
			ends_at DATETIME(6) NOT NULL,
			settled_at DATETIME(6) NULL,
			rules_json JSON NULL,
			rewards_json JSON NULL,
			created_at DATETIME(6) NOT NULL,
			updated_at DATETIME(6) NOT NULL,
			INDEX idx_pvp_seasons_status (status, starts_at, ends_at)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,
		`CREATE TABLE IF NOT EXISTS pvp_season_players (
			season_id VARCHAR(64) NOT NULL,
			player_id VARCHAR(64) NOT NULL,
			nickname VARCHAR(64) NOT NULL DEFAULT '',
			faction VARCHAR(32) NOT NULL DEFAULT '',
			` + "`rank`" + ` INT NOT NULL DEFAULT 0,
			points BIGINT NOT NULL DEFAULT 0,
			rating BIGINT NOT NULL DEFAULT 0,
			wins INT NOT NULL DEFAULT 0,
			losses INT NOT NULL DEFAULT 0,
			defense_wins INT NOT NULL DEFAULT 0,
			defense_losses INT NOT NULL DEFAULT 0,
			last_battle_at DATETIME(6) NULL,
			reward_mail_id VARCHAR(64) NOT NULL DEFAULT '',
			reward_sent_at DATETIME(6) NULL,
			created_at DATETIME(6) NOT NULL,
			updated_at DATETIME(6) NOT NULL,
			PRIMARY KEY (season_id, player_id),
			INDEX idx_pvp_season_points (season_id, points),
			INDEX idx_pvp_season_rating (season_id, rating),
			INDEX idx_pvp_season_player (player_id, updated_at)
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
	if err := addColumnIfMissing(ctx, db, `ALTER TABLE battle_reports ADD COLUMN event_id VARCHAR(64) NOT NULL DEFAULT '' AFTER player_id`); err != nil {
		return err
	}
	if err := addColumnIfMissing(ctx, db, `ALTER TABLE battle_reports ADD COLUMN owner_player_id VARCHAR(64) NOT NULL DEFAULT '' AFTER event_id`); err != nil {
		return err
	}
	if err := addColumnIfMissing(ctx, db, `ALTER TABLE battle_reports ADD COLUMN view_type VARCHAR(32) NOT NULL DEFAULT 'attack' AFTER owner_player_id`); err != nil {
		return err
	}
	if err := addColumnIfMissing(ctx, db, `ALTER TABLE battle_reports ADD COLUMN source_type VARCHAR(32) NOT NULL DEFAULT 'npc_city' AFTER view_type`); err != nil {
		return err
	}
	if err := addColumnIfMissing(ctx, db, `ALTER TABLE battle_reports ADD COLUMN battle_type VARCHAR(32) NOT NULL DEFAULT 'attack' AFTER source_type`); err != nil {
		return err
	}
	if err := addColumnIfMissing(ctx, db, `ALTER TABLE battle_reports ADD COLUMN result VARCHAR(32) NOT NULL DEFAULT '' AFTER battle_type`); err != nil {
		return err
	}
	if err := addColumnIfMissing(ctx, db, `ALTER TABLE battle_reports ADD COLUMN title VARCHAR(128) NOT NULL DEFAULT '' AFTER result`); err != nil {
		return err
	}
	if err := addColumnIfMissing(ctx, db, `ALTER TABLE battle_reports ADD COLUMN summary VARCHAR(512) NOT NULL DEFAULT '' AFTER title`); err != nil {
		return err
	}
	if err := addColumnIfMissing(ctx, db, `ALTER TABLE battle_reports ADD COLUMN target_type VARCHAR(32) NOT NULL DEFAULT '' AFTER summary`); err != nil {
		return err
	}
	if err := addColumnIfMissing(ctx, db, `ALTER TABLE battle_reports ADD COLUMN target_id VARCHAR(64) NOT NULL DEFAULT '' AFTER target_type`); err != nil {
		return err
	}
	if err := addColumnIfMissing(ctx, db, `ALTER TABLE battle_reports ADD COLUMN target_name VARCHAR(128) NOT NULL DEFAULT '' AFTER target_id`); err != nil {
		return err
	}
	if err := addColumnIfMissing(ctx, db, `ALTER TABLE battle_reports ADD COLUMN detail_json JSON NULL AFTER target_name`); err != nil {
		return err
	}
	if _, err := db.ExecContext(ctx, `UPDATE battle_reports SET owner_player_id = player_id WHERE owner_player_id = ''`); err != nil {
		return err
	}
	if _, err := db.ExecContext(ctx, `UPDATE battle_reports SET view_type = type WHERE view_type = '' OR view_type = 'attack'`); err != nil {
		return err
	}
	if _, err := db.ExecContext(ctx, `CREATE INDEX idx_battle_reports_owner ON battle_reports (owner_player_id, view_type, created_at)`); err != nil && !isDuplicateKeyName(err) {
		return err
	}
	if _, err := db.ExecContext(ctx, `CREATE INDEX idx_battle_reports_event ON battle_reports (event_id)`); err != nil && !isDuplicateKeyName(err) {
		return err
	}
	if _, err := db.ExecContext(ctx, `CREATE INDEX idx_battle_reports_source ON battle_reports (source_type, target_id, created_at)`); err != nil && !isDuplicateKeyName(err) {
		return err
	}
	if _, err := db.ExecContext(ctx, `CREATE INDEX idx_battle_reports_type ON battle_reports (source_type, view_type, battle_type, created_at)`); err != nil && !isDuplicateKeyName(err) {
		return err
	}
	if err := addColumnIfMissing(ctx, db, `ALTER TABLE player_reinforcements ADD COLUMN owner_player_id VARCHAR(64) NOT NULL DEFAULT '' AFTER to_player_id`); err != nil {
		return err
	}
	if err := addColumnIfMissing(ctx, db, `ALTER TABLE player_reinforcements ADD COLUMN host_player_id VARCHAR(64) NOT NULL DEFAULT '' AFTER owner_player_id`); err != nil {
		return err
	}
	if err := addColumnIfMissing(ctx, db, `ALTER TABLE player_reinforcements ADD COLUMN source_type VARCHAR(64) NOT NULL DEFAULT 'reinforcement' AFTER host_player_id`); err != nil {
		return err
	}
	if err := addColumnIfMissing(ctx, db, `ALTER TABLE player_reinforcements ADD COLUMN source_id VARCHAR(128) NOT NULL DEFAULT '' AFTER source_type`); err != nil {
		return err
	}
	if err := addColumnIfMissing(ctx, db, `ALTER TABLE player_reinforcements ADD COLUMN source_faction VARCHAR(32) NOT NULL DEFAULT '' AFTER source_id`); err != nil {
		return err
	}
	if err := addColumnIfMissing(ctx, db, `ALTER TABLE player_reinforcements ADD COLUMN rules_json JSON NULL AFTER buff_snapshot_json`); err != nil {
		return err
	}
	if _, err := db.ExecContext(ctx, `UPDATE player_reinforcements SET owner_player_id = from_player_id WHERE owner_player_id = ''`); err != nil {
		return err
	}
	if _, err := db.ExecContext(ctx, `UPDATE player_reinforcements SET host_player_id = to_player_id WHERE host_player_id = ''`); err != nil {
		return err
	}
	if _, err := db.ExecContext(ctx, `UPDATE player_reinforcements SET source_id = reinforcement_id WHERE source_id = ''`); err != nil {
		return err
	}
	if _, err := db.ExecContext(ctx, `CREATE INDEX idx_reinforcements_owner_status ON player_reinforcements (owner_player_id, status)`); err != nil && !isDuplicateKeyName(err) {
		return err
	}
	if _, err := db.ExecContext(ctx, `CREATE INDEX idx_reinforcements_host_status ON player_reinforcements (host_player_id, status)`); err != nil && !isDuplicateKeyName(err) {
		return err
	}
	if _, err := db.ExecContext(ctx, `CREATE INDEX idx_reinforcements_source ON player_reinforcements (source_type, source_id)`); err != nil && !isDuplicateKeyName(err) {
		return err
	}
	if err := addColumnIfMissing(ctx, db, `ALTER TABLE pvp_marches ADD COLUMN accelerated_times INT NOT NULL DEFAULT 0 AFTER battle_id`); err != nil {
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
