package builtins

import (
	"github.com/grpmsoft/gosh/internal/domain/session"
	"github.com/grpmsoft/gosh/internal/domain/shared"
	"os"
	"path/filepath"
	"strings"
)

// CdCommand represents the cd (change directory) command
// Supports:
// - cd          - go to HOME
// - cd <path>   - go to specified directory
// - cd ~        - go to HOME
// - cd ~/path   - go to HOME subdirectory
// - cd -        - go to previous directory
type CdCommand struct {
	targetPath string
	session    *session.Session
}

// NewCdCommand creates a new cd command
func NewCdCommand(args []string, sess *session.Session) (*CdCommand, error) {
	if sess == nil {
		return nil, shared.NewDomainError(
			"NewCdCommand",
			shared.ErrInvalidArgument,
			"session cannot be nil",
		)
	}

	var targetPath string
	if len(args) == 0 {
		// cd without arguments - go to HOME
		targetPath = ""
	} else {
		targetPath = args[0]
	}

	return &CdCommand{
		targetPath: targetPath,
		session:    sess,
	}, nil
}

// Execute executes the cd command
func (c *CdCommand) Execute() error {
	// Determine target directory
	targetDir, err := c.resolveTargetDirectory()
	if err != nil {
		return err
	}

	// Validate directory exists
	if err := c.validateDirectory(targetDir); err != nil {
		return err
	}

	// Normalize path
	absPath, err := filepath.Abs(targetDir)
	if err != nil {
		return shared.NewDomainError(
			"cd",
			shared.ErrInvalidPath,
			"invalid path: "+err.Error(),
		)
	}

	// Change directory in session (saves previous directory)
	if err := c.session.ChangeDirectory(absPath); err != nil {
		return err
	}

	// Change actual process directory
	if err := os.Chdir(absPath); err != nil {
		return shared.NewDomainError(
			"cd",
			shared.ErrInvalidPath,
			"failed to change directory: "+err.Error(),
		)
	}

	return nil
}

// resolveTargetDirectory determines the target directory considering special symbols
func (c *CdCommand) resolveTargetDirectory() (string, error) {
	// Case 1: cd without arguments - go to HOME
	if c.targetPath == "" {
		return c.getHomeDirectory()
	}

	// Case 2: cd - - go to previous directory
	if c.targetPath == "-" {
		previousDir := c.session.PreviousDirectory()
		if previousDir == "" {
			return "", shared.NewDomainError(
				"cd",
				shared.ErrInvalidPath,
				"OLDPWD not set",
			)
		}
		// Output directory like in bash
		// (in real shell this is done via stdout, but here we simplify)
		return previousDir, nil
	}

	// Case 3: cd ~ or cd ~/path - tilde expansion
	if strings.HasPrefix(c.targetPath, "~") {
		homeDir, err := c.getHomeDirectory()
		if err != nil {
			return "", err
		}

		// Replace ~ with home directory
		if c.targetPath == "~" {
			return homeDir, nil
		}
		if strings.HasPrefix(c.targetPath, "~/") || strings.HasPrefix(c.targetPath, "~\\") {
			return filepath.Join(homeDir, c.targetPath[2:]), nil
		}
	}

	// Case 4: regular path (absolute or relative)
	if filepath.IsAbs(c.targetPath) {
		return c.targetPath, nil
	}

	// Relative path - resolve relative to current directory
	return filepath.Join(c.session.WorkingDirectory(), c.targetPath), nil
}

// getHomeDirectory returns the user's home directory
func (c *CdCommand) getHomeDirectory() (string, error) {
	// First try $HOME (Unix)
	if homeDir, ok := c.session.Environment().Get("HOME"); ok && homeDir != "" {
		return homeDir, nil
	}

	// Then try %USERPROFILE% (Windows)
	if homeDir, ok := c.session.Environment().Get("USERPROFILE"); ok && homeDir != "" {
		return homeDir, nil
	}

	// Fallback to os.UserHomeDir()
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", shared.NewDomainError(
			"cd",
			shared.ErrInvalidPath,
			"HOME directory not found: "+err.Error(),
		)
	}

	return homeDir, nil
}

// validateDirectory checks that the path exists and is a directory
func (c *CdCommand) validateDirectory(path string) error {
	// Check existence
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return shared.NewDomainError(
				"cd",
				shared.ErrDirectoryNotFound,
				"no such file or directory: "+path,
			)
		}
		return shared.NewDomainError(
			"cd",
			shared.ErrInvalidPath,
			"cannot access directory: "+err.Error(),
		)
	}

	// Check that it's a directory
	if !info.IsDir() {
		return shared.NewDomainError(
			"cd",
			shared.ErrInvalidPath,
			"not a directory: "+path,
		)
	}

	return nil
}
