package repl

import (
	"testing"

	"github.com/grpmsoft/gosh/internal/domain/config"

	"github.com/phoenix-tui/phoenix/tea/api"
	"github.com/stretchr/testify/assert"
)

func TestSwitchUIMode(t *testing.T) {
	t.Run("switches to Classic mode with Alt+1", func(t *testing.T) {
		m := createTestModelForHelpers(t)
		m.config.UI.AllowModeSwitching = true
		m.config.UI.Mode = config.UIModeWarp
		m.width = 80
		m.height = 24

		// Act
		m2, _ := m.switchUIMode("alt+1")

		// Assert
		assert.Equal(t, config.UIModeClassic, m2.config.UI.Mode)
	})

	t.Run("switches to Warp mode with Alt+2", func(t *testing.T) {
		m := createTestModelForHelpers(t)
		m.config.UI.AllowModeSwitching = true
		m.config.UI.Mode = config.UIModeClassic
		m.width = 80
		m.height = 24

		// Act
		m2, _ := m.switchUIMode("alt+2")

		// Assert
		assert.Equal(t, config.UIModeWarp, m2.config.UI.Mode)
	})

	t.Run("switches to Compact mode with Alt+3", func(t *testing.T) {
		m := createTestModelForHelpers(t)
		m.config.UI.AllowModeSwitching = true
		m.config.UI.Mode = config.UIModeWarp
		m.width = 80
		m.height = 24

		// Act
		m2, _ := m.switchUIMode("alt+3")

		// Assert
		assert.Equal(t, config.UIModeCompact, m2.config.UI.Mode)
	})

	t.Run("switches to Chat mode with Alt+4", func(t *testing.T) {
		m := createTestModelForHelpers(t)
		m.config.UI.AllowModeSwitching = true
		m.config.UI.Mode = config.UIModeWarp
		m.width = 80
		m.height = 24

		// Act
		m2, _ := m.switchUIMode("alt+4")

		// Assert
		assert.Equal(t, config.UIModeChat, m2.config.UI.Mode)
	})

	t.Run("does nothing when already in target mode", func(t *testing.T) {
		m := createTestModelForHelpers(t)
		m.config.UI.AllowModeSwitching = true
		m.config.UI.Mode = config.UIModeWarp
		m.width = 80
		m.height = 24

		// Act
		m2, _ := m.switchUIMode("alt+2") // Already in Warp

		// Assert
		assert.Equal(t, config.UIModeWarp, m2.config.UI.Mode)
	})

	t.Run("ignores unknown key", func(t *testing.T) {
		m := createTestModelForHelpers(t)
		m.config.UI.AllowModeSwitching = true
		initialMode := m.config.UI.Mode

		// Act
		m2, _ := m.switchUIMode("ctrl+f9")

		// Assert
		assert.Equal(t, initialMode, m2.config.UI.Mode)
	})
}

func TestHandleModeCommand(t *testing.T) {
	t.Run("shows current mode when no arguments", func(t *testing.T) {
		m := createTestModelForHelpers(t)
		m.config.UI.AllowModeSwitching = true
		m.config.UI.Mode = config.UIModeWarp

		// Act
		m2, _ := m.handleModeCommand(":mode")

		// Assert
		// Should have added output about current mode
		assert.Greater(t, len(m2.output), len(m.output))
	})

	t.Run("switches to classic mode", func(t *testing.T) {
		m := createTestModelForHelpers(t)
		m.config.UI.AllowModeSwitching = true
		m.config.UI.Mode = config.UIModeWarp
		m.width = 80
		m.height = 24

		// Act
		m2, _ := m.handleModeCommand(":mode classic")

		// Assert
		assert.Equal(t, config.UIModeClassic, m2.config.UI.Mode)
	})

	t.Run("switches to warp mode", func(t *testing.T) {
		m := createTestModelForHelpers(t)
		m.config.UI.AllowModeSwitching = true
		m.config.UI.Mode = config.UIModeClassic
		m.width = 80
		m.height = 24

		// Act
		m2, _ := m.handleModeCommand(":mode warp")

		// Assert
		assert.Equal(t, config.UIModeWarp, m2.config.UI.Mode)
	})

	t.Run("switches to compact mode", func(t *testing.T) {
		m := createTestModelForHelpers(t)
		m.config.UI.AllowModeSwitching = true
		m.config.UI.Mode = config.UIModeWarp
		m.width = 80
		m.height = 24

		// Act
		m2, _ := m.handleModeCommand(":mode compact")

		// Assert
		assert.Equal(t, config.UIModeCompact, m2.config.UI.Mode)
	})

	t.Run("switches to chat mode", func(t *testing.T) {
		m := createTestModelForHelpers(t)
		m.config.UI.AllowModeSwitching = true
		m.config.UI.Mode = config.UIModeWarp
		m.width = 80
		m.height = 24

		// Act
		m2, _ := m.handleModeCommand(":mode chat")

		// Assert
		assert.Equal(t, config.UIModeChat, m2.config.UI.Mode)
	})

	t.Run("handles unknown mode", func(t *testing.T) {
		m := createTestModelForHelpers(t)
		m.config.UI.AllowModeSwitching = true
		initialMode := m.config.UI.Mode

		// Act
		m2, _ := m.handleModeCommand(":mode invalid")

		// Assert
		assert.Equal(t, initialMode, m2.config.UI.Mode)
		// Should have added error message
		assert.Greater(t, len(m2.output), len(m.output))
	})

	t.Run("fails when mode switching disabled", func(t *testing.T) {
		m := createTestModelForHelpers(t)
		m.config.UI.AllowModeSwitching = false
		initialMode := m.config.UI.Mode

		// Act
		m2, _ := m.handleModeCommand(":mode classic")

		// Assert
		assert.Equal(t, initialMode, m2.config.UI.Mode)
		// Should have added error message
		assert.Greater(t, len(m2.output), len(m.output))
	})

	t.Run("notifies when already in target mode", func(t *testing.T) {
		m := createTestModelForHelpers(t)
		m.config.UI.AllowModeSwitching = true
		m.config.UI.Mode = config.UIModeWarp

		// Act
		m2, _ := m.handleModeCommand(":mode warp")

		// Assert
		assert.Equal(t, config.UIModeWarp, m2.config.UI.Mode)
		// Should have added notification
		assert.Greater(t, len(m2.output), len(m.output))
	})
}

func TestHandleTabCompletion(t *testing.T) {
	t.Run("generates completions on first Tab", func(t *testing.T) {
		m := createTestModelForHelpers(t)
		m.shellInput.SetValue("ec")

		// Act
		m2, _ := m.handleTabCompletion()

		// Assert
		assert.True(t, m2.completionActive)
		assert.Greater(t, len(m2.completions), 0)
		assert.Equal(t, 0, m2.completionIndex)
		// Should have completed to "echo"
		assert.Equal(t, "echo", m2.shellInput.Value())
	})

	t.Run("cycles through completions on repeated Tab", func(t *testing.T) {
		m := createTestModelForHelpers(t)
		m.shellInput.SetValue("e")
		m.completionActive = false

		// First Tab
		m2, _ := m.handleTabCompletion()

		// Second Tab
		if len(m2.completions) > 1 {
			m3, _ := m2.handleTabCompletion()
			assert.Equal(t, 1, m3.completionIndex)
		}
	})

	t.Run("returns empty completions for empty input", func(t *testing.T) {
		m := createTestModelForHelpers(t)
		m.shellInput.SetValue("")

		// Act
		m2, _ := m.handleTabCompletion()

		// Assert
		assert.False(t, m2.completionActive)
		assert.Equal(t, 0, len(m2.completions))
	})

	t.Run("returns empty completions for no matches", func(t *testing.T) {
		m := createTestModelForHelpers(t)
		m.shellInput.SetValue("zzzzz")

		// Act
		m2, _ := m.handleTabCompletion()

		// Assert
		assert.False(t, m2.completionActive)
		assert.Equal(t, 0, len(m2.completions))
	})
}

func TestGenerateCompletions(t *testing.T) {
	t.Run("completes builtin commands", func(t *testing.T) {
		m := createTestModelForHelpers(t)

		// Act
		completions := m.generateCompletions("ec")

		// Assert
		assert.Contains(t, completions, "echo")
	})

	t.Run("completes multiple matching builtins", func(t *testing.T) {
		m := createTestModelForHelpers(t)

		// Act
		completions := m.generateCompletions("e")

		// Assert
		// Should match: echo, exit, export, env
		assert.GreaterOrEqual(t, len(completions), 2)
		assert.Contains(t, completions, "echo")
		assert.Contains(t, completions, "exit")
	})

	t.Run("completes aliases", func(t *testing.T) {
		m := createTestModelForHelpers(t)
		m.currentSession.SetAlias("ll", "ls -la")

		// Act
		completions := m.generateCompletions("l")

		// Assert
		assert.Contains(t, completions, "ll")
	})

	t.Run("returns empty for empty input", func(t *testing.T) {
		m := createTestModelForHelpers(t)

		// Act
		completions := m.generateCompletions("")

		// Assert
		assert.Equal(t, 0, len(completions))
	})

	t.Run("returns empty for no matches", func(t *testing.T) {
		m := createTestModelForHelpers(t)

		// Act
		completions := m.generateCompletions("zzzzz")

		// Assert
		assert.Equal(t, 0, len(completions))
	})

	t.Run("completes file paths", func(t *testing.T) {
		m := createTestModelForHelpers(t)

		// Act - try to complete files in current directory
		// This will depend on actual files in test environment
		completions := m.generateCompletions("cd .")

		// Assert - should not panic
		_ = completions
	})
}

func TestUpdateWindowSize(t *testing.T) {
	t.Run("updates dimensions on WindowSizeMsg", func(t *testing.T) {
		m := createTestModelForHelpers(t)
		m.ready = false

		// Act
		msg := api.WindowSizeMsg{Width: 100, Height: 30}
		m2, _ := m.Update(msg)

		// Assert
		assert.Equal(t, 100, m2.width)
		assert.Equal(t, 30, m2.height)
		assert.True(t, m2.ready)
	})

	t.Run("adjusts viewport height for Classic mode", func(t *testing.T) {
		m := createTestModelForHelpers(t)
		m.config.UI.Mode = config.UIModeClassic

		// Act
		msg := api.WindowSizeMsg{Width: 80, Height: 24}
		m2, _ := m.Update(msg)

		// Assert
		// Classic mode uses full height
		assert.Equal(t, 24, m2.viewport.Height())
	})

	t.Run("adjusts viewport height for Compact mode", func(t *testing.T) {
		m := createTestModelForHelpers(t)
		m.config.UI.Mode = config.UIModeCompact

		// Act
		msg := api.WindowSizeMsg{Width: 80, Height: 24}
		m2, _ := m.Update(msg)

		// Assert
		// Compact mode reserves 1 line
		assert.Equal(t, 23, m2.viewport.Height())
	})

	t.Run("adjusts viewport height for Warp mode", func(t *testing.T) {
		m := createTestModelForHelpers(t)
		m.config.UI.Mode = config.UIModeWarp

		// Act
		msg := api.WindowSizeMsg{Width: 80, Height: 24}
		m2, _ := m.Update(msg)

		// Assert
		// Warp mode reserves 3 lines
		assert.Equal(t, 21, m2.viewport.Height())
	})
}

func TestHandleKeyPress(t *testing.T) {
	t.Run("quits on Ctrl+C", func(t *testing.T) {
		m := createTestModelForHelpers(t)

		// Act
		msg := api.KeyMsg{Type: api.KeyCtrlC} // Dedicated Ctrl+C type
		m2, cmd := m.handleKeyPress(msg)

		// Assert
		assert.True(t, m2.quitting)
		assert.NotNil(t, cmd)
	})

	t.Run("quits on Ctrl+D with empty input", func(t *testing.T) {
		m := createTestModelForHelpers(t)
		m.shellInput.SetValue("")

		// Act
		msg := api.KeyMsg{Type: api.KeyRune, Rune: 'd', Ctrl: true} // Ctrl+D
		m2, _ := m.handleKeyPress(msg)

		// Assert
		assert.True(t, m2.quitting)
	})

	t.Run("does not quit on Ctrl+D with input", func(t *testing.T) {
		m := createTestModelForHelpers(t)
		m.shellInput.SetValue("some text")

		// Act
		msg := api.KeyMsg{Type: api.KeyRune, Rune: 'd', Ctrl: true} // Ctrl+D
		m2, _ := m.handleKeyPress(msg)

		// Assert
		assert.False(t, m2.quitting)
	})

	t.Run("opens help on F1", func(t *testing.T) {
		m := createTestModelForHelpers(t)
		m.showingHelp = false

		// Act
		msg := api.KeyMsg{Type: api.KeyF1}
		m2, _ := m.handleKeyPress(msg)

		// Assert
		assert.True(t, m2.showingHelp)
	})

	t.Run("closes help on ESC", func(t *testing.T) {
		m := createTestModelForHelpers(t)
		m.showingHelp = true

		// Act
		msg := api.KeyMsg{Type: api.KeyEsc}
		m2, _ := m.handleKeyPress(msg)

		// Assert
		assert.False(t, m2.showingHelp)
	})

	t.Run("clears screen on Ctrl+L", func(t *testing.T) {
		m := createTestModelForHelpers(t)
		m.addOutputRaw("line 1")
		m.addOutputRaw("line 2")
		initialLen := len(m.output)

		// Act
		msg := api.KeyMsg{Type: api.KeyRune, Rune: 'l', Ctrl: true} // Ctrl+L
		m2, _ := m.handleKeyPress(msg)

		// Assert
		assert.Less(t, len(m2.output), initialLen)
		assert.True(t, m2.autoScroll)
	})

	t.Run("disables auto-scroll on PageUp", func(t *testing.T) {
		m := createTestModelForHelpers(t)
		m.autoScroll = true

		// Act
		msg := api.KeyMsg{Type: api.KeyPgUp}
		m2, _ := m.handleKeyPress(msg)

		// Assert
		assert.False(t, m2.autoScroll)
	})

	t.Run("enables auto-scroll on text input", func(t *testing.T) {
		m := createTestModelForHelpers(t)
		m.autoScroll = false

		// Act
		msg := api.KeyMsg{Type: api.KeyRune, Rune: 'a'}
		m2, _ := m.handleKeyPress(msg)

		// Assert
		assert.True(t, m2.autoScroll)
	})

	t.Run("resets completion on non-Tab key", func(t *testing.T) {
		m := createTestModelForHelpers(t)
		m.completionActive = true
		m.completions = []string{"echo", "exit"}
		m.completionIndex = 0

		// Act
		msg := api.KeyMsg{Type: api.KeyRune, Rune: 'a'}
		m2, _ := m.handleKeyPress(msg)

		// Assert
		assert.False(t, m2.completionActive)
		assert.Equal(t, 0, len(m2.completions))
		assert.Equal(t, -1, m2.completionIndex)
	})
}

func TestUpdateCommandExecutedMsg(t *testing.T) {
	t.Run("updates state on command completion", func(t *testing.T) {
		m := createTestModelForHelpers(t)
		m.executing = true

		// Act
		msg := commandExecutedMsg{
			output:   "test output",
			err:      nil,
			exitCode: 0,
		}
		m2, _ := m.Update(msg)

		// Assert
		assert.False(t, m2.executing)
		assert.Equal(t, 0, m2.lastExitCode)
	})

	t.Run("adds output to viewport in non-Classic mode", func(t *testing.T) {
		m := createTestModelForHelpers(t)
		m.config.UI.Mode = config.UIModeWarp
		m.executing = true
		initialLen := len(m.output)

		// Act
		msg := commandExecutedMsg{
			output:   "test output",
			err:      nil,
			exitCode: 0,
		}
		m2, _ := m.Update(msg)

		// Assert
		assert.Greater(t, len(m2.output), initialLen)
	})

	t.Run("updates exit code on error", func(t *testing.T) {
		m := createTestModelForHelpers(t)
		m.executing = true

		// Act
		msg := commandExecutedMsg{
			output:   "",
			err:      assert.AnError,
			exitCode: 1,
		}
		m2, _ := m.Update(msg)

		// Assert
		assert.False(t, m2.executing)
		assert.Equal(t, 1, m2.lastExitCode)
	})
}

func TestUpdateMouseMsg(t *testing.T) {
	t.Run("disables auto-scroll on mouse wheel", func(t *testing.T) {
		m := createTestModelForHelpers(t)
		m.autoScroll = true

		// Act
		msg := api.MouseMsg{Action: api.MouseActionPress, Button: api.MouseButtonWheelUp}
		m2, _ := m.Update(msg)

		// Assert
		assert.False(t, m2.autoScroll)
	})
}
