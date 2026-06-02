// Package traits 包含所有将领特性实现
//
// 每个特性是一个独立文件，结构清晰：
//   - 实现 general.Trait 接口
//   - 在 init() 里调用 general.Register()
//   - 完全独立，不依赖其他特性
package traits

import (
	"math/rand"

	"hero3/internal/general"
)

// Meiren 美人计（甄宓）
//
// 战斗开始前触发：
// - 按 captureRate 比例俘虏敌方各兵种（随机）
// - 单种兵俘虏数量不超过 captureMax
// - 同阵营 / NPC：俘虏归我方军队
// - 跨阵营 PvP：俘虏进我方驻防
// - 剩余的兵正常进入战斗
type Meiren struct{}

func init() {
	general.Register(&Meiren{})
}

func (m *Meiren) ID() string   { return "meiren" }
func (m *Meiren) Name() string { return "美人计" }

func (m *Meiren) Description(p general.Params) string {
	return "战斗开始前，魅惑敌方部分残兵归己方所用"
}

func (m *Meiren) ParamSchema() []general.ParamField {
	return []general.ParamField{
		{Key: "captureRate", Label: "俘虏比例", Description: "俘虏敌方每种兵的比例（0.1 = 10%）", Default: 0.1, Min: 0, Max: 1, Step: 0.01},
		{Key: "captureMax", Label: "单兵种上限", Description: "单个兵种俘虏数量上限", Default: 1000, Min: 0, Max: 100000, Step: 100},
		{Key: "triggerChance", Label: "触发概率", Description: "1.0 = 必触发，0.5 = 50% 概率", Default: 1.0, Min: 0, Max: 1, Step: 0.05},
	}
}

func (m *Meiren) Subscribe() []general.EventSubscription {
	return []general.EventSubscription{
		{
			Event:    general.EventBeforeBattle,
			Priority: 100, // 美人计要在战斗前最早执行（俘虏后剩余的兵才进入战斗）
			Handle:   m.beforeBattle,
		},
	}
}

func (m *Meiren) beforeBattle(ctx general.EventContext, p general.Params) {
	c, ok := ctx.(*general.BeforeBattleContext)
	if !ok {
		return
	}
	// 只在进攻方拥有此特性时生效
	if !c.AttackerOwnsTrait {
		return
	}

	// 触发概率
	chance := p.FloatOr("triggerChance", 1.0)
	if chance < 1.0 && rand.Float64() > chance {
		return
	}

	rate := p.FloatOr("captureRate", 0.1)
	maxPerType := p.IntOr("captureMax", 1000)
	if rate <= 0 {
		return
	}

	// 决定俘虏归属
	var dest map[string]int
	if c.SameFaction || !c.IsPvP {
		// 同阵营或 NPC：进入军队
		if c.CapturedToArmy == nil {
			c.CapturedToArmy = map[string]int{}
		}
		dest = c.CapturedToArmy
	} else {
		// 跨阵营：进入驻防
		if c.CapturedToGarrison == nil {
			c.CapturedToGarrison = map[string]int{}
		}
		dest = c.CapturedToGarrison
	}

	// 从敌方按比例俘虏，并从 Defender.Units 中扣除
	totalCapturedAll := 0
	for i := range c.Defender.Units {
		unit := &c.Defender.Units[i]
		if unit.Count <= 0 {
			continue
		}
		captured := int(float64(unit.Count) * rate)
		if captured > maxPerType {
			captured = maxPerType
		}
		if captured <= 0 {
			continue
		}
		unit.Count -= captured
		dest[unit.ID] += captured
		totalCapturedAll += captured
	}

	// 实际俘虏了才记录触发
	if totalCapturedAll > 0 {
		if c.Triggered == nil {
			c.Triggered = map[string]general.TraitOutcome{}
		}
		c.Triggered["meiren"] = general.TraitOutcome{
			TraitID: "meiren",
			Name:    "美人计",
			Detail: map[string]interface{}{
				"totalCaptured": totalCapturedAll,
			},
		}
	}
}
