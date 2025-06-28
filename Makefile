.PHONY: help build test lint format clean docker-build docker-run

# Default target
help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

# Build the application
build: ## Build the application
	go build -o bin/monitor cmd/monitor/main.go

# Run tests
test: ## Run tests
	go test -v ./...

# Run tests with coverage
test-coverage: ## Run tests with coverage report
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Lint the code
lint: ## Run linter
	golangci-lint run

# Format the code
format: ## Format Go code
	go fmt ./...
	goimports -w .

# Clean build artifacts
clean: ## Clean build artifacts
	rm -rf bin/
	rm -f coverage.out coverage.html

# Install dependencies
deps: ## Install dependencies
	go mod download
	go mod tidy

# Install development tools
install-tools: ## Install development tools
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install golang.org/x/tools/cmd/goimports@latest
	go install github.com/go-delve/delve/cmd/dlv@latest

# Run the application
run: ## Run the application
	go run cmd/monitor/main.go

# Docker build
docker-build: ## Build Docker image
	docker build -t gswarm-sidecar .

# Docker run
docker-run: ## Run with Docker Compose
	docker-compose up --build

# Pre-commit checks
pre-commit: format lint test ## Run all pre-commit checks

# Development setup
setup: install-tools deps ## Setup development environment 