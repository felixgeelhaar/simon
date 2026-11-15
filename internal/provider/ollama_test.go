package provider

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func newTestOllama(t *testing.T, h http.HandlerFunc) *OllamaProvider {
	t.Helper()
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	t.Setenv("OLLAMA_HOST", srv.URL)
	p, err := NewOllamaProvider("test-model")
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func TestOllamaHostNormalisation(t *testing.T) {
	tests := []struct{ env, want string }{
		{"", "http://localhost:11434"},
		{"example:1234", "http://example:1234"},
		{"https://example/", "https://example"},
	}
	for _, tt := range tests {
		t.Setenv("OLLAMA_HOST", tt.env)
		p, err := NewOllamaProvider("")
		if err != nil {
			t.Fatal(err)
		}
		if p.baseURL != tt.want {
			t.Errorf("OLLAMA_HOST=%q: baseURL = %q, want %q", tt.env, p.baseURL, tt.want)
		}
		if p.model != "llama3.2" {
			t.Errorf("default model = %q", p.model)
		}
	}
}

func TestOllamaChat(t *testing.T) {
	var got ollamaChatRequest
	p := newTestOllama(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/chat" {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Errorf("decode: %v", err)
		}
		_, _ = w.Write([]byte(`{"message":{"role":"assistant","content":"hi","tool_calls":[{"function":{"name":"run_shell","arguments":{"cmd":"ls"}}}]},"done":true,"eval_count":7,"prompt_eval_count":3}`))
	})

	resp, err := p.Chat(context.Background(), []Message{{Role: "user", Content: "hello"}})
	if err != nil {
		t.Fatal(err)
	}
	if got.Model != "test-model" || got.Stream || len(got.Messages) != 1 || got.Messages[0].Content != "hello" || len(got.Tools) != 1 {
		t.Errorf("bad request body: %+v", got)
	}
	if resp.Content != "hi" {
		t.Errorf("content = %q", resp.Content)
	}
	if len(resp.ToolCalls) != 1 || resp.ToolCalls[0].Name != "run_shell" || resp.ToolCalls[0].Args != `{"cmd":"ls"}` {
		t.Errorf("tool calls = %+v", resp.ToolCalls)
	}
	if resp.Usage.TotalTokens != 10 {
		t.Errorf("total tokens = %d, want 10", resp.Usage.TotalTokens)
	}
}

func TestOllamaChatNotDoneReportsNoTokens(t *testing.T) {
	p := newTestOllama(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"message":{"role":"assistant","content":"x"},"done":false,"eval_count":5}`))
	})
	resp, err := p.Chat(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if resp.Usage.TotalTokens != 0 {
		t.Errorf("tokens = %d, want 0", resp.Usage.TotalTokens)
	}
}

func TestOllamaChatHTTPError(t *testing.T) {
	p := newTestOllama(t, func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "model not found", http.StatusNotFound)
	})
	_, err := p.Chat(context.Background(), nil)
	if err == nil || !strings.Contains(err.Error(), "404") || !strings.Contains(err.Error(), "model not found") {
		t.Errorf("err = %v", err)
	}
}

func TestOllamaChatMalformedJSON(t *testing.T) {
	p := newTestOllama(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`not json`))
	})
	if _, err := p.Chat(context.Background(), nil); err == nil {
		t.Error("expected decode error")
	}
}

func TestOllamaEmbed(t *testing.T) {
	var got map[string]string
	p := newTestOllama(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/embeddings" {
			t.Errorf("path = %s", r.URL.Path)
		}
		_ = json.NewDecoder(r.Body).Decode(&got)
		_, _ = w.Write([]byte(`{"embedding":[0.5,-1,2.25]}`))
	})
	vec, err := p.Embed(context.Background(), "some text")
	if err != nil {
		t.Fatal(err)
	}
	if got["model"] != "test-model" || got["prompt"] != "some text" {
		t.Errorf("request = %v", got)
	}
	want := []float32{0.5, -1, 2.25}
	if len(vec) != len(want) {
		t.Fatalf("vec = %v", vec)
	}
	for i := range want {
		if vec[i] != want[i] {
			t.Errorf("vec[%d] = %v, want %v", i, vec[i], want[i])
		}
	}
}

func TestOllamaEmbedHTTPError(t *testing.T) {
	p := newTestOllama(t, func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	})
	if _, err := p.Embed(context.Background(), "x"); err == nil {
		t.Error("expected error")
	}
}
