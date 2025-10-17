package repl

import (
	"fmt"
	"strings"

	input "github.com/phoenix-tui/phoenix/components/input/api"
	"github.com/phoenix-tui/phoenix/tea/api"
	"github.com/grpmsoft/gosh/internal/domain/history"
)

// ShellInput wraps Phoenix TextInput with shell-specific features.
//
// Key Features:
// - Public cursor position API (ContentParts method) ⭐ KEY DIFFERENTIATOR
// - History navigation (Up/Down arrows)
// - Syntax highlighting support (Phase 6)
// - Emoji/Unicode correct rendering
// - Cursor blinking animation
//
// This wrapper adds shell-specific functionality to the universal Phoenix TextInput component.
type ShellInput struct {
	base          *input.Input
	history       *history.History
	historyNav    *history.Navigator
	cursorVisible bool // For blinking animation (controlled by parent Model)
}

// NewShellInput creates a new shell input component.
// width: visible width in columns
// hist: command history for Up/Down navigation
func NewShellInput(width int, hist *history.History) *ShellInput {
	inp := input.New(width).Focused(true)

	return &ShellInput{
		base:          inp,
		history:       hist,
		historyNav:    hist.NewNavigator(),
		cursorVisible: true, // Start with cursor visible
	}
}

// ContentParts returns (before cursor, at cursor, after cursor).
// This PUBLIC API enables syntax highlighting + visible cursor!
// This is THE KEY FEATURE we migrated to Phoenix for.
func (s *ShellInput) ContentParts() (string, string, string) {
	return s.base.ContentParts()
}

// Value returns current input text.
func (s *ShellInput) Value() string {
	return s.base.Value()
}

// SetValue sets input text and moves cursor to end.
func (s *ShellInput) SetValue(text string) {
	s.base = s.base.SetContent(text, len(text))
}

// Reset clears input and moves cursor to start.
func (s *ShellInput) Reset() {
	s.base = s.base.SetContent("", 0)
}

// SetWidth updates input width.
func (s *ShellInput) SetWidth(width int) {
	s.base = s.base.Width(width)
}

// Focus gives focus to input.
func (s *ShellInput) Focus() {
	s.base = s.base.Focused(true)
}

// Blur removes focus from input.
func (s *ShellInput) Blur() {
	s.base = s.base.Focused(false)
}

// SetCursorVisible sets cursor visibility for blinking animation.
func (s *ShellInput) SetCursorVisible(visible bool) {
	s.cursorVisible = visible
}

// Update handles input events.
//
// Special handling:
// - Up arrow: History navigation (previous command)
// - Down arrow: History navigation (next command)
// - All other keys: Delegate to base TextInput
//
// Returns updated ShellInput and any commands to execute.
func (s *ShellInput) Update(msg api.Msg) (*ShellInput, api.Cmd) {
	switch msg := msg.(type) {
	case api.KeyMsg:
		switch msg.Type {
		case api.KeyUp:
			// History navigation (Up arrow - older commands)
			if cmd, ok := s.historyNav.Backward(); ok {
				s.SetValue(cmd)
				return s, nil
			}
			// If no history or already at oldest, ignore
			return s, nil

		case api.KeyDown:
			// History navigation (Down arrow - newer commands)
			if cmd, ok := s.historyNav.Forward(); ok {
				s.SetValue(cmd)
				return s, nil
			}
			// End of history - clear input (standard shell behavior)
			s.Reset()
			return s, nil
		}
	}

	// Delegate all other events to base TextInput
	var cmd api.Cmd
	s.base, cmd = s.base.Update(msg)
	return s, cmd
}

// View renders the input with syntax highlighting.
//
// PowerShell approach: Apply syntax highlighting to FULL text (preserves context!),
// then position system cursor using ANSI escape codes.
// The terminal automatically inverts the character under cursor.
func (s *ShellInput) View() string {
	before, at, after := s.ContentParts()
	fullText := before + at + after

	// Apply syntax highlighting to ENTIRE text (preserves command/argument context!)
	highlighted := applySyntaxHighlighting(fullText)

	// After rendering highlighted text, move cursor back to correct position
	// We render the full highlighted text, then move cursor left
	charsAfterCursor := countVisibleChars(at + after)
	if charsAfterCursor > 0 {
		// Move cursor left by number of characters after cursor position
		// ANSI: \033[{n}D = move left n columns
		highlighted += fmt.Sprintf("\033[%dD", charsAfterCursor)
	}

	return highlighted
}

// renderCursor renders visible blinking cursor character.
// Uses ANSI escape codes for reverse video (background <-> foreground swap).
func renderCursor(char string) string {
	if char == "" {
		// At end of line - render block cursor
		char = " "
	}
	// ANSI escape: ESC[7m = reverse video, ESC[27m = normal video
	return "\033[7m" + char + "\033[27m"
}

// renderCursorWithBlink renders cursor with blinking support.
//
// Blinking behavior:
// - cursorVisible = true: render cursor with reverse video (inverted colors)
// - cursorVisible = false: render plain character (cursor invisible)
//
// This creates the blinking effect when combined with ticker that toggles cursorVisible.
func (s *ShellInput) renderCursorWithBlink(char string) string {
	if char == "" {
		// At end of line - use space for cursor
		char = " "
	}

	// If cursor not visible (blink off state), render plain character
	if !s.cursorVisible {
		return char
	}

	// Cursor visible (blink on state) - apply reverse video (invert colors)
	// ANSI: \033[7m = reverse video, \033[27m = normal video
	return "\033[7m" + char + "\033[27m"
}

// applySyntaxHighlighting applies ANSI color codes to shell syntax elements.
//
// Highlighting rules (matching repl_render.go for consistency):
// - Commands (first word): Bright Yellow + Bold (\033[1;33m) - stands out!
// - Options (-x, --long): Dark Gray (\033[90m) - subdued
// - Arguments: Green (\033[32m) - visible but not dominant
// - Strings ("...", '...'): Yellow (\033[33m)
// - Pipes (|): Bold white (\033[1;37m)
// - Redirects (>, >>, <): Bold white (\033[1;37m)
// - Variables ($VAR, ${VAR}): Magenta (\033[35m)
// - Operators (&&, ||, ;): Bold white (\033[1;37m)
//
// IMPORTANT: Currently only works with ASCII.
// For UTF-8 (Russian, Chinese, etc.), returns text as-is to avoid breaking multi-byte sequences.
func applySyntaxHighlighting(text string) string {
	if text == "" {
		return text
	}

	// TEMPORARY: Check if text contains non-ASCII (UTF-8) characters
	// If yes, return as-is to avoid breaking multi-byte sequences
	// TODO: Rewrite to work with runes instead of bytes
	for _, b := range []byte(text) {
		if b >= 0x80 {
			// Contains UTF-8 multi-byte character - return as-is (no highlighting)
			return text
		}
	}

	// Work character by character to preserve cursor position (ASCII only)
	var result strings.Builder
	inString := false
	inSingleQuote := false
	inDoubleQuote := false
	wordStart := 0
	isFirstWord := true

	for i := 0; i < len(text); i++ {
		ch := text[i]

		// Handle quotes
		if ch == '"' && (i == 0 || text[i-1] != '\\') {
			if !inSingleQuote {
				if !inDoubleQuote {
					result.WriteString("\033[33m") // Start yellow
				} else {
					result.WriteString(string(ch))
					result.WriteString("\033[0m") // End yellow
					inString = false
					inDoubleQuote = false
					wordStart = i + 1
					continue
				}
				inDoubleQuote = !inDoubleQuote
				inString = true
			}
			result.WriteString(string(ch))
			continue
		}

		if ch == '\'' && (i == 0 || text[i-1] != '\\') {
			if !inDoubleQuote {
				if !inSingleQuote {
					result.WriteString("\033[33m") // Start yellow
				} else {
					result.WriteString(string(ch))
					result.WriteString("\033[0m") // End yellow
					inString = false
					inSingleQuote = false
					wordStart = i + 1
					continue
				}
				inSingleQuote = !inSingleQuote
				inString = true
			}
			result.WriteString(string(ch))
			continue
		}

		// If inside string, just add character
		if inString {
			result.WriteString(string(ch))
			continue
		}

		// Handle special characters
		if ch == '|' || ch == '>' || ch == '<' || ch == ';' {
			// Pipes, redirects, semicolons - bold white
			result.WriteString("\033[1;37m")
			result.WriteString(string(ch))
			result.WriteString("\033[0m")
			wordStart = i + 1
			isFirstWord = false
			continue
		}

		// Handle space (word boundary)
		if ch == ' ' {
			result.WriteString(string(ch))
			wordStart = i + 1
			isFirstWord = false
			continue
		}

		// Handle operators (&&, ||)
		if i < len(text)-1 && ((ch == '&' && text[i+1] == '&') || (ch == '|' && text[i+1] == '|')) {
			result.WriteString("\033[1;37m")
			result.WriteString(string(ch))
			result.WriteString(string(text[i+1]))
			result.WriteString("\033[0m")
			i++ // Skip next character
			wordStart = i + 1
			isFirstWord = false
			continue
		}

		// Handle variables ($VAR, ${VAR})
		if ch == '$' {
			result.WriteString("\033[35m") // Magenta
			result.WriteString(string(ch))
			// Continue until end of variable name
			j := i + 1
			if j < len(text) && text[j] == '{' {
				// ${VAR} syntax
				result.WriteString("{")
				j++
				for j < len(text) && text[j] != '}' {
					result.WriteString(string(text[j]))
					j++
				}
				if j < len(text) {
					result.WriteString("}")
				}
			} else {
				// $VAR syntax
				for j < len(text) && (isAlphaNumeric(text[j]) || text[j] == '_') {
					result.WriteString(string(text[j]))
					j++
				}
			}
			result.WriteString("\033[0m")
			i = j - 1
			continue
		}

		// Regular word - determine color
		if wordStart == i {
			// Start of word
			word := extractWord(text[i:])
			if strings.HasPrefix(word, "-") {
				// Option - dark gray (subdued)
				result.WriteString("\033[90m")
				result.WriteString(string(ch))
			} else if isFirstWord {
				// Command (first word) - bright yellow + bold (stands out!)
				result.WriteString("\033[1;33m")
				result.WriteString(string(ch))
			} else {
				// Argument - green (visible but not dominant)
				result.WriteString("\033[32m")
				result.WriteString(string(ch))
			}
		} else {
			// Middle of word
			result.WriteString(string(ch))
		}

		// Check if end of word
		if i == len(text)-1 || text[i+1] == ' ' || text[i+1] == '|' || text[i+1] == '>' || text[i+1] == '<' {
			// End current color for all colored words (command, option, argument)
			word := text[wordStart : i+1]
			if strings.HasPrefix(word, "-") || (isFirstWord && wordStart == 0) || (!isFirstWord && wordStart > 0) {
				result.WriteString("\033[0m")
			}
		}
	}

	return result.String()
}

// Helper functions for syntax highlighting

func isOption(text string) bool {
	words := strings.Fields(text)
	if len(words) == 0 {
		return false
	}
	lastWord := words[len(words)-1]
	return strings.HasPrefix(lastWord, "-")
}

func isInString(text string) bool {
	doubleQuotes := strings.Count(text, "\"")
	singleQuotes := strings.Count(text, "'")
	// Simplified check - odd number means we're inside a string
	return (doubleQuotes%2 == 1) || (singleQuotes%2 == 1)
}

func isCommand(text string) bool {
	// Command is the first word before any space/pipe/redirect
	trimmed := strings.TrimSpace(text)
	return !strings.ContainsAny(trimmed, " |><;")
}

func extractWord(text string) string {
	for i, ch := range text {
		if ch == ' ' || ch == '|' || ch == '>' || ch == '<' || ch == ';' {
			return text[:i]
		}
	}
	return text
}

func isAlphaNumeric(ch byte) bool {
	return (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9')
}

// countVisibleChars counts visible characters in text, skipping ANSI escape codes.
//
// PowerShell approach: When positioning cursor, we need to count only visible characters,
// ignoring ANSI color codes (\033[...m).
//
// Example:
//
//	text = "\033[32mecho\033[0m hello"  // "echo hello" with green "echo"
//	countVisibleChars(text) == 10       // 4 (echo) + 1 (space) + 5 (hello)
func countVisibleChars(text string) int {
	count := 0
	i := 0

	for i < len(text) {
		// ANSI escape sequence starts with ESC (0x1B or \033)
		if text[i] == '\033' && i+1 < len(text) && text[i+1] == '[' {
			// Skip ESC and [
			i += 2

			// Skip until we find 'm' (end of ANSI sequence)
			for i < len(text) && text[i] != 'm' {
				i++
			}

			// Skip the 'm'
			if i < len(text) {
				i++
			}

			continue
		}

		// Regular visible character
		count++
		i++
	}

	return count
}
