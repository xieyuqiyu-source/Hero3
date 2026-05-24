package game

import (
	"errors"
	"log/slog"
	"strings"
	"time"

	"hero3/internal/combat"
)

var (
	ErrNpcNotFound      = errors.New("npc city not found")
	ErrNoUnitsSelected  = errors.New("no units selected for dispatch")
	ErrInsufficientArmy = errors.New("insufficient army for dispatch")
)

// AttackNpcRequest 攻击 NPC 请求
type AttackNpcRequest struct {
	PlayerID string         `json:"playerId"`
	NpcID    string         `json:"npcId"`
	Mode     string         `json:"mode"` // "attack" or "plunder"
	Units    map[string]int `json:"units"` // unitType → count
}

// AttackNpcResponse 攻击 NPC 响应
type AttackNpcResponse struct {
	BattleReport BattleReport `json:"battleReport"`
	State        GameState    `json:"state"`
}

// ScoutNpcRequest 侦查 NPC 请求
type ScoutNpcRequest struct {
	PlayerID string `json:"playerId"`
	NpcID    string `json:"npcId"`
}

// ScoutNpcResponse 侦查 NPC 响应
type ScoutNpcResponse struct {
	Success      bool          `json:"success"`
	BattleReport BattleReport  `json:"battleReport"`
	NpcCity      *NpcCity      `json:"npcCity"`
	State        GameState     `json:"state"`
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

	state, err := s.repo.GetState(playerID)
	if err != nil {
		return AttackNpcResponse{}, err
	}

	now := time.Now()
	state, _ = settleResources(state, now)

	// 确保 NPC 状态存在
	if state.NpcState == nil || len(state.NpcState.Cities) == 0 {
		return AttackNpcResponse{}, ErrNpcNotFound
	}

	// 结算 NPC 资源和守军
	settleNpcCities(state.NpcState, now)

	// 找到目标 NPC
	npcIdx := -1
	for i, city := range state.NpcState.Cities {
		if city.ID == npcID {
			npcIdx = i
			break
		}
	}
	if npcIdx == -1 {
		return AttackNpcResponse{}, ErrNpcNotFound
	}

	npc := &state.NpcState.Cities[npcIdx]

	// 校验并扣除出征兵力
	attackerUnits, err := validateAndConsumeArmy(&state, req.Units)
	if err != nil {
		return AttackNpcResponse{}, err
	}

	// 构建战斗输入
	ruleID := "official_attack"
	if mode == "plunder" {
		ruleID = "official_plunder"
	}

	combatInput := combat.CombatInput{
		RuleID:   ruleID,
		Attacker: buildCombatArmy(state.Player.Faction, attackerUnits),
		Defender: buildNpcCombatArmy(npc),
		WallLevel: 0, // NPC 无城墙
	}

	// 执行战斗
	result := combat.Resolve(combatInput)

	// 应用战斗结果
	report := applyNpcBattleResult(&state, npc, result, attackerUnits, mode, now)

	// 保存战报到独立存储
	if err := s.repo.SaveReport(report); err != nil { slog.Warn("battle report save failed", "error", err, "reportId", report.ID) }

	// 保存玩家状态
	state.ServerTime = now.UTC().Format(resourceDateLayout)
	if err := s.repo.SaveState(state, now); err != nil {
		return AttackNpcResponse{}, err
	}

	// 加载战报列表用于返回
	reports, listErr := s.repo.ListReports(state.Player.ID, 50)
if listErr != nil { slog.Warn("list reports failed", "error", listErr) }
	state.RecentBattleReports = reports

	return AttackNpcResponse{
		BattleReport: report,
		State:        state,
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

	state, err := s.repo.GetState(playerID)
	if err != nil {
		return ScoutNpcResponse{}, err
	}

	now := time.Now()
	state, _ = settleResources(state, now)

	if state.NpcState == nil || len(state.NpcState.Cities) == 0 {
		return ScoutNpcResponse{}, ErrNpcNotFound
	}

	settleNpcCities(state.NpcState, now)

	// 找到目标 NPC
	var targetNpc *NpcCity
	var npcIdx int
	for i, city := range state.NpcState.Cities {
		if city.ID == npcID {
			targetNpc = &state.NpcState.Cities[i]
			npcIdx = i
			break
		}
	}
	if targetNpc == nil {
		return ScoutNpcResponse{}, ErrNpcNotFound
	}

	// 找到玩家侦察兵
	scoutUnitID := findScoutUnit(state.Player.Faction)
	if scoutUnitID == "" {
		return ScoutNpcResponse{}, ErrNoUnitsSelected
	}

	// 获取玩家侦察兵数量
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
		return ScoutNpcResponse{}, ErrInsufficientArmy
	}

	// 获取 NPC 侦察兵数量
	npcScoutUnitID := findScoutUnit(targetNpc.Faction)
	npcScoutCount := 0
	for _, u := range targetNpc.Army {
		if u.UnitType == npcScoutUnitID {
			npcScoutCount = u.Amount
			break
		}
	}

	// 侦察兵对决：比数量
	nowStr := now.UTC().Format(resourceDateLayout)
	playerLost := 0
	npcLost := 0
	scoutSuccess := false

	if npcScoutCount <= 0 {
		// NPC 无侦察兵，直接成功，玩家无损耗
		scoutSuccess = true
	} else if playerScoutCount > npcScoutCount {
		// 玩家多 → 成功，玩家损失 = NPC 数量，NPC 全灭
		playerLost = npcScoutCount
		npcLost = npcScoutCount
		scoutSuccess = true
	} else if playerScoutCount == npcScoutCount {
		// 相等 → 双方全灭，侦查失败
		playerLost = playerScoutCount
		npcLost = npcScoutCount
		scoutSuccess = false
	} else {
		// NPC 多 → 失败，玩家全灭，NPC 损失 = 玩家数量
		playerLost = playerScoutCount
		npcLost = playerScoutCount
		scoutSuccess = false
	}

	// 扣减玩家侦察兵
	if playerLost > 0 && playerScoutIdx >= 0 {
		state.Army[playerScoutIdx].Amount -= playerLost
		if state.Army[playerScoutIdx].Amount <= 0 {
			state.Army = append(state.Army[:playerScoutIdx], state.Army[playerScoutIdx+1:]...)
		}
	}

	// 扣减 NPC 侦察兵
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

	// 生成侦查战报
	reportResult := "attacker_victory"
	if !scoutSuccess {
		reportResult = "defender_victory"
	}

	lostUnits := map[string]int{}
	if playerLost > 0 {
		lostUnits[scoutUnitID] = playerLost
	}
	dispatchedUnits := map[string]int{scoutUnitID: playerScoutCount}

	report := BattleReport{
		ID:              "br_" + randomID(8),
		PlayerID:        state.Player.ID,
		TargetID:        targetNpc.ID,
		TargetName:      targetNpc.Name + "（NPC）",
		Type:            "scout",
		Result:          reportResult,
		PlayerPower:     playerScoutCount,
		EnemyPower:      npcScoutCount,
		DispatchedUnits: dispatchedUnits,
		LostUnits:       lostUnits,
		DefenderFaction: targetNpc.Faction,
		DefenderRevealed: scoutSuccess,
		Rewards:         map[string]int{},
		Read:            false,
		CreatedAt:       nowStr,
	}

	// 如果侦查成功，记录防守方信息
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
	} else {
		report.DefenderUnits = map[string]int{}
		report.DefenderLostUnits = map[string]int{}
		report.DefenderResources = map[string]int{}
	}

	// 保存战报
	if err := s.repo.SaveReport(report); err != nil { slog.Warn("battle report save failed", "error", err, "reportId", report.ID) }

	// 保存状态
	state.ServerTime = nowStr
	if err := s.repo.SaveState(state, now); err != nil {
		return ScoutNpcResponse{}, err
	}

	// 加载战报列表
	reports, listErr := s.repo.ListReports(state.Player.ID, 50)
if listErr != nil { slog.Warn("list reports failed", "error", listErr) }
	state.RecentBattleReports = reports

	// 返回结果
	response := ScoutNpcResponse{
		Success:      scoutSuccess,
		BattleReport: report,
		State:        state,
	}
	if scoutSuccess {
		response.NpcCity = targetNpc
	}

	return response, nil
}

// --- 内部函数 ---

func validateAndConsumeArmy(state *GameState, units map[string]int) ([]combat.Unit, error) {
	if len(units) == 0 {
		return nil, ErrNoUnitsSelected
	}

	faction := state.Player.Faction
	var combatUnits []combat.Unit

	for unitType, count := range units {
		if count <= 0 {
			continue
		}

		// 检查玩家是否有足够兵力
		found := false
		for i, armyUnit := range state.Army {
			if armyUnit.UnitType == unitType {
				if armyUnit.Amount < count {
					return nil, ErrInsufficientArmy
				}
				state.Army[i].Amount -= count
				found = true
				break
			}
		}
		if !found {
			return nil, ErrInsufficientArmy
		}

		// 获取兵种配置
		unitCfg, exists := GetUnitConfig(faction, unitType)
		if !exists {
			return nil, ErrUnitNotFound
		}

		combatUnits = append(combatUnits, combat.Unit{
			ID:              unitType,
			Category:        unitCfg.Category,
			Count:           count,
			Attack:          unitCfg.Stats["attack"],
			InfantryDefense: unitCfg.Stats["infantryDefense"],
			CavalryDefense:  unitCfg.Stats["cavalryDefense"],
			CarryCapacity:   unitCfg.Stats["carryCapacity"],
		})
	}

	if len(combatUnits) == 0 {
		return nil, ErrNoUnitsSelected
	}

	// 清理 0 数量的兵种
	cleanArmy := state.Army[:0]
	for _, u := range state.Army {
		if u.Amount > 0 {
			cleanArmy = append(cleanArmy, u)
		}
	}
	state.Army = cleanArmy

	return combatUnits, nil
}

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

		// 资源入库玩家
		for resType, amount := range plundered {
			state.Resources.Items[resType] += amount
			cap := state.Resources.Capacity[resType]
			if cap > 0 && state.Resources.Items[resType] > cap {
				state.Resources.Items[resType] = cap
			}
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
		Read:              false,
		CreatedAt:         nowStr,
	}

	// 战报已通过 repo.SaveReport 独立存储，不再内嵌到 state
	return report
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
		key       string
		amount    int
		fraction  float64
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

func addToArmy(army *[]ArmyUnit, unitType string, amount int) {
	for i := range *army {
		if (*army)[i].UnitType == unitType {
			(*army)[i].Amount += amount
			return
		}
	}
	*army = append(*army, ArmyUnit{UnitType: unitType, Amount: amount})
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

// MarkReportsRead 标记所有战报为已读
func (s *Service) MarkReportsRead(playerID string) (GameState, error) {
	playerID = strings.TrimSpace(playerID)
	if playerID == "" {
		return GameState{}, ErrPlayerNotFound
	}

	if err := s.repo.MarkReportsRead(playerID); err != nil {
		return GameState{}, err
	}

	state, err := s.repo.GetState(playerID)
	if err != nil {
		return GameState{}, err
	}

	reports, listErr := s.repo.ListReports(playerID, 50)
if listErr != nil { slog.Warn("list reports failed", "error", listErr) }
	state.RecentBattleReports = reports
	return state, nil
}

// MarkSingleReportRead 标记单条战报为已读
func (s *Service) MarkSingleReportRead(playerID string, reportID string) (GameState, error) {
	playerID = strings.TrimSpace(playerID)
	reportID = strings.TrimSpace(reportID)
	if playerID == "" {
		return GameState{}, ErrPlayerNotFound
	}

	if err := s.repo.MarkSingleReportRead(playerID, reportID); err != nil {
		return GameState{}, err
	}

	state, err := s.repo.GetState(playerID)
	if err != nil {
		return GameState{}, err
	}

	reports, listErr := s.repo.ListReports(playerID, 50)
if listErr != nil { slog.Warn("list reports failed", "error", listErr) }
	state.RecentBattleReports = reports
	return state, nil
}

// DeleteReport 删除单条战报
func (s *Service) DeleteReport(playerID string, reportID string) (GameState, error) {
	playerID = strings.TrimSpace(playerID)
	reportID = strings.TrimSpace(reportID)
	if playerID == "" {
		return GameState{}, ErrPlayerNotFound
	}
	if reportID == "" {
		return GameState{}, errors.New("reportId is required")
	}

	if err := s.repo.DeleteReport(playerID, reportID); err != nil {
		return GameState{}, err
	}

	state, err := s.repo.GetState(playerID)
	if err != nil {
		return GameState{}, err
	}

	reports, listErr := s.repo.ListReports(playerID, 50)
if listErr != nil { slog.Warn("list reports failed", "error", listErr) }
	state.RecentBattleReports = reports
	return state, nil
}

// DeleteAllReports 一键删除所有战报
func (s *Service) DeleteAllReports(playerID string) (GameState, error) {
	playerID = strings.TrimSpace(playerID)
	if playerID == "" {
		return GameState{}, ErrPlayerNotFound
	}

	if err := s.repo.DeleteAllReports(playerID); err != nil {
		return GameState{}, err
	}

	state, err := s.repo.GetState(playerID)
	if err != nil {
		return GameState{}, err
	}

	reports, listErr := s.repo.ListReports(playerID, 50)
if listErr != nil { slog.Warn("list reports failed", "error", listErr) }
	state.RecentBattleReports = reports
	return state, nil
}
