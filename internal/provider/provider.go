package provider

import (
	"context"
)

// Message represents a chat message.
type Message struct {
	Role       string     `json:"role"`
	Content    string     `json:"content"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"` // For tool results
}

// Response represents the output from the model.
type Response struct {
	Content      string `json:"content"`
	ToolCalls    []ToolCall `json:"tool_calls,omitempty"`
	Usage        Usage  `json:"usage"`
}

type ToolCall struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Args string `json:"args"`
}

type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// Provider defines the interface for AI model interactions.
type Provider interface {
	// Chat sends a list of messages to the model and returns a response.
	Chat(ctx context.Context, messages []Message) (*Response, error)
	
	// Embed generates a vector embedding for the given text.
	Embed(ctx context.Context, text string) ([]float32, error)
	
	// Name returns the provider identifier (e.g., "mock", "openai").
	Name() string
}
