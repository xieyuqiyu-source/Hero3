// 本文件验证魏武号令参与不同军事模块并发时只提交合法的权威状态。
package game

import (
	"errors"
	"reflect"
	"sync"
	"testing"
	"time"
)

// transientDispatchConflictRepository 为派兵事务注入一次可重试的 MySQL 冲突。
type transientDispatchConflictRepository struct {
	*MemoryRepository
	mu                    sync.Mutex
	pvpFailures           int
	reinforcementFailures int
	pvpCalls              int
	reinforcementCalls    int
}

// CreatePvpMarchWithState 在故障预算耗尽前返回死锁，随后执行真实内存事务。
func (r *transientDispatchConflictRepository) CreatePvpMarchWithState(attackerPlayerID string, defenderPlayerID string, updatedAt time.Time, update func(attacker *GameState, defender *GameState) (PvpMarch, error)) (GameState, GameState, PvpMarch, error) {
	r.mu.Lock()
	r.pvpCalls++
	if r.pvpFailures > 0 {
		r.pvpFailures--
		r.mu.Unlock()
		return GameState{}, GameState{}, PvpMarch{}, errors.New("Error 1213: Deadlock found when trying to get lock")
	}
	r.mu.Unlock()
	return r.MemoryRepository.CreatePvpMarchWithState(attackerPlayerID, defenderPlayerID, updatedAt, update)
}

// CreateReinforcementWithState 在故障预算耗尽前返回死锁，随后执行真实内存事务。
func (r *transientDispatchConflictRepository) CreateReinforcementWithState(fromPlayerID string, toPlayerID string, updatedAt time.Time, update func(from *GameState, to *GameState, targetRecords []Reinforcement) (Reinforcement, error)) (GameState, GameState, Reinforcement, error) {
	r.mu.Lock()
	r.reinforcementCalls++
	if r.reinforcementFailures > 0 {
		r.reinforcementFailures--
		r.mu.Unlock()
		return GameState{}, GameState{}, Reinforcement{}, errors.New("Error 1213: Deadlock found when trying to get lock")
	}
	r.mu.Unlock()
	return r.MemoryRepository.CreateReinforcementWithState(fromPlayerID, toPlayerID, updatedAt, update)
}

// TestCaoCaoCrossDispatchSerializesProductionArmyAndAssignment 验证跨模块争用同一兵力和武将时只能成功一个动作。
func TestCaoCaoCrossDispatchSerializesProductionArmyAndAssignment(t *testing.T) {
	svc, repo, from, target := newPvpTestService(t)
	setRealCaoCaoGuardConfig(t)
	prepareCaoCaoDispatchSettlementState(t, repo, &from, &target)

	type dispatchResult struct {
		kind          string
		attack        PvpAttackResponse
		reinforcement ReinforcementResponse
		err           error
	}
	start := make(chan struct{})
	results := make(chan dispatchResult, 2)
	var workers sync.WaitGroup
	workers.Add(2)
	go func() {
		defer workers.Done()
		<-start
		response, err := svc.StartPvpAttack(PvpAttackRequest{
			PlayerID: from.Player.ID, TargetPlayerID: target.Player.ID, MarchMode: PvpMarchTypeAttack,
			Troops: map[string]int{"huWei": 100}, GeneralIDs: []string{"caocao"},
		})
		results <- dispatchResult{kind: "attack", attack: response, err: err}
	}()
	go func() {
		defer workers.Done()
		<-start
		response, err := svc.SendReinforcement(SendReinforcementRequest{
			FromPlayerID: from.Player.ID, TargetPlayerID: target.Player.ID,
			Troops: map[string]int{"huWei": 100}, GeneralIDs: []string{"caocao"},
		})
		results <- dispatchResult{kind: "reinforcement", reinforcement: response, err: err}
	}()
	close(start)
	workers.Wait()
	close(results)

	succeeded := 0
	insufficient := 0
	successKind := ""
	var responseArmy []ArmyUnit
	var responseResources ResourceState
	var responseProduction ResourceProduction
	var responseSettledAt string
	var responseProgress map[string]float64
	var responseAssignments []GeneralAssignment
	for result := range results {
		switch {
		case result.err == nil:
			succeeded++
			successKind = result.kind
			if result.kind == "attack" {
				responseArmy = result.attack.Army
				responseResources = result.attack.Resources
				responseProduction = result.attack.ResourceProduction
				responseSettledAt = result.attack.ResourceSettledAt
				responseProgress = result.attack.GeneralTraitProgress
				responseAssignments = result.attack.GeneralAssignments
			} else {
				patch := result.reinforcement.Patch
				if patch.Resources == nil || patch.ResourceProduction == nil || patch.GeneralTraitProgress == nil {
					t.Fatalf("successful reinforcement must return complete authority patch: %+v", patch)
				}
				responseArmy = patch.Army
				responseResources = *patch.Resources
				responseProduction = *patch.ResourceProduction
				responseSettledAt = patch.ResourceSettledAt
				responseProgress = *patch.GeneralTraitProgress
				responseAssignments = patch.GeneralAssignments
			}
		case errors.Is(result.err, ErrInsufficientArmy):
			insufficient++
		default:
			t.Fatalf("unexpected %s dispatch result: %v", result.kind, result.err)
		}
	}
	if succeeded != 1 || insufficient != 1 {
		t.Fatalf("expected one success and one insufficient-army result, success=%d insufficient=%d", succeeded, insufficient)
	}

	stored, err := repo.GetState(from.Player.ID)
	if err != nil {
		t.Fatalf("GetState sender failed: %v", err)
	}
	if !reflect.DeepEqual(stored.Army, responseArmy) || !reflect.DeepEqual(stored.Resources, responseResources) || !reflect.DeepEqual(stored.ResourceProduction, responseProduction) || stored.ResourceSettledAt != responseSettledAt || !reflect.DeepEqual(stored.GeneralTraitProgress, responseProgress) || !reflect.DeepEqual(stored.GeneralAssignments, responseAssignments) {
		t.Fatalf("stored state must equal the only successful response\nkind=%s\nstored=%+v\narmy=%+v resources=%+v production=%+v settledAt=%s progress=%+v assignments=%+v", successKind, stored, responseArmy, responseResources, responseProduction, responseSettledAt, responseProgress, responseAssignments)
	}
	remaining := armySliceToMap(stored.Army)["huWei"]
	if remaining <= 0 || remaining >= 100 {
		t.Fatalf("expected one pending production settlement and one 100-unit dispatch, remaining=%d army=%+v", remaining, stored.Army)
	}
	caocaoAssignments := 0
	for _, assignment := range stored.GeneralAssignments {
		if assignment.GeneralID != "caocao" || assignment.Slot == "main" {
			continue
		}
		caocaoAssignments++
		wantSlot := PVPModuleID
		if successKind == "reinforcement" {
			wantSlot = ReinforcementModuleID
		}
		if assignment.Slot != wantSlot {
			t.Fatalf("expected Cao Cao assigned only to successful module %s, assignment=%+v", wantSlot, assignment)
		}
	}
	if caocaoAssignments != 1 || generalAvailableAtHome(stored.GeneralAssignments, "caocao") {
		t.Fatalf("expected exactly one occupied Cao Cao assignment, assignments=%+v", stored.GeneralAssignments)
	}

	marches, err := repo.ListPvpMarchesForPlayer(from.Player.ID)
	if err != nil {
		t.Fatalf("ListPvpMarchesForPlayer failed: %v", err)
	}
	reinforcements, err := repo.ListSentReinforcements(from.Player.ID)
	if err != nil {
		t.Fatalf("ListSentReinforcements failed: %v", err)
	}
	if len(marches)+len(reinforcements) != 1 || len(marches) != map[bool]int{true: 1, false: 0}[successKind == "attack"] || len(reinforcements) != map[bool]int{true: 1, false: 0}[successKind == "reinforcement"] {
		t.Fatalf("expected only successful module record, kind=%s marches=%+v reinforcements=%+v", successKind, marches, reinforcements)
	}
	pvpState, err := svc.GetPvpState(from.Player.ID)
	if err != nil || pvpState.State.DailyAttackCount != len(marches) {
		t.Fatalf("expected daily attack count to follow only persisted PVP march, pvp=%+v marches=%d err=%v", pvpState, len(marches), err)
	}
	assertNoPreCombatMarchReport(t, repo, from.Player.ID)
	assertNoPreCombatMarchReport(t, repo, target.Player.ID)
}

// TestCaoCaoPvpAndNpcAttackSerializeProductionArmyAndReports 验证创建行军与即时 NPC 战斗争用时不会双扣兵或留下失败战报。
func TestCaoCaoPvpAndNpcAttackSerializeProductionArmyAndReports(t *testing.T) {
	svc, repo, from := newNpcAtomicTraitTestService(t, "cross_pvp_npc")
	now := time.Now().UTC()
	var err error
	from, err = repo.UpdateNpcState(from.Player.ID, now, func(state *GameState) error {
		state.NpcState.Cities[0].Army = []ArmyUnit{{UnitType: "azureDragon", Amount: 1000}}
		state.NpcState.Cities[0].MaxArmy = []ArmyUnit{{UnitType: "azureDragon", Amount: 1000}}
		return nil
	})
	if err != nil {
		t.Fatalf("UpdateNpcState defender failed: %v", err)
	}
	beforeCaoCaoExp := pvpTestGeneralExp(from, "caocao")
	targetAccount := Account{ID: "account_cross_pvp_target", Username: "cross_pvp_target", CreatedAt: now}
	if err := repo.CreateAccount(targetAccount); err != nil {
		t.Fatalf("CreateAccount target failed: %v", err)
	}
	target := newPlayerState("player_cross_pvp_target", "跨模块目标", "shu", "liubei", now)
	target.Army = []ArmyUnit{{UnitType: "shuInfantry", Amount: 100}}
	if err := repo.CreatePlayer(targetAccount.ID, target, now); err != nil {
		t.Fatalf("CreatePlayer target failed: %v", err)
	}
	if _, err := repo.AssignWorldPosition(from.Player.ID, defaultWorldID, 10, 10, "test"); err != nil {
		t.Fatalf("AssignWorldPosition sender failed: %v", err)
	}
	if _, err := repo.AssignWorldPosition(target.Player.ID, defaultWorldID, 20, 10, "test"); err != nil {
		t.Fatalf("AssignWorldPosition target failed: %v", err)
	}

	type attackResult struct {
		kind string
		pvp  PvpAttackResponse
		npc  AttackNpcResponse
		err  error
	}
	start := make(chan struct{})
	results := make(chan attackResult, 2)
	var workers sync.WaitGroup
	workers.Add(2)
	go func() {
		defer workers.Done()
		<-start
		response, err := svc.StartPvpAttack(PvpAttackRequest{
			PlayerID: from.Player.ID, TargetPlayerID: target.Player.ID, MarchMode: PvpMarchTypeAttack,
			Troops: map[string]int{"huWei": 100}, GeneralIDs: []string{"caocao"},
		})
		results <- attackResult{kind: "pvp", pvp: response, err: err}
	}()
	go func() {
		defer workers.Done()
		<-start
		response, err := svc.AttackNpc(AttackNpcRequest{
			PlayerID: from.Player.ID, NpcID: from.NpcState.Cities[0].ID, Mode: "attack",
			Units: map[string]int{"huWei": 100}, GeneralIDs: []string{"caocao"},
		})
		results <- attackResult{kind: "npc", npc: response, err: err}
	}()
	close(start)
	workers.Wait()
	close(results)

	succeeded := 0
	conflicted := 0
	successKind := ""
	var successfulPvp PvpAttackResponse
	var successfulNpc AttackNpcResponse
	for result := range results {
		switch {
		case result.err == nil:
			succeeded++
			successKind = result.kind
			successfulPvp = result.pvp
			successfulNpc = result.npc
		case errors.Is(result.err, ErrInsufficientArmy), errors.Is(result.err, ErrGeneralBusy):
			conflicted++
		default:
			t.Fatalf("unexpected %s attack result: %v", result.kind, result.err)
		}
	}
	if succeeded != 1 || conflicted != 1 {
		t.Fatalf("expected one success and one army-or-general conflict, success=%d conflict=%d kind=%s", succeeded, conflicted, successKind)
	}

	stored, err := repo.GetState(from.Player.ID)
	if err != nil {
		t.Fatalf("GetState sender failed: %v", err)
	}
	if successKind == "pvp" {
		if !reflect.DeepEqual(stored.Army, successfulPvp.Army) || !reflect.DeepEqual(stored.Resources, successfulPvp.Resources) || stored.ResourceSettledAt != successfulPvp.ResourceSettledAt || !reflect.DeepEqual(stored.GeneralTraitProgress, successfulPvp.GeneralTraitProgress) || !reflect.DeepEqual(stored.GeneralAssignments, successfulPvp.GeneralAssignments) {
			t.Fatalf("PVP success response must remain authoritative after failed NPC request, stored=%+v response=%+v", stored, successfulPvp)
		}
		storedExp := pvpTestGeneralExp(stored, "caocao")
		responseExp := pvpTestGeneralExp(GameState{Generals: successfulPvp.Generals}, "caocao")
		if storedExp != beforeCaoCaoExp || responseExp != storedExp {
			t.Fatalf("PVP march creation must not grant Cao Cao battle exp, before=%d stored=%d response=%d", beforeCaoCaoExp, storedExp, responseExp)
		}
	} else {
		if !reflect.DeepEqual(stored.Army, successfulNpc.Army) || !reflect.DeepEqual(stored.Resources, successfulNpc.Resources) || stored.ResourceSettledAt != successfulNpc.ResourceSettledAt || !reflect.DeepEqual(stored.GeneralTraitProgress, successfulNpc.GeneralTraitProgress) || !reflect.DeepEqual(stored.NpcState, successfulNpc.NpcState) {
			t.Fatalf("NPC success response must remain authoritative after failed PVP request, stored=%+v response=%+v", stored, successfulNpc)
		}
		storedExp := pvpTestGeneralExp(stored, "caocao")
		responseExp := pvpTestGeneralExp(GameState{Generals: successfulNpc.Generals}, "caocao")
		if successfulNpc.BattleReport.GeneralExpGained <= 0 || storedExp != beforeCaoCaoExp+successfulNpc.BattleReport.GeneralExpGained || responseExp != storedExp {
			t.Fatalf("NPC success must grant exactly the reported Cao Cao exp, before=%d report=%d stored=%d response=%d", beforeCaoCaoExp, successfulNpc.BattleReport.GeneralExpGained, storedExp, responseExp)
		}
	}

	marches, err := repo.ListPvpMarchesForPlayer(from.Player.ID)
	if err != nil {
		t.Fatalf("ListPvpMarchesForPlayer failed: %v", err)
	}
	reports, total, err := repo.ListReports(from.Player.ID, 10, 0)
	if err != nil {
		t.Fatalf("ListReports failed: %v", err)
	}
	wantPvp := map[bool]int{true: 1, false: 0}[successKind == "pvp"]
	wantReports := map[bool]int{true: 1, false: 0}[successKind == "npc"]
	if len(marches) != wantPvp || len(reports) != wantReports || total != wantReports {
		t.Fatalf("expected side effects only from successful attack, kind=%s marches=%+v reports=%+v total=%d", successKind, marches, reports, total)
	}
	pvpState, err := svc.GetPvpState(from.Player.ID)
	if err != nil || pvpState.State.DailyAttackCount != wantPvp {
		t.Fatalf("expected daily attack count to follow persisted PVP march only, pvp=%+v want=%d err=%v", pvpState, wantPvp, err)
	}
	if successKind == "pvp" && generalAvailableAtHome(stored.GeneralAssignments, "caocao") {
		t.Fatalf("expected Cao Cao occupied by successful PVP march, assignments=%+v", stored.GeneralAssignments)
	}
	if successKind == "npc" && !generalAvailableAtHome(stored.GeneralAssignments, "caocao") {
		t.Fatalf("expected instant NPC battle to leave Cao Cao at home, assignments=%+v", stored.GeneralAssignments)
	}
}

// TestCaoCaoReinforcementAndNpcAttackSerializeProductionArmyAndReports 验证增援与即时 NPC 战斗争用时只提交成功动作的状态和记录。
func TestCaoCaoReinforcementAndNpcAttackSerializeProductionArmyAndReports(t *testing.T) {
	svc, repo, from := newNpcAtomicTraitTestService(t, "cross_reinforcement_npc")
	now := time.Now().UTC()
	var err error
	from, err = repo.UpdateNpcState(from.Player.ID, now, func(state *GameState) error {
		state.NpcState.Cities[0].Army = []ArmyUnit{{UnitType: "azureDragon", Amount: 1000}}
		state.NpcState.Cities[0].MaxArmy = []ArmyUnit{{UnitType: "azureDragon", Amount: 1000}}
		return nil
	})
	if err != nil {
		t.Fatalf("UpdateNpcState defender failed: %v", err)
	}
	beforeCaoCaoExp := pvpTestGeneralExp(from, "caocao")
	targetAccount := Account{ID: "account_cross_reinforcement_target", Username: "cross_reinforcement_target", CreatedAt: now}
	if err := repo.CreateAccount(targetAccount); err != nil {
		t.Fatalf("CreateAccount target failed: %v", err)
	}
	target := newPlayerState("player_cross_reinforcement_target", "跨模块增援目标", "shu", "liubei", now)
	if err := repo.CreatePlayer(targetAccount.ID, target, now); err != nil {
		t.Fatalf("CreatePlayer target failed: %v", err)
	}
	if _, err := repo.AssignWorldPosition(from.Player.ID, defaultWorldID, 10, 10, "test"); err != nil {
		t.Fatalf("AssignWorldPosition sender failed: %v", err)
	}
	if _, err := repo.AssignWorldPosition(target.Player.ID, defaultWorldID, 20, 10, "test"); err != nil {
		t.Fatalf("AssignWorldPosition target failed: %v", err)
	}
	targetBefore, err := repo.GetState(target.Player.ID)
	if err != nil {
		t.Fatalf("GetState target before failed: %v", err)
	}

	type dispatchResult struct {
		kind          string
		reinforcement ReinforcementResponse
		npc           AttackNpcResponse
		err           error
	}
	start := make(chan struct{})
	results := make(chan dispatchResult, 2)
	var workers sync.WaitGroup
	workers.Add(2)
	go func() {
		defer workers.Done()
		<-start
		response, err := svc.SendReinforcement(SendReinforcementRequest{
			FromPlayerID: from.Player.ID, TargetPlayerID: target.Player.ID,
			Troops: map[string]int{"huWei": 100}, GeneralIDs: []string{"caocao"},
		})
		results <- dispatchResult{kind: "reinforcement", reinforcement: response, err: err}
	}()
	go func() {
		defer workers.Done()
		<-start
		response, err := svc.AttackNpc(AttackNpcRequest{
			PlayerID: from.Player.ID, NpcID: from.NpcState.Cities[0].ID, Mode: "attack",
			Units: map[string]int{"huWei": 100}, GeneralIDs: []string{"caocao"},
		})
		results <- dispatchResult{kind: "npc", npc: response, err: err}
	}()
	close(start)
	workers.Wait()
	close(results)

	succeeded := 0
	conflicted := 0
	successKind := ""
	var successfulReinforcement ReinforcementResponse
	var successfulNpc AttackNpcResponse
	for result := range results {
		switch {
		case result.err == nil:
			succeeded++
			successKind = result.kind
			successfulReinforcement = result.reinforcement
			successfulNpc = result.npc
		case errors.Is(result.err, ErrInsufficientArmy), errors.Is(result.err, ErrGeneralBusy):
			conflicted++
		default:
			t.Fatalf("unexpected %s dispatch result: %v", result.kind, result.err)
		}
	}
	if succeeded != 1 || conflicted != 1 {
		t.Fatalf("expected one success and one army-or-general conflict, success=%d conflict=%d kind=%s", succeeded, conflicted, successKind)
	}

	stored, err := repo.GetState(from.Player.ID)
	if err != nil {
		t.Fatalf("GetState sender failed: %v", err)
	}
	if successKind == "reinforcement" {
		patch := successfulReinforcement.Patch
		if patch.Resources == nil || patch.ResourceProduction == nil || patch.GeneralTraitProgress == nil {
			t.Fatalf("successful reinforcement must return complete authority patch: %+v", patch)
		}
		if !reflect.DeepEqual(stored.Army, patch.Army) || !reflect.DeepEqual(stored.Resources, *patch.Resources) || !reflect.DeepEqual(stored.ResourceProduction, *patch.ResourceProduction) || stored.ResourceSettledAt != patch.ResourceSettledAt || !reflect.DeepEqual(stored.GeneralTraitProgress, *patch.GeneralTraitProgress) || !reflect.DeepEqual(stored.GeneralAssignments, patch.GeneralAssignments) {
			t.Fatalf("reinforcement success response must remain authoritative after failed NPC request, stored=%+v patch=%+v", stored, patch)
		}
		storedExp := pvpTestGeneralExp(stored, "caocao")
		responseExp := pvpTestGeneralExp(GameState{Generals: patch.Generals}, "caocao")
		if storedExp != beforeCaoCaoExp || responseExp != storedExp {
			t.Fatalf("reinforcement dispatch must not grant Cao Cao battle exp, before=%d stored=%d response=%d", beforeCaoCaoExp, storedExp, responseExp)
		}
	} else {
		if !reflect.DeepEqual(stored.Army, successfulNpc.Army) || !reflect.DeepEqual(stored.Resources, successfulNpc.Resources) || stored.ResourceSettledAt != successfulNpc.ResourceSettledAt || !reflect.DeepEqual(stored.GeneralTraitProgress, successfulNpc.GeneralTraitProgress) || !reflect.DeepEqual(stored.NpcState, successfulNpc.NpcState) {
			t.Fatalf("NPC success response must remain authoritative after failed reinforcement request, stored=%+v response=%+v", stored, successfulNpc)
		}
		storedExp := pvpTestGeneralExp(stored, "caocao")
		responseExp := pvpTestGeneralExp(GameState{Generals: successfulNpc.Generals}, "caocao")
		if successfulNpc.BattleReport.GeneralExpGained <= 0 || storedExp != beforeCaoCaoExp+successfulNpc.BattleReport.GeneralExpGained || responseExp != storedExp {
			t.Fatalf("NPC success must grant exactly the reported Cao Cao exp, before=%d report=%d stored=%d response=%d", beforeCaoCaoExp, successfulNpc.BattleReport.GeneralExpGained, storedExp, responseExp)
		}
	}

	sent, err := repo.ListSentReinforcements(from.Player.ID)
	if err != nil {
		t.Fatalf("ListSentReinforcements failed: %v", err)
	}
	received, err := repo.ListReceivedReinforcements(target.Player.ID)
	if err != nil {
		t.Fatalf("ListReceivedReinforcements failed: %v", err)
	}
	reports, total, err := repo.ListReports(from.Player.ID, 10, 0)
	if err != nil {
		t.Fatalf("ListReports failed: %v", err)
	}
	wantReinforcements := map[bool]int{true: 1, false: 0}[successKind == "reinforcement"]
	wantReports := map[bool]int{true: 1, false: 0}[successKind == "npc"]
	if len(sent) != wantReinforcements || len(received) != wantReinforcements || len(reports) != wantReports || total != wantReports {
		t.Fatalf("expected side effects only from successful action, kind=%s sent=%+v received=%+v reports=%+v total=%d", successKind, sent, received, reports, total)
	}
	if successKind == "reinforcement" {
		if sent[0].ID != successfulReinforcement.Reinforcement.ID || received[0].ID != sent[0].ID || generalAvailableAtHome(stored.GeneralAssignments, "caocao") {
			t.Fatalf("expected one matching reinforcement and occupied Cao Cao, response=%+v sent=%+v received=%+v assignments=%+v", successfulReinforcement.Reinforcement, sent, received, stored.GeneralAssignments)
		}
	} else if !generalAvailableAtHome(stored.GeneralAssignments, "caocao") {
		t.Fatalf("expected instant NPC battle to leave Cao Cao at home, assignments=%+v", stored.GeneralAssignments)
	}
	targetAfter, err := repo.GetState(target.Player.ID)
	if err != nil {
		t.Fatalf("GetState target after failed: %v", err)
	}
	if !reflect.DeepEqual(targetAfter, targetBefore) {
		t.Fatalf("dispatch creation and NPC battle must not mutate reinforcement target state\nbefore=%+v\nafter=%+v", targetBefore, targetAfter)
	}
	marches, err := repo.ListPvpMarchesForPlayer(from.Player.ID)
	if err != nil || len(marches) != 0 {
		t.Fatalf("NPC and reinforcement race must create no PVP march, marches=%+v err=%v", marches, err)
	}
}

// TestDispatchCreationRetriesTransientStorageConflict 验证派兵入口重试死锁后不会重复扣兵或创建记录。
func TestDispatchCreationRetriesTransientStorageConflict(t *testing.T) {
	t.Run("PVP attack", func(t *testing.T) {
		_, repo, from, target := newPvpTestService(t)
		if _, err := repo.UpdatePlayerState(from.Player.ID, time.Now(), func(state *GameState) error {
			state.Army = []ArmyUnit{{UnitType: "weiInfantry", Amount: 100}}
			return nil
		}); err != nil {
			t.Fatalf("prepare PVP army failed: %v", err)
		}
		wrapped := &transientDispatchConflictRepository{MemoryRepository: repo, pvpFailures: 1}
		svc := NewServiceWithRepository(wrapped)
		result, err := svc.StartPvpAttack(PvpAttackRequest{
			PlayerID: from.Player.ID, TargetPlayerID: target.Player.ID, MarchMode: PvpMarchTypeAttack,
			Troops: map[string]int{"weiInfantry": 30},
		})
		if err != nil {
			t.Fatalf("StartPvpAttack after retry failed: %v", err)
		}
		marches, listErr := repo.ListPvpMarchesForPlayer(from.Player.ID)
		if listErr != nil || wrapped.pvpCalls != 2 || len(marches) != 1 || marches[0].ID != result.March.ID || armySliceToMap(result.Army)["weiInfantry"] != 70 {
			t.Fatalf("expected one PVP commit after two attempts, calls=%d result=%+v marches=%+v err=%v", wrapped.pvpCalls, result, marches, listErr)
		}
	})

	t.Run("PVP scout", func(t *testing.T) {
		_, repo, from, target := newPvpTestService(t)
		addPvpScoutTestUnits(t)
		if _, err := repo.UpdatePlayerState(from.Player.ID, time.Now(), func(state *GameState) error {
			state.Army = []ArmyUnit{{UnitType: "weiScout", Amount: 5}}
			return nil
		}); err != nil {
			t.Fatalf("prepare PVP scout army failed: %v", err)
		}
		wrapped := &transientDispatchConflictRepository{MemoryRepository: repo, pvpFailures: 1}
		svc := NewServiceWithRepository(wrapped)
		result, err := svc.ScoutPvpTarget(PvpScoutRequest{PlayerID: from.Player.ID, TargetPlayerID: target.Player.ID})
		if err != nil {
			t.Fatalf("ScoutPvpTarget after retry failed: %v", err)
		}
		marches, listErr := repo.ListPvpMarchesForPlayer(from.Player.ID)
		if listErr != nil || wrapped.pvpCalls != 2 || len(marches) != 1 || marches[0].ID != result.March.ID || armySliceToMap(result.Army)["weiScout"] != 0 {
			t.Fatalf("expected one scout commit after two attempts, calls=%d result=%+v marches=%+v err=%v", wrapped.pvpCalls, result, marches, listErr)
		}
	})

	t.Run("reinforcement", func(t *testing.T) {
		_, repo, from, target := newReinforcementTestService(t)
		if _, err := repo.UpdatePlayerState(from.Player.ID, time.Now(), func(state *GameState) error {
			state.Army = []ArmyUnit{{UnitType: "weiInfantry", Amount: 100}}
			return nil
		}); err != nil {
			t.Fatalf("prepare reinforcement army failed: %v", err)
		}
		wrapped := &transientDispatchConflictRepository{MemoryRepository: repo, reinforcementFailures: 1}
		svc := NewServiceWithRepository(wrapped)
		result, err := svc.SendReinforcement(SendReinforcementRequest{
			FromPlayerID: from.Player.ID, TargetPlayerID: target.Player.ID,
			Troops: map[string]int{"weiInfantry": 30},
		})
		if err != nil {
			t.Fatalf("SendReinforcement after retry failed: %v", err)
		}
		records, listErr := repo.ListSentReinforcements(from.Player.ID)
		if listErr != nil || wrapped.reinforcementCalls != 2 || len(records) != 1 || records[0].ID != result.Reinforcement.ID || armySliceToMap(result.Patch.Army)["weiInfantry"] != 70 {
			t.Fatalf("expected one reinforcement commit after two attempts, calls=%d result=%+v records=%+v err=%v", wrapped.reinforcementCalls, result, records, listErr)
		}
	})

	t.Run("PVP retry exhausted", func(t *testing.T) {
		_, repo, from, target := newPvpTestService(t)
		if _, err := repo.UpdatePlayerState(from.Player.ID, time.Now(), func(state *GameState) error {
			state.Army = []ArmyUnit{{UnitType: "weiInfantry", Amount: 100}}
			return nil
		}); err != nil {
			t.Fatalf("prepare PVP army failed: %v", err)
		}
		wrapped := &transientDispatchConflictRepository{MemoryRepository: repo, pvpFailures: 3}
		svc := NewServiceWithRepository(wrapped)
		_, err := svc.StartPvpAttack(PvpAttackRequest{
			PlayerID: from.Player.ID, TargetPlayerID: target.Player.ID, MarchMode: PvpMarchTypeAttack,
			Troops: map[string]int{"weiInfantry": 30},
		})
		marches, listErr := repo.ListPvpMarchesForPlayer(from.Player.ID)
		stored, stateErr := repo.GetState(from.Player.ID)
		if err == nil || listErr != nil || stateErr != nil || wrapped.pvpCalls != 3 || len(marches) != 0 || armySliceToMap(stored.Army)["weiInfantry"] != 100 {
			t.Fatalf("expected exhausted PVP retries with no commit, calls=%d err=%v marches=%+v listErr=%v state=%+v stateErr=%v", wrapped.pvpCalls, err, marches, listErr, stored, stateErr)
		}
	})

	t.Run("reinforcement retry exhausted", func(t *testing.T) {
		_, repo, from, target := newReinforcementTestService(t)
		if _, err := repo.UpdatePlayerState(from.Player.ID, time.Now(), func(state *GameState) error {
			state.Army = []ArmyUnit{{UnitType: "weiInfantry", Amount: 100}}
			return nil
		}); err != nil {
			t.Fatalf("prepare reinforcement army failed: %v", err)
		}
		wrapped := &transientDispatchConflictRepository{MemoryRepository: repo, reinforcementFailures: 3}
		svc := NewServiceWithRepository(wrapped)
		_, err := svc.SendReinforcement(SendReinforcementRequest{
			FromPlayerID: from.Player.ID, TargetPlayerID: target.Player.ID,
			Troops: map[string]int{"weiInfantry": 30},
		})
		records, listErr := repo.ListSentReinforcements(from.Player.ID)
		stored, stateErr := repo.GetState(from.Player.ID)
		if err == nil || listErr != nil || stateErr != nil || wrapped.reinforcementCalls != 3 || len(records) != 0 || armySliceToMap(stored.Army)["weiInfantry"] != 100 {
			t.Fatalf("expected exhausted reinforcement retries with no commit, calls=%d err=%v records=%+v listErr=%v state=%+v stateErr=%v", wrapped.reinforcementCalls, err, records, listErr, stored, stateErr)
		}
	})
}
