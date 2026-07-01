// 本文件定义天机轮转老虎机玩法配置和校验逻辑。
package game

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

type SlotConfig struct {
	MinLineBet           int          `json:"minLineBet"`
	LineCount            int          `json:"lineCount"`
	MaxFreeSpinsPerRound int          `json:"maxFreeSpinsPerRound"`
	Symbols              []SlotSymbol `json:"symbols"`
}

type SlotSymbol struct {
	ID                 string                `json:"id"`
	Name               string                `json:"name"`
	Rarity             string                `json:"rarity"`
	Type               string                `json:"type"`
	Weight             int                   `json:"weight"`
	Multiplier         int                   `json:"multiplier,omitempty"`
	FreeSpins          int                   `json:"freeSpins,omitempty"`
	RetriggerFreeSpins int                   `json:"retriggerFreeSpins,omitempty"`
	BonusMultipliers   []SlotBonusMultiplier `json:"bonusMultipliers,omitempty"`
}

type SlotBonusMultiplier struct {
	Multiplier int `json:"multiplier"`
	Weight     int `json:"weight"`
}

var (
	slotMu     sync.RWMutex
	slotConfig SlotConfig
)

var defaultSlotConfig = SlotConfig{
	MinLineBet:           1000,
	LineCount:            5,
	MaxFreeSpinsPerRound: 20,
	Symbols: []SlotSymbol{
		{ID: "bronze_charm", Name: "玄铜符", Rarity: "common", Type: "normal", Weight: 30, Multiplier: 3},
		{ID: "silver_charm", Name: "白银符", Rarity: "rare", Type: "normal", Weight: 22, Multiplier: 6},
		{ID: "gold_charm", Name: "赤金符", Rarity: "rare", Type: "normal", Weight: 16, Multiplier: 12},
		{ID: "jade_seal", Name: "玉玺", Rarity: "epic", Type: "normal", Weight: 10, Multiplier: 30},
		{ID: "tiger_tally", Name: "虎符", Rarity: "epic", Type: "normal", Weight: 6, Multiplier: 80},
		{ID: "heaven_order", Name: "天命令", Rarity: "legendary", Type: "normal", Weight: 2, Multiplier: 250},
		{ID: "wild", Name: "天机令", Rarity: "epic", Type: "wild", Weight: 5, Multiplier: 250},
		{ID: "scatter", Name: "星陨", Rarity: "epic", Type: "scatter", Weight: 4, FreeSpins: 5, RetriggerFreeSpins: 3},
		{ID: "bonus", Name: "宝匣", Rarity: "rare", Type: "bonus", Weight: 5, BonusMultipliers: []SlotBonusMultiplier{
			{Multiplier: 5, Weight: 50},
			{Multiplier: 10, Weight: 30},
			{Multiplier: 20, Weight: 15},
			{Multiplier: 50, Weight: 5},
		}},
	},
}

// LoadSlotConfig 从 JSON 文件加载天机轮转配置。
func LoadSlotConfig(path string) error {
	if path == "" {
		return nil
	}
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var cfg SlotConfig
	if err := json.Unmarshal(content, &cfg); err != nil {
		return err
	}
	return SetSlotConfig(cfg)
}

// GetSlotConfig 返回当前天机轮转配置；未加载时使用默认配置。
func GetSlotConfig() SlotConfig {
	slotMu.RLock()
	defer slotMu.RUnlock()
	if len(slotConfig.Symbols) == 0 {
		return cloneSlotConfig(defaultSlotConfig)
	}
	return cloneSlotConfig(slotConfig)
}

// SetSlotConfig 校验并替换当前天机轮转配置。
func SetSlotConfig(cfg SlotConfig) error {
	if err := ValidateSlotConfig(cfg); err != nil {
		return err
	}
	slotMu.Lock()
	slotConfig = cloneSlotConfig(cfg)
	slotMu.Unlock()
	return nil
}

// SaveSlotConfig 保存并启用天机轮转配置。
func SaveSlotConfig(path string, cfg SlotConfig) error {
	if err := SetSlotConfig(cfg); err != nil {
		return err
	}
	if path == "" {
		return nil
	}
	content, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, append(content, '\n'), 0o644)
}

// ValidateSlotConfig 校验天机轮转押注和图案配置。
func ValidateSlotConfig(cfg SlotConfig) error {
	if cfg.MinLineBet <= 0 {
		return errors.New("invalid slot bet config")
	}
	if cfg.LineCount <= 0 || cfg.MaxFreeSpinsPerRound <= 0 {
		return errors.New("invalid slot line or free spin config")
	}
	if len(cfg.Symbols) == 0 {
		return errors.New("slot symbols are required")
	}
	seen := map[string]bool{}
	hasWild := false
	hasScatter := false
	hasBonus := false
	hasHeavenOrder := false
	for _, symbol := range cfg.Symbols {
		id := strings.TrimSpace(symbol.ID)
		symbolType := strings.TrimSpace(symbol.Type)
		if id == "" || strings.TrimSpace(symbol.Name) == "" || strings.TrimSpace(symbol.Rarity) == "" || symbolType == "" {
			return errors.New("slot symbol id, name, rarity and type are required")
		}
		if seen[id] {
			return errors.New("duplicate slot symbol id: " + id)
		}
		seen[id] = true
		if symbol.Weight <= 0 {
			return errors.New("slot symbol weight must be positive: " + id)
		}
		switch symbolType {
		case "normal":
			if symbol.Multiplier <= 0 {
				return errors.New("slot normal symbol multiplier must be positive: " + id)
			}
			if id == "heaven_order" {
				hasHeavenOrder = true
			}
		case "wild":
			if symbol.Multiplier <= 0 {
				return errors.New("slot wild multiplier must be positive: " + id)
			}
			hasWild = true
		case "scatter":
			if symbol.FreeSpins <= 0 || symbol.RetriggerFreeSpins <= 0 {
				return errors.New("slot scatter free spin config must be positive: " + id)
			}
			hasScatter = true
		case "bonus":
			if len(symbol.BonusMultipliers) == 0 {
				return errors.New("slot bonus multipliers are required: " + id)
			}
			for _, bonus := range symbol.BonusMultipliers {
				if bonus.Multiplier <= 0 || bonus.Weight <= 0 {
					return errors.New("slot bonus multiplier and weight must be positive: " + id)
				}
			}
			hasBonus = true
		default:
			return errors.New("unsupported slot symbol type: " + symbolType)
		}
	}
	if !hasWild || !hasScatter || !hasBonus || !hasHeavenOrder {
		return errors.New("slot config must include wild, scatter, bonus and heaven_order symbols")
	}
	return nil
}

// cloneSlotConfig 复制配置，避免调用方修改全局配置。
func cloneSlotConfig(cfg SlotConfig) SlotConfig {
	result := SlotConfig{
		MinLineBet:           cfg.MinLineBet,
		LineCount:            cfg.LineCount,
		MaxFreeSpinsPerRound: cfg.MaxFreeSpinsPerRound,
		Symbols:              append([]SlotSymbol(nil), cfg.Symbols...),
	}
	for i := range result.Symbols {
		result.Symbols[i].BonusMultipliers = append([]SlotBonusMultiplier(nil), cfg.Symbols[i].BonusMultipliers...)
	}
	return result
}
