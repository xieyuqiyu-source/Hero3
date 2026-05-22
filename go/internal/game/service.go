package game

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"
)

var (
	ErrAccountExists      = errors.New("account already exists")
	ErrAccountNotFound    = errors.New("account not found")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrPlayerNotFound     = errors.New("player not found")
	ErrBuildingNotFound   = errors.New("building not found")
	ErrInsufficientRes    = errors.New("insufficient resources")
	ErrAlreadyUpgrading   = errors.New("building is already upgrading")
	ErrMaxLevel           = errors.New("building is at max level")
)

const resourceDateLayout = time.RFC3339

type Service struct {
	repo        Repository
	balancePath string
}

type BootstrapResponse struct {
	GameName string   `json:"gameName"`
	Modules  []string `json:"modules"`
	Message  string   `json:"message"`
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

func (s *Service) GetBalance() BalanceConfig {
	return GetBalanceConfig()
}

func (s *Service) UpdateBalance(config BalanceConfig) error {
	if err := SaveBalanceConfig(s.balancePath, config); err != nil {
		return err
	}
	return SetBalanceConfig(config)
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

func (s *Service) CreatePlayer(accountID string, nickname string, faction string) (string, GameState, error) {
	nickname = strings.TrimSpace(nickname)
	faction = strings.TrimSpace(faction)
	if nickname == "" || faction == "" {
		return "", GameState{}, ErrPlayerNotFound
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
	state := newPlayerState(playerID, nickname, faction, now)
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
	if playerID == "" || playerID == "demo-player" {
		return ErrPlayerNotFound
	}
	return s.repo.DeletePlayer(playerID)
}

func (s *Service) FillResources(playerID string) (GameState, error) {
	playerID = strings.TrimSpace(playerID)
	if playerID == "" {
		return GameState{}, ErrPlayerNotFound
	}

	state, err := s.repo.GetState(playerID)
	if err != nil {
		return GameState{}, err
	}

	now := time.Now()
	state, _ = settleResources(state, now)

	// 将所有资源填满到容量上限
	for resType, cap := range state.Resources.Capacity {
		state.Resources.Items[resType] = cap
	}

	state.ResourceSettledAt = now.UTC().Format(resourceDateLayout)
	state.ServerTime = now.UTC().Format(resourceDateLayout)

	if err := s.repo.SaveState(state, now); err != nil {
		return GameState{}, err
	}

	return state, nil
}

func (s *Service) UpgradeBuilding(playerID string, buildingID string) (GameState, error) {
	playerID = strings.TrimSpace(playerID)
	buildingID = strings.TrimSpace(buildingID)
	if playerID == "" || buildingID == "" {
		return GameState{}, ErrBuildingNotFound
	}

	state, err := s.repo.GetState(playerID)
	if err != nil {
		return GameState{}, err
	}

	now := time.Now()

	// 先结算资源和已完成的升级
	state, _ = settleResources(state, now)

	// 找到目标建筑
	buildingIdx := -1
	for i, b := range state.Buildings {
		if b.ID == buildingID {
			buildingIdx = i
			break
		}
	}
	if buildingIdx == -1 {
		return GameState{}, ErrBuildingNotFound
	}

	building := &state.Buildings[buildingIdx]

	// 检查是否正在升级
	if building.UpgradeEndsAt != nil {
		return GameState{}, ErrAlreadyUpgrading
	}

	// 获取建筑配置
	config, exists := getBuildingConfig(building.Type)
	if !exists {
		return GameState{}, ErrBuildingNotFound
	}

	// 检查是否已满级
	currentLevel := building.Level
	upgradeCost, hasCost := config.UpgradeCostByLevel[currentLevel]
	if !hasCost {
		return GameState{}, ErrMaxLevel
	}

	// 检查资源是否足够
	for resType, cost := range upgradeCost {
		if state.Resources.Items[resType] < cost {
			return GameState{}, ErrInsufficientRes
		}
	}

	// 扣减资源
	for resType, cost := range upgradeCost {
		state.Resources.Items[resType] -= cost
	}

	// 设置升级倒计时
	upgradeSeconds := 60 // 默认
	if seconds, ok := config.UpgradeSecondsByLevel[currentLevel]; ok {
		upgradeSeconds = seconds
	}
	endsAt := now.Add(time.Duration(upgradeSeconds) * time.Second).UTC().Format(resourceDateLayout)
	building.UpgradeEndsAt = &endsAt

	// 更新结算时间和服务器时间
	state.ResourceSettledAt = now.UTC().Format(resourceDateLayout)
	state.ServerTime = now.UTC().Format(resourceDateLayout)

	// 保存
	if err := s.repo.SaveState(state, now); err != nil {
		return GameState{}, err
	}

	return state, nil
}

func (s *Service) UpgradeBuildingBatch(playerID string) (GameState, int, error) {
	playerID = strings.TrimSpace(playerID)
	if playerID == "" {
		return GameState{}, 0, ErrPlayerNotFound
	}

	state, err := s.repo.GetState(playerID)
	if err != nil {
		return GameState{}, 0, err
	}

	now := time.Now()
	state, _ = settleResources(state, now)

	// 收集可升级的资源建筑（未在升级中、有升级配置）按等级排序
	type candidate struct {
		index int
		level int
	}
	var candidates []candidate
	for i, b := range state.Buildings {
		if b.UpgradeEndsAt != nil {
			continue
		}
		config, exists := getBuildingConfig(b.Type)
		if !exists || config.ResourceType == "" {
			continue // 只处理资源建筑
		}
		if _, hasCost := config.UpgradeCostByLevel[b.Level]; !hasCost {
			continue // 已满级
		}
		candidates = append(candidates, candidate{index: i, level: b.Level})
	}

	// 按等级从低到高排序
	for i := 0; i < len(candidates)-1; i++ {
		for j := i + 1; j < len(candidates); j++ {
			if candidates[j].level < candidates[i].level {
				candidates[i], candidates[j] = candidates[j], candidates[i]
			}
		}
	}

	upgraded := 0
	for _, c := range candidates {
		building := &state.Buildings[c.index]
		config, _ := getBuildingConfig(building.Type)
		upgradeCost := config.UpgradeCostByLevel[building.Level]

		// 检查资源是否足够
		enough := true
		for resType, cost := range upgradeCost {
			if state.Resources.Items[resType] < cost {
				enough = false
				break
			}
		}
		if !enough {
			continue
		}

		// 扣减资源
		for resType, cost := range upgradeCost {
			state.Resources.Items[resType] -= cost
		}

		// 设置升级倒计时
		upgradeSeconds := 60
		if seconds, ok := config.UpgradeSecondsByLevel[building.Level]; ok {
			upgradeSeconds = seconds
		}
		endsAt := now.Add(time.Duration(upgradeSeconds) * time.Second).UTC().Format(resourceDateLayout)
		building.UpgradeEndsAt = &endsAt
		upgraded++
	}

	if upgraded == 0 {
		return state, 0, ErrInsufficientRes
	}

	state.ResourceSettledAt = now.UTC().Format(resourceDateLayout)
	state.ServerTime = now.UTC().Format(resourceDateLayout)

	if err := s.repo.SaveState(state, now); err != nil {
		return GameState{}, 0, err
	}

	return state, upgraded, nil
}

func (s *Service) GetState(playerID string) (GameState, error) {
	if strings.TrimSpace(playerID) == "" {
		playerID = "demo-player"
	}

	state, err := s.repo.GetState(playerID)
	if err != nil {
		if errors.Is(err, ErrPlayerNotFound) && playerID == "demo-player" {
			state = newDemoState(time.Now())
			return state, nil
		}
		return GameState{}, err
	}

	state, changed := settleResources(state, time.Now())
	if changed {
		if err := s.repo.SaveState(state, time.Now()); err != nil {
			return GameState{}, err
		}
	}
	return state, nil
}

func (s *Service) Bootstrap() BootstrapResponse {
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
		Message: "Hero3 后端基础服务已就绪，具体玩法逻辑待接入。",
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

func settleResources(state GameState, now time.Time) (GameState, bool) {
	now = now.UTC()
	changed := false

	// 检查并完成已到期的建筑升级
	for i, b := range state.Buildings {
		if b.UpgradeEndsAt == nil {
			continue
		}
		endsAt, err := time.Parse(resourceDateLayout, *b.UpgradeEndsAt)
		if err != nil {
			continue
		}
		if now.After(endsAt) || now.Equal(endsAt) {
			state.Buildings[i].Level++
			state.Buildings[i].UpgradeEndsAt = nil
			changed = true
		}
	}

	production := calculateResourceProduction(state.Buildings)
	if !reflect.DeepEqual(state.ResourceProduction, production) {
		state.ResourceProduction = production
		changed = true
	}
	capacity := calculateResourceCapacity(state.Buildings)
	if !reflect.DeepEqual(state.Resources.Capacity, capacity) {
		state.Resources.Capacity = capacity
		changed = true
	}
	if state.Resources.Items == nil {
		state.Resources.Items = map[string]int{}
		changed = true
	}

	settledAt := now
	if strings.TrimSpace(state.ResourceSettledAt) != "" {
		if parsed, err := time.Parse(resourceDateLayout, state.ResourceSettledAt); err == nil {
			settledAt = parsed.UTC()
		}
	} else {
		state.ResourceSettledAt = now.Format(resourceDateLayout)
		changed = true
	}

	if now.After(settledAt) {
		elapsedSeconds := now.Sub(settledAt).Seconds()
		nextResources := state.Resources
		nextResources.Items = copyResourceMap(state.Resources.Items)
		for resourceType, perHour := range state.ResourceProduction {
			nextResources.Items[resourceType] = addProducedResource(
				nextResources.Items[resourceType],
				perHour,
				elapsedSeconds,
				nextResources.Capacity[resourceType],
			)
		}

		if !reflect.DeepEqual(nextResources, state.Resources) || hasWholeResourceTick(state.ResourceProduction, elapsedSeconds) {
			state.Resources = nextResources
			state.ResourceSettledAt = now.Format(resourceDateLayout)
			changed = true
		}
	}

	state.ServerTime = now.Format(resourceDateLayout)
	return state, changed
}

func addProducedResource(current int, perHour int, elapsedSeconds float64, capacity int) int {
	if current >= capacity || perHour <= 0 || elapsedSeconds <= 0 {
		return min(current, capacity)
	}

	produced := int(float64(perHour) * elapsedSeconds / 3600)
	if produced <= 0 {
		return current
	}

	return min(current+produced, capacity)
}

func hasWholeResourceTick(production ResourceProduction, elapsedSeconds float64) bool {
	for _, perHour := range production {
		if perHour*int(elapsedSeconds) >= 3600 {
			return true
		}
	}
	return false
}

func calculateResourceProduction(buildings []Building) ResourceProduction {
	production := ResourceProduction{}
	balance := currentBalance()
	for resourceType, value := range balance.BaseProduction {
		production[resourceType] = value
	}

	for _, building := range buildings {
		config, exists := getBuildingConfig(building.Type)
		if !exists || config.ResourceType == "" {
			continue
		}
		production[config.ResourceType] += valueByLevel(config.ProductionByLevel, building.Level)
	}

	return production
}

func calculateResourceCapacity(buildings []Building) map[string]int {
	balance := currentBalance()
	capacity := valueByLevel(balance.Buildings["warehouse"].CapacityByLevel, 0)
	for _, building := range buildings {
		config, exists := getBuildingConfig(building.Type)
		if !exists || len(config.CapacityByLevel) == 0 {
			continue
		}
		capacity = valueByLevel(config.CapacityByLevel, building.Level)
	}
	return map[string]int{
		"wood":  capacity,
		"stone": capacity,
		"iron":  capacity,
		"food":  capacity,
	}
}

func copyResourceMap(source map[string]int) map[string]int {
	next := make(map[string]int, len(source))
	for key, value := range source {
		next[key] = value
	}
	return next
}

func coreResourceTypes() []string {
	return []string{"wood", "stone", "iron", "food"}
}
