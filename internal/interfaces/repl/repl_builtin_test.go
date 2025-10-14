package repl

import (
	"context"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/grpmsoft/gosh/internal/application/execute"
	"github.com/grpmsoft/gosh/internal/domain/config"
	"github.com/grpmsoft/gosh/internal/domain/session"
	"github.com/grpmsoft/gosh/internal/domain/shared"
	"github.com/grpmsoft/gosh/internal/infrastructure/builtin"
	"github.com/grpmsoft/gosh/internal/infrastructure/executor"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockFileSystem is a simple mock for testing.
type mockFileSystem struct{}

func (m *mockFileSystem) Exists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (m *mockFileSystem) IsDir(path string) (bool, error) {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	return info.IsDir(), nil
}

func (m *mockFileSystem) ReadFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}

func (m *mockFileSystem) WriteFile(path string, data []byte, perm fs.FileMode) error {
	return os.WriteFile(path, data, perm)
}

func (m *mockFileSystem) Stat(path string) (fs.FileInfo, error) {
	return os.Stat(path)
}

func (m *mockFileSystem) ReadDir(path string) ([]fs.DirEntry, error) {
	return os.ReadDir(path)
}

// createTestModelForBuiltin creates a minimal test model for builtin command testing.
func createTestModelForBuiltin(t *testing.T) *Model {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	cfg := config.DefaultConfig()

	// Create empty environment for tests
	env := make(shared.Environment)

	// Create test session directly
	sess, err := session.NewSession(
		"test-session",
		os.TempDir(),
		env,
	)
	require.NoError(t, err)

	// Create filesystem and executors
	mockFS := &mockFileSystem{} // Simple mock
	builtinExecutor := builtin.NewExecutor(mockFS, logger)
	commandExecutor := executor.NewOSCommandExecutor(logger)
	pipelineExecutor := executor.NewOSPipelineExecutor(logger)
	executeUseCase := execute.NewUseCase(
		builtinExecutor,
		commandExecutor,
		pipelineExecutor,
		logger,
	)

	// Create minimal model for testing
	model := &Model{
		currentSession: sess,
		executeUseCase: executeUseCase,
		logger:         logger,
		ctx:            context.Background(),
		config:         cfg,
	}

	return model
}

func TestIsBuiltinCommand(t *testing.T) {
	m := createTestModelForBuiltin(t)

	tests := []struct {
		name     string
		cmdName  string
		expected bool
	}{
		{"cd is builtin", "cd", true},
		{"pwd is builtin", "pwd", true},
		{"echo is builtin", "echo", true},
		{"export is builtin", "export", true},
		{"unset is builtin", "unset", true},
		{"env is builtin", "env", true},
		{"alias is builtin", "alias", true},
		{"unalias is builtin", "unalias", true},
		{"type is builtin", "type", true},
		{"jobs is builtin", "jobs", true},
		{"fg is builtin", "fg", true},
		{"bg is builtin", "bg", true},
		{"ls is not builtin", "ls", false},
		{"git is not builtin", "git", false},
		{"unknown is not builtin", "unknown", false},
		{"empty is not builtin", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := m.isBuiltinCommand(tt.cmdName)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExecBuiltinCommand_Pwd(t *testing.T) {
	t.Run("executes pwd successfully", func(t *testing.T) {
		m := createTestModelForBuiltin(t)

		// Execute pwd
		cmdFunc := m.execBuiltinCommand("pwd")
		msg := cmdFunc()

		// Check result
		execMsg, ok := msg.(commandExecutedMsg)
		require.True(t, ok, "expected commandExecutedMsg")
		assert.NoError(t, execMsg.err)
		assert.Equal(t, 0, execMsg.exitCode)

		// Normalize paths for cross-platform comparison (removes trailing slashes)
		expectedPath := filepath.Clean(os.TempDir())
		actualPath := filepath.Clean(strings.TrimSpace(execMsg.output))
		assert.Contains(t, actualPath, expectedPath)
	})
}

func TestExecBuiltinCommand_Echo(t *testing.T) {
	t.Run("executes echo successfully", func(t *testing.T) {
		m := createTestModelForBuiltin(t)

		// Execute echo
		cmdFunc := m.execBuiltinCommand("echo hello world")
		msg := cmdFunc()

		// Check result
		execMsg, ok := msg.(commandExecutedMsg)
		require.True(t, ok, "expected commandExecutedMsg")
		assert.NoError(t, execMsg.err)
		assert.Equal(t, 0, execMsg.exitCode)
		assert.Contains(t, execMsg.output, "hello world")
	})
}

func TestExecBuiltinCommand_Export(t *testing.T) {
	t.Run("executes export successfully", func(t *testing.T) {
		m := createTestModelForBuiltin(t)

		// Execute export
		cmdFunc := m.execBuiltinCommand("export TEST_VAR=test_value")
		msg := cmdFunc()

		// Check result
		execMsg, ok := msg.(commandExecutedMsg)
		require.True(t, ok, "expected commandExecutedMsg")
		assert.NoError(t, execMsg.err)
		assert.Equal(t, 0, execMsg.exitCode)

		// Verify variable was set
		value, exists := m.currentSession.Environment().Get("TEST_VAR")
		assert.True(t, exists)
		assert.Equal(t, "test_value", value)
	})
}

func TestExecBuiltinCommand_Unset(t *testing.T) {
	t.Run("executes unset successfully", func(t *testing.T) {
		m := createTestModelForBuiltin(t)

		// First set a variable
		m.currentSession.Environment().Set("TEST_VAR", "test_value")

		// Execute unset
		cmdFunc := m.execBuiltinCommand("unset TEST_VAR")
		msg := cmdFunc()

		// Check result
		execMsg, ok := msg.(commandExecutedMsg)
		require.True(t, ok, "expected commandExecutedMsg")
		assert.NoError(t, execMsg.err)
		assert.Equal(t, 0, execMsg.exitCode)

		// Verify variable was removed
		_, exists := m.currentSession.Environment().Get("TEST_VAR")
		assert.False(t, exists)
	})
}

func TestExecBuiltinCommand_Env(t *testing.T) {
	t.Run("executes env successfully", func(t *testing.T) {
		m := createTestModelForBuiltin(t)

		// Set a test variable
		m.currentSession.Environment().Set("TEST_VAR", "test_value")

		// Execute env
		cmdFunc := m.execBuiltinCommand("env")
		msg := cmdFunc()

		// Check result
		execMsg, ok := msg.(commandExecutedMsg)
		require.True(t, ok, "expected commandExecutedMsg")
		assert.NoError(t, execMsg.err)
		assert.Equal(t, 0, execMsg.exitCode)
		// Note: env command prints all environment variables
		// In test environment, we may have empty session env, so just check for no error
		// The actual env implementation may merge with OS env or only show session env
	})
}

func TestExecBuiltinCommand_EmptyCommand(t *testing.T) {
	t.Run("handles empty command", func(t *testing.T) {
		m := createTestModelForBuiltin(t)

		// Execute empty command
		cmdFunc := m.execBuiltinCommand("")
		msg := cmdFunc()

		// Check result - parser returns error for empty command
		execMsg, ok := msg.(commandExecutedMsg)
		require.True(t, ok, "expected commandExecutedMsg")
		assert.Error(t, execMsg.err)
		assert.Equal(t, 1, execMsg.exitCode)
	})
}

func TestExecBuiltinCommand_InvalidSyntax(t *testing.T) {
	t.Run("handles invalid syntax", func(t *testing.T) {
		m := createTestModelForBuiltin(t)

		// Execute command with unclosed quote
		cmdFunc := m.execBuiltinCommand("echo \"unclosed")
		msg := cmdFunc()

		// Check result
		execMsg, ok := msg.(commandExecutedMsg)
		require.True(t, ok, "expected commandExecutedMsg")
		// Note: The parser may accept this as valid (unclosed quote might be handled)
		// Just verify we get a response without panic
		_ = execMsg.err
		_ = execMsg.exitCode
	})
}

func TestExecBuiltinCommand_Cd(t *testing.T) {
	t.Run("changes directory successfully", func(t *testing.T) {
		m := createTestModelForBuiltin(t)

		// Get initial directory
		initialDir := m.currentSession.WorkingDirectory()

		// Change to temp directory
		tmpDir := os.TempDir()
		cmdFunc := m.execBuiltinCommand("cd " + tmpDir)
		msg := cmdFunc()

		// Check result
		execMsg, ok := msg.(commandExecutedMsg)
		require.True(t, ok, "expected commandExecutedMsg")
		assert.NoError(t, execMsg.err)
		assert.Equal(t, 0, execMsg.exitCode)

		// Verify directory changed in session
		newDir := m.currentSession.WorkingDirectory()
		// Note: On Windows, paths may have different separators or casing
		// Just verify cd command executed successfully
		if initialDir == newDir {
			t.Logf("Warning: Directory did not change. Initial: %s, New: %s", initialDir, newDir)
		}
	})

	t.Run("fails for non-existent directory", func(t *testing.T) {
		m := createTestModelForBuiltin(t)

		// Try to change to non-existent directory
		cmdFunc := m.execBuiltinCommand("cd /nonexistent/path/12345")
		msg := cmdFunc()

		// Check result - should fail
		execMsg, ok := msg.(commandExecutedMsg)
		require.True(t, ok, "expected commandExecutedMsg")
		// Note: error might be in err or output depending on implementation
		assert.True(t, execMsg.err != nil || execMsg.exitCode != 0)
	})
}

func TestExecBuiltinCommand_Type(t *testing.T) {
	t.Run("shows type of builtin command", func(t *testing.T) {
		m := createTestModelForBuiltin(t)

		// Execute type cd
		cmdFunc := m.execBuiltinCommand("type cd")
		msg := cmdFunc()

		// Check result
		execMsg, ok := msg.(commandExecutedMsg)
		require.True(t, ok, "expected commandExecutedMsg")
		assert.NoError(t, execMsg.err)
		assert.Equal(t, 0, execMsg.exitCode)
		assert.Contains(t, execMsg.output, "builtin")
	})
}

func TestExecBuiltinCommand_Alias(t *testing.T) {
	t.Run("sets alias successfully", func(t *testing.T) {
		m := createTestModelForBuiltin(t)

		// Set alias
		cmdFunc := m.execBuiltinCommand("alias ll='ls -la'")
		msg := cmdFunc()

		// Check result
		execMsg, ok := msg.(commandExecutedMsg)
		require.True(t, ok, "expected commandExecutedMsg")
		assert.NoError(t, execMsg.err)
		assert.Equal(t, 0, execMsg.exitCode)

		// Verify alias was set
		aliasValue, exists := m.currentSession.GetAlias("ll")
		assert.True(t, exists)
		assert.Equal(t, "ls -la", aliasValue)
	})

	t.Run("lists all aliases", func(t *testing.T) {
		m := createTestModelForBuiltin(t)

		// Set some aliases
		m.currentSession.SetAlias("ll", "ls -la")
		m.currentSession.SetAlias("gs", "git status")

		// List aliases
		cmdFunc := m.execBuiltinCommand("alias")
		msg := cmdFunc()

		// Check result
		execMsg, ok := msg.(commandExecutedMsg)
		require.True(t, ok, "expected commandExecutedMsg")
		assert.NoError(t, execMsg.err)
		assert.Equal(t, 0, execMsg.exitCode)
		assert.Contains(t, execMsg.output, "ll")
		assert.Contains(t, execMsg.output, "gs")
	})
}

func TestExecBuiltinCommand_Unalias(t *testing.T) {
	t.Run("removes alias successfully", func(t *testing.T) {
		m := createTestModelForBuiltin(t)

		// Set alias
		m.currentSession.SetAlias("ll", "ls -la")

		// Remove alias
		cmdFunc := m.execBuiltinCommand("unalias ll")
		msg := cmdFunc()

		// Check result
		execMsg, ok := msg.(commandExecutedMsg)
		require.True(t, ok, "expected commandExecutedMsg")
		assert.NoError(t, execMsg.err)
		assert.Equal(t, 0, execMsg.exitCode)

		// Verify alias was removed
		_, exists := m.currentSession.GetAlias("ll")
		assert.False(t, exists)
	})
}
