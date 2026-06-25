package effect

import (
	"strings"

	corebuilding "hero3/internal/core/building"
	corereward "hero3/internal/core/reward"
)

// 本文件定义稳定效果契约。核心只描述效果，不直接修改玩家存档。

const (
	TypeReward           = "reward"
	TypeBuildingMutation = "building_mutation"
	TypeModifier         = "modifier"
)

type Effect struct {
	Type             string                 `json:"type"`
	Source           string                 `json:"source,omitempty"`
	ID               string                 `json:"id,omitempty"`
	Amount           int                    `json:"amount,omitempty"`
	Rewards          []corereward.Reward    `json:"rewards,omitempty"`
	BuildingMutation *corebuilding.Mutation `json:"buildingMutation,omitempty"`
	Modifier         *ModifierEffect        `json:"modifier,omitempty"`
	Metadata         map[string]any         `json:"metadata,omitempty"`
}

type ModifierEffect struct {
	Key    string  `json:"key"`
	Value  float64 `json:"value"`
	Mode   string  `json:"mode"`
	Hours  int     `json:"hours,omitempty"`
	Source string  `json:"source,omitempty"`
	Note   string  `json:"note,omitempty"`
	Stack  int     `json:"stack,omitempty"`
}

type Context struct {
	AccountID string
	PlayerID  string
	RefType   string
	RefID     string
	Reason    string
	Source    string
}

type ApplyResult struct {
	Applied int      `json:"applied"`
	Types   []string `json:"types"`
}

// NormalizeType 规范化效果类型。
func NormalizeType(effectType string) string {
	return strings.TrimSpace(effectType)
}

// RewardEffect 把奖励列表包装成标准效果。
func RewardEffect(source string, rewards ...corereward.Reward) Effect {
	return Effect{
		Type:    TypeReward,
		Source:  strings.TrimSpace(source),
		Rewards: rewards,
	}
}

// BuildingMutationEffect 把建筑变更包装成标准效果。
func BuildingMutationEffect(source string, mutation corebuilding.Mutation) Effect {
	return Effect{
		Type:             TypeBuildingMutation,
		Source:           strings.TrimSpace(source),
		ID:               mutation.BuildingID,
		BuildingMutation: &mutation,
	}
}

// ModifierEffectRequest 把 Modifier/Buff 变更包装成标准效果。
func ModifierEffectRequest(source string, modifier ModifierEffect) Effect {
	return Effect{
		Type:     TypeModifier,
		Source:   strings.TrimSpace(source),
		ID:       modifier.Key,
		Modifier: &modifier,
	}
}
