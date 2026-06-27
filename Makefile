.PHONY: build run test lint vet clean install dev server

BINARY := baize
GOFLAGS := -ldflags="-s -w"
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")

build: ## Build the binary
	go build $(GOFLAGS) -o $(BINARY) ./cli/

run: build ## Build and run interactive mode
	./$(BINARY)

server: build ## Build and run API server
	./$(BINARY) server --port 9779

install: ## Install to $GOPATH/bin
	go install ./cli/

test: ## Run all tests
	go test -v -race -count=1 ./...

test-short: ## Run tests (no race detector, faster)
	go test -v -count=1 ./...

bench: ## Run benchmarks
	go test -bench=. -benchmem ./...

lint: ## Run all linters
	golangci-lint run ./...

vet: ## Run go vet
	go vet ./...

fmt: ## Format code
	go fmt ./...

fmt-check: ## Check code formatting
	@test -z "$$(gofmt -l . | grep -v vendor)" || (echo "Files need formatting:" && gofmt -l . && exit 1)

cover: ## Run tests with coverage
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

clean: ## Remove build artifacts
	rm -f $(BINARY) coverage.out coverage.html

dev: ## Start development (server + watch)
	@echo "Starting Baize server on http://127.0.0.1:9779"
	./$(BINARY) server --port 9779

all: fmt lint test build ## Format, lint, test, build

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'
