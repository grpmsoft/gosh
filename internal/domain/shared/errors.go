package shared

import (
	"errors"
	"fmt"
)

var (
	// Domain-level errors following DDD practices
	ErrInvalidCommand    = errors.New("invalid command")
	ErrEmptyCommand      = errors.New("command cannot be empty")
	ErrCommandNotFound   = errors.New("command not found")
	ErrInvalidArgument   = errors.New("invalid argument")
	ErrInvalidState      = errors.New("invalid state")
	ErrProcessFailed     = errors.New("process execution failed")
	ErrPipelineFailed    = errors.New("pipeline execution failed")
	ErrRedirectionFailed = errors.New("redirection failed")
	ErrSessionClosed     = errors.New("session is closed")
	ErrInvalidPath       = errors.New("invalid path")
	ErrPermissionDenied  = errors.New("permission denied")
	ErrDirectoryNotFound = errors.New("directory not found")
	ErrBuiltinFailed     = errors.New("built-in command failed")
)

// DomainError represents a domain error with context
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

// NewDomainError creates a new domain error
func NewDomainError(op string, err error, context string) *DomainError {
	return &DomainError{
		Op:      op,
		Err:     err,
		Context: context,
	}
}
