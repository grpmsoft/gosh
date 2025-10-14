# 🤝 Contributing to GoSh

Thank you for your interest in contributing to GoSh! This document provides guidelines and information for contributors.

---

## 📋 Table of Contents
- [Development Philosophy](#-development-philosophy)
- [Quick Start](#-quick-start)
- [Code Formatting](#-code-formatting)
- [Testing Requirements](#-testing-requirements)
- [Project Structure](#-project-structure)
- [Code Style](#-code-style)
- [Git Workflow](#-git-workflow)
- [Pull Request Process](#-pull-request-process)

---

## 🎯 Development Philosophy

GoSh follows modern software engineering practices:

- **Test-Driven Development (TDD)**: Write tests before implementation
- **Domain-Driven Design (DDD)**: Rich domain models with business logic
- **Hexagonal Architecture**: Clean separation of concerns
- **Semantic Versioning 2.0.0**: Clear version semantics
- **Conventional Commits**: Structured commit messages

---

## 🚀 Quick Start

### Prerequisites
- **Go**: 1.25 or higher
- **Git**: For version control
- **Make**: For automation tasks

### Setup Development Environment

```bash
# Clone repository
git clone https://github.com/grpmsoft/gosh.git
cd gosh

# Install dependencies
go mod download

# Run tests (ensure everything works)
make test

# Build binary
make build

# Run GoSh
./gosh
```

---

## 🎨 Code Formatting

### ⚠️ CRITICAL: Formatting is Mandatory

**Code formatting is strictly enforced and automatically checked in CI.**

### Before Submitting ANY Code

**ALWAYS run these commands before committing:**

```bash
# Format all Go code
make fmt

# Verify formatting (CI check)
make fmt-check
```

### Why Formatting Matters

1. **Consistency**: Uniform code style across the entire codebase
2. **Readability**: Makes code easier to review and maintain
3. **CI/CD**: Unformatted code will **FAIL** CI checks

### What `make fmt` Does

```bash
gofmt -w -s .    # Format all Go files
go mod tidy      # Clean up module dependencies
```

### CI Enforcement

Our CI pipeline includes:
```bash
make ci  # Runs: fmt-check, test, lint
```

If your code is not properly formatted:
- ❌ CI will fail
- ❌ PR cannot be merged
- ⚠️ You'll see errors like: "ERROR: The following files are not formatted"

---

---

## 📁 Project Structure

```
gosh/
├── cmd/gosh/              # Application entry point
├── internal/
│   ├── domain/            # Domain layer (business logic)
│   │   ├── history/       # History aggregate
│   │   ├── session/       # Session aggregate
│   │   └── process/       # Process entity
│   ├── application/       # Application layer (use cases)
│   ├── infrastructure/    # Infrastructure layer (adapters)
│   └── interfaces/        # Interface layer (UI/CLI)
├── docs/                  # Documentation
├── ROADMAP.md            # Development roadmap
├── CHANGELOG.md          # Version history
└── README.md             # Project overview
```

### Where to Put New Code

| What | Where | Why |
|------|-------|-----|
| Business logic | `internal/domain/` | Pure logic, no I/O |
| Orchestration | `internal/application/` | Coordinate domain + infra |
| File/DB/Network | `internal/infrastructure/` | External dependencies |
| UI/CLI | `internal/interfaces/` | User interaction |

---

## 🎨 Code Style

### Naming Conventions
- **Exported** (public): `PascalCase`
- **Unexported** (private): `camelCase`
- **Constants**: `UPPER_SNAKE_CASE` or `PascalCase`
- **Test functions**: `TestTypeName_MethodName`

### Comments
- Document all exported functions, types, and methods
- Use godoc conventions
- Explain "why" not "what" in complex logic

---

## 🔀 Git Workflow

### Branch Strategy
- `main` - Stable releases
- `feature/xxx` - Feature branches
- `fix/xxx` - Bug fix branches

### Commit Messages

Follow [Conventional Commits](https://www.conventionalcommits.org/):

```
<type>: <description>

[optional body]

[optional footer]
```

**Types**: `feat`, `fix`, `docs`, `style`, `refactor`, `test`, `chore`, `perf`

**Examples**:
```bash
feat: add history search with fuzzy matching
fix: correct viewport height in Classic mode
docs: update roadmap with v0.2.0 tasks
```

### Development Workflow

```bash
# 1. Create feature branch
git checkout -b feature/your-feature

# 2. Make changes with tests

# 3. Format and test
make fmt
make test

# 4. Commit
git add .
git commit -m "feat: your feature"

# 5. Push
git push origin feature/your-feature
```

---

## 🧪 Testing Requirements

### Coverage Standards

- **Minimum 70% overall coverage**
- **Minimum 90% for business logic** (domain/ and application/ layers)

### Running Tests

```bash
# Run all tests
make test

# Run with coverage report
make test-coverage

# Run with race detector
make test-race

# Open coverage report
# coverage.html will be generated
```

### Test Structure

```
internal/
├── domain/
│   ├── history/
│   │   ├── history.go
│   │   └── history_test.go          # Unit tests
│   ├── session/
│   │   ├── session.go
│   │   └── session_test.go
```

### Test Naming Convention

```go
// Unit test
func TestHistory_Add(t *testing.T) { ... }

// Table-driven test
func TestHistory_Add_EdgeCases(t *testing.T) {
    tests := []struct {
        name string
        // ...
    }{ ... }
}
```

---

## 📝 Commit Guidelines

### Commit Message Format

```
<type>: <subject>

<body>

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude <noreply@anthropic.com>
```

### Types

- `feat:` - New feature
- `fix:` - Bug fix
- `docs:` - Documentation changes
- `style:` - Formatting (no code changes)
- `refactor:` - Code restructuring
- `test:` - Adding tests
- `chore:` - Maintenance tasks
- `perf:` - Performance improvements

### Examples

```bash
# Good commits
git commit -m "feat: add Ctrl+R fuzzy search for history"
git commit -m "fix: resolve history deduplication bug"
git commit -m "docs: update README with installation instructions"

# Bad commits (avoid)
git commit -m "fix stuff"
git commit -m "WIP"
git commit -m "asdfasdf"
```

---

## 🔀 Pull Request Process

### Before Creating PR

1. ✅ Code is formatted (`make fmt`)
2. ✅ All tests pass (`make test`)
3. ✅ Linter passes (`make lint`)
4. ✅ CI checks pass (`make ci`)
5. ✅ Documentation updated
6. ✅ Commit messages follow guidelines

### PR Template

```markdown
## Summary
Brief description of changes

## Changes
- Added X
- Modified Y
- Fixed Z

## Testing
- Unit tests added/updated
- Integration tests added/updated
- Manual testing performed

## Checklist
- [ ] Code formatted (`make fmt`)
- [ ] Tests pass (`make test`)
- [ ] Linter passes (`make lint`)
- [ ] Documentation updated
```

### Review Process

1. Create PR against `main` branch
2. CI checks must pass (formatting, tests, linting)
3. Code review by maintainers
4. Address review comments
5. Squash merge when approved

---

## 🚨 Common Pitfalls

### ❌ Don't

- Don't commit without running `make fmt`
- Don't skip tests
- Don't ignore linter warnings
- Don't commit unformatted code
- Don't commit TODO comments without tracking
- Don't hardcode secrets/credentials

### ✅ Do

- Always format code before committing
- Write tests for new features
- Update documentation
- Follow existing code patterns
- Use meaningful variable names
- Add comments for complex logic
- Use context for cancellation

---

## 🔧 Development Tools

### Required Tools

```bash
# Go 1.25+
go version

# golangci-lint v2.5.0+
golangci-lint version

# gofmt (included with Go)
gofmt -version

# make (for Makefile targets)
make --version
```

### Recommended IDE Setup

#### VS Code
```json
{
  "go.formatTool": "gofmt",
  "editor.formatOnSave": true,
  "go.lintTool": "golangci-lint",
  "go.lintOnSave": "workspace"
}
```

#### GoLand
- Enable "Go fmt on save"
- Enable "golangci-lint" integration

---

---

## 🐛 Reporting Bugs

### Bug Report Template
```markdown
**GoSh Version**: v0.1.0-beta.4
**Platform**: Windows 11 / Linux Ubuntu / macOS 14
**Go Version**: 1.25

**Description**: Clear description of the bug

**Steps to Reproduce**:
1. Run `gosh`
2. Execute `command xyz`
3. Observe error

**Expected**: What should happen
**Actual**: What actually happens
**Logs**: If applicable
```

---

## 💡 Feature Requests

### Feature Request Template
```markdown
**Feature**: Clear feature name
**Problem**: What problem does this solve?
**Solution**: How should it work?
**Use Case**: Real-world scenario
```

---

## 📚 Resources

- [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- [Effective Go](https://golang.org/doc/effective_go)
- [Project README](README.md)
- [Development Roadmap](ROADMAP.md)
- [Changelog](CHANGELOG.md)

---

## 🆘 Getting Help

- **Issues**: [GitHub Issues](https://github.com/grpmsoft/gosh/issues)
- **Discussions**: [GitHub Discussions](https://github.com/grpmsoft/gosh/discussions)
- **Documentation**: Check `docs/` directory

---

## 🎉 Recognition

Contributors will be recognized in:
- CHANGELOG.md for significant contributions
- GitHub contributors page
- Release notes

---

## 📄 License

By contributing, you agree that your contributions will be licensed under the MIT License.

---

**Thank you for contributing to GoSh!** 🚀

*Last updated: 2025-10-14*
