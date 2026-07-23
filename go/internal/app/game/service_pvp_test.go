// 本文件验证 PVP 行军、战斗结算、将领特性和双方战报一致性。
package game

import (
	"errors"
	"hero3/internal/core/combat"
	"hero3/internal/core/general"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

func TestPvpAttackRejectsSelfAndSameAccount(t *testing.T) {
	svc, _, attacker, defender := newPvpTestService(t)
	if _, err := svc.StartPvpAttack(PvpAttackRequest{
		PlayerID:       attacker.Player.ID,
		TargetPlayerID: attacker.Player.ID,
		Troops:         map[string]int{"weiInfantry": 10},
	}); !errors.Is(err, ErrPvpTargetSelf) {
		t.Fatalf("expected self target error, got %v", err)
	}
	sameAccount := newPlayerState("player_pvp_same_account", "同账号", "wu", "sunquan", time.Now())
	repo := svc.repo.(*MemoryRepository)
	if err := repo.CreatePlayer("account_pvp_a", sameAccount, time.Now()); err != nil {
		t.Fatalf("CreatePlayer same account failed: %v", err)
	}
	if _, err := svc.StartPvpAttack(PvpAttackRequest{
		PlayerID:       attacker.Player.ID,
		TargetPlayerID: sameAccount.Player.ID,
		Troops:         map[string]int{"weiInfantry": 10},
	}); !errors.Is(err, ErrPvpSameAccountTarget) {
		t.Fatalf("expected same account target error, got %v", err)
	}
	attacker.Army = []ArmyUnit{{UnitType: "weiInfantry", Amount: 20}}
	repo.players[attacker.Player.ID] = attacker
	if _, err := svc.StartPvpAttack(PvpAttackRequest{
		PlayerID:       attacker.Player.ID,
		TargetPlayerID: defender.Player.ID,
		Troops:         map[string]int{"weiInfantry": 10},
	}); err != nil {
		t.Fatalf("expected different account target to pass, got %v", err)
	}
}

// TestPvpReinforcementReportIncludesZeroLossParticipant 验证实际参战但零损失的援军仍收到协防战报。
func TestPvpReinforcementReportIncludesZeroLossParticipant(t *testing.T) {
	now := time.Date(2026, 7, 10, 13, 0, 0, 0, time.UTC)
	record := Reinforcement{
		ID:                "rein_zero_loss",
		OwnerPlayerID:     "player_rein_owner",
		FromPlayerID:      "player_rein_owner",
		FromPlayerName:    "零损援军",
		FromPlayerFaction: "wei",
		Status:            ReinforcementStatusStationed,
		RemainingTroops:   map[string]int{"weiInfantry": 100},
		Rules:             GarrisonRules{CanFight: true},
	}
	changed := applyPvpReinforcementLosses([]Reinforcement{record}, map[string]map[string]int{}, now)
	if len(changed) != 1 {
		t.Fatalf("expected zero-loss participant recorded as changed, got %+v", changed)
	}
	attacker := newPlayerState("player_rein_attacker", "进攻方", "shu", "liubei", now)
	defender := newPlayerState("player_rein_defender", "防守方", "wei", "caocao", now)
	baseReport := buildPvpBattleReport("br_zero_loss_defense", &defender, &attacker, &PvpMarch{MarchType: PvpMarchTypeAttack}, "defender_victory", 120, 80, map[string]int{"weiInfantry": 10}, map[string]int{}, map[string]int{"shuInfantry": 10}, map[string]int{}, map[string]int{}, now.Format(resourceDateLayout), "defense")
	baseReport.EventID = "event_zero_loss"
	baseReport.ViewType = ReportViewDefense
	baseReport.OwnerSide = ReportOwnerSideDefender
	baseReport.PvpReinforcements = []DefenseReinforcementUnit{reinforcementReportSnapshot(record, record.RemainingTroops, 12)}
	baseReport = NormalizeBattleReport(baseReport)
	reports := buildPvpReinforcementReportsByPhase(&baseReport, baseReport.EventID, &attacker, &defender, changed, map[string]map[string]int{}, map[string]int{record.ID: 12}, reinforcementTraitReportPhases{}, baseReport.Result, now.Format(resourceDateLayout))
	if len(reports) != 1 || reports[0].GeneralExpGained != 12 {
		t.Fatalf("expected zero-loss reinforcement report with exp, got %+v", reports)
	}
	if len(reports[0].PvpReinforcements) != 1 || reports[0].PvpReinforcements[0].GeneralExpGained != 12 {
		t.Fatalf("expected reinforcement snapshot exp, got %+v", reports[0].PvpReinforcements)
	}
	if reports[0].Detail == nil || reports[0].Detail.SecondarySide == nil || reports[0].Detail.PrimarySide.Role != "attacker" || reports[0].Detail.SecondarySide.Role != "defender" {
		t.Fatalf("expected reinforcement owner to receive complete attack and defense snapshot, got %+v", reports[0].Detail)
	}
}

// TestApplyReinforcementGeneralExpFromReportsTargetsOwnerBatch 验证完整援军列表不会导致多份战报重复发放经验。
func TestApplyReinforcementGeneralExpFromReportsTargetsOwnerBatch(t *testing.T) {
	svc, repo, firstHelper, secondHelper := newPvpTestService(t)
	now := time.Date(2026, 7, 17, 16, 0, 0, 0, time.UTC)
	allReinforcements := []DefenseReinforcementUnit{
		{ReinforcementID: "rein_first", FromPlayerID: firstHelper.Player.ID, Generals: []ReinforcementGeneralSnapshot{{ID: "caocao", Name: "曹操"}}},
		{ReinforcementID: "rein_second", FromPlayerID: secondHelper.Player.ID, Generals: []ReinforcementGeneralSnapshot{{ID: "liubei", Name: "刘备"}}},
	}
	repo.reinforcements["rein_first"] = Reinforcement{ID: "rein_first", FromPlayerID: firstHelper.Player.ID, ToPlayerID: secondHelper.Player.ID, OwnerPlayerID: firstHelper.Player.ID, HostPlayerID: secondHelper.Player.ID, Generals: []ReinforcementGeneralSnapshot{{ID: "caocao", Name: "曹操", Level: 1}}}
	repo.reinforcements["rein_second"] = Reinforcement{ID: "rein_second", FromPlayerID: secondHelper.Player.ID, ToPlayerID: firstHelper.Player.ID, OwnerPlayerID: secondHelper.Player.ID, HostPlayerID: firstHelper.Player.ID, Generals: []ReinforcementGeneralSnapshot{{ID: "liubei", Name: "刘备", Level: 1}}}
	reports := []BattleReport{
		{ID: "br_rein_first", EventID: "battle_reward_once", PlayerID: firstHelper.Player.ID, OwnerPlayerID: firstHelper.Player.ID, ViewType: ReportViewReinforcement, GeneralExpGained: 12, PvpReinforcements: allReinforcements, Detail: &BattleReportDetail{Extra: map[string]interface{}{"reinforcement": map[string]interface{}{"reinforcementId": "rein_first"}}}},
		{ID: "br_rein_second", EventID: "battle_reward_once", PlayerID: secondHelper.Player.ID, OwnerPlayerID: secondHelper.Player.ID, ViewType: ReportViewReinforcement, GeneralExpGained: 20, PvpReinforcements: allReinforcements, Detail: &BattleReportDetail{Extra: map[string]interface{}{"reinforcement": map[string]interface{}{"reinforcementId": "rein_second"}}}},
	}

	if err := svc.applyReinforcementGeneralExpFromReports(reports, now); err != nil {
		t.Fatalf("apply reinforcement general exp: %v", err)
	}
	if err := svc.applyReinforcementGeneralExpFromReports(reports, now.Add(time.Minute)); err != nil {
		t.Fatalf("repeat reinforcement general exp: %v", err)
	}
	updatedFirst, _ := repo.GetState(firstHelper.Player.ID)
	updatedSecond, _ := repo.GetState(secondHelper.Player.ID)
	if got := pvpTestGeneralExp(updatedFirst, "caocao"); got != 12 {
		t.Fatalf("expected first helper to gain 12 exp once, got %d", got)
	}
	if got := pvpTestGeneralExp(updatedSecond, "liubei"); got != 20 {
		t.Fatalf("expected second helper to gain 20 exp once, got %d", got)
	}
	if stored := repo.reinforcements["rein_first"].Generals[0]; stored.Exp != 12 || stored.Level != updatedFirst.Generals[0].Level {
		t.Fatalf("expected first reinforcement progress baseline updated, got %+v", stored)
	}
	if stored := repo.reinforcements["rein_second"].Generals[0]; stored.Exp != 20 || stored.Level != updatedSecond.Generals[0].Level {
		t.Fatalf("expected second reinforcement progress baseline updated, got %+v", stored)
	}
	if !reinforcementGeneralExpWasApplied(repo.reinforcements["rein_first"].RewardState, "battle_reward_once") || !reinforcementGeneralExpWasApplied(repo.reinforcements["rein_second"].RewardState, "battle_reward_once") {
		t.Fatalf("expected both reinforcement rewards to retain idempotency markers")
	}
}

// TestApplyPvpReinforcementGeneralExpReturnsRepositoryError 验证援军发奖失败会返回结算调用方。
func TestApplyPvpReinforcementGeneralExpReturnsRepositoryError(t *testing.T) {
	svc, _, helper, _ := newPvpTestService(t)
	now := time.Date(2026, 7, 17, 16, 30, 0, 0, time.UTC)
	battle := PvpBattle{
		ID: "battle_missing_reinforcement",
		Result: map[string]any{
			"reinforcementGeneralExp": map[string]int{"rein_missing": 15},
		},
		ReinforcementSnapshot: []DefenseReinforcementUnit{{
			ReinforcementID: "rein_missing",
			FromPlayerID:    helper.Player.ID,
			Generals:        []ReinforcementGeneralSnapshot{{ID: "caocao", Name: "曹操"}},
		}},
	}

	if err := svc.applyPvpReinforcementGeneralExp(battle, now); !errors.Is(err, ErrReinforcementNotFound) {
		t.Fatalf("expected missing reinforcement error, got %v", err)
	}
}

// TestPvpReinforcementLossesPreserveBattleStartSnapshot 验证损耗结算不会污染主战报使用的战前兵力。
func TestPvpReinforcementLossesPreserveBattleStartSnapshot(t *testing.T) {
	now := time.Date(2026, 7, 10, 14, 0, 0, 0, time.UTC)
	record := Reinforcement{
		ID:                "rein_snapshot",
		FromPlayerID:      "player_snapshot",
		FromPlayerName:    "快照援军",
		FromPlayerFaction: "wei",
		Status:            ReinforcementStatusStationed,
		RemainingTroops:   map[string]int{"weiInfantry": 100},
		Losses:            map[string]int{"weiInfantry": 5},
		Generals: []ReinforcementGeneralSnapshot{{
			ID: "caocao", Name: "曹操", Level: 1, Exp: generalExpRequiredForLevelForTest(2) - 5,
		}},
		Rules: GarrisonRules{CanFight: true},
	}

	changed := applyPvpReinforcementLosses([]Reinforcement{record}, map[string]map[string]int{
		record.ID: {"weiInfantry": 30},
	}, now)
	if len(changed) != 1 || changed[0].RemainingTroops["weiInfantry"] != 70 || changed[0].Losses["weiInfantry"] != 35 {
		t.Fatalf("expected changed reinforcement to contain losses, got %+v", changed)
	}
	if record.RemainingTroops["weiInfantry"] != 100 || record.Losses["weiInfantry"] != 5 {
		t.Fatalf("expected battle-start record to remain unchanged, got %+v", record)
	}

	snapshot := buildPvpReinforcementSnapshot([]Reinforcement{record}, map[string]int{record.ID: 12})
	if len(snapshot) != 1 || snapshot[0].Troops["weiInfantry"] != 100 || snapshot[0].GeneralExpGained != 12 {
		t.Fatalf("expected battle-start troops and general exp in snapshot, got %+v", snapshot)
	}
	general := snapshot[0].Generals[0]
	if general.Exp != 0 || general.Level != 2 || general.GeneralExpGained == nil || *general.GeneralExpGained != 12 || general.GeneralLevelBefore == nil || *general.GeneralLevelBefore != 1 || general.GeneralLevelAfter == nil || *general.GeneralLevelAfter != 2 {
		t.Fatalf("expected reinforcement general level result without exposing cumulative exp, got %+v", general)
	}
}

// TestPvpReinforcementSnapshotExcludesNonParticipants 验证主战报只保存实际可参战驻防。
func TestPvpReinforcementSnapshotExcludesNonParticipants(t *testing.T) {
	base := Reinforcement{
		ID:                "rein_active",
		FromPlayerID:      "player_active",
		FromPlayerFaction: "wei",
		Status:            ReinforcementStatusStationed,
		RemainingTroops:   map[string]int{"weiInfantry": 100},
		Rules:             GarrisonRules{CanFight: true},
	}
	marching := base
	marching.ID = "rein_marching"
	marching.Status = ReinforcementStatusMarching
	disabled := base
	disabled.ID = "rein_disabled"
	disabled.Rules = GarrisonRules{CanFight: false, CanRecall: true}
	empty := base
	empty.ID = "rein_empty"
	empty.RemainingTroops = map[string]int{}

	snapshot := buildPvpReinforcementSnapshot([]Reinforcement{base, marching, disabled, empty}, map[string]int{base.ID: 12})
	if len(snapshot) != 1 || snapshot[0].ReinforcementID != base.ID || snapshot[0].GeneralExpGained != 12 {
		t.Fatalf("expected only active reinforcement in report snapshot, got %+v", snapshot)
	}
}

// TestPvpBattleProjectionCannotBypassReportVisibility 验证 PVP 战斗接口不能绕过战报可见性读取防守快照。
func TestPvpBattleProjectionCannotBypassReportVisibility(t *testing.T) {
	repo := NewMemoryRepository()
	service := NewServiceWithRepository(repo)
	report := NormalizeBattleReport(BattleReport{
		ID: "br_pvp_hidden", PlayerID: "player_pvp_hidden", OwnerPlayerID: "player_pvp_hidden",
		ViewType: ReportViewAttack, SourceType: ReportSourcePlayerCity, BattleType: "attack", Type: "attack",
		TargetID: "player_pvp_target", Result: "defender_victory", DefenderRevealed: false,
		CreatedAt: time.Now().UTC().Format(resourceDateLayout),
	})
	report.Detail.Visibility.ShowEnemyGenerals = false
	if err := repo.SaveReport(report); err != nil {
		t.Fatalf("SaveReport failed: %v", err)
	}
	battle := service.projectPvpBattleForPlayer(PvpBattle{
		AttackerPlayerID: "player_pvp_hidden", AttackerReportID: report.ID,
		DefenderSnapshot:      map[string]any{"troops": map[string]int{"shuInfantry": 100}},
		ReinforcementSnapshot: []DefenseReinforcementUnit{{ReinforcementID: "rein_hidden", Troops: map[string]int{"shuInfantry": 20}, Generals: []ReinforcementGeneralSnapshot{{ID: "guanyu"}}, GeneralExpGained: 88, GeneralLevelBefore: 10, GeneralLevelAfter: 11}},
	}, report.PlayerID)
	if battle.DefenderSnapshot != nil || len(battle.ReinforcementSnapshot[0].Troops) != 0 || len(battle.ReinforcementSnapshot[0].Generals) != 0 || battle.ReinforcementSnapshot[0].GeneralExpGained != 0 || battle.ReinforcementSnapshot[0].GeneralLevelBefore != 0 || battle.ReinforcementSnapshot[0].GeneralLevelAfter != 0 {
		t.Fatalf("expected PVP enemy snapshots redacted, got %+v", battle)
	}
}

func TestPvpTargetsExposeStableWorldPositions(t *testing.T) {
	svc, repo, attacker, defender := newPvpTestService(t)
	if _, err := repo.AssignWorldPosition(attacker.Player.ID, defaultWorldID, 0, 0, "test"); err != nil {
		t.Fatalf("AssignWorldPosition attacker failed: %v", err)
	}
	if _, err := repo.AssignWorldPosition(defender.Player.ID, defaultWorldID, 3, 4, "test"); err != nil {
		t.Fatalf("AssignWorldPosition defender failed: %v", err)
	}

	first, err := svc.ListPvpTargets(attacker.Player.ID)
	if err != nil {
		t.Fatalf("ListPvpTargets first failed: %v", err)
	}
	second, err := svc.ListPvpTargets(attacker.Player.ID)
	if err != nil {
		t.Fatalf("ListPvpTargets second failed: %v", err)
	}
	if first.Self != second.Self {
		t.Fatalf("expected stable self position, first=%+v second=%+v", first.Self, second.Self)
	}
	if first.Self.X != 0 || first.Self.Y != 0 || first.WorldSize != defaultPvpWorldSize {
		t.Fatalf("unexpected self world position: %+v world=%d", first.Self, first.WorldSize)
	}
	if len(first.Items) == 0 || first.Items[0].PlayerID != defender.Player.ID || first.Items[0].Position.X != 3 || first.Items[0].Position.Y != 4 || first.Items[0].Distance != 7 {
		t.Fatalf("expected target positions and distance, got %+v", first.Items)
	}
	if first.Items[0].Direction == "" || first.Items[0].ReinforceSeconds <= 0 {
		t.Fatalf("expected target direction and reinforcement estimate, got %+v", first.Items[0])
	}
}

func TestPvpTargetsExposeWorldMapActionFields(t *testing.T) {
	svc, repo, attacker, defender := newPvpTestService(t)
	now := time.Now().UTC()
	sameAccount := newPlayerState("player_pvp_target_same_account", "同账号城", "wu", "sunquan", now)
	if err := repo.CreatePlayer("account_pvp_a", sameAccount, now); err != nil {
		t.Fatalf("CreatePlayer same account failed: %v", err)
	}
	if _, err := repo.AssignWorldPosition(attacker.Player.ID, defaultWorldID, 10, 10, "test"); err != nil {
		t.Fatalf("AssignWorldPosition attacker failed: %v", err)
	}
	if _, err := repo.AssignWorldPosition(defender.Player.ID, defaultWorldID, 12, 10, "test"); err != nil {
		t.Fatalf("AssignWorldPosition defender failed: %v", err)
	}
	if _, err := repo.AssignWorldPosition(sameAccount.Player.ID, defaultWorldID, 11, 10, "test"); err != nil {
		t.Fatalf("AssignWorldPosition same account failed: %v", err)
	}
	targets, err := svc.ListPvpTargets(attacker.Player.ID)
	if err != nil {
		t.Fatalf("ListPvpTargets failed: %v", err)
	}
	normal := findPvpTargetSummaryForTest(targets.Items, defender.Player.ID)
	if normal.PlayerID == "" {
		t.Fatalf("expected defender target, got %+v", targets.Items)
	}
	if normal.Relation != WorldRelationOther || normal.Status != WorldTargetStatusAttackable || !normal.CanScout || !normal.CanAttack || !normal.CanPlunder || !normal.CanReinforce {
		t.Fatalf("expected normal target to expose world map action fields, got %+v", normal)
	}
	if _, err := svc.SetPvpProtection(defender.Player.ID, PvpProtectionTypeManual, time.Hour, "test", now); err != nil {
		t.Fatalf("SetPvpProtection defender failed: %v", err)
	}
	truceTargets, err := svc.ListPvpTargets(attacker.Player.ID)
	if err != nil {
		t.Fatalf("ListPvpTargets truce failed: %v", err)
	}
	truce := findPvpTargetSummaryForTest(truceTargets.Items, defender.Player.ID)
	if truce.Status != WorldTargetStatusTruce || truce.CanAttack || truce.CanPlunder {
		t.Fatalf("expected manual protection to expose truce status, got %+v", truce)
	}
	if truce.AttackReason != "目标处于免战保护" || truce.PlunderReason != "目标处于免战保护" {
		t.Fatalf("expected truce action reasons, got %+v", truce)
	}
	defenderState, err := repo.GetPvpPlayerState(defender.Player.ID, now)
	if err != nil {
		t.Fatalf("GetPvpPlayerState defender failed: %v", err)
	}
	defenderState.ProtectionType = ""
	defenderState.ProtectedUntil = ""
	if err := repo.SavePvpPlayerState(defenderState, now); err != nil {
		t.Fatalf("SavePvpPlayerState defender clear failed: %v", err)
	}
	same := findPvpTargetSummaryForTest(targets.Items, sameAccount.Player.ID)
	if same.PlayerID == "" {
		t.Fatalf("expected same account target, got %+v", targets.Items)
	}
	if same.CanScout || same.CanAttack || same.CanPlunder || !same.CanReinforce {
		t.Fatalf("expected same account target action fields, got %+v", same)
	}
	if same.ScoutReason != "同账号存档不能侦查" || same.AttackReason != "同账号存档不能攻击" || same.PlunderReason != "同账号存档不能掠夺" {
		t.Fatalf("expected same account action reasons, got %+v", same)
	}
	if _, err := svc.SetPvpProtection(sameAccount.Player.ID, PvpProtectionTypeManual, time.Hour, "same account truce", now); err != nil {
		t.Fatalf("SetPvpProtection same account failed: %v", err)
	}
	sameTruceTargets, err := svc.ListPvpTargets(attacker.Player.ID)
	if err != nil {
		t.Fatalf("ListPvpTargets same account truce failed: %v", err)
	}
	sameTruce := findPvpTargetSummaryForTest(sameTruceTargets.Items, sameAccount.Player.ID)
	if sameTruce.Status != WorldTargetStatusTruce {
		t.Fatalf("expected same account truce status to be visible, got %+v", sameTruce)
	}
	if sameTruce.AttackReason != "同账号存档不能攻击" || sameTruce.PlunderReason != "同账号存档不能掠夺" {
		t.Fatalf("expected same account reasons not to be overridden by truce, got %+v", sameTruce)
	}
	state := newDefaultPvpPlayerState(attacker.Player.ID, now)
	state.DailyAttackCount = state.DailyAttackLimit
	if err := repo.SavePvpPlayerState(state, now); err != nil {
		t.Fatalf("SavePvpPlayerState daily limit failed: %v", err)
	}
	limitedTargets, err := svc.ListPvpTargets(attacker.Player.ID)
	if err != nil {
		t.Fatalf("ListPvpTargets daily limit failed: %v", err)
	}
	limited := findPvpTargetSummaryForTest(limitedTargets.Items, defender.Player.ID)
	if limited.CanAttack || limited.CanPlunder || !limited.CanScout || !limited.CanReinforce || limited.Status != WorldTargetStatusUnavailable {
		t.Fatalf("expected daily limit to disable attack and plunder only, got %+v", limited)
	}
	if limited.AttackReason != "今日攻击次数已用完" || limited.PlunderReason != "今日攻击次数已用完" {
		t.Fatalf("expected daily limit action reasons, got %+v", limited)
	}
	limitedSame := findPvpTargetSummaryForTest(limitedTargets.Items, sameAccount.Player.ID)
	if limitedSame.AttackReason != "同账号存档不能攻击" || limitedSame.PlunderReason != "同账号存档不能掠夺" {
		t.Fatalf("expected daily limit not to override same account reasons, got %+v", limitedSame)
	}
}

// findPvpTargetSummaryForTest 在测试目标列表中查找指定玩家。
func findPvpTargetSummaryForTest(items []PvpTargetSummary, playerID string) PvpTargetSummary {
	for _, item := range items {
		if item.PlayerID == playerID {
			return item
		}
	}
	return PvpTargetSummary{}
}

func TestPvpTargetsFilterByMapViewport(t *testing.T) {
	svc, repo, attacker, defender := newPvpTestService(t)
	if _, err := repo.AssignWorldPosition(attacker.Player.ID, defaultWorldID, 0, 0, "test"); err != nil {
		t.Fatalf("AssignWorldPosition attacker failed: %v", err)
	}
	if _, err := repo.AssignWorldPosition(defender.Player.ID, defaultWorldID, 3, 4, "test"); err != nil {
		t.Fatalf("AssignWorldPosition defender failed: %v", err)
	}

	near, err := svc.ListPvpTargetsInArea(attacker.Player.ID, PvpTargetFilter{
		CenterX: 3,
		CenterY: 4,
		Radius:  1,
		Limit:   10,
	})
	if err != nil {
		t.Fatalf("ListPvpTargetsInArea near failed: %v", err)
	}
	if len(near.Items) != 1 || near.Items[0].PlayerID != defender.Player.ID {
		t.Fatalf("expected defender in near viewport, got %+v", near.Items)
	}

	far, err := svc.ListPvpTargetsInArea(attacker.Player.ID, PvpTargetFilter{
		CenterX: 0,
		CenterY: 0,
		Radius:  1,
		Limit:   10,
	})
	if err != nil {
		t.Fatalf("ListPvpTargetsInArea far failed: %v", err)
	}
	for _, target := range far.Items {
		if target.PlayerID == defender.Player.ID {
			t.Fatalf("expected defender outside far viewport, got %+v", far.Items)
		}
	}
}

func TestPvpScoutUsesFactionScoutUnitsAndRevealsOnSurvival(t *testing.T) {
	svc, repo, attacker, defender := newPvpTestService(t)
	addPvpScoutTestUnits(t)
	oldSettledAt := time.Now().Add(-2 * time.Hour).UTC()
	attacker.Army = []ArmyUnit{{UnitType: "weiScout", Amount: 5}, {UnitType: "weiInfantry", Amount: 20}}
	defender.Army = []ArmyUnit{{UnitType: "shuScout", Amount: 2}, {UnitType: "shuInfantry", Amount: 10}}
	defender.Resources.Items = map[string]int{"wood": 0, "stone": 0, "iron": 0, "food": 0}
	defender.Resources.Capacity = map[string]int{"wood": 4800, "stone": 4800, "iron": 4800, "food": 4800}
	defender.ResourceSettledAt = oldSettledAt.Format(resourceDateLayout)
	repo.players[attacker.Player.ID] = attacker
	repo.players[defender.Player.ID] = defender

	result, err := svc.ScoutPvpTarget(PvpScoutRequest{PlayerID: attacker.Player.ID, TargetPlayerID: defender.Player.ID})
	if err != nil {
		t.Fatalf("ScoutPvpTarget failed: %v", err)
	}
	if result.March.MarchType != PvpMarchTypeScout || result.March.Status != PvpMarchStatusMarching {
		t.Fatalf("expected marching scout, got %+v", result.March)
	}
	if armySliceToMap(result.Army)["weiScout"] != 0 || result.March.AttackTroops["weiScout"] != 5 {
		t.Fatalf("expected all scouts to leave city, response=%+v", result)
	}
	if _, total, err := repo.ListReportsByQuery(BattleReportQuery{PlayerID: attacker.Player.ID, BattleType: "scout", Page: 1, PageSize: 10}); err != nil || total != 0 {
		t.Fatalf("scout report must not exist before arrival, total=%d err=%v", total, err)
	}
	march := repo.pvpMarches[result.March.ID]
	march.ArrivesAt = time.Now().Add(-time.Second).UTC().Format(resourceDateLayout)
	repo.pvpMarches[march.ID] = march
	if _, err := svc.ResolvePvpMarch(march.ID); err != nil {
		t.Fatalf("ResolvePvpMarch scout failed: %v", err)
	}
	resolvedMarch := repo.pvpMarches[march.ID]
	if resolvedMarch.Status != PvpMarchStatusReturning || resolvedMarch.AttackTroops["weiScout"] != 3 {
		t.Fatalf("expected three surviving scouts to return, got %+v", resolvedMarch)
	}
	attackerReports, total, err := repo.ListReportsByQuery(BattleReportQuery{PlayerID: attacker.Player.ID, BattleType: "scout", Page: 1, PageSize: 10})
	if err != nil || total != 1 || len(attackerReports) != 1 {
		t.Fatalf("expected one attacker scout report, total=%d reports=%+v err=%v", total, attackerReports, err)
	}
	report := attackerReports[0]
	if report.ViewType != ReportViewScout || report.OwnerOutcome != ReportOwnerOutcomeIntelSuccess {
		t.Fatalf("expected scout view and intel success, got view=%s ownerOutcome=%s", report.ViewType, report.OwnerOutcome)
	}
	if !report.DefenderRevealed || report.DispatchedUnits["weiScout"] != 5 || report.LostUnits["weiScout"] != 2 {
		t.Fatalf("unexpected successful scout report: %+v", report)
	}
	if report.DefenderLostUnits["shuScout"] != 2 {
		t.Fatalf("expected defender scouts lost, got %+v", report.DefenderLostUnits)
	}
	updatedDefender, _ := repo.GetState(defender.Player.ID)
	if armySliceToMap(updatedDefender.Army)["shuScout"] != 0 {
		t.Fatalf("expected defender scout units removed, got %+v", updatedDefender.Army)
	}
	if updatedDefender.Resources.Items["wood"] <= 0 || report.DefenderResources["wood"] <= 0 {
		t.Fatalf("expected scout to settle defender resources into report, state=%+v report=%+v", updatedDefender.Resources.Items, report.DefenderResources)
	}
	settledAt, err := time.Parse(resourceDateLayout, updatedDefender.ResourceSettledAt)
	if err != nil || !settledAt.After(oldSettledAt) {
		t.Fatalf("expected defender resource settlement timestamp to advance, got %s err=%v", updatedDefender.ResourceSettledAt, err)
	}
	if report.Detail == nil || !report.Detail.Visibility.ShowEnemyResources {
		t.Fatalf("expected standard scout detail to reveal resources, got %+v", report.Detail)
	}
	scoutExtra, ok := report.Detail.Extra["scout"].(map[string]interface{})
	if !ok || scoutExtra["success"] != true || scoutExtra["scoutUnitType"] != "weiScout" {
		t.Fatalf("expected scout extra snapshot, got %+v", report.Detail.Extra)
	}
	defenderReports, total, err := repo.ListReportsByQuery(BattleReportQuery{PlayerID: defender.Player.ID, ViewType: ReportViewDefense, BattleType: "scout", Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("ListReportsByQuery defender scout failed: %v", err)
	}
	if total != 1 || len(defenderReports) != 1 {
		t.Fatalf("expected one defender scout report, total=%d reports=%+v", total, defenderReports)
	}
	if defenderReports[0].PlayerID != defender.Player.ID || defenderReports[0].TargetID != attacker.Player.ID || defenderReports[0].Title == "" {
		t.Fatalf("unexpected defender scout report: %+v", defenderReports[0])
	}
	resolvedMarch.ReturnsAt = time.Now().Add(-time.Second).UTC().Format(resourceDateLayout)
	repo.pvpMarches[resolvedMarch.ID] = resolvedMarch
	if _, err := svc.CompletePvpRecall(resolvedMarch.ID); err != nil {
		t.Fatalf("CompletePvpRecall scout failed: %v", err)
	}
	updatedAttacker, _ := repo.GetState(attacker.Player.ID)
	if armySliceToMap(updatedAttacker.Army)["weiScout"] != 3 || repo.pvpMarches[resolvedMarch.ID].Status != PvpMarchStatusResolved {
		t.Fatalf("expected surviving scouts to return to city, army=%+v march=%+v", updatedAttacker.Army, repo.pvpMarches[resolvedMarch.ID])
	}
}

func TestPvpScoutFailureHidesTargetIntel(t *testing.T) {
	svc, repo, attacker, defender := newPvpTestService(t)
	addPvpScoutTestUnits(t)
	attacker.Army = []ArmyUnit{{UnitType: "weiScout", Amount: 2}}
	defender.Army = []ArmyUnit{{UnitType: "shuScout", Amount: 5}, {UnitType: "shuInfantry", Amount: 10}}
	repo.players[attacker.Player.ID] = attacker
	repo.players[defender.Player.ID] = defender

	result, err := svc.ScoutPvpTarget(PvpScoutRequest{PlayerID: attacker.Player.ID, TargetPlayerID: defender.Player.ID})
	if err != nil {
		t.Fatalf("ScoutPvpTarget failed: %v", err)
	}
	march := repo.pvpMarches[result.March.ID]
	march.ArrivesAt = time.Now().Add(-time.Second).UTC().Format(resourceDateLayout)
	repo.pvpMarches[march.ID] = march
	if _, err := svc.ResolvePvpMarch(march.ID); err != nil {
		t.Fatalf("ResolvePvpMarch scout failed: %v", err)
	}
	reports, total, err := repo.ListReportsByQuery(BattleReportQuery{PlayerID: attacker.Player.ID, BattleType: "scout", Page: 1, PageSize: 10})
	if err != nil || total != 1 || len(reports) != 1 {
		t.Fatalf("expected failed scout report, total=%d reports=%+v err=%v", total, reports, err)
	}
	report := reports[0]
	if report.DefenderRevealed || len(report.DefenderUnits) != 0 || len(report.DefenderResources) != 0 {
		t.Fatalf("failed scout should hide target intel, got %+v", report)
	}
	if report.Detail == nil || report.Detail.Visibility.ShowEnemyResources {
		t.Fatalf("expected standard detail to hide scout intel, got %+v", report.Detail)
	}
	updatedAttacker, _ := repo.GetState(attacker.Player.ID)
	if armySliceToMap(updatedAttacker.Army)["weiScout"] != 0 || repo.pvpMarches[march.ID].Status != PvpMarchStatusResolved {
		t.Fatalf("expected attacker scouts annihilated without return, army=%+v march=%+v", updatedAttacker.Army, repo.pvpMarches[march.ID])
	}
}

func TestPvpScoutRequiresOwnFactionScoutUnit(t *testing.T) {
	svc, repo, attacker, defender := newPvpTestService(t)
	addPvpScoutTestUnits(t)
	attacker.Army = []ArmyUnit{{UnitType: "weiInfantry", Amount: 20}, {UnitType: "shuScout", Amount: 10}}
	defender.Army = []ArmyUnit{{UnitType: "shuScout", Amount: 1}}
	repo.players[attacker.Player.ID] = attacker
	repo.players[defender.Player.ID] = defender

	if _, err := svc.ScoutPvpTarget(PvpScoutRequest{PlayerID: attacker.Player.ID, TargetPlayerID: defender.Player.ID}); !errors.Is(err, ErrInsufficientArmy) {
		t.Fatalf("expected own faction scout requirement, got %v", err)
	}
}

func TestPvpMarchResolvesBattleAndReturnsSurvivors(t *testing.T) {
	svc, repo, attacker, defender := newPvpTestService(t)
	attacker.Army = []ArmyUnit{{UnitType: "weiInfantry", Amount: 100}}
	defender.Army = []ArmyUnit{{UnitType: "weiInfantry", Amount: 5}}
	repo.players[attacker.Player.ID] = attacker
	repo.players[defender.Player.ID] = defender

	started, err := svc.StartPvpAttack(PvpAttackRequest{
		PlayerID:       attacker.Player.ID,
		TargetPlayerID: defender.Player.ID,
		MarchMode:      PvpMarchTypeAttack,
		Troops:         map[string]int{"weiInfantry": 50},
		GeneralIDs:     []string{"caocao"},
	})
	if err != nil {
		t.Fatalf("StartPvpAttack failed: %v", err)
	}
	afterStart, err := repo.GetState(attacker.Player.ID)
	if err != nil {
		t.Fatalf("GetState attacker failed: %v", err)
	}
	if armySliceToMap(afterStart.Army)["weiInfantry"] != 50 {
		t.Fatalf("expected 50 infantry reserved for march, got %+v", afterStart.Army)
	}
	if len(started.GeneralAssignments) == 0 {
		t.Fatalf("expected pvp response to include general assignments")
	}

	forcePvpMarchDue(t, repo, started.March.ID)
	battle, err := svc.ResolvePvpMarch(started.March.ID)
	if err != nil {
		t.Fatalf("ResolvePvpMarch failed: %v", err)
	}
	if battle.Status != PvpBattleStatusResolved || battle.AttackerReportID == "" || battle.DefenderReportID == "" {
		t.Fatalf("unexpected battle result: %+v", battle)
	}
	if battle.Result["pointsDelta"] == nil {
		t.Fatalf("expected battle points delta, got %+v", battle.Result)
	}
	attackerGenerals, ok := battle.AttackerSnapshot["generals"].([]PvpGeneralSnapshot)
	if !ok || len(attackerGenerals) != 1 || attackerGenerals[0].ID != "caocao" {
		t.Fatalf("expected attacker general snapshot, got %+v", battle.AttackerSnapshot["generals"])
	}
	defenderGenerals, ok := battle.DefenderSnapshot["generals"].([]PvpGeneralSnapshot)
	if !ok || len(defenderGenerals) != 1 || defenderGenerals[0].ID != "liubei" {
		t.Fatalf("expected defender auto general snapshot, got %+v", battle.DefenderSnapshot["generals"])
	}
	march, err := repo.GetPvpMarch(started.March.ID)
	if err != nil {
		t.Fatalf("GetPvpMarch failed: %v", err)
	}
	if march.Status != PvpMarchStatusReturning || march.BattleID != battle.ID {
		t.Fatalf("unexpected march after resolve: %+v", march)
	}
	survivors := totalTroops(march.AttackTroops)
	if survivors <= 0 {
		t.Fatalf("expected surviving troops to return, got %+v", march.AttackTroops)
	}
	afterBattle, err := repo.GetState(attacker.Player.ID)
	if err != nil {
		t.Fatalf("GetState after battle failed: %v", err)
	}
	if armySliceToMap(afterBattle.Army)["weiInfantry"] != 50 {
		t.Fatalf("expected survivors still returning after battle, got %+v", afterBattle.Army)
	}
	forcePvpReturnDue(t, repo, started.March.ID)
	completed, err := svc.CompletePvpRecall(started.March.ID)
	if err != nil {
		t.Fatalf("CompletePvpRecall battle return failed: %v", err)
	}
	if completed.March.Status != PvpMarchStatusResolved {
		t.Fatalf("expected battle return to resolve march, got %+v", completed.March)
	}
	if armySliceToMap(completed.Army)["weiInfantry"] != 50+survivors {
		t.Fatalf("expected battle survivors returned, survivors=%d army=%+v", survivors, completed.Army)
	}
	reports, total, err := repo.ListReports(attacker.Player.ID, 10, 0)
	if err != nil || total == 0 || reports[0].ID != battle.AttackerReportID {
		t.Fatalf("expected attacker report saved, reports=%+v total=%d err=%v", reports, total, err)
	}
	if reports[0].PvpPointsDelta["self"] == 0 {
		t.Fatalf("expected attacker report pvp points delta, got %+v", reports[0].PvpPointsDelta)
	}
	if len(reports[0].PvpAttackerGenerals) != 1 || reports[0].PvpAttackerGenerals[0].ID != "caocao" {
		t.Fatalf("expected attacker report general snapshot, got %+v", reports[0].PvpAttackerGenerals)
	}
	defenderReports, defenderReportTotal, err := repo.ListReports(defender.Player.ID, 10, 0)
	if err != nil || defenderReportTotal == 0 || defenderReports[0].ID != battle.DefenderReportID {
		t.Fatalf("expected defender report saved, reports=%+v total=%d err=%v", defenderReports, defenderReportTotal, err)
	}
	if len(defenderReports[0].PvpDefenderGenerals) != 1 || defenderReports[0].PvpDefenderGenerals[0].ID != "liubei" {
		t.Fatalf("expected defender report auto general snapshot, got %+v", defenderReports[0].PvpDefenderGenerals)
	}
	if defenderReports[0].Result == "defender_defeat" || defenderReports[0].OwnerSide != ReportOwnerSideDefender || defenderReports[0].OwnerOutcome == "" {
		t.Fatalf("expected defender report to use standard owner semantics, got result=%s ownerSide=%s ownerOutcome=%s", defenderReports[0].Result, defenderReports[0].OwnerSide, defenderReports[0].OwnerOutcome)
	}
	attackerLosses := pvpTestLossesFromBattle(t, battle, "attacker")
	defenderLosses := pvpTestLossesFromBattle(t, battle, "defender")
	expectedAttackerExp := calculateGeneralBattleExpFromLosses(defender.Player.Faction, pvpTestUnitLosses(defenderLosses))
	expectedDefenderExp := calculateGeneralBattleExpFromLosses(attacker.Player.Faction, pvpTestUnitLosses(attackerLosses))
	if expectedAttackerExp <= 0 || expectedDefenderExp <= 0 {
		t.Fatalf("expected both sides to gain positive exp, attackerExp=%d defenderExp=%d battleLosses=%+v", expectedAttackerExp, expectedDefenderExp, battle.Losses)
	}
	attackerAfterExp, err := repo.GetState(attacker.Player.ID)
	if err != nil {
		t.Fatalf("GetState attacker exp failed: %v", err)
	}
	defenderAfterExp, err := repo.GetState(defender.Player.ID)
	if err != nil {
		t.Fatalf("GetState defender exp failed: %v", err)
	}
	if got := pvpTestGeneralExp(attackerAfterExp, "caocao"); got != expectedAttackerExp {
		t.Fatalf("expected attacker general exp %d, got %d losses=%+v", expectedAttackerExp, got, defenderLosses)
	}
	if got := pvpTestGeneralExp(defenderAfterExp, "liubei"); got != expectedDefenderExp {
		t.Fatalf("expected defender general exp %d, got %d losses=%+v", expectedDefenderExp, got, attackerLosses)
	}
	if reports[0].GeneralExpGained != expectedAttackerExp {
		t.Fatalf("expected attacker report exp %d, got %d", expectedAttackerExp, reports[0].GeneralExpGained)
	}
	if defenderReports[0].GeneralExpGained != expectedDefenderExp {
		t.Fatalf("expected defender report exp %d, got %d", expectedDefenderExp, defenderReports[0].GeneralExpGained)
	}
	if defenderReports[0].Detail == nil || len(defenderReports[0].Detail.PrimarySide.Generals) != 1 || defenderReports[0].Detail.SecondarySide == nil || len(defenderReports[0].Detail.SecondarySide.Generals) != 1 {
		t.Fatalf("expected defense report to preserve both general snapshots, got %+v", defenderReports[0].Detail)
	}
	defenseViewAttacker := defenderReports[0].Detail.PrimarySide.Generals[0]
	defenseViewDefender := defenderReports[0].Detail.SecondarySide.Generals[0]
	if defenseViewAttacker.GeneralExpGained == nil || *defenseViewAttacker.GeneralExpGained != expectedAttackerExp {
		t.Fatalf("expected defense report attacker exp %d, got %+v", expectedAttackerExp, defenseViewAttacker)
	}
	if defenseViewDefender.GeneralExpGained == nil || *defenseViewDefender.GeneralExpGained != expectedDefenderExp {
		t.Fatalf("expected defense report defender exp %d, got %+v", expectedDefenderExp, defenseViewDefender)
	}
	detail, err := svc.GetPvpBattle(attacker.Player.ID, battle.ID)
	if err != nil {
		t.Fatalf("GetPvpBattle failed: %v", err)
	}
	if detail.ID != battle.ID || detail.Result["pointsDelta"] == nil {
		t.Fatalf("unexpected battle detail: %+v", detail)
	}
	attackerPvp, err := svc.GetPvpState(attacker.Player.ID)
	if err != nil {
		t.Fatalf("GetPvpState attacker failed: %v", err)
	}
	defenderPvp, err := svc.GetPvpState(defender.Player.ID)
	if err != nil {
		t.Fatalf("GetPvpState defender failed: %v", err)
	}
	if attackerPvp.SeasonPoints == 0 {
		t.Fatalf("expected attacker PVP points updated, got %+v", attackerPvp)
	}
	if len(defenderPvp.RevengeRecords) != 1 || defenderPvp.RevengeRecords[0].AttackerPlayerID != attacker.Player.ID {
		t.Fatalf("expected defender revenge record against attacker, got %+v", defenderPvp.RevengeRecords)
	}
	rankings, err := svc.ListPvpRankings(attacker.Player.ID, 10)
	if err != nil {
		t.Fatalf("ListPvpRankings failed: %v", err)
	}
	if len(rankings.Items) < 2 || rankings.Items[0].PlayerID != attacker.Player.ID || rankings.Items[0].Points == 0 {
		t.Fatalf("expected attacker to lead rankings, got %+v", rankings.Items)
	}
	season, err := svc.GetPvpSeason(attacker.Player.ID)
	if err != nil {
		t.Fatalf("GetPvpSeason failed: %v", err)
	}
	if season.Season.ID == "" || season.Self == nil || season.Self.PlayerID != attacker.Player.ID {
		t.Fatalf("unexpected season response: %+v", season)
	}
}

// TestPvpGeneralLevelChangeMatchesStoredStateAndBothReportFormats 验证主战武将升级前后值与真实状态、旧战报和标准战报一致。
func TestPvpGeneralLevelChangeMatchesStoredStateAndBothReportFormats(t *testing.T) {
	svc, repo, attacker, defender := newPvpTestService(t)
	attacker.Army = []ArmyUnit{{UnitType: "weiInfantry", Amount: 100}}
	defender.Army = []ArmyUnit{{UnitType: "weiInfantry", Amount: 5}}
	baselineExp := generalExpRequiredForLevelForTest(2) - 1
	setPvpTestGeneralProgress(&attacker, "caocao", 1, baselineExp)
	repo.players[attacker.Player.ID] = attacker
	repo.players[defender.Player.ID] = defender

	started, err := svc.StartPvpAttack(PvpAttackRequest{
		PlayerID: attacker.Player.ID, TargetPlayerID: defender.Player.ID, MarchMode: PvpMarchTypeAttack,
		Troops: map[string]int{"weiInfantry": 50}, GeneralIDs: []string{"caocao"},
	})
	if err != nil {
		t.Fatalf("StartPvpAttack failed: %v", err)
	}
	forcePvpMarchDue(t, repo, started.March.ID)
	battle, err := svc.ResolvePvpMarch(started.March.ID)
	if err != nil {
		t.Fatalf("ResolvePvpMarch failed: %v", err)
	}
	reports, _, err := repo.ListReports(attacker.Player.ID, 10, 0)
	if err != nil || len(reports) == 0 || reports[0].ID != battle.AttackerReportID {
		t.Fatalf("expected attacker report, reports=%+v err=%v", reports, err)
	}
	report := reports[0]
	if report.GeneralExpGained <= 0 || report.GeneralLevelBefore != 1 || report.GeneralLevelAfter != 2 {
		t.Fatalf("expected positive exp and level 1 -> 2 in legacy report, got %+v", report)
	}
	if report.Detail == nil || report.Detail.Rewards.GeneralExp != report.GeneralExpGained || report.Detail.Rewards.GeneralLevelBefore != 1 || report.Detail.Rewards.GeneralLevelAfter != 2 {
		t.Fatalf("expected standard report rewards to match legacy progress, detail=%+v", report.Detail)
	}
	stored, err := repo.GetState(attacker.Player.ID)
	if err != nil {
		t.Fatalf("GetState attacker failed: %v", err)
	}
	if got := pvpTestGeneralExp(stored, "caocao"); got != baselineExp+report.GeneralExpGained {
		t.Fatalf("expected cumulative exp %d, got %d", baselineExp+report.GeneralExpGained, got)
	}
	if got := pvpTestGeneralLevel(stored, "caocao"); got != report.GeneralLevelAfter {
		t.Fatalf("expected stored level %d, got %d", report.GeneralLevelAfter, got)
	}
}

// TestPvpAttackRejectsMultipleGeneralsAtomically 验证武将校验失败不会遗留结算、扣兵、占用、行军或战报副作用。
func TestPvpAttackRejectsMultipleGeneralsAtomically(t *testing.T) {
	svc, repo, attacker, defender := newPvpTestService(t)
	setRealCaoCaoGuardConfig(t)
	attacker.Army = []ArmyUnit{{UnitType: "huWei", Amount: 100}}
	attacker.Generals = append(attacker.Generals, *newGeneral("wei", "xiahoudun"))
	attacker.ResourceSettledAt = time.Now().UTC().Add(-3 * time.Second).Format(resourceDateLayout)
	attacker.GeneralTraitProgress = map[string]float64{guardProductionProgressKey("caocao", "weiwu_haoling", "huWei"): 0.5}
	repo.players[attacker.Player.ID] = attacker
	before, err := repo.GetState(attacker.Player.ID)
	if err != nil {
		t.Fatalf("GetState before failed: %v", err)
	}

	if _, err = svc.StartPvpAttack(PvpAttackRequest{
		PlayerID:       attacker.Player.ID,
		TargetPlayerID: defender.Player.ID,
		MarchMode:      PvpMarchTypeAttack,
		Troops:         map[string]int{"huWei": 10},
		GeneralIDs:     []string{"caocao", "xiahoudun"},
	}); !errors.Is(err, ErrInvalidGeneral) {
		t.Fatalf("expected multiple generals to be rejected, got %v", err)
	}
	after, err := repo.GetState(attacker.Player.ID)
	if err != nil {
		t.Fatalf("GetState after failed: %v", err)
	}
	if !reflect.DeepEqual(after, before) {
		t.Fatalf("failed PVP request must leave player state unchanged\nbefore=%+v\nafter=%+v", before, after)
	}
	if marches, listErr := repo.ListPvpMarchesForPlayer(attacker.Player.ID); listErr != nil || len(marches) != 0 {
		t.Fatalf("failed PVP request must create no march, marches=%+v err=%v", marches, listErr)
	}
	if reports, total, listErr := repo.ListReports(attacker.Player.ID, 10, 0); listErr != nil || total != 0 || len(reports) != 0 {
		t.Fatalf("failed PVP request must create no report, reports=%+v total=%d err=%v", reports, total, listErr)
	}
}

// TestPvpCarriedGeneralTriggersTrait 验证携带武将的火攻会改变真实守军并写入一致战报。
func TestPvpCarriedGeneralTriggersTrait(t *testing.T) {
	setTestFactionsAndGenerals(t, FactionsConfig{
		"wei": {Name: "魏国", Generals: []GeneralInfo{{ID: "caocao", Name: "曹操"}}},
		"shu": {Name: "蜀国", Generals: []GeneralInfo{{ID: "liubei", Name: "刘备"}}},
	}, GeneralsConfig{
		Enabled: true,
		Heroes: map[string]GeneralHeroConfig{
			"caocao": {
				ID:      "caocao",
				Name:    "曹操",
				Faction: "wei",
				Enabled: true,
				Traits: []GeneralTraitConfig{{
					TraitID: "huogong",
					Enabled: true,
					Params:  map[string]float64{"damagePercent": 0.5, "triggerChance": 1},
				}},
			},
			"liubei": {ID: "liubei", Name: "刘备", Faction: "shu", Enabled: true},
		},
	})
	svc, repo, attacker, defender := newPvpTestService(t)
	attacker.Army = []ArmyUnit{{UnitType: "weiInfantry", Amount: 100}}
	defender.Army = []ArmyUnit{{UnitType: "weiInfantry", Amount: 100}}
	repo.players[attacker.Player.ID] = attacker
	repo.players[defender.Player.ID] = defender

	started, err := svc.StartPvpAttack(PvpAttackRequest{
		PlayerID:       attacker.Player.ID,
		TargetPlayerID: defender.Player.ID,
		MarchMode:      PvpMarchTypeAttack,
		Troops:         map[string]int{"weiInfantry": 1},
		GeneralIDs:     []string{"caocao"},
	})
	if err != nil {
		t.Fatalf("StartPvpAttack failed: %v", err)
	}
	forcePvpMarchDue(t, repo, started.March.ID)
	battle, err := svc.ResolvePvpMarch(started.March.ID)
	if err != nil {
		t.Fatalf("ResolvePvpMarch failed: %v", err)
	}
	reports, _, err := repo.ListReports(attacker.Player.ID, 10, 0)
	if err != nil || len(reports) == 0 {
		t.Fatalf("expected attacker report, reports=%+v err=%v", reports, err)
	}
	report := reports[0]
	if report.ID != battle.AttackerReportID {
		t.Fatalf("expected latest attacker report %s, got %s", battle.AttackerReportID, report.ID)
	}
	if len(report.TraitTriggered) == 0 || report.TraitOutcomes["huogong"].TraitID != "huogong" {
		t.Fatalf("expected huogong trait in pvp report, got triggered=%+v outcomes=%+v", report.TraitTriggered, report.TraitOutcomes)
	}
	if report.DefenderLostUnits["weiInfantry"] <= 0 {
		t.Fatalf("expected huogong to add defender losses, got %+v", report.DefenderLostUnits)
	}
	extraDamage, ok := report.TraitOutcomes["huogong"].Detail["extraDamage"].(int)
	if !ok || extraDamage <= 0 || extraDamage > report.DefenderLostUnits["weiInfantry"] {
		t.Fatalf("expected fire report number to be part of real defender losses, extra=%v losses=%+v", report.TraitOutcomes["huogong"].Detail["extraDamage"], report.DefenderLostUnits)
	}
	byUnit, ok := report.TraitOutcomes["huogong"].Detail["targetExtraLosses"].(map[string]int)
	if !ok || byUnit["weiInfantry"] != extraDamage {
		t.Fatalf("expected fire per-unit loss to match total %d, outcome=%+v", extraDamage, report.TraitOutcomes["huogong"])
	}
	if report.TraitOutcomes["huogong"].Detail["triggerChance"] != 1.0 {
		t.Fatalf("expected persisted fire trigger chance 1, outcome=%+v", report.TraitOutcomes["huogong"])
	}
	if report.TraitOutcomes["huogong"].Detail["damagePercent"] != 0.5 {
		t.Fatalf("expected persisted fire design damage 0.5, outcome=%+v", report.TraitOutcomes["huogong"])
	}
	defenderAfter, err := repo.GetState(defender.Player.ID)
	if err != nil {
		t.Fatalf("GetState defender failed: %v", err)
	}
	if got := armySliceToMap(defenderAfter.Army)["weiInfantry"]; got != 100-report.DefenderLostUnits["weiInfantry"] {
		t.Fatalf("expected report losses to match real defender army, remaining=%d reportLoss=%d", got, report.DefenderLostUnits["weiInfantry"])
	}
}

// TestPvpZhouYuTraitsStackAcrossBattlePhases 验证周瑜战前加攻与战后火攻按阶段叠加，并与双方战报和真实兵力一致。
func TestPvpZhouYuTraitsStackAcrossBattlePhases(t *testing.T) {
	setTestFactionsAndGenerals(t, FactionsConfig{
		"wu":  {Name: "吴国", Generals: []GeneralInfo{{ID: "zhouyu", Name: "周瑜"}}},
		"shu": {Name: "蜀国", Generals: []GeneralInfo{{ID: "liubei", Name: "刘备"}}},
	}, GeneralsConfig{Enabled: true, Heroes: map[string]GeneralHeroConfig{
		"zhouyu": {
			ID: "zhouyu", Name: "周瑜", Faction: "wu", Enabled: true,
			SpecialTrait: GeneralTraitConfig{
				TraitID: "huogong", TraitType: general.TraitTypeSpecial, Enabled: true,
				Scope: "enemy_army", AllowedSides: []string{"attacker"},
				Params: map[string]float64{"effectRate": 0.25, "damagePercent": 0.25, "triggerChance": 1},
			},
			BonusTrait: GeneralTraitConfig{
				TraitID: "meizhoulang_junlue", TraitType: general.TraitTypeBonus, Enabled: true,
				Scope: "self_army", AllowedSides: []string{"attacker"},
				Params: map[string]float64{"attackBonusRate": 0.05},
			},
		},
		"liubei": {ID: "liubei", Name: "刘备", Faction: "shu", Enabled: true},
	}})
	svc, repo, attacker, defender := newPvpTestServiceForGenerals(t, "wu", "zhouyu", "shu", "liubei")
	attacker.Army = []ArmyUnit{{UnitType: "wuInfantry", Amount: 100}}
	defender.Army = []ArmyUnit{{UnitType: "shuInfantry", Amount: 1000}}
	repo.players[attacker.Player.ID] = attacker
	repo.players[defender.Player.ID] = defender

	started, err := svc.StartPvpAttack(PvpAttackRequest{
		PlayerID: attacker.Player.ID, TargetPlayerID: defender.Player.ID, MarchMode: PvpMarchTypeAttack,
		Troops: map[string]int{"wuInfantry": 100}, GeneralIDs: []string{"zhouyu"},
	})
	if err != nil {
		t.Fatalf("StartPvpAttack failed: %v", err)
	}
	forcePvpMarchDue(t, repo, started.March.ID)
	battle, err := svc.ResolvePvpMarch(started.March.ID)
	if err != nil {
		t.Fatalf("ResolvePvpMarch failed: %v", err)
	}
	if attackPower, ok := battle.Result["attackerPower"].(float64); !ok || attackPower != 1100 {
		t.Fatalf("expected Zhou Yu pre-battle bonus to raise attack power to 1100, got %v", battle.Result["attackerPower"])
	}

	attackerReports, _, err := repo.ListReports(attacker.Player.ID, 10, 0)
	if err != nil || len(attackerReports) == 0 || attackerReports[0].ID != battle.AttackerReportID {
		t.Fatalf("expected attacker report, reports=%+v err=%v", attackerReports, err)
	}
	defenderReports, _, err := repo.ListReports(defender.Player.ID, 10, 0)
	if err != nil || len(defenderReports) == 0 || defenderReports[0].ID != battle.DefenderReportID {
		t.Fatalf("expected defender report, reports=%+v err=%v", defenderReports, err)
	}
	wantTimeline := []string{"meizhoulang_junlue", "huogong"}
	for _, report := range []BattleReport{attackerReports[0], defenderReports[0]} {
		if !reflect.DeepEqual(report.TraitTriggered, wantTimeline) {
			t.Fatalf("expected cross-phase trait timeline %v, report=%s timeline=%v", wantTimeline, report.ID, report.TraitTriggered)
		}
		if report.Detail == nil || len(report.Detail.Traits) != 2 || report.Detail.Traits[0].TraitID != wantTimeline[0] || report.Detail.Traits[1].TraitID != wantTimeline[1] {
			t.Fatalf("expected standard report to preserve pre-battle then after-combat order, report=%s detail=%+v", report.ID, report.Detail)
		}
		if !standardDetailHasGeneral(report.Detail, "zhouyu") {
			t.Fatalf("expected both views to retain Zhou Yu snapshot, report=%s detail=%+v", report.ID, report.Detail)
		}
	}

	attackOutcome := attackerReports[0].TraitOutcomes["meizhoulang_junlue"]
	attackModified, ok := attackOutcome.Detail["attackModifiedUnits"].(map[string]int)
	if !ok || attackModified["wuInfantry"] != 1 || attackOutcome.Detail["attackBonusRate"] != 0.05 {
		t.Fatalf("expected actual +1 attack and 5%% design rate, outcome=%+v", attackOutcome)
	}
	fireOutcome := attackerReports[0].TraitOutcomes["huogong"]
	fireExtra, ok := fireOutcome.Detail["extraDamage"].(int)
	if !ok || fireExtra != 250 {
		t.Fatalf("expected fire to add 250 losses from original 1000 defenders, outcome=%+v", fireOutcome)
	}
	byUnit, ok := fireOutcome.Detail["targetExtraLosses"].(map[string]int)
	if !ok || byUnit["shuInfantry"] != fireExtra {
		t.Fatalf("expected fire per-unit losses to equal total extra damage, outcome=%+v", fireOutcome)
	}
	totalLoss := attackerReports[0].DefenderLostUnits["shuInfantry"]
	if baseLoss := totalLoss - fireExtra; baseLoss <= 0 {
		t.Fatalf("expected final loss to contain positive base combat loss plus fire, total=%d fire=%d", totalLoss, fireExtra)
	}
	storedDefender, err := repo.GetState(defender.Player.ID)
	if err != nil {
		t.Fatalf("GetState defender failed: %v", err)
	}
	if got, want := armySliceToMap(storedDefender.Army)["shuInfantry"], 1000-totalLoss; got != want {
		t.Fatalf("expected stacked trait report and real defender army to reconcile, got=%d want=%d", got, want)
	}
}

// TestPvpSimaYiDefenseTraitsStackBeforeBattle 验证司马懿守城时真实战前伤亡与己方加防共同进入战力、经验、返程和双方战报。
func TestPvpSimaYiDefenseTraitsStackBeforeBattle(t *testing.T) {
	setTestFactionsAndGenerals(t, FactionsConfig{
		"shu": {Name: "蜀国", Generals: []GeneralInfo{{ID: "liubei", Name: "刘备"}}},
		"wei": {Name: "魏国", Generals: []GeneralInfo{{ID: "simayi", Name: "司马懿"}}},
	}, GeneralsConfig{Enabled: true, Heroes: map[string]GeneralHeroConfig{
		"liubei": {ID: "liubei", Name: "刘备", Faction: "shu", Enabled: true},
		"simayi": {
			ID: "simayi", Name: "司马懿", Faction: "wei", Enabled: true,
			SpecialTrait: GeneralTraitConfig{
				TraitID: "yibing_touxi", TraitType: general.TraitTypeSpecial, Enabled: true,
				Scope: "enemy_army", Params: map[string]float64{"triggerChance": 1, "effectRate": 0.35},
			},
			BonusTrait: GeneralTraitConfig{
				TraitID: "mouding_houfa", TraitType: general.TraitTypeBonus, Enabled: true,
				Scope: "self_army", AllowedSides: []string{"defender", "reinforcement"},
				Params: map[string]float64{"defenseBonusRate": 0.35, "triggerChance": 1},
			},
		},
	}})
	svc, repo, attacker, defender := newPvpTestServiceForGenerals(t, "shu", "liubei", "wei", "simayi")
	attacker.Army = []ArmyUnit{{UnitType: "shuInfantry", Amount: 1000}}
	defender.Army = []ArmyUnit{{UnitType: "weiInfantry", Amount: 100}}
	defender.Buildings = nil
	repo.players[attacker.Player.ID] = attacker
	repo.players[defender.Player.ID] = defender

	started, err := svc.StartPvpAttack(PvpAttackRequest{
		PlayerID: attacker.Player.ID, TargetPlayerID: defender.Player.ID, MarchMode: PvpMarchTypeAttack,
		Troops: map[string]int{"shuInfantry": 1000}, GeneralIDs: []string{"liubei"},
	})
	if err != nil {
		t.Fatalf("StartPvpAttack failed: %v", err)
	}
	forcePvpMarchDue(t, repo, started.March.ID)
	battle, err := svc.ResolvePvpMarch(started.March.ID)
	if err != nil {
		t.Fatalf("ResolvePvpMarch failed: %v", err)
	}
	attackPower, attackOK := battle.Result["attackerPower"].(float64)
	defensePower, defenseOK := battle.Result["defensePower"].(float64)
	if !attackOK || !defenseOK || attackPower != 6500 || defensePower != 1400 {
		t.Fatalf("expected 350 direct losses then 6500 attack against 35%% boosted 1400 defense, result=%+v", battle.Result)
	}

	attackerReports, _, err := repo.ListReports(attacker.Player.ID, 10, 0)
	if err != nil || len(attackerReports) == 0 || attackerReports[0].ID != battle.AttackerReportID {
		t.Fatalf("expected attacker report, reports=%+v err=%v", attackerReports, err)
	}
	defenderReports, _, err := repo.ListReports(defender.Player.ID, 10, 0)
	if err != nil || len(defenderReports) == 0 || defenderReports[0].ID != battle.DefenderReportID {
		t.Fatalf("expected defender report, reports=%+v err=%v", defenderReports, err)
	}
	for _, report := range []BattleReport{attackerReports[0], defenderReports[0]} {
		preDamage, ok := report.TraitOutcomes["yibing_touxi"].Detail["preBattleAffected"].(map[string]int)
		if !ok || preDamage["shuInfantry"] != 350 {
			t.Fatalf("expected both reports to record 350 real pre-battle losses, report=%s outcome=%+v", report.ID, report.TraitOutcomes["yibing_touxi"])
		}
		infantry, infantryOK := report.TraitOutcomes["mouding_houfa"].Detail["infantryDefenseModifiedUnits"].(map[string]int)
		cavalry, cavalryOK := report.TraitOutcomes["mouding_houfa"].Detail["cavalryDefenseModifiedUnits"].(map[string]int)
		if !infantryOK || !cavalryOK || infantry["weiInfantry"] != 4 || cavalry["weiInfantry"] != 3 || report.TraitOutcomes["mouding_houfa"].Detail["defenseBonusRate"] != 0.35 {
			t.Fatalf("expected both reports to record actual +4/+3 defense and 35%% design rate, report=%s outcome=%+v", report.ID, report.TraitOutcomes["mouding_houfa"])
		}
		if report.Detail == nil || len(report.Detail.Traits) != 2 || !standardReportHasTrait(report.Detail, "yibing_touxi") || !standardReportHasTrait(report.Detail, "mouding_houfa") {
			t.Fatalf("expected both pre-battle traits in standard timeline, report=%s detail=%+v", report.ID, report.Detail)
		}
		for _, trait := range report.Detail.Traits {
			if trait.OwnerSide != "secondary" || trait.GeneralID != "simayi" {
				t.Fatalf("expected Sima Yi defender ownership in both views, report=%s trait=%+v", report.ID, trait)
			}
		}
	}

	attackerLoss := attackerReports[0].LostUnits["shuInfantry"]
	if attackerLoss <= 350 || attackerLoss >= 1000 {
		t.Fatalf("expected final attacker loss to include 350 pre-damage plus bounded core loss, got %d", attackerLoss)
	}
	storedMarch, err := repo.GetPvpMarch(started.March.ID)
	if err != nil {
		t.Fatalf("GetPvpMarch failed: %v", err)
	}
	if storedMarch.Status != PvpMarchStatusReturning || storedMarch.AttackTroops["shuInfantry"] != 1000-attackerLoss {
		t.Fatalf("expected returning troops to match report loss, march=%+v loss=%d", storedMarch, attackerLoss)
	}
	if defenderReports[0].GeneralExpGained != attackerLoss {
		t.Fatalf("expected Sima Yi exp to include all real attacker losses, exp=%d loss=%d", defenderReports[0].GeneralExpGained, attackerLoss)
	}
	storedDefender, err := repo.GetState(defender.Player.ID)
	if err != nil {
		t.Fatalf("GetState defender failed: %v", err)
	}
	if got := pvpTestGeneralExp(storedDefender, "simayi"); got != defenderReports[0].GeneralExpGained {
		t.Fatalf("expected stored Sima Yi exp %d to match report, got %d", defenderReports[0].GeneralExpGained, got)
	}
	if got, want := armySliceToMap(storedDefender.Army)["weiInfantry"], 100-defenderReports[0].LostUnits["weiInfantry"]; got != want {
		t.Fatalf("expected defender inventory and report loss to reconcile, got=%d want=%d", got, want)
	}
}

// TestPvpGuanYuAttackTraitsStackBeforeBattle 验证关羽主动进攻时水淹真实伤亡与武圣加攻共同进入战力、经验、返程和双方战报。
func TestPvpGuanYuAttackTraitsStackBeforeBattle(t *testing.T) {
	setTestFactionsAndGenerals(t, FactionsConfig{
		"shu": {Name: "蜀国", Generals: []GeneralInfo{{ID: "guanyu", Name: "关羽"}}},
		"wei": {Name: "魏国", Generals: []GeneralInfo{{ID: "caocao", Name: "曹操"}}},
	}, GeneralsConfig{Enabled: true, Heroes: map[string]GeneralHeroConfig{
		"guanyu": {
			ID: "guanyu", Name: "关羽", Faction: "shu", Enabled: true,
			SpecialTrait: GeneralTraitConfig{
				TraitID: "shuiyan_qijun", TraitType: general.TraitTypeSpecial, Enabled: true,
				Scope:  "enemy_army",
				Params: map[string]float64{"triggerChance": 1, "effectRate": 0.35, "maxAffectedRate": 0.35},
			},
			BonusTrait: GeneralTraitConfig{
				TraitID: "wusheng_pojun", TraitType: general.TraitTypeBonus, Enabled: true,
				Scope: "self_army", AllowedSides: []string{"attacker"},
				Params: map[string]float64{"attackBonusRate": 0.2},
			},
		},
		"caocao": {ID: "caocao", Name: "曹操", Faction: "wei", Enabled: true},
	}})
	svc, repo, attacker, defender := newPvpTestServiceForGenerals(t, "shu", "guanyu", "wei", "caocao")
	attacker.Army = []ArmyUnit{{UnitType: "shuInfantry", Amount: 1000}}
	defender.Army = []ArmyUnit{{UnitType: "weiInfantry", Amount: 1000}}
	repo.players[attacker.Player.ID] = attacker
	repo.players[defender.Player.ID] = defender

	started, err := svc.StartPvpAttack(PvpAttackRequest{
		PlayerID: attacker.Player.ID, TargetPlayerID: defender.Player.ID, MarchMode: PvpMarchTypeAttack,
		Troops: map[string]int{"shuInfantry": 1000}, GeneralIDs: []string{"guanyu"},
	})
	if err != nil {
		t.Fatalf("StartPvpAttack failed: %v", err)
	}
	forcePvpMarchDue(t, repo, started.March.ID)
	battle, err := svc.ResolvePvpMarch(started.March.ID)
	if err != nil {
		t.Fatalf("ResolvePvpMarch failed: %v", err)
	}
	attackPower, attackPowerOK := battle.Result["attackerPower"].(float64)
	defensePower, defensePowerOK := battle.Result["defensePower"].(float64)
	if !attackPowerOK || !defensePowerOK || attackPower != 12000 || defensePower != 6695 {
		t.Fatalf("expected 1000 attackers at 12 attack and 650 defenders at 10 defense plus 3%% wall, got %v/%v", battle.Result["attackerPower"], battle.Result["defensePower"])
	}

	attackerReports, _, err := repo.ListReports(attacker.Player.ID, 10, 0)
	if err != nil || len(attackerReports) == 0 || attackerReports[0].ID != battle.AttackerReportID {
		t.Fatalf("expected attacker report, reports=%+v err=%v", attackerReports, err)
	}
	defenderReports, _, err := repo.ListReports(defender.Player.ID, 10, 0)
	if err != nil || len(defenderReports) == 0 || defenderReports[0].ID != battle.DefenderReportID {
		t.Fatalf("expected defender report, reports=%+v err=%v", defenderReports, err)
	}
	wantTimeline := []string{"shuiyan_qijun", "wusheng_pojun"}
	for _, report := range []BattleReport{attackerReports[0], defenderReports[0]} {
		if !reflect.DeepEqual(report.TraitTriggered, wantTimeline) {
			t.Fatalf("expected Guan Yu pre-battle timeline %v, report=%s timeline=%v", wantTimeline, report.ID, report.TraitTriggered)
		}
		preDamage, ok := report.TraitOutcomes["shuiyan_qijun"].Detail["preBattleAffected"].(map[string]int)
		if !ok || preDamage["weiInfantry"] != 350 {
			t.Fatalf("expected both reports to record 350 real pre-battle losses, report=%s outcome=%+v", report.ID, report.TraitOutcomes["shuiyan_qijun"])
		}
		attackModified, ok := report.TraitOutcomes["wusheng_pojun"].Detail["attackModifiedUnits"].(map[string]int)
		if !ok || attackModified["shuInfantry"] != 2 || report.TraitOutcomes["wusheng_pojun"].Detail["attackBonusRate"] != 0.2 {
			t.Fatalf("expected both reports to record actual +2 attack and 20%% design rate, report=%s outcome=%+v", report.ID, report.TraitOutcomes["wusheng_pojun"])
		}
		if report.Detail == nil || len(report.Detail.Traits) != 2 || report.Detail.Traits[0].TraitID != wantTimeline[0] || report.Detail.Traits[1].TraitID != wantTimeline[1] {
			t.Fatalf("expected standard report to preserve both Guan Yu pre-battle traits, report=%s detail=%+v", report.ID, report.Detail)
		}
		for _, trait := range report.Detail.Traits {
			if trait.OwnerSide != "primary" || trait.GeneralID != "guanyu" {
				t.Fatalf("expected Guan Yu attacker ownership in both views, report=%s trait=%+v", report.ID, trait)
			}
		}
		if !standardDetailHasGeneral(report.Detail, "guanyu") {
			t.Fatalf("expected both views to retain Guan Yu snapshot, report=%s detail=%+v", report.ID, report.Detail)
		}
	}

	attackerReport := attackerReports[0]
	defenderLoss := attackerReport.DefenderLostUnits["weiInfantry"]
	if defenderLoss != 1000 {
		t.Fatalf("expected 350 pre-damage plus 650 core losses to eliminate all defenders, got %d", defenderLoss)
	}
	storedDefender, err := repo.GetState(defender.Player.ID)
	if err != nil {
		t.Fatalf("GetState defender failed: %v", err)
	}
	if got, want := armySliceToMap(storedDefender.Army)["weiInfantry"], 1000-defenderLoss; got != want {
		t.Fatalf("expected defender inventory and report loss to reconcile, got=%d want=%d", got, want)
	}
	storedMarch, err := repo.GetPvpMarch(started.March.ID)
	if err != nil {
		t.Fatalf("GetPvpMarch failed: %v", err)
	}
	attackerLoss := attackerReport.LostUnits["shuInfantry"]
	if attackerLoss != 436 {
		t.Fatalf("expected power ratio to cause 436 attacker losses, got %d", attackerLoss)
	}
	if storedMarch.Status != PvpMarchStatusReturning || storedMarch.AttackTroops["shuInfantry"] != 1000-attackerLoss {
		t.Fatalf("expected returning troops to match report loss, march=%+v loss=%d", storedMarch, attackerLoss)
	}
	storedAttacker, err := repo.GetState(attacker.Player.ID)
	if err != nil {
		t.Fatalf("GetState attacker failed: %v", err)
	}
	if got := pvpTestGeneralExp(storedAttacker, "guanyu"); got != attackerReport.GeneralExpGained || got != defenderLoss {
		t.Fatalf("expected Guan Yu exp to equal real defender losses, stored=%d report=%d losses=%d", got, attackerReport.GeneralExpGained, defenderLoss)
	}
	for _, report := range []BattleReport{attackerReports[0], defenderReports[0]} {
		if report.Detail == nil || report.Detail.SecondarySide == nil || report.Detail.PrimarySide.Role != "attacker" || report.Detail.SecondarySide.Role != "defender" {
			t.Fatalf("expected objective attacker and defender sides in both standard reports, report=%s detail=%+v", report.ID, report.Detail)
		}
		assertStandardUnitRow(t, report.ID, report.Detail.PrimarySide, "shuInfantry", 1000, 436, 564)
		assertStandardUnitRow(t, report.ID, *report.Detail.SecondarySide, "weiInfantry", 1000, 1000, 0)
	}
}

// TestPvpZhangFeiAttackTraitsKeepSuppressedTroops 验证张飞主动进攻时临时压制与步兵加攻共同生效，压制兵战后仍保留。
func TestPvpZhangFeiAttackTraitsKeepSuppressedTroops(t *testing.T) {
	setTestFactionsAndGenerals(t, FactionsConfig{
		"shu": {Name: "蜀国", Generals: []GeneralInfo{{ID: "zhangfei", Name: "张飞"}}},
		"wei": {Name: "魏国", Generals: []GeneralInfo{{ID: "caocao", Name: "曹操"}}},
	}, GeneralsConfig{Enabled: true, Heroes: map[string]GeneralHeroConfig{
		"zhangfei": {
			ID: "zhangfei", Name: "张飞", Faction: "shu", Enabled: true,
			SpecialTrait: GeneralTraitConfig{
				TraitID: "zhenhe_quanjun", TraitType: general.TraitTypeSpecial, Enabled: true,
				Scope:  "enemy_army",
				Params: map[string]float64{"triggerChance": 1, "effectRate": 0.5, "maxAffectedRate": 0.5},
			},
			BonusTrait: GeneralTraitConfig{
				TraitID: "wanren_nuhou", TraitType: general.TraitTypeBonus, Enabled: true,
				Scope: "self_army", TargetUnitType: "infantry", AllowedSides: []string{"attacker"},
				Params: map[string]float64{"attackBonusRate": 0.2},
			},
		},
		"caocao": {ID: "caocao", Name: "曹操", Faction: "wei", Enabled: true},
	}})
	svc, repo, attacker, defender := newPvpTestServiceForGenerals(t, "shu", "zhangfei", "wei", "caocao")
	attacker.Army = []ArmyUnit{{UnitType: "shuInfantry", Amount: 1000}}
	defender.Army = []ArmyUnit{{UnitType: "weiInfantry", Amount: 1000}}
	repo.players[attacker.Player.ID] = attacker
	repo.players[defender.Player.ID] = defender

	started, err := svc.StartPvpAttack(PvpAttackRequest{
		PlayerID: attacker.Player.ID, TargetPlayerID: defender.Player.ID, MarchMode: PvpMarchTypeAttack,
		Troops: map[string]int{"shuInfantry": 1000}, GeneralIDs: []string{"zhangfei"},
	})
	if err != nil {
		t.Fatalf("StartPvpAttack failed: %v", err)
	}
	forcePvpMarchDue(t, repo, started.March.ID)
	battle, err := svc.ResolvePvpMarch(started.March.ID)
	if err != nil {
		t.Fatalf("ResolvePvpMarch failed: %v", err)
	}
	attackPower, attackPowerOK := battle.Result["attackerPower"].(float64)
	defensePower, defensePowerOK := battle.Result["defensePower"].(float64)
	if !attackPowerOK || !defensePowerOK || attackPower != 12000 || defensePower != 5150 {
		t.Fatalf("expected 1000 attackers at 12 attack and 500 active defenders plus 3%% wall, got %v/%v", battle.Result["attackerPower"], battle.Result["defensePower"])
	}

	attackerReports, _, err := repo.ListReports(attacker.Player.ID, 10, 0)
	if err != nil || len(attackerReports) == 0 || attackerReports[0].ID != battle.AttackerReportID {
		t.Fatalf("expected attacker report, reports=%+v err=%v", attackerReports, err)
	}
	defenderReports, _, err := repo.ListReports(defender.Player.ID, 10, 0)
	if err != nil || len(defenderReports) == 0 || defenderReports[0].ID != battle.DefenderReportID {
		t.Fatalf("expected defender report, reports=%+v err=%v", defenderReports, err)
	}
	wantTimeline := []string{"zhenhe_quanjun", "wanren_nuhou"}
	for _, report := range []BattleReport{attackerReports[0], defenderReports[0]} {
		if !reflect.DeepEqual(report.TraitTriggered, wantTimeline) {
			t.Fatalf("expected special then bonus trait timeline %v, report=%s timeline=%v", wantTimeline, report.ID, report.TraitTriggered)
		}
		suppressed, ok := report.TraitOutcomes["zhenhe_quanjun"].Detail["suppressedUnits"].(map[string]int)
		if !ok || suppressed["weiInfantry"] != 500 {
			t.Fatalf("expected both reports to record 500 temporary suppressions, report=%s outcome=%+v", report.ID, report.TraitOutcomes["zhenhe_quanjun"])
		}
		attackModified, ok := report.TraitOutcomes["wanren_nuhou"].Detail["attackModifiedUnits"].(map[string]int)
		if !ok || attackModified["shuInfantry"] != 2 || report.TraitOutcomes["wanren_nuhou"].Detail["attackBonusRate"] != 0.2 {
			t.Fatalf("expected both reports to record actual +2 attack and 20%% design rate, report=%s outcome=%+v", report.ID, report.TraitOutcomes["wanren_nuhou"])
		}
		if report.Detail == nil || len(report.Detail.Traits) != 2 || report.Detail.Traits[0].TraitID != wantTimeline[0] || report.Detail.Traits[1].TraitID != wantTimeline[1] {
			t.Fatalf("expected standard report to preserve special then bonus order, report=%s detail=%+v", report.ID, report.Detail)
		}
		for _, trait := range report.Detail.Traits {
			if trait.OwnerSide != "primary" || trait.GeneralID != "zhangfei" {
				t.Fatalf("expected Zhang Fei attacker ownership in both views, report=%s trait=%+v", report.ID, trait)
			}
		}
	}

	attackerReport := attackerReports[0]
	if got := attackerReport.DefenderLostUnits["weiInfantry"]; got != 500 {
		t.Fatalf("expected only 500 participating defenders to die, got %d", got)
	}
	if got := attackerReport.LostUnits["shuInfantry"]; got != 300 {
		t.Fatalf("expected power ratio to cause 300 attacker losses, got %d", got)
	}
	storedDefender, err := repo.GetState(defender.Player.ID)
	if err != nil {
		t.Fatalf("GetState defender failed: %v", err)
	}
	if got := armySliceToMap(storedDefender.Army)["weiInfantry"]; got != 500 {
		t.Fatalf("expected 500 suppressed defenders to remain after battle, got %d", got)
	}
	storedMarch, err := repo.GetPvpMarch(started.March.ID)
	if err != nil {
		t.Fatalf("GetPvpMarch failed: %v", err)
	}
	if storedMarch.Status != PvpMarchStatusReturning || storedMarch.AttackTroops["shuInfantry"] != 700 {
		t.Fatalf("expected 700 attackers to return, march=%+v", storedMarch)
	}
	storedAttacker, err := repo.GetState(attacker.Player.ID)
	if err != nil {
		t.Fatalf("GetState attacker failed: %v", err)
	}
	if got := pvpTestGeneralExp(storedAttacker, "zhangfei"); got != 500 || attackerReport.GeneralExpGained != 500 {
		t.Fatalf("expected Zhang Fei exp to count 500 real deaths only, stored=%d report=%d", got, attackerReport.GeneralExpGained)
	}
	for _, report := range []BattleReport{attackerReports[0], defenderReports[0]} {
		if report.Detail == nil || report.Detail.SecondarySide == nil {
			t.Fatalf("expected standard two-sided report, report=%s detail=%+v", report.ID, report.Detail)
		}
		assertStandardUnitRow(t, report.ID, report.Detail.PrimarySide, "shuInfantry", 1000, 300, 700)
		assertStandardUnitRow(t, report.ID, *report.Detail.SecondarySide, "weiInfantry", 1000, 500, 500)
	}
}

// TestPvpZhangLiaoCavalryTraitsKeepSuppressedTroops 验证张辽主动进攻时临时压制与骑兵加攻共同生效，并使用敌方骑防计算战力。
func TestPvpZhangLiaoCavalryTraitsKeepSuppressedTroops(t *testing.T) {
	setTestFactionsAndGenerals(t, FactionsConfig{
		"wei": {Name: "魏国", Generals: []GeneralInfo{{ID: "zhangliao", Name: "张辽"}}},
		"shu": {Name: "蜀国", Generals: []GeneralInfo{{ID: "liubei", Name: "刘备"}}},
	}, GeneralsConfig{Enabled: true, Heroes: map[string]GeneralHeroConfig{
		"zhangliao": {
			ID: "zhangliao", Name: "张辽", Faction: "wei", Enabled: true,
			SpecialTrait: GeneralTraitConfig{
				TraitID: "weizhen_zhenhe", TraitType: general.TraitTypeSpecial, Enabled: true,
				Scope: "enemy_army", AllowedSides: []string{"attacker"},
				Params: map[string]float64{"triggerChance": 1, "effectRate": 0.25},
			},
			BonusTrait: GeneralTraitConfig{
				TraitID: "weizhen_xiaoyao", TraitType: general.TraitTypeBonus, Enabled: true,
				Scope: "self_army", TargetUnitType: "cavalry", AllowedSides: []string{"attacker"},
				Params: map[string]float64{"triggerChance": 1, "attackBonusRate": 0.35},
			},
		},
		"liubei": {ID: "liubei", Name: "刘备", Faction: "shu", Enabled: true},
	}})
	svc, repo, attacker, defender := newPvpTestServiceForGenerals(t, "wei", "zhangliao", "shu", "liubei")
	attacker.Army = []ArmyUnit{{UnitType: "weiCavalry", Amount: 1000}}
	defender.Army = []ArmyUnit{{UnitType: "shuInfantry", Amount: 1000}}
	repo.players[attacker.Player.ID] = attacker
	repo.players[defender.Player.ID] = defender

	started, err := svc.StartPvpAttack(PvpAttackRequest{
		PlayerID: attacker.Player.ID, TargetPlayerID: defender.Player.ID, MarchMode: PvpMarchTypeAttack,
		Troops: map[string]int{"weiCavalry": 1000}, GeneralIDs: []string{"zhangliao"},
	})
	if err != nil {
		t.Fatalf("StartPvpAttack failed: %v", err)
	}
	forcePvpMarchDue(t, repo, started.March.ID)
	battle, err := svc.ResolvePvpMarch(started.March.ID)
	if err != nil {
		t.Fatalf("ResolvePvpMarch failed: %v", err)
	}
	attackPower, attackPowerOK := battle.Result["attackerPower"].(float64)
	defensePower, defensePowerOK := battle.Result["defensePower"].(float64)
	if !attackPowerOK || !defensePowerOK || attackPower != 19000 || defensePower != 6120 {
		t.Fatalf("expected 1000 cavalry at 19 attack and 750 active defenders at 8 cavalry defense plus 2%% Shu wall, got %v/%v", battle.Result["attackerPower"], battle.Result["defensePower"])
	}

	attackerReports, _, err := repo.ListReports(attacker.Player.ID, 10, 0)
	if err != nil || len(attackerReports) == 0 || attackerReports[0].ID != battle.AttackerReportID {
		t.Fatalf("expected attacker report, reports=%+v err=%v", attackerReports, err)
	}
	defenderReports, _, err := repo.ListReports(defender.Player.ID, 10, 0)
	if err != nil || len(defenderReports) == 0 || defenderReports[0].ID != battle.DefenderReportID {
		t.Fatalf("expected defender report, reports=%+v err=%v", defenderReports, err)
	}
	wantTimeline := []string{"weizhen_zhenhe", "weizhen_xiaoyao"}
	for _, report := range []BattleReport{attackerReports[0], defenderReports[0]} {
		if !reflect.DeepEqual(report.TraitTriggered, wantTimeline) {
			t.Fatalf("expected Zhang Liao special then bonus timeline %v, report=%s timeline=%v", wantTimeline, report.ID, report.TraitTriggered)
		}
		suppressed, ok := report.TraitOutcomes["weizhen_zhenhe"].Detail["suppressedUnits"].(map[string]int)
		if !ok || suppressed["shuInfantry"] != 250 {
			t.Fatalf("expected both reports to record 250 temporary suppressions, report=%s outcome=%+v", report.ID, report.TraitOutcomes["weizhen_zhenhe"])
		}
		fled, fledOK := report.TraitOutcomes["weizhen_zhenhe"].Detail["fledUnits"].(map[string]int)
		returned, returnedOK := report.TraitOutcomes["weizhen_zhenhe"].Detail["returnedUnits"].(map[string]int)
		if !fledOK || !returnedOK || fled["shuInfantry"] != 250 || returned["shuInfantry"] != 250 {
			t.Fatalf("expected battle report to record 250 fled and returned troops, report=%s outcome=%+v", report.ID, report.TraitOutcomes["weizhen_zhenhe"])
		}
		attackModified, ok := report.TraitOutcomes["weizhen_xiaoyao"].Detail["attackModifiedUnits"].(map[string]int)
		if !ok || attackModified["weiCavalry"] != 5 || report.TraitOutcomes["weizhen_xiaoyao"].Detail["attackBonusRate"] != 0.35 {
			t.Fatalf("expected both reports to record actual +5 cavalry attack and 35%% design rate, report=%s outcome=%+v", report.ID, report.TraitOutcomes["weizhen_xiaoyao"])
		}
		if report.Detail == nil || len(report.Detail.Traits) != 2 || report.Detail.Traits[0].TraitID != wantTimeline[0] || report.Detail.Traits[1].TraitID != wantTimeline[1] {
			t.Fatalf("expected standard report to preserve Zhang Liao trait order, report=%s detail=%+v", report.ID, report.Detail)
		}
		for _, trait := range report.Detail.Traits {
			if trait.OwnerSide != "primary" || trait.GeneralID != "zhangliao" {
				t.Fatalf("expected Zhang Liao attacker ownership in both views, report=%s trait=%+v", report.ID, trait)
			}
		}
	}

	attackerReport := attackerReports[0]
	if got := attackerReport.DefenderLostUnits["shuInfantry"]; got != 750 {
		t.Fatalf("expected only 750 participating defenders to die, got %d", got)
	}
	if got := attackerReport.LostUnits["weiCavalry"]; got != 199 {
		t.Fatalf("expected power ratio to cause 199 cavalry losses, got %d", got)
	}
	storedDefender, err := repo.GetState(defender.Player.ID)
	if err != nil {
		t.Fatalf("GetState defender failed: %v", err)
	}
	if got := armySliceToMap(storedDefender.Army)["shuInfantry"]; got != 250 {
		t.Fatalf("expected 250 fled defenders to return after battle, got %d", got)
	}
	storedMarch, err := repo.GetPvpMarch(started.March.ID)
	if err != nil {
		t.Fatalf("GetPvpMarch failed: %v", err)
	}
	if storedMarch.Status != PvpMarchStatusReturning || storedMarch.AttackTroops["weiCavalry"] != 801 {
		t.Fatalf("expected 801 cavalry to return, march=%+v", storedMarch)
	}
	storedAttacker, err := repo.GetState(attacker.Player.ID)
	if err != nil {
		t.Fatalf("GetState attacker failed: %v", err)
	}
	if got := pvpTestGeneralExp(storedAttacker, "zhangliao"); got != 750 || attackerReport.GeneralExpGained != 750 {
		t.Fatalf("expected Zhang Liao exp to count 750 real deaths only, stored=%d report=%d", got, attackerReport.GeneralExpGained)
	}
	for _, report := range []BattleReport{attackerReports[0], defenderReports[0]} {
		if report.Detail == nil || report.Detail.SecondarySide == nil {
			t.Fatalf("expected standard two-sided report, report=%s detail=%+v", report.ID, report.Detail)
		}
		attackerUnit := BattleReportUnit{}
		defenderUnit := BattleReportUnit{}
		for _, unit := range report.Detail.PrimarySide.Units {
			if unit.UnitType == "weiCavalry" {
				attackerUnit = unit
				break
			}
		}
		for _, unit := range report.Detail.SecondarySide.Units {
			if unit.UnitType == "shuInfantry" {
				defenderUnit = unit
				break
			}
		}
		if attackerUnit.UnitType != "weiCavalry" || attackerUnit.AmountBefore != 1000 || attackerUnit.Lost != 199 || attackerUnit.Survived != 801 {
			t.Fatalf("expected standard attacker row 1000/199/801, report=%s unit=%+v", report.ID, attackerUnit)
		}
		if defenderUnit.UnitType != "shuInfantry" || defenderUnit.AmountBefore != 1000 || defenderUnit.Lost != 750 || defenderUnit.Survived != 250 {
			t.Fatalf("expected standard defender row 1000/750/250, report=%s unit=%+v", report.ID, defenderUnit)
		}
	}
}

// TestPvpHuangZhongDefenseBreakAndExtraDamageReconcilePlunder 验证黄忠战前破防与战后追加伤害在掠夺战中分别记录基础战损和实际增量。
func TestPvpHuangZhongDefenseBreakAndExtraDamageReconcilePlunder(t *testing.T) {
	setTestFactionsAndGenerals(t, FactionsConfig{
		"shu": {Name: "蜀国", Generals: []GeneralInfo{{ID: "huangzhong", Name: "黄忠"}}},
		"wei": {Name: "魏国", Generals: []GeneralInfo{{ID: "caocao", Name: "曹操"}}},
	}, GeneralsConfig{Enabled: true, Heroes: map[string]GeneralHeroConfig{
		"huangzhong": {
			ID: "huangzhong", Name: "黄忠", Faction: "shu", Enabled: true,
			SpecialTrait: GeneralTraitConfig{
				TraitID: "baibu_chuanyang", TraitType: general.TraitTypeSpecial, Enabled: true,
				Scope: "enemy_army", AllowedSides: []string{"attacker"},
				Params: map[string]float64{"triggerChance": 1, "enemyDefenseReductionRate": 0.2},
			},
			BonusTrait: GeneralTraitConfig{
				TraitID: "laodang_yizhuang", TraitType: general.TraitTypeBonus, Enabled: true,
				Scope: "enemy_army", Params: map[string]float64{"effectRate": 0.1},
			},
		},
		"caocao": {ID: "caocao", Name: "曹操", Faction: "wei", Enabled: true},
	}})
	svc, repo, attacker, defender := newPvpTestServiceForGenerals(t, "shu", "huangzhong", "wei", "caocao")
	attacker.Army = []ArmyUnit{{UnitType: "shuInfantry", Amount: 1000}}
	attacker.Resources.Items = map[string]int{"wood": 0, "stone": 0, "iron": 0, "food": 0}
	attacker.Resources.Capacity = map[string]int{"wood": 100000, "stone": 100000, "iron": 100000, "food": 100000}
	defender.Army = []ArmyUnit{{UnitType: "weiInfantry", Amount: 1000}}
	defender.Buildings = nil
	defender.Resources.Items = map[string]int{"wood": 10000, "stone": 0, "iron": 0, "food": 0}
	defender.Resources.Capacity = map[string]int{"wood": 0, "stone": 0, "iron": 0, "food": 0}
	nowText := time.Now().UTC().Format(resourceDateLayout)
	attacker.ResourceSettledAt = nowText
	defender.ResourceSettledAt = nowText
	repo.players[attacker.Player.ID] = attacker
	repo.players[defender.Player.ID] = defender

	started, err := svc.StartPvpAttack(PvpAttackRequest{
		PlayerID: attacker.Player.ID, TargetPlayerID: defender.Player.ID, MarchMode: PvpMarchTypePlunder,
		Troops: map[string]int{"shuInfantry": 1000}, GeneralIDs: []string{"huangzhong"},
	})
	if err != nil {
		t.Fatalf("StartPvpAttack failed: %v", err)
	}
	forcePvpMarchDue(t, repo, started.March.ID)
	battle, err := svc.ResolvePvpMarch(started.March.ID)
	if err != nil {
		t.Fatalf("ResolvePvpMarch failed: %v", err)
	}
	attackPower, attackPowerOK := battle.Result["attackerPower"].(float64)
	defensePower, defensePowerOK := battle.Result["defensePower"].(float64)
	if !attackPowerOK || !defensePowerOK || attackPower != 10000 || defensePower != 8000 {
		t.Fatalf("expected 10000 attack power and defender infantry defense reduced 10 -> 8, got %v/%v", battle.Result["attackerPower"], battle.Result["defensePower"])
	}

	attackerReports, _, err := repo.ListReports(attacker.Player.ID, 10, 0)
	if err != nil || len(attackerReports) == 0 || attackerReports[0].ID != battle.AttackerReportID {
		t.Fatalf("expected attacker report, reports=%+v err=%v", attackerReports, err)
	}
	defenderReports, _, err := repo.ListReports(defender.Player.ID, 10, 0)
	if err != nil || len(defenderReports) == 0 || defenderReports[0].ID != battle.DefenderReportID {
		t.Fatalf("expected defender report, reports=%+v err=%v", defenderReports, err)
	}
	wantTimeline := []string{"baibu_chuanyang", "laodang_yizhuang"}
	for _, report := range []BattleReport{attackerReports[0], defenderReports[0]} {
		if !reflect.DeepEqual(report.TraitTriggered, wantTimeline) {
			t.Fatalf("expected before-battle defense break then after-combat damage timeline %v, report=%s timeline=%v", wantTimeline, report.ID, report.TraitTriggered)
		}
		breakOutcome := report.TraitOutcomes["baibu_chuanyang"]
		infantry, infantryOK := breakOutcome.Detail["infantryDefenseModifiedUnits"].(map[string]int)
		cavalry, cavalryOK := breakOutcome.Detail["cavalryDefenseModifiedUnits"].(map[string]int)
		if !infantryOK || !cavalryOK || breakOutcome.Detail["enemyDefenseReductionRate"] != 0.2 || infantry["weiInfantry"] != -2 || cavalry["weiInfantry"] != -2 {
			t.Fatalf("expected both reports to record formal 20%% defense break and actual -2/-2, report=%s outcome=%+v", report.ID, breakOutcome)
		}
		extra, ok := report.TraitOutcomes["laodang_yizhuang"].Detail["extraLosses"].(map[string]int)
		if !ok || extra["weiInfantry"] != 100 || report.TraitOutcomes["laodang_yizhuang"].Detail["effectRate"] != 0.1 {
			t.Fatalf("expected both reports to record 100 actual extra losses and 10%% design rate, report=%s outcome=%+v", report.ID, report.TraitOutcomes["laodang_yizhuang"])
		}
		if report.Detail == nil || len(report.Detail.Traits) != 2 || report.Detail.Traits[0].TraitID != wantTimeline[0] || report.Detail.Traits[1].TraitID != wantTimeline[1] {
			t.Fatalf("expected standard report to preserve Huang Zhong cross-phase order, report=%s detail=%+v", report.ID, report.Detail)
		}
	}

	attackerReport := attackerReports[0]
	if attackerReport.OwnerOutcome != ReportOwnerOutcomeVictory || attackerReport.LostUnits["shuInfantry"] != 421 || attackerReport.DefenderLostUnits["weiInfantry"] != 678 {
		t.Fatalf("expected plunder victory with attacker loss 421 and defender total loss 678, report=%+v", attackerReport)
	}
	if battle.Plunder["wood"] <= 0 || attackerReport.Rewards["wood"] != battle.Plunder["wood"] || attackerReport.Detail.Rewards.Resources["wood"] != battle.Plunder["wood"] {
		t.Fatalf("expected battle, legacy report and standard report to share the final wood plunder, battle=%+v legacy=%+v standard=%+v", battle.Plunder, attackerReport.Rewards, attackerReport.Detail.Rewards.Resources)
	}
	extra := attackerReport.TraitOutcomes["laodang_yizhuang"].Detail["extraLosses"].(map[string]int)["weiInfantry"]
	if baseLoss := attackerReport.DefenderLostUnits["weiInfantry"] - extra; baseLoss != 578 {
		t.Fatalf("expected core base loss 578 plus exactly 100 extra loss, total=%d extra=%d", attackerReport.DefenderLostUnits["weiInfantry"], extra)
	}
	storedDefender, err := repo.GetState(defender.Player.ID)
	if err != nil || armySliceToMap(storedDefender.Army)["weiInfantry"] != 322 {
		t.Fatalf("expected defender authoritative army 322, state=%+v err=%v", storedDefender.Army, err)
	}
	storedMarch, err := repo.GetPvpMarch(started.March.ID)
	if err != nil || storedMarch.Status != PvpMarchStatusReturning || storedMarch.AttackTroops["shuInfantry"] != 579 {
		t.Fatalf("expected 579 attackers to return, march=%+v err=%v", storedMarch, err)
	}
	storedAttacker, err := repo.GetState(attacker.Player.ID)
	if err != nil || pvpTestGeneralExp(storedAttacker, "huangzhong") != 678 || attackerReport.GeneralExpGained != 678 {
		t.Fatalf("expected Huang Zhong exp to include base and extra real deaths, stored=%d report=%d err=%v", pvpTestGeneralExp(storedAttacker, "huangzhong"), attackerReport.GeneralExpGained, err)
	}
	if storedAttacker.Resources.Items["wood"] != battle.Plunder["wood"] || storedDefender.Resources.Items["wood"] != 10000-battle.Plunder["wood"] {
		t.Fatalf("expected final resources to match reported plunder, attacker=%d defender=%d plunder=%d", storedAttacker.Resources.Items["wood"], storedDefender.Resources.Items["wood"], battle.Plunder["wood"])
	}
	for _, report := range []BattleReport{attackerReports[0], defenderReports[0]} {
		if report.Detail == nil || report.Detail.SecondarySide == nil {
			t.Fatalf("expected standard two-sided report, report=%s detail=%+v", report.ID, report.Detail)
		}
		assertStandardUnitRow(t, report.ID, report.Detail.PrimarySide, "shuInfantry", 1000, 421, 579)
		assertStandardUnitRow(t, report.ID, *report.Detail.SecondarySide, "weiInfantry", 1000, 678, 322)
	}
}

// TestPvpSunCeCavalryAndPursuitTraitsReconcilePlunder 验证孙策只强化霸王骑，并在加攻后的掠夺胜利结算中追加真实追击损失。
func TestPvpSunCeCavalryAndPursuitTraitsReconcilePlunder(t *testing.T) {
	setTestFactionsAndGenerals(t, FactionsConfig{
		"wu":  {Name: "吴国", Generals: []GeneralInfo{{ID: "sunce", Name: "孙策"}}},
		"wei": {Name: "魏国", Generals: []GeneralInfo{{ID: "caocao", Name: "曹操"}}},
	}, GeneralsConfig{Enabled: true, Heroes: map[string]GeneralHeroConfig{
		"sunce": {
			ID: "sunce", Name: "孙策", Faction: "wu", Enabled: true,
			SpecialTrait: GeneralTraitConfig{
				TraitID: "xiaobawang_zhuiji", TraitType: general.TraitTypeSpecial, Enabled: true,
				Scope: "enemy_army", AllowedScenes: []string{"plunder"}, RequiredOutcome: "win",
				Params: map[string]float64{"triggerChance": 1, "effectRate": 0.1},
			},
			BonusTrait: GeneralTraitConfig{
				TraitID: "xiaobawang_tieqi", TraitType: general.TraitTypeBonus, Enabled: true,
				Scope: "self_army", TargetUnitType: "overlordRider", AllowedSides: []string{"attacker"},
				Params: map[string]float64{"unitAttackFlat": 50},
			},
		},
		"caocao": {ID: "caocao", Name: "曹操", Faction: "wei", Enabled: true},
	}})
	svc, repo, attacker, defender := newPvpTestServiceForGenerals(t, "wu", "sunce", "wei", "caocao")
	unitsMu.Lock()
	activeUnits["wu"]["overlordRider"] = UnitConfig{
		Name: "霸王骑", Category: "cavalry",
		Stats: map[string]int{"attack": 28, "infantryDefense": 10, "cavalryDefense": 33, "carryCapacity": 130, "upkeep": 4},
	}
	unitsMu.Unlock()
	attacker.Army = []ArmyUnit{{UnitType: "overlordRider", Amount: 200}}
	attacker.Resources.Items = map[string]int{"wood": 0, "stone": 0, "iron": 0, "food": 0}
	attacker.Resources.Capacity = map[string]int{"wood": 100000, "stone": 100000, "iron": 100000, "food": 100000}
	defender.Army = []ArmyUnit{{UnitType: "weiInfantry", Amount: 1000}}
	defender.Buildings = nil
	defender.Resources.Items = map[string]int{"wood": 10000, "stone": 0, "iron": 0, "food": 0}
	defender.Resources.Capacity = map[string]int{"wood": 0, "stone": 0, "iron": 0, "food": 0}
	nowText := time.Now().UTC().Format(resourceDateLayout)
	attacker.ResourceSettledAt = nowText
	defender.ResourceSettledAt = nowText
	repo.players[attacker.Player.ID] = attacker
	repo.players[defender.Player.ID] = defender

	started, err := svc.StartPvpAttack(PvpAttackRequest{
		PlayerID: attacker.Player.ID, TargetPlayerID: defender.Player.ID, MarchMode: PvpMarchTypePlunder,
		Troops: map[string]int{"overlordRider": 200}, GeneralIDs: []string{"sunce"},
	})
	if err != nil {
		t.Fatalf("StartPvpAttack failed: %v", err)
	}
	forcePvpMarchDue(t, repo, started.March.ID)
	battle, err := svc.ResolvePvpMarch(started.March.ID)
	if err != nil {
		t.Fatalf("ResolvePvpMarch failed: %v", err)
	}
	attackPower, attackPowerOK := battle.Result["attackerPower"].(float64)
	defensePower, defensePowerOK := battle.Result["defensePower"].(float64)
	if !attackPowerOK || !defensePowerOK || attackPower != 15600 || defensePower != 8000 {
		t.Fatalf("expected 200 overlord riders at 28+50 attack against cavalry defense 8, got %v/%v", battle.Result["attackerPower"], battle.Result["defensePower"])
	}

	attackerReports, _, err := repo.ListReports(attacker.Player.ID, 10, 0)
	if err != nil || len(attackerReports) == 0 || attackerReports[0].ID != battle.AttackerReportID {
		t.Fatalf("expected attacker report, reports=%+v err=%v", attackerReports, err)
	}
	defenderReports, _, err := repo.ListReports(defender.Player.ID, 10, 0)
	if err != nil || len(defenderReports) == 0 || defenderReports[0].ID != battle.DefenderReportID {
		t.Fatalf("expected defender report, reports=%+v err=%v", defenderReports, err)
	}
	wantTimeline := []string{"xiaobawang_tieqi", "xiaobawang_zhuiji"}
	for _, report := range []BattleReport{attackerReports[0], defenderReports[0]} {
		if !reflect.DeepEqual(report.TraitTriggered, wantTimeline) {
			t.Fatalf("expected before-battle cavalry bonus then victory pursuit timeline %v, report=%s timeline=%v", wantTimeline, report.ID, report.TraitTriggered)
		}
		attackOutcome := report.TraitOutcomes["xiaobawang_tieqi"]
		attackModified, ok := attackOutcome.Detail["attackModifiedUnits"].(map[string]int)
		unitAttackFlat, designOK := attackOutcome.Detail["unitAttackFlat"].(float64)
		if !ok || !designOK || attackModified["overlordRider"] != 50 || len(attackModified) != 1 || unitAttackFlat != 50 {
			t.Fatalf("expected both reports to record only overlord rider +50 attack, report=%s outcome=%+v", report.ID, report.TraitOutcomes["xiaobawang_tieqi"])
		}
		extra, ok := report.TraitOutcomes["xiaobawang_zhuiji"].Detail["extraLosses"].(map[string]int)
		if !ok || extra["weiInfantry"] != 100 || report.TraitOutcomes["xiaobawang_zhuiji"].Detail["effectRate"] != 0.1 || report.TraitOutcomes["xiaobawang_zhuiji"].Detail["triggerChance"] != float64(1) {
			t.Fatalf("expected both reports to record 100 actual pursuit losses with formal values, report=%s outcome=%+v", report.ID, report.TraitOutcomes["xiaobawang_zhuiji"])
		}
		if report.Detail == nil || len(report.Detail.Traits) != 2 || report.Detail.Traits[0].TraitID != wantTimeline[0] || report.Detail.Traits[1].TraitID != wantTimeline[1] {
			t.Fatalf("expected standard report to preserve Sun Ce cross-phase order, report=%s detail=%+v", report.ID, report.Detail)
		}
	}

	attackerReport := attackerReports[0]
	if attackerReport.OwnerOutcome != ReportOwnerOutcomeVictory || attackerReport.LostUnits["overlordRider"] != 55 || attackerReport.DefenderLostUnits["weiInfantry"] != 821 {
		t.Fatalf("expected plunder victory with attacker loss 55 and defender total loss 821, report=%+v", attackerReport)
	}
	extra := attackerReport.TraitOutcomes["xiaobawang_zhuiji"].Detail["extraLosses"].(map[string]int)["weiInfantry"]
	if baseLoss := attackerReport.DefenderLostUnits["weiInfantry"] - extra; baseLoss != 721 {
		t.Fatalf("expected core base loss 721 plus exactly 100 pursuit loss, total=%d extra=%d", attackerReport.DefenderLostUnits["weiInfantry"], extra)
	}
	if battle.Plunder["wood"] <= 0 || attackerReport.Rewards["wood"] != battle.Plunder["wood"] || attackerReport.Detail.Rewards.Resources["wood"] != battle.Plunder["wood"] {
		t.Fatalf("expected battle and both report formats to share final wood plunder, battle=%+v legacy=%+v standard=%+v", battle.Plunder, attackerReport.Rewards, attackerReport.Detail.Rewards.Resources)
	}
	storedDefender, err := repo.GetState(defender.Player.ID)
	if err != nil || armySliceToMap(storedDefender.Army)["weiInfantry"] != 179 {
		t.Fatalf("expected defender authoritative army 179, state=%+v err=%v", storedDefender.Army, err)
	}
	storedMarch, err := repo.GetPvpMarch(started.March.ID)
	if err != nil || storedMarch.Status != PvpMarchStatusReturning || storedMarch.AttackTroops["overlordRider"] != 145 {
		t.Fatalf("expected 145 overlord riders to return, march=%+v err=%v", storedMarch, err)
	}
	storedAttacker, err := repo.GetState(attacker.Player.ID)
	if err != nil || pvpTestGeneralExp(storedAttacker, "sunce") != 821 || attackerReport.GeneralExpGained != 821 {
		t.Fatalf("expected Sun Ce exp to include base and pursuit deaths, stored=%d report=%d err=%v", pvpTestGeneralExp(storedAttacker, "sunce"), attackerReport.GeneralExpGained, err)
	}
	if storedAttacker.Resources.Items["wood"] != battle.Plunder["wood"] || storedDefender.Resources.Items["wood"] != 10000-battle.Plunder["wood"] {
		t.Fatalf("expected final resources to match reported plunder, attacker=%d defender=%d plunder=%d", storedAttacker.Resources.Items["wood"], storedDefender.Resources.Items["wood"], battle.Plunder["wood"])
	}
	for _, report := range []BattleReport{attackerReports[0], defenderReports[0]} {
		if report.Detail == nil || report.Detail.SecondarySide == nil {
			t.Fatalf("expected standard two-sided report, report=%s detail=%+v", report.ID, report.Detail)
		}
		attackerUnit := BattleReportUnit{}
		for _, unit := range report.Detail.PrimarySide.Units {
			if unit.UnitType == "overlordRider" {
				attackerUnit = unit
				break
			}
		}
		defenderUnit := BattleReportUnit{}
		for _, unit := range report.Detail.SecondarySide.Units {
			if unit.UnitType == "weiInfantry" {
				defenderUnit = unit
				break
			}
		}
		if attackerUnit.AmountBefore != 200 || attackerUnit.Lost != 55 || attackerUnit.Survived != 145 {
			t.Fatalf("expected standard overlord rider row 200/55/145, report=%s unit=%+v", report.ID, attackerUnit)
		}
		if defenderUnit.AmountBefore != 1000 || defenderUnit.Lost != 821 || defenderUnit.Survived != 179 {
			t.Fatalf("expected standard defender row 1000/821/179, report=%s unit=%+v", report.ID, defenderUnit)
		}
	}
}

// TestPvpZhugeLiangSuppressionTraitsKeepSeparateSemantics 验证诸葛亮的临时兵力压制与战前全体特性封禁分别改变正确口径。
func TestPvpZhugeLiangSuppressionTraitsKeepSeparateSemantics(t *testing.T) {
	type runResult struct {
		battle           PvpBattle
		attackerReport   BattleReport
		defenderReport   BattleReport
		storedMarch      PvpMarch
		storedDefender   GameState
		attackerLoss     int
		defenderLoss     int
		defenderUnitType string
	}
	run := func(wolongChance float64) runResult {
		setTestFactionsAndGenerals(t, FactionsConfig{
			"shu": {Name: "蜀国", Generals: []GeneralInfo{{ID: "zhugeliang", Name: "诸葛亮"}}},
			"wei": {Name: "魏国", Generals: []GeneralInfo{{ID: "damage_general", Name: "增伤守将"}}},
		}, GeneralsConfig{Enabled: true, Heroes: map[string]GeneralHeroConfig{
			"zhugeliang": {
				ID: "zhugeliang", Name: "诸葛亮", Faction: "shu", Enabled: true,
				SpecialTrait: GeneralTraitConfig{
					TraitID: "qimen_dunjia", TraitType: general.TraitTypeSpecial, Enabled: true,
					Scope:  "enemy_army",
					Params: map[string]float64{"effectRate": 0.25, "maxAffectedRate": 0.25, "triggerChance": 1},
				},
				BonusTrait: GeneralTraitConfig{
					TraitID: "wolong_mouzhi", TraitType: general.TraitTypeBonus, Enabled: true,
					Scope:  "enemy_traits",
					Params: map[string]float64{"triggerChance": wolongChance},
				},
			},
			"damage_general": {
				ID: "damage_general", Name: "增伤守将", Faction: "wei", Enabled: true,
				BonusTrait: GeneralTraitConfig{
					TraitID: "laodang_yizhuang", TraitType: general.TraitTypeBonus, Enabled: true,
					Scope: "enemy_army", Params: map[string]float64{"effectRate": 0.1, "triggerChance": 1},
				},
			},
		}})
		svc, repo, attacker, defender := newPvpTestServiceForGenerals(t, "shu", "zhugeliang", "wei", "damage_general")
		attacker.Army = []ArmyUnit{{UnitType: "shuInfantry", Amount: 1000}}
		defender.Army = []ArmyUnit{{UnitType: "weiInfantry", Amount: 100}}
		defender.Buildings = nil
		repo.players[attacker.Player.ID] = attacker
		repo.players[defender.Player.ID] = defender

		started, err := svc.StartPvpAttack(PvpAttackRequest{
			PlayerID: attacker.Player.ID, TargetPlayerID: defender.Player.ID, MarchMode: PvpMarchTypeAttack,
			Troops: map[string]int{"shuInfantry": 1000}, GeneralIDs: []string{"zhugeliang"},
		})
		if err != nil {
			t.Fatalf("StartPvpAttack failed: %v", err)
		}
		forcePvpMarchDue(t, repo, started.March.ID)
		battle, err := svc.ResolvePvpMarch(started.March.ID)
		if err != nil {
			t.Fatalf("ResolvePvpMarch failed: %v", err)
		}
		attackerReports, _, err := repo.ListReports(attacker.Player.ID, 10, 0)
		if err != nil || len(attackerReports) == 0 || attackerReports[0].ID != battle.AttackerReportID {
			t.Fatalf("expected attacker report, reports=%+v err=%v", attackerReports, err)
		}
		defenderReports, _, err := repo.ListReports(defender.Player.ID, 10, 0)
		if err != nil || len(defenderReports) == 0 || defenderReports[0].ID != battle.DefenderReportID {
			t.Fatalf("expected defender report, reports=%+v err=%v", defenderReports, err)
		}
		storedMarch, err := repo.GetPvpMarch(started.March.ID)
		if err != nil {
			t.Fatalf("GetPvpMarch failed: %v", err)
		}
		storedDefender, err := repo.GetState(defender.Player.ID)
		if err != nil {
			t.Fatalf("GetState defender failed: %v", err)
		}
		return runResult{
			battle: battle, attackerReport: attackerReports[0], defenderReport: defenderReports[0],
			storedMarch: storedMarch, storedDefender: storedDefender,
			attackerLoss: attackerReports[0].LostUnits["shuInfantry"], defenderLoss: attackerReports[0].DefenderLostUnits["weiInfantry"],
			defenderUnitType: "weiInfantry",
		}
	}

	control := run(0)
	suppressed := run(1)
	for _, result := range []runResult{control, suppressed} {
		if defensePower, ok := result.battle.Result["defensePower"].(float64); !ok || defensePower != 750 {
			t.Fatalf("expected 25 temporarily suppressed defenders to leave 750 defense power, got %v", result.battle.Result["defensePower"])
		}
		for _, report := range []BattleReport{result.attackerReport, result.defenderReport} {
			outcome, ok := report.TraitOutcomes["qimen_dunjia"]
			suppressedUnits, unitsOK := outcome.Detail["suppressedUnits"].(map[string]int)
			if !ok || !unitsOK || suppressedUnits[result.defenderUnitType] != 25 {
				t.Fatalf("expected both reports to keep 25 temporary suppressions, report=%s outcome=%+v", report.ID, outcome)
			}
		}
		if got, want := armySliceToMap(result.storedDefender.Army)[result.defenderUnitType], 100-result.defenderLoss; got != want || got < 25 {
			t.Fatalf("expected temporarily suppressed troops to remain in real defender army, got=%d want=%d loss=%d", got, want, result.defenderLoss)
		}
		if result.storedMarch.AttackTroops["shuInfantry"] != 1000-result.attackerLoss {
			t.Fatalf("expected returning army to match final attacker loss, march=%+v loss=%d", result.storedMarch.AttackTroops, result.attackerLoss)
		}
	}

	controlDamage, ok := control.attackerReport.TraitOutcomes["laodang_yizhuang"].Detail["extraLosses"].(map[string]int)
	if !ok || controlDamage["shuInfantry"] != 100 {
		t.Fatalf("expected control defender trait to add 100 real attacker losses, outcome=%+v", control.attackerReport.TraitOutcomes["laodang_yizhuang"])
	}
	if control.attackerLoss-suppressed.attackerLoss != controlDamage["shuInfantry"] {
		t.Fatalf("expected Wolong suppression to remove exactly 100 extra losses, control=%d suppressed=%d", control.attackerLoss, suppressed.attackerLoss)
	}
	for _, report := range []BattleReport{suppressed.attackerReport, suppressed.defenderReport} {
		if _, exists := report.TraitOutcomes["laodang_yizhuang"]; exists || standardReportHasTrait(report.Detail, "laodang_yizhuang") {
			t.Fatalf("expected disabled enemy damage trait absent from both report formats, report=%s outcomes=%+v", report.ID, report.TraitOutcomes)
		}
		outcome, ok := report.TraitOutcomes["wolong_mouzhi"]
		if !ok || outcome.Detail["disabledGeneralCount"] != 1 || outcome.Detail["disabledTraitCount"] != 1 || outcome.Detail["triggerChance"] != float64(1) {
			t.Fatalf("expected all trigger traits of one enemy general disabled before battle, report=%s outcome=%+v", report.ID, outcome)
		}
		if !standardReportHasTrait(report.Detail, "qimen_dunjia") || !standardReportHasTrait(report.Detail, "wolong_mouzhi") {
			t.Fatalf("expected standard timeline to retain both distinct suppressions, report=%s detail=%+v", report.ID, report.Detail)
		}
	}
}

// TestPvpLuXunDamageTraitsOnlyReportActualCappedChanges 验证陆逊双追加伤害按剩余兵力封顶，并省略没有实际增量的后续特性。
func TestPvpLuXunDamageTraitsOnlyReportActualCappedChanges(t *testing.T) {
	type runResult struct {
		battle         PvpBattle
		attackerReport BattleReport
		defenderReport BattleReport
		storedMarch    PvpMarch
		storedDefender GameState
	}
	run := func(fireChance float64) runResult {
		setTestFactionsAndGenerals(t, FactionsConfig{
			"wu":  {Name: "吴国", Generals: []GeneralInfo{{ID: "luxun", Name: "陆逊"}}},
			"shu": {Name: "蜀国", Generals: []GeneralInfo{{ID: "liubei", Name: "刘备"}}},
		}, GeneralsConfig{Enabled: true, Heroes: map[string]GeneralHeroConfig{
			"luxun": {
				ID: "luxun", Name: "陆逊", Faction: "wu", Enabled: true,
				SpecialTrait: GeneralTraitConfig{
					TraitID: "huoshao_lianying", TraitType: general.TraitTypeSpecial, Enabled: true,
					Scope: "enemy_army", TargetUnitType: "infantry",
					Params: map[string]float64{"effectRate": 1, "maxAffectedRate": 1, "triggerChance": fireChance},
				},
				BonusTrait: GeneralTraitConfig{
					TraitID: "lianying_zengshang", TraitType: general.TraitTypeBonus, Enabled: true,
					Scope: "enemy_army", TargetUnitType: "infantry",
					Params: map[string]float64{"effectRate": 0.1, "triggerChance": 1},
				},
			},
			"liubei": {ID: "liubei", Name: "刘备", Faction: "shu", Enabled: true},
		}})
		svc, repo, attacker, defender := newPvpTestServiceForGenerals(t, "wu", "luxun", "shu", "liubei")
		attacker.Army = []ArmyUnit{{UnitType: "wuInfantry", Amount: 1200}}
		defender.Army = []ArmyUnit{{UnitType: "shuInfantry", Amount: 1000}}
		defender.Buildings = nil
		repo.players[attacker.Player.ID] = attacker
		repo.players[defender.Player.ID] = defender

		started, err := svc.StartPvpAttack(PvpAttackRequest{
			PlayerID: attacker.Player.ID, TargetPlayerID: defender.Player.ID, MarchMode: PvpMarchTypePlunder,
			Troops: map[string]int{"wuInfantry": 1200}, GeneralIDs: []string{"luxun"},
		})
		if err != nil {
			t.Fatalf("StartPvpAttack failed: %v", err)
		}
		forcePvpMarchDue(t, repo, started.March.ID)
		battle, err := svc.ResolvePvpMarch(started.March.ID)
		if err != nil {
			t.Fatalf("ResolvePvpMarch failed: %v", err)
		}
		attackerReports, _, err := repo.ListReports(attacker.Player.ID, 10, 0)
		if err != nil || len(attackerReports) == 0 || attackerReports[0].ID != battle.AttackerReportID {
			t.Fatalf("expected attacker report, reports=%+v err=%v", attackerReports, err)
		}
		defenderReports, _, err := repo.ListReports(defender.Player.ID, 10, 0)
		if err != nil || len(defenderReports) == 0 || defenderReports[0].ID != battle.DefenderReportID {
			t.Fatalf("expected defender report, reports=%+v err=%v", defenderReports, err)
		}
		storedMarch, err := repo.GetPvpMarch(started.March.ID)
		if err != nil {
			t.Fatalf("GetPvpMarch failed: %v", err)
		}
		storedDefender, err := repo.GetState(defender.Player.ID)
		if err != nil {
			t.Fatalf("GetState defender failed: %v", err)
		}
		return runResult{
			battle: battle, attackerReport: attackerReports[0], defenderReport: defenderReports[0],
			storedMarch: storedMarch, storedDefender: storedDefender,
		}
	}

	bonusOnly := run(0)
	fireCapped := run(1)
	for _, result := range []runResult{bonusOnly, fireCapped} {
		if attackPower, ok := result.battle.Result["attackerPower"].(float64); !ok || attackPower != 12000 {
			t.Fatalf("expected stable attacker power 12000, got %v", result.battle.Result["attackerPower"])
		}
		if defensePower, ok := result.battle.Result["defensePower"].(float64); !ok || defensePower != 10000 {
			t.Fatalf("expected stable defender power 10000, got %v", result.battle.Result["defensePower"])
		}
		attackerLoss := result.attackerReport.LostUnits["wuInfantry"]
		if result.storedMarch.AttackTroops["wuInfantry"] != 1200-attackerLoss {
			t.Fatalf("expected returning troops to match attacker report, march=%+v loss=%d", result.storedMarch.AttackTroops, attackerLoss)
		}
		defenderLoss := result.attackerReport.DefenderLostUnits["shuInfantry"]
		if got, want := armySliceToMap(result.storedDefender.Army)["shuInfantry"], 1000-defenderLoss; got != want {
			t.Fatalf("expected defender state and report to reconcile, got=%d want=%d", got, want)
		}
		for _, report := range []BattleReport{result.attackerReport, result.defenderReport} {
			if len(report.PvpAttackerGenerals) != 1 || !pvpSnapshotHasTrait(report.PvpAttackerGenerals[0], "huoshao_lianying") || !pvpSnapshotHasTrait(report.PvpAttackerGenerals[0], "lianying_zengshang") {
				t.Fatalf("expected Lu Xun snapshot to retain both owned traits, report=%s snapshots=%+v", report.ID, report.PvpAttackerGenerals)
			}
		}
	}

	bonusExtra, ok := bonusOnly.attackerReport.TraitOutcomes["lianying_zengshang"].Detail["targetExtraLosses"].(map[string]int)
	if !ok || bonusExtra["shuInfantry"] != 100 {
		t.Fatalf("expected Lianying bonus to add 100 losses when fire misses, outcome=%+v", bonusOnly.attackerReport.TraitOutcomes["lianying_zengshang"])
	}
	baseDefenderLoss := bonusOnly.attackerReport.DefenderLostUnits["shuInfantry"] - bonusExtra["shuInfantry"]
	if baseDefenderLoss <= 0 || baseDefenderLoss >= 1000 {
		t.Fatalf("expected bounded base plunder loss, total=%d bonus=%d", bonusOnly.attackerReport.DefenderLostUnits["shuInfantry"], bonusExtra["shuInfantry"])
	}
	for _, report := range []BattleReport{bonusOnly.attackerReport, bonusOnly.defenderReport} {
		if _, exists := report.TraitOutcomes["huoshao_lianying"]; exists || standardReportHasTrait(report.Detail, "huoshao_lianying") {
			t.Fatalf("expected missed fire absent from actual timeline, report=%s outcomes=%+v", report.ID, report.TraitOutcomes)
		}
		if !standardReportHasTrait(report.Detail, "lianying_zengshang") {
			t.Fatalf("expected effective Lianying bonus in standard timeline, report=%s detail=%+v", report.ID, report.Detail)
		}
	}

	fireExtra, ok := fireCapped.attackerReport.TraitOutcomes["huoshao_lianying"].Detail["targetExtraLosses"].(map[string]int)
	if !ok || fireExtra["shuInfantry"] != 1000-baseDefenderLoss {
		t.Fatalf("expected fire to add only remaining %d losses, outcome=%+v", 1000-baseDefenderLoss, fireCapped.attackerReport.TraitOutcomes["huoshao_lianying"])
	}
	if fireCapped.attackerReport.DefenderLostUnits["shuInfantry"] != 1000 || armySliceToMap(fireCapped.storedDefender.Army)["shuInfantry"] != 0 {
		t.Fatalf("expected capped fire to leave no defenders, report=%+v state=%+v", fireCapped.attackerReport.DefenderLostUnits, fireCapped.storedDefender.Army)
	}
	for _, report := range []BattleReport{fireCapped.attackerReport, fireCapped.defenderReport} {
		if _, exists := report.TraitOutcomes["lianying_zengshang"]; exists || standardReportHasTrait(report.Detail, "lianying_zengshang") {
			t.Fatalf("expected zero-change Lianying bonus omitted after capped fire, report=%s outcomes=%+v", report.ID, report.TraitOutcomes)
		}
		if !standardReportHasTrait(report.Detail, "huoshao_lianying") {
			t.Fatalf("expected effective fire in standard timeline, report=%s detail=%+v", report.ID, report.Detail)
		}
	}
}

// TestPvpAttackWithoutGeneralDoesNotBorrowHomeGeneralTrait 验证主动出征不携将时不会借用城内主将的属性、特性、经验或战报快照。
func TestPvpAttackWithoutGeneralDoesNotBorrowHomeGeneralTrait(t *testing.T) {
	setTestFactionsAndGenerals(t, FactionsConfig{
		"wei": {Name: "魏国", Generals: []GeneralInfo{{ID: "caocao", Name: "曹操"}}},
		"shu": {Name: "蜀国", Generals: []GeneralInfo{{ID: "liubei", Name: "刘备"}}},
	}, GeneralsConfig{Enabled: true, Heroes: map[string]GeneralHeroConfig{
		"caocao": {
			ID: "caocao", Name: "曹操", Faction: "wei", Enabled: true,
			Buffs: map[string]float64{StatAttackBonus: 1},
			SpecialTrait: GeneralTraitConfig{
				TraitID: "huogong", TraitType: general.TraitTypeSpecial, Enabled: true,
				Scope: "enemy_army", AllowedSides: []string{"attacker"},
				Params: map[string]float64{"damagePercent": 0.5, "triggerChance": 1},
			},
		},
		"liubei": {ID: "liubei", Name: "刘备", Faction: "shu", Enabled: true},
	}})
	svc, repo, attacker, defender := newPvpTestService(t)
	attacker.Army = []ArmyUnit{{UnitType: "weiInfantry", Amount: 100}}
	defender.Army = []ArmyUnit{{UnitType: "shuInfantry", Amount: 100}}
	repo.players[attacker.Player.ID] = attacker
	repo.players[defender.Player.ID] = defender

	started, err := svc.StartPvpAttack(PvpAttackRequest{
		PlayerID: attacker.Player.ID, TargetPlayerID: defender.Player.ID, MarchMode: PvpMarchTypeAttack,
		Troops: map[string]int{"weiInfantry": 100},
	})
	if err != nil {
		t.Fatalf("StartPvpAttack without general failed: %v", err)
	}
	if len(started.March.AttackGenerals) != 0 {
		t.Fatalf("expected general-free march, got %+v", started.March.AttackGenerals)
	}
	forcePvpMarchDue(t, repo, started.March.ID)
	battle, err := svc.ResolvePvpMarch(started.March.ID)
	if err != nil {
		t.Fatalf("ResolvePvpMarch without general failed: %v", err)
	}
	if attackPower, ok := battle.Result["attackerPower"].(float64); !ok || attackPower != 1000 {
		t.Fatalf("expected base attack power 1000 without home general buff, got %v", battle.Result["attackerPower"])
	}

	attackerReports, _, err := repo.ListReports(attacker.Player.ID, 10, 0)
	if err != nil || len(attackerReports) == 0 || attackerReports[0].ID != battle.AttackerReportID {
		t.Fatalf("expected attacker report, reports=%+v err=%v", attackerReports, err)
	}
	defenderReports, _, err := repo.ListReports(defender.Player.ID, 10, 0)
	if err != nil || len(defenderReports) == 0 || defenderReports[0].ID != battle.DefenderReportID {
		t.Fatalf("expected defender report, reports=%+v err=%v", defenderReports, err)
	}
	for _, report := range []BattleReport{attackerReports[0], defenderReports[0]} {
		if _, triggered := report.TraitOutcomes["huogong"]; triggered || len(report.TraitTriggered) != 0 {
			t.Fatalf("expected no borrowed home-general trait in either report, report=%s outcomes=%+v", report.ID, report.TraitOutcomes)
		}
		if len(report.PvpAttackerGenerals) != 0 || report.Detail == nil || len(report.Detail.Traits) != 0 {
			t.Fatalf("expected no attacker general snapshot or trait timeline, report=%+v", report)
		}
	}
	if attackerReports[0].GeneralExpGained != 0 {
		t.Fatalf("expected no attacker general exp, got %d", attackerReports[0].GeneralExpGained)
	}
	storedAttacker, err := repo.GetState(attacker.Player.ID)
	if err != nil {
		t.Fatalf("GetState attacker failed: %v", err)
	}
	if got := pvpTestGeneralExp(storedAttacker, "caocao"); got != 0 {
		t.Fatalf("expected home general exp unchanged, got %d", got)
	}
	storedDefender, err := repo.GetState(defender.Player.ID)
	if err != nil {
		t.Fatalf("GetState defender failed: %v", err)
	}
	if got, want := armySliceToMap(storedDefender.Army)["shuInfantry"], 100-attackerReports[0].DefenderLostUnits["shuInfantry"]; got != want {
		t.Fatalf("expected defender state and report losses to reconcile, got=%d want=%d", got, want)
	}
}

// TestPvpTraitDisabledDuringMarchUsesSettlementConfigEverywhere 验证攻方关闭、守方开启特性后统一采用结算时配置。
func TestPvpTraitDisabledDuringMarchUsesSettlementConfigEverywhere(t *testing.T) {
	setTestFactionsAndGenerals(t, FactionsConfig{
		"wei": {Name: "魏国", Generals: []GeneralInfo{{ID: "caocao", Name: "曹操"}}},
		"shu": {Name: "蜀国", Generals: []GeneralInfo{{ID: "sunquan", Name: "孙权"}}},
	}, GeneralsConfig{Enabled: true, Heroes: map[string]GeneralHeroConfig{
		"caocao": {
			ID: "caocao", Name: "曹操", Faction: "wei", Enabled: true,
			Buffs: map[string]float64{StatAttackBonus: 1},
			SpecialTrait: GeneralTraitConfig{
				TraitID: "huogong", TraitType: general.TraitTypeSpecial, Enabled: true,
				Scope: "enemy_army", AllowedSides: []string{"attacker"},
				Params: map[string]float64{"damagePercent": 0.5, "triggerChance": 1},
			},
		},
		"sunquan": {
			ID: "sunquan", Name: "孙权", Faction: "shu", Enabled: true,
			BonusTrait: GeneralTraitConfig{
				TraitID: "jiangdong_gushou", TraitType: general.TraitTypeBonus, Enabled: false,
				Scope: "self_army", AllowedSides: []string{"defender", "reinforcement"},
				Params: map[string]float64{"defenseBonusRate": 0.5, "triggerChance": 1},
			},
		},
	}})
	svc, repo, attacker, defender := newPvpTestServiceForGenerals(t, "wei", "caocao", "shu", "sunquan")
	unitsMu.Lock()
	shuInfantry := activeUnits["shu"]["shuInfantry"]
	shuInfantry.Stats["upkeep"] = 0
	activeUnits["shu"]["shuInfantry"] = shuInfantry
	unitsMu.Unlock()
	attacker.Army = []ArmyUnit{{UnitType: "weiInfantry", Amount: 100}}
	defender.Army = []ArmyUnit{{UnitType: "shuInfantry", Amount: 1000}}
	defender.Buildings = nil
	repo.players[attacker.Player.ID] = attacker
	repo.players[defender.Player.ID] = defender

	started, err := svc.StartPvpAttack(PvpAttackRequest{
		PlayerID: attacker.Player.ID, TargetPlayerID: defender.Player.ID, MarchMode: PvpMarchTypeAttack,
		Troops: map[string]int{"weiInfantry": 100}, GeneralIDs: []string{"caocao"},
	})
	if err != nil {
		t.Fatalf("StartPvpAttack failed: %v", err)
	}
	settlementConfig := GetGeneralsConfig()
	disabledCaoCao := settlementConfig.Heroes["caocao"]
	disabledCaoCao.SpecialTrait.Enabled = false
	settlementConfig.Heroes["caocao"] = disabledCaoCao
	enabledSunQuan := settlementConfig.Heroes["sunquan"]
	enabledSunQuan.BonusTrait.Enabled = true
	settlementConfig.Heroes["sunquan"] = enabledSunQuan
	if err := SetGeneralsConfig(settlementConfig); err != nil {
		t.Fatalf("switch attacker and defender traits before settlement failed: %v", err)
	}
	forcePvpMarchDue(t, repo, started.March.ID)
	battle, err := svc.ResolvePvpMarch(started.March.ID)
	if err != nil {
		t.Fatalf("ResolvePvpMarch failed: %v", err)
	}
	if attackPower, ok := battle.Result["attackerPower"].(float64); !ok || attackPower != 2000 {
		t.Fatalf("expected enabled hero buff to remain while only fire trait is disabled, got %v", battle.Result["attackerPower"])
	}
	if defensePower, ok := battle.Result["defensePower"].(float64); !ok || defensePower != 15000 {
		t.Fatalf("expected newly enabled settlement defense trait to produce defense power 15000, got %v", battle.Result["defensePower"])
	}

	attackerReports, _, err := repo.ListReports(attacker.Player.ID, 10, 0)
	if err != nil || len(attackerReports) == 0 || attackerReports[0].ID != battle.AttackerReportID {
		t.Fatalf("expected attacker report, reports=%+v err=%v", attackerReports, err)
	}
	defenderReports, _, err := repo.ListReports(defender.Player.ID, 10, 0)
	if err != nil || len(defenderReports) == 0 || defenderReports[0].ID != battle.DefenderReportID {
		t.Fatalf("expected defender report, reports=%+v err=%v", defenderReports, err)
	}
	for _, report := range []BattleReport{attackerReports[0], defenderReports[0]} {
		if _, triggered := report.TraitOutcomes["huogong"]; triggered {
			t.Fatalf("expected disabled fire not to trigger, report=%s outcomes=%+v", report.ID, report.TraitOutcomes)
		}
		if len(report.PvpAttackerGenerals) != 1 || pvpSnapshotHasTrait(report.PvpAttackerGenerals[0], "huogong") {
			t.Fatalf("expected settlement snapshot to omit disabled fire, report=%s snapshots=%+v", report.ID, report.PvpAttackerGenerals)
		}
		if report.PvpAttackerGenerals[0].Buffs[StatAttackBonus] != 1 {
			t.Fatalf("expected settlement snapshot to preserve enabled hero buff, report=%s snapshot=%+v", report.ID, report.PvpAttackerGenerals[0])
		}
		if report.Detail == nil || standardReportHasTrait(report.Detail, "huogong") || standardDetailGeneralHasTrait(report.Detail, "huogong") {
			t.Fatalf("expected standard report to omit disabled fire from timeline and general snapshot, detail=%+v", report.Detail)
		}
		outcome, triggered := report.TraitOutcomes["jiangdong_gushou"]
		if !triggered || outcome.OwnerSide != "defender" || outcome.OwnerGeneralID != "sunquan" {
			t.Fatalf("expected settlement-enabled defense trait in both reports, report=%s outcome=%+v", report.ID, outcome)
		}
		if len(report.PvpDefenderGenerals) != 1 || !pvpSnapshotHasTrait(report.PvpDefenderGenerals[0], "jiangdong_gushou") {
			t.Fatalf("expected settlement defender snapshot to include Jiangdong Gushou, report=%s snapshots=%+v", report.ID, report.PvpDefenderGenerals)
		}
		if !standardReportHasTrait(report.Detail, "jiangdong_gushou") || !standardDetailGeneralHasTrait(report.Detail, "jiangdong_gushou") {
			t.Fatalf("expected standard report to include settlement-enabled defense trait, detail=%+v", report.Detail)
		}
	}
	if attackerReports[0].GeneralExpGained != 0 {
		t.Fatalf("expected attacker zero-exp fixture not to refresh Cao Cao indirectly, got %d", attackerReports[0].GeneralExpGained)
	}
	storedDefender, err := repo.GetState(defender.Player.ID)
	if err != nil {
		t.Fatalf("GetState defender failed: %v", err)
	}
	if got, want := armySliceToMap(storedDefender.Army)["shuInfantry"], 1000-attackerReports[0].DefenderLostUnits["shuInfantry"]; got != want {
		t.Fatalf("expected disabled-trait battle state and report to reconcile, got=%d want=%d", got, want)
	}
}

// TestPvpDefenderGeneralLeavingDuringIncomingMarchDoesNotDefend 验证敌军到达前离城的主将不再提供守城属性、特性、经验或战报快照。
func TestPvpDefenderGeneralLeavingDuringIncomingMarchDoesNotDefend(t *testing.T) {
	defenseTrait := GeneralTraitConfig{
		TraitID: "jiangdong_gushou", TraitType: general.TraitTypeBonus, Enabled: true,
		Scope: "self_army", AllowedSides: []string{"defender", "reinforcement"},
		Params: map[string]float64{"defenseBonusRate": 0.5, "triggerChance": 1},
	}
	setTestFactionsAndGenerals(t, FactionsConfig{
		"wei": {Name: "魏国", Generals: []GeneralInfo{{ID: "caocao", Name: "曹操"}}},
		"shu": {Name: "蜀国", Generals: []GeneralInfo{{ID: "sunquan", Name: "孙权"}}},
		"wu":  {Name: "吴国", Generals: []GeneralInfo{{ID: "lvmeng", Name: "吕蒙"}}},
	}, GeneralsConfig{Enabled: true, Heroes: map[string]GeneralHeroConfig{
		"caocao": {ID: "caocao", Name: "曹操", Faction: "wei", Enabled: true},
		"sunquan": {
			ID: "sunquan", Name: "孙权", Faction: "shu", Enabled: true,
			Buffs: map[string]float64{StatDefenseBonus: 1}, BonusTrait: defenseTrait,
		},
		"lvmeng": {ID: "lvmeng", Name: "吕蒙", Faction: "wu", Enabled: true},
	}})
	svc, repo, attacker, defender := newPvpTestServiceForGenerals(t, "wei", "caocao", "shu", "sunquan")
	now := time.Now()
	host := newPlayerState("player_pvp_defender_host", "第三方城池", "wu", "lvmeng", now)
	if err := repo.CreatePlayer("account_pvp_b", host, now); err != nil {
		t.Fatalf("CreatePlayer reinforcement host failed: %v", err)
	}
	attacker.Army = []ArmyUnit{{UnitType: "weiInfantry", Amount: 100}}
	defender.Army = []ArmyUnit{{UnitType: "shuInfantry", Amount: 101}}
	repo.players[attacker.Player.ID] = attacker
	repo.players[defender.Player.ID] = defender

	started, err := svc.StartPvpAttack(PvpAttackRequest{
		PlayerID: attacker.Player.ID, TargetPlayerID: defender.Player.ID, MarchMode: PvpMarchTypeAttack,
		Troops: map[string]int{"weiInfantry": 100},
	})
	if err != nil {
		t.Fatalf("StartPvpAttack failed: %v", err)
	}
	away, err := svc.SendReinforcement(SendReinforcementRequest{
		FromPlayerID: defender.Player.ID, TargetPlayerID: host.Player.ID,
		Troops: map[string]int{"shuInfantry": 1}, GeneralIDs: []string{"sunquan"},
	})
	if err != nil {
		t.Fatalf("SendReinforcement defender general away failed: %v", err)
	}
	if away.Reinforcement.Status != ReinforcementStatusMarching || len(away.Reinforcement.Generals) != 1 {
		t.Fatalf("expected Sun Quan to be away in marching reinforcement, record=%+v", away.Reinforcement)
	}
	forcePvpMarchDue(t, repo, started.March.ID)
	battle, err := svc.ResolvePvpMarch(started.March.ID)
	if err != nil {
		t.Fatalf("ResolvePvpMarch failed: %v", err)
	}
	if defensePower, ok := battle.Result["defensePower"].(float64); !ok || defensePower != 1020 {
		t.Fatalf("expected city-wall baseline defense power 1020 after home general leaves, got %v", battle.Result["defensePower"])
	}

	attackerReports, _, err := repo.ListReports(attacker.Player.ID, 10, 0)
	if err != nil || len(attackerReports) == 0 || attackerReports[0].ID != battle.AttackerReportID {
		t.Fatalf("expected attacker report, reports=%+v err=%v", attackerReports, err)
	}
	defenderReports, _, err := repo.ListReports(defender.Player.ID, 10, 0)
	if err != nil || len(defenderReports) == 0 || defenderReports[0].ID != battle.DefenderReportID {
		t.Fatalf("expected defender report, reports=%+v err=%v", defenderReports, err)
	}
	for _, report := range []BattleReport{attackerReports[0], defenderReports[0]} {
		if _, triggered := report.TraitOutcomes["jiangdong_gushou"]; triggered {
			t.Fatalf("expected away defender trait not to trigger, report=%s outcomes=%+v", report.ID, report.TraitOutcomes)
		}
		if len(report.PvpDefenderGenerals) != 0 || report.Detail == nil || standardDetailHasGeneral(report.Detail, "sunquan") {
			t.Fatalf("expected away defender absent from all report snapshots, report=%+v", report)
		}
	}
	if defenderReports[0].GeneralExpGained != 0 {
		t.Fatalf("expected away defender general not to gain defense exp, got %d", defenderReports[0].GeneralExpGained)
	}
	storedDefender, err := repo.GetState(defender.Player.ID)
	if err != nil {
		t.Fatalf("GetState defender failed: %v", err)
	}
	if got := pvpTestGeneralExp(storedDefender, "sunquan"); got != 0 {
		t.Fatalf("expected away Sun Quan exp unchanged, got %d", got)
	}
	if got, want := armySliceToMap(storedDefender.Army)["shuInfantry"], 100-defenderReports[0].LostUnits["shuInfantry"]; got != want {
		t.Fatalf("expected remaining home army and defense report to reconcile, got=%d want=%d", got, want)
	}
	awayRecord, err := repo.GetReinforcement(away.Reinforcement.ID)
	if err != nil || awayRecord.Status != ReinforcementStatusMarching || len(awayRecord.Generals) != 1 {
		t.Fatalf("expected outbound reinforcement to remain separate from home defense, record=%+v err=%v", awayRecord, err)
	}
}

// TestPvpDefenderGeneralReturningBeforeIncomingMarchDefendsAgain 验证敌军到达前真实归城的主将会恢复守城属性、特性、经验和战报快照。
func TestPvpDefenderGeneralReturningBeforeIncomingMarchDefendsAgain(t *testing.T) {
	defenseTrait := GeneralTraitConfig{
		TraitID: "jiangdong_gushou", TraitType: general.TraitTypeBonus, Enabled: true,
		Scope: "self_army", AllowedSides: []string{"defender", "reinforcement"},
		Params: map[string]float64{"defenseBonusRate": 0.5, "triggerChance": 1},
	}
	setTestFactionsAndGenerals(t, FactionsConfig{
		"wei": {Name: "魏国", Generals: []GeneralInfo{{ID: "caocao", Name: "曹操"}}},
		"shu": {Name: "蜀国", Generals: []GeneralInfo{{ID: "sunquan", Name: "孙权"}}},
		"wu":  {Name: "吴国", Generals: []GeneralInfo{{ID: "lvmeng", Name: "吕蒙"}}},
	}, GeneralsConfig{Enabled: true, Heroes: map[string]GeneralHeroConfig{
		"caocao": {ID: "caocao", Name: "曹操", Faction: "wei", Enabled: true},
		"sunquan": {
			ID: "sunquan", Name: "孙权", Faction: "shu", Enabled: true,
			Buffs: map[string]float64{StatDefenseBonus: 1}, BonusTrait: defenseTrait,
		},
		"lvmeng": {ID: "lvmeng", Name: "吕蒙", Faction: "wu", Enabled: true},
	}})
	svc, repo, attacker, defender := newPvpTestServiceForGenerals(t, "wei", "caocao", "shu", "sunquan")
	now := time.Now()
	host := newPlayerState("player_pvp_returned_defender_host", "第三方城池", "wu", "lvmeng", now)
	if err := repo.CreatePlayer("account_pvp_b", host, now); err != nil {
		t.Fatalf("CreatePlayer reinforcement host failed: %v", err)
	}
	attacker.Army = []ArmyUnit{{UnitType: "weiInfantry", Amount: 100}}
	defender.Army = []ArmyUnit{{UnitType: "shuInfantry", Amount: 101}}
	repo.players[attacker.Player.ID] = attacker
	repo.players[defender.Player.ID] = defender

	started, err := svc.StartPvpAttack(PvpAttackRequest{
		PlayerID: attacker.Player.ID, TargetPlayerID: defender.Player.ID, MarchMode: PvpMarchTypeAttack,
		Troops: map[string]int{"weiInfantry": 100},
	})
	if err != nil {
		t.Fatalf("StartPvpAttack failed: %v", err)
	}
	sent, err := svc.SendReinforcement(SendReinforcementRequest{
		FromPlayerID: defender.Player.ID, TargetPlayerID: host.Player.ID,
		Troops: map[string]int{"shuInfantry": 1}, GeneralIDs: []string{"sunquan"},
	})
	if err != nil {
		t.Fatalf("SendReinforcement defender general away failed: %v", err)
	}
	forceReinforcementDue(t, repo, sent.Reinforcement.ID, true)
	if _, err := svc.MarkReinforcementArrived(sent.Reinforcement.ID); err != nil {
		t.Fatalf("MarkReinforcementArrived failed: %v", err)
	}
	if _, err := svc.RecallReinforcement(defender.Player.ID, sent.Reinforcement.ID); err != nil {
		t.Fatalf("RecallReinforcement failed: %v", err)
	}
	forceReinforcementDue(t, repo, sent.Reinforcement.ID, false)
	if _, err := svc.CompleteReinforcementReturn(sent.Reinforcement.ID); err != nil {
		t.Fatalf("CompleteReinforcementReturn failed: %v", err)
	}
	returned, err := repo.GetState(defender.Player.ID)
	if err != nil {
		t.Fatalf("GetState returned defender failed: %v", err)
	}
	if !generalAvailableAtHome(returned.GeneralAssignments, "sunquan") || armySliceToMap(returned.Army)["shuInfantry"] != 101 {
		t.Fatalf("expected Sun Quan and one troop fully returned before defense, state=%+v", returned)
	}

	forcePvpMarchDue(t, repo, started.March.ID)
	battle, err := svc.ResolvePvpMarch(started.March.ID)
	if err != nil {
		t.Fatalf("ResolvePvpMarch failed: %v", err)
	}
	defensePower, ok := battle.Result["defensePower"].(float64)
	if !ok || math.Abs(defensePower-3090.6) > 1e-6 {
		t.Fatalf("expected returned Sun Quan buff + Jiangdong defense + wall power 3090.6, got %v", battle.Result["defensePower"])
	}

	attackerReports, _, err := repo.ListReports(attacker.Player.ID, 10, 0)
	if err != nil || len(attackerReports) == 0 || attackerReports[0].ID != battle.AttackerReportID {
		t.Fatalf("expected attacker report, reports=%+v err=%v", attackerReports, err)
	}
	defenderReports, _, err := repo.ListReports(defender.Player.ID, 10, 0)
	if err != nil || len(defenderReports) == 0 || defenderReports[0].ID != battle.DefenderReportID {
		t.Fatalf("expected defender report, reports=%+v err=%v", defenderReports, err)
	}
	for _, report := range []BattleReport{attackerReports[0], defenderReports[0]} {
		outcome, triggered := report.TraitOutcomes["jiangdong_gushou"]
		if !triggered || outcome.OwnerSide != "defender" || outcome.OwnerGeneralID != "sunquan" {
			t.Fatalf("expected returned Sun Quan defense trait in both reports, report=%s outcomes=%+v", report.ID, report.TraitOutcomes)
		}
		if len(report.PvpDefenderGenerals) != 1 || report.PvpDefenderGenerals[0].ID != "sunquan" || !standardDetailHasGeneral(report.Detail, "sunquan") {
			t.Fatalf("expected returned Sun Quan in all defense snapshots, report=%+v", report)
		}
		if !standardReportHasTrait(report.Detail, "jiangdong_gushou") {
			t.Fatalf("expected returned defense trait in standard timeline, detail=%+v", report.Detail)
		}
	}
	if defenderReports[0].GeneralExpGained <= 0 {
		t.Fatalf("expected returned defender general to gain defense exp, got %d", defenderReports[0].GeneralExpGained)
	}
	storedDefender, err := repo.GetState(defender.Player.ID)
	if err != nil {
		t.Fatalf("GetState defender after battle failed: %v", err)
	}
	if got := pvpTestGeneralExp(storedDefender, "sunquan"); got != defenderReports[0].GeneralExpGained {
		t.Fatalf("expected Sun Quan stored exp %d to match report, got %d", defenderReports[0].GeneralExpGained, got)
	}
	if got, want := armySliceToMap(storedDefender.Army)["shuInfantry"], 101-defenderReports[0].LostUnits["shuInfantry"]; got != want {
		t.Fatalf("expected returned defender army and report to reconcile, got=%d want=%d", got, want)
	}
}

// pvpSnapshotHasTrait 判断 PVP 将领快照是否仍包含指定特性。
func pvpSnapshotHasTrait(snapshot PvpGeneralSnapshot, traitID string) bool {
	for _, trait := range snapshot.Traits {
		if trait.TraitID == traitID {
			return true
		}
	}
	return false
}

// standardGeneralHasTrait 判断标准战报将领快照是否包含指定特性。
func standardGeneralHasTrait(generals []BattleReportGeneral, traitID string) bool {
	for _, item := range generals {
		for _, trait := range item.Traits {
			if trait.TraitID == traitID {
				return true
			}
		}
	}
	return false
}

// standardDetailGeneralHasTrait 同时检查标准战报主次双方将领快照。
func standardDetailGeneralHasTrait(detail *BattleReportDetail, traitID string) bool {
	if detail == nil {
		return false
	}
	if standardGeneralHasTrait(detail.PrimarySide.Generals, traitID) {
		return true
	}
	return detail.SecondarySide != nil && standardGeneralHasTrait(detail.SecondarySide.Generals, traitID)
}

// standardDetailHasGeneral 同时检查标准战报主次双方是否包含指定将领。
func standardDetailHasGeneral(detail *BattleReportDetail, generalID string) bool {
	if detail == nil {
		return false
	}
	for _, item := range detail.PrimarySide.Generals {
		if item.ID == generalID {
			return true
		}
	}
	if detail.SecondarySide != nil {
		for _, item := range detail.SecondarySide.Generals {
			if item.ID == generalID {
				return true
			}
		}
	}
	return false
}

// standardReportHasTrait 判断标准战报真实触发时间线是否包含指定特性。
func standardReportHasTrait(detail *BattleReportDetail, traitID string) bool {
	if detail == nil {
		return false
	}
	for _, trait := range detail.Traits {
		if trait.TraitID == traitID {
			return true
		}
	}
	return false
}

// TestPvpRenzhuReportMatchesReturningTroops 验证仁主守护复活数、行军返回兵力和最终主城兵力一致。
func TestPvpRenzhuReportMatchesReturningTroops(t *testing.T) {
	setTestFactionsAndGenerals(t, FactionsConfig{
		"wei": {Name: "魏国", Generals: []GeneralInfo{{ID: "caocao", Name: "曹操"}}},
		"shu": {Name: "蜀国", Generals: []GeneralInfo{{ID: "liubei", Name: "刘备"}}},
	}, GeneralsConfig{
		Enabled: true,
		Heroes: map[string]GeneralHeroConfig{
			"caocao": {
				ID: "caocao", Name: "曹操", Faction: "wei", Enabled: true,
				BonusTrait: GeneralTraitConfig{
					TraitID: "renzhu_shouhu", TraitType: general.TraitTypeBonus, Enabled: true, Scope: "self_army",
					AllowedSides: []string{"attacker", "defender", "reinforcement"},
					Params:       map[string]float64{"effectRate": 0.35, "triggerChance": 1},
				},
			},
			"liubei": {ID: "liubei", Name: "刘备", Faction: "shu", Enabled: true},
		},
	})
	svc, repo, attacker, defender := newPvpTestService(t)
	attacker.Army = []ArmyUnit{{UnitType: "weiInfantry", Amount: 100}}
	defender.Army = []ArmyUnit{{UnitType: "weiInfantry", Amount: 100}}
	repo.players[attacker.Player.ID] = attacker
	repo.players[defender.Player.ID] = defender

	started, err := svc.StartPvpAttack(PvpAttackRequest{
		PlayerID: attacker.Player.ID, TargetPlayerID: defender.Player.ID, MarchMode: PvpMarchTypeAttack,
		Troops: map[string]int{"weiInfantry": 100}, GeneralIDs: []string{"caocao"},
	})
	if err != nil {
		t.Fatalf("StartPvpAttack failed: %v", err)
	}
	forcePvpMarchDue(t, repo, started.March.ID)
	battle, err := svc.ResolvePvpMarch(started.March.ID)
	if err != nil {
		t.Fatalf("ResolvePvpMarch failed: %v", err)
	}
	reports, _, err := repo.ListReports(attacker.Player.ID, 10, 0)
	if err != nil || len(reports) == 0 || reports[0].ID != battle.AttackerReportID {
		t.Fatalf("expected attacker report, reports=%+v err=%v", reports, err)
	}
	report := reports[0]
	lost := report.LostUnits["weiInfantry"]
	revived := report.RevivedUnits["weiInfantry"]
	if lost <= 0 || revived != int(math.Floor(float64(lost)*0.35)) {
		t.Fatalf("expected report to revive 35%% of real losses, lost=%d revived=%d outcomes=%+v", lost, revived, report.TraitOutcomes)
	}
	outcome := report.TraitOutcomes["renzhu_shouhu"]
	revivedDetail, ok := outcome.Detail["revivedUnits"].(map[string]int)
	if !ok || revivedDetail["weiInfantry"] != revived {
		t.Fatalf("expected trait detail to match report revived units, outcome=%+v report=%+v", outcome, report.RevivedUnits)
	}
	march, err := repo.GetPvpMarch(started.March.ID)
	if err != nil {
		t.Fatalf("GetPvpMarch failed: %v", err)
	}
	expectedReturned := 100 - lost + revived
	if report.SurvivedUnits["weiInfantry"] != expectedReturned || report.Detail == nil {
		t.Fatalf("expected report survivor snapshot %d, got legacy=%+v detail=%+v", expectedReturned, report.SurvivedUnits, report.Detail)
	}
	for _, unit := range report.Detail.PrimarySide.Units {
		if unit.UnitType == "weiInfantry" && unit.Survived != expectedReturned {
			t.Fatalf("expected standard report survivor %d, got %+v", expectedReturned, unit)
		}
	}
	if march.AttackTroops["weiInfantry"] != expectedReturned {
		t.Fatalf("expected returning troops %d = 100 - %d + %d, got %+v", expectedReturned, lost, revived, march.AttackTroops)
	}
	forcePvpReturnDue(t, repo, started.March.ID)
	completed, err := svc.CompletePvpRecall(started.March.ID)
	if err != nil {
		t.Fatalf("CompletePvpRecall failed: %v", err)
	}
	if got := armySliceToMap(completed.Army)["weiInfantry"]; got != expectedReturned {
		t.Fatalf("expected returned army to match report formula %d, got %d", expectedReturned, got)
	}
}

// TestPvpGuicaiYiceRevivesConfiguredShareAfterDefeat 验证鬼才遗策在战败后按 GM 比例复活并进入真实返回队列与战报。
func TestPvpGuicaiYiceRevivesConfiguredShareAfterDefeat(t *testing.T) {
	setTestFactionsAndGenerals(t, FactionsConfig{
		"wei": {Name: "魏国", Generals: []GeneralInfo{{ID: "caocao", Name: "曹操"}}},
		"shu": {Name: "蜀国", Generals: []GeneralInfo{{ID: "liubei", Name: "刘备"}}},
	}, GeneralsConfig{
		Enabled: true,
		Heroes: map[string]GeneralHeroConfig{
			"caocao": {
				ID: "caocao", Name: "曹操", Faction: "wei", Enabled: true,
				BonusTrait: GeneralTraitConfig{TraitID: "guicai_yice", TraitType: general.TraitTypeBonus, Enabled: true, Scope: "self_army", AllowedSides: []string{"attacker", "defender", "reinforcement"}, Params: map[string]float64{"effectRate": 0.5, "triggerChance": 1}},
			},
			"liubei": {ID: "liubei", Name: "刘备", Faction: "shu", Enabled: true},
		},
	})
	svc, repo, attacker, defender := newPvpTestService(t)
	attacker.Army = []ArmyUnit{{UnitType: "weiInfantry", Amount: 100}}
	defender.Army = []ArmyUnit{{UnitType: "weiInfantry", Amount: 1000}}
	repo.players[attacker.Player.ID] = attacker
	repo.players[defender.Player.ID] = defender

	started, err := svc.StartPvpAttack(PvpAttackRequest{
		PlayerID: attacker.Player.ID, TargetPlayerID: defender.Player.ID, MarchMode: PvpMarchTypeAttack,
		Troops: map[string]int{"weiInfantry": 100}, GeneralIDs: []string{"caocao"},
	})
	if err != nil {
		t.Fatalf("StartPvpAttack failed: %v", err)
	}
	forcePvpMarchDue(t, repo, started.March.ID)
	battle, err := svc.ResolvePvpMarch(started.March.ID)
	if err != nil {
		t.Fatalf("ResolvePvpMarch failed: %v", err)
	}
	reports, _, err := repo.ListReports(attacker.Player.ID, 10, 0)
	if err != nil || len(reports) == 0 || reports[0].ID != battle.AttackerReportID {
		t.Fatalf("expected attacker report, reports=%+v err=%v", reports, err)
	}
	report := reports[0]
	if report.OwnerOutcome != ReportOwnerOutcomeDefeat {
		t.Fatalf("expected attacker defeat, got %+v", report)
	}
	lost := report.LostUnits["weiInfantry"]
	returned := report.RevivedUnits["weiInfantry"]
	if lost <= 0 || returned != lost/2 {
		t.Fatalf("expected Guicai Yice to revive half of actual losses, lost=%d revived=%d", lost, returned)
	}
	outcome := report.TraitOutcomes["guicai_yice"]
	actualLost, lostOK := outcome.Detail["actualLostUnits"].(map[string]int)
	revivedUnits, revivedOK := outcome.Detail["revivedUnits"].(map[string]int)
	if !lostOK || !revivedOK || actualLost["weiInfantry"] != lost || revivedUnits["weiInfantry"] != returned || outcome.Detail["effectRate"] != 0.5 {
		t.Fatalf("expected actual loss and revived detail to match report, got %+v", outcome)
	}
	expectedSurvived := 100 - lost + returned
	march, err := repo.GetPvpMarch(started.March.ID)
	if err != nil {
		t.Fatalf("GetPvpMarch failed: %v", err)
	}
	if march.AttackTroops["weiInfantry"] != expectedSurvived || report.SurvivedUnits["weiInfantry"] != expectedSurvived {
		t.Fatalf("expected march and report survivors %d, march=%+v report=%+v", expectedSurvived, march.AttackTroops, report.SurvivedUnits)
	}
}

// TestPvpGuicaiYiceAlsoRevivesAfterVictory 验证鬼才遗策不再受战败条件限制。
func TestPvpGuicaiYiceAlsoRevivesAfterVictory(t *testing.T) {
	setTestFactionsAndGenerals(t, FactionsConfig{
		"wei": {Name: "魏国", Generals: []GeneralInfo{{ID: "caocao", Name: "曹操"}}},
		"shu": {Name: "蜀国", Generals: []GeneralInfo{{ID: "liubei", Name: "刘备"}}},
	}, GeneralsConfig{
		Enabled: true,
		Heroes: map[string]GeneralHeroConfig{
			"caocao": {
				ID: "caocao", Name: "曹操", Faction: "wei", Enabled: true,
				BonusTrait: GeneralTraitConfig{TraitID: "guicai_yice", TraitType: general.TraitTypeBonus, Enabled: true, Scope: "self_army", AllowedSides: []string{"attacker", "defender", "reinforcement"}, Params: map[string]float64{"effectRate": 0.5, "triggerChance": 1}},
			},
			"liubei": {ID: "liubei", Name: "刘备", Faction: "shu", Enabled: true},
		},
	})
	svc, repo, attacker, defender := newPvpTestService(t)
	attacker.Army = []ArmyUnit{{UnitType: "weiInfantry", Amount: 1000}}
	defender.Army = []ArmyUnit{{UnitType: "weiInfantry", Amount: 900}}
	repo.players[attacker.Player.ID] = attacker
	repo.players[defender.Player.ID] = defender

	started, err := svc.StartPvpAttack(PvpAttackRequest{
		PlayerID: attacker.Player.ID, TargetPlayerID: defender.Player.ID, MarchMode: PvpMarchTypeAttack,
		Troops: map[string]int{"weiInfantry": 1000}, GeneralIDs: []string{"caocao"},
	})
	if err != nil {
		t.Fatalf("StartPvpAttack failed: %v", err)
	}
	forcePvpMarchDue(t, repo, started.March.ID)
	battle, err := svc.ResolvePvpMarch(started.March.ID)
	if err != nil {
		t.Fatalf("ResolvePvpMarch failed: %v", err)
	}
	reports, _, err := repo.ListReports(attacker.Player.ID, 10, 0)
	if err != nil || len(reports) == 0 || reports[0].ID != battle.AttackerReportID {
		t.Fatalf("expected attacker report, reports=%+v err=%v", reports, err)
	}
	report := reports[0]
	if report.OwnerOutcome != ReportOwnerOutcomeVictory {
		t.Fatalf("expected attacker victory, got %+v", report)
	}
	lost := report.LostUnits["weiInfantry"]
	revived := report.RevivedUnits["weiInfantry"]
	outcome, ok := report.TraitOutcomes["guicai_yice"]
	revivedUnits, revivedOK := outcome.Detail["revivedUnits"].(map[string]int)
	if lost <= 0 || !ok || !revivedOK || revived != lost/2 || revivedUnits["weiInfantry"] != revived {
		t.Fatalf("expected Guicai to revive half of actual losses after victory, lost=%d revived=%d outcome=%+v", lost, revived, outcome)
	}
	expectedSurvived := 1000 - lost + revived
	march, err := repo.GetPvpMarch(started.March.ID)
	if err != nil {
		t.Fatalf("GetPvpMarch failed: %v", err)
	}
	if march.AttackTroops["weiInfantry"] != expectedSurvived || report.SurvivedUnits["weiInfantry"] != expectedSurvived {
		t.Fatalf("expected survivors including revival %d, march=%+v report=%+v", expectedSurvived, march.AttackTroops, report.SurvivedUnits)
	}
	for _, unit := range report.Detail.PrimarySide.Units {
		if unit.UnitType == "weiInfantry" && unit.Survived != expectedSurvived {
			t.Fatalf("expected standard report survivor %d, got %+v", expectedSurvived, unit)
		}
	}
}

// TestPvpXiaobawangZhuijiOnlyAppliesAfterPlunderVictory 验证孙策追击只在掠夺获胜后形成真实追加损失。
func TestPvpXiaobawangZhuijiOnlyAppliesAfterPlunderVictory(t *testing.T) {
	tests := []struct {
		name          string
		marchMode     string
		attackerCount int
		defenderCount int
		wantOutcome   string
		wantTriggered bool
	}{
		{name: "掠夺获胜触发", marchMode: PvpMarchTypePlunder, attackerCount: 200, defenderCount: 100, wantOutcome: ReportOwnerOutcomeVictory, wantTriggered: true},
		{name: "普通进攻不触发", marchMode: PvpMarchTypeAttack, attackerCount: 200, defenderCount: 100, wantOutcome: ReportOwnerOutcomeVictory, wantTriggered: false},
		{name: "掠夺战败不触发", marchMode: PvpMarchTypePlunder, attackerCount: 100, defenderCount: 200, wantOutcome: ReportOwnerOutcomeDefeat, wantTriggered: false},
		{name: "掠夺平局不触发", marchMode: PvpMarchTypePlunder, attackerCount: 100, defenderCount: 100, wantOutcome: ReportOwnerOutcomeDraw, wantTriggered: false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			setTestFactionsAndGenerals(t, FactionsConfig{
				"wu":  {Name: "吴国", Generals: []GeneralInfo{{ID: "sunce", Name: "孙策"}}},
				"shu": {Name: "蜀国", Generals: []GeneralInfo{{ID: "liubei", Name: "刘备"}}},
			}, GeneralsConfig{
				Enabled: true,
				Heroes: map[string]GeneralHeroConfig{
					"sunce": {
						ID: "sunce", Name: "孙策", Faction: "wu", Enabled: true,
						SpecialTrait: GeneralTraitConfig{TraitID: "xiaobawang_zhuiji", TraitType: general.TraitTypeSpecial, Enabled: true, Scope: "enemy_army", AllowedScenes: []string{"plunder"}, RequiredOutcome: "win", Params: map[string]float64{"effectRate": 0.5, "triggerChance": 1}},
					},
					"liubei": {ID: "liubei", Name: "刘备", Faction: "shu", Enabled: true},
				},
			})
			svc, repo, attacker, defender := newPvpTestServiceForGenerals(t, "wu", "sunce", "shu", "liubei")
			attacker.Army = []ArmyUnit{{UnitType: "wuInfantry", Amount: tc.attackerCount}}
			defender.Army = []ArmyUnit{{UnitType: "shuInfantry", Amount: tc.defenderCount}}
			defender.Buildings = nil
			repo.players[attacker.Player.ID] = attacker
			repo.players[defender.Player.ID] = defender

			started, err := svc.StartPvpAttack(PvpAttackRequest{
				PlayerID: attacker.Player.ID, TargetPlayerID: defender.Player.ID, MarchMode: tc.marchMode,
				Troops: map[string]int{"wuInfantry": tc.attackerCount}, GeneralIDs: []string{"sunce"},
			})
			if err != nil {
				t.Fatalf("StartPvpAttack failed: %v", err)
			}
			forcePvpMarchDue(t, repo, started.March.ID)
			battle, err := svc.ResolvePvpMarch(started.March.ID)
			if err != nil {
				t.Fatalf("ResolvePvpMarch failed: %v", err)
			}
			attackerReports, _, err := repo.ListReports(attacker.Player.ID, 10, 0)
			if err != nil || len(attackerReports) == 0 || attackerReports[0].ID != battle.AttackerReportID {
				t.Fatalf("expected attacker report, reports=%+v err=%v", attackerReports, err)
			}
			report := attackerReports[0]
			if report.OwnerOutcome != tc.wantOutcome {
				t.Fatalf("expected owner outcome %s, got %+v", tc.wantOutcome, report)
			}
			outcome, triggered := report.TraitOutcomes["xiaobawang_zhuiji"]
			if triggered != tc.wantTriggered {
				t.Fatalf("expected triggered=%t, outcomes=%+v", tc.wantTriggered, report.TraitOutcomes)
			}
			defenderReports, _, err := repo.ListReports(defender.Player.ID, 10, 0)
			if err != nil || len(defenderReports) == 0 || defenderReports[0].ID != battle.DefenderReportID {
				t.Fatalf("expected defender report, reports=%+v err=%v", defenderReports, err)
			}
			defenderReport := defenderReports[0]
			finalDefender, err := repo.GetState(defender.Player.ID)
			if err != nil {
				t.Fatalf("GetState defender failed: %v", err)
			}
			if got, want := armySliceToMap(finalDefender.Army)["shuInfantry"], tc.defenderCount-defenderReport.LostUnits["shuInfantry"]; got != want {
				t.Fatalf("expected defender state %d to match report loss, got %d report=%+v", want, got, defenderReport.LostUnits)
			}
			if tc.wantOutcome == ReportOwnerOutcomeDraw {
				attackerLosses := pvpTestLossesFromBattle(t, battle, "attacker")
				defenderLosses := pvpTestLossesFromBattle(t, battle, "defender")
				if battle.Result["winner"] != "draw" || battle.Result["attackerPower"] != float64(1000) || battle.Result["defensePower"] != float64(1000) || attackerLosses["wuInfantry"] != 50 || defenderLosses["shuInfantry"] != 50 {
					t.Fatalf("expected exact 1000/1000 draw with losses 50/50, result=%+v losses=%+v", battle.Result, battle.Losses)
				}
				if report.SurvivedUnits["wuInfantry"] != 50 || defenderReport.SurvivedUnits["shuInfantry"] != 50 || report.GeneralExpGained != 50 || defenderReport.GeneralExpGained != 50 {
					t.Fatalf("expected draw survivors and exp 50/50, reports=%+v/%+v", report, defenderReport)
				}
			}
			if tc.wantTriggered {
				if outcome.Detail["effectRate"] != 0.5 {
					t.Fatalf("expected persisted pursuit design rate 0.5, got %+v", outcome)
				}
				extraLosses, ok := outcome.Detail["extraLosses"].(map[string]int)
				if !ok || extraLosses["shuInfantry"] <= 0 {
					t.Fatalf("expected positive pursuit losses, got %+v", outcome)
				}
				found := false
				for _, trait := range report.Detail.Traits {
					if trait.TraitID == "xiaobawang_zhuiji" {
						found = true
						if trait.Detail["effectRate"] != 0.5 {
							t.Fatalf("expected standard report pursuit design rate 0.5, got %+v", trait.Detail)
						}
						if values, ok := trait.Detail["extraLosses"].(map[string]int); !ok || values["shuInfantry"] != extraLosses["shuInfantry"] {
							t.Fatalf("expected standard report pursuit values %+v, got %+v", extraLosses, trait.Detail)
						}
					}
				}
				if !found {
					t.Fatalf("expected pursuit in standard report, got %+v", report.Detail.Traits)
				}
			}
		})
	}
}

// TestPvpXiliangTujiMatchesCavalryCategoryInRealBattle 验证马超在真实 PVP 中只给具体骑兵追加损失，并同步真实状态和双方战报。
func TestPvpXiliangTujiMatchesCavalryCategoryInRealBattle(t *testing.T) {
	setTestFactionsAndGenerals(t, FactionsConfig{
		"shu": {Name: "蜀国", Generals: []GeneralInfo{{ID: "machao", Name: "马超"}}},
		"wu":  {Name: "吴国", Generals: []GeneralInfo{{ID: "sunquan", Name: "孙权"}}},
	}, GeneralsConfig{
		Enabled: true,
		Heroes: map[string]GeneralHeroConfig{
			"machao": {
				ID: "machao", Name: "马超", Faction: "shu", Enabled: true,
				SpecialTrait: GeneralTraitConfig{
					TraitID: "xiliang_tuji", TraitType: general.TraitTypeSpecial, Enabled: true,
					Scope: "enemy_army", TargetUnitType: "cavalry",
					Params: map[string]float64{"effectRate": 0.2, "triggerChance": 1},
				},
			},
			"sunquan": {ID: "sunquan", Name: "孙权", Faction: "wu", Enabled: true},
		},
	})
	svc, repo, attacker, defender := newPvpTestServiceForGenerals(t, "shu", "machao", "wu", "sunquan")
	unitsMu.Lock()
	activeUnits["wu"]["wuCavalry"] = UnitConfig{
		Name: "吴测试骑兵", Category: "cavalry",
		Stats: map[string]int{"attack": 10, "infantryDefense": 10, "cavalryDefense": 10, "carryCapacity": 5, "upkeep": 1},
	}
	unitsMu.Unlock()

	attacker.Army = []ArmyUnit{{UnitType: "shuInfantry", Amount: 100}}
	defender.Army = []ArmyUnit{{UnitType: "wuInfantry", Amount: 1000}, {UnitType: "wuCavalry", Amount: 1000}}
	repo.players[attacker.Player.ID] = attacker
	repo.players[defender.Player.ID] = defender

	started, err := svc.StartPvpAttack(PvpAttackRequest{
		PlayerID: attacker.Player.ID, TargetPlayerID: defender.Player.ID, MarchMode: PvpMarchTypeAttack,
		Troops: map[string]int{"shuInfantry": 100}, GeneralIDs: []string{"machao"},
	})
	if err != nil {
		t.Fatalf("StartPvpAttack failed: %v", err)
	}
	forcePvpMarchDue(t, repo, started.March.ID)
	battle, err := svc.ResolvePvpMarch(started.March.ID)
	if err != nil {
		t.Fatalf("ResolvePvpMarch failed: %v", err)
	}
	attackerReports, _, err := repo.ListReports(attacker.Player.ID, 10, 0)
	if err != nil || len(attackerReports) == 0 || attackerReports[0].ID != battle.AttackerReportID {
		t.Fatalf("expected attacker report, reports=%+v err=%v", attackerReports, err)
	}
	defenderReports, _, err := repo.ListReports(defender.Player.ID, 10, 0)
	if err != nil || len(defenderReports) == 0 || defenderReports[0].ID != battle.DefenderReportID {
		t.Fatalf("expected defender report, reports=%+v err=%v", defenderReports, err)
	}
	attackerOutcome := attackerReports[0].TraitOutcomes["xiliang_tuji"]
	if attackerOutcome.Detail["effectRate"] != 0.2 {
		t.Fatalf("expected persisted Xiliang design rate 0.2, got %+v", attackerOutcome)
	}
	extra, ok := attackerOutcome.Detail["targetExtraLosses"].(map[string]int)
	if !ok || extra["wuCavalry"] <= 0 || extra["wuInfantry"] != 0 || len(extra) != 1 {
		t.Fatalf("expected only concrete cavalry to receive extra losses, got %+v", attackerOutcome)
	}
	defenderOutcome := defenderReports[0].TraitOutcomes["xiliang_tuji"]
	if defenderOutcome.Detail["effectRate"] != 0.2 {
		t.Fatalf("expected defender report Xiliang design rate 0.2, got %+v", defenderOutcome)
	}
	defenderExtra, ok := defenderOutcome.Detail["targetExtraLosses"].(map[string]int)
	if !ok || defenderExtra["wuCavalry"] != extra["wuCavalry"] {
		t.Fatalf("expected both reports to use same cavalry extra losses, attacker=%+v defender=%+v", extra, defenderOutcome)
	}
	battleLosses := pvpTestLossesFromBattle(t, battle, "defender")
	if battleLosses["wuCavalry"] != defenderReports[0].LostUnits["wuCavalry"] || battleLosses["wuInfantry"] != defenderReports[0].LostUnits["wuInfantry"] {
		t.Fatalf("expected battle and defender report losses to match, battle=%+v report=%+v", battleLosses, defenderReports[0].LostUnits)
	}
	storedDefender, err := repo.GetState(defender.Player.ID)
	if err != nil {
		t.Fatalf("GetState defender failed: %v", err)
	}
	remaining := armySliceToMap(storedDefender.Army)
	if remaining["wuCavalry"] != 1000-battleLosses["wuCavalry"] || remaining["wuInfantry"] != 1000-battleLosses["wuInfantry"] {
		t.Fatalf("expected real defender army to match report losses, remaining=%+v losses=%+v", remaining, battleLosses)
	}
}

// TestPvpAfterCombatDamageTraitsWorkForBothMainSides 验证四项战后追加伤害由进攻或防守主将携带时都真实扣除敌军并写入双方战报。
func TestPvpAfterCombatDamageTraitsWorkForBothMainSides(t *testing.T) {
	cases := []struct {
		name           string
		generalID      string
		generalName    string
		faction        string
		traitID        string
		traitType      string
		targetUnitType string
		effectRate     float64
		detailKey      string
		killsTarget    bool
	}{
		{name: "黄忠老当益壮", generalID: "huangzhong", generalName: "黄忠", faction: "shu", traitID: "laodang_yizhuang", traitType: general.TraitTypeBonus, effectRate: 0.1, detailKey: "extraLosses"},
		{name: "陆逊火烧联营", generalID: "luxun", generalName: "陆逊", faction: "wu", traitID: "huoshao_lianying", traitType: general.TraitTypeSpecial, targetUnitType: "infantry", effectRate: 1, detailKey: "targetExtraLosses", killsTarget: true},
		{name: "陆逊连营增伤", generalID: "luxun", generalName: "陆逊", faction: "wu", traitID: "lianying_zengshang", traitType: general.TraitTypeBonus, targetUnitType: "infantry", effectRate: 0.1, detailKey: "targetExtraLosses"},
		{name: "黄盖苦肉反击", generalID: "huanggai", generalName: "黄盖", faction: "wu", traitID: "kurou_fanji", traitType: general.TraitTypeBonus, effectRate: 0.1, detailKey: "extraLosses"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			for _, ownerSide := range []string{"attacker", "defender"} {
				t.Run(ownerSide, func(t *testing.T) {
					traitConfig := GeneralTraitConfig{
						TraitID: tc.traitID, TraitType: tc.traitType, Enabled: true, Scope: "enemy_army", TargetUnitType: tc.targetUnitType,
						Params: map[string]float64{"effectRate": tc.effectRate, "triggerChance": 1},
					}
					hero := GeneralHeroConfig{ID: tc.generalID, Name: tc.generalName, Faction: tc.faction, Enabled: true}
					if tc.traitType == general.TraitTypeSpecial {
						hero.SpecialTrait = traitConfig
					} else {
						hero.BonusTrait = traitConfig
					}
					setTestFactionsAndGenerals(t, FactionsConfig{
						"wei":      {Name: "魏国", Generals: []GeneralInfo{{ID: "opponent", Name: "对手"}}},
						tc.faction: {Name: tc.faction, Generals: []GeneralInfo{{ID: tc.generalID, Name: tc.generalName}}},
					}, GeneralsConfig{Enabled: true, Heroes: map[string]GeneralHeroConfig{
						"opponent":   {ID: "opponent", Name: "对手", Faction: "wei", Enabled: true},
						tc.generalID: hero,
					}})

					attackerFaction, attackerGeneralID := tc.faction, tc.generalID
					defenderFaction, defenderGeneralID := "wei", "opponent"
					attackerCount, defenderCount := 100, 1000
					generalIDs := []string{tc.generalID}
					if ownerSide == "defender" {
						attackerFaction, attackerGeneralID = "wei", "opponent"
						defenderFaction, defenderGeneralID = tc.faction, tc.generalID
						attackerCount, defenderCount = 1000, 100
						generalIDs = nil
					}
					svc, repo, attacker, defender := newPvpTestServiceForGenerals(t, attackerFaction, attackerGeneralID, defenderFaction, defenderGeneralID)
					attackerUnitType := attackerFaction + "Infantry"
					defenderUnitType := defenderFaction + "Infantry"
					attacker.Army = []ArmyUnit{{UnitType: attackerUnitType, Amount: attackerCount}}
					defender.Army = []ArmyUnit{{UnitType: defenderUnitType, Amount: defenderCount}}
					defender.Buildings = nil
					repo.players[attacker.Player.ID] = attacker
					repo.players[defender.Player.ID] = defender

					started, err := svc.StartPvpAttack(PvpAttackRequest{
						PlayerID: attacker.Player.ID, TargetPlayerID: defender.Player.ID, MarchMode: PvpMarchTypeAttack,
						Troops: map[string]int{attackerUnitType: attackerCount}, GeneralIDs: generalIDs,
					})
					if err != nil {
						t.Fatalf("StartPvpAttack failed: %v", err)
					}
					forcePvpMarchDue(t, repo, started.March.ID)
					battle, err := svc.ResolvePvpMarch(started.March.ID)
					if err != nil {
						t.Fatalf("ResolvePvpMarch failed: %v", err)
					}
					attackerReports, _, err := repo.ListReports(attacker.Player.ID, 10, 0)
					if err != nil || len(attackerReports) == 0 || attackerReports[0].ID != battle.AttackerReportID {
						t.Fatalf("expected attacker report, reports=%+v err=%v", attackerReports, err)
					}
					defenderReports, _, err := repo.ListReports(defender.Player.ID, 10, 0)
					if err != nil || len(defenderReports) == 0 || defenderReports[0].ID != battle.DefenderReportID {
						t.Fatalf("expected defender report, reports=%+v err=%v", defenderReports, err)
					}

					enemySide := "defender"
					enemyUnitType := defenderUnitType
					enemyCount := defenderCount
					if ownerSide == "defender" {
						enemySide = "attacker"
						enemyUnitType = attackerUnitType
						enemyCount = attackerCount
					}
					battleLosses := pvpTestLossesFromBattle(t, battle, enemySide)
					actualExtra := 0
					for _, report := range []BattleReport{attackerReports[0], defenderReports[0]} {
						outcome, ok := report.TraitOutcomes[tc.traitID]
						if !ok || outcome.OwnerSide != ownerSide || outcome.OwnerGeneralID != tc.generalID {
							t.Fatalf("expected %s outcome owned by %s/%s, got %+v", tc.traitID, ownerSide, tc.generalID, report.TraitOutcomes)
						}
						if outcome.Detail["effectRate"] != tc.effectRate {
							t.Fatalf("expected %s design rate %.2f, got %+v", tc.traitID, tc.effectRate, outcome.Detail)
						}
						values, ok := outcome.Detail[tc.detailKey].(map[string]int)
						if !ok || values[enemyUnitType] <= 0 || len(values) != 1 {
							t.Fatalf("expected positive concrete-unit %s, got %+v", tc.detailKey, outcome)
						}
						if actualExtra == 0 {
							actualExtra = values[enemyUnitType]
						} else if values[enemyUnitType] != actualExtra {
							t.Fatalf("expected both reports to use extra loss %d, got %+v", actualExtra, values)
						}
						standardFound := false
						for _, standardTrait := range report.Detail.Traits {
							if standardTrait.TraitID != tc.traitID || standardTrait.GeneralID != tc.generalID || standardTrait.OwnerRole != ownerSide {
								continue
							}
							standardValues, ok := standardTrait.Detail[tc.detailKey].(map[string]int)
							if !ok || standardValues[enemyUnitType] != actualExtra {
								t.Fatalf("expected standard report value %d, got %+v", actualExtra, standardTrait.Detail)
							}
							if standardTrait.Detail["effectRate"] != tc.effectRate {
								t.Fatalf("expected standard report design rate %.2f, got %+v", tc.effectRate, standardTrait.Detail)
							}
							standardFound = true
						}
						if !standardFound {
							t.Fatalf("expected %s in standard report, got %+v", tc.traitID, report.Detail.Traits)
						}
					}
					if tc.killsTarget {
						if battleLosses[enemyUnitType] != enemyCount {
							t.Fatalf("expected full target loss %d, got %+v", enemyCount, battleLosses)
						}
					} else if actualExtra != int(float64(enemyCount)*tc.effectRate) {
						t.Fatalf("expected actual extra loss %d, got %d", int(float64(enemyCount)*tc.effectRate), actualExtra)
					}
					if ownerSide == "attacker" {
						if attackerReports[0].DefenderLostUnits[enemyUnitType] != battleLosses[enemyUnitType] || defenderReports[0].LostUnits[enemyUnitType] != battleLosses[enemyUnitType] {
							t.Fatalf("expected legacy reports to agree on defender loss, battle=%+v attacker=%+v defender=%+v", battleLosses, attackerReports[0].DefenderLostUnits, defenderReports[0].LostUnits)
						}
					} else if attackerReports[0].LostUnits[enemyUnitType] != battleLosses[enemyUnitType] || defenderReports[0].DefenderLostUnits[enemyUnitType] != battleLosses[enemyUnitType] {
						t.Fatalf("expected legacy reports to agree on attacker loss, battle=%+v attacker=%+v defender=%+v", battleLosses, attackerReports[0].LostUnits, defenderReports[0].DefenderLostUnits)
					}
					standardEnemySide := defenderReports[0].Detail.SecondarySide
					if ownerSide == "defender" {
						standardEnemySide = &defenderReports[0].Detail.PrimarySide
					}
					standardLoss := -1
					if standardEnemySide != nil {
						for _, unit := range standardEnemySide.Units {
							if unit.UnitType == enemyUnitType {
								standardLoss = unit.Lost
							}
						}
					}
					if standardLoss != battleLosses[enemyUnitType] {
						t.Fatalf("expected standard report enemy loss %d, got %d detail=%+v", battleLosses[enemyUnitType], standardLoss, defenderReports[0].Detail)
					}

					if ownerSide == "attacker" {
						storedDefender, err := repo.GetState(defender.Player.ID)
						if err != nil {
							t.Fatalf("GetState defender failed: %v", err)
						}
						if got, want := armySliceToMap(storedDefender.Army)[enemyUnitType], enemyCount-battleLosses[enemyUnitType]; got != want {
							t.Fatalf("expected defender state %d, got %d losses=%+v", want, got, battleLosses)
						}
					} else {
						storedMarch, err := repo.GetPvpMarch(started.March.ID)
						if err != nil {
							t.Fatalf("GetPvpMarch failed: %v", err)
						}
						if got, want := storedMarch.AttackTroops[enemyUnitType], enemyCount-battleLosses[enemyUnitType]; got != want {
							t.Fatalf("expected returning march %d, got %d losses=%+v", want, got, battleLosses)
						}
					}
				})
			}
		})
	}
}

// TestPvpSuppressionTraitsPreserveSuppressedTroopsForBothMainSides 验证三项临时压制按合法方向降低本场参战战力且不把压制兵误算成阵亡。
func TestPvpSuppressionTraitsPreserveSuppressedTroopsForBothMainSides(t *testing.T) {
	cases := []struct {
		name        string
		generalID   string
		generalName string
		faction     string
		traitID     string
		effectRate  float64
		ownerSides  []string
		noMaxField  bool
	}{
		{name: "张辽震慑全军", generalID: "zhangliao", generalName: "张辽", faction: "wei", traitID: "weizhen_zhenhe", effectRate: 0.25, ownerSides: []string{"attacker"}, noMaxField: true},
		{name: "张飞震慑全军", generalID: "zhangfei", generalName: "张飞", faction: "shu", traitID: "zhenhe_quanjun", effectRate: 0.5},
		{name: "诸葛亮奇门遁甲", generalID: "zhugeliang", generalName: "诸葛亮", faction: "shu", traitID: "qimen_dunjia", effectRate: 0.25},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ownerSides := tc.ownerSides
			if len(ownerSides) == 0 {
				ownerSides = []string{"attacker", "defender"}
			}
			for _, ownerSide := range ownerSides {
				t.Run(ownerSide, func(t *testing.T) {
					opponentFaction := "wei"
					if tc.faction == opponentFaction {
						opponentFaction = "wu"
					}
					traitParams := map[string]float64{"effectRate": tc.effectRate, "maxAffectedRate": tc.effectRate, "triggerChance": 1}
					allowedSides := []string(nil)
					if tc.noMaxField {
						delete(traitParams, "maxAffectedRate")
						allowedSides = []string{"attacker"}
					}
					setTestFactionsAndGenerals(t, FactionsConfig{
						tc.faction:      {Name: tc.faction, Generals: []GeneralInfo{{ID: tc.generalID, Name: tc.generalName}}},
						opponentFaction: {Name: opponentFaction, Generals: []GeneralInfo{{ID: "opponent", Name: "对手"}}},
					}, GeneralsConfig{Enabled: true, Heroes: map[string]GeneralHeroConfig{
						tc.generalID: {
							ID: tc.generalID, Name: tc.generalName, Faction: tc.faction, Enabled: true,
							SpecialTrait: GeneralTraitConfig{
								TraitID: tc.traitID, TraitType: general.TraitTypeSpecial, Enabled: true, Scope: "enemy_army", AllowedSides: allowedSides,
								Params: traitParams,
							},
						},
						"opponent": {ID: "opponent", Name: "对手", Faction: opponentFaction, Enabled: true},
					}})

					attackerFaction, attackerGeneralID := tc.faction, tc.generalID
					defenderFaction, defenderGeneralID := opponentFaction, "opponent"
					attackerCount, defenderCount := 100, 1000
					generalIDs := []string{tc.generalID}
					if ownerSide == "defender" {
						attackerFaction, attackerGeneralID = opponentFaction, "opponent"
						defenderFaction, defenderGeneralID = tc.faction, tc.generalID
						attackerCount, defenderCount = 1000, 100
						generalIDs = nil
					}
					svc, repo, attacker, defender := newPvpTestServiceForGenerals(t, attackerFaction, attackerGeneralID, defenderFaction, defenderGeneralID)
					attackerUnitType := attackerFaction + "Infantry"
					defenderUnitType := defenderFaction + "Infantry"
					attacker.Army = []ArmyUnit{{UnitType: attackerUnitType, Amount: attackerCount}}
					defender.Army = []ArmyUnit{{UnitType: defenderUnitType, Amount: defenderCount}}
					defender.Buildings = nil
					repo.players[attacker.Player.ID] = attacker
					repo.players[defender.Player.ID] = defender

					started, err := svc.StartPvpAttack(PvpAttackRequest{
						PlayerID: attacker.Player.ID, TargetPlayerID: defender.Player.ID, MarchMode: PvpMarchTypeAttack,
						Troops: map[string]int{attackerUnitType: attackerCount}, GeneralIDs: generalIDs,
					})
					if err != nil {
						t.Fatalf("StartPvpAttack failed: %v", err)
					}
					forcePvpMarchDue(t, repo, started.March.ID)
					battle, err := svc.ResolvePvpMarch(started.March.ID)
					if err != nil {
						t.Fatalf("ResolvePvpMarch failed: %v", err)
					}
					attackerReports, _, err := repo.ListReports(attacker.Player.ID, 10, 0)
					if err != nil || len(attackerReports) == 0 || attackerReports[0].ID != battle.AttackerReportID {
						t.Fatalf("expected attacker report, reports=%+v err=%v", attackerReports, err)
					}
					defenderReports, _, err := repo.ListReports(defender.Player.ID, 10, 0)
					if err != nil || len(defenderReports) == 0 || defenderReports[0].ID != battle.DefenderReportID {
						t.Fatalf("expected defender report, reports=%+v err=%v", defenderReports, err)
					}

					enemySide, enemyUnitType, enemyCount := "defender", defenderUnitType, defenderCount
					powerKey := "defensePower"
					if ownerSide == "defender" {
						enemySide, enemyUnitType, enemyCount = "attacker", attackerUnitType, attackerCount
						powerKey = "attackerPower"
					}
					wantSuppressed := int(float64(enemyCount) * tc.effectRate)
					wantPower := float64((enemyCount - wantSuppressed) * 10)
					if got, ok := battle.Result[powerKey].(float64); !ok || got != wantPower {
						t.Fatalf("expected %s %v after suppressing %d, got %+v", powerKey, wantPower, wantSuppressed, battle.Result[powerKey])
					}
					for _, report := range []BattleReport{attackerReports[0], defenderReports[0]} {
						outcome, ok := report.TraitOutcomes[tc.traitID]
						if !ok || outcome.OwnerSide != ownerSide || outcome.OwnerGeneralID != tc.generalID {
							t.Fatalf("expected %s outcome owned by %s/%s, got %+v", tc.traitID, ownerSide, tc.generalID, report.TraitOutcomes)
						}
						suppressed, ok := outcome.Detail["suppressedUnits"].(map[string]int)
						if !ok || suppressed[enemyUnitType] != wantSuppressed || len(suppressed) != 1 {
							t.Fatalf("expected suppressed %s=%d, got %+v", enemyUnitType, wantSuppressed, outcome)
						}
						standardFound := false
						for _, standardTrait := range report.Detail.Traits {
							if standardTrait.TraitID != tc.traitID || standardTrait.GeneralID != tc.generalID || standardTrait.OwnerRole != ownerSide {
								continue
							}
							standardSuppressed, ok := standardTrait.Detail["suppressedUnits"].(map[string]int)
							if !ok || standardSuppressed[enemyUnitType] != wantSuppressed {
								t.Fatalf("expected standard report suppressed %d, got %+v", wantSuppressed, standardTrait.Detail)
							}
							standardFound = true
						}
						if !standardFound {
							t.Fatalf("expected %s in standard report, got %+v", tc.traitID, report.Detail.Traits)
						}
					}

					battleLosses := pvpTestLossesFromBattle(t, battle, enemySide)
					if battleLosses[enemyUnitType] > enemyCount-wantSuppressed {
						t.Fatalf("suppressed troops must not become losses, suppressed=%d losses=%+v", wantSuppressed, battleLosses)
					}
					if ownerSide == "attacker" {
						if attackerReports[0].DefenderLostUnits[enemyUnitType] != battleLosses[enemyUnitType] || defenderReports[0].LostUnits[enemyUnitType] != battleLosses[enemyUnitType] {
							t.Fatalf("expected reports to agree on defender loss, battle=%+v attacker=%+v defender=%+v", battleLosses, attackerReports[0].DefenderLostUnits, defenderReports[0].LostUnits)
						}
						storedDefender, err := repo.GetState(defender.Player.ID)
						if err != nil {
							t.Fatalf("GetState defender failed: %v", err)
						}
						if got, want := armySliceToMap(storedDefender.Army)[enemyUnitType], enemyCount-battleLosses[enemyUnitType]; got != want {
							t.Fatalf("expected suppressed defenders preserved at %d, got %d", want, got)
						}
					} else {
						if attackerReports[0].LostUnits[enemyUnitType] != battleLosses[enemyUnitType] || defenderReports[0].DefenderLostUnits[enemyUnitType] != battleLosses[enemyUnitType] {
							t.Fatalf("expected reports to agree on attacker loss, battle=%+v attacker=%+v defender=%+v", battleLosses, attackerReports[0].LostUnits, defenderReports[0].DefenderLostUnits)
						}
						storedMarch, err := repo.GetPvpMarch(started.March.ID)
						if err != nil {
							t.Fatalf("GetPvpMarch failed: %v", err)
						}
						if got, want := storedMarch.AttackTroops[enemyUnitType], enemyCount-battleLosses[enemyUnitType]; got != want {
							t.Fatalf("expected suppressed attackers to return at %d, got %d", want, got)
						}
					}

					standardEnemySide := defenderReports[0].Detail.SecondarySide
					if ownerSide == "defender" {
						standardEnemySide = &defenderReports[0].Detail.PrimarySide
					}
					standardFound := false
					if standardEnemySide != nil {
						for _, unit := range standardEnemySide.Units {
							if unit.UnitType == enemyUnitType && unit.AmountBefore == enemyCount && unit.Dispatched == enemyCount && unit.Lost == battleLosses[enemyUnitType] && unit.Survived == enemyCount-battleLosses[enemyUnitType] {
								standardFound = true
							}
						}
					}
					if !standardFound {
						t.Fatalf("expected standard report to preserve full baseline and real losses, side=%+v losses=%+v", standardEnemySide, battleLosses)
					}
				})
			}
		})
	}
}

// TestPvpWeiwuTongyuProvidesRealPowerOnBothMainSides 验证同一防御模板在两侧均能保留归属，且只有防守侧修正进入本场防御战力。
func TestPvpWeiwuTongyuProvidesRealPowerOnBothMainSides(t *testing.T) {
	for _, ownerSide := range []string{"attacker", "defender"} {
		t.Run(ownerSide, func(t *testing.T) {
			setTestFactionsAndGenerals(t, FactionsConfig{
				"wei": {Name: "魏国", Generals: []GeneralInfo{{ID: "caocao", Name: "曹操"}}},
				"wu":  {Name: "吴国", Generals: []GeneralInfo{{ID: "opponent", Name: "对手"}}},
			}, GeneralsConfig{Enabled: true, Heroes: map[string]GeneralHeroConfig{
				"caocao": {
					ID: "caocao", Name: "曹操", Faction: "wei", Enabled: true,
					BonusTrait: GeneralTraitConfig{
						TraitID: "weiwu_tongyu", TraitType: general.TraitTypeBonus, Enabled: true, Scope: "self_army", TargetUnitType: "huWei",
						Params: map[string]float64{"defenseBonusRate": 0.15, "triggerChance": 1},
					},
				},
				"opponent": {ID: "opponent", Name: "对手", Faction: "wu", Enabled: true},
			}})

			attackerFaction, attackerGeneralID, attackerUnitType := "wei", "caocao", "huWei"
			defenderFaction, defenderGeneralID, defenderUnitType := "wu", "opponent", "wuInfantry"
			generalIDs := []string{"caocao"}
			if ownerSide == "defender" {
				attackerFaction, attackerGeneralID, attackerUnitType = "wu", "opponent", "wuInfantry"
				defenderFaction, defenderGeneralID, defenderUnitType = "wei", "caocao", "huWei"
				generalIDs = nil
			}
			svc, repo, attacker, defender := newPvpTestServiceForGenerals(t, attackerFaction, attackerGeneralID, defenderFaction, defenderGeneralID)
			unitsMu.Lock()
			activeUnits["wei"]["huWei"] = UnitConfig{
				Name: "虎卫", Category: "infantry",
				Stats: map[string]int{"attack": 10, "infantryDefense": 10, "cavalryDefense": 8, "carryCapacity": 5, "upkeep": 1},
			}
			unitsMu.Unlock()
			attacker.Army = []ArmyUnit{{UnitType: attackerUnitType, Amount: 100}}
			defender.Army = []ArmyUnit{{UnitType: defenderUnitType, Amount: 100}}
			defender.Buildings = nil
			repo.players[attacker.Player.ID] = attacker
			repo.players[defender.Player.ID] = defender

			started, err := svc.StartPvpAttack(PvpAttackRequest{
				PlayerID: attacker.Player.ID, TargetPlayerID: defender.Player.ID, MarchMode: PvpMarchTypeAttack,
				Troops: map[string]int{attackerUnitType: 100}, GeneralIDs: generalIDs,
			})
			if err != nil {
				t.Fatalf("StartPvpAttack failed: %v", err)
			}
			forcePvpMarchDue(t, repo, started.March.ID)
			battle, err := svc.ResolvePvpMarch(started.March.ID)
			if err != nil {
				t.Fatalf("ResolvePvpMarch failed: %v", err)
			}
			powerKey := "attackerPower"
			if ownerSide == "defender" {
				powerKey = "defensePower"
			}
			wantPower := float64(1000)
			if ownerSide == "defender" {
				wantPower = 1200
			}
			if got, ok := battle.Result[powerKey].(float64); !ok || got != wantPower {
				t.Fatalf("expected %s %.0f, got %+v", powerKey, wantPower, battle.Result[powerKey])
			}

			attackerReports, _, err := repo.ListReports(attacker.Player.ID, 10, 0)
			if err != nil || len(attackerReports) == 0 || attackerReports[0].ID != battle.AttackerReportID {
				t.Fatalf("expected attacker report, reports=%+v err=%v", attackerReports, err)
			}
			defenderReports, _, err := repo.ListReports(defender.Player.ID, 10, 0)
			if err != nil || len(defenderReports) == 0 || defenderReports[0].ID != battle.DefenderReportID {
				t.Fatalf("expected defender report, reports=%+v err=%v", defenderReports, err)
			}
			for _, report := range []BattleReport{attackerReports[0], defenderReports[0]} {
				outcome, ok := report.TraitOutcomes["weiwu_tongyu"]
				if !ok || outcome.OwnerSide != ownerSide || outcome.OwnerGeneralID != "caocao" {
					t.Fatalf("expected Weiwu Tongyu owned by %s/caocao, got %+v", ownerSide, report.TraitOutcomes)
				}
				infantry, infantryOK := outcome.Detail["infantryDefenseModifiedUnits"].(map[string]int)
				cavalry, cavalryOK := outcome.Detail["cavalryDefenseModifiedUnits"].(map[string]int)
				defenseBonusRate, defenseRateOK := outcome.Detail["defenseBonusRate"].(float64)
				if !infantryOK || !cavalryOK || !defenseRateOK || defenseBonusRate != 0.15 || infantry["huWei"] != 2 || cavalry["huWei"] != 1 {
					t.Fatalf("expected actual HuWei defense deltas +2/+1, got %+v", outcome)
				}
				standardFound := false
				for _, trait := range report.Detail.Traits {
					if trait.TraitID != "weiwu_tongyu" || trait.GeneralID != "caocao" || trait.OwnerRole != ownerSide {
						continue
					}
					standardInfantry, infantryOK := trait.Detail["infantryDefenseModifiedUnits"].(map[string]int)
					standardCavalry, cavalryOK := trait.Detail["cavalryDefenseModifiedUnits"].(map[string]int)
					standardDefenseRate, defenseRateOK := trait.Detail["defenseBonusRate"].(float64)
					if !infantryOK || !cavalryOK || !defenseRateOK || standardDefenseRate != 0.15 || standardInfantry["huWei"] != 2 || standardCavalry["huWei"] != 1 {
						t.Fatalf("expected standard report HuWei defense deltas +2/+1, got %+v", trait.Detail)
					}
					standardFound = true
				}
				if !standardFound {
					t.Fatalf("expected Weiwu Tongyu in standard report, got %+v", report.Detail.Traits)
				}
			}

			attackerLosses := pvpTestLossesFromBattle(t, battle, "attacker")
			defenderLosses := pvpTestLossesFromBattle(t, battle, "defender")
			if attackerReports[0].LostUnits[attackerUnitType] != attackerLosses[attackerUnitType] || defenderReports[0].LostUnits[defenderUnitType] != defenderLosses[defenderUnitType] {
				t.Fatalf("expected own report losses to match battle, battle=%+v reports=%+v/%+v", battle.Losses, attackerReports[0].LostUnits, defenderReports[0].LostUnits)
			}
			storedDefender, err := repo.GetState(defender.Player.ID)
			if err != nil {
				t.Fatalf("GetState defender failed: %v", err)
			}
			if got, want := armySliceToMap(storedDefender.Army)[defenderUnitType], 100-defenderLosses[defenderUnitType]; got != want {
				t.Fatalf("expected defender state %d, got %d", want, got)
			}
			storedMarch, err := repo.GetPvpMarch(started.March.ID)
			if err != nil {
				t.Fatalf("GetPvpMarch failed: %v", err)
			}
			if got, want := storedMarch.AttackTroops[attackerUnitType], 100-attackerLosses[attackerUnitType]; got != want {
				t.Fatalf("expected attacker return %d, got %d", want, got)
			}
		})
	}
}

// TestPvpFormalTraitSuppressorsPreventRealEnemyDamage 验证卧龙奇谋和苦肉计在攻守双方都真实阻止敌方特性，而不只写压制战报。
func TestPvpFormalTraitSuppressorsPreventRealEnemyDamage(t *testing.T) {
	cases := []struct {
		name        string
		generalID   string
		generalName string
		faction     string
		traitID     string
		traitType   string
	}{
		{name: "诸葛亮卧龙奇谋", generalID: "zhugeliang", generalName: "诸葛亮", faction: "shu", traitID: "wolong_mouzhi", traitType: general.TraitTypeBonus},
		{name: "黄盖苦肉计", generalID: "huanggai", generalName: "黄盖", faction: "wu", traitID: "kurouji", traitType: general.TraitTypeSpecial},
	}
	type runResult struct {
		battle          PvpBattle
		attackerReport  BattleReport
		defenderReport  BattleReport
		actorUnitType   string
		actorLoss       int
		actualRemaining int
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			for _, ownerSide := range []string{"attacker", "defender"} {
				t.Run(ownerSide, func(t *testing.T) {
					run := func(suppressionEnabled bool) runResult {
						opponentFaction := "wei"
						if tc.faction == opponentFaction {
							opponentFaction = "wu"
						}
						suppressTrait := GeneralTraitConfig{
							TraitID: tc.traitID, TraitType: tc.traitType, Enabled: true, Scope: "enemy_traits",
							Params: map[string]float64{"disableTraitCount": 1, "triggerChance": 0},
						}
						if suppressionEnabled {
							suppressTrait.Params["triggerChance"] = 1
						}
						suppressor := GeneralHeroConfig{ID: tc.generalID, Name: tc.generalName, Faction: tc.faction, Enabled: true}
						if tc.traitType == general.TraitTypeSpecial {
							suppressor.SpecialTrait = suppressTrait
						} else {
							suppressor.BonusTrait = suppressTrait
						}
						setTestFactionsAndGenerals(t, FactionsConfig{
							tc.faction:      {Name: tc.faction, Generals: []GeneralInfo{{ID: tc.generalID, Name: tc.generalName}}},
							opponentFaction: {Name: opponentFaction, Generals: []GeneralInfo{{ID: "damage_general", Name: "增伤对手"}}},
						}, GeneralsConfig{Enabled: true, Heroes: map[string]GeneralHeroConfig{
							tc.generalID: suppressor,
							"damage_general": {
								ID: "damage_general", Name: "增伤对手", Faction: opponentFaction, Enabled: true,
								BonusTrait: GeneralTraitConfig{
									TraitID: "laodang_yizhuang", TraitType: general.TraitTypeBonus, Enabled: true, Scope: "enemy_army",
									Params: map[string]float64{"effectRate": 0.5, "triggerChance": 1},
								},
							},
						}})

						attackerFaction, attackerGeneralID := tc.faction, tc.generalID
						defenderFaction, defenderGeneralID := opponentFaction, "damage_general"
						attackerCount, defenderCount := 1000, 100
						generalIDs := []string{tc.generalID}
						if ownerSide == "defender" {
							attackerFaction, attackerGeneralID = opponentFaction, "damage_general"
							defenderFaction, defenderGeneralID = tc.faction, tc.generalID
							attackerCount, defenderCount = 100, 1000
							generalIDs = []string{"damage_general"}
						}
						svc, repo, attacker, defender := newPvpTestServiceForGenerals(t, attackerFaction, attackerGeneralID, defenderFaction, defenderGeneralID)
						attackerUnitType := attackerFaction + "Infantry"
						defenderUnitType := defenderFaction + "Infantry"
						attacker.Army = []ArmyUnit{{UnitType: attackerUnitType, Amount: attackerCount}}
						defender.Army = []ArmyUnit{{UnitType: defenderUnitType, Amount: defenderCount}}
						defender.Buildings = nil
						repo.players[attacker.Player.ID] = attacker
						repo.players[defender.Player.ID] = defender

						started, err := svc.StartPvpAttack(PvpAttackRequest{
							PlayerID: attacker.Player.ID, TargetPlayerID: defender.Player.ID, MarchMode: PvpMarchTypeAttack,
							Troops: map[string]int{attackerUnitType: attackerCount}, GeneralIDs: generalIDs,
						})
						if err != nil {
							t.Fatalf("StartPvpAttack failed: %v", err)
						}
						forcePvpMarchDue(t, repo, started.March.ID)
						battle, err := svc.ResolvePvpMarch(started.March.ID)
						if err != nil {
							t.Fatalf("ResolvePvpMarch failed: %v", err)
						}
						attackerReports, _, err := repo.ListReports(attacker.Player.ID, 10, 0)
						if err != nil || len(attackerReports) == 0 || attackerReports[0].ID != battle.AttackerReportID {
							t.Fatalf("expected attacker report, reports=%+v err=%v", attackerReports, err)
						}
						defenderReports, _, err := repo.ListReports(defender.Player.ID, 10, 0)
						if err != nil || len(defenderReports) == 0 || defenderReports[0].ID != battle.DefenderReportID {
							t.Fatalf("expected defender report, reports=%+v err=%v", defenderReports, err)
						}
						actorUnitType := attackerUnitType
						actorLoss := pvpTestLossesFromBattle(t, battle, "attacker")[actorUnitType]
						storedMarch, err := repo.GetPvpMarch(started.March.ID)
						if err != nil {
							t.Fatalf("GetPvpMarch failed: %v", err)
						}
						actualRemaining := storedMarch.AttackTroops[actorUnitType]
						if ownerSide == "defender" {
							actorUnitType = defenderUnitType
							actorLoss = pvpTestLossesFromBattle(t, battle, "defender")[actorUnitType]
							storedDefender, err := repo.GetState(defender.Player.ID)
							if err != nil {
								t.Fatalf("GetState defender failed: %v", err)
							}
							actualRemaining = armySliceToMap(storedDefender.Army)[actorUnitType]
						}
						return runResult{
							battle: battle, attackerReport: attackerReports[0], defenderReport: defenderReports[0],
							actorUnitType: actorUnitType, actorLoss: actorLoss, actualRemaining: actualRemaining,
						}
					}

					control := run(false)
					controlOutcome, ok := control.attackerReport.TraitOutcomes["laodang_yizhuang"]
					if !ok {
						t.Fatalf("expected control damage trait, got %+v", control.attackerReport.TraitOutcomes)
					}
					controlExtra, ok := controlOutcome.Detail["extraLosses"].(map[string]int)
					if !ok || controlExtra[control.actorUnitType] <= 0 {
						t.Fatalf("expected control extra losses for %s, got %+v", control.actorUnitType, controlOutcome)
					}
					suppressed := run(true)
					if suppressed.actorUnitType != control.actorUnitType || control.actorLoss-suppressed.actorLoss != controlExtra[control.actorUnitType] {
						t.Fatalf("expected suppression to remove exactly %d losses, control=%d suppressed=%d", controlExtra[control.actorUnitType], control.actorLoss, suppressed.actorLoss)
					}
					if suppressed.actualRemaining != 1000-suppressed.actorLoss {
						t.Fatalf("expected real actor troops %d, got %d", 1000-suppressed.actorLoss, suppressed.actualRemaining)
					}
					for _, report := range []BattleReport{suppressed.attackerReport, suppressed.defenderReport} {
						if _, triggered := report.TraitOutcomes["laodang_yizhuang"]; triggered {
							t.Fatalf("expected enemy damage trait suppressed, got %+v", report.TraitOutcomes)
						}
						outcome, ok := report.TraitOutcomes[tc.traitID]
						if !ok || outcome.OwnerSide != ownerSide || outcome.OwnerGeneralID != tc.generalID {
							t.Fatalf("expected %s owned by %s/%s, got %+v", tc.traitID, ownerSide, tc.generalID, report.TraitOutcomes)
						}
						if tc.traitID == "wolong_mouzhi" {
							if outcome.Detail["disabledGeneralCount"] != 1 || outcome.Detail["disabledTraitCount"] != 1 || outcome.Detail["triggerChance"] != float64(1) {
								t.Fatalf("expected Wolong to disable all trigger traits of one enemy general, got %+v", outcome)
							}
						} else if outcome.Detail["disableTraitCount"] != 1 || outcome.Detail["disabledTraitCount"] != 1 || outcome.Detail["triggerChance"] != float64(1) {
							t.Fatalf("expected one disabled follow-up trait, got %+v", outcome)
						}
						standardFound := false
						for _, trait := range report.Detail.Traits {
							if trait.TraitID == "laodang_yizhuang" {
								t.Fatalf("expected standard report to omit suppressed damage, got %+v", report.Detail.Traits)
							}
							if trait.TraitID == tc.traitID && trait.GeneralID == tc.generalID && trait.OwnerRole == ownerSide {
								if tc.traitID == "wolong_mouzhi" {
									if trait.Detail["disabledGeneralCount"] != 1 || trait.Detail["disabledTraitCount"] != 1 || trait.Detail["triggerChance"] != float64(1) {
										t.Fatalf("expected standard report all enemy trigger traits disabled, got %+v", trait.Detail)
									}
								} else if trait.Detail["disableTraitCount"] != 1 || trait.Detail["disabledTraitCount"] != 1 || trait.Detail["triggerChance"] != float64(1) {
									t.Fatalf("expected standard report one disabled follow-up trait, got %+v", trait.Detail)
								}
								standardFound = true
							}
						}
						if !standardFound {
							t.Fatalf("expected %s in standard report, got %+v", tc.traitID, report.Detail.Traits)
						}
					}
					ownReportLoss := suppressed.attackerReport.LostUnits[suppressed.actorUnitType]
					if ownerSide == "defender" {
						ownReportLoss = suppressed.defenderReport.LostUnits[suppressed.actorUnitType]
					}
					if ownReportLoss != suppressed.actorLoss {
						t.Fatalf("expected owner report loss %d, got %d", suppressed.actorLoss, ownReportLoss)
					}
				})
			}
		})
	}
}

// TestPvpJiangdongGushouOnlyTriggersForDefender 验证同一防御特性只由防守将领触发。
func TestPvpJiangdongGushouOnlyTriggersForDefender(t *testing.T) {
	defenseTrait := GeneralTraitConfig{
		TraitID:      "jiangdong_gushou",
		TraitType:    general.TraitTypeBonus,
		Enabled:      true,
		Scope:        "self_army",
		AllowedSides: []string{"defender", "reinforcement"},
		Params:       map[string]float64{"defenseBonusRate": 0.5, "triggerChance": 1},
	}
	setTestFactionsAndGenerals(t, FactionsConfig{
		"wei": {Name: "魏国", Generals: []GeneralInfo{{ID: "caocao", Name: "曹操"}}},
		"shu": {Name: "蜀国", Generals: []GeneralInfo{{ID: "liubei", Name: "刘备"}}},
	}, GeneralsConfig{
		Enabled: true,
		Heroes: map[string]GeneralHeroConfig{
			"caocao": {ID: "caocao", Name: "曹操", Faction: "wei", Enabled: true, BonusTrait: defenseTrait},
			"liubei": {ID: "liubei", Name: "刘备", Faction: "shu", Enabled: true, BonusTrait: defenseTrait},
		},
	})
	svc, repo, attacker, defender := newPvpTestService(t)
	attacker.Army = []ArmyUnit{{UnitType: "weiInfantry", Amount: 100}}
	defender.Army = []ArmyUnit{{UnitType: "weiInfantry", Amount: 100}}
	repo.players[attacker.Player.ID] = attacker
	repo.players[defender.Player.ID] = defender

	started, err := svc.StartPvpAttack(PvpAttackRequest{
		PlayerID: attacker.Player.ID, TargetPlayerID: defender.Player.ID,
		MarchMode: PvpMarchTypeAttack, Troops: map[string]int{"weiInfantry": 50}, GeneralIDs: []string{"caocao"},
	})
	if err != nil {
		t.Fatalf("StartPvpAttack failed: %v", err)
	}
	forcePvpMarchDue(t, repo, started.March.ID)
	battle, err := svc.ResolvePvpMarch(started.March.ID)
	if err != nil {
		t.Fatalf("ResolvePvpMarch failed: %v", err)
	}
	reports, _, err := repo.ListReports(attacker.Player.ID, 10, 0)
	if err != nil || len(reports) == 0 || reports[0].ID != battle.AttackerReportID {
		t.Fatalf("expected attacker report, reports=%+v err=%v", reports, err)
	}
	outcome, ok := reports[0].TraitOutcomes["jiangdong_gushou"]
	if !ok {
		t.Fatalf("expected defender's jiangdong_gushou in report, outcomes=%+v", reports[0].TraitOutcomes)
	}
	if outcome.OwnerSide != "defender" || outcome.OwnerGeneralID != "liubei" {
		t.Fatalf("expected only defender general to trigger, got %+v", outcome)
	}
}

// TestPvpEnemyDefenseReductionAndDefenderBonusStackWithActualDeltas 验证攻方破防与守方加防按真实顺序叠加并写入双方战报。
func TestPvpEnemyDefenseReductionAndDefenderBonusStackWithActualDeltas(t *testing.T) {
	setTestFactionsAndGenerals(t, FactionsConfig{
		"wei": {Name: "魏国", Generals: []GeneralInfo{{ID: "breaker", Name: "破防将领"}}},
		"wu":  {Name: "吴国", Generals: []GeneralInfo{{ID: "guardian", Name: "守城将领"}}},
	}, GeneralsConfig{
		Enabled: true,
		Heroes: map[string]GeneralHeroConfig{
			"breaker": {
				ID: "breaker", Name: "破防将领", Faction: "wei", Enabled: true,
				SpecialTrait: GeneralTraitConfig{TraitID: "huchi_chongzhen", TraitType: general.TraitTypeSpecial, Enabled: true, Scope: "enemy_army", AllowedSides: []string{"attacker"}, Params: map[string]float64{"enemyDefenseReductionRate": 0.3, "triggerChance": 1}},
			},
			"guardian": {
				ID: "guardian", Name: "守城将领", Faction: "wu", Enabled: true,
				BonusTrait: GeneralTraitConfig{TraitID: "jiangdong_gushou", TraitType: general.TraitTypeBonus, Enabled: true, Scope: "self_army", AllowedSides: []string{"defender"}, Params: map[string]float64{"defenseBonusRate": 0.5, "triggerChance": 1}},
			},
		},
	})
	svc, repo, attacker, defender := newPvpTestServiceForGenerals(t, "wei", "breaker", "wu", "guardian")
	attacker.Army = []ArmyUnit{{UnitType: "weiInfantry", Amount: 100}}
	defender.Army = []ArmyUnit{{UnitType: "wuInfantry", Amount: 100}}
	defender.Buildings = nil
	repo.players[attacker.Player.ID] = attacker
	repo.players[defender.Player.ID] = defender

	started, err := svc.StartPvpAttack(PvpAttackRequest{
		PlayerID: attacker.Player.ID, TargetPlayerID: defender.Player.ID,
		MarchMode: PvpMarchTypePlunder, Troops: map[string]int{"weiInfantry": 100}, GeneralIDs: []string{"breaker"},
	})
	if err != nil {
		t.Fatalf("StartPvpAttack failed: %v", err)
	}
	forcePvpMarchDue(t, repo, started.March.ID)
	battle, err := svc.ResolvePvpMarch(started.March.ID)
	if err != nil {
		t.Fatalf("ResolvePvpMarch failed: %v", err)
	}
	defensePower, ok := battle.Result["defensePower"].(float64)
	if !ok || defensePower != 1100 {
		t.Fatalf("expected defense 10 -> 7 -> 11 and power 1100, got %+v", battle.Result["defensePower"])
	}

	attackerReports, _, err := repo.ListReports(attacker.Player.ID, 10, 0)
	if err != nil || len(attackerReports) == 0 || attackerReports[0].ID != battle.AttackerReportID {
		t.Fatalf("expected attacker report, reports=%+v err=%v", attackerReports, err)
	}
	defenderReports, _, err := repo.ListReports(defender.Player.ID, 10, 0)
	if err != nil || len(defenderReports) == 0 || defenderReports[0].ID != battle.DefenderReportID {
		t.Fatalf("expected defender report, reports=%+v err=%v", defenderReports, err)
	}
	for _, report := range []BattleReport{attackerReports[0], defenderReports[0]} {
		breakerOutcome, breakerOK := report.TraitOutcomes["huchi_chongzhen"]
		guardianOutcome, guardianOK := report.TraitOutcomes["jiangdong_gushou"]
		if !breakerOK || !guardianOK || breakerOutcome.OwnerSide != "attacker" || guardianOutcome.OwnerSide != "defender" {
			t.Fatalf("expected attacker reduction and defender bonus in both reports, outcomes=%+v", report.TraitOutcomes)
		}
		breakerInfantry, breakerInfantryOK := breakerOutcome.Detail["infantryDefenseModifiedUnits"].(map[string]int)
		breakerCavalry, breakerCavalryOK := breakerOutcome.Detail["cavalryDefenseModifiedUnits"].(map[string]int)
		guardianInfantry, guardianInfantryOK := guardianOutcome.Detail["infantryDefenseModifiedUnits"].(map[string]int)
		guardianCavalry, guardianCavalryOK := guardianOutcome.Detail["cavalryDefenseModifiedUnits"].(map[string]int)
		if !breakerInfantryOK || !breakerCavalryOK || !guardianInfantryOK || !guardianCavalryOK ||
			breakerInfantry["wuInfantry"] != -3 || breakerCavalry["wuInfantry"] != -2 ||
			guardianInfantry["wuInfantry"] != 4 || guardianCavalry["wuInfantry"] != 3 {
			t.Fatalf("expected report deltas -3/-2 then +4/+3, breaker=%+v guardian=%+v", breakerOutcome, guardianOutcome)
		}
	}
}

// TestPvpBothSidesSuppressionResolveSimultaneously 验证真实 PVP 中双方压制同时生效并同步双方战报与军队状态。
func TestPvpBothSidesSuppressionResolveSimultaneously(t *testing.T) {
	setTestFactionsAndGenerals(t, FactionsConfig{
		"wu":  {Name: "吴国", Generals: []GeneralInfo{{ID: "attacker_general", Name: "进攻将领"}}},
		"shu": {Name: "蜀国", Generals: []GeneralInfo{{ID: "defender_general", Name: "防守将领"}}},
	}, GeneralsConfig{
		Enabled: true,
		Heroes: map[string]GeneralHeroConfig{
			"attacker_general": {
				ID: "attacker_general", Name: "进攻将领", Faction: "wu", Enabled: true,
				SpecialTrait: GeneralTraitConfig{TraitID: "kurouji", TraitType: general.TraitTypeSpecial, Enabled: true, Scope: "enemy_traits", Params: map[string]float64{"disableTraitCount": 1, "triggerChance": 1}},
				BonusTrait:   GeneralTraitConfig{TraitID: "kurou_fanji", TraitType: general.TraitTypeBonus, Enabled: true, Scope: "enemy_army", Params: map[string]float64{"effectRate": 0.1, "triggerChance": 1}},
			},
			"defender_general": {
				ID: "defender_general", Name: "防守将领", Faction: "shu", Enabled: true,
				SpecialTrait: GeneralTraitConfig{TraitID: "kurouji", TraitType: general.TraitTypeSpecial, Enabled: true, Scope: "enemy_traits", Params: map[string]float64{"disableTraitCount": 1, "triggerChance": 1}},
				BonusTrait:   GeneralTraitConfig{TraitID: "laodang_yizhuang", TraitType: general.TraitTypeBonus, Enabled: true, Scope: "enemy_army", Params: map[string]float64{"effectRate": 0.1, "triggerChance": 1}},
			},
		},
	})
	svc, repo, attacker, defender := newPvpTestServiceForGenerals(t, "wu", "attacker_general", "shu", "defender_general")
	attacker.Army = []ArmyUnit{{UnitType: "wuInfantry", Amount: 100}}
	defender.Army = []ArmyUnit{{UnitType: "shuInfantry", Amount: 100}}
	repo.players[attacker.Player.ID] = attacker
	repo.players[defender.Player.ID] = defender

	started, err := svc.StartPvpAttack(PvpAttackRequest{
		PlayerID: attacker.Player.ID, TargetPlayerID: defender.Player.ID,
		MarchMode: PvpMarchTypePlunder, Troops: map[string]int{"wuInfantry": 100}, GeneralIDs: []string{"attacker_general"},
	})
	if err != nil {
		t.Fatalf("StartPvpAttack failed: %v", err)
	}
	forcePvpMarchDue(t, repo, started.March.ID)
	battle, err := svc.ResolvePvpMarch(started.March.ID)
	if err != nil {
		t.Fatalf("ResolvePvpMarch failed: %v", err)
	}

	attackerReports, _, err := repo.ListReports(attacker.Player.ID, 10, 0)
	if err != nil || len(attackerReports) == 0 || attackerReports[0].ID != battle.AttackerReportID {
		t.Fatalf("expected attacker report, reports=%+v err=%v", attackerReports, err)
	}
	defenderReports, _, err := repo.ListReports(defender.Player.ID, 10, 0)
	if err != nil || len(defenderReports) == 0 || defenderReports[0].ID != battle.DefenderReportID {
		t.Fatalf("expected defender report, reports=%+v err=%v", defenderReports, err)
	}
	for _, report := range []BattleReport{attackerReports[0], defenderReports[0]} {
		suppressionSides := map[string]bool{}
		for _, outcome := range report.TraitOutcomes {
			if outcome.TraitID == "kurouji" {
				suppressionSides[outcome.OwnerSide] = true
				if outcome.Detail["disableTraitCount"] != 1 || outcome.Detail["disabledTraitCount"] != 1 {
					t.Fatalf("expected each Kurouji to suppress exactly one lower-priority enemy trait, report=%s outcome=%+v", report.ID, outcome)
				}
			}
			if outcome.TraitID == "kurou_fanji" || outcome.TraitID == "laodang_yizhuang" {
				t.Fatalf("expected both lower-priority damage traits suppressed, outcomes=%+v", report.TraitOutcomes)
			}
		}
		if !suppressionSides["attacker"] || !suppressionSides["defender"] {
			t.Fatalf("expected both suppression outcomes in report, sides=%+v outcomes=%+v", suppressionSides, report.TraitOutcomes)
		}
		if report.Detail == nil || len(report.Detail.Traits) != 2 {
			t.Fatalf("expected two formal suppression entries in standard timeline, report=%s detail=%+v", report.ID, report.Detail)
		}
		for index, wantRole := range []string{"attacker", "defender"} {
			trait := report.Detail.Traits[index]
			if trait.TraitID != "kurouji" || trait.OwnerRole != wantRole || trait.Detail["disableTraitCount"] != 1 || trait.Detail["disabledTraitCount"] != 1 {
				t.Fatalf("expected ordered attacker/defender Kurouji entries with actual count 1, report=%s index=%d trait=%+v", report.ID, index, trait)
			}
		}
	}

	attackerLosses := pvpTestLossesFromBattle(t, battle, "attacker")
	defenderLosses := pvpTestLossesFromBattle(t, battle, "defender")
	if battle.Result["attackerPower"] != float64(1000) || battle.Result["defensePower"] != float64(1020) ||
		attackerLosses["wuInfantry"] != 50 || defenderLosses["shuInfantry"] != 49 {
		t.Fatalf("expected 1000/1020 power with public wall bonus and unchanged 50/49 core losses after mutual suppression, result=%+v losses=%+v", battle.Result, battle.Losses)
	}
	if attackerReports[0].LostUnits["wuInfantry"] != attackerLosses["wuInfantry"] || defenderReports[0].LostUnits["shuInfantry"] != defenderLosses["shuInfantry"] {
		t.Fatalf("expected reports to match battle losses, battle=%+v reports=%+v/%+v", battle.Losses, attackerReports[0].LostUnits, defenderReports[0].LostUnits)
	}
	storedMarch, err := repo.GetPvpMarch(started.March.ID)
	if err != nil {
		t.Fatalf("GetPvpMarch failed: %v", err)
	}
	storedDefender, err := repo.GetState(defender.Player.ID)
	if err != nil {
		t.Fatalf("GetState defender failed: %v", err)
	}
	if got, want := storedMarch.AttackTroops["wuInfantry"], 100-attackerLosses["wuInfantry"]; got != want {
		t.Fatalf("expected attacker march army %d, got %d", want, got)
	}
	if got, want := armySliceToMap(storedDefender.Army)["shuInfantry"], 100-defenderLosses["shuInfantry"]; got != want {
		t.Fatalf("expected defender army %d, got %d", want, got)
	}
	storedAttacker, err := repo.GetState(attacker.Player.ID)
	if err != nil {
		t.Fatalf("GetState attacker failed: %v", err)
	}
	if pvpTestGeneralExp(storedAttacker, "attacker_general") != 49 || pvpTestGeneralExp(storedDefender, "defender_general") != 50 ||
		attackerReports[0].GeneralExpGained != 49 || defenderReports[0].GeneralExpGained != 50 {
		t.Fatalf("expected generals to gain only the opponent's real 49/50 core losses, states=%+v/%+v reports=%d/%d", storedAttacker.Generals, storedDefender.Generals, attackerReports[0].GeneralExpGained, defenderReports[0].GeneralExpGained)
	}
}

// TestPvpPlunderReportUsesAttackView 验证 PVP 掠夺仍使用进攻视角标准详情。
func TestPvpPlunderReportUsesAttackView(t *testing.T) {
	_, _, attacker, defender := newPvpTestService(t)
	now := time.Now().UTC().Format(resourceDateLayout)
	march := &PvpMarch{ID: "march_plunder_report", MarchType: PvpMarchTypePlunder}
	report := NormalizeBattleReport(buildPvpBattleReport(
		"br_pvp_plunder",
		&attacker,
		&defender,
		march,
		"attacker_victory",
		120,
		80,
		map[string]int{"weiInfantry": 40},
		map[string]int{"weiInfantry": 4},
		map[string]int{"shuInfantry": 20},
		map[string]int{"shuInfantry": 10},
		map[string]int{"wood": 120},
		now,
		PvpMarchTypePlunder,
	))
	if report.ViewType != ReportViewAttack || report.BattleType != PvpMarchTypePlunder {
		t.Fatalf("expected plunder report to use attack view and plunder battle type, got view=%s battle=%s", report.ViewType, report.BattleType)
	}
	if report.Detail == nil || report.Detail.PrimarySide.Role != "attacker" || report.Detail.SecondarySide == nil || report.Detail.SecondarySide.Role != "defender" {
		t.Fatalf("expected attack-style detail for plunder report, got %+v", report.Detail)
	}
	if report.Detail.Rewards.Resources["wood"] != 120 {
		t.Fatalf("expected plundered resources in rewards snapshot, got %+v", report.Detail.Rewards)
	}
}

// TestPvpPlunderTraitsSettleFinalResourcesAndReports 验证甘宁增益和孙权减益后的最终掠夺值真实转移并写入双方战报。
func TestPvpPlunderTraitsSettleFinalResourcesAndReports(t *testing.T) {
	setTestFactionsAndGenerals(t, FactionsConfig{
		"wu":  {Name: "吴国", Generals: []GeneralInfo{{ID: "ganning", Name: "甘宁"}}},
		"wei": {Name: "魏国", Generals: []GeneralInfo{{ID: "sunquan", Name: "孙权"}}},
	}, GeneralsConfig{
		Enabled: true,
		Heroes: map[string]GeneralHeroConfig{
			"ganning": {
				ID: "ganning", Name: "甘宁", Faction: "wu", Enabled: true,
				SpecialTrait: GeneralTraitConfig{
					TraitID: "jinfan_jielue", TraitType: general.TraitTypeSpecial, Enabled: true,
					Scope: "self_plunder", AllowedSides: []string{"attacker"}, AllowedScenes: []string{"plunder"},
					Params: map[string]float64{"plunderBonusRate": 0.2, "triggerChance": 1},
				},
			},
			"sunquan": {
				ID: "sunquan", Name: "孙权", Faction: "wei", Enabled: true,
				SpecialTrait: GeneralTraitConfig{
					TraitID: "jiangdong_haoling", TraitType: general.TraitTypeSpecial, Enabled: true,
					Scope: "enemy_plunder", AllowedSides: []string{"defender"}, AllowedScenes: []string{"plunder"},
					Params: map[string]float64{"plunderBonusRate": -0.2, "triggerChance": 1},
				},
			},
		},
	})
	svc, repo, attacker, defender := newPvpTestServiceForGenerals(t, "wu", "ganning", "wei", "sunquan")
	attacker.Army = []ArmyUnit{{UnitType: "wuInfantry", Amount: 1000}}
	attacker.Resources.Items = map[string]int{"wood": 0, "stone": 0, "iron": 0, "food": 0}
	attacker.Resources.Capacity = map[string]int{"wood": 100000, "stone": 100000, "iron": 100000, "food": 100000}
	defender.Army = []ArmyUnit{{UnitType: "weiInfantry", Amount: 10}}
	defender.Resources.Items = map[string]int{"wood": 10000, "stone": 0, "iron": 0, "food": 0}
	defender.Resources.Capacity = map[string]int{"wood": 0, "stone": 0, "iron": 0, "food": 0}
	nowText := time.Now().UTC().Format(resourceDateLayout)
	attacker.ResourceSettledAt = nowText
	defender.ResourceSettledAt = nowText
	repo.players[attacker.Player.ID] = attacker
	repo.players[defender.Player.ID] = defender

	started, err := svc.StartPvpAttack(PvpAttackRequest{
		PlayerID: attacker.Player.ID, TargetPlayerID: defender.Player.ID, MarchMode: PvpMarchTypePlunder,
		Troops: map[string]int{"wuInfantry": 1000}, GeneralIDs: []string{"ganning"},
	})
	if err != nil {
		t.Fatalf("StartPvpAttack failed: %v", err)
	}
	forcePvpMarchDue(t, repo, started.March.ID)
	battle, err := svc.ResolvePvpMarch(started.March.ID)
	if err != nil {
		t.Fatalf("ResolvePvpMarch failed: %v", err)
	}
	if battle.Plunder["wood"] <= 0 {
		t.Fatalf("expected successful plunder, got %+v", battle.Plunder)
	}
	attackerReports, _, err := repo.ListReports(attacker.Player.ID, 10, 0)
	if err != nil || len(attackerReports) == 0 || attackerReports[0].ID != battle.AttackerReportID {
		t.Fatalf("expected attacker report, reports=%+v err=%v", attackerReports, err)
	}
	defenderReports, _, err := repo.ListReports(defender.Player.ID, 10, 0)
	if err != nil || len(defenderReports) == 0 || defenderReports[0].ID != battle.DefenderReportID {
		t.Fatalf("expected defender report, reports=%+v err=%v", defenderReports, err)
	}
	attackerReport := attackerReports[0]
	if attackerReport.Rewards["wood"] != battle.Plunder["wood"] || attackerReport.Detail.Rewards.Resources["wood"] != battle.Plunder["wood"] {
		t.Fatalf("expected legacy and standard report rewards to match battle, battle=%+v report=%+v detail=%+v", battle.Plunder, attackerReport.Rewards, attackerReport.Detail.Rewards.Resources)
	}
	ganningOutcome, ganningOK := attackerReport.TraitOutcomes["jinfan_jielue"]
	sunquanOutcome, sunquanOK := attackerReport.TraitOutcomes["jiangdong_haoling"]
	if !ganningOK || !sunquanOK {
		t.Fatalf("expected both plunder traits in report, got %+v", attackerReport.TraitOutcomes)
	}
	ganningValues, ganningDetailOK := ganningOutcome.Detail["plunderDelta"].(map[string]int)
	sunquanValues, sunquanDetailOK := sunquanOutcome.Detail["plunderDelta"].(map[string]int)
	if !ganningDetailOK || !sunquanDetailOK {
		t.Fatalf("expected structured plunder deltas, ganning=%+v sunquan=%+v", ganningOutcome, sunquanOutcome)
	}
	ganningDelta := ganningValues["wood"]
	sunquanDelta := sunquanValues["wood"]
	if ganningDelta <= 0 || sunquanDelta >= 0 {
		t.Fatalf("expected positive Ganning and negative Sun Quan deltas, got ganning=%d sunquan=%d", ganningDelta, sunquanDelta)
	}
	for _, report := range []BattleReport{attackerReport, defenderReports[0]} {
		expectations := map[string]struct {
			rate  float64
			delta int
		}{
			"jinfan_jielue":     {rate: 0.2, delta: ganningDelta},
			"jiangdong_haoling": {rate: -0.2, delta: sunquanDelta},
		}
		for traitID, expectation := range expectations {
			outcome := report.TraitOutcomes[traitID]
			values, ok := outcome.Detail["plunderDelta"].(map[string]int)
			if !ok || values["wood"] != expectation.delta || outcome.Detail["plunderBonusRate"] != expectation.rate {
				t.Fatalf("expected %s design rate %.1f and actual delta %d in both reports, got %+v", traitID, expectation.rate, expectation.delta, outcome)
			}
			standardMatched := false
			for _, trait := range report.Detail.Traits {
				if trait.TraitID != traitID {
					continue
				}
				standardValues, standardOK := trait.Detail["plunderDelta"].(map[string]int)
				if !standardOK || standardValues["wood"] != expectation.delta || trait.Detail["plunderBonusRate"] != expectation.rate {
					t.Fatalf("expected standard %s design rate %.1f and actual delta %d, got %+v", traitID, expectation.rate, expectation.delta, trait.Detail)
				}
				standardMatched = true
			}
			if !standardMatched {
				t.Fatalf("expected %s in standard report, got %+v", traitID, report.Detail.Traits)
			}
		}
	}
	storedAttacker, err := repo.GetState(attacker.Player.ID)
	if err != nil {
		t.Fatalf("GetState attacker failed: %v", err)
	}
	storedDefender, err := repo.GetState(defender.Player.ID)
	if err != nil {
		t.Fatalf("GetState defender failed: %v", err)
	}
	if storedAttacker.Resources.Items["wood"] != battle.Plunder["wood"] || storedDefender.Resources.Items["wood"] != 10000-battle.Plunder["wood"] {
		t.Fatalf("expected final resource transfer to match report, attacker=%d defender=%d plunder=%d", storedAttacker.Resources.Items["wood"], storedDefender.Resources.Items["wood"], battle.Plunder["wood"])
	}

	retriedBattle, err := svc.ResolvePvpMarch(started.March.ID)
	if err != nil || retriedBattle.ID != battle.ID || retriedBattle.Plunder["wood"] != battle.Plunder["wood"] || retriedBattle.AttackerReportID != battle.AttackerReportID || retriedBattle.DefenderReportID != battle.DefenderReportID {
		t.Fatalf("expected idempotent read to return original plunder battle, original=%+v retried=%+v err=%v", battle, retriedBattle, err)
	}
	retriedAttacker, attackerErr := repo.GetState(attacker.Player.ID)
	retriedDefender, defenderErr := repo.GetState(defender.Player.ID)
	if attackerErr != nil || defenderErr != nil || retriedAttacker.Resources.Items["wood"] != battle.Plunder["wood"] || retriedDefender.Resources.Items["wood"] != 10000-battle.Plunder["wood"] {
		t.Fatalf("expected retry not to transfer resources twice, attacker=%+v defender=%+v errors=%v/%v", retriedAttacker.Resources.Items, retriedDefender.Resources.Items, attackerErr, defenderErr)
	}
	retriedAttackerReports, attackerTotal, attackerReportErr := repo.ListReports(attacker.Player.ID, 10, 0)
	retriedDefenderReports, defenderTotal, defenderReportErr := repo.ListReports(defender.Player.ID, 10, 0)
	if attackerReportErr != nil || defenderReportErr != nil || attackerTotal != 1 || defenderTotal != 1 || len(retriedAttackerReports) != 1 || len(retriedDefenderReports) != 1 || retriedAttackerReports[0].ID != battle.AttackerReportID || retriedDefenderReports[0].ID != battle.DefenderReportID {
		t.Fatalf("expected retry not to duplicate plunder reports, totals=%d/%d reports=%+v/%+v errors=%v/%v", attackerTotal, defenderTotal, retriedAttackerReports, retriedDefenderReports, attackerReportErr, defenderReportErr)
	}
	for _, report := range []BattleReport{retriedAttackerReports[0], retriedDefenderReports[0]} {
		if len(report.TraitTriggered) != 2 || len(report.TraitOutcomes) != 2 {
			t.Fatalf("expected retry to preserve exactly two plunder outcomes, report=%s triggered=%+v outcomes=%+v", report.ID, report.TraitTriggered, report.TraitOutcomes)
		}
		for traitID, wantDelta := range map[string]int{"jinfan_jielue": ganningDelta, "jiangdong_haoling": sunquanDelta} {
			values, ok := report.TraitOutcomes[traitID].Detail["plunderDelta"].(map[string]int)
			if !ok || values["wood"] != wantDelta {
				t.Fatalf("expected retry to preserve %s delta %d, report=%s outcome=%+v", traitID, wantDelta, report.ID, report.TraitOutcomes[traitID])
			}
		}
	}
}

// TestPvpPlunderTraitsDoNotTriggerOutsideSuccessfulPlunder 验证普通进攻和掠夺战败都不会修改资源或生成掠夺特性战报。
func TestPvpPlunderTraitsDoNotTriggerOutsideSuccessfulPlunder(t *testing.T) {
	tests := []struct {
		name          string
		marchMode     string
		attackerCount int
		defenderCount int
		wantWinner    string
	}{
		{name: "普通进攻获胜", marchMode: PvpMarchTypeAttack, attackerCount: 1000, defenderCount: 10, wantWinner: "attacker"},
		{name: "掠夺战败", marchMode: PvpMarchTypePlunder, attackerCount: 10, defenderCount: 1000, wantWinner: "defender"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			setTestFactionsAndGenerals(t, FactionsConfig{
				"wu":  {Name: "吴国", Generals: []GeneralInfo{{ID: "ganning", Name: "甘宁"}, {ID: "sunquan", Name: "孙权"}}},
				"wei": {Name: "魏国", Generals: []GeneralInfo{{ID: "caocao", Name: "曹操"}}},
			}, GeneralsConfig{Enabled: true, Heroes: map[string]GeneralHeroConfig{
				"ganning": {
					ID: "ganning", Name: "甘宁", Faction: "wu", Enabled: true,
					SpecialTrait: GeneralTraitConfig{TraitID: "jinfan_jielue", TraitType: general.TraitTypeSpecial, Enabled: true, Scope: "self_plunder", AllowedSides: []string{"attacker"}, AllowedScenes: []string{"plunder"}, Params: map[string]float64{"plunderBonusRate": 0.2, "triggerChance": 1}},
				},
				"sunquan": {
					ID: "sunquan", Name: "孙权", Faction: "wu", Enabled: true,
					SpecialTrait: GeneralTraitConfig{TraitID: "jiangdong_haoling", TraitType: general.TraitTypeSpecial, Enabled: true, Scope: "enemy_plunder", AllowedSides: []string{"defender"}, AllowedScenes: []string{"plunder"}, Params: map[string]float64{"plunderBonusRate": -0.2, "triggerChance": 1}},
				},
				"caocao": {ID: "caocao", Name: "曹操", Faction: "wei", Enabled: true},
			}})
			svc, repo, attacker, defender := newPvpTestServiceForGenerals(t, "wu", "ganning", "wu", "sunquan")
			attacker.Army = []ArmyUnit{{UnitType: "wuInfantry", Amount: tc.attackerCount}}
			attacker.Resources.Items = map[string]int{"wood": 0, "stone": 0, "iron": 0, "food": 0}
			attacker.Resources.Capacity = map[string]int{"wood": 100000, "stone": 100000, "iron": 100000, "food": 100000}
			defender.Army = []ArmyUnit{{UnitType: "wuInfantry", Amount: tc.defenderCount}}
			defender.Buildings = nil
			defender.Resources.Items = map[string]int{"wood": 10000, "stone": 0, "iron": 0, "food": 0}
			nowText := time.Now().UTC().Format(resourceDateLayout)
			attacker.ResourceSettledAt = nowText
			defender.ResourceSettledAt = nowText
			repo.players[attacker.Player.ID] = attacker
			repo.players[defender.Player.ID] = defender

			started, err := svc.StartPvpAttack(PvpAttackRequest{
				PlayerID: attacker.Player.ID, TargetPlayerID: defender.Player.ID, MarchMode: tc.marchMode,
				Troops: map[string]int{"wuInfantry": tc.attackerCount}, GeneralIDs: []string{"ganning"},
			})
			if err != nil {
				t.Fatalf("StartPvpAttack failed: %v", err)
			}
			forcePvpMarchDue(t, repo, started.March.ID)
			battle, err := svc.ResolvePvpMarch(started.March.ID)
			if err != nil || battle.Result["winner"] != tc.wantWinner {
				t.Fatalf("expected winner %s, battle=%+v err=%v", tc.wantWinner, battle, err)
			}
			if totalTroops(battle.Plunder) != 0 {
				t.Fatalf("expected no resource transfer, got %+v", battle.Plunder)
			}
			for _, playerID := range []string{attacker.Player.ID, defender.Player.ID} {
				reports, _, listErr := repo.ListReports(playerID, 10, 0)
				if listErr != nil || len(reports) == 0 {
					t.Fatalf("expected report for %s, reports=%+v err=%v", playerID, reports, listErr)
				}
				for _, traitID := range []string{"jinfan_jielue", "jiangdong_haoling"} {
					if _, triggered := reports[0].TraitOutcomes[traitID]; triggered {
						t.Fatalf("expected %s not to trigger in %s, outcomes=%+v", traitID, tc.name, reports[0].TraitOutcomes)
					}
				}
			}
			storedAttacker, getAttackerErr := repo.GetState(attacker.Player.ID)
			storedDefender, getDefenderErr := repo.GetState(defender.Player.ID)
			if getAttackerErr != nil || getDefenderErr != nil || storedAttacker.Resources.Items["wood"] != 0 || storedDefender.Resources.Items["wood"] != 10000 {
				t.Fatalf("expected resources unchanged, attacker=%+v defender=%+v errors=%v/%v", storedAttacker.Resources.Items, storedDefender.Resources.Items, getAttackerErr, getDefenderErr)
			}
		})
	}
}

// TestPvpJinfanQixiOnlyBoostsRealPlunderBattle 验证甘宁两项特性只在掠夺战分别修正攻击和最终资源。
func TestPvpJinfanQixiOnlyBoostsRealPlunderBattle(t *testing.T) {
	tests := []struct {
		name          string
		marchMode     string
		wantPower     float64
		wantWinner    string
		wantTriggered bool
	}{
		{name: "普通进攻不生效", marchMode: PvpMarchTypeAttack, wantPower: 1000, wantWinner: "defender", wantTriggered: false},
		{name: "掠夺战提升攻击并反转胜负", marchMode: PvpMarchTypePlunder, wantPower: 1100, wantWinner: "attacker", wantTriggered: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			setTestFactionsAndGenerals(t, FactionsConfig{
				"wu":  {Name: "吴国", Generals: []GeneralInfo{{ID: "ganning", Name: "甘宁"}}},
				"wei": {Name: "魏国", Generals: []GeneralInfo{{ID: "caocao", Name: "曹操"}}},
			}, GeneralsConfig{Enabled: true, Heroes: map[string]GeneralHeroConfig{
				"ganning": {
					ID: "ganning", Name: "甘宁", Faction: "wu", Enabled: true,
					SpecialTrait: GeneralTraitConfig{
						TraitID: "jinfan_jielue", TraitType: general.TraitTypeSpecial, Enabled: true,
						Scope: "self_plunder", AllowedSides: []string{"attacker"}, AllowedScenes: []string{"plunder"},
						Params: map[string]float64{"plunderBonusRate": 0.2, "triggerChance": 1},
					},
					BonusTrait: GeneralTraitConfig{
						TraitID: "jinfan_qixi", TraitType: general.TraitTypeBonus, Enabled: true,
						Scope: "self_army", AllowedSides: []string{"attacker"}, AllowedScenes: []string{"plunder"},
						Params: map[string]float64{"attackBonusRate": 0.1, "triggerChance": 1},
					},
				},
				"caocao": {ID: "caocao", Name: "曹操", Faction: "wei", Enabled: true},
			}})
			svc, repo, attacker, defender := newPvpTestServiceForGenerals(t, "wu", "ganning", "wei", "caocao")
			attacker.Army = []ArmyUnit{{UnitType: "wuInfantry", Amount: 100}}
			attacker.Resources.Items = map[string]int{"wood": 0, "stone": 0, "iron": 0, "food": 0}
			attacker.Resources.Capacity = map[string]int{"wood": 100000, "stone": 100000, "iron": 100000, "food": 100000}
			defender.Army = []ArmyUnit{{UnitType: "weiInfantry", Amount: 105}}
			defender.Buildings = nil
			defender.Resources.Items = map[string]int{"wood": 10000, "stone": 0, "iron": 0, "food": 0}
			defender.Resources.Capacity = map[string]int{"wood": 0, "stone": 0, "iron": 0, "food": 0}
			nowText := time.Now().UTC().Format(resourceDateLayout)
			attacker.ResourceSettledAt = nowText
			defender.ResourceSettledAt = nowText
			repo.players[attacker.Player.ID] = attacker
			repo.players[defender.Player.ID] = defender

			started, err := svc.StartPvpAttack(PvpAttackRequest{
				PlayerID: attacker.Player.ID, TargetPlayerID: defender.Player.ID, MarchMode: tc.marchMode,
				Troops: map[string]int{"wuInfantry": 100}, GeneralIDs: []string{"ganning"},
			})
			if err != nil {
				t.Fatalf("StartPvpAttack failed: %v", err)
			}
			forcePvpMarchDue(t, repo, started.March.ID)
			battle, err := svc.ResolvePvpMarch(started.March.ID)
			if err != nil {
				t.Fatalf("ResolvePvpMarch failed: %v", err)
			}
			attackPower, ok := battle.Result["attackerPower"].(float64)
			if !ok || attackPower != tc.wantPower || battle.Result["winner"] != tc.wantWinner {
				t.Fatalf("expected power %.0f winner %s, got %+v", tc.wantPower, tc.wantWinner, battle.Result)
			}
			attackerReports, _, err := repo.ListReports(attacker.Player.ID, 10, 0)
			if err != nil || len(attackerReports) == 0 || attackerReports[0].ID != battle.AttackerReportID {
				t.Fatalf("expected attacker report, reports=%+v err=%v", attackerReports, err)
			}
			defenderReports, _, err := repo.ListReports(defender.Player.ID, 10, 0)
			if err != nil || len(defenderReports) == 0 || defenderReports[0].ID != battle.DefenderReportID {
				t.Fatalf("expected defender report, reports=%+v err=%v", defenderReports, err)
			}
			for _, report := range []BattleReport{attackerReports[0], defenderReports[0]} {
				outcome, triggered := report.TraitOutcomes["jinfan_qixi"]
				if triggered != tc.wantTriggered {
					t.Fatalf("expected triggered=%t, outcomes=%+v", tc.wantTriggered, report.TraitOutcomes)
				}
				_, plunderTriggered := report.TraitOutcomes["jinfan_jielue"]
				if plunderTriggered != tc.wantTriggered {
					t.Fatalf("expected plunder trait triggered=%t, outcomes=%+v", tc.wantTriggered, report.TraitOutcomes)
				}
				if tc.wantTriggered {
					modified, detailOK := outcome.Detail["attackModifiedUnits"].(map[string]int)
					designRate, rateOK := outcome.Detail["attackBonusRate"].(float64)
					if !detailOK || !rateOK || designRate != 0.1 || modified["wuInfantry"] != 1 || outcome.OwnerSide != "attacker" {
						t.Fatalf("expected real attack delta +1 owned by attacker, outcome=%+v", outcome)
					}
					plunderOutcome := report.TraitOutcomes["jinfan_jielue"]
					plunderDelta, deltaOK := plunderOutcome.Detail["plunderDelta"].(map[string]int)
					plunderRate, plunderRateOK := plunderOutcome.Detail["plunderBonusRate"].(float64)
					if !deltaOK || !plunderRateOK || plunderRate != 0.2 || plunderDelta["wood"] != 52 || plunderOutcome.OwnerSide != "attacker" {
						t.Fatalf("expected real wood plunder delta +52 owned by attacker, outcome=%+v", plunderOutcome)
					}
					wantTimeline := []string{"jinfan_qixi", "jinfan_jielue"}
					if !reflect.DeepEqual(report.TraitTriggered, wantTimeline) || report.Detail == nil || len(report.Detail.Traits) != 2 || report.Detail.Traits[0].TraitID != wantTimeline[0] || report.Detail.Traits[1].TraitID != wantTimeline[1] {
						t.Fatalf("expected attack then plunder timeline %v, report=%s legacy=%v detail=%+v", wantTimeline, report.ID, report.TraitTriggered, report.Detail)
					}
				} else if len(report.TraitTriggered) != 0 || standardReportHasTrait(report.Detail, "jinfan_qixi") || standardReportHasTrait(report.Detail, "jinfan_jielue") {
					t.Fatalf("expected normal attack to omit both Gan Ning traits, report=%s legacy=%v detail=%+v", report.ID, report.TraitTriggered, report.Detail)
				}
			}
			storedAttacker, attackerErr := repo.GetState(attacker.Player.ID)
			storedDefender, err := repo.GetState(defender.Player.ID)
			if attackerErr != nil || err != nil {
				t.Fatalf("GetState failed: attacker=%v defender=%v", attackerErr, err)
			}
			defenderLosses := pvpTestLossesFromBattle(t, battle, "defender")
			if got, want := armySliceToMap(storedDefender.Army)["weiInfantry"], 105-defenderLosses["weiInfantry"]; got != want {
				t.Fatalf("expected defender state %d to match battle losses, got %d losses=%+v", want, got, defenderLosses)
			}
			if !tc.wantTriggered {
				if totalTroops(battle.Plunder) != 0 || storedAttacker.Resources.Items["wood"] != 0 || storedDefender.Resources.Items["wood"] != 10000 {
					t.Fatalf("expected normal attack resources unchanged, battle=%+v attacker=%+v defender=%+v", battle.Plunder, storedAttacker.Resources.Items, storedDefender.Resources.Items)
				}
				return
			}

			attackerLosses := pvpTestLossesFromBattle(t, battle, "attacker")
			attackerReport := attackerReports[0]
			if attackerLosses["wuInfantry"] != 48 || defenderLosses["weiInfantry"] != 54 || attackerReport.LostUnits["wuInfantry"] != 48 || attackerReport.DefenderLostUnits["weiInfantry"] != 54 {
				t.Fatalf("expected exact losses 48/54 in battle and legacy report, battle=%+v report=%+v", battle.Losses, attackerReport)
			}
			if battle.Plunder["wood"] != 312 || attackerReport.Rewards["wood"] != 312 || attackerReport.Detail.Rewards.Resources["wood"] != 312 {
				t.Fatalf("expected final wood plunder 312 in battle and both report formats, battle=%+v legacy=%+v standard=%+v", battle.Plunder, attackerReport.Rewards, attackerReport.Detail.Rewards.Resources)
			}
			if storedAttacker.Resources.Items["wood"] != 312 || storedDefender.Resources.Items["wood"] != 9688 {
				t.Fatalf("expected authoritative resources 312/9688, attacker=%+v defender=%+v", storedAttacker.Resources.Items, storedDefender.Resources.Items)
			}
			storedMarch, marchErr := repo.GetPvpMarch(started.March.ID)
			if marchErr != nil || storedMarch.Status != PvpMarchStatusReturning || storedMarch.AttackTroops["wuInfantry"] != 52 {
				t.Fatalf("expected 52 attackers to return, march=%+v err=%v", storedMarch, marchErr)
			}
			if pvpTestGeneralExp(storedAttacker, "ganning") != 54 || attackerReport.GeneralExpGained != 54 {
				t.Fatalf("expected Gan Ning exp to equal 54 real defender deaths, stored=%d report=%d", pvpTestGeneralExp(storedAttacker, "ganning"), attackerReport.GeneralExpGained)
			}
			for _, report := range []BattleReport{attackerReports[0], defenderReports[0]} {
				if report.Detail == nil || report.Detail.SecondarySide == nil {
					t.Fatalf("expected standard two-sided report, report=%s detail=%+v", report.ID, report.Detail)
				}
				assertStandardUnitRow(t, report.ID, report.Detail.PrimarySide, "wuInfantry", 100, 48, 52)
				assertStandardUnitRow(t, report.ID, *report.Detail.SecondarySide, "weiInfantry", 105, 54, 51)
			}
		})
	}
}

// TestPvpSunQuanDefenseAndPlunderTraitsReconcile 验证孙权战前加防和战败后减掠夺在同一场真实 PVP 中完整对账。
func TestPvpSunQuanDefenseAndPlunderTraitsReconcile(t *testing.T) {
	setTestFactionsAndGenerals(t, FactionsConfig{
		"wei": {Name: "魏国", Generals: []GeneralInfo{{ID: "caocao", Name: "曹操"}}},
		"wu":  {Name: "吴国", Generals: []GeneralInfo{{ID: "sunquan", Name: "孙权"}}},
	}, GeneralsConfig{Enabled: true, Heroes: map[string]GeneralHeroConfig{
		"caocao": {ID: "caocao", Name: "曹操", Faction: "wei", Enabled: true},
		"sunquan": {
			ID: "sunquan", Name: "孙权", Faction: "wu", Enabled: true,
			SpecialTrait: GeneralTraitConfig{
				TraitID: "jiangdong_haoling", TraitType: general.TraitTypeSpecial, Enabled: true,
				Scope: "enemy_plunder", AllowedSides: []string{"defender"}, AllowedScenes: []string{"plunder"},
				Params: map[string]float64{"plunderBonusRate": -0.2},
			},
			BonusTrait: GeneralTraitConfig{
				TraitID: "jiangdong_gushou", TraitType: general.TraitTypeBonus, Enabled: true,
				Scope: "self_army", AllowedSides: []string{"defender", "reinforcement"},
				Params: map[string]float64{"triggerChance": 1, "defenseBonusRate": 0.5},
			},
		},
	}})
	svc, repo, attacker, defender := newPvpTestServiceForGenerals(t, "wei", "caocao", "wu", "sunquan")
	attacker.Army = []ArmyUnit{{UnitType: "weiInfantry", Amount: 200}}
	attacker.Resources.Items = map[string]int{"wood": 0, "stone": 0, "iron": 0, "food": 0}
	attacker.Resources.Capacity = map[string]int{"wood": 100000, "stone": 100000, "iron": 100000, "food": 100000}
	defender.Army = []ArmyUnit{{UnitType: "wuInfantry", Amount: 100}}
	defender.Buildings = nil
	defender.Resources.Items = map[string]int{"wood": 10000, "stone": 0, "iron": 0, "food": 0}
	defender.Resources.Capacity = map[string]int{"wood": 0, "stone": 0, "iron": 0, "food": 0}
	nowText := time.Now().UTC().Format(resourceDateLayout)
	attacker.ResourceSettledAt = nowText
	defender.ResourceSettledAt = nowText
	repo.players[attacker.Player.ID] = attacker
	repo.players[defender.Player.ID] = defender

	started, err := svc.StartPvpAttack(PvpAttackRequest{
		PlayerID: attacker.Player.ID, TargetPlayerID: defender.Player.ID, MarchMode: PvpMarchTypePlunder,
		Troops: map[string]int{"weiInfantry": 200}, GeneralIDs: []string{"caocao"},
	})
	if err != nil {
		t.Fatalf("StartPvpAttack failed: %v", err)
	}
	forcePvpMarchDue(t, repo, started.March.ID)
	battle, err := svc.ResolvePvpMarch(started.March.ID)
	if err != nil {
		t.Fatalf("ResolvePvpMarch failed: %v", err)
	}
	attackPower, attackOK := battle.Result["attackerPower"].(float64)
	defensePower, defenseOK := battle.Result["defensePower"].(float64)
	if !attackOK || !defenseOK || attackPower != 2000 || defensePower != 1500 || battle.Result["winner"] != "attacker" {
		t.Fatalf("expected 2000/1500 attacker victory after Sun Quan defense bonus, result=%+v", battle.Result)
	}

	attackerReports, _, err := repo.ListReports(attacker.Player.ID, 10, 0)
	if err != nil || len(attackerReports) == 0 || attackerReports[0].ID != battle.AttackerReportID {
		t.Fatalf("expected attacker report, reports=%+v err=%v", attackerReports, err)
	}
	defenderReports, _, err := repo.ListReports(defender.Player.ID, 10, 0)
	if err != nil || len(defenderReports) == 0 || defenderReports[0].ID != battle.DefenderReportID {
		t.Fatalf("expected defender report, reports=%+v err=%v", defenderReports, err)
	}
	wantTimeline := []string{"jiangdong_gushou", "jiangdong_haoling"}
	for _, report := range []BattleReport{attackerReports[0], defenderReports[0]} {
		if !reflect.DeepEqual(report.TraitTriggered, wantTimeline) {
			t.Fatalf("expected defense then plunder timeline %v, report=%s timeline=%v", wantTimeline, report.ID, report.TraitTriggered)
		}
		defenseOutcome := report.TraitOutcomes["jiangdong_gushou"]
		infantry, infantryOK := defenseOutcome.Detail["infantryDefenseModifiedUnits"].(map[string]int)
		cavalry, cavalryOK := defenseOutcome.Detail["cavalryDefenseModifiedUnits"].(map[string]int)
		defenseRate, defenseRateOK := defenseOutcome.Detail["defenseBonusRate"].(float64)
		if !infantryOK || !cavalryOK || !defenseRateOK || defenseRate != 0.5 || infantry["wuInfantry"] != 5 || cavalry["wuInfantry"] != 4 || defenseOutcome.OwnerSide != "defender" || defenseOutcome.OwnerGeneralID != "sunquan" {
			t.Fatalf("expected Sun Quan defense +5/+4 owned by defender, report=%s outcome=%+v", report.ID, defenseOutcome)
		}
		plunderOutcome := report.TraitOutcomes["jiangdong_haoling"]
		plunderDelta, deltaOK := plunderOutcome.Detail["plunderDelta"].(map[string]int)
		plunderRate, plunderRateOK := plunderOutcome.Detail["plunderBonusRate"].(float64)
		if !deltaOK || !plunderRateOK || plunderRate != -0.2 || plunderDelta["wood"] != -121 || plunderOutcome.OwnerSide != "defender" || plunderOutcome.OwnerGeneralID != "sunquan" {
			t.Fatalf("expected Sun Quan wood plunder delta -121 owned by defender, report=%s outcome=%+v", report.ID, plunderOutcome)
		}
		if report.Detail == nil || len(report.Detail.Traits) != 2 || report.Detail.Traits[0].TraitID != wantTimeline[0] || report.Detail.Traits[1].TraitID != wantTimeline[1] || report.Detail.Traits[0].OwnerRole != "defender" || report.Detail.Traits[1].OwnerRole != "defender" {
			t.Fatalf("expected standard defender-owned cross-phase timeline, report=%s detail=%+v", report.ID, report.Detail)
		}
	}

	attackerLosses := pvpTestLossesFromBattle(t, battle, "attacker")
	defenderLosses := pvpTestLossesFromBattle(t, battle, "defender")
	attackerReport := attackerReports[0]
	defenderReport := defenderReports[0]
	if attackerLosses["weiInfantry"] != 79 || defenderLosses["wuInfantry"] != 60 || attackerReport.LostUnits["weiInfantry"] != 79 || attackerReport.DefenderLostUnits["wuInfantry"] != 60 {
		t.Fatalf("expected exact losses 79/60 in battle and reports, battle=%+v attackerReport=%+v", battle.Losses, attackerReport)
	}
	if battle.Plunder["wood"] != 484 || attackerReport.Rewards["wood"] != 484 || attackerReport.Detail.Rewards.Resources["wood"] != 484 {
		t.Fatalf("expected final wood plunder 484 in battle and reports, battle=%+v legacy=%+v standard=%+v", battle.Plunder, attackerReport.Rewards, attackerReport.Detail.Rewards.Resources)
	}
	storedAttacker, attackerErr := repo.GetState(attacker.Player.ID)
	storedDefender, defenderErr := repo.GetState(defender.Player.ID)
	if attackerErr != nil || defenderErr != nil {
		t.Fatalf("GetState failed: attacker=%v defender=%v", attackerErr, defenderErr)
	}
	if storedAttacker.Resources.Items["wood"] != 484 || storedDefender.Resources.Items["wood"] != 9516 || armySliceToMap(storedDefender.Army)["wuInfantry"] != 40 {
		t.Fatalf("expected resources 484/9516 and defender army 40, attacker=%+v defender=%+v army=%+v", storedAttacker.Resources.Items, storedDefender.Resources.Items, storedDefender.Army)
	}
	storedMarch, marchErr := repo.GetPvpMarch(started.March.ID)
	if marchErr != nil || storedMarch.Status != PvpMarchStatusReturning || storedMarch.AttackTroops["weiInfantry"] != 121 {
		t.Fatalf("expected 121 attackers to return, march=%+v err=%v", storedMarch, marchErr)
	}
	if pvpTestGeneralExp(storedAttacker, "caocao") != 60 || attackerReport.GeneralExpGained != 60 || pvpTestGeneralExp(storedDefender, "sunquan") != 79 || defenderReport.GeneralExpGained != 79 {
		t.Fatalf("expected attacker/defender exp 60/79, stored=%d/%d reports=%d/%d", pvpTestGeneralExp(storedAttacker, "caocao"), pvpTestGeneralExp(storedDefender, "sunquan"), attackerReport.GeneralExpGained, defenderReport.GeneralExpGained)
	}
	for _, report := range []BattleReport{attackerReport, defenderReport} {
		if report.Detail == nil || report.Detail.SecondarySide == nil {
			t.Fatalf("expected standard two-sided report, report=%s detail=%+v", report.ID, report.Detail)
		}
		attackerUnit := BattleReportUnit{}
		for _, unit := range report.Detail.PrimarySide.Units {
			if unit.UnitType == "weiInfantry" {
				attackerUnit = unit
				break
			}
		}
		defenderUnit := BattleReportUnit{}
		for _, unit := range report.Detail.SecondarySide.Units {
			if unit.UnitType == "wuInfantry" {
				defenderUnit = unit
				break
			}
		}
		if attackerUnit.UnitType != "weiInfantry" || attackerUnit.AmountBefore != 200 || attackerUnit.Lost != 79 || attackerUnit.Survived != 121 {
			t.Fatalf("expected standard attacker row 200/79/121, report=%s unit=%+v", report.ID, attackerUnit)
		}
		if defenderUnit.UnitType != "wuInfantry" || defenderUnit.AmountBefore != 100 || defenderUnit.Lost != 60 || defenderUnit.Survived != 40 {
			t.Fatalf("expected standard defender row 100/60/40, report=%s unit=%+v", report.ID, defenderUnit)
		}
	}
}

// TestPvpHuangGaiSuppressionAndCounterTraitsReconcile 验证黄盖压制敌方后续特性时仍保留自身反击并完整结算双方状态。
func TestPvpHuangGaiSuppressionAndCounterTraitsReconcile(t *testing.T) {
	setTestFactionsAndGenerals(t, FactionsConfig{
		"wu":  {Name: "吴国", Generals: []GeneralInfo{{ID: "huanggai", Name: "黄盖"}}},
		"shu": {Name: "蜀国", Generals: []GeneralInfo{{ID: "huangzhong", Name: "黄忠"}}},
	}, GeneralsConfig{Enabled: true, Heroes: map[string]GeneralHeroConfig{
		"huanggai": {
			ID: "huanggai", Name: "黄盖", Faction: "wu", Enabled: true,
			SpecialTrait: GeneralTraitConfig{
				TraitID: "kurouji", TraitType: general.TraitTypeSpecial, Enabled: true, Scope: "enemy_traits",
				Params: map[string]float64{"triggerChance": 1, "disableTraitCount": 1},
			},
			BonusTrait: GeneralTraitConfig{
				TraitID: "kurou_fanji", TraitType: general.TraitTypeBonus, Enabled: true, Scope: "enemy_army",
				Params: map[string]float64{"effectRate": 0.1},
			},
		},
		"huangzhong": {
			ID: "huangzhong", Name: "黄忠", Faction: "shu", Enabled: true,
			BonusTrait: GeneralTraitConfig{
				TraitID: "laodang_yizhuang", TraitType: general.TraitTypeBonus, Enabled: true, Scope: "enemy_army",
				Params: map[string]float64{"effectRate": 0.5},
			},
		},
	}})
	svc, repo, attacker, defender := newPvpTestServiceForGenerals(t, "wu", "huanggai", "shu", "huangzhong")
	attacker.Army = []ArmyUnit{{UnitType: "wuInfantry", Amount: 1000}}
	defender.Army = []ArmyUnit{{UnitType: "shuInfantry", Amount: 1000}}
	defender.Buildings = nil
	repo.players[attacker.Player.ID] = attacker
	repo.players[defender.Player.ID] = defender

	started, err := svc.StartPvpAttack(PvpAttackRequest{
		PlayerID: attacker.Player.ID, TargetPlayerID: defender.Player.ID, MarchMode: PvpMarchTypePlunder,
		Troops: map[string]int{"wuInfantry": 1000}, GeneralIDs: []string{"huanggai"},
	})
	if err != nil {
		t.Fatalf("StartPvpAttack failed: %v", err)
	}
	forcePvpMarchDue(t, repo, started.March.ID)
	battle, err := svc.ResolvePvpMarch(started.March.ID)
	if err != nil {
		t.Fatalf("ResolvePvpMarch failed: %v", err)
	}
	attackPower, attackOK := battle.Result["attackerPower"].(float64)
	defensePower, defenseOK := battle.Result["defensePower"].(float64)
	if !attackOK || !defenseOK || attackPower != 10000 || defensePower != 10000 || battle.Result["winner"] != "draw" {
		t.Fatalf("expected equal-power plunder draw, result=%+v", battle.Result)
	}
	if totalTroops(battle.Plunder) != 0 {
		t.Fatalf("expected draw not to transfer resources, plunder=%+v", battle.Plunder)
	}

	attackerReports, _, err := repo.ListReports(attacker.Player.ID, 10, 0)
	if err != nil || len(attackerReports) == 0 || attackerReports[0].ID != battle.AttackerReportID {
		t.Fatalf("expected attacker report, reports=%+v err=%v", attackerReports, err)
	}
	defenderReports, _, err := repo.ListReports(defender.Player.ID, 10, 0)
	if err != nil || len(defenderReports) == 0 || defenderReports[0].ID != battle.DefenderReportID {
		t.Fatalf("expected defender report, reports=%+v err=%v", defenderReports, err)
	}
	wantTimeline := []string{"kurouji", "kurou_fanji"}
	for _, report := range []BattleReport{attackerReports[0], defenderReports[0]} {
		if !reflect.DeepEqual(report.TraitTriggered, wantTimeline) {
			t.Fatalf("expected suppression then own counter timeline %v, report=%s timeline=%v", wantTimeline, report.ID, report.TraitTriggered)
		}
		if _, exists := report.TraitOutcomes["laodang_yizhuang"]; exists || standardReportHasTrait(report.Detail, "laodang_yizhuang") {
			t.Fatalf("expected enemy follow-up damage suppressed in both report formats, report=%s outcomes=%+v detail=%+v", report.ID, report.TraitOutcomes, report.Detail)
		}
		suppression := report.TraitOutcomes["kurouji"]
		if suppression.OwnerSide != "attacker" || suppression.OwnerGeneralID != "huanggai" || suppression.Detail["disableTraitCount"] != 1 || suppression.Detail["disabledTraitCount"] != 1 || suppression.Detail["triggerChance"] != float64(1) {
			t.Fatalf("expected Huang Gai to suppress exactly one enemy trait, report=%s outcome=%+v", report.ID, suppression)
		}
		counter := report.TraitOutcomes["kurou_fanji"]
		extra, extraOK := counter.Detail["extraLosses"].(map[string]int)
		effectRate, rateOK := counter.Detail["effectRate"].(float64)
		if !extraOK || !rateOK || effectRate != 0.1 || extra["shuInfantry"] != 100 || counter.OwnerSide != "attacker" || counter.OwnerGeneralID != "huanggai" {
			t.Fatalf("expected Huang Gai own counter to remain and add 100 losses, report=%s outcome=%+v", report.ID, counter)
		}
		if report.Detail == nil || len(report.Detail.Traits) != 2 || report.Detail.Traits[0].TraitID != wantTimeline[0] || report.Detail.Traits[1].TraitID != wantTimeline[1] || report.Detail.Traits[0].OwnerRole != "attacker" || report.Detail.Traits[1].OwnerRole != "attacker" {
			t.Fatalf("expected standard attacker-owned suppression then counter timeline, report=%s detail=%+v", report.ID, report.Detail)
		}
	}

	attackerLosses := pvpTestLossesFromBattle(t, battle, "attacker")
	defenderLosses := pvpTestLossesFromBattle(t, battle, "defender")
	attackerReport := attackerReports[0]
	defenderReport := defenderReports[0]
	if attackerLosses["wuInfantry"] != 500 || defenderLosses["shuInfantry"] != 600 || attackerReport.LostUnits["wuInfantry"] != 500 || attackerReport.DefenderLostUnits["shuInfantry"] != 600 {
		t.Fatalf("expected core draw 500/500 plus Huang Gai counter 100 only on defender, battle=%+v report=%+v", battle.Losses, attackerReport)
	}
	storedMarch, marchErr := repo.GetPvpMarch(started.March.ID)
	storedAttacker, attackerErr := repo.GetState(attacker.Player.ID)
	storedDefender, defenderErr := repo.GetState(defender.Player.ID)
	if marchErr != nil || attackerErr != nil || defenderErr != nil {
		t.Fatalf("expected stored battle state, march=%v attacker=%v defender=%v", marchErr, attackerErr, defenderErr)
	}
	if storedMarch.Status != PvpMarchStatusReturning || storedMarch.AttackTroops["wuInfantry"] != 500 || armySliceToMap(storedDefender.Army)["shuInfantry"] != 400 {
		t.Fatalf("expected attacker return 500 and defender remain 400, march=%+v defender=%+v", storedMarch, storedDefender.Army)
	}
	if pvpTestGeneralExp(storedAttacker, "huanggai") != 600 || attackerReport.GeneralExpGained != 600 || pvpTestGeneralExp(storedDefender, "huangzhong") != 500 || defenderReport.GeneralExpGained != 500 {
		t.Fatalf("expected attacker/defender exp 600/500 from final real deaths, stored=%d/%d reports=%d/%d", pvpTestGeneralExp(storedAttacker, "huanggai"), pvpTestGeneralExp(storedDefender, "huangzhong"), attackerReport.GeneralExpGained, defenderReport.GeneralExpGained)
	}
	for _, report := range []BattleReport{attackerReport, defenderReport} {
		if report.Detail == nil || report.Detail.SecondarySide == nil {
			t.Fatalf("expected standard two-sided report, report=%s detail=%+v", report.ID, report.Detail)
		}
		attackerUnit := report.Detail.PrimarySide.Units[0]
		defenderUnit := report.Detail.SecondarySide.Units[0]
		if attackerUnit.UnitType != "wuInfantry" || attackerUnit.AmountBefore != 1000 || attackerUnit.Lost != 500 || attackerUnit.Survived != 500 {
			t.Fatalf("expected standard attacker row 1000/500/500, report=%s unit=%+v", report.ID, attackerUnit)
		}
		if defenderUnit.UnitType != "shuInfantry" || defenderUnit.AmountBefore != 1000 || defenderUnit.Lost != 600 || defenderUnit.Survived != 400 {
			t.Fatalf("expected standard defender row 1000/600/400, report=%s unit=%+v", report.ID, defenderUnit)
		}
	}
}

// TestPvpMaChaoPassiveAndCavalryDamageTraitsReconcile 验证马超被动武力改变核心战力但不进入触发时间线，西凉突击只记录实际骑兵损失。
func TestPvpMaChaoPassiveAndCavalryDamageTraitsReconcile(t *testing.T) {
	setTestFactionsAndGenerals(t, FactionsConfig{
		"shu": {Name: "蜀国", Generals: []GeneralInfo{{ID: "machao", Name: "马超"}}},
		"wei": {Name: "魏国", Generals: []GeneralInfo{{ID: "caocao", Name: "曹操"}}},
	}, GeneralsConfig{Enabled: true, Heroes: map[string]GeneralHeroConfig{
		"machao": {
			ID: "machao", Name: "马超", Faction: "shu", Enabled: true,
			SpecialTrait: GeneralTraitConfig{
				TraitID: "xiliang_tuji", TraitType: general.TraitTypeSpecial, Enabled: true,
				Scope: "enemy_army", TargetUnitType: "cavalry",
				Params: map[string]float64{"triggerChance": 1, "effectRate": 0.12},
			},
			BonusTrait: GeneralTraitConfig{
				TraitID: "tianshen_xiafan", TraitType: general.TraitTypeBonus, Enabled: true,
				Scope: "self_army", Params: map[string]float64{"forceBonus": 20},
			},
		},
		"caocao": {ID: "caocao", Name: "曹操", Faction: "wei", Enabled: true},
	}})
	svc, repo, attacker, defender := newPvpTestServiceForGenerals(t, "shu", "machao", "wei", "caocao")
	unitsMu.Lock()
	activeUnits["wei"]["weiCavalry"] = UnitConfig{
		Name: "魏骑兵", Category: "cavalry",
		Stats: map[string]int{"attack": 10, "infantryDefense": 10, "cavalryDefense": 8, "carryCapacity": 5, "upkeep": 1},
	}
	unitsMu.Unlock()
	attacker.Army = []ArmyUnit{{UnitType: "shuInfantry", Amount: 1000}}
	attacker.Resources.Items = map[string]int{"wood": 0, "stone": 0, "iron": 0, "food": 0}
	defender.Army = []ArmyUnit{{UnitType: "weiCavalry", Amount: 1000}}
	defender.Buildings = nil
	defender.Resources.Items = map[string]int{"wood": 0, "stone": 0, "iron": 0, "food": 0}
	repo.players[attacker.Player.ID] = attacker
	repo.players[defender.Player.ID] = defender

	started, err := svc.StartPvpAttack(PvpAttackRequest{
		PlayerID: attacker.Player.ID, TargetPlayerID: defender.Player.ID, MarchMode: PvpMarchTypePlunder,
		Troops: map[string]int{"shuInfantry": 1000}, GeneralIDs: []string{"machao"},
	})
	if err != nil {
		t.Fatalf("StartPvpAttack failed: %v", err)
	}
	forcePvpMarchDue(t, repo, started.March.ID)
	battle, err := svc.ResolvePvpMarch(started.March.ID)
	if err != nil {
		t.Fatalf("ResolvePvpMarch failed: %v", err)
	}
	attackPower, attackOK := battle.Result["attackerPower"].(float64)
	defensePower, defenseOK := battle.Result["defensePower"].(float64)
	if !attackOK || !defenseOK || attackPower != 14000 || defensePower != 10000 || battle.Result["winner"] != "attacker" {
		t.Fatalf("expected passive force to produce 14000/10000 attacker victory, result=%+v", battle.Result)
	}
	if totalTroops(battle.Plunder) != 0 {
		t.Fatalf("expected empty defender resources not to produce plunder, got %+v", battle.Plunder)
	}

	attackerReports, _, err := repo.ListReports(attacker.Player.ID, 10, 0)
	if err != nil || len(attackerReports) == 0 || attackerReports[0].ID != battle.AttackerReportID {
		t.Fatalf("expected attacker report, reports=%+v err=%v", attackerReports, err)
	}
	defenderReports, _, err := repo.ListReports(defender.Player.ID, 10, 0)
	if err != nil || len(defenderReports) == 0 || defenderReports[0].ID != battle.DefenderReportID {
		t.Fatalf("expected defender report, reports=%+v err=%v", defenderReports, err)
	}
	for _, report := range []BattleReport{attackerReports[0], defenderReports[0]} {
		if !reflect.DeepEqual(report.TraitTriggered, []string{"xiliang_tuji"}) || len(report.TraitOutcomes) != 1 {
			t.Fatalf("expected only Xiliang in trigger timeline, report=%s timeline=%v outcomes=%+v", report.ID, report.TraitTriggered, report.TraitOutcomes)
		}
		if _, exists := report.TraitOutcomes["tianshen_xiafan"]; exists || standardReportHasTrait(report.Detail, "tianshen_xiafan") {
			t.Fatalf("expected passive trait absent from both trigger formats, report=%s outcomes=%+v detail=%+v", report.ID, report.TraitOutcomes, report.Detail)
		}
		outcome := report.TraitOutcomes["xiliang_tuji"]
		extra, extraOK := outcome.Detail["targetExtraLosses"].(map[string]int)
		effectRate, rateOK := outcome.Detail["effectRate"].(float64)
		if !extraOK || !rateOK || effectRate != 0.12 || len(extra) != 1 || extra["weiCavalry"] != 120 || outcome.OwnerSide != "attacker" || outcome.OwnerGeneralID != "machao" {
			t.Fatalf("expected Xiliang to add exactly 120 cavalry losses, report=%s outcome=%+v", report.ID, outcome)
		}
		if len(report.PvpAttackerGenerals) != 1 || report.PvpAttackerGenerals[0].Level != 1 || math.Abs(report.PvpAttackerGenerals[0].Buffs[StatAttackBonus]-0.4) > 1e-9 || !pvpSnapshotHasTrait(report.PvpAttackerGenerals[0], "tianshen_xiafan") || !pvpSnapshotHasTrait(report.PvpAttackerGenerals[0], "xiliang_tuji") {
			t.Fatalf("expected Ma Chao snapshot to retain passive 40%% modifier and both owned traits, report=%s snapshot=%+v", report.ID, report.PvpAttackerGenerals)
		}
		if report.Detail == nil || len(report.Detail.Traits) != 1 || report.Detail.Traits[0].TraitID != "xiliang_tuji" || !standardDetailGeneralHasTrait(report.Detail, "tianshen_xiafan") || !standardDetailGeneralHasTrait(report.Detail, "xiliang_tuji") {
			t.Fatalf("expected standard timeline to contain only trigger while general snapshot keeps both traits, report=%s detail=%+v", report.ID, report.Detail)
		}
	}

	attackerLosses := pvpTestLossesFromBattle(t, battle, "attacker")
	defenderLosses := pvpTestLossesFromBattle(t, battle, "defender")
	attackerReport := attackerReports[0]
	defenderReport := defenderReports[0]
	if attackerLosses["shuInfantry"] != 382 || defenderLosses["weiCavalry"] != 737 || attackerReport.LostUnits["shuInfantry"] != 382 || attackerReport.DefenderLostUnits["weiCavalry"] != 737 {
		t.Fatalf("expected passive-adjusted core losses 382/617 plus Xiliang 120, battle=%+v report=%+v", battle.Losses, attackerReport)
	}
	storedMarch, marchErr := repo.GetPvpMarch(started.March.ID)
	storedAttacker, attackerErr := repo.GetState(attacker.Player.ID)
	storedDefender, defenderErr := repo.GetState(defender.Player.ID)
	if marchErr != nil || attackerErr != nil || defenderErr != nil {
		t.Fatalf("expected stored battle state, march=%v attacker=%v defender=%v", marchErr, attackerErr, defenderErr)
	}
	if storedMarch.Status != PvpMarchStatusReturning || storedMarch.AttackTroops["shuInfantry"] != 618 || armySliceToMap(storedDefender.Army)["weiCavalry"] != 263 {
		t.Fatalf("expected attacker return 618 and cavalry remain 263, march=%+v defender=%+v", storedMarch, storedDefender.Army)
	}
	if pvpTestGeneralExp(storedAttacker, "machao") != 737 || attackerReport.GeneralExpGained != 737 || pvpTestGeneralExp(storedDefender, "caocao") != 382 || defenderReport.GeneralExpGained != 382 {
		t.Fatalf("expected attacker/defender exp 737/382 from final real losses, stored=%d/%d reports=%d/%d", pvpTestGeneralExp(storedAttacker, "machao"), pvpTestGeneralExp(storedDefender, "caocao"), attackerReport.GeneralExpGained, defenderReport.GeneralExpGained)
	}
	if pvpTestGeneralLevel(storedAttacker, "machao") != 2 || attackerReport.GeneralLevelBefore != 1 || attackerReport.GeneralLevelAfter != 2 || attackerReport.Detail.Rewards.GeneralLevelBefore != 1 || attackerReport.Detail.Rewards.GeneralLevelAfter != 2 {
		t.Fatalf("expected battle snapshot level 1 and separate post-battle upgrade 1 -> 2, stored=%d legacy=%d/%d standard=%+v", pvpTestGeneralLevel(storedAttacker, "machao"), attackerReport.GeneralLevelBefore, attackerReport.GeneralLevelAfter, attackerReport.Detail.Rewards)
	}
	for _, report := range []BattleReport{attackerReport, defenderReport} {
		if report.Detail == nil || report.Detail.SecondarySide == nil {
			t.Fatalf("expected standard two-sided report, report=%s detail=%+v", report.ID, report.Detail)
		}
		attackerUnit := BattleReportUnit{}
		for _, unit := range report.Detail.PrimarySide.Units {
			if unit.UnitType == "shuInfantry" {
				attackerUnit = unit
				break
			}
		}
		defenderUnit := BattleReportUnit{}
		for _, unit := range report.Detail.SecondarySide.Units {
			if unit.UnitType == "weiCavalry" {
				defenderUnit = unit
				break
			}
		}
		if attackerUnit.UnitType != "shuInfantry" || attackerUnit.AmountBefore != 1000 || attackerUnit.Lost != 382 || attackerUnit.Survived != 618 {
			t.Fatalf("expected standard attacker row 1000/382/618, report=%s unit=%+v", report.ID, attackerUnit)
		}
		if defenderUnit.UnitType != "weiCavalry" || defenderUnit.AmountBefore != 1000 || defenderUnit.Lost != 737 || defenderUnit.Survived != 263 {
			t.Fatalf("expected standard defender row 1000/737/263, report=%s unit=%+v", report.ID, defenderUnit)
		}
	}
}

// TestPvpWeiYanDirectionalTraitsReconcile 验证魏延进攻破防与防守加防互斥生效，并与双方战报和权威兵力对账。
func TestPvpWeiYanDirectionalTraitsReconcile(t *testing.T) {
	tests := []struct {
		name                 string
		attackerFaction      string
		attackerGeneralID    string
		defenderFaction      string
		defenderGeneralID    string
		activeTraitID        string
		inactiveTraitID      string
		ownerSide            string
		targetUnitType       string
		wantAttackPower      float64
		wantDefensePower     float64
		wantAttackerLosses   int
		wantDefenderLosses   int
		wantAttackerSurvived int
		wantDefenderSurvived int
		wantWeiYanExp        int
		wantInfantryDelta    int
		wantCavalryDelta     int
	}{
		{
			name:            "进攻只触发奇兵绕后",
			attackerFaction: "shu", attackerGeneralID: "weiyan",
			defenderFaction: "wei", defenderGeneralID: "caocao",
			activeTraitID: "qibing_raohou", inactiveTraitID: "gushou_hanzhong", ownerSide: "attacker", targetUnitType: "weiInfantry",
			wantAttackPower: 10000, wantDefensePower: 8000,
			wantAttackerLosses: 421, wantDefenderLosses: 578, wantAttackerSurvived: 579, wantDefenderSurvived: 422,
			wantWeiYanExp: 578, wantInfantryDelta: -2, wantCavalryDelta: -2,
		},
		{
			name:            "防守只触发固守汉中",
			attackerFaction: "wei", attackerGeneralID: "caocao",
			defenderFaction: "shu", defenderGeneralID: "weiyan",
			activeTraitID: "gushou_hanzhong", inactiveTraitID: "qibing_raohou", ownerSide: "defender", targetUnitType: "shuInfantry",
			wantAttackPower: 10000, wantDefensePower: 30000,
			wantAttackerLosses: 826, wantDefenderLosses: 173, wantAttackerSurvived: 174, wantDefenderSurvived: 827,
			wantWeiYanExp: 826, wantInfantryDelta: 20, wantCavalryDelta: 20,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			setTestFactionsAndGenerals(t, FactionsConfig{
				"shu": {Name: "蜀国", Generals: []GeneralInfo{{ID: "weiyan", Name: "魏延"}}},
				"wei": {Name: "魏国", Generals: []GeneralInfo{{ID: "caocao", Name: "曹操"}}},
			}, GeneralsConfig{Enabled: true, Heroes: map[string]GeneralHeroConfig{
				"weiyan": {
					ID: "weiyan", Name: "魏延", Faction: "shu", Enabled: true,
					SpecialTrait: GeneralTraitConfig{
						TraitID: "qibing_raohou", TraitType: general.TraitTypeSpecial, Enabled: true,
						Scope: "enemy_army", AllowedSides: []string{"attacker"},
						Params: map[string]float64{"triggerChance": 1, "enemyDefenseReductionRate": 0.2},
					},
					BonusTrait: GeneralTraitConfig{
						TraitID: "gushou_hanzhong", TraitType: general.TraitTypeBonus, Enabled: true,
						Scope: "self_army", AllowedSides: []string{"defender", "reinforcement"},
						Params: map[string]float64{"triggerChance": 1, "generalDefenseFlat": 20},
					},
				},
				"caocao": {ID: "caocao", Name: "曹操", Faction: "wei", Enabled: true},
			}})
			svc, repo, attacker, defender := newPvpTestServiceForGenerals(
				t, tc.attackerFaction, tc.attackerGeneralID, tc.defenderFaction, tc.defenderGeneralID,
			)
			attackerUnitType := tc.attackerFaction + "Infantry"
			defenderUnitType := tc.defenderFaction + "Infantry"
			attacker.Army = []ArmyUnit{{UnitType: attackerUnitType, Amount: 1000}}
			attacker.Resources.Items = map[string]int{"wood": 0, "stone": 0, "iron": 0, "food": 0}
			defender.Army = []ArmyUnit{{UnitType: defenderUnitType, Amount: 1000}}
			defender.Buildings = nil
			defender.Resources.Items = map[string]int{"wood": 0, "stone": 0, "iron": 0, "food": 0}
			repo.players[attacker.Player.ID] = attacker
			repo.players[defender.Player.ID] = defender

			started, err := svc.StartPvpAttack(PvpAttackRequest{
				PlayerID: attacker.Player.ID, TargetPlayerID: defender.Player.ID, MarchMode: PvpMarchTypePlunder,
				Troops: map[string]int{attackerUnitType: 1000}, GeneralIDs: []string{tc.attackerGeneralID},
			})
			if err != nil {
				t.Fatalf("StartPvpAttack failed: %v", err)
			}
			forcePvpMarchDue(t, repo, started.March.ID)
			battle, err := svc.ResolvePvpMarch(started.March.ID)
			if err != nil {
				t.Fatalf("ResolvePvpMarch failed: %v", err)
			}
			attackPower, attackOK := battle.Result["attackerPower"].(float64)
			defensePower, defenseOK := battle.Result["defensePower"].(float64)
			if !attackOK || !defenseOK || attackPower != tc.wantAttackPower || defensePower != tc.wantDefensePower {
				t.Fatalf("expected powers %.0f/%.0f, got %+v", tc.wantAttackPower, tc.wantDefensePower, battle.Result)
			}

			attackerReports, _, err := repo.ListReports(attacker.Player.ID, 10, 0)
			if err != nil || len(attackerReports) == 0 || attackerReports[0].ID != battle.AttackerReportID {
				t.Fatalf("expected attacker report, reports=%+v err=%v", attackerReports, err)
			}
			defenderReports, _, err := repo.ListReports(defender.Player.ID, 10, 0)
			if err != nil || len(defenderReports) == 0 || defenderReports[0].ID != battle.DefenderReportID {
				t.Fatalf("expected defender report, reports=%+v err=%v", defenderReports, err)
			}
			for _, report := range []BattleReport{attackerReports[0], defenderReports[0]} {
				if !reflect.DeepEqual(report.TraitTriggered, []string{tc.activeTraitID}) || len(report.TraitOutcomes) != 1 {
					t.Fatalf("expected only %s in trigger timeline, report=%s timeline=%v outcomes=%+v", tc.activeTraitID, report.ID, report.TraitTriggered, report.TraitOutcomes)
				}
				if _, exists := report.TraitOutcomes[tc.inactiveTraitID]; exists || standardReportHasTrait(report.Detail, tc.inactiveTraitID) {
					t.Fatalf("expected direction-inactive trait %s absent, report=%s outcomes=%+v detail=%+v", tc.inactiveTraitID, report.ID, report.TraitOutcomes, report.Detail)
				}
				outcome := report.TraitOutcomes[tc.activeTraitID]
				infantry, infantryOK := outcome.Detail["infantryDefenseModifiedUnits"].(map[string]int)
				cavalry, cavalryOK := outcome.Detail["cavalryDefenseModifiedUnits"].(map[string]int)
				if !infantryOK || !cavalryOK || infantry[tc.targetUnitType] != tc.wantInfantryDelta || cavalry[tc.targetUnitType] != tc.wantCavalryDelta || outcome.OwnerSide != tc.ownerSide || outcome.OwnerGeneralID != "weiyan" {
					t.Fatalf("expected real defense deltas %d/%d owned by %s Wei Yan, report=%s outcome=%+v", tc.wantInfantryDelta, tc.wantCavalryDelta, tc.ownerSide, report.ID, outcome)
				}
				if tc.activeTraitID == "qibing_raohou" {
					if rate, ok := outcome.Detail["enemyDefenseReductionRate"].(float64); !ok || rate != 0.2 {
						t.Fatalf("expected Qibing design rate 0.2, report=%s outcome=%+v", report.ID, outcome)
					}
				} else if flat, ok := outcome.Detail["generalDefenseFlat"].(float64); !ok || flat != 20 {
					t.Fatalf("expected Gushou design flat 20, report=%s outcome=%+v", report.ID, outcome)
				}
				if report.Detail == nil || len(report.Detail.Traits) != 1 || report.Detail.Traits[0].TraitID != tc.activeTraitID || !standardDetailGeneralHasTrait(report.Detail, "qibing_raohou") || !standardDetailGeneralHasTrait(report.Detail, "gushou_hanzhong") {
					t.Fatalf("expected standard timeline to keep only active trait while Wei Yan snapshot keeps both, report=%s detail=%+v", report.ID, report.Detail)
				}
				standardOutcome := report.Detail.Traits[0]
				standardInfantry, standardInfantryOK := standardOutcome.Detail["infantryDefenseModifiedUnits"].(map[string]int)
				standardCavalry, standardCavalryOK := standardOutcome.Detail["cavalryDefenseModifiedUnits"].(map[string]int)
				if !standardInfantryOK || !standardCavalryOK || standardInfantry[tc.targetUnitType] != tc.wantInfantryDelta || standardCavalry[tc.targetUnitType] != tc.wantCavalryDelta {
					t.Fatalf("expected standard actual defense deltas %d/%d, report=%s trait=%+v", tc.wantInfantryDelta, tc.wantCavalryDelta, report.ID, standardOutcome)
				}
				weiYanSnapshots := report.PvpAttackerGenerals
				if tc.ownerSide == "defender" {
					weiYanSnapshots = report.PvpDefenderGenerals
				}
				if len(weiYanSnapshots) != 1 || weiYanSnapshots[0].ID != "weiyan" || weiYanSnapshots[0].Level != 1 || !pvpSnapshotHasTrait(weiYanSnapshots[0], "qibing_raohou") || !pvpSnapshotHasTrait(weiYanSnapshots[0], "gushou_hanzhong") {
					t.Fatalf("expected level 1 Wei Yan snapshot with both owned traits, report=%s snapshots=%+v", report.ID, weiYanSnapshots)
				}
			}

			attackerLosses := pvpTestLossesFromBattle(t, battle, "attacker")
			defenderLosses := pvpTestLossesFromBattle(t, battle, "defender")
			if attackerLosses[attackerUnitType] != tc.wantAttackerLosses || defenderLosses[defenderUnitType] != tc.wantDefenderLosses {
				t.Fatalf("expected losses %d/%d, got attacker=%+v defender=%+v", tc.wantAttackerLosses, tc.wantDefenderLosses, attackerLosses, defenderLosses)
			}
			storedMarch, marchErr := repo.GetPvpMarch(started.March.ID)
			storedAttacker, attackerErr := repo.GetState(attacker.Player.ID)
			storedDefender, defenderErr := repo.GetState(defender.Player.ID)
			if marchErr != nil || attackerErr != nil || defenderErr != nil {
				t.Fatalf("expected stored battle state, march=%v attacker=%v defender=%v", marchErr, attackerErr, defenderErr)
			}
			if storedMarch.Status != PvpMarchStatusReturning || storedMarch.AttackTroops[attackerUnitType] != tc.wantAttackerSurvived || armySliceToMap(storedDefender.Army)[defenderUnitType] != tc.wantDefenderSurvived {
				t.Fatalf("expected authoritative survivors %d/%d, march=%+v defender=%+v", tc.wantAttackerSurvived, tc.wantDefenderSurvived, storedMarch, storedDefender.Army)
			}
			weiYanState := storedAttacker
			weiYanReport := attackerReports[0]
			if tc.ownerSide == "defender" {
				weiYanState = storedDefender
				weiYanReport = defenderReports[0]
			}
			if pvpTestGeneralExp(weiYanState, "weiyan") != tc.wantWeiYanExp || weiYanReport.GeneralExpGained != tc.wantWeiYanExp || weiYanReport.Detail.Rewards.GeneralExp != tc.wantWeiYanExp {
				t.Fatalf("expected Wei Yan exp %d from real enemy losses, stored=%d legacy=%d standard=%+v", tc.wantWeiYanExp, pvpTestGeneralExp(weiYanState, "weiyan"), weiYanReport.GeneralExpGained, weiYanReport.Detail.Rewards)
			}
			if pvpTestGeneralLevel(weiYanState, "weiyan") != 2 || weiYanReport.GeneralLevelBefore != 1 || weiYanReport.GeneralLevelAfter != 2 || weiYanReport.Detail.Rewards.GeneralLevelBefore != 1 || weiYanReport.Detail.Rewards.GeneralLevelAfter != 2 {
				t.Fatalf("expected Wei Yan snapshot level 1 and separate upgrade 1 -> 2, stored=%d legacy=%d/%d standard=%+v", pvpTestGeneralLevel(weiYanState, "weiyan"), weiYanReport.GeneralLevelBefore, weiYanReport.GeneralLevelAfter, weiYanReport.Detail.Rewards)
			}
			for _, report := range []BattleReport{attackerReports[0], defenderReports[0]} {
				if report.Detail == nil || report.Detail.SecondarySide == nil {
					t.Fatalf("expected standard two-sided report, report=%s detail=%+v", report.ID, report.Detail)
				}
				attackerUnit := BattleReportUnit{}
				for _, unit := range report.Detail.PrimarySide.Units {
					if unit.UnitType == attackerUnitType {
						attackerUnit = unit
						break
					}
				}
				defenderUnit := BattleReportUnit{}
				for _, unit := range report.Detail.SecondarySide.Units {
					if unit.UnitType == defenderUnitType {
						defenderUnit = unit
						break
					}
				}
				if attackerUnit.UnitType != attackerUnitType || attackerUnit.AmountBefore != 1000 || attackerUnit.Lost != tc.wantAttackerLosses || attackerUnit.Survived != tc.wantAttackerSurvived {
					t.Fatalf("expected standard attacker row 1000/%d/%d, report=%s unit=%+v", tc.wantAttackerLosses, tc.wantAttackerSurvived, report.ID, attackerUnit)
				}
				if defenderUnit.UnitType != defenderUnitType || defenderUnit.AmountBefore != 1000 || defenderUnit.Lost != tc.wantDefenderLosses || defenderUnit.Survived != tc.wantDefenderSurvived {
					t.Fatalf("expected standard defender row 1000/%d/%d, report=%s unit=%+v", tc.wantDefenderLosses, tc.wantDefenderSurvived, report.ID, defenderUnit)
				}
			}
		})
	}
}

// TestPvpQibingRaohouOnlyTriggersForAttacker 验证奇兵绕后只由主动进攻的魏延降低敌军真实防御。
func TestPvpQibingRaohouOnlyTriggersForAttacker(t *testing.T) {
	tests := []struct {
		name              string
		attackerFaction   string
		attackerGeneralID string
		defenderFaction   string
		defenderGeneralID string
		wantDefensePower  float64
		wantTriggered     bool
	}{
		{
			name: "进攻方魏延降低敌军防御", attackerFaction: "shu", attackerGeneralID: "weiyan",
			defenderFaction: "wei", defenderGeneralID: "caocao", wantDefensePower: 800, wantTriggered: true,
		},
		{
			name: "防守方魏延不触发", attackerFaction: "wei", attackerGeneralID: "caocao",
			defenderFaction: "shu", defenderGeneralID: "weiyan", wantDefensePower: 1000, wantTriggered: false,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			setTestFactionsAndGenerals(t, FactionsConfig{
				"shu": {Name: "蜀国", Generals: []GeneralInfo{{ID: "weiyan", Name: "魏延"}}},
				"wei": {Name: "魏国", Generals: []GeneralInfo{{ID: "caocao", Name: "曹操"}}},
			}, GeneralsConfig{Enabled: true, Heroes: map[string]GeneralHeroConfig{
				"weiyan": {
					ID: "weiyan", Name: "魏延", Faction: "shu", Enabled: true,
					SpecialTrait: GeneralTraitConfig{
						TraitID: "qibing_raohou", TraitType: general.TraitTypeSpecial, Enabled: true,
						Scope: "enemy_army", AllowedSides: []string{"attacker"},
						Params: map[string]float64{"enemyDefenseReductionRate": 0.2, "triggerChance": 1},
					},
				},
				"caocao": {ID: "caocao", Name: "曹操", Faction: "wei", Enabled: true},
			}})
			svc, repo, attacker, defender := newPvpTestServiceForGenerals(
				t, tc.attackerFaction, tc.attackerGeneralID, tc.defenderFaction, tc.defenderGeneralID,
			)
			attackerUnitType := tc.attackerFaction + "Infantry"
			defenderUnitType := tc.defenderFaction + "Infantry"
			attacker.Army = []ArmyUnit{{UnitType: attackerUnitType, Amount: 100}}
			defender.Army = []ArmyUnit{{UnitType: defenderUnitType, Amount: 100}}
			defender.Buildings = nil
			repo.players[attacker.Player.ID] = attacker
			repo.players[defender.Player.ID] = defender

			started, err := svc.StartPvpAttack(PvpAttackRequest{
				PlayerID: attacker.Player.ID, TargetPlayerID: defender.Player.ID,
				MarchMode: PvpMarchTypeAttack, Troops: map[string]int{attackerUnitType: 100}, GeneralIDs: []string{tc.attackerGeneralID},
			})
			if err != nil {
				t.Fatalf("StartPvpAttack failed: %v", err)
			}
			forcePvpMarchDue(t, repo, started.March.ID)
			battle, err := svc.ResolvePvpMarch(started.March.ID)
			if err != nil {
				t.Fatalf("ResolvePvpMarch failed: %v", err)
			}
			defensePower, ok := battle.Result["defensePower"].(float64)
			if !ok || defensePower != tc.wantDefensePower {
				t.Fatalf("expected defense power %.0f, got %+v", tc.wantDefensePower, battle.Result)
			}

			attackerReports, _, err := repo.ListReports(attacker.Player.ID, 10, 0)
			if err != nil || len(attackerReports) == 0 || attackerReports[0].ID != battle.AttackerReportID {
				t.Fatalf("expected attacker report, reports=%+v err=%v", attackerReports, err)
			}
			defenderReports, _, err := repo.ListReports(defender.Player.ID, 10, 0)
			if err != nil || len(defenderReports) == 0 || defenderReports[0].ID != battle.DefenderReportID {
				t.Fatalf("expected defender report, reports=%+v err=%v", defenderReports, err)
			}
			for _, report := range []BattleReport{attackerReports[0], defenderReports[0]} {
				outcome, triggered := report.TraitOutcomes["qibing_raohou"]
				if triggered != tc.wantTriggered {
					t.Fatalf("expected triggered=%t, outcomes=%+v", tc.wantTriggered, report.TraitOutcomes)
				}
				standardTriggered := false
				if report.Detail != nil {
					for _, trait := range report.Detail.Traits {
						if trait.TraitID != "qibing_raohou" {
							continue
						}
						standardTriggered = true
						standardInfantry, standardInfantryOK := trait.Detail["infantryDefenseModifiedUnits"].(map[string]int)
						standardCavalry, standardCavalryOK := trait.Detail["cavalryDefenseModifiedUnits"].(map[string]int)
						designValue, designOK := trait.Detail["enemyDefenseReductionRate"].(float64)
						if !standardInfantryOK || !standardCavalryOK || !designOK || designValue != 0.2 || standardInfantry[defenderUnitType] != -2 || standardCavalry[defenderUnitType] != -2 {
							t.Fatalf("expected standard report defense deltas -2/-2, trait=%+v", trait)
						}
					}
				}
				if standardTriggered != tc.wantTriggered {
					t.Fatalf("expected standard report triggered=%t, detail=%+v", tc.wantTriggered, report.Detail)
				}
				if tc.wantTriggered {
					infantry, infantryOK := outcome.Detail["infantryDefenseModifiedUnits"].(map[string]int)
					cavalry, cavalryOK := outcome.Detail["cavalryDefenseModifiedUnits"].(map[string]int)
					designValue, designOK := outcome.Detail["enemyDefenseReductionRate"].(float64)
					if !infantryOK || !cavalryOK || !designOK || designValue != 0.2 || infantry[defenderUnitType] != -2 || cavalry[defenderUnitType] != -2 ||
						outcome.OwnerSide != "attacker" || outcome.OwnerGeneralID != "weiyan" {
						t.Fatalf("expected actual defense deltas -2/-2 owned by attacking Weiyan, outcome=%+v", outcome)
					}
				}
			}

			storedDefender, err := repo.GetState(defender.Player.ID)
			if err != nil {
				t.Fatalf("GetState defender failed: %v", err)
			}
			defenderLosses := pvpTestLossesFromBattle(t, battle, "defender")
			if got, want := armySliceToMap(storedDefender.Army)[defenderUnitType], 100-defenderLosses[defenderUnitType]; got != want {
				t.Fatalf("expected defender state %d to match battle losses, got %d losses=%+v", want, got, defenderLosses)
			}
		})
	}
}

// TestPvpDefenseOnlyTraitsOnlyTriggerForDefender 验证纯防御特性不会在进攻侧产生无效触发，只修改真实守军防御。
func TestPvpDefenseOnlyTraitsOnlyTriggerForDefender(t *testing.T) {
	traits := []struct {
		name               string
		traitID            string
		generalID          string
		generalName        string
		generalFaction     string
		opponentID         string
		opponentName       string
		opponentFaction    string
		params             map[string]float64
		wantDefensePower   float64
		wantInfantryChange int
		wantCavalryChange  int
	}{
		{
			name: "盾阵防御", traitID: "dunzhen_fangyu", generalID: "xiahouyuan", generalName: "夏侯渊", generalFaction: "wei",
			opponentID: "liubei", opponentName: "刘备", opponentFaction: "shu",
			params:           map[string]float64{"defenseBonusRate": 0.3, "triggerChance": 1},
			wantDefensePower: 1300, wantInfantryChange: 3, wantCavalryChange: 2,
		},
		{
			name: "固守汉中", traitID: "gushou_hanzhong", generalID: "weiyan", generalName: "魏延", generalFaction: "shu",
			opponentID: "caocao", opponentName: "曹操", opponentFaction: "wei",
			params:           map[string]float64{"generalDefenseFlat": 20, "triggerChance": 1},
			wantDefensePower: 3000, wantInfantryChange: 20, wantCavalryChange: 20,
		},
	}
	for _, tc := range traits {
		t.Run(tc.name, func(t *testing.T) {
			designKey := "defenseBonusRate"
			wantDesignValue := tc.params[designKey]
			if tc.params["generalDefenseFlat"] > 0 {
				designKey = "generalDefenseFlat"
				wantDesignValue = tc.params[designKey]
			}
			for _, carrierSide := range []string{"attacker", "defender"} {
				t.Run(carrierSide, func(t *testing.T) {
					setTestFactionsAndGenerals(t, FactionsConfig{
						tc.generalFaction:  {Name: tc.generalFaction, Generals: []GeneralInfo{{ID: tc.generalID, Name: tc.generalName}}},
						tc.opponentFaction: {Name: tc.opponentFaction, Generals: []GeneralInfo{{ID: tc.opponentID, Name: tc.opponentName}}},
					}, GeneralsConfig{Enabled: true, Heroes: map[string]GeneralHeroConfig{
						tc.generalID: {
							ID: tc.generalID, Name: tc.generalName, Faction: tc.generalFaction, Enabled: true,
							BonusTrait: GeneralTraitConfig{
								TraitID: tc.traitID, TraitType: general.TraitTypeBonus, Enabled: true, Scope: "self_army",
								AllowedSides: []string{"defender", "reinforcement"}, Params: tc.params,
							},
						},
						tc.opponentID: {ID: tc.opponentID, Name: tc.opponentName, Faction: tc.opponentFaction, Enabled: true},
					}})
					attackerFaction, attackerGeneralID := tc.opponentFaction, tc.opponentID
					defenderFaction, defenderGeneralID := tc.generalFaction, tc.generalID
					if carrierSide == "attacker" {
						attackerFaction, attackerGeneralID = tc.generalFaction, tc.generalID
						defenderFaction, defenderGeneralID = tc.opponentFaction, tc.opponentID
					}
					svc, repo, attacker, defender := newPvpTestServiceForGenerals(
						t, attackerFaction, attackerGeneralID, defenderFaction, defenderGeneralID,
					)
					attackerUnitType := attackerFaction + "Infantry"
					defenderUnitType := defenderFaction + "Infantry"
					attacker.Army = []ArmyUnit{{UnitType: attackerUnitType, Amount: 100}}
					defender.Army = []ArmyUnit{{UnitType: defenderUnitType, Amount: 100}}
					defender.Buildings = nil
					repo.players[attacker.Player.ID] = attacker
					repo.players[defender.Player.ID] = defender

					started, err := svc.StartPvpAttack(PvpAttackRequest{
						PlayerID: attacker.Player.ID, TargetPlayerID: defender.Player.ID,
						MarchMode: PvpMarchTypeAttack, Troops: map[string]int{attackerUnitType: 100}, GeneralIDs: []string{attackerGeneralID},
					})
					if err != nil {
						t.Fatalf("StartPvpAttack failed: %v", err)
					}
					forcePvpMarchDue(t, repo, started.March.ID)
					battle, err := svc.ResolvePvpMarch(started.March.ID)
					if err != nil {
						t.Fatalf("ResolvePvpMarch failed: %v", err)
					}
					wantTriggered := carrierSide == "defender"
					wantDefensePower := float64(1000)
					if wantTriggered {
						wantDefensePower = tc.wantDefensePower
					}
					defensePower, ok := battle.Result["defensePower"].(float64)
					if !ok || defensePower != wantDefensePower {
						t.Fatalf("expected defense power %.0f, got %+v", wantDefensePower, battle.Result)
					}

					attackerReports, _, err := repo.ListReports(attacker.Player.ID, 10, 0)
					if err != nil || len(attackerReports) == 0 || attackerReports[0].ID != battle.AttackerReportID {
						t.Fatalf("expected attacker report, reports=%+v err=%v", attackerReports, err)
					}
					defenderReports, _, err := repo.ListReports(defender.Player.ID, 10, 0)
					if err != nil || len(defenderReports) == 0 || defenderReports[0].ID != battle.DefenderReportID {
						t.Fatalf("expected defender report, reports=%+v err=%v", defenderReports, err)
					}
					for _, report := range []BattleReport{attackerReports[0], defenderReports[0]} {
						outcome, triggered := report.TraitOutcomes[tc.traitID]
						if triggered != wantTriggered {
							t.Fatalf("expected triggered=%t, outcomes=%+v", wantTriggered, report.TraitOutcomes)
						}
						standardTriggered := false
						if report.Detail != nil {
							for _, trait := range report.Detail.Traits {
								if trait.TraitID != tc.traitID {
									continue
								}
								standardTriggered = true
								standardInfantry, standardInfantryOK := trait.Detail["infantryDefenseModifiedUnits"].(map[string]int)
								standardCavalry, standardCavalryOK := trait.Detail["cavalryDefenseModifiedUnits"].(map[string]int)
								designValue, designOK := trait.Detail[designKey].(float64)
								if !standardInfantryOK || !standardCavalryOK || !designOK || designValue != wantDesignValue || standardInfantry[defenderUnitType] != tc.wantInfantryChange || standardCavalry[defenderUnitType] != tc.wantCavalryChange {
									t.Fatalf("expected standard report actual defense deltas, trait=%+v", trait)
								}
							}
						}
						if standardTriggered != wantTriggered {
							t.Fatalf("expected standard report triggered=%t, detail=%+v", wantTriggered, report.Detail)
						}
						if wantTriggered {
							infantry, infantryOK := outcome.Detail["infantryDefenseModifiedUnits"].(map[string]int)
							cavalry, cavalryOK := outcome.Detail["cavalryDefenseModifiedUnits"].(map[string]int)
							designValue, designOK := outcome.Detail[designKey].(float64)
							if !infantryOK || !cavalryOK || !designOK || designValue != wantDesignValue || infantry[defenderUnitType] != tc.wantInfantryChange || cavalry[defenderUnitType] != tc.wantCavalryChange ||
								outcome.OwnerSide != "defender" || outcome.OwnerGeneralID != tc.generalID {
								t.Fatalf("expected real defense deltas owned by defender, outcome=%+v", outcome)
							}
						}
					}

					storedDefender, err := repo.GetState(defender.Player.ID)
					if err != nil {
						t.Fatalf("GetState defender failed: %v", err)
					}
					defenderLosses := pvpTestLossesFromBattle(t, battle, "defender")
					if got, want := armySliceToMap(storedDefender.Army)[defenderUnitType], 100-defenderLosses[defenderUnitType]; got != want {
						t.Fatalf("expected defender state %d to match battle losses, got %d losses=%+v", want, got, defenderLosses)
					}
				})
			}
		})
	}
}

// TestPvpPreBattleStatTraitsOnlyTriggerOnEffectiveSide 验证战前属性特性只在核心实际使用该属性的方向触发。
func TestPvpPreBattleStatTraitsOnlyTriggerOnEffectiveSide(t *testing.T) {
	traits := []struct {
		name               string
		traitID            string
		traitType          string
		generalID          string
		generalName        string
		generalFaction     string
		opponentID         string
		opponentName       string
		opponentFaction    string
		activeSide         string
		params             map[string]float64
		wantAttackPower    float64
		wantDefensePower   float64
		wantAttackChange   int
		wantInfantryChange int
		wantCavalryChange  int
	}{
		{
			name: "谋定后发", traitID: "mouding_houfa", traitType: general.TraitTypeBonus,
			generalID: "simayi", generalName: "司马懿", generalFaction: "wei",
			opponentID: "liubei", opponentName: "刘备", opponentFaction: "shu", activeSide: "defender",
			params:          map[string]float64{"defenseBonusRate": 0.35, "triggerChance": 1},
			wantAttackPower: 1000, wantDefensePower: 1400, wantInfantryChange: 4, wantCavalryChange: 3,
		},
		{
			name: "魅惑扰阵", traitID: "meihuo_raozhen", traitType: general.TraitTypeBonus,
			generalID: "zhenmi", generalName: "甄宓", generalFaction: "wei",
			opponentID: "liubei", opponentName: "刘备", opponentFaction: "shu", activeSide: "attacker",
			params:          map[string]float64{"enemyDefenseReductionRate": 0.1, "triggerChance": 1},
			wantAttackPower: 1000, wantDefensePower: 900, wantInfantryChange: -1, wantCavalryChange: -1,
		},
		{
			name: "虎痴冲阵", traitID: "huchi_chongzhen", traitType: general.TraitTypeSpecial,
			generalID: "xuchu", generalName: "许褚", generalFaction: "wei",
			opponentID: "liubei", opponentName: "刘备", opponentFaction: "shu", activeSide: "attacker",
			params:          map[string]float64{"enemyDefenseReductionRate": 0.3, "triggerChance": 1},
			wantAttackPower: 1000, wantDefensePower: 700, wantInfantryChange: -3, wantCavalryChange: -2,
		},
		{
			name: "百步穿杨", traitID: "baibu_chuanyang", traitType: general.TraitTypeSpecial,
			generalID: "huangzhong", generalName: "黄忠", generalFaction: "shu",
			opponentID: "caocao", opponentName: "曹操", opponentFaction: "wei", activeSide: "attacker",
			params:          map[string]float64{"enemyDefenseReductionRate": 0.2, "triggerChance": 1},
			wantAttackPower: 1000, wantDefensePower: 800, wantInfantryChange: -2, wantCavalryChange: -2,
		},
	}
	for _, tc := range traits {
		t.Run(tc.name, func(t *testing.T) {
			designKey := "enemyDefenseReductionRate"
			wantDesignValue := tc.params[designKey]
			if tc.traitID == "mouding_houfa" {
				designKey = "defenseBonusRate"
				wantDesignValue = tc.params[designKey]
			} else if tc.wantAttackChange != 0 {
				designKey = "attackReductionRate"
				wantDesignValue = tc.params["effectRate"]
			}
			for _, carrierSide := range []string{"attacker", "defender"} {
				t.Run(carrierSide, func(t *testing.T) {
					traitScope := "enemy_army"
					if tc.traitID == "mouding_houfa" {
						traitScope = "self_army"
					}
					traitConfig := GeneralTraitConfig{
						TraitID: tc.traitID, TraitType: tc.traitType, Enabled: true, Scope: traitScope,
						AllowedSides: []string{tc.activeSide}, Params: tc.params,
					}
					carrierHero := GeneralHeroConfig{ID: tc.generalID, Name: tc.generalName, Faction: tc.generalFaction, Enabled: true}
					if tc.traitType == general.TraitTypeSpecial {
						carrierHero.SpecialTrait = traitConfig
					} else {
						carrierHero.BonusTrait = traitConfig
					}
					setTestFactionsAndGenerals(t, FactionsConfig{
						tc.generalFaction:  {Name: tc.generalFaction, Generals: []GeneralInfo{{ID: tc.generalID, Name: tc.generalName}}},
						tc.opponentFaction: {Name: tc.opponentFaction, Generals: []GeneralInfo{{ID: tc.opponentID, Name: tc.opponentName}}},
					}, GeneralsConfig{Enabled: true, Heroes: map[string]GeneralHeroConfig{
						tc.generalID:  carrierHero,
						tc.opponentID: {ID: tc.opponentID, Name: tc.opponentName, Faction: tc.opponentFaction, Enabled: true},
					}})
					attackerFaction, attackerGeneralID := tc.opponentFaction, tc.opponentID
					defenderFaction, defenderGeneralID := tc.generalFaction, tc.generalID
					if carrierSide == "attacker" {
						attackerFaction, attackerGeneralID = tc.generalFaction, tc.generalID
						defenderFaction, defenderGeneralID = tc.opponentFaction, tc.opponentID
					}
					svc, repo, attacker, defender := newPvpTestServiceForGenerals(
						t, attackerFaction, attackerGeneralID, defenderFaction, defenderGeneralID,
					)
					attackerUnitType := attackerFaction + "Infantry"
					defenderUnitType := defenderFaction + "Infantry"
					attacker.Army = []ArmyUnit{{UnitType: attackerUnitType, Amount: 100}}
					defender.Army = []ArmyUnit{{UnitType: defenderUnitType, Amount: 100}}
					defender.Buildings = nil
					repo.players[attacker.Player.ID] = attacker
					repo.players[defender.Player.ID] = defender

					started, err := svc.StartPvpAttack(PvpAttackRequest{
						PlayerID: attacker.Player.ID, TargetPlayerID: defender.Player.ID,
						MarchMode: PvpMarchTypeAttack, Troops: map[string]int{attackerUnitType: 100}, GeneralIDs: []string{attackerGeneralID},
					})
					if err != nil {
						t.Fatalf("StartPvpAttack failed: %v", err)
					}
					forcePvpMarchDue(t, repo, started.March.ID)
					battle, err := svc.ResolvePvpMarch(started.March.ID)
					if err != nil {
						t.Fatalf("ResolvePvpMarch failed: %v", err)
					}
					wantTriggered := carrierSide == tc.activeSide
					wantAttackPower, wantDefensePower := float64(1000), float64(1000)
					if wantTriggered {
						wantAttackPower, wantDefensePower = tc.wantAttackPower, tc.wantDefensePower
					}
					attackPower, attackOK := battle.Result["attackerPower"].(float64)
					defensePower, defenseOK := battle.Result["defensePower"].(float64)
					if !attackOK || !defenseOK || attackPower != wantAttackPower || defensePower != wantDefensePower {
						t.Fatalf("expected powers %.0f/%.0f, got %+v", wantAttackPower, wantDefensePower, battle.Result)
					}

					attackerReports, _, err := repo.ListReports(attacker.Player.ID, 10, 0)
					if err != nil || len(attackerReports) == 0 || attackerReports[0].ID != battle.AttackerReportID {
						t.Fatalf("expected attacker report, reports=%+v err=%v", attackerReports, err)
					}
					defenderReports, _, err := repo.ListReports(defender.Player.ID, 10, 0)
					if err != nil || len(defenderReports) == 0 || defenderReports[0].ID != battle.DefenderReportID {
						t.Fatalf("expected defender report, reports=%+v err=%v", defenderReports, err)
					}
					targetUnitType := defenderUnitType
					if tc.wantAttackChange != 0 {
						targetUnitType = attackerUnitType
					}
					assertDetail := func(detail map[string]interface{}) {
						designValue, designOK := detail[designKey].(float64)
						if !designOK || designValue != wantDesignValue {
							t.Fatalf("expected design value %s=%.2f, detail=%+v", designKey, wantDesignValue, detail)
						}
						if tc.wantAttackChange != 0 {
							values, ok := detail["attackModifiedUnits"].(map[string]int)
							if !ok || values[targetUnitType] != tc.wantAttackChange {
								t.Fatalf("expected actual attack delta %d, detail=%+v", tc.wantAttackChange, detail)
							}
							return
						}
						infantry, infantryOK := detail["infantryDefenseModifiedUnits"].(map[string]int)
						cavalry, cavalryOK := detail["cavalryDefenseModifiedUnits"].(map[string]int)
						if !infantryOK || !cavalryOK || infantry[targetUnitType] != tc.wantInfantryChange || cavalry[targetUnitType] != tc.wantCavalryChange {
							t.Fatalf("expected actual defense deltas %d/%d, detail=%+v", tc.wantInfantryChange, tc.wantCavalryChange, detail)
						}
					}
					for _, report := range []BattleReport{attackerReports[0], defenderReports[0]} {
						outcome, triggered := report.TraitOutcomes[tc.traitID]
						if triggered != wantTriggered {
							t.Fatalf("expected triggered=%t, outcomes=%+v", wantTriggered, report.TraitOutcomes)
						}
						standardTriggered := false
						if report.Detail != nil {
							for _, trait := range report.Detail.Traits {
								if trait.TraitID != tc.traitID {
									continue
								}
								standardTriggered = true
								assertDetail(trait.Detail)
							}
						}
						if standardTriggered != wantTriggered {
							t.Fatalf("expected standard report triggered=%t, detail=%+v", wantTriggered, report.Detail)
						}
						if wantTriggered {
							assertDetail(outcome.Detail)
							if outcome.OwnerSide != tc.activeSide || outcome.OwnerGeneralID != tc.generalID {
								t.Fatalf("expected outcome owned by %s %s, got %+v", tc.activeSide, tc.generalID, outcome)
							}
						}
					}

					storedDefender, err := repo.GetState(defender.Player.ID)
					if err != nil {
						t.Fatalf("GetState defender failed: %v", err)
					}
					defenderLosses := pvpTestLossesFromBattle(t, battle, "defender")
					if got, want := armySliceToMap(storedDefender.Army)[defenderUnitType], 100-defenderLosses[defenderUnitType]; got != want {
						t.Fatalf("expected defender state %d to match battle losses, got %d losses=%+v", want, got, defenderLosses)
					}
				})
			}
		})
	}
}

// TestPvpAttackOnlyTraitsDoNotTriggerForDefender 验证六项纯攻击加成只在主动进攻时进入真实攻击战力。
func TestPvpAttackOnlyTraitsDoNotTriggerForDefender(t *testing.T) {
	traits := []struct {
		name             string
		traitID          string
		generalID        string
		generalName      string
		generalFaction   string
		opponentID       string
		opponentName     string
		opponentFaction  string
		targetUnitType   string
		attackUnitType   string
		attackCategory   string
		params           map[string]float64
		wantAttackPower  float64
		wantAttackChange int
	}{
		{
			name: "死战到底", traitID: "sizhandaodi", generalID: "dianwei", generalName: "典韦", generalFaction: "wei",
			opponentID: "liubei", opponentName: "刘备", opponentFaction: "shu", targetUnitType: "infantry",
			params: map[string]float64{"attackBonusRate": 0.35, "triggerChance": 1}, wantAttackPower: 1400, wantAttackChange: 4,
		},
		{
			name: "威震逍遥", traitID: "weizhen_xiaoyao", generalID: "zhangliao", generalName: "张辽", generalFaction: "wei",
			opponentID: "liubei", opponentName: "刘备", opponentFaction: "shu", targetUnitType: "cavalry", attackUnitType: "weiCavalry", attackCategory: "cavalry",
			params: map[string]float64{"attackBonusRate": 0.35, "triggerChance": 1}, wantAttackPower: 1400, wantAttackChange: 4,
		},
		{
			name: "武圣破军", traitID: "wusheng_pojun", generalID: "guanyu", generalName: "关羽", generalFaction: "shu",
			opponentID: "caocao", opponentName: "曹操", opponentFaction: "wei",
			params: map[string]float64{"attackBonusRate": 0.2, "triggerChance": 1}, wantAttackPower: 1200, wantAttackChange: 2,
		},
		{
			name: "万人怒吼", traitID: "wanren_nuhou", generalID: "zhangfei", generalName: "张飞", generalFaction: "shu",
			opponentID: "caocao", opponentName: "曹操", opponentFaction: "wei", targetUnitType: "infantry",
			params: map[string]float64{"attackBonusRate": 0.2, "triggerChance": 1}, wantAttackPower: 1200, wantAttackChange: 2,
		},
		{
			name: "小霸王", traitID: "xiaobawang_tieqi", generalID: "sunce", generalName: "孙策", generalFaction: "wu",
			opponentID: "caocao", opponentName: "曹操", opponentFaction: "wei", targetUnitType: "overlordRider", attackUnitType: "overlordRider", attackCategory: "cavalry",
			params: map[string]float64{"unitAttackFlat": 50, "triggerChance": 1}, wantAttackPower: 6000, wantAttackChange: 50,
		},
		{
			name: "美周郎军略", traitID: "meizhoulang_junlue", generalID: "zhouyu", generalName: "周瑜", generalFaction: "wu",
			opponentID: "caocao", opponentName: "曹操", opponentFaction: "wei",
			params: map[string]float64{"attackBonusRate": 0.05, "triggerChance": 1}, wantAttackPower: 1100, wantAttackChange: 1,
		},
	}
	for _, tc := range traits {
		t.Run(tc.name, func(t *testing.T) {
			for _, carrierSide := range []string{"attacker", "defender"} {
				t.Run(carrierSide, func(t *testing.T) {
					designKey := "attackBonusRate"
					wantDesignValue := tc.params[designKey]
					if tc.params["unitAttackFlat"] > 0 {
						designKey = "unitAttackFlat"
						wantDesignValue = tc.params[designKey]
					}
					setTestFactionsAndGenerals(t, FactionsConfig{
						tc.generalFaction:  {Name: tc.generalFaction, Generals: []GeneralInfo{{ID: tc.generalID, Name: tc.generalName}}},
						tc.opponentFaction: {Name: tc.opponentFaction, Generals: []GeneralInfo{{ID: tc.opponentID, Name: tc.opponentName}}},
					}, GeneralsConfig{Enabled: true, Heroes: map[string]GeneralHeroConfig{
						tc.generalID: {
							ID: tc.generalID, Name: tc.generalName, Faction: tc.generalFaction, Enabled: true,
							BonusTrait: GeneralTraitConfig{
								TraitID: tc.traitID, TraitType: general.TraitTypeBonus, Enabled: true, Scope: "self_army",
								TargetUnitType: tc.targetUnitType, AllowedSides: []string{"attacker"}, Params: tc.params,
							},
						},
						tc.opponentID: {ID: tc.opponentID, Name: tc.opponentName, Faction: tc.opponentFaction, Enabled: true},
					}})
					attackerFaction, attackerGeneralID := tc.opponentFaction, tc.opponentID
					defenderFaction, defenderGeneralID := tc.generalFaction, tc.generalID
					if carrierSide == "attacker" {
						attackerFaction, attackerGeneralID = tc.generalFaction, tc.generalID
						defenderFaction, defenderGeneralID = tc.opponentFaction, tc.opponentID
					}
					svc, repo, attacker, defender := newPvpTestServiceForGenerals(
						t, attackerFaction, attackerGeneralID, defenderFaction, defenderGeneralID,
					)
					attackerUnitType := attackerFaction + "Infantry"
					if carrierSide == "attacker" && tc.attackUnitType != "" {
						attackerUnitType = tc.attackUnitType
						unitsMu.Lock()
						activeUnits[attackerFaction][attackerUnitType] = UnitConfig{
							Name: tc.name + "测试兵", Category: tc.attackCategory,
							Stats: map[string]int{"attack": 10, "infantryDefense": 10, "cavalryDefense": 8, "carryCapacity": 5, "upkeep": 1},
						}
						unitsMu.Unlock()
					}
					defenderUnitType := defenderFaction + "Infantry"
					attacker.Army = []ArmyUnit{{UnitType: attackerUnitType, Amount: 100}}
					defender.Army = []ArmyUnit{{UnitType: defenderUnitType, Amount: 100}}
					defender.Buildings = nil
					repo.players[attacker.Player.ID] = attacker
					repo.players[defender.Player.ID] = defender

					started, err := svc.StartPvpAttack(PvpAttackRequest{
						PlayerID: attacker.Player.ID, TargetPlayerID: defender.Player.ID,
						MarchMode: PvpMarchTypeAttack, Troops: map[string]int{attackerUnitType: 100}, GeneralIDs: []string{attackerGeneralID},
					})
					if err != nil {
						t.Fatalf("StartPvpAttack failed: %v", err)
					}
					forcePvpMarchDue(t, repo, started.March.ID)
					battle, err := svc.ResolvePvpMarch(started.March.ID)
					if err != nil {
						t.Fatalf("ResolvePvpMarch failed: %v", err)
					}
					wantTriggered := carrierSide == "attacker"
					wantAttackPower := float64(1000)
					if wantTriggered {
						wantAttackPower = tc.wantAttackPower
					}
					attackPower, ok := battle.Result["attackerPower"].(float64)
					if !ok || attackPower != wantAttackPower {
						t.Fatalf("expected attack power %.0f, got %+v", wantAttackPower, battle.Result)
					}

					attackerReports, _, err := repo.ListReports(attacker.Player.ID, 10, 0)
					if err != nil || len(attackerReports) == 0 || attackerReports[0].ID != battle.AttackerReportID {
						t.Fatalf("expected attacker report, reports=%+v err=%v", attackerReports, err)
					}
					defenderReports, _, err := repo.ListReports(defender.Player.ID, 10, 0)
					if err != nil || len(defenderReports) == 0 || defenderReports[0].ID != battle.DefenderReportID {
						t.Fatalf("expected defender report, reports=%+v err=%v", defenderReports, err)
					}
					for _, report := range []BattleReport{attackerReports[0], defenderReports[0]} {
						outcome, triggered := report.TraitOutcomes[tc.traitID]
						if triggered != wantTriggered {
							t.Fatalf("expected triggered=%t, outcomes=%+v", wantTriggered, report.TraitOutcomes)
						}
						standardTriggered := false
						if report.Detail != nil {
							for _, trait := range report.Detail.Traits {
								if trait.TraitID != tc.traitID {
									continue
								}
								standardTriggered = true
								modified, detailOK := trait.Detail["attackModifiedUnits"].(map[string]int)
								designValue, designOK := trait.Detail[designKey].(float64)
								if !detailOK || !designOK || designValue != wantDesignValue || modified[attackerUnitType] != tc.wantAttackChange {
									t.Fatalf("expected standard report attack delta %d, trait=%+v", tc.wantAttackChange, trait)
								}
							}
						}
						if standardTriggered != wantTriggered {
							t.Fatalf("expected standard report triggered=%t, detail=%+v", wantTriggered, report.Detail)
						}
						if wantTriggered {
							modified, detailOK := outcome.Detail["attackModifiedUnits"].(map[string]int)
							designValue, designOK := outcome.Detail[designKey].(float64)
							if !detailOK || !designOK || designValue != wantDesignValue || modified[attackerUnitType] != tc.wantAttackChange || outcome.OwnerSide != "attacker" || outcome.OwnerGeneralID != tc.generalID {
								t.Fatalf("expected attacking general actual attack delta %d, outcome=%+v", tc.wantAttackChange, outcome)
							}
						}
					}

					storedDefender, err := repo.GetState(defender.Player.ID)
					if err != nil {
						t.Fatalf("GetState defender failed: %v", err)
					}
					defenderLosses := pvpTestLossesFromBattle(t, battle, "defender")
					if got, want := armySliceToMap(storedDefender.Army)[defenderUnitType], 100-defenderLosses[defenderUnitType]; got != want {
						t.Fatalf("expected defender state %d to match battle losses, got %d losses=%+v", want, got, defenderLosses)
					}
				})
			}
		})
	}
}

func TestPvpBattleCreatesReinforcementOwnerReport(t *testing.T) {
	svc, repo, attacker, defender := newPvpTestService(t)
	now := time.Now()
	helperAccount := Account{ID: "account_pvp_helper", Username: "pvp_helper", PasswordHash: "hash", CreatedAt: now}
	if err := repo.CreateAccount(helperAccount); err != nil {
		t.Fatalf("CreateAccount helper failed: %v", err)
	}
	helper := newPlayerState("player_pvp_helper", "援军方", "wu", "sunquan", now)
	if err := repo.CreatePlayer(helperAccount.ID, helper, now); err != nil {
		t.Fatalf("CreatePlayer helper failed: %v", err)
	}
	attacker.Army = []ArmyUnit{{UnitType: "weiInfantry", Amount: 400}}
	defender.Army = []ArmyUnit{{UnitType: "weiInfantry", Amount: 1}}
	repo.players[attacker.Player.ID] = attacker
	repo.players[defender.Player.ID] = defender
	reinforcement := Reinforcement{
		ID:                "reinforcement_pvp_report",
		FromPlayerID:      helper.Player.ID,
		FromPlayerName:    helper.Player.Nickname,
		FromPlayerFaction: helper.Player.Faction,
		ToPlayerID:        defender.Player.ID,
		ToPlayerName:      defender.Player.Nickname,
		ToPlayerFaction:   defender.Player.Faction,
		OwnerPlayerID:     helper.Player.ID,
		HostPlayerID:      defender.Player.ID,
		SourceType:        GarrisonSourceReinforcement,
		SourceID:          "reinforcement_pvp_report",
		TargetType:        ReinforcementTargetPlayerCity,
		TargetID:          defender.Player.ID,
		Status:            ReinforcementStatusStationed,
		Troops:            map[string]int{"weiInfantry": 20},
		RemainingTroops:   map[string]int{"weiInfantry": 20},
		Generals: []ReinforcementGeneralSnapshot{{
			ID:    "sunquan",
			Name:  "孙权",
			Level: 1,
		}},
		Losses:    map[string]int{},
		Rules:     defaultGarrisonRules(GarrisonSourceReinforcement),
		SentAt:    now.Add(-4 * time.Hour).UTC().Format(resourceDateLayout),
		ArrivedAt: now.Add(-3 * time.Hour).UTC().Format(resourceDateLayout),
		CreatedAt: now.Add(-4 * time.Hour).UTC().Format(resourceDateLayout),
		UpdatedAt: now.Add(-3 * time.Hour).UTC().Format(resourceDateLayout),
	}
	repo.reinforcements[reinforcement.ID] = reinforcement

	started, err := svc.StartPvpAttack(PvpAttackRequest{
		PlayerID:       attacker.Player.ID,
		TargetPlayerID: defender.Player.ID,
		MarchMode:      PvpMarchTypeAttack,
		Troops:         map[string]int{"weiInfantry": 300},
	})
	if err != nil {
		t.Fatalf("StartPvpAttack failed: %v", err)
	}
	forcePvpMarchDue(t, repo, started.March.ID)
	battle, err := svc.ResolvePvpMarch(started.March.ID)
	if err != nil {
		t.Fatalf("ResolvePvpMarch failed: %v", err)
	}
	helperReports, total, err := repo.ListReportsByQuery(BattleReportQuery{PlayerID: helper.Player.ID, ViewType: ReportViewReinforcement, Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("ListReportsByQuery helper failed: %v", err)
	}
	if total != 1 || len(helperReports) != 1 {
		t.Fatalf("expected one helper reinforcement report, total=%d reports=%+v battle=%+v", total, helperReports, battle)
	}
	if helperReports[0].EventID != battle.ID || helperReports[0].Detail == nil || helperReports[0].Detail.SecondarySide == nil {
		t.Fatalf("unexpected helper report detail: %+v", helperReports[0])
	}
	if helperReports[0].GeneralExpGained <= 0 || len(helperReports[0].PvpReinforcements) != 1 || helperReports[0].PvpReinforcements[0].GeneralExpGained != helperReports[0].GeneralExpGained {
		t.Fatalf("expected reinforcement report and snapshot to use the same positive exp, report=%+v", helperReports[0])
	}
	if helperReports[0].Detail.PrimarySide.Role != "attacker" || helperReports[0].Detail.PrimarySide.TargetID != attacker.Player.ID || helperReports[0].Detail.SecondarySide.Role != "defender" || helperReports[0].Detail.SecondarySide.PlayerID != defender.Player.ID {
		t.Fatalf("expected helper report to preserve the same attacker and defender snapshots, got %+v", helperReports[0].Detail)
	}
	if len(helperReports[0].PvpReinforcements) != 1 || helperReports[0].PvpReinforcements[0].ReinforcementID != reinforcement.ID {
		t.Fatalf("expected helper report to preserve all reinforcement snapshots, got %+v", helperReports[0].PvpReinforcements)
	}
	reinforcementExtra, ok := helperReports[0].Detail.Extra["reinforcement"].(map[string]interface{})
	if !ok || reinforcementExtra["hostPlayerId"] != defender.Player.ID || reinforcementExtra["attackerPlayerId"] != attacker.Player.ID {
		t.Fatalf("expected reinforcement context with host and attacker, got %+v", helperReports[0].Detail.Extra)
	}
	updated := repo.reinforcements[reinforcement.ID]
	if updated.LastBattleReportID != helperReports[0].ID {
		t.Fatalf("expected reinforcement last report %s, got %+v", helperReports[0].ID, updated)
	}
	eventReports, err := svc.ListReportsByEventForAdmin(battle.ID)
	if err != nil {
		t.Fatalf("ListReportsByEventForAdmin failed: %v", err)
	}
	if len(eventReports) != 3 {
		t.Fatalf("expected attacker, defender and reinforcement reports for event, got %+v", eventReports)
	}
	helperAfterFirst, err := repo.GetState(helper.Player.ID)
	if err != nil {
		t.Fatalf("GetState helper after first resolve failed: %v", err)
	}
	helperExpAfterFirst := pvpTestGeneralExp(helperAfterFirst, "sunquan")
	if helperExpAfterFirst <= 0 {
		t.Fatalf("expected reinforcement general to gain exp, got %d", helperExpAfterFirst)
	}
	attackerPvpAfterFirst, err := svc.GetPvpState(attacker.Player.ID)
	if err != nil {
		t.Fatalf("GetPvpState attacker after first resolve failed: %v", err)
	}
	defenderPvpAfterFirst, err := svc.GetPvpState(defender.Player.ID)
	if err != nil {
		t.Fatalf("GetPvpState defender after first resolve failed: %v", err)
	}
	storedMarch, err := repo.GetPvpMarch(started.March.ID)
	if err != nil {
		t.Fatalf("GetPvpMarch failed: %v", err)
	}
	if storedMarch.Status == PvpMarchStatusReturning {
		forcePvpReturnDue(t, repo, storedMarch.ID)
		if _, err := svc.CompletePvpRecall(storedMarch.ID); err != nil {
			t.Fatalf("CompletePvpRecall failed: %v", err)
		}
	}
	if _, err := svc.ResolvePvpMarch(started.March.ID); err != nil {
		t.Fatalf("repeated resolved PVP read failed: %v", err)
	}
	helperAfterRepeat, err := repo.GetState(helper.Player.ID)
	if err != nil {
		t.Fatalf("GetState helper after repeated resolve failed: %v", err)
	}
	attackerPvpAfterRepeat, err := svc.GetPvpState(attacker.Player.ID)
	if err != nil {
		t.Fatalf("GetPvpState attacker after repeated resolve failed: %v", err)
	}
	defenderPvpAfterRepeat, err := svc.GetPvpState(defender.Player.ID)
	if err != nil {
		t.Fatalf("GetPvpState defender after repeated resolve failed: %v", err)
	}
	if got := pvpTestGeneralExp(helperAfterRepeat, "sunquan"); got != helperExpAfterFirst {
		t.Fatalf("expected reinforcement exp to stay %d after repeated resolve, got %d", helperExpAfterFirst, got)
	}
	if attackerPvpAfterRepeat.SeasonPoints != attackerPvpAfterFirst.SeasonPoints || defenderPvpAfterRepeat.SeasonPoints != defenderPvpAfterFirst.SeasonPoints {
		t.Fatalf("expected PVP points unchanged after repeated resolve, before=%d/%d after=%d/%d", attackerPvpAfterFirst.SeasonPoints, defenderPvpAfterFirst.SeasonPoints, attackerPvpAfterRepeat.SeasonPoints, defenderPvpAfterRepeat.SeasonPoints)
	}
	helperReports, total, err = repo.ListReportsByQuery(BattleReportQuery{PlayerID: helper.Player.ID, ViewType: ReportViewReinforcement, Page: 1, PageSize: 10})
	if err != nil || total != 1 || len(helperReports) != 1 {
		t.Fatalf("expected repeated resolve not to duplicate reinforcement report, total=%d reports=%+v err=%v", total, helperReports, err)
	}
}

func TestAdminSettlePvpSeasonSnapshotsRankingsAndSendsRewardMail(t *testing.T) {
	svc, repo, attacker, defender := newPvpTestService(t)
	attacker.Army = []ArmyUnit{{UnitType: "weiInfantry", Amount: 100}}
	defender.Army = []ArmyUnit{{UnitType: "weiInfantry", Amount: 5}}
	repo.players[attacker.Player.ID] = attacker
	repo.players[defender.Player.ID] = defender

	started, err := svc.StartPvpAttack(PvpAttackRequest{
		PlayerID:       attacker.Player.ID,
		TargetPlayerID: defender.Player.ID,
		MarchMode:      PvpMarchTypeAttack,
		Troops:         map[string]int{"weiInfantry": 50},
	})
	if err != nil {
		t.Fatalf("StartPvpAttack failed: %v", err)
	}
	forcePvpMarchDue(t, repo, started.March.ID)
	if _, err := svc.ResolvePvpMarch(started.March.ID); err != nil {
		t.Fatalf("ResolvePvpMarch failed: %v", err)
	}

	overview, err := svc.AdminPvpSeasons()
	if err != nil {
		t.Fatalf("AdminPvpSeasons failed: %v", err)
	}
	if overview.Current.ID == "" || overview.Current.Status != PvpSeasonStatusActive {
		t.Fatalf("unexpected current season: %+v", overview.Current)
	}
	settled, err := svc.AdminSettlePvpSeason(overview.Current.ID)
	if err != nil {
		t.Fatalf("AdminSettlePvpSeason failed: %v", err)
	}
	if settled.Season.Status != PvpSeasonStatusSettled || settled.Season.SettledAt == "" {
		t.Fatalf("expected settled season, got %+v", settled.Season)
	}
	if len(settled.Players) < 2 || settled.Players[0].PlayerID != attacker.Player.ID || settled.Players[0].RewardMailID == "" {
		t.Fatalf("unexpected season players: %+v", settled.Players)
	}
	mails, total, err := repo.ListMails(attacker.Player.ID, PvpSeasonRewardMailType, 10, 0)
	if err != nil {
		t.Fatalf("ListMails failed: %v", err)
	}
	if total != 1 || len(mails) != 1 || mails[0].MailType != PvpSeasonRewardMailType || len(mails[0].Attachments) != 1 {
		t.Fatalf("expected one pvp reward mail, mails=%+v total=%d", mails, total)
	}
	again, err := svc.AdminSettlePvpSeason(overview.Current.ID)
	if err != nil {
		t.Fatalf("AdminSettlePvpSeason again failed: %v", err)
	}
	if again.RewardMail != 0 {
		t.Fatalf("settled season should not resend rewards, got %+v", again)
	}
}

func TestAdminCreateAndUpdatePvpSeason(t *testing.T) {
	svc, _, _, _ := newPvpTestService(t)
	created, err := svc.AdminCreatePvpSeason(AdminSavePvpSeasonRequest{
		ID:       "season_custom",
		Name:     "自定义赛季",
		Status:   PvpSeasonStatusActive,
		StartsAt: "2026-07-01T00:00:00Z",
		EndsAt:   "2026-08-01T00:00:00Z",
		Rewards:  map[string]any{"rank1CityGold": 1500},
	})
	if err != nil {
		t.Fatalf("AdminCreatePvpSeason failed: %v", err)
	}
	if created.ID != "season_custom" || created.Name != "自定义赛季" {
		t.Fatalf("unexpected created season: %+v", created)
	}
	updated, err := svc.AdminUpdatePvpSeason("season_custom", AdminSavePvpSeasonRequest{
		Name:     "自定义赛季二期",
		Status:   PvpSeasonStatusActive,
		StartsAt: "2026-07-01T00:00:00Z",
		EndsAt:   "2026-08-15T00:00:00Z",
	})
	if err != nil {
		t.Fatalf("AdminUpdatePvpSeason failed: %v", err)
	}
	if updated.Name != "自定义赛季二期" || updated.CreatedAt != created.CreatedAt {
		t.Fatalf("unexpected updated season: %+v created=%+v", updated, created)
	}
}

func TestPvpMarchRecallReturnsTroopsWhenReturnDue(t *testing.T) {
	svc, repo, attacker, defender := newPvpTestService(t)
	attacker.Army = []ArmyUnit{{UnitType: "weiInfantry", Amount: 100}}
	repo.players[attacker.Player.ID] = attacker
	repo.players[defender.Player.ID] = defender

	started, err := svc.StartPvpAttack(PvpAttackRequest{
		PlayerID:       attacker.Player.ID,
		TargetPlayerID: defender.Player.ID,
		MarchMode:      PvpMarchTypeAttack,
		Troops:         map[string]int{"weiInfantry": 40},
	})
	if err != nil {
		t.Fatalf("StartPvpAttack failed: %v", err)
	}
	recalled, err := svc.RecallPvpMarch(attacker.Player.ID, started.March.ID)
	if err != nil {
		t.Fatalf("RecallPvpMarch failed: %v", err)
	}
	if recalled.March.Status != PvpMarchStatusReturning {
		t.Fatalf("expected returning march, got %+v", recalled.March)
	}
	afterRecall, err := repo.GetState(attacker.Player.ID)
	if err != nil {
		t.Fatalf("GetState after recall failed: %v", err)
	}
	if armySliceToMap(afterRecall.Army)["weiInfantry"] != 60 {
		t.Fatalf("expected troops still out while returning, got %+v", afterRecall.Army)
	}

	forcePvpReturnDue(t, repo, started.March.ID)
	completed, err := svc.CompletePvpRecall(started.March.ID)
	if err != nil {
		t.Fatalf("CompletePvpRecall failed: %v", err)
	}
	if completed.March.Status != PvpMarchStatusRecalled {
		t.Fatalf("expected recalled march, got %+v", completed.March)
	}
	if armySliceToMap(completed.Army)["weiInfantry"] != 100 {
		t.Fatalf("expected troops returned, got %+v", completed.Army)
	}
}

func TestPvpMarchRecallExpiresAfterTwoMinutes(t *testing.T) {
	svc, repo, attacker, defender := newPvpTestService(t)
	attacker.Army = []ArmyUnit{{UnitType: "weiInfantry", Amount: 100}}
	repo.players[attacker.Player.ID] = attacker
	repo.players[defender.Player.ID] = defender

	started, err := svc.StartPvpAttack(PvpAttackRequest{
		PlayerID:       attacker.Player.ID,
		TargetPlayerID: defender.Player.ID,
		MarchMode:      PvpMarchTypeAttack,
		Troops:         map[string]int{"weiInfantry": 40},
	})
	if err != nil {
		t.Fatalf("StartPvpAttack failed: %v", err)
	}
	agePvpMarch(t, repo, started.March.ID, 3*time.Minute)
	if _, err := svc.RecallPvpMarch(attacker.Player.ID, started.March.ID); !errors.Is(err, ErrPvpMarchNotRecallable) {
		t.Fatalf("expected recall window error, got %v", err)
	}
}

func TestMilitaryViewSettlesDuePvpReturn(t *testing.T) {
	svc, repo, attacker, defender := newPvpTestService(t)
	attacker.Army = []ArmyUnit{{UnitType: "weiInfantry", Amount: 100}}
	repo.players[attacker.Player.ID] = attacker
	repo.players[defender.Player.ID] = defender

	started, err := svc.StartPvpAttack(PvpAttackRequest{
		PlayerID:       attacker.Player.ID,
		TargetPlayerID: defender.Player.ID,
		MarchMode:      PvpMarchTypeAttack,
		Troops:         map[string]int{"weiInfantry": 40},
	})
	if err != nil {
		t.Fatalf("StartPvpAttack failed: %v", err)
	}
	if _, err := svc.RecallPvpMarch(attacker.Player.ID, started.March.ID); err != nil {
		t.Fatalf("RecallPvpMarch failed: %v", err)
	}
	forcePvpReturnDue(t, repo, started.March.ID)

	view, err := svc.GetMilitaryView(attacker.Player.ID)
	if err != nil {
		t.Fatalf("GetMilitaryView failed: %v", err)
	}
	if armySliceToMap(view.Army)["weiInfantry"] != 100 {
		t.Fatalf("expected military view to return due pvp troops, got %+v", view.Army)
	}
	completed, err := repo.GetPvpMarch(started.March.ID)
	if err != nil {
		t.Fatalf("GetPvpMarch completed failed: %v", err)
	}
	if completed.Status != PvpMarchStatusRecalled {
		t.Fatalf("expected recalled status after military view, got %+v", completed)
	}
}

func TestAdminForceResolvePvpMarchBeforeArrival(t *testing.T) {
	svc, repo, attacker, defender := newPvpTestService(t)
	attacker.Army = []ArmyUnit{{UnitType: "weiInfantry", Amount: 100}}
	defender.Army = []ArmyUnit{{UnitType: "weiInfantry", Amount: 1}}
	repo.players[attacker.Player.ID] = attacker
	repo.players[defender.Player.ID] = defender
	started, err := svc.StartPvpAttack(PvpAttackRequest{
		PlayerID:       attacker.Player.ID,
		TargetPlayerID: defender.Player.ID,
		Troops:         map[string]int{"weiInfantry": 10},
	})
	if err != nil {
		t.Fatalf("StartPvpAttack failed: %v", err)
	}
	if _, err := svc.ResolvePvpMarch(started.March.ID); !errors.Is(err, ErrPvpMarchNotReady) {
		t.Fatalf("expected normal resolve to wait for arrival, got %v", err)
	}
	battle, err := svc.AdminForceResolvePvpMarch(started.March.ID)
	if err != nil {
		t.Fatalf("AdminForceResolvePvpMarch failed: %v", err)
	}
	if battle.ID == "" || battle.Status != PvpBattleStatusResolved {
		t.Fatalf("expected resolved battle, got %+v", battle)
	}
	march, err := repo.GetPvpMarch(started.March.ID)
	if err != nil {
		t.Fatalf("GetPvpMarch failed: %v", err)
	}
	if march.Status != PvpMarchStatusReturning || march.BattleID != battle.ID {
		t.Fatalf("expected force resolved battle to enter return, got %+v", march)
	}
}

func TestPvpMarchReleasesGeneralsWhenAttackersAnnihilated(t *testing.T) {
	svc, repo, attacker, defender := newPvpTestService(t)
	attacker.Army = []ArmyUnit{{UnitType: "weiInfantry", Amount: 10}}
	defender.Army = []ArmyUnit{{UnitType: "weiInfantry", Amount: 10000}}
	repo.players[attacker.Player.ID] = attacker
	repo.players[defender.Player.ID] = defender

	started, err := svc.StartPvpAttack(PvpAttackRequest{
		PlayerID:       attacker.Player.ID,
		TargetPlayerID: defender.Player.ID,
		Troops:         map[string]int{"weiInfantry": 10},
		GeneralIDs:     []string{"caocao"},
	})
	if err != nil {
		t.Fatalf("StartPvpAttack failed: %v", err)
	}
	forcePvpMarchDue(t, repo, started.March.ID)
	if _, err := svc.ResolvePvpMarch(started.March.ID); err != nil {
		t.Fatalf("ResolvePvpMarch failed: %v", err)
	}
	march, err := repo.GetPvpMarch(started.March.ID)
	if err != nil {
		t.Fatalf("GetPvpMarch failed: %v", err)
	}
	if march.Status != PvpMarchStatusResolved || totalTroops(march.AttackTroops) != 0 {
		t.Fatalf("expected annihilated attackers to finish without return, got %+v", march)
	}
	state, err := repo.GetState(attacker.Player.ID)
	if err != nil {
		t.Fatalf("GetState failed: %v", err)
	}
	for _, assignment := range state.GeneralAssignments {
		if assignment.ModuleID == PVPModuleID {
			t.Fatalf("expected pvp general assignment released when all troops die, got %+v", state.GeneralAssignments)
		}
	}
}

func TestPvpMarchResolvesWhenDefenderHasNoTroops(t *testing.T) {
	svc, repo, attacker, defender := newPvpTestService(t)
	attacker.Army = []ArmyUnit{{UnitType: "weiInfantry", Amount: 100}}
	defender.Army = nil
	repo.players[attacker.Player.ID] = attacker
	repo.players[defender.Player.ID] = defender

	started, err := svc.StartPvpAttack(PvpAttackRequest{
		PlayerID:       attacker.Player.ID,
		TargetPlayerID: defender.Player.ID,
		Troops:         map[string]int{"weiInfantry": 40},
	})
	if err != nil {
		t.Fatalf("StartPvpAttack failed: %v", err)
	}
	forcePvpMarchDue(t, repo, started.March.ID)
	if err := svc.SettleDuePvpMarches(attacker.Player.ID); err != nil {
		t.Fatalf("SettleDuePvpMarches failed: %v", err)
	}
	march, err := repo.GetPvpMarch(started.March.ID)
	if err != nil {
		t.Fatalf("GetPvpMarch failed: %v", err)
	}
	if march.Status != PvpMarchStatusReturning || totalTroops(march.AttackTroops) != 40 {
		t.Fatalf("expected empty defender to create survivor return, got %+v", march)
	}
}

func TestPvpMarchDurationUsesSlowestUnitSpeed(t *testing.T) {
	svc, repo, attacker, defender := newPvpTestService(t)
	unitsMu.Lock()
	activeUnits["wei"]["weiInfantry"] = UnitConfig{Name: "魏步兵", Category: "infantry", Stats: map[string]int{"attack": 10, "infantryDefense": 10, "cavalryDefense": 8, "speed": 6, "carryCapacity": 5, "upkeep": 1}}
	activeUnits["wei"]["weiCavalry"] = UnitConfig{Name: "魏骑兵", Category: "cavalry", Stats: map[string]int{"attack": 14, "infantryDefense": 8, "cavalryDefense": 10, "speed": 30, "carryCapacity": 6, "upkeep": 2}}
	unitsMu.Unlock()
	attacker.Army = []ArmyUnit{{UnitType: "weiInfantry", Amount: 100}, {UnitType: "weiCavalry", Amount: 100}}
	for i := range attacker.Buildings {
		if attacker.Buildings[i].Type == "relay_station" {
			attacker.Buildings[i].Level = 20
		}
	}
	defender.Army = nil
	repo.players[attacker.Player.ID] = attacker
	repo.players[defender.Player.ID] = defender
	if _, err := repo.AssignWorldPosition(attacker.Player.ID, defaultWorldID, 10, 10, "test"); err != nil {
		t.Fatalf("AssignWorldPosition attacker failed: %v", err)
	}
	if _, err := repo.AssignWorldPosition(defender.Player.ID, defaultWorldID, 46, 10, "test"); err != nil {
		t.Fatalf("AssignWorldPosition defender failed: %v", err)
	}

	fast, err := svc.StartPvpAttack(PvpAttackRequest{
		PlayerID:       attacker.Player.ID,
		TargetPlayerID: defender.Player.ID,
		Troops:         map[string]int{"weiCavalry": 30},
	})
	if err != nil {
		t.Fatalf("StartPvpAttack fast failed: %v", err)
	}
	if fast.March.DurationSeconds != 300 || math.Abs(fast.March.SpeedMultiplier-36) > 1e-9 {
		t.Fatalf("expected speed 30 with relay station level 20 to take 300 seconds, got %+v", fast.March)
	}
	forcePvpMarchDue(t, repo, fast.March.ID)
	if _, err := svc.ResolvePvpMarch(fast.March.ID); err != nil {
		t.Fatalf("ResolvePvpMarch fast failed: %v", err)
	}
	fastReturn, err := repo.GetPvpMarch(fast.March.ID)
	if err != nil {
		t.Fatalf("GetPvpMarch fast failed: %v", err)
	}
	returnStartedAt, _ := time.Parse(resourceDateLayout, fastReturn.ReturnStartedAt)
	returnsAt, _ := time.Parse(resourceDateLayout, fastReturn.ReturnsAt)
	if int(returnsAt.Sub(returnStartedAt).Seconds()) != 300 {
		t.Fatalf("expected speed 30 return with relay station level 20 to take 300 seconds, got %+v", fastReturn)
	}
	repo.mu.Lock()
	pvpState := repo.pvpPlayerStates[attacker.Player.ID]
	pvpState.CooldownUntil = ""
	repo.pvpPlayerStates[attacker.Player.ID] = pvpState
	defenderPvpState := repo.pvpPlayerStates[defender.Player.ID]
	defenderPvpState.ProtectionType = ""
	defenderPvpState.ProtectedUntil = ""
	repo.pvpPlayerStates[defender.Player.ID] = defenderPvpState
	repo.mu.Unlock()

	mixed, err := svc.StartPvpAttack(PvpAttackRequest{
		PlayerID:       attacker.Player.ID,
		TargetPlayerID: defender.Player.ID,
		Troops:         map[string]int{"weiInfantry": 10, "weiCavalry": 10},
	})
	if err != nil {
		t.Fatalf("StartPvpAttack mixed failed: %v", err)
	}
	if mixed.March.DurationSeconds != 1500 || math.Abs(mixed.March.SpeedMultiplier-7.2) > 1e-9 {
		t.Fatalf("expected mixed march to use slowest speed 6 plus relay station level 20, got %+v", mixed.March)
	}
}

func TestPvpModifierSourcesOnlyUseCarriedGenerals(t *testing.T) {
	setTestFactionsAndGenerals(t, FactionsConfig{
		"wei": {Generals: []GeneralInfo{{ID: "caocao", Name: "曹操"}}},
	}, GeneralsConfig{
		Enabled: true,
		Heroes: map[string]GeneralHeroConfig{
			"caocao": {ID: "caocao", Name: "曹操", Faction: "wei", Enabled: true, Buffs: map[string]float64{StatAttackBonus: 0.5}},
		},
	})
	_, _, attacker, _ := newPvpTestService(t)
	now := time.Now()
	EnsureGeneralRoster(&attacker, now)
	if attacker.General == nil {
		t.Fatal("expected attacker general")
	}
	attacker.Buildings = nil
	attacker.Buffs = nil
	syncActiveGeneralToRoster(&attacker)

	withoutGeneral := ComputeIntAttributeAt(100, StatAttackBonus, now, pvpModifierSourcesForGenerals(&attacker, nil)...)
	withGeneral := ComputeIntAttributeAt(100, StatAttackBonus, now, pvpModifierSourcesForGenerals(&attacker, []string{attacker.General.ID})...)
	if withoutGeneral != 100 {
		t.Fatalf("expected pvp without carried general to ignore home general bonus, got %d", withoutGeneral)
	}
	if withGeneral <= withoutGeneral {
		t.Fatalf("expected carried general to increase pvp attack, without=%d with=%d", withoutGeneral, withGeneral)
	}
}

func TestPvpMarchListFailsInvalidEmptyAttackTroopsWithoutBlocking(t *testing.T) {
	svc, repo, attacker, defender := newPvpTestService(t)
	now := time.Now().UTC()
	repo.pvpMarches["pvp_march_empty_attack"] = PvpMarch{
		ID:               "pvp_march_empty_attack",
		AttackerPlayerID: attacker.Player.ID,
		AttackerName:     attacker.Player.Nickname,
		AttackerFaction:  attacker.Player.Faction,
		DefenderPlayerID: defender.Player.ID,
		DefenderName:     defender.Player.Nickname,
		DefenderFaction:  defender.Player.Faction,
		MarchType:        PvpMarchTypeAttack,
		Status:           PvpMarchStatusMarching,
		AttackTroops:     map[string]int{},
		AttackGenerals:   []string{"caocao"},
		StartedAt:        now.Add(-2 * time.Hour).Format(resourceDateLayout),
		ArrivesAt:        now.Add(-time.Second).Format(resourceDateLayout),
		CreatedAt:        now.Add(-2 * time.Hour).Format(resourceDateLayout),
		UpdatedAt:        now.Add(-2 * time.Hour).Format(resourceDateLayout),
	}
	repo.players[attacker.Player.ID] = attacker
	if _, err := svc.ListPvpMarches(attacker.Player.ID); err != nil {
		t.Fatalf("ListPvpMarches should not be blocked by invalid empty troops: %v", err)
	}
	march, err := repo.GetPvpMarch("pvp_march_empty_attack")
	if err != nil {
		t.Fatalf("GetPvpMarch failed: %v", err)
	}
	if march.Status != PvpMarchStatusFailed {
		t.Fatalf("expected invalid empty attack march failed, got %+v", march)
	}
}

// TestPvpMarchListRedactsIncomingEnemyForce 验证防守方来袭队列不会提前泄露敌军兵力和武将。
func TestPvpMarchListRedactsIncomingEnemyForce(t *testing.T) {
	svc, repo, attacker, defender := newPvpTestService(t)
	now := time.Now().UTC()
	repo.pvpMarches["pvp_incoming_safe"] = PvpMarch{
		ID:               "pvp_incoming_safe",
		AttackerPlayerID: attacker.Player.ID,
		AttackerName:     attacker.Player.Nickname,
		AttackerFaction:  attacker.Player.Faction,
		DefenderPlayerID: defender.Player.ID,
		DefenderName:     defender.Player.Nickname,
		DefenderFaction:  defender.Player.Faction,
		MarchType:        PvpMarchTypeAttack,
		Status:           PvpMarchStatusMarching,
		AttackTroops:     map[string]int{"weiInfantry": 88},
		AttackGenerals:   []string{"caocao"},
		SpeedMultiplier:  2,
		DurationSeconds:  600,
		StartedAt:        now.Format(resourceDateLayout),
		ArrivesAt:        now.Add(10 * time.Minute).Format(resourceDateLayout),
		CreatedAt:        now.Format(resourceDateLayout),
		UpdatedAt:        now.Format(resourceDateLayout),
	}

	defenderView, err := svc.ListPvpMarches(defender.Player.ID)
	if err != nil || len(defenderView.Items) != 1 {
		t.Fatalf("expected one incoming march, err=%v items=%+v", err, defenderView.Items)
	}
	if len(defenderView.Items[0].AttackTroops) != 0 || len(defenderView.Items[0].AttackGenerals) != 0 {
		t.Fatalf("incoming march leaked enemy force: %+v", defenderView.Items[0])
	}
	if defenderView.Items[0].ViewerRole != PvpMarchViewerRoleIncoming || defenderView.Items[0].SpeedMultiplier != 0 || defenderView.Items[0].DurationSeconds != 0 {
		t.Fatalf("expected a redacted incoming projection, got %+v", defenderView.Items[0])
	}
	if defenderView.Items[0].StartedAt != "" || defenderView.Items[0].CreatedAt != "" || defenderView.Items[0].UpdatedAt != "" {
		t.Fatalf("incoming march leaked timestamps that reveal travel speed: %+v", defenderView.Items[0])
	}
	attackerView, err := svc.ListPvpMarches(attacker.Player.ID)
	if err != nil || len(attackerView.Items) != 1 {
		t.Fatalf("expected one sent march, err=%v items=%+v", err, attackerView.Items)
	}
	if attackerView.Items[0].ViewerRole != PvpMarchViewerRoleSent || attackerView.Items[0].AttackTroops["weiInfantry"] != 88 || len(attackerView.Items[0].AttackGenerals) != 1 {
		t.Fatalf("expected attacker to keep full sent march, got %+v", attackerView.Items[0])
	}
	if attackerView.Items[0].StartedAt == "" || attackerView.Items[0].CreatedAt == "" || attackerView.Items[0].UpdatedAt == "" {
		t.Fatalf("expected attacker to keep full march timestamps, got %+v", attackerView.Items[0])
	}
	adminView, err := svc.AdminPvpMarches(attacker.Player.ID, 10)
	if err != nil || len(adminView.Items) != 1 || adminView.Items[0].AttackTroops["weiInfantry"] != 88 || adminView.Items[0].StartedAt == "" {
		t.Fatalf("expected admin view to keep raw march details, err=%v items=%+v", err, adminView.Items)
	}
}

func TestAdminCancelPvpMarchReturnsTroops(t *testing.T) {
	svc, repo, attacker, defender := newPvpTestService(t)
	attacker.Army = []ArmyUnit{{UnitType: "weiInfantry", Amount: 100}}
	repo.players[attacker.Player.ID] = attacker
	repo.players[defender.Player.ID] = defender
	started, err := svc.StartPvpAttack(PvpAttackRequest{
		PlayerID:       attacker.Player.ID,
		TargetPlayerID: defender.Player.ID,
		Troops:         map[string]int{"weiInfantry": 40},
		GeneralIDs:     []string{"caocao"},
	})
	if err != nil {
		t.Fatalf("StartPvpAttack failed: %v", err)
	}
	cancelled, err := svc.AdminCancelPvpMarch(started.March.ID)
	if err != nil {
		t.Fatalf("AdminCancelPvpMarch failed: %v", err)
	}
	if cancelled.March.Status != PvpMarchStatusCancelled {
		t.Fatalf("expected cancelled march, got %+v", cancelled.March)
	}
	if armySliceToMap(cancelled.Army)["weiInfantry"] != 100 {
		t.Fatalf("expected cancelled troops returned, got %+v", cancelled.Army)
	}
	if len(cancelled.Generals) == 0 {
		t.Fatalf("expected generals returned in patch, got %+v", cancelled.Generals)
	}
	if _, err := svc.AdminCancelPvpMarch(started.March.ID); !errors.Is(err, ErrPvpMarchNotRecallable) {
		t.Fatalf("expected repeated cancel rejected, got %v", err)
	}
}

func TestPvpMarchAccelerateDeductsCityGoldAndShortensArrival(t *testing.T) {
	svc, repo, attacker, defender := newPvpTestService(t)
	attacker.Army = []ArmyUnit{{UnitType: "weiInfantry", Amount: 100}}
	attacker.CityGold = 100
	repo.players[attacker.Player.ID] = attacker
	repo.players[defender.Player.ID] = defender

	started, err := svc.StartPvpAttack(PvpAttackRequest{
		PlayerID:       attacker.Player.ID,
		TargetPlayerID: defender.Player.ID,
		MarchMode:      PvpMarchTypeAttack,
		Troops:         map[string]int{"weiInfantry": 40},
	})
	if err != nil {
		t.Fatalf("StartPvpAttack failed: %v", err)
	}
	beforeArrivesAt, err := time.Parse(resourceDateLayout, started.March.ArrivesAt)
	if err != nil {
		t.Fatalf("parse before arrivesAt failed: %v", err)
	}

	accelerated, err := svc.AcceleratePvpMarch(attacker.Player.ID, started.March.ID)
	if err != nil {
		t.Fatalf("AcceleratePvpMarch failed: %v", err)
	}
	afterArrivesAt, err := time.Parse(resourceDateLayout, accelerated.March.ArrivesAt)
	if err != nil {
		t.Fatalf("parse after arrivesAt failed: %v", err)
	}
	if !afterArrivesAt.Before(beforeArrivesAt) {
		t.Fatalf("expected arrival shortened, before=%s after=%s", beforeArrivesAt, afterArrivesAt)
	}
	if accelerated.Cost != pvpAccelerateFixedCityGoldCost || int(accelerated.CityGold) != 100-accelerated.Cost {
		t.Fatalf("unexpected cost/cityGold: cost=%d cityGold=%d", accelerated.Cost, accelerated.CityGold)
	}
	if accelerated.March.AcceleratedTimes != 1 || accelerated.March.SpeedMultiplier <= 1 {
		t.Fatalf("expected acceleration metadata updated, march=%+v", accelerated.March)
	}
	second, err := svc.AcceleratePvpMarch(attacker.Player.ID, started.March.ID)
	if err != nil {
		t.Fatalf("second AcceleratePvpMarch failed: %v", err)
	}
	if second.March.AcceleratedTimes != 2 {
		t.Fatalf("expected second acceleration count, got %+v", second.March)
	}
	if _, err := svc.AcceleratePvpMarch(attacker.Player.ID, started.March.ID); !errors.Is(err, ErrPvpMarchNotAccelerable) {
		t.Fatalf("expected third acceleration rejected, got %v", err)
	}
}

func TestPvpBattleReturnUsesActualAcceleratedOutboundDuration(t *testing.T) {
	svc, repo, attacker, defender := newPvpTestService(t)
	attacker.Army = []ArmyUnit{{UnitType: "weiInfantry", Amount: 100}}
	attacker.CityGold = 100
	defender.Army = nil
	repo.players[attacker.Player.ID] = attacker
	repo.players[defender.Player.ID] = defender

	started, err := svc.StartPvpAttack(PvpAttackRequest{
		PlayerID:       attacker.Player.ID,
		TargetPlayerID: defender.Player.ID,
		MarchMode:      PvpMarchTypeAttack,
		Troops:         map[string]int{"weiInfantry": 40},
	})
	if err != nil {
		t.Fatalf("StartPvpAttack failed: %v", err)
	}
	first, err := svc.AcceleratePvpMarch(attacker.Player.ID, started.March.ID)
	if err != nil {
		t.Fatalf("first AcceleratePvpMarch failed: %v", err)
	}
	second, err := svc.AcceleratePvpMarch(attacker.Player.ID, started.March.ID)
	if err != nil {
		t.Fatalf("second AcceleratePvpMarch failed: %v", err)
	}
	startedAt, err := time.Parse(resourceDateLayout, second.March.StartedAt)
	if err != nil {
		t.Fatalf("parse startedAt failed: %v", err)
	}
	arrivesAt, err := time.Parse(resourceDateLayout, second.March.ArrivesAt)
	if err != nil {
		t.Fatalf("parse arrivesAt failed: %v", err)
	}
	actualOutboundSeconds := int(math.Ceil(arrivesAt.Sub(startedAt).Seconds()))
	if actualOutboundSeconds <= 0 || actualOutboundSeconds >= started.March.DurationSeconds {
		t.Fatalf("expected acceleration to shorten outbound duration, first=%+v second=%+v actual=%d original=%d", first.March, second.March, actualOutboundSeconds, started.March.DurationSeconds)
	}
	forcePvpMarchDueWithOutboundDuration(t, repo, started.March.ID, time.Duration(actualOutboundSeconds)*time.Second)

	if _, err := svc.ResolvePvpMarch(started.March.ID); err != nil {
		t.Fatalf("ResolvePvpMarch failed: %v", err)
	}
	march, err := repo.GetPvpMarch(started.March.ID)
	if err != nil {
		t.Fatalf("GetPvpMarch failed: %v", err)
	}
	returnStartedAt, err := time.Parse(resourceDateLayout, march.ReturnStartedAt)
	if err != nil {
		t.Fatalf("parse returnStartedAt failed: %v", err)
	}
	returnsAt, err := time.Parse(resourceDateLayout, march.ReturnsAt)
	if err != nil {
		t.Fatalf("parse returnsAt failed: %v", err)
	}
	if got := int(returnsAt.Sub(returnStartedAt).Seconds()); got != actualOutboundSeconds {
		t.Fatalf("expected return duration to equal actual accelerated outbound %d, got %d march=%+v", actualOutboundSeconds, got, march)
	}
}

func TestPvpAttackRespectsDailyLimitAndProtection(t *testing.T) {
	svc, repo, attacker, defender := newPvpTestService(t)
	now := time.Now().UTC()
	attacker.Army = []ArmyUnit{{UnitType: "weiInfantry", Amount: 100}}
	repo.players[attacker.Player.ID] = attacker
	repo.players[defender.Player.ID] = defender

	state := newDefaultPvpPlayerState(attacker.Player.ID, now)
	state.DailyAttackCount = state.DailyAttackLimit
	if err := repo.SavePvpPlayerState(state, now); err != nil {
		t.Fatalf("SavePvpPlayerState daily limit failed: %v", err)
	}
	if _, err := svc.StartPvpAttack(PvpAttackRequest{
		PlayerID:       attacker.Player.ID,
		TargetPlayerID: defender.Player.ID,
		Troops:         map[string]int{"weiInfantry": 1},
	}); !errors.Is(err, ErrPvpDailyLimitReached) {
		t.Fatalf("expected daily limit error, got %v", err)
	}

	state.DailyAttackCount = 0
	if err := repo.SavePvpPlayerState(state, now); err != nil {
		t.Fatalf("SavePvpPlayerState clear daily failed: %v", err)
	}
	defenderPvp := newDefaultPvpPlayerState(defender.Player.ID, now)
	defenderPvp.ProtectedUntil = now.Add(time.Minute).Format(resourceDateLayout)
	defenderPvp.ProtectionType = "defeat"
	defenderPvp.Status = "protected"
	if err := repo.SavePvpPlayerState(defenderPvp, now); err != nil {
		t.Fatalf("SavePvpPlayerState defender protection failed: %v", err)
	}
	if _, err := svc.StartPvpAttack(PvpAttackRequest{
		PlayerID:       attacker.Player.ID,
		TargetPlayerID: defender.Player.ID,
		Troops:         map[string]int{"weiInfantry": 1},
	}); !errors.Is(err, ErrPvpTargetProtected) {
		t.Fatalf("expected target protected error, got %v", err)
	}
}

func TestPvpManualProtectionItemBlocksAttackAndBreaksOnAttack(t *testing.T) {
	loadPvpProtectionTestItemsConfig(t)
	svc, repo, attacker, defender := newPvpTestService(t)
	attacker.Army = []ArmyUnit{{UnitType: "weiInfantry", Amount: 100}}
	defender.Army = []ArmyUnit{{UnitType: "greedyWolf", Amount: 100}}
	repo.players[attacker.Player.ID] = attacker
	repo.players[defender.Player.ID] = defender
	if _, err := svc.GrantItem(defender.Player.ID, "test_pvp_truce", 1); err != nil {
		t.Fatalf("GrantItem truce failed: %v", err)
	}
	if _, err := svc.UseItem(defender.Player.ID, "test_pvp_truce", 1); err != nil {
		t.Fatalf("UseItem truce failed: %v", err)
	}
	defenderPvp, err := svc.GetPvpState(defender.Player.ID)
	if err != nil {
		t.Fatalf("GetPvpState defender failed: %v", err)
	}
	if defenderPvp.State.ProtectionType != PvpProtectionTypeManual || defenderPvp.State.ProtectedUntil == "" {
		t.Fatalf("expected manual protection from item, got %+v", defenderPvp.State)
	}
	if _, err := svc.StartPvpAttack(PvpAttackRequest{
		PlayerID:       attacker.Player.ID,
		TargetPlayerID: defender.Player.ID,
		Troops:         map[string]int{"weiInfantry": 1},
	}); !errors.Is(err, ErrPvpTargetProtected) {
		t.Fatalf("expected truce protected target error, got %v", err)
	}

	// 主动攻击会打破自己的手动免战。
	if _, err := svc.SetPvpProtection(attacker.Player.ID, PvpProtectionTypeManual, time.Hour, "test", time.Now().UTC()); err != nil {
		t.Fatalf("SetPvpProtection attacker failed: %v", err)
	}
	third := newPlayerState("player_pvp_third", "第三方", "wu", "sunquan", time.Now())
	third.Army = []ArmyUnit{{UnitType: "shadowGuard", Amount: 1}}
	if err := repo.CreatePlayer("account_pvp_b", third, time.Now()); err != nil {
		t.Fatalf("CreatePlayer third failed: %v", err)
	}
	if _, err := svc.StartPvpAttack(PvpAttackRequest{
		PlayerID:       attacker.Player.ID,
		TargetPlayerID: third.Player.ID,
		Troops:         map[string]int{"weiInfantry": 1},
	}); err != nil {
		t.Fatalf("StartPvpAttack should break manual protection and pass, got %v", err)
	}
	attackerPvp, err := svc.GetPvpState(attacker.Player.ID)
	if err != nil {
		t.Fatalf("GetPvpState attacker failed: %v", err)
	}
	if attackerPvp.State.ProtectionType != "" || attackerPvp.State.ProtectedUntil != "" {
		t.Fatalf("expected manual protection broken by attack, got %+v", attackerPvp.State)
	}
}

func TestPvpMaintenanceProtectionHasPriority(t *testing.T) {
	svc, _, _, defender := newPvpTestService(t)
	now := time.Now().UTC()
	maintenance, err := svc.SetPvpProtection(defender.Player.ID, PvpProtectionTypeMaintenance, 2*time.Hour, "maintenance", now)
	if err != nil {
		t.Fatalf("SetPvpProtection maintenance failed: %v", err)
	}
	if maintenance.State.ProtectionType != PvpProtectionTypeMaintenance {
		t.Fatalf("expected maintenance protection, got %+v", maintenance.State)
	}
	manual, err := svc.SetPvpProtection(defender.Player.ID, PvpProtectionTypeManual, time.Hour, "manual", now)
	if err != nil {
		t.Fatalf("SetPvpProtection manual failed: %v", err)
	}
	if manual.State.ProtectionType != PvpProtectionTypeMaintenance {
		t.Fatalf("manual protection should not override maintenance, got %+v", manual.State)
	}
}

func TestPvpCityWallAppliesFactionDefense(t *testing.T) {
	setTestCombatUnitsConfig(t)
	originalCombat := combat.GetCombatConfig()
	t.Cleanup(func() {
		if err := combat.SaveCombatConfig("", originalCombat); err != nil {
			t.Fatalf("restore combat config: %v", err)
		}
	})

	cfg := combat.GetCombatConfig()
	cfg.WallConfig["shu"] = combat.WallEntry{Base: 1.02, Hardness: 1.35, MinDamagedLevelFrom20: 16, MaxDamagedLevelFrom20: 17}
	if err := combat.SaveCombatConfig("", cfg); err != nil {
		t.Fatalf("set combat config: %v", err)
	}

	now := time.Now()
	attacker := newPlayerState("player_pvp_wall_attacker", "攻方", "wei", "", now)
	defender := newPlayerState("player_pvp_wall_defender", "守方", "shu", "", now)
	attacker.Buildings = nil
	defender.Buildings = []Building{{ID: "city_wall-1", Type: "city_wall", Level: 20}}
	defender.Army = []ArmyUnit{{UnitType: "weiInfantry", Amount: 10}}
	march := &PvpMarch{
		ID:             "pvp_wall_march",
		MarchType:      "attack",
		AttackTroops:   map[string]int{"weiInfantry": 10},
		AttackGenerals: nil,
	}

	battle, attackerReport, defenderReport, _, _, err := resolvePvpCombat(&attacker, &defender, nil, march, now)
	if err != nil {
		t.Fatalf("resolvePvpCombat failed: %v", err)
	}

	factionBonus := math.Pow(1.02, 20) - 1
	if attackerReport.PvpWall == nil || defenderReport.PvpWall == nil {
		t.Fatalf("expected pvp wall snapshot, attacker=%+v defender=%+v", attackerReport.PvpWall, defenderReport.PvpWall)
	}
	if math.Abs(attackerReport.PvpWall.FactionDefenseBonus-factionBonus) > 1e-6 {
		t.Fatalf("expected wall faction bonus %.4f, got %+v", factionBonus, attackerReport.PvpWall)
	}
	if math.Abs(attackerReport.PvpWall.TotalDefenseBonus-factionBonus) > 1e-6 {
		t.Fatalf("expected wall total bonus %.4f, got %+v", factionBonus, attackerReport.PvpWall)
	}
	if attackerReport.PvpWall.Hardness != 1.35 || attackerReport.PvpWall.MinDamagedLevelFrom20 != 16 || attackerReport.PvpWall.MaxDamagedLevelFrom20 != 17 {
		t.Fatalf("expected wall hardness snapshot, got %+v", attackerReport.PvpWall)
	}

	defensePower, ok := battle.Result["defensePower"].(float64)
	if !ok {
		t.Fatalf("expected numeric defense power, got %+v", battle.Result["defensePower"])
	}
	expectedDefensePower := 100 * math.Pow(1.02, 20)
	if math.Abs(defensePower-expectedDefensePower) > 1e-6 {
		t.Fatalf("expected defense power %.4f, got %.4f", expectedDefensePower, defensePower)
	}
}

// newPvpTestService 创建默认的魏攻蜀守 PVP 测试环境。
func newPvpTestService(t *testing.T) (*Service, *MemoryRepository, GameState, GameState) {
	return newPvpTestServiceForGenerals(t, "wei", "caocao", "shu", "liubei")
}

// newPvpTestServiceForGenerals 按指定阵营和主将创建可复用的 PVP 测试环境。
func newPvpTestServiceForGenerals(t *testing.T, attackerFaction string, attackerGeneralID string, defenderFaction string, defenderGeneralID string) (*Service, *MemoryRepository, GameState, GameState) {
	t.Helper()
	setTestCombatUnitsConfig(t)
	unitsMu.Lock()
	for _, faction := range []string{attackerFaction, defenderFaction} {
		if faction == "" {
			continue
		}
		if activeUnits[faction] == nil {
			activeUnits[faction] = FactionUnits{}
		}
		unitType := faction + "Infantry"
		if _, exists := activeUnits[faction][unitType]; !exists {
			activeUnits[faction][unitType] = UnitConfig{
				Name: faction + "测试步兵", Category: "infantry",
				Stats: map[string]int{"attack": 10, "infantryDefense": 10, "cavalryDefense": 8, "carryCapacity": 5, "upkeep": 1},
			}
		}
	}
	unitsMu.Unlock()
	now := time.Now()
	repo := NewMemoryRepository()
	accountA := Account{ID: "account_pvp_a", Username: "pvp_a", PasswordHash: "hash", CreatedAt: now}
	accountB := Account{ID: "account_pvp_b", Username: "pvp_b", PasswordHash: "hash", CreatedAt: now}
	if err := repo.CreateAccount(accountA); err != nil {
		t.Fatalf("CreateAccount A failed: %v", err)
	}
	if err := repo.CreateAccount(accountB); err != nil {
		t.Fatalf("CreateAccount B failed: %v", err)
	}
	attacker := newPlayerState("player_pvp_attacker", "攻击方", attackerFaction, attackerGeneralID, now)
	defender := newPlayerState("player_pvp_defender", "防守方", defenderFaction, defenderGeneralID, now)
	if err := repo.CreatePlayer(accountA.ID, attacker, now); err != nil {
		t.Fatalf("CreatePlayer attacker failed: %v", err)
	}
	if err := repo.CreatePlayer(accountB.ID, defender, now); err != nil {
		t.Fatalf("CreatePlayer defender failed: %v", err)
	}
	return NewServiceWithRepository(repo), repo, attacker, defender
}

func addPvpScoutTestUnits(t *testing.T) {
	t.Helper()
	unitsMu.Lock()
	defer unitsMu.Unlock()
	if activeUnits == nil {
		activeUnits = UnitsConfig{}
	}
	if activeUnits["wei"] == nil {
		activeUnits["wei"] = FactionUnits{}
	}
	activeUnits["wei"]["weiScout"] = UnitConfig{
		Name:     "魏侦察兵",
		Category: "cavalry",
		Role:     "scout",
		Stats:    map[string]int{"attack": 1, "infantryDefense": 1, "cavalryDefense": 1, "speed": 30, "carryCapacity": 1, "upkeep": 1},
	}
	if activeUnits["shu"] == nil {
		activeUnits["shu"] = FactionUnits{}
	}
	activeUnits["shu"]["shuScout"] = UnitConfig{
		Name:     "蜀侦察兵",
		Category: "infantry",
		Role:     "scout",
		Stats:    map[string]int{"attack": 1, "infantryDefense": 1, "cavalryDefense": 1, "speed": 30, "carryCapacity": 1, "upkeep": 1},
	}
	activeUnits["shu"]["shuInfantry"] = UnitConfig{
		Name:     "蜀步兵",
		Category: "infantry",
		Stats:    map[string]int{"attack": 10, "infantryDefense": 10, "cavalryDefense": 8, "carryCapacity": 5, "upkeep": 1},
	}
}

// pvpTestLossesFromBattle 从 PVP 战斗记录里取出某一侧损失。
func pvpTestLossesFromBattle(t *testing.T, battle PvpBattle, side string) map[string]int {
	t.Helper()
	losses, ok := battle.Losses[side].(map[string]int)
	if !ok {
		t.Fatalf("expected %s losses map, got %+v", side, battle.Losses[side])
	}
	return losses
}

// pvpTestUnitLosses 把 map 损失转换为核心战斗损失结构。
func pvpTestUnitLosses(losses map[string]int) []combat.UnitLoss {
	result := make([]combat.UnitLoss, 0, len(losses))
	for unitType, amount := range losses {
		result = append(result, combat.UnitLoss{ID: unitType, Losses: amount})
	}
	return result
}

// pvpTestGeneralExp 读取测试玩家指定武将经验。
func pvpTestGeneralExp(state GameState, generalID string) int {
	for _, general := range state.Generals {
		if general.ID == generalID {
			return general.Exp
		}
	}
	return 0
}

// pvpTestGeneralLevel 读取测试玩家指定武将等级。
func pvpTestGeneralLevel(state GameState, generalID string) int {
	for _, owned := range state.Generals {
		if owned.ID == generalID {
			return owned.Level
		}
	}
	if state.General != nil && state.General.ID == generalID {
		return state.General.Level
	}
	return 0
}

// setPvpTestGeneralProgress 同步测试状态中的名册武将和当前主将进度。
func setPvpTestGeneralProgress(state *GameState, generalID string, level int, exp int) {
	if state == nil {
		return
	}
	for index := range state.Generals {
		if state.Generals[index].ID == generalID {
			state.Generals[index].Level = level
			state.Generals[index].Exp = exp
		}
	}
	if state.General != nil && state.General.ID == generalID {
		state.General.Level = level
		state.General.Exp = exp
	}
}

func loadPvpProtectionTestItemsConfig(t *testing.T) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "items.json")
	data := []byte(`{
		"test_pvp_truce": {
			"id": "test_pvp_truce",
			"name": "测试免战令",
			"description": "测试用",
			"type": "consumable",
			"rarity": "rare",
			"usable": true,
			"stackable": true,
			"maxStack": 999999,
			"useTarget": "player",
			"effects": [
				{ "type": "pvp_protection", "protectionType": "manual", "durationSeconds": 3600 }
			]
		}
	}`)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("write pvp item config: %v", err)
	}
	if err := LoadItemsConfig(path); err != nil {
		t.Fatalf("load pvp item config: %v", err)
	}
	t.Cleanup(func() {
		_ = LoadItemsConfig(filepath.Join("..", "..", "..", "config", "items.json"))
	})
}

func forcePvpMarchDue(t *testing.T, repo *MemoryRepository, marchID string) {
	t.Helper()
	repo.mu.Lock()
	defer repo.mu.Unlock()
	march := repo.pvpMarches[marchID]
	march.ArrivesAt = time.Now().Add(-time.Second).UTC().Format(resourceDateLayout)
	repo.pvpMarches[marchID] = march
}

func forcePvpMarchDueWithOutboundDuration(t *testing.T, repo *MemoryRepository, marchID string, duration time.Duration) {
	t.Helper()
	if duration <= 0 {
		duration = time.Second
	}
	repo.mu.Lock()
	defer repo.mu.Unlock()
	arrivesAt := time.Now().Add(-time.Second).UTC()
	march := repo.pvpMarches[marchID]
	march.StartedAt = arrivesAt.Add(-duration).Format(resourceDateLayout)
	march.ArrivesAt = arrivesAt.Format(resourceDateLayout)
	repo.pvpMarches[marchID] = march
}

func agePvpMarch(t *testing.T, repo *MemoryRepository, marchID string, age time.Duration) {
	t.Helper()
	repo.mu.Lock()
	defer repo.mu.Unlock()
	march := repo.pvpMarches[marchID]
	march.StartedAt = time.Now().Add(-age).UTC().Format(resourceDateLayout)
	repo.pvpMarches[marchID] = march
}

func forcePvpReturnDue(t *testing.T, repo *MemoryRepository, marchID string) {
	t.Helper()
	repo.mu.Lock()
	defer repo.mu.Unlock()
	march := repo.pvpMarches[marchID]
	march.ReturnsAt = time.Now().Add(-time.Second).UTC().Format(resourceDateLayout)
	repo.pvpMarches[marchID] = march
}
