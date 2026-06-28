// 本文件测试轮回绝境副本服务的基础闭环。
package game

import (
	"path/filepath"
	"testing"
	"time"
)

// TestReincarnationStartAndAttack 验证轮回绝境可以开启并结算进攻波。
func TestReincarnationStartAndAttack(t *testing.T) {
	root := filepath.Join("..", "..", "..", "config")
	if err := LoadUnitsConfig(filepath.Join(root, "units")); err != nil {
		t.Fatalf("LoadUnitsConfig failed: %v", err)
	}
	if err := LoadItemsConfig(filepath.Join(root, "items.json")); err != nil {
		t.Fatalf("LoadItemsConfig failed: %v", err)
	}
	if err := LoadDropPoolsConfig(filepath.Join(root, "drop_pools.json")); err != nil {
		t.Fatalf("LoadDropPoolsConfig failed: %v", err)
	}
	if err := LoadReincarnationConfig(filepath.Join(root, "reincarnation.json")); err != nil {
		t.Fatalf("LoadReincarnationConfig failed: %v", err)
	}

	repo := NewMemoryRepository()
	service := NewServiceWithRepository(repo)
	now := time.Date(2026, 6, 29, 12, 0, 0, 0, time.UTC)
	if err := repo.CreateAccount(Account{ID: "account_reincarnation", Username: "reincarnation", CreatedAt: now}); err != nil {
		t.Fatalf("CreateAccount failed: %v", err)
	}
	state := newPlayerState("player_reincarnation", "轮回测试", "wei", "caocao", now)
	state.Army = []ArmyUnit{{UnitType: "qingZhouArmy", Amount: 50000}}
	if err := repo.CreatePlayer("account_reincarnation", state, now); err != nil {
		t.Fatalf("CreatePlayer failed: %v", err)
	}

	started, err := service.StartReincarnationRun(state.Player.ID, 1)
	if err != nil {
		t.Fatalf("StartReincarnationRun failed: %v", err)
	}
	if started.Run.CurrentWave != 1 || len(started.Run.Waves) != ReincarnationWaveCount {
		t.Fatalf("unexpected run state: %+v", started.Run)
	}
	wave := started.Run.Waves[0]
	if wave.WaveType != ReincarnationWaveAttack {
		t.Fatalf("expected first wave attack, got %s", wave.WaveType)
	}

	result, err := service.AttackReincarnationWave(state.Player.ID, wave.ID, map[string]int{"qingZhouArmy": 1000}, "attack-once")
	if err != nil {
		t.Fatalf("AttackReincarnationWave failed: %v", err)
	}
	if result.BattleReport == nil || result.BattleReport.SourceType != ReportSourceDungeon {
		t.Fatalf("expected dungeon battle report, got %+v", result.BattleReport)
	}
	if len(result.Army) == 0 {
		t.Fatalf("expected army patch after battle")
	}
	saved, found, err := repo.GetActiveReincarnationRun(state.Player.ID, time.Now())
	if err != nil || !found {
		t.Fatalf("expected saved active run, found=%v err=%v", found, err)
	}
	if len(saved.Battles) != 1 {
		t.Fatalf("expected one battle record, got %d", len(saved.Battles))
	}

	repeated, err := service.AttackReincarnationWave(state.Player.ID, wave.ID, map[string]int{"qingZhouArmy": 1000}, "attack-once")
	if err != nil {
		t.Fatalf("repeated AttackReincarnationWave failed: %v", err)
	}
	if repeated.BattleReport == nil || repeated.BattleReport.ID != result.BattleReport.ID {
		t.Fatalf("expected repeated action to reuse report %s, got %+v", result.BattleReport.ID, repeated.BattleReport)
	}
	savedAfterRepeat, _, err := repo.GetActiveReincarnationRun(state.Player.ID, time.Now())
	if err != nil {
		t.Fatalf("GetActiveReincarnationRun after repeat failed: %v", err)
	}
	if len(savedAfterRepeat.Battles) != 1 {
		t.Fatalf("expected repeated action to keep one battle record, got %d", len(savedAfterRepeat.Battles))
	}
}

// TestReincarnationDefenseFailureGrantsClearedRewards 验证防守失败会发放此前已通关波次奖励。
func TestReincarnationDefenseFailureGrantsClearedRewards(t *testing.T) {
	root := filepath.Join("..", "..", "..", "config")
	if err := LoadUnitsConfig(filepath.Join(root, "units")); err != nil {
		t.Fatalf("LoadUnitsConfig failed: %v", err)
	}
	if err := LoadItemsConfig(filepath.Join(root, "items.json")); err != nil {
		t.Fatalf("LoadItemsConfig failed: %v", err)
	}
	if err := LoadDropPoolsConfig(filepath.Join(root, "drop_pools.json")); err != nil {
		t.Fatalf("LoadDropPoolsConfig failed: %v", err)
	}
	if err := LoadReincarnationConfig(filepath.Join(root, "reincarnation.json")); err != nil {
		t.Fatalf("LoadReincarnationConfig failed: %v", err)
	}

	repo := NewMemoryRepository()
	service := NewServiceWithRepository(repo)
	now := time.Date(2026, 6, 29, 13, 0, 0, 0, time.UTC)
	if err := repo.CreateAccount(Account{ID: "account_reincarnation_fail", Username: "reincarnation_fail", CreatedAt: now}); err != nil {
		t.Fatalf("CreateAccount failed: %v", err)
	}
	state := newPlayerState("player_reincarnation_fail", "轮回失败测试", "wei", "caocao", now)
	state.Army = []ArmyUnit{{UnitType: "qingZhouArmy", Amount: 500000}}
	if err := repo.CreatePlayer("account_reincarnation_fail", state, now); err != nil {
		t.Fatalf("CreatePlayer failed: %v", err)
	}

	started, err := service.StartReincarnationRun(state.Player.ID, 1)
	if err != nil {
		t.Fatalf("StartReincarnationRun failed: %v", err)
	}
	run := started.Run
	for attempt := 0; attempt < 10 && run.CurrentWave == 1; attempt++ {
		wave := run.Waves[0]
		result, err := service.AttackReincarnationWave(state.Player.ID, wave.ID, map[string]int{"qingZhouArmy": 10000}, "clear-wave-1-"+string(rune('a'+attempt)))
		if err != nil {
			t.Fatalf("AttackReincarnationWave attempt %d failed: %v", attempt, err)
		}
		run = result.Run
	}
	if run.CurrentWave != 2 {
		t.Fatalf("expected wave 1 cleared, current wave=%d remaining=%+v", run.CurrentWave, run.Waves[0].EnemyRemaining)
	}

	defenseWave := run.Waves[1]
	result, err := service.ReadyReincarnationDefense(state.Player.ID, defenseWave.ID, map[string]int{"qingZhouArmy": 1}, "defense-fail-once")
	if err != nil {
		t.Fatalf("ReadyReincarnationDefense failed: %v", err)
	}
	if result.Run.Status != ReincarnationRunRewarded || result.Run.RewardGrantedAt == nil {
		t.Fatalf("expected failed run rewards granted, got status=%s rewardAt=%v", result.Run.Status, result.Run.RewardGrantedAt)
	}
	if result.Inventory["general_exp_small"].Amount < 1 {
		t.Fatalf("expected first wave reward in inventory, got %+v", result.Inventory)
	}
	if len(result.InventorySlots) == 0 {
		t.Fatalf("expected inventory slots returned after reward grant")
	}
}
