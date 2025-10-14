package builtins

import (
	"fmt"

	"github.com/grpmsoft/gosh/internal/domain/session"
	"github.com/grpmsoft/gosh/internal/domain/shared"
)

// UnaliasCommand represents the unalias command.
// Removes custom aliases.
// Supports:
// - unalias name       - remove single alias.
// - unalias n1 n2 n3   - remove multiple aliases.
// - unalias -a         - remove all aliases.
type UnaliasCommand struct {
	args    []string
	session *session.Session
}

// NewUnaliasCommand creates a new unalias command.
func NewUnaliasCommand(args []string, sess *session.Session) (*UnaliasCommand, error) {
	if sess == nil {
		return nil, shared.NewDomainError(
			"NewUnaliasCommand",
			shared.ErrInvalidArgument,
			"session cannot be nil",
		)
	}

	if len(args) == 0 {
		return nil, shared.NewDomainError(
			"unalias",
			shared.ErrInvalidArgument,
			"missing alias name",
		)
	}

	return &UnaliasCommand{
		args:    args,
		session: sess,
	}, nil
}

// Execute executes the unalias command.
func (u *UnaliasCommand) Execute() error {
	// Check for -a flag (remove all)
	if len(u.args) == 1 && u.args[0] == "-a" {
		return u.removeAllAliases()
	}

	// Remove specified aliases
	for _, name := range u.args {
		if err := u.removeAlias(name); err != nil {
			return err
		}
	}

	return nil
}

// removeAlias removes a single alias.
func (u *UnaliasCommand) removeAlias(name string) error {
	// Check if alias exists
	if _, ok := u.session.GetAlias(name); !ok {
		return shared.NewDomainError(
			"unalias",
			shared.ErrInvalidArgument,
			fmt.Sprintf("alias not found: %s", name),
		)
	}

	// Remove alias
	return u.session.RemoveAlias(name)
}

// removeAllAliases removes all aliases.
func (u *UnaliasCommand) removeAllAliases() error {
	aliases := u.session.GetAllAliases()

	// Remove each alias
	for name := range aliases {
		if err := u.session.RemoveAlias(name); err != nil {
			return err
		}
	}

	return nil
}
