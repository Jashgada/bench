package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func loadProject(name string) (Project, error) {
	if name == "" {
		var err error
		name, err = currentProject()
		if err != nil {
			return Project{}, err
		}
	}
	directory, err := projectDirectory(name)
	if err != nil {
		return Project{}, err
	}
	data, err := os.ReadFile(filepath.Join(directory, "project.json"))
	if err != nil {
		return Project{}, fmt.Errorf("read project: %w", err)
	}
	var project Project
	if err := json.Unmarshal(data, &project); err != nil {
		return Project{}, fmt.Errorf("parse project: %w", err)
	}
	return project, nil
}

func setCurrentProject(name string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("find home directory: %w", err)
	}
	benchDir := filepath.Join(home, ".bench")
	if err := os.MkdirAll(benchDir, 0o755); err != nil {
		return fmt.Errorf("create bench directory: %w", err)
	}
	if err := os.WriteFile(filepath.Join(benchDir, "current"), []byte(name+"\n"), 0o644); err != nil {
		return fmt.Errorf("write current project: %w", err)
	}
	return nil
}

func currentProject() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("find home directory: %w", err)
	}
	data, err := os.ReadFile(filepath.Join(home, ".bench", "current"))
	if err != nil {
		return "", fmt.Errorf("no current project; run bench init or use --project: %w", err)
	}
	name := filepath.Clean(strings.TrimSpace(string(data)))
	if name == "." || name == "" {
		return "", fmt.Errorf("current project is empty")
	}
	return name, nil
}

func writeProject(project Project) error {
	directory, err := projectDirectory(project.Name)
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(project, "", "  ")
	if err != nil {
		return fmt.Errorf("encode project: %w", err)
	}
	if err := os.MkdirAll(directory, 0o755); err != nil {
		return fmt.Errorf("create project directory: %w", err)
	}
	if err := os.WriteFile(filepath.Join(directory, "project.json"), append(data, '\n'), 0o644); err != nil {
		return fmt.Errorf("write project: %w", err)
	}
	return nil
}
