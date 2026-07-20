// 本文件验证出征、侦查和增援成功响应与事务内留城特性结算后的权威状态完全一致。
package game

import (
	"reflect"
	"testing"
	"time"
)

// TestPvpAttackResponseIncludesCaoCaoSettlementState 验证出征响应同步魏武号令产兵后的完整时间基线。
func TestPvpAttackResponseIncludesCaoCaoSettlementState(t *testing.T) {
	svc, repo, attacker, defender := newPvpTestService(t)
	setRealCaoCaoGuardConfig(t)
	prepareCaoCaoDispatchSettlementState(t, repo, &attacker, &defender)
	response, err := svc.StartPvpAttack(PvpAttackRequest{
		PlayerID: attacker.Player.ID, TargetPlayerID: defender.Player.ID, MarchMode: PvpMarchTypeAttack,
		Troops: map[string]int{"huWei": 100}, GeneralIDs: []string{"caocao"},
	})
	if err != nil {
		t.Fatalf("StartPvpAttack failed: %v", err)
	}
	stored, err := repo.GetState(attacker.Player.ID)
	if err != nil {
		t.Fatalf("GetState attacker failed: %v", err)
	}
	assertPvpDispatchSettlementResponse(t, response.Resources, response.ResourceProduction, response.ResourceSettledAt, response.GeneralTraitProgress, stored)
}

// TestPvpScoutResponseIncludesCaoCaoSettlementState 验证侦查响应同步产兵、侦察兵扣除和特性小数进度。
func TestPvpScoutResponseIncludesCaoCaoSettlementState(t *testing.T) {
	svc, repo, attacker, defender := newPvpTestService(t)
	setRealCaoCaoGuardConfig(t)
	prepareCaoCaoDispatchSettlementState(t, repo, &attacker, &defender)
	scoutUnit := findScoutUnit(attacker.Player.Faction)
	if scoutUnit == "" {
		t.Fatal("expected Wei scout unit")
	}
	attacker.Army = append(attacker.Army, ArmyUnit{UnitType: scoutUnit, Amount: 5})
	repo.players[attacker.Player.ID] = attacker
	response, err := svc.ScoutPvpTarget(PvpScoutRequest{PlayerID: attacker.Player.ID, TargetPlayerID: defender.Player.ID})
	if err != nil {
		t.Fatalf("ScoutPvpTarget failed: %v", err)
	}
	stored, err := repo.GetState(attacker.Player.ID)
	if err != nil {
		t.Fatalf("GetState scout failed: %v", err)
	}
	if armySliceToMap(response.Army)[scoutUnit] != 0 {
		t.Fatalf("expected all scouts dispatched in response, army=%+v", response.Army)
	}
	assertPvpDispatchSettlementResponse(t, response.Resources, response.ResourceProduction, response.ResourceSettledAt, response.GeneralTraitProgress, stored)
}

// TestReinforcementResponseIncludesCaoCaoSettlementState 验证增援补丁同步留城结算后的资源和特性进度。
func TestReinforcementResponseIncludesCaoCaoSettlementState(t *testing.T) {
	svc, repo, from, to := newPvpTestService(t)
	setRealCaoCaoGuardConfig(t)
	prepareCaoCaoDispatchSettlementState(t, repo, &from, &to)
	response, err := svc.SendReinforcement(SendReinforcementRequest{
		FromPlayerID: from.Player.ID, TargetPlayerID: to.Player.ID,
		Troops: map[string]int{"huWei": 100}, GeneralIDs: []string{"caocao"},
	})
	if err != nil {
		t.Fatalf("SendReinforcement failed: %v", err)
	}
	stored, err := repo.GetState(from.Player.ID)
	if err != nil {
		t.Fatalf("GetState reinforcement sender failed: %v", err)
	}
	patch := response.Patch
	if patch.Resources == nil || patch.ResourceProduction == nil || patch.GeneralTraitProgress == nil {
		t.Fatalf("reinforcement dispatch patch must include complete settlement fields, patch=%+v", patch)
	}
	assertPvpDispatchSettlementResponse(t, *patch.Resources, *patch.ResourceProduction, patch.ResourceSettledAt, *patch.GeneralTraitProgress, stored)
}

// TestNpcAttackResponseIncludesCaoCaoSettlementState 验证 NPC 进攻响应同步魏武号令结算后的完整状态。
func TestNpcAttackResponseIncludesCaoCaoSettlementState(t *testing.T) {
	svc, repo, before := newNpcAtomicTraitTestService(t, "response_attack")
	response, err := svc.AttackNpc(AttackNpcRequest{
		PlayerID: before.Player.ID, NpcID: before.NpcState.Cities[0].ID, Mode: "attack",
		Units: map[string]int{"huWei": 10}, GeneralIDs: []string{"caocao"},
	})
	if err != nil {
		t.Fatalf("AttackNpc failed: %v", err)
	}
	stored, err := repo.GetState(before.Player.ID)
	if err != nil {
		t.Fatalf("GetState NPC attacker failed: %v", err)
	}
	assertPvpDispatchSettlementResponse(t, response.Resources, response.ResourceProduction, response.ResourceSettledAt, response.GeneralTraitProgress, stored)
}

// TestNpcSweepResponseIncludesCaoCaoSettlementState 验证 NPC 扫荡聚合响应保留事务内最终结算基线。
func TestNpcSweepResponseIncludesCaoCaoSettlementState(t *testing.T) {
	svc, repo, before := newNpcAtomicTraitTestService(t, "response_sweep")
	response, err := svc.SweepNpc(SweepNpcRequest{
		PlayerID: before.Player.ID, NpcIDs: []string{before.NpcState.Cities[0].ID}, Mode: "attack",
		GeneralIDs: []string{"caocao"},
	})
	if err != nil {
		t.Fatalf("SweepNpc failed: %v", err)
	}
	stored, err := repo.GetState(before.Player.ID)
	if err != nil {
		t.Fatalf("GetState NPC sweeper failed: %v", err)
	}
	assertPvpDispatchSettlementResponse(t, response.Resources, response.ResourceProduction, response.ResourceSettledAt, response.GeneralTraitProgress, stored)
}

// TestNpcScoutResponseIncludesCaoCaoSettlementState 验证 NPC 侦查响应同时同步产兵、资源时间和特性进度。
func TestNpcScoutResponseIncludesCaoCaoSettlementState(t *testing.T) {
	svc, repo, before := newNpcAtomicTraitTestService(t, "response_scout")
	scoutUnit := findScoutUnit(before.Player.Faction)
	if scoutUnit == "" {
		t.Fatal("expected Wei scout unit")
	}
	before.Army = append(before.Army, ArmyUnit{UnitType: scoutUnit, Amount: 5})
	repo.players[before.Player.ID] = before
	response, err := svc.ScoutNpc(ScoutNpcRequest{PlayerID: before.Player.ID, NpcID: before.NpcState.Cities[0].ID})
	if err != nil {
		t.Fatalf("ScoutNpc failed: %v", err)
	}
	stored, err := repo.GetState(before.Player.ID)
	if err != nil {
		t.Fatalf("GetState NPC scout failed: %v", err)
	}
	assertPvpDispatchSettlementResponse(t, response.Resources, response.ResourceProduction, response.ResourceSettledAt, response.GeneralTraitProgress, stored)
}

// prepareCaoCaoDispatchSettlementState 准备魏武号令已有小数进度且达到结算时间的派兵存档。
func prepareCaoCaoDispatchSettlementState(t *testing.T, repo *MemoryRepository, from *GameState, to *GameState) {
	t.Helper()
	settledAt := time.Now().UTC().Add(-3 * time.Second).Truncate(time.Second)
	from.Army = []ArmyUnit{{UnitType: "huWei", Amount: 100}}
	from.ResourceSettledAt = settledAt.Format(resourceDateLayout)
	from.ServerTime = settledAt.Format(resourceDateLayout)
	from.GeneralTraitProgress = map[string]float64{guardProductionProgressKey("caocao", "weiwu_haoling", "huWei"): 0.5}
	repo.players[from.Player.ID] = *from
	repo.players[to.Player.ID] = *to
	if _, err := repo.AssignWorldPosition(from.Player.ID, defaultWorldID, 10, 10, "test"); err != nil {
		t.Fatalf("AssignWorldPosition sender failed: %v", err)
	}
	if _, err := repo.AssignWorldPosition(to.Player.ID, defaultWorldID, 20, 10, "test"); err != nil {
		t.Fatalf("AssignWorldPosition target failed: %v", err)
	}
}

// assertPvpDispatchSettlementResponse 核对动作响应与持久化资源、产量、时点和特性进度逐项相等。
func assertPvpDispatchSettlementResponse(t *testing.T, resources ResourceState, production ResourceProduction, settledAt string, progress map[string]float64, stored GameState) {
	t.Helper()
	if !reflect.DeepEqual(resources, stored.Resources) || !reflect.DeepEqual(production, stored.ResourceProduction) || settledAt != stored.ResourceSettledAt || !reflect.DeepEqual(progress, stored.GeneralTraitProgress) {
		t.Fatalf("dispatch settlement response must match stored state\nresources=%+v/%+v\nproduction=%+v/%+v\nsettledAt=%s/%s\nprogress=%+v/%+v", resources, stored.Resources, production, stored.ResourceProduction, settledAt, stored.ResourceSettledAt, progress, stored.GeneralTraitProgress)
	}
}
