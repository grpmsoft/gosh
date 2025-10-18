package repl

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/grpmsoft/gosh/internal/domain/command"
	"github.com/grpmsoft/gosh/internal/domain/config"
	"github.com/grpmsoft/gosh/internal/domain/process"
	"github.com/grpmsoft/gosh/internal/domain/session"
	"github.com/grpmsoft/gosh/internal/interfaces/parser"

	"github.com/phoenix-tui/phoenix/tea/api"
	"mvdan.cc/sh/v3/expand"
	"mvdan.cc/sh/v3/interp"
	"mvdan.cc/sh/v3/syntax"
)

const (
	extBash = ".bash"
	extSh   = ".sh"
)

// expandAliases recursively expands aliases in command.
// Returns expanded command or error on cyclic dependency.
func (m *Model) expandAliases(commandLine string, depth int) (string, error) {
	const maxDepth = 10 // Protection against infinite recursion

	// Check recursion depth
	if depth > maxDepth {
		return "", fmt.Errorf("alias expansion exceeded maximum depth (possible recursive alias)")
	}

	// Extract first word (command name)
	parts := strings.Fields(commandLine)
	if len(parts) == 0 {
		return commandLine, nil
	}

	cmdName := parts[0]

	// Check if first word is an alias
	aliasCommand, isAlias := m.currentSession.GetAlias(cmdName)
	if !isAlias {
		// Not an alias - return as is
		return commandLine, nil
	}

	// Expand alias
	// If command had arguments, add them to expanded alias
	var expandedCommand string
	if len(parts) > 1 {
		// Alias + original arguments
		args := strings.Join(parts[1:], " ")
		expandedCommand = aliasCommand + " " + args
	} else {
		// Only alias without arguments
		expandedCommand = aliasCommand
	}

	// Recursively expand aliases in result (in case of alias to alias)
	return m.expandAliases(expandedCommand, depth+1)
}

// executeCommand executes entered command.
func (m Model) executeCommand() (Model, api.Cmd) {
	// Get value from m.inputText (already synced with appropriate input component)
	// This correctly handles both single-line and multiline modes
	value := strings.TrimSpace(m.inputText)

	// Empty command
	if value == "" {
		return m, nil
	}

	// Reset completion
	m.completionActive = false
	m.completions = []string{}
	m.completionIndex = -1
	m.beforeCompletion = ""

	// Add to history via use case (auto-save if configured)
	if err := m.addToHistoryUC.Execute(value); err != nil {
		m.logger.Warn("Failed to add command to history", "error", err)
	}

	// Create new navigator to reset position
	m.historyNavigator = m.currentSession.NewHistoryNavigator()

	// Show command in output with prompt and syntax highlighting (ANSI codes only)
	// Classic mode: print directly to stdout (like bash)
	// Other modes: add to viewport buffer
	if m.Config.UI.Mode == config.UIModeClassic {
		// Render final command line WITHOUT cursor before freezing
		// CRITICAL: In multiline mode, we need to clear ALL lines (not just current line!)

		if m.multilineMode {
			// Multiline: clear all lines before printing final command
			// Phoenix Terminal API - 10x faster on Windows Console! ⚡
			lines := m.shellTextArea.Lines()
			numLines := len(lines)

			// ClearLines() uses Windows Console API when available!
			_ = m.terminal.ClearLines(numLines)
		} else {
			// Single-line: just clear current line
			// Phoenix Terminal API - platform-optimized
			_ = m.terminal.ClearLine()
		}

		// Render prompt + command (no cursor!)
		fmt.Print(m.renderPromptForHistoryANSI())                      // Prompt
		fmt.Print(m.applySyntaxHighlight(value))                       // Command (no cursor!)
		fmt.Print("\n")                                                // Freeze and move to next line
	} else {
		// Add to viewport buffer
		m.addOutputRaw(m.renderPromptForHistoryANSI() + m.applySyntaxHighlight(value))
	}

	// Clear both input components
	m.shellInput.Reset()
	m.shellTextArea.Reset()

	// Reset to single-line mode
	m.multilineMode = false

	// Sync input state
	m.inputText = ""
	m.cursorPos = 0

	// Built-in exit command
	if value == "exit" || value == "quit" {
		m.quitting = true
		return m, api.Quit()
	}

	// Built-in clear command
	if value == "clear" || value == "cls" {
		m.output = make([]string, 0)
		return m, nil // Phoenix doesn't have ClearScreen, we handle it in View
	}

	// Built-in help command
	if value == "help" {
		m.showHelp()
		return m, nil
	}

	// Built-in :mode command for switching UI modes
	if strings.HasPrefix(value, ":mode ") || value == ":mode" {
		return m.handleModeCommand(value)
	}

	// Expand aliases (if command is an alias)
	expandedValue, err := m.expandAliases(value, 0)
	if err != nil {
		m.addOutputRaw("\033[31mError: " + err.Error() + "\033[0m")
		m.updateViewportContent()
		// FollowMode handles auto-scroll in render functions
		return m, nil
	}

	// Use expanded command for further execution
	value = expandedValue

	// Determine command type and execution method
	cmdName, cmdArgs := m.extractCommandName(value)

	// Check if this is a shell script
	scriptPath, isScript := m.isShellScript(cmdName)

	if isScript {
		// Shell script (.sh/.bash)
		if m.isInteractiveCommand(cmdName) {
			// Interactive script (with read, clear, menu) - via bash + tea.ExecProcess
			return m, m.execInteractiveCommand(value) //nolint:gocritic // evalOrder: Bubbletea MVU pattern requires this format
		}
		// Regular script - execute NATIVELY via mvdan.cc/sh
		m.executing = true
		return m, m.executeShellScriptNative(scriptPath, cmdArgs) //nolint:gocritic // evalOrder: Bubbletea MVU pattern requires this format
	}

	// Interactive command (vim, ssh, etc.) - via tea.ExecProcess
	if m.isInteractiveCommand(cmdName) {
		return m, m.execInteractiveCommand(value) //nolint:gocritic // evalOrder: Bubbletea MVU pattern requires this format
	}

	// Check if this is a builtin command (cd, export, unset)
	// They must execute synchronously in shell process
	if m.isBuiltinCommand(cmdName) {
		m.executing = true
		return m, m.execBuiltinCommand(value) //nolint:gocritic // evalOrder: Bubbletea MVU pattern requires this format
	}

	// Regular command - execute asynchronously with output capture
	m.executing = true
	return m, m.execCommandAsync(value) //nolint:gocritic // evalOrder: Bubbletea MVU pattern requires this format
}

// showHelp shows help (text version for help command).
func (m *Model) showHelp() {
	m.addOutputRaw("\033[1;33mGoSh - Built-in Commands:\033[0m")
	m.addOutputRaw("  cd <dir>     - Change directory")
	m.addOutputRaw("  pwd          - Print working directory")
	m.addOutputRaw("  echo <text>  - Print text")
	m.addOutputRaw("  export VAR=value - Set environment variable")
	m.addOutputRaw("  unset VAR    - Unset environment variable")
	m.addOutputRaw("  env          - Show environment")
	m.addOutputRaw("  type <cmd>   - Show command type")
	m.addOutputRaw("  alias        - List/create aliases")
	m.addOutputRaw("  unalias      - Remove alias")
	m.addOutputRaw("  jobs         - List background jobs")
	m.addOutputRaw("  fg, bg       - Foreground/background job control")
	m.addOutputRaw("  clear, cls   - Clear screen")
	m.addOutputRaw("  help         - Show this help")
	m.addOutputRaw("  exit, quit   - Exit shell")
	m.addOutputRaw("")

	// UI modes (if switching allowed)
	if m.Config.UI.AllowModeSwitching {
		m.addOutputRaw("\033[1;33mUI Mode Switching:\033[0m")
		m.addOutputRaw("  :mode        - Show current UI mode")
		m.addOutputRaw("  :mode <name> - Switch UI mode (classic/warp/compact/chat)")
		m.addOutputRaw("")
	}

	m.addOutputRaw("\033[1;36mPress F1 or ? for visual keyboard shortcuts\033[0m")
	m.updateViewportContent()
}

// execCommandAsync executes command in background.
func (m *Model) execCommandAsync(commandLine string) api.Cmd {
	return func() api.Msg {
		// Parse command
		cmd, pipe, err := parser.ParseCommandLine(commandLine)
		if err != nil {
			return commandExecutedMsg{
				err:      err,
				exitCode: 1,
			}
		}

		// Single command execution through OSCommandExecutor (handles redirections)
		if cmd != nil {
			// Prepare command (check scripts and add interpreter if needed)
			cmdName, cmdArgs := m.prepareCommand(cmd.Name(), cmd.Args())

			// Update command with prepared name and arguments
			preparedCmd, err := command.NewCommand(cmdName, cmdArgs, command.TypeExternal)
			if err != nil {
				return commandExecutedMsg{
					err:      err,
					exitCode: 1,
				}
			}

			// Copy redirections from original command
			for _, redir := range cmd.Redirections() {
				if err := preparedCmd.AddRedirection(redir); err != nil {
					return commandExecutedMsg{
						err:      err,
						exitCode: 1,
					}
				}
			}

			// Execute command via OSCommandExecutor (supports redirections)
			proc, err := m.commandExecutor.Execute(m.ctx, preparedCmd, m.currentSession)
			if err != nil {
				return commandExecutedMsg{
					err:      err,
					exitCode: 1,
				}
			}

			// Combine stdout and stderr
			output := proc.Stdout()
			if stderr := proc.Stderr(); stderr != "" {
				if output != "" {
					output += "\n"
				}
				output += stderr
			}

			// Determine exitCode and error
			exitCode := int(proc.ExitCode())
			var execErr error
			if proc.State() == process.StateFailed {
				execErr = proc.Error()
			}

			return commandExecutedMsg{
				output:   output,
				err:      execErr,
				exitCode: exitCode,
			}
		}

		if pipe != nil {
			// Execute pipeline via OSPipelineExecutor
			processes, err := m.pipelineExecutor.Execute(m.ctx, pipe.Commands(), m.currentSession)
			if err != nil {
				return commandExecutedMsg{
					err:      fmt.Errorf("pipeline execution failed: %w", err),
					exitCode: 1,
				}
			}

			// Get last process (it contains final output)
			if len(processes) == 0 {
				return commandExecutedMsg{
					output:   "",
					exitCode: 0,
				}
			}

			lastProcess := processes[len(processes)-1]

			// Combine stdout and stderr
			output := lastProcess.Stdout()
			if stderr := lastProcess.Stderr(); stderr != "" {
				if output != "" {
					output += "\n"
				}
				output += stderr
			}

			// Check last process status
			exitCode := int(lastProcess.ExitCode())
			var execErr error
			if lastProcess.State() == process.StateFailed {
				execErr = lastProcess.Error()
			}

			return commandExecutedMsg{
				output:   output,
				err:      execErr,
				exitCode: exitCode,
			}
		}

		return commandExecutedMsg{
			output:   "",
			exitCode: 0,
		}
	}
}

// prepareCommand prepares command for execution.
// Detects scripts and adds necessary interpreter (sh, bash, cmd, powershell).
func (m *Model) prepareCommand(cmdName string, cmdArgs []string) (finalCmd string, finalArgs []string) {
	// Check if command is a script file
	var scriptPath string

	// If path is relative or absolute (universal check for all OS)
	if strings.HasPrefix(cmdName, ".") || strings.ContainsRune(cmdName, filepath.Separator) || filepath.IsAbs(cmdName) {
		// Check file existence
		if filepath.IsAbs(cmdName) {
			scriptPath = cmdName
		} else {
			scriptPath = filepath.Join(m.currentSession.WorkingDirectory(), cmdName)
		}

		// Check if file exists
		if _, err := os.Stat(scriptPath); err != nil {
			// File doesn't exist - return as is (will result in exec error)
			return cmdName, cmdArgs
		}

		// Determine script type by extension
		ext := strings.ToLower(filepath.Ext(scriptPath))

		switch ext {
		case extSh, extBash:
			// Shell script - run via sh or bash
			// Check bash availability, otherwise use sh
			interpreter := "sh"
			if _, err := exec.LookPath("bash"); err == nil {
				interpreter = "bash"
			}
			// Git Bash on Windows understands Windows paths directly!
			// Pass path as is (don't convert)
			// Return: bash script.sh args...
			newArgs := append([]string{scriptPath}, cmdArgs...)
			return interpreter, newArgs

		case ".bat", ".cmd":
			// Windows batch - run via cmd /c
			newArgs := append([]string{"/c", scriptPath}, cmdArgs...)
			return "cmd", newArgs

		case ".ps1":
			// PowerShell script - run via powershell -File
			newArgs := append([]string{"-File", scriptPath}, cmdArgs...)
			return "powershell", newArgs
		}
	}

	// Not a script or not found - return as is
	return cmdName, cmdArgs
}

// isShellScript checks if command is a .sh/.bash script.
// Returns (path to script, true) if it is a script, otherwise ("", false).
func (m *Model) isShellScript(cmdName string) (string, bool) {
	// Check if this is a file (relative or absolute path)
	if !strings.HasPrefix(cmdName, ".") && !strings.ContainsRune(cmdName, filepath.Separator) && !filepath.IsAbs(cmdName) {
		// Doesn't look like a file path
		return "", false
	}

	// Get full path
	var scriptPath string
	if filepath.IsAbs(cmdName) {
		scriptPath = cmdName
	} else {
		scriptPath = filepath.Join(m.currentSession.WorkingDirectory(), cmdName)
	}

	// Check if file exists
	if _, err := os.Stat(scriptPath); err != nil {
		return "", false
	}

	// Check extension
	ext := strings.ToLower(filepath.Ext(scriptPath))
	if ext == extSh || ext == extBash {
		return scriptPath, true
	}

	return "", false
}

// extractCommandName extracts command name from string.
func (m *Model) extractCommandName(commandLine string) (cmdName string, cmdArgs []string) {
	// Parse command
	cmd, _, err := parser.ParseCommandLine(commandLine)
	if err != nil || cmd == nil {
		// Fallback: take first word
		parts := strings.Fields(commandLine)
		if len(parts) == 0 {
			return "", nil
		}
		return parts[0], parts[1:]
	}

	return cmd.Name(), cmd.Args()
}

// isInteractiveCommand determines if command requires interactive terminal.
func (m *Model) isInteractiveCommand(cmdName string) bool {
	// Check if this is a script (universal check for all OS)
	if strings.HasPrefix(cmdName, ".") || strings.ContainsRune(cmdName, filepath.Separator) || filepath.IsAbs(cmdName) {
		var scriptPath string
		if filepath.IsAbs(cmdName) {
			scriptPath = cmdName
		} else {
			scriptPath = filepath.Join(m.currentSession.WorkingDirectory(), cmdName)
		}

		// Check file extension
		ext := strings.ToLower(filepath.Ext(scriptPath))
		switch ext {
		case extSh, extBash, ".bat", ".cmd", ".ps1":
			// Scripts may require interactive mode (read, input, etc.)
			return true
		}
	}

	// List of known interactive commands
	interactiveCommands := map[string]bool{
		"vi":     true,
		"vim":    true,
		"nvim":   true,
		"nano":   true,
		"emacs":  true,
		"less":   true,
		"more":   true,
		"top":    true,
		"htop":   true,
		"ssh":    true,
		"telnet": true,
		"ftp":    true,
		"sftp":   true,
		"python": true, // Python REPL
		"node":   true, // Node.js REPL
		"irb":    true, // Ruby REPL
		"psql":   true, // PostgreSQL
		"mysql":  true, // MySQL
		"mongo":  true, // MongoDB
	}

	// Check base command name (without path)
	baseName := filepath.Base(cmdName)
	baseName = strings.TrimSuffix(baseName, filepath.Ext(baseName))

	return interactiveCommands[baseName]
}

// executeShellScriptNative executes .sh/.bash script natively via mvdan.cc/sh.
func (m *Model) executeShellScriptNative(scriptPath string, args []string) api.Cmd {
	return func() api.Msg {
		// Open script file
		file, err := os.Open(scriptPath) //nolint:gosec // G304: This is a shell - dynamic script execution is expected
		if err != nil {
			return commandExecutedMsg{
				err:      fmt.Errorf("failed to open script: %w", err),
				exitCode: 1,
			}
		}
		defer func() { _ = file.Close() }()

		// Parse script
		scriptParser := syntax.NewParser()
		prog, err := scriptParser.Parse(file, scriptPath)
		if err != nil {
			return commandExecutedMsg{
				err:      fmt.Errorf("failed to parse script: %w", err),
				exitCode: 1,
			}
		}

		// Create buffers for output capture
		var stdout, stderr bytes.Buffer

		// Create interpreter with our settings
		runner, err := interp.New(
			interp.StdIO(nil, &stdout, &stderr),                     // Capture stdout/stderr
			interp.Dir(m.currentSession.WorkingDirectory()),         // Working directory
			interp.Env(expandEnv(m.currentSession)),                 // Environment variables
			interp.Params(append([]string{scriptPath}, args...)...), // Script arguments ($0, $1, ...)
		)
		if err != nil {
			return commandExecutedMsg{
				err:      fmt.Errorf("failed to create interpreter: %w", err),
				exitCode: 1,
			}
		}

		// Execute script
		err = runner.Run(m.ctx, prog)

		// Determine exit code (v3.12.0+ API)
		exitCode := 0
		if err != nil {
			// New API: ExitStatus returns uint8
			var exitStatus interp.ExitStatus
			if errors.As(err, &exitStatus) {
				exitCode = int(exitStatus)
			} else if err != nil {
				// Other error (not exit status)
				exitCode = 1
			}
		}

		// Combine stdout and stderr
		output := stdout.String()
		if stderr.Len() > 0 {
			if output != "" {
				output += "\n"
			}
			output += stderr.String()
		}

		return commandExecutedMsg{
			output:   output,
			err:      err,
			exitCode: exitCode,
		}
	}
}

// sessionEnviron adapter for session.Environment → expand.Environ.
type sessionEnviron struct {
	sess *session.Session
}

func (e *sessionEnviron) Get(name string) expand.Variable {
	value, exists := e.sess.Environment().Get(name)
	if !exists {
		// Variable not found - return empty
		return expand.Variable{}
	}
	return expand.Variable{
		Exported: true,
		Kind:     expand.String,
		Str:      value,
	}
}

func (e *sessionEnviron) Each(fn func(name string, vr expand.Variable) bool) {
	// Iterate over all environment variables
	envSlice := e.sess.Environment().ToSlice()
	for _, envVar := range envSlice {
		// Parse "KEY=VALUE"
		parts := strings.SplitN(envVar, "=", 2)
		if len(parts) == 2 {
			name := parts[0]
			value := parts[1]
			vr := expand.Variable{
				Exported: true,
				Kind:     expand.String,
				Str:      value,
			}
			if !fn(name, vr) {
				break
			}
		}
	}
}

// expandEnv creates adapter for session.Environment.
func expandEnv(sess *session.Session) expand.Environ {
	return &sessionEnviron{sess: sess}
}

// execInteractiveCommand executes interactive command via api.ExecProcess.
// Used for scripts requiring full TTY (clear, read, menu).
func (m *Model) execInteractiveCommand(commandLine string) api.Cmd {
	// Parse command
	cmd, pipe, err := parser.ParseCommandLine(commandLine)
	if err != nil {
		return func() api.Msg {
			return commandExecutedMsg{
				err:      err,
				exitCode: 1,
			}
		}
	}

	// Pipes not yet supported in interactive mode
	if pipe != nil {
		return func() api.Msg {
			return commandExecutedMsg{
				output:   "[Interactive pipes not yet supported]",
				exitCode: 1,
			}
		}
	}

	if cmd == nil {
		return func() api.Msg {
			return commandExecutedMsg{
				output:   "",
				exitCode: 0,
			}
		}
	}

	// Prepare command (scripts via interpreters)
	cmdName, cmdArgs := m.prepareCommand(cmd.Name(), cmd.Args())

	// Create exec.Cmd with proper settings
	osCmd := exec.Command(cmdName, cmdArgs...) //nolint:gosec // G204: This is a shell - command execution with user input is expected
	osCmd.Dir = m.currentSession.WorkingDirectory()
	osCmd.Env = m.currentSession.Environment().ToSlice()

	// TODO: Phoenix doesn't have ExecProcess yet - need to add it to Phoenix tea/api
	// For now, return error
	return func() api.Msg {
		return commandExecutedMsg{
			output:   "[Interactive commands not yet supported - Phoenix migration in progress]",
			err:      fmt.Errorf("ExecProcess not available in Phoenix yet"),
			exitCode: 1,
		}
	}
}
