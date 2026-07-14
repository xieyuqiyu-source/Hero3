// 本文件测试世界地图玩家城池坐标和视图规则。
package game

import (
	"encoding/json"
	"errors"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestCreatePlayerAssignsWorldPosition(t *testing.T) {
	setTestFactionsAndGenerals(t, FactionsConfig{
		"wei": {Generals: []GeneralInfo{{ID: "caocao", Name: "曹操"}}},
	}, GeneralsConfig{
		Enabled: true,
		Heroes: map[string]GeneralHeroConfig{
			"caocao": {ID: "caocao", Name: "曹操", Faction: "wei", Enabled: true},
		},
	})
	now := time.Now()
	repo := NewMemoryRepository()
	svc := NewServiceWithRepository(repo)
	if err := repo.CreateAccount(Account{ID: "account_world_create", Username: "world_create", CreatedAt: now}); err != nil {
		t.Fatalf("CreateAccount failed: %v", err)
	}
	playerID, _, err := svc.CreatePlayer("account_world_create", "新城", "wei", "caocao")
	if err != nil {
		t.Fatalf("CreatePlayer failed: %v", err)
	}
	position, err := repo.GetWorldPosition(playerID)
	if err != nil {
		t.Fatalf("expected world position for new player: %v", err)
	}
	if position.PlayerID != playerID || position.WorldID != defaultWorldID || position.AssignedBy != "create" {
		t.Fatalf("unexpected new player position: %+v", position)
	}
	if !worldCoordinateInBounds(position.X, position.Y) {
		t.Fatalf("new player coordinate out of bounds: %+v", position)
	}
}

func TestCreatePlayerUsesRandomWorldCoordinateStart(t *testing.T) {
	setTestFactionsAndGenerals(t, FactionsConfig{
		"wei": {Generals: []GeneralInfo{{ID: "caocao", Name: "曹操"}}},
	}, GeneralsConfig{
		Enabled: true,
		Heroes: map[string]GeneralHeroConfig{
			"caocao": {ID: "caocao", Name: "曹操", Faction: "wei", Enabled: true},
		},
	})
	previous := generateWorldMapCreateCoordinate
	generateWorldMapCreateCoordinate = func() (WorldCoordinate, error) {
		return WorldCoordinate{X: 44, Y: 55}, nil
	}
	defer func() {
		generateWorldMapCreateCoordinate = previous
	}()
	now := time.Now()
	repo := NewMemoryRepository()
	svc := NewServiceWithRepository(repo)
	if err := repo.CreateAccount(Account{ID: "account_world_create_random", Username: "world_create_random", CreatedAt: now}); err != nil {
		t.Fatalf("CreateAccount failed: %v", err)
	}
	playerID, _, err := svc.CreatePlayer("account_world_create_random", "随机新城", "wei", "caocao")
	if err != nil {
		t.Fatalf("CreatePlayer failed: %v", err)
	}
	position, err := repo.GetWorldPosition(playerID)
	if err != nil {
		t.Fatalf("expected world position for new player: %v", err)
	}
	if position.X != 44 || position.Y != 55 || position.AssignedBy != "create" {
		t.Fatalf("expected new player to use random coordinate start, got %+v", position)
	}
}

func TestWorldMapViewEnsuresAuthoritativePositionsAndHidesArmy(t *testing.T) {
	svc, repo, attacker, defender := newPvpTestService(t)
	if _, err := repo.AssignWorldPosition(attacker.Player.ID, defaultWorldID, 10, 10, "test"); err != nil {
		t.Fatalf("AssignWorldPosition attacker failed: %v", err)
	}
	if _, err := repo.AssignWorldPosition(defender.Player.ID, defaultWorldID, 13, 14, "test"); err != nil {
		t.Fatalf("AssignWorldPosition defender failed: %v", err)
	}
	view, err := svc.GetWorldMapView(attacker.Player.ID, -1, -1, 10)
	if err != nil {
		t.Fatalf("GetWorldMapView failed: %v", err)
	}
	if view.CenterX != 10 || view.CenterY != 10 || view.Width != 100 || view.Height != 100 {
		t.Fatalf("expected view centered on self in 100x100 world, got %+v", view)
	}
	if view.Self.X != 10 || view.Self.Y != 10 {
		t.Fatalf("expected authoritative self position, got %+v", view.Self)
	}
	var selfTarget WorldMapTarget
	var target WorldMapTarget
	for _, item := range view.Targets {
		if item.PlayerID == attacker.Player.ID {
			selfTarget = item
		}
		if item.PlayerID == defender.Player.ID {
			target = item
		}
	}
	if selfTarget.PlayerID == "" {
		t.Fatalf("expected self target in view, got %+v", view.Targets)
	}
	if selfTarget.Relation != WorldRelationSelf || selfTarget.Status != WorldTargetStatusSelf || selfTarget.Distance != 0 || selfTarget.Direction != "原地" {
		t.Fatalf("expected self target relation/status/distance, got %+v", selfTarget)
	}
	if selfTarget.CanScout || selfTarget.CanAttack || selfTarget.CanPlunder || selfTarget.CanReinforce {
		t.Fatalf("expected self target actions disabled, got %+v", selfTarget)
	}
	if selfTarget.ScoutReason != "自己的城池" || selfTarget.AttackReason != "自己的城池" || selfTarget.PlunderReason != "自己的城池" || selfTarget.ReinforceReason != "自己的城池" {
		t.Fatalf("expected self target reasons, got %+v", selfTarget)
	}
	if target.PlayerID == "" {
		t.Fatalf("expected defender target in view, got %+v", view.Targets)
	}
	if target.Distance != 7 || target.Direction != "东南" {
		t.Fatalf("expected manhattan distance 7 and southeast direction, got %+v", target)
	}
	if target.Level <= 0 || target.CanAttack != true || target.CanPlunder != true || target.CanScout != true {
		t.Fatalf("expected basic city info and operations, got %+v", target)
	}
}

func TestWorldMapViewFiltersTargetsByRequestedRadius(t *testing.T) {
	svc, repo, attacker, defender := newPvpTestService(t)
	if _, err := repo.AssignWorldPosition(attacker.Player.ID, defaultWorldID, 10, 10, "test"); err != nil {
		t.Fatalf("AssignWorldPosition attacker failed: %v", err)
	}
	if _, err := repo.AssignWorldPosition(defender.Player.ID, defaultWorldID, 14, 10, "test"); err != nil {
		t.Fatalf("AssignWorldPosition defender failed: %v", err)
	}
	nearView, err := svc.GetWorldMapView(attacker.Player.ID, 10, 10, 3)
	if err != nil {
		t.Fatalf("GetWorldMapView near failed: %v", err)
	}
	for _, target := range nearView.Targets {
		if target.PlayerID == defender.Player.ID {
			t.Fatalf("expected defender outside radius 3, got %+v", nearView.Targets)
		}
	}
	targetView, err := svc.GetWorldMapView(attacker.Player.ID, 14, 10, 0)
	if err != nil {
		t.Fatalf("GetWorldMapView target center failed: %v", err)
	}
	if targetView.CenterX != 14 || targetView.CenterY != 10 || targetView.Radius != defaultWorldRadius {
		t.Fatalf("expected requested center with default radius, got %+v", targetView)
	}
	foundTarget := false
	for _, target := range targetView.Targets {
		if target.PlayerID == defender.Player.ID {
			foundTarget = true
			if target.Distance != 4 {
				t.Fatalf("expected distance from self to stay 4, got %+v", target)
			}
		}
	}
	if !foundTarget {
		t.Fatalf("expected defender inside target-centered view, got %+v", targetView.Targets)
	}
}

func TestWorldMapViewFillsEdgeBounds(t *testing.T) {
	svc, repo, attacker, defender := newPvpTestService(t)
	if _, err := repo.AssignWorldPosition(attacker.Player.ID, defaultWorldID, 0, 0, "test"); err != nil {
		t.Fatalf("AssignWorldPosition attacker failed: %v", err)
	}
	if _, err := repo.AssignWorldPosition(defender.Player.ID, defaultWorldID, 20, 20, "test"); err != nil {
		t.Fatalf("AssignWorldPosition defender failed: %v", err)
	}
	minX, maxX, minY, maxY := worldMapViewBounds(0, 0, 10)
	if minX != 0 || maxX != 20 || minY != 0 || maxY != 20 {
		t.Fatalf("expected edge bounds to fill 0-20, got x=%d-%d y=%d-%d", minX, maxX, minY, maxY)
	}
	minX, maxX, minY, maxY = worldMapViewBounds(99, 99, 10)
	if minX != 79 || maxX != 99 || minY != 79 || maxY != 99 {
		t.Fatalf("expected southeast edge bounds to fill 79-99, got x=%d-%d y=%d-%d", minX, maxX, minY, maxY)
	}
	view, err := svc.GetWorldMapView(attacker.Player.ID, 0, 0, 10)
	if err != nil {
		t.Fatalf("GetWorldMapView failed: %v", err)
	}
	foundTarget := false
	for _, target := range view.Targets {
		if target.PlayerID == defender.Player.ID {
			foundTarget = true
			if target.Distance != 40 {
				t.Fatalf("expected edge-filled target distance 40, got %+v", target)
			}
		}
	}
	if !foundTarget {
		t.Fatalf("expected edge-filled view to include coordinate 20,20 target, got %+v", view.Targets)
	}
}

func TestWorldMapViewBoundsFullRadiusCoversEntireWorld(t *testing.T) {
	for _, center := range []WorldCoordinate{
		{X: 0, Y: 0},
		{X: 50, Y: 50},
		{X: 99, Y: 99},
	} {
		minX, maxX, minY, maxY := worldMapViewBounds(center.X, center.Y, maxWorldViewRadius)
		if minX != 0 || maxX != defaultWorldWidth-1 || minY != 0 || maxY != defaultWorldHeight-1 {
			t.Fatalf("expected full radius from %+v to cover whole world, got x=%d-%d y=%d-%d", center, minX, maxX, minY, maxY)
		}
	}
}

func TestWorldMapTargetDisablesSameAccountScout(t *testing.T) {
	svc, repo, attacker, _ := newPvpTestService(t)
	now := time.Now()
	sameAccount := newPlayerState("player_world_same_account", "同账号", "wu", "sunquan", now)
	if err := repo.CreatePlayer("account_pvp_a", sameAccount, now); err != nil {
		t.Fatalf("CreatePlayer same account failed: %v", err)
	}
	if _, err := repo.AssignWorldPosition(attacker.Player.ID, defaultWorldID, 10, 10, "test"); err != nil {
		t.Fatalf("AssignWorldPosition attacker failed: %v", err)
	}
	if _, err := repo.AssignWorldPosition(sameAccount.Player.ID, defaultWorldID, 11, 10, "test"); err != nil {
		t.Fatalf("AssignWorldPosition same account failed: %v", err)
	}
	if _, err := svc.SetPvpProtection(sameAccount.Player.ID, PvpProtectionTypeManual, time.Hour, "test", now); err != nil {
		t.Fatalf("SetPvpProtection same account failed: %v", err)
	}
	view, err := svc.GetWorldMapView(attacker.Player.ID, 10, 10, 10)
	if err != nil {
		t.Fatalf("GetWorldMapView failed: %v", err)
	}
	var target WorldMapTarget
	for _, item := range view.Targets {
		if item.PlayerID == sameAccount.Player.ID {
			target = item
			break
		}
	}
	if target.PlayerID == "" {
		t.Fatalf("expected same account target in view, got %+v", view.Targets)
	}
	if !target.SameAccount {
		t.Fatalf("expected same account target marker, got %+v", target)
	}
	if target.CanScout || target.CanAttack || target.CanPlunder {
		t.Fatalf("expected same account scout/attack/plunder disabled, got %+v", target)
	}
	if !target.CanReinforce {
		t.Fatalf("expected same account reinforcement to follow reinforcement rules, got %+v", target)
	}
	if target.Status != WorldTargetStatusTruce {
		t.Fatalf("expected same account truce status to be visible, got %+v", target)
	}
	if target.ScoutReason != "同账号存档不能侦查" || target.AttackReason != "同账号存档不能攻击" || target.PlunderReason != "同账号存档不能掠夺" {
		t.Fatalf("expected same account action reasons, got %+v", target)
	}
}

func TestWorldMapTargetDisablesAttackWhenDailyLimitReached(t *testing.T) {
	svc, repo, attacker, defender := newPvpTestService(t)
	now := time.Now().UTC()
	state := newDefaultPvpPlayerState(attacker.Player.ID, now)
	state.DailyAttackCount = state.DailyAttackLimit
	if err := repo.SavePvpPlayerState(state, now); err != nil {
		t.Fatalf("SavePvpPlayerState daily limit failed: %v", err)
	}
	if _, err := repo.AssignWorldPosition(attacker.Player.ID, defaultWorldID, 10, 10, "test"); err != nil {
		t.Fatalf("AssignWorldPosition attacker failed: %v", err)
	}
	if _, err := repo.AssignWorldPosition(defender.Player.ID, defaultWorldID, 12, 10, "test"); err != nil {
		t.Fatalf("AssignWorldPosition defender failed: %v", err)
	}
	view, err := svc.GetWorldMapView(attacker.Player.ID, 10, 10, 10)
	if err != nil {
		t.Fatalf("GetWorldMapView failed: %v", err)
	}
	var target WorldMapTarget
	for _, item := range view.Targets {
		if item.PlayerID == defender.Player.ID {
			target = item
			break
		}
	}
	if target.PlayerID == "" {
		t.Fatalf("expected defender target in view, got %+v", view.Targets)
	}
	if target.CanAttack || target.CanPlunder {
		t.Fatalf("expected daily limit to disable attack and plunder, got %+v", target)
	}
	if !target.CanScout || !target.CanReinforce {
		t.Fatalf("expected daily limit not to disable scout or reinforce, got %+v", target)
	}
	if target.AttackReason != "今日攻击次数已用完" || target.PlunderReason != "今日攻击次数已用完" {
		t.Fatalf("expected daily limit reasons, got %+v", target)
	}
	if target.Status != WorldTargetStatusUnavailable {
		t.Fatalf("expected unavailable status when daily attack limit is reached, got %+v", target)
	}
}

func TestWorldMapTargetDisablesReinforceWhenTargetSourceSlotsFull(t *testing.T) {
	svc, repo, attacker, defender := newPvpTestService(t)
	now := time.Now()
	for i := 0; i < defaultReinforcementMaxSources; i++ {
		accountID := "account_world_reinforce_source_" + strconv.Itoa(i)
		if err := repo.CreateAccount(Account{ID: accountID, Username: accountID, PasswordHash: "hash", CreatedAt: now}); err != nil {
			t.Fatalf("CreateAccount source failed: %v", err)
		}
		source := newPlayerState("player_world_reinforce_source_"+strconv.Itoa(i), "援军来源", "wei", "caocao", now)
		if err := repo.CreatePlayer(accountID, source, now); err != nil {
			t.Fatalf("CreatePlayer source failed: %v", err)
		}
		repo.reinforcements["reinforcement_world_slot_"+strconv.Itoa(i)] = Reinforcement{
			ID:            "reinforcement_world_slot_" + strconv.Itoa(i),
			FromPlayerID:  source.Player.ID,
			ToPlayerID:    defender.Player.ID,
			OwnerPlayerID: source.Player.ID,
			HostPlayerID:  defender.Player.ID,
			SourceType:    GarrisonSourceReinforcement,
			Status:        ReinforcementStatusStationed,
		}
	}
	if _, err := repo.AssignWorldPosition(attacker.Player.ID, defaultWorldID, 10, 10, "test"); err != nil {
		t.Fatalf("AssignWorldPosition attacker failed: %v", err)
	}
	if _, err := repo.AssignWorldPosition(defender.Player.ID, defaultWorldID, 12, 10, "test"); err != nil {
		t.Fatalf("AssignWorldPosition defender failed: %v", err)
	}
	if _, err := svc.SetPvpProtection(defender.Player.ID, PvpProtectionTypeManual, time.Hour, "test", now); err != nil {
		t.Fatalf("SetPvpProtection defender failed: %v", err)
	}
	view, err := svc.GetWorldMapView(attacker.Player.ID, 10, 10, 10)
	if err != nil {
		t.Fatalf("GetWorldMapView failed: %v", err)
	}
	var target WorldMapTarget
	for _, item := range view.Targets {
		if item.PlayerID == defender.Player.ID {
			target = item
			break
		}
	}
	if target.PlayerID == "" {
		t.Fatalf("expected defender target in view, got %+v", view.Targets)
	}
	if target.CanReinforce {
		t.Fatalf("expected world map target to disable reinforce when source slots are full, got %+v", target)
	}
	if target.Status != WorldTargetStatusTruce {
		t.Fatalf("expected manual protection to be shown as truce, got %+v", target)
	}
	if target.AttackReason != "目标处于免战保护" || target.PlunderReason != "目标处于免战保护" {
		t.Fatalf("expected protected attack/plunder reasons, got %+v", target)
	}
	if target.ReinforceReason != "目标增援来源已满" {
		t.Fatalf("expected reinforcement full reason, got %+v", target)
	}
	if target.Reason != "目标处于免战保护" {
		t.Fatalf("expected compatible reason to keep first disabled action reason, got %+v", target)
	}
}

func TestWorldMapViewOnlyLazyCreatesViewerPosition(t *testing.T) {
	svc, repo, attacker, defender := newPvpTestService(t)
	if _, err := repo.GetWorldPosition(attacker.Player.ID); !errors.Is(err, ErrPlayerNotFound) {
		t.Fatalf("expected attacker to start without world position, got %v", err)
	}
	if _, err := repo.GetWorldPosition(defender.Player.ID); !errors.Is(err, ErrPlayerNotFound) {
		t.Fatalf("expected defender to start without world position, got %v", err)
	}
	view, err := svc.GetWorldMapView(attacker.Player.ID, -1, -1, 100)
	if err != nil {
		t.Fatalf("GetWorldMapView failed: %v", err)
	}
	selfPosition, err := repo.GetWorldPosition(attacker.Player.ID)
	if err != nil {
		t.Fatalf("expected lazy self world position: %v", err)
	}
	if selfPosition.AssignedBy != "lazy_create" {
		t.Fatalf("expected viewer to receive lazy_create assignment, got %+v", selfPosition)
	}
	if _, err := repo.GetWorldPosition(defender.Player.ID); !errors.Is(err, ErrPlayerNotFound) {
		t.Fatalf("map read must not scan and mutate unrelated players, got %v", err)
	}
	if view.Self.X != selfPosition.X || view.Self.Y != selfPosition.Y {
		t.Fatalf("expected response self to use lazy position, view=%+v saved=%+v", view.Self, selfPosition)
	}
	foundTarget := false
	for _, target := range view.Targets {
		if target.PlayerID == defender.Player.ID {
			foundTarget = true
		}
	}
	if foundTarget {
		t.Fatalf("player without migrated world position must stay out of read-only map result, got %+v", view.Targets)
	}
}

func TestWorldMapViewJSONDoesNotExposeHiddenTargetDetails(t *testing.T) {
	svc, repo, attacker, defender := newPvpTestService(t)
	if _, err := repo.AssignWorldPosition(attacker.Player.ID, defaultWorldID, 10, 10, "test"); err != nil {
		t.Fatalf("AssignWorldPosition attacker failed: %v", err)
	}
	if _, err := repo.AssignWorldPosition(defender.Player.ID, defaultWorldID, 12, 10, "test"); err != nil {
		t.Fatalf("AssignWorldPosition defender failed: %v", err)
	}
	view, err := svc.GetWorldMapView(attacker.Player.ID, -1, -1, 10)
	if err != nil {
		t.Fatalf("GetWorldMapView failed: %v", err)
	}
	payload, err := json.Marshal(view)
	if err != nil {
		t.Fatalf("marshal world map view: %v", err)
	}
	body := string(payload)
	for _, hiddenField := range []string{"totalArmy", "resources", "generals", "garrison", "troops"} {
		if strings.Contains(body, hiddenField) {
			t.Fatalf("world map view leaked hidden field %q in %s", hiddenField, body)
		}
	}
}

func TestWorldMapViewSupportsFullMapRadiusForClientCache(t *testing.T) {
	svc, repo, attacker, defender := newPvpTestService(t)
	if _, err := repo.AssignWorldPosition(attacker.Player.ID, defaultWorldID, 0, 0, "test"); err != nil {
		t.Fatalf("AssignWorldPosition attacker failed: %v", err)
	}
	if _, err := repo.AssignWorldPosition(defender.Player.ID, defaultWorldID, 99, 99, "test"); err != nil {
		t.Fatalf("AssignWorldPosition defender failed: %v", err)
	}
	view, err := svc.GetWorldMapView(attacker.Player.ID, -1, -1, 100)
	if err != nil {
		t.Fatalf("GetWorldMapView failed: %v", err)
	}
	if view.Radius != 100 {
		t.Fatalf("expected full map radius to be preserved, got %d", view.Radius)
	}
	foundFarTarget := false
	for _, target := range view.Targets {
		if target.PlayerID == defender.Player.ID {
			foundFarTarget = true
			if target.Distance != 198 {
				t.Fatalf("expected far target distance 198, got %+v", target)
			}
		}
	}
	if !foundFarTarget {
		t.Fatalf("expected full map view to include far target, got %+v", view.Targets)
	}
}

func TestWorldMapPositionsAreUniqueWithPreferredConflict(t *testing.T) {
	now := time.Now()
	repo := NewMemoryRepository()
	if err := repo.CreateAccount(Account{ID: "account_world", Username: "world", CreatedAt: now}); err != nil {
		t.Fatalf("CreateAccount failed: %v", err)
	}
	first := newPlayerState("player_world_first", "甲", "wei", "caocao", now)
	second := newPlayerState("player_world_second", "乙", "shu", "liubei", now)
	if err := repo.CreatePlayer("account_world", first, now); err != nil {
		t.Fatalf("CreatePlayer first failed: %v", err)
	}
	if err := repo.CreatePlayer("account_world", second, now); err != nil {
		t.Fatalf("CreatePlayer second failed: %v", err)
	}
	preferred := &WorldCoordinate{X: 20, Y: 20}
	firstPosition, err := repo.EnsureWorldPosition(first.Player.ID, "test", preferred)
	if err != nil {
		t.Fatalf("EnsureWorldPosition first failed: %v", err)
	}
	secondPosition, err := repo.EnsureWorldPosition(second.Player.ID, "test", preferred)
	if err != nil {
		t.Fatalf("EnsureWorldPosition second failed: %v", err)
	}
	if firstPosition.X != 20 || firstPosition.Y != 20 {
		t.Fatalf("expected first player to use preferred coordinate, got %+v", firstPosition)
	}
	if secondPosition.X == firstPosition.X && secondPosition.Y == firstPosition.Y {
		t.Fatalf("expected second player to resolve conflict, got first=%+v second=%+v", firstPosition, secondPosition)
	}
}

func TestWorldMapCoordinateCandidatesUseStableSearchOrder(t *testing.T) {
	candidates := WorldMapCoordinateCandidates(20, 20)
	expected := []WorldCoordinate{
		{X: 20, Y: 20},
		{X: 20, Y: 19},
		{X: 21, Y: 20},
		{X: 20, Y: 21},
		{X: 19, Y: 20},
		{X: 20, Y: 18},
	}
	for i, coordinate := range expected {
		if candidates[i] != coordinate {
			t.Fatalf("expected candidate %d to be %+v, got %+v", i, coordinate, candidates[i])
		}
	}
	edgeCandidates := WorldMapCoordinateCandidates(0, 0)
	expectedEdge := []WorldCoordinate{
		{X: 0, Y: 0},
		{X: 1, Y: 0},
		{X: 0, Y: 1},
	}
	for i, coordinate := range expectedEdge {
		if edgeCandidates[i] != coordinate {
			t.Fatalf("expected edge candidate %d to be %+v, got %+v", i, coordinate, edgeCandidates[i])
		}
	}
}

func TestWorldMapCoordinateCandidatesCoverWorldWithoutDuplicates(t *testing.T) {
	for _, start := range []WorldCoordinate{{X: 50, Y: 50}, {X: 0, Y: 0}, {X: 99, Y: 99}} {
		candidates := WorldMapCoordinateCandidates(start.X, start.Y)
		if len(candidates) != defaultWorldWidth*defaultWorldHeight {
			t.Fatalf("expected full world candidates from %+v, got %d", start, len(candidates))
		}
		seen := map[WorldCoordinate]bool{}
		for _, coordinate := range candidates {
			if !worldCoordinateInBounds(coordinate.X, coordinate.Y) {
				t.Fatalf("candidate out of bounds from %+v: %+v", start, coordinate)
			}
			if seen[coordinate] {
				t.Fatalf("duplicate candidate from %+v: %+v", start, coordinate)
			}
			seen[coordinate] = true
		}
	}
}

func TestCalculateWorldMarchSecondsUsesDistanceAndSpeed(t *testing.T) {
	now := time.Now()
	if got := CalculateWorldMarchSeconds(1, 1, now, nil); got != 300 {
		t.Fatalf("expected speed 1 one grid to take 300 seconds, got %d", got)
	}
	if got := CalculateWorldMarchSeconds(1, 5, now, nil); got != 60 {
		t.Fatalf("expected speed 5 one grid to take 60 seconds, got %d", got)
	}
	if got := CalculateWorldMarchSeconds(1, 15, now, nil); got != 20 {
		t.Fatalf("expected speed 15 one grid to take 20 seconds, got %d", got)
	}
	if got := CalculateWorldMarchSeconds(7, 5, now, nil); got != 420 {
		t.Fatalf("expected distance 7 speed 5 to take 420 seconds, got %d", got)
	}
	if got := CalculateWorldMarchSeconds(198, 1, now, nil); got != maxWorldMarchSeconds {
		t.Fatalf("expected far slow world march to clamp to 3 hours, got %d", got)
	}
}

func TestMigrateWorldPositionsIsRepeatable(t *testing.T) {
	now := time.Now()
	repo := NewMemoryRepository()
	if err := repo.CreateAccount(Account{ID: "account_world_migrate", Username: "world_migrate", CreatedAt: now}); err != nil {
		t.Fatalf("CreateAccount failed: %v", err)
	}
	first := newPlayerState("player_world_migrate_a", "甲", "wei", "caocao", now)
	second := newPlayerState("player_world_migrate_b", "乙", "shu", "liubei", now)
	if err := repo.CreatePlayer("account_world_migrate", first, now); err != nil {
		t.Fatalf("CreatePlayer first failed: %v", err)
	}
	if err := repo.CreatePlayer("account_world_migrate", second, now); err != nil {
		t.Fatalf("CreatePlayer second failed: %v", err)
	}
	svc := NewServiceWithRepository(repo)
	firstResult, err := svc.MigrateWorldPositions()
	if err != nil {
		t.Fatalf("MigrateWorldPositions first failed: %v", err)
	}
	if firstResult.Total != 2 || firstResult.Created != 2 || firstResult.Skipped != 0 || firstResult.Failed != 0 {
		t.Fatalf("unexpected first migration result: %+v", firstResult)
	}
	firstPosition, err := repo.GetWorldPosition(first.Player.ID)
	if err != nil {
		t.Fatalf("GetWorldPosition first failed: %v", err)
	}
	expectedFirst := LegacyWorldCoordinateForPlayer(first.Player.ID)
	if firstPosition.X != expectedFirst.X || firstPosition.Y != expectedFirst.Y {
		t.Fatalf("expected migration to preserve legacy coordinate %+v, got %+v", expectedFirst, firstPosition)
	}
	secondResult, err := svc.MigrateWorldPositions()
	if err != nil {
		t.Fatalf("MigrateWorldPositions second failed: %v", err)
	}
	if secondResult.Total != 2 || secondResult.Created != 0 || secondResult.Skipped != 2 || secondResult.Failed != 0 {
		t.Fatalf("unexpected repeat migration result: %+v", secondResult)
	}
}

func TestMigrateWorldPositionsCountsCoordinateConflicts(t *testing.T) {
	now := time.Now()
	repo := NewMemoryRepository()
	if err := repo.CreateAccount(Account{ID: "account_world_migrate_conflict", Username: "world_migrate_conflict", CreatedAt: now}); err != nil {
		t.Fatalf("CreateAccount failed: %v", err)
	}
	occupied := newPlayerState("player_world_migrate_occupied", "占位", "wei", "caocao", now)
	migrating := newPlayerState("player_world_migrate_conflicted", "冲突", "shu", "liubei", now)
	if err := repo.CreatePlayer("account_world_migrate_conflict", occupied, now); err != nil {
		t.Fatalf("CreatePlayer occupied failed: %v", err)
	}
	if err := repo.CreatePlayer("account_world_migrate_conflict", migrating, now); err != nil {
		t.Fatalf("CreatePlayer migrating failed: %v", err)
	}
	expected := LegacyWorldCoordinateForPlayer(migrating.Player.ID)
	if _, err := repo.AssignWorldPosition(occupied.Player.ID, defaultWorldID, expected.X, expected.Y, "test"); err != nil {
		t.Fatalf("AssignWorldPosition occupied failed: %v", err)
	}
	result, err := NewServiceWithRepository(repo).MigrateWorldPositions()
	if err != nil {
		t.Fatalf("MigrateWorldPositions failed: %v", err)
	}
	if result.Total != 2 || result.Created != 1 || result.Skipped != 1 || result.Conflicts != 1 || result.Failed != 0 {
		t.Fatalf("unexpected migration conflict result: %+v", result)
	}
	if len(result.ConflictDetails) != 1 || !strings.Contains(result.ConflictDetails[0], migrating.Player.ID) {
		t.Fatalf("expected conflict detail for migrating player, got %+v", result.ConflictDetails)
	}
	position, err := repo.GetWorldPosition(migrating.Player.ID)
	if err != nil {
		t.Fatalf("GetWorldPosition migrating failed: %v", err)
	}
	nearest := WorldMapCoordinateCandidates(expected.X, expected.Y)[1]
	if position.X != nearest.X || position.Y != nearest.Y {
		t.Fatalf("expected migration conflict to use nearest free coordinate %+v, got %+v", nearest, position)
	}
	wantDetail := "(" + strconv.Itoa(expected.X) + "," + strconv.Itoa(expected.Y) + ") -> (" + strconv.Itoa(nearest.X) + "," + strconv.Itoa(nearest.Y) + ")"
	if !strings.Contains(result.ConflictDetails[0], wantDetail) {
		t.Fatalf("expected conflict detail to include %s, got %+v", wantDetail, result.ConflictDetails)
	}
}

func TestMigrateWorldPositionsPassesLegacyCoordinateAsPreferred(t *testing.T) {
	source, err := os.ReadFile("service_world_map.go")
	if err != nil {
		t.Fatalf("ReadFile service_world_map.go failed: %v", err)
	}
	text := string(source)
	if !strings.Contains(text, "expected := LegacyWorldCoordinateForPlayer(player.ID)") {
		t.Fatalf("expected migration to calculate legacy coordinate explicitly")
	}
	if !strings.Contains(text, `s.ensureWorldPosition(player.ID, "migration", &expected)`) {
		t.Fatalf("expected migration to pass legacy coordinate as preferred start")
	}
}

func TestMigrateWorldPositionsReturnsFirstFailureError(t *testing.T) {
	now := time.Now()
	repo := NewMemoryRepository()
	if err := repo.CreateAccount(Account{ID: "account_world_migrate_error", Username: "world_migrate_error", CreatedAt: now}); err != nil {
		t.Fatalf("CreateAccount failed: %v", err)
	}
	player := newPlayerState("player_world_migrate_error", "错误", "wei", "caocao", now)
	if err := repo.CreatePlayer("account_world_migrate_error", player, now); err != nil {
		t.Fatalf("CreatePlayer failed: %v", err)
	}
	storageErr := errors.New("world position storage unavailable")
	result, err := NewServiceWithRepository(&worldMapGetPositionErrorRepo{MemoryRepository: repo, err: storageErr}).MigrateWorldPositions()
	if !errors.Is(err, storageErr) {
		t.Fatalf("expected storage error, got %v", err)
	}
	if result.Failed != 1 || len(result.Failures) != 1 || !strings.Contains(result.Failures[0], storageErr.Error()) {
		t.Fatalf("unexpected migration failure result: %+v", result)
	}
}

func TestLegacyWorldCoordinateForPlayerUsesPvpMapping(t *testing.T) {
	playerID := " player_world_legacy "
	legacy := pvpWorldPositionForPlayer(playerID)
	coordinate := LegacyWorldCoordinateForPlayer(playerID)
	if coordinate.X != legacy.X || coordinate.Y != legacy.Y {
		t.Fatalf("expected current in-range PVP coordinate to stay unchanged, got coordinate=%+v legacy=%+v", coordinate, legacy)
	}
	trimmed := LegacyWorldCoordinateForPlayer("player_world_legacy")
	if coordinate.X != trimmed.X || coordinate.Y != trimmed.Y {
		t.Fatalf("expected legacy coordinate to trim player id, got spaced=%+v trimmed=%+v", coordinate, trimmed)
	}
}

func TestPvpCompatibilityWorldPositionUsesCurrentWorldSize(t *testing.T) {
	position := pvpWorldPositionForPlayer("player_world_region_compat")
	if position.X < 0 || position.X >= defaultWorldWidth || position.Y < 0 || position.Y >= defaultWorldHeight {
		t.Fatalf("expected PVP compatibility coordinate inside current world, got %+v", position)
	}
	if pvpRegionID(0, 0) != 1 {
		t.Fatalf("expected northwest 20-grid region to be 1, got %d", pvpRegionID(0, 0))
	}
	if pvpRegionID(99, 99) != 25 {
		t.Fatalf("expected southeast 20-grid region in 100x100 world to be 25, got %d", pvpRegionID(99, 99))
	}
	if pvpRegionID(100, 100) != 25 {
		t.Fatalf("expected out-of-range compatibility coordinate to clamp into southeast region, got %d", pvpRegionID(100, 100))
	}
}

func TestMapLegacyWorldCoordinateToCurrentSupportsSignedHistory(t *testing.T) {
	cases := []struct {
		name string
		old  WorldCoordinate
		want WorldCoordinate
	}{
		{name: "keep current zero based coordinate", old: WorldCoordinate{X: 0, Y: 99}, want: WorldCoordinate{X: 0, Y: 99}},
		{name: "map negative edge", old: WorldCoordinate{X: -100, Y: -100}, want: WorldCoordinate{X: 0, Y: 0}},
		{name: "map signed origin", old: WorldCoordinate{X: 0, Y: 0}, want: WorldCoordinate{X: 0, Y: 0}},
		{name: "map positive edge", old: WorldCoordinate{X: 100, Y: 100}, want: WorldCoordinate{X: 99, Y: 99}},
		{name: "map mixed signed coordinate", old: WorldCoordinate{X: -50, Y: 50}, want: WorldCoordinate{X: 24, Y: 50}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := mapLegacyWorldCoordinateToCurrent(tc.old); got != tc.want {
				t.Fatalf("expected %+v to map to %+v, got %+v", tc.old, tc.want, got)
			}
		})
	}
	if got := mapLegacyWorldAxisToCurrent(0, defaultWorldWidth); got != 0 {
		t.Fatalf("expected in-range zero coordinate to stay 0, got %d", got)
	}
	if got := mapLegacyWorldAxisToCurrent(100, defaultWorldWidth); got != 99 {
		t.Fatalf("expected signed edge 100 to map to 99, got %d", got)
	}
	if got := mapLegacyWorldAxisToCurrent(-1, defaultWorldWidth); got != 49 {
		t.Fatalf("expected signed coordinate -1 to map near middle, got %d", got)
	}
}

type worldMapGetPositionErrorRepo struct {
	*MemoryRepository
	err error
}

func (r *worldMapGetPositionErrorRepo) GetWorldPosition(playerID string) (WorldPosition, error) {
	return WorldPosition{}, r.err
}

func TestWorldMapFullReturnsExplicitError(t *testing.T) {
	now := time.Now()
	repo := NewMemoryRepository()
	if err := repo.CreateAccount(Account{ID: "account_world_full", Username: "world_full", CreatedAt: now}); err != nil {
		t.Fatalf("CreateAccount failed: %v", err)
	}
	state := newPlayerState("player_world_full", "满图", "wei", "caocao", now)
	if err := repo.CreatePlayer("account_world_full", state, now); err != nil {
		t.Fatalf("CreatePlayer failed: %v", err)
	}
	for y := 0; y < defaultWorldHeight; y++ {
		for x := 0; x < defaultWorldWidth; x++ {
			playerID := "occupied"
			repo.worldPositions[playerID+time.Duration(y*defaultWorldWidth+x).String()] = WorldPosition{
				PlayerID:   playerID,
				WorldID:    defaultWorldID,
				X:          x,
				Y:          y,
				AssignedBy: "test",
			}
		}
	}
	_, err := repo.EnsureWorldPosition(state.Player.ID, "test", nil)
	if !errors.Is(err, ErrWorldMapFull) {
		t.Fatalf("expected ErrWorldMapFull, got %v", err)
	}
}

func TestCreatePlayerCleansUpWhenWorldMapIsFull(t *testing.T) {
	setTestFactionsAndGenerals(t, FactionsConfig{
		"wei": {Generals: []GeneralInfo{{ID: "caocao", Name: "曹操"}}},
	}, GeneralsConfig{
		Enabled: true,
		Heroes: map[string]GeneralHeroConfig{
			"caocao": {ID: "caocao", Name: "曹操", Faction: "wei", Enabled: true},
		},
	})
	previous := generateWorldMapCreateCoordinate
	generateWorldMapCreateCoordinate = func() (WorldCoordinate, error) {
		return WorldCoordinate{X: 0, Y: 0}, nil
	}
	defer func() {
		generateWorldMapCreateCoordinate = previous
	}()
	now := time.Now()
	repo := NewMemoryRepository()
	svc := NewServiceWithRepository(repo)
	if err := repo.CreateAccount(Account{ID: "account_world_create_full", Username: "world_create_full", CreatedAt: now}); err != nil {
		t.Fatalf("CreateAccount failed: %v", err)
	}
	for y := 0; y < defaultWorldHeight; y++ {
		for x := 0; x < defaultWorldWidth; x++ {
			playerID := "occupied_create_" + strconv.Itoa(y*defaultWorldWidth+x)
			repo.worldPositions[playerID] = WorldPosition{
				PlayerID:   playerID,
				WorldID:    defaultWorldID,
				X:          x,
				Y:          y,
				AssignedBy: "test",
			}
		}
	}
	playerID, _, err := svc.CreatePlayer("account_world_create_full", "满图创建", "wei", "caocao")
	if !errors.Is(err, ErrWorldMapFull) {
		t.Fatalf("expected ErrWorldMapFull, got playerID=%s err=%v", playerID, err)
	}
	players, err := svc.ListPlayers("account_world_create_full")
	if err != nil {
		t.Fatalf("ListPlayers failed: %v", err)
	}
	if len(players) != 0 {
		t.Fatalf("expected failed create to leave no player, got %+v", players)
	}
}

func TestDeletePlayerRemovesWorldPosition(t *testing.T) {
	now := time.Now()
	repo := NewMemoryRepository()
	svc := NewServiceWithRepository(repo)
	if err := repo.CreateAccount(Account{ID: "account_world_delete_position", Username: "world_delete_position", CreatedAt: now}); err != nil {
		t.Fatalf("CreateAccount failed: %v", err)
	}
	state := newPlayerState("player_world_delete_position", "删坐标", "wei", "caocao", now)
	state.DeleteRequestedAt = now.Add(-2 * time.Hour).UTC().Format(resourceDateLayout)
	state.DeleteScheduledAt = now.Add(-time.Hour).UTC().Format(resourceDateLayout)
	if err := repo.CreatePlayer("account_world_delete_position", state, now); err != nil {
		t.Fatalf("CreatePlayer failed: %v", err)
	}
	if _, err := repo.AssignWorldPosition(state.Player.ID, defaultWorldID, 9, 9, "test"); err != nil {
		t.Fatalf("AssignWorldPosition failed: %v", err)
	}
	if _, err := svc.DeletePlayer(state.Player.ID); err != nil {
		t.Fatalf("DeletePlayer failed: %v", err)
	}
	if _, err := repo.GetWorldPosition(state.Player.ID); !errors.Is(err, ErrPlayerNotFound) {
		t.Fatalf("expected world position to be deleted, got %v", err)
	}
	replacement := newPlayerState("player_world_delete_replacement", "新坐标", "shu", "liubei", now)
	if err := repo.CreatePlayer("account_world_delete_position", replacement, now); err != nil {
		t.Fatalf("CreatePlayer replacement failed: %v", err)
	}
	if _, err := repo.AssignWorldPosition(replacement.Player.ID, defaultWorldID, 9, 9, "test"); err != nil {
		t.Fatalf("expected deleted coordinate to be reusable, got %v", err)
	}
}

func TestGetWorldMapPlayerCityTarget(t *testing.T) {
	svc, repo, attacker, defender := newPvpTestService(t)
	if _, err := repo.AssignWorldPosition(attacker.Player.ID, defaultWorldID, 20, 20, "test"); err != nil {
		t.Fatalf("AssignWorldPosition attacker failed: %v", err)
	}
	if _, err := repo.AssignWorldPosition(defender.Player.ID, defaultWorldID, 21, 22, "test"); err != nil {
		t.Fatalf("AssignWorldPosition defender failed: %v", err)
	}
	target, err := svc.GetWorldMapPlayerCityTarget(attacker.Player.ID, defender.Player.ID)
	if err != nil {
		t.Fatalf("GetWorldMapPlayerCityTarget failed: %v", err)
	}
	if target.PlayerID != defender.Player.ID || target.Distance != 3 || target.Direction != "东南" {
		t.Fatalf("unexpected target detail: %+v", target)
	}
	if target.CanAttack != true || target.CanPlunder != true || target.CanScout != true {
		t.Fatalf("expected target operations enabled, got %+v", target)
	}
	payload, err := json.Marshal(target)
	if err != nil {
		t.Fatalf("marshal target detail: %v", err)
	}
	body := string(payload)
	for _, hiddenField := range []string{"totalArmy", "resources", "generals", "garrison", "troops"} {
		if strings.Contains(body, hiddenField) {
			t.Fatalf("world map target detail leaked hidden field %q in %s", hiddenField, body)
		}
	}
	selfTarget, err := svc.GetWorldMapPlayerCityTarget(attacker.Player.ID, attacker.Player.ID)
	if err != nil {
		t.Fatalf("GetWorldMapPlayerCityTarget self failed: %v", err)
	}
	if selfTarget.Relation != WorldRelationSelf || selfTarget.Status != WorldTargetStatusSelf || selfTarget.Distance != 0 || selfTarget.Direction != "原地" {
		t.Fatalf("unexpected self target detail: %+v", selfTarget)
	}
	if selfTarget.CanScout || selfTarget.CanAttack || selfTarget.CanPlunder || selfTarget.CanReinforce {
		t.Fatalf("expected self target detail actions disabled, got %+v", selfTarget)
	}
	if selfTarget.ScoutReason != "自己的城池" || selfTarget.AttackReason != "自己的城池" || selfTarget.PlunderReason != "自己的城池" || selfTarget.ReinforceReason != "自己的城池" {
		t.Fatalf("expected self target detail reasons, got %+v", selfTarget)
	}
}

func TestAdminAssignWorldPositionRejectsOccupiedCoordinate(t *testing.T) {
	svc, repo, attacker, defender := newPvpTestService(t)
	if _, err := repo.AssignWorldPosition(attacker.Player.ID, defaultWorldID, 30, 30, "test"); err != nil {
		t.Fatalf("AssignWorldPosition attacker failed: %v", err)
	}
	if _, err := svc.AdminAssignWorldPosition(defender.Player.ID, 30, 30); err != ErrInvalidWorldCoordinate {
		t.Fatalf("expected occupied coordinate error, got %v", err)
	}
	position, err := svc.AdminAssignWorldPosition(defender.Player.ID, 31, 30)
	if err != nil {
		t.Fatalf("AdminAssignWorldPosition failed: %v", err)
	}
	if position.X != 31 || position.Y != 30 || position.AssignedBy != "admin" {
		t.Fatalf("unexpected admin position: %+v", position)
	}
	loaded, err := svc.AdminGetWorldPosition(defender.Player.ID)
	if err != nil {
		t.Fatalf("AdminGetWorldPosition failed: %v", err)
	}
	if loaded.X != 31 || loaded.Y != 30 {
		t.Fatalf("expected saved admin position, got %+v", loaded)
	}
}

func TestAdminCheckWorldCoordinateReportsOccupancy(t *testing.T) {
	svc, repo, attacker, _ := newPvpTestService(t)
	if _, err := repo.AssignWorldPosition(attacker.Player.ID, defaultWorldID, 32, 33, "test"); err != nil {
		t.Fatalf("AssignWorldPosition attacker failed: %v", err)
	}
	occupied, err := svc.AdminCheckWorldCoordinate(32, 33)
	if err != nil {
		t.Fatalf("AdminCheckWorldCoordinate occupied failed: %v", err)
	}
	if occupied.WorldID != defaultWorldID || occupied.X != 32 || occupied.Y != 33 || !occupied.Occupied || occupied.PlayerID != attacker.Player.ID {
		t.Fatalf("unexpected occupied coordinate check: %+v", occupied)
	}
	empty, err := svc.AdminCheckWorldCoordinate(34, 33)
	if err != nil {
		t.Fatalf("AdminCheckWorldCoordinate empty failed: %v", err)
	}
	if empty.WorldID != defaultWorldID || empty.X != 34 || empty.Y != 33 || empty.Occupied || empty.PlayerID != "" {
		t.Fatalf("unexpected empty coordinate check: %+v", empty)
	}
	if _, err := svc.AdminCheckWorldCoordinate(100, 0); err != ErrInvalidWorldCoordinate {
		t.Fatalf("expected invalid coordinate error, got %v", err)
	}
}

func TestAdminWorldPositionRejectsInvalidOrMissingPlayer(t *testing.T) {
	svc, _, attacker, _ := newPvpTestService(t)
	for _, coordinate := range []WorldCoordinate{{X: -1, Y: 0}, {X: 0, Y: 100}} {
		if _, err := svc.AdminAssignWorldPosition(attacker.Player.ID, coordinate.X, coordinate.Y); err != ErrInvalidWorldCoordinate {
			t.Fatalf("expected invalid coordinate error for %+v, got %v", coordinate, err)
		}
	}
	if _, err := svc.AdminAssignWorldPosition("missing_player", 1, 1); !errors.Is(err, ErrPlayerNotFound) {
		t.Fatalf("expected missing player error on assign, got %v", err)
	}
	if _, err := svc.AdminGetWorldPosition("missing_player"); !errors.Is(err, ErrPlayerNotFound) {
		t.Fatalf("expected missing player error on query, got %v", err)
	}
}

func TestAdminWorldMapOccupancy(t *testing.T) {
	svc, repo, attacker, defender := newPvpTestService(t)
	if _, err := repo.AssignWorldPosition(attacker.Player.ID, defaultWorldID, 40, 40, "test"); err != nil {
		t.Fatalf("AssignWorldPosition attacker failed: %v", err)
	}
	if _, err := repo.AssignWorldPosition(defender.Player.ID, defaultWorldID, 41, 40, "test"); err != nil {
		t.Fatalf("AssignWorldPosition defender failed: %v", err)
	}
	stats, err := svc.AdminWorldMapOccupancy()
	if err != nil {
		t.Fatalf("AdminWorldMapOccupancy failed: %v", err)
	}
	if stats.WorldID != defaultWorldID || stats.Width != 100 || stats.Height != 100 || stats.TotalCells != 10000 {
		t.Fatalf("unexpected world metadata: %+v", stats)
	}
	if stats.OccupiedCells != 2 || stats.AvailableCells != 9998 {
		t.Fatalf("unexpected occupancy counts: %+v", stats)
	}
	if stats.OccupancyRate != 0.0002 {
		t.Fatalf("unexpected occupancy rate: %+v", stats)
	}
}
