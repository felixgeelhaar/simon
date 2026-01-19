# Simon — AI Agent Governance Runtime 🧙‍♂️

Simon is a local-first infrastructure layer designed to govern AI agent execution. It acts as a deterministic control plane that enforces clarity, discipline, and resource limits to make AI coding tools more predictable, secure, and cost-effective.

---

## 🔮 Core Value Proposition

Modern AI coding tools are powerful but fragile. Simon exists to make them usable longer, cheaper, and more predictably — without constant human babysitting.

*   **Coach**: Enforces structured prompts and a clear "Definition of Done".
*   **Guard**: Hard budgets for tokens, iterations, and safe command/file scoping.
*   **Runtime**: Episodic execution with rolling summarization to prevent context window collapse.
*   **MCP Proxy**: Intercepts and digests tool outputs to reduce noise and maintain security.
*   **Memory**: Vector-based experience archival to learn from past successful sessions.

---

## 🚀 Quick Start

### Installation

```bash
# Clone the repository
git clone https://github.com/felixgeelhaar/simon.git
cd simon

# Build the binary
go build -o simon cmd/simon/main.go
```

### Configuration

Simon supports multiple providers including **Ollama (local)**, **OpenAI**, **Anthropic**, and **Gemini**.

```bash
# Configure your OpenAI key
./simon config set openai.api_key your-api-key

# (Optional) Configure OpenRouter or custom Base URL
./simon config set openai.base_url https://openrouter.ai/api/v1
```

### Execution

Define your task in a YAML file (`task.yaml`):

```yaml
goal: "Create a Go CLI project that prints 'Hello, Simon!'"
definition_of_done: "A working go.mod and main.go exist."
evidence: ["main.go", "go.mod"]
```

Run with your preferred provider:

```bash
# Using local Ollama (Llama 3.2 recommended)
./simon run task.yaml --provider ollama

# Using OpenAI GPT-4o with Interactive TUI
./simon run task.yaml -i --provider openai --model gpt-4o
```

---

## 🛡️ Governance & Security

Simon operates on a **Deterministic Control Plane**. AI models are treated as external workers, while the enforcement logic is rule-based and absolute.

*   **Budget Enforcement**: Execution halts immediately if token or iteration limits are reached.
*   **Command Scoping**: Only authorized shell commands (e.g., `go`, `git`, `ls`) are permitted.
*   **Verification-Driven**: Tasks are not marked complete until the defined "Evidence" is verified by the runtime.

---

## 🛠️ Architecture

Simon is built in Go and utilizes a plugin-ready architecture.

*   **Frontend**: Cobra CLI / Bubbletea TUI.
*   **Storage**: SQLite (Metadata, Memory, Config) + Local Filesystem (Artifacts).
*   **Execution**: Episodic loop with rolling summarization.
*   **Plugins**: gRPC-based (hashicorp/go-plugin) for extensible Coach and Guard logic.

---

## 🏗️ Deployment

### GitHub Pages
1. Go to **Settings > Pages** in your repository.
2. Set **Source** to **GitHub Actions**.
3. The site will deploy automatically to `https://felixgeelhaar.github.io/simon`.

### Homebrew Tap
Simon is distributed via a Homebrew Tap. To ensure the automated releases work:
1. Create a repository named `homebrew-tap` (if not already existing).
2. Add a repository secret named `HOMEBREW_TAP_TOKEN` with a Fine-grained PAT that has write access to your tap repository.
3. Update `.github/workflows/release.yml` to use this token.

---

## 🤝 Contributing

We welcome contributions! Please check out our [Roadmap](.roady/spec.yaml) to see what we're building next.

1. Fork the repo
2. Create your feature branch (`git checkout -b feature/magic`)
3. Commit your changes (`git commit -m 'Add some magic'`)
4. Push to the branch (`git push origin feature/magic`)
5. Open a Pull Request

---

## 📜 License

Distributed under the MIT License. See `LICENSE` for more information.

---

*Built for Power Developers by [Felix Geelhaar](https://github.com/felixgeelhaar)*
