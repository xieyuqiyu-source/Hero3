// 本文件验证 NPC 手动刷新扣费、原子性和流水记录。
package game

import (
	"errors"
	"testing"
	"time"
)

// createNpcRefreshFixture 创建带指定账户金币的玩家和服务。
func createNpcRefreshFixture(t *testing.T, gold int) (*Service, *MemoryRepository, Account, GameState) {
	t.Helper()
	now := time.Now().UTC()
	repo := NewMemoryRepository()
	account := Account{ID: "account_npc_refresh", Username: "npc_refresh", PasswordHash: "hash", Gold: gold, CreatedAt: now}
	if err := repo.CreateAccount(account); err != nil {
		t.Fatalf("create account: %v", err)
	}
	state := newPlayerState("player_npc_refresh", "刷新测试", "wei", "caocao", now)
	if err := repo.CreatePlayer(account.ID, state, now); err != nil {
		t.Fatalf("create player: %v", err)
	}
	return NewServiceWithRepository(repo), repo, account, state
}

// TestRefreshNpcCitiesDeductsAccountGold 验证成功刷新只扣固定 100 账户金币。
func TestRefreshNpcCitiesDeductsAccountGold(t *testing.T) {
	service, repo, account, state := createNpcRefreshFixture(t, 250)
	result, err := service.RefreshNpcCities(state.Player.ID)
	if err != nil {
		t.Fatalf("refresh npc cities: %v", err)
	}
	if result.AccountGold != 150 || result.Cost != NpcRefreshGoldCost || len(result.Cities) != GetNpcConfig().TotalCities {
		t.Fatalf("unexpected refresh result: %+v", result)
	}
	current, err := repo.GetAccountByID(account.ID)
	if err != nil || current.Gold != 150 {
		t.Fatalf("expected persisted gold 150, account=%+v err=%v", current, err)
	}
	entries, err := service.ListGoldLedger(GoldLedgerFilter{AccountID: account.ID, RefType: LedgerRefNpcRefresh})
	if err != nil || len(entries) != 1 || entries[0].Amount != 100 || entries[0].BalanceAfter != 150 {
		t.Fatalf("unexpected npc refresh ledger: %+v err=%v", entries, err)
	}
}

// TestRefreshNpcCitiesRejectsInsufficientGold 验证余额不足时金币和 NPC 状态均不改变。
func TestRefreshNpcCitiesRejectsInsufficientGold(t *testing.T) {
	service, repo, account, state := createNpcRefreshFixture(t, 99)
	before, err := service.GetNpcCities(state.Player.ID)
	if err != nil {
		t.Fatalf("seed npc cities: %v", err)
	}
	if _, err = service.RefreshNpcCities(state.Player.ID); !errors.Is(err, ErrInsufficientGold) {
		t.Fatalf("expected insufficient gold, got %v", err)
	}
	current, _ := repo.GetAccountByID(account.ID)
	after, err := service.GetNpcCities(state.Player.ID)
	if err != nil {
		t.Fatalf("reload npc cities: %v", err)
	}
	if current.Gold != 99 || after.LastRefreshedAt != before.LastRefreshedAt {
		t.Fatalf("insufficient refresh changed state: gold=%d before=%s after=%s", current.Gold, before.LastRefreshedAt, after.LastRefreshedAt)
	}
}
