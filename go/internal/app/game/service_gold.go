package game

import (
	"errors"
	"log/slog"
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

// speedUpCost 计算加速所需城金（剩余秒数 / 每城金折抵秒数，向上取整，最少 1）
func speedUpCost(remainingSeconds int) int {
	rate := currentBalance().CityGoldPerSecond
	if rate <= 0 {
		rate = 120
	}
	cost := (remainingSeconds + rate - 1) / rate // 向上取整
	if cost < 1 {
		cost = 1
	}
	return cost
}

// AddGold 给存档增加城金（原子操作），仅返回货币局部结果。
func (s *Service) AddGold(playerID string, amount int, reason string) (CurrencyActionResult, error) {
	playerID = strings.TrimSpace(playerID)
	if playerID == "" {
		return CurrencyActionResult{}, ErrPlayerNotFound
	}
	if amount <= 0 {
		return CurrencyActionResult{}, ErrInvalidGoldAmount
	}

	result, err := s.GrantRewards(playerID, []Reward{{
		Type:   RewardTypeCityGold,
		ID:     RewardTypeCityGold,
		Amount: amount,
	}}, RewardGrantContext{
		PlayerID: playerID,
		RefType:  LedgerRefAdminAdjust,
		Reason:   reason,
	})
	if err != nil {
		return CurrencyActionResult{}, err
	}

	return BuildCurrencyActionResult(result.State, 0), nil
}

// DeductGold 从存档扣除城金（原子操作），仅返回货币局部结果。
func (s *Service) DeductGold(playerID string, amount int, reason string) (CurrencyActionResult, error) {
	playerID = strings.TrimSpace(playerID)
	if playerID == "" {
		return CurrencyActionResult{}, ErrPlayerNotFound
	}
	if amount <= 0 {
		return CurrencyActionResult{}, ErrInvalidGoldAmount
	}

	now := time.Now()
	refID := "city_gold_deduct_" + randomID(10)
	state, err := s.repo.UpdateScopedRewardState(playerID, RewardAssetScope{Currency: true}, now, func(state *GameState) error {
		if int(state.CityGold) < amount {
			return ErrInsufficientCityGold
		}
		state.CityGold -= FlexInt(amount)
		state.ServerTime = now.UTC().Format(resourceDateLayout)
		return nil
	})
	if err != nil {
		return CurrencyActionResult{}, err
	}

	s.recordLedger(GoldLedgerEntry{
		PlayerID:     playerID,
		Currency:     LedgerCurrencyCityGold,
		Direction:    LedgerDirectionDebit,
		Amount:       amount,
		BalanceAfter: int(state.CityGold),
		RefType:      LedgerRefAdminAdjust,
		RefID:        refID,
		Reason:       reason,
	})
	s.publishCurrencyChanged(playerID, "", refID, LedgerRefAdminAdjust)

	return BuildCurrencyActionResult(state, 0), nil
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

// ExchangeGoldToCityGold 金币转城金，仅返回货币局部结果。
func (s *Service) ExchangeGoldToCityGold(accountID string, playerID string, goldAmount int) (CurrencyActionResult, error) {
	accountID = strings.TrimSpace(accountID)
	playerID = strings.TrimSpace(playerID)
	if accountID == "" {
		return CurrencyActionResult{}, ErrAccountNotFound
	}
	if playerID == "" {
		return CurrencyActionResult{}, ErrPlayerNotFound
	}
	if goldAmount <= 0 {
		return CurrencyActionResult{}, ErrInvalidGoldAmount
	}

	now := time.Now()
	cityGoldGain := goldAmount * exchangeRate()
	refID := "exchange_" + randomID(10)

	account, state, err := s.repo.UpdateScopedAccountRewardState(accountID, playerID, RewardAssetScope{Currency: true}, now, func(account *Account, state *GameState) error {
		if err := ensureExchangeCooldown(*state, now); err != nil {
			return err
		}
		if account.Gold < goldAmount {
			return ErrInsufficientGold
		}
		account.Gold -= goldAmount
		state.CityGold += FlexInt(cityGoldGain)
		state.LastExchangeAt = now.UTC().Format(resourceDateLayout)
		state.ServerTime = now.UTC().Format(resourceDateLayout)
		return nil
	})
	if err != nil {
		return CurrencyActionResult{}, err
	}

	s.recordLedger(GoldLedgerEntry{
		AccountID:    accountID,
		PlayerID:     playerID,
		Currency:     LedgerCurrencyGold,
		Direction:    LedgerDirectionDebit,
		Amount:       goldAmount,
		BalanceAfter: account.Gold,
		RefType:      LedgerRefExchange,
		RefID:        refID,
	})
	s.recordLedger(GoldLedgerEntry{
		AccountID:    accountID,
		PlayerID:     playerID,
		Currency:     LedgerCurrencyCityGold,
		Direction:    LedgerDirectionCredit,
		Amount:       cityGoldGain,
		BalanceAfter: int(state.CityGold),
		RefType:      LedgerRefExchange,
		RefID:        refID,
	})
	s.publishCurrencyChanged(playerID, accountID, refID, LedgerRefExchange)

	return BuildCurrencyActionResult(state, account.Gold), nil
}

// ExchangeCityGoldToGold 城金转金币，仅返回货币局部结果。
func (s *Service) ExchangeCityGoldToGold(accountID string, playerID string, cityGoldAmount int) (CurrencyActionResult, error) {
	accountID = strings.TrimSpace(accountID)
	playerID = strings.TrimSpace(playerID)
	if accountID == "" {
		return CurrencyActionResult{}, ErrAccountNotFound
	}
	if playerID == "" {
		return CurrencyActionResult{}, ErrPlayerNotFound
	}
	if cityGoldAmount < reverseExchangeRate() {
		return CurrencyActionResult{}, ErrInvalidGoldAmount
	}

	now := time.Now()
	goldGain := cityGoldAmount / reverseExchangeRate()
	refID := "exchange_" + randomID(10)

	account, state, err := s.repo.UpdateScopedAccountRewardState(accountID, playerID, RewardAssetScope{Currency: true}, now, func(account *Account, state *GameState) error {
		if err := ensureExchangeCooldown(*state, now); err != nil {
			return err
		}
		if int(state.CityGold) < cityGoldAmount {
			return ErrInsufficientCityGold
		}
		state.CityGold -= FlexInt(cityGoldAmount)
		account.Gold += goldGain
		state.LastExchangeAt = now.UTC().Format(resourceDateLayout)
		state.ServerTime = now.UTC().Format(resourceDateLayout)
		return nil
	})
	if err != nil {
		return CurrencyActionResult{}, err
	}

	s.recordLedger(GoldLedgerEntry{
		AccountID:    accountID,
		PlayerID:     playerID,
		Currency:     LedgerCurrencyCityGold,
		Direction:    LedgerDirectionDebit,
		Amount:       cityGoldAmount,
		BalanceAfter: int(state.CityGold),
		RefType:      LedgerRefExchange,
		RefID:        refID,
	})
	s.recordLedger(GoldLedgerEntry{
		AccountID:    accountID,
		PlayerID:     playerID,
		Currency:     LedgerCurrencyGold,
		Direction:    LedgerDirectionCredit,
		Amount:       goldGain,
		BalanceAfter: account.Gold,
		RefType:      LedgerRefExchange,
		RefID:        refID,
	})
	s.publishCurrencyChanged(playerID, accountID, refID, LedgerRefExchange)

	return BuildCurrencyActionResult(state, account.Gold), nil
}

func ensureExchangeCooldown(state GameState, now time.Time) error {
	cooldown := exchangeCooldownSeconds()
	if cooldown <= 0 || state.LastExchangeAt == "" {
		return nil
	}
	lastExchange, err := time.Parse(resourceDateLayout, state.LastExchangeAt)
	if err == nil && now.Sub(lastExchange).Seconds() < float64(cooldown) {
		return ErrExchangeCooldown
	}
	return nil
}

func (s *Service) publishCurrencyChanged(playerID string, accountID string, refID string, refType string) {
	s.publishEvent(GameEvent{
		Type:      EventCurrencyChanged,
		PlayerID:  playerID,
		AccountID: accountID,
		RefType:   refType,
		RefID:     refID,
		CreatedAt: time.Now().UTC().Format(resourceDateLayout),
	})
}

// recordLedger 写一条货币流水。失败时降级为 warn 日志，不影响业务流程。
// 调用方在 repo 余额操作成功后调用，传入操作后的快照余额。
func (s *Service) recordLedger(entry GoldLedgerEntry) {
	if entry.CreatedAt == "" {
		entry.CreatedAt = time.Now().UTC().Format(time.RFC3339)
	}
	// 城金流水若没传 AccountID 则反查（失败也忽略，只是流水缺一项关联信息）
	if entry.Currency == LedgerCurrencyCityGold && entry.AccountID == "" && entry.PlayerID != "" {
		if accountID, err := s.repo.GetAccountIDByPlayerID(entry.PlayerID); err == nil {
			entry.AccountID = accountID
		}
	}
	if err := s.repo.WriteGoldLedger(entry); err != nil {
		slog.Warn("gold ledger write failed",
			"error", err,
			"currency", entry.Currency,
			"direction", entry.Direction,
			"amount", entry.Amount,
			"playerId", entry.PlayerID,
			"accountId", entry.AccountID,
			"refType", entry.RefType,
			"refId", entry.RefID,
		)
	}
}
