// 本文件归口玩家城金等玩家级货币的 MySQL 权威存储。
package storage

import (
	"database/sql"
	"errors"
	"time"

	"hero3/internal/app/game"
)

type playerCurrencySnapshot struct {
	CityGold       game.FlexInt
	LastExchangeAt string
}

// overlayAuthoritativeCurrency 使用玩家货币权威表覆盖兼容快照；旧玩家缺表时保留 state_json 兜底。
func (r *MySQLRepository) overlayAuthoritativeCurrency(state *game.GameState, playerID string) error {
	snapshot, found, err := loadPlayerCurrency(r.db, playerID)
	if err != nil {
		return err
	}
	applyAuthoritativeCurrency(state, snapshot, found)
	return nil
}

// overlayAuthoritativeCurrencyTx 在事务内锁定玩家货币；旧玩家缺表时把旧快照回填到权威表。
func overlayAuthoritativeCurrencyTx(tx *sql.Tx, state *game.GameState, playerID string, updatedAt time.Time) error {
	snapshot, found, err := loadPlayerCurrencyTx(tx, playerID)
	if err != nil {
		return err
	}
	if found {
		applyAuthoritativeCurrency(state, snapshot, true)
		return nil
	}
	if err := syncPlayerCurrencyTx(tx, playerID, state, updatedAt.UTC()); err != nil {
		return err
	}
	return nil
}

// applyAuthoritativeCurrency 将货币权威表写回 GameState。
func applyAuthoritativeCurrency(state *game.GameState, snapshot playerCurrencySnapshot, found bool) {
	if !found {
		return
	}
	state.CityGold = snapshot.CityGold
	state.LastExchangeAt = snapshot.LastExchangeAt
}

// currencySnapshotFromState 生成用于比较的玩家货币快照。
func currencySnapshotFromState(state game.GameState) playerCurrencySnapshot {
	return playerCurrencySnapshot{
		CityGold:       state.CityGold,
		LastExchangeAt: state.LastExchangeAt,
	}
}

// currencySnapshotChanged 判断玩家货币是否发生变化。
func currencySnapshotChanged(before playerCurrencySnapshot, state game.GameState) bool {
	after := currencySnapshotFromState(state)
	return before.CityGold != after.CityGold || before.LastExchangeAt != after.LastExchangeAt
}

// loadPlayerCurrency 从 player_currencies 读取玩家货币。
func loadPlayerCurrency(queryer rowQueryer, playerID string) (playerCurrencySnapshot, bool, error) {
	return loadPlayerCurrencyWithQuery(queryer, playerID, "")
}

// loadPlayerCurrencyTx 在事务内读取并锁定玩家货币。
func loadPlayerCurrencyTx(tx *sql.Tx, playerID string) (playerCurrencySnapshot, bool, error) {
	return loadPlayerCurrencyWithQuery(tx, playerID, " FOR UPDATE")
}

// loadPlayerCurrencyWithQuery 读取玩家货币行。
func loadPlayerCurrencyWithQuery(queryer rowQueryer, playerID string, lockClause string) (playerCurrencySnapshot, bool, error) {
	var cityGold int
	var lastExchangeAt sql.NullTime
	err := queryer.QueryRow(
		`SELECT city_gold, last_exchange_at
		 FROM player_currencies
		 WHERE player_id = ?
		 LIMIT 1`+lockClause,
		playerID,
	).Scan(&cityGold, &lastExchangeAt)
	if errors.Is(err, sql.ErrNoRows) {
		return playerCurrencySnapshot{}, false, nil
	}
	if err != nil {
		return playerCurrencySnapshot{}, false, err
	}
	snapshot := playerCurrencySnapshot{CityGold: game.FlexInt(cityGold)}
	if lastExchangeAt.Valid {
		snapshot.LastExchangeAt = lastExchangeAt.Time.UTC().Format(time.RFC3339)
	}
	return snapshot, true, nil
}

// syncPlayerCurrencyTx 把玩家货币写入权威表。
func syncPlayerCurrencyTx(tx *sql.Tx, playerID string, state *game.GameState, updatedAt time.Time) error {
	lastExchangeAt := parseStorageTime(state.LastExchangeAt)
	_, err := tx.Exec(
		`INSERT INTO player_currencies (player_id, city_gold, last_exchange_at, updated_at)
		 VALUES (?, ?, ?, ?)
		 ON DUPLICATE KEY UPDATE
			city_gold = VALUES(city_gold),
			last_exchange_at = VALUES(last_exchange_at),
			updated_at = VALUES(updated_at)`,
		playerID,
		int(state.CityGold),
		nullableTimeArg(lastExchangeAt),
		updatedAt.UTC(),
	)
	return err
}
