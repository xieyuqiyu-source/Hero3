// 本文件测试 PVP 行军和战斗结算主链路。
package game

import (
	"errors"
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

func TestPvpTargetsExposeStableWorldPositions(t *testing.T) {
	svc, _, attacker, _ := newPvpTestService(t)

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
	if first.Self.X <= 0 || first.Self.Y <= 0 || first.WorldSize != defaultPvpWorldSize {
		t.Fatalf("unexpected self world position: %+v world=%d", first.Self, first.WorldSize)
	}
	if len(first.Items) == 0 || first.Items[0].Position.X <= 0 || first.Items[0].Distance <= 0 {
		t.Fatalf("expected target positions and distance, got %+v", first.Items)
	}
}

func TestPvpTargetsFilterByMapViewport(t *testing.T) {
	svc, _, attacker, defender := newPvpTestService(t)
	defenderPosition := pvpWorldPositionForPlayer(defender.Player.ID)

	near, err := svc.ListPvpTargetsInArea(attacker.Player.ID, PvpTargetFilter{
		CenterX: defenderPosition.X,
		CenterY: defenderPosition.Y,
		Radius:  1,
		Limit:   10,
	})
	if err != nil {
		t.Fatalf("ListPvpTargetsInArea near failed: %v", err)
	}
	if len(near.Items) != 1 || near.Items[0].PlayerID != defender.Player.ID {
		t.Fatalf("expected defender in near viewport, got %+v", near.Items)
	}

	farX := 1
	if defenderPosition.X < defaultPvpWorldSize/2 {
		farX = defaultPvpWorldSize
	}
	farY := 1
	if defenderPosition.Y < defaultPvpWorldSize/2 {
		farY = defaultPvpWorldSize
	}
	far, err := svc.ListPvpTargetsInArea(attacker.Player.ID, PvpTargetFilter{
		CenterX: farX,
		CenterY: farY,
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
	attacker.Army = []ArmyUnit{{UnitType: "weiScout", Amount: 5}, {UnitType: "weiInfantry", Amount: 20}}
	defender.Army = []ArmyUnit{{UnitType: "shuScout", Amount: 2}, {UnitType: "shuInfantry", Amount: 10}}
	repo.players[attacker.Player.ID] = attacker
	repo.players[defender.Player.ID] = defender

	result, err := svc.ScoutPvpTarget(PvpScoutRequest{PlayerID: attacker.Player.ID, TargetPlayerID: defender.Player.ID})
	if err != nil {
		t.Fatalf("ScoutPvpTarget failed: %v", err)
	}
	if !result.Success || !result.BattleReport.DefenderRevealed {
		t.Fatalf("expected successful scout with revealed defender, got %+v", result)
	}
	if result.BattleReport.DispatchedUnits["weiScout"] != 5 || result.BattleReport.LostUnits["weiScout"] != 2 {
		t.Fatalf("unexpected scout losses: %+v", result.BattleReport)
	}
	if result.BattleReport.DefenderLostUnits["shuScout"] != 2 {
		t.Fatalf("expected defender scouts lost, got %+v", result.BattleReport.DefenderLostUnits)
	}
	updatedAttacker, _ := repo.GetState(attacker.Player.ID)
	updatedDefender, _ := repo.GetState(defender.Player.ID)
	if armySliceToMap(updatedAttacker.Army)["weiScout"] != 3 {
		t.Fatalf("expected 3 scout units remain, got %+v", updatedAttacker.Army)
	}
	if armySliceToMap(updatedDefender.Army)["shuScout"] != 0 {
		t.Fatalf("expected defender scout units removed, got %+v", updatedDefender.Army)
	}
	if result.BattleReport.Detail == nil || !result.BattleReport.Detail.Visibility.ShowEnemyResources {
		t.Fatalf("expected standard scout detail to reveal resources, got %+v", result.BattleReport.Detail)
	}
	scoutExtra, ok := result.BattleReport.Detail.Extra["scout"].(map[string]interface{})
	if !ok || scoutExtra["success"] != true || scoutExtra["scoutUnitType"] != "weiScout" {
		t.Fatalf("expected scout extra snapshot, got %+v", result.BattleReport.Detail.Extra)
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
	if result.Success || result.BattleReport.DefenderRevealed {
		t.Fatalf("expected failed scout, got %+v", result)
	}
	if len(result.BattleReport.DefenderUnits) != 0 || len(result.BattleReport.DefenderResources) != 0 {
		t.Fatalf("failed scout should hide target intel, got units=%+v resources=%+v", result.BattleReport.DefenderUnits, result.BattleReport.DefenderResources)
	}
	if result.BattleReport.Detail == nil || result.BattleReport.Detail.Visibility.ShowEnemyResources {
		t.Fatalf("expected standard detail to hide scout intel, got %+v", result.BattleReport.Detail)
	}
	updatedAttacker, _ := repo.GetState(attacker.Player.ID)
	if armySliceToMap(updatedAttacker.Army)["weiScout"] != 0 {
		t.Fatalf("expected attacker scouts annihilated, got %+v", updatedAttacker.Army)
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
}

func TestPvpAttackRespectsCooldownDailyLimitAndProtection(t *testing.T) {
	svc, repo, attacker, defender := newPvpTestService(t)
	now := time.Now().UTC()
	attacker.Army = []ArmyUnit{{UnitType: "weiInfantry", Amount: 100}}
	repo.players[attacker.Player.ID] = attacker
	repo.players[defender.Player.ID] = defender

	state := newDefaultPvpPlayerState(attacker.Player.ID, now)
	state.CooldownUntil = now.Add(time.Minute).Format(resourceDateLayout)
	if err := repo.SavePvpPlayerState(state, now); err != nil {
		t.Fatalf("SavePvpPlayerState cooldown failed: %v", err)
	}
	if _, err := svc.StartPvpAttack(PvpAttackRequest{
		PlayerID:       attacker.Player.ID,
		TargetPlayerID: defender.Player.ID,
		Troops:         map[string]int{"weiInfantry": 1},
	}); !errors.Is(err, ErrPvpAttackCooldown) {
		t.Fatalf("expected cooldown error, got %v", err)
	}

	state.CooldownUntil = ""
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

func forcePvpReturnDue(t *testing.T, repo *MemoryRepository, marchID string) {
	t.Helper()
	repo.mu.Lock()
	defer repo.mu.Unlock()
	march := repo.pvpMarches[marchID]
	march.ReturnsAt = time.Now().Add(-time.Second).UTC().Format(resourceDateLayout)
	repo.pvpMarches[marchID] = march
}
