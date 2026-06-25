package game

import (
	"strings"
	"time"

	corebuilding "hero3/internal/core/building"
)

const (
	BuildingStatusNormal       = corebuilding.StatusNormal
	BuildingStatusConstructing = corebuilding.StatusConstructing
	BuildingStatusUpgrading    = corebuilding.StatusUpgrading
	BuildingStatusLocked       = corebuilding.StatusLocked
	BuildingStatusDamaged      = corebuilding.StatusDamaged
	BuildingStatusDestroyed    = corebuilding.StatusDestroyed
	BuildingStatusRepairing    = corebuilding.StatusRepairing
	BuildingStatusDisabled     = corebuilding.StatusDisabled
	BuildingStatusProtected    = corebuilding.StatusProtected
	BuildingStatusOccupied     = corebuilding.StatusOccupied

	BuildingMutationSetStatus       = corebuilding.MutationSetStatus
	BuildingMutationLevelDelta      = corebuilding.MutationLevelDelta
	BuildingMutationStartUpgrade    = corebuilding.MutationStartUpgrade
	BuildingMutationCompleteUpgrade = corebuilding.MutationCompleteUpgrade
	BuildingMutationDamage          = corebuilding.MutationDamage
	BuildingMutationDestroy         = corebuilding.MutationDestroy
	BuildingMutationRepair          = corebuilding.MutationRepair
	BuildingMutationLock            = corebuilding.MutationLock
	BuildingMutationUnlock          = corebuilding.MutationUnlock
	BuildingMutationOccupy          = corebuilding.MutationOccupy
	BuildingMutationRelease         = corebuilding.MutationRelease
)

type BuildingMutation = corebuilding.Mutation

func normalizeBuildingStatus(status string) string {
	return corebuilding.NormalizeStatus(status)
}

func buildingIsOperational(building Building) bool {
	return corebuilding.IsOperational(building.Status)
}

func buildingCanStartUpgrade(building Building) bool {
	return corebuilding.CanStartUpgrade(building.Status) && building.UpgradeEndsAt == nil
}

func applyBuildingMutation(building *Building, mutation BuildingMutation, now time.Time) error {
	if building == nil {
		return ErrBuildingNotFound
	}
	switch strings.TrimSpace(mutation.Type) {
	case corebuilding.MutationSetStatus:
		status := normalizeBuildingStatus(mutation.Status)
		if !corebuilding.IsValidStatus(status) {
			return ErrInvalidBuildingStatus
		}
		building.Status = status
	case corebuilding.MutationLevelDelta:
		next := building.Level + mutation.LevelDelta
		if next < 0 {
			next = 0
		}
		building.Level = next
	case corebuilding.MutationStartUpgrade:
		if !buildingCanStartUpgrade(*building) {
			return ErrBuildingStatusBlocked
		}
		building.Status = BuildingStatusUpgrading
	case corebuilding.MutationCompleteUpgrade:
		building.Level++
		building.UpgradeEndsAt = nil
		building.Status = BuildingStatusNormal
	case corebuilding.MutationDamage:
		building.Status = BuildingStatusDamaged
	case corebuilding.MutationDestroy:
		building.Status = BuildingStatusDestroyed
		building.UpgradeEndsAt = nil
	case corebuilding.MutationRepair:
		building.Status = BuildingStatusNormal
	case corebuilding.MutationLock:
		building.Status = BuildingStatusLocked
	case corebuilding.MutationUnlock:
		building.Status = BuildingStatusNormal
	case corebuilding.MutationOccupy:
		building.Status = BuildingStatusOccupied
	case corebuilding.MutationRelease:
		building.Status = BuildingStatusNormal
	default:
		return ErrInvalidBuildingMutation
	}
	if building.Status == BuildingStatusNormal {
		building.StatusEndsAt = nil
	}
	_ = now
	return nil
}

func (s *Service) MutateBuilding(playerID string, mutation BuildingMutation) (GameState, error) {
	playerID = strings.TrimSpace(playerID)
	mutation.BuildingID = strings.TrimSpace(mutation.BuildingID)
	if playerID == "" {
		return GameState{}, ErrPlayerNotFound
	}
	if mutation.BuildingID == "" {
		return GameState{}, ErrBuildingNotFound
	}

	now := time.Now()
	var before, after coreAssetSnapshot
	state, err := s.repo.UpdatePlayerState(playerID, now, func(state *GameState) error {
		nextState, _ := settleResources(*state, now)
		*state = nextState
		before = snapshotCoreAssets(state)

		building := findBuildingByID(state, mutation.BuildingID)
		if building == nil {
			return ErrBuildingNotFound
		}
		if err := applyBuildingMutation(building, mutation, now); err != nil {
			return err
		}
		state.ResourceSettledAt = now.UTC().Format(resourceDateLayout)
		state.ServerTime = now.UTC().Format(resourceDateLayout)
		nextState, _ = settleResources(*state, now)
		*state = nextState
		after = snapshotCoreAssets(state)
		return nil
	})
	if err != nil {
		return GameState{}, err
	}
	s.publishCoreAssetDiff(playerID, firstNonEmpty(mutation.Reason, mutation.Type), mutation.BuildingID, before, after, now)
	hydrateStateForResponse(&state, now)
	return state, nil
}

func findBuildingByID(state *GameState, buildingID string) *Building {
	if state == nil {
		return nil
	}
	for i := range state.Buildings {
		if state.Buildings[i].ID == buildingID {
			return &state.Buildings[i]
		}
	}
	return nil
}
