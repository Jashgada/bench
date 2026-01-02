package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all APIs in the current project",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Listing APIs...")
		// TODO: Load project and list APIs
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
}
