// Package session provides use cases for shell session management.
package session

import (
	"log/slog"
	"os"
	"sync"

	"github.com/grpmsoft/gosh/internal/domain/session"
	"github.com/grpmsoft/gosh/internal/domain/shared"
)

// Manager manages shell sessions.
type Manager struct {
	sessions map[string]*session.Session
	mu       sync.RWMutex
	logger   *slog.Logger
}

// NewManager creates a new session manager.
func NewManager(logger *slog.Logger) *Manager {
	return &Manager{
		sessions: make(map[string]*session.Session),
		logger:   logger,
	}
}

// CreateSession creates a new session.
func (sm *Manager) CreateSession(id string) (*session.Session, error) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	// Check if session already exists
	if _, exists := sm.sessions[id]; exists {
		return nil, shared.NewDomainError(
			"CreateSession",
			shared.ErrInvalidArgument,
			"session already exists",
		)
	}

	// Get current directory
	workingDir, err := os.Getwd()
	if err != nil {
		return nil, shared.NewDomainError(
			"CreateSession",
			shared.ErrInvalidPath,
			"failed to get working directory: "+err.Error(),
		)
	}

	// Get environment variables
	env := sm.getEnvironment()

	// Create session
	sess, err := session.NewSession(id, workingDir, env)
	if err != nil {
		return nil, err
	}

	sm.sessions[id] = sess
	sm.logger.Info("session created", "id", id, "workingDir", workingDir)

	return sess, nil
}

// GetSession gets a session by ID.
func (sm *Manager) GetSession(id string) (*session.Session, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	sess, exists := sm.sessions[id]
	if !exists {
		return nil, shared.NewDomainError(
			"GetSession",
			shared.ErrInvalidArgument,
			"session not found",
		)
	}

	return sess, nil
}

// CloseSession closes a session.
func (sm *Manager) CloseSession(id string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	sess, exists := sm.sessions[id]
	if !exists {
		return shared.NewDomainError(
			"CloseSession",
			shared.ErrInvalidArgument,
			"session not found",
		)
	}

	if err := sess.Close(); err != nil {
		return err
	}

	delete(sm.sessions, id)
	sm.logger.Info("session closed", "id", id)

	return nil
}

// ActiveSessions returns a list of active sessions.
func (sm *Manager) ActiveSessions() []*session.Session {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	active := make([]*session.Session, 0, len(sm.sessions))
	for _, sess := range sm.sessions {
		if sess.IsActive() {
			active = append(active, sess)
		}
	}

	return active
}

// getEnvironment gets environment variables.
func (sm *Manager) getEnvironment() shared.Environment {
	env := make(shared.Environment)
	for _, e := range os.Environ() {
		// Parse "KEY=VALUE"
		for i := 0; i < len(e); i++ {
			if e[i] == '=' {
				key := e[:i]
				value := e[i+1:]
				env[key] = value
				break
			}
		}
	}
	return env
}
