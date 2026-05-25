package game

import (
	"errors"
	"strings"
	"time"
)

var (
	ErrInsufficientGold     = errors.New("insufficient gold")
	ErrInsufficientCityGold = errors.New("insufficient city gold")
	ErrInvalidGoldAmount    = errors.New("invalid gold amount")
	ErrExchangeCooldown     = errors.New("exchange is on cooldown")
)

// 兑换配置
const (
	ExchangeRate            = 10   // 1 金币 = 10 城金
	ReverseExchangeRate     = 15   // 15 城金 = 1 金币（反向有损耗）
	ExchangeCooldownSeconds = 3600 // 兑换冷却 1 小时
)

// AddGold 给存档增加城金（原子操作）
func (s *Service) AddGold(playerID string, amount int, reason string) (GameState, error) {
	playerID = strings.TrimSpace(playerID)
	if playerID == "" {
		return GameState{}, ErrPlayerNotFound
	}
	if amount <= 0 {
		return GameState{}, ErrInvalidGoldAmount
	}

	// 原子加城金
	if _, err := s.repo.AddCityGold(playerID, amount); err != nil {
		return GameState{}, err
	}

	// 读取最新状态返回
	state, err := s.repo.GetState(playerID)
	if err != nil {
		return GameState{}, err
	}

	_ = reason // TODO: 记录流水日志
	return state, nil
}

// DeductGold 从存档扣除城金（原子操作，余额不足返回 ErrInsufficientCityGold）
func (s *Service) DeductGold(playerID string, amount int, reason string) (GameState, error) {
	playerID = strings.TrimSpace(playerID)
	if playerID == "" {
		return GameState{}, ErrPlayerNotFound
	}
	if amount <= 0 {
		return GameState{}, ErrInvalidGoldAmount
	}

	// 原子扣城金（余额不足会返回错误）
	if _, err := s.repo.DeductCityGold(playerID, amount); err != nil {
		return GameState{}, err
	}

	state, err := s.repo.GetState(playerID)
	if err != nil {
		return GameState{}, err
	}

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

// ExchangeGoldToCityGold 金币 → 城金（1 金币 = 10 城金，原子操作）
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

	// 检查冷却
	state, err := s.repo.GetState(playerID)
	if err != nil {
		return GameState{}, err
	}
	now := time.Now()
	if state.LastExchangeAt != "" {
		lastExchange, parseErr := time.Parse(resourceDateLayout, state.LastExchangeAt)
		if parseErr == nil && now.Sub(lastExchange).Seconds() < float64(ExchangeCooldownSeconds) {
			return GameState{}, ErrExchangeCooldown
		}
	}

	// 原子扣除账户金币（余额不足返回 ErrInsufficientGold）
	if err := s.repo.DeductAccountGold(accountID, goldAmount); err != nil {
		return GameState{}, err
	}

	// 原子加城金
	cityGoldGain := goldAmount * ExchangeRate
	if _, err := s.repo.AddCityGold(playerID, cityGoldGain); err != nil {
		// 回滚账户金币（尽力而为）
		_ = s.repo.AddAccountGold(accountID, goldAmount)
		return GameState{}, err
	}

	// 更新冷却时间
	state, _ = s.repo.GetState(playerID)
	state.LastExchangeAt = now.UTC().Format(resourceDateLayout)
	state.ServerTime = now.UTC().Format(resourceDateLayout)
	_ = s.repo.SaveState(state, now)

	// 返回最新状态
	state, _ = s.repo.GetState(playerID)
	return state, nil
}

// ExchangeCityGoldToGold 城金 → 金币（15 城金 = 1 金币，有损耗，原子操作）
func (s *Service) ExchangeCityGoldToGold(accountID string, playerID string, cityGoldAmount int) (GameState, error) {
	accountID = strings.TrimSpace(accountID)
	playerID = strings.TrimSpace(playerID)
	if accountID == "" {
		return GameState{}, ErrAccountNotFound
	}
	if playerID == "" {
		return GameState{}, ErrPlayerNotFound
	}
	if cityGoldAmount < ReverseExchangeRate {
		return GameState{}, ErrInvalidGoldAmount
	}

	// 检查冷却
	state, err := s.repo.GetState(playerID)
	if err != nil {
		return GameState{}, err
	}
	now := time.Now()
	if state.LastExchangeAt != "" {
		lastExchange, parseErr := time.Parse(resourceDateLayout, state.LastExchangeAt)
		if parseErr == nil && now.Sub(lastExchange).Seconds() < float64(ExchangeCooldownSeconds) {
			return GameState{}, ErrExchangeCooldown
		}
	}

	// 原子扣城金（余额不足返回 ErrInsufficientCityGold）
	if _, err := s.repo.DeductCityGold(playerID, cityGoldAmount); err != nil {
		return GameState{}, err
	}

	// 原子加账户金币
	goldGain := cityGoldAmount / ReverseExchangeRate
	if err := s.repo.AddAccountGold(accountID, goldGain); err != nil {
		// 回滚城金（尽力而为）
		_, _ = s.repo.AddCityGold(playerID, cityGoldAmount)
		return GameState{}, err
	}

	// 更新冷却时间
	state, _ = s.repo.GetState(playerID)
	state.LastExchangeAt = now.UTC().Format(resourceDateLayout)
	state.ServerTime = now.UTC().Format(resourceDateLayout)
	_ = s.repo.SaveState(state, now)

	// 返回最新状态
	state, _ = s.repo.GetState(playerID)
	return state, nil
}
