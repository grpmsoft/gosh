// Package shared provides common domain types and error definitions.
package shared

import (
	"errors"
	"fmt"
)

var (
	// ErrInvalidCommand indicates an invalid command was provided.
	ErrInvalidCommand = errors.New("invalid command")
	// ErrEmptyCommand indicates the command cannot be empty.
	ErrEmptyCommand = errors.New("command cannot be empty")
	// ErrCommandNotFound indicates the command was not found.
	ErrCommandNotFound = errors.New("command not found")
	// ErrInvalidArgument indicates an invalid argument was provided.
	ErrInvalidArgument = errors.New("invalid argument")
	// ErrInvalidState indicates an invalid state transition.
	ErrInvalidState = errors.New("invalid state")
	// ErrProcessFailed indicates process execution failed.
	ErrProcessFailed = errors.New("process execution failed")
	// ErrPipelineFailed indicates pipeline execution failed.
	ErrPipelineFailed = errors.New("pipeline execution failed")
	// ErrRedirectionFailed indicates redirection failed.
	ErrRedirectionFailed = errors.New("redirection failed")
	// ErrSessionClosed indicates the session is closed.
	ErrSessionClosed = errors.New("session is closed")
	// ErrInvalidPath indicates an invalid path was provided.
	ErrInvalidPath = errors.New("invalid path")
	// ErrPermissionDenied indicates permission was denied.
	ErrPermissionDenied = errors.New("permission denied")
	// ErrDirectoryNotFound indicates the directory was not found.
	ErrDirectoryNotFound = errors.New("directory not found")
	// ErrBuiltinFailed indicates a built-in command failed.
	ErrBuiltinFailed = errors.New("built-in command failed")
	// ErrSkipCommand is a sentinel error for parser to skip non-command tokens.
	ErrSkipCommand = errors.New("skip command")
)

// DomainError represents a domain error with context.
type DomainError struct {
	Op      string // Operation where the error occurred
	Err     error  // Original error
	Context string // Additional context
}

func (e *DomainError) Error() string {
	if e.Context != "" {
		return fmt.Sprintf("%s: %v (%s)", e.Op, e.Err, e.Context)
	}
	return fmt.Sprintf("%s: %v", e.Op, e.Err)
}

func (e *DomainError) Unwrap() error {
	return e.Err
}

// NewDomainError creates a new domain error.
func NewDomainError(op string, err error, context string) *DomainError {
	return &DomainError{
		Op:      op,
		Err:     err,
		Context: context,
	}
}
