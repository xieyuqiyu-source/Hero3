// 本文件归口玩家资源结算、产量计算和资源类将领特性。
package game

import (
	"math"
	"reflect"
	"strings"
	"time"
)

func (s *Service) AdjustResources(playerID string, adjustments map[string]int) (GameState, error) {
	playerID = strings.TrimSpace(playerID)
	if playerID == "" {
		return GameState{}, ErrPlayerNotFound
	}

	now := time.Now()
	var before, after coreAssetSnapshot
	state, err := s.repo.UpdateResourceState(playerID, now, func(state *GameState) error {
		nextState, _ := settleResources(*state, now)
		*state = nextState
		before = snapshotCoreAssets(state)

		for resType, delta := range adjustments {
			if _, err := adjustResourceCapped(state, resType, delta); err != nil {
				return err
			}
		}

		state.ResourceSettledAt = now.UTC().Format(resourceDateLayout)
		state.ServerTime = now.UTC().Format(resourceDateLayout)
		after = snapshotCoreAssets(state)
		return nil
	})
	if err != nil {
		return GameState{}, err
	}
	s.publishCoreAssetDiff(playerID, "resource_adjust", "", before, after, now)

	return state, nil
}

func (s *Service) FillResources(playerID string) (GameState, error) {
	playerID = strings.TrimSpace(playerID)
	if playerID == "" {
		return GameState{}, ErrPlayerNotFound
	}

	now := time.Now()
	var before, after coreAssetSnapshot
	state, err := s.repo.UpdateResourceState(playerID, now, func(state *GameState) error {
		nextState, _ := settleResources(*state, now)
		*state = nextState
		before = snapshotCoreAssets(state)

		if err := fillResourcesToCapacity(state); err != nil {
			return err
		}

		state.ResourceSettledAt = now.UTC().Format(resourceDateLayout)
		state.ServerTime = now.UTC().Format(resourceDateLayout)
		after = snapshotCoreAssets(state)
		return nil
	})
	if err != nil {
		return GameState{}, err
	}
	s.publishCoreAssetDiff(playerID, "resource_fill", "", before, after, now)

	return state, nil
}

// FillResourcesPaid 一键爆仓（消耗城金，3000 资源 = 1 城金）
func (s *Service) FillResourcesPaid(playerID string) (GameState, int, error) {
	playerID = strings.TrimSpace(playerID)
	if playerID == "" {
		return GameState{}, 0, ErrPlayerNotFound
	}

	now := time.Now()
	cost := 0
	var before, after coreAssetSnapshot
	state, err := s.repo.UpdateResourceState(playerID, now, func(state *GameState) error {
		nextState, _ := settleResources(*state, now)
		*state = nextState
		before = snapshotCoreAssets(state)

		totalNeeded := 0
		for resType, cap := range state.Resources.Capacity {
			current := state.Resources.Items[resType]
			if current < cap {
				totalNeeded += cap - current
			}
		}

		if totalNeeded == 0 {
			state.ServerTime = now.UTC().Format(resourceDateLayout)
			after = before
			return nil
		}

		cost = (totalNeeded + 2999) / 3000
		if cost < 1 {
			cost = 1
		}
		if int(state.CityGold) < cost {
			return ErrInsufficientCityGold
		}
		state.CityGold -= FlexInt(cost)

		if err := fillResourcesToCapacity(state); err != nil {
			return err
		}

		state.ResourceSettledAt = now.UTC().Format(resourceDateLayout)
		state.ServerTime = now.UTC().Format(resourceDateLayout)
		after = snapshotCoreAssets(state)
		return nil
	})
	if err != nil {
		return GameState{}, 0, err
	}
	if cost > 0 {
		s.recordLedger(GoldLedgerEntry{
			PlayerID:     playerID,
			Currency:     LedgerCurrencyCityGold,
			Direction:    LedgerDirectionDebit,
			Amount:       cost,
			BalanceAfter: int(state.CityGold),
			RefType:      "resource_fill",
		})
		s.publishCurrencyChanged(playerID, "", "", "resource_fill")
	}
	s.publishCoreAssetDiff(playerID, "resource_fill", "", before, after, now)

	return state, cost, nil
}

// settleResources 结算资源产出、建筑升级完成、征兵队列完成
func settleResources(state GameState, now time.Time) (GameState, bool) {
	// 结算时间与 RFC3339 持久化精度保持一致，防止同一秒重复读取时反复结算小数秒。
	now = now.UTC().Truncate(time.Second)
	changed := false

	if ApplyConstructionBureauResourceSlots(&state, now) {
		changed = true
	}

	if state.Resources.Items == nil || state.Resources.Capacity == nil {
		_ = ensureResourceState(&state)
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
		endsAt      time.Time
		buildingIdx int // >=0 表示建筑升级完成事件，-1 表示非建筑事件
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
			state.Buildings[i].Status = BuildingStatusNormal
			ApplyConstructionBureauResourceSlots(&state, now)
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
		if applyGuardPerMinuteTraits(&state, now.Sub(settledAt).Seconds()) {
			changed = true
		}

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
				state.Buildings[event.buildingIdx].Status = BuildingStatusNormal
				ApplyConstructionBureauResourceSlots(&state, event.endsAt)
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
			if err := replaceResourceItems(&state, resources); err != nil {
				return state, changed
			}
			state.ResourceSettledAt = now.Format(resourceDateLayout)
			changed = true
		} else if state.ResourceSettledAt != now.Format(resourceDateLayout) {
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
		if err := replaceResourceCapacity(&state, capacity); err != nil {
			return state, changed
		}
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
				AddArmyUnit(&state, queue.UnitType, queue.Amount)
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

// settleResourcesBeforeGeneralRelease 在移除离城占用前完成结算，防止归城后追溯获得离城期间的城内加成。
func settleResourcesBeforeGeneralRelease(state *GameState, now time.Time) {
	if state == nil {
		return
	}
	next, _ := settleResources(*state, now)
	*state = next
}

// refreshResourcesAfterGeneralRelease 零时长刷新归城后的当前产量，不追补已经按离城状态结算的资源。
func refreshResourcesAfterGeneralRelease(state *GameState, now time.Time) {
	if state == nil {
		return
	}
	next, _ := settleResources(*state, now)
	*state = next
}

func addProducedResource(current int, perHour int, elapsedSeconds float64, capacity int) int {
	if current >= capacity || perHour <= 0 || elapsedSeconds <= 0 {
		return current
	}

	produced := int(float64(perHour) * elapsedSeconds / 3600)
	if produced <= 0 {
		return current
	}

	return min(current+produced, capacity)
}

// applyGuardPerMinuteTraits 按资源结算间隔应用将领产兵类特性。
func applyGuardPerMinuteTraits(state *GameState, elapsedSeconds float64) bool {
	if state == nil || state.General == nil || elapsedSeconds <= 0 || !generalAvailableAtHome(state.GeneralAssignments, state.General.ID) {
		return false
	}
	state.GeneralTraitProgress = cloneGeneralTraitProgress(state.GeneralTraitProgress)
	generalCopy := cloneGeneral(*state.General)
	applyHeroConfigToGeneral(&generalCopy)
	changed := false
	for _, trait := range generalCopy.Traits {
		perMinute := trait.Params["guardPerMinute"]
		if perMinute <= 0 {
			continue
		}
		unitType := strings.TrimSpace(trait.TargetUnitType)
		if unitType == "" {
			unitType = firstCombatUnitByCategory(state.Player.Faction, "special")
		}
		if unitType == "" {
			continue
		}
		progressKey := guardProductionProgressKey(generalCopy.ID, trait.TraitID, unitType)
		amount, remainder := calculateGuardProduction(state.GeneralTraitProgress[progressKey], perMinute, elapsedSeconds, guardProductionLimit(trait.Params))
		if updateGeneralTraitProgress(state.GeneralTraitProgress, progressKey, remainder) {
			changed = true
		}
		if amount <= 0 {
			continue
		}
		AddArmyUnit(state, unitType, amount)
		changed = true
	}
	return changed
}

// calculateGuardProduction 按累计小数进度计算本次整数产兵量，并在触及单次上限时丢弃超额进度。
func calculateGuardProduction(progress float64, perMinute float64, elapsedSeconds float64, maxPerSettle int) (int, float64) {
	total := progress + perMinute*elapsedSeconds/60
	if total <= 0 {
		return 0, 0
	}
	if maxPerSettle > 0 && total >= float64(maxPerSettle) {
		return maxPerSettle, 0
	}
	amount := int(math.Floor(total + 1e-9))
	remainder := total - float64(amount)
	if remainder < 1e-9 {
		remainder = 0
	}
	return amount, remainder
}

// guardProductionLimit 读取单次产兵上限，并兼容尚未经过配置归一化的旧参数。
func guardProductionLimit(params map[string]float64) int {
	if value := int(params["maxGuardPerSettle"]); value > 0 {
		return value
	}
	return int(params["maxGuardPerDay"])
}

// guardProductionProgressKey 为不同将领、特性和目标兵种隔离累计进度。
func guardProductionProgressKey(generalID string, traitID string, unitType string) string {
	return strings.Join([]string{generalID, traitID, unitType}, ":")
}

// cloneGeneralTraitProgress 复制特性进度，避免值语义 GameState 仍共享原始 map。
func cloneGeneralTraitProgress(progress map[string]float64) map[string]float64 {
	cloned := make(map[string]float64, len(progress))
	for key, value := range progress {
		cloned[key] = value
	}
	return cloned
}

// updateGeneralTraitProgress 保存有效小数进度，整数边界时移除无意义的零值。
func updateGeneralTraitProgress(progress map[string]float64, key string, value float64) bool {
	previous, existed := progress[key]
	if value <= 0 {
		if existed {
			delete(progress, key)
			return true
		}
		return false
	}
	if existed && math.Abs(previous-value) < 1e-9 {
		return false
	}
	progress[key] = value
	return true
}

// firstCombatUnitByCategory 返回指定阵营中第一个可战斗兵种。
func firstCombatUnitByCategory(faction string, category string) string {
	unitsMu.RLock()
	defer unitsMu.RUnlock()
	for unitType, cfg := range activeUnits[faction] {
		if cfg.Category == category && !isNonCombatUnit(cfg) {
			return unitType
		}
	}
	return ""
}

// --- Modifier 管线辅助函数 ---

// applySpeedBonus 通过 Modifier 管线计算速度加成后的实际时间。
// 速度类 modifier 表示“速度倍率加成”：flat +11 => 12 倍速，percentAdd +0.2 => 1.2 倍速。
func applySpeedBonus(baseSeconds int, key string, now time.Time, sources []ModifierSource) int {
	speedFactor := computeSpeedFactor(key, now, sources)
	if speedFactor <= 0 {
		return baseSeconds
	}
	result := float64(baseSeconds) / speedFactor
	if result < 1 {
		return 1 // 最少 1 秒
	}
	return int(result)
}

func computeSpeedFactor(key string, now time.Time, sources []ModifierSource) float64 {
	var additive float64
	multiplier := 1.0

	for _, src := range sources {
		if src == nil {
			continue
		}
		for _, mod := range src.Modifiers(now) {
			if mod.Key != key {
				continue
			}
			switch mod.Mode {
			case "flat", "percentAdd":
				additive += mod.Value
			case "percentMultiply":
				multiplier *= 1 + mod.Value
			}
		}
	}

	return (1 + additive) * multiplier
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

func copyResourceMap(source map[string]int) map[string]int {
	next := make(map[string]int, len(source))
	for key, value := range source {
		next[key] = value
	}
	return next
}

func coreResourceTypes() []string {
	defs := ListResourceTypeDefinitions()
	types := make([]string, 0, len(defs))
	for _, def := range defs {
		types = append(types, def.Type)
	}
	return types
}

func isCoreResourceType(resourceType string) bool {
	return IsResourceTypeRegistered(resourceType)
}
