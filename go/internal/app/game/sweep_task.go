// 本文件定义 NPC 一键扫荡任务模型，用于把长耗时扫荡请求改为后台任务。
package game

import "strings"

const (
	SweepTaskStatusQueued    = "queued"
	SweepTaskStatusRunning   = "running"
	SweepTaskStatusCompleted = "completed"
	SweepTaskStatusFailed    = "failed"
)

// NpcSweepTask 记录一次 NPC 一键扫荡后台任务的请求、进度和最终结果。
type NpcSweepTask struct {
	ID          string            `json:"id"`
	PlayerID    string            `json:"playerId"`
	NpcIDs      []string          `json:"npcIds"`
	Mode        string            `json:"mode"`
	GeneralIDs  []string          `json:"generalIds,omitempty"`
	Status      string            `json:"status"`
	Requested   int               `json:"requested"`
	Done        int               `json:"done"`
	Failed      int               `json:"failed"`
	Stopped     bool              `json:"stopped"`
	Error       string            `json:"error,omitempty"`
	Result      *SweepNpcResponse `json:"result,omitempty"`
	CreatedAt   string            `json:"createdAt"`
	UpdatedAt   string            `json:"updatedAt"`
	StartedAt   string            `json:"startedAt,omitempty"`
	CompletedAt string            `json:"completedAt,omitempty"`
}

// Normalize 整理扫荡任务字段，确保状态和模式有稳定默认值。
func (task *NpcSweepTask) Normalize() {
	if task == nil {
		return
	}
	task.PlayerID = strings.TrimSpace(task.PlayerID)
	task.Mode = strings.TrimSpace(task.Mode)
	if task.Mode != "attack" && task.Mode != "plunder" {
		task.Mode = "attack"
	}
	task.NpcIDs = normalizeSweepNpcIDs(task.NpcIDs)
	task.GeneralIDs = normalizeStringIDs(task.GeneralIDs)
	task.Requested = len(task.NpcIDs)
	if strings.TrimSpace(task.Status) == "" {
		task.Status = SweepTaskStatusQueued
	}
}

// normalizeStringIDs 去重并过滤空字符串 ID。
func normalizeStringIDs(values []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		id := strings.TrimSpace(value)
		if id == "" || seen[id] {
			continue
		}
		seen[id] = true
		out = append(out, id)
	}
	return out
}
