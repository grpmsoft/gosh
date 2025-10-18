package repl

import (
	"strings"

	textarea "github.com/phoenix-tui/phoenix/components/input/textarea/api"
	"github.com/phoenix-tui/phoenix/tea/api"
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
	base              textarea.TextArea
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
	ta := textarea.New().Size(width, height)

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
	s.base = s.base.SetValue(text)
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
func (s *ShellTextArea) Update(msg api.Msg) (*ShellTextArea, api.Cmd) {
	// Delegate ALL events to base TextArea
	// No special handling needed - TextArea handles everything
	var cmd api.Cmd
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
func (s *ShellTextArea) ViewWithPrompts(primaryPrompt, continuationPrompt string) string {
	// Get rendered view from Phoenix TextArea (with cursor!)
	rendered := s.base.View()

	// Split into lines
	lines := strings.Split(rendered, "\n")

	// Add prompts to each line
	var result strings.Builder
	for i, line := range lines {
		if i == 0 {
			result.WriteString(primaryPrompt)
		} else {
			result.WriteString(continuationPrompt)
		}
		result.WriteString(line)

		// Add newline (except last line)
		if i < len(lines)-1 {
			result.WriteString("\n")
		}
	}

	return result.String()
}
