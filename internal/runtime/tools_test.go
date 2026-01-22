package runtime

import (
	"context"
	"errors"
	"testing"

	"github.com/felixgeelhaar/simon/internal/provider"
)

func TestNewToolRegistry(t *testing.T) {
	tr := NewToolRegistry()
	if tr == nil {
		t.Fatal("expected non-nil ToolRegistry")
	}
	if tr.tools == nil {
		t.Fatal("expected non-nil tools map")
	}
	if tr.executors == nil {
		t.Fatal("expected non-nil executors map")
	}
}

func TestToolRegistry_Register(t *testing.T) {
	tr := NewToolRegistry()

	tool := ToolDefinition{
		Name:        "test_tool",
		Description: "A test tool",
		Parameters:  map[string]interface{}{"type": "object"},
	}

	executor := func(ctx context.Context, sessionID string, call provider.ToolCall) (string, error) {
		return "executed", nil
	}

	err := tr.Register(tool, executor)
	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}

	// Try to register again - should fail
	err = tr.Register(tool, executor)
	if err == nil {
		t.Error("expected error when registering duplicate tool")
	}
}

func TestToolRegistry_Unregister(t *testing.T) {
	tr := NewToolRegistry()

	tool := ToolDefinition{Name: "test_tool"}
	tr.Register(tool, nil)

	if !tr.HasTool("test_tool") {
		t.Error("tool should exist before unregister")
	}

	tr.Unregister("test_tool")

	if tr.HasTool("test_tool") {
		t.Error("tool should not exist after unregister")
	}
}

func TestToolRegistry_Get(t *testing.T) {
	tr := NewToolRegistry()

	tool := ToolDefinition{
		Name:        "test_tool",
		Description: "A test tool",
	}
	tr.Register(tool, nil)

	retrieved, ok := tr.Get("test_tool")
	if !ok {
		t.Fatal("expected tool to be found")
	}
	if retrieved.Name != "test_tool" {
		t.Errorf("expected name 'test_tool', got %q", retrieved.Name)
	}
	if retrieved.Description != "A test tool" {
		t.Errorf("expected description 'A test tool', got %q", retrieved.Description)
	}

	_, ok = tr.Get("nonexistent")
	if ok {
		t.Error("expected tool not to be found")
	}
}

func TestToolRegistry_List(t *testing.T) {
	tr := NewToolRegistry()

	tr.Register(ToolDefinition{Name: "tool1"}, nil)
	tr.Register(ToolDefinition{Name: "tool2"}, nil)
	tr.Register(ToolDefinition{Name: "tool3"}, nil)

	tools := tr.List()
	if len(tools) != 3 {
		t.Errorf("expected 3 tools, got %d", len(tools))
	}
}

func TestToolRegistry_Execute(t *testing.T) {
	tr := NewToolRegistry()

	tool := ToolDefinition{Name: "echo"}
	executor := func(ctx context.Context, sessionID string, call provider.ToolCall) (string, error) {
		return "output: " + call.Args, nil
	}
	tr.Register(tool, executor)

	result, err := tr.Execute(context.Background(), "sess-1", provider.ToolCall{
		Name: "echo",
		Args: "hello",
	})

	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}
	if result != "output: hello" {
		t.Errorf("expected 'output: hello', got %q", result)
	}
}

func TestToolRegistry_ExecuteUnknownTool(t *testing.T) {
	tr := NewToolRegistry()

	_, err := tr.Execute(context.Background(), "sess-1", provider.ToolCall{
		Name: "unknown",
	})

	if err == nil {
		t.Error("expected error for unknown tool")
	}
}

func TestToolRegistry_ExecuteWithError(t *testing.T) {
	tr := NewToolRegistry()

	tool := ToolDefinition{Name: "failing"}
	executor := func(ctx context.Context, sessionID string, call provider.ToolCall) (string, error) {
		return "", errors.New("execution failed")
	}
	tr.Register(tool, executor)

	_, err := tr.Execute(context.Background(), "sess-1", provider.ToolCall{
		Name: "failing",
	})

	if err == nil {
		t.Error("expected error from failing executor")
	}
}

func TestToolRegistry_HasTool(t *testing.T) {
	tr := NewToolRegistry()
	tr.Register(ToolDefinition{Name: "exists"}, nil)

	if !tr.HasTool("exists") {
		t.Error("expected tool to exist")
	}
	if tr.HasTool("does_not_exist") {
		t.Error("expected tool not to exist")
	}
}

func TestToolRegistry_Count(t *testing.T) {
	tr := NewToolRegistry()

	if tr.Count() != 0 {
		t.Error("expected count 0")
	}

	tr.Register(ToolDefinition{Name: "tool1"}, nil)
	if tr.Count() != 1 {
		t.Errorf("expected count 1, got %d", tr.Count())
	}

	tr.Register(ToolDefinition{Name: "tool2"}, nil)
	if tr.Count() != 2 {
		t.Errorf("expected count 2, got %d", tr.Count())
	}
}

func TestToolRegistry_GetToolsForProvider(t *testing.T) {
	tr := NewToolRegistry()

	tr.Register(ToolDefinition{
		Name:        "run_shell",
		Description: "Execute shell command",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"cmd": map[string]interface{}{"type": "string"},
			},
		},
	}, nil)

	tools := tr.GetToolsForProvider()
	if len(tools) != 1 {
		t.Errorf("expected 1 tool, got %d", len(tools))
	}

	tool := tools[0]
	if tool["type"] != "function" {
		t.Errorf("expected type 'function', got %v", tool["type"])
	}

	fn, ok := tool["function"].(map[string]interface{})
	if !ok {
		t.Fatal("expected function to be a map")
	}
	if fn["name"] != "run_shell" {
		t.Errorf("expected name 'run_shell', got %v", fn["name"])
	}
}

func TestToolRegistry_ExecuteWithContext(t *testing.T) {
	tr := NewToolRegistry()

	executed := false
	tool := ToolDefinition{Name: "context_aware"}
	executor := func(ctx context.Context, sessionID string, call provider.ToolCall) (string, error) {
		executed = true
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		default:
			return "done", nil
		}
	}
	tr.Register(tool, executor)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	result, err := tr.Execute(ctx, "sess-1", provider.ToolCall{Name: "context_aware"})

	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}
	if !executed {
		t.Error("executor was not called")
	}
	if result != "done" {
		t.Errorf("expected 'done', got %q", result)
	}
}
