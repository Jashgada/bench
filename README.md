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

From a checkout, `make install` formats, tests, and installs the latest build
to Go's bin directory. Other useful targets are `make check`, `make build`,
`make run`, and `make serve`.

## Quick Start

Start the included local Petstore mock API in a separate terminal:

```bash
bench serve
```

Then import the matching fixture against it:

```bash
bench init cmd/testdata/pets.json --name pets --base-url http://localhost:8080
bench list
bench run listPets
```

The mock API starts with two pets and supports `GET`/`POST /pets` plus
`GET`/`PUT`/`DELETE /pets/{id}`.

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
bench
```

The browser is designed around a K9s-like keyboard workflow: use `j`/`k` or
the arrow keys to move, `enter` to inspect an operation, `r` to run it, `/` to
filter, `esc` to go back, and `q` to quit. Running an operation suspends the
browser and reuses the normal prompts for parameters and request JSON.

The command bar also supports project navigation:

```text
:projects       open the project picker
:p              short alias for :projects
:project pets   switch directly to the pets project
:envs           open the environment picker
:env dev        switch directly to the dev environment
:curl ...       run an ad-hoc curl command without a saved operation
:group pokemon   show only operations tagged pokemon
:group all       clear the tag group filter
:reload         reload the current project from local storage
:update         fetch the current project source and reload it
```

## Environments

Environments hold `{{variable}}` values used to substitute the base URL,
parameters, headers, and request bodies at request time. They are strictly
local to your machine.

```bash
bench env add dev --set host=http://localhost:8080 --set api_key='$MY_API_KEY'
bench env use dev
bench env list
bench env rm dev
```

Values starting with `$` resolve from process environment variables at request
time, so secrets never need to touch disk. Unset variables produce a warning
in the response panel. Inside the TUI, `:envs` opens a picker and the status
bar shows the active environment.

## Authentication

If the spec declares `components.securitySchemes` and an operation declares
`security`, bench injects credentials automatically. Credentials come from the
active environment, keyed by scheme name:

```bash
bench env add prod --set bearerAuth='$PROD_TOKEN'
bench env add prod --set basicAuth_username=alice --set basicAuth_password='$PROD_PASS'
bench env add prod --set keyAuth='$PROD_API_KEY'   # apiKey header schemes
```

Supported schemes: `http` bearer, `http` basic, and header-based `apiKey`.
Headers you supply explicitly always win over injected credentials. OAuth
flows are not supported.

## Ad-hoc requests

The `:curl` command runs any curl-style command line through bench's engine,
so the response lands in the same scrollable, searchable, copyable panel as
normal requests:

```text
:curl -X POST https://api.example.com/pets -H 'X-Tag: best friend' -d '{"name":"Fido"}'
```

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

- OpenAPI 3 JSON and YAML
- Operation IDs, methods, paths, parameters, request bodies, and server URLs
- Local normalized JSON project storage
- HTTP request execution with response headers and formatted JSON output

Not yet supported:

- Swagger 2 and automatic `$ref` resolution
- Authentication, environments, and secrets
- Full JSON Schema-driven prompting
