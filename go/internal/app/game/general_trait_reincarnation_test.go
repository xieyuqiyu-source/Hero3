// 本文件验证正式将领特性在轮回绝境攻防波中的真实状态、战斗记录、战报与幂等性。
package game

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"testing"
	"time"

	"hero3/internal/core/general"
)

type reincarnationTraitFixture struct {
	service    *Service
	repo       *MemoryRepository
	playerID   string
	wave       ReincarnationWave
	playerUnit string
	enemyUnit  string
	initial    int
}

// ensureReincarnationTraitUnit 注册等属性测试步兵，便于精确核对特性实际变化。
func ensureReincarnationTraitUnit(faction string, unitType string, name string) {
	ensureReincarnationTraitUnitCategory(faction, unitType, name, "infantry")
}

// ensureReincarnationTraitUnitCategory 注册指定分类的等属性测试兵种。
func ensureReincarnationTraitUnitCategory(faction string, unitType string, name string, category string) {
	unitsMu.Lock()
	defer unitsMu.Unlock()
	if activeUnits[faction] == nil {
		activeUnits[faction] = FactionUnits{}
	}
	activeUnits[faction][unitType] = UnitConfig{
		Name: name, Category: category,
		Stats: map[string]int{"attack": 10, "infantryDefense": 10, "cavalryDefense": 8, "carryCapacity": 5, "upkeep": 1},
	}
}

// newReincarnationTraitFixture 构造携带指定正式特性的单波轮回实例。
func newReincarnationTraitFixture(t *testing.T, id string, hero GeneralHeroConfig, enemyFaction string, waveType string, initial int, enemyAmount int) reincarnationTraitFixture {
	return newReincarnationTraitFixtureWithUnits(t, id, hero, enemyFaction, waveType, initial, enemyAmount, hero.Faction+"Infantry", "infantry", enemyFaction+"Infantry", "infantry")
}

// newReincarnationTraitFixtureWithUnits 使用指定兵种构造单波轮回实例。
func newReincarnationTraitFixtureWithUnits(t *testing.T, id string, hero GeneralHeroConfig, enemyFaction string, waveType string, initial int, enemyAmount int, playerUnit string, playerCategory string, enemyUnit string, enemyCategory string) reincarnationTraitFixture {
	t.Helper()
	originalUnits := GetUnitsConfig()
	originalReincarnationConfig := GetReincarnationConfig()
	t.Cleanup(func() {
		unitsMu.Lock()
		activeUnits = originalUnits
		unitsMu.Unlock()
		reincarnationConfigMu.Lock()
		reincarnationConfig = originalReincarnationConfig
		reincarnationConfigMu.Unlock()
	})
	root := filepath.Join("..", "..", "..", "config")
	if err := LoadUnitsConfig(filepath.Join(root, "units")); err != nil {
		t.Fatalf("LoadUnitsConfig failed: %v", err)
	}
	if err := LoadReincarnationConfig(filepath.Join(root, "reincarnation.json")); err != nil {
		t.Fatalf("LoadReincarnationConfig failed: %v", err)
	}
	setTestFactionsAndGenerals(t, FactionsConfig{
		hero.Faction: {Name: hero.Faction, Generals: []GeneralInfo{{ID: hero.ID, Name: hero.Name}}},
	}, GeneralsConfig{Enabled: true, Heroes: map[string]GeneralHeroConfig{hero.ID: hero}})

	now := time.Now().UTC()
	repo := NewMemoryRepository()
	service := NewServiceWithRepository(repo)
	accountID := "account_reincarnation_trait_" + id
	playerID := "player_reincarnation_trait_" + id
	if err := repo.CreateAccount(Account{ID: accountID, Username: id, CreatedAt: now}); err != nil {
		t.Fatalf("CreateAccount failed: %v", err)
	}
	ensureReincarnationTraitUnitCategory(hero.Faction, playerUnit, hero.Name+"测试兵", playerCategory)
	ensureReincarnationTraitUnitCategory(enemyFaction, enemyUnit, "轮回敌军测试兵", enemyCategory)
	state := newPlayerState(playerID, hero.Name+"轮回测试", hero.Faction, hero.ID, now)
	state.Army = []ArmyUnit{{UnitType: playerUnit, Amount: initial}}
	if err := repo.CreatePlayer(accountID, state, now); err != nil {
		t.Fatalf("CreatePlayer failed: %v", err)
	}
	waveIndex := 1
	if waveType == ReincarnationWaveDefense {
		waveIndex = 2
	}
	runID := "ra_trait_" + id
	wave := ReincarnationWave{
		ID:    runID + "_w" + map[bool]string{true: "02", false: "01"}[waveIndex == 2],
		RunID: runID, WaveIndex: waveIndex, WaveType: waveType, EnemyFaction: enemyFaction,
		EnemyTroops: map[string]int{enemyUnit: enemyAmount}, EnemyRemaining: map[string]int{enemyUnit: enemyAmount},
		AllyBonus: ReincarnationBonus{Label: "无加成"}, EnemyBonus: ReincarnationBonus{Label: "无加成"},
		Status: ReincarnationWaveActive, StartedAt: now,
	}
	run := ReincarnationRun{
		ID: runID, PlayerID: playerID, Level: 1, LevelName: "特性验收", Status: ReincarnationRunRunning,
		CurrentWave: waveIndex, StartedAt: now, ExpiresAt: now.Add(time.Hour), Waves: []ReincarnationWave{wave},
		PendingRewards: []Reward{}, Battles: []ReincarnationBattle{}, CreatedAt: now, UpdatedAt: now,
	}
	if err := repo.SaveReincarnationRun(run); err != nil {
		t.Fatalf("SaveReincarnationRun failed: %v", err)
	}
	return reincarnationTraitFixture{service: service, repo: repo, playerID: playerID, wave: wave, playerUnit: playerUnit, enemyUnit: enemyUnit, initial: initial}
}

// fightReincarnationTraitFixture 执行夹具中的攻防波。
func fightReincarnationTraitFixture(t *testing.T, fixture reincarnationTraitFixture, amount int, generalID string, actionID string) ReincarnationActionResult {
	t.Helper()
	generalIDs := []string(nil)
	if generalID != "" {
		generalIDs = []string{generalID}
	}
	var result ReincarnationActionResult
	var err error
	if fixture.wave.WaveType == ReincarnationWaveDefense {
		result, err = fixture.service.ReadyReincarnationDefense(fixture.playerID, fixture.wave.ID, map[string]int{fixture.playerUnit: amount}, generalIDs, actionID)
	} else {
		result, err = fixture.service.AttackReincarnationWave(fixture.playerID, fixture.wave.ID, map[string]int{fixture.playerUnit: amount}, generalIDs, actionID)
	}
	if err != nil {
		t.Fatalf("resolve reincarnation wave failed: %v", err)
	}
	if result.BattleReport == nil {
		t.Fatalf("expected reincarnation battle report")
	}
	return result
}

// TestReincarnationAttackAppliesBeforeBattleTraits 验证关羽在进攻波先真实扣兵，再用武圣加攻后的属性进入核心战力。
func TestReincarnationAttackAppliesBeforeBattleTraits(t *testing.T) {
	hero := GeneralHeroConfig{
		ID: "guanyu", Name: "关羽", Faction: "shu", Enabled: true,
		SpecialTrait: GeneralTraitConfig{TraitID: "shuiyan_qijun", TraitType: general.TraitTypeSpecial, Enabled: true, Scope: "enemy_army", Params: map[string]float64{"effectRate": 0.35, "maxAffectedRate": 0.35, "triggerChance": 1}},
		BonusTrait:   GeneralTraitConfig{TraitID: "wusheng_pojun", TraitType: general.TraitTypeBonus, Enabled: true, Scope: "self_army", AllowedSides: []string{"attacker"}, Params: map[string]float64{"attackBonusRate": 0.2}},
	}
	fixture := newReincarnationTraitFixture(t, "guanyu_attack", hero, "wei", ReincarnationWaveAttack, 500, 1000)
	result := fightReincarnationTraitFixture(t, fixture, 100, hero.ID, "guanyu-attack-once")
	report := result.BattleReport
	preDamage, ok := report.TraitOutcomes["shuiyan_qijun"].Detail["preBattleAffected"].(map[string]int)
	attackModified, modifiedOK := report.TraitOutcomes["wusheng_pojun"].Detail["attackModifiedUnits"].(map[string]int)
	if !ok || preDamage[fixture.enemyUnit] != 350 || !modifiedOK || attackModified[fixture.playerUnit] != 2 {
		t.Fatalf("expected exact pre-damage 350 and attack +2, outcomes=%+v", report.TraitOutcomes)
	}
	if report.PlayerPower != 1200 || report.DefenderLostUnits[fixture.enemyUnit] < 350 {
		t.Fatalf("expected modified attack power and pre-damage in final losses, power=%d losses=%+v", report.PlayerPower, report.DefenderLostUnits)
	}
	storedRun, err := fixture.repo.GetReincarnationRun(result.Run.ID)
	if err != nil || len(storedRun.Battles) != 1 {
		t.Fatalf("expected one stored trait battle, run=%+v err=%v", storedRun, err)
	}
	battle := storedRun.Battles[0]
	if battle.EnemyLosses[fixture.enemyUnit] != report.DefenderLostUnits[fixture.enemyUnit] || battle.EnemyRemaining[fixture.enemyUnit] != result.Run.Waves[0].EnemyRemaining[fixture.enemyUnit] || len(battle.TraitOutcomes) != 2 {
		t.Fatalf("expected battle record to match report and wave, battle=%+v report=%+v", battle, report)
	}
	storedState, err := fixture.repo.GetState(fixture.playerID)
	if err != nil || armySliceToMap(storedState.Army)[fixture.playerUnit] != fixture.initial-report.LostUnits[fixture.playerUnit] {
		t.Fatalf("expected real player army to match losses, state=%+v err=%v", storedState.Army, err)
	}
}

// TestReincarnationAttackAppliesAfterBattleRecovery 验证刘备进攻波的复活和返兵进入权威兵力、战报与战斗记录。
func TestReincarnationAttackAppliesAfterBattleRecovery(t *testing.T) {
	hero := GeneralHeroConfig{
		ID: "liubei", Name: "刘备", Faction: "shu", Enabled: true,
		SpecialTrait: GeneralTraitConfig{TraitID: "rende", TraitType: general.TraitTypeSpecial, Enabled: true, Scope: "self_army", Params: map[string]float64{"effectRate": 0.5, "maxReviveCount": 10000, "triggerChance": 1}},
		BonusTrait:   GeneralTraitConfig{TraitID: "renzhu_shouhu", TraitType: general.TraitTypeBonus, Enabled: true, Scope: "self_army", Params: map[string]float64{"lossReductionRate": 0.1, "maxReturnCount": 1000, "triggerChance": 1}},
	}
	fixture := newReincarnationTraitFixture(t, "liubei_recovery", hero, "wei", ReincarnationWaveAttack, 500, 1000)
	result := fightReincarnationTraitFixture(t, fixture, 100, hero.ID, "liubei-recovery-once")
	report := result.BattleReport
	revived := report.RevivedUnits[fixture.playerUnit]
	if report.LostUnits[fixture.playerUnit] <= 0 || revived <= 0 || len(report.TraitOutcomes) != 2 {
		t.Fatalf("expected real losses and two recovery traits, report=%+v", report)
	}
	expectedArmy := fixture.initial - report.LostUnits[fixture.playerUnit] + revived
	storedState, err := fixture.repo.GetState(fixture.playerID)
	if err != nil || armySliceToMap(storedState.Army)[fixture.playerUnit] != expectedArmy {
		t.Fatalf("expected recovered army %d, state=%+v err=%v", expectedArmy, storedState.Army, err)
	}
	storedRun, err := fixture.repo.GetReincarnationRun(result.Run.ID)
	if err != nil || len(storedRun.Battles) != 1 || storedRun.Battles[0].RevivedUnits[fixture.playerUnit] != revived || storedRun.Battles[0].SurvivedTroops[fixture.playerUnit] != report.SurvivedUnits[fixture.playerUnit] {
		t.Fatalf("expected recovery in battle record, run=%+v err=%v", storedRun, err)
	}
}

// TestReincarnationAttackZhenMiTraitsModifyPowerWithoutCapture 验证轮回进攻使用甄宓修改后的攻防，且旧俘虏效果彻底失效。
func TestReincarnationAttackZhenMiTraitsModifyPowerWithoutCapture(t *testing.T) {
	hero := GeneralHeroConfig{
		ID: "zhenmi", Name: "甄宓", Faction: "wei", Enabled: true,
		SpecialTrait: GeneralTraitConfig{TraitID: "meiren", TraitType: general.TraitTypeSpecial, Enabled: true, Scope: "self_army", AllowedSides: []string{"attacker"}, Params: map[string]float64{"attackBonusRate": 0.25, "triggerChance": 1}},
		BonusTrait:   GeneralTraitConfig{TraitID: "meihuo_raozhen", TraitType: general.TraitTypeBonus, Enabled: true, Scope: "enemy_army", AllowedSides: []string{"attacker"}, Params: map[string]float64{"enemyDefenseReductionRate": 0.25, "triggerChance": 1}},
	}
	fixture := newReincarnationTraitFixture(t, "zhenmi_stats", hero, "shu", ReincarnationWaveAttack, 500, 1000)
	result := fightReincarnationTraitFixture(t, fixture, 100, hero.ID, "zhenmi-stats-once")
	report := result.BattleReport
	if report.PlayerPower != 1300 || report.EnemyPower != 8000 || len(report.TraitOutcomes) != 2 {
		t.Fatalf("expected modified reincarnation power 1300/8000 and two outcomes, report=%+v", report)
	}
	if len(report.CapturedToGarrison) != 0 || len(report.CapturedUnits) != 0 {
		t.Fatalf("expected removed capture behavior to stay empty, report=%+v", report)
	}
	remaining := result.Run.Waves[0].EnemyRemaining[fixture.enemyUnit]
	if remaining != 1000-report.DefenderLostUnits[fixture.enemyUnit] {
		t.Fatalf("expected enemy state to deduct only recalculated deaths, remaining=%d report=%+v", remaining, report)
	}
	if report.Detail == nil || report.Detail.SecondarySide == nil {
		t.Fatalf("expected complete standard report sides, detail=%+v", report.Detail)
	}
	assertStandardUnitRow(t, report.ID, *report.Detail.SecondarySide, fixture.enemyUnit, 1000, report.DefenderLostUnits[fixture.enemyUnit], remaining)
	storedBattle := result.Run.Battles[0]
	if len(storedBattle.EnemyCaptured) != 0 || storedBattle.EnemyRemaining[fixture.enemyUnit] != remaining || storedBattle.TraitOutcomes["meiren"].TraitID != "meiren" || storedBattle.TraitOutcomes["meihuo_raozhen"].TraitID != "meihuo_raozhen" {
		t.Fatalf("expected current stat traits without captures in battle record, battle=%+v", storedBattle)
	}
	repeated := fightReincarnationTraitFixture(t, fixture, 100, hero.ID, "zhenmi-stats-once")
	if repeated.BattleReport.ID != report.ID {
		t.Fatalf("expected duplicate action to reuse report %s, got %s", report.ID, repeated.BattleReport.ID)
	}
	if len(repeated.Run.Battles) != 1 {
		t.Fatalf("expected duplicate action not to add a second battle, run=%+v", repeated.Run)
	}
}

// TestReincarnationDefenseAppliesDefenseAndEnemyDamageTraits 验证孙权加防与黄忠战后伤害在防守波真实生效。
func TestReincarnationDefenseAppliesDefenseAndEnemyDamageTraits(t *testing.T) {
	t.Run("sunquan_defense", func(t *testing.T) {
		hero := GeneralHeroConfig{
			ID: "sunquan", Name: "孙权", Faction: "wu", Enabled: true,
			BonusTrait: GeneralTraitConfig{TraitID: "jiangdong_gushou", TraitType: general.TraitTypeBonus, Enabled: true, Scope: "self_army", AllowedSides: []string{"defender", "reinforcement"}, Params: map[string]float64{"defenseBonusRate": 0.5, "triggerChance": 1}},
		}
		fixture := newReincarnationTraitFixture(t, "sunquan_defense", hero, "wei", ReincarnationWaveDefense, 500, 100)
		result := fightReincarnationTraitFixture(t, fixture, 100, hero.ID, "sunquan-defense-once")
		report := result.BattleReport
		if report.PlayerPower != 1500 || report.TraitOutcomes["jiangdong_gushou"].OwnerSide != "defender" {
			t.Fatalf("expected defense power 1500 and defender outcome, report=%+v", report)
		}
		storedState, err := fixture.repo.GetState(fixture.playerID)
		if err != nil || armySliceToMap(storedState.Army)[fixture.playerUnit] != fixture.initial-report.LostUnits[fixture.playerUnit] {
			t.Fatalf("expected defense state to match report, state=%+v err=%v", storedState.Army, err)
		}
	})

	t.Run("huangzhong_enemy_damage", func(t *testing.T) {
		hero := GeneralHeroConfig{
			ID: "huangzhong", Name: "黄忠", Faction: "shu", Enabled: true,
			BonusTrait: GeneralTraitConfig{TraitID: "laodang_yizhuang", TraitType: general.TraitTypeBonus, Enabled: true, Scope: "enemy_army", Params: map[string]float64{"effectRate": 0.1, "triggerChance": 1}},
		}
		fixture := newReincarnationTraitFixture(t, "huangzhong_defense", hero, "wei", ReincarnationWaveDefense, 500, 1000)
		result := fightReincarnationTraitFixture(t, fixture, 100, hero.ID, "huangzhong-defense-once")
		report := result.BattleReport
		extra, ok := report.TraitOutcomes["laodang_yizhuang"].Detail["extraLosses"].(map[string]int)
		if !ok || extra[fixture.enemyUnit] != 100 || report.TraitOutcomes["laodang_yizhuang"].OwnerSide != "defender" {
			t.Fatalf("expected defender-owned enemy extra losses 100, outcomes=%+v", report.TraitOutcomes)
		}
		if result.Run.Waves[0].EnemyRemaining[fixture.enemyUnit] != 1000-report.DefenderLostUnits[fixture.enemyUnit] {
			t.Fatalf("expected enemy wave state to match final losses, wave=%+v report=%+v", result.Run.Waves[0], report)
		}
	})
}

// TestReincarnationDefenseAppliesAfterBattleRecovery 验证刘备防守波返兵从主城真实损失中恢复并进入防守侧战报。
func TestReincarnationDefenseAppliesAfterBattleRecovery(t *testing.T) {
	hero := GeneralHeroConfig{
		ID: "liubei", Name: "刘备", Faction: "shu", Enabled: true,
		SpecialTrait: GeneralTraitConfig{TraitID: "rende", TraitType: general.TraitTypeSpecial, Enabled: true, Scope: "self_army", Params: map[string]float64{"effectRate": 0.5, "maxReviveCount": 10000, "triggerChance": 1}},
		BonusTrait:   GeneralTraitConfig{TraitID: "renzhu_shouhu", TraitType: general.TraitTypeBonus, Enabled: true, Scope: "self_army", Params: map[string]float64{"lossReductionRate": 0.1, "maxReturnCount": 1000, "triggerChance": 1}},
	}
	fixture := newReincarnationTraitFixture(t, "liubei_defense_recovery", hero, "wei", ReincarnationWaveDefense, 500, 1000)
	result := fightReincarnationTraitFixture(t, fixture, 100, hero.ID, "liubei-defense-recovery-once")
	report := result.BattleReport
	revived := report.RevivedUnits[fixture.playerUnit]
	if report.LostUnits[fixture.playerUnit] <= 0 || revived <= 0 || report.TraitOutcomes["rende"].OwnerSide != "defender" || report.TraitOutcomes["renzhu_shouhu"].OwnerSide != "defender" {
		t.Fatalf("expected defender-owned recovery outcomes, report=%+v", report)
	}
	expectedArmy := fixture.initial - report.LostUnits[fixture.playerUnit] + revived
	storedState, err := fixture.repo.GetState(fixture.playerID)
	if err != nil || armySliceToMap(storedState.Army)[fixture.playerUnit] != expectedArmy {
		t.Fatalf("expected recovered defense army %d, state=%+v err=%v", expectedArmy, storedState.Army, err)
	}
	if report.SurvivedUnits[fixture.playerUnit] != 100-report.LostUnits[fixture.playerUnit]+revived || result.Run.Battles[0].RevivedUnits[fixture.playerUnit] != revived {
		t.Fatalf("expected defense report and battle recovery to reconcile, report=%+v battle=%+v", report, result.Run.Battles[0])
	}
}

// TestReincarnationLossOnlyRecoveryDoesNotTreatDrawAsDefeat 验证轮回攻防平局都不会触发仅战败返兵。
func TestReincarnationLossOnlyRecoveryDoesNotTreatDrawAsDefeat(t *testing.T) {
	cases := []struct {
		name        string
		generalID   string
		generalName string
		traitID     string
		traitType   string
		rate        float64
	}{
		{name: "典韦护主死战", generalID: "dianwei", generalName: "典韦", traitID: "huzhu_sizhan", traitType: general.TraitTypeSpecial, rate: 0.15},
		{name: "郭嘉鬼才遗策", generalID: "guojia", generalName: "郭嘉", traitID: "guicai_yice", traitType: general.TraitTypeBonus, rate: 0.1},
	}
	for _, tc := range cases {
		for _, waveType := range []string{ReincarnationWaveAttack, ReincarnationWaveDefense} {
			t.Run(tc.name+"_"+waveType, func(t *testing.T) {
				trait := GeneralTraitConfig{
					TraitID: tc.traitID, TraitType: tc.traitType, Enabled: true, Scope: "self_army", RequiredOutcome: "loss",
					Params: map[string]float64{"lossReductionRate": tc.rate, "maxReturnCount": 10000, "triggerChance": 1},
				}
				hero := GeneralHeroConfig{ID: tc.generalID, Name: tc.generalName, Faction: "wei", Enabled: true}
				if tc.traitType == general.TraitTypeSpecial {
					hero.SpecialTrait = trait
				} else {
					hero.BonusTrait = trait
				}
				fixture := newReincarnationTraitFixture(t, "draw_"+tc.traitID+"_"+waveType, hero, "shu", waveType, 500, 100)
				result := fightReincarnationTraitFixture(t, fixture, 100, hero.ID, "draw-"+tc.traitID+"-"+waveType)
				report := result.BattleReport

				if report.Result != "draw" || report.WinnerSide != ReportWinnerDraw || report.OwnerOutcome != ReportOwnerOutcomeDraw || report.PlayerPower != 1000 || report.EnemyPower != 1000 {
					t.Fatalf("expected equal-power reincarnation draw, report=%+v", report)
				}
				if report.LostUnits[fixture.playerUnit] != 100 || report.DefenderLostUnits[fixture.enemyUnit] != 100 || report.SurvivedUnits[fixture.playerUnit] != 0 || len(report.RevivedUnits) != 0 {
					t.Fatalf("expected draw to keep original full losses without returns, report=%+v", report)
				}
				if report.GeneralExpGained != 100 || len(report.TraitTriggered) != 0 || len(report.TraitOutcomes) != 0 || standardReportHasTrait(report.Detail, tc.traitID) {
					t.Fatalf("expected draw experience 100 and empty loss-only timeline, report=%+v", report)
				}
				generals := report.PvpAttackerGenerals
				if waveType == ReincarnationWaveDefense {
					generals = report.PvpDefenderGenerals
				}
				if len(generals) != 1 || !pvpSnapshotHasTrait(generals[0], tc.traitID) {
					t.Fatalf("expected owned trait to remain in carried-general snapshot, generals=%+v", generals)
				}
				storedState, err := fixture.repo.GetState(fixture.playerID)
				if err != nil || armySliceToMap(storedState.Army)[fixture.playerUnit] != 400 {
					t.Fatalf("expected authoritative army 400 after full draw loss, state=%+v err=%v", storedState.Army, err)
				}
				if len(result.Run.Battles) != 1 || len(result.Run.Battles[0].RevivedUnits) != 0 || result.Run.Battles[0].SurvivedTroops[fixture.playerUnit] != 0 || len(result.Run.Battles[0].TraitOutcomes) != 0 {
					t.Fatalf("expected stored dungeon battle to preserve draw without return, battles=%+v", result.Run.Battles)
				}
			})
		}
	}
}

// TestReincarnationDefenseRejectsAttackerOnlyTrait 验证周瑜火攻不会在轮回防守波误触发。
func TestReincarnationDefenseRejectsAttackerOnlyTrait(t *testing.T) {
	hero := GeneralHeroConfig{
		ID: "zhouyu", Name: "周瑜", Faction: "wu", Enabled: true,
		SpecialTrait: GeneralTraitConfig{TraitID: "huogong", TraitType: general.TraitTypeSpecial, Enabled: true, Scope: "enemy_army", AllowedSides: []string{"attacker"}, Params: map[string]float64{"effectRate": 0.25, "damagePercent": 0.25, "triggerChance": 1}},
	}
	fixture := newReincarnationTraitFixture(t, "zhouyu_defense", hero, "wei", ReincarnationWaveDefense, 500, 100)
	result := fightReincarnationTraitFixture(t, fixture, 100, hero.ID, "zhouyu-defense-once")
	if _, triggered := result.BattleReport.TraitOutcomes["huogong"]; triggered || standardReportHasTrait(result.BattleReport.Detail, "huogong") {
		t.Fatalf("attacker-only fire attack must not trigger in dungeon defense, report=%+v", result.BattleReport)
	}
}

// TestReincarnationWithoutGeneralDoesNotBorrowHomeGeneral 验证轮回攻防空将领列表不会借用城内主将。
func TestReincarnationWithoutGeneralDoesNotBorrowHomeGeneral(t *testing.T) {
	hero := GeneralHeroConfig{
		ID: "caocao", Name: "曹操", Faction: "wei", Enabled: true,
		Buffs: map[string]float64{StatAttackBonus: 1},
		SpecialTrait: GeneralTraitConfig{
			TraitID: "huogong", TraitType: general.TraitTypeSpecial, Enabled: true,
			Scope: "enemy_army", AllowedSides: []string{"attacker"},
			Params: map[string]float64{"effectRate": 0.25, "damagePercent": 0.25, "triggerChance": 1},
		},
		BonusTrait: GeneralTraitConfig{
			TraitID: "weiwu_tongyu", TraitType: general.TraitTypeBonus, Enabled: true,
			Scope: "self_army", TargetUnitType: "huWei", AllowedSides: []string{"defender", "reinforcement"},
			Params: map[string]float64{"defenseBonusRate": 0.15, "triggerChance": 1},
		},
	}
	for _, waveType := range []string{ReincarnationWaveAttack, ReincarnationWaveDefense} {
		waveType := waveType
		t.Run(waveType, func(t *testing.T) {
			fixture := newReincarnationTraitFixtureWithUnits(
				t, "without_general_"+waveType, hero, "shu", waveType,
				500, 100, "huWei", "infantry", "shuInfantry", "infantry",
			)
			result := fightReincarnationTraitFixture(t, fixture, 100, "", "without-general-"+waveType)
			report := result.BattleReport
			if report.PlayerPower != 1000 {
				t.Fatalf("expected base player power 1000 without carried general, got %d", report.PlayerPower)
			}
			if len(report.TraitTriggered) != 0 || len(report.TraitOutcomes) != 0 || report.Detail == nil || len(report.Detail.Traits) != 0 || len(result.Run.Battles[0].TraitOutcomes) != 0 {
				t.Fatalf("expected no borrowed home-general timeline, report=%+v battle=%+v", report, result.Run.Battles[0])
			}
			if len(report.PvpAttackerGenerals) != 0 || len(report.PvpDefenderGenerals) != 0 || len(report.Detail.PrimarySide.Generals) != 0 || report.Detail.SecondarySide == nil || len(report.Detail.SecondarySide.Generals) != 0 {
				t.Fatalf("expected no borrowed general snapshot on either side, report=%+v", report)
			}
			stored, err := fixture.repo.GetState(fixture.playerID)
			if err != nil {
				t.Fatalf("GetState failed: %v", err)
			}
			if pvpTestGeneralExp(stored, hero.ID) != 0 || report.GeneralExpGained != 0 || report.GeneralLevelBefore != 0 || report.GeneralLevelAfter != 0 {
				t.Fatalf("expected home general progress unchanged, state=%+v report=%+v", stored.Generals, report)
			}
			wantArmy := fixture.initial - report.LostUnits[fixture.playerUnit]
			if armySliceToMap(stored.Army)[fixture.playerUnit] != wantArmy || report.SurvivedUnits[fixture.playerUnit] != 100-report.LostUnits[fixture.playerUnit] || result.Run.Battles[0].SurvivedTroops[fixture.playerUnit] != report.SurvivedUnits[fixture.playerUnit] {
				t.Fatalf("expected real army, report and dungeon battle to reconcile, army=%+v report=%+v battle=%+v", stored.Army, report, result.Run.Battles[0])
			}
		})
	}
}

type reincarnationFormalTraitCase struct {
	generalID   string
	generalName string
	traitID     string
	waveType    string
	expect      bool
}

// loadFormalReincarnationGenerals 读取正式将领配置作为副本适用矩阵的唯一输入。
func loadFormalReincarnationGenerals(t *testing.T) GeneralsConfig {
	t.Helper()
	path := filepath.Join("..", "..", "..", "config", "generals.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read formal generals config failed: %v", err)
	}
	var cfg GeneralsConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("decode formal generals config failed: %v", err)
	}
	return NormalizeGeneralsConfig(cfg)
}

// formalReincarnationTraitCases 把 50 项正式特性展开为进攻波和防守波的明确预期。
func formalReincarnationTraitCases(t *testing.T, cfg GeneralsConfig) []reincarnationFormalTraitCase {
	t.Helper()
	attackerOnly := map[string]bool{
		"meiren": true, "meihuo_raozhen": true, "huchi_chongzhen": true, "pojun_pofang": true,
		"sizhandaodi": true, "weizhen_xiaoyao": true, "wusheng_pojun": true, "wanren_nuhou": true,
		"baibu_chuanyang": true, "qibing_raohou": true, "xiaobawang_tieqi": true, "huogong": true,
		"meizhoulang_junlue": true,
	}
	defenderOnly := map[string]bool{
		"mouding_houfa": true, "dunzhen_fangyu": true, "longdan_jiuyuan": true,
		"gushou_hanzhong": true, "jiangdong_gushou": true, "weiwu_tongyu": true,
	}
	bothSides := map[string]bool{
		"yibing_touxi": true, "huzhu_sizhan": true, "weizhen_zhenhe": true,
		"guicai_yice": true, "rende": true, "renzhu_shouhu": true, "shuiyan_qijun": true,
		"zhenhe_quanjun": true, "qimen_dunjia": true, "wolong_mouzhi": true, "xiliang_tuji": true,
		"laodang_yizhuang": true, "huoshao_lianying": true, "lianying_zengshang": true,
		"kurouji": true, "kurou_fanji": true,
	}
	nonBattle := map[string]bool{
		"weiwu_haoling": true, "jixing_benxi": true, "shengui_zhicai": true, "wangzuo_zhicai": true,
		"neizheng_jingying": true, "qijin_qichu": true, "tianshen_xiafan": true,
		"jiangdong_haoling": true, "xiaobawang_zhuiji": true, "baiyi_dujiang": true,
		"baiyi_jixing": true, "kuairu_shandian": true, "xinyi_yonglie": true,
		"jinfan_jielue": true, "jinfan_qixi": true,
	}

	heroIDs := make([]string, 0, len(cfg.Heroes))
	for heroID := range cfg.Heroes {
		heroIDs = append(heroIDs, heroID)
	}
	sort.Strings(heroIDs)
	cases := make([]reincarnationFormalTraitCase, 0, len(heroIDs)*4)
	seen := map[string]bool{}
	for _, heroID := range heroIDs {
		hero := cfg.Heroes[heroID]
		for _, trait := range []GeneralTraitConfig{hero.SpecialTrait, hero.BonusTrait} {
			if trait.TraitID == "" {
				continue
			}
			classified := attackerOnly[trait.TraitID] || defenderOnly[trait.TraitID] || bothSides[trait.TraitID] || nonBattle[trait.TraitID]
			if !classified {
				t.Fatalf("formal trait %s has no explicit reincarnation classification", trait.TraitID)
			}
			if seen[trait.TraitID] {
				t.Fatalf("formal trait %s is configured more than once", trait.TraitID)
			}
			seen[trait.TraitID] = true
			cases = append(cases,
				reincarnationFormalTraitCase{generalID: hero.ID, generalName: hero.Name, traitID: trait.TraitID, waveType: ReincarnationWaveAttack, expect: attackerOnly[trait.TraitID] || bothSides[trait.TraitID]},
				reincarnationFormalTraitCase{generalID: hero.ID, generalName: hero.Name, traitID: trait.TraitID, waveType: ReincarnationWaveDefense, expect: defenderOnly[trait.TraitID] || bothSides[trait.TraitID]},
			)
		}
	}
	if len(seen) != 50 || len(cases) != 100 {
		t.Fatalf("expected 50 formal traits and 100 attack/defense cases, got traits=%d cases=%d", len(seen), len(cases))
	}
	return cases
}

// isolatedReincarnationFormalHero 仅启用本次矩阵要检查的一项正式特性并固定命中概率。
func isolatedReincarnationFormalHero(t *testing.T, cfg GeneralsConfig, tc reincarnationFormalTraitCase) GeneralHeroConfig {
	t.Helper()
	hero, ok := cfg.Heroes[tc.generalID]
	if !ok {
		t.Fatalf("formal general %s is missing", tc.generalID)
	}
	hero = cloneHeroConfig(hero)
	hero.Traits = nil
	if hero.SpecialTrait.TraitID == tc.traitID {
		hero.SpecialTrait.Enabled = true
		hero.SpecialTrait.Params["triggerChance"] = 1
		hero.BonusTrait.Enabled = false
	} else if hero.BonusTrait.TraitID == tc.traitID {
		hero.BonusTrait.Enabled = true
		hero.BonusTrait.Params["triggerChance"] = 1
		hero.SpecialTrait.Enabled = false
	} else {
		t.Fatalf("formal general %s does not own trait %s", tc.generalID, tc.traitID)
	}
	return hero
}

// reincarnationFormalTraitUnits 返回能让指定兵种效果命中真实目标的测试兵种。
func reincarnationFormalTraitUnits(hero GeneralHeroConfig, enemyFaction string, traitID string) (string, string, string, string) {
	playerUnit, playerCategory := hero.Faction+"Infantry", "infantry"
	enemyUnit, enemyCategory := enemyFaction+"Infantry", "infantry"
	switch traitID {
	case "weiwu_tongyu":
		playerUnit = "huWei"
	case "weizhen_xiaoyao":
		playerUnit, playerCategory = hero.Faction+"Cavalry", "cavalry"
	case "xiaobawang_tieqi":
		playerUnit, playerCategory = "overlordRider", "cavalry"
	case "xiliang_tuji":
		enemyUnit, enemyCategory = enemyFaction+"Cavalry", "cavalry"
	}
	return playerUnit, playerCategory, enemyUnit, enemyCategory
}

// reincarnationTraitHasActualIntegerChange 判断特性结果是否包含本场真实非零整数变化。
func reincarnationTraitHasActualIntegerChange(detail map[string]interface{}) bool {
	actualKeys := []string{
		"attackModifiedUnits", "infantryDefenseModifiedUnits", "cavalryDefenseModifiedUnits",
		"preBattleAffected", "suppressedUnits", "capturedUnits", "capturedToGarrison",
		"extraLosses", "targetExtraLosses", "reducedLosses", "revivedUnits", "returnedUnits",
	}
	for _, key := range actualKeys {
		values, ok := detail[key].(map[string]int)
		if !ok {
			continue
		}
		for _, value := range values {
			if value != 0 {
				return true
			}
		}
	}
	return false
}

// TestFormalReincarnationTraitApplicabilityMatrix 验证全部正式特性在轮回攻防波中的正反方向和实际结果。
func TestFormalReincarnationTraitApplicabilityMatrix(t *testing.T) {
	cfg := loadFormalReincarnationGenerals(t)
	for _, tc := range formalReincarnationTraitCases(t, cfg) {
		tc := tc
		waveName := map[string]string{ReincarnationWaveAttack: "进攻波", ReincarnationWaveDefense: "防守波"}[tc.waveType]
		t.Run(tc.generalName+"/"+tc.traitID+"/"+waveName, func(t *testing.T) {
			hero := isolatedReincarnationFormalHero(t, cfg, tc)
			enemyFaction := "wei"
			if hero.Faction == enemyFaction {
				enemyFaction = "shu"
			}
			playerUnit, playerCategory, enemyUnit, enemyCategory := reincarnationFormalTraitUnits(hero, enemyFaction, tc.traitID)
			fixture := newReincarnationTraitFixtureWithUnits(
				t, "matrix_"+tc.traitID+"_"+tc.waveType, hero, enemyFaction, tc.waveType,
				500, 1000, playerUnit, playerCategory, enemyUnit, enemyCategory,
			)
			result := fightReincarnationTraitFixture(t, fixture, 100, hero.ID, "matrix-action-"+tc.traitID+"-"+tc.waveType)
			report := result.BattleReport
			outcome, triggered := report.TraitOutcomes[tc.traitID]
			if triggered != tc.expect || standardReportHasTrait(report.Detail, tc.traitID) != tc.expect {
				t.Fatalf("expected trigger=%v in %s, outcomes=%+v detail=%+v", tc.expect, waveName, report.TraitOutcomes, report.Detail)
			}
			if len(report.TraitOutcomes) != map[bool]int{true: 1, false: 0}[tc.expect] || len(result.Run.Battles) != 1 || len(result.Run.Battles[0].TraitOutcomes) != len(report.TraitOutcomes) {
				t.Fatalf("expected isolated report and stored battle outcomes to match, report=%+v battle=%+v", report.TraitOutcomes, result.Run.Battles)
			}
			if !standardDetailGeneralHasTrait(report.Detail, tc.traitID) {
				t.Fatalf("expected owned trait %s to remain in pre-exp general snapshot, detail=%+v", tc.traitID, report.Detail)
			}
			if tc.expect {
				wantOwner := "attacker"
				if tc.waveType == ReincarnationWaveDefense {
					wantOwner = "defender"
				}
				if outcome.OwnerSide != wantOwner || outcome.OwnerGeneralID != hero.ID {
					t.Fatalf("expected outcome owner %s/%s, outcome=%+v", wantOwner, hero.ID, outcome)
				}
				if tc.traitID != "wolong_mouzhi" && tc.traitID != "kurouji" && !reincarnationTraitHasActualIntegerChange(outcome.Detail) {
					t.Fatalf("expected non-zero actual integer change for %s, outcome=%+v", tc.traitID, outcome)
				}
				if (tc.traitID == "wolong_mouzhi" || tc.traitID == "kurouji") && outcome.Detail["disabledTraitCount"] != 0 {
					t.Fatalf("expected no-target suppressor to preserve actual zero, outcome=%+v", outcome)
				}
			}
			if tc.traitID == "tianshen_xiafan" && tc.waveType == ReincarnationWaveAttack && (triggered || report.PlayerPower <= 1000) {
				t.Fatalf("expected passive force to change attack power without trigger outcome, power=%d outcome=%+v", report.PlayerPower, outcome)
			}
		})
	}
}
