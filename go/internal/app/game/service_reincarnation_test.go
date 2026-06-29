// 本文件测试轮回绝境副本服务的基础闭环。
package game

import (
	"path/filepath"
	"testing"
	"time"

	"hero3/internal/core/combat"
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

	result, err := service.AttackReincarnationWave(state.Player.ID, wave.ID, map[string]int{"qingZhouArmy": 1000}, nil, "attack-once")
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
	stateAfterNoGeneral, err := repo.GetState(state.Player.ID)
	if err != nil {
		t.Fatalf("GetState after no-general attack failed: %v", err)
	}
	if got := pvpTestGeneralExp(stateAfterNoGeneral, "caocao"); got != 0 {
		t.Fatalf("expected no exp when reincarnation attack carries no general, got %d", got)
	}

	repeated, err := service.AttackReincarnationWave(state.Player.ID, wave.ID, map[string]int{"qingZhouArmy": 1000}, nil, "attack-once")
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

// TestReincarnationSelectedGeneralGainsExp 验证副本携带武将杀敌后发放武将经验。
func TestReincarnationSelectedGeneralGainsExp(t *testing.T) {
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
	now := time.Date(2026, 6, 29, 12, 10, 0, 0, time.UTC)
	if err := repo.CreateAccount(Account{ID: "account_reincarnation_general", Username: "reincarnation_general", CreatedAt: now}); err != nil {
		t.Fatalf("CreateAccount failed: %v", err)
	}
	state := newPlayerState("player_reincarnation_general", "轮回武将测试", "wei", "caocao", now)
	state.Army = []ArmyUnit{{UnitType: "qingZhouArmy", Amount: 50000}}
	if err := repo.CreatePlayer("account_reincarnation_general", state, now); err != nil {
		t.Fatalf("CreatePlayer failed: %v", err)
	}

	started, err := service.StartReincarnationRun(state.Player.ID, 1)
	if err != nil {
		t.Fatalf("StartReincarnationRun failed: %v", err)
	}
	run := started.Run
	firstEnemyUnit := combatUnitIDsForFaction(run.Waves[0].EnemyFaction)[0]
	run.Waves[0].EnemyTroops = map[string]int{firstEnemyUnit: 1}
	run.Waves[0].EnemyRemaining = map[string]int{firstEnemyUnit: 1}
	if err := repo.SaveReincarnationRun(run); err != nil {
		t.Fatalf("SaveReincarnationRun test fixture failed: %v", err)
	}

	result, err := service.AttackReincarnationWave(state.Player.ID, run.Waves[0].ID, map[string]int{"qingZhouArmy": 1000}, []string{"caocao"}, "attack-with-general")
	if err != nil {
		t.Fatalf("AttackReincarnationWave with general failed: %v", err)
	}
	if result.BattleReport == nil || result.BattleReport.GeneralExpGained <= 0 || len(result.BattleReport.PvpAttackerGenerals) != 1 {
		t.Fatalf("expected reincarnation report general exp and snapshot, got %+v", result.BattleReport)
	}
	if result.General == nil || result.General.Exp <= 0 {
		t.Fatalf("expected action result to return updated general, got %+v", result.General)
	}
	stored, err := repo.GetState(state.Player.ID)
	if err != nil {
		t.Fatalf("GetState failed: %v", err)
	}
	if got := pvpTestGeneralExp(stored, "caocao"); got != result.BattleReport.GeneralExpGained {
		t.Fatalf("expected stored general exp %d, got %d", result.BattleReport.GeneralExpGained, got)
	}
}

// TestReincarnationDefenseReportMapsPlayerAsDefender 验证防守波战报按玩家防守视角展示双方战力和武将。
func TestReincarnationDefenseReportMapsPlayerAsDefender(t *testing.T) {
	now := time.Date(2026, 6, 29, 13, 0, 0, 0, time.UTC)
	state := newPlayerState("player_reincarnation_defense_report", "防守战报", "wei", "caocao", now)
	run := ReincarnationRun{
		ID:     "run_defense_report",
		Level:  1,
		Status: ReincarnationRunRunning,
	}
	wave := ReincarnationWave{
		WaveIndex:      2,
		WaveType:       ReincarnationWaveDefense,
		EnemyFaction:   "shu",
		EnemyRemaining: map[string]int{"shuInfantry": 80},
		AllyBonus:      ReincarnationBonus{Label: "己方防守加成"},
		EnemyBonus:     ReincarnationBonus{Label: "敌方进攻加成"},
		RewardPreview:  []Reward{{Type: RewardTypeItem, ID: "training_token_small", Amount: 1}},
	}
	result := combat.CombatResult{
		Winner:       "defender",
		AttackPower:  900,
		DefensePower: 1200,
	}

	report := buildReincarnationReport(run, wave, &state, map[string]int{"qingZhouArmy": 100}, map[string]int{"qingZhouArmy": 5}, map[string]int{"shuInfantry": 20}, result, ReportViewDefense, true, now)
	report.PvpDefenderGenerals = buildPvpGeneralSnapshots(&state, []string{"caocao"})
	report.Detail = nil
	report = NormalizeBattleReport(report)

	if report.PlayerPower != 1200 || report.EnemyPower != 900 {
		t.Fatalf("expected player defense power 1200 and enemy attack power 900, got player=%d enemy=%d", report.PlayerPower, report.EnemyPower)
	}
	if report.Detail == nil || report.Detail.SecondarySide == nil {
		t.Fatalf("expected normalized defense detail, got %+v", report.Detail)
	}
	if report.Detail.PrimarySide.Role != "attacker" || report.Detail.PrimarySide.Power != 900 {
		t.Fatalf("expected enemy attacker side power 900, got %+v", report.Detail.PrimarySide)
	}
	if report.Detail.SecondarySide.Role != "defender" || report.Detail.SecondarySide.Power != 1200 {
		t.Fatalf("expected player defender side power 1200, got %+v", report.Detail.SecondarySide)
	}
	if len(report.Detail.PrimarySide.Generals) != 0 {
		t.Fatalf("dungeon attacker should not show player general, got %+v", report.Detail.PrimarySide.Generals)
	}
	if len(report.Detail.SecondarySide.Generals) != 1 || report.Detail.SecondarySide.Generals[0].ID != "caocao" {
		t.Fatalf("expected player general on defender side, got %+v", report.Detail.SecondarySide.Generals)
	}
}

// TestReincarnationBonusResetCostsGold 验证重置当前波随机加成会消耗账户金币并写流水。
func TestReincarnationBonusResetCostsGold(t *testing.T) {
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
	now := time.Date(2026, 6, 29, 12, 30, 0, 0, time.UTC)
	accountID := "account_reincarnation_reset"
	if err := repo.CreateAccount(Account{ID: accountID, Username: "reincarnation_reset", Gold: 100, CreatedAt: now}); err != nil {
		t.Fatalf("CreateAccount failed: %v", err)
	}
	state := newPlayerState("player_reincarnation_reset", "轮回重置测试", "wei", "caocao", now)
	if err := repo.CreatePlayer(accountID, state, now); err != nil {
		t.Fatalf("CreatePlayer failed: %v", err)
	}

	started, err := service.StartReincarnationRun(state.Player.ID, 1)
	if err != nil {
		t.Fatalf("StartReincarnationRun failed: %v", err)
	}
	wave := started.Run.Waves[0]
	reset, err := service.ResetReincarnationWaveBonus(state.Player.ID, wave.ID)
	if err != nil {
		t.Fatalf("ResetReincarnationWaveBonus failed: %v", err)
	}
	if reset.Cost != GetReincarnationConfig().BonusResetGoldCost {
		t.Fatalf("expected reset cost %d, got %d", GetReincarnationConfig().BonusResetGoldCost, reset.Cost)
	}
	if reset.AccountGold != 100-reset.Cost {
		t.Fatalf("expected account gold %d, got %d", 100-reset.Cost, reset.AccountGold)
	}
	accountAfter, err := repo.GetAccountByID(accountID)
	if err != nil {
		t.Fatalf("GetAccountByID failed: %v", err)
	}
	if accountAfter.Gold != reset.AccountGold {
		t.Fatalf("expected stored gold %d, got %d", reset.AccountGold, accountAfter.Gold)
	}
	entries, err := service.ListGoldLedger(GoldLedgerFilter{AccountID: accountID, RefType: LedgerRefReincarnationBonusReset})
	if err != nil {
		t.Fatalf("ListGoldLedger failed: %v", err)
	}
	if len(entries) != 1 || entries[0].Amount != reset.Cost || entries[0].RefID != wave.ID {
		t.Fatalf("unexpected reset ledger entries: %+v", entries)
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
	firstEnemyUnit := combatUnitIDsForFaction(run.Waves[0].EnemyFaction)[0]
	secondEnemyUnit := combatUnitIDsForFaction(run.Waves[1].EnemyFaction)[0]
	run.Waves[0].EnemyTroops = map[string]int{firstEnemyUnit: 1}
	run.Waves[0].EnemyRemaining = map[string]int{firstEnemyUnit: 1}
	run.Waves[1].EnemyTroops = map[string]int{secondEnemyUnit: 100000}
	run.Waves[1].EnemyRemaining = map[string]int{secondEnemyUnit: 100000}
	if err := repo.SaveReincarnationRun(run); err != nil {
		t.Fatalf("SaveReincarnationRun test fixture failed: %v", err)
	}
	for attempt := 0; attempt < 10 && run.CurrentWave == 1; attempt++ {
		wave := run.Waves[0]
		result, err := service.AttackReincarnationWave(state.Player.ID, wave.ID, map[string]int{"qingZhouArmy": 10000}, nil, "clear-wave-1-"+string(rune('a'+attempt)))
		if err != nil {
			t.Fatalf("AttackReincarnationWave attempt %d failed: %v", attempt, err)
		}
		run = result.Run
	}
	if run.CurrentWave != 2 {
		t.Fatalf("expected wave 1 cleared, current wave=%d remaining=%+v", run.CurrentWave, run.Waves[0].EnemyRemaining)
	}

	defenseWave := run.Waves[1]
	result, err := service.ReadyReincarnationDefense(state.Player.ID, defenseWave.ID, map[string]int{"qingZhouArmy": 1}, nil, "defense-fail-once")
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
