package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGoshrcService_Load_FileNotExists(t *testing.T) {
	// Arrange
	tmpDir := t.TempDir()
	goshrcPath := filepath.Join(tmpDir, ".goshrc")
	service := NewGoshrcService(goshrcPath)

	// Act
	data, err := service.Load()

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, data)
	assert.Empty(t, data.Aliases)
	assert.Empty(t, data.Environment)
}

func TestGoshrcService_Load_EmptyFile(t *testing.T) {
	// Arrange
	tmpDir := t.TempDir()
	goshrcPath := filepath.Join(tmpDir, ".goshrc")

	// Create empty file
	err := os.WriteFile(goshrcPath, []byte(""), 0o644)
	require.NoError(t, err)

	service := NewGoshrcService(goshrcPath)

	// Act
	data, err := service.Load()

	// Assert
	require.NoError(t, err)
	assert.Empty(t, data.Aliases)
	assert.Empty(t, data.Environment)
}

func TestGoshrcService_Load_WithAliases(t *testing.T) {
	// Arrange
	tmpDir := t.TempDir()
	goshrcPath := filepath.Join(tmpDir, ".goshrc")

	content := `# Comment
alias ll='ls -la'
alias gs='git status'
alias gp="git push"
alias pwd=pwd
`
	err := os.WriteFile(goshrcPath, []byte(content), 0o644)
	require.NoError(t, err)

	service := NewGoshrcService(goshrcPath)

	// Act
	data, err := service.Load()

	// Assert
	require.NoError(t, err)
	assert.Len(t, data.Aliases, 4)
	assert.Equal(t, "ls -la", data.Aliases["ll"])
	assert.Equal(t, "git status", data.Aliases["gs"])
	assert.Equal(t, "git push", data.Aliases["gp"])
	assert.Equal(t, "pwd", data.Aliases["pwd"])
}

func TestGoshrcService_Load_WithEnvironment(t *testing.T) {
	// Arrange
	tmpDir := t.TempDir()
	goshrcPath := filepath.Join(tmpDir, ".goshrc")

	content := `# Environment
export PATH='/usr/local/bin'
export HOME="/home/user"
`
	err := os.WriteFile(goshrcPath, []byte(content), 0o644)
	require.NoError(t, err)

	service := NewGoshrcService(goshrcPath)

	// Act
	data, err := service.Load()

	// Assert
	require.NoError(t, err)
	assert.Len(t, data.Environment, 2)
	assert.Equal(t, "/usr/local/bin", data.Environment["PATH"])
	assert.Equal(t, "/home/user", data.Environment["HOME"])
}

func TestGoshrcService_Load_WithCommentsAndEmptyLines(t *testing.T) {
	// Arrange
	tmpDir := t.TempDir()
	goshrcPath := filepath.Join(tmpDir, ".goshrc")

	content := `# .goshrc - GoSh configuration
# This is a comment

alias ll='ls -la'

# Another comment
alias gs='git status'

`
	err := os.WriteFile(goshrcPath, []byte(content), 0o644)
	require.NoError(t, err)

	service := NewGoshrcService(goshrcPath)

	// Act
	data, err := service.Load()

	// Assert
	require.NoError(t, err)
	assert.Len(t, data.Aliases, 2)
	assert.Equal(t, "ls -la", data.Aliases["ll"])
	assert.Equal(t, "git status", data.Aliases["gs"])
}

func TestGoshrcService_Save_NewFile(t *testing.T) {
	// Arrange
	tmpDir := t.TempDir()
	goshrcPath := filepath.Join(tmpDir, ".goshrc")
	service := NewGoshrcService(goshrcPath)

	data := &GoshrcData{
		Aliases: map[string]string{
			"ll": "ls -la",
			"gs": "git status",
		},
		Environment: map[string]string{
			"PATH": "/usr/local/bin",
		},
	}

	// Act
	err := service.Save(data)

	// Assert
	require.NoError(t, err)

	// Check that file was created
	_, err = os.Stat(goshrcPath)
	assert.NoError(t, err)

	// Check content
	content, err := os.ReadFile(goshrcPath)
	require.NoError(t, err)

	contentStr := string(content)
	assert.Contains(t, contentStr, "alias ll='ls -la'")
	assert.Contains(t, contentStr, "alias gs='git status'")
	assert.Contains(t, contentStr, "export PATH='/usr/local/bin'")
}

func TestGoshrcService_SaveAndLoad_RoundTrip(t *testing.T) {
	// Arrange
	tmpDir := t.TempDir()
	goshrcPath := filepath.Join(tmpDir, ".goshrc")
	service := NewGoshrcService(goshrcPath)

	originalData := &GoshrcData{
		Aliases: map[string]string{
			"ll": "ls -la",
			"gs": "git status",
			"gp": "git push",
		},
		Environment: map[string]string{
			"PATH": "/usr/local/bin",
			"HOME": "/home/user",
		},
	}

	// Act - Save
	err := service.Save(originalData)
	require.NoError(t, err)

	// Act - Load
	loadedData, err := service.Load()

	// Assert
	require.NoError(t, err)
	assert.Equal(t, originalData.Aliases, loadedData.Aliases)
	assert.Equal(t, originalData.Environment, loadedData.Environment)
}

func TestGoshrcService_Save_OverwriteExisting(t *testing.T) {
	// Arrange
	tmpDir := t.TempDir()
	goshrcPath := filepath.Join(tmpDir, ".goshrc")
	service := NewGoshrcService(goshrcPath)

	// Create initial file
	initialData := &GoshrcData{
		Aliases: map[string]string{
			"ll": "ls -la",
		},
	}
	err := service.Save(initialData)
	require.NoError(t, err)

	// Act - Overwrite
	newData := &GoshrcData{
		Aliases: map[string]string{
			"gs": "git status",
		},
	}
	err = service.Save(newData)
	require.NoError(t, err)

	// Assert - Check that old data was replaced
	loadedData, err := service.Load()
	require.NoError(t, err)
	assert.Len(t, loadedData.Aliases, 1)
	assert.Equal(t, "git status", loadedData.Aliases["gs"])
	assert.NotContains(t, loadedData.Aliases, "ll")
}

func TestGoshrcService_Load_MalformedLines(t *testing.T) {
	// Arrange
	tmpDir := t.TempDir()
	goshrcPath := filepath.Join(tmpDir, ".goshrc")

	content := `# Valid alias
alias ll='ls -la'

# Malformed lines (should be ignored gracefully)
this is not a valid line
alias incomplete
export
random text

# Another valid alias
alias gs='git status'
`
	err := os.WriteFile(goshrcPath, []byte(content), 0o644)
	require.NoError(t, err)

	service := NewGoshrcService(goshrcPath)

	// Act
	data, err := service.Load()

	// Assert - Should not error, just ignore malformed lines
	require.NoError(t, err)
	assert.Len(t, data.Aliases, 2)
	assert.Equal(t, "ls -la", data.Aliases["ll"])
	assert.Equal(t, "git status", data.Aliases["gs"])
}

func TestGetDefaultGoshrcPath(t *testing.T) {
	// Act
	path, err := GetDefaultGoshrcPath()

	// Assert
	require.NoError(t, err)
	assert.NotEmpty(t, path)
	assert.Contains(t, path, ".goshrc")
}
