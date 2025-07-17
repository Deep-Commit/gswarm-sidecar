package system

import (
	"context"
	"log"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/load"
	"github.com/shirou/gopsutil/mem"

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
	log.Println("Starting hardware monitoring...")

	// Start hardware monitoring goroutine
	go m.startHardwareMonitor(ctx)

	// TODO: Implement other system monitoring
	// - Monitor Docker containers
	// - Health check endpoints

	<-ctx.Done()
	log.Println("Hardware monitoring stopped")
}

func (m *Monitor) startHardwareMonitor(ctx context.Context) {
	ticker := time.NewTicker(time.Duration(m.cfg.System.PollInterval) * time.Second)
	defer ticker.Stop()

	var batch []map[string]interface{}

	for {
		select {
		case <-ctx.Done():
			// Send any remaining batch before shutting down
			if len(batch) > 0 {
				m.sendHardwareBatch(batch)
			}
			return
		case <-ticker.C:
			metrics := m.collectHardwareMetrics()
			if metrics != nil {
				event := map[string]interface{}{
					"type":      "hardware_snapshot",
					"timestamp": time.Now().UTC().Format(time.RFC3339),
					"node_id":   m.cfg.NodeID,
					"metrics":   metrics,
				}

				batch = append(batch, event)

				if len(batch) >= m.cfg.System.BatchSize {
					m.sendHardwareBatch(batch)
					batch = nil
				}
			}
		}
	}
}

func (m *Monitor) collectHardwareMetrics() map[string]interface{} {
	metrics := make(map[string]interface{})

	// Collect CPU metrics
	if m.cfg.System.EnableCPU {
		if cpuMetrics := m.collectCPUMetrics(); cpuMetrics != nil {
			metrics["cpu"] = cpuMetrics
		}
	}

	// Collect RAM metrics
	if m.cfg.System.EnableRAM {
		if ramMetrics := m.collectRAMMetrics(); ramMetrics != nil {
			metrics["ram"] = ramMetrics
		}
	}

	// Collect GPU metrics
	if m.cfg.System.EnableGPU {
		if gpuMetrics := m.collectGPUMetrics(); len(gpuMetrics) > 0 {
			metrics["gpu"] = gpuMetrics
		}
	}

	if len(metrics) == 0 {
		return nil
	}
	return metrics
}

func (m *Monitor) collectCPUMetrics() map[string]interface{} {
	// Get CPU percentage
	cpuPercent, err := cpu.Percent(0, false)
	if err != nil {
		log.Printf("Failed to get CPU percentage: %v", err)
		return nil
	}

	// Get CPU info for core count
	cpuInfo, err := cpu.Info()
	if err != nil {
		log.Printf("Failed to get CPU info: %v", err)
		return nil
	}

	// Get load average
	loadAvg, err := load.Avg()
	if err != nil {
		log.Printf("Failed to get load average: %v", err)
		return nil
	}

	return map[string]interface{}{
		"percent":   cpuPercent[0],
		"cores":     len(cpuInfo),
		"load_avg":  []float64{loadAvg.Load1, loadAvg.Load5, loadAvg.Load15},
	}
}

func (m *Monitor) collectRAMMetrics() map[string]interface{} {
	// Get virtual memory stats
	vm, err := mem.VirtualMemory()
	if err != nil {
		log.Printf("Failed to get virtual memory stats: %v", err)
		return nil
	}

	// Get swap memory stats
	swap, err := mem.SwapMemory()
	if err != nil {
		log.Printf("Failed to get swap memory stats: %v", err)
		// Continue without swap data
	}

	metrics := map[string]interface{}{
		"total_mb":      vm.Total / 1024 / 1024,
		"used_mb":       vm.Used / 1024 / 1024,
		"available_mb":  vm.Available / 1024 / 1024,
		"percent_used":  vm.UsedPercent,
	}

	if swap != nil {
		metrics["swap_total_mb"] = swap.Total / 1024 / 1024
		metrics["swap_used_mb"] = swap.Used / 1024 / 1024
		metrics["swap_percent_used"] = swap.UsedPercent
	}

	return metrics
}

func (m *Monitor) collectGPUMetrics() []map[string]interface{} {
	var gpuMetrics []map[string]interface{}

	// Try to run nvidia-smi to get GPU metrics
	cmd := exec.Command("nvidia-smi", "--query-gpu=utilization.gpu,temperature.gpu,memory.used,memory.total", "--format=csv,noheader")
	output, err := cmd.Output()
	if err != nil {
		// GPU not available or nvidia-smi not installed, skip silently
		return gpuMetrics
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for i, line := range lines {
		if line == "" {
			continue
		}

		parts := strings.Split(line, ", ")
		if len(parts) == 4 {
			util, _ := strconv.ParseFloat(strings.TrimSuffix(parts[0], " %"), 64)
			temp, _ := strconv.ParseFloat(parts[1], 64)
			used, _ := strconv.ParseFloat(strings.TrimSuffix(parts[2], " MiB"), 64)
			total, _ := strconv.ParseFloat(strings.TrimSuffix(parts[3], " MiB"), 64)

			gpuMetrics = append(gpuMetrics, map[string]interface{}{
				"index":           i,
				"util_percent":    util,
				"temp_c":          temp,
				"vram_used_mb":    used,
				"vram_total_mb":   total,
			})
		}
	}

	return gpuMetrics
}

func (m *Monitor) sendHardwareBatch(batch []map[string]interface{}) {
	if len(batch) == 0 {
		return
	}

	// Create hardware metrics structure
	hardwareMetrics := &processor.HardwareMetrics{}

	// Process the batch to extract metrics
	for _, event := range batch {
		if metrics, ok := event["metrics"].(map[string]interface{}); ok {
			// Extract CPU metrics from the first event (they should be consistent)
			if cpuData, ok := metrics["cpu"].(map[string]interface{}); ok {
				if percent, ok := cpuData["percent"].(float64); ok {
					hardwareMetrics.CPU.UsagePercent = percent
				}
				if cores, ok := cpuData["cores"].(int); ok {
					hardwareMetrics.CPU.CoreCount = cores
				}
			}

			// Extract RAM metrics
			if ramData, ok := metrics["ram"].(map[string]interface{}); ok {
				if total, ok := ramData["total_mb"].(uint64); ok {
					hardwareMetrics.RAM.Total = total * 1024 * 1024 // Convert back to bytes
				}
				if used, ok := ramData["used_mb"].(uint64); ok {
					hardwareMetrics.RAM.Used = used * 1024 * 1024 // Convert back to bytes
				}
				if percent, ok := ramData["percent_used"].(float64); ok {
					hardwareMetrics.RAM.UsagePercent = percent
				}
				// Extract swap memory data
				if swapTotal, ok := ramData["swap_total_mb"].(uint64); ok {
					hardwareMetrics.RAM.SwapTotal = swapTotal * 1024 * 1024 // Convert back to bytes
				}
				if swapUsed, ok := ramData["swap_used_mb"].(uint64); ok {
					hardwareMetrics.RAM.SwapUsed = swapUsed * 1024 * 1024 // Convert back to bytes
				}
				if swapPercent, ok := ramData["swap_percent_used"].(float64); ok {
					hardwareMetrics.RAM.SwapPercent = swapPercent
				}
			}

			// Extract GPU metrics
			if gpuData, ok := metrics["gpu"].([]map[string]interface{}); ok {
				for _, gpu := range gpuData {
					if index, ok := gpu["index"].(int); ok {
						gpuMetric := processor.GPUMetrics{Index: index}

						if util, ok := gpu["util_percent"].(float64); ok {
							gpuMetric.UtilPercent = util
						}
						if temp, ok := gpu["temp_c"].(float64); ok {
							gpuMetric.TempC = temp
						}
						if vramUsed, ok := gpu["vram_used_mb"].(float64); ok {
							gpuMetric.VRAMUsedMB = vramUsed
						}
						if vramTotal, ok := gpu["vram_total_mb"].(float64); ok {
							gpuMetric.VRAMTotalMB = vramTotal
						}

						hardwareMetrics.GPU = append(hardwareMetrics.GPU, gpuMetric)
					}
				}
			}
		}
	}

	// Send the hardware metrics
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := m.processor.ProcessHardware(ctx, hardwareMetrics)
	if err != nil {
		log.Printf("Failed to send hardware metrics: %v", err)
	} else {
		log.Printf("Sent hardware metrics batch with %d events", len(batch))
	}
}
