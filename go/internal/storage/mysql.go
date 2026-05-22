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
	}

	for _, statement := range statements {
		if _, err := db.ExecContext(ctx, statement); err != nil {
			return err
		}
	}

	return nil
}

func (r *MySQLRepository) CreateAccount(account game.Account) error {
	_, err := r.db.Exec(
		`INSERT INTO accounts (id, username, password_hash, created_at) VALUES (?, ?, ?, ?)`,
		account.ID,
		account.Username,
		account.PasswordHash,
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
		`SELECT id, username, password_hash, created_at FROM accounts WHERE username = ? LIMIT 1`,
		username,
	).Scan(&account.ID, &account.Username, &account.PasswordHash, &account.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return game.Account{}, game.ErrAccountNotFound
	}
	return account, err
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
