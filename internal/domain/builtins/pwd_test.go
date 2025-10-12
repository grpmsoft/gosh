package builtins

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPwdCommand_Execute(t *testing.T) {
	// Arrange
	sess := createTestSession(t)
	stdout := &bytes.Buffer{}

	// Act
	cmd, err := NewPwdCommand(sess, stdout)
	require.NoError(t, err)
	err = cmd.Execute()

	// Assert
	require.NoError(t, err)
	output := stdout.String()
	assert.Contains(t, output, sess.WorkingDirectory())
}

func TestNewPwdCommand_NilSession(t *testing.T) {
	// Act
	cmd, err := NewPwdCommand(nil, &bytes.Buffer{})

	// Assert
	require.Error(t, err)
	assert.Nil(t, cmd)
}

func TestNewPwdCommand_NilStdout(t *testing.T) {
	// Arrange
	sess := createTestSession(t)

	// Act
	cmd, err := NewPwdCommand(sess, nil)

	// Assert
	require.Error(t, err)
	assert.Nil(t, cmd)
}
