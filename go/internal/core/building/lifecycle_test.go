package building

import "testing"

func TestNormalizeStatusDefaultsToNormal(t *testing.T) {
	if got := NormalizeStatus(""); got != StatusNormal {
		t.Fatalf("NormalizeStatus() = %q, want %q", got, StatusNormal)
	}
}

func TestStatusValidation(t *testing.T) {
	if !IsValidStatus(StatusDamaged) {
		t.Fatal("expected damaged to be valid")
	}
	if IsValidStatus("unknown") {
		t.Fatal("expected unknown status to be invalid")
	}
}

func TestCanStartUpgrade(t *testing.T) {
	if !CanStartUpgrade("") {
		t.Fatal("legacy empty status should be upgradeable")
	}
	if CanStartUpgrade(StatusDestroyed) {
		t.Fatal("destroyed building should not be upgradeable")
	}
}
