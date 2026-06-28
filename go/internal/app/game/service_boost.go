package game

import (
	"errors"
	"sort"
	"strings"
	"time"
)

var (
	ErrBoostActive     = errors.New("production boost is still active")
	ErrInvalidBoost    = errors.New("invalid boost multiplier")
	ErrInvalidDuration = errors.New("invalid boost duration")
)

// 默认加成倍率，用于旧配置缺少倍率表时兜底。
var defaultBoostMultipliers = map[int]int{2: 1, 4: 3, 8: 8, 16: 20}

// 默认持续时间，用于旧配置缺少时长表时兜底。
var defaultBoostDurationFactors = map[int]int{1: 1, 6: 5, 12: 9, 24: 16}

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
	} else if v, ok := defaultBoostMultipliers[multiplier]; ok {
		mf = v
	}

	df := 1
	if cfg.BoostDurationFactor != nil {
		if v, ok := cfg.BoostDurationFactor[hours]; ok {
			df = v
		}
	} else if v, ok := defaultBoostDurationFactors[hours]; ok {
		df = v
	}

	return baseCost * mf * df
}

// PurchaseBoost 购买产量加成（消耗城金）
func (s *Service) PurchaseBoost(playerID string, multiplier int, hours int) (GameState, error) {
	playerID = strings.TrimSpace(playerID)
	if playerID == "" {
		return GameState{}, ErrPlayerNotFound
	}
	if !validBoostMultiplier(multiplier) {
		return GameState{}, ErrInvalidBoost
	}
	if !validBoostHours(hours) {
		return GameState{}, ErrInvalidDuration
	}

	now := time.Now()
	cost := boostCost(multiplier, hours)
	state, err := s.repo.UpdateResourceState(playerID, now, func(state *GameState) error {
		nextState, _ := settleResources(*state, now)
		*state = nextState

		if int(state.CityGold) < cost {
			return ErrInsufficientCityGold
		}
		state.CityGold -= FlexInt(cost)

		// 支持重复购买叠加：倍率累乘，时长接在当前未过期时间之后。
		state.ProductionBoost = stackBoostMultiplier(state.ProductionBoost, state.ProductionBoostEnd, multiplier, now)
		state.ProductionBoostEnd = extendBoostEnd(state.ProductionBoostEnd, hours, now)
		state.ServerTime = now.UTC().Format(resourceDateLayout)

		// 通过 Modifier 管线重新计算产量（含加成）
		modSources := CollectModifierSources(state)
		production := calculateResourceProduction(state.Buildings, state.General)
		production = applyProductionModifiers(production, now, modSources)
		state.ResourceProduction = production
		state.ResourceSettledAt = now.UTC().Format(resourceDateLayout)
		return nil
	})
	if err != nil {
		return GameState{}, err
	}
	s.recordLedger(GoldLedgerEntry{
		PlayerID:     playerID,
		Currency:     LedgerCurrencyCityGold,
		Direction:    LedgerDirectionDebit,
		Amount:       cost,
		BalanceAfter: int(state.CityGold),
		RefType:      LedgerRefBoostPurchase,
		Reason:       "production_boost",
	})
	s.publishCurrencyChanged(playerID, "", "", LedgerRefBoostPurchase)

	return state, nil
}

// GetBoostCost 查询加成价格（供前端展示）
func GetBoostCost(multiplier int, hours int) int {
	return boostCost(multiplier, hours)
}

// GetBoostMultipliers 返回当前配置开放的购买加成倍率。
func GetBoostMultipliers() []int {
	cfg := currentBalance()
	if len(cfg.BoostMultiplierFactor) > 0 {
		return sortedIntKeys(cfg.BoostMultiplierFactor)
	}
	return sortedIntKeys(defaultBoostMultipliers)
}

// GetBoostHours 返回当前配置开放的购买加成时长。
func GetBoostHours() []int {
	cfg := currentBalance()
	if len(cfg.BoostDurationFactor) > 0 {
		return sortedIntKeys(cfg.BoostDurationFactor)
	}
	return sortedIntKeys(defaultBoostDurationFactors)
}

// PurchaseCapacityBoost 购买仓库容量加成（消耗城金，价格同产量加成）
func (s *Service) PurchaseCapacityBoost(playerID string, multiplier int, hours int) (GameState, error) {
	playerID = strings.TrimSpace(playerID)
	if playerID == "" {
		return GameState{}, ErrPlayerNotFound
	}
	if !validBoostMultiplier(multiplier) {
		return GameState{}, ErrInvalidBoost
	}
	if !validBoostHours(hours) {
		return GameState{}, ErrInvalidDuration
	}

	now := time.Now()
	cost := boostCost(multiplier, hours)
	state, err := s.repo.UpdateResourceState(playerID, now, func(state *GameState) error {
		nextState, _ := settleResources(*state, now)
		*state = nextState

		if int(state.CityGold) < cost {
			return ErrInsufficientCityGold
		}
		state.CityGold -= FlexInt(cost)

		// 支持重复购买叠加：倍率累乘，时长接在当前未过期时间之后。
		state.CapacityBoost = stackBoostMultiplier(state.CapacityBoost, state.CapacityBoostEnd, multiplier, now)
		state.CapacityBoostEnd = extendBoostEnd(state.CapacityBoostEnd, hours, now)
		state.ServerTime = now.UTC().Format(resourceDateLayout)

		// 通过 Modifier 管线重新计算容量（含加成）
		modSources := CollectModifierSources(state)
		capacity := calculateResourceCapacity(state.Buildings)
		capacity = applyCapacityModifiers(capacity, now, modSources)
		if err := replaceResourceCapacity(state, capacity); err != nil {
			return err
		}
		state.ResourceSettledAt = now.UTC().Format(resourceDateLayout)
		return nil
	})
	if err != nil {
		return GameState{}, err
	}
	s.recordLedger(GoldLedgerEntry{
		PlayerID:     playerID,
		Currency:     LedgerCurrencyCityGold,
		Direction:    LedgerDirectionDebit,
		Amount:       cost,
		BalanceAfter: int(state.CityGold),
		RefType:      LedgerRefBoostPurchase,
		Reason:       "capacity_boost",
	})
	s.publishCurrencyChanged(playerID, "", "", LedgerRefBoostPurchase)

	return state, nil
}

// validBoostMultiplier 判断倍率是否在当前配置中开放。
func validBoostMultiplier(multiplier int) bool {
	cfg := currentBalance()
	if len(cfg.BoostMultiplierFactor) > 0 {
		_, ok := cfg.BoostMultiplierFactor[multiplier]
		return ok
	}
	_, ok := defaultBoostMultipliers[multiplier]
	return ok
}

// validBoostHours 判断购买时长是否在当前配置中开放。
func validBoostHours(hours int) bool {
	cfg := currentBalance()
	if len(cfg.BoostDurationFactor) > 0 {
		_, ok := cfg.BoostDurationFactor[hours]
		return ok
	}
	_, ok := defaultBoostDurationFactors[hours]
	return ok
}

// stackBoostMultiplier 叠加当前仍生效的购买加成倍率。
func stackBoostMultiplier(current int, currentEnd string, added int, now time.Time) int {
	if current <= 1 || currentEnd == "" {
		return added
	}
	expiresAt, err := time.Parse(resourceDateLayout, currentEnd)
	if err != nil || !now.Before(expiresAt) {
		return added
	}
	return current * added
}

// extendBoostEnd 把新增时长接到当前未过期结束时间之后。
func extendBoostEnd(currentEnd string, addedHours int, now time.Time) string {
	base := now
	if currentEnd != "" {
		if expiresAt, err := time.Parse(resourceDateLayout, currentEnd); err == nil && now.Before(expiresAt) {
			base = expiresAt
		}
	}
	return base.Add(time.Duration(addedHours) * time.Hour).UTC().Format(resourceDateLayout)
}

// sortedIntKeys 返回按升序排列的 int map key。
func sortedIntKeys(values map[int]int) []int {
	keys := make([]int, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Ints(keys)
	return keys
}
