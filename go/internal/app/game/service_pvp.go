// 本文件实现 PVP 系统应用服务，负责目标、行军、战斗结算和战报。
package game

import (
	"encoding/json"
	"errors"
	"hash/fnv"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"hero3/internal/core/combat"
	"hero3/internal/core/general"
)

var (
	ErrPvpTargetSelf          = errors.New("cannot attack self")
	ErrPvpSameAccountTarget   = errors.New("cannot attack another save in same account")
	ErrPvpTargetProtected     = errors.New("pvp target is protected")
	ErrPvpDailyLimitReached   = errors.New("pvp daily attack limit reached")
	ErrPvpMarchNotReady       = errors.New("pvp march has not arrived")
	ErrPvpMarchNotRecallable  = errors.New("pvp march cannot be recalled")
	ErrPvpMarchNotAccelerable = errors.New("pvp march cannot be accelerated")
	ErrInvalidPvpSeason       = errors.New("invalid pvp season")
)

// ListPvpTargets 返回可展示的玩家目标列表。
func (s *Service) ListPvpTargets(playerID string) (PvpTargetsResponse, error) {
	return s.ListPvpTargetsInArea(playerID, PvpTargetFilter{CenterX: -1, CenterY: -1, Radius: pvpMaxViewRadius(), Limit: 1000})
}

// ListPvpTargetsInArea 返回指定地图视野内的玩家目标列表。
func (s *Service) ListPvpTargetsInArea(playerID string, filter PvpTargetFilter) (PvpTargetsResponse, error) {
	playerID = strings.TrimSpace(playerID)
	if playerID == "" {
		return PvpTargetsResponse{}, ErrPlayerNotFound
	}
	requestAccountID, err := s.repo.GetAccountIDByPlayerID(playerID)
	if err != nil {
		return PvpTargetsResponse{}, err
	}
	accounts, err := s.repo.ListAccounts()
	if err != nil {
		return PvpTargetsResponse{}, err
	}
	items := []PvpTargetSummary{}
	now := time.Now().UTC()
	requestPvpState, _ := s.repo.GetPvpPlayerState(playerID, now)
	selfWorldPosition, err := s.ensureWorldPosition(playerID, "lazy_create", nil)
	if err != nil {
		return PvpTargetsResponse{}, err
	}
	selfPosition := pvpWorldToPvpPosition(selfWorldPosition)
	filter = normalizePvpTargetFilter(filter, selfPosition)
	for _, account := range accounts {
		for _, player := range account.Players {
			if player.ID == playerID {
				continue
			}
			worldPosition, err := s.ensureWorldPosition(player.ID, "lazy_create", nil)
			if err != nil {
				return PvpTargetsResponse{}, err
			}
			position := pvpWorldToPvpPosition(worldPosition)
			distance := pvpCoordinateDistance(selfPosition, position)
			if filter.Radius > 0 {
				if int(math.Abs(float64(position.X-filter.CenterX))) > filter.Radius || int(math.Abs(float64(position.Y-filter.CenterY))) > filter.Radius {
					continue
				}
			}
			relation := WorldRelationOther
			status := WorldTargetStatusNormal
			canScout := true
			canAttack := account.ID != requestAccountID
			canPlunder := canAttack
			canReinforce := true
			scoutReason := ""
			attackReason := ""
			plunderReason := ""
			reinforceReason := ""
			protectedUntil := ""
			if account.ID == requestAccountID {
				canScout = false
				canAttack = false
				canPlunder = false
				scoutReason = "同账号存档不能侦查"
				attackReason = "同账号存档不能攻击"
				plunderReason = "同账号存档不能掠夺"
			}
			available, err := s.canReinforceWorldMapTarget(playerID, player.ID)
			if err != nil {
				return PvpTargetsResponse{}, err
			}
			if !available {
				canReinforce = false
				reinforceReason = "目标增援来源已满"
			}
			targetPvpState, _ := s.repo.GetPvpPlayerState(player.ID, now)
			protected, protectionType, activeUntil := activePvpProtection(targetPvpState, now)
			if protected {
				status = worldTargetStatusForProtection(protectionType)
				canAttack = false
				canPlunder = false
				protectedUntil = activeUntil
				if attackReason == "" {
					attackReason = pvpProtectionReason(protectionType)
				}
				if plunderReason == "" {
					plunderReason = pvpProtectionReason(protectionType)
				}
			} else if canAttack {
				status = WorldTargetStatusAttackable
			}
			if requestPvpState.DailyAttackLimit > 0 && requestPvpState.DailyAttackCount >= requestPvpState.DailyAttackLimit && account.ID != requestAccountID {
				canAttack = false
				canPlunder = false
				attackReason = "今日攻击次数已用完"
				plunderReason = "今日攻击次数已用完"
				status = WorldTargetStatusUnavailable
			}
			reason := firstNonEmpty(scoutReason, attackReason, plunderReason, reinforceReason)
			items = append(items, PvpTargetSummary{
				PlayerID:         player.ID,
				Nickname:         player.Nickname,
				Faction:          player.Faction,
				Position:         position,
				Distance:         distance,
				Direction:        pvpCoordinateDirection(selfPosition, position),
				ReinforceSeconds: reinforcementTravelSecondsForDistance(distance, 1, now, nil),
				TotalArmy:        player.TotalArmy,
				BuildingLevel:    player.BuildingLevel,
				Relation:         relation,
				Status:           status,
				CanScout:         canScout,
				CanAttack:        canAttack,
				CanPlunder:       canPlunder,
				CanReinforce:     canReinforce,
				Protected:        protectedUntil != "",
				ProtectedUntil:   protectedUntil,
				Reason:           reason,
				ScoutReason:      scoutReason,
				AttackReason:     attackReason,
				PlunderReason:    plunderReason,
				ReinforceReason:  reinforceReason,
			})
		}
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].Distance != items[j].Distance {
			return items[i].Distance < items[j].Distance
		}
		if items[i].TotalArmy == items[j].TotalArmy {
			return items[i].PlayerID < items[j].PlayerID
		}
		return items[i].TotalArmy > items[j].TotalArmy
	})
	if filter.Limit > 0 && len(items) > filter.Limit {
		items = items[:filter.Limit]
	}
	return PvpTargetsResponse{
		Items:     items,
		Self:      selfPosition,
		WorldSize: defaultPvpWorldSize,
		CenterX:   filter.CenterX,
		CenterY:   filter.CenterY,
		Radius:    filter.Radius,
	}, nil
}

// GetPvpTarget 返回单个玩家 PVP 目标摘要。
func (s *Service) GetPvpTarget(playerID string, targetPlayerID string) (PvpTargetSummary, error) {
	targets, err := s.ListPvpTargetsInArea(playerID, PvpTargetFilter{CenterX: -1, CenterY: -1, Radius: pvpMaxViewRadius(), Limit: 200})
	if err != nil {
		return PvpTargetSummary{}, err
	}
	for _, target := range targets.Items {
		if target.PlayerID == strings.TrimSpace(targetPlayerID) {
			return target, nil
		}
	}
	return PvpTargetSummary{}, ErrPlayerNotFound
}

// normalizePvpTargetFilter 规范化 PVP 地图视野筛选参数。
func normalizePvpTargetFilter(filter PvpTargetFilter, self PvpWorldPosition) PvpTargetFilter {
	if filter.CenterX < 0 {
		filter.CenterX = self.X
	}
	if filter.CenterY < 0 {
		filter.CenterY = self.Y
	}
	filter.CenterX = clampInt(filter.CenterX, 0, defaultWorldWidth-1)
	filter.CenterY = clampInt(filter.CenterY, 0, defaultWorldHeight-1)
	if filter.Radius < 0 {
		filter.Radius = 0
	}
	if filter.Radius == 0 {
		filter.Radius = defaultPvpTargetRadius
	}
	if filter.Radius > pvpMaxViewRadius() {
		filter.Radius = pvpMaxViewRadius()
	}
	if filter.Limit <= 0 {
		filter.Limit = defaultPvpTargetLimit
	}
	if filter.Limit > 1000 {
		filter.Limit = 1000
	}
	return filter
}

// pvpWorldPositionForPlayer 根据玩家 ID 生成稳定伪随机世界坐标。
func pvpWorldPositionForPlayer(playerID string) PvpWorldPosition {
	hash := fnv.New64a()
	_, _ = hash.Write([]byte(strings.TrimSpace(playerID)))
	value := hash.Sum64()
	x := int(value % uint64(defaultWorldWidth))
	y := int((value / uint64(defaultWorldWidth)) % uint64(defaultWorldHeight))
	return PvpWorldPosition{
		WorldID:  defaultPvpWorldID,
		X:        x,
		Y:        y,
		RegionID: pvpRegionID(x, y),
	}
}

// pvpRegionID 为旧 PVP 兼容字段按 20 格区块生成区域编号。
func pvpRegionID(x int, y int) int {
	regionSize := 20
	regionColumns := defaultWorldWidth / regionSize
	col := clampInt(x, 0, defaultWorldWidth-1) / regionSize
	row := clampInt(y, 0, defaultWorldHeight-1) / regionSize
	return row*regionColumns + col + 1
}

// pvpMaxViewRadius 返回足够覆盖整张中心坐标地图的视野半径。
func pvpMaxViewRadius() int {
	return defaultWorldWidth + defaultWorldHeight
}

// pvpCoordinateDistance 返回两个地图坐标之间的直线距离。
func pvpCoordinateDistance(a PvpWorldPosition, b PvpWorldPosition) int {
	return worldMapDistance(WorldCoordinate{X: a.X, Y: a.Y}, WorldCoordinate{X: b.X, Y: b.Y})
}

// pvpCoordinateDirection 返回目标相对自己的八方向文本。
func pvpCoordinateDirection(a PvpWorldPosition, b PvpWorldPosition) string {
	dx := b.X - a.X
	dy := b.Y - a.Y
	if dx == 0 && dy == 0 {
		return "原地"
	}
	vertical := ""
	if dy < 0 {
		vertical = "北"
	} else if dy > 0 {
		vertical = "南"
	}
	horizontal := ""
	if dx < 0 {
		horizontal = "西"
	} else if dx > 0 {
		horizontal = "东"
	}
	return horizontal + vertical
}

// pvpWorldToPvpPosition 转换权威世界坐标到旧 PVP 兼容结构。
func pvpWorldToPvpPosition(position WorldPosition) PvpWorldPosition {
	return PvpWorldPosition{WorldID: position.WorldID, X: position.X, Y: position.Y, RegionID: pvpRegionID(position.X, position.Y)}
}

// clampInt 限制整数范围。
func clampInt(value int, min int, max int) int {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

// ScoutPvpTarget 派出当前阵营的全部侦察兵前往玩家目标。
func (s *Service) ScoutPvpTarget(req PvpScoutRequest) (PvpScoutResponse, error) {
	playerID := strings.TrimSpace(req.PlayerID)
	targetPlayerID := strings.TrimSpace(req.TargetPlayerID)
	if err := s.ensurePvpTargetAllowed(playerID, targetPlayerID); err != nil {
		return PvpScoutResponse{}, err
	}
	now := time.Now().UTC()
	attackerPosition, err := s.ensureWorldPosition(playerID, "lazy_create", nil)
	if err != nil {
		return PvpScoutResponse{}, err
	}
	defenderPosition, err := s.ensureWorldPosition(targetPlayerID, "lazy_create", nil)
	if err != nil {
		return PvpScoutResponse{}, err
	}
	distance := worldMapDistance(WorldCoordinate{X: attackerPosition.X, Y: attackerPosition.Y}, WorldCoordinate{X: defenderPosition.X, Y: defenderPosition.Y})
	nowText := now.Format(resourceDateLayout)
	var attackerState GameState
	var march PvpMarch
	for attempt := 0; attempt < 3; attempt++ {
		attackerState, _, march, err = s.repo.CreatePvpMarchWithState(playerID, targetPlayerID, now, func(attacker *GameState, defender *GameState) (PvpMarch, error) {
			nextState, _ := settleResources(*attacker, now)
			*attacker = nextState
			scoutUnitID := findScoutUnit(attacker.Player.Faction)
			if scoutUnitID == "" {
				return PvpMarch{}, ErrNoUnitsSelected
			}
			scoutCount, scoutIdx := armyUnitAmountAndIndex(attacker.Army, scoutUnitID)
			if scoutCount <= 0 {
				return PvpMarch{}, ErrInsufficientArmy
			}
			troops := map[string]int{scoutUnitID: scoutCount}
			durationSeconds, speedMultiplier, err := calculatePvpMarchTravel(distance, attacker.Player.Faction, troops, now, nil)
			if err != nil {
				return PvpMarch{}, err
			}
			durationSeconds = dispatchMarchCreateTraits(durationSeconds, PvpMarchTypeScout, attacker, nil)
			speedMultiplier = float64(defaultPvpMarchSeconds) / float64(durationSeconds)
			reduceArmyUnitAt(&attacker.Army, scoutIdx, scoutCount)
			attacker.ServerTime = nowText
			return PvpMarch{
				ID:               "pvp_march_" + randomID(12),
				AttackerPlayerID: attacker.Player.ID,
				AttackerName:     attacker.Player.Nickname,
				AttackerFaction:  attacker.Player.Faction,
				DefenderPlayerID: defender.Player.ID,
				DefenderName:     defender.Player.Nickname,
				DefenderFaction:  defender.Player.Faction,
				MarchType:        PvpMarchTypeScout,
				Status:           PvpMarchStatusMarching,
				AttackTroops:     troops,
				SpeedMultiplier:  speedMultiplier,
				DurationSeconds:  durationSeconds,
				StartedAt:        nowText,
				ArrivesAt:        now.Add(time.Duration(durationSeconds) * time.Second).Format(resourceDateLayout),
				CreatedAt:        nowText,
				UpdatedAt:        nowText,
			}, nil
		})
		if err == nil {
			break
		}
		if !isRetryableStorageConflict(err) || attempt == 2 {
			return PvpScoutResponse{}, err
		}
		time.Sleep(time.Duration(attempt+1) * 80 * time.Millisecond)
	}
	if err != nil {
		return PvpScoutResponse{}, err
	}
	return PvpScoutResponse{
		March:                march,
		Army:                 attackerState.Army,
		Resources:            attackerState.Resources,
		ResourceProduction:   attackerState.ResourceProduction,
		ResourceSettledAt:    attackerState.ResourceSettledAt,
		GeneralTraitProgress: cloneGeneralTraitProgress(attackerState.GeneralTraitProgress),
		ServerTime:           attackerState.ServerTime,
	}, nil
}

// armyUnitAmountAndIndex 返回指定兵种数量和位置。
func armyUnitAmountAndIndex(army []ArmyUnit, unitType string) (int, int) {
	if strings.TrimSpace(unitType) == "" {
		return 0, -1
	}
	for i, unit := range army {
		if unit.UnitType == unitType {
			return unit.Amount, i
		}
	}
	return 0, -1
}

// reduceArmyUnitAt 从指定位置扣减兵力，扣空后移除。
func reduceArmyUnitAt(army *[]ArmyUnit, idx int, amount int) {
	if army == nil || idx < 0 || idx >= len(*army) || amount <= 0 {
		return
	}
	(*army)[idx].Amount -= amount
	if (*army)[idx].Amount <= 0 {
		*army = append((*army)[:idx], (*army)[idx+1:]...)
	}
}

// resolveScoutSkirmish 按侦查规则结算双方侦察兵损耗和是否侦查成功。
func resolveScoutSkirmish(scoutCount int, targetScoutCount int) (int, int, bool) {
	if targetScoutCount <= 0 {
		return 0, 0, true
	}
	if scoutCount > targetScoutCount {
		return targetScoutCount, targetScoutCount, true
	}
	if scoutCount == targetScoutCount {
		return scoutCount, targetScoutCount, false
	}
	return scoutCount, scoutCount, false
}

// StartPvpAttack 创建 PVP 行军，扣出攻击方兵力并占用出征武将。
func (s *Service) StartPvpAttack(req PvpAttackRequest) (PvpAttackResponse, error) {
	playerID := strings.TrimSpace(req.PlayerID)
	targetPlayerID := strings.TrimSpace(req.TargetPlayerID)
	if err := s.ensurePvpTargetAllowed(playerID, targetPlayerID); err != nil {
		return PvpAttackResponse{}, err
	}
	now := time.Now().UTC()
	attackerPvpState, defenderPvpState, err := s.validatePvpAttackState(playerID, targetPlayerID, now)
	if err != nil {
		return PvpAttackResponse{}, err
	}
	attackerPosition, err := s.ensureWorldPosition(playerID, "lazy_create", nil)
	if err != nil {
		return PvpAttackResponse{}, err
	}
	defenderPosition, err := s.ensureWorldPosition(targetPlayerID, "lazy_create", nil)
	if err != nil {
		return PvpAttackResponse{}, err
	}
	distance := worldMapDistance(WorldCoordinate{X: attackerPosition.X, Y: attackerPosition.Y}, WorldCoordinate{X: defenderPosition.X, Y: defenderPosition.Y})
	troops := normalizePositiveTroops(req.Troops)
	if len(troops) == 0 {
		return PvpAttackResponse{}, ErrNoUnitsSelected
	}
	mode := normalizePvpMarchMode(req.MarchMode)
	nowText := now.Format(resourceDateLayout)
	var attackerState GameState
	var march PvpMarch
	for attempt := 0; attempt < 3; attempt++ {
		attackerState, _, march, err = s.repo.CreatePvpMarchWithState(playerID, targetPlayerID, now, func(attacker *GameState, defender *GameState) (PvpMarch, error) {
			nextState, _ := settleResources(*attacker, now)
			*attacker = nextState
			EnsureGeneralRoster(attacker, now)
			if _, err := validateAndConsumeArmy(attacker, troops); err != nil {
				return PvpMarch{}, err
			}
			marchID := "pvp_march_" + randomID(12)
			generalIDs, err := reservePvpGenerals(attacker, req.GeneralIDs, marchID, now)
			if err != nil {
				return PvpMarch{}, err
			}
			durationSeconds, speedMultiplier, err := calculatePvpMarchTravel(distance, attacker.Player.Faction, troops, now, pvpModifierSourcesForGenerals(attacker, generalIDs))
			if err != nil {
				return PvpMarch{}, err
			}
			durationSeconds = dispatchMarchCreateTraits(durationSeconds, mode, attacker, generalIDs)
			speedMultiplier = float64(defaultPvpMarchSeconds) / float64(durationSeconds)
			arrivesAt := now.Add(time.Duration(durationSeconds) * time.Second).Format(resourceDateLayout)
			attacker.ServerTime = nowText
			return PvpMarch{
				ID:               marchID,
				AttackerPlayerID: attacker.Player.ID,
				AttackerName:     attacker.Player.Nickname,
				AttackerFaction:  attacker.Player.Faction,
				DefenderPlayerID: defender.Player.ID,
				DefenderName:     defender.Player.Nickname,
				DefenderFaction:  defender.Player.Faction,
				MarchType:        mode,
				Status:           PvpMarchStatusMarching,
				AttackTroops:     cloneStringIntMap(troops),
				AttackGenerals:   generalIDs,
				SpeedMultiplier:  speedMultiplier,
				DurationSeconds:  durationSeconds,
				StartedAt:        nowText,
				ArrivesAt:        arrivesAt,
				CreatedAt:        nowText,
				UpdatedAt:        nowText,
			}, nil
		})
		if err == nil {
			break
		}
		if !isRetryableStorageConflict(err) || attempt == 2 {
			return PvpAttackResponse{}, err
		}
		time.Sleep(time.Duration(attempt+1) * 80 * time.Millisecond)
	}
	if err != nil {
		return PvpAttackResponse{}, err
	}
	attackerPvpState.DailyAttackCount++
	clearBreakablePvpProtectionOnAttack(&attackerPvpState, now)
	attackerPvpState.CooldownUntil = ""
	attackerPvpState.Status = "normal"
	attackerPvpState.UpdatedAt = nowText
	_ = defenderPvpState
	if err := s.repo.SavePvpPlayerState(attackerPvpState, now); err != nil {
		return PvpAttackResponse{}, err
	}
	return PvpAttackResponse{
		March:                march,
		Army:                 attackerState.Army,
		Resources:            attackerState.Resources,
		ResourceProduction:   attackerState.ResourceProduction,
		ResourceSettledAt:    attackerState.ResourceSettledAt,
		GeneralTraitProgress: cloneGeneralTraitProgress(attackerState.GeneralTraitProgress),
		Generals:             attackerState.Generals,
		GeneralAssignments:   attackerState.GeneralAssignments,
		ServerTime:           attackerState.ServerTime,
	}, nil
}

// ListPvpMarches 返回玩家相关行军，并尝试结算已经到达的行军。
func (s *Service) ListPvpMarches(playerID string) (PvpMarchListResponse, error) {
	playerID = strings.TrimSpace(playerID)
	if playerID == "" {
		return PvpMarchListResponse{}, ErrPlayerNotFound
	}
	if err := s.SettleDuePvpMarches(playerID); err != nil {
		return PvpMarchListResponse{}, err
	}
	items, err := s.repo.ListPvpMarchesForPlayer(playerID)
	if err != nil {
		return PvpMarchListResponse{}, err
	}
	sort.Slice(items, func(i, j int) bool { return items[i].UpdatedAt > items[j].UpdatedAt })
	for index := range items {
		items[index] = projectPvpMarchForViewer(items[index], playerID)
	}
	return PvpMarchListResponse{Items: items}, nil
}

// projectPvpMarchForViewer 按查看方生成安全行军视图，防守方只能预知来源、类型和到达时间。
func projectPvpMarchForViewer(march PvpMarch, playerID string) PvpMarch {
	projected := clonePvpMarch(march)
	if projected.AttackerPlayerID == playerID {
		projected.ViewerRole = PvpMarchViewerRoleSent
		return projected
	}
	if projected.DefenderPlayerID == playerID {
		projected.ViewerRole = PvpMarchViewerRoleIncoming
		projected.AttackTroops = map[string]int{}
		projected.AttackGenerals = nil
		projected.SpeedMultiplier = 0
		projected.DurationSeconds = 0
		projected.StartedAt = ""
		projected.ReturnStartedAt = ""
		projected.ReturnsAt = ""
		projected.ResolvedAt = ""
		projected.AttackerReportID = ""
		projected.DefenderReportID = ""
		projected.BattleID = ""
		projected.AcceleratedTimes = 0
		projected.CreatedAt = ""
		projected.UpdatedAt = ""
	}
	return projected
}

// RecallPvpMarch 召回玩家自己的 PVP 行军，进入返程状态。
func (s *Service) RecallPvpMarch(playerID string, marchID string) (PvpMarchActionResponse, error) {
	playerID = strings.TrimSpace(playerID)
	if playerID == "" {
		return PvpMarchActionResponse{}, ErrPlayerNotFound
	}
	now := time.Now().UTC()
	state, march, err := s.repo.UpdatePvpMarchWithAttackerState(strings.TrimSpace(marchID), now, func(attacker *GameState, march *PvpMarch) error {
		if march.AttackerPlayerID != playerID {
			return ErrPlayerNotFound
		}
		if march.Status != PvpMarchStatusMarching {
			return ErrPvpMarchNotRecallable
		}
		if !pvpMarchRecallableAt(march, now) {
			return ErrPvpMarchNotRecallable
		}
		arrivesAt, err := time.Parse(resourceDateLayout, strings.TrimSpace(march.ArrivesAt))
		if err == nil && !arrivesAt.After(now) {
			return ErrPvpMarchNotRecallable
		}
		returnSeconds := calculatePvpRecallReturnSeconds(march, now)
		nowText := now.Format(resourceDateLayout)
		march.Status = PvpMarchStatusReturning
		march.ReturnStartedAt = nowText
		march.ReturnsAt = now.Add(time.Duration(returnSeconds) * time.Second).Format(resourceDateLayout)
		march.UpdatedAt = nowText
		updatePvpGeneralAssignmentStatus(attacker, march, PvpMarchStatusReturning)
		attacker.ServerTime = nowText
		return nil
	})
	if err != nil {
		return PvpMarchActionResponse{}, err
	}
	return PvpMarchActionResponse{March: march, Army: state.Army, Generals: state.Generals, ServerTime: state.ServerTime}, nil
}

// pvpMarchRecallableAt 判断 PVP 行军是否仍处于出发后可召回窗口。
func pvpMarchRecallableAt(march *PvpMarch, now time.Time) bool {
	if march == nil {
		return false
	}
	startedAt, err := time.Parse(resourceDateLayout, strings.TrimSpace(march.StartedAt))
	if err != nil {
		return false
	}
	return now.Sub(startedAt) <= pvpRecallWindowSeconds*time.Second
}

// AcceleratePvpMarch 使用城金加速玩家自己的 PVP 行军。
func (s *Service) AcceleratePvpMarch(playerID string, marchID string) (PvpMarchActionResponse, error) {
	playerID = strings.TrimSpace(playerID)
	if playerID == "" {
		return PvpMarchActionResponse{}, ErrPlayerNotFound
	}
	now := time.Now().UTC()
	cost := 0
	state, march, err := s.repo.UpdatePvpMarchWithAttackerState(strings.TrimSpace(marchID), now, func(attacker *GameState, march *PvpMarch) error {
		if march.AttackerPlayerID != playerID {
			return ErrPlayerNotFound
		}
		if march.Status != PvpMarchStatusMarching {
			return ErrPvpMarchNotAccelerable
		}
		if march.AcceleratedTimes >= pvpMaxAccelerateTimes {
			return ErrPvpMarchNotAccelerable
		}
		arrivesAt, err := time.Parse(resourceDateLayout, strings.TrimSpace(march.ArrivesAt))
		if err != nil || !arrivesAt.After(now) {
			return ErrPvpMarchNotAccelerable
		}
		remainingSeconds := int(math.Ceil(arrivesAt.Sub(now).Seconds()))
		if remainingSeconds <= 1 {
			return ErrPvpMarchNotAccelerable
		}
		cost = pvpAccelerateFixedCityGoldCost
		if int(attacker.CityGold) < cost {
			return ErrInsufficientCityGold
		}
		attacker.CityGold -= FlexInt(cost)
		nextArrivesAt := now.Add(time.Duration((remainingSeconds+1)/2) * time.Second)
		march.ArrivesAt = nextArrivesAt.Format(resourceDateLayout)
		march.AcceleratedTimes++
		march.SpeedMultiplier = calculatePvpSpeedMultiplier(march)
		march.UpdatedAt = now.Format(resourceDateLayout)
		attacker.ServerTime = now.Format(resourceDateLayout)
		return nil
	})
	if err != nil {
		return PvpMarchActionResponse{}, err
	}
	if cost > 0 {
		s.recordLedger(GoldLedgerEntry{
			PlayerID:     playerID,
			Currency:     LedgerCurrencyCityGold,
			Direction:    LedgerDirectionDebit,
			Amount:       cost,
			BalanceAfter: int(state.CityGold),
			RefType:      LedgerRefPvpMarchAccelerate,
			RefID:        march.ID,
			Reason:       "pvp_march_accelerate",
		})
		s.publishCurrencyChanged(playerID, "", march.ID, LedgerRefPvpMarchAccelerate)
	}
	return PvpMarchActionResponse{March: march, Army: state.Army, Generals: state.Generals, CityGold: state.CityGold, Cost: cost, ServerTime: state.ServerTime}, nil
}

// ListPvpBattles 返回玩家相关 PVP 战斗记录。
func (s *Service) ListPvpBattles(playerID string) (PvpBattleListResponse, error) {
	if err := s.SettleDuePvpMarches(playerID); err != nil {
		return PvpBattleListResponse{}, err
	}
	items, err := s.repo.ListPvpBattlesForPlayer(strings.TrimSpace(playerID))
	if err != nil {
		return PvpBattleListResponse{}, err
	}
	for i := range items {
		items[i] = s.projectPvpBattleForPlayer(items[i], playerID)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].UpdatedAt > items[j].UpdatedAt })
	return PvpBattleListResponse{Items: items}, nil
}

// GetPvpBattle 返回单场 PVP 战斗详情，并校验玩家参与关系。
func (s *Service) GetPvpBattle(playerID string, battleID string) (PvpBattle, error) {
	playerID = strings.TrimSpace(playerID)
	if playerID == "" {
		return PvpBattle{}, ErrPlayerNotFound
	}
	battle, err := s.repo.GetPvpBattle(strings.TrimSpace(battleID))
	if err != nil {
		return PvpBattle{}, err
	}
	if battle.AttackerPlayerID != playerID && battle.DefenderPlayerID != playerID {
		return PvpBattle{}, ErrPlayerNotFound
	}
	return s.projectPvpBattleForPlayer(battle, playerID), nil
}

// projectPvpBattleForPlayer 让 PVP 战斗接口服从对应玩家战报的情报可见性，避免绕过战报详情读取原始快照。
func (s *Service) projectPvpBattleForPlayer(battle PvpBattle, playerID string) PvpBattle {
	reportID := battle.AttackerReportID
	isAttacker := battle.AttackerPlayerID == playerID
	if !isAttacker {
		reportID = battle.DefenderReportID
	}
	report, err := s.repo.GetReportForPlayer(playerID, reportID)
	if err != nil {
		if isAttacker {
			battle.DefenderSnapshot = nil
			battle.ReinforcementSnapshot = nil
		} else {
			battle.AttackerSnapshot = nil
		}
		return battle
	}
	visible := projectBattleReportForViewer(report)
	if visible.Detail == nil {
		if isAttacker {
			battle.DefenderSnapshot = nil
			battle.ReinforcementSnapshot = nil
		} else {
			battle.AttackerSnapshot = nil
		}
		return battle
	}
	visibility := visible.Detail.Visibility
	if !visibility.ShowEnemyRemainingUnits || !visibility.ShowEnemyResources || !visibility.ShowEnemyGenerals || !visibility.ShowEnemyCityDefense {
		if isAttacker {
			battle.DefenderSnapshot = nil
		} else {
			battle.AttackerSnapshot = nil
		}
	}
	if isAttacker && (!visibility.ShowEnemyRemainingUnits || !visibility.ShowEnemyGenerals) {
		battle.ReinforcementSnapshot = append([]DefenseReinforcementUnit(nil), battle.ReinforcementSnapshot...)
		for i := range battle.ReinforcementSnapshot {
			if !visibility.ShowEnemyRemainingUnits {
				battle.ReinforcementSnapshot[i].Troops = map[string]int{}
			}
			if !visibility.ShowEnemyGenerals {
				battle.ReinforcementSnapshot[i].Generals = nil
				battle.ReinforcementSnapshot[i].GeneralExpGained = 0
				battle.ReinforcementSnapshot[i].GeneralLevelBefore = 0
				battle.ReinforcementSnapshot[i].GeneralLevelAfter = 0
				battle.ReinforcementSnapshot[i].Buffs = nil
			}
		}
	}
	return battle
}

// GetPvpState 返回玩家 PVP 状态、积分统计和复仇记录。
func (s *Service) GetPvpState(playerID string) (PvpStateResponse, error) {
	playerID = strings.TrimSpace(playerID)
	if playerID == "" {
		return PvpStateResponse{}, ErrPlayerNotFound
	}
	now := time.Now().UTC()
	state, err := s.repo.GetPvpPlayerState(playerID, now)
	if err != nil {
		return PvpStateResponse{}, err
	}
	records := filterActivePvpRevengeRecords(pvpRevengeRecordsFromMetadata(state.Metadata), now)
	state.Metadata["revengeRecords"] = records
	if err := s.repo.SavePvpPlayerState(state, now); err != nil {
		return PvpStateResponse{}, err
	}
	return PvpStateResponse{
		State:          state,
		SeasonPoints:   pvpMetadataInt(state.Metadata, "seasonPoints"),
		Rating:         pvpMetadataInt(state.Metadata, "rating"),
		AttackWins:     pvpMetadataInt(state.Metadata, "attackWins"),
		DefenseWins:    pvpMetadataInt(state.Metadata, "defenseWins"),
		Losses:         pvpMetadataInt(state.Metadata, "losses"),
		RevengeRecords: records,
		ServerTime:     now.Format(resourceDateLayout),
	}, nil
}

// ListPvpRevengeRecords 返回玩家当前可用复仇记录。
func (s *Service) ListPvpRevengeRecords(playerID string) (PvpRevengeListResponse, error) {
	state, err := s.GetPvpState(playerID)
	if err != nil {
		return PvpRevengeListResponse{}, err
	}
	return PvpRevengeListResponse{Items: state.RevengeRecords, ServerTime: state.ServerTime}, nil
}

// GetPvpSeason 返回当前独立赛季摘要和玩家自身排行。
func (s *Service) GetPvpSeason(playerID string) (PvpSeasonResponse, error) {
	playerID = strings.TrimSpace(playerID)
	if playerID == "" {
		return PvpSeasonResponse{}, ErrPlayerNotFound
	}
	now := time.Now().UTC()
	season, err := s.ensureCurrentPvpSeason(now)
	if err != nil {
		return PvpSeasonResponse{}, err
	}
	rankings, err := s.buildPvpRankings(now)
	if err != nil {
		return PvpSeasonResponse{}, err
	}
	var self *PvpRankingEntry
	for i := range rankings {
		if rankings[i].PlayerID == playerID {
			item := rankings[i]
			self = &item
			break
		}
	}
	if self == nil {
		if _, err := s.repo.GetPvpPlayerState(playerID, now); err != nil {
			return PvpSeasonResponse{}, err
		}
	}
	return PvpSeasonResponse{Season: pvpSeasonSummaryFromRecord(season), Self: self, ServerTime: now.Format(resourceDateLayout)}, nil
}

// ListPvpRankings 返回当前独立赛季排行榜。
func (s *Service) ListPvpRankings(playerID string, limit int) (PvpRankingResponse, error) {
	playerID = strings.TrimSpace(playerID)
	if playerID == "" {
		return PvpRankingResponse{}, ErrPlayerNotFound
	}
	if limit <= 0 {
		limit = 50
	}
	if limit > 100 {
		limit = 100
	}
	now := time.Now().UTC()
	season, err := s.ensureCurrentPvpSeason(now)
	if err != nil {
		return PvpRankingResponse{}, err
	}
	rankings, err := s.buildPvpRankings(now)
	if err != nil {
		return PvpRankingResponse{}, err
	}
	var self *PvpRankingEntry
	for i := range rankings {
		if rankings[i].PlayerID == playerID {
			item := rankings[i]
			self = &item
			break
		}
	}
	if len(rankings) > limit {
		rankings = rankings[:limit]
	}
	return PvpRankingResponse{Season: pvpSeasonSummaryFromRecord(season), Items: rankings, Self: self, ServerTime: now.Format(resourceDateLayout)}, nil
}

// SetPvpProtection 设置玩家 PVP 保护状态，供免战道具、系统保护和维护保护复用。
func (s *Service) SetPvpProtection(playerID string, protectionType string, duration time.Duration, reason string, now time.Time) (PvpStateResponse, error) {
	playerID = strings.TrimSpace(playerID)
	protectionType = normalizePvpProtectionType(protectionType)
	if playerID == "" {
		return PvpStateResponse{}, ErrPlayerNotFound
	}
	if protectionType == "" || duration <= 0 {
		return PvpStateResponse{}, ErrInvalidAmount
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	state, err := s.repo.GetPvpPlayerState(playerID, now)
	if err != nil {
		return PvpStateResponse{}, err
	}
	protectedUntil := now.Add(duration).UTC().Format(resourceDateLayout)
	if currentProtected, currentType, currentUntil := activePvpProtection(state, now); currentProtected && !shouldOverridePvpProtection(currentType, currentUntil, protectionType, protectedUntil) {
		return s.GetPvpState(playerID)
	}
	state.Status = "protected"
	state.ProtectionType = protectionType
	state.ProtectedUntil = protectedUntil
	if state.Metadata == nil {
		state.Metadata = map[string]any{}
	}
	state.Metadata["protectionReason"] = strings.TrimSpace(reason)
	state.Metadata["protectionUpdatedAt"] = now.UTC().Format(resourceDateLayout)
	state.UpdatedAt = now.UTC().Format(resourceDateLayout)
	if err := s.repo.SavePvpPlayerState(state, now); err != nil {
		return PvpStateResponse{}, err
	}
	return s.GetPvpState(playerID)
}

// AdminPvpOverview 返回 GM 后台 PVP 只读总览。
func (s *Service) AdminPvpOverview(playerID string, limit int) (AdminPvpOverviewResponse, error) {
	playerID = strings.TrimSpace(playerID)
	limit = normalizeAdminPvpLimit(limit)
	now := time.Now().UTC()
	marches, err := s.AdminPvpMarches(playerID, limit)
	if err != nil {
		return AdminPvpOverviewResponse{}, err
	}
	battles, err := s.AdminPvpBattles(playerID, limit)
	if err != nil {
		return AdminPvpOverviewResponse{}, err
	}
	rankings, err := s.buildPvpRankings(now)
	if err != nil {
		return AdminPvpOverviewResponse{}, err
	}
	if len(rankings) > limit {
		rankings = rankings[:limit]
	}
	var player *PvpStateResponse
	if playerID != "" {
		state, err := s.GetPvpState(playerID)
		if err != nil {
			return AdminPvpOverviewResponse{}, err
		}
		player = &state
	}
	season, err := s.ensureCurrentPvpSeason(now)
	if err != nil {
		return AdminPvpOverviewResponse{}, err
	}
	return AdminPvpOverviewResponse{
		PlayerID:   playerID,
		Player:     player,
		Season:     pvpSeasonSummaryFromRecord(season),
		Rankings:   rankings,
		Marches:    marches.Items,
		Battles:    battles.Items,
		ServerTime: now.Format(resourceDateLayout),
	}, nil
}

// AdminPvpMarches 返回 GM 后台 PVP 行军列表，可按玩家筛选。
func (s *Service) AdminPvpMarches(playerID string, limit int) (PvpMarchListResponse, error) {
	playerID = strings.TrimSpace(playerID)
	limit = normalizeAdminPvpLimit(limit)
	playerIDs, err := s.adminPvpPlayerIDs(playerID)
	if err != nil {
		return PvpMarchListResponse{}, err
	}
	seen := map[string]bool{}
	items := []PvpMarch{}
	for _, id := range playerIDs {
		marches, err := s.repo.ListPvpMarchesForPlayer(id)
		if err != nil {
			return PvpMarchListResponse{}, err
		}
		for _, march := range marches {
			if seen[march.ID] {
				continue
			}
			seen[march.ID] = true
			items = append(items, march)
		}
	}
	sort.Slice(items, func(i, j int) bool { return items[i].UpdatedAt > items[j].UpdatedAt })
	if len(items) > limit {
		items = items[:limit]
	}
	return PvpMarchListResponse{Items: items}, nil
}

// AdminPvpBattles 返回 GM 后台 PVP 战斗列表，可按玩家筛选。
func (s *Service) AdminPvpBattles(playerID string, limit int) (PvpBattleListResponse, error) {
	playerID = strings.TrimSpace(playerID)
	limit = normalizeAdminPvpLimit(limit)
	playerIDs, err := s.adminPvpPlayerIDs(playerID)
	if err != nil {
		return PvpBattleListResponse{}, err
	}
	seen := map[string]bool{}
	items := []PvpBattle{}
	for _, id := range playerIDs {
		battles, err := s.repo.ListPvpBattlesForPlayer(id)
		if err != nil {
			return PvpBattleListResponse{}, err
		}
		for _, battle := range battles {
			if seen[battle.ID] {
				continue
			}
			seen[battle.ID] = true
			items = append(items, battle)
		}
	}
	sort.Slice(items, func(i, j int) bool { return items[i].UpdatedAt > items[j].UpdatedAt })
	if len(items) > limit {
		items = items[:limit]
	}
	return PvpBattleListResponse{Items: items}, nil
}

// AdminPvpSeasons 返回 GM 赛季列表。
func (s *Service) AdminPvpSeasons() (AdminPvpSeasonListResponse, error) {
	now := time.Now().UTC()
	current, err := s.ensureCurrentPvpSeason(now)
	if err != nil {
		return AdminPvpSeasonListResponse{}, err
	}
	seasons, err := s.repo.ListPvpSeasons()
	if err != nil {
		return AdminPvpSeasonListResponse{}, err
	}
	return AdminPvpSeasonListResponse{
		Current:    pvpSeasonSummaryFromRecord(current),
		Seasons:    seasons,
		ServerTime: now.Format(resourceDateLayout),
	}, nil
}

// AdminCreatePvpSeason 创建一个 GM 配置的 PVP 赛季。
func (s *Service) AdminCreatePvpSeason(req AdminSavePvpSeasonRequest) (PvpSeasonRecord, error) {
	now := time.Now().UTC()
	season, err := buildAdminPvpSeasonRecord(req, now)
	if err != nil {
		return PvpSeasonRecord{}, err
	}
	if season.ID == "" {
		season.ID = "pvp_season_" + randomID(12)
	}
	if err := s.repo.SavePvpSeason(season, now); err != nil {
		return PvpSeasonRecord{}, err
	}
	return season, nil
}

// AdminUpdatePvpSeason 更新一个 GM 配置的 PVP 赛季。
func (s *Service) AdminUpdatePvpSeason(seasonID string, req AdminSavePvpSeasonRequest) (PvpSeasonRecord, error) {
	seasonID = strings.TrimSpace(seasonID)
	if seasonID == "" {
		return PvpSeasonRecord{}, ErrInvalidPvpSeason
	}
	existing, err := s.findPvpSeasonByID(seasonID)
	if err != nil {
		return PvpSeasonRecord{}, err
	}
	now := time.Now().UTC()
	req.ID = seasonID
	season, err := buildAdminPvpSeasonRecord(req, now)
	if err != nil {
		return PvpSeasonRecord{}, err
	}
	season.CreatedAt = existing.CreatedAt
	if season.SettledAt == "" {
		season.SettledAt = existing.SettledAt
	}
	if err := s.repo.SavePvpSeason(season, now); err != nil {
		return PvpSeasonRecord{}, err
	}
	return season, nil
}

// AdminSettlePvpSeason 结算赛季，固化排行榜并发送奖励邮件。
func (s *Service) AdminSettlePvpSeason(seasonID string) (AdminSettlePvpSeasonResponse, error) {
	now := time.Now().UTC()
	season, err := s.ensureCurrentPvpSeason(now)
	if err != nil {
		return AdminSettlePvpSeasonResponse{}, err
	}
	seasonID = strings.TrimSpace(seasonID)
	if seasonID != "" && seasonID != season.ID {
		seasons, err := s.repo.ListPvpSeasons()
		if err != nil {
			return AdminSettlePvpSeasonResponse{}, err
		}
		found := false
		for _, item := range seasons {
			if item.ID == seasonID {
				season = item
				found = true
				break
			}
		}
		if !found {
			return AdminSettlePvpSeasonResponse{}, ErrPlayerNotFound
		}
	}
	if season.Status == PvpSeasonStatusSettled {
		players, err := s.repo.ListPvpSeasonPlayers(season.ID)
		if err != nil {
			return AdminSettlePvpSeasonResponse{}, err
		}
		return AdminSettlePvpSeasonResponse{Season: season, Players: players, ServerTime: now.Format(resourceDateLayout)}, nil
	}
	rankings, err := s.buildPvpRankings(now)
	if err != nil {
		return AdminSettlePvpSeasonResponse{}, err
	}
	players := make([]PvpSeasonPlayerRecord, 0, len(rankings))
	rewardMailCount := 0
	nowText := now.Format(resourceDateLayout)
	for _, ranking := range rankings {
		rewardAmount := pvpSeasonRewardCityGold(ranking.Rank)
		rewardMailID := ""
		rewardSentAt := ""
		if rewardAmount > 0 {
			mail, err := s.SendMail(SendMailRequest{
				PlayerID:    ranking.PlayerID,
				MailType:    PvpSeasonRewardMailType,
				SenderType:  "system",
				SenderName:  "PVP 赛季",
				Title:       season.Name + " 赛季奖励",
				Content:     "你在本赛季 PVP 排行中获得第 " + strconv.Itoa(ranking.Rank) + " 名，请领取赛季奖励。",
				Attachments: []MailAttachment{{Type: RewardTypeCityGold, ItemID: RewardTypeCityGold, Amount: rewardAmount}},
				SourceType:  "pvp_season",
				SourceID:    season.ID,
			})
			if err != nil {
				return AdminSettlePvpSeasonResponse{}, err
			}
			rewardMailID = mail.ID
			rewardSentAt = mail.CreatedAt
			rewardMailCount++
		}
		players = append(players, PvpSeasonPlayerRecord{
			SeasonID:     season.ID,
			PlayerID:     ranking.PlayerID,
			Nickname:     ranking.Nickname,
			Faction:      ranking.Faction,
			Rank:         ranking.Rank,
			Points:       ranking.Points,
			Rating:       ranking.Rating,
			Wins:         ranking.AttackWins,
			Losses:       ranking.Losses,
			DefenseWins:  ranking.DefenseWins,
			RewardMailID: rewardMailID,
			RewardSentAt: rewardSentAt,
			CreatedAt:    nowText,
			UpdatedAt:    nowText,
		})
	}
	if err := s.repo.SavePvpSeasonPlayers(season.ID, players, now); err != nil {
		return AdminSettlePvpSeasonResponse{}, err
	}
	season.Status = PvpSeasonStatusSettled
	season.SettledAt = nowText
	if err := s.repo.SavePvpSeason(season, now); err != nil {
		return AdminSettlePvpSeasonResponse{}, err
	}
	return AdminSettlePvpSeasonResponse{Season: season, Players: players, RewardMail: rewardMailCount, ServerTime: nowText}, nil
}

// AdminForceResolvePvpMarch 强制把未到达 PVP 行军推进到战斗结算。
func (s *Service) AdminForceResolvePvpMarch(marchID string) (PvpBattle, error) {
	marchID = strings.TrimSpace(marchID)
	if marchID == "" {
		return PvpBattle{}, ErrPlayerNotFound
	}
	now := time.Now().UTC()
	if _, err := s.repo.UpdatePvpMarch(marchID, now, func(march *PvpMarch) error {
		if march.Status != PvpMarchStatusMarching && march.Status != PvpMarchStatusResolving {
			return ErrPvpMarchNotReady
		}
		march.ArrivesAt = now.Format(resourceDateLayout)
		march.UpdatedAt = march.ArrivesAt
		return nil
	}); err != nil {
		return PvpBattle{}, err
	}
	return s.ResolvePvpMarch(marchID)
}

// AdminCancelPvpMarch 取消未结算的异常 PVP 行军，并返还攻击方出征兵力。
func (s *Service) AdminCancelPvpMarch(marchID string) (PvpMarchActionResponse, error) {
	marchID = strings.TrimSpace(marchID)
	if marchID == "" {
		return PvpMarchActionResponse{}, ErrPlayerNotFound
	}
	now := time.Now().UTC()
	state, march, err := s.repo.UpdatePvpMarchWithAttackerState(marchID, now, func(attacker *GameState, march *PvpMarch) error {
		if march.BattleID != "" || march.Status == PvpMarchStatusResolved || march.Status == PvpMarchStatusRecalled || march.Status == PvpMarchStatusCancelled {
			return ErrPvpMarchNotRecallable
		}
		if march.Status != PvpMarchStatusMarching && march.Status != PvpMarchStatusReturning && march.Status != PvpMarchStatusResolving {
			return ErrPvpMarchNotRecallable
		}
		for unitType, amount := range march.AttackTroops {
			if amount > 0 {
				addToArmy(&attacker.Army, unitType, amount)
			}
		}
		settleResourcesBeforeGeneralRelease(attacker, now)
		releasePvpGenerals(attacker, march)
		refreshResourcesAfterGeneralRelease(attacker, now)
		nowText := now.Format(resourceDateLayout)
		march.Status = PvpMarchStatusCancelled
		march.ResolvedAt = nowText
		march.UpdatedAt = nowText
		attacker.ServerTime = nowText
		return nil
	})
	if err != nil {
		return PvpMarchActionResponse{}, err
	}
	return PvpMarchActionResponse{March: march, Army: state.Army, Generals: state.Generals, ServerTime: state.ServerTime}, nil
}

// adminPvpPlayerIDs 返回 GM 查询要扫描的玩家 ID。
func (s *Service) adminPvpPlayerIDs(playerID string) ([]string, error) {
	playerID = strings.TrimSpace(playerID)
	if playerID != "" {
		if _, err := s.repo.GetState(playerID); err != nil {
			return nil, err
		}
		return []string{playerID}, nil
	}
	accounts, err := s.repo.ListAccounts()
	if err != nil {
		return nil, err
	}
	playerIDs := []string{}
	for _, account := range accounts {
		for _, player := range account.Players {
			if strings.TrimSpace(player.ID) != "" {
				playerIDs = append(playerIDs, player.ID)
			}
		}
	}
	return playerIDs, nil
}

// normalizeAdminPvpLimit 归一化 GM PVP 查询列表上限。
func normalizeAdminPvpLimit(limit int) int {
	if limit <= 0 {
		return 100
	}
	if limit > 300 {
		return 300
	}
	return limit
}

// SettleDuePvpMarches 结算玩家相关且已到达的 PVP 行军。
func (s *Service) SettleDuePvpMarches(playerID string) error {
	due, err := s.repo.ListDuePvpMarches(strings.TrimSpace(playerID), time.Now().UTC())
	if err != nil {
		return err
	}
	for _, march := range due {
		switch march.Status {
		case PvpMarchStatusMarching:
			if _, err := s.ResolvePvpMarch(march.ID); err != nil && !errors.Is(err, ErrPvpMarchNotReady) {
				if errors.Is(err, ErrNoUnitsSelected) {
					if _, failErr := s.FailInvalidPvpMarch(march.ID); failErr != nil {
						return failErr
					}
					continue
				}
				return err
			}
		case PvpMarchStatusReturning:
			if _, err := s.CompletePvpRecall(march.ID); err != nil {
				return err
			}
		}
	}
	return nil
}

// FailInvalidPvpMarch 将无法进入战斗的异常 PVP 行军置为失败，并释放出征武将。
func (s *Service) FailInvalidPvpMarch(marchID string) (PvpMarchActionResponse, error) {
	now := time.Now().UTC()
	state, march, err := s.repo.UpdatePvpMarchWithAttackerState(strings.TrimSpace(marchID), now, func(attacker *GameState, march *PvpMarch) error {
		if march.Status != PvpMarchStatusMarching && march.Status != PvpMarchStatusResolving {
			return ErrPvpMarchNotRecallable
		}
		settleResourcesBeforeGeneralRelease(attacker, now)
		releasePvpGenerals(attacker, march)
		refreshResourcesAfterGeneralRelease(attacker, now)
		nowText := now.Format(resourceDateLayout)
		march.Status = PvpMarchStatusFailed
		march.ResolvedAt = nowText
		march.UpdatedAt = nowText
		attacker.ServerTime = nowText
		return nil
	})
	if err != nil {
		return PvpMarchActionResponse{}, err
	}
	return PvpMarchActionResponse{March: march, Army: state.Army, Generals: state.Generals, ServerTime: state.ServerTime}, nil
}

// CompletePvpRecall 完成 PVP 返程；召回行军标记为已召回，战后幸存返程标记为已结算。
func (s *Service) CompletePvpRecall(marchID string) (PvpMarchActionResponse, error) {
	now := time.Now().UTC()
	state, march, err := s.repo.UpdatePvpMarchWithAttackerState(strings.TrimSpace(marchID), now, func(attacker *GameState, march *PvpMarch) error {
		if march.Status != PvpMarchStatusReturning {
			return ErrPvpMarchNotRecallable
		}
		returnsAt, err := time.Parse(resourceDateLayout, strings.TrimSpace(march.ReturnsAt))
		if err == nil && returnsAt.After(now) {
			return ErrPvpMarchNotReady
		}
		for unitType, amount := range march.AttackTroops {
			if amount > 0 {
				addToArmy(&attacker.Army, unitType, amount)
			}
		}
		settleResourcesBeforeGeneralRelease(attacker, now)
		releasePvpGenerals(attacker, march)
		refreshResourcesAfterGeneralRelease(attacker, now)
		nowText := now.Format(resourceDateLayout)
		if strings.TrimSpace(march.AttackerReportID) != "" {
			march.Status = PvpMarchStatusResolved
		} else {
			march.Status = PvpMarchStatusRecalled
		}
		march.ResolvedAt = nowText
		march.UpdatedAt = nowText
		attacker.ServerTime = nowText
		return nil
	})
	if err != nil {
		return PvpMarchActionResponse{}, err
	}
	return PvpMarchActionResponse{March: march, Army: state.Army, Generals: state.Generals, ServerTime: state.ServerTime}, nil
}

// ResolvePvpMarch 结算一条已经到达的 PVP 行军。
func (s *Service) ResolvePvpMarch(marchID string) (PvpBattle, error) {
	now := time.Now().UTC()
	marchID = strings.TrimSpace(marchID)
	if existingMarch, lookupErr := s.repo.GetPvpMarch(marchID); lookupErr == nil && strings.TrimSpace(existingMarch.BattleID) != "" && (existingMarch.Status == PvpMarchStatusReturning || existingMarch.Status == PvpMarchStatusResolved) {
		battle, err := s.repo.GetPvpBattle(existingMarch.BattleID)
		if err != nil {
			return PvpBattle{}, err
		}
		if err := s.applyPvpReinforcementGeneralExp(battle, now); err != nil {
			return battle, err
		}
		return battle, nil
	}
	_, _, _, battle, attackerReport, defenderReport, err := s.repo.ResolvePvpBattleTransaction(marchID, now, func(attacker *GameState, defender *GameState, reinforcements []Reinforcement, march *PvpMarch) (PvpBattle, BattleReport, BattleReport, []BattleReport, []Reinforcement, error) {
		arrivesAt, err := time.Parse(resourceDateLayout, march.ArrivesAt)
		if err == nil && arrivesAt.After(now) {
			return PvpBattle{}, BattleReport{}, BattleReport{}, nil, nil, ErrPvpMarchNotReady
		}
		EnsureGeneralRoster(attacker, now)
		EnsureGeneralRoster(defender, now)
		nextDefender, _ := settleResources(*defender, now)
		*defender = nextDefender
		if march.MarchType == PvpMarchTypeScout {
			attackerReport, defenderReport, err := resolvePvpScoutMarch(attacker, defender, march, now)
			if err != nil {
				return PvpBattle{}, BattleReport{}, BattleReport{}, nil, nil, err
			}
			nowText := now.Format(resourceDateLayout)
			if totalTroops(march.AttackTroops) > 0 {
				returnSeconds := calculatePvpOutboundTravelSeconds(march)
				march.Status = PvpMarchStatusReturning
				march.ReturnStartedAt = nowText
				march.ReturnsAt = now.Add(time.Duration(returnSeconds) * time.Second).Format(resourceDateLayout)
				march.SpeedMultiplier = calculatePvpSpeedMultiplier(march)
			} else {
				march.Status = PvpMarchStatusResolved
				march.ResolvedAt = nowText
				settleResourcesBeforeGeneralRelease(attacker, now)
				releasePvpGenerals(attacker, march)
				refreshResourcesAfterGeneralRelease(attacker, now)
			}
			march.AttackerReportID = attackerReport.ID
			march.DefenderReportID = defenderReport.ID
			march.UpdatedAt = nowText
			attacker.ServerTime = nowText
			defender.ServerTime = nowText
			return PvpBattle{}, attackerReport, defenderReport, nil, nil, nil
		}
		result, attackerReport, defenderReport, reinforcementReports, changedReinforcements, err := resolvePvpCombat(attacker, defender, reinforcements, march, now)
		if err != nil {
			return PvpBattle{}, BattleReport{}, BattleReport{}, nil, nil, err
		}
		nowText := now.Format(resourceDateLayout)
		if totalTroops(march.AttackTroops) > 0 {
			returnSeconds := calculatePvpOutboundTravelSeconds(march)
			march.Status = PvpMarchStatusReturning
			march.ReturnStartedAt = nowText
			march.ReturnsAt = now.Add(time.Duration(returnSeconds) * time.Second).Format(resourceDateLayout)
			march.SpeedMultiplier = calculatePvpSpeedMultiplier(march)
			updatePvpGeneralAssignmentStatus(attacker, march, PvpMarchStatusReturning)
		} else {
			march.Status = PvpMarchStatusResolved
			march.ResolvedAt = nowText
			settleResourcesBeforeGeneralRelease(attacker, now)
			releasePvpGenerals(attacker, march)
			refreshResourcesAfterGeneralRelease(attacker, now)
		}
		march.AttackerReportID = attackerReport.ID
		march.DefenderReportID = defenderReport.ID
		march.BattleID = result.ID
		march.UpdatedAt = nowText
		attacker.ServerTime = nowText
		defender.ServerTime = nowText
		return result, attackerReport, defenderReport, reinforcementReports, changedReinforcements, nil
	})
	// 仓储读取已结算战斗时不会再次返回新战报，以此区分首次提交和幂等读取。
	newlyResolved := attackerReport.ID != "" || defenderReport.ID != ""
	if err == nil && battle.ID != "" {
		if newlyResolved {
			s.applyPvpBattleStateEffects(battle, now)
		}
		if expErr := s.applyPvpReinforcementGeneralExp(battle, now); expErr != nil {
			return battle, expErr
		}
	}
	return battle, err
}

// resolvePvpScoutMarch 在侦查行军抵达后结算双方侦察兵损耗并生成双视角战报。
func resolvePvpScoutMarch(attacker *GameState, defender *GameState, march *PvpMarch, now time.Time) (BattleReport, BattleReport, error) {
	if attacker == nil || defender == nil || march == nil {
		return BattleReport{}, BattleReport{}, ErrPlayerNotFound
	}
	scoutUnitID := findScoutUnit(attacker.Player.Faction)
	scoutCount := march.AttackTroops[scoutUnitID]
	if scoutUnitID == "" || scoutCount <= 0 {
		return BattleReport{}, BattleReport{}, ErrNoUnitsSelected
	}
	targetScoutUnitID := findScoutUnit(defender.Player.Faction)
	targetScoutCount, targetScoutIdx := armyUnitAmountAndIndex(defender.Army, targetScoutUnitID)
	targetUnitsBefore := armySliceToMap(defender.Army)
	scoutLost, targetLost, success := resolveScoutSkirmish(scoutCount, targetScoutCount)
	if targetLost > 0 && targetScoutIdx >= 0 {
		reduceArmyUnitAt(&defender.Army, targetScoutIdx, targetLost)
	}
	scoutSurvived := scoutCount - scoutLost
	if scoutSurvived > 0 {
		march.AttackTroops = map[string]int{scoutUnitID: scoutSurvived}
	} else {
		march.AttackTroops = map[string]int{}
	}
	lostUnits := map[string]int{}
	if scoutLost > 0 {
		lostUnits[scoutUnitID] = scoutLost
	}
	targetLostUnits := map[string]int{}
	if targetLost > 0 && targetScoutUnitID != "" {
		targetLostUnits[targetScoutUnitID] = targetLost
	}
	result := "attacker_victory"
	if !success {
		result = "defender_victory"
	}
	nowText := now.Format(resourceDateLayout)
	scoutUnitsBefore := map[string]int{scoutUnitID: scoutCount}
	attackerReport := BattleReport{
		ID:                "br_pvp_scout_" + randomID(8),
		PlayerID:          attacker.Player.ID,
		OwnerPlayerID:     attacker.Player.ID,
		ViewType:          ReportViewScout,
		OwnerSide:         ReportOwnerSideScout,
		PlayerFaction:     attacker.Player.Faction,
		PlayerName:        attacker.Player.Nickname,
		TargetID:          defender.Player.ID,
		TargetName:        defender.Player.Nickname + "（玩家）",
		Type:              PvpMarchTypeScout,
		SourceType:        ReportSourcePlayerCity,
		BattleType:        PvpMarchTypeScout,
		Result:            result,
		OwnerOutcome:      map[bool]string{true: ReportOwnerOutcomeIntelSuccess, false: ReportOwnerOutcomeIntelFailed}[success],
		PlayerPower:       scoutCount,
		EnemyPower:        targetScoutCount,
		DispatchedUnits:   scoutUnitsBefore,
		LostUnits:         lostUnits,
		DefenderFaction:   defender.Player.Faction,
		DefenderLostUnits: targetLostUnits,
		DefenderRevealed:  success,
		Rewards:           map[string]int{},
		Read:              false,
		CreatedAt:         nowText,
	}
	if success {
		attackerReport.DefenderUnits = armySliceToMap(defender.Army)
		attackerReport.DefenderResources = copyResources(defender.Resources.Items)
	} else {
		attackerReport.DefenderUnits = map[string]int{}
		attackerReport.DefenderResources = map[string]int{}
	}
	defenderReport := BattleReport{
		ID:                "br_pvp_scout_def_" + randomID(8),
		PlayerID:          defender.Player.ID,
		OwnerPlayerID:     defender.Player.ID,
		ViewType:          ReportViewDefense,
		OwnerSide:         ReportOwnerSideDefender,
		OwnerOutcome:      ReportOwnerOutcomeNotice,
		PlayerFaction:     defender.Player.Faction,
		PlayerName:        defender.Player.Nickname,
		TargetID:          attacker.Player.ID,
		TargetName:        attacker.Player.Nickname + "（玩家）",
		Type:              PvpMarchTypeScout,
		SourceType:        ReportSourcePlayerCity,
		BattleType:        PvpMarchTypeScout,
		Title:             attacker.Player.Nickname + " 侦查 " + defender.Player.Nickname + "（玩家）",
		Result:            result,
		PlayerPower:       targetScoutCount,
		EnemyPower:        scoutCount,
		DispatchedUnits:   targetUnitsBefore,
		LostUnits:         targetLostUnits,
		DefenderFaction:   attacker.Player.Faction,
		DefenderUnits:     scoutUnitsBefore,
		DefenderLostUnits: lostUnits,
		DefenderRevealed:  true,
		DefenderResources: map[string]int{},
		Rewards:           map[string]int{},
		Read:              false,
		CreatedAt:         nowText,
	}
	return NormalizeBattleReport(attackerReport), NormalizeBattleReport(defenderReport), nil
}

// ensurePvpTargetAllowed 校验 PVP 目标是否允许交互。
func (s *Service) ensurePvpTargetAllowed(playerID string, targetPlayerID string) error {
	playerID = strings.TrimSpace(playerID)
	targetPlayerID = strings.TrimSpace(targetPlayerID)
	if playerID == "" || targetPlayerID == "" {
		return ErrPlayerNotFound
	}
	if playerID == targetPlayerID {
		return ErrPvpTargetSelf
	}
	attackerAccountID, err := s.repo.GetAccountIDByPlayerID(playerID)
	if err != nil {
		return err
	}
	defenderAccountID, err := s.repo.GetAccountIDByPlayerID(targetPlayerID)
	if err != nil {
		return err
	}
	if attackerAccountID == defenderAccountID {
		return ErrPvpSameAccountTarget
	}
	return nil
}

func normalizePvpMarchMode(mode string) string {
	if strings.TrimSpace(mode) == PvpMarchTypePlunder {
		return PvpMarchTypePlunder
	}
	return PvpMarchTypeAttack
}

// validatePvpAttackState 校验保护和每日次数。
func (s *Service) validatePvpAttackState(playerID string, targetPlayerID string, now time.Time) (PvpPlayerState, PvpPlayerState, error) {
	attackerState, err := s.repo.GetPvpPlayerState(playerID, now)
	if err != nil {
		return PvpPlayerState{}, PvpPlayerState{}, err
	}
	defenderState, err := s.repo.GetPvpPlayerState(targetPlayerID, now)
	if err != nil {
		return PvpPlayerState{}, PvpPlayerState{}, err
	}
	if protected, _, _ := activePvpProtection(defenderState, now); protected {
		return PvpPlayerState{}, PvpPlayerState{}, ErrPvpTargetProtected
	}
	if attackerState.DailyAttackLimit > 0 && attackerState.DailyAttackCount >= attackerState.DailyAttackLimit {
		return PvpPlayerState{}, PvpPlayerState{}, ErrPvpDailyLimitReached
	}
	return attackerState, defenderState, nil
}

// applyPvpBattleStateEffects 根据战斗结果写保护期等 PVP 状态副作用。
func (s *Service) applyPvpBattleStateEffects(battle PvpBattle, now time.Time) {
	if battle.ID == "" || battle.Status != PvpBattleStatusResolved {
		return
	}
	winner, _ := battle.Result["winner"].(string)
	attackerDelta, defenderDelta := pvpPointDeltas(winner)
	attackerState, err := s.repo.GetPvpPlayerState(battle.AttackerPlayerID, now)
	if err == nil {
		applyPvpPointsToState(&attackerState, attackerDelta, winner == "attacker", false)
		if winner == "attacker" {
			closeSuccessfulPvpRevenge(&attackerState, battle.DefenderPlayerID, now)
		}
		_ = s.repo.SavePvpPlayerState(attackerState, now)
	}
	defenderState, err := s.repo.GetPvpPlayerState(battle.DefenderPlayerID, now)
	if err != nil {
		return
	}
	applyPvpPointsToState(&defenderState, defenderDelta, false, winner == "defender")
	upsertPvpRevengeRecord(&defenderState, battle.AttackerPlayerID, battle.MarchID, battle.ID, now)
	if winner == "attacker" {
		defenderState.Status = "protected"
		defenderState.ProtectionType = PvpProtectionTypeDefeat
		defenderState.ProtectedUntil = now.Add(defaultPvpDefeatProtectSec * time.Second).Format(resourceDateLayout)
	}
	defenderState.UpdatedAt = now.Format(resourceDateLayout)
	_ = s.repo.SavePvpPlayerState(defenderState, now)
}

const reinforcementGeneralExpAppliedBattlesKey = "generalExpAppliedBattles"

// applyPvpReinforcementGeneralExp 给参战援军携带武将补发守城杀敌经验。
func (s *Service) applyPvpReinforcementGeneralExp(battle PvpBattle, now time.Time) error {
	expByReinforcement := pvpBattleReinforcementGeneralExp(battle)
	if len(expByReinforcement) == 0 {
		return nil
	}
	for _, record := range battle.ReinforcementSnapshot {
		exp := expByReinforcement[record.ReinforcementID]
		if exp <= 0 || strings.TrimSpace(record.FromPlayerID) == "" || len(record.Generals) == 0 {
			continue
		}
		if err := s.applyReinforcementGeneralExp(record.ReinforcementID, record.FromPlayerID, battle.ID, record.Generals, exp, battle.Status == PvpBattleStatusResolved, now); err != nil {
			return err
		}
	}
	return nil
}

// applyReinforcementGeneralExp 原子发放协防经验，已结算恢复流程可在驻防记录清理后改用事件账本。
func (s *Service) applyReinforcementGeneralExp(reinforcementID string, playerID string, battleID string, generals []ReinforcementGeneralSnapshot, exp int, allowResolvedRecovery bool, now time.Time) error {
	if exp <= 0 || strings.TrimSpace(reinforcementID) == "" || strings.TrimSpace(playerID) == "" || strings.TrimSpace(battleID) == "" || len(generals) == 0 {
		return nil
	}
	generalIDs := make([]string, 0, len(generals))
	for _, general := range generals {
		if strings.TrimSpace(general.ID) != "" {
			generalIDs = append(generalIDs, general.ID)
		}
	}
	if len(generalIDs) == 0 {
		return nil
	}
	_, _, _, err := s.repo.UpdateReinforcement(reinforcementID, now, func(from *GameState, _ *GameState, record *Reinforcement) error {
		if strings.TrimSpace(record.FromPlayerID) != strings.TrimSpace(playerID) {
			return ErrReinforcementNotFound
		}
		if reinforcementGeneralExpWasApplied(record.RewardState, battleID) {
			return nil
		}
		EnsureGeneralRoster(from, now)
		applyGeneralBattleExpToRoster(from, generalIDs, exp)
		refreshGeneralDerivedState(from, now)
		for index := range record.Generals {
			if current, ok := findOwnedGeneral(from.Generals, record.Generals[index].ID); ok {
				record.Generals[index].Level = current.Level
				record.Generals[index].Exp = current.Exp
				record.Generals[index].GeneralExpGained = nil
				record.Generals[index].GeneralLevelBefore = nil
				record.Generals[index].GeneralLevelAfter = nil
			}
		}
		record.RewardState = markReinforcementGeneralExpApplied(record.RewardState, battleID)
		record.UpdatedAt = now.UTC().Format(resourceDateLayout)
		return nil
	})
	if allowResolvedRecovery && errors.Is(err, ErrReinforcementNotFound) {
		_, err = s.repo.ApplyGeneralExpEvent(playerID, battleID+"|"+reinforcementID, now, func(state *GameState) error {
			EnsureGeneralRoster(state, now)
			applyGeneralBattleExpToRoster(state, generalIDs, exp)
			refreshGeneralDerivedState(state, now)
			return nil
		})
	}
	return err
}

// reinforcementGeneralExpWasApplied 判断指定战斗的援军武将经验是否已原子发放。
func reinforcementGeneralExpWasApplied(state map[string]any, battleID string) bool {
	if strings.TrimSpace(battleID) == "" {
		return false
	}
	raw, ok := state[reinforcementGeneralExpAppliedBattlesKey]
	if !ok {
		return false
	}
	switch applied := raw.(type) {
	case map[string]any:
		value, exists := applied[battleID]
		flag, _ := value.(bool)
		return exists && flag
	case map[string]bool:
		return applied[battleID]
	default:
		return false
	}
}

// markReinforcementGeneralExpApplied 记录已发奖战斗，供并发重试在同一事务内去重。
func markReinforcementGeneralExpApplied(state map[string]any, battleID string) map[string]any {
	next := cloneAnyMap(state)
	applied := map[string]any{}
	switch current := next[reinforcementGeneralExpAppliedBattlesKey].(type) {
	case map[string]any:
		for id, value := range current {
			applied[id] = value
		}
	case map[string]bool:
		for id, value := range current {
			applied[id] = value
		}
	}
	applied[battleID] = true
	next[reinforcementGeneralExpAppliedBattlesKey] = applied
	return next
}

// reinforcementReportOwnerSnapshot 从完整参战列表中定位当前协防战报真正拥有的援军批次。
func reinforcementReportOwnerSnapshot(report BattleReport) (DefenseReinforcementUnit, bool) {
	ownerPlayerID := strings.TrimSpace(valueOrDefault(report.OwnerPlayerID, report.PlayerID))
	reinforcementID := ""
	if report.Detail != nil {
		if context, ok := report.Detail.Extra["reinforcement"].(map[string]interface{}); ok {
			reinforcementID, _ = context["reinforcementId"].(string)
			reinforcementID = strings.TrimSpace(reinforcementID)
		}
	}
	matches := []DefenseReinforcementUnit{}
	for _, reinforcement := range report.PvpReinforcements {
		if ownerPlayerID != "" && strings.TrimSpace(reinforcement.FromPlayerID) != ownerPlayerID {
			continue
		}
		if reinforcementID != "" && strings.TrimSpace(reinforcement.ReinforcementID) != reinforcementID {
			continue
		}
		matches = append(matches, reinforcement)
	}
	if len(matches) != 1 {
		return DefenseReinforcementUnit{}, false
	}
	return matches[0], true
}

// applyReinforcementGeneralExpFromReports 根据每份协防战报的拥有者批次补发携带武将经验。
func (s *Service) applyReinforcementGeneralExpFromReports(reports []BattleReport, now time.Time) error {
	for _, report := range reports {
		if report.ViewType != ReportViewReinforcement || report.GeneralExpGained <= 0 {
			continue
		}
		reinforcement, ok := reinforcementReportOwnerSnapshot(report)
		if !ok {
			continue
		}
		battleID := strings.TrimSpace(report.EventID)
		if battleID == "" {
			battleID = strings.TrimSpace(report.ID)
		}
		if err := s.applyReinforcementGeneralExp(reinforcement.ReinforcementID, reinforcement.FromPlayerID, battleID, reinforcement.Generals, report.GeneralExpGained, false, now); err != nil {
			return err
		}
	}
	return nil
}

// pvpBattleReinforcementGeneralExp 从战斗结果中读取每支援军应获得的武将经验。
func pvpBattleReinforcementGeneralExp(battle PvpBattle) map[string]int {
	raw, ok := battle.Result["reinforcementGeneralExp"]
	if !ok {
		return nil
	}
	switch value := raw.(type) {
	case map[string]int:
		return cloneStringIntMap(value)
	case map[string]any:
		result := map[string]int{}
		for key, item := range value {
			switch typed := item.(type) {
			case int:
				result[key] = typed
			case float64:
				result[key] = int(typed)
			}
		}
		return result
	default:
		return nil
	}
}

// newDefaultPvpPlayerState 创建默认 PVP 状态。
func newDefaultPvpPlayerState(playerID string, now time.Time) PvpPlayerState {
	nowText := now.UTC().Format(resourceDateLayout)
	resetAt := time.Date(now.UTC().Year(), now.UTC().Month(), now.UTC().Day()+1, 0, 0, 0, 0, time.UTC)
	return PvpPlayerState{
		PlayerID:         strings.TrimSpace(playerID),
		Status:           "normal",
		DailyAttackLimit: defaultPvpDailyAttackLimit,
		DailyResetAt:     resetAt.Format(resourceDateLayout),
		TargetCooldown:   map[string]string{},
		Metadata:         map[string]any{},
		CreatedAt:        nowText,
		UpdatedAt:        nowText,
	}
}

// NewDefaultPvpPlayerStateForStorage 供基础设施层初始化默认 PVP 状态。
func NewDefaultPvpPlayerStateForStorage(playerID string, now time.Time) PvpPlayerState {
	return newDefaultPvpPlayerState(playerID, now)
}

// normalizePvpPlayerState 归一化每日重置、过期保护和默认上限。
func normalizePvpPlayerState(state PvpPlayerState, now time.Time) PvpPlayerState {
	if state.PlayerID == "" {
		return state
	}
	if state.DailyAttackLimit <= 0 {
		state.DailyAttackLimit = defaultPvpDailyAttackLimit
	}
	resetAt, err := time.Parse(resourceDateLayout, strings.TrimSpace(state.DailyResetAt))
	if err != nil || !resetAt.After(now.UTC()) {
		state.DailyAttackCount = 0
		nextReset := time.Date(now.UTC().Year(), now.UTC().Month(), now.UTC().Day()+1, 0, 0, 0, 0, time.UTC)
		state.DailyResetAt = nextReset.Format(resourceDateLayout)
	}
	if state.TargetCooldown == nil {
		state.TargetCooldown = map[string]string{}
	}
	if state.Metadata == nil {
		state.Metadata = map[string]any{}
	}
	if !pvpTimeAfter(state.ProtectedUntil, now) {
		state.ProtectionType = ""
		state.ProtectedUntil = ""
	}
	if pvpTimeAfter(state.ProtectedUntil, now) {
		state.Status = "protected"
	} else {
		state.Status = "normal"
	}
	if state.CreatedAt == "" {
		state.CreatedAt = now.UTC().Format(resourceDateLayout)
	}
	state.UpdatedAt = now.UTC().Format(resourceDateLayout)
	return state
}

// NormalizePvpPlayerStateForStorage 供基础设施层复用 PVP 状态归一化规则。
func NormalizePvpPlayerStateForStorage(state PvpPlayerState, now time.Time) PvpPlayerState {
	return normalizePvpPlayerState(state, now)
}

func pvpTimeAfter(value string, now time.Time) bool {
	parsed, err := time.Parse(resourceDateLayout, strings.TrimSpace(value))
	return err == nil && parsed.After(now.UTC())
}

// activePvpProtection 返回当前有效 PVP 保护信息。
func activePvpProtection(state PvpPlayerState, now time.Time) (bool, string, string) {
	if !pvpTimeAfter(state.ProtectedUntil, now) {
		return false, "", ""
	}
	protectionType := normalizePvpProtectionType(state.ProtectionType)
	if protectionType == "" {
		protectionType = PvpProtectionTypeSystem
	}
	return true, protectionType, state.ProtectedUntil
}

// normalizePvpProtectionType 归一化 PVP 保护类型。
func normalizePvpProtectionType(protectionType string) string {
	switch strings.TrimSpace(protectionType) {
	case PvpProtectionTypeNewbie:
		return PvpProtectionTypeNewbie
	case PvpProtectionTypeDefeat:
		return PvpProtectionTypeDefeat
	case PvpProtectionTypeManual:
		return PvpProtectionTypeManual
	case PvpProtectionTypeSystem:
		return PvpProtectionTypeSystem
	case PvpProtectionTypeMaintenance:
		return PvpProtectionTypeMaintenance
	default:
		return ""
	}
}

// pvpProtectionReason 返回前端目标列表可读的保护原因。
func pvpProtectionReason(protectionType string) string {
	switch normalizePvpProtectionType(protectionType) {
	case PvpProtectionTypeManual:
		return "目标处于免战保护"
	case PvpProtectionTypeMaintenance:
		return "目标处于维护保护"
	case PvpProtectionTypeSystem:
		return "目标处于系统保护"
	case PvpProtectionTypeNewbie:
		return "目标处于新手保护"
	default:
		return "目标处于保护期"
	}
}

// clearBreakablePvpProtectionOnAttack 主动攻击会打破普通保护，但不打破系统和维护保护。
func clearBreakablePvpProtectionOnAttack(state *PvpPlayerState, now time.Time) {
	if state == nil {
		return
	}
	protected, protectionType, _ := activePvpProtection(*state, now)
	if !protected {
		return
	}
	switch protectionType {
	case PvpProtectionTypeManual, PvpProtectionTypeNewbie, PvpProtectionTypeDefeat:
		state.ProtectionType = ""
		state.ProtectedUntil = ""
		if state.Metadata != nil {
			state.Metadata["protectionBrokenAt"] = now.UTC().Format(resourceDateLayout)
		}
	}
}

// shouldOverridePvpProtection 判断新保护是否可以覆盖当前保护。
func shouldOverridePvpProtection(currentType string, currentUntil string, nextType string, nextUntil string) bool {
	currentPriority := pvpProtectionPriority(currentType)
	nextPriority := pvpProtectionPriority(nextType)
	if nextPriority > currentPriority {
		return true
	}
	if nextPriority < currentPriority {
		return false
	}
	return nextUntil > currentUntil
}

// pvpProtectionPriority 返回保护类型优先级，维护和系统保护不能被普通免战削弱。
func pvpProtectionPriority(protectionType string) int {
	switch normalizePvpProtectionType(protectionType) {
	case PvpProtectionTypeMaintenance:
		return 4
	case PvpProtectionTypeSystem:
		return 3
	case PvpProtectionTypeManual:
		return 2
	case PvpProtectionTypeNewbie, PvpProtectionTypeDefeat:
		return 1
	default:
		return 0
	}
}

// buildPvpRankings 从玩家 PVP 状态构建第一版默认赛季排行榜。
func (s *Service) buildPvpRankings(now time.Time) ([]PvpRankingEntry, error) {
	accounts, err := s.repo.ListAccounts()
	if err != nil {
		return nil, err
	}
	items := []PvpRankingEntry{}
	for _, account := range accounts {
		for _, player := range account.Players {
			state, err := s.repo.GetPvpPlayerState(player.ID, now)
			if err != nil {
				continue
			}
			items = append(items, PvpRankingEntry{
				PlayerID:    player.ID,
				Nickname:    player.Nickname,
				Faction:     player.Faction,
				Points:      pvpMetadataInt(state.Metadata, "seasonPoints"),
				Rating:      pvpMetadataInt(state.Metadata, "rating"),
				AttackWins:  pvpMetadataInt(state.Metadata, "attackWins"),
				DefenseWins: pvpMetadataInt(state.Metadata, "defenseWins"),
				Losses:      pvpMetadataInt(state.Metadata, "losses"),
				UpdatedAt:   state.UpdatedAt,
			})
		}
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].Points != items[j].Points {
			return items[i].Points > items[j].Points
		}
		if items[i].Rating != items[j].Rating {
			return items[i].Rating > items[j].Rating
		}
		return items[i].PlayerID < items[j].PlayerID
	})
	for i := range items {
		items[i].Rank = i + 1
	}
	return items, nil
}

// ensureCurrentPvpSeason 确保当前自然月 PVP 赛季存在。
func (s *Service) ensureCurrentPvpSeason(now time.Time) (PvpSeasonRecord, error) {
	season, err := s.repo.GetCurrentPvpSeason(now)
	if err == nil {
		return season, nil
	}
	if !errors.Is(err, ErrPlayerNotFound) {
		return PvpSeasonRecord{}, err
	}
	season = defaultPvpSeasonRecord(now)
	seasons, listErr := s.repo.ListPvpSeasons()
	if listErr != nil {
		return PvpSeasonRecord{}, listErr
	}
	for _, item := range seasons {
		if item.ID == season.ID {
			return item, nil
		}
	}
	if err := s.repo.SavePvpSeason(season, now.UTC()); err != nil {
		return PvpSeasonRecord{}, err
	}
	return season, nil
}

// defaultPvpSeasonRecord 构造当前自然月默认 PVP 赛季。
func defaultPvpSeasonRecord(now time.Time) PvpSeasonRecord {
	year, month, _ := now.UTC().Date()
	startsAt := time.Date(year, month, 1, 0, 0, 0, 0, time.UTC)
	endsAt := startsAt.AddDate(0, 1, 0)
	nowText := now.UTC().Format(resourceDateLayout)
	return PvpSeasonRecord{
		ID:       startsAt.Format("2006-01"),
		Name:     startsAt.Format("2006-01") + " PVP 赛季",
		Status:   PvpSeasonStatusActive,
		StartsAt: startsAt.Format(resourceDateLayout),
		EndsAt:   endsAt.Format(resourceDateLayout),
		Rewards: map[string]any{
			"rank1CityGold":  1000,
			"rank3CityGold":  500,
			"rank10CityGold": 200,
		},
		CreatedAt: nowText,
		UpdatedAt: nowText,
	}
}

// buildAdminPvpSeasonRecord 校验 GM 请求并生成赛季记录。
func buildAdminPvpSeasonRecord(req AdminSavePvpSeasonRequest, now time.Time) (PvpSeasonRecord, error) {
	name := strings.TrimSpace(req.Name)
	startsAt, err := parsePvpSeasonTime(req.StartsAt)
	if err != nil {
		return PvpSeasonRecord{}, ErrInvalidPvpSeason
	}
	endsAt, err := parsePvpSeasonTime(req.EndsAt)
	if err != nil {
		return PvpSeasonRecord{}, ErrInvalidPvpSeason
	}
	if name == "" || !startsAt.Before(endsAt) {
		return PvpSeasonRecord{}, ErrInvalidPvpSeason
	}
	status := normalizePvpSeasonStatus(req.Status)
	nowText := now.UTC().Format(resourceDateLayout)
	return PvpSeasonRecord{
		ID:        strings.TrimSpace(req.ID),
		Name:      name,
		Status:    status,
		StartsAt:  startsAt.UTC().Format(resourceDateLayout),
		EndsAt:    endsAt.UTC().Format(resourceDateLayout),
		Rules:     cloneAnyMap(req.Rules),
		Rewards:   cloneAnyMap(req.Rewards),
		CreatedAt: nowText,
		UpdatedAt: nowText,
	}, nil
}

// parsePvpSeasonTime 解析 GM 传入的赛季时间。
func parsePvpSeasonTime(value string) (time.Time, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}, ErrInvalidPvpSeason
	}
	parsed, err := time.Parse(time.RFC3339, value)
	if err == nil {
		return parsed.UTC(), nil
	}
	parsed, err = time.Parse("2006-01-02", value)
	if err == nil {
		return parsed.UTC(), nil
	}
	return time.Time{}, ErrInvalidPvpSeason
}

// normalizePvpSeasonStatus 归一化 GM 赛季状态。
func normalizePvpSeasonStatus(status string) string {
	switch strings.TrimSpace(status) {
	case PvpSeasonStatusSettled:
		return PvpSeasonStatusSettled
	default:
		return PvpSeasonStatusActive
	}
}

// findPvpSeasonByID 从仓储列表中查找指定赛季。
func (s *Service) findPvpSeasonByID(seasonID string) (PvpSeasonRecord, error) {
	seasons, err := s.repo.ListPvpSeasons()
	if err != nil {
		return PvpSeasonRecord{}, err
	}
	for _, season := range seasons {
		if season.ID == seasonID {
			return season, nil
		}
	}
	return PvpSeasonRecord{}, ErrPlayerNotFound
}

// pvpSeasonSummaryFromRecord 转换赛季记录为玩家和 GM 通用摘要。
func pvpSeasonSummaryFromRecord(season PvpSeasonRecord) PvpSeasonSummary {
	return PvpSeasonSummary{
		ID:        season.ID,
		Name:      season.Name,
		Status:    season.Status,
		StartsAt:  season.StartsAt,
		EndsAt:    season.EndsAt,
		UpdatedAt: season.UpdatedAt,
	}
}

// currentPvpSeasonSummary 返回当前自然月默认赛季摘要，兼容旧测试和调用。
func currentPvpSeasonSummary(now time.Time) PvpSeasonSummary {
	return pvpSeasonSummaryFromRecord(defaultPvpSeasonRecord(now))
}

// pvpSeasonRewardCityGold 返回第一版赛季排名奖励城金。
func pvpSeasonRewardCityGold(rank int) int {
	switch {
	case rank == 1:
		return 1000
	case rank >= 2 && rank <= 3:
		return 500
	case rank >= 4 && rank <= 10:
		return 200
	default:
		return 0
	}
}

// pvpPointDeltas 返回第一版 PVP 胜负积分变化。
func pvpPointDeltas(winner string) (int, int) {
	switch strings.TrimSpace(winner) {
	case "attacker":
		return 10, -5
	case "defender":
		return -3, 8
	default:
		return 1, 1
	}
}

// applyPvpPointsToState 写入玩家积分和胜负统计。
func applyPvpPointsToState(state *PvpPlayerState, delta int, attackWin bool, defenseWin bool) {
	if state == nil {
		return
	}
	if state.Metadata == nil {
		state.Metadata = map[string]any{}
	}
	points := pvpMetadataInt(state.Metadata, "seasonPoints") + delta
	if points < 0 {
		points = 0
	}
	state.Metadata["seasonPoints"] = points
	state.Metadata["rating"] = 1000 + points
	if attackWin {
		state.Metadata["attackWins"] = pvpMetadataInt(state.Metadata, "attackWins") + 1
	} else if defenseWin {
		state.Metadata["defenseWins"] = pvpMetadataInt(state.Metadata, "defenseWins") + 1
	} else if delta < 0 {
		state.Metadata["losses"] = pvpMetadataInt(state.Metadata, "losses") + 1
	}
}

// pvpMetadataInt 从 PVP metadata 中兼容读取整数。
func pvpMetadataInt(metadata map[string]any, key string) int {
	if metadata == nil {
		return 0
	}
	switch value := metadata[key].(type) {
	case int:
		return value
	case int64:
		return int(value)
	case float64:
		return int(value)
	case json.Number:
		parsed, _ := value.Int64()
		return int(parsed)
	default:
		return 0
	}
}

// upsertPvpRevengeRecord 为防守方写入一条复仇记录。
func upsertPvpRevengeRecord(state *PvpPlayerState, attackerPlayerID string, marchID string, battleID string, now time.Time) {
	if state == nil {
		return
	}
	if state.Metadata == nil {
		state.Metadata = map[string]any{}
	}
	records := filterActivePvpRevengeRecords(pvpRevengeRecordsFromMetadata(state.Metadata), now)
	for i := range records {
		if records[i].MarchID == marchID {
			records[i].BattleID = battleID
			state.Metadata["revengeRecords"] = records
			return
		}
	}
	nowText := now.UTC().Format(resourceDateLayout)
	records = append(records, PvpRevengeRecord{
		ID:               "pvp_revenge_" + randomID(10),
		DefenderPlayerID: state.PlayerID,
		AttackerPlayerID: strings.TrimSpace(attackerPlayerID),
		MarchID:          strings.TrimSpace(marchID),
		BattleID:         strings.TrimSpace(battleID),
		Status:           "open",
		CreatedAt:        nowText,
		ExpiresAt:        now.Add(defaultPvpRevengeExpireSec * time.Second).UTC().Format(resourceDateLayout),
	})
	state.Metadata["revengeRecords"] = records
}

// closeSuccessfulPvpRevenge 复仇攻击成功后关闭对应记录。
func closeSuccessfulPvpRevenge(state *PvpPlayerState, targetPlayerID string, now time.Time) {
	if state == nil || state.Metadata == nil {
		return
	}
	records := pvpRevengeRecordsFromMetadata(state.Metadata)
	changed := false
	for i := range records {
		if records[i].Status == "open" && records[i].AttackerPlayerID == strings.TrimSpace(targetPlayerID) {
			records[i].Status = "closed"
			records[i].ClosedAt = now.UTC().Format(resourceDateLayout)
			changed = true
		}
	}
	if changed {
		state.Metadata["revengeRecords"] = records
	}
}

// filterActivePvpRevengeRecords 过滤未过期且未关闭的复仇记录。
func filterActivePvpRevengeRecords(records []PvpRevengeRecord, now time.Time) []PvpRevengeRecord {
	result := make([]PvpRevengeRecord, 0, len(records))
	for _, record := range records {
		if record.Status != "open" {
			continue
		}
		if !pvpTimeAfter(record.ExpiresAt, now) {
			continue
		}
		result = append(result, record)
	}
	return result
}

// pvpRevengeRecordsFromMetadata 从 metadata 中解析复仇记录。
func pvpRevengeRecordsFromMetadata(metadata map[string]any) []PvpRevengeRecord {
	if metadata == nil || metadata["revengeRecords"] == nil {
		return []PvpRevengeRecord{}
	}
	content, err := json.Marshal(metadata["revengeRecords"])
	if err != nil {
		return []PvpRevengeRecord{}
	}
	var records []PvpRevengeRecord
	if err := json.Unmarshal(content, &records); err != nil {
		return []PvpRevengeRecord{}
	}
	return records
}

// combatSceneForPVP 根据 PVP 行军模式选择核心战斗规则场景。
func combatSceneForPVP(mode string) string {
	if strings.TrimSpace(mode) == PvpMarchTypePlunder {
		return combat.ScenePVPPlunder
	}
	return combat.ScenePVPAttack
}

// calculatePvpMarchTravel 按出征队伍最慢兵种速度和行军加成计算 PVP 单程行军时间。
func calculatePvpMarchTravel(distance int, preferredFaction string, troops map[string]int, now time.Time, sources []ModifierSource) (int, float64, error) {
	if len(troops) == 0 {
		return 0, 0, ErrNoUnitsSelected
	}
	minSpeed := 0
	for unitType, amount := range troops {
		if amount <= 0 {
			continue
		}
		unitCfg, _, ok := findAnyUnitConfig(preferredFaction, unitType)
		if !ok {
			return 0, 0, ErrUnitNotFound
		}
		if isNonCombatUnit(unitCfg) {
			return 0, 0, ErrNonCombatUnit
		}
		speed := unitCfg.Stats["speed"]
		if speed <= 0 {
			speed = 1
		}
		speed += ComputeIntAttributeAt(0, unitSpeedFlatModifierKey(unitType), now, sources...)
		if minSpeed == 0 || speed < minSpeed {
			minSpeed = speed
		}
	}
	if minSpeed <= 0 {
		return 0, 0, ErrNoUnitsSelected
	}
	seconds := CalculateWorldMarchSeconds(distance, minSpeed, now, sources)
	if seconds <= 0 {
		seconds = 1
	}
	speedMultiplier := float64(defaultPvpMarchSeconds) / float64(seconds)
	return seconds, speedMultiplier, nil
}

// calculatePvpRecallReturnSeconds 按已经行军的距离计算召回返程时间。
func calculatePvpRecallReturnSeconds(march *PvpMarch, now time.Time) int {
	if march == nil {
		return 1
	}
	durationSeconds := march.DurationSeconds
	if durationSeconds <= 0 {
		durationSeconds = defaultPvpMarchSeconds
	}
	startedAt, err := time.Parse(resourceDateLayout, strings.TrimSpace(march.StartedAt))
	if err != nil {
		return durationSeconds
	}
	elapsedSeconds := int(math.Ceil(now.Sub(startedAt).Seconds()))
	if elapsedSeconds <= 0 {
		return 1
	}
	if elapsedSeconds > durationSeconds {
		return durationSeconds
	}
	return elapsedSeconds
}

// calculatePvpOutboundTravelSeconds 返回本次出征从开始到实际抵达花费的秒数，用于战后返程对称。
func calculatePvpOutboundTravelSeconds(march *PvpMarch) int {
	if march == nil {
		return defaultPvpMarchSeconds
	}
	startedAt, errStart := time.Parse(resourceDateLayout, strings.TrimSpace(march.StartedAt))
	arrivesAt, errArrive := time.Parse(resourceDateLayout, strings.TrimSpace(march.ArrivesAt))
	if errStart == nil && errArrive == nil && arrivesAt.After(startedAt) {
		seconds := int(math.Ceil(arrivesAt.Sub(startedAt).Seconds()))
		if seconds > 0 {
			return seconds
		}
	}
	if march.DurationSeconds > 0 {
		return march.DurationSeconds
	}
	return defaultPvpMarchSeconds
}

// calculatePvpSpeedMultiplier 根据当前到达时间反推展示用行军速度倍率。
func calculatePvpSpeedMultiplier(march *PvpMarch) float64 {
	if march == nil || march.DurationSeconds <= 0 {
		return 1
	}
	startedAt, errStart := time.Parse(resourceDateLayout, strings.TrimSpace(march.StartedAt))
	arrivesAt, errArrive := time.Parse(resourceDateLayout, strings.TrimSpace(march.ArrivesAt))
	if errStart != nil || errArrive != nil || !arrivesAt.After(startedAt) {
		return 1
	}
	actualSeconds := arrivesAt.Sub(startedAt).Seconds()
	if actualSeconds <= 0 {
		return 1
	}
	multiplier := float64(march.DurationSeconds) / actualSeconds
	if multiplier < 1 {
		return 1
	}
	return multiplier
}

// reservePvpGenerals 占用 PVP 出征武将。
func reservePvpGenerals(state *GameState, generalIDs []string, marchID string, now time.Time) ([]string, error) {
	result := []string{}
	seen := map[string]bool{}
	for _, generalID := range generalIDs {
		generalID = strings.TrimSpace(generalID)
		if generalID == "" || seen[generalID] {
			continue
		}
		seen[generalID] = true
		if _, ok := findOwnedGeneral(state.Generals, generalID); !ok {
			return nil, ErrGeneralNotFound
		}
		if !generalAvailableForReinforcement(state.GeneralAssignments, generalID) {
			return nil, ErrGeneralBusy
		}
		state.GeneralAssignments = append(state.GeneralAssignments, GeneralAssignment{
			ID:         marchID + "_" + generalID,
			GeneralID:  generalID,
			Slot:       PVPModuleID,
			ModuleID:   PVPModuleID,
			Status:     PvpMarchStatusMarching,
			AssignedAt: now.UTC().Format(resourceDateLayout),
		})
		result = append(result, generalID)
		if len(result) > 1 {
			return nil, ErrInvalidGeneral
		}
	}
	return result, nil
}

// releasePvpGenerals 释放 PVP 出征武将占用。
func releasePvpGenerals(state *GameState, march *PvpMarch) {
	if len(march.AttackGenerals) == 0 || len(state.GeneralAssignments) == 0 {
		return
	}
	next := state.GeneralAssignments[:0]
	prefix := march.ID + "_"
	for _, assignment := range state.GeneralAssignments {
		if assignment.ModuleID == PVPModuleID && strings.HasPrefix(assignment.ID, prefix) {
			continue
		}
		next = append(next, assignment)
	}
	state.GeneralAssignments = next
}

// updatePvpGeneralAssignmentStatus 更新 PVP 行军携带武将占用状态。
func updatePvpGeneralAssignmentStatus(state *GameState, march *PvpMarch, status string) {
	if len(march.AttackGenerals) == 0 || len(state.GeneralAssignments) == 0 {
		return
	}
	prefix := march.ID + "_"
	for i := range state.GeneralAssignments {
		if state.GeneralAssignments[i].ModuleID == PVPModuleID && strings.HasPrefix(state.GeneralAssignments[i].ID, prefix) {
			state.GeneralAssignments[i].Status = status
		}
	}
}

// resolvePvpCombat 调用核心战斗引擎并回写双方和援军损耗。
func resolvePvpCombat(attacker *GameState, defender *GameState, reinforcements []Reinforcement, march *PvpMarch, now time.Time) (PvpBattle, BattleReport, BattleReport, []BattleReport, []Reinforcement, error) {
	dispatchedTroops := cloneStringIntMap(march.AttackTroops)
	attackerUnits, err := buildSimulatedCombatUnits(attacker.Player.Faction, march.AttackTroops, now, pvpModifierSourcesForGenerals(attacker, march.AttackGenerals)...)
	if err != nil {
		return PvpBattle{}, BattleReport{}, BattleReport{}, nil, nil, err
	}
	attackerTraits := buildActiveTraitsForGeneralIDs(attacker, march.AttackGenerals)
	defenderGeneralIDs := pvpDefenseGeneralIDs(defender)
	defenderTraits := buildActiveTraitsForGeneralIDs(defender, defenderGeneralIDs)
	traitControl := resolveBattleTraitControl(attackerTraits, defenderTraits, reinforcements, march.MarchType)
	attackerTraits = filterDisabledCombatTraits(attackerTraits, traitControl.DisabledSides["attacker"])
	defenderTraits = filterDisabledCombatTraits(defenderTraits, traitControl.DisabledSides["defender"])
	defenderOwnTroops := armySliceToMap(defender.Army)
	defenderUnits, sourceGroups, reinforcementBeforeOutcomes, err := buildPvpDefenderUnits(defender, reinforcements, now, march.MarchType, traitControl.DisabledSides["defender"])
	if errors.Is(err, ErrNoUnitsSelected) {
		defenderUnits = []combat.Unit{}
		sourceGroups = []pvpDefenseSourceGroup{}
	} else if err != nil {
		return PvpBattle{}, BattleReport{}, BattleReport{}, nil, nil, err
	}
	attackerArmy := buildCombatArmy(attacker.Player.Faction, attackerUnits)
	defenderArmy := combat.Army{Faction: defender.Player.Faction, Units: append([]combat.Unit(nil), defenderUnits...)}
	wallLevel := pvpDefenderCityWallLevel(defender)
	wallSnapshot := buildPvpWallSnapshot(defender.Player.Faction, wallLevel)
	reinforcementBeforeOutcomes = mergeReinforcementTraitOutcomeMaps(reinforcementBeforeOutcomes, traitControl.ReinforcementOutcomes)
	beforeCtx := &general.BeforeBattleContext{
		Attacker:          &attackerArmy,
		Defender:          &defenderArmy,
		AttackerOwnsTrait: true,
		DefenderOwnsTrait: false,
		IsPvP:             true,
		SameFaction:       attacker.Player.Faction == defender.Player.Faction,
		Scene:             march.MarchType,
	}
	general.Dispatch(beforeCtx, attackerTraits)
	defenderBeforeCtx := &general.BeforeBattleContext{
		Attacker:          &attackerArmy,
		Defender:          &defenderArmy,
		AttackerOwnsTrait: false,
		DefenderOwnsTrait: true,
		IsPvP:             true,
		SameFaction:       attacker.Player.Faction == defender.Player.Faction,
		Scene:             march.MarchType,
	}
	general.Dispatch(defenderBeforeCtx, defenderTraits)
	reinforcementEnemyBeforeOutcomes, reinforcementEnemyBeforeContexts := applyReinforcementEnemyBeforeBattleTraits(reinforcements, &attackerArmy, &defenderArmy, march.MarchType, traitControl.DisabledSides["defender"])
	reinforcementBeforeOutcomes = mergeReinforcementTraitOutcomeMaps(reinforcementBeforeOutcomes, reinforcementEnemyBeforeOutcomes)
	capturedDefenderLosses := map[string]int{}
	capturedReinforcementLosses := map[string]map[string]int{}
	capturedUnits := mergeTroopMaps(beforeCtx.CapturedToArmy, beforeCtx.CapturedToGarrison)
	if len(capturedUnits) > 0 {
		capturedDefenderLosses, capturedReinforcementLosses = allocatePvpDefenderLosses(capturedUnits, sourceGroups)
		sourceGroups = reducePvpSourceGroups(sourceGroups, capturedDefenderLosses, capturedReinforcementLosses)
		applyArmyLosses(defender, capturedDefenderLosses)
	}
	result := combat.Resolve(combat.CombatInput{
		RuleID:      activeCombatRuleID(combatSceneForPVP(march.MarchType)),
		Attacker:    attackerArmy,
		Defender:    defenderArmy,
		WallLevel:   wallLevel,
		WallFaction: defender.Player.Faction,
	})
	preBattleContexts := []*general.BeforeBattleContext{beforeCtx, defenderBeforeCtx}
	preBattleContexts = append(preBattleContexts, reinforcementEnemyBeforeContexts...)
	applyPreBattleLossesToCombatResult(&result, preBattleContexts...)
	afterCombatCtx := &general.AfterCombatResolveContext{
		Result:            &result,
		Attacker:          &attackerArmy,
		Defender:          &defenderArmy,
		AttackerOwnsTrait: true,
		DefenderOwnsTrait: true,
		IsAttackerOnly:    true,
		Scene:             march.MarchType,
	}
	sharedDefenderTraits, sourceDefenderTraits := partitionMainDefenderSourceTraits(defenderTraits)
	combinedAfterCombatTraits := append(withTraitOwnerSide(attackerTraits, "attacker"), withTraitOwnerSide(sharedDefenderTraits, "defender")...)
	combinedAfterCombatTraits = append(combinedAfterCombatTraits, activeReinforcementEnemyTraits(reinforcements, traitControl.DisabledSides["defender"])...)
	general.Dispatch(afterCombatCtx, combinedAfterCombatTraits)
	attackerLosses := combatLossMap(result.AttackerLosses)
	defenderTotalLosses := combatLossMap(result.DefenderLosses)
	attackerSurvivors := map[string]int{}
	for _, unit := range attackerUnits {
		survived := unit.Count - attackerLosses[unit.ID]
		if survived > 0 {
			attackerSurvivors[unit.ID] += survived
		}
	}
	for unitType, amount := range beforeCtx.CapturedToArmy {
		if amount > 0 {
			attackerSurvivors[unitType] += amount
		}
	}
	march.AttackTroops = normalizePositiveTroops(attackerSurvivors)
	defenderLosses, reinforcementLosses := allocatePvpDefenderLosses(defenderTotalLosses, sourceGroups)
	defenderLosses = applyMainDefenderAfterCombatTraits(defenderOwnTroops, defenderLosses, defender.Player.Faction, sourceDefenderTraits, afterCombatCtx)
	applyArmyLosses(defender, defenderLosses)
	reinforcementResolution := resolveReinforcementAfterBattleTraits(reinforcements, reinforcementLosses, result.Winner, march.MarchType, afterCombatCtx.DisabledTraitSide["defender"], traitControl.DisabledSides["defender"])
	totalReinforcementLosses := mergeNestedStringIntMap(capturedReinforcementLosses, reinforcementResolution.FinalLosses)
	reinforcementAfterOutcomes := reinforcementResolution.Outcomes
	reinforcementSelfAfterCombatOutcomes := reinforcementResolution.AfterCombatOutcomes
	reinforcementAfterBattleOutcomes := reinforcementResolution.AfterBattleOutcomes
	reinforcementSuppressedTraits := reinforcementResolution.SuppressedCount
	if reinforcementSuppressedTraits > 0 {
		general.RecordActualTraitSuppressions(afterCombatCtx, "defender", reinforcementSuppressedTraits)
		afterCombatCtx.DisabledTraitSide["defender"] -= reinforcementSuppressedTraits
	}
	mainAfterCombatOutcomes, reinforcementEnemyAfterOutcomes := splitReinforcementTraitOutcomes(afterCombatCtx.Triggered, reinforcements)
	afterCombatCtx.Triggered = mainAfterCombatOutcomes
	reinforcementTraitOutcomes := mergeReinforcementTraitOutcomeMaps(nil, reinforcementBeforeOutcomes)
	reinforcementTraitOutcomes = mergeReinforcementTraitOutcomeMaps(reinforcementTraitOutcomes, reinforcementEnemyAfterOutcomes)
	reinforcementTraitOutcomes = mergeReinforcementTraitOutcomeMaps(reinforcementTraitOutcomes, reinforcementAfterOutcomes)
	changedReinforcements := applyPvpReinforcementLosses(reinforcements, totalReinforcementLosses, now)
	attackerAfterBattleCtx := &general.AfterBattleContext{
		PlayerArmy:   attackerSurvivors,
		PlayerLosses: attackerLosses,
		IsAttacker:   true,
		Won:          result.Winner == "attacker",
		Winner:       result.Winner,
		Scene:        march.MarchType,
	}
	general.Dispatch(attackerAfterBattleCtx, attackerTraits)
	if len(attackerAfterBattleCtx.Revived) > 0 {
		attackerSurvivors = attackerAfterBattleCtx.PlayerArmy
		march.AttackTroops = normalizePositiveTroops(attackerSurvivors)
	}
	defenderArmyMap := armySliceToMap(defender.Army)
	defenderAfterBattleCtx := &general.AfterBattleContext{
		PlayerArmy:   defenderArmyMap,
		PlayerLosses: defenderLosses,
		IsAttacker:   false,
		Won:          result.Winner == "defender",
		Winner:       result.Winner,
		Scene:        march.MarchType,
	}
	general.Dispatch(defenderAfterBattleCtx, defenderTraits)
	if len(defenderAfterBattleCtx.Revived) > 0 {
		defender.Army = armyMapToSlice(defenderArmyMap)
	}
	// 参战将领快照必须冻结在经验结算前，避免本场升级后的属性伪装成本场实际战斗属性。
	attackerGenerals := buildPvpGeneralSnapshots(attacker, march.AttackGenerals)
	defenderGenerals := buildPvpDefenseGeneralSnapshots(defender)
	defendingCombatLosses := combatLossesFromSources(defenderLosses, reinforcementResolution.AfterCombatLosses)
	attackerExp := calculateGeneralBattleExpFromLosses(defender.Player.Faction, defendingCombatLosses)
	defenderExp := calculateGeneralBattleExpFromLosses(attacker.Player.Faction, result.AttackerLosses)
	attackerExpResult := applyGeneralBattleExpToRoster(attacker, march.AttackGenerals, attackerExp)
	defenderExpResult := applyGeneralBattleExpToRoster(defender, defenderGeneralIDs, defenderExp)
	reinforcementGeneralExp := pvpReinforcementGeneralExpByID(reinforcements, defenderExp)
	defenderLossRatio := pvpDefenderLossRatio(sourceGroups, combatLossMap(defendingCombatLosses))
	revealThreshold := enemyLossRevealThreshold(defender, now)
	attackerSurvived := totalTroops(attackerSurvivors) > 0
	attackerCanSeeDefenderRemaining := attackerSurvived || defenderLossRatio >= revealThreshold
	plundered := map[string]int{}
	plunderOutcomes := map[string]general.TraitOutcome{}
	if march.MarchType == PvpMarchTypePlunder && result.Winner == "attacker" && result.SurvivingCarry > 0 {
		plundered = calculatePvpPlunder(defender, result.SurvivingCarry)
		if next, outcomes := dispatchPlunderTraits(attacker, march.AttackGenerals, plundered, march.MarchType, "attacker", traitControl.DisabledSides["attacker"]); len(outcomes) > 0 {
			plundered = next
			for k, v := range outcomes {
				plunderOutcomes[k] = v
			}
		} else {
			plundered = next
		}
		if next, outcomes := dispatchPlunderTraits(defender, defenderGeneralIDs, plundered, march.MarchType, "defender", traitControl.DisabledSides["defender"]); len(outcomes) > 0 {
			plundered = next
			for k, v := range outcomes {
				plunderOutcomes[k] = v
			}
		} else {
			plundered = next
		}
		plundered = applyPvpPlunder(defender, attacker, plundered)
	}
	nowText := now.Format(resourceDateLayout)
	battleID := "pvp_battle_" + randomID(12)
	attackerReportID := "br_pvp_atk_" + randomID(8)
	defenderReportID := "br_pvp_def_" + randomID(8)
	reportResult := "attacker_victory"
	if result.Winner == "defender" {
		reportResult = "defender_victory"
	} else if result.Winner == "draw" {
		reportResult = "draw"
	}
	attackerPointsDelta, defenderPointsDelta := pvpPointDeltas(result.Winner)
	reinforcementSnapshot := buildPvpReinforcementSnapshot(reinforcements, reinforcementGeneralExp)
	attackerReport := buildPvpBattleReport(attackerReportID, attacker, defender, march, reportResult, int(result.AttackPower), int(result.DefensePower), dispatchedTroops, attackerLosses, defenderOwnTroops, defenderLosses, plundered, nowText, march.MarchType)
	attackerReport.EventID = battleID
	attackerReport.ViewType = ReportViewAttack
	attackerReport.SourceType = ReportSourcePlayerCity
	attackerReport.BattleType = march.MarchType
	attackerReport.DefenderRevealed = attackerCanSeeDefenderRemaining
	attackerReport.EnemyLossRevealThreshold = revealThreshold
	attackerReport.EnemyLossRatio = defenderLossRatio
	attackerReport.PvpPointsDelta = map[string]int{"self": attackerPointsDelta, "target": defenderPointsDelta}
	attackerReport.PvpAttackerGenerals = attackerGenerals
	attackerReport.PvpDefenderGenerals = defenderGenerals
	attackerReport.PvpReinforcements = reinforcementSnapshot
	attackerReport.PvpReinforcementLosses = cloneNestedStringIntMap(totalReinforcementLosses)
	attackerReport.PvpWall = wallSnapshot
	attackerReport.SurvivedUnits = cloneStringIntMap(attackerSurvivors)
	attackerReport.CapturedUnits = cloneStringIntMap(beforeCtx.CapturedToArmy)
	attackerReport.CapturedToGarrison = cloneStringIntMap(beforeCtx.CapturedToGarrison)
	if len(attackerAfterBattleCtx.Revived) > 0 {
		attackerReport.RevivedUnits = cloneStringIntMap(attackerAfterBattleCtx.Revived)
	}
	mergeTraitOutcomes(&attackerReport, traitControl.MainOutcomes)
	mergeTraitOutcomes(&attackerReport, beforeCtx.Triggered)
	mergeTraitOutcomes(&attackerReport, defenderBeforeCtx.Triggered)
	mergeTraitOutcomes(&attackerReport, flattenReinforcementTraitOutcomes(reinforcementBeforeOutcomes))
	mergeTraitOutcomes(&attackerReport, afterCombatCtx.Triggered)
	mergeTraitOutcomes(&attackerReport, flattenReinforcementTraitOutcomes(reinforcementEnemyAfterOutcomes))
	mergeTraitOutcomes(&attackerReport, flattenReinforcementTraitOutcomes(reinforcementSelfAfterCombatOutcomes))
	mergeTraitOutcomes(&attackerReport, flattenReinforcementTraitOutcomes(reinforcementAfterBattleOutcomes))
	mergeTraitOutcomes(&attackerReport, attackerAfterBattleCtx.Triggered)
	mergeTraitOutcomes(&attackerReport, defenderAfterBattleCtx.Triggered)
	mergeTraitOutcomes(&attackerReport, plunderOutcomes)
	if attackerExpResult.Gained > 0 {
		attackerReport.GeneralExpGained = attackerExpResult.Gained
		attackerReport.GeneralLevelBefore = attackerExpResult.LevelBefore
		attackerReport.GeneralLevelAfter = attackerExpResult.LevelAfter
	}
	defenderReport := buildPvpBattleReport(defenderReportID, defender, attacker, march, invertPvpReportResult(reportResult), int(result.DefensePower), int(result.AttackPower), defenderOwnTroops, defenderLosses, dispatchedTroops, attackerLosses, map[string]int{}, nowText, "defense")
	defenderReport.EventID = battleID
	defenderReport.ViewType = ReportViewDefense
	defenderReport.SourceType = ReportSourcePlayerCity
	defenderReport.BattleType = march.MarchType
	defenderReport.DefenderRevealed = true
	defenderReport.EnemyLossRevealThreshold = revealThreshold
	defenderReport.EnemyLossRatio = defenderLossRatio
	defenderReport.PvpPointsDelta = map[string]int{"self": defenderPointsDelta, "target": attackerPointsDelta}
	defenderReport.PvpAttackerGenerals = attackerGenerals
	defenderReport.PvpDefenderGenerals = defenderGenerals
	defenderReport.PvpReinforcements = reinforcementSnapshot
	defenderReport.PvpReinforcementLosses = cloneNestedStringIntMap(totalReinforcementLosses)
	defenderReport.PvpWall = wallSnapshot
	defenderReport.SurvivedUnits = cloneStringIntMap(defenderArmyMap)
	defenderReport.CapturedUnits = cloneStringIntMap(beforeCtx.CapturedToArmy)
	defenderReport.CapturedToGarrison = cloneStringIntMap(beforeCtx.CapturedToGarrison)
	if len(defenderAfterBattleCtx.Revived) > 0 {
		defenderReport.RevivedUnits = cloneStringIntMap(defenderAfterBattleCtx.Revived)
	}
	mergeTraitOutcomes(&defenderReport, traitControl.MainOutcomes)
	mergeTraitOutcomes(&defenderReport, beforeCtx.Triggered)
	mergeTraitOutcomes(&defenderReport, defenderBeforeCtx.Triggered)
	mergeTraitOutcomes(&defenderReport, flattenReinforcementTraitOutcomes(reinforcementBeforeOutcomes))
	mergeTraitOutcomes(&defenderReport, afterCombatCtx.Triggered)
	mergeTraitOutcomes(&defenderReport, flattenReinforcementTraitOutcomes(reinforcementEnemyAfterOutcomes))
	mergeTraitOutcomes(&defenderReport, flattenReinforcementTraitOutcomes(reinforcementSelfAfterCombatOutcomes))
	mergeTraitOutcomes(&defenderReport, flattenReinforcementTraitOutcomes(reinforcementAfterBattleOutcomes))
	mergeTraitOutcomes(&defenderReport, attackerAfterBattleCtx.Triggered)
	mergeTraitOutcomes(&defenderReport, defenderAfterBattleCtx.Triggered)
	mergeTraitOutcomes(&defenderReport, plunderOutcomes)
	if defenderExpResult.Gained > 0 {
		defenderReport.GeneralExpGained = defenderExpResult.Gained
		defenderReport.GeneralLevelBefore = defenderExpResult.LevelBefore
		defenderReport.GeneralLevelAfter = defenderExpResult.LevelAfter
	}
	attackerReport = NormalizeBattleReport(attackerReport)
	defenderReport = NormalizeBattleReport(defenderReport)
	syncPvpReportFinalSurvivors(&attackerReport, attackerSurvivors, defenderArmyMap)
	syncPvpReportFinalSurvivors(&defenderReport, attackerSurvivors, defenderArmyMap)
	reinforcementReports := buildPvpReinforcementReportsByPhase(&defenderReport, battleID, attacker, defender, changedReinforcements, totalReinforcementLosses, reinforcementGeneralExp, reinforcementTraitReportPhases{
		Before: reinforcementBeforeOutcomes, EnemyAfterCombat: reinforcementEnemyAfterOutcomes,
		SelfAfterCombat: reinforcementSelfAfterCombatOutcomes, AfterBattle: reinforcementAfterBattleOutcomes, All: reinforcementTraitOutcomes,
	}, reportResult, nowText)
	eventReports := append([]BattleReport{attackerReport, defenderReport}, reinforcementReports...)
	eventReports = synchronizeBattleReportGeneralResults(eventReports)
	attackerReport = eventReports[0]
	defenderReport = eventReports[1]
	reinforcementReports = eventReports[2:]
	battle := PvpBattle{
		ID:                    battleID,
		MarchID:               march.ID,
		AttackerPlayerID:      attacker.Player.ID,
		DefenderPlayerID:      defender.Player.ID,
		Status:                PvpBattleStatusResolved,
		AttackerSnapshot:      map[string]any{"troops": cloneStringIntMap(dispatchedTroops), "faction": attacker.Player.Faction, "generals": attackerReport.PvpAttackerGenerals},
		DefenderSnapshot:      map[string]any{"troops": cloneStringIntMap(defenderOwnTroops), "faction": defender.Player.Faction, "generals": defenderReport.PvpDefenderGenerals},
		ReinforcementSnapshot: reinforcementSnapshot,
		Result:                map[string]any{"winner": result.Winner, "attackerPower": result.AttackPower, "defensePower": result.DefensePower, "pointsDelta": map[string]int{"attacker": attackerPointsDelta, "defender": defenderPointsDelta}, "reinforcementGeneralExp": reinforcementGeneralExp},
		Losses:                map[string]any{"attacker": attackerLosses, "defender": mergeTroopMaps(capturedDefenderLosses, defenderLosses), "reinforcements": totalReinforcementLosses},
		Plunder:               plundered,
		AttackerReportID:      attackerReportID,
		DefenderReportID:      defenderReportID,
		ResolvedAt:            nowText,
		CreatedAt:             nowText,
		UpdatedAt:             nowText,
	}
	return battle, attackerReport, defenderReport, reinforcementReports, changedReinforcements, nil
}

// syncPvpReportFinalSurvivors 用战后最终快照修正双方标准详情，且不为已隐藏的兵种补行。
func syncPvpReportFinalSurvivors(report *BattleReport, attackerSurvivors map[string]int, defenderSurvivors map[string]int) {
	if report == nil || report.Detail == nil {
		return
	}
	sides := []*BattleReportSide{&report.Detail.PrimarySide}
	if report.Detail.SecondarySide != nil {
		sides = append(sides, report.Detail.SecondarySide)
	}
	for _, side := range sides {
		finalUnits := defenderSurvivors
		if side.Role == "attacker" {
			finalUnits = attackerSurvivors
		}
		for index := range side.Units {
			side.Units[index].Survived = max(0, finalUnits[side.Units[index].UnitType])
		}
	}
}

// pvpDefenseGeneralIDs 返回本次 PVP 防守方参战主将 ID。
func pvpDefenseGeneralIDs(state *GameState) []string {
	general := pvpHomeDefenseGeneral(state)
	if general == nil || strings.TrimSpace(general.ID) == "" {
		return nil
	}
	return []string{general.ID}
}

// pvpHomeDefenseGeneral 返回真正留在城里的防守主将，出征或增援中的武将不参与守城。
func pvpHomeDefenseGeneral(state *GameState) *General {
	if state == nil || state.General == nil {
		return nil
	}
	generalID := strings.TrimSpace(state.General.ID)
	if !generalAvailableAtHome(state.GeneralAssignments, generalID) {
		return nil
	}
	if general, ok := findOwnedGeneral(state.Generals, generalID); ok {
		return cloneGeneralPtr(general)
	}
	return cloneGeneralPtr(*state.General)
}

// pvpReinforcementGeneralExpByID 计算每支实际带武将参战援军的武将经验。
func pvpReinforcementGeneralExpByID(records []Reinforcement, exp int) map[string]int {
	if exp <= 0 || len(records) == 0 {
		return nil
	}
	result := map[string]int{}
	for _, record := range records {
		normalizeGarrisonRecord(&record)
		if record.Status != ReinforcementStatusStationed || !record.Rules.CanFight || totalTroops(record.RemainingTroops) <= 0 || len(record.Generals) == 0 {
			continue
		}
		result[record.ID] = exp
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

// pvpModifierSourcesForGenerals 构建 PVP 出征专用加成：基础加成保留，武将加成只来自本次携带武将。
func pvpModifierSourcesForGenerals(state *GameState, generalIDs []string) []ModifierSource {
	return modifierSourcesForBattleGenerals(state, generalIDs)
}

// pvpModifierSourcesForHomeDefense 构建守城专用加成，只有留在城内的主将会提供武将加成。
func pvpModifierSourcesForHomeDefense(state *GameState) []ModifierSource {
	if state == nil {
		return nil
	}
	base := *state
	base.General = nil
	base.Buildings = buildingsWithoutType(state.Buildings, "city_wall")
	sources := CollectModifierSources(&base)
	if general := pvpHomeDefenseGeneral(state); general != nil {
		applyHeroConfigToGeneral(general)
		sources = append(sources, &GeneralModifierSource{General: general})
	}
	return sources
}

// pvpDefenderCityWallLevel 返回守城方当前生效城墙等级。
func pvpDefenderCityWallLevel(state *GameState) int {
	if state == nil {
		return 0
	}
	level := 0
	for _, building := range state.Buildings {
		if building.Type != "city_wall" || !buildingIsOperational(building) {
			continue
		}
		if building.Level > level {
			level = building.Level
		}
	}
	return level
}

// buildPvpWallSnapshot 生成城墙系数与硬度预留快照，供战报和后续攻城破坏逻辑复用。
func buildPvpWallSnapshot(faction string, level int) *PvpWallSnapshot {
	if level <= 0 || strings.TrimSpace(faction) == "" {
		return nil
	}
	cfg := combat.GetCombatConfig()
	entry, ok := cfg.WallConfig[faction]
	if !ok || entry.Base <= 0 {
		return nil
	}
	multiplier := math.Pow(entry.Base, float64(level))
	factionDefenseBonus := multiplier - 1
	return &PvpWallSnapshot{
		Faction:               faction,
		Level:                 level,
		Base:                  entry.Base,
		Multiplier:            multiplier,
		FactionDefenseBonus:   factionDefenseBonus,
		TotalDefenseBonus:     factionDefenseBonus,
		Hardness:              entry.Hardness,
		MinDamagedLevelFrom20: entry.MinDamagedLevelFrom20,
		MaxDamagedLevelFrom20: entry.MaxDamagedLevelFrom20,
	}
}

// buildingsWithoutType 返回排除指定类型后的建筑列表。
func buildingsWithoutType(buildings []Building, buildingType string) []Building {
	next := make([]Building, 0, len(buildings))
	for _, building := range buildings {
		if building.Type == buildingType {
			continue
		}
		next = append(next, building)
	}
	return next
}

type pvpDefenseSourceGroup struct {
	Key             string
	ReinforcementID string
	UnitType        string
	Amount          int
}

func buildPvpDefenderUnits(defender *GameState, reinforcements []Reinforcement, now time.Time, scene string, combatTraitsDisabled ...bool) ([]combat.Unit, []pvpDefenseSourceGroup, map[string]map[string]general.TraitOutcome, error) {
	groups := []pvpDefenseSourceGroup{}
	units := []combat.Unit{}
	reinforcementOutcomes := map[string]map[string]general.TraitOutcome{}
	for _, armyUnit := range defender.Army {
		if armyUnit.Amount <= 0 {
			continue
		}
		groups = append(groups, pvpDefenseSourceGroup{Key: "defender", UnitType: armyUnit.UnitType, Amount: armyUnit.Amount})
		unitCfg, _, ok := findAnyUnitConfig(defender.Player.Faction, armyUnit.UnitType)
		if !ok {
			return nil, nil, nil, ErrUnitNotFound
		}
		units = append(units, buildCombatUnitFromConfig(armyUnit.UnitType, armyUnit.Amount, unitCfg, now, pvpModifierSourcesForHomeDefense(defender)...))
	}
	for _, record := range reinforcements {
		normalizeGarrisonRecord(&record)
		if record.Status != ReinforcementStatusStationed || !record.Rules.CanFight {
			continue
		}
		recordUnits := []combat.Unit{}
		for unitType, amount := range record.RemainingTroops {
			if amount <= 0 {
				continue
			}
			groups = append(groups, pvpDefenseSourceGroup{Key: "reinforcement", ReinforcementID: record.ID, UnitType: unitType, Amount: amount})
			unitCfg, _, ok := findAnyUnitConfig(record.FromPlayerFaction, unitType)
			if !ok {
				return nil, nil, nil, ErrUnitNotFound
			}
			recordUnits = append(recordUnits, buildCombatUnitFromConfig(unitType, amount, unitCfg, now, modifierSourcesForReinforcement(record)...))
		}
		recordUnits, reinforcementOutcomes[record.ID] = applyReinforcementBeforeBattleTraits(record, recordUnits, scene, len(combatTraitsDisabled) > 0 && combatTraitsDisabled[0])
		units = append(units, recordUnits...)
	}
	if len(units) == 0 {
		return nil, nil, nil, ErrNoUnitsSelected
	}
	return units, groups, reinforcementOutcomes, nil
}

// modifierSourcesForReinforcement 把援军出发时保存的携带武将加成快照转换为本场守城加成来源。
func modifierSourcesForReinforcement(record Reinforcement) []ModifierSource {
	if len(record.BuffSnapshot) == 0 {
		return nil
	}
	mods := make([]Modifier, 0, len(record.BuffSnapshot))
	for _, item := range record.BuffSnapshot {
		if strings.TrimSpace(item.Key) == "" || item.Value == 0 {
			continue
		}
		mode := item.Mode
		if strings.TrimSpace(mode) == "" {
			mode = "percentAdd"
		}
		mods = append(mods, Modifier{
			Key:   item.Key,
			Value: item.Value,
			Mode:  mode,
		})
	}
	if len(mods) == 0 {
		return nil
	}
	return []ModifierSource{&StaticModifierSource{Name: "援军武将", Mods: mods}}
}

func findAnyUnitConfig(preferredFaction string, unitType string) (UnitConfig, string, bool) {
	if cfg, ok := GetUnitConfig(preferredFaction, unitType); ok {
		return cfg, preferredFaction, true
	}
	for _, faction := range []string{"wei", "shu", "wu", "neutral"} {
		if cfg, ok := GetUnitConfig(faction, unitType); ok {
			return cfg, faction, true
		}
	}
	return UnitConfig{}, "", false
}

func combatLossMap(losses []combat.UnitLoss) map[string]int {
	result := map[string]int{}
	for _, loss := range losses {
		if loss.Losses > 0 {
			result[loss.ID] += loss.Losses
		}
	}
	return result
}

func allocatePvpDefenderLosses(totalLosses map[string]int, groups []pvpDefenseSourceGroup) (map[string]int, map[string]map[string]int) {
	defenderLosses := map[string]int{}
	reinforcementLosses := map[string]map[string]int{}
	for unitType, totalLoss := range totalLosses {
		if totalLoss <= 0 {
			continue
		}
		totalAmount := 0
		for _, group := range groups {
			if group.UnitType == unitType {
				totalAmount += group.Amount
			}
		}
		if totalAmount <= 0 {
			continue
		}
		assigned := 0
		type remainder struct {
			index int
			value float64
		}
		remainders := []remainder{}
		for i, group := range groups {
			if group.UnitType != unitType || group.Amount <= 0 {
				continue
			}
			exact := float64(totalLoss) * float64(group.Amount) / float64(totalAmount)
			lost := int(math.Floor(exact))
			if lost > group.Amount {
				lost = group.Amount
			}
			assigned += lost
			addPvpSourceLoss(defenderLosses, reinforcementLosses, group, lost)
			remainders = append(remainders, remainder{index: i, value: exact - float64(lost)})
		}
		sort.Slice(remainders, func(i, j int) bool { return remainders[i].value > remainders[j].value })
		for remaining := totalLoss - assigned; remaining > 0 && len(remainders) > 0; remaining-- {
			group := groups[remainders[(totalLoss-assigned-remaining)%len(remainders)].index]
			addPvpSourceLoss(defenderLosses, reinforcementLosses, group, 1)
		}
	}
	return defenderLosses, reinforcementLosses
}

func reducePvpSourceGroups(groups []pvpDefenseSourceGroup, defenderReductions map[string]int, reinforcementReductions map[string]map[string]int) []pvpDefenseSourceGroup {
	result := make([]pvpDefenseSourceGroup, 0, len(groups))
	for _, group := range groups {
		reduced := 0
		if group.Key == "defender" {
			reduced = defenderReductions[group.UnitType]
		} else if group.ReinforcementID != "" {
			reduced = reinforcementReductions[group.ReinforcementID][group.UnitType]
		}
		group.Amount -= reduced
		if group.Amount > 0 {
			result = append(result, group)
		}
	}
	return result
}

func mergeNestedStringIntMap(groups ...map[string]map[string]int) map[string]map[string]int {
	result := map[string]map[string]int{}
	for _, group := range groups {
		for outer, inner := range group {
			if result[outer] == nil {
				result[outer] = map[string]int{}
			}
			for key, value := range inner {
				if value > 0 {
					result[outer][key] += value
				}
			}
		}
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

// pvpDefenderLossRatio 计算守方含援军在内的本场损失比例。
func pvpDefenderLossRatio(groups []pvpDefenseSourceGroup, totalLosses map[string]int) float64 {
	before := 0
	for _, group := range groups {
		if group.Amount > 0 {
			before += group.Amount
		}
	}
	if before <= 0 {
		return 0
	}
	lost := 0
	for _, amount := range totalLosses {
		if amount > 0 {
			lost += amount
		}
	}
	return float64(lost) / float64(before)
}

// enemyLossRevealThreshold 通过 Modifier 管线计算敌方剩余兵力暴露阈值。
func enemyLossRevealThreshold(defender *GameState, now time.Time) float64 {
	threshold := ComputeAttributeAt(0.25, StatEnemyLossRevealThreshold, now, pvpModifierSourcesForHomeDefense(defender)...)
	if threshold < 0 {
		return 0
	}
	if threshold > 1 {
		return 1
	}
	return threshold
}

func addPvpSourceLoss(defenderLosses map[string]int, reinforcementLosses map[string]map[string]int, group pvpDefenseSourceGroup, lost int) {
	if lost <= 0 {
		return
	}
	if group.Key == "defender" {
		defenderLosses[group.UnitType] += lost
		return
	}
	if reinforcementLosses[group.ReinforcementID] == nil {
		reinforcementLosses[group.ReinforcementID] = map[string]int{}
	}
	reinforcementLosses[group.ReinforcementID][group.UnitType] += lost
}

func applyArmyLosses(state *GameState, losses map[string]int) {
	for unitType, lost := range losses {
		for i := range state.Army {
			if state.Army[i].UnitType == unitType {
				state.Army[i].Amount -= lost
				if state.Army[i].Amount < 0 {
					state.Army[i].Amount = 0
				}
				break
			}
		}
	}
	state.Army = armyMapToSlice(armySliceToMap(state.Army))
}

// applyPvpReinforcementLosses 记录所有实际参战驻防；即使零损失也必须进入战报和经验结算链路。
func applyPvpReinforcementLosses(records []Reinforcement, losses map[string]map[string]int, now time.Time) []Reinforcement {
	changed := []Reinforcement{}
	nowText := now.Format(resourceDateLayout)
	for _, record := range records {
		normalizeGarrisonRecord(&record)
		if record.Status != ReinforcementStatusStationed || !record.Rules.CanFight || totalTroops(record.RemainingTroops) <= 0 {
			continue
		}
		// record 是结构体副本，但内部 map 仍与战前快照共享；写损耗前必须深拷贝。
		record.RemainingTroops = cloneStringIntMap(record.RemainingTroops)
		record.Losses = cloneStringIntMap(record.Losses)
		unitLosses := losses[record.ID]
		if record.Losses == nil {
			record.Losses = map[string]int{}
		}
		for unitType, lost := range unitLosses {
			if lost <= 0 {
				continue
			}
			before := record.RemainingTroops[unitType]
			record.RemainingTroops[unitType] = before - lost
			if record.RemainingTroops[unitType] < 0 {
				record.RemainingTroops[unitType] = 0
			}
			record.Losses[unitType] += lost
		}
		record.LastBattleAt = nowText
		record.UpdatedAt = nowText
		total := 0
		for _, amount := range record.RemainingTroops {
			total += amount
		}
		if total <= 0 {
			record.IsAnnihilated = true
			record.Status = ReinforcementStatusCompleted
			record.ReturnedAt = nowText
		}
		changed = append(changed, record)
	}
	return changed
}

// reinforcementTraitReportPhases 保存援军各阶段结果，避免战报合并时丢失真实执行顺序。
type reinforcementTraitReportPhases struct {
	Before           map[string]map[string]general.TraitOutcome
	EnemyAfterCombat map[string]map[string]general.TraitOutcome
	SelfAfterCombat  map[string]map[string]general.TraitOutcome
	AfterBattle      map[string]map[string]general.TraitOutcome
	All              map[string]map[string]general.TraitOutcome
}

// buildPvpReinforcementReports 保留旧调用契约并按已有结果顺序生成独立增援战报。
func buildPvpReinforcementReports(eventID string, attacker *GameState, defender *GameState, changed []Reinforcement, losses map[string]map[string]int, expByReinforcement map[string]int, traitOutcomes map[string]map[string]general.TraitOutcome, battleResult string, nowText string) []BattleReport {
	return buildPvpReinforcementReportsByPhase(nil, eventID, attacker, defender, changed, losses, expByReinforcement, reinforcementTraitReportPhases{All: traitOutcomes}, battleResult, nowText)
}

// buildPvpReinforcementReportsByPhase 以完整防守战报为底稿，并叠加每支援军自己的分阶段结果。
func buildPvpReinforcementReportsByPhase(baseDefenseReport *BattleReport, eventID string, attacker *GameState, defender *GameState, changed []Reinforcement, losses map[string]map[string]int, expByReinforcement map[string]int, traitPhases reinforcementTraitReportPhases, battleResult string, nowText string) []BattleReport {
	reports := []BattleReport{}
	for i := range changed {
		record := &changed[i]
		normalizeGarrisonRecord(record)
		unitLosses := cloneStringIntMap(losses[record.ID])
		before := cloneStringIntMap(record.RemainingTroops)
		for unitType, lost := range unitLosses {
			if lost > 0 {
				before[unitType] += lost
			}
		}
		reportID := "br_pvp_rein_" + randomID(8)
		record.LastBattleReportID = reportID
		reinforcementSnapshot := reinforcementReportSnapshot(*record, before, expByReinforcement[record.ID])
		var report BattleReport
		if baseDefenseReport != nil {
			report = cloneBattleReport(*baseDefenseReport)
			report.DefenderLostUnits = nil
			report.TraitTriggered = nil
			report.TraitOutcomes = nil
			if report.Detail != nil {
				report.Detail.Traits = nil
			}
		} else {
			report = BattleReport{
				TargetID:         defender.Player.ID,
				TargetName:       defender.Player.Nickname + "（驻防城）",
				SourceType:       ReportSourcePlayerCity,
				BattleType:       "reinforcement_battle",
				Result:           battleResult,
				DefenderFaction:  defender.Player.Faction,
				DefenderRevealed: false,
				Title:            "增援" + defender.Player.Nickname + "参战",
				PvpReinforcements: []DefenseReinforcementUnit{
					reinforcementSnapshot,
				},
				PvpReinforcementLosses: map[string]map[string]int{record.ID: unitLosses},
			}
		}
		report.ID = reportID
		report.EventID = eventID
		report.PlayerID = record.OwnerPlayerID
		report.OwnerPlayerID = record.OwnerPlayerID
		report.PlayerFaction = record.FromPlayerFaction
		report.PlayerName = record.FromPlayerName
		report.Type = "reinforce"
		report.ViewType = ReportViewReinforcement
		report.OwnerSide = ReportOwnerSideReinforcement
		report.DispatchedUnits = before
		report.LostUnits = unitLosses
		report.Rewards = map[string]int{}
		report.Summary = buildReinforcementReportSummary(record, unitLosses)
		report.OwnerOutcome = inferReportOwnerOutcome(report)
		ownerSnapshotFound := false
		for index := range report.PvpReinforcements {
			if report.PvpReinforcements[index].ReinforcementID == record.ID {
				report.PvpReinforcements[index] = reinforcementSnapshot
				ownerSnapshotFound = true
			}
		}
		if !ownerSnapshotFound {
			report.PvpReinforcements = append(report.PvpReinforcements, reinforcementSnapshot)
		}
		report.Read = false
		report.CreatedAt = nowText
		report.Share = nil
		report.GeneralExpGained = expByReinforcement[record.ID]
		report.GeneralLevelBefore = reinforcementSnapshot.GeneralLevelBefore
		report.GeneralLevelAfter = reinforcementSnapshot.GeneralLevelAfter
		phaseOutcomes := []map[string]map[string]general.TraitOutcome{traitPhases.Before, traitPhases.EnemyAfterCombat, traitPhases.SelfAfterCombat, traitPhases.AfterBattle}
		mergedByPhase := false
		for _, outcomes := range phaseOutcomes {
			if len(outcomes[record.ID]) == 0 {
				continue
			}
			mergedByPhase = true
			mergeTraitOutcomes(&report, outcomes[record.ID])
		}
		if !mergedByPhase {
			mergeTraitOutcomes(&report, traitPhases.All[record.ID])
		}
		report.RevivedUnits = reinforcementReturnedUnits(traitPhases.All[record.ID])
		report.LostUnits = reinforcementGrossLosses(before, unitLosses, report.RevivedUnits)
		report.SurvivedUnits = calculateReportSurvivedUnits(before, report.LostUnits, report.RevivedUnits, nil)
		if report.PvpReinforcementLosses == nil {
			report.PvpReinforcementLosses = map[string]map[string]int{}
		}
		report.PvpReinforcementLosses[record.ID] = cloneStringIntMap(unitLosses)
		if report.Detail == nil {
			detail := BuildBattleReportDetail(report)
			report.Detail = &detail
		} else {
			report.Detail.Traits = convertReportTraits(report)
		}
		report.Detail.ID = reportID
		report.Detail.OwnerPlayerID = record.OwnerPlayerID
		report.Detail.ViewType = ReportViewReinforcement
		report.Detail.ViewLabel = reportViewLabel(ReportViewReinforcement)
		report.Detail.OwnerSide = ReportOwnerSideReinforcement
		report.Detail.OwnerOutcome = report.OwnerOutcome
		report.Detail.Rewards = BattleReportRewards{
			GeneralExp:         report.GeneralExpGained,
			GeneralLevelBefore: report.GeneralLevelBefore,
			GeneralLevelAfter:  report.GeneralLevelAfter,
		}
		report.Detail.Visibility = buildReportVisibility(report)
		report.Detail.Read = false
		report.Detail.Share = nil
		report.Detail.Extra = mergeReportExtraMap(report.Detail.Extra, map[string]interface{}{
			"reinforcement": buildReinforcementReportExtra(eventID, attacker, defender, record, before, unitLosses, expByReinforcement[record.ID], battleResult),
		})
		reports = append(reports, NormalizeBattleReport(report))
	}
	return reports
}

// reinforcementReturnedUnits 汇总援军战后复活和损失返还的实际兵力。
func reinforcementReturnedUnits(outcomes map[string]general.TraitOutcome) map[string]int {
	returned := map[string]int{}
	for _, outcome := range outcomes {
		for _, key := range []string{"revivedUnits", "returnedUnits"} {
			units, ok := traitDetailIntMap(outcome.Detail[key])
			if !ok {
				continue
			}
			for unitType, amount := range units {
				if amount > 0 {
					returned[unitType] += amount
				}
			}
		}
	}
	if len(returned) == 0 {
		return nil
	}
	return returned
}

// reinforcementGrossLosses 由最终净损失和战后归队数还原战斗原始阵亡。
func reinforcementGrossLosses(dispatched map[string]int, finalLosses map[string]int, returned map[string]int) map[string]int {
	gross := cloneStringIntMap(finalLosses)
	for unitType, amount := range returned {
		if amount <= 0 {
			continue
		}
		gross[unitType] += amount
		if sent := dispatched[unitType]; sent >= 0 && gross[unitType] > sent {
			gross[unitType] = sent
		}
	}
	return gross
}

// buildReinforcementReportSummary 生成增援战报摘要。
func buildReinforcementReportSummary(record *Reinforcement, losses map[string]int) string {
	lost := 0
	for _, amount := range losses {
		lost += amount
	}
	if record.IsAnnihilated {
		return "援军参战并全灭"
	}
	return "援军参战，损失 " + strconv.Itoa(lost) + " 兵"
}

// reinforcementReportSnapshot 生成增援战报中的本批队伍快照。
func reinforcementReportSnapshot(record Reinforcement, before map[string]int, generalExp int) DefenseReinforcementUnit {
	progress := reinforcementGeneralProgress(record.Generals, generalExp)
	return DefenseReinforcementUnit{
		ReinforcementID:    record.ID,
		FromPlayerID:       record.OwnerPlayerID,
		FromPlayerName:     record.FromPlayerName,
		Faction:            record.FromPlayerFaction,
		Troops:             cloneStringIntMap(before),
		Generals:           buildReinforcementGeneralBattleSnapshots(record.Generals, generalExp),
		GeneralExpGained:   generalExp,
		GeneralLevelBefore: progress.LevelBefore,
		GeneralLevelAfter:  progress.LevelAfter,
		Buffs:              append([]ModifierBreakdownItem(nil), record.BuffSnapshot...),
		SourceTags: map[string]string{
			"source_type":      record.SourceType,
			"source_player_id": record.OwnerPlayerID,
			"source_record_id": record.ID,
			"source_id":        record.SourceID,
		},
	}
}

// buildReinforcementReportExtra 构造协防方可读上下文，说明帮谁、挡谁和自身损耗。
func buildReinforcementReportExtra(eventID string, attacker *GameState, defender *GameState, record *Reinforcement, before map[string]int, losses map[string]int, generalExp int, battleResult string) map[string]interface{} {
	if record == nil || defender == nil {
		return map[string]interface{}{}
	}
	host := map[string]interface{}{
		"playerId":   defender.Player.ID,
		"playerName": defender.Player.Nickname,
		"cityName":   defender.Player.Nickname,
	}
	attackerInfo := map[string]interface{}{}
	if attacker != nil {
		attackerInfo = map[string]interface{}{
			"playerId":   attacker.Player.ID,
			"playerName": attacker.Player.Nickname,
			"cityName":   attacker.Player.Nickname,
		}
	}
	return map[string]interface{}{
		"reinforcementId":   record.ID,
		"hostPlayerId":      defender.Player.ID,
		"hostPlayerName":    defender.Player.Nickname,
		"hostCityName":      defender.Player.Nickname,
		"attackerPlayerId":  attackerInfo["playerId"],
		"attackerName":      attackerInfo["playerName"],
		"attackerCityName":  attackerInfo["cityName"],
		"battleEventId":     eventID,
		"battleResult":      battleResult,
		"host":              host,
		"attacker":          attackerInfo,
		"ownerContribution": buildReinforcementOwnerContribution(record, before, losses, generalExp),
	}
}

// buildReinforcementOwnerContribution 汇总协防方自己的出动、阵亡、剩余和武将经验。
func buildReinforcementOwnerContribution(record *Reinforcement, before map[string]int, losses map[string]int, generalExp int) map[string]interface{} {
	survived := cloneStringIntMap(before)
	for unitType, lost := range losses {
		survived[unitType] -= lost
		if survived[unitType] < 0 {
			survived[unitType] = 0
		}
	}
	return map[string]interface{}{
		"troopsBefore":   cloneStringIntMap(before),
		"troopsLost":     cloneStringIntMap(losses),
		"troopsSurvived": survived,
		"generalExp":     generalExp,
		"generals":       buildReinforcementGeneralBattleSnapshots(record.Generals, generalExp),
	}
}

// calculatePvpPlunder 只计算受负重和保护量限制的基础掠夺，不提前修改双方资源。
func calculatePvpPlunder(defender *GameState, carryCapacity int) map[string]int {
	if defender.Resources.Items == nil {
		return map[string]int{}
	}
	available := map[string]int{}
	total := 0
	for resourceType, amount := range defender.Resources.Items {
		protected := defender.Resources.Capacity[resourceType] / 5
		canTake := amount - protected
		if canTake > 0 {
			available[resourceType] = canTake
			total += canTake
		}
	}
	if total <= 0 || carryCapacity <= 0 {
		return map[string]int{}
	}
	if carryCapacity > total {
		carryCapacity = total
	}
	plundered := map[string]int{}
	assigned := 0
	for resourceType, amount := range available {
		take := int(math.Floor(float64(carryCapacity) * float64(amount) / float64(total)))
		if take > amount {
			take = amount
		}
		plundered[resourceType] = take
		assigned += take
	}
	for resourceType, amount := range available {
		if assigned >= carryCapacity {
			break
		}
		if plundered[resourceType] < amount {
			plundered[resourceType]++
			assigned++
		}
	}
	return plundered
}

// applyPvpPlunder 按特性修正后的最终数量转移资源，并返回双方真实结算值。
func applyPvpPlunder(defender *GameState, attacker *GameState, requested map[string]int) map[string]int {
	settled := map[string]int{}
	if defender == nil || attacker == nil || len(requested) == 0 {
		return settled
	}
	for resourceType, amount := range requested {
		if amount <= 0 {
			continue
		}
		protected := defender.Resources.Capacity[resourceType] / 5
		available := defender.Resources.Items[resourceType] - protected
		if available <= 0 {
			continue
		}
		if amount > available {
			amount = available
		}
		granted, _, err := addResourceCapped(attacker, resourceType, amount)
		if err != nil || granted <= 0 {
			continue
		}
		defender.Resources.Items[resourceType] -= granted
		settled[resourceType] = granted
	}
	return settled
}

func buildPvpBattleReport(id string, owner *GameState, target *GameState, march *PvpMarch, result string, playerPower int, enemyPower int, dispatched map[string]int, lost map[string]int, defenderUnits map[string]int, defenderLost map[string]int, rewards map[string]int, nowText string, reportType string) BattleReport {
	return BattleReport{
		ID:                id,
		PlayerID:          owner.Player.ID,
		OwnerPlayerID:     owner.Player.ID,
		PlayerFaction:     owner.Player.Faction,
		PlayerName:        owner.Player.Nickname,
		TargetID:          target.Player.ID,
		TargetName:        target.Player.Nickname + "（玩家）",
		Type:              reportType,
		SourceType:        ReportSourcePlayerCity,
		BattleType:        reportType,
		Result:            result,
		PlayerPower:       playerPower,
		EnemyPower:        enemyPower,
		DispatchedUnits:   cloneStringIntMap(dispatched),
		LostUnits:         cloneStringIntMap(lost),
		DefenderFaction:   target.Player.Faction,
		DefenderUnits:     cloneStringIntMap(defenderUnits),
		DefenderLostUnits: cloneStringIntMap(defenderLost),
		DefenderRevealed:  true,
		DefenderResources: copyResources(target.Resources.Items),
		Rewards:           cloneStringIntMap(rewards),
		Read:              false,
		CreatedAt:         nowText,
	}
}

// cloneNestedStringIntMap 复制双层兵种数量映射，避免战报和战斗结果共享引用。
func cloneNestedStringIntMap(src map[string]map[string]int) map[string]map[string]int {
	if len(src) == 0 {
		return nil
	}
	dst := make(map[string]map[string]int, len(src))
	for key, value := range src {
		dst[key] = cloneStringIntMap(value)
	}
	return dst
}

func invertPvpReportResult(result string) string {
	switch result {
	case "attacker_victory":
		return "attacker_victory"
	case "defender_victory":
		return "defender_victory"
	default:
		return result
	}
}

// buildPvpReinforcementSnapshot 保存战前增援兵力，并附带本次战斗的逐队武将经验。
func buildPvpReinforcementSnapshot(records []Reinforcement, generalExpByID map[string]int) []DefenseReinforcementUnit {
	result := []DefenseReinforcementUnit{}
	for _, record := range records {
		normalizeGarrisonRecord(&record)
		if record.Status != ReinforcementStatusStationed || !record.Rules.CanFight || totalTroops(record.RemainingTroops) <= 0 {
			continue
		}
		generalExp := generalExpByID[record.ID]
		progress := reinforcementGeneralProgress(record.Generals, generalExp)
		result = append(result, DefenseReinforcementUnit{
			ReinforcementID:    record.ID,
			FromPlayerID:       record.FromPlayerID,
			FromPlayerName:     record.FromPlayerName,
			Faction:            record.FromPlayerFaction,
			Troops:             cloneStringIntMap(record.RemainingTroops),
			Generals:           buildReinforcementGeneralBattleSnapshots(record.Generals, generalExp),
			GeneralExpGained:   generalExp,
			GeneralLevelBefore: progress.LevelBefore,
			GeneralLevelAfter:  progress.LevelAfter,
			Buffs:              append([]ModifierBreakdownItem(nil), record.BuffSnapshot...),
			SourceTags:         map[string]string{"source_type": record.SourceType, "source_id": record.SourceID},
		})
	}
	return result
}

// buildReinforcementGeneralBattleSnapshots 计算援军武将本场经验和等级变化，并移除仅供结算使用的累计经验。
func buildReinforcementGeneralBattleSnapshots(generals []ReinforcementGeneralSnapshot, gained int) []ReinforcementGeneralSnapshot {
	result := cloneReinforcementGenerals(generals)
	if gained < 0 {
		gained = 0
	}
	for index := range result {
		before := result[index].Level
		if before <= 0 {
			before = 1
		}
		baseExp := result[index].Exp
		if minimum := generalExpRequiredForLevel(before); baseExp < minimum {
			baseExp = minimum
		}
		temporary := General{Level: before, Exp: baseExp}
		after := before
		if gained > 0 {
			temporary.Exp += gained
			promoteGeneralByExp(&temporary)
			after = temporary.Level
		}
		gainedSnapshot := gained
		beforeSnapshot := before
		afterSnapshot := after
		result[index].Level = after
		result[index].Exp = 0
		result[index].GeneralExpGained = &gainedSnapshot
		result[index].GeneralLevelBefore = &beforeSnapshot
		result[index].GeneralLevelAfter = &afterSnapshot
	}
	return result
}

// reinforcementGeneralProgress 根据战斗前累计经验计算援军武将本场的等级变化。
func reinforcementGeneralProgress(generals []ReinforcementGeneralSnapshot, gained int) generalExpResult {
	if gained <= 0 || len(generals) == 0 || strings.TrimSpace(generals[0].ID) == "" {
		return generalExpResult{}
	}
	levelBefore := generals[0].Level
	if levelBefore <= 0 {
		levelBefore = 1
	}
	baseExp := maxInt(0, generals[0].Exp)
	if minimum := generalExpRequiredForLevel(levelBefore); baseExp < minimum {
		baseExp = minimum
	}
	temporary := General{Level: levelBefore, Exp: baseExp + gained}
	promoteGeneralByExp(&temporary)
	return generalExpResult{Gained: gained, LevelBefore: levelBefore, LevelAfter: temporary.Level}
}

// advanceReinforcementGeneralSnapshots 推进驻防记录中的经验基线，供同一援军下一场战斗继续准确结算。
func advanceReinforcementGeneralSnapshots(records []Reinforcement, generalExpByID map[string]int) {
	for recordIndex := range records {
		gained := generalExpByID[records[recordIndex].ID]
		if gained <= 0 {
			continue
		}
		for generalIndex := range records[recordIndex].Generals {
			progress := reinforcementGeneralProgress([]ReinforcementGeneralSnapshot{records[recordIndex].Generals[generalIndex]}, gained)
			if progress.Gained <= 0 {
				continue
			}
			records[recordIndex].Generals[generalIndex].Exp += gained
			records[recordIndex].Generals[generalIndex].Level = progress.LevelAfter
		}
	}
}

// buildPvpGeneralSnapshots 按指定武将 ID 生成 PVP 参战武将快照。
func buildPvpGeneralSnapshots(state *GameState, generalIDs []string) []PvpGeneralSnapshot {
	if state == nil || len(generalIDs) == 0 {
		return nil
	}
	result := []PvpGeneralSnapshot{}
	seen := map[string]bool{}
	for _, generalID := range generalIDs {
		generalID = strings.TrimSpace(generalID)
		if generalID == "" || seen[generalID] {
			continue
		}
		seen[generalID] = true
		if general, ok := findOwnedGeneral(state.Generals, generalID); ok {
			result = append(result, snapshotPvpGeneral(general))
		}
	}
	return result
}

// buildPvpDefenseGeneralSnapshots 选择当前主将作为第一版 PVP 防守武将。
func buildPvpDefenseGeneralSnapshots(state *GameState) []PvpGeneralSnapshot {
	general := pvpHomeDefenseGeneral(state)
	if general == nil {
		return nil
	}
	return []PvpGeneralSnapshot{snapshotPvpGeneral(*general)}
}

// snapshotPvpGeneral 复制 PVP 战斗中需要展示的武将字段。
func snapshotPvpGeneral(general General) PvpGeneralSnapshot {
	return PvpGeneralSnapshot{
		ID:             general.ID,
		Name:           general.Name,
		Level:          general.Level,
		Stats:          cloneStringIntMap(general.Stats),
		EffectiveStats: cloneStringIntMap(general.EffectiveStats),
		Attributes:     cloneFloatMap(general.Attributes),
		Buffs:          cloneFloatMap(general.Buffs),
		Traits:         cloneGeneralTraitInstances(general.Traits),
	}
}

// clonePvpGeneralSnapshots 深拷贝参战将领快照及其嵌套 Buff、特性字段。
func clonePvpGeneralSnapshots(items []PvpGeneralSnapshot) []PvpGeneralSnapshot {
	if len(items) == 0 {
		return nil
	}
	cloned := make([]PvpGeneralSnapshot, len(items))
	for index, item := range items {
		cloned[index] = item
		cloned[index].Stats = cloneStringIntMap(item.Stats)
		cloned[index].EffectiveStats = cloneStringIntMap(item.EffectiveStats)
		cloned[index].Attributes = cloneFloatMap(item.Attributes)
		cloned[index].Buffs = cloneFloatMap(item.Buffs)
		cloned[index].Traits = cloneGeneralTraitInstances(item.Traits)
	}
	return cloned
}
