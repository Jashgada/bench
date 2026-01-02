package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var runCmd = &cobra.Command{
	Use:   "run <api-name>",
	Short: "Execute an API (prompts for required params)",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		apiName := args[0]
		fmt.Printf("Running API: %s\n", apiName)
		// TODO: Load API, prompt for params, execute
	},
}

func init() {
	rootCmd.AddCommand(runCmd)
}
