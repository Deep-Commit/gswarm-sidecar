package system

import (
	"context"
	"gswarm-sidecar/internal/config"
)

type Monitor struct {
	cfg *config.Config
}

func New(cfg *config.Config) *Monitor {
	return &Monitor{
		cfg: cfg,
	}
}

func (m *Monitor) Start(ctx context.Context) {
	// TODO: Implement system monitoring
	// - Monitor hardware metrics
	// - Monitor Docker containers
	// - Health check endpoints
	<-ctx.Done()
}
