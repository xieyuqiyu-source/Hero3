package game

// 本文件归口建筑配置注册表与玩家默认核心建筑补齐逻辑。

// GetBuildingConfig 获取指定建筑类型的配置。
func GetBuildingConfig(buildingType string) (BuildingConfig, bool) {
	config, exists := currentBalance().Buildings[buildingType]
	return config, exists
}

// ListBuildingConfigs 返回当前启用的全部建筑配置。
func ListBuildingConfigs() map[string]BuildingConfig {
	buildings := currentBalance().Buildings
	next := make(map[string]BuildingConfig, len(buildings))
	for key, config := range buildings {
		next[key] = cloneBuildingConfig(config)
	}
	return next
}

// BuildingTypeRegistered 判断建筑类型是否已经注册到当前配置。
func BuildingTypeRegistered(buildingType string) bool {
	_, exists := GetBuildingConfig(buildingType)
	return exists
}

// getBuildingConfig 保留内部旧入口，统一委托到建筑注册表。
func getBuildingConfig(buildingType string) (BuildingConfig, bool) {
	return GetBuildingConfig(buildingType)
}

// ensureCoreBuildings 确保老存档也拥有核心军事建筑。
func ensureCoreBuildings(state *GameState) bool {
	if state == nil {
		return false
	}
	required := []Building{
		{ID: "infantry_camp-1", Type: "infantry_camp", Level: 1},
		{ID: "cavalry_camp-1", Type: "cavalry_camp", Level: 1},
		{ID: "weapon_bureau-1", Type: "weapon_bureau", Level: 1},
		{ID: "armor_bureau-1", Type: "armor_bureau", Level: 1},
	}
	changed := false
	for _, requiredBuilding := range required {
		exists := false
		for _, building := range state.Buildings {
			if building.Type == requiredBuilding.Type {
				exists = true
				break
			}
		}
		if !exists {
			state.Buildings = append(state.Buildings, requiredBuilding)
			changed = true
		}
	}
	return changed
}
