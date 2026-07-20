// 本文件实现刘备“仁德天下”的战后复活和战报明细。
package traits

import (
	"math"

	"hero3/internal/core/general"
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

// init 注册仁德特性。
func init() {
	general.Register(&Rende{})
}

// ID 返回仁德特性 ID。
func (r *Rende) ID() string { return "rende" }

// Name 返回仁德展示名称。
func (r *Rende) Name() string { return "仁德天下" }

// Type 返回仁德特性类型。
func (r *Rende) Type() string { return general.TraitTypeSpecial }

// Description 返回仁德当前效果说明。
func (r *Rende) Description(p general.Params) string {
	return "战斗结束后，损失的兵有概率复活归队"
}

// ParamSchema 返回仁德可配置参数。
func (r *Rende) ParamSchema() []general.ParamField {
	return []general.ParamField{
		{Key: "effectRate", Label: "复活比例", Description: "损失兵的复活比例", Default: 0.5, Min: 0, Max: 1, Step: 0.01},
		{Key: "maxReviveCount", Label: "单场复活上限", Description: "单场最多复活士兵数量", Default: 10000, Min: 0, Max: 1000000, Step: 100},
		{Key: "reviveRate", Label: "复活比例", Description: "损失兵的复活比例", Default: 0.2, Min: 0, Max: 1, Step: 0.01},
		{Key: "triggerChance", Label: "触发概率", Description: "复活技能的发动概率", Default: 0.5, Min: 0, Max: 1, Step: 0.05},
	}
}

// Subscribe 订阅战斗结束事件。
func (r *Rende) Subscribe() []general.EventSubscription {
	return []general.EventSubscription{
		{
			Event:    general.EventAfterBattle,
			Priority: 50,
			Handle:   r.afterBattle,
		},
	}
}

// afterBattle 计算各兵种真实复活数并写入特性战报。
func (r *Rende) afterBattle(ctx general.EventContext, p general.Params) {
	c, ok := ctx.(*general.AfterBattleContext)
	if !ok {
		return
	}

	if !triggeredWithDefault(p, 0.5) {
		return
	}

	// 复活比例（限制在 0-1 之间，防止复活超过 100%）
	rate := p.FloatWithBounds("effectRate", p.FloatOr("reviveRate", 0.2), 0, 1)
	maxRevive := p.IntWithBounds("maxReviveCount", 10000, 0, 1000000)
	if rate <= 0 || len(c.PlayerLosses) == 0 {
		return
	}

	if c.Revived == nil {
		c.Revived = map[string]int{}
	}

	totalRevived := 0
	revivedUnits := map[string]int{}
	for _, unitType := range sortedLossUnitTypes(c.PlayerLosses) {
		lost := c.PlayerLosses[unitType]
		revived := int(math.Floor(float64(lost) * rate))
		remainingLoss := lost - c.Revived[unitType]
		if revived > remainingLoss {
			revived = remainingLoss
		}
		if revived <= 0 {
			continue
		}
		if maxRevive > 0 && totalRevived+revived > maxRevive {
			revived = maxRevive - totalRevived
		}
		if revived <= 0 {
			break
		}
		c.Revived[unitType] += revived
		c.PlayerArmy[unitType] += revived
		revivedUnits[unitType] += revived
		totalRevived += revived
	}

	if totalRevived > 0 {
		if c.Triggered == nil {
			c.Triggered = map[string]general.TraitOutcome{}
		}
		c.Triggered["rende"] = general.TraitOutcome{
			TraitID:        "rende",
			Name:           "仁德天下",
			TraitType:      general.TraitTypeSpecial,
			OwnerSide:      c.Actor.Side,
			OwnerGeneralID: c.Actor.GeneralID,
			OwnerPlayerID:  c.Actor.PlayerID,
			Scope:          c.Actor.Scope,
			Detail: map[string]interface{}{
				"totalRevived":   totalRevived,
				"revivedUnits":   revivedUnits,
				"effectRate":     rate,
				"maxReviveCount": maxRevive,
				"triggerChance":  p.FloatWithBounds("triggerChance", 0.5, 0, 1),
			},
		}
	}
}
