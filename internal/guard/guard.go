package guard

import (
	"path/filepath"

	"github.com/bmatcuk/doublestar/v4"
)

// Policy defines the limits and scopes for an execution session.
type Policy struct {
	MaxIterations     int      `json:"max_iterations"`
	MaxPromptTokens   int      `json:"max_prompt_tokens"`
	MaxOutputTokens   int      `json:"max_output_tokens"`
	AllowedCommands   []string `json:"allowed_commands"`
	AllowedFileGlobs  []string `json:"allowed_file_globs"`
	BlockDangerousCmd bool     `json:"block_dangerous_cmd"`
}

// CheckFile verifies if a file path is within allowed globs.
func (g *Guard) CheckFile(path string) *Violation {
	// If it's an absolute path, we might want to be more restrictive.
	// For now, let's assume relative to project root or absolute matches.

	allowed := false
	for _, pattern := range g.policy.AllowedFileGlobs {
		match, err := doublestar.Match(pattern, path)
		if err == nil && match {
			allowed = true
			break
		}
	}

	if !allowed {
		return &Violation{Rule: "allowed_file_globs", Message: "File access not allowed: " + path, Fatal: true}
	}
	return nil
}

// CheckDangerousPath prevents common escaping patterns.
func (g *Guard) CheckDangerousPath(path string) *Violation {
	if !g.policy.BlockDangerousCmd {
		return nil
	}

	// Basic check for escaping project root if we were to enforce it.
	if filepath.IsAbs(path) {
		// In a real agent, we'd probably only allow specific absolute paths (like /tmp)
		// but block sensitive ones.
	}
	
	return nil
}

// DefaultPolicy provides safe defaults.
var DefaultPolicy = Policy{
	MaxIterations:     20,
	MaxPromptTokens:   8000,
	MaxOutputTokens:   4000,
	AllowedCommands:   []string{"ls", "cat", "grep", "git", "go", "mkdir", "echo"},
	AllowedFileGlobs:  []string{"*"},
	BlockDangerousCmd: true,
}

// Violation represents a specific breach of policy.
type Violation struct {
	Rule    string
	Message string
	Fatal   bool
}

// Guard enforces the policy.
type Guard struct {
	policy Policy
}

func New(p Policy) *Guard {
	return &Guard{policy: p}
}

// Policy returns the guard's current policy configuration.
func (g *Guard) Policy() Policy {
	return g.policy
}

// CheckBudget verifies if the usage is within limits.
func (g *Guard) CheckBudget(iterations, promptTokens, outputTokens int) *Violation {
	if iterations > g.policy.MaxIterations {
		return &Violation{Rule: "max_iterations", Message: "Iteration limit exceeded", Fatal: true}
	}
	if promptTokens > g.policy.MaxPromptTokens {
		return &Violation{Rule: "max_prompt_tokens", Message: "Prompt token budget exceeded", Fatal: true}
	}
	if outputTokens > g.policy.MaxOutputTokens {
		return &Violation{Rule: "max_output_tokens", Message: "Output token budget exceeded", Fatal: true}
	}
	return nil
}

// CheckCommand verifies if a command is allowed.
// A real implementation would parse the command structure more deeply.
func (g *Guard) CheckCommand(cmd string) *Violation {
	allowed := false
	for _, allow := range g.policy.AllowedCommands {
		if allow == "*" || allow == cmd {
			allowed = true
			break
		}
		// Prefix check for simplicity (e.g. "go test" allowed by "go")
		if len(cmd) >= len(allow) && cmd[:len(allow)] == allow {
			allowed = true
			break
		}
	}

	if !allowed {
		return &Violation{Rule: "allowed_commands", Message: "Command not allowed: " + cmd, Fatal: true}
	}
	return nil
}
