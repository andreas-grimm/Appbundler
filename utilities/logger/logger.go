// Package logger provides a simple logging system for the application bundler.
// It supports different log levels (Debug, Info, Warn, Error, Fatal) and can
// output to stdout, a file, or both simultaneously. Silent mode can suppress non-error messages.
package logger

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"
)

// Package-level variables for logger configuration
var (
	logFile     string                    // Path to log file (if logging to file)
	logDest     = log.New(os.Stdout, "", log.Ldate|log.Ltime) // Default: log to stdout
	logFileDest *log.Logger                // Logger for file output (nil if not set)
	logLevel    string                    // Current log level (not currently used)
	silence     bool = false              // If true, suppress non-error messages
)

// SetSilent enables or disables silent mode.
// When silent mode is enabled, only error messages are displayed.
// This is useful for automated scripts or when verbose output is not needed.
//
// Parameters:
//   - isSilent: true to enable silent mode, false to show all messages
func SetSilent(isSilent bool) {
	silence = isSilent
}

// logFormat formats a log message with optional values and then prints it.
// This is an internal helper function used by the public logging functions.
//
// Parameters:
//   - logType: Type of log message (Debug, Info, Warn, Error, Fatal)
//   - format: Format string (like fmt.Sprintf)
//   - values: Optional values to format into the message
func logFormat(logType string, format string, values ...any) {
	var logMessage string

	// If no values provided, use format string as-is
	// Otherwise, format it with the provided values
	if values == nil {
		logMessage = format
	} else {
		logMessage = fmt.Sprintf(format, values...)
	}

	logPrint(logType, logMessage)
}

// logPrint is the core logging function that actually writes the message.
// It handles special cases like Error (which exits the program) and silent mode.
// If a log file is configured, messages are written to both stdout and the file.
//
// Parameters:
//   - logType: Type of log message
//   - message: The message to log
func logPrint(logType string, message string) {
	// Format the log message with type prefix
	logMessage := "[" + logType + "] " + message

	// Error messages always print and exit the program
	if logType == "Error" {
		// Write to file if configured before exiting
		if logFileDest != nil {
			logFileDest.Println(logMessage)
		}
		log.Fatalln(logMessage)
		return
	}

	// In silent mode, suppress non-error messages to stdout
	// But still write to file if configured
	if !silence {
		logDest.Println(logMessage)
	}

	// Always write to file if configured (even in silent mode for non-errors)
	if logFileDest != nil {
		logFileDest.Println(logMessage)
	}
}

// The following functions provide different log levels for different purposes.
// Each function can take either a simple string or a format string with values.

// Debug logs a debug message (detailed information for developers).
// These messages are typically only useful during development and debugging.
func Debug(format string, values ...any) {
	if values != nil {
		logFormat("Debug", format, values)
	} else {
		logPrint("Debug", format)
	}
}

// Info logs an informational message (general information about program execution).
// These messages inform users about what the program is doing.
func Info(format string, values ...any) {
	if values != nil {
		logFormat("Info", format, values)
	} else {
		logPrint("Info", format)
	}
}

// Warn logs a warning message (something unexpected but not fatal).
// The program continues execution after a warning.
func Warn(format string, values ...any) {
	if values != nil {
		logFormat("Warn", format, values)
	} else {
		logPrint("Warn", format)
	}
}

// Error logs an error message and exits the program.
// This function is for critical errors that prevent the program from continuing.
// It both logs the error and calls panic to stop execution.
//
// Parameters:
//   - err: The error to log
func Error(err error) {
	logPrint("Error", err.Error())
	panic("Error: " + err.Error())
}

// Fatal logs a fatal error message (similar to Error but takes a format string).
// This is for critical errors that should stop program execution.
func Fatal(format string, values ...any) {
	if values != nil {
		logFormat("Fatal", format, values)
	} else {
		logPrint("Fatal", format)
	}
}

// SetLogFile sets up logging to a file in addition to stdout.
// The log file will be created with a name based on the application name and current date/time.
// Format: <appName>_YYYY-MM-DD_HH-MM-SS.log
// If logDir is empty, the file will be created in the current directory.
//
// Parameters:
//   - appName: Name of the application (used in filename)
//   - logDir: Directory where the log file should be created (empty string = current directory)
//
// Returns an error if the log file cannot be created or opened.
//
// Note: This enables dual logging - messages will be written to both stdout and the file.
// The file is opened in append mode, so new logs are added to the end if the file already exists.
func SetLogFile(appName string, logDir string) error {
	// Generate filename with application name and current date/time
	// Format: MyApp_2025-01-15_14-30-45.log
	now := time.Now()
	timeStr := now.Format("2006-01-02_15-04-05") // Go's reference time format
	fileName := fmt.Sprintf("%s_%s.log", appName, timeStr)

	// Construct full path
	var filePath string
	if logDir != "" {
		// Ensure directory exists
		if err := os.MkdirAll(logDir, 0755); err != nil {
			return fmt.Errorf("failed to create log directory: %v", err)
		}
		filePath = filepath.Join(logDir, fileName)
	} else {
		filePath = fileName
	}

	// Open file in read-write mode, create if it doesn't exist, append to existing content
	// 0666 = rw-rw-rw- permissions (readable/writable by all)
	file, err := os.OpenFile(filePath, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		return fmt.Errorf("failed to open log file: %v", err)
	}

	// Store the file path and create a logger for file output
	logFile = filePath
	logFileDest = log.New(file, "", log.Ldate|log.Ltime)

	return nil
}

// SetLogFileWithPath sets up logging to a specific file path.
// This is an alternative to SetLogFile that allows full control over the file path.
// If a log file is already set, it will be replaced.
//
// Parameters:
//   - filePath: Full path to the log file (will be created if it doesn't exist)
//
// Returns an error if the log file cannot be created or opened.
//
// Note: This enables dual logging - messages will be written to both stdout and the file.
func SetLogFileWithPath(filePath string) error {
	// Ensure the directory exists
	dir := filepath.Dir(filePath)
	if dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create log directory: %v", err)
		}
	}

	// Open file in read-write mode, create if it doesn't exist, append to existing content
	// 0666 = rw-rw-rw- permissions (readable/writable by all)
	file, err := os.OpenFile(filePath, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		return fmt.Errorf("failed to open log file: %v", err)
	}

	// Store the file path and create a logger for file output
	logFile = filePath
	logFileDest = log.New(file, "", log.Ldate|log.Ltime)

	return nil
}

// GetLogFilePath returns the current log file path, or empty string if no log file is set.
func GetLogFilePath() string {
	return logFile
}
