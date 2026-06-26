// Hero3 游戏仓储接口和内存仓储实现。
package game

import (
	"errors"
	"strings"
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
	MailAddressExists(nickname string, mailCode string) (bool, error)
	FindPlayerByMailAddress(nickname string, mailCode string) (PlayerSummary, error)
	ListAccounts() ([]AccountSummary, error)
	ListPlayers(accountID string) ([]PlayerSummary, error)
	CreatePlayer(accountID string, state GameState, updatedAt time.Time) error
	DeleteAccount(accountID string) error
	DeletePlayer(playerID string) error
	GetState(playerID string) (GameState, error)
	SaveState(state GameState, updatedAt time.Time) error
	SaveStates(states []GameState, updatedAt time.Time) error
	SaveStateAndCreateMarch(state GameState, march PvpMarch, updatedAt time.Time) error
	SavePvpSettlement(attackerState GameState, defenderState GameState, attackerReport BattleReport, defenderReport BattleReport, march PvpMarch, updatedAt time.Time) error
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

	// Marches
	CreateMarch(march PvpMarch) error
	UpdateMarch(march PvpMarch) error
	GetMarchByID(marchID string) (PvpMarch, error)
	ListMarchesForPlayer(playerID string) ([]PvpMarch, error)
	ListDueMarches(now time.Time, limit int) ([]PvpMarch, error)
	ClaimMarchForResolution(marchID string, now time.Time) (PvpMarch, error)

	// Mails
	SaveMail(mail Mail) error
	GetMailByID(mailID string) (Mail, error)
	ListMails(playerID string, mailType string, limit int, offset int) ([]Mail, int, error)
	CountUnreadMails(playerID string) (int, error)
	MarkMailRead(playerID string, mailID string, readAt time.Time) error
	DeleteMail(playerID string, mailID string) error
	ClaimMailAttachments(playerID string, mailID string, claimedAt time.Time) (MailClaimResult, error)

	// Announcements
	CreateAnnouncement(announcement Announcement) error
	UpdateAnnouncement(announcement Announcement) error
	GetAnnouncementByID(announcementID string) (Announcement, error)
	ListAdminAnnouncements() ([]Announcement, error)
	ListVisibleAnnouncements(playerID string, now time.Time) ([]Announcement, error)
	MarkAnnouncementRead(playerID string, announcementID string, readAt time.Time) error

	// MiniGame Records
	SaveMiniGameRecord(record MiniGameRecord) error
	ListMiniGameRecords(playerID string, gameType string, limit int, offset int, stockOnly bool) ([]MiniGameRecord, int, error)
	RedeemMiniGameRecord(playerID string, recordID string, amount int, redeemedAt time.Time) (MiniGameRedeemResult, error)
	RedeemAllFactionMiniGameRecords(playerID string, gameType string, redeemedAt time.Time) (MiniGameRedeemAllResult, error)

	// Gold Ledger（货币流水，写入失败由调用方降级处理）
	WriteGoldLedger(entry GoldLedgerEntry) error
	ListGoldLedger(filter GoldLedgerFilter) ([]GoldLedgerEntry, error)
	GetAccountIDByPlayerID(playerID string) (string, error)
}

type MemoryRepository struct {
	mu                sync.RWMutex
	accounts          map[string]Account
	accountByName     map[string]string
	accountPlayers    map[string][]string
	players           map[string]GameState
	playerUpdatedAt   map[string]time.Time
	reports           map[string][]BattleReport // playerID → reports
	marches           map[string]PvpMarch
	mails             map[string][]Mail // playerID → mails
	announcements     map[string]Announcement
	announcementReads map[string]map[string]time.Time
	miniGameRecords   map[string][]MiniGameRecord // playerID → records
	ledger            []GoldLedgerEntry
	ledgerNextID      int64
}

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{
		accounts:          make(map[string]Account),
		accountByName:     make(map[string]string),
		accountPlayers:    make(map[string][]string),
		players:           make(map[string]GameState),
		playerUpdatedAt:   make(map[string]time.Time),
		reports:           make(map[string][]BattleReport),
		marches:           make(map[string]PvpMarch),
		mails:             make(map[string][]Mail),
		announcements:     make(map[string]Announcement),
		announcementReads: make(map[string]map[string]time.Time),
		miniGameRecords:   make(map[string][]MiniGameRecord),
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

func (r *MemoryRepository) MailAddressExists(nickname string, mailCode string) (bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, state := range r.players {
		if state.Player.Nickname == nickname && state.Player.MailCode == mailCode {
			return true, nil
		}
	}
	return false, nil
}

func (r *MemoryRepository) FindPlayerByMailAddress(nickname string, mailCode string) (PlayerSummary, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, state := range r.players {
		if state.Player.Nickname == nickname && state.Player.MailCode == mailCode {
			return PlayerSummary{
				ID:       state.Player.ID,
				Nickname: state.Player.Nickname,
				Faction:  state.Player.Faction,
				MailCode: state.Player.MailCode,
			}, nil
		}
	}
	return PlayerSummary{}, ErrPlayerNotFound
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
		delete(r.mails, playerID)
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
	delete(r.mails, playerID)
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

	NormalizeGameState(&state)
	return state, nil
}

func (r *MemoryRepository) SaveState(state GameState, updatedAt time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.players[state.Player.ID]; !exists {
		return ErrPlayerNotFound
	}

	NormalizeGameState(&state)
	r.players[state.Player.ID] = state
	r.playerUpdatedAt[state.Player.ID] = updatedAt
	return nil
}

// SaveStates 批量保存多个玩家状态，用于玩家互攻和增援这类双玩家变更。
func (r *MemoryRepository) SaveStates(states []GameState, updatedAt time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for i := range states {
		if _, exists := r.players[states[i].Player.ID]; !exists {
			return ErrPlayerNotFound
		}
		NormalizeGameState(&states[i])
	}
	for _, state := range states {
		r.players[state.Player.ID] = state
		r.playerUpdatedAt[state.Player.ID] = updatedAt
	}
	return nil
}

func (r *MemoryRepository) SaveStateAndCreateMarch(state GameState, march PvpMarch, updatedAt time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.players[state.Player.ID]; !exists {
		return ErrPlayerNotFound
	}
	if march.ID == "" {
		return ErrInvalidMarch
	}
	NormalizeGameState(&state)
	r.players[state.Player.ID] = state
	r.playerUpdatedAt[state.Player.ID] = updatedAt
	r.marches[march.ID] = march
	return nil
}

func (r *MemoryRepository) SavePvpSettlement(attackerState GameState, defenderState GameState, attackerReport BattleReport, defenderReport BattleReport, march PvpMarch, updatedAt time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.players[attackerState.Player.ID]; !exists {
		return ErrPlayerNotFound
	}
	if _, exists := r.players[defenderState.Player.ID]; !exists {
		return ErrPlayerNotFound
	}
	if _, exists := r.marches[march.ID]; !exists {
		return ErrMarchNotFound
	}
	NormalizeGameState(&attackerState)
	NormalizeGameState(&defenderState)
	r.players[attackerState.Player.ID] = attackerState
	r.players[defenderState.Player.ID] = defenderState
	r.playerUpdatedAt[attackerState.Player.ID] = updatedAt
	r.playerUpdatedAt[defenderState.Player.ID] = updatedAt
	r.reports[attackerReport.PlayerID] = append([]BattleReport{attackerReport}, r.reports[attackerReport.PlayerID]...)
	r.reports[defenderReport.PlayerID] = append([]BattleReport{defenderReport}, r.reports[defenderReport.PlayerID]...)
	r.marches[march.ID] = march
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

// --- March Methods (MemoryRepository) ---

func (r *MemoryRepository) CreateMarch(march PvpMarch) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if march.ID == "" {
		return ErrInvalidMarch
	}
	r.marches[march.ID] = march
	return nil
}

func (r *MemoryRepository) UpdateMarch(march PvpMarch) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.marches[march.ID]; !exists {
		return ErrMarchNotFound
	}
	r.marches[march.ID] = march
	return nil
}

func (r *MemoryRepository) GetMarchByID(marchID string) (PvpMarch, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	march, exists := r.marches[marchID]
	if !exists {
		return PvpMarch{}, ErrMarchNotFound
	}
	return march, nil
}

func (r *MemoryRepository) ListMarchesForPlayer(playerID string) ([]PvpMarch, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	marches := []PvpMarch{}
	for _, march := range r.marches {
		if march.AttackerPlayerID == playerID || march.DefenderPlayerID == playerID {
			marches = append(marches, march)
		}
	}
	sortMarches(marches)
	return marches, nil
}

func (r *MemoryRepository) ListDueMarches(now time.Time, limit int) ([]PvpMarch, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if limit <= 0 {
		limit = 100
	}
	marches := []PvpMarch{}
	for _, march := range r.marches {
		if march.Status != MarchStatusMarching {
			continue
		}
		arrivesAt, ok := parseMarchTime(march.ArrivesAt)
		if !ok || arrivesAt.After(now.UTC()) {
			continue
		}
		marches = append(marches, march)
	}
	sortMarchesByArrival(marches)
	if len(marches) > limit {
		marches = marches[:limit]
	}
	return marches, nil
}

func (r *MemoryRepository) ClaimMarchForResolution(marchID string, now time.Time) (PvpMarch, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	march, exists := r.marches[marchID]
	if !exists {
		return PvpMarch{}, ErrMarchNotFound
	}
	if march.Status != MarchStatusMarching {
		return PvpMarch{}, ErrMarchNotDue
	}
	arrivesAt, ok := parseMarchTime(march.ArrivesAt)
	if !ok || arrivesAt.After(now.UTC()) {
		return PvpMarch{}, ErrMarchNotDue
	}
	march.Status = MarchStatusResolving
	march.UpdatedAt = now.UTC().Format(resourceDateLayout)
	r.marches[march.ID] = march
	return march, nil
}

// --- Mail Methods (MemoryRepository) ---

func (r *MemoryRepository) SaveMail(mail Mail) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.mails[mail.PlayerID] = append([]Mail{mail}, r.mails[mail.PlayerID]...)
	if len(r.mails[mail.PlayerID]) > 1000 {
		r.mails[mail.PlayerID] = r.mails[mail.PlayerID][:1000]
	}
	return nil
}

func (r *MemoryRepository) GetMailByID(mailID string) (Mail, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, mails := range r.mails {
		for _, mail := range mails {
			if mail.ID == mailID {
				return mail, nil
			}
		}
	}
	return Mail{}, errors.New("mail not found")
}

func (r *MemoryRepository) ListMails(playerID string, mailType string, limit int, offset int) ([]Mail, int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	mailType = strings.TrimSpace(mailType)
	all := r.mails[playerID]
	total := 0
	result := []Mail{}
	for _, mail := range all {
		if mail.DeletedByPlayer {
			continue
		}
		if mailType != "" && mail.MailType != mailType {
			continue
		}
		total++
		if offset > 0 {
			offset--
			continue
		}
		result = append(result, mail)
		if limit > 0 && len(result) >= limit {
			break
		}
	}
	return result, total, nil
}

func (r *MemoryRepository) CountUnreadMails(playerID string) (int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	count := 0
	for _, mail := range r.mails[playerID] {
		if !mail.IsRead && !mail.DeletedByPlayer {
			count++
		}
	}
	return count, nil
}

func (r *MemoryRepository) MarkMailRead(playerID string, mailID string, readAt time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	readAtText := readAt.UTC().Format(resourceDateLayout)
	for i := range r.mails[playerID] {
		if r.mails[playerID][i].ID == mailID {
			r.mails[playerID][i].IsRead = true
			if r.mails[playerID][i].ReadAt == "" {
				r.mails[playerID][i].ReadAt = readAtText
			}
			return nil
		}
	}
	return errors.New("mail not found")
}

func (r *MemoryRepository) DeleteMail(playerID string, mailID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for i := range r.mails[playerID] {
		if r.mails[playerID][i].ID == mailID {
			r.mails[playerID][i].DeletedByPlayer = true
			return nil
		}
	}
	return errors.New("mail not found")
}

func (r *MemoryRepository) ClaimMailAttachments(playerID string, mailID string, claimedAt time.Time) (MailClaimResult, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	state, exists := r.players[playerID]
	if !exists {
		return MailClaimResult{}, ErrPlayerNotFound
	}
	if state.Resources.Items == nil {
		state.Resources.Items = map[string]int{}
	}

	for i := range r.mails[playerID] {
		mail := r.mails[playerID][i]
		if mail.ID != mailID || mail.DeletedByPlayer {
			continue
		}
		if len(mail.Attachments) == 0 {
			return MailClaimResult{}, ErrMailNoAttachments
		}
		if mail.IsClaimed {
			return MailClaimResult{}, ErrMailAlreadyClaimed
		}
		granted, accountGold, err := ApplyMailAttachmentsToState(&state, mail.Attachments)
		if err != nil {
			return MailClaimResult{}, err
		}
		if accountGold > 0 {
			accountID := ""
			for candidateAccountID, playerIDs := range r.accountPlayers {
				for _, candidatePlayerID := range playerIDs {
					if candidatePlayerID == playerID {
						accountID = candidateAccountID
						break
					}
				}
				if accountID != "" {
					break
				}
			}
			if accountID == "" {
				return MailClaimResult{}, ErrAccountNotFound
			}
			account := r.accounts[accountID]
			account.Gold += accountGold
			r.accounts[accountID] = account
		}
		mail.IsClaimed = true
		mail.ClaimedAt = claimedAt.UTC().Format(resourceDateLayout)
		r.mails[playerID][i] = mail
		r.players[playerID] = state
		return MailClaimResult{
			Mail:         mail,
			Resources:    state.Resources,
			Inventory:    state.Inventory,
			CityGold:     int(state.CityGold),
			AccountGold:  accountGold,
			GrantedItems: granted,
		}, nil
	}
	return MailClaimResult{}, ErrMailNotFound
}

// --- Announcement Methods (MemoryRepository) ---

func (r *MemoryRepository) CreateAnnouncement(announcement Announcement) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if announcement.ID == "" {
		return ErrInvalidAnnouncement
	}
	r.announcements[announcement.ID] = announcement
	return nil
}

func (r *MemoryRepository) UpdateAnnouncement(announcement Announcement) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.announcements[announcement.ID]; !exists {
		return ErrAnnouncementNotFound
	}
	r.announcements[announcement.ID] = announcement
	return nil
}

func (r *MemoryRepository) GetAnnouncementByID(announcementID string) (Announcement, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	announcement, exists := r.announcements[announcementID]
	if !exists {
		return Announcement{}, ErrAnnouncementNotFound
	}
	return announcement, nil
}

func (r *MemoryRepository) ListAdminAnnouncements() ([]Announcement, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	items := make([]Announcement, 0, len(r.announcements))
	for _, announcement := range r.announcements {
		items = append(items, announcement)
	}
	sortAnnouncements(items)
	return items, nil
}

func (r *MemoryRepository) ListVisibleAnnouncements(playerID string, now time.Time) ([]Announcement, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	readMap := r.announcementReads[playerID]
	items := []Announcement{}
	for _, announcement := range r.announcements {
		if !isAnnouncementVisible(announcement, now) {
			continue
		}
		if _, ok := readMap[announcement.ID]; ok {
			announcement.Read = true
		}
		items = append(items, announcement)
	}
	sortAnnouncements(items)
	return items, nil
}

func (r *MemoryRepository) MarkAnnouncementRead(playerID string, announcementID string, readAt time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.announcements[announcementID]; !exists {
		return ErrAnnouncementNotFound
	}
	if r.announcementReads[playerID] == nil {
		r.announcementReads[playerID] = map[string]time.Time{}
	}
	r.announcementReads[playerID][announcementID] = readAt.UTC()
	return nil
}

// --- MiniGame Record Methods (MemoryRepository) ---

func (r *MemoryRepository) SaveMiniGameRecord(record MiniGameRecord) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if record.GameType == "fishing" && record.RemainingAmount == 0 && record.RewardAmount > 0 {
		record.RemainingAmount = record.RewardAmount
	}

	r.miniGameRecords[record.PlayerID] = append([]MiniGameRecord{record}, r.miniGameRecords[record.PlayerID]...)
	// 保留最多 500 条
	if len(r.miniGameRecords[record.PlayerID]) > 500 {
		r.miniGameRecords[record.PlayerID] = r.miniGameRecords[record.PlayerID][:500]
	}
	return nil
}

// ListMiniGameRecords 按条件分页查询小游戏记录，stockOnly 只返回可兑换库存。
func (r *MemoryRepository) ListMiniGameRecords(playerID string, gameType string, limit int, offset int, stockOnly bool) ([]MiniGameRecord, int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	all := r.miniGameRecords[playerID]
	if gameType != "" || stockOnly {
		filtered := make([]MiniGameRecord, 0, len(all))
		for _, record := range all {
			if gameType != "" && record.GameType != gameType {
				continue
			}
			if stockOnly && (record.RewardUnit == "" || record.RemainingAmount <= 0) {
				continue
			}
			filtered = append(filtered, record)
		}
		all = filtered
	}
	total := len(all)
	if offset < 0 {
		offset = 0
	}
	if offset >= total {
		return []MiniGameRecord{}, total, nil
	}
	end := total
	if limit > 0 && offset+limit < end {
		end = offset + limit
	}
	return all[offset:end], total, nil
}

func (r *MemoryRepository) RedeemMiniGameRecord(playerID string, recordID string, amount int, redeemedAt time.Time) (MiniGameRedeemResult, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	state, exists := r.players[playerID]
	if !exists {
		return MiniGameRedeemResult{}, ErrPlayerNotFound
	}
	records := r.miniGameRecords[playerID]
	for i := range records {
		record := records[i]
		if record.ID != recordID {
			continue
		}
		if record.GameType != "fishing" || record.RewardUnit == "" || record.RewardAmount <= 0 {
			return MiniGameRedeemResult{}, ErrInvalidMiniGame
		}
		if record.RemainingAmount <= 0 {
			return MiniGameRedeemResult{}, ErrMiniGameStockShort
		}
		if amount > record.RemainingAmount {
			return MiniGameRedeemResult{}, ErrMiniGameStockShort
		}
		_, unitID, unitCfg, ok := FindUnitByName(record.RewardUnit)
		if !ok {
			return MiniGameRedeemResult{}, ErrCrossFactionReward
		}
		record.RemainingAmount -= amount
		records[i] = record
		r.miniGameRecords[playerID] = records
		AddOwnedRewardUnit(&state, unitID, amount)
		state.ServerTime = redeemedAt.UTC().Format(resourceDateLayout)
		r.players[playerID] = state
		r.playerUpdatedAt[playerID] = redeemedAt.UTC()
		return MiniGameRedeemResult{
			Record:         record,
			State:          state,
			RedeemedUnitID: unitID,
			RedeemedUnit:   unitCfg.Name,
			RedeemedAmount: amount,
		}, nil
	}
	return MiniGameRedeemResult{}, ErrMiniGameNotFound
}

func (r *MemoryRepository) RedeemAllFactionMiniGameRecords(playerID string, gameType string, redeemedAt time.Time) (MiniGameRedeemAllResult, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	state, exists := r.players[playerID]
	if !exists {
		return MiniGameRedeemAllResult{}, ErrPlayerNotFound
	}
	if gameType == "" {
		gameType = "fishing"
	}

	records := r.miniGameRecords[playerID]
	redeemedUnits := map[string]int{}
	skippedUnits := map[string]int{}
	redeemedRecords := 0
	skippedRecords := 0
	for i := range records {
		record := records[i]
		if record.GameType != gameType || record.RewardUnit == "" || record.RemainingAmount <= 0 {
			continue
		}
		_, unitID, unitCfg, ok := FindUnitByName(record.RewardUnit)
		if !ok {
			skippedUnits[record.RewardUnit] += record.RemainingAmount
			skippedRecords++
			continue
		}
		amount := record.RemainingAmount
		records[i].RemainingAmount = 0
		AddOwnedRewardUnit(&state, unitID, amount)
		redeemedUnits[unitCfg.Name] += amount
		redeemedRecords++
	}
	if redeemedRecords == 0 {
		state.ServerTime = redeemedAt.UTC().Format(resourceDateLayout)
		return MiniGameRedeemAllResult{
			State:          state,
			RedeemedUnits:  redeemedUnits,
			SkippedUnits:   skippedUnits,
			SkippedRecords: skippedRecords,
		}, nil
	}

	state.ServerTime = redeemedAt.UTC().Format(resourceDateLayout)
	r.miniGameRecords[playerID] = records
	r.players[playerID] = state
	r.playerUpdatedAt[playerID] = redeemedAt.UTC()
	total := 0
	for _, amount := range redeemedUnits {
		total += amount
	}
	return MiniGameRedeemAllResult{
		State:           state,
		RedeemedUnits:   redeemedUnits,
		RedeemedAmount:  total,
		RedeemedRecords: redeemedRecords,
		SkippedUnits:    skippedUnits,
		SkippedRecords:  skippedRecords,
	}, nil
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
