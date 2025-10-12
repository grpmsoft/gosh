package history

import "github.com/grpmsoft/gosh/internal/domain/history"

// SearchHistoryUseCase implements history search functionality (Ctrl+R)
type SearchHistoryUseCase struct {
	history    *history.History
	repository HistoryRepository
}

// NewSearchHistoryUseCase creates a new search use case
// Takes a History instance to work with (typically from Session)
func NewSearchHistoryUseCase(h *history.History, repo HistoryRepository) *SearchHistoryUseCase {
	return &SearchHistoryUseCase{
		history:    h,
		repository: repo,
	}
}

// Execute performs a search in the history
func (uc *SearchHistoryUseCase) Execute(query string) ([]string, error) {
	return uc.history.Search(query), nil
}
