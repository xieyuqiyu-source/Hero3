package building

import "strings"

const (
	StatusNormal       = "normal"
	StatusConstructing = "constructing"
	StatusUpgrading    = "upgrading"
	StatusLocked       = "locked"
	StatusDamaged      = "damaged"
	StatusDestroyed    = "destroyed"
	StatusRepairing    = "repairing"
	StatusDisabled     = "disabled"
	StatusProtected    = "protected"
	StatusOccupied     = "occupied"
)

const (
	MutationSetStatus       = "set_status"
	MutationLevelDelta      = "level_delta"
	MutationStartUpgrade    = "start_upgrade"
	MutationCompleteUpgrade = "complete_upgrade"
	MutationDamage          = "damage"
	MutationDestroy         = "destroy"
	MutationRepair          = "repair"
	MutationLock            = "lock"
	MutationUnlock          = "unlock"
	MutationOccupy          = "occupy"
	MutationRelease         = "release"
)

type Mutation struct {
	Type       string
	BuildingID string
	LevelDelta int
	Status     string
	Reason     string
}

func NormalizeStatus(status string) string {
	status = strings.TrimSpace(status)
	if status == "" {
		return StatusNormal
	}
	return status
}

func IsValidStatus(status string) bool {
	switch NormalizeStatus(status) {
	case StatusNormal,
		StatusConstructing,
		StatusUpgrading,
		StatusLocked,
		StatusDamaged,
		StatusDestroyed,
		StatusRepairing,
		StatusDisabled,
		StatusProtected,
		StatusOccupied:
		return true
	default:
		return false
	}
}

func CanStartUpgrade(status string) bool {
	return NormalizeStatus(status) == StatusNormal
}

func IsOperational(status string) bool {
	switch NormalizeStatus(status) {
	case StatusDestroyed, StatusDisabled:
		return false
	default:
		return true
	}
}
