package runtime

import (
	"context"
	"fmt"
	"sync"

	"github.com/felixgeelhaar/simon/internal/provider"
)

// ToolDefinition represents a tool that can be called by the AI.
type ToolDefinition struct {
	Name        string
	Description string
	Parameters  map[string]interface{}
}

// ToolExecutor is a function that executes a tool call and returns the result.
type ToolExecutor func(ctx context.Context, sessionID string, call provider.ToolCall) (string, error)

// ToolRegistry manages available tools and their execution.
// It provides a centralized way to register, discover, and execute tools.
type ToolRegistry struct {
	mu        sync.RWMutex
	tools     map[string]ToolDefinition
	executors map[string]ToolExecutor
}

// NewToolRegistry creates a new tool registry.
func NewToolRegistry() *ToolRegistry {
	return &ToolRegistry{
		tools:     make(map[string]ToolDefinition),
		executors: make(map[string]ToolExecutor),
	}
}

// Register adds a tool to the registry.
func (tr *ToolRegistry) Register(tool ToolDefinition, executor ToolExecutor) error {
	tr.mu.Lock()
	defer tr.mu.Unlock()

	if _, exists := tr.tools[tool.Name]; exists {
		return fmt.Errorf("tool %q already registered", tool.Name)
	}

	tr.tools[tool.Name] = tool
	tr.executors[tool.Name] = executor
	return nil
}

// Unregister removes a tool from the registry.
func (tr *ToolRegistry) Unregister(name string) {
	tr.mu.Lock()
	defer tr.mu.Unlock()

	delete(tr.tools, name)
	delete(tr.executors, name)
}

// Get returns a tool definition by name.
func (tr *ToolRegistry) Get(name string) (ToolDefinition, bool) {
	tr.mu.RLock()
	defer tr.mu.RUnlock()

	tool, ok := tr.tools[name]
	return tool, ok
}

// List returns all registered tool definitions.
func (tr *ToolRegistry) List() []ToolDefinition {
	tr.mu.RLock()
	defer tr.mu.RUnlock()

	tools := make([]ToolDefinition, 0, len(tr.tools))
	for _, tool := range tr.tools {
		tools = append(tools, tool)
	}
	return tools
}

// Execute runs a tool and returns its result.
func (tr *ToolRegistry) Execute(ctx context.Context, sessionID string, call provider.ToolCall) (string, error) {
	tr.mu.RLock()
	executor, ok := tr.executors[call.Name]
	tr.mu.RUnlock()

	if !ok {
		return "", fmt.Errorf("unknown tool: %s", call.Name)
	}

	return executor(ctx, sessionID, call)
}

// HasTool checks if a tool is registered.
func (tr *ToolRegistry) HasTool(name string) bool {
	tr.mu.RLock()
	defer tr.mu.RUnlock()

	_, ok := tr.tools[name]
	return ok
}

// Count returns the number of registered tools.
func (tr *ToolRegistry) Count() int {
	tr.mu.RLock()
	defer tr.mu.RUnlock()

	return len(tr.tools)
}

// GetToolsForProvider returns tool definitions in a format suitable for provider APIs.
func (tr *ToolRegistry) GetToolsForProvider() []map[string]interface{} {
	tr.mu.RLock()
	defer tr.mu.RUnlock()

	result := make([]map[string]interface{}, 0, len(tr.tools))
	for _, tool := range tr.tools {
		result = append(result, map[string]interface{}{
			"type": "function",
			"function": map[string]interface{}{
				"name":        tool.Name,
				"description": tool.Description,
				"parameters":  tool.Parameters,
			},
		})
	}
	return result
}
