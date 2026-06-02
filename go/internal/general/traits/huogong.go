package traits

import (
	"math"
	"math/rand"

	"hero3/internal/general"
)

// Huogong 火攻（周瑜）
//
// 战斗结算后触发：
// - 按 triggerChance 概率触发
// - 触发后给敌方加 damagePercent 的额外损失（按敌方原始兵力计算）
// - 不论强弱都能触发，弱玩家也能让强玩家损失惨重
// - 只对主动出征生效（防守不触发）
type Huogong struct{}

func init() {
	general.Register(&Huogong{})
}

func (h *Huogong) ID() string   { return "huogong" }
func (h *Huogong) Name() string { return "火攻" }

func (h *Huogong) Description(p general.Params) string {
	return "战斗中有概率发动火攻，对敌方造成额外百分比伤害"
}

func (h *Huogong) ParamSchema() []general.ParamField {
	return []general.ParamField{
		{Key: "damagePercent", Label: "额外伤害百分比", Description: "敌方按原始兵力额外损失的比例", Default: 0.15, Min: 0, Max: 1, Step: 0.01},
		{Key: "triggerChance", Label: "触发概率", Description: "战斗时发动火攻的概率", Default: 0.6, Min: 0, Max: 1, Step: 0.05},
	}
}

func (h *Huogong) Subscribe() []general.EventSubscription {
	return []general.EventSubscription{
		{
			Event:    general.EventAfterCombatResolve,
			Priority: 50,
			Handle:   h.afterCombat,
		},
	}
}

func (h *Huogong) afterCombat(ctx general.EventContext, p general.Params) {
	c, ok := ctx.(*general.AfterCombatResolveContext)
	if !ok {
		return
	}
	// 只在进攻方拥有时生效
	if !c.AttackerOwnsTrait {
		return
	}

	chance := p.FloatOr("triggerChance", 0.6)
	if rand.Float64() > chance {
		return
	}

	damagePct := p.FloatOr("damagePercent", 0.15)
	if damagePct <= 0 {
		return
	}

	// 给敌方各兵种增加额外损失（按 Count × damagePct，向下取整，至少 1）
	totalExtra := 0
	for i := range c.Result.DefenderLosses {
		loss := &c.Result.DefenderLosses[i]
		extra := int(math.Floor(float64(loss.Count) * damagePct))
		if extra <= 0 {
			extra = 1
		}
		loss.Losses += extra
		if loss.Losses > loss.Count {
			extra -= loss.Losses - loss.Count
			loss.Losses = loss.Count
		}
		if extra > 0 {
			totalExtra += extra
		}
	}

	// 重新计算敌方损失率
	var totalCount, totalLost int
	for _, l := range c.Result.DefenderLosses {
		totalCount += l.Count
		totalLost += l.Losses
	}
	if totalCount > 0 {
		c.Result.DefenderLossRate = float64(totalLost) / float64(totalCount)
	}

	// 记录触发
	if totalExtra > 0 {
		if c.Triggered == nil {
			c.Triggered = map[string]general.TraitOutcome{}
		}
		c.Triggered["huogong"] = general.TraitOutcome{
			TraitID: "huogong",
			Name:    "火攻",
			Detail: map[string]interface{}{
				"extraDamage":   totalExtra,
				"damagePercent": damagePct,
			},
		}
	}
}
