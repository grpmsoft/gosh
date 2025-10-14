// Package history provides infrastructure for persisting command history to files.
package history

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/grpmsoft/gosh/internal/domain/history"
)

// FileHistoryRepository implements history persistence to a file.
// This is an Adapter in Hexagonal Architecture (Ports & Adapters).
type FileHistoryRepository struct {
	filePath string // Path to history file (e.g., ~/.gosh_history)
}

// NewFileHistoryRepository creates a new file-based history repository.
// Expands ~ to home directory if present in path.
func NewFileHistoryRepository(filePath string) *FileHistoryRepository {
	expandedPath := expandTilde(filePath)
	return &FileHistoryRepository{
		filePath: expandedPath,
	}
}

// FilePath returns the expanded file path.
func (r *FileHistoryRepository) FilePath() string {
	return r.filePath
}

// Save persists history to file.
// Creates parent directories if needed.
// Overwrites existing file.
// Commands are saved in chronological order (oldest first).
func (r *FileHistoryRepository) Save(h *history.History) error {
	// Create parent directories if they don't exist
	dir := filepath.Dir(r.filePath)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	// Create or truncate file
	file, err := os.Create(r.filePath)
	if err != nil {
		return fmt.Errorf("failed to create history file: %w", err)
	}
	defer func() { _ = file.Close() }()

	// Write commands in chronological order (oldest first)
	commands := h.ToSlice()
	for _, cmd := range commands {
		if _, err := fmt.Fprintln(file, cmd); err != nil {
			return fmt.Errorf("failed to write command to history: %w", err)
		}
	}

	return nil
}

// Load loads history from file into the provided History instance.
// Returns no error if file doesn't exist (empty history is OK).
// Skips empty lines and whitespace-only lines.
// Commands are loaded in chronological order (oldest first).
func (r *FileHistoryRepository) Load(h *history.History) error {
	// Check if file exists
	if _, err := os.Stat(r.filePath); os.IsNotExist(err) {
		// File doesn't exist - not an error, just empty history
		return nil
	}

	// Open file for reading
	file, err := os.Open(r.filePath)
	if err != nil {
		return fmt.Errorf("failed to open history file: %w", err)
	}
	defer func() { _ = file.Close() }()

	// Read lines from file
	lines := make([]string, 0)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		// Skip empty lines and whitespace-only lines
		if strings.TrimSpace(line) != "" {
			lines = append(lines, line)
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("failed to read history file: %w", err)
	}

	// Load into History domain model
	return h.FromSlice(lines)
}

// Append adds a single command to the end of history file.
// Uses O_APPEND for atomic writes (safe for concurrent access).
// More efficient than Save() for incremental updates.
// Creates file and parent directories if they don't exist.
func (r *FileHistoryRepository) Append(command string) error {
	// Skip empty commands
	if strings.TrimSpace(command) == "" {
		return nil
	}

	// Create parent directories if they don't exist
	dir := filepath.Dir(r.filePath)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	// Open file in append mode (O_APPEND ensures atomic writes)
	// O_CREATE creates file if it doesn't exist
	// O_WRONLY opens for writing only
	file, err := os.OpenFile(r.filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return fmt.Errorf("failed to open history file for append: %w", err)
	}
	defer func() { _ = file.Close() }()

	// Write command with newline
	// O_APPEND mode ensures this write is atomic on most systems
	if _, err := fmt.Fprintln(file, command); err != nil {
		return fmt.Errorf("failed to append command to history: %w", err)
	}

	// Sync to ensure data is written to disk
	// This is especially important on Windows to release file locks promptly
	if err := file.Sync(); err != nil {
		return fmt.Errorf("failed to sync history file: %w", err)
	}

	return nil
}

// expandTilde expands ~ to user's home directory.
func expandTilde(path string) string {
	if !strings.HasPrefix(path, "~") {
		return path
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		// If we can't get home dir, return path unchanged
		return path
	}

	if path == "~" {
		return homeDir
	}

	// Replace ~ with home directory
	return filepath.Join(homeDir, path[2:]) // Skip "~/"
}
