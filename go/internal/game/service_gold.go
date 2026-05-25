package game

import (
	"errors"
	"strings"
	"time"
)

var (
	ErrInsufficientGold  = errors.New("insufficient gold")
	ErrInvalidGoldAmount = errors.New("invalid gold amount")
)

// GoldLog 金币流水记录
type GoldLog struct {
	PlayerID  string `json:"playerId"`
	Amount    int    `json:"amount"`  // 正数=获得，负数=消耗
	Balance   int    `json:"balance"` // 操作后余额
	Reason    string `json:"reason"`  // 来源/用途标识
	CreatedAt string `json:"createdAt"`
}

// AddGold 给存档增加城金
// reason 示例: "npc_drop", "daily_login", "system_grant", "exchange"
func (s *Service) AddGold(playerID string, amount int, reason string) (GameState, error) {
	playerID = strings.TrimSpace(playerID)
	if playerID == "" {
		return GameState{}, ErrPlayerNotFound
	}
	if amount <= 0 {
		return GameState{}, ErrInvalidGoldAmount
	}

	state, err := s.repo.GetState(playerID)
	if err != nil {
		return GameState{}, err
	}

	state.CityGold += amount

	now := time.Now()
	state.ServerTime = now.UTC().Format(resourceDateLayout)
	if err := s.repo.SaveState(state, now); err != nil {
		return GameState{}, err
	}

	// TODO: 记录城金流水日志（后续接入 gold_logs 表）
	_ = reason

	return state, nil
}

// DeductGold 从存档扣除城金
// reason 示例: "production_boost", "instant_recruit", "npc_refresh", "instant_upgrade"
func (s *Service) DeductGold(playerID string, amount int, reason string) (GameState, error) {
	playerID = strings.TrimSpace(playerID)
	if playerID == "" {
		return GameState{}, ErrPlayerNotFound
	}
	if amount <= 0 {
		return GameState{}, ErrInvalidGoldAmount
	}

	state, err := s.repo.GetState(playerID)
	if err != nil {
		return GameState{}, err
	}

	if state.CityGold < amount {
		return GameState{}, ErrInsufficientGold
	}

	state.CityGold -= amount

	now := time.Now()
	state.ServerTime = now.UTC().Format(resourceDateLayout)
	if err := s.repo.SaveState(state, now); err != nil {
		return GameState{}, err
	}

	// TODO: 记录城金流水日志
	_ = reason

	return state, nil
}

// GetGold 查询存档城金余额
func (s *Service) GetGold(playerID string) (int, error) {
	playerID = strings.TrimSpace(playerID)
	if playerID == "" {
		return 0, ErrPlayerNotFound
	}

	state, err := s.repo.GetState(playerID)
	if err != nil {
		return 0, err
	}

	return state.CityGold, nil
}
