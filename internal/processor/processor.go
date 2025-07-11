package processor

import (
	"context"
	"fmt"
	"time"

	"gswarm-sidecar/internal/config"
	"gswarm-sidecar/internal/transmitter"
)

type Processor struct {
	transmitter *transmitter.Transmitter
	nodeID      string
	cfg         *config.Config
}

type LogMetrics struct {
	SwarmLogs []LogEntry `json:"swarm_logs"`
	YarnLogs  []LogEntry `json:"yarn_logs"`
	WandbLogs []LogEntry `json:"wandb_logs"`
}

type LogEntry struct {
	Timestamp time.Time `json:"timestamp"`
	Level     string    `json:"level"`
	Message   string    `json:"message"`
	Source    string    `json:"source"`
}

type DHTMetrics struct {
	PeerCount    int                    `json:"peer_count"`
	ActivePeers  []string               `json:"active_peers"`
	NetworkStats map[string]interface{} `json:"network_stats"`
}

type BlockchainMetrics struct {
	ContractEvents []ContractEvent `json:"contract_events"`
	GasUsed        uint64          `json:"gas_used"`
	BlockNumber    uint64          `json:"block_number"`

	Participation  uint64          `json:"participation"`
	TotalRewards   int64           `json:"total_rewards"`
	TotalWins      uint64          `json:"total_wins"`
}

type ContractEvent struct {
	EventType string                 `json:"event_type"`
	Timestamp time.Time              `json:"timestamp"`
	Data      map[string]interface{} `json:"data"`
	BlockHash string                 `json:"block_hash"`
	TxHash    string                 `json:"tx_hash"`
}

type SystemMetrics struct {
	CPU     CPUMetrics     `json:"cpu"`
	Memory  MemoryMetrics  `json:"memory"`
	Disk    DiskMetrics    `json:"disk"`
	Network NetworkMetrics `json:"network"`
}

type CPUMetrics struct {
	UsagePercent float64 `json:"usage_percent"`
	CoreCount    int     `json:"core_count"`
	Temperature  float64 `json:"temperature"`
}

type MemoryMetrics struct {
	Total        uint64  `json:"total"`
	Used         uint64  `json:"used"`
	Available    uint64  `json:"available"`
	UsagePercent float64 `json:"usage_percent"`
}

type DiskMetrics struct {
	Total        uint64  `json:"total"`
	Used         uint64  `json:"used"`
	Available    uint64  `json:"available"`
	UsagePercent float64 `json:"usage_percent"`
}

type NetworkMetrics struct {
	BytesSent       uint64 `json:"bytes_sent"`
	BytesReceived   uint64 `json:"bytes_received"`
	PacketsSent     uint64 `json:"packets_sent"`
	PacketsReceived uint64 `json:"packets_received"`
}

func New(transmitter *transmitter.Transmitter, nodeID string, cfg *config.Config) *Processor {
	return &Processor{
		transmitter: transmitter,
		nodeID:      nodeID,
		cfg:         cfg,
	}
}

func (p *Processor) ProcessLogs(ctx context.Context, metrics *LogMetrics) error {
	data := &transmitter.MetricsData{
		NodeID:      p.nodeID,
		Timestamp:   time.Now(),
		MetricsType: "logs",
		Data: map[string]interface{}{
			"swarm_logs": metrics.SwarmLogs,
			"yarn_logs":  metrics.YarnLogs,
			"wandb_logs": metrics.WandbLogs,
		},
	}

	err := p.transmitter.SendMetrics(ctx, data)
	if err != nil {
		return fmt.Errorf("failed to send log metrics: %w", err)
	}
	return nil
}

func (p *Processor) ProcessDHT(ctx context.Context, metrics *DHTMetrics) error {
	data := &transmitter.MetricsData{
		NodeID:      p.nodeID,
		Timestamp:   time.Now(),
		MetricsType: "dht",
		Data: map[string]interface{}{
			"peer_count":    metrics.PeerCount,
			"active_peers":  metrics.ActivePeers,
			"network_stats": metrics.NetworkStats,
		},
	}

	err := p.transmitter.SendMetrics(ctx, data)
	if err != nil {
		return fmt.Errorf("failed to send DHT metrics: %w", err)
	}
	return nil
}

func (p *Processor) ProcessBlockchain(ctx context.Context, metrics *BlockchainMetrics) error {
	data := &transmitter.MetricsData{
		NodeID:      p.nodeID,
		Timestamp:   time.Now(),
		MetricsType: "blockchain",
		Data: map[string]interface{}{
			"contract_events": metrics.ContractEvents,
			"gas_used":        metrics.GasUsed,
			"block_number":    metrics.BlockNumber,
			"participation":   metrics.Participation,
			"total_rewards":   metrics.TotalRewards,
			"total_wins":      metrics.TotalWins,
		},
	}

	err := p.transmitter.SendJSON(ctx, p.cfg.API.BlockchainLatestEndpoint, data, p.cfg.JWTToken)
	if err != nil {
		return fmt.Errorf("failed to send blockchain metrics: %w", err)
	}
	return nil
}

func (p *Processor) ProcessSystem(ctx context.Context, metrics *SystemMetrics) error {
	data := &transmitter.MetricsData{
		NodeID:      p.nodeID,
		Timestamp:   time.Now(),
		MetricsType: "system",
		Data: map[string]interface{}{
			"cpu":     metrics.CPU,
			"memory":  metrics.Memory,
			"disk":    metrics.Disk,
			"network": metrics.Network,
		},
	}

	err := p.transmitter.SendMetrics(ctx, data)
	if err != nil {
		return fmt.Errorf("failed to send system metrics: %w", err)
	}
	return nil
}

func (p *Processor) SendHealth(ctx context.Context, status, details string) error {
	data := &transmitter.HealthData{
		NodeID:    p.nodeID,
		Timestamp: time.Now(),
		Status:    status,
		Details:   details,
	}

	err := p.transmitter.SendHealth(ctx, data)
	if err != nil {
		return fmt.Errorf("failed to send health data: %w", err)
	}
	return nil
}
