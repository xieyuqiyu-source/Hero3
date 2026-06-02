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

	"hero3/internal/combat"
)

var (
	ErrAccountExists      = errors.New("account already exists")
	ErrAccountNotFound    = errors.New("account not found")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrPlayerNotFound     = errors.New("player not found")
	ErrBuildingNotFound   = errors.New("building not found")
	ErrInsufficientRes    = errors.New("insufficient resources")
	ErrAlreadyUpgrading   = errors.New("building is already upgrading")
	ErrNotUpgrading       = errors.New("building is not upgrading")
	ErrMaxLevel           = errors.New("building is at max level")
	ErrUnitNotFound       = errors.New("unit not found")
	ErrInvalidAmount      = errors.New("invalid recruit amount")
	ErrQueueFull          = errors.New("recruit queue is full")
	ErrInvalidGeneral     = errors.New("invalid general for faction")
)

const resourceDateLayout = time.RFC3339

type Service struct {
	repo            Repository
	playerLocks     sync.Map // per-player 互斥锁，防止并发购买/兑换竞态
	balancePath     string
	factionsPath    string
	unitsDir        string
	npcConfigPath   string
	combatPath      string
	generalsPath    string
}

// getPlayerLock 获取指定玩家的互斥锁（懒创建）
func (s *Service) getPlayerLock(playerID string) *sync.Mutex {
	val, _ := s.playerLocks.LoadOrStore(playerID, &sync.Mutex{})
	return val.(*sync.Mutex)
}

type BootstrapResponse struct {
	GameName string         `json:"gameName"`
	Modules  []string       `json:"modules"`
	Balance  BalanceConfig  `json:"balance"`
	Factions FactionsConfig `json:"factions"`
	Units    UnitsConfig    `json:"units"`
	Message  string         `json:"message"`
}

func NewService() *Service {
	return NewServiceWithRepository(NewMemoryRepository())
}

func NewServiceWithRepository(repo Repository) *Service {
	return &Service{repo: repo}
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
	if err := SaveBalanceConfig(s.balancePath, config); err != nil {
		return err
	}
	return SetBalanceConfig(config)
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

func (s *Service) LoginAccount(username string, password string) (Account, error) {
	username = strings.TrimSpace(username)

	account, err := s.repo.GetAccountByUsername(username)
	if err != nil {
		return Account{}, ErrInvalidCredentials
	}

	if account.PasswordHash != hashPassword(password) {
		return Account{}, ErrInvalidCredentials
	}

	return account, nil
}

func (s *Service) ListPlayers(accountID string) ([]PlayerSummary, error) {
	return s.repo.ListPlayers(accountID)
}

func (s *Service) ListAccounts() ([]AccountSummary, error) {
	return s.repo.ListAccounts()
}

func (s *Service) GetAccountByID(accountID string) (Account, error) {
	return s.repo.GetAccountByID(accountID)
}

func (s *Service) AddAccountGoldAdmin(accountID string, amount int) error {
	return s.repo.AddAccountGold(accountID, amount)
}

func (s *Service) CreatePlayer(accountID string, nickname string, faction string, generalID string) (string, GameState, error) {
	nickname = strings.TrimSpace(nickname)
	faction = strings.TrimSpace(faction)
	generalID = strings.TrimSpace(generalID)
	if nickname == "" || faction == "" {
		return "", GameState{}, ErrPlayerNotFound
	}

	// 校验 generalID 是否属于所选阵营
	if generalID == "" {
		return "", GameState{}, ErrInvalidGeneral
	}
	factions := GetFactionsConfig()
	fc, factionExists := factions[faction]
	if !factionExists {
		return "", GameState{}, ErrInvalidGeneral
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
	if err := s.repo.CreatePlayer(accountID, state, now); err != nil {
		return "", GameState{}, err
	}

	return playerID, state, nil
}

func (s *Service) DeleteAccount(accountID string) error {
	accountID = strings.TrimSpace(accountID)
	if accountID == "" {
		return ErrAccountNotFound
	}
	return s.repo.DeleteAccount(accountID)
}

func (s *Service) DeletePlayer(playerID string) error {
	playerID = strings.TrimSpace(playerID)
	if playerID == "" {
		return ErrPlayerNotFound
	}
	return s.repo.DeletePlayer(playerID)
}

func (s *Service) GetState(playerID string) (GameState, error) {
	playerID = strings.TrimSpace(playerID)
	if playerID == "" {
		return GameState{}, ErrPlayerNotFound
	}

	state, err := s.repo.GetState(playerID)
	if err != nil {
		return GameState{}, err
	}

	state, changed := settleResources(state, time.Now())

	// 旧存档没有将领数据时，根据阵营分配默认将领
	if state.General == nil && state.Player.Faction != "" {
		defaultGenerals := map[string]string{
			"wei": "caocao",
			"shu": "liubei",
			"wu":  "sunquan",
		}
		if gid, ok := defaultGenerals[state.Player.Faction]; ok {
			state.General = newGeneral(state.Player.Faction, gid)
			changed = true
		}
	}

	if changed {
		if err := s.repo.SaveState(state, time.Now()); err != nil {
			return GameState{}, err
		}
	}

	// 总是重新应用 GeneralsConfig（GM 可能修改了配置，运行时同步生效）
	if state.General != nil {
		applyHeroConfigToGeneral(state.General)
	}

	// 从独立存储加载战报
	reports, listErr := s.repo.ListReports(playerID, 50)
	if listErr != nil {
		slog.Warn("list reports failed", "error", listErr)
	}
	state.RecentBattleReports = reports

	// 填充当前生效的加成明细（供前端 tooltip 展示）
	state.ActiveModifiers = GetModifierBreakdown(&state, time.Now())

	return state, nil
}

func (s *Service) Bootstrap() BootstrapResponse {
	balance := currentBalance()
	factions := GetFactionsConfig()
	units := GetUnitsConfig()
	return BootstrapResponse{
		GameName: "Hero3",
		Modules: []string{
			"player",
			"city",
			"resource",
			"military",
			"map",
			"combat",
			"save",
		},
		Balance:  balance,
		Factions: factions,
		Units:    units,
		Message:  "Hero3 后端基础服务已就绪，具体玩法逻辑待接入。",
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
