package history_test

import (
	"testing"

	"github.com/grpmsoft/gosh/internal/domain/history"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestHistory_NewHistory tests creating a new history with settings
func TestHistory_NewHistory(t *testing.T) {
	t.Run("creates history with default config", func(t *testing.T) {
		h := history.NewHistory(history.DefaultConfig())

		assert.NotNil(t, h)
		assert.Equal(t, 0, h.Size())
		assert.True(t, h.IsEmpty()) // Fixed: empty history should return true
	})

	t.Run("creates history with custom max size", func(t *testing.T) {
		cfg := history.Config{
			MaxSize:          100,
			SaveToFile:       true,
			DeduplicateAdded: true,
		}
		h := history.NewHistory(cfg)

		assert.NotNil(t, h)
		assert.Equal(t, 100, h.MaxSize())
	})
}

// TestHistory_Add tests adding commands to history
func TestHistory_Add(t *testing.T) {
	t.Run("adds single command", func(t *testing.T) {
		h := history.NewHistory(history.DefaultConfig())

		err := h.Add("ls -la")
		require.NoError(t, err)

		assert.Equal(t, 1, h.Size())
	})

	t.Run("adds multiple commands in order", func(t *testing.T) {
		h := history.NewHistory(history.DefaultConfig())

		h.Add("git status")
		h.Add("git add .")
		h.Add("git commit")

		assert.Equal(t, 3, h.Size())

		// Last command should be at the top (reverse order)
		recent := h.GetRecent(3)
		assert.Equal(t, "git commit", recent[0])
		assert.Equal(t, "git add .", recent[1])
		assert.Equal(t, "git status", recent[2])
	})

	t.Run("rejects empty commands", func(t *testing.T) {
		h := history.NewHistory(history.DefaultConfig())

		err := h.Add("")
		assert.Error(t, err)
		assert.Equal(t, 0, h.Size())

		err = h.Add("   ")
		assert.Error(t, err)
		assert.Equal(t, 0, h.Size())
	})

	t.Run("trims whitespace from commands", func(t *testing.T) {
		h := history.NewHistory(history.DefaultConfig())

		h.Add("  ls -la  ")
		h.Add("\tgit status\n")

		recent := h.GetRecent(2)
		assert.Equal(t, "git status", recent[0])
		assert.Equal(t, "ls -la", recent[1])
	})

	t.Run("deduplicates consecutive identical commands", func(t *testing.T) {
		cfg := history.Config{
			MaxSize:          1000,
			DeduplicateAdded: true,
		}
		h := history.NewHistory(cfg)

		h.Add("ls")
		h.Add("ls")
		h.Add("ls")
		h.Add("pwd")
		h.Add("pwd")

		// Only consecutive unique commands
		assert.Equal(t, 2, h.Size())
		recent := h.GetRecent(2)
		assert.Equal(t, "pwd", recent[0])
		assert.Equal(t, "ls", recent[1])
	})

	t.Run("allows duplicate if not consecutive when dedup enabled", func(t *testing.T) {
		cfg := history.Config{
			MaxSize:          1000,
			DeduplicateAdded: true,
		}
		h := history.NewHistory(cfg)

		h.Add("ls")
		h.Add("pwd")
		h.Add("ls") // Allowed (not consecutive)

		assert.Equal(t, 3, h.Size())
	})

	t.Run("respects max size limit", func(t *testing.T) {
		cfg := history.Config{
			MaxSize:          3,
			DeduplicateAdded: false,
		}
		h := history.NewHistory(cfg)

		h.Add("cmd1")
		h.Add("cmd2")
		h.Add("cmd3")
		h.Add("cmd4") // Will evict cmd1

		assert.Equal(t, 3, h.Size())

		// Old command removed
		recent := h.GetRecent(3)
		assert.Equal(t, "cmd4", recent[0])
		assert.Equal(t, "cmd3", recent[1])
		assert.Equal(t, "cmd2", recent[2])
	})
}

// TestHistory_Search tests searching commands in history (Ctrl+R functionality)
func TestHistory_Search(t *testing.T) {
	t.Run("finds exact match", func(t *testing.T) {
		h := history.NewHistory(history.DefaultConfig())
		h.Add("git status")
		h.Add("git commit")
		h.Add("ls -la")

		results := h.Search("git status")
		require.Len(t, results, 1)
		assert.Equal(t, "git status", results[0])
	})

	t.Run("finds partial match (substring)", func(t *testing.T) {
		h := history.NewHistory(history.DefaultConfig())
		h.Add("git status")
		h.Add("git commit -m 'fix'")
		h.Add("git push origin main")
		h.Add("ls -la")

		results := h.Search("git")
		assert.Len(t, results, 3)

		// Order: from newest to oldest
		assert.Equal(t, "git push origin main", results[0])
		assert.Equal(t, "git commit -m 'fix'", results[1])
		assert.Equal(t, "git status", results[2])
	})

	t.Run("search is case insensitive", func(t *testing.T) {
		h := history.NewHistory(history.DefaultConfig())
		h.Add("Git Status")
		h.Add("GIT COMMIT")

		results := h.Search("git")
		assert.Len(t, results, 2)
	})

	t.Run("returns empty for no matches", func(t *testing.T) {
		h := history.NewHistory(history.DefaultConfig())
		h.Add("ls -la")
		h.Add("pwd")

		results := h.Search("git")
		assert.Empty(t, results)
	})

	t.Run("returns empty for empty search query", func(t *testing.T) {
		h := history.NewHistory(history.DefaultConfig())
		h.Add("ls -la")

		results := h.Search("")
		assert.Empty(t, results)
	})

	t.Run("limits search results", func(t *testing.T) {
		h := history.NewHistory(history.DefaultConfig())
		for i := 1; i <= 100; i++ {
			h.Add("git status")
		}

		results := h.Search("git")
		// UI limit: no more than 50 results
		assert.LessOrEqual(t, len(results), 50)
	})
}

// TestHistory_GetRecent tests retrieving recent commands
func TestHistory_GetRecent(t *testing.T) {
	t.Run("returns recent commands in reverse order", func(t *testing.T) {
		h := history.NewHistory(history.DefaultConfig())
		h.Add("cmd1")
		h.Add("cmd2")
		h.Add("cmd3")

		recent := h.GetRecent(2)
		require.Len(t, recent, 2)
		assert.Equal(t, "cmd3", recent[0])
		assert.Equal(t, "cmd2", recent[1])
	})

	t.Run("returns all if requested more than size", func(t *testing.T) {
		h := history.NewHistory(history.DefaultConfig())
		h.Add("cmd1")
		h.Add("cmd2")

		recent := h.GetRecent(10)
		assert.Len(t, recent, 2)
	})

	t.Run("returns empty for zero or negative", func(t *testing.T) {
		h := history.NewHistory(history.DefaultConfig())
		h.Add("cmd1")

		assert.Empty(t, h.GetRecent(0))
		assert.Empty(t, h.GetRecent(-5))
	})
}

// TestHistory_Clear tests clearing history
func TestHistory_Clear(t *testing.T) {
	t.Run("clears all history", func(t *testing.T) {
		h := history.NewHistory(history.DefaultConfig())
		h.Add("cmd1")
		h.Add("cmd2")
		h.Add("cmd3")

		h.Clear()

		assert.Equal(t, 0, h.Size())
		assert.Empty(t, h.GetRecent(10))
		assert.Empty(t, h.Search("cmd"))
	})

	t.Run("can add after clear", func(t *testing.T) {
		h := history.NewHistory(history.DefaultConfig())
		h.Add("cmd1")
		h.Clear()
		h.Add("cmd2")

		assert.Equal(t, 1, h.Size())
		recent := h.GetRecent(1)
		assert.Equal(t, "cmd2", recent[0])
	})
}

// TestHistory_Navigation tests history navigation (Up/Down arrows)
func TestHistory_Navigation(t *testing.T) {
	t.Run("navigates backward through history", func(t *testing.T) {
		h := history.NewHistory(history.DefaultConfig())
		h.Add("cmd1")
		h.Add("cmd2")
		h.Add("cmd3")

		nav := h.NewNavigator()

		// Initial state: empty
		assert.Equal(t, "", nav.Current())

		// Backward (Up)
		cmd, ok := nav.Backward()
		assert.True(t, ok)
		assert.Equal(t, "cmd3", cmd)

		cmd, ok = nav.Backward()
		assert.True(t, ok)
		assert.Equal(t, "cmd2", cmd)

		cmd, ok = nav.Backward()
		assert.True(t, ok)
		assert.Equal(t, "cmd1", cmd)

		// Reached the beginning
		cmd, ok = nav.Backward()
		assert.False(t, ok)
		assert.Equal(t, "cmd1", cmd) // Stay at the first
	})

	t.Run("navigates forward through history", func(t *testing.T) {
		h := history.NewHistory(history.DefaultConfig())
		h.Add("cmd1")
		h.Add("cmd2")
		h.Add("cmd3")

		nav := h.NewNavigator()

		// Go to the beginning
		nav.Backward()
		nav.Backward()
		nav.Backward()

		// Now forward (Down)
		cmd, ok := nav.Forward()
		assert.True(t, ok)
		assert.Equal(t, "cmd2", cmd)

		cmd, ok = nav.Forward()
		assert.True(t, ok)
		assert.Equal(t, "cmd3", cmd)

		// Reached the end - return empty string
		cmd, ok = nav.Forward()
		assert.True(t, ok)
		assert.Equal(t, "", cmd)

		// Nowhere further to go
		cmd, ok = nav.Forward()
		assert.False(t, ok)
		assert.Equal(t, "", cmd)
	})

	t.Run("resets navigation on new command", func(t *testing.T) {
		h := history.NewHistory(history.DefaultConfig())
		h.Add("cmd1")
		h.Add("cmd2")

		nav := h.NewNavigator()
		nav.Backward()
		nav.Backward()

		// Add new command
		h.Add("cmd3")

		// Navigator should be reset
		nav = h.NewNavigator()
		cmd, ok := nav.Backward()
		assert.True(t, ok)
		assert.Equal(t, "cmd3", cmd)
	})
}

// TestHistory_ToSlice tests exporting history as a slice
func TestHistory_ToSlice(t *testing.T) {
	t.Run("exports history as slice in chronological order", func(t *testing.T) {
		h := history.NewHistory(history.DefaultConfig())
		h.Add("cmd1")
		h.Add("cmd2")
		h.Add("cmd3")

		slice := h.ToSlice()
		require.Len(t, slice, 3)

		// Chronological order (old to new)
		assert.Equal(t, "cmd1", slice[0])
		assert.Equal(t, "cmd2", slice[1])
		assert.Equal(t, "cmd3", slice[2])
	})

	t.Run("returns empty slice for empty history", func(t *testing.T) {
		h := history.NewHistory(history.DefaultConfig())
		assert.Empty(t, h.ToSlice())
	})
}

// TestHistory_FromSlice tests loading history from a slice
func TestHistory_FromSlice(t *testing.T) {
	t.Run("loads history from slice", func(t *testing.T) {
		h := history.NewHistory(history.DefaultConfig())

		lines := []string{"cmd1", "cmd2", "cmd3"}
		err := h.FromSlice(lines)
		require.NoError(t, err)

		assert.Equal(t, 3, h.Size())
		recent := h.GetRecent(3)
		assert.Equal(t, "cmd3", recent[0])
		assert.Equal(t, "cmd2", recent[1])
		assert.Equal(t, "cmd1", recent[2])
	})

	t.Run("skips empty lines", func(t *testing.T) {
		h := history.NewHistory(history.DefaultConfig())

		lines := []string{"cmd1", "", "cmd2", "   ", "cmd3"}
		err := h.FromSlice(lines)
		require.NoError(t, err)

		assert.Equal(t, 3, h.Size())
	})

	t.Run("respects max size when loading", func(t *testing.T) {
		cfg := history.Config{
			MaxSize:          2,
			DeduplicateAdded: false,
		}
		h := history.NewHistory(cfg)

		lines := []string{"cmd1", "cmd2", "cmd3", "cmd4"}
		err := h.FromSlice(lines)
		require.NoError(t, err)

		// Only last 2
		assert.Equal(t, 2, h.Size())
		recent := h.GetRecent(2)
		assert.Equal(t, "cmd4", recent[0])
		assert.Equal(t, "cmd3", recent[1])
	})

	t.Run("replaces existing history", func(t *testing.T) {
		h := history.NewHistory(history.DefaultConfig())
		h.Add("old1")
		h.Add("old2")

		lines := []string{"new1", "new2"}
		err := h.FromSlice(lines)
		require.NoError(t, err)

		assert.Equal(t, 2, h.Size())
		recent := h.GetRecent(2)
		assert.Equal(t, "new2", recent[0])
		assert.Equal(t, "new1", recent[1])
	})
}
