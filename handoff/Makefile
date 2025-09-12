# Handoff Agent Makefile

.PHONY: help build test lint run clean docker-up docker-down deps format vet

# Default target
help:
	@echo "Available targets:"
	@echo "  build      - Build the handoff agent binary"
	@echo "  test       - Run tests with coverage"
	@echo "  lint       - Run linting"
	@echo "  run        - Run the handoff agent service"
	@echo "  clean      - Clean build artifacts"
	@echo "  docker-up  - Start Redis with docker-compose"
	@echo "  docker-down- Stop Redis docker-compose"
	@echo "  deps       - Download and tidy dependencies"
	@echo "  format     - Format Go code"
	@echo "  vet        - Run go vet"

# Build settings
BINARY_NAME=handoff-agent
BUILD_DIR=./bin
MAIN_PACKAGE=./cmd

build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	go build -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_PACKAGE)

test:
	@echo "Running tests..."
	go test -race -cover -v ./...

lint:
	@echo "Running linting..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not found, installing..."; \
		go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest; \
		golangci-lint run; \
	fi

run: build docker-up
	@echo "Starting handoff agent..."
	$(BUILD_DIR)/$(BINARY_NAME) -config config.json

clean:
	@echo "Cleaning build artifacts..."
	rm -rf $(BUILD_DIR)
	go clean

docker-up:
	@echo "Starting Redis with docker-compose..."
	@cd .. && docker-compose up -d redis

docker-down:
	@echo "Stopping Redis docker-compose..."
	@cd .. && docker-compose down

deps:
	@echo "Downloading and tidying dependencies..."
	go mod download
	go mod tidy

format:
	@echo "Formatting Go code..."
	go fmt ./...

vet:
	@echo "Running go vet..."
	go vet ./...

# Development workflow
dev: deps format vet lint test build

# Full CI pipeline
ci: deps format vet lint test

# Integration test (requires Redis)
integration-test: docker-up
	@echo "Running integration tests..."
	go test -tags=integration -v ./...
	@$(MAKE) docker-down

# Benchmark tests
bench:
	@echo "Running benchmarks..."
	go test -bench=. -benchmem ./...

# Install binary to GOPATH/bin
install: build
	@echo "Installing $(BINARY_NAME)..."
	go install $(MAIN_PACKAGE)

# Generate test coverage report
coverage:
	@echo "Generating test coverage report..."
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Run with different log levels
run-debug: build docker-up
	$(BUILD_DIR)/$(BINARY_NAME) -config config.json -log-level debug

run-quiet: build docker-up
	$(BUILD_DIR)/$(BINARY_NAME) -config config.json -log-level error

# Docker build (if needed in future)
docker-build:
	@echo "Building Docker image..."
	docker build -t handoff-agent:latest .

# Check for security issues
security:
	@echo "Running security scan..."
	@if command -v gosec >/dev/null 2>&1; then \
		gosec ./...; \
	else \
		echo "gosec not found, installing..."; \
		go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest; \
		gosec ./...; \
	fi