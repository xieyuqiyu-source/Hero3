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
	MinBet      int          `json:"minBet"`
	MaxBet      int          `json:"maxBet"`
	MaxBetRatio float64      `json:"maxBetRatio"`
	Symbols     []SlotSymbol `json:"symbols"`
}

type SlotSymbol struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Rarity     string `json:"rarity"`
	Weight     int    `json:"weight"`
	Multiplier int    `json:"multiplier"`
}

var (
	slotMu     sync.RWMutex
	slotConfig SlotConfig
)

var defaultSlotConfig = SlotConfig{
	MinBet:      1000,
	MaxBet:      1000000,
	MaxBetRatio: 0.05,
	Symbols: []SlotSymbol{
		{ID: "bronze_charm", Name: "玄铜符", Rarity: "common", Weight: 35, Multiplier: 5},
		{ID: "silver_charm", Name: "白银符", Rarity: "rare", Weight: 25, Multiplier: 12},
		{ID: "gold_charm", Name: "赤金符", Rarity: "rare", Weight: 18, Multiplier: 30},
		{ID: "jade_seal", Name: "玉玺", Rarity: "epic", Weight: 12, Multiplier: 80},
		{ID: "tiger_tally", Name: "虎符", Rarity: "epic", Weight: 7, Multiplier: 250},
		{ID: "heaven_order", Name: "天命令", Rarity: "legendary", Weight: 3, Multiplier: 1000},
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
	if cfg.MinBet <= 0 || cfg.MaxBet < cfg.MinBet || cfg.MaxBetRatio <= 0 || cfg.MaxBetRatio > 1 {
		return errors.New("invalid slot bet config")
	}
	if len(cfg.Symbols) == 0 {
		return errors.New("slot symbols are required")
	}
	seen := map[string]bool{}
	for _, symbol := range cfg.Symbols {
		id := strings.TrimSpace(symbol.ID)
		if id == "" || strings.TrimSpace(symbol.Name) == "" || strings.TrimSpace(symbol.Rarity) == "" {
			return errors.New("slot symbol id, name and rarity are required")
		}
		if seen[id] {
			return errors.New("duplicate slot symbol id: " + id)
		}
		seen[id] = true
		if symbol.Weight <= 0 || symbol.Multiplier <= 0 {
			return errors.New("slot symbol weight and multiplier must be positive: " + id)
		}
	}
	return nil
}

// cloneSlotConfig 复制配置，避免调用方修改全局配置。
func cloneSlotConfig(cfg SlotConfig) SlotConfig {
	return SlotConfig{
		MinBet:      cfg.MinBet,
		MaxBet:      cfg.MaxBet,
		MaxBetRatio: cfg.MaxBetRatio,
		Symbols:     append([]SlotSymbol(nil), cfg.Symbols...),
	}
}
