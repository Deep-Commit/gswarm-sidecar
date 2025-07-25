package logs

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
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

	"bufio"

	"regexp"

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
	maxNilLines      = 10 // Stop tailing after this many consecutive nil lines
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

	// Down detector state
	if m.cfg.Telegram.AlertOnDown && m.cfg.Telegram.BotToken != "" && m.cfg.Telegram.ChatID != "" {
		log.Printf("[INFO] Down detector with Telegram alerting enabled")
		lastEventTime := time.Now()
		alertSent := false
		delay := time.Duration(m.cfg.Telegram.DownAlertDelay) * time.Second
		if delay <= 0 {
			delay = 300 * time.Second // default 5 min
		}
		// Channel to receive log activity pings
		activityCh := make(chan struct{}, 1)
		// Start down detector goroutine
		go func() {
			for {
				select {
				case <-ctx.Done():
					return
				case <-activityCh:
					lastEventTime = time.Now()
					if alertSent {
						log.Printf("[INFO] Node activity resumed, resetting down alert state")
						alertSent = false
					}
				default:
					time.Sleep(2 * time.Second)
					if !alertSent && time.Since(lastEventTime) > delay {
						msg := fmt.Sprintf("[gswarm-sidecar] ALERT: Node '%s' appears DOWN. No log activity for %dm.", m.cfg.NodeID, int(delay.Minutes()))
						err := sendTelegramAlert(m.cfg.Telegram.BotToken, m.cfg.Telegram.ChatID, msg)
						if err != nil {
							log.Printf("[ERROR] Failed to send Telegram alert: %v", err)
						} else {
							log.Printf("[INFO] Sent Telegram down alert: %s", msg)
							alertSent = true
						}
					}
				}
			}
		}()
		// Wrap log file tailers to notify activityCh on new events
		for _, logPath := range m.cfg.LogMonitoring.LogFiles {
			log.Printf("[INFO] Starting to tail log file: %s", logPath)
			wg.Add(1)
			go func(path string) {
				defer wg.Done()
				m.tailLogFileWithOffsetAndActivity(ctx, path, offsets, activityCh)
			}(logPath)
		}
	} else {
		for _, logPath := range m.cfg.LogMonitoring.LogFiles {
			log.Printf("[INFO] Starting to tail log file: %s", logPath)
			wg.Add(1)
			go func(path string) {
				defer wg.Done()
				m.tailLogFileWithOffset(ctx, path, offsets)
			}(logPath)
		}
	}
	wg.Wait()
}

// tailLogFile tails a log file and processes new lines in real time
func (m *Monitor) tailLogFile(ctx context.Context, path string) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		log.Printf("[WARN] Log file does not exist: %s. Skipping tail for this file.", path)
		return
	}
	// Check if file is empty
	fi, err := os.Stat(path)
	if err == nil && fi.Size() == 0 {
		log.Printf("[WARN] Log file is empty: %s. Skipping tail for this file.", path)
		return
	}

	t, err := tail.TailFile(path, tail.Config{Follow: true, ReOpen: true, Logger: tail.DiscardingLogger})
	if err != nil {
		log.Printf("[ERROR] Failed to tail log file %s: %v\n", path, err)
		return
	}
	log.Printf("[INFO] Successfully tailing log file: %s", path)
	batch := make([]MetricEvent, 0, m.cfg.LogMonitoring.BatchSize)
	var nilLineWarned bool
	nilLineCount := 0
	for {
		select {
		case <-ctx.Done():
			log.Printf("[INFO] Context done, stopping tail for file: %s", path)
			return
		case line := <-t.Lines:
			if line == nil {
				nilLineCount++
				if !nilLineWarned {
					log.Printf("[WARN] Received nil line from tail for file: %s (will suppress further warnings)", path)
					nilLineWarned = true
				}
				if nilLineCount >= maxNilLines {
					log.Printf("[WARN] Too many nil lines from tail for file: %s. Stopping tail for this file.", path)
					return
				}
				continue
			}
			nilLineWarned = false // reset if we get a real line
			nilLineCount = 0
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
	absPath, _ := filepath.Abs(path)
	var seekLine int64 = 0
	if off, ok := offsets[absPath]; ok {
		seekLine = off
		log.Printf("[INFO] Seeking to line %d in %s", seekLine, absPath)
	} else {
		// No offset: only ingest last N lines
		n := m.cfg.LogMonitoring.InitialTailLines
		if n <= 0 {
			n = 100 // fallback default
		}
		// Count total lines in file
		file, err := os.Open(path)
		if err == nil {
			defer file.Close()
			total := int64(0)
			scanner := bufio.NewScanner(file)
			for scanner.Scan() {
				total++
			}
			if err := scanner.Err(); err == nil {
				seekLine = total - int64(n)
				if seekLine < 0 {
					seekLine = 0
				}
				log.Printf("[INFO] No offset found, will start ingesting from line %d (last %d lines of %d)", seekLine, n, total)
			}
		}
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
	// Scrub PII from all events before sending
	for i := range batch {
		scrubPII(&batch[i])
	}
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
	// Scrub PII from all events before sending
	for i := range batch {
		scrubPII(&batch[i])
	}
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

// sendTelegramAlert sends a message to the configured Telegram chat using the bot token and chat ID.
func sendTelegramAlert(botToken, chatID, message string) error {
	url := "https://api.telegram.org/bot" + botToken + "/sendMessage"
	payload := map[string]string{
		"chat_id": chatID,
		"text":    message,
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(data))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("telegram API error: %d %s", resp.StatusCode, string(body))
	}
	return nil
}

// Add a new tailLogFileWithOffsetAndActivity method
func (m *Monitor) tailLogFileWithOffsetAndActivity(ctx context.Context, path string, offsets fileOffsets, activityCh chan<- struct{}) {
	// Check if file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		log.Printf("[WARN] Log file does not exist: %s. Skipping tail for this file.", path)
		return
	}
	// Check if file is empty
	fi, err := os.Stat(path)
	if err == nil && fi.Size() == 0 {
		log.Printf("[WARN] Log file is empty: %s. Skipping tail for this file.", path)
		return
	}

	absPath, _ := filepath.Abs(path)
	var seekLine int64 = 0
	if off, ok := offsets[absPath]; ok {
		seekLine = off
		log.Printf("[INFO] Seeking to line %d in %s", seekLine, absPath)
	} else {
		n := m.cfg.LogMonitoring.InitialTailLines
		if n <= 0 {
			n = 100 // fallback default
		}
		file, err := os.Open(path)
		if err == nil {
			defer file.Close()
			total := int64(0)
			scanner := bufio.NewScanner(file)
			for scanner.Scan() {
				total++
			}
			if err := scanner.Err(); err == nil {
				seekLine = total - int64(n)
				if seekLine < 0 {
					seekLine = 0
				}
				log.Printf("[INFO] No offset found, will start ingesting from line %d (last %d lines of %d)", seekLine, n, total)
			}
		}
	}
	t, err := tail.TailFile(path, tail.Config{Follow: true, ReOpen: true, Logger: tail.DiscardingLogger})
	if err != nil {
		log.Printf("[ERROR] Failed to tail log file %s: %v\n", path, err)
		return
	}
	log.Printf("[INFO] Successfully tailing log file: %s", path)
	batch := make([]MetricEvent, 0, m.cfg.LogMonitoring.BatchSize)
	lineNum := int64(0)
	flushInterval := 10 * time.Second
	if m.cfg.LogMonitoring.BatchFlushInterval > 0 {
		flushInterval = time.Duration(m.cfg.LogMonitoring.BatchFlushInterval) * time.Second
	}
	flushTimer := time.NewTimer(flushInterval)
	defer flushTimer.Stop()
	// Skip lines up to seekLine
	for lineNum < seekLine {
		line := <-t.Lines
		if line == nil {
			continue
		}
		lineNum++
	}
	var nilLineWarned bool
	nilLineCount := 0
	for {
		select {
		case <-ctx.Done():
			log.Printf("[INFO] Context done, stopping tail for file: %s", path)
			if len(batch) > 0 {
				log.Printf("[INFO] Flushing remaining batch before exit for file: %s", path)
				m.postBatchWithOffset(ctx, batch, absPath, lineNum, offsets)
			}
			return
		case line := <-t.Lines:
			if line == nil {
				nilLineCount++
				if !nilLineWarned {
					log.Printf("[WARN] Received nil line from tail for file: %s (will suppress further warnings)", path)
					nilLineWarned = true
				}
				if nilLineCount >= maxNilLines {
					log.Printf("[WARN] Too many nil lines from tail for file: %s. Stopping tail for this file.", path)
					return
				}
				continue
			}
			nilLineWarned = false // reset if we get a real line
			nilLineCount = 0
			// Notify activity
			select { case activityCh <- struct{}{}: default: }
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

// --- PII Scrubber ---
// scrubPII redacts emails, IP addresses, environment settings, and wallet addresses from a MetricEvent (recursively)
func scrubPII(event *MetricEvent) {
	// Regex patterns
	emailRegex := regexp.MustCompile(`[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}`)
	ipRegex := regexp.MustCompile(`\b(?:[0-9]{1,3}\.){3}[0-9]{1,3}\b`)
	walletRegex := regexp.MustCompile(`0x[a-fA-F0-9]{40}`)
	envVarRegex := regexp.MustCompile(`(?i)(API_KEY|SECRET|PASSWORD|TOKEN|JWT|PRIVATE_KEY|ENV|CONFIG|DATABASE_URL|DB_PASS|ACCESS_KEY|SECRET_KEY)=[^\s]+`)
	// Serial numbers and device IDs (UUID, GUID, HWID, etc.)
	serialRegex := regexp.MustCompile(`(?i)(serial|device[_-]?id|uuid|guid|hwid|cpuid|gpuid)[\s:=]+[a-zA-Z0-9\-]{6,}`)
	// Generic long hex strings (potential device IDs)
	longHexRegex := regexp.MustCompile(`\b[a-fA-F0-9]{16,}\b`)

	event.Details = scrubMap(event.Details, emailRegex, ipRegex, walletRegex, envVarRegex, serialRegex, longHexRegex)
}

func scrubMap(m map[string]interface{}, regexes ...*regexp.Regexp) map[string]interface{} {
	for k, v := range m {
		switch val := v.(type) {
		case string:
			for _, re := range regexes {
				if re.MatchString(val) {
					val = re.ReplaceAllString(val, "[REDACTED]")
				}
			}
			m[k] = val
		case map[string]interface{}:
			m[k] = scrubMap(val, regexes...)
		case []interface{}:
			m[k] = scrubSlice(val, regexes...)
		}
	}
	return m
}

func scrubSlice(arr []interface{}, regexes ...*regexp.Regexp) []interface{} {
	for i, v := range arr {
		switch val := v.(type) {
		case string:
			for _, re := range regexes {
				if re.MatchString(val) {
					val = re.ReplaceAllString(val, "[REDACTED]")
				}
			}
			arr[i] = val
		case map[string]interface{}:
			arr[i] = scrubMap(val, regexes...)
		case []interface{}:
			arr[i] = scrubSlice(val, regexes...)
		}
	}
	return arr
}
// --- END PII Scrubber ---
