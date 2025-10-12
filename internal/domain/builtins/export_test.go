package builtins

import (
	"bytes"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExportCommand_Execute_SingleVariable(t *testing.T) {
	// Arrange
	sess := createTestSession(t)
	stdout := &bytes.Buffer{}

	// Act
	cmd, err := NewExportCommand([]string{"TEST_VAR=test_value"}, sess, stdout)
	require.NoError(t, err)
	err = cmd.Execute()

	// Assert
	require.NoError(t, err)
	value, ok := sess.Environment().Get("TEST_VAR")
	assert.True(t, ok)
	assert.Equal(t, "test_value", value)
	assert.Equal(t, "test_value", os.Getenv("TEST_VAR"))

	// Cleanup
	os.Unsetenv("TEST_VAR")
}

func TestExportCommand_Execute_MultipleVariables(t *testing.T) {
	// Arrange
	sess := createTestSession(t)
	stdout := &bytes.Buffer{}

	// Act
	cmd, err := NewExportCommand([]string{"VAR1=value1", "VAR2=value2"}, sess, stdout)
	require.NoError(t, err)
	err = cmd.Execute()

	// Assert
	require.NoError(t, err)
	v1, ok1 := sess.Environment().Get("VAR1")
	v2, ok2 := sess.Environment().Get("VAR2")
	assert.True(t, ok1)
	assert.True(t, ok2)
	assert.Equal(t, "value1", v1)
	assert.Equal(t, "value2", v2)

	// Cleanup
	os.Unsetenv("VAR1")
	os.Unsetenv("VAR2")
}

func TestExportCommand_Execute_WithQuotes(t *testing.T) {
	// Arrange
	sess := createTestSession(t)
	stdout := &bytes.Buffer{}

	// Act
	cmd, err := NewExportCommand([]string{`TEST_VAR="hello world"`}, sess, stdout)
	require.NoError(t, err)
	err = cmd.Execute()

	// Assert
	require.NoError(t, err)
	value, ok := sess.Environment().Get("TEST_VAR")
	assert.True(t, ok)
	assert.Equal(t, "hello world", value) // Quotes should be removed

	// Cleanup
	os.Unsetenv("TEST_VAR")
}

func TestExportCommand_Execute_InvalidFormat(t *testing.T) {
	// Arrange
	sess := createTestSession(t)
	stdout := &bytes.Buffer{}

	// Act
	cmd, err := NewExportCommand([]string{"INVALID_FORMAT"}, sess, stdout)
	require.NoError(t, err)
	err = cmd.Execute()

	// Assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid format")
}

func TestExportCommand_Execute_InvalidVariableName(t *testing.T) {
	// Arrange
	sess := createTestSession(t)
	stdout := &bytes.Buffer{}

	testCases := []struct {
		name string
		arg  string
	}{
		{"starts with digit", "1VAR=value"},
		{"contains dash", "VAR-NAME=value"},
		{"contains space", "VAR NAME=value"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Act
			cmd, err := NewExportCommand([]string{tc.arg}, sess, stdout)
			require.NoError(t, err)
			err = cmd.Execute()

			// Assert
			require.Error(t, err)
			assert.Contains(t, err.Error(), "invalid variable name")
		})
	}
}

func TestExportCommand_Execute_NoArgs_PrintsAllVariables(t *testing.T) {
	// Arrange
	sess := createTestSession(t)
	sess.SetEnvironmentVariable("TEST_VAR1", "value1")
	sess.SetEnvironmentVariable("TEST_VAR2", "value2")

	stdout := &bytes.Buffer{}

	// Act
	cmd, err := NewExportCommand([]string{}, sess, stdout)
	require.NoError(t, err)
	err = cmd.Execute()

	// Assert
	require.NoError(t, err)
	output := stdout.String()
	assert.Contains(t, output, "TEST_VAR1")
	assert.Contains(t, output, "TEST_VAR2")
	assert.Contains(t, output, "value1")
	assert.Contains(t, output, "value2")
}

func TestNewExportCommand_NilSession(t *testing.T) {
	// Act
	cmd, err := NewExportCommand([]string{"VAR=value"}, nil, &bytes.Buffer{})

	// Assert
	require.Error(t, err)
	assert.Nil(t, cmd)
	assert.Contains(t, err.Error(), "session cannot be nil")
}

func TestNewExportCommand_NilStdout(t *testing.T) {
	// Arrange
	sess := createTestSession(t)

	// Act
	cmd, err := NewExportCommand([]string{"VAR=value"}, sess, nil)

	// Assert
	require.Error(t, err)
	assert.Nil(t, cmd)
	assert.Contains(t, err.Error(), "stdout cannot be nil")
}
