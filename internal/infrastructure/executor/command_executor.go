package executor

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"github.com/grpmsoft/gosh/internal/domain/command"
	"github.com/grpmsoft/gosh/internal/domain/process"
	"github.com/grpmsoft/gosh/internal/domain/session"
	"github.com/grpmsoft/gosh/internal/domain/shared"
	"log/slog"
	"os"
	"os/exec"
	"time"

	"github.com/google/uuid"
)

// OSCommandExecutor - adapter for executing external commands through OS
type OSCommandExecutor struct {
	logger *slog.Logger
}

// NewOSCommandExecutor creates a new executor
func NewOSCommandExecutor(logger *slog.Logger) *OSCommandExecutor {
	return &OSCommandExecutor{
		logger: logger,
	}
}

// Execute executes an external command
func (e *OSCommandExecutor) Execute(
	ctx context.Context,
	cmd *command.Command,
	sess *session.Session,
) (*process.Process, error) {
	// Create process
	proc, err := process.NewProcess(uuid.New().String(), cmd)
	if err != nil {
		return nil, err
	}

	// Create OS command
	osCmd := exec.CommandContext(ctx, cmd.Name(), cmd.Args()...)

	// Set working directory
	osCmd.Dir = sess.WorkingDirectory()

	// Set environment variables
	osCmd.Env = sess.Environment().ToSlice()

	// Prepare buffers for stdout/stderr
	var stdout, stderr bytes.Buffer
	osCmd.Stdout = &stdout
	osCmd.Stderr = &stderr

	// Handle redirections
	openFiles, err := e.handleRedirections(cmd, osCmd, &stdout, &stderr)
	if err != nil {
		return nil, err
	}

	// Close files after command completion (foreground only)
	// Background commands close files in monitoring goroutine
	if !cmd.IsBackground() {
		defer func() {
			for _, f := range openFiles {
				if closeErr := f.Close(); closeErr != nil {
					e.logger.Warn("failed to close redirected file",
						"file", f.Name(),
						"error", closeErr,
					)
				}
			}
		}()
	}

	// Start process
	startTime := time.Now()
	if err := osCmd.Start(); err != nil {
		e.logger.Error("failed to start command",
			"command", cmd.Name(),
			"error", err,
		)
		return nil, shared.NewDomainError(
			"Execute",
			shared.ErrProcessFailed,
			err.Error(),
		)
	}

	// Mark process as started
	if err := proc.Start(osCmd.Process.Pid); err != nil {
		return nil, err
	}

	e.logger.Info("command started",
		"command", cmd.FullCommand(),
		"pid", osCmd.Process.Pid,
	)

	// Wait for completion or run in background
	if !cmd.IsBackground() {
		// Foreground: wait synchronously
		err = osCmd.Wait()
		duration := time.Since(startTime)

		exitCode := shared.ExitSuccess
		if err != nil {
			var exitErr *exec.ExitError
			if errors.As(err, &exitErr) {
				exitCode = shared.ExitCode(exitErr.ExitCode())
			} else {
				exitCode = shared.ExitError
			}

			e.logger.Warn("command failed",
				"command", cmd.FullCommand(),
				"exitCode", exitCode,
				"duration", duration,
				"error", err,
			)

			// Mark process as failed
			if err := proc.Fail(err, exitCode, stdout.String(), stderr.String()); err != nil {
				return nil, err
			}
		} else {
			e.logger.Info("command completed",
				"command", cmd.FullCommand(),
				"duration", duration,
			)

			// Mark process as completed successfully
			if err := proc.Complete(exitCode, stdout.String(), stderr.String()); err != nil {
				return nil, err
			}
		}
	} else {
		// Background: monitor completion in goroutine
		go e.monitorBackgroundProcess(osCmd, proc, cmd, startTime, &stdout, &stderr, openFiles)
	}

	return proc, nil
}

// monitorBackgroundProcess monitors background process completion in a goroutine
func (e *OSCommandExecutor) monitorBackgroundProcess(
	osCmd *exec.Cmd,
	proc *process.Process,
	cmd *command.Command,
	startTime time.Time,
	stdout, stderr *bytes.Buffer,
	openFiles []*os.File,
) {
	// Wait for process completion
	err := osCmd.Wait()
	duration := time.Since(startTime)

	// Close all redirected files
	for _, f := range openFiles {
		if closeErr := f.Close(); closeErr != nil {
			e.logger.Warn("failed to close redirected file in background job",
				"file", f.Name(),
				"error", closeErr,
			)
		}
	}

	// Determine exit code
	exitCode := shared.ExitSuccess
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			exitCode = shared.ExitCode(exitErr.ExitCode())
		} else {
			exitCode = shared.ExitError
		}

		e.logger.Info("background job failed",
			"command", cmd.FullCommand(),
			"pid", osCmd.Process.Pid,
			"exitCode", exitCode,
			"duration", duration,
			"error", err,
		)

		// Mark process as failed
		if procErr := proc.Fail(err, exitCode, stdout.String(), stderr.String()); procErr != nil {
			e.logger.Error("failed to mark background process as failed",
				"command", cmd.FullCommand(),
				"error", procErr,
			)
		}
	} else {
		e.logger.Info("background job completed",
			"command", cmd.FullCommand(),
			"pid", osCmd.Process.Pid,
			"duration", duration,
		)

		// Mark process as completed successfully
		if procErr := proc.Complete(exitCode, stdout.String(), stderr.String()); procErr != nil {
			e.logger.Error("failed to mark background process as completed",
				"command", cmd.FullCommand(),
				"error", procErr,
			)
		}
	}
}

// handleRedirections handles input/output redirections
func (e *OSCommandExecutor) handleRedirections(
	cmd *command.Command,
	osCmd *exec.Cmd,
	stdout, stderr *bytes.Buffer,
) ([]*os.File, error) {
	redirections := cmd.Redirections()
	if len(redirections) == 0 {
		return nil, nil
	}

	var openFiles []*os.File

	for _, redir := range redirections {
		switch redir.Type {
		case command.RedirectInput:
			// N< - input redirection from file to FD N (default FD 0)
			file, err := os.Open(redir.Target)
			if err != nil {
				e.closeFiles(openFiles)
				return nil, shared.NewDomainError(
					"handleRedirections",
					shared.ErrProcessFailed,
					fmt.Sprintf("failed to open input file '%s': %v", redir.Target, err),
				)
			}

			// Only stdin (FD 0) is supported in os/exec
			if redir.SourceFD == 0 {
				osCmd.Stdin = file
			} else {
				e.logger.Warn("input redirection to FD other than 0 not fully supported",
					"fd", redir.SourceFD, "file", redir.Target)
			}

			openFiles = append(openFiles, file)
			e.logger.Debug("input redirected", "fd", redir.SourceFD, "file", redir.Target)

		case command.RedirectOutput:
			// N> - output redirection to file from FD N (default FD 1)
			file, err := os.Create(redir.Target)
			if err != nil {
				e.closeFiles(openFiles)
				return nil, shared.NewDomainError(
					"handleRedirections",
					shared.ErrProcessFailed,
					fmt.Sprintf("failed to create output file '%s': %v", redir.Target, err),
				)
			}

			// os/exec only supports stdout (FD 1) and stderr (FD 2)
			switch redir.SourceFD {
			case 1:
				osCmd.Stdout = file
			case 2:
				osCmd.Stderr = file
			default:
				e.logger.Warn("output redirection from FD other than 1 or 2 not fully supported",
					"fd", redir.SourceFD, "file", redir.Target)
			}

			openFiles = append(openFiles, file)
			e.logger.Debug("output redirected", "fd", redir.SourceFD, "file", redir.Target)

		case command.RedirectAppend:
			// N>> - append redirection to file from FD N (default FD 1)
			file, err := os.OpenFile(redir.Target, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0644)
			if err != nil {
				e.closeFiles(openFiles)
				return nil, shared.NewDomainError(
					"handleRedirections",
					shared.ErrProcessFailed,
					fmt.Sprintf("failed to open file for append '%s': %v", redir.Target, err),
				)
			}

			switch redir.SourceFD {
			case 1:
				osCmd.Stdout = file
			case 2:
				osCmd.Stderr = file
			default:
				e.logger.Warn("append redirection from FD other than 1 or 2 not fully supported",
					"fd", redir.SourceFD, "file", redir.Target)
			}

			openFiles = append(openFiles, file)
			e.logger.Debug("output redirected (append)", "fd", redir.SourceFD, "file", redir.Target)

		case command.RedirectDup:
			// N>&M - duplicate FD M to FD N (e.g., 2>&1)
			// Note: os/exec has limitations here, we handle common case 2>&1
			if redir.SourceFD == 2 && redir.Target == "1" {
				// 2>&1 - redirect stderr to stdout
				osCmd.Stderr = osCmd.Stdout
				e.logger.Debug("stderr duplicated to stdout", "fd", redir.SourceFD)
			} else if redir.SourceFD == 1 && redir.Target == "2" {
				// 1>&2 - redirect stdout to stderr
				osCmd.Stdout = osCmd.Stderr
				e.logger.Debug("stdout duplicated to stderr", "fd", redir.SourceFD)
			} else {
				e.logger.Warn("FD duplication not fully supported",
					"source_fd", redir.SourceFD, "target", redir.Target)
			}

		case command.RedirectPipe:
			// Pipes are handled in OSPipelineExecutor, ignore here
			continue
		}
	}

	return openFiles, nil
}

// closeFiles closes all open files (helper function)
func (e *OSCommandExecutor) closeFiles(files []*os.File) {
	for _, f := range files {
		if err := f.Close(); err != nil {
			e.logger.Warn("failed to close file during cleanup",
				"file", f.Name(),
				"error", err,
			)
		}
	}
}
