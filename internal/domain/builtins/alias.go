// Package builtins provides domain models for shell builtin commands.
package builtins

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/grpmsoft/gosh/internal/domain/session"
	"github.com/grpmsoft/gosh/internal/domain/shared"
)

// AliasCommand represents the alias command.
// Manages custom command aliases.
// Supports:
// - alias              - print all aliases.
// - alias name='cmd'   - create alias.
// - alias name="cmd"   - create with double quotes.
// - alias name=cmd     - create without quotes.
type AliasCommand struct {
	args    []string
	session *session.Session
	stdout  io.Writer
}

// NewAliasCommand creates a new alias command.
func NewAliasCommand(args []string, sess *session.Session, stdout io.Writer) (*AliasCommand, error) {
	if sess == nil {
		return nil, shared.NewDomainError(
			"NewAliasCommand",
			shared.ErrInvalidArgument,
			"session cannot be nil",
		)
	}
	if stdout == nil {
		return nil, shared.NewDomainError(
			"NewAliasCommand",
			shared.ErrInvalidArgument,
			"stdout cannot be nil",
		)
	}

	return &AliasCommand{
		args:    args,
		session: sess,
		stdout:  stdout,
	}, nil
}

// Execute executes the alias command.
func (a *AliasCommand) Execute() error {
	// If no arguments - print all aliases
	if len(a.args) == 0 {
		return a.printAllAliases()
	}

	// If one argument without '=' - show specific alias
	if len(a.args) == 1 && !strings.Contains(a.args[0], "=") {
		return a.printAlias(a.args[0])
	}

	// Otherwise - create alias(es)
	for _, arg := range a.args {
		if err := a.createAlias(arg); err != nil {
			return err
		}
	}

	return nil
}

// printAllAliases prints all aliases sorted by name.
func (a *AliasCommand) printAllAliases() error {
	aliases := a.session.GetAllAliases()

	if len(aliases) == 0 {
		// Print nothing if no aliases (like bash)
		return nil
	}

	// Sort by name for stable output
	names := make([]string, 0, len(aliases))
	for name := range aliases {
		names = append(names, name)
	}
	sort.Strings(names)

	// Print in format: alias name='command'
	for _, name := range names {
		command := aliases[name]
		_, _ = fmt.Fprintf(a.stdout, "alias %s='%s'\n", name, command)
	}

	return nil
}

// printAlias prints a specific alias.
func (a *AliasCommand) printAlias(name string) error {
	command, ok := a.session.GetAlias(name)
	if !ok {
		return shared.NewDomainError(
			"alias",
			shared.ErrInvalidArgument,
			fmt.Sprintf("alias not found: %s", name),
		)
	}

	_, _ = fmt.Fprintf(a.stdout, "alias %s='%s'\n", name, command)
	return nil
}

// createAlias creates a new alias.
func (a *AliasCommand) createAlias(arg string) error {
	// Split into name=command
	parts := strings.SplitN(arg, "=", 2)
	if len(parts) != 2 {
		return shared.NewDomainError(
			"alias",
			shared.ErrInvalidArgument,
			fmt.Sprintf("invalid format: '%s' (expected name='command')", arg),
		)
	}

	name := strings.TrimSpace(parts[0])
	command := parts[1]

	// Validate name
	if err := a.validateAliasName(name); err != nil {
		return err
	}

	// Remove quotes from command if present
	command = a.unquoteCommand(command)

	// Check for empty command
	if strings.TrimSpace(command) == "" {
		return shared.NewDomainError(
			"alias",
			shared.ErrInvalidArgument,
			"alias command cannot be empty",
		)
	}

	// Create alias in session
	return a.session.SetAlias(name, command)
}

// validateAliasName validates alias name.
func (a *AliasCommand) validateAliasName(name string) error {
	if name == "" {
		return shared.NewDomainError(
			"alias",
			shared.ErrInvalidArgument,
			"alias name cannot be empty",
		)
	}

	// Name must not contain special characters
	for _, ch := range name {
		if !isValidAliasChar(ch) {
			return shared.NewDomainError(
				"alias",
				shared.ErrInvalidArgument,
				fmt.Sprintf("invalid alias name: '%s' (contains invalid character '%c')", name, ch),
			)
		}
	}

	return nil
}

// isValidAliasChar checks if character is allowed in alias name.
func isValidAliasChar(ch rune) bool {
	// Allowed: letters, digits, underscore, hyphen, dot
	return (ch >= 'a' && ch <= 'z') ||
		(ch >= 'A' && ch <= 'Z') ||
		(ch >= '0' && ch <= '9') ||
		ch == '_' ||
		ch == '-' ||
		ch == '.'
}

// unquoteCommand removes quotes from command.
func (a *AliasCommand) unquoteCommand(command string) string {
	command = strings.TrimSpace(command)

	// Remove double quotes
	if strings.HasPrefix(command, "\"") && strings.HasSuffix(command, "\"") && len(command) >= 2 {
		return command[1 : len(command)-1]
	}

	// Remove single quotes
	if strings.HasPrefix(command, "'") && strings.HasSuffix(command, "'") && len(command) >= 2 {
		return command[1 : len(command)-1]
	}

	return command
}
