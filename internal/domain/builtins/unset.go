package builtins

import (
	"fmt"
	"os"

	"github.com/grpmsoft/gosh/internal/domain/session"
	"github.com/grpmsoft/gosh/internal/domain/shared"
)

// UnsetCommand represents the unset command.
// Removes environment variables.
// Supports:
// - unset VAR        - remove single variable.
// - unset A B C      - remove multiple variables.
type UnsetCommand struct {
	varNames []string
	session  *session.Session
}

// NewUnsetCommand creates a new unset command.
func NewUnsetCommand(args []string, sess *session.Session) (*UnsetCommand, error) {
	if sess == nil {
		return nil, shared.NewDomainError(
			"NewUnsetCommand",
			shared.ErrInvalidArgument,
			"session cannot be nil",
		)
	}

	if len(args) == 0 {
		return nil, shared.NewDomainError(
			"unset",
			shared.ErrInvalidArgument,
			"missing variable name",
		)
	}

	return &UnsetCommand{
		varNames: args,
		session:  sess,
	}, nil
}

// Execute executes the unset command.
func (u *UnsetCommand) Execute() error {
	for _, varName := range u.varNames {
		if err := u.unsetVariable(varName); err != nil {
			return err
		}
	}

	return nil
}

// unsetVariable removes a single variable.
func (u *UnsetCommand) unsetVariable(varName string) error {
	if varName == "" {
		return shared.NewDomainError(
			"unset",
			shared.ErrInvalidArgument,
			"variable name cannot be empty",
		)
	}

	// Remove from session
	if err := u.session.UnsetEnvironmentVariable(varName); err != nil {
		return err
	}

	// Remove from current process
	if err := os.Unsetenv(varName); err != nil {
		return shared.NewDomainError(
			"unset",
			shared.ErrInvalidArgument,
			fmt.Sprintf("failed to unset environment variable '%s': %s", varName, err.Error()),
		)
	}

	return nil
}
