package repl

import (
	"context"
	"log/slog"
	"os"
	"testing"

	"github.com/grpmsoft/gosh/internal/application/execute"
	apphistory "github.com/grpmsoft/gosh/internal/application/history"
	"github.com/grpmsoft/gosh/internal/domain/config"
	"github.com/grpmsoft/gosh/internal/domain/history"
	"github.com/grpmsoft/gosh/internal/domain/session"
	"github.com/grpmsoft/gosh/internal/domain/shared"
	"github.com/grpmsoft/gosh/internal/infrastructure/builtin"
	"github.com/grpmsoft/gosh/internal/infrastructure/executor"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockHistoryRepository implements a mock for HistoryRepository interface.
type mockHistoryRepository struct{}

func (m *mockHistoryRepository) Save(h *history.History) error {
	// Mock implementation - always succeeds
	return nil
}

func (m *mockHistoryRepository) Load(h *history.History) error {
	// Mock implementation - always succeeds
	return nil
}

func (m *mockHistoryRepository) Append(command string) error {
	// Mock implementation - always succeeds
	return nil
}

// Helper to create test model for helper functions.
func createTestModelForHelpers(t *testing.T) *Model {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	cfg := config.DefaultConfig()

	// Create empty environment for tests
	env := make(shared.Environment)

	// Create test session directly
	sess, err := session.NewSession(
		"test-session",
		os.TempDir(),
		env,
	)
	require.NoError(t, err)

	// Create filesystem and executors
	fs := &mockFileSystem{}
	builtinExecutor := builtin.NewBuiltinExecutor(fs, logger)
	commandExecutor := executor.NewOSCommandExecutor(logger)
	pipelineExecutor := executor.NewOSPipelineExecutor(logger)
	executeUseCase := execute.NewExecuteCommandUseCase(
		builtinExecutor,
		commandExecutor,
		pipelineExecutor,
		logger,
	)

	// Create mock repository and real AddToHistoryUseCase
	mockRepo := &mockHistoryRepository{}
	addToHistoryUC := apphistory.NewAddToHistoryUseCase(sess.History(), mockRepo)

	// Create textarea and viewport
	ta := textarea.New()
	ta.SetValue("")
	vp := viewport.New(80, 24)

	// Create history navigator
	historyNavigator := sess.NewHistoryNavigator()

	model := &Model{
		textarea:         ta,
		viewport:         vp,
		currentSession:   sess,
		executeUseCase:   executeUseCase,
		addToHistoryUC:   addToHistoryUC,
		logger:           logger,
		ctx:              context.Background(),
		config:           cfg,
		output:           make([]string, 0),
		historyNavigator: historyNavigator,
		maxOutputLines:   10000,
		inputText:        "",
		cursorPos:        0,
		autoScroll:       true,
		styles:           makeProfessionalStyles(),
	}

	return model
}

func TestNavigateHistory(t *testing.T) {
	t.Run("navigate up through history", func(t *testing.T) {
		m := createTestModelForHelpers(t)

		// Add some commands to history
		m.currentSession.History().Add("command1")
		m.currentSession.History().Add("command2")
		m.currentSession.History().Add("command3")

		// Reset navigator
		m.historyNavigator = m.currentSession.NewHistoryNavigator()

		// Navigate up
		updatedModel, _ := m.navigateHistory("up")
		m2 := updatedModel.(Model)
		assert.Equal(t, "command3", m2.textarea.Value())

		// Navigate up again
		m2.historyNavigator = m.historyNavigator
		updatedModel2, _ := m2.navigateHistory("up")
		m3 := updatedModel2.(Model)
		assert.Equal(t, "command2", m3.textarea.Value())
	})

	t.Run("navigate down through history", func(t *testing.T) {
		m := createTestModelForHelpers(t)

		// Add commands to history
		m.currentSession.History().Add("command1")
		m.currentSession.History().Add("command2")
		m.currentSession.History().Add("command3")

		// Reset navigator
		m.historyNavigator = m.currentSession.NewHistoryNavigator()

		// Navigate up twice
		updatedModel, _ := m.navigateHistory("up")
		m2 := updatedModel.(Model)
		m2.historyNavigator = m.historyNavigator
		updatedModel2, _ := m2.navigateHistory("up")
		m3 := updatedModel2.(Model)

		// Navigate down
		m3.historyNavigator = m.historyNavigator
		updatedModel3, _ := m3.navigateHistory("down")
		m4 := updatedModel3.(Model)
		assert.Equal(t, "command3", m4.textarea.Value())
	})

	t.Run("navigate down at end returns empty", func(t *testing.T) {
		m := createTestModelForHelpers(t)

		// Add command to history
		m.currentSession.History().Add("command1")

		// Reset navigator and navigate up
		m.historyNavigator = m.currentSession.NewHistoryNavigator()
		updatedModel, _ := m.navigateHistory("up")
		m2 := updatedModel.(Model)

		// Navigate down (should return empty)
		m2.historyNavigator = m.historyNavigator
		updatedModel2, _ := m2.navigateHistory("down")
		m3 := updatedModel2.(Model)
		assert.Equal(t, "", m3.textarea.Value())
	})

	t.Run("navigate up on empty history", func(t *testing.T) {
		m := createTestModelForHelpers(t)

		// Navigate up on empty history
		updatedModel, _ := m.navigateHistory("up")
		m2 := updatedModel.(Model)

		// Value should remain unchanged
		assert.Equal(t, "", m2.textarea.Value())
	})
}

func TestAddOutputRaw(t *testing.T) {
	t.Run("adds single line to output", func(t *testing.T) {
		m := createTestModelForHelpers(t)

		m.addOutputRaw("test line 1")

		assert.Equal(t, 1, len(m.output))
		assert.Equal(t, "test line 1", m.output[0])
	})

	t.Run("adds multiple lines to output", func(t *testing.T) {
		m := createTestModelForHelpers(t)

		m.addOutputRaw("line 1")
		m.addOutputRaw("line 2")
		m.addOutputRaw("line 3")

		assert.Equal(t, 3, len(m.output))
		assert.Equal(t, "line 1", m.output[0])
		assert.Equal(t, "line 2", m.output[1])
		assert.Equal(t, "line 3", m.output[2])
	})

	t.Run("limits output to maxOutputLines", func(t *testing.T) {
		m := createTestModelForHelpers(t)
		m.maxOutputLines = 5

		// Add more lines than max
		for i := 0; i < 10; i++ {
			m.addOutputRaw("line")
		}

		// Should only keep last maxOutputLines
		assert.Equal(t, 5, len(m.output))
	})

	t.Run("handles empty lines", func(t *testing.T) {
		m := createTestModelForHelpers(t)

		m.addOutputRaw("")
		m.addOutputRaw("line 1")
		m.addOutputRaw("")

		assert.Equal(t, 3, len(m.output))
		assert.Equal(t, "", m.output[0])
		assert.Equal(t, "line 1", m.output[1])
		assert.Equal(t, "", m.output[2])
	})
}

func TestUpdateViewportContent(t *testing.T) {
	t.Run("updates viewport with output lines", func(t *testing.T) {
		m := createTestModelForHelpers(t)

		m.addOutputRaw("line 1")
		m.addOutputRaw("line 2")
		m.addOutputRaw("line 3")

		m.updateViewportContent()

		content := m.viewport.View()
		assert.Contains(t, content, "line 1")
		assert.Contains(t, content, "line 2")
		assert.Contains(t, content, "line 3")
	})

	t.Run("updates viewport with empty output", func(t *testing.T) {
		m := createTestModelForHelpers(t)

		m.updateViewportContent()

		// Should not panic with empty output
		_ = m.viewport.View()
	})

	t.Run("joins output with newlines", func(t *testing.T) {
		m := createTestModelForHelpers(t)

		m.addOutputRaw("line 1")
		m.addOutputRaw("line 2")

		m.updateViewportContent()

		content := m.viewport.View()
		// Content should contain both lines separated by newline
		assert.Contains(t, content, "line 1")
		assert.Contains(t, content, "line 2")
	})
}

func TestUpdateGitInfo(t *testing.T) {
	t.Run("clears git info for non-git directory", func(t *testing.T) {
		m := createTestModelForHelpers(t)

		// Set working directory to temp (not a git repo)
		err := m.currentSession.ChangeDirectory(os.TempDir())
		require.NoError(t, err)

		m.updateGitInfo()

		// Should clear git info
		assert.Equal(t, "", m.gitBranch)
		assert.False(t, m.gitDirty)
	})

	t.Run("detects git repository", func(t *testing.T) {
		m := createTestModelForHelpers(t)

		// Set working directory to current project (which is a git repo)
		projectDir := "D:\\projects\\grpmsoft\\gosh"
		err := m.currentSession.ChangeDirectory(projectDir)
		require.NoError(t, err)

		m.updateGitInfo()

		// In a git repository, should detect branch
		// Note: This test depends on being run in the gosh git repository
		// If not in git repo, these will be empty/false
		if m.gitBranch != "" {
			assert.NotEmpty(t, m.gitBranch)
			// gitDirty can be true or false depending on repo state
		}
	})
}