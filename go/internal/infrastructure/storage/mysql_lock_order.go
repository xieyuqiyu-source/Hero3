// 本文件提供 MySQL 跨玩家事务的固定加锁顺序，降低互相攻击或增援时的死锁概率。
package storage

import (
	"database/sql"
	"strings"

	"hero3/internal/app/game"
)

type lockedGameState struct {
	state game.GameState
	json  []byte
}

// loadReinforcementPlayerPairOrderedTx 按玩家 ID 固定顺序锁定两名玩家，并按调用方语义返回。
func loadReinforcementPlayerPairOrderedTx(tx *sql.Tx, firstPlayerID string, secondPlayerID string) (game.GameState, []byte, game.GameState, []byte, error) {
	return loadPlayerPairOrderedTx(tx, firstPlayerID, secondPlayerID, loadReinforcementPlayerStateTx)
}

// loadPvpPlayerPairOrderedTx 按玩家 ID 固定顺序锁定两名 PVP 玩家，并按攻击/防守语义返回。
func loadPvpPlayerPairOrderedTx(tx *sql.Tx, firstPlayerID string, secondPlayerID string) (game.GameState, []byte, game.GameState, []byte, error) {
	return loadPlayerPairOrderedTx(tx, firstPlayerID, secondPlayerID, loadPvpPlayerStateTx)
}

// loadPlayerPairOrderedTx 使用传入加载器按稳定顺序读取并锁定两个玩家状态。
func loadPlayerPairOrderedTx(
	tx *sql.Tx,
	firstPlayerID string,
	secondPlayerID string,
	loader func(*sql.Tx, string) (game.GameState, []byte, error),
) (game.GameState, []byte, game.GameState, []byte, error) {
	firstPlayerID = strings.TrimSpace(firstPlayerID)
	secondPlayerID = strings.TrimSpace(secondPlayerID)
	if firstPlayerID == secondPlayerID {
		state, stateJSON, err := loader(tx, firstPlayerID)
		return state, stateJSON, state, stateJSON, err
	}

	locked := map[string]lockedGameState{}
	ordered := []string{firstPlayerID, secondPlayerID}
	if ordered[1] < ordered[0] {
		ordered[0], ordered[1] = ordered[1], ordered[0]
	}
	for _, playerID := range ordered {
		state, stateJSON, err := loader(tx, playerID)
		if err != nil {
			return game.GameState{}, nil, game.GameState{}, nil, err
		}
		locked[playerID] = lockedGameState{state: state, json: stateJSON}
	}

	first := locked[firstPlayerID]
	second := locked[secondPlayerID]
	return first.state, first.json, second.state, second.json, nil
}
