package cmd

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestEnvironmentRoundTrip(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	env := Environment{Name: "dev", Values: map[string]string{"host": "http://localhost:8080"}}
	if err := saveEnvironment("pets", env); err != nil {
		t.Fatal(err)
	}
	names, err := listEnvironments("pets")
	if err != nil {
		t.Fatal(err)
	}
	if len(names) != 1 || names[0] != "dev" {
		t.Fatalf("unexpected environments: %#v", names)
	}
	loaded, err := loadEnvironment("pets", "dev")
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Values["host"] != "http://localhost:8080" {
		t.Fatalf("unexpected values: %#v", loaded.Values)
	}
	if err := setCurrentEnvironment("pets", "dev"); err != nil {
		t.Fatal(err)
	}
	if currentEnvironmentName("pets") != "dev" {
		t.Fatal("current environment not persisted")
	}
	if err := deleteEnvironment("pets", "dev"); err != nil {
		t.Fatal(err)
	}
	if currentEnvironmentName("pets") != "" {
		t.Fatal("current environment pointer not cleared")
	}
	names, _ = listEnvironments("pets")
	if len(names) != 0 {
		t.Fatalf("environment not deleted: %#v", names)
	}
}

func TestSubstituteVars(t *testing.T) {
	values := map[string]string{"host": "example.com", "port": "8080"}
	cases := []struct{ input, want string }{
		{"{{host}}", "example.com"},
		{"http://{{ host }}:{{port}}/x", "http://example.com:8080/x"},
		{"{{unknown}}", "{{unknown}}"},
		{"no vars", "no vars"},
	}
	for _, tc := range cases {
		if got := substituteVars(tc.input, values); got != tc.want {
			t.Errorf("substituteVars(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestResolveEnvironmentIndirection(t *testing.T) {
	t.Setenv("BENCH_TEST_TOKEN", "secret")
	env := Environment{Name: "dev", Values: map[string]string{
		"token":  "$BENCH_TEST_TOKEN",
		"broken": "$BENCH_TEST_MISSING_VAR",
		"plain":  "literal",
	}}
	values, missing := resolveEnvironmentValues(env)
	if values["token"] != "secret" {
		t.Fatalf("indirection not resolved: %#v", values)
	}
	if values["plain"] != "literal" {
		t.Fatalf("literal changed: %#v", values)
	}
	if len(missing) != 1 || missing[0] != "BENCH_TEST_MISSING_VAR" {
		t.Fatalf("unexpected missing: %#v", missing)
	}
}

func TestParseCurl(t *testing.T) {
	cases := []struct {
		name    string
		input   string
		want    curlRequest
		wantErr bool
	}{
		{
			name:  "simple get",
			input: "curl https://api.example.com/pets",
			want:  curlRequest{Method: "GET", URL: "https://api.example.com/pets"},
		},
		{
			name:  "post with data and header",
			input: `curl -X POST 'https://api.example.com/pets' -H 'X-Tag: best friend' -d '{"name":"Fido"}'`,
			want: curlRequest{
				Method:  "POST",
				URL:     "https://api.example.com/pets",
				Headers: [][2]string{{"X-Tag", "best friend"}},
				Body:    `{"name":"Fido"}`,
			},
		},
		{
			name:  "data implies post",
			input: `curl -d '{"a":1}' https://api.example.com`,
			want:  curlRequest{Method: "POST", URL: "https://api.example.com", Body: `{"a":1}`},
		},
		{
			name:  "double quoted body with escaped quotes",
			input: `curl -d "{\"a\":\"it's\"}" https://api.example.com`,
			want:  curlRequest{Method: "POST", URL: "https://api.example.com", Body: `{"a":"it's"}`},
		},
		{
			name:  "ignores unknown flags",
			input: "curl -sSL https://api.example.com",
			want:  curlRequest{Method: "GET", URL: "https://api.example.com"},
		},
		{
			name:    "no url",
			input:   "curl -X GET",
			wantErr: true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parseCurl(tc.input)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error, got %#v", got)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if got.Method != tc.want.Method || got.URL != tc.want.URL || got.Body != tc.want.Body {
				t.Fatalf("got %#v, want %#v", got, tc.want)
			}
			if len(got.Headers) != len(tc.want.Headers) {
				t.Fatalf("headers: got %#v, want %#v", got.Headers, tc.want.Headers)
			}
			for i, header := range tc.want.Headers {
				if got.Headers[i] != header {
					t.Fatalf("header %d: got %#v, want %#v", i, got.Headers[i], header)
				}
			}
		})
	}
}

func TestExecuteRequestSubstitutionAndAuth(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("BENCH_SECRET", "top-secret")

	var gotAuth, gotKey, gotQuery, gotBody string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotKey = r.Header.Get("X-API-Key")
		gotQuery = r.URL.Query().Get("q")
		body := make([]byte, r.ContentLength)
		_, _ = r.Body.Read(body)
		gotBody = string(body)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	project := Project{
		Name:    "authproj",
		BaseURL: "{{host}}",
		SecuritySchemes: map[string]securityScheme{
			"bearer": {Type: "http", Scheme: "bearer"},
			"key":    {Type: "apiKey", Name: "X-API-Key", In: "header"},
		},
	}
	if err := writeProject(project); err != nil {
		t.Fatal(err)
	}
	if err := saveEnvironment("authproj", Environment{Name: "dev", Values: map[string]string{
		"host":  server.URL,
		"token": "$BENCH_SECRET",
	}}); err != nil {
		t.Fatal(err)
	}
	if err := setCurrentEnvironment("authproj", "dev"); err != nil {
		t.Fatal(err)
	}

	api := API{
		Name:        "search",
		Method:      "POST",
		Path:        "/search",
		QueryParams: []Parameter{{Name: "q"}},
		Security:    []string{"bearer"},
	}

	result := executeRequest(project, api, map[string]string{"q": "{{host}}"}, []byte(`{"token":"{{token}}"}`))
	if result.Error != nil {
		t.Fatal(result.Error)
	}
	if gotAuth != "Bearer top-secret" {
		t.Fatalf("bearer not applied: %q", gotAuth)
	}
	if gotQuery != server.URL {
		t.Fatalf("query substitution failed: %q", gotQuery)
	}
	if gotBody != `{"token":"top-secret"}` {
		t.Fatalf("body substitution failed: %q", gotBody)
	}

	api.Security = []string{"key"}
	result = executeRequest(project, api, map[string]string{"q": "x"}, nil)
	if result.Error != nil {
		t.Fatal(result.Error)
	}
	if gotKey != "" {
		t.Fatalf("unexpected api key: %q", gotKey)
	}
	if len(result.Warnings) == 0 {
		t.Fatal("expected warning for missing credential")
	}

	if err := saveEnvironment("authproj", Environment{Name: "dev", Values: map[string]string{
		"host": server.URL, "token": "$BENCH_SECRET", "key": "abc123",
	}}); err != nil {
		t.Fatal(err)
	}
	result = executeRequest(project, api, map[string]string{"q": "x"}, nil)
	if result.Error != nil {
		t.Fatal(result.Error)
	}
	if gotKey != "abc123" {
		t.Fatalf("api key not applied: %q", gotKey)
	}
}

func TestBasicAuthInjection(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	var gotAuth string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	project := Project{
		Name:    "basicproj",
		BaseURL: server.URL,
		SecuritySchemes: map[string]securityScheme{
			"basic": {Type: "http", Scheme: "basic"},
		},
	}
	if err := writeProject(project); err != nil {
		t.Fatal(err)
	}
	if err := saveEnvironment("basicproj", Environment{Name: "dev", Values: map[string]string{
		"basic_username": "alice",
		"basic_password": "wonderland",
	}}); err != nil {
		t.Fatal(err)
	}
	if err := setCurrentEnvironment("basicproj", "dev"); err != nil {
		t.Fatal(err)
	}
	api := API{Name: "me", Method: "GET", Path: "/me", Security: []string{"basic"}}
	result := executeRequest(project, api, nil, nil)
	if result.Error != nil {
		t.Fatal(result.Error)
	}
	want := "Basic " + base64.StdEncoding.EncodeToString([]byte("alice:wonderland"))
	if gotAuth != want {
		t.Fatalf("basic auth not applied: %q, want %q", gotAuth, want)
	}
}

func TestParseProjectSecurity(t *testing.T) {
	spec := []byte(`{
		"openapi": "3.0.0",
		"servers": [{"url": "https://api.example.test"}],
		"components": {
			"securitySchemes": {
				"bearerAuth": {"type": "http", "scheme": "bearer"},
				"keyAuth": {"type": "apiKey", "name": "X-Key", "in": "header"}
			}
		},
		"paths": {
			"/secure": {
				"get": {
					"operationId": "secureOp",
					"security": [{"bearerAuth": []}]
				}
			}
		}
	}`)
	project, err := parseProject(spec, "sec", "", "/tmp/sec.json")
	if err != nil {
		t.Fatal(err)
	}
	if len(project.SecuritySchemes) != 2 {
		t.Fatalf("schemes not parsed: %#v", project.SecuritySchemes)
	}
	if project.SecuritySchemes["keyAuth"].Name != "X-Key" {
		t.Fatalf("scheme details not parsed: %#v", project.SecuritySchemes["keyAuth"])
	}
	if len(project.APIs) != 1 || len(project.APIs[0].Security) != 1 || project.APIs[0].Security[0] != "bearerAuth" {
		t.Fatalf("security requirement not parsed: %#v", project.APIs)
	}
	data, err := json.Marshal(project)
	if err != nil {
		t.Fatal(err)
	}
	var roundTrip Project
	if err := json.Unmarshal(data, &roundTrip); err != nil {
		t.Fatal(err)
	}
	if roundTrip.SecuritySchemes["bearerAuth"].Scheme != "bearer" {
		t.Fatalf("schemes lost in round trip: %#v", roundTrip.SecuritySchemes)
	}
}
