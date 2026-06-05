package traits

import (
	"math"
	"math/rand"

	"hero3/internal/combat"
	"hero3/internal/general"
)

// WeizhenXiaoyao 威震逍遥（张辽）
//
// 进攻方以少打多时触发：
// - 按敌我口粮比提升触发概率
// - 触发后让防守方部分兵力本场不参与防御计算
// - 被震慑的兵不死亡，战后仍保留
type WeizhenXiaoyao struct{}

func init() {
	general.Register(&WeizhenXiaoyao{})
}

func (w *WeizhenXiaoyao) ID() string   { return "weizhenxiaoyao" }
func (w *WeizhenXiaoyao) Name() string { return "威震逍遥" }

func (w *WeizhenXiaoyao) Description(p general.Params) string {
	return "以少打多时概率震慑敌军，使部分守军不参与本场防御"
}

func (w *WeizhenXiaoyao) ParamSchema() []general.ParamField {
	return []general.ParamField{
		{Key: "baseChance", Label: "基础触发概率", Description: "满足以少打多时的基础触发概率", Default: 0.08, Min: 0, Max: 1, Step: 0.01},
		{Key: "chancePerRatio", Label: "每倍差距概率", Description: "敌我口粮比每高 1 倍增加的触发概率", Default: 0.04, Min: 0, Max: 1, Step: 0.01},
		{Key: "maxChance", Label: "最高触发概率", Description: "触发概率上限", Default: 0.35, Min: 0, Max: 1, Step: 0.01},
		{Key: "baseSuppressRate", Label: "基础震慑比例", Description: "触发后的基础震慑比例", Default: 0.08, Min: 0, Max: 1, Step: 0.01},
		{Key: "suppressPerRatio", Label: "每倍差距震慑", Description: "敌我口粮比每高 1 倍增加的震慑比例", Default: 0.02, Min: 0, Max: 1, Step: 0.01},
		{Key: "maxSuppressRate", Label: "最高震慑比例", Description: "震慑比例上限", Default: 0.20, Min: 0, Max: 1, Step: 0.01},
	}
}

func (w *WeizhenXiaoyao) Subscribe() []general.EventSubscription {
	return []general.EventSubscription{
		{
			Event:    general.EventBeforeBattle,
			Priority: 80,
			Handle:   w.beforeBattle,
		},
	}
}

func (w *WeizhenXiaoyao) beforeBattle(ctx general.EventContext, p general.Params) {
	c, ok := ctx.(*general.BeforeBattleContext)
	if !ok {
		return
	}
	if !c.AttackerOwnsTrait || c.Attacker == nil || c.Defender == nil {
		return
	}

	attackerFood := armyFood(c.Attacker.Units)
	defenderFood := armyFood(c.Defender.Units)
	if attackerFood <= 0 || defenderFood <= attackerFood {
		return
	}

	ratio := defenderFood / attackerFood
	chance := bounded(p.FloatOr("baseChance", 0.08), 0, 1) +
		(ratio-1)*bounded(p.FloatOr("chancePerRatio", 0.04), 0, 1)
	chance = minFloat(chance, bounded(p.FloatOr("maxChance", 0.35), 0, 1))
	if chance <= 0 || rand.Float64() > chance {
		return
	}

	suppressRate := bounded(p.FloatOr("baseSuppressRate", 0.08), 0, 1) +
		(ratio-1)*bounded(p.FloatOr("suppressPerRatio", 0.02), 0, 1)
	suppressRate = minFloat(suppressRate, bounded(p.FloatOr("maxSuppressRate", 0.20), 0, 1))
	if suppressRate <= 0 {
		return
	}

	suppressed := suppressDefenderUnits(c.Defender, suppressRate)
	totalSuppressed := 0
	for _, count := range suppressed {
		totalSuppressed += count
	}
	if totalSuppressed <= 0 {
		return
	}

	if c.Triggered == nil {
		c.Triggered = map[string]general.TraitOutcome{}
	}
	c.Triggered["weizhenxiaoyao"] = general.TraitOutcome{
		TraitID: "weizhenxiaoyao",
		Name:    "威震逍遥",
		Detail: map[string]interface{}{
			"foodRatio":       roundFloat(ratio, 2),
			"triggerChance":   chance,
			"suppressRate":    suppressRate,
			"totalSuppressed": totalSuppressed,
			"suppressedUnits": suppressed,
		},
	}
}

func armyFood(units []combat.Unit) float64 {
	var total float64
	for _, unit := range units {
		if unit.Count <= 0 {
			continue
		}
		upkeep := unit.Upkeep
		if upkeep < 0 {
			upkeep = 0
		}
		total += float64(unit.Count * upkeep)
	}
	return total
}

func suppressDefenderUnits(army *combat.Army, rate float64) map[string]int {
	suppressed := map[string]int{}
	totalDefenders := 0
	for _, unit := range army.Units {
		if unit.Count > 0 {
			totalDefenders += unit.Count
		}
	}
	totalToSuppress := int(math.Floor(float64(totalDefenders) * rate))
	if totalToSuppress <= 0 {
		return suppressed
	}

	type fractionEntry struct {
		index    int
		fraction float64
	}
	fractions := make([]fractionEntry, 0, len(army.Units))
	remaining := totalToSuppress

	for i := range army.Units {
		unit := &army.Units[i]
		if unit.Count <= 0 {
			continue
		}
		exact := float64(unit.Count) * rate
		count := int(math.Floor(exact))
		if count > unit.Count {
			count = unit.Count
		}
		if count > remaining {
			count = remaining
		}
		fractions = append(fractions, fractionEntry{
			index:    i,
			fraction: exact - math.Floor(exact),
		})
		if count <= 0 {
			continue
		}
		unit.Count -= count
		suppressed[unit.ID] += count
		remaining -= count
	}

	for i := 0; i < len(fractions)-1; i++ {
		for j := i + 1; j < len(fractions); j++ {
			if fractions[j].fraction > fractions[i].fraction {
				fractions[i], fractions[j] = fractions[j], fractions[i]
			}
		}
	}
	for _, item := range fractions {
		if remaining <= 0 {
			break
		}
		unit := &army.Units[item.index]
		if unit.Count <= 0 {
			continue
		}
		unit.Count--
		suppressed[unit.ID]++
		remaining--
	}
	return suppressed
}

func bounded(value, min, max float64) float64 {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

func minFloat(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

func roundFloat(value float64, digits int) float64 {
	factor := math.Pow(10, float64(digits))
	return math.Round(value*factor) / factor
}
