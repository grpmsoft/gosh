package job_test

import (
	"testing"

	"github.com/grpmsoft/gosh/internal/domain/command"
	"github.com/grpmsoft/gosh/internal/domain/job"
	"github.com/grpmsoft/gosh/internal/domain/process"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewJob tests creating a new job
func TestNewJob(t *testing.T) {
	t.Run("creates job with valid parameters", func(t *testing.T) {
		cmd, _ := command.NewCommand("sleep", []string{"10"}, command.TypeExternal)
		proc, _ := process.NewProcess("proc1", cmd)

		j, err := job.NewJob("job1", 1, cmd, proc)

		require.NoError(t, err)
		require.NotNil(t, j)
		assert.Equal(t, "job1", j.ID())
		assert.Equal(t, 1, j.JobNumber())
		assert.Equal(t, job.StateRunning, j.State())
		assert.False(t, j.IsForeground())
		assert.True(t, j.IsBackground())
		assert.True(t, j.IsRunning())
		assert.False(t, j.IsFinished())
	})

	t.Run("returns error for empty id", func(t *testing.T) {
		cmd, _ := command.NewCommand("sleep", []string{"10"}, command.TypeExternal)
		proc, _ := process.NewProcess("proc1", cmd)

		j, err := job.NewJob("", 1, cmd, proc)

		assert.Error(t, err)
		assert.Nil(t, j)
		assert.Contains(t, err.Error(), "job id cannot be empty")
	})

	t.Run("returns error for invalid job number", func(t *testing.T) {
		cmd, _ := command.NewCommand("sleep", []string{"10"}, command.TypeExternal)
		proc, _ := process.NewProcess("proc1", cmd)

		j, err := job.NewJob("job1", 0, cmd, proc)

		assert.Error(t, err)
		assert.Nil(t, j)
		assert.Contains(t, err.Error(), "job number must be positive")
	})

	t.Run("returns error for nil command", func(t *testing.T) {
		cmd, _ := command.NewCommand("sleep", []string{"10"}, command.TypeExternal)
		proc, _ := process.NewProcess("proc1", cmd)

		j, err := job.NewJob("job1", 1, nil, proc)

		assert.Error(t, err)
		assert.Nil(t, j)
		assert.Contains(t, err.Error(), "command cannot be nil")
	})

	t.Run("returns error for nil process", func(t *testing.T) {
		cmd, _ := command.NewCommand("sleep", []string{"10"}, command.TypeExternal)

		j, err := job.NewJob("job1", 1, cmd, nil)

		assert.Error(t, err)
		assert.Nil(t, j)
		assert.Contains(t, err.Error(), "process cannot be nil")
	})
}

// TestJob_Stop tests stopping a job
func TestJob_Stop(t *testing.T) {
	t.Run("stops running job", func(t *testing.T) {
		cmd, _ := command.NewCommand("sleep", []string{"10"}, command.TypeExternal)
		proc, _ := process.NewProcess("proc1", cmd)
		j, _ := job.NewJob("job1", 1, cmd, proc)

		err := j.Stop()

		require.NoError(t, err)
		assert.Equal(t, job.StateStopped, j.State())
		assert.True(t, j.IsStopped())
	})

	t.Run("returns error when stopping non-running job", func(t *testing.T) {
		cmd, _ := command.NewCommand("sleep", []string{"10"}, command.TypeExternal)
		proc, _ := process.NewProcess("proc1", cmd)
		j, _ := job.NewJob("job1", 1, cmd, proc)
		j.Stop()

		err := j.Stop()

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "job is not running")
	})
}

// TestJob_Resume tests resuming a job
func TestJob_Resume(t *testing.T) {
	t.Run("resumes stopped job", func(t *testing.T) {
		cmd, _ := command.NewCommand("sleep", []string{"10"}, command.TypeExternal)
		proc, _ := process.NewProcess("proc1", cmd)
		j, _ := job.NewJob("job1", 1, cmd, proc)
		j.Stop()

		err := j.Resume()

		require.NoError(t, err)
		assert.Equal(t, job.StateRunning, j.State())
		assert.True(t, j.IsRunning())
	})

	t.Run("returns error when resuming non-stopped job", func(t *testing.T) {
		cmd, _ := command.NewCommand("sleep", []string{"10"}, command.TypeExternal)
		proc, _ := process.NewProcess("proc1", cmd)
		j, _ := job.NewJob("job1", 1, cmd, proc)

		err := j.Resume()

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "job is not stopped")
	})
}

// TestJob_Complete tests successful job completion
func TestJob_Complete(t *testing.T) {
	t.Run("completes running job", func(t *testing.T) {
		cmd, _ := command.NewCommand("sleep", []string{"10"}, command.TypeExternal)
		proc, _ := process.NewProcess("proc1", cmd)
		j, _ := job.NewJob("job1", 1, cmd, proc)

		err := j.Complete()

		require.NoError(t, err)
		assert.Equal(t, job.StateCompleted, j.State())
		assert.True(t, j.IsFinished())
		assert.False(t, j.EndTime().IsZero())
	})

	t.Run("returns error when completing non-running job", func(t *testing.T) {
		cmd, _ := command.NewCommand("sleep", []string{"10"}, command.TypeExternal)
		proc, _ := process.NewProcess("proc1", cmd)
		j, _ := job.NewJob("job1", 1, cmd, proc)
		j.Complete()

		err := j.Complete()

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "job is not running")
	})
}

// TestJob_Fail tests job completion with error
func TestJob_Fail(t *testing.T) {
	t.Run("fails running job", func(t *testing.T) {
		cmd, _ := command.NewCommand("sleep", []string{"10"}, command.TypeExternal)
		proc, _ := process.NewProcess("proc1", cmd)
		j, _ := job.NewJob("job1", 1, cmd, proc)

		err := j.Fail()

		require.NoError(t, err)
		assert.Equal(t, job.StateFailed, j.State())
		assert.True(t, j.IsFinished())
		assert.False(t, j.EndTime().IsZero())
	})
}

// TestJob_BringToForeground tests bringing job to foreground
func TestJob_BringToForeground(t *testing.T) {
	t.Run("brings background job to foreground", func(t *testing.T) {
		cmd, _ := command.NewCommand("sleep", []string{"10"}, command.TypeExternal)
		proc, _ := process.NewProcess("proc1", cmd)
		j, _ := job.NewJob("job1", 1, cmd, proc)

		err := j.BringToForeground()

		require.NoError(t, err)
		assert.True(t, j.IsForeground())
		assert.False(t, j.IsBackground())
	})

	t.Run("brings stopped job to foreground", func(t *testing.T) {
		cmd, _ := command.NewCommand("sleep", []string{"10"}, command.TypeExternal)
		proc, _ := process.NewProcess("proc1", cmd)
		j, _ := job.NewJob("job1", 1, cmd, proc)
		j.Stop()

		err := j.BringToForeground()

		require.NoError(t, err)
		assert.True(t, j.IsForeground())
	})

	t.Run("returns error for finished job", func(t *testing.T) {
		cmd, _ := command.NewCommand("sleep", []string{"10"}, command.TypeExternal)
		proc, _ := process.NewProcess("proc1", cmd)
		j, _ := job.NewJob("job1", 1, cmd, proc)
		j.Complete()

		err := j.BringToForeground()

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot bring finished job to foreground")
	})
}

// TestJob_SendToBackground tests sending job to background
func TestJob_SendToBackground(t *testing.T) {
	t.Run("sends foreground job to background", func(t *testing.T) {
		cmd, _ := command.NewCommand("sleep", []string{"10"}, command.TypeExternal)
		proc, _ := process.NewProcess("proc1", cmd)
		j, _ := job.NewJob("job1", 1, cmd, proc)
		j.BringToForeground()

		err := j.SendToBackground()

		require.NoError(t, err)
		assert.True(t, j.IsBackground())
		assert.False(t, j.IsForeground())
	})

	t.Run("returns error for finished job", func(t *testing.T) {
		cmd, _ := command.NewCommand("sleep", []string{"10"}, command.TypeExternal)
		proc, _ := process.NewProcess("proc1", cmd)
		j, _ := job.NewJob("job1", 1, cmd, proc)
		j.Complete()

		err := j.SendToBackground()

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot send finished job to background")
	})
}

// TestJob_StatusLine tests status formatting
func TestJob_StatusLine(t *testing.T) {
	t.Run("formats status for background running job", func(t *testing.T) {
		cmd, _ := command.NewCommand("sleep", []string{"10"}, command.TypeExternal)
		proc, _ := process.NewProcess("proc1", cmd)
		j, _ := job.NewJob("job1", 1, cmd, proc)

		status := j.StatusLine()

		assert.Contains(t, status, "[Running]")
		assert.Contains(t, status, "sleep 10")
	})

	t.Run("formats status for foreground job with + marker", func(t *testing.T) {
		cmd, _ := command.NewCommand("sleep", []string{"10"}, command.TypeExternal)
		proc, _ := process.NewProcess("proc1", cmd)
		j, _ := job.NewJob("job1", 1, cmd, proc)
		j.BringToForeground()

		status := j.StatusLine()

		assert.Contains(t, status, "+")
		assert.Contains(t, status, "[Running]")
	})

	t.Run("formats status for stopped job", func(t *testing.T) {
		cmd, _ := command.NewCommand("sleep", []string{"10"}, command.TypeExternal)
		proc, _ := process.NewProcess("proc1", cmd)
		j, _ := job.NewJob("job1", 1, cmd, proc)
		j.Stop()

		status := j.StatusLine()

		assert.Contains(t, status, "[Stopped]")
	})

	t.Run("formats status for completed job", func(t *testing.T) {
		cmd, _ := command.NewCommand("sleep", []string{"10"}, command.TypeExternal)
		proc, _ := process.NewProcess("proc1", cmd)
		j, _ := job.NewJob("job1", 1, cmd, proc)
		j.Complete()

		status := j.StatusLine()

		assert.Contains(t, status, "[Done]")
	})
}
