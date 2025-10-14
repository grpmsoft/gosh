// Package history provides use cases for command history management.
package history

import "github.com/grpmsoft/gosh/internal/domain/history"

// AddToHistoryUseCase implements adding commands to history.
type AddToHistoryUseCase struct {
	history    *history.History
	repository Repository
}

// NewAddToHistoryUseCase creates a new add to history use case.
// Takes a History instance to work with (typically from Session).
func NewAddToHistoryUseCase(h *history.History, repo Repository) *AddToHistoryUseCase {
	return &AddToHistoryUseCase{
		history:    h,
		repository: repo,
	}
}

// Execute adds a command to history and persists if SaveToFile is enabled.
func (uc *AddToHistoryUseCase) Execute(cmd string) error {
	// Add command to domain model
	if err := uc.history.Add(cmd); err != nil {
		return err
	}

	// Persist only if SaveToFile is enabled in config
	if uc.history.Config().SaveToFile {
		// Use Append instead of Save for efficiency and multi-terminal safety
		// Append is atomic and won't overwrite other terminals' history
		return uc.repository.Append(cmd)
	}

	return nil
}
