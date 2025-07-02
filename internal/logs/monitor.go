package logs

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"gswarm-sidecar/internal/config"
	"gswarm-sidecar/internal/processor"

	"github.com/hpcloud/tail"
)

type Monitor struct {
	cfg       *config.Config
	processor *processor.Processor
}

// MetricEvent represents a parsed log event/metric
// (Extend Details as needed for your use case)
type MetricEvent struct {
	NodeID    string                 `json:"node_id"`
	Timestamp time.Time              `json:"timestamp"`
	EventType string                 `json:"event_type"`
	Details   map[string]interface{} `json:"details"`
}

const (
	splitPartsFull   = 4
	splitPartsShort  = 2
	batchPostTimeout = 5 * time.Second
	statusCodeError  = 300
)

func New(cfg *config.Config, processor *processor.Processor) *Monitor {
	return &Monitor{
		cfg:       cfg,
		processor: processor,
	}
}

func (m *Monitor) Start(ctx context.Context) {
	var wg sync.WaitGroup
	for _, logPath := range m.cfg.LogMonitoring.LogFiles {
		wg.Add(1)
		go func(path string) {
			defer wg.Done()
			m.tailLogFile(ctx, path)
		}(logPath)
	}
	wg.Wait()
}

// tailLogFile tails a log file and processes new lines in real time
func (m *Monitor) tailLogFile(ctx context.Context, path string) {
	t, err := tail.TailFile(path, tail.Config{Follow: true, ReOpen: true, Logger: tail.DiscardingLogger})
	if err != nil {
		log.Printf("Failed to tail log file %s: %v\n", path, err)
		return
	}
	batch := make([]MetricEvent, 0, m.cfg.LogMonitoring.BatchSize)
	for {
		select {
		case <-ctx.Done():
			return
		case line := <-t.Lines:
			if line == nil {
				continue
			}
			event := parseSwarmLogLine(line.Text, m.cfg)
			if event != nil {
				batch = append(batch, *event)
				if len(batch) >= m.cfg.LogMonitoring.BatchSize {
					m.postBatch(ctx, batch)
					batch = batch[:0]
				}
			}
		}
	}
}

// parseSwarmLogLine parses a line from swarm.log and returns a MetricEvent if relevant
func parseSwarmLogLine(line string, cfg *config.Config) *MetricEvent {
	parts := strings.SplitN(line, " - ", splitPartsFull)
	if len(parts) < splitPartsFull {
		return nil // skip lines that don't match expected format
	}
	ts, err := time.Parse("2006-01-02 15:04:05,000", parts[0])
	if err != nil {
		ts = time.Now()
	}
	level := strings.TrimSpace(parts[1])
	logger := strings.TrimSpace(parts[2])
	msg := strings.TrimSpace(parts[3])

	// Special case: peer join event
	if strings.Contains(msg, "Joining swarm with initial_peers") {
		peers := extractPeersFromLine(msg)
		return &MetricEvent{
			NodeID:    cfg.NodeID,
			Timestamp: ts,
			EventType: "peer_event",
			Details: map[string]interface{}{
				"action": "join",
				"peers":  peers,
				"logger": logger,
				"raw":    msg,
			},
		}
	}

	// General case: emit an event for every log line
	eventType := strings.ToLower(level)
	switch eventType {
	case "error":
		eventType = "error"
	case "info":
		eventType = "info"
	case "debug":
		eventType = "debug"
		// extend as needed
	}

	return &MetricEvent{
		NodeID:    cfg.NodeID,
		Timestamp: ts,
		EventType: eventType,
		Details: map[string]interface{}{
			"logger":  logger,
			"message": msg,
		},
	}
}

// extractPeersFromLine extracts peer addresses from a log line
func extractPeersFromLine(line string) []string {
	start := strings.Index(line, "[")
	end := strings.Index(line, "]")
	if start == -1 || end == -1 || end <= start {
		return nil
	}
	peersStr := line[start+1 : end]
	peers := strings.Split(peersStr, ", ")
	for i := range peers {
		peers[i] = strings.Trim(peers[i], "' ")
	}
	return peers
}

// postBatch posts a batch of MetricEvents to the API
func (m *Monitor) postBatch(ctx context.Context, batch []MetricEvent) {
	data, err := json.MarshalIndent(batch, "", "  ")
	if err != nil {
		log.Printf("Failed to marshal batch: %v\n", err)
		return
	}

	apiURL := m.cfg.LogMonitoring.APIEndpoint
	authToken := m.cfg.JWTToken
	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewBuffer(data))
	if err != nil {
		log.Printf("Failed to create request: %v\n", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	if authToken != "" {
		req.Header.Set("Authorization", "Bearer "+authToken)
	}
	client := &http.Client{Timeout: batchPostTimeout}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Failed to POST batch: %v\n", err)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode >= statusCodeError {
		log.Printf("API returned status %d\n", resp.StatusCode)
	}
}
