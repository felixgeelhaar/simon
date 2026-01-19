package provider

import (
	"context"
	"time"
)

// StubProvider is a simple provider for testing.
type StubProvider struct {
	Responses []Response
}

func NewStubProvider() *StubProvider {
	return &StubProvider{
		Responses: []Response{
			{
				Content: "I will start the task.",
				Usage: Usage{PromptTokens: 100, CompletionTokens: 20, TotalTokens: 120},
			},
			{
				Content: "Checking system status...",
				ToolCalls: []ToolCall{
					{ID: "call_1", Name: "run_shell", Args: `{"cmd": "echo hello"}`},
				},
				Usage: Usage{PromptTokens: 150, CompletionTokens: 30, TotalTokens: 180},
			},
			{
				Content: "Task complete.",
				Usage: Usage{PromptTokens: 200, CompletionTokens: 10, TotalTokens: 210},
			},
		},
	}
}

func (m *StubProvider) Chat(ctx context.Context, messages []Message) (*Response, error) {
	// Simulate latency
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(10 * time.Millisecond):
	}

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