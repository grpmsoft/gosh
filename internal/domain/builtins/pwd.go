package builtins

import (
	"fmt"
	"github.com/grpmsoft/gosh/internal/domain/session"
	"github.com/grpmsoft/gosh/internal/domain/shared"
	"io"
)

// PwdCommand represents the pwd (print working directory) command
// Prints the current working directory
type PwdCommand struct {
	session *session.Session
	stdout  io.Writer
}

// NewPwdCommand creates a new pwd command
func NewPwdCommand(sess *session.Session, stdout io.Writer) (*PwdCommand, error) {
	if sess == nil {
		return nil, shared.NewDomainError(
			"NewPwdCommand",
			shared.ErrInvalidArgument,
			"session cannot be nil",
		)
	}
	if stdout == nil {
		return nil, shared.NewDomainError(
			"NewPwdCommand",
			shared.ErrInvalidArgument,
			"stdout cannot be nil",
		)
	}

	return &PwdCommand{
		session: sess,
		stdout:  stdout,
	}, nil
}

// Execute executes the pwd command
func (p *PwdCommand) Execute() error {
	currentDir := p.session.WorkingDirectory()
	_, _ = fmt.Fprintln(p.stdout, currentDir)
	return nil
}
