// 本文件实现 NPC 战斗、侦查和战斗模拟服务。
package game

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"time"

	"hero3/internal/core/combat"
	"hero3/internal/core/general"
)

var (
	ErrNpcNotFound      = errors.New("npc city not found")
	ErrNoUnitsSelected  = errors.New("no units selected for dispatch")
	ErrInsufficientArmy = errors.New("insufficient army for dispatch")
	ErrNoSweepTargets   = errors.New("no npc sweep targets")
)

// AttackNpcRequest 攻击 NPC 请求
type AttackNpcRequest struct {
	PlayerID    string         `json:"playerId"`
	NpcID       string         `json:"npcId"`
	Mode        string         `json:"mode"`  // "attack" or "plunder"
	Units       map[string]int `json:"units"` // unitType → count
	GeneralIDs  []string       `json:"generalIds,omitempty"`
	EffectRefID string         `json:"-"`
}

// AttackNpcResponse 攻击 NPC 响应
type AttackNpcResponse struct {
	BattleReport BattleReport  `json:"battleReport"`
	Resources    ResourceState `json:"resources"`
	Army         []ArmyUnit    `json:"army"`
	General      *General      `json:"general,omitempty"`
	Generals     []General     `json:"generals,omitempty"`
	CityGold     FlexInt       `json:"cityGold"`
	NpcState     *NpcState     `json:"npcState,omitempty"`
	ServerTime   string        `json:"serverTime"`
}

// SweepNpcRequest 批量扫荡 NPC 请求。
type SweepNpcRequest struct {
	PlayerID   string   `json:"playerId"`
	NpcIDs     []string `json:"npcIds"`
	Mode       string   `json:"mode"`
	GeneralIDs []string `json:"generalIds,omitempty"`
}

// SweepNpcResponse 批量扫荡 NPC 响应。
type SweepNpcResponse struct {
	BattleReport BattleReport  `json:"battleReport"`
	Done         int           `json:"done"`
	Failed       int           `json:"failed"`
	Stopped      bool          `json:"stopped"`
	Resources    ResourceState `json:"resources"`
	Army         []ArmyUnit    `json:"army"`
	General      *General      `json:"general,omitempty"`
	Generals     []General     `json:"generals,omitempty"`
	CityGold     FlexInt       `json:"cityGold"`
	NpcState     *NpcState     `json:"npcState,omitempty"`
	ServerTime   string        `json:"serverTime"`
}

// BattleSimulationRequest 战斗模拟请求：只计算战果，不扣兵、不保存战报。
type BattleSimulationRequest struct {
	PlayerID             string         `json:"playerId"`
	Mode                 string         `json:"mode"` // "attack" or "plunder"
	AttackerFaction      string         `json:"attackerFaction"`
	DefenderFaction      string         `json:"defenderFaction"`
	AttackerUnits        map[string]int `json:"attackerUnits"`
	DefenderUnits        map[string]int `json:"defenderUnits"`
	ApplyAttackerBonuses bool           `json:"applyAttackerBonuses"`
	ApplyDefenderBonuses bool           `json:"applyDefenderBonuses"`
}

// BattleSimulationResponse 战斗模拟结果。
type BattleSimulationResponse struct {
	Result   combat.CombatResult `json:"result"`
	Attacker combat.Army         `json:"attacker"`
	Defender combat.Army         `json:"defender"`
}

// ScoutNpcRequest 侦查 NPC 请求
type ScoutNpcRequest struct {
	PlayerID string `json:"playerId"`
	NpcID    string `json:"npcId"`
}

// ScoutNpcResponse 侦查 NPC 响应
type ScoutNpcResponse struct {
	Success      bool         `json:"success"`
	BattleReport BattleReport `json:"battleReport"`
	NpcCity      *NpcCity     `json:"npcCity"`
	Army         []ArmyUnit   `json:"army"`
	NpcState     *NpcState    `json:"npcState,omitempty"`
	ServerTime   string       `json:"serverTime"`
}

// AttackNpc 攻击 NPC 城池
func (s *Service) AttackNpc(req AttackNpcRequest) (AttackNpcResponse, error) {
	playerID := strings.TrimSpace(req.PlayerID)
	if playerID == "" {
		return AttackNpcResponse{}, ErrPlayerNotFound
	}
	unlock, ok := s.tryPlayerLockIfIdle(playerID)
	if !ok {
		return AttackNpcResponse{}, ErrOperationTooFast
	}
	defer unlock()

	return s.attackNpc(req, true)
}

// combatScopeForUnits 从出兵请求中提取本次战斗需要锁定的兵种范围。
func combatScopeForAttack(units map[string]int, generalIDs []string) CombatAssetScope {
	scope := CombatAssetScope{}
	seenUnits := map[string]struct{}{}
	for unitType, count := range units {
		unitType = strings.TrimSpace(unitType)
		if unitType == "" || count <= 0 {
			continue
		}
		if _, ok := seenUnits[unitType]; ok {
			continue
		}
		seenUnits[unitType] = struct{}{}
		scope.UnitTypes = append(scope.UnitTypes, unitType)
	}
	seenGenerals := map[string]struct{}{}
	for _, generalID := range generalIDs {
		generalID = strings.TrimSpace(generalID)
		if generalID == "" {
			continue
		}
		if _, ok := seenGenerals[generalID]; ok {
			continue
		}
		seenGenerals[generalID] = struct{}{}
		scope.GeneralIDs = append(scope.GeneralIDs, generalID)
	}
	scope.InventoryItemIDs = npcBattleDropCandidateItemIDs()
	if len(scope.InventoryItemIDs) == 0 {
		scope.SkipInventory = true
	}
	return scope
}

// combatScopeForScoutFaction 从玩家阵营推导侦查事务需要锁定的侦察兵种。
func combatScopeForScoutFaction(faction string) (CombatAssetScope, error) {
	scoutUnitID := findScoutUnit(faction)
	if scoutUnitID == "" {
		return CombatAssetScope{}, ErrNoUnitsSelected
	}
	return CombatAssetScope{UnitTypes: []string{scoutUnitID}, SkipInventory: true}, nil
}

// npcBattleDropCandidateItemIDs 汇总当前 NPC 配置可能产出的道具，用于把战斗背包锁收窄到候选掉落物。
func npcBattleDropCandidateItemIDs() []string {
	cfg := GetNpcConfig()
	seenPools := map[string]struct{}{}
	seenItems := map[string]struct{}{}
	for _, tier := range cfg.Tiers {
		collectDropPoolItemIDs(tier.DropPoolID, seenPools, seenItems)
	}
	itemIDs := make([]string, 0, len(seenItems))
	for itemID := range seenItems {
		itemIDs = append(itemIDs, itemID)
	}
	sort.Strings(itemIDs)
	return itemIDs
}

// collectDropPoolItemIDs 递归收集掉落池中的道具 ID，并防止配置递归导致无限循环。
func collectDropPoolItemIDs(poolID string, seenPools map[string]struct{}, seenItems map[string]struct{}) {
	poolID = strings.TrimSpace(poolID)
	if poolID == "" {
		return
	}
	if _, ok := seenPools[poolID]; ok {
		return
	}
	seenPools[poolID] = struct{}{}
	pool, ok := GetDropPoolDefinition(poolID)
	if !ok {
		return
	}
	for _, reward := range dropPoolCycleItems(pool) {
		switch strings.TrimSpace(reward.Type) {
		case RewardTypeItem:
			itemID := strings.TrimSpace(reward.ID)
			if itemID != "" {
				seenItems[itemID] = struct{}{}
			}
		case "drop_pool":
			collectDropPoolItemIDs(reward.DropPoolID, seenPools, seenItems)
		}
	}
}

// attackNpc 结算一次 NPC 战斗；saveReport 控制是否保存单场战报和发布战斗事件。
func (s *Service) attackNpc(req AttackNpcRequest, saveReport bool) (AttackNpcResponse, error) {
	playerID := strings.TrimSpace(req.PlayerID)
	npcID := strings.TrimSpace(req.NpcID)
	mode := strings.TrimSpace(req.Mode)

	if playerID == "" {
		return AttackNpcResponse{}, ErrPlayerNotFound
	}
	if npcID == "" {
		return AttackNpcResponse{}, ErrNpcNotFound
	}
	if mode == "" {
		mode = "attack"
	}
	if mode != "attack" && mode != "plunder" {
		mode = "attack"
	}

	now := time.Now()
	var report BattleReport
	var rewardApply RewardApplyResult
	capturedToGarrison := map[string]int{}
	capturedSourceFaction := ""
	var state GameState
	var err error
	for attempt := 0; attempt < 3; attempt++ {
		report = BattleReport{}
		rewardApply = RewardApplyResult{}
		capturedToGarrison = map[string]int{}
		capturedSourceFaction = ""
		state, err = s.repo.UpdateCombatState(playerID, combatScopeForAttack(req.Units, req.GeneralIDs), now, func(state *GameState) error {
			prepareNpcCombatState(state, now)
			outcome, err := resolveNpcBattleOnState(state, req, now)
			if err != nil {
				return err
			}
			report = outcome.Report
			rewardApply = outcome.RewardApply
			capturedToGarrison = outcome.CapturedToGarrison
			capturedSourceFaction = outcome.CapturedSourceFaction
			state.ServerTime = now.UTC().Format(resourceDateLayout)
			return nil
		})
		if err == nil {
			break
		}
		if !isRetryableStorageConflict(err) || attempt == 2 {
			return AttackNpcResponse{}, err
		}
		slog.Warn("npc attack transaction retry after storage conflict", "playerId", playerID, "npcId", npcID, "attempt", attempt+1, "error", err)
		time.Sleep(time.Duration(attempt+1) * 80 * time.Millisecond)
	}
	if err != nil {
		return AttackNpcResponse{}, err
	}
	if saveReport {
		s.flushRewardSideEffects(rewardApply)
	} else {
		s.flushRewardSideEffectsWithoutEvents(rewardApply)
	}

	if len(capturedToGarrison) > 0 {
		effectRefID := strings.TrimSpace(req.EffectRefID)
		if effectRefID == "" {
			effectRefID = report.ID
		}
		result, err := s.CreateGarrisonDetachment(CreateGarrisonDetachmentRequest{
			OwnerPlayerID: state.Player.ID,
			HostPlayerID:  state.Player.ID,
			SourceType:    GarrisonSourceCaptured,
			SourceID:      effectRefID,
			SourceFaction: capturedSourceFaction,
			Troops:        capturedToGarrison,
			Metadata: map[string]any{
				"reason":   "beauty_trap_capture",
				"reportId": effectRefID,
				"npcId":    npcID,
			},
		})
		if err != nil {
			return AttackNpcResponse{}, err
		}
		if result.Patch.ServerTime != "" {
			state.ServerTime = result.Patch.ServerTime
		}
	}

	if saveReport {
		if report.OverflowCityGold > 0 {
			s.recordLedger(GoldLedgerEntry{
				PlayerID:     state.Player.ID,
				Currency:     LedgerCurrencyCityGold,
				Direction:    LedgerDirectionCredit,
				Amount:       report.OverflowCityGold,
				BalanceAfter: int(state.CityGold),
				RefType:      LedgerRefBattleOverflow,
				RefID:        report.ID,
			})
		}

		createResult, err := s.CreateBattleReports(BattleReportCreateInput{
			EventID:    report.EventID,
			SourceType: report.SourceType,
			SourceID:   report.TargetID,
			BattleType: report.BattleType,
			Result:     report.Result,
			OccurredAt: report.CreatedAt,
			Reports:    []BattleReport{report},
		})
		if err != nil {
			return AttackNpcResponse{}, err
		} else if len(createResult.Reports) > 0 {
			report = createResult.Reports[0]
		}
		s.publishBattleRewardEvents(state.Player.ID, report)
		s.publishBattleFinished(state.Player.ID, report)

		s.attachReportSummary(&state, state.Player.ID)
	}

	return AttackNpcResponse{
		BattleReport: report,
		Resources:    state.Resources,
		Army:         state.Army,
		General:      state.General,
		Generals:     state.Generals,
		CityGold:     state.CityGold,
		NpcState:     state.NpcState,
		ServerTime:   state.ServerTime,
	}, nil
}

type npcBattleStateOutcome struct {
	Report                BattleReport
	RewardApply           RewardApplyResult
	CapturedToGarrison    map[string]int
	CapturedSourceFaction string
}

// prepareNpcCombatState 在战斗事务内做一次玩家和 NPC 的惰性结算。
func prepareNpcCombatState(state *GameState, now time.Time) {
	if state.General != nil {
		applyHeroConfigToGeneral(state.General)
	}
	EnsureGeneralRoster(state, now)
	nextState, _ := settleResources(*state, now)
	*state = nextState
	if state.NpcState != nil {
		settleNpcCities(state.NpcState, now)
	}
}

// resolveNpcBattleOnState 在已锁定并加载的玩家状态内结算单场 NPC 战斗。
func resolveNpcBattleOnState(state *GameState, req AttackNpcRequest, now time.Time) (npcBattleStateOutcome, error) {
	npcID := strings.TrimSpace(req.NpcID)
	mode := strings.TrimSpace(req.Mode)
	if mode != "attack" && mode != "plunder" {
		mode = "attack"
	}
	if state.NpcState == nil || len(state.NpcState.Cities) == 0 {
		return npcBattleStateOutcome{}, ErrNpcNotFound
	}
	npcIdx := -1
	for i, city := range state.NpcState.Cities {
		if city.ID == npcID {
			npcIdx = i
			break
		}
	}
	if npcIdx == -1 {
		return npcBattleStateOutcome{}, ErrNpcNotFound
	}

	npc := &state.NpcState.Cities[npcIdx]
	generalIDs, err := normalizeBattleGeneralIDs(state, req.GeneralIDs)
	if err != nil {
		return npcBattleStateOutcome{}, err
	}
	attackerUnits, err := validateAndConsumeArmyWithModifiers(state, req.Units, modifierSourcesForBattleGenerals(state, generalIDs)...)
	if err != nil {
		return npcBattleStateOutcome{}, err
	}

	ruleID := activeCombatRuleID(combatSceneForPVE(mode))
	attackerArmy := buildCombatArmy(state.Player.Faction, attackerUnits)
	defenderArmy := buildNpcCombatArmy(npc)
	activeTraits := buildActiveTraitsForGeneralIDs(state, generalIDs)
	beforeCtx := &general.BeforeBattleContext{
		Attacker:          &attackerArmy,
		Defender:          &defenderArmy,
		AttackerOwnsTrait: true,
		DefenderOwnsTrait: false,
		IsPvP:             false,
		SameFaction:       true,
	}
	general.Dispatch(beforeCtx, activeTraits)
	capturedToArmy, routedToGarrison := splitCapturedUnitsByOwnerFaction(state.Player.Faction, beforeCtx.CapturedToArmy)
	capturedToGarrison := mergeTroopMaps(routedToGarrison, beforeCtx.CapturedToGarrison)
	for unitType, count := range capturedToArmy {
		mergeIntoArmy(state, unitType, count)
	}

	result := combat.Resolve(combat.CombatInput{
		RuleID:    ruleID,
		Attacker:  attackerArmy,
		Defender:  defenderArmy,
		WallLevel: 0,
	})

	afterCombatCtx := &general.AfterCombatResolveContext{
		Result:            &result,
		Attacker:          &attackerArmy,
		Defender:          &defenderArmy,
		AttackerOwnsTrait: true,
		DefenderOwnsTrait: false,
		IsAttackerOnly:    true,
	}
	general.Dispatch(afterCombatCtx, activeTraits)

	report := applyNpcBattleResult(state, npc, result, attackerUnits, mode, now)
	if len(capturedToArmy) > 0 {
		report.CapturedUnits = capturedToArmy
	}
	if len(capturedToGarrison) > 0 {
		report.CapturedToGarrison = cloneStringIntMap(capturedToGarrison)
	}
	mergeTraitOutcomes(&report, beforeCtx.Triggered)
	mergeTraitOutcomes(&report, afterCombatCtx.Triggered)

	playerArmyMap := armySliceToMap(state.Army)
	afterBattleCtx := &general.AfterBattleContext{
		PlayerArmy:   playerArmyMap,
		PlayerLosses: report.LostUnits,
		IsAttacker:   true,
		Won:          report.Result == "attacker_victory",
	}
	general.Dispatch(afterBattleCtx, activeTraits)
	if len(afterBattleCtx.Revived) > 0 {
		state.Army = armyMapToSlice(playerArmyMap)
		if report.RevivedUnits == nil {
			report.RevivedUnits = map[string]int{}
		}
		for k, v := range afterBattleCtx.Revived {
			report.RevivedUnits[k] = v
		}
	}
	mergeTraitOutcomes(&report, afterBattleCtx.Triggered)

	expResult := applyGeneralBattleExpToRoster(state, generalIDs, calculateGeneralBattleExpFromLosses(npc.Faction, result.DefenderLosses))
	if expResult.Gained > 0 {
		report.GeneralExpGained = expResult.Gained
		report.GeneralLevelBefore = expResult.LevelBefore
		report.GeneralLevelAfter = expResult.LevelAfter
	}
	effectRefID := strings.TrimSpace(req.EffectRefID)
	if effectRefID == "" {
		effectRefID = report.ID
	}
	var rewardApply RewardApplyResult
	dropRewards, dropSnapshots, err := rollNpcBattleDrops(npc, report)
	if err != nil {
		return npcBattleStateOutcome{}, err
	}
	if len(dropRewards) > 0 {
		apply, err := ApplyRewardsToStateWithContext(state, dropRewards, RewardGrantContext{
			PlayerID: state.Player.ID,
			RefType:  LedgerRefBattleReward,
			RefID:    effectRefID,
			Reason:   "npc_battle_drop",
		}, now)
		if err != nil {
			return npcBattleStateOutcome{}, err
		}
		mergeRewardApplyResult(&rewardApply, apply)
		report.Drops = dropSnapshots
	}
	report.PvpAttackerGenerals = buildPvpGeneralSnapshots(state, generalIDs)
	report.GrantedRewards = buildBattleGrantedRewards(report)

	if report.OverflowCityGold > 0 {
		state.CityGold += FlexInt(report.OverflowCityGold)
	}
	return npcBattleStateOutcome{
		Report:                report,
		RewardApply:           rewardApply,
		CapturedToGarrison:    capturedToGarrison,
		CapturedSourceFaction: npc.Faction,
	}, nil
}

const maxNpcSweepTargets = 20
const slowNpcSweepThreshold = 2 * time.Second
const slowNpcSweepTargetThreshold = 2 * time.Second
const battleReportSaveAttempts = 3
const battleReportSaveRetryDelay = 120 * time.Millisecond

// SweepNpc 批量扫荡 NPC 城池，并把多场扫荡合并为一条战报。
func (s *Service) SweepNpc(req SweepNpcRequest) (SweepNpcResponse, error) {
	return s.sweepNpc(req, maxNpcSweepTargets)
}

// sweepNpc 执行实际扫荡结算；limit 为 0 时不截断目标，供后台任务处理完整扫荡。
func (s *Service) sweepNpc(req SweepNpcRequest, limit int) (SweepNpcResponse, error) {
	sweepStartedAt := time.Now()
	playerID := strings.TrimSpace(req.PlayerID)
	if playerID == "" {
		return SweepNpcResponse{}, ErrPlayerNotFound
	}
	unlock, ok := s.tryPlayerLockIfIdle(playerID)
	if !ok {
		return SweepNpcResponse{}, ErrOperationTooFast
	}
	defer unlock()

	npcIDs := normalizeSweepNpcIDs(req.NpcIDs)
	requestedInput := len(npcIDs)
	if len(npcIDs) == 0 {
		return SweepNpcResponse{}, ErrNoSweepTargets
	}
	truncated := false
	if limit > 0 && len(npcIDs) > limit {
		npcIDs = npcIDs[:limit]
		truncated = true
	}

	mode := strings.TrimSpace(req.Mode)
	if mode != "attack" && mode != "plunder" {
		mode = "attack"
	}

	stateLoadStartedAt := time.Now()
	state, err := s.repo.GetState(playerID)
	stateLoadDuration := time.Since(stateLoadStartedAt)
	if err != nil {
		return SweepNpcResponse{}, err
	}
	if len(sweepUnitsFromArmy(state.Army)) == 0 {
		return SweepNpcResponse{}, ErrNoUnitsSelected
	}
	sweepReportID := "br_" + randomID(8)
	initialResponse := SweepNpcResponse{
		Resources:  state.Resources,
		Army:       state.Army,
		General:    state.General,
		Generals:   state.Generals,
		CityGold:   state.CityGold,
		NpcState:   state.NpcState,
		ServerTime: state.ServerTime,
	}
	response := initialResponse
	var reports []BattleReport
	var rewardApply RewardApplyResult
	var capturedGarrisons []struct {
		NpcID         string
		SourceFaction string
		Troops        map[string]int
	}
	var combatDuration time.Duration
	scope := combatScopeForAttack(sweepUnitsFromArmy(state.Army), req.GeneralIDs)
	for attempt := 0; attempt < 3; attempt++ {
		attemptResponse := initialResponse
		attemptReports := make([]BattleReport, 0, len(npcIDs))
		attemptRewardApply := RewardApplyResult{}
		attemptCapturedGarrisons := []struct {
			NpcID         string
			SourceFaction string
			Troops        map[string]int
		}{}
		attemptCombatDuration := time.Duration(0)
		state, err = s.repo.UpdateCombatState(playerID, scope, time.Now(), func(state *GameState) error {
			prepareNpcCombatState(state, time.Now())
			if state.NpcState == nil || len(state.NpcState.Cities) == 0 {
				return ErrNpcNotFound
			}
			for _, npcID := range npcIDs {
				units := sweepUnitsFromArmy(state.Army)
				if len(units) == 0 {
					attemptResponse.Stopped = true
					break
				}

				targetStartedAt := time.Now()
				outcome, err := resolveNpcBattleOnState(state, AttackNpcRequest{
					PlayerID:    playerID,
					NpcID:       npcID,
					Mode:        mode,
					Units:       units,
					GeneralIDs:  req.GeneralIDs,
					EffectRefID: sweepReportID,
				}, time.Now())
				targetDuration := time.Since(targetStartedAt)
				attemptCombatDuration += targetDuration
				if targetDuration >= slowNpcSweepTargetThreshold {
					slog.Warn("npc sweep target slow", "playerId", playerID, "npcId", npcID, "duration_ms", targetDuration.Milliseconds())
				}
				if err != nil {
					if errors.Is(err, ErrGeneralNotFound) || errors.Is(err, ErrGeneralBusy) {
						return err
					}
					if errors.Is(err, ErrNoUnitsSelected) || errors.Is(err, ErrInsufficientArmy) {
						attemptResponse.Stopped = true
						break
					}
					if errors.Is(err, ErrNpcNotFound) {
						attemptResponse.Failed++
						continue
					}
					return err
				}

				attemptResponse.Done++
				attemptReports = append(attemptReports, outcome.Report)
				mergeRewardApplyResult(&attemptRewardApply, outcome.RewardApply)
				if len(outcome.CapturedToGarrison) > 0 {
					attemptCapturedGarrisons = append(attemptCapturedGarrisons, struct {
						NpcID         string
						SourceFaction string
						Troops        map[string]int
					}{
						NpcID:         npcID,
						SourceFaction: outcome.CapturedSourceFaction,
						Troops:        cloneStringIntMap(outcome.CapturedToGarrison),
					})
				}
			}
			attemptResponse.Resources = state.Resources
			attemptResponse.Army = state.Army
			attemptResponse.General = state.General
			attemptResponse.Generals = state.Generals
			attemptResponse.CityGold = state.CityGold
			attemptResponse.NpcState = state.NpcState
			attemptResponse.ServerTime = time.Now().UTC().Format(resourceDateLayout)
			state.ServerTime = attemptResponse.ServerTime
			return nil
		})
		if err == nil {
			response = attemptResponse
			reports = attemptReports
			rewardApply = attemptRewardApply
			capturedGarrisons = attemptCapturedGarrisons
			combatDuration = attemptCombatDuration
			break
		}
		if !isRetryableStorageConflict(err) || attempt == 2 {
			return SweepNpcResponse{}, err
		}
		slog.Warn("npc sweep transaction retry after storage conflict", "playerId", playerID, "attempt", attempt+1, "error", err)
		time.Sleep(time.Duration(attempt+1) * 80 * time.Millisecond)
	}
	s.flushRewardSideEffectsWithoutEvents(rewardApply)
	for _, captured := range capturedGarrisons {
		result, err := s.CreateGarrisonDetachment(CreateGarrisonDetachmentRequest{
			OwnerPlayerID: playerID,
			HostPlayerID:  playerID,
			SourceType:    GarrisonSourceCaptured,
			SourceID:      sweepReportID,
			SourceFaction: captured.SourceFaction,
			Troops:        captured.Troops,
			Metadata: map[string]any{
				"reason":   "beauty_trap_capture",
				"reportId": sweepReportID,
				"npcId":    captured.NpcID,
			},
		})
		if err != nil {
			return SweepNpcResponse{}, err
		}
		if result.Patch.ServerTime != "" {
			response.ServerTime = result.Patch.ServerTime
		}
	}

	if len(reports) == 0 {
		logNpcSweepCompleted(playerID, requestedInput, len(npcIDs), response.Done, response.Failed, response.Stopped, truncated, time.Since(sweepStartedAt), stateLoadDuration, combatDuration, 0)
		return response, nil
	}

	reportStartedAt := time.Now()
	report := buildNpcSweepReport(sweepReportID, reports, mode, len(npcIDs), response.Failed, response.Stopped)
	createResult, err := s.CreateBattleReports(BattleReportCreateInput{
		EventID:    report.EventID,
		SourceType: report.SourceType,
		SourceID:   report.TargetID,
		BattleType: report.BattleType,
		Result:     report.Result,
		OccurredAt: report.CreatedAt,
		Reports:    []BattleReport{report},
		Extra: map[string]interface{}{
			"requested": len(npcIDs),
			"success":   response.Done,
			"failed":    response.Failed,
			"stopped":   response.Stopped,
		},
	})
	if err != nil {
		return SweepNpcResponse{}, err
	} else if len(createResult.Reports) > 0 {
		for _, savedReport := range createResult.Reports {
			if savedReport.ID == report.ID {
				report = savedReport
				break
			}
		}
	}

	if report.OverflowCityGold > 0 {
		s.recordLedger(GoldLedgerEntry{
			PlayerID:     playerID,
			Currency:     LedgerCurrencyCityGold,
			Direction:    LedgerDirectionCredit,
			Amount:       report.OverflowCityGold,
			BalanceAfter: int(response.CityGold),
			RefType:      LedgerRefBattleOverflow,
			RefID:        report.ID,
		})
	}
	s.publishBattleRewardEvents(playerID, report)
	s.publishBattleFinished(playerID, report)
	response.BattleReport = report
	reportDuration := time.Since(reportStartedAt)
	logNpcSweepCompleted(playerID, requestedInput, len(npcIDs), response.Done, response.Failed, response.Stopped, truncated, time.Since(sweepStartedAt), stateLoadDuration, combatDuration, reportDuration)
	return response, nil
}

// logNpcSweepCompleted 记录扫荡链路的关键耗时，便于线上判断是否需要任务化或批量事务改造。
func logNpcSweepCompleted(playerID string, requestedInput int, requested int, done int, failed int, stopped bool, truncated bool, totalDuration time.Duration, stateLoadDuration time.Duration, combatDuration time.Duration, reportDuration time.Duration) {
	attrs := []any{
		"playerId", playerID,
		"requested_input", requestedInput,
		"requested", requested,
		"done", done,
		"failed", failed,
		"stopped", stopped,
		"truncated", truncated,
		"duration_ms", totalDuration.Milliseconds(),
		"state_load_ms", stateLoadDuration.Milliseconds(),
		"combat_ms", combatDuration.Milliseconds(),
		"report_ms", reportDuration.Milliseconds(),
	}
	if totalDuration >= slowNpcSweepThreshold {
		slog.Warn("npc sweep completed slowly", attrs...)
		return
	}
	slog.Info("npc sweep completed", attrs...)
}

// normalizeSweepNpcIDs 去重并过滤空 NPC ID，避免一次扫荡重复打同一个目标。
func normalizeSweepNpcIDs(values []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		id := strings.TrimSpace(value)
		if id == "" || seen[id] {
			continue
		}
		seen[id] = true
		out = append(out, id)
	}
	return out
}

// sweepUnitsFromArmy 使用当前剩余全部兵力作为下一场扫荡出征兵力。
func sweepUnitsFromArmy(army []ArmyUnit) map[string]int {
	units := map[string]int{}
	for _, unit := range army {
		if unit.UnitType != "" && unit.Amount > 0 {
			units[unit.UnitType] = unit.Amount
		}
	}
	return units
}

// buildNpcSweepReport 把多场 NPC 战斗结果合并为一条扫荡战报。
func buildNpcSweepReport(reportID string, reports []BattleReport, mode string, requested int, failed int, stopped bool) BattleReport {
	first := reports[0]
	last := reports[len(reports)-1]
	aggregate := BattleReport{
		ID:               reportID,
		PlayerID:         first.PlayerID,
		OwnerPlayerID:    first.PlayerID,
		ViewType:         ReportViewAttack,
		SourceType:       ReportSourceNPCCity,
		BattleType:       "sweep",
		Title:            "NPC 扫荡",
		PlayerFaction:    first.PlayerFaction,
		PlayerName:       first.PlayerName,
		TargetID:         "npc_sweep",
		TargetName:       "NPC 扫荡",
		Type:             mode,
		Result:           "attacker_victory",
		PlayerPower:      first.PlayerPower,
		DispatchedUnits:  cloneStringIntMap(first.DispatchedUnits),
		DefenderFaction:  first.DefenderFaction,
		DefenderRevealed: true,
		Read:             false,
		CreatedAt:        last.CreatedAt,
	}

	levelBeforeSet := false
	for _, report := range reports {
		aggregate.EnemyPower += report.EnemyPower
		aggregate.LostUnits = mergeTroopMaps(aggregate.LostUnits, report.LostUnits)
		aggregate.DefenderUnits = mergeTroopMaps(aggregate.DefenderUnits, report.DefenderUnits)
		aggregate.DefenderLostUnits = mergeTroopMaps(aggregate.DefenderLostUnits, report.DefenderLostUnits)
		aggregate.DefenderResources = mergeTroopMaps(aggregate.DefenderResources, report.DefenderResources)
		aggregate.Rewards = mergeTroopMaps(aggregate.Rewards, report.Rewards)
		aggregate.Overflow = mergeTroopMaps(aggregate.Overflow, report.Overflow)
		aggregate.CapturedUnits = mergeTroopMaps(aggregate.CapturedUnits, report.CapturedUnits)
		aggregate.CapturedToGarrison = mergeTroopMaps(aggregate.CapturedToGarrison, report.CapturedToGarrison)
		aggregate.RevivedUnits = mergeTroopMaps(aggregate.RevivedUnits, report.RevivedUnits)
		aggregate.OverflowCityGold += report.OverflowCityGold
		aggregate.GeneralExpGained += report.GeneralExpGained
		if report.GeneralExpGained > 0 {
			if !levelBeforeSet {
				aggregate.GeneralLevelBefore = report.GeneralLevelBefore
				levelBeforeSet = true
			}
			aggregate.GeneralLevelAfter = report.GeneralLevelAfter
		}
		aggregate.Drops = append(aggregate.Drops, report.Drops...)
		aggregate.PvpAttackerGenerals = latestPvpGeneralSnapshots(aggregate.PvpAttackerGenerals, report.PvpAttackerGenerals)
		mergeReportTraitOutcomes(&aggregate, report)
	}
	aggregate.GrantedRewards = buildBattleGrantedRewards(aggregate)
	aggregate.Summary = fmt.Sprintf("扫荡 %d 城，成功 %d 场，失败 %d 场，获得 %d 城金，武将经验 +%d。", requested, len(reports), failed, aggregate.OverflowCityGold, aggregate.GeneralExpGained)
	detail := BuildBattleReportDetail(aggregate)
	if detail.Extra == nil {
		detail.Extra = map[string]interface{}{}
	}
	detail.Extra["sweep"] = map[string]interface{}{
		"requested": requested,
		"success":   len(reports),
		"failed":    failed,
		"stopped":   stopped,
		"mode":      mode,
		"defenders": buildNpcSweepDefenders(reports),
	}
	aggregate.Detail = &detail
	return aggregate
}

// buildNpcSweepDefenders 构造扫荡聚合战报里的轻量 NPC 防守方明细，完整战损使用聚合字段保存。
func buildNpcSweepDefenders(reports []BattleReport) []BattleReportSweepDefender {
	defenders := make([]BattleReportSweepDefender, 0, len(reports))
	for _, report := range reports {
		defenders = append(defenders, BattleReportSweepDefender{
			TargetID:         report.TargetID,
			TargetName:       report.TargetName,
			Faction:          report.DefenderFaction,
			FactionLabel:     factionLabel(report.DefenderFaction),
			Power:            report.EnemyPower,
			Result:           report.Result,
			DefenderRevealed: report.DefenderRevealed,
		})
	}
	return defenders
}

// latestPvpGeneralSnapshots 保留最新的武将快照，确保聚合战报展示扫荡结束后的等级。
func latestPvpGeneralSnapshots(current []PvpGeneralSnapshot, next []PvpGeneralSnapshot) []PvpGeneralSnapshot {
	if len(next) == 0 {
		return current
	}
	return append([]PvpGeneralSnapshot(nil), next...)
}

// mergeReportTraitOutcomes 合并单场战报中的特性触发信息。
func mergeReportTraitOutcomes(target *BattleReport, source BattleReport) {
	if target == nil {
		return
	}
	if len(source.TraitOutcomes) > 0 && target.TraitOutcomes == nil {
		target.TraitOutcomes = map[string]TraitOutcomeReport{}
	}
	for _, traitID := range source.TraitTriggered {
		alreadyIn := false
		for _, existing := range target.TraitTriggered {
			if existing == traitID {
				alreadyIn = true
				break
			}
		}
		if !alreadyIn {
			target.TraitTriggered = append(target.TraitTriggered, traitID)
		}
		if outcome, ok := source.TraitOutcomes[traitID]; ok {
			target.TraitOutcomes[traitID] = outcome
		}
	}
}

// isRetryableStorageConflict 判断数据库事务冲突是否适合立即重试。
func isRetryableStorageConflict(err error) bool {
	if err == nil {
		return false
	}
	text := strings.ToLower(err.Error())
	return strings.Contains(text, "deadlock") ||
		strings.Contains(text, "error 1213") ||
		strings.Contains(text, "40001") ||
		strings.Contains(text, "lock wait timeout") ||
		strings.Contains(text, "error 1205")
}

// SimulateBattle 使用和 NPC 进攻一致的战斗规则计算结果，但不改变任何玩家状态。
func (s *Service) SimulateBattle(req BattleSimulationRequest) (BattleSimulationResponse, error) {
	playerID := strings.TrimSpace(req.PlayerID)
	if playerID == "" {
		return BattleSimulationResponse{}, ErrPlayerNotFound
	}

	state, err := s.repo.GetState(playerID)
	if err != nil {
		return BattleSimulationResponse{}, err
	}
	if state.General != nil {
		applyHeroConfigToGeneral(state.General)
	}

	mode := strings.TrimSpace(req.Mode)
	if mode != "attack" && mode != "plunder" {
		mode = "attack"
	}
	attackerFaction := strings.TrimSpace(req.AttackerFaction)
	if attackerFaction == "" {
		attackerFaction = state.Player.Faction
	}
	defenderFaction := strings.TrimSpace(req.DefenderFaction)
	if defenderFaction == "" {
		defenderFaction = attackerFaction
	}

	now := time.Now()
	modSources := CollectModifierSources(&state)
	attackerSources := []ModifierSource(nil)
	if req.ApplyAttackerBonuses {
		attackerSources = modSources
	}
	defenderSources := []ModifierSource(nil)
	if req.ApplyDefenderBonuses {
		defenderSources = modSources
	}

	attackerUnits, err := buildSimulatedCombatUnits(attackerFaction, req.AttackerUnits, now, attackerSources...)
	if err != nil {
		return BattleSimulationResponse{}, err
	}
	defenderUnits, err := buildSimulatedCombatUnits(defenderFaction, req.DefenderUnits, now, defenderSources...)
	if err != nil {
		return BattleSimulationResponse{}, err
	}

	ruleID := activeCombatRuleID(combatSceneForPVE(mode))

	attackerArmy := buildCombatArmy(attackerFaction, attackerUnits)
	defenderArmy := buildCombatArmy(defenderFaction, defenderUnits)
	result := combat.Resolve(combat.CombatInput{
		RuleID:    ruleID,
		Attacker:  attackerArmy,
		Defender:  defenderArmy,
		WallLevel: 0,
	})

	return BattleSimulationResponse{
		Result:   result,
		Attacker: attackerArmy,
		Defender: defenderArmy,
	}, nil
}

// ScoutNpc 侦查 NPC 城池
// 规则：玩家侦察兵 vs NPC 侦察兵，比数量。多的赢，少的全灭，多的损失=对方数量。
// 玩家存活 ≥ 1 → 侦查成功；全灭 → 失败。
func (s *Service) ScoutNpc(req ScoutNpcRequest) (ScoutNpcResponse, error) {
	playerID := strings.TrimSpace(req.PlayerID)
	npcID := strings.TrimSpace(req.NpcID)

	if playerID == "" {
		return ScoutNpcResponse{}, ErrPlayerNotFound
	}
	if npcID == "" {
		return ScoutNpcResponse{}, ErrNpcNotFound
	}

	now := time.Now()
	var report BattleReport
	var scoutSuccess bool
	var revealedNpc *NpcCity
	summary, err := s.repo.GetPlayerSummaryView(playerID)
	if err != nil {
		return ScoutNpcResponse{}, err
	}
	scoutScope, err := combatScopeForScoutFaction(summary.Player.Faction)
	if err != nil {
		return ScoutNpcResponse{}, err
	}
	state, err := s.repo.UpdateCombatState(playerID, scoutScope, now, func(state *GameState) error {
		nextState, _ := settleResources(*state, now)
		*state = nextState

		if state.NpcState == nil || len(state.NpcState.Cities) == 0 {
			return ErrNpcNotFound
		}

		settleNpcCities(state.NpcState, now)

		var targetNpc *NpcCity
		npcIdx := -1
		for i, city := range state.NpcState.Cities {
			if city.ID == npcID {
				targetNpc = &state.NpcState.Cities[i]
				npcIdx = i
				break
			}
		}
		if targetNpc == nil {
			return ErrNpcNotFound
		}

		scoutUnitID := findScoutUnit(state.Player.Faction)
		if scoutUnitID == "" {
			return ErrNoUnitsSelected
		}

		playerScoutCount := 0
		playerScoutIdx := -1
		for i, u := range state.Army {
			if u.UnitType == scoutUnitID {
				playerScoutCount = u.Amount
				playerScoutIdx = i
				break
			}
		}
		if playerScoutCount <= 0 {
			return ErrInsufficientArmy
		}

		npcScoutUnitID := findScoutUnit(targetNpc.Faction)
		npcScoutCount := 0
		for _, u := range targetNpc.Army {
			if u.UnitType == npcScoutUnitID {
				npcScoutCount = u.Amount
				break
			}
		}

		nowStr := now.UTC().Format(resourceDateLayout)
		playerLost := 0
		npcLost := 0
		scoutSuccess = false

		if npcScoutCount <= 0 {
			scoutSuccess = true
		} else if playerScoutCount > npcScoutCount {
			playerLost = npcScoutCount
			npcLost = npcScoutCount
			scoutSuccess = true
		} else if playerScoutCount == npcScoutCount {
			playerLost = playerScoutCount
			npcLost = npcScoutCount
		} else {
			playerLost = playerScoutCount
			npcLost = playerScoutCount
		}

		if playerLost > 0 && playerScoutIdx >= 0 {
			state.Army[playerScoutIdx].Amount -= playerLost
			if state.Army[playerScoutIdx].Amount <= 0 {
				state.Army = append(state.Army[:playerScoutIdx], state.Army[playerScoutIdx+1:]...)
			}
		}

		if npcLost > 0 && npcScoutUnitID != "" {
			for i := range state.NpcState.Cities[npcIdx].Army {
				if state.NpcState.Cities[npcIdx].Army[i].UnitType == npcScoutUnitID {
					state.NpcState.Cities[npcIdx].Army[i].Amount -= npcLost
					if state.NpcState.Cities[npcIdx].Army[i].Amount < 0 {
						state.NpcState.Cities[npcIdx].Army[i].Amount = 0
					}
					break
				}
			}
			state.NpcState.Cities[npcIdx].ArmySettledAt = nowStr
		}

		reportResult := "attacker_victory"
		if !scoutSuccess {
			reportResult = "defender_victory"
		}

		lostUnits := map[string]int{}
		if playerLost > 0 {
			lostUnits[scoutUnitID] = playerLost
		}
		report = BattleReport{
			ID:               "br_" + randomID(8),
			PlayerID:         state.Player.ID,
			PlayerFaction:    state.Player.Faction,
			PlayerName:       state.Player.Nickname,
			TargetID:         targetNpc.ID,
			TargetName:       targetNpc.Name + "（NPC）",
			Type:             "scout",
			Result:           reportResult,
			PlayerPower:      playerScoutCount,
			EnemyPower:       npcScoutCount,
			DispatchedUnits:  map[string]int{scoutUnitID: playerScoutCount},
			LostUnits:        lostUnits,
			DefenderFaction:  targetNpc.Faction,
			DefenderRevealed: scoutSuccess,
			Rewards:          map[string]int{},
			Read:             false,
			CreatedAt:        nowStr,
		}

		if scoutSuccess {
			report.DefenderUnits = map[string]int{}
			for _, u := range targetNpc.Army {
				if u.Amount > 0 {
					report.DefenderUnits[u.UnitType] = u.Amount
				}
			}
			report.DefenderResources = copyResources(targetNpc.Resources)
			report.DefenderLostUnits = map[string]int{}
			if npcLost > 0 && npcScoutUnitID != "" {
				report.DefenderLostUnits[npcScoutUnitID] = npcLost
			}
			npcCopy := *targetNpc
			npcCopy.Army = append([]ArmyUnit(nil), targetNpc.Army...)
			npcCopy.Resources = copyResources(targetNpc.Resources)
			revealedNpc = &npcCopy
		} else {
			report.DefenderUnits = map[string]int{}
			report.DefenderLostUnits = map[string]int{}
			report.DefenderResources = map[string]int{}
		}

		state.ServerTime = nowStr
		return nil
	})
	if err != nil {
		return ScoutNpcResponse{}, err
	}

	createResult, err := s.CreateBattleReports(BattleReportCreateInput{
		EventID:    report.EventID,
		SourceType: report.SourceType,
		SourceID:   report.TargetID,
		BattleType: report.BattleType,
		Result:     report.Result,
		OccurredAt: report.CreatedAt,
		Reports:    []BattleReport{report},
	})
	if err != nil {
		return ScoutNpcResponse{}, err
	} else if len(createResult.Reports) > 0 {
		report = createResult.Reports[0]
	}
	s.publishBattleFinished(state.Player.ID, report)

	s.attachReportSummary(&state, state.Player.ID)

	// 返回结果
	response := ScoutNpcResponse{
		Success:      scoutSuccess,
		BattleReport: report,
		Army:         state.Army,
		NpcState:     state.NpcState,
		ServerTime:   state.ServerTime,
	}
	if scoutSuccess {
		response.NpcCity = revealedNpc
	}

	return response, nil
}

func (s *Service) publishBattleFinished(playerID string, report BattleReport) {
	s.publishEvent(GameEvent{
		Type:     EventBattleFinished,
		PlayerID: playerID,
		RefType:  report.Type,
		RefID:    report.ID,
		Payload: map[string]any{
			"targetId":         report.TargetID,
			"result":           report.Result,
			"rewards":          report.Rewards,
			"grantedRewards":   report.GrantedRewards,
			"overflowCityGold": report.OverflowCityGold,
			"generalExpGained": report.GeneralExpGained,
		},
		CreatedAt: time.Now().UTC().Format(resourceDateLayout),
	})
}

func (s *Service) publishBattleRewardEvents(playerID string, report BattleReport) {
	if len(report.GrantedRewards) == 0 {
		return
	}
	now := time.Now()
	for _, reward := range report.GrantedRewards {
		s.publishEvent(buildRewardEvent(RewardGrantContext{
			PlayerID: playerID,
			RefType:  LedgerRefBattleReward,
			RefID:    report.ID,
			Reason:   "battle_reward",
		}, playerID, reward, reward.Amount, now))
	}
}

// --- 内部函数 ---

func buildCombatArmy(faction string, units []combat.Unit) combat.Army {
	return combat.Army{
		Faction: faction,
		Units:   units,
	}
}

func buildNpcCombatArmy(npc *NpcCity) combat.Army {
	units := make([]combat.Unit, 0, len(npc.Army))
	factionUnits := GetFactionUnits(npc.Faction)
	traitBuffs := collectTraitBuffs(npc)

	for _, armyUnit := range npc.Army {
		if armyUnit.Amount <= 0 {
			continue
		}
		unitCfg, exists := factionUnits[armyUnit.UnitType]
		if !exists {
			continue
		}
		units = append(units, combat.Unit{
			ID:              armyUnit.UnitType,
			Category:        unitCfg.Category,
			Count:           armyUnit.Amount,
			Attack:          traitBuffs.applyAttack(unitCfg.Stats["attack"]),
			InfantryDefense: traitBuffs.applyInfantryDefense(unitCfg.Stats["infantryDefense"]),
			CavalryDefense:  traitBuffs.applyCavalryDefense(unitCfg.Stats["cavalryDefense"]),
			CarryCapacity:   unitCfg.Stats["carryCapacity"],
			Upkeep:          unitCfg.Stats["upkeep"],
		})
	}

	return combat.Army{
		Faction: npc.Faction,
		Units:   units,
	}
}

func applyNpcBattleResult(state *GameState, npc *NpcCity, result combat.CombatResult, attackerUnits []combat.Unit, mode string, now time.Time) BattleReport {
	nowStr := now.UTC().Format(resourceDateLayout)

	// 记录出征数量（战斗前）
	dispatchedUnits := map[string]int{}
	for _, unit := range attackerUnits {
		dispatchedUnits[unit.ID] = unit.Count
	}

	// 记录防守方兵种（战斗前）；以战斗结果携带的结算快照为权威，避免和战后 NPC 状态混用。
	defenderUnits := map[string]int{}
	for _, loss := range result.DefenderLosses {
		if loss.Count > 0 {
			defenderUnits[loss.ID] = loss.Count
		}
	}

	// 计算玩家损失和存活
	playerLosses := map[string]int{}
	for _, loss := range result.AttackerLosses {
		playerLosses[loss.ID] = loss.Losses
	}

	// 计算防守方损失
	defenderLostUnits := map[string]int{}
	for _, loss := range result.DefenderLosses {
		if loss.Losses > 0 {
			defenderLostUnits[loss.ID] = loss.Losses
		}
	}

	// 判断防守方是否暴露（战损 >= 25%）
	totalDefenderBefore := 0
	for _, u := range defenderUnits {
		totalDefenderBefore += u
	}
	totalDefenderLost := 0
	for _, v := range defenderLostUnits {
		totalDefenderLost += v
	}
	defenderRevealed := totalDefenderBefore == 0 || float64(totalDefenderLost)/float64(totalDefenderBefore) >= 0.25

	// 如果未暴露，清空防守方详细信息
	if !defenderRevealed {
		defenderUnits = map[string]int{}
		defenderLostUnits = map[string]int{}
	}

	// 归还存活部队
	for _, unit := range attackerUnits {
		survived := unit.Count - playerLosses[unit.ID]
		if survived > 0 {
			addToArmy(&state.Army, unit.ID, survived)
		}
	}

	// 扣减 NPC 守军
	for _, loss := range result.DefenderLosses {
		if loss.Losses <= 0 {
			continue
		}
		for i := range npc.Army {
			if npc.Army[i].UnitType == loss.ID {
				npc.Army[i].Amount -= loss.Losses
				if npc.Army[i].Amount < 0 {
					npc.Army[i].Amount = 0
				}
				break
			}
		}
	}
	// 重置守军结算时间
	npc.ArmySettledAt = nowStr

	// 掠夺资源（仅进攻方胜时）
	plundered := map[string]int{}
	overflowDetail := map[string]int{}
	overflowCityGold := 0
	if result.Winner == "attacker" && result.SurvivingCarry > 0 {
		plundered = calculatePlunder(npc, result.SurvivingCarry)
		// 扣减 NPC 资源
		for resType, amount := range plundered {
			npc.Resources[resType] -= amount
			if npc.Resources[resType] < 0 {
				npc.Resources[resType] = 0
			}
		}
		// 重置资源结算时间
		npc.ResourceSettledAt = nowStr

		// 资源入库玩家（溢出部分按比例转城金）
		totalOverflow := 0
		for resType, amount := range plundered {
			_, overflow, _ := addResourceCapped(state, resType, amount)
			if overflow > 0 {
				totalOverflow += overflow
				overflowDetail[resType] = overflow
			}
		}
		// 溢出转城金
		overflowRate := currentBalance().OverflowToCityGold
		if overflowRate <= 0 {
			overflowRate = 200
		}
		if totalOverflow >= overflowRate {
			overflowCityGold = totalOverflow / overflowRate
			// 注意：不在此处直接修改 state.CityGold，由调用方通过原子操作处理
		}
	}

	// 生成战报
	reportResult := "attacker_victory"
	if result.Winner == "defender" {
		reportResult = "defender_victory"
	} else if result.Winner == "draw" {
		reportResult = "draw"
	}

	report := BattleReport{
		ID:                "br_" + randomID(8),
		PlayerID:          state.Player.ID,
		PlayerFaction:     state.Player.Faction,
		PlayerName:        state.Player.Nickname,
		TargetID:          npc.ID,
		TargetName:        npc.Name + "（NPC）",
		Type:              mode,
		Result:            reportResult,
		PlayerPower:       int(result.AttackPower),
		EnemyPower:        int(result.DefensePower),
		DispatchedUnits:   dispatchedUnits,
		LostUnits:         playerLosses,
		DefenderFaction:   npc.Faction,
		DefenderUnits:     defenderUnits,
		DefenderLostUnits: defenderLostUnits,
		DefenderRevealed:  defenderRevealed,
		DefenderResources: copyResources(npc.Resources),
		Rewards:           plundered,
		Overflow:          overflowDetail,
		OverflowCityGold:  overflowCityGold,
		Read:              false,
		CreatedAt:         nowStr,
	}

	// 战报通过统一战报服务独立存储，不再内嵌到 state。
	return report
}

func buildBattleGrantedRewards(report BattleReport) []Reward {
	rewards := []Reward{}
	for resourceID, amount := range report.Rewards {
		appliedAmount := amount - report.Overflow[resourceID]
		if appliedAmount <= 0 {
			continue
		}
		rewards = append(rewards, Reward{
			Type:   RewardTypeResource,
			ID:     resourceID,
			Amount: appliedAmount,
		})
	}
	if report.OverflowCityGold > 0 {
		rewards = append(rewards, Reward{
			Type:   RewardTypeCityGold,
			ID:     RewardTypeCityGold,
			Amount: report.OverflowCityGold,
		})
	}
	if report.GeneralExpGained > 0 {
		rewards = append(rewards, Reward{
			Type:   RewardTypeGeneralExp,
			ID:     "current_general",
			Amount: report.GeneralExpGained,
		})
	}
	for _, drop := range report.Drops {
		if drop.Type == RewardTypeItem && drop.ItemID != "" && drop.Amount > 0 {
			rewards = append(rewards, Reward{
				Type:   RewardTypeItem,
				ID:     drop.ItemID,
				Amount: drop.Amount,
			})
		}
	}
	return rewards
}

// rollNpcBattleDrops 按 NPC 层级绑定的掉落池生成胜利掉落。
func rollNpcBattleDrops(npc *NpcCity, report BattleReport) ([]Reward, []BattleReportDrop, error) {
	if npc == nil || report.Result != "attacker_victory" {
		return nil, nil, nil
	}
	cfg := GetNpcConfig()
	tierCfg, ok := cfg.Tiers[strings.TrimSpace(npc.Tier)]
	if !ok || strings.TrimSpace(tierCfg.DropPoolID) == "" {
		return nil, nil, nil
	}
	rewards, err := RollDropPoolRewards(tierCfg.DropPoolID)
	if err != nil {
		return nil, nil, err
	}
	return rewards, buildBattleReportDrops(rewards), nil
}

// buildBattleReportDrops 把标准奖励转换为战报掉落快照。
func buildBattleReportDrops(rewards []Reward) []BattleReportDrop {
	drops := []BattleReportDrop{}
	for _, reward := range rewards {
		if reward.Type != RewardTypeItem || reward.Amount <= 0 {
			continue
		}
		drop := BattleReportDrop{
			Type:   reward.Type,
			ItemID: reward.ID,
			Amount: reward.Amount,
		}
		if item, ok := GetItemDefinition(reward.ID); ok {
			drop.Name = item.Name
			drop.Quality = item.Quality
		}
		drops = append(drops, drop)
	}
	return drops
}

func calculatePlunder(npc *NpcCity, carryCapacity int) map[string]int {
	// 计算 NPC 总资源
	totalResources := 0
	for _, amount := range npc.Resources {
		totalResources += amount
	}

	if totalResources <= 0 {
		return map[string]int{}
	}

	// 实际可掠夺 = min(运载量, 总资源)
	effectiveCarry := carryCapacity
	if effectiveCarry > totalResources {
		effectiveCarry = totalResources
	}

	// 按比例分配
	plundered := map[string]int{}
	assigned := 0

	type resEntry struct {
		key      string
		amount   int
		fraction float64
	}
	var entries []resEntry

	for resType, amount := range npc.Resources {
		if amount <= 0 {
			continue
		}
		exact := float64(effectiveCarry) * float64(amount) / float64(totalResources)
		floor := int(exact)
		if floor > amount {
			floor = amount
		}
		plundered[resType] = floor
		assigned += floor
		entries = append(entries, resEntry{resType, amount, exact - float64(floor)})
	}

	// 补余数
	remaining := effectiveCarry - assigned
	// 按小数部分从大到小排序
	for i := 0; i < len(entries)-1; i++ {
		for j := i + 1; j < len(entries); j++ {
			if entries[j].fraction > entries[i].fraction {
				entries[i], entries[j] = entries[j], entries[i]
			}
		}
	}
	for _, e := range entries {
		if remaining <= 0 {
			break
		}
		if plundered[e.key] < e.amount {
			plundered[e.key]++
			remaining--
		}
	}

	return plundered
}

func copyResources(src map[string]int) map[string]int {
	if src == nil {
		return map[string]int{}
	}
	dst := make(map[string]int, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

func findScoutUnit(faction string) string {
	factionUnits := GetFactionUnits(faction)
	if factionUnits == nil {
		return ""
	}
	for unitID, unit := range factionUnits {
		if unit.Role == "scout" {
			return unitID
		}
	}
	return ""
}

// GetReportByID 供 GM 内部查询按主键读取原始战报。
func (s *Service) GetReportByID(reportID string) (BattleReport, error) {
	report, err := s.repo.GetReportByID(reportID)
	if err != nil {
		return BattleReport{}, err
	}
	return NormalizeBattleReport(report), nil
}

// GetReportForPlayer 获取玩家自己的标准战报详情。
func (s *Service) GetReportForPlayer(playerID string, reportID string) (BattleReport, error) {
	playerID = strings.TrimSpace(playerID)
	reportID = strings.TrimSpace(reportID)
	if playerID == "" || reportID == "" {
		return BattleReport{}, ErrPlayerNotFound
	}
	report, err := s.repo.GetReportForPlayer(playerID, reportID)
	if err != nil {
		return BattleReport{}, err
	}
	return projectBattleReportForViewer(report), nil
}

// GetReportEventForPlayer 获取玩家可见的同事件战报上下文。
func (s *Service) GetReportEventForPlayer(playerID string, reportID string) (BattleReportEventContext, error) {
	report, err := s.GetReportForPlayer(playerID, reportID)
	if err != nil {
		return BattleReportEventContext{}, err
	}
	eventID := strings.TrimSpace(report.EventID)
	if eventID == "" {
		return BattleReportEventContext{}, errors.New("event not found")
	}
	event, err := s.GetBattleEventForAdmin(eventID)
	if err != nil {
		return BattleReportEventContext{}, err
	}
	allReports, err := s.ListReportsByEventForAdmin(eventID)
	if err != nil {
		return BattleReportEventContext{}, err
	}
	reports := make([]BattleReport, 0, len(allReports))
	for _, item := range allReports {
		if item.PlayerID == playerID || item.OwnerPlayerID == playerID {
			reports = append(reports, projectBattleReportForViewer(item))
		}
	}
	allParticipants, err := s.ListParticipantsByEventForAdmin(eventID)
	if err != nil {
		return BattleReportEventContext{}, err
	}
	participants := make([]BattleReportParticipant, 0, len(allParticipants))
	for _, item := range allParticipants {
		if item.PlayerID == playerID {
			participants = append(participants, item)
		}
	}
	return BattleReportEventContext{Event: projectBattleEventForViewer(event), Reports: reports, Participants: participants}, nil
}

// projectBattleEventForViewer 玩家同事件上下文只保留事件索引，不返回 GM 原始快照和结算诊断数据。
func projectBattleEventForViewer(event BattleEvent) BattleEvent {
	event.Summary = nil
	event.Snapshot = nil
	event.ResultData = nil
	return event
}

// GetSharedReportByToken 通过分享 token 读取公开战报。
func (s *Service) GetSharedReportByToken(token string) (BattleReport, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return BattleReport{}, errors.New("token is required")
	}
	report, err := s.repo.GetReportByShareToken(token)
	if err != nil {
		return BattleReport{}, err
	}
	return projectBattleReportForViewer(report), nil
}

// ListReports 分页获取玩家军情战报。
func (s *Service) ListReports(playerID string, page int, pageSize int) (BattleReportPage, error) {
	playerID = strings.TrimSpace(playerID)
	if playerID == "" {
		return BattleReportPage{}, ErrPlayerNotFound
	}
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}
	if pageSize > 50 {
		pageSize = 50
	}

	offset := (page - 1) * pageSize
	reports, total, err := s.repo.ListReports(playerID, pageSize, offset)
	if err != nil {
		return BattleReportPage{}, err
	}

	return BattleReportPage{
		Reports:  reports,
		Page:     page,
		PageSize: pageSize,
		Total:    total,
	}, nil
}

// ListReportsByQuery 按标准查询条件分页获取玩家战报。
func (s *Service) ListReportsByQuery(query BattleReportQuery) (BattleReportPage, error) {
	query.PlayerID = strings.TrimSpace(query.PlayerID)
	if query.PlayerID == "" {
		return BattleReportPage{}, ErrPlayerNotFound
	}
	if query.Page < 1 {
		query.Page = 1
	}
	if query.PageSize < 1 {
		query.PageSize = 10
	}
	if query.PageSize > 50 {
		query.PageSize = 50
	}
	reports, total, err := s.repo.ListReportsByQuery(query)
	if err != nil {
		return BattleReportPage{}, err
	}
	return BattleReportPage{
		Reports:  reports,
		Page:     query.Page,
		PageSize: query.PageSize,
		Total:    total,
	}, nil
}

// ShareBattleReport 为玩家战报创建分享 token。
func (s *Service) ShareBattleReport(playerID string, reportID string) (BattleReportShareLink, error) {
	playerID = strings.TrimSpace(playerID)
	reportID = strings.TrimSpace(reportID)
	if playerID == "" || reportID == "" {
		return BattleReportShareLink{}, errors.New("playerId and reportId are required")
	}
	return s.repo.CreateBattleReportShareLink(playerID, reportID, "public", time.Time{})
}

// CreateBattleReports 是玩法模块接入统一战报系统的标准入口。
func (s *Service) CreateBattleReports(input BattleReportCreateInput) (BattleReportCreateResult, error) {
	if len(input.Reports) == 0 {
		return BattleReportCreateResult{}, errors.New("reports are required")
	}
	occurredAt := strings.TrimSpace(input.OccurredAt)
	if occurredAt == "" {
		occurredAt = time.Now().UTC().Format(time.RFC3339)
	}
	eventID := strings.TrimSpace(input.EventID)
	if eventID == "" {
		eventID = "event_" + randomID(12)
	}
	reports := make([]BattleReport, 0, len(input.Reports))
	for _, report := range input.Reports {
		if strings.TrimSpace(report.ID) == "" {
			return BattleReportCreateResult{}, errors.New("report id is required")
		}
		if strings.TrimSpace(report.PlayerID) == "" {
			return BattleReportCreateResult{}, errors.New("report playerId is required")
		}
		report.EventID = valueOrDefault(report.EventID, eventID)
		report.SourceType = valueOrDefault(report.SourceType, input.SourceType)
		report.BattleType = valueOrDefault(report.BattleType, input.BattleType)
		report.Result = valueOrDefault(report.Result, input.Result)
		report.OwnerOutcome = valueOrDefault(report.OwnerOutcome, input.OwnerOutcome)
		report.CreatedAt = valueOrDefault(report.CreatedAt, occurredAt)
		report = NormalizeBattleReport(report)
		reports = append(reports, report)
	}
	event := buildBattleEventFromReports(input, eventID, occurredAt, reports)
	if err := s.saveBattleReportsWithConfirmation(event, reports); err != nil {
		return BattleReportCreateResult{}, err
	}
	visibleReports := make([]BattleReport, 0, len(reports))
	for _, report := range reports {
		visibleReports = append(visibleReports, projectBattleReportForViewer(report))
	}
	return BattleReportCreateResult{Event: event, Reports: visibleReports}, nil
}

// saveBattleReportsWithConfirmation 保存战报并在失败时重试、查回确认，避免战斗已完成但战报静默丢失。
func (s *Service) saveBattleReportsWithConfirmation(event BattleEvent, reports []BattleReport) error {
	var lastErr error
	for attempt := 1; attempt <= battleReportSaveAttempts; attempt++ {
		if err := s.repo.SaveReportBundle(event, reports); err != nil {
			lastErr = err
			if s.battleReportBundlePersisted(event, reports) {
				return nil
			}
			if attempt < battleReportSaveAttempts {
				slog.Warn("battle report save retry", "attempt", attempt, "error", err)
				time.Sleep(time.Duration(attempt) * battleReportSaveRetryDelay)
				continue
			}
			break
		}
		return nil
	}
	if s.battleReportBundlePersisted(event, reports) {
		return nil
	}
	return fmt.Errorf("battle report save failed after %d attempts: %w", battleReportSaveAttempts, lastErr)
}

// battleReportBundlePersisted 同时确认事件和全部玩家视角战报已经落库。
func (s *Service) battleReportBundlePersisted(event BattleEvent, reports []BattleReport) bool {
	if !s.battleReportsPersisted(reports) {
		return false
	}
	if strings.TrimSpace(event.ID) == "" {
		return true
	}
	stored, err := s.repo.GetBattleEventForAdmin(event.ID)
	return err == nil && stored.ID == event.ID
}

// battleReportsPersisted 按玩家和战报 ID 查回确认，用于处理提交成功但连接返回失败的边界情况。
func (s *Service) battleReportsPersisted(reports []BattleReport) bool {
	if len(reports) == 0 {
		return false
	}
	for _, report := range reports {
		if strings.TrimSpace(report.ID) == "" || strings.TrimSpace(report.PlayerID) == "" {
			return false
		}
		if _, err := s.repo.GetReportForPlayer(report.PlayerID, report.ID); err != nil {
			return false
		}
	}
	return true
}

// buildBattleEventFromReports 从标准战报创建输入生成事件快照。
func buildBattleEventFromReports(input BattleReportCreateInput, eventID string, occurredAt string, reports []BattleReport) BattleEvent {
	first := reports[0]
	event := BuildBattleEventFromReport(first)
	sourceType := valueOrDefault(input.SourceType, first.SourceType)
	battleType := valueOrDefault(input.BattleType, first.BattleType)
	result := valueOrDefault(input.Result, first.Result)
	sourceID := valueOrDefault(input.SourceID, first.TargetID)
	createdAt := occurredAt
	event.ID = eventID
	event.SourceType = sourceType
	event.SourceID = sourceID
	event.Scene = first.ViewType
	event.BattleType = battleType
	event.Result = result
	event.RelatedMarchID = input.RelatedMarchID
	event.RelatedReinforcementID = input.RelatedReinforcementID
	event.Summary = input.Extra
	event.OccurredAt = occurredAt
	event.CreatedAt = createdAt
	return event
}

// BuildBattleEventFromReport 从标准双方快照推导事件攻守方，兼容只有防守视角的战报。
func BuildBattleEventFromReport(report BattleReport) BattleEvent {
	report = NormalizeBattleReport(report)
	event := BattleEvent{
		ID:         report.EventID,
		SourceType: report.SourceType,
		SourceID:   report.TargetID,
		Scene:      report.ViewType,
		BattleType: report.BattleType,
		Result:     report.Result,
		OccurredAt: report.CreatedAt,
		CreatedAt:  report.CreatedAt,
	}
	if report.Detail == nil {
		event.AttackerPlayerID = report.PlayerID
		event.DefenderPlayerID = report.TargetID
		event.AttackerName = report.PlayerName
		event.DefenderName = report.TargetName
		event.AttackerFaction = report.PlayerFaction
		event.DefenderFaction = report.DefenderFaction
		return event
	}
	attacker := report.Detail.PrimarySide
	event.AttackerPlayerID = battleReportSideID(attacker)
	event.AttackerName = battleReportSideName(attacker)
	event.AttackerFaction = attacker.Faction
	if report.Detail.SecondarySide != nil {
		defender := *report.Detail.SecondarySide
		event.DefenderPlayerID = battleReportSideID(defender)
		event.DefenderName = battleReportSideName(defender)
		event.DefenderFaction = defender.Faction
	}
	event.Summary = report.Detail.Extra
	return event
}

// battleReportSideID 返回参战方的玩家或目标标识。
func battleReportSideID(side BattleReportSide) string {
	return valueOrDefault(side.PlayerID, valueOrDefault(side.TargetID, side.CityID))
}

// battleReportSideName 返回参战方的玩家、目标或城池名称。
func battleReportSideName(side BattleReportSide) string {
	return valueOrDefault(side.PlayerName, valueOrDefault(side.TargetName, side.CityName))
}

// projectBattleReportForViewer 对玩家和公开分享响应执行服务端情报脱敏，原始快照仍仅供 GM 排查。
func projectBattleReportForViewer(raw BattleReport) BattleReport {
	report := cloneBattleReport(raw)
	report = NormalizeBattleReport(report)
	if report.Detail == nil {
		return report
	}
	visibility := report.Detail.Visibility
	enemy := battleReportEnemySide(&report)
	if !visibility.ShowEnemyRemainingUnits {
		report.DefenderUnits = map[string]int{}
		if enemy != nil {
			enemy.Units = []BattleReportUnit{}
		}
		if report.OwnerSide == ReportOwnerSideAttacker || report.OwnerSide == ReportOwnerSideScout {
			report.Detail.Extra = cloneBattleReportExtraForRedaction(report.Detail.Extra)
			for i := range report.PvpReinforcements {
				report.PvpReinforcements[i].Troops = map[string]int{}
			}
			redactReportExtraReinforcementTroops(report.Detail.Extra)
		}
	}
	if !visibility.ShowEnemyResources {
		report.DefenderResources = map[string]int{}
		if enemy != nil {
			enemy.Resources = map[string]int{}
		}
	}
	if !visibility.ShowEnemyGenerals {
		if enemy != nil {
			enemy.Generals = []BattleReportGeneral{}
		}
		if report.OwnerSide == ReportOwnerSideDefender {
			report.PvpAttackerGenerals = nil
		} else {
			report.PvpDefenderGenerals = nil
			report.Detail.Extra = cloneBattleReportExtraForRedaction(report.Detail.Extra)
			for i := range report.PvpReinforcements {
				report.PvpReinforcements[i].Generals = nil
			}
			redactReportExtraReinforcementGenerals(report.Detail.Extra)
		}
	}
	if !visibility.ShowEnemyCityDefense {
		report.PvpWall = nil
		report.Detail.Extra = cloneBattleReportExtraForRedaction(report.Detail.Extra)
		redactReportExtraCityDefense(report.Detail.Extra)
	}
	return report
}

// cloneBattleReport 深拷贝战报，避免响应脱敏修改仓储中的原始快照。
func cloneBattleReport(report BattleReport) BattleReport {
	cloned := report
	cloned.DispatchedUnits = cloneStringIntMap(report.DispatchedUnits)
	cloned.LostUnits = cloneStringIntMap(report.LostUnits)
	cloned.DefenderUnits = cloneStringIntMap(report.DefenderUnits)
	cloned.DefenderLostUnits = cloneStringIntMap(report.DefenderLostUnits)
	cloned.DefenderResources = cloneStringIntMap(report.DefenderResources)
	cloned.Rewards = cloneStringIntMap(report.Rewards)
	cloned.PvpAttackerGenerals = append([]PvpGeneralSnapshot(nil), report.PvpAttackerGenerals...)
	cloned.PvpDefenderGenerals = append([]PvpGeneralSnapshot(nil), report.PvpDefenderGenerals...)
	cloned.PvpReinforcements = append([]DefenseReinforcementUnit(nil), report.PvpReinforcements...)
	for i := range cloned.PvpReinforcements {
		cloned.PvpReinforcements[i].Troops = cloneStringIntMap(report.PvpReinforcements[i].Troops)
	}
	if report.Detail != nil {
		detail := *report.Detail
		detail.PrimarySide = cloneBattleReportSide(report.Detail.PrimarySide)
		if report.Detail.SecondarySide != nil {
			secondary := cloneBattleReportSide(*report.Detail.SecondarySide)
			detail.SecondarySide = &secondary
		}
		cloned.Detail = &detail
	}
	return cloned
}

// cloneBattleReportSide 复制响应脱敏会修改的参与方集合。
func cloneBattleReportSide(side BattleReportSide) BattleReportSide {
	cloned := side
	cloned.Units = append([]BattleReportUnit(nil), side.Units...)
	cloned.Resources = cloneStringIntMap(side.Resources)
	cloned.Generals = append([]BattleReportGeneral(nil), side.Generals...)
	return cloned
}

// cloneBattleReportExtraForRedaction 仅在需要删除动态情报时复制 Extra，保留普通响应里的强类型扩展值。
func cloneBattleReportExtraForRedaction(extra map[string]interface{}) map[string]interface{} {
	data, err := json.Marshal(extra)
	if err != nil {
		return map[string]interface{}{}
	}
	cloned := map[string]interface{}{}
	if err := json.Unmarshal(data, &cloned); err != nil {
		return map[string]interface{}{}
	}
	return cloned
}

// battleReportEnemySide 返回当前拥有者视角下的敌方标准快照。
func battleReportEnemySide(report *BattleReport) *BattleReportSide {
	if report == nil || report.Detail == nil {
		return nil
	}
	if report.OwnerSide == ReportOwnerSideDefender {
		return &report.Detail.PrimarySide
	}
	if report.OwnerSide == ReportOwnerSideReinforcement {
		return report.Detail.SecondarySide
	}
	return report.Detail.SecondarySide
}

// redactReportExtraReinforcementTroops 清除动态 extra 中可反推敌方剩余兵力的援军快照。
func redactReportExtraReinforcementTroops(extra map[string]interface{}) {
	pvp, ok := extra["pvp"].(map[string]interface{})
	if !ok {
		return
	}
	items, ok := pvp["reinforcements"].([]interface{})
	if !ok {
		return
	}
	for _, item := range items {
		if snapshot, ok := item.(map[string]interface{}); ok {
			snapshot["troops"] = map[string]interface{}{}
		}
	}
}

// redactReportExtraReinforcementGenerals 清除动态援军快照里的敌方武将情报。
func redactReportExtraReinforcementGenerals(extra map[string]interface{}) {
	pvp, ok := extra["pvp"].(map[string]interface{})
	if !ok {
		return
	}
	items, ok := pvp["reinforcements"].([]interface{})
	if !ok {
		return
	}
	for _, item := range items {
		if snapshot, ok := item.(map[string]interface{}); ok {
			snapshot["generals"] = []interface{}{}
		}
	}
}

// redactReportExtraCityDefense 清除当前视角不可见的城防快照。
func redactReportExtraCityDefense(extra map[string]interface{}) {
	pvp, ok := extra["pvp"].(map[string]interface{})
	if ok {
		delete(pvp, "wall")
	}
}

// ListBattleEventsForAdmin 返回 GM 战斗事件列表。
func (s *Service) ListBattleEventsForAdmin(query BattleEventQuery) (BattleEventPage, error) {
	if query.Page < 1 {
		query.Page = 1
	}
	if query.PageSize < 1 {
		query.PageSize = 20
	}
	if query.PageSize > 100 {
		query.PageSize = 100
	}
	items, total, err := s.repo.ListBattleEventsForAdmin(query)
	if err != nil {
		return BattleEventPage{}, err
	}
	return BattleEventPage{Items: items, Page: query.Page, PageSize: query.PageSize, Total: total}, nil
}

// GetBattleEventForAdmin 返回 GM 单个战斗事件详情。
func (s *Service) GetBattleEventForAdmin(eventID string) (BattleEvent, error) {
	eventID = strings.TrimSpace(eventID)
	if eventID == "" {
		return BattleEvent{}, errors.New("eventId is required")
	}
	return s.repo.GetBattleEventForAdmin(eventID)
}

// ListReportsByEventForAdmin 返回同一事件下所有玩家视角战报。
func (s *Service) ListReportsByEventForAdmin(eventID string) ([]BattleReport, error) {
	eventID = strings.TrimSpace(eventID)
	if eventID == "" {
		return nil, errors.New("eventId is required")
	}
	return s.repo.ListReportsByEventForAdmin(eventID)
}

// ListParticipantsByEventForAdmin 返回同一事件下所有参与方快照。
func (s *Service) ListParticipantsByEventForAdmin(eventID string) ([]BattleReportParticipant, error) {
	eventID = strings.TrimSpace(eventID)
	if eventID == "" {
		return nil, errors.New("eventId is required")
	}
	return s.repo.ListParticipantsByEventForAdmin(eventID)
}

// MarkReportsRead 标记所有战报为已读，并返回战报未读数局部结果。
func (s *Service) MarkReportsRead(playerID string) (ReportActionResult, error) {
	playerID = strings.TrimSpace(playerID)
	if playerID == "" {
		return ReportActionResult{}, ErrPlayerNotFound
	}

	if err := s.repo.MarkReportsRead(playerID); err != nil {
		return ReportActionResult{}, err
	}

	return s.buildReportActionResult(playerID)
}

// MarkSingleReportRead 标记单条战报为已读，并返回战报未读数局部结果。
func (s *Service) MarkSingleReportRead(playerID string, reportID string) (ReportActionResult, error) {
	playerID = strings.TrimSpace(playerID)
	reportID = strings.TrimSpace(reportID)
	if playerID == "" {
		return ReportActionResult{}, ErrPlayerNotFound
	}

	if err := s.repo.MarkSingleReportRead(playerID, reportID); err != nil {
		return ReportActionResult{}, err
	}

	return s.buildReportActionResult(playerID)
}

// MarkReportsReadByView 标记指定视角 Tab 的战报为已读。
func (s *Service) MarkReportsReadByView(playerID string, viewType string) (ReportActionResult, error) {
	playerID = strings.TrimSpace(playerID)
	viewType = strings.TrimSpace(viewType)
	if playerID == "" {
		return ReportActionResult{}, ErrPlayerNotFound
	}
	if err := s.repo.MarkReportsReadByView(playerID, viewType); err != nil {
		return ReportActionResult{}, err
	}
	return s.buildReportActionResult(playerID)
}

// DeleteReport 删除单条战报，并返回战报未读数局部结果。
func (s *Service) DeleteReport(playerID string, reportID string) (ReportActionResult, error) {
	playerID = strings.TrimSpace(playerID)
	reportID = strings.TrimSpace(reportID)
	if playerID == "" {
		return ReportActionResult{}, ErrPlayerNotFound
	}
	if reportID == "" {
		return ReportActionResult{}, errors.New("reportId is required")
	}

	if err := s.repo.DeleteReport(playerID, reportID); err != nil {
		return ReportActionResult{}, err
	}

	return s.buildReportActionResult(playerID)
}

// DeleteReportsByView 删除指定视角 Tab 下的战报，保留旧调用兼容。
func (s *Service) DeleteReportsByView(playerID string, viewType string) (ReportActionResult, error) {
	return s.DeleteReportsByFilter(BattleReportDeleteFilter{PlayerID: playerID, ViewType: viewType})
}

// DeleteReportsByFilter 按与列表一致的视角和战斗类型筛选批量删除战报。
func (s *Service) DeleteReportsByFilter(filter BattleReportDeleteFilter) (ReportActionResult, error) {
	playerID := strings.TrimSpace(filter.PlayerID)
	filter.PlayerID = playerID
	filter.ViewType = strings.TrimSpace(filter.ViewType)
	filter.BattleType = strings.TrimSpace(filter.BattleType)
	if playerID == "" {
		return ReportActionResult{}, ErrPlayerNotFound
	}
	if err := s.repo.DeleteReportsByFilter(filter); err != nil {
		return ReportActionResult{}, err
	}
	return s.buildReportActionResult(playerID)
}

// DeleteAllReports 一键删除所有战报，并返回战报未读数局部结果。
func (s *Service) DeleteAllReports(playerID string) (ReportActionResult, error) {
	playerID = strings.TrimSpace(playerID)
	if playerID == "" {
		return ReportActionResult{}, ErrPlayerNotFound
	}

	if err := s.repo.DeleteAllReports(playerID); err != nil {
		return ReportActionResult{}, err
	}

	return s.buildReportActionResult(playerID)
}

// buildReportActionResult 统计战报未读数，避免战报写操作回读完整玩家状态。
func (s *Service) buildReportActionResult(playerID string) (ReportActionResult, error) {
	unread, err := s.repo.CountUnreadReports(playerID)
	if err != nil {
		return ReportActionResult{}, err
	}
	return ReportActionResult{
		UnreadMessageCount: unread,
		ServerTime:         time.Now().UTC().Format(resourceDateLayout),
	}, nil
}
