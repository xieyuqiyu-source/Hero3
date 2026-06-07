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
	newBalance, err := s.repo.AddCityGold(playerID, amount)
	if err != nil {
		return GameState{}, err
	}

	s.recordLedger(GoldLedgerEntry{
		PlayerID:     playerID,
		Currency:     LedgerCurrencyCityGold,
		Direction:    LedgerDirectionCredit,
		Amount:       amount,
		BalanceAfter: newBalance,
		RefType:      LedgerRefAdminAdjust,
		Reason:       reason,
	})

	// 读取最新状态返回
	state, err := s.repo.GetState(playerID)
	if err != nil {
		return GameState{}, err
	}

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
	newBalance, err := s.repo.DeductCityGold(playerID, amount)
	if err != nil {
		return GameState{}, err
	}

	s.recordLedger(GoldLedgerEntry{
		PlayerID:     playerID,
		Currency:     LedgerCurrencyCityGold,
		Direction:    LedgerDirectionDebit,
		Amount:       amount,
		BalanceAfter: newBalance,
		RefType:      LedgerRefAdminAdjust,
		Reason:       reason,
	})

	state, err := s.repo.GetState(playerID)
	if err != nil {
		return GameState{}, err
	}

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

	// 加锁防止并发兑换绕过冷却
	lock := s.getPlayerLock(playerID)
	lock.Lock()
	defer lock.Unlock()

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

	refID := "exchange_" + randomID(10)

	// 流水：金币支出
	account, _ := s.repo.GetAccountByID(accountID)
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
	// 流水：城金收入
	stateAfter, _ := s.repo.GetState(playerID)
	s.recordLedger(GoldLedgerEntry{
		AccountID:    accountID,
		PlayerID:     playerID,
		Currency:     LedgerCurrencyCityGold,
		Direction:    LedgerDirectionCredit,
		Amount:       cityGoldGain,
		BalanceAfter: int(stateAfter.CityGold),
		RefType:      LedgerRefExchange,
		RefID:        refID,
	})

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

	// 加锁防止并发兑换绕过冷却
	lock := s.getPlayerLock(playerID)
	lock.Lock()
	defer lock.Unlock()

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

	refID := "exchange_" + randomID(10)

	// 流水：城金支出
	stateAfter, _ := s.repo.GetState(playerID)
	s.recordLedger(GoldLedgerEntry{
		AccountID:    accountID,
		PlayerID:     playerID,
		Currency:     LedgerCurrencyCityGold,
		Direction:    LedgerDirectionDebit,
		Amount:       cityGoldAmount,
		BalanceAfter: int(stateAfter.CityGold),
		RefType:      LedgerRefExchange,
		RefID:        refID,
	})
	// 流水：金币收入
	account, _ := s.repo.GetAccountByID(accountID)
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

	// 更新冷却时间
	state, _ = s.repo.GetState(playerID)
	state.LastExchangeAt = now.UTC().Format(resourceDateLayout)
	state.ServerTime = now.UTC().Format(resourceDateLayout)
	_ = s.repo.SaveState(state, now)

	// 返回最新状态
	state, _ = s.repo.GetState(playerID)
	return state, nil
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
