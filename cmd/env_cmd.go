package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

var (
	envProjectName string
	envSetFlags    []string
)

func resolveEnvProject() (string, error) {
	if envProjectName != "" {
		return envProjectName, nil
	}
	return currentProject()
}

var envCmd = &cobra.Command{
	Use:   "env",
	Short: "Manage local environments for a project",
}

var envListCmd = &cobra.Command{
	Use:   "list",
	Short: "List environments",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		project, err := resolveEnvProject()
		if err != nil {
			return err
		}
		names, err := listEnvironments(project)
		if err != nil {
			return err
		}
		active := currentEnvironmentName(project)
		out := cmd.OutOrStdout()
		if len(names) == 0 {
			fmt.Fprintln(out, "No environments.")
			return nil
		}
		for _, name := range names {
			suffix := ""
			if name == active {
				suffix = " (active)"
			}
			fmt.Fprintln(out, name+suffix)
		}
		return nil
	},
}

var envAddCmd = &cobra.Command{
	Use:   "add <name>",
	Short: "Create or update an environment (--set key=value, repeatable)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		project, err := resolveEnvProject()
		if err != nil {
			return err
		}
		values := map[string]string{}
		existing, err := loadEnvironment(project, args[0])
		if err == nil && existing.Values != nil {
			values = existing.Values
		}
		for _, pair := range envSetFlags {
			key, value, found := strings.Cut(pair, "=")
			if !found {
				return fmt.Errorf("invalid --set %q: expected key=value", pair)
			}
			values[strings.TrimSpace(key)] = value
		}
		if err := saveEnvironment(project, Environment{Name: args[0], Values: values}); err != nil {
			return err
		}
		fmt.Fprintf(cmd.OutOrStdout(), "Saved environment %q for project %q\n", args[0], project)
		return nil
	},
}

var envUseCmd = &cobra.Command{
	Use:   "use <name>",
	Short: "Set the active environment",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		project, err := resolveEnvProject()
		if err != nil {
			return err
		}
		if _, err := loadEnvironment(project, args[0]); err != nil {
			return err
		}
		if err := setCurrentEnvironment(project, args[0]); err != nil {
			return err
		}
		fmt.Fprintf(cmd.OutOrStdout(), "Active environment for %q is now %q\n", project, args[0])
		return nil
	},
}

var envRemoveCmd = &cobra.Command{
	Use:   "rm <name>",
	Short: "Delete an environment",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		project, err := resolveEnvProject()
		if err != nil {
			return err
		}
		if err := deleteEnvironment(project, args[0]); err != nil {
			return err
		}
		fmt.Fprintf(cmd.OutOrStdout(), "Deleted environment %q\n", args[0])
		return nil
	},
}

func init() {
	for _, sub := range []*cobra.Command{envListCmd, envAddCmd, envUseCmd, envRemoveCmd} {
		sub.Flags().StringVarP(&envProjectName, "project", "p", "", "Project name")
	}
	envAddCmd.Flags().StringArrayVar(&envSetFlags, "set", nil, "Set a variable as key=value (repeatable)")
	envCmd.AddCommand(envListCmd, envAddCmd, envUseCmd, envRemoveCmd)
	rootCmd.AddCommand(envCmd)
}
