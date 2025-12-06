.PHONY: all build test clean install lint fmt help

# Variables
BINARY_NAME := nettune
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_DATE := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS := -ldflags "-X github.com/jtsang4/nettune/pkg/version.Version=$(VERSION) \
                     -X github.com/jtsang4/nettune/pkg/version.GitCommit=$(GIT_COMMIT) \
                     -X github.com/jtsang4/nettune/pkg/version.BuildDate=$(BUILD_DATE)"

# Output directory
DIST_DIR := dist

# Go settings
GOCMD := go
GOBUILD := $(GOCMD) build
GOTEST := $(GOCMD) test
GOCLEAN := $(GOCMD) clean
GOGET := $(GOCMD) get
GOMOD := $(GOCMD) mod

all: build

## Build the binary for the current platform
build:
	@echo "Building $(BINARY_NAME)..."
	$(GOBUILD) $(LDFLAGS) -o $(DIST_DIR)/$(BINARY_NAME) ./cmd/nettune

## Build for all supported platforms
build-all:
	@echo "Building for all platforms..."
	@mkdir -p $(DIST_DIR)
	GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(DIST_DIR)/$(BINARY_NAME)-linux-amd64 ./cmd/nettune
	GOOS=linux GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(DIST_DIR)/$(BINARY_NAME)-linux-arm64 ./cmd/nettune
	GOOS=darwin GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(DIST_DIR)/$(BINARY_NAME)-darwin-amd64 ./cmd/nettune
	GOOS=darwin GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(DIST_DIR)/$(BINARY_NAME)-darwin-arm64 ./cmd/nettune
	@echo "Generating checksums..."
	@cd $(DIST_DIR) && sha256sum $(BINARY_NAME)-* > checksums.txt

## Run tests
test:
	$(GOTEST) -v -race ./...

## Run tests with coverage
test-coverage:
	$(GOTEST) -v -race -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html

## Run short tests (no integration tests)
test-short:
	$(GOTEST) -v -short ./...

## Clean build artifacts
clean:
	$(GOCLEAN)
	rm -rf $(DIST_DIR)
	rm -f coverage.out coverage.html

## Install the binary
install:
	$(GOBUILD) $(LDFLAGS) -o $(GOPATH)/bin/$(BINARY_NAME) ./cmd/nettune

## Run linter
lint:
	golangci-lint run ./...

## Format code
fmt:
	gofmt -s -w .

## Tidy dependencies
tidy:
	$(GOMOD) tidy

## Download dependencies
deps:
	$(GOMOD) download

## Run the server (for development)
run-server:
	$(GOCMD) run ./cmd/nettune server --api-key=dev-key

## Run the client (for development)
run-client:
	$(GOCMD) run ./cmd/nettune client --api-key=dev-key --server=http://127.0.0.1:9876

## Show version info
version:
	@echo "Version: $(VERSION)"
	@echo "Git Commit: $(GIT_COMMIT)"
	@echo "Build Date: $(BUILD_DATE)"

## Show help
help:
	@echo "Nettune Makefile"
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@grep -E '^##' Makefile | sed 's/## /  /'
	@echo ""
	@echo "Variables:"
	@echo "  VERSION=$(VERSION)"
	@echo "  GIT_COMMIT=$(GIT_COMMIT)"
