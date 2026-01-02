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
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
