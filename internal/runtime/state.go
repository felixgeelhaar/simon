package runtime

import (
	"sync"
	"time"

	"github.com/felixgeelhaar/simon/internal/provider"
	"github.com/felixgeelhaar/simon/internal/store"
)

// SessionState represents the current state of a session execution.
type SessionState struct {
	SessionID        string
	CurrentIteration int
	TotalPromptTokens  int
	TotalOutputTokens  int
	History          []provider.Message
	Status           string
	StartedAt        time.Time
	LastUpdatedAt    time.Time
}

// StateManager handles session state tracking and persistence.
// It provides thread-safe access to session state and manages
// the lifecycle of session data.
type StateManager struct {
	mu       sync.RWMutex
	store    store.Storage
	sessions map[string]*SessionState
}

// NewStateManager creates a new state manager.
func NewStateManager(s store.Storage) *StateManager {
	return &StateManager{
		store:    s,
		sessions: make(map[string]*SessionState),
	}
}

// InitSession initializes a new session state.
func (sm *StateManager) InitSession(sessionID string) *SessionState {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	state := &SessionState{
		SessionID:        sessionID,
		CurrentIteration: 0,
		History:          make([]provider.Message, 0),
		Status:           "initialized",
		StartedAt:        time.Now(),
		LastUpdatedAt:    time.Now(),
	}

	sm.sessions[sessionID] = state
	return state
}

// GetState returns the current state for a session.
func (sm *StateManager) GetState(sessionID string) *SessionState {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.sessions[sessionID]
}

// IncrementIteration increments the iteration counter and returns the new value.
func (sm *StateManager) IncrementIteration(sessionID string) int {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if state, ok := sm.sessions[sessionID]; ok {
		state.CurrentIteration++
		state.LastUpdatedAt = time.Now()
		return state.CurrentIteration
	}
	return 0
}

// AddTokenUsage adds token usage to the session totals.
func (sm *StateManager) AddTokenUsage(sessionID string, promptTokens, outputTokens int) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if state, ok := sm.sessions[sessionID]; ok {
		state.TotalPromptTokens += promptTokens
		state.TotalOutputTokens += outputTokens
		state.LastUpdatedAt = time.Now()
	}
}

// GetTokenUsage returns the current token usage for a session.
func (sm *StateManager) GetTokenUsage(sessionID string) (promptTokens, outputTokens int) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	if state, ok := sm.sessions[sessionID]; ok {
		return state.TotalPromptTokens, state.TotalOutputTokens
	}
	return 0, 0
}

// AppendHistory adds a message to the session history.
func (sm *StateManager) AppendHistory(sessionID string, msg provider.Message) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if state, ok := sm.sessions[sessionID]; ok {
		state.History = append(state.History, msg)
		state.LastUpdatedAt = time.Now()
	}
}

// GetHistory returns a copy of the session history.
func (sm *StateManager) GetHistory(sessionID string) []provider.Message {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	if state, ok := sm.sessions[sessionID]; ok {
		history := make([]provider.Message, len(state.History))
		copy(history, state.History)
		return history
	}
	return nil
}

// ReplaceHistory replaces the entire session history.
func (sm *StateManager) ReplaceHistory(sessionID string, history []provider.Message) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if state, ok := sm.sessions[sessionID]; ok {
		state.History = history
		state.LastUpdatedAt = time.Now()
	}
}

// SetStatus updates the session status.
func (sm *StateManager) SetStatus(sessionID, status string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if state, ok := sm.sessions[sessionID]; ok {
		state.Status = status
		state.LastUpdatedAt = time.Now()
	}
}

// GetStatus returns the current session status.
func (sm *StateManager) GetStatus(sessionID string) string {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	if state, ok := sm.sessions[sessionID]; ok {
		return state.Status
	}
	return ""
}

// PersistSession saves the session state to the store.
func (sm *StateManager) PersistSession(sessionID string) error {
	sm.mu.RLock()
	state, ok := sm.sessions[sessionID]
	sm.mu.RUnlock()

	if !ok {
		return nil
	}

	session, err := sm.store.GetSession(sessionID)
	if err != nil {
		return err
	}

	session.Status = state.Status
	session.UpdatedAt = state.LastUpdatedAt

	return sm.store.UpdateSession(session)
}

// CleanupSession removes the session state from memory.
func (sm *StateManager) CleanupSession(sessionID string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	delete(sm.sessions, sessionID)
}

// HistoryLength returns the number of messages in the session history.
func (sm *StateManager) HistoryLength(sessionID string) int {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	if state, ok := sm.sessions[sessionID]; ok {
		return len(state.History)
	}
	return 0
}
