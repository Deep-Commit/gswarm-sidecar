package logs

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
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
	offsetsFile      = "sidecar_offsets.json"
)

type fileOffsets map[string]int64

func loadOffsets() (fileOffsets, error) {
	data, err := ioutil.ReadFile(offsetsFile)
	if err != nil {
		if os.IsNotExist(err) {
			return make(fileOffsets), nil
		}
		return nil, err
	}
	var offsets fileOffsets
	if err := json.Unmarshal(data, &offsets); err != nil {
		return nil, err
	}
	return offsets, nil
}

func saveOffsets(offsets fileOffsets) error {
	data, err := json.MarshalIndent(offsets, "", "  ")
	if err != nil {
		return err
	}
	return ioutil.WriteFile(offsetsFile, data, 0644)
}

func New(cfg *config.Config, processor *processor.Processor) *Monitor {
	return &Monitor{
		cfg:       cfg,
		processor: processor,
	}
}

func (m *Monitor) Start(ctx context.Context) {
	offsets, err := loadOffsets()
	if err != nil {
		log.Printf("[ERROR] Failed to load offsets: %v", err)
		offsets = make(fileOffsets)
	}
	var wg sync.WaitGroup
	for _, logPath := range m.cfg.LogMonitoring.LogFiles {
		log.Printf("[INFO] Starting to tail log file: %s", logPath)
		wg.Add(1)
		go func(path string) {
			defer wg.Done()
			m.tailLogFileWithOffset(ctx, path, offsets)
		}(logPath)
	}
	wg.Wait()
}

// tailLogFile tails a log file and processes new lines in real time
func (m *Monitor) tailLogFile(ctx context.Context, path string) {
	t, err := tail.TailFile(path, tail.Config{Follow: true, ReOpen: true, Logger: tail.DiscardingLogger})
	if err != nil {
		log.Printf("[ERROR] Failed to tail log file %s: %v\n", path, err)
		return
	}
	log.Printf("[INFO] Successfully tailing log file: %s", path)
	batch := make([]MetricEvent, 0, m.cfg.LogMonitoring.BatchSize)
	for {
		select {
		case <-ctx.Done():
			log.Printf("[INFO] Context done, stopping tail for file: %s", path)
			return
		case line := <-t.Lines:
			if line == nil {
				log.Printf("[WARN] Received nil line from tail for file: %s", path)
				continue
			}
			log.Printf("[DEBUG] Read new line from %s: %s", path, line.Text)
			event := parseSwarmLogLine(line.Text, m.cfg)
			if event != nil {
				log.Printf("[DEBUG] Created MetricEvent: %+v", *event)
				batch = append(batch, *event)
				if len(batch) >= m.cfg.LogMonitoring.BatchSize {
					log.Printf("[INFO] Batch size reached (%d), sending batch", m.cfg.LogMonitoring.BatchSize)
					m.postBatch(ctx, batch)
					batch = batch[:0]
				}
			} else {
				log.Printf("[DEBUG] Skipped line (did not produce MetricEvent): %s", line.Text)
			}
		}
	}
}

// tailLogFileWithOffset tails a log file and processes new lines in real time, with offset tracking
func (m *Monitor) tailLogFileWithOffset(ctx context.Context, path string, offsets fileOffsets) {
	// Open file to get offset
	absPath, _ := filepath.Abs(path)
	var seekLine int64 = 0
	if off, ok := offsets[absPath]; ok {
		seekLine = off
		log.Printf("[INFO] Seeking to line %d in %s", seekLine, absPath)
	}

	t, err := tail.TailFile(path, tail.Config{Follow: true, ReOpen: true, Logger: tail.DiscardingLogger})
	if err != nil {
		log.Printf("[ERROR] Failed to tail log file %s: %v\n", path, err)
		return
	}
	log.Printf("[INFO] Successfully tailing log file: %s", path)
	batch := make([]MetricEvent, 0, m.cfg.LogMonitoring.BatchSize)
	lineNum := int64(0)
	// Skip lines up to seekLine
	for lineNum < seekLine {
		line := <-t.Lines
		if line == nil {
			continue
		}
		lineNum++
	}

	// Get batch flush interval from config, default to 10s if not set
	flushInterval := 10 * time.Second
	if m.cfg.LogMonitoring.BatchFlushInterval > 0 {
		flushInterval = time.Duration(m.cfg.LogMonitoring.BatchFlushInterval) * time.Second
	}
	flushTimer := time.NewTimer(flushInterval)
	defer flushTimer.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Printf("[INFO] Context done, stopping tail for file: %s", path)
			// Flush any remaining batch before exit
			if len(batch) > 0 {
				log.Printf("[INFO] Flushing remaining batch before exit for file: %s", path)
				m.postBatchWithOffset(ctx, batch, absPath, lineNum, offsets)
			}
			return
		case line := <-t.Lines:
			if line == nil {
				log.Printf("[WARN] Received nil line from tail for file: %s", path)
				continue
			}
			log.Printf("[DEBUG] Read new line from %s: %s", path, line.Text)
			lineNum++
			event := parseSwarmLogLine(line.Text, m.cfg)
			if event != nil {
				log.Printf("[DEBUG] Created MetricEvent: %+v", *event)
				batch = append(batch, *event)
				if len(batch) >= m.cfg.LogMonitoring.BatchSize {
					log.Printf("[INFO] Batch size reached (%d), sending batch", m.cfg.LogMonitoring.BatchSize)
					if m.postBatchWithOffset(ctx, batch, absPath, lineNum, offsets) {
						batch = batch[:0]
					}
					flushTimer.Reset(flushInterval)
				} else {
					// Reset timer on new line if batch not full
					flushTimer.Reset(flushInterval)
				}
			} else {
				log.Printf("[DEBUG] Skipped line (did not produce MetricEvent): %s", line.Text)
			}
		case <-flushTimer.C:
			if len(batch) > 0 {
				log.Printf("[INFO] Batch flush interval reached, sending batch of %d for file: %s", len(batch), path)
				if m.postBatchWithOffset(ctx, batch, absPath, lineNum, offsets) {
					batch = batch[:0]
				}
			}
			flushTimer.Reset(flushInterval)
		}
	}
}

// parseSwarmLogLine parses a line from swarm.log and returns a MetricEvent if relevant
func parseSwarmLogLine(line string, cfg *config.Config) *MetricEvent {
	parts := strings.SplitN(line, " - ", splitPartsFull)
	if len(parts) < splitPartsFull {
		log.Printf("[DEBUG] Line does not match expected format, sending as raw: %s", line)
		return &MetricEvent{
			NodeID:    cfg.NodeID,
			Timestamp: time.Now(),
			EventType: "raw",
			Details: map[string]interface{}{
				"raw_line": line,
			},
		}
	}
	ts, err := time.Parse("2006-01-02 15:04:05,000", parts[0])
	if err != nil {
		log.Printf("[WARN] Failed to parse timestamp, using current time. Line: %s, Error: %v", line, err)
		ts = time.Now()
	}
	level := strings.TrimSpace(parts[1])
	logger := strings.TrimSpace(parts[2])
	msg := strings.TrimSpace(parts[3])

	// Special case: peer join event
	if strings.Contains(msg, "Joining swarm with initial_peers") {
		peers := extractPeersFromLine(msg)
		log.Printf("[DEBUG] Detected peer join event. Peers: %v", peers)
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
		log.Printf("[DEBUG] Could not extract peers from line: %s", line)
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
		log.Printf("[ERROR] Failed to marshal batch: %v\n", err)
		return
	}

	// Debug: print the batch payload being sent
	log.Printf("[DEBUG] Sending batch payload: %s\n", string(data))

	apiURL := m.cfg.LogMonitoring.APIEndpoint
	authToken := m.cfg.JWTToken
	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewBuffer(data))
	if err != nil {
		log.Printf("[ERROR] Failed to create request: %v\n", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	if authToken != "" {
		req.Header.Set("Authorization", "Bearer "+authToken)
	}
	client := &http.Client{Timeout: batchPostTimeout}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("[ERROR] Failed to POST batch: %v\n", err)
		return
	}
	defer resp.Body.Close()

	// Debug: print the response status and body
	respBody, _ := io.ReadAll(resp.Body)
	log.Printf("[DEBUG] API response status: %d, body: %s\n", resp.StatusCode, string(respBody))

	if resp.StatusCode >= statusCodeError {
		log.Printf("[ERROR] API returned status %d\n", resp.StatusCode)
	} else {
		log.Printf("[INFO] Successfully posted batch of %d events", len(batch))
	}
}

// postBatchWithOffset posts a batch of MetricEvents to the API, with offset tracking
func (m *Monitor) postBatchWithOffset(ctx context.Context, batch []MetricEvent, absPath string, lineNum int64, offsets fileOffsets) bool {
	data, err := json.MarshalIndent(batch, "", "  ")
	if err != nil {
		log.Printf("[ERROR] Failed to marshal batch: %v\n", err)
		return false
	}

	log.Printf("[DEBUG] Sending batch payload: %s\n", string(data))

	apiURL := m.cfg.LogMonitoring.APIEndpoint
	authToken := m.cfg.JWTToken
	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewBuffer(data))
	if err != nil {
		log.Printf("[ERROR] Failed to create request: %v\n", err)
		return false
	}
	req.Header.Set("Content-Type", "application/json")
	if authToken != "" {
		req.Header.Set("Authorization", "Bearer "+authToken)
	}
	client := &http.Client{Timeout: batchPostTimeout}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("[ERROR] Failed to POST batch: %v\n", err)
		return false
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	log.Printf("[DEBUG] API response status: %d, body: %s\n", resp.StatusCode, string(respBody))

	if resp.StatusCode >= statusCodeError {
		log.Printf("[ERROR] API returned status %d\n", resp.StatusCode)
		return false
	} else {
		log.Printf("[INFO] Successfully posted batch of %d events", len(batch))
		offsets[absPath] = lineNum
		err := saveOffsets(offsets)
		if err != nil {
			log.Printf("[ERROR] Failed to save offsets: %v", err)
		}
		return true
	}
}
