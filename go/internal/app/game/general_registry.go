package game

// 本文件归口武将配置注册表查询入口。

// GetGeneralHeroConfig 根据武将 ID 获取武将配置。
func GetGeneralHeroConfig(generalID string) (GeneralHeroConfig, bool) {
	generalsMu.RLock()
	defer generalsMu.RUnlock()
	hero, ok := activeGenerals.Heroes[generalID]
	if !ok {
		return GeneralHeroConfig{}, false
	}
	return cloneHeroConfig(hero), true
}

// ListGeneralHeroConfigs 返回当前启用配置中的全部武将配置。
func ListGeneralHeroConfigs() map[string]GeneralHeroConfig {
	generalsMu.RLock()
	defer generalsMu.RUnlock()
	next := make(map[string]GeneralHeroConfig, len(activeGenerals.Heroes))
	for id, hero := range activeGenerals.Heroes {
		next[id] = cloneHeroConfig(hero)
	}
	return next
}

// GeneralRegistered 判断武将配置是否已经注册。
func GeneralRegistered(generalID string) bool {
	_, exists := GetGeneralHeroConfig(generalID)
	return exists
}

// GetHeroConfig 保留旧入口，统一委托到武将注册表。
func GetHeroConfig(generalID string) (GeneralHeroConfig, bool) {
	return GetGeneralHeroConfig(generalID)
}
