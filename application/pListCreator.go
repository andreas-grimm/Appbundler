// Package application: This file generates the Info.plist file required by macOS.
// Info.plist is an XML property list file that contains metadata about the application,
// such as bundle identifier, version, executable name, icon, and minimum OS version.
// macOS reads this file to understand how to launch and display the application.
package application

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"text/template"
)

// plistTemplate is an XML template for the Info.plist file.
// It uses Go's text/template package to fill in values from the configuration.
// The template syntax {{.FieldName}} will be replaced with actual values.
const plistTemplate = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>CFBundleIdentifier</key>
    <string>{{.BundleIdentifier}}</string>
    <key>CFBundleName</key>
    <string>{{.BundleName}}</string>
    <key>CFBundleDisplayName</key>
    <string>{{.BundleDisplayName}}</string>
    <key>CFBundleVersion</key>
    <string>{{.BundleVersion}}</string>
    <key>CFBundleShortVersionString</key>
    <string>{{.ShortVersionString}}</string>
    <key>CFBundleExecutable</key>
    <string>{{.ExecutableName}}</string>
    <key>CFBundleSignature</key>
    <string>{{.Signature}}</string>
    <key>LSMinimumSystemVersion</key>
    <string>{{.MinSystemVersion}}</string>
    <key>CFBundleIconFile</key>
    <string>{{.IconFile}}</string>
    <key>CFBundlePackageType</key>
    <string>{{.PackageType}}</string>
    <key>NSHumanReadableCopyright</key>
    <string>{{.Copyright}}</string>
    {{if .PrincipalClass}}<key>NSPrincipalClass</key>
    <string>{{.PrincipalClass}}</string>{{end}}
    {{if .MainNibFile}}<key>NSMainNibFile</key>
    <string>{{.MainNibFile}}</string>{{end}}
</dict>
</plist>`

// InfoPlistData holds the data that will be inserted into the Info.plist template.
// Each field corresponds to a key in the macOS bundle metadata system:
//   - BundleIdentifier: Unique reverse-DNS identifier (e.g., com.example.myapp)
//   - BundleName: Short name of the application
//   - BundleDisplayName: User-visible name
//   - BundleVersion: Build version number (monotonically increasing)
//   - ShortVersionString: User-visible version (e.g., "1.0.0")
//   - ExecutableName: Name of the file to execute when app launches
//   - Signature: Build signature
//   - MinSystemVersion: Minimum macOS version required (e.g., "10.13.0")
//   - IconFile: Name of the icon file in Resources/ directory
//   - PackageType: Usually "APPL" for applications
//   - Copyright: Copyright notice
//   - PrincipalClass: Principal class (usually NSApplication)
//   - MainNibFile: Main NIB file
type InfoPlistData struct {
	BundleIdentifier   string
	BundleName         string
	BundleDisplayName  string
	BundleVersion      string
	ShortVersionString string
	ExecutableName     string
	Signature          string
	MinSystemVersion   string
	IconFile           string
	PackageType        string
	Copyright          string
	PrincipalClass     string
	MainNibFile        string
}

// CreatePlist generates the Info.plist file in Contents/ directory.
// This file is required by macOS to identify and launch the application.
// The function:
//  1. Reads configuration values from the YAML file
//  2. Validates that all required fields are present
//  3. Uses Go's template engine to generate the XML file
//
// Returns an error if:
//   - Required fields are missing from configuration
//   - File creation fails
//   - Template parsing or execution fails
func CreatePlist() error {
	// Initialize the structure that will hold Info.plist data
	var plistStructure InfoPlistData

	// Populate the structure with values from the configuration file
	// These getter functions read from the packageInfo variable set by Read()
	plistStructure.BundleIdentifier = GetBundleIdentifier()
	plistStructure.BundleVersion = GetBundleVersion()
	plistStructure.BundleName = GetBundleName()
	plistStructure.BundleDisplayName = GetBundleDisplayName()
	plistStructure.ShortVersionString = GetCFBundleShortVersionString()
	plistStructure.ExecutableName = GetBundleExecutable()
	plistStructure.Signature = GetBundleSignature()
	plistStructure.MinSystemVersion = GetMinimumMacOSVersion()
	plistStructure.IconFile = GetIconFileName()
	plistStructure.PackageType = GetPackageType()
	plistStructure.Copyright = GetNSHumanReadableCopyright()
	plistStructure.PrincipalClass = GetNSPrincipalClass()
	plistStructure.MainNibFile = GetNSMainNibFile()

	// Info.plist must be in Contents/ directory (required by macOS)
	plistFileName := filepath.Join(contentsDir, "Info.plist")

	// Create PkgInfo file as well (required by some older macOS versions)
	if err := CreatePkgInfo(); err != nil {
		return err
	}

	// Validate that all mandatory fields are present
	// macOS requires these fields to be non-empty for the bundle to work correctly
	// Note: fmt.Sprintf here doesn't do anything useful - could be removed or used for logging
	fmt.Sprintf("pList:", plistStructure.BundleIdentifier, plistStructure.BundleVersion, plistStructure.BundleName, plistStructure.ExecutableName, plistStructure.MinSystemVersion, plistStructure.IconFile)
	if plistStructure.BundleIdentifier == "" || plistStructure.BundleVersion == "" || plistStructure.BundleName == "" ||
		plistStructure.ExecutableName == "" || plistStructure.MinSystemVersion == "" || plistStructure.IconFile == "" {
		return errors.New("Info.plist <mandatory fields missing>")
	}

	// Create the Info.plist file
	file, err := os.Create(plistFileName)
	if err != nil {
		return cleanAfterError(err)
	}
	defer file.Close() // Ensure file is closed when function exits

	// Parse the XML template
	// The template contains placeholders like {{.BundleIdentifier}} that will be replaced
	tmpl, err := template.New("plist").Parse(plistTemplate)
	if err != nil {
		return cleanAfterError(err)
	}

	// Execute the template: replace placeholders with actual values and write to file
	// This generates the final XML content
	err = tmpl.Execute(file, plistStructure)
	if err != nil {
		return cleanAfterError(err)
	}

	return nil
}

// CreatePkgInfo generates the PkgInfo file in Contents/ directory.
// This file contains the package type (APPL) and creator signature (????).
// It's a legacy requirement but still good practice for macOS bundles.
func CreatePkgInfo() error {
	pkgInfoFileName := filepath.Join(contentsDir, "PkgInfo")

	file, err := os.Create(pkgInfoFileName)
	if err != nil {
		return cleanAfterError(err)
	}
	defer file.Close()

	// PkgInfo content: 4 bytes for type (APPL) + 4 bytes for signature (default ????)
	// The signature can be customized, but ???? is the standard default for generic apps
	packageType := GetPackageType()
	if len(packageType) != 4 {
		packageType = "APPL"
	}

	signature := GetBundleSignature()
	if len(signature) != 4 {
		signature = "????"
	}

	_, err = file.WriteString(packageType + signature)
	return err
}

// cleanAfterError handles cleanup of the directory structure when an error occurs.
// This prevents leaving partial or broken bundles on disk.
func cleanAfterError(err error) error {
	if applicationDirectory != "" {
		DeleteAll()
	}
	return err
}
