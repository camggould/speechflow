.DEFAULT_GOAL := help

# Version metadata injected into the binary at build time.
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
DATE    ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)

LDFLAGS := -X 'github.com/camggould/speechflow/internal/cli.buildVersion=$(VERSION)' \
           -X 'github.com/camggould/speechflow/internal/cli.buildCommit=$(COMMIT)' \
           -X 'github.com/camggould/speechflow/internal/cli.buildDate=$(DATE)'

.PHONY: help
help:  ## Show this help.
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-12s\033[0m %s\n", $$1, $$2}'

.PHONY: setup
setup:  ## Install Go and UI dependencies; regenerate TS types.
	go mod download
	cd ui && npm install
	$(MAKE) ui-types

.PHONY: dev
dev:  ## Build UI then run the API server with the embedded build.
	$(MAKE) ui
	$(MAKE) build
	./speechflow serve

# sqlite_fts5 enables FTS5 full-text search in the bundled SQLite amalgamation.
GOTAGS ?= sqlite_fts5

.PHONY: build
build:  ## Build the speechflow binary.
	go build -tags "$(GOTAGS)" -ldflags="$(LDFLAGS)" -o speechflow ./cmd/speechflow

.PHONY: test
test:  ## Run all Go tests.
	go test -tags "$(GOTAGS)" ./...

.PHONY: lint
lint:  ## Run go vet.
	go vet -tags "$(GOTAGS)" ./...

.PHONY: ui
ui:  ## Build the UI (produces internal/uifs/dist/).
	cd ui && npm run build

.PHONY: ui-dev
ui-dev:  ## Start the UI dev server with HMR.
	cd ui && npm run dev

.PHONY: ui-types
ui-types:  ## Regenerate TypeScript types from Go core types.
	tygo generate

.PHONY: clean
clean:  ## Remove build artifacts.
	rm -f speechflow coverage.out
