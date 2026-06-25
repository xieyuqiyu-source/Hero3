package modifier

import (
	"testing"
	"time"
)

type testSource struct {
	mods []Modifier
}

func (s testSource) Modifiers(now time.Time) []Modifier { return s.mods }
func (s testSource) SourceName() string                 { return "test" }
func (s testSource) ExpiresAt() []time.Time             { return nil }

func TestComputeAttributeCombinesModes(t *testing.T) {
	source := testSource{mods: []Modifier{
		{Key: "attackBonus", Value: 20, Mode: ModeFlat},
		{Key: "attackBonus", Value: 0.5, Mode: ModePercentAdd},
		{Key: "attackBonus", Value: 1, Mode: ModePercentMultiply},
		{Key: "defenseBonus", Value: 999, Mode: ModeFlat},
	}}

	got := ComputeAttributeAt(100, "attackBonus", time.Unix(0, 0), source)
	want := 360.0
	if got != want {
		t.Fatalf("ComputeAttributeAt() = %v, want %v", got, want)
	}
}

func TestIsValidMode(t *testing.T) {
	for _, mode := range []string{ModeFlat, ModePercentAdd, ModePercentMultiply} {
		if !IsValidMode(mode) {
			t.Fatalf("expected mode %q to be valid", mode)
		}
	}
	if IsValidMode("custom") {
		t.Fatal("expected custom mode to be invalid")
	}
}
