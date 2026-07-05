// 本文件维护黄巾起义的 GM 配置结构、默认值和校验逻辑。
package game

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"sync"
)

const (
	gameConfigKeyYellowTurban = "yellow_turban"
	ThousandTentCampType      = "thousand_tent_camp"
)

// YellowTurbanConfig 是黄巾起义玩法的完整运营配置。
type YellowTurbanConfig struct {
	Enabled                     bool                          `json:"enabled"`
	ThousandTentCamp            ThousandTentCampConfig        `json:"thousandTentCamp"`
	CheckIntervalMinutes        int                           `json:"checkIntervalMinutes"`
	MaxIncomingMarchesPerPlayer int                           `json:"maxIncomingMarchesPerPlayer"`
	MarchSpeedMultiplier        float64                       `json:"marchSpeedMultiplier"`
	Regions                     []YellowTurbanRegionConfig    `json:"regions"`
	RiskLevels                  []YellowTurbanRiskLevelConfig `json:"riskLevels"`
}

// ThousandTentCampConfig 描述千帐营建筑的口粮上限和金币升级费用。
type ThousandTentCampConfig struct {
	Enabled                bool   `json:"enabled"`
	BuildingType           string `json:"buildingType"`
	Name                   string `json:"name"`
	Description            string `json:"description"`
	CapacityByLevel        []int  `json:"capacityByLevel"`
	GoldUpgradeCostByLevel []int  `json:"goldUpgradeCostByLevel"`
}

// YellowTurbanRegionConfig 描述黄巾城池地区与可用兵种池。
type YellowTurbanRegionConfig struct {
	ID            string   `json:"id"`
	Name          string   `json:"name"`
	Faction       string   `json:"faction"`
	Enabled       bool     `json:"enabled"`
	CityCount     int      `json:"cityCount"`
	UnitPool      []string `json:"unitPool,omitempty"`
	ExcludedUnits []string `json:"excludedUnits,omitempty"`
	MinUnitKinds  int      `json:"minUnitKinds"`
	MaxUnitKinds  int      `json:"maxUnitKinds"`
}

// YellowTurbanRiskLevelConfig 描述口粮压力对应的黄巾风险档位。
type YellowTurbanRiskLevelConfig struct {
	ID           int     `json:"id"`
	Name         string  `json:"name"`
	Color        string  `json:"color"`
	MinPressure  float64 `json:"minPressure"`
	MaxPressure  float64 `json:"maxPressure"`
	MinRatio     float64 `json:"minRatio"`
	MaxRatio     float64 `json:"maxRatio"`
	MinUnitKinds int     `json:"minUnitKinds"`
	MaxUnitKinds int     `json:"maxUnitKinds"`
	MaxIncoming  int     `json:"maxIncoming"`
	Enabled      bool    `json:"enabled"`
}

var (
	yellowTurbanMu     sync.RWMutex
	activeYellowTurban = defaultYellowTurbanConfig()
)

// LoadYellowTurbanConfig 从文件加载黄巾起义配置。
func LoadYellowTurbanConfig(path string) error {
	if path == "" {
		SetYellowTurbanConfig(defaultYellowTurbanConfig())
		return nil
	}
	content, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			SetYellowTurbanConfig(defaultYellowTurbanConfig())
			return nil
		}
		return err
	}
	var cfg YellowTurbanConfig
	if err := json.Unmarshal(content, &cfg); err != nil {
		return err
	}
	if err := ValidateYellowTurbanConfig(cfg); err != nil {
		return err
	}
	SetYellowTurbanConfig(cfg)
	return nil
}

// SaveYellowTurbanConfig 保存黄巾起义配置到文件并刷新内存快照。
func SaveYellowTurbanConfig(path string, cfg YellowTurbanConfig) error {
	if err := ValidateYellowTurbanConfig(cfg); err != nil {
		return err
	}
	if path != "" {
		content, err := json.MarshalIndent(cfg, "", "  ")
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(path, append(content, '\n'), 0o644); err != nil {
			return err
		}
	}
	SetYellowTurbanConfig(cfg)
	return nil
}

// GetYellowTurbanConfig 返回黄巾起义配置快照。
func GetYellowTurbanConfig() YellowTurbanConfig {
	yellowTurbanMu.RLock()
	defer yellowTurbanMu.RUnlock()
	return cloneYellowTurbanConfig(activeYellowTurban)
}

// SetYellowTurbanConfig 设置黄巾起义配置快照。
func SetYellowTurbanConfig(cfg YellowTurbanConfig) {
	yellowTurbanMu.Lock()
	activeYellowTurban = normalizeYellowTurbanConfig(cfg)
	yellowTurbanMu.Unlock()
}

// ValidateYellowTurbanConfig 校验黄巾起义配置，避免 GM 保存不可用规则。
func ValidateYellowTurbanConfig(cfg YellowTurbanConfig) error {
	cfg = normalizeYellowTurbanConfig(cfg)
	if len(cfg.ThousandTentCamp.CapacityByLevel) < 25 {
		return errors.New("thousand tent camp requires 25 capacity levels")
	}
	if len(cfg.ThousandTentCamp.GoldUpgradeCostByLevel) < 25 {
		return errors.New("thousand tent camp requires 25 gold upgrade costs")
	}
	if cfg.CheckIntervalMinutes <= 0 {
		return errors.New("yellow turban check interval must be positive")
	}
	if cfg.MaxIncomingMarchesPerPlayer <= 0 {
		return errors.New("yellow turban max incoming marches must be positive")
	}
	if cfg.MarchSpeedMultiplier <= 0 {
		return errors.New("yellow turban march speed multiplier must be positive")
	}
	enabledLevels := 0
	for _, level := range cfg.RiskLevels {
		if !level.Enabled {
			continue
		}
		if level.ID <= 0 || level.Name == "" || level.MinPressure < 1 || level.MaxRatio <= 0 || level.MaxRatio < level.MinRatio {
			return errors.New("invalid yellow turban risk level")
		}
		enabledLevels++
	}
	if enabledLevels == 0 {
		return errors.New("yellow turban requires at least one enabled risk level")
	}
	return nil
}

// normalizeYellowTurbanConfig 补齐默认值并稳定排序。
func normalizeYellowTurbanConfig(cfg YellowTurbanConfig) YellowTurbanConfig {
	def := defaultYellowTurbanConfig()
	if cfg.ThousandTentCamp.BuildingType == "" {
		cfg.ThousandTentCamp.BuildingType = ThousandTentCampType
	}
	if cfg.ThousandTentCamp.Name == "" {
		cfg.ThousandTentCamp.Name = def.ThousandTentCamp.Name
	}
	if cfg.ThousandTentCamp.Description == "" {
		cfg.ThousandTentCamp.Description = def.ThousandTentCamp.Description
	}
	if len(cfg.ThousandTentCamp.CapacityByLevel) == 0 {
		cfg.ThousandTentCamp.CapacityByLevel = append([]int(nil), def.ThousandTentCamp.CapacityByLevel...)
	}
	if len(cfg.ThousandTentCamp.GoldUpgradeCostByLevel) == 0 {
		cfg.ThousandTentCamp.GoldUpgradeCostByLevel = append([]int(nil), def.ThousandTentCamp.GoldUpgradeCostByLevel...)
	}
	if cfg.CheckIntervalMinutes <= 0 {
		cfg.CheckIntervalMinutes = def.CheckIntervalMinutes
	}
	if cfg.MaxIncomingMarchesPerPlayer <= 0 {
		cfg.MaxIncomingMarchesPerPlayer = def.MaxIncomingMarchesPerPlayer
	}
	if cfg.MarchSpeedMultiplier <= 0 {
		cfg.MarchSpeedMultiplier = def.MarchSpeedMultiplier
	}
	if len(cfg.Regions) == 0 {
		cfg.Regions = append([]YellowTurbanRegionConfig(nil), def.Regions...)
	}
	if len(cfg.RiskLevels) == 0 {
		cfg.RiskLevels = append([]YellowTurbanRiskLevelConfig(nil), def.RiskLevels...)
	}
	for i := range cfg.Regions {
		if cfg.Regions[i].CityCount <= 0 {
			cfg.Regions[i].CityCount = 10
		}
		if cfg.Regions[i].MinUnitKinds <= 0 {
			cfg.Regions[i].MinUnitKinds = 2
		}
		if cfg.Regions[i].MaxUnitKinds < cfg.Regions[i].MinUnitKinds {
			cfg.Regions[i].MaxUnitKinds = cfg.Regions[i].MinUnitKinds
		}
	}
	sort.Slice(cfg.RiskLevels, func(i, j int) bool { return cfg.RiskLevels[i].ID < cfg.RiskLevels[j].ID })
	return cfg
}

// defaultYellowTurbanConfig 返回黄巾起义第一版默认配置。
func defaultYellowTurbanConfig() YellowTurbanConfig {
	return YellowTurbanConfig{
		Enabled:                     true,
		CheckIntervalMinutes:        10,
		MaxIncomingMarchesPerPlayer: 6,
		MarchSpeedMultiplier:        2,
		ThousandTentCamp: ThousandTentCampConfig{
			Enabled:      true,
			BuildingType: ThousandTentCampType,
			Name:         "千帐营",
			Description:  "提升可安全承载的口粮兵力，超出口粮上限后会引来黄巾军持续进攻。",
			CapacityByLevel: []int{
				100000, 300000, 600000, 1000000, 2000000,
				4000000, 8000000, 15000000, 30000000, 50000000,
				80000000, 120000000, 180000000, 280000000, 400000000,
				600000000, 900000000, 1200000000, 1600000000, 2000000000,
				2000000000, 4000000000, 6000000000, 8000000000, 10000000000,
			},
			GoldUpgradeCostByLevel: []int{
				0, 20, 40, 70, 100,
				150, 220, 300, 400, 520,
				650, 800, 980, 1180, 1380,
				1550, 1700, 1840, 1940, 2000,
				2200, 2400, 2600, 2800, 3000,
			},
		},
		Regions: []YellowTurbanRegionConfig{
			{ID: "wei", Name: "黄巾军·魏地", Faction: "wei", Enabled: true, CityCount: 10, MinUnitKinds: 2, MaxUnitKinds: 4},
			{ID: "shu", Name: "黄巾军·蜀地", Faction: "shu", Enabled: true, CityCount: 10, MinUnitKinds: 2, MaxUnitKinds: 4},
			{ID: "wu", Name: "黄巾军·吴地", Faction: "wu", Enabled: true, CityCount: 10, MinUnitKinds: 2, MaxUnitKinds: 4},
		},
		RiskLevels: []YellowTurbanRiskLevelConfig{
			{ID: 1, Name: "黄巾·流寇", Color: "#d9a400", MinPressure: 1, MaxPressure: 1.3, MinRatio: 0.08, MaxRatio: 0.12, MinUnitKinds: 2, MaxUnitKinds: 3, MaxIncoming: 6, Enabled: true},
			{ID: 2, Name: "黄巾·乱军", Color: "#f97316", MinPressure: 1.3, MaxPressure: 1.8, MinRatio: 0.15, MaxRatio: 0.22, MinUnitKinds: 2, MaxUnitKinds: 4, MaxIncoming: 6, Enabled: true},
			{ID: 3, Name: "黄巾·大营", Color: "#ea580c", MinPressure: 1.8, MaxPressure: 2.5, MinRatio: 0.25, MaxRatio: 0.38, MinUnitKinds: 3, MaxUnitKinds: 5, MaxIncoming: 6, Enabled: true},
			{ID: 4, Name: "黄巾·军团", Color: "#dc2626", MinPressure: 2.5, MaxPressure: 0, MinRatio: 0.45, MaxRatio: 0.65, MinUnitKinds: 3, MaxUnitKinds: 6, MaxIncoming: 6, Enabled: true},
		},
	}
}

// cloneYellowTurbanConfig 复制配置，避免调用方改写全局快照。
func cloneYellowTurbanConfig(cfg YellowTurbanConfig) YellowTurbanConfig {
	cfg.ThousandTentCamp.CapacityByLevel = append([]int(nil), cfg.ThousandTentCamp.CapacityByLevel...)
	cfg.ThousandTentCamp.GoldUpgradeCostByLevel = append([]int(nil), cfg.ThousandTentCamp.GoldUpgradeCostByLevel...)
	cfg.Regions = append([]YellowTurbanRegionConfig(nil), cfg.Regions...)
	cfg.RiskLevels = append([]YellowTurbanRiskLevelConfig(nil), cfg.RiskLevels...)
	for i := range cfg.Regions {
		cfg.Regions[i].UnitPool = append([]string(nil), cfg.Regions[i].UnitPool...)
		cfg.Regions[i].ExcludedUnits = append([]string(nil), cfg.Regions[i].ExcludedUnits...)
	}
	return cfg
}
