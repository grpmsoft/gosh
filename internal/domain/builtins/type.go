package builtins

import (
	"fmt"
	"io"
	"os/exec"

	"github.com/grpmsoft/gosh/internal/domain/command"
	"github.com/grpmsoft/gosh/internal/domain/session"
	"github.com/grpmsoft/gosh/internal/domain/shared"
)

// TypeCommand represents the type command.
// Determines command type: builtin, alias or external.
// Supports:
// - type cmd         - determine type of single command.
// - type cmd1 cmd2   - determine type of multiple commands.
type TypeCommand struct {
	commandNames []string
	session      *session.Session
	stdout       io.Writer
}

// CommandType represents the type of command.
type CommandType int

// Command type constants define possible command classifications.
const (
	// CommandTypeBuiltin represents shell builtin commands.
	CommandTypeBuiltin CommandType = iota
	CommandTypeAlias
	CommandTypeExternal
	CommandTypeNotFound
)

// NewTypeCommand creates a new type command.
func NewTypeCommand(args []string, sess *session.Session, stdout io.Writer) (*TypeCommand, error) {
	if sess == nil {
		return nil, shared.NewDomainError(
			"NewTypeCommand",
			shared.ErrInvalidArgument,
			"session cannot be nil",
		)
	}
	if stdout == nil {
		return nil, shared.NewDomainError(
			"NewTypeCommand",
			shared.ErrInvalidArgument,
			"stdout cannot be nil",
		)
	}

	if len(args) == 0 {
		return nil, shared.NewDomainError(
			"type",
			shared.ErrInvalidArgument,
			"missing command name",
		)
	}

	return &TypeCommand{
		commandNames: args,
		session:      sess,
		stdout:       stdout,
	}, nil
}

// Execute executes the type command.
func (t *TypeCommand) Execute() error {
	for _, cmdName := range t.commandNames {
		t.printCommandType(cmdName)
	}

	return nil
}

// printCommandType prints the command type.
func (t *TypeCommand) printCommandType(cmdName string) {
	cmdType, details := t.getCommandType(cmdName)

	switch cmdType {
	case CommandTypeBuiltin:
		_, _ = fmt.Fprintf(t.stdout, "%s is a shell builtin\n", cmdName)

	case CommandTypeAlias:
		_, _ = fmt.Fprintf(t.stdout, "%s is aliased to '%s'\n", cmdName, details)

	case CommandTypeExternal:
		_, _ = fmt.Fprintf(t.stdout, "%s is %s\n", cmdName, details)

	case CommandTypeNotFound:
		_, _ = fmt.Fprintf(t.stdout, "%s: not found\n", cmdName)
	}
}

// getCommandType determines the command type.
func (t *TypeCommand) getCommandType(cmdName string) (cmdType CommandType, details string) {
	// Check 1: Builtin command?
	if command.IsBuiltinCommand(cmdName) {
		return CommandTypeBuiltin, ""
	}

	// Check 2: Alias?
	if aliasCommand, ok := t.session.GetAlias(cmdName); ok {
		return CommandTypeAlias, aliasCommand
	}

	// Check 3: External command?
	if path, err := exec.LookPath(cmdName); err == nil {
		return CommandTypeExternal, path
	}

	// Not found
	return CommandTypeNotFound, ""
}
