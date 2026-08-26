package cmd

import "testing"

func TestCurlCommand(t *testing.T) {
	tests := []struct {
		name    string
		project Project
		api     API
		params  map[string]string
		body    []byte
		want    string
	}{
		{
			name:    "get without params",
			project: Project{Name: "p", BaseURL: "http://api.example.com"},
			api:     API{Name: "listPets", Method: "GET", Path: "/pets"},
			want:    `curl 'http://api.example.com/pets'`,
		},
		{
			name:    "post with body adds method and data",
			project: Project{Name: "p", BaseURL: "http://api.example.com"},
			api:     API{Name: "createPet", Method: "POST", Path: "/pets"},
			body:    []byte(`{"name":"rex"}`),
			want:    `curl -X POST 'http://api.example.com/pets' -H 'Content-Type: application/json' -d '{"name":"rex"}'`,
		},
		{
			name:    "path and query substitution",
			project: Project{Name: "p", BaseURL: "http://api.example.com/"},
			api: API{
				Name: "getPet", Method: "GET", Path: "/pets/{id}",
				PathParams:  []Parameter{{Name: "id"}},
				QueryParams: []Parameter{{Name: "full"}},
			},
			params: map[string]string{"id": "42", "full": "true", "ignored": "x"},
			want:   `curl 'http://api.example.com/pets/42?full=true'`,
		},
		{
			name:    "headers included",
			project: Project{Name: "p", BaseURL: "http://api.example.com"},
			api: API{
				Name: "upload", Method: "PUT", Path: "/files",
				Headers: []Parameter{{Name: "X-Token"}, {Name: "Content-Type"}},
			},
			params: map[string]string{"X-Token": "secret", "Content-Type": "text/plain"},
			body:   []byte("hello"),
			want:   `curl -X PUT 'http://api.example.com/files' -H 'X-Token: secret' -H 'Content-Type: text/plain' -d 'hello'`,
		},
		{
			name:    "single quotes in values are escaped",
			project: Project{Name: "p", BaseURL: "http://api.example.com"},
			api:     API{Name: "createPet", Method: "DELETE", Path: "/pets/o'brien"},
			want:    `curl -X DELETE 'http://api.example.com/pets/o'\''brien'`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := curlCommand(tt.project, tt.api, tt.params, tt.body)
			if got != tt.want {
				t.Fatalf("curlCommand() =\n%q\nwant\n%q", got, tt.want)
			}
		})
	}
}
