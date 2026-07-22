// 本文件测试增援系统的核心生命周期和资产边界。
package game

import (
	"errors"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"
)

// TestValidateReinforcementSourceGeneralRejectsSecondGeneral 验证同玩家多批增援不能携带第二名武将。
func TestValidateReinforcementSourceGeneralRejectsSecondGeneral(t *testing.T) {
	records := []Reinforcement{{
		ID:            "rein_existing_general",
		FromPlayerID:  "player_same_source",
		OwnerPlayerID: "player_same_source",
		SourceType:    GarrisonSourceReinforcement,
		Status:        ReinforcementStatusStationed,
		Generals:      []ReinforcementGeneralSnapshot{{ID: "caocao", Name: "曹操"}},
	}}
	if err := validateReinforcementSourceGeneral("player_same_source", []string{"zhangliao"}, records); !errors.Is(err, ErrGeneralBusy) {
		t.Fatalf("expected second reinforcement general rejected, got %v", err)
	}
	if err := validateReinforcementSourceGeneral("player_same_source", nil, records); err != nil {
		t.Fatalf("expected general-free follow-up reinforcement allowed, got %v", err)
	}
}

func TestSendReinforcementConsumesArmyAndReservesGeneral(t *testing.T) {
	svc, repo, from, to := newReinforcementTestService(t)
	from.Army = []ArmyUnit{{UnitType: "weiInfantry", Amount: 100}}
	for index := range from.Generals {
		if from.Generals[index].ID == "caocao" {
			from.Generals[index].Exp = 123
		}
	}
	if from.General != nil && from.General.ID == "caocao" {
		from.General.Exp = 123
	}
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
	if len(result.Reinforcement.Generals) != 1 || result.Reinforcement.Generals[0].Exp != 0 {
		t.Fatalf("expected public send response to hide cumulative general exp, got %+v", result.Reinforcement.Generals)
	}
	raw, err := repo.GetReinforcement(result.Reinforcement.ID)
	if err != nil || len(raw.Generals) != 1 || raw.Generals[0].Exp != 123 {
		t.Fatalf("expected repository to retain internal general exp baseline, record=%+v err=%v", raw, err)
	}
	raw.RewardState = markReinforcementGeneralExpApplied(raw.RewardState, "battle_private_marker")
	repo.reinforcements[raw.ID] = raw
	received, err := svc.ListReceivedReinforcements(to.Player.ID)
	if err != nil || len(received.Items) != 1 || received.Items[0].Generals[0].Exp != 0 {
		t.Fatalf("expected receiver list to hide sender cumulative general exp, response=%+v err=%v", received, err)
	}
	if reinforcementGeneralExpWasApplied(received.Items[0].RewardState, "battle_private_marker") {
		t.Fatalf("expected public reinforcement response to hide internal reward marker, got %+v", received.Items[0].RewardState)
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

// TestSendReinforcementRejectsMultipleGeneralsAtomically 确保武将校验失败不会遗留任何玩家资产或业务记录副作用。
func TestSendReinforcementRejectsMultipleGeneralsAtomically(t *testing.T) {
	svc, repo, from, to := newReinforcementTestService(t)
	setRealCaoCaoGuardConfig(t)
	from.Army = []ArmyUnit{{UnitType: "huWei", Amount: 100}}
	from.Generals = append(from.Generals, *newGeneral("wei", "xiahoudun"))
	from.ResourceSettledAt = time.Now().UTC().Add(-3 * time.Second).Format(resourceDateLayout)
	from.GeneralTraitProgress = map[string]float64{guardProductionProgressKey("caocao", "weiwu_haoling", "huWei"): 0.5}
	repo.players[from.Player.ID] = from
	before, err := repo.GetState(from.Player.ID)
	if err != nil {
		t.Fatalf("GetState before failed: %v", err)
	}

	if _, err = svc.SendReinforcement(SendReinforcementRequest{
		FromPlayerID:   from.Player.ID,
		TargetPlayerID: to.Player.ID,
		Troops:         map[string]int{"huWei": 10},
		GeneralIDs:     []string{"caocao", "xiahoudun"},
	}); !errors.Is(err, ErrInvalidGeneral) {
		t.Fatalf("expected multiple generals to be rejected, got %v", err)
	}

	stored, err := repo.GetState(from.Player.ID)
	if err != nil {
		t.Fatalf("GetState after failed: %v", err)
	}
	if !reflect.DeepEqual(stored, before) {
		t.Fatalf("failed reinforcement request must leave player state unchanged\nbefore=%+v\nafter=%+v", before, stored)
	}
	if records, listErr := repo.ListSentReinforcements(from.Player.ID); listErr != nil || len(records) != 0 {
		t.Fatalf("failed reinforcement request must create no record, records=%+v err=%v", records, listErr)
	}
	if reports, total, listErr := repo.ListReports(from.Player.ID, 10, 0); listErr != nil || total != 0 || len(reports) != 0 {
		t.Fatalf("failed reinforcement request must create no report, reports=%+v total=%d err=%v", reports, total, listErr)
	}
}

func TestSendReinforcementSettlesCaoCaoGuardProductionBeforeConsume(t *testing.T) {
	svc, repo, from, to := newReinforcementTestService(t)
	setRealCaoCaoGuardConfig(t)
	settledAt := time.Now().UTC().Add(-24 * time.Hour)
	from.Army = []ArmyUnit{}
	from.ResourceSettledAt = settledAt.Format(resourceDateLayout)
	from.ServerTime = settledAt.Format(resourceDateLayout)
	repo.players[from.Player.ID] = from

	result, err := svc.SendReinforcement(SendReinforcementRequest{
		FromPlayerID:   from.Player.ID,
		TargetPlayerID: to.Player.ID,
		Troops:         map[string]int{"huWei": 1},
	})
	if err != nil {
		t.Fatalf("SendReinforcement should use settled Cao Cao guard production: %v", err)
	}
	if got := result.Reinforcement.Troops["huWei"]; got != 1 {
		t.Fatalf("expected sent huWei 1, got %d", got)
	}
	stored, err := repo.GetState(from.Player.ID)
	if err != nil {
		t.Fatalf("GetState failed: %v", err)
	}
	if got := armySliceToMap(stored.Army)["huWei"]; got != 431999 {
		t.Fatalf("expected remaining settled huWei 431999, got %d army=%+v", got, stored.Army)
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
	if result.Reinforcement.SpeedMultiplier != 1 || result.Reinforcement.MarchSeconds <= 0 {
		t.Fatalf("client speedMultiplier should be ignored, got speed %.2f seconds %d", result.Reinforcement.SpeedMultiplier, result.Reinforcement.MarchSeconds)
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
	sent, err := svc.ListSentReinforcements(from.Player.ID)
	if err != nil {
		t.Fatalf("ListSentReinforcements failed: %v", err)
	}
	if len(sent.Items) != 0 {
		t.Fatalf("expected completed reinforcement hidden from sender, got %+v", sent.Items)
	}
	next, err := repo.GetState(from.Player.ID)
	if err != nil {
		t.Fatalf("GetState failed: %v", err)
	}
	if got := next.Army[0].Amount; got != 60 {
		t.Fatalf("expected no dead troops returned, got army %d", got)
	}
}

func TestRecallReinforcementOnRoadUsesElapsedReturnSeconds(t *testing.T) {
	svc, repo, from, to := newReinforcementTestService(t)
	from.Army = []ArmyUnit{{UnitType: "weiInfantry", Amount: 100}}
	repo.players[from.Player.ID] = from
	result, err := svc.SendReinforcement(SendReinforcementRequest{
		FromPlayerID:   from.Player.ID,
		TargetPlayerID: to.Player.ID,
		Troops:         map[string]int{"weiInfantry": 40},
	})
	if err != nil {
		t.Fatalf("SendReinforcement failed: %v", err)
	}
	mutateReinforcementForTest(t, repo, result.Reinforcement.ID, func(record *Reinforcement) {
		now := time.Now().UTC()
		record.SentAt = now.Add(-10 * time.Minute).Format(resourceDateLayout)
		record.MarchSeconds = 3600
		record.ReturnSeconds = 3600
		record.ExpectedArriveAt = now.Add(50 * time.Minute).Format(resourceDateLayout)
	})
	recalled, err := svc.RecallReinforcement(from.Player.ID, result.Reinforcement.ID)
	if err != nil {
		t.Fatalf("RecallReinforcement failed: %v", err)
	}
	if recalled.Reinforcement.Status != ReinforcementStatusReturning {
		t.Fatalf("expected returning, got %s", recalled.Reinforcement.Status)
	}
	if recalled.Reinforcement.ReturnSeconds < 600 || recalled.Reinforcement.ReturnSeconds > 605 {
		t.Fatalf("expected road return around 600 seconds, got %d", recalled.Reinforcement.ReturnSeconds)
	}
}

func TestExpelReinforcementOnRoadUsesElapsedReturnSeconds(t *testing.T) {
	svc, repo, from, to := newReinforcementTestService(t)
	from.Army = []ArmyUnit{{UnitType: "weiInfantry", Amount: 100}}
	repo.players[from.Player.ID] = from
	result, err := svc.SendReinforcement(SendReinforcementRequest{
		FromPlayerID:   from.Player.ID,
		TargetPlayerID: to.Player.ID,
		Troops:         map[string]int{"weiInfantry": 40},
	})
	if err != nil {
		t.Fatalf("SendReinforcement failed: %v", err)
	}
	mutateReinforcementForTest(t, repo, result.Reinforcement.ID, func(record *Reinforcement) {
		now := time.Now().UTC()
		record.SentAt = now.Add(-7 * time.Minute).Format(resourceDateLayout)
		record.MarchSeconds = 3600
		record.ReturnSeconds = 3600
		record.ExpectedArriveAt = now.Add(53 * time.Minute).Format(resourceDateLayout)
	})
	expelled, err := svc.ExpelReinforcement(to.Player.ID, result.Reinforcement.ID)
	if err != nil {
		t.Fatalf("ExpelReinforcement failed: %v", err)
	}
	if expelled.Reinforcement.ReturnSeconds < 420 || expelled.Reinforcement.ReturnSeconds > 425 {
		t.Fatalf("expected road expel return around 420 seconds, got %d", expelled.Reinforcement.ReturnSeconds)
	}
	received, err := svc.ListReceivedReinforcements(to.Player.ID)
	if err != nil {
		t.Fatalf("ListReceivedReinforcements failed: %v", err)
	}
	if len(received.Items) != 0 {
		t.Fatalf("expected expelled reinforcement hidden from receiver, got %+v", received.Items)
	}
	sent, err := svc.ListSentReinforcements(from.Player.ID)
	if err != nil {
		t.Fatalf("ListSentReinforcements failed: %v", err)
	}
	if len(sent.Items) != 1 || sent.Items[0].Status != ReinforcementStatusReturning {
		t.Fatalf("expected sender to see returning reinforcement, got %+v", sent.Items)
	}
}

func TestStationedReinforcementReturnUsesActualOutboundSeconds(t *testing.T) {
	svc, repo, from, to := newReinforcementTestService(t)
	from.Army = []ArmyUnit{{UnitType: "weiInfantry", Amount: 100}}
	repo.players[from.Player.ID] = from
	result, err := svc.SendReinforcement(SendReinforcementRequest{
		FromPlayerID:   from.Player.ID,
		TargetPlayerID: to.Player.ID,
		Troops:         map[string]int{"weiInfantry": 40},
	})
	if err != nil {
		t.Fatalf("SendReinforcement failed: %v", err)
	}
	mutateReinforcementForTest(t, repo, result.Reinforcement.ID, func(record *Reinforcement) {
		now := time.Now().UTC()
		record.Status = ReinforcementStatusStationed
		record.SentAt = now.Add(-60 * time.Minute).Format(resourceDateLayout)
		record.ArrivedAt = now.Add(-20 * time.Minute).Format(resourceDateLayout)
		record.MarchSeconds = 3600
		record.ReturnSeconds = 3600
	})
	recalled, err := svc.RecallReinforcement(from.Player.ID, result.Reinforcement.ID)
	if err != nil {
		t.Fatalf("RecallReinforcement failed: %v", err)
	}
	if recalled.Reinforcement.ReturnSeconds != 2400 {
		t.Fatalf("expected stationed return 2400 seconds, got %d", recalled.Reinforcement.ReturnSeconds)
	}
}

func TestAccelerateReinforcementDeductsCityGoldAndLimitsTimes(t *testing.T) {
	svc, repo, from, to := newReinforcementTestService(t)
	from.CityGold = 100
	from.Army = []ArmyUnit{{UnitType: "weiInfantry", Amount: 100}}
	repo.players[from.Player.ID] = from
	result, err := svc.SendReinforcement(SendReinforcementRequest{
		FromPlayerID:   from.Player.ID,
		TargetPlayerID: to.Player.ID,
		Troops:         map[string]int{"weiInfantry": 40},
	})
	if err != nil {
		t.Fatalf("SendReinforcement failed: %v", err)
	}
	mutateReinforcementForTest(t, repo, result.Reinforcement.ID, func(record *Reinforcement) {
		now := time.Now().UTC()
		record.SentAt = now.Format(resourceDateLayout)
		record.MarchSeconds = 3600
		record.ReturnSeconds = 3600
		record.ExpectedArriveAt = now.Add(time.Hour).Format(resourceDateLayout)
	})
	first, err := svc.AccelerateReinforcement(from.Player.ID, result.Reinforcement.ID)
	if err != nil {
		t.Fatalf("AccelerateReinforcement first failed: %v", err)
	}
	if first.Cost != pvpAccelerateFixedCityGoldCost || first.CityGold != 90 {
		t.Fatalf("unexpected first accelerate cost/cityGold: cost=%d cityGold=%d", first.Cost, first.CityGold)
	}
	if first.Reinforcement.MarchSeconds >= 3600 {
		t.Fatalf("expected march seconds shortened, got %d", first.Reinforcement.MarchSeconds)
	}
	if got := reinforcementAcceleratedTimes(first.Reinforcement.Metadata); got != 1 {
		t.Fatalf("expected acceleratedTimes 1, got %d", got)
	}
	second, err := svc.AccelerateReinforcement(from.Player.ID, result.Reinforcement.ID)
	if err != nil {
		t.Fatalf("AccelerateReinforcement second failed: %v", err)
	}
	if second.CityGold != 80 {
		t.Fatalf("expected second cityGold 80, got %d", second.CityGold)
	}
	if _, err := svc.AccelerateReinforcement(from.Player.ID, result.Reinforcement.ID); !errors.Is(err, ErrReinforcementNotAccelerable) {
		t.Fatalf("expected third accelerate blocked, got %v", err)
	}
	entries, err := svc.ListGoldLedger(GoldLedgerFilter{PlayerID: from.Player.ID, RefType: LedgerRefReinforcementAccelerate})
	if err != nil {
		t.Fatalf("ListGoldLedger failed: %v", err)
	}
	if len(entries) != 2 || entries[0].Amount != pvpAccelerateFixedCityGoldCost {
		t.Fatalf("expected two accelerate ledgers, got %+v", entries)
	}
}

func TestAccelerateReinforcementRequiresSenderAndCityGold(t *testing.T) {
	svc, repo, from, to := newReinforcementTestService(t)
	from.CityGold = 5
	from.Army = []ArmyUnit{{UnitType: "weiInfantry", Amount: 100}}
	repo.players[from.Player.ID] = from
	result, err := svc.SendReinforcement(SendReinforcementRequest{
		FromPlayerID:   from.Player.ID,
		TargetPlayerID: to.Player.ID,
		Troops:         map[string]int{"weiInfantry": 40},
	})
	if err != nil {
		t.Fatalf("SendReinforcement failed: %v", err)
	}
	if _, err := svc.AccelerateReinforcement(to.Player.ID, result.Reinforcement.ID); !errors.Is(err, ErrReinforcementNotFound) {
		t.Fatalf("expected receiver cannot accelerate, got %v", err)
	}
	if _, err := svc.AccelerateReinforcement(from.Player.ID, result.Reinforcement.ID); !errors.Is(err, ErrInsufficientCityGold) {
		t.Fatalf("expected insufficient city gold, got %v", err)
	}
}

func TestHistoricalSlowClientSpeedReinforcementIsClamped(t *testing.T) {
	svc, repo, from, to := newReinforcementTestService(t)
	from.Army = []ArmyUnit{{UnitType: "weiInfantry", Amount: 100}}
	repo.players[from.Player.ID] = from
	result, err := svc.SendReinforcement(SendReinforcementRequest{
		FromPlayerID:   from.Player.ID,
		TargetPlayerID: to.Player.ID,
		Troops:         map[string]int{"weiInfantry": 40},
	})
	if err != nil {
		t.Fatalf("SendReinforcement failed: %v", err)
	}
	mutateReinforcementForTest(t, repo, result.Reinforcement.ID, func(record *Reinforcement) {
		record.MarchSeconds = 179 * 3600
		record.ReturnSeconds = 179 * 3600
		record.SentAt = time.Now().UTC().Add(-4 * time.Hour).Format(resourceDateLayout)
	})
	if err := svc.SettleReinforcementsForPlayer(from.Player.ID); err != nil {
		t.Fatalf("SettleReinforcementsForPlayer failed: %v", err)
	}
	record, err := svc.GetReinforcement(from.Player.ID, result.Reinforcement.ID)
	if err != nil {
		t.Fatalf("GetReinforcement failed: %v", err)
	}
	if record.Status != ReinforcementStatusStationed {
		t.Fatalf("expected historical slow reinforcement to arrive after clamp, got %s", record.Status)
	}
	if record.MarchSeconds != maxReinforcementMarchSeconds {
		t.Fatalf("expected march seconds clamped to max, got %d", record.MarchSeconds)
	}
}

func TestReinforcementTravelSecondsUsesFiveMinutesPerGrid(t *testing.T) {
	now := time.Now()
	if got := reinforcementTravelSecondsForDistance(1, 1, now, nil); got != 5*60 {
		t.Fatalf("expected one grid with speed 1 to take 5 minutes, got %d", got)
	}
	if got := reinforcementTravelSecondsForDistance(1, 5, now, nil); got != 60 {
		t.Fatalf("expected one grid with speed 5 to take 1 minute, got %d", got)
	}
	if got := reinforcementTravelSecondsForDistance(36, 1, now, nil); got != maxReinforcementMarchSeconds {
		t.Fatalf("expected 36 grids with speed 1 to reach max 3 hours, got %d", got)
	}
	if got := reinforcementTravelSecondsForDistance(100, 5, now, nil); got != 6000 {
		t.Fatalf("expected far speed 5 reinforcement to use real distance under max, got %d", got)
	}
	if got := reinforcementTravelSecondsForDistance(2000, 1, now, nil); got != maxReinforcementMarchSeconds {
		t.Fatalf("expected far distance clamped to 3 hours, got %d", got)
	}
}

func TestReinforcementTravelUsesSlowestSelectedUnitSpeed(t *testing.T) {
	setTestCombatUnitsConfig(t)
	unitsMu.Lock()
	weiUnits := activeUnits["wei"]
	infantry := weiUnits["weiInfantry"]
	infantry.Stats["speed"] = 1
	weiUnits["weiInfantry"] = infantry
	cavalry := weiUnits["weiCavalry"]
	cavalry.Stats["speed"] = 5
	weiUnits["weiCavalry"] = cavalry
	activeUnits["wei"] = weiUnits
	unitsMu.Unlock()

	if got := reinforcementSlowestUnitSpeed("wei", map[string]int{"weiCavalry": 10}); got != 5 {
		t.Fatalf("expected cavalry-only speed 5, got %.2f", got)
	}
	if got := reinforcementSlowestUnitSpeed("wei", map[string]int{"weiInfantry": 10, "weiCavalry": 10}); got != 1 {
		t.Fatalf("expected mixed troops to use slowest speed 1, got %.2f", got)
	}
}

func TestSendReinforcementUsesWorldMapDistanceAndUnitSpeed(t *testing.T) {
	svc, repo, from, to := newReinforcementTestService(t)
	unitsMu.Lock()
	weiUnits := activeUnits["wei"]
	infantry := weiUnits["weiInfantry"]
	infantry.Stats["speed"] = 5
	weiUnits["weiInfantry"] = infantry
	activeUnits["wei"] = weiUnits
	unitsMu.Unlock()
	from.Army = []ArmyUnit{{UnitType: "weiInfantry", Amount: 100}}
	for i := range from.Buildings {
		if from.Buildings[i].Type == "relay_station" {
			from.Buildings[i].Level = 0
		}
	}
	repo.players[from.Player.ID] = from
	if _, err := repo.AssignWorldPosition(from.Player.ID, defaultWorldID, 10, 10, "test"); err != nil {
		t.Fatalf("AssignWorldPosition from failed: %v", err)
	}
	if _, err := repo.AssignWorldPosition(to.Player.ID, defaultWorldID, 13, 14, "test"); err != nil {
		t.Fatalf("AssignWorldPosition to failed: %v", err)
	}

	result, err := svc.SendReinforcement(SendReinforcementRequest{
		FromPlayerID:   from.Player.ID,
		TargetPlayerID: to.Player.ID,
		Troops:         map[string]int{"weiInfantry": 20},
	})
	if err != nil {
		t.Fatalf("SendReinforcement failed: %v", err)
	}
	if result.Reinforcement.MarchSeconds != 420 {
		t.Fatalf("expected distance 7 with speed 5 to take 420 seconds, got %+v", result.Reinforcement)
	}
	if result.Reinforcement.ReturnSeconds != 420 {
		t.Fatalf("expected return seconds to use same world map travel seconds, got %+v", result.Reinforcement)
	}
}

func TestReinforcementDistanceTravelReusesWorldMarchFormula(t *testing.T) {
	source, err := os.ReadFile("service_reinforcement.go")
	if err != nil {
		t.Fatalf("read service_reinforcement.go: %v", err)
	}
	body := string(source)
	start := strings.Index(body, "func reinforcementTravelSecondsForDistance")
	if start < 0 {
		t.Fatalf("missing reinforcementTravelSecondsForDistance")
	}
	end := strings.Index(body[start:], "\n}\n\n// reinforcementSlowestUnitSpeed")
	if end < 0 {
		t.Fatalf("could not locate reinforcementTravelSecondsForDistance body")
	}
	fn := body[start : start+end]
	if !strings.Contains(fn, "CalculateWorldMarchSeconds(distance, int(math.Floor(unitSpeed)), now, sources)") {
		t.Fatalf("reinforcement distance travel should reuse world march formula, got:\n%s", fn)
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

func mutateReinforcementForTest(t *testing.T, repo *MemoryRepository, reinforcementID string, mutate func(*Reinforcement)) {
	t.Helper()
	repo.mu.Lock()
	defer repo.mu.Unlock()
	record := repo.reinforcements[reinforcementID]
	mutate(&record)
	repo.reinforcements[reinforcementID] = record
}

func forceReinforcementDue(t *testing.T, repo *MemoryRepository, reinforcementID string, outbound bool) {
	t.Helper()
	mutateReinforcementForTest(t, repo, reinforcementID, func(record *Reinforcement) {
		old := time.Now().Add(-4 * time.Hour).UTC().Format(resourceDateLayout)
		if outbound {
			record.SentAt = old
		} else {
			record.ReturnStartedAt = old
		}
	})
}
