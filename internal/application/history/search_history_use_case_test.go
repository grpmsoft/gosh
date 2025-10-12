package history_test

import (
	"testing"

	apphistory "github.com/grpmsoft/gosh/internal/application/history"
	"github.com/grpmsoft/gosh/internal/domain/history"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockHistoryRepository mock for testing use case
type MockHistoryRepository struct {
	SaveCalled   bool
	LoadCalled   bool
	AppendCalled bool
	SaveError    error
	LoadError    error
	AppendError  error
	History      *history.History
}

func (m *MockHistoryRepository) Save(h *history.History) error {
	m.SaveCalled = true
	m.History = h
	return m.SaveError
}

func (m *MockHistoryRepository) Load(h *history.History) error {
	m.LoadCalled = true
	if m.History != nil {
		// Copy data from mock history
		h.FromSlice(m.History.ToSlice())
	}
	return m.LoadError
}

func (m *MockHistoryRepository) Append(command string) error {
	m.AppendCalled = true
	return m.AppendError
}

// TestSearchHistoryUseCase_Execute checks the main search scenario
func TestSearchHistoryUseCase_Execute(t *testing.T) {
	t.Run("searches and returns matching commands", func(t *testing.T) {
		// Preparation: create history with commands
		h := history.NewHistory(history.DefaultConfig())
		h.Add("git status")
		h.Add("git commit")
		h.Add("npm install")
		h.Add("git push")

		mockRepo := &MockHistoryRepository{History: h}

		useCase := apphistory.NewSearchHistoryUseCase(h, mockRepo)

		// Execute search
		results, err := useCase.Execute("git")
		require.NoError(t, err)

		// Check results
		assert.Len(t, results, 3)
		assert.Equal(t, "git push", results[0])
		assert.Equal(t, "git commit", results[1])
		assert.Equal(t, "git status", results[2])
	})

	t.Run("returns empty for no matches", func(t *testing.T) {
		h := history.NewHistory(history.DefaultConfig())
		h.Add("npm install")
		h.Add("npm test")

		mockRepo := &MockHistoryRepository{History: h}
		useCase := apphistory.NewSearchHistoryUseCase(h, mockRepo)

		results, err := useCase.Execute("git")
		require.NoError(t, err)
		assert.Empty(t, results)
	})

	t.Run("handles empty history", func(t *testing.T) {
		h := history.NewHistory(history.DefaultConfig())
		mockRepo := &MockHistoryRepository{History: h}
		useCase := apphistory.NewSearchHistoryUseCase(h, mockRepo)

		results, err := useCase.Execute("test")
		require.NoError(t, err)
		assert.Empty(t, results)
	})
}

// TestAddToHistoryUseCase_Execute checks adding a command to history
func TestAddToHistoryUseCase_Execute(t *testing.T) {
	t.Run("adds command and persists to repository", func(t *testing.T) {
		h := history.NewHistory(history.DefaultConfig())
		mockRepo := &MockHistoryRepository{History: h}

		useCase := apphistory.NewAddToHistoryUseCase(h, mockRepo)

		err := useCase.Execute("git status")
		require.NoError(t, err)

		// Check that command was added
		assert.Equal(t, 1, h.Size())
		assert.Equal(t, "git status", h.GetRecent(1)[0])

		// Check that repository was called for adding (Append, not Save)
		assert.True(t, mockRepo.AppendCalled)
	})

	t.Run("rejects empty commands", func(t *testing.T) {
		h := history.NewHistory(history.DefaultConfig())
		mockRepo := &MockHistoryRepository{History: h}
		useCase := apphistory.NewAddToHistoryUseCase(h, mockRepo)

		err := useCase.Execute("")
		assert.Error(t, err)
		assert.Equal(t, 0, h.Size())

		// Append should not be called for empty commands
		assert.False(t, mockRepo.AppendCalled)
	})

	t.Run("auto-persists after each add", func(t *testing.T) {
		h := history.NewHistory(history.Config{
			MaxSize:    1000,
			SaveToFile: true, // Auto-save enabled
		})
		mockRepo := &MockHistoryRepository{History: h}
		useCase := apphistory.NewAddToHistoryUseCase(h, mockRepo)

		useCase.Execute("cmd1")
		assert.True(t, mockRepo.AppendCalled)

		mockRepo.AppendCalled = false
		useCase.Execute("cmd2")
		assert.True(t, mockRepo.AppendCalled)
	})

	t.Run("skips persistence if SaveToFile is false", func(t *testing.T) {
		h := history.NewHistory(history.Config{
			MaxSize:    1000,
			SaveToFile: false, // Auto-save disabled
		})
		mockRepo := &MockHistoryRepository{History: h}
		useCase := apphistory.NewAddToHistoryUseCase(h, mockRepo)

		useCase.Execute("cmd1")
		assert.False(t, mockRepo.AppendCalled)
	})
}

// TestLoadHistoryUseCase_Execute checks loading history on startup
func TestLoadHistoryUseCase_Execute(t *testing.T) {
	t.Run("loads history from repository", func(t *testing.T) {
		// Preparation: create saved history
		savedHistory := history.NewHistory(history.DefaultConfig())
		savedHistory.Add("cmd1")
		savedHistory.Add("cmd2")
		savedHistory.Add("cmd3")

		mockRepo := &MockHistoryRepository{History: savedHistory}

		// Create empty history for loading
		h := history.NewHistory(history.DefaultConfig())
		useCase := apphistory.NewLoadHistoryUseCase(mockRepo)

		err := useCase.Execute(h)
		require.NoError(t, err)

		// Check that history was loaded
		assert.Equal(t, 3, h.Size())
		assert.Equal(t, savedHistory.ToSlice(), h.ToSlice())

		// Check that repository was called
		assert.True(t, mockRepo.LoadCalled)
	})

	t.Run("handles empty repository", func(t *testing.T) {
		emptyHistory := history.NewHistory(history.DefaultConfig())
		mockRepo := &MockHistoryRepository{History: emptyHistory}

		h := history.NewHistory(history.DefaultConfig())
		useCase := apphistory.NewLoadHistoryUseCase(mockRepo)

		err := useCase.Execute(h)
		require.NoError(t, err)
		assert.Equal(t, 0, h.Size())
	})
}

// TestClearHistoryUseCase_Execute checks clearing history
func TestClearHistoryUseCase_Execute(t *testing.T) {
	t.Run("clears history and persists", func(t *testing.T) {
		h := history.NewHistory(history.DefaultConfig())
		h.Add("cmd1")
		h.Add("cmd2")
		h.Add("cmd3")

		mockRepo := &MockHistoryRepository{History: h}
		useCase := apphistory.NewClearHistoryUseCase(mockRepo)

		err := useCase.Execute(h)
		require.NoError(t, err)

		// Check that history was cleared
		assert.Equal(t, 0, h.Size())

		// Check that changes were saved
		assert.True(t, mockRepo.SaveCalled)
	})
}

// TestHistoryUseCaseIntegration checks interaction between use cases
func TestHistoryUseCaseIntegration(t *testing.T) {
	t.Run("full workflow: add, search, clear", func(t *testing.T) {
		h := history.NewHistory(history.Config{
			MaxSize:    100,
			SaveToFile: true,
		})
		mockRepo := &MockHistoryRepository{History: h}

		addUseCase := apphistory.NewAddToHistoryUseCase(h, mockRepo)
		searchUseCase := apphistory.NewSearchHistoryUseCase(h, mockRepo)
		clearUseCase := apphistory.NewClearHistoryUseCase(mockRepo)

		// 1. Add commands
		addUseCase.Execute("git init")
		addUseCase.Execute("git add .")
		addUseCase.Execute("git commit")
		addUseCase.Execute("npm install")

		// 2. Search commands
		results, err := searchUseCase.Execute("git")
		require.NoError(t, err)
		assert.Len(t, results, 3)

		// 3. Clear history
		err = clearUseCase.Execute(h)
		require.NoError(t, err)

		// 4. Check that history is empty
		results, err = searchUseCase.Execute("git")
		require.NoError(t, err)
		assert.Empty(t, results)
	})
}
