// Package fileManagement provides utilities for file and directory operations.
// This package handles copying files and directories, preserving permissions,
// handling symlinks, and finding executable programs in the system PATH.
package fileManagement

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
)

// CopyDirectory recursively copies a directory tree from source to destination.
// This function:
//   - Preserves file permissions and ownership
//   - Handles directories, regular files, and symlinks
//   - Maintains the directory structure
//
// Parameters:
//   - scrDir: Source directory to copy from
//   - dest: Destination directory to copy to
//
// Returns an error if any file operation fails.
func CopyDirectory(scrDir, dest string) error {
	// Read all entries in the source directory
	entries, err := os.ReadDir(scrDir)
	if err != nil {
		return err
	}

	// Process each entry (file, directory, or symlink)
	for _, entry := range entries {
		sourcePath := filepath.Join(scrDir, entry.Name())
		destPath := filepath.Join(dest, entry.Name())

		// Get file information to determine type and permissions
		fileInfo, err := os.Stat(sourcePath)
		if err != nil {
			return err
		}

		// Get ownership information (UID/GID) for preserving file ownership
		stat, ok := fileInfo.Sys().(*syscall.Stat_t)
		if !ok {
			return fmt.Errorf("failed to get raw syscall.Stat_t data for '%s'", sourcePath)
		}

		// Handle different file types differently
		switch fileInfo.Mode() & os.ModeType {
		case os.ModeDir:
			// Recursively copy subdirectories
			if err := CreateIfNotExists(destPath, 0755); err != nil {
				return err
			}
			if err := CopyDirectory(sourcePath, destPath); err != nil {
				return err
			}
		case os.ModeSymlink:
			// Copy symlinks by recreating them (not following the link)
			if err := CopySymLink(sourcePath, destPath); err != nil {
				return err
			}
		default:
			// Copy regular files
			if err := Copy(sourcePath, destPath); err != nil {
				return err
			}
		}

		// Preserve file ownership (UID/GID)
		// Note: This may fail if running without appropriate permissions
		if err := os.Lchown(destPath, int(stat.Uid), int(stat.Gid)); err != nil {
			return err
		}

		// Get entry info for permissions
		fInfo, err := entry.Info()
		if err != nil {
			return err
		}

		// Preserve file permissions (but not for symlinks - they have their own permissions)
		isSymlink := fInfo.Mode()&os.ModeSymlink != 0
		if !isSymlink {
			if err := os.Chmod(destPath, fInfo.Mode()); err != nil {
				return err
			}
		}
	}
	return nil
}

// Copy copies a single file from source to destination.
// This is a simple file copy operation that doesn't preserve metadata.
// For preserving permissions and ownership, use CopyDirectory.
//
// Parameters:
//   - srcFile: Path to the source file
//   - dstFile: Path to the destination file
//
// Returns an error if the copy operation fails.
func Copy(srcFile, dstFile string) error {
	// Create the destination file
	out, err := os.Create(dstFile)
	if err != nil {
		return err
	}
	defer out.Close() // Ensure file is closed when function exits

	// Open the source file for reading
	in, err := os.Open(srcFile)
	if err != nil {
		return err
	}
	defer in.Close() // Ensure file is closed when function exits

	// Copy the file contents efficiently
	// io.Copy handles the transfer in chunks, even for large files
	_, err = io.Copy(out, in)
	if err != nil {
		return err
	}

	return nil
}

// Exists checks if a file or directory exists at the given path.
//
// Parameters:
//   - filePath: Path to check
//
// Returns true if the path exists, false otherwise.
func Exists(filePath string) bool {
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return false
	}

	return true
}

// CreateIfNotExists creates a directory if it doesn't already exist.
// This is an idempotent operation - it's safe to call multiple times.
//
// Parameters:
//   - dir: Directory path to create
//   - perm: File permissions (e.g., 0755)
//
// Returns an error if directory creation fails.
func CreateIfNotExists(dir string, perm os.FileMode) error {
	// If directory already exists, nothing to do
	if Exists(dir) {
		return nil
	}

	// Create directory and all parent directories
	// os.MkdirAll is idempotent - it won't fail if parts of the path already exist
	if err := os.MkdirAll(dir, perm); err != nil {
		return fmt.Errorf("failed to create directory: '%s', error: '%s'", dir, err.Error())
	}

	return nil
}

// CopySymLink copies a symlink by reading its target and creating a new symlink.
// This preserves the symlink itself, not the file it points to.
//
// Parameters:
//   - source: Path to the source symlink
//   - dest: Path where the new symlink should be created
//
// Returns an error if the operation fails.
func CopySymLink(source, dest string) error {
	// Read the target of the source symlink
	link, err := os.Readlink(source)
	if err != nil {
		return err
	}
	// Create a new symlink with the same target
	return os.Symlink(link, dest)
}

// FindProgramPath locates an executable program in the system PATH.
// This is useful for finding system tools like "codesign", "security", "zip", etc.
//
// Parameters:
//   - program: Name of the program to find (e.g., "codesign", "java")
//
// Returns:
//   - Full path to the executable
//   - An error if the program is not found in PATH
func FindProgramPath(program string) (string, error) {
	// exec.LookPath searches for the executable in directories listed in PATH
	path, err := exec.LookPath(program)
	if err != nil {
		return "", fmt.Errorf("program %q not found in PATH", program)
	}
	return path, nil
}
