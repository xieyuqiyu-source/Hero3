package event

import (
	"sync"
	"time"
)

type Event struct {
	Type      string         `json:"type"`
	PlayerID  string         `json:"playerId,omitempty"`
	AccountID string         `json:"accountId,omitempty"`
	RefType   string         `json:"refType,omitempty"`
	RefID     string         `json:"refId,omitempty"`
	Payload   map[string]any `json:"payload,omitempty"`
	CreatedAt string         `json:"createdAt"`
}

type Handler func(Event)

type TypeValidator func(eventType string) bool

type TimestampFunc func(time.Time) string

type Bus struct {
	mu        sync.RWMutex
	handlers  map[string][]Handler
	validate  TypeValidator
	timestamp TimestampFunc
}

func NewBus(validate TypeValidator, timestamp TimestampFunc) *Bus {
	if validate == nil {
		validate = func(string) bool { return true }
	}
	if timestamp == nil {
		timestamp = func(t time.Time) string { return t.UTC().Format(time.RFC3339) }
	}
	return &Bus{
		handlers:  map[string][]Handler{},
		validate:  validate,
		timestamp: timestamp,
	}
}

func (b *Bus) Subscribe(eventType string, handler Handler) {
	if b == nil || eventType == "" || handler == nil {
		return
	}
	if !b.validate(eventType) {
		return
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	b.handlers[eventType] = append(b.handlers[eventType], handler)
}

func (b *Bus) Publish(event Event) {
	if b == nil || event.Type == "" {
		return
	}
	if !b.validate(event.Type) {
		return
	}
	if event.CreatedAt == "" {
		event.CreatedAt = b.timestamp(time.Now())
	}

	b.mu.RLock()
	handlers := append([]Handler(nil), b.handlers[event.Type]...)
	b.mu.RUnlock()

	for _, handler := range handlers {
		handler(event)
	}
}
