package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

// Project is the small, normalized representation used by bench after import.
// Commands do not need to know how the original OpenAPI document was shaped.
type Project struct {
	Name    string `json:"name"`
	BaseURL string `json:"base_url"`
	Source  string `json:"source"`
	APIs    []API  `json:"apis"`
}

type API struct {
	Name              string          `json:"name"`
	Method            string          `json:"method"`
	Path              string          `json:"path"`
	PathParams        []Parameter     `json:"path_params,omitempty"`
	QueryParams       []Parameter     `json:"query_params,omitempty"`
	Headers           []Parameter     `json:"headers,omitempty"`
	RequestBodySchema json.RawMessage `json:"request_body_schema,omitempty"`
}

type Parameter struct {
	Name     string `json:"name"`
	Required bool   `json:"required"`
}

// These types describe only the parts of OpenAPI 3 that init currently needs.
type openAPISpec struct {
	OpenAPI string `json:"openapi"`
	Info    struct {
		Title string `json:"title"`
	} `json:"info"`
	Servers []struct {
		URL string `json:"url"`
	} `json:"servers"`
	Paths map[string]map[string]json.RawMessage `json:"paths"`
}

type operation struct {
	OperationID string         `json:"operationId"`
	Summary     string         `json:"summary"`
	Parameters  []openAPIParam `json:"parameters"`
	RequestBody *requestBody   `json:"requestBody"`
}

type openAPIParam struct {
	Name     string `json:"name"`
	In       string `json:"in"`
	Required bool   `json:"required"`
}

type requestBody struct {
	Content map[string]struct {
		Schema json.RawMessage `json:"schema"`
	} `json:"content"`
}

var projectName string
var baseURL string

var initCmd = &cobra.Command{
	Use:   "init <swagger.json>",
	Short: "Create a new project from a Swagger/OpenAPI spec",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return initializeProject(args[0])
	},
}

func init() {
	initCmd.Flags().StringVar(&projectName, "name", "", "Project name")
	initCmd.Flags().StringVar(&baseURL, "base-url", "", "Override the server URL from the spec")
	if err := initCmd.MarkFlagRequired("name"); err != nil {
		panic(err)
	}
	rootCmd.AddCommand(initCmd)
}

func initializeProject(specPath string) error {
	data, err := readSpecSource(specPath)
	if err != nil {
		return err
	}

	name := projectName
	if name == "" {
		return fmt.Errorf("project name is required")
	}
	source := specPath
	if !strings.HasPrefix(source, "http://") && !strings.HasPrefix(source, "https://") {
		source, err = filepath.Abs(source)
		if err != nil {
			return fmt.Errorf("resolve spec path: %w", err)
		}
	}
	project, err := parseProject(data, name, baseURL, source)
	if err != nil {
		return err
	}

	projectDir, err := projectDirectory(name)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		return fmt.Errorf("create project directory: %w", err)
	}

	if err := writeProject(project); err != nil {
		return err
	}
	if err := setCurrentProject(name); err != nil {
		return err
	}

	fmt.Printf("Initialized project %q with %d APIs at %s\n", name, len(project.APIs), filepath.Join(projectDir, "project.json"))
	return nil
}

func parseProject(data []byte, name, overrideBaseURL, source string) (Project, error) {
	var spec openAPISpec
	if err := json.Unmarshal(data, &spec); err != nil {
		return Project{}, fmt.Errorf("parse OpenAPI JSON: %w", err)
	}
	if !strings.HasPrefix(spec.OpenAPI, "3.") {
		return Project{}, fmt.Errorf("unsupported spec: expected OpenAPI 3.x JSON")
	}
	serverURL := overrideBaseURL
	if serverURL == "" && len(spec.Servers) > 0 {
		serverURL = spec.Servers[0].URL
	}
	project := Project{Name: name, BaseURL: serverURL, Source: source}
	for path, pathItem := range spec.Paths {
		for method, rawOperation := range pathItem {
			if !isHTTPMethod(method) {
				continue
			}
			var op operation
			if err := json.Unmarshal(rawOperation, &op); err != nil {
				return Project{}, fmt.Errorf("parse %s %s: %w", method, path, err)
			}
			apiName := op.OperationID
			if apiName == "" {
				apiName = op.Summary
			}
			if apiName == "" {
				apiName = strings.ToUpper(method) + " " + path
			}
			api := API{Name: apiName, Method: strings.ToUpper(method), Path: path}
			for _, param := range op.Parameters {
				normalized := Parameter{Name: param.Name, Required: param.Required}
				switch param.In {
				case "path":
					api.PathParams = append(api.PathParams, normalized)
				case "query":
					api.QueryParams = append(api.QueryParams, normalized)
				case "header":
					api.Headers = append(api.Headers, normalized)
				}
			}
			if op.RequestBody != nil {
				for _, mediaType := range op.RequestBody.Content {
					api.RequestBodySchema = mediaType.Schema
					break
				}
			}
			project.APIs = append(project.APIs, api)
		}
	}
	return project, nil
}

func readSpecSource(source string) ([]byte, error) {
	if strings.HasPrefix(source, "http://") || strings.HasPrefix(source, "https://") {
		response, err := http.Get(source)
		if err != nil {
			return nil, fmt.Errorf("fetch spec: %w", err)
		}
		defer response.Body.Close()
		if response.StatusCode < 200 || response.StatusCode >= 300 {
			return nil, fmt.Errorf("fetch spec: unexpected HTTP status %s", response.Status)
		}
		data, err := io.ReadAll(response.Body)
		if err != nil {
			return nil, fmt.Errorf("read fetched spec: %w", err)
		}
		return data, nil
	}
	data, err := os.ReadFile(source)
	if err != nil {
		return nil, fmt.Errorf("read spec: %w", err)
	}
	return data, nil
}

func projectDirectory(name string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("find home directory: %w", err)
	}
	return filepath.Join(home, ".bench", "projects", name), nil
}

func isHTTPMethod(method string) bool {
	switch strings.ToUpper(method) {
	case "GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS", "TRACE":
		return true
	default:
		return false
	}
}
