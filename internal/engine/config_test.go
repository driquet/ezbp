package engine

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadConfig_NoConfigFileExists(t *testing.T) {
	configDir := t.TempDir()

	config, err := LoadConfigFromFile(configDir)
	require.NoError(t, err, "loadConfig() should not return an error when no config exists")

	expectedConfigFilePath := filepath.Join(configDir, defaultConfigFileName)
	expectedDatabasePath := filepath.Join(configDir, defaultDatabaseFileName)
	assert.FileExists(t, expectedConfigFilePath, "config.toml should be created")

	// Verify default values
	assert.Equal(t, "terminal", config.DefaultUI, "DefaultUI should be 'terminal'")
	assert.Equal(t, "rofi", config.Rofi.Path, "RofiUI.Path should be 'rofi'")
	assert.Equal(t, expectedDatabasePath, config.DatabasePath, "DatabasePathe should be '%s'", expectedConfigFilePath)
}

func TestLoadConfig_ConfigFileExistsValid(t *testing.T) {
	configDir := t.TempDir()

	customDatabasePath := filepath.Join(t.TempDir(), "custom_ezbp.db")

	configFilePath := filepath.Join(configDir, defaultConfigFileName)

	// Test with a config file that specifies default_ui = "rofi" and a custom rofi path
	customRofiPath := "/usr/local/bin/rofi-custom"
	fileContent := []byte(fmt.Sprintf(`
database_path = "%s"
default_ui = "rofi"
[rofi]
  path = "%s"
`, customDatabasePath, customRofiPath))
	err := os.WriteFile(configFilePath, fileContent, 0600)
	require.NoError(t, err)

	config, err := LoadConfigFromFile(configDir)
	require.NoError(t, err)
	assert.Equal(t, customDatabasePath, config.DatabasePath)
	assert.Equal(t, "rofi", config.DefaultUI)
	assert.Equal(t, customRofiPath, config.Rofi.Path)
}

func TestLoadConfig_ConfigFileExistsInvalidDefaultUI(t *testing.T) {
	configDir := t.TempDir()

	configFilePath := filepath.Join(configDir, defaultConfigFileName)
	fileContent := []byte(`default_ui = "invalid_ui_value"`)
	err := os.WriteFile(configFilePath, fileContent, 0600)
	require.NoError(t, err)

	config, err := LoadConfigFromFile(configDir)
	require.NoError(t, err)

	// DefaultUI should be defaulted to "terminal"
	assert.Equal(t, "terminal", config.DefaultUI, "DefaultUI should default to 'terminal' if invalid value in config")
}

func TestLoadConfig_ConfigFileExistsMissingPath(t *testing.T) {
	configDir := t.TempDir()

	configFilePath := filepath.Join(configDir, defaultConfigFileName)
	fileContent := []byte(`default_ui = "rofi"`)
	err := os.WriteFile(configFilePath, fileContent, 0600)
	require.NoError(t, err)

	config, err := LoadConfigFromFile(configDir)
	require.NoError(t, err)

	expectedDatabasePath := filepath.Join(configDir, defaultDatabaseFileName)
	assert.Equal(t, expectedDatabasePath, config.DatabasePath, "DatabasePath should default if missing in config file")
}

func TestLoadConfig_ConfigFileExistsMalformed(t *testing.T) {
	configDir := t.TempDir()

	configFilePath := filepath.Join(configDir, defaultConfigFileName)
	fileContent := []byte(`database_path = "this is not valid toml`) // Malformed TOML
	err := os.WriteFile(configFilePath, fileContent, 0600)
	require.NoError(t, err)

	_, err = LoadConfigFromFile(configDir)
	require.Error(t, err, "loadConfig should return an error for malformed TOML")
}
