package runtime

import (
	"sync"
	"time"
)

// EventType represents the type of runtime event.
type EventType string

const (
	EventIterationStart   EventType = "iteration_start"
	EventIterationEnd     EventType = "iteration_end"
	EventToolCallStart    EventType = "tool_call_start"
	EventToolCallEnd      EventType = "tool_call_end"
	EventProviderRequest  EventType = "provider_request"
	EventProviderResponse EventType = "provider_response"
	EventGuardViolation   EventType = "guard_violation"
	EventVerificationPass EventType = "verification_pass"
	EventVerificationFail EventType = "verification_fail"
	EventSessionComplete  EventType = "session_complete"
	EventSessionError     EventType = "session_error"
	EventMemoryArchived   EventType = "memory_archived"
	EventContextPruned    EventType = "context_pruned"
)

// Event represents a runtime event with associated data.
type Event struct {
	Type      EventType
	Timestamp time.Time
	SessionID string
	Data      map[string]interface{}
}

// EventHandler is a function that handles events.
type EventHandler func(Event)

// EventBus manages event publication and subscription.
// It provides a decoupled way for runtime components to communicate.
type EventBus struct {
	mu       sync.RWMutex
	handlers map[EventType][]EventHandler
	allHandlers []EventHandler
}

// NewEventBus creates a new event bus.
func NewEventBus() *EventBus {
	return &EventBus{
		handlers: make(map[EventType][]EventHandler),
	}
}

// Subscribe registers a handler for a specific event type.
func (eb *EventBus) Subscribe(eventType EventType, handler EventHandler) {
	eb.mu.Lock()
	defer eb.mu.Unlock()
	eb.handlers[eventType] = append(eb.handlers[eventType], handler)
}

// SubscribeAll registers a handler for all event types.
func (eb *EventBus) SubscribeAll(handler EventHandler) {
	eb.mu.Lock()
	defer eb.mu.Unlock()
	eb.allHandlers = append(eb.allHandlers, handler)
}

// Publish sends an event to all registered handlers.
func (eb *EventBus) Publish(event Event) {
	eb.mu.RLock()
	defer eb.mu.RUnlock()

	// Set timestamp if not already set
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}

	// Notify specific handlers
	if handlers, ok := eb.handlers[event.Type]; ok {
		for _, handler := range handlers {
			handler(event)
		}
	}

	// Notify all-event handlers
	for _, handler := range eb.allHandlers {
		handler(event)
	}
}

// PublishSimple is a convenience method for publishing events without additional data.
func (eb *EventBus) PublishSimple(eventType EventType, sessionID string) {
	eb.Publish(Event{
		Type:      eventType,
		SessionID: sessionID,
	})
}

// PublishWithData publishes an event with associated data.
func (eb *EventBus) PublishWithData(eventType EventType, sessionID string, data map[string]interface{}) {
	eb.Publish(Event{
		Type:      eventType,
		SessionID: sessionID,
		Data:      data,
	})
}
