package e2e

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/felixgeelhaar/simon/internal/coach"
	"github.com/felixgeelhaar/simon/internal/guard"
	"github.com/felixgeelhaar/simon/internal/mcp"
	"github.com/felixgeelhaar/simon/internal/observe"
	"github.com/felixgeelhaar/simon/internal/provider"
	"github.com/felixgeelhaar/simon/internal/runtime"
	"github.com/felixgeelhaar/simon/internal/store"
)

// SmartStub simulates a capable model (like GPT-4/Codex) solving the task
type SmartStub struct {
	iteration int
}

func (s *SmartStub) Name() string { return "smart-stub" }

func (s *SmartStub) Embed(ctx context.Context, text string) ([]float32, error) {
	return []float32{0.1, 0.2, 0.3}, nil
}

func (s *SmartStub) Chat(ctx context.Context, messages []provider.Message) (*provider.Response, error) {

	s.iteration++

	

	// Helper to make safe args

	makeArgs := func(cmd string) string {

		args := map[string]string{"cmd": cmd}

		b, _ := json.Marshal(args)

		return string(b)

	}



	// Step 1: Create Directory

	if s.iteration == 1 {

		return &provider.Response{

			Content: "I'll start by creating the directory.",

			ToolCalls: []provider.ToolCall{

				{ID: "call1", Name: "run_shell", Args: makeArgs("mkdir -p playground/hello-simon")},

			},

			Usage: provider.Usage{TotalTokens: 50},

		}, nil

	}



	// Step 2: Write go.mod

	if s.iteration == 2 {

		return &provider.Response{

			Content: "Now I'll create the go.mod file.",

			ToolCalls: []provider.ToolCall{

				{ID: "call2", Name: "run_shell", Args: makeArgs("echo 'module hello-simon\n\ngo 1.21' > playground/hello-simon/go.mod")},

			},

			Usage: provider.Usage{TotalTokens: 50},

		}, nil

	}



	// Step 3: Write Main File

	if s.iteration == 3 {

		code := `package main\n\nimport "fmt"\n\nfunc main() {\n\tfmt.Println("Hello, Simon!")\n}`

		return &provider.Response{

			Content: "Creating the main.go file.",

			ToolCalls: []provider.ToolCall{

				{ID: "call3", Name: "run_shell", Args: makeArgs("echo -e '" + code + "' > playground/hello-simon/main.go")},

			},

			Usage: provider.Usage{TotalTokens: 100},

		}, nil

	}



	// Step 4: Done
	return &provider.Response{
		Content: "Task complete. I have created the project and verified it.",
		Usage: provider.Usage{TotalTokens: 20},
	},
	nil
}

func TestE2E_SmartAgent_BuildProject(t *testing.T) {
	// Setup
	tmpDir, _ := os.MkdirTemp("", "simon-smart-e2e-*")
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "meta.db")
	artDir := filepath.Join(tmpDir, "artifacts")
	
	// Create the playground dir in tmpDir so the stub's commands work relative to it
	// But wait, the stub uses "playground/hello-simon". 
	// The test runs in a temp dir, but the stub commands execute in CWD?
	// No, Proxy executes in CWD.
	// For this test, we must ensure we run inside tmpDir or adjust stub commands.
	// Changing CWD in test is unsafe for parallel tests, but safe for this specific test function.
	
	origWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origWd)

	s, _ := store.NewSQLiteStore(dbPath, artDir)
	
	// Allow mkdir and echo
	policy := guard.DefaultPolicy
	policy.AllowedCommands = append(policy.AllowedCommands, "mkdir", "echo")
	g := guard.New(policy)
	c := coach.New()
	o := observe.New(os.Stdout)
	p := &SmartStub{}
	mp := mcp.NewProxy(s, g)
	r := runtime.New(s, g, c, o, p, mp)

	// Create Spec
	specPath := "task.yaml"
	os.WriteFile(specPath, []byte(`
goal: "Create hello-simon"
definition_of_done: "Runs"
evidence:
  - "playground/hello-simon/go.mod"
  - "playground/hello-simon/main.go"
constraints: []
`), 0600)

	session := &store.Session{
		ID:        "sess-smart",
		CreatedAt: time.Now(),
		Status:    "active",
		Metadata:  map[string]string{"spec": specPath},
	}
	s.CreateSession(session)

	// Execute
	if err := r.ExecuteSession(context.Background(), "sess-smart"); err != nil {
		// Debug: List files
		filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
			t.Logf("Found file: %s", path)
			return nil
		})
		// Debug: Read artifacts
		arts, _ := s.ListArtifacts("sess-smart")
		for _, a := range arts {
			_, content, _ := s.GetArtifact(a.ID)
			t.Logf("Artifact %s (%s):\n%s", a.Path, a.Type, string(content))
		}
		t.Fatalf("Session failed: %v", err)
	}

	// Verify Results
	if _, err := os.Stat("playground/hello-simon/go.mod"); os.IsNotExist(err) {
		t.Error("go.mod missing")
	}
	if _, err := os.Stat("playground/hello-simon/main.go"); os.IsNotExist(err) {
		t.Error("main.go missing")
	}
}
