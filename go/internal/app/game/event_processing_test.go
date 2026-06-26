package game

import "testing"

func TestClaimEventForModuleIsIdempotent(t *testing.T) {
	repo := NewMemoryRepository()
	service := NewServiceWithRepository(repo)
	event := GameEvent{
		Type:      EventRewardGranted,
		PlayerID:  "player_event_claim",
		RefType:   "test",
		RefID:     "reward_1",
		CreatedAt: "2026-06-26T12:00:00Z",
	}

	first, err := service.ClaimEventForModule("mail", "test_handler", event)
	if err != nil {
		t.Fatalf("claim first: %v", err)
	}
	second, err := service.ClaimEventForModule("mail", "test_handler", event)
	if err != nil {
		t.Fatalf("claim second: %v", err)
	}
	if !first || second {
		t.Fatalf("expected first claim true and second false, got first=%v second=%v", first, second)
	}
}
