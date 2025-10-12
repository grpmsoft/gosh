package repl

import (
	"context"
	"log/slog"
	"os"
	"testing"

	"github.com/grpmsoft/gosh/internal/domain/config"
	"github.com/grpmsoft/gosh/internal/domain/session"
	"github.com/grpmsoft/gosh/internal/domain/shared"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// createTestModel создает тестовую модель REPL для тестирования
func createTestModel(t *testing.T) *Model {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	cfg := config.DefaultConfig()

	// Создаем пустое окружение для тестов
	env := make(shared.Environment)

	// Создаем тестовую сессию напрямую
	sess, err := session.NewSession(
		"test-session",
		os.Getenv("HOME"),
		env,
	)
	require.NoError(t, err)

	// Создаем минимальную модель для тестирования expandAliases
	model := &Model{
		currentSession: sess,
		logger:         logger,
		ctx:            context.Background(),
		config:         cfg,
	}

	return model
}

func TestExpandAliases_NoAlias(t *testing.T) {
	// Arrange
	m := createTestModel(t)

	// Act
	expanded, err := m.expandAliases("ls -la", 0)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, "ls -la", expanded)
}

func TestExpandAliases_SingleAlias(t *testing.T) {
	// Arrange
	m := createTestModel(t)
	m.currentSession.SetAlias("ll", "ls -la")

	// Act
	expanded, err := m.expandAliases("ll", 0)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, "ls -la", expanded)
}

func TestExpandAliases_AliasWithArguments(t *testing.T) {
	// Arrange
	m := createTestModel(t)
	m.currentSession.SetAlias("ll", "ls -la")

	// Act
	expanded, err := m.expandAliases("ll /tmp", 0)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, "ls -la /tmp", expanded)
}

func TestExpandAliases_RecursiveAlias(t *testing.T) {
	// Arrange
	m := createTestModel(t)
	m.currentSession.SetAlias("l", "ls")
	m.currentSession.SetAlias("ll", "l -la")

	// Act
	expanded, err := m.expandAliases("ll", 0)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, "ls -la", expanded)
}

func TestExpandAliases_RecursiveAliasWithArguments(t *testing.T) {
	// Arrange
	m := createTestModel(t)
	m.currentSession.SetAlias("l", "ls")
	m.currentSession.SetAlias("ll", "l -la")

	// Act
	expanded, err := m.expandAliases("ll /home", 0)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, "ls -la /home", expanded)
}

func TestExpandAliases_CircularAlias(t *testing.T) {
	// Arrange
	m := createTestModel(t)
	m.currentSession.SetAlias("a", "b")
	m.currentSession.SetAlias("b", "c")
	m.currentSession.SetAlias("c", "d")
	m.currentSession.SetAlias("d", "e")
	m.currentSession.SetAlias("e", "f")
	m.currentSession.SetAlias("f", "g")
	m.currentSession.SetAlias("g", "h")
	m.currentSession.SetAlias("h", "i")
	m.currentSession.SetAlias("i", "j")
	m.currentSession.SetAlias("j", "k")
	m.currentSession.SetAlias("k", "a") // Circular reference

	// Act
	_, err := m.expandAliases("a", 0)

	// Assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), "exceeded maximum depth")
}

func TestExpandAliases_DeepButNotCircular(t *testing.T) {
	// Arrange
	m := createTestModel(t)
	// Create a chain of 10 aliases (should work, limit is 10)
	m.currentSession.SetAlias("a1", "a2")
	m.currentSession.SetAlias("a2", "a3")
	m.currentSession.SetAlias("a3", "a4")
	m.currentSession.SetAlias("a4", "a5")
	m.currentSession.SetAlias("a5", "a6")
	m.currentSession.SetAlias("a6", "a7")
	m.currentSession.SetAlias("a7", "a8")
	m.currentSession.SetAlias("a8", "a9")
	m.currentSession.SetAlias("a9", "echo test")

	// Act
	expanded, err := m.expandAliases("a1", 0)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, "echo test", expanded)
}

func TestExpandAliases_ComplexCommand(t *testing.T) {
	// Arrange
	m := createTestModel(t)
	m.currentSession.SetAlias("gs", "git status")

	// Act
	expanded, err := m.expandAliases("gs --short", 0)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, "git status --short", expanded)
}

func TestExpandAliases_AliasToBuiltin(t *testing.T) {
	// Arrange
	m := createTestModel(t)
	m.currentSession.SetAlias("p", "pwd")

	// Act
	expanded, err := m.expandAliases("p", 0)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, "pwd", expanded)
}

func TestExpandAliases_EmptyCommand(t *testing.T) {
	// Arrange
	m := createTestModel(t)

	// Act
	expanded, err := m.expandAliases("", 0)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, "", expanded)
}

func TestExpandAliases_MultipleAliasesInSession(t *testing.T) {
	// Arrange
	m := createTestModel(t)
	m.currentSession.SetAlias("ll", "ls -la")
	m.currentSession.SetAlias("gs", "git status")
	m.currentSession.SetAlias("gp", "git push")

	// Act - only first command should be expanded
	expanded, err := m.expandAliases("gs", 0)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, "git status", expanded)
}
