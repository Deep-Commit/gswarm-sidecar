package logs

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
	// TODO: Implement log file monitoring
	// - Monitor swarm.log
	// - Monitor yarn.log
	// - Monitor wandb logs
	// - Send processed data via processor
	<-ctx.Done()
}
