# 🐚 GoSh - Cross-Platform Go Shell

**Version**: 0.1.0-beta.4
**Status**: Beta - Code Quality & API Refinement (breaking changes in this release)
**License**: MIT

A modern, cross-platform shell written in Go with beautiful TUI and native script execution.

## ✨ Features

### 🎨 **4 Beautiful UI Modes**
- **Classic** - Traditional bash-like interface
- **Warp** - Modern, polished UI
- **Compact** - Minimal, space-efficient
- **Chat** - Telegram-like conversational interface

### 📜 **Command History** (v0.1.0-alpha.2)
- **Persistent History**: Commands automatically save to `~/.gosh_history`
- **Auto-Load**: History loads on shell startup
- **Up/Down Navigation**: Navigate through command history with arrow keys
- **Smart Deduplication**: Consecutive identical commands are automatically deduplicated
- **Configurable Limit**: History respects 10,000 command limit (configurable)

### 🚀 **Command Execution**
- **External Commands**: Execute any system command via os/exec
- **Interactive Mode**: Full TTY support for programs like vim, ssh, nano
- **Native Shell Scripts**: Execute .sh/.bash scripts natively using mvdan.cc/sh (no bash.exe dependency on Windows!)
- **Script Detection**: Automatically detects .sh, .bash, .bat, .cmd, .ps1 scripts

### 🎨 **Syntax Highlighting**
- Inline ANSI syntax highlighting
- Commands, options, and arguments color-coded
- Real-time highlighting as you type

### 🔧 **Git Integration**
- Current branch displayed in prompt
- Dirty status indicator
- Works in any git repository

### ⌨️ **Input & Navigation**
- **Tab Completion**: Basic command and file completion
- **Multi-line Input**: Alt+Enter for multiline commands
- **Viewport Scrolling**: PgUp/PgDn, Mouse Wheel support
- **Auto-scroll**: Automatically scroll to bottom on new output
- **Visual Help Overlay**: Press F1 or ? for keyboard shortcuts reference (since beta.3)
- **UI Mode Switching**: Alt+1-4 hotkeys to switch modes on-the-fly (since beta.2)

## 📦 Installation

### Prerequisites
- Go 1.25 or higher
- Git (for installation from source)

### Build from Source
```bash
git clone https://github.com/grpmsoft/gosh.git
cd gosh
go build -o gosh ./cmd/gosh

# On Windows:
go build -o gosh.exe ./cmd/gosh
```

### Quick Start
```bash
# Run GoSh
./gosh

# Or on Windows:
gosh.exe
```

## 🎯 Usage

### Basic Commands
```bash
# Execute any system command
ls -la
git status
npm install

# Run interactive programs
vim myfile.txt
ssh user@server
nano config.yml

# Execute shell scripts (native POSIX support!)
./my-script.sh
```

### History Features
```bash
# Navigate history
# Use Up/Down arrow keys to browse previous commands

# History is automatically saved to ~/.gosh_history
# History loads automatically on shell startup
```

### UI Modes & Keyboard Shortcuts
```bash
# Get help
F1 or ?          # Open visual help overlay (ESC to close)
help             # Show built-in commands

# Switch UI modes
Alt+1            # Classic mode (traditional bash-like)
Alt+2            # Warp mode (modern polished)
Alt+3            # Compact mode (minimal space)
Alt+4            # Chat mode (telegram-like)

# Or use :mode command
:mode            # Show current UI mode
:mode classic    # Switch to Classic
:mode warp       # Switch to Warp
:mode compact    # Switch to Compact
:mode chat       # Switch to Chat
```

## ⚙️ Configuration

### History Configuration
History is stored in `~/.gosh_history` and automatically managed.

**Default Settings**:
- Max size: 10,000 commands
- Auto-save: Enabled
- Deduplication: Consecutive duplicates removed

### .goshrc Configuration (v0.1.0-alpha.4)
GoSh supports `.goshrc` configuration file for customization:
- **Location**: `~/.goshrc` (home directory)
- **Aliases**: Define custom command aliases
- **Environment**: Set environment variables
- **Auto-load**: Loaded automatically on shell startup

**Example .goshrc**:
```bash
# Aliases
alias ll='ls -la'
alias gs='git status'

# Environment variables
export EDITOR=vim
export GOPATH=$HOME/go
```

## 🏗️ Architecture

GoSh is built using modern software architecture patterns:

- **DDD (Domain-Driven Design)**: Rich domain models with business logic
- **Hexagonal Architecture**: Clean separation of concerns
- **Bubbletea TUI Framework**: Elm Architecture (Model-View-Update)
- **Native POSIX Shell**: mvdan.cc/sh v3.12.0 for cross-platform script execution

### Project Structure
```
gosh/
├── cmd/gosh/                    # Entry point
├── internal/
│   ├── domain/                  # Domain models (Session, History, Process)
│   │   ├── history/            # History aggregate (30+ tests)
│   │   ├── session/            # Session aggregate
│   │   └── process/            # Process management
│   ├── application/             # Use cases (10+ tests)
│   │   └── history/            # History use cases
│   ├── infrastructure/          # External adapters (15+ tests)
│   │   ├── history/            # File persistence
│   │   └── executor/           # Command execution
│   └── interfaces/              # UI layer
│       └── repl/               # Bubbletea REPL
└── docs/                        # Documentation
    └── dev/                     # Development docs
```

## 🧪 Testing

GoSh has comprehensive test coverage:

```bash
# Run all tests
go test ./...

# Run with coverage
go test -cover ./...

# Run specific package
go test ./internal/domain/history/...
```

**Current Test Coverage**:
- Domain layer: 95%+
- Application layer: 90%+
- Infrastructure layer: 80%+
- Total: 130+ tests passing across 21 packages

## 🗺️ Roadmap

### Current Version: v0.1.0-beta.4 ✅
**Status**: In Development - Code Quality & API Refinement (breaking changes in this release)

**What's New in beta.4**:
- [x] Comprehensive linter cleanup (524→46 errors, 91% reduction)
- [x] Security fixes (gosec): file permissions, secure defaults
- [x] API improvements: renamed stuttering types for Go idioms
- [x] Code quality: removed dead code, fixed deprecations
- [x] Documentation: 385 comment improvements, architecture docs

**Previously Implemented** (alpha.1 → beta.3):
- [x] History persistence and navigation (alpha.1-2)
- [x] Built-in commands: cd, pwd, export, unset, type, alias (alpha.3-4)
- [x] .goshrc configuration file (alpha.4)
- [x] Unix-style pipelines (|) (alpha.5)
- [x] File redirections (>, >>, <, 2>) (alpha.6)
- [x] Background jobs (&, jobs, fg, bg) (beta.1)
- [x] Glob patterns (*, ?, [], {}) (beta.2)
- [x] Visual help overlay (F1/?) (beta.3)
- [x] 4 UI modes: Classic, Warp, Compact, Chat
- [x] 130+ tests, CI/CD on 3 platforms

### Next: v0.1.0-rc.1 (After beta.4)
- [ ] Address community feedback from beta.4
- [ ] Fix critical bugs reported by users
- [ ] Performance optimizations if needed
- [ ] Final polish before stable release

### Future: v0.1.0 (Stable Release)
- [ ] No critical bugs
- [ ] Community-validated features
- [ ] Complete documentation
- [ ] Production-ready

### Post-Release: v0.2.0
Based on community feedback:
- Ctrl+R fuzzy search UI
- Enhanced scripting support
- Advanced configuration
- Features requested by community

See [RELEASE_ROADMAP.md](docs/dev/RELEASE_ROADMAP.md) for detailed roadmap.

## 📝 Changelog

See [CHANGELOG.md](CHANGELOG.md) for detailed version history.

## 🤝 Contributing

Contributions are welcome! This project follows:

- **Semantic Versioning 2.0.0**
- **Conventional Commits** (feat:, fix:, docs:, etc.)
- **Test-Driven Development** (TDD)
- **Domain-Driven Design** principles

### Development Workflow
```bash
# Clone repository
git clone https://github.com/grpmsoft/gosh.git
cd gosh

# Run tests
go test ./...

# Build
go build -o gosh ./cmd/gosh

# Run
./gosh
```

## 📄 License

MIT License - see [LICENSE](LICENSE) for details.

Third-party library licenses - see [NOTICE](NOTICE) for details.

## 🙏 Acknowledgments

### Special Thanks

**Professor Ancha Baranova** - This project would not have been possible without her invaluable help and support. Her assistance was crucial in bringing GoSh to life.

### Open Source Libraries

- **Bubbletea** - Charm's excellent TUI framework
- **mvdan.cc/sh** - Native POSIX shell interpreter
- **Lipgloss** - Terminal styling library
- **Bubbles** - TUI components

## 📞 Support

- **Issues**: [GitHub Issues](https://github.com/grpmsoft/gosh/issues)
- **Discussions**: [GitHub Discussions](https://github.com/grpmsoft/gosh/discussions)
- **Documentation**: [docs/](docs/)

---

**Built with ❤️ using Go and modern software architecture practices**

*GoSh aims to be the best cross-platform shell with native script execution and beautiful UI* 🚀
