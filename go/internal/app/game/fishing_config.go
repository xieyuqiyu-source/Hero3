package game

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

type FishingConfig struct {
	Rarities map[string]FishingRarityConfig `json:"rarities"`
	Baits    []FishingBaitConfig            `json:"baits"`
	FishPool []FishingFishConfig            `json:"fishPool"`
}

type FishingRarityConfig struct {
	Label  string  `json:"label"`
	Color  string  `json:"color"`
	Bg     string  `json:"bg"`
	Border string  `json:"border"`
	Weight float64 `json:"weight"`
	Glow   string  `json:"glow"`
}

type FishingBaitConfig struct {
	ID           string  `json:"id"`
	Name         string  `json:"name"`
	Tier         string  `json:"tier"`
	Description  string  `json:"description"`
	RarityBoost  float64 `json:"rarityBoost"`
	CityGoldCost int     `json:"cityGoldCost"`
	BiteChance   float64 `json:"biteChance"`
	BiteWindowMs int     `json:"biteWindowMs"`
	SweetStart   int     `json:"sweetStart"`
	SweetEnd     int     `json:"sweetEnd"`
}

type FishingFishConfig struct {
	Name         string `json:"name"`
	Rarity       string `json:"rarity"`
	Reward       string `json:"reward"`
	RewardAmount int    `json:"rewardAmount"`
	Description  string `json:"description"`
	Emoji        string `json:"emoji"`
}

var (
	fishingMu     sync.RWMutex
	fishingConfig FishingConfig
)

var fallbackFishingBaitCosts = map[string]int{
	"coarse": 0,
	"shrimp": 30,
	"golden": 120,
	"dragon": 500,
}

func LoadFishingConfig(path string) error {
	if path == "" {
		return nil
	}
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var cfg FishingConfig
	if err := json.Unmarshal(content, &cfg); err != nil {
		return err
	}
	return SetFishingConfig(cfg)
}

func GetFishingConfig() FishingConfig {
	fishingMu.RLock()
	defer fishingMu.RUnlock()
	return cloneFishingConfig(fishingConfig)
}

func SetFishingConfig(cfg FishingConfig) error {
	if err := ValidateFishingConfig(cfg); err != nil {
		return err
	}
	fishingMu.Lock()
	fishingConfig = cloneFishingConfig(cfg)
	fishingMu.Unlock()
	return nil
}

func SaveFishingConfig(path string, cfg FishingConfig) error {
	if err := SetFishingConfig(cfg); err != nil {
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

func ValidateFishingConfig(cfg FishingConfig) error {
	if len(cfg.Rarities) == 0 {
		return errors.New("fishing rarities are required")
	}
	for id, rarity := range cfg.Rarities {
		if strings.TrimSpace(id) == "" || strings.TrimSpace(rarity.Label) == "" {
			return errors.New("fishing rarity id and label are required")
		}
		if rarity.Weight <= 0 {
			return errors.New("fishing rarity weight must be positive: " + id)
		}
	}
	if len(cfg.Baits) == 0 {
		return errors.New("fishing baits are required")
	}
	seenBaits := map[string]bool{}
	for _, bait := range cfg.Baits {
		id := strings.TrimSpace(bait.ID)
		if id == "" || strings.TrimSpace(bait.Name) == "" {
			return errors.New("fishing bait id and name are required")
		}
		if seenBaits[id] {
			return errors.New("duplicate fishing bait id: " + id)
		}
		seenBaits[id] = true
		if bait.RarityBoost <= 0 {
			return errors.New("fishing bait rarityBoost must be positive: " + id)
		}
		if bait.CityGoldCost < 0 || bait.BiteChance < 0 || bait.BiteChance > 1 || bait.BiteWindowMs <= 0 {
			return errors.New("invalid fishing bait numeric config: " + id)
		}
		if bait.SweetStart < 0 || bait.SweetEnd > 100 || bait.SweetStart >= bait.SweetEnd {
			return errors.New("invalid fishing bait sweet range: " + id)
		}
	}
	if len(cfg.FishPool) == 0 {
		return errors.New("fishing fishPool is required")
	}
	for _, fish := range cfg.FishPool {
		if strings.TrimSpace(fish.Name) == "" || strings.TrimSpace(fish.Rarity) == "" {
			return errors.New("fishing fish name and rarity are required")
		}
		if _, ok := cfg.Rarities[fish.Rarity]; !ok {
			return errors.New("fishing fish references unknown rarity: " + fish.Name)
		}
		if strings.TrimSpace(fish.Reward) == "" || fish.RewardAmount <= 0 {
			return errors.New("fishing fish reward and amount are required: " + fish.Name)
		}
	}
	return nil
}

func GetFishingBaitCost(baitID string) (int, bool) {
	baitID = strings.TrimSpace(baitID)
	fishingMu.RLock()
	defer fishingMu.RUnlock()
	for _, bait := range fishingConfig.Baits {
		if bait.ID == baitID {
			return bait.CityGoldCost, true
		}
	}
	cost, ok := fallbackFishingBaitCosts[baitID]
	return cost, ok
}

func cloneFishingConfig(cfg FishingConfig) FishingConfig {
	result := FishingConfig{
		Rarities: make(map[string]FishingRarityConfig, len(cfg.Rarities)),
		Baits:    append([]FishingBaitConfig(nil), cfg.Baits...),
		FishPool: append([]FishingFishConfig(nil), cfg.FishPool...),
	}
	for key, value := range cfg.Rarities {
		result.Rarities[key] = value
	}
	return result
}
