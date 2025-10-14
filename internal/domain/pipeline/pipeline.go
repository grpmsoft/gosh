// Package pipeline provides domain models for command pipelines and their composition.
package pipeline

import (
	"github.com/grpmsoft/gosh/internal/domain/command"
	"github.com/grpmsoft/gosh/internal/domain/shared"
)

// Pipeline represents a command pipeline (Value Object).
// Immutable by nature, following DDD principles.
type Pipeline struct {
	commands []*command.Command
}

// NewPipeline creates a new command pipeline.
func NewPipeline(commands []*command.Command) (*Pipeline, error) {
	if len(commands) == 0 {
		return nil, shared.NewDomainError(
			"NewPipeline",
			shared.ErrInvalidCommand,
			"pipeline must contain at least one command",
		)
	}

	// Check that all commands are valid
	for _, cmd := range commands {
		if cmd == nil {
			return nil, shared.NewDomainError(
				"NewPipeline",
				shared.ErrInvalidCommand,
				"command is nil",
			)
		}
	}

	// Clone commands for immutability
	clonedCommands := make([]*command.Command, len(commands))
	for i, cmd := range commands {
		clonedCommands[i] = cmd.Clone()
	}

	return &Pipeline{
		commands: clonedCommands,
	}, nil
}

// Commands returns a copy of commands in the pipeline.
func (p *Pipeline) Commands() []*command.Command {
	commands := make([]*command.Command, len(p.commands))
	for i, cmd := range p.commands {
		commands[i] = cmd.Clone()
	}
	return commands
}

// Length returns the number of commands in the pipeline.
func (p *Pipeline) Length() int {
	return len(p.commands)
}

// First returns the first command.
func (p *Pipeline) First() *command.Command {
	if len(p.commands) == 0 {
		return nil
	}
	return p.commands[0].Clone()
}

// Last returns the last command.
func (p *Pipeline) Last() *command.Command {
	if len(p.commands) == 0 {
		return nil
	}
	return p.commands[len(p.commands)-1].Clone()
}

// At returns the command at index.
func (p *Pipeline) At(index int) (*command.Command, error) {
	if index < 0 || index >= len(p.commands) {
		return nil, shared.NewDomainError(
			"At",
			shared.ErrInvalidArgument,
			"index out of range",
		)
	}
	return p.commands[index].Clone(), nil
}

// IsEmpty checks if the pipeline is empty.
func (p *Pipeline) IsEmpty() bool {
	return len(p.commands) == 0
}

// IsSingle checks if the pipeline contains only one command.
func (p *Pipeline) IsSingle() bool {
	return len(p.commands) == 1
}

// Equals compares two pipelines (Value Object equality).
func (p *Pipeline) Equals(other *Pipeline) bool {
	if other == nil {
		return false
	}

	if len(p.commands) != len(other.commands) {
		return false
	}

	for i, cmd := range p.commands {
		if cmd.Name() != other.commands[i].Name() {
			return false
		}
	}

	return true
}
