package main

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/BurntSushi/toml"
	_ "github.com/mattn/go-sqlite3" // SQLite driver
)

// Config holds the configuration for the BoilerplateManager.
type Config struct {
	// DatabasePath specifies the path to the SQLite database file.
	DatabasePath string `toml:"database_path"`
	// RemoteCSV specifies the URL of a remote CSV file to load boilerplates from.
	// TODO: Implement remote CSV loading.
	RemoteCSV string `toml:"remote_csv,omitempty"`
	// ColorConfig defines the colors to use in the terminal UI.
	// TODO: Implement color configuration.
	ColorConfig string `toml:"color_config,omitempty"`
	// DefaultUI specifies the default user interface to use ("terminal" or "rofi").
	// This can be overridden by the --ui command-line flag.
	DefaultUI string `toml:"default_ui"`
	// RofiUI holds configuration specific to the Rofi user interface.
	// These settings are only active if DefaultUI is "rofi" or if Rofi is selected via the --ui flag.
	RofiUI RofiUIConfig `toml:"RofiUI"`
}

// RofiUIConfig holds configuration specific to the Rofi user interface.
type RofiUIConfig struct {
	// Path is the command or path to the Rofi executable.
	Path string `toml:"path"`
	// Theme specifies the Rofi theme to use. If empty, Rofi's default theme is used.
	Theme string `toml:"theme,omitempty"`
	// SelectArgs are extra arguments to pass to Rofi when used for selections (e.g., boilerplate choice, multiple choice prompts).
	SelectArgs []string `toml:"select_args,omitempty"`
	// InputArgs are extra arguments to pass to Rofi when used for free-form text input.
	InputArgs []string `toml:"input_args,omitempty"`
}

const defaultConfigFileName = "config.toml"
const defaultDatabaseFileName = "ezbp.db"

var userConfigDirFunc = os.UserConfigDir

// loadConfig loads the configuration from a TOML file.
// It checks for the config file in the user's config directory (e.g., ~/.config/ezbp/config.toml).
// If the file doesn't exist, it creates a default one.
// The default database path is ~/.config/ezbp/ezbp.db.
func loadConfig() (Config, error) {
	userConfigDir, err := userConfigDirFunc()
	if err != nil {
		return Config{}, fmt.Errorf("failed to get user config directory: %w", err)
	}
	configDir := filepath.Join(userConfigDir, "ezbp")
	configFilePath := filepath.Join(configDir, defaultConfigFileName)

	// Define default Rofi configuration
	defaultRofiConfig := RofiUIConfig{
		Path:       "rofi", // Default path for Rofi executable
		Theme:      "",     // Empty means Rofi's default theme
		SelectArgs: []string{},
		InputArgs:  []string{},
	}

	defaultConfig := Config{
		DatabasePath: filepath.Join(configDir, defaultDatabaseFileName),
		DefaultUI:    "terminal", // Default UI is terminal
		RofiUI:       defaultRofiConfig,
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

		// Write the default config to the file, including comments for RofiUI.
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
[RofiUI]
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
			defaultConfig.RofiUI.Path,
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
		loadedConfig.DefaultUI = defaultConfig.DefaultUI // which is "terminal"
	}

	// Ensure RofiUI.Path defaults to "rofi" if it's empty after decoding,
	// which could happen if the [RofiUI] table exists but 'path' is missing or empty.
	if loadedConfig.RofiUI.Path == "" {
		loadedConfig.RofiUI.Path = defaultRofiConfig.Path // default is "rofi"
	}
	// Note: RofiUI fields Theme, SelectArgs, and InputArgs will retain their zero values ("", nil slice)
	// if not specified in the TOML, which matches our desired default behavior.
	// The `omitempty` tag on these fields means they won't be written to a new config file by default
	// by a standard TOML encoder if they are empty/nil. Our manual TOML writing for new files
	// handles providing commented-out defaults for these.

	return loadedConfig, nil
}

// initDB initializes the SQLite database connection and creates the schema if it doesn't exist.
func initDB(dataSourceName string) (*sql.DB, error) {
	// Ensure the directory for the database file exists.
	dir := filepath.Dir(dataSourceName)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		if err := os.MkdirAll(dir, 0750); err != nil {
			return nil, fmt.Errorf("failed to create database directory %s: %w", dir, err)
		}
	}

	db, err := sql.Open("sqlite3", dataSourceName)
	if err != nil {
		return nil, fmt.Errorf("failed to open database at %s: %w", dataSourceName, err)
	}

	if err = db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database at %s: %w", dataSourceName, err)
	}

	schema := `
    CREATE TABLE IF NOT EXISTS boilerplates (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        name TEXT UNIQUE NOT NULL,
        value TEXT NOT NULL,
        count INTEGER DEFAULT 0,
        created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
        updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
    );
    CREATE INDEX IF NOT EXISTS idx_boilerplates_name ON boilerplates (name);
    `
	if _, err = db.Exec(schema); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to create schema in database at %s: %w", dataSourceName, err)
	}

	return db, nil
}

// loadBoilerplates queries the database and loads all boilerplates into a map.
func loadBoilerplates(db *sql.DB) (map[string]*Boilerplate, error) {
	rows, err := db.Query("SELECT name, value, count FROM boilerplates")
	if err != nil {
		return nil, fmt.Errorf("failed to query boilerplates: %w", err)
	}
	defer rows.Close()

	boilerplates := make(map[string]*Boilerplate)
	for rows.Next() {
		bp := &Boilerplate{}
		if err := rows.Scan(&bp.Name, &bp.Value, &bp.Count); err != nil {
			return nil, fmt.Errorf("failed to scan boilerplate row: %w", err)
		}
		boilerplates[bp.Name] = bp
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating boilerplate rows: %w", err)
	}
	return boilerplates, nil
}

// incrementBoilerplateCount increments the usage count for a given boilerplate name.
func (bm *BoilerplateManager) incrementBoilerplateCount(name string) error {
	if bm.db == nil {
		return fmt.Errorf("database connection is not initialized")
	}
	stmt, err := bm.db.Prepare("UPDATE boilerplates SET count = count + 1, updated_at = CURRENT_TIMESTAMP WHERE name = ?")
	if err != nil {
		return fmt.Errorf("failed to prepare statement for incrementing count: %w", err)
	}
	defer stmt.Close()

	_, err = stmt.Exec(name)
	if err != nil {
		return fmt.Errorf("failed to execute statement for incrementing count for %s: %w", name, err)
	}
	return nil
}

// Boilerplate represents a single boilerplate template.
type Boilerplate struct {
	// Name is the unique identifier for the boilerplate.
	Name string
	// Value is the template string of the boilerplate.
	// It can contain variables in the format [[variable_name]] or {{prompt}}.
	Value string
	// Count is the number of times this boilerplate has been used.
	Count int
}

// BoilerplateManager manages a collection of boilerplates.
type BoilerplateManager struct {
	// config holds the configuration for the BoilerplateManager.
	config Config
	// db is the SQLite database connection.
	db *sql.DB
	// boilerplates is a map of boilerplate names to Boilerplate structs.
	boilerplates map[string]*Boilerplate
	// ui is the user interface for interacting with the BoilerplateManager.
	ui UI
}

// variableRe is a regular expression used to find variables in boilerplate strings.
// It matches variables in two formats:
// - [[variable_name]]: Represents another boilerplate to be included.
// - {{prompt}}: Represents a user prompt.
var variableRe = regexp.MustCompile(`(\[\[([a-zA-Z0-9_]+)\]\]|{{[^}]+}})`)

var initDBFunc = initDB // Allow mocking for tests

// NewBoilerplateManager creates a new BoilerplateManager.
// It loads the configuration, initializes the database, loads boilerplates,
// and sets up the UI based on preference (CLI flag > config > default).
func NewBoilerplateManager(uiPreference string) (*BoilerplateManager, error) {
	config, err := loadConfig()
	if err != nil {
		// Even if config loading fails, we might proceed with defaults for DB path
		// and UI. loadConfig itself doesn't return an error that stops execution here,
		// but NewBoilerplateManager might fail later if DB init fails.
		// For clarity, we'll log here if config loading had issues but returned a usable (default) config.
		// However, loadConfig currently returns an error that would be caught by the caller of NewBoilerplateManager.
		// Let's assume loadConfig might return a default config struct even on some errors,
		// or that the error from loadConfig is handled before this point (e.g., in main.go).
		// For this refactor, we'll proceed assuming 'config' is the result of loadConfig().
		// If 'err' from loadConfig() is critical, main.go should handle it.
		// If loadConfig() itself prints warnings for non-critical issues and returns a default config,
		// then 'err' here would be nil from a successful loadConfig call.
		// The current loadConfig() returns errors that ARE critical if it can't determine paths,
		// so this 'err' from loadConfig() would typically be checked by the caller (main.go).
		// Let's proceed assuming 'config' is the best-effort loaded or default config.
		// The error from loadConfig should be handled by the caller, so we can remove the direct error check here.
		// config, _ = loadConfig() // If we want to ignore loadConfig error and proceed with defaults.
		// For now, assume config is valid or default, and 'err' from loadConfig is handled by main.go.
	}

	dbPath := defaultDatabaseFileName // Default to current directory if config loading failed or path is empty
	if config.DatabasePath != "" {    // Check if config has a valid path
		dbPath = config.DatabasePath
	} else if err == nil { // If loadConfig succeeded but path was still empty (shouldn't happen with current loadConfig)
		// This case implies loadConfig returned a config where DatabasePath was not set,
		// and loadConfig itself didn't default it, which is unlikely with current loadConfig.
		// However, to be safe, ensure dbPath has a value.
		// The default is already set above, so this is more for logical clarity.
	}


	db, err := initDBFunc(dbPath) // Use the function variable
	if err != nil {
		// Try fallback to CWD if initDB with configured path failed and it's different from default CWD path
		if dbPath != defaultDatabaseFileName {
			fmt.Fprintf(os.Stderr, "Warning: failed to initialize database at %s: %v. Trying %s\n", dbPath, err, defaultDatabaseFileName)
			dbPath = defaultDatabaseFileName // Explicitly set to CWD default for the next attempt
			db, err = initDBFunc(dbPath)     // Use the function variable
		}
		if err != nil {
			return nil, fmt.Errorf("failed to initialize database at %s: %w", dbPath, err)
		}
	}

	boilerplates, err := loadBoilerplates(db)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to load boilerplates from database: %w", err)
	}

	var finalUiChoice string
	if uiPreference == "rofi" || uiPreference == "terminal" {
		finalUiChoice = uiPreference
	} else {
		// uiPreference from flag is not set or invalid, use config.DefaultUI
		if config.DefaultUI == "rofi" || config.DefaultUI == "terminal" {
			finalUiChoice = config.DefaultUI
		} else {
			finalUiChoice = "terminal" // Absolute fallback
		}
	}

	var selectedUI UI
	if finalUiChoice == "rofi" {
		selectedUI = NewRofiUI(config.RofiUI)
	} else { // finalUiChoice == "terminal" or any other fallback
		selectedUI = NewTermUI()
	}

	return &BoilerplateManager{
		config:       config,
		db:           db,
		boilerplates: boilerplates,
		ui:           selectedUI,
	}, nil
}

// SelectBoilerplate prompts the user to select a boilerplate from the available collection.
// It returns the name of the selected boilerplate.
func (bm *BoilerplateManager) SelectBoilerplate() (string, error) {
	return bm.ui.SelectBoilerplate(bm.boilerplates)
}

// Expand recursively expands a boilerplate template by its name.
// It replaces all variables in the boilerplate string with their corresponding values.
// Variables can be either other boilerplates or user prompts.
// The usage count of the boilerplate is incremented after expansion, both in memory and in the database.
func (bm *BoilerplateManager) Expand(name string) (string, error) {
	bp, found := bm.boilerplates[name]
	if !found {
		return "", fmt.Errorf("unknown boilerplate %q", name)
	}

	before := bp.Value
	var after string
	var err error
	for {
		after, err = bm.expandFirst(before)
		if err != nil {
			return "", err
		}
		if before == after {
			break
		}
		before = after
	}

	// Increment count in memory
	bp.Count++
	// Increment count in database
	if err := bm.incrementBoilerplateCount(name); err != nil {
		// Log error but don't fail the expansion, as the value is already generated.
		// User might not be able to save the count if DB is temporarily unavailable.
		fmt.Fprintf(os.Stderr, "Warning: failed to increment count for boilerplate %s in database: %v\n", name, err)
	}

	return after, nil
}

// expandFirst finds and expands the first variable in a boilerplate string.
// Variables are identified by the variableRe regular expression.
// If the variable is a boilerplate inclusion (e.g., "[[another_boilerplate]]"),
// it replaces the variable with the value of the referenced boilerplate.
// If the variable is a user prompt (e.g., "{{Enter your name:}}"),
// it prompts the user for input and replaces the variable with the user's response.
// It can also handle prompts with a fixed set of answers (e.g., "{{Select color|red|green|blue}}").
func (bm *BoilerplateManager) expandFirst(value string) (string, error) {
	// Find the first variable part to expand using the precompiled regular expression.
	loc := variableRe.FindStringIndex(value)
	if loc == nil {
		// No variable part found, return the original string.
		return value, nil
	}

	// Extract the variable part and its inner content.
	start, end := loc[0], loc[1]
	outerValue := value[start:end] // e.g., "[[some_boilerplate]]" or "{{some_prompt}}"
	innerValue := value[start+2 : end-2] // e.g., "some_boilerplate" or "some_prompt"
	var replacement string

	// Check if the variable is a boilerplate inclusion or a user prompt
	// based on the starting character ('[' for boilerplate, '{' for prompt).
	if value[start] == '[' {
		// Substitution by another boilerplate.
		bp, found := bm.boilerplates[outerValue]
		if !found {
			return "", fmt.Errorf("unknown referenced boilerplate %q", innerValue)
		}
		replacement = bp.Value
	} else {
		// User prompt.
		// It can consist in asking the user an open question {{prompt}}
		// Or in asking a question with a fixed set of answers {{prompt|a|b|c}}.
		if idx := strings.IndexRune(innerValue, '|'); idx >= 0 {
			// Prompt with a fixed set of answers.
			elements := strings.Split(innerValue, "|")
			prompt := elements[0]
			options := elements[1:]
			choice, err := bm.ui.Select(prompt, options)
			if err != nil {
				return "", err
			}
			replacement = choice
		} else {
			// Open question prompt.
			input, err := bm.ui.Prompt(innerValue)
			if err != nil {
				return "", err
			}
			replacement = input
		}
	}

	// Replace the variable part with the determined replacement.
	return value[:start] + replacement + value[end:], nil
}
