package config

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
	"time"
)

// AgentConfig contains runtime settings for the metrics collection agent.
type AgentConfig struct {
	ServerAddress  string
	GRPCAddress    string
	PollInterval   time.Duration
	ReportInterval time.Duration
	Key            string
	CryptoKey      string
	RateLimit      int
}

type agentFileConfig struct {
	Address        *string `json:"address"`
	GRPCAddress    *string `json:"grpc_address"`
	ReportInterval *string `json:"report_interval"`
	PollInterval   *string `json:"poll_interval"`
	Key            *string `json:"key"`
	CryptoKey      *string `json:"crypto_key"`
	RateLimit      *int    `json:"rate_limit"`
}

// NewAgentConfig returns the default agent configuration.
func NewAgentConfig() AgentConfig {
	return AgentConfig{
		ServerAddress:  "localhost:8080",
		PollInterval:   2 * time.Second,
		ReportInterval: 10 * time.Second,
		RateLimit:      1,
	}
}

// ParseAgentConfig reads the agent config file, flags, and environment variables.
// Environment variables override flags, and flags override values from the file.
func ParseAgentConfig(args []string) (AgentConfig, error) {
	probe := NewAgentConfig()
	configPath, err := parseAgentFlags(&probe, args)
	if err != nil {
		return AgentConfig{}, err
	}
	if value, ok := os.LookupEnv("CONFIG"); ok {
		configPath = value
	}

	cfg := NewAgentConfig()
	if configPath != "" {
		if err := loadAgentFile(configPath, &cfg); err != nil {
			return AgentConfig{}, err
		}
	}

	if _, err := parseAgentFlags(&cfg, args); err != nil {
		return AgentConfig{}, err
	}

	if value, ok, err := lookupEnvDurationSeconds("REPORT_INTERVAL"); err != nil {
		return AgentConfig{}, fmt.Errorf("invalid REPORT_INTERVAL value: %w", err)
	} else if ok {
		cfg.ReportInterval = value
	}

	if value, ok, err := lookupEnvDurationSeconds("POLL_INTERVAL"); err != nil {
		return AgentConfig{}, fmt.Errorf("invalid POLL_INTERVAL value: %w", err)
	} else if ok {
		cfg.PollInterval = value
	}

	if value, ok, err := lookupEnvInt("RATE_LIMIT"); err != nil {
		return AgentConfig{}, fmt.Errorf("invalid RATE_LIMIT value: %w", err)
	} else if ok {
		cfg.RateLimit = value
	}

	if value, ok := os.LookupEnv("ADDRESS"); ok {
		cfg.ServerAddress = value
	}

	if value, ok := os.LookupEnv("GRPC_ADDRESS"); ok {
		cfg.GRPCAddress = value
	}

	if value, ok := os.LookupEnv("KEY"); ok {
		cfg.Key = value
	}

	if value, ok := os.LookupEnv("CRYPTO_KEY"); ok {
		cfg.CryptoKey = value
	}

	if cfg.ReportInterval < 0 {
		return AgentConfig{}, fmt.Errorf("invalid report interval value %d: interval must be non-negative seconds", int(cfg.ReportInterval/time.Second))
	}

	if cfg.PollInterval < 0 {
		return AgentConfig{}, fmt.Errorf("invalid poll interval value %d: interval must be non-negative seconds", int(cfg.PollInterval/time.Second))
	}

	if cfg.RateLimit <= 0 {
		return AgentConfig{}, fmt.Errorf("invalid rate limit value %d: value must be positive", cfg.RateLimit)
	}

	return cfg, nil
}

func parseAgentFlags(cfg *AgentConfig, args []string) (string, error) {
	var reportSeconds int
	var pollSeconds int
	var configPath string

	fs := flag.NewFlagSet("agent", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.StringVar(&cfg.ServerAddress, "a", cfg.ServerAddress, "HTTP server endpoint address")
	fs.StringVar(&cfg.GRPCAddress, "grpc-address", cfg.GRPCAddress, "gRPC server endpoint address")
	fs.IntVar(&reportSeconds, "r", int(cfg.ReportInterval/time.Second), "report interval in seconds")
	fs.IntVar(&pollSeconds, "p", int(cfg.PollInterval/time.Second), "poll interval in seconds")
	fs.StringVar(&cfg.Key, "k", cfg.Key, "hash signature key")
	fs.StringVar(&cfg.CryptoKey, "crypto-key", cfg.CryptoKey, "path to the public encryption key")
	fs.IntVar(&cfg.RateLimit, "l", cfg.RateLimit, "maximum number of concurrent outgoing requests")
	fs.StringVar(&configPath, "c", configPath, "path to the JSON configuration file")
	fs.StringVar(&configPath, "config", configPath, "path to the JSON configuration file")

	if err := fs.Parse(args); err != nil {
		return "", err
	}

	fs.Visit(func(f *flag.Flag) {
		switch f.Name {
		case "r":
			cfg.ReportInterval = time.Duration(reportSeconds) * time.Second
		case "p":
			cfg.PollInterval = time.Duration(pollSeconds) * time.Second
		}
	})

	return configPath, nil
}

func loadAgentFile(path string, cfg *AgentConfig) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read agent config file %q: %w", path, err)
	}

	var fileCfg agentFileConfig
	if err := json.Unmarshal(data, &fileCfg); err != nil {
		return fmt.Errorf("parse agent config file %q: %w", path, err)
	}

	if fileCfg.Address != nil {
		cfg.ServerAddress = *fileCfg.Address
	}
	if fileCfg.GRPCAddress != nil {
		cfg.GRPCAddress = *fileCfg.GRPCAddress
	}
	if fileCfg.ReportInterval != nil {
		cfg.ReportInterval, err = time.ParseDuration(*fileCfg.ReportInterval)
		if err != nil {
			return fmt.Errorf("invalid report_interval in agent config file %q: %w", path, err)
		}
	}
	if fileCfg.PollInterval != nil {
		cfg.PollInterval, err = time.ParseDuration(*fileCfg.PollInterval)
		if err != nil {
			return fmt.Errorf("invalid poll_interval in agent config file %q: %w", path, err)
		}
	}
	if fileCfg.Key != nil {
		cfg.Key = *fileCfg.Key
	}
	if fileCfg.CryptoKey != nil {
		cfg.CryptoKey = *fileCfg.CryptoKey
	}
	if fileCfg.RateLimit != nil {
		cfg.RateLimit = *fileCfg.RateLimit
	}

	return nil
}

func lookupEnvInt(key string) (int, bool, error) {
	value, ok := os.LookupEnv(key)
	if !ok {
		return 0, false, nil
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, true, fmt.Errorf("%q is not a valid integer", value)
	}

	return parsed, true, nil
}

func lookupEnvDurationSeconds(key string) (time.Duration, bool, error) {
	value, ok, err := lookupEnvInt(key)
	if err != nil || !ok {
		return 0, ok, err
	}

	return time.Duration(value) * time.Second, true, nil
}
