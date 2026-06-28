package game

import "time"

// 本文件归口征兵时长和征兵速度加成计算。

// calculateRecruitDurationSeconds 计算指定兵种数量的实际征兵时长。
func calculateRecruitDurationSeconds(unitConfig UnitConfig, amount int, now time.Time, sources []ModifierSource) int {
	totalSeconds := unitConfig.TrainSeconds * amount
	totalSeconds = applySpeedBonus(totalSeconds, StatRecruitSpeedBonus, now, sources)
	return applyCategoryRecruitSpeedBonus(totalSeconds, unitConfig.Category, now, sources)
}

// applyCategoryRecruitSpeedBonus 应用兵种分类征兵速度加成。
func applyCategoryRecruitSpeedBonus(baseSeconds int, category string, now time.Time, sources []ModifierSource) int {
	switch category {
	case "infantry":
		return applySpeedBonus(baseSeconds, StatInfantryRecruitSpeedBonus, now, sources)
	case "cavalry":
		return applySpeedBonus(baseSeconds, StatCavalryRecruitSpeedBonus, now, sources)
	case "siege":
		return applySpeedBonus(baseSeconds, StatSiegeRecruitSpeedBonus, now, sources)
	case "special":
		return applySpeedBonus(baseSeconds, StatSpecialRecruitSpeedBonus, now, sources)
	default:
		return baseSeconds
	}
}

// calculateRecruitCost 计算实际征兵消耗，统一应用征兵消耗减免。
func calculateRecruitCost(unitConfig UnitConfig, amount int, now time.Time, sources []ModifierSource) ResourceMap {
	factor := recruitCostFactor(unitConfig.Category, now, sources)
	totalCost := make(ResourceMap, len(unitConfig.Cost))
	for resType, costPer := range unitConfig.Cost {
		baseCost := costPer * amount
		if baseCost <= 0 {
			totalCost[resType] = 0
			continue
		}
		reduced := int(float64(baseCost) * factor)
		if reduced < 1 {
			reduced = 1
		}
		totalCost[resType] = reduced
	}
	return totalCost
}

// recruitCostFactor 返回征兵成本折扣系数，最高减免 80%，避免成本被堆到 0。
func recruitCostFactor(category string, now time.Time, sources []ModifierSource) float64 {
	reduction := ComputeAttributeAt(0, StatRecruitCostReduction, now, sources...)
	switch category {
	case "siege":
		reduction += ComputeAttributeAt(0, StatSiegeRecruitCostReduction, now, sources...)
	case "special":
		reduction += ComputeAttributeAt(0, StatSpecialRecruitCostReduction, now, sources...)
	}
	if reduction < 0 {
		reduction = 0
	}
	if reduction > 0.8 {
		reduction = 0.8
	}
	return 1 - reduction
}
