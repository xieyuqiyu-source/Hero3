// 本文件维护仙池垂钓的 GM 配置结构、校验和内存快照。
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
	ID                  string             `json:"id"`
	Name                string             `json:"name"`
	Tier                string             `json:"tier"`
	Description         string             `json:"description"`
	RarityBoost         float64            `json:"rarityBoost"`
	RarityWeights       map[string]float64 `json:"rarityWeights,omitempty"`
	MinRarity           string             `json:"minRarity,omitempty"`
	MaxRarity           string             `json:"maxRarity,omitempty"`
	BiteDelayMultiplier float64            `json:"biteDelayMultiplier,omitempty"`
	CityGoldCost        int                `json:"cityGoldCost"`
	BiteChance          float64            `json:"biteChance"`
	BiteWindowMs        int                `json:"biteWindowMs"`
	SweetStart          int                `json:"sweetStart"`
	SweetEnd            int                `json:"sweetEnd"`
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
	"coarse":   0,
	"shrimp":   30,
	"golden":   120,
	"dragon":   500,
	"shenlong": 5000,
}

// LoadFishingConfig 从文件读取钓鱼模板配置并加载到内存。
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

// GetFishingConfig 返回当前钓鱼配置的防御性副本。
func GetFishingConfig() FishingConfig {
	fishingMu.RLock()
	defer fishingMu.RUnlock()
	return cloneFishingConfig(fishingConfig)
}

// SetFishingConfig 校验并替换当前内存中的钓鱼配置。
func SetFishingConfig(cfg FishingConfig) error {
	if err := ValidateFishingConfig(cfg); err != nil {
		return err
	}
	fishingMu.Lock()
	fishingConfig = cloneFishingConfig(cfg)
	fishingMu.Unlock()
	return nil
}

// SaveFishingConfig 保存钓鱼配置到文件；线上 GM 流程不再使用它写发布目录。
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

// ValidateFishingConfig 校验钓鱼配置结构，防止 GM 保存不可抽取或不可结算的配置。
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
	rarityOrder := buildFishingRarityOrder(cfg.Rarities)
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
		if err := validateFishingBaitRarityControls(id, bait, cfg.Rarities, rarityOrder); err != nil {
			return err
		}
		if bait.BiteDelayMultiplier < 0 {
			return errors.New("invalid fishing bait delay multiplier: " + id)
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

// GetFishingBaitCost 返回指定鱼饵的城金消耗。
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

// cloneFishingConfig 深拷贝钓鱼配置，避免调用方修改全局内存配置。
func cloneFishingConfig(cfg FishingConfig) FishingConfig {
	result := FishingConfig{
		Rarities: make(map[string]FishingRarityConfig, len(cfg.Rarities)),
		Baits:    make([]FishingBaitConfig, len(cfg.Baits)),
		FishPool: append([]FishingFishConfig(nil), cfg.FishPool...),
	}
	for key, value := range cfg.Rarities {
		result.Rarities[key] = value
	}
	for index, bait := range cfg.Baits {
		result.Baits[index] = bait
		if bait.RarityWeights != nil {
			result.Baits[index].RarityWeights = make(map[string]float64, len(bait.RarityWeights))
			for key, value := range bait.RarityWeights {
				result.Baits[index].RarityWeights[key] = value
			}
		}
	}
	return result
}

// validateFishingBaitRarityControls 校验单个鱼饵的品质范围和独立权重。
func validateFishingBaitRarityControls(id string, bait FishingBaitConfig, rarities map[string]FishingRarityConfig, rarityOrder map[string]int) error {
	minRank := -1
	maxRank := -1
	minRarity := strings.TrimSpace(bait.MinRarity)
	maxRarity := strings.TrimSpace(bait.MaxRarity)
	if minRarity != "" {
		rank, ok := rarityOrder[minRarity]
		if !ok {
			return errors.New("fishing bait minRarity references unknown rarity: " + id)
		}
		minRank = rank
	}
	if maxRarity != "" {
		rank, ok := rarityOrder[maxRarity]
		if !ok {
			return errors.New("fishing bait maxRarity references unknown rarity: " + id)
		}
		maxRank = rank
	}
	if minRank >= 0 && maxRank >= 0 && minRank > maxRank {
		return errors.New("invalid fishing bait rarity range: " + id)
	}
	if len(bait.RarityWeights) == 0 {
		return nil
	}
	hasPositiveWeight := false
	for rarityID, weight := range bait.RarityWeights {
		rarityID = strings.TrimSpace(rarityID)
		if rarityID == "" {
			return errors.New("fishing bait rarity weight id is required: " + id)
		}
		if _, ok := rarities[rarityID]; !ok {
			return errors.New("fishing bait rarityWeights references unknown rarity: " + id + "." + rarityID)
		}
		if weight < 0 {
			return errors.New("fishing bait rarityWeights must not be negative: " + id + "." + rarityID)
		}
		rank := rarityOrder[rarityID]
		if weight > 0 && (minRank < 0 || rank >= minRank) && (maxRank < 0 || rank <= maxRank) {
			hasPositiveWeight = true
		}
	}
	if !hasPositiveWeight {
		return errors.New("fishing bait rarityWeights must include at least one allowed positive weight: " + id)
	}
	return nil
}

// buildFishingRarityOrder 给常用品质固定排序，未知自定义品质追加在后面。
func buildFishingRarityOrder(rarities map[string]FishingRarityConfig) map[string]int {
	order := map[string]int{}
	for rank, rarityID := range []string{"common", "rare", "epic", "legendary", "mythic"} {
		if _, ok := rarities[rarityID]; ok {
			order[rarityID] = rank
		}
	}
	nextRank := len(order)
	for rarityID := range rarities {
		if _, ok := order[rarityID]; ok {
			continue
		}
		order[rarityID] = nextRank
		nextRank++
	}
	return order
}
