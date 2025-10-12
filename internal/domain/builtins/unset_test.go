package builtins

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUnsetCommand_Execute_SingleVariable(t *testing.T) {
	// Arrange
	sess := createTestSession(t)
	sess.SetEnvironmentVariable("TEST_VAR", "value")
	os.Setenv("TEST_VAR", "value")

	// Act
	cmd, err := NewUnsetCommand([]string{"TEST_VAR"}, sess)
	require.NoError(t, err)
	err = cmd.Execute()

	// Assert
	require.NoError(t, err)
	_, ok := sess.Environment().Get("TEST_VAR")
	assert.False(t, ok)
	assert.Empty(t, os.Getenv("TEST_VAR"))
}

func TestUnsetCommand_Execute_MultipleVariables(t *testing.T) {
	// Arrange
	sess := createTestSession(t)
	sess.SetEnvironmentVariable("VAR1", "value1")
	sess.SetEnvironmentVariable("VAR2", "value2")
	os.Setenv("VAR1", "value1")
	os.Setenv("VAR2", "value2")

	// Act
	cmd, err := NewUnsetCommand([]string{"VAR1", "VAR2"}, sess)
	require.NoError(t, err)
	err = cmd.Execute()

	// Assert
	require.NoError(t, err)
	_, ok1 := sess.Environment().Get("VAR1")
	_, ok2 := sess.Environment().Get("VAR2")
	assert.False(t, ok1)
	assert.False(t, ok2)
}

func TestNewUnsetCommand_NoArgs(t *testing.T) {
	// Arrange
	sess := createTestSession(t)

	// Act
	cmd, err := NewUnsetCommand([]string{}, sess)

	// Assert
	require.Error(t, err)
	assert.Nil(t, cmd)
	assert.Contains(t, err.Error(), "missing variable name")
}

func TestNewUnsetCommand_NilSession(t *testing.T) {
	// Act
	cmd, err := NewUnsetCommand([]string{"VAR"}, nil)

	// Assert
	require.Error(t, err)
	assert.Nil(t, cmd)
}
