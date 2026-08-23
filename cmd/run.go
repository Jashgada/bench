package cmd

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

var runCmd = &cobra.Command{
	Use:   "run <operation-id>",
	Short: "Execute an API (prompts for required params)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		project, err := loadProject(runProjectName)
		if err != nil {
			return err
		}
		var api *API
		for i := range project.APIs {
			if project.APIs[i].Name == args[0] {
				api = &project.APIs[i]
				break
			}
		}
		if api == nil {
			return fmt.Errorf("operation %q not found", args[0])
		}
		return executeAPI(cmd, project, *api)
	},
}

var runProjectName string
var runBody string
var runBodyFile string

func init() {
	runCmd.Flags().StringVarP(&runProjectName, "project", "p", "", "Project name to use")
	runCmd.Flags().StringVar(&runBody, "body", "", "Request body JSON")
	runCmd.Flags().StringVar(&runBodyFile, "body-file", "", "Read request body from a file")
	rootCmd.AddCommand(runCmd)
}

func executeAPI(cmd *cobra.Command, project Project, api API) error {
	reader := bufio.NewReader(cmd.InOrStdin())
	path := api.Path
	for _, parameter := range api.PathParams {
		if parameter.Required {
			value, err := promptValue(cmd, reader, parameter.Name)
			if err != nil {
				return err
			}
			path = strings.ReplaceAll(path, "{"+parameter.Name+"}", url.PathEscape(value))
		}
	}
	requestURL := strings.TrimRight(project.BaseURL, "/") + "/" + strings.TrimLeft(path, "/")
	parsed, err := url.Parse(requestURL)
	if err != nil {
		return fmt.Errorf("build request URL: %w", err)
	}
	query := parsed.Query()
	for _, parameter := range api.QueryParams {
		if parameter.Required {
			value, err := promptValue(cmd, reader, parameter.Name)
			if err != nil {
				return err
			}
			query.Set(parameter.Name, value)
		}
	}
	parsed.RawQuery = query.Encode()
	var body io.Reader
	bodyData, err := readRequestBody(cmd, reader)
	if err != nil {
		return err
	}
	if len(bodyData) > 0 {
		body = bytes.NewReader(bodyData)
	}
	request, err := http.NewRequest(api.Method, parsed.String(), body)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	for _, header := range api.Headers {
		if header.Required {
			value, err := promptValue(cmd, reader, header.Name)
			if err != nil {
				return err
			}
			request.Header.Set(header.Name, value)
		}
	}
	if len(bodyData) > 0 && request.Header.Get("Content-Type") == "" {
		request.Header.Set("Content-Type", "application/json")
	}
	start := time.Now()
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return fmt.Errorf("execute request: %w", err)
	}
	defer response.Body.Close()
	responseData, err := io.ReadAll(response.Body)
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}
	out := cmd.OutOrStdout()
	fmt.Fprintf(out, "Status: %s\nTiming: %s\nHeaders:\n", response.Status, time.Since(start).Round(time.Millisecond))
	for key, values := range response.Header {
		for _, value := range values {
			fmt.Fprintf(out, "%s: %s\n", key, value)
		}
	}
	fmt.Fprintln(out, "\nResponse:")
	if json.Valid(responseData) {
		var pretty bytes.Buffer
		if json.Indent(&pretty, responseData, "", "  ") == nil {
			fmt.Fprintln(out, pretty.String())
			return nil
		}
	}
	_, err = out.Write(responseData)
	if len(responseData) > 0 && responseData[len(responseData)-1] != '\n' {
		_, err = fmt.Fprintln(out)
	}
	return err
}

func promptValue(cmd *cobra.Command, reader *bufio.Reader, name string) (string, error) {
	fmt.Fprintf(cmd.OutOrStdout(), "%s: ", name)
	value, err := reader.ReadString('\n')
	if err != nil && err != io.EOF {
		return "", fmt.Errorf("read %s: %w", name, err)
	}
	value = strings.TrimSpace(value)
	if value == "" {
		return "", fmt.Errorf("%s is required", name)
	}
	return value, nil
}

func readRequestBody(cmd *cobra.Command, reader *bufio.Reader) ([]byte, error) {
	if runBody != "" && runBodyFile != "" {
		return nil, fmt.Errorf("use only one of --body or --body-file")
	}
	if runBodyFile != "" {
		data, err := os.ReadFile(runBodyFile)
		if err != nil {
			return nil, fmt.Errorf("read body file: %w", err)
		}
		return data, nil
	}
	if runBody != "" {
		return []byte(runBody), nil
	}
	if file, ok := cmd.InOrStdin().(*os.File); ok {
		if info, err := file.Stat(); err == nil && info.Mode()&os.ModeCharDevice == 0 {
			return io.ReadAll(reader)
		}
	}
	return nil, nil
}
