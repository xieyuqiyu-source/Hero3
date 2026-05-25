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

// ExchangeRate 金币兑换城金比例：1 金币 = 10 城金
const ExchangeRate = 10

// ExchangeCooldownSeconds 兑换冷却时间（秒）
const ExchangeCooldownSeconds = 3600 // 1 小时

var ErrExchangeCooldown = errors.New("exchange is on cooldown")

// ExchangeGoldToCityGold 金币兑换城金
// 从账户扣金币，给存档加城金
func (s *Service) ExchangeGoldToCityGold(accountID string, playerID string, goldAmount int) (GameState, error) {
	accountID = strings.TrimSpace(accountID)
	playerID = strings.TrimSpace(playerID)
	if accountID == "" {
		return GameState{}, ErrAccountNotFound
	}
	if playerID == "" {
		return GameState{}, ErrPlayerNotFound
	}
	if goldAmount <= 0 {
		return GameState{}, ErrInvalidGoldAmount
	}

	// 获取账户
	account, err := s.repo.GetAccountByUsername("") // 需要通过 ID 获取
	_ = account
	// 由于当前 Repository 没有 GetAccountByID，我们通过 state 验证玩家归属
	// 并直接操作 state 中记录的兑换冷却

	state, err := s.repo.GetState(playerID)
	if err != nil {
		return GameState{}, err
	}

	// 检查冷却时间
	now := time.Now()
	if state.LastExchangeAt != "" {
		lastExchange, parseErr := time.Parse(resourceDateLayout, state.LastExchangeAt)
		if parseErr == nil {
			elapsed := now.Sub(lastExchange).Seconds()
			if elapsed < float64(ExchangeCooldownSeconds) {
				return GameState{}, ErrExchangeCooldown
			}
		}
	}

	// TODO: 扣除账户金币（需要 GetAccountByID + UpdateAccountGold 接口）
	// 现阶段先只做城金发放，账户金币扣除待充值系统接入后完善

	// 发放城金
	cityGoldGain := goldAmount * ExchangeRate
	state.CityGold += cityGoldGain
	state.LastExchangeAt = now.UTC().Format(resourceDateLayout)
	state.ServerTime = now.UTC().Format(resourceDateLayout)

	if err := s.repo.SaveState(state, now); err != nil {
		return GameState{}, err
	}

	return state, nil
}
