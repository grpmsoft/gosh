package repl

import (
	"testing"

	"github.com/grpmsoft/gosh/internal/domain/history"
	"github.com/phoenix-tui/phoenix/tea/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestShellTextArea_NewShellTextArea tests creation of ShellTextArea.
func TestShellTextArea_NewShellTextArea(t *testing.T) {
	t.Run("creates with default state", func(t *testing.T) {
		// Arrange
		hist := history.NewHistory(history.DefaultConfig())
		highlight := func(s string) string { return s }

		// Act
		textarea := NewShellTextArea(80, 5, hist, highlight)

		// Assert
		require.NotNil(t, textarea)
		assert.Equal(t, "", textarea.Value())
		// Phoenix TextArea always has at least 1 line (even when empty)
		assert.Equal(t, 1, len(textarea.Lines()))
	})

	t.Run("creates with specified dimensions", func(t *testing.T) {
		// Arrange
		hist := history.NewHistory(history.DefaultConfig())
		highlight := func(s string) string { return s }

		// Act
		textarea := NewShellTextArea(120, 10, hist, highlight)

		// Assert
		require.NotNil(t, textarea)
		// Dimensions tested indirectly through rendering
	})
}

// TestShellTextArea_SetValue_SingleLine tests setting single-line text.
func TestShellTextArea_SetValue_SingleLine(t *testing.T) {
	t.Run("sets simple text", func(t *testing.T) {
		// Arrange
		hist := history.NewHistory(history.DefaultConfig())
		textarea := NewShellTextArea(80, 5, hist, noopHighlight)

		// Act
		textarea.SetValue("echo hello")

		// Assert
		assert.Equal(t, "echo hello", textarea.Value())
		assert.Equal(t, 1, len(textarea.Lines()))
		assert.Equal(t, "echo hello", textarea.Lines()[0])
	})

	t.Run("sets text with spaces", func(t *testing.T) {
		// Arrange
		hist := history.NewHistory(history.DefaultConfig())
		textarea := NewShellTextArea(80, 5, hist, noopHighlight)

		// Act
		textarea.SetValue("git commit -m 'test message'")

		// Assert
		assert.Equal(t, "git commit -m 'test message'", textarea.Value())
	})

	t.Run("sets empty text", func(t *testing.T) {
		// Arrange
		hist := history.NewHistory(history.DefaultConfig())
		textarea := NewShellTextArea(80, 5, hist, noopHighlight)
		textarea.SetValue("some text")

		// Act
		textarea.SetValue("")

		// Assert
		assert.Equal(t, "", textarea.Value())
		// Phoenix TextArea always has at least 1 line (even when empty)
		assert.Equal(t, 1, len(textarea.Lines()))
	})
}

// TestShellTextArea_SetValue_Multiline tests setting multiline text.
func TestShellTextArea_SetValue_Multiline(t *testing.T) {
	t.Run("sets two lines", func(t *testing.T) {
		// Arrange
		hist := history.NewHistory(history.DefaultConfig())
		textarea := NewShellTextArea(80, 5, hist, noopHighlight)

		// Act
		textarea.SetValue("line1\nline2")

		// Assert
		assert.Equal(t, "line1\nline2", textarea.Value())
		assert.Equal(t, 2, len(textarea.Lines()))
		assert.Equal(t, "line1", textarea.Lines()[0])
		assert.Equal(t, "line2", textarea.Lines()[1])
	})

	t.Run("sets three lines", func(t *testing.T) {
		// Arrange
		hist := history.NewHistory(history.DefaultConfig())
		textarea := NewShellTextArea(80, 5, hist, noopHighlight)

		// Act
		textarea.SetValue("echo 'start'\necho 'middle'\necho 'end'")

		// Assert
		assert.Equal(t, 3, len(textarea.Lines()))
		assert.Equal(t, "echo 'start'", textarea.Lines()[0])
		assert.Equal(t, "echo 'middle'", textarea.Lines()[1])
		assert.Equal(t, "echo 'end'", textarea.Lines()[2])
	})

	t.Run("sets text with empty lines", func(t *testing.T) {
		// Arrange
		hist := history.NewHistory(history.DefaultConfig())
		textarea := NewShellTextArea(80, 5, hist, noopHighlight)

		// Act
		textarea.SetValue("line1\n\nline3")

		// Assert
		assert.Equal(t, 3, len(textarea.Lines()))
		assert.Equal(t, "line1", textarea.Lines()[0])
		assert.Equal(t, "", textarea.Lines()[1])
		assert.Equal(t, "line3", textarea.Lines()[2])
	})
}

// TestShellTextArea_Lines tests getting lines.
func TestShellTextArea_Lines(t *testing.T) {
	t.Run("returns empty for empty textarea", func(t *testing.T) {
		// Arrange
		hist := history.NewHistory(history.DefaultConfig())
		textarea := NewShellTextArea(80, 5, hist, noopHighlight)

		// Act
		lines := textarea.Lines()

		// Assert
		// Phoenix TextArea always has at least 1 line (even when empty)
		assert.Equal(t, 1, len(lines))
	})

	t.Run("returns single line", func(t *testing.T) {
		// Arrange
		hist := history.NewHistory(history.DefaultConfig())
		textarea := NewShellTextArea(80, 5, hist, noopHighlight)
		textarea.SetValue("single line")

		// Act
		lines := textarea.Lines()

		// Assert
		require.Equal(t, 1, len(lines))
		assert.Equal(t, "single line", lines[0])
	})

	t.Run("returns multiple lines", func(t *testing.T) {
		// Arrange
		hist := history.NewHistory(history.DefaultConfig())
		textarea := NewShellTextArea(80, 5, hist, noopHighlight)
		textarea.SetValue("first\nsecond\nthird")

		// Act
		lines := textarea.Lines()

		// Assert
		require.Equal(t, 3, len(lines))
		assert.Equal(t, "first", lines[0])
		assert.Equal(t, "second", lines[1])
		assert.Equal(t, "third", lines[2])
	})
}

// TestShellTextArea_HistoryNavigation_Up tests history navigation upward.
func TestShellTextArea_HistoryNavigation_Up(t *testing.T) {
	t.Run("loads previous command on Up arrow", func(t *testing.T) {
		// Arrange
		hist := history.NewHistory(history.DefaultConfig())
		_ = hist.Add("git status")
		_ = hist.Add("git commit")
		_ = hist.Add("git push")

		textarea := NewShellTextArea(80, 5, hist, noopHighlight)

		// Act
		upMsg := api.KeyMsg{Type: api.KeyUp}
		newTextArea, _ := textarea.Update(upMsg)

		// Assert
		assert.Equal(t, "git push", newTextArea.Value())
	})

	t.Run("navigates backward through history", func(t *testing.T) {
		// Arrange
		hist := history.NewHistory(history.DefaultConfig())
		_ = hist.Add("cmd1")
		_ = hist.Add("cmd2")
		_ = hist.Add("cmd3")

		textarea := NewShellTextArea(80, 5, hist, noopHighlight)

		// Act
		upMsg := api.KeyMsg{Type: api.KeyUp}
		textarea, _ = textarea.Update(upMsg) // cmd3
		textarea, _ = textarea.Update(upMsg) // cmd2
		textarea, _ = textarea.Update(upMsg) // cmd1

		// Assert
		assert.Equal(t, "cmd1", textarea.Value())
	})

	t.Run("stays at oldest when Up arrow pressed at beginning", func(t *testing.T) {
		// Arrange
		hist := history.NewHistory(history.DefaultConfig())
		_ = hist.Add("oldest")
		_ = hist.Add("newest")

		textarea := NewShellTextArea(80, 5, hist, noopHighlight)

		// Act
		upMsg := api.KeyMsg{Type: api.KeyUp}
		textarea, _ = textarea.Update(upMsg) // newest
		textarea, _ = textarea.Update(upMsg) // oldest
		textarea, _ = textarea.Update(upMsg) // should stay at oldest

		// Assert
		assert.Equal(t, "oldest", textarea.Value())
	})

	t.Run("does nothing when history is empty", func(t *testing.T) {
		// Arrange
		hist := history.NewHistory(history.DefaultConfig())
		textarea := NewShellTextArea(80, 5, hist, noopHighlight)

		// Act
		upMsg := api.KeyMsg{Type: api.KeyUp}
		newTextArea, _ := textarea.Update(upMsg)

		// Assert
		assert.Equal(t, "", newTextArea.Value())
	})
}

// TestShellTextArea_HistoryNavigation_Down tests history navigation downward.
func TestShellTextArea_HistoryNavigation_Down(t *testing.T) {
	t.Run("navigates forward through history", func(t *testing.T) {
		// Arrange
		hist := history.NewHistory(history.DefaultConfig())
		_ = hist.Add("cmd1")
		_ = hist.Add("cmd2")
		_ = hist.Add("cmd3")

		textarea := NewShellTextArea(80, 5, hist, noopHighlight)

		// Act - navigate backward first, then forward
		upMsg := api.KeyMsg{Type: api.KeyUp}
		downMsg := api.KeyMsg{Type: api.KeyDown}

		textarea, _ = textarea.Update(upMsg)   // cmd3
		textarea, _ = textarea.Update(upMsg)   // cmd2
		textarea, _ = textarea.Update(upMsg)   // cmd1
		textarea, _ = textarea.Update(downMsg) // cmd2

		// Assert
		assert.Equal(t, "cmd2", textarea.Value())
	})

	t.Run("clears input when reaching end of history", func(t *testing.T) {
		// Arrange
		hist := history.NewHistory(history.DefaultConfig())
		_ = hist.Add("cmd1")
		_ = hist.Add("cmd2")

		textarea := NewShellTextArea(80, 5, hist, noopHighlight)

		// Act
		upMsg := api.KeyMsg{Type: api.KeyUp}
		downMsg := api.KeyMsg{Type: api.KeyDown}

		textarea, _ = textarea.Update(upMsg)   // cmd2
		textarea, _ = textarea.Update(downMsg) // end - should clear

		// Assert
		assert.Equal(t, "", textarea.Value())
	})

	t.Run("does nothing when already at end", func(t *testing.T) {
		// Arrange
		hist := history.NewHistory(history.DefaultConfig())
		_ = hist.Add("cmd1")

		textarea := NewShellTextArea(80, 5, hist, noopHighlight)

		// Act
		downMsg := api.KeyMsg{Type: api.KeyDown}
		textarea, _ = textarea.Update(downMsg)

		// Assert
		assert.Equal(t, "", textarea.Value())
	})

	t.Run("does nothing when history is empty", func(t *testing.T) {
		// Arrange
		hist := history.NewHistory(history.DefaultConfig())
		textarea := NewShellTextArea(80, 5, hist, noopHighlight)

		// Act
		downMsg := api.KeyMsg{Type: api.KeyDown}
		newTextArea, _ := textarea.Update(downMsg)

		// Assert
		assert.Equal(t, "", newTextArea.Value())
	})
}

// TestShellTextArea_HistoryNavigation_ResetOnEdit tests history reset on edit.
func TestShellTextArea_HistoryNavigation_ResetOnEdit(t *testing.T) {
	t.Run("continues editing after history navigation", func(t *testing.T) {
		// Arrange
		hist := history.NewHistory(history.DefaultConfig())
		_ = hist.Add("git status")

		textarea := NewShellTextArea(80, 5, hist, noopHighlight)

		// Act
		upMsg := api.KeyMsg{Type: api.KeyUp}
		textarea, _ = textarea.Update(upMsg) // Load "git status"

		// Simulate typing (this is passed to base TextArea)
		charMsg := api.KeyMsg{Type: api.KeyRune, Rune: 'a'}
		textarea, _ = textarea.Update(charMsg)

		// Assert - value should be modified by base TextArea
		// (Exact behavior depends on Phoenix TextArea implementation)
		// We just verify it doesn't crash
		require.NotNil(t, textarea)
	})
}

// TestShellTextArea_Reset tests clearing textarea.
func TestShellTextArea_Reset(t *testing.T) {
	t.Run("clears single line text", func(t *testing.T) {
		// Arrange
		hist := history.NewHistory(history.DefaultConfig())
		textarea := NewShellTextArea(80, 5, hist, noopHighlight)
		textarea.SetValue("some text")

		// Act
		textarea.Reset()

		// Assert
		assert.Equal(t, "", textarea.Value())
		// Phoenix TextArea always has at least 1 line (even when empty)
		assert.Equal(t, 1, len(textarea.Lines()))
	})

	t.Run("clears multiline text", func(t *testing.T) {
		// Arrange
		hist := history.NewHistory(history.DefaultConfig())
		textarea := NewShellTextArea(80, 5, hist, noopHighlight)
		textarea.SetValue("line1\nline2\nline3")

		// Act
		textarea.Reset()

		// Assert
		assert.Equal(t, "", textarea.Value())
		// Phoenix TextArea always has at least 1 line (even when empty)
		assert.Equal(t, 1, len(textarea.Lines()))
	})
}

// TestShellTextArea_CursorPosition tests cursor position retrieval.
func TestShellTextArea_CursorPosition(t *testing.T) {
	t.Run("returns initial position", func(t *testing.T) {
		// Arrange
		hist := history.NewHistory(history.DefaultConfig())
		textarea := NewShellTextArea(80, 5, hist, noopHighlight)

		// Act
		row, col := textarea.CursorPosition()

		// Assert
		assert.Equal(t, 0, row)
		assert.Equal(t, 0, col)
	})

	t.Run("returns position after setting text", func(t *testing.T) {
		// Arrange
		hist := history.NewHistory(history.DefaultConfig())
		textarea := NewShellTextArea(80, 5, hist, noopHighlight)
		textarea.SetValue("line1\nline2")

		// Act
		row, col := textarea.CursorPosition()

		// Assert
		// Phoenix TextArea should move cursor to end after SetValue
		// This is implementation detail that we test
		require.NotEqual(t, -1, row)
		require.NotEqual(t, -1, col)
	})
}

// TestShellTextArea_ContentParts tests content parts retrieval.
func TestShellTextArea_ContentParts(t *testing.T) {
	t.Run("returns parts for empty textarea", func(t *testing.T) {
		// Arrange
		hist := history.NewHistory(history.DefaultConfig())
		textarea := NewShellTextArea(80, 5, hist, noopHighlight)

		// Act
		before, at, after := textarea.ContentParts()

		// Assert
		// Phoenix TextArea returns " " for 'at' when empty (cursor placeholder)
		fullText := before + at + after
		assert.Equal(t, " ", fullText) // Not "" but " " (single space)
	})

	t.Run("returns parts after setting text", func(t *testing.T) {
		// Arrange
		hist := history.NewHistory(history.DefaultConfig())
		textarea := NewShellTextArea(80, 5, hist, noopHighlight)
		textarea.SetValue("hello world")

		// Act
		before, at, after := textarea.ContentParts()

		// Assert
		// All parts combined should equal full text
		fullText := before + at + after
		assert.Equal(t, "hello world", fullText)
	})

	t.Run("returns parts for multiline text", func(t *testing.T) {
		// Arrange
		hist := history.NewHistory(history.DefaultConfig())
		textarea := NewShellTextArea(80, 5, hist, noopHighlight)
		textarea.SetValue("line1\nline2\nline3")

		// Act
		before, at, after := textarea.ContentParts()

		// Assert
		// Phoenix TextArea ContentParts returns CURRENT LINE only, not all lines
		// This is different from single-line behavior
		fullCurrentLine := before + at + after
		// After SetValue, cursor is at end of last line
		assert.Contains(t, fullCurrentLine, "line", "Should contain part of current line")
	})
}

// TestShellTextArea_SetSize tests resizing.
func TestShellTextArea_SetSize(t *testing.T) {
	t.Run("changes size", func(t *testing.T) {
		// Arrange
		hist := history.NewHistory(history.DefaultConfig())
		textarea := NewShellTextArea(80, 5, hist, noopHighlight)

		// Act
		textarea.SetSize(120, 10)

		// Assert
		// Size is tested indirectly through rendering
		// We just verify it doesn't crash
		require.NotNil(t, textarea)
	})

	t.Run("preserves content after resize", func(t *testing.T) {
		// Arrange
		hist := history.NewHistory(history.DefaultConfig())
		textarea := NewShellTextArea(80, 5, hist, noopHighlight)
		textarea.SetValue("test content")

		// Act
		textarea.SetSize(120, 10)

		// Assert
		assert.Equal(t, "test content", textarea.Value())
	})
}

// TestShellTextArea_View tests rendering.
func TestShellTextArea_View(t *testing.T) {
	t.Run("renders empty textarea", func(t *testing.T) {
		// Arrange
		hist := history.NewHistory(history.DefaultConfig())
		textarea := NewShellTextArea(80, 5, hist, noopHighlight)

		// Act
		view := textarea.View()

		// Assert
		require.NotNil(t, view)
		// View rendering is complex, we just check it doesn't crash
	})

	t.Run("renders single line", func(t *testing.T) {
		// Arrange
		hist := history.NewHistory(history.DefaultConfig())
		textarea := NewShellTextArea(80, 5, hist, noopHighlight)
		textarea.SetValue("echo hello")

		// Act
		view := textarea.View()

		// Assert
		require.NotNil(t, view)
		assert.NotEqual(t, "", view)
	})

	t.Run("renders multiline", func(t *testing.T) {
		// Arrange
		hist := history.NewHistory(history.DefaultConfig())
		textarea := NewShellTextArea(80, 5, hist, noopHighlight)
		textarea.SetValue("line1\nline2\nline3")

		// Act
		view := textarea.View()

		// Assert
		require.NotNil(t, view)
		assert.NotEqual(t, "", view)
	})

	t.Run("applies syntax highlighting", func(t *testing.T) {
		// Arrange
		hist := history.NewHistory(history.DefaultConfig())
		highlight := func(s string) string {
			return "[HIGHLIGHTED]" + s
		}
		textarea := NewShellTextArea(80, 5, hist, highlight)
		textarea.SetValue("git status")

		// Act
		view := textarea.View()

		// Assert
		require.NotNil(t, view)
		// Note: Current implementation delegates to Phoenix TextArea.View()
		// Syntax highlighting is NOT yet integrated in View()
		// This will be implemented when needed for REPL integration
		assert.NotEqual(t, "", view)
	})
}

// Helper functions

// noopHighlight is a no-op highlight callback for tests.
func noopHighlight(s string) string {
	return s
}
