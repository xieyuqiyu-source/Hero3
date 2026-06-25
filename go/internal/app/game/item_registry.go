package game

import "strings"

// 本文件归口道具配置注册表查询入口。

// GetItemsConfig 获取当前全部道具配置。
func GetItemsConfig() ItemsConfig {
	itemsMu.RLock()
	defer itemsMu.RUnlock()
	return cloneItemsConfig(itemsConfig)
}

// GetItemDefinition 获取指定道具定义。
func GetItemDefinition(itemID string) (ItemDefinition, bool) {
	itemID = strings.TrimSpace(itemID)
	itemsMu.RLock()
	defer itemsMu.RUnlock()
	item, ok := itemsConfig[itemID]
	if !ok {
		return ItemDefinition{}, false
	}
	return cloneItemDefinition(item), true
}

// ItemRegistered 判断道具是否已注册。
func ItemRegistered(itemID string) bool {
	_, exists := GetItemDefinition(itemID)
	return exists
}

// cloneItemsConfig 复制全部道具配置。
func cloneItemsConfig(source ItemsConfig) ItemsConfig {
	if source == nil {
		return nil
	}
	next := make(ItemsConfig, len(source))
	for id, item := range source {
		next[id] = cloneItemDefinition(item)
	}
	return next
}

// cloneItemDefinition 复制单个道具定义。
func cloneItemDefinition(source ItemDefinition) ItemDefinition {
	next := source
	next.Effects = make([]ItemEffect, len(source.Effects))
	for i, effect := range source.Effects {
		next.Effects[i] = cloneItemEffect(effect)
	}
	if source.Metadata != nil {
		next.Metadata = make(map[string]interface{}, len(source.Metadata))
		for key, value := range source.Metadata {
			next.Metadata[key] = value
		}
	}
	return next
}

// cloneItemEffect 复制单个道具效果定义。
func cloneItemEffect(source ItemEffect) ItemEffect {
	next := source
	next.Resources = cloneIntStringMap(source.Resources)
	if source.UnitByFaction != nil {
		next.UnitByFaction = make(map[string]string, len(source.UnitByFaction))
		for key, value := range source.UnitByFaction {
			next.UnitByFaction[key] = value
		}
	}
	return next
}
