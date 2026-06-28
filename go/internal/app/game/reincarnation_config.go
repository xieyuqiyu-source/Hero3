// 本文件负责轮回绝境副本配置的加载、校验和全局读取。
package game

import (
	"encoding/json"
	"errors"
	"os"
	"sort"
	"strings"
	"sync"
)

type ReincarnationConfig struct {
	Levels                  []ReincarnationLevelConfig `json:"levels"`
	Waves                   []ReincarnationWaveConfig  `json:"waves"`
	EnemyFactions           []string                   `json:"enemyFactions"`
	BonusValues             []float64                  `json:"bonusValues"`
	DefenseCountdownSeconds int                        `json:"defenseCountdownSeconds"`
}

type ReincarnationLevelConfig struct {
	Level           int    `json:"level"`
	Name            string `json:"name"`
	WavePowerBase   int64  `json:"wavePowerBase"`
	PlayerTroopCap  int64  `json:"playerTroopCap"`
	EnemyTroopBase  int64  `json:"enemyTroopBase"`
	DurationSeconds int    `json:"durationSeconds"`
	RewardExpCap    int64  `json:"rewardExpCap"`
	Enabled         bool   `json:"enabled"`
}

type ReincarnationWaveConfig struct {
	WaveIndex     int      `json:"waveIndex"`
	RewardPreview []Reward `json:"rewardPreview,omitempty"`
	ExpBudgetRate float64  `json:"expBudgetRate"`
	ExpRandomMin  float64  `json:"expRandomMin"`
	ExpRandomMax  float64  `json:"expRandomMax"`
	FixedRewards  []Reward `json:"fixedRewards"`
	DropPoolID    string   `json:"dropPoolId,omitempty"`
}

var (
	reincarnationConfigMu sync.RWMutex
	reincarnationConfig   = ReincarnationConfig{}
)

// LoadReincarnationConfig 从 JSON 文件加载轮回绝境配置。
func LoadReincarnationConfig(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var cfg ReincarnationConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return err
	}
	if err := ValidateReincarnationConfig(cfg); err != nil {
		return err
	}
	reincarnationConfigMu.Lock()
	defer reincarnationConfigMu.Unlock()
	reincarnationConfig = cloneReincarnationConfig(cfg)
	return nil
}

// SaveReincarnationConfig 校验并保存轮回绝境配置。
func SaveReincarnationConfig(path string, cfg ReincarnationConfig) error {
	if err := ValidateReincarnationConfig(cfg); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(path, append(data, '\n'), 0o644); err != nil {
		return err
	}
	reincarnationConfigMu.Lock()
	defer reincarnationConfigMu.Unlock()
	reincarnationConfig = cloneReincarnationConfig(cfg)
	return nil
}

// GetReincarnationConfig 返回当前轮回绝境配置副本。
func GetReincarnationConfig() ReincarnationConfig {
	reincarnationConfigMu.RLock()
	defer reincarnationConfigMu.RUnlock()
	return cloneReincarnationConfig(reincarnationConfig)
}

// ValidateReincarnationConfig 校验轮回绝境配置是否满足第一版规则。
func ValidateReincarnationConfig(cfg ReincarnationConfig) error {
	if len(cfg.Levels) == 0 {
		return errors.New("reincarnation levels are required")
	}
	seenLevels := map[int]bool{}
	for _, level := range cfg.Levels {
		if level.Level <= 0 || strings.TrimSpace(level.Name) == "" {
			return errors.New("reincarnation level and name are required")
		}
		if seenLevels[level.Level] {
			return errors.New("duplicate reincarnation level")
		}
		seenLevels[level.Level] = true
		if level.EnemyTroopBase <= 0 || level.PlayerTroopCap <= 0 || level.DurationSeconds <= 0 {
			return errors.New("reincarnation level troop base, cap and duration must be positive")
		}
	}
	if len(cfg.Waves) != ReincarnationWaveCount {
		return errors.New("reincarnation must define 18 waves")
	}
	seenWaves := map[int]bool{}
	for _, wave := range cfg.Waves {
		if wave.WaveIndex < 1 || wave.WaveIndex > ReincarnationWaveCount || seenWaves[wave.WaveIndex] {
			return errors.New("invalid reincarnation wave index")
		}
		seenWaves[wave.WaveIndex] = true
		if wave.ExpRandomMin <= 0 || wave.ExpRandomMax < wave.ExpRandomMin {
			return errors.New("invalid reincarnation wave random range")
		}
		if strings.TrimSpace(wave.DropPoolID) != "" {
			if _, ok := GetDropPoolDefinition(wave.DropPoolID); !ok && len(GetDropPoolsConfig()) > 0 {
				return ErrDropPoolNotFound
			}
		}
	}
	if len(cfg.EnemyFactions) == 0 {
		return errors.New("reincarnation enemy factions are required")
	}
	if len(cfg.BonusValues) == 0 {
		return errors.New("reincarnation bonus values are required")
	}
	return nil
}

// reincarnationLevelConfig 查找指定层级配置。
func reincarnationLevelConfig(level int) (ReincarnationLevelConfig, bool) {
	cfg := GetReincarnationConfig()
	for _, item := range cfg.Levels {
		if item.Level == level {
			return item, true
		}
	}
	return ReincarnationLevelConfig{}, false
}

// reincarnationWaveConfig 查找指定波次奖励配置。
func reincarnationWaveConfig(waveIndex int) ReincarnationWaveConfig {
	cfg := GetReincarnationConfig()
	for _, item := range cfg.Waves {
		if item.WaveIndex == waveIndex {
			return item
		}
	}
	return ReincarnationWaveConfig{WaveIndex: waveIndex, ExpRandomMin: 1, ExpRandomMax: 1}
}

func cloneReincarnationConfig(source ReincarnationConfig) ReincarnationConfig {
	next := source
	next.Levels = append([]ReincarnationLevelConfig(nil), source.Levels...)
	next.Waves = append([]ReincarnationWaveConfig(nil), source.Waves...)
	for i := range next.Waves {
		next.Waves[i].RewardPreview = append([]Reward(nil), source.Waves[i].RewardPreview...)
		next.Waves[i].FixedRewards = append([]Reward(nil), source.Waves[i].FixedRewards...)
	}
	next.EnemyFactions = append([]string(nil), source.EnemyFactions...)
	next.BonusValues = append([]float64(nil), source.BonusValues...)
	sort.Slice(next.Levels, func(i, j int) bool { return next.Levels[i].Level < next.Levels[j].Level })
	sort.Slice(next.Waves, func(i, j int) bool { return next.Waves[i].WaveIndex < next.Waves[j].WaveIndex })
	return next
}
