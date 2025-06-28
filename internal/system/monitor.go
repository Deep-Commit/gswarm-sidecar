package system

import (
	"context"

	"gswarm-sidecar/internal/config"
	"gswarm-sidecar/internal/processor"
)

type Monitor struct {
	cfg       *config.Config
	processor *processor.Processor
}

func New(cfg *config.Config, processor *processor.Processor) *Monitor {
	return &Monitor{
		cfg:       cfg,
		processor: processor,
	}
}

func (m *Monitor) Start(ctx context.Context) {
	// TODO: Implement system monitoring
	// - Monitor hardware metrics
	// - Monitor Docker containers
	// - Health check endpoints
	// - Send processed data via processor
	<-ctx.Done()
}
