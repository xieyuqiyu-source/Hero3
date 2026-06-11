package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/go-sql-driver/mysql"

	"hero3/internal/game"
)

type MySQLRepository struct {
	db *sql.DB
}

func NewMySQLRepository(db *sql.DB) *MySQLRepository {
	return &MySQLRepository{db: db}
}

func OpenMySQL(ctx context.Context, dsn string) (*sql.DB, error) {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(20)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(30 * time.Minute)

	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}

	return db, nil
}

func MigrateMySQL(ctx context.Context, db *sql.DB) error {
	statements := []string{
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
			state_json JSON NOT NULL,
			created_at DATETIME(6) NOT NULL,
			updated_at DATETIME(6) NOT NULL,
			INDEX idx_players_account_updated (account_id, updated_at),
			CONSTRAINT fk_players_account
				FOREIGN KEY (account_id) REFERENCES accounts(id)
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
		`CREATE TABLE IF NOT EXISTS minigame_records (
			id VARCHAR(64) PRIMARY KEY,
			player_id VARCHAR(64) NOT NULL,
			game_type VARCHAR(32) NOT NULL,
			result_name VARCHAR(64) NOT NULL,
			rarity VARCHAR(32) NOT NULL,
			reward_unit VARCHAR(64) NOT NULL DEFAULT '',
			reward_amount INT NOT NULL DEFAULT 0,
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
	if err := addColumnIfMissing(ctx, db, `ALTER TABLE minigame_records ADD COLUMN bet_unit VARCHAR(64) NOT NULL DEFAULT ''`); err != nil {
		return err
	}
	if err := addColumnIfMissing(ctx, db, `ALTER TABLE minigame_records ADD COLUMN bet_amount INT NOT NULL DEFAULT 0`); err != nil {
		return err
	}

	return nil
}

func (r *MySQLRepository) CreateAccount(account game.Account) error {
	_, err := r.db.Exec(
		`INSERT INTO accounts (id, username, password_hash, gold, created_at) VALUES (?, ?, ?, ?, ?)`,
		account.ID,
		account.Username,
		account.PasswordHash,
		account.Gold,
		account.CreatedAt.UTC(),
	)
	if isDuplicateEntry(err) {
		return game.ErrAccountExists
	}
	return err
}

func (r *MySQLRepository) GetAccountByUsername(username string) (game.Account, error) {
	var account game.Account
	err := r.db.QueryRow(
		`SELECT id, username, password_hash, gold, created_at FROM accounts WHERE username = ? LIMIT 1`,
		username,
	).Scan(&account.ID, &account.Username, &account.PasswordHash, &account.Gold, &account.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return game.Account{}, game.ErrAccountNotFound
	}
	return account, err
}

func (r *MySQLRepository) GetAccountByID(accountID string) (game.Account, error) {
	var account game.Account
	err := r.db.QueryRow(
		`SELECT id, username, password_hash, gold, created_at FROM accounts WHERE id = ? LIMIT 1`,
		accountID,
	).Scan(&account.ID, &account.Username, &account.PasswordHash, &account.Gold, &account.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return game.Account{}, game.ErrAccountNotFound
	}
	return account, err
}

func (r *MySQLRepository) UpdateAccountGold(accountID string, gold int) error {
	result, err := r.db.Exec(`UPDATE accounts SET gold = ? WHERE id = ?`, gold, accountID)
	if err != nil {
		return err
	}
	affected, _ := result.RowsAffected()
	if affected == 0 {
		return game.ErrAccountNotFound
	}
	return nil
}

func (r *MySQLRepository) AddAccountGold(accountID string, amount int) error {
	result, err := r.db.Exec(`UPDATE accounts SET gold = gold + ? WHERE id = ?`, amount, accountID)
	if err != nil {
		return err
	}
	affected, _ := result.RowsAffected()
	if affected == 0 {
		return game.ErrAccountNotFound
	}
	return nil
}

func (r *MySQLRepository) DeductAccountGold(accountID string, amount int) error {
	result, err := r.db.Exec(`UPDATE accounts SET gold = gold - ? WHERE id = ? AND gold >= ?`, amount, accountID, amount)
	if err != nil {
		return err
	}
	affected, _ := result.RowsAffected()
	if affected == 0 {
		// 区分：账户不存在 vs 余额不足
		var exists int
		if scanErr := r.db.QueryRow(`SELECT 1 FROM accounts WHERE id = ? LIMIT 1`, accountID).Scan(&exists); scanErr != nil {
			return game.ErrAccountNotFound
		}
		return game.ErrInsufficientGold
	}
	return nil
}

func (r *MySQLRepository) AddCityGold(playerID string, amount int) (int, error) {
	result, err := r.db.Exec(
		`UPDATE players SET state_json = JSON_SET(state_json, '$.cityGold', CAST(IFNULL(JSON_EXTRACT(state_json, '$.cityGold'), 0) + 0 + ? AS SIGNED)) WHERE id = ?`,
		amount, playerID,
	)
	if err != nil {
		return 0, err
	}
	affected, _ := result.RowsAffected()
	if affected == 0 {
		return 0, game.ErrPlayerNotFound
	}
	var balance int
	err = r.db.QueryRow(`SELECT CAST(IFNULL(JSON_EXTRACT(state_json, '$.cityGold'), 0) + 0 AS SIGNED) FROM players WHERE id = ?`, playerID).Scan(&balance)
	return balance, err
}

func (r *MySQLRepository) DeductCityGold(playerID string, amount int) (int, error) {
	result, err := r.db.Exec(
		`UPDATE players SET state_json = JSON_SET(state_json, '$.cityGold', CAST(IFNULL(JSON_EXTRACT(state_json, '$.cityGold'), 0) + 0 - ? AS SIGNED)) WHERE id = ? AND IFNULL(JSON_EXTRACT(state_json, '$.cityGold'), 0) + 0 >= ?`,
		amount, playerID, amount,
	)
	if err != nil {
		return 0, err
	}
	affected, _ := result.RowsAffected()
	if affected == 0 {
		var exists int
		if scanErr := r.db.QueryRow(`SELECT 1 FROM players WHERE id = ? LIMIT 1`, playerID).Scan(&exists); scanErr != nil {
			return 0, game.ErrPlayerNotFound
		}
		return 0, game.ErrInsufficientCityGold
	}
	var balance int
	err = r.db.QueryRow(`SELECT CAST(IFNULL(JSON_EXTRACT(state_json, '$.cityGold'), 0) + 0 AS SIGNED) FROM players WHERE id = ?`, playerID).Scan(&balance)
	return balance, err
}

func (r *MySQLRepository) ExchangeGoldToCityGold(accountID string, playerID string, goldAmount int, cityGoldGain int) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// 扣账户金币
	result, err := tx.Exec(`UPDATE accounts SET gold = gold - ? WHERE id = ? AND gold >= ?`, goldAmount, accountID, goldAmount)
	if err != nil {
		return err
	}
	affected, _ := result.RowsAffected()
	if affected == 0 {
		var exists int
		if scanErr := tx.QueryRow(`SELECT 1 FROM accounts WHERE id = ? LIMIT 1`, accountID).Scan(&exists); scanErr != nil {
			return game.ErrAccountNotFound
		}
		return game.ErrInsufficientGold
	}

	// 加城金
	result, err = tx.Exec(
		`UPDATE players SET state_json = JSON_SET(state_json, '$.cityGold', CAST(IFNULL(JSON_EXTRACT(state_json, '$.cityGold'), 0) + 0 + ? AS SIGNED)) WHERE id = ?`,
		cityGoldGain, playerID,
	)
	if err != nil {
		return err
	}
	affected, _ = result.RowsAffected()
	if affected == 0 {
		return game.ErrPlayerNotFound
	}

	return tx.Commit()
}

func (r *MySQLRepository) ExchangeCityGoldToGold(accountID string, playerID string, cityGoldAmount int, goldGain int) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// 扣城金
	result, err := tx.Exec(
		`UPDATE players SET state_json = JSON_SET(state_json, '$.cityGold', CAST(IFNULL(JSON_EXTRACT(state_json, '$.cityGold'), 0) + 0 - ? AS SIGNED)) WHERE id = ? AND IFNULL(JSON_EXTRACT(state_json, '$.cityGold'), 0) + 0 >= ?`,
		cityGoldAmount, playerID, cityGoldAmount,
	)
	if err != nil {
		return err
	}
	affected, _ := result.RowsAffected()
	if affected == 0 {
		var exists int
		if scanErr := tx.QueryRow(`SELECT 1 FROM players WHERE id = ? LIMIT 1`, playerID).Scan(&exists); scanErr != nil {
			return game.ErrPlayerNotFound
		}
		return game.ErrInsufficientCityGold
	}

	// 加账户金币
	result, err = tx.Exec(`UPDATE accounts SET gold = gold + ? WHERE id = ?`, goldGain, accountID)
	if err != nil {
		return err
	}
	affected, _ = result.RowsAffected()
	if affected == 0 {
		return game.ErrAccountNotFound
	}

	return tx.Commit()
}

func (r *MySQLRepository) AccountExists(accountID string) (bool, error) {
	var exists int
	err := r.db.QueryRow(`SELECT 1 FROM accounts WHERE id = ? LIMIT 1`, accountID).Scan(&exists)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	return err == nil, err
}

func (r *MySQLRepository) ListAccounts() ([]game.AccountSummary, error) {
	rows, err := r.db.Query(
		`SELECT
			a.id,
			a.username,
			a.created_at,
			p.id,
			p.nickname,
			p.faction,
			p.updated_at
		FROM accounts a
		LEFT JOIN players p ON p.account_id = a.id
		ORDER BY a.created_at DESC, p.updated_at DESC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	accountOrder := []string{}
	accountMap := map[string]*game.AccountSummary{}
	for rows.Next() {
		var accountID string
		var username string
		var createdAt time.Time
		var playerID sql.NullString
		var nickname sql.NullString
		var faction sql.NullString
		var updatedAt sql.NullTime

		if err := rows.Scan(&accountID, &username, &createdAt, &playerID, &nickname, &faction, &updatedAt); err != nil {
			return nil, err
		}

		account, exists := accountMap[accountID]
		if !exists {
			accountOrder = append(accountOrder, accountID)
			account = &game.AccountSummary{
				ID:        accountID,
				Username:  username,
				CreatedAt: createdAt.UTC().Format(time.RFC3339),
				Players:   []game.PlayerSummary{},
			}
			accountMap[accountID] = account
		}

		if playerID.Valid {
			player := game.PlayerSummary{
				ID:       playerID.String,
				Nickname: nickname.String,
				Faction:  faction.String,
			}
			if updatedAt.Valid {
				player.UpdatedAt = updatedAt.Time.UTC().Format(time.RFC3339)
			}
			account.Players = append(account.Players, player)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	accounts := make([]game.AccountSummary, 0, len(accountOrder))
	for _, accountID := range accountOrder {
		accounts = append(accounts, *accountMap[accountID])
	}
	return accounts, nil
}

func (r *MySQLRepository) ListPlayers(accountID string) ([]game.PlayerSummary, error) {
	exists, err := r.AccountExists(accountID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, game.ErrAccountNotFound
	}

	rows, err := r.db.Query(
		`SELECT id, nickname, faction, state_json, updated_at
		 FROM players
		 WHERE account_id = ?
		 ORDER BY updated_at DESC`,
		accountID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	players := []game.PlayerSummary{}
	for rows.Next() {
		var id, nickname, faction string
		var stateJSON []byte
		var updatedAt time.Time
		if err := rows.Scan(&id, &nickname, &faction, &stateJSON, &updatedAt); err != nil {
			return nil, err
		}

		summary := game.PlayerSummary{
			ID:        id,
			Nickname:  nickname,
			Faction:   faction,
			UpdatedAt: updatedAt.UTC().Format(time.RFC3339),
		}

		// 从 state_json 提取摘要
		var state game.GameState
		if err := json.Unmarshal(stateJSON, &state); err == nil {
			for _, unit := range state.Army {
				summary.TotalArmy += unit.Amount
			}
			for _, b := range state.Buildings {
				summary.BuildingLevel += b.Level
			}
		}

		players = append(players, summary)
	}

	return players, rows.Err()
}

func (r *MySQLRepository) CreatePlayer(accountID string, state game.GameState, updatedAt time.Time) error {
	stateJSON, err := json.Marshal(state)
	if err != nil {
		return err
	}

	now := updatedAt.UTC()
	_, err = r.db.Exec(
		`INSERT INTO players (id, account_id, nickname, faction, state_json, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		state.Player.ID,
		accountID,
		state.Player.Nickname,
		state.Player.Faction,
		stateJSON,
		now,
		now,
	)
	return err
}

func (r *MySQLRepository) DeleteAccount(accountID string) error {
	result, err := r.db.Exec(`DELETE FROM accounts WHERE id = ?`, accountID)
	if err != nil {
		return err
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return game.ErrAccountNotFound
	}
	return nil
}

func (r *MySQLRepository) DeletePlayer(playerID string) error {
	// 先删战报
	_, _ = r.db.Exec(`DELETE FROM battle_reports WHERE player_id = ?`, playerID)

	result, err := r.db.Exec(`DELETE FROM players WHERE id = ?`, playerID)
	if err != nil {
		return err
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return game.ErrPlayerNotFound
	}
	return nil
}

func (r *MySQLRepository) GetState(playerID string) (game.GameState, error) {
	var stateJSON []byte
	err := r.db.QueryRow(`SELECT state_json FROM players WHERE id = ? LIMIT 1`, playerID).Scan(&stateJSON)
	if errors.Is(err, sql.ErrNoRows) {
		return game.GameState{}, game.ErrPlayerNotFound
	}
	if err != nil {
		return game.GameState{}, err
	}

	var state game.GameState
	if err := json.Unmarshal(stateJSON, &state); err != nil {
		return game.GameState{}, err
	}
	return state, nil
}

func (r *MySQLRepository) SaveState(state game.GameState, updatedAt time.Time) error {
	stateJSON, err := json.Marshal(state)
	if err != nil {
		return err
	}

	result, err := r.db.Exec(
		`UPDATE players
		 SET nickname = ?, faction = ?, state_json = ?, updated_at = ?
		 WHERE id = ?`,
		state.Player.Nickname,
		state.Player.Faction,
		stateJSON,
		updatedAt.UTC(),
		state.Player.ID,
	)
	if err != nil {
		return err
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return game.ErrPlayerNotFound
	}
	return nil
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

// --- Battle Report Methods ---

func (r *MySQLRepository) SaveReport(report game.BattleReport) error {
	reportJSON, err := json.Marshal(report)
	if err != nil {
		return err
	}

	createdAt, _ := time.Parse(time.RFC3339, report.CreatedAt)
	if createdAt.IsZero() {
		createdAt = time.Now()
	}

	_, err = r.db.Exec(
		`INSERT INTO battle_reports (id, player_id, report_json, type, is_read, deleted_by_player, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		report.ID,
		report.PlayerID,
		reportJSON,
		report.Type,
		report.Read,
		false,
		createdAt.UTC(),
	)
	return err
}

func (r *MySQLRepository) GetReportByID(reportID string) (game.BattleReport, error) {
	var reportJSON []byte
	err := r.db.QueryRow(`SELECT report_json FROM battle_reports WHERE id = ? LIMIT 1`, reportID).Scan(&reportJSON)
	if err != nil {
		return game.BattleReport{}, errors.New("report not found")
	}
	var report game.BattleReport
	if err := json.Unmarshal(reportJSON, &report); err != nil {
		return game.BattleReport{}, err
	}
	return report, nil
}

func (r *MySQLRepository) ListReports(playerID string, limit int, offset int) ([]game.BattleReport, int, error) {
	threeDaysAgo := time.Now().Add(-3 * 24 * time.Hour).UTC()
	if limit <= 0 {
		limit = 10
	}
	if offset < 0 {
		offset = 0
	}

	var total int
	if err := r.db.QueryRow(
		`SELECT COUNT(*) FROM battle_reports
		 WHERE player_id = ? AND deleted_by_player = 0 AND created_at > ?`,
		playerID, threeDaysAgo,
	).Scan(&total); err != nil {
		return nil, 0, err
	}

	rows, err := r.db.Query(
		`SELECT report_json, is_read FROM battle_reports
		 WHERE player_id = ? AND deleted_by_player = 0 AND created_at > ?
		 ORDER BY created_at DESC LIMIT ? OFFSET ?`,
		playerID, threeDaysAgo, limit, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var reports []game.BattleReport
	for rows.Next() {
		var reportJSON []byte
		var isRead bool
		if err := rows.Scan(&reportJSON, &isRead); err != nil {
			return nil, 0, err
		}
		var report game.BattleReport
		if err := json.Unmarshal(reportJSON, &report); err != nil {
			continue
		}
		report.Read = isRead
		reports = append(reports, report)
	}
	return reports, total, rows.Err()
}

func (r *MySQLRepository) ListAllReports(playerID string) ([]game.BattleReport, error) {
	rows, err := r.db.Query(
		`SELECT report_json FROM battle_reports
		 WHERE player_id = ?
		 ORDER BY created_at DESC`,
		playerID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var reports []game.BattleReport
	for rows.Next() {
		var reportJSON []byte
		if err := rows.Scan(&reportJSON); err != nil {
			return nil, err
		}
		var report game.BattleReport
		if err := json.Unmarshal(reportJSON, &report); err != nil {
			continue
		}
		reports = append(reports, report)
	}
	return reports, rows.Err()
}

func (r *MySQLRepository) MarkReportsRead(playerID string) error {
	_, err := r.db.Exec(
		`UPDATE battle_reports SET is_read = 1 WHERE player_id = ? AND is_read = 0 AND deleted_by_player = 0`,
		playerID,
	)
	return err
}

func (r *MySQLRepository) MarkSingleReportRead(playerID string, reportID string) error {
	_, err := r.db.Exec(
		`UPDATE battle_reports SET is_read = 1 WHERE id = ? AND player_id = ?`,
		reportID, playerID,
	)
	return err
}

func (r *MySQLRepository) DeleteReport(playerID string, reportID string) error {
	_, err := r.db.Exec(
		`UPDATE battle_reports SET deleted_by_player = 1 WHERE id = ? AND player_id = ?`,
		reportID, playerID,
	)
	return err
}

func (r *MySQLRepository) DeleteAllReports(playerID string) error {
	_, err := r.db.Exec(
		`UPDATE battle_reports SET deleted_by_player = 1 WHERE player_id = ? AND deleted_by_player = 0`,
		playerID,
	)
	return err
}

func (r *MySQLRepository) CountUnreadReports(playerID string) (int, error) {
	threeDaysAgo := time.Now().Add(-3 * 24 * time.Hour).UTC()
	var count int
	err := r.db.QueryRow(
		`SELECT COUNT(*) FROM battle_reports WHERE player_id = ? AND is_read = 0 AND deleted_by_player = 0 AND created_at > ?`,
		playerID, threeDaysAgo,
	).Scan(&count)
	return count, err
}

// --- MiniGame Record Methods ---

func (r *MySQLRepository) SaveMiniGameRecord(record game.MiniGameRecord) error {
	createdAt, _ := time.Parse(time.RFC3339, record.CreatedAt)
	if createdAt.IsZero() {
		createdAt = time.Now()
	}

	_, err := r.db.Exec(
		`INSERT INTO minigame_records (id, player_id, game_type, result_name, rarity, reward_unit, reward_amount, bet_unit, bet_amount, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		record.ID,
		record.PlayerID,
		record.GameType,
		record.ResultName,
		record.Rarity,
		record.RewardUnit,
		record.RewardAmount,
		record.BetUnit,
		record.BetAmount,
		createdAt.UTC(),
	)
	return err
}

func (r *MySQLRepository) ListMiniGameRecords(playerID string, limit int) ([]game.MiniGameRecord, error) {
	rows, err := r.db.Query(
		`SELECT id, player_id, game_type, result_name, rarity, reward_unit, reward_amount, bet_unit, bet_amount, created_at
		 FROM minigame_records
		 WHERE player_id = ?
		 ORDER BY created_at DESC
		 LIMIT ?`,
		playerID, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []game.MiniGameRecord
	for rows.Next() {
		var r game.MiniGameRecord
		var createdAt time.Time
		if err := rows.Scan(&r.ID, &r.PlayerID, &r.GameType, &r.ResultName, &r.Rarity, &r.RewardUnit, &r.RewardAmount, &r.BetUnit, &r.BetAmount, &createdAt); err != nil {
			return nil, err
		}
		r.CreatedAt = createdAt.UTC().Format(time.RFC3339)
		records = append(records, r)
	}
	return records, rows.Err()
}

// --- Gold Ledger Methods ---

func (r *MySQLRepository) WriteGoldLedger(entry game.GoldLedgerEntry) error {
	createdAt, _ := time.Parse(time.RFC3339, entry.CreatedAt)
	if createdAt.IsZero() {
		createdAt = time.Now()
	}

	_, err := r.db.Exec(
		`INSERT INTO gold_ledger
		 (account_id, player_id, currency, direction, amount, balance_after, ref_type, ref_id, reason, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		entry.AccountID,
		entry.PlayerID,
		entry.Currency,
		entry.Direction,
		entry.Amount,
		entry.BalanceAfter,
		entry.RefType,
		entry.RefID,
		entry.Reason,
		createdAt.UTC(),
	)
	return err
}

func (r *MySQLRepository) ListGoldLedger(filter game.GoldLedgerFilter) ([]game.GoldLedgerEntry, error) {
	limit := filter.Limit
	if limit <= 0 || limit > 1000 {
		limit = 200
	}

	query := `SELECT id, account_id, player_id, currency, direction, amount, balance_after, ref_type, ref_id, reason, created_at
		FROM gold_ledger WHERE 1=1`
	args := []interface{}{}
	if filter.AccountID != "" {
		query += " AND account_id = ?"
		args = append(args, filter.AccountID)
	}
	if filter.PlayerID != "" {
		query += " AND player_id = ?"
		args = append(args, filter.PlayerID)
	}
	if filter.Currency != "" {
		query += " AND currency = ?"
		args = append(args, filter.Currency)
	}
	if filter.RefType != "" {
		query += " AND ref_type = ?"
		args = append(args, filter.RefType)
	}
	if !filter.From.IsZero() {
		query += " AND created_at >= ?"
		args = append(args, filter.From.UTC())
	}
	if !filter.To.IsZero() {
		query += " AND created_at <= ?"
		args = append(args, filter.To.UTC())
	}
	query += " ORDER BY id DESC LIMIT ?"
	args = append(args, limit)

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []game.GoldLedgerEntry
	for rows.Next() {
		var e game.GoldLedgerEntry
		var createdAt time.Time
		if err := rows.Scan(
			&e.ID, &e.AccountID, &e.PlayerID, &e.Currency, &e.Direction,
			&e.Amount, &e.BalanceAfter, &e.RefType, &e.RefID, &e.Reason, &createdAt,
		); err != nil {
			return nil, err
		}
		e.CreatedAt = createdAt.UTC().Format(time.RFC3339)
		entries = append(entries, e)
	}
	return entries, rows.Err()
}

func (r *MySQLRepository) GetAccountIDByPlayerID(playerID string) (string, error) {
	var accountID string
	err := r.db.QueryRow(`SELECT account_id FROM players WHERE id = ? LIMIT 1`, playerID).Scan(&accountID)
	if errors.Is(err, sql.ErrNoRows) {
		return "", game.ErrPlayerNotFound
	}
	return accountID, err
}
