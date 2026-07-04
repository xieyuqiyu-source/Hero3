// 本文件实现内存版 NPC 扫荡任务仓储，供开发模式和服务层测试使用。
package game

import (
	"encoding/json"
	"errors"
	"time"
)

var ErrNpcSweepTaskNotFound = errors.New("npc sweep task not found")
var ErrNpcSweepTaskRunning = errors.New("npc sweep task already running")

// CreateNpcSweepTask 原子创建 NPC 扫荡任务；同一玩家已有活跃任务时返回该任务和占用错误。
func (r *MemoryRepository) CreateNpcSweepTask(task NpcSweepTask) (NpcSweepTask, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	task.Normalize()
	for _, active := range r.npcSweepTasks {
		if active.PlayerID != task.PlayerID {
			continue
		}
		if active.Status == SweepTaskStatusQueued || active.Status == SweepTaskStatusRunning {
			return cloneNpcSweepTask(active), ErrNpcSweepTaskRunning
		}
	}
	if task.ID == "" {
		task.ID = "sweep_task_" + randomID(12)
	}
	if task.CreatedAt == "" {
		task.CreatedAt = time.Now().UTC().Format(resourceDateLayout)
	}
	if task.UpdatedAt == "" {
		task.UpdatedAt = task.CreatedAt
	}
	r.npcSweepTasks[task.ID] = cloneNpcSweepTask(task)
	return cloneNpcSweepTask(task), nil
}

// GetNpcSweepTask 读取指定玩家的一键扫荡任务。
func (r *MemoryRepository) GetNpcSweepTask(playerID string, taskID string) (NpcSweepTask, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	task, ok := r.npcSweepTasks[taskID]
	if !ok || task.PlayerID != playerID {
		return NpcSweepTask{}, ErrNpcSweepTaskNotFound
	}
	return cloneNpcSweepTask(task), nil
}

// FindActiveNpcSweepTask 查找玩家当前未完成的扫荡任务。
func (r *MemoryRepository) FindActiveNpcSweepTask(playerID string) (NpcSweepTask, bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, task := range r.npcSweepTasks {
		if task.PlayerID != playerID {
			continue
		}
		if task.Status == SweepTaskStatusQueued || task.Status == SweepTaskStatusRunning {
			return cloneNpcSweepTask(task), true, nil
		}
	}
	return NpcSweepTask{}, false, nil
}

// UpdateNpcSweepTask 在仓储锁内更新扫荡任务进度和结果。
func (r *MemoryRepository) UpdateNpcSweepTask(taskID string, updatedAt time.Time, update func(task *NpcSweepTask) error) (NpcSweepTask, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	task, ok := r.npcSweepTasks[taskID]
	if !ok {
		return NpcSweepTask{}, ErrNpcSweepTaskNotFound
	}
	task = cloneNpcSweepTask(task)
	if update != nil {
		if err := update(&task); err != nil {
			return NpcSweepTask{}, err
		}
	}
	task.Normalize()
	if updatedAt.IsZero() {
		updatedAt = time.Now()
	}
	task.UpdatedAt = updatedAt.UTC().Format(resourceDateLayout)
	r.npcSweepTasks[taskID] = cloneNpcSweepTask(task)
	return cloneNpcSweepTask(task), nil
}

// FailActiveNpcSweepTasks 把进程重启后无人执行的活跃扫荡任务统一置为失败。
func (r *MemoryRepository) FailActiveNpcSweepTasks(updatedAt time.Time, reason string) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if updatedAt.IsZero() {
		updatedAt = time.Now()
	}
	if reason == "" {
		reason = "sweep task interrupted by service restart"
	}
	updated := 0
	completedAt := updatedAt.UTC().Format(resourceDateLayout)
	for id, task := range r.npcSweepTasks {
		if task.Status != SweepTaskStatusQueued && task.Status != SweepTaskStatusRunning {
			continue
		}
		task.Status = SweepTaskStatusFailed
		task.Error = reason
		task.UpdatedAt = completedAt
		task.CompletedAt = completedAt
		r.npcSweepTasks[id] = cloneNpcSweepTask(task)
		updated++
	}
	return updated, nil
}

// DeleteNpcSweepTasksCompletedBefore 删除过期的已完成或失败扫荡任务，控制任务表体积。
func (r *MemoryRepository) DeleteNpcSweepTasksCompletedBefore(cutoff time.Time) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	deleted := 0
	for id, task := range r.npcSweepTasks {
		if task.Status != SweepTaskStatusCompleted && task.Status != SweepTaskStatusFailed {
			continue
		}
		completedAt, err := time.Parse(resourceDateLayout, task.CompletedAt)
		if err != nil || completedAt.IsZero() || !completedAt.Before(cutoff) {
			continue
		}
		delete(r.npcSweepTasks, id)
		deleted++
	}
	return deleted, nil
}

// cloneNpcSweepTask 深拷贝扫荡任务，隔离仓储内部状态。
func cloneNpcSweepTask(task NpcSweepTask) NpcSweepTask {
	data, err := json.Marshal(task)
	if err != nil {
		return task
	}
	var out NpcSweepTask
	if err := json.Unmarshal(data, &out); err != nil {
		return task
	}
	return out
}
