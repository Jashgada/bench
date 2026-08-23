# bench

`bench` turns an OpenAPI 3 JSON document into an executable local API collection.
It is a request runner first, with a terminal browser for discovering operations.

## Installation

Build from source with Go 1.24 or newer:

```bash
go install github.com/jashgada/bench@latest
```

Make sure Go's bin directory is on your `PATH`. Alternatively, build a local
binary from a checkout:

```bash
go build -o bench .
```

## Quick Start

Import an OpenAPI 3 JSON document. A project name is required:

```bash
bench init petstore.json --name pets
```

List operations in the current project:

```bash
bench list
bench list --filter pet
```

Open the interactive terminal browser:

```bash
bench search
```

The browser is designed around a K9s-like keyboard workflow: use `j`/`k` or
the arrow keys to move, `enter` to inspect an operation, `/` to filter, `esc`
to go back, and `q` to quit.

Run an operation by operation ID. Required path, query, and header parameters
are prompted interactively:

```bash
bench run listPets
```

Request bodies can be supplied inline, from a file, or through stdin:

```bash
bench run createPet --body '{"name":"Fido"}'
bench run createPet --body-file request.json
cat request.json | bench run createPet
```

Refresh a project from the source used during initialization:

```bash
bench update --project pets
```

Delete a project:

```bash
bench delete --project pets
```

Use `--project <name>` with `list` or `run` when working with a project other
than the current one.

## Storage

Projects are stored locally at:

```text
~/.bench/projects/<name>/project.json
```

The most recently initialized project is used by default. The original local
file path or HTTP(S) URL is stored so `bench update` can refresh it.

## Current Scope

Supported:

- OpenAPI 3 JSON
- Operation IDs, methods, paths, parameters, request bodies, and server URLs
- Local normalized JSON project storage
- HTTP request execution with response headers and formatted JSON output

Not yet supported:

- Swagger 2, YAML, and automatic `$ref` resolution
- Authentication, environments, and secrets
- Full JSON Schema-driven prompting
