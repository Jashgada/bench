package cmd

import (
	"github.com/spf13/cobra"
)

var searchProjectName string

var searchCmd = &cobra.Command{
	Use:   "search",
	Short: "Open the terminal API browser",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runTUI(searchProjectName)
	},
}

func init() {
	searchCmd.Flags().StringVarP(&searchProjectName, "project", "p", "", "Project name to browse")
	rootCmd.AddCommand(searchCmd)
}
