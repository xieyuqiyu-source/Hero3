// 本文件归口 MySQL 金币和城金流水仓储方法。
package storage

import (
	"database/sql"
	"encoding/json"
	"fmt"
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

// WriteItemLedger 写入物品获得或消耗流水。
func (r *MySQLRepository) WriteItemLedger(entry game.ItemLedgerEntry) error {
	if entry.ID == "" {
		entry.ID = fmt.Sprintf("item_ledger_%d", time.Now().UnixNano())
	}
	createdAt, _ := time.Parse(time.RFC3339, entry.CreatedAt)
	if createdAt.IsZero() {
		createdAt = time.Now()
	}
	metadata, err := json.Marshal(entry.Metadata)
	if err != nil {
		return err
	}
	if entry.Metadata == nil {
		metadata = nil
	}
	_, err = r.db.Exec(
		`INSERT INTO item_ledger
		 (id, player_id, item_id, change_amount, before_amount, after_amount, reason, ref_type, ref_id, metadata_json, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		entry.ID,
		entry.PlayerID,
		entry.ItemID,
		entry.ChangeAmount,
		entry.BeforeAmount,
		entry.AfterAmount,
		entry.Reason,
		entry.RefType,
		entry.RefID,
		nullableJSONArg(metadata),
		createdAt.UTC(),
	)
	return err
}

// ListItemLedger 按筛选条件读取物品流水。
func (r *MySQLRepository) ListItemLedger(filter game.ItemLedgerFilter) ([]game.ItemLedgerEntry, int, error) {
	limit := filter.Limit
	if limit <= 0 || limit > 1000 {
		limit = 200
	}
	offset := filter.Offset
	if offset < 0 {
		offset = 0
	}
	where := " FROM item_ledger WHERE 1=1"
	args := []interface{}{}
	if filter.PlayerID != "" {
		where += " AND player_id = ?"
		args = append(args, filter.PlayerID)
	}
	if filter.ItemID != "" {
		where += " AND item_id = ?"
		args = append(args, filter.ItemID)
	}
	if filter.RefType != "" {
		where += " AND ref_type = ?"
		args = append(args, filter.RefType)
	}
	if filter.From != "" {
		if parsed, err := time.Parse(time.RFC3339, filter.From); err == nil {
			where += " AND created_at >= ?"
			args = append(args, parsed.UTC())
		}
	}
	if filter.To != "" {
		if parsed, err := time.Parse(time.RFC3339, filter.To); err == nil {
			where += " AND created_at <= ?"
			args = append(args, parsed.UTC())
		}
	}

	var total int
	if err := r.db.QueryRow(`SELECT COUNT(*)`+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	query := `SELECT id, player_id, item_id, change_amount, before_amount, after_amount, reason, ref_type, ref_id, metadata_json, created_at` + where + ` ORDER BY created_at DESC, id DESC LIMIT ? OFFSET ?`
	queryArgs := append(append([]interface{}{}, args...), limit, offset)
	rows, err := r.db.Query(query, queryArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	entries := []game.ItemLedgerEntry{}
	for rows.Next() {
		var entry game.ItemLedgerEntry
		var metadata sql.NullString
		var createdAt time.Time
		if err := rows.Scan(&entry.ID, &entry.PlayerID, &entry.ItemID, &entry.ChangeAmount, &entry.BeforeAmount, &entry.AfterAmount, &entry.Reason, &entry.RefType, &entry.RefID, &metadata, &createdAt); err != nil {
			return nil, 0, err
		}
		if metadata.Valid && metadata.String != "" {
			_ = json.Unmarshal([]byte(metadata.String), &entry.Metadata)
		}
		entry.CreatedAt = createdAt.UTC().Format(time.RFC3339)
		entries = append(entries, entry)
	}
	return entries, total, rows.Err()
}

func nullableJSONArg(data []byte) any {
	if len(data) == 0 {
		return nil
	}
	return string(data)
}
