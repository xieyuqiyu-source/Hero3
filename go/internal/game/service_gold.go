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

// 兑换配置（从 balance 配置读取，支持热更）
func exchangeRate() int {
	r := currentBalance().ExchangeRate
	if r <= 0 {
		return 10
	}
	return r
}

func reverseExchangeRate() int {
	r := currentBalance().ReverseExchangeRate
	if r <= 0 {
		return 15
	}
	return r
}

func exchangeCooldownSeconds() int {
	return currentBalance().ExchangeCooldownSecs // 0 表示无冷却
}

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

	return int(state.CityGold), nil
}

// ExchangeGoldToCityGold 金币 → 城金（事务操作，比例从配置读取）
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
	cooldown := exchangeCooldownSeconds()
	if cooldown > 0 && state.LastExchangeAt != "" {
		lastExchange, parseErr := time.Parse(resourceDateLayout, state.LastExchangeAt)
		if parseErr == nil && now.Sub(lastExchange).Seconds() < float64(cooldown) {
			return GameState{}, ErrExchangeCooldown
		}
	}

	// 事务：扣账户金币 + 加城金（要么都成功，要么都不执行）
	cityGoldGain := goldAmount * exchangeRate()
	if err := s.repo.ExchangeGoldToCityGold(accountID, playerID, goldAmount, cityGoldGain); err != nil {
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

// ExchangeCityGoldToGold 城金 → 金币（有损耗，事务操作，比例从配置读取）
func (s *Service) ExchangeCityGoldToGold(accountID string, playerID string, cityGoldAmount int) (GameState, error) {
	accountID = strings.TrimSpace(accountID)
	playerID = strings.TrimSpace(playerID)
	if accountID == "" {
		return GameState{}, ErrAccountNotFound
	}
	if playerID == "" {
		return GameState{}, ErrPlayerNotFound
	}
	if cityGoldAmount < reverseExchangeRate() {
		return GameState{}, ErrInvalidGoldAmount
	}

	// 检查冷却
	state, err := s.repo.GetState(playerID)
	if err != nil {
		return GameState{}, err
	}
	now := time.Now()
	cooldown := exchangeCooldownSeconds()
	if cooldown > 0 && state.LastExchangeAt != "" {
		lastExchange, parseErr := time.Parse(resourceDateLayout, state.LastExchangeAt)
		if parseErr == nil && now.Sub(lastExchange).Seconds() < float64(cooldown) {
			return GameState{}, ErrExchangeCooldown
		}
	}

	// 事务：扣城金 + 加账户金币（要么都成功，要么都不执行）
	goldGain := cityGoldAmount / reverseExchangeRate()
	if err := s.repo.ExchangeCityGoldToGold(accountID, playerID, cityGoldAmount, goldGain); err != nil {
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
