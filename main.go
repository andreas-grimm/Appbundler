// Package main is the entry point for the appbundler application.
// This tool creates macOS application bundles (.app) from executables or Java JAR files.
// It handles directory structure creation, Info.plist generation, icon copying, and code signing.
package main

import (
	"appbundler/application"
	"appbundler/utilities/logger"
	"flag"
	"fmt"
	"os"
)

// Command-line flags define the behavior of the application bundler.
// These flags allow users to customize the bundling process without modifying code.
var (
	// applicationNameFlag: Override the application name from the YAML config file.
	// If not provided, the name from application.yaml will be used.
	applicationNameFlag = flag.String("app", "my_app", "Name of the application bundle (default 'name' value in the application file)")

	// packageFileFlag: Path to the YAML configuration file containing bundle metadata.
	// This file defines bundle identifier, version, executable name, icon, etc.
	packageFileFlag = flag.String("application", "application.yaml", "Package description file")

	// cleanFlag: If true, removes any existing .app bundle before creating a new one.
	// Useful when rebuilding to ensure a clean state.
	cleanFlag = flag.Bool("clean", false, "Clean existing structure before rebuilding")

	// signFlag: If true, code signs the application bundle using a development certificate.
	// Required for distribution and Gatekeeper compatibility on macOS.
	signFlag = flag.Bool("sign", false, "Sign the application structure with a real development key")

	// deleteFlag: If true, removes the created bundle after building (useful for testing).
	deleteFlag = flag.Bool("delete", false, "Delete the application structure")

	// notariseFlag: If true, submits the app to Apple for notarization.
	// Notarization is required for distribution outside the Mac App Store.
	notariseFlag = flag.Bool("notarize", false, "Notarize for distribution")

	// appleIDProfileFlag: The name of the keychain profile containing Apple ID credentials.
	// Required if -notarize is used.
	appleIDProfileFlag = flag.String("profile", "", "Apple ID profile name for notarization")

	// silentFlag: If true, suppresses informational log messages (only errors will be shown).
	silentFlag = flag.Bool("silent", false, "Silent mode during installation")

	// logDirFlag: Directory where log files should be written. If set, enables file logging.
	// Log files are named with the application name and timestamp: <appName>_YYYY-MM-DD_HH-MM-SS.log
	logDirFlag = flag.String("logdir", "", "Directory for log files (enables file logging)")
)

// main is the entry point of the application bundler.
// It orchestrates the entire bundling process in the following order:
// 1. Parse command-line flags
// 2. Read configuration from YAML file
// 3. Create directory structure (.app/Contents/...)
// 4. Generate Info.plist file
// 5. Copy executable (or JAR + create launcher script)
// 6. Copy icon file
// 7. Optionally sign the application
// 8. Optionally clean up temporary files
func main() {
	// Parse all command-line flags defined above
	flag.Parse()

	// Get the application name from command-line flag
	applicationName := *applicationNameFlag

	// Configure logger to suppress output if silent mode is enabled
	if silentFlag != nil && *silentFlag {
		logger.SetSilent(*silentFlag)
	}

	// Read the YAML configuration file that contains bundle metadata
	// This populates internal structures with bundle identifier, version, executable name, etc.
	packageFileError := application.Read(*packageFileFlag)
	if packageFileError != nil {
		logger.Debug(packageFileError.Error())
		os.Exit(1)
	}

	// Step 0: Validate the configuration and check if all source files exist
	// This prevents partial builds by ensuring everything is ready before we start
	if err := application.ValidateConfiguration(); err != nil {
		logger.Error(err)
		os.Exit(1)
	}

	// If no application name was provided via command-line, use the name from the config file
	if applicationNameFlag == nil || applicationName == "" {
		applicationName = application.GetBundleName()
		logger.Debug("Application name is %s", applicationName)
	}

	// Set up file logging if log directory is specified
	// This enables dual logging: messages go to both stdout and the log file
	if logDirFlag != nil && *logDirFlag != "" {
		// Use the determined application name for the log file
		logFileName := applicationName
		if logFileName == "" {
			logFileName = "appbundler" // Fallback if no name available
		}

		err := logger.SetLogFile(logFileName, *logDirFlag)
		if err != nil {
			// Log the error but don't exit - file logging is optional
			logger.Debug("Failed to set up file logging: %v", err)
		} else {
			logger.Info("Logging to file: %s", logger.GetLogFilePath())
		}
	}

	// If clean flag is set, remove any existing bundle to start fresh
	if *cleanFlag == true {
		logger.Debug("Delete previous generated files")
		application.DeleteAll()
	}

	logger.Debug("Name of the application bundle description file: %s", *packageFileFlag)

	// Step 1: Create the macOS bundle directory structure
	// This creates: MyApp.app/Contents/{MacOS, Resources, Java/runtime}
	packageFileError = application.CreateDirectoryStructure(application.GetBundleName())
	if packageFileError != nil {
		errorExit(packageFileError)
		return
	}

	// Step 2: Generate the Info.plist file
	// Info.plist is required by macOS to identify and launch the application
	// It contains metadata like bundle identifier, version, executable name, icon, etc.
	packageFileError = application.CreatePlist()
	if packageFileError != nil {
		errorExit(packageFileError)
	}

	// Step 3: Copy the executable file into the bundle
	// For JAR files: copies JAR, optionally bundles Java runtime, and creates a launcher script
	// For compiled executables: copies the binary and makes it executable
	packageFileError = application.CopyExecutable()
	if packageFileError != nil {
		errorExit(packageFileError)
	}

	// Step 4: Copy the application icon to Resources directory
	// The icon file (usually .icns format) is required for proper macOS integration
	packageFileError = application.CopyIcon()
	if packageFileError != nil {
		errorExit(packageFileError)
	}

	// Step 5: Code sign the application bundle (optional)
	// Code signing is required for:
	// - Distribution outside the Mac App Store
	// - Passing Gatekeeper checks
	// - Notarization (if distributing)
	// Uses the first available development certificate from the keychain
	if signFlag != nil && *signFlag == true {
		packageFileError = application.SignApplication()
		if packageFileError != nil {
			errorExit(packageFileError)
		}
	}

	// Step 6: Notarize the application bundle (optional)
	// Notarization requires the bundle to be signed first.
	// It also requires an Apple ID profile for credentials.
	if notariseFlag != nil && *notariseFlag == true {
		if appleIDProfileFlag == nil || *appleIDProfileFlag == "" {
			errorExit(fmt.Errorf("notarization requires an Apple ID profile (use -profile <name>)"))
		}

		logger.Info("Starting notarization process (this may take several minutes)...")
		packageFileError = application.NotarizeApplication(application.GetBundleName(), *appleIDProfileFlag)
		if packageFileError != nil {
			errorExit(packageFileError)
		}
		logger.Info("Notarization completed successfully")
	}

	// Step 7: Clean up (optional, mainly for testing)
	// If delete flag is set, remove the bundle after creation
	if deleteFlag != nil && *deleteFlag {
		packageFileError = application.DeleteAll()
		if packageFileError != nil {
			errorExit(packageFileError)
		}
	}

	logger.Info("Application Bundler completed successfully")
}

// errorExit is a helper function that handles errors by logging them and exiting the program.
// This ensures that any error during the bundling process stops execution immediately
// and provides clear feedback to the user about what went wrong.
func errorExit(err error) {
	if err != nil {
		logger.Error(err)
		os.Exit(1)
	}
}
