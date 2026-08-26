package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "bench",
	Short: "curl lets you lift. bench lets you lift heavier.",
	Long:  `bench is a fast CLI tool to supercharge curl with Swagger-powered API collections.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runTUI("")
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
