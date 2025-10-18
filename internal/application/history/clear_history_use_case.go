package history

import "github.com/grpmsoft/gosh/internal/domain/history"

// ClearHistoryUseCase implements clearing history.
type ClearHistoryUseCase struct {
	repository Repository
}

// NewClearHistoryUseCase creates a new clear history use case.
func NewClearHistoryUseCase(repo Repository) *ClearHistoryUseCase {
	return &ClearHistoryUseCase{
		repository: repo,
	}
}

// Execute clears the history and persists the change.
func (uc *ClearHistoryUseCase) Execute(h *history.History) error {
	// Clear history in domain model
	h.Clear()

	// Persist the empty history
	return uc.repository.Save(h)
}
