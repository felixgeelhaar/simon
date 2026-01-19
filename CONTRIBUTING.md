# Contributing to Simon 🧙‍♂️

We're excited that you're interested in helping make Simon the best governance runtime for AI agents!

## 📜 Code of Conduct

By participating in this project, you agree to maintain a professional and welcoming environment for everyone.

## 🛠️ How to Contribute

### 1. Report Bugs
If you find a bug, please open an issue describing:
- What happened
- What you expected to happen
- Steps to reproduce
- Your environment (OS, Go version, Simon version)

### 2. Suggest Features
We're always looking for ways to improve Simon. Open an issue with the "feature request" tag to start a discussion.

### 3. Submit Pull Requests
1. **Fork** the repository.
2. **Clone** your fork locally.
3. **Create a branch** for your changes (`git checkout -b feature/cool-new-thing`).
4. **Make your changes**. Ensure you add tests if applicable.
5. **Run tests**: `go test ./...`
6. **Lint your code**: We follow standard Go idioms.
7. **Commit** your changes (`git commit -m 'feat: add support for X'`).
8. **Push** to your fork (`git push origin feature/cool-new-thing`).
9. **Open a Pull Request** against the `main` branch.

## 🏗️ Development Setup

```bash
# Clone and build
git clone https://github.com/your-username/simon.git
cd simon
go build -o simon cmd/simon/main.go

# Run tests
go test -v ./...
```

## 🗺️ Roadmap

Check out the [Specification](.roady/spec.yaml) managed by Roady to see our planned V1 features.

---

*Thank you for your contributions!*
