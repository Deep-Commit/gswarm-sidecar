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
├── Dockerfile                   # Container definition
├── docker-compose.yml           # Deployment configuration
├── go.mod                       # Go module dependencies
└── README.md                    # This file
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

1. **Build and run with Docker Compose:**
   ```bash
   docker-compose up --build
   ```

2. **Run locally:**
   ```bash
   go mod download
   go run cmd/monitor/main.go
   ```

3. **Access health endpoint:**
   ```bash
   curl http://localhost:8080/health
   ```

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
