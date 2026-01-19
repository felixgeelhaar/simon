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
				Content: "Analyzing goal and initializing governance guard...",
				Usage:   Usage{PromptTokens: 100, CompletionTokens: 20, TotalTokens: 120},
			},
			{
				Content: "Searching memory for relevant past experiences...",
				Usage:   Usage{PromptTokens: 150, CompletionTokens: 25, TotalTokens: 175},
			},
			{
				Content: "Checking environment health...",
				ToolCalls: []ToolCall{
					{ID: "call_1", Name: "run_shell", Args: `{"cmd": "ls -la"}`},
				},
				Usage: Usage{PromptTokens: 200, CompletionTokens: 30, TotalTokens: 230},
			},
			{
				Content: "Executing primary implementation...",
				ToolCalls: []ToolCall{
					{ID: "call_2", Name: "run_shell", Args: `{"cmd": "echo 'wizardly output' > demo_task.yaml"}`},
				},
				Usage: Usage{PromptTokens: 250, CompletionTokens: 40, TotalTokens: 290},
			},
			{
				Content: "Verifying evidence and finalizing session...",
				Usage:   Usage{PromptTokens: 300, CompletionTokens: 20, TotalTokens: 320},
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
