package game

import (
	"errors"
	"strings"
	"time"
)

var (
	ErrInsufficientGold  = errors.New("insufficient gold")
	ErrInvalidGoldAmount = errors.New("invalid gold amount")
	ErrExchangeCooldown  = errors.New("exchange is on cooldown")
)

// 兑换配置
const (
	ExchangeRate            = 10   // 1 金币 = 10 城金
	ReverseExchangeRate     = 15   // 15 城金 = 1 金币（反向有损耗）
	ExchangeCooldownSeconds = 3600 // 兑换冷却 1 小时
)

// AddGold 给存档增加城金（内部调用，如 NPC 掉落、任务奖励）
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

	_ = reason // TODO: 记录流水日志
	return state, nil
}

// DeductGold 从存档扣除城金（内部调用，如加速、刷新）
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

// ExchangeGoldToCityGold 金币 → 城金（1 金币 = 10 城金）
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

	// 获取账户，校验金币余额
	account, err := s.repo.GetAccountByID(accountID)
	if err != nil {
		return GameState{}, err
	}
	if account.Gold < goldAmount {
		return GameState{}, ErrInsufficientGold
	}

	// 获取存档，校验冷却
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

	// 扣除账户金币
	account.Gold -= goldAmount
	if err := s.repo.UpdateAccountGold(accountID, account.Gold); err != nil {
		return GameState{}, err
	}

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

// ExchangeCityGoldToGold 城金 → 金币（15 城金 = 1 金币，有损耗）
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

	// 获取存档，校验城金余额和冷却
	state, err := s.repo.GetState(playerID)
	if err != nil {
		return GameState{}, err
	}

	if state.CityGold < cityGoldAmount {
		return GameState{}, ErrInsufficientGold
	}

	now := time.Now()
	if state.LastExchangeAt != "" {
		lastExchange, parseErr := time.Parse(resourceDateLayout, state.LastExchangeAt)
		if parseErr == nil && now.Sub(lastExchange).Seconds() < float64(ExchangeCooldownSeconds) {
			return GameState{}, ErrExchangeCooldown
		}
	}

	// 计算获得金币（向下取整）
	goldGain := cityGoldAmount / ReverseExchangeRate

	// 扣除城金
	state.CityGold -= cityGoldAmount
	state.LastExchangeAt = now.UTC().Format(resourceDateLayout)
	state.ServerTime = now.UTC().Format(resourceDateLayout)

	if err := s.repo.SaveState(state, now); err != nil {
		return GameState{}, err
	}

	// 增加账户金币
	account, err := s.repo.GetAccountByID(accountID)
	if err != nil {
		return GameState{}, err
	}
	account.Gold += goldGain
	if err := s.repo.UpdateAccountGold(accountID, account.Gold); err != nil {
		return GameState{}, err
	}

	return state, nil
}
