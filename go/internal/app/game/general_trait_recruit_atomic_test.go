// 本文件验证征兵减耗特性的失败事务回滚，以及军事响应携带完整的非战斗特性结算状态。
package game

import (
	"errors"
	"reflect"
	"testing"
	"time"

	"hero3/internal/core/general"
)

// TestRecruitCostTraitFailureRollsBackPendingSettlement 验证折后资源仍不足时不会提交资源结算或征兵队列。
func TestRecruitCostTraitFailureRollsBackPendingSettlement(t *testing.T) {
	for _, traitCase := range []struct {
		generalID   string
		generalName string
		traitID     string
		rate        float64
	}{
		{generalID: "guojia", generalName: "郭嘉", traitID: "shengui_zhicai", rate: 0.5},
		{generalID: "xunyu", generalName: "荀彧", traitID: "wangzuo_zhicai", rate: 0.05},
	} {
		t.Run(traitCase.generalName+traitCase.traitID, func(t *testing.T) {
			setTestCombatUnitsConfig(t)
			unitsMu.Lock()
			unit := activeUnits["wei"]["weiInfantry"]
			unit.Cost = map[string]int{"wood": 100}
			unit.TrainSeconds = 60
			activeUnits["wei"]["weiInfantry"] = unit
			unitsMu.Unlock()
			setTestFactionsAndGenerals(t, FactionsConfig{
				"wei": {Name: "魏国", Generals: []GeneralInfo{{ID: traitCase.generalID, Name: traitCase.generalName}}},
			}, GeneralsConfig{Enabled: true, Heroes: map[string]GeneralHeroConfig{
				traitCase.generalID: {
					ID: traitCase.generalID, Name: traitCase.generalName, Faction: "wei", Enabled: true,
					SpecialTrait: GeneralTraitConfig{
						TraitID: traitCase.traitID, TraitType: general.TraitTypeSpecial, Enabled: true, Scope: "self_city",
						Params: map[string]float64{"resourceCostReduction": traitCase.rate, "triggerChance": 1},
					},
				},
			}})

			svc := NewService()
			repo := svc.repo.(*MemoryRepository)
			now := time.Now().UTC().Truncate(time.Second)
			account := Account{ID: "account_recruit_atomic_" + traitCase.generalID, Username: "recruit_atomic_" + traitCase.generalID, PasswordHash: "hash", CreatedAt: now}
			if err := repo.CreateAccount(account); err != nil {
				t.Fatalf("CreateAccount failed: %v", err)
			}
			state := newPlayerState("player_recruit_atomic_"+traitCase.generalID, "Recruit Atomic", "wei", traitCase.generalID, now)
			state.Resources.Items["wood"] = 0
			state.Resources.Capacity["wood"] = 100000000
			state.ResourceSettledAt = now.Add(-time.Hour).Format(resourceDateLayout)
			if err := repo.CreatePlayer(account.ID, state, now); err != nil {
				t.Fatalf("CreatePlayer failed: %v", err)
			}
			before, err := repo.GetState(state.Player.ID)
			if err != nil {
				t.Fatalf("GetState before recruit failed: %v", err)
			}

			if _, err := svc.Recruit(state.Player.ID, "weiInfantry", 100000); !errors.Is(err, ErrInsufficientRes) {
				t.Fatalf("expected insufficient resources after discounted cost, got %v", err)
			}
			after, err := repo.GetState(state.Player.ID)
			if err != nil {
				t.Fatalf("GetState after recruit failed: %v", err)
			}
			if !reflect.DeepEqual(after, before) {
				t.Fatalf("failed recruit must preserve exact state\nbefore=%+v\nafter=%+v", before, after)
			}
			assertNoBattleReportsForTraitProcess(t, repo, state.Player.ID, traitCase.traitID+" failed recruit")
		})
	}
}

// TestBuildMilitaryActionResultIncludesTraitSettlementState 验证征兵响应不会丢失资源时点和非战斗特性进度。
func TestBuildMilitaryActionResultIncludesTraitSettlementState(t *testing.T) {
	state := GameState{
		Resources:            ResourceState{Items: map[string]int{"wood": 321}, Capacity: map[string]int{"wood": 1000}},
		ResourceProduction:   ResourceProduction{"wood": 53},
		ResourceSettledAt:    "2026-07-19T09:30:00Z",
		GeneralTraitProgress: map[string]float64{"caocao:weiwu_haoling:huWei": 0.5},
		ServerTime:           "2026-07-19T09:30:00Z",
	}
	result := BuildMilitaryActionResult(state)
	if result.ResourceProduction["wood"] != 53 || result.ResourceSettledAt != state.ResourceSettledAt || result.GeneralTraitProgress["caocao:weiwu_haoling:huWei"] != 0.5 {
		t.Fatalf("expected complete military settlement projection, result=%+v", result)
	}
	result.GeneralTraitProgress["caocao:weiwu_haoling:huWei"] = 0.25
	if state.GeneralTraitProgress["caocao:weiwu_haoling:huWei"] != 0.5 {
		t.Fatalf("military projection must not share trait progress map with state")
	}
	empty := BuildMilitaryActionResult(GameState{})
	if empty.GeneralTraitProgress == nil || len(empty.GeneralTraitProgress) != 0 {
		t.Fatalf("empty progress must be encoded as an authoritative empty object, got %+v", empty.GeneralTraitProgress)
	}
}
