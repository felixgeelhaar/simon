package runtime

import (
	"sync"
	"testing"

	"github.com/felixgeelhaar/simon/internal/provider"
)

// mockStorage implements store.Storage for testing
type mockStorage struct{}

func (m *mockStorage) CreateSession(session interface{}) error                         { return nil }
func (m *mockStorage) GetSession(id string) (interface{}, error)                       { return nil, nil }
func (m *mockStorage) UpdateSession(session interface{}) error                         { return nil }
func (m *mockStorage) SaveArtifact(artifact interface{}, content []byte) error         { return nil }
func (m *mockStorage) GetArtifact(id string) (interface{}, []byte, error)              { return nil, nil, nil }
func (m *mockStorage) ListArtifacts(sessionID string) ([]interface{}, error)           { return nil, nil }
func (m *mockStorage) SetConfig(key, value string) error                               { return nil }
func (m *mockStorage) GetConfig(key string) (string, error)                            { return "", nil }
func (m *mockStorage) AddMemory(content string, vector []float32, meta map[string]string) error { return nil }
func (m *mockStorage) SearchMemory(vector []float32, limit int) ([]interface{}, error) { return nil, nil }
func (m *mockStorage) Close() error                                                    { return nil }

func TestNewStateManager(t *testing.T) {
	sm := NewStateManager(nil)
	if sm == nil {
		t.Fatal("expected non-nil StateManager")
	}
	if sm.sessions == nil {
		t.Fatal("expected non-nil sessions map")
	}
}

func TestStateManager_InitSession(t *testing.T) {
	sm := NewStateManager(nil)
	state := sm.InitSession("test-session")

	if state == nil {
		t.Fatal("expected non-nil SessionState")
	}
	if state.SessionID != "test-session" {
		t.Errorf("expected session ID 'test-session', got %q", state.SessionID)
	}
	if state.CurrentIteration != 0 {
		t.Errorf("expected iteration 0, got %d", state.CurrentIteration)
	}
	if state.Status != "initialized" {
		t.Errorf("expected status 'initialized', got %q", state.Status)
	}
}

func TestStateManager_GetState(t *testing.T) {
	sm := NewStateManager(nil)
	sm.InitSession("sess-1")

	state := sm.GetState("sess-1")
	if state == nil {
		t.Fatal("expected non-nil state")
	}

	missing := sm.GetState("nonexistent")
	if missing != nil {
		t.Error("expected nil for nonexistent session")
	}
}

func TestStateManager_IncrementIteration(t *testing.T) {
	sm := NewStateManager(nil)
	sm.InitSession("sess-1")

	iter1 := sm.IncrementIteration("sess-1")
	if iter1 != 1 {
		t.Errorf("expected iteration 1, got %d", iter1)
	}

	iter2 := sm.IncrementIteration("sess-1")
	if iter2 != 2 {
		t.Errorf("expected iteration 2, got %d", iter2)
	}

	// Nonexistent session
	iter := sm.IncrementIteration("nonexistent")
	if iter != 0 {
		t.Errorf("expected 0 for nonexistent session, got %d", iter)
	}
}

func TestStateManager_TokenUsage(t *testing.T) {
	sm := NewStateManager(nil)
	sm.InitSession("sess-1")

	sm.AddTokenUsage("sess-1", 100, 50)
	prompt, output := sm.GetTokenUsage("sess-1")
	if prompt != 100 || output != 50 {
		t.Errorf("expected (100, 50), got (%d, %d)", prompt, output)
	}

	sm.AddTokenUsage("sess-1", 200, 100)
	prompt, output = sm.GetTokenUsage("sess-1")
	if prompt != 300 || output != 150 {
		t.Errorf("expected (300, 150), got (%d, %d)", prompt, output)
	}
}

func TestStateManager_History(t *testing.T) {
	sm := NewStateManager(nil)
	sm.InitSession("sess-1")

	msg1 := provider.Message{Role: "user", Content: "hello"}
	msg2 := provider.Message{Role: "assistant", Content: "hi"}

	sm.AppendHistory("sess-1", msg1)
	sm.AppendHistory("sess-1", msg2)

	history := sm.GetHistory("sess-1")
	if len(history) != 2 {
		t.Errorf("expected 2 messages, got %d", len(history))
	}
	if history[0].Role != "user" {
		t.Error("first message should be from user")
	}
	if history[1].Role != "assistant" {
		t.Error("second message should be from assistant")
	}
}

func TestStateManager_ReplaceHistory(t *testing.T) {
	sm := NewStateManager(nil)
	sm.InitSession("sess-1")

	sm.AppendHistory("sess-1", provider.Message{Role: "user", Content: "old"})

	newHistory := []provider.Message{
		{Role: "user", Content: "new"},
	}
	sm.ReplaceHistory("sess-1", newHistory)

	history := sm.GetHistory("sess-1")
	if len(history) != 1 {
		t.Errorf("expected 1 message, got %d", len(history))
	}
	if history[0].Content != "new" {
		t.Errorf("expected content 'new', got %q", history[0].Content)
	}
}

func TestStateManager_Status(t *testing.T) {
	sm := NewStateManager(nil)
	sm.InitSession("sess-1")

	sm.SetStatus("sess-1", "running")
	status := sm.GetStatus("sess-1")
	if status != "running" {
		t.Errorf("expected status 'running', got %q", status)
	}

	sm.SetStatus("sess-1", "completed")
	status = sm.GetStatus("sess-1")
	if status != "completed" {
		t.Errorf("expected status 'completed', got %q", status)
	}
}

func TestStateManager_CleanupSession(t *testing.T) {
	sm := NewStateManager(nil)
	sm.InitSession("sess-1")

	sm.CleanupSession("sess-1")

	state := sm.GetState("sess-1")
	if state != nil {
		t.Error("expected nil state after cleanup")
	}
}

func TestStateManager_HistoryLength(t *testing.T) {
	sm := NewStateManager(nil)
	sm.InitSession("sess-1")

	if sm.HistoryLength("sess-1") != 0 {
		t.Error("expected history length 0")
	}

	sm.AppendHistory("sess-1", provider.Message{Role: "user", Content: "1"})
	sm.AppendHistory("sess-1", provider.Message{Role: "assistant", Content: "2"})

	if sm.HistoryLength("sess-1") != 2 {
		t.Errorf("expected history length 2, got %d", sm.HistoryLength("sess-1"))
	}
}

func TestStateManager_ConcurrentAccess(t *testing.T) {
	sm := NewStateManager(nil)
	sm.InitSession("sess-1")

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(3)
		go func() {
			defer wg.Done()
			sm.IncrementIteration("sess-1")
		}()
		go func() {
			defer wg.Done()
			sm.AddTokenUsage("sess-1", 1, 1)
		}()
		go func() {
			defer wg.Done()
			sm.AppendHistory("sess-1", provider.Message{Role: "user", Content: "test"})
		}()
	}
	wg.Wait()

	state := sm.GetState("sess-1")
	if state.CurrentIteration != 100 {
		t.Errorf("expected 100 iterations, got %d", state.CurrentIteration)
	}
}

func TestStateManager_GetHistoryReturnsACopy(t *testing.T) {
	sm := NewStateManager(nil)
	sm.InitSession("sess-1")
	sm.AppendHistory("sess-1", provider.Message{Role: "user", Content: "original"})

	history := sm.GetHistory("sess-1")
	history[0].Content = "modified"

	// Original should be unchanged
	originalHistory := sm.GetHistory("sess-1")
	if originalHistory[0].Content != "original" {
		t.Error("GetHistory should return a copy, not the original slice")
	}
}
