package monitor

import (
	"context"
	"sync"

	"gswarm-sidecar/internal/blockchain"
	"gswarm-sidecar/internal/config"
	"gswarm-sidecar/internal/dht"
	"gswarm-sidecar/internal/logs"
	"gswarm-sidecar/internal/processor"
	"gswarm-sidecar/internal/system"
	"gswarm-sidecar/internal/transmitter"
)

const numMonitors = 4

type Monitor struct {
	cfg         *config.Config
	logs        *logs.Monitor
	dht         *dht.Monitor
	blockchain  *blockchain.Monitor
	system      *system.Monitor
	processor   *processor.Processor
	transmitter *transmitter.Transmitter

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
	// Initialize transmitter and processor
	m.transmitter = transmitter.New(m.cfg)
	m.processor = processor.New(m.transmitter, "gensyn-node-001") // TODO: Get actual node ID

	// Initialize monitoring components
	m.logs = logs.New(m.cfg, m.processor)
	m.dht = dht.New(m.cfg, m.processor)
	m.blockchain = blockchain.New(m.cfg, m.processor)
	m.system = system.New(m.cfg, m.processor)

	// Start monitoring components
	m.wg.Add(numMonitors)

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
