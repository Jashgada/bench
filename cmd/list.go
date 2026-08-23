package cmd

import (
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all APIs in the current project",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		project, err := loadProject(listProjectName)
		if err != nil {
			return err
		}
		fmt.Fprintf(cmd.OutOrStdout(), "Project: %s\n", project.Name)
		apis := append([]API(nil), project.APIs...)
		sort.Slice(apis, func(i, j int) bool { return apis[i].Name < apis[j].Name })
		for _, api := range apis {
			if listFilter != "" && !strings.Contains(strings.ToLower(api.Name+" "+api.Method+" "+api.Path), strings.ToLower(listFilter)) {
				continue
			}
			fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s\t%s\n", api.Method, api.Path, api.Name)
		}
		return nil
	},
}

var listProjectName string
var listFilter string

func init() {
	listCmd.Flags().StringVarP(&listProjectName, "project", "p", "", "Project name to list")
	listCmd.Flags().StringVarP(&listFilter, "filter", "f", "", "Text to search in method, path, or operation name")
	rootCmd.AddCommand(listCmd)
}
