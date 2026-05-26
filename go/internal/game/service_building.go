package game

import (
	"strings"
	"time"
)

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

// InstantCompleteBuilding 极速完成建筑升级（消耗城金）
func (s *Service) InstantCompleteBuilding(playerID string, buildingID string) (GameState, error) {
	playerID = strings.TrimSpace(playerID)
	buildingID = strings.TrimSpace(buildingID)
	if playerID == "" {
		return GameState{}, ErrPlayerNotFound
	}
	if buildingID == "" {
		return GameState{}, ErrBuildingNotFound
	}

	state, err := s.repo.GetState(playerID)
	if err != nil {
		return GameState{}, err
	}

	now := time.Now()
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

	// 必须正在升级中
	if building.UpgradeEndsAt == nil {
		return GameState{}, ErrNotUpgrading
	}

	// 计算剩余秒数
	endsAt, err := time.Parse(resourceDateLayout, *building.UpgradeEndsAt)
	if err != nil {
		return GameState{}, ErrNotUpgrading
	}
	remainingSecs := int(endsAt.Sub(now).Seconds())
	if remainingSecs <= 0 {
		// 已经完成了，直接结算
		state, _ = settleResources(state, now)
		state.ServerTime = now.UTC().Format(resourceDateLayout)
		_ = s.repo.SaveState(state, now)
		return state, nil
	}

	// 计算城金花费并扣除
	cost := speedUpCost(remainingSecs)
	if _, err := s.repo.DeductCityGold(playerID, cost); err != nil {
		return GameState{}, err
	}

	// 立即完成升级
	building.Level++
	building.UpgradeEndsAt = nil

	// 重新计算产量和容量
	state.ResourceSettledAt = now.UTC().Format(resourceDateLayout)
	state.ServerTime = now.UTC().Format(resourceDateLayout)

	// 重新读取城金余额（已扣除）
	latestState, _ := s.repo.GetState(playerID)
	state.CityGold = latestState.CityGold

	if err := s.repo.SaveState(state, now); err != nil {
		return GameState{}, err
	}

	// 重新结算一次确保产量/容量更新
	state, _ = settleResources(state, now)
	state.ServerTime = now.UTC().Format(resourceDateLayout)
	_ = s.repo.SaveState(state, now)

	return state, nil
}
