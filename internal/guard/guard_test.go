package guard

import (
	"testing"
)

func TestGuard_CheckFile(t *testing.T) {
	g := New(Policy{
		AllowedFileGlobs: []string{"internal/**/*.go", "cmd/*.go"},
	})

	t.Run("Allowed", func(t *testing.T) {
		if v := g.CheckFile("internal/coach/coach.go"); v != nil {
			t.Errorf("Unexpected violation: %v", v.Message)
		}
		if v := g.CheckFile("cmd/main.go"); v != nil {
			t.Errorf("Unexpected violation: %v", v.Message)
		}
	})

	t.Run("Blocked", func(t *testing.T) {
		if v := g.CheckFile("pkg/api/api.go"); v == nil {
			t.Error("Expected violation for pkg/")
		}
		if v := g.CheckFile("/etc/passwd"); v == nil {
			t.Error("Expected violation for absolute path")
		}
	})
}

func TestGuard_CheckBudget(t *testing.T) {
	g := New(Policy{
		MaxIterations:   5,
		MaxPromptTokens: 1000,
		MaxOutputTokens: 500,
	})

	t.Run("Within", func(t *testing.T) {
		if v := g.CheckBudget(3, 500, 200); v != nil {
			t.Errorf("Unexpected violation: %v", v.Message)
		}
	})

	t.Run("Iteration Exceeded", func(t *testing.T) {
		if v := g.CheckBudget(6, 100, 100); v == nil {
			t.Error("Expected iteration violation")
		}
	})

	t.Run("Prompt Tokens Exceeded", func(t *testing.T) {
		if v := g.CheckBudget(1, 1500, 100); v == nil {
			t.Error("Expected prompt token violation")
		}
	})

	t.Run("Output Tokens Exceeded", func(t *testing.T) {
		if v := g.CheckBudget(1, 100, 600); v == nil {
			t.Error("Expected output token violation")
		}
	})
}

func TestGuard_CheckCommand(t *testing.T) {
	g := New(Policy{
		AllowedCommands: []string{"go", "ls", "grep"},
	})

	t.Run("Allowed Exact", func(t *testing.T) {
		if v := g.CheckCommand("ls"); v != nil {
			t.Errorf("Unexpected violation: %v", v.Message)
		}
	})

	t.Run("Allowed Prefix", func(t *testing.T) {
		if v := g.CheckCommand("go test"); v != nil {
			t.Errorf("Unexpected violation: %v", v.Message)
		}
	})

	t.Run("Blocked", func(t *testing.T) {
		if v := g.CheckCommand("rm -rf /"); v == nil {
			t.Error("Expected violation for rm")
		}
	})

	t.Run("Wildcard", func(t *testing.T) {
		gw := New(Policy{AllowedCommands: []string{"*"}})
		if v := gw.CheckCommand("anything"); v != nil {
			t.Error("Expected no violation for wildcard")
		}
	})
}

func TestGuard_CheckDangerousPath(t *testing.T) {
	g := New(Policy{BlockDangerousCmd: true})
	// Current implementation is a stub, but let's test it
	if v := g.CheckDangerousPath("/abs/path"); v != nil {
		t.Error("Expected no violation for now (stub)")
	}
	
	g2 := New(Policy{BlockDangerousCmd: false})
	if v := g2.CheckDangerousPath("/any"); v != nil {
		t.Error("Expected no violation when disabled")
	}
}
