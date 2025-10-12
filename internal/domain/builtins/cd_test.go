package builtins

import (
	"github.com/grpmsoft/gosh/internal/domain/session"
	"github.com/grpmsoft/gosh/internal/domain/shared"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCdCommand_Execute_NoArgs(t *testing.T) {
	// Arrange
	homeDir, _ := os.UserHomeDir()
	sess := createTestSession(t)

	// Act
	cmd, err := NewCdCommand([]string{}, sess)
	require.NoError(t, err)
	err = cmd.Execute()

	// Assert
	require.NoError(t, err)
	assert.Equal(t, homeDir, sess.WorkingDirectory())
}

func TestCdCommand_Execute_AbsolutePath(t *testing.T) {
	// Arrange
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir) // Restore original directory after test

	tmpDir := t.TempDir()
	sess := createTestSession(t)

	// Act
	cmd, err := NewCdCommand([]string{tmpDir}, sess)
	require.NoError(t, err)
	err = cmd.Execute()

	// Assert
	require.NoError(t, err)
	assert.Equal(t, tmpDir, sess.WorkingDirectory())
}

func TestCdCommand_Execute_RelativePath(t *testing.T) {
	// Arrange
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir) // Restore original directory after test

	tmpDir := t.TempDir()
	subDir := filepath.Join(tmpDir, "subdir")
	os.Mkdir(subDir, 0755)

	sess := createTestSession(t)
	sess.ChangeDirectory(tmpDir)
	os.Chdir(tmpDir)

	// Act
	cmd, err := NewCdCommand([]string{"subdir"}, sess)
	require.NoError(t, err)
	err = cmd.Execute()

	// Assert
	require.NoError(t, err)
	assert.Equal(t, subDir, sess.WorkingDirectory())
}

func TestCdCommand_Execute_Tilde(t *testing.T) {
	// Arrange
	homeDir, _ := os.UserHomeDir()
	sess := createTestSession(t)

	// Act
	cmd, err := NewCdCommand([]string{"~"}, sess)
	require.NoError(t, err)
	err = cmd.Execute()

	// Assert
	require.NoError(t, err)
	assert.Equal(t, homeDir, sess.WorkingDirectory())
}

func TestCdCommand_Execute_TildeWithPath(t *testing.T) {
	// Arrange
	homeDir, _ := os.UserHomeDir()
	sess := createTestSession(t)

	// Create test directory in home
	testDir := filepath.Join(homeDir, ".gosh_test_cd")
	os.MkdirAll(testDir, 0755)
	defer os.RemoveAll(testDir)

	// Act
	cmd, err := NewCdCommand([]string{"~/.gosh_test_cd"}, sess)
	require.NoError(t, err)
	err = cmd.Execute()

	// Assert
	require.NoError(t, err)
	assert.Equal(t, testDir, sess.WorkingDirectory())
}

func TestCdCommand_Execute_Dash_PreviousDirectory(t *testing.T) {
	// Arrange
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir) // Restore original directory after test

	tmpDir1 := t.TempDir()
	tmpDir2 := t.TempDir()

	sess := createTestSession(t)

	// First cd to tmpDir1
	sess.ChangeDirectory(tmpDir1)
	os.Chdir(tmpDir1)

	// Then cd to tmpDir2
	sess.ChangeDirectory(tmpDir2)
	os.Chdir(tmpDir2)

	// Act - cd back to tmpDir1 using cd -
	cmd, err := NewCdCommand([]string{"-"}, sess)
	require.NoError(t, err)
	err = cmd.Execute()

	// Assert
	require.NoError(t, err)
	assert.Equal(t, tmpDir1, sess.WorkingDirectory())
}

func TestCdCommand_Execute_Dash_NoPreviousDirectory(t *testing.T) {
	// Arrange
	sess := createTestSession(t)
	// previousDir is empty by default

	// Act
	cmd, err := NewCdCommand([]string{"-"}, sess)
	require.NoError(t, err)
	err = cmd.Execute()

	// Assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), "OLDPWD not set")
}

func TestCdCommand_Execute_DirectoryNotFound(t *testing.T) {
	// Arrange
	sess := createTestSession(t)

	// Act
	cmd, err := NewCdCommand([]string{"/nonexistent/directory"}, sess)
	require.NoError(t, err)
	err = cmd.Execute()

	// Assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no such file or directory")
}

func TestCdCommand_Execute_NotADirectory(t *testing.T) {
	// Arrange
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "file.txt")
	os.WriteFile(tmpFile, []byte("test"), 0644)

	sess := createTestSession(t)

	// Act
	cmd, err := NewCdCommand([]string{tmpFile}, sess)
	require.NoError(t, err)
	err = cmd.Execute()

	// Assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not a directory")
}

func TestNewCdCommand_NilSession(t *testing.T) {
	// Act
	cmd, err := NewCdCommand([]string{"/tmp"}, nil)

	// Assert
	require.Error(t, err)
	assert.Nil(t, cmd)
	assert.Contains(t, err.Error(), "session cannot be nil")
}

// Helper function
func createTestSession(t *testing.T) *session.Session {
	// Use a safe working directory (temp dir or home)
	workingDir := t.TempDir()

	env := make(shared.Environment)
	env["PATH"] = "/usr/bin"
	env["HOME"] = os.Getenv("HOME")
	if env["HOME"] == "" {
		env["HOME"] = os.Getenv("USERPROFILE") // Windows
	}

	sess, err := session.NewSession("test-session", workingDir, env)
	require.NoError(t, err)

	return sess
}
