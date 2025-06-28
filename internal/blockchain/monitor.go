package blockchain

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
	// TODO: Implement blockchain monitoring
	// - Connect to Gensyn testnet
	// - Monitor smart contract events
	// - Track training submissions
	// - Send processed data via processor
	<-ctx.Done()
}
