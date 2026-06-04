package traits

import (
	"math"
	"math/rand"

	"hero3/internal/general"
)

// Rende 仁德（刘备）
//
// 战斗结束后触发：
// - 按 triggerChance 概率触发
// - 触发后将损失的兵按 reviveRate 比例复活，加回军队
// - 进攻 / 防守 都能触发
// - 胜利 / 失败 都能触发
// - 所有兵种都能复活
type Rende struct{}

func init() {
	general.Register(&Rende{})
}

func (r *Rende) ID() string   { return "rende" }
func (r *Rende) Name() string { return "仁德" }

func (r *Rende) Description(p general.Params) string {
	return "战斗结束后，损失的兵有概率复活归队"
}

func (r *Rende) ParamSchema() []general.ParamField {
	return []general.ParamField{
		{Key: "reviveRate", Label: "复活比例", Description: "损失兵的复活比例", Default: 0.2, Min: 0, Max: 1, Step: 0.01},
		{Key: "triggerChance", Label: "触发概率", Description: "复活技能的发动概率", Default: 0.5, Min: 0, Max: 1, Step: 0.05},
	}
}

func (r *Rende) Subscribe() []general.EventSubscription {
	return []general.EventSubscription{
		{
			Event:    general.EventAfterBattle,
			Priority: 50,
			Handle:   r.afterBattle,
		},
	}
}

func (r *Rende) afterBattle(ctx general.EventContext, p general.Params) {
	c, ok := ctx.(*general.AfterBattleContext)
	if !ok {
		return
	}

	// 触发概率（限制在 0-1 之间）
	chance := p.FloatWithBounds("triggerChance", 0.5, 0, 1)
	if rand.Float64() > chance {
		return
	}

	// 复活比例（限制在 0-1 之间，防止复活超过 100%）
	rate := p.FloatWithBounds("reviveRate", 0.2, 0, 1)
	if rate <= 0 || len(c.PlayerLosses) == 0 {
		return
	}

	if c.Revived == nil {
		c.Revived = map[string]int{}
	}

	totalRevived := 0
	for unitType, lost := range c.PlayerLosses {
		if lost <= 0 {
			continue
		}
		revived := int(math.Floor(float64(lost) * rate))
		if revived <= 0 {
			continue
		}
		c.Revived[unitType] += revived
		c.PlayerArmy[unitType] += revived
		totalRevived += revived
	}

	if totalRevived > 0 {
		if c.Triggered == nil {
			c.Triggered = map[string]general.TraitOutcome{}
		}
		c.Triggered["rende"] = general.TraitOutcome{
			TraitID: "rende",
			Name:    "仁德",
			Detail: map[string]interface{}{
				"totalRevived": totalRevived,
			},
		}
	}
}
