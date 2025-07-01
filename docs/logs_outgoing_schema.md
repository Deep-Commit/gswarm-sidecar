# Logs Module: Outgoing Data Structures

This document describes the **actual Go struct** used by the RL Swarm Sidecar logs module for outgoing metrics/events posted to the central API. Use this as the canonical reference for backend/API and UI development for the logs module only.

## Canonical Go Struct

```go
// internal/logs/monitor.go
// MetricEvent represents a parsed log event/metric
// (Extend Details as needed for your use case)
type MetricEvent struct {
    NodeID    string                 `json:"node_id"`
    Timestamp time.Time              `json:"timestamp"`
    EventType string                 `json:"event_type"`
    Details   map[string]interface{} `json:"details"`
}
```

### Field Descriptions
| Field      | Type                   | Description                                  |
|------------|------------------------|----------------------------------------------|
| node_id    | string                 | Unique identifier for the node               |
| timestamp  | RFC3339 string (time.Time) | Event time (RFC3339 format)             |
| event_type | string                 | Type of event/metric (see below)             |
| details    | map[string]interface{} | Event-specific data (see event types below)  |

- **All outgoing log events are serialized as JSON using this structure.**
- The `details` field is a flexible map and may contain different keys/values depending on the event type.

---

## Event Types and Example Details

(See below for event types and example payloads. Update as new event types are added.)

### 1. `training_progress`
```json
{
  "node_id": "node-123",
  "timestamp": "2024-06-07T12:34:56Z",
  "event_type": "training_progress",
  "details": {
    "epoch": 5,
    "accuracy": 0.92,
    "loss": 0.08
  }
}
```

### 2. `peer_event`
```json
{
  "node_id": "node-123",
  "timestamp": "2024-06-07T12:35:00Z",
  "event_type": "peer_event",
  "details": {
    "peer_id": "node-456",
    "action": "joined"
  }
}
```

### 3. `error`
```json
{
  "node_id": "node-123",
  "timestamp": "2024-06-07T12:36:00Z",
  "event_type": "error",
  "details": {
    "message": "Failed to connect to peer",
    "code": "PEER_CONN_FAIL"
  }
}
```

### 4. `auth_event`
```json
{
  "node_id": "node-123",
  "timestamp": "2024-06-07T12:37:00Z",
  "event_type": "auth_event",
  "details": {
    "user": "alice",
    "action": "login",
    "status": "success"
  }
}
```

---

- All timestamps are in RFC3339 format (e.g., `2024-06-07T12:34:56Z`).
- The `details` object may be extended with additional fields as new event types are added.
- For batching, the API may receive an array of these objects in a single POST.
- This document should be updated whenever new event types or fields are added.
