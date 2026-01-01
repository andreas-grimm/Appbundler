// Package application: This file handles reading and parsing the YAML configuration file.
// The configuration file (typically application.yaml) contains all the metadata needed
// to create the macOS application bundle, such as bundle identifier, version, executable
// name, icon, and Java runtime settings.
package application

import (
	"appbundler/utilities/logger"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// packageInfo is a package-level variable that stores the parsed configuration.
// It's populated by the Read() function and accessed by getter functions.
var packageInfo packageParameter

// packageParameter defines the structure of the YAML configuration file.
// The `yaml:"tag"` annotations map YAML keys to struct fields.
// This struct holds all the information needed to create a macOS application bundle.
type packageParameter struct {
	// Bundle metadata (required for Info.plist)
	BundleIdentifier  string `yaml:"id"`           // Unique reverse-DNS identifier (e.g., com.example.myapp)
	BundleName        string `yaml:"name"`         // Short name of the bundle (e.g., MyApp)
	BundleVersion     string `yaml:"version"`      // Build version number (e.g., "1" or "1.0.0")
	BundleDisplayName string `yaml:"display_name"` // User-visible name (can be localized)
	BundlePackageType string `yaml:"type"`         // Package type, default is "APPL"
	BundleExecutable  string `yaml:"executable"`   // Name of the main executable file (CFBundleExecutable)
	BundleSignature   string `yaml:"signature"`    // Build signature (monotonically increasing version string)

	// Executable file location
	ExecFileName      string `yaml:"exec_file"`           // Name of the executable/JAR file to package
	ExecFileDirectory string `yaml:"exec_file_directory"` // Directory containing the executable/JAR

	// Icon file location
	IconFileName      string `yaml:"icon_file"`           // Name of the icon file (typically .icns)
	IconFileDirectory string `yaml:"icon_file_directory"` // Directory containing the icon file

	// Additional macOS bundle properties (optional)
	MinimumMacOSVersion        string `yaml:"system_minimal_os_version"` // Minimum macOS version (e.g., "10.13.0")
	CFBundleDocumentTypes      string `yaml:"document_types"`            // Document types this app can open
	CFBundleShortVersionString string `yaml:"short_version_string"`      // User-visible version (e.g., "1.0.0")
	NSHumanReadableCopyright   string `yaml:"readable_copyright"`        // Copyright notice
	NSMainNibFile              string `yaml:"main_nib_file"`             // Main NIB file (for Cocoa apps)
	NSPrincipalClass           string `yaml:"principle_class"`           // Principal class (usually NSApplication)

	// Java-specific settings (for JAR-based applications)
	LocalJava          string `yaml:"local_java"`           // "true" to bundle Java runtime, "false" to use system Java
	LocalJavaHome      string `yaml:"local_java_home"`      // Path to Java installation to bundle (if local_java is true)
	LocalExecDirectory string `yaml:"local_exec_directory"` // Alternative executable directory
}

// Read parses the YAML configuration file and populates the packageInfo variable.
// This function must be called before any other application functions that need
// configuration data (like GetBundleName(), GetExecutableName(), etc.).
//
// Parameters:
//   - packageFileName: Path to the YAML configuration file (defaults to "application.yaml")
//
// Returns an error if:
//   - File cannot be opened
//   - File cannot be read
//   - YAML parsing fails
func Read(packageFileName string) error {
	var err error

	// Default to "application.yaml" if no filename is provided
	if packageFileName == "" {
		packageFileName = "application.yaml"
	}

	// Open the YAML configuration file
	file, err := os.Open(packageFileName)
	if err != nil {
		logger.Error(err)
		return err
	}

	// Read the entire file contents into memory
	// For large files, streaming might be better, but YAML files are typically small
	data, err := io.ReadAll(file)
	if err != nil {
		logger.Error(err)
		return err
	}

	// Parse the YAML data into the packageInfo struct
	// yaml.Unmarshal uses the struct field tags (yaml:"key") to map YAML keys to fields
	if err := yaml.Unmarshal(data, &packageInfo); err != nil {
		logger.Error(err)
		return err
	}

	// Close the file (though defer would be better practice)
	err = file.Close()
	return err
}

// The following functions are getters that provide access to configuration values.
// They read from the packageInfo variable that was populated by Read().
// These functions provide a clean API and allow for future validation or transformation logic.

// ValidateConfiguration ensures that all required files and directories exist
// before the bundling process begins. This prevents partial builds.
func ValidateConfiguration() error {
	// 1. Check executable
	execFile := GetExecutableName()
	execDir := GetExecutableDirectory()
	if GetLocalExecDirectory() != "" {
		execDir = GetLocalExecDirectory()
	}

	fullExecPath := filepath.Join(execDir, execFile)
	if _, err := os.Stat(fullExecPath); os.IsNotExist(err) {
		return fmt.Errorf("executable file not found: %s", fullExecPath)
	}

	// 2. Check icon file
	iconFile := GetIconFileName()
	if iconFile != "" {
		iconDir := GetIconFileDirectory()
		fullIconPath := filepath.Join(iconDir, iconFile)
		if _, err := os.Stat(fullIconPath); os.IsNotExist(err) {
			return fmt.Errorf("icon file not found: %s", fullIconPath)
		}
	}

	// 3. Check Java Home if local Java is enabled
	if GetUseLocalJava() {
		javaHome := GetJavaHomeDirectory()
		if _, err := os.Stat(javaHome); os.IsNotExist(err) {
			return fmt.Errorf("local Java home directory not found: %s", javaHome)
		}
	}

	return nil
}

// GetBundleIdentifier returns the unique bundle identifier (e.g., "com.example.myapp").
func GetBundleIdentifier() string {
	return packageInfo.BundleIdentifier
}

// GetBundleName returns the short name of the bundle.
func GetBundleName() string {
	return packageInfo.BundleName
}

// GetBundleVersion returns the build version number.
func GetBundleVersion() string {
	return packageInfo.BundleVersion
}

// GetBundleExecutable returns the name of the executable file (CFBundleExecutable in Info.plist).
func GetBundleExecutable() string {
	return packageInfo.BundleExecutable
}

// GetMinimumMacOSVersion returns the minimum macOS version required (e.g., "10.13.0").
func GetMinimumMacOSVersion() string {
	return packageInfo.MinimumMacOSVersion
}

// GetIconFileName returns the name of the icon file (without directory path).
func GetIconFileName() string {
	return packageInfo.IconFileName
}

// GetIconFileDirectory returns the directory containing the icon file.
func GetIconFileDirectory() string {
	return packageInfo.IconFileDirectory
}

// GetPackageType returns the bundle package type, defaulting to "APP" if not specified.
// Typically this is "APPL" for applications.
func GetPackageType() string {
	if packageInfo.BundlePackageType != "" {
		return packageInfo.BundlePackageType
	}
	return "APP"
}

// GetExecutableName returns the name of the executable/JAR file to be packaged.
func GetExecutableName() string {
	return packageInfo.ExecFileName
}

// GetExecutableDirectory returns the directory containing the executable/JAR file.
func GetExecutableDirectory() string {
	return packageInfo.ExecFileDirectory
}

// GetUseLocalJava returns true if the configuration specifies bundling a local Java runtime.
// This checks if the "local_java" YAML field is set to "true" (case-insensitive).
func GetUseLocalJava() bool {
	if strings.ToLower(packageInfo.LocalJava) == "true" {
		return true
	}
	return false
}

// GetJavaHomeDirectory returns the path to the Java installation to bundle (if local_java is enabled).
func GetJavaHomeDirectory() string {
	return packageInfo.LocalJavaHome
}

// GetBundleDisplayName returns the user-visible name of the bundle.
func GetBundleDisplayName() string {
	return packageInfo.BundleDisplayName
}

// GetBundleSignature returns the build signature.
func GetBundleSignature() string {
	return packageInfo.BundleSignature
}

// GetCFBundleDocumentTypes returns the document types this app can open.
func GetCFBundleDocumentTypes() string {
	return packageInfo.CFBundleDocumentTypes
}

// GetCFBundleShortVersionString returns the user-visible version string.
func GetCFBundleShortVersionString() string {
	return packageInfo.CFBundleShortVersionString
}

// GetNSHumanReadableCopyright returns the copyright notice.
func GetNSHumanReadableCopyright() string {
	return packageInfo.NSHumanReadableCopyright
}

// GetNSMainNibFile returns the main NIB file name.
func GetNSMainNibFile() string {
	return packageInfo.NSMainNibFile
}

// GetNSPrincipalClass returns the principal class.
func GetNSPrincipalClass() string {
	return packageInfo.NSPrincipalClass
}

// GetLocalExecDirectory returns the alternative executable directory.
func GetLocalExecDirectory() string {
	return packageInfo.LocalExecDirectory
}
