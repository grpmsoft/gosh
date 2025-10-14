// Package history provides domain models for command history management and navigation.
package history

import (
	"errors"
	"strings"
)

// Domain errors.
var (
	ErrEmptyCommand = errors.New("command cannot be empty")
)

// Config defines history configuration.
type Config struct {
	MaxSize          int  // Maximum number of commands to store
	SaveToFile       bool // Whether to persist history to file
	DeduplicateAdded bool // Skip consecutive duplicate commands
}

// DefaultConfig returns the default history configuration.
func DefaultConfig() Config {
	return Config{
		MaxSize:          10000,
		SaveToFile:       true,
		DeduplicateAdded: true,
	}
}

// History represents command history with rich domain model.
// This is an Aggregate Root in DDD terms.
type History struct {
	commands    []string // Commands in chronological order (oldest first)
	lastCommand string   // Last added command (for deduplication)
	config      Config   // Configuration
}

// NewHistory creates a new History instance with the given configuration.
func NewHistory(cfg Config) *History {
	return &History{
		commands:    make([]string, 0, cfg.MaxSize),
		lastCommand: "",
		config:      cfg,
	}
}

// Add adds a command to the history.
// Returns error if command is empty after trimming.
func (h *History) Add(cmd string) error {
	// Trim whitespace
	cmd = strings.TrimSpace(cmd)

	// Validate: reject empty commands
	if cmd == "" {
		return ErrEmptyCommand
	}

	// Deduplication: skip if same as last command
	if h.config.DeduplicateAdded && h.lastCommand == cmd {
		return nil
	}

	// Add to history
	h.commands = append(h.commands, cmd)
	h.lastCommand = cmd

	// Enforce max size limit
	if len(h.commands) > h.config.MaxSize {
		// Remove oldest commands to maintain size limit
		excess := len(h.commands) - h.config.MaxSize
		h.commands = h.commands[excess:]
	}

	return nil
}

// Size returns the number of commands in history.
func (h *History) Size() int {
	return len(h.commands)
}

// IsEmpty returns true if history is empty.
func (h *History) IsEmpty() bool {
	return len(h.commands) == 0
}

// MaxSize returns the maximum size from configuration.
func (h *History) MaxSize() int {
	return h.config.MaxSize
}

// Config returns the history configuration.
func (h *History) Config() Config {
	return h.config
}

// Search performs case-insensitive substring search in history.
// Returns matching commands in reverse chronological order (newest first).
// Limited to 50 results for UI performance.
func (h *History) Search(query string) []string {
	// Empty query returns empty result
	if strings.TrimSpace(query) == "" {
		return []string{}
	}

	// Case-insensitive search
	queryLower := strings.ToLower(query)

	results := make([]string, 0)

	// Search from newest to oldest
	for i := len(h.commands) - 1; i >= 0; i-- {
		cmd := h.commands[i]
		cmdLower := strings.ToLower(cmd)

		if strings.Contains(cmdLower, queryLower) {
			results = append(results, cmd)

			// Limit results to 50 for UI performance
			if len(results) >= 50 {
				break
			}
		}
	}

	return results
}

// GetRecent returns the N most recent commands in reverse chronological order (newest first).
// If n is greater than history size, returns all commands.
// If n <= 0, returns empty slice.
func (h *History) GetRecent(n int) []string {
	if n <= 0 {
		return []string{}
	}

	// Limit n to actual history size
	if n > len(h.commands) {
		n = len(h.commands)
	}

	// Return last n commands in reverse order
	result := make([]string, n)
	for i := 0; i < n; i++ {
		result[i] = h.commands[len(h.commands)-1-i]
	}

	return result
}

// Clear removes all commands from history.
func (h *History) Clear() {
	h.commands = make([]string, 0, h.config.MaxSize)
	h.lastCommand = ""
}

// ToSlice exports history as a slice in chronological order (oldest first).
// This is used for persistence.
func (h *History) ToSlice() []string {
	// Return copy to prevent external modification
	result := make([]string, len(h.commands))
	copy(result, h.commands)
	return result
}

// FromSlice loads history from a slice of commands in chronological order.
// This replaces existing history.
// Skips empty lines.
// Respects max size limit (takes last N commands if slice is larger).
func (h *History) FromSlice(lines []string) error {
	// Clear existing history
	h.commands = make([]string, 0, h.config.MaxSize)
	h.lastCommand = ""

	// Filter out empty lines
	filtered := make([]string, 0, len(lines))
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			filtered = append(filtered, trimmed)
		}
	}

	// If filtered slice is larger than max size, take last N commands
	startIdx := 0
	if len(filtered) > h.config.MaxSize {
		startIdx = len(filtered) - h.config.MaxSize
	}

	// Load commands
	h.commands = make([]string, 0, h.config.MaxSize)
	for i := startIdx; i < len(filtered); i++ {
		h.commands = append(h.commands, filtered[i])
	}

	// Update last command
	if len(h.commands) > 0 {
		h.lastCommand = h.commands[len(h.commands)-1]
	}

	return nil
}

// NewNavigator creates a new Navigator for Up/Down arrow navigation.
func (h *History) NewNavigator() *Navigator {
	return &Navigator{
		history:  h,
		position: -1, // Start at end (before newest command)
	}
}

// Navigator handles Up/Down arrow navigation through history.
// This is a separate entity to maintain navigation state independently of history.
type Navigator struct {
	history  *History // Reference to history
	position int      // Current position (-1 = at end, 0 = oldest, Size-1 = newest)
}

// Current returns the command at current position.
// Returns empty string if at end position.
func (n *Navigator) Current() string {
	if n.position < 0 || n.position >= len(n.history.commands) {
		return ""
	}
	return n.history.commands[n.position]
}

// Backward moves to older command (Up arrow).
// Returns (command, true) if moved, (current_or_oldest, false) if at beginning.
func (n *Navigator) Backward() (string, bool) {
	// If at end, move to newest
	if n.position < 0 {
		if len(n.history.commands) == 0 {
			return "", false
		}
		n.position = len(n.history.commands) - 1
		return n.history.commands[n.position], true
	}

	// Already at oldest
	if n.position == 0 {
		return n.history.commands[n.position], false
	}

	// Move backward
	n.position--
	return n.history.commands[n.position], true
}

// Forward moves to newer command (Down arrow).
// Returns (command, true) if moved.
// Returns ("", true) when reaching end position.
// Returns ("", false) when already past end.
func (n *Navigator) Forward() (string, bool) {
	// Already past end
	if n.position < -1 {
		return "", false
	}

	// At end position
	if n.position == -1 {
		return "", false
	}

	// Move forward
	n.position++

	// Reached end (beyond newest command)
	if n.position >= len(n.history.commands) {
		n.position = -1
		return "", true
	}

	return n.history.commands[n.position], true
}
