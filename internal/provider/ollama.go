package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"

	"github.com/ollama/ollama/api"
)

type OllamaProvider struct {
	client *api.Client
	model  string
}

func NewOllamaProvider(model string) (*OllamaProvider, error) {
	if model == "" {
		model = "llama3.2"
	}
	
	baseURL := "http://localhost:11434"
	if envURL := os.Getenv("OLLAMA_HOST"); envURL != "" {
		baseURL = envURL
	}
	uri, _ := url.Parse(baseURL)
	client := api.NewClient(uri, http.DefaultClient)

	return &OllamaProvider{
		client: client,
		model:  model,
	}, nil
}

func (p *OllamaProvider) Name() string {
	return "ollama"
}

func (p *OllamaProvider) Chat(ctx context.Context, messages []Message) (*Response, error) {
	var apiMsgs []api.Message
	for _, m := range messages {
		apiMsgs = append(apiMsgs, api.Message{
			Role:    m.Role,
			Content: m.Content,
		})
	}

	props := api.NewToolPropertiesMap()
	props.Set("cmd", api.ToolProperty{
		Type:        api.PropertyType{"string"},
		Description: "The command to run",
	})
	props.Set("dir", api.ToolProperty{
		Type:        api.PropertyType{"string"},
		Description: "The directory to run the command in",
	})

	tools := []api.Tool{
		{
			Type: "function",
			Function: api.ToolFunction{
				Name:        "run_shell",
				Description: "Execute a shell command",
				Parameters: api.ToolFunctionParameters{
					Type:       "object",
					Properties: props,
					Required:   []string{"cmd"},
				},
			},
		},
	}

	req := &api.ChatRequest{
		Model:    p.model,
		Messages: apiMsgs,
		Stream:   new(bool), // false
		Tools:    tools,
	}

	var respContent string
	var totalTokens int
	var toolCalls []ToolCall

	err := p.client.Chat(ctx, req, func(resp api.ChatResponse) error {
		respContent += resp.Message.Content
		if resp.Done {
			totalTokens = resp.EvalCount + resp.PromptEvalCount
		}
		
		for _, tc := range resp.Message.ToolCalls {
			argsBytes, _ := json.Marshal(tc.Function.Arguments)
			toolCalls = append(toolCalls, ToolCall{
				ID:   "call_" + tc.Function.Name,
				Name: tc.Function.Name,
				Args: string(argsBytes),
			})
		}
		
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("ollama chat failed: %w", err)
	}

	return &Response{
		Content:   respContent,
		ToolCalls: toolCalls,
		Usage:     usageFromTokens(totalTokens),
	}, nil
}

func usageFromTokens(total int) Usage {
	return Usage{
		TotalTokens:      total,
		PromptTokens:     0,
		CompletionTokens: total,
	}
}

func (p *OllamaProvider) Embed(ctx context.Context, text string) ([]float32, error) {
	req := &api.EmbeddingRequest{
		Model:  p.model,
		Prompt: text,
	}
	resp, err := p.client.Embeddings(ctx, req)
	if err != nil {
		return nil, err
	}
	vec := make([]float32, len(resp.Embedding))
	for i, v := range resp.Embedding {
		vec[i] = float32(v)
	}
	return vec, nil
}