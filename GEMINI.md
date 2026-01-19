# Simon

## Project Overview
Simon is a local-first infrastructure tool designed to govern AI agent execution. It acts as a runtime layer that enforces clarity, discipline, and resource limits (tokens, iterations) to make AI coding tools more predictable and cost-effective.

**Core Value Proposition:**
*   **Coach:** Enforces structured prompts and definition of done.
*   **Guard:** Hard budgets for context, scope, and loops.
*   **Runtime:** Episodic execution to manage context windows effectively.
*   **MCP Proxy:** Intercepts and digests tool outputs to reduce noise.

**Architecture:**
*   **Language:** Go
*   **Design Pattern:** Plugin-based (gRPC), separating control logic (deterministic) from AI execution (probabilistic).
*   **Data Store:** SQLite (metadata) + Filesystem (artifacts).

## Documentation
The project is currently in the design phase. Key documentation is located in the `docs/` directory:
*   [Vision](docs/vision.md): High-level philosophy and problem space.
*   [PRD](docs/prd.md): Product requirements, capabilities, and user outcomes.
*   [TDD](docs/tdd.md): Technical architecture, design goals, and library choices.

## Development Status
*   **Phase:** Design / Inception
*   **Current Structure:** Documentation only. Implementation pending.

## Planned Tech Stack & Libraries
*   `felixgeelhaar/statekit` (State management)
*   `felixgeelhaar/fortify` (Resilience)
*   `felixgeelhaar/bolt` (Logging)
*   `felixgeelhaar/mcp-go` (MCP support - conditional)

## Roadmap
1.  Initialize Go module.
2.  Scaffold core architecture (Coach, Guard, Runtime).
3.  Implement SQLite storage layer.
4.  Develop CLI interface.
