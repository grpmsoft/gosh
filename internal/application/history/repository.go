package history

import "github.com/grpmsoft/gosh/internal/domain/history"

// Repository defines the interface for history persistence.
// This is a Port in Hexagonal Architecture (Ports & Adapters).
type Repository interface {
	// Save persists the history to storage
	Save(h *history.History) error

	// Load loads history from storage into the given History instance
	Load(h *history.History) error

	// Append adds a single command to the end of history file
	// More efficient than Save for incremental updates
	// Thread-safe with file locking for multi-terminal support
	Append(command string) error
}
