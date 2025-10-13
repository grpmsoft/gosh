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

// TestOSPipelineExecutor_Simple tests a simple pipeline of two commands
func TestOSPipelineExecutor_Simple(t *testing.T) {
	t.Run("echo hello | wc -l", func(t *testing.T) {
		// Setup
		logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
		executor := executor.NewOSPipelineExecutor(logger)

		env := make(shared.Environment)
		sess, err := session.NewSession("test", os.TempDir(), env)
		require.NoError(t, err)

		// Create commands for pipeline: echo hello | wc -l
		cmd1, err := command.NewCommand("echo", []string{"hello"}, command.TypeExternal)
		require.NoError(t, err)

		cmd2, err := command.NewCommand("wc", []string{"-l"}, command.TypeExternal)
		require.NoError(t, err)

		// Execute pipeline
		processes, err := executor.Execute(context.Background(), []*command.Command{cmd1, cmd2}, sess)
		require.NoError(t, err)
		require.Len(t, processes, 2)

		// Check last process (wc)
		lastProc := processes[1]
		assert.Equal(t, process.StateCompleted, lastProc.State())
		assert.Equal(t, 0, int(lastProc.ExitCode()))

		// wc -l should return "1" (one line)
		output := strings.TrimSpace(lastProc.Stdout())
		assert.Equal(t, "1", output)
	})
}

// TestOSPipelineExecutor_MultiStage tests a pipeline of three commands
func TestOSPipelineExecutor_MultiStage(t *testing.T) {
	t.Run("echo -e 'a\\nb\\nc' | grep b | wc -l", func(t *testing.T) {
		// Setup
		logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
		executor := executor.NewOSPipelineExecutor(logger)

		env := make(shared.Environment)
		sess, err := session.NewSession("test", os.TempDir(), env)
		require.NoError(t, err)

		// Create commands: echo -e "a\nb\nc" | grep b | wc -l
		cmd1, err := command.NewCommand("echo", []string{"-e", "a\\nb\\nc"}, command.TypeExternal)
		require.NoError(t, err)

		cmd2, err := command.NewCommand("grep", []string{"b"}, command.TypeExternal)
		require.NoError(t, err)

		cmd3, err := command.NewCommand("wc", []string{"-l"}, command.TypeExternal)
		require.NoError(t, err)

		// Execute pipeline
		processes, err := executor.Execute(context.Background(), []*command.Command{cmd1, cmd2, cmd3}, sess)
		require.NoError(t, err)
		require.Len(t, processes, 3)

		// Check last process
		lastProc := processes[2]
		assert.Equal(t, process.StateCompleted, lastProc.State())

		// Should be one line with "b"
		output := strings.TrimSpace(lastProc.Stdout())
		assert.Equal(t, "1", output)
	})
}

// TestOSPipelineExecutor_ErrorPropagation tests that errors propagate correctly
func TestOSPipelineExecutor_ErrorPropagation(t *testing.T) {
	t.Run("failing command in pipeline", func(t *testing.T) {
		// Setup
		logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
		executor := executor.NewOSPipelineExecutor(logger)

		env := make(shared.Environment)
		sess, err := session.NewSession("test", os.TempDir(), env)
		require.NoError(t, err)

		// Create pipeline where first command fails: false | echo test
		cmd1, err := command.NewCommand("false", []string{}, command.TypeExternal)
		require.NoError(t, err)

		cmd2, err := command.NewCommand("echo", []string{"test"}, command.TypeExternal)
		require.NoError(t, err)

		// Execute pipeline
		processes, err := executor.Execute(context.Background(), []*command.Command{cmd1, cmd2}, sess)
		require.NoError(t, err) // Execute itself does not return error
		require.Len(t, processes, 2)

		// First process should be Failed
		firstProc := processes[0]
		assert.Equal(t, process.StateFailed, firstProc.State())
		assert.NotEqual(t, 0, int(firstProc.ExitCode()))

		// Second process can be Completed (echo will work independently)
		secondProc := processes[1]
		assert.Equal(t, process.StateCompleted, secondProc.State())
	})
}

// TestOSPipelineExecutor_EmptyCommands tests handling of empty command list
func TestOSPipelineExecutor_EmptyCommands(t *testing.T) {
	t.Run("empty commands list returns error", func(t *testing.T) {
		logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
		executor := executor.NewOSPipelineExecutor(logger)

		env := make(shared.Environment)
		sess, err := session.NewSession("test", os.TempDir(), env)
		require.NoError(t, err)

		// Execute with empty list
		processes, err := executor.Execute(context.Background(), []*command.Command{}, sess)
		assert.Error(t, err)
		assert.Nil(t, processes)
	})
}

// TestOSPipelineExecutor_SingleCommand tests a pipeline with a single command
func TestOSPipelineExecutor_SingleCommand(t *testing.T) {
	t.Run("single command in pipeline", func(t *testing.T) {
		logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
		executor := executor.NewOSPipelineExecutor(logger)

		env := make(shared.Environment)
		sess, err := session.NewSession("test", os.TempDir(), env)
		require.NoError(t, err)

		// One command: echo test
		cmd, err := command.NewCommand("echo", []string{"test"}, command.TypeExternal)
		require.NoError(t, err)

		processes, err := executor.Execute(context.Background(), []*command.Command{cmd}, sess)
		require.NoError(t, err)
		require.Len(t, processes, 1)

		// Check result
		proc := processes[0]
		assert.Equal(t, process.StateCompleted, proc.State())
		assert.Equal(t, "test\n", proc.Stdout())
	})
}

// TestOSPipelineExecutor_LargeOutput tests handling large output
func TestOSPipelineExecutor_LargeOutput(t *testing.T) {
	t.Run("handles large output through pipe", func(t *testing.T) {
		logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
		executor := executor.NewOSPipelineExecutor(logger)

		env := make(shared.Environment)
		sess, err := session.NewSession("test", os.TempDir(), env)
		require.NoError(t, err)

		// Create large output: seq 1000 | wc -l
		cmd1, err := command.NewCommand("seq", []string{"1000"}, command.TypeExternal)
		require.NoError(t, err)

		cmd2, err := command.NewCommand("wc", []string{"-l"}, command.TypeExternal)
		require.NoError(t, err)

		processes, err := executor.Execute(context.Background(), []*command.Command{cmd1, cmd2}, sess)
		require.NoError(t, err)
		require.Len(t, processes, 2)

		// Check that we got 1000 lines
		lastProc := processes[1]
		assert.Equal(t, process.StateCompleted, lastProc.State())
		output := strings.TrimSpace(lastProc.Stdout())
		assert.Equal(t, "1000", output)
	})
}
