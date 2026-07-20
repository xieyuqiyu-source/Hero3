// 本文件验证援军武将经验在失败重试和重复结算时保持原子且只发放一次。
package game

import (
	"errors"
	"testing"
	"time"
)

type failingGeneralExpRepository struct {
	*MemoryRepository
	failures int
}

// ApplyGeneralExpEvent 按测试配置注入经验事务失败。
func (r *failingGeneralExpRepository) ApplyGeneralExpEvent(playerID string, eventKey string, updatedAt time.Time, update func(state *GameState) error) (bool, error) {
	if r.failures > 0 {
		r.failures--
		return false, errors.New("injected general exp failure")
	}
	return r.MemoryRepository.ApplyGeneralExpEvent(playerID, eventKey, updatedAt, update)
}

// TestMemoryApplyGeneralExpEventIsAtomicAndIdempotent 验证失败不占用标记且成功后重复调用不重复写入。
func TestMemoryApplyGeneralExpEventIsAtomicAndIdempotent(t *testing.T) {
	repo := NewMemoryRepository()
	now := time.Now().UTC()
	account := Account{ID: "account_exp_atomic", Username: "exp_atomic", CreatedAt: now}
	if err := repo.CreateAccount(account); err != nil {
		t.Fatalf("CreateAccount failed: %v", err)
	}
	state := newPlayerState("player_exp_atomic", "经验原子性", "wei", "caocao", now)
	EnsureGeneralRoster(&state, now)
	if err := repo.CreatePlayer(account.ID, state, now); err != nil {
		t.Fatalf("CreatePlayer failed: %v", err)
	}
	injected := errors.New("callback failed")
	applied, err := repo.ApplyGeneralExpEvent(state.Player.ID, "battle|reinforcement", now, func(current *GameState) error {
		current.Generals[0].Exp += 50
		return injected
	})
	if applied || !errors.Is(err, injected) {
		t.Fatalf("expected failed callback not to apply, applied=%v err=%v", applied, err)
	}
	stored, err := repo.GetState(state.Player.ID)
	if err != nil || stored.Generals[0].Exp != 0 {
		t.Fatalf("expected failed callback rollback, state=%+v err=%v", stored.Generals, err)
	}
	applied, err = repo.ApplyGeneralExpEvent(state.Player.ID, "battle|reinforcement", now, func(current *GameState) error {
		current.Generals[0].Exp += 50
		return nil
	})
	if err != nil || !applied {
		t.Fatalf("expected retry to apply, applied=%v err=%v", applied, err)
	}
	callbackCalled := false
	applied, err = repo.ApplyGeneralExpEvent(state.Player.ID, "battle|reinforcement", now, func(current *GameState) error {
		callbackCalled = true
		current.Generals[0].Exp += 50
		return nil
	})
	if err != nil || applied || callbackCalled {
		t.Fatalf("expected duplicate event to skip callback, applied=%v called=%v err=%v", applied, callbackCalled, err)
	}
	stored, err = repo.GetState(state.Player.ID)
	if err != nil || stored.Generals[0].Exp != 50 {
		t.Fatalf("expected exactly one exp grant, state=%+v err=%v", stored.Generals, err)
	}
}

// TestResolvedReturningPvpRetriesReinforcementGeneralExp 验证已进入返程的战斗可重试补发且不会重复发放经验。
func TestResolvedReturningPvpRetriesReinforcementGeneralExp(t *testing.T) {
	repo := NewMemoryRepository()
	failingRepo := &failingGeneralExpRepository{MemoryRepository: repo, failures: 1}
	svc := NewServiceWithRepository(failingRepo)
	now := time.Now().UTC()
	for _, fixture := range []struct {
		accountID string
		playerID  string
		username  string
		generalID string
	}{
		{accountID: "account_exp_attacker", playerID: "player_exp_attacker", username: "exp_attacker", generalID: "caocao"},
		{accountID: "account_exp_defender", playerID: "player_exp_defender", username: "exp_defender", generalID: "liubei"},
		{accountID: "account_exp_helper", playerID: "player_exp_helper", username: "exp_helper", generalID: "sunquan"},
	} {
		account := Account{ID: fixture.accountID, Username: fixture.username, CreatedAt: now}
		if err := repo.CreateAccount(account); err != nil {
			t.Fatalf("CreateAccount %s failed: %v", fixture.playerID, err)
		}
		state := newPlayerState(fixture.playerID, fixture.username, "wei", fixture.generalID, now)
		EnsureGeneralRoster(&state, now)
		if err := repo.CreatePlayer(account.ID, state, now); err != nil {
			t.Fatalf("CreatePlayer %s failed: %v", fixture.playerID, err)
		}
	}
	battle := PvpBattle{
		ID: "battle_exp_retry", MarchID: "march_exp_retry", Status: PvpBattleStatusResolved,
		AttackerPlayerID: "player_exp_attacker", DefenderPlayerID: "player_exp_defender",
		Result: map[string]interface{}{"reinforcementGeneralExp": map[string]int{"rein_exp_retry": 100}},
		ReinforcementSnapshot: []DefenseReinforcementUnit{{
			ReinforcementID: "rein_exp_retry", FromPlayerID: "player_exp_helper",
			Generals: []ReinforcementGeneralSnapshot{{ID: "sunquan"}},
		}},
	}
	repo.pvpBattles[battle.ID] = battle
	repo.pvpMarches[battle.MarchID] = PvpMarch{
		ID: battle.MarchID, AttackerPlayerID: battle.AttackerPlayerID, DefenderPlayerID: battle.DefenderPlayerID,
		BattleID: battle.ID, Status: PvpMarchStatusReturning,
	}
	resolved, err := svc.ResolvePvpMarch(battle.MarchID)
	if err == nil || resolved.ID != battle.ID {
		t.Fatalf("expected first exp settlement failure after battle recovery, battle=%+v err=%v", resolved, err)
	}
	if _, err := svc.ResolvePvpMarch(battle.MarchID); err != nil {
		t.Fatalf("expected second call to recover exp settlement: %v", err)
	}
	if _, err := svc.ResolvePvpMarch(battle.MarchID); err != nil {
		t.Fatalf("expected duplicate recovery to be idempotent: %v", err)
	}
	stored, err := repo.GetState("player_exp_helper")
	if err != nil || pvpTestGeneralExp(stored, "sunquan") != 100 {
		t.Fatalf("expected helper exp exactly 100 after retry, generals=%+v err=%v", stored.Generals, err)
	}
}
