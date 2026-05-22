package game

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
)

type BuildingConfig struct {
	Type                  string              `json:"type"`
	Name                  string              `json:"name"`
	ResourceType          string              `json:"resourceType,omitempty"`
	ProductionByLevel     []int               `json:"productionByLevel,omitempty"`
	CapacityByLevel       []int               `json:"capacityByLevel,omitempty"`
	UpgradeCostByLevel    map[int]ResourceMap `json:"upgradeCostByLevel,omitempty"`
	UpgradeSecondsByLevel map[int]int         `json:"upgradeSecondsByLevel,omitempty"`
}

type BalanceConfig struct {
	BaseProduction ResourceMap               `json:"baseProduction"`
	Buildings      map[string]BuildingConfig `json:"buildings"`
}

type ResourceMap map[string]int

var (
	balanceMu     sync.RWMutex
	activeBalance = cloneBalance(defaultBalance)
)

var defaultBalance = BalanceConfig{
	BaseProduction: ResourceMap{
		"wood":  30,
		"stone": 30,
		"iron":  20,
		"food":  40,
	},
	Buildings: map[string]BuildingConfig{
		"wood_camp": {
			Type:              "wood_camp",
			Name:              "伐木场",
			ResourceType:      "wood",
			ProductionByLevel: []int{0, 18, 36, 54, 72, 96, 126, 162, 204, 252, 306},
			UpgradeCostByLevel: map[int]ResourceMap{
				1: {"wood": 50, "stone": 80, "iron": 30, "food": 20},
				2: {"wood": 70, "stone": 110, "iron": 45, "food": 30},
				3: {"wood": 95, "stone": 150, "iron": 65, "food": 45},
			},
			UpgradeSecondsByLevel: map[int]int{1: 30, 2: 45, 3: 70},
		},
		"stone_quarry": {
			Type:              "stone_quarry",
			Name:              "采石场",
			ResourceType:      "stone",
			ProductionByLevel: []int{0, 16, 32, 48, 64, 86, 114, 148, 188, 234, 286},
			UpgradeCostByLevel: map[int]ResourceMap{
				1: {"wood": 80, "stone": 50, "iron": 35, "food": 20},
				2: {"wood": 110, "stone": 70, "iron": 50, "food": 30},
				3: {"wood": 150, "stone": 95, "iron": 70, "food": 45},
			},
			UpgradeSecondsByLevel: map[int]int{1: 30, 2: 45, 3: 70},
		},
		"iron_mine": {
			Type:              "iron_mine",
			Name:              "铁矿",
			ResourceType:      "iron",
			ProductionByLevel: []int{0, 14, 28, 42, 56, 76, 102, 132, 168, 210, 258},
			UpgradeCostByLevel: map[int]ResourceMap{
				1: {"wood": 75, "stone": 70, "iron": 45, "food": 25},
				2: {"wood": 105, "stone": 100, "iron": 65, "food": 35},
				3: {"wood": 145, "stone": 140, "iron": 90, "food": 50},
			},
			UpgradeSecondsByLevel: map[int]int{1: 35, 2: 55, 3: 85},
		},
		"farm": {
			Type:              "farm",
			Name:              "农田",
			ResourceType:      "food",
			ProductionByLevel: []int{0, 20, 40, 60, 80, 108, 142, 182, 228, 282, 342},
			UpgradeCostByLevel: map[int]ResourceMap{
				1: {"wood": 45, "stone": 55, "iron": 25, "food": 35},
				2: {"wood": 65, "stone": 80, "iron": 40, "food": 50},
				3: {"wood": 90, "stone": 110, "iron": 55, "food": 70},
			},
			UpgradeSecondsByLevel: map[int]int{1: 25, 2: 40, 3: 65},
		},
		"warehouse": {
			Type:            "warehouse",
			Name:            "仓库",
			CapacityByLevel: []int{5000, 7500, 10000, 13000, 16500, 20500, 25000, 30000, 35500, 41500, 48000},
			UpgradeCostByLevel: map[int]ResourceMap{
				1: {"wood": 120, "stone": 100, "iron": 80, "food": 40},
				2: {"wood": 180, "stone": 150, "iron": 120, "food": 60},
				3: {"wood": 260, "stone": 220, "iron": 175, "food": 90},
			},
			UpgradeSecondsByLevel: map[int]int{1: 45, 2: 75, 3: 120},
		},
	},
}

func getBuildingConfig(buildingType string) (BuildingConfig, bool) {
	config, exists := currentBalance().Buildings[buildingType]
	return config, exists
}

func GetBalanceConfig() BalanceConfig {
	return currentBalance()
}

func SetBalanceConfig(config BalanceConfig) error {
	if err := validateBalance(config); err != nil {
		return err
	}

	balanceMu.Lock()
	activeBalance = cloneBalance(config)
	balanceMu.Unlock()
	return nil
}

func LoadBalanceConfig(path string) error {
	if path == "" {
		return nil
	}

	content, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return SaveBalanceConfig(path, currentBalance())
		}
		return err
	}

	var config BalanceConfig
	if err := json.Unmarshal(content, &config); err != nil {
		return err
	}
	return SetBalanceConfig(config)
}

func SaveBalanceConfig(path string, config BalanceConfig) error {
	if err := validateBalance(config); err != nil {
		return err
	}
	if path == "" {
		return nil
	}

	content, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, append(content, '\n'), 0o644)
}

func currentBalance() BalanceConfig {
	balanceMu.RLock()
	defer balanceMu.RUnlock()
	return cloneBalance(activeBalance)
}

func validateBalance(config BalanceConfig) error {
	if len(config.BaseProduction) == 0 {
		return errors.New("baseProduction is required")
	}
	if len(config.Buildings) == 0 {
		return errors.New("buildings is required")
	}
	if _, exists := config.Buildings["warehouse"]; !exists {
		return errors.New("warehouse config is required")
	}
	return nil
}

func cloneBalance(source BalanceConfig) BalanceConfig {
	next := BalanceConfig{
		BaseProduction: cloneResourceMap(source.BaseProduction),
		Buildings:      make(map[string]BuildingConfig, len(source.Buildings)),
	}
	for key, building := range source.Buildings {
		next.Buildings[key] = cloneBuildingConfig(building)
	}
	return next
}

func cloneBuildingConfig(source BuildingConfig) BuildingConfig {
	next := BuildingConfig{
		Type:                  source.Type,
		Name:                  source.Name,
		ResourceType:          source.ResourceType,
		ProductionByLevel:     append([]int(nil), source.ProductionByLevel...),
		CapacityByLevel:       append([]int(nil), source.CapacityByLevel...),
		UpgradeCostByLevel:    make(map[int]ResourceMap, len(source.UpgradeCostByLevel)),
		UpgradeSecondsByLevel: make(map[int]int, len(source.UpgradeSecondsByLevel)),
	}
	for level, cost := range source.UpgradeCostByLevel {
		next.UpgradeCostByLevel[level] = cloneResourceMap(cost)
	}
	for level, seconds := range source.UpgradeSecondsByLevel {
		next.UpgradeSecondsByLevel[level] = seconds
	}
	return next
}

func cloneResourceMap(source ResourceMap) ResourceMap {
	next := make(ResourceMap, len(source))
	for key, value := range source {
		next[key] = value
	}
	return next
}

func valueByLevel(values []int, level int) int {
	if len(values) == 0 {
		return 0
	}
	if level <= 0 {
		return values[0]
	}
	if level < len(values) {
		return values[level]
	}
	return values[len(values)-1]
}
