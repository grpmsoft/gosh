# 🗺️ GoSh Development Roadmap

This document outlines the planned development roadmap for GoSh, organized by release milestones.

---

## 📍 Current Version: v0.1.0-beta.7 ✅

**Status**: Released (2025-10-14)
**Focus**: Git-Flow + Cross-Platform Testing

### What's New
- ✅ **macOS CI fix**: Fixed path comparison in pwd test (cross-platform compatibility)
- ✅ **Git-Flow implementation**: Release branches for safer releases (feature → release → CI → main → tag)
- ✅ **CI validation**: Tests run on release branches before merging to main
- ✅ **All platforms passing**: Linux, macOS, Windows CI tests successful
- ✅ **Documentation updates**: Complete git-flow workflow in CONTRIBUTING.md and RELEASE_PROCESS.md

### Previously in v0.1.0-beta.6
- ✅ Classic mode spinner fix, UX improvements
- ✅ Command echo to terminal, configurable separator
- ✅ gocritic tuning, dependency cleanup
- ✅ Linter errors resolved (47 → 0)

### Previously in v0.1.0-beta.4
- ✅ Comprehensive linter cleanup (524→46 errors, 91% reduction)
- ✅ Security fixes (gosec): file permissions, secure defaults
- ✅ API improvements: renamed stuttering types for Go idioms
- ✅ Code quality: removed dead code, fixed deprecations
- ✅ Documentation: 385 comment improvements, architecture docs
- ✅ REPL refactoring: split 1777-line monolith into 6 focused modules

### Previously Implemented (alpha.1 → beta.3)
- ✅ History persistence and navigation (alpha.1-2)
- ✅ Built-in commands: cd, pwd, export, unset, type, alias (alpha.3-4)
- ✅ .goshrc configuration file (alpha.4)
- ✅ Unix-style pipelines (`|`) (alpha.5)
- ✅ File redirections (`>`, `>>`, `<`, `2>`) with full FD support (alpha.6, beta.4)
- ✅ Background jobs (`&`, `jobs`, `fg`, `bg`) (beta.1)
- ✅ Glob patterns (`*`, `?`, `[]`, `{}`) (beta.2)
- ✅ Visual help overlay (F1/?) (beta.3)
- ✅ 4 UI modes: Classic, Warp, Compact, Chat
- ✅ 130+ tests, CI/CD on 3 platforms (Linux/macOS/Windows)

---

## 🔄 In Progress: v0.1.0-beta.8 (Phoenix TUI Integration)

**Status**: Active Development
**Focus**: Phoenix TUI v0.2.1 Migration + Interactive Commands

### What's New
- ✅ **Phoenix TUI v0.2.2**: Complete migration from Charm Bubbletea/Lipgloss
  - 10x performance (differential rendering, 29,000 FPS)
  - Perfect Unicode support (emoji, CJK width)
  - TTY Control System for interactive commands
- ✅ **Interactive Commands**: Universal ExecProcessWithTTY
  - All external commands run with proper TTY control (like bash)
  - vim, ssh, claude, python REPL — all work natively
  - Pipe-based CancelableReader (MSYS/mintty compatible)
- ✅ **Classic Mode**: No alt screen (true bash-like behavior)
- ✅ **Dependency cleanup**: Removed go.mod replace directives

---

## 🎯 Next: v0.1.0-rc.1

**Status**: Planning
**Focus**: Community Feedback & Bug Fixes

### Goals
- [ ] Gather and address community feedback
- [ ] Fix critical bugs reported by users
- [ ] Performance optimizations based on real-world usage
- [ ] Final polish before stable release
- [ ] Update documentation based on user questions

### Success Criteria
- Zero critical bugs
- Community validation of core features
- Performance meets expectations
- Documentation complete and clear

---

## 🚀 Target: v0.1.0 Stable

**Status**: Future
**Focus**: Production-Ready Stable Release

### Goals
- [ ] No critical bugs remaining
- [ ] All features from roadmap implemented
- [ ] Complete user and developer documentation
- [ ] Performance benchmarks published
- [ ] Production-ready for daily use

### Success Criteria
- Used by at least 100 users without major issues
- Test coverage > 85% across all layers
- Documentation covers all use cases
- Community considers it stable

---

## 🌟 Future: v0.2.0

**Status**: Planned
**Focus**: Enhanced User Experience

### Features Based on Community Feedback
- [ ] **Ctrl+R Fuzzy Search UI**: Visual fuzzy history search (like fzf)
- [ ] **Enhanced Scripting**: Improved POSIX compatibility
- [ ] **Advanced Configuration**: More .goshrc options
- [ ] **Plugin System**: Allow community extensions (research phase)
- [ ] **Performance Profiling**: Built-in performance metrics

### Community-Driven Features
- [ ] Features requested by users during v0.1.0 feedback period
- [ ] Platform-specific improvements (Windows/Linux/macOS)
- [ ] Integration with popular tools (git, docker, etc.)

---

## 🔮 Long-Term Vision

### Potential Features (Under Research)

#### Remote Shell Support (v0.3.0+)
- Session persistence without active SSH connection
- Inspired by tmux/screen + Mosh
- See: [docs/dev/REMOTE_SHELL_PROPOSAL.md](docs/dev/REMOTE_SHELL_PROPOSAL.md) (internal)

#### Advanced Scripting (v0.4.0+)
- Go-native scripting alongside POSIX
- Type-safe script execution
- Better error handling than bash

#### Multi-User Features (v0.5.0+)
- Shared sessions (pair programming)
- Permission management
- Collaborative debugging

#### Cross-Platform Improvements
- Windows native support (beyond MSYS2)
- Better PowerShell interop
- macOS-specific features

---

## 📊 Development Principles

### Quality Over Speed
- TDD (Test-Driven Development) mandatory
- Domain-Driven Design principles
- 90%+ test coverage for business logic

### Community-Driven
- Features prioritized by community feedback
- Transparent development process
- Open roadmap discussion

### Pure Go Philosophy
- Stay pure Go (no CGo) for cross-platform benefits
- Exception: Only if community demands critical feature
- See: [docs/dev/MVDAN_SH_LIMITATIONS_ANALYSIS.md](docs/dev/MVDAN_SH_LIMITATIONS_ANALYSIS.md) (internal)

### Semantic Versioning
- Strict adherence to SemVer 2.0.0
- Breaking changes only in major versions
- Clear migration guides for breaking changes

---

## 🤝 How to Influence the Roadmap

### Community Feedback
- Open GitHub Issues for feature requests
- Vote on existing issues with 👍 reactions
- Participate in GitHub Discussions

### Contributing
- See [CONTRIBUTING.md](CONTRIBUTING.md) for development guidelines
- Pick issues tagged with `good-first-issue`
- Propose new features via GitHub Discussions

### Testing & Feedback
- Try beta releases and report bugs
- Share use cases and workflows
- Suggest improvements based on real-world usage

---

## 📝 Roadmap Updates

This roadmap is a living document and will be updated based on:
- Community feedback and feature requests
- Technical discoveries and limitations
- Resource availability and priorities
- Real-world usage patterns

**Last Updated**: 2026-02-06
**Next Review**: After v0.1.0-rc.1 release

---

## 📞 Questions?

- **Issues**: [GitHub Issues](https://github.com/grpmsoft/gosh/issues)
- **Discussions**: [GitHub Discussions](https://github.com/grpmsoft/gosh/discussions)
- **Detailed Changelog**: [CHANGELOG.md](CHANGELOG.md)

---

*GoSh aims to be the best cross-platform shell with native script execution and beautiful UI* 🚀
