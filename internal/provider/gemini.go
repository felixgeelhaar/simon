package provider

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
)

type GeminiProvider struct {
	client *genai.Client
	model  string
}

func NewGeminiProvider(apiKey, model string) (*GeminiProvider, error) {
	if apiKey == "" {
		return nil, errors.New("API key is required")
	}

	ctx := context.Background()
	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		return nil, fmt.Errorf("failed to create gemini client: %w", err)
	}

	if model == "" {
		model = "gemini-1.5-pro-latest"
	}

	return &GeminiProvider{
		client: client,
		model:  model,
	}, nil
}

func (p *GeminiProvider) Name() string {
	return "gemini"
}

func (p *GeminiProvider) Chat(ctx context.Context, messages []Message) (*Response, error) {
	geminiModel := p.client.GenerativeModel(p.model)
	
	geminiModel.Tools = []*genai.Tool{
		{
			FunctionDeclarations: []*genai.FunctionDeclaration{
				{
					Name:        "run_shell",
					Description: "Execute a shell command",
					Parameters: &genai.Schema{
						Type: genai.TypeObject,
						Properties: map[string]*genai.Schema{
							"cmd": {
								Type:        genai.TypeString,
								Description: "The command to run",
							},
							"dir": {
								Type:        genai.TypeString,
								Description: "The directory to run the command in",
							},
						},
						Required: []string{"cmd"},
					},
				},
			},
		},
	}

	cs := geminiModel.StartChat()
	
	var history []*genai.Content
	if len(messages) > 0 {
		for _, m := range messages[:len(messages)-1] {
			role := "user"
			if m.Role == "assistant" {
				role = "model"
			}
			
			content := &genai.Content{
				Role: role,
			}

			if m.ToolCallID != "" {
				content.Role = "user"
				content.Parts = append(content.Parts, genai.FunctionResponse{
					Name: m.ToolCallID,
					Response: map[string]any{"result": m.Content},
				})
			} else {
				if m.Content != "" {
					content.Parts = append(content.Parts, genai.Text(m.Content))
				}
				for _, tc := range m.ToolCalls {
					var args map[string]any
					json.Unmarshal([]byte(tc.Args), &args)
					content.Parts = append(content.Parts, genai.FunctionCall{
						Name: tc.Name,
						Args: args,
					})
				}
			}
			history = append(history, content)
		}
		cs.History = history
	}

	lastMsg := messages[len(messages)-1]
	resp, err := cs.SendMessage(ctx, genai.Text(lastMsg.Content))
	if err != nil {
		return nil, fmt.Errorf("gemini completion failed: %w", err)
	}

	if len(resp.Candidates) == 0 {
		return nil, fmt.Errorf("no candidates returned")
	}
	cand := resp.Candidates[0]
	
	var contentStr string
	var toolCalls []ToolCall

	for _, part := range cand.Content.Parts {
		switch v := part.(type) {
		case genai.Text:
			contentStr += string(v)
		case genai.FunctionCall:
			argsBytes, _ := json.Marshal(v.Args)
			toolCalls = append(toolCalls, ToolCall{
				ID:   v.Name, 
				Name: v.Name,
				Args: string(argsBytes),
			})
		}
	}

	usage := Usage{
		PromptTokens:     int(resp.UsageMetadata.PromptTokenCount),
		CompletionTokens: int(resp.UsageMetadata.CandidatesTokenCount),
		TotalTokens:      int(resp.UsageMetadata.TotalTokenCount),
	}

	return &Response{
		Content:   contentStr,
		ToolCalls: toolCalls,
		Usage:     usage,
	}, nil
}

func (p *GeminiProvider) Embed(ctx context.Context, text string) ([]float32, error) {
	em := p.client.EmbeddingModel("text-embedding-004")
	res, err := em.EmbedContent(ctx, genai.Text(text))
	if err != nil {
		return nil, err
	}
	if res.Embedding == nil {
		return nil, fmt.Errorf("no embedding returned")
	}
	return res.Embedding.Values, nil
}
