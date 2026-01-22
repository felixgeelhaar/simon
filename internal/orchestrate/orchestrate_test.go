package orchestrate

import (
	"context"
	"testing"

	"github.com/felixgeelhaar/simon/internal/provider"
)

// mockProvider implements provider.Provider for testing
type mockProvider struct {
	name string
}

func (m *mockProvider) Chat(ctx context.Context, messages []provider.Message) (*provider.Response, error) {
	return &provider.Response{
		Content: "mock response from " + m.name,
		Usage:   provider.Usage{TotalTokens: 10},
	}, nil
}

func (m *mockProvider) Embed(ctx context.Context, text string) ([]float32, error) {
	return []float32{0.1, 0.2, 0.3}, nil
}

func (m *mockProvider) Name() string {
	return m.name
}

func TestAgentType_Constants(t *testing.T) {
	testCases := []struct {
		agentType AgentType
		expected  string
	}{
		{AgentTypePlanner, "planner"},
		{AgentTypeExecutor, "executor"},
		{AgentTypeReviewer, "reviewer"},
	}

	for _, tc := range testCases {
		if string(tc.agentType) != tc.expected {
			t.Errorf("expected %q, got %q", tc.expected, tc.agentType)
		}
	}
}

func TestNew(t *testing.T) {
	planner := &mockProvider{name: "planner"}
	executor := &mockProvider{name: "executor"}
	reviewer := &mockProvider{name: "reviewer"}

	orch := New(planner, executor, reviewer)

	if orch == nil {
		t.Fatal("expected non-nil orchestrator")
	}
	if orch.Planner != planner {
		t.Error("Planner not set correctly")
	}
	if orch.Executor != executor {
		t.Error("Executor not set correctly")
	}
	if orch.Reviewer != reviewer {
		t.Error("Reviewer not set correctly")
	}
}

func TestMultiAgentOrchestrator_Execute(t *testing.T) {
	planner := &mockProvider{name: "planner"}
	executor := &mockProvider{name: "executor"}
	reviewer := &mockProvider{name: "reviewer"}

	orch := New(planner, executor, reviewer)
	ctx := context.Background()

	err := orch.Execute(ctx, "test goal")
	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}
}

func TestMultiAgentOrchestrator_ExecuteWithContext(t *testing.T) {
	planner := &mockProvider{name: "planner"}
	executor := &mockProvider{name: "executor"}
	reviewer := &mockProvider{name: "reviewer"}

	orch := New(planner, executor, reviewer)

	// Test with cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// Execute should still work (current implementation doesn't check context)
	err := orch.Execute(ctx, "test goal")
	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}
}

func TestMultiAgentOrchestrator_NilProviders(t *testing.T) {
	// Test that the orchestrator can be created with nil providers
	// This tests the struct behavior, not recommended for production
	orch := New(nil, nil, nil)

	if orch == nil {
		t.Fatal("expected non-nil orchestrator")
	}
	if orch.Planner != nil {
		t.Error("expected nil Planner")
	}
}
