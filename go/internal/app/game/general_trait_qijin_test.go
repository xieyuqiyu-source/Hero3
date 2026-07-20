// 本文件验证赵云七进七出真实缩短 PVP 出征与增援时长，并保持行军记录数值一致。
package game

import (
	"math"
	"testing"
	"time"

	"hero3/internal/core/general"
)

// setQijinTestGenerals 配置赵云七进七出，可切换为无特性的对照组。
func setQijinTestGenerals(t *testing.T, enabled bool) {
	t.Helper()
	zhaoyun := GeneralHeroConfig{ID: "zhaoyun", Name: "赵云", Faction: "shu", Enabled: true}
	if enabled {
		zhaoyun.BonusTrait = GeneralTraitConfig{
			TraitID: "qijin_qichu", TraitType: general.TraitTypeBonus, Enabled: true, Scope: "self_army",
			Params: map[string]float64{"speedBonusRate": 1, "minMarchSeconds": 60, "triggerChance": 1},
		}
	}
	setTestFactionsAndGenerals(t, FactionsConfig{
		"shu": {Name: "蜀国", Generals: []GeneralInfo{{ID: "zhaoyun", Name: "赵云"}}},
		"wei": {Name: "魏国", Generals: []GeneralInfo{{ID: "caocao", Name: "曹操"}}},
	}, GeneralsConfig{Enabled: true, Heroes: map[string]GeneralHeroConfig{
		"zhaoyun": zhaoyun,
		"caocao":  {ID: "caocao", Name: "曹操", Faction: "wei", Enabled: true},
	}})
}

// runQijinPvpMarch 创建一条赵云领军的真实 PVP 出征记录。
func runQijinPvpMarch(t *testing.T, enabled bool) (PvpMarch, int) {
	t.Helper()
	setQijinTestGenerals(t, enabled)
	svc, repo, attacker, defender := newPvpTestServiceForGenerals(t, "shu", "zhaoyun", "wei", "caocao")
	attacker.Army = []ArmyUnit{{UnitType: "shuInfantry", Amount: 100}}
	defender.Army = []ArmyUnit{{UnitType: "weiInfantry", Amount: 100}}
	repo.players[attacker.Player.ID] = attacker
	repo.players[defender.Player.ID] = defender
	if _, err := repo.AssignWorldPosition(attacker.Player.ID, defaultWorldID, 10, 10, "test"); err != nil {
		t.Fatalf("AssignWorldPosition attacker failed: %v", err)
	}
	if _, err := repo.AssignWorldPosition(defender.Player.ID, defaultWorldID, 20, 10, "test"); err != nil {
		t.Fatalf("AssignWorldPosition defender failed: %v", err)
	}
	result, err := svc.StartPvpAttack(PvpAttackRequest{
		PlayerID: attacker.Player.ID, TargetPlayerID: defender.Player.ID, MarchMode: PvpMarchTypeAttack,
		Troops: map[string]int{"shuInfantry": 100}, GeneralIDs: []string{"zhaoyun"},
	})
	if err != nil {
		t.Fatalf("StartPvpAttack failed: %v", err)
	}
	reports, total, err := repo.ListReports(attacker.Player.ID, 10, 0)
	if err != nil || total != 0 || len(reports) != 0 {
		t.Fatalf("expected march trait not to create battle report before combat, reports=%+v total=%d err=%v", reports, total, err)
	}
	return result.March, total
}

// runQijinReinforcement 创建一条赵云领军的真实增援记录。
func runQijinReinforcement(t *testing.T, enabled bool) Reinforcement {
	t.Helper()
	setQijinTestGenerals(t, enabled)
	svc, repo, from, to := newPvpTestServiceForGenerals(t, "shu", "zhaoyun", "wei", "caocao")
	from.Army = []ArmyUnit{{UnitType: "shuInfantry", Amount: 100}}
	to.Army = []ArmyUnit{{UnitType: "weiInfantry", Amount: 100}}
	repo.players[from.Player.ID] = from
	repo.players[to.Player.ID] = to
	if _, err := repo.AssignWorldPosition(from.Player.ID, defaultWorldID, 10, 10, "test"); err != nil {
		t.Fatalf("AssignWorldPosition from failed: %v", err)
	}
	if _, err := repo.AssignWorldPosition(to.Player.ID, defaultWorldID, 20, 10, "test"); err != nil {
		t.Fatalf("AssignWorldPosition to failed: %v", err)
	}
	result, err := svc.SendReinforcement(SendReinforcementRequest{
		FromPlayerID: from.Player.ID, TargetPlayerID: to.Player.ID,
		Troops: map[string]int{"shuInfantry": 100}, GeneralIDs: []string{"zhaoyun"},
	})
	if err != nil {
		t.Fatalf("SendReinforcement failed: %v", err)
	}
	reports, total, err := repo.ListReports(from.Player.ID, 10, 0)
	if err != nil || total != 0 || len(reports) != 0 {
		t.Fatalf("expected reinforcement trait not to create battle report before combat, reports=%+v total=%d err=%v", reports, total, err)
	}
	return result.Reinforcement
}

// assertMarchDurationTimestamps 核对开始、到达时间与最终行军秒数一致。
func assertMarchDurationTimestamps(t *testing.T, startedAt string, arrivesAt string, duration int) {
	t.Helper()
	started, err := time.Parse(resourceDateLayout, startedAt)
	if err != nil {
		t.Fatalf("parse startedAt failed: %v", err)
	}
	arrives, err := time.Parse(resourceDateLayout, arrivesAt)
	if err != nil {
		t.Fatalf("parse arrivesAt failed: %v", err)
	}
	if int(arrives.Sub(started).Seconds()) != duration {
		t.Fatalf("expected timestamp delta %d seconds, got %s -> %s", duration, startedAt, arrivesAt)
	}
}

// TestQijinQichuChangesRealPvpAndReinforcementMarches 验证七进七出同时进入真实出征和增援记录，但不伪装成战斗触发。
func TestQijinQichuChangesRealPvpAndReinforcementMarches(t *testing.T) {
	controlPvp, _ := runQijinPvpMarch(t, false)
	activePvp, _ := runQijinPvpMarch(t, true)
	wantPvp := int(math.Ceil(float64(controlPvp.DurationSeconds) / 2))
	if wantPvp < 60 {
		wantPvp = 60
	}
	if activePvp.DurationSeconds != wantPvp || math.Abs(activePvp.SpeedMultiplier-controlPvp.SpeedMultiplier*float64(controlPvp.DurationSeconds)/float64(wantPvp)) > 1e-9 {
		t.Fatalf("expected PVP duration %d and matching speed multiplier, control=%+v active=%+v", wantPvp, controlPvp, activePvp)
	}
	assertMarchDurationTimestamps(t, activePvp.StartedAt, activePvp.ArrivesAt, activePvp.DurationSeconds)

	controlReinforcement := runQijinReinforcement(t, false)
	activeReinforcement := runQijinReinforcement(t, true)
	wantReinforcement := int(math.Ceil(float64(controlReinforcement.MarchSeconds) / 2))
	if wantReinforcement < 60 {
		wantReinforcement = 60
	}
	wantSpeed := controlReinforcement.SpeedMultiplier * float64(controlReinforcement.MarchSeconds) / float64(wantReinforcement)
	if activeReinforcement.MarchSeconds != wantReinforcement || activeReinforcement.ReturnSeconds != wantReinforcement || math.Abs(activeReinforcement.SpeedMultiplier-wantSpeed) > 1e-9 {
		t.Fatalf("expected reinforcement duration %d and speed %.4f, control=%+v active=%+v", wantReinforcement, wantSpeed, controlReinforcement, activeReinforcement)
	}
	assertMarchDurationTimestamps(t, activeReinforcement.SentAt, activeReinforcement.ExpectedArriveAt, activeReinforcement.MarchSeconds)
}
