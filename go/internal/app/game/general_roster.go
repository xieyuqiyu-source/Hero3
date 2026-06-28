// 本文件归口玩家多武将名册和武将占用状态的应用层规则。
package game

import (
	"strings"
	"time"
)

const (
	GeneralAssignmentMain = "main"
	GeneralStatusActive   = "active"
)

// EnsureGeneralRoster 确保旧单武将存档拥有多武将名册和主将占用记录。
func EnsureGeneralRoster(state *GameState, now time.Time) {
	if state == nil {
		return
	}
	if state.General != nil {
		applyHeroConfigToGeneral(state.General)
	}
	if len(state.Generals) == 0 && state.General != nil {
		state.Generals = []General{cloneGeneral(*state.General)}
	}
	if len(state.Generals) > 0 {
		for index := range state.Generals {
			applyHeroConfigToGeneral(&state.Generals[index])
		}
	}
	if state.General == nil && len(state.Generals) > 0 {
		state.General = cloneGeneralPtr(state.Generals[0])
	}
	if state.General != nil && !ownedGeneralExists(state.Generals, state.General.ID) {
		state.Generals = append(state.Generals, cloneGeneral(*state.General))
	}
	if len(state.GeneralAssignments) == 0 && state.General != nil {
		state.GeneralAssignments = []GeneralAssignment{newMainGeneralAssignment(state.General.ID, now)}
	}
	syncActiveGeneralToRoster(state)
	if mainID := mainAssignedGeneralID(state.GeneralAssignments); mainID != "" {
		if general, ok := findOwnedGeneral(state.Generals, mainID); ok {
			state.General = cloneGeneralPtr(general)
		}
	}
	syncActiveGeneralToRoster(state)
}

// AddOwnedGeneral 给玩家增加一个已拥有武将。
func AddOwnedGeneral(state *GameState, generalID string, now time.Time) error {
	generalID = strings.TrimSpace(generalID)
	if state == nil {
		return ErrPlayerNotFound
	}
	if generalID == "" {
		return ErrInvalidGeneral
	}
	hero, ok := GetHeroConfig(generalID)
	if !ok || !hero.Enabled {
		return ErrInvalidGeneral
	}
	if state.Player.Faction != "" && hero.Faction != state.Player.Faction {
		return ErrInvalidGeneral
	}
	EnsureGeneralRoster(state, now)
	if ownedGeneralExists(state.Generals, generalID) {
		return ErrInvalidGeneral
	}
	general := newGeneral(hero.Faction, generalID)
	if general == nil {
		return ErrInvalidGeneral
	}
	state.Generals = append(state.Generals, *general)
	if state.General == nil {
		state.General = cloneGeneralPtr(*general)
		state.GeneralAssignments = upsertMainGeneralAssignment(state.GeneralAssignments, generalID, now)
	}
	return nil
}

// SetActiveGeneral 把已拥有武将设置为当前主将。
func SetActiveGeneral(state *GameState, generalID string, now time.Time) error {
	generalID = strings.TrimSpace(generalID)
	if state == nil {
		return ErrPlayerNotFound
	}
	EnsureGeneralRoster(state, now)
	general, ok := findOwnedGeneral(state.Generals, generalID)
	if !ok {
		return ErrGeneralNotFound
	}
	if !generalAvailableForReinforcement(state.GeneralAssignments, generalID) {
		return ErrGeneralBusy
	}
	state.General = cloneGeneralPtr(general)
	state.GeneralAssignments = upsertMainGeneralAssignment(state.GeneralAssignments, generalID, now)
	return nil
}

// syncActiveGeneralToRoster 把当前主将修改同步回已拥有武将列表和主将占用。
func syncActiveGeneralToRoster(state *GameState) {
	if state == nil || state.General == nil {
		return
	}
	applyHeroConfigToGeneral(state.General)
	for index := range state.Generals {
		if state.Generals[index].ID == state.General.ID {
			state.Generals[index] = cloneGeneral(*state.General)
			return
		}
	}
	state.Generals = append(state.Generals, cloneGeneral(*state.General))
}

func newMainGeneralAssignment(generalID string, now time.Time) GeneralAssignment {
	return GeneralAssignment{
		ID:         GeneralAssignmentMain,
		GeneralID:  strings.TrimSpace(generalID),
		Slot:       GeneralAssignmentMain,
		Status:     GeneralStatusActive,
		AssignedAt: now.UTC().Format(resourceDateLayout),
	}
}

func upsertMainGeneralAssignment(assignments []GeneralAssignment, generalID string, now time.Time) []GeneralAssignment {
	next := make([]GeneralAssignment, 0, len(assignments)+1)
	found := false
	for _, assignment := range assignments {
		if assignment.ID == GeneralAssignmentMain || assignment.Slot == GeneralAssignmentMain {
			assignment.ID = GeneralAssignmentMain
			assignment.Slot = GeneralAssignmentMain
			assignment.GeneralID = generalID
			assignment.Status = GeneralStatusActive
			if strings.TrimSpace(assignment.AssignedAt) == "" {
				assignment.AssignedAt = now.UTC().Format(resourceDateLayout)
			}
			found = true
		}
		next = append(next, assignment)
	}
	if !found {
		next = append(next, newMainGeneralAssignment(generalID, now))
	}
	return next
}

func mainAssignedGeneralID(assignments []GeneralAssignment) string {
	for _, assignment := range assignments {
		if assignment.ID == GeneralAssignmentMain || assignment.Slot == GeneralAssignmentMain {
			return strings.TrimSpace(assignment.GeneralID)
		}
	}
	return ""
}

func ownedGeneralExists(generals []General, generalID string) bool {
	_, ok := findOwnedGeneral(generals, generalID)
	return ok
}

func findOwnedGeneral(generals []General, generalID string) (General, bool) {
	generalID = strings.TrimSpace(generalID)
	for _, general := range generals {
		if strings.TrimSpace(general.ID) == generalID {
			return general, true
		}
	}
	return General{}, false
}

func cloneGeneral(src General) General {
	dst := src
	if src.Stats != nil {
		dst.Stats = make(map[string]int, len(src.Stats))
		for key, value := range src.Stats {
			dst.Stats[key] = value
		}
	}
	if src.Attributes != nil {
		dst.Attributes = make(map[string]float64, len(src.Attributes))
		for key, value := range src.Attributes {
			dst.Attributes[key] = value
		}
	}
	if src.Buffs != nil {
		dst.Buffs = make(map[string]float64, len(src.Buffs))
		for key, value := range src.Buffs {
			dst.Buffs[key] = value
		}
	}
	if src.AttributeBreakdown != nil {
		dst.AttributeBreakdown = make(map[string][]GeneralAttributeBreakdownItem, len(src.AttributeBreakdown))
		for key, values := range src.AttributeBreakdown {
			dst.AttributeBreakdown[key] = append([]GeneralAttributeBreakdownItem{}, values...)
		}
	}
	if src.Traits != nil {
		dst.Traits = append([]GeneralTraitInstance{}, src.Traits...)
	}
	return dst
}

func cloneGeneralPtr(src General) *General {
	dst := cloneGeneral(src)
	return &dst
}
