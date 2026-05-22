package game

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
)

// UnitConfig 兵种配置 — 所有属性用 map 存储，支持无限扩展
type UnitConfig struct {
	Name         string         `json:"name"`
	Description  string         `json:"description"`
	Category     string         `json:"category"`
	Icon         string         `json:"icon"`
	Stats        map[string]int `json:"stats"`
	Cost         map[string]int `json:"cost"`
	TrainSeconds int            `json:"trainSeconds"`
	Unlock       map[string]any `json:"unlock"`
}

// FactionUnits 单阵营的兵种集合：map[unitId]UnitConfig
type FactionUnits map[string]UnitConfig

// UnitsConfig 全部阵营的兵种：map[faction]FactionUnits
type UnitsConfig map[string]FactionUnits

// FactionConfig 阵营配置
type FactionConfig struct {
	Name        string             `json:"name"`
	Description string             `json:"description"`
	Icon        string             `json:"icon"`
	Traits      map[string]float64 `json:"traits"`
}

// FactionsConfig 全部阵营：map[factionId]FactionConfig
type FactionsConfig map[string]FactionConfig

var (
	unitsMu       sync.RWMutex
	activeUnits   = UnitsConfig{}
	factionsMu    sync.RWMutex
	activeFactions = FactionsConfig{}
)

// GetUnitsConfig 获取当前兵种配置
func GetUnitsConfig() UnitsConfig {
	unitsMu.RLock()
	defer unitsMu.RUnlock()
	return activeUnits
}

// GetFactionsConfig 获取当前阵营配置
func GetFactionsConfig() FactionsConfig {
	factionsMu.RLock()
	defer factionsMu.RUnlock()
	return activeFactions
}

// GetFactionUnits 获取指定阵营的兵种
func GetFactionUnits(faction string) FactionUnits {
	unitsMu.RLock()
	defer unitsMu.RUnlock()
	return activeUnits[faction]
}

// GetUnitConfig 获取指定阵营的指定兵种
func GetUnitConfig(faction string, unitID string) (UnitConfig, bool) {
	units := GetFactionUnits(faction)
	if units == nil {
		return UnitConfig{}, false
	}
	config, exists := units[unitID]
	return config, exists
}

// LoadFactionsConfig 从 JSON 文件加载阵营配置
func LoadFactionsConfig(path string) error {
	if path == "" {
		return nil
	}

	content, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}

	var config FactionsConfig
	if err := json.Unmarshal(content, &config); err != nil {
		return err
	}

	factionsMu.Lock()
	activeFactions = config
	factionsMu.Unlock()
	return nil
}

// LoadUnitsConfig 从目录加载所有阵营的兵种配置
// 目录下每个 JSON 文件名（不含扩展名）作为阵营 ID
func LoadUnitsConfig(dir string) error {
	if dir == "" {
		return nil
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}

	config := UnitsConfig{}
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		faction := entry.Name()[:len(entry.Name())-5] // 去掉 .json
		content, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			return err
		}

		var units FactionUnits
		if err := json.Unmarshal(content, &units); err != nil {
			return err
		}
		config[faction] = units
	}

	unitsMu.Lock()
	activeUnits = config
	unitsMu.Unlock()
	return nil
}

// ApplyTrait 应用阵营加成到基础值
func ApplyTrait(faction FactionConfig, traitKey string, base int) int {
	multiplier, ok := faction.Traits[traitKey]
	if !ok || multiplier == 0 {
		multiplier = 1.0
	}
	return int(float64(base) * multiplier)
}

// ApplyTraits 应用多个加成源到基础值
func ApplyTraits(base int, key string, sources ...map[string]float64) int {
	result := float64(base)
	for _, source := range sources {
		if multiplier, ok := source[key]; ok && multiplier != 0 {
			result *= multiplier
		}
	}
	return int(result)
}
