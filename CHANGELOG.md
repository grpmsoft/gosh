# Changelog

All notable changes to GoSh will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Planned
- Ctrl+R fuzzy search UI

## [0.1.0-beta.2] - 2025-01-11

### Added
- **Glob Pattern Expansion**: Full support for Unix-style filename wildcards
- **Wildcard patterns**: `*` (any characters), `?` (single character), `[]` (character sets/ranges)
- **Pattern examples**:
  - `ls *.go` - list all .go files
  - `cat test?.txt` - match test1.txt, test2.txt (single character)
  - `rm file[123].txt` - match file1.txt, file2.txt, file3.txt
  - `cat file[1-3].txt` - range syntax
- **Bash-like behavior**: Error if no matches found (prevents accidental operations)
- **Cross-platform**: Uses Go's `filepath.Glob()` for compatibility
- **Mixed arguments**: Glob and literal arguments work together (`echo *.go hello`)

### Technical
- Added `expandGlobs()` function in parser
- Pattern detection with `containsGlobPattern()` helper
- Glob expansion happens before command creation
- 13 comprehensive tests including integration tests
- Error handling for invalid patterns and no-match cases

## [0.1.0-beta.1] - 2025-01-11

### Added
- **Background Jobs (&)**: Full support for Unix-style background job execution
- **Job tracking**: Jobs tracked with automatic numbering (%1, %2, etc.)
- **jobs command**: List all background jobs with status (`[1] + [Running] sleep 10 &`)
- **fg command**: Bring background job to foreground (`fg %1` or just `fg`)
- **bg command**: Resume stopped job in background (`bg %1` or just `bg`)
- **Job notifications**: Automatic display when background jobs complete (`[1] Done    sleep 10 &`)
- **Job domain model**: Rich domain entities (Job, JobManager) with complete lifecycle
- **Process monitoring**: Background goroutines monitor process completion asynchronously
- **State synchronization**: Job state syncs with Process state automatically

### Technical
- Created Job aggregate (Entity) with states: Running, Stopped, Completed, Failed
- Created JobManager domain service with thread-safe job tracking (RWMutex)
- Integrated JobManager into Session aggregate
- Added 48 comprehensive tests for Job domain (includes concurrency tests)
- Modified OSCommandExecutor to spawn monitoring goroutines for background processes
- File handle management: Background jobs close files in monitoring goroutine
- REPL checks for completed jobs before each prompt
- Jobs automatically removed from tracking after completion
- Support for %n and n syntax for job numbers
- Default behavior: operate on most recent job if no argument provided

## [0.1.0-alpha.6] - 2025-01-11

### Added
- **IO Redirections (>, >>, <, 2>)**: Complete file redirection support
- **Output redirection (>)**: Write stdout to file with overwrite (`echo hello > output.txt`)
- **Append redirection (>>)**: Append stdout to file (`cat data.txt >> log.txt`)
- **Input redirection (<)**: Read stdin from file (`cat < input.txt`)
- **Error redirection (2>)**: Redirect stderr to file (`ls nonexistent 2> error.log`)
- **Multiple redirections**: Combine multiple operators (`cat < input.txt > output.txt`)
- **Comprehensive tests**: 5 test cases passing (output, input, error, multiple, error handling)

### Technical
- Implemented handleRedirections() in OSCommandExecutor with proper file lifecycle management
- File handle cleanup using defer for all opened redirection files
- Integration with REPL via OSCommandExecutor for single commands
- Redirection support for both single commands and pipelines
- Error handling for nonexistent input files and permission issues
- Note: Windows/MSYS has file descriptor inheritance limitations with O_APPEND flag (functionality works, tests skipped)

## [0.1.0-alpha.5] - 2025-01-11

### Added
- **Basic Pipelines (|)**: Full support for command pipelines (`ls | grep gosh`, `echo test | wc -l`)
- **OSPipelineExecutor**: Infrastructure layer executor for piping stdout→stdin between commands
- **Pipeline domain model**: Value Object with Commands(), Length(), First(), Last(), At() methods
- **Parser integration**: Pipe operator detection and Pipeline object creation
- **REPL integration**: Pipeline execution through integrated executor
- **Error propagation**: Proper handling of failures in pipeline chains
- **Comprehensive tests**: 6 test cases covering simple pipes, multi-stage pipes, error handling, large output

### Technical
- Integrated OSPipelineExecutor into REPL Model
- Used `os/exec` StdoutPipe() for atomic pipe connection
- Last process in chain contains final output (stdout/stderr)
- Processes track state (Created→Running→Completed/Failed)
- All 70+ tests passing including new pipeline tests

## [0.1.0-alpha.4] - 2025-01-11

### Added
- **Aliases support**: Create custom command shortcuts with `alias name='command'`
- **Alias management**: `alias` (list all), `alias name` (show specific), `unalias name` (remove)
- **Recursive alias expansion**: Aliases can reference other aliases (with cycle detection)
- **.goshrc file**: Auto-load aliases and environment variables from `~/.goshrc` on startup
- **Environment variables in .goshrc**: Define environment using `export KEY=value`
- **Multi-terminal safe history**: Atomic append operations prevent history corruption
- **History.Append() method**: Efficient incremental history updates (O_APPEND flag)
- **Concurrent access tests**: 100 simultaneous writes verified safe
- **Warp mode viewport fix**: Correct viewport height recalculation on UI mode switch

### Changed
- History now uses Append() instead of Save() for better performance and safety
- Session tracks both env vars and aliases in memory
- AddToHistoryUseCase uses atomic append for file safety

### Technical
- GoshrcService for .goshrc parsing and saving
- MockHistoryRepository updated with Append support
- UI mode switching now recalculates viewport height correctly
- All 70+ tests passing

## [0.1.0-alpha.3] - 2025-01-11

### Added
- **Built-in Commands Domain Layer**: Rich domain models for cd, pwd, export, unset, type
- **cd with ~ expansion**: `cd ~` and `cd ~/path` now work correctly
- **cd - (previous directory)**: Navigate to previous directory with `cd -`
- **export validation**: Variable names validated (must start with letter/underscore)
- **type with aliases**: `type` command now checks aliases in addition to builtins and externals
- **Session previous directory**: Session tracks previous directory for `cd -`

### Changed
- Moved builtin command logic from Infrastructure to Domain layer (DDD compliance)
- BuiltinExecutor now delegates to domain commands instead of implementing logic
- Improved export command: strips quotes from values automatically
- Enhanced error messages for builtin commands

### Technical
- Created `internal/domain/builtins/` package with 5 command files
- 40+ unit tests for builtin commands (100% coverage)
- Session.ChangeDirectory() now tracks previous directory
- All tests passing (100+ total across project)

## [0.1.0-alpha.2] - 2025-01-11

### Added
- **History persistence**: Commands now save to `~/.gosh_history` automatically
- **History auto-load**: History loads from file on shell startup
- **Up/Down navigation**: Navigate through command history with arrow keys using History.Navigator
- **Deduplication**: Consecutive identical commands are automatically deduplicated
- **Max size limit**: History respects 10,000 command limit (configurable)

### Changed
- Session now uses rich History domain model instead of simple `[]string`
- REPL uses History.Navigator for arrow key navigation
- History operations use Use Cases (LoadHistoryUseCase, AddToHistoryUseCase)

### Technical
- Integrated History aggregate into Session domain (DDD)
- Wired FileHistoryRepository into REPL initialization
- Replaced old history implementation with new domain model
- All 60+ tests passing

## [0.1.0-alpha.1] - 2025-01-11

### Added
- **History domain model**: Rich domain model with 30+ tests
- **History persistence**: FileHistoryRepository with file operations (15 tests)
- **History use cases**: SearchHistoryUseCase, AddToHistoryUseCase, LoadHistoryUseCase, ClearHistoryUseCase (10 tests)
- **Tilde expansion**: `~/.gosh_history` path support
- **Special characters**: Proper handling in history file

### Technical
- Implemented complete DDD architecture for History (Domain/Application/Infrastructure layers)
- TDD Red → Green → Refactor cycle for all History features
- Test coverage: 95%+ domain, 90%+ application, 80%+ infrastructure

## [0.0.0-alpha] - 2025-01-10

### Added
- **Initial release**: GoSh - Cross-platform Go Shell
- **4 UI modes**: Classic (bash-like), Warp (modern), Compact (minimal), Chat (telegram-like)
- **Command execution**: External commands via os/exec
- **Interactive mode**: TTY programs (vim, ssh, etc.) via tea.ExecProcess
- **Native shell scripts**: .sh/.bash execution via mvdan.cc/sh (no bash.exe dependency)
- **Syntax highlighting**: Inline ANSI highlighting for commands, options, arguments
- **Git integration**: Branch and dirty status in prompt
- **Tab completion**: Basic command and file completion
- **Multi-line input**: Alt+Enter for multiline commands
- **Viewport scrolling**: PgUp/PgDn, Mouse Wheel support
- **Auto-scroll**: Automatically scroll to bottom on new output

### Technical
- Bubbletea TUI framework (Elm Architecture)
- DDD + Hexagonal architecture
- Session management with Environment, Variables, Aliases
- Cross-platform path handling with filepath package
- Native POSIX shell interpreter (mvdan.cc/sh v3.12.0)

[unreleased]: https://github.com/grpmsoft/gosh/compare/v0.1.0-beta.2...HEAD
[0.1.0-beta.2]: https://github.com/grpmsoft/gosh/compare/v0.1.0-beta.1...v0.1.0-beta.2
[0.1.0-beta.1]: https://github.com/grpmsoft/gosh/compare/v0.1.0-alpha.6...v0.1.0-beta.1
[0.1.0-alpha.6]: https://github.com/grpmsoft/gosh/compare/v0.1.0-alpha.5...v0.1.0-alpha.6
[0.1.0-alpha.5]: https://github.com/grpmsoft/gosh/compare/v0.1.0-alpha.4...v0.1.0-alpha.5
[0.1.0-alpha.4]: https://github.com/grpmsoft/gosh/compare/v0.1.0-alpha.3...v0.1.0-alpha.4
[0.1.0-alpha.3]: https://github.com/grpmsoft/gosh/compare/v0.1.0-alpha.2...v0.1.0-alpha.3
[0.1.0-alpha.2]: https://github.com/grpmsoft/gosh/compare/v0.1.0-alpha.1...v0.1.0-alpha.2
[0.1.0-alpha.1]: https://github.com/grpmsoft/gosh/compare/v0.0.0-alpha...v0.1.0-alpha.1
[0.0.0-alpha]: https://github.com/grpmsoft/gosh/releases/tag/v0.0.0-alpha
