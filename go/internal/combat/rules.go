package combat

import (
	"encoding/json"
	"errors"
	"os"
	"sync"
)

// RuleConfig 单条战斗规则配置
type RuleConfig struct {
	ID               string  `json:"id"`
	Name             string  `json:"name"`
	Mode             string  `json:"mode"`             // "attack" or "plunder"
	Exponent         float64 `json:"exponent"`         // 损失指数（默认 1.422）
	EqualResult      string  `json:"equalResult"`      // "mutual_destruction" or "defender_wins"
	LossDistribution string  `json:"lossDistribution"` // "proportional" or "weak_first"
	DefenseFormula   string  `json:"defenseFormula"`   // "weighted"（按步骑加权）
}

// CombatConfig 战斗系统总配置
type CombatConfig struct {
	ActiveRules map[string]string    `json:"activeCombatRules"` // 场景 → 规则 ID
	Rules       map[string]RuleConfig `json:"rules"`
	WallConfig  map[string]WallEntry  `json:"wallConfig"`
}

// WallEntry 城墙配置
type WallEntry struct {
	Base float64 `json:"base"` // 城墙系数底数
}

// --- 全局配置管理 ---

var (
	combatMu     sync.RWMutex
	activeCombat = defaultCombatConfig()
)

func GetCombatConfig() CombatConfig {
	combatMu.RLock()
	defer combatMu.RUnlock()
	return activeCombat
}

func GetRule(ruleID string) (RuleConfig, bool) {
	cfg := GetCombatConfig()
	rule, ok := cfg.Rules[ruleID]
	return rule, ok
}

func GetActiveRule(scene string) (RuleConfig, bool) {
	cfg := GetCombatConfig()
	ruleID, ok := cfg.ActiveRules[scene]
	if !ok {
		return RuleConfig{}, false
	}
	rule, exists := cfg.Rules[ruleID]
	return rule, exists
}

func LoadCombatConfig(path string) error {
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

	var config CombatConfig
	if err := json.Unmarshal(content, &config); err != nil {
		return err
	}

	combatMu.Lock()
	activeCombat = config
	combatMu.Unlock()
	return nil
}

func SaveCombatConfig(path string, config CombatConfig) error {
	if path == "" {
		combatMu.Lock()
		activeCombat = config
		combatMu.Unlock()
		return nil
	}

	content, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(path, append(content, '\n'), 0o644); err != nil {
		return err
	}

	combatMu.Lock()
	activeCombat = config
	combatMu.Unlock()
	return nil
}

func defaultCombatConfig() CombatConfig {
	return CombatConfig{
		ActiveRules: map[string]string{
			"pve_attack":  "official_attack",
			"pve_plunder": "official_plunder",
			"pvp_attack":  "official_attack",
			"pvp_plunder": "official_plunder",
		},
		Rules: map[string]RuleConfig{
			"official_attack": {
				ID:               "official_attack",
				Name:             "官方攻击规则",
				Mode:             "attack",
				Exponent:         1.422,
				EqualResult:      "mutual_destruction",
				LossDistribution: "proportional",
				DefenseFormula:   "weighted",
			},
			"official_plunder": {
				ID:               "official_plunder",
				Name:             "官方掠夺规则",
				Mode:             "plunder",
				Exponent:         1.422,
				EqualResult:      "half_each",
				LossDistribution: "proportional",
				DefenseFormula:   "weighted",
			},
		},
		WallConfig: map[string]WallEntry{
			"wei": {Base: 1.03},
			"shu": {Base: 1.02},
			"wu":  {Base: 1.025},
		},
	}
}
