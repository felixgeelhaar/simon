package mcp

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/felixgeelhaar/simon/internal/guard"
	"github.com/felixgeelhaar/simon/internal/provider"
	"github.com/felixgeelhaar/simon/internal/store"
)

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
			// Try to stringify if it's not a string (e.g. array of strings)
			// Llama 3 might decide to pass ["ls", "-l"] instead of "ls -l"
			bytes, _ := json.Marshal(v)
			cmdStr = string(bytes)
			// This might produce "[\"ls\",\"-l\"]" which bash won't like.
			// Ideally we join if it's a slice.
			if slice, ok := v.([]interface{}); ok {
				var parts []string
				for _, s := range slice {
					parts = append(parts, fmt.Sprint(s))
				}
				cmdStr = strings.Join(parts, " ")
			}
		}

		dirVal, hasDir := args["dir"]
		var dirStr string
		if hasDir {
			if s, ok := dirVal.(string); ok {
				dirStr = s
			}
		}

		// 1. Guard Check
		parts := strings.Fields(cmdStr)
		if len(parts) == 0 {
			return "", fmt.Errorf("empty command")
		}
		if v := p.guard.CheckCommand(parts[0]); v != nil {
			return "", fmt.Errorf("guard violation: %s", v.Message)
		}

		// 2. Real Execution with Timeout
		execCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()

		cmd := exec.CommandContext(execCtx, "bash", "-c", cmdStr) // #nosec G204
		if dirStr != "" {
			cmd.Dir = dirStr
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

func (p *Proxy) hash(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:])
}