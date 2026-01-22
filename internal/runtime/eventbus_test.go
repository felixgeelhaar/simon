package runtime

import (
	"sync"
	"testing"
	"time"
)

func TestNewEventBus(t *testing.T) {
	eb := NewEventBus()
	if eb == nil {
		t.Fatal("expected non-nil EventBus")
	}
	if eb.handlers == nil {
		t.Fatal("expected non-nil handlers map")
	}
}

func TestEventBus_Subscribe(t *testing.T) {
	eb := NewEventBus()
	called := false

	eb.Subscribe(EventIterationStart, func(e Event) {
		called = true
	})

	eb.Publish(Event{Type: EventIterationStart})

	if !called {
		t.Error("handler was not called")
	}
}

func TestEventBus_SubscribeAll(t *testing.T) {
	eb := NewEventBus()
	count := 0

	eb.SubscribeAll(func(e Event) {
		count++
	})

	eb.Publish(Event{Type: EventIterationStart})
	eb.Publish(Event{Type: EventIterationEnd})
	eb.Publish(Event{Type: EventSessionComplete})

	if count != 3 {
		t.Errorf("expected 3 calls, got %d", count)
	}
}

func TestEventBus_PublishWithData(t *testing.T) {
	eb := NewEventBus()
	var received Event

	eb.Subscribe(EventToolCallStart, func(e Event) {
		received = e
	})

	data := map[string]interface{}{"tool": "run_shell"}
	eb.PublishWithData(EventToolCallStart, "sess-123", data)

	if received.SessionID != "sess-123" {
		t.Errorf("expected session 'sess-123', got %q", received.SessionID)
	}
	if received.Data["tool"] != "run_shell" {
		t.Error("data not properly passed")
	}
}

func TestEventBus_PublishSimple(t *testing.T) {
	eb := NewEventBus()
	var received Event

	eb.Subscribe(EventSessionComplete, func(e Event) {
		received = e
	})

	eb.PublishSimple(EventSessionComplete, "sess-456")

	if received.SessionID != "sess-456" {
		t.Errorf("expected session 'sess-456', got %q", received.SessionID)
	}
	if received.Type != EventSessionComplete {
		t.Errorf("expected type EventSessionComplete, got %v", received.Type)
	}
}

func TestEventBus_TimestampAutoSet(t *testing.T) {
	eb := NewEventBus()
	var received Event

	eb.Subscribe(EventIterationStart, func(e Event) {
		received = e
	})

	before := time.Now()
	eb.Publish(Event{Type: EventIterationStart})
	after := time.Now()

	if received.Timestamp.Before(before) || received.Timestamp.After(after) {
		t.Error("timestamp not set correctly")
	}
}

func TestEventBus_MultipleHandlers(t *testing.T) {
	eb := NewEventBus()
	count := 0
	var mu sync.Mutex

	for i := 0; i < 5; i++ {
		eb.Subscribe(EventIterationStart, func(e Event) {
			mu.Lock()
			count++
			mu.Unlock()
		})
	}

	eb.Publish(Event{Type: EventIterationStart})

	mu.Lock()
	defer mu.Unlock()
	if count != 5 {
		t.Errorf("expected 5 handler calls, got %d", count)
	}
}

func TestEventBus_DifferentEventTypes(t *testing.T) {
	eb := NewEventBus()
	startCalled := false
	endCalled := false

	eb.Subscribe(EventIterationStart, func(e Event) {
		startCalled = true
	})
	eb.Subscribe(EventIterationEnd, func(e Event) {
		endCalled = true
	})

	eb.Publish(Event{Type: EventIterationStart})

	if !startCalled {
		t.Error("start handler was not called")
	}
	if endCalled {
		t.Error("end handler should not have been called")
	}
}

func TestEventBus_ConcurrentPublish(t *testing.T) {
	eb := NewEventBus()
	var count int
	var mu sync.Mutex

	eb.SubscribeAll(func(e Event) {
		mu.Lock()
		count++
		mu.Unlock()
	})

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			eb.Publish(Event{Type: EventIterationStart})
		}()
	}
	wg.Wait()

	mu.Lock()
	defer mu.Unlock()
	if count != 100 {
		t.Errorf("expected 100 events, got %d", count)
	}
}

func TestEventType_Constants(t *testing.T) {
	types := []EventType{
		EventIterationStart,
		EventIterationEnd,
		EventToolCallStart,
		EventToolCallEnd,
		EventProviderRequest,
		EventProviderResponse,
		EventGuardViolation,
		EventVerificationPass,
		EventVerificationFail,
		EventSessionComplete,
		EventSessionError,
		EventMemoryArchived,
		EventContextPruned,
	}

	for _, et := range types {
		if string(et) == "" {
			t.Error("event type should not be empty")
		}
	}
}
