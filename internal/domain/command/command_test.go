package command

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCommand_Success(t *testing.T) {
	// Arrange
	name := "ls"
	args := []string{"-la", "/tmp"}

	// Act
	cmd, err := NewCommand(name, args, TypeExternal)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, cmd)
	assert.Equal(t, "ls", cmd.Name())
	assert.Equal(t, args, cmd.Args())
	assert.Equal(t, TypeExternal, cmd.Type())
	assert.False(t, cmd.IsBackground())
}

func TestNewCommand_EmptyName(t *testing.T) {
	// Arrange
	name := ""
	args := []string{}

	// Act
	cmd, err := NewCommand(name, args, TypeExternal)

	// Assert
	require.Error(t, err)
	assert.Nil(t, cmd)
}

func TestNewCommand_WhitespaceName(t *testing.T) {
	// Arrange
	name := "   "
	args := []string{}

	// Act
	cmd, err := NewCommand(name, args, TypeExternal)

	// Assert
	require.Error(t, err)
	assert.Nil(t, cmd)
}

func TestCommand_IsBuiltin(t *testing.T) {
	// Arrange
	cmd, err := NewCommand("cd", []string{"/tmp"}, TypeBuiltin)
	require.NoError(t, err)

	// Act & Assert
	assert.True(t, cmd.IsBuiltin())
	assert.False(t, cmd.IsExternal())
	assert.False(t, cmd.IsPipeline())
}

func TestCommand_AddRedirection(t *testing.T) {
	// Arrange
	cmd, err := NewCommand("ls", []string{}, TypeExternal)
	require.NoError(t, err)

	redir := Redirection{
		Type:   RedirectOutput,
		Target: "output.txt",
	}

	// Act
	err = cmd.AddRedirection(redir)

	// Assert
	require.NoError(t, err)
	redirects := cmd.Redirections()
	assert.Len(t, redirects, 1)
	assert.Equal(t, RedirectOutput, redirects[0].Type)
	assert.Equal(t, "output.txt", redirects[0].Target)
}

func TestCommand_SetBackground(t *testing.T) {
	// Arrange
	cmd, err := NewCommand("sleep", []string{"10"}, TypeExternal)
	require.NoError(t, err)

	// Act
	cmd.SetBackground(true)

	// Assert
	assert.True(t, cmd.IsBackground())
}

func TestCommand_FullCommand(t *testing.T) {
	// Arrange
	cmd, err := NewCommand("git", []string{"status", "-s"}, TypeExternal)
	require.NoError(t, err)

	// Act
	fullCmd := cmd.FullCommand()

	// Assert
	assert.Equal(t, "git status -s", fullCmd)
}

func TestCommand_Clone(t *testing.T) {
	// Arrange
	original, err := NewCommand("echo", []string{"hello"}, TypeExternal)
	require.NoError(t, err)

	// Act
	cloned := original.Clone()

	// Assert
	assert.Equal(t, original.Name(), cloned.Name())
	assert.Equal(t, original.Args(), cloned.Args())
	assert.Equal(t, original.Type(), cloned.Type())

	// Verify independence
	cloned.SetBackground(true)
	assert.False(t, original.IsBackground())
	assert.True(t, cloned.IsBackground())
}

func TestIsBuiltinCommand(t *testing.T) {
	tests := []struct {
		name     string
		command  string
		expected bool
	}{
		{"cd is builtin", "cd", true},
		{"pwd is builtin", "pwd", true},
		{"echo is builtin", "echo", true},
		{"exit is builtin", "exit", true},
		{"export is builtin", "export", true},
		{"ls is not builtin", "ls", false},
		{"git is not builtin", "git", false},
		{"unknown is not builtin", "unknown", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsBuiltinCommand(tt.command)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetCommandType(t *testing.T) {
	tests := []struct {
		name     string
		command  string
		expected Type
	}{
		{"cd returns builtin", "cd", TypeBuiltin},
		{"pwd returns builtin", "pwd", TypeBuiltin},
		{"ls returns external", "ls", TypeExternal},
		{"git returns external", "git", TypeExternal},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetCommandType(tt.command)
			assert.Equal(t, tt.expected, result)
		})
	}
}
