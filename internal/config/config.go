package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Logs struct {
		SwarmLogPath string `yaml:"swarm_log_path"`
		YarnLogPath  string `yaml:"yarn_log_path"`
		WandbLogPath string `yaml:"wandb_log_path"`
	} `yaml:"logs"`

	DHT struct {
		BootstrapPeers []string `yaml:"bootstrap_peers"`
		Port           int      `yaml:"port"`
	} `yaml:"dht"`

	Blockchain struct {
		ContractAddress string `yaml:"contract_address"`
		RPCURL          string `yaml:"rpc_url"`
		ChainID         int64  `yaml:"chain_id"`
	} `yaml:"blockchain"`

	System struct {
		MetricsInterval int `yaml:"metrics_interval"`
		HealthPort      int `yaml:"health_port"`
	} `yaml:"system"`

	Storage struct {
		DataPath string `yaml:"data_path"`
	} `yaml:"storage"`

	API struct {
		BaseURL         string `yaml:"base_url"`
		MetricsEndpoint string `yaml:"metrics_endpoint"`
		HealthEndpoint  string `yaml:"health_endpoint"`
		AuthToken       string `yaml:"auth_token"`
		Timeout         int    `yaml:"timeout"`
		RetryCount      int    `yaml:"retry_count"`
	} `yaml:"api"`

	LogMonitoring struct {
		APIEndpoint string   `yaml:"api_endpoint"`
		AuthToken   string   `yaml:"auth_token"`
		BatchSize   int      `yaml:"batch_size"`
		LogFiles    []string `yaml:"log_files"`
		BatchFlushInterval int `yaml:"batch_flush_interval"`
	} `yaml:"log_monitoring"`

	NodeID string `yaml:"node_id"`
	JWTToken string `yaml:"jwt_token"`
}

func Load() (*Config, error) {
	configPath := "configs/config.yaml"
	if os.Getenv("CONFIG_PATH") != "" {
		configPath = os.Getenv("CONFIG_PATH")
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &cfg, nil
}
