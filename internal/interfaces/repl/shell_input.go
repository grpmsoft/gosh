package repl

import (
	"fmt"

	input "github.com/phoenix-tui/phoenix/components/input/api"
	"github.com/phoenix-tui/phoenix/tea/api"
	"github.com/grpmsoft/gosh/internal/domain/history"
)

// ShellInput wraps Phoenix TextInput with shell-specific features.
//
// Key Features:
// - Public cursor position API (ContentParts method) ⭐ KEY DIFFERENTIATOR
// - History navigation (Up/Down arrows)
// - Syntax highlighting support (via callback)
// - Emoji/Unicode correct rendering
// - Cursor blinking state (delegated to Phoenix TextInput)
//
// This wrapper adds shell-specific functionality to the universal Phoenix TextInput component.
type ShellInput struct {
	base              *input.Input
	history           *history.History
	historyNav        *history.Navigator
	cursorVisible     bool                 // Cursor blink state (controlled by parent Model tick)
	highlightCallback func(string) string  // Callback for syntax highlighting
}

// NewShellInput creates a new shell input component.
// width: visible width in columns
// hist: command history for Up/Down navigation
// highlight: callback for syntax highlighting (use Model.applySyntaxHighlight)
func NewShellInput(width int, hist *history.History, highlight func(string) string) *ShellInput {
	inp := input.New(width).Focused(true)

	return &ShellInput{
		base:              inp,
		history:           hist,
		historyNav:        hist.NewNavigator(),
		cursorVisible:     true,  // Start with cursor visible
		highlightCallback: highlight,
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
// ═══════════════════════════════════════════════════════════════════════════
// IMPORTANT: Cursor Blinking
// ═══════════════════════════════════════════════════════════════════════════
//
// Cursor blinking should be implemented in Phoenix TextInput component,
// not in the application layer (GoSh). This keeps the separation of concerns:
//
// - Phoenix TextInput: Handles cursor rendering and blinking (UI concern)
// - GoSh ShellInput: Handles syntax highlighting and history (app concern)
//
// Currently: Cursor is visible but does NOT blink (Phoenix TextInput limitation)
// TODO: Add blinking support to Phoenix TextInput (tick command + cursorVisible flag)
//
// ═══════════════════════════════════════════════════════════════════════════
func (s *ShellInput) View() string {
	before, at, after := s.ContentParts()
	fullText := before + at + after

	// Apply syntax highlighting via callback (from Model.applySyntaxHighlight)
	// This uses simple word-based highlighting that works correctly during typing
	highlighted := s.highlightCallback(fullText)

	// Position cursor: move left from end of highlighted text
	// CRITICAL: Use rune count, not byte count (fixes Russian/Chinese positioning!)
	runesAfterCursor := len([]rune(at + after))
	if runesAfterCursor > 0 {
		// ANSI: \033[{n}D = move cursor left n columns
		highlighted += fmt.Sprintf("\033[%dD", runesAfterCursor)
	}

	return highlighted
}

// Note: Syntax highlighting is now handled by Model.applySyntaxHighlight callback.
// The old complex character-by-character highlighting has been removed in favor of
// simple word-based highlighting that works correctly during typing.
