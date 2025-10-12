package ports

import "io/fs"

// FileSystem - port for filesystem operations
type FileSystem interface {
	// Exists checks if a file/directory exists
	Exists(path string) (bool, error)

	// IsDir checks if the path is a directory
	IsDir(path string) (bool, error)

	// ReadFile reads file contents
	ReadFile(path string) ([]byte, error)

	// WriteFile writes contents to a file
	WriteFile(path string, data []byte, perm fs.FileMode) error

	// Stat returns file information
	Stat(path string) (fs.FileInfo, error)

	// ReadDir reads directory contents
	ReadDir(path string) ([]fs.DirEntry, error)
}
