package transmitter

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"gswarm-sidecar/internal/config"
)

type Transmitter struct {
	cfg    *config.Config
	client *http.Client
}

type MetricsData struct {
	NodeID      string                 `json:"node_id"`
	Timestamp   time.Time              `json:"timestamp"`
	MetricsType string                 `json:"metrics_type"`
	Data        map[string]interface{} `json:"data"`
}

type HealthData struct {
	NodeID    string    `json:"node_id"`
	Timestamp time.Time `json:"timestamp"`
	Status    string    `json:"status"`
	Details   string    `json:"details"`
}

func New(cfg *config.Config) *Transmitter {
	client := &http.Client{
		Timeout: time.Duration(cfg.API.Timeout) * time.Second,
	}

	return &Transmitter{
		cfg:    cfg,
		client: client,
	}
}

func (t *Transmitter) SendMetrics(ctx context.Context, data *MetricsData) error {
	url := fmt.Sprintf("%s%s", t.cfg.API.BaseURL, t.cfg.API.MetricsEndpoint)

	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal metrics data: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if t.cfg.API.AuthToken != "" {
		req.Header.Set("Authorization", "Bearer "+t.cfg.API.AuthToken)
	}

	return t.sendWithRetry(req)
}

func (t *Transmitter) SendHealth(ctx context.Context, data *HealthData) error {
	url := fmt.Sprintf("%s%s", t.cfg.API.BaseURL, t.cfg.API.HealthEndpoint)

	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal health data: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if t.cfg.API.AuthToken != "" {
		req.Header.Set("Authorization", "Bearer "+t.cfg.API.AuthToken)
	}

	return t.sendWithRetry(req)
}

func (t *Transmitter) sendWithRetry(req *http.Request) error {
	var lastErr error

	for i := 0; i <= t.cfg.API.RetryCount; i++ {
		resp, err := t.client.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("request failed: %w", err)
			if i < t.cfg.API.RetryCount {
				time.Sleep(time.Duration(i+1) * time.Second)
				continue
			}
			return lastErr
		}
		defer resp.Body.Close()

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return nil
		}

		lastErr = fmt.Errorf("API returned status %d", resp.StatusCode)
		if i < t.cfg.API.RetryCount {
			time.Sleep(time.Duration(i+1) * time.Second)
			continue
		}
	}

	return lastErr
}
