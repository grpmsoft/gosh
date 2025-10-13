package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/grpmsoft/gosh/internal/application/execute"
	appsession "github.com/grpmsoft/gosh/internal/application/session"
	"github.com/grpmsoft/gosh/internal/infrastructure/builtin"
	configLoader "github.com/grpmsoft/gosh/internal/infrastructure/config"
	"github.com/grpmsoft/gosh/internal/infrastructure/executor"
	"github.com/grpmsoft/gosh/internal/infrastructure/filesystem"
	"github.com/grpmsoft/gosh/internal/interfaces/repl"
)

// setupLogger sets up the logger for writing to a file
func setupLogger() *slog.Logger {
	logFile, err := os.OpenFile("gosh.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		logFile = os.Stderr // Fallback
	}

	return slog.New(slog.NewTextHandler(logFile, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
}

// bootstrapREPL creates and configures REPL with dependencies
func bootstrapREPL(logger *slog.Logger, ctx context.Context) (*repl.Model, error) {
	// Load configuration
	loader := configLoader.NewLoader()
	cfg, err := loader.Load()
	if err != nil {
		logger.Warn("Failed to load config, using defaults", "error", err)
	}

	// Create dependencies (Dependency Injection)
	fs := filesystem.NewOSFileSystem()
	builtinExec := builtin.NewBuiltinExecutor(fs, logger)
	commandExec := executor.NewOSCommandExecutor(logger)
	pipelineExec := executor.NewOSPipelineExecutor(logger)

	// Create use cases
	sessionManager := appsession.NewSessionManager(logger)
	executeUseCase := execute.NewExecuteCommandUseCase(
		builtinExec,
		commandExec,
		pipelineExec,
		logger,
	)

	// Create REPL with configuration
	return repl.NewBubbleteaREPL(sessionManager, executeUseCase, logger, ctx, cfg)
}

// bootstrapNonInteractive creates dependencies for non-interactive mode (-c flag)
func bootstrapNonInteractive(logger *slog.Logger) (*appsession.SessionManager, *execute.ExecuteCommandUseCase, error) {
	// Create dependencies (Dependency Injection)
	fs := filesystem.NewOSFileSystem()
	builtinExec := builtin.NewBuiltinExecutor(fs, logger)
	commandExec := executor.NewOSCommandExecutor(logger)
	pipelineExec := executor.NewOSPipelineExecutor(logger)

	// Create use cases
	sessionManager := appsession.NewSessionManager(logger)
	executeUseCase := execute.NewExecuteCommandUseCase(
		builtinExec,
		commandExec,
		pipelineExec,
		logger,
	)

	return sessionManager, executeUseCase, nil
}
