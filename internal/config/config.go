package config

import (
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
}

func Load() (*Config, error) {
	configPath := "configs/config.yaml"
	if os.Getenv("CONFIG_PATH") != "" {
		configPath = os.Getenv("CONFIG_PATH")
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}
