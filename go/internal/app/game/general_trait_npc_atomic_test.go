// 本文件验证 NPC 单攻和扫荡的武将校验失败会原子回滚留城特性结算及全部业务副作用。
package game

import (
	"errors"
	"reflect"
	"testing"
	"time"
)

// newNpcAtomicTraitTestService 创建一个若事务提交就会触发魏武号令产兵的 NPC 测试存档。
func newNpcAtomicTraitTestService(t *testing.T, suffix string) (*Service, *MemoryRepository, GameState) {
	t.Helper()
	setRealCaoCaoGuardConfig(t)
	now := time.Now().UTC()
	repo := NewMemoryRepository()
	svc := NewServiceWithRepository(repo)
	account := Account{ID: "account_npc_atomic_" + suffix, Username: "npc_atomic_" + suffix, CreatedAt: now}
	if err := repo.CreateAccount(account); err != nil {
		t.Fatalf("CreateAccount failed: %v", err)
	}
	state := newPlayerState("player_npc_atomic_"+suffix, "NPC 原子性测试", "wei", "caocao", now)
	state.Army = []ArmyUnit{{UnitType: "huWei", Amount: 100}}
	state.Generals = append(state.Generals, *newGeneral("wei", "xiahoudun"))
	state.ResourceSettledAt = now.Add(-3 * time.Second).Format(resourceDateLayout)
	state.GeneralTraitProgress = map[string]float64{guardProductionProgressKey("caocao", "weiwu_haoling", "huWei"): 0.5}
	state.NpcState = &NpcState{
		Cities: []NpcCity{{
			ID: "npc_atomic_" + suffix, Name: "原子性测试城", Faction: "shu",
			Resources: map[string]int{"wood": 1000}, StorageCapacity: map[string]int{"wood": 1000}, ProductionPerHour: map[string]int{},
			Army: []ArmyUnit{{UnitType: "shuInfantry", Amount: 100}}, MaxArmy: []ArmyUnit{{UnitType: "shuInfantry", Amount: 100}},
			ResourceSettledAt: now.Format(resourceDateLayout), ArmySettledAt: now.Format(resourceDateLayout), GeneratedAt: now.Format(resourceDateLayout),
		}},
		LastRefreshedAt: now.Format(resourceDateLayout),
	}
	if err := repo.CreatePlayer(account.ID, state, now); err != nil {
		t.Fatalf("CreatePlayer failed: %v", err)
	}
	stored, err := repo.GetState(state.Player.ID)
	if err != nil {
		t.Fatalf("GetState before failed: %v", err)
	}
	return svc, repo, stored
}

// assertNpcTraitFailureHasNoSideEffects 核对失败请求没有修改玩家、NPC、特性进度或战报记录。
func assertNpcTraitFailureHasNoSideEffects(t *testing.T, repo *MemoryRepository, before GameState) {
	t.Helper()
	after, err := repo.GetState(before.Player.ID)
	if err != nil {
		t.Fatalf("GetState after failed: %v", err)
	}
	if !reflect.DeepEqual(after, before) {
		t.Fatalf("failed NPC request must leave state unchanged\nbefore=%+v\nafter=%+v", before, after)
	}
	reports, total, err := repo.ListReports(before.Player.ID, 10, 0)
	if err != nil || total != 0 || len(reports) != 0 {
		t.Fatalf("failed NPC request must create no report, reports=%+v total=%d err=%v", reports, total, err)
	}
}

// TestNpcAttackRejectsMultipleGeneralsAtomically 验证单城攻击失败不会提交魏武号令结算或任何战斗状态。
func TestNpcAttackRejectsMultipleGeneralsAtomically(t *testing.T) {
	svc, repo, before := newNpcAtomicTraitTestService(t, "attack")
	if _, err := svc.AttackNpc(AttackNpcRequest{
		PlayerID: before.Player.ID, NpcID: before.NpcState.Cities[0].ID, Mode: "attack",
		Units: map[string]int{"huWei": 10}, GeneralIDs: []string{"caocao", "xiahoudun"},
	}); !errors.Is(err, ErrInvalidGeneral) {
		t.Fatalf("expected multiple generals to be rejected, got %v", err)
	}
	assertNpcTraitFailureHasNoSideEffects(t, repo, before)
}

// TestNpcSweepRejectsMultipleGeneralsAtomically 验证扫荡失败不会提交首轮惰性结算、NPC 变化或聚合战报。
func TestNpcSweepRejectsMultipleGeneralsAtomically(t *testing.T) {
	svc, repo, before := newNpcAtomicTraitTestService(t, "sweep")
	if _, err := svc.SweepNpc(SweepNpcRequest{
		PlayerID: before.Player.ID, NpcIDs: []string{before.NpcState.Cities[0].ID}, Mode: "attack",
		GeneralIDs: []string{"caocao", "xiahoudun"},
	}); !errors.Is(err, ErrInvalidGeneral) {
		t.Fatalf("expected multiple generals to be rejected, got %v", err)
	}
	assertNpcTraitFailureHasNoSideEffects(t, repo, before)
}
