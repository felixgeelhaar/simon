package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
)

type AnthropicProvider struct {
	apiKey  string
	model   string
	baseURL string
	client  *http.Client
}

func NewAnthropicProvider(apiKey, model string) (*AnthropicProvider, error) {
	if apiKey == "" {
		return nil, errors.New("API key is required")
	}

	if model == "" {
		model = "claude-3-opus-20240229"
	}

	return &AnthropicProvider{
		apiKey:  apiKey,
		model:   model,
		baseURL: "https://api.anthropic.com/v1/messages",
		client:  &http.Client{},
	}, nil
}

func (p *AnthropicProvider) Name() string {
	return "anthropic"
}

// Anthropic types for request/response
type anthropicMessage struct {
	Role    string                  `json:"role"`
	Content []anthropicContentBlock `json:"content"`
}

type anthropicRequest struct {
	Model     string             `json:"model"`
	Messages  []anthropicMessage `json:"messages"`
	MaxTokens int                `json:"max_tokens"`
	Tools     []anthropicTool    `json:"tools,omitempty"`
}

type anthropicTool struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	InputSchema interface{} `json:"input_schema"`
}

type anthropicResponse struct {
	ID           string                  `json:"id"`
	Content      []anthropicContentBlock `json:"content"`
	Usage        anthropicUsage          `json:"usage"`
	StopReason   string                  `json:"stop_reason"`
	Error        *anthropicError         `json:"error,omitempty"`
}

type anthropicContentBlock struct {
	Type      string          `json:"type"`
	Text      string          `json:"text,omitempty"`
	Input     json.RawMessage `json:"input,omitempty"`      // For tool use
	Name      string          `json:"name,omitempty"`       // For tool use
	ID        string          `json:"id,omitempty"`         // For tool use
	ToolUseID string          `json:"tool_use_id,omitempty"` // For tool result
	Content   string          `json:"content,omitempty"`     // For tool result
}

type anthropicUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

type anthropicError struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

// SetBaseURL allows overriding the API endpoint (useful for tests)
func (p *AnthropicProvider) SetBaseURL(url string) {
	p.baseURL = url
}

func (p *AnthropicProvider) Chat(ctx context.Context, messages []Message) (*Response, error) {
	url := p.baseURL

	var anthropicMsgs []anthropicMessage
	for _, m := range messages {
		role := m.Role
		if role == "tool" {
			role = "user"
		}

		var content []anthropicContentBlock
		if m.ToolCallID != "" {
			content = append(content, anthropicContentBlock{
				Type:      "tool_result",
				ToolUseID: m.ToolCallID,
				Content:   m.Content,
			})
		} else {
			if m.Content != "" {
				content = append(content, anthropicContentBlock{
					Type: "text",
					Text: m.Content,
				})
			}
			for _, tc := range m.ToolCalls {
				content = append(content, anthropicContentBlock{
					Type: "tool_use",
					ID:   tc.ID,
					Name: tc.Name,
					Input: json.RawMessage(tc.Args),
				})
			}
		}

		anthropicMsgs = append(anthropicMsgs, anthropicMessage{
			Role:    role,
			Content: content,
		})
	}

	// Tool definition for run_shell
	tools := []anthropicTool{
		{
			Name:        "run_shell",
			Description: "Execute a shell command",
			InputSchema: map[string]interface{}{
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
	}

	reqBody := anthropicRequest{
		Model:     p.model,
		Messages:  anthropicMsgs,
		MaxTokens: 4096,
		Tools:     tools,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, err
	}

	req.Header.Set("x-api-key", p.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")
	req.Header.Set("content-type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("anthropic api error: %s", string(body))
	}

	var anthropicResp anthropicResponse
	if err := json.Unmarshal(body, &anthropicResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if anthropicResp.Error != nil {
		return nil, fmt.Errorf("anthropic error: %s", anthropicResp.Error.Message)
	}

	var contentStr string
	var toolCalls []ToolCall

	for _, block := range anthropicResp.Content {
		if block.Type == "text" {
			contentStr += block.Text
		} else if block.Type == "tool_use" {
			toolCalls = append(toolCalls, ToolCall{
				ID:   block.ID,
				Name: block.Name,
				Args: string(block.Input),
			})
		}
	}

	return &Response{
		Content:   contentStr,
		ToolCalls: toolCalls,
		Usage: Usage{
			PromptTokens:     anthropicResp.Usage.InputTokens,
			CompletionTokens: anthropicResp.Usage.OutputTokens,
			TotalTokens:      anthropicResp.Usage.InputTokens + anthropicResp.Usage.OutputTokens,
		},
	}, nil
}

func (p *AnthropicProvider) Embed(ctx context.Context, text string) ([]float32, error) {
	return nil, fmt.Errorf("embeddings not supported by Anthropic provider")
}