// 本文件实现世界地图玩家城池视图、坐标分配和行军时间计算。
package game

import (
	"crypto/rand"
	"errors"
	"math"
	"math/big"
	"sort"
	"strconv"
	"strings"
	"time"
)

var generateWorldMapCreateCoordinate = randomWorldCoordinate

const (
	legacySignedWorldCoordinateRadius = 100
	legacySignedWorldCoordinateSize   = legacySignedWorldCoordinateRadius*2 + 1
)

// GetWorldMapView 返回玩家世界地图视野。
func (s *Service) GetWorldMapView(playerID string, centerX int, centerY int, radius int) (WorldMapViewResponse, error) {
	playerID = strings.TrimSpace(playerID)
	if playerID == "" {
		return WorldMapViewResponse{}, ErrPlayerNotFound
	}
	now := time.Now().UTC()
	self, err := s.ensureWorldPosition(playerID, "lazy_create", nil)
	if err != nil {
		return WorldMapViewResponse{}, err
	}
	if radius <= 0 {
		radius = defaultWorldRadius
	}
	if radius > maxWorldViewRadius {
		radius = maxWorldViewRadius
	}
	if centerX < 0 || centerX >= defaultWorldWidth {
		centerX = self.X
	}
	if centerY < 0 || centerY >= defaultWorldHeight {
		centerY = self.Y
	}
	minX, maxX, minY, maxY := worldMapViewBounds(centerX, centerY, radius)
	cities, err := s.repo.ListWorldMapPlayerCities(defaultWorldID, minX, maxX, minY, maxY)
	if err != nil {
		return WorldMapViewResponse{}, err
	}
	targets, err := s.buildWorldMapTargets(playerID, self, cities, now)
	if err != nil {
		return WorldMapViewResponse{}, err
	}
	return WorldMapViewResponse{
		WorldID:    defaultWorldID,
		Width:      defaultWorldWidth,
		Height:     defaultWorldHeight,
		Self:       self,
		CenterX:    centerX,
		CenterY:    centerY,
		Radius:     radius,
		Targets:    targets,
		ServerTime: now.Format(resourceDateLayout),
	}, nil
}

// GetWorldMapPlayerCityTarget 返回单个玩家城池目标详情。
func (s *Service) GetWorldMapPlayerCityTarget(viewerID string, targetPlayerID string) (WorldMapTarget, error) {
	viewerID = strings.TrimSpace(viewerID)
	targetPlayerID = strings.TrimSpace(targetPlayerID)
	if viewerID == "" || targetPlayerID == "" {
		return WorldMapTarget{}, ErrPlayerNotFound
	}
	now := time.Now().UTC()
	self, err := s.ensureWorldPosition(viewerID, "lazy_create", nil)
	if err != nil {
		return WorldMapTarget{}, err
	}
	target, err := s.ensureWorldPosition(targetPlayerID, "lazy_create", nil)
	if err != nil {
		return WorldMapTarget{}, err
	}
	cities, err := s.repo.ListWorldMapPlayerCities(target.WorldID, target.X, target.X, target.Y, target.Y)
	if err != nil {
		return WorldMapTarget{}, err
	}
	targets, err := s.buildWorldMapTargets(viewerID, self, cities, now)
	if err != nil {
		return WorldMapTarget{}, err
	}
	if len(targets) == 0 {
		return WorldMapTarget{}, ErrPlayerNotFound
	}
	return targets[0], nil
}

// AdminGetWorldPosition 返回 GM 查询的玩家权威世界坐标。
func (s *Service) AdminGetWorldPosition(playerID string) (WorldPosition, error) {
	playerID = strings.TrimSpace(playerID)
	if playerID == "" {
		return WorldPosition{}, ErrPlayerNotFound
	}
	return s.ensureWorldPosition(playerID, "admin_query", nil)
}

// AdminAssignWorldPosition 允许 GM 手动调整玩家权威世界坐标。
func (s *Service) AdminAssignWorldPosition(playerID string, x int, y int) (WorldPosition, error) {
	playerID = strings.TrimSpace(playerID)
	if playerID == "" {
		return WorldPosition{}, ErrPlayerNotFound
	}
	if !worldCoordinateInBounds(x, y) {
		return WorldPosition{}, ErrInvalidWorldCoordinate
	}
	return s.repo.AssignWorldPosition(playerID, defaultWorldID, x, y, "admin")
}

// AdminCheckWorldCoordinate 返回 GM 调整坐标前的占格检查结果。
func (s *Service) AdminCheckWorldCoordinate(x int, y int) (WorldMapCoordinateCheck, error) {
	if !worldCoordinateInBounds(x, y) {
		return WorldMapCoordinateCheck{}, ErrInvalidWorldCoordinate
	}
	positions, err := s.repo.ListWorldPositions(defaultWorldID, x, x, y, y)
	if err != nil {
		return WorldMapCoordinateCheck{}, err
	}
	result := WorldMapCoordinateCheck{WorldID: defaultWorldID, X: x, Y: y}
	if len(positions) > 0 {
		result.Occupied = true
		result.PlayerID = positions[0].PlayerID
	}
	return result, nil
}

// AdminWorldMapOccupancy 返回 GM 查看用的世界地图占用统计。
func (s *Service) AdminWorldMapOccupancy() (WorldMapOccupancyStats, error) {
	occupied, err := s.repo.CountWorldPositions(defaultWorldID)
	if err != nil {
		return WorldMapOccupancyStats{}, err
	}
	total := defaultWorldWidth * defaultWorldHeight
	available := total - occupied
	if available < 0 {
		available = 0
	}
	return WorldMapOccupancyStats{
		WorldID:        defaultWorldID,
		Width:          defaultWorldWidth,
		Height:         defaultWorldHeight,
		TotalCells:     total,
		OccupiedCells:  occupied,
		AvailableCells: available,
		OccupancyRate:  float64(occupied) / float64(total),
	}, nil
}

// MigrateWorldPositions 为所有已有玩家补齐世界地图权威坐标。
func (s *Service) MigrateWorldPositions() (WorldMapMigrationResult, error) {
	accounts, err := s.repo.ListAccounts()
	if err != nil {
		return WorldMapMigrationResult{}, err
	}
	result := WorldMapMigrationResult{}
	var firstFailureErr error
	for _, account := range accounts {
		for _, player := range account.Players {
			result.Total++
			if _, err := s.repo.GetWorldPosition(player.ID); err == nil {
				result.Skipped++
				continue
			} else if !errors.Is(err, ErrPlayerNotFound) {
				if firstFailureErr == nil {
					firstFailureErr = err
				}
				result.Failed++
				result.Failures = append(result.Failures, player.ID+": "+err.Error())
				continue
			}
			expected := LegacyWorldCoordinateForPlayer(player.ID)
			position, err := s.ensureWorldPosition(player.ID, "migration", &expected)
			if err != nil {
				if firstFailureErr == nil {
					firstFailureErr = err
				}
				result.Failed++
				result.Failures = append(result.Failures, player.ID+": "+err.Error())
				continue
			}
			if position.X != expected.X || position.Y != expected.Y {
				result.Conflicts++
				result.ConflictDetails = append(result.ConflictDetails, player.ID+": ("+strconv.Itoa(expected.X)+","+strconv.Itoa(expected.Y)+") -> ("+strconv.Itoa(position.X)+","+strconv.Itoa(position.Y)+")")
			}
			result.Created++
		}
	}
	if result.Failed > 0 {
		return result, firstFailureErr
	}
	return result, nil
}

// ensureWorldPosition 确保玩家有权威坐标。
func (s *Service) ensureWorldPosition(playerID string, assignedBy string, preferred *WorldCoordinate) (WorldPosition, error) {
	position, err := s.repo.EnsureWorldPosition(playerID, assignedBy, preferred)
	if err != nil {
		return WorldPosition{}, err
	}
	return position, nil
}

// buildWorldMapTargets 把当前范围内的轻量玩家城池投影转换为地图目标。
func (s *Service) buildWorldMapTargets(viewerID string, self WorldPosition, cities []WorldMapPlayerCity, now time.Time) ([]WorldMapTarget, error) {
	requestAccountID, _ := s.repo.GetAccountIDByPlayerID(viewerID)
	requestPvpState, err := s.repo.GetPvpPlayerState(viewerID, now)
	if err != nil {
		return nil, err
	}
	attackLimitReached := requestPvpState.DailyAttackLimit > 0 && requestPvpState.DailyAttackCount >= requestPvpState.DailyAttackLimit
	yellowCities := BuildYellowTurbanCities(GetYellowTurbanConfig())
	targets := make([]WorldMapTarget, 0, len(cities)+len(yellowCities))
	for _, city := range cities {
		position := city.Position
		relation := WorldRelationOther
		sameAccount := requestAccountID != "" && city.AccountID == requestAccountID
		status := WorldTargetStatusNormal
		canScout := true
		canAttack := !sameAccount
		canPlunder := canAttack
		canReinforce := true
		scoutReason := ""
		attackReason := ""
		plunderReason := ""
		reinforceReason := ""
		if position.PlayerID == viewerID {
			relation = WorldRelationSelf
			status = WorldTargetStatusSelf
			canScout = false
			canAttack = false
			canPlunder = false
			canReinforce = false
			scoutReason = "自己的城池"
			attackReason = "自己的城池"
			plunderReason = "自己的城池"
			reinforceReason = "自己的城池"
		} else if sameAccount {
			canScout = false
			canAttack = false
			canPlunder = false
			scoutReason = "同账号存档不能侦查"
			attackReason = "同账号存档不能攻击"
			plunderReason = "同账号存档不能掠夺"
		}
		if position.PlayerID != viewerID {
			available, err := s.canReinforceWorldMapTarget(viewerID, position.PlayerID)
			if err != nil {
				return nil, err
			}
			if !available {
				canReinforce = false
				reinforceReason = "目标增援来源已满"
			}
		}
		targetPvpState, _ := s.repo.GetPvpPlayerState(position.PlayerID, now)
		protected, protectionType, _ := activePvpProtection(targetPvpState, now)
		if protected && position.PlayerID != viewerID {
			status = worldTargetStatusForProtection(protectionType)
			canAttack = false
			canPlunder = false
			if attackReason == "" {
				attackReason = pvpProtectionReason(protectionType)
			}
			if plunderReason == "" {
				plunderReason = pvpProtectionReason(protectionType)
			}
		} else if canAttack {
			status = WorldTargetStatusAttackable
		}
		if attackLimitReached && position.PlayerID != viewerID && city.AccountID != requestAccountID {
			canAttack = false
			canPlunder = false
			attackReason = "今日攻击次数已用完"
			plunderReason = "今日攻击次数已用完"
			status = WorldTargetStatusUnavailable
		}
		reason := firstNonEmpty(scoutReason, attackReason, plunderReason, reinforceReason)
		targets = append(targets, WorldMapTarget{
			TargetType:      WorldMapTargetPlayerCity,
			TargetID:        position.PlayerID,
			PlayerID:        position.PlayerID,
			Name:            city.Name,
			Faction:         city.Faction,
			Relation:        relation,
			SameAccount:     sameAccount,
			Level:           city.BuildingLevel,
			Status:          status,
			X:               position.X,
			Y:               position.Y,
			Distance:        worldMapDistance(WorldCoordinate{X: self.X, Y: self.Y}, WorldCoordinate{X: position.X, Y: position.Y}),
			Direction:       worldMapDirection(WorldCoordinate{X: self.X, Y: self.Y}, WorldCoordinate{X: position.X, Y: position.Y}),
			CanScout:        canScout,
			CanAttack:       canAttack,
			CanPlunder:      canPlunder,
			CanReinforce:    canReinforce,
			Reason:          reason,
			ScoutReason:     scoutReason,
			AttackReason:    attackReason,
			PlunderReason:   plunderReason,
			ReinforceReason: reinforceReason,
		})
	}
	for _, city := range yellowCities {
		targets = append(targets, WorldMapTarget{
			TargetType:      WorldMapTargetYellowTurban,
			TargetID:        city.ID,
			Name:            city.Name,
			Faction:         city.Faction,
			Relation:        WorldRelationOther,
			Level:           1,
			Status:          WorldTargetStatusUnavailable,
			X:               city.X,
			Y:               city.Y,
			Distance:        worldMapDistance(WorldCoordinate{X: self.X, Y: self.Y}, WorldCoordinate{X: city.X, Y: city.Y}),
			Direction:       worldMapDirection(WorldCoordinate{X: self.X, Y: self.Y}, WorldCoordinate{X: city.X, Y: city.Y}),
			CanScout:        false,
			CanAttack:       false,
			CanPlunder:      false,
			CanReinforce:    false,
			Reason:          "黄巾势力暂不可主动围剿",
			ScoutReason:     "黄巾势力暂不可侦查",
			AttackReason:    "第一版暂不开放围剿黄巾城池",
			PlunderReason:   "黄巾势力不可掠夺",
			ReinforceReason: "黄巾势力不可增援",
		})
	}
	sort.Slice(targets, func(i, j int) bool {
		if targets[i].Distance == targets[j].Distance {
			return targets[i].TargetID < targets[j].TargetID
		}
		return targets[i].Distance < targets[j].Distance
	})
	return targets, nil
}

// canReinforceWorldMapTarget 用真实增援槽位规则判断地图目标是否可增援。
func (s *Service) canReinforceWorldMapTarget(viewerID string, targetPlayerID string) (bool, error) {
	records, err := s.repo.ListReceivedReinforcements(targetPlayerID)
	if err != nil {
		return false, err
	}
	if err := ensureReinforcementSourceSlot(viewerID, records); err != nil {
		if errors.Is(err, ErrReinforcementSlotFull) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// CalculateWorldMarchSeconds 按地图距离、最低速度和行军加成计算行军时间。
func CalculateWorldMarchSeconds(distance int, speed int, now time.Time, sources []ModifierSource) int {
	if distance <= 0 {
		return 0
	}
	if speed <= 0 {
		speed = 1
	}
	baseSeconds := int(math.Ceil(float64(distance*reinforcementSecondsPerGrid) / float64(speed)))
	seconds := applySpeedBonus(baseSeconds, StatMarchSpeedBonus, now, sources)
	if seconds < 1 {
		return 1
	}
	if seconds > maxWorldMarchSeconds {
		return maxWorldMarchSeconds
	}
	return seconds
}

// worldMapDistance 返回曼哈顿地图距离。
func worldMapDistance(a WorldCoordinate, b WorldCoordinate) int {
	return int(math.Abs(float64(a.X-b.X))) + int(math.Abs(float64(a.Y-b.Y)))
}

// worldMapViewBounds 计算贴近边界时仍补满合法坐标的视野范围。
func worldMapViewBounds(centerX int, centerY int, radius int) (int, int, int, int) {
	safeRadius := radius
	if safeRadius < 0 {
		safeRadius = 0
	}
	width := minInt(defaultWorldWidth, safeRadius*2+1)
	height := minInt(defaultWorldHeight, safeRadius*2+1)
	centerX = clampInt(centerX, 0, defaultWorldWidth-1)
	centerY = clampInt(centerY, 0, defaultWorldHeight-1)
	minX := clampInt(centerX-safeRadius, 0, defaultWorldWidth-width)
	minY := clampInt(centerY-safeRadius, 0, defaultWorldHeight-height)
	return minX, minX + width - 1, minY, minY + height - 1
}

// worldMapDirection 返回目标相对自己的八方向文本。
func worldMapDirection(a WorldCoordinate, b WorldCoordinate) string {
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

// worldTargetStatusForProtection 将 PVP 保护类型映射为世界地图目标状态。
func worldTargetStatusForProtection(protectionType string) string {
	if normalizePvpProtectionType(protectionType) == PvpProtectionTypeManual {
		return WorldTargetStatusTruce
	}
	return WorldTargetStatusProtected
}

// worldMapPreferredCoordinate 生成坐标分配的优先起点。
func worldMapPreferredCoordinate(playerID string, preferred *WorldCoordinate) WorldCoordinate {
	if preferred != nil && worldCoordinateInBounds(preferred.X, preferred.Y) {
		return *preferred
	}
	return LegacyWorldCoordinateForPlayer(playerID)
}

// LegacyWorldCoordinateForPlayer 将旧 PVP 伪随机坐标映射为第一版世界地图坐标。
func LegacyWorldCoordinateForPlayer(playerID string) WorldCoordinate {
	legacy := pvpWorldPositionForPlayer(playerID)
	return mapLegacyWorldCoordinateToCurrent(WorldCoordinate{X: legacy.X, Y: legacy.Y})
}

// mapLegacyWorldCoordinateToCurrent 将历史地图坐标转换为第一版 0-99 坐标。
func mapLegacyWorldCoordinateToCurrent(coordinate WorldCoordinate) WorldCoordinate {
	if worldCoordinateInBounds(coordinate.X, coordinate.Y) {
		return coordinate
	}
	return WorldCoordinate{
		X: mapLegacyWorldAxisToCurrent(coordinate.X, defaultWorldWidth),
		Y: mapLegacyWorldAxisToCurrent(coordinate.Y, defaultWorldHeight),
	}
}

// mapLegacyWorldAxisToCurrent 将单轴历史坐标映射到当前地图范围。
func mapLegacyWorldAxisToCurrent(value int, size int) int {
	if size <= 0 {
		return 0
	}
	if value >= 0 && value < size {
		return value
	}
	mapped := int(math.Floor(float64(value+legacySignedWorldCoordinateRadius) * float64(size) / float64(legacySignedWorldCoordinateSize)))
	return clampInt(mapped, 0, size-1)
}

// randomWorldCoordinate 为新建玩家生成随机世界坐标起点。
func randomWorldCoordinate() (WorldCoordinate, error) {
	total := defaultWorldWidth * defaultWorldHeight
	value, err := rand.Int(rand.Reader, big.NewInt(int64(total)))
	if err != nil {
		return WorldCoordinate{}, err
	}
	index := int(value.Int64())
	return WorldCoordinate{
		X: index % defaultWorldWidth,
		Y: index / defaultWorldWidth,
	}, nil
}

// findAvailableWorldCoordinateLocked 从优先坐标向外稳定查找空格。
func findAvailableWorldCoordinateLocked(positions map[string]WorldPosition, worldID string, startX int, startY int) (int, int, bool) {
	occupied := map[[2]int]bool{}
	for _, position := range positions {
		if position.WorldID == worldID {
			occupied[[2]int{position.X, position.Y}] = true
		}
	}
	for _, coordinate := range WorldMapCoordinateCandidates(startX, startY) {
		if !occupied[[2]int{coordinate.X, coordinate.Y}] {
			return coordinate.X, coordinate.Y, true
		}
	}
	return 0, 0, false
}

// WorldMapCoordinateCandidates 按稳定顺序生成从起点向外扩散的合法坐标。
func WorldMapCoordinateCandidates(startX int, startY int) []WorldCoordinate {
	candidates := []WorldCoordinate{}
	for radius := 0; radius < defaultWorldWidth+defaultWorldHeight; radius++ {
		for _, coordinate := range worldMapCoordinateRing(startX, startY, radius) {
			if worldCoordinateInBounds(coordinate.X, coordinate.Y) {
				candidates = append(candidates, coordinate)
			}
		}
	}
	return candidates
}

// worldMapCoordinateRing 按上、右、下、左和外圈顺序生成同半径候选坐标。
func worldMapCoordinateRing(startX int, startY int, radius int) []WorldCoordinate {
	if radius <= 0 {
		return []WorldCoordinate{{X: startX, Y: startY}}
	}
	candidates := []WorldCoordinate{
		{X: startX, Y: startY - radius},
		{X: startX + radius, Y: startY},
		{X: startX, Y: startY + radius},
		{X: startX - radius, Y: startY},
	}
	seen := map[[2]int]bool{}
	for _, coordinate := range candidates {
		seen[[2]int{coordinate.X, coordinate.Y}] = true
	}
	for y := startY - radius; y <= startY+radius; y++ {
		for x := startX - radius; x <= startX+radius; x++ {
			if worldMapDistance(WorldCoordinate{X: startX, Y: startY}, WorldCoordinate{X: x, Y: y}) != radius {
				continue
			}
			key := [2]int{x, y}
			if seen[key] {
				continue
			}
			seen[key] = true
			candidates = append(candidates, WorldCoordinate{X: x, Y: y})
		}
	}
	return candidates
}

// worldCoordinateInBounds 判断坐标是否在第一版世界地图内。
func worldCoordinateInBounds(x int, y int) bool {
	return x >= 0 && x < defaultWorldWidth && y >= 0 && y < defaultWorldHeight
}
