package e2e

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestE2E_RunSession(t *testing.T) {
	// 1. Build Binary
	rootDir, _ := filepath.Abs("../../")
	binPath := filepath.Join(rootDir, "simon_e2e")
	
	buildCmd := exec.Command("go", "build", "-o", binPath, "github.com/felixgeelhaar/simon/cmd/simon")
	buildCmd.Dir = rootDir
	if out, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("Failed to build simon: %v\n%s", err, out)
	}
	defer os.Remove(binPath)

	// 2. Setup Env
	tmpDir, _ := os.MkdirTemp("", "simon-e2e-*")
	defer os.RemoveAll(tmpDir)

	// We need to override the HOME dir so simon uses a fresh DB/Config in tmpDir
	// simon uses os.UserHomeDir() -> .simon
	// We can set HOME env var for the command
	
specPath := filepath.Join(tmpDir, "task.yaml")
evidencePath := filepath.Join(tmpDir, "evidence.txt")
	
	// Create Spec
specContent := "goal: E2E Test\ndefinition_of_done: Done\nevidence: [" + evidencePath + "]"
os.WriteFile(specPath, []byte(specContent), 0600)

	// 3. Run Simon (Fail Verification first)
	// The StubProvider returns "Task complete." immediately.
	// Runtime will check evidencePath, which doesn't exist yet.
	// It will loop.
	// We need the StubProvider to be smart enough or we need to simulate the user creating the file?
	// The current StubProvider in internal/provider/stub.go has 3 responses:
	// 1. "I will start..."
	// 2. "Checking system..." (tool call echo)
	// 3. "Task complete."
	
	// If we run this, it will:
	// Iter 1: "I will start"
	// Iter 2: Tool call "echo hello" -> MCP executes -> Output saved -> History updated
	// Iter 3: "Task complete." -> Verify evidence -> Fail (File missing) -> Loop continues
	// Iter 4: Stub provider runs out of responses -> returns "Task complete." again? 
	// Wait, my StubProvider implementation:
	/*
	if len(m.Responses) == 0 {
		return &Response{Content: "Task complete.", Usage: Usage{}}, nil
	}
	*/
	// So it keeps saying Task complete.
	// The verification keeps failing.
	// The Guard eventually stops it (MaxIterations).
	
	// For a SUCCESSFUL E2E test, we need the file to exist.
	// Or we need the agent to create it using `run_shell`.
	// The StubProvider has a hardcoded tool call: `{"cmd": "echo hello"}`.
	// It does NOT create the evidence file.
	
	// So with the current StubProvider, an E2E run will effectively fail verification until timeout.
	// That's actually a valid test of the system (it works, it just fails the task).
	
	// Let's create the evidence file beforehand so verification passes on Iter 3.
os.WriteFile(evidencePath, []byte("proof"), 0600)

	runCmd := exec.Command(binPath, "run", specPath, "--provider=stub")
	runCmd.Env = append(os.Environ(), "HOME="+tmpDir)
	output, err := runCmd.CombinedOutput()
	
outStr := string(output)
	t.Logf("Output:\n%s", outStr)

	if err != nil {
		t.Fatalf("Simon failed: %v", err)
	}

	// 4. Validate Output
	if !strings.Contains(outStr, "Session execution cycle complete") {
		t.Error("Expected successful completion message")
	}
	if !strings.Contains(outStr, "verification successful") {
		t.Error("Expected verification success")
	}

	// 5. Validate Persistence
simonDir := filepath.Join(tmpDir, ".simon")
	if _, err := os.Stat(filepath.Join(simonDir, "metadata.db")); os.IsNotExist(err) {
		t.Error("metadata.db not created")
	}
	if _, err := os.Stat(filepath.Join(simonDir, "artifacts")); os.IsNotExist(err) {
		t.Error("artifacts dir not created")
	}
}
