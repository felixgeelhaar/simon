# Simon — Technical Design Document (TDD)

## Design Goals

- Deterministic behavior
- Predictable cost and execution
- Clear separation of concerns
- Extensibility without core modification

---

## High-Level Architecture

User
└── Simon CLI / TUI
└── Coach
└── Guard (Policy Engine)
└── Runtime Engine
├── Provider Adapter
├── Tool Gateway (MCP Proxy)
└── Artifact Store

---

Libraris we want to use:

- felixgeelhaar/statekit # state
- felixgeelhaar/fortify # Resilience
- felixgeelhaar/bolt # high-efficient logging
- felixgeelhaar/mcp-go # Only if needed

## Core Architectural Decisions

### Deterministic Control Plane

- No LLMs in Simon’s control logic
- Enforcement is rule-based
- AI models are treated as external workers

---

### Episodic Execution Model

- Each iteration is stateless from the model’s perspective
- Session state persists on disk
- Summaries replace growing transcripts

This prevents context window exhaustion.

---

### Policy as Code

- Budgets and scopes are declarative
- Policies evaluated before execution
- Violations halt execution

---

### Plugin Architecture

**Mechanism**

- gRPC-based plugins (HashiCorp go-plugin style)

**Extension Points**

- Coach plugins
- Context selectors
- Tool output reducers
- Loop strategies
- Provider adapters

Plugins are isolated, versioned, and optional.

---

## Tool Gateway (MCP Proxy)

- JSON-RPC interception
- Raw tool outputs stored as artifacts
- Reduced digests returned to runtime
- Targeted extraction supported on demand

---

## Storage Model

- SQLite:
  - sessions
  - state
  - artifact index
- Filesystem:
  - raw artifacts
- Markdown:
  - human-readable summaries

---

## Observability

- Structured logs
- OpenTelemetry spans per iteration
- Deterministic replay supported

---

## Security & Safety Considerations

- File and command scope enforcement
- Plugin isolation boundaries
- Explicit stop conditions for loops
