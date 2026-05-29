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

	// 收集 settledAt 到 now 之间所有状态变化时间点，作为切片边界
	// 包括：建筑升级完成、加成到期等
	type timeSliceEvent struct {
		endsAt       time.Time
		buildingIdx  int  // >=0 表示建筑升级完成事件，-1 表示非建筑事件
	}
	var sliceEvents []timeSliceEvent

	// 建筑升级完成事件
	for i, b := range state.Buildings {
		if b.UpgradeEndsAt == nil {
			continue
		}
		endsAt, err := time.Parse(resourceDateLayout, *b.UpgradeEndsAt)
		if err != nil {
			continue
		}
		if (now.After(endsAt) || now.Equal(endsAt)) && endsAt.After(settledAt) {
			sliceEvents = append(sliceEvents, timeSliceEvent{endsAt: endsAt.UTC(), buildingIdx: i})
		} else if now.After(endsAt) || now.Equal(endsAt) {
			// 升级在 settledAt 之前就完成了，直接完成
			state.Buildings[i].Level++
			state.Buildings[i].UpgradeEndsAt = nil
			changed = true
		}
	}

	// 加成到期事件（从所有 ModifierSource 自动收集到期时间作为切片边界）
	allModSources := CollectModifierSources(&state)
	for _, src := range allModSources {
		for _, t := range src.ExpiresAt() {
			if t.After(settledAt) && (now.After(t) || now.Equal(t)) {
				sliceEvents = append(sliceEvents, timeSliceEvent{endsAt: t.UTC(), buildingIdx: -1})
			}
		}
	}

	// 按时间排序
	for i := 0; i < len(sliceEvents)-1; i++ {
		for j := i + 1; j < len(sliceEvents); j++ {
			if sliceEvents[j].endsAt.Before(sliceEvents[i].endsAt) {
				sliceEvents[i], sliceEvents[j] = sliceEvents[j], sliceEvents[i]
			}
		}
	}

	// 时间切片结算：settledAt → event1 → event2 → ... → now
	// 每个切片用 sliceStart 时间点判断加成是否生效，确保离线期间加成中途过期时正确结算
	if now.After(settledAt) {
		sliceStart := settledAt
		resources := copyResourceMap(state.Resources.Items)
		capacity := calculateResourceCapacity(state.Buildings)

		// 用 sliceStart 判断加成是否生效
		modSources := CollectModifierSources(&state)
		capacity = applyCapacityModifiers(capacity, sliceStart, modSources)

		for _, event := range sliceEvents {
			// 用 sliceStart 时间点计算该切片的产出（加成是否生效取决于 sliceStart）
			production := calculateResourceProduction(state.Buildings, state.General)
			production = applyProductionModifiers(production, sliceStart, modSources)
			capacity = calculateResourceCapacity(state.Buildings)
			capacity = applyCapacityModifiers(capacity, sliceStart, modSources)

			elapsed := event.endsAt.Sub(sliceStart).Seconds()
			if elapsed > 0 {
				for resType, perHour := range production {
					resources[resType] = addProducedResource(
						resources[resType], perHour, elapsed, capacity[resType],
					)
				}
			}

			// 如果是建筑升级完成事件，执行升级
			if event.buildingIdx >= 0 {
				state.Buildings[event.buildingIdx].Level++
				state.Buildings[event.buildingIdx].UpgradeEndsAt = nil
				changed = true
			}

			sliceStart = event.endsAt
		}

		// 最后一段：从最后一个事件到 now
		production := calculateResourceProduction(state.Buildings, state.General)
		production = applyProductionModifiers(production, sliceStart, modSources)
		capacity = calculateResourceCapacity(state.Buildings)
		capacity = applyCapacityModifiers(capacity, sliceStart, modSources)

		elapsed := now.Sub(sliceStart).Seconds()
		if elapsed > 0 {
			for resType, perHour := range production {
				resources[resType] = addProducedResource(
					resources[resType], perHour, elapsed, capacity[resType],
				)
			}
		}

		if !reflect.DeepEqual(resources, state.Resources.Items) || len(sliceEvents) > 0 {
			state.Resources.Items = resources
			state.ResourceSettledAt = now.Format(resourceDateLayout)
			changed = true
		}
	}

	// 清理过期加成
	cleanExpiredBoosts(&state, now)
	cleanExpiredBuffs(&state, now)

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

// applySpeedBonus 通过 Modifier 管线计算速度加成后的实际时间
// 速度加成越高时间越短：实际时间 = 原始时间 / (1 + bonus)
func applySpeedBonus(baseSeconds int, key string, now time.Time, sources []ModifierSource) int {
	bonus := ComputeAttributeAt(0, key, now, sources...) // 只取加成部分（base=0）
	if bonus <= 0 {
		return baseSeconds
	}
	result := float64(baseSeconds) / (1 + bonus)
	if result < 1 {
		return 1 // 最少 1 秒
	}
	return int(result)
}

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
