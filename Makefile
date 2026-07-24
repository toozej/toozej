# Set sane defaults for Make
SHELL = bash
.DELETE_ON_ERROR:
MAKEFLAGS += --warn-undefined-variables
MAKEFLAGS += --no-builtin-rules

# Set default goal such that `make` runs `make help`
.DEFAULT_GOAL := help

# Build info
BUILDER = $(shell whoami)@$(shell hostname)
NOW = $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

# Version control
VERSION = $(shell git describe --tags --dirty --always 2>/dev/null || echo "dev")
COMMIT = $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BRANCH = $(shell git rev-parse --abbrev-ref HEAD 2>/dev/null || echo "unknown")

# Linker flags
PKG = $(shell head -n 1 go.mod | cut -c 8-)
VER = $(PKG)/internal/version
LDFLAGS = -s -w \
	-X $(VER).Version=$(VERSION) \
	-X $(VER).Commit=$(COMMIT) \
	-X $(VER).Branch=$(BRANCH) \
	-X $(VER).BuiltAt=$(NOW) \
	-X $(VER).Builder=$(BUILDER)

BINARY_NAME = readme-gen
BIN_DIR = $(CURDIR)/bin

.PHONY: all tidy fmt vet test build run clean help

all: tidy fmt vet test build ## Run default workflow (tidy, fmt, vet, test, build)

tidy: ## Run `go mod tidy`
	go mod tidy

fmt: ## Run `go fmt`
	go fmt ./...

vet: ## Run `go vet`
	go vet ./...

test: ## Run `go test` with race detection
	mkdir -p $(BIN_DIR)
	go test -race -coverprofile=$(BIN_DIR)/coverage.out ./...
	@grep -v -e " 1$$" $(BIN_DIR)/coverage.out || true

build: ## Build the readme-gen binary into ./bin
	mkdir -p $(BIN_DIR)
	CGO_ENABLED=0 go build -trimpath -ldflags="$(LDFLAGS)" -o $(BIN_DIR)/$(BINARY_NAME) ./cmd/readme-gen

run: build ## Run the readme-gen binary (requires GITHUB_TOKEN env var)
	$(BIN_DIR)/$(BINARY_NAME) -template $(CURDIR)/templates/README.md.tpl -output $(CURDIR)/README.md

clean: ## Remove built binaries and coverage files
	rm -rf $(BIN_DIR)

help: ## Display help text
	@grep -E '^[a-zA-Z_-]+ ?:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-25s\033[0m %s\n", $$1, $$2}'
