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

// setupLogger настраивает логгер для записи в файл
func setupLogger() *slog.Logger {
	logFile, err := os.OpenFile("gosh.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		logFile = os.Stderr // Fallback
	}

	return slog.New(slog.NewTextHandler(logFile, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
}

// bootstrapREPL создает и настраивает REPL с зависимостями
func bootstrapREPL(logger *slog.Logger, ctx context.Context) (*repl.Model, error) {
	// Загружаем конфигурацию
	loader := configLoader.NewLoader()
	cfg, err := loader.Load()
	if err != nil {
		logger.Warn("Failed to load config, using defaults", "error", err)
	}

	// Создаем зависимости (Dependency Injection)
	fs := filesystem.NewOSFileSystem()
	builtinExec := builtin.NewBuiltinExecutor(fs, logger)
	commandExec := executor.NewOSCommandExecutor(logger)
	pipelineExec := executor.NewOSPipelineExecutor(logger)

	// Создаем use cases
	sessionManager := appsession.NewSessionManager(logger)
	executeUseCase := execute.NewExecuteCommandUseCase(
		builtinExec,
		commandExec,
		pipelineExec,
		logger,
	)

	// Создаем REPL с конфигурацией
	return repl.NewBubbleteaREPL(sessionManager, executeUseCase, logger, ctx, cfg)
}
