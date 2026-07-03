// 本文件测试 GM 配置以数据库为线上真实来源的装载和保存流程。
package game

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestSetFishingPathSeedsRepositoryFromFile 验证首次启动会把文件模板种入配置仓储。
func TestSetFishingPathSeedsRepositoryFromFile(t *testing.T) {
	repo := NewMemoryRepository()
	svc := NewServiceWithRepository(repo)
	path := writeFishingConfigFixture(t, testFishingConfig(11))

	if err := svc.SetFishingPath(path); err != nil {
		t.Fatalf("SetFishingPath failed: %v", err)
	}

	record, exists, err := repo.GetGameConfig(gameConfigKeyFishing)
	if err != nil {
		t.Fatalf("GetGameConfig failed: %v", err)
	}
	if !exists {
		t.Fatal("expected fishing config to be seeded")
	}
	stored := decodeFishingConfigFixture(t, record.ValueJSON)
	if stored.Baits[0].CityGoldCost != 11 {
		t.Fatalf("expected seeded file cost 11, got %+v", stored.Baits[0])
	}
}

// TestSetFishingPathPrefersRepositoryConfig 验证数据库已有配置时不会被文件模板覆盖。
func TestSetFishingPathPrefersRepositoryConfig(t *testing.T) {
	repo := NewMemoryRepository()
	dbConfig := testFishingConfig(22)
	content, err := json.Marshal(dbConfig)
	if err != nil {
		t.Fatalf("marshal db config: %v", err)
	}
	if _, err := repo.SaveGameConfig(gameConfigKeyFishing, content, "test", time.Now().UTC()); err != nil {
		t.Fatalf("SaveGameConfig failed: %v", err)
	}

	svc := NewServiceWithRepository(repo)
	path := writeFishingConfigFixture(t, testFishingConfig(11))
	if err := svc.SetFishingPath(path); err != nil {
		t.Fatalf("SetFishingPath failed: %v", err)
	}

	active := GetFishingConfig()
	if active.Baits[0].CityGoldCost != 22 {
		t.Fatalf("expected repository cost 22, got %+v", active.Baits[0])
	}
}

// TestUpdateFishingConfigWritesRepositoryOnly 验证 GM 修改只写配置仓储，不回写发布目录 JSON。
func TestUpdateFishingConfigWritesRepositoryOnly(t *testing.T) {
	repo := NewMemoryRepository()
	svc := NewServiceWithRepository(repo)
	path := writeFishingConfigFixture(t, testFishingConfig(11))
	if err := svc.SetFishingPath(path); err != nil {
		t.Fatalf("SetFishingPath failed: %v", err)
	}
	beforeFile, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read before file: %v", err)
	}

	updated := testFishingConfig(33)
	if err := svc.UpdateFishingConfig(updated); err != nil {
		t.Fatalf("UpdateFishingConfig failed: %v", err)
	}

	afterFile, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read after file: %v", err)
	}
	if string(beforeFile) != string(afterFile) {
		t.Fatal("expected GM update not to rewrite fishing config file")
	}

	record, exists, err := repo.GetGameConfig(gameConfigKeyFishing)
	if err != nil {
		t.Fatalf("GetGameConfig failed: %v", err)
	}
	if !exists {
		t.Fatal("expected fishing config in repository")
	}
	stored := decodeFishingConfigFixture(t, record.ValueJSON)
	if stored.Baits[0].CityGoldCost != 33 {
		t.Fatalf("expected repository cost 33, got %+v", stored.Baits[0])
	}
}

// testFishingConfig 构造一份最小但完整的钓鱼配置。
func testFishingConfig(cityGoldCost int) FishingConfig {
	return FishingConfig{
		Rarities: map[string]FishingRarityConfig{
			"common":    {Label: "普通", Weight: 65},
			"rare":      {Label: "稀有", Weight: 22},
			"epic":      {Label: "史诗", Weight: 8},
			"legendary": {Label: "传说", Weight: 1.5},
			"mythic":    {Label: "神话", Weight: 0.1},
		},
		Baits: []FishingBaitConfig{
			{
				ID: "coarse", Name: "粗饵", Tier: "一阶", Description: "测试鱼饵",
				RarityBoost: 1, CityGoldCost: cityGoldCost, BiteChance: 0.8,
				RarityWeights: map[string]float64{"common": 90, "rare": 10, "epic": 0, "legendary": 0, "mythic": 0},
				MinRarity:     "common", MaxRarity: "legendary", BiteDelayMultiplier: 1.1,
				BiteWindowMs: 1500, SweetStart: 50, SweetEnd: 80,
			},
		},
		FishPool: []FishingFishConfig{
			{Name: "草鱼", Rarity: "common", Reward: "青州军", RewardAmount: 100, Description: "测试鱼", Emoji: "fish"},
		},
	}
}

// TestValidateFishingConfigRejectsInvalidRarityWeights 验证鱼饵独立品质权重会拦截未知品质。
func TestValidateFishingConfigRejectsInvalidRarityWeights(t *testing.T) {
	cfg := testFishingConfig(11)
	cfg.Baits[0].RarityWeights = map[string]float64{"unknown": 1}

	if err := ValidateFishingConfig(cfg); err == nil {
		t.Fatal("expected invalid rarityWeights to be rejected")
	}
}

// TestValidateFishingConfigRejectsInvalidRarityRange 验证鱼饵最低/最高品质范围必须按品质顺序递增。
func TestValidateFishingConfigRejectsInvalidRarityRange(t *testing.T) {
	cfg := testFishingConfig(11)
	cfg.Baits[0].MinRarity = "legendary"
	cfg.Baits[0].MaxRarity = "rare"

	if err := ValidateFishingConfig(cfg); err == nil {
		t.Fatal("expected invalid rarity range to be rejected")
	}
}

// writeFishingConfigFixture 写入测试用钓鱼配置文件。
func writeFishingConfigFixture(t *testing.T, cfg FishingConfig) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "fishing.json")
	content, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		t.Fatalf("marshal fishing config: %v", err)
	}
	if err := os.WriteFile(path, append(content, '\n'), 0o644); err != nil {
		t.Fatalf("write fishing config: %v", err)
	}
	return path
}

// decodeFishingConfigFixture 解析测试中的钓鱼配置 JSON。
func decodeFishingConfigFixture(t *testing.T, content []byte) FishingConfig {
	t.Helper()
	var cfg FishingConfig
	if err := json.Unmarshal(content, &cfg); err != nil {
		t.Fatalf("decode fishing config: %v", err)
	}
	return cfg
}
