package e2e

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
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

func TestE2E_RunSession(t *testing.T) {
	// 1. Setup temp directory
	tmpDir, _ := os.MkdirTemp("", "simon-e2e-*")
	defer os.RemoveAll(tmpDir)

	// Create spec and evidence files
	specPath := filepath.Join(tmpDir, "task.yaml")
	evidencePath := filepath.Join(tmpDir, "evidence.txt")

	specContent := "goal: E2E Test\ndefinition_of_done: Done\nevidence: [" + evidencePath + "]"
	os.WriteFile(specPath, []byte(specContent), 0600)
	os.WriteFile(evidencePath, []byte("proof"), 0600)

	// 2. Initialize components
	simonDir := filepath.Join(tmpDir, ".simon")
	storeLayer, err := store.NewSQLiteStore(
		filepath.Join(simonDir, "metadata.db"),
		filepath.Join(simonDir, "artifacts"),
	)
	if err != nil {
		t.Fatalf("Failed to init store: %v", err)
	}
	defer storeLayer.Close()

	// Use stub provider for testing (internal use only, not exposed via CLI)
	stubProvider := provider.NewStubProvider()

	var logBuf bytes.Buffer
	obs := observe.New(&logBuf, true)
	defer obs.Close()

	g := guard.New(guard.DefaultPolicy)
	c := coach.New()
	mp := mcp.NewProxy(storeLayer, g)
	rt := runtime.New(storeLayer, g, c, obs, stubProvider, mp)

	// 3. Create and run session
	sessID := "session-e2e-test"
	session := &store.Session{
		ID:        sessID,
		CreatedAt: time.Now(),
		Status:    "initialized",
		Metadata:  map[string]string{"env": "test", "spec": specPath},
	}

	if err := storeLayer.CreateSession(session); err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	ctx := context.Background()
	err = rt.ExecuteSession(ctx, sessID)

	outStr := logBuf.String()
	t.Logf("Output:\n%s", outStr)

	if err != nil {
		t.Fatalf("Session failed: %v", err)
	}

	// 4. Validate Output
	if !strings.Contains(outStr, "verification successful") {
		t.Error("Expected verification success")
	}

	// 5. Validate Persistence
	if _, err := os.Stat(filepath.Join(simonDir, "metadata.db")); os.IsNotExist(err) {
		t.Error("metadata.db not created")
	}
	if _, err := os.Stat(filepath.Join(simonDir, "artifacts")); os.IsNotExist(err) {
		t.Error("artifacts dir not created")
	}
}
