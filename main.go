package main

import (
	"fmt"
	"os"

	"github.com/atotto/clipboard"
	"github.com/spf13/cobra"
)

var (
	rootCmd = &cobra.Command{
		Use: "ezbp",
	}
	boilerplateCmd = &cobra.Command{
		Use: "boilerplate",
	}
	boilerplateExpandCmd = &cobra.Command{
		Use: "expand",
		// Args: cobra.PositionalArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return boilerplateExpand()
		},
	}
)

// - boilerplate
//   - expand <key> --clipboard --interactive --forever: Expand one boilerplate
//   - list: List boilerplates
// - remote:
//      - list:
//      - update:
//      - add/rm
// - wizard

func main() {
	boilerplateCmd.AddCommand(boilerplateExpandCmd)

	rootCmd.AddCommand(boilerplateCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func boilerplateExpand() error {
	bm, err := NewBoilerplateManager()
	if err != nil {
		return err
	}

	for {
		name, err := bm.SelectBoilerplate()
		if err != nil {
			return err
		}

		value, err := bm.Expand(name)
		if err != nil {
			return err
		}

		if err := clipboard.WriteAll(value); err != nil {
			return err
		}
	}
}
