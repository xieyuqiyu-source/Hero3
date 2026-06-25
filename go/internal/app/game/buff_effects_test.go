package game

import (
	"errors"
	"testing"
	"time"
)

// 本文件验证 Buff/DeBuff 的 Modifier scope 和过期过滤。

func TestValidateBuffModifierSpecRejectsInvalidScope(t *testing.T) {
	if err := validateBuffModifierSpec("badBonus", "percentAdd"); !errors.Is(err, ErrInvalidBuffKey) {
		t.Fatalf("expected ErrInvalidBuffKey, got %v", err)
	}
	if err := validateBuffModifierSpec(StatAttackBonus, "badMode"); !errors.Is(err, ErrInvalidBuffMode) {
		t.Fatalf("expected ErrInvalidBuffMode, got %v", err)
	}
	if err := validateBuffModifierSpec(StatAttackBonus, "percentAdd"); err != nil {
		t.Fatalf("expected valid buff modifier spec, got %v", err)
	}
}

func TestBuffListSourceFiltersExpiredModifiers(t *testing.T) {
	now := time.Now()
	expired := now.Add(-time.Hour).UTC().Format(resourceDateLayout)
	future := now.Add(time.Hour).UTC().Format(resourceDateLayout)
	source := &BuffListSource{Buffs: []Buff{
		{Key: StatAttackBonus, Value: 0.1, Mode: "percentAdd", ExpiresAt: expired},
		{Key: StatDefenseBonus, Value: 0.2, Mode: "percentAdd", ExpiresAt: future},
	}}

	mods := source.Modifiers(now)
	if len(mods) != 1 {
		t.Fatalf("expected 1 active modifier, got %d", len(mods))
	}
	if mods[0].Key != StatDefenseBonus {
		t.Fatalf("expected active defense bonus, got %+v", mods[0])
	}
}

func TestCleanExpiredBuffsRemovesExpiredState(t *testing.T) {
	now := time.Now()
	expired := now.Add(-time.Hour).UTC().Format(resourceDateLayout)
	future := now.Add(time.Hour).UTC().Format(resourceDateLayout)
	state := GameState{Buffs: []Buff{
		{ID: "expired", Key: StatAttackBonus, Mode: "percentAdd", ExpiresAt: expired},
		{ID: "active", Key: StatDefenseBonus, Mode: "percentAdd", ExpiresAt: future},
	}}

	cleanExpiredBuffs(&state, now)
	if len(state.Buffs) != 1 || state.Buffs[0].ID != "active" {
		t.Fatalf("expected only active buff to remain, got %+v", state.Buffs)
	}
}
