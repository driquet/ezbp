package main

import (
	"fmt"
	"os"
	"os/exec"
	"reflect"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMain is required for the TestHelperProcess pattern.
func TestMain(m *testing.M) {
	// Before all tests:
	// We could set up global mocks here if absolutely necessary,
	// but it's generally better to set them up per-test or per-suite.

	retCode := m.Run()

	// After all tests:
	// Teardown global mocks if any.

	os.Exit(retCode)
}

// mockExecCommand sets up the TestHelperProcess pattern for a test.
// It replaces the global execCommand with a version that calls the test binary itself.
// The actual command name and args are passed as arguments to the helper process.
func mockExecCommand(t *testing.T) {
	originalExecCommand := execCommand
	execCommand = func(name string, args ...string) *exec.Cmd {
		// Prepend the helper process indicator and the original command/args
		cs := []string{"-test.run=TestHelperProcess", "--", name}
		cs = append(cs, args...)
		cmd := exec.Command(os.Args[0], cs...)
		// Set an environment variable to signal that this is a helper process execution
		cmd.Env = append(cmd.Env, "GO_WANT_HELPER_PROCESS=1")
		return cmd
	}
	// Restore the original execCommand after the test
	t.Cleanup(func() {
		execCommand = originalExecCommand
	})
}

// TestHelperProcess isn't a real test but a helper sub-process
// invoked by mocked exec.Command calls.
func TestHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return // Not in helper process mode
	}
	defer os.Exit(0) // Ensure the helper process exits

	args := os.Args // os.Args[0] is test binary, os.Args[1] is "-test.run=TestHelperProcess", os.Args[2] is "--"
	cmdArgs := args[3:] // Actual command and its arguments

	// cmdArgs[0] is the command (e.g., "rofi")
	// cmdArgs[1:] are its arguments

	// Simulate Rofi behavior based on arguments
	// This part needs to be carefully crafted for each test case's expectation.
	// For example, if "rofi" is called with "-dmenu" and "-p" for a prompt:
	if cmdArgs[0] == "rofi" {
		// Check for specific test case scenarios via environment variables
		// set by the main test function before calling the UI method.
		rofiSimulate := os.Getenv("ROFI_SIMULATE")
		switch rofiSimulate {
		case "SELECT_BOILERPLATE_SUCCESS":
			// Expecting input like "  count name\n..."
			// Simulate user selecting "   2 bp2"
			fmt.Fprint(os.Stdout, "   2 bp2")
			os.Exit(0)
		case "SELECT_BOILERPLATE_CANCEL":
			os.Exit(1) // Rofi exits with 1 on cancel
		case "SELECT_CHOICE_SUCCESS":
			// Simulate user selecting "Choice2"
			fmt.Fprint(os.Stdout, "Choice2")
			os.Exit(0)
		case "SELECT_CHOICE_CANCEL":
			os.Exit(1)
		case "PROMPT_SUCCESS":
			// Simulate user entering "User Input Text"
			fmt.Fprint(os.Stdout, "User Input Text")
			os.Exit(0)
		case "PROMPT_CANCEL":
			os.Exit(1)
		case "ROFI_ERROR":
			fmt.Fprint(os.Stderr, "Rofi critical error")
			os.Exit(2) // Some other error code
		case "ROFI_NOT_FOUND":
			// This case is harder to simulate here as exec.Command itself would error earlier.
			// The mockExecCommand would return an exec.Cmd that points to a non-existent helper,
			// or the error is simulated directly in the test.
			// For this helper, we'll assume 'rofi' was "found" (as this helper is 'rofi').
			// The actual "not found" error is tested by making execCommand return exec.ErrNotFound.
			os.Exit(127) // Command not found
		default:
			// Default behavior if no specific simulation is set, or for unexpected calls
			fmt.Fprintf(os.Stderr, "TestHelperProcess: Unhandled Rofi simulation: %s with args %v\n", rofiSimulate, cmdArgs)
			os.Exit(127) // Generic error
		}
	} else {
		fmt.Fprintf(os.Stderr, "TestHelperProcess: Unhandled command: %s\n", cmdArgs[0])
		os.Exit(127)
	}
}

func TestRofiUI_SelectBoilerplate(t *testing.T) {
	boilerplates := map[string]*Boilerplate{
		"bp1": {Name: "bp1", Value: "val1", Count: 10},
		"bp2": {Name: "bp2", Value: "val2", Count: 2}, // This will be selected by mock
		"bp3": {Name: "bp3", Value: "val3", Count: 5},
	}
	rofiConfig := RofiUIConfig{Path: "rofi", Theme: "test_theme.rasi", SelectArgs: []string{"-i"}}

	t.Run("Success", func(t *testing.T) {
		mockExecCommand(t)
		os.Setenv("ROFI_SIMULATE", "SELECT_BOILERPLATE_SUCCESS")
		defer os.Unsetenv("ROFI_SIMULATE")

		ui := NewRofiUI(rofiConfig)
		selected, err := ui.SelectBoilerplate(boilerplates)

		require.NoError(t, err)
		assert.Equal(t, "bp2", selected)
	})

	t.Run("Cancellation", func(t *testing.T) {
		mockExecCommand(t)
		os.Setenv("ROFI_SIMULATE", "SELECT_BOILERPLATE_CANCEL")
		defer os.Unsetenv("ROFI_SIMULATE")

		ui := NewRofiUI(rofiConfig)
		_, err := ui.SelectBoilerplate(boilerplates)

		assert.ErrorIs(t, err, ErrUserAborted, "Expected ErrUserAborted on Rofi cancellation")
	})

	t.Run("Rofi Error", func(t *testing.T) {
		mockExecCommand(t)
		os.Setenv("ROFI_SIMULATE", "ROFI_ERROR")
		defer os.Unsetenv("ROFI_SIMULATE")

		ui := NewRofiUI(rofiConfig)
		_, err := ui.SelectBoilerplate(boilerplates)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "Rofi critical error", "Error message should include Rofi's stderr")
	})

	t.Run("Rofi Not Found", func(t *testing.T) {
		originalExecCommand := execCommand
		execCommand = func(name string, args ...string) *exec.Cmd {
			// Simulate the error that os/exec.LookPath returns when a command is not found.
			// This error is then wrapped by exec.Cmd.Start or exec.Cmd.Run.
			// The key is that the error is of type *exec.Error and has Err == exec.ErrNotFound.
			return &exec.Cmd{Path: name, Args: args, Err: exec.ErrNotFound}
		}
		t.Cleanup(func() { execCommand = originalExecCommand })

		ui := NewRofiUI(RofiUIConfig{Path: "nonexistentrofi"}) // Config path that won't be found
		_, err := ui.SelectBoilerplate(boilerplates)

		require.Error(t, err, "SelectBoilerplate should error if Rofi is not found")
		// Note: The actual error wrapping might make it hard to directly assert exec.ErrNotFound
		// without more complex error unwrapping, depending on how `cmd.Run()` reports it.
		// The error message from `runRofi` will typically include "executable file not found".
		assert.Contains(t, err.Error(), "executable file not found", "Error message should indicate Rofi not found")
	})
}

func TestRofiUI_Select(t *testing.T) {
	choices := []string{"Choice1", "Choice2", "Choice3"}
	rofiConfig := RofiUIConfig{Path: "rofi", SelectArgs: []string{"-lines", "3"}}

	t.Run("Success", func(t *testing.T) {
		mockExecCommand(t)
		os.Setenv("ROFI_SIMULATE", "SELECT_CHOICE_SUCCESS")
		defer os.Unsetenv("ROFI_SIMULATE")

		ui := NewRofiUI(rofiConfig)
		selected, err := ui.Select("Choose one:", choices)

		require.NoError(t, err)
		assert.Equal(t, "Choice2", selected) // TestHelperProcess simulates "Choice2"
	})

	t.Run("Cancellation", func(t *testing.T) {
		mockExecCommand(t)
		os.Setenv("ROFI_SIMULATE", "SELECT_CHOICE_CANCEL")
		defer os.Unsetenv("ROFI_SIMULATE")

		ui := NewRofiUI(rofiConfig)
		_, err := ui.Select("Choose one:", choices)

		assert.ErrorIs(t, err, ErrUserAborted)
	})

	t.Run("Rofi Error", func(t *testing.T) {
		mockExecCommand(t)
		os.Setenv("ROFI_SIMULATE", "ROFI_ERROR")
		defer os.Unsetenv("ROFI_SIMULATE")

		ui := NewRofiUI(rofiConfig)
		_, err := ui.Select("Choose one:", choices)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "Rofi critical error")
	})
}

func TestRofiUI_Prompt(t *testing.T) {
	rofiConfig := RofiUIConfig{Path: "rofi", InputArgs: []string{"-font", "mono 12"}}

	t.Run("Success", func(t *testing.T) {
		mockExecCommand(t)
		os.Setenv("ROFI_SIMULATE", "PROMPT_SUCCESS")
		defer os.Unsetenv("ROFI_SIMULATE")

		ui := NewRofiUI(rofiConfig)
		input, err := ui.Prompt("Enter value:")

		require.NoError(t, err)
		assert.Equal(t, "User Input Text", input) // TestHelperProcess simulates this
	})

	t.Run("Cancellation", func(t *testing.T) {
		mockExecCommand(t)
		os.Setenv("ROFI_SIMULATE", "PROMPT_CANCEL")
		defer os.Unsetenv("ROFI_SIMULATE")

		ui := NewRofiUI(rofiConfig)
		_, err := ui.Prompt("Enter value:")

		assert.ErrorIs(t, err, ErrUserAborted)
	})

	t.Run("Rofi Error", func(t *testing.T) {
		mockExecCommand(t)
		os.Setenv("ROFI_SIMULATE", "ROFI_ERROR")
		defer os.Unsetenv("ROFI_SIMULATE")

		ui := NewRofiUI(rofiConfig)
		_, err := ui.Prompt("Enter value:")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "Rofi critical error")
	})
}

// Helper to compare arg slices, ignoring order of some Rofi flags if necessary
// For now, a simple reflect.DeepEqual should suffice if arg order is consistent.
func checkRofiArgs(t *testing.T, expectedArgs []string, actualArgs []string) {
	t.Helper() // Marks this function as a test helper
	// TODO: Implement more sophisticated arg checking if Rofi flag order is variable
	// For example, check presence of key flags like -dmenu, -p <prompt>, -theme <theme>
	// and then compare the remaining config.SelectArgs or config.InputArgs.
	// For now, we assume RofiUI builds them in a consistent order.
	if !reflect.DeepEqual(expectedArgs, actualArgs) {
		t.Errorf("Rofi arguments mismatch:\nExpected: %v\nActual:   %v", expectedArgs, actualArgs)
	}
}
