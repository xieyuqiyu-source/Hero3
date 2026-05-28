package game

import (
	"errors"
	"strings"
	"time"
)

var (
	ErrBoostActive       = errors.New("production boost is still active")
	ErrInvalidBoost      = errors.New("invalid boost multiplier")
	ErrInvalidDuration   = errors.New("invalid boost duration")
)

// 允许的加成倍率
var validBoostMultipliers = map[int]bool{2: true, 4: true, 8: true, 16: true}

// 允许的持续时间（小时）
var validBoostHours = map[int]bool{1: true, 6: true, 12: true, 24: true}

// boostCost 计算产量加成的城金花费（从 balance 配置读取）
func boostCost(multiplier int, hours int) int {
	cfg := currentBalance()
	baseCost := cfg.BoostBaseCost
	if baseCost <= 0 {
		baseCost = 30
	}

	mf := 1
	if cfg.BoostMultiplierFactor != nil {
		if v, ok := cfg.BoostMultiplierFactor[multiplier]; ok {
			mf = v
		}
	}

	df := 1
	if cfg.BoostDurationFactor != nil {
		if v, ok := cfg.BoostDurationFactor[hours]; ok {
			df = v
		}
	}

	return baseCost * mf * df
}

// PurchaseBoost 购买产量加成（消耗城金）
func (s *Service) PurchaseBoost(playerID string, multiplier int, hours int) (GameState, error) {
	playerID = strings.TrimSpace(playerID)
	if playerID == "" {
		return GameState{}, ErrPlayerNotFound
	}
	if !validBoostMultipliers[multiplier] {
		return GameState{}, ErrInvalidBoost
	}
	if !validBoostHours[hours] {
		return GameState{}, ErrInvalidDuration
	}

	state, err := s.repo.GetState(playerID)
	if err != nil {
		return GameState{}, err
	}

	now := time.Now()
	state, _ = settleResources(state, now)

	// 检查是否已有活跃加成
	if state.ProductionBoost > 1 && state.ProductionBoostEnd != "" {
		expiresAt, parseErr := time.Parse(resourceDateLayout, state.ProductionBoostEnd)
		if parseErr == nil && now.Before(expiresAt) {
			return GameState{}, ErrBoostActive
		}
	}

	// 计算费用并扣除城金
	cost := boostCost(multiplier, hours)
	if _, err := s.repo.DeductCityGold(playerID, cost); err != nil {
		return GameState{}, err
	}

	// 设置加成
	state, _ = s.repo.GetState(playerID) // 重新读取（城金已扣）
	state.ProductionBoost = multiplier
	state.ProductionBoostEnd = now.Add(time.Duration(hours) * time.Hour).UTC().Format(resourceDateLayout)
	state.ServerTime = now.UTC().Format(resourceDateLayout)

	// 通过 Modifier 管线重新计算产量（含加成）
	modSources := CollectModifierSources(&state)
	production := calculateResourceProduction(state.Buildings, state.General)
	production = applyProductionModifiers(production, now, modSources)
	state.ResourceProduction = production
	state.ResourceSettledAt = now.UTC().Format(resourceDateLayout)

	if err := s.repo.SaveState(state, now); err != nil {
		return GameState{}, err
	}

	return state, nil
}

// GetBoostCost 查询加成价格（供前端展示）
func GetBoostCost(multiplier int, hours int) int {
	return boostCost(multiplier, hours)
}

// PurchaseCapacityBoost 购买仓库容量加成（消耗城金，价格同产量加成）
func (s *Service) PurchaseCapacityBoost(playerID string, multiplier int, hours int) (GameState, error) {
	playerID = strings.TrimSpace(playerID)
	if playerID == "" {
		return GameState{}, ErrPlayerNotFound
	}
	if !validBoostMultipliers[multiplier] {
		return GameState{}, ErrInvalidBoost
	}
	if !validBoostHours[hours] {
		return GameState{}, ErrInvalidDuration
	}

	state, err := s.repo.GetState(playerID)
	if err != nil {
		return GameState{}, err
	}

	now := time.Now()
	state, _ = settleResources(state, now)

	// 检查是否已有活跃容量加成
	if state.CapacityBoost > 1 && state.CapacityBoostEnd != "" {
		expiresAt, parseErr := time.Parse(resourceDateLayout, state.CapacityBoostEnd)
		if parseErr == nil && now.Before(expiresAt) {
			return GameState{}, ErrBoostActive
		}
	}

	// 计算费用并扣除城金
	cost := boostCost(multiplier, hours)
	if _, err := s.repo.DeductCityGold(playerID, cost); err != nil {
		return GameState{}, err
	}

	// 设置容量加成
	state, _ = s.repo.GetState(playerID)
	state.CapacityBoost = multiplier
	state.CapacityBoostEnd = now.Add(time.Duration(hours) * time.Hour).UTC().Format(resourceDateLayout)
	state.ServerTime = now.UTC().Format(resourceDateLayout)

	// 通过 Modifier 管线重新计算容量（含加成）
	modSources := CollectModifierSources(&state)
	capacity := calculateResourceCapacity(state.Buildings)
	capacity = applyCapacityModifiers(capacity, now, modSources)
	state.Resources.Capacity = capacity
	state.ResourceSettledAt = now.UTC().Format(resourceDateLayout)

	if err := s.repo.SaveState(state, now); err != nil {
		return GameState{}, err
	}

	return state, nil
}
