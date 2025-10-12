package job

import (
	"github.com/grpmsoft/gosh/internal/domain/command"
	"github.com/grpmsoft/gosh/internal/domain/process"
	"github.com/grpmsoft/gosh/internal/domain/shared"
	"time"
)

// State represents the state of a background job
type State int

const (
	StateRunning   State = iota // Running
	StateStopped                // Stopped (Ctrl+Z)
	StateCompleted              // Completed successfully
	StateFailed                 // Failed with error
)

// String returns the string representation of the state
func (s State) String() string {
	switch s {
	case StateRunning:
		return "Running"
	case StateStopped:
		return "Stopped"
	case StateCompleted:
		return "Done"
	case StateFailed:
		return "Exit"
	default:
		return "Unknown"
	}
}

// Job represents a background job (Entity)
// Has identity (id) and mutable state
type Job struct {
	id           string
	jobNumber    int // Number for user display (%1, %2, etc.)
	command      *command.Command
	process      *process.Process
	state        State
	startTime    time.Time
	endTime      time.Time
	isForeground bool // Is currently running in foreground
}

// NewJob creates a new background job
func NewJob(id string, jobNumber int, cmd *command.Command, proc *process.Process) (*Job, error) {
	if id == "" {
		return nil, shared.NewDomainError(
			"NewJob",
			shared.ErrInvalidArgument,
			"job id cannot be empty",
		)
	}

	if jobNumber <= 0 {
		return nil, shared.NewDomainError(
			"NewJob",
			shared.ErrInvalidArgument,
			"job number must be positive",
		)
	}

	if cmd == nil {
		return nil, shared.NewDomainError(
			"NewJob",
			shared.ErrInvalidCommand,
			"command cannot be nil",
		)
	}

	if proc == nil {
		return nil, shared.NewDomainError(
			"NewJob",
			shared.ErrInvalidArgument,
			"process cannot be nil",
		)
	}

	return &Job{
		id:           id,
		jobNumber:    jobNumber,
		command:      cmd.Clone(),
		process:      proc,
		state:        StateRunning,
		startTime:    time.Now(),
		isForeground: false, // Initially always in background
	}, nil
}

// ID returns the unique job identifier
func (j *Job) ID() string {
	return j.id
}

// JobNumber returns the job number for user display
func (j *Job) JobNumber() int {
	return j.jobNumber
}

// Command returns the job command
func (j *Job) Command() *command.Command {
	return j.command.Clone()
}

// Process returns the job process
func (j *Job) Process() *process.Process {
	return j.process
}

// State returns the current job state
func (j *Job) State() State {
	return j.state
}

// StartTime returns the start time
func (j *Job) StartTime() time.Time {
	return j.startTime
}

// EndTime returns the end time
func (j *Job) EndTime() time.Time {
	return j.endTime
}

// IsForeground checks if the job is running in foreground
func (j *Job) IsForeground() bool {
	return j.isForeground
}

// IsBackground checks if the job is running in background
func (j *Job) IsBackground() bool {
	return !j.isForeground
}

// IsRunning checks if the job is running
func (j *Job) IsRunning() bool {
	return j.state == StateRunning
}

// IsStopped checks if the job is stopped
func (j *Job) IsStopped() bool {
	return j.state == StateStopped
}

// IsFinished checks if the job is finished
func (j *Job) IsFinished() bool {
	return j.state == StateCompleted || j.state == StateFailed
}

// Stop stops the job (Ctrl+Z)
func (j *Job) Stop() error {
	if j.state != StateRunning {
		return shared.NewDomainError(
			"Stop",
			shared.ErrInvalidState,
			"job is not running",
		)
	}

	j.state = StateStopped
	return nil
}

// Resume resumes a stopped job
func (j *Job) Resume() error {
	if j.state != StateStopped {
		return shared.NewDomainError(
			"Resume",
			shared.ErrInvalidState,
			"job is not stopped",
		)
	}

	j.state = StateRunning
	return nil
}

// Complete completes the job successfully
func (j *Job) Complete() error {
	if j.state != StateRunning {
		return shared.NewDomainError(
			"Complete",
			shared.ErrInvalidState,
			"job is not running",
		)
	}

	j.state = StateCompleted
	j.endTime = time.Now()
	return nil
}

// Fail completes the job with an error
func (j *Job) Fail() error {
	if j.state != StateRunning {
		return shared.NewDomainError(
			"Fail",
			shared.ErrInvalidState,
			"job is not running",
		)
	}

	j.state = StateFailed
	j.endTime = time.Now()
	return nil
}

// BringToForeground brings the job to foreground (fg)
func (j *Job) BringToForeground() error {
	if j.IsFinished() {
		return shared.NewDomainError(
			"BringToForeground",
			shared.ErrInvalidState,
			"cannot bring finished job to foreground",
		)
	}

	j.isForeground = true
	return nil
}

// SendToBackground sends the job to background (bg)
func (j *Job) SendToBackground() error {
	if j.IsFinished() {
		return shared.NewDomainError(
			"SendToBackground",
			shared.ErrInvalidState,
			"cannot send finished job to background",
		)
	}

	j.isForeground = false
	return nil
}

// StatusLine returns a status line for display in jobs
func (j *Job) StatusLine() string {
	var status string
	if j.isForeground {
		status = "+"
	} else {
		status = " "
	}

	return status + " [" + j.state.String() + "] " + j.command.FullCommand()
}
