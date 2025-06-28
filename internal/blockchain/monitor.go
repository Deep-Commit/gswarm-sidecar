package blockchain

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
	// TODO: Implement blockchain monitoring
	// - Connect to Gensyn testnet
	// - Monitor smart contract events
	// - Track training submissions
	<-ctx.Done()
}
