package builtins

import (
	"fmt"
	"github.com/grpmsoft/gosh/internal/domain/session"
	"github.com/grpmsoft/gosh/internal/domain/shared"
	"io"
	"os"
	"regexp"
	"strings"
)

// ExportCommand represents the export command
// Exports environment variables
// Supports:
// - export             - print all variables (like env)
// - export VAR=value   - set variable
// - export VAR="value" - set with quotes
// - export A=1 B=2     - multiple variable assignment
type ExportCommand struct {
	args    []string
	session *session.Session
	stdout  io.Writer
}

// validVarNameRegex validates variable name
// Name can contain: letters, digits, underscore, but CANNOT start with digit
var validVarNameRegex = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)

// NewExportCommand creates a new export command
func NewExportCommand(args []string, sess *session.Session, stdout io.Writer) (*ExportCommand, error) {
	if sess == nil {
		return nil, shared.NewDomainError(
			"NewExportCommand",
			shared.ErrInvalidArgument,
			"session cannot be nil",
		)
	}
	if stdout == nil {
		return nil, shared.NewDomainError(
			"NewExportCommand",
			shared.ErrInvalidArgument,
			"stdout cannot be nil",
		)
	}

	return &ExportCommand{
		args:    args,
		session: sess,
		stdout:  stdout,
	}, nil
}

// Execute executes the export command
func (e *ExportCommand) Execute() error {
	// If no arguments - print all environment variables
	if len(e.args) == 0 {
		return e.printAllVariables()
	}

	// Process each argument as KEY=VALUE
	for _, arg := range e.args {
		if err := e.exportVariable(arg); err != nil {
			return err
		}
	}

	return nil
}

// exportVariable exports a single variable
func (e *ExportCommand) exportVariable(arg string) error {
	// Split into KEY=VALUE
	parts := strings.SplitN(arg, "=", 2)
	if len(parts) != 2 {
		return shared.NewDomainError(
			"export",
			shared.ErrInvalidArgument,
			fmt.Sprintf("invalid format: '%s' (expected KEY=VALUE)", arg),
		)
	}

	key := strings.TrimSpace(parts[0])
	value := parts[1]

	// Validate variable name
	if err := e.validateVariableName(key); err != nil {
		return err
	}

	// Remove quotes from value if present
	value = e.unquoteValue(value)

	// Set in session
	if err := e.session.SetEnvironmentVariable(key, value); err != nil {
		return err
	}

	// Set in current process
	if err := os.Setenv(key, value); err != nil {
		return shared.NewDomainError(
			"export",
			shared.ErrInvalidArgument,
			fmt.Sprintf("failed to set environment variable: %s", err.Error()),
		)
	}

	return nil
}

// validateVariableName validates variable name
func (e *ExportCommand) validateVariableName(name string) error {
	if name == "" {
		return shared.NewDomainError(
			"export",
			shared.ErrInvalidArgument,
			"variable name cannot be empty",
		)
	}

	if !validVarNameRegex.MatchString(name) {
		return shared.NewDomainError(
			"export",
			shared.ErrInvalidArgument,
			fmt.Sprintf("invalid variable name: '%s' (must start with letter or underscore, contain only alphanumeric and underscore)", name),
		)
	}

	return nil
}

// unquoteValue removes quotes from value
func (e *ExportCommand) unquoteValue(value string) string {
	value = strings.TrimSpace(value)

	// Remove double quotes
	if strings.HasPrefix(value, "\"") && strings.HasSuffix(value, "\"") && len(value) >= 2 {
		return value[1 : len(value)-1]
	}

	// Remove single quotes
	if strings.HasPrefix(value, "'") && strings.HasSuffix(value, "'") && len(value) >= 2 {
		return value[1 : len(value)-1]
	}

	return value
}

// printAllVariables prints all environment variables
func (e *ExportCommand) printAllVariables() error {
	env := e.session.Environment()

	// Sort keys for stable output (optional)
	for key, value := range env {
		_, _ = fmt.Fprintf(e.stdout, "export %s=\"%s\"\n", key, value)
	}

	return nil
}
