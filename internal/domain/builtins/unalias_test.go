package builtins

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUnaliasCommand_Execute_Single(t *testing.T) {
	// Arrange
	sess := createTestSession(t)
	sess.SetAlias("ll", "ls -la")

	// Act
	cmd, err := NewUnaliasCommand([]string{"ll"}, sess)
	require.NoError(t, err)
	err = cmd.Execute()

	// Assert
	require.NoError(t, err)
	_, ok := sess.GetAlias("ll")
	assert.False(t, ok)
}

func TestUnaliasCommand_Execute_Multiple(t *testing.T) {
	// Arrange
	sess := createTestSession(t)
	sess.SetAlias("ll", "ls -la")
	sess.SetAlias("gs", "git status")
	sess.SetAlias("gp", "git push")

	// Act
	cmd, err := NewUnaliasCommand([]string{"ll", "gs"}, sess)
	require.NoError(t, err)
	err = cmd.Execute()

	// Assert
	require.NoError(t, err)
	_, ok1 := sess.GetAlias("ll")
	_, ok2 := sess.GetAlias("gs")
	_, ok3 := sess.GetAlias("gp")
	assert.False(t, ok1)
	assert.False(t, ok2)
	assert.True(t, ok3) // gp should still exist
}

func TestUnaliasCommand_Execute_All(t *testing.T) {
	// Arrange
	sess := createTestSession(t)
	sess.SetAlias("ll", "ls -la")
	sess.SetAlias("gs", "git status")
	sess.SetAlias("gp", "git push")

	// Act
	cmd, err := NewUnaliasCommand([]string{"-a"}, sess)
	require.NoError(t, err)
	err = cmd.Execute()

	// Assert
	require.NoError(t, err)
	aliases := sess.GetAllAliases()
	assert.Empty(t, aliases)
}

func TestUnaliasCommand_Execute_NotFound(t *testing.T) {
	// Arrange
	sess := createTestSession(t)

	// Act
	cmd, err := NewUnaliasCommand([]string{"nonexistent"}, sess)
	require.NoError(t, err)
	err = cmd.Execute()

	// Assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), "alias not found")
}

func TestNewUnaliasCommand_NoArgs(t *testing.T) {
	// Arrange
	sess := createTestSession(t)

	// Act
	cmd, err := NewUnaliasCommand([]string{}, sess)

	// Assert
	require.Error(t, err)
	assert.Nil(t, cmd)
	assert.Contains(t, err.Error(), "missing alias name")
}

func TestNewUnaliasCommand_NilSession(t *testing.T) {
	// Act
	cmd, err := NewUnaliasCommand([]string{"ll"}, nil)

	// Assert
	require.Error(t, err)
	assert.Nil(t, cmd)
}
