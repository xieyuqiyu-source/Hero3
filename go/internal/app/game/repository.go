// 本文件定义游戏服务使用的仓储接口和内存仓储基础结构。
package game

import (
	"encoding/json"
	"errors"
	"sort"
	"strings"
	"sync"
	"time"
)

type AccountRepository interface {
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
	ListAllPlayers() ([]PlayerSummary, error)
	GetAccountIDByPlayerID(playerID string) (string, error)
}

const battleReportVisibleCapPerView = 1000

type PlayerStateRepository interface {
	CreatePlayer(accountID string, state GameState, updatedAt time.Time) error
	DeleteAccount(accountID string) error
	DeletePlayer(playerID string) error
	GetState(playerID string) (GameState, error)
	UpdatePlayerState(playerID string, updatedAt time.Time, update func(state *GameState) error) (GameState, error)
}

type PlayerMetaRepository interface {
	UpdatePlayerMetaState(playerID string, updatedAt time.Time, update func(state *GameState) error) (GameState, error)
}

type NpcStateRepository interface {
	UpdateNpcState(playerID string, updatedAt time.Time, update func(state *GameState) error) (GameState, error)
}

type BuildingAssetRepository interface {
	UpdateBuildingResourceState(playerID string, updatedAt time.Time, update func(state *GameState) error) (GameState, error)
	UpdateAccountBuildingResourceState(accountID string, playerID string, updatedAt time.Time, update func(account *Account, state *GameState) error) (Account, GameState, error)
}

type RecruitAssetRepository interface {
	UpdateRecruitState(playerID string, updatedAt time.Time, update func(state *GameState) error) (GameState, error)
}

type ResourceAssetRepository interface {
	UpdateResourceState(playerID string, updatedAt time.Time, update func(state *GameState) error) (GameState, error)
}

type ItemAssetRepository interface {
	UpdateItemState(playerID string, updatedAt time.Time, update func(state *GameState) error) (GameState, error)
	UpdateGeneralExpItemState(playerID string, itemID string, updatedAt time.Time, update func(state *GameState) error) (GameState, error)
}

type GeneralAssetRepository interface {
	UpdateGeneralState(playerID string, updatedAt time.Time, update func(state *GameState) error) (GameState, error)
	UpdateAccountGeneralState(accountID string, playerID string, updatedAt time.Time, update func(account *Account, state *GameState) error) (Account, GameState, error)
}

// CombatAssetScope 描述一次战斗事务已知需要锁定的资产范围。
type CombatAssetScope struct {
	UnitTypes        []string
	GeneralIDs       []string
	InventoryItemIDs []string
	SkipInventory    bool
}

type CombatAssetRepository interface {
	UpdateCombatState(playerID string, scope CombatAssetScope, updatedAt time.Time, update func(state *GameState) error) (GameState, error)
}

type RewardAssetRepository interface {
	UpdateRewardState(playerID string, updatedAt time.Time, update func(state *GameState) error) (GameState, error)
	UpdateScopedRewardState(playerID string, scope RewardAssetScope, updatedAt time.Time, update func(state *GameState) error) (GameState, error)
	UpdateAccountRewardState(accountID string, playerID string, updatedAt time.Time, update func(account *Account, state *GameState) error) (Account, GameState, error)
	UpdateScopedAccountRewardState(accountID string, playerID string, scope RewardAssetScope, updatedAt time.Time, update func(account *Account, state *GameState) error) (Account, GameState, error)
}

// RewardAssetScope 描述奖励发放事务需要锁定和写回的资产范围。
type RewardAssetScope struct {
	Resources        bool
	Currency         bool
	AllInventory     bool
	InventoryItemIDs []string
	AllArmy          bool
	UnitTypes        []string
	AllGenerals      bool
	GeneralIDs       []string
	CurrentGeneral   bool
	Buffs            bool
}

type AccountAssetRepository interface {
	UpdateAccountPlayerState(accountID string, playerID string, updatedAt time.Time, update func(account *Account, state *GameState) error) (Account, GameState, error)
}

type ReportRepository interface {
	SaveReport(report BattleReport) error
	SaveReports(reports []BattleReport) error
	GetReportByID(reportID string) (BattleReport, error)
	GetReportForPlayer(playerID string, reportID string) (BattleReport, error)
	GetReportByShareToken(token string) (BattleReport, error)
	ListReports(playerID string, limit int, offset int) ([]BattleReport, int, error)
	ListReportsByQuery(query BattleReportQuery) ([]BattleReport, int, error)
	ListAllReports(playerID string) ([]BattleReport, error)
	MarkReportsRead(playerID string) error
	MarkReportsReadByView(playerID string, viewType string) error
	MarkSingleReportRead(playerID string, reportID string) error
	DeleteReport(playerID string, reportID string) error
	DeleteReportsByView(playerID string, viewType string) error
	DeleteAllReports(playerID string) error
	CreateBattleReportShareLink(playerID string, reportID string, visibility string, expiresAt time.Time) (BattleReportShareLink, error)
	ListBattleEventsForAdmin(query BattleEventQuery) ([]BattleEvent, int, error)
	GetBattleEventForAdmin(eventID string) (BattleEvent, error)
	ListReportsByEventForAdmin(eventID string) ([]BattleReport, error)
	ListParticipantsByEventForAdmin(eventID string) ([]BattleReportParticipant, error)
	CountUnreadReports(playerID string) (int, error)
}

type MailRepository interface {
	SaveMail(mail Mail) error
	GetMailByID(mailID string) (Mail, error)
	ListMails(playerID string, mailType string, limit int, offset int) ([]Mail, int, error)
	CountUnreadMails(playerID string) (int, error)
	MarkMailRead(playerID string, mailID string, readAt time.Time) error
	DeleteMail(playerID string, mailID string) error
	UpdateMailPlayerState(playerID string, mailID string, updatedAt time.Time, update func(account *Account, state *GameState, mail *Mail) error) (Account, GameState, Mail, error)
}

type MiniGameRecordRepository interface {
	SaveMiniGameRecord(record MiniGameRecord) error
	ListMiniGameRecords(playerID string, gameType string, limit int, offset int) ([]MiniGameRecord, int, error)
	UpdateMiniGamePlayerState(playerID string, updatedAt time.Time, update func(state *GameState, records []MiniGameRecord) ([]MiniGameRecord, error)) (GameState, []MiniGameRecord, error)
}

type PlayerViewRepository interface {
	GetPlayerSummaryView(playerID string) (PlayerSummaryView, error)
	GetCityView(playerID string) (CityView, error)
	GetResourceView(playerID string) (ResourceView, error)
	GetMilitaryView(playerID string) (MilitaryView, error)
	GetInventoryView(playerID string) (InventoryView, error)
	GetGeneralsView(playerID string) (GeneralsView, error)
}

type ReinforcementRepository interface {
	CreateReinforcementWithState(fromPlayerID string, toPlayerID string, updatedAt time.Time, update func(from *GameState, to *GameState, targetRecords []Reinforcement) (Reinforcement, error)) (GameState, GameState, Reinforcement, error)
	UpdateReinforcement(reinforcementID string, updatedAt time.Time, update func(from *GameState, to *GameState, reinforcement *Reinforcement) error) (GameState, GameState, Reinforcement, error)
	GetReinforcement(reinforcementID string) (Reinforcement, error)
	ListSentReinforcements(playerID string) ([]Reinforcement, error)
	ListReceivedReinforcements(playerID string) ([]Reinforcement, error)
}

type PvpRepository interface {
	CreatePvpMarchWithState(attackerPlayerID string, defenderPlayerID string, updatedAt time.Time, update func(attacker *GameState, defender *GameState) (PvpMarch, error)) (GameState, GameState, PvpMarch, error)
	UpdatePvpScoutStates(scoutPlayerID string, targetPlayerID string, updatedAt time.Time, update func(scout *GameState, target *GameState) error) (GameState, GameState, error)
	GetPvpMarch(marchID string) (PvpMarch, error)
	UpdatePvpMarch(marchID string, updatedAt time.Time, update func(march *PvpMarch) error) (PvpMarch, error)
	UpdatePvpMarchWithAttackerState(marchID string, updatedAt time.Time, update func(attacker *GameState, march *PvpMarch) error) (GameState, PvpMarch, error)
	ListPvpMarchesForPlayer(playerID string) ([]PvpMarch, error)
	ListDuePvpMarches(playerID string, now time.Time) ([]PvpMarch, error)
	ResolvePvpBattleTransaction(marchID string, updatedAt time.Time, update func(attacker *GameState, defender *GameState, reinforcements []Reinforcement, march *PvpMarch) (PvpBattle, BattleReport, BattleReport, []BattleReport, []Reinforcement, error)) (GameState, GameState, PvpMarch, PvpBattle, BattleReport, BattleReport, error)
	GetPvpPlayerState(playerID string, now time.Time) (PvpPlayerState, error)
	SavePvpPlayerState(state PvpPlayerState, updatedAt time.Time) error
	GetPvpBattle(battleID string) (PvpBattle, error)
	ListPvpBattlesForPlayer(playerID string) ([]PvpBattle, error)
	GetCurrentPvpSeason(now time.Time) (PvpSeasonRecord, error)
	SavePvpSeason(season PvpSeasonRecord, updatedAt time.Time) error
	ListPvpSeasons() ([]PvpSeasonRecord, error)
	SavePvpSeasonPlayers(seasonID string, players []PvpSeasonPlayerRecord, updatedAt time.Time) error
	ListPvpSeasonPlayers(seasonID string) ([]PvpSeasonPlayerRecord, error)
}

type AnnouncementRepository interface {
	PromoteDueScheduledAnnouncements(now time.Time) error
	GetAnnouncementPlayerContext(playerID string) (AnnouncementPlayerContext, error)
	ListVisibleAnnouncements(ctx AnnouncementPlayerContext, filter AnnouncementListFilter, now time.Time) ([]AnnouncementSummary, int, error)
	GetVisibleAnnouncementDetail(ctx AnnouncementPlayerContext, announcementID string, now time.Time) (AnnouncementDetail, error)
	MarkAnnouncementRead(ctx AnnouncementPlayerContext, announcementID string, now time.Time) (AnnouncementReadState, error)
	MarkAnnouncementPopupShown(ctx AnnouncementPlayerContext, announcementID string, now time.Time) (AnnouncementReadState, error)
	DismissAnnouncement(ctx AnnouncementPlayerContext, announcementID string, now time.Time) (AnnouncementReadState, error)
	ListAdminAnnouncements(filter AdminAnnouncementFilter) ([]Announcement, int, error)
	GetAdminAnnouncement(announcementID string) (Announcement, error)
	SaveAnnouncement(announcement Announcement) (Announcement, error)
	UpdateAnnouncementStatus(announcementID string, status string, now time.Time) (Announcement, error)
	DeleteAnnouncementDraft(announcementID string) error
}

type GoldLedgerRepository interface {
	WriteGoldLedger(entry GoldLedgerEntry) error
	ListGoldLedger(filter GoldLedgerFilter) ([]GoldLedgerEntry, error)
}

type ItemLedgerRepository interface {
	WriteItemLedger(entry ItemLedgerEntry) error
	ListItemLedger(filter ItemLedgerFilter) ([]ItemLedgerEntry, int, error)
}

type EventProcessingRepository interface {
	ClaimEventProcessing(moduleID string, handlerKey string, eventKey string, processedAt time.Time) (bool, error)
}

type ReincarnationRepository interface {
	GetActiveReincarnationRun(playerID string, now time.Time) (ReincarnationRun, bool, error)
	GetReincarnationRun(runID string) (ReincarnationRun, error)
	SaveReincarnationRun(run ReincarnationRun) error
	UpdateReincarnationRunWithState(playerID string, runID string, updatedAt time.Time, update func(state *GameState, run *ReincarnationRun) ([]BattleReport, error)) (GameState, ReincarnationRun, []BattleReport, error)
	ListReincarnationRuns(playerID string, limit int, offset int) ([]ReincarnationRun, int, error)
}

type Repository interface {
	AccountRepository
	PlayerStateRepository
	PlayerMetaRepository
	NpcStateRepository
	BuildingAssetRepository
	RecruitAssetRepository
	ResourceAssetRepository
	ItemAssetRepository
	GeneralAssetRepository
	CombatAssetRepository
	RewardAssetRepository
	AccountAssetRepository
	ReportRepository
	MailRepository
	MiniGameRecordRepository
	PlayerViewRepository
	ReinforcementRepository
	PvpRepository
	AnnouncementRepository
	GoldLedgerRepository
	ItemLedgerRepository
	EventProcessingRepository
	ReincarnationRepository
}

type MemoryRepository struct {
	mu                sync.RWMutex
	accounts          map[string]Account
	accountByName     map[string]string
	accountPlayers    map[string][]string
	players           map[string]GameState
	playerUpdatedAt   map[string]time.Time
	reports           map[string][]BattleReport   // playerID → reports
	mails             map[string][]Mail           // playerID → mails
	miniGameRecords   map[string][]MiniGameRecord // playerID → records
	reinforcements    map[string]Reinforcement
	pvpMarches        map[string]PvpMarch
	pvpBattles        map[string]PvpBattle
	pvpPlayerStates   map[string]PvpPlayerState
	pvpSeasons        map[string]PvpSeasonRecord
	pvpSeasonPlayers  map[string][]PvpSeasonPlayerRecord
	announcements     map[string]Announcement
	announcementReads map[string]AnnouncementReadState
	ledger            []GoldLedgerEntry
	ledgerNextID      int64
	itemLedger        []ItemLedgerEntry
	eventClaims       map[string]struct{}
	reincarnationRuns map[string]ReincarnationRun
}

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{
		accounts:          make(map[string]Account),
		accountByName:     make(map[string]string),
		accountPlayers:    make(map[string][]string),
		players:           make(map[string]GameState),
		playerUpdatedAt:   make(map[string]time.Time),
		reports:           make(map[string][]BattleReport),
		mails:             make(map[string][]Mail),
		miniGameRecords:   make(map[string][]MiniGameRecord),
		reinforcements:    make(map[string]Reinforcement),
		pvpMarches:        make(map[string]PvpMarch),
		pvpBattles:        make(map[string]PvpBattle),
		pvpPlayerStates:   make(map[string]PvpPlayerState),
		pvpSeasons:        make(map[string]PvpSeasonRecord),
		pvpSeasonPlayers:  make(map[string][]PvpSeasonPlayerRecord),
		announcements:     make(map[string]Announcement),
		announcementReads: make(map[string]AnnouncementReadState),
		eventClaims:       make(map[string]struct{}),
		reincarnationRuns: make(map[string]ReincarnationRun),
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

// ListAllPlayers 返回全服玩家摘要，用于系统信函和全服喊话投递。
func (r *MemoryRepository) ListAllPlayers() ([]PlayerSummary, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	players := make([]PlayerSummary, 0, len(r.players))
	for playerID, state := range r.players {
		players = append(players, buildPlayerSummary(state, r.playerUpdatedAt[playerID]))
	}
	sort.Slice(players, func(i, j int) bool {
		return players[i].UpdatedAt > players[j].UpdatedAt
	})
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

	return cloneGameState(state)
}

func cloneGameState(state GameState) (GameState, error) {
	content, err := json.Marshal(state)
	if err != nil {
		return GameState{}, err
	}
	var cloned GameState
	if err := json.Unmarshal(content, &cloned); err != nil {
		return GameState{}, err
	}
	return cloned, nil
}

// GetPlayerSummaryView 从内存状态投影玩家摘要视图。
func (r *MemoryRepository) GetPlayerSummaryView(playerID string) (PlayerSummaryView, error) {
	state, err := r.GetState(playerID)
	if err != nil {
		return PlayerSummaryView{}, err
	}
	return PlayerSummaryView{
		Player:             state.Player,
		CityGold:           state.CityGold,
		UnreadMessageCount: state.UnreadMessageCount,
		UnreadMailCount:    state.UnreadMailCount,
		ServerTime:         state.ServerTime,
	}, nil
}

// GetCityView 从内存状态投影城池视图。
func (r *MemoryRepository) GetCityView(playerID string) (CityView, error) {
	state, err := r.GetState(playerID)
	if err != nil {
		return CityView{}, err
	}
	return CityView{
		Player:             state.Player,
		Buildings:          state.Buildings,
		ResourceSlots:      state.ResourceSlots,
		Resources:          state.Resources,
		ResourceProduction: state.ResourceProduction,
		CityGold:           state.CityGold,
		ActiveModifiers:    state.ActiveModifiers,
		ServerTime:         state.ServerTime,
	}, nil
}

// GetResourceView 从内存状态投影资源视图。
func (r *MemoryRepository) GetResourceView(playerID string) (ResourceView, error) {
	state, err := r.GetState(playerID)
	if err != nil {
		return ResourceView{}, err
	}
	return ResourceView{
		Resources:          state.Resources,
		ResourceProduction: state.ResourceProduction,
		ResourceSettledAt:  state.ResourceSettledAt,
		ProductionBoost:    state.ProductionBoost,
		ProductionBoostEnd: state.ProductionBoostEnd,
		CapacityBoost:      state.CapacityBoost,
		CapacityBoostEnd:   state.CapacityBoostEnd,
		ActiveModifiers:    state.ActiveModifiers,
		ServerTime:         state.ServerTime,
	}, nil
}

// GetMilitaryView 从内存状态投影军事视图。
func (r *MemoryRepository) GetMilitaryView(playerID string) (MilitaryView, error) {
	state, err := r.GetState(playerID)
	if err != nil {
		return MilitaryView{}, err
	}
	return MilitaryView{
		Army:               state.Army,
		RecruitQueues:      state.RecruitQueues,
		Resources:          state.Resources,
		CityGold:           state.CityGold,
		Buildings:          state.Buildings,
		ActiveModifiers:    state.ActiveModifiers,
		General:            state.General,
		Generals:           state.Generals,
		GeneralAssignments: state.GeneralAssignments,
		ServerTime:         state.ServerTime,
	}, nil
}

// GetInventoryView 从内存状态投影背包视图。
func (r *MemoryRepository) GetInventoryView(playerID string) (InventoryView, error) {
	state, err := r.GetState(playerID)
	if err != nil {
		return InventoryView{}, err
	}
	if state.Inventory == nil {
		state.Inventory = map[string]ItemStack{}
	}
	normalizeInventoryState(&state, time.Now())
	return InventoryView{Inventory: state.Inventory, InventorySlots: state.InventorySlots, ServerTime: state.ServerTime}, nil
}

// GetGeneralsView 从内存状态投影武将视图。
func (r *MemoryRepository) GetGeneralsView(playerID string) (GeneralsView, error) {
	state, err := r.GetState(playerID)
	if err != nil {
		return GeneralsView{}, err
	}
	return GeneralsView{
		General:            state.General,
		Generals:           state.Generals,
		GeneralAssignments: state.GeneralAssignments,
		GeneralChangeUntil: state.GeneralChangeUntil,
		ActiveModifiers:    state.ActiveModifiers,
		ServerTime:         state.ServerTime,
	}, nil
}

func (r *MemoryRepository) UpdatePlayerState(playerID string, updatedAt time.Time, update func(state *GameState) error) (GameState, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	state, exists := r.players[playerID]
	if !exists {
		return GameState{}, ErrPlayerNotFound
	}
	state, err := cloneGameState(state)
	if err != nil {
		return GameState{}, err
	}
	if update != nil {
		if err := update(&state); err != nil {
			return GameState{}, err
		}
	}

	r.players[playerID] = state
	r.playerUpdatedAt[playerID] = updatedAt
	return state, nil
}

func (r *MemoryRepository) UpdatePlayerMetaState(playerID string, updatedAt time.Time, update func(state *GameState) error) (GameState, error) {
	return r.UpdatePlayerState(playerID, updatedAt, update)
}

func (r *MemoryRepository) UpdateNpcState(playerID string, updatedAt time.Time, update func(state *GameState) error) (GameState, error) {
	return r.UpdatePlayerState(playerID, updatedAt, update)
}

func (r *MemoryRepository) UpdateBuildingResourceState(playerID string, updatedAt time.Time, update func(state *GameState) error) (GameState, error) {
	return r.UpdatePlayerState(playerID, updatedAt, update)
}

func (r *MemoryRepository) UpdateAccountBuildingResourceState(accountID string, playerID string, updatedAt time.Time, update func(account *Account, state *GameState) error) (Account, GameState, error) {
	return r.UpdateAccountPlayerState(accountID, playerID, updatedAt, update)
}

func (r *MemoryRepository) UpdateRecruitState(playerID string, updatedAt time.Time, update func(state *GameState) error) (GameState, error) {
	return r.UpdatePlayerState(playerID, updatedAt, update)
}

func (r *MemoryRepository) UpdateResourceState(playerID string, updatedAt time.Time, update func(state *GameState) error) (GameState, error) {
	return r.UpdatePlayerState(playerID, updatedAt, update)
}

func (r *MemoryRepository) UpdateItemState(playerID string, updatedAt time.Time, update func(state *GameState) error) (GameState, error) {
	return r.UpdatePlayerState(playerID, updatedAt, update)
}

func (r *MemoryRepository) UpdateGeneralExpItemState(playerID string, itemID string, updatedAt time.Time, update func(state *GameState) error) (GameState, error) {
	return r.UpdatePlayerState(playerID, updatedAt, update)
}

func (r *MemoryRepository) UpdateGeneralState(playerID string, updatedAt time.Time, update func(state *GameState) error) (GameState, error) {
	return r.UpdatePlayerState(playerID, updatedAt, update)
}

func (r *MemoryRepository) UpdateAccountGeneralState(accountID string, playerID string, updatedAt time.Time, update func(account *Account, state *GameState) error) (Account, GameState, error) {
	return r.UpdateAccountPlayerState(accountID, playerID, updatedAt, update)
}

func (r *MemoryRepository) UpdateCombatState(playerID string, scope CombatAssetScope, updatedAt time.Time, update func(state *GameState) error) (GameState, error) {
	return r.UpdatePlayerState(playerID, updatedAt, update)
}

func (r *MemoryRepository) UpdateRewardState(playerID string, updatedAt time.Time, update func(state *GameState) error) (GameState, error) {
	return r.UpdatePlayerState(playerID, updatedAt, update)
}

func (r *MemoryRepository) UpdateScopedRewardState(playerID string, scope RewardAssetScope, updatedAt time.Time, update func(state *GameState) error) (GameState, error) {
	return r.UpdatePlayerState(playerID, updatedAt, update)
}

func (r *MemoryRepository) UpdateAccountRewardState(accountID string, playerID string, updatedAt time.Time, update func(account *Account, state *GameState) error) (Account, GameState, error) {
	return r.UpdateAccountPlayerState(accountID, playerID, updatedAt, update)
}

func (r *MemoryRepository) UpdateScopedAccountRewardState(accountID string, playerID string, scope RewardAssetScope, updatedAt time.Time, update func(account *Account, state *GameState) error) (Account, GameState, error) {
	return r.UpdateAccountPlayerState(accountID, playerID, updatedAt, update)
}

func (r *MemoryRepository) UpdateAccountPlayerState(accountID string, playerID string, updatedAt time.Time, update func(account *Account, state *GameState) error) (Account, GameState, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	account, exists := r.accounts[accountID]
	if !exists {
		return Account{}, GameState{}, ErrAccountNotFound
	}
	state, exists := r.players[playerID]
	if !exists {
		return Account{}, GameState{}, ErrPlayerNotFound
	}
	state, err := cloneGameState(state)
	if err != nil {
		return Account{}, GameState{}, err
	}

	ownsPlayer := false
	for _, candidatePlayerID := range r.accountPlayers[accountID] {
		if candidatePlayerID == playerID {
			ownsPlayer = true
			break
		}
	}
	if !ownsPlayer {
		return Account{}, GameState{}, ErrPlayerNotFound
	}

	if update != nil {
		if err := update(&account, &state); err != nil {
			return Account{}, GameState{}, err
		}
	}

	r.accounts[accountID] = account
	r.players[playerID] = state
	r.playerUpdatedAt[playerID] = updatedAt.UTC()
	return account, state, nil
}

func (r *MemoryRepository) ClaimEventProcessing(moduleID string, handlerKey string, eventKey string, processedAt time.Time) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	key := strings.TrimSpace(moduleID) + "|" + strings.TrimSpace(handlerKey) + "|" + strings.TrimSpace(eventKey)
	if _, exists := r.eventClaims[key]; exists {
		return false, nil
	}
	r.eventClaims[key] = struct{}{}
	return true, nil
}

// --- Battle Report Methods (MemoryRepository) ---

func (r *MemoryRepository) SaveReport(report BattleReport) error {
	return r.SaveReports([]BattleReport{report})
}

// SaveReports 批量保存标准战报，内存仓储用同一把锁模拟事务边界。
func (r *MemoryRepository) SaveReports(reports []BattleReport) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, report := range reports {
		report = NormalizeBattleReport(report)
		r.reports[report.PlayerID] = append([]BattleReport{report}, r.reports[report.PlayerID]...)
		enforceMemoryBattleReportVisibleCap(r.reports[report.PlayerID], report.ViewType)
	}
	return nil
}

// enforceMemoryBattleReportVisibleCap 软删除同一视角超过上限的旧可见战报。
func enforceMemoryBattleReportVisibleCap(reports []BattleReport, viewType string) {
	visible := 0
	for i := range reports {
		report := NormalizeBattleReport(reports[i])
		if report.DeletedByPlayer || report.ViewType != viewType {
			continue
		}
		visible++
		if visible > battleReportVisibleCapPerView {
			reports[i].DeletedByPlayer = true
		}
	}
}

// GetReportForPlayer 获取指定玩家拥有且未删除的战报。
func (r *MemoryRepository) GetReportForPlayer(playerID string, reportID string) (BattleReport, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, report := range r.reports[playerID] {
		report = NormalizeBattleReport(report)
		if report.ID == reportID && !report.DeletedByPlayer {
			return report, nil
		}
	}
	return BattleReport{}, errors.New("report not found")
}

// GetReportByShareToken 通过分享 token 获取公开战报。
func (r *MemoryRepository) GetReportByShareToken(token string) (BattleReport, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, reports := range r.reports {
		for _, report := range reports {
			report = NormalizeBattleReport(report)
			if report.Share != nil && report.Share.Token == token {
				return report, nil
			}
		}
	}
	return BattleReport{}, errors.New("report not found")
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

// ListReportsByQuery 按标准筛选条件分页查询玩家战报。
func (r *MemoryRepository) ListReportsByQuery(query BattleReportQuery) ([]BattleReport, int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	limit := query.PageSize
	if limit <= 0 {
		limit = 10
	}
	if limit > 50 {
		limit = 50
	}
	offset := 0
	if query.Page > 1 {
		offset = (query.Page - 1) * limit
	}
	timeFrom := query.TimeFrom
	if timeFrom.IsZero() {
		timeFrom = time.Now().Add(-3 * 24 * time.Hour)
	}

	var result []BattleReport
	total := 0
	for _, raw := range r.reports[query.PlayerID] {
		report := NormalizeBattleReport(raw)
		if !query.IncludeDeleted && report.DeletedByPlayer {
			continue
		}
		createdAt, err := time.Parse(time.RFC3339, report.CreatedAt)
		if err == nil {
			if createdAt.Before(timeFrom) {
				continue
			}
			if !query.TimeTo.IsZero() && createdAt.After(query.TimeTo) {
				continue
			}
		}
		if query.ViewType != "" && report.ViewType != query.ViewType {
			continue
		}
		if query.SourceType != "" && report.SourceType != query.SourceType {
			continue
		}
		if query.BattleType != "" && report.BattleType != query.BattleType {
			continue
		}
		if query.Result != "" && report.Result != query.Result {
			continue
		}
		total++
		if offset > 0 {
			offset--
			continue
		}
		result = append(result, report)
		if len(result) >= limit {
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

// MarkReportsReadByView 标记指定视角 Tab 的战报为已读。
func (r *MemoryRepository) MarkReportsReadByView(playerID string, viewType string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for i := range r.reports[playerID] {
		report := NormalizeBattleReport(r.reports[playerID][i])
		if viewType == "" || report.ViewType == viewType {
			r.reports[playerID][i].Read = true
		}
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

// DeleteReportsByView 删除指定视角 Tab 下的战报。
func (r *MemoryRepository) DeleteReportsByView(playerID string, viewType string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for i := range r.reports[playerID] {
		report := NormalizeBattleReport(r.reports[playerID][i])
		if viewType == "" || report.ViewType == viewType {
			r.reports[playerID][i].DeletedByPlayer = true
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

// CreateBattleReportShareLink 为玩家战报创建分享 token。
func (r *MemoryRepository) CreateBattleReportShareLink(playerID string, reportID string, visibility string, expiresAt time.Time) (BattleReportShareLink, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now().UTC()
	link := BattleReportShareLink{
		ID:         "share_" + randomID(12),
		ReportID:   reportID,
		Token:      "br_" + randomID(24),
		Visibility: visibility,
		CreatedAt:  now.Format(time.RFC3339),
	}
	if !expiresAt.IsZero() {
		link.ExpiresAt = expiresAt.UTC().Format(time.RFC3339)
	}
	for i := range r.reports[playerID] {
		if r.reports[playerID][i].ID == reportID {
			if r.reports[playerID][i].Share == nil {
				r.reports[playerID][i].Share = &BattleReportShare{}
			}
			r.reports[playerID][i].Share.Token = link.Token
			r.reports[playerID][i].Share.Visibility = link.Visibility
			r.reports[playerID][i].Share.ExpiresAt = link.ExpiresAt
			return link, nil
		}
	}
	return BattleReportShareLink{}, errors.New("report not found")
}

// ListBattleEventsForAdmin 从内存战报中按 eventId 汇总战斗事件。
func (r *MemoryRepository) ListBattleEventsForAdmin(query BattleEventQuery) ([]BattleEvent, int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	eventsByID := map[string]BattleEvent{}
	for _, reports := range r.reports {
		for _, raw := range reports {
			report := NormalizeBattleReport(raw)
			if query.PlayerID != "" && report.PlayerID != query.PlayerID {
				continue
			}
			if query.EventID != "" && report.EventID != query.EventID {
				continue
			}
			if query.SourceType != "" && report.SourceType != query.SourceType {
				continue
			}
			if query.BattleType != "" && report.BattleType != query.BattleType {
				continue
			}
			if query.Result != "" && report.Result != query.Result {
				continue
			}
			if _, exists := eventsByID[report.EventID]; exists {
				continue
			}
			eventsByID[report.EventID] = BattleEvent{
				ID:               report.EventID,
				SourceType:       report.SourceType,
				SourceID:         report.TargetID,
				Scene:            report.ViewType,
				BattleType:       report.BattleType,
				Result:           report.Result,
				AttackerPlayerID: report.PlayerID,
				DefenderPlayerID: report.TargetID,
				AttackerName:     report.PlayerName,
				DefenderName:     report.TargetName,
				AttackerFaction:  report.PlayerFaction,
				DefenderFaction:  report.DefenderFaction,
				OccurredAt:       report.CreatedAt,
				CreatedAt:        report.CreatedAt,
			}
		}
	}
	items := make([]BattleEvent, 0, len(eventsByID))
	for _, event := range eventsByID {
		items = append(items, event)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].OccurredAt > items[j].OccurredAt })
	total := len(items)
	pageSize := query.PageSize
	if pageSize <= 0 {
		pageSize = 20
	}
	if query.Page <= 0 {
		query.Page = 1
	}
	start := (query.Page - 1) * pageSize
	if start >= len(items) {
		return []BattleEvent{}, total, nil
	}
	end := start + pageSize
	if end > len(items) {
		end = len(items)
	}
	return items[start:end], total, nil
}

// GetBattleEventForAdmin 获取单个战斗事件。
func (r *MemoryRepository) GetBattleEventForAdmin(eventID string) (BattleEvent, error) {
	items, total, err := r.ListBattleEventsForAdmin(BattleEventQuery{EventID: eventID, Page: 1, PageSize: 1})
	if err != nil {
		return BattleEvent{}, err
	}
	if total == 0 || len(items) == 0 {
		return BattleEvent{}, errors.New("battle event not found")
	}
	return items[0], nil
}

// ListReportsByEventForAdmin 返回同一事件下所有玩家视角战报。
func (r *MemoryRepository) ListReportsByEventForAdmin(eventID string) ([]BattleReport, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var reports []BattleReport
	for _, playerReports := range r.reports {
		for _, raw := range playerReports {
			report := NormalizeBattleReport(raw)
			if report.EventID == eventID {
				reports = append(reports, report)
			}
		}
	}
	sort.Slice(reports, func(i, j int) bool { return reports[i].CreatedAt > reports[j].CreatedAt })
	return reports, nil
}

// ListParticipantsByEventForAdmin 从内存战报标准详情中汇总事件参与方快照。
func (r *MemoryRepository) ListParticipantsByEventForAdmin(eventID string) ([]BattleReportParticipant, error) {
	reports, err := r.ListReportsByEventForAdmin(eventID)
	if err != nil {
		return nil, err
	}
	participants := make([]BattleReportParticipant, 0, len(reports)*2)
	for _, report := range reports {
		if report.Detail == nil {
			continue
		}
		participants = append(participants, buildMemoryBattleReportParticipant(report, "primary", report.Detail.PrimarySide))
		if report.Detail.SecondarySide != nil {
			participants = append(participants, buildMemoryBattleReportParticipant(report, "secondary", *report.Detail.SecondarySide))
		}
	}
	sort.Slice(participants, func(i, j int) bool {
		if participants[i].ReportID == participants[j].ReportID {
			return participants[i].Role < participants[j].Role
		}
		return participants[i].ReportID < participants[j].ReportID
	})
	return participants, nil
}

// buildMemoryBattleReportParticipant 将标准战报一侧转换为内存 GM 参与方快照。
func buildMemoryBattleReportParticipant(report BattleReport, side string, snapshot BattleReportSide) BattleReportParticipant {
	troopsBefore := map[string]int{}
	troopsLost := map[string]int{}
	troopsSurvived := map[string]int{}
	for _, unit := range snapshot.Units {
		troopsBefore[unit.UnitType] = unit.AmountBefore
		troopsLost[unit.UnitType] = unit.Lost
		troopsSurvived[unit.UnitType] = unit.Survived
	}
	return BattleReportParticipant{
		ID:             "participant_" + report.ID + "_" + side,
		EventID:        report.EventID,
		ReportID:       report.ID,
		PlayerID:       snapshot.PlayerID,
		Role:           snapshot.Role,
		Faction:        snapshot.Faction,
		Nickname:       snapshot.PlayerName,
		CityName:       snapshot.CityName,
		TroopsBefore:   troopsBefore,
		TroopsLost:     troopsLost,
		TroopsSurvived: troopsSurvived,
		Generals:       snapshot.Generals,
		Rewards:        report.Detail.Rewards,
		CreatedAt:      report.CreatedAt,
	}
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

func (r *MemoryRepository) UpdateMailPlayerState(playerID string, mailID string, updatedAt time.Time, update func(account *Account, state *GameState, mail *Mail) error) (Account, GameState, Mail, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	state, exists := r.players[playerID]
	if !exists {
		return Account{}, GameState{}, Mail{}, ErrPlayerNotFound
	}
	state, err := cloneGameState(state)
	if err != nil {
		return Account{}, GameState{}, Mail{}, err
	}

	for i := range r.mails[playerID] {
		mail := r.mails[playerID][i]
		if mail.ID != mailID || mail.DeletedByPlayer {
			continue
		}
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
			return Account{}, GameState{}, Mail{}, ErrAccountNotFound
		}
		account, exists := r.accounts[accountID]
		if !exists {
			return Account{}, GameState{}, Mail{}, ErrAccountNotFound
		}
		if update != nil {
			if err := update(&account, &state, &mail); err != nil {
				return Account{}, GameState{}, Mail{}, err
			}
		}
		r.accounts[accountID] = account
		r.mails[playerID][i] = mail
		r.players[playerID] = state
		r.playerUpdatedAt[playerID] = updatedAt.UTC()
		return account, state, mail, nil
	}
	return Account{}, GameState{}, Mail{}, ErrMailNotFound
}

// --- MiniGame Record Methods (MemoryRepository) ---

func (r *MemoryRepository) SaveMiniGameRecord(record MiniGameRecord) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if miniGameRecordHasRedeemableReward(record.GameType) && record.RemainingAmount == 0 && record.RewardAmount > 0 {
		record.RemainingAmount = record.RewardAmount
	}

	r.miniGameRecords[record.PlayerID] = append([]MiniGameRecord{record}, r.miniGameRecords[record.PlayerID]...)
	// 保留最多 500 条
	if len(r.miniGameRecords[record.PlayerID]) > 500 {
		r.miniGameRecords[record.PlayerID] = r.miniGameRecords[record.PlayerID][:500]
	}
	return nil
}

func (r *MemoryRepository) ListMiniGameRecords(playerID string, gameType string, limit int, offset int) ([]MiniGameRecord, int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	all := r.miniGameRecords[playerID]
	if gameType != "" {
		filtered := make([]MiniGameRecord, 0, len(all))
		for _, record := range all {
			if record.GameType == gameType {
				filtered = append(filtered, record)
			}
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

func (r *MemoryRepository) UpdateMiniGamePlayerState(playerID string, updatedAt time.Time, update func(state *GameState, records []MiniGameRecord) ([]MiniGameRecord, error)) (GameState, []MiniGameRecord, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	state, exists := r.players[playerID]
	if !exists {
		return GameState{}, nil, ErrPlayerNotFound
	}
	state, err := cloneGameState(state)
	if err != nil {
		return GameState{}, nil, err
	}
	records := append([]MiniGameRecord(nil), r.miniGameRecords[playerID]...)
	if update != nil {
		records, err = update(&state, records)
		if err != nil {
			return GameState{}, nil, err
		}
	}
	r.miniGameRecords[playerID] = records
	r.players[playerID] = state
	r.playerUpdatedAt[playerID] = updatedAt.UTC()
	return state, records, nil
}

// CreateReinforcementWithState 在内存事务中创建增援并更新双方相关资产。
func (r *MemoryRepository) CreateReinforcementWithState(fromPlayerID string, toPlayerID string, updatedAt time.Time, update func(from *GameState, to *GameState, targetRecords []Reinforcement) (Reinforcement, error)) (GameState, GameState, Reinforcement, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	from, exists := r.players[fromPlayerID]
	if !exists {
		return GameState{}, GameState{}, Reinforcement{}, ErrPlayerNotFound
	}
	to, exists := r.players[toPlayerID]
	if !exists {
		return GameState{}, GameState{}, Reinforcement{}, ErrPlayerNotFound
	}
	from, err := cloneGameState(from)
	if err != nil {
		return GameState{}, GameState{}, Reinforcement{}, err
	}
	to, err = cloneGameState(to)
	if err != nil {
		return GameState{}, GameState{}, Reinforcement{}, err
	}
	targetRecords := r.memoryReceivedReinforcementsLocked(toPlayerID)
	record, err := update(&from, &to, targetRecords)
	if err != nil {
		return GameState{}, GameState{}, Reinforcement{}, err
	}
	r.players[fromPlayerID] = from
	r.players[toPlayerID] = to
	r.playerUpdatedAt[fromPlayerID] = updatedAt.UTC()
	r.playerUpdatedAt[toPlayerID] = updatedAt.UTC()
	r.reinforcements[record.ID] = record
	return from, to, record, nil
}

// UpdateReinforcement 在内存事务中更新单个增援和双方相关资产。
func (r *MemoryRepository) UpdateReinforcement(reinforcementID string, updatedAt time.Time, update func(from *GameState, to *GameState, reinforcement *Reinforcement) error) (GameState, GameState, Reinforcement, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	record, exists := r.reinforcements[reinforcementID]
	if !exists {
		return GameState{}, GameState{}, Reinforcement{}, ErrReinforcementNotFound
	}
	from, exists := r.players[record.FromPlayerID]
	if !exists {
		return GameState{}, GameState{}, Reinforcement{}, ErrPlayerNotFound
	}
	to, exists := r.players[record.ToPlayerID]
	if !exists {
		return GameState{}, GameState{}, Reinforcement{}, ErrPlayerNotFound
	}
	from, err := cloneGameState(from)
	if err != nil {
		return GameState{}, GameState{}, Reinforcement{}, err
	}
	to, err = cloneGameState(to)
	if err != nil {
		return GameState{}, GameState{}, Reinforcement{}, err
	}
	previousReinforcementID := record.ID
	if update != nil {
		if err := update(&from, &to, &record); err != nil {
			return GameState{}, GameState{}, Reinforcement{}, err
		}
	}
	r.players[record.FromPlayerID] = from
	r.players[record.ToPlayerID] = to
	r.playerUpdatedAt[record.FromPlayerID] = updatedAt.UTC()
	r.playerUpdatedAt[record.ToPlayerID] = updatedAt.UTC()
	if previousReinforcementID != record.ID {
		delete(r.reinforcements, previousReinforcementID)
	}
	r.reinforcements[record.ID] = record
	return from, to, record, nil
}

// GetReinforcement 读取单个增援批次。
func (r *MemoryRepository) GetReinforcement(reinforcementID string) (Reinforcement, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	record, exists := r.reinforcements[reinforcementID]
	if !exists {
		return Reinforcement{}, ErrReinforcementNotFound
	}
	return cloneReinforcement(record), nil
}

// ListSentReinforcements 读取玩家派出的增援。
func (r *MemoryRepository) ListSentReinforcements(playerID string) ([]Reinforcement, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := []Reinforcement{}
	for _, record := range r.reinforcements {
		if record.FromPlayerID == playerID {
			result = append(result, cloneReinforcement(record))
		}
	}
	return result, nil
}

// ListReceivedReinforcements 读取玩家收到的增援。
func (r *MemoryRepository) ListReceivedReinforcements(playerID string) ([]Reinforcement, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return cloneReinforcements(r.memoryReceivedReinforcementsLocked(playerID)), nil
}

func (r *MemoryRepository) memoryReceivedReinforcementsLocked(playerID string) []Reinforcement {
	result := []Reinforcement{}
	for _, record := range r.reinforcements {
		if record.ToPlayerID == playerID {
			result = append(result, record)
		}
	}
	return result
}

// CreatePvpMarchWithState 在内存仓储中创建 PVP 行军并同步扣出攻击方资产。
func (r *MemoryRepository) CreatePvpMarchWithState(attackerPlayerID string, defenderPlayerID string, updatedAt time.Time, update func(attacker *GameState, defender *GameState) (PvpMarch, error)) (GameState, GameState, PvpMarch, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	attacker, ok := r.players[attackerPlayerID]
	if !ok {
		return GameState{}, GameState{}, PvpMarch{}, ErrPlayerNotFound
	}
	defender, ok := r.players[defenderPlayerID]
	if !ok {
		return GameState{}, GameState{}, PvpMarch{}, ErrPlayerNotFound
	}
	march, err := update(&attacker, &defender)
	if err != nil {
		return GameState{}, GameState{}, PvpMarch{}, err
	}
	r.players[attackerPlayerID] = attacker
	r.players[defenderPlayerID] = defender
	r.playerUpdatedAt[attackerPlayerID] = updatedAt.UTC()
	r.playerUpdatedAt[defenderPlayerID] = updatedAt.UTC()
	r.pvpMarches[march.ID] = march
	return attacker, defender, march, nil
}

// UpdatePvpScoutStates 在内存仓储中同一锁范围内更新侦查双方状态。
func (r *MemoryRepository) UpdatePvpScoutStates(scoutPlayerID string, targetPlayerID string, updatedAt time.Time, update func(scout *GameState, target *GameState) error) (GameState, GameState, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	scout, ok := r.players[scoutPlayerID]
	if !ok {
		return GameState{}, GameState{}, ErrPlayerNotFound
	}
	target, ok := r.players[targetPlayerID]
	if !ok {
		return GameState{}, GameState{}, ErrPlayerNotFound
	}
	if err := update(&scout, &target); err != nil {
		return GameState{}, GameState{}, err
	}
	r.players[scoutPlayerID] = scout
	r.players[targetPlayerID] = target
	r.playerUpdatedAt[scoutPlayerID] = updatedAt.UTC()
	r.playerUpdatedAt[targetPlayerID] = updatedAt.UTC()
	return scout, target, nil
}

// GetPvpMarch 读取单条 PVP 行军。
func (r *MemoryRepository) GetPvpMarch(marchID string) (PvpMarch, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	march, ok := r.pvpMarches[marchID]
	if !ok {
		return PvpMarch{}, ErrPlayerNotFound
	}
	return clonePvpMarch(march), nil
}

// UpdatePvpMarch 更新单条 PVP 行军。
func (r *MemoryRepository) UpdatePvpMarch(marchID string, updatedAt time.Time, update func(march *PvpMarch) error) (PvpMarch, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	march, ok := r.pvpMarches[marchID]
	if !ok {
		return PvpMarch{}, ErrPlayerNotFound
	}
	if err := update(&march); err != nil {
		return PvpMarch{}, err
	}
	march.UpdatedAt = updatedAt.UTC().Format(resourceDateLayout)
	r.pvpMarches[marchID] = march
	return clonePvpMarch(march), nil
}

// UpdatePvpMarchWithAttackerState 在内存仓储中同时更新 PVP 行军和攻击方状态。
func (r *MemoryRepository) UpdatePvpMarchWithAttackerState(marchID string, updatedAt time.Time, update func(attacker *GameState, march *PvpMarch) error) (GameState, PvpMarch, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	march, ok := r.pvpMarches[marchID]
	if !ok {
		return GameState{}, PvpMarch{}, ErrPlayerNotFound
	}
	attacker, ok := r.players[march.AttackerPlayerID]
	if !ok {
		return GameState{}, PvpMarch{}, ErrPlayerNotFound
	}
	if err := update(&attacker, &march); err != nil {
		return GameState{}, PvpMarch{}, err
	}
	attacker.ServerTime = updatedAt.UTC().Format(resourceDateLayout)
	march.UpdatedAt = updatedAt.UTC().Format(resourceDateLayout)
	r.players[attacker.Player.ID] = attacker
	r.playerUpdatedAt[attacker.Player.ID] = updatedAt.UTC()
	r.pvpMarches[march.ID] = march
	return attacker, clonePvpMarch(march), nil
}

// ListPvpMarchesForPlayer 返回玩家相关的 PVP 行军。
func (r *MemoryRepository) ListPvpMarchesForPlayer(playerID string) ([]PvpMarch, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := []PvpMarch{}
	for _, march := range r.pvpMarches {
		if march.AttackerPlayerID == playerID || march.DefenderPlayerID == playerID {
			result = append(result, clonePvpMarch(march))
		}
	}
	return result, nil
}

// ListDuePvpMarches 返回玩家相关且已经到达的行军。
func (r *MemoryRepository) ListDuePvpMarches(playerID string, now time.Time) ([]PvpMarch, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := []PvpMarch{}
	for _, march := range r.pvpMarches {
		if march.AttackerPlayerID != playerID && march.DefenderPlayerID != playerID {
			continue
		}
		switch march.Status {
		case PvpMarchStatusMarching:
			arrivesAt, err := time.Parse(resourceDateLayout, march.ArrivesAt)
			if err == nil && !arrivesAt.After(now.UTC()) {
				result = append(result, clonePvpMarch(march))
			}
		case PvpMarchStatusReturning:
			returnsAt, err := time.Parse(resourceDateLayout, march.ReturnsAt)
			if err == nil && !returnsAt.After(now.UTC()) {
				result = append(result, clonePvpMarch(march))
			}
		}
	}
	return result, nil
}

// ResolvePvpBattleTransaction 在内存仓储中结算一场 PVP 战斗。
func (r *MemoryRepository) ResolvePvpBattleTransaction(marchID string, updatedAt time.Time, update func(attacker *GameState, defender *GameState, reinforcements []Reinforcement, march *PvpMarch) (PvpBattle, BattleReport, BattleReport, []BattleReport, []Reinforcement, error)) (GameState, GameState, PvpMarch, PvpBattle, BattleReport, BattleReport, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	march, ok := r.pvpMarches[marchID]
	if !ok {
		return GameState{}, GameState{}, PvpMarch{}, PvpBattle{}, BattleReport{}, BattleReport{}, ErrPlayerNotFound
	}
	if march.Status == PvpMarchStatusResolved {
		battle, _ := r.pvpBattles[march.BattleID]
		return r.players[march.AttackerPlayerID], r.players[march.DefenderPlayerID], clonePvpMarch(march), battle, BattleReport{}, BattleReport{}, nil
	}
	if march.Status != PvpMarchStatusMarching && march.Status != PvpMarchStatusResolving {
		return GameState{}, GameState{}, PvpMarch{}, PvpBattle{}, BattleReport{}, BattleReport{}, ErrInvalidReinforcement
	}
	attacker := r.players[march.AttackerPlayerID]
	defender := r.players[march.DefenderPlayerID]
	targetRecords := []Reinforcement{}
	for _, record := range r.reinforcements {
		normalizeGarrisonRecord(&record)
		if record.HostPlayerID == defender.Player.ID && record.Status == ReinforcementStatusStationed && record.Rules.CanFight {
			targetRecords = append(targetRecords, cloneReinforcement(record))
		}
	}
	march.Status = PvpMarchStatusResolving
	battle, attackerReport, defenderReport, reinforcementReports, changedReinforcements, err := update(&attacker, &defender, targetRecords, &march)
	if err != nil {
		return GameState{}, GameState{}, PvpMarch{}, PvpBattle{}, BattleReport{}, BattleReport{}, err
	}
	for _, record := range changedReinforcements {
		r.reinforcements[record.ID] = record
	}
	r.players[attacker.Player.ID] = attacker
	r.players[defender.Player.ID] = defender
	r.playerUpdatedAt[attacker.Player.ID] = updatedAt.UTC()
	r.playerUpdatedAt[defender.Player.ID] = updatedAt.UTC()
	r.pvpMarches[march.ID] = march
	r.pvpBattles[battle.ID] = battle
	if attackerReport.ID != "" {
		r.reports[attackerReport.PlayerID] = append([]BattleReport{attackerReport}, r.reports[attackerReport.PlayerID]...)
	}
	if defenderReport.ID != "" {
		r.reports[defenderReport.PlayerID] = append([]BattleReport{defenderReport}, r.reports[defenderReport.PlayerID]...)
	}
	for _, report := range reinforcementReports {
		if report.ID != "" {
			r.reports[report.PlayerID] = append([]BattleReport{NormalizeBattleReport(report)}, r.reports[report.PlayerID]...)
		}
	}
	return attacker, defender, clonePvpMarch(march), battle, attackerReport, defenderReport, nil
}

// GetPvpBattle 读取单条 PVP 战斗。
func (r *MemoryRepository) GetPvpBattle(battleID string) (PvpBattle, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	battle, ok := r.pvpBattles[battleID]
	if !ok {
		return PvpBattle{}, ErrPlayerNotFound
	}
	return battle, nil
}

// ListPvpBattlesForPlayer 返回玩家相关 PVP 战斗。
func (r *MemoryRepository) ListPvpBattlesForPlayer(playerID string) ([]PvpBattle, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := []PvpBattle{}
	for _, battle := range r.pvpBattles {
		if battle.AttackerPlayerID == playerID || battle.DefenderPlayerID == playerID {
			result = append(result, battle)
		}
	}
	return result, nil
}

// GetPvpPlayerState 读取或初始化玩家 PVP 状态。
func (r *MemoryRepository) GetPvpPlayerState(playerID string, now time.Time) (PvpPlayerState, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.players[playerID]; !ok {
		return PvpPlayerState{}, ErrPlayerNotFound
	}
	state, ok := r.pvpPlayerStates[playerID]
	if !ok {
		state = newDefaultPvpPlayerState(playerID, now)
		r.pvpPlayerStates[playerID] = state
	}
	state = normalizePvpPlayerState(state, now)
	r.pvpPlayerStates[playerID] = state
	return state, nil
}

// SavePvpPlayerState 保存玩家 PVP 状态。
func (r *MemoryRepository) SavePvpPlayerState(state PvpPlayerState, updatedAt time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.players[state.PlayerID]; !ok {
		return ErrPlayerNotFound
	}
	state.UpdatedAt = updatedAt.UTC().Format(resourceDateLayout)
	r.pvpPlayerStates[state.PlayerID] = state
	return nil
}

// GetCurrentPvpSeason 返回当前时间所在的 PVP 赛季。
func (r *MemoryRepository) GetCurrentPvpSeason(now time.Time) (PvpSeasonRecord, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, season := range r.pvpSeasons {
		startsAt, startErr := time.Parse(resourceDateLayout, season.StartsAt)
		endsAt, endErr := time.Parse(resourceDateLayout, season.EndsAt)
		if startErr == nil && endErr == nil && !now.UTC().Before(startsAt) && now.UTC().Before(endsAt) && season.Status == PvpSeasonStatusActive {
			return clonePvpSeasonRecord(season), nil
		}
	}
	return PvpSeasonRecord{}, ErrPlayerNotFound
}

// SavePvpSeason 保存 PVP 赛季定义。
func (r *MemoryRepository) SavePvpSeason(season PvpSeasonRecord, updatedAt time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if season.ID == "" {
		return ErrPlayerNotFound
	}
	season.UpdatedAt = updatedAt.UTC().Format(resourceDateLayout)
	if season.CreatedAt == "" {
		season.CreatedAt = season.UpdatedAt
	}
	r.pvpSeasons[season.ID] = clonePvpSeasonRecord(season)
	return nil
}

// ListPvpSeasons 返回全部 PVP 赛季。
func (r *MemoryRepository) ListPvpSeasons() ([]PvpSeasonRecord, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	items := make([]PvpSeasonRecord, 0, len(r.pvpSeasons))
	for _, season := range r.pvpSeasons {
		items = append(items, clonePvpSeasonRecord(season))
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].StartsAt > items[j].StartsAt
	})
	return items, nil
}

// SavePvpSeasonPlayers 保存赛季玩家结算快照。
func (r *MemoryRepository) SavePvpSeasonPlayers(seasonID string, players []PvpSeasonPlayerRecord, updatedAt time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	updatedAtText := updatedAt.UTC().Format(resourceDateLayout)
	next := make([]PvpSeasonPlayerRecord, 0, len(players))
	for _, player := range players {
		player.SeasonID = seasonID
		player.UpdatedAt = updatedAtText
		if player.CreatedAt == "" {
			player.CreatedAt = updatedAtText
		}
		next = append(next, player)
	}
	r.pvpSeasonPlayers[seasonID] = next
	return nil
}

// ListPvpSeasonPlayers 返回赛季玩家结算快照。
func (r *MemoryRepository) ListPvpSeasonPlayers(seasonID string) ([]PvpSeasonPlayerRecord, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	items := append([]PvpSeasonPlayerRecord(nil), r.pvpSeasonPlayers[seasonID]...)
	sort.Slice(items, func(i, j int) bool {
		return items[i].Rank < items[j].Rank
	})
	return items, nil
}

func clonePvpMarch(march PvpMarch) PvpMarch {
	march.AttackTroops = cloneStringIntMap(march.AttackTroops)
	march.AttackGenerals = append([]string(nil), march.AttackGenerals...)
	return march
}

// clonePvpSeasonRecord 复制赛季记录中的可变字段。
func clonePvpSeasonRecord(season PvpSeasonRecord) PvpSeasonRecord {
	season.Rules = cloneAnyMap(season.Rules)
	season.Rewards = cloneAnyMap(season.Rewards)
	return season
}

// --- Gold Ledger Methods (MemoryRepository) ---

func (r *MemoryRepository) WriteGoldLedger(entry GoldLedgerEntry) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.appendGoldLedgerLocked(entry)
	return nil
}

func (r *MemoryRepository) appendGoldLedgerLocked(entry GoldLedgerEntry) {
	r.ledgerNextID++
	entry.ID = r.ledgerNextID
	if entry.CreatedAt == "" {
		entry.CreatedAt = time.Now().UTC().Format(time.RFC3339)
	}
	r.ledger = append(r.ledger, entry)
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

// WriteItemLedger 写入内存物品流水。
func (r *MemoryRepository) WriteItemLedger(entry ItemLedgerEntry) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if entry.ID == "" {
		entry.ID = randomID(12)
	}
	if entry.CreatedAt == "" {
		entry.CreatedAt = time.Now().UTC().Format(time.RFC3339)
	}
	r.itemLedger = append(r.itemLedger, entry)
	return nil
}

// ListItemLedger 按筛选条件读取内存物品流水。
func (r *MemoryRepository) ListItemLedger(filter ItemLedgerFilter) ([]ItemLedgerEntry, int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	limit := filter.Limit
	if limit <= 0 || limit > 1000 {
		limit = 200
	}
	offset := filter.Offset
	if offset < 0 {
		offset = 0
	}
	matches := []ItemLedgerEntry{}
	for i := len(r.itemLedger) - 1; i >= 0; i-- {
		entry := r.itemLedger[i]
		if filter.PlayerID != "" && entry.PlayerID != filter.PlayerID {
			continue
		}
		if filter.ItemID != "" && entry.ItemID != filter.ItemID {
			continue
		}
		if filter.RefType != "" && entry.RefType != filter.RefType {
			continue
		}
		matches = append(matches, entry)
	}
	total := len(matches)
	if offset >= total {
		return []ItemLedgerEntry{}, total, nil
	}
	end := offset + limit
	if end > total {
		end = total
	}
	return append([]ItemLedgerEntry(nil), matches[offset:end]...), total, nil
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
