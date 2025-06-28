package monitor

import (
	"context"
	"sync"

	"gswarm-sidecar/internal/blockchain"
	"gswarm-sidecar/internal/config"
	"gswarm-sidecar/internal/dht"
	"gswarm-sidecar/internal/logs"
	"gswarm-sidecar/internal/system"
)

type Monitor struct {
	cfg        *config.Config
	logs       *logs.Monitor
	dht        *dht.Monitor
	blockchain *blockchain.Monitor
	system     *system.Monitor

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

func New(cfg *config.Config) *Monitor {
	ctx, cancel := context.WithCancel(context.Background())

	return &Monitor{
		cfg:    cfg,
		ctx:    ctx,
		cancel: cancel,
	}
}

func (m *Monitor) Start() error {
	// Initialize components
	m.logs = logs.New(m.cfg)
	m.dht = dht.New(m.cfg)
	m.blockchain = blockchain.New(m.cfg)
	m.system = system.New(m.cfg)

	// Start monitoring components
	m.wg.Add(4)

	go func() {
		defer m.wg.Done()
		m.logs.Start(m.ctx)
	}()

	go func() {
		defer m.wg.Done()
		m.dht.Start(m.ctx)
	}()

	go func() {
		defer m.wg.Done()
		m.blockchain.Start(m.ctx)
	}()

	go func() {
		defer m.wg.Done()
		m.system.Start(m.ctx)
	}()

	return nil
}

func (m *Monitor) Stop() {
	m.cancel()
	m.wg.Wait()
}
