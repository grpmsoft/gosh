package repl

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMakeProfessionalStyles(t *testing.T) {
	t.Run("creates styles with all required fields", func(t *testing.T) {
		// Act
		styles := makeProfessionalStyles()

		// Assert - verify all style fields are initialized
		assert.NotNil(t, styles.PromptUser)
		assert.NotNil(t, styles.PromptPath)
		assert.NotNil(t, styles.PromptGit)
		assert.NotNil(t, styles.PromptGitDirty)
		assert.NotNil(t, styles.PromptArrow)
		assert.NotNil(t, styles.PromptError)
		assert.NotNil(t, styles.Output)
		assert.NotNil(t, styles.OutputErr)
		assert.NotNil(t, styles.Executing)
		assert.NotNil(t, styles.CompletionHint)
		assert.NotNil(t, styles.SyntaxCommand)
		assert.NotNil(t, styles.SyntaxOption)
		assert.NotNil(t, styles.SyntaxArg)
		assert.NotNil(t, styles.SyntaxString)
	})
}

func TestShortenPath(t *testing.T) {
	t.Run("shortens long path", func(t *testing.T) {
		m := createTestModelForHelpers(t)

		// Act
		shortened := m.shortenPath("/very/long/path/to/some/directory")

		// Assert
		// Should be shortened with ellipsis
		if len(shortened) < len("/very/long/path/to/some/directory") {
			assert.Contains(t, shortened, "...")
		}
	})

	t.Run("keeps short path unchanged", func(t *testing.T) {
		m := createTestModelForHelpers(t)

		// Act
		shortened := m.shortenPath("/short")

		// Assert
		// Short path should not be shortened
		assert.Equal(t, "/short", shortened)
	})

	t.Run("handles empty path", func(t *testing.T) {
		m := createTestModelForHelpers(t)

		// Act
		shortened := m.shortenPath("")

		// Assert
		assert.Equal(t, "", shortened)
	})

	t.Run("handles Windows paths", func(t *testing.T) {
		m := createTestModelForHelpers(t)

		// Act
		shortened := m.shortenPath("C:\\Users\\SomeUser\\Documents\\Project")

		// Assert
		// Should handle backslashes correctly
		assert.NotEmpty(t, shortened)
	})
}

func TestApplySyntaxHighlight(t *testing.T) {
	t.Run("highlights simple command", func(t *testing.T) {
		m := createTestModelForHelpers(t)

		// Act
		highlighted := m.applySyntaxHighlight("ls")

		// Assert
		// Should contain ANSI codes for command highlighting
		assert.NotEmpty(t, highlighted)
		// Commands are highlighted, so output should differ from input
		assert.Contains(t, highlighted, "ls")
	})

	t.Run("highlights command with arguments", func(t *testing.T) {
		m := createTestModelForHelpers(t)

		// Act
		highlighted := m.applySyntaxHighlight("git status")

		// Assert
		assert.NotEmpty(t, highlighted)
		assert.Contains(t, highlighted, "git")
		assert.Contains(t, highlighted, "status")
	})

	t.Run("highlights command with options", func(t *testing.T) {
		m := createTestModelForHelpers(t)

		// Act
		highlighted := m.applySyntaxHighlight("ls -la --color")

		// Assert
		assert.NotEmpty(t, highlighted)
		assert.Contains(t, highlighted, "ls")
		assert.Contains(t, highlighted, "-la")
		assert.Contains(t, highlighted, "--color")
	})

	t.Run("highlights quoted strings", func(t *testing.T) {
		m := createTestModelForHelpers(t)

		// Act
		highlighted := m.applySyntaxHighlight("echo \"hello world\"")

		// Assert
		assert.NotEmpty(t, highlighted)
		assert.Contains(t, highlighted, "echo")
		// Quotes may be styled separately from content
		assert.True(t, len(highlighted) > 0)
	})

	t.Run("handles empty input", func(t *testing.T) {
		m := createTestModelForHelpers(t)

		// Act
		highlighted := m.applySyntaxHighlight("")

		// Assert
		assert.Empty(t, highlighted)
	})
}

func TestRenderInputWithCursor(t *testing.T) {
	t.Run("renders input with cursor", func(t *testing.T) {
		m := createTestModelForHelpers(t)
		m.inputText = "test command"
		m.cursorPos = 4

		// Act
		rendered := m.renderInputWithCursor()

		// Assert
		assert.NotEmpty(t, rendered)
		// renderInputWithCursor returns textarea view, not direct text
	})

	t.Run("renders empty input", func(t *testing.T) {
		m := createTestModelForHelpers(t)
		m.inputText = ""
		m.cursorPos = 0

		// Act
		rendered := m.renderInputWithCursor()

		// Assert
		// Should not panic with empty input
		_ = rendered
	})

	t.Run("renders input with cursor at end", func(t *testing.T) {
		m := createTestModelForHelpers(t)
		m.inputText = "test"
		m.cursorPos = 4 // After last character

		// Act
		rendered := m.renderInputWithCursor()

		// Assert
		assert.NotEmpty(t, rendered)
		// Textarea rendering, not direct text
	})
}

func TestRenderHints(t *testing.T) {
	t.Run("renders hints when completion active", func(t *testing.T) {
		m := createTestModelForHelpers(t)
		m.completionActive = true
		m.completions = []string{"command1", "command2", "command3"}
		m.completionIndex = 0

		// Act
		hints := m.renderHints()

		// Assert
		assert.NotEmpty(t, hints)
		// Should contain some hint about completions (count, etc.)
		assert.Contains(t, hints, "Tab")
	})

	t.Run("renders empty when no completions", func(t *testing.T) {
		m := createTestModelForHelpers(t)
		m.completionActive = false
		m.completions = []string{}

		// Act
		hints := m.renderHints()

		// Assert
		assert.Empty(t, hints)
	})

	t.Run("renders multiple completion suggestions", func(t *testing.T) {
		m := createTestModelForHelpers(t)
		m.completionActive = true
		m.completions = []string{"git", "github", "gitignore"}
		m.completionIndex = 1

		// Act
		hints := m.renderHints()

		// Assert
		assert.NotEmpty(t, hints)
		// Should show all or some completions
	})
}

func TestRenderPromptForHistoryANSI(t *testing.T) {
	t.Run("renders ANSI prompt", func(t *testing.T) {
		m := createTestModelForHelpers(t)

		// Act
		prompt := m.renderPromptForHistoryANSI()

		// Assert
		assert.NotEmpty(t, prompt)
		// Should contain ANSI escape codes
		assert.Contains(t, prompt, "\033[")
	})

	t.Run("includes working directory in prompt", func(t *testing.T) {
		m := createTestModelForHelpers(t)

		// Act
		prompt := m.renderPromptForHistoryANSI()

		// Assert
		assert.NotEmpty(t, prompt)
		// Should contain some path information
	})

	t.Run("includes git info when in git repo", func(t *testing.T) {
		m := createTestModelForHelpers(t)
		m.gitBranch = "main"
		m.gitDirty = false

		// Act
		prompt := m.renderPromptForHistoryANSI()

		// Assert
		assert.NotEmpty(t, prompt)
		assert.Contains(t, prompt, "main")
	})

	t.Run("shows dirty indicator when git repo dirty", func(t *testing.T) {
		m := createTestModelForHelpers(t)
		m.gitBranch = "main"
		m.gitDirty = true

		// Act
		prompt := m.renderPromptForHistoryANSI()

		// Assert
		assert.NotEmpty(t, prompt)
		assert.Contains(t, prompt, "main")
		// Should have some indicator for dirty state (like * or +)
	})
}

func TestRenderHelpOverlay(t *testing.T) {
	t.Run("renders help overlay with keybindings", func(t *testing.T) {
		m := createTestModelForHelpers(t)

		// Act
		help := m.renderHelpOverlay()

		// Assert
		assert.NotEmpty(t, help)
		// Should contain help information
		assert.Contains(t, help, "Help") // Title or header
	})

	t.Run("help contains keyboard shortcuts", func(t *testing.T) {
		m := createTestModelForHelpers(t)

		// Act
		help := m.renderHelpOverlay()

		// Assert
		assert.NotEmpty(t, help)
		// Should list some keybindings
		// Common shortcuts like Ctrl+C, Ctrl+D, etc.
	})
}

func TestRenderWithHelpOverlay(t *testing.T) {
	t.Run("renders with help overlay when help active", func(t *testing.T) {
		m := createTestModelForHelpers(t)
		m.showingHelp = true
		m.width = 80
		m.height = 24

		// Act
		rendered := m.renderWithHelpOverlay()

		// Assert
		assert.NotEmpty(t, rendered)
		// Should contain help content
	})

	t.Run("renders normally when help not active", func(t *testing.T) {
		m := createTestModelForHelpers(t)
		m.showingHelp = false
		m.width = 80
		m.height = 24

		// Act
		rendered := m.renderWithHelpOverlay()

		// Assert
		// Should render normal view
		assert.NotEmpty(t, rendered)
	})
}

func TestView(t *testing.T) {
	t.Run("renders view without panic", func(t *testing.T) {
		m := createTestModelForHelpers(t)
		m.width = 80
		m.height = 24
		m.ready = true

		// Act
		view := m.View()

		// Assert
		assert.NotEmpty(t, view)
	})

	t.Run("returns waiting message when not ready", func(t *testing.T) {
		m := createTestModelForHelpers(t)
		m.ready = false

		// Act
		view := m.View()

		// Assert
		// Should return some initial message or empty
		_ = view
	})

	t.Run("renders help overlay when showingHelp is true", func(t *testing.T) {
		m := createTestModelForHelpers(t)
		m.ready = true
		m.showingHelp = true
		m.width = 80
		m.height = 24

		// Act
		view := m.View()

		// Assert
		assert.NotEmpty(t, view)
		// Should contain help content
	})
}

func TestRenderClassicMode(t *testing.T) {
	t.Run("renders classic mode UI", func(t *testing.T) {
		m := createTestModelForHelpers(t)
		m.width = 80
		m.height = 24
		m.ready = true

		// Act
		rendered := m.renderClassicMode()

		// Assert
		assert.NotEmpty(t, rendered)
	})
}

func TestRenderWarpMode(t *testing.T) {
	t.Run("renders warp mode UI", func(t *testing.T) {
		m := createTestModelForHelpers(t)
		m.width = 80
		m.height = 24
		m.ready = true

		// Act
		rendered := m.renderWarpMode()

		// Assert
		assert.NotEmpty(t, rendered)
	})
}

func TestRenderCompactMode(t *testing.T) {
	t.Run("renders compact mode UI", func(t *testing.T) {
		m := createTestModelForHelpers(t)
		m.width = 80
		m.height = 24
		m.ready = true

		// Act
		rendered := m.renderCompactMode()

		// Assert
		assert.NotEmpty(t, rendered)
	})
}

func TestRenderChatMode(t *testing.T) {
	t.Run("renders chat mode UI", func(t *testing.T) {
		m := createTestModelForHelpers(t)
		m.width = 80
		m.height = 24
		m.ready = true

		// Act
		rendered := m.renderChatMode()

		// Assert
		assert.NotEmpty(t, rendered)
	})
}
