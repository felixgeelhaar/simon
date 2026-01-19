# Simon — Product Requirements Document (PRD)

## Product Goals

Simon is successful when:

- tasks complete in fewer conversational turns
- long-running sessions do not require manual resets
- tool-heavy workflows remain viable
- users trust Simon to stop waste proactively

---

## User Problems

1. Users waste turns clarifying vague prompts.
2. Agents re-plan instead of executing.
3. Tool outputs consume most of the context window.
4. Long loops run past the point of usefulness.
5. Users only realize cost problems after limits are hit.

---

## Core Product Capabilities

### 1. Coach (Pre-Execution)

**User Outcome**
Fewer clarification turns.

**Requirements**

- Structured task specification:
  - Goal
  - Definition of Done
  - Constraints
  - Evidence
- Deterministic prompt linting
- Detection of missing signal (e.g. no DoD, no evidence)
- Estimated usage / burn preview before execution

**Out of Scope**

- Generative prompt rewriting
- Prompt experimentation tooling

---

### 2. Guard (Always-On Enforcement)

**User Outcome**
No catastrophic waste.

**Requirements**

- Hard budgets for:
  - prompt size
  - evidence size
  - diffs
  - tool outputs
- Scope enforcement:
  - allowed files
  - allowed commands
- Iteration limits
- Repeat-failure detection
- Explicit stop conditions

Violations block execution rather than warn.

---

### 3. Runtime (Segmented Execution Loop)

**User Outcome**
Run longer without context collapse.

**Requirements**

- Episodic execution (fresh model call per iteration)
- Externalized state
- Rolling session summary
- Deterministic execution state machine
- Verification-driven loops (tests, commands)

---

### 4. Tool Output Control (MCP Firewall)

**User Outcome**
Tool tokens become manageable.

**Requirements**

- MCP proxy capability
- Artifact storage for raw tool outputs
- Digest + extract pattern
- First-class support for:
  - Playwright
  - Chrome DevTools
  - logs and traces

---

## Non-Goals

Simon will not:

- act as an IDE
- replace AI models
- perform silent refactors
- run unlimited autonomous agents
- bypass provider usage limits

---

## Constraints

- Must work with existing CLIs
- Local-first execution
- Go-based implementation
- Extensible via plugins
- Graceful degradation when MCP is unavailable
