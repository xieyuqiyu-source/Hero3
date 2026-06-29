// 本文件归口 MySQL 局部视图只读查询，避免高频页面先组装完整 GameState。
package storage

import (
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"hero3/internal/app/game"
)

type playerViewBase struct {
	state    game.GameState
	mailCode string
}

// GetPlayerSummaryView 直接读取玩家摘要需要的轻量字段。
func (r *MySQLRepository) GetPlayerSummaryView(playerID string) (game.PlayerSummaryView, error) {
	base, err := r.loadPlayerViewBase(playerID)
	if err != nil {
		return game.PlayerSummaryView{}, err
	}
	game.ProjectStateForView(&base.state, time.Now())
	return game.PlayerSummaryView{
		Player:             base.state.Player,
		CityGold:           base.state.CityGold,
		UnreadMessageCount: base.state.UnreadMessageCount,
		UnreadMailCount:    base.state.UnreadMailCount,
		ServerTime:         base.state.ServerTime,
	}, nil
}

// GetCityView 直接读取城池页面需要的资源、建筑、资源田和 Buff。
func (r *MySQLRepository) GetCityView(playerID string) (game.CityView, error) {
	state, err := r.loadViewState(playerID, viewLoadOptions{resources: true, buildings: true, resourceSlots: true, generals: true, buffs: true})
	if err != nil {
		return game.CityView{}, err
	}
	game.ProjectStateForView(&state, time.Now())
	return game.CityView{
		Player:             state.Player,
		Buildings:          state.Buildings,
		ResourceSlots:      state.ResourceSlots,
		Resources:          state.Resources,
		ResourceProduction: state.ResourceProduction,
		CityGold:           state.CityGold,
		ActiveModifiers:    state.ActiveModifiers,
		ServerTime:         state.ServerTime,
	}, nil
}

// GetResourceView 直接读取资源栏需要的资源、建筑、资源田、武将和 Buff。
func (r *MySQLRepository) GetResourceView(playerID string) (game.ResourceView, error) {
	state, err := r.loadViewState(playerID, viewLoadOptions{resources: true, buildings: true, resourceSlots: true, generals: true, buffs: true})
	if err != nil {
		return game.ResourceView{}, err
	}
	game.ProjectStateForView(&state, time.Now())
	return game.ResourceView{
		Resources:          state.Resources,
		ResourceProduction: state.ResourceProduction,
		ResourceSettledAt:  state.ResourceSettledAt,
		ProductionBoost:    state.ProductionBoost,
		ProductionBoostEnd: state.ProductionBoostEnd,
		CapacityBoost:      state.CapacityBoost,
		CapacityBoostEnd:   state.CapacityBoostEnd,
		ActiveModifiers:    state.ActiveModifiers,
		ServerTime:         state.ServerTime,
	}, nil
}

// GetMilitaryView 直接读取军事页面需要的兵力、征兵队列、武将和必要资源。
func (r *MySQLRepository) GetMilitaryView(playerID string) (game.MilitaryView, error) {
	state, err := r.loadViewState(playerID, viewLoadOptions{resources: true, buildings: true, resourceSlots: true, army: true, recruitQueues: true, generals: true, buffs: true})
	if err != nil {
		return game.MilitaryView{}, err
	}
	game.ProjectStateForView(&state, time.Now())
	return game.MilitaryView{
		Army:               state.Army,
		RecruitQueues:      state.RecruitQueues,
		Resources:          state.Resources,
		CityGold:           state.CityGold,
		Buildings:          state.Buildings,
		ActiveModifiers:    state.ActiveModifiers,
		General:            state.General,
		Generals:           state.Generals,
		GeneralAssignments: state.GeneralAssignments,
		ServerTime:         state.ServerTime,
	}, nil
}

// GetInventoryView 直接读取背包页面需要的背包权威表。
func (r *MySQLRepository) GetInventoryView(playerID string) (game.InventoryView, error) {
	state, err := r.loadViewState(playerID, viewLoadOptions{inventory: true})
	if err != nil {
		return game.InventoryView{}, err
	}
	game.ProjectStateForView(&state, time.Now())
	if state.Inventory == nil {
		state.Inventory = map[string]game.ItemStack{}
	}
	return game.InventoryView{Inventory: state.Inventory, InventorySlots: state.InventorySlots, ServerTime: state.ServerTime}, nil
}

// GetGeneralsView 直接读取武将页面需要的武将、占用和 Buff。
func (r *MySQLRepository) GetGeneralsView(playerID string) (game.GeneralsView, error) {
	state, err := r.loadViewState(playerID, viewLoadOptions{resources: true, buildings: true, resourceSlots: true, generals: true, buffs: true})
	if err != nil {
		return game.GeneralsView{}, err
	}
	game.ProjectStateForView(&state, time.Now())
	return game.GeneralsView{
		General:            state.General,
		Generals:           state.Generals,
		GeneralAssignments: state.GeneralAssignments,
		GeneralChangeUntil: state.GeneralChangeUntil,
		ActiveModifiers:    state.ActiveModifiers,
		ServerTime:         state.ServerTime,
	}, nil
}

type viewLoadOptions struct {
	resources     bool
	inventory     bool
	buildings     bool
	resourceSlots bool
	army          bool
	recruitQueues bool
	generals      bool
	buffs         bool
}

func (r *MySQLRepository) loadPlayerViewBase(playerID string) (playerViewBase, error) {
	var stateJSON []byte
	var mailCode string
	err := r.db.QueryRow(`SELECT state_json, mail_code FROM players WHERE id = ? LIMIT 1`, playerID).Scan(&stateJSON, &mailCode)
	if errors.Is(err, sql.ErrNoRows) {
		return playerViewBase{}, game.ErrPlayerNotFound
	}
	if err != nil {
		return playerViewBase{}, err
	}
	var state game.GameState
	if err := json.Unmarshal(stateJSON, &state); err != nil {
		return playerViewBase{}, err
	}
	if state.Player.MailCode == "" {
		state.Player.MailCode = mailCode
	}
	return playerViewBase{state: state, mailCode: mailCode}, nil
}

func (r *MySQLRepository) loadViewState(playerID string, options viewLoadOptions) (game.GameState, error) {
	base, err := r.loadPlayerViewBase(playerID)
	if err != nil {
		return game.GameState{}, err
	}
	state := base.state
	if options.resources {
		resources, found, err := loadPlayerResources(r.db, playerID)
		if err != nil {
			return game.GameState{}, err
		}
		if err := applyAuthoritativeResources(&state, resources, found); err != nil {
			return game.GameState{}, err
		}
	}
	if options.inventory {
		inventory, slots, found, err := loadPlayerInventory(r.db, playerID)
		if err != nil {
			return game.GameState{}, err
		}
		if err := applyAuthoritativeInventory(&state, inventory, slots, found); err != nil {
			return game.GameState{}, err
		}
	}
	if options.buildings {
		buildings, found, err := loadPlayerBuildings(r.db, playerID)
		if err != nil {
			return game.GameState{}, err
		}
		if err := applyAuthoritativeBuildings(&state, buildings, found); err != nil {
			return game.GameState{}, err
		}
	}
	if options.resourceSlots {
		slots, found, err := loadPlayerResourceSlots(r.db, playerID)
		if err != nil {
			return game.GameState{}, err
		}
		if err := applyAuthoritativeResourceSlots(&state, slots, found); err != nil {
			return game.GameState{}, err
		}
	}
	if options.army {
		army, found, err := loadPlayerArmy(r.db, playerID)
		if err != nil {
			return game.GameState{}, err
		}
		if err := applyAuthoritativeArmy(&state, army, found); err != nil {
			return game.GameState{}, err
		}
	}
	if options.recruitQueues {
		queues, found, err := loadPlayerRecruitQueues(r.db, playerID)
		if err != nil {
			return game.GameState{}, err
		}
		if err := applyAuthoritativeRecruitQueues(&state, queues, found); err != nil {
			return game.GameState{}, err
		}
	}
	if options.generals {
		generals, found, err := loadPlayerGenerals(r.db, playerID)
		if err != nil {
			return game.GameState{}, err
		}
		assignments, _, err := loadPlayerGeneralAssignments(r.db, playerID)
		if err != nil {
			return game.GameState{}, err
		}
		if err := applyAuthoritativeGenerals(&state, generals, assignments, found); err != nil {
			return game.GameState{}, err
		}
	}
	if options.buffs {
		buffs, found, err := loadPlayerBuffs(r.db, playerID)
		if err != nil {
			return game.GameState{}, err
		}
		if err := applyAuthoritativeBuffs(&state, buffs, found); err != nil {
			return game.GameState{}, err
		}
	}
	return state, nil
}
