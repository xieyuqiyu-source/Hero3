// 本文件归口 MySQL 账号、账号资产和玩家列表仓储方法。
package storage

import (
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"hero3/internal/app/game"
)

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

func (r *MySQLRepository) AccountExists(accountID string) (bool, error) {
	var exists int
	err := r.db.QueryRow(`SELECT 1 FROM accounts WHERE id = ? LIMIT 1`, accountID).Scan(&exists)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	return err == nil, err
}

func (r *MySQLRepository) MailAddressExists(nickname string, mailCode string) (bool, error) {
	var exists int
	err := r.db.QueryRow(
		`SELECT 1 FROM players WHERE nickname = ? AND mail_code = ? LIMIT 1`,
		nickname, mailCode,
	).Scan(&exists)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	return err == nil, err
}

func (r *MySQLRepository) FindPlayerByMailAddress(nickname string, mailCode string) (game.PlayerSummary, error) {
	var summary game.PlayerSummary
	var updatedAt time.Time
	err := r.db.QueryRow(
		`SELECT id, nickname, faction, mail_code, updated_at
		 FROM players
		 WHERE nickname = ? AND mail_code = ?
		 LIMIT 1`,
		nickname, mailCode,
	).Scan(&summary.ID, &summary.Nickname, &summary.Faction, &summary.MailCode, &updatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return game.PlayerSummary{}, game.ErrPlayerNotFound
	}
	if err != nil {
		return game.PlayerSummary{}, err
	}
	summary.UpdatedAt = updatedAt.UTC().Format(time.RFC3339)
	return summary, nil
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
			p.mail_code,
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
		var mailCode sql.NullString
		var updatedAt sql.NullTime

		if err := rows.Scan(&accountID, &username, &createdAt, &playerID, &nickname, &faction, &mailCode, &updatedAt); err != nil {
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
				MailCode: mailCode.String,
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
		`SELECT id, nickname, faction, mail_code, state_json, updated_at
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
		var id, nickname, faction, mailCode string
		var stateJSON []byte
		var updatedAt time.Time
		if err := rows.Scan(&id, &nickname, &faction, &mailCode, &stateJSON, &updatedAt); err != nil {
			return nil, err
		}

		summary := game.PlayerSummary{
			ID:        id,
			Nickname:  nickname,
			Faction:   faction,
			MailCode:  mailCode,
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

func (r *MySQLRepository) GetAccountIDByPlayerID(playerID string) (string, error) {
	var accountID string
	err := r.db.QueryRow(`SELECT account_id FROM players WHERE id = ? LIMIT 1`, playerID).Scan(&accountID)
	if errors.Is(err, sql.ErrNoRows) {
		return "", game.ErrPlayerNotFound
	}
	return accountID, err
}
