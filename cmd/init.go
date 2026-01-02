package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init <swagger.json>",
	Short: "Create a new project from a Swagger/OpenAPI spec",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		swaggerPath := args[0]
		fmt.Printf("Initializing project from: %s\n", swaggerPath)
		// TODO: Parse swagger and create project
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
}
