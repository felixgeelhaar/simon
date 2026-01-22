package mcp

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/felixgeelhaar/simon/internal/guard"
	"github.com/felixgeelhaar/simon/internal/provider"
	"github.com/felixgeelhaar/simon/internal/store"
)

// dangerousPatterns contains regex patterns that indicate potentially dangerous command constructs
var dangerousPatterns = []*regexp.Regexp{
	regexp.MustCompile(`;\s*\w`),             // Command chaining with semicolon (but allow trailing semicolons)
	regexp.MustCompile(`\|[^|]`),             // Pipe to another command (single pipe)
	regexp.MustCompile(`\|\|`),               // OR operator
	regexp.MustCompile(`&&`),                 // AND operator
	regexp.MustCompile(`\$\(`),               // Command substitution
	regexp.MustCompile("`"),                  // Backtick substitution
	regexp.MustCompile(`>>`),                 // Append redirect (allow single > for overwrite)
	regexp.MustCompile(`<\(`),                // Process substitution
	regexp.MustCompile(`\$\{`),               // Variable expansion with braces
	regexp.MustCompile(`(?i)\beval\s`),       // eval command
	regexp.MustCompile(`(?i)\bsource\s`),     // source command
	regexp.MustCompile(`(?i)\bexec\s`),       // exec command
	regexp.MustCompile(`(?i)\bnc\s`),         // netcat
	regexp.MustCompile(`(?i)\bcurl\b.*\|\s*sh`),  // curl pipe to sh
	regexp.MustCompile(`(?i)\bwget\b.*\|\s*sh`),  // wget pipe to sh
	regexp.MustCompile(`(?i)\b(bash|sh|zsh)\s+-c`), // Shell execution with -c
}

type Proxy struct {
	store store.Storage
	guard *guard.Guard
}

func NewProxy(s store.Storage, g *guard.Guard) *Proxy {
	return &Proxy{store: s, guard: g}
}

// ToolResult represents the processed outcome of a tool call.
type ToolResult struct {
	ToolCallID string
	Name       string
	Digest     string
	IsError    bool
}

// HandleToolCalls processes a batch of tool calls, executing them,
// storing the raw outputs as artifacts, and returning digests for the context.
func (p *Proxy) HandleToolCalls(ctx context.Context, sessionID string, calls []provider.ToolCall) ([]ToolResult, error) {
	var results []ToolResult

	for _, call := range calls {
		// 1. Execute
		rawOutput, err := p.execute(ctx, call)
		isError := false
		if err != nil {
			rawOutput = fmt.Sprintf("Error executing tool: %v\n%s", err, rawOutput)
			isError = true
		}

		// 2. Store Raw Artifact
		digestStr := p.hash(rawOutput)
		uniqueID := fmt.Sprintf("%s-%d", call.ID, time.Now().UnixNano())
		artifactPath := fmt.Sprintf("artifacts/%s/%s_%s.txt", sessionID, call.Name, uniqueID)
		
		artifact := &store.Artifact{
			ID:        fmt.Sprintf("art-%s-%s", sessionID, uniqueID),
			SessionID: sessionID,
			Path:      artifactPath,
			Type:      "tool_output",
			CreatedAt: time.Now(),
			Digest:    digestStr,
		}

		if err := p.store.SaveArtifact(artifact, []byte(rawOutput)); err != nil {
			return nil, fmt.Errorf("failed to save artifact: %w", err)
		}

		// 3. Create Digest for Context
		displayDigest := rawOutput
		if len(displayDigest) > 200 {
			displayDigest = displayDigest[:197] + "..."
		}
		
		results = append(results, ToolResult{
			ToolCallID: call.ID,
			Name:       call.Name,
			Digest:     fmt.Sprintf("Tool %s executed. Output stored at %s. Summary: %s", call.Name, artifactPath, displayDigest),
			IsError:    isError,
		})
	}

	return results, nil
}

// validateCommand checks for dangerous command patterns that could lead to injection
func (p *Proxy) validateCommand(cmdStr string) error {
	for _, pattern := range dangerousPatterns {
		if pattern.MatchString(cmdStr) {
			return fmt.Errorf("potentially dangerous command pattern detected: %s", pattern.String())
		}
	}
	return nil
}

// validateShellCommand performs additional validation for commands executed through bash
// This is more permissive than validateCommand since shell features are expected
func (p *Proxy) validateShellCommand(cmdStr string) error {
	// Block the most dangerous patterns even in shell mode
	shellDangerPatterns := []*regexp.Regexp{
		regexp.MustCompile(`(?i)\beval\s`),             // eval command
		regexp.MustCompile(`(?i)\bsource\s`),           // source command
		regexp.MustCompile("`"),                        // Backtick substitution
		regexp.MustCompile(`\$\(`),                     // Command substitution
		regexp.MustCompile(`(?i)\bcurl\b.*\|\s*sh`),    // curl pipe to sh
		regexp.MustCompile(`(?i)\bwget\b.*\|\s*sh`),    // wget pipe to sh
		regexp.MustCompile(`(?i)\b(bash|sh|zsh)\s+-c`), // Nested shell execution
		regexp.MustCompile(`(?i)\bsudo\s`),             // sudo command
		regexp.MustCompile(`(?i)\bchmod\s+[0-7]*7`),    // Making files world-writable
		regexp.MustCompile(`(?i)/etc/passwd`),          // Accessing passwd file
		regexp.MustCompile(`(?i)/etc/shadow`),          // Accessing shadow file
		regexp.MustCompile(`(?i)~/.ssh`),               // Accessing SSH keys
		regexp.MustCompile(`(?i)rm\s+-rf\s+/`),         // Dangerous rm command
	}

	for _, pattern := range shellDangerPatterns {
		if pattern.MatchString(cmdStr) {
			return fmt.Errorf("dangerous shell command pattern blocked: %s", pattern.String())
		}
	}

	return nil
}

// parseCommand safely parses a command string into executable and arguments
// Returns the command name and its arguments separately
func (p *Proxy) parseCommand(cmdStr string) (string, []string, error) {
	// Trim whitespace
	cmdStr = strings.TrimSpace(cmdStr)
	if cmdStr == "" {
		return "", nil, fmt.Errorf("empty command")
	}

	// Use a simple shell-like parsing that respects quotes
	var args []string
	var current strings.Builder
	inSingleQuote := false
	inDoubleQuote := false
	escaped := false

	for i := 0; i < len(cmdStr); i++ {
		c := cmdStr[i]

		if escaped {
			current.WriteByte(c)
			escaped = false
			continue
		}

		if c == '\\' && !inSingleQuote {
			escaped = true
			continue
		}

		if c == '\'' && !inDoubleQuote {
			inSingleQuote = !inSingleQuote
			continue
		}

		if c == '"' && !inSingleQuote {
			inDoubleQuote = !inDoubleQuote
			continue
		}

		if c == ' ' && !inSingleQuote && !inDoubleQuote {
			if current.Len() > 0 {
				args = append(args, current.String())
				current.Reset()
			}
			continue
		}

		current.WriteByte(c)
	}

	if current.Len() > 0 {
		args = append(args, current.String())
	}

	if inSingleQuote || inDoubleQuote {
		return "", nil, fmt.Errorf("unclosed quote in command")
	}

	if len(args) == 0 {
		return "", nil, fmt.Errorf("no command specified")
	}

	return args[0], args[1:], nil
}

// sanitizeWorkDir validates and sanitizes the working directory path
func (p *Proxy) sanitizeWorkDir(dir string) (string, error) {
	if dir == "" {
		return "", nil
	}

	// Clean the path to remove any .. or . components
	cleanPath := filepath.Clean(dir)

	// Resolve to absolute path
	absPath, err := filepath.Abs(cleanPath)
	if err != nil {
		return "", fmt.Errorf("invalid directory path: %w", err)
	}

	// Check for path traversal attempts
	if strings.Contains(dir, "..") {
		return "", fmt.Errorf("path traversal not allowed in working directory")
	}

	return absPath, nil
}

func (p *Proxy) execute(ctx context.Context, call provider.ToolCall) (string, error) {
	switch call.Name {
	case "run_shell":
		var args map[string]interface{}
		if err := json.Unmarshal([]byte(call.Args), &args); err != nil {
			return "", fmt.Errorf("invalid args: %w", err)
		}
		cmdVal, ok := args["cmd"]
		if !ok {
			return "", fmt.Errorf("missing cmd argument")
		}

		var cmdStr string
		switch v := cmdVal.(type) {
		case string:
			cmdStr = v
		default:
			// Handle array of strings (e.g., ["ls", "-l"])
			if slice, ok := v.([]interface{}); ok {
				var parts []string
				for _, s := range slice {
					parts = append(parts, fmt.Sprint(s))
				}
				cmdStr = strings.Join(parts, " ")
			} else {
				return "", fmt.Errorf("cmd must be a string or array of strings")
			}
		}

		// 1. Validate command for dangerous patterns
		if err := p.validateCommand(cmdStr); err != nil {
			return "", fmt.Errorf("command validation failed: %w", err)
		}

		// 2. Parse command into executable and arguments
		cmdName, cmdArgs, err := p.parseCommand(cmdStr)
		if err != nil {
			return "", fmt.Errorf("failed to parse command: %w", err)
		}

		// 3. Guard Check on the command name
		if v := p.guard.CheckCommand(cmdName); v != nil {
			return "", fmt.Errorf("guard violation: %s", v.Message)
		}

		// 4. Sanitize working directory if provided
		dirVal, hasDir := args["dir"]
		var dirStr string
		if hasDir {
			if s, ok := dirVal.(string); ok {
				dirStr, err = p.sanitizeWorkDir(s)
				if err != nil {
					return "", fmt.Errorf("invalid working directory: %w", err)
				}
			}
		}

		// 5. Determine execution mode based on command complexity
		// If command contains shell features (redirection), use bash with strict validation
		// Otherwise use direct exec for maximum safety
		needsShell := strings.ContainsAny(cmdStr, "><")

		// 6. Real Execution with Timeout
		execCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()

		var cmd *exec.Cmd
		if needsShell {
			// Use bash for commands requiring shell features, but validate first
			// Additional validation for shell mode
			if err := p.validateShellCommand(cmdStr); err != nil {
				cancel()
				return "", fmt.Errorf("shell command validation failed: %w", err)
			}
			cmd = exec.CommandContext(execCtx, "/bin/bash", "-c", cmdStr)
		} else {
			// Direct execution for simple commands (safer)
			cmdPath, err := exec.LookPath(cmdName)
			if err != nil {
				cancel()
				return "", fmt.Errorf("command not found: %s", cmdName)
			}
			cmd = exec.CommandContext(execCtx, cmdPath, cmdArgs...)
		}

		if dirStr != "" {
			cmd.Dir = dirStr
		}

		// Set a clean environment to prevent environment variable injection
		cmd.Env = []string{
			"PATH=/usr/local/bin:/usr/bin:/bin",
			"HOME=" + getHomeDir(),
			"LANG=en_US.UTF-8",
		}

		output, err := cmd.CombinedOutput()

		result := string(output)
		if err != nil {
			if execCtx.Err() == context.DeadlineExceeded {
				return result + "\n[ERROR] Command timed out", fmt.Errorf("command timed out")
			}
			return result + fmt.Sprintf("\n[ERROR] %v", err), nil
		}
		return result, nil

	default:
		return "Unknown tool", fmt.Errorf("unknown tool: %s", call.Name)
	}
}

// getHomeDir returns the user's home directory or a safe default
func getHomeDir() string {
	if home, err := filepath.Abs("."); err == nil {
		return home
	}
	return "/tmp"
}

func (p *Proxy) hash(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:])
}