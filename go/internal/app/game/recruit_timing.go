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
	default:
		return baseSeconds
	}
}
