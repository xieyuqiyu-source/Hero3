package game

import "hero3/internal/core/combat"

// 本文件归口武将等级、经验、属性点和成长属性计算。

const (
	GeneralMaxLevel            = 100
	GeneralMaxStatPointsPerKey = 100
	GeneralLevelPercentAtMax   = 2.0
	GeneralStatPercentPerPoint = 0.02
	GeneralResetStatsGoldCost  = 10
	generalExpQuadraticFactor  = 50
	generalExpQuarticFactor    = 500
)

var generalStatKeys = []string{"force", "intelligence", "politics", "command"}

type generalExpResult struct {
	Gained      int
	LevelBefore int
	LevelAfter  int
}

// applyGeneralBattleExp 给武将发放战斗经验并刷新派生属性。
func applyGeneralBattleExp(g *General, gained int) generalExpResult {
	if g == nil {
		return generalExpResult{}
	}
	if gained <= 0 {
		applyHeroConfigToGeneral(g)
		return generalExpResult{}
	}

	if g.Level <= 0 {
		g.Level = 1
	}
	before := g.Level
	g.Exp += gained
	promoteGeneralByExp(g)
	applyHeroConfigToGeneral(g)

	return generalExpResult{
		Gained:      gained,
		LevelBefore: before,
		LevelAfter:  g.Level,
	}
}

// promoteGeneralByExp 按经验曲线提升武将等级。
func promoteGeneralByExp(g *General) {
	if g == nil {
		return
	}
	if g.Level <= 0 {
		g.Level = 1
	}
	for g.Level < GeneralMaxLevel && g.Exp >= generalExpRequiredForLevel(g.Level+1) {
		g.Level++
	}
	if g.Level > GeneralMaxLevel {
		g.Level = GeneralMaxLevel
	}
}

// nextGeneralLevelExp 返回下一级所需累计经验。
func nextGeneralLevelExp(level int) int {
	if level >= GeneralMaxLevel {
		return 0
	}
	return generalExpRequiredForLevel(level + 1)
}

// generalExpRequiredForLevel 返回指定等级所需累计经验。
func generalExpRequiredForLevel(level int) int {
	if level <= 1 {
		return 0
	}
	if level > GeneralMaxLevel {
		level = GeneralMaxLevel
	}
	cfg := GetGeneralsConfig()
	if idx := level - 1; idx >= 0 && idx < len(cfg.Common.ExpCurve) {
		return cfg.Common.ExpCurve[idx]
	}
	n := level - 1
	n2 := n * n
	n4 := n2 * n2
	return generalExpQuadraticFactor*n2 + generalExpQuarticFactor*n4
}

// calculateGeneralBattleExpFromLosses 根据击杀单位维护费计算武将战斗经验。
func calculateGeneralBattleExpFromLosses(faction string, losses []combat.UnitLoss) int {
	if len(losses) == 0 {
		return 0
	}
	total := 0
	for _, loss := range losses {
		if loss.Losses <= 0 {
			continue
		}
		upkeep := unitUpkeepForBattleExp(faction, loss.ID)
		if upkeep <= 0 {
			continue
		}
		total += loss.Losses * upkeep
	}
	return total
}

// unitUpkeepForBattleExp 获取战斗经验使用的兵种维护费，PVP 混编援军会回退到全阵营查找。
func unitUpkeepForBattleExp(preferredFaction string, unitType string) int {
	if unit, ok := GetUnitConfig(preferredFaction, unitType); ok {
		return unit.Stats["upkeep"]
	}
	for _, faction := range []string{"wei", "shu", "wu", "neutral"} {
		if unit, ok := GetUnitConfig(faction, unitType); ok {
			return unit.Stats["upkeep"]
		}
	}
	return 0
}

// applyGeneralBattleExpToRoster 给指定参战武将发放战斗经验，并同步当前主将。
func applyGeneralBattleExpToRoster(state *GameState, generalIDs []string, gained int) generalExpResult {
	if state == nil || gained <= 0 || len(generalIDs) == 0 {
		return generalExpResult{}
	}
	seen := map[string]bool{}
	first := generalExpResult{}
	activeID := ""
	if state.General != nil {
		activeID = state.General.ID
	}
	for _, generalID := range generalIDs {
		if generalID == "" || seen[generalID] {
			continue
		}
		seen[generalID] = true
		for index := range state.Generals {
			if state.Generals[index].ID != generalID {
				continue
			}
			result := applyGeneralBattleExp(&state.Generals[index], gained)
			if result.Gained > 0 && first.Gained == 0 {
				first = result
			}
			if activeID == generalID {
				state.General = cloneGeneralPtr(state.Generals[index])
			}
			break
		}
	}
	return first
}

// generalLevelAttributes 返回等级成长提供的通用属性。
func generalLevelAttributes(level int) map[string]float64 {
	if level <= 1 {
		return nil
	}
	if level > GeneralMaxLevel {
		level = GeneralMaxLevel
	}
	ratio := float64(level-1) / float64(GeneralMaxLevel-1)
	value := ratio * GeneralLevelPercentAtMax
	return map[string]float64{
		StatAttackBonus:  value,
		StatDefenseBonus: value,
	}
}

// normalizeGeneralStats 规范化武将属性点。
func normalizeGeneralStats(stats map[string]int) map[string]int {
	result := make(map[string]int, len(generalStatKeys))
	for _, key := range generalStatKeys {
		value := 0
		if stats != nil {
			value = stats[key]
		}
		if value < 0 {
			value = 0
		}
		if value > GeneralMaxStatPointsPerKey {
			value = GeneralMaxStatPointsPerKey
		}
		result[key] = value
	}
	return result
}

// availableGeneralStatPoints 计算当前等级剩余可分配属性点。
func availableGeneralStatPoints(level int, stats map[string]int) int {
	if level <= 0 {
		level = 1
	}
	if level > GeneralMaxLevel {
		level = GeneralMaxLevel
	}
	total := level
	for _, key := range generalStatKeys {
		total -= stats[key]
	}
	if total < 0 {
		return 0
	}
	return total
}

// generalStatAttributes 计算属性点提供的通用加成。
func generalStatAttributes(stats map[string]int) map[string]float64 {
	if len(stats) == 0 {
		return nil
	}

	attrs := map[string]float64{}
	addGeneralAttribute(attrs, StatAttackBonus, float64(stats["force"])*GeneralStatPercentPerPoint)
	addGeneralAttribute(attrs, StatRecruitSpeedBonus, float64(stats["intelligence"])*GeneralStatPercentPerPoint)
	addGeneralAttribute(attrs, StatMarchSpeedBonus, float64(stats["intelligence"])*GeneralStatPercentPerPoint)
	addGeneralAttribute(attrs, StatProductionBonus, float64(stats["politics"])*GeneralStatPercentPerPoint)
	addGeneralAttribute(attrs, StatCapacityBonus, float64(stats["politics"])*GeneralStatPercentPerPoint)
	addGeneralAttribute(attrs, StatDefenseBonus, float64(stats["command"])*GeneralStatPercentPerPoint)
	return attrs
}

// generalLevelAttributesForTest 暴露等级成长计算给测试使用。
func generalLevelAttributesForTest(level int) map[string]float64 {
	return generalLevelAttributes(level)
}

// generalExpRequiredForLevelForTest 暴露经验曲线计算给测试使用。
func generalExpRequiredForLevelForTest(level int) int {
	return generalExpRequiredForLevel(level)
}
