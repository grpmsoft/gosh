// Package main provides the entry point and dependency injection for the GoSh shell application.
package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/grpmsoft/gosh/internal/application/execute"
	appsession "github.com/grpmsoft/gosh/internal/application/session"
	"github.com/grpmsoft/gosh/internal/domain/config"
	"github.com/grpmsoft/gosh/internal/infrastructure/builtin"
	configLoader "github.com/grpmsoft/gosh/internal/infrastructure/config"
	"github.com/grpmsoft/gosh/internal/infrastructure/executor"
	"github.com/grpmsoft/gosh/internal/infrastructure/filesystem"
	"github.com/grpmsoft/gosh/internal/interfaces/repl"
)

// setupLogger sets up the logger for writing to a file.
func setupLogger() *slog.Logger {
	logFile, err := os.OpenFile("gosh.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		logFile = os.Stderr // Fallback
	}

	return slog.New(slog.NewTextHandler(logFile, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
}

// bootstrapREPL creates and configures REPL with dependencies.
// modeOverride overrides UI mode from --mode CLI flag (empty = use config default).
func bootstrapREPL(logger *slog.Logger, ctx context.Context, modeOverride string) (*repl.Model, error) {
	// Load configuration
	loader := configLoader.NewLoader()
	cfg, err := loader.Load()
	if err != nil {
		logger.Warn("Failed to load config, using defaults", "error", err)
	}

	// Override UI mode from CLI flag (before creating model — affects welcome messages)
	if modeOverride != "" {
		switch modeOverride {
		case "classic":
			cfg.UI.Mode = config.UIModeClassic
		case "warp":
			cfg.UI.Mode = config.UIModeWarp
		case "compact":
			cfg.UI.Mode = config.UIModeCompact
		case "chat":
			cfg.UI.Mode = config.UIModeChat
		}
	}

	// Create dependencies (Dependency Injection)
	fs := filesystem.NewOSFileSystem()
	builtinExec := builtin.NewExecutor(fs, logger)
	commandExec := executor.NewOSCommandExecutor(logger)
	pipelineExec := executor.NewOSPipelineExecutor(logger)

	// Create use cases
	sessionManager := appsession.NewManager(logger)
	executeUseCase := execute.NewUseCase(
		builtinExec,
		commandExec,
		pipelineExec,
		logger,
	)

	// Create REPL with configuration
	return repl.NewBubbleteaREPL(sessionManager, executeUseCase, logger, ctx, cfg)
}

// bootstrapNonInteractive creates dependencies for non-interactive mode (-c flag).
func bootstrapNonInteractive(logger *slog.Logger) (*appsession.Manager, *execute.UseCase) {
	// Create dependencies (Dependency Injection)
	fs := filesystem.NewOSFileSystem()
	builtinExec := builtin.NewExecutor(fs, logger)
	commandExec := executor.NewOSCommandExecutor(logger)
	pipelineExec := executor.NewOSPipelineExecutor(logger)

	// Create use cases
	sessionManager := appsession.NewManager(logger)
	executeUseCase := execute.NewUseCase(
		builtinExec,
		commandExec,
		pipelineExec,
		logger,
	)

	return sessionManager, executeUseCase
}
