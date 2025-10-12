package parser

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestExpandGlobs_NoPattern tests that non-glob arguments pass through unchanged
func TestExpandGlobs_NoPattern(t *testing.T) {
	args := []string{"file.txt", "test", "hello"}
	result, err := expandGlobs(args)

	require.NoError(t, err)
	assert.Equal(t, args, result)
}

// TestExpandGlobs_Star tests * pattern expansion
func TestExpandGlobs_Star(t *testing.T) {
	// Create temp directory with test files
	tmpDir := t.TempDir()

	// Create test files
	files := []string{"test1.go", "test2.go", "test3.txt"}
	for _, file := range files {
		path := filepath.Join(tmpDir, file)
		err := os.WriteFile(path, []byte("test"), 0644)
		require.NoError(t, err)
	}

	// Change to temp directory for test
	oldWd, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(oldWd)

	err = os.Chdir(tmpDir)
	require.NoError(t, err)

	t.Run("*.go matches .go files", func(t *testing.T) {
		args := []string{"*.go"}
		result, err := expandGlobs(args)

		require.NoError(t, err)
		assert.Len(t, result, 2)
		assert.Contains(t, result, "test1.go")
		assert.Contains(t, result, "test2.go")
	})

	t.Run("*.txt matches .txt files", func(t *testing.T) {
		args := []string{"*.txt"}
		result, err := expandGlobs(args)

		require.NoError(t, err)
		assert.Len(t, result, 1)
		assert.Contains(t, result, "test3.txt")
	})

	t.Run("test* matches all test files", func(t *testing.T) {
		args := []string{"test*"}
		result, err := expandGlobs(args)

		require.NoError(t, err)
		assert.Len(t, result, 3)
	})
}

// TestExpandGlobs_Question tests ? pattern expansion
func TestExpandGlobs_Question(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test files
	files := []string{"test1.txt", "test2.txt", "test10.txt"}
	for _, file := range files {
		path := filepath.Join(tmpDir, file)
		err := os.WriteFile(path, []byte("test"), 0644)
		require.NoError(t, err)
	}

	oldWd, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(oldWd)

	err = os.Chdir(tmpDir)
	require.NoError(t, err)

	t.Run("test?.txt matches single digit", func(t *testing.T) {
		args := []string{"test?.txt"}
		result, err := expandGlobs(args)

		require.NoError(t, err)
		assert.Len(t, result, 2)
		assert.Contains(t, result, "test1.txt")
		assert.Contains(t, result, "test2.txt")
		assert.NotContains(t, result, "test10.txt")
	})
}

// TestExpandGlobs_Brackets tests [] pattern expansion
func TestExpandGlobs_Brackets(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test files
	files := []string{"file1.txt", "file2.txt", "file3.txt", "file4.txt"}
	for _, file := range files {
		path := filepath.Join(tmpDir, file)
		err := os.WriteFile(path, []byte("test"), 0644)
		require.NoError(t, err)
	}

	oldWd, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(oldWd)

	err = os.Chdir(tmpDir)
	require.NoError(t, err)

	t.Run("file[12].txt matches files 1 and 2", func(t *testing.T) {
		args := []string{"file[12].txt"}
		result, err := expandGlobs(args)

		require.NoError(t, err)
		assert.Len(t, result, 2)
		assert.Contains(t, result, "file1.txt")
		assert.Contains(t, result, "file2.txt")
		assert.NotContains(t, result, "file3.txt")
	})

	t.Run("file[1-3].txt matches range", func(t *testing.T) {
		args := []string{"file[1-3].txt"}
		result, err := expandGlobs(args)

		require.NoError(t, err)
		assert.Len(t, result, 3)
		assert.Contains(t, result, "file1.txt")
		assert.Contains(t, result, "file2.txt")
		assert.Contains(t, result, "file3.txt")
		assert.NotContains(t, result, "file4.txt")
	})
}

// TestExpandGlobs_NoMatches tests bash-like behavior: error on no matches
func TestExpandGlobs_NoMatches(t *testing.T) {
	tmpDir := t.TempDir()

	oldWd, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(oldWd)

	err = os.Chdir(tmpDir)
	require.NoError(t, err)

	t.Run("no matches returns error", func(t *testing.T) {
		args := []string{"*.nonexistent"}
		result, err := expandGlobs(args)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "no matches found")
	})
}

// TestExpandGlobs_Mixed tests mixing glob and non-glob arguments
func TestExpandGlobs_Mixed(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test files
	files := []string{"test.go", "main.go"}
	for _, file := range files {
		path := filepath.Join(tmpDir, file)
		err := os.WriteFile(path, []byte("test"), 0644)
		require.NoError(t, err)
	}

	oldWd, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(oldWd)

	err = os.Chdir(tmpDir)
	require.NoError(t, err)

	t.Run("mix glob and literal arguments", func(t *testing.T) {
		args := []string{"echo", "*.go", "hello"}
		result, err := expandGlobs(args)

		require.NoError(t, err)
		assert.Greater(t, len(result), 2) // echo + 2 .go files + hello
		assert.Equal(t, "echo", result[0])
		assert.Equal(t, "hello", result[len(result)-1])
	})
}

// TestContainsGlobPattern tests pattern detection
func TestContainsGlobPattern(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"*.go", true},
		{"test?.txt", true},
		{"file[123].txt", true},
		{"normal.txt", false},
		{"test", false},
		{"path/to/file", false},
		{"*.tar.gz", true},
		{"test[", true}, // incomplete bracket is still a pattern
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := containsGlobPattern(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestParseCommandLine_WithGlob tests full integration
func TestParseCommandLine_WithGlob(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test files
	files := []string{"file1.go", "file2.go"}
	for _, file := range files {
		path := filepath.Join(tmpDir, file)
		err := os.WriteFile(path, []byte("test"), 0644)
		require.NoError(t, err)
	}

	oldWd, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(oldWd)

	err = os.Chdir(tmpDir)
	require.NoError(t, err)

	t.Run("ls *.go expands to file list", func(t *testing.T) {
		cmd, pipe, err := ParseCommandLine("ls *.go")

		require.NoError(t, err)
		assert.Nil(t, pipe)
		require.NotNil(t, cmd)

		assert.Equal(t, "ls", cmd.Name())
		assert.Len(t, cmd.Args(), 2)
		assert.Contains(t, cmd.Args(), "file1.go")
		assert.Contains(t, cmd.Args(), "file2.go")
	})
}
