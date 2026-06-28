package game

import (
	"sort"
	"strconv"
	"strings"
	"time"
)

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

// EnsureCoreBuildings 确保玩家拥有当前版本要求的核心建筑。
func EnsureCoreBuildings(state *GameState) bool {
	return ensureCoreBuildings(state)
}

// BuildResourceSlotsFromBuildings 根据资源建筑生成玩家资源田格子快照。
func BuildResourceSlotsFromBuildings(buildings []Building, now time.Time) []ResourceSlot {
	slots := make([]ResourceSlot, 0, len(buildings))
	unlockedAt := now.UTC().Format(resourceDateLayout)
	for _, building := range buildings {
		building.ID = strings.TrimSpace(building.ID)
		building.Type = strings.TrimSpace(building.Type)
		if building.ID == "" || building.Type == "" {
			continue
		}
		config, exists := GetBuildingConfig(building.Type)
		if !exists || strings.TrimSpace(config.ResourceType) == "" {
			continue
		}
		slots = append(slots, ResourceSlot{
			ID:           building.ID,
			ResourceType: strings.TrimSpace(config.ResourceType),
			BuildingID:   building.ID,
			UnlockedBy:   "initial_building",
			UnlockedAt:   unlockedAt,
		})
	}
	sort.Slice(slots, func(i, j int) bool {
		return slots[i].ID < slots[j].ID
	})
	return slots
}

// ensureCoreBuildings 确保老存档也拥有核心建筑。
func ensureCoreBuildings(state *GameState) bool {
	if state == nil {
		return false
	}
	required := []Building{
		{ID: "infantry_camp-1", Type: "infantry_camp", Level: 1},
		{ID: "cavalry_camp-1", Type: "cavalry_camp", Level: 1},
		{ID: "siege_camp-1", Type: "siege_camp", Level: 1},
		{ID: "special_camp-1", Type: "special_camp", Level: 1},
		{ID: "weapon_bureau-1", Type: "weapon_bureau", Level: 1},
		{ID: "armor_bureau-1", Type: "armor_bureau", Level: 1},
		{ID: "construction_bureau-1", Type: "construction_bureau", Level: 1},
		{ID: "administration-1", Type: "administration", Level: 1},
		{ID: "relay_station-1", Type: "relay_station", Level: 1},
		{ID: "city_wall-1", Type: "city_wall", Level: 1},
		{ID: "beacon_tower-1", Type: "beacon_tower", Level: 1},
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

// ApplyConstructionBureauResourceSlots 根据建造司等级补齐额外资源田建筑和格子。
func ApplyConstructionBureauResourceSlots(state *GameState, now time.Time) bool {
	if state == nil {
		return false
	}
	unlockedRounds := constructionBureauUnlockedSlotRoundCount(state.Buildings)
	types := constructionBureauResourceSlotTypes(unlockedRounds)
	changed := pruneConstructionBureauResourceSlots(state, len(types))
	for i := 0; i < len(types); i++ {
		buildingID := constructionBureauResourceSlotID(i + 1)
		if findBuildingByID(state, buildingID) != nil {
			continue
		}
		state.Buildings = append(state.Buildings, Building{
			ID:    buildingID,
			Type:  types[i],
			Level: 1,
		})
		changed = true
	}
	return EnsureResourceSlotsForBuildings(state, now) || changed
}

// EnsureResourceSlotsForBuildings 为资源建筑补齐资源田格子，同时保留已有格子的时间戳和来源。
func EnsureResourceSlotsForBuildings(state *GameState, now time.Time) bool {
	if state == nil {
		return false
	}
	if len(state.ResourceSlots) == 0 {
		slots := BuildResourceSlotsFromBuildings(state.Buildings, now)
		if len(slots) == 0 {
			return false
		}
		state.ResourceSlots = slots
		return true
	}
	existing := map[string]struct{}{}
	for _, slot := range state.ResourceSlots {
		existing[strings.TrimSpace(slot.ID)] = struct{}{}
	}
	changed := false
	unlockedAt := now.UTC().Format(resourceDateLayout)
	for _, building := range state.Buildings {
		building.ID = strings.TrimSpace(building.ID)
		building.Type = strings.TrimSpace(building.Type)
		if building.ID == "" || building.Type == "" {
			continue
		}
		if _, ok := existing[building.ID]; ok {
			continue
		}
		config, exists := GetBuildingConfig(building.Type)
		if !exists || strings.TrimSpace(config.ResourceType) == "" {
			continue
		}
		state.ResourceSlots = append(state.ResourceSlots, ResourceSlot{
			ID:           building.ID,
			ResourceType: strings.TrimSpace(config.ResourceType),
			BuildingID:   building.ID,
			UnlockedBy:   "initial_building",
			UnlockedAt:   unlockedAt,
		})
		existing[building.ID] = struct{}{}
		changed = true
	}
	if changed {
		sort.Slice(state.ResourceSlots, func(i, j int) bool {
			return state.ResourceSlots[i].ID < state.ResourceSlots[j].ID
		})
	}
	return changed
}

// constructionBureauUnlockedSlotRoundCount 返回建造司等级已解锁的资源田批次数。
func constructionBureauUnlockedSlotRoundCount(buildings []Building) int {
	level := 0
	for _, building := range buildings {
		if building.Type == "construction_bureau" && building.Level > level {
			level = building.Level
		}
	}
	thresholds := []int{5, 10, 15, 20, 25}
	count := 0
	for _, threshold := range thresholds {
		if level >= threshold {
			count++
		}
	}
	return count
}

// constructionBureauResourceSlotTypes 返回建造司按批次解锁的资源田类型。
func constructionBureauResourceSlotTypes(rounds int) []string {
	if rounds <= 0 {
		return nil
	}
	resourceTypes := []string{"wood_camp", "stone_quarry", "iron_mine", "farm"}
	types := make([]string, 0, rounds*len(resourceTypes))
	for i := 0; i < rounds; i++ {
		types = append(types, resourceTypes...)
	}
	return types
}

// pruneConstructionBureauResourceSlots 移除当前建造司等级尚未解锁的额外资源田。
func pruneConstructionBureauResourceSlots(state *GameState, allowedCount int) bool {
	changed := false
	nextBuildings := state.Buildings[:0]
	for _, building := range state.Buildings {
		index := constructionBureauResourceSlotIndex(building.ID)
		if index > 0 && index > allowedCount {
			changed = true
			continue
		}
		nextBuildings = append(nextBuildings, building)
	}
	state.Buildings = nextBuildings

	nextSlots := state.ResourceSlots[:0]
	for _, slot := range state.ResourceSlots {
		slotIndex := constructionBureauResourceSlotIndex(slot.ID)
		buildingIndex := constructionBureauResourceSlotIndex(slot.BuildingID)
		if (slotIndex > 0 && slotIndex > allowedCount) || (buildingIndex > 0 && buildingIndex > allowedCount) {
			changed = true
			continue
		}
		nextSlots = append(nextSlots, slot)
	}
	state.ResourceSlots = nextSlots
	return changed
}

// constructionBureauResourceSlotID 返回建造司解锁资源田对应的稳定建筑 ID。
func constructionBureauResourceSlotID(index int) string {
	return "construction_resource_slot-" + strconv.Itoa(index)
}

// constructionBureauResourceSlotIndex 解析建造司资源田稳定 ID 对应的序号。
func constructionBureauResourceSlotIndex(id string) int {
	id = strings.TrimSpace(id)
	const prefix = "construction_resource_slot-"
	if !strings.HasPrefix(id, prefix) {
		return 0
	}
	index, err := strconv.Atoi(strings.TrimPrefix(id, prefix))
	if err != nil || index <= 0 {
		return 0
	}
	return index
}
