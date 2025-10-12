package history

import "github.com/grpmsoft/gosh/internal/domain/history"

// LoadHistoryUseCase implements loading history from persistence
type LoadHistoryUseCase struct {
	repository HistoryRepository
}

// NewLoadHistoryUseCase creates a new load history use case
func NewLoadHistoryUseCase(repo HistoryRepository) *LoadHistoryUseCase {
	return &LoadHistoryUseCase{
		repository: repo,
	}
}

// Execute loads history from repository into the provided History instance
func (uc *LoadHistoryUseCase) Execute(h *history.History) error {
	return uc.repository.Load(h)
}
