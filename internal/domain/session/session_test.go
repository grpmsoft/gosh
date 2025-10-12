package session

import (
	"github.com/grpmsoft/gosh/internal/domain/shared"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSession_Success(t *testing.T) {
	// Arrange
	id := "test-session"
	workingDir, _ := os.Getwd()
	env := make(shared.Environment)
	env["TEST"] = "value"

	// Act
	sess, err := NewSession(id, workingDir, env)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, sess)
	assert.Equal(t, id, sess.ID())
	assert.True(t, sess.IsActive())
}

func TestNewSession_EmptyID(t *testing.T) {
	// Arrange
	id := ""
	workingDir, _ := os.Getwd()
	env := make(shared.Environment)

	// Act
	sess, err := NewSession(id, workingDir, env)

	// Assert
	require.Error(t, err)
	assert.Nil(t, sess)
}

func TestNewSession_EmptyWorkingDir(t *testing.T) {
	// Arrange
	id := "test"
	workingDir := ""
	env := make(shared.Environment)

	// Act
	sess, err := NewSession(id, workingDir, env)

	// Assert
	require.Error(t, err)
	assert.Nil(t, sess)
}

func TestSession_ChangeDirectory(t *testing.T) {
	// Arrange
	sess := createTestSession(t)
	newDir := os.TempDir()

	// Act
	err := sess.ChangeDirectory(newDir)

	// Assert
	require.NoError(t, err)
	absPath, _ := filepath.Abs(newDir)
	assert.Equal(t, absPath, sess.WorkingDirectory())
}

func TestSession_SetEnvironmentVariable(t *testing.T) {
	// Arrange
	sess := createTestSession(t)

	// Act
	err := sess.SetEnvironmentVariable("TEST_VAR", "test_value")

	// Assert
	require.NoError(t, err)
	env := sess.Environment()
	value, ok := env.Get("TEST_VAR")
	assert.True(t, ok)
	assert.Equal(t, "test_value", value)
}

func TestSession_UnsetEnvironmentVariable(t *testing.T) {
	// Arrange
	sess := createTestSession(t)
	sess.SetEnvironmentVariable("TO_REMOVE", "value")

	// Act
	err := sess.UnsetEnvironmentVariable("TO_REMOVE")

	// Assert
	require.NoError(t, err)
	env := sess.Environment()
	_, ok := env.Get("TO_REMOVE")
	assert.False(t, ok)
}

func TestSession_AddToHistory(t *testing.T) {
	// Arrange
	sess := createTestSession(t)

	// Act
	err := sess.AddToHistory("ls -la")
	require.NoError(t, err)
	err = sess.AddToHistory("pwd")
	require.NoError(t, err)

	// Assert
	historySlice := sess.History().ToSlice()
	assert.Len(t, historySlice, 2)
	assert.Equal(t, "ls -la", historySlice[0])
	assert.Equal(t, "pwd", historySlice[1])
}

func TestSession_AddToHistory_SkipsEmpty(t *testing.T) {
	// Arrange
	sess := createTestSession(t)

	// Act
	err := sess.AddToHistory("")

	// Assert
	require.NoError(t, err)
	assert.Equal(t, 0, sess.History().Size())
}

func TestSession_SetVariable(t *testing.T) {
	// Arrange
	sess := createTestSession(t)

	// Act
	err := sess.SetVariable("MY_VAR", "my_value")

	// Assert
	require.NoError(t, err)
	value, ok := sess.GetVariable("MY_VAR")
	assert.True(t, ok)
	assert.Equal(t, "my_value", value)
}

func TestSession_SetAlias(t *testing.T) {
	// Arrange
	sess := createTestSession(t)

	// Act
	err := sess.SetAlias("ll", "ls -la")

	// Assert
	require.NoError(t, err)
	alias, ok := sess.GetAlias("ll")
	assert.True(t, ok)
	assert.Equal(t, "ls -la", alias)
}

func TestSession_Close(t *testing.T) {
	// Arrange
	sess := createTestSession(t)
	assert.True(t, sess.IsActive())

	// Act
	err := sess.Close()

	// Assert
	require.NoError(t, err)
	assert.False(t, sess.IsActive())
}

func TestSession_OperationsAfterClose(t *testing.T) {
	// Arrange
	sess := createTestSession(t)
	sess.Close()

	// Act & Assert
	err := sess.AddToHistory("command")
	assert.Error(t, err)

	err = sess.SetEnvironmentVariable("VAR", "value")
	assert.Error(t, err)
}

// Helper function
func createTestSession(t *testing.T) *Session {
	workingDir, err := os.Getwd()
	require.NoError(t, err)

	env := make(shared.Environment)
	env["PATH"] = "/usr/bin"

	sess, err := NewSession("test-session", workingDir, env)
	require.NoError(t, err)

	return sess
}
