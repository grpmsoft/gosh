package job

import (
	"sync"

	"github.com/google/uuid"
	"github.com/grpmsoft/gosh/internal/domain/command"
	"github.com/grpmsoft/gosh/internal/domain/process"
	"github.com/grpmsoft/gosh/internal/domain/shared"
)

// Manager manages background jobs (Domain Service).
// Thread-safe collection of Job entities.
type Manager struct {
	jobs         map[string]*Job // job ID -> Job
	jobsByNumber map[int]*Job    // job number -> Job
	nextNumber   int             // next available job number
	mu           sync.RWMutex    // protects concurrent access
}

// NewManager creates a new job manager.
func NewManager() *Manager {
	return &Manager{
		jobs:         make(map[string]*Job),
		jobsByNumber: make(map[int]*Job),
		nextNumber:   1,
	}
}

// AddJob adds a new background job.
// Automatically assigns job number and generates ID.
func (jm *Manager) AddJob(cmd *command.Command, proc *process.Process) (*Job, error) {
	if cmd == nil {
		return nil, shared.NewDomainError(
			"AddJob",
			shared.ErrInvalidCommand,
			"command cannot be nil",
		)
	}

	if proc == nil {
		return nil, shared.NewDomainError(
			"AddJob",
			shared.ErrInvalidArgument,
			"process cannot be nil",
		)
	}

	jm.mu.Lock()
	defer jm.mu.Unlock()

	// Generate unique ID
	jobID := uuid.New().String()

	// Assign job number
	jobNumber := jm.nextNumber
	jm.nextNumber++

	// Create job
	job, err := NewJob(jobID, jobNumber, cmd, proc)
	if err != nil {
		return nil, err
	}

	// Store job
	jm.jobs[jobID] = job
	jm.jobsByNumber[jobNumber] = job

	return job, nil
}

// GetJob returns job by job number (%1, %2, etc.).
func (jm *Manager) GetJob(jobNumber int) (*Job, error) {
	jm.mu.RLock()
	defer jm.mu.RUnlock()

	job, exists := jm.jobsByNumber[jobNumber]
	if !exists {
		return nil, shared.NewDomainError(
			"GetJob",
			shared.ErrInvalidArgument,
			"no such job",
		)
	}

	return job, nil
}

// GetJobByID returns job by UUID.
func (jm *Manager) GetJobByID(jobID string) (*Job, error) {
	jm.mu.RLock()
	defer jm.mu.RUnlock()

	job, exists := jm.jobs[jobID]
	if !exists {
		return nil, shared.NewDomainError(
			"GetJobByID",
			shared.ErrInvalidArgument,
			"no such job",
		)
	}

	return job, nil
}

// ListJobs returns all jobs (running, stopped, finished).
// Returns jobs sorted by job number.
func (jm *Manager) ListJobs() []*Job {
	jm.mu.RLock()
	defer jm.mu.RUnlock()

	// Collect all jobs
	jobs := make([]*Job, 0, len(jm.jobs))
	for _, job := range jm.jobs {
		jobs = append(jobs, job)
	}

	// Sort by job number (simple bubble sort for small collections)
	for i := 0; i < len(jobs)-1; i++ {
		for j := 0; j < len(jobs)-i-1; j++ {
			if jobs[j].JobNumber() > jobs[j+1].JobNumber() {
				jobs[j], jobs[j+1] = jobs[j+1], jobs[j]
			}
		}
	}

	return jobs
}

// ListActiveJobs returns only running and stopped jobs (excludes finished).
func (jm *Manager) ListActiveJobs() []*Job {
	jm.mu.RLock()
	defer jm.mu.RUnlock()

	// Collect active jobs
	jobs := make([]*Job, 0)
	for _, job := range jm.jobs {
		if !job.IsFinished() {
			jobs = append(jobs, job)
		}
	}

	// Sort by job number
	for i := 0; i < len(jobs)-1; i++ {
		for j := 0; j < len(jobs)-i-1; j++ {
			if jobs[j].JobNumber() > jobs[j+1].JobNumber() {
				jobs[j], jobs[j+1] = jobs[j+1], jobs[j]
			}
		}
	}

	return jobs
}

// RemoveFinishedJobs removes all finished jobs from tracking.
// Returns count of removed jobs.
func (jm *Manager) RemoveFinishedJobs() int {
	jm.mu.Lock()
	defer jm.mu.Unlock()

	count := 0
	for jobID, job := range jm.jobs {
		if job.IsFinished() {
			delete(jm.jobs, jobID)
			delete(jm.jobsByNumber, job.JobNumber())
			count++
		}
	}

	return count
}

// RemoveJob removes specific job by job number.
// Returns error if job doesn't exist or is still running.
func (jm *Manager) RemoveJob(jobNumber int) error {
	jm.mu.Lock()
	defer jm.mu.Unlock()

	job, exists := jm.jobsByNumber[jobNumber]
	if !exists {
		return shared.NewDomainError(
			"RemoveJob",
			shared.ErrInvalidArgument,
			"no such job",
		)
	}

	// Safety check: don't remove running jobs
	if job.IsRunning() {
		return shared.NewDomainError(
			"RemoveJob",
			shared.ErrInvalidState,
			"cannot remove running job",
		)
	}

	// Remove from both maps
	delete(jm.jobs, job.ID())
	delete(jm.jobsByNumber, jobNumber)

	return nil
}

// Count returns total number of tracked jobs.
func (jm *Manager) Count() int {
	jm.mu.RLock()
	defer jm.mu.RUnlock()

	return len(jm.jobs)
}

// HasJobs returns true if there are any tracked jobs.
func (jm *Manager) HasJobs() bool {
	jm.mu.RLock()
	defer jm.mu.RUnlock()

	return len(jm.jobs) > 0
}

// Clear removes all jobs (useful for cleanup/testing).
func (jm *Manager) Clear() {
	jm.mu.Lock()
	defer jm.mu.Unlock()

	jm.jobs = make(map[string]*Job)
	jm.jobsByNumber = make(map[int]*Job)
	jm.nextNumber = 1
}
