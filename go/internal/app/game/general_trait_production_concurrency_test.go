// 本文件验证曹操留城产兵在同一结算时点并发刷新时只推进一次权威兵力和小数进度。
package game

import (
	"sync"
	"testing"
	"time"
)

// TestWeiwuHaolingConcurrentSettlementIsSerialized 验证并发结算不会重复发兵或留下错误进度。
func TestWeiwuHaolingConcurrentSettlementIsSerialized(t *testing.T) {
	setRealCaoCaoGuardConfig(t)
	base := time.Date(2026, 7, 19, 8, 0, 0, 0, time.UTC)
	now := base.Add(3 * time.Second)
	repo := NewMemoryRepository()
	svc := NewServiceWithRepository(repo)
	state := newPlayerState("player_guard_concurrent", "曹操并发结算", "wei", "caocao", base)
	state.Army = nil
	state.ResourceSettledAt = base.Format(resourceDateLayout)
	repo.players[state.Player.ID] = state

	const workers = 32
	start := make(chan struct{})
	errors := make(chan error, workers)
	var wait sync.WaitGroup
	wait.Add(workers)
	for index := 0; index < workers; index++ {
		go func() {
			defer wait.Done()
			<-start
			_, err := svc.settlePlayerProduction(state.Player.ID, now)
			errors <- err
		}()
	}
	close(start)
	wait.Wait()
	close(errors)
	for err := range errors {
		if err != nil {
			t.Fatalf("concurrent settlement failed: %v", err)
		}
	}

	stored, err := repo.GetState(state.Player.ID)
	if err != nil {
		t.Fatalf("GetState failed: %v", err)
	}
	if got := armySliceToMap(stored.Army)["huWei"]; got != 15 {
		t.Fatalf("expected 3 seconds at 300 per minute to produce exactly 15 guards once, got %d army=%+v", got, stored.Army)
	}
	if stored.ResourceSettledAt != now.Format(resourceDateLayout) {
		t.Fatalf("expected settlement timestamp %s, got %s", now.Format(resourceDateLayout), stored.ResourceSettledAt)
	}
	if len(stored.GeneralTraitProgress) != 0 {
		t.Fatalf("expected exact production to consume fractional progress, got %+v", stored.GeneralTraitProgress)
	}
	reports, total, err := repo.ListReports(state.Player.ID, 10, 0)
	if err != nil || total != 0 || len(reports) != 0 {
		t.Fatalf("expected concurrent production not to create battle reports, total=%d reports=%+v err=%v", total, reports, err)
	}
}
