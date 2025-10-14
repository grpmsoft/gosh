// Package execute provides use cases for command execution orchestration.
package execute

import (
	"context"
	"log/slog"

	"github.com/grpmsoft/gosh/internal/application/ports"
	"github.com/grpmsoft/gosh/internal/domain/command"
	"github.com/grpmsoft/gosh/internal/domain/pipeline"
	"github.com/grpmsoft/gosh/internal/domain/process"
	"github.com/grpmsoft/gosh/internal/domain/session"
	"github.com/grpmsoft/gosh/internal/domain/shared"
	"github.com/grpmsoft/gosh/internal/interfaces/parser"
)

// CommandRequest - request to execute a command.
type CommandRequest struct {
	CommandLine string
	SessionID   string
}

// CommandResponse - response to command execution.
type CommandResponse struct {
	Process  *process.Process
	Stdout   string
	Stderr   string
	ExitCode shared.ExitCode
}

// UseCase - use case for executing commands.
type UseCase struct {
	builtinExecutor  ports.BuiltinExecutor
	commandExecutor  ports.CommandExecutor
	pipelineExecutor ports.PipelineExecutor
	logger           *slog.Logger
}

// NewUseCase creates a new use case.
func NewUseCase(
	builtinExecutor ports.BuiltinExecutor,
	commandExecutor ports.CommandExecutor,
	pipelineExecutor ports.PipelineExecutor,
	logger *slog.Logger,
) *UseCase {
	return &UseCase{
		builtinExecutor:  builtinExecutor,
		commandExecutor:  commandExecutor,
		pipelineExecutor: pipelineExecutor,
		logger:           logger,
	}
}

// Execute executes a command in the context of a session.
func (uc *UseCase) Execute(
	ctx context.Context,
	req CommandRequest,
	sess *session.Session,
) (*CommandResponse, error) {
	// Add command to history
	if err := sess.AddToHistory(req.CommandLine); err != nil {
		uc.logger.Warn("failed to add command to history", "error", err)
	}

	// Parse command line
	cmd, pipe, err := uc.parseCommandLine(req.CommandLine)
	if err != nil {
		return nil, err
	}

	// If this is a pipeline
	if pipe != nil {
		return uc.executePipeline(ctx, pipe, sess)
	}

	// If this is a single command
	return uc.executeCommand(ctx, cmd, sess)
}

// parseCommandLine parses the command line.
func (uc *UseCase) parseCommandLine(
	commandLine string,
) (*command.Command, *pipeline.Pipeline, error) {
	// Use parser from interfaces layer
	// TODO: Refactor to use parser as dependency (DI)
	// For now we import parser directly
	return parseCommandLineHelper(commandLine)
}

// executeCommand executes a single command.
func (uc *UseCase) executeCommand(
	ctx context.Context,
	cmd *command.Command,
	sess *session.Session,
) (*CommandResponse, error) {
	uc.logger.Info("executing command",
		"command", cmd.Name(),
		"session", sess.ID(),
	)

	// Check if the command is builtin
	if uc.builtinExecutor.CanExecute(cmd) {
		stdout, stderr, err := uc.builtinExecutor.Execute(ctx, cmd, sess)
		if err != nil {
			return nil, err
		}

		return &CommandResponse{
			Stdout:   stdout,
			Stderr:   stderr,
			ExitCode: shared.ExitSuccess,
		}, nil
	}

	// Execute external command
	proc, err := uc.commandExecutor.Execute(ctx, cmd, sess)
	if err != nil {
		return nil, err
	}

	// Register process in session
	if err := sess.RegisterProcess(proc); err != nil {
		uc.logger.Warn("failed to register process", "error", err)
	}

	// If this is a background command, add job to JobManager
	if cmd.IsBackground() {
		jobManager := sess.JobManager()
		if jobManager != nil {
			job, err := jobManager.AddJob(cmd, proc)
			if err != nil {
				uc.logger.Warn("failed to add background job", "error", err)
			} else {
				uc.logger.Info("background job started",
					"jobNumber", job.JobNumber(),
					"pid", proc.PID(),
					"command", cmd.FullCommand(),
				)
			}
		}
	}

	return &CommandResponse{
		Process:  proc,
		Stdout:   proc.Stdout(),
		Stderr:   proc.Stderr(),
		ExitCode: proc.ExitCode(),
	}, nil
}

// executePipeline executes a pipeline of commands.
func (uc *UseCase) executePipeline(
	ctx context.Context,
	pipe *pipeline.Pipeline,
	sess *session.Session,
) (*CommandResponse, error) {
	uc.logger.Info("executing pipeline",
		"length", pipe.Length(),
		"session", sess.ID(),
	)

	// Execute pipeline
	processes, err := uc.pipelineExecutor.Execute(ctx, pipe.Commands(), sess)
	if err != nil {
		return nil, err
	}

	// Register all processes in session
	for _, proc := range processes {
		if err := sess.RegisterProcess(proc); err != nil {
			uc.logger.Warn("failed to register process", "error", err)
		}
	}

	// Return result of the last process
	if len(processes) > 0 {
		lastProc := processes[len(processes)-1]
		return &CommandResponse{
			Process:  lastProc,
			Stdout:   lastProc.Stdout(),
			Stderr:   lastProc.Stderr(),
			ExitCode: lastProc.ExitCode(),
		}, nil
	}

	return &CommandResponse{
		ExitCode: shared.ExitSuccess,
	}, nil
}

// parseCommandLineHelper parses the command line using the parser package.
// TODO: Refactor to proper DI (dependency injection).
func parseCommandLineHelper(commandLine string) (*command.Command, *pipeline.Pipeline, error) {
	return parser.ParseCommandLine(commandLine)
}
