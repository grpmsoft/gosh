# GoSh Makefile
# Cross-platform Go shell

BINARY = gosh
VERSION ?= v0.1.0-beta.2
GOARCH ?= amd64
GOOS ?= $(shell go env GOOS)

# Version injection via ldflags
GIT_COMMIT = $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_DATE = $(shell date -u '+%Y-%m-%d_%H:%M:%S' 2>/dev/null || echo "unknown")

LDFLAGS = -ldflags "\
	-X 'main.Version=$(VERSION)' \
	-X 'main.GitCommit=$(GIT_COMMIT)' \
	-X 'main.BuildDate=$(BUILD_DATE)'"

# Default target
.DEFAULT_GOAL := build

# Build binary
build:
	@echo "Building $(BINARY) $(VERSION) for $(GOOS)/$(GOARCH)..."
	@mkdir -p bin
	GO111MODULE=on go build $(LDFLAGS) -o bin/$(BINARY)$(shell test "$(GOOS)" = "windows" && echo ".exe") ./cmd/gosh

# Build with debugging symbols
build-debug:
	@echo "Building $(BINARY) $(VERSION) with debug symbols..."
	@mkdir -p bin
	GO111MODULE=on go build $(LDFLAGS) -gcflags="all=-N -l" -o bin/$(BINARY)$(shell test "$(GOOS)" = "windows" && echo ".exe") ./cmd/gosh

# Install to system (Unix-like only)
install: build
	@echo "Installing $(BINARY)..."
	@if [ "$(GOOS)" != "windows" ]; then \
		install -m 0755 bin/$(BINARY) /usr/local/bin/$(BINARY); \
		echo "Installed to /usr/local/bin/$(BINARY)"; \
	else \
		echo "ERROR: Install target not supported on Windows"; \
		echo "Please copy bin/$(BINARY).exe to a directory in your PATH manually"; \
		exit 1; \
	fi

# Run all tests
test:
	@echo "Running tests..."
	go test -v -coverprofile=coverage.out ./...

# Run tests with coverage report
test-coverage: test
	@echo "Generating coverage report..."
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

# Run tests with race detector
test-race:
	@echo "Running tests with race detector..."
	go test -v -race -coverprofile=coverage.out ./...

# Run benchmarks
benchmark:
	@echo "Running benchmarks..."
	go test -bench=. -benchmem ./...

# Run linter
lint:
	@echo "Running linter..."
	golangci-lint run --timeout=5m

# Format code
fmt:
	@echo "Formatting code..."
	gofmt -w -s .
	go mod tidy

# Check code formatting (CI-friendly, no changes)
fmt-check:
	@echo "Checking code formatting..."
	@if [ -n "$$(gofmt -l .)" ]; then \
		echo "ERROR: The following files are not formatted:"; \
		gofmt -l .; \
		echo ""; \
		echo "Run 'make fmt' to fix formatting issues."; \
		exit 1; \
	fi
	@echo "All files are properly formatted ✓"

# Clean build artifacts
clean:
	@echo "Cleaning..."
	rm -rf bin/
	rm -rf dist/
	rm -f coverage.out coverage.html

# Run shell locally
run: build
	@echo "Running $(BINARY)..."
	./bin/$(BINARY)$(shell test "$(GOOS)" = "windows" && echo ".exe")

# Create release tarball
release: build
	@echo "Creating release package..."
	tar -czf $(BINARY)-$(VERSION)-$(GOOS)-$(GOARCH).tar.gz \
		-C bin $(BINARY)$(shell test "$(GOOS)" = "windows" && echo ".exe")
	@echo "Release: $(BINARY)-$(VERSION)-$(GOOS)-$(GOARCH).tar.gz"

# Multi-platform builds
build-linux:
	@echo "Building for Linux amd64..."
	GOOS=linux GOARCH=amd64 $(MAKE) build
	@mv bin/$(BINARY) bin/$(BINARY)-linux-amd64

build-linux-arm64:
	@echo "Building for Linux arm64..."
	GOOS=linux GOARCH=arm64 $(MAKE) build
	@mv bin/$(BINARY) bin/$(BINARY)-linux-arm64

build-windows:
	@echo "Building for Windows amd64..."
	GOOS=windows GOARCH=amd64 $(MAKE) build
	@mv bin/$(BINARY).exe bin/$(BINARY)-windows-amd64.exe

build-darwin:
	@echo "Building for macOS amd64..."
	GOOS=darwin GOARCH=amd64 $(MAKE) build
	@mv bin/$(BINARY) bin/$(BINARY)-darwin-amd64

build-darwin-arm64:
	@echo "Building for macOS arm64 (Apple Silicon)..."
	GOOS=darwin GOARCH=arm64 $(MAKE) build
	@mv bin/$(BINARY) bin/$(BINARY)-darwin-arm64

build-all: build-linux build-linux-arm64 build-windows build-darwin build-darwin-arm64
	@echo "All platforms built successfully!"

# Development workflow
dev: fmt lint test build
	@echo "Development build complete!"

# CI/CD checks (includes formatting check)
ci: fmt-check test lint
	@echo "CI checks passed!"

# Help
help:
	@echo "GoSh Makefile"
	@echo ""
	@echo "Usage:"
	@echo "  make build         - Build binary for current platform"
	@echo "  make test          - Run tests"
	@echo "  make test-coverage - Run tests with coverage report"
	@echo "  make test-race     - Run tests with race detector"
	@echo "  make benchmark     - Run benchmarks"
	@echo "  make lint          - Run linter"
	@echo "  make fmt           - Format code"
	@echo "  make fmt-check     - Check code formatting (CI)"
	@echo "  make clean         - Clean build artifacts"
	@echo "  make install       - Install to /usr/local/bin (Unix-like only)"
	@echo "  make run           - Run shell locally"
	@echo "  make release       - Create release tarball"
	@echo "  make build-all     - Build for all platforms"
	@echo "  make dev           - Full development workflow"
	@echo "  make ci            - CI/CD checks"
	@echo ""
	@echo "Platform-specific builds:"
	@echo "  make build-linux       - Linux amd64"
	@echo "  make build-linux-arm64 - Linux arm64"
	@echo "  make build-windows     - Windows amd64"
	@echo "  make build-darwin      - macOS amd64 (Intel)"
	@echo "  make build-darwin-arm64 - macOS arm64 (Apple Silicon)"
	@echo ""
	@echo "Version: $(VERSION)"
	@echo "Commit:  $(GIT_COMMIT)"
	@echo "Date:    $(BUILD_DATE)"

.PHONY: build build-debug install test test-coverage test-race benchmark lint fmt fmt-check clean \
	run release build-linux build-linux-arm64 build-windows build-darwin build-darwin-arm64 build-all \
	dev ci help
