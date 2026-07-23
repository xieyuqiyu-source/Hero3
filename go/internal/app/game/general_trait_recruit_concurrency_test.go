// 本文件验证征兵减耗特性在同一玩家并发提交时不会重复扣费、超扣资源或创建重复队列。
package game

import (
	"errors"
	"sync"
	"testing"
	"time"

	"hero3/internal/core/general"
)

// TestRecruitCostTraitsSerializeConcurrentRequests 验证资源只够一笔折后征兵时，两次并发请求只能成功一次。
func TestRecruitCostTraitsSerializeConcurrentRequests(t *testing.T) {
	for _, traitCase := range []struct {
		generalID   string
		generalName string
		traitID     string
		rate        float64
		available   int
	}{
		{generalID: "xunyu", generalName: "荀彧", traitID: "wangzuo_zhicai", rate: 0.05, available: 190},
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
			now := time.Now().UTC()
			account := Account{ID: "account_recruit_concurrent_" + traitCase.generalID, Username: "recruit_concurrent_" + traitCase.generalID, PasswordHash: "hash", CreatedAt: now}
			if err := repo.CreateAccount(account); err != nil {
				t.Fatalf("CreateAccount failed: %v", err)
			}
			state := newPlayerState("player_recruit_concurrent_"+traitCase.generalID, "Recruit Concurrent", "wei", traitCase.generalID, now)
			state.Buildings = nil
			state.ResourceSlots = nil
			state.ResourceProduction = ResourceProduction{}
			state.ResourceSettledAt = now.Format(resourceDateLayout)
			state.Resources.Items = map[string]int{"wood": traitCase.available}
			state.Resources.Capacity = map[string]int{"wood": 100000}
			if err := repo.CreatePlayer(account.ID, state, now); err != nil {
				t.Fatalf("CreatePlayer failed: %v", err)
			}

			start := make(chan struct{})
			errorsByRequest := make(chan error, 2)
			var workers sync.WaitGroup
			for range 2 {
				workers.Add(1)
				go func() {
					defer workers.Done()
					<-start
					_, err := svc.Recruit(state.Player.ID, "weiInfantry", 2)
					errorsByRequest <- err
				}()
			}
			close(start)
			workers.Wait()
			close(errorsByRequest)

			succeeded := 0
			insufficient := 0
			for err := range errorsByRequest {
				switch {
				case err == nil:
					succeeded++
				case errors.Is(err, ErrInsufficientRes):
					insufficient++
				default:
					t.Fatalf("unexpected concurrent recruit error: %v", err)
				}
			}
			if succeeded != 1 || insufficient != 1 {
				t.Fatalf("expected one success and one insufficient-resource result, success=%d insufficient=%d", succeeded, insufficient)
			}

			stored, err := repo.GetState(state.Player.ID)
			if err != nil {
				t.Fatalf("GetState failed: %v", err)
			}
			if stored.Resources.Items["wood"] != 0 || len(stored.RecruitQueues) != 1 || stored.RecruitQueues[0].UnitType != "weiInfantry" || stored.RecruitQueues[0].Amount != 2 {
				t.Fatalf("expected one discounted queue and zero remaining wood, state=%+v", stored)
			}
			assertNoBattleReportsForTraitProcess(t, repo, state.Player.ID, traitCase.traitID+" concurrent recruit")
		})
	}
}
