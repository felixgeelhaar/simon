package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

// OllamaProvider talks to an Ollama server over its documented HTTP API
// (/api/chat and /api/embeddings). It deliberately does not import the
// github.com/ollama/ollama module, which bundles the whole server.
type OllamaProvider struct {
	baseURL string
	client  *http.Client
	model   string
}

func NewOllamaProvider(model string) (*OllamaProvider, error) {
	if model == "" {
		model = "llama3.2"
	}

	baseURL := "http://localhost:11434"
	if envURL := os.Getenv("OLLAMA_HOST"); envURL != "" {
		baseURL = envURL
	}
	if !strings.Contains(baseURL, "://") {
		baseURL = "http://" + baseURL
	}

	return &OllamaProvider{
		baseURL: strings.TrimRight(baseURL, "/"),
		client:  http.DefaultClient,
		model:   model,
	}, nil
}

func (p *OllamaProvider) Name() string {
	return "ollama"
}

type ollamaMessage struct {
	Role      string           `json:"role"`
	Content   string           `json:"content"`
	ToolCalls []ollamaToolCall `json:"tool_calls,omitempty"`
}

type ollamaToolCall struct {
	Function struct {
		Name      string          `json:"name"`
		Arguments json.RawMessage `json:"arguments"`
	} `json:"function"`
}

type ollamaChatRequest struct {
	Model    string          `json:"model"`
	Messages []ollamaMessage `json:"messages"`
	Stream   bool            `json:"stream"`
	Tools    []any           `json:"tools,omitempty"`
}

type ollamaChatResponse struct {
	Message         ollamaMessage `json:"message"`
	Done            bool          `json:"done"`
	EvalCount       int           `json:"eval_count"`
	PromptEvalCount int           `json:"prompt_eval_count"`
}

func (p *OllamaProvider) post(ctx context.Context, path string, in, out any) error {
	body, err := json.Marshal(in)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+path, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		msg, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("status %d: %s", resp.StatusCode, strings.TrimSpace(string(msg)))
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

func (p *OllamaProvider) Chat(ctx context.Context, messages []Message) (*Response, error) {
	apiMsgs := make([]ollamaMessage, 0, len(messages))
	for _, m := range messages {
		apiMsgs = append(apiMsgs, ollamaMessage{Role: m.Role, Content: m.Content})
	}

	tools := []any{
		map[string]any{
			"type": "function",
			"function": map[string]any{
				"name":        "run_shell",
				"description": "Execute a shell command",
				"parameters": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"cmd": map[string]any{"type": "string", "description": "The command to run"},
						"dir": map[string]any{"type": "string", "description": "The directory to run the command in"},
					},
					"required": []string{"cmd"},
				},
			},
		},
	}

	var resp ollamaChatResponse
	err := p.post(ctx, "/api/chat", ollamaChatRequest{
		Model:    p.model,
		Messages: apiMsgs,
		Stream:   false,
		Tools:    tools,
	}, &resp)
	if err != nil {
		return nil, fmt.Errorf("ollama chat failed: %w", err)
	}

	var toolCalls []ToolCall
	for _, tc := range resp.Message.ToolCalls {
		toolCalls = append(toolCalls, ToolCall{
			ID:   "call_" + tc.Function.Name,
			Name: tc.Function.Name,
			Args: string(tc.Function.Arguments),
		})
	}

	totalTokens := 0
	if resp.Done {
		totalTokens = resp.EvalCount + resp.PromptEvalCount
	}

	return &Response{
		Content:   resp.Message.Content,
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
	var resp struct {
		Embedding []float64 `json:"embedding"`
	}
	if err := p.post(ctx, "/api/embeddings", map[string]string{"model": p.model, "prompt": text}, &resp); err != nil {
		return nil, err
	}
	vec := make([]float32, len(resp.Embedding))
	for i, v := range resp.Embedding {
		vec[i] = float32(v)
	}
	return vec, nil
}
