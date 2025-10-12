# Contributing to GoSh

Thank you for contributing to GoSh! This document provides guidelines for contributing to the project.

---

## 📋 Table of Contents
- [Code Formatting](#code-formatting)
- [Development Workflow](#development-workflow)
- [Testing Requirements](#testing-requirements)
- [Commit Guidelines](#commit-guidelines)
- [Pull Request Process](#pull-request-process)

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

## 🔄 Development Workflow

### 1. Setup

```bash
# Clone repository
git clone https://github.com/grpmsoft/gosh.git
cd gosh

# Install dependencies
go mod download
```

### 2. Create Feature Branch

```bash
git checkout -b feature/your-feature-name
```

### 3. Make Changes

- Write your code
- Add tests (minimum 70% coverage)
- **Format your code**: `make fmt`

### 4. Pre-Commit Checks

**MANDATORY before committing:**

```bash
# Full development workflow
make dev

# This runs: fmt, lint, test, build
```

Or individually:
```bash
make fmt        # Format code
make fmt-check  # Verify formatting
make test       # Run tests
make lint       # Run linter
make build      # Build binary
```

### 5. Commit Changes

```bash
git add .
git commit -m "feat: your feature description"

# Generated with Claude Code
# Co-Authored-By: Claude <noreply@anthropic.com>
```

### 6. Push and Create PR

```bash
git push origin feature/your-feature-name
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

## 📚 Resources

- [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- [Effective Go](https://golang.org/doc/effective_go)
- [Project README](README.md)
- [Release Roadmap](docs/dev/RELEASE_ROADMAP.md)
- [Testing Strategy](docs/dev/TESTING_STRATEGY.md)

---

## 🆘 Getting Help

### Issues

If you encounter problems:
1. Check existing issues: https://github.com/grpmsoft/gosh/issues
2. Create new issue with:
   - Clear description
   - Steps to reproduce
   - Expected vs actual behavior
   - Environment details (OS, Go version, etc.)

### Questions

For questions:
- Open a GitHub Discussion
- Check project documentation
- Review existing PRs for examples

---

## 📄 License

By contributing, you agree that your contributions will be licensed under the same license as the project (MIT).

---

*This document is mandatory reading for all contributors.*
*Last updated: 2025-01-12*
