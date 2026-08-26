package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var updateProjectName string

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Refresh a project from its original OpenAPI JSON source",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return updateProjectFromSource(updateProjectName)
	},
}

func init() {
	updateCmd.Flags().StringVarP(&updateProjectName, "project", "p", "", "Project name to update")
	if err := updateCmd.MarkFlagRequired("project"); err != nil {
		panic(err)
	}
	rootCmd.AddCommand(updateCmd)
}

func updateProjectFromSource(name string) error {
	_, err := refreshProjectFromSource(name)
	if err != nil {
		return err
	}
	fmt.Printf("Updated project %q\n", name)
	return nil
}

func refreshProjectFromSource(name string) (Project, error) {
	if name == "" {
		return Project{}, fmt.Errorf("project name is required")
	}
	directory, err := projectDirectory(name)
	if err != nil {
		return Project{}, err
	}
	projectPath := filepath.Join(directory, "project.json")
	data, err := os.ReadFile(projectPath)
	if err != nil {
		return Project{}, fmt.Errorf("read project: %w", err)
	}
	var existing Project
	if err := json.Unmarshal(data, &existing); err != nil {
		return Project{}, fmt.Errorf("parse project: %w", err)
	}
	if existing.Source == "" {
		return Project{}, fmt.Errorf("project %q has no original source; reinitialize it", name)
	}
	specData, err := readSpecSource(existing.Source)
	if err != nil {
		return Project{}, err
	}
	updated, err := parseProject(specData, existing.Name, existing.BaseURL, existing.Source)
	if err != nil {
		return Project{}, err
	}
	if err := writeProject(updated); err != nil {
		return Project{}, err
	}
	return updated, nil
}
