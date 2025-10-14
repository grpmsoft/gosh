package repl

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/grpmsoft/gosh/internal/domain/config"

	tea "github.com/charmbracelet/bubbletea"
)

// All methods in this file use Bubbletea's MVU (Model-View-Update) pattern,.
// which requires value receivers. The "hugeParam" warnings are false positives.
//
//nolint:gocritic // All Model methods: Bubbletea MVU requires value receivers

// Update handles messages (Elm Architecture).
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		taCmd tea.Cmd
		vpCmd tea.Cmd
		spCmd tea.Cmd
	)

	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKeyPress(msg)

	case tea.MouseMsg:
		// Handle mouse wheel for viewport
		if msg.Action == tea.MouseActionPress && (msg.Button == tea.MouseButtonWheelUp || msg.Button == tea.MouseButtonWheelDown) {
			m.autoScroll = false // Disable auto-scroll on manual scrolling
			m.viewport, vpCmd = m.viewport.Update(msg)
			return m, vpCmd
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.textarea.SetWidth(msg.Width)

		// Update viewport size
		// Classic mode: prompt inside viewport, use full height (we preserve scroll via YOffset)
		// Other modes: prompt outside viewport, reserve space
		var viewportHeight int
		switch m.config.UI.Mode {
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
		m.viewport.Width = msg.Width
		m.viewport.Height = viewportHeight
		m.updateViewportContent()

		m.ready = true
		return m, nil

	case commandExecutedMsg:
		m.executing = false
		m.lastExitCode = msg.exitCode

		// Handle output differently based on UI mode
		if msg.output != "" {
			if m.config.UI.Mode == config.UIModeClassic {
				// Classic mode: Print directly to stdout (native terminal scrolling)
				// Output sequence (bash-style):
				// 1. User types command: "user@host $ ls█"
				// 2. Presses Enter → command line is frozen in history
				// 3. Output prints line by line
				// 4. Separator printed (configurable via OutputSeparator)
				// 5. Prompt reappears below output

				// Print command output line by line
				lines := strings.Split(strings.TrimRight(msg.output, "\n"), "\n")
				for _, line := range lines {
					fmt.Println(line) // Each line includes \n
				}

				// Print separator after output (configurable)
				if m.config.UI.OutputSeparator != "" {
					fmt.Print(m.config.UI.OutputSeparator)
				}
			} else {
				// Other modes (Warp/Compact/Chat): Use viewport for scrolling
				// Add blank line for visual separation
				m.addOutputRaw("")

				// Split output into lines and store in viewport buffer
				lines := strings.Split(strings.TrimRight(msg.output, "\n"), "\n")
				for _, line := range lines {
					m.addOutputRaw(line)
				}
			}
		}

		// Show additional error if present (e.g. "exit status 1")
		// Usually msg.err contains only exit status, real stderr is already in msg.output
		if msg.err != nil && msg.output == "" {
			if m.config.UI.Mode == config.UIModeClassic {
				// Classic mode: print error directly to stdout
				fmt.Println("\033[31mError: " + msg.err.Error() + "\033[0m")
			} else {
				// Other modes: add to viewport buffer
				m.addOutputRaw("\033[31mError: " + msg.err.Error() + "\033[0m")
			}
		}

		// Update Git status after each command
		m.updateGitInfo()

		// Update viewport content and scroll (only for non-Classic modes)
		if m.config.UI.Mode != config.UIModeClassic {
			m.updateViewportContent()
			if m.autoScroll {
				m.viewport.GotoBottom()
			}
		}

		return m, nil
	}

	// Update textarea
	m.textarea, taCmd = m.textarea.Update(msg)

	// Sync our input state with textarea
	m.inputText = m.textarea.Value()
	// Cursor always at end after normal input (textarea doesn't give position API)
	m.cursorPos = len([]rune(m.inputText))

	// Update viewport (for PageUp/PageDown scrolling)
	m.viewport, vpCmd = m.viewport.Update(msg)

	// Update spinner if executing
	if m.executing {
		m.executingSpinner, spCmd = m.executingSpinner.Update(msg)
		return m, tea.Batch(taCmd, vpCmd, spCmd)
	}

	return m, tea.Batch(taCmd, vpCmd)
}

// handleKeyPress handles key presses.
func (m Model) handleKeyPress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// ESC - close help overlay (if open).
	if msg.String() == "esc" && m.showingHelp {
		m.showingHelp = false
		return m, nil
	}

	// F1 or ? - open help overlay.
	if msg.String() == "f1" || msg.String() == "?" {
		m.showingHelp = true
		return m, nil
	}

	// If showing help - block other keys.
	if m.showingHelp {
		return m, nil
	}

	switch msg.String() {
	case "ctrl+c":
		m.quitting = true
		return m, tea.Quit

	case "ctrl+d":
		if m.textarea.Value() == "" {
			m.quitting = true
			return m, tea.Quit
		}

	case "enter":
		// Regular Enter - execute command.
		m.autoScroll = true // Enable auto-scroll when executing command.
		return m.executeCommand()

	case "alt+enter":
		// Alt+Enter - add new line (multiline).
		currentHeight := m.textarea.Height()
		if currentHeight < 10 {
			m.textarea.SetHeight(currentHeight + 1)
		}
		// Let textarea handle new line insertion.
		var cmd tea.Cmd
		m.textarea, cmd = m.textarea.Update(msg)
		return m, cmd

	case "up", "down":
		// Command history.
		return m.navigateHistory(msg.String())

	case "tab":
		// Tab-completion.
		return m.handleTabCompletion()

	case "ctrl+l":
		// Clear screen.
		m.output = make([]string, 0)
		m.updateViewportContent()
		m.autoScroll = true
		return m, tea.ClearScreen

	case "pgup", "pgdown":
		// Viewport scrolling.
		m.autoScroll = false
		var cmd tea.Cmd
		m.viewport, cmd = m.viewport.Update(msg)
		return m, cmd

	// Hotkeys for switching UI modes (Alt+1-4).
	case "alt+1", "alt+2", "alt+3", "alt+4":
		if m.config.UI.AllowModeSwitching {
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

	// Return auto-scroll on any input.
	if msg.Type == tea.KeyRunes {
		m.autoScroll = true
	}

	var cmd tea.Cmd
	m.textarea, cmd = m.textarea.Update(msg)
	return m, cmd
}

// switchUIMode switches UI mode.
func (m Model) switchUIMode(key string) (tea.Model, tea.Cmd) {
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
	if m.config.UI.Mode == newMode {
		return m, nil
	}

	// Switch mode.
	oldMode := m.config.UI.Mode
	m.config.UI.Mode = newMode

	// Recalculate viewport height for new mode.
	// Classic: prompt inside, full height; Others: prompt outside, reserve space.
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
	m.viewport.Height = viewportHeight

	// Log switch.
	m.logger.Info("UI mode switched", "from", oldMode, "to", newMode)

	// Show mode switch notification.
	notification := fmt.Sprintf("\033[90m[UI Mode: %s]\033[0m", newMode)
	if newMode == config.UIModeClassic {
		// Classic mode: print notification directly to stdout.
		fmt.Println(notification)
	} else {
		// Other modes: add to viewport buffer.
		m.addOutputRaw(notification)
		m.updateViewportContent()

		// Scroll down if auto-scroll enabled.
		if m.autoScroll {
			m.viewport.GotoBottom()
		}
	}

	return m, nil
}

// handleModeCommand handles :mode command for switching UI modes.
func (m Model) handleModeCommand(commandLine string) (tea.Model, tea.Cmd) {
	// Check if mode switching is enabled.
	if !m.config.UI.AllowModeSwitching {
		m.addOutputRaw("\033[31mError: UI mode switching is disabled in config\033[0m")
		m.updateViewportContent()
		if m.autoScroll {
			m.viewport.GotoBottom()
		}
		return m, nil
	}

	// Parse command arguments.
	parts := strings.Fields(commandLine)

	// If only ":mode" without arguments - show current mode.
	if len(parts) == 1 {
		m.addOutputRaw(fmt.Sprintf("\033[90mCurrent UI mode: \033[1;32m%s\033[0m", m.config.UI.Mode))
		m.addOutputRaw("\033[90mAvailable modes: classic, warp, compact, chat\033[0m")
		m.addOutputRaw("\033[90mUsage: :mode <name>\033[0m")
		m.updateViewportContent()
		if m.autoScroll {
			m.viewport.GotoBottom()
		}
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
		if m.autoScroll {
			m.viewport.GotoBottom()
		}
		return m, nil
	}

	// If already in this mode - just notify.
	if m.config.UI.Mode == newMode {
		m.addOutputRaw(fmt.Sprintf("\033[90mAlready in %s mode\033[0m", newMode))
		m.updateViewportContent()
		if m.autoScroll {
			m.viewport.GotoBottom()
		}
		return m, nil
	}

	// Switch mode.
	oldMode := m.config.UI.Mode
	m.config.UI.Mode = newMode

	// Recalculate viewport height for new mode.
	// Classic: prompt inside, full height; Others: prompt outside, reserve space.
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
	m.viewport.Height = viewportHeight

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

		// Scroll down if auto-scroll enabled.
		if m.autoScroll {
			m.viewport.GotoBottom()
		}
	}

	return m, nil
}

// handleTabCompletion handles Tab-completion.
func (m Model) handleTabCompletion() (tea.Model, tea.Cmd) {
	input := m.textarea.Value()

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
		m.textarea.SetValue(m.completions[0])
		// Sync input state.
		m.inputText = m.completions[0]
		m.cursorPos = len([]rune(m.inputText))
		return m, nil
	}

	// Repeated Tab presses - cycle through variants.
	if len(m.completions) > 0 {
		m.completionIndex = (m.completionIndex + 1) % len(m.completions)
		m.textarea.SetValue(m.completions[m.completionIndex])
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
