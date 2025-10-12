package executor_test

import (
	"context"
	"testing"

	"github.com/grpmsoft/gosh/internal/domain/command"
	"github.com/grpmsoft/gosh/internal/domain/process"
	"github.com/grpmsoft/gosh/internal/domain/session"
	"github.com/grpmsoft/gosh/internal/domain/shared"
	"github.com/grpmsoft/gosh/internal/infrastructure/executor"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"log/slog"
	"os"
	"strings"
)

// TestOSPipelineExecutor_Simple проверяет простой pipeline из двух команд
func TestOSPipelineExecutor_Simple(t *testing.T) {
	t.Run("echo hello | wc -l", func(t *testing.T) {
		// Подготовка
		logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
		executor := executor.NewOSPipelineExecutor(logger)

		env := make(shared.Environment)
		sess, err := session.NewSession("test", os.TempDir(), env)
		require.NoError(t, err)

		// Создаем команды для pipeline: echo hello | wc -l
		cmd1, err := command.NewCommand("echo", []string{"hello"}, command.TypeExternal)
		require.NoError(t, err)

		cmd2, err := command.NewCommand("wc", []string{"-l"}, command.TypeExternal)
		require.NoError(t, err)

		// Выполняем pipeline
		processes, err := executor.Execute(context.Background(), []*command.Command{cmd1, cmd2}, sess)
		require.NoError(t, err)
		require.Len(t, processes, 2)

		// Проверяем последний процесс (wc)
		lastProc := processes[1]
		assert.Equal(t, process.StateCompleted, lastProc.State())
		assert.Equal(t, 0, int(lastProc.ExitCode()))

		// wc -l должен вернуть "1" (одна строка)
		output := strings.TrimSpace(lastProc.Stdout())
		assert.Equal(t, "1", output)
	})
}

// TestOSPipelineExecutor_MultiStage проверяет pipeline из трех команд
func TestOSPipelineExecutor_MultiStage(t *testing.T) {
	t.Run("echo -e 'a\\nb\\nc' | grep b | wc -l", func(t *testing.T) {
		// Подготовка
		logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
		executor := executor.NewOSPipelineExecutor(logger)

		env := make(shared.Environment)
		sess, err := session.NewSession("test", os.TempDir(), env)
		require.NoError(t, err)

		// Создаем команды: echo -e "a\nb\nc" | grep b | wc -l
		cmd1, err := command.NewCommand("echo", []string{"-e", "a\\nb\\nc"}, command.TypeExternal)
		require.NoError(t, err)

		cmd2, err := command.NewCommand("grep", []string{"b"}, command.TypeExternal)
		require.NoError(t, err)

		cmd3, err := command.NewCommand("wc", []string{"-l"}, command.TypeExternal)
		require.NoError(t, err)

		// Выполняем pipeline
		processes, err := executor.Execute(context.Background(), []*command.Command{cmd1, cmd2, cmd3}, sess)
		require.NoError(t, err)
		require.Len(t, processes, 3)

		// Проверяем последний процесс
		lastProc := processes[2]
		assert.Equal(t, process.StateCompleted, lastProc.State())

		// Должна быть одна строка с "b"
		output := strings.TrimSpace(lastProc.Stdout())
		assert.Equal(t, "1", output)
	})
}

// TestOSPipelineExecutor_ErrorPropagation проверяет что ошибки распространяются корректно
func TestOSPipelineExecutor_ErrorPropagation(t *testing.T) {
	t.Run("failing command in pipeline", func(t *testing.T) {
		// Подготовка
		logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
		executor := executor.NewOSPipelineExecutor(logger)

		env := make(shared.Environment)
		sess, err := session.NewSession("test", os.TempDir(), env)
		require.NoError(t, err)

		// Создаем pipeline где первая команда фейлится: false | echo test
		cmd1, err := command.NewCommand("false", []string{}, command.TypeExternal)
		require.NoError(t, err)

		cmd2, err := command.NewCommand("echo", []string{"test"}, command.TypeExternal)
		require.NoError(t, err)

		// Выполняем pipeline
		processes, err := executor.Execute(context.Background(), []*command.Command{cmd1, cmd2}, sess)
		require.NoError(t, err) // Execute сам по себе не возвращает ошибку
		require.Len(t, processes, 2)

		// Первый процесс должен быть Failed
		firstProc := processes[0]
		assert.Equal(t, process.StateFailed, firstProc.State())
		assert.NotEqual(t, 0, int(firstProc.ExitCode()))

		// Второй процесс может быть Completed (echo отработает независимо)
		secondProc := processes[1]
		assert.Equal(t, process.StateCompleted, secondProc.State())
	})
}

// TestOSPipelineExecutor_EmptyCommands проверяет обработку пустого списка команд
func TestOSPipelineExecutor_EmptyCommands(t *testing.T) {
	t.Run("empty commands list returns error", func(t *testing.T) {
		logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
		executor := executor.NewOSPipelineExecutor(logger)

		env := make(shared.Environment)
		sess, err := session.NewSession("test", os.TempDir(), env)
		require.NoError(t, err)

		// Выполняем с пустым списком
		processes, err := executor.Execute(context.Background(), []*command.Command{}, sess)
		assert.Error(t, err)
		assert.Nil(t, processes)
	})
}

// TestOSPipelineExecutor_SingleCommand проверяет pipeline из одной команды
func TestOSPipelineExecutor_SingleCommand(t *testing.T) {
	t.Run("single command in pipeline", func(t *testing.T) {
		logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
		executor := executor.NewOSPipelineExecutor(logger)

		env := make(shared.Environment)
		sess, err := session.NewSession("test", os.TempDir(), env)
		require.NoError(t, err)

		// Одна команда: echo test
		cmd, err := command.NewCommand("echo", []string{"test"}, command.TypeExternal)
		require.NoError(t, err)

		processes, err := executor.Execute(context.Background(), []*command.Command{cmd}, sess)
		require.NoError(t, err)
		require.Len(t, processes, 1)

		// Проверяем результат
		proc := processes[0]
		assert.Equal(t, process.StateCompleted, proc.State())
		assert.Equal(t, "test\n", proc.Stdout())
	})
}

// TestOSPipelineExecutor_LargeOutput проверяет работу с большим выводом
func TestOSPipelineExecutor_LargeOutput(t *testing.T) {
	t.Run("handles large output through pipe", func(t *testing.T) {
		logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
		executor := executor.NewOSPipelineExecutor(logger)

		env := make(shared.Environment)
		sess, err := session.NewSession("test", os.TempDir(), env)
		require.NoError(t, err)

		// Создаем большой вывод: seq 1000 | wc -l
		cmd1, err := command.NewCommand("seq", []string{"1000"}, command.TypeExternal)
		require.NoError(t, err)

		cmd2, err := command.NewCommand("wc", []string{"-l"}, command.TypeExternal)
		require.NoError(t, err)

		processes, err := executor.Execute(context.Background(), []*command.Command{cmd1, cmd2}, sess)
		require.NoError(t, err)
		require.Len(t, processes, 2)

		// Проверяем что получили 1000 строк
		lastProc := processes[1]
		assert.Equal(t, process.StateCompleted, lastProc.State())
		output := strings.TrimSpace(lastProc.Stdout())
		assert.Equal(t, "1000", output)
	})
}
