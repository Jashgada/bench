package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var searchCmd = &cobra.Command{
	Use:   "search",
	Short: "Open fuzzy finder to search and select an API",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Opening search...")
		// TODO: Launch bubbletea fuzzy finder
	},
}

func init() {
	rootCmd.AddCommand(searchCmd)
}
