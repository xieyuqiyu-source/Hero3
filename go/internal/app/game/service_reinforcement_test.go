// 本文件测试增援系统的核心生命周期和资产边界。
package game

import (
	"errors"
	"testing"
	"time"
)

func TestSendReinforcementConsumesArmyAndReservesGeneral(t *testing.T) {
	svc, repo, from, to := newReinforcementTestService(t)
	from.Army = []ArmyUnit{{UnitType: "weiInfantry", Amount: 100}}
	repo.players[from.Player.ID] = from

	result, err := svc.SendReinforcement(SendReinforcementRequest{
		FromPlayerID:   from.Player.ID,
		TargetPlayerID: to.Player.ID,
		Troops:         map[string]int{"weiInfantry": 30},
		GeneralIDs:     []string{"caocao"},
	})
	if err != nil {
		t.Fatalf("SendReinforcement failed: %v", err)
	}
	if result.Reinforcement.Status != ReinforcementStatusMarching {
		t.Fatalf("expected marching, got %s", result.Reinforcement.Status)
	}
	if got := result.Patch.Army[0].Amount; got != 70 {
		t.Fatalf("expected remaining army 70, got %d", got)
	}
	if len(result.Patch.GeneralAssignments) != 2 {
		t.Fatalf("expected main + reinforcement assignment, got %+v", result.Patch.GeneralAssignments)
	}
	if _, err := svc.SendReinforcement(SendReinforcementRequest{
		FromPlayerID:   from.Player.ID,
		TargetPlayerID: to.Player.ID,
		Troops:         map[string]int{"weiInfantry": 1},
		GeneralIDs:     []string{"caocao"},
	}); !errors.Is(err, ErrGeneralBusy) {
		t.Fatalf("expected ErrGeneralBusy, got %v", err)
	}
}

func TestReinforcementSourceSlotLimitAndSameSourceAppend(t *testing.T) {
	svc, repo, from, to := newReinforcementTestService(t)
	from.Army = []ArmyUnit{{UnitType: "weiInfantry", Amount: 500}}
	repo.players[from.Player.ID] = from
	for i := 0; i < 4; i++ {
		player := newPlayerState("slot_from_"+string(rune('a'+i)), "来源", "wei", "caocao", time.Now())
		player.Army = []ArmyUnit{{UnitType: "weiInfantry", Amount: 20}}
		repo.players[player.Player.ID] = player
		if _, err := svc.SendReinforcement(SendReinforcementRequest{
			FromPlayerID:   player.Player.ID,
			TargetPlayerID: to.Player.ID,
			Troops:         map[string]int{"weiInfantry": 1},
		}); err != nil {
			t.Fatalf("seed reinforcement failed: %v", err)
		}
	}
	if _, err := svc.SendReinforcement(SendReinforcementRequest{
		FromPlayerID:   from.Player.ID,
		TargetPlayerID: to.Player.ID,
		Troops:         map[string]int{"weiInfantry": 1},
	}); err != nil {
		t.Fatalf("fifth source should be accepted: %v", err)
	}
	if _, err := svc.SendReinforcement(SendReinforcementRequest{
		FromPlayerID:   from.Player.ID,
		TargetPlayerID: to.Player.ID,
		Troops:         map[string]int{"weiInfantry": 1},
	}); err != nil {
		t.Fatalf("same source append should be accepted: %v", err)
	}
	extra := newPlayerState("slot_from_extra", "第六", "wei", "caocao", time.Now())
	extra.Army = []ArmyUnit{{UnitType: "weiInfantry", Amount: 20}}
	repo.players[extra.Player.ID] = extra
	if _, err := svc.SendReinforcement(SendReinforcementRequest{
		FromPlayerID:   extra.Player.ID,
		TargetPlayerID: to.Player.ID,
		Troops:         map[string]int{"weiInfantry": 1},
	}); !errors.Is(err, ErrReinforcementSlotFull) {
		t.Fatalf("expected slot full, got %v", err)
	}
}

func TestObtainedGarrisonOccupiesOwnSlot(t *testing.T) {
	svc, repo, _, to := newReinforcementTestService(t)
	if _, err := svc.CreateGarrisonDetachment(CreateGarrisonDetachmentRequest{
		OwnerPlayerID: to.Player.ID,
		HostPlayerID:  to.Player.ID,
		SourceType:    GarrisonSourceCaptured,
		SourceFaction: "wu",
		Troops:        map[string]int{"wuArcher": 25},
	}); err != nil {
		t.Fatalf("CreateGarrisonDetachment failed: %v", err)
	}
	for i := 0; i < 4; i++ {
		player := newPlayerState("own_slot_from_"+string(rune('a'+i)), "来源", "wei", "caocao", time.Now())
		player.Army = []ArmyUnit{{UnitType: "weiInfantry", Amount: 20}}
		repo.players[player.Player.ID] = player
		if _, err := svc.SendReinforcement(SendReinforcementRequest{
			FromPlayerID:   player.Player.ID,
			TargetPlayerID: to.Player.ID,
			Troops:         map[string]int{"weiInfantry": 1},
		}); err != nil {
			t.Fatalf("seed reinforcement failed: %v", err)
		}
	}
	extra := newPlayerState("own_slot_from_extra", "第六", "wei", "caocao", time.Now())
	extra.Army = []ArmyUnit{{UnitType: "weiInfantry", Amount: 20}}
	repo.players[extra.Player.ID] = extra
	if _, err := svc.SendReinforcement(SendReinforcementRequest{
		FromPlayerID:   extra.Player.ID,
		TargetPlayerID: to.Player.ID,
		Troops:         map[string]int{"weiInfantry": 1},
	}); !errors.Is(err, ErrReinforcementSlotFull) {
		t.Fatalf("expected own obtained garrison to occupy one slot, got %v", err)
	}
}

func TestCreateCapturedGarrisonDoesNotEnterRegularArmy(t *testing.T) {
	svc, repo, _, to := newReinforcementTestService(t)
	beforeArmy := len(to.Army)
	result, err := svc.CreateGarrisonDetachment(CreateGarrisonDetachmentRequest{
		OwnerPlayerID: to.Player.ID,
		HostPlayerID:  to.Player.ID,
		SourceType:    GarrisonSourceCaptured,
		SourceID:      "beauty_trap_test",
		SourceFaction: "wu",
		Troops:        map[string]int{"wuArcher": 25},
		Metadata:      map[string]any{"reason": "test"},
	})
	if err != nil {
		t.Fatalf("CreateGarrisonDetachment failed: %v", err)
	}
	if result.Reinforcement.Status != ReinforcementStatusStationed {
		t.Fatalf("expected stationed, got %s", result.Reinforcement.Status)
	}
	if result.Reinforcement.SourceType != GarrisonSourceObtained {
		t.Fatalf("expected obtained source, got %s", result.Reinforcement.SourceType)
	}
	if result.Reinforcement.Rules.CanRecall || result.Reinforcement.Rules.CanExpel || result.Reinforcement.Rules.CanReturn {
		t.Fatalf("captured garrison should not behave like reinforcement: %+v", result.Reinforcement.Rules)
	}
	state, err := repo.GetState(to.Player.ID)
	if err != nil {
		t.Fatalf("GetState failed: %v", err)
	}
	if len(state.Army) != beforeArmy {
		t.Fatalf("captured garrison should not enter regular army, got %+v", state.Army)
	}
	defenders, err := svc.BuildDefenseReinforcementUnits(to.Player.ID)
	if err != nil {
		t.Fatalf("BuildDefenseReinforcementUnits failed: %v", err)
	}
	if len(defenders) != 1 || defenders[0].SourceTags["source_type"] != GarrisonSourceObtained || defenders[0].Troops["wuArcher"] != 25 {
		t.Fatalf("unexpected captured defenders: %+v", defenders)
	}
	if _, err := svc.RecallReinforcement(to.Player.ID, result.Reinforcement.ID); !errors.Is(err, ErrInvalidReinforcement) {
		t.Fatalf("captured garrison should not be recalled, got %v", err)
	}
}

func TestCreateObtainedGarrisonMergesIntoSingleOwnedTeam(t *testing.T) {
	svc, _, _, to := newReinforcementTestService(t)
	first, err := svc.CreateGarrisonDetachment(CreateGarrisonDetachmentRequest{
		OwnerPlayerID: to.Player.ID,
		HostPlayerID:  to.Player.ID,
		SourceType:    GarrisonSourceCaptured,
		SourceID:      "capture_a",
		SourceFaction: "wu",
		Troops:        map[string]int{"wuArcher": 25},
	})
	if err != nil {
		t.Fatalf("CreateGarrisonDetachment first failed: %v", err)
	}
	second, err := svc.CreateGarrisonDetachment(CreateGarrisonDetachmentRequest{
		OwnerPlayerID: to.Player.ID,
		HostPlayerID:  to.Player.ID,
		SourceType:    GarrisonSourceEventReward,
		SourceID:      "event_b",
		SourceFaction: "shu",
		Troops:        map[string]int{"wuArcher": 5, "southernElephant": 11},
	})
	if err != nil {
		t.Fatalf("CreateGarrisonDetachment second failed: %v", err)
	}
	if first.Reinforcement.ID != second.Reinforcement.ID {
		t.Fatalf("expected obtained garrison to merge into same record, got %s and %s", first.Reinforcement.ID, second.Reinforcement.ID)
	}
	if second.Reinforcement.RemainingTroops["wuArcher"] != 30 || second.Reinforcement.RemainingTroops["southernElephant"] != 11 {
		t.Fatalf("unexpected merged troops: %+v", second.Reinforcement.RemainingTroops)
	}
	received, err := svc.ListReceivedReinforcements(to.Player.ID)
	if err != nil {
		t.Fatalf("ListReceivedReinforcements failed: %v", err)
	}
	if len(received.Items) != 1 || received.Items[0].SourceType != GarrisonSourceObtained {
		t.Fatalf("expected one obtained display team, got %+v", received.Items)
	}
	if received.Items[0].ID != "garrison_obtained_"+to.Player.ID {
		t.Fatalf("expected stable obtained garrison id, got %s", received.Items[0].ID)
	}
}

func TestReceivedGarrisonsDisplayObtainedAndReinforcementSources(t *testing.T) {
	svc, repo, from, to := newReinforcementTestService(t)
	from.Army = []ArmyUnit{{UnitType: "weiInfantry", Amount: 100}}
	repo.players[from.Player.ID] = from
	if _, err := svc.CreateGarrisonDetachment(CreateGarrisonDetachmentRequest{
		OwnerPlayerID: to.Player.ID,
		HostPlayerID:  to.Player.ID,
		SourceType:    GarrisonSourceCaptured,
		SourceFaction: "wu",
		Troops:        map[string]int{"wuArcher": 25},
	}); err != nil {
		t.Fatalf("CreateGarrisonDetachment failed: %v", err)
	}
	reinforcement, err := svc.SendReinforcement(SendReinforcementRequest{
		FromPlayerID:   from.Player.ID,
		TargetPlayerID: to.Player.ID,
		Troops:         map[string]int{"weiInfantry": 40},
	})
	if err != nil {
		t.Fatalf("SendReinforcement failed: %v", err)
	}
	forceReinforcementDue(t, repo, reinforcement.Reinforcement.ID, true)
	if _, err := svc.MarkReinforcementArrived(reinforcement.Reinforcement.ID); err != nil {
		t.Fatalf("MarkReinforcementArrived failed: %v", err)
	}
	received, err := svc.ListReceivedReinforcements(to.Player.ID)
	if err != nil {
		t.Fatalf("ListReceivedReinforcements failed: %v", err)
	}
	if len(received.Items) != 2 {
		t.Fatalf("expected obtained + reinforcement display teams, got %+v", received.Items)
	}
	counts := map[string]int{}
	for _, item := range received.Items {
		counts[item.SourceType]++
	}
	if counts[GarrisonSourceObtained] != 1 || counts[GarrisonSourceReinforcement] != 1 {
		t.Fatalf("expected one obtained and one reinforcement source, got %+v", counts)
	}
}

func TestReinforcementArrivalBattleLossAndReturnIdempotent(t *testing.T) {
	svc, repo, from, to := newReinforcementTestService(t)
	from.Army = []ArmyUnit{{UnitType: "weiInfantry", Amount: 100}}
	repo.players[from.Player.ID] = from
	result, err := svc.SendReinforcement(SendReinforcementRequest{
		FromPlayerID:    from.Player.ID,
		TargetPlayerID:  to.Player.ID,
		Troops:          map[string]int{"weiInfantry": 40},
		SpeedMultiplier: 2,
	})
	if err != nil {
		t.Fatalf("SendReinforcement failed: %v", err)
	}
	if result.Reinforcement.MarchSeconds != 5346 {
		t.Fatalf("expected speed and relay station to shorten march to 5346, got %d", result.Reinforcement.MarchSeconds)
	}
	forceReinforcementDue(t, repo, result.Reinforcement.ID, true)
	arrived, err := svc.MarkReinforcementArrived(result.Reinforcement.ID)
	if err != nil {
		t.Fatalf("MarkReinforcementArrived failed: %v", err)
	}
	if arrived.Status != ReinforcementStatusStationed {
		t.Fatalf("expected stationed, got %s", arrived.Status)
	}
	defenders, err := svc.BuildDefenseReinforcementUnits(to.Player.ID)
	if err != nil {
		t.Fatalf("BuildDefenseReinforcementUnits failed: %v", err)
	}
	if len(defenders) != 1 || defenders[0].Troops["weiInfantry"] != 40 {
		t.Fatalf("unexpected defense reinforcements: %+v", defenders)
	}
	if err := svc.ApplyReinforcementBattleResult("br_test", []ReinforcementLoss{{
		ReinforcementID: result.Reinforcement.ID,
		FromPlayerID:    from.Player.ID,
		UnitType:        "weiInfantry",
		LostAmount:      40,
	}}); err != nil {
		t.Fatalf("ApplyReinforcementBattleResult failed: %v", err)
	}
	record, err := svc.GetReinforcement(from.Player.ID, result.Reinforcement.ID)
	if err != nil {
		t.Fatalf("GetReinforcement failed: %v", err)
	}
	if record.Status != ReinforcementStatusReturning || !record.IsAnnihilated {
		t.Fatalf("expected annihilated returning record, got %+v", record)
	}
	forceReinforcementDue(t, repo, result.Reinforcement.ID, false)
	if _, err := svc.CompleteReinforcementReturn(result.Reinforcement.ID); err != nil {
		t.Fatalf("CompleteReinforcementReturn failed: %v", err)
	}
	if _, err := svc.CompleteReinforcementReturn(result.Reinforcement.ID); err != nil {
		t.Fatalf("second CompleteReinforcementReturn should be idempotent: %v", err)
	}
	next, err := repo.GetState(from.Player.ID)
	if err != nil {
		t.Fatalf("GetState failed: %v", err)
	}
	if got := next.Army[0].Amount; got != 60 {
		t.Fatalf("expected no dead troops returned, got army %d", got)
	}
}

func newReinforcementTestService(t *testing.T) (*Service, *MemoryRepository, GameState, GameState) {
	t.Helper()
	setTestCombatUnitsConfig(t)
	now := time.Now()
	repo := NewMemoryRepository()
	account := Account{ID: "account_reinforcement", Username: "reinforcement", PasswordHash: "hash", CreatedAt: now}
	if err := repo.CreateAccount(account); err != nil {
		t.Fatalf("CreateAccount failed: %v", err)
	}
	from := newPlayerState("player_reinforcement_from", "派出方", "wei", "caocao", now)
	to := newPlayerState("player_reinforcement_to", "接收方", "shu", "liubei", now)
	if err := repo.CreatePlayer(account.ID, from, now); err != nil {
		t.Fatalf("CreatePlayer from failed: %v", err)
	}
	if err := repo.CreatePlayer(account.ID, to, now); err != nil {
		t.Fatalf("CreatePlayer to failed: %v", err)
	}
	return NewServiceWithRepository(repo), repo, from, to
}

func forceReinforcementDue(t *testing.T, repo *MemoryRepository, reinforcementID string, outbound bool) {
	t.Helper()
	repo.mu.Lock()
	defer repo.mu.Unlock()
	record := repo.reinforcements[reinforcementID]
	old := time.Now().Add(-4 * time.Hour).UTC().Format(resourceDateLayout)
	if outbound {
		record.SentAt = old
	} else {
		record.ReturnStartedAt = old
	}
	repo.reinforcements[reinforcementID] = record
}
