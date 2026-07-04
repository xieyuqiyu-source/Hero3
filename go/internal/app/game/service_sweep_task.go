// 本文件实现 NPC 一键扫荡后台任务服务，把长耗时 HTTP 请求拆成提交任务和查询任务。
package game

import (
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"
)

const maxNpcSweepTaskTargets = 50
const npcSweepTaskRetention = 14 * 24 * time.Hour

// StartNpcSweepTask 创建 NPC 扫荡任务并异步执行。
func (s *Service) StartNpcSweepTask(req SweepNpcRequest) (NpcSweepTask, error) {
	playerID := strings.TrimSpace(req.PlayerID)
	if playerID == "" {
		return NpcSweepTask{}, ErrPlayerNotFound
	}
	npcIDs := normalizeSweepNpcIDs(req.NpcIDs)
	if len(npcIDs) == 0 {
		return NpcSweepTask{}, ErrNoSweepTargets
	}
	if len(npcIDs) > maxNpcSweepTaskTargets {
		return NpcSweepTask{}, ErrSweepTargetsTooMany
	}
	mode := strings.TrimSpace(req.Mode)
	if mode != "attack" && mode != "plunder" {
		mode = "attack"
	}
	now := time.Now().UTC()
	task := NpcSweepTask{
		ID:         "sweep_task_" + randomID(12),
		PlayerID:   playerID,
		NpcIDs:     npcIDs,
		Mode:       mode,
		GeneralIDs: normalizeStringIDs(req.GeneralIDs),
		Status:     SweepTaskStatusQueued,
		Requested:  len(npcIDs),
		CreatedAt:  now.Format(resourceDateLayout),
		UpdatedAt:  now.Format(resourceDateLayout),
	}
	created, err := s.repo.CreateNpcSweepTask(task)
	if err != nil {
		if errors.Is(err, ErrNpcSweepTaskRunning) {
			return created, ErrNpcSweepTaskRunning
		}
		return NpcSweepTask{}, err
	}
	go s.runNpcSweepTask(created.ID)
	return created, nil
}

// GetNpcSweepTask 查询指定玩家的一键扫荡任务。
func (s *Service) GetNpcSweepTask(playerID string, taskID string) (NpcSweepTask, error) {
	playerID = strings.TrimSpace(playerID)
	if playerID == "" {
		return NpcSweepTask{}, ErrPlayerNotFound
	}
	taskID = strings.TrimSpace(taskID)
	if taskID == "" {
		return NpcSweepTask{}, ErrNpcSweepTaskNotFound
	}
	return s.repo.GetNpcSweepTask(playerID, taskID)
}

// runNpcSweepTask 在后台执行完整 NPC 扫荡任务，并把最终结果写回任务表。
func (s *Service) runNpcSweepTask(taskID string) {
	defer func() {
		if recovered := recover(); recovered != nil {
			finishedAt := time.Now().UTC()
			_, err := s.repo.UpdateNpcSweepTask(taskID, finishedAt, func(task *NpcSweepTask) error {
				task.Status = SweepTaskStatusFailed
				task.Error = fmt.Sprintf("sweep task panic: %v", recovered)
				task.CompletedAt = finishedAt.Format(resourceDateLayout)
				return nil
			})
			if err != nil {
				slog.Error("npc sweep task panic update failed", "taskId", taskID, "panic", recovered, "error", err)
				return
			}
			slog.Error("npc sweep task recovered panic", "taskId", taskID, "panic", recovered)
		}
	}()

	now := time.Now().UTC()
	task, err := s.repo.UpdateNpcSweepTask(taskID, now, func(task *NpcSweepTask) error {
		task.Status = SweepTaskStatusRunning
		task.StartedAt = now.Format(resourceDateLayout)
		return nil
	})
	if err != nil {
		slog.Warn("npc sweep task start failed", "taskId", taskID, "error", err)
		return
	}

	result, err := s.sweepNpc(SweepNpcRequest{
		PlayerID:   task.PlayerID,
		NpcIDs:     task.NpcIDs,
		Mode:       task.Mode,
		GeneralIDs: task.GeneralIDs,
	}, 0)
	finishedAt := time.Now().UTC()
	if err != nil {
		_, updateErr := s.repo.UpdateNpcSweepTask(taskID, finishedAt, func(task *NpcSweepTask) error {
			task.Status = SweepTaskStatusFailed
			task.Error = err.Error()
			task.CompletedAt = finishedAt.Format(resourceDateLayout)
			return nil
		})
		if updateErr != nil {
			slog.Warn("npc sweep task failure update failed", "taskId", taskID, "error", updateErr)
		}
		return
	}

	_, updateErr := s.repo.UpdateNpcSweepTask(taskID, finishedAt, func(task *NpcSweepTask) error {
		task.Status = SweepTaskStatusCompleted
		task.Done = result.Done
		task.Failed = result.Failed
		task.Stopped = result.Stopped
		task.Result = &result
		task.CompletedAt = finishedAt.Format(resourceDateLayout)
		return nil
	})
	if updateErr != nil {
		slog.Warn("npc sweep task completion update failed", "taskId", taskID, "error", updateErr)
	}
}

// IsNpcSweepTaskRunning 判断错误是否表示玩家已有扫荡任务未完成。
func IsNpcSweepTaskRunning(err error) bool {
	return errors.Is(err, ErrNpcSweepTaskRunning)
}

// recoverNpcSweepTasksOnStartup 处理上一次服务进程遗留的扫荡任务，并清理过期任务记录。
func (s *Service) recoverNpcSweepTasksOnStartup() {
	if s == nil || s.repo == nil {
		return
	}
	now := time.Now().UTC()
	recovered, err := s.repo.FailActiveNpcSweepTasks(now, "服务重启，扫荡任务已中断，请重新发起")
	if err != nil {
		slog.Warn("npc sweep task startup recovery failed", "error", err)
	} else if recovered > 0 {
		slog.Warn("npc sweep tasks marked failed on startup", "count", recovered)
	}
	deleted, err := s.repo.DeleteNpcSweepTasksCompletedBefore(now.Add(-npcSweepTaskRetention))
	if err != nil {
		slog.Warn("npc sweep task retention cleanup failed", "error", err)
	} else if deleted > 0 {
		slog.Info("old npc sweep tasks cleaned", "count", deleted)
	}
}
