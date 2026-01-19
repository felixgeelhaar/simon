package provider

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestOpenAIProvider(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"choices": [{"message": {"content": "hello", "role": "assistant"}}],
			"usage": {"prompt_tokens": 10, "completion_tokens": 5, "total_tokens": 15}
		}`))
	}))
	defer server.Close()

	p, _ := NewOpenAIProvider("test-key", server.URL, "gpt-4")
	if p.Name() != "openai" {
		t.Errorf("Expected 'openai', got '%s'", p.Name())
	}

	resp, err := p.Chat(context.Background(), []Message{{Role: "user", Content: "hi"}})
	if err != nil {
		t.Fatalf("Chat failed: %v", err)
	}
	if resp.Content != "hello" {
		t.Errorf("Expected 'hello', got '%s'", resp.Content)
	}
}

func TestOllamaProvider(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// Correct mock for /api/chat
		w.Write([]byte(`{"message": {"content": "hi from ollama"}, "done": true, "eval_count": 10, "prompt_eval_count": 5}`))
	}))
	defer server.Close()

	os.Setenv("OLLAMA_HOST", server.URL)
	defer os.Unsetenv("OLLAMA_HOST")

	p, _ := NewOllamaProvider("llama3")
	if p.Name() != "ollama" {
		t.Errorf("Expected 'ollama', got '%s'", p.Name())
	}

	resp, err := p.Chat(context.Background(), []Message{{Role: "user", Content: "hi"}})
	if err != nil {
		t.Fatalf("Chat failed: %v", err)
	}
	if resp.Content != "hi from ollama" {
		t.Errorf("Expected 'hi from ollama', got '%s'", resp.Content)
	}
}

func TestAnthropicProvider(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"id": "msg_123",
			"content": [{"type": "text", "text": "hello from claude"}],
			"usage": {"input_tokens": 5, "output_tokens": 5}
		}`))
	}))
	defer server.Close()

	p, _ := NewAnthropicProvider("test-key", "claude-3")
	p.SetBaseURL(server.URL)
	if p.Name() != "anthropic" {
		t.Errorf("Expected 'anthropic', got '%s'", p.Name())
	}
	
	resp, err := p.Chat(context.Background(), []Message{{Role: "user", Content: "hi"}})
	if err != nil {
		t.Fatalf("Chat failed: %v", err)
	}
	if resp.Content != "hello from claude" {
		t.Errorf("Expected 'hello from claude', got '%s'", resp.Content)
	}
}

func TestGeminiProvider_Name(t *testing.T) {
	// genai.NewClient might not connect immediately, allowing us to test Name()
	// providing a key to pass the check
	p, err := NewGeminiProvider("fake-key", "gemini-pro")
	if err != nil {
		t.Logf("Skipping Gemini Name test due to client init error: %v", err)
		return
	}
	if p.Name() != "gemini" {
		t.Errorf("Expected 'gemini', got '%s'", p.Name())
	}
}

func TestStubProvider_Timeout(t *testing.T) {
	p := NewStubProvider()
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately
	_, err := p.Chat(ctx, []Message{{Content: "hi"}})
	if err == nil {
		t.Error("Expected error on canceled context")
	}
}


func TestOpenAIProvider_Init(t *testing.T) {
	_, err := NewOpenAIProvider("", "", "")
	if err == nil {
		t.Error("Expected error for empty key")
	}
}

func TestStubProvider(t *testing.T) {
	p := NewStubProvider()
	if p.Name() != "stub" {
		t.Errorf("Expected 'stub', got '%s'", p.Name())
	}
	resp, err := p.Chat(context.Background(), []Message{{Role: "user", Content: "hi"}})
	if err != nil {
		t.Fatalf("Chat failed: %v", err)
	}
	if resp.Content == "" {
		t.Error("Expected content")
	}
}

func TestAnthropicProvider_ToolCalls(t *testing.T) {

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		w.Header().Set("Content-Type", "application/json")

		w.Write([]byte(`{

			"id": "msg_123",

			"content": [

				{"type": "text", "text": "I will run a command"},

				{"type": "tool_use", "id": "tc_1", "name": "run_shell", "input": {"cmd": "whoami"}}

			],

			"usage": {"input_tokens": 5, "output_tokens": 10}

		}`))

	}))

	defer server.Close()



	p, _ := NewAnthropicProvider("test-key", "claude-3")

	p.SetBaseURL(server.URL)

	

	resp, err := p.Chat(context.Background(), []Message{{Role: "user", Content: "run whoami"}})

	if err != nil {

		t.Fatalf("Chat failed: %v", err)

	}

	if len(resp.ToolCalls) != 1 {

		t.Errorf("Expected 1 tool call, got %d", len(resp.ToolCalls))

	}

	if resp.ToolCalls[0].Name != "run_shell" {

		t.Errorf("Expected 'run_shell', got '%s'", resp.ToolCalls[0].Name)

	}

}



func TestProvider_Errors(t *testing.T) {

	t.Run("OpenAI Error", func(t *testing.T) {

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			w.WriteHeader(500)

		}))

		defer server.Close()

		p, _ := NewOpenAIProvider("key", server.URL, "")

		_, err := p.Chat(context.Background(), []Message{{Content: "hi"}})

		if err == nil {

			t.Error("Expected error")

		}

	})



		t.Run("Anthropic Error", func(t *testing.T) {



			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {



				w.WriteHeader(401)



			}))



			defer server.Close()



			p, _ := NewAnthropicProvider("key", "")



			p.SetBaseURL(server.URL)



			_, err := p.Chat(context.Background(), []Message{{Content: "hi"}})



			if err == nil {



				t.Error("Expected error")



			}



		})



	}



	


