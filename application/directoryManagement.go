// Package application: This file manages the directory structure of macOS application bundles.
// macOS requires a specific directory structure for .app bundles:
//   MyApp.app/
//     Contents/
//       Info.plist          (required metadata file)
//       MacOS/              (executable files go here)
//       Resources/          (icons, images, etc.)
//       Java/               (optional, for bundled Java runtime)
//         runtime/          (Java installation if local_java is enabled)
package application

import (
	"appbundler/utilities/logger"
	"errors"
	"os"
	"path/filepath"
)

// Package-level variables storing paths to key directories in the bundle.
// These are set by CreateDirectoryStructure() and used by other functions.
var (
	applicationDirectory string // Root of the bundle: MyApp.app
	contentsDir         string // Contents/ directory (required by macOS)
	macosDir            string // Contents/MacOS/ (executables go here)
	resourcesDir        string // Contents/Resources/ (icons, assets)
	javaDir             string // Contents/Java/ (for bundled Java runtime)
	runtimeDir          string // Contents/Java/runtime/ (actual Java installation)
)

// CreateDirectoryStructure creates the complete directory hierarchy for a macOS application bundle.
// This function builds the required structure that macOS expects for .app bundles.
//
// Directory structure created:
//   applicationRoot.app/
//     Contents/
//       MacOS/          (executables)
//       Resources/      (icons, assets)
//       Java/           (optional, for Java apps)
//         runtime/      (bundled Java installation)
//
// Parameters:
//   - applicationRoot: Base name of the application (without .app extension)
//
// Returns an error if:
//   - applicationRoot is empty
//   - Any directory creation fails
//
// Note: If any directory creation fails, the function attempts to clean up
// by deleting the partially created bundle.
func CreateDirectoryStructure(applicationRoot string) error {
	logger.Info("Creating and setting up the bundle directories")
	
	// Validate that application root name is provided
	// All macOS application bundles must have a .app extension
	if applicationRoot != "" {
		// Build the complete directory paths
		applicationDirectory = applicationRoot + ".app"                    // MyApp.app
		contentsDir = filepath.Join(applicationDirectory, "Contents")    // MyApp.app/Contents
		macosDir = filepath.Join(contentsDir, "MacOS")                   // MyApp.app/Contents/MacOS
		resourcesDir = filepath.Join(contentsDir, "Resources")            // MyApp.app/Contents/Resources
		javaDir = filepath.Join(contentsDir, "Java")                     // MyApp.app/Contents/Java
		runtimeDir = filepath.Join(javaDir, "runtime")                    // MyApp.app/Contents/Java/runtime
	} else {
		applicationError := errors.New("Application root directory cannot be empty")
		return applicationError
	}

	// Create each directory in the hierarchy
	// If any creation fails, clean up and return the error
	// This ensures we don't leave partial bundles on disk

	creationError := createDir(applicationDirectory)
	if creationError != nil {
		DeleteAll()
		return creationError
	}

	creationError = createDir(contentsDir)
	if creationError != nil {
		DeleteAll()
		return creationError
	}

	creationError = createDir(macosDir)
	if creationError != nil {
		DeleteAll()
		return creationError
	}

	creationError = createDir(resourcesDir)
	if creationError != nil {
		DeleteAll()
		return creationError
	}

	creationError = createDir(javaDir)
	if creationError != nil {
		DeleteAll()
		return creationError
	}

	creationError = createDir(runtimeDir)
	if creationError != nil {
		DeleteAll()
		return creationError
	}
	return nil
}

// createDir creates a directory and all necessary parent directories.
// Uses os.MkdirAll which is idempotent - it won't fail if the directory already exists.
//
// Parameters:
//   - path: Full path of the directory to create
//
// Returns an error if directory creation fails.
func createDir(path string) error {
	// 0755 = rwxr-xr-x permissions:
	// - Owner: read, write, execute
	// - Group: read, execute
	// - Others: read, execute
	err := os.MkdirAll(path, 0755)
	if err != nil {
		logger.Debug("Error creating directory:", path, err)
	}

	return err
}

// DeleteAll removes the entire application bundle directory structure.
// This is used for cleanup operations (--clean flag) or when errors occur during creation.
//
// Returns an error if the deletion fails.
func DeleteAll() error {
	logger.Info("Delete all bundle directories")
	
	// os.RemoveAll recursively deletes the directory and all its contents
	err := os.RemoveAll(applicationDirectory)
	if err != nil {
		logger.Debug("Error deleting directory:", applicationDirectory, err)
	}

	return err
}
