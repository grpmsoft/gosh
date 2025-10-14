package builtins

import (
	"fmt"
	"io"

	"github.com/grpmsoft/gosh/internal/domain/session"
	"github.com/grpmsoft/gosh/internal/domain/shared"
)

// JobsCommand represents the jobs builtin command.
// Lists all background jobs with their status.
type JobsCommand struct {
	session *session.Session
	stdout  io.Writer
}

// NewJobsCommand creates a new jobs command.
func NewJobsCommand(sess *session.Session, stdout io.Writer) (*JobsCommand, error) {
	if sess == nil {
		return nil, shared.NewDomainError(
			"NewJobsCommand",
			shared.ErrInvalidArgument,
			"session cannot be nil",
		)
	}
	if stdout == nil {
		return nil, shared.NewDomainError(
			"NewJobsCommand",
			shared.ErrInvalidArgument,
			"stdout cannot be nil",
		)
	}

	return &JobsCommand{
		session: sess,
		stdout:  stdout,
	}, nil
}

// Execute executes the jobs command.
// Lists all background jobs sorted by job number.
func (j *JobsCommand) Execute() error {
	jobManager := j.session.JobManager()
	if jobManager == nil {
		return shared.NewDomainError(
			"Execute",
			shared.ErrInvalidState,
			"job manager not available",
		)
	}

	jobs := jobManager.ListJobs()
	if len(jobs) == 0 {
		return nil // No jobs to display
	}

	// Display each job with format: [JobNum] Status Command
	for _, job := range jobs {
		_, _ = fmt.Fprintf(j.stdout, "[%d]%s\n", job.JobNumber(), job.StatusLine())
	}

	return nil
}
