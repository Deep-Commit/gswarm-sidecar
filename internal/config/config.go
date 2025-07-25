package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type TelegramConfig struct {
	BotToken       string `yaml:"bot_token"`
	ChatID         string `yaml:"chat_id"`
	AlertOnDown    bool   `yaml:"alert_on_down"`
	DownAlertDelay int    `yaml:"down_alert_delay"` // seconds
}

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
		ContractAddress   string `yaml:"contract_address"`
		RPCURL            string `yaml:"rpc_url"`
		ChainID           int64  `yaml:"chain_id"`
		ContractABIPath   string `yaml:"contract_abi_path"`
		PollInterval      int    `yaml:"poll_interval"` // in seconds
		SendInterval      int    `yaml:"send_interval"` // in seconds, for latest blockchain metrics
		NodeEOA           string `yaml:"node_eoa"`
		NodePeerID        string `yaml:"node_peer_id"`
		ContractABI       string // not mapped to yaml, loaded from file
	} `yaml:"blockchain"`

	System struct {
		MetricsInterval int  `yaml:"metrics_interval"`
		HealthPort      int  `yaml:"health_port"`
		PollInterval    int  `yaml:"poll_interval"` // Seconds, default 10
		EnableGPU       bool `yaml:"enable_gpu"`    // True if NVIDIA GPU present
		EnableCPU       bool `yaml:"enable_cpu"`    // Default true
		EnableRAM       bool `yaml:"enable_ram"`    // Default true
		BatchSize       int  `yaml:"batch_size"`    // Default 10
	} `yaml:"system"`

	Storage struct {
		DataPath string `yaml:"data_path"`
	} `yaml:"storage"`

	API struct {
		BaseURL                  string `yaml:"base_url"`
		MetricsEndpoint          string `yaml:"metrics_endpoint"`
		HealthEndpoint           string `yaml:"health_endpoint"`
		AuthToken                string `yaml:"auth_token"`
		Timeout                  int    `yaml:"timeout"`
		RetryCount               int    `yaml:"retry_count"`
		BlockchainLatestEndpoint string `yaml:"blockchain_latest_endpoint"`
	} `yaml:"api"`

	LogMonitoring struct {
		APIEndpoint        string   `yaml:"api_endpoint"`
		AuthToken          string   `yaml:"auth_token"`
		BatchSize          int      `yaml:"batch_size"`
		BatchFlushInterval int      `yaml:"batch_flush_interval"`
		LogFiles           []string `yaml:"log_files"`
		InitialTailLines   int      `yaml:"initial_tail_lines"`
	} `yaml:"log_monitoring"`

	NodeID   string `yaml:"node_id"`
	JWTToken string `yaml:"jwt_token"`

	Telegram TelegramConfig `yaml:"telegram"`
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

	if cfg.Blockchain.ContractABIPath != "" {
		abiBytes, err := os.ReadFile(cfg.Blockchain.ContractABIPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read contract ABI file: %w", err)
		}
		cfg.Blockchain.ContractABI = string(abiBytes)
	}

	// Set defaults for system monitoring if not specified
	if cfg.System.PollInterval == 0 {
		cfg.System.PollInterval = 10 // Default 10s
	}
	if cfg.System.BatchSize == 0 {
		cfg.System.BatchSize = 10 // Default batch size
	}
	if !cfg.System.EnableCPU {
		cfg.System.EnableCPU = true // Default true
	}
	if !cfg.System.EnableRAM {
		cfg.System.EnableRAM = true // Default true
	}

	return &cfg, nil
}
