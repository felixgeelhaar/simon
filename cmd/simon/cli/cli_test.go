package cli

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/felixgeelhaar/simon/internal/observe"
	"github.com/felixgeelhaar/simon/internal/provider"
	"github.com/felixgeelhaar/simon/internal/store"
)

func TestRunner(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "cli-test-*")
	defer os.RemoveAll(tmpDir)

	s, _ := store.NewSQLiteStore(filepath.Join(tmpDir, "db"), filepath.Join(tmpDir, "artifacts"))
	p := provider.NewStubProvider()
	o := observe.New(os.Stdout, true)

	specPath := filepath.Join(tmpDir, "spec.yaml")
	os.WriteFile(specPath, []byte("goal: test\ndefinition_of_done: test\nevidence: [test.txt]"), 0600)
	os.WriteFile(filepath.Join(tmpDir, "test.txt"), []byte("evidence"), 0600)

	// Since we run in tmpDir, we need to ensure the runtime verifies the file in tmpDir.
	// But runtime executes relative to CWD usually.
	// Spec validation checks existence? No, Coach checks non-empty list.
	// Runtime checks file existence.
	// We should probably change CWD or use absolute path in evidence.
	
	absEvidence := filepath.Join(tmpDir, "test.txt")
	os.WriteFile(specPath, []byte("goal: test\ndefinition_of_done: test\nevidence: ["+absEvidence+"]"), 0600)

	r := NewRunner(o, s, p, specPath, nil)
	if err := r.Run(context.Background()); err != nil {
		t.Fatalf("Run failed: %v", err)
	}
}

func TestCLI_Root(t *testing.T) {
	// Setup store location for test
	tmpDir, _ := os.MkdirTemp("", "cli-test-*")
	defer os.RemoveAll(tmpDir)
	
	// We can't easily mock the home dir used in getStore without refactoring
	// but we can at least test that commands are registered.
	
	if len(RootCmd.Commands()) < 3 {
		t.Errorf("Expected at least 3 subcommands (run, list, config), got %d", len(RootCmd.Commands()))
	}
}

func TestCLI_Config(t *testing.T) {
	// This test will interact with the real filesystem (~/.simon)
	// unless we refactor to allow injecting the store path.
	// For a quick prototype test, we'll just check if the commands exist.
	
	found := false
	for _, cmd := range RootCmd.Commands() {
		if cmd.Name() == "config" {
			found = true
			if len(cmd.Commands()) < 2 {
				t.Errorf("Expected set and get subcommands for config, got %d", len(cmd.Commands()))
			}
		}
	}
	if !found {
		t.Error("config command not found")
	}
}
