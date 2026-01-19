package runtime

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/felixgeelhaar/simon/internal/coach"
	"github.com/felixgeelhaar/simon/internal/guard"
	"github.com/felixgeelhaar/simon/internal/mcp"
	"github.com/felixgeelhaar/simon/internal/observe"
	"github.com/felixgeelhaar/simon/internal/provider"
	"github.com/felixgeelhaar/simon/internal/store"
)

func TestRuntime_ExecuteSession(t *testing.T) {
	// Setup Deps
	tmpDir, _ := os.MkdirTemp("", "runtime-test-*")
	defer os.RemoveAll(tmpDir)

	s, _ := store.NewSQLiteStore(filepath.Join(tmpDir, "db"), filepath.Join(tmpDir, "artifacts"))
	g := guard.New(guard.DefaultPolicy)
	c := coach.New()
	o := observe.New(os.Stdout)
	
t.Run("Successful Run", func(t *testing.T) {
		specPath := filepath.Join(tmpDir, "spec_success.yaml")
		os.WriteFile(specPath, []byte("goal: test\ndefinition_of_done: test\nevidence: []"), 0600)

		p := provider.NewStubProvider()
		mp := mcp.NewProxy(s, g)
		r := New(s, g, c, o, p, mp)

		session := &store.Session{
			ID:        "sess-success",
			CreatedAt: time.Now(),
			Status:    "active",
			Metadata:  map[string]string{"spec": specPath},
		}
		s.CreateSession(session)

		if err := r.ExecuteSession(context.Background(), "sess-success"); err != nil {
			t.Fatalf("ExecuteSession failed: %v", err)
		}

		updated, _ := s.GetSession("sess-success")
		if updated.Status != "completed" {
			t.Errorf("Expected status 'completed', got '%s'", updated.Status)
		}
	})

	t.Run("Verification Failure then Success", func(t *testing.T) {
		evidencePath := filepath.Join(tmpDir, "evidence.txt")
		specPath := filepath.Join(tmpDir, "spec_verify.yaml")
		os.WriteFile(specPath, []byte("goal: test\ndefinition_of_done: test\nevidence: ["+evidencePath+"]"), 0600)

		// Stub provider that says "Task complete." then "Really complete"
		p := &provider.StubProvider{
			Responses: []provider.Response{
				{Content: "Task complete.", Usage: provider.Usage{TotalTokens: 10}},
				{Content: "Fixed it. Task complete.", Usage: provider.Usage{TotalTokens: 10}},
			},
		}
		gLenient := guard.New(guard.Policy{MaxIterations: 100, MaxPromptTokens: 10000, MaxOutputTokens: 10000})
		mp := mcp.NewProxy(s, gLenient)
		r := New(s, gLenient, c, o, p, mp)

		session := &store.Session{
			ID:        "sess-verify",
			CreatedAt: time.Now(),
			Status:    "active",
			Metadata:  map[string]string{"spec": specPath},
		}
		s.CreateSession(session)

		// Create evidence file after the first failure
		// The loop will:
		// 1. Call provider -> "Task complete."
		// 2. verifyEvidence -> FAIL (missing file)
		// 3. history append verification fail
		// 4. Call provider -> "Fixed it. Task complete."
		// 5. verifyEvidence -> PASS (if file exists)
		
		// Trigger file creation
		go func() {
			time.Sleep(200 * time.Millisecond)
			os.WriteFile(evidencePath, []byte("evidence"), 0600)
		}()

		if err := r.ExecuteSession(context.Background(), "sess-verify"); err != nil {
			t.Fatalf("ExecuteSession failed: %v", err)
		}
	})

	t.Run("Guard Violation", func(t *testing.T) {
		specPath := filepath.Join(tmpDir, "spec_guard.yaml")
		os.WriteFile(specPath, []byte("goal: test\nevidence: []"), 0600)

		gLimited := guard.New(guard.Policy{MaxIterations: 1})
		p := provider.NewStubProvider()
		mp := mcp.NewProxy(s, gLimited)
		r := New(s, gLimited, c, o, p, mp)

		session := &store.Session{
			ID:        "sess-guard",
			CreatedAt: time.Now(),
			Status:    "active",
			Metadata:  map[string]string{"spec": specPath},
		}
		s.CreateSession(session)

		err := r.ExecuteSession(context.Background(), "sess-guard")
		if err == nil {
			t.Error("Expected guard violation error")
		}
	})
}