package game

import "strings"

// 本文件归口兵种配置注册表查询入口。

// GetUnitsConfig 获取当前全部兵种配置。
func GetUnitsConfig() UnitsConfig {
	unitsMu.RLock()
	defer unitsMu.RUnlock()
	return cloneUnitsConfig(activeUnits)
}

// GetFactionUnits 获取指定阵营的兵种配置。
func GetFactionUnits(faction string) FactionUnits {
	unitsMu.RLock()
	defer unitsMu.RUnlock()
	return cloneFactionUnits(activeUnits[faction])
}

// GetUnitConfig 获取指定阵营的指定兵种配置。
func GetUnitConfig(faction string, unitID string) (UnitConfig, bool) {
	unitsMu.RLock()
	defer unitsMu.RUnlock()
	config, exists := activeUnits[faction][unitID]
	if !exists {
		return UnitConfig{}, false
	}
	return cloneUnitConfig(config), true
}

// UnitRegistered 判断指定阵营兵种是否已经注册。
func UnitRegistered(faction string, unitID string) bool {
	_, exists := GetUnitConfig(faction, unitID)
	return exists
}

// FindFactionUnitByName 根据显示名称查找阵营兵种。
func FindFactionUnitByName(faction string, unitName string) (string, UnitConfig, bool) {
	unitName = strings.TrimSpace(unitName)
	if unitName == "" {
		return "", UnitConfig{}, false
	}
	if alias, ok := unitNameAliases[unitName]; ok {
		unitName = alias
	}
	unitsMu.RLock()
	defer unitsMu.RUnlock()
	for unitID, config := range activeUnits[faction] {
		if strings.TrimSpace(config.Name) == unitName {
			return unitID, cloneUnitConfig(config), true
		}
	}
	return "", UnitConfig{}, false
}

// FindAnyFactionUnitByName 根据显示名称在全部阵营中查找兵种。
func FindAnyFactionUnitByName(unitName string) (string, string, UnitConfig, bool) {
	unitName = strings.TrimSpace(unitName)
	if unitName == "" {
		return "", "", UnitConfig{}, false
	}
	if alias, ok := unitNameAliases[unitName]; ok {
		unitName = alias
	}
	unitsMu.RLock()
	defer unitsMu.RUnlock()
	for faction, factionUnits := range activeUnits {
		for unitID, config := range factionUnits {
			if strings.TrimSpace(config.Name) == unitName {
				return faction, unitID, cloneUnitConfig(config), true
			}
		}
	}
	return "", "", UnitConfig{}, false
}

// cloneUnitsConfig 复制全部兵种配置，避免调用方修改全局配置。
func cloneUnitsConfig(source UnitsConfig) UnitsConfig {
	if source == nil {
		return nil
	}
	next := make(UnitsConfig, len(source))
	for faction, units := range source {
		next[faction] = cloneFactionUnits(units)
	}
	return next
}

// cloneFactionUnits 复制单阵营兵种配置。
func cloneFactionUnits(source FactionUnits) FactionUnits {
	if source == nil {
		return nil
	}
	next := make(FactionUnits, len(source))
	for unitID, config := range source {
		next[unitID] = cloneUnitConfig(config)
	}
	return next
}

// cloneUnitConfig 复制单个兵种配置。
func cloneUnitConfig(source UnitConfig) UnitConfig {
	next := source
	next.Stats = cloneIntStringMap(source.Stats)
	next.Cost = cloneIntStringMap(source.Cost)
	if source.Unlock != nil {
		next.Unlock = make(map[string]any, len(source.Unlock))
		for key, value := range source.Unlock {
			next.Unlock[key] = value
		}
	}
	return next
}

// cloneIntStringMap 复制 string 到 int 的配置表。
func cloneIntStringMap(source map[string]int) map[string]int {
	if source == nil {
		return nil
	}
	next := make(map[string]int, len(source))
	for key, value := range source {
		next[key] = value
	}
	return next
}
