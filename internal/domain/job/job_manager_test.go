package job_test

import (
	"sync"
	"testing"

	"github.com/grpmsoft/gosh/internal/domain/command"
	"github.com/grpmsoft/gosh/internal/domain/job"
	"github.com/grpmsoft/gosh/internal/domain/process"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewJobManager creates new job manager
func TestNewJobManager(t *testing.T) {
	jm := job.NewJobManager()

	require.NotNil(t, jm)
	assert.Equal(t, 0, jm.Count())
	assert.False(t, jm.HasJobs())
}

// TestJobManager_AddJob tests adding jobs
func TestJobManager_AddJob(t *testing.T) {
	t.Run("adds job and assigns job number", func(t *testing.T) {
		jm := job.NewJobManager()
		cmd, _ := command.NewCommand("sleep", []string{"10"}, command.TypeExternal)
		proc, _ := process.NewProcess("proc1", cmd)

		j, err := jm.AddJob(cmd, proc)

		require.NoError(t, err)
		require.NotNil(t, j)
		assert.Equal(t, 1, j.JobNumber())
		assert.Equal(t, 1, jm.Count())
		assert.True(t, jm.HasJobs())
	})

	t.Run("assigns sequential job numbers", func(t *testing.T) {
		jm := job.NewJobManager()

		cmd1, _ := command.NewCommand("sleep", []string{"10"}, command.TypeExternal)
		proc1, _ := process.NewProcess("proc1", cmd1)
		j1, _ := jm.AddJob(cmd1, proc1)

		cmd2, _ := command.NewCommand("sleep", []string{"20"}, command.TypeExternal)
		proc2, _ := process.NewProcess("proc2", cmd2)
		j2, _ := jm.AddJob(cmd2, proc2)

		assert.Equal(t, 1, j1.JobNumber())
		assert.Equal(t, 2, j2.JobNumber())
		assert.Equal(t, 2, jm.Count())
	})

	t.Run("returns error for nil command", func(t *testing.T) {
		jm := job.NewJobManager()
		cmd, _ := command.NewCommand("sleep", []string{"10"}, command.TypeExternal)
		proc, _ := process.NewProcess("proc1", cmd)

		j, err := jm.AddJob(nil, proc)

		assert.Error(t, err)
		assert.Nil(t, j)
		assert.Contains(t, err.Error(), "command cannot be nil")
	})

	t.Run("returns error for nil process", func(t *testing.T) {
		jm := job.NewJobManager()
		cmd, _ := command.NewCommand("sleep", []string{"10"}, command.TypeExternal)

		j, err := jm.AddJob(cmd, nil)

		assert.Error(t, err)
		assert.Nil(t, j)
		assert.Contains(t, err.Error(), "process cannot be nil")
	})
}

// TestJobManager_GetJob tests retrieving jobs by number
func TestJobManager_GetJob(t *testing.T) {
	t.Run("gets job by job number", func(t *testing.T) {
		jm := job.NewJobManager()
		cmd, _ := command.NewCommand("sleep", []string{"10"}, command.TypeExternal)
		proc, _ := process.NewProcess("proc1", cmd)
		added, _ := jm.AddJob(cmd, proc)

		retrieved, err := jm.GetJob(1)

		require.NoError(t, err)
		assert.Equal(t, added.ID(), retrieved.ID())
		assert.Equal(t, 1, retrieved.JobNumber())
	})

	t.Run("returns error for non-existent job number", func(t *testing.T) {
		jm := job.NewJobManager()

		j, err := jm.GetJob(999)

		assert.Error(t, err)
		assert.Nil(t, j)
		assert.Contains(t, err.Error(), "no such job")
	})
}

// TestJobManager_GetJobByID tests retrieving jobs by ID
func TestJobManager_GetJobByID(t *testing.T) {
	t.Run("gets job by ID", func(t *testing.T) {
		jm := job.NewJobManager()
		cmd, _ := command.NewCommand("sleep", []string{"10"}, command.TypeExternal)
		proc, _ := process.NewProcess("proc1", cmd)
		added, _ := jm.AddJob(cmd, proc)

		retrieved, err := jm.GetJobByID(added.ID())

		require.NoError(t, err)
		assert.Equal(t, added.ID(), retrieved.ID())
	})

	t.Run("returns error for non-existent job ID", func(t *testing.T) {
		jm := job.NewJobManager()

		j, err := jm.GetJobByID("non-existent-id")

		assert.Error(t, err)
		assert.Nil(t, j)
		assert.Contains(t, err.Error(), "no such job")
	})
}

// TestJobManager_ListJobs tests listing all jobs
func TestJobManager_ListJobs(t *testing.T) {
	t.Run("lists all jobs sorted by job number", func(t *testing.T) {
		jm := job.NewJobManager()

		cmd1, _ := command.NewCommand("sleep", []string{"10"}, command.TypeExternal)
		proc1, _ := process.NewProcess("proc1", cmd1)
		jm.AddJob(cmd1, proc1)

		cmd2, _ := command.NewCommand("sleep", []string{"20"}, command.TypeExternal)
		proc2, _ := process.NewProcess("proc2", cmd2)
		jm.AddJob(cmd2, proc2)

		jobs := jm.ListJobs()

		assert.Equal(t, 2, len(jobs))
		assert.Equal(t, 1, jobs[0].JobNumber())
		assert.Equal(t, 2, jobs[1].JobNumber())
	})

	t.Run("returns empty list when no jobs", func(t *testing.T) {
		jm := job.NewJobManager()

		jobs := jm.ListJobs()

		assert.Equal(t, 0, len(jobs))
	})
}

// TestJobManager_ListActiveJobs tests listing active jobs
func TestJobManager_ListActiveJobs(t *testing.T) {
	t.Run("lists only running and stopped jobs", func(t *testing.T) {
		jm := job.NewJobManager()

		// Add running job
		cmd1, _ := command.NewCommand("sleep", []string{"10"}, command.TypeExternal)
		proc1, _ := process.NewProcess("proc1", cmd1)
		jm.AddJob(cmd1, proc1)

		// Add stopped job
		cmd2, _ := command.NewCommand("sleep", []string{"20"}, command.TypeExternal)
		proc2, _ := process.NewProcess("proc2", cmd2)
		j2, _ := jm.AddJob(cmd2, proc2)
		j2.Stop()

		// Add finished job
		cmd3, _ := command.NewCommand("sleep", []string{"30"}, command.TypeExternal)
		proc3, _ := process.NewProcess("proc3", cmd3)
		j3, _ := jm.AddJob(cmd3, proc3)
		j3.Complete()

		activeJobs := jm.ListActiveJobs()

		assert.Equal(t, 2, len(activeJobs))
		assert.Equal(t, 1, activeJobs[0].JobNumber())
		assert.Equal(t, 2, activeJobs[1].JobNumber())
	})
}

// TestJobManager_RemoveFinishedJobs tests removing finished jobs
func TestJobManager_RemoveFinishedJobs(t *testing.T) {
	t.Run("removes all finished jobs", func(t *testing.T) {
		jm := job.NewJobManager()

		// Add running job
		cmd1, _ := command.NewCommand("sleep", []string{"10"}, command.TypeExternal)
		proc1, _ := process.NewProcess("proc1", cmd1)
		jm.AddJob(cmd1, proc1)

		// Add finished job
		cmd2, _ := command.NewCommand("sleep", []string{"20"}, command.TypeExternal)
		proc2, _ := process.NewProcess("proc2", cmd2)
		j2, _ := jm.AddJob(cmd2, proc2)
		j2.Complete()

		// Add failed job
		cmd3, _ := command.NewCommand("sleep", []string{"30"}, command.TypeExternal)
		proc3, _ := process.NewProcess("proc3", cmd3)
		j3, _ := jm.AddJob(cmd3, proc3)
		j3.Fail()

		count := jm.RemoveFinishedJobs()

		assert.Equal(t, 2, count)
		assert.Equal(t, 1, jm.Count())
	})

	t.Run("returns zero when no finished jobs", func(t *testing.T) {
		jm := job.NewJobManager()

		cmd, _ := command.NewCommand("sleep", []string{"10"}, command.TypeExternal)
		proc, _ := process.NewProcess("proc1", cmd)
		jm.AddJob(cmd, proc)

		count := jm.RemoveFinishedJobs()

		assert.Equal(t, 0, count)
		assert.Equal(t, 1, jm.Count())
	})
}

// TestJobManager_RemoveJob tests removing specific job
func TestJobManager_RemoveJob(t *testing.T) {
	t.Run("removes finished job by number", func(t *testing.T) {
		jm := job.NewJobManager()

		cmd, _ := command.NewCommand("sleep", []string{"10"}, command.TypeExternal)
		proc, _ := process.NewProcess("proc1", cmd)
		j, _ := jm.AddJob(cmd, proc)
		j.Complete()

		err := jm.RemoveJob(1)

		require.NoError(t, err)
		assert.Equal(t, 0, jm.Count())
	})

	t.Run("returns error when removing running job", func(t *testing.T) {
		jm := job.NewJobManager()

		cmd, _ := command.NewCommand("sleep", []string{"10"}, command.TypeExternal)
		proc, _ := process.NewProcess("proc1", cmd)
		jm.AddJob(cmd, proc)

		err := jm.RemoveJob(1)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot remove running job")
		assert.Equal(t, 1, jm.Count())
	})

	t.Run("returns error for non-existent job number", func(t *testing.T) {
		jm := job.NewJobManager()

		err := jm.RemoveJob(999)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no such job")
	})
}

// TestJobManager_Clear tests clearing all jobs
func TestJobManager_Clear(t *testing.T) {
	t.Run("clears all jobs", func(t *testing.T) {
		jm := job.NewJobManager()

		cmd, _ := command.NewCommand("sleep", []string{"10"}, command.TypeExternal)
		proc, _ := process.NewProcess("proc1", cmd)
		jm.AddJob(cmd, proc)

		jm.Clear()

		assert.Equal(t, 0, jm.Count())
		assert.False(t, jm.HasJobs())
	})
}

// TestJobManager_ConcurrentAccess tests thread-safety
func TestJobManager_ConcurrentAccess(t *testing.T) {
	t.Run("handles concurrent additions", func(t *testing.T) {
		jm := job.NewJobManager()
		var wg sync.WaitGroup

		// Add 100 jobs concurrently
		for i := 0; i < 100; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				cmd, _ := command.NewCommand("sleep", []string{"10"}, command.TypeExternal)
				proc, _ := process.NewProcess("proc1", cmd)
				jm.AddJob(cmd, proc)
			}()
		}

		wg.Wait()

		assert.Equal(t, 100, jm.Count())
	})

	t.Run("handles concurrent reads and writes", func(t *testing.T) {
		jm := job.NewJobManager()
		var wg sync.WaitGroup

		// Add initial jobs
		for i := 0; i < 10; i++ {
			cmd, _ := command.NewCommand("sleep", []string{"10"}, command.TypeExternal)
			proc, _ := process.NewProcess("proc1", cmd)
			jm.AddJob(cmd, proc)
		}

		// Concurrent reads
		for i := 0; i < 50; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				jm.ListJobs()
				jm.ListActiveJobs()
			}()
		}

		// Concurrent writes
		for i := 0; i < 50; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				cmd, _ := command.NewCommand("sleep", []string{"10"}, command.TypeExternal)
				proc, _ := process.NewProcess("proc1", cmd)
				jm.AddJob(cmd, proc)
			}()
		}

		wg.Wait()

		assert.Equal(t, 60, jm.Count())
	})
}
