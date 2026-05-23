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

	// 计算训练总时间（串行：基于队列最后一个任务的结束时间）
	totalSeconds := unitConfig.TrainSeconds * amount
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
