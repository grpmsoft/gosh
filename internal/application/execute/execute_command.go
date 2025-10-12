package execute

import (
	"context"
	"github.com/grpmsoft/gosh/internal/application/ports"
	"github.com/grpmsoft/gosh/internal/domain/command"
	"github.com/grpmsoft/gosh/internal/domain/pipeline"
	"github.com/grpmsoft/gosh/internal/domain/process"
	"github.com/grpmsoft/gosh/internal/domain/session"
	"github.com/grpmsoft/gosh/internal/domain/shared"
	"log/slog"
)

// ExecuteCommandRequest - request to execute a command
type ExecuteCommandRequest struct {
	CommandLine string
	SessionID   string
}

// ExecuteCommandResponse - response to command execution
type ExecuteCommandResponse struct {
	Process  *process.Process
	Stdout   string
	Stderr   string
	ExitCode shared.ExitCode
}

// ExecuteCommandUseCase - use case for executing commands
type ExecuteCommandUseCase struct {
	builtinExecutor  ports.BuiltinExecutor
	commandExecutor  ports.CommandExecutor
	pipelineExecutor ports.PipelineExecutor
	logger           *slog.Logger
}

// NewExecuteCommandUseCase creates a new use case
func NewExecuteCommandUseCase(
	builtinExecutor ports.BuiltinExecutor,
	commandExecutor ports.CommandExecutor,
	pipelineExecutor ports.PipelineExecutor,
	logger *slog.Logger,
) *ExecuteCommandUseCase {
	return &ExecuteCommandUseCase{
		builtinExecutor:  builtinExecutor,
		commandExecutor:  commandExecutor,
		pipelineExecutor: pipelineExecutor,
		logger:           logger,
	}
}

// Execute executes a command in the context of a session
func (uc *ExecuteCommandUseCase) Execute(
	ctx context.Context,
	req ExecuteCommandRequest,
	sess *session.Session,
) (*ExecuteCommandResponse, error) {
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

// parseCommandLine parses the command line
func (uc *ExecuteCommandUseCase) parseCommandLine(
	commandLine string,
) (*command.Command, *pipeline.Pipeline, error) {
	// Use parser from interfaces layer
	// In a real application the parser should be passed as a dependency
	// but for simplicity we use direct call for now
	// TODO: Refactor to use parser as dependency
	return nil, nil, shared.NewDomainError(
		"parseCommandLine",
		shared.ErrInvalidCommand,
		"parser not implemented yet",
	)
}

// executeCommand executes a single command
func (uc *ExecuteCommandUseCase) executeCommand(
	ctx context.Context,
	cmd *command.Command,
	sess *session.Session,
) (*ExecuteCommandResponse, error) {
	uc.logger.Info("executing command",
		"command", cmd.Name(),
		"session", sess.ID(),
	)

	// Check if the command is builtin
	if uc.builtinExecutor.CanExecute(cmd) {
		if err := uc.builtinExecutor.Execute(ctx, cmd, sess); err != nil {
			return nil, err
		}

		return &ExecuteCommandResponse{
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

	return &ExecuteCommandResponse{
		Process:  proc,
		Stdout:   proc.Stdout(),
		Stderr:   proc.Stderr(),
		ExitCode: proc.ExitCode(),
	}, nil
}

// executePipeline executes a pipeline of commands
func (uc *ExecuteCommandUseCase) executePipeline(
	ctx context.Context,
	pipe *pipeline.Pipeline,
	sess *session.Session,
) (*ExecuteCommandResponse, error) {
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
		return &ExecuteCommandResponse{
			Process:  lastProc,
			Stdout:   lastProc.Stdout(),
			Stderr:   lastProc.Stderr(),
			ExitCode: lastProc.ExitCode(),
		}, nil
	}

	return &ExecuteCommandResponse{
		ExitCode: shared.ExitSuccess,
	}, nil
}
