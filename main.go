// Package main implements the command-line interface for ezbp.
// It uses the cobra library to define commands and flags.
package main

import (
	"fmt"
	"os"

	"github.com/atotto/clipboard"
	"github.com/spf13/cobra"
)

var (
	// rootCmd is the root command for ezbp.
	// It doesn't have a RunE function as it's a container for other commands.
	rootCmd = &cobra.Command{
		Use:   "ezbp",
		Short: "ezbp is a CLI tool for managing and using text boilerplates.",
	}
	// boilerplateCmd is the command for managing boilerplates.
	// It's a subcommand of rootCmd.
	boilerplateCmd = &cobra.Command{
		Use:   "boilerplate",
		Short: "Manage boilerplates.",
	}
	// boilerplateExpandCmd is the command for expanding a boilerplate.
	// It's a subcommand of boilerplateCmd.
	boilerplateExpandCmd = &cobra.Command{
		Use:   "expand",
		Short: "Expand a boilerplate.",
		// Args: cobra.PositionalArgs, // TODO: Add support for positional arguments to specify boilerplate name.
		RunE: func(cmd *cobra.Command, args []string) error {
			// Retrieve the --ui flag value
			uiFlagValue, err := cmd.Flags().GetString("ui")
			if err != nil {
				// This error typically means the flag wasn't defined,
				// which shouldn't happen if set up in init().
				return fmt.Errorf("internal error: could not retrieve UI flag: %w", err)
			}
			return boilerplateExpand(uiFlagValue) // Pass the flag value
		},
	}
)

func init() {
	// Add flags to boilerplateExpandCmd
	// The value of this flag will be read in the boilerplateExpand function.
	boilerplateExpandCmd.Flags().String("ui", "", "Specify UI: 'terminal' or 'rofi'. Overrides config.")
}

// TODO: Define more commands and flags based on these comments.
// - boilerplate
//   - expand <key> --clipboard --interactive --forever: Expand one boilerplate
//   - list: List boilerplates
// - remote:
//      - list:
//      - update:
//      - add/rm
// - wizard

// main is the entry point of the application.
// It sets up the cobra commands and executes the root command.
func main() {
	// Add boilerplateExpandCmd as a subcommand of boilerplateCmd.
	boilerplateCmd.AddCommand(boilerplateExpandCmd)

	// Add boilerplateCmd as a subcommand of rootCmd.
	rootCmd.AddCommand(boilerplateCmd)

	// Execute the root command.
	// If an error occurs, print it to stderr and exit with status 1.
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// boilerplateExpand handles the logic for the "boilerplate expand" command.
// It creates a new BoilerplateManager, prompts the user to select a boilerplate,
// expands the selected boilerplate, and copies the result to the clipboard.
// This function is designed to run in a loop, allowing the user to expand multiple boilerplates.
func boilerplateExpand(uiPreference string) error {
	// Create a new BoilerplateManager, passing the UI preference from the flag.
	bm, err := NewBoilerplateManager(uiPreference)
	if err != nil {
		return fmt.Errorf("failed to create boilerplate manager: %w", err)
	}

	// Loop indefinitely to allow expanding multiple boilerplates.
	for {
		// Prompt the user to select a boilerplate.
		name, err := bm.SelectBoilerplate()
		if err != nil {
			return fmt.Errorf("failed to select boilerplate: %w", err)
		}

		// Expand the selected boilerplate.
		value, err := bm.Expand(name)
		if err != nil {
			return fmt.Errorf("failed to expand boilerplate %q: %w", name, err)
		}

		// Copy the expanded boilerplate to the clipboard.
		if err := clipboard.WriteAll(value); err != nil {
			return fmt.Errorf("failed to copy to clipboard: %w", err)
		}
		// TODO: Add a confirmation message that the value was copied to the clipboard.
		// fmt.Printf("Boilerplate %q expanded and copied to clipboard.\n", name)
	}
}
