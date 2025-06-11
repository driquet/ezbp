package ui

import (
	"bytes"
	"errors"
	"fmt"
	"os/exec"
	"sort"
	"strings"

	"github.com/driquet/ezbp/internal/boilerplate"
)

// RofiConfig holds configuration specific to the Rofi user interface.
type RofiConfig struct {
	// Path is the command or path to the Rofi executable.
	Path string `toml:"path"`
	// Theme specifies the Rofi theme to use. If empty, Rofi's default theme is used.
	Theme string `toml:"theme,omitempty"`
	// SelectArgs are extra arguments to pass to Rofi when used for selections (e.g., boilerplate choice, multiple choice prompts).
	SelectArgs []string `toml:"select_args,omitempty"`
	// InputArgs are extra arguments to pass to Rofi when used for free-form text input.
	InputArgs []string `toml:"input_args,omitempty"`
}

// RofiUI implements the UI interface using Rofi for user interactions.
type RofiUI struct {
	config RofiConfig // Holds Rofi-specific configuration from main Config
}

// NewRofiUI creates a new RofiUI instance with the given Rofi configuration.
func NewRofiUI(config RofiConfig) UI {
	return &RofiUI{config: config}
}

// runRofi executes a Rofi command with the given arguments and input string.
// It returns the selected string or an error.
func (u *RofiUI) runRofi(prompt string, input string, args []string) (string, error) {
	cmdArgs := []string{"-dmenu"}
	if prompt != "" {
		cmdArgs = append(cmdArgs, "-p", prompt)
	}

	if u.config.Theme != "" {
		cmdArgs = append(cmdArgs, "-theme", u.config.Theme)
	}
	cmdArgs = append(cmdArgs, args...)

	cmd := exec.Command(u.config.Path, cmdArgs...)

	var stdout, stderr bytes.Buffer
	cmd.Stdin = strings.NewReader(input)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		// Check if the error is due to user cancellation (e.g., pressing Esc).
		// Rofi typically exits with status 1 on Esc. Other non-zero exits might be actual errors.
		var exitError *exec.ExitError
		if errors.As(err, &exitError) {
			// Exit code 1 is often user cancellation.
			// Exit code 10, 11, 12, 13 are for custom keybindings in Rofi.
			// We'll treat exit code 1 as cancellation.
			if exitError.ExitCode() == 1 {
				return "", ErrUserAborted
			}
		}
		return "", fmt.Errorf("rofi command failed: %w\nStderr: %s", err, stderr.String())
	}

	selected := strings.TrimSpace(stdout.String())
	// If Rofi was cancelled in a way that results in a 0 exit code but empty output (less common),
	// also treat as cancellation.
	if selected == "" && input != "" { // only consider empty output as cancellation if there was input to select from
		return "", ErrUserAborted
	}

	return selected, nil
}

// SelectBoilerplate implements the UI interface method for selecting a boilerplate using Rofi.
func (u *RofiUI) SelectBoilerplate(boilerplates map[string]*boilerplate.Boilerplate) (string, error) {
	var bps []*boilerplate.Boilerplate
	for _, bp := range boilerplates {
		bps = append(bps, bp)
	}
	sort.Slice(bps, func(i, j int) bool {
		return bps[i].Count > bps[j].Count
	})

	var rofiInput strings.Builder

	for _, bp := range bps {
		// Format: "123 boilerplate_name" - Rofi will display this.
		displayString := fmt.Sprintf("%5d %s", bp.Count, bp.Name)
		rofiInput.WriteString(displayString + "\n")
	}

	selected, err := u.runRofi("Select Boilerplate", rofiInput.String(), u.config.SelectArgs)
	if err != nil {
		return "", err
	}

	// Remove the boilerplate count prefix from the selected display string.
	parts := strings.Fields(selected)
	if len(parts) <= 1 {
		return "", fmt.Errorf("incorrect format for the rofi selection")
	}

	name := strings.Join(parts[1:], " ")
	if _, found := boilerplates[name]; !found {
		return "", fmt.Errorf("selected boilerplate display string %q not found in original list", selected)
	}

	return name, nil
}

// Select implements the UI interface method for selecting from a list of choices using Rofi.
func (u *RofiUI) Select(prompt string, choices []string) (string, error) {
	if len(choices) == 0 {
		return "", fmt.Errorf("no choices provided for selection")
	}
	rofiInput := strings.Join(choices, "\n")
	return u.runRofi(prompt, rofiInput, u.config.SelectArgs)
}

// Prompt implements the UI interface method for prompting the user for input using Rofi.
func (u *RofiUI) Prompt(prompt string) (string, error) {
	// For text input, Rofi's dmenu typically expects no stdin, or specific flags.
	// We pass an empty input string and rely on runRofi's handling for input mode.
	// Additional args for input mode are taken from u.config.InputArgs.
	response, err := u.runRofi(prompt, "", u.config.InputArgs)
	if err != nil {
		return "", err
	}
	// If Rofi input is cancelled (e.g. Esc), runRofi should return ErrUserAborted.
	// If it returns empty string for other reasons (e.g. user just hits enter),
	// it's still a valid (empty) input.
	return response, nil
}
