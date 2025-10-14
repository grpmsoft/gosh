# 🐚 GoSh - Cross-Platform Go Shell

**Version**: 0.1.0-beta.5
**Status**: Beta - Classic Mode Rendering Fixes
**License**: MIT

A modern, cross-platform shell written in Go with beautiful TUI and native script execution.

## ✨ Features

### 🎨 **4 Beautiful UI Modes**
- **Classic** - Traditional bash-like interface
- **Warp** - Modern, polished UI
- **Compact** - Minimal, space-efficient
- **Chat** - Telegram-like conversational interface

### 📜 **Command History**
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
- **Visual Help Overlay**: Press F1 or ? for keyboard shortcuts reference
- **UI Mode Switching**: Alt+1-4 hotkeys to switch modes on-the-fly

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

### .goshrc Configuration
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

## 🗺️ Project Status

**Current**: v0.1.0-beta.5 (Classic Mode Rendering Fixes)
**Next**: v0.1.0-rc.1 (Community Feedback)
**Target**: v0.1.0 Stable (Q2 2025)

See [ROADMAP.md](ROADMAP.md) for detailed development plan.

## 📝 Documentation

- **[ROADMAP.md](ROADMAP.md)** - Development roadmap and future plans
- **[CHANGELOG.md](CHANGELOG.md)** - Detailed version history
- **[CONTRIBUTING.md](CONTRIBUTING.md)** - How to contribute
- **[LICENSE](LICENSE)** - MIT License

## 🤝 Contributing

Contributions are welcome! See [CONTRIBUTING.md](CONTRIBUTING.md) for:
- Development setup and workflow
- Testing requirements (TDD)
- Code style guidelines
- Pull request process

Quick start:
```bash
git clone https://github.com/grpmsoft/gosh.git
cd gosh
make test && make build
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
