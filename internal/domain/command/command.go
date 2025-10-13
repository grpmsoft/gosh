package command

import (
	"github.com/grpmsoft/gosh/internal/domain/shared"
	"strings"
)

// Type defines the command type
type Type int

const (
	TypeExternal Type = iota // External command (ls, cat, etc.)
	TypeBuiltin              // Built-in command (cd, pwd, exit)
	TypePipeline             // Pipeline of commands
)

// Redirection represents input/output redirection
// Supports bash-style file descriptor redirections:
// - N<file (read from file to FD N, default 0<)
// - N>file (write from FD N to file, default 1>)
// - N>>file (append from FD N to file, default 1>>)
// - N>&M (duplicate FD M to FD N, e.g., 2>&1)
type Redirection struct {
	Type     RedirectionType
	SourceFD int    // Source file descriptor (e.g., 2 in "2>file" or "2>&1")
	Target   string // Target file path or "&N" for FD duplication
}

// RedirectionType redirection type
type RedirectionType int

const (
	RedirectInput  RedirectionType = iota // N< (default 0<)
	RedirectOutput                        // N> (default 1>)
	RedirectAppend                        // N>> (default 1>>)
	RedirectDup                           // N>&M (FD duplication)
	RedirectPipe                          // | (pipe operator)
)

// Command represents a shell command (Aggregate Root)
// Following Rich Domain Model, command contains business validation logic
type Command struct {
	name         string
	args         []string
	cmdType      Type
	redirections []Redirection
	background   bool
}

// NewCommand creates a new command with validation
func NewCommand(name string, args []string, cmdType Type) (*Command, error) {
	if name == "" {
		return nil, shared.ErrEmptyCommand
	}

	// Clean name from whitespace
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, shared.ErrEmptyCommand
	}

	cmd := &Command{
		name:         name,
		args:         make([]string, len(args)),
		cmdType:      cmdType,
		redirections: make([]Redirection, 0),
		background:   false,
	}

	// Copy arguments for immutability
	copy(cmd.args, args)

	return cmd, nil
}

// Name returns the command name
func (c *Command) Name() string {
	return c.name
}

// Args returns a copy of arguments
func (c *Command) Args() []string {
	args := make([]string, len(c.args))
	copy(args, c.args)
	return args
}

// Type returns the command type
func (c *Command) Type() Type {
	return c.cmdType
}

// Redirections returns a copy of redirections
func (c *Command) Redirections() []Redirection {
	redirects := make([]Redirection, len(c.redirections))
	copy(redirects, c.redirections)
	return redirects
}

// IsBackground checks if the command should execute in background
func (c *Command) IsBackground() bool {
	return c.background
}

// AddRedirection adds a redirection (following Rich Model principle)
func (c *Command) AddRedirection(redir Redirection) error {
	// Validate redirection - Target is required for file redirections
	if redir.Target == "" && redir.Type != RedirectPipe {
		return shared.NewDomainError(
			"AddRedirection",
			shared.ErrInvalidArgument,
			"target cannot be empty",
		)
	}

	c.redirections = append(c.redirections, redir)
	return nil
}

// SetBackground sets the background execution flag
func (c *Command) SetBackground(background bool) {
	c.background = background
}

// IsBuiltin checks if the command is built-in
func (c *Command) IsBuiltin() bool {
	return c.cmdType == TypeBuiltin
}

// IsExternal checks if the command is external
func (c *Command) IsExternal() bool {
	return c.cmdType == TypeExternal
}

// IsPipeline checks if the command is a pipeline
func (c *Command) IsPipeline() bool {
	return c.cmdType == TypePipeline
}

// FullCommand returns the full command with arguments
func (c *Command) FullCommand() string {
	if len(c.args) == 0 {
		return c.name
	}
	return c.name + " " + strings.Join(c.args, " ")
}

// Clone creates a copy of the command (for immutability)
func (c *Command) Clone() *Command {
	clone := &Command{
		name:         c.name,
		args:         make([]string, len(c.args)),
		cmdType:      c.cmdType,
		redirections: make([]Redirection, len(c.redirections)),
		background:   c.background,
	}
	copy(clone.args, c.args)
	copy(clone.redirections, c.redirections)
	return clone
}
