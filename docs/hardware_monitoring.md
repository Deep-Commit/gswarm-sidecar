# Hardware Monitoring

The gswarm-sidecar now includes comprehensive hardware monitoring capabilities that collect CPU, RAM, and GPU metrics from your nodes.

## Features

### CPU Monitoring
- **Usage Percentage**: Real-time CPU utilization
- **Core Count**: Number of CPU cores
- **Load Average**: 1, 5, and 15-minute load averages

### RAM Monitoring
- **Total Memory**: Total available RAM in MB
- **Used Memory**: Currently used RAM in MB
- **Available Memory**: Free RAM in MB
- **Usage Percentage**: RAM utilization percentage
- **Swap Memory**: Total, used, and percentage of swap space (if available)

### GPU Monitoring (NVIDIA)
- **GPU Utilization**: GPU usage percentage
- **Temperature**: GPU temperature in Celsius
- **VRAM Usage**: Used and total VRAM in MB
- **Multi-GPU Support**: Automatically detects and monitors multiple GPUs

## Configuration

Add the following section to your `configs/config.yaml`:

```yaml
system:
  poll_interval: 10      # Polling interval in seconds (default: 10)
  enable_gpu: true       # Enable GPU monitoring (default: true)
  enable_cpu: true       # Enable CPU monitoring (default: true)
  enable_ram: true       # Enable RAM monitoring (default: true)
  batch_size: 10         # Number of metrics to batch before sending (default: 10)
```

## Data Format

Hardware metrics are sent as JSON events with the following structure:

```json
{
  "type": "hardware_snapshot",
  "timestamp": "2024-01-01T12:00:00Z",
  "node_id": "my-node-123",
  "metrics": {
    "cpu": {
      "percent": 45.2,
      "cores": 8,
      "load_avg": [1.2, 1.1, 1.0]
    },
    "ram": {
      "total_mb": 16384,
      "used_mb": 8192,
      "available_mb": 8192,
      "percent_used": 50.0,
      "swap_total_mb": 4096,
      "swap_used_mb": 1024,
      "swap_percent_used": 25.0
    },
    "gpu": [
      {
        "index": 0,
        "util_percent": 78.5,
        "temp_c": 65.0,
        "vram_used_mb": 6144,
        "vram_total_mb": 8192
      }
    ]
  }
}
```

**Note**: The wallet address is extracted from the JWT token in the Authorization header, so it's not included in the metrics payload.

## Requirements

### CPU and RAM Monitoring
- Uses `gopsutil` library (automatically installed)
- Works on Linux, macOS, and Windows

### GPU Monitoring
- Requires NVIDIA GPU with `nvidia-smi` installed
- Automatically detects if GPU monitoring is available
- Gracefully handles systems without NVIDIA GPUs

## API Integration

Hardware metrics are sent to the configured API endpoint using the existing JWT authentication and retry mechanisms. The metrics are batched and sent when the batch size is reached or when the monitoring stops.

**Authentication**: The JWT token in the Authorization header contains the wallet address, which your API can extract and use for data association.

## Use Cases

### Performance Optimization
- Monitor CPU usage to optimize workload distribution
- Track RAM utilization to prevent memory bottlenecks
- Monitor GPU temperature to prevent thermal throttling

### Resource Management
- Identify underutilized resources
- Plan capacity based on actual usage patterns
- Optimize for cost-effective resource allocation

### Health Monitoring
- Detect hardware issues early
- Monitor thermal performance
- Track resource degradation over time

## Troubleshooting

### GPU Monitoring Not Working
1. Ensure `nvidia-smi` is installed and accessible
2. Check that NVIDIA drivers are properly installed
3. Verify GPU is detected by running `nvidia-smi` manually

### High CPU Usage
- The monitoring itself uses minimal resources
- Polling interval can be increased to reduce overhead
- Consider disabling unused metrics (GPU on CPU-only systems)

### API Connection Issues
- Verify JWT token is valid
- Check network connectivity to API endpoint
- Review retry configuration in main config
