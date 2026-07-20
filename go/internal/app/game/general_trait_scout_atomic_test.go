// 本文件验证侦查创建失败时不会提交留城产兵特性结算、兵力、行军或战报副作用。
package game

import (
	"errors"
	"reflect"
	"testing"
	"time"
)

// TestPvpScoutWithoutScoutRollsBackCaoCaoGuardSettlement 验证没有侦察兵时魏武号令的待结算产兵也必须随请求回滚。
func TestPvpScoutWithoutScoutRollsBackCaoCaoGuardSettlement(t *testing.T) {
	svc, repo, attacker, defender := newPvpTestService(t)
	setRealCaoCaoGuardConfig(t)
	settledAt := time.Now().UTC().Add(-3 * time.Second).Truncate(time.Second)
	attacker.Army = []ArmyUnit{{UnitType: "huWei", Amount: 100}}
	attacker.ResourceSettledAt = settledAt.Format(resourceDateLayout)
	attacker.ServerTime = settledAt.Format(resourceDateLayout)
	attacker.GeneralTraitProgress = map[string]float64{guardProductionProgressKey("caocao", "weiwu_haoling", "huWei"): 0.5}
	repo.players[attacker.Player.ID] = attacker
	repo.players[defender.Player.ID] = defender
	if _, err := repo.AssignWorldPosition(attacker.Player.ID, defaultWorldID, 10, 10, "test"); err != nil {
		t.Fatalf("AssignWorldPosition attacker failed: %v", err)
	}
	if _, err := repo.AssignWorldPosition(defender.Player.ID, defaultWorldID, 20, 10, "test"); err != nil {
		t.Fatalf("AssignWorldPosition defender failed: %v", err)
	}
	beforeAttacker, err := repo.GetState(attacker.Player.ID)
	if err != nil {
		t.Fatalf("GetState attacker before failed: %v", err)
	}
	beforeDefender, err := repo.GetState(defender.Player.ID)
	if err != nil {
		t.Fatalf("GetState defender before failed: %v", err)
	}

	if _, err = svc.ScoutPvpTarget(PvpScoutRequest{PlayerID: attacker.Player.ID, TargetPlayerID: defender.Player.ID}); !errors.Is(err, ErrInsufficientArmy) {
		t.Fatalf("expected scout failure for missing faction scout, got %v", err)
	}
	afterAttacker, err := repo.GetState(attacker.Player.ID)
	if err != nil {
		t.Fatalf("GetState attacker after failed: %v", err)
	}
	afterDefender, err := repo.GetState(defender.Player.ID)
	if err != nil {
		t.Fatalf("GetState defender after failed: %v", err)
	}
	if !reflect.DeepEqual(afterAttacker, beforeAttacker) {
		t.Fatalf("failed scout must roll back guard production and player state\nbefore=%+v\nafter=%+v", beforeAttacker, afterAttacker)
	}
	if !reflect.DeepEqual(afterDefender, beforeDefender) {
		t.Fatalf("failed scout must leave target state unchanged\nbefore=%+v\nafter=%+v", beforeDefender, afterDefender)
	}
	if marches, listErr := repo.ListPvpMarchesForPlayer(attacker.Player.ID); listErr != nil || len(marches) != 0 {
		t.Fatalf("failed scout must create no march, marches=%+v err=%v", marches, listErr)
	}
	for _, playerID := range []string{attacker.Player.ID, defender.Player.ID} {
		if reports, total, listErr := repo.ListReports(playerID, 10, 0); listErr != nil || total != 0 || len(reports) != 0 {
			t.Fatalf("failed scout must create no report for %s, reports=%+v total=%d err=%v", playerID, reports, total, listErr)
		}
	}
}

// TestNpcScoutWithoutScoutRollsBackCaoCaoGuardSettlement 验证 NPC 侦查失败会回滚留城产兵、NPC 结算和战报副作用。
func TestNpcScoutWithoutScoutRollsBackCaoCaoGuardSettlement(t *testing.T) {
	svc, repo, before := newNpcAtomicTraitTestService(t, "scout")

	if _, err := svc.ScoutNpc(ScoutNpcRequest{
		PlayerID: before.Player.ID,
		NpcID:    before.NpcState.Cities[0].ID,
	}); !errors.Is(err, ErrInsufficientArmy) {
		t.Fatalf("expected NPC scout failure for missing faction scout, got %v", err)
	}

	assertNpcTraitFailureHasNoSideEffects(t, repo, before)
}
