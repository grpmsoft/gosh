package parser

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/grpmsoft/gosh/internal/domain/command"
	"github.com/grpmsoft/gosh/internal/domain/pipeline"
	"github.com/grpmsoft/gosh/internal/domain/shared"
)

// Parser performs syntactic analysis of tokens.
type Parser struct {
	tokens []Token
	pos    int
}

// NewParser creates a new parser.
func NewParser(tokens []Token) *Parser {
	return &Parser{
		tokens: tokens,
		pos:    0,
	}
}

// Parse parses tokens into a command or pipeline.
func (p *Parser) Parse() (*command.Command, *pipeline.Pipeline, error) {
	commands, err := p.parseCommands()
	if err != nil {
		return nil, nil, err
	}

	if len(commands) == 0 {
		return nil, nil, shared.ErrEmptyCommand
	}

	// If single command - return command
	if len(commands) == 1 {
		return commands[0], nil, nil
	}

	// If multiple - create pipeline
	pipe, err := pipeline.NewPipeline(commands)
	if err != nil {
		return nil, nil, err
	}

	return nil, pipe, nil
}

// parseCommands parses a sequence of commands.
func (p *Parser) parseCommands() ([]*command.Command, error) {
	commands := make([]*command.Command, 0)

	for !p.isAtEnd() {
		cmd, err := p.parseCommand()
		if err != nil {
			// Skip non-command tokens (pipes, semicolons, etc.)
			if errors.Is(err, shared.ErrSkipCommand) {
				// Continue to next token
				continue
			}
			return nil, err
		}

		if cmd != nil {
			commands = append(commands, cmd)
		}

		// Check separators
		if p.current().Type == TokenPipe {
			p.advance() // Consume |
			continue
		}

		if p.current().Type == TokenSemicolon {
			p.advance() // Consume ;
			// Semicolon separates commands, but we only support one for now
			break
		}

		if p.current().Type == TokenEOF {
			break
		}
	}

	return commands, nil
}

// parseCommand parses a single command.
func (p *Parser) parseCommand() (*command.Command, error) {
	if p.isAtEnd() {
		return nil, shared.ErrSkipCommand
	}

	// Skip empty tokens
	if p.current().Type != TokenWord {
		return nil, shared.ErrSkipCommand
	}

	// Read command name
	name := p.current().Value
	p.advance()

	// Determine command type
	cmdType := command.GetCommandType(name)

	// Read arguments
	args := make([]string, 0)
	for !p.isAtEnd() && p.current().Type == TokenWord {
		args = append(args, p.current().Value)
		p.advance()
	}

	// Expand glob patterns in arguments
	expandedArgs, err := expandGlobs(args)
	if err != nil {
		return nil, err
	}

	// Create command
	cmd, err := command.NewCommand(name, expandedArgs, cmdType)
	if err != nil {
		return nil, err
	}

	// Parse redirections
	for !p.isAtEnd() {
		token := p.current()

		var redirType command.RedirectionType
		sourceFD := token.FD

		switch token.Type {
		case TokenRedirectIn:
			redirType = command.RedirectInput
		case TokenRedirectOut:
			redirType = command.RedirectOutput
		case TokenRedirectAppend:
			redirType = command.RedirectAppend
		case TokenRedirectDup:
			// N>&M - FD duplication
			redirType = command.RedirectDup
		default:
			goto done
		}

		p.advance()

		// For FD duplication (N>&M), target is &M
		// For file redirections, target is filename
		if p.isAtEnd() || p.current().Type != TokenWord {
			return nil, shared.NewDomainError(
				"parseCommand",
				shared.ErrInvalidCommand,
				"expected filename or &N after redirection",
			)
		}

		target := p.current().Value
		p.advance()

		// Add redirection with FD support
		err := cmd.AddRedirection(command.Redirection{
			Type:     redirType,
			SourceFD: sourceFD,
			Target:   target,
		})
		if err != nil {
			return nil, err
		}
	}

done:
	// Check for background execution
	if !p.isAtEnd() && p.current().Type == TokenBackground {
		cmd.SetBackground(true)
		p.advance()
	}

	return cmd, nil
}

// current returns the current token.
func (p *Parser) current() Token {
	if p.pos >= len(p.tokens) {
		return Token{Type: TokenEOF}
	}
	return p.tokens[p.pos]
}

// advance moves to the next token.
func (p *Parser) advance() {
	if p.pos < len(p.tokens) {
		p.pos++
	}
}

// isAtEnd checks if end of tokens reached.
func (p *Parser) isAtEnd() bool {
	return p.pos >= len(p.tokens) || p.current().Type == TokenEOF
}

// ParseCommandLine is a convenience function for parsing a command line.
func ParseCommandLine(input string) (*command.Command, *pipeline.Pipeline, error) {
	lexer := NewLexer(input)
	tokens, err := lexer.Tokenize()
	if err != nil {
		return nil, nil, err
	}

	parser := NewParser(tokens)
	return parser.Parse()
}

// expandGlobs expands glob patterns (*, ?, []) in arguments.
// Returns expanded list or error if pattern has no matches.
func expandGlobs(args []string) ([]string, error) {
	result := make([]string, 0, len(args))

	for _, arg := range args {
		// Check if argument contains glob pattern characters
		if containsGlobPattern(arg) {
			// Expand glob pattern
			matches, err := filepath.Glob(arg)
			if err != nil {
				return nil, shared.NewDomainError(
					"expandGlobs",
					shared.ErrInvalidArgument,
					fmt.Sprintf("invalid glob pattern '%s': %v", arg, err),
				)
			}

			// Bash behavior: no matches = error
			if len(matches) == 0 {
				return nil, shared.NewDomainError(
					"expandGlobs",
					shared.ErrInvalidArgument,
					fmt.Sprintf("no matches found for pattern '%s'", arg),
				)
			}

			// Add all matches
			result = append(result, matches...)
		} else {
			// Not a glob pattern, keep as-is
			result = append(result, arg)
		}
	}

	return result, nil
}

// containsGlobPattern checks if string contains glob pattern characters.
func containsGlobPattern(s string) bool {
	return strings.ContainsAny(s, "*?[]")
}
