package repl

import (
	"context"
	"log/slog"
	"os"
	"testing"

	"github.com/grpmsoft/gosh/internal/application/execute"
	appsession "github.com/grpmsoft/gosh/internal/application/session"
	"github.com/grpmsoft/gosh/internal/domain/config"
	"github.com/grpmsoft/gosh/internal/infrastructure/builtin"
	"github.com/grpmsoft/gosh/internal/infrastructure/executor"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewBubbleteaREPL(t *testing.T) {
	t.Run("creates REPL with default configuration", func(t *testing.T) {
		// Arrange
		logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
		cfg := config.DefaultConfig()
		ctx := context.Background()

		// Create dependencies
		fs := &mockFileSystem{}
		builtinExecutor := builtin.NewExecutor(fs, logger)
		commandExecutor := executor.NewOSCommandExecutor(logger)
		pipelineExecutor := executor.NewOSPipelineExecutor(logger)
		executeUseCase := execute.NewUseCase(
			builtinExecutor,
			commandExecutor,
			pipelineExecutor,
			logger,
		)
		sessionManager := appsession.NewManager(logger)

		// Act
		model, err := NewBubbleteaREPL(sessionManager, executeUseCase, logger, ctx, cfg)

		// Assert
		require.NoError(t, err)
		require.NotNil(t, model)

		// Verify model initialization
		assert.NotNil(t, model.currentSession)
		assert.NotNil(t, model.executeUseCase)
		assert.NotNil(t, model.logger)
		assert.NotNil(t, model.ctx)
		assert.NotNil(t, model.config)
		assert.NotNil(t, model.historyNavigator)
		assert.NotNil(t, model.historyRepo)
		assert.NotNil(t, model.addToHistoryUC)

		// Verify default values
		assert.Equal(t, 10000, model.maxOutputLines)
		assert.False(t, model.ready)
		assert.False(t, model.quitting)
		assert.False(t, model.executing)
		assert.Equal(t, 0, model.lastExitCode)
		assert.True(t, model.autoScroll)
		assert.False(t, model.showingHelp)
		assert.Equal(t, -1, model.completionIndex)
		assert.False(t, model.completionActive)

		// Verify output buffer is initialized
		assert.NotNil(t, model.output)
		assert.GreaterOrEqual(t, len(model.output), 0)

		// Verify styles are initialized
		assert.NotNil(t, model.styles)
	})

	t.Run("creates REPL with Classic UI mode", func(t *testing.T) {
		// Arrange
		logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
		cfg := config.DefaultConfig()
		cfg.UI.Mode = config.UIModeClassic
		ctx := context.Background()

		fs := &mockFileSystem{}
		builtinExecutor := builtin.NewExecutor(fs, logger)
		commandExecutor := executor.NewOSCommandExecutor(logger)
		pipelineExecutor := executor.NewOSPipelineExecutor(logger)
		executeUseCase := execute.NewUseCase(
			builtinExecutor,
			commandExecutor,
			pipelineExecutor,
			logger,
		)
		sessionManager := appsession.NewManager(logger)

		// Act
		model, err := NewBubbleteaREPL(sessionManager, executeUseCase, logger, ctx, cfg)

		// Assert
		require.NoError(t, err)
		require.NotNil(t, model)
		assert.Equal(t, config.UIModeClassic, model.config.UI.Mode)

		// In Classic mode, welcome messages are printed to stdout, not added to output buffer
		// So output should be empty or very small
		assert.LessOrEqual(t, len(model.output), 1)
	})

	t.Run("creates REPL with Warp UI mode", func(t *testing.T) {
		// Arrange
		logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
		cfg := config.DefaultConfig()
		cfg.UI.Mode = config.UIModeWarp
		ctx := context.Background()

		fs := &mockFileSystem{}
		builtinExecutor := builtin.NewExecutor(fs, logger)
		commandExecutor := executor.NewOSCommandExecutor(logger)
		pipelineExecutor := executor.NewOSPipelineExecutor(logger)
		executeUseCase := execute.NewUseCase(
			builtinExecutor,
			commandExecutor,
			pipelineExecutor,
			logger,
		)
		sessionManager := appsession.NewManager(logger)

		// Act
		model, err := NewBubbleteaREPL(sessionManager, executeUseCase, logger, ctx, cfg)

		// Assert
		require.NoError(t, err)
		require.NotNil(t, model)
		assert.Equal(t, config.UIModeWarp, model.config.UI.Mode)

		// In non-Classic modes, welcome messages are added to output buffer
		assert.Greater(t, len(model.output), 0)
	})

	t.Run("initializes viewport correctly", func(t *testing.T) {
		// Arrange
		logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
		cfg := config.DefaultConfig()
		ctx := context.Background()

		fs := &mockFileSystem{}
		builtinExecutor := builtin.NewExecutor(fs, logger)
		commandExecutor := executor.NewOSCommandExecutor(logger)
		pipelineExecutor := executor.NewOSPipelineExecutor(logger)
		executeUseCase := execute.NewUseCase(
			builtinExecutor,
			commandExecutor,
			pipelineExecutor,
			logger,
		)
		sessionManager := appsession.NewManager(logger)

		// Act
		model, err := NewBubbleteaREPL(sessionManager, executeUseCase, logger, ctx, cfg)

		// Assert
		require.NoError(t, err)
		require.NotNil(t, model)

		// Verify viewport is initialized
		assert.NotNil(t, model.viewport)
		// Viewport should have mouse wheel enabled
		// Up/Down keys should be disabled (used for history navigation)
	})

	t.Run("initializes shell input correctly", func(t *testing.T) {
		// Arrange
		logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
		cfg := config.DefaultConfig()
		ctx := context.Background()

		fs := &mockFileSystem{}
		builtinExecutor := builtin.NewExecutor(fs, logger)
		commandExecutor := executor.NewOSCommandExecutor(logger)
		pipelineExecutor := executor.NewOSPipelineExecutor(logger)
		executeUseCase := execute.NewUseCase(
			builtinExecutor,
			commandExecutor,
			pipelineExecutor,
			logger,
		)
		sessionManager := appsession.NewManager(logger)

		// Act
		model, err := NewBubbleteaREPL(sessionManager, executeUseCase, logger, ctx, cfg)

		// Assert
		require.NoError(t, err)
		require.NotNil(t, model)

		// Verify shell input is initialized
		assert.NotNil(t, model.shellInput)
		assert.Empty(t, model.shellInput.Value())
	})

	t.Run("loads history from file", func(t *testing.T) {
		// Arrange
		logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
		cfg := config.DefaultConfig()
		ctx := context.Background()

		fs := &mockFileSystem{}
		builtinExecutor := builtin.NewExecutor(fs, logger)
		commandExecutor := executor.NewOSCommandExecutor(logger)
		pipelineExecutor := executor.NewOSPipelineExecutor(logger)
		executeUseCase := execute.NewUseCase(
			builtinExecutor,
			commandExecutor,
			pipelineExecutor,
			logger,
		)
		sessionManager := appsession.NewManager(logger)

		// Act
		model, err := NewBubbleteaREPL(sessionManager, executeUseCase, logger, ctx, cfg)

		// Assert
		require.NoError(t, err)
		require.NotNil(t, model)

		// Verify history components are initialized
		assert.NotNil(t, model.historyRepo)
		assert.NotNil(t, model.historyNavigator)
		assert.NotNil(t, model.addToHistoryUC)

		// History should be empty or loaded from file
		// We don't assert specific size as it depends on existing history file
		assert.NotNil(t, model.currentSession.History())
	})
}

func TestModelInit(t *testing.T) {
	t.Run("Init can be called successfully", func(t *testing.T) {
		// Arrange
		logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
		cfg := config.DefaultConfig()
		ctx := context.Background()

		fs := &mockFileSystem{}
		builtinExecutor := builtin.NewExecutor(fs, logger)
		commandExecutor := executor.NewOSCommandExecutor(logger)
		pipelineExecutor := executor.NewOSPipelineExecutor(logger)
		executeUseCase := execute.NewUseCase(
			builtinExecutor,
			commandExecutor,
			pipelineExecutor,
			logger,
		)
		sessionManager := appsession.NewManager(logger)

		model, err := NewBubbleteaREPL(sessionManager, executeUseCase, logger, ctx, cfg)
		require.NoError(t, err)

		// Act
		cmd := model.Init()

		// Assert
		// Phoenix ShellInput handles cursor blinking internally, so cmd can be nil
		// This is different from Bubbles textarea which returned a Blink command
		// Just verify Init() doesn't panic
		_ = cmd // cmd may be nil with Phoenix
	})
}

func TestGetHistoryFilePath(t *testing.T) {
	t.Run("returns history file path in user home directory", func(t *testing.T) {
		// Act
		path := getHistoryFilePath()

		// Assert
		assert.NotEmpty(t, path)
		assert.Contains(t, path, ".gosh_history")

		// Should contain home directory or fallback to /tmp
		home, err := os.UserHomeDir()
		if err == nil {
			assert.Contains(t, path, home)
		} else {
			assert.Contains(t, path, "/tmp")
		}
	})
}

func TestCommandExecutedMsg(t *testing.T) {
	t.Run("commandExecutedMsg struct holds command result", func(t *testing.T) {
		// Arrange & Act
		msg := commandExecutedMsg{
			output:   "test output",
			err:      nil,
			exitCode: 0,
		}

		// Assert
		assert.Equal(t, "test output", msg.output)
		assert.NoError(t, msg.err)
		assert.Equal(t, 0, msg.exitCode)
	})

	t.Run("commandExecutedMsg can hold error", func(t *testing.T) {
		// Arrange & Act
		testErr := assert.AnError
		msg := commandExecutedMsg{
			output:   "",
			err:      testErr,
			exitCode: 1,
		}

		// Assert
		assert.Empty(t, msg.output)
		assert.Error(t, msg.err)
		assert.Equal(t, 1, msg.exitCode)
	})
}
