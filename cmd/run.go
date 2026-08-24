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

type ResponseResult struct {
	Status  string
	Timing  time.Duration
	Headers http.Header
	Body    []byte
	Error   error
}

func executeRequest(project Project, api API, params map[string]string, body []byte) ResponseResult {
	path := api.Path
	for _, p := range api.PathParams {
		if val, ok := params[p.Name]; ok && val != "" {
			path = strings.ReplaceAll(path, "{"+p.Name+"}", url.PathEscape(val))
		}
	}
	requestURL := strings.TrimRight(project.BaseURL, "/") + "/" + strings.TrimLeft(path, "/")
	parsed, err := url.Parse(requestURL)
	if err != nil {
		return ResponseResult{Error: fmt.Errorf("build request URL: %w", err)}
	}
	query := parsed.Query()
	for _, p := range api.QueryParams {
		if val, ok := params[p.Name]; ok && val != "" {
			query.Set(p.Name, val)
		}
	}
	parsed.RawQuery = query.Encode()

	var bodyReader io.Reader
	if len(body) > 0 {
		bodyReader = bytes.NewReader(body)
	}
	request, err := http.NewRequest(api.Method, parsed.String(), bodyReader)
	if err != nil {
		return ResponseResult{Error: fmt.Errorf("create request: %w", err)}
	}
	for _, h := range api.Headers {
		if val, ok := params[h.Name]; ok && val != "" {
			request.Header.Set(h.Name, val)
		}
	}
	if len(body) > 0 && request.Header.Get("Content-Type") == "" {
		request.Header.Set("Content-Type", "application/json")
	}

	start := time.Now()
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return ResponseResult{Error: fmt.Errorf("execute request: %w", err)}
	}
	defer response.Body.Close()
	responseData, err := io.ReadAll(response.Body)
	if err != nil {
		return ResponseResult{Error: fmt.Errorf("read response: %w", err)}
	}
	return ResponseResult{
		Status:  response.Status,
		Timing:  time.Since(start).Round(time.Millisecond),
		Headers: response.Header,
		Body:    responseData,
	}
}

func executeAPI(cmd *cobra.Command, project Project, api API) error {
	reader := bufio.NewReader(cmd.InOrStdin())
	params := map[string]string{}
	for _, p := range api.PathParams {
		if p.Required {
			val, err := promptValue(cmd, reader, p.Name)
			if err != nil {
				return err
			}
			params[p.Name] = val
		}
	}
	for _, p := range api.QueryParams {
		if p.Required {
			val, err := promptValue(cmd, reader, p.Name)
			if err != nil {
				return err
			}
			params[p.Name] = val
		}
	}
	for _, h := range api.Headers {
		if h.Required {
			val, err := promptValue(cmd, reader, h.Name)
			if err != nil {
				return err
			}
			params[h.Name] = val
		}
	}
	bodyData, err := readRequestBody(cmd, reader, len(api.RequestBodySchema) > 0, api.RequestBodyRequired)
	if err != nil {
		return err
	}

	result := executeRequest(project, api, params, bodyData)
	if result.Error != nil {
		return result.Error
	}

	out := cmd.OutOrStdout()
	fmt.Fprintf(out, "Status: %s\nTiming: %s\nHeaders:\n", result.Status, result.Timing)
	for key, values := range result.Headers {
		for _, value := range values {
			fmt.Fprintf(out, "%s: %s\n", key, value)
		}
	}
	fmt.Fprintln(out, "\nResponse:")
	if json.Valid(result.Body) {
		var pretty bytes.Buffer
		if json.Indent(&pretty, result.Body, "", "  ") == nil {
			fmt.Fprintln(out, pretty.String())
			return nil
		}
	}
	_, err = out.Write(result.Body)
	if len(result.Body) > 0 && result.Body[len(result.Body)-1] != '\n' {
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

func readRequestBody(cmd *cobra.Command, reader *bufio.Reader, hasSchema, required bool) ([]byte, error) {
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
	if hasSchema {
		fmt.Fprint(cmd.OutOrStdout(), "Body JSON (optional): ")
		value, err := reader.ReadString('\n')
		if err != nil && err != io.EOF {
			return nil, fmt.Errorf("read request body: %w", err)
		}
		value = strings.TrimSpace(value)
		if value == "" && required {
			return nil, fmt.Errorf("request body is required")
		}
		return []byte(value), nil
	}
	return nil, nil
}
