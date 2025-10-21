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
// - Syntax highlighting with SMART CACHING (only when text changes!)
// - Emoji/Unicode correct rendering
// - Cursor blinking state (delegated to Phoenix TextInput)
//
// Performance Optimization:
// - Highlighting applied in Update() (when text changes)
// - View() uses cached result (fast!)
// - No highlighting on cursor movement (arrows) - text unchanged!
type ShellInput struct {
	base              input.Input          // Phoenix Input (value type, not pointer)
	history           *history.History
	historyNav        *history.Navigator
	cursorVisible     bool                 // Cursor blink state (controlled by parent Model tick)
	highlightCallback func(string) string  // Callback for syntax highlighting

	// Highlighting cache (PERFORMANCE CRITICAL!)
	lastPlainText     string // Last plain text we highlighted
	cachedHighlighted string // Cached highlighted version (with ANSI codes)
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
		lastPlainText:     "",    // Empty initially
		cachedHighlighted: "",    // No cached highlighting yet
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

	// Clear cache - new text needs re-highlighting
	// updateHighlightCache() will be called in Update() when text change is detected
	s.lastPlainText = ""
	s.cachedHighlighted = ""
}

// Reset clears input and moves cursor to start.
func (s *ShellInput) Reset() {
	s.base = s.base.SetContent("", 0)

	// CRITICAL: Clear highlighting cache!
	// Without this, View() continues to show old cached highlighted text
	s.lastPlainText = ""
	s.cachedHighlighted = ""
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
// Performance Optimization (CRITICAL!):
// - Check if text content changed after update
// - If changed: re-apply syntax highlighting and cache result
// - If unchanged (cursor movement): use cached highlighting
//
// Returns updated ShellInput and any commands to execute.
func (s *ShellInput) Update(msg api.Msg) (*ShellInput, api.Cmd) {
	// Save old text for comparison
	oldText := s.base.Value()

	switch msg := msg.(type) {
	case api.KeyMsg:
		switch msg.Type {
		case api.KeyUp:
			// History navigation (Up arrow - older commands)
			if cmd, ok := s.historyNav.Backward(); ok {
				s.SetValue(cmd)
				// Text changed - update highlight cache!
				s.updateHighlightCache()
				return s, nil
			}
			// If no history or already at oldest, ignore
			return s, nil

		case api.KeyDown:
			// History navigation (Down arrow - newer commands)
			if cmd, ok := s.historyNav.Forward(); ok {
				s.SetValue(cmd)
				// Text changed - update highlight cache!
				s.updateHighlightCache()
				return s, nil
			}
			// End of history - clear input (standard shell behavior)
			s.Reset()
			// Text changed - update highlight cache!
			s.updateHighlightCache()
			return s, nil
		}
	}

	// Delegate all other events to base TextInput
	var cmd api.Cmd
	s.base, cmd = s.base.Update(msg)

	// CRITICAL: Check if text content changed
	newText := s.base.Value()
	if newText != oldText {
		// Text changed - re-apply syntax highlighting!
		s.updateHighlightCache()
	}
	// If text unchanged (e.g., arrow keys) - cache still valid, no re-highlighting!

	return s, cmd
}

// updateHighlightCache applies syntax highlighting and caches the result.
// Only called when text content actually changes!
func (s *ShellInput) updateHighlightCache() {
	currentText := s.base.Value()

	// Apply syntax highlighting (if callback provided)
	if s.highlightCallback != nil {
		s.cachedHighlighted = s.highlightCallback(currentText)
	} else {
		s.cachedHighlighted = currentText
	}

	// Update cache key
	s.lastPlainText = currentText
}

// View renders the input with CACHED syntax highlighting.
//
// ═══════════════════════════════════════════════════════════════════════════
// PERFORMANCE ARCHITECTURE (Senior Developer Pattern!)
// ═══════════════════════════════════════════════════════════════════════════
//
// Problem:
//   - View() called frequently (every frame)
//   - Applying syntax highlighting here = SLOW (CPU-intensive)
//   - Result: Input lag, especially with arrow keys
//
// Solution (MVC Separation):
//   - Update() = Data changes → Apply highlighting, cache result
//   - View() = Rendering only → Use cached highlighted text
//
// Performance Benefits:
//   ✅ Highlighting ONLY when text changes (Insert, Delete, Backspace)
//   ✅ NO highlighting on cursor movement (Arrow keys) - cache reused!
//   ✅ Fast and responsive like native shells
//
// Algorithm:
//   1. Use cachedHighlighted (pre-computed in Update())
//   2. Get text after cursor for terminal cursor positioning
//   3. Render highlighted text + move cursor LEFT by len(after)
//   4. Terminal cursor now at correct position!
//
// Example: "echo hello" with cursor at position 5
//   - cachedHighlighted = "\033[1;33mecho\033[0m \033[32mhello\033[0m"
//   - after = "hello" (5 chars)
//   - Render highlighted + \033[5D (move left 5)
//   - Terminal cursor at position 5 ✓
//
// ═══════════════════════════════════════════════════════════════════════════
func (s *ShellInput) View() string {
	// Use CACHED highlighted text (computed in Update() only when text changed!)
	highlighted := s.cachedHighlighted

	// If cache empty (initial state), use plain text
	if highlighted == "" {
		highlighted = s.base.Value()
	}

	// Get cursor position in PLAIN text
	cursorPos := s.base.CursorPosition()

	// Calculate VISUAL length of highlighted text (ignoring ANSI codes!)
	// This is CRITICAL - terminal counts only visible chars, not ANSI bytes
	// PSReadLine does the same: LengthInBufferCells() skips ESC sequences
	visualLen := countVisibleChars(highlighted)

	// Move left from end of visual text to cursor position
	// cursorPos is grapheme offset in PLAIN text (without ANSI codes)
	// visualLen is visible character count in HIGHLIGHTED text (ANSI codes ignored)
	moveLeft := visualLen - cursorPos

	// Clamp to prevent moving left past beginning
	if moveLeft < 0 {
		moveLeft = 0
	}

	// Render highlighted text and position cursor
	// NOTE: Cursor is already shown and blinking (set in main.go with \033[?25h and \033[5 q)
	// We ONLY position it here, no hide/show (prevents cursor flickering!)
	if moveLeft > 0 {
		// Move cursor left to correct position
		// PSReadLine uses SetCursorPosition, we use ANSI \033[{n}D
		return fmt.Sprintf("%s\033[%dD", highlighted, moveLeft)
	}

	// Cursor already at end
	return highlighted
}

// Note: Syntax highlighting is now handled by Model.applySyntaxHighlight callback.
// The old complex character-by-character highlighting has been removed in favor of
// simple word-based highlighting that works correctly during typing.
//
// countVisibleChars() is defined in repl_render.go and counts visible characters,
// skipping ANSI escape sequences (same logic as PSReadLine's LengthInBufferCells()).
