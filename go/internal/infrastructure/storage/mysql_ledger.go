// 本文件归口 MySQL 金币和城金流水仓储方法。
package storage

import (
	"time"

	"hero3/internal/app/game"
)

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
