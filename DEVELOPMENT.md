# Development Guide

This guide covers the development tools and practices used in the GSwarm Side Car project.

## Prerequisites

- Go 1.24 or later
- Docker and Docker Compose
- Make (for using the Makefile)

## Quick Setup

1. **Clone and setup the project:**
   ```bash
   git clone <repository-url>
   cd gswarm-sidecar
   make setup
   ```

2. **Install pre-commit hooks:**
   ```bash
   pre-commit install
   ```

## Development Tools

### Makefile Commands

The project includes a comprehensive Makefile with common development tasks:

```bash
make help          # Show all available commands
make setup         # Install tools and dependencies
make build         # Build the application
make test          # Run tests
make test-coverage # Run tests with coverage report
make lint          # Run linter
make format        # Format code
make clean         # Clean build artifacts
make run           # Run the application
make docker-build  # Build Docker image
make docker-run    # Run with Docker Compose
make pre-commit    # Run all pre-commit checks
```

### Code Quality Tools

#### golangci-lint
- **Configuration**: `.golangci.yml`
- **Usage**: `make lint` or `golangci-lint run`
- **Purpose**: Comprehensive Go linting with multiple linters

#### Pre-commit Hooks
- **Configuration**: `.pre-commit-config.yaml`
- **Purpose**: Automatically run code quality checks before commits
- **Installation**: `pre-commit install`

#### EditorConfig
- **Configuration**: `.editorconfig`
- **Purpose**: Consistent coding style across different editors

### VS Code Setup

The project includes VS Code workspace settings for:
- Go language server
- Automatic formatting with goimports
- Linting with golangci-lint
- File formatting on save

Recommended extensions are listed in `.vscode/extensions.json`.

## Development Workflow

### 1. Starting Development
```bash
make setup          # Install tools
pre-commit install  # Install git hooks
make deps           # Install dependencies
```

### 2. Daily Development
```bash
make run            # Run the application
make test           # Run tests
make lint           # Check code quality
```

### 3. Before Committing
```bash
make pre-commit     # Run all checks
# or let pre-commit hooks handle it automatically
```

### 4. Building and Deploying
```bash
make build          # Build binary
make docker-build   # Build Docker image
make docker-run     # Run with Docker Compose
```

## Testing

### Running Tests
```bash
make test                    # Run all tests
make test-coverage          # Run tests with coverage
go test ./internal/...      # Run tests for specific package
go test -v -race ./...      # Run tests with race detection
```

### Test Structure
- Unit tests: `*_test.go` files alongside source code
- Integration tests: `tests/` directory (if needed)
- Test coverage target: 80%+

## Code Style

### Go Code
- Use `gofmt` and `goimports` for formatting
- Follow Go naming conventions
- Use meaningful variable and function names
- Add comments for exported functions and types

### Configuration Files
- YAML files: 2-space indentation
- JSON files: 2-space indentation
- Go files: Tab indentation

## Continuous Integration

The project uses GitHub Actions for CI/CD:
- **Tests**: Run on multiple Go versions
- **Linting**: golangci-lint checks
- **Security**: gosec security scanning
- **Build**: Verify builds work
- **Docker**: Build and test Docker images

## Debugging

### Using Delve
```bash
dlv debug cmd/monitor/main.go
```

### VS Code Debugging
The project includes VS Code debug configuration for:
- Debugging the main application
- Debugging tests
- Remote debugging

## Common Issues

### Linting Errors
- Run `make lint` to see all issues
- Use `golangci-lint run --fix` to auto-fix some issues
- Check `.golangci.yml` for linter configuration

### Import Issues
- Use `goimports -w .` to fix import organization
- Ensure all imports are used or removed

### Test Failures
- Check test coverage with `make test-coverage`
- Run tests with verbose output: `go test -v ./...`
- Use race detection: `go test -race ./...`

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Run `make pre-commit` to ensure code quality
5. Submit a pull request

## Resources

- [Go Best Practices](https://golang.org/doc/effective_go.html)
- [golangci-lint Documentation](https://golangci-lint.run/)
- [Pre-commit Documentation](https://pre-commit.com/)
- [VS Code Go Extension](https://marketplace.visualstudio.com/items?itemName=golang.go)
