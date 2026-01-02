# bench

> curl lets you lift. bench lets you lift *heavier*.

A fast CLI tool to supercharge curl with Swagger-powered API collections.

## Vision

bench is a CLI-native alternative to Postman. Import a Swagger/OpenAPI spec, and instantly get a searchable collection of APIs ready to execute.

## Core Workflow

1. **Import** - Point bench at a Swagger JSON → parses and stores all endpoints
2. **Search** - Fuzzy finder dropdown to browse APIs by name, path, or method (GET/POST/etc.)
3. **Execute** - We know the method, path, headers from the spec. User just provides dynamic values (path params, query params, request body)

## Key Features

- [ ] Swagger/OpenAPI JSON parser
- [ ] Project/collection storage (local JSON or SQLite)
- [ ] Fuzzy search TUI for API discovery
- [ ] Smart prompting for required params only
- [ ] Request execution with timing
- [ ] Response formatting (JSON pretty-print, syntax highlighting)

## Tech Stack

- **Language:** Go
- **CLI Framework:** Cobra
- **TUI/Dropdown:** Bubbletea + Bubbles
- **HTTP Client:** net/http
- **Storage:** JSON files per project (simple) or SQLite (if needed)

## Commands (Planned)

```
bench init <swagger.json>     # Create a new project from Swagger spec
bench list                    # List all APIs in current project
bench search                  # Open fuzzy finder to search/select API
bench run <api-name>          # Execute an API (prompts for params)
```

## Data Model

### Project
- name
- base_url
- apis[]

### API
- name (operation ID or generated)
- method (GET, POST, PUT, DELETE, etc.)
- path (/users/{id})
- path_params[]
- query_params[]
- request_body_schema
- headers

## Directory Structure

```
~/.bench/
  projects/
    my-api/
      project.json
```

## Getting Started

TBD

## Usage Examples

TBD
