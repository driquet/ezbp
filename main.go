// Package main implements the command-line interface for ezbp.
// It uses the cobra library to define commands and flags.
package main

import (
	"fmt"
	"os"

	"github.com/atotto/clipboard"
	"github.com/driquet/ezbp/internal/database"
	"github.com/driquet/ezbp/internal/editor"
	"github.com/driquet/ezbp/internal/engine"
	"github.com/spf13/cobra"
)

var (
	ui         string
	forever    bool
	config     engine.Config
	configPath string
	db         database.Database
	bm         *engine.Engine
)

func setupRuntime(cmd *cobra.Command, args []string) error {
	var err error

	// Possible custom config path
	if configPath == "" {
		configPath, err = engine.ConfigDirPath()
		if err != nil {
			return err
		}
	}

	// Load configuration
	config, err = engine.LoadConfigFromFile(configPath)
	if err != nil {
		return err
	}

	// Override values with flags
	if ui != "" {
		config.DefaultUI = ui
	}

	// Load database
	db, err = database.NewSQLiteDatabase(config.DatabasePath)
	if err != nil {
		return err
	}

	// Create a new BoilerplateManager
	bm, err = engine.NewEngine(db, config)
	if err != nil {
		return fmt.Errorf("failed to create engine: %w", err)
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
				value, err := editor.Edit(config.Editor, "")
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
			bp, found := bm.Get(args[0])
			if !found {
				return fmt.Errorf("unknown boilerplate %q", args[0])
			}

			if len(args) == 1 {
				content := bp.Value
				value, err := editor.Edit(config.Editor, content)
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
			if _, found := bm.Get(args[0]); !found {
				return fmt.Errorf("unknown boilerplate %q", args[0])
			}

			return bm.Delete(args[0])
		},
	}
	boilerplateExpandCmd = &cobra.Command{
		Use:      "expand",
		Short:    "Expand a boilerplate.",
		Args:     cobra.RangeArgs(0, 1),
		PreRunE:  setupRuntime,
		PostRunE: tearDownRuntime,
		RunE: func(cmd *cobra.Command, args []string) error {
			return boilerplateExpand(args) // Pass the flag value
		},
	}
	boilerplateImportCmd = &cobra.Command{
		Use:   "import <file.csv>",
		Short: "Import boilerplates from a CSV file",
		Long: `Import boilerplate templates from a CSV file.

The CSV file must contain a header row with the fields "name" and "value".
Each subsequent row should represent a boilerplate name and its content.

If a boilerplate with the same name already exists, you will be prompted to
choose whether to keep the existing value, update it, or apply the choice for all.`,
		Example: `  # Import boilerplates from a CSV file
  ezbp boilerplate import templates.csv`,
		Args:     cobra.ExactArgs(1),
		PreRunE:  setupRuntime,
		PostRunE: tearDownRuntime,
		RunE: func(cmd *cobra.Command, args []string) error {
			return bm.ImportBoilerplatesFromCSV(args[0])
		},
	}
)

// TODO: Define more commands and flags based on these comments.
// - boilerplate
//   - list: List boilerplates
// - remote:
//      - list:
//      - update:
//      - add/rm

func main() {
	// Flags
	rootCmd.PersistentFlags().StringVar(&configPath, "config", "", "Overrides default configuration path.")

	boilerplateExpandCmd.Flags().BoolVarP(&forever, "forever", "f", false, "Continuously expand boilerplates.")
	boilerplateExpandCmd.Flags().StringVar(&ui, "ui", "", "Specify UI: 'terminal' or 'rofi'. Overrides config.")

	boilerplateCmd.AddCommand(
		boilerplateAddCmd,
		boilerplateEditCmd,
		boilerplateDelCmd,
		boilerplateExpandCmd,
		boilerplateImportCmd,
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
func boilerplateExpand(args []string) error {
	if len(args) == 1 {
		// Expand the selected boilerplate.
		value, err := bm.Expand(args[0])
		if err != nil {
			return fmt.Errorf("failed to expand boilerplate %q: %w", args[0], err)
		}

		// Copy the expanded boilerplate to the clipboard.
		if err := clipboard.WriteAll(value); err != nil {
			return fmt.Errorf("failed to copy to clipboard: %w", err)
		}

		return nil
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

		if !forever {
			break
		}
	}

	return nil
}
