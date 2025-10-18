package repl

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/grpmsoft/gosh/internal/domain/config"

	"github.com/phoenix-tui/phoenix/style/api"
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
	switch m.Config.UI.Mode {
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
	// While executing, don't render anything (output is being printed directly to stdout)
	// This prevents View() from overwriting command output with prompt
	if m.executing {
		return ""
	}

	var b strings.Builder

	// Phoenix writes View() at current cursor position without terminal control.
	// We need to:
	// 1. Return cursor to beginning of line (\r)
	// 2. Clear the line (\033[2K)
	// 3. Render prompt + input
	//
	// This ensures prompt is always visible and properly positioned.
	b.WriteString("\r\033[2K") // CR + clear entire line

	// Check multiline mode
	if m.multilineMode {
		// Multiline mode: render with continuation prompts
		b.WriteString(m.renderMultilineInput())
	} else {
		// Single-line mode: normal prompt + input
		b.WriteString(m.renderPromptForHistoryANSI())
		b.WriteString(m.renderInputWithCursor())
	}

	b.WriteString(m.renderHints())

	// Return prompt+input (no viewport, no history rendering).
	// History is already in terminal via fmt.Println() from Update().
	return b.String()
}

// renderWarpMode renders modern Warp-like mode.
// Prompt on top, output below with separator.
func (m Model) renderWarpMode() string {
	// Update viewport content (Phoenix fluent API with FollowMode)
	// FollowMode automatically scrolls to bottom when content changes
	m.viewport = m.viewport.FollowMode(m.autoScroll).SetContent(strings.Join(m.output, "\n"))

	var b strings.Builder

	// Prompt at TOP.
	if !m.executing {
		b.WriteString(m.renderPromptForHistoryANSI())
	} else {
		// Simple text indicator without spinner (Phoenix Progress can be added later)
		b.WriteString(style.Render(m.styles.Executing, "⟳ Executing..."))
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
	// Update viewport content (Phoenix fluent API with FollowMode)
	m.viewport = m.viewport.FollowMode(m.autoScroll).SetContent(strings.Join(m.output, "\n"))

	var b strings.Builder

	// Output history at TOP.
	b.WriteString(m.viewport.View())
	b.WriteString("\n")

	// Executing indicator (compact) or prompt.
	if m.executing {
		b.WriteString("⟳ ") // Simple rotating arrow icon
	} else {
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
	// Update viewport content (Phoenix fluent API with FollowMode)
	m.viewport = m.viewport.FollowMode(m.autoScroll).SetContent(strings.Join(m.output, "\n"))

	var b strings.Builder

	// Output history at TOP (main screen area).
	b.WriteString(m.viewport.View())
	b.WriteString("\n")

	// Separator.
	b.WriteString(strings.Repeat("─", m.width))
	b.WriteString("\n")

	// Executing indicator or prompt.
	if m.executing {
		// Simple text indicator without spinner (Phoenix Progress can be added later)
		b.WriteString(style.Render(m.styles.Executing, "⟳ Executing..."))
		b.WriteString(" ")
	} else {
		// Compact prompt for chat mode.
		b.WriteString(style.Render(m.styles.PromptArrow, "→ "))
	}

	// Input (fixed at bottom).
	b.WriteString(m.renderInputWithCursor())

	// Hints.
	b.WriteString(m.renderHints())

	return b.String()
}

// renderMultilineInput renders multiline input with continuation prompt.
//
// Delegates to ShellTextArea.ViewWithPrompts() which handles:
// - Phoenix TextArea rendering (with cursor!)
// - Adding prompts to each line
//
// First line uses existing renderPromptForHistoryANSI().
// Subsequent lines use continuation prompt ">>    " (4 spaces for alignment).
func (m Model) renderMultilineInput() string {
	const continuationPrompt = ">>    " // 4 spaces for alignment with normal prompt

	// Phoenix TextArea handles all rendering including cursor!
	// We just provide the prompts
	return m.shellTextArea.ViewWithPrompts(
		m.renderPromptForHistoryANSI(),
		continuationPrompt,
	)
}

// renderInputWithCursor renders input with visible cursor and syntax highlighting.
//
// Delegates to ShellInput.View() which:
//   1. Applies syntax highlighting (via highlightCallback)
//   2. Renders text using Phoenix Input
//   3. Positions terminal cursor correctly
func (m Model) renderInputWithCursor() string {
	// Update cursor visibility for blinking animation
	m.shellInput.SetCursorVisible(m.cursorVisible)

	// ShellInput handles everything: highlighting + cursor positioning
	return m.shellInput.View()
}

// renderHints renders hints (completion, scroll indicator).
func (m Model) renderHints() string {
	var hints []string

	// Completion hint.
	if m.completionActive && len(m.completions) > 1 {
		hint := fmt.Sprintf("[Tab: %d/%d]", m.completionIndex+1, len(m.completions))
		hints = append(hints, style.Render(m.styles.CompletionHint, hint))
	}

	// Scroll indicator (Phoenix Viewport: IsAtBottom instead of ScrollPercent)
	if !m.autoScroll && !m.viewport.IsAtBottom() {
		hints = append(hints, style.Render(m.styles.CompletionHint, "[↑ scrolled]"))
	}

	if len(hints) > 0 {
		return " " + strings.Join(hints, " ")
	}

	return ""
}

// applySyntaxHighlight applies simple bash syntax highlighting WITHOUT Chroma.
// IMPORTANT: Preserves ALL whitespace (spaces, tabs, etc.) to avoid cursor positioning issues!
func (m Model) applySyntaxHighlight(text string) string {
	if text == "" {
		return ""
	}

	var result strings.Builder
	var currentWord strings.Builder
	wordIndex := 0
	inWord := false

	for _, ch := range text {
		if ch == ' ' || ch == '\t' {
			// Whitespace - flush current word if any
			if inWord {
				// Highlight the word
				word := currentWord.String()
				switch {
				case wordIndex == 0:
					// First word = COMMAND (YELLOW)
					result.WriteString("\033[1;33m") // Bright Yellow
					result.WriteString(word)
					result.WriteString("\033[0m")
				case strings.HasPrefix(word, "-"):
					// Option (GRAY)
					result.WriteString("\033[90m") // Dark Gray
					result.WriteString(word)
					result.WriteString("\033[0m")
				default:
					// Argument (GREEN)
					result.WriteString("\033[32m") // Green
					result.WriteString(word)
					result.WriteString("\033[0m")
				}
				currentWord.Reset()
				wordIndex++
				inWord = false
			}
			// Preserve the whitespace character AS-IS
			result.WriteRune(ch)
		} else {
			// Non-whitespace - accumulate word
			currentWord.WriteRune(ch)
			inWord = true
		}
	}

	// Flush last word if any
	if inWord {
		word := currentWord.String()
		switch {
		case wordIndex == 0:
			// First word = COMMAND (YELLOW)
			result.WriteString("\033[1;33m")
			result.WriteString(word)
			result.WriteString("\033[0m")
		case strings.HasPrefix(word, "-"):
			// Option (GRAY)
			result.WriteString("\033[90m")
			result.WriteString(word)
			result.WriteString("\033[0m")
		default:
			// Argument (GREEN)
			result.WriteString("\033[32m")
			result.WriteString(word)
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

	// Calculate centering (simple centering without Phoenix render Place for now)
	lines := strings.Split(helpOverlay, "\n")
	maxLen := 0
	for _, line := range lines {
		if len(line) > maxLen {
			maxLen = len(line)
		}
	}

	verticalPadding := (m.height - len(lines)) / 2
	horizontalPadding := (m.width - maxLen) / 2

	var result strings.Builder
	for i := 0; i < verticalPadding; i++ {
		result.WriteString("\n")
	}
	for _, line := range lines {
		result.WriteString(strings.Repeat(" ", horizontalPadding))
		result.WriteString(line)
		result.WriteString("\n")
	}

	return result.String()
}

// renderHelpOverlay creates modal help window.
func (m Model) renderHelpOverlay() string {
	// Style for overlay box using Phoenix Style
	boxStyle := style.New().
		Border(style.RoundedBorder).
		BorderColor(style.Color256(12)). // Blue.
		Padding(style.NewPadding(1, 2, 1, 2)).
		Width(60).
		Background(style.Color256(0)). // Black background.
		Foreground(style.Color256(15)) // White text.

	titleStyle := style.New().
		Foreground(style.Color256(11)). // Yellow.
		Bold(true)

	sectionStyle := style.New().
		Foreground(style.Color256(10)). // Green.
		Bold(true)

	keyStyle := style.New().
		Foreground(style.Color256(14)) // Cyan.

	var content strings.Builder

	// Title.
	content.WriteString(style.Render(titleStyle, "GoSh Keyboard Shortcuts"))
	content.WriteString("\n\n")

	// Navigation.
	content.WriteString(style.Render(sectionStyle, "Navigation:"))
	content.WriteString("\n")
	content.WriteString(style.Render(keyStyle, "  ↑/↓       ") + " - Command history\n")
	content.WriteString(style.Render(keyStyle, "  Tab       ") + " - Auto-complete\n")
	content.WriteString(style.Render(keyStyle, "  PgUp/PgDn ") + " - Scroll output\n")
	content.WriteString("\n")

	// Input.
	content.WriteString(style.Render(sectionStyle, "Input:"))
	content.WriteString("\n")
	content.WriteString(style.Render(keyStyle, "  Enter     ") + " - Execute command\n")
	content.WriteString(style.Render(keyStyle, "  Alt+Enter ") + " - Multi-line input\n")
	content.WriteString(style.Render(keyStyle, "  Ctrl+L    ") + " - Clear screen\n")
	content.WriteString("\n")

	// UI Modes (if mode switching is allowed).
	if m.Config.UI.AllowModeSwitching {
		content.WriteString(style.Render(sectionStyle, "UI Modes:"))
		content.WriteString("\n")
		content.WriteString(style.Render(keyStyle, "  Alt+1     ") + " - Classic mode\n")
		content.WriteString(style.Render(keyStyle, "  Alt+2     ") + " - Warp mode\n")
		content.WriteString(style.Render(keyStyle, "  Alt+3     ") + " - Compact mode\n")
		content.WriteString(style.Render(keyStyle, "  Alt+4     ") + " - Chat mode\n")
		content.WriteString("\n")
	}

	// Help.
	content.WriteString(style.Render(sectionStyle, "Help:"))
	content.WriteString("\n")
	content.WriteString(style.Render(keyStyle, "  F1 or ?   ") + " - This help\n")
	content.WriteString(style.Render(keyStyle, "  help      ") + " - Built-in commands\n")
	content.WriteString(style.Render(keyStyle, "  ESC       ") + " - Close this help\n")
	content.WriteString("\n")

	// Exit.
	content.WriteString(style.Render(sectionStyle, "Exit:"))
	content.WriteString("\n")
	content.WriteString(style.Render(keyStyle, "  Ctrl+C/D  ") + " - Exit shell\n")
	content.WriteString(style.Render(keyStyle, "  exit      ") + " - Exit shell\n")

	return style.Render(boxStyle, content.String())
}

// countVisibleChars counts visible characters in a string, skipping ANSI escape sequences.
// This is needed for correct cursor positioning when syntax highlighting is applied.
func countVisibleChars(s string) int {
	count := 0
	inEscape := false

	for _, r := range s {
		if r == '\033' {
			// Start of ANSI escape sequence
			inEscape = true
			continue
		}

		if inEscape {
			// Skip until we find 'm' (end of color code) or other terminator
			if r == 'm' || r == 'H' || r == 'J' || r == 'K' || r == 'A' || r == 'B' || r == 'C' || r == 'D' {
				inEscape = false
			}
			continue
		}

		// Visible character
		count++
	}

	return count
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
// Uses Phoenix TUI Framework's style library.
func makeProfessionalStyles() Styles {
	return Styles{
		// Prompt - PowerShell/Bash inspired colors.
		PromptUser: style.New().
			Foreground(style.Color256(10)). // Green.
			Bold(true),

		PromptPath: style.New().
			Foreground(style.Color256(12)), // Blue.

		PromptGit: style.New().
			Foreground(style.Color256(13)), // Purple.

		PromptGitDirty: style.New().
			Foreground(style.Color256(11)). // Yellow.
			Bold(true),

		PromptArrow: style.New().
			Foreground(style.Color256(10)). // Green.
			Bold(true),

		PromptError: style.New().
			Foreground(style.Color256(9)). // Red.
			Bold(true),

		// Output.
		Output: style.New().
			Foreground(style.Color256(15)), // White.

		OutputErr: style.New().
			Foreground(style.Color256(9)), // Red.

		// Executing.
		Executing: style.New().
			Foreground(style.Color256(12)). // Blue.
			Italic(true),

		// Completion hint.
		CompletionHint: style.New().
			Foreground(style.Color256(240)). // Gray.
			Italic(true),

		// Syntax highlighting (basic ANSI colors for compatibility).
		SyntaxCommand: style.New().
			Foreground(style.Color256(11)). // Bright yellow (command bright).
			Bold(true),

		SyntaxOption: style.New().
			Foreground(style.Color256(8)), // Dark gray (options dim).

		SyntaxArg: style.New().
			Foreground(style.Color256(7)), // Light gray (arguments normal).

		SyntaxString: style.New().
			Foreground(style.Color256(14)), // Cyan (strings).
	}
}
