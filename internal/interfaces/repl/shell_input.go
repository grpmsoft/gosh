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
	// Disable Phoenix cursor rendering - we'll use the terminal's native cursor instead
	// This gives us a real blinking cursor like PowerShell, instead of reverse video
	inp := input.New(width).Focused(true).ShowCursor(false)

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
	s.base = s.base.Width(width).ShowCursor(false)
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

// View renders the input WITHOUT syntax highlighting.
//
// ═══════════════════════════════════════════════════════════════════════════
// CRITICAL FIX: Syntax Highlighting Removed!
// ═══════════════════════════════════════════════════════════════════════════
//
// Previously: Applied syntax highlighting in View(), which caused:
//   - Performance issues (highlighting on every render)
//   - Cursor positioning errors (ANSI codes changed text length)
//   - Input lag (especially with arrow keys)
//
// Solution: Use Phoenix Input's View() directly
//   - No syntax highlighting (can be added later with proper implementation)
//   - Terminal cursor positioned correctly (ShowCursor(false) + \033[{n}D)
//   - Fast and responsive!
//
// Terminal cursor positioning:
//   1. Phoenix renders plain text (ShowCursor(false) = no reverse video)
//   2. Get text after cursor from ContentParts()
//   3. Move cursor LEFT by length of "after"
//   4. Terminal cursor now at correct position!
//
// Example: "hello world" with cursor at position 6
//   - Phoenix renders: "hello world"
//   - after = "world" (5 chars)
//   - Move LEFT 5: \033[5D
//   - Cursor at position 6 ✓
//
// ═══════════════════════════════════════════════════════════════════════════
func (s *ShellInput) View() string {
	// Get plain text from Phoenix Input
	view := s.base.View()

	// Get text after cursor for positioning
	_, _, after := s.base.ContentParts()

	// Calculate how many columns to move LEFT
	afterLen := len([]rune(after))

	if afterLen > 0 {
		// Render text + move cursor left to correct position
		return fmt.Sprintf("%s\033[%dD", view, afterLen)
	}

	// Cursor already at end
	return view
}

// Note: Syntax highlighting is now handled by Model.applySyntaxHighlight callback.
// The old complex character-by-character highlighting has been removed in favor of
// simple word-based highlighting that works correctly during typing.
