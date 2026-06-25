package event

import (
	"testing"
	"time"
)

func TestBusPublishesOnlyRegisteredEvents(t *testing.T) {
	bus := NewBus(func(eventType string) bool {
		return eventType == "reward.granted"
	}, func(time.Time) string {
		return "2026-06-25T00:00:00Z"
	})

	var got []Event
	bus.Subscribe("unknown", func(event Event) {
		t.Fatal("unknown event handler should not be registered")
	})
	bus.Subscribe("reward.granted", func(event Event) {
		got = append(got, event)
	})

	bus.Publish(Event{Type: "unknown"})
	bus.Publish(Event{Type: "reward.granted", PlayerID: "p1"})

	if len(got) != 1 {
		t.Fatalf("published events = %d, want 1", len(got))
	}
	if got[0].CreatedAt != "2026-06-25T00:00:00Z" {
		t.Fatalf("CreatedAt = %q", got[0].CreatedAt)
	}
}
