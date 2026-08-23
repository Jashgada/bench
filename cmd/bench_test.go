package cmd

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestParseProject(t *testing.T) {
	spec := []byte(`{"openapi":"3.0.0","servers":[{"url":"https://api.example.test"}],"paths":{"/pets/{id}":{"get":{"operationId":"getPet","parameters":[{"name":"id","in":"path","required":true},{"name":"limit","in":"query","required":false}]}}}}`)
	project, err := parseProject(spec, "pets", "", "/tmp/pets.json")
	if err != nil {
		t.Fatal(err)
	}
	if project.BaseURL != "https://api.example.test" || project.Source != "/tmp/pets.json" {
		t.Fatalf("unexpected project: %#v", project)
	}
	if len(project.APIs) != 1 || project.APIs[0].Name != "getPet" || len(project.APIs[0].PathParams) != 1 {
		t.Fatalf("unexpected APIs: %#v", project.APIs)
	}
}

func TestProjectRoundTrip(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	original := Project{Name: "pets", BaseURL: "https://example.test", Source: "/tmp/pets.json", APIs: []API{{Name: "listPets", Method: "GET", Path: "/pets"}}}
	if err := writeProject(original); err != nil {
		t.Fatal(err)
	}
	loaded, err := loadProject("pets")
	if err != nil {
		t.Fatal(err)
	}
	if !jsonEqual(original, loaded) {
		t.Fatalf("round trip mismatch: %#v != %#v", original, loaded)
	}
}

func TestUpdateProjectFromSource(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	specPath := filepath.Join(t.TempDir(), "pets.json")
	writeSpec := func(operation string) {
		spec := `{"openapi":"3.0.0","paths":{"/pets":{"get":{"operationId":"` + operation + `"}}}}`
		if err := os.WriteFile(specPath, []byte(spec), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	writeSpec("listPets")
	data, _ := os.ReadFile(specPath)
	project, err := parseProject(data, "pets", "", specPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := writeProject(project); err != nil {
		t.Fatal(err)
	}
	writeSpec("listAllPets")
	if err := updateProjectFromSource("pets"); err != nil {
		t.Fatal(err)
	}
	updated, err := loadProject("pets")
	if err != nil {
		t.Fatal(err)
	}
	if updated.APIs[0].Name != "listAllPets" {
		t.Fatalf("project was not refreshed: %#v", updated.APIs)
	}
}

func TestExecuteAPI(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/pets/one" || r.URL.Query().Get("verbose") != "true" {
			t.Errorf("unexpected URL: %s", r.URL.String())
		}
		if r.Method != http.MethodPost || r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("unexpected request: %s %s", r.Method, r.Header.Get("Content-Type"))
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("read body: %v", err)
		}
		if string(body) != `{"name":"Fido"}` {
			t.Errorf("unexpected body: %s", body)
		}
		w.Header().Set("X-Test", "ok")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	api := API{Name: "updatePet", Method: "POST", Path: "/pets/{id}", PathParams: []Parameter{{Name: "id", Required: true}}, QueryParams: []Parameter{{Name: "verbose", Required: true}}}
	cmd := &cobra.Command{}
	cmd.SetIn(strings.NewReader("one\ntrue\n"))
	var output bytes.Buffer
	cmd.SetOut(&output)
	runBody = `{"name":"Fido"}`
	runBodyFile = ""
	err := executeAPIWithBaseURL(cmd, Project{BaseURL: server.URL}, api)
	runBody = ""
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), "Status: 200 OK") || !strings.Contains(output.String(), `"ok": true`) {
		t.Fatalf("unexpected output: %s", output.String())
	}
}

func executeAPIWithBaseURL(cmd *cobra.Command, project Project, api API) error {
	return executeAPI(cmd, project, api)
}

func jsonEqual(a, b Project) bool {
	left, _ := json.Marshal(a)
	right, _ := json.Marshal(b)
	return bytes.Equal(left, right)
}
