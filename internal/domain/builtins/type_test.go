package builtins

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTypeCommand_Execute_Builtin(t *testing.T) {
	// Arrange
	sess := createTestSession(t)
	stdout := &bytes.Buffer{}

	// Act
	cmd, err := NewTypeCommand([]string{"cd"}, sess, stdout)
	require.NoError(t, err)
	err = cmd.Execute()

	// Assert
	require.NoError(t, err)
	output := stdout.String()
	assert.Contains(t, output, "cd is a shell builtin")
}

func TestTypeCommand_Execute_External(t *testing.T) {
	// Arrange
	sess := createTestSession(t)
	stdout := &bytes.Buffer{}

	// Act - "go" is an external command if Go is installed
	cmd, err := NewTypeCommand([]string{"go"}, sess, stdout)
	require.NoError(t, err)
	err = cmd.Execute()

	// Assert
	require.NoError(t, err)
	output := stdout.String()
	// Should be either external or not found
	assert.True(t,
		bytes.Contains([]byte(output), []byte("go is")) ||
			bytes.Contains([]byte(output), []byte("not found")))
}

func TestTypeCommand_Execute_Alias(t *testing.T) {
	// Arrange
	sess := createTestSession(t)
	sess.SetAlias("ll", "ls -la")
	stdout := &bytes.Buffer{}

	// Act
	cmd, err := NewTypeCommand([]string{"ll"}, sess, stdout)
	require.NoError(t, err)
	err = cmd.Execute()

	// Assert
	require.NoError(t, err)
	output := stdout.String()
	assert.Contains(t, output, "ll is aliased to 'ls -la'")
}

func TestTypeCommand_Execute_NotFound(t *testing.T) {
	// Arrange
	sess := createTestSession(t)
	stdout := &bytes.Buffer{}

	// Act
	cmd, err := NewTypeCommand([]string{"nonexistentcommand12345"}, sess, stdout)
	require.NoError(t, err)
	err = cmd.Execute()

	// Assert
	require.NoError(t, err)
	output := stdout.String()
	assert.Contains(t, output, "not found")
}

func TestTypeCommand_Execute_MultipleCommands(t *testing.T) {
	// Arrange
	sess := createTestSession(t)
	stdout := &bytes.Buffer{}

	// Act
	cmd, err := NewTypeCommand([]string{"cd", "pwd"}, sess, stdout)
	require.NoError(t, err)
	err = cmd.Execute()

	// Assert
	require.NoError(t, err)
	output := stdout.String()
	assert.Contains(t, output, "cd is a shell builtin")
	assert.Contains(t, output, "pwd is a shell builtin")
}

func TestNewTypeCommand_NoArgs(t *testing.T) {
	// Arrange
	sess := createTestSession(t)

	// Act
	cmd, err := NewTypeCommand([]string{}, sess, &bytes.Buffer{})

	// Assert
	require.Error(t, err)
	assert.Nil(t, cmd)
	assert.Contains(t, err.Error(), "missing command name")
}

func TestNewTypeCommand_NilSession(t *testing.T) {
	// Act
	cmd, err := NewTypeCommand([]string{"cd"}, nil, &bytes.Buffer{})

	// Assert
	require.Error(t, err)
	assert.Nil(t, cmd)
}

func TestNewTypeCommand_NilStdout(t *testing.T) {
	// Arrange
	sess := createTestSession(t)

	// Act
	cmd, err := NewTypeCommand([]string{"cd"}, sess, nil)

	// Assert
	require.Error(t, err)
	assert.Nil(t, cmd)
}
