package game

import (
	"errors"
	"sync"
	"time"
)

type Repository interface {
	CreateAccount(account Account) error
	GetAccountByUsername(username string) (Account, error)
	GetAccountByID(accountID string) (Account, error)
	UpdateAccountGold(accountID string, gold int) error
	AddAccountGold(accountID string, amount int) error
	DeductAccountGold(accountID string, amount int) error
	AccountExists(accountID string) (bool, error)
	ListAccounts() ([]AccountSummary, error)
	ListPlayers(accountID string) ([]PlayerSummary, error)
	CreatePlayer(accountID string, state GameState, updatedAt time.Time) error
	DeleteAccount(accountID string) error
	DeletePlayer(playerID string) error
	GetState(playerID string) (GameState, error)
	SaveState(state GameState, updatedAt time.Time) error
	// 城金原子操作
	AddCityGold(playerID string, amount int) (int, error)    // 返回操作后余额
	DeductCityGold(playerID string, amount int) (int, error) // 余额不足返回 ErrInsufficientCityGold

	// 金币兑换事务操作（保证原子性）
	ExchangeGoldToCityGold(accountID string, playerID string, goldAmount int, cityGoldGain int) error
	ExchangeCityGoldToGold(accountID string, playerID string, cityGoldAmount int, goldGain int) error

	// Battle Reports
	SaveReport(report BattleReport) error
	GetReportByID(reportID string) (BattleReport, error)
	ListReports(playerID string, limit int, offset int) ([]BattleReport, int, error)
	ListAllReports(playerID string) ([]BattleReport, error)
	MarkReportsRead(playerID string) error
	MarkSingleReportRead(playerID string, reportID string) error
	DeleteReport(playerID string, reportID string) error
	DeleteAllReports(playerID string) error
	CountUnreadReports(playerID string) (int, error)

	// MiniGame Records
	SaveMiniGameRecord(record MiniGameRecord) error
	ListMiniGameRecords(playerID string, limit int) ([]MiniGameRecord, error)

	// Gold Ledger（货币流水，写入失败由调用方降级处理）
	WriteGoldLedger(entry GoldLedgerEntry) error
	ListGoldLedger(filter GoldLedgerFilter) ([]GoldLedgerEntry, error)
	GetAccountIDByPlayerID(playerID string) (string, error)
}

type MemoryRepository struct {
	mu              sync.RWMutex
	accounts        map[string]Account
	accountByName   map[string]string
	accountPlayers  map[string][]string
	players         map[string]GameState
	playerUpdatedAt map[string]time.Time
	reports         map[string][]BattleReport   // playerID → reports
	miniGameRecords map[string][]MiniGameRecord // playerID → records
	ledger          []GoldLedgerEntry
	ledgerNextID    int64
}

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{
		accounts:        make(map[string]Account),
		accountByName:   make(map[string]string),
		accountPlayers:  make(map[string][]string),
		players:         make(map[string]GameState),
		playerUpdatedAt: make(map[string]time.Time),
		reports:         make(map[string][]BattleReport),
		miniGameRecords: make(map[string][]MiniGameRecord),
	}
}

func (r *MemoryRepository) CreateAccount(account Account) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.accountByName[account.Username]; exists {
		return ErrAccountExists
	}

	r.accounts[account.ID] = account
	r.accountByName[account.Username] = account.ID
	r.accountPlayers[account.ID] = []string{}
	return nil
}

func (r *MemoryRepository) GetAccountByUsername(username string) (Account, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	accountID, exists := r.accountByName[username]
	if !exists {
		return Account{}, ErrAccountNotFound
	}

	return r.accounts[accountID], nil
}

func (r *MemoryRepository) GetAccountByID(accountID string) (Account, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	account, exists := r.accounts[accountID]
	if !exists {
		return Account{}, ErrAccountNotFound
	}
	return account, nil
}

func (r *MemoryRepository) UpdateAccountGold(accountID string, gold int) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	account, exists := r.accounts[accountID]
	if !exists {
		return ErrAccountNotFound
	}
	account.Gold = gold
	r.accounts[accountID] = account
	return nil
}

func (r *MemoryRepository) AddAccountGold(accountID string, amount int) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	account, exists := r.accounts[accountID]
	if !exists {
		return ErrAccountNotFound
	}
	account.Gold += amount
	r.accounts[accountID] = account
	return nil
}

func (r *MemoryRepository) DeductAccountGold(accountID string, amount int) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	account, exists := r.accounts[accountID]
	if !exists {
		return ErrAccountNotFound
	}
	if account.Gold < amount {
		return ErrInsufficientGold
	}
	account.Gold -= amount
	r.accounts[accountID] = account
	return nil
}

func (r *MemoryRepository) AddCityGold(playerID string, amount int) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	state, exists := r.players[playerID]
	if !exists {
		return 0, ErrPlayerNotFound
	}
	state.CityGold += FlexInt(amount)
	r.players[playerID] = state
	return int(state.CityGold), nil
}

func (r *MemoryRepository) DeductCityGold(playerID string, amount int) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	state, exists := r.players[playerID]
	if !exists {
		return 0, ErrPlayerNotFound
	}
	if int(state.CityGold) < amount {
		return 0, ErrInsufficientCityGold
	}
	state.CityGold -= FlexInt(amount)
	r.players[playerID] = state
	return int(state.CityGold), nil
}

func (r *MemoryRepository) ExchangeGoldToCityGold(accountID string, playerID string, goldAmount int, cityGoldGain int) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	account, exists := r.accounts[accountID]
	if !exists {
		return ErrAccountNotFound
	}
	if account.Gold < goldAmount {
		return ErrInsufficientGold
	}

	state, exists := r.players[playerID]
	if !exists {
		return ErrPlayerNotFound
	}

	// 同一把锁内完成，天然原子
	account.Gold -= goldAmount
	r.accounts[accountID] = account
	state.CityGold += FlexInt(cityGoldGain)
	r.players[playerID] = state
	return nil
}

func (r *MemoryRepository) ExchangeCityGoldToGold(accountID string, playerID string, cityGoldAmount int, goldGain int) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	state, exists := r.players[playerID]
	if !exists {
		return ErrPlayerNotFound
	}
	if int(state.CityGold) < cityGoldAmount {
		return ErrInsufficientCityGold
	}

	account, exists := r.accounts[accountID]
	if !exists {
		return ErrAccountNotFound
	}

	// 同一把锁内完成，天然原子
	state.CityGold -= FlexInt(cityGoldAmount)
	r.players[playerID] = state
	account.Gold += goldGain
	r.accounts[accountID] = account
	return nil
}

func (r *MemoryRepository) AccountExists(accountID string) (bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	_, exists := r.accounts[accountID]
	return exists, nil
}

func (r *MemoryRepository) ListAccounts() ([]AccountSummary, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	accounts := make([]AccountSummary, 0, len(r.accounts))
	for _, account := range r.accounts {
		playerIDs := r.accountPlayers[account.ID]
		players := make([]PlayerSummary, 0, len(playerIDs))
		for _, playerID := range playerIDs {
			state, exists := r.players[playerID]
			if !exists {
				continue
			}
			players = append(players, buildPlayerSummary(state, r.playerUpdatedAt[playerID]))
		}

		accounts = append(accounts, AccountSummary{
			ID:        account.ID,
			Username:  account.Username,
			CreatedAt: account.CreatedAt.UTC().Format(time.RFC3339),
			Players:   players,
		})
	}

	return accounts, nil
}

func (r *MemoryRepository) ListPlayers(accountID string) ([]PlayerSummary, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if _, exists := r.accounts[accountID]; !exists {
		return nil, ErrAccountNotFound
	}

	playerIDs := r.accountPlayers[accountID]
	players := make([]PlayerSummary, 0, len(playerIDs))
	for _, playerID := range playerIDs {
		state, exists := r.players[playerID]
		if !exists {
			continue
		}
		players = append(players, buildPlayerSummary(state, r.playerUpdatedAt[playerID]))
	}

	return players, nil
}

func (r *MemoryRepository) CreatePlayer(accountID string, state GameState, updatedAt time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.accounts[accountID]; !exists {
		return ErrAccountNotFound
	}

	r.players[state.Player.ID] = state
	r.playerUpdatedAt[state.Player.ID] = updatedAt
	r.accountPlayers[accountID] = append(r.accountPlayers[accountID], state.Player.ID)
	return nil
}

func (r *MemoryRepository) DeleteAccount(accountID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	account, exists := r.accounts[accountID]
	if !exists {
		return ErrAccountNotFound
	}

	for _, playerID := range r.accountPlayers[accountID] {
		delete(r.players, playerID)
		delete(r.playerUpdatedAt, playerID)
		delete(r.reports, playerID)
	}
	delete(r.accountPlayers, accountID)
	delete(r.accountByName, account.Username)
	delete(r.accounts, accountID)
	return nil
}

func (r *MemoryRepository) DeletePlayer(playerID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.players[playerID]; !exists {
		return ErrPlayerNotFound
	}

	delete(r.players, playerID)
	delete(r.playerUpdatedAt, playerID)
	delete(r.reports, playerID)
	for accountID, playerIDs := range r.accountPlayers {
		nextPlayerIDs := playerIDs[:0]
		for _, currentID := range playerIDs {
			if currentID != playerID {
				nextPlayerIDs = append(nextPlayerIDs, currentID)
			}
		}
		r.accountPlayers[accountID] = nextPlayerIDs
	}
	return nil
}

func (r *MemoryRepository) GetState(playerID string) (GameState, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	state, exists := r.players[playerID]
	if !exists {
		return GameState{}, ErrPlayerNotFound
	}

	return state, nil
}

func (r *MemoryRepository) SaveState(state GameState, updatedAt time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.players[state.Player.ID]; !exists {
		return ErrPlayerNotFound
	}

	r.players[state.Player.ID] = state
	r.playerUpdatedAt[state.Player.ID] = updatedAt
	return nil
}

// --- Battle Report Methods (MemoryRepository) ---

func (r *MemoryRepository) SaveReport(report BattleReport) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.reports[report.PlayerID] = append([]BattleReport{report}, r.reports[report.PlayerID]...)
	// 保留最多 1000 条
	if len(r.reports[report.PlayerID]) > 1000 {
		r.reports[report.PlayerID] = r.reports[report.PlayerID][:1000]
	}
	return nil
}

func (r *MemoryRepository) GetReportByID(reportID string) (BattleReport, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, reports := range r.reports {
		for _, report := range reports {
			if report.ID == reportID {
				return report, nil
			}
		}
	}
	return BattleReport{}, errors.New("report not found")
}

func (r *MemoryRepository) ListReports(playerID string, limit int, offset int) ([]BattleReport, int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	all := r.reports[playerID]
	var result []BattleReport
	total := 0
	threeDaysAgo := time.Now().Add(-3 * 24 * time.Hour)

	for _, report := range all {
		if report.DeletedByPlayer {
			continue
		}
		createdAt, err := time.Parse(time.RFC3339, report.CreatedAt)
		if err == nil && createdAt.Before(threeDaysAgo) {
			continue
		}
		total++
		if offset > 0 {
			offset--
			continue
		}
		result = append(result, report)
		if limit > 0 && len(result) >= limit {
			break
		}
	}
	return result, total, nil
}

func (r *MemoryRepository) ListAllReports(playerID string) ([]BattleReport, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.reports[playerID], nil
}

func (r *MemoryRepository) MarkReportsRead(playerID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for i := range r.reports[playerID] {
		r.reports[playerID][i].Read = true
	}
	return nil
}

func (r *MemoryRepository) MarkSingleReportRead(playerID string, reportID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for i := range r.reports[playerID] {
		if r.reports[playerID][i].ID == reportID {
			r.reports[playerID][i].Read = true
			return nil
		}
	}
	return nil
}

func (r *MemoryRepository) DeleteReport(playerID string, reportID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for i := range r.reports[playerID] {
		if r.reports[playerID][i].ID == reportID {
			r.reports[playerID][i].DeletedByPlayer = true
			return nil
		}
	}
	return nil
}

func (r *MemoryRepository) DeleteAllReports(playerID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for i := range r.reports[playerID] {
		r.reports[playerID][i].DeletedByPlayer = true
	}
	return nil
}

func (r *MemoryRepository) CountUnreadReports(playerID string) (int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	count := 0
	threeDaysAgo := time.Now().Add(-3 * 24 * time.Hour)
	for _, report := range r.reports[playerID] {
		if report.Read || report.DeletedByPlayer {
			continue
		}
		createdAt, err := time.Parse(time.RFC3339, report.CreatedAt)
		if err == nil && createdAt.Before(threeDaysAgo) {
			continue
		}
		if !report.Read && !report.DeletedByPlayer {
			count++
		}
	}
	return count, nil
}

// --- MiniGame Record Methods (MemoryRepository) ---

func (r *MemoryRepository) SaveMiniGameRecord(record MiniGameRecord) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.miniGameRecords[record.PlayerID] = append([]MiniGameRecord{record}, r.miniGameRecords[record.PlayerID]...)
	// 保留最多 500 条
	if len(r.miniGameRecords[record.PlayerID]) > 500 {
		r.miniGameRecords[record.PlayerID] = r.miniGameRecords[record.PlayerID][:500]
	}
	return nil
}

func (r *MemoryRepository) ListMiniGameRecords(playerID string, limit int) ([]MiniGameRecord, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	all := r.miniGameRecords[playerID]
	if limit > 0 && len(all) > limit {
		return all[:limit], nil
	}
	return all, nil
}

// --- Gold Ledger Methods (MemoryRepository) ---

func (r *MemoryRepository) WriteGoldLedger(entry GoldLedgerEntry) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.ledgerNextID++
	entry.ID = r.ledgerNextID
	if entry.CreatedAt == "" {
		entry.CreatedAt = time.Now().UTC().Format(time.RFC3339)
	}
	r.ledger = append(r.ledger, entry)
	return nil
}

func (r *MemoryRepository) ListGoldLedger(filter GoldLedgerFilter) ([]GoldLedgerEntry, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	limit := filter.Limit
	if limit <= 0 || limit > 1000 {
		limit = 200
	}

	matches := make([]GoldLedgerEntry, 0, limit)
	for i := len(r.ledger) - 1; i >= 0 && len(matches) < limit; i-- {
		entry := r.ledger[i]
		if filter.AccountID != "" && entry.AccountID != filter.AccountID {
			continue
		}
		if filter.PlayerID != "" && entry.PlayerID != filter.PlayerID {
			continue
		}
		if filter.Currency != "" && entry.Currency != filter.Currency {
			continue
		}
		if filter.RefType != "" && entry.RefType != filter.RefType {
			continue
		}
		if !filter.From.IsZero() || !filter.To.IsZero() {
			createdAt, err := time.Parse(time.RFC3339, entry.CreatedAt)
			if err != nil {
				continue
			}
			if !filter.From.IsZero() && createdAt.Before(filter.From) {
				continue
			}
			if !filter.To.IsZero() && createdAt.After(filter.To) {
				continue
			}
		}
		matches = append(matches, entry)
	}
	return matches, nil
}

func (r *MemoryRepository) GetAccountIDByPlayerID(playerID string) (string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for accountID, playerIDs := range r.accountPlayers {
		for _, id := range playerIDs {
			if id == playerID {
				return accountID, nil
			}
		}
	}
	return "", ErrPlayerNotFound
}
