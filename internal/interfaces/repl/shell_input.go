package repl

import (
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
//
// This wrapper adds shell-specific functionality to the universal Phoenix TextInput component.
type ShellInput struct {
	base       *input.Input
	history    *history.History
	historyNav *history.Navigator
}

// NewShellInput creates a new shell input component.
// width: visible width in columns
// hist: command history for Up/Down navigation
func NewShellInput(width int, hist *history.History) *ShellInput {
	inp := input.New(width).Focused(true)

	return &ShellInput{
		base:       inp,
		history:    hist,
		historyNav: hist.NewNavigator(),
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

// View renders the input with syntax highlighting + cursor.
//
// Phase 6: Syntax highlighting with visible cursor (COMPLETE).
//
// The cursor is ALWAYS visible when typing - this is the main goal of Phase 2!
// Syntax highlighting is applied to text before and after cursor separately.
func (s *ShellInput) View() string {
	before, at, after := s.ContentParts()

	// Apply syntax highlighting to parts before and after cursor
	highlightedBefore := applySyntaxHighlighting(before)
	highlightedAfter := applySyntaxHighlighting(after)

	// Render cursor with syntax highlighting applied to character under cursor
	highlightedCursor := renderCursorWithHighlight(at, before)

	return highlightedBefore + highlightedCursor + highlightedAfter
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

// renderCursorWithHighlight renders cursor with syntax highlighting context.
// Determines the color of the character under cursor based on what comes before it.
//
// IMPORTANT: Apply reverse video to plain character, NOT colored one!
// If we apply reverse video to coloredChar, the \033[0m reset code will cancel reverse video.
func renderCursorWithHighlight(char string, before string) string {
	if char == "" {
		// At end of line - render block cursor
		char = " "
	}

	// Apply reverse video to PLAIN character for cursor visibility
	// This ensures cursor is always visible regardless of syntax highlighting
	// ANSI: \033[7m = reverse video, \033[27m = normal video
	return "\033[7m" + char + "\033[27m"
}

// applySyntaxHighlighting applies ANSI color codes to shell syntax elements.
//
// Highlighting rules:
// - Commands (first word): Green
// - Options (-x, --long): Cyan
// - Strings ("...", '...'): Yellow
// - Pipes (|): Bold white
// - Redirects (>, >>, <): Bold white
// - Variables ($VAR, ${VAR}): Magenta
// - Operators (&&, ||, ;): Bold white
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
				// Option - cyan
				result.WriteString("\033[36m")
				result.WriteString(string(ch))
			} else if isFirstWord {
				// Command (first word) - green
				result.WriteString("\033[32m")
				result.WriteString(string(ch))
			} else {
				// Argument - default color
				result.WriteString(string(ch))
			}
		} else {
			// Middle of word
			result.WriteString(string(ch))
		}

		// Check if end of word
		if i == len(text)-1 || text[i+1] == ' ' || text[i+1] == '|' || text[i+1] == '>' || text[i+1] == '<' {
			// End current color
			word := text[wordStart : i+1]
			if strings.HasPrefix(word, "-") || (isFirstWord && wordStart == 0) {
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
