// 本文件测试 PVP 行军和战斗结算主链路。
package game

import (
	"errors"
	"hero3/internal/core/combat"
	"math"
	"os"
	"path/filepath"
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
	reports := buildPvpReinforcementReports("event_zero_loss", &attacker, &defender, changed, map[string]map[string]int{}, map[string]int{record.ID: 12}, "defender_victory", now.Format(resourceDateLayout))
	if len(reports) != 1 || reports[0].GeneralExpGained != 12 {
		t.Fatalf("expected zero-loss reinforcement report with exp, got %+v", reports)
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
		ReinforcementSnapshot: []DefenseReinforcementUnit{{ReinforcementID: "rein_hidden", Troops: map[string]int{"shuInfantry": 20}, Generals: []ReinforcementGeneralSnapshot{{ID: "guanyu"}}}},
	}, report.PlayerID)
	if battle.DefenderSnapshot != nil || len(battle.ReinforcementSnapshot[0].Troops) != 0 || len(battle.ReinforcementSnapshot[0].Generals) != 0 {
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

func TestPvpAttackRejectsMultipleGenerals(t *testing.T) {
	svc, _, attacker, defender := newPvpTestService(t)
	attacker.Army = []ArmyUnit{{UnitType: "weiInfantry", Amount: 100}}
	attacker.Generals = append(attacker.Generals, *newGeneral("wei", "xiahoudun"))
	repo := svc.repo.(*MemoryRepository)
	repo.players[attacker.Player.ID] = attacker

	if _, err := svc.StartPvpAttack(PvpAttackRequest{
		PlayerID:       attacker.Player.ID,
		TargetPlayerID: defender.Player.ID,
		MarchMode:      PvpMarchTypeAttack,
		Troops:         map[string]int{"weiInfantry": 10},
		GeneralIDs:     []string{"caocao", "xiahoudun"},
	}); !errors.Is(err, ErrInvalidGeneral) {
		t.Fatalf("expected multiple generals to be rejected, got %v", err)
	}
}

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
		Losses:            map[string]int{},
		Rules:             defaultGarrisonRules(GarrisonSourceReinforcement),
		SentAt:            now.Add(-4 * time.Hour).UTC().Format(resourceDateLayout),
		ArrivedAt:         now.Add(-3 * time.Hour).UTC().Format(resourceDateLayout),
		CreatedAt:         now.Add(-4 * time.Hour).UTC().Format(resourceDateLayout),
		UpdatedAt:         now.Add(-3 * time.Hour).UTC().Format(resourceDateLayout),
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
	if helperReports[0].EventID != battle.ID || helperReports[0].Detail == nil || helperReports[0].Detail.SecondarySide != nil {
		t.Fatalf("unexpected helper report detail: %+v", helperReports[0])
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

func newPvpTestService(t *testing.T) (*Service, *MemoryRepository, GameState, GameState) {
	t.Helper()
	setTestCombatUnitsConfig(t)
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
	attacker := newPlayerState("player_pvp_attacker", "攻击方", "wei", "caocao", now)
	defender := newPlayerState("player_pvp_defender", "防守方", "shu", "liubei", now)
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
