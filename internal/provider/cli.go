package provider

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

type CLIProvider struct {
	binaryPath string
	args       []string
}

func NewCLIProvider(binaryPath string, args []string) (*CLIProvider, error) {
	if binaryPath == "" {
		return nil, fmt.Errorf("binary path is required for CLI provider")
	}
	return &CLIProvider{
		binaryPath: binaryPath,
		args:       args,
	}, nil
}

func (p *CLIProvider) Name() string {
	return "cli-" + p.binaryPath
}

func (p *CLIProvider) Chat(ctx context.Context, messages []Message) (*Response, error) {
	var prompt string
	if len(messages) > 0 {
		prompt = messages[len(messages)-1].Content
	}

	fullArgs := append(p.args, prompt)
	
	execCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()

	cmd := exec.CommandContext(execCtx, p.binaryPath, fullArgs...)
	
	output, err := cmd.CombinedOutput()
	result := string(output)
	
	if err != nil {
		if execCtx.Err() == context.DeadlineExceeded {
			return nil, fmt.Errorf("cli agent timed out: %w", err)
		}
		return nil, fmt.Errorf("cli agent failed: %w\nOutput: %s", err, result)
	}

	return &Response{
		Content: result,
		Usage: Usage{
			TotalTokens: len(strings.Fields(result)), 
		},
	}, nil
}

func (p *CLIProvider) Embed(ctx context.Context, text string) ([]float32, error) {
	return nil, fmt.Errorf("embeddings not supported by CLI provider")
}
