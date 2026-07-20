// 本文件验证行军加速特性参与同玩家并发出征时只扣一次兵、占用一次武将并创建一条行军。
package game

import (
	"errors"
	"math"
	"sync"
	"testing"

	"hero3/internal/core/general"
)

// TestLvmengMarchTraitsSerializeConcurrentDispatch 验证资源只够一支军队时，两次并发出征只能成功一次。
func TestLvmengMarchTraitsSerializeConcurrentDispatch(t *testing.T) {
	tc := realMarchTraitCase{
		name: "吕蒙双行军特性并发出征", faction: "wu", generalID: "lvmeng", generalName: "吕蒙",
		specialTrait: marchTraitConfig("baiyi_dujiang", general.TraitTypeSpecial, 0.2, 60, 1),
		bonusTrait:   marchTraitConfig("baiyi_jixing", general.TraitTypeBonus, 0.2, 60, 1),
		rates:        []float64{0.2, 0.2}, minimum: 60,
	}
	control := runRealMarchTraitPvp(t, tc, false)
	opponentFaction, opponentID, unitType := setRealMarchTraitGenerals(t, tc, true)
	svc, repo, attacker, defender := newPvpTestServiceForGenerals(t, tc.faction, tc.generalID, opponentFaction, opponentID)
	attacker.Army = []ArmyUnit{{UnitType: "wuInfantry", Amount: 100}}
	defender.Army = []ArmyUnit{{UnitType: opponentFaction + "Infantry", Amount: 100}}
	defender.Buildings = nil
	repo.players[attacker.Player.ID] = attacker
	repo.players[defender.Player.ID] = defender
	if _, err := repo.AssignWorldPosition(attacker.Player.ID, defaultWorldID, 10, 10, "test"); err != nil {
		t.Fatalf("AssignWorldPosition attacker failed: %v", err)
	}
	if _, err := repo.AssignWorldPosition(defender.Player.ID, defaultWorldID, 20, 10, "test"); err != nil {
		t.Fatalf("AssignWorldPosition defender failed: %v", err)
	}

	start := make(chan struct{})
	type dispatchResult struct {
		response PvpAttackResponse
		err      error
	}
	results := make(chan dispatchResult, 2)
	var workers sync.WaitGroup
	for range 2 {
		workers.Add(1)
		go func() {
			defer workers.Done()
			<-start
			response, err := svc.StartPvpAttack(PvpAttackRequest{
				PlayerID: attacker.Player.ID, TargetPlayerID: defender.Player.ID, MarchMode: PvpMarchTypeAttack,
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
	var created PvpMarch
	for result := range results {
		switch {
		case result.err == nil:
			succeeded++
			created = result.response.March
		case errors.Is(result.err, ErrInsufficientArmy):
			insufficient++
		default:
			t.Fatalf("unexpected concurrent dispatch error: %v", result.err)
		}
	}
	if succeeded != 1 || insufficient != 1 || created.ID == "" {
		t.Fatalf("expected one success and one insufficient-army result, success=%d insufficient=%d march=%+v", succeeded, insufficient, created)
	}
	wantDuration := applyExpectedMarchRates(control.DurationSeconds, tc.rates, tc.minimum)
	if created.DurationSeconds != wantDuration || math.Abs(created.SpeedMultiplier-float64(defaultPvpMarchSeconds)/float64(wantDuration)) > 1e-9 {
		t.Fatalf("expected one dual-trait accelerated march with duration %d, march=%+v", wantDuration, created)
	}
	assertMarchDurationTimestamps(t, created.StartedAt, created.ArrivesAt, wantDuration)

	stored, err := repo.GetState(attacker.Player.ID)
	if err != nil {
		t.Fatalf("GetState attacker failed: %v", err)
	}
	if len(stored.Army) != 0 {
		t.Fatalf("expected 100 infantry consumed exactly once, army=%+v", stored.Army)
	}
	pvpAssignments := 0
	for _, assignment := range stored.GeneralAssignments {
		if assignment.GeneralID == tc.generalID && assignment.Slot == PVPModuleID {
			pvpAssignments++
			if assignment.ID != created.ID+"_lvmeng" || assignment.Status != PvpMarchStatusMarching {
				t.Fatalf("expected assignment to point at the only march, assignment=%+v march=%+v", assignment, created)
			}
		}
	}
	if pvpAssignments != 1 || generalAvailableAtHome(stored.GeneralAssignments, tc.generalID) {
		t.Fatalf("expected Lvmeng occupied exactly once, assignments=%+v", stored.GeneralAssignments)
	}
	marches, err := repo.ListPvpMarchesForPlayer(attacker.Player.ID)
	if err != nil || len(marches) != 1 || marches[0].ID != created.ID || marches[0].DurationSeconds != wantDuration {
		t.Fatalf("expected exactly one persisted accelerated march, marches=%+v err=%v", marches, err)
	}
	assertNoPreCombatMarchReport(t, repo, attacker.Player.ID)

	attackerPvp, err := svc.GetPvpState(attacker.Player.ID)
	if err != nil || attackerPvp.State.DailyAttackCount != 1 {
		t.Fatalf("expected daily attack count advanced once, pvp=%+v err=%v", attackerPvp, err)
	}
	storedDefender, err := repo.GetState(defender.Player.ID)
	if err != nil || len(storedDefender.Army) != 1 || storedDefender.Army[0].Amount != 100 {
		t.Fatalf("expected target state untouched before arrival, state=%+v err=%v", storedDefender, err)
	}
}
