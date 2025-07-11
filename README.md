# GSwarm Side Car Monitoring System

A Go-based side car monitoring system for monitoring **Gensyn AI nodes** in the GSwarm distributed training network. This system collects metrics from running Gensyn AI swarm nodes without modifying the existing codebase, providing real-time data to feed into the gswarm UI.

## Overview

The GSwarm Side Car Monitoring System is designed to monitor Gensyn AI nodes that participate in distributed training tasks. It provides comprehensive monitoring capabilities for:

- **Gensyn AI Node Performance**: Track training progress, model updates, and node health
- **Distributed Training Metrics**: Monitor peer-to-peer communication and model synchronization
- **Blockchain Integration**: Track training submissions and rewards on the Gensyn testnet
- **System Resources**: Monitor hardware utilization and container performance

## Project Structure

```
gswarm-sidecar/
├── cmd/
│   └── monitor/
│       └── main.go              # Main entry point
├── internal/
│   ├── config/
│   │   └── config.go            # Configuration management
│   ├── monitor/
│   │   └── monitor.go           # Main monitor coordinator
│   ├── logs/
│   │   └── monitor.go           # Log file monitoring
│   ├── dht/
│   │   └── monitor.go           # DHT network monitoring
│   ├── blockchain/
│   │   └── monitor.go           # Blockchain event monitoring
│   └── system/
│       └── monitor.go           # System resource monitoring
├── configs/
│   └── config.yaml              # Configuration file
├── .github/workflows/           # CI/CD workflows
├── .vscode/                     # VS Code workspace settings
├── Dockerfile                   # Container definition
├── docker-compose.yml           # Deployment configuration
├── go.mod                       # Go module dependencies
├── Makefile                     # Development tasks
├── .golangci.yml               # Linting configuration
├── .pre-commit-config.yaml     # Pre-commit hooks
├── .editorconfig               # Editor configuration
├── DEVELOPMENT.md              # Development guide
└── README.md                   # This file
```

## Features

- **Gensyn AI Node Monitoring**: Tracks training progress, model updates, and node participation
- **Log File Monitoring**: Tracks swarm.log, yarn.log, and wandb logs from Gensyn nodes
- **DHT Network Monitoring**: Monitors Hivemind DHT peer connections and model synchronization
- **Blockchain Monitoring**: Tracks smart contract events on Gensyn testnet (training submissions, rewards)
- **System Resource Monitoring**: Hardware metrics and Docker container monitoring for AI nodes
- **Health Check Endpoints**: REST API for Gensyn node health and metrics
- **Data Processing**: Aggregates and normalizes metrics data from distributed training
- **Alerting**: Generates alerts for critical events in the Gensyn AI network

## Quick Start

1. **Setup development environment:**
   ```bash
   make setup
   pre-commit install
   ```

2. **Build and run with Docker Compose:**
   ```bash
   make docker-run
   ```

3. **Run locally:**
   ```bash
   make run
   ```

4. **Access health endpoint:**
   ```bash
   curl http://localhost:8080/health
   ```

## Running Without Docker (Command Line Go)

You can run the GSwarm Sidecar directly from the command line using Go, without Docker. This is useful for development, debugging, or running on bare metal.

### Prerequisites
- Go 1.24 or later (https://golang.org/dl/)
- (Optional) [pre-commit](https://pre-commit.com/) for code quality hooks

### Setup
1. **Clone the repository and install dependencies:**
   ```bash
   git clone <repository-url>
   cd gswarm-sidecar
   go mod download
   ```
2. **Edit your configuration:**
   - Copy and edit `configs/config.yaml` to match your environment (log file paths, blockchain settings, API tokens, etc).

### Running the Sidecar

You can run the monitor directly with Go:
```bash
go run cmd/monitor/main.go
```

Or build a binary and run it:
```bash
go build -o gswarm-sidecar cmd/monitor/main.go
./gswarm-sidecar
```

### Stopping
- Press `Ctrl+C` to gracefully stop the sidecar.

### Notes
- The sidecar will read `configs/config.yaml` by default. To use a different config, set the `CONFIG_PATH` environment variable:
  ```bash
  CONFIG_PATH=/path/to/your_config.yaml go run cmd/monitor/main.go
  ```
- All logs will be printed to stdout by default.
- Make sure your Go version matches the required version in `go.mod` for best compatibility.

## Development Tools

This project uses several development tools to maintain code quality and consistency:

### Makefile Commands
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
- **golangci-lint**: Comprehensive Go linting with multiple linters
- **Pre-commit hooks**: Automated code quality checks before commits
- **EditorConfig**: Consistent coding style across editors
- **VS Code settings**: Optimized workspace configuration for Go development

### Continuous Integration
- **GitHub Actions**: Automated testing, linting, and building
- **Multi-version testing**: Tests against Go 1.24 and 1.25
- **Security scanning**: Automated security checks with gosec
- **Docker builds**: Automated Docker image building

For detailed development information, see [DEVELOPMENT.md](DEVELOPMENT.md).

## Configuration

Edit `configs/config.yaml` to customize:
- Log file paths for Gensyn AI node logs
- DHT bootstrap peers for the Gensyn network
- Blockchain contract details (Gensyn testnet)
- System monitoring intervals
- Storage locations for node metrics

## Gensyn AI Node Integration

This monitoring system is specifically designed to work with Gensyn AI nodes that:

- Participate in distributed training tasks
- Use the Hivemind DHT for peer-to-peer communication
- Submit training results to the Gensyn blockchain
- Run in containerized environments

The system monitors these nodes without requiring any modifications to the existing Gensyn AI codebase.

## Development

This is a bare bones structure. Each component (logs, dht, blockchain, system) can be implemented incrementally by focusing on one area at a time. The system is designed to be non-intrusive and work alongside existing Gensyn AI node deployments.

## License

See LICENSE file for details.

## Log File Monitoring and Central API Posting

This feature enables real-time monitoring of RL Swarm log files and posts key metrics/events to a central API endpoint for aggregation and analysis.

### Monitored Log Files
- `<RL Swarm root>/logs/swarm.log` — Main RL Swarm log (training progress, peer events, errors)
- `<RL Swarm root>/logs/yarn.log` — Modal login server log (authentication events, login issues)
- (Optional) `<RL Swarm root>/logs/wandb/debug.log` — Weights & Biases debug log (advanced metrics, debug info)

> **Note:** By default, only `swarm.log` and `yarn.log` are monitored, as Weights & Biases provides its own UI.

### How It Works
- The system tails the log files in real time.
- Each new line is parsed for relevant events/metrics.
- Extracted events are sent as JSON payloads to a central API endpoint.
- Handles network errors, retries (with backoff), and batching.
- Configuration is managed via `configs/config.yaml`.

### Example Event JSON Structure
```json
{
  "node_id": "node-123",
  "timestamp": "2024-06-07T12:34:56Z",
  "event_type": "training_progress",
  "details": {
    "epoch": 5,
    "accuracy": 0.92
  }
}
```

### Configuration (`configs/config.yaml`)
```yaml
api_endpoint: "https://central-api.example.com/gswarm/metrics"
auth_token: "YOUR_BEARER_TOKEN_HERE"  # Optional
batch_size: 10                          # Number of events to batch per POST
log_files:
  - "/path/to/rl-swarm/logs/swarm.log"
  - "/path/to/rl-swarm/logs/yarn.log"
  # - "/path/to/rl-swarm/logs/wandb/debug.log"  # Uncomment to enable
```

- `api_endpoint`: URL of the central API to receive metrics/events
- `auth_token`: Bearer token for authentication (optional)
- `batch_size`: Number of events to send in each POST (default: 10)
- `log_files`: List of log files to monitor

### Security
- If `auth_token` is set, an `Authorization: Bearer <token>` header is added to each request.
- Use HTTPS for secure transmission.

### Testing
- You can set `api_endpoint` to a mock server for local testing.
- All failed POSTs are logged and retried with exponential backoff.

### Example Go Posting Skeleton
```go
package logs

import (
    "bytes"
    "encoding/json"
    "fmt"
    "net/http"
    "time"
)

type MetricEvent struct {
    NodeID    string                 `json:"node_id"`
    Timestamp time.Time              `json:"timestamp"`
    EventType string                 `json:"event_type"`
    Details   map[string]interface{} `json:"details"`
}

func postMetricEvent(apiURL, authToken string, event MetricEvent) error {
    data, err := json.Marshal(event)
    if err != nil {
        return err
    }
    req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(data))
    if err != nil {
        return err
    }
    req.Header.Set("Content-Type", "application/json")
    if authToken != "" {
        req.Header.Set("Authorization", "Bearer "+authToken)
    }
    client := &http.Client{Timeout: 5 * time.Second}
    resp, err := client.Do(req)
    if err != nil {
        return err
    }
    defer resp.Body.Close()
    if resp.StatusCode >= 300 {
        return fmt.Errorf("API returned status %d", resp.StatusCode)
    }
    return nil
}
```
