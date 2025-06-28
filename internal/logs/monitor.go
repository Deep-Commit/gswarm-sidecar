package logs

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
	// TODO: Implement log file monitoring
	// - Monitor swarm.log
	// - Monitor yarn.log
	// - Monitor wandb logs
	<-ctx.Done()
}
