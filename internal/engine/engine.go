package engine

import (
	"errors"
	"fmt"
	"maps"
	"os"
	"regexp"
	"slices"
	"strings"

	"github.com/driquet/ezbp/internal/boilerplate"
	"github.com/driquet/ezbp/internal/database"
	"github.com/driquet/ezbp/internal/ui"
)

// Engine manages a collection of boilerplates.
type Engine struct {
	config       Config
	db           database.Database
	ui           ui.UI
	boilerplates map[string]*boilerplate.Boilerplate
}

var (
	ErrBoilerplateAlreadyExist = errors.New("boilerplate already exists")
	ErrBoilerplateUnknown      = errors.New("boilerplate not found")
)

// variableRe is a regular expression used to find variables in boilerplate strings.
// It matches variables in two formats:
// - [[variable_name]]: Represents another boilerplate to be included.
// - {{prompt}}: Represents a user prompt.
var variableRe = regexp.MustCompile(`(\[\[([a-zA-Z0-9_]+)\]\]|{{[^}]+}})`)

// NewEngine creates a new Engine.
// It loads the configuration, initializes the database, loads boilerplates,
// and sets up the UI based on preference (CLI flag > config > default).
func NewEngine(db database.Database, config Config) (*Engine, error) {
	var selectedUI ui.UI
	if config.DefaultUI == "rofi" {
		selectedUI = ui.NewRofiUI(config.Rofi)
	} else { // finalUiChoice == "terminal" or any other fallback
		selectedUI = ui.NewTerminalUI()
	}

	boilerplates, err := db.GetAllBoilerplates()
	if err != nil {
		return nil, err
	}

	return &Engine{
		config:       config,
		db:           db,
		ui:           selectedUI,
		boilerplates: boilerplates,
	}, nil
}

// Names returns the names of the boilerplates.
func (bm *Engine) Names() []string {
	return slices.Sorted(maps.Keys(bm.boilerplates))
}

// Get retrieves a boilerplate by name and returns whether it was found.
func (bm *Engine) Get(name string) (*boilerplate.Boilerplate, bool) {
	bp, found := bm.boilerplates[name]
	return bp, found
}

// SelectBoilerplate prompts the user to select a boilerplate from the available collection.
// It returns the name of the selected boilerplate.
func (bm *Engine) SelectBoilerplate() (string, error) {
	return bm.ui.SelectBoilerplate(bm.boilerplates)
}

// Add creates a new boilerplate with the given name and value.
// Returns an error if the name or value is empty, or if a boilerplate with the same name already exists.
func (bm *Engine) Add(name string, value string) error {
	if name == "" {
		return errors.New("empty boilerplate name")
	}

	if value == "" {
		return errors.New("empty boilerplate value")
	}

	if _, found := bm.boilerplates[name]; found {
		return ErrBoilerplateAlreadyExist
	}

	bp := &boilerplate.Boilerplate{
		Name:  name,
		Value: value,
		Count: 0,
	}

	// Add boilerplate both to local map and database.
	bm.boilerplates[name] = bp
	if err := bm.db.CreateBoilerplate(bp); err != nil {
		return err
	}

	return nil
}

// Edit updates the value of an existing boilerplate.
// Returns an error if the name or value is empty, or if the boilerplate doesn't exist.
func (bm *Engine) Edit(name string, value string) error {
	if name == "" {
		return errors.New("empty boilerplate name")
	}

	if value == "" {
		return errors.New("empty boilerplate value")
	}

	bp, found := bm.boilerplates[name]
	if !found {
		return ErrBoilerplateUnknown
	}

	bp.Value = value

	// Edit boilerplate both in local map and database.
	bm.boilerplates[name] = bp
	if err := bm.db.UpdateBoilerplate(bp); err != nil {
		return err
	}

	return nil
}

// Delete removes a boilerplate by name.
// Returns an error if the name is empty, unknown, or deletion fails.
func (bm *Engine) Delete(name string) error {
	if name == "" {
		return errors.New("empty boilerplate name")
	}

	if _, found := bm.boilerplates[name]; !found {
		return ErrBoilerplateUnknown
	}

	if err := bm.db.DeleteBoilerplate(name); err != nil {
		return err
	}

	return nil
}

// Expand recursively expands a boilerplate template by its name.
// It replaces all variables in the boilerplate string with their corresponding values.
// Variables can be either other boilerplates or user prompts.
// The usage count of the boilerplate is incremented after expansion, both in memory and in the database.
func (bm *Engine) Expand(name string) (string, error) {
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

	if err := bm.incrementBoilerplateCount(name); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to increment count for boilerplate %s in database: %v\n", name, err)
	}

	return after, nil
}

func (bm *Engine) incrementBoilerplateCount(name string) error {
	if _, found := bm.boilerplates[name]; !found {
		return fmt.Errorf("unknown boilerplate %q", name)
	}
	bm.boilerplates[name].Count++

	if err := bm.db.IncBoilerplateCount(name); err != nil {
		return err
	}

	return nil
}

// expandFirst finds and expands the first variable in a boilerplate string.
// Variables are identified by the variableRe regular expression.
// If the variable is a boilerplate inclusion (e.g., "[[another_boilerplate]]"),
// it replaces the variable with the value of the referenced boilerplate.
// If the variable is a user prompt (e.g., "{{Enter your name:}}"),
// it prompts the user for input and replaces the variable with the user's response.
// It can also handle prompts with a fixed set of answers (e.g., "{{Select color|red|green|blue}}").
func (bm *Engine) expandFirst(value string) (string, error) {
	// Find the first variable part to expand using the precompiled regular expression.
	loc := variableRe.FindStringIndex(value)
	if loc == nil {
		// No variable part found, return the original string.
		return value, nil
	}

	// Extract the variable part and its inner content.
	start, end := loc[0], loc[1]
	outerValue := value[start:end]       // e.g., "[[some_boilerplate]]" or "{{some_prompt}}"
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
