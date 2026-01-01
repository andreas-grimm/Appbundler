// Package application: This file handles copying the application icon into the bundle.
// The icon file (typically .icns format) is placed in Contents/Resources/ and
// referenced in the Info.plist file. macOS uses this icon to display the app
// in Finder, Dock, and other system locations.
package application

import (
	"appbundler/utilities/logger"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// CopyIcon copies the application icon file from the source location to
// Contents/Resources/ within the bundle. The icon file name is specified in
// the configuration YAML file.
//
// Returns an error if:
//   - Icon filename is not defined in config
//   - Source file doesn't exist
//   - Copy operation fails
func CopyIcon() error {
	logger.Info("Copying the Icon File")

	// Get icon filename and directory from configuration
	iconSource := GetIconFileName()
	iconDirectory := GetIconFileDirectory()
	
	// Destination path: Contents/Resources/icon_filename.icns
	iconPath := filepath.Join(resourcesDir, iconSource)

	// Validate that icon filename is defined in configuration
	// An empty icon filename means no icon was specified
	if iconSource == "" {
		var err error

		err = errors.New(fmt.Sprintf("icon filename %s is not defined", iconSource))
		logger.Debug("failed to open source file:", iconSource, err.Error())
		return err
	}

	// If a directory was specified for the icon, construct the full source path
	// Example: icon_file_directory: ./test/icon, icon_file: appIcon.icns
	// Results in: ./test/icon/appIcon.icns
	if iconDirectory != "" {
		iconSource = filepath.Join(iconDirectory, iconSource)
	}

	// Open the source icon file for reading
	sourceFile, err := os.Open(iconSource)
	if err != nil {
		logger.Debug("failed to open source file:", iconSource, err.Error())
		return err
	}
	defer sourceFile.Close() // Ensure file is closed when function exits

	// Create the destination icon file in Contents/Resources/
	destinationFile, err := os.Create(iconPath)
	if err != nil {
		logger.Debug("failed to open the destination file:", iconPath, err.Error())
		return err
	}
	defer destinationFile.Close() // Ensure file is closed when function exits

	// Copy the icon file contents from source to destination
	// io.Copy efficiently handles the transfer, even for large files
	_, err = io.Copy(destinationFile, sourceFile)
	if err != nil {
		logger.Debug("failed to copy icon from source to destination file:", iconPath, err.Error())
		return err
	}

	// Preserve the original file permissions from the source icon
	// This ensures the icon file has appropriate read permissions
	sourceFileInfo, err := sourceFile.Stat()
	if err != nil {
		logger.Debug("failed to stat source file:", iconSource, err.Error())
		return err
	}

	// Apply the same permissions to the destination file
	err = os.Chmod(iconPath, sourceFileInfo.Mode())
	if err != nil {
		logger.Debug("failed to set permissions on destination file:", iconPath, err.Error())
		return err
	}

	return nil
}
