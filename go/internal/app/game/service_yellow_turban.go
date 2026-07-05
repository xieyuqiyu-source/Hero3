// 本文件实现黄巾起义玩法服务，负责口粮检测、派兵、行军和防守结算。
package game

import (
	"errors"
	"hash/fnv"
	"math"
	"math/rand"
	"sort"
	"strconv"
	"strings"
	"time"

	"hero3/internal/core/combat"
)

// GetYellowTurbanStatus 返回玩家黄巾起义据点状态。
func (s *Service) GetYellowTurbanStatus(playerID string) (YellowTurbanStatusResponse, error) {
	if err := s.SettleDueYellowTurbanMarches(playerID); err != nil {
		return YellowTurbanStatusResponse{}, err
	}
	now := time.Now().UTC()
	state, err := s.repo.GetState(strings.TrimSpace(playerID))
	if err != nil {
		return YellowTurbanStatusResponse{}, err
	}
	cfg := GetYellowTurbanConfig()
	pressure := CalculateFoodPressure(state, cfg)
	incoming, err := s.repo.ListYellowTurbanMarchesForPlayer(playerID)
	if err != nil {
		return YellowTurbanStatusResponse{}, err
	}
	active := make([]YellowTurbanMarch, 0, len(incoming))
	for _, march := range incoming {
		if yellowTurbanMarchActive(march.Status) {
			active = append(active, march)
		}
	}
	sortYellowTurbanMarches(active)
	return YellowTurbanStatusResponse{
		Enabled:              cfg.Enabled,
		FoodPressure:         pressure,
		CheckIntervalMinutes: cfg.CheckIntervalMinutes,
		NextCheckAt:          nextYellowTurbanCheckAt(now, cfg).Format(resourceDateLayout),
		IncomingCount:        len(active),
		MaxIncoming:          maxIncomingForPressure(cfg, pressure),
		Incoming:             active,
		Cities:               BuildYellowTurbanCities(cfg),
		ServerTime:           now.Format(resourceDateLayout),
	}, nil
}

// CheckYellowTurbanForPlayer 执行一次玩家黄巾检测，超限时派出一波黄巾。
func (s *Service) CheckYellowTurbanForPlayer(playerID string) (YellowTurbanCheckResult, error) {
	now := time.Now().UTC()
	if err := s.SettleDueYellowTurbanMarches(playerID); err != nil {
		return YellowTurbanCheckResult{}, err
	}
	cfg := GetYellowTurbanConfig()
	state, err := s.repo.GetState(strings.TrimSpace(playerID))
	if err != nil {
		return YellowTurbanCheckResult{}, err
	}
	pressure := CalculateFoodPressure(state, cfg)
	result := YellowTurbanCheckResult{
		Checked:      true,
		FoodPressure: pressure,
		MaxIncoming:  maxIncomingForPressure(cfg, pressure),
		ServerTime:   now.Format(resourceDateLayout),
	}
	if !cfg.Enabled {
		result.Reason = "黄巾检测未开启"
		return result, nil
	}
	if !pressure.OverCapacity {
		result.Reason = "口粮未超出千帐营承载上限"
		return result, nil
	}
	activeCount, err := s.repo.CountActiveYellowTurbanMarches(playerID)
	if err != nil {
		return YellowTurbanCheckResult{}, err
	}
	result.IncomingCount = activeCount
	if activeCount >= result.MaxIncoming {
		result.Reason = "已达到最大同时来袭路数"
		return result, nil
	}
	march, err := s.spawnYellowTurbanMarch(state, pressure, cfg, now)
	if err != nil {
		return YellowTurbanCheckResult{}, err
	}
	result.Spawned = true
	result.March = &march
	result.IncomingCount = activeCount + 1
	return result, nil
}

// CheckYellowTurbanForAllPlayers 扫描所有玩家并执行一次黄巾检测。
func (s *Service) CheckYellowTurbanForAllPlayers() ([]YellowTurbanCheckResult, error) {
	players, err := s.repo.ListAllPlayers()
	if err != nil {
		return nil, err
	}
	results := make([]YellowTurbanCheckResult, 0, len(players))
	for _, player := range players {
		result, err := s.CheckYellowTurbanForPlayer(player.ID)
		if err != nil {
			result = YellowTurbanCheckResult{Checked: false, Reason: err.Error(), ServerTime: time.Now().UTC().Format(resourceDateLayout)}
		}
		results = append(results, result)
	}
	return results, nil
}

// SettleDueYellowTurbanMarches 结算指定玩家已经到达的黄巾来袭。
func (s *Service) SettleDueYellowTurbanMarches(playerID string) error {
	now := time.Now().UTC()
	marches, err := s.repo.ListDueYellowTurbanMarches(playerID, now)
	if err != nil {
		return err
	}
	for _, march := range marches {
		if _, err := s.ResolveYellowTurbanMarch(march.ID); err != nil && !errors.Is(err, ErrYellowTurbanMarchNotFound) {
			return err
		}
	}
	return nil
}

// ResolveYellowTurbanMarch 结算一条已经到达的黄巾来袭。
func (s *Service) ResolveYellowTurbanMarch(marchID string) (BattleReport, error) {
	now := time.Now().UTC()
	march, err := s.repo.GetYellowTurbanMarch(strings.TrimSpace(marchID))
	if err != nil {
		return BattleReport{}, err
	}
	if march.Status != YellowTurbanMarchStatusMarching {
		return BattleReport{}, nil
	}
	arrivesAt, err := time.Parse(resourceDateLayout, march.ArrivesAt)
	if err == nil && arrivesAt.After(now) {
		return BattleReport{}, ErrPvpMarchNotReady
	}
	if _, err := s.repo.UpdateYellowTurbanMarch(march.ID, now, func(m *YellowTurbanMarch) error {
		m.Status = YellowTurbanMarchStatusResolving
		return nil
	}); err != nil {
		return BattleReport{}, err
	}
	var report BattleReport
	updated, err := s.repo.UpdateCombatState(march.TargetPlayerID, CombatAssetScope{SkipInventory: true}, now, func(state *GameState) error {
		EnsureGeneralRoster(state, now)
		defenderBefore := armySliceToMap(state.Army)
		attackerUnits, err := buildSimulatedCombatUnits(march.SourceFaction, march.Troops, now)
		if err != nil {
			return err
		}
		defenderUnits, err := buildSimulatedCombatUnits(state.Player.Faction, defenderBefore, now, CollectModifierSources(state)...)
		if errors.Is(err, ErrNoUnitsSelected) {
			defenderUnits = []combat.Unit{}
		} else if err != nil {
			return err
		}
		battle := combat.Resolve(combat.CombatInput{
			RuleID:      activeCombatRuleID(combat.ScenePVPAttack),
			Attacker:    combat.Army{Faction: march.SourceFaction, Units: attackerUnits},
			Defender:    combat.Army{Faction: state.Player.Faction, Units: defenderUnits},
			WallLevel:   pvpDefenderCityWallLevel(state),
			WallFaction: state.Player.Faction,
		})
		defenderLosses := combatLossMap(battle.DefenderLosses)
		attackerLosses := combatLossMap(battle.AttackerLosses)
		applyArmyLosses(state, defenderLosses)
		nowText := now.Format(resourceDateLayout)
		reportResult := "attacker_victory"
		if battle.Winner == "defender" {
			reportResult = "defender_victory"
		} else if battle.Winner == "draw" {
			reportResult = "draw"
		}
		report = BattleReport{
			ID:                "br_yt_def_" + randomID(8),
			PlayerID:          state.Player.ID,
			OwnerPlayerID:     state.Player.ID,
			ViewType:          ReportViewDefense,
			SourceType:        ReportSourceYellowTurban,
			BattleType:        BattleTypeYellowTurban,
			Title:             march.SourceName + " 进攻 " + state.Player.Nickname,
			Summary:           march.RiskLevelName + " 已抵达城下",
			PlayerFaction:     state.Player.Faction,
			PlayerName:        state.Player.Nickname,
			TargetID:          march.SourceCityID,
			TargetName:        march.SourceName,
			Type:              "defense",
			Result:            invertPvpReportResult(reportResult),
			PlayerPower:       int(battle.DefensePower),
			EnemyPower:        int(battle.AttackPower),
			DispatchedUnits:   defenderBefore,
			LostUnits:         defenderLosses,
			DefenderFaction:   march.SourceFaction,
			DefenderUnits:     cloneStringIntMap(march.Troops),
			DefenderLostUnits: attackerLosses,
			DefenderRevealed:  true,
			DefenderResources: map[string]int{},
			Rewards:           map[string]int{},
			Read:              false,
			CreatedAt:         nowText,
		}
		report = NormalizeBattleReport(report)
		state.ServerTime = nowText
		return nil
	})
	if err != nil {
		_, _ = s.repo.UpdateYellowTurbanMarch(march.ID, now, func(m *YellowTurbanMarch) error {
			m.Status = YellowTurbanMarchStatusFailed
			m.Error = err.Error()
			return nil
		})
		return BattleReport{}, err
	}
	_ = updated
	if _, err := s.CreateBattleReports(BattleReportCreateInput{
		SourceType: ReportSourceYellowTurban,
		SourceID:   march.ID,
		BattleType: BattleTypeYellowTurban,
		Result:     report.Result,
		Reports:    []BattleReport{report},
		Extra: map[string]interface{}{
			"yellowTurbanMarchId": march.ID,
			"sourceCityId":        march.SourceCityID,
			"riskLevelId":         march.RiskLevelID,
		},
	}); err != nil {
		return BattleReport{}, err
	}
	_, err = s.repo.UpdateYellowTurbanMarch(march.ID, now, func(m *YellowTurbanMarch) error {
		m.Status = YellowTurbanMarchStatusResolved
		m.ResolvedAt = now.Format(resourceDateLayout)
		m.DefenderReportID = report.ID
		return nil
	})
	if err != nil {
		return BattleReport{}, err
	}
	return report, nil
}

// ResolveYellowTurbanMarchForPlayer 校验来袭归属后结算黄巾来袭。
func (s *Service) ResolveYellowTurbanMarchForPlayer(playerID string, marchID string) (BattleReport, error) {
	march, err := s.repo.GetYellowTurbanMarch(strings.TrimSpace(marchID))
	if err != nil {
		return BattleReport{}, err
	}
	if strings.TrimSpace(march.TargetPlayerID) != strings.TrimSpace(playerID) {
		return BattleReport{}, ErrYellowTurbanMarchNotFound
	}
	return s.ResolveYellowTurbanMarch(march.ID)
}

// CalculateFoodPressure 根据玩家兵力和千帐营等级计算口粮压力。
func CalculateFoodPressure(state GameState, cfg YellowTurbanConfig) FoodPressureState {
	currentFood := CurrentArmyFood(state)
	level := thousandTentCampLevel(state.Buildings)
	capacity := thousandTentCampCapacity(level, cfg)
	pressure := 0.0
	if capacity > 0 {
		pressure = float64(currentFood) / float64(capacity)
	}
	result := FoodPressureState{
		CurrentFood:       currentFood,
		FoodCapacity:      capacity,
		Pressure:          pressure,
		OverCapacity:      capacity > 0 && currentFood > capacity,
		ThousandTentLevel: level,
	}
	if result.OverCapacity {
		if levelCfg, ok := yellowTurbanRiskLevelForPressure(cfg, pressure); ok {
			result.RiskLevelID = levelCfg.ID
			result.RiskLevelName = levelCfg.Name
			result.RiskColor = levelCfg.Color
		}
	}
	return result
}

// CurrentArmyFood 计算玩家当前全部战斗兵力口粮。
func CurrentArmyFood(state GameState) int {
	total := 0
	for _, unit := range state.Army {
		if unit.Amount <= 0 {
			continue
		}
		cfg, ok := GetUnitConfig(state.Player.Faction, unit.UnitType)
		if !ok || isNonCombatUnit(cfg) {
			continue
		}
		total += unit.Amount * cfg.Stats["upkeep"]
	}
	return total
}

// BuildYellowTurbanCities 根据配置生成稳定黄巾城池。
func BuildYellowTurbanCities(cfg YellowTurbanConfig) []YellowTurbanCity {
	cities := []YellowTurbanCity{}
	for _, region := range cfg.Regions {
		if !region.Enabled {
			continue
		}
		for i := 1; i <= region.CityCount; i++ {
			x, y := yellowTurbanCityCoordinate(region.ID, i)
			id := "yt_city_" + region.ID + "_" + strconv.Itoa(i)
			cities = append(cities, YellowTurbanCity{
				ID:       id,
				Name:     region.Name,
				RegionID: region.ID,
				Faction:  region.Faction,
				WorldID:  defaultWorldID,
				X:        x,
				Y:        y,
				Enabled:  true,
			})
		}
	}
	return cities
}

// spawnYellowTurbanMarch 生成并保存一条黄巾来袭。
func (s *Service) spawnYellowTurbanMarch(state GameState, pressure FoodPressureState, cfg YellowTurbanConfig, now time.Time) (YellowTurbanMarch, error) {
	levelCfg, ok := yellowTurbanRiskLevelForPressure(cfg, pressure.Pressure)
	if !ok {
		return YellowTurbanMarch{}, errors.New("no yellow turban risk level matched")
	}
	cities := BuildYellowTurbanCities(cfg)
	if len(cities) == 0 {
		return YellowTurbanMarch{}, errors.New("no yellow turban city enabled")
	}
	source := pickYellowTurbanCity(cities, state.Player.ID, now)
	troops, err := generateYellowTurbanTroops(source.Faction, pressure.CurrentFood, levelCfg, cfg, source.RegionID, now)
	if err != nil {
		return YellowTurbanMarch{}, err
	}
	playerPos, err := s.ensureWorldPosition(state.Player.ID, "yellow_turban_check", nil)
	if err != nil {
		return YellowTurbanMarch{}, err
	}
	distance := worldMapDistance(WorldCoordinate{X: source.X, Y: source.Y}, WorldCoordinate{X: playerPos.X, Y: playerPos.Y})
	duration := yellowTurbanMarchSeconds(distance, troops, source.Faction, cfg)
	startedAt := now.Format(resourceDateLayout)
	arrivesAt := now.Add(time.Duration(duration) * time.Second).Format(resourceDateLayout)
	march := YellowTurbanMarch{
		ID:              "yt_march_" + randomID(12),
		TargetPlayerID:  state.Player.ID,
		SourceCityID:    source.ID,
		SourceName:      source.Name,
		SourceFaction:   source.Faction,
		SourceRegionID:  source.RegionID,
		RiskLevelID:     levelCfg.ID,
		RiskLevelName:   levelCfg.Name,
		PlayerFood:      pressure.CurrentFood,
		FoodCapacity:    pressure.FoodCapacity,
		Pressure:        pressure.Pressure,
		Troops:          troops,
		Status:          YellowTurbanMarchStatusMarching,
		DurationSeconds: duration,
		StartedAt:       startedAt,
		ArrivesAt:       arrivesAt,
		CreatedAt:       startedAt,
		UpdatedAt:       startedAt,
	}
	return s.repo.CreateYellowTurbanMarch(march)
}

// yellowTurbanRiskLevelForPressure 匹配口粮压力档位。
func yellowTurbanRiskLevelForPressure(cfg YellowTurbanConfig, pressure float64) (YellowTurbanRiskLevelConfig, bool) {
	for _, level := range cfg.RiskLevels {
		if !level.Enabled {
			continue
		}
		if pressure >= level.MinPressure && (level.MaxPressure <= 0 || pressure < level.MaxPressure) {
			return level, true
		}
	}
	return YellowTurbanRiskLevelConfig{}, false
}

// thousandTentCampLevel 返回千帐营等级，旧存档缺失时按 1 级兜底。
func thousandTentCampLevel(buildings []Building) int {
	level := 1
	for _, building := range buildings {
		if building.Type == ThousandTentCampType && building.Level > level {
			level = building.Level
		}
	}
	return level
}

// thousandTentCampCapacity 返回千帐营等级对应口粮上限。
func thousandTentCampCapacity(level int, cfg YellowTurbanConfig) int {
	if level <= 0 {
		level = 1
	}
	if level > len(cfg.ThousandTentCamp.CapacityByLevel) {
		level = len(cfg.ThousandTentCamp.CapacityByLevel)
	}
	return cfg.ThousandTentCamp.CapacityByLevel[level-1]
}

// maxIncomingForPressure 返回当前压力档位的最大来袭路数。
func maxIncomingForPressure(cfg YellowTurbanConfig, pressure FoodPressureState) int {
	maxIncoming := cfg.MaxIncomingMarchesPerPlayer
	if level, ok := yellowTurbanRiskLevelForPressure(cfg, pressure.Pressure); ok && level.MaxIncoming > 0 {
		maxIncoming = level.MaxIncoming
	}
	if maxIncoming <= 0 {
		maxIncoming = 6
	}
	return maxIncoming
}

// generateYellowTurbanTroops 按当前总口粮和风险倍率生成黄巾兵力快照。
func generateYellowTurbanTroops(faction string, currentFood int, level YellowTurbanRiskLevelConfig, cfg YellowTurbanConfig, regionID string, now time.Time) (map[string]int, error) {
	units := availableYellowTurbanUnits(faction, cfg, regionID)
	if len(units) == 0 {
		return nil, ErrUnitNotFound
	}
	r := rand.New(rand.NewSource(now.UnixNano() + int64(currentFood) + int64(level.ID)))
	ratio := level.MinRatio
	if level.MaxRatio > level.MinRatio {
		ratio += r.Float64() * (level.MaxRatio - level.MinRatio)
	}
	foodBudget := int(math.Round(float64(currentFood) * ratio))
	if foodBudget <= 0 {
		foodBudget = 1
	}
	kinds := clampInt(level.MinUnitKinds, 1, len(units))
	if level.MaxUnitKinds > kinds {
		kinds = clampInt(kinds+r.Intn(level.MaxUnitKinds-kinds+1), 1, len(units))
	}
	r.Shuffle(len(units), func(i, j int) { units[i], units[j] = units[j], units[i] })
	selected := units[:kinds]
	troops := map[string]int{}
	remaining := foodBudget
	for i, unitID := range selected {
		unitCfg, _ := GetUnitConfig(faction, unitID)
		upkeep := unitCfg.Stats["upkeep"]
		if upkeep <= 0 {
			continue
		}
		share := remaining
		if i < len(selected)-1 {
			share = remaining / (len(selected) - i)
		}
		count := share / upkeep
		if count <= 0 {
			count = 1
		}
		troops[unitID] += count
		remaining -= count * upkeep
	}
	if len(troops) == 0 {
		return nil, ErrNoUnitsSelected
	}
	return troops, nil
}

// availableYellowTurbanUnits 返回指定地区可用战斗兵种。
func availableYellowTurbanUnits(faction string, cfg YellowTurbanConfig, regionID string) []string {
	include := map[string]struct{}{}
	exclude := map[string]struct{}{}
	for _, region := range cfg.Regions {
		if region.ID != regionID {
			continue
		}
		for _, unitID := range region.UnitPool {
			include[strings.TrimSpace(unitID)] = struct{}{}
		}
		for _, unitID := range region.ExcludedUnits {
			exclude[strings.TrimSpace(unitID)] = struct{}{}
		}
		break
	}
	factionUnits := GetFactionUnits(faction)
	units := []string{}
	for unitID, unitCfg := range factionUnits {
		if _, blocked := exclude[unitID]; blocked {
			continue
		}
		if len(include) > 0 {
			if _, ok := include[unitID]; !ok {
				continue
			}
		}
		if isNonCombatUnit(unitCfg) || unitCfg.Role == "scout" {
			continue
		}
		units = append(units, unitID)
	}
	sort.Strings(units)
	return units
}

// yellowTurbanMarchSeconds 计算黄巾行军时间。
func yellowTurbanMarchSeconds(distance int, troops map[string]int, faction string, cfg YellowTurbanConfig) int {
	duration, _, err := calculatePvpMarchTravel(distance, faction, troops, time.Now().UTC(), nil)
	if err != nil || duration <= 0 {
		duration = defaultPvpMarchSeconds
	}
	multiplier := cfg.MarchSpeedMultiplier
	if multiplier <= 0 {
		multiplier = 2
	}
	duration = int(math.Ceil(float64(duration) / multiplier))
	if duration < 60 {
		duration = 60
	}
	return duration
}

// pickYellowTurbanCity 为本次检测选择一个黄巾来源城池。
func pickYellowTurbanCity(cities []YellowTurbanCity, playerID string, now time.Time) YellowTurbanCity {
	if len(cities) == 1 {
		return cities[0]
	}
	hash := fnv.New64a()
	_, _ = hash.Write([]byte(playerID + now.Format("200601021504")))
	index := int(hash.Sum64() % uint64(len(cities)))
	return cities[index]
}

// yellowTurbanCityCoordinate 生成稳定的黄巾城池坐标。
func yellowTurbanCityCoordinate(regionID string, index int) (int, int) {
	hash := fnv.New64a()
	_, _ = hash.Write([]byte(regionID + ":" + strconv.Itoa(index)))
	value := hash.Sum64()
	return int(value % defaultWorldWidth), int((value / defaultWorldWidth) % defaultWorldHeight)
}

// nextYellowTurbanCheckAt 返回展示用的下一次检测时间。
func nextYellowTurbanCheckAt(now time.Time, cfg YellowTurbanConfig) time.Time {
	interval := time.Duration(cfg.CheckIntervalMinutes) * time.Minute
	if interval <= 0 {
		interval = 10 * time.Minute
	}
	epoch := now.Unix()
	next := ((epoch / int64(interval.Seconds())) + 1) * int64(interval.Seconds())
	return time.Unix(next, 0).UTC()
}
