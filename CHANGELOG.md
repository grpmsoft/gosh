# Changelog

All notable changes to GoSh will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Planned
- Ctrl+R fuzzy search UI
- Gather community feedback on beta.3
- v0.1.0-rc.1 (after feedback collection)
- v0.1.0 stable release

## [0.1.0-beta.3.1] - 2025-10-12

### Fixed - Critical Keyboard Input Bug 🐛
- **Keyboard input**: Fixed 'b' and 'f' keys being intercepted for viewport scrolling
  - Issue: Keys 'b' and 'f' were bound to viewport navigation (Vim-style), preventing text input
  - Impact: Users on Alpine Linux (foot terminal) couldn't type words containing these letters
  - Solution: Removed 'f' and 'b' from viewport scroll shortcuts (PgUp/PgDn still work)
  - Reporter: Community user on Alpine Linux with foot terminal
  - Tests: All 130+ tests passing, keyboard input validated

### Technical
- Removed 'f' and 'b' from `case "pgup", "pgdown", "f", "b"` keyboard handler
- Viewport scrolling still available via PgUp/PgDn and mouse wheel
- Quick hotfix release to unblock beta testers

## [0.1.0-beta.3] - 2025-01-12

### Added - Help System Overhaul 🎨
- **Visual Help Overlay**: Beautiful keyboard shortcuts reference with F1 or ?
  - Centered modal window with rounded borders
  - Color-coded sections (Navigation, Input, UI Modes, Help, Exit)
  - Press ESC to close
  - Follows IBM CUA 1987 tradition (F1 = Help for 38 years!)
- **:mode Command**: Vim-style command for UI mode switching
  - `:mode` - Show current UI mode and available options
  - `:mode classic|warp|compact|chat` - Switch to specified mode
  - Validates mode names and shows helpful error messages
  - Works alongside existing Ctrl+F5-F8 hotkeys

### Changed - Keyboard Shortcuts 🎹
- **Help keys reorganized**:
  - F1 → Open help overlay (was: switch to Classic mode)
  - ? → Open help overlay (new, modern TUI pattern from k9s/lazygit)
  - ESC → Close help overlay (standard modal close pattern)
- **UI Mode switching moved**:
  - Ctrl+F5 → Classic mode (was: F1)
  - Ctrl+F6 → Warp mode (was: F2)
  - Ctrl+F7 → Compact mode (was: F3)
  - Ctrl+F8 → Chat mode (was: F4)
  - Rationale: Frees F2-F8 for future frequent features, avoids Ctrl+F4=Close Tab conflict
- **Updated help command**: Now shows `:mode` usage when UI mode switching is enabled

### Technical
- Research-driven keyboard design (analyzed IBM CUA, k9s, lazygit, Vim patterns)
- Chose `:mode` over `/mode` following Vim/k9s tradition for meta-commands
- Help overlay rendered using lipgloss with center placement
- All shortcuts documented in visual overlay and `help` command
- Welcome message updated to reflect new shortcuts
- 0 linter warnings, 130+ tests passing

### Improved UX
- F1 and ? are more discoverable than F5-F8 for help
- Visual help overlay is more informative than text-based `help` command
- `:mode` provides fallback for users who prefer typing over hotkeys
- ESC as universal "close/cancel" pattern

## [0.1.0-beta.2] - 2025-10-12

### Repository Publication 🎉
- **Published**: GoSh repository is now public at https://github.com/grpmsoft/gosh
- **Status**: First public beta release
- **Professional infrastructure**: Complete CI/CD, linting, testing, documentation

### Features
- **Glob Pattern Expansion**: Full support for Unix-style filename wildcards (`*.go`, `test?.txt`, `file[1-3].txt`)
- **Background Jobs**: Full Unix-style job control (`&`, `jobs`, `fg`, `bg`)
- **Pipelines**: Command pipelines (`ls | grep gosh`)
- **Redirections**: IO redirections (`>`, `>>`, `<`, `2>`)
- **Aliases**: Command shortcuts with `.goshrc` support
- **Built-in Commands**: cd, pwd, echo, export, unset, env, alias, type, jobs, fg, bg
- **Command History**: Persistent history with Up/Down navigation
- **4 UI Modes**: Classic, Warp, Compact, Chat
- **Native Script Execution**: .sh/.bash scripts via mvdan.cc/sh (no bash.exe dependency on Windows)
- **Git Integration**: Branch and dirty status in prompt
- **Syntax Highlighting**: Real-time ANSI highlighting
- **Tab Completion**: Command and file completion

### Infrastructure
- MIT License
- CI/CD on Linux/macOS/Windows
- GoReleaser for automated releases
- 130+ tests, 0 linter warnings

---

*Development history omitted for brevity. Beta.2 was the first public release.*

[unreleased]: https://github.com/grpmsoft/gosh/compare/v0.1.0-beta.3.1...HEAD
[0.1.0-beta.3.1]: https://github.com/grpmsoft/gosh/compare/v0.1.0-beta.3...v0.1.0-beta.3.1
[0.1.0-beta.3]: https://github.com/grpmsoft/gosh/compare/v0.1.0-beta.2...v0.1.0-beta.3
[0.1.0-beta.2]: https://github.com/grpmsoft/gosh/releases/tag/v0.1.0-beta.2
