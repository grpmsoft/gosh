package repl

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/grpmsoft/gosh/internal/domain/config"

	"github.com/charmbracelet/lipgloss"
)

// All methods in this file use Bubbletea's MVU (Model-View-Update) pattern,.
// which requires value receivers. The "hugeParam" warnings are false positives.
//
//nolint:gocritic // All Model methods: Bubbletea MVU requires value receivers

// View renders UI (Elm Architecture).
func (m Model) View() string {
	if !m.ready {
		return ""
	}

	if m.quitting {
		return ""
	}

	// If showing help overlay - render it on top of main UI
	if m.showingHelp {
		return m.renderWithHelpOverlay()
	}

	// Choose rendering based on UI mode
	switch m.config.UI.Mode {
	case config.UIModeClassic:
		return m.renderClassicMode()
	case config.UIModeWarp:
		return m.renderWarpMode()
	case config.UIModeCompact:
		return m.renderCompactMode()
	case config.UIModeChat:
		return m.renderChatMode()
	default:
		return m.renderClassicMode() // Fallback.
	}
}

// renderClassicMode renders Classic mode (bash/pwsh).
// Uses native terminal scrolling like real bash (no viewport wrapper).
//
// IMPORTANT: Classic mode does NOT use viewport for rendering history.
// Instead, command output is printed directly to stdout via fmt.Println() in Update().
// This allows:
//   - Native terminal scrollback to work (PgUp/PgDn, mouse wheel).
//   - History remains in terminal after shell exit (like bash).
//   - No artificial viewport limitations.
//
// We only render the current prompt + input line here (last line of terminal).
func (m Model) renderClassicMode() string {
	var b strings.Builder

	// Render only the current input line (prompt + input).
	// Classic mode: NO spinner (like real bash/pwsh).
	// When executing, simply don't render anything - command output will appear naturally.
	if !m.executing {
		// Normal prompt with input.
		b.WriteString(m.renderPromptForHistoryANSI())
		b.WriteString(m.renderInputWithCursor())
		b.WriteString(m.renderHints())
	}

	// Return only prompt+input (no viewport, no history rendering).
	// History is already in terminal via fmt.Println() from Update().
	return b.String()
}

// renderWarpMode renders modern Warp-like mode.
// Prompt on top, output below with separator.
func (m Model) renderWarpMode() string {
	// Update viewport content.
	m.viewport.SetContent(strings.Join(m.output, "\n"))

	if m.autoScroll {
		m.viewport.GotoBottom()
	}

	var b strings.Builder

	// Prompt at TOP.
	if !m.executing {
		b.WriteString(m.renderPromptForHistoryANSI())
	} else {
		b.WriteString(m.executingSpinner.View())
		b.WriteString(" ")
		b.WriteString(m.styles.Executing.Render("Executing..."))
		b.WriteString(" ")
	}

	// Input.
	b.WriteString(m.renderInputWithCursor())

	// Hints.
	b.WriteString(m.renderHints())

	b.WriteString("\n")

	// Separator.
	b.WriteString(strings.Repeat("─", m.width))
	b.WriteString("\n")

	// Output history at BOTTOM.
	b.WriteString(m.viewport.View())

	return b.String()
}

// renderCompactMode renders compact mode.
// Minimalist prompt, maximum space for output.
func (m Model) renderCompactMode() string {
	// Update viewport content.
	m.viewport.SetContent(strings.Join(m.output, "\n"))

	if m.autoScroll {
		m.viewport.GotoBottom()
	}

	var b strings.Builder

	// Output history at TOP.
	b.WriteString(m.viewport.View())
	b.WriteString("\n")

	// Executing indicator (compact).
	if m.executing {
		b.WriteString(m.executingSpinner.View())
		b.WriteString(" ")
	}

	// Compact prompt.
	if !m.executing {
		b.WriteString("$ ")
	}

	// Input.
	b.WriteString(m.renderInputWithCursor())

	// Hints (compact).
	b.WriteString(m.renderHints())

	return b.String()
}

// renderChatMode renders chat mode (Telegram/ChatGPT-like).
// Input fixed at bottom, history scrolls at top.
func (m Model) renderChatMode() string {
	// Update viewport content.
	m.viewport.SetContent(strings.Join(m.output, "\n"))

	if m.autoScroll {
		m.viewport.GotoBottom()
	}

	var b strings.Builder

	// Output history at TOP (main screen area).
	b.WriteString(m.viewport.View())
	b.WriteString("\n")

	// Separator.
	b.WriteString(strings.Repeat("─", m.width))
	b.WriteString("\n")

	// Executing indicator.
	if m.executing {
		b.WriteString(m.executingSpinner.View())
		b.WriteString(" ")
		b.WriteString(m.styles.Executing.Render("Executing..."))
		b.WriteString(" ")
	} else {
		// Compact prompt for chat mode.
		b.WriteString(m.styles.PromptArrow.Render("→ "))
	}

	// Input (fixed at bottom).
	b.WriteString(m.renderInputWithCursor())

	// Hints.
	b.WriteString(m.renderHints())

	return b.String()
}

// renderInputWithCursor renders input with visible cursor.
//
// Phase 2: Using Phoenix ShellInput with public ContentParts() API.
// This enables syntax highlighting + visible cursor (to be implemented in Phase 6).
//
// Current: Cursor is visible and works correctly.
// Next (Phase 6): Add syntax highlighting using ContentParts().
func (m Model) renderInputWithCursor() string {
	// Phoenix ShellInput with public cursor API!
	return m.shellInput.View()
}

// renderHints renders hints (completion, scroll indicator).
func (m Model) renderHints() string {
	var hints []string

	// Completion hint.
	if m.completionActive && len(m.completions) > 1 {
		hint := fmt.Sprintf("[Tab: %d/%d]", m.completionIndex+1, len(m.completions))
		hints = append(hints, m.styles.CompletionHint.Render(hint))
	}

	// Scroll indicator.
	if !m.autoScroll && m.viewport.ScrollPercent() < 0.99 {
		hints = append(hints, m.styles.CompletionHint.Render("[↑ scrolled]"))
	}

	if len(hints) > 0 {
		return " " + strings.Join(hints, " ")
	}

	return ""
}

// applySyntaxHighlight applies simple bash syntax highlighting WITHOUT Chroma.
func (m Model) applySyntaxHighlight(text string) string {
	if text == "" {
		return ""
	}

	// Simple highlighting: split into tokens by spaces.
	parts := strings.Fields(text)
	if len(parts) == 0 {
		return text
	}

	var result strings.Builder

	for i, part := range parts {
		if i > 0 {
			result.WriteString(" ") // Space between tokens.
		}

		switch {
		case i == 0:
			// First word = COMMAND (YELLOW).
			result.WriteString("\033[1;33m") // Bright Yellow.
			result.WriteString(part)
			result.WriteString("\033[0m")
		case strings.HasPrefix(part, "-"):
			// Option (GRAY).
			result.WriteString("\033[90m") // Dark Gray.
			result.WriteString(part)
			result.WriteString("\033[0m")
		default:
			// Argument (GREEN).
			result.WriteString("\033[32m") // Green.
			result.WriteString(part)
			result.WriteString("\033[0m")
		}
	}

	return result.String()
}

// renderPromptForHistoryANSI renders prompt for history (ANSI codes only).
func (m Model) renderPromptForHistoryANSI() string {
	const (
		reset      = "\033[0m"
		boldGreen  = "\033[1;32m" // username@hostname.
		blue       = "\033[34m"   // path.
		purple     = "\033[35m"   // git clean.
		boldYellow = "\033[1;33m" // git dirty.
	)

	username := os.Getenv("USER")
	if username == "" {
		username = os.Getenv("USERNAME")
	}
	hostname, _ := os.Hostname()

	workDir := m.currentSession.WorkingDirectory()
	displayPath := m.shortenPath(workDir)

	var result strings.Builder

	// username@hostname (bold green).
	result.WriteString(boldGreen)
	result.WriteString(fmt.Sprintf("%s@%s", username, hostname))
	result.WriteString(reset)
	result.WriteString(" ")

	// path (blue).
	result.WriteString(blue)
	result.WriteString(displayPath)
	result.WriteString(reset)

	// git status.
	if m.gitBranch != "" {
		result.WriteString(" ")
		if m.gitDirty {
			result.WriteString(boldYellow)
			result.WriteString(fmt.Sprintf("(%s *)", m.gitBranch))
			result.WriteString(reset)
		} else {
			result.WriteString(purple)
			result.WriteString(fmt.Sprintf("(%s)", m.gitBranch))
			result.WriteString(reset)
		}
	}

	// arrow (bold green).
	result.WriteString(" ")
	result.WriteString(boldGreen)
	result.WriteString("$")
	result.WriteString(reset)
	result.WriteString(" ")

	return result.String()
}

// renderWithHelpOverlay renders help overlay on top of main UI.
func (m Model) renderWithHelpOverlay() string {
	// Create help overlay.
	helpOverlay := m.renderHelpOverlay()

	// Place overlay at screen center.
	return lipgloss.Place(
		m.width, m.height,
		lipgloss.Center, lipgloss.Center,
		helpOverlay,
	)
}

// renderHelpOverlay creates modal help window.
func (m Model) renderHelpOverlay() string {
	// Style for overlay box.
	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("12")). // Blue.
		Padding(1, 2).
		Width(60).
		Background(lipgloss.Color("0")). // Black background.
		Foreground(lipgloss.Color("15")) // White text.

	titleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("11")). // Yellow.
		Bold(true)

	sectionStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("10")). // Green.
		Bold(true)

	keyStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("14")) // Cyan.

	var content strings.Builder

	// Title.
	content.WriteString(titleStyle.Render("GoSh Keyboard Shortcuts"))
	content.WriteString("\n\n")

	// Navigation.
	content.WriteString(sectionStyle.Render("Navigation:"))
	content.WriteString("\n")
	content.WriteString(keyStyle.Render("  ↑/↓       ") + " - Command history\n")
	content.WriteString(keyStyle.Render("  Tab       ") + " - Auto-complete\n")
	content.WriteString(keyStyle.Render("  PgUp/PgDn ") + " - Scroll output\n")
	content.WriteString("\n")

	// Input.
	content.WriteString(sectionStyle.Render("Input:"))
	content.WriteString("\n")
	content.WriteString(keyStyle.Render("  Enter     ") + " - Execute command\n")
	content.WriteString(keyStyle.Render("  Alt+Enter ") + " - Multi-line input\n")
	content.WriteString(keyStyle.Render("  Ctrl+L    ") + " - Clear screen\n")
	content.WriteString("\n")

	// UI Modes (if mode switching is allowed).
	if m.config.UI.AllowModeSwitching {
		content.WriteString(sectionStyle.Render("UI Modes:"))
		content.WriteString("\n")
		content.WriteString(keyStyle.Render("  Alt+1     ") + " - Classic mode\n")
		content.WriteString(keyStyle.Render("  Alt+2     ") + " - Warp mode\n")
		content.WriteString(keyStyle.Render("  Alt+3     ") + " - Compact mode\n")
		content.WriteString(keyStyle.Render("  Alt+4     ") + " - Chat mode\n")
		content.WriteString("\n")
	}

	// Help.
	content.WriteString(sectionStyle.Render("Help:"))
	content.WriteString("\n")
	content.WriteString(keyStyle.Render("  F1 or ?   ") + " - This help\n")
	content.WriteString(keyStyle.Render("  help      ") + " - Built-in commands\n")
	content.WriteString(keyStyle.Render("  ESC       ") + " - Close this help\n")
	content.WriteString("\n")

	// Exit.
	content.WriteString(sectionStyle.Render("Exit:"))
	content.WriteString("\n")
	content.WriteString(keyStyle.Render("  Ctrl+C/D  ") + " - Exit shell\n")
	content.WriteString(keyStyle.Render("  exit      ") + " - Exit shell\n")

	return boxStyle.Render(content.String())
}

// shortenPath shortens path for display.
func (m Model) shortenPath(path string) string {
	home, _ := os.UserHomeDir()

	// Replace home with ~.
	if strings.HasPrefix(path, home) {
		path = "~" + strings.TrimPrefix(path, home)
	}

	// If path is too long, show only last 3 components.
	parts := strings.Split(path, string(filepath.Separator))
	if len(parts) > 3 && !strings.HasPrefix(path, "~") {
		path = ".../" + strings.Join(parts[len(parts)-2:], "/")
	}

	return path
}

// makeProfessionalStyles creates professional styles like PowerShell/Git Bash.
func makeProfessionalStyles() Styles {
	return Styles{
		// Prompt - PowerShell/Bash inspired colors.
		PromptUser: lipgloss.NewStyle().
			Foreground(lipgloss.Color("10")). // Green.
			Bold(true),

		PromptPath: lipgloss.NewStyle().
			Foreground(lipgloss.Color("12")), // Blue.

		PromptGit: lipgloss.NewStyle().
			Foreground(lipgloss.Color("13")), // Purple.

		PromptGitDirty: lipgloss.NewStyle().
			Foreground(lipgloss.Color("11")). // Yellow.
			Bold(true),

		PromptArrow: lipgloss.NewStyle().
			Foreground(lipgloss.Color("10")). // Green.
			Bold(true),

		PromptError: lipgloss.NewStyle().
			Foreground(lipgloss.Color("9")). // Red.
			Bold(true),

		// Output.
		Output: lipgloss.NewStyle().
			Foreground(lipgloss.Color("15")), // White.

		OutputErr: lipgloss.NewStyle().
			Foreground(lipgloss.Color("9")), // Red.

		// Executing.
		Executing: lipgloss.NewStyle().
			Foreground(lipgloss.Color("12")). // Blue.
			Italic(true),

		// Completion hint.
		CompletionHint: lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")). // Gray.
			Italic(true),

		// Syntax highlighting (basic ANSI colors for compatibility).
		SyntaxCommand: lipgloss.NewStyle().
			Foreground(lipgloss.Color("11")). // Bright yellow (command bright).
			Bold(true),

		SyntaxOption: lipgloss.NewStyle().
			Foreground(lipgloss.Color("8")), // Dark gray (options dim).

		SyntaxArg: lipgloss.NewStyle().
			Foreground(lipgloss.Color("7")), // Light gray (arguments normal).

		SyntaxString: lipgloss.NewStyle().
			Foreground(lipgloss.Color("14")), // Cyan (strings).
	}
}
