package repl

import (
	"fmt"
	"strings"

	"github.com/phoenix-tui/phoenix/components/input"
	"github.com/phoenix-tui/phoenix/tea"
	"github.com/grpmsoft/gosh/internal/domain/history"
)

// ShellTextArea wraps Phoenix TextArea with shell-specific features.
//
// Key Features:
// - Multiline editing support
// - Public cursor position API (CursorPosition, ContentParts methods)
// - History navigation (Up/Down arrows)
// - Syntax highlighting support (via callback)
// - Line-based API (Lines method)
//
// This wrapper adds shell-specific functionality to the universal Phoenix TextArea component.
type ShellTextArea struct {
	base              input.TextArea
	history           *history.History
	historyNav        *history.Navigator
	highlightCallback func(string) string // Callback for syntax highlighting
}

// NewShellTextArea creates a new shell textarea component.
// width: visible width in columns
// height: visible height in rows
// hist: command history for Up/Down navigation
// highlight: callback for syntax highlighting (use Model.applySyntaxHighlight)
func NewShellTextArea(width, height int, hist *history.History, highlight func(string) string) *ShellTextArea {
	// Disable Phoenix cursor rendering - we'll use the terminal's native cursor instead
	// This gives us correct cursor positioning without inserting "█" character into text
	ta := input.NewTextArea().Size(width, height).ShowCursor(false)

	return &ShellTextArea{
		base:              ta,
		history:           hist,
		historyNav:        hist.NewNavigator(),
		highlightCallback: highlight,
	}
}

// Value returns current textarea text as single string.
func (s *ShellTextArea) Value() string {
	if s == nil {
		return ""
	}
	return s.base.Value()
}

// SetValue sets textarea text and moves cursor to end.
func (s *ShellTextArea) SetValue(text string) {
	if s == nil {
		return
	}
	// CRITICAL: SetValue() resets cursor to (0,0), so we MUST call MoveCursorToEnd()
	// to place cursor at end of text (for multiline editing)
	s.base = s.base.SetValue(text).MoveCursorToEnd()
}

// Lines returns all lines in the textarea.
func (s *ShellTextArea) Lines() []string {
	if s == nil {
		return []string{}
	}
	return s.base.Lines()
}

// Reset clears textarea and moves cursor to start.
func (s *ShellTextArea) Reset() {
	if s == nil {
		return
	}
	s.base = s.base.SetValue("")
}

// SetSize updates textarea dimensions.
func (s *ShellTextArea) SetSize(width, height int) {
	if s == nil {
		return
	}
	s.base = s.base.Size(width, height)
}

// CursorPosition returns current cursor position (row, col).
func (s *ShellTextArea) CursorPosition() (row, col int) {
	if s == nil {
		return 0, 0
	}
	return s.base.CursorPosition()
}

// ContentParts returns (before cursor, at cursor, after cursor).
// This PUBLIC API enables syntax highlighting with visible cursor!
func (s *ShellTextArea) ContentParts() (string, string, string) {
	if s == nil {
		return "", "", ""
	}
	return s.base.ContentParts()
}

// Update handles textarea events.
//
// In multiline mode, all keys are handled by Phoenix TextArea:
// - Up/Down arrows: Move cursor between lines (NOT history navigation)
// - Ctrl+P/Ctrl+N: Move up/down (Emacs keybindings)
// - Ctrl+A/Ctrl+E: Move to line start/end
// - Ctrl+K/Ctrl+Y: Kill/yank (Emacs kill ring)
// - etc.
//
// NOTE: History navigation is NOT available in multiline mode.
// To use history, exit multiline mode (execute or clear the textarea).
// This matches bash behavior - multiline editing prioritizes cursor navigation.
//
// Returns updated ShellTextArea and any commands to execute.
func (s *ShellTextArea) Update(msg tea.Msg) (*ShellTextArea, tea.Cmd) {
	// Delegate ALL events to base TextArea
	// No special handling needed - TextArea handles everything
	var cmd tea.Cmd
	s.base, cmd = s.base.Update(msg)
	return s, cmd
}

// View renders the textarea with syntax highlighting.
//
// Unlike ShellInput which manually applies highlighting character-by-character,
// ShellTextArea delegates rendering to Phoenix TextArea and applies highlighting
// to the entire content.
//
// This is simpler but may have different cursor positioning behavior compared to ShellInput.
func (s *ShellTextArea) View() string {
	// Phoenix TextArea handles cursor rendering internally
	return s.base.View()
}

// ViewWithPrompts renders the textarea with custom prompts for each line.
// primaryPrompt: prompt for first line (e.g., "gosh> ")
// continuationPrompt: prompt for subsequent lines (e.g., ">>    ")
//
// Algorithm (similar to ShellInput.View()):
// 1. Get plain text content
// 2. Apply syntax highlighting to entire text
// 3. Split highlighted text into lines
// 4. Add prompts to each line
// 5. Calculate cursor position (row, col) with prompt offset
// 6. Render all lines + position cursor with ANSI codes
func (s *ShellTextArea) ViewWithPrompts(primaryPrompt, continuationPrompt string) string {
	// Get plain text content
	plainText := s.base.Value()

	// Apply syntax highlighting (if callback provided)
	var highlightedText string
	if s.highlightCallback != nil {
		highlightedText = s.highlightCallback(plainText)
	} else {
		highlightedText = plainText
	}

	// Split highlighted text into lines
	highlightedLines := strings.Split(highlightedText, "\n")

	// Get cursor position in PLAIN text (row, col)
	cursorRow, cursorCol := s.base.CursorPosition()

	// Build output with prompts
	var result strings.Builder
	for i, line := range highlightedLines {
		// Add prompt
		var prompt string
		if i == 0 {
			prompt = primaryPrompt
		} else {
			prompt = continuationPrompt
		}
		result.WriteString(prompt)
		result.WriteString(line)

		// Add newline (except last line)
		if i < len(highlightedLines)-1 {
			result.WriteString("\n")
		}
	}

	// Calculate cursor positioning
	// Terminal cursor is at end of rendered text, we need to move it to correct position

	// 1. Count how many lines are AFTER cursor row
	linesAfterCursor := len(highlightedLines) - cursorRow - 1

	// 2. Get current line's visual length (to know where cursor should be)
	currentLineHighlighted := highlightedLines[cursorRow]
	currentLineVisualLen := countVisibleChars(currentLineHighlighted)

	// 3. Calculate how many chars to move left from end of current line
	moveLeft := currentLineVisualLen - cursorCol

	// Clamp to prevent negative
	if moveLeft < 0 {
		moveLeft = 0
	}

	// 4. Position cursor using ANSI codes
	var cursorPositioning strings.Builder

	// Move up if cursor is not on last line
	if linesAfterCursor > 0 {
		cursorPositioning.WriteString(fmt.Sprintf("\033[%dA", linesAfterCursor))
	}

	// Move to correct column: move to start of line, then right by (prompt + cursorCol)
	// Alternative: move left from end by moveLeft
	if moveLeft > 0 {
		cursorPositioning.WriteString(fmt.Sprintf("\033[%dD", moveLeft))
	}

	return result.String() + cursorPositioning.String()
}
