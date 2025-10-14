package config

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/grpmsoft/gosh/internal/domain/shared"
)

// GoshrcService manages the .goshrc file.
// File format:
//
//	# Comments start with #
//	alias name='command'
//	alias name=command
//	export VAR=value
type GoshrcService struct {
	filePath string
}

// NewGoshrcService creates a new service for working with .goshrc.
func NewGoshrcService(filePath string) *GoshrcService {
	return &GoshrcService{
		filePath: filePath,
	}
}

// GetDefaultGoshrcPath returns the default path to .goshrc.
func GetDefaultGoshrcPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(homeDir, ".goshrc"), nil
}

// GoshrcData contains data from .goshrc.
type GoshrcData struct {
	Aliases     map[string]string // alias name -> command
	Environment map[string]string // env var name -> value
}

// Load loads configuration from .goshrc.
func (s *GoshrcService) Load() (*GoshrcData, error) {
	data := &GoshrcData{
		Aliases:     make(map[string]string),
		Environment: make(map[string]string),
	}

	// Check if file exists
	if _, err := os.Stat(s.filePath); os.IsNotExist(err) {
		// File doesn't exist - return empty configuration (not an error)
		return data, nil
	}

	// Open file
	file, err := os.Open(s.filePath)
	if err != nil {
		return nil, shared.NewDomainError(
			"GoshrcService.Load",
			shared.ErrInvalidArgument,
			fmt.Sprintf("failed to open .goshrc: %v", err),
		)
	}
	defer func() { _ = file.Close() }()

	// Patterns for parsing
	aliasPattern := regexp.MustCompile(`^alias\s+(\w+)=(.+)$`)
	exportPattern := regexp.MustCompile(`^export\s+(\w+)=(.+)$`)

	// Read line by line
	scanner := bufio.NewScanner(file)
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Parse alias
		if matches := aliasPattern.FindStringSubmatch(line); matches != nil {
			name := matches[1]
			command := matches[2]

			// Remove quotes from command
			command = unquoteValue(command)

			data.Aliases[name] = command
			continue
		}

		// Parse export (for future use)
		if matches := exportPattern.FindStringSubmatch(line); matches != nil {
			name := matches[1]
			value := matches[2]

			// Remove quotes from value
			value = unquoteValue(value)

			data.Environment[name] = value
			continue
		}

		// Unknown line - ignore or log
		// (don't return error to be resilient to format changes)
	}

	if err := scanner.Err(); err != nil {
		return nil, shared.NewDomainError(
			"GoshrcService.Load",
			shared.ErrInvalidArgument,
			fmt.Sprintf("failed to read .goshrc: %v", err),
		)
	}

	return data, nil
}

// Save saves configuration to .goshrc.
func (s *GoshrcService) Save(data *GoshrcData) error {
	// Create temporary file
	tmpFile := s.filePath + ".tmp"

	file, err := os.Create(tmpFile) //nolint:gosec // G304: Controlled by user config file path
	if err != nil {
		return shared.NewDomainError(
			"GoshrcService.Save",
			shared.ErrInvalidArgument,
			fmt.Sprintf("failed to create .goshrc: %v", err),
		)
	}
	defer func() { _ = file.Close() }()

	// Write header
	_, _ = fmt.Fprintln(file, "# .goshrc - GoSh configuration file")
	_, _ = fmt.Fprintln(file, "# Auto-loaded on shell startup")
	_, _ = fmt.Fprintln(file)

	// Write aliases
	if len(data.Aliases) > 0 {
		_, _ = fmt.Fprintln(file, "# Aliases")
		for name, command := range data.Aliases {
			// Always use single quotes for consistency
			_, _ = fmt.Fprintf(file, "alias %s='%s'\n", name, command)
		}
		_, _ = fmt.Fprintln(file)
	}

	// Write environment variables (for future use)
	if len(data.Environment) > 0 {
		_, _ = fmt.Fprintln(file, "# Environment variables")
		for name, value := range data.Environment {
			// Always use single quotes for consistency
			_, _ = fmt.Fprintf(file, "export %s='%s'\n", name, value)
		}
		_, _ = fmt.Fprintln(file)
	}

	// Close file before renaming
	_ = file.Close()

	// Atomically replace old file with new one
	if err := os.Rename(tmpFile, s.filePath); err != nil {
		// Try to delete temporary file
		_ = os.Remove(tmpFile)
		return shared.NewDomainError(
			"GoshrcService.Save",
			shared.ErrInvalidArgument,
			fmt.Sprintf("failed to save .goshrc: %v", err),
		)
	}

	return nil
}

// unquoteValue removes single or double quotes from value.
func unquoteValue(value string) string {
	value = strings.TrimSpace(value)

	// Remove double quotes
	if strings.HasPrefix(value, "\"") && strings.HasSuffix(value, "\"") && len(value) >= 2 {
		return value[1 : len(value)-1]
	}

	// Remove single quotes
	if strings.HasPrefix(value, "'") && strings.HasSuffix(value, "'") && len(value) >= 2 {
		return value[1 : len(value)-1]
	}

	return value
}
