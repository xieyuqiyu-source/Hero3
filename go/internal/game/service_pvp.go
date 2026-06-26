// Hero3 PVP 与行军服务，负责玩家目标、出征行军、加速和到达结算。
package game

import (
	"math"
	"sort"
	"strings"
	"time"

	"hero3/internal/combat"
	"hero3/internal/general"
)

// ListPvpTargets 返回当前玩家可攻击目标，排除自己和同账号存档。
func (s *Service) ListPvpTargets(playerID string, page int, pageSize int) (PvpTargetPage, error) {
	playerID = strings.TrimSpace(playerID)
	if playerID == "" {
		return PvpTargetPage{}, ErrPlayerNotFound
	}
	attackerAccountID, err := s.repo.GetAccountIDByPlayerID(playerID)
	if err != nil {
		return PvpTargetPage{}, err
	}
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 50 {
		pageSize = 50
	}

	accounts, err := s.repo.ListAccounts()
	if err != nil {
		return PvpTargetPage{}, err
	}
	targets := []PvpTarget{}
	for _, account := range accounts {
		if account.ID == attackerAccountID {
			continue
		}
		for _, player := range account.Players {
			if player.ID == playerID {
				continue
			}
			targets = append(targets, PvpTarget{
				PlayerID:      player.ID,
				Nickname:      player.Nickname,
				Faction:       player.Faction,
				TotalArmy:     player.TotalArmy,
				BuildingLevel: player.BuildingLevel,
				UpdatedAt:     player.UpdatedAt,
			})
		}
	}
	sort.SliceStable(targets, func(i, j int) bool {
		return targets[i].UpdatedAt > targets[j].UpdatedAt
	})

	total := len(targets)
	start := (page - 1) * pageSize
	if start >= total {
		return PvpTargetPage{Targets: []PvpTarget{}, Page: page, PageSize: pageSize, Total: total}, nil
	}
	end := start + pageSize
	if end > total {
		end = total
	}
	return PvpTargetPage{Targets: targets[start:end], Page: page, PageSize: pageSize, Total: total}, nil
}

// GetMarchConfig 返回当前行军配置。
func (s *Service) GetMarchConfig() MarchConfig {
	return currentBalance().March
}

// UpdateMarchConfig 更新行军配置并持久化到 balance 文件。
func (s *Service) UpdateMarchConfig(config MarchConfig) (MarchConfig, error) {
	balance := currentBalance()
	balance.March = normalizeMarchConfig(config)
	if err := s.UpdateBalance(balance); err != nil {
		return MarchConfig{}, err
	}
	return currentBalance().March, nil
}

// GetPvpTarget 返回目标玩家简要信息，不暴露完整军队。
func (s *Service) GetPvpTarget(attackerPlayerID string, targetPlayerID string) (PvpTarget, error) {
	page, err := s.ListPvpTargets(attackerPlayerID, 1, 50)
	if err != nil {
		return PvpTarget{}, err
	}
	for pageIndex := 1; ; pageIndex++ {
		if pageIndex > 1 {
			page, err = s.ListPvpTargets(attackerPlayerID, pageIndex, 50)
			if err != nil {
				return PvpTarget{}, err
			}
		}
		for _, target := range page.Targets {
			if target.PlayerID == targetPlayerID {
				return target, nil
			}
		}
		if pageIndex*page.PageSize >= page.Total {
			break
		}
	}
	return PvpTarget{}, ErrPlayerNotFound
}

// StartPvpAttack 校验并扣除主军队兵力，创建 PVP 出征行军。
func (s *Service) StartPvpAttack(req AttackPlayerRequest) (AttackPlayerResponse, error) {
	playerID := strings.TrimSpace(req.PlayerID)
	targetPlayerID := strings.TrimSpace(req.TargetPlayerID)
	if playerID == "" {
		return AttackPlayerResponse{}, ErrPlayerNotFound
	}
	if targetPlayerID == "" || targetPlayerID == playerID {
		return AttackPlayerResponse{}, ErrInvalidPlayerTarget
	}

	attackerLock := s.getPlayerLock(playerID)
	attackerLock.Lock()
	defer attackerLock.Unlock()

	attackerAccountID, err := s.repo.GetAccountIDByPlayerID(playerID)
	if err != nil {
		return AttackPlayerResponse{}, err
	}
	defenderAccountID, err := s.repo.GetAccountIDByPlayerID(targetPlayerID)
	if err != nil {
		return AttackPlayerResponse{}, err
	}
	if attackerAccountID == defenderAccountID {
		return AttackPlayerResponse{}, ErrSameAccountTarget
	}

	attackerState, err := s.repo.GetState(playerID)
	if err != nil {
		return AttackPlayerResponse{}, err
	}
	if isNpcTargetID(attackerState, targetPlayerID) {
		return AttackPlayerResponse{}, ErrInvalidPlayerTarget
	}
	defenderState, err := s.repo.GetState(targetPlayerID)
	if err != nil {
		return AttackPlayerResponse{}, err
	}

	now := time.Now()
	attackerState, _ = settleResources(attackerState, now)
	attackerUnits, err := validateAndConsumeArmy(&attackerState, req.Units)
	if err != nil {
		return AttackPlayerResponse{}, err
	}
	dispatched := combatUnitsToAmountMap(attackerUnits)
	slowestSpeed := slowestUnitSpeed(dispatched)
	if slowestSpeed <= 0 {
		return AttackPlayerResponse{}, ErrInvalidMarch
	}

	marchConfig := currentBalance().March
	duration := calculateMarchDurationSeconds(slowestSpeed, marchConfig)
	nowStr := now.UTC().Format(resourceDateLayout)
	arrivesAt := now.UTC().Add(time.Duration(duration) * time.Second).Format(resourceDateLayout)
	march := PvpMarch{
		ID:                            "march_" + randomID(12),
		AttackerPlayerID:              attackerState.Player.ID,
		AttackerName:                  attackerState.Player.Nickname,
		AttackerFaction:               attackerState.Player.Faction,
		DefenderPlayerID:              defenderState.Player.ID,
		DefenderName:                  defenderState.Player.Nickname,
		DefenderFaction:               defenderState.Player.Faction,
		Type:                          MarchTypePvpAttack,
		Units:                         dispatched,
		SlowestSpeed:                  slowestSpeed,
		DurationSeconds:               duration,
		StartedAt:                     nowStr,
		ArrivesAt:                     arrivesAt,
		Status:                        MarchStatusMarching,
		MaxDurationSeconds:            marchConfig.MaxDurationSeconds,
		MinDurationSeconds:            marchConfig.MinDurationSeconds,
		SpeedScale:                    marchConfig.SpeedScale,
		AccelerateCostCityGold:        marchConfig.Accelerate.CostCityGold,
		AccelerateReduceRate:          marchConfig.Accelerate.ReduceRate,
		AccelerateMinRemainingSeconds: marchConfig.Accelerate.MinRemainingSeconds,
		CreatedAt:                     nowStr,
		UpdatedAt:                     nowStr,
	}

	attackerState.ServerTime = nowStr
	if err := s.repo.SaveState(attackerState, now); err != nil {
		return AttackPlayerResponse{}, err
	}
	if err := s.repo.CreateMarch(march); err != nil {
		return AttackPlayerResponse{}, err
	}
	s.attachReportSummary(&attackerState, attackerState.Player.ID)
	return AttackPlayerResponse{
		MarchID:         march.ID,
		StartedAt:       march.StartedAt,
		ArrivesAt:       march.ArrivesAt,
		DurationSeconds: march.DurationSeconds,
		SlowestSpeed:    march.SlowestSpeed,
		Units:           march.Units,
		March:           march,
		State:           attackerState,
	}, nil
}

// ListPvpMarches 返回玩家相关行军，incoming 不暴露敌方兵力。
func (s *Service) ListPvpMarches(playerID string, now time.Time) ([]PvpMarchView, error) {
	playerID = strings.TrimSpace(playerID)
	if playerID == "" {
		return nil, ErrPlayerNotFound
	}
	if now.IsZero() {
		now = time.Now()
	}
	_ = s.SettleDueMarches(now)
	marches, err := s.repo.ListMarchesForPlayer(playerID)
	if err != nil {
		return nil, err
	}
	views := make([]PvpMarchView, 0, len(marches))
	for _, march := range marches {
		views = append(views, buildMarchView(march, playerID, now))
	}
	return views, nil
}

// AcceleratePvpMarch 花费城金让当前剩余时间按行军固化配置缩短。
func (s *Service) AcceleratePvpMarch(playerID string, marchID string, now time.Time) (PvpMarchView, GameState, error) {
	playerID = strings.TrimSpace(playerID)
	marchID = strings.TrimSpace(marchID)
	if playerID == "" {
		return PvpMarchView{}, GameState{}, ErrPlayerNotFound
	}
	if now.IsZero() {
		now = time.Now()
	}
	lock := s.getPlayerLock(playerID)
	lock.Lock()
	defer lock.Unlock()

	march, err := s.repo.GetMarchByID(marchID)
	if err != nil {
		return PvpMarchView{}, GameState{}, err
	}
	if march.AttackerPlayerID != playerID || march.Status != MarchStatusMarching {
		return PvpMarchView{}, GameState{}, ErrMarchNotAccelerable
	}
	arrivesAt, ok := parseMarchTime(march.ArrivesAt)
	if !ok || !arrivesAt.After(now.UTC()) {
		return PvpMarchView{}, GameState{}, ErrMarchNotAccelerable
	}
	remaining := int(math.Ceil(arrivesAt.Sub(now.UTC()).Seconds()))
	minRemaining := march.AccelerateMinRemainingSeconds
	if minRemaining <= 0 {
		minRemaining = 300
	}
	if remaining <= minRemaining {
		return PvpMarchView{}, GameState{}, ErrMarchNotAccelerable
	}
	reduceRate := march.AccelerateReduceRate
	if reduceRate <= 0 || reduceRate >= 1 {
		reduceRate = 0.5
	}
	nextRemaining := int(math.Ceil(float64(remaining) * reduceRate))
	if nextRemaining < minRemaining {
		nextRemaining = minRemaining
	}
	if nextRemaining >= remaining {
		return PvpMarchView{}, GameState{}, ErrMarchNotAccelerable
	}

	cost := march.AccelerateCostCityGold
	if cost <= 0 {
		cost = 50
	}
	newCityGold, err := s.repo.DeductCityGold(playerID, cost)
	if err != nil {
		return PvpMarchView{}, GameState{}, err
	}
	state, err := s.repo.GetState(playerID)
	if err != nil {
		return PvpMarchView{}, GameState{}, err
	}
	state.CityGold = FlexInt(newCityGold)
	march.AcceleratedTimes++
	march.ArrivesAt = now.UTC().Add(time.Duration(nextRemaining) * time.Second).Format(resourceDateLayout)
	march.UpdatedAt = now.UTC().Format(resourceDateLayout)
	if err := s.repo.UpdateMarch(march); err != nil {
		return PvpMarchView{}, GameState{}, err
	}
	s.recordLedger(GoldLedgerEntry{
		PlayerID:     playerID,
		Currency:     LedgerCurrencyCityGold,
		Direction:    LedgerDirectionDebit,
		Amount:       cost,
		BalanceAfter: newCityGold,
		RefType:      "pvp_march_accelerate",
		RefID:        march.ID,
		Reason:       "pvp_march_accelerate",
		CreatedAt:    now.UTC().Format(resourceDateLayout),
	})
	return buildMarchView(march, playerID, now), state, nil
}

// SettleDueMarches 统一结算所有到期行军，幂等跳过非 marching 行军。
func (s *Service) SettleDueMarches(now time.Time) error {
	if now.IsZero() {
		now = time.Now()
	}
	due, err := s.repo.ListDueMarches(now, 100)
	if err != nil {
		return err
	}
	for _, march := range due {
		if err := s.settlePvpMarch(march.ID, now); err != nil {
			continue
		}
	}
	return nil
}

// settlePvpMarch 结算单条 PVP 行军。
func (s *Service) settlePvpMarch(marchID string, now time.Time) error {
	claimed, err := s.repo.ClaimMarchForResolution(marchID, now)
	if err != nil {
		return err
	}
	unlock := s.lockPlayers(claimed.AttackerPlayerID, claimed.DefenderPlayerID)
	defer unlock()

	attackerState, err := s.repo.GetState(claimed.AttackerPlayerID)
	if err != nil {
		return s.restoreMarching(claimed, now, err)
	}
	defenderState, err := s.repo.GetState(claimed.DefenderPlayerID)
	if err != nil {
		return s.restoreMarching(claimed, now, err)
	}
	if attackerState.General != nil {
		applyHeroConfigToGeneral(attackerState.General)
	}
	if defenderState.General != nil {
		applyHeroConfigToGeneral(defenderState.General)
	}
	attackerState, _ = settleResources(attackerState, now)
	defenderState, _ = settleResources(defenderState, now)

	attackerUnits, err := buildDispatchedCombatUnits(&attackerState, claimed.Units, now)
	if err != nil {
		return s.restoreMarching(claimed, now, err)
	}
	defenderUnits, defenderSources := buildPlayerDefenseCombatUnits(defenderState, now)
	ruleID := "official_attack"
	attackerArmy := buildCombatArmy(attackerState.Player.Faction, attackerUnits)
	defenderArmy := buildCombatArmy(defenderState.Player.Faction, defenderUnits)
	attackerTraits := buildActiveTraits(attackerState.General)
	defenderTraits := buildActiveTraits(defenderState.General)

	beforeAttackCtx := &general.BeforeBattleContext{
		Attacker:          &attackerArmy,
		Defender:          &defenderArmy,
		AttackerOwnsTrait: true,
		DefenderOwnsTrait: false,
		IsPvP:             true,
		SameFaction:       attackerState.Player.Faction == defenderState.Player.Faction,
	}
	general.Dispatch(beforeAttackCtx, attackerTraits)
	capturedToArmy := applyPvpCapturedUnits(&attackerState, &defenderState, defenderSources, beforeAttackCtx.CapturedToArmy, false)
	capturedToGarrison := applyPvpCapturedUnits(&attackerState, &defenderState, defenderSources, beforeAttackCtx.CapturedToGarrison, true)

	beforeDefenseCtx := &general.BeforeBattleContext{
		Attacker:          &attackerArmy,
		Defender:          &defenderArmy,
		AttackerOwnsTrait: false,
		DefenderOwnsTrait: true,
		IsPvP:             true,
		SameFaction:       attackerState.Player.Faction == defenderState.Player.Faction,
	}
	general.Dispatch(beforeDefenseCtx, defenderTraits)

	result := combat.Resolve(combat.CombatInput{
		RuleID:   ruleID,
		Attacker: attackerArmy,
		Defender: defenderArmy,
	})
	afterAttackCtx := &general.AfterCombatResolveContext{
		Result:            &result,
		Attacker:          &attackerArmy,
		Defender:          &defenderArmy,
		AttackerOwnsTrait: true,
		DefenderOwnsTrait: false,
		IsAttackerOnly:    true,
	}
	general.Dispatch(afterAttackCtx, attackerTraits)
	afterDefenseCtx := &general.AfterCombatResolveContext{
		Result:            &result,
		Attacker:          &attackerArmy,
		Defender:          &defenderArmy,
		AttackerOwnsTrait: false,
		DefenderOwnsTrait: true,
		IsAttackerOnly:    false,
	}
	general.Dispatch(afterDefenseCtx, defenderTraits)

	report := applyPlayerBattleResult(&attackerState, &defenderState, result, attackerUnits, defenderSources, "pvp_attack", now)
	report.CapturedUnits = capturedToArmy
	report.CapturedToGarrison = capturedToGarrison
	mergeTraitOutcomes(&report, beforeAttackCtx.Triggered)
	mergeTraitOutcomes(&report, beforeDefenseCtx.Triggered)
	mergeTraitOutcomes(&report, afterAttackCtx.Triggered)
	mergeTraitOutcomes(&report, afterDefenseCtx.Triggered)
	applyPvpAfterBattleTraits(&report, &attackerState, &defenderState, attackerTraits, defenderTraits)
	attackerExp := calculatePvpGeneralExpFromLosses(result.DefenderLosses)
	attackerExpResult := applyGeneralBattleExp(attackerState.General, attackerExp)
	if attackerExpResult.Gained > 0 {
		report.GeneralExpGained = attackerExpResult.Gained
		report.GeneralLevelBefore = attackerExpResult.LevelBefore
		report.GeneralLevelAfter = attackerExpResult.LevelAfter
	}
	defenderReport := report
	defenderReport.ID = "br_" + randomID(8)
	defenderReport.PlayerID = defenderState.Player.ID
	defenderReport.PlayerFaction = defenderState.Player.Faction
	defenderReport.PlayerName = defenderState.Player.Nickname
	defenderReport.TargetID = attackerState.Player.ID
	defenderReport.TargetName = attackerState.Player.Nickname
	defenderExp := calculatePvpGeneralExpFromLosses(result.AttackerLosses)
	defenderExpResult := applyGeneralBattleExp(defenderState.General, defenderExp)
	if defenderExpResult.Gained > 0 {
		defenderReport.GeneralExpGained = defenderExpResult.Gained
		defenderReport.GeneralLevelBefore = defenderExpResult.LevelBefore
		defenderReport.GeneralLevelAfter = defenderExpResult.LevelAfter
	} else {
		defenderReport.GeneralExpGained = 0
		defenderReport.GeneralLevelBefore = 0
		defenderReport.GeneralLevelAfter = 0
	}

	nowStr := now.UTC().Format(resourceDateLayout)
	attackerState.ServerTime = nowStr
	defenderState.ServerTime = nowStr
	if err := s.repo.SaveStates([]GameState{attackerState, defenderState}, now); err != nil {
		return s.restoreMarching(claimed, now, err)
	}
	if err := s.repo.SaveReport(report); err != nil {
		return s.restoreMarching(claimed, now, err)
	}
	if err := s.repo.SaveReport(defenderReport); err != nil {
		return s.restoreMarching(claimed, now, err)
	}
	claimed.Status = MarchStatusResolved
	claimed.ResolvedAt = nowStr
	claimed.AttackerReportID = report.ID
	claimed.DefenderReportID = defenderReport.ID
	claimed.UpdatedAt = nowStr
	return s.repo.UpdateMarch(claimed)
}

func (s *Service) restoreMarching(march PvpMarch, now time.Time, cause error) error {
	march.Status = MarchStatusMarching
	march.UpdatedAt = now.UTC().Format(resourceDateLayout)
	_ = s.repo.UpdateMarch(march)
	return cause
}

func (s *Service) lockPlayers(playerA string, playerB string) func() {
	if playerA > playerB {
		playerA, playerB = playerB, playerA
	}
	first := s.getPlayerLock(playerA)
	first.Lock()
	if playerB == playerA {
		return func() { first.Unlock() }
	}
	second := s.getPlayerLock(playerB)
	second.Lock()
	return func() {
		second.Unlock()
		first.Unlock()
	}
}

func buildDispatchedCombatUnits(state *GameState, units map[string]int, now time.Time) ([]combat.Unit, error) {
	if len(units) == 0 {
		return nil, ErrNoUnitsSelected
	}
	modSources := CollectModifierSources(state)
	combatUnits := []combat.Unit{}
	for unitType, count := range units {
		if count <= 0 {
			continue
		}
		_, unitCfg, exists := FindUnitConfigByID(unitType)
		if !exists {
			return nil, ErrUnitNotFound
		}
		if isNonCombatUnit(unitCfg) {
			return nil, ErrNonCombatUnit
		}
		combatUnits = append(combatUnits, buildCombatUnit(unitType, unitType, unitCfg, count, now, modSources))
	}
	if len(combatUnits) == 0 {
		return nil, ErrNoUnitsSelected
	}
	return combatUnits, nil
}

func calculateMarchDurationSeconds(slowestSpeed int, config MarchConfig) int {
	config = normalizeMarchConfig(config)
	speed := math.Max(1, float64(slowestSpeed)*config.SpeedScale)
	raw := float64(config.MaxDurationSeconds) / speed
	duration := int(math.Ceil(raw))
	if duration < config.MinDurationSeconds {
		duration = config.MinDurationSeconds
	}
	if duration > config.MaxDurationSeconds {
		duration = config.MaxDurationSeconds
	}
	return duration
}

func slowestUnitSpeed(units map[string]int) int {
	slowest := 0
	for unitType, amount := range units {
		if amount <= 0 {
			continue
		}
		_, unitCfg, ok := FindUnitConfigByID(unitType)
		if !ok || isNonCombatUnit(unitCfg) {
			return 0
		}
		speed := unitCfg.Stats["speed"]
		if speed <= 0 {
			speed = 1
		}
		if slowest == 0 || speed < slowest {
			slowest = speed
		}
	}
	return slowest
}

func buildMarchView(march PvpMarch, playerID string, now time.Time) PvpMarchView {
	direction := "incoming"
	units := map[string]int(nil)
	sourcePlayerID := march.AttackerPlayerID
	sourceName := march.AttackerName
	sourceFaction := march.AttackerFaction
	targetPlayerID := march.DefenderPlayerID
	targetName := march.DefenderName
	targetFaction := march.DefenderFaction
	canAccelerate := false
	if march.AttackerPlayerID == playerID {
		direction = "outgoing"
		units = copyIntMap(march.Units)
		canAccelerate = march.Status == MarchStatusMarching
	}
	remaining := 0
	if arrivesAt, ok := parseMarchTime(march.ArrivesAt); ok && arrivesAt.After(now.UTC()) {
		remaining = int(math.Ceil(arrivesAt.Sub(now.UTC()).Seconds()))
	}
	return PvpMarchView{
		ID:               march.ID,
		Direction:        direction,
		Type:             march.Type,
		Status:           march.Status,
		SourcePlayerID:   sourcePlayerID,
		SourceName:       sourceName,
		SourceFaction:    sourceFaction,
		TargetPlayerID:   targetPlayerID,
		TargetName:       targetName,
		TargetFaction:    targetFaction,
		Units:            units,
		StartedAt:        march.StartedAt,
		ArrivesAt:        march.ArrivesAt,
		RemainingSeconds: remaining,
		AcceleratedTimes: march.AcceleratedTimes,
		CanAccelerate:    canAccelerate,
		AccelerateCost:   march.AccelerateCostCityGold,
		AttackerReportID: march.AttackerReportID,
		DefenderReportID: march.DefenderReportID,
	}
}

func sortMarches(marches []PvpMarch) {
	sort.SliceStable(marches, func(i, j int) bool {
		left, _ := parseMarchTime(marches[i].ArrivesAt)
		right, _ := parseMarchTime(marches[j].ArrivesAt)
		if !left.Equal(right) {
			return left.Before(right)
		}
		return marches[i].ID < marches[j].ID
	})
}

func sortMarchesByArrival(marches []PvpMarch) {
	sortMarches(marches)
}

func parseMarchTime(value string) (time.Time, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}, false
	}
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return time.Time{}, false
	}
	return parsed.UTC(), true
}

func copyIntMap(source map[string]int) map[string]int {
	next := make(map[string]int, len(source))
	for key, value := range source {
		next[key] = value
	}
	return next
}

func applyPvpCapturedUnits(attackerState *GameState, defenderState *GameState, sources []defenderUnitSource, captured map[string]int, toGarrison bool) map[string]int {
	if len(captured) == 0 {
		return nil
	}
	sourceByID := map[string]defenderUnitSource{}
	for _, source := range sources {
		sourceByID[source.CombatID] = source
	}
	applied := map[string]int{}
	for combatID, count := range captured {
		if count <= 0 {
			continue
		}
		source, ok := sourceByID[combatID]
		if !ok {
			unitType := strings.TrimSpace(strings.Split(combatID, "@")[0])
			if unitType == "" {
				continue
			}
			source = defenderUnitSource{CombatID: combatID, UnitType: unitType, Pool: defenderPoolArmy}
		}
		if source.Pool == defenderPoolGarrison {
			deductArmyUnit(&defenderState.GarrisonArmy, source.UnitType, count)
		} else {
			deductArmyUnit(&defenderState.Army, source.UnitType, count)
		}
		if toGarrison {
			addToArmy(&attackerState.GarrisonArmy, source.UnitType, count)
		} else {
			addToArmy(&attackerState.Army, source.UnitType, count)
		}
		applied[source.UnitType] += count
	}
	cleanArmyUnits(&attackerState.Army)
	cleanArmyUnits(&attackerState.GarrisonArmy)
	cleanArmyUnits(&defenderState.Army)
	cleanArmyUnits(&defenderState.GarrisonArmy)
	if len(applied) == 0 {
		return nil
	}
	return applied
}

func applyPvpAfterBattleTraits(report *BattleReport, attackerState *GameState, defenderState *GameState, attackerTraits []general.ActiveTrait, defenderTraits []general.ActiveTrait) {
	attackerArmy := armySliceToMap(attackerState.Army)
	attackerCtx := &general.AfterBattleContext{
		PlayerArmy:   attackerArmy,
		PlayerLosses: report.LostUnits,
		IsAttacker:   true,
		Won:          report.Result == "attacker_victory",
	}
	general.Dispatch(attackerCtx, attackerTraits)
	if len(attackerCtx.Revived) > 0 {
		attackerState.Army = armyMapToSlice(attackerArmy)
		report.RevivedUnits = mergeArmyMaps(report.RevivedUnits, attackerCtx.Revived)
	}
	mergeTraitOutcomes(report, attackerCtx.Triggered)

	defenderArmy := mergeArmyMaps(armySliceToMap(defenderState.Army), armySliceToMap(defenderState.GarrisonArmy))
	defenderCtx := &general.AfterBattleContext{
		PlayerArmy:   defenderArmy,
		PlayerLosses: report.DefenderLostUnits,
		IsAttacker:   false,
		Won:          report.Result == "defender_victory",
	}
	general.Dispatch(defenderCtx, defenderTraits)
	if len(defenderCtx.Revived) > 0 {
		applyDefenderRevivedByPools(defenderState, report, defenderCtx.Revived)
		report.RevivedUnits = mergeArmyMaps(report.RevivedUnits, defenderCtx.Revived)
	}
	mergeTraitOutcomes(report, defenderCtx.Triggered)
}

func applyDefenderRevivedByPools(defenderState *GameState, report *BattleReport, revived map[string]int) {
	for unitType, revivedCount := range revived {
		if revivedCount <= 0 {
			continue
		}
		mainLost := report.DefenderArmyLostUnits[unitType]
		garrisonLost := report.DefenderGarrisonLostUnits[unitType]
		totalLost := mainLost + garrisonLost
		if totalLost <= 0 {
			addToArmy(&defenderState.Army, unitType, revivedCount)
			continue
		}
		mainRevived := int(math.Floor(float64(revivedCount) * float64(mainLost) / float64(totalLost)))
		if mainLost > 0 && mainRevived == 0 {
			mainRevived = 1
		}
		if mainRevived > revivedCount {
			mainRevived = revivedCount
		}
		garrisonRevived := revivedCount - mainRevived
		if mainRevived > 0 {
			addToArmy(&defenderState.Army, unitType, mainRevived)
		}
		if garrisonRevived > 0 {
			addToArmy(&defenderState.GarrisonArmy, unitType, garrisonRevived)
		}
	}
	cleanArmyUnits(&defenderState.Army)
	cleanArmyUnits(&defenderState.GarrisonArmy)
}

func calculatePvpGeneralExpFromLosses(losses []combat.UnitLoss) int {
	total := 0
	for _, loss := range losses {
		if loss.Losses <= 0 {
			continue
		}
		unitType := strings.TrimSpace(strings.Split(loss.ID, "@")[0])
		if unitType == "" {
			continue
		}
		_, unitCfg, ok := FindUnitConfigByID(unitType)
		if !ok {
			continue
		}
		upkeep := unitCfg.Stats["upkeep"]
		if upkeep <= 0 {
			continue
		}
		total += loss.Losses * upkeep
	}
	return total
}
