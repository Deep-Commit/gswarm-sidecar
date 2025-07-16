# GSwarm Side Car Monitoring System

**Easily monitor your Gensyn AI nodes—no coding required!**

This tool helps you track the health and performance of your Gensyn AI nodes. Just update a simple config file, then launch the monitor with a single command. No need to change your existing node setup or write any code.

---

## Quick Start for Beginners

**1. Get Your JWT Token**

To use the monitor, you need a JWT token from the GSwarm dashboard:

- Go to [https://gswarm.dev](https://gswarm.dev)
- Connect your Metamask wallet (you'll see a prompt on the site)
- After logging in, go to your dashboard/settings and copy your JWT token

**2. Edit the Configuration File**

Before running the monitor, tell it where to find your log files and where to send the data.

- Open the file: `configs/config.yaml`
- Paste your JWT token into the `jwt_token` field:

```yaml
jwt_token: "YOUR_JWT_TOKEN_HERE"
```
- Update these lines to match your setup:

```yaml
log_monitoring:
  api_endpoint: "https://central-api.example.com/gswarm/metrics"  # Where to send metrics
  log_files:
    - "/path/to/rl-swarm/logs/swarm.log"   # Main log file
    - "/path/to/rl-swarm/logs/yarn.log"    # Login server log
#   - "/path/to/rl-swarm/logs/wandb/debug.log"  # (Optional) Advanced metrics
```

**3. Start the Monitor**

If you have Docker installed (recommended for easiest setup):

```bash
docker compose up
```

Or, if you prefer to run it directly (requires Go):

```bash
make run
```

**4. How to Know It's Working**

- The monitor will print logs to your terminal window.
- If you see messages about reading your log files and sending data, it's working!
- You can also check your dashboard at [https://gswarm.dev](https://gswarm.dev) to see if your node is reporting.

---

## What Does This Do?

- Watches your Gensyn AI node log files in real time
- Sends important events and metrics to a central dashboard (API)
- Helps you keep track of node health, training progress, and more
- No changes needed to your Gensyn AI node code

---

## Need Help?

- Double-check your log file paths in `configs/config.yaml`
- Make sure Docker is installed (for easiest setup)
- For advanced help, see the [DEVELOPMENT.md](DEVELOPMENT.md) or ask your Gensyn support contact

---

## Advanced: Development & Customization

If you want to build, test, or develop the monitor yourself, see below for more details.

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
