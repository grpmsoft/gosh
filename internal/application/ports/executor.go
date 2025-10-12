package ports

import (
	"context"
	"github.com/grpmsoft/gosh/internal/domain/command"
	"github.com/grpmsoft/gosh/internal/domain/process"
	"github.com/grpmsoft/gosh/internal/domain/session"
)

// CommandExecutor - port for executing commands (Hexagonal Architecture)
type CommandExecutor interface {
	Execute(ctx context.Context, cmd *command.Command, sess *session.Session) (*process.Process, error)
}

// BuiltinExecutor - port for executing builtin commands
type BuiltinExecutor interface {
	Execute(ctx context.Context, cmd *command.Command, sess *session.Session) error
	CanExecute(cmd *command.Command) bool
}

// PipelineExecutor - port for executing command pipelines
type PipelineExecutor interface {
	Execute(ctx context.Context, commands []*command.Command, sess *session.Session) ([]*process.Process, error)
}
