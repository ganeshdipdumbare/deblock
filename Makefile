.PHONY: all build test run clean deps lint docker-build docker-run docker-stop

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOTEST=$(GOCMD) test
GOCLEAN=$(GOCMD) clean
GOMOD=$(GOCMD) mod

# Binary name
BINARY_NAME=deblock

# Build directory
BUILD_DIR=./build

# Linter
GOLANGCI_LINT=golangci-lint

# Docker Compose
DOCKER_COMPOSE=docker-compose

# Default target
all: deps lint test build

# Install dependencies
deps:
	$(GOMOD) download
	$(GOMOD) tidy

# Run linter
lint:
	$(GOLANGCI_LINT) run ./...

# Run tests
test: test-integration

# Run tests with coverage
test-coverage:
	$(GOTEST) -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html

# Build the binary
build:
	mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME) -v ./
	chmod +x $(BUILD_DIR)/$(BINARY_NAME)
	@echo "Checking binary permissions..."
	@ls -l $(BUILD_DIR)/$(BINARY_NAME)
	@if [ ! -x $(BUILD_DIR)/$(BINARY_NAME) ]; then \
		echo "Error: Binary is not executable"; \
		exit 1; \
	fi

# Clean build artifacts
clean:
	$(GOCLEAN)
	rm -rf $(BUILD_DIR)
	rm -f coverage.out coverage.html

# Run the application locally
run:
	$(GOCMD) run .

# Docker build with verbose output
docker-build:
	@echo "Building Docker image..."
	$(DOCKER_COMPOSE) build --progress=plain

# Start Docker services with verbose output
docker-run:
	@echo "Starting Docker services..."
	$(DOCKER_COMPOSE) up -d --force-recreate
	@echo "Checking container status..."
	$(DOCKER_COMPOSE) ps
	@echo "Showing container logs..."
	$(DOCKER_COMPOSE) logs deblock

# Stop Docker services
docker-stop:
	$(DOCKER_COMPOSE) down

# Restart Docker services
docker-restart: docker-stop docker-run

# View Docker logs
docker-logs:
	$(DOCKER_COMPOSE) logs -f

# Generate Swagger documentation
swagger:
	swag init -g cmd/rest.go \
		--parseDependency \
		--parseInternal \
		--parseDepth 1 \
		--outputTypes go \
		--generalInfo cmd/rest.go \
		--output docs

# Help target
help:
	@echo "Available targets:"
	@echo "  all            - Run deps, lint, test, and build"
	@echo "  deps           - Download dependencies"
	@echo "  lint           - Run golangci-lint"
	@echo "  test           - Run tests"
	@echo "  test-coverage  - Run tests with coverage report"
	@echo "  build          - Build the binary"
	@echo "  clean          - Clean build artifacts"
	@echo "  run            - Run the application locally"
	@echo "  docker-build   - Build Docker images"
	@echo "  docker-run     - Start Docker services"
	@echo "  docker-stop    - Stop Docker services"
	@echo "  docker-restart - Restart Docker services"
	@echo "  docker-logs    - View Docker service logs"
	@echo "  swagger        - Generate Swagger documentation"
	@echo "  help           - Show this help message"

# Default target
.DEFAULT_GOAL := help

# Integration tests for Ethereum client
test-ethereum-integration:
	@echo "Running Ethereum client integration tests"
	@TEST_ETHEREUM_RPC_URL=$(ETHEREUM_RPC_URL) \
	 TEST_ETHEREUM_WS_URL=$(ETHEREUM_WS_URL) \
	 go test ./internal/blockchain -v -run TestSubscribeToBlocks

# Run integration tests
test-integration:
	@echo "Running integration tests"
	@go test -tags=integration ./... -v -run=Integration
