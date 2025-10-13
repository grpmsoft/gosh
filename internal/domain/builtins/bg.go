package builtins

import (
	"fmt"
	"github.com/grpmsoft/gosh/internal/domain/job"
	"github.com/grpmsoft/gosh/internal/domain/session"
	"github.com/grpmsoft/gosh/internal/domain/shared"
	"strconv"
	"strings"
)

// BgCommand represents the bg builtin command
// Resumes a stopped job in background
type BgCommand struct {
	args    []string
	session *session.Session
}

// NewBgCommand creates a new bg command
func NewBgCommand(args []string, sess *session.Session) (*BgCommand, error) {
	if sess == nil {
		return nil, shared.NewDomainError(
			"NewBgCommand",
			shared.ErrInvalidArgument,
			"session cannot be nil",
		)
	}

	return &BgCommand{
		args:    args,
		session: sess,
	}, nil
}

// Execute executes the bg command
// Syntax: bg [%n] - resumes job n in background, or most recent stopped job if no arg
func (b *BgCommand) Execute() error {
	jobManager := b.session.JobManager()
	if jobManager == nil {
		return shared.NewDomainError(
			"Execute",
			shared.ErrInvalidState,
			"job manager not available",
		)
	}

	// Determine which job to resume in background
	var jobNumber int
	if len(b.args) == 0 {
		// No argument: use most recent stopped job
		allJobs := jobManager.ListJobs()
		var stoppedJob *job.Job
		for i := len(allJobs) - 1; i >= 0; i-- {
			if allJobs[i].IsStopped() {
				stoppedJob = allJobs[i]
				break
			}
		}

		if stoppedJob == nil {
			return shared.NewDomainError(
				"Execute",
				shared.ErrInvalidArgument,
				"no stopped job",
			)
		}
		jobNumber = stoppedJob.JobNumber()
	} else {
		// Parse job number from argument (supports %n or just n)
		arg := b.args[0]
		arg = strings.TrimPrefix(arg, "%")

		var err error
		jobNumber, err = strconv.Atoi(arg)
		if err != nil {
			return shared.NewDomainError(
				"Execute",
				shared.ErrInvalidArgument,
				fmt.Sprintf("invalid job number: %s", b.args[0]),
			)
		}
	}

	// Get the job
	targetJob, err := jobManager.GetJob(jobNumber)
	if err != nil {
		return err
	}

	// Verify job is stopped
	if !targetJob.IsStopped() {
		return shared.NewDomainError(
			"Execute",
			shared.ErrInvalidState,
			fmt.Sprintf("job %d is not stopped", jobNumber),
		)
	}

	// Send to background (if it was foreground)
	if err := targetJob.SendToBackground(); err != nil {
		// Ignore error if already in background
		if !strings.Contains(err.Error(), "finished") {
			return err
		}
	}

	// Resume the job
	if err := targetJob.Resume(); err != nil {
		return err
	}

	// TODO: This will need to integrate with process control
	// For now, just mark the job as running in background
	// Later, need to send SIGCONT to the process

	return nil
}
