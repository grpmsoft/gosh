package filesystem

import (
	"io/fs"
	"os"
)

// OSFileSystem - adapter for working with real filesystem
type OSFileSystem struct{}

// NewOSFileSystem creates a new filesystem adapter
func NewOSFileSystem() *OSFileSystem {
	return &OSFileSystem{}
}

// Exists checks if a file/directory exists
func (f *OSFileSystem) Exists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// IsDir checks if the path is a directory
func (f *OSFileSystem) IsDir(path string) (bool, error) {
	info, err := os.Stat(path)
	if err != nil {
		return false, err
	}
	return info.IsDir(), nil
}

// ReadFile reads file contents
func (f *OSFileSystem) ReadFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}

// WriteFile writes contents to a file
func (f *OSFileSystem) WriteFile(path string, data []byte, perm fs.FileMode) error {
	return os.WriteFile(path, data, perm)
}

// Stat returns file information
func (f *OSFileSystem) Stat(path string) (fs.FileInfo, error) {
	return os.Stat(path)
}

// ReadDir reads directory contents
func (f *OSFileSystem) ReadDir(path string) ([]fs.DirEntry, error) {
	return os.ReadDir(path)
}
