package executor

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"os/exec"

	"github.com/grpmsoft/gosh/internal/domain/command"
	"github.com/grpmsoft/gosh/internal/domain/process"
	"github.com/grpmsoft/gosh/internal/domain/session"
	"github.com/grpmsoft/gosh/internal/domain/shared"

	"github.com/google/uuid"
)

// OSPipelineExecutor - adapter for executing command pipelines.
type OSPipelineExecutor struct {
	logger *slog.Logger
}

// NewOSPipelineExecutor creates a new pipeline executor.
func NewOSPipelineExecutor(logger *slog.Logger) *OSPipelineExecutor {
	return &OSPipelineExecutor{
		logger: logger,
	}
}

// Execute executes a command pipeline.
func (e *OSPipelineExecutor) Execute(
	ctx context.Context,
	commands []*command.Command,
	sess *session.Session,
) ([]*process.Process, error) {
	if len(commands) == 0 {
		return nil, shared.ErrEmptyCommand
	}

	e.logger.Info("executing pipeline",
		"commands", len(commands),
	)

	// Create OS commands
	osCommands := make([]*exec.Cmd, len(commands))
	processes := make([]*process.Process, len(commands))

	for i, cmd := range commands {
		osCmd := exec.CommandContext(ctx, cmd.Name(), cmd.Args()...) //nolint:gosec // G204: This is a shell - command execution with user input is expected
		osCmd.Dir = sess.WorkingDirectory()
		osCmd.Env = sess.Environment().ToSlice()
		osCommands[i] = osCmd

		// Create process
		proc, err := process.NewProcess(uuid.New().String(), cmd)
		if err != nil {
			return nil, err
		}
		processes[i] = proc
	}

	// Link commands through pipe
	for i := 0; i < len(osCommands)-1; i++ {
		stdout, err := osCommands[i].StdoutPipe()
		if err != nil {
			return nil, shared.NewDomainError(
				"Execute",
				shared.ErrPipelineFailed,
				"failed to create stdout pipe: "+err.Error(),
			)
		}
		osCommands[i+1].Stdin = stdout
	}

	// Last command outputs to buffer
	var finalOut, finalErr bytes.Buffer
	osCommands[len(osCommands)-1].Stdout = &finalOut
	osCommands[len(osCommands)-1].Stderr = &finalErr

	// Start all commands
	for i, osCmd := range osCommands {
		if err := osCmd.Start(); err != nil {
			e.logger.Error("failed to start pipeline command",
				"command", commands[i].Name(),
				"error", err,
			)
			return nil, shared.NewDomainError(
				"Execute",
				shared.ErrPipelineFailed,
				err.Error(),
			)
		}

		// Mark process as started
		if err := processes[i].Start(osCmd.Process.Pid); err != nil {
			return nil, err
		}

		e.logger.Debug("pipeline command started",
			"command", commands[i].FullCommand(),
			"pid", osCmd.Process.Pid,
		)
	}

	// Wait for completion of all commands
	for i, osCmd := range osCommands {
		err := osCmd.Wait()

		exitCode := shared.ExitSuccess
		stdout := ""
		stderr := ""

		// For last command, capture output data
		if i == len(osCommands)-1 {
			stdout = finalOut.String()
			stderr = finalErr.String()
		}

		if err != nil {
			var exitErr *exec.ExitError
			if errors.As(err, &exitErr) {
				exitCode = shared.ExitCode(exitErr.ExitCode())
			} else {
				exitCode = shared.ExitError
			}

			e.logger.Warn("pipeline command failed",
				"command", commands[i].FullCommand(),
				"exitCode", exitCode,
			)

			if err := processes[i].Fail(err, exitCode, stdout, stderr); err != nil {
				return nil, err
			}
		} else {
			if err := processes[i].Complete(exitCode, stdout, stderr); err != nil {
				return nil, err
			}
		}
	}

	e.logger.Info("pipeline completed",
		"commands", len(commands),
	)

	return processes, nil
}
