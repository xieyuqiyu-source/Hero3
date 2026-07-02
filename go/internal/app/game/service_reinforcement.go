// 本文件实现增援系统的应用服务，负责兵力扣出、武将占用、生命周期和战斗接入。
package game

import (
	"encoding/json"
	"math"
	"sort"
	"strings"
	"time"
)

// SendReinforcement 发起一批前往目标玩家城池的增援。
func (s *Service) SendReinforcement(req SendReinforcementRequest) (ReinforcementResponse, error) {
	fromPlayerID := strings.TrimSpace(req.FromPlayerID)
	toPlayerID := strings.TrimSpace(req.TargetPlayerID)
	if fromPlayerID == "" || toPlayerID == "" {
		return ReinforcementResponse{}, ErrPlayerNotFound
	}
	if strings.HasPrefix(toPlayerID, "npc") {
		return ReinforcementResponse{}, ErrReinforcementTargetNPC
	}
	if fromPlayerID == toPlayerID {
		return ReinforcementResponse{}, ErrReinforcementTargetSelf
	}
	troops := normalizePositiveTroops(req.Troops)
	if len(troops) == 0 {
		return ReinforcementResponse{}, ErrNoUnitsSelected
	}
	speed := normalizeReinforcementSpeed(req.SpeedMultiplier)
	now := time.Now()
	nowText := now.UTC().Format(resourceDateLayout)
	fromPosition, err := s.ensureWorldPosition(fromPlayerID, "lazy_create", nil)
	if err != nil {
		return ReinforcementResponse{}, err
	}
	toPosition, err := s.ensureWorldPosition(toPlayerID, "lazy_create", nil)
	if err != nil {
		return ReinforcementResponse{}, err
	}
	distance := worldMapDistance(WorldCoordinate{X: fromPosition.X, Y: fromPosition.Y}, WorldCoordinate{X: toPosition.X, Y: toPosition.Y})

	fromState, _, record, err := s.repo.CreateReinforcementWithState(fromPlayerID, toPlayerID, now, func(from *GameState, to *GameState, targetRecords []Reinforcement) (Reinforcement, error) {
		if err := ensureReinforcementSourceSlot(fromPlayerID, targetRecords); err != nil {
			return Reinforcement{}, err
		}
		if _, err := validateAndConsumeArmy(from, troops); err != nil {
			return Reinforcement{}, err
		}
		EnsureGeneralRoster(from, now)
		reinforcementID := "reinforcement_" + randomID(12)
		generals, err := reserveReinforcementGenerals(from, req.GeneralIDs, reinforcementID, now)
		if err != nil {
			return Reinforcement{}, err
		}
		generalIDs := make([]string, 0, len(generals))
		for _, item := range generals {
			generalIDs = append(generalIDs, item.ID)
		}
		marchSeconds := reinforcementTravelSecondsForDistance(distance, reinforcementSlowestUnitSpeed(from.Player.Faction, troops), now, CollectModifierSources(from))
		marchSeconds = dispatchMarchCreateTraits(marchSeconds, "reinforcement", from, generalIDs)
		expectedArriveAt := now.Add(time.Duration(marchSeconds) * time.Second).UTC().Format(resourceDateLayout)
		record := Reinforcement{
			ID:                reinforcementID,
			FromPlayerID:      from.Player.ID,
			FromPlayerName:    from.Player.Nickname,
			FromPlayerFaction: from.Player.Faction,
			ToPlayerID:        to.Player.ID,
			ToPlayerName:      to.Player.Nickname,
			ToPlayerFaction:   to.Player.Faction,
			OwnerPlayerID:     from.Player.ID,
			HostPlayerID:      to.Player.ID,
			SourceType:        GarrisonSourceReinforcement,
			SourceID:          reinforcementID,
			TargetType:        ReinforcementTargetPlayerCity,
			TargetID:          to.Player.ID,
			Status:            ReinforcementStatusMarching,
			Troops:            cloneStringIntMap(troops),
			RemainingTroops:   cloneStringIntMap(troops),
			Generals:          generals,
			BuffSnapshot:      reinforcementBuffSnapshot(generals),
			Rules:             defaultGarrisonRules(GarrisonSourceReinforcement),
			SpeedMultiplier:   speed,
			MarchSeconds:      marchSeconds,
			ReturnSeconds:     marchSeconds,
			SentAt:            nowText,
			ExpectedArriveAt:  expectedArriveAt,
			Losses:            map[string]int{},
			RewardState:       map[string]any{},
			MailState:         map[string]any{"sent": true},
			Metadata:          map[string]any{},
			CreatedAt:         nowText,
			UpdatedAt:         nowText,
		}
		from.ServerTime = nowText
		return record, nil
	})
	if err != nil {
		return ReinforcementResponse{}, err
	}
	return ReinforcementResponse{Reinforcement: record, Patch: BuildGarrisonActionResult(fromState)}, nil
}

// CreateGarrisonDetachment 创建一批非增援来源的驻防队伍，不写入玩家常规军队。
func (s *Service) CreateGarrisonDetachment(req CreateGarrisonDetachmentRequest) (ReinforcementResponse, error) {
	ownerPlayerID := strings.TrimSpace(req.OwnerPlayerID)
	hostPlayerID := strings.TrimSpace(req.HostPlayerID)
	if ownerPlayerID == "" || hostPlayerID == "" {
		return ReinforcementResponse{}, ErrPlayerNotFound
	}
	sourceType := normalizeGarrisonSourceType(req.SourceType)
	if sourceType == GarrisonSourceReinforcement {
		return ReinforcementResponse{}, ErrInvalidReinforcement
	}
	sourceType = GarrisonSourceObtained
	troops := normalizePositiveTroops(req.Troops)
	if len(troops) == 0 {
		return ReinforcementResponse{}, ErrNoUnitsSelected
	}
	now := time.Now()
	nowText := now.UTC().Format(resourceDateLayout)
	if sourceType == GarrisonSourceObtained {
		existing, err := s.findObtainedGarrison(ownerPlayerID, hostPlayerID)
		if err != nil {
			return ReinforcementResponse{}, err
		}
		if existing.ID != "" {
			ownerState, _, record, err := s.repo.UpdateReinforcement(existing.ID, now, func(owner *GameState, host *GameState, record *Reinforcement) error {
				normalizeGarrisonRecord(record)
				record.ID = obtainedGarrisonID(owner.Player.ID)
				record.FromPlayerID = owner.Player.ID
				record.ToPlayerID = host.Player.ID
				record.OwnerPlayerID = owner.Player.ID
				record.HostPlayerID = host.Player.ID
				record.SourceType = GarrisonSourceObtained
				record.SourceID = GarrisonSourceObtained
				record.FromPlayerFaction = firstNonEmpty(strings.TrimSpace(req.SourceFaction), record.FromPlayerFaction, owner.Player.Faction)
				record.Status = ReinforcementStatusStationed
				record.Rules = defaultGarrisonRules(GarrisonSourceObtained)
				record.Troops = mergeTroopMaps(record.Troops, troops)
				record.RemainingTroops = mergeTroopMaps(record.RemainingTroops, troops)
				record.Metadata = mergeGarrisonMetadata(record.Metadata, req.Metadata)
				record.UpdatedAt = nowText
				if record.ArrivedAt == "" {
					record.ArrivedAt = nowText
				}
				owner.ServerTime = nowText
				if owner.Player.ID != host.Player.ID {
					host.ServerTime = nowText
				}
				return nil
			})
			if err != nil {
				return ReinforcementResponse{}, err
			}
			return ReinforcementResponse{Reinforcement: record, Patch: BuildGarrisonActionResult(ownerState)}, nil
		}
	}
	ownerState, _, record, err := s.repo.CreateReinforcementWithState(ownerPlayerID, hostPlayerID, now, func(owner *GameState, host *GameState, targetRecords []Reinforcement) (Reinforcement, error) {
		detachmentID := "garrison_" + randomID(12)
		if sourceType == GarrisonSourceObtained {
			detachmentID = obtainedGarrisonID(owner.Player.ID)
		}
		sourceFaction := strings.TrimSpace(req.SourceFaction)
		if sourceFaction == "" {
			sourceFaction = owner.Player.Faction
		}
		record := Reinforcement{
			ID:                detachmentID,
			FromPlayerID:      owner.Player.ID,
			FromPlayerName:    owner.Player.Nickname,
			FromPlayerFaction: sourceFaction,
			ToPlayerID:        host.Player.ID,
			ToPlayerName:      host.Player.Nickname,
			ToPlayerFaction:   host.Player.Faction,
			OwnerPlayerID:     owner.Player.ID,
			HostPlayerID:      host.Player.ID,
			SourceType:        GarrisonSourceObtained,
			SourceID:          strings.TrimSpace(req.SourceID),
			TargetType:        ReinforcementTargetPlayerCity,
			TargetID:          host.Player.ID,
			Status:            ReinforcementStatusStationed,
			Troops:            cloneStringIntMap(troops),
			RemainingTroops:   cloneStringIntMap(troops),
			Rules:             defaultGarrisonRules(sourceType),
			SpeedMultiplier:   1,
			MarchSeconds:      0,
			ReturnSeconds:     0,
			SentAt:            nowText,
			ArrivedAt:         nowText,
			Losses:            map[string]int{},
			RewardState:       map[string]any{},
			MailState:         map[string]any{},
			Metadata:          cloneAnyMap(req.Metadata),
			CreatedAt:         nowText,
			UpdatedAt:         nowText,
		}
		if record.SourceID == "" {
			record.SourceID = detachmentID
		}
		record.SourceID = GarrisonSourceObtained
		owner.ServerTime = nowText
		if owner.Player.ID != host.Player.ID {
			host.ServerTime = nowText
		}
		return record, nil
	})
	if err != nil {
		return ReinforcementResponse{}, err
	}
	return ReinforcementResponse{Reinforcement: record, Patch: BuildGarrisonActionResult(ownerState)}, nil
}

// ListSentReinforcements 返回玩家派出的增援，并请求触发到达/返程结算。
func (s *Service) ListSentReinforcements(playerID string) (ReinforcementListResponse, error) {
	playerID = strings.TrimSpace(playerID)
	if playerID == "" {
		return ReinforcementListResponse{}, ErrPlayerNotFound
	}
	if err := s.SettleReinforcementsForPlayer(playerID); err != nil {
		return ReinforcementListResponse{}, err
	}
	items, err := s.repo.ListSentReinforcements(playerID)
	if err != nil {
		return ReinforcementListResponse{}, err
	}
	normalizeGarrisonRecords(items)
	items = filterTrueReinforcements(items)
	sortReinforcements(items)
	return ReinforcementListResponse{Items: items}, nil
}

// ListReceivedReinforcements 返回玩家收到的增援，并请求触发到达/返程结算。
func (s *Service) ListReceivedReinforcements(playerID string) (ReinforcementListResponse, error) {
	playerID = strings.TrimSpace(playerID)
	if playerID == "" {
		return ReinforcementListResponse{}, ErrPlayerNotFound
	}
	if err := s.SettleReinforcementsForPlayer(playerID); err != nil {
		return ReinforcementListResponse{}, err
	}
	items, err := s.repo.ListReceivedReinforcements(playerID)
	if err != nil {
		return ReinforcementListResponse{}, err
	}
	normalizeGarrisonRecords(items)
	items = filterReceivedGarrisonRecords(items)
	items = aggregateObtainedGarrisons(playerID, items)
	sortReinforcements(items)
	return ReinforcementListResponse{Items: items}, nil
}

// GetReinforcement 返回单个增援批次详情。
func (s *Service) GetReinforcement(playerID string, reinforcementID string) (Reinforcement, error) {
	record, err := s.repo.GetReinforcement(strings.TrimSpace(reinforcementID))
	if err != nil {
		return Reinforcement{}, err
	}
	normalizeGarrisonRecord(&record)
	playerID = strings.TrimSpace(playerID)
	if playerID != "" && playerID != record.FromPlayerID && playerID != record.ToPlayerID && playerID != record.OwnerPlayerID && playerID != record.HostPlayerID {
		return Reinforcement{}, ErrReinforcementNotFound
	}
	return record, nil
}

// RecallReinforcement 由派出方召回援军。
func (s *Service) RecallReinforcement(playerID string, reinforcementID string) (ReinforcementResponse, error) {
	playerID = strings.TrimSpace(playerID)
	now := time.Now()
	from, _, record, err := s.repo.UpdateReinforcement(strings.TrimSpace(reinforcementID), now, func(from *GameState, to *GameState, record *Reinforcement) error {
		normalizeGarrisonRecord(record)
		if record.FromPlayerID != playerID {
			return ErrReinforcementNotFound
		}
		if !record.Rules.CanRecall {
			return ErrInvalidReinforcement
		}
		if err := startReinforcementReturn(record, ReinforcementStatusReturning, "recalled", now); err != nil {
			return err
		}
		from.ServerTime = now.UTC().Format(resourceDateLayout)
		return nil
	})
	if err != nil {
		return ReinforcementResponse{}, err
	}
	return ReinforcementResponse{Reinforcement: record, Patch: BuildGarrisonActionResult(from)}, nil
}

// ExpelReinforcement 由接收方遣返援军。
func (s *Service) ExpelReinforcement(playerID string, reinforcementID string) (ReinforcementResponse, error) {
	playerID = strings.TrimSpace(playerID)
	now := time.Now()
	_, to, record, err := s.repo.UpdateReinforcement(strings.TrimSpace(reinforcementID), now, func(from *GameState, to *GameState, record *Reinforcement) error {
		normalizeGarrisonRecord(record)
		if record.ToPlayerID != playerID {
			return ErrReinforcementNotFound
		}
		if !record.Rules.CanExpel {
			return ErrInvalidReinforcement
		}
		if err := startReinforcementReturn(record, ReinforcementStatusReturning, "expelled", now); err != nil {
			return err
		}
		to.ServerTime = now.UTC().Format(resourceDateLayout)
		return nil
	})
	if err != nil {
		return ReinforcementResponse{}, err
	}
	return ReinforcementResponse{Reinforcement: record, Patch: BuildGarrisonActionResult(to)}, nil
}

// AccelerateReinforcement 使用城金加速自己派出的行军中援军。
func (s *Service) AccelerateReinforcement(playerID string, reinforcementID string) (ReinforcementActionResponse, error) {
	playerID = strings.TrimSpace(playerID)
	if playerID == "" {
		return ReinforcementActionResponse{}, ErrPlayerNotFound
	}
	now := time.Now().UTC()
	cost := 0
	from, _, record, err := s.repo.UpdateReinforcement(strings.TrimSpace(reinforcementID), now, func(from *GameState, to *GameState, record *Reinforcement) error {
		normalizeGarrisonRecord(record)
		if record.FromPlayerID != playerID {
			return ErrReinforcementNotFound
		}
		if record.SourceType != GarrisonSourceReinforcement || record.Status != ReinforcementStatusMarching {
			return ErrReinforcementNotAccelerable
		}
		if reinforcementAcceleratedTimes(record.Metadata) >= pvpMaxAccelerateTimes {
			return ErrReinforcementNotAccelerable
		}
		arrivesAt, err := time.Parse(resourceDateLayout, strings.TrimSpace(record.ExpectedArriveAt))
		if err != nil || !arrivesAt.After(now) {
			return ErrReinforcementNotAccelerable
		}
		remainingSeconds := int(math.Ceil(arrivesAt.Sub(now).Seconds()))
		if remainingSeconds <= 1 {
			return ErrReinforcementNotAccelerable
		}
		cost = pvpAccelerateFixedCityGoldCost
		if int(from.CityGold) < cost {
			return ErrInsufficientCityGold
		}
		sentAt, err := time.Parse(resourceDateLayout, strings.TrimSpace(record.SentAt))
		if err != nil {
			return ErrReinforcementNotAccelerable
		}
		elapsedSeconds := int(math.Ceil(now.Sub(sentAt).Seconds()))
		if elapsedSeconds < 0 {
			elapsedSeconds = 0
		}
		nextRemainingSeconds := (remainingSeconds + 1) / 2
		nextMarchSeconds := elapsedSeconds + nextRemainingSeconds
		if nextMarchSeconds < 1 {
			nextMarchSeconds = 1
		}
		from.CityGold -= FlexInt(cost)
		record.MarchSeconds = nextMarchSeconds
		record.ReturnSeconds = nextMarchSeconds
		record.ExpectedArriveAt = now.Add(time.Duration(nextRemainingSeconds) * time.Second).Format(resourceDateLayout)
		record.SpeedMultiplier = calculateReinforcementSpeedMultiplier(nextMarchSeconds)
		record.Metadata = appendReinforcementAccelerateMetadata(record.Metadata, now, cost, remainingSeconds, nextRemainingSeconds)
		record.UpdatedAt = now.Format(resourceDateLayout)
		from.ServerTime = now.Format(resourceDateLayout)
		return nil
	})
	if err != nil {
		return ReinforcementActionResponse{}, err
	}
	if cost > 0 {
		s.recordLedger(GoldLedgerEntry{
			PlayerID:     playerID,
			Currency:     LedgerCurrencyCityGold,
			Direction:    LedgerDirectionDebit,
			Amount:       cost,
			BalanceAfter: int(from.CityGold),
			RefType:      LedgerRefReinforcementAccelerate,
			RefID:        record.ID,
			Reason:       "reinforcement_accelerate",
		})
		s.publishCurrencyChanged(playerID, "", record.ID, LedgerRefReinforcementAccelerate)
	}
	return ReinforcementActionResponse{
		Reinforcement: record,
		Patch:         BuildGarrisonActionResult(from),
		CityGold:      from.CityGold,
		Cost:          cost,
		ServerTime:    from.ServerTime,
	}, nil
}

// MarkReinforcementArrived 把到达时间已满足的援军转为驻扎。
func (s *Service) MarkReinforcementArrived(reinforcementID string) (Reinforcement, error) {
	now := time.Now()
	_, _, record, err := s.repo.UpdateReinforcement(strings.TrimSpace(reinforcementID), now, func(from *GameState, to *GameState, record *Reinforcement) error {
		normalizeGarrisonRecord(record)
		return markReinforcementArrived(record, now)
	})
	return record, err
}

// CompleteReinforcementReturn 完成返程，返还剩余兵力并释放武将。
func (s *Service) CompleteReinforcementReturn(reinforcementID string) (Reinforcement, error) {
	now := time.Now()
	_, _, record, err := s.repo.UpdateReinforcement(strings.TrimSpace(reinforcementID), now, func(from *GameState, to *GameState, record *Reinforcement) error {
		normalizeGarrisonRecord(record)
		return completeReinforcementReturn(from, record, now)
	})
	return record, err
}

// SettleReinforcementsForPlayer 请求触发结算某个玩家相关的到达和返程。
func (s *Service) SettleReinforcementsForPlayer(playerID string) error {
	sent, err := s.repo.ListSentReinforcements(playerID)
	if err != nil {
		return err
	}
	received, err := s.repo.ListReceivedReinforcements(playerID)
	if err != nil {
		return err
	}
	seen := map[string]bool{}
	for _, record := range append(sent, received...) {
		normalizeGarrisonRecord(&record)
		if seen[record.ID] {
			continue
		}
		seen[record.ID] = true
		if record.Status == ReinforcementStatusMarching && reinforcementDue(record.SentAt, record.MarchSeconds, time.Now()) {
			if _, err := s.MarkReinforcementArrived(record.ID); err != nil {
				return err
			}
		}
		if record.Status == ReinforcementStatusReturning && reinforcementDue(record.ReturnStartedAt, record.ReturnSeconds, time.Now()) {
			if _, err := s.CompleteReinforcementReturn(record.ID); err != nil {
				return err
			}
		}
	}
	return nil
}

// BuildDefenseReinforcementUnits 构建目标玩家当前可参战的驻扎援军。
func (s *Service) BuildDefenseReinforcementUnits(targetPlayerID string) ([]DefenseReinforcementUnit, error) {
	targetPlayerID = strings.TrimSpace(targetPlayerID)
	if targetPlayerID == "" {
		return nil, ErrPlayerNotFound
	}
	if err := s.SettleReinforcementsForPlayer(targetPlayerID); err != nil {
		return nil, err
	}
	records, err := s.repo.ListReceivedReinforcements(targetPlayerID)
	if err != nil {
		return nil, err
	}
	units := []DefenseReinforcementUnit{}
	for _, record := range records {
		normalizeGarrisonRecord(&record)
		if record.Status != ReinforcementStatusStationed {
			continue
		}
		if !record.Rules.CanFight {
			continue
		}
		units = append(units, DefenseReinforcementUnit{
			ReinforcementID: record.ID,
			FromPlayerID:    record.OwnerPlayerID,
			Faction:         record.FromPlayerFaction,
			Troops:          cloneStringIntMap(record.RemainingTroops),
			Generals:        cloneReinforcementGenerals(record.Generals),
			Buffs:           append([]ModifierBreakdownItem(nil), record.BuffSnapshot...),
			SourceTags: map[string]string{
				"source_type":      record.SourceType,
				"source_player_id": record.OwnerPlayerID,
				"source_record_id": record.ID,
				"source_id":        record.SourceID,
			},
		})
	}
	return units, nil
}

// ApplyReinforcementBattleResult 应用战后援军损耗并保留战报关联。
func (s *Service) ApplyReinforcementBattleResult(reportID string, losses []ReinforcementLoss) error {
	now := time.Now()
	grouped := map[string][]ReinforcementLoss{}
	for _, loss := range losses {
		if strings.TrimSpace(loss.ReinforcementID) == "" || loss.LostAmount <= 0 {
			continue
		}
		grouped[loss.ReinforcementID] = append(grouped[loss.ReinforcementID], loss)
	}
	for reinforcementID, items := range grouped {
		if _, _, _, err := s.repo.UpdateReinforcement(reinforcementID, now, func(from *GameState, to *GameState, record *Reinforcement) error {
			normalizeGarrisonRecord(record)
			if record.Status != ReinforcementStatusStationed && record.Status != ReinforcementStatusFighting {
				return nil
			}
			record.Status = ReinforcementStatusFighting
			if record.Losses == nil {
				record.Losses = map[string]int{}
			}
			for _, item := range items {
				unitType := strings.TrimSpace(item.UnitType)
				if unitType == "" || item.LostAmount <= 0 {
					continue
				}
				current := record.RemainingTroops[unitType]
				lost := item.LostAmount
				if lost > current {
					lost = current
				}
				record.RemainingTroops[unitType] = current - lost
				record.Losses[unitType] += lost
			}
			cleanIntMap(record.RemainingTroops)
			record.LastBattleReportID = strings.TrimSpace(reportID)
			record.LastBattleAt = now.UTC().Format(resourceDateLayout)
			record.UpdatedAt = record.LastBattleAt
			if totalTroops(record.RemainingTroops) <= 0 {
				record.IsAnnihilated = true
				if !record.Rules.CanReturn {
					record.Status = ReinforcementStatusCompleted
					record.ReturnedAt = now.UTC().Format(resourceDateLayout)
					return nil
				}
				return startReinforcementReturn(record, ReinforcementStatusReturning, "annihilated", now)
			}
			record.Status = ReinforcementStatusStationed
			return nil
		}); err != nil {
			return err
		}
	}
	return nil
}

func normalizePositiveTroops(troops map[string]int) map[string]int {
	result := map[string]int{}
	for unitType, amount := range troops {
		unitType = strings.TrimSpace(unitType)
		if unitType != "" && amount > 0 {
			result[unitType] += amount
		}
	}
	return result
}

func normalizeReinforcementSpeed(speed float64) float64 {
	return 1
}

func reinforcementTravelSeconds(speed float64, now time.Time, sources []ModifierSource) int {
	baseSeconds := int(math.Round(float64(defaultReinforcementMarchSeconds) / normalizeReinforcementSpeed(speed)))
	seconds := applySpeedBonus(baseSeconds, StatMarchSpeedBonus, now, sources)
	seconds = clampInt(seconds, minReinforcementMarchSeconds, maxReinforcementMarchSeconds)
	if seconds <= 0 {
		return 1
	}
	return seconds
}

// reinforcementTravelSecondsForDistance 按地图距离缩放增援行军时间。
func reinforcementTravelSecondsForDistance(distance int, unitSpeed float64, now time.Time, sources []ModifierSource) int {
	if distance < 1 {
		distance = 1
	}
	if unitSpeed < 1 {
		unitSpeed = 1
	}
	return CalculateWorldMarchSeconds(distance, int(math.Floor(unitSpeed)), now, sources)
}

// reinforcementSlowestUnitSpeed 返回携带部队中最慢兵种速度，缺省按速度 1 处理。
func reinforcementSlowestUnitSpeed(preferredFaction string, troops map[string]int) float64 {
	slowest := 0
	for unitType, amount := range troops {
		if strings.TrimSpace(unitType) == "" || amount <= 0 {
			continue
		}
		speed := 1
		if cfg, _, ok := findAnyUnitConfig(preferredFaction, unitType); ok && cfg.Stats["speed"] > 0 {
			speed = cfg.Stats["speed"]
		}
		if slowest == 0 || speed < slowest {
			slowest = speed
		}
	}
	if slowest <= 0 {
		return 1
	}
	return float64(slowest)
}

func ensureReinforcementSourceSlot(fromPlayerID string, records []Reinforcement) error {
	sources := map[string]bool{}
	for _, record := range records {
		normalizeGarrisonRecord(&record)
		if !reinforcementOccupiesSlot(record.Status) {
			continue
		}
		if record.SourceType == GarrisonSourceObtained {
			sources[record.HostPlayerID] = true
			continue
		}
		sources[record.OwnerPlayerID] = true
	}
	if sources[fromPlayerID] {
		return nil
	}
	if len(sources) >= defaultReinforcementMaxSources {
		return ErrReinforcementSlotFull
	}
	return nil
}

func reinforcementOccupiesSlot(status string) bool {
	return status == ReinforcementStatusMarching || status == ReinforcementStatusStationed || status == ReinforcementStatusFighting
}

// obtainedGarrisonID 返回玩家自己的获得驻防队伍固定 ID。
func obtainedGarrisonID(playerID string) string {
	return "garrison_obtained_" + strings.TrimSpace(playerID)
}

// findObtainedGarrison 查找玩家自己的“获得”驻防队伍，用于后续兵种合并。
func (s *Service) findObtainedGarrison(ownerPlayerID string, hostPlayerID string) (Reinforcement, error) {
	items, err := s.repo.ListReceivedReinforcements(hostPlayerID)
	if err != nil {
		return Reinforcement{}, err
	}
	stableID := obtainedGarrisonID(ownerPlayerID)
	fallback := Reinforcement{}
	for _, item := range items {
		normalizeGarrisonRecord(&item)
		if !reinforcementOccupiesSlot(item.Status) {
			continue
		}
		if item.ID == stableID {
			return item, nil
		}
		if fallback.ID == "" && item.OwnerPlayerID == ownerPlayerID && item.HostPlayerID == hostPlayerID && item.SourceType == GarrisonSourceObtained {
			fallback = item
		}
	}
	return fallback, nil
}

func normalizeGarrisonSourceType(sourceType string) string {
	switch strings.TrimSpace(sourceType) {
	case GarrisonSourceObtained:
		return GarrisonSourceObtained
	case GarrisonSourceCaptured:
		return GarrisonSourceObtained
	case GarrisonSourceMercenary:
		return GarrisonSourceObtained
	case GarrisonSourceEventReward:
		return GarrisonSourceObtained
	case GarrisonSourceSystem:
		return GarrisonSourceObtained
	case GarrisonSourceReinforcement:
		return GarrisonSourceReinforcement
	default:
		return GarrisonSourceObtained
	}
}

func defaultGarrisonRules(sourceType string) GarrisonRules {
	switch normalizeGarrisonSourceType(sourceType) {
	case GarrisonSourceReinforcement:
		return GarrisonRules{CanRecall: true, CanExpel: true, CanReturn: true, CanFight: true}
	case GarrisonSourceObtained:
		return GarrisonRules{CanFight: true, CanConvert: true, CanRelease: true}
	default:
		return GarrisonRules{CanFight: true}
	}
}

func normalizeGarrisonRecord(record *Reinforcement) {
	if record == nil {
		return
	}
	if record.OwnerPlayerID == "" {
		record.OwnerPlayerID = record.FromPlayerID
	}
	if record.HostPlayerID == "" {
		record.HostPlayerID = record.ToPlayerID
	}
	if record.SourceType == "" {
		record.SourceType = GarrisonSourceReinforcement
	}
	record.SourceType = normalizeGarrisonSourceType(record.SourceType)
	if record.SourceID == "" {
		record.SourceID = record.ID
	}
	if !record.Rules.CanRecall && !record.Rules.CanExpel && !record.Rules.CanReturn && !record.Rules.CanFight && !record.Rules.CanConvert && !record.Rules.CanRelease {
		record.Rules = defaultGarrisonRules(record.SourceType)
	}
	normalizeReinforcementTiming(record)
}

// normalizeReinforcementTiming 修正历史客户端倍率导致的异常长行军时间。
func normalizeReinforcementTiming(record *Reinforcement) {
	if record == nil || record.SourceType != GarrisonSourceReinforcement {
		return
	}
	if record.MarchSeconds > maxReinforcementMarchSeconds {
		record.MarchSeconds = maxReinforcementMarchSeconds
	}
	if record.ReturnSeconds > maxReinforcementMarchSeconds {
		record.ReturnSeconds = maxReinforcementMarchSeconds
	}
	if record.Status == ReinforcementStatusMarching && record.SentAt != "" && record.MarchSeconds > 0 {
		if sentAt, err := time.Parse(resourceDateLayout, strings.TrimSpace(record.SentAt)); err == nil {
			record.ExpectedArriveAt = sentAt.Add(time.Duration(record.MarchSeconds) * time.Second).UTC().Format(resourceDateLayout)
		}
	}
	if record.Status == ReinforcementStatusReturning && record.ReturnStartedAt != "" && record.ReturnSeconds > 0 {
		if startedAt, err := time.Parse(resourceDateLayout, strings.TrimSpace(record.ReturnStartedAt)); err == nil {
			record.ExpectedReturnedAt = startedAt.Add(time.Duration(record.ReturnSeconds) * time.Second).UTC().Format(resourceDateLayout)
		}
	}
}

func normalizeGarrisonRecords(records []Reinforcement) {
	for i := range records {
		normalizeGarrisonRecord(&records[i])
	}
}

func filterTrueReinforcements(records []Reinforcement) []Reinforcement {
	result := make([]Reinforcement, 0, len(records))
	for _, record := range records {
		if record.SourceType == GarrisonSourceReinforcement && isVisibleReinforcementStatus(record.Status) {
			result = append(result, record)
		}
	}
	return result
}

// filterReceivedGarrisonRecords 过滤被增援方可见驻防，返程队伍只归派出方可见。
func filterReceivedGarrisonRecords(records []Reinforcement) []Reinforcement {
	result := make([]Reinforcement, 0, len(records))
	for _, record := range records {
		if !isVisibleReceivedGarrisonStatus(record.Status) {
			continue
		}
		result = append(result, record)
	}
	return result
}

// isVisibleReinforcementStatus 判断派出方增援状态面板需要展示的进行中状态。
func isVisibleReinforcementStatus(status string) bool {
	return status == ReinforcementStatusMarching ||
		status == ReinforcementStatusStationed ||
		status == ReinforcementStatusFighting ||
		status == ReinforcementStatusReturning
}

// isVisibleReceivedGarrisonStatus 判断被增援方仍可管理的驻防状态。
func isVisibleReceivedGarrisonStatus(status string) bool {
	return status == ReinforcementStatusMarching ||
		status == ReinforcementStatusStationed ||
		status == ReinforcementStatusFighting
}

func aggregateObtainedGarrisons(playerID string, records []Reinforcement) []Reinforcement {
	result := make([]Reinforcement, 0, len(records))
	var obtained *Reinforcement
	for _, record := range records {
		normalizeGarrisonRecord(&record)
		if record.SourceType != GarrisonSourceObtained || record.OwnerPlayerID != playerID || record.HostPlayerID != playerID {
			result = append(result, record)
			continue
		}
		if obtained == nil {
			next := record
			next.ID = obtainedGarrisonID(playerID)
			next.SourceType = GarrisonSourceObtained
			next.SourceID = GarrisonSourceObtained
			next.Troops = cloneStringIntMap(record.Troops)
			next.RemainingTroops = cloneStringIntMap(record.RemainingTroops)
			obtained = &next
			continue
		}
		obtained.Troops = mergeTroopMaps(obtained.Troops, record.Troops)
		obtained.RemainingTroops = mergeTroopMaps(obtained.RemainingTroops, record.RemainingTroops)
		if record.UpdatedAt > obtained.UpdatedAt {
			obtained.UpdatedAt = record.UpdatedAt
		}
		if record.CreatedAt < obtained.CreatedAt {
			obtained.CreatedAt = record.CreatedAt
		}
	}
	if obtained != nil {
		result = append(result, *obtained)
	}
	return result
}

func cloneAnyMap(src map[string]any) map[string]any {
	if len(src) == 0 {
		return map[string]any{}
	}
	dst := make(map[string]any, len(src))
	for key, value := range src {
		dst[key] = value
	}
	return dst
}

func reinforcementAcceleratedTimes(metadata map[string]any) int {
	if len(metadata) == 0 {
		return 0
	}
	switch value := metadata["acceleratedTimes"].(type) {
	case int:
		return value
	case int64:
		return int(value)
	case float64:
		return int(value)
	case json.Number:
		n, _ := value.Int64()
		return int(n)
	default:
		return 0
	}
}

func appendReinforcementAccelerateMetadata(metadata map[string]any, now time.Time, cost int, beforeRemaining int, afterRemaining int) map[string]any {
	next := cloneAnyMap(metadata)
	times := reinforcementAcceleratedTimes(next) + 1
	next["acceleratedTimes"] = times
	logs := []any{}
	if existing, ok := next["accelerateLogs"].([]any); ok {
		logs = append(logs, existing...)
	}
	logs = append(logs, map[string]any{
		"acceleratedAt":    now.Format(resourceDateLayout),
		"cost":             cost,
		"remainingBefore":  beforeRemaining,
		"remainingAfter":   afterRemaining,
		"acceleratedTimes": times,
	})
	next["accelerateLogs"] = logs
	return next
}

func calculateReinforcementSpeedMultiplier(marchSeconds int) float64 {
	if marchSeconds <= 0 {
		return 1
	}
	return math.Round((float64(defaultReinforcementMarchSeconds)/float64(marchSeconds))*100) / 100
}

func reserveReinforcementGenerals(state *GameState, generalIDs []string, reinforcementID string, now time.Time) ([]ReinforcementGeneralSnapshot, error) {
	result := []ReinforcementGeneralSnapshot{}
	seen := map[string]bool{}
	for _, generalID := range generalIDs {
		generalID = strings.TrimSpace(generalID)
		if generalID == "" || seen[generalID] {
			continue
		}
		seen[generalID] = true
		general, ok := findOwnedGeneral(state.Generals, generalID)
		if !ok {
			return nil, ErrGeneralNotFound
		}
		if !generalAvailableForReinforcement(state.GeneralAssignments, generalID) {
			return nil, ErrGeneralBusy
		}
		assignmentID := reinforcementAssignmentID(reinforcementID, generalID)
		state.GeneralAssignments = append(state.GeneralAssignments, GeneralAssignment{
			ID:         assignmentID,
			GeneralID:  generalID,
			Slot:       ReinforcementModuleID,
			ModuleID:   ReinforcementModuleID,
			Status:     ReinforcementStatusMarching,
			AssignedAt: now.UTC().Format(resourceDateLayout),
		})
		result = append(result, ReinforcementGeneralSnapshot{
			ID:         general.ID,
			Name:       general.Name,
			Level:      general.Level,
			Stats:      cloneStringIntMap(general.Stats),
			Attributes: cloneFloatMap(general.Attributes),
			Buffs:      cloneFloatMap(general.Buffs),
			Traits:     append([]GeneralTraitInstance(nil), general.Traits...),
			Assignment: assignmentID,
		})
		if len(result) > 1 {
			return nil, ErrInvalidGeneral
		}
	}
	return result, nil
}

func generalAvailableForReinforcement(assignments []GeneralAssignment, generalID string) bool {
	for _, assignment := range assignments {
		if strings.TrimSpace(assignment.GeneralID) != generalID {
			continue
		}
		if assignment.ID == GeneralAssignmentMain || assignment.Slot == GeneralAssignmentMain {
			continue
		}
		return false
	}
	return true
}

func reinforcementAssignmentID(reinforcementID string, generalID string) string {
	return reinforcementID + "_" + strings.TrimSpace(generalID)
}

func reinforcementBuffSnapshot(generals []ReinforcementGeneralSnapshot) []ModifierBreakdownItem {
	items := []ModifierBreakdownItem{}
	for _, general := range generals {
		for key, value := range general.Buffs {
			if value == 0 {
				continue
			}
			items = append(items, ModifierBreakdownItem{
				Source: "援军武将:" + general.Name,
				Key:    key,
				Value:  value,
				Mode:   "percentAdd",
			})
		}
	}
	return items
}

func markReinforcementArrived(record *Reinforcement, now time.Time) error {
	if record.Status != ReinforcementStatusMarching {
		return nil
	}
	if !reinforcementDue(record.SentAt, record.MarchSeconds, now) {
		return nil
	}
	nowText := now.UTC().Format(resourceDateLayout)
	record.Status = ReinforcementStatusStationed
	record.ArrivedAt = nowText
	record.UpdatedAt = nowText
	record.MailState = markStateFlag(record.MailState, "arrived")
	return nil
}

func startReinforcementReturn(record *Reinforcement, status string, reason string, now time.Time) error {
	if record.Status == ReinforcementStatusFighting && reason != "annihilated" {
		return ErrReinforcementBusy
	}
	if record.Status == ReinforcementStatusReturning || record.Status == ReinforcementStatusCompleted {
		return nil
	}
	if record.Status != ReinforcementStatusMarching && record.Status != ReinforcementStatusStationed && record.Status != ReinforcementStatusFighting {
		return ErrInvalidReinforcement
	}
	nowText := now.UTC().Format(resourceDateLayout)
	returnSeconds := calculateReinforcementReturnSeconds(record, now)
	record.Status = status
	record.ReturnSeconds = returnSeconds
	record.ReturnStartedAt = nowText
	record.ExpectedReturnedAt = now.Add(time.Duration(returnSeconds) * time.Second).UTC().Format(resourceDateLayout)
	if reason == "recalled" {
		record.RecalledAt = nowText
	}
	if reason == "expelled" {
		record.ExpelledAt = nowText
	}
	record.MailState = markStateFlag(record.MailState, reason)
	record.UpdatedAt = nowText
	return nil
}

func calculateReinforcementReturnSeconds(record *Reinforcement, now time.Time) int {
	if record.Status == ReinforcementStatusMarching {
		sentAt, err := time.Parse(resourceDateLayout, strings.TrimSpace(record.SentAt))
		if err == nil {
			elapsed := int(math.Ceil(now.Sub(sentAt).Seconds()))
			if elapsed < 1 {
				elapsed = 1
			}
			marchSeconds := record.MarchSeconds
			if marchSeconds < 1 {
				marchSeconds = record.ReturnSeconds
			}
			if marchSeconds > 0 && elapsed > marchSeconds {
				elapsed = marchSeconds
			}
			return elapsed
		}
	}
	if record.Status == ReinforcementStatusStationed {
		sentAt, sentErr := time.Parse(resourceDateLayout, strings.TrimSpace(record.SentAt))
		arrivedAt, arrivedErr := time.Parse(resourceDateLayout, strings.TrimSpace(record.ArrivedAt))
		if sentErr == nil && arrivedErr == nil && arrivedAt.After(sentAt) {
			seconds := int(math.Ceil(arrivedAt.Sub(sentAt).Seconds()))
			if seconds > 0 {
				return seconds
			}
		}
	}
	if record.ReturnSeconds > 0 {
		return record.ReturnSeconds
	}
	if record.MarchSeconds > 0 {
		return record.MarchSeconds
	}
	return 1
}

func completeReinforcementReturn(from *GameState, record *Reinforcement, now time.Time) error {
	if record.Status == ReinforcementStatusCompleted {
		return nil
	}
	if record.Status != ReinforcementStatusReturning {
		return ErrInvalidReinforcement
	}
	if !reinforcementDue(record.ReturnStartedAt, record.ReturnSeconds, now) {
		return nil
	}
	for unitType, amount := range record.RemainingTroops {
		if amount > 0 {
			addToArmy(&from.Army, unitType, amount)
		}
	}
	releaseReinforcementGenerals(from, record)
	nowText := now.UTC().Format(resourceDateLayout)
	record.Status = ReinforcementStatusCompleted
	record.ReturnedAt = nowText
	record.UpdatedAt = nowText
	record.MailState = markStateFlag(record.MailState, "returned")
	from.ServerTime = nowText
	return nil
}

func releaseReinforcementGenerals(state *GameState, record *Reinforcement) {
	if len(record.Generals) == 0 || len(state.GeneralAssignments) == 0 {
		return
	}
	assignmentIDs := map[string]bool{}
	for _, general := range record.Generals {
		if strings.TrimSpace(general.Assignment) != "" {
			assignmentIDs[general.Assignment] = true
		}
	}
	next := state.GeneralAssignments[:0]
	for _, assignment := range state.GeneralAssignments {
		if assignmentIDs[assignment.ID] {
			continue
		}
		next = append(next, assignment)
	}
	state.GeneralAssignments = next
}

func reinforcementDue(start string, seconds int, now time.Time) bool {
	if seconds < 0 {
		seconds = 0
	}
	parsed, err := time.Parse(resourceDateLayout, strings.TrimSpace(start))
	if err != nil {
		return false
	}
	return !now.Before(parsed.Add(time.Duration(seconds) * time.Second))
}

func markStateFlag(state map[string]any, key string) map[string]any {
	if state == nil {
		state = map[string]any{}
	}
	state[key] = true
	return state
}

func totalTroops(troops map[string]int) int {
	total := 0
	for _, amount := range troops {
		if amount > 0 {
			total += amount
		}
	}
	return total
}

func cleanIntMap(values map[string]int) {
	for key, value := range values {
		if value <= 0 {
			delete(values, key)
		}
	}
}

func cloneStringIntMap(src map[string]int) map[string]int {
	if src == nil {
		return nil
	}
	dst := make(map[string]int, len(src))
	for key, value := range src {
		dst[key] = value
	}
	return dst
}

func cloneFloatMap(src map[string]float64) map[string]float64 {
	if src == nil {
		return nil
	}
	dst := make(map[string]float64, len(src))
	for key, value := range src {
		dst[key] = value
	}
	return dst
}

func mergeGarrisonMetadata(current map[string]any, incoming map[string]any) map[string]any {
	next := cloneAnyMap(current)
	for key, value := range incoming {
		if strings.TrimSpace(key) == "" {
			continue
		}
		next[key] = value
	}
	return next
}

func cloneReinforcementGenerals(src []ReinforcementGeneralSnapshot) []ReinforcementGeneralSnapshot {
	dst := make([]ReinforcementGeneralSnapshot, len(src))
	for i, item := range src {
		dst[i] = item
		dst[i].Stats = cloneStringIntMap(item.Stats)
		dst[i].Attributes = cloneFloatMap(item.Attributes)
		dst[i].Buffs = cloneFloatMap(item.Buffs)
		dst[i].Traits = append([]GeneralTraitInstance(nil), item.Traits...)
	}
	return dst
}

func cloneReinforcement(record Reinforcement) Reinforcement {
	content, err := json.Marshal(record)
	if err != nil {
		return record
	}
	var cloned Reinforcement
	if err := json.Unmarshal(content, &cloned); err != nil {
		return record
	}
	return cloned
}

func cloneReinforcements(records []Reinforcement) []Reinforcement {
	result := make([]Reinforcement, len(records))
	for i, record := range records {
		result[i] = cloneReinforcement(record)
	}
	return result
}

func sortReinforcements(records []Reinforcement) {
	sort.Slice(records, func(i, j int) bool {
		return records[i].CreatedAt > records[j].CreatedAt
	})
}
