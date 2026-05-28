package game

import (
	"reflect"
	"strings"
	"time"
)

func (s *Service) AdjustResources(playerID string, adjustments map[string]int) (GameState, error) {
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

	for resType, delta := range adjustments {
		current := state.Resources.Items[resType]
		next := current + delta
		if next < 0 {
			next = 0
		}
		cap := state.Resources.Capacity[resType]
		if cap > 0 && next > cap {
			next = cap
		}
		state.Resources.Items[resType] = next
	}

	state.ResourceSettledAt = now.UTC().Format(resourceDateLayout)
	state.ServerTime = now.UTC().Format(resourceDateLayout)

	if err := s.repo.SaveState(state, now); err != nil {
		return GameState{}, err
	}

	return state, nil
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

// FillResourcesPaid 一键爆仓（消耗城金，3000 资源 = 1 城金）
func (s *Service) FillResourcesPaid(playerID string) (GameState, int, error) {
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

	// 计算需要补充的总资源量
	totalNeeded := 0
	for resType, cap := range state.Resources.Capacity {
		current := state.Resources.Items[resType]
		if current < cap {
			totalNeeded += cap - current
		}
	}

	if totalNeeded == 0 {
		return state, 0, nil
	}

	// 计算城金花费（3000 资源 = 1 城金，向上取整）
	cost := (totalNeeded + 2999) / 3000
	if cost < 1 {
		cost = 1
	}

	// 扣城金
	if _, err := s.repo.DeductCityGold(playerID, cost); err != nil {
		return GameState{}, 0, err
	}

	// 重新读取状态（城金已扣）
	state, _ = s.repo.GetState(playerID)
	state, _ = settleResources(state, now)

	// 填满资源
	for resType, cap := range state.Resources.Capacity {
		state.Resources.Items[resType] = cap
	}

	state.ResourceSettledAt = now.UTC().Format(resourceDateLayout)
	state.ServerTime = now.UTC().Format(resourceDateLayout)

	if err := s.repo.SaveState(state, now); err != nil {
		return GameState{}, 0, err
	}

	return state, cost, nil
}

// settleResources 结算资源产出、建筑升级完成、征兵队列完成
func settleResources(state GameState, now time.Time) (GameState, bool) {
	now = now.UTC()
	changed := false

	if state.Resources.Items == nil {
		state.Resources.Items = map[string]int{}
		changed = true
	}

	// 解析上次结算时间
	settledAt := now
	if strings.TrimSpace(state.ResourceSettledAt) != "" {
		if parsed, err := time.Parse(resourceDateLayout, state.ResourceSettledAt); err == nil {
			settledAt = parsed.UTC()
		}
	} else {
		state.ResourceSettledAt = now.Format(resourceDateLayout)
		changed = true
	}

	// 收集 settledAt 到 now 之间所有升级完成的时间点，作为切片边界
	type upgradeEvent struct {
		index  int
		endsAt time.Time
	}
	var events []upgradeEvent
	for i, b := range state.Buildings {
		if b.UpgradeEndsAt == nil {
			continue
		}
		endsAt, err := time.Parse(resourceDateLayout, *b.UpgradeEndsAt)
		if err != nil {
			continue
		}
		if (now.After(endsAt) || now.Equal(endsAt)) && endsAt.After(settledAt) {
			events = append(events, upgradeEvent{index: i, endsAt: endsAt.UTC()})
		} else if now.After(endsAt) || now.Equal(endsAt) {
			// 升级在 settledAt 之前就完成了，直接完成
			state.Buildings[i].Level++
			state.Buildings[i].UpgradeEndsAt = nil
			changed = true
		}
	}

	// 按完成时间排序
	for i := 0; i < len(events)-1; i++ {
		for j := i + 1; j < len(events); j++ {
			if events[j].endsAt.Before(events[i].endsAt) {
				events[i], events[j] = events[j], events[i]
			}
		}
	}

	// 时间切片结算：settledAt → event1 → event2 → ... → now
	if now.After(settledAt) {
		sliceStart := settledAt
		resources := copyResourceMap(state.Resources.Items)
		capacity := calculateResourceCapacity(state.Buildings)

		// 通过 Modifier 管线计算容量加成
		modSources := CollectModifierSources(&state)
		capacity = applyCapacityModifiers(capacity, now, modSources)

		for _, event := range events {
			// 用当前建筑等级（升级前）算这段时间的产出（含所有加成）
			production := calculateResourceProduction(state.Buildings, state.General)
			production = applyProductionModifiers(production, now, modSources)
			elapsed := event.endsAt.Sub(sliceStart).Seconds()
			if elapsed > 0 {
				for resType, perHour := range production {
					resources[resType] = addProducedResource(
						resources[resType], perHour, elapsed, capacity[resType],
					)
				}
			}

			// 完成升级
			state.Buildings[event.index].Level++
			state.Buildings[event.index].UpgradeEndsAt = nil
			changed = true

			// 升级后容量可能变化
			capacity = calculateResourceCapacity(state.Buildings)
			capacity = applyCapacityModifiers(capacity, now, modSources)
			sliceStart = event.endsAt
		}

		// 最后一段：从最后一个升级完成时间到 now
		production := calculateResourceProduction(state.Buildings, state.General)
		production = applyProductionModifiers(production, now, modSources)
		elapsed := now.Sub(sliceStart).Seconds()
		if elapsed > 0 {
			for resType, perHour := range production {
				resources[resType] = addProducedResource(
					resources[resType], perHour, elapsed, capacity[resType],
				)
			}
		}

		if !reflect.DeepEqual(resources, state.Resources.Items) || len(events) > 0 {
			state.Resources.Items = resources
			state.ResourceSettledAt = now.Format(resourceDateLayout)
			changed = true
		}
	}

	// 清理过期加成
	cleanExpiredBoosts(&state, now)

	// 通过 Modifier 管线更新产量和容量（反映最终建筑等级 + 所有加成）
	modSources := CollectModifierSources(&state)
	production := calculateResourceProduction(state.Buildings, state.General)
	production = applyProductionModifiers(production, now, modSources)
	if !reflect.DeepEqual(state.ResourceProduction, production) {
		state.ResourceProduction = production
		changed = true
	}
	capacity := calculateResourceCapacity(state.Buildings)
	capacity = applyCapacityModifiers(capacity, now, modSources)
	if !reflect.DeepEqual(state.Resources.Capacity, capacity) {
		state.Resources.Capacity = capacity
		changed = true
	}

	// 检查并完成已到期的征兵队列
	if len(state.RecruitQueues) > 0 {
		remaining := state.RecruitQueues[:0]
		for _, queue := range state.RecruitQueues {
			endsAt, err := time.Parse(resourceDateLayout, queue.EndsAt)
			if err != nil {
				remaining = append(remaining, queue)
				continue
			}
			if now.After(endsAt) || now.Equal(endsAt) {
				// 征兵完成，加入军队
				found := false
				for i, unit := range state.Army {
					if unit.UnitType == queue.UnitType {
						state.Army[i].Amount += queue.Amount
						found = true
						break
					}
				}
				if !found {
					state.Army = append(state.Army, ArmyUnit{
						UnitType: queue.UnitType,
						Amount:   queue.Amount,
					})
				}
				changed = true
			} else {
				remaining = append(remaining, queue)
			}
		}
		if len(remaining) != len(state.RecruitQueues) {
			state.RecruitQueues = remaining
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

func calculateResourceProduction(buildings []Building, general *General) ResourceProduction {
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

	// 注意：将领加成不在这里应用，统一由 applyProductionModifiers 通过 Modifier 管线处理
	return production
}

// --- Modifier 管线辅助函数 ---

// applyProductionModifiers 通过 Modifier 管线对产量应用所有加成
func applyProductionModifiers(production ResourceProduction, now time.Time, sources []ModifierSource) ResourceProduction {
	result := ResourceProduction{}
	for resType, value := range production {
		// 先检查资源专属加成（如 "woodProductionBonus"）
		specific := ComputeIntAttributeAt(value, resType+"ProductionBonus", now, sources...)
		// 再应用通用产量加成（"productionBonus"）
		final := ComputeIntAttributeAt(specific, "productionBonus", now, sources...)
		result[resType] = final
	}
	return result
}

// applyCapacityModifiers 通过 Modifier 管线对容量应用所有加成
func applyCapacityModifiers(capacity map[string]int, now time.Time, sources []ModifierSource) map[string]int {
	result := make(map[string]int, len(capacity))
	for resType, value := range capacity {
		result[resType] = ComputeIntAttributeAt(value, "capacityBonus", now, sources...)
	}
	return result
}

// cleanExpiredBoosts 清理过期的加成字段
func cleanExpiredBoosts(state *GameState, now time.Time) {
	if state.ProductionBoost > 1 && state.ProductionBoostEnd != "" {
		if expiresAt, err := time.Parse(resourceDateLayout, state.ProductionBoostEnd); err == nil && now.After(expiresAt) {
			state.ProductionBoost = 0
			state.ProductionBoostEnd = ""
		}
	}
	if state.CapacityBoost > 1 && state.CapacityBoostEnd != "" {
		if expiresAt, err := time.Parse(resourceDateLayout, state.CapacityBoostEnd); err == nil && now.After(expiresAt) {
			state.CapacityBoost = 0
			state.CapacityBoostEnd = ""
		}
	}
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
