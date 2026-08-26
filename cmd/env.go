package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

type Environment struct {
	Name   string            `json:"name"`
	Values map[string]string `json:"values"`
}

func environmentsDirectory(project string) (string, error) {
	dir, err := projectDirectory(project)
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "environments"), nil
}

func listEnvironments(project string) ([]string, error) {
	dir, err := environmentsDirectory(project)
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, fmt.Errorf("read environments: %w", err)
	}
	var names []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".json") {
			names = append(names, strings.TrimSuffix(entry.Name(), ".json"))
		}
	}
	sort.Strings(names)
	return names, nil
}

func saveEnvironment(project string, env Environment) error {
	if strings.TrimSpace(env.Name) == "" {
		return fmt.Errorf("environment name is required")
	}
	if env.Values == nil {
		env.Values = map[string]string{}
	}
	dir, err := environmentsDirectory(project)
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(env, "", "  ")
	if err != nil {
		return fmt.Errorf("encode environment: %w", err)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create environments directory: %w", err)
	}
	path := filepath.Join(dir, env.Name+".json")
	if err := os.WriteFile(path, append(data, '\n'), 0o600); err != nil {
		return fmt.Errorf("write environment: %w", err)
	}
	return nil
}

func loadEnvironment(project, name string) (Environment, error) {
	dir, err := environmentsDirectory(project)
	if err != nil {
		return Environment{}, err
	}
	data, err := os.ReadFile(filepath.Join(dir, name+".json"))
	if err != nil {
		return Environment{}, fmt.Errorf("read environment: %w", err)
	}
	var env Environment
	if err := json.Unmarshal(data, &env); err != nil {
		return Environment{}, fmt.Errorf("parse environment: %w", err)
	}
	return env, nil
}

func deleteEnvironment(project, name string) error {
	dir, err := environmentsDirectory(project)
	if err != nil {
		return err
	}
	if err := os.Remove(filepath.Join(dir, name+".json")); err != nil {
		return fmt.Errorf("delete environment: %w", err)
	}
	if currentEnvironmentName(project) == name {
		_ = os.Remove(filepath.Join(mustProjectDirectory(project), "current_env"))
	}
	return nil
}

func setCurrentEnvironment(project, name string) error {
	data, err := json.MarshalIndent(Environment{Name: name}, "", "  ")
	if err != nil {
		return fmt.Errorf("encode environment pointer: %w", err)
	}
	path := filepath.Join(mustProjectDirectory(project), "current_env")
	if err := os.WriteFile(path, append(data, '\n'), 0o600); err != nil {
		return fmt.Errorf("write current environment: %w", err)
	}
	return nil
}

func currentEnvironmentName(project string) string {
	data, err := os.ReadFile(filepath.Join(mustProjectDirectory(project), "current_env"))
	if err != nil {
		return ""
	}
	var pointer Environment
	if json.Unmarshal(data, &pointer) != nil {
		return ""
	}
	return pointer.Name
}

func mustProjectDirectory(project string) string {
	dir, _ := projectDirectory(project)
	return dir
}

// resolveEnvironmentValues expands $VAR indirections. Values beginning with $
// read from process environment variables so secrets never live on disk.
func resolveEnvironmentValues(env Environment) (map[string]string, []string) {
	values := make(map[string]string, len(env.Values))
	var missing []string
	keys := make([]string, 0, len(env.Values))
	for key := range env.Values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		value := env.Values[key]
		if strings.HasPrefix(value, "$") && len(value) > 1 {
			resolved := os.Getenv(value[1:])
			if resolved == "" {
				missing = append(missing, value[1:])
				continue
			}
			value = resolved
		}
		values[key] = value
	}
	return values, missing
}

var variablePattern = regexp.MustCompile(`\{\{\s*([A-Za-z0-9_.-]+)\s*\}\}`)

// substituteVars replaces {{key}} references with environment values.
// Unknown variables are left untouched so mistakes stay visible.
func substituteVars(input string, values map[string]string) string {
	if len(values) == 0 || !strings.Contains(input, "{{") {
		return input
	}
	return variablePattern.ReplaceAllStringFunc(input, func(match string) string {
		submatches := variablePattern.FindStringSubmatch(match)
		if len(submatches) < 2 {
			return match
		}
		if value, ok := values[submatches[1]]; ok {
			return value
		}
		return match
	})
}

// activeEnvironmentValues loads the project's selected environment and
// resolves its values. A missing environment is not an error; requests run
// un-substituted and auth simply does not apply.
func activeEnvironmentValues(project string) (map[string]string, string, []string) {
	name := currentEnvironmentName(project)
	if name == "" {
		return map[string]string{}, "", nil
	}
	env, err := loadEnvironment(project, name)
	if err != nil {
		return map[string]string{}, name, nil
	}
	values, missing := resolveEnvironmentValues(env)
	return values, name, missing
}
