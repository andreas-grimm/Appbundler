# AppBundler

AppBundler is a Go-based tool designed to package Java applications (JAR files) or compiled binaries into professional macOS application bundles (`.app`). It automates the creation of the directory structure, generation of `Info.plist`, bundling of icons, and optionally handles code signing and notarization.

## Features

- **Automated Bundle Creation**: Generates the standard macOS `.app` directory hierarchy.
- **Java Support**: Specifically handles JAR files by creating a native bash launcher script.
- **Optional Java Runtime Bundling**: Can include a local Java runtime (JRE/JDK) inside the app bundle for self-contained distribution.
- **Metadata Management**: Easily configure `Info.plist` properties via a YAML file.
- **Code Signing**: Integrates with Apple's `codesign` tool to sign your application.
- **Notarization**: Supports the `xcrun notarytool` workflow for Apple notarization.
- **Pre-flight Validation**: Verifies source files exist before starting the build process.

## Installation

Ensure you have Go installed on your system, then clone the repository and build the binary:

```bash
go build -o appbundler main.go
```

## Usage

Run the `appbundler` executable with the desired flags:

```bash
./appbundler [flags]
```

### Command-Line Flags

| Flag | Default | Description |
| :--- | :--- | :--- |
| `-application` | `application.yaml` | Path to the YAML configuration file. |
| `-app` | `my_app` | Override the application name (overrides the `name` in YAML). |
| `-clean` | `false` | Remove existing `.app` bundle before rebuilding. |
| `-sign` | `false` | Sign the bundle with a valid development certificate from your keychain. |
| `-notarize` | `false` | Submit the application for Apple notarization. |
| `-profile` | (empty) | Apple ID keychain profile name (required for `-notarize`). |
| `-silent` | `false` | Suppress informational log messages. |
| `-logdir` | (empty) | Directory to save log files (enables file logging). |
| `-delete` | `false` | Delete the created bundle after building (mainly for testing). |

## Configuration (`application.yaml`)

The tool uses a YAML file to define the application's metadata and build settings.

### Example `application.yaml`

```yaml
# Basic Metadata
id: "com.example.myapp"
name: "MyApp"
display_name: "My Awesome Application"
version: "1.0.0"
short_version_string: "1.0"
executable: "launcher" # The name of the launcher script in Contents/MacOS
signature: "APPL"
readable_copyright: "Copyright © 2024 Example Inc."

# File Locations
exec_file: "my-app.jar"
exec_file_directory: "./dist"
icon_file: "appIcon.icns"
icon_file_directory: "./assets"

# Optional macOS Properties
system_minimal_os_version: "10.13.0"
principle_class: "NSApplication"

# Java Settings
local_java: "true" # Set "true" to bundle a local JRE/JDK
local_java_home: "/Library/Java/JavaVirtualMachines/zulu-17.jdk/Contents/Home"
```

### Configuration Fields

- **`id`**: Unique bundle identifier (e.g., `com.company.app`).
- **`name`**: Internal bundle name.
- **`executable`**: The name of the binary/script that macOS will execute.
- **`exec_file`**: The source JAR or binary to be packaged.
- **`local_java`**: Set to `"true"` to enable bundling of a Java runtime.
- **`local_java_home`**: Path to the Java installation you want to bundle.

## Workflow

1. **Validation**: Checks if the JAR/binary, icon, and Java Home (if enabled) exist.
2. **Directory Structure**: Creates `Contents/MacOS`, `Contents/Resources`, and `Contents/Java/runtime`.
3. **Plist Generation**: Creates `Info.plist` and `PkgInfo`.
4. **Copying**: 
    - Copies the icon to `Resources`.
    - Copies the JAR/binary to `MacOS`.
    - If `local_java` is true, copies the entire Java runtime to `Java/runtime`.
5. **Launcher**: Creates a bash script in `MacOS` that sets `JAVA_HOME` and executes the JAR.
6. **Signing**: Runs `codesign` with hardened runtime and timestamping.
7. **Notarization**: Zips the app and submits it via `notarytool`.

## Requirements

- **macOS**: This tool is designed to run on macOS.
- **Xcode Command Line Tools**: Required for `codesign` and `notarytool`.
- **Certificates**: A valid Apple Development or Distribution certificate in your keychain is required for signing.
