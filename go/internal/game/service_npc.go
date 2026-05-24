package game

import (
	"math/rand"
	"strings"
	"time"
)

// SetNpcConfigPath 加载 NPC 配置
func (s *Service) SetNpcConfigPath(path string) error {
	s.npcConfigPath = path
	return LoadNpcConfig(path)
}

// GetNpcCities 获取玩家的 NPC 城池列表（自动检查是否需要刷新）
func (s *Service) GetNpcCities(playerID string) (NpcState, error) {
	playerID = strings.TrimSpace(playerID)
	if playerID == "" {
		return NpcState{}, ErrPlayerNotFound
	}

	state, err := s.repo.GetState(playerID)
	if err != nil {
		return NpcState{}, err
	}

	now := time.Now()
	npcState := state.NpcState
	changed := false

	// 检查是否需要自动刷新
	if npcState == nil || needsNpcRefresh(npcState, now) {
		npcState = generateNpcState(now)
		state.NpcState = npcState
		changed = true
	} else {
		// 惰性结算 NPC 资源和守军
		settled := settleNpcCities(npcState, now)
		if settled {
			changed = true
		}
	}

	if changed {
		if err := s.repo.SaveState(state, now); err != nil {
			return NpcState{}, err
		}
	}

	return *npcState, nil
}

// RefreshNpcCities 手动刷新 NPC 城池（消耗金币）
// TODO: 接入金币系统后，在此处检查并扣除 manualRefreshCostGold
func (s *Service) RefreshNpcCities(playerID string) (NpcState, error) {
	playerID = strings.TrimSpace(playerID)
	if playerID == "" {
		return NpcState{}, ErrPlayerNotFound
	}

	state, err := s.repo.GetState(playerID)
	if err != nil {
		return NpcState{}, err
	}

	now := time.Now()
	npcState := generateNpcState(now)
	state.NpcState = npcState

	if err := s.repo.SaveState(state, now); err != nil {
		return NpcState{}, err
	}

	return *npcState, nil
}

// --- 内部逻辑 ---

func needsNpcRefresh(npcState *NpcState, now time.Time) bool {
	if npcState == nil || len(npcState.Cities) == 0 {
		return true
	}
	if npcState.LastRefreshedAt == "" {
		return true
	}

	cfg := GetNpcConfig()
	lastRefreshed, err := time.Parse(resourceDateLayout, npcState.LastRefreshedAt)
	if err != nil {
		return true
	}

	return now.Sub(lastRefreshed).Hours() >= float64(cfg.RefreshIntervalHours)
}

func generateNpcState(now time.Time) *NpcState {
	cfg := GetNpcConfig()
	cities := generateNpcCities(cfg, now)
	return &NpcState{
		Cities:          cities,
		LastRefreshedAt: now.UTC().Format(resourceDateLayout),
	}
}

func generateNpcCities(cfg NpcConfig, now time.Time) []NpcCity {
	// 分配各等级数量
	tierCounts := allocateTierCounts(cfg)

	// 随机地名（不重复）
	names := pickRandomNames(cfg.CityNames, cfg.TotalCities)

	// 生成城池
	cities := make([]NpcCity, 0, cfg.TotalCities)
	nameIdx := 0

	tierOrder := []string{"small", "medium", "large", "golden"}
	for _, tier := range tierOrder {
		count := tierCounts[tier]
		tierCfg := cfg.Tiers[tier]

		for i := 0; i < count; i++ {
			name := "未知城池"
			if nameIdx < len(names) {
				name = names[nameIdx]
				nameIdx++
			}

			city := generateSingleNpcCity(cfg, tierCfg, tier, name, now)
			cities = append(cities, city)
		}
	}

	return cities
}

func allocateTierCounts(cfg NpcConfig) map[string]int {
	counts := map[string]int{}
	total := cfg.TotalCities

	// 先放保底
	guaranteed := 0
	for tier, tierCfg := range cfg.Tiers {
		counts[tier] = tierCfg.Count.Guaranteed
		guaranteed += tierCfg.Count.Guaranteed
	}

	remaining := total - guaranteed

	// 金色单独判定
	hasGolden := rand.Float64() < cfg.GoldenAppearRate
	if hasGolden && remaining > 0 {
		counts["golden"]++
		remaining--
	}

	// 剩余按权重分配（不含金色）
	type weightedTier struct {
		tier   string
		weight int
	}
	var pool []weightedTier
	for _, tier := range []string{"small", "medium", "large"} {
		if w := cfg.Tiers[tier].Count.Weight; w > 0 {
			pool = append(pool, weightedTier{tier, w})
		}
	}

	for remaining > 0 && len(pool) > 0 {
		totalWeight := 0
		for _, p := range pool {
			totalWeight += p.weight
		}
		if totalWeight <= 0 {
			// 无法分配，全给小型兜底
			counts["small"] += remaining
			break
		}
		roll := rand.Intn(totalWeight)
		cumulative := 0
		for _, p := range pool {
			cumulative += p.weight
			if roll < cumulative {
				counts[p.tier]++
				remaining--
				break
			}
		}
	}

	return counts
}

func generateSingleNpcCity(cfg NpcConfig, tierCfg NpcTierConfig, tier string, name string, now time.Time) NpcCity {
	// 随机阵营（魏蜀吴等概率）
	factions := []string{"wei", "shu", "wu"}
	faction := factions[rand.Intn(len(factions))]

	// 计算产量和仓库
	multiplier := tierCfg.Multiplier
	production := int(float64(cfg.BaseProduction) * multiplier)
	storage := int(float64(cfg.BaseStorage) * multiplier)

	// 随机恢复特性
	profile := pickRecoveryProfile(cfg.RecoveryProfiles)

	// 生成守军
	army := generateNpcArmy(faction, tierCfg)

	// 生成词条
	traits := pickTraits(cfg.TraitPool, tierCfg.TraitCount)

	// 初始资源 = 仓库满
	resources := map[string]int{
		"wood":  storage,
		"stone": storage,
		"iron":  storage,
		"food":  storage,
	}
	storageMap := map[string]int{
		"wood":  storage,
		"stone": storage,
		"iron":  storage,
		"food":  storage,
	}
	productionMap := map[string]int{
		"wood":  production,
		"stone": production,
		"iron":  production,
		"food":  production,
	}

	// 守军恢复速率 = 总兵力 / 24h × 恢复特性倍率
	totalArmy := 0
	for _, u := range army {
		totalArmy += u.Amount
	}
	armyRecoveryRate := float64(totalArmy) / 24.0 * profile.ArmyMultiplier

	nowStr := now.UTC().Format(resourceDateLayout)

	return NpcCity{
		ID:                "npc_" + randomID(8),
		Name:              name,
		Faction:           faction,
		Tier:              tier,
		Resources:         resources,
		StorageCapacity:   storageMap,
		ProductionPerHour: productionMap,
		Army:              army,
		MaxArmy:           copyArmyUnits(army),
		ArmyRecoveryRate:  armyRecoveryRate,
		RecoveryProfile:   profile.ID,
		Traits:            traits,
		ResourceSettledAt: nowStr,
		ArmySettledAt:     nowStr,
		GeneratedAt:       nowStr,
	}
}

func generateNpcArmy(faction string, tierCfg NpcTierConfig) []ArmyUnit {
	// 获取该阵营的兵种列表
	factionUnits := GetFactionUnits(faction)
	if factionUnits == nil {
		return []ArmyUnit{}
	}

	// 按类别分组
	var infantry, cavalry, siege, special []string
	for unitID, unit := range factionUnits {
		switch unit.Category {
		case "infantry":
			infantry = append(infantry, unitID)
		case "cavalry":
			cavalry = append(cavalry, unitID)
		case "siege":
			siege = append(siege, unitID)
		case "special":
			// 排除商人等非战斗单位
			if unit.Stats["attack"] > 0 {
				special = append(special, unitID)
			}
		}
	}

	// 确定兵种数量
	numTypes := randRange(tierCfg.ArmyTypes.Min, tierCfg.ArmyTypes.Max)
	totalArmy := randRange(tierCfg.ArmyRange.Min, tierCfg.ArmyRange.Max)

	// 选择兵种（优先步兵，逐步加入其他类型）
	var selectedUnits []string
	if len(infantry) > 0 {
		selectedUnits = append(selectedUnits, infantry[rand.Intn(len(infantry))])
	}
	if numTypes >= 2 && len(cavalry) > 0 {
		selectedUnits = append(selectedUnits, cavalry[rand.Intn(len(cavalry))])
	}
	if numTypes >= 3 && len(siege) > 0 {
		selectedUnits = append(selectedUnits, siege[rand.Intn(len(siege))])
	}
	if numTypes >= 4 && len(special) > 0 {
		selectedUnits = append(selectedUnits, special[rand.Intn(len(special))])
	}
	// 如果还需要更多兵种，从所有可用中随机补
	allAvailable := append(append(append(infantry, cavalry...), siege...), special...)
	for len(selectedUnits) < numTypes && len(allAvailable) > 0 {
		pick := allAvailable[rand.Intn(len(allAvailable))]
		// 避免重复
		duplicate := false
		for _, s := range selectedUnits {
			if s == pick {
				duplicate = true
				break
			}
		}
		if !duplicate {
			selectedUnits = append(selectedUnits, pick)
		}
		// 防止死循环
		if len(selectedUnits) >= len(allAvailable) {
			break
		}
	}

	if len(selectedUnits) == 0 {
		return []ArmyUnit{}
	}

	// 分配兵力（按随机权重）
	army := make([]ArmyUnit, 0, len(selectedUnits))
	weights := make([]int, len(selectedUnits))
	totalWeight := 0
	for i := range selectedUnits {
		w := rand.Intn(100) + 20 // 20-119 的随机权重
		weights[i] = w
		totalWeight += w
	}

	assignedTotal := 0
	for i, unitID := range selectedUnits {
		amount := totalArmy * weights[i] / totalWeight
		if amount < 1 {
			amount = 1
		}
		army = append(army, ArmyUnit{UnitType: unitID, Amount: amount})
		assignedTotal += amount
	}

	// 余数补给第一个兵种
	if assignedTotal < totalArmy && len(army) > 0 {
		army[0].Amount += totalArmy - assignedTotal
	}

	return army
}

func pickRecoveryProfile(profiles []NpcRecoveryProfile) NpcRecoveryProfile {
	if len(profiles) == 0 {
		return NpcRecoveryProfile{ID: "normal", Name: "平庸", ArmyMultiplier: 1.0, ResourceMultiplier: 1.0}
	}

	totalWeight := 0
	for _, p := range profiles {
		totalWeight += p.Weight
	}
	if totalWeight <= 0 {
		return profiles[0]
	}

	roll := rand.Intn(totalWeight)
	cumulative := 0
	for _, p := range profiles {
		cumulative += p.Weight
		if roll < cumulative {
			return p
		}
	}

	return profiles[0]
}

func pickTraits(pool []NpcTraitConfig, countRange IntRange) []NpcTrait {
	count := randRange(countRange.Min, countRange.Max)
	if count <= 0 || len(pool) == 0 {
		return []NpcTrait{}
	}

	// 按权重随机选取（不重复）
	traits := make([]NpcTrait, 0, count)
	used := map[string]bool{}

	for len(traits) < count {
		totalWeight := 0
		for _, t := range pool {
			if !used[t.ID] {
				totalWeight += t.Weight
			}
		}
		if totalWeight <= 0 {
			break
		}

		roll := rand.Intn(totalWeight)
		cumulative := 0
		for _, t := range pool {
			if used[t.ID] {
				continue
			}
			cumulative += t.Weight
			if roll < cumulative {
				traits = append(traits, NpcTrait{
					ID:    t.ID,
					Name:  t.Name,
					Buffs: t.Buffs,
				})
				used[t.ID] = true
				break
			}
		}
	}

	return traits
}

func pickRandomNames(pool []string, count int) []string {
	if len(pool) <= count {
		return pool
	}

	// Fisher-Yates shuffle 取前 count 个
	shuffled := make([]string, len(pool))
	copy(shuffled, pool)
	for i := len(shuffled) - 1; i > 0; i-- {
		j := rand.Intn(i + 1)
		shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
	}

	return shuffled[:count]
}

// settleNpcCities 惰性结算所有 NPC 城池的资源和守军
func settleNpcCities(npcState *NpcState, now time.Time) bool {
	if npcState == nil {
		return false
	}

	changed := false
	nowStr := now.UTC().Format(resourceDateLayout)

	for i := range npcState.Cities {
		city := &npcState.Cities[i]
		cityChanged := false

		// 结算资源
		if city.ResourceSettledAt != "" {
			settledAt, err := time.Parse(resourceDateLayout, city.ResourceSettledAt)
			if err == nil {
				elapsed := now.Sub(settledAt).Seconds()
				if elapsed > 0 {
					// 获取恢复特性倍率
					resourceMult := getResourceMultiplier(city.RecoveryProfile)
					// 获取词条产量加成
					traitBuffs := collectTraitBuffs(city)
					for resType, perHour := range city.ProductionPerHour {
						effectivePerHour := int(float64(perHour) * resourceMult)
						effectivePerHour = traitBuffs.applyProductionRate(effectivePerHour)
						produced := int(float64(effectivePerHour) * elapsed / 3600)
						if produced > 0 {
							current := city.Resources[resType]
							cap := city.StorageCapacity[resType]
							next := current + produced
							if next > cap {
								next = cap
							}
							if next != current {
								city.Resources[resType] = next
								cityChanged = true
							}
						}
					}
					if cityChanged {
						city.ResourceSettledAt = nowStr
					}
				}
			}
		}

		// 结算守军恢复
		if city.ArmySettledAt != "" && city.ArmyRecoveryRate > 0 {
			settledAt, err := time.Parse(resourceDateLayout, city.ArmySettledAt)
			if err == nil {
				elapsed := now.Sub(settledAt).Hours()
				if elapsed > 0 {
					// 词条加成恢复速度
					traitBuffs := collectTraitBuffs(city)
					effectiveRate := traitBuffs.applyArmyRecovery(city.ArmyRecoveryRate)
					recoveredTotal := int(effectiveRate * elapsed)
					if recoveredTotal > 0 {
						armyChanged := recoverNpcArmy(city, recoveredTotal, traitBuffs)
						if armyChanged {
							city.ArmySettledAt = nowStr
							cityChanged = true
						}
					}
				}
			}
		}

		if cityChanged {
			changed = true
		}
	}

	return changed
}

// recoverNpcArmy 按比例恢复守军到上限（词条可提升上限）
func recoverNpcArmy(city *NpcCity, totalRecovery int, traitBuffs NpcTraitBuffs) bool {
	if totalRecovery <= 0 {
		return false
	}

	// 计算当前缺口（词条加成后的上限）
	type deficit struct {
		index   int
		missing int
		cap     int
	}
	var deficits []deficit
	totalMissing := 0

	for i, maxUnit := range city.MaxArmy {
		current := 0
		for _, u := range city.Army {
			if u.UnitType == maxUnit.UnitType {
				current = u.Amount
				break
			}
		}
		effectiveCap := traitBuffs.applyArmyCap(maxUnit.Amount)
		missing := effectiveCap - current
		if missing > 0 {
			deficits = append(deficits, deficit{i, missing, effectiveCap})
			totalMissing += missing
		}
	}

	if totalMissing == 0 {
		return false
	}

	// 按缺口比例分配恢复量
	recovered := min(totalRecovery, totalMissing)
	changed := false

	for _, d := range deficits {
		share := recovered * d.missing / totalMissing
		if share <= 0 {
			continue
		}

		maxUnit := city.MaxArmy[d.index]
		found := false
		for j := range city.Army {
			if city.Army[j].UnitType == maxUnit.UnitType {
				newAmount := city.Army[j].Amount + share
				if newAmount > d.cap {
					newAmount = d.cap
				}
				if newAmount != city.Army[j].Amount {
					city.Army[j].Amount = newAmount
					changed = true
				}
				found = true
				break
			}
		}
		if !found {
			// 兵种被打空了，重新加入
			city.Army = append(city.Army, ArmyUnit{
				UnitType: maxUnit.UnitType,
				Amount:   min(share, d.cap),
			})
			changed = true
		}
	}

	return changed
}

func getResourceMultiplier(profileID string) float64 {
	cfg := GetNpcConfig()
	for _, p := range cfg.RecoveryProfiles {
		if p.ID == profileID {
			return p.ResourceMultiplier
		}
	}
	return 1.0
}

func copyArmyUnits(army []ArmyUnit) []ArmyUnit {
	result := make([]ArmyUnit, len(army))
	copy(result, army)
	return result
}

func randRange(min, max int) int {
	if min >= max {
		return min
	}
	return min + rand.Intn(max-min+1)
}
