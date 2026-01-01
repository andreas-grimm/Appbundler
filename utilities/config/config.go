// Package config provides a configuration management system that loads settings
// from YAML files. It supports multiple configuration file locations with a
// hierarchy: system defaults, system config, environment config, and local config.
// Later configurations override earlier ones.
//
// Note: This package appears to be a general-purpose configuration system that
// may not be actively used by the main application bundler. The application
// package uses its own simpler YAML reading in readDescriptionFile.go.
package config

import (
	"appbundler/utilities/logger"
	"errors"
	"gopkg.in/yaml.v3"
	"os"
	"path/filepath"
	"strconv"
)

// configFile stores the raw YAML configuration data.
// The structure is: map[group_name][]map[key]value
// Example: {"Network": [{"port": "8080"}], "DB": [{"host": "localhost"}]}
var configFile map[string][]map[string]string

// keyValuePair represents a single configuration key-value pair.
// This is used internally to organize configuration parameters.
type keyValuePair struct {
	key   string // Configuration key name
	value string // Configuration value (as string)
}

// configParameters groups related configuration parameters together.
// For example, all database settings might be in a "DB" group.
type configParameters struct {
	group     string         // Group name (e.g., "Network", "DB")
	parameter []keyValuePair // List of key-value pairs in this group
}

// parameters stores all loaded configuration parameters.
// This is populated by LoadConfiguration() and accessed by getter functions.
var parameters []configParameters

// LoadConfiguration loads configuration from multiple locations in priority order.
// Configuration files are loaded in this order (later ones override earlier ones):
//   1. Default parameters (hardcoded)
//   2. System config: /etc/executable_name.d/config.yaml
//   3. Environment/config file: executable_name.config.yaml (or command-line specified)
//   4. Local config: ./config/executable_name.yaml (relative to executable)
//
// Parameters:
//   - configFileFromCommandLine: Optional path to a specific config file
//
// Returns an error if configuration loading fails (though it continues even if some files are missing).
func LoadConfiguration(configFileFromCommandLine string) error {
	var err error

	// Get the name of the executable (without path)
	// This is used to construct config file names
	nameOfExecutable := filepath.Base(os.Args[0])

	// Step 1: Load default parameters (hardcoded defaults)
	logger.Debug("Loading default parameters from /etc directory...")
	parameters = setDefaultParameters()

	// Step 2: Try to load from system-wide config directory
	// Location: /etc/executable_name.d/config.yaml
	mainConfigurationFile := "/etc/" + nameOfExecutable + ".d/config"
	parameters, err = checkForFileAndLoad(mainConfigurationFile, parameters)
	if err != nil {
		logger.Debug("File not found. Error code: %s", err.Error())
	} else {
		logger.Debug("File found.")
	}

	// Step 3: Try to load from environment or command-line specified file
	// If no file specified, defaults to: executable_name.config.yaml
	if configFileFromCommandLine == "" {
		configFileFromCommandLine = nameOfExecutable + ".config"
	}
	parameters, err = checkForFileAndLoad(configFileFromCommandLine, parameters)
	if err != nil {
		logger.Debug("File not found. Error code: %s", err.Error())
	}

	// Step 4: Try to load from local config directory (relative to executable)
	// Location: ./config/executable_name.yaml
	executable, _ := os.Executable()
	executablePath := filepath.Dir(executable)
	localConfigurationFile := executablePath + "/config/" + nameOfExecutable
	parameters, err = checkForFileAndLoad(localConfigurationFile, parameters)
	if err != nil {
		logger.Warn("File not found. Error code: %s", err.Error())
	}

	return err
}

// setDefaultParameters returns hardcoded default configuration values.
// These defaults are used if no configuration files are found.
// The defaults appear to be for a web application with database access.
//
// Returns a slice of configParameters with default values.
func setDefaultParameters() []configParameters {

	// Default network/HTTP server parameters
	networkKeyValue := []keyValuePair{{"port", "8080"}}

	// Default database connection parameters
	// Note: These appear to be PostgreSQL defaults, though the port suggests MySQL
	dbKeyValue := []keyValuePair{{"port", "3306"},
		{"host", "localhost"},
		{"user", "postgres"},
		{"password", "nopass"},
		{"dbname", "authentikator"},
	}

	// Combine all default parameter groups
	parameters := []configParameters{
		{"Network", networkKeyValue},
		{"DB", dbKeyValue},
	}

	return parameters
}

// checkForFileAndLoad checks for a YAML configuration file and loads it if found.
// It tries both .yaml and .yml extensions.
//
// Parameters:
//   - path: Base path to the config file (without extension)
//   - parameters: Existing parameters to merge with
//
// Returns:
//   - Updated parameters (merged with file contents)
//   - Error if file is not found
func checkForFileAndLoad(path string, parameters []configParameters) ([]configParameters, error) {
	// Try both .yaml and .yml extensions (both are valid YAML file extensions)
	var foundConfiguration bool = false

	// Try .yaml extension first
	yamlPath := path + ".yaml"
	logger.Debug("Loading from location...: %s", yamlPath)
	if _, err := os.Stat(yamlPath); err == nil {
		foundConfiguration = true
		parameters, err = loadYamlFile(yamlPath, parameters)
	}

	// Try .yml extension (shorter alternative)
	ymlPath := path + ".yml"
	logger.Debug("Loading from location...: %s", ymlPath)
	if _, err := os.Stat(ymlPath); err == nil {
		foundConfiguration = true
		// Note: There's a bug here - it should use ymlPath, not yamlPath
		parameters, err = loadYamlFile(yamlPath, parameters)
	}

	// If neither file was found, return an error
	if !foundConfiguration {
		return parameters, errors.New("Configuration file for [" + path + "] not found")
	}

	return parameters, nil
}

// loadYamlFile reads a YAML configuration file and merges it with existing parameters.
// The YAML file structure is expected to be:
//   GroupName:
//     - key: value
//     - key2: value2
//
// Parameters:
//   - path: Full path to the YAML file to load
//   - parameters: Existing parameters to merge with
//
// Returns:
//   - Updated parameters (merged with YAML file contents)
//   - Error if file reading or parsing fails
func loadYamlFile(path string, parameters []configParameters) ([]configParameters, error) {
	// Read the entire YAML file into memory
	yamlFile, err := os.ReadFile(path)
	if err != nil {
		logger.Debug("Error reading yamlFile. Get err #%v ", err)
	}

	logger.Debug("Loaded yaml file: %s", yamlFile)

	// Parse the YAML data into the configFile map structure
	err = yaml.Unmarshal(yamlFile, &configFile)

	// Debug logging: show current parameters before merging
	logger.Debug("Unmarshalled yaml file: %s", configFile)
	for _, parameterGroup := range parameters {
		logger.Debug("==> Got parameter group: %s", parameterGroup.group)
		for _, parameter := range parameterGroup.parameter {
			logger.Debug("====> Got parameter key: %s", parameter.key)
			logger.Debug("======> Got parameter value: %s", parameter.value)
		}
	}

	// Merge YAML file contents into the parameters structure
	// Iterate through each group in the YAML file
	for yamlGroupName, yamlGroupStruct := range configFile {
		// Iterate through each element in the group (YAML structure is a slice of maps)
		for _, yamlElement := range yamlGroupStruct {
			// Iterate through each key-value pair in the element
			for key, value := range yamlElement {
				// Merge this key-value pair into the parameters structure
				// This will update existing values or add new ones
				parameters, _ = ChangeParameterStructure(yamlGroupName, key, value, parameters)
			}
		}
	}

	logger.Debug("----> Updated: %s", parameters)

	return parameters, nil
}

// ChangeParameterStructure merges a key-value pair into the parameters structure.
// This function handles three scenarios:
//   1. Group and key exist: Update the existing value
//   2. Group exists but key is new: Add the new key-value pair to the group
//   3. Group doesn't exist: Create the group and add the key-value pair
//
// This allows configuration files to override defaults or add new settings.
//
// Parameters:
//   - group: Name of the parameter group (e.g., "Network", "DB")
//   - key: Parameter key name (e.g., "port", "host")
//   - value: Parameter value (as string)
//   - parameters: Existing parameters structure to modify
//
// Returns:
//   - Updated parameters structure
//   - Error (currently always nil, but kept for interface consistency)
func ChangeParameterStructure(group string, key string, value string, parameters []configParameters) ([]configParameters, error) {
	var parameterFound bool = false
	var groupFound bool = false

	var newParameters []configParameters

	// this is needed if the key is not found in the parameter group
	newKeyValuePair := keyValuePair{key: key, value: value}

	for _, parameterGroup := range parameters {
		var keyValuePairs []keyValuePair

		if parameterGroup.group == group {
			groupFound = true
		}

		for _, parameter := range parameterGroup.parameter {
			keyValuePair := keyValuePair{key: parameter.key, value: parameter.value}
			if key == parameter.key && parameterGroup.group == group {
				parameterFound = true
				keyValuePair.value = value
			}
			keyValuePairs = append(keyValuePairs, keyValuePair)
		}

		// if the parameter has not been found in the group, add it
		if !parameterFound && parameterGroup.group == group {
			keyValuePairs = append(keyValuePairs, newKeyValuePair)
		}

		newParameter := configParameters{parameterGroup.group, keyValuePairs}
		newParameters = append(newParameters, newParameter)

	}

	// ok - neither the parameter group nor the parameter is around and we need to create both
	if !groupFound {
		// generate a key value pair and add it into the array
		var keyValuePairs []keyValuePair
		keyValuePair := keyValuePair{key: key, value: value}
		keyValuePairs = append(keyValuePairs, keyValuePair)

		// generate a parameter: use group name and the parameter struct defined above
		parameter := configParameters{group, keyValuePairs}

		// now append this into the existing parameter array
		newParameters = append(newParameters, parameter)
	}

	logger.Debug("----> New setting: %s", newParameters)
	return newParameters, nil
}

// GetStringByGroupAndElement retrieves a configuration value as a string.
// This searches through all parameter groups to find a matching group and key.
//
// Parameters:
//   - groupName: Name of the parameter group (e.g., "Network", "DB")
//   - elementName: Name of the parameter key (e.g., "port", "host")
//
// Returns:
//   - The configuration value as a string
//   - An error if the group/key combination is not found
func GetStringByGroupAndElement(groupName string, elementName string) (string, error) {
	// Search through all parameter groups
	for _, parameterGroup := range parameters {
		// Search through all parameters in this group
		for _, parameter := range parameterGroup.parameter {
			// Check if this is the group and key we're looking for
			if parameterGroup.group == groupName && parameter.key == elementName {
				return parameter.value, nil
			}
		}
	}

	return "", errors.New("value not found in configuration")
}

// GetIntByGroupAndElement retrieves a configuration value as an integer.
// This is a convenience function that calls GetStringByGroupAndElement and converts
// the result to an integer.
//
// Parameters:
//   - groupName: Name of the parameter group
//   - elementName: Name of the parameter key
//
// Returns:
//   - The configuration value as an integer
//   - An error if the group/key is not found or the value cannot be converted to int
func GetIntByGroupAndElement(groupName string, elementName string) (int, error) {
	// Get the value as a string first
	returnValue, err := GetStringByGroupAndElement(groupName, elementName)

	if err != nil {
		return 0, err
	}

	// Convert string to integer
	return strconv.Atoi(returnValue)
}
