package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/BurntSushi/toml"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "github.com/mattn/go-sqlite3" // SQLite driver for tests
)

// setupTestDB creates an initialized SQLite database for testing.
// It can create either an in-memory DB or a file-based DB in a temporary directory.
func setupTestDB(t *testing.T, useFile bool) *sql.DB {
	t.Helper()
	var dbPath string
	if useFile {
		dbPath = filepath.Join(t.TempDir(), "test.db")
	} else {
		// For in-memory, ensure each test gets a unique DB by using a unique name.
		// Using "file::memory:?cache=shared" with a unique name allows multiple connections to the same in-memory DB if needed.
		// For isolated tests, just ":memory:" is fine if the *sql.DB is not closed and reopened.
		// To be safe and simple for isolated tests, we'll use a unique file path in a temp dir.
		// Or, more simply for truly isolated in-memory: "file:memdbN?mode=memory&cache=shared" where N is unique.
		// However, the simplest for a single *sql.DB instance per test is just ":memory:".
		// Let's use a temporary file path for consistency with initDB's dir creation logic.
		dbPath = filepath.Join(t.TempDir(), fmt.Sprintf("test-%s.db", t.Name()))
	}

	// Use the real initDB function to create the schema.
	// The initDBFunc var is for mocking NewBoilerplateManager's internal call.
	db, err := initDB(dbPath)
	require.NoError(t, err, "Failed to initialize test database with schema")

	// Cleanup: Close the database connection after the test.
	t.Cleanup(func() {
		err := db.Close()
		if err != nil {
			t.Logf("Warning: error closing test database: %v", err)
		}
		if useFile || (filepath.Ext(dbPath) == ".db") { // Attempt to remove if it was a file
			os.Remove(dbPath)
		}
	})
	return db
}

func TestLoadConfig_NoConfigFileExists(t *testing.T) {
	tempHome := t.TempDir()
	originalUserConfigDirFunc := userConfigDirFunc
	userConfigDirFunc = func() (string, error) { return tempHome, nil }
	defer func() { userConfigDirFunc = originalUserConfigDirFunc }()

	config, err := loadConfig()
	require.NoError(t, err, "loadConfig() should not return an error when no config exists")

	expectedConfigDir := filepath.Join(tempHome, "ezbp")
	expectedConfigFilePath := filepath.Join(expectedConfigDir, defaultConfigFileName)
	expectedDatabasePath := filepath.Join(expectedConfigDir, defaultDatabaseFileName)

	assert.FileExists(t, expectedConfigFilePath, "config.toml should be created")

	assert.Equal(t, expectedDatabasePath, config.DatabasePath, "DatabasePath should be the default path")
	assert.Equal(t, "terminal", config.DefaultUI, "DefaultUI should be 'terminal'")
	assert.Equal(t, "rofi", config.RofiUI.Path, "RofiUI.Path should be 'rofi'")

	// Verify content of created config.toml
	fileContent, err := os.ReadFile(expectedConfigFilePath)
	require.NoError(t, err, "Failed to read created config.toml")
	contentStr := string(fileContent)
	assert.Contains(t, contentStr, `default_ui = "terminal"`, "Default config file should contain default_ui = 'terminal'")
	assert.NotContains(t, contentStr, "enabled =", "Default config file should not contain 'enabled =' for RofiUI")
	assert.Contains(t, contentStr, `path = "rofi"`, "Default config file should contain rofi path")

	// Decode and check struct again to be sure
	var createdConfig Config
	_, err = toml.DecodeFile(expectedConfigFilePath, &createdConfig)
	require.NoError(t, err, "Decoding created config.toml failed")
	assert.Equal(t, expectedDatabasePath, createdConfig.DatabasePath)
	assert.Equal(t, "terminal", createdConfig.DefaultUI)
	assert.Equal(t, "rofi", createdConfig.RofiUI.Path)
}

func TestLoadConfig_ConfigFileExistsValid(t *testing.T) {
	tempHome := t.TempDir()
	originalUserConfigDirFunc := userConfigDirFunc
	userConfigDirFunc = func() (string, error) { return tempHome, nil }
	defer func() { userConfigDirFunc = originalUserConfigDirFunc }()

	customDatabasePath := filepath.Join(t.TempDir(), "custom_ezbp.db")
	ezbpConfigDir := filepath.Join(tempHome, "ezbp")
	err := os.MkdirAll(ezbpConfigDir, 0750)
	require.NoError(t, err)

	configFilePath := filepath.Join(ezbpConfigDir, defaultConfigFileName)
	// Test with a config file that specifies default_ui = "rofi" and a custom rofi path
	customRofiPath := "/usr/local/bin/rofi-custom"
	fileContent := []byte(fmt.Sprintf(`
database_path = "%s"
default_ui = "rofi"
[RofiUI]
  path = "%s"
`, customDatabasePath, customRofiPath))
	err = os.WriteFile(configFilePath, fileContent, 0600)
	require.NoError(t, err)

	config, err := loadConfig()
	require.NoError(t, err)
	assert.Equal(t, customDatabasePath, config.DatabasePath)
	assert.Equal(t, "rofi", config.DefaultUI)
	assert.Equal(t, customRofiPath, config.RofiUI.Path)
}

func TestLoadConfig_ConfigFileExistsInvalidDefaultUI(t *testing.T) {
	tempHome := t.TempDir()
	originalUserConfigDirFunc := userConfigDirFunc
	userConfigDirFunc = func() (string, error) { return tempHome, nil }
	defer func() { userConfigDirFunc = originalUserConfigDirFunc }()

	ezbpConfigDir := filepath.Join(tempHome, "ezbp")
	err := os.MkdirAll(ezbpConfigDir, 0750)
	require.NoError(t, err)

	configFilePath := filepath.Join(ezbpConfigDir, defaultConfigFileName)
	fileContent := []byte(`default_ui = "invalid_ui_value"`)
	err = os.WriteFile(configFilePath, fileContent, 0600)
	require.NoError(t, err)

	config, err := loadConfig()
	require.NoError(t, err)
	// DefaultUI should be defaulted to "terminal"
	assert.Equal(t, "terminal", config.DefaultUI, "DefaultUI should default to 'terminal' if invalid value in config")
}


func TestLoadConfig_ConfigFileExistsMissingPath(t *testing.T) {
	tempHome := t.TempDir()
	originalUserConfigDirFunc := userConfigDirFunc
	userConfigDirFunc = func() (string, error) { return tempHome, nil }
	defer func() { userConfigDirFunc = originalUserConfigDirFunc }()

	ezbpConfigDir := filepath.Join(tempHome, "ezbp")
	err := os.MkdirAll(ezbpConfigDir, 0750)
	require.NoError(t, err)

	configFilePath := filepath.Join(ezbpConfigDir, defaultConfigFileName)
	fileContent := []byte(`remote_csv = "http://example.com/remote.csv"`)
	err = os.WriteFile(configFilePath, fileContent, 0600)
	require.NoError(t, err)

	config, err := loadConfig()
	require.NoError(t, err)

	expectedDefaultPath := filepath.Join(ezbpConfigDir, defaultDatabaseFileName)
	assert.Equal(t, expectedDefaultPath, config.DatabasePath, "DatabasePath should default if missing in config file")
}

func TestLoadConfig_ConfigFileExistsMalformed(t *testing.T) {
	tempHome := t.TempDir()
	originalUserConfigDirFunc := userConfigDirFunc
	userConfigDirFunc = func() (string, error) { return tempHome, nil }
	defer func() { userConfigDirFunc = originalUserConfigDirFunc }()

	ezbpConfigDir := filepath.Join(tempHome, "ezbp")
	err := os.MkdirAll(ezbpConfigDir, 0750)
	require.NoError(t, err)

	configFilePath := filepath.Join(ezbpConfigDir, defaultConfigFileName)
	fileContent := []byte(`database_path = "this is not valid toml`) // Malformed TOML
	err = os.WriteFile(configFilePath, fileContent, 0600)
	require.NoError(t, err)

	_, err = loadConfig()
	require.Error(t, err, "loadConfig should return an error for malformed TOML")
}

func TestLoadConfig_UserConfigDirError(t *testing.T) {
	originalUserConfigDirFunc := userConfigDirFunc
	userConfigDirFunc = func() (string, error) {
		return "", fmt.Errorf("simulated user config dir error")
	}
	defer func() { userConfigDirFunc = originalUserConfigDirFunc }()

	_, err := loadConfig()
	require.Error(t, err, "loadConfig() should return an error if UserConfigDir fails")
	assert.Contains(t, err.Error(), "failed to get user config directory")
}

func TestInitDB_Real(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test_init.db")
	db, err := initDB(dbPath) // Use the real initDB
	require.NoError(t, err)
	defer db.Close()
	defer os.Remove(dbPath)

	// Check if table exists
	var tableName string
	err = db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='boilerplates';").Scan(&tableName)
	require.NoError(t, err, "boilerplates table should exist")
	assert.Equal(t, "boilerplates", tableName)

	// Check if index exists
	var indexName string
	err = db.QueryRow("SELECT name FROM sqlite_master WHERE type='index' AND name='idx_boilerplates_name';").Scan(&indexName)
	require.NoError(t, err, "idx_boilerplates_name index should exist")
	assert.Equal(t, "idx_boilerplates_name", indexName)

	// Test idempotency: Call initDB again on the same DB
	db.Close() // Close first connection
	db2, err := initDB(dbPath)
	require.NoError(t, err, "initDB should be idempotent and not fail on existing DB")
	defer db2.Close()
	// Basic check to ensure connection is valid
	err = db2.Ping()
	require.NoError(t, err, "Ping should succeed on re-initialized DB")
}

func TestLoadBoilerplates(t *testing.T) {
	db := setupTestDB(t, false) // Use in-memory DB for this test helper

	// 1. Test empty database
	bps, err := loadBoilerplates(db)
	require.NoError(t, err)
	assert.Empty(t, bps, "Should return empty map for empty DB")

	// 2. Test with data
	_, err = db.Exec("INSERT INTO boilerplates (name, value, count) VALUES (?, ?, ?), (?, ?, ?)",
		"bp1", "value1", 10,
		"bp2", "value2", 20)
	require.NoError(t, err)

	bps, err = loadBoilerplates(db)
	require.NoError(t, err)
	require.Len(t, bps, 2, "Should load 2 boilerplates")
	assert.Equal(t, "value1", bps["bp1"].Value)
	assert.Equal(t, 10, bps["bp1"].Count)
	assert.Equal(t, "value2", bps["bp2"].Value)
	assert.Equal(t, 20, bps["bp2"].Count)
}

func TestBoilerplateManager_IncrementBoilerplateCount(t *testing.T) {
	db := setupTestDB(t, false)
	bm := &BoilerplateManager{db: db} // Directly set the DB for testing this unit

	// Insert a test boilerplate
	_, err := db.Exec("INSERT INTO boilerplates (name, value, count) VALUES (?, ?, ?)", "test_bp", "test_value", 5)
	require.NoError(t, err)

	err = bm.incrementBoilerplateCount("test_bp")
	require.NoError(t, err)

	var newCount int
	err = db.QueryRow("SELECT count FROM boilerplates WHERE name = ?", "test_bp").Scan(&newCount)
	require.NoError(t, err)
	assert.Equal(t, 6, newCount, "Count should be incremented")

	// Test incrementing non-existing boilerplate (should not error, but also not change anything)
	err = bm.incrementBoilerplateCount("non_existent_bp")
	require.NoError(t, err) // Assuming it's designed to not error out
}

func TestExpand(t *testing.T) {
	// Store original initDBFunc and restore after test
	originalInitDB := initDBFunc
	defer func() { initDBFunc = originalInitDB }()

	// Mock initDBFunc to use a test DB
	var testDB *sql.DB
	initDBFunc = func(dataSourceName string) (*sql.DB, error) {
		// The dataSourceName passed by NewBoilerplateManager (based on config) will be ignored.
		// We return a connection to our per-test in-memory DB.
		// We re-initialize testDB for each call to initDBFunc that this test might trigger (though it should be once)
		testDB = setupTestDB(t, false) // false for in-memory
		return testDB, nil
	}
	
	// Populate the test database (which testDB now points to)
	// Need to get a handle to testDB. This is a bit tricky because initDBFunc creates it.
	// A simpler way: setupTestDB *outside* and have initDBFunc *return* it.
	// Let's refine the mocking for testDB handle:
	
	currentTestDB := setupTestDB(t, false) // Create a single test DB for this test case.
	initDBFunc = func(dataSourceName string) (*sql.DB, error) {
		return currentTestDB, nil // Always return this specific test DB
	}


	_, err := currentTestDB.Exec("INSERT INTO boilerplates (name, value, count) VALUES (?, ?, ?), (?, ?, ?), (?, ?, ?)",
		"foobar", "the text of foobar", 0,
		"barfoo", "before [[foobar]] after", 0,
		"fizzbuzz", "john [[barfoo]] doe", 0)
	require.NoError(t, err)

	// NewBoilerplateManager will now use the mocked initDBFunc,
	// which returns our currentTestDB connection.
	// We pass a dummy config path, as it will be ignored by the mocked initDBFunc.
	// However, loadConfig is still called, so we need to mock userConfigDirFunc.
	tempHome := t.TempDir()
	originalUserConfigDir := userConfigDirFunc
	userConfigDirFunc = func() (string, error) { return tempHome, nil }
	defer func() { userConfigDirFunc = originalUserConfigDir }()


	bm, err := NewBoilerplateManager("") // Pass empty uiPreference, relying on config
	require.NoError(t, err, "NewBoilerplateManager failed")
	bm.ui = &NoopUI{} // Use a UI that doesn't require interaction


	testCases := map[string]struct {
		name          string
		expected      string
		expectedCount int
	}{
		"no variable": {
			name:          "foobar",
			expected:      "the text of foobar",
			expectedCount: 1,
		},
		"substitution": {
			name:          "barfoo",
			expected:      "before the text of foobar after",
			expectedCount: 1, // count for barfoo
		},
		"nested substitution": {
			name:          "fizzbuzz",
			expected:      "john before the text of foobar after doe",
			expectedCount: 1, // count for fizzbuzz
		},
	}

	for testName, tc := range testCases {
		t.Run(testName, func(t *testing.T) {
			// Reset count for foobar if it's involved, as it's used as a sub-boilerplate
			if tc.name == "barfoo" || tc.name == "fizzbuzz" {
				_, err := currentTestDB.Exec("UPDATE boilerplates SET count = 0 WHERE name = ?", "foobar")
				require.NoError(t, err)
				bm.boilerplates["foobar"].Count = 0 // also reset in-memory map
			}
			if tc.name == "fizzbuzz" { // if fizzbuzz, barfoo is also a sub-boilerplate
				_, err := currentTestDB.Exec("UPDATE boilerplates SET count = 0 WHERE name = ?", "barfoo")
				require.NoError(t, err)
				bm.boilerplates["barfoo"].Count = 0
			}


			got, err := bm.Expand(tc.name)
			require.NoError(t, err)
			assert.Equal(t, tc.expected, got)

			// Verify count in DB
			var count int
			err = currentTestDB.QueryRow("SELECT count FROM boilerplates WHERE name = ?", tc.name).Scan(&count)
			require.NoError(t, err, "Failed to query count for %s", tc.name)
			assert.Equal(t, tc.expectedCount, count, "Count in DB mismatch for %s", tc.name)

			// Verify count in memory map
			assert.Equal(t, tc.expectedCount, bm.boilerplates[tc.name].Count, "Count in memory map mismatch for %s", tc.name)
		})
	}
}


// NoopUI is a UI implementation that does nothing, for tests that don't need UI interaction.
type NoopUI struct{}

func (n *NoopUI) SelectBoilerplate(boilerplates map[string]*Boilerplate) (string, error) { return "", nil }
func (n *NoopUI) Select(prompt string, choices []string) (string, error)                { return "", nil }
func (n *NoopUI) Prompt(prompt string) (string, error)                                  { return "", nil }

func TestNewBoilerplateManager_UISelection(t *testing.T) {
	// Mock userConfigDirFunc for all sub-tests
	tempHome := t.TempDir()
	originalUserConfigDir := userConfigDirFunc
	userConfigDirFunc = func() (string, error) { return tempHome, nil }
	defer func() { userConfigDirFunc = originalUserConfigDir }()

	// Mock initDBFunc for all sub-tests
	originalInitDB := initDBFunc
	defer func() { initDBFunc = originalInitDB }()

	currentTestDB := setupTestDB(t, false) // Single in-memory DB for all these UI selection tests
	initDBFunc = func(dataSourceName string) (*sql.DB, error) {
		return currentTestDB, nil
	}

	tests := []struct {
		name             string
		uiPreference     string       // Value for --ui flag
		configDefaultUI  string       // Value for config.DefaultUI
		expectUIType     reflect.Type // Expected type of bm.ui
		configShouldFail bool         // If true, simulate loadConfig returning an error
	}{
		{
			name:            "CLI Flag Rofi",
			uiPreference:    "rofi",
			configDefaultUI: "terminal", // Config should be overridden
			expectUIType:    reflect.TypeOf(&RofiUI{}),
		},
		{
			name:            "CLI Flag Terminal",
			uiPreference:    "terminal",
			configDefaultUI: "rofi", // Config should be overridden
			expectUIType:    reflect.TypeOf(&TermUI{}),
		},
		{
			name:            "Config Rofi (CLI flag empty)",
			uiPreference:    "",
			configDefaultUI: "rofi",
			expectUIType:    reflect.TypeOf(&RofiUI{}),
		},
		{
			name:            "Config Terminal (CLI flag empty)",
			uiPreference:    "",
			configDefaultUI: "terminal",
			expectUIType:    reflect.TypeOf(&TermUI{}),
		},
		{
			name:            "Config Invalid (CLI flag empty)",
			uiPreference:    "",
			configDefaultUI: "invalid_ui", // Invalid config value
			expectUIType:    reflect.TypeOf(&TermUI{}), // Should default to TermUI
		},
		{
			name:            "CLI Flag Invalid, Config Rofi",
			uiPreference:    "invalid_flag_ui",
			configDefaultUI: "rofi",
			expectUIType:    reflect.TypeOf(&RofiUI{}), // Should use config
		},
		{
			name:            "CLI Flag Invalid, Config Terminal",
			uiPreference:    "invalid_flag_ui",
			configDefaultUI: "terminal",
			expectUIType:    reflect.TypeOf(&TermUI{}), // Should use config
		},
		{
			name:            "CLI Flag Invalid, Config Invalid",
			uiPreference:    "invalid_flag_ui",
			configDefaultUI: "invalid_config_ui",
			expectUIType:    reflect.TypeOf(&TermUI{}), // Should default to TermUI
		},
		{
			name:             "LoadConfig Fails, CLI Flag Rofi",
			uiPreference:     "rofi",
			configShouldFail: true, // Simulate loadConfig error
			expectUIType:     reflect.TypeOf(&RofiUI{}),
		},
		{
			name:             "LoadConfig Fails, CLI Flag Terminal",
			uiPreference:     "terminal",
			configShouldFail: true, // Simulate loadConfig error
			expectUIType:     reflect.TypeOf(&TermUI{}),
		},
		{
			name:             "LoadConfig Fails, No CLI Flag (fallback to terminal)",
			uiPreference:     "",
			configShouldFail: true, // Simulate loadConfig error
			expectUIType:     reflect.TypeOf(&TermUI{}),
		},
	}

	originalLoadConfig := loadConfig
	defer func() { loadConfig = originalLoadConfig }()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Mock loadConfig to return specific config values or an error
			loadConfig = func() (Config, error) {
				if tt.configShouldFail {
					// Return a default-like config even on error, as NewBM might try to use it
					return Config{
						DatabasePath: filepath.Join(tempHome, "ezbp", defaultDatabaseFileName), // Default path
						DefaultUI:    "terminal", // Default UI
						RofiUI:       RofiUIConfig{Path: "rofi"},
					}, fmt.Errorf("simulated loadConfig error")
				}
				// This is the config NewBoilerplateManager will see
				return Config{
					DatabasePath: filepath.Join(tempHome, "ezbp", defaultDatabaseFileName), // Needs a valid path for initDB
					DefaultUI:    tt.configDefaultUI,
					RofiUI:       RofiUIConfig{Path: "rofi"}, // Basic Rofi config
				}, nil
			}

			bm, err := NewBoilerplateManager(tt.uiPreference)
			// We expect NewBoilerplateManager to succeed even if loadConfig simulated an error,
			// as long as DB init can proceed (which it should with mocked initDBFunc).
			// The warnings from loadConfig failure would be printed to os.Stderr.
			require.NoError(t, err, "NewBoilerplateManager failed for: %s", tt.name)
			assert.NotNil(t, bm, "BoilerplateManager should not be nil for: %s", tt.name)
			assert.IsType(t, tt.expectUIType, bm.ui, "UI type mismatch for: %s", tt.name)
		})
	}
}


func TestMain(m *testing.M) {
	// This TestMain is a good place if we needed global setup/teardown,
	// but individual test setup (like t.TempDir, setupTestDB) is often preferred for isolation.
	// For now, it just runs the tests.
	os.Exit(m.Run())
			},
		},
	}

	for name, test := range map[string]struct {
		name     string
		expected string
	}{
		"no variable": {
			name:     "foobar",
			expected: "the text of foobar",
		},
		"substitution": {
			name:     "barfoo",
			expected: "before the text of foobar after",
		},
		"nested substitution": {
			name:     "fizzbuzz",
			expected: "john before the text of foobar after doe",
		},
	} {
		t.Run(name, func(t *testing.T) {
			got, err := bm.Expand(test.name)
			require.NoError(t, err)
			assert.Equal(t, test.expected, got)
		})
	}
}
