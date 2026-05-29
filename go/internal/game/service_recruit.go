package game

import (
	"strings"
	"time"
)

func (s *Service) Recruit(playerID string, unitID string, amount int) (GameState, error) {
	playerID = strings.TrimSpace(playerID)
	unitID = strings.TrimSpace(unitID)
	if playerID == "" {
		return GameState{}, ErrPlayerNotFound
	}
	if unitID == "" {
		return GameState{}, ErrUnitNotFound
	}
	if amount <= 0 || amount > 100000 {
		return GameState{}, ErrInvalidAmount
	}

	state, err := s.repo.GetState(playerID)
	if err != nil {
		return GameState{}, err
	}

	now := time.Now()
	state, _ = settleResources(state, now)

	// 检查队列上限
	pendingCount := len(state.RecruitQueues)
	if pendingCount >= 5 {
		return GameState{}, ErrQueueFull
	}

	// 查找兵种配置
	unitConfig, exists := GetUnitConfig(state.Player.Faction, unitID)
	if !exists {
		return GameState{}, ErrUnitNotFound
	}

	// 检查资源是否足够（单价 × 数量）
	for resType, costPer := range unitConfig.Cost {
		totalCost := costPer * amount
		if state.Resources.Items[resType] < totalCost {
			return GameState{}, ErrInsufficientRes
		}
	}

	// 扣减资源
	for resType, costPer := range unitConfig.Cost {
		state.Resources.Items[resType] -= costPer * amount
	}

	// 计算训练总时间（经过 recruitSpeedBonus 加成缩短，串行：基于队列最后一个任务的结束时间）
	totalSeconds := unitConfig.TrainSeconds * amount
	modSources := CollectModifierSources(&state)
	totalSeconds = applySpeedBonus(totalSeconds, "recruitSpeedBonus", now, modSources)
	queueStart := now
	for _, q := range state.RecruitQueues {
		if parsed, err := time.Parse(resourceDateLayout, q.EndsAt); err == nil && parsed.After(queueStart) {
			queueStart = parsed
		}
	}
	endsAt := queueStart.Add(time.Duration(totalSeconds) * time.Second).UTC().Format(resourceDateLayout)

	// 添加征兵队列
	queue := RecruitQueue{
		ID:       "rq_" + randomID(8),
		UnitType: unitID,
		Amount:   amount,
		EndsAt:   endsAt,
	}
	state.RecruitQueues = append(state.RecruitQueues, queue)

	state.ResourceSettledAt = now.UTC().Format(resourceDateLayout)
	state.ServerTime = now.UTC().Format(resourceDateLayout)

	if err := s.repo.SaveState(state, now); err != nil {
		return GameState{}, err
	}

	return state, nil
}

// InstantCompleteRecruit 极速完成征兵队列（消耗城金）
func (s *Service) InstantCompleteRecruit(playerID string, queueID string) (GameState, error) {
	playerID = strings.TrimSpace(playerID)
	queueID = strings.TrimSpace(queueID)
	if playerID == "" {
		return GameState{}, ErrPlayerNotFound
	}
	if queueID == "" {
		return GameState{}, ErrQueueFull
	}

	state, err := s.repo.GetState(playerID)
	if err != nil {
		return GameState{}, err
	}

	now := time.Now()
	state, _ = settleResources(state, now)

	// 找到目标队列，计算剩余时间和费用
	queueIdx := -1
	for i, queue := range state.RecruitQueues {
		if queue.ID == queueID {
			queueIdx = i
			break
		}
	}
	if queueIdx == -1 {
		return GameState{}, ErrQueueFull
	}

	queue := state.RecruitQueues[queueIdx]

	// 计算该队列自身的训练时长（不含排队等待时间，经过 recruitSpeedBonus 缩短）
	unitCfg, unitExists := GetUnitConfig(state.Player.Faction, queue.UnitType)
	trainSeconds := 0
	if unitExists {
		trainSeconds = unitCfg.TrainSeconds * queue.Amount
		instantModSources := CollectModifierSources(&state)
		trainSeconds = applySpeedBonus(trainSeconds, "recruitSpeedBonus", now, instantModSources)
	}

	// 计算城金花费并扣除
	if trainSeconds > 0 {
		cost := speedUpCost(trainSeconds)
		if _, err := s.repo.DeductCityGold(playerID, cost); err != nil {
			return GameState{}, err
		}
		// 重新读取最新状态（城金已扣）
		state, _ = s.repo.GetState(playerID)
		state, _ = settleResources(state, now)
	}

	// 找到目标队列并立即完成（重新定位，因为 state 可能重新加载了）
	found := false
	for i, q := range state.RecruitQueues {
		if q.ID == queueID {
			// 加入军队
			addedToArmy := false
			for j := range state.Army {
				if state.Army[j].UnitType == q.UnitType {
					state.Army[j].Amount += q.Amount
					addedToArmy = true
					break
				}
			}
			if !addedToArmy {
				state.Army = append(state.Army, ArmyUnit{
					UnitType: q.UnitType,
					Amount:   q.Amount,
				})
			}

			// 从队列移除
			state.RecruitQueues = append(state.RecruitQueues[:i], state.RecruitQueues[i+1:]...)

			// 后续队列的 endsAt 需要前移
			if i < len(state.RecruitQueues) {
				prevEnd := now
				if i > 0 {
					if parsed, err := time.Parse(resourceDateLayout, state.RecruitQueues[i-1].EndsAt); err == nil {
						prevEnd = parsed
					}
				}
				for j := i; j < len(state.RecruitQueues); j++ {
					rq := &state.RecruitQueues[j]
					unitCfg, exists := GetUnitConfig(state.Player.Faction, rq.UnitType)
					if !exists {
						continue
					}
					duration := time.Duration(unitCfg.TrainSeconds*rq.Amount) * time.Second
					newEnd := prevEnd.Add(duration)
					rq.EndsAt = newEnd.UTC().Format(resourceDateLayout)
					prevEnd = newEnd
				}
			}

			found = true
			break
		}
	}

	if !found {
		return GameState{}, ErrQueueFull
	}

	state.ResourceSettledAt = now.UTC().Format(resourceDateLayout)
	state.ServerTime = now.UTC().Format(resourceDateLayout)

	if err := s.repo.SaveState(state, now); err != nil {
		return GameState{}, err
	}

	return state, nil
}
