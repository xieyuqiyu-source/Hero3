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
	BaseProduction        ResourceMap               `json:"baseProduction"`
	Buildings             map[string]BuildingConfig `json:"buildings"`
	OverflowToCityGold    int                       `json:"overflowToCityGold"`    // 多少溢出资源兑换 1 城金（默认 200）
	ExchangeRate          int                       `json:"exchangeRate"`          // 1 金币 = N 城金（默认 10）
	ReverseExchangeRate   int                       `json:"reverseExchangeRate"`   // N 城金 = 1 金币（默认 15，有损耗）
	ExchangeCooldownSecs  int                       `json:"exchangeCooldownSecs"`  // 兑换冷却秒数（默认 3600）
}

type ResourceMap map[string]int

var (
	balanceMu     sync.RWMutex
	activeBalance = cloneBalance(defaultBalance)
)

var defaultBalance = BalanceConfig{
	BaseProduction: ResourceMap{
		"wood":  0,
		"stone": 0,
		"iron":  0,
		"food":  0,
	},
	OverflowToCityGold:   200,  // 200 溢出资源 = 1 城金
	ExchangeRate:         10,   // 1 金币 = 10 城金
	ReverseExchangeRate:  15,   // 15 城金 = 1 金币（反向有损耗）
	ExchangeCooldownSecs: 3600, // 兑换冷却 1 小时
	Buildings: map[string]BuildingConfig{
		"wood_camp": {
			Type:              "wood_camp",
			Name:              "伐木场",
			ResourceType:      "wood",
			ProductionByLevel: []int{4, 10, 18, 30, 44, 66, 100, 140, 200, 290, 400, 560, 750, 990, 1270, 1600, 2000, 2600, 3200, 4000, 4900},
			UpgradeCostByLevel: map[int]ResourceMap{
				0:  {"wood": 80, "stone": 200, "iron": 100, "food": 120},
				1:  {"wood": 130, "stone": 330, "iron": 170, "food": 200},
				2:  {"wood": 220, "stone": 560, "iron": 280, "food": 330},
				3:  {"wood": 370, "stone": 930, "iron": 470, "food": 560},
				4:  {"wood": 620, "stone": 1560, "iron": 780, "food": 930},
				5:  {"wood": 1040, "stone": 2600, "iron": 1300, "food": 1560},
				6:  {"wood": 1740, "stone": 4340, "iron": 2170, "food": 2600},
				7:  {"wood": 2900, "stone": 7250, "iron": 3620, "food": 4350},
				8:  {"wood": 4840, "stone": 12100, "iron": 6050, "food": 7260},
				9:  {"wood": 8080, "stone": 20210, "iron": 10100, "food": 12120},
				10: {"wood": 13500, "stone": 33740, "iron": 16870, "food": 20250},
				11: {"wood": 22540, "stone": 56350, "iron": 28180, "food": 33810},
				12: {"wood": 37640, "stone": 94110, "iron": 47050, "food": 56460},
				13: {"wood": 62860, "stone": 157160, "iron": 78580, "food": 94300},
				14: {"wood": 104980, "stone": 262460, "iron": 131230, "food": 157480},
				15: {"wood": 175320, "stone": 438310, "iron": 219150, "food": 262990},
				16: {"wood": 292780, "stone": 731980, "iron": 365980, "food": 439190},
				17: {"wood": 488940, "stone": 1222400, "iron": 611190, "food": 733450},
				18: {"wood": 816530, "stone": 2041410, "iron": 1020690, "food": 1224860},
				19: {"wood": 1363600, "stone": 3409150, "iron": 1704550, "food": 2045520},
			},
			UpgradeSecondsByLevel: map[int]int{0: 48, 1: 60, 2: 75, 3: 93, 4: 117, 5: 146, 6: 183, 7: 228, 8: 286, 9: 357, 10: 447, 11: 558, 12: 698, 13: 873, 14: 1091, 15: 1364, 16: 1705, 17: 2131, 18: 2664, 19: 3330},
		},
		"stone_quarry": {
			Type:              "stone_quarry",
			Name:              "采石场",
			ResourceType:      "stone",
			ProductionByLevel: []int{0, 10, 18, 30, 44, 66, 100, 140, 200, 290, 400, 560, 750, 990, 1270, 1600, 2000, 2600, 3200, 4000, 4900},
			UpgradeCostByLevel: map[int]ResourceMap{
				0:  {"wood": 160, "stone": 80, "iron": 160, "food": 100},
				1:  {"wood": 270, "stone": 130, "iron": 270, "food": 170},
				2:  {"wood": 450, "stone": 220, "iron": 450, "food": 280},
				3:  {"wood": 750, "stone": 370, "iron": 750, "food": 470},
				4:  {"wood": 1240, "stone": 620, "iron": 1240, "food": 780},
				5:  {"wood": 2080, "stone": 1040, "iron": 2080, "food": 1300},
				6:  {"wood": 3470, "stone": 1740, "iron": 3470, "food": 2170},
				7:  {"wood": 5800, "stone": 2900, "iron": 5800, "food": 3620},
				8:  {"wood": 9680, "stone": 4840, "iron": 9680, "food": 6050},
				9:  {"wood": 16160, "stone": 8080, "iron": 16160, "food": 10100},
				10: {"wood": 27000, "stone": 13500, "iron": 27000, "food": 16870},
				11: {"wood": 45080, "stone": 22540, "iron": 45080, "food": 28180},
				12: {"wood": 75290, "stone": 37640, "iron": 75290, "food": 47050},
				13: {"wood": 125730, "stone": 62860, "iron": 125730, "food": 78580},
				14: {"wood": 209970, "stone": 104980, "iron": 209970, "food": 131230},
				15: {"wood": 350650, "stone": 175320, "iron": 350650, "food": 219150},
				16: {"wood": 585590, "stone": 292780, "iron": 585590, "food": 365980},
				17: {"wood": 977940, "stone": 488940, "iron": 977940, "food": 611190},
				18: {"wood": 1633160, "stone": 816530, "iron": 1633160, "food": 1020690},
				19: {"wood": 2727380, "stone": 1363600, "iron": 2727380, "food": 1704550},
			},
			UpgradeSecondsByLevel: map[int]int{0: 20, 1: 30, 2: 45, 3: 67, 4: 101, 5: 151, 6: 227, 7: 341, 8: 512, 9: 768, 10: 1153, 11: 1729, 12: 2594, 13: 3892, 14: 5838, 15: 8757, 16: 13136, 17: 19705, 18: 29557, 19: 44336},
		},
		"iron_mine": {
			Type:              "iron_mine",
			Name:              "铁矿",
			ResourceType:      "iron",
			ProductionByLevel: []int{4, 10, 18, 30, 44, 66, 100, 140, 200, 290, 400, 560, 750, 990, 1270, 1600, 2000, 2600, 3200, 4000, 4900},
			UpgradeCostByLevel: map[int]ResourceMap{
				0:  {"wood": 200, "stone": 160, "iron": 60, "food": 120},
				1:  {"wood": 330, "stone": 270, "iron": 100, "food": 200},
				2:  {"wood": 560, "stone": 450, "iron": 170, "food": 330},
				3:  {"wood": 930, "stone": 750, "iron": 280, "food": 560},
				4:  {"wood": 1560, "stone": 1240, "iron": 470, "food": 930},
				5:  {"wood": 2600, "stone": 2080, "iron": 780, "food": 1560},
				6:  {"wood": 4340, "stone": 3470, "iron": 1300, "food": 2600},
				7:  {"wood": 7250, "stone": 5800, "iron": 2170, "food": 4350},
				8:  {"wood": 12100, "stone": 9680, "iron": 3630, "food": 7260},
				9:  {"wood": 20210, "stone": 16160, "iron": 6060, "food": 12120},
				10: {"wood": 33740, "stone": 27000, "iron": 10120, "food": 20250},
				11: {"wood": 56350, "stone": 45080, "iron": 16910, "food": 33810},
				12: {"wood": 94110, "stone": 75290, "iron": 28230, "food": 56460},
				13: {"wood": 157160, "stone": 125730, "iron": 47150, "food": 94300},
				14: {"wood": 262460, "stone": 209970, "iron": 78740, "food": 157480},
				15: {"wood": 438310, "stone": 350650, "iron": 131500, "food": 262990},
				16: {"wood": 731980, "stone": 585590, "iron": 219600, "food": 439190},
				17: {"wood": 1222400, "stone": 977940, "iron": 366730, "food": 733450},
				18: {"wood": 2041410, "stone": 1633160, "iron": 612440, "food": 1224860},
				19: {"wood": 3409150, "stone": 2727380, "iron": 1022780, "food": 2045520},
			},
			UpgradeSecondsByLevel: map[int]int{0: 48, 1: 60, 2: 75, 3: 93, 4: 117, 5: 146, 6: 183, 7: 228, 8: 286, 9: 357, 10: 447, 11: 558, 12: 698, 13: 873, 14: 1091, 15: 1364, 16: 1705, 17: 2131, 18: 2664, 19: 3330},
		},
		"farm": {
			Type:              "farm",
			Name:              "农田",
			ResourceType:      "food",
			ProductionByLevel: []int{0, 10, 18, 30, 44, 66, 100, 140, 200, 290, 400, 560, 750, 990, 1270, 1600, 2000, 2600, 3200, 4000, 4900},
			UpgradeCostByLevel: map[int]ResourceMap{
				0:  {"wood": 120, "stone": 100, "iron": 120, "food": 60},
				1:  {"wood": 200, "stone": 170, "iron": 200, "food": 100},
				2:  {"wood": 330, "stone": 280, "iron": 330, "food": 170},
				3:  {"wood": 560, "stone": 470, "iron": 560, "food": 280},
				4:  {"wood": 930, "stone": 780, "iron": 930, "food": 470},
				5:  {"wood": 1560, "stone": 1300, "iron": 1560, "food": 780},
				6:  {"wood": 2600, "stone": 2170, "iron": 2600, "food": 1300},
				7:  {"wood": 4350, "stone": 3620, "iron": 4350, "food": 2170},
				8:  {"wood": 7260, "stone": 6050, "iron": 7260, "food": 3630},
				9:  {"wood": 12120, "stone": 10100, "iron": 12120, "food": 6060},
				10: {"wood": 20250, "stone": 16870, "iron": 20250, "food": 10120},
				11: {"wood": 33810, "stone": 28180, "iron": 33810, "food": 16910},
				12: {"wood": 56460, "stone": 47050, "iron": 56460, "food": 28230},
				13: {"wood": 94300, "stone": 78580, "iron": 94300, "food": 47150},
				14: {"wood": 157480, "stone": 131230, "iron": 157480, "food": 78740},
				15: {"wood": 262990, "stone": 219150, "iron": 262990, "food": 131500},
				16: {"wood": 439190, "stone": 365980, "iron": 439190, "food": 219600},
				17: {"wood": 733450, "stone": 611190, "iron": 733450, "food": 366730},
				18: {"wood": 1224860, "stone": 1020690, "iron": 1224860, "food": 612440},
				19: {"wood": 2045520, "stone": 1704550, "iron": 2045520, "food": 1022780},
			},
			UpgradeSecondsByLevel: map[int]int{0: 20, 1: 30, 2: 45, 3: 67, 4: 101, 5: 151, 6: 227, 7: 341, 8: 512, 9: 768, 10: 1153, 11: 1729, 12: 2594, 13: 3892, 14: 5838, 15: 8757, 16: 13136, 17: 19705, 18: 29557, 19: 44336},
		},
		"warehouse": {
			Type:            "warehouse",
			Name:            "仓库",
			CapacityByLevel: []int{3200, 4800, 6800, 9200, 12400, 16000, 20000, 25200, 31200, 38400, 47200, 57600, 70400, 85600, 103600, 125200, 151600, 182800, 220400, 265600, 320000},
			UpgradeCostByLevel: map[int]ResourceMap{
				0:  {"wood": 260, "stone": 320, "iron": 180, "food": 80},
				1:  {"wood": 330, "stone": 410, "iron": 230, "food": 100},
				2:  {"wood": 430, "stone": 520, "iron": 290, "food": 130},
				3:  {"wood": 550, "stone": 670, "iron": 380, "food": 170},
				4:  {"wood": 700, "stone": 860, "iron": 480, "food": 210},
				5:  {"wood": 890, "stone": 1100, "iron": 620, "food": 270},
				6:  {"wood": 1140, "stone": 1410, "iron": 790, "food": 350},
				7:  {"wood": 1460, "stone": 1800, "iron": 1010, "food": 450},
				8:  {"wood": 1870, "stone": 2310, "iron": 1300, "food": 580},
				9:  {"wood": 2400, "stone": 2950, "iron": 1660, "food": 740},
				10: {"wood": 3070, "stone": 3780, "iron": 2130, "food": 940},
				11: {"wood": 3930, "stone": 4840, "iron": 2720, "food": 1210},
				12: {"wood": 5030, "stone": 6190, "iron": 3480, "food": 1550},
				13: {"wood": 6440, "stone": 7920, "iron": 4460, "food": 1980},
				14: {"wood": 8240, "stone": 10140, "iron": 5700, "food": 2540},
				15: {"wood": 10550, "stone": 12980, "iron": 7300, "food": 3250},
				16: {"wood": 13500, "stone": 16620, "iron": 9350, "food": 4150},
				17: {"wood": 17280, "stone": 21270, "iron": 11960, "food": 5320},
				18: {"wood": 22120, "stone": 27220, "iron": 15310, "food": 6810},
				19: {"wood": 28310, "stone": 34840, "iron": 19600, "food": 8710},
			},
			UpgradeSecondsByLevel: map[int]int{0: 85, 1: 120, 2: 168, 3: 235, 4: 329, 5: 460, 6: 645, 7: 903, 8: 1264, 9: 1770, 10: 2479, 11: 3471, 12: 4859, 13: 6803, 14: 9524, 15: 13334, 16: 18668, 17: 26135, 18: 36589, 19: 51225},
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
		BaseProduction:       cloneResourceMap(source.BaseProduction),
		Buildings:            make(map[string]BuildingConfig, len(source.Buildings)),
		OverflowToCityGold:   source.OverflowToCityGold,
		ExchangeRate:         source.ExchangeRate,
		ReverseExchangeRate:  source.ReverseExchangeRate,
		ExchangeCooldownSecs: source.ExchangeCooldownSecs,
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
