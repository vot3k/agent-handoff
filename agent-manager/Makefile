# Makefile for Agent Manager HTTP Server

.PHONY: build test lint run clean help server manager publisher

# Variables
BINARY_NAME_SERVER = agent-server
BINARY_NAME_MANAGER = agent-manager
BINARY_NAME_PUBLISHER = agent-publisher
BUILD_DIR = bin
CMD_SERVER_DIR = cmd/server
CMD_MANAGER_DIR = cmd/manager  
CMD_PUBLISHER_DIR = cmd/publisher

# Go parameters
GOCMD = go
GOBUILD = $(GOCMD) build
GOTEST = $(GOCMD) test
GOMOD = $(GOCMD) mod
GOVET = $(GOCMD) vet
GOFMT = gofmt

# Build all binaries
build: server manager publisher

# Build HTTP server
server:
	@echo "Building HTTP server..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME_SERVER) ./$(CMD_SERVER_DIR)

# Build existing manager
manager:
	@echo "Building manager..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME_MANAGER) ./$(CMD_MANAGER_DIR)

# Build existing publisher
publisher:
	@echo "Building publisher..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME_PUBLISHER) ./$(CMD_PUBLISHER_DIR)

# Run tests
test:
	@echo "Running tests..."
	$(GOTEST) -race -cover ./...

# Run tests with verbose output
test-verbose:
	@echo "Running tests with verbose output..."
	$(GOTEST) -race -cover -v ./...

# Run benchmarks
bench:
	@echo "Running benchmarks..."
	$(GOTEST) -bench=. -benchmem ./...

# Lint code
lint:
	@echo "Running go vet..."
	$(GOVET) ./...
	@echo "Checking gofmt..."
	@if [ -n "$(shell $(GOFMT) -l .)" ]; then \
		echo "Code is not properly formatted. Please run 'make fmt'"; \
		$(GOFMT) -l .; \
		exit 1; \
	fi

# Format code
fmt:
	@echo "Formatting code..."
	$(GOFMT) -w .

# Download dependencies
deps:
	@echo "Downloading dependencies..."
	$(GOMOD) download
	$(GOMOD) tidy

# Run HTTP server locally
run-server: server
	@echo "Starting HTTP server..."
	./$(BUILD_DIR)/$(BINARY_NAME_SERVER)

# Run manager locally
run-manager: manager
	@echo "Starting manager..."
	./$(BUILD_DIR)/$(BINARY_NAME_MANAGER)

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf $(BUILD_DIR)

# Docker build
docker-build:
	@echo "Building Docker image..."
	docker build -t agent-manager:latest .

# Development setup
dev-setup:
	@echo "Setting up development environment..."
	@echo "Checking Go version..."
	@$(GOCMD) version
	@echo "Installing dependencies..."
	$(MAKE) deps

# Check for security vulnerabilities
security:
	@echo "Checking for security vulnerabilities..."
	@command -v govulncheck >/dev/null 2>&1 || { echo "Installing govulncheck..."; $(GOCMD) install golang.org/x/vuln/cmd/govulncheck@latest; }
	govulncheck ./...

# Generate test coverage report
coverage:
	@echo "Generating coverage report..."
	$(GOTEST) -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Quick development cycle
dev: clean lint test build
	@echo "Development build complete!"

# Production build with optimizations
prod-build:
	@echo "Building for production..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 GOOS=linux $(GOBUILD) -a -installsuffix cgo -ldflags '-w -s' -o $(BUILD_DIR)/$(BINARY_NAME_SERVER)-linux ./$(CMD_SERVER_DIR)
	CGO_ENABLED=0 GOOS=linux $(GOBUILD) -a -installsuffix cgo -ldflags '-w -s' -o $(BUILD_DIR)/$(BINARY_NAME_MANAGER)-linux ./$(CMD_MANAGER_DIR)

# Help
help:
	@echo "Available targets:"
	@echo "  build        - Build all binaries"
	@echo "  server       - Build HTTP server binary"
	@echo "  manager      - Build manager binary" 
	@echo "  publisher    - Build publisher binary"
	@echo "  test         - Run tests"
	@echo "  test-verbose - Run tests with verbose output"
	@echo "  bench        - Run benchmarks"
	@echo "  lint         - Lint code"
	@echo "  fmt          - Format code"
	@echo "  deps         - Download dependencies"
	@echo "  run-server   - Build and run HTTP server"
	@echo "  run-manager  - Build and run manager"
	@echo "  clean        - Clean build artifacts"
	@echo "  docker-build - Build Docker image"
	@echo "  dev-setup    - Setup development environment"
	@echo "  security     - Check for security vulnerabilities"
	@echo "  coverage     - Generate test coverage report"
	@echo "  dev          - Quick development cycle (clean, lint, test, build)"
	@echo "  prod-build   - Production build with optimizations"
	@echo "  help         - Show this help"

# Default target
.DEFAULT_GOAL := build