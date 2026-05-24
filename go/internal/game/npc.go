package game

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// NpcCity NPC 城池数据模型
// NOTE: 战斗扣兵/扣资源后，必须重置 ArmySettledAt / ResourceSettledAt 为当前时间，
// 否则下次结算会按旧时间戳计算出错误的恢复量。
type NpcCity struct {
	ID                string         `json:"id"`
	Name              string         `json:"name"`
	Faction           string         `json:"faction"`
	Tier              string         `json:"tier"`
	Resources         map[string]int `json:"resources"`
	StorageCapacity   map[string]int `json:"storageCapacity"`
	ProductionPerHour map[string]int `json:"productionPerHour"`
	Army              []ArmyUnit     `json:"army"`
	MaxArmy           []ArmyUnit     `json:"maxArmy"`
	ArmyRecoveryRate  float64        `json:"armyRecoveryRate"`
	RecoveryProfile   string         `json:"recoveryProfile"`
	Traits            []NpcTrait     `json:"traits"`
	ResourceSettledAt string         `json:"resourceSettledAt"`
	ArmySettledAt     string         `json:"armySettledAt"`
	GeneratedAt       string         `json:"generatedAt"`
}

// NpcTrait NPC 城池词条
type NpcTrait struct {
	ID    string             `json:"id"`
	Name  string             `json:"name"`
	Buffs map[string]float64 `json:"buffs"`
}

// NpcState 玩家的 NPC 城池状态
type NpcState struct {
	Cities          []NpcCity `json:"cities"`
	LastRefreshedAt string    `json:"lastRefreshedAt"`
}

// --- NPC 配置 ---

type NpcConfig struct {
	BaseProduction       int                      `json:"baseProduction"`
	BaseStorage          int                      `json:"baseStorage"`
	RefreshIntervalHours int                      `json:"refreshIntervalHours"`
	ManualRefreshCost    int                      `json:"manualRefreshCostGold"`
	GoldenAppearRate     float64                  `json:"goldenAppearRate"`
	TotalCities          int                      `json:"totalCities"`
	Tiers                map[string]NpcTierConfig `json:"tiers"`
	RecoveryProfiles     []NpcRecoveryProfile     `json:"recoveryProfiles"`
	TraitPool            []NpcTraitConfig         `json:"traitPool"`
	CityNames            []string                 `json:"cityNames"`
	ScoutCost            map[string]int           `json:"scoutCost"`
}

type NpcTierConfig struct {
	Multiplier float64      `json:"multiplier"`
	ArmyRange  IntRange     `json:"armyRange"`
	ArmyTypes  IntRange     `json:"armyTypes"`
	TraitCount IntRange     `json:"traitCount"`
	Count      NpcCountRule `json:"count"`
}

type IntRange struct {
	Min int `json:"min"`
	Max int `json:"max"`
}

type NpcCountRule struct {
	Guaranteed int `json:"guaranteed"`
	Weight     int `json:"weight"`
}

type NpcRecoveryProfile struct {
	ID                 string  `json:"id"`
	Name               string  `json:"name"`
	ArmyMultiplier     float64 `json:"armyMultiplier"`
	ResourceMultiplier float64 `json:"resourceMultiplier"`
	Weight             int     `json:"weight"`
}

type NpcTraitConfig struct {
	ID     string             `json:"id"`
	Name   string             `json:"name"`
	Buffs  map[string]float64 `json:"buffs"`
	Weight int                `json:"weight"`
}

// --- 全局 NPC 配置管理 ---

var (
	npcMu        sync.RWMutex
	activeNpcCfg = defaultNpcConfig()
)

func GetNpcConfig() NpcConfig {
	npcMu.RLock()
	defer npcMu.RUnlock()
	return activeNpcCfg
}

func SetNpcConfig(config NpcConfig) error {
	if err := validateNpcConfig(config); err != nil {
		return err
	}

	npcMu.Lock()
	activeNpcCfg = config
	npcMu.Unlock()
	return nil
}

func LoadNpcConfig(path string) error {
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

	var config NpcConfig
	if err := json.Unmarshal(content, &config); err != nil {
		return err
	}

	if err := validateNpcConfig(config); err != nil {
		return err
	}

	npcMu.Lock()
	activeNpcCfg = config
	npcMu.Unlock()
	return nil
}

func SaveNpcConfig(path string, config NpcConfig) error {
	if err := validateNpcConfig(config); err != nil {
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

func validateNpcConfig(cfg NpcConfig) error {
	if cfg.TotalCities <= 0 {
		return errors.New("npc config: totalCities must be > 0")
	}
	if cfg.BaseProduction <= 0 {
		return errors.New("npc config: baseProduction must be > 0")
	}
	if cfg.BaseStorage <= 0 {
		return errors.New("npc config: baseStorage must be > 0")
	}
	if cfg.GoldenAppearRate < 0 || cfg.GoldenAppearRate > 1 {
		return errors.New("npc config: goldenAppearRate must be between 0 and 1")
	}

	// 检查必需的 tier 存在
	requiredTiers := []string{"small", "medium", "large", "golden"}
	for _, tier := range requiredTiers {
		if _, ok := cfg.Tiers[tier]; !ok {
			return fmt.Errorf("npc config: missing required tier %q", tier)
		}
	}

	// 检查每个 tier 的字段合法性
	guaranteed := 0
	for tier, tierCfg := range cfg.Tiers {
		if tierCfg.Multiplier <= 0 {
			return fmt.Errorf("npc config: tier %q multiplier must be > 0", tier)
		}
		if tierCfg.ArmyRange.Min < 0 || tierCfg.ArmyRange.Max < tierCfg.ArmyRange.Min {
			return fmt.Errorf("npc config: tier %q armyRange invalid (min=%d, max=%d)", tier, tierCfg.ArmyRange.Min, tierCfg.ArmyRange.Max)
		}
		if tierCfg.ArmyTypes.Min < 0 || tierCfg.ArmyTypes.Max < tierCfg.ArmyTypes.Min {
			return fmt.Errorf("npc config: tier %q armyTypes invalid (min=%d, max=%d)", tier, tierCfg.ArmyTypes.Min, tierCfg.ArmyTypes.Max)
		}
		if tierCfg.TraitCount.Min < 0 || tierCfg.TraitCount.Max < tierCfg.TraitCount.Min {
			return fmt.Errorf("npc config: tier %q traitCount invalid (min=%d, max=%d)", tier, tierCfg.TraitCount.Min, tierCfg.TraitCount.Max)
		}
		guaranteed += tierCfg.Count.Guaranteed
	}

	// 检查 guaranteed 总和不超过 totalCities
	if guaranteed > cfg.TotalCities {
		return errors.New("npc config: guaranteed count sum exceeds totalCities")
	}

	// 检查至少有一个非金色 tier 有 weight > 0（用于分配剩余名额）
	hasWeight := false
	for tier, tierCfg := range cfg.Tiers {
		if tier != "golden" && tierCfg.Count.Weight > 0 {
			hasWeight = true
			break
		}
	}
	if guaranteed < cfg.TotalCities && !hasWeight {
		return errors.New("npc config: no tier has weight > 0 for remaining allocation")
	}

	// 检查 recoveryProfiles 权重
	totalRecoveryWeight := 0
	for _, p := range cfg.RecoveryProfiles {
		totalRecoveryWeight += p.Weight
	}
	if totalRecoveryWeight <= 0 {
		return errors.New("npc config: recoveryProfiles total weight must be > 0")
	}

	// 检查 traitPool 权重（如果有 tier 需要词条）
	maxTraitCount := 0
	for _, tier := range cfg.Tiers {
		if tier.TraitCount.Max > maxTraitCount {
			maxTraitCount = tier.TraitCount.Max
		}
	}
	if maxTraitCount > 0 {
		totalTraitWeight := 0
		for _, t := range cfg.TraitPool {
			totalTraitWeight += t.Weight
		}
		if totalTraitWeight <= 0 {
			return errors.New("npc config: traitPool total weight must be > 0 when traits are enabled")
		}
	}

	return nil
}

func defaultNpcConfig() NpcConfig {
	return NpcConfig{
		BaseProduction:       24500,
		BaseStorage:          320000,
		RefreshIntervalHours: 24,
		ManualRefreshCost:    50,
		GoldenAppearRate:     0.2,
		TotalCities:          12,
		Tiers: map[string]NpcTierConfig{
			"small":  {Multiplier: 1.0, ArmyRange: IntRange{50, 300}, ArmyTypes: IntRange{1, 2}, TraitCount: IntRange{0, 0}, Count: NpcCountRule{4, 5}},
			"medium": {Multiplier: 4.0, ArmyRange: IntRange{300, 1500}, ArmyTypes: IntRange{2, 3}, TraitCount: IntRange{0, 1}, Count: NpcCountRule{2, 3}},
			"large":  {Multiplier: 8.0, ArmyRange: IntRange{1500, 5000}, ArmyTypes: IntRange{3, 4}, TraitCount: IntRange{0, 2}, Count: NpcCountRule{1, 2}},
			"golden": {Multiplier: 16.0, ArmyRange: IntRange{5000, 15000}, ArmyTypes: IntRange{4, 5}, TraitCount: IntRange{1, 3}, Count: NpcCountRule{0, 0}},
		},
		CityNames: []string{"洛阳", "长安", "成都", "建业", "襄阳", "江陵"},
	}
}
