// Package process provides domain models for OS process lifecycle management.
package process

import (
	"time"

	"github.com/grpmsoft/gosh/internal/domain/command"
	"github.com/grpmsoft/gosh/internal/domain/shared"
)

// State represents the process state.
type State int

const (
	// StateCreated indicates the process has been created but not started.
	StateCreated State = iota
	// StateRunning indicates the process is currently running.
	StateRunning
	// StateCompleted indicates the process completed successfully.
	StateCompleted
	// StateFailed indicates the process failed with an error.
	StateFailed
	// StateTerminated indicates the process was forcibly terminated.
	StateTerminated
)

// String returns the string representation of the state.
func (s State) String() string {
	switch s {
	case StateCreated:
		return "created"
	case StateRunning:
		return "running"
	case StateCompleted:
		return "completed"
	case StateFailed:
		return "failed"
	case StateTerminated:
		return "terminated"
	default:
		return "unknown"
	}
}

// Process represents a running process (Entity).
// Has identity and mutable state.
type Process struct {
	id        string
	command   *command.Command
	state     State
	pid       int
	exitCode  shared.ExitCode
	startTime time.Time
	endTime   time.Time
	stdout    string
	stderr    string
	err       error
}

// NewProcess creates a new process.
func NewProcess(id string, cmd *command.Command) (*Process, error) {
	if id == "" {
		return nil, shared.NewDomainError(
			"NewProcess",
			shared.ErrInvalidArgument,
			"process id cannot be empty",
		)
	}

	if cmd == nil {
		return nil, shared.NewDomainError(
			"NewProcess",
			shared.ErrInvalidCommand,
			"command cannot be nil",
		)
	}

	return &Process{
		id:       id,
		command:  cmd.Clone(),
		state:    StateCreated,
		exitCode: shared.ExitSuccess,
	}, nil
}

// ID returns the process identifier.
func (p *Process) ID() string {
	return p.id
}

// Command returns the process command.
func (p *Process) Command() *command.Command {
	return p.command.Clone()
}

// State returns the current process state.
func (p *Process) State() State {
	return p.state
}

// PID returns the process PID.
func (p *Process) PID() int {
	return p.pid
}

// ExitCode returns the exit code.
func (p *Process) ExitCode() shared.ExitCode {
	return p.exitCode
}

// StartTime returns the start time.
func (p *Process) StartTime() time.Time {
	return p.startTime
}

// EndTime returns the end time.
func (p *Process) EndTime() time.Time {
	return p.endTime
}

// Duration returns the execution duration.
func (p *Process) Duration() time.Duration {
	if p.endTime.IsZero() {
		return time.Since(p.startTime)
	}
	return p.endTime.Sub(p.startTime)
}

// Stdout returns the process stdout.
func (p *Process) Stdout() string {
	return p.stdout
}

// Stderr returns the process stderr.
func (p *Process) Stderr() string {
	return p.stderr
}

// Error returns the execution error.
func (p *Process) Error() error {
	return p.err
}

// Start begins process execution (business logic for state transition).
func (p *Process) Start(pid int) error {
	if p.state != StateCreated {
		return shared.NewDomainError(
			"Start",
			shared.ErrProcessFailed,
			"process already started",
		)
	}

	p.state = StateRunning
	p.pid = pid
	p.startTime = time.Now()
	return nil
}

// Complete completes the process successfully.
func (p *Process) Complete(exitCode shared.ExitCode, stdout, stderr string) error {
	if p.state != StateRunning {
		return shared.NewDomainError(
			"Complete",
			shared.ErrProcessFailed,
			"process is not running",
		)
	}

	p.state = StateCompleted
	p.exitCode = exitCode
	p.stdout = stdout
	p.stderr = stderr
	p.endTime = time.Now()
	return nil
}

// Fail completes the process with an error.
func (p *Process) Fail(err error, exitCode shared.ExitCode, stdout, stderr string) error {
	if p.state != StateRunning {
		return shared.NewDomainError(
			"Fail",
			shared.ErrProcessFailed,
			"process is not running",
		)
	}

	p.state = StateFailed
	p.exitCode = exitCode
	p.stdout = stdout
	p.stderr = stderr
	p.err = err
	p.endTime = time.Now()
	return nil
}

// Terminate forcibly terminates the process.
func (p *Process) Terminate() error {
	if p.state != StateRunning {
		return shared.NewDomainError(
			"Terminate",
			shared.ErrProcessFailed,
			"process is not running",
		)
	}

	p.state = StateTerminated
	p.endTime = time.Now()
	return nil
}

// IsRunning checks if the process is running.
func (p *Process) IsRunning() bool {
	return p.state == StateRunning
}

// IsCompleted checks if the process completed successfully.
func (p *Process) IsCompleted() bool {
	return p.state == StateCompleted
}

// IsFailed checks if the process failed.
func (p *Process) IsFailed() bool {
	return p.state == StateFailed
}

// IsTerminated checks if the process was terminated.
func (p *Process) IsTerminated() bool {
	return p.state == StateTerminated
}

// IsFinished checks if the process has finished in any state.
func (p *Process) IsFinished() bool {
	return p.state == StateCompleted || p.state == StateFailed || p.state == StateTerminated
}
