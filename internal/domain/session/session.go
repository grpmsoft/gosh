package session

import (
	"errors"
	"github.com/grpmsoft/gosh/internal/domain/history"
	"github.com/grpmsoft/gosh/internal/domain/job"
	"github.com/grpmsoft/gosh/internal/domain/process"
	"github.com/grpmsoft/gosh/internal/domain/shared"
	"path/filepath"
	"sync"
	"time"
)

// Session represents a shell session (Aggregate Root)
// Manages shell state: current directory, environment variables, history
type Session struct {
	id              string
	workingDir      string
	previousDir     string // For cd - command
	environment     shared.Environment
	commandHistory  *history.History // Rich domain model for command history
	jobManager      *job.JobManager  // Background job management
	processes       map[string]*process.Process
	variables       map[string]string
	aliases         map[string]string
	startTime       time.Time
	lastCommandTime time.Time
	active          bool
	mu              sync.RWMutex
}

// NewSession creates a new session
func NewSession(id string, workingDir string, env shared.Environment) (*Session, error) {
	if id == "" {
		return nil, shared.NewDomainError(
			"NewSession",
			shared.ErrInvalidArgument,
			"session id cannot be empty",
		)
	}

	if workingDir == "" {
		return nil, shared.NewDomainError(
			"NewSession",
			shared.ErrInvalidPath,
			"working directory cannot be empty",
		)
	}

	// Normalize the path
	absPath, err := filepath.Abs(workingDir)
	if err != nil {
		return nil, shared.NewDomainError(
			"NewSession",
			shared.ErrInvalidPath,
			"invalid working directory: "+err.Error(),
		)
	}

	// Create history with default config (with persistence enabled)
	historyConfig := history.DefaultConfig()
	historyConfig.SaveToFile = true

	return &Session{
		id:             id,
		workingDir:     absPath,
		environment:    env.Clone(),
		commandHistory: history.NewHistory(historyConfig),
		jobManager:     job.NewJobManager(),
		processes:      make(map[string]*process.Process),
		variables:      make(map[string]string),
		aliases:        make(map[string]string),
		startTime:      time.Now(),
		active:         true,
	}, nil
}

// ID returns the session identifier
func (s *Session) ID() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.id
}

// WorkingDirectory returns the current working directory
func (s *Session) WorkingDirectory() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.workingDir
}

// PreviousDirectory returns the previous working directory (for cd -)
func (s *Session) PreviousDirectory() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.previousDir
}

// ChangeDirectory changes the working directory (business rule)
func (s *Session) ChangeDirectory(path string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.active {
		return shared.ErrSessionClosed
	}

	// Handle relative paths
	var newPath string
	if filepath.IsAbs(path) {
		newPath = path
	} else {
		newPath = filepath.Join(s.workingDir, path)
	}

	// Normalize the path
	absPath, err := filepath.Abs(newPath)
	if err != nil {
		return shared.NewDomainError(
			"ChangeDirectory",
			shared.ErrInvalidPath,
			err.Error(),
		)
	}

	// Save the current directory as previous before changing
	s.previousDir = s.workingDir
	s.workingDir = absPath
	return nil
}

// Environment returns a copy of environment variables
func (s *Session) Environment() shared.Environment {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.environment.Clone()
}

// SetEnvironmentVariable sets an environment variable
func (s *Session) SetEnvironmentVariable(key, value string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.active {
		return shared.ErrSessionClosed
	}

	if key == "" {
		return shared.NewDomainError(
			"SetEnvironmentVariable",
			shared.ErrInvalidArgument,
			"key cannot be empty",
		)
	}

	s.environment.Set(key, value)
	return nil
}

// UnsetEnvironmentVariable removes an environment variable
func (s *Session) UnsetEnvironmentVariable(key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.active {
		return shared.ErrSessionClosed
	}

	s.environment.Unset(key)
	return nil
}

// AddToHistory adds a command to history
func (s *Session) AddToHistory(command string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.active {
		return shared.ErrSessionClosed
	}

	// Delegate to History domain model
	if err := s.commandHistory.Add(command); err != nil {
		// Only return error for real failures, not for empty commands
		if !errors.Is(err, history.ErrEmptyCommand) {
			return err
		}
		// Empty commands are silently ignored
		return nil
	}

	s.lastCommandTime = time.Now()
	return nil
}

// History returns the History aggregate for use by application layer
func (s *Session) History() *history.History {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.commandHistory
}

// JobManager returns the JobManager for background job control
func (s *Session) JobManager() *job.JobManager {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.jobManager
}

// GetHistoryRecent returns recent commands (convenience method for REPL)
func (s *Session) GetHistoryRecent(n int) []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.commandHistory.GetRecent(n)
}

// NewHistoryNavigator creates a new navigator for Up/Down arrow keys
func (s *Session) NewHistoryNavigator() *history.Navigator {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.commandHistory.NewNavigator()
}

// RegisterProcess registers a process in the session
func (s *Session) RegisterProcess(proc *process.Process) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.active {
		return shared.ErrSessionClosed
	}

	if proc == nil {
		return shared.NewDomainError(
			"RegisterProcess",
			shared.ErrInvalidArgument,
			"process cannot be nil",
		)
	}

	s.processes[proc.ID()] = proc
	return nil
}

// GetProcess retrieves a process by ID
func (s *Session) GetProcess(id string) (*process.Process, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	proc, ok := s.processes[id]
	if !ok {
		return nil, shared.NewDomainError(
			"GetProcess",
			shared.ErrProcessFailed,
			"process not found",
		)
	}

	return proc, nil
}

// RunningProcesses returns a list of running processes
func (s *Session) RunningProcesses() []*process.Process {
	s.mu.RLock()
	defer s.mu.RUnlock()

	running := make([]*process.Process, 0)
	for _, proc := range s.processes {
		if proc.IsRunning() {
			running = append(running, proc)
		}
	}

	return running
}

// SetVariable sets a shell variable
func (s *Session) SetVariable(key, value string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.active {
		return shared.ErrSessionClosed
	}

	if key == "" {
		return shared.NewDomainError(
			"SetVariable",
			shared.ErrInvalidArgument,
			"key cannot be empty",
		)
	}

	s.variables[key] = value
	return nil
}

// GetVariable retrieves a shell variable
func (s *Session) GetVariable(key string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	value, ok := s.variables[key]
	return value, ok
}

// SetAlias sets an alias
func (s *Session) SetAlias(name, command string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.active {
		return shared.ErrSessionClosed
	}

	if name == "" {
		return shared.NewDomainError(
			"SetAlias",
			shared.ErrInvalidArgument,
			"alias name cannot be empty",
		)
	}

	s.aliases[name] = command
	return nil
}

// GetAlias retrieves an alias
func (s *Session) GetAlias(name string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	value, ok := s.aliases[name]
	return value, ok
}

// GetAllAliases returns a copy of all aliases
func (s *Session) GetAllAliases() map[string]string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Return a copy for safety
	aliasesCopy := make(map[string]string, len(s.aliases))
	for k, v := range s.aliases {
		aliasesCopy[k] = v
	}
	return aliasesCopy
}

// RemoveAlias removes an alias
func (s *Session) RemoveAlias(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.active {
		return shared.ErrSessionClosed
	}

	if name == "" {
		return shared.NewDomainError(
			"RemoveAlias",
			shared.ErrInvalidArgument,
			"alias name cannot be empty",
		)
	}

	delete(s.aliases, name)
	return nil
}

// Close closes the session
func (s *Session) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.active {
		return shared.ErrSessionClosed
	}

	s.active = false
	return nil
}

// IsActive checks if the session is active
func (s *Session) IsActive() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.active
}

// StartTime returns the session start time
func (s *Session) StartTime() time.Time {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.startTime
}

// LastCommandTime returns the time of the last command
func (s *Session) LastCommandTime() time.Time {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.lastCommandTime
}

// Duration returns the session duration
func (s *Session) Duration() time.Duration {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return time.Since(s.startTime)
}
