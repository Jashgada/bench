GO := $(shell command -v go 2>/dev/null || printf /usr/local/go/bin/go)
BIN := $(shell $(GO) env GOPATH)/bin/bench

.PHONY: all fmt test build check install serve run

all: check build

fmt:
	$(GO)fmt -w main.go cmd/*.go

test:
	$(GO) test ./...

build:
	$(GO) build ./...

check: fmt test

install: check
	$(GO) install .
	@printf 'Installed bench at %s\n' '$(BIN)'

serve:
	$(GO) run . serve

run:
	$(GO) run .
