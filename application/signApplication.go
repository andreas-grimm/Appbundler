// Package application: This file handles code signing of macOS application bundles.
// Code signing is required for:
//   - Distribution outside the Mac App Store
//   - Passing Gatekeeper security checks
//   - Notarization (required for distribution)
//
// The signing process uses Apple's codesign tool and automatically finds
// an available development certificate in the keychain.
package application

import (
	"appbundler/utilities/fileManagement"
	"appbundler/utilities/logger"
	"bytes"
	"fmt"
	"os/exec"
	"regexp"
)

// getDefaultSigningIdentity finds the first available code signing certificate in the keychain.
// It uses the macOS "security" command-line tool to query the keychain for valid
// code signing identities (development certificates).
//
// Returns:
//   - The certificate name (e.g., "Apple Development: John Doe (ABCD123456)")
//   - An error if no certificate is found or the security tool fails
func getDefaultSigningIdentity() (string, error) {
	// Find the "security" command-line tool (part of macOS)
	securityPath, err := fileManagement.FindProgramPath("security")
	if err != nil {
		logger.Error(err)
		return "", err
	}

	// Run: security find-identity -p codesigning -v
	// This lists all code signing certificates in the keychain
	cmd := exec.Command(securityPath, "find-identity", "-p", "codesigning", "-v")

	// Capture the command output
	var out bytes.Buffer
	cmd.Stdout = &out

	// Execute the command
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to run security tool: %v", err)
	}

	// Parse the output to extract the certificate name
	// Example output line:
	//   1) ABCDEF1234567890ABCDEF1234567890ABCDEF12 "Apple Development: John Doe (ABCD123456)"
	// The regex captures the quoted certificate name
	re := regexp.MustCompile(`\d+\)\s+[A-F0-9]+\s+"(.+?)"`)
	matches := re.FindStringSubmatch(out.String())
	if len(matches) < 2 {
		return "", fmt.Errorf("no valid code signing identity found in keychain")
	}

	// Return the first matching certificate name
	return matches[1], nil
}

// SignApplication code signs the entire application bundle using Apple's codesign tool.
// This function:
//  1. Finds the codesign tool
//  2. Automatically discovers a signing certificate from the keychain
//  3. Signs the bundle (currently commented out - needs implementation)
//
// Returns an error if:
//   - codesign tool is not found
//   - No signing certificate is available
//   - Signing process fails
//
// Note: The actual signing command is currently commented out and needs to be enabled.
func SignApplication() error {
	// Find the "codesign" command-line tool (part of macOS Xcode Command Line Tools)
	codeSignPath, err := fileManagement.FindProgramPath("codesign")
	if err != nil {
		logger.Error(err)
		return err
	}

	logger.Debug("Program codesign found at: %s", codeSignPath)

	// Automatically find a code signing certificate in the keychain
	identity, err := getDefaultSigningIdentity()
	if err != nil {
		logger.Error(err)
		return err
	}

	logger.Debug("Identity used: %s", identity)

	// The codesign command signs the entire bundle recursively:
	//   --sign: Sign with the specified identity
	//   --deep: Sign nested code (frameworks, helpers, etc.)
	//   --force: Replace existing signature
	//   --options runtime: Enable hardened runtime (required for notarization)
	//   --timestamp: Request timestamp from Apple (required for notarization)
	cmd := exec.Command(codeSignPath, "--sign", identity, "--deep", "--force", "--options", "runtime", "--timestamp", applicationDirectory)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to sign %q: %v\n%s", applicationDirectory, err, stderr.String())
	}

	// Verify the signature after signing
	err = VerifyApplicationSignature(applicationDirectory)
	return err
}

// VerifyApplicationSignature verifies that an application bundle is properly code signed.
// This is useful for testing and ensuring the signing process completed successfully.
//
// Parameters:
//   - appPath: Path to the .app bundle to verify
//
// Returns an error if:
//   - codesign tool is not found
//   - Signature verification fails (invalid, missing, or corrupted signature)
func VerifyApplicationSignature(appPath string) error {
	codeSignPath, err := fileManagement.FindProgramPath("codesign")
	if err != nil {
		logger.Error(err)
		return err
	}

	// Run: codesign --verify --deep --strict --verbose=2 appPath
	//   --verify: Verify the signature
	//   --deep: Verify nested code recursively
	//   --strict: Use strict verification (fails on warnings)
	//   --verbose=2: Show detailed verification information
	cmd := exec.Command(codeSignPath, "--verify", "--deep", "--strict", "--verbose=2", appPath)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("signature verification failed for %q: %v\n%s", appPath, err, stderr.String())
	}
	return nil
}

// NotarizeApplication submits the application bundle to Apple for notarization.
// Notarization is required for distributing apps outside the Mac App Store.
// Apple scans the app for malware and security issues.
//
// Parameters:
//   - applicationRoot: Path to the .app bundle (without .app extension)
//   - appleIDProfile: Keychain profile name containing Apple ID credentials
//
// Returns an error if:
//   - Required tools (zip, xcrun) are not found
//   - Zipping the app fails
//   - Notarization submission fails
//
// Note: The app must be code signed before notarization.
// Note: Notarization requires an Apple Developer account.
func NotarizeApplication(applicationRoot string, appleIDProfile string) error {
	// Apple requires the app to be zipped or in a DMG for notarization
	zipApplication := applicationRoot + ".zip"

	// Find the zip command-line tool
	zipPath, err := fileManagement.FindProgramPath("zip")
	if err != nil {
		logger.Error(err)
		return err
	}

	// Create a zip file containing the entire .app bundle
	// -r flag means recursive (include all files and subdirectories)
	if err = exec.Command(zipPath, "-r", zipApplication, applicationRoot).Run(); err != nil {
		return fmt.Errorf("failed to zip app for notarization: %v", err)
	}

	// Find xcrun (Xcode command-line tool runner)
	xcrunPath, err := fileManagement.FindProgramPath("xcrun")
	if err != nil {
		logger.Error(err)
		return err
	}

	// Submit the zip file to Apple for notarization
	// --keychain-profile: Use stored Apple ID credentials from keychain
	// --wait: Wait for notarization to complete (can take several minutes)
	cmd := exec.Command(xcrunPath, "notarytool", "submit", zipApplication,
		"--keychain-profile", appleIDProfile, "--wait")

	var out, stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("notarization failed: %v\n%s", err, stderr.String())
	}

	logger.Debug("Notarization output:\n%s\n", out.String())
	return nil
}
