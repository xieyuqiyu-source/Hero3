package game

import "time"

// 本文件归口建筑对资源、容量与 Modifier 管线产生的效果。

// calculateResourceProduction 按当前建筑状态计算基础资源产量。
func calculateResourceProduction(buildings []Building, general *General) ResourceProduction {
	production := ResourceProduction{}
	balance := currentBalance()
	for resourceType, value := range balance.BaseProduction {
		production[resourceType] = value
	}

	for _, building := range buildings {
		if !buildingIsOperational(building) {
			continue
		}
		config, exists := getBuildingConfig(building.Type)
		if !exists || config.ResourceType == "" {
			continue
		}
		production[config.ResourceType] += valueByLevel(config.ProductionByLevel, building.Level)
	}

	// 注意：将领加成不在这里应用，统一由 applyProductionModifiers 通过 Modifier 管线处理。
	return production
}

// calculateResourceCapacity 按当前建筑状态计算资源容量。
func calculateResourceCapacity(buildings []Building) map[string]int {
	balance := currentBalance()
	capacity := valueByLevel(balance.Buildings["warehouse"].CapacityByLevel, 0)
	for _, building := range buildings {
		if !buildingIsOperational(building) {
			continue
		}
		config, exists := getBuildingConfig(building.Type)
		if !exists || len(config.CapacityByLevel) == 0 {
			continue
		}
		capacity = valueByLevel(config.CapacityByLevel, building.Level)
	}
	return map[string]int{
		"wood":  capacity,
		"stone": capacity,
		"iron":  capacity,
		"food":  capacity,
	}
}

// BuildingBonusSource 功能建筑加成来源。
type BuildingBonusSource struct {
	Buildings []Building
}

// SourceName 返回建筑加成来源名称。
func (b *BuildingBonusSource) SourceName() string { return "军事建筑" }

// ExpiresAt 返回建筑加成过期时间，建筑常驻效果没有单独过期时间。
func (b *BuildingBonusSource) ExpiresAt() []time.Time { return nil }

// Modifiers 返回当前生效建筑提供的 Modifier。
func (b *BuildingBonusSource) Modifiers(now time.Time) []Modifier {
	mods := make([]Modifier, 0, len(b.Buildings))
	for _, building := range b.Buildings {
		if !buildingIsOperational(building) {
			continue
		}
		if building.Level <= 0 {
			continue
		}
		config, exists := getBuildingConfig(building.Type)
		if !exists {
			continue
		}
		for _, mod := range config.ModifiersByLevel[building.Level] {
			if mod.Value == 0 {
				continue
			}
			mods = append(mods, mod)
		}
	}
	return mods
}
