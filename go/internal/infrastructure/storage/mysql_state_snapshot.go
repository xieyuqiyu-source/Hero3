// 本文件归口 players.state_json 轻量兼容快照生成逻辑。
package storage

import (
	"encoding/json"

	"hero3/internal/app/game"
)

// playerStateSnapshot 是 players.state_json 的轻量兼容结构。
type playerStateSnapshot struct {
	Player               game.Player        `json:"player"`
	ResourceSettledAt    string             `json:"resourceSettledAt,omitempty"`
	GeneralTraitProgress map[string]float64 `json:"generalTraitProgress,omitempty"`
	ProductionBoost      int                `json:"productionBoost,omitempty"`
	ProductionBoostEnd   string             `json:"productionBoostEnd,omitempty"`
	CapacityBoost        int                `json:"capacityBoost,omitempty"`
	CapacityBoostEnd     string             `json:"capacityBoostEnd,omitempty"`
	MapTargets           []game.MapTarget   `json:"mapTargets,omitempty"`
	UnreadMessageCount   int                `json:"unreadMessageCount,omitempty"`
	UnreadMailCount      int                `json:"unreadMailCount,omitempty"`
	DeleteRequestedAt    string             `json:"deleteRequestedAt,omitempty"`
	DeleteScheduledAt    string             `json:"deleteScheduledAt,omitempty"`
}

// marshalPlayerStateSnapshot 生成轻量兼容快照，避免把已有权威表承载的大字段反复写入 players.state_json。
func marshalPlayerStateSnapshot(state game.GameState) ([]byte, error) {
	return json.Marshal(compactPlayerStateSnapshot(state))
}

// compactPlayerStateSnapshot 只保留尚未拆出权威表或用于兼容恢复的字段。
func compactPlayerStateSnapshot(state game.GameState) playerStateSnapshot {
	return playerStateSnapshot{
		Player:               state.Player,
		ResourceSettledAt:    state.ResourceSettledAt,
		GeneralTraitProgress: state.GeneralTraitProgress,
		ProductionBoost:      state.ProductionBoost,
		ProductionBoostEnd:   state.ProductionBoostEnd,
		CapacityBoost:        state.CapacityBoost,
		CapacityBoostEnd:     state.CapacityBoostEnd,
		MapTargets:           state.MapTargets,
		UnreadMessageCount:   state.UnreadMessageCount,
		UnreadMailCount:      state.UnreadMailCount,
		DeleteRequestedAt:    state.DeleteRequestedAt,
		DeleteScheduledAt:    state.DeleteScheduledAt,
	}
}
