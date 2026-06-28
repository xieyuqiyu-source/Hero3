// 本文件实现内存仓储中的轮回绝境副本状态读写。
package game

import (
	"encoding/json"
	"sort"
	"time"
)

// GetActiveReincarnationRun 获取玩家当前仍在进行中的轮回实例。
func (r *MemoryRepository) GetActiveReincarnationRun(playerID string, now time.Time) (ReincarnationRun, bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, run := range r.reincarnationRuns {
		if run.PlayerID == playerID && run.Status == ReincarnationRunRunning {
			return cloneReincarnationRun(run), true, nil
		}
	}
	return ReincarnationRun{}, false, nil
}

// GetReincarnationRun 获取指定轮回实例。
func (r *MemoryRepository) GetReincarnationRun(runID string) (ReincarnationRun, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	run, ok := r.reincarnationRuns[runID]
	if !ok {
		return ReincarnationRun{}, ErrReincarnationRunNotFound
	}
	return cloneReincarnationRun(run), nil
}

// SaveReincarnationRun 保存轮回实例快照。
func (r *MemoryRepository) SaveReincarnationRun(run ReincarnationRun) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.reincarnationRuns[run.ID] = cloneReincarnationRun(run)
	return nil
}

// UpdateReincarnationRunWithState 在同一内存锁内更新玩家状态和轮回实例。
func (r *MemoryRepository) UpdateReincarnationRunWithState(playerID string, runID string, updatedAt time.Time, update func(state *GameState, run *ReincarnationRun) ([]BattleReport, error)) (GameState, ReincarnationRun, []BattleReport, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	state, ok := r.players[playerID]
	if !ok {
		return GameState{}, ReincarnationRun{}, nil, ErrPlayerNotFound
	}
	run, ok := r.reincarnationRuns[runID]
	if !ok {
		return GameState{}, ReincarnationRun{}, nil, ErrReincarnationRunNotFound
	}
	reports, err := update(&state, &run)
	if err != nil {
		return GameState{}, ReincarnationRun{}, nil, err
	}
	run.UpdatedAt = updatedAt
	cloned, err := cloneGameState(state)
	if err != nil {
		return GameState{}, ReincarnationRun{}, nil, err
	}
	r.players[playerID] = cloned
	r.playerUpdatedAt[playerID] = updatedAt
	r.reincarnationRuns[run.ID] = cloneReincarnationRun(run)
	for _, report := range reports {
		normalized := NormalizeBattleReport(report)
		r.reports[normalized.PlayerID] = append(r.reports[normalized.PlayerID], normalized)
	}
	resultState, err := cloneGameState(state)
	if err != nil {
		return GameState{}, ReincarnationRun{}, nil, err
	}
	return resultState, cloneReincarnationRun(run), append([]BattleReport(nil), reports...), nil
}

// ListReincarnationRuns 查询玩家轮回实例列表。
func (r *MemoryRepository) ListReincarnationRuns(playerID string, limit int, offset int) ([]ReincarnationRun, int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	items := []ReincarnationRun{}
	for _, run := range r.reincarnationRuns {
		if playerID == "" || run.PlayerID == playerID {
			items = append(items, cloneReincarnationRun(run))
		}
	}
	sort.Slice(items, func(i, j int) bool { return items[i].StartedAt.After(items[j].StartedAt) })
	total := len(items)
	if offset > total {
		return []ReincarnationRun{}, total, nil
	}
	end := offset + limit
	if limit <= 0 || end > total {
		end = total
	}
	return items[offset:end], total, nil
}

func cloneReincarnationRun(run ReincarnationRun) ReincarnationRun {
	data, _ := json.Marshal(run)
	var cloned ReincarnationRun
	_ = json.Unmarshal(data, &cloned)
	return cloned
}
