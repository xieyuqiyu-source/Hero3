// 本文件实现 PVP 系统应用服务，负责目标、行军、战斗结算和战报。
package game

import (
	"encoding/json"
	"errors"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"hero3/internal/core/combat"
)

var (
	ErrPvpTargetSelf          = errors.New("cannot attack self")
	ErrPvpSameAccountTarget   = errors.New("cannot attack another save in same account")
	ErrPvpTargetProtected     = errors.New("pvp target is protected")
	ErrPvpAttackCooldown      = errors.New("pvp attack is on cooldown")
	ErrPvpDailyLimitReached   = errors.New("pvp daily attack limit reached")
	ErrPvpMarchNotReady       = errors.New("pvp march has not arrived")
	ErrPvpMarchNotRecallable  = errors.New("pvp march cannot be recalled")
	ErrPvpMarchNotAccelerable = errors.New("pvp march cannot be accelerated")
	ErrInvalidPvpSeason       = errors.New("invalid pvp season")
)

// ListPvpTargets 返回可展示的玩家目标列表。
func (s *Service) ListPvpTargets(playerID string) (PvpTargetsResponse, error) {
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
	for _, account := range accounts {
		for _, player := range account.Players {
			if player.ID == playerID {
				continue
			}
			canAttack := account.ID != requestAccountID
			reason := ""
			protectedUntil := ""
			cooldownUntil := requestPvpState.CooldownUntil
			targetPvpState, _ := s.repo.GetPvpPlayerState(player.ID, now)
			protected, protectionType, activeUntil := activePvpProtection(targetPvpState, now)
			if protected {
				canAttack = false
				protectedUntil = activeUntil
				reason = pvpProtectionReason(protectionType)
			}
			if pvpTimeAfter(requestPvpState.CooldownUntil, now) {
				canAttack = false
				reason = "攻击冷却中"
			}
			if requestPvpState.DailyAttackLimit > 0 && requestPvpState.DailyAttackCount >= requestPvpState.DailyAttackLimit {
				canAttack = false
				reason = "今日攻击次数已用完"
			}
			if account.ID == requestAccountID {
				reason = "同账号存档不能攻击"
			}
			items = append(items, PvpTargetSummary{
				PlayerID:       player.ID,
				Nickname:       player.Nickname,
				Faction:        player.Faction,
				TotalArmy:      player.TotalArmy,
				BuildingLevel:  player.BuildingLevel,
				CanAttack:      canAttack,
				CanReinforce:   true,
				Protected:      protectedUntil != "",
				ProtectedUntil: protectedUntil,
				CooldownUntil:  cooldownUntil,
				Reason:         reason,
			})
		}
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].TotalArmy == items[j].TotalArmy {
			return items[i].PlayerID < items[j].PlayerID
		}
		return items[i].TotalArmy > items[j].TotalArmy
	})
	return PvpTargetsResponse{Items: items}, nil
}

// GetPvpTarget 返回单个玩家 PVP 目标摘要。
func (s *Service) GetPvpTarget(playerID string, targetPlayerID string) (PvpTargetSummary, error) {
	targets, err := s.ListPvpTargets(playerID)
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

// ScoutPvpTarget 侦查玩家目标并生成侦查战报。
func (s *Service) ScoutPvpTarget(req PvpScoutRequest) (PvpScoutResponse, error) {
	playerID := strings.TrimSpace(req.PlayerID)
	targetPlayerID := strings.TrimSpace(req.TargetPlayerID)
	if err := s.ensurePvpTargetAllowed(playerID, targetPlayerID); err != nil {
		return PvpScoutResponse{}, err
	}
	scout, err := s.repo.GetState(playerID)
	if err != nil {
		return PvpScoutResponse{}, err
	}
	target, err := s.repo.GetState(targetPlayerID)
	if err != nil {
		return PvpScoutResponse{}, err
	}
	now := time.Now().UTC()
	report := BattleReport{
		ID:                "br_pvp_scout_" + randomID(8),
		PlayerID:          scout.Player.ID,
		PlayerFaction:     scout.Player.Faction,
		PlayerName:        scout.Player.Nickname,
		TargetID:          target.Player.ID,
		TargetName:        target.Player.Nickname + "（玩家）",
		Type:              "scout",
		Result:            "success",
		DispatchedUnits:   map[string]int{},
		LostUnits:         map[string]int{},
		DefenderFaction:   target.Player.Faction,
		DefenderUnits:     armySliceToMap(target.Army),
		DefenderLostUnits: map[string]int{},
		DefenderRevealed:  true,
		DefenderResources: copyResources(target.Resources.Items),
		Rewards:           map[string]int{},
		Read:              false,
		CreatedAt:         now.Format(resourceDateLayout),
	}
	if err := s.repo.SaveReport(report); err != nil {
		return PvpScoutResponse{}, err
	}
	return PvpScoutResponse{Success: true, BattleReport: report, ServerTime: now.Format(resourceDateLayout)}, nil
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
	troops := normalizePositiveTroops(req.Troops)
	if len(troops) == 0 {
		return PvpAttackResponse{}, ErrNoUnitsSelected
	}
	mode := normalizePvpMarchMode(req.MarchMode)
	nowText := now.Format(resourceDateLayout)
	attackerState, _, march, err := s.repo.CreatePvpMarchWithState(playerID, targetPlayerID, now, func(attacker *GameState, defender *GameState) (PvpMarch, error) {
		nextState, _ := settleResources(*attacker, now)
		*attacker = nextState
		if _, err := validateAndConsumeArmy(attacker, troops); err != nil {
			return PvpMarch{}, err
		}
		durationSeconds, speedMultiplier, err := calculatePvpMarchTravel(attacker.Player.Faction, troops, now, CollectModifierSources(attacker))
		if err != nil {
			return PvpMarch{}, err
		}
		arrivesAt := now.Add(time.Duration(durationSeconds) * time.Second).Format(resourceDateLayout)
		EnsureGeneralRoster(attacker, now)
		marchID := "pvp_march_" + randomID(12)
		generalIDs, err := reservePvpGenerals(attacker, req.GeneralIDs, marchID, now)
		if err != nil {
			return PvpMarch{}, err
		}
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
	if err != nil {
		return PvpAttackResponse{}, err
	}
	attackerPvpState.DailyAttackCount++
	clearBreakablePvpProtectionOnAttack(&attackerPvpState, now)
	attackerPvpState.CooldownUntil = now.Add(defaultPvpAttackCooldownSec * time.Second).Format(resourceDateLayout)
	attackerPvpState.Status = "cooldown"
	attackerPvpState.UpdatedAt = nowText
	_ = defenderPvpState
	if err := s.repo.SavePvpPlayerState(attackerPvpState, now); err != nil {
		return PvpAttackResponse{}, err
	}
	return PvpAttackResponse{March: march, Army: attackerState.Army, Generals: attackerState.Generals, ServerTime: attackerState.ServerTime}, nil
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
	return PvpMarchListResponse{Items: items}, nil
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
	return battle, nil
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
		releasePvpGenerals(attacker, march)
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
		releasePvpGenerals(attacker, march)
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
		releasePvpGenerals(attacker, march)
		nowText := now.Format(resourceDateLayout)
		if strings.TrimSpace(march.BattleID) != "" {
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
	_, _, _, battle, _, _, err := s.repo.ResolvePvpBattleTransaction(strings.TrimSpace(marchID), now, func(attacker *GameState, defender *GameState, reinforcements []Reinforcement, march *PvpMarch) (PvpBattle, BattleReport, BattleReport, []Reinforcement, error) {
		arrivesAt, err := time.Parse(resourceDateLayout, march.ArrivesAt)
		if err == nil && arrivesAt.After(now) {
			return PvpBattle{}, BattleReport{}, BattleReport{}, nil, ErrPvpMarchNotReady
		}
		EnsureGeneralRoster(attacker, now)
		EnsureGeneralRoster(defender, now)
		nextDefender, _ := settleResources(*defender, now)
		*defender = nextDefender
		result, attackerReport, defenderReport, changedReinforcements, err := resolvePvpCombat(attacker, defender, reinforcements, march, now)
		if err != nil {
			return PvpBattle{}, BattleReport{}, BattleReport{}, nil, err
		}
		nowText := now.Format(resourceDateLayout)
		if totalTroops(march.AttackTroops) > 0 {
			returnSeconds, returnSpeed, err := calculatePvpMarchTravel(attacker.Player.Faction, march.AttackTroops, now, CollectModifierSources(attacker))
			if err != nil {
				return PvpBattle{}, BattleReport{}, BattleReport{}, nil, err
			}
			march.Status = PvpMarchStatusReturning
			march.ReturnStartedAt = nowText
			march.ReturnsAt = now.Add(time.Duration(returnSeconds) * time.Second).Format(resourceDateLayout)
			march.SpeedMultiplier = returnSpeed
			updatePvpGeneralAssignmentStatus(attacker, march, PvpMarchStatusReturning)
		} else {
			march.Status = PvpMarchStatusResolved
			march.ResolvedAt = nowText
			releasePvpGenerals(attacker, march)
		}
		march.AttackerReportID = attackerReport.ID
		march.DefenderReportID = defenderReport.ID
		march.BattleID = result.ID
		march.UpdatedAt = nowText
		attacker.ServerTime = nowText
		defender.ServerTime = nowText
		return result, attackerReport, defenderReport, changedReinforcements, nil
	})
	if err == nil {
		s.applyPvpBattleStateEffects(battle, now)
	}
	return battle, err
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

// validatePvpAttackState 校验保护、冷却和每日次数。
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
	if pvpTimeAfter(attackerState.CooldownUntil, now) {
		return PvpPlayerState{}, PvpPlayerState{}, ErrPvpAttackCooldown
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
	} else if pvpTimeAfter(state.CooldownUntil, now) {
		state.Status = "cooldown"
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
func calculatePvpMarchTravel(preferredFaction string, troops map[string]int, now time.Time, sources []ModifierSource) (int, float64, error) {
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
		if minSpeed == 0 || speed < minSpeed {
			minSpeed = speed
		}
	}
	if minSpeed <= 0 {
		return 0, 0, ErrNoUnitsSelected
	}
	baseSeconds := int(math.Round(float64(defaultPvpMarchSeconds) / float64(minSpeed)))
	seconds := applySpeedBonus(baseSeconds, StatMarchSpeedBonus, now, sources)
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
func resolvePvpCombat(attacker *GameState, defender *GameState, reinforcements []Reinforcement, march *PvpMarch, now time.Time) (PvpBattle, BattleReport, BattleReport, []Reinforcement, error) {
	dispatchedTroops := cloneStringIntMap(march.AttackTroops)
	attackerUnits, err := buildSimulatedCombatUnits(attacker.Player.Faction, march.AttackTroops, now, CollectModifierSources(attacker)...)
	if err != nil {
		return PvpBattle{}, BattleReport{}, BattleReport{}, nil, err
	}
	defenderOwnTroops := armySliceToMap(defender.Army)
	defenderUnits, sourceGroups, err := buildPvpDefenderUnits(defender, reinforcements, now)
	if errors.Is(err, ErrNoUnitsSelected) {
		defenderUnits = []combat.Unit{}
		sourceGroups = []pvpDefenseSourceGroup{}
	} else if err != nil {
		return PvpBattle{}, BattleReport{}, BattleReport{}, nil, err
	}
	attackerArmy := buildCombatArmy(attacker.Player.Faction, attackerUnits)
	defenderArmy := combat.Army{Faction: defender.Player.Faction, Units: defenderUnits}
	result := combat.Resolve(combat.CombatInput{
		RuleID:   activeCombatRuleID(combatSceneForPVP(march.MarchType)),
		Attacker: attackerArmy,
		Defender: defenderArmy,
	})
	attackerLosses := combatLossMap(result.AttackerLosses)
	defenderTotalLosses := combatLossMap(result.DefenderLosses)
	attackerSurvivors := map[string]int{}
	for _, unit := range attackerUnits {
		survived := unit.Count - attackerLosses[unit.ID]
		if survived > 0 {
			attackerSurvivors[unit.ID] += survived
		}
	}
	march.AttackTroops = normalizePositiveTroops(attackerSurvivors)
	defenderLosses, reinforcementLosses := allocatePvpDefenderLosses(defenderTotalLosses, sourceGroups)
	applyArmyLosses(defender, defenderLosses)
	changedReinforcements := applyPvpReinforcementLosses(reinforcements, reinforcementLosses, now)
	plundered := map[string]int{}
	if result.Winner == "attacker" && result.SurvivingCarry > 0 {
		plundered = calculatePvpPlunder(defender, attacker, result.SurvivingCarry)
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
	reinforcementSnapshot := buildPvpReinforcementSnapshot(reinforcements)
	attackerGenerals := buildPvpGeneralSnapshots(attacker, march.AttackGenerals)
	defenderGenerals := buildPvpDefenseGeneralSnapshots(defender)
	attackerReport := buildPvpBattleReport(attackerReportID, attacker, defender, march, reportResult, int(result.AttackPower), int(result.DefensePower), dispatchedTroops, attackerLosses, defenderOwnTroops, defenderLosses, plundered, nowText, march.MarchType)
	attackerReport.PvpPointsDelta = map[string]int{"self": attackerPointsDelta, "target": defenderPointsDelta}
	attackerReport.PvpAttackerGenerals = attackerGenerals
	attackerReport.PvpDefenderGenerals = defenderGenerals
	attackerReport.PvpReinforcements = reinforcementSnapshot
	attackerReport.PvpReinforcementLosses = cloneNestedStringIntMap(reinforcementLosses)
	defenderReport := buildPvpBattleReport(defenderReportID, defender, attacker, march, invertPvpReportResult(reportResult), int(result.DefensePower), int(result.AttackPower), defenderOwnTroops, defenderLosses, dispatchedTroops, attackerLosses, map[string]int{}, nowText, "defense")
	defenderReport.PvpPointsDelta = map[string]int{"self": defenderPointsDelta, "target": attackerPointsDelta}
	defenderReport.PvpAttackerGenerals = attackerGenerals
	defenderReport.PvpDefenderGenerals = defenderGenerals
	defenderReport.PvpReinforcements = reinforcementSnapshot
	defenderReport.PvpReinforcementLosses = cloneNestedStringIntMap(reinforcementLosses)
	battle := PvpBattle{
		ID:                    battleID,
		MarchID:               march.ID,
		AttackerPlayerID:      attacker.Player.ID,
		DefenderPlayerID:      defender.Player.ID,
		Status:                PvpBattleStatusResolved,
		AttackerSnapshot:      map[string]any{"troops": cloneStringIntMap(dispatchedTroops), "faction": attacker.Player.Faction, "generals": attackerGenerals},
		DefenderSnapshot:      map[string]any{"troops": cloneStringIntMap(defenderOwnTroops), "faction": defender.Player.Faction, "generals": defenderGenerals},
		ReinforcementSnapshot: reinforcementSnapshot,
		Result:                map[string]any{"winner": result.Winner, "attackerPower": result.AttackPower, "defensePower": result.DefensePower, "pointsDelta": map[string]int{"attacker": attackerPointsDelta, "defender": defenderPointsDelta}},
		Losses:                map[string]any{"attacker": attackerLosses, "defender": defenderLosses, "reinforcements": reinforcementLosses},
		Plunder:               plundered,
		AttackerReportID:      attackerReportID,
		DefenderReportID:      defenderReportID,
		ResolvedAt:            nowText,
		CreatedAt:             nowText,
		UpdatedAt:             nowText,
	}
	return battle, attackerReport, defenderReport, changedReinforcements, nil
}

type pvpDefenseSourceGroup struct {
	Key             string
	ReinforcementID string
	UnitType        string
	Amount          int
}

func buildPvpDefenderUnits(defender *GameState, reinforcements []Reinforcement, now time.Time) ([]combat.Unit, []pvpDefenseSourceGroup, error) {
	merged := map[string]int{}
	groups := []pvpDefenseSourceGroup{}
	for _, armyUnit := range defender.Army {
		if armyUnit.Amount <= 0 {
			continue
		}
		merged[armyUnit.UnitType] += armyUnit.Amount
		groups = append(groups, pvpDefenseSourceGroup{Key: "defender", UnitType: armyUnit.UnitType, Amount: armyUnit.Amount})
	}
	for _, record := range reinforcements {
		normalizeGarrisonRecord(&record)
		for unitType, amount := range record.RemainingTroops {
			if amount <= 0 {
				continue
			}
			merged[unitType] += amount
			groups = append(groups, pvpDefenseSourceGroup{Key: "reinforcement", ReinforcementID: record.ID, UnitType: unitType, Amount: amount})
		}
	}
	units := []combat.Unit{}
	for unitType, amount := range merged {
		unitCfg, faction, ok := findAnyUnitConfig(defender.Player.Faction, unitType)
		if !ok {
			return nil, nil, ErrUnitNotFound
		}
		units = append(units, buildCombatUnitFromConfig(unitType, amount, unitCfg, now, CollectModifierSources(defender)...))
		_ = faction
	}
	if len(units) == 0 {
		return nil, nil, ErrNoUnitsSelected
	}
	return units, groups, nil
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

func applyPvpReinforcementLosses(records []Reinforcement, losses map[string]map[string]int, now time.Time) []Reinforcement {
	changed := []Reinforcement{}
	nowText := now.Format(resourceDateLayout)
	for _, record := range records {
		unitLosses := losses[record.ID]
		if len(unitLosses) == 0 {
			continue
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

func calculatePvpPlunder(defender *GameState, attacker *GameState, carryCapacity int) map[string]int {
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
	for resourceType, amount := range plundered {
		defender.Resources.Items[resourceType] -= amount
		if defender.Resources.Items[resourceType] < 0 {
			defender.Resources.Items[resourceType] = 0
		}
		_, _, _ = addResourceCapped(attacker, resourceType, amount)
	}
	return plundered
}

func buildPvpBattleReport(id string, owner *GameState, target *GameState, march *PvpMarch, result string, playerPower int, enemyPower int, dispatched map[string]int, lost map[string]int, defenderUnits map[string]int, defenderLost map[string]int, rewards map[string]int, nowText string, reportType string) BattleReport {
	return BattleReport{
		ID:                id,
		PlayerID:          owner.Player.ID,
		PlayerFaction:     owner.Player.Faction,
		PlayerName:        owner.Player.Nickname,
		TargetID:          target.Player.ID,
		TargetName:        target.Player.Nickname + "（玩家）",
		Type:              reportType,
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
		return "defender_defeat"
	case "defender_victory":
		return "defender_victory"
	default:
		return result
	}
}

func buildPvpReinforcementSnapshot(records []Reinforcement) []DefenseReinforcementUnit {
	result := []DefenseReinforcementUnit{}
	for _, record := range records {
		normalizeGarrisonRecord(&record)
		result = append(result, DefenseReinforcementUnit{
			ReinforcementID: record.ID,
			FromPlayerID:    record.FromPlayerID,
			Faction:         record.FromPlayerFaction,
			Troops:          cloneStringIntMap(record.RemainingTroops),
			Generals:        append([]ReinforcementGeneralSnapshot(nil), record.Generals...),
			Buffs:           append([]ModifierBreakdownItem(nil), record.BuffSnapshot...),
			SourceTags:      map[string]string{"source_type": record.SourceType, "source_id": record.SourceID},
		})
	}
	return result
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
	if state == nil {
		return nil
	}
	if state.General == nil {
		return nil
	}
	return []PvpGeneralSnapshot{snapshotPvpGeneral(*state.General)}
}

// snapshotPvpGeneral 复制 PVP 战斗中需要展示的武将字段。
func snapshotPvpGeneral(general General) PvpGeneralSnapshot {
	return PvpGeneralSnapshot{
		ID:    general.ID,
		Name:  general.Name,
		Level: general.Level,
		Buffs: cloneFloatMap(general.Buffs),
	}
}
