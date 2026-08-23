package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var deleteProjectName string

var deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete the current project",
	Args:  cobra.ExactArgs(0),
	RunE: func(cmd *cobra.Command, args []string) error {
		return deleteProject(deleteProjectName)
	},
}

func init() {
	deleteCmd.Flags().StringVarP(&deleteProjectName, "project", "p", "", "Project name to delete")
	if err := deleteCmd.MarkFlagRequired("project"); err != nil {
		panic(err)
	}
	rootCmd.AddCommand(deleteCmd)
}

func deleteProject(projectName string) error {
	if projectName == "" {
		return fmt.Errorf("project name is required")
	}
	directory, err := projectDirectory(projectName)
	if err != nil {
		return err
	}
	fmt.Println("Deleting Project...", projectName)
	if err := os.RemoveAll(directory); err != nil {
		return fmt.Errorf("delete project: %w", err)
	}
	fmt.Println("Project deleted successfully")
	return nil
}
