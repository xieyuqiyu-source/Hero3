// 本文件实现 NPC 战斗、侦查和战斗模拟服务。
package game

import (
	"errors"
	"log/slog"
	"strings"
	"time"

	"hero3/internal/core/combat"
	"hero3/internal/core/general"
)

var (
	ErrNpcNotFound      = errors.New("npc city not found")
	ErrNoUnitsSelected  = errors.New("no units selected for dispatch")
	ErrInsufficientArmy = errors.New("insufficient army for dispatch")
)

// AttackNpcRequest 攻击 NPC 请求
type AttackNpcRequest struct {
	PlayerID   string         `json:"playerId"`
	NpcID      string         `json:"npcId"`
	Mode       string         `json:"mode"`  // "attack" or "plunder"
	Units      map[string]int `json:"units"` // unitType → count
	GeneralIDs []string       `json:"generalIds,omitempty"`
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
	state, err := s.repo.UpdateCombatState(playerID, now, func(state *GameState) error {
		if state.General != nil {
			applyHeroConfigToGeneral(state.General)
		}
		EnsureGeneralRoster(state, now)

		nextState, _ := settleResources(*state, now)
		*state = nextState

		if state.NpcState == nil || len(state.NpcState.Cities) == 0 {
			return ErrNpcNotFound
		}

		settleNpcCities(state.NpcState, now)

		npcIdx := -1
		for i, city := range state.NpcState.Cities {
			if city.ID == npcID {
				npcIdx = i
				break
			}
		}
		if npcIdx == -1 {
			return ErrNpcNotFound
		}

		npc := &state.NpcState.Cities[npcIdx]
		generalIDs, err := normalizeBattleGeneralIDs(state, req.GeneralIDs)
		if err != nil {
			return err
		}
		attackerUnits, err := validateAndConsumeArmyWithModifiers(state, req.Units, modifierSourcesForBattleGenerals(state, generalIDs)...)
		if err != nil {
			return err
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
		capturedToGarrison = mergeTroopMaps(routedToGarrison, beforeCtx.CapturedToGarrison)
		capturedSourceFaction = npc.Faction
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

		report = applyNpcBattleResult(state, npc, result, attackerUnits, mode, now)
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
		dropRewards, dropSnapshots, err := rollNpcBattleDrops(npc, report)
		if err != nil {
			return err
		}
		if len(dropRewards) > 0 {
			apply, err := ApplyRewardsToStateWithContext(state, dropRewards, RewardGrantContext{
				PlayerID: state.Player.ID,
				RefType:  LedgerRefBattleReward,
				RefID:    report.ID,
				Reason:   "npc_battle_drop",
			}, now)
			if err != nil {
				return err
			}
			mergeRewardApplyResult(&rewardApply, apply)
			report.Drops = dropSnapshots
		}
		report.PvpAttackerGenerals = buildPvpGeneralSnapshots(state, generalIDs)
		report.GrantedRewards = buildBattleGrantedRewards(report)

		if report.OverflowCityGold > 0 {
			state.CityGold += FlexInt(report.OverflowCityGold)
		}
		state.ServerTime = now.UTC().Format(resourceDateLayout)
		return nil
	})
	if err != nil {
		return AttackNpcResponse{}, err
	}
	s.flushRewardSideEffects(rewardApply)

	if len(capturedToGarrison) > 0 {
		result, err := s.CreateGarrisonDetachment(CreateGarrisonDetachmentRequest{
			OwnerPlayerID: state.Player.ID,
			HostPlayerID:  state.Player.ID,
			SourceType:    GarrisonSourceCaptured,
			SourceID:      report.ID,
			SourceFaction: capturedSourceFaction,
			Troops:        capturedToGarrison,
			Metadata: map[string]any{
				"reason":   "beauty_trap_capture",
				"reportId": report.ID,
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
		slog.Warn("battle report save failed", "error", err, "reportId", report.ID)
	} else if len(createResult.Reports) > 0 {
		report = createResult.Reports[0]
	}
	s.publishBattleRewardEvents(state.Player.ID, report)
	s.publishBattleFinished(state.Player.ID, report)

	s.attachReportSummary(&state, state.Player.ID)

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
	state, err := s.repo.UpdateCombatState(playerID, now, func(state *GameState) error {
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
		slog.Warn("battle report save failed", "error", err, "reportId", report.ID)
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

	// 记录防守方兵种（战斗前）
	defenderUnits := map[string]int{}
	for _, u := range npc.Army {
		if u.Amount > 0 {
			defenderUnits[u.UnitType] = u.Amount
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

// GetReportByID 公开获取单条战报（用于分享链接）
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
	return NormalizeBattleReport(report), nil
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
	return NormalizeBattleReport(report), nil
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
		report.CreatedAt = valueOrDefault(report.CreatedAt, occurredAt)
		report = NormalizeBattleReport(report)
		reports = append(reports, report)
	}
	if err := s.repo.SaveReports(reports); err != nil {
		return BattleReportCreateResult{}, err
	}
	event := buildBattleEventFromReports(input, eventID, occurredAt, reports)
	return BattleReportCreateResult{Event: event, Reports: reports}, nil
}

// buildBattleEventFromReports 从标准战报创建输入生成事件快照。
func buildBattleEventFromReports(input BattleReportCreateInput, eventID string, occurredAt string, reports []BattleReport) BattleEvent {
	first := reports[0]
	sourceType := valueOrDefault(input.SourceType, first.SourceType)
	battleType := valueOrDefault(input.BattleType, first.BattleType)
	result := valueOrDefault(input.Result, first.Result)
	sourceID := valueOrDefault(input.SourceID, first.TargetID)
	createdAt := occurredAt
	return BattleEvent{
		ID:                     eventID,
		SourceType:             sourceType,
		SourceID:               sourceID,
		Scene:                  first.ViewType,
		BattleType:             battleType,
		Result:                 result,
		AttackerPlayerID:       first.PlayerID,
		DefenderPlayerID:       first.TargetID,
		AttackerName:           first.PlayerName,
		DefenderName:           first.TargetName,
		AttackerFaction:        first.PlayerFaction,
		DefenderFaction:        first.DefenderFaction,
		RelatedMarchID:         input.RelatedMarchID,
		RelatedReinforcementID: input.RelatedReinforcementID,
		Summary:                input.Extra,
		OccurredAt:             occurredAt,
		CreatedAt:              createdAt,
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

// DeleteReportsByView 删除指定视角 Tab 下的战报。
func (s *Service) DeleteReportsByView(playerID string, viewType string) (ReportActionResult, error) {
	playerID = strings.TrimSpace(playerID)
	viewType = strings.TrimSpace(viewType)
	if playerID == "" {
		return ReportActionResult{}, ErrPlayerNotFound
	}
	if err := s.repo.DeleteReportsByView(playerID, viewType); err != nil {
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
