// Package application contains the core logic for creating macOS application bundles.
// This file handles copying executable files (both compiled binaries and Java JAR files)
// into the appropriate location within the .app bundle structure.
package application

import (
	"appbundler/utilities/fileManagement"
	"appbundler/utilities/logger"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// CopyExecutable copies the executable file into the macOS bundle.
// It determines whether the executable is a JAR file or a compiled binary and
// handles each case appropriately:
//   - JAR files: Copies JAR, optionally bundles Java runtime, and creates a launcher script
//   - Compiled binaries: Copies the binary and sets executable permissions
//
// Returns an error if the copy operation fails.
func CopyExecutable() error {
	logger.Info("Copying the Executable")

	var err error

	// Get the executable filename and directory from the configuration
	execFile := GetExecutableName()
	execPath := GetExecutableDirectory()

	// If local_exec_directory is provided, use it instead of the default exec_file_directory
	if GetLocalExecDirectory() != "" {
		execPath = GetLocalExecDirectory()
	}

	// Determine if this is a Java JAR file or a compiled executable
	// JAR files need special handling: they require a launcher script and optionally a Java runtime
	if strings.HasSuffix(execFile, "jar") {
		err = copyJarExec(execPath, execFile)
	} else {
		// For compiled executables (Go binaries, C/C++ binaries, etc.), just copy and set permissions
		err = copyCompExec(execPath, execFile)
	}

	if err != nil {
		logger.Debug("failed to copy executable file:", execFile, err.Error())
	}

	return nil
}

// copyJarExec handles copying Java JAR files and creating a launcher script.
// This function performs three main tasks:
//  1. Optionally copies the Java runtime into the bundle (if local_java is enabled)
//  2. Copies the JAR file into the MacOS directory
//  3. Creates a bash script that launches the JAR file
//
// Parameters:
//   - execPath: Directory containing the JAR file
//   - execFile: Name of the JAR file
//
// Returns an error if any step fails.
func copyJarExec(execPath string, execFile string) error {
	var err error

	// The launcher script will be created in Contents/MacOS/ with the bundle executable name
	// This is the file that macOS will execute when the user double-clicks the app
	executableName := filepath.Join(macosDir, GetBundleExecutable())

	// Step 1: Optionally bundle a local Java runtime
	// If local_java is set to true in the config, copy the entire Java installation
	// into Contents/Java/runtime. This makes the app self-contained and doesn't
	// require users to have Java installed on their system.
	if GetUseLocalJava() == true {
		javaSourceName := GetJavaHomeDirectory()
		javaDestName := filepath.Join(javaDir, "runtime")

		// Copy the entire Java installation directory (this can be large, ~200MB+)
		err = fileManagement.CopyDirectory(javaSourceName, javaDestName)
		if err != nil {
			logger.Debug("failed to copy java installation:", javaSourceName, err.Error())
			return err
		}
	}

	// Step 2: Copy the JAR file into Contents/MacOS/
	// The JAR file will be executed by the launcher script
	compiledJarSourceName := filepath.Join(execPath, execFile)
	compiledJarTargetName := filepath.Join(macosDir, execFile)

	err = fileManagement.Copy(compiledJarSourceName, compiledJarTargetName)
	if err != nil {
		logger.Debug("failed to copy java executable:", compiledJarSourceName, err.Error())
		return err
	}

	// Step 3: Create a shell script launcher
	// macOS will execute this script when the app is launched
	// The script runs the JAR file using either the bundled Java or system Java
	file, err := os.Create(executableName)
	if err != nil || file == nil {
		logger.Debug("failed to generate start script:", executableName)
		return err
	}

	// Generate the shell script content
	// If using local Java, the script sets JAVA_HOME to the bundled runtime
	var startString string

	if GetUseLocalJava() == true {
		// Script for bundled Java runtime
		startString = fmt.Sprintf("#!/bin/bash\n\nDIR=\"$(cd \"$(dirname \"$0\")\" && pwd)\"\nexport JAVA_HOME=\"$DIR/../Java/runtime\"\n\"$JAVA_HOME/bin/java\" -jar \"$DIR/%s\"\n", execFile)
	} else {
		// Script for system Java
		startString = fmt.Sprintf("#!/bin/bash\n\nDIR=\"$(cd \"$(dirname \"$0\")\" && pwd)\"\njava -jar \"$DIR/%s\"\n", execFile)
	}

	_, err = file.WriteString(startString)
	if err != nil {
		return err
	}

	err = file.Close()

	// Make the script executable (required for macOS to run it)
	// 0755 = rwxr-xr-x: owner can read/write/execute, others can read/execute
	err = os.Chmod(executableName, 0755)
	if err != nil {
		logger.Debug("failed to make script executable")
		return err
	}

	return nil
}

// copyCompExec handles copying compiled executable binaries (Go, C/C++, etc.).
// Unlike JAR files, compiled executables don't need a launcher script - they can
// be executed directly by macOS.
//
// Parameters:
//   - execPath: Directory containing the executable file
//   - execFile: Name of the executable file
//
// Returns an error if the copy operation fails.
func copyCompExec(execPath string, execFile string) error {
	// Destination path: Contents/MacOS/executable_name
	executablePath := filepath.Join(macosDir, execFile)
	sourceFileName := filepath.Join(execPath, execFile)

	// Copy the executable binary from source to the bundle
	err := fileManagement.Copy(sourceFileName, executablePath)
	if err != nil {
		logger.Debug("failed to copy executable file from source to destination file:", sourceFileName, err.Error())
		return err
	}

	// Set executable permissions (required for macOS to run the binary)
	// 0755 = rwxr-xr-x: owner can read/write/execute, others can read/execute
	err = os.Chmod(executablePath, 0755)
	if err != nil {
		return err
	}

	return nil
}
