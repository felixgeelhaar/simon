package mcp

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/felixgeelhaar/simon/internal/guard"
	"github.com/felixgeelhaar/simon/internal/provider"
	"github.com/felixgeelhaar/simon/internal/store"
)

func TestProxy_HandleToolCalls(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "mcp-test-*")
	defer os.RemoveAll(tmpDir)

	s, _ := store.NewSQLiteStore(filepath.Join(tmpDir, "db"), filepath.Join(tmpDir, "artifacts"))
	g := guard.New(guard.Policy{
		AllowedCommands: []string{"echo"},
	})
	p := NewProxy(s, g)

	session := &store.Session{ID: "sess-mcp", CreatedAt: time.Now()}
	s.CreateSession(session)

	t.Run("Allowed Command", func(t *testing.T) {
		calls := []provider.ToolCall{
			{ID: "call-1", Name: "run_shell", Args: `{"cmd": "echo hello"}`},
		}

		results, err := p.HandleToolCalls(context.Background(), "sess-mcp", calls)
		if err != nil {
			t.Fatalf("HandleToolCalls failed: %v", err)
		}

		if len(results) != 1 {
			t.Errorf("Expected 1 result, got %d", len(results))
		}
		if results[0].IsError {
			t.Errorf("Expected no error, got error: %s", results[0].Digest)
		}
	})

	t.Run("Blocked Command", func(t *testing.T) {
		calls := []provider.ToolCall{
			{ID: "call-2", Name: "run_shell", Args: `{"cmd": "rm -rf /"}`},
		}

		results, err := p.HandleToolCalls(context.Background(), "sess-mcp", calls)
		if err != nil {
			t.Fatalf("HandleToolCalls failed: %v", err)
		}

		if !results[0].IsError {
			t.Error("Expected error for blocked command")
		}
	})

	t.Run("Invalid Args", func(t *testing.T) {
		calls := []provider.ToolCall{
			{ID: "call-3", Name: "run_shell", Args: `invalid`},
		}

		results, err := p.HandleToolCalls(context.Background(), "sess-mcp", calls)
		if err != nil {
			t.Fatalf("HandleToolCalls failed: %v", err)
		}

		if !results[0].IsError {
			t.Error("Expected error for invalid args")
		}
	})

	t.Run("Unknown Tool", func(t *testing.T) {
		calls := []provider.ToolCall{
			{ID: "call-4", Name: "unknown", Args: `{}`},
		}

		results, err := p.HandleToolCalls(context.Background(), "sess-mcp", calls)
		if err != nil {
			t.Fatalf("HandleToolCalls failed: %v", err)
		}

		if !results[0].IsError {
			t.Error("Expected error for unknown tool")
		}
	})
}