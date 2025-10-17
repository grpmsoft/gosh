package repl

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExpandAliases(t *testing.T) {
	t.Run("expands single alias", func(t *testing.T) {
		m := createTestModelForHelpers(t)
		m.currentSession.SetAlias("ll", "ls -la")

		// Act
		expanded, err := m.expandAliases("ll", 0)

		// Assert
		require.NoError(t, err)
		assert.Equal(t, "ls -la", expanded)
	})

	t.Run("expands alias with arguments", func(t *testing.T) {
		m := createTestModelForHelpers(t)
		m.currentSession.SetAlias("ll", "ls -la")

		// Act
		expanded, err := m.expandAliases("ll /tmp", 0)

		// Assert
		require.NoError(t, err)
		assert.Contains(t, expanded, "ls -la")
		assert.Contains(t, expanded, "/tmp")
	})

	t.Run("expands nested aliases with depth limit", func(t *testing.T) {
		m := createTestModelForHelpers(t)
		m.currentSession.SetAlias("a", "b")
		m.currentSession.SetAlias("b", "c")
		m.currentSession.SetAlias("c", "echo hello")

		// Act
		expanded, err := m.expandAliases("a", 0)

		// Assert
		require.NoError(t, err)
		// Should expand up to max depth (10)
		assert.Contains(t, expanded, "echo hello")
	})

	t.Run("detects circular alias references", func(t *testing.T) {
		m := createTestModelForHelpers(t)
		m.currentSession.SetAlias("a", "b")
		m.currentSession.SetAlias("b", "a")

		// Act
		_, err := m.expandAliases("a", 0)

		// Assert
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "depth")
	})

	t.Run("returns unchanged for non-alias command", func(t *testing.T) {
		m := createTestModelForHelpers(t)

		// Act
		expanded, err := m.expandAliases("echo hello", 0)

		// Assert
		require.NoError(t, err)
		assert.Equal(t, "echo hello", expanded)
	})

	t.Run("handles empty command", func(t *testing.T) {
		m := createTestModelForHelpers(t)

		// Act
		expanded, err := m.expandAliases("", 0)

		// Assert
		require.NoError(t, err)
		assert.Equal(t, "", expanded)
	})

	t.Run("prevents infinite recursion with depth exactly at limit", func(t *testing.T) {
		m := createTestModelForHelpers(t)
		// Create chain of 11 aliases (exceeds max depth of 10)
		m.currentSession.SetAlias("a1", "a2")
		m.currentSession.SetAlias("a2", "a3")
		m.currentSession.SetAlias("a3", "a4")
		m.currentSession.SetAlias("a4", "a5")
		m.currentSession.SetAlias("a5", "a6")
		m.currentSession.SetAlias("a6", "a7")
		m.currentSession.SetAlias("a7", "a8")
		m.currentSession.SetAlias("a8", "a9")
		m.currentSession.SetAlias("a9", "a10")
		m.currentSession.SetAlias("a10", "a11")
		m.currentSession.SetAlias("a11", "echo test")

		// Act - should fail due to depth limit
		_, err := m.expandAliases("a1", 0)

		// Assert
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "depth")
	})

	t.Run("expands alias with special characters in arguments", func(t *testing.T) {
		m := createTestModelForHelpers(t)
		m.currentSession.SetAlias("ll", "ls -la")

		// Act
		expanded, err := m.expandAliases("ll '/path/with spaces'", 0)

		// Assert
		require.NoError(t, err)
		assert.Contains(t, expanded, "ls -la")
		assert.Contains(t, expanded, "/path/with spaces")
	})
}

func TestExtractCommandName(t *testing.T) {
	t.Run("extracts command name from simple command", func(t *testing.T) {
		m := createTestModelForHelpers(t)

		// Act
		cmdName, args := m.extractCommandName("ls -la")

		// Assert
		assert.Equal(t, "ls", cmdName)
		assert.Equal(t, []string{"-la"}, args)
	})

	t.Run("extracts command name with multiple arguments", func(t *testing.T) {
		m := createTestModelForHelpers(t)

		// Act
		cmdName, args := m.extractCommandName("git commit -m \"test message\"")

		// Assert
		assert.Equal(t, "git", cmdName)
		assert.Greater(t, len(args), 0)
	})

	t.Run("handles command without arguments", func(t *testing.T) {
		m := createTestModelForHelpers(t)

		// Act
		cmdName, args := m.extractCommandName("pwd")

		// Assert
		assert.Equal(t, "pwd", cmdName)
		assert.Equal(t, 0, len(args))
	})

	t.Run("handles empty command", func(t *testing.T) {
		m := createTestModelForHelpers(t)

		// Act
		cmdName, args := m.extractCommandName("")

		// Assert
		assert.Equal(t, "", cmdName)
		assert.Equal(t, 0, len(args))
	})

	t.Run("handles command with pipe (extracts first command)", func(t *testing.T) {
		m := createTestModelForHelpers(t)

		// Act
		cmdName, args := m.extractCommandName("ls -la | grep test")

		// Assert
		// Should extract first command from pipeline
		assert.Equal(t, "ls", cmdName)
		assert.Contains(t, args, "-la")
	})

	t.Run("handles command with redirection", func(t *testing.T) {
		m := createTestModelForHelpers(t)

		// Act
		cmdName, args := m.extractCommandName("echo test > output.txt")

		// Assert
		assert.Equal(t, "echo", cmdName)
		// Should extract args without redirection operator
		assert.NotEmpty(t, args)
	})
}

func TestIsInteractiveCommand(t *testing.T) {
	t.Run("recognizes vim as interactive", func(t *testing.T) {
		m := createTestModelForHelpers(t)

		// Act
		isInteractive := m.isInteractiveCommand("vim")

		// Assert
		assert.True(t, isInteractive)
	})

	t.Run("recognizes nano as interactive", func(t *testing.T) {
		m := createTestModelForHelpers(t)

		// Act
		isInteractive := m.isInteractiveCommand("nano")

		// Assert
		assert.True(t, isInteractive)
	})

	t.Run("recognizes ssh as interactive", func(t *testing.T) {
		m := createTestModelForHelpers(t)

		// Act
		isInteractive := m.isInteractiveCommand("ssh")

		// Assert
		assert.True(t, isInteractive)
	})

	t.Run("recognizes less as interactive", func(t *testing.T) {
		m := createTestModelForHelpers(t)

		// Act
		isInteractive := m.isInteractiveCommand("less")

		// Assert
		assert.True(t, isInteractive)
	})

	t.Run("recognizes top as interactive", func(t *testing.T) {
		m := createTestModelForHelpers(t)

		// Act
		isInteractive := m.isInteractiveCommand("top")

		// Assert
		assert.True(t, isInteractive)
	})

	t.Run("recognizes htop as interactive", func(t *testing.T) {
		m := createTestModelForHelpers(t)

		// Act
		isInteractive := m.isInteractiveCommand("htop")

		// Assert
		assert.True(t, isInteractive)
	})

	t.Run("recognizes python REPL as interactive", func(t *testing.T) {
		m := createTestModelForHelpers(t)

		// Act
		isInteractive := m.isInteractiveCommand("python")

		// Assert
		assert.True(t, isInteractive, "Python REPL should be interactive")
	})

	t.Run("recognizes node REPL as interactive", func(t *testing.T) {
		m := createTestModelForHelpers(t)

		// Act
		isInteractive := m.isInteractiveCommand("node")

		// Assert
		assert.True(t, isInteractive, "Node.js REPL should be interactive")
	})

	t.Run("recognizes psql as interactive", func(t *testing.T) {
		m := createTestModelForHelpers(t)

		// Act
		isInteractive := m.isInteractiveCommand("psql")

		// Assert
		assert.True(t, isInteractive, "PostgreSQL client should be interactive")
	})

	t.Run("non-interactive command returns false", func(t *testing.T) {
		m := createTestModelForHelpers(t)

		// Act
		isInteractive := m.isInteractiveCommand("ls")

		// Assert
		assert.False(t, isInteractive)
	})

	t.Run("empty command returns false", func(t *testing.T) {
		m := createTestModelForHelpers(t)

		// Act
		isInteractive := m.isInteractiveCommand("")

		// Assert
		assert.False(t, isInteractive)
	})

	t.Run("recognizes shell scripts as interactive", func(t *testing.T) {
		m := createTestModelForHelpers(t)

		// Act
		isInteractive := m.isInteractiveCommand("./script.sh")

		// Assert
		assert.True(t, isInteractive, "Shell scripts may require interactive mode")
	})

	t.Run("recognizes batch scripts as interactive", func(t *testing.T) {
		m := createTestModelForHelpers(t)

		// Act
		isInteractive := m.isInteractiveCommand("./script.bat")

		// Assert
		assert.True(t, isInteractive, "Batch scripts may require interactive mode")
	})

	t.Run("recognizes PowerShell scripts as interactive", func(t *testing.T) {
		m := createTestModelForHelpers(t)

		// Act
		isInteractive := m.isInteractiveCommand("./script.ps1")

		// Assert
		assert.True(t, isInteractive, "PowerShell scripts may require interactive mode")
	})
}

func TestIsShellScript(t *testing.T) {
	t.Run("checks for script extension", func(t *testing.T) {
		m := createTestModelForHelpers(t)

		// Act - isShellScript checks file existence, not just extension
		// So for non-existent files, it will return false
		_, isScript := m.isShellScript("nonexistent.sh")

		// Assert - file doesn't exist, so false
		assert.False(t, isScript)
	})

	t.Run("non-script returns false", func(t *testing.T) {
		m := createTestModelForHelpers(t)

		// Act
		_, isScript := m.isShellScript("document.txt")

		// Assert
		assert.False(t, isScript)
	})

	t.Run("regular command returns false", func(t *testing.T) {
		m := createTestModelForHelpers(t)

		// Act
		_, isScript := m.isShellScript("ls")

		// Assert
		assert.False(t, isScript, "Plain command without path should not be detected as script")
	})

	t.Run("returns false for files without .sh/.bash extension", func(t *testing.T) {
		m := createTestModelForHelpers(t)

		// Act
		_, isScript := m.isShellScript("./file.txt")

		// Assert
		assert.False(t, isScript, "Text files should not be detected as shell scripts")
	})

	t.Run("returns false for absolute path without proper extension", func(t *testing.T) {
		m := createTestModelForHelpers(t)

		// Act
		_, isScript := m.isShellScript("/usr/bin/ls")

		// Assert
		assert.False(t, isScript, "Executables without .sh/.bash extension should not be detected as scripts")
	})
}

func TestPrepareCommand(t *testing.T) {
	t.Run("prepares simple command", func(t *testing.T) {
		m := createTestModelForHelpers(t)

		// Act
		cmdName, cmdArgs := m.prepareCommand("ls", []string{"-la"})

		// Assert
		assert.Equal(t, "ls", cmdName)
		assert.Equal(t, []string{"-la"}, cmdArgs)
	})

	t.Run("prepares command without arguments", func(t *testing.T) {
		m := createTestModelForHelpers(t)

		// Act
		cmdName, cmdArgs := m.prepareCommand("pwd", []string{})

		// Assert
		assert.Equal(t, "pwd", cmdName)
		assert.Equal(t, 0, len(cmdArgs))
	})

	t.Run("prepares command with multiple arguments", func(t *testing.T) {
		m := createTestModelForHelpers(t)

		// Act
		cmdName, cmdArgs := m.prepareCommand("git", []string{"commit", "-m", "test"})

		// Assert
		assert.Equal(t, "git", cmdName)
		assert.Equal(t, []string{"commit", "-m", "test"}, cmdArgs)
	})

	t.Run("returns command as-is for non-existent script", func(t *testing.T) {
		m := createTestModelForHelpers(t)

		// Act
		cmdName, cmdArgs := m.prepareCommand("./nonexistent.sh", []string{"arg1"})

		// Assert
		assert.Equal(t, "./nonexistent.sh", cmdName, "Non-existent scripts should be returned as-is")
		assert.Equal(t, []string{"arg1"}, cmdArgs)
	})

	t.Run("preserves arguments order", func(t *testing.T) {
		m := createTestModelForHelpers(t)

		// Act
		cmdName, cmdArgs := m.prepareCommand("git", []string{"commit", "-m", "message", "--amend"})

		// Assert
		assert.Equal(t, "git", cmdName)
		assert.Equal(t, []string{"commit", "-m", "message", "--amend"}, cmdArgs)
	})
}

func TestShowHelp(t *testing.T) {
	t.Run("shows help without panic", func(t *testing.T) {
		m := createTestModelForHelpers(t)

		// Act - should not panic
		m.showHelp()

		// Assert - check output was added
		assert.Greater(t, len(m.output), 0)
	})

	t.Run("help contains useful information", func(t *testing.T) {
		m := createTestModelForHelpers(t)
		initialLen := len(m.output)

		// Act
		m.showHelp()

		// Assert - should add multiple lines
		assert.Greater(t, len(m.output), initialLen)
	})

	t.Run("help contains builtin commands", func(t *testing.T) {
		m := createTestModelForHelpers(t)

		// Act
		m.showHelp()

		// Assert - combine all output lines
		allOutput := ""
		for _, line := range m.output {
			allOutput += line
		}

		// Check for key builtin commands
		assert.Contains(t, allOutput, "cd", "Help should mention cd command")
		assert.Contains(t, allOutput, "pwd", "Help should mention pwd command")
		assert.Contains(t, allOutput, "echo", "Help should mention echo command")
		assert.Contains(t, allOutput, "export", "Help should mention export command")
		assert.Contains(t, allOutput, "exit", "Help should mention exit command")
	})

	t.Run("help shows UI mode switching when enabled", func(t *testing.T) {
		m := createTestModelForHelpers(t)
		m.config.UI.AllowModeSwitching = true

		// Act
		m.showHelp()

		// Assert
		allOutput := ""
		for _, line := range m.output {
			allOutput += line
		}

		assert.Contains(t, allOutput, ":mode", "Help should mention mode switching command")
	})

	t.Run("help updates viewport content", func(t *testing.T) {
		m := createTestModelForHelpers(t)

		// Act
		m.showHelp()

		// Assert - viewport should be updated (content should change)
		// Note: We can't directly compare viewport content, but output should be longer
		assert.Greater(t, len(m.output), 0)
	})
}

func TestExecuteCommand(t *testing.T) {
	t.Run("handles empty command", func(t *testing.T) {
		m := createTestModelForHelpers(t)
		m.shellInput.SetValue("")

		// Act
		m2, _ := m.executeCommand()

		// Assert - should clear shellInput
		assert.Empty(t, m2.shellInput.Value())
	})

	t.Run("handles clear command", func(t *testing.T) {
		m := createTestModelForHelpers(t)
		m.addOutputRaw("line 1")
		m.addOutputRaw("line 2")
		m.shellInput.SetValue("clear")

		// Act
		m2, _ := m.executeCommand()

		// Assert
		assert.Empty(t, m2.shellInput.Value())
		// Output should be cleared
	})

	t.Run("handles help command", func(t *testing.T) {
		m := createTestModelForHelpers(t)
		m.shellInput.SetValue("help")
		initialLen := len(m.output)

		// Act
		m2, _ := m.executeCommand()

		// Assert
		assert.Empty(t, m2.shellInput.Value())
		assert.Greater(t, len(m2.output), initialLen)
	})

	t.Run("handles mode command", func(t *testing.T) {
		m := createTestModelForHelpers(t)
		m.config.UI.AllowModeSwitching = true
		m.shellInput.SetValue(":mode")

		// Act
		m2, _ := m.executeCommand()

		// Assert
		assert.Empty(t, m2.shellInput.Value())
	})

	// NOTE: Height test removed - Phoenix ShellInput doesn't have Height() method
	// Multiline is handled automatically via Alt+Enter

	t.Run("clears completion state on execute", func(t *testing.T) {
		m := createTestModelForHelpers(t)
		m.completionActive = true
		m.completions = []string{"test1", "test2"}
		m.completionIndex = 1
		m.beforeCompletion = "te"
		m.shellInput.SetValue("clear")

		// Act
		m2, _ := m.executeCommand()

		// Assert
		assert.False(t, m2.completionActive, "Completion should be cleared")
		assert.Empty(t, m2.completions, "Completions list should be empty")
		assert.Equal(t, -1, m2.completionIndex, "Completion index should be reset")
		assert.Empty(t, m2.beforeCompletion, "Before completion text should be cleared")
	})

	t.Run("syncs input state on execute", func(t *testing.T) {
		m := createTestModelForHelpers(t)
		m.inputText = "old text"
		m.cursorPos = 5
		m.shellInput.SetValue("clear")

		// Act
		m2, _ := m.executeCommand()

		// Assert
		assert.Empty(t, m2.inputText, "Input text should be cleared")
		assert.Equal(t, 0, m2.cursorPos, "Cursor position should be reset")
	})

	t.Run("handles cls command same as clear", func(t *testing.T) {
		m := createTestModelForHelpers(t)
		m.addOutputRaw("line 1")
		m.shellInput.SetValue("cls")

		// Act
		m2, _ := m.executeCommand()

		// Assert
		assert.Empty(t, m2.shellInput.Value())
		// Output should be cleared (same as clear command)
	})

	t.Run("handles quit command same as exit", func(t *testing.T) {
		m := createTestModelForHelpers(t)
		m.shellInput.SetValue("quit")

		// Act
		m2, cmd := m.executeCommand()

		// Assert
		assert.True(t, m2.quitting, "Should set quitting flag")
		assert.NotNil(t, cmd, "Should return quit command")
	})
}
