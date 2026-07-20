// 本文件实现轮回绝境副本的开启、战斗、波次推进和奖励结算。
package game

import (
	"errors"
	"fmt"
	"math"
	mathrand "math/rand"
	"sort"
	"strings"
	"time"

	"hero3/internal/core/combat"
	"hero3/internal/core/general"
)

// reincarnationCombatResolution 保存轮回模块复用核心事件管线后的阶段结果。
type reincarnationCombatResolution struct {
	Result      combat.CombatResult
	Before      *general.BeforeBattleContext
	AfterCombat *general.AfterCombatResolveContext
}

// GetReincarnationConfigForPlayer 返回轮回绝境配置。
func (s *Service) GetReincarnationConfigForPlayer() ReincarnationConfig {
	return GetReincarnationConfig()
}

// GetActiveReincarnationRun 获取玩家当前轮回实例，并顺手处理超时状态。
func (s *Service) GetActiveReincarnationRun(playerID string) (ReincarnationRunResponse, error) {
	now := time.Now().UTC()
	run, found, err := s.repo.GetActiveReincarnationRun(playerID, now)
	if err != nil {
		return ReincarnationRunResponse{}, err
	}
	if found && run.Status == ReincarnationRunRunning && now.After(run.ExpiresAt) {
		settled, err := s.SettleReincarnationRun(playerID)
		if err != nil {
			return ReincarnationRunResponse{}, err
		}
		run = settled.Run
	}
	state, err := s.repo.GetState(playerID)
	if err != nil {
		return ReincarnationRunResponse{}, err
	}
	hydrateStateForResponse(&state, now)
	if !found {
		return ReincarnationRunResponse{Army: state.Army, ServerTime: now.Format(resourceDateLayout)}, nil
	}
	return ReincarnationRunResponse{Run: &run, Army: state.Army, ServerTime: now.Format(resourceDateLayout)}, nil
}

// StartReincarnationRun 开启一个新的轮回绝境实例。
func (s *Service) StartReincarnationRun(playerID string, level int) (ReincarnationActionResult, error) {
	now := time.Now().UTC()
	if _, found, err := s.repo.GetActiveReincarnationRun(playerID, now); err != nil {
		return ReincarnationActionResult{}, err
	} else if found {
		return ReincarnationActionResult{}, ErrReincarnationActive
	}
	levelCfg, ok := reincarnationLevelConfig(level)
	if !ok || !levelCfg.Enabled {
		return ReincarnationActionResult{}, ErrInvalidReincarnation
	}
	state, err := s.repo.GetState(playerID)
	if err != nil {
		return ReincarnationActionResult{}, err
	}
	run := s.buildReincarnationRun(state, levelCfg, now)
	if err := s.repo.SaveReincarnationRun(run); err != nil {
		return ReincarnationActionResult{}, err
	}
	hydrateStateForResponse(&state, now)
	return ReincarnationActionResult{Run: run, Army: state.Army, ServerTime: now.Format(resourceDateLayout)}, nil
}

// ResetReincarnationWaveBonus 消耗账户金币重置当前波随机加成。
func (s *Service) ResetReincarnationWaveBonus(playerID string, waveID string) (ReincarnationActionResult, error) {
	now := time.Now().UTC()
	playerID = strings.TrimSpace(playerID)
	waveID = strings.TrimSpace(waveID)
	if playerID == "" {
		return ReincarnationActionResult{}, ErrPlayerNotFound
	}
	if waveID == "" {
		return ReincarnationActionResult{}, ErrInvalidReincarnation
	}
	run, err := s.repo.GetReincarnationRun(runIDFromWaveID(waveID))
	if err != nil {
		return ReincarnationActionResult{}, err
	}
	if run.PlayerID != playerID || run.Status != ReincarnationRunRunning || now.After(run.ExpiresAt) {
		return ReincarnationActionResult{}, ErrInvalidReincarnation
	}
	wave := activeReincarnationWave(&run)
	if wave == nil || wave.ID != waveID || wave.Status != ReincarnationWaveActive || hasReincarnationBattleForWave(run, waveID) {
		return ReincarnationActionResult{}, ErrInvalidReincarnation
	}
	cost := GetReincarnationConfig().BonusResetGoldCost
	if cost <= 0 {
		return ReincarnationActionResult{}, ErrInvalidReincarnation
	}
	accountID, err := s.repo.GetAccountIDByPlayerID(playerID)
	if err != nil {
		return ReincarnationActionResult{}, err
	}
	account, state, err := s.repo.UpdateAccountBuildingResourceState(accountID, playerID, now, func(account *Account, state *GameState) error {
		if account.Gold < cost {
			return ErrInsufficientGold
		}
		account.Gold -= cost
		state.ServerTime = now.Format(resourceDateLayout)
		return nil
	})
	if err != nil {
		return ReincarnationActionResult{}, err
	}
	rng := mathrand.New(mathrand.NewSource(now.UnixNano()))
	wave.AllyBonus = buildReincarnationBonus("ally", state.Player.Faction, wave.WaveType, rng)
	wave.EnemyBonus = buildReincarnationBonus("enemy", wave.EnemyFaction, wave.WaveType, rng)
	run.UpdatedAt = now
	if err := s.repo.SaveReincarnationRun(run); err != nil {
		return ReincarnationActionResult{}, err
	}
	s.recordLedger(GoldLedgerEntry{
		AccountID:    account.ID,
		PlayerID:     playerID,
		Currency:     LedgerCurrencyGold,
		Direction:    LedgerDirectionDebit,
		Amount:       cost,
		BalanceAfter: account.Gold,
		RefType:      LedgerRefReincarnationBonusReset,
		RefID:        waveID,
	})
	s.publishCurrencyChanged(playerID, account.ID, waveID, LedgerRefReincarnationBonusReset)
	hydrateStateForResponse(&state, now)
	result := buildReincarnationActionResult(run, nil, state, now)
	result.AccountGold = account.Gold
	result.Cost = cost
	return result, nil
}

// AttackReincarnationWave 结算一次进攻波出兵。
func (s *Service) AttackReincarnationWave(playerID string, waveID string, troops map[string]int, generalIDs []string, clientActionID string) (ReincarnationActionResult, error) {
	now := time.Now().UTC()
	var report BattleReport
	duplicateReportID := ""
	state, run, reports, err := s.repo.UpdateReincarnationRunWithState(playerID, runIDFromWaveID(waveID), now, func(state *GameState, run *ReincarnationRun) ([]BattleReport, error) {
		EnsureGeneralRoster(state, now)
		if existing := findReincarnationBattleByAction(*run, waveID, clientActionID); existing != nil {
			duplicateReportID = existing.ReportID
			return nil, nil
		}
		wave := activeReincarnationWave(run)
		if wave == nil || wave.ID != waveID || wave.WaveType != ReincarnationWaveAttack || wave.Status != ReincarnationWaveActive {
			return nil, ErrInvalidReincarnation
		}
		if now.After(run.ExpiresAt) {
			expireReincarnationRun(run, wave, now)
			return nil, nil
		}
		battleGeneralIDs, err := normalizeBattleGeneralIDs(state, generalIDs)
		if err != nil {
			return nil, err
		}
		playerUnits, err := validateAndConsumeArmyWithModifiers(state, troops, modifierSourcesForBattleGenerals(state, battleGeneralIDs)...)
		if err != nil {
			return nil, err
		}
		applyReincarnationBonus(playerUnits, wave.AllyBonus)
		enemyUnits := buildReincarnationEnemyUnits(wave.EnemyFaction, wave.EnemyRemaining, now, wave.EnemyBonus)
		activeTraits := buildActiveTraitsForGeneralIDs(state, battleGeneralIDs)
		resolution := resolveReincarnationTraitCombat(
			combat.Army{Faction: state.Player.Faction, Units: playerUnits},
			combat.Army{Faction: wave.EnemyFaction, Units: enemyUnits},
			activeTraits, "attacker", wave.WaveType,
		)
		result := resolution.Result
		capturedToArmy, routedToGarrison := splitCapturedUnitsByOwnerFaction(state.Player.Faction, resolution.Before.CapturedToArmy)
		capturedToGarrison := mergeTroopMaps(routedToGarrison, resolution.Before.CapturedToGarrison)
		capturedEnemyUnits := mergeTroopMaps(capturedToArmy, capturedToGarrison)
		routeNpcCaptureTraitOutcomes(resolution.Before.Triggered, capturedToArmy, capturedToGarrison)
		for unitType, amount := range capturedToArmy {
			mergeIntoArmy(state, unitType, amount)
		}
		playerLosses, enemyLosses := lossMaps(result.AttackerLosses), lossMaps(result.DefenderLosses)
		refundSurvivors(state, playerUnits, playerLosses)
		subtractTroops(wave.EnemyRemaining, capturedEnemyUnits)
		subtractTroops(wave.EnemyRemaining, enemyLosses)
		passed := totalTroops(wave.EnemyRemaining) <= 0
		if passed {
			clearReincarnationWave(run, wave, now)
		}
		afterBattleCtx := applyReincarnationAfterBattleTraits(state, activeTraits, playerLosses, true, result.Winner, wave.WaveType)
		generalSnapshots := buildPvpGeneralSnapshots(state, battleGeneralIDs)
		report = buildReincarnationReport(*run, *wave, state, troops, playerLosses, enemyLosses, result, ReportViewAttack, passed, now)
		report.CapturedUnits = cloneStringIntMap(capturedToArmy)
		report.CapturedToGarrison = cloneStringIntMap(capturedToGarrison)
		report.DefenderUnits = mergeTroopMaps(report.DefenderUnits, capturedEnemyUnits)
		report.RevivedUnits = cloneStringIntMap(afterBattleCtx.Revived)
		report.SurvivedUnits = calculateReportSurvivedUnits(troops, playerLosses, afterBattleCtx.Revived, capturedToArmy)
		mergeTraitOutcomes(&report, resolution.Before.Triggered)
		mergeTraitOutcomes(&report, resolution.AfterCombat.Triggered)
		mergeTraitOutcomes(&report, afterBattleCtx.Triggered)
		expResult := applyGeneralBattleExpToRoster(state, battleGeneralIDs, calculateGeneralBattleExpFromLosses(wave.EnemyFaction, result.DefenderLosses))
		if expResult.Gained > 0 {
			report.GeneralExpGained = expResult.Gained
			report.GeneralLevelBefore = expResult.LevelBefore
			report.GeneralLevelAfter = expResult.LevelAfter
		}
		report.PvpAttackerGenerals = generalSnapshots
		report.Detail = nil
		report = NormalizeBattleReport(report)
		run.Battles = append(run.Battles, buildReincarnationBattle(*run, *wave, troops, playerLosses, enemyLosses, passed, report, clientActionID, now))
		return []BattleReport{report}, nil
	})
	if err != nil {
		return ReincarnationActionResult{}, err
	}
	if len(reports) > 0 {
		report = NormalizeBattleReport(reports[0])
	} else if duplicateReportID != "" {
		if existingReport, getErr := s.GetReportForPlayer(playerID, duplicateReportID); getErr == nil {
			report = existingReport
		}
	}
	if shouldSettleReincarnation(run) {
		if settled, grantResult, settleErr := s.grantReincarnationRewards(run, now); settleErr == nil {
			run = settled
			if grantResult != nil {
				state = grantResult.State
			}
		}
	}
	hydrateStateForResponse(&state, now)
	return buildReincarnationActionResult(run, &report, state, now), nil
}

// ReadyReincarnationDefense 结算防守波。
func (s *Service) ReadyReincarnationDefense(playerID string, waveID string, troops map[string]int, generalIDs []string, clientActionID string) (ReincarnationActionResult, error) {
	now := time.Now().UTC()
	var report BattleReport
	duplicateReportID := ""
	state, run, reports, err := s.repo.UpdateReincarnationRunWithState(playerID, runIDFromWaveID(waveID), now, func(state *GameState, run *ReincarnationRun) ([]BattleReport, error) {
		EnsureGeneralRoster(state, now)
		if existing := findReincarnationBattleByAction(*run, waveID, clientActionID); existing != nil {
			duplicateReportID = existing.ReportID
			return nil, nil
		}
		wave := activeReincarnationWave(run)
		if wave == nil || wave.ID != waveID || wave.WaveType != ReincarnationWaveDefense || wave.Status != ReincarnationWaveActive {
			return nil, ErrInvalidReincarnation
		}
		if now.After(run.ExpiresAt) {
			expireReincarnationRun(run, wave, now)
			return nil, nil
		}
		if err := validateArmyAvailability(state, troops); err != nil {
			return nil, err
		}
		battleGeneralIDs, err := normalizeBattleGeneralIDs(state, generalIDs)
		if err != nil {
			return nil, err
		}
		defenderUnits, err := buildSimulatedCombatUnits(state.Player.Faction, troops, now, modifierSourcesForBattleGenerals(state, battleGeneralIDs)...)
		if err != nil {
			return nil, err
		}
		applyReincarnationBonus(defenderUnits, wave.AllyBonus)
		enemyUnits := buildReincarnationEnemyUnits(wave.EnemyFaction, wave.EnemyRemaining, now, wave.EnemyBonus)
		activeTraits := buildActiveTraitsForGeneralIDs(state, battleGeneralIDs)
		resolution := resolveReincarnationTraitCombat(
			combat.Army{Faction: wave.EnemyFaction, Units: enemyUnits},
			combat.Army{Faction: state.Player.Faction, Units: defenderUnits},
			activeTraits, "defender", wave.WaveType,
		)
		result := resolution.Result
		playerLosses, enemyLosses := lossMaps(result.DefenderLosses), lossMaps(result.AttackerLosses)
		if err := deductArmyLosses(state, playerLosses); err != nil {
			return nil, err
		}
		subtractTroops(wave.EnemyRemaining, enemyLosses)
		passed := result.Winner != "attacker"
		readyAt := now
		resolveAt := now.Add(time.Duration(GetReincarnationConfig().DefenseCountdownSeconds) * time.Second)
		wave.DefenseReadyAt = &readyAt
		wave.DefenseResolveAt = &resolveAt
		if passed {
			clearReincarnationWave(run, wave, now)
		} else {
			failReincarnationRun(run, wave, "defense_failed", now)
		}
		afterBattleCtx := applyReincarnationAfterBattleTraits(state, activeTraits, playerLosses, false, result.Winner, wave.WaveType)
		generalSnapshots := buildPvpGeneralSnapshots(state, battleGeneralIDs)
		report = buildReincarnationReport(*run, *wave, state, troops, playerLosses, enemyLosses, result, ReportViewDefense, passed, now)
		report.RevivedUnits = cloneStringIntMap(afterBattleCtx.Revived)
		report.SurvivedUnits = calculateReportSurvivedUnits(troops, playerLosses, afterBattleCtx.Revived, nil)
		mergeTraitOutcomes(&report, resolution.Before.Triggered)
		mergeTraitOutcomes(&report, resolution.AfterCombat.Triggered)
		mergeTraitOutcomes(&report, afterBattleCtx.Triggered)
		expResult := applyGeneralBattleExpToRoster(state, battleGeneralIDs, calculateGeneralBattleExpFromLosses(wave.EnemyFaction, result.AttackerLosses))
		if expResult.Gained > 0 {
			report.GeneralExpGained = expResult.Gained
			report.GeneralLevelBefore = expResult.LevelBefore
			report.GeneralLevelAfter = expResult.LevelAfter
		}
		report.PvpDefenderGenerals = generalSnapshots
		report.Detail = nil
		report = NormalizeBattleReport(report)
		run.Battles = append(run.Battles, buildReincarnationBattle(*run, *wave, troops, playerLosses, enemyLosses, passed, report, clientActionID, now))
		return []BattleReport{report}, nil
	})
	if err != nil {
		return ReincarnationActionResult{}, err
	}
	if len(reports) > 0 {
		report = NormalizeBattleReport(reports[0])
	} else if duplicateReportID != "" {
		if existingReport, getErr := s.GetReportForPlayer(playerID, duplicateReportID); getErr == nil {
			report = existingReport
		}
	}
	if shouldSettleReincarnation(run) {
		if settled, grantResult, settleErr := s.grantReincarnationRewards(run, now); settleErr == nil {
			run = settled
			if grantResult != nil {
				state = grantResult.State
			}
		}
	}
	hydrateStateForResponse(&state, now)
	return buildReincarnationActionResult(run, &report, state, now), nil
}

// SettleReincarnationRun 主动结算已结束或超时的轮回实例。
func (s *Service) SettleReincarnationRun(playerID string) (ReincarnationActionResult, error) {
	now := time.Now().UTC()
	active, found, err := s.repo.GetActiveReincarnationRun(playerID, now)
	if err != nil {
		return ReincarnationActionResult{}, err
	}
	if !found {
		return ReincarnationActionResult{}, ErrReincarnationRunNotFound
	}
	state, run, _, err := s.repo.UpdateReincarnationRunWithState(playerID, active.ID, now, func(state *GameState, run *ReincarnationRun) ([]BattleReport, error) {
		if run.Status == ReincarnationRunRunning && now.After(run.ExpiresAt) {
			wave := activeReincarnationWave(run)
			expireReincarnationRun(run, wave, now)
		}
		return nil, nil
	})
	if err != nil {
		return ReincarnationActionResult{}, err
	}
	if shouldSettleReincarnation(run) {
		grantResult := (*RewardGrantResult)(nil)
		run, grantResult, err = s.grantReincarnationRewards(run, now)
		if err != nil {
			return ReincarnationActionResult{}, err
		}
		if grantResult != nil {
			state = grantResult.State
		}
	}
	hydrateStateForResponse(&state, now)
	return buildReincarnationActionResult(run, nil, state, now), nil
}

// ExitReincarnationRun 退出已经结束的轮回实例，并保留实例、战报和已发放奖励。
func (s *Service) ExitReincarnationRun(playerID string, runID string) (ReincarnationExitResult, error) {
	now := time.Now().UTC()
	run, err := s.repo.GetReincarnationRun(strings.TrimSpace(runID))
	if err != nil || run.PlayerID != playerID {
		return ReincarnationExitResult{}, ErrReincarnationRunNotFound
	}
	if run.Status == ReincarnationRunRunning {
		return ReincarnationExitResult{}, ErrInvalidReincarnation
	}
	if shouldSettleReincarnation(run) {
		run, _, err = s.grantReincarnationRewards(run, now)
		if err != nil {
			return ReincarnationExitResult{}, err
		}
	}
	if run.ExitedAt == nil {
		run.ExitedAt = &now
		run.UpdatedAt = now
		if err := s.repo.SaveReincarnationRun(run); err != nil {
			return ReincarnationExitResult{}, err
		}
	}
	return ReincarnationExitResult{RunID: run.ID, ExitedAt: run.ExitedAt.UTC().Format(resourceDateLayout), ServerTime: now.Format(resourceDateLayout)}, nil
}

// ListReincarnationReports 查询轮回绝境副本战报。
func (s *Service) ListReincarnationReports(playerID string, page int, pageSize int) (BattleReportPage, error) {
	return s.ListReportsByQuery(BattleReportQuery{
		PlayerID:   playerID,
		SourceType: ReportSourceDungeon,
		Page:       page,
		PageSize:   pageSize,
	})
}

// ListReincarnationRunsForAdmin 查询 GM 视角轮回实例。
func (s *Service) ListReincarnationRunsForAdmin(playerID string, limit int, offset int) ([]ReincarnationRun, int, error) {
	return s.repo.ListReincarnationRuns(playerID, limit, offset)
}

// GetReincarnationRunForAdmin 查询 GM 视角单个轮回实例。
func (s *Service) GetReincarnationRunForAdmin(runID string) (ReincarnationRun, error) {
	return s.repo.GetReincarnationRun(runID)
}

// ForceSettleReincarnationRunForAdmin 强制结束异常轮回并结算已累计奖励。
func (s *Service) ForceSettleReincarnationRunForAdmin(runID string) (ReincarnationRun, error) {
	now := time.Now().UTC()
	run, err := s.repo.GetReincarnationRun(runID)
	if err != nil {
		return ReincarnationRun{}, err
	}
	_, run, _, err = s.repo.UpdateReincarnationRunWithState(run.PlayerID, run.ID, now, func(state *GameState, run *ReincarnationRun) ([]BattleReport, error) {
		if run.Status == ReincarnationRunRunning {
			wave := activeReincarnationWave(run)
			if wave != nil {
				wave.Status = ReincarnationWaveExpired
			}
			run.Status = ReincarnationRunExpired
			run.EndedReason = "admin_force_settle"
			run.FailedAt = &now
		}
		return nil, nil
	})
	if err != nil {
		return ReincarnationRun{}, err
	}
	if shouldSettleReincarnation(run) {
		settled, _, err := s.grantReincarnationRewards(run, now)
		return settled, err
	}
	return run, nil
}

// RepairReincarnationRewardForAdmin 幂等修复轮回奖励发放状态。
func (s *Service) RepairReincarnationRewardForAdmin(runID string) (ReincarnationRun, error) {
	now := time.Now().UTC()
	run, err := s.repo.GetReincarnationRun(runID)
	if err != nil {
		return ReincarnationRun{}, err
	}
	if run.Status == ReincarnationRunRunning {
		return ReincarnationRun{}, ErrInvalidReincarnation
	}
	settled, _, err := s.grantReincarnationRewards(run, now)
	return settled, err
}

func (s *Service) buildReincarnationRun(state GameState, levelCfg ReincarnationLevelConfig, now time.Time) ReincarnationRun {
	runID := "ra_" + randomID(8)
	rng := mathrand.New(mathrand.NewSource(now.UnixNano()))
	run := ReincarnationRun{
		ID:             runID,
		PlayerID:       state.Player.ID,
		Level:          levelCfg.Level,
		LevelName:      levelCfg.Name,
		Status:         ReincarnationRunRunning,
		CurrentWave:    1,
		StartedAt:      now,
		ExpiresAt:      now.Add(time.Duration(levelCfg.DurationSeconds) * time.Second),
		PendingRewards: []Reward{},
		Waves:          make([]ReincarnationWave, 0, ReincarnationWaveCount),
		Battles:        []ReincarnationBattle{},
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	for i := 1; i <= ReincarnationWaveCount; i++ {
		waveType := ReincarnationWaveAttack
		if i%2 == 0 {
			waveType = ReincarnationWaveDefense
		}
		status := ReincarnationWaveLocked
		if i == 1 {
			status = ReincarnationWaveActive
		}
		enemyFaction := pickReincarnationEnemyFaction(rng)
		enemyTroops := generateReincarnationEnemyTroops(levelCfg, enemyFaction, i, rng)
		run.Waves = append(run.Waves, ReincarnationWave{
			ID:             fmt.Sprintf("%s_w%02d", runID, i),
			RunID:          runID,
			WaveIndex:      i,
			WaveType:       waveType,
			EnemyFaction:   enemyFaction,
			EnemyTroops:    enemyTroops,
			EnemyRemaining: cloneStringIntMap(enemyTroops),
			AllyBonus:      buildReincarnationBonus("ally", state.Player.Faction, waveType, rng),
			EnemyBonus:     buildReincarnationBonus("enemy", enemyFaction, waveType, rng),
			RewardPreview:  buildReincarnationWaveRewards(levelCfg, i),
			TroopCap:       int(levelCfg.PlayerTroopCap),
			Status:         status,
			StartedAt:      now,
		})
	}
	return run
}

func activeReincarnationWave(run *ReincarnationRun) *ReincarnationWave {
	for i := range run.Waves {
		if run.Waves[i].WaveIndex == run.CurrentWave {
			return &run.Waves[i]
		}
	}
	return nil
}

func clearReincarnationWave(run *ReincarnationRun, wave *ReincarnationWave, now time.Time) {
	wave.Status = ReincarnationWaveCleared
	wave.ClearedAt = &now
	wave.RewardResult = append([]Reward(nil), wave.RewardPreview...)
	run.PendingRewards = mergeRewards(append(run.PendingRewards, wave.RewardResult...))
	if wave.WaveIndex >= ReincarnationWaveCount {
		run.Status = ReincarnationRunCompleted
		run.EndedReason = "completed"
		run.CompletedAt = &now
		return
	}
	run.CurrentWave = wave.WaveIndex + 1
	for i := range run.Waves {
		if run.Waves[i].WaveIndex == run.CurrentWave {
			run.Waves[i].Status = ReincarnationWaveActive
			run.Waves[i].StartedAt = now
			break
		}
	}
}

func failReincarnationRun(run *ReincarnationRun, wave *ReincarnationWave, reason string, now time.Time) {
	wave.Status = ReincarnationWaveFailed
	run.Status = ReincarnationRunFailed
	run.EndedReason = reason
	run.FailedAt = &now
}

func expireReincarnationRun(run *ReincarnationRun, wave *ReincarnationWave, now time.Time) {
	if wave != nil && wave.Status == ReincarnationWaveActive {
		wave.Status = ReincarnationWaveExpired
	}
	run.Status = ReincarnationRunExpired
	run.EndedReason = "expired"
	run.FailedAt = &now
}

func shouldSettleReincarnation(run ReincarnationRun) bool {
	return run.RewardGrantedAt == nil && run.Status != ReincarnationRunRunning
}

func (s *Service) grantReincarnationRewards(run ReincarnationRun, now time.Time) (ReincarnationRun, *RewardGrantResult, error) {
	if run.RewardGrantedAt != nil {
		return run, nil, nil
	}
	var grantResult *RewardGrantResult
	if len(run.PendingRewards) > 0 {
		result, err := s.GrantRewards(run.PlayerID, mergeRewards(run.PendingRewards), RewardGrantContext{
			Reason:  "reincarnation_abyss",
			RefType: "dungeon",
			RefID:   run.ID,
		})
		if err != nil {
			return run, nil, err
		}
		grantResult = &result
	}
	run.Status = ReincarnationRunRewarded
	run.RewardGrantedAt = &now
	run.UpdatedAt = now
	if err := s.repo.SaveReincarnationRun(run); err != nil {
		return run, nil, err
	}
	return run, grantResult, nil
}

func buildReincarnationActionResult(run ReincarnationRun, report *BattleReport, state GameState, now time.Time) ReincarnationActionResult {
	return ReincarnationActionResult{
		Run:            run,
		BattleReport:   report,
		Army:           state.Army,
		Inventory:      state.Inventory,
		InventorySlots: state.InventorySlots,
		General:        state.General,
		Generals:       state.Generals,
		ServerTime:     now.Format(resourceDateLayout),
	}
}

// resolveReincarnationTraitCombat 通过标准事件管线结算轮回攻防波的战前和战斗结算后特性。
func resolveReincarnationTraitCombat(attacker combat.Army, defender combat.Army, activeTraits []general.ActiveTrait, playerSide string, scene string) reincarnationCombatResolution {
	playerIsAttacker := playerSide == "attacker"
	beforeCtx := &general.BeforeBattleContext{
		Attacker:          &attacker,
		Defender:          &defender,
		AttackerOwnsTrait: playerIsAttacker,
		DefenderOwnsTrait: !playerIsAttacker,
		IsPvP:             false,
		SameFaction:       attacker.Faction == defender.Faction,
		Scene:             scene,
	}
	ownedTraits := withTraitOwnerSide(activeTraits, playerSide)
	general.Dispatch(beforeCtx, ownedTraits)
	result := combat.Resolve(combat.CombatInput{
		RuleID:   activeCombatRuleID(combat.ScenePVEAttack),
		Attacker: attacker,
		Defender: defender,
	})
	applyPreBattleLossesToCombatResult(&result, beforeCtx)
	afterCombatCtx := &general.AfterCombatResolveContext{
		Result:            &result,
		Attacker:          &attacker,
		Defender:          &defender,
		AttackerOwnsTrait: playerIsAttacker,
		DefenderOwnsTrait: !playerIsAttacker,
		IsAttackerOnly:    playerIsAttacker,
		Scene:             scene,
	}
	general.Dispatch(afterCombatCtx, ownedTraits)
	return reincarnationCombatResolution{Result: result, Before: beforeCtx, AfterCombat: afterCombatCtx}
}

// applyReincarnationAfterBattleTraits 把轮回战后复活或返兵写回玩家权威兵力。
func applyReincarnationAfterBattleTraits(state *GameState, activeTraits []general.ActiveTrait, losses map[string]int, isAttacker bool, winner string, scene string) *general.AfterBattleContext {
	playerArmy := armySliceToMap(state.Army)
	winner = strings.ToLower(strings.TrimSpace(winner))
	won := winner == "attacker"
	if !isAttacker {
		won = winner == "defender"
	}
	ctx := &general.AfterBattleContext{
		PlayerArmy:   playerArmy,
		PlayerLosses: losses,
		IsAttacker:   isAttacker,
		Won:          won,
		Winner:       winner,
		Scene:        scene,
	}
	ownerSide := "defender"
	if isAttacker {
		ownerSide = "attacker"
	}
	general.Dispatch(ctx, withTraitOwnerSide(activeTraits, ownerSide))
	if len(ctx.Revived) > 0 {
		state.Army = armyMapToSlice(playerArmy)
	}
	return ctx
}

func pickReincarnationEnemyFaction(rng *mathrand.Rand) string {
	cfg := GetReincarnationConfig()
	if len(cfg.EnemyFactions) == 0 {
		return "wei"
	}
	return cfg.EnemyFactions[rng.Intn(len(cfg.EnemyFactions))]
}

func generateReincarnationEnemyTroops(levelCfg ReincarnationLevelConfig, faction string, waveIndex int, rng *mathrand.Rand) map[string]int {
	units := combatUnitIDsForFaction(faction)
	if len(units) == 0 {
		return map[string]int{}
	}
	scale := 0.82 + rng.Float64()*0.32 + math.Min(0.18, float64(waveIndex-1)*0.01)
	total := int(math.Max(1, float64(levelCfg.EnemyTroopBase)*scale))
	result := map[string]int{}
	count := 1 + rng.Intn(minInt(4, len(units)))
	selected := rng.Perm(len(units))[:count]
	weights := make([]int, 0, count)
	weightTotal := 0
	for range selected {
		weight := 1 + rng.Intn(100)
		weights = append(weights, weight)
		weightTotal += weight
	}
	assigned := 0
	for i, unitIndex := range selected {
		unitType := units[unitIndex]
		share := total * weights[i] / weightTotal
		if i == len(selected)-1 {
			share = total - assigned
		}
		if share > 0 {
			result[unitType] = share
			assigned += share
		}
	}
	return result
}

func combatUnitIDsForFaction(faction string) []string {
	units := GetUnitsConfig()[faction]
	ids := []string{}
	for id, cfg := range units {
		if !isNonCombatUnit(cfg) {
			ids = append(ids, id)
		}
	}
	sort.Strings(ids)
	if len(ids) > 4 {
		return ids[:4]
	}
	return ids
}

func buildReincarnationBonus(side string, faction string, waveType string, rng *mathrand.Rand) ReincarnationBonus {
	stat := "attack"
	if (side == "ally" && waveType == ReincarnationWaveDefense) || (side == "enemy" && waveType == ReincarnationWaveAttack) {
		stat = "defense"
	}
	cfg := GetReincarnationConfig()
	value := 0.3
	if len(cfg.BonusValues) > 0 {
		value = cfg.BonusValues[rng.Intn(len(cfg.BonusValues))]
	}
	if strings.TrimSpace(faction) == "" {
		faction = "wei"
	}
	units := combatUnitIDsForFaction(faction)
	unitType := ""
	unitName := "全军"
	if len(units) > 0 {
		unitType = units[rng.Intn(len(units))]
		if unitCfg, ok := GetUnitConfig(faction, unitType); ok {
			unitName = unitCfg.Name
		}
	}
	labelStat := "攻击"
	if stat == "defense" {
		labelStat = "防御"
	}
	return ReincarnationBonus{
		Side:     side,
		UnitType: unitType,
		Stat:     stat,
		Value:    value,
		Label:    fmt.Sprintf("%s%s +%d%%", unitName, labelStat, int(value*100)),
		UnitName: unitName,
		Faction:  faction,
	}
}

func buildReincarnationWaveRewards(levelCfg ReincarnationLevelConfig, waveIndex int) []Reward {
	waveCfg := reincarnationWaveConfig(waveIndex)
	rewards := append([]Reward(nil), waveCfg.FixedRewards...)
	if strings.TrimSpace(waveCfg.DropPoolID) != "" {
		if rolled, err := RollDropPoolRewards(waveCfg.DropPoolID); err == nil {
			rewards = append(rewards, rolled...)
		}
	}
	return mergeRewards(rewards)
}

func buildReincarnationEnemyUnits(faction string, troops map[string]int, now time.Time, bonus ReincarnationBonus) []combat.Unit {
	units := []combat.Unit{}
	for unitType, amount := range troops {
		if amount <= 0 {
			continue
		}
		if cfg, ok := GetUnitConfig(faction, unitType); ok {
			units = append(units, buildCombatUnitFromConfig(unitType, amount, cfg, now))
		}
	}
	applyReincarnationBonus(units, bonus)
	return units
}

func applyReincarnationBonus(units []combat.Unit, bonus ReincarnationBonus) {
	for i := range units {
		if bonus.UnitType != "" && units[i].ID != bonus.UnitType {
			continue
		}
		multiplier := 1 + bonus.Value
		if bonus.Stat == "attack" {
			units[i].Attack = int(float64(units[i].Attack) * multiplier)
		} else {
			units[i].InfantryDefense = int(float64(units[i].InfantryDefense) * multiplier)
			units[i].CavalryDefense = int(float64(units[i].CavalryDefense) * multiplier)
		}
	}
}

func lossMaps(losses []combat.UnitLoss) map[string]int {
	result := map[string]int{}
	for _, loss := range losses {
		if loss.Losses > 0 {
			result[loss.ID] += loss.Losses
		}
	}
	return result
}

func refundSurvivors(state *GameState, units []combat.Unit, losses map[string]int) {
	for _, unit := range units {
		if survived := unit.Count - losses[unit.ID]; survived > 0 {
			addToArmy(&state.Army, unit.ID, survived)
		}
	}
}

func deductArmyLosses(state *GameState, losses map[string]int) error {
	army := armySliceToMap(state.Army)
	for unitType, amount := range losses {
		if amount <= 0 {
			continue
		}
		if army[unitType] < amount {
			return ErrInsufficientArmy
		}
		army[unitType] -= amount
	}
	state.Army = armyMapToSlice(army)
	return nil
}

func subtractTroops(troops map[string]int, losses map[string]int) {
	for unitType, amount := range losses {
		troops[unitType] -= amount
		if troops[unitType] < 0 {
			troops[unitType] = 0
		}
	}
}

func buildReincarnationBattle(run ReincarnationRun, wave ReincarnationWave, troops map[string]int, losses map[string]int, enemyLosses map[string]int, passed bool, report BattleReport, clientActionID string, now time.Time) ReincarnationBattle {
	return ReincarnationBattle{
		ID:             "rab_" + randomID(8),
		RunID:          run.ID,
		WaveID:         wave.ID,
		PlayerID:       run.PlayerID,
		ClientActionID: strings.TrimSpace(clientActionID),
		WaveIndex:      wave.WaveIndex,
		WaveType:       wave.WaveType,
		AttackTroops:   cloneStringIntMap(troops),
		Losses:         cloneStringIntMap(losses),
		RevivedUnits:   cloneStringIntMap(report.RevivedUnits),
		SurvivedTroops: cloneStringIntMap(report.SurvivedUnits),
		EnemyLosses:    cloneStringIntMap(enemyLosses),
		EnemyCaptured:  mergeTroopMaps(report.CapturedUnits, report.CapturedToGarrison),
		EnemyRemaining: cloneStringIntMap(wave.EnemyRemaining),
		TraitOutcomes:  cloneTraitOutcomeReports(report.TraitOutcomes),
		Passed:         passed,
		ReportID:       report.ID,
		CreatedAt:      now,
	}
}

// cloneTraitOutcomeReports 复制副本战斗记录中的特性结果，避免和战报对象共享 map。
func cloneTraitOutcomeReports(source map[string]TraitOutcomeReport) map[string]TraitOutcomeReport {
	if len(source) == 0 {
		return nil
	}
	result := make(map[string]TraitOutcomeReport, len(source))
	for key, outcome := range source {
		cloned := outcome
		cloned.Detail = cloneBattleReportDynamicMap(outcome.Detail)
		result[key] = cloned
	}
	return result
}

func findReincarnationBattleByAction(run ReincarnationRun, waveID string, clientActionID string) *ReincarnationBattle {
	clientActionID = strings.TrimSpace(clientActionID)
	if clientActionID == "" {
		return nil
	}
	for i := range run.Battles {
		if run.Battles[i].WaveID == waveID && run.Battles[i].ClientActionID == clientActionID {
			return &run.Battles[i]
		}
	}
	return nil
}

func hasReincarnationBattleForWave(run ReincarnationRun, waveID string) bool {
	for i := range run.Battles {
		if run.Battles[i].WaveID == waveID {
			return true
		}
	}
	return false
}

func buildReincarnationReport(run ReincarnationRun, wave ReincarnationWave, state *GameState, troops map[string]int, losses map[string]int, enemyLosses map[string]int, result combat.CombatResult, viewType string, passed bool, now time.Time) BattleReport {
	reportResult := "draw"
	if result.Winner == "attacker" {
		reportResult = "attacker_victory"
	} else if result.Winner == "defender" {
		reportResult = "defender_victory"
	}
	battleType := "dungeon_reincarnation_attack"
	if wave.WaveType == ReincarnationWaveDefense {
		battleType = "dungeon_reincarnation_defense"
	}
	enemyBefore := addIntMaps(wave.EnemyRemaining, enemyLosses)
	playerPower := int(result.AttackPower)
	enemyPower := int(result.DefensePower)
	if viewType == ReportViewDefense {
		playerPower = int(result.DefensePower)
		enemyPower = int(result.AttackPower)
	}
	report := NormalizeBattleReport(BattleReport{
		ID:                "br_" + randomID(8),
		EventID:           "event_ra_" + randomID(8),
		PlayerID:          state.Player.ID,
		OwnerPlayerID:     state.Player.ID,
		PlayerFaction:     state.Player.Faction,
		PlayerName:        state.Player.Nickname,
		ViewType:          viewType,
		SourceType:        ReportSourceDungeon,
		BattleType:        battleType,
		TargetID:          run.ID,
		TargetName:        fmt.Sprintf("轮回绝境 %d 层 第 %d 波", run.Level, wave.WaveIndex),
		Type:              wave.WaveType,
		Result:            reportResult,
		PlayerPower:       playerPower,
		EnemyPower:        enemyPower,
		DispatchedUnits:   cloneStringIntMap(troops),
		LostUnits:         cloneStringIntMap(losses),
		DefenderFaction:   wave.EnemyFaction,
		DefenderUnits:     enemyBefore,
		DefenderLostUnits: cloneStringIntMap(enemyLosses),
		DefenderRevealed:  true,
		Rewards:           rewardsToLegacyMap(wave.RewardPreview),
		Title:             fmt.Sprintf("轮回绝境 第 %d 波", wave.WaveIndex),
		Summary:           fmt.Sprintf("%s，己方%s，敌方%s", map[bool]string{true: "本波通过", false: "本波未通过"}[passed], wave.AllyBonus.Label, wave.EnemyBonus.Label),
		Read:              false,
		CreatedAt:         now.Format(resourceDateLayout),
	})
	if report.Detail != nil {
		report.Detail.Extra = mergeReportExtraMap(report.Detail.Extra, map[string]interface{}{
			"dungeon": map[string]interface{}{"rewardMode": "preview"},
		})
	}
	return report
}

func addIntMaps(left map[string]int, right map[string]int) map[string]int {
	result := cloneStringIntMap(left)
	for key, value := range right {
		result[key] += value
	}
	return result
}

func rewardsToLegacyMap(rewards []Reward) map[string]int {
	result := map[string]int{}
	for _, reward := range rewards {
		key := reward.ID
		if key == "" {
			key = reward.Type
		}
		result[key] += reward.Amount
	}
	return result
}

func mergeRewards(rewards []Reward) []Reward {
	merged := map[string]Reward{}
	for _, reward := range rewards {
		if reward.Amount <= 0 || reward.Type == "" {
			continue
		}
		key := reward.Type + ":" + reward.ID
		current := merged[key]
		if current.Type == "" {
			current = reward
		} else {
			current.Amount += reward.Amount
		}
		merged[key] = current
	}
	keys := make([]string, 0, len(merged))
	for key := range merged {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	result := make([]Reward, 0, len(keys))
	for _, key := range keys {
		result = append(result, merged[key])
	}
	return result
}

func runIDFromWaveID(waveID string) string {
	if index := strings.LastIndex(waveID, "_w"); index > 0 {
		return waveID[:index]
	}
	return waveID
}

func firstReincarnationError(err error) error {
	if errors.Is(err, ErrReincarnationRunNotFound) {
		return ErrReincarnationRunNotFound
	}
	return err
}
