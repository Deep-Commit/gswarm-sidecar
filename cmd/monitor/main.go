package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"gswarm-sidecar/internal/config"
	"gswarm-sidecar/internal/monitor"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Initialize monitor
	monitor := monitor.New(cfg)

	// Start monitoring
	if err := monitor.Start(); err != nil {
		log.Fatalf("Failed to start monitor: %v", err)
	}

	// Wait for shutdown signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	// Graceful shutdown
	monitor.Stop()
}
