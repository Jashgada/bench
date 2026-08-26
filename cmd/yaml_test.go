package cmd

import "testing"

func TestParseYAMLProject(t *testing.T) {
	spec := []byte(`openapi: 3.1.0
info:
  title: Pokemon API
servers:
  - url: https://pokeapi.co
paths:
  /api/v2/pokemon/{id}/:
    get:
      operationId: pokemon_retrieve
      parameters:
        - name: id
          in: path
          required: true
          schema:
            type: string
  /api/v2/pokemon/:
    get:
      operationId: pokemon_list
      parameters:
        - name: limit
          in: query
          required: false
          schema:
            type: integer
`)
	project, err := parseProject(spec, "pokemon", "", "pokemon.yml")
	if err != nil {
		t.Fatal(err)
	}
	if project.BaseURL != "https://pokeapi.co" || len(project.APIs) != 2 {
		t.Fatalf("unexpected project: %#v", project)
	}
	if project.APIs[0].Name == "" || project.APIs[1].Name == "" {
		t.Fatalf("missing operation IDs: %#v", project.APIs)
	}
}
