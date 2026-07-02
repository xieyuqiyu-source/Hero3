// 本文件定义游戏服务主结构、基础错误、配置加载和通用玩家能力。
package game

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"hero3/internal/core/combat"
)

var (
	ErrAccountExists               = errors.New("account already exists")
	ErrAccountNotFound             = errors.New("account not found")
	ErrInvalidCredentials          = errors.New("invalid credentials")
	ErrPlayerNotFound              = errors.New("player not found")
	ErrBuildingNotFound            = errors.New("building not found")
	ErrInsufficientRes             = errors.New("insufficient resources")
	ErrAlreadyUpgrading            = errors.New("building is already upgrading")
	ErrNotUpgrading                = errors.New("building is not upgrading")
	ErrMaxLevel                    = errors.New("building is at max level")
	ErrInvalidBuildingStatus       = errors.New("invalid building status")
	ErrInvalidBuildingMutation     = errors.New("invalid building mutation")
	ErrBuildingStatusBlocked       = errors.New("building status blocks this action")
	ErrInvalidEffectType           = errors.New("invalid effect type")
	ErrMixedEffectAssets           = errors.New("mixed effect assets require split execution")
	ErrUnitNotFound                = errors.New("unit not found")
	ErrNonCombatUnit               = errors.New("unit cannot participate in combat")
	ErrInvalidBuffKey              = errors.New("invalid buff key")
	ErrInvalidBuffMode             = errors.New("invalid buff mode")
	ErrInvalidAmount               = errors.New("invalid recruit amount")
	ErrQueueFull                   = errors.New("recruit queue is full")
	ErrInvalidGeneral              = errors.New("invalid general for faction")
	ErrGeneralNotFound             = errors.New("general not found")
	ErrGeneralChangeCooldown       = errors.New("general change is on cooldown")
	ErrInvalidStatKey              = errors.New("invalid general stat")
	ErrNoStatPoints                = errors.New("no general stat points available")
	ErrStatMaxLevel                = errors.New("general stat is at max level")
	ErrMailNotFound                = errors.New("mail not found")
	ErrInvalidMail                 = errors.New("invalid mail")
	ErrMailAlreadyClaimed          = errors.New("mail already claimed")
	ErrMailNoAttachments           = errors.New("mail has no attachments")
	ErrMailExpired                 = errors.New("mail expired")
	ErrMailClaimForbidden          = errors.New("mail attachments cannot be claimed")
	ErrMailInvalidAttachment       = errors.New("invalid mail attachment")
	ErrMailRecipientSelf           = errors.New("cannot send mail to yourself")
	ErrMiniGameNotFound            = errors.New("minigame record not found")
	ErrInvalidMiniGame             = errors.New("invalid minigame record")
	ErrMiniGameBetTooLow           = errors.New("minigame bet amount too low")
	ErrMiniGameBetTooHigh          = errors.New("minigame bet amount exceeds limit")
	ErrInvalidBait                 = errors.New("invalid fishing bait")
	ErrCrossFactionReward          = errors.New("reward unit is not available for current faction")
	ErrMiniGameStockShort          = errors.New("insufficient minigame reward stock")
	ErrItemNotFound                = errors.New("item not found")
	ErrItemNotUsable               = errors.New("item is not usable")
	ErrInsufficientItem            = errors.New("insufficient item")
	ErrInventoryFull               = errors.New("inventory is full")
	ErrDropPoolNotFound            = errors.New("drop pool not found")
	ErrItemIDLocked                = errors.New("物品 ID 已锁定，不能删除或改名")
	ErrReincarnationRunNotFound    = errors.New("reincarnation run not found")
	ErrReincarnationActive         = errors.New("reincarnation run already active")
	ErrInvalidReincarnation        = errors.New("invalid reincarnation run")
	ErrReinforcementNotFound       = errors.New("reinforcement not found")
	ErrInvalidReinforcement        = errors.New("invalid reinforcement")
	ErrReinforcementTargetSelf     = errors.New("cannot reinforce yourself")
	ErrReinforcementTargetNPC      = errors.New("npc cannot be reinforced")
	ErrReinforcementSlotFull       = errors.New("reinforcement source slots are full")
	ErrReinforcementBusy           = errors.New("reinforcement is busy")
	ErrReinforcementNotAccelerable = errors.New("reinforcement cannot be accelerated")
	ErrGeneralBusy                 = errors.New("general is already assigned")
	ErrWorldMapFull                = errors.New("world map is full")
	ErrInvalidWorldCoordinate      = errors.New("invalid world coordinate")
	ErrOperationTooFast            = errors.New("操作太快，请稍后再试")
)

const resourceDateLayout = time.RFC3339
const playerDeletionDelay = time.Hour

type Service struct {
	repo              Repository
	eventBus          *EventBus
	playerLocks       sync.Map // per-player 互斥锁，防止同一存档的关键资产操作并发竞态
	balancePath       string
	factionsPath      string
	unitsDir          string
	npcConfigPath     string
	combatPath        string
	generalsPath      string
	itemsPath         string
	dropPoolsPath     string
	fishingPath       string
	slotPath          string
	reincarnationPath string
}

// getPlayerLock 获取指定玩家的互斥锁（懒创建）
func (s *Service) getPlayerLock(playerID string) *sync.Mutex {
	val, _ := s.playerLocks.LoadOrStore(playerID, &sync.Mutex{})
	return val.(*sync.Mutex)
}

// tryPlayerLockIfIdle 尝试获取玩家锁；若同一存档已有关键操作在执行，立即返回失败。
func (s *Service) tryPlayerLockIfIdle(playerID string) (func(), bool) {
	lock := s.getPlayerLock(playerID)
	if !lock.TryLock() {
		return nil, false
	}
	return lock.Unlock, true
}

type BootstrapResponse struct {
	GameName      string              `json:"gameName"`
	Modules       []string            `json:"modules"`
	Balance       BalanceConfig       `json:"balance"`
	Factions      FactionsConfig      `json:"factions"`
	Units         UnitsConfig         `json:"units"`
	Items         ItemsConfig         `json:"items"`
	DropPools     DropPoolsConfig     `json:"dropPools"`
	Combat        combat.CombatConfig `json:"combat"`
	Fishing       FishingConfig       `json:"fishing"`
	Slot          SlotConfig          `json:"slot"`
	Reincarnation ReincarnationConfig `json:"reincarnation"`
	Message       string              `json:"message"`
}

const (
	newPlayerRewardMailType = "reward"
	newPlayerRewardGold     = 10000
)

func NewService() *Service {
	return NewServiceWithRepository(NewMemoryRepository())
}

func NewServiceWithRepository(repo Repository) *Service {
	return &Service{repo: repo, eventBus: NewEventBus()}
}

func (s *Service) SubscribeEvent(eventType string, handler EventHandler) {
	if s.eventBus == nil {
		s.eventBus = NewEventBus()
	}
	s.eventBus.Subscribe(eventType, handler)
}

func (s *Service) publishEvent(event GameEvent) {
	if s.eventBus == nil {
		s.eventBus = NewEventBus()
	}
	s.eventBus.Publish(event)
}

func (s *Service) SetBalancePath(path string) error {
	s.balancePath = path
	return LoadBalanceConfig(path)
}

func (s *Service) SetFactionsPath(path string) error {
	s.factionsPath = path
	return LoadFactionsConfig(path)
}

func (s *Service) SetUnitsDir(dir string) error {
	s.unitsDir = dir
	return LoadUnitsConfig(dir)
}

func (s *Service) SetCombatPath(path string) error {
	s.combatPath = path
	return combat.LoadCombatConfig(path)
}

func (s *Service) SetGeneralsPath(path string) error {
	s.generalsPath = path
	return LoadGeneralsConfig(path)
}

func (s *Service) SetItemsPath(path string) error {
	s.itemsPath = path
	return LoadItemsConfig(path)
}

func (s *Service) SetDropPoolsPath(path string) error {
	s.dropPoolsPath = path
	return LoadDropPoolsConfig(path)
}

func (s *Service) SetFishingPath(path string) error {
	s.fishingPath = path
	return LoadFishingConfig(path)
}

// SetSlotPath 设置并加载天机轮转配置。
func (s *Service) SetSlotPath(path string) error {
	s.slotPath = path
	return LoadSlotConfig(path)
}

// SetReincarnationPath 设置并加载轮回绝境配置。
func (s *Service) SetReincarnationPath(path string) error {
	s.reincarnationPath = path
	return LoadReincarnationConfig(path)
}

func (s *Service) GetGeneralsConfig() GeneralsConfig {
	return GetGeneralsConfig()
}

func (s *Service) UpdateGeneralsConfig(cfg GeneralsConfig) error {
	return SaveGeneralsConfig(s.generalsPath, cfg)
}

func (s *Service) GetCombatConfig() combat.CombatConfig {
	return combat.GetCombatConfig()
}

func (s *Service) UpdateCombatConfig(config combat.CombatConfig) error {
	return combat.SaveCombatConfig(s.combatPath, config)
}

func (s *Service) GetFishingConfig() FishingConfig {
	return GetFishingConfig()
}

func (s *Service) UpdateFishingConfig(config FishingConfig) error {
	return SaveFishingConfig(s.fishingPath, config)
}

// GetSlotConfig 返回天机轮转配置。
func (s *Service) GetSlotConfig() SlotConfig {
	return GetSlotConfig()
}

// UpdateSlotConfig 保存天机轮转配置。
func (s *Service) UpdateSlotConfig(config SlotConfig) error {
	return SaveSlotConfig(s.slotPath, config)
}

// GetReincarnationConfig 返回轮回绝境配置。
func (s *Service) GetReincarnationConfig() ReincarnationConfig {
	return GetReincarnationConfig()
}

// UpdateReincarnationConfig 保存轮回绝境配置。
func (s *Service) UpdateReincarnationConfig(config ReincarnationConfig) error {
	return SaveReincarnationConfig(s.reincarnationPath, config)
}

func (s *Service) GetFactionsConfig() FactionsConfig {
	return GetFactionsConfig()
}

func (s *Service) UpdateFactionsConfig(config FactionsConfig) error {
	return SaveFactionsConfig(s.factionsPath, config)
}

func (s *Service) GetUnitsConfig() UnitsConfig {
	return GetUnitsConfig()
}

func (s *Service) UpdateFactionUnits(faction string, units FactionUnits) error {
	return SaveFactionUnits(s.unitsDir, faction, units)
}

func (s *Service) GetBalance() BalanceConfig {
	return GetBalanceConfig()
}

func (s *Service) UpdateBalance(config BalanceConfig) error {
	if err := SetBalanceConfig(config); err != nil {
		return err
	}
	return SaveBalanceConfig(s.balancePath, GetBalanceConfig())
}

func (s *Service) GetNpcConfig() NpcConfig {
	return GetNpcConfig()
}

func (s *Service) UpdateNpcConfig(config NpcConfig) error {
	if err := SaveNpcConfig(s.npcConfigPath, config); err != nil {
		return err
	}
	return SetNpcConfig(config)
}

// RegisterAccount 注册轻账号，并保存密码哈希。
func (s *Service) RegisterAccount(username string, password string) (Account, error) {
	username = strings.TrimSpace(username)
	if username == "" || strings.TrimSpace(password) == "" {
		return Account{}, ErrInvalidCredentials
	}

	account := Account{
		ID:           "acc_" + randomID(12),
		Username:     username,
		PasswordHash: hashPassword(password),
		CreatedAt:    time.Now(),
	}
	if err := s.repo.CreateAccount(account); err != nil {
		return Account{}, err
	}

	return account, nil
}

// LoginAccount 校验账号密码，数据库异常会原样返回给接口层处理。
func (s *Service) LoginAccount(username string, password string) (Account, error) {
	username = strings.TrimSpace(username)

	account, err := s.repo.GetAccountByUsername(username)
	if err != nil {
		if errors.Is(err, ErrAccountNotFound) {
			return Account{}, ErrInvalidCredentials
		}
		return Account{}, err
	}

	if account.PasswordHash != hashPassword(password) {
		return Account{}, ErrInvalidCredentials
	}

	return account, nil
}

func (s *Service) ListPlayers(accountID string) ([]PlayerSummary, error) {
	players, err := s.repo.ListPlayers(accountID)
	if err != nil {
		return nil, err
	}
	if !s.purgeDuePlayerDeletions(players, time.Now()) {
		return players, nil
	}
	return s.repo.ListPlayers(accountID)
}

func (s *Service) ListAccounts() ([]AccountSummary, error) {
	return s.repo.ListAccounts()
}

func (s *Service) GetAccountByID(accountID string) (Account, error) {
	return s.repo.GetAccountByID(accountID)
}

func (s *Service) ListGoldLedger(filter GoldLedgerFilter) ([]GoldLedgerEntry, error) {
	return s.repo.ListGoldLedger(filter)
}

func (s *Service) AddAccountGoldAdmin(accountID string, amount int) error {
	accountID = strings.TrimSpace(accountID)
	if accountID == "" {
		return ErrAccountNotFound
	}
	if amount <= 0 {
		return ErrInvalidGoldAmount
	}

	if err := s.repo.AddAccountGold(accountID, amount); err != nil {
		return err
	}
	account, err := s.repo.GetAccountByID(accountID)
	if err != nil {
		return err
	}
	s.recordLedger(GoldLedgerEntry{
		AccountID:    accountID,
		Currency:     LedgerCurrencyGold,
		Direction:    LedgerDirectionCredit,
		Amount:       amount,
		BalanceAfter: account.Gold,
		RefType:      LedgerRefAdminAdjust,
		Reason:       "admin_add_account_gold",
	})
	s.publishCurrencyChanged("", accountID, "", LedgerRefAdminAdjust)
	return nil
}

func (s *Service) CreatePlayer(accountID string, nickname string, faction string, generalID string) (string, GameState, error) {
	nickname = strings.TrimSpace(nickname)
	faction = strings.TrimSpace(faction)
	generalID = strings.TrimSpace(generalID)
	if nickname == "" || faction == "" {
		return "", GameState{}, ErrPlayerNotFound
	}

	factions := GetFactionsConfig()
	fc, factionExists := factions[faction]
	if !factionExists {
		return "", GameState{}, ErrInvalidGeneral
	}
	if generalID == "" {
		generalID = defaultGeneralForFaction(faction, fc)
	}
	valid := false
	for _, g := range fc.Generals {
		if g.ID == generalID {
			valid = true
			break
		}
	}
	if !valid {
		return "", GameState{}, ErrInvalidGeneral
	}
	hero, ok := GetHeroConfig(generalID)
	if !ok || !hero.Enabled || hero.Faction != faction {
		return "", GameState{}, ErrInvalidGeneral
	}

	exists, err := s.repo.AccountExists(accountID)
	if err != nil {
		return "", GameState{}, err
	}
	if !exists {
		return "", GameState{}, ErrAccountNotFound
	}

	now := time.Now()
	playerID := "player_" + randomID(12)
	state := newPlayerState(playerID, nickname, faction, generalID, now)
	mailCode, err := s.generateMailCode(nickname)
	if err != nil {
		return "", GameState{}, err
	}
	state.Player.MailCode = mailCode
	if err := s.repo.CreatePlayer(accountID, state, now); err != nil {
		return "", GameState{}, err
	}
	createCoordinate, err := generateWorldMapCreateCoordinate()
	if err != nil {
		s.cleanupFailedPlayerCreate(playerID)
		return "", GameState{}, err
	}
	if _, err := s.ensureWorldPosition(playerID, "create", &createCoordinate); err != nil {
		s.cleanupFailedPlayerCreate(playerID)
		return "", GameState{}, err
	}
	if _, err := s.sendNewPlayerRewardMail(playerID); err != nil {
		s.cleanupFailedPlayerCreate(playerID)
		return "", GameState{}, err
	}
	s.publishEvent(GameEvent{
		Type:      EventPlayerCreated,
		PlayerID:  playerID,
		AccountID: accountID,
		RefType:   "player_create",
		RefID:     playerID,
		Payload: map[string]any{
			"nickname": nickname,
			"faction":  faction,
			"general":  generalID,
		},
		CreatedAt: now.UTC().Format(resourceDateLayout),
	})

	return playerID, state, nil
}

// cleanupFailedPlayerCreate 清理创建流程后半段失败时已经写入的玩家记录。
func (s *Service) cleanupFailedPlayerCreate(playerID string) {
	_ = s.repo.DeletePlayer(playerID)
}

// sendNewPlayerRewardMail 给新建角色投递可领取的新手奖励信函。
func (s *Service) sendNewPlayerRewardMail(playerID string) (Mail, error) {
	return s.SendMail(SendMailRequest{
		PlayerID:   playerID,
		MailType:   newPlayerRewardMailType,
		SenderType: "system",
		SenderName: "系统",
		Title:      "新手奖励",
		Content:    "欢迎来到 Hero3。请领取新手奖励，愿这笔金币助你快速建立第一座强城。",
		Attachments: []MailAttachment{{
			Type:   RewardTypeGold,
			ItemID: RewardTypeGold,
			Amount: newPlayerRewardGold,
		}},
		SourceType: "system",
		SourceID:   "new_player_reward",
	})
}

// defaultGeneralForFaction 返回阵营第一个可用将领，用于兼容未显式选择将领的旧客户端。
func defaultGeneralForFaction(faction string, fc FactionConfig) string {
	for _, general := range fc.Generals {
		hero, ok := GetHeroConfig(general.ID)
		if ok && hero.Enabled && hero.Faction == faction {
			return general.ID
		}
	}
	return ""
}

func (s *Service) DeleteAccount(accountID string) error {
	accountID = strings.TrimSpace(accountID)
	if accountID == "" {
		return ErrAccountNotFound
	}
	return s.repo.DeleteAccount(accountID)
}

func (s *Service) DeletePlayer(playerID string) (PlayerDeletionResult, error) {
	playerID = strings.TrimSpace(playerID)
	if playerID == "" {
		return PlayerDeletionResult{}, ErrPlayerNotFound
	}
	state, err := s.repo.GetState(playerID)
	if err != nil {
		return PlayerDeletionResult{}, err
	}
	now := time.Now().UTC()
	if state.DeleteScheduledAt != "" {
		if scheduledAt, err := time.Parse(resourceDateLayout, state.DeleteScheduledAt); err == nil && !now.Before(scheduledAt) {
			if err := s.repo.DeletePlayer(playerID); err != nil {
				return PlayerDeletionResult{}, err
			}
			return PlayerDeletionResult{Status: "deleted", PlayerID: playerID}, nil
		}
		return PlayerDeletionResult{
			Status:            "scheduled",
			PlayerID:          playerID,
			DeleteRequestedAt: state.DeleteRequestedAt,
			DeleteScheduledAt: state.DeleteScheduledAt,
		}, nil
	}
	requestedAt := now.Format(resourceDateLayout)
	scheduledAt := now.Add(playerDeletionDelay).Format(resourceDateLayout)
	if _, err := s.repo.UpdatePlayerState(playerID, now, func(state *GameState) error {
		state.DeleteRequestedAt = requestedAt
		state.DeleteScheduledAt = scheduledAt
		state.ServerTime = requestedAt
		return nil
	}); err != nil {
		return PlayerDeletionResult{}, err
	}
	return PlayerDeletionResult{
		Status:            "scheduled",
		PlayerID:          playerID,
		DeleteRequestedAt: requestedAt,
		DeleteScheduledAt: scheduledAt,
	}, nil
}

func (s *Service) RestorePlayerDeletion(playerID string) (PlayerDeletionResult, error) {
	playerID = strings.TrimSpace(playerID)
	if playerID == "" {
		return PlayerDeletionResult{}, ErrPlayerNotFound
	}
	now := time.Now().UTC()
	state, err := s.repo.UpdatePlayerState(playerID, now, func(state *GameState) error {
		state.DeleteRequestedAt = ""
		state.DeleteScheduledAt = ""
		state.ServerTime = now.Format(resourceDateLayout)
		return nil
	})
	if err != nil {
		return PlayerDeletionResult{}, err
	}
	return PlayerDeletionResult{Status: "restored", PlayerID: state.Player.ID}, nil
}

func (s *Service) purgeDuePlayerDeletions(players []PlayerSummary, now time.Time) bool {
	purged := false
	now = now.UTC()
	for _, player := range players {
		if player.DeleteScheduledAt == "" {
			continue
		}
		scheduledAt, err := time.Parse(resourceDateLayout, player.DeleteScheduledAt)
		if err != nil || now.Before(scheduledAt) {
			continue
		}
		if err := s.repo.DeletePlayer(player.ID); err == nil {
			purged = true
		}
	}
	return purged
}

func (s *Service) GetState(playerID string) (GameState, error) {
	playerID = strings.TrimSpace(playerID)
	if playerID == "" {
		return GameState{}, ErrPlayerNotFound
	}

	now := time.Now()
	state, err := s.repo.GetState(playerID)
	if err != nil {
		return GameState{}, err
	}

	applyGeneralConfigForView(&state)
	state, _ = settleResources(state, now)

	s.attachReportSummary(&state, playerID)
	s.attachMailSummary(&state, playerID)

	hydrateStateForResponse(&state, now)

	return state, nil
}

// applyGeneralConfigForView 只为读取响应补充武将配置派生字段，不补齐长期武将资产。
func applyGeneralConfigForView(state *GameState) {
	if state == nil {
		return
	}
	if state.General != nil {
		applyHeroConfigToGeneral(state.General)
	}
	for index := range state.Generals {
		applyHeroConfigToGeneral(&state.Generals[index])
	}
}

func (s *Service) attachReportSummary(state *GameState, playerID string) {
	if state == nil {
		return
	}

	// 军情列表已经由 /news/reports 分页接口承载，状态入口只保留未读数量。
	state.RecentBattleReports = []BattleReport{}

	unreadCount, countErr := s.repo.CountUnreadReports(playerID)
	if countErr != nil {
		slog.Warn("count unread reports failed", "error", countErr)
		return
	}
	state.UnreadMessageCount = unreadCount
}

func (s *Service) attachMailSummary(state *GameState, playerID string) {
	if state == nil {
		return
	}
	unreadCount, err := s.repo.CountUnreadMails(playerID)
	if err != nil {
		slog.Warn("count unread mails failed", "error", err)
		return
	}
	state.UnreadMailCount = unreadCount
}

func hydrateStateForResponse(state *GameState, now time.Time) {
	if state == nil {
		return
	}
	state.ServerTime = now.UTC().Format(resourceDateLayout)
	normalizeInventoryState(state, now)
	state.ActiveModifiers = GetModifierBreakdown(state, now)
}

func (s *Service) Bootstrap() BootstrapResponse {
	balance := currentBalance()
	factions := GetFactionsConfig()
	units := GetUnitsConfig()
	items := GetItemsConfig()
	dropPools := GetDropPoolsConfig()
	combatConfig := combat.GetCombatConfig()
	fishing := GetFishingConfig()
	slot := GetSlotConfig()
	reincarnation := GetReincarnationConfig()
	return BootstrapResponse{
		GameName: "Hero3",
		Modules: append([]string{
			"player",
			"city",
			"resource",
			"military",
			"map",
			"combat",
			"save",
			"item",
		}, ListGameplayModuleIDs()...),
		Balance:       balance,
		Factions:      factions,
		Units:         units,
		Items:         items,
		DropPools:     dropPools,
		Combat:        combatConfig,
		Fishing:       fishing,
		Slot:          slot,
		Reincarnation: reincarnation,
		Message:       "Hero3 后端基础服务已就绪，具体玩法逻辑待接入。",
	}
}

func hashPassword(password string) string {
	sum := sha256.Sum256([]byte(password))
	return hex.EncodeToString(sum[:])
}

func randomID(bytesCount int) string {
	bytes := make([]byte, bytesCount)
	if _, err := rand.Read(bytes); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(bytes)
}

func (s *Service) generateMailCode(nickname string) (string, error) {
	nickname = strings.TrimSpace(nickname)
	if nickname == "" {
		return "", ErrPlayerNotFound
	}
	for i := 0; i < 20; i++ {
		raw := randomID(3)
		if len(raw) > 6 {
			raw = raw[:6]
		}
		code := ""
		for _, ch := range raw {
			if ch >= '0' && ch <= '9' {
				code += string(ch)
			} else {
				code += fmt.Sprintf("%d", int(ch)%10)
			}
			if len(code) == 6 {
				break
			}
		}
		for len(code) < 6 {
			code += "0"
		}
		exists, err := s.repo.MailAddressExists(nickname, code)
		if err != nil {
			return "", err
		}
		if !exists {
			return code, nil
		}
	}
	return "", ErrInvalidMail
}

// OwnsPlayer 校验指定 accountID 是否拥有指定 playerID
// 用于认证中间件的归属校验
func (s *Service) OwnsPlayer(accountID string, playerID string) (bool, error) {
	accountID = strings.TrimSpace(accountID)
	playerID = strings.TrimSpace(playerID)
	if accountID == "" || playerID == "" {
		return false, nil
	}

	players, err := s.repo.ListPlayers(accountID)
	if err != nil {
		if errors.Is(err, ErrAccountNotFound) {
			return false, nil
		}
		return false, err
	}

	for _, p := range players {
		if p.ID == playerID {
			return true, nil
		}
	}
	return false, nil
}
