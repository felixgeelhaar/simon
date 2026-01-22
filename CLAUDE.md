# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Simon is a local-first AI Agent Governance Runtime written in Go. It acts as a deterministic control plane that enforces structure, discipline, and resource limits for AI coding tools. Simon does not use AI in its control logic - enforcement is rule-based and absolute.

**Key Capabilities:**
- **Coach**: Validates task specifications (goal, definition of done, evidence)
- **Guard**: Enforces hard budgets (tokens, iterations) and command/file scoping
- **Runtime**: Episodic execution with rolling summarization to prevent context window collapse
- **MCP Proxy**: Intercepts tool executions, stores artifacts, returns digests
- **Memory**: Vector-based experience archival for learning from past sessions

## Build & Development Commands

```bash
# Build the binary
go build -o simon cmd/simon/main.go

# Run all tests
go test ./...

# Run tests with verbose output
go test -v ./...

# Run tests with coverage
go test -cover ./...

# Run a single package's tests
go test -v ./internal/runtime/...

# Run a specific test
go test -v -run TestRuntime_ExecuteSession ./internal/runtime/...

# Run the CLI
./simon run demo_task.yaml --provider ollama
./simon run task.yaml -i --provider openai --model gpt-4o

# Configure providers
./simon config set openai.api_key <key>
./simon config set openai.base_url https://openrouter.ai/api/v1
```

**Website (in `website/`):**
```bash
npm run dev      # Development server
npm run build    # Production build
npm run preview  # Preview production build
```

## Architecture

```
User (CLI/TUI)
    ↓
Cobra CLI / Bubbletea TUI
    ↓
Coach (Spec Loading & Validation)
    ↓
Guard (Policy Enforcement)
    ↓
Runtime Engine (Episodic Execution Loop)
    ├── Provider Adapter (OpenAI, Anthropic, Gemini, Ollama, CLI)
    ├── MCP Proxy (Tool Execution & Artifact Storage)
    ├── Observer (Logging & OpenTelemetry)
    └── Storage Layer (SQLite + Filesystem)
```

### Core Packages

| Package | Location | Purpose |
|---------|----------|---------|
| **coach** | `internal/coach/` | TaskSpec loading, validation, prompt linting |
| **guard** | `internal/guard/` | Policy enforcement, budget checking, command/file validation |
| **runtime** | `internal/runtime/` | Main execution loop, context management, verification |
| **provider** | `internal/provider/` | AI model adapters (OpenAI, Anthropic, Gemini, Ollama, Stub) |
| **mcp** | `internal/mcp/` | Tool execution proxy, artifact management |
| **store** | `internal/store/` | SQLite storage, artifact persistence, vector memory |
| **observe** | `internal/observe/` | Structured logging with Bolt, OpenTelemetry tracing |
| **plugin** | `internal/plugin/` | gRPC plugin system (HashiCorp go-plugin) |
| **ui** | `internal/ui/` | TUI (Bubbletea) and silent UI modes |

### Key Interfaces

**Provider Interface** (`internal/provider/provider.go`):
```go
type Provider interface {
    Chat(ctx context.Context, messages []Message) (*Response, error)
    Embed(ctx context.Context, text string) ([]float32, error)
    Name() string
}
```

**Storage Interface** (`internal/store/types.go`):
```go
type Storage interface {
    CreateSession(*Session) error
    GetSession(id string) (*Session, error)
    SaveArtifact(*Artifact, []byte) error
    AddMemory(content string, vector []float32, meta map[string]string) error
    SearchMemory(vector []float32, limit int) ([]MemoryItem, error)
}
```

### Execution Flow

1. **Load TaskSpec** - Coach validates YAML spec (goal, definition_of_done, evidence)
2. **Guard Check** - Verify budget compliance before each iteration
3. **Context Management** - Summarize if history exceeds limits
4. **Provider Call** - Get model response
5. **Tool Execution** - MCP Proxy executes tools, stores artifacts, returns digests
6. **Verification** - Check if evidence files exist
7. **Memory Archival** - Store successful sessions for future reference

### Policy Enforcement

Guard violations are fatal and halt execution immediately. Default policy:
- `MaxIterations`: 20
- `MaxPromptTokens`: 8000
- `MaxOutputTokens`: 4000
- `AllowedCommands`: `["ls", "cat", "grep", "git", "go", "mkdir", "echo"]`

## Task Specification Format

```yaml
goal: "Create a Go CLI project that prints 'Hello, Simon!'"
definition_of_done: "A working go.mod and main.go exist."
evidence: ["main.go", "go.mod"]
```

## Testing Patterns

Tests use temporary directories and the `StubProvider` for deterministic testing:

```go
func TestExample(t *testing.T) {
    tmpDir, _ := os.MkdirTemp("", "test-*")
    defer os.RemoveAll(tmpDir)

    s, _ := store.NewSQLiteStore(filepath.Join(tmpDir, "db"), filepath.Join(tmpDir, "artifacts"))
    g := guard.New(guard.DefaultPolicy)
    p := provider.NewStubProvider()
    // ... test logic
}
```

## Release Process

Releases are managed via GoReleaser and triggered by version tags:

```bash
# Tag and release (automated via GitHub Actions)
git tag v1.0.0
git push origin v1.0.0
```

Binaries are published to GitHub Releases and Homebrew Tap.

## Configuration Storage

- Database: `~/.simon/data.db` (SQLite)
- Artifacts: `~/.simon/artifacts/`
- Config keys: `openai.api_key`, `openai.base_url`, `anthropic.api_key`, `gemini.api_key`, `ollama.host`
