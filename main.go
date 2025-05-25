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
	ui         string
	config     Config
	configPath string
	db         Database
	bm         *BoilerplateManager
)

func setupRuntime(cmd *cobra.Command, args []string) error {
	var err error

	// Possible custom config path
	if configPath == "" {
		configPath, err = configDirPath()
		if err != nil {
			return err
		}
	}

	// Load configuration
	config, err = loadConfigFromFile(configPath)
	if err != nil {
		return err
	}

	// Override values with flags
	if ui != "" {
		config.DefaultUI = ui
	}

	// Load database
	db, err = NewSQLiteDatabase(config.DatabasePath)
	if err != nil {
		return err
	}

	// Create a new BoilerplateManager
	bm, err = NewBoilerplateManager(db, config)
	if err != nil {
		return fmt.Errorf("failed to create boilerplate manager: %w", err)
	}

	return nil
}

func tearDownRuntime(cmd *cobra.Command, args []string) error {
	if err := db.Close(); err != nil {
		return err
	}

	return nil
}

var (
	rootCmd = &cobra.Command{
		Use:   "ezbp",
		Short: "ezbp is a CLI tool for managing and using text boilerplates.",
	}
	boilerplateCmd = &cobra.Command{
		Use:   "boilerplate",
		Short: "Manage boilerplates.",
	}
	boilerplateAddCmd = &cobra.Command{
		Use:   "add <name> [content]",
		Short: "Add a new boilerplate template",
		Long: `Add a new boilerplate template with the specified name.

If only the name is provided, your default editor will open allowing you to
compose the boilerplate content interactively. The boilerplate will be saved
when you close the editor.

The editor used can be configured in your configuration file or will fall back
to the EDITOR environment variable. If neither is set, a system default editor
will be used.

If both name and content are provided, the boilerplate will be created
immediately with the specified content.`,
		Example: `  # Open editor to create a boilerplate interactively
  ezbp boilerplate add my-boilerplate-name

  # Create a boilerplate with inline content
  ezbp boilerplate add my-boilerplate-name "Hello World!"

  # The editor priority is: config file > EDITOR env var > system default
  # Set your preferred editor:
  export EDITOR=vim
  ezbp boilerplate add my-boilerplate-name`,
		Args:     cobra.RangeArgs(1, 2),
		PreRunE:  setupRuntime,
		PostRunE: tearDownRuntime,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 1 {
				value, err := editContent(config.Editor, "")
				if err != nil {
					return err
				}
				return bm.Add(args[0], value)
			}
			return bm.Add(args[0], args[1])
		},
	}
	boilerplateEditCmd = &cobra.Command{
		Use:   "edit <name> [content]",
		Short: "Edit a new boilerplate template",
		Long: `Edit an existing boilerplate template with the specified name.

If only the name is provided, your default editor will open allowing you to
compose the boilerplate content interactively. The boilerplate will be saved
when you close the editor.

The editor used can be configured in your configuration file or will fall back
to the EDITOR environment variable. If neither is set, a system default editor
will be used.

If both name and content are provided, the boilerplate will be edited
immediately with the specified content.`,
		Example: `  # Open editor to edit a boilerplate interactively
  ezbp boilerplate edit my-boilerplate-name

  # Edit a boilerplate with inline content
  ezbp boilerplate edit my-boilerplate-name "Hello World!"

  # The editor priority is: config file > EDITOR env var > system default
  # Set your preferred editor:
  export EDITOR=vim
  ezbp boilerplate edit my-boilerplate-name`,
		Args:     cobra.RangeArgs(1, 2),
		PreRunE:  setupRuntime,
		PostRunE: tearDownRuntime,
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			// Only provide completions for the first argument (name)
			if len(args) == 0 {
				// Return existing boilerplate names for reference/awareness
				// Note: These are existing names, user might want to create a new one
				names := bm.Names()
				return names, cobra.ShellCompDirectiveNoFileComp
			}
			// No completion for content argument
			return nil, cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if _, found := bm.boilerplates[args[0]]; !found {
				return fmt.Errorf("unknown boilerplate %q", args[0])
			}

			if len(args) == 1 {
				content := bm.boilerplates[args[0]].Value
				value, err := editContent(config.Editor, content)
				if err != nil {
					return err
				}
				return bm.Edit(args[0], value)
			}
			return bm.Edit(args[0], args[1])
		},
	}
	boilerplateDelCmd = &cobra.Command{
		Use:   "del <name>",
		Short: "Delete a boilerplate template",
		Long: `Delete an existing boilerplate template by name.

This will permanently remove the specified boilerplate from your collection.
Use with caution as this operation cannot be undone.`,
		Example: `  # Delete a boilerplate named 'my-function'
  myapp del my-function`,
		Args: cobra.ExactArgs(1),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			// Only provide completions for the first argument (name)
			if len(args) == 0 {
				// Return existing boilerplate names for deletion
				names := bm.Names()
				return names, cobra.ShellCompDirectiveNoFileComp
			}
			// No more arguments needed
			return nil, cobra.ShellCompDirectiveNoFileComp
		},
		PreRunE:  setupRuntime,
		PostRunE: tearDownRuntime,
		RunE: func(cmd *cobra.Command, args []string) error {
			if _, found := bm.boilerplates[args[0]]; !found {
				return fmt.Errorf("unknown boilerplate %q", args[0])
			}

			return bm.Delete(args[0])
		},
	}
	boilerplateExpandCmd = &cobra.Command{
		Use:   "expand",
		Short: "Expand a boilerplate.",
		// Args: cobra.PositionalArgs, // TODO: Add support for positional arguments to specify boilerplate name instead of interactive mode.
		PreRunE:  setupRuntime,
		PostRunE: tearDownRuntime,
		RunE: func(cmd *cobra.Command, args []string) error {
			return boilerplateExpand() // Pass the flag value
		},
	}
)

// TODO: Define more commands and flags based on these comments.
// - boilerplate
//   - expand <key> --clipboard --forever [name]
//   - list: List boilerplates
// - remote:
//      - list:
//      - update:
//      - add/rm

func main() {
	// Flags
	rootCmd.PersistentFlags().StringVar(&configPath, "config", "", "Overrides default configuration path.")
	boilerplateExpandCmd.Flags().StringVar(&ui, "ui", "", "Specify UI: 'terminal' or 'rofi'. Overrides config.")

	boilerplateCmd.AddCommand(
		boilerplateAddCmd,
		boilerplateEditCmd,
		boilerplateDelCmd,
		boilerplateExpandCmd,
	)

	rootCmd.AddCommand(boilerplateCmd)
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

// boilerplateExpand handles the logic for the "boilerplate expand" command.
// It creates a new BoilerplateManager, prompts the user to select a boilerplate,
// expands the selected boilerplate, and copies the result to the clipboard.
// This function is designed to run in a loop, allowing the user to expand multiple boilerplates.
func boilerplateExpand() error {
	// Loop indefinitely to allow expanding multiple boilerplates.
	// TODO: The loop mechanism should be decided based on the --forever flag
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
		// TODO: The clipboard feature should be enabled using the --clipboard flag (stdout otherwise).
		// TODO: Add a confirmation message that the value was copied to the clipboard.
		// fmt.Printf("Boilerplate %q expanded and copied to clipboard.\n", name)
		if err := clipboard.WriteAll(value); err != nil {
			return fmt.Errorf("failed to copy to clipboard: %w", err)
		}
	}
}
