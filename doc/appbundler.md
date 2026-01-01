# AppBundler: Low-Level Technical Documentation

This document provides a detailed technical deep-dive into the `appbundler` project. It is intended for junior Go developers who want to understand the internal workings, architecture, and implementation details of the tool.

---

## 1. Project Architecture Overview

`appbundler` is structured as a modular Go command-line tool. It follows a functional pipeline approach to transform a set of input files (executable/JAR, icon, configuration) into a standard macOS `.app` bundle.

### Directory Structure
- `main.go`: Entry point, CLI flag parsing, and high-level orchestration.
- `application/`: Core business logic for macOS bundle creation.
  - `readDescriptionFile.go`: YAML configuration parsing and data access.
  - `directoryManagement.go`: Filesystem structure creation and cleanup.
  - `pListCreator.go`: XML generation for `Info.plist` and `PkgInfo`.
  - `copyExecutable.go`: Logic for handling binaries and Java JARs (including launcher scripts).
  - `copyIcon.go`: Asset management for the application icon.
  - `signApplication.go`: Integration with Apple's `codesign` and `notarytool`.
- `utilities/`: Cross-cutting concerns.
  - `fileManagement/`: Low-level file/directory operations and path discovery.
  - `logger/`: Internal logging system.
  - `config/`: (Optional) Global configuration settings.

---

## 2. Core Package: `application`

### Configuration Management (`readDescriptionFile.go`)
The configuration is driven by a YAML file (usually `application.yaml`). 
- **Implementation**: Uses `gopkg.in/yaml.v3`.
- **Data Model**: The `packageParameter` struct uses struct tags (e.g., `yaml:"id"`) to map YAML keys to Go fields.
- **Access Pattern**: The `packageInfo` variable is private to the package. Access is provided via public getter functions (e.g., `GetBundleName()`). This encapsulates the data and allows for future validation logic within the getters.
- **Validation**: `ValidateConfiguration()` performs "pre-flight" checks using `os.Stat` to ensure source files exist before the build starts, preventing half-finished bundles.

### Bundle Structure (`directoryManagement.go`)
macOS apps are specifically structured directories.
- **Constants & Variables**: The package maintains state for key directory paths (`contentsDir`, `macosDir`, etc.) after they are initialized.
- **`CreateDirectoryStructure`**: Uses `os.MkdirAll` with `0755` permissions. It is designed to be atomic-like: if any step fails, it calls `DeleteAll()` to clean up.

### The Plist Engine (`pListCreator.go`)
- **Go Templates**: Uses `text/template`. This is a powerful way to generate structured text (XML) by injecting Go struct data into a predefined template string (`plistTemplate`).
- **`InfoPlistData`**: A dedicated "Data Transfer Object" (DTO) struct used specifically for the template engine, separating the raw config from the template needs.
- **PkgInfo**: Generates the legacy 8-byte `PkgInfo` file by concatenating the 4-byte Package Type and 4-byte Signature.

### Executable & Java Support (`copyExecutable.go`)
This is the most complex part of the pipeline.
- **JAR vs. Binary**: The tool detects `.jar` extensions to switch logic.
- **Launcher Script**: For Java, it generates a Bash script using `fmt.Sprintf`.
  - **Dynamic Pathing**: The script uses `DIR="$(cd "$(dirname "$0")" && pwd)"` to ensure it works regardless of where the app is launched from.
  - **Bundled JRE**: If `local_java` is true, the script sets `JAVA_HOME` to point *inside* the bundle.
- **Permissions**: Crucially uses `os.Chmod(path, 0755)` to ensure the copied binary or generated script is actually executable by the OS.

### Security Integration (`signApplication.go`)
- **Subprocess Execution**: Uses `os/exec` to call system tools (`codesign`, `security`, `xcrun`).
- **Regex Parsing**: `getDefaultSigningIdentity()` runs `security find-identity` and uses a regular expression (`regexp` package) to scrape the certificate name from the command's standard output.
- **Hardened Runtime**: When signing, it passes `--options runtime` and `--timestamp`, which are mandatory requirements for modern macOS notarization.

---

## 3. Utilities & Best Practices

### File Management (`utilities/fileManagement/`)
- **Path Discovery**: `FindProgramPath` uses `exec.LookPath` to find system tools (like `codesign`) in the user's `$PATH`.
- **Recursive Copy**: `CopyDirectory` is implemented to handle the recursive transfer of the Java runtime environment.

### Logger (`utilities/logger/`)
- A wrapper around the standard `log` package.
- Supports "Silent Mode" via a boolean flag and "File Logging" by redirecting output to an `os.File`.

---

## 4. The Build Pipeline (Orchestration)

The `main.go` file acts as the "Controller". It follows this strict execution order:

1. **Initialization**: Parse CLI flags (`flag` package).
2. **Config Load**: `application.Read()`.
3. **Pre-flight**: `application.ValidateConfiguration()`.
4. **Log Setup**: Optional file logging.
5. **Execution Steps**:
   - `CreateDirectoryStructure`
   - `CreatePlist`
   - `CopyExecutable`
   - `CopyIcon`
   - `SignApplication` (Optional)
   - `NotarizeApplication` (Optional)
6. **Error Handling**: Every function returns an `error`. The `errorExit` helper ensures that the first failure stops the process and logs the cause.

---

## 5. Tips for Junior Developers

1. **Error Handling**: In Go, always check the `error` return value. In this project, errors are bubbled up to `main.go` where they trigger a process exit.
2. **Pathing**: Always use `filepath.Join` instead of manual string concatenation for paths. This ensures cross-platform compatibility (handling `/` vs `\`).
3. **Subprocesses**: When using `os/exec`, always capture `Stderr` to a `bytes.Buffer`. If the command fails, the error message from the tool (e.g., Apple's `codesign`) is usually in `Stderr`, not `Stdout`.
4. **Deferred Cleanup**: While `os.RemoveAll` is used manually here for cleanup, remember `defer` for closing files (`file.Close()`) to prevent resource leaks.
5. **Templates**: If you need to add new keys to `Info.plist`, you must update the `plistTemplate` string, the `InfoPlistData` struct, and the mapping logic in `CreatePlist()`.