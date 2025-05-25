package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

// Config holds the configuration for the BoilerplateManager.
type Config struct {
	// DatabasePath specifies the path to the SQLite database file.
	DatabasePath string `toml:"database_path"`
	// DefaultUI specifies the default user interface to use ("terminal" or "rofi").
	// This can be overridden by the --ui command-line flag.
	DefaultUI string `toml:"default_ui"`
	// TODO: Description
	// TODO: Update default configuration file
	Editor string `toml:"editor"`
	// Rofi holds configuration specific to the Rofi user interface.
	// These settings are only active if DefaultUI is "rofi" or if Rofi is selected via the --ui flag.
	Rofi RofiConfig `toml:"rofi"`
}

const (
	defaultConfigFileName   = "config.toml"
	defaultDatabaseFileName = "ezbp.db"
)

// TODO:
func configDirPath() (string, error) {
	userConfigPath, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("unable to retrieve user config path: %w", err)
	}

	configDirPath := filepath.Join(userConfigPath, "ezbp")

	// Create directory if needed
	if _, err := os.Stat(configDirPath); os.IsNotExist(err) {
		// Config file does not exist, create it with default values.
		if err := os.MkdirAll(configDirPath, 0750); err != nil {
			return "", fmt.Errorf("failed to create config directory %s: %w", configDirPath, err)
		}
	}

	return configDirPath, nil
}

// loadConfigFromFile loads the configuration from a TOML file.
// It checks for the config file in the user's config directory (e.g., ~/.config/ezbp/config.toml).
// If the file doesn't exist, it creates a default one.
// The default database path is ~/.config/ezbp/ezbp.db.
func loadConfigFromFile(configDir string) (Config, error) {
	configFilePath := filepath.Join(configDir, defaultConfigFileName)

	// Define default Rofi configuration
	defaultRofiConfig := RofiConfig{
		Path:       "rofi", // Default path for Rofi executable
		Theme:      "",     // Empty means Rofi's default theme
		SelectArgs: []string{},
		InputArgs:  []string{},
	}

	defaultConfig := Config{
		DatabasePath: filepath.Join(configDir, defaultDatabaseFileName),
		DefaultUI:    "terminal", // Default UI is terminal
		Rofi:         defaultRofiConfig,
	}

	if _, err := os.Stat(configFilePath); os.IsNotExist(err) {
		// Config file does not exist, create it with default values.
		if err := os.MkdirAll(configDir, 0750); err != nil {
			return Config{}, fmt.Errorf("failed to create config directory %s: %w", configDir, err)
		}
		f, err := os.Create(configFilePath)
		if err != nil {
			return Config{}, fmt.Errorf("failed to create config file %s: %w", configFilePath, err)
		}
		defer f.Close()

		// Write the default config to the file, including comments for Rofi.
		// The direct toml.Encoder.Encode doesn't easily support comments for individual fields in a nested struct
		// in the way we want for a default config file.
		// So, we'll manually construct the TOML content for a new file to include comments.
		defaultTomlContent := fmt.Sprintf(`database_path = "%s"
# remote_csv = ""
# color_config = ""

# default_ui specifies the default user interface.
# Valid options are "terminal" or "rofi".
# This can be overridden by the --ui command-line flag.
default_ui = "%s"

# Rofi User Interface settings
# These settings are used if default_ui = "rofi" or --ui=rofi is specified.
[rofi]
  # Path to the Rofi executable.
  path = "%s"
  # Optional: Specify a Rofi theme file (e.g., "solarized", "dracula").
  # If empty, Rofi's default theme or theme specified in Rofi's own config will be used.
  # theme = ""
  # Extra arguments to pass to Rofi for selection dialogs (e.g., choosing a boilerplate).
  # Example: select_args = ["-i", "-p", "Choose:"] (case-insensitive, custom prompt)
  # select_args = []
  # Extra arguments to pass to Rofi for input dialogs (e.g., free-form text prompts).
  # Example: input_args = ["-password"] (for password-style input)
  # input_args = []
`, defaultConfig.DatabasePath, // Use Go's string formatting to escape path if needed
			defaultConfig.DefaultUI,
			defaultConfig.Rofi.Path,
		)

		if _, err := f.WriteString(defaultTomlContent); err != nil {
			return Config{}, fmt.Errorf("failed to write default config content to %s: %w", configFilePath, err)
		}

		// Return the defaultConfig struct which has all defaults correctly set.
		return defaultConfig, nil
	} else if err != nil {
		return Config{}, fmt.Errorf("failed to stat config file %s: %w", configFilePath, err)
	}

	var loadedConfig Config
	if _, err := toml.DecodeFile(configFilePath, &loadedConfig); err != nil {
		return Config{}, fmt.Errorf("failed to decode config file %s: %w", configFilePath, err)
	}

	if loadedConfig.DatabasePath == "" {
		loadedConfig.DatabasePath = defaultConfig.DatabasePath
	}

	// Validate DefaultUI or set to default
	if loadedConfig.DefaultUI != "rofi" && loadedConfig.DefaultUI != "terminal" {
		loadedConfig.DefaultUI = defaultConfig.DefaultUI
	}

	// Ensure Rofi.Path defaults to "rofi" if it's empty after decoding,
	// which could happen if the [Rofi] table exists but 'path' is missing or empty.
	if loadedConfig.Rofi.Path == "" {
		loadedConfig.Rofi.Path = defaultRofiConfig.Path
	}

	return loadedConfig, nil
}
