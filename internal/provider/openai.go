package provider

import (
	"context"
	"errors"
	"fmt"

	openai "github.com/sashabaranov/go-openai"
)

type OpenAIProvider struct {
	client *openai.Client
	model  string
}

func NewOpenAIProvider(apiKey, baseURL, model string) (*OpenAIProvider, error) {
	if apiKey == "" {
		return nil, errors.New("API key is required")
	}

	config := openai.DefaultConfig(apiKey)
	if baseURL != "" {
		config.BaseURL = baseURL
	}

	client := openai.NewClientWithConfig(config)
	if model == "" {
		model = openai.GPT4TurboPreview
	}

	return &OpenAIProvider{
		client: client,
		model:  model,
	}, nil
}

func (p *OpenAIProvider) Name() string {
	return "openai"
}

func (p *OpenAIProvider) Chat(ctx context.Context, messages []Message) (*Response, error) {
	reqMsgs := make([]openai.ChatCompletionMessage, len(messages))
	for i, m := range messages {
		msg := openai.ChatCompletionMessage{
			Role:    m.Role,
			Content: m.Content,
		}
		
		if len(m.ToolCalls) > 0 {
			for _, tc := range m.ToolCalls {
				msg.ToolCalls = append(msg.ToolCalls, openai.ToolCall{
					ID:   tc.ID,
					Type: openai.ToolTypeFunction,
					Function: openai.FunctionCall{
						Name:      tc.Name,
						Arguments: tc.Args,
					},
				})
			}
		}
		
		if m.ToolCallID != "" {
			msg.ToolCallID = m.ToolCallID
		}
		
		reqMsgs[i] = msg
	}

	// Define tools
	tools := []openai.Tool{
		{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        "run_shell",
				Description: "Execute a shell command",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"cmd": map[string]interface{}{
							"type":        "string",
							"description": "The command to run",
						},
						"dir": map[string]interface{}{
							"type":        "string",
							"description": "The directory to run the command in",
						},
					},
					"required": []string{"cmd"},
				},
			},
		},
	}

	resp, err := p.client.CreateChatCompletion(
		ctx,
		openai.ChatCompletionRequest{
			Model:    p.model,
			Messages: reqMsgs,
			Tools:    tools,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("openai completion failed: %w", err)
	}

	choice := resp.Choices[0]
	
	result := &Response{
		Content: choice.Message.Content,
		Usage: Usage{
			PromptTokens:     resp.Usage.PromptTokens,
			CompletionTokens: resp.Usage.CompletionTokens,
			TotalTokens:      resp.Usage.TotalTokens,
		},
	}

	if len(choice.Message.ToolCalls) > 0 {
		for _, tc := range choice.Message.ToolCalls {
			result.ToolCalls = append(result.ToolCalls, ToolCall{
				ID:   tc.ID,
				Name: tc.Function.Name,
				Args: tc.Function.Arguments,
			})
		}
	}

	return result, nil
}

func (p *OpenAIProvider) Embed(ctx context.Context, text string) ([]float32, error) {
	resp, err := p.client.CreateEmbeddings(
		ctx,
		openai.EmbeddingRequest{
			Input: []string{text},
			Model: openai.SmallEmbedding3,
		},
	)
	if err != nil {
		return nil, err
	}
	if len(resp.Data) == 0 {
		return nil, fmt.Errorf("no embedding returned")
	}
	return resp.Data[0].Embedding, nil
}