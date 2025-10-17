package repl

import (
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
// Phase 2: Simple rendering with visible cursor (no syntax highlighting yet).
// Phase 6: Will add syntax highlighting using ContentParts().
//
// The cursor is ALWAYS visible when typing - this is the main goal of Phase 2!
func (s *ShellInput) View() string {
	before, at, after := s.ContentParts()

	// Phase 2: Simple rendering (no syntax highlighting yet)
	// Just ensure cursor is visible using reverse video
	return before + renderCursor(at) + after
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
