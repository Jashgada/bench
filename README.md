# bench

`bench` turns an OpenAPI 3 JSON or YAML document into an executable local API
collection. It is a request runner first, with a K9s-style terminal browser for
discovering and executing operations.

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
to Go's bin directory. Other useful targets are `make check` (format + test)
and `make build`.

## Quick Start

Import an OpenAPI 3 spec. A project name is required:

```bash
bench init petstore.json --name pets
```

Or straight from a URL:

```bash
bench init https://raw.githubusercontent.com/PokeAPI/pokeapi/master/openapi.yml --name pokemon
```

Open the interactive terminal browser:

```bash
bench
```

The browser follows a K9s-like keyboard workflow:

- `j`/`k` or arrow keys move and scroll
- `enter` inspects an operation
- `r` runs it, filling parameters and request body inline
- `/` filters operations, `?` shows all keybindings
- `:` opens command mode, `q` quits

The response appears in a split panel below the browser with status, timing,
and pretty-printed JSON. The body is searchable (`/`), scrollable, and copyable
(`c`), with a headers tab (`tab` to switch).

## Command Bar

```text
:projects        open the project picker
:p               short alias for :projects
:project pets    switch directly to the pets project
:envs            open the environment picker
:env dev         switch directly to the dev environment
:curl ...        run an ad-hoc curl command without a saved operation
:group pokemon   show only operations tagged pokemon
:group all       clear the tag group filter
:reload          reload the current project from local storage
:update          fetch the current project source and reload it
:q, :quit        quit bench
```

## CLI Usage

List operations in the current project:

```bash
bench list
bench list --filter pet
```

Run an operation by operation ID. Required path, query, and header parameters
are prompted interactively. Request bodies can be supplied inline, from a
file, or through stdin:

```bash
bench run listPets
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

## Ad-hoc Requests

The `:curl` command runs any curl-style command line through bench's engine,
so the response lands in the same scrollable, searchable, copyable panel as
normal requests:

```text
:curl -X POST https://api.example.com/pets -H 'X-Tag: best friend' -d '{"name":"Fido"}'
```

## Storage

Projects are stored locally at:

```text
~/.bench/projects/<name>/project.json
```

Environments live alongside them:

```text
~/.bench/projects/<name>/environments/<env>.json
```

The most recently initialized project is used by default. The original local
file path or HTTP(S) URL is stored so `bench update` can refresh it.

## Current Scope

Supported:

- OpenAPI 3 JSON and YAML
- Operation IDs, methods, paths, parameters, tags, request bodies, and server URLs
- Security scheme parsing with automatic credential injection
- Local normalized JSON project storage and environments
- HTTP request execution with response headers and formatted JSON output

Not yet supported:

- Swagger 2 and automatic `$ref` resolution
- OAuth flows
- Full JSON Schema-driven prompting
