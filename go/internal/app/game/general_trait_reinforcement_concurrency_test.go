// 本文件验证行军加速特性参与同玩家并发增援时只扣一次兵、占用一次武将并创建一条援军。
package game

import (
	"errors"
	"math"
	"sync"
	"testing"

	"hero3/internal/core/general"
)

// TestLvmengMarchTraitsSerializeConcurrentReinforcement 验证资源只够一支援军时，两次并发派遣只能成功一次。
func TestLvmengMarchTraitsSerializeConcurrentReinforcement(t *testing.T) {
	tc := realMarchTraitCase{
		name: "吕蒙双行军特性并发增援", faction: "wu", generalID: "lvmeng", generalName: "吕蒙",
		specialTrait: marchTraitConfig("baiyi_dujiang", general.TraitTypeSpecial, 0.2, 60, 1),
		bonusTrait:   marchTraitConfig("baiyi_jixing", general.TraitTypeBonus, 0.2, 60, 1),
		rates:        []float64{0.2, 0.2}, minimum: 60,
	}
	control := runRealMarchTraitReinforcement(t, tc, false)
	opponentFaction, opponentID, unitType := setRealMarchTraitGenerals(t, tc, true)
	svc, repo, from, to := newPvpTestServiceForGenerals(t, tc.faction, tc.generalID, opponentFaction, opponentID)
	from.Army = []ArmyUnit{{UnitType: unitType, Amount: 100}}
	to.Army = []ArmyUnit{{UnitType: opponentFaction + "Infantry", Amount: 100}}
	repo.players[from.Player.ID] = from
	repo.players[to.Player.ID] = to
	if _, err := repo.AssignWorldPosition(from.Player.ID, defaultWorldID, 10, 10, "test"); err != nil {
		t.Fatalf("AssignWorldPosition from failed: %v", err)
	}
	if _, err := repo.AssignWorldPosition(to.Player.ID, defaultWorldID, 20, 10, "test"); err != nil {
		t.Fatalf("AssignWorldPosition to failed: %v", err)
	}

	start := make(chan struct{})
	type dispatchResult struct {
		response ReinforcementResponse
		err      error
	}
	results := make(chan dispatchResult, 2)
	var workers sync.WaitGroup
	for range 2 {
		workers.Add(1)
		go func() {
			defer workers.Done()
			<-start
			response, err := svc.SendReinforcement(SendReinforcementRequest{
				FromPlayerID: from.Player.ID, TargetPlayerID: to.Player.ID,
				Troops: map[string]int{unitType: 100}, GeneralIDs: []string{tc.generalID},
			})
			results <- dispatchResult{response: response, err: err}
		}()
	}
	close(start)
	workers.Wait()
	close(results)

	succeeded := 0
	insufficient := 0
	var created ReinforcementResponse
	for result := range results {
		switch {
		case result.err == nil:
			succeeded++
			created = result.response
		case errors.Is(result.err, ErrInsufficientArmy):
			insufficient++
		default:
			t.Fatalf("unexpected concurrent reinforcement error: %v", result.err)
		}
	}
	if succeeded != 1 || insufficient != 1 || created.Reinforcement.ID == "" {
		t.Fatalf("expected one success and one insufficient-army result, success=%d insufficient=%d response=%+v", succeeded, insufficient, created)
	}

	wantDuration := applyExpectedMarchRates(control.MarchSeconds, tc.rates, tc.minimum)
	record := created.Reinforcement
	wantSpeed := float64(control.MarchSeconds) / float64(wantDuration)
	if record.MarchSeconds != wantDuration || record.ReturnSeconds != wantDuration || math.Abs(record.SpeedMultiplier-wantSpeed) > 1e-9 {
		t.Fatalf("expected one dual-trait accelerated reinforcement with duration %d and speed %.6f, control=%+v record=%+v", wantDuration, wantSpeed, control, record)
	}
	assertMarchDurationTimestamps(t, record.SentAt, record.ExpectedArriveAt, wantDuration)
	if len(record.Generals) != 1 || record.Generals[0].ID != tc.generalID || record.Generals[0].Assignment != record.ID+"_"+tc.generalID {
		t.Fatalf("expected Lvmeng snapshot assigned to the only reinforcement, record=%+v", record)
	}

	stored, err := repo.GetState(from.Player.ID)
	if err != nil {
		t.Fatalf("GetState sender failed: %v", err)
	}
	if len(stored.Army) != 0 || len(created.Patch.Army) != 0 {
		t.Fatalf("expected 100 infantry consumed exactly once in state and patch, state=%+v patch=%+v", stored.Army, created.Patch.Army)
	}
	reinforcementAssignments := 0
	for _, assignment := range stored.GeneralAssignments {
		if assignment.GeneralID == tc.generalID && assignment.Slot == ReinforcementModuleID {
			reinforcementAssignments++
			if assignment.ID != record.ID+"_"+tc.generalID || assignment.Status != ReinforcementStatusMarching {
				t.Fatalf("expected assignment to point at the only reinforcement, assignment=%+v record=%+v", assignment, record)
			}
		}
	}
	if reinforcementAssignments != 1 || generalAvailableAtHome(stored.GeneralAssignments, tc.generalID) {
		t.Fatalf("expected Lvmeng occupied exactly once, assignments=%+v", stored.GeneralAssignments)
	}

	sent, err := repo.ListSentReinforcements(from.Player.ID)
	if err != nil || len(sent) != 1 || sent[0].ID != record.ID || sent[0].MarchSeconds != wantDuration {
		t.Fatalf("expected exactly one persisted accelerated reinforcement, records=%+v err=%v", sent, err)
	}
	received, err := repo.ListReceivedReinforcements(to.Player.ID)
	if err != nil || len(received) != 1 || received[0].ID != record.ID {
		t.Fatalf("expected target to receive exactly the same reinforcement, records=%+v err=%v", received, err)
	}
	storedTarget, err := repo.GetState(to.Player.ID)
	if err != nil || len(storedTarget.Army) != 1 || storedTarget.Army[0].Amount != 100 {
		t.Fatalf("expected target assets untouched before arrival, state=%+v err=%v", storedTarget, err)
	}
	for _, assignment := range storedTarget.GeneralAssignments {
		if assignment.Slot == ReinforcementModuleID {
			t.Fatalf("expected no reinforcement assignment added to target before arrival, assignments=%+v", storedTarget.GeneralAssignments)
		}
	}
	assertNoPreCombatMarchReport(t, repo, from.Player.ID)
	assertNoPreCombatMarchReport(t, repo, to.Player.ID)
}
