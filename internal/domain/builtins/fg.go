package builtins

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/grpmsoft/gosh/internal/domain/session"
	"github.com/grpmsoft/gosh/internal/domain/shared"
)

// FgCommand represents the fg builtin command.
// Brings a background job to foreground.
type FgCommand struct {
	args    []string
	session *session.Session
}

// NewFgCommand creates a new fg command.
func NewFgCommand(args []string, sess *session.Session) (*FgCommand, error) {
	if sess == nil {
		return nil, shared.NewDomainError(
			"NewFgCommand",
			shared.ErrInvalidArgument,
			"session cannot be nil",
		)
	}

	return &FgCommand{
		args:    args,
		session: sess,
	}, nil
}

// Execute executes the fg command.
// Syntax: fg [%n] - brings job n to foreground, or most recent if no arg.
func (f *FgCommand) Execute() error {
	jobManager := f.session.JobManager()
	if jobManager == nil {
		return shared.NewDomainError(
			"Execute",
			shared.ErrInvalidState,
			"job manager not available",
		)
	}

	// Determine which job to bring to foreground
	var jobNumber int
	if len(f.args) == 0 {
		// No argument: use most recent active job
		activeJobs := jobManager.ListActiveJobs()
		if len(activeJobs) == 0 {
			return shared.NewDomainError(
				"Execute",
				shared.ErrInvalidArgument,
				"no current job",
			)
		}
		// Most recent is last in sorted list
		jobNumber = activeJobs[len(activeJobs)-1].JobNumber()
	} else {
		// Parse job number from argument (supports %n or just n)
		arg := f.args[0]
		arg = strings.TrimPrefix(arg, "%")

		var err error
		jobNumber, err = strconv.Atoi(arg)
		if err != nil {
			return shared.NewDomainError(
				"Execute",
				shared.ErrInvalidArgument,
				fmt.Sprintf("invalid job number: %s", f.args[0]),
			)
		}
	}

	// Get the job
	job, err := jobManager.GetJob(jobNumber)
	if err != nil {
		return err
	}

	// Bring to foreground
	if err := job.BringToForeground(); err != nil {
		return err
	}

	// If job is stopped, resume it
	if job.IsStopped() {
		if err := job.Resume(); err != nil {
			return err
		}
	}

	// TODO: This will need to integrate with process control
	// For now, just mark the job as foreground
	// Later, REPL will need to wait for this job to complete

	return nil
}
