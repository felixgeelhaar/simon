package provider

import (
	"context"
	"sync"
	"time"
)

// StubProvider is a simple provider for testing.
// It is thread-safe and can be used in concurrent tests.
type StubProvider struct {
	mu        sync.Mutex
	Responses []Response
}

func NewStubProvider() *StubProvider {
	return &StubProvider{
		Responses: []Response{
			{
				Content: "I'll create a Go CLI application. First, let me check the current directory structure.",
				Usage:   Usage{PromptTokens: 100, CompletionTokens: 25, TotalTokens: 125},
			},
			{
				Content: "Checking for existing Go modules and project files...",
				ToolCalls: []ToolCall{
					{ID: "call_1", Name: "run_shell", Args: `{"cmd": "ls -la"}`},
				},
				Usage: Usage{PromptTokens: 150, CompletionTokens: 30, TotalTokens: 180},
			},
			{
				Content: "The evidence file exists. Now verifying the task specification matches our goal...",
				ToolCalls: []ToolCall{
					{ID: "call_2", Name: "run_shell", Args: `{"cmd": "cat demo_task.yaml"}`},
				},
				Usage: Usage{PromptTokens: 200, CompletionTokens: 35, TotalTokens: 235},
			},
			{
				Content: "Goal confirmed: Create hello world Go CLI. Evidence path verified. Preparing completion.",
				Usage:   Usage{PromptTokens: 250, CompletionTokens: 30, TotalTokens: 280},
			},
			{
				Content: "All evidence files confirmed. Session objectives achieved within budget constraints.",
				Usage:   Usage{PromptTokens: 300, CompletionTokens: 25, TotalTokens: 325},
			},
			{
				Content: "Task complete.",
				Usage:   Usage{PromptTokens: 350, CompletionTokens: 10, TotalTokens: 360},
			},
		},
	}
}

func (m *StubProvider) Chat(ctx context.Context, messages []Message) (*Response, error) {
	// Cinematic latency
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(1500 * time.Millisecond):
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if len(m.Responses) == 0 {
		return &Response{Content: "Task complete.", Usage: Usage{}}, nil
	}

	resp := m.Responses[0]
	m.Responses = m.Responses[1:]
	return &resp, nil
}

func (m *StubProvider) Embed(ctx context.Context, text string) ([]float32, error) {
	return []float32{0.1, 0.2, 0.3}, nil
}

func (m *StubProvider) Name() string {
	return "stub"
}
