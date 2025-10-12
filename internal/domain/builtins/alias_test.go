package builtins

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAliasCommand_Execute_Create(t *testing.T) {
	// Arrange
	sess := createTestSession(t)
	stdout := &bytes.Buffer{}

	// Act
	cmd, err := NewAliasCommand([]string{"ll=ls -la"}, sess, stdout)
	require.NoError(t, err)
	err = cmd.Execute()

	// Assert
	require.NoError(t, err)
	command, ok := sess.GetAlias("ll")
	assert.True(t, ok)
	assert.Equal(t, "ls -la", command)
}

func TestAliasCommand_Execute_CreateWithQuotes(t *testing.T) {
	// Arrange
	sess := createTestSession(t)
	stdout := &bytes.Buffer{}

	testCases := []struct {
		name     string
		arg      string
		expected string
	}{
		{"single quotes", "ll='ls -la'", "ls -la"},
		{"double quotes", `ll="ls -la"`, "ls -la"},
		{"no quotes", "ll=ls -la", "ls -la"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Act
			cmd, err := NewAliasCommand([]string{tc.arg}, sess, stdout)
			require.NoError(t, err)
			err = cmd.Execute()

			// Assert
			require.NoError(t, err)
			command, ok := sess.GetAlias("ll")
			assert.True(t, ok)
			assert.Equal(t, tc.expected, command)
		})
	}
}

func TestAliasCommand_Execute_ListAll(t *testing.T) {
	// Arrange
	sess := createTestSession(t)
	sess.SetAlias("ll", "ls -la")
	sess.SetAlias("gs", "git status")
	stdout := &bytes.Buffer{}

	// Act
	cmd, err := NewAliasCommand([]string{}, sess, stdout)
	require.NoError(t, err)
	err = cmd.Execute()

	// Assert
	require.NoError(t, err)
	output := stdout.String()
	assert.Contains(t, output, "alias ll='ls -la'")
	assert.Contains(t, output, "alias gs='git status'")
}

func TestAliasCommand_Execute_PrintSpecific(t *testing.T) {
	// Arrange
	sess := createTestSession(t)
	sess.SetAlias("ll", "ls -la")
	stdout := &bytes.Buffer{}

	// Act
	cmd, err := NewAliasCommand([]string{"ll"}, sess, stdout)
	require.NoError(t, err)
	err = cmd.Execute()

	// Assert
	require.NoError(t, err)
	output := stdout.String()
	assert.Contains(t, output, "alias ll='ls -la'")
}

func TestAliasCommand_Execute_PrintNotFound(t *testing.T) {
	// Arrange
	sess := createTestSession(t)
	stdout := &bytes.Buffer{}

	// Act
	cmd, err := NewAliasCommand([]string{"nonexistent"}, sess, stdout)
	require.NoError(t, err)
	err = cmd.Execute()

	// Assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), "alias not found")
}

func TestAliasCommand_Execute_InvalidFormat(t *testing.T) {
	// Arrange
	sess := createTestSession(t)
	stdout := &bytes.Buffer{}

	// Act
	cmd, err := NewAliasCommand([]string{"invalid_no_equals"}, sess, stdout)
	require.NoError(t, err)
	// This should try to print "invalid_no_equals" alias
	err = cmd.Execute()

	// Assert - should fail because alias doesn't exist
	require.Error(t, err)
}

func TestAliasCommand_Execute_InvalidName(t *testing.T) {
	// Arrange
	sess := createTestSession(t)
	stdout := &bytes.Buffer{}

	testCases := []struct {
		name string
		arg  string
	}{
		{"contains space", "bad name=ls"},
		{"contains special char", "bad@name=ls"},
		{"empty name", "=ls"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Act
			cmd, err := NewAliasCommand([]string{tc.arg}, sess, stdout)
			require.NoError(t, err)
			err = cmd.Execute()

			// Assert
			require.Error(t, err)
		})
	}
}

func TestAliasCommand_Execute_EmptyCommand(t *testing.T) {
	// Arrange
	sess := createTestSession(t)
	stdout := &bytes.Buffer{}

	// Act
	cmd, err := NewAliasCommand([]string{"ll="}, sess, stdout)
	require.NoError(t, err)
	err = cmd.Execute()

	// Assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), "command cannot be empty")
}

func TestAliasCommand_Execute_Multiple(t *testing.T) {
	// Arrange
	sess := createTestSession(t)
	stdout := &bytes.Buffer{}

	// Act
	cmd, err := NewAliasCommand([]string{"ll=ls -la", "gs=git status"}, sess, stdout)
	require.NoError(t, err)
	err = cmd.Execute()

	// Assert
	require.NoError(t, err)
	cmd1, ok1 := sess.GetAlias("ll")
	cmd2, ok2 := sess.GetAlias("gs")
	assert.True(t, ok1)
	assert.True(t, ok2)
	assert.Equal(t, "ls -la", cmd1)
	assert.Equal(t, "git status", cmd2)
}

func TestNewAliasCommand_NilSession(t *testing.T) {
	// Act
	cmd, err := NewAliasCommand([]string{}, nil, &bytes.Buffer{})

	// Assert
	require.Error(t, err)
	assert.Nil(t, cmd)
}

func TestNewAliasCommand_NilStdout(t *testing.T) {
	// Arrange
	sess := createTestSession(t)

	// Act
	cmd, err := NewAliasCommand([]string{}, sess, nil)

	// Assert
	require.Error(t, err)
	assert.Nil(t, cmd)
}
