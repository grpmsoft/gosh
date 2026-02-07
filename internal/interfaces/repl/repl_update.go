package repl

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/grpmsoft/gosh/internal/domain/config"
	clipapi "github.com/phoenix-tui/phoenix/clipboard"
	"github.com/phoenix-tui/phoenix/tea"
)

// All methods in this file use Bubbletea's MVU (Model-View-Update) pattern,.
// which requires value receivers. The "hugeParam" warnings are false positives.
//
//nolint:gocritic // All Model methods: Bubbletea MVU requires value receivers

// Update handles messages (Elm Architecture).
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	var (
		taCmd tea.Cmd
	)

	switch msg := msg.(type) {
	case setProgramMsg:
		// Inject program reference for ExecProcess (interactive commands)
		// This message is sent once from main.go after Program creation
		// MVU pattern copies Model, so program must be set via message not directly
		m.program = msg.program
		m.logger.Info("Program reference injected", "is_nil", m.program == nil)
		return m, nil

	case tea.TickMsg:
		// Tick is no longer needed - we use terminal's native blinking cursor!
		// Terminal cursor blinks automatically (set via \033[5 q in main.go)
		// Phoenix-rendered cursor (reverse video) is disabled via ShowCursor(false)
		//
		// Previously: Tick toggled m.cursorVisible every 500ms for Phoenix cursor
		// Now: Terminal handles blinking, no perma-redraw needed!
		return m, nil

	case tea.KeyMsg:
		return m.handleKeyPress(msg)

	case tea.MouseMsg:
		// Handle mouse wheel for viewport (Phoenix Viewport uses api types now)
		if msg.Action == tea.MouseActionPress && (msg.Button == tea.MouseButtonWheelUp || msg.Button == tea.MouseButtonWheelDown) {
			m.autoScroll = false // Disable auto-scroll on manual scrolling
			// Phoenix Viewport.Update() returns (*Viewport, tea.Cmd) directly
			var vpCmd tea.Cmd
			m.viewport, vpCmd = m.viewport.Update(msg)
			return m, vpCmd
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.shellInput.SetWidth(msg.Width)
		m.shellTextArea.SetSize(msg.Width, 5) // Fixed height for textarea (5 lines)

		// Update viewport size
		// Classic mode: prompt inside viewport, use full height (we preserve scroll via YOffset)
		// Other modes: prompt outside viewport, reserve space
		var viewportHeight int
		switch m.Config.UI.Mode {
		case config.UIModeClassic:
			// Classic mode - prompt inside viewport, full screen height
			viewportHeight = msg.Height
		case config.UIModeCompact:
			// Compact mode - reserve 1 line for prompt
			viewportHeight = msg.Height - 1
		default:
			// Warp/Chat - reserve 3 lines (prompt + separator)
			viewportHeight = msg.Height - 3
		}

		if viewportHeight < 1 {
			viewportHeight = 1
		}
		// Phoenix Viewport uses fluent SetSize() API
		// Width set to large value — viewport's truncateLine() counts ANSI escape
		// bytes as visible width, garbling colors. Terminal wraps natively.
		m.viewport = m.viewport.SetSize(10000, viewportHeight)
		m.updateViewportContent()

		m.ready = true
		return m, nil

	case commandExecutedMsg:
		m.executing = false
		m.lastExitCode = msg.exitCode

		// Restore cursor visibility after ExecProcess
		// Phoenix Resume() hides cursor (for TUI alt screen mode), but Classic mode
		// uses native terminal cursor. Show it once here, not on every render.
		if m.Config.UI.Mode == config.UIModeClassic {
			_ = m.terminal.ShowCursor()
		}

		// Handle output differently based on UI mode
		if msg.output != "" {
			if m.Config.UI.Mode == config.UIModeClassic {
				// Classic mode: Print directly to stdout (NO alt screen, NO viewport!)
				// Output stays in terminal history like bash
				lines := strings.Split(strings.TrimRight(msg.output, "\n"), "\n")
				for _, line := range lines {
					fmt.Println(line)
				}

				// Print separator after output (configurable)
				if m.Config.UI.OutputSeparator != "" {
					fmt.Print(m.Config.UI.OutputSeparator)
				}
			} else {
				// Other modes: Use viewport for scrolling
				m.addOutputRaw("")
				lines := strings.Split(strings.TrimRight(msg.output, "\n"), "\n")
				for _, line := range lines {
					m.addOutputRaw(line)
				}
			}
		}

		// Show additional error if present
		if msg.err != nil && msg.output == "" {
			if m.Config.UI.Mode == config.UIModeClassic {
				fmt.Println("\033[31mError: " + msg.err.Error() + "\033[0m")
			} else {
				m.addOutputRaw("\033[31mError: " + msg.err.Error() + "\033[0m")
			}
		}

		// Update Git status after each command
		m.updateGitInfo()

		// Update viewport content (only for non-Classic modes)
		if m.Config.UI.Mode != config.UIModeClassic {
			m.updateViewportContent()
		}

		return m, nil
	}

	// Update appropriate input component based on mode
	if m.multilineMode {
		m.shellTextArea, taCmd = m.shellTextArea.Update(msg)
		m.inputText = m.shellTextArea.Value()
	} else {
		m.shellInput, taCmd = m.shellInput.Update(msg)
		m.inputText = m.shellInput.Value()
	}

	// Cursor always at end after normal input (textarea doesn't give position API)
	m.cursorPos = len([]rune(m.inputText))

	// Update viewport (for PageUp/PageDown scrolling) - Phoenix Viewport
	// Note: Viewport handles its own key bindings internally
	// We don't need to update it here since we're handling keys in handleKeyPress

	// No spinner update needed - Phoenix migration removed spinner
	// Executing state is shown via text in render functions

	return m, taCmd
}

// handleKeyPress handles key presses.
func (m Model) handleKeyPress(msg tea.KeyMsg) (Model, tea.Cmd) {
	// IMPORTANT: Check msg.Type for Enter FIRST (before String() checks)
	// Phoenix may send KeyEnter as Type when Enter is pressed after UTF-8 input
	if msg.Type == tea.KeyEnter {
		// Get current input from ACTIVE component (critical for correct multiline switch!)
		var cmd string
		if m.multilineMode {
			cmd = m.shellTextArea.Value()
		} else {
			cmd = m.shellInput.Value()
		}

		// Check if command is incomplete (unclosed quotes, backslash, pipe, etc.)
		if m.isIncomplete(cmd) && !m.multilineMode {
			// Switch to multiline mode
			m.multilineMode = true
			// CRITICAL: Print newline to start multiline on fresh line
			// This ensures we have clean space for multiline rendering
			fmt.Println() // Move to next line
			m.shellTextArea.SetValue(cmd + "\n") // Add newline
			// Sync state
			m.inputText = m.shellTextArea.Value()
			m.cursorPos = len([]rune(m.inputText))
			return m, nil
		}

		// Command is complete - execute it
		m.autoScroll = true
		return m.executeCommand()
	}

	// ESC - close help overlay (if open).
	if msg.String() == "esc" && m.showingHelp {
		m.showingHelp = false
		return m, nil
	}

	// F1 or ? - open help overlay.
	if msg.String() == "F1" || msg.String() == "?" {
		m.showingHelp = true
		return m, nil
	}

	// If showing help - block other keys.
	if m.showingHelp {
		return m, nil
	}

	switch msg.String() {
	case "ctrl+c":
		if m.Config.UI.Mode != config.UIModeClassic {
			fmt.Print("\033[?1049l") // Exit alt screen before quit
		}
		m.quitting = true
		return m, tea.Quit()

	case "ctrl+d":
		// Check if input is empty (respect multilineMode)
		isEmpty := false
		if m.multilineMode {
			isEmpty = m.shellTextArea.Value() == ""
		} else {
			isEmpty = m.shellInput.Value() == ""
		}
		if isEmpty {
			if m.Config.UI.Mode != config.UIModeClassic {
				fmt.Print("\033[?1049l") // Exit alt screen before quit
			}
			m.quitting = true
			return m, tea.Quit()
		}

	case "ctrl+v":
		// Paste from clipboard (respect multilineMode)
		text, err := clipapi.Read()
		if err == nil && text != "" {
			if m.multilineMode {
				// Insert clipboard text in textarea
				currentValue := m.shellTextArea.Value()
				m.shellTextArea.SetValue(currentValue + text)
				m.inputText = m.shellTextArea.Value()
			} else {
				// Insert clipboard text in single-line input
				currentValue := m.shellInput.Value()
				m.shellInput.SetValue(currentValue + text)
				m.inputText = m.shellInput.Value()
			}
			m.cursorPos = len([]rune(m.inputText))
		}
		return m, nil

	case "enter":
		// Regular Enter - this case is redundant (handled above via KeyEnter)
		// But keep for compatibility with string-based key handling
		var cmd string
		if m.multilineMode {
			cmd = m.shellTextArea.Value()
		} else {
			cmd = m.shellInput.Value()
		}

		// Check if command is incomplete (unclosed quotes, backslash, pipe, etc.)
		if m.isIncomplete(cmd) && !m.multilineMode {
			// Switch to multiline mode
			m.multilineMode = true
			// CRITICAL: Print newline to start multiline on fresh line
			// This ensures we have clean space for multiline rendering
			fmt.Println() // Move to next line
			m.shellTextArea.SetValue(cmd + "\n") // Add newline
			// Sync state
			m.inputText = m.shellTextArea.Value()
			m.cursorPos = len([]rune(m.inputText))
			return m, nil
		}

		// Command is complete - execute it
		m.autoScroll = true
		return m.executeCommand()

	case "alt+enter":
		// Alt+Enter - force multiline mode or insert newline
		if !m.multilineMode {
			// Switch to multiline mode
			m.multilineMode = true
			// CRITICAL: Print newline to start multiline on fresh line
			fmt.Println() // Move to next line
			currentValue := m.shellInput.Value() // Use shellInput.Value() directly!
			m.shellTextArea.SetValue(currentValue + "\n")
			m.inputText = m.shellTextArea.Value()
			m.cursorPos = len([]rune(m.inputText))
			return m, nil
		}
		// Already in multiline - insert newline
		var cmd tea.Cmd
		m.shellTextArea, cmd = m.shellTextArea.Update(tea.KeyMsg{Type: tea.KeyEnter})
		m.inputText = m.shellTextArea.Value()
		m.cursorPos = len([]rune(m.inputText))
		return m, cmd

	case "up", "down", "↑", "↓":
		// Command history (Phoenix returns "↑"/"↓" for arrow keys).
		m.ghostSuggestion = "" // Clear predictive suggestion during navigation
		dir := directionUp
		if msg.String() == "down" || msg.String() == "↓" {
			dir = directionDown
		}
		return m.navigateHistory(dir)

	case "right", "→":
		// Accept predictive suggestion (PSReadLine behavior)
		// Only accept when ghost suggestion exists and not in multiline mode
		if m.ghostSuggestion != "" && !m.multilineMode {
			m.shellInput.SetValue(m.ghostSuggestion)
			m.shellInput.RefreshHighlight()
			m.inputText = m.ghostSuggestion
			m.cursorPos = len([]rune(m.inputText))
			m.ghostSuggestion = ""
			return m, nil
		}
		// No suggestion → fall through to normal right arrow (cursor movement)

	case "tab":
		// Tab-completion.
		m.ghostSuggestion = "" // Clear predictive suggestion during completion
		return m.handleTabCompletion()

	case "ctrl+l":
		// Clear screen.
		m.output = make([]string, 0)
		m.updateViewportContent()
		m.autoScroll = true
		return m, nil // Phoenix doesn't have ClearScreen, we handle it in View

	case "pgup", "pgdown":
		// Viewport scrolling - Phoenix Viewport handles internally (uses api types)
		m.autoScroll = false
		var vpCmd tea.Cmd
		m.viewport, vpCmd = m.viewport.Update(msg)
		return m, vpCmd

	// Hotkeys for switching UI modes (Alt+1-4).
	case "alt+1", "alt+2", "alt+3", "alt+4":
		if m.Config.UI.AllowModeSwitching {
			return m.switchUIMode(msg.String())
		}
	}

	// Reset completion on any other input.
	if m.completionActive {
		m.completionActive = false
		m.completions = []string{}
		m.completionIndex = -1
		m.beforeCompletion = ""
	}

	// Return auto-scroll and show cursor on any input.
	if msg.Type == tea.KeyRune || msg.Type == tea.KeySpace {
		m.autoScroll = true
		m.cursorVisible = true // Show cursor immediately when typing
	}

	// CRITICAL: Delegate to appropriate input component based on mode
	var cmd tea.Cmd
	if m.multilineMode {
		m.shellTextArea, cmd = m.shellTextArea.Update(msg)
		m.inputText = m.shellTextArea.Value()
	} else {
		m.shellInput, cmd = m.shellInput.Update(msg)
		m.inputText = m.shellInput.Value()
	}

	// CRITICAL: Sync cursor position after update
	m.cursorPos = len([]rune(m.inputText))

	// Update predictive suggestion from history (PSReadLine-style IntelliSense)
	m.ghostSuggestion = ""
	if m.inputText != "" && !m.multilineMode && !m.completionActive {
		m.ghostSuggestion = m.currentSession.History().SearchPrefix(m.inputText)
	}

	return m, cmd
}

// switchUIMode switches UI mode with manual alt screen management.
// No Program restart needed — alt screen is toggled via ANSI escape sequences.
func (m Model) switchUIMode(key string) (Model, tea.Cmd) {
	var newMode config.UIMode

	switch key {
	case "alt+1":
		newMode = config.UIModeClassic
	case "alt+2":
		newMode = config.UIModeWarp
	case "alt+3":
		newMode = config.UIModeCompact
	case "alt+4":
		newMode = config.UIModeChat
	default:
		return m, nil
	}

	// If already in this mode - do nothing.
	if m.Config.UI.Mode == newMode {
		return m, nil
	}

	oldMode := m.Config.UI.Mode
	wasClassic := oldMode == config.UIModeClassic
	isClassic := newMode == config.UIModeClassic

	// Toggle alt screen when crossing classic ↔ non-classic boundary.
	if wasClassic && !isClassic {
		// Classic → non-classic: enter alt screen
		fmt.Print("\033[?1049h")
		m.altScreenActive = true
	} else if !wasClassic && isClassic {
		// Non-classic → classic: exit alt screen (restores normal terminal buffer)
		fmt.Print("\033[?1049l")
		m.altScreenActive = false
	}

	m.Config.UI.Mode = newMode

	// Recalculate viewport height for new mode.
	var viewportHeight int
	switch newMode {
	case config.UIModeClassic:
		viewportHeight = m.height
	case config.UIModeCompact:
		viewportHeight = m.height - 1
	default:
		viewportHeight = m.height - 3
	}

	if viewportHeight < 1 {
		viewportHeight = 1
	}
	m.viewport = m.viewport.SetSize(10000, viewportHeight)

	// Log switch.
	m.logger.Info("UI mode switched", "from", oldMode, "to", newMode)

	// Show mode switch notification.
	notification := fmt.Sprintf("\033[90m[UI Mode: %s]\033[0m", newMode)
	if newMode == config.UIModeClassic {
		fmt.Println(notification)
	} else {
		m.addOutputRaw(notification)
		m.updateViewportContent()
	}

	return m, nil
}

// handleModeCommand handles :mode command for switching UI modes.
func (m Model) handleModeCommand(commandLine string) (Model, tea.Cmd) {
	// Check if mode switching is enabled.
	if !m.Config.UI.AllowModeSwitching {
		m.addOutputRaw("\033[31mError: UI mode switching is disabled in config\033[0m")
		m.updateViewportContent()
		// FollowMode handles auto-scroll in render functions
		return m, nil
	}

	// Parse command arguments.
	parts := strings.Fields(commandLine)

	// If only ":mode" without arguments - show current mode.
	if len(parts) == 1 {
		m.addOutputRaw(fmt.Sprintf("\033[90mCurrent UI mode: \033[1;32m%s\033[0m", m.Config.UI.Mode))
		m.addOutputRaw("\033[90mAvailable modes: classic, warp, compact, chat\033[0m")
		m.addOutputRaw("\033[90mUsage: :mode <name>\033[0m")
		m.updateViewportContent()
		// FollowMode handles auto-scroll in render functions
		return m, nil
	}

	// Get mode name.
	modeName := strings.ToLower(parts[1])

	// Map names to modes.
	var newMode config.UIMode
	switch modeName {
	case "classic":
		newMode = config.UIModeClassic
	case "warp":
		newMode = config.UIModeWarp
	case "compact":
		newMode = config.UIModeCompact
	case "chat":
		newMode = config.UIModeChat
	default:
		m.addOutputRaw(fmt.Sprintf("\033[31mError: unknown mode '%s'\033[0m", modeName))
		m.addOutputRaw("\033[90mAvailable modes: classic, warp, compact, chat\033[0m")
		m.updateViewportContent()
		// FollowMode handles auto-scroll in render functions
		return m, nil
	}

	// If already in this mode - just notify.
	if m.Config.UI.Mode == newMode {
		m.addOutputRaw(fmt.Sprintf("\033[90mAlready in %s mode\033[0m", newMode))
		m.updateViewportContent()
		// FollowMode handles auto-scroll in render functions
		return m, nil
	}

	// Toggle alt screen when crossing classic ↔ non-classic boundary.
	wasClassic := m.Config.UI.Mode == config.UIModeClassic
	isClassic := newMode == config.UIModeClassic

	if wasClassic && !isClassic {
		fmt.Print("\033[?1049h") // Enter alt screen
		m.altScreenActive = true
	} else if !wasClassic && isClassic {
		fmt.Print("\033[?1049l") // Exit alt screen
		m.altScreenActive = false
	}

	oldMode := m.Config.UI.Mode
	m.Config.UI.Mode = newMode

	// Recalculate viewport height for new mode.
	var viewportHeight int
	switch newMode {
	case config.UIModeClassic:
		viewportHeight = m.height
	case config.UIModeCompact:
		viewportHeight = m.height - 1
	default:
		viewportHeight = m.height - 3
	}

	if viewportHeight < 1 {
		viewportHeight = 1
	}
	m.viewport = m.viewport.SetSize(10000, viewportHeight)

	// Log switch.
	m.logger.Info("UI mode switched via :mode command", "from", oldMode, "to", newMode)

	// Show mode switch notification.
	notification := fmt.Sprintf("\033[90m[UI Mode: %s]\033[0m", newMode)
	if newMode == config.UIModeClassic {
		// Classic mode: print notification directly to stdout.
		fmt.Println(notification)
	} else {
		// Other modes: add to viewport buffer.
		m.addOutputRaw(notification)
		m.updateViewportContent()
	}

	return m, nil
}

// handleTabCompletion handles Tab-completion.
func (m Model) handleTabCompletion() (Model, tea.Cmd) {
	// Tab-completion only works in single-line mode
	// In multiline mode, tab should insert tab character (handled by TextArea)
	if m.multilineMode {
		// Delegate to textarea (will insert tab or spaces)
		var cmd tea.Cmd
		m.shellTextArea, cmd = m.shellTextArea.Update(tea.KeyMsg{Type: tea.KeyTab})
		m.inputText = m.shellTextArea.Value()
		m.cursorPos = len([]rune(m.inputText))
		return m, cmd
	}

	// Single-line mode - do tab-completion
	input := m.shellInput.Value()

	// First Tab press - generate completions.
	if !m.completionActive {
		m.beforeCompletion = input
		m.completions = m.generateCompletions(input)
		m.completionIndex = -1

		if len(m.completions) == 0 {
			return m, nil
		}

		m.completionActive = true
		m.completionIndex = 0
		m.shellInput.SetValue(m.completions[0])
		// Sync input state.
		m.inputText = m.completions[0]
		m.cursorPos = len([]rune(m.inputText))
		return m, nil
	}

	// Repeated Tab presses - cycle through variants.
	if len(m.completions) > 0 {
		m.completionIndex = (m.completionIndex + 1) % len(m.completions)
		m.shellInput.SetValue(m.completions[m.completionIndex])
		// Sync input state.
		m.inputText = m.completions[m.completionIndex]
		m.cursorPos = len([]rune(m.inputText))
	}

	return m, nil
}

// generateCompletions generates autocompletion variants.
func (m *Model) generateCompletions(input string) []string {
	completions := []string{}

	if input == "" {
		return completions
	}

	// Parse input to determine what to complete.
	parts := strings.Fields(input)
	if len(parts) == 0 {
		return completions
	}

	// First word - command.
	if len(parts) == 1 {
		prefix := parts[0]

		// Built-in commands.
		builtins := []string{"cd", "pwd", "echo", "exit", "help", "clear", "export", "unset", "env", "type", "alias", "unalias"}
		for _, cmd := range builtins {
			if strings.HasPrefix(cmd, prefix) {
				completions = append(completions, cmd)
			}
		}

		// Aliases.
		aliases := m.currentSession.GetAllAliases()
		for aliasName := range aliases {
			if strings.HasPrefix(aliasName, prefix) {
				completions = append(completions, aliasName)
			}
		}

		// PATH commands can be added later.
		return completions
	}

	// Remaining words - files/directories.
	lastPart := parts[len(parts)-1]
	dirPath := filepath.Dir(lastPart)
	baseName := filepath.Base(lastPart)

	if dirPath == "." {
		dirPath = m.currentSession.WorkingDirectory()
	} else if !filepath.IsAbs(dirPath) {
		dirPath = filepath.Join(m.currentSession.WorkingDirectory(), dirPath)
	}

	// Read directory.
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return completions
	}

	// Filter by prefix.
	prefix := input[:len(input)-len(baseName)]
	for _, entry := range entries {
		name := entry.Name()
		if strings.HasPrefix(name, baseName) {
			completion := prefix + name
			if entry.IsDir() {
				completion += string(filepath.Separator)
			}
			completions = append(completions, completion)
		}
	}

	return completions
}
